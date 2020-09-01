package web

import (
	"errors"
	"kaepora/internal/back"
	"kaepora/internal/util"
	"log"
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

	stats, err := s.back.GetPlayerStats(player.ID)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	leagues, err := s.back.GetLeaguesMap()
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	matches, players, err := s.back.GetPlayerMatches(player.ID)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 1*time.Hour)
	s.response(w, r, http.StatusOK, "one_player.html", struct {
		Player      back.Player
		PlayerStats back.PlayerStats
		Leagues     map[util.UUIDAsBlob]back.League
		Matches     []back.Match
		Players     map[util.UUIDAsBlob]back.Player
	}{
		Player:      player,
		PlayerStats: stats,
		Leagues:     leagues,
		Matches:     matches,
		Players:     players,
	})
}

func (s *Server) getOnePlayerGraph(w http.ResponseWriter, r *http.Request) {
	playerName := chi.URLParam(r, "playerName")
	shortcode := chi.URLParam(r, "shortcode")

	var (
		graph []byte
		err   error
	)
	switch chi.URLParam(r, "graphName") {
	case "rating":
		graph, err = s.back.GetPlayerRatingGraph(playerName, shortcode)
	case "seedtime":
		graph, err = s.back.GetPlayerSeedTimeGraph(playerName, shortcode)
	default:
		s.error(w, r, errors.New("invalid graph name"), http.StatusNotFound)
		return
	}
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 1*time.Hour)
	w.Header().Set("Content-Type", "image/svg+xml")
	if _, err := w.Write(graph); err != nil {
		log.Printf("warning: error when writing graph: %s", err)
	}
}
