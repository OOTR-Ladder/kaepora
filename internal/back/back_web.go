package back

import "github.com/jmoiron/sqlx"

func (b *Back) GetLeaderboardForShortcode(shortcode string, maxDeviation int) ([]LeaderboardEntry, error) {
	var ret []LeaderboardEntry

	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		league, err := getLeagueByShortCode(tx, shortcode)
		if err != nil {
			return err
		}

		return tx.Select(&ret, `
            SELECT
                Player.Name AS PlayerName,
                PlayerRating.Rating AS Rating,
                PlayerRating.Deviation AS Deviation
            FROM PlayerRating
            INNER JOIN Player ON(PlayerRating.PlayerID = Player.ID)
            WHERE PlayerRating.LeagueID = ? AND PlayerRating.Deviation < ?
            ORDER BY PlayerRating.Rating DESC
        `, league.ID, maxDeviation)
	}); err != nil {
		return nil, err
	}

	return ret, nil
}
