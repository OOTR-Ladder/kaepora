package back

import (
	"database/sql"
	"io"
	"kaepora/internal/util"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

// StatsMisc holds miscellaneous stats about a league.
type StatsMisc struct {
	RankedPlayers, PlayersOnLeaderboard       int
	SeedsPlayed, Forfeits, DoubleForfeits     int
	FirstLadderRace                           util.TimeAsDateTimeTZ
	TotalSeedTime                             time.Duration
	AveragePlayersPerRace, MostPlayersInARace int
}

func (b *Back) GetMiscStats(shortcode string) (misc StatsMisc, _ error) { // nolint:funlen
	start := time.Now()
	defer func() { log.Printf("info: computed misc stats in %s", time.Since(start)) }()

	if err := b.transaction(func(tx *sqlx.Tx) error {
		league, err := getLeagueByShortCode(tx, shortcode)
		if err != nil {
			return err
		}

		queries := []struct {
			Dst   interface{}
			Query string
			Args  []interface{}
		}{
			{&misc.RankedPlayers, `SELECT COUNT(*) FROM PlayerRating WHERE LeagueID = ?`, []interface{}{league.ID}},
			{
				&misc.PlayersOnLeaderboard,
				`SELECT COUNT(*) FROM PlayerRating WHERE LeagueID = ? AND Deviation < ?`,
				[]interface{}{league.ID, DeviationThreshold},
			},

			{
				&misc.SeedsPlayed,
				`SELECT COUNT(*) FROM Match
                INNER JOIN MatchSession ON (Match.MatchSessionID = MatchSession.ID)
                WHERE MatchSession.LeagueID = ? AND MatchSession.Status = ?`,
				[]interface{}{league.ID, MatchSessionStatusClosed},
			},
			{
				&misc.Forfeits,
				`SELECT COUNT(*) FROM MatchEntry
                INNER JOIN Match ON (MatchEntry.MatchID = Match.ID)
                INNER JOIN MatchSession ON (Match.MatchSessionID = MatchSession.ID)
                WHERE Match.LeagueID = ? AND MatchEntry.Status = ? AND MatchSession.Status = ?`,
				[]interface{}{league.ID, MatchEntryStatusForfeit, MatchSessionStatusClosed},
			},
			{
				&misc.DoubleForfeits,
				`SELECT COUNT(*) FROM (SELECT COUNT(*) as cnt FROM "MatchEntry"
                INNER JOIN Match ON (MatchEntry.MatchID = Match.ID)
                INNER JOIN MatchSession ON (Match.MatchSessionID = MatchSession.ID)
                WHERE Match.LeagueID = ? AND MatchEntry.Status == ? AND MatchSession.Status = ?
                GROUP BY MatchEntry.MatchID HAVING cnt > 1)`,
				[]interface{}{league.ID, MatchEntryStatusForfeit, MatchSessionStatusClosed},
			},
			{
				&misc.TotalSeedTime,
				`SELECT COALESCE(? * SUM(MatchEntry.EndedAt - MatchEntry.StartedAt), 0) FROM MatchEntry
                INNER JOIN Match ON (MatchEntry.MatchID = Match.ID)
                INNER JOIN MatchSession ON (Match.MatchSessionID = MatchSession.ID)
                WHERE Match.LeagueID = ? AND MatchSession.Status = ?`,
				[]interface{}{time.Second, league.ID, MatchSessionStatusClosed},
			},
			{
				&misc.FirstLadderRace,
				`SELECT StartDate FROM MatchSession
                WHERE LeagueID = ?
                ORDER BY StartDate ASC LIMIT 1`,
				[]interface{}{league.ID},
			},
			{
				&misc.AveragePlayersPerRace,
				`SELECT COALESCE(round(avg(json_array_length(PlayerIDs))), 0)
                FROM MatchSession WHERE LeagueID = ? AND MatchSession.Status = ?`,
				[]interface{}{league.ID, MatchSessionStatusClosed},
			},
			{
				&misc.MostPlayersInARace,
				`SELECT COALESCE(max(json_array_length(PlayerIDs)), 0)
                FROM MatchSession WHERE LeagueID = ? AND MatchSession.Status = ?`,
				[]interface{}{league.ID, MatchSessionStatusClosed},
			},
		}

		for _, v := range queries {
			if err := tx.Get(v.Dst, v.Query, v.Args...); err != nil {
				// Ignore empty results, that's just an empty league.
				if err != sql.ErrNoRows {
					return err
				}
			}
		}

		return nil
	}); err != nil {
		return StatsMisc{}, err
	}

	return misc, nil
}

// MapSpoilerLogs applies a function on all the spoilers log of completed
// matches of a league.
func (b *Back) MapSpoilerLogs(
	shortcode string,
	cb func(io.Reader) error,
) error {
	return b.transaction(func(tx *sqlx.Tx) error {
		league, err := getLeagueByShortCode(tx, shortcode)
		if err != nil {
			return err
		}

		rows, err := tx.Query(`
            SELECT Match.SpoilerLog FROM Match
            INNER JOIN MatchSession ON (Match.MatchSessionID = MatchSession.ID)
            WHERE Match.LeagueID = ? AND MatchSession.Status = ? AND Match.EndedAt IS NOT NULL`,
			league.ID, MatchSessionStatusClosed,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		var buf util.ZLIBBlob
		for rows.Next() {
			if err := rows.Scan(&buf); err != nil {
				return err
			}

			if err := cb(buf.Uncompressed()); err != nil {
				return err
			}

			buf = buf[:0]
		}

		return rows.Err()
	})
}
