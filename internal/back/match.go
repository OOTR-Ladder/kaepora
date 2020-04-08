package back

import (
	"database/sql"
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

	Generator string
	Settings  string
	Seed      string

	Entries []MatchEntry `db:"-"`
}

func NewMatch(tx *sqlx.Tx, session MatchSession) (Match, error) {
	league, err := getLeagueByID(tx, session.LeagueID)
	if err != nil {
		return Match{}, err
	}
	game, err := getGameByID(tx, league.GameID)
	if err != nil {
		return Match{}, err
	}

	return Match{
		ID:        util.NewUUIDAsBlob(),
		CreatedAt: util.TimeAsTimestamp(time.Now()),

		LeagueID:       session.LeagueID,
		MatchSessionID: session.ID,

		Generator: game.Generator,
		Settings:  league.Settings,
	}, nil
}

func (m *Match) GetPlayerEntry(playerID util.UUIDAsBlob) (MatchEntry, error) {
	for k := range m.Entries {
		if m.Entries[k].PlayerID == playerID {
			return m.Entries[k], nil
		}
	}

	return MatchEntry{}, sql.ErrNoRows
}

type MatchEntryStatus int

const ( // this is stored in DB, don't change values
	MatchEntryStatusWaiting    MatchEntryStatus = 0 // MatchSession preparing
	MatchEntryStatusInProgress MatchEntryStatus = 1 // MatchSession in progress
	MatchEntryStatusFinished   MatchEntryStatus = 2
	MatchEntryStatusForfeit    MatchEntryStatus = 3 // (automatic loss)
)

type MatchEntryOutcome int

const ( // this is stored in DB, don't change values
	MatchEntryOutcomeLoss MatchEntryOutcome = -1
	MatchEntryOutcomeDraw MatchEntryOutcome = 0
	MatchEntryOutcomeWin  MatchEntryOutcome = 1
)

type MatchEntry struct {
	MatchID   util.UUIDAsBlob
	PlayerID  util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp
	StartedAt util.NullTimeAsTimestamp
	EndedAt   util.NullTimeAsTimestamp

	Status  MatchEntryStatus
	Outcome MatchEntryOutcome
}

func (m *MatchEntry) forfeit() {
	m.EndedAt = util.NewNullTimeAsTimestamp(time.Now())
	m.Status = MatchEntryStatusForfeit
}

func (m *MatchEntry) update(tx *sqlx.Tx) error {
	query, args, err := squirrel.Update("MatchEntry").SetMap(squirrel.Eq{
		"StartedAt": m.StartedAt,
		"EndedAt":   m.EndedAt,
		"Status":    m.Status,
		"Outcome":   m.Outcome,
	}).Where(squirrel.Eq{
		"MatchEntry.MatchID":  m.MatchID,
		"MatchEntry.PlayerID": m.PlayerID,
	}).ToSql()
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
    LEFT JOIN MatchEntry ON (MatchEntry.MatchID = Match.ID)
    WHERE Match.MatchSessionID = ? AND MatchEntry.PlayerID = ?
    LIMIT 1
    `
	if err := tx.Get(&match, query, session.ID, player.ID); err != nil {
		return Match{}, err
	}

	query = `SELECT * FROM MatchEntry WHERE MatchEntry.MatchID = ? LIMIT 1`
	if err := tx.Select(&match.Entries, query, match.ID); err != nil {
		return Match{}, err
	}

	return match, nil
}
