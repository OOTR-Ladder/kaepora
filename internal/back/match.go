package back

import (
	"kaepora/internal/util"
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

type MatchEntryStatus int

const ( // this is stored in DB, don't change values
	MatchEntryStatusWaiting    MatchEntryStatus = 0
	MatchEntryStatusInProgress MatchEntryStatus = 1
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
