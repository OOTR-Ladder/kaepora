package web

import (
	"kaepora/internal/back"
	"net/http"
	"time"

	"github.com/go-chi/chi"
)

func (s *Server) getOnePlayer(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	player, err := s.back.GetPlayerByName(name)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 1*time.Hour)
	s.response(w, r, http.StatusOK, "one_player.html", struct {
		Player back.Player
	}{
		Player: player,
	})
}
