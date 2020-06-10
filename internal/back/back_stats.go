package back

import (
	"io"
	"kaepora/internal/util"

	"github.com/jmoiron/sqlx"
)

type StatsMisc struct {
	RegisteredPlayers, RankedPlayers, PlayersOnLeaderboard int
	SeedsPlayed, Forfeits, DoubleForfeits                  int
	FirstLadderRace                                        util.TimeAsDateTimeTZ
	AveragePlayersPerRace, MostPlayersInARace              int
}

func (b *Back) GetMiscStats(shortcode string) (misc StatsMisc, _ error) { // nolint:funlen
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
			{&misc.RegisteredPlayers, `SELECT COUNT(*) FROM Player`, nil},
			{&misc.RankedPlayers, `SELECT COUNT(*) FROM PlayerRating WHERE LeagueID = ?`, []interface{}{league.ID}},
			{
				&misc.PlayersOnLeaderboard,
				`SELECT COUNT(*) FROM PlayerRating WHERE LeagueID = ? AND Deviation < ?`,
				[]interface{}{league.ID, DeviationThreshold},
			},

			{
				&misc.SeedsPlayed,
				`SELECT COUNT(*) FROM Match WHERE LeagueID = ?`,
				[]interface{}{league.ID},
			},
			{
				&misc.Forfeits,
				`SELECT COUNT(*) FROM MatchEntry
                LEFT JOIN Match ON (MatchEntry.MatchID = Match.ID)
                WHERE Match.LeagueID = ? AND MatchEntry.Status = ?`,
				[]interface{}{league.ID, MatchEntryStatusForfeit},
			},
			{
				&misc.DoubleForfeits,
				`SELECT COUNT(*) FROM (SELECT COUNT(*) as cnt FROM "MatchEntry"
                LEFT JOIN Match ON (MatchEntry.MatchID = Match.ID)
                WHERE Match.LeagueID = ? AND MatchEntry.Status == ?
                GROUP BY MatchEntry.MatchID HAVING cnt > 1)`,
				[]interface{}{league.ID, MatchEntryStatusForfeit},
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
				`SELECT round(avg(json_array_length(PlayerIDs)))
                FROM MatchSession WHERE LeagueID = ?`,
				[]interface{}{league.ID},
			},
			{
				&misc.MostPlayersInARace,
				`SELECT max(json_array_length(PlayerIDs))
                FROM MatchSession WHERE LeagueID = ?`,
				[]interface{}{league.ID},
			},
		}

		for _, v := range queries {
			if err := tx.Get(v.Dst, v.Query, v.Args...); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return StatsMisc{}, err
	}

	return misc, nil
}

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
            SELECT SpoilerLog FROM Match WHERE LeagueID = ?
            AND EndedAt IS NOT NULL`, // HACK: ensure we don't leak stats on in-progress matches
			league.ID,
		)
		if err != nil {
			return err
		}

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
