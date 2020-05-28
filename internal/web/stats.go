package web

import (
	"kaepora/internal/back"
	"net/http"
	"time"
)

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	misc, err := s.back.GetMiscStats()
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 1*time.Hour)
	s.response(w, r, http.StatusOK, "seed_stats.html", struct {
		Misc back.StatsMisc
	}{misc})
}
