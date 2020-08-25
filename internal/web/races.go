package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"kaepora/internal/back"
	"kaepora/internal/generator/oot"
	"kaepora/internal/util"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi"
)

// getAllMatchSession shows a shortened recap of previous races.
func (s *Server) getAllMatchSession(w http.ResponseWriter, r *http.Request) {
	statuses := []back.MatchSessionStatus{
		back.MatchSessionStatusClosed,
	}

	if s.isUserAdmin(r) {
		statuses = append(
			statuses,
			back.MatchSessionStatusPreparing,
			back.MatchSessionStatusInProgress,
		)
	}

	sessions, leagues, err := s.back.GetMatchSessions(
		time.Now().Add(-30*24*time.Hour),
		time.Now(),
		statuses,
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

	if !s.isUserAdmin(r) {
		for k := range matches {
			if !matches[k].HasEnded() {
				s.error(w, r, errors.New("this session is still in progress"), http.StatusForbidden)
				return
			}
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

	league, err := s.back.GetLeague(match.LeagueID)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	if !s.isUserAdmin(r) && !match.HasEnded() {
		if err := s.canSignedPlayerSeeSpoilerLog(r, match); err != nil {
			s.error(w, r, err, http.StatusInternalServerError)
			return
		}
	}

	raw, err := ioutil.ReadAll(match.SpoilerLog.Uncompressed())
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("raw") == "1" {
		s.sendRawSpoilerLog(w, league, match, raw)
		return
	}

	var parsed oot.SpoilerLog
	if err := json.Unmarshal(raw, &parsed); err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	settings, err := s.getSettingsDiff(match.GeneratorState, r.Context().Value(ctxKeyLocale).(string))
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 1*time.Hour)
	s.response(w, r, http.StatusOK, "spoilers.html", struct {
		Match    back.Match
		Settings map[string]back.SettingsDocumentationValueEntry
		JSON     string
		Log      oot.SpoilerLog
	}{match, settings, string(raw), parsed})
}

func (s *Server) canSignedPlayerSeeSpoilerLog(r *http.Request, match back.Match) error {
	if match.HasEnded() {
		return nil
	}

	player, err := s.getSignedPlayer(r)
	if err != nil {
		return err
	}
	if player == nil {
		return util.ErrPublic("match has not ended")
	}

	entry, _, err := match.GetPlayerAndOpponentEntries(player.ID)
	if err != nil {
		return err
	}
	if !entry.HasEnded() {
		return util.ErrPublic("match has not ended")
	}

	return nil
}

func (s *Server) sendRawSpoilerLog(w http.ResponseWriter, league back.League, match back.Match, raw []byte) {
	s.cache(w, "public", 1*time.Hour)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf(
			`attachment; filename="%s_%s_%s.spoilers.json"`,
			league.ShortCode,
			match.StartedAt.Time.Time().Format("2006-01-02_15h04"),
			match.Seed,
		),
	)

	if _, err := w.Write(raw); err != nil {
		log.Printf("warning: %s", err)
	}
}

func (s *Server) getSettingsDiff(
	stateJSON []byte,
	locale string,
) (map[string]back.SettingsDocumentationValueEntry, error) {
	var state oot.State
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return nil, err
	}
	if len(state.SettingsPatch) == 0 {
		return nil, nil
	}

	doc, err := back.LoadSettingsDocumentation(locale)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]back.SettingsDocumentationValueEntry, len(state.SettingsPatch))
	for k, v := range state.SettingsPatch {
		setting := doc[k]
		value := setting.GetValueEntry(v)

		if setting.Title == "" {
			continue
		}
		if value.Title == "" {
			continue
		}

		ret[setting.Title] = value
	}

	return ret, nil
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

	var ret []scheduleEntry

	for k := range leagues {
		lastFoundStart := start
		for next := start; !next.IsZero() && next.Before(end); next = leagues[k].Schedule.NextBetween(next, end) {
			if next == lastFoundStart {
				continue
			}

			lastFoundStart = next
			ret = append(ret, scheduleEntry{
				LeagueName: leagues[k].Name,
				StartDate:  next,
			})
		}
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].StartDate.Before(ret[j].StartDate)
	})

	return ret, nil
}

type scheduleEntry struct {
	LeagueName string
	StartDate  time.Time
}
