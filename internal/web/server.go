package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"kaepora/internal/back"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	fs := http.StripPrefix("/_/", http.FileServer(http.Dir(
		filepath.Join(baseDir, "static"),
	)))
	r.HandleFunc("/_/*", func(w http.ResponseWriter, r *http.Request) {
		s.cache(w, "public", 1*time.Hour)
		fs.ServeHTTP(w, r)
	})

	r.Get("/favicon.ico", s.favicon(fs))

	r.With(s.langDetect).Route("/{locale}", func(r chi.Router) {
		r.Get("/rules", s.markdownContent(baseDir, "rules.md"))
		r.Get("/documentation", s.markdownContent(baseDir, "documentation.md"))

		r.Get("/leaderboard/{shortcode}", s.leaderboard)
		r.Get("/history", s.history)
		r.Get("/schedule", s.schedule)
		r.Get("/stats/ratings.svg", s.statsRatings)
		r.Get("/stats", s.stats)
		r.Get("/", s.index)

		r.NotFound(s.notFound)
	})

	r.Get("/", s.redirectToLocale)

	return r
}

type ctxKey int

const ctxKeyLocale ctxKey = iota

func chooseLocale(candidates ...string) string {
	matcher := language.NewMatcher([]language.Tag{
		language.English,
		language.French,
	})

	locale, _ := language.MatchStrings(
		matcher,
		candidates...,
	)
	base, _ := locale.Base()
	return base.String()
}

func (s *Server) langDetect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		param := chi.URLParam(r, "locale")

		// Invalid locale, force 404 in accept-language lang.
		// This is necessary because otherwise users could bypass the
		// aggressive caching by specifying a bogus locale and get back a fresh
		// English page.
		if _, ok := s.locales[param]; !ok {
			log.Printf("warning: user requested invalid locale: %s", param)
			ctx := context.WithValue(r.Context(), ctxKeyLocale, "en")
			s.notFound(w, r.WithContext(ctx))
			return
		}

		locale := chooseLocale(param, r.Header.Get("Accept-Language"))
		ctx := context.WithValue(r.Context(), ctxKeyLocale, locale)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	s.error(w, r, errors.New(http.StatusText(http.StatusNotFound)), http.StatusNotFound)
}

// Server contains the state required to serve the OOTRLadder website over HTTP.
type Server struct {
	http    *http.Server
	back    *back.Back
	tpl     map[string]*template.Template // Indexed by file name (eg. "index.html")
	locales map[string]*gotext.Locale     // Indexed by lowercase ISO 639-2 (eg. "fr")

	// Secret key for HMAC token verification
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

// getResourcesDir returns the absolute path to the web server static resources.
func getResourcesDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Join(wd, "resources/web"), nil
}

// Serve starts the HTTP and blocks until the done channel is closed.
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

// response renders the given template with the given payload available in ".Payload".
func (s *Server) response(
	w http.ResponseWriter,
	r *http.Request,
	code int, // HTTP return code to write
	template string,
	payload interface{},
) {
	tpl, ok := s.tpl[template]
	if !ok {
		log.Printf("template not found: %s", template)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(code)

	leagues, err := s.back.GetLeagues()
	if err != nil {
		log.Printf("error: %s", err)
		return
	}

	wrapped := struct {
		Locale  string
		Leagues []back.League
		Payload interface{}
	}{
		r.Context().Value(ctxKeyLocale).(string),
		leagues,
		payload,
	}

	if err := tpl.ExecuteTemplate(w, "base", wrapped); err != nil {
		log.Printf("error: unable to render template: %s", err)
	}
}

func (s *Server) error(w http.ResponseWriter, r *http.Request, err error, code int) {
	if errors.Is(err, sql.ErrNoRows) {
		code = http.StatusNotFound
	}

	log.Printf("error: HTTP %d: %v", code, err)
	s.response(w, r, code, "error.html", struct {
		HTTPCode int
	}{code})
	w.WriteHeader(code)
}

func (s *Server) cache(w http.ResponseWriter, scope string, d time.Duration) { // nolint:unparam
	w.Header().Set("Cache-Control", fmt.Sprintf("%s,max-age=%d", scope, d/time.Second))
}

func (s *Server) favicon(fs http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/_/favicon.ico"

		s.cache(w, "public", 365*24*time.Hour)
		fs.ServeHTTP(w, r)
	}
}

// markdownContent serves the given markdown file out of the
// "<baseDir>/content/<locale>" directory as HTML.
func (s *Server) markdownContent(baseDir, name string) http.HandlerFunc {
	pathFmt := filepath.Join(baseDir, "content", "%s", name)

	return func(w http.ResponseWriter, r *http.Request) {
		locale := r.Context().Value(ctxKeyLocale).(string)
		if len(locale) != 2 || (locale[0] == '.' || locale[0] == '/') {
			s.error(w, r, fmt.Errorf("got a dangerous locale: %s", locale), http.StatusBadRequest)
			return
		}

		path := fmt.Sprintf(pathFmt, locale)
		md, err := ioutil.ReadFile(path)
		if err != nil {
			s.error(w, r, err, http.StatusInternalServerError)
			return
		}

		// HACK, prefix every absolute link to local content with the locale.
		parsed := template.HTML(strings.ReplaceAll( // nolint:gosec
			string(blackfriday.Run(md)),
			`href="/`,
			`href="/`+locale+`/`,
		))
		title := getMarkdownTitle(path)

		s.cache(w, "public", 1*time.Hour)
		s.response(w, r, http.StatusOK, "markdown.html", struct {
			Title    string
			Markdown template.HTML
		}{
			Title:    title,
			Markdown: parsed,
		})
	}
}

// getMarkdownTitle fetches the sibling file of a ".md" with the ".title"
// extension and returns its contents.
// If we ever need more stuff, change this to a "getMarkdownMeta".
func getMarkdownTitle(mdPath string) string {
	titlePath := mdPath[:len(mdPath)-2] + "title"
	title, err := ioutil.ReadFile(titlePath)
	if err != nil {
		return ""
	}

	return string(title)
}

func (s *Server) redirectToLocale(w http.ResponseWriter, r *http.Request) {
	locale := chooseLocale(r.Header.Get("Accept-Language"))
	http.Redirect(w, r, "/"+locale, http.StatusTemporaryRedirect)
}
