package back

import (
	"fmt"
	"kaepora/internal/util"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

// A Match is a single 1v1 belonging to a MatchSession, it has its own unique
// seed so each 1v1 is unique.
type Match struct {
	ID             util.UUIDAsBlob
	LeagueID       util.UUIDAsBlob
	MatchSessionID util.UUIDAsBlob

	CreatedAt util.TimeAsTimestamp
	StartedAt util.NullTimeAsTimestamp
	EndedAt   util.NullTimeAsTimestamp // when the two players completed their side of the race.

	Generator string
	Settings  string
	// Seed is the actual random generator initial seed, not to be confused
	// with the misnomer "seed" that designates the generated ROM/patch
	Seed string

	SpoilerLog     util.ZLIBBlob // arbitrary JSON, compressed
	GeneratorState []byte        // arbitrary JSON, depends on Generator
	SeedPatch      []byte        // arbitrary binary, depends on Generator, hopefully already compressed

	// Two entries, one per Player.
	Entries []MatchEntry `db:"-"`
}

func NewMatch(tx *sqlx.Tx, session MatchSession, seed string) (Match, error) {
	league, err := getLeagueByID(tx, session.LeagueID)
	if err != nil {
		return Match{}, err
	}

	return Match{
		ID:             util.NewUUIDAsBlob(),
		CreatedAt:      util.TimeAsTimestamp(time.Now()),
		LeagueID:       session.LeagueID,
		MatchSessionID: session.ID,
		Generator:      league.Generator,
		Settings:       league.Settings,
		Seed:           seed,
	}, nil
}

func (m *Match) IsDoubleForfeit() bool {
	return m.Entries[0].Status == MatchEntryStatusForfeit &&
		m.Entries[1].Status == MatchEntryStatusForfeit
}

func (m *Match) WinningEntry() MatchEntry {
	if m.Entries[1].HasWon() {
		return m.Entries[1]
	}

	return m.Entries[0]
}

func (m *Match) LosingEntry() MatchEntry {
	if m.Entries[1].HasWon() {
		return m.Entries[0]
	}

	return m.Entries[1]
}

func (m *Match) end() {
	m.EndedAt = util.NewNullTimeAsTimestamp(time.Now())
}

func (m *Match) HasEnded() bool {
	return m.EndedAt.Valid
}

func (m *Match) insert(tx *sqlx.Tx) error {
	m.ensureNotNULL()

	query, args, err := squirrel.Insert("Match").SetMap(squirrel.Eq{
		"ID":             m.ID,
		"LeagueID":       m.LeagueID,
		"MatchSessionID": m.MatchSessionID,

		"CreatedAt":      m.CreatedAt,
		"StartedAt":      m.StartedAt,
		"EndedAt":        m.EndedAt,
		"Generator":      m.Generator,
		"Settings":       m.Settings,
		"Seed":           m.Seed,
		"SpoilerLog":     m.SpoilerLog,
		"GeneratorState": m.GeneratorState,
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
	m.ensureNotNULL()

	query, args, err := squirrel.Update("Match").SetMap(squirrel.Eq{
		"StartedAt": m.StartedAt,
		"EndedAt":   m.EndedAt,

		"SpoilerLog":     m.SpoilerLog,
		"GeneratorState": m.GeneratorState,
	}).Where("Match.ID = ?", m.ID).ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func (m *Match) ensureNotNULL() {
	if m.SeedPatch == nil {
		m.SeedPatch = []byte{}
	}
	if m.SpoilerLog == nil {
		m.SpoilerLog = []byte{}
	}
	if m.GeneratorState == nil {
		m.GeneratorState = []byte{}
	}
}

func getMatchByPlayerAndSession(tx *sqlx.Tx, playerID, sessionID util.UUIDAsBlob) (Match, error) {
	query := `
    SELECT Match.* FROM Match
    INNER JOIN MatchEntry ON (MatchEntry.MatchID = Match.ID)
    WHERE Match.MatchSessionID = ? AND MatchEntry.PlayerID = ?
    LIMIT 1
    `

	var match Match
	if err := tx.Get(&match, query, sessionID, playerID); err != nil {
		return Match{}, fmt.Errorf("could not fetch match: %w", err)
	}

	if err := injectEntries(tx, &match); err != nil {
		return Match{}, err
	}

	return match, nil
}

func getMatchesByPeriod(tx *sqlx.Tx, leagueID util.UUIDAsBlob, from, to util.TimeAsTimestamp) ([]Match, error) {
	var matches []Match
	query := `
        SELECT Match.* FROM Match
        WHERE Match.LeagueID = ? AND Match.StartedAt >= ? AND Match.StartedAt < ?
        ORDER BY StartedAt ASC
        `

	if err := tx.Select(&matches, query, leagueID, from, to); err != nil {
		return nil, fmt.Errorf("could not fetch matches: %w", err)
	}

	for k := range matches {
		if err := injectEntries(tx, &matches[k]); err != nil {
			return nil, err
		}
	}

	return matches, nil
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
	query := `SELECT Match.* FROM Match WHERE Match.Seed = ? LIMIT 1`
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
		return nil, fmt.Errorf("could not fetch matches: %w", err)
	}

	for k := range matches {
		if err := injectEntries(tx, &matches[k]); err != nil {
			return nil, err
		}
	}

	return matches, nil
}

func (m *Match) GetPlayerAndOpponentEntries(playerID util.UUIDAsBlob) (MatchEntry, MatchEntry, error) {
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

func getFirstMatchStartOfLeague(tx *sqlx.Tx, leagueID util.UUIDAsBlob) (util.TimeAsTimestamp, error) {
	var ret util.NullTimeAsTimestamp
	if err := tx.Get(
		&ret,
		`SELECT StartedAt FROM Match
        WHERE StartedAt IS NOT NULL AND LeagueID = ?
        ORDER BY StartedAt LIMIT 1`,
		leagueID,
	); err != nil {
		return util.TimeAsTimestamp{}, err
	}

	return ret.Time, nil
}

func getMatchByID(tx *sqlx.Tx, id util.UUIDAsBlob) (Match, error) {
	var match Match
	query := `SELECT * FROM Match WHERE Match.ID = ? LIMIT 1`
	if err := tx.Get(&match, query, id); err != nil {
		return Match{}, fmt.Errorf("could not fetch match: %w", err)
	}

	if err := injectEntries(tx, &match); err != nil {
		return Match{}, err
	}

	return match, nil
}
