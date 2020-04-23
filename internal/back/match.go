package back

import (
	"fmt"
	"kaepora/internal/util"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type Match struct {
	ID             util.UUIDAsBlob
	LeagueID       util.UUIDAsBlob
	MatchSessionID util.UUIDAsBlob

	CreatedAt util.TimeAsTimestamp
	StartedAt util.NullTimeAsTimestamp
	EndedAt   util.NullTimeAsTimestamp

	Generator  string
	Settings   string
	Seed       string
	SpoilerLog string

	Entries []MatchEntry `db:"-"`
}

func NewMatch(tx *sqlx.Tx, session MatchSession, seed string) (Match, error) {
	league, err := getLeagueByID(tx, session.LeagueID)
	if err != nil {
		return Match{}, err
	}
	game, err := getGameByID(tx, league.GameID)
	if err != nil {
		return Match{}, err
	}

	return Match{
		ID:             util.NewUUIDAsBlob(),
		CreatedAt:      util.TimeAsTimestamp(time.Now()),
		LeagueID:       session.LeagueID,
		MatchSessionID: session.ID,
		Generator:      game.Generator,
		Settings:       league.Settings,
		Seed:           seed,
	}, nil
}

func (m *Match) end() {
	m.EndedAt = util.NewNullTimeAsTimestamp(time.Now())
}

func (m *Match) hasEnded() bool {
	return m.EndedAt.Valid
}

func (m *Match) insert(tx *sqlx.Tx) error {
	query, args, err := squirrel.Insert("Match").SetMap(squirrel.Eq{
		"ID":             m.ID,
		"LeagueID":       m.LeagueID,
		"MatchSessionID": m.MatchSessionID,

		"CreatedAt":  m.CreatedAt,
		"StartedAt":  m.StartedAt,
		"EndedAt":    m.EndedAt,
		"Generator":  m.Generator,
		"Settings":   m.Settings,
		"Seed":       m.Seed,
		"SpoilerLog": m.SpoilerLog,
	}).ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func (m *Match) update(tx *sqlx.Tx) error {
	query, args, err := squirrel.Update("Match").SetMap(squirrel.Eq{
		"StartedAt":  m.StartedAt,
		"EndedAt":    m.EndedAt,
		"SpoilerLog": m.SpoilerLog,
	}).Where("Match.ID = ?", m.ID).ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func getMatchByPlayerAndSession(tx *sqlx.Tx, player Player, session MatchSession) (Match, error) {
	var match Match
	query := `
    SELECT Match.* FROM Match
    INNER JOIN MatchEntry ON (MatchEntry.MatchID = Match.ID)
    WHERE Match.MatchSessionID = ? AND MatchEntry.PlayerID = ?
    LIMIT 1
    `
	if err := tx.Get(&match, query, session.ID, player.ID); err != nil {
		return Match{}, fmt.Errorf("could not fetch match: %w", err)
	}

	if err := injectEntries(tx, &match); err != nil {
		return Match{}, err
	}

	return match, nil
}

func injectEntries(tx *sqlx.Tx, match *Match) error {
	query := `SELECT * FROM MatchEntry WHERE MatchEntry.MatchID = ?`
	if err := tx.Select(&match.Entries, query, match.ID); err != nil {
		return fmt.Errorf("could not fetch entries: %w", err)
	}

	return nil
}

func getMatchBySeed(tx *sqlx.Tx, seed string) (Match, error) {
	var match Match
	query := `SELECT Match.* FROM Match WHERE Match.Seed = ?  LIMIT 1`
	if err := tx.Get(&match, query, seed); err != nil {
		return Match{}, fmt.Errorf("could not fetch match: %w", err)
	}

	if err := injectEntries(tx, &match); err != nil {
		return Match{}, err
	}

	return match, nil
}

func getMatchesBySessionID(tx *sqlx.Tx, sessionID util.UUIDAsBlob) ([]Match, error) {
	var matches []Match
	query := `SELECT Match.* FROM Match WHERE Match.MatchSessionID = ?`
	if err := tx.Select(&matches, query, sessionID); err != nil {
		return nil, fmt.Errorf("could not fetch match: %w", err)
	}

	for k := range matches {
		if err := injectEntries(tx, &matches[k]); err != nil {
			return nil, err
		}
	}

	return matches, nil
}

func (m *Match) getPlayerAndOpponentEntries(playerID util.UUIDAsBlob) (MatchEntry, MatchEntry, error) {
	if len(m.Entries) != 2 {
		return MatchEntry{}, MatchEntry{}, fmt.Errorf("invalid Match %s: not exactly 2 MatchEntry", m.ID)
	}

	if m.Entries[0].PlayerID == playerID {
		return m.Entries[0], m.Entries[1], nil
	} else if m.Entries[1].PlayerID == playerID {
		return m.Entries[1], m.Entries[0], nil
	}

	return MatchEntry{}, MatchEntry{}, fmt.Errorf("could not find MatchEntry for player %s in Match %d", playerID, m.ID)
}
