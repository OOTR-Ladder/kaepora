package web

import (
	"kaepora/internal/back"
	"kaepora/internal/util"
	"net/http"
	"time"

	"github.com/go-chi/chi"
)

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	sessions, leagues, err := s.back.GetMatchSessions(
		time.Now().Add(-30*24*time.Hour),
		time.Now(),
		[]back.MatchSessionStatus{
			back.MatchSessionStatusClosed,
		},
		`StartDate DESC`,
	)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 1*time.Hour)
	s.response(w, r, http.StatusOK, "history.html", struct {
		MatchSessions []back.MatchSession
		Leagues       map[util.UUIDAsBlob]back.League
	}{
		sessions,
		leagues,
	})
}

func (s *Server) leaderboard(w http.ResponseWriter, r *http.Request) {
	shortcode := chi.URLParam(r, "shortcode")
	league, err := s.back.GetLeagueByShortcode(shortcode)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	leaderboard, err := s.back.GetLeaderboardForShortcode(shortcode, back.DeviationThreshold)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 5*time.Minute)
	s.response(w, r, http.StatusOK, "leaderboard.html", struct {
		League      back.League
		Leaderboard []back.LeaderboardEntry
	}{league, leaderboard})
}
