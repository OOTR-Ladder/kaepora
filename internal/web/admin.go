package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"kaepora/internal/back"
	"kaepora/internal/back/schedule"
	"kaepora/internal/generator/factory"
	"kaepora/internal/util"
	"net/http"
)

func (s *Server) adminAllLeagues(w http.ResponseWriter, r *http.Request) {
	leagues, err := s.back.GetLeagues()
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.response(w, r, http.StatusOK, "admin/all_leagues.html", struct {
		Leagues []back.League
	}{
		leagues,
	})
}

func (s *Server) adminOneLeague(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r, "id")
	if err != nil {
		s.error(w, r, err, http.StatusBadRequest)
		return
	}

	league, err := s.back.GetLeague(id)
	if err != nil {
		s.notFound(w, r)
		return
	}

	var (
		saved  bool
		errStr string
	)

	if r.Method == "POST" {
		switch {
		case r.PostFormValue("action-save") != "":
			var err error
			league, err = s.adminSaveOneLeague(r, league)
			if err != nil {
				errStr = err.Error()
			} else {
				saved = true
			}
		case r.PostFormValue("action-delete") != "":
			if err := s.back.DeleteLeague(id); err != nil {
				s.error(w, r, err, http.StatusInternalServerError)
				return
			}

			locale := r.Context().Value(ctxKeyLocale).(string)
			http.Redirect(w, r, "/"+locale+"/admin/leagues", http.StatusFound)
			return
		}
	}

	s.response(w, r, http.StatusOK, "admin/one_league.html", struct {
		League back.League
		Saved  bool
		Error  string
	}{
		league,
		saved,
		errStr,
	})
}

func (s *Server) adminSaveOneLeague(r *http.Request, l back.League) (back.League, error) {
	var e []error
	l.Name = r.PostFormValue("Name")
	if l.Name == "" {
		e = append(e, errors.New("field Name must not be empty"))
	}

	l.ShortCode = r.PostFormValue("ShortCode")
	if l.ShortCode == "" {
		e = append(e, errors.New("field ShortCode must not be empty"))
	}

	l.Settings = r.PostFormValue("Settings")
	if l.Settings == "" {
		e = append(e, errors.New("field Settings must not be empty"))
	}

	l.Generator = r.PostFormValue("Generator")
	_, err := factory.New(nil).NewGenerator(l.Generator)
	if err != nil {
		e = append(e, fmt.Errorf("unable to parse Generator: %s", err))
	}

	var conf schedule.Config
	if err := json.Unmarshal([]byte(r.PostFormValue("Schedule")), &conf); err != nil {
		e = append(e, fmt.Errorf("invalid Schedule JSON: %s", err))
	} else {
		l.Schedule = conf // Keep the bad value so the user can edit it.

		if _, err := schedule.New(l.Schedule); err != nil {
			e = append(e, fmt.Errorf("invalid Schedule configuration: %s", err))
		}
	}

	if err := util.ConcatErrors(e); err != nil {
		return l, err
	}

	return l, s.back.UpdateLeague(l)
}
