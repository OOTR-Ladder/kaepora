package web

import (
	"kaepora/internal/back"
	"kaepora/internal/util"
	"net/http"
	"sort"
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

func (s *Server) schedule(w http.ResponseWriter, r *http.Request) {
	data, err := s.getIndexTemplateData()
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	lastStartDate := time.Now()
	if len(data.MatchSessions) > 0 {
		lastStartDate = data.MatchSessions[len(data.MatchSessions)-1].StartDate.Time()
	}
	if len(data.MatchSessions) > 1 {
		data.MatchSessions = data.MatchSessions[:1]
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

	return ret, nil
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
