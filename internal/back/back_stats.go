package back

import (
	"kaepora/internal/util"

	"github.com/jmoiron/sqlx"
)

type StatsMisc struct {
	RegisteredPlayers, RankedPlayers, PlayersOnLeaderboard int
	SeedsPlayed, Forfeits, DoubleForfeits                  int
	FirstLadderRace                                        util.TimeAsDateTimeTZ
	AveragePlayersPerRace, MostPlayersInARace              int
}

func (b *Back) GetMiscStats() (misc StatsMisc, _ error) {
	if err := b.transaction(func(tx *sqlx.Tx) error {
		std, err := getLeagueByShortCode(tx, "std")
		if err != nil {
			return err
		}

		queries := []struct {
			Dst   interface{}
			Query string
			Args  []interface{}
		}{
			{&misc.RegisteredPlayers, `SELECT COUNT(*) FROM Player`, nil},
			{&misc.RankedPlayers, `SELECT COUNT(*) FROM PlayerRating WHERE LeagueID = ?`, []interface{}{std.ID}},
			{
				&misc.PlayersOnLeaderboard,
				`SELECT COUNT(*) FROM PlayerRating WHERE LeagueID = ? AND Deviation < ?`,
				[]interface{}{std.ID, DeviationThreshold},
			},

			{&misc.SeedsPlayed, `SELECT COUNT(*) FROM Match`, nil},
			{&misc.Forfeits, `SELECT COUNT(*) FROM MatchEntry WHERE Status = ?`, []interface{}{MatchEntryStatusForfeit}},
			{
				&misc.DoubleForfeits,
				`SELECT COUNT(*) FROM (SELECT COUNT(*) as cnt FROM "MatchEntry"
                WHERE MatchEntry.Status == ? GROUP BY MatchID HAVING cnt > 1)`,
				[]interface{}{MatchEntryStatusForfeit},
			},

			{&misc.FirstLadderRace, `SELECT StartDate FROM MatchSession ORDER BY StartDate ASC LIMIT 1`, nil},
			{&misc.AveragePlayersPerRace, `SELECT round(avg(json_array_length(PlayerIDs))) FROM MatchSession`, nil},
			{&misc.MostPlayersInARace, `SELECT max(json_array_length(PlayerIDs)) FROM MatchSession`, nil},
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
