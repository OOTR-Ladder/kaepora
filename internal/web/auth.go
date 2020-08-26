package web

import (
	"context"
	"fmt"
	"kaepora/internal/back"
	"kaepora/internal/util"
	"log"
	"net/http"
	"time"
)

const (
	authCookieName     = "auth"
	authCookieLifetime = 7 * 24 * time.Hour
)

func (s *Server) tokenAuthenticator(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		player, err := s.getSignedPlayer(r)
		if err != nil {
			log.Printf("error: token auth: %s", err)
		}
		// immediate auth for dev and player that don't want to  OAuth
		if player != nil {
			if err := s.setCookiePlayer(w, player.ID); err != nil {
				log.Printf("error: unable to write auth cookie: %s", err)
			}

			u := r.URL
			q := u.Query()
			q.Del("t")
			u.RawQuery = q.Encode()

			http.Redirect(w, r, u.String(), http.StatusFound)
			return
		}

		player, err = s.playerFromCookie(r)
		if err != nil {
			log.Printf("error: cookie auth: %s", err)
			player = nil
		}

		h.ServeHTTP(w, r.WithContext(withPlayer(r.Context(), player)))
	})
}

func (s *Server) setCookiePlayer(w http.ResponseWriter, playerID util.UUIDAsBlob) error {
	encoded, err := s.sc.Encode(authCookieName, playerID)
	if err != nil {
		return fmt.Errorf("unable to encode auth cookie: %s", err)
	}

	var domain string
	if !s.config.DevMode { // host:port is problematic
		domain = s.config.Domain
	}

	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    encoded,
		Path:     "/",
		Expires:  time.Now().Add(authCookieLifetime),
		Domain:   domain,
		HttpOnly: true,
		Secure:   !s.config.DevMode,
	})

	return nil
}

func (s *Server) playerFromCookie(r *http.Request) (*back.Player, error) {
	var playerID util.UUIDAsBlob
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		// no cookie, ignore successfully
		return nil, nil
	}

	if err := s.sc.Decode(authCookieName, cookie.Value, &playerID); err != nil {
		return nil, fmt.Errorf("error: unable to decode auth cookie: %s", err)
	}

	player, err := s.back.GetPlayerByID(playerID)
	if err != nil {
		return nil, fmt.Errorf("error: unable to fetch player from auth cookie: %s", err)
	}

	return &player, nil
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

	// Don't allow access to spoiler logs and stuff if the user is in a race.
	if s.back.PlayerIsInSession(player.ID) {
		return false
	}

	return s.config.IsDiscordIDAdmin(player.DiscordID.String)
}
