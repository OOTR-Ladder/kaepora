package web

import (
	"encoding/json"
	"fmt"
	"kaepora/internal/back"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func (s *Server) setupRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	r.Get("/", noContent)

	// I indend the v1 to be a hacky, quick'n dirty implementation, with no
	// pagination nor any fancy stuff.
	r.Get("/v1/leagues", s.getLeagues)
	r.Get("/v1/league/{shortcode}/leaderboard", s.getLeaderboard)
	r.Get("/v1/players", noContent)
	r.Get("/v1/player/{id}", noContent)
	r.Get("/v1/session/{id}", noContent)

	return r
}

type Server struct {
	http     *http.Server
	back     *back.Back
	tokenKey string
}

func NewServer(back *back.Back, tokenKey string) *Server {
	s := &Server{
		tokenKey: tokenKey,
		back:     back,
	}

	s.http = &http.Server{
		Addr:         "127.0.0.1:3001",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  10 * time.Second,
		Handler:      s.setupRouter(),
	}

	return s
}

func noContent(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) Serve(wg *sync.WaitGroup, done <-chan struct{}) {
	log.Println("info: starting HTTP server")
	wg.Add(1)
	defer wg.Done()

	s.setupRouter()

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

func (s *Server) response(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	response, err := json.Marshal(data)
	if err != nil {
		log.Printf("error: unable to marshal response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(code)

	if _, err := w.Write(response); err != nil {
		log.Printf("error: unable to send response: %s", err)
	}
}

func (s *Server) error(w http.ResponseWriter, err error, code int) {
	log.Printf("error: %s", err)
	w.WriteHeader(code)
}

func (s *Server) cache(w http.ResponseWriter, scope string, d time.Duration) {
	w.Header().Set("Cache-Control", fmt.Sprintf("%s,max-age=%d", scope, d/time.Second))
}
