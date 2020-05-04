package web

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"kaepora/internal/back"
	"kaepora/internal/util"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/leonelquinteros/gotext"
	"github.com/russross/blackfriday/v2"
	"golang.org/x/text/language"
)

func (s *Server) setupRouter(baseDir string) *chi.Mux {
	middleware.DefaultLogger = middleware.RequestLogger(&middleware.DefaultLogFormatter{
		Logger: log.New(os.Stdout, "web: ", 0),
	})

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(langDetect)

	fs := http.StripPrefix("/_/", http.FileServer(http.Dir(
		filepath.Join(baseDir, "static"),
	)))
	r.HandleFunc("/_/*", func(w http.ResponseWriter, r *http.Request) {
		s.cache(w, "public", 1*time.Hour)
		fs.ServeHTTP(w, r)
	})

	r.Get("/rules", s.markdownContent(baseDir, "rules.md"))
	r.Get("/documentation", s.markdownContent(baseDir, "documentation.md"))
	r.Get("/", s.index)

	return r
}

type ctxKey int

const ctxKeyLocale ctxKey = iota

func langDetect(next http.Handler) http.Handler {
	matcher := language.NewMatcher([]language.Tag{
		language.English,
		language.French,
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale, _ := language.MatchStrings(
			matcher,
			r.URL.Query().Get("lang"),
			r.Header.Get("Accept-Language"),
		)
		base, _ := locale.Base()
		key := base.String()

		ctx := context.WithValue(r.Context(), ctxKeyLocale, key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type Server struct {
	http    *http.Server
	back    *back.Back
	tpl     map[string]*template.Template
	locales map[string]*gotext.Locale

	tokenKey string
}

func NewServer(back *back.Back, tokenKey string) (*Server, error) {
	baseDir, err := getResourcesDir()
	if err != nil {
		return nil, err
	}

	s := &Server{
		tokenKey: tokenKey,
		back:     back,
		locales:  map[string]*gotext.Locale{},
		http: &http.Server{
			Addr:         "127.0.0.1:3001",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			IdleTimeout:  10 * time.Second,
		},
	}

	for _, k := range []string{"en", "fr"} {
		s.locales[k] = gotext.NewLocale(filepath.Join(baseDir, "locales"), k)
		s.locales[k].AddDomain("default")
	}

	s.http.Handler = s.setupRouter(baseDir)
	s.tpl, err = s.loadTemplates(baseDir)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func getResourcesDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Join(wd, "resources/web"), nil
}

func (s *Server) Serve(wg *sync.WaitGroup, done <-chan struct{}) {
	log.Println("info: starting HTTP server")
	wg.Add(1)
	defer wg.Done()

	go func() {
		err := s.http.ListenAndServe()
		if err == http.ErrServerClosed {
			log.Println("info: HTTP server closed")
			return
		}

		log.Fatalf("webserver crashed: %s", err)
	}()

	<-done
	if err := s.http.Close(); err != nil {
		log.Printf("warning: unable to close webserver: %s", err)
	}
}

func (s *Server) response(w http.ResponseWriter, r *http.Request, code int, template string, payload interface{}) {
	tpl, ok := s.tpl[template]
	if !ok {
		s.error(w, fmt.Errorf("template not found: %s", template), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(code)

	wrapped := struct {
		Locale  string
		Payload interface{}
	}{
		r.Context().Value(ctxKeyLocale).(string),
		payload,
	}

	if err := tpl.ExecuteTemplate(w, "base", wrapped); err != nil {
		log.Printf("error: unable to render template: %s", err)
	}
}

func (s *Server) error(w http.ResponseWriter, err error, code int) {
	log.Printf("error: %s", err)
	w.WriteHeader(code)
}

func (s *Server) cache(w http.ResponseWriter, scope string, d time.Duration) {
	w.Header().Set("Cache-Control", fmt.Sprintf("%s,max-age=%d", scope, d/time.Second))
}

func (s *Server) markdownContent(baseDir, name string) http.HandlerFunc {
	pathFmt := filepath.Join(baseDir, "content", "%s", name)

	return func(w http.ResponseWriter, r *http.Request) {
		locale := r.Context().Value(ctxKeyLocale).(string)
		if len(locale) != 2 || (locale[0] == '.' || locale[0] == '/') {
			s.error(w, fmt.Errorf("got a dangerous locale: %s", locale), http.StatusBadRequest)
			return
		}

		md, err := ioutil.ReadFile(fmt.Sprintf(pathFmt, locale))
		if err != nil {
			s.error(w, err, http.StatusInternalServerError)
			return
		}

		parsed := template.HTML(blackfriday.Run(md)) // nolint:gosec

		s.cache(w, "public", 1*time.Hour)
		s.response(w, r, http.StatusOK, "markdown.html", parsed)
	}
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	top20, err := s.getStdTop20()
	if err != nil {
		s.error(w, err, http.StatusInternalServerError)
		return
	}

	sessions, leagues, err := s.back.GetMatchSessionsAroundNow()
	if err != nil {
		s.error(w, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 1*time.Minute)
	s.response(w, r, http.StatusOK, "index.html", struct {
		Top20         []back.LeaderboardEntry
		MatchSessions []back.MatchSession
		Leagues       map[util.UUIDAsBlob]back.League
	}{
		top20,
		sessions,
		leagues,
	})
}

func (s *Server) getStdTop20() ([]back.LeaderboardEntry, error) {
	leaderboard, err := s.back.GetLeaderboardForShortcode(
		"std",
		back.DeviationThreshold,
	)
	if err != nil {
		return nil, err
	}

	if len(leaderboard) > 20 {
		leaderboard = leaderboard[:20]
	}

	return leaderboard, nil
}
