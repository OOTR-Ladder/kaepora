package back

import (
	"kaepora/internal/util"
)

type Match struct {
	ID        util.UUIDAsBlob
	LeagueID  util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp
	StartedAt util.NullTimeAsTimestamp
	EndedAt   util.NullTimeAsTimestamp

	Generator string
	Settings  string
	Seed      string

	Entries []MatchEntry `db:"-"`
}

type MatchStatus int

const ( // this is stored in DB, don't change values
	MatchStatusWaiting    = 0
	MatchStatusInProgress = 1
	MatchStatusForfeit    = 2 // (hard loss)
	MatchStatusCanceled   = 3 // (no impact on score)
)

type MatchOutcome int

const ( // this is stored in DB, don't change values
	MatchOutcomeLoss = -1
	MatchOutcomeDraw = 0
	MatchOutcomeWin  = 1
)

type MatchEntry struct {
	MatchID   util.UUIDAsBlob
	PlayerID  util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp
	StartedAt util.NullTimeAsTimestamp
	EndedAt   util.NullTimeAsTimestamp

	Status  MatchStatus
	Outcome MatchOutcome
}
