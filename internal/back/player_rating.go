package back

import (
	"database/sql"
	"kaepora/internal/util"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	glicko "github.com/zelenin/go-glicko2"
)

// TODO seems like an OK cutoff right now, but will need to be change
// later I've seen a RD of 50 being the average for active players.
const DeviationThreshold = 220

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

func NewPlayerRating(playerID, leagueID util.UUIDAsBlob) PlayerRating {
	return PlayerRating{
		PlayerID:  playerID,
		LeagueID:  leagueID,
		CreatedAt: util.TimeAsTimestamp(time.Now()),

		Rating:     glicko.RATING_BASE_R,
		Deviation:  glicko.RATING_BASE_RD,
		Volatility: glicko.RATING_BASE_SIGMA,
	}
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
			return NewPlayerRating(playerID, leagueID), nil
		}
		return PlayerRating{}, err
	}

	return ret, nil
}

// getGlickoRatingsForLeague returns players indexed by Player ID.
func getGlickoPlayersForLeague(
	tx *sqlx.Tx, leagueID util.UUIDAsBlob,
) (map[util.UUIDAsBlob]*glicko.Player, error) {
	query := `SELECT * FROM PlayerRating WHERE PlayerRating.LeagueID = ?`
	var ratings []PlayerRating
	if err := tx.Select(&ratings, query, leagueID); err != nil {
		return nil, err
	}

	ret := make(map[util.UUIDAsBlob]*glicko.Player, len(ratings))
	for k := range ratings {
		ret[ratings[k].PlayerID] = glicko.NewPlayer(ratings[k].GlickoRating())
	}

	return ret, nil
}

func (r PlayerRating) upsert(tx *sqlx.Tx) error {
	query, args, err := squirrel.Insert("PlayerRating").SetMap(squirrel.Eq{
		"PlayerID":   r.PlayerID,
		"LeagueID":   r.LeagueID,
		"CreatedAt":  r.CreatedAt,
		"Rating":     r.Rating,
		"Deviation":  r.Deviation,
		"Volatility": r.Volatility,
	}).ToSql()
	if err != nil {
		return err
	}

	query += ` ON CONFLICT(PlayerID, LeagueID) DO UPDATE SET
        Rating=excluded.Rating,
        Deviation=excluded.Deviation,
        Volatility=excluded.Volatility
    `

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func (r PlayerRating) insertHistory(tx *sqlx.Tx) error {
	query, args, err := squirrel.Insert("PlayerRatingHistory").SetMap(squirrel.Eq{
		"PlayerID":   r.PlayerID,
		"LeagueID":   r.LeagueID,
		"CreatedAt":  r.CreatedAt,
		"Rating":     r.Rating,
		"Deviation":  r.Deviation,
		"Volatility": r.Volatility,
	}).ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}
