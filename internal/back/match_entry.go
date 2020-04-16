package back

import (
	"kaepora/internal/util"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type MatchEntry struct {
	MatchID   util.UUIDAsBlob
	PlayerID  util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp
	StartedAt util.NullTimeAsTimestamp
	EndedAt   util.NullTimeAsTimestamp
	Status    MatchEntryStatus
	Outcome   MatchEntryOutcome
	Comment   string
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

func NewMatchEntry(matchID, playerID util.UUIDAsBlob) MatchEntry {
	return MatchEntry{
		MatchID:   matchID,
		PlayerID:  playerID,
		CreatedAt: util.TimeAsTimestamp(time.Now()),

		Status:  MatchEntryStatusWaiting,
		Outcome: MatchEntryOutcomeDraw,
	}
}

func (m *MatchEntry) insert(tx *sqlx.Tx) error {
	query, args, err := squirrel.Insert("MatchEntry").SetMap(squirrel.Eq{
		"MatchID":   m.MatchID,
		"PlayerID":  m.PlayerID,
		"CreatedAt": m.CreatedAt,
		"StartedAt": m.StartedAt,
		"EndedAt":   m.EndedAt,
		"Status":    m.Status,
		"Outcome":   m.Outcome,
		"Comment":   m.Comment,
	}).ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func (m *MatchEntry) forfeit(against *MatchEntry, match *Match) {
	m.EndedAt = util.NewNullTimeAsTimestamp(time.Now())
	m.Status = MatchEntryStatusForfeit

	switch against.Status {
	case MatchEntryStatusWaiting:
	case MatchEntryStatusInProgress:
		m.Outcome = MatchEntryOutcomeLoss
		against.Outcome = MatchEntryOutcomeWin
	case MatchEntryStatusFinished:
		m.Outcome = MatchEntryOutcomeLoss
		against.Outcome = MatchEntryOutcomeWin
		match.end()
	case MatchEntryStatusForfeit:
		m.Outcome = MatchEntryOutcomeDraw
		against.Outcome = MatchEntryOutcomeDraw
		match.end()
	}
}

func (m *MatchEntry) complete(against *MatchEntry, match *Match) {
	m.EndedAt = util.NewNullTimeAsTimestamp(time.Now())
	m.Status = MatchEntryStatusFinished

	switch against.Status {
	case MatchEntryStatusWaiting:
		panic("unreachable: can't complete with an opponent in MatchEntryStatusWaiting")
	case MatchEntryStatusInProgress:
		m.Outcome = MatchEntryOutcomeWin
		against.Outcome = MatchEntryOutcomeLoss
	case MatchEntryStatusForfeit:
		m.Outcome = MatchEntryOutcomeWin
		against.Outcome = MatchEntryOutcomeLoss
		match.end()
	case MatchEntryStatusFinished:
		m.Outcome = MatchEntryOutcomeLoss
		against.Outcome = MatchEntryOutcomeWin
		match.end()
	}
}

func (m *MatchEntry) update(tx *sqlx.Tx) error {
	query, args, err := squirrel.Update("MatchEntry").SetMap(squirrel.Eq{
		"StartedAt": m.StartedAt,
		"EndedAt":   m.EndedAt,
		"Status":    m.Status,
		"Outcome":   m.Outcome,
		"Comment":   m.Comment,
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
