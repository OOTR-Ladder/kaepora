package web

import (
	"kaepora/internal/back"
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

	s.response(w, r, http.StatusOK, "admin/one_league.html", struct {
		League back.League
	}{
		league,
	})
}
