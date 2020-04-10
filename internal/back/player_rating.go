package back

import (
	"database/sql"
	"kaepora/internal/util"
	"time"

	"github.com/jmoiron/sqlx"
	glicko "github.com/zelenin/go-glicko2"
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

func (r PlayerRating) GlickoRating() *glicko.Rating {
	return glicko.NewRating(r.Rating, r.Deviation, r.Volatility)
}

// getPlayerRating get the current rating for a player in a league or creates
// and returns a default rating on the fly.
func getPlayerRating(
	tx *sqlx.Tx, playerID util.UUIDAsBlob, leagueID util.UUIDAsBlob,
) (PlayerRating, error) {
	var ret PlayerRating
	query := `SELECT * FROM PlayerRating WHERE PlayerID = ? AND LeagueID = ? LIMIT 1`
	err := tx.Get(&ret, query, playerID, leagueID)
	if err != nil {
		if err == sql.ErrNoRows {
			return PlayerRating{
				PlayerID:  playerID,
				LeagueID:  leagueID,
				CreatedAt: util.TimeAsTimestamp(time.Now()),

				Rating:     glicko.RATING_BASE_R,
				Deviation:  glicko.RATING_BASE_RD,
				Volatility: glicko.RATING_BASE_SIGMA,
			}, nil
		}
		return PlayerRating{}, err
	}

	return ret, nil
}
