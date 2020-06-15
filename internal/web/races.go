package web

import (
	"errors"
	"io"
	"kaepora/internal/back"
	"kaepora/internal/util"
	"log"
	"net/http"
	"reflect"
	"sort"
	"time"

	"github.com/go-chi/chi"
)

// getAllMatchSession shows a shortened recap of previous races.
func (s *Server) getAllMatchSession(w http.ResponseWriter, r *http.Request) {
	sessions, leagues, err := s.back.GetMatchSessions(
		time.Now().Add(-30*24*time.Hour),
		time.Now(),
		[]back.MatchSessionStatus{
			back.MatchSessionStatusClosed,
		},
		`DATETIME(StartDate) DESC`,
	)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 1*time.Hour)
	s.response(w, r, http.StatusOK, "sessions.html", struct {
		MatchSessions []back.MatchSession
		Leagues       map[util.UUIDAsBlob]back.League
	}{
		sessions,
		leagues,
	})
}

// getMatchSession shows the details of one MatchSession.
func (s *Server) getOneMatchSession(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r, "id")
	if err != nil {
		s.error(w, r, err, http.StatusNotFound)
		return
	}

	session, matches, players, err := s.back.GetMatchSession(id)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	for k := range matches {
		if !matches[k].HasEnded() {
			s.error(w, r, errors.New("this session is still in progress"), http.StatusForbidden)
			return
		}
	}

	league, err := s.back.GetLeague(session.LeagueID)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 1*time.Hour)
	s.response(w, r, http.StatusOK, "one_session.html", struct {
		MatchSession back.MatchSession
		League       back.League
		Matches      []back.Match
		Players      map[util.UUIDAsBlob]back.Player
	}{
		MatchSession: session,
		League:       league,
		Matches:      matches,
		Players:      players,
	})
}

func (s *Server) getSpoilerLog(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r, "id")
	if err != nil {
		s.error(w, r, err, http.StatusNotFound)
		return
	}

	match, err := s.back.GetMatch(id)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	if !match.HasEnded() {
		s.error(w, r, errors.New("match has not ended"), http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	s.cache(w, "public", 1*time.Hour)
	if _, err := io.Copy(w, match.SpoilerLog.Uncompressed()); err != nil {
		log.Printf("warning: %s", err)
	}
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

func (s *Server) schedule(w http.ResponseWriter, r *http.Request) {
	data, err := s.getIndexTemplateData()
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	lastStartDate := time.Now()
	if len(data.MatchSessions) > 1 {
		data.MatchSessions = data.MatchSessions[:1]
		lastStartDate = data.MatchSessions[0].StartDate.Time()
	}

	schedules, err := s.getSchedulesBetween(
		lastStartDate,
		lastStartDate.Add(7*24*time.Hour),
	)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 5*time.Minute)
	s.response(w, r, http.StatusOK, "schedule.html", struct {
		nextRacesTemplateData
		Schedules []scheduleEntry
	}{
		nextRacesTemplateData: data,
		Schedules:             schedules,
	})
}

func (s *Server) getSchedulesBetween(start, end time.Time) ([]scheduleEntry, error) {
	leagues, err := s.back.GetLeagues()
	if err != nil {
		return nil, err
	}

	lastFoundStart := map[util.UUIDAsBlob]time.Time{}
	var ret []scheduleEntry

	for i := start; i.Before(end); {
		least := end

		for k := range leagues {
			next := leagues[k].Schedule.NextBetween(i, end)
			if next.IsZero() || next == lastFoundStart[leagues[k].ID] {
				continue
			}
			lastFoundStart[leagues[k].ID] = next

			if next.Before(least) {
				least = next
			}

			ret = append(ret, scheduleEntry{
				LeagueName: leagues[k].Name,
				StartDate:  next,
			})
		}

		i = least
	}

	sort.Sort(sortByDate(ret))

	// HACK: There's definitely something wrong in the schedule generation as
	// duplicates are not supposed to happen. TODO
	deduped := make([]scheduleEntry, 0, len(ret))
	for _, v := range ret {
		if len(deduped) > 0 {
			if reflect.DeepEqual(deduped[len(deduped)-1], v) {
				continue
			}
		}

		deduped = append(deduped, v)
	}

	return deduped, nil
}

type scheduleEntry struct {
	LeagueName string
	StartDate  time.Time
}

type sortByDate []scheduleEntry

func (a sortByDate) Len() int {
	return len([]scheduleEntry(a))
}

func (a sortByDate) Less(i, j int) bool {
	return a[i].StartDate.Before(a[j].StartDate)
}

func (a sortByDate) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
