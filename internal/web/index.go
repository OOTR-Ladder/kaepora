package web

import (
	"context"
	"database/sql"
	"errors"
	"kaepora/internal/back"
	"kaepora/internal/util"
	"net/http"
	"time"
)

// index serves the homepage with a quick recap of the std league.
func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	// HACK: hack-ish way to handle bad first path element.
	// The index acts as a catch all.
	if r.URL.Path != "/"+r.Context().Value(ctxKeyLocale).(string) {
		s.error(w, r, nil, http.StatusNotFound)
		return
	}

	data, err := s.getIndexTemplateData(r.Context())
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.response(w, r, http.StatusOK, "index.html", data)
}

type nextRacesTemplateData struct {
	Top3          []back.LeaderboardEntry
	MatchSessions []back.MatchSession
	Leagues       map[util.UUIDAsBlob]back.League

	// When we have an AuthenticatedPlayer
	JoinedSession *back.MatchSession
}

func (s *Server) getIndexTemplateData(ctx context.Context) (nextRacesTemplateData, error) {
	sessions, leagues, err := s.back.GetMatchSessions(
		time.Now().Add(-12*time.Hour),
		time.Now().Add(48*time.Hour),
		[]back.MatchSessionStatus{
			back.MatchSessionStatusWaiting,
			back.MatchSessionStatusJoinable,
			back.MatchSessionStatusPreparing,
			back.MatchSessionStatusInProgress,
		},
		`StartDate ASC`,
	)
	if err != nil {
		return nextRacesTemplateData{}, err
	}

	shortcode := "std"
	if len(sessions) > 0 {
		shortcode = leagues[sessions[0].LeagueID].ShortCode
	}

	top3, err := s.getStdTop3(shortcode)
	if err != nil {
		return nextRacesTemplateData{}, err
	}

	var joinedSession *back.MatchSession
	if player := playerFromContext(ctx); player != nil {
		session, err := s.back.GetPlayerActiveSession(player.ID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nextRacesTemplateData{}, err
			}
		} else {
			joinedSession = &session
		}
	}

	return nextRacesTemplateData{
		top3,
		sessions,
		leagues,
		joinedSession,
	}, nil
}

// getStdTop3 returns the Top 3 leaderboard.
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
