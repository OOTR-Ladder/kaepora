package web

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"kaepora/internal/back"
	"kaepora/internal/util"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

const (
	authCookieName          = "auth"
	authCookieLifetime      = 7 * 24 * time.Hour
	authStateCookieName     = "auth_state"
	authStateCookieLifetime = 0 // session cookie
)

func (s *Server) authenticator(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		player, err := s.playerFromCookie(r)
		if err != nil {
			log.Printf("error: cookie auth: %s", err)
			player = nil
		}

		h.ServeHTTP(w, r.WithContext(withPlayer(r.Context(), player)))
	})
}

func (s *Server) playerFromCookie(r *http.Request) (*back.Player, error) {
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		// no cookie, ignore successfully
		return nil, nil
	}

	var playerIDStr string
	if err := s.sc.Decode(authCookieName, cookie.Value, &playerIDStr); err != nil {
		return nil, fmt.Errorf("error: unable to decode auth cookie: %s", err)
	}

	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		return nil, err
	}

	player, err := s.back.GetPlayerByID(util.UUIDAsBlob(playerID))
	if err != nil {
		return nil, fmt.Errorf("error: unable to fetch player from auth cookie: %s", err)
	}

	return &player, nil
}

func playerFromContext(ctx context.Context) *back.Player {
	return ctx.Value(ctxKeyAuthPlayer).(*back.Player)
}

func playerFromRequest(r *http.Request) *back.Player {
	return playerFromContext(r.Context())
}

func withPlayer(ctx context.Context, player *back.Player) context.Context {
	return context.WithValue(ctx, ctxKeyAuthPlayer, player)
}

func (s *Server) isAuthenticatedUserAdmin(r *http.Request) bool {
	return s.isPlayerAdmin(playerFromRequest(r))
}

func (s *Server) isPlayerAdmin(p *back.Player) bool {
	if p == nil {
		return false
	}

	// Don't allow access to spoiler logs and stuff if the user is in a race.
	if !s.config.DevMode && s.back.PlayerIsInSession(p.ID) {
		return false
	}

	return s.config.IsDiscordIDAdmin(p.DiscordID.String)
}

func (s *Server) authDiscord(w http.ResponseWriter, r *http.Request) {
	conf := s.config.Discord.OAuth2(s.config)

	// Step 1, no code = redirect to OAuth2 provider.
	code := r.URL.Query().Get("code")
	if code == "" {
		s.authDiscordRedirect(w, r, conf)
		return
	}

	// Step 2, redirected from OAuth2 provider, obtain token and user.
	user, err := s.getDiscordUserFromOAuth2Code(r, conf, code)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	// Clear state cookie, no longer needed.
	if err := s.deleteCookie(w, authStateCookieName); err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	// We have an user, log as the user and create it if needed
	player, err := s.back.GetPlayerByDiscordID(user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("info: registering new Player for Discord OAuth2 user %s#%s", user.ID, user.Discriminator)
		player, err = s.back.RegisterDiscordPlayer(user.ID, user.Name+"#"+user.Discriminator)
	}
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	if err := s.setAuthCookie(w, player.ID); err != nil {
		log.Printf("error: unable to write auth cookie: %s", err)
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

type discordUserPayload struct {
	ID            string `json:"id"`
	Name          string `json:"username"`
	Discriminator string `json:"discriminator"`
}

func (s *Server) getDiscordUserFromOAuth2Code(
	r *http.Request,
	conf *oauth2.Config,
	code string,
) (discordUserPayload, error) {
	if err := s.checkOAuthState(r); err != nil {
		return discordUserPayload{}, err
	}
	token, err := conf.Exchange(
		r.Context(),
		code,
	)
	if err != nil {
		return discordUserPayload{}, err
	}
	client := conf.Client(r.Context(), token)

	req, err := http.NewRequestWithContext(r.Context(), "GET", "https://discord.com/api/users/@me", nil)
	if err != nil {
		return discordUserPayload{}, err
	}
	res, err := client.Do(req)
	if err != nil {
		return discordUserPayload{}, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return discordUserPayload{}, err
	}

	var payload discordUserPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return discordUserPayload{}, err
	}

	return payload, nil
}

func (s *Server) checkOAuthState(r *http.Request) error {
	cookie, err := r.Cookie(authStateCookieName)
	if err != nil {
		return errors.New("no state cookie")
	}

	var localState string
	if err := s.sc.Decode(authStateCookieName, cookie.Value, &localState); err != nil {
		return fmt.Errorf("error: unable to decode auth cookie: %s", err)
	}

	remoteState := r.URL.Query().Get("state")
	if remoteState == "" {
		return errors.New("empty remote state")
	}

	if subtle.ConstantTimeCompare([]byte(localState), []byte(remoteState)) != 1 {
		return errors.New("local and remote state do not match")
	}

	return nil
}

func (s *Server) authDiscordRedirect(w http.ResponseWriter, r *http.Request, conf *oauth2.Config) {
	state, err := randomState()
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	url := conf.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("response_type", "code"),
	)

	if err := s.setEncodedCookie(
		w, authStateCookieName, state,
		authStateCookieLifetime,
	); err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func randomState() (string, error) {
	size := 32
	buf := make([]byte, size)
	c, err := rand.Read(buf)
	if err != nil {
		return "", err
	}

	if c != size {
		return "", fmt.Errorf("read %d bytes, expected %d", c, size)
	}

	return hex.EncodeToString(buf), nil
}

func (s *Server) deleteCookie(w http.ResponseWriter, name string) error {
	return s.setEncodedCookie(w, name, "", -1)
}

func (s *Server) setAuthCookie(
	w http.ResponseWriter,
	playerID util.UUIDAsBlob,
) error {
	return s.setEncodedCookie(
		w, authCookieName, playerID.String(),
		authCookieLifetime,
	)
}

func (s *Server) setEncodedCookie(
	w http.ResponseWriter,
	name, value string,
	lifetime time.Duration, // < 0 delete, 0 session, > 0 cookie
) error {
	encoded, err := s.sc.Encode(name, value)
	if err != nil {
		return fmt.Errorf("unable to encode auth state cookie: %s", err)
	}

	var domain string
	if !s.config.DevMode { // host:port is problematic
		domain = s.config.Domain
	}

	var maxAge int
	var expires time.Time
	switch {
	case lifetime > 0:
		expires = time.Now().Add(authCookieLifetime)
	case lifetime < 0:
		maxAge = -1
	case lifetime == 0:
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    encoded,
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
		Domain:   domain,
		HttpOnly: true,
		Secure:   !s.config.DevMode,
	})

	return nil
}
