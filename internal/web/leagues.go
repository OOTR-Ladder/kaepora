package web

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi"
)

func (s *Server) getLeagues(w http.ResponseWriter, _ *http.Request) {
	games, leagues, times, err := s.back.GetGamesLeaguesAndTheirNextSessionStartDate()
	if err != nil {
		s.error(w, err, http.StatusInternalServerError)
		return
	}

	nextLeagueSessions := make(map[string]time.Time, len(times))
	for k, v := range times {
		nextLeagueSessions[k.UUID().String()] = v
	}

	s.cache(w, "public", 5*time.Minute)
	s.response(w, http.StatusOK, map[string]interface{}{
		"games":                games,
		"leagues":              leagues,
		"next_league_sessions": nextLeagueSessions,
	})
}

func (s *Server) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	leaderboard, err := s.back.GetLeaderboardForShortcode(
		chi.URLParam(r, "shortcode"),
		250, // seems like an OK cutoff right now, but will need to be change later TODO
	)
	if errors.Is(err, sql.ErrNoRows) {
		s.error(w, err, http.StatusNotFound)
		return
	}

	if err != nil {
		s.error(w, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 1*time.Hour)
	s.response(w, http.StatusOK, leaderboard)
}
