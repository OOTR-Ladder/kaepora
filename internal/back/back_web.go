package back

import (
	"kaepora/internal/util"
	"time"

	"github.com/jmoiron/sqlx"
)

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

func (b *Back) GetMatchSessionsAroundNow() ([]MatchSession, map[util.UUIDAsBlob]League, error) {
	var (
		sessions []MatchSession
		leagues  map[util.UUIDAsBlob]League
	)

	now := time.Now()

	if err := b.transaction(func(tx *sqlx.Tx) error {
		query := `SELECT * FROM MatchSession
        WHERE DATETIME(StartDate) >= DATETIME(?) AND
              DATETIME(StartDate) <= DATETIME(?) AND
              Status != ?
        ORDER BY StartDate ASC`
		if err := tx.Select(
			&sessions, query,
			now.Add(-24*time.Hour),
			now.Add(24*time.Hour),
			MatchSessionStatusClosed,
		); err != nil {
			return err
		}
		ids := make([]util.UUIDAsBlob, 0, len(sessions))
		for k := range sessions {
			ids = append(ids, sessions[k].LeagueID)
		}

		query, args, err := sqlx.In(`SELECT * FROM League WHERE ID IN (?)`, ids)
		if err != nil {
			return err
		}
		query = tx.Rebind(query)
		sLeagues := make([]League, 0, len(ids))
		leagues = make(map[util.UUIDAsBlob]League, len(ids))
		if err := tx.Select(&sLeagues, query, args...); err != nil {
			return err
		}
		for _, v := range sLeagues {
			leagues[v.ID] = v
		}

		return nil
	}); err != nil {
		return nil, nil, err
	}

	return sessions, leagues, nil
}
