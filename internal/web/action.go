package web

import (
	"fmt"
	"kaepora/internal/util"
	"net/http"

	"github.com/google/uuid"
)

func (s *Server) doAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.error(w, r, err, http.StatusBadRequest)
		return
	}

	player := playerFromRequest(r)
	if player == nil {
		s.error(w, r, fmt.Errorf("not authenticated"), http.StatusForbidden)
		return
	}

	var err error
	switch r.PostForm.Get("Action") {
	case "join":
		var sessionID util.UUIDAsBlob
		if sessionIDStr := r.PostForm.Get("MatchSessionID"); sessionIDStr != "" {
			id, err := uuid.Parse(sessionIDStr)
			if err != nil {
				s.error(w, r, err, http.StatusBadRequest)
				return
			}
			sessionID = util.UUIDAsBlob(id)
		}
		err = s.back.JoinMatchSessionByID(sessionID, player.ID)
	case "cancel":
		_, err = s.back.CancelActiveMatchSession(player.ID)
	}

	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	url := r.PostForm.Get("Redirect")
	if url == "" {
		url = "/"
	}
	http.Redirect(w, r, url, http.StatusFound)
}
