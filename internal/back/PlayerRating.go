package back

import (
	"kaepora/internal/util"
)

type PlayerRating struct {
	PlayerID  util.UUIDAsBlob
	LeagueID  util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp

	// Glicko-2
	Rating     float64
	Deviation  float64
	Volatility float64
}
