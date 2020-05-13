package web

import (
	"kaepora/internal/back"
	"kaepora/internal/util"
	"net/http"
	"time"
)

// index serves the homepage with a quick recap of the std league.
func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	top3, err := s.getStdTop3("std")
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	sessions, leagues, err := s.back.GetMatchSessions(
		time.Now().Add(-24*time.Hour),
		time.Now().Add(2*24*time.Hour),
		[]back.MatchSessionStatus{
			back.MatchSessionStatusWaiting,
			back.MatchSessionStatusJoinable,
			back.MatchSessionStatusPreparing,
			back.MatchSessionStatusInProgress,
		},
		`StartDate ASC`,
	)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 1*time.Minute)
	s.response(w, r, http.StatusOK, "index.html", struct {
		Top3          []back.LeaderboardEntry
		MatchSessions []back.MatchSession
		Leagues       map[util.UUIDAsBlob]back.League
	}{
		top3,
		sessions,
		leagues,
	})
}

// getStdTop3 returns the Top 3 leaderboard
func (s *Server) getStdTop3(shortcode string) ([]back.LeaderboardEntry, error) {
	leaderboard, err := s.back.GetLeaderboardForShortcode(
		shortcode,
		back.DeviationThreshold,
	)
	if err != nil {
		return nil, err
	}

	if len(leaderboard) > 3 {
		leaderboard = leaderboard[:3]
	}

	return leaderboard, nil
}
