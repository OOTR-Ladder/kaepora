package web

import (
	"context"
	"kaepora/internal/back"
	"log"
	"net/http"
)

func (s *Server) tokenAuthenticator(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		player, err := s.getSignedPlayer(r)
		if err != nil || player == nil {
			player = nil
			if err != nil {
				log.Printf("error: invalid auth token: %s", err)
			}
		}

		h.ServeHTTP(w, r.WithContext(withPlayer(r.Context(), player)))
	})
}

func playerFromContext(r *http.Request) *back.Player {
	return r.Context().Value(ctxKeyAuthPlayer).(*back.Player)
}

func withPlayer(ctx context.Context, player *back.Player) context.Context {
	return context.WithValue(ctx, ctxKeyAuthPlayer, player)
}

func (s *Server) getSignedPlayer(r *http.Request) (*back.Player, error) {
	tokenID, err := queryID(r, "t")
	if err != nil {
		// Empty or invalid token, just ignore it.
		return nil, nil
	}

	player, err := s.back.GetPlayerFromTokenID(tokenID)
	if err != nil {
		return nil, err
	}

	return player, nil
}

func (s *Server) isAuthenticatedUserAdmin(r *http.Request) bool {
	player := playerFromContext(r)
	if !s.config.IsDiscordIDAdmin(player.DiscordID.String) {
		return false
	}

	// Don't allow access to spoiler logs and stuff if the user is in a race.
	return !s.back.PlayerIsInSession(player.ID)
}
