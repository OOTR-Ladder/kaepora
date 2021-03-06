package back

import (
	"database/sql"
	"kaepora/internal/util"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	glicko "github.com/zelenin/go-glicko2"
)

// DeviationThreshold is value under which players must stay to appear on the
// leaderboards.
const DeviationThreshold = 200

type PlayerRating struct {
	PlayerID  util.UUIDAsBlob
	LeagueID  util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp

	// Used only on PlayerRatingHistory table
	RatingPeriodStartedAt util.TimeAsTimestamp

	// Glicko-2
	Rating     float64
	Deviation  float64
	Volatility float64
}

// Range returns the range in which we believe the true player rating is
// with 95 % accuracy.
func (r PlayerRating) Range() (int, int) {
	return int(r.Rating - (2.0 * r.Deviation)),
		int(r.Rating + (2.0 * r.Deviation))
}

func (r PlayerRating) GlickoRating() *glicko.Rating {
	return glicko.NewRating(r.Rating, r.Deviation, r.Volatility)
}

func (r *PlayerRating) SetRating(g *glicko.Rating) {
	r.Rating = g.R()
	r.Deviation = g.Rd()
	r.Volatility = g.Sigma()
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
	tx *sqlx.Tx,
	leagueID util.UUIDAsBlob,
	periodStart util.TimeAsTimestamp,
) (map[util.UUIDAsBlob]*glicko.Player, error) {
	query := `
        SELECT * FROM PlayerRatingHistory
        WHERE LeagueID = ? AND RatingPeriodStartedAt = ?`

	var ratings []PlayerRating
	if err := tx.Select(&ratings, query, leagueID, periodStart); err != nil {
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

func (r PlayerRating) upsertHistory(tx *sqlx.Tx, ratingPeriodStartedAt util.TimeAsTimestamp) error {
	query, args, err := squirrel.Insert("PlayerRatingHistory").SetMap(squirrel.Eq{
		"PlayerID":              r.PlayerID,
		"LeagueID":              r.LeagueID,
		"CreatedAt":             r.CreatedAt,
		"RatingPeriodStartedAt": ratingPeriodStartedAt,
		"Rating":                r.Rating,
		"Deviation":             r.Deviation,
		"Volatility":            r.Volatility,
	}).ToSql()
	if err != nil {
		return err
	}

	query += `
        ON CONFLICT(PlayerID, LeagueID, RatingPeriodStartedAt)
        DO UPDATE SET
            Rating=excluded.Rating,
            Deviation=excluded.Deviation,
            Volatility=excluded.Volatility
    `

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}
