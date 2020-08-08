package back

// This file contains functions specific to the web frontend.
// Please do not call them outside of the webserver.

import (
	"bytes"
	"fmt"
	"html/template"
	"kaepora/internal/util"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

func (b *Back) GetLeaderboardForShortcode(shortcode string, maxDeviation int) (out []LeaderboardEntry, _ error) {
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		out, err = b.getLeaderboardForShortcode(tx, shortcode, maxDeviation)
		return err
	}); err != nil {
		return nil, err
	}

	return out, nil
}

func (b *Back) getLeaderboardForShortcode(
	tx *sqlx.Tx,
	shortcode string,
	maxDeviation int,
) ([]LeaderboardEntry, error) {
	league, err := getLeagueByShortCode(tx, shortcode)
	if err != nil {
		return nil, err
	}

	bans := b.config.DiscordBannedUserIDs
	if len(bans) == 0 {
		bans = []string{"0"}
	}

	query, args, err := sqlx.In(`
            SELECT
                Player.Name AS PlayerName,
                Player.StreamURL AS PlayerStreamURL,
                PlayerRating.Rating AS Rating,
                PlayerRating.Deviation AS Deviation,
                SUM(CASE WHEN MatchEntry.Outcome = ? THEN 1 ELSE 0 END) AS Wins,
                SUM(CASE WHEN MatchEntry.Outcome = ? THEN 1 ELSE 0 END) AS Losses,
                SUM(CASE WHEN MatchEntry.Outcome = ? THEN 1 ELSE 0 END) AS Draws,
                SUM(CASE WHEN MatchEntry.Status = ? THEN 1 ELSE 0 END) AS Forfeits
            FROM PlayerRating
            INNER JOIN Player ON(PlayerRating.PlayerID = Player.ID)
            LEFT JOIN MatchEntry ON(PlayerRating.PlayerID = MatchEntry.PlayerID AND MatchEntry.Status != ?)
            LEFT JOIN Match ON(Match.ID = MatchEntry.MatchID)
            WHERE
                Match.LeagueID = ?
                AND PlayerRating.LeagueID = ?
                AND PlayerRating.Deviation < ?
                AND (Player.DiscordID NOT IN(?) OR Player.DiscordID IS NULL)
            GROUP BY Player.ID
            ORDER BY (PlayerRating.Rating - (2*PlayerRating.Deviation)) DESC
        `,
		MatchEntryOutcomeWin,
		MatchEntryOutcomeLoss,
		MatchEntryOutcomeDraw,
		MatchEntryStatusForfeit,
		MatchEntryStatusInProgress,
		league.ID, league.ID,
		maxDeviation,
		bans,
	)
	if err != nil {
		return nil, err
	}

	query = tx.Rebind(query)
	var ret []LeaderboardEntry
	if err := tx.Select(&ret, query, args...); err != nil {
		return nil, err
	}

	return ret, nil
}

func (b *Back) GetLeagues() (ret []League, _ error) {
	return ret, b.transaction(func(tx *sqlx.Tx) (err error) {
		ret, err = getLeagues(tx)
		return err
	})
}

// GetLeaguesMap returns the list of leagues indexed by their ID.
func (b *Back) GetLeaguesMap() (map[util.UUIDAsBlob]League, error) {
	s, err := b.GetLeagues()
	if err != nil {
		return nil, err
	}

	ret := make(map[util.UUIDAsBlob]League, len(s))
	for k := range s {
		ret[s[k].ID] = s[k]
	}

	return ret, nil
}

func (b *Back) GetLeagueByShortcode(shortcode string) (ret League, _ error) {
	return ret, b.transaction(func(tx *sqlx.Tx) (err error) {
		ret, err = getLeagueByShortCode(tx, shortcode)
		return err
	})
}

func (b *Back) GetLeague(id util.UUIDAsBlob) (ret League, _ error) {
	return ret, b.transaction(func(tx *sqlx.Tx) (err error) {
		ret, err = getLeagueByID(tx, id)
		return err
	})
}

// GetMatchSessions returns sessions in a timeframe that have the given
// statuses ordered by the given SQL clause.
// The leagues the sessions belong to are returned indexed by their ID.
func (b *Back) GetMatchSessions(
	fromDate, toDate time.Time,
	statuses []MatchSessionStatus,
	order string,
) ([]MatchSession, map[util.UUIDAsBlob]League, error) {
	var (
		sessions []MatchSession
		leagues  map[util.UUIDAsBlob]League
	)

	if err := b.transaction(func(tx *sqlx.Tx) error {
		query, args, err := sqlx.In(`
            SELECT * FROM MatchSession
            WHERE DATETIME(StartDate) >= DATETIME(?) AND
                  DATETIME(StartDate) <= DATETIME(?) AND
                  Status IN(?)`,
			fromDate, toDate, statuses,
		)
		if err != nil {
			return err
		}
		query = tx.Rebind(query) + ` ORDER BY ` + order

		if err := tx.Select(&sessions, query, args...); err != nil {
			return err
		}
		ids := make([]util.UUIDAsBlob, 0, len(sessions))
		for k := range sessions {
			ids = append(ids, sessions[k].LeagueID)
		}

		if len(ids) == 0 {
			return nil
		}

		query, args, err = sqlx.In(`SELECT * FROM League WHERE ID IN (?)`, ids)
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

func (b *Back) GetPlayerRatings(shortcode string) (ret []PlayerRating, _ error) {
	return ret, b.transaction(func(tx *sqlx.Tx) (err error) {
		league, err := getLeagueByShortCode(tx, shortcode)
		if err != nil {
			return err
		}

		if err := tx.Select(
			&ret,
			`SELECT * FROM PlayerRating WHERE LeagueID = ? AND Deviation < ?`,
			league.ID, DeviationThreshold,
		); err != nil {
			return err
		}

		return nil
	})
}

// GetMatchSession returns a single session and all its matches and players indexed by their ID.
func (b *Back) GetMatchSession(id util.UUIDAsBlob) (
	session MatchSession,
	matches []Match,
	players map[util.UUIDAsBlob]Player,
	_ error,
) {
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		session, err = getMatchSessionByID(tx, id)
		if err != nil {
			return err
		}

		matches, err = getMatchesBySessionID(tx, id)
		if err != nil {
			return err
		}

		players, err = getPlayersByMatches(tx, matches)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return MatchSession{}, nil, nil, err
	}

	return session, matches, players, nil
}

func (b *Back) GetMatch(id util.UUIDAsBlob) (match Match, _ error) {
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		match, err = getMatchByID(tx, id)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return Match{}, err
	}

	return match, nil
}

func (b *Back) GetPlayerByName(name string) (player Player, _ error) {
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		player, err = getPlayerByName(tx, name)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return Player{}, err
	}

	return player, nil
}

// PlayerPerformance is the overall performance of a single Player on a single League.
type PlayerPerformance struct {
	LeagueID util.UUIDAsBlob
	Rating   PlayerRating
	WLChart  template.HTML // Win/Losses pie chart, SVG
	RRDChart template.HTML // Rating/Rating Deviation graph, SVG

	Wins, Losses, Draws, Forfeits int
}

// PlayerStats holds the performances of a single Player over all its Leagues.
type PlayerStats struct {
	Performances []PlayerPerformance
}

func (b *Back) GetPlayerStats(playerID util.UUIDAsBlob) (stats PlayerStats, _ error) {
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		if err := tx.Select(
			&stats.Performances,
			`SELECT
                Match.LeagueID,
                SUM(CASE WHEN MatchEntry.Outcome = ? THEN 1 ELSE 0 END) AS Wins,
                SUM(CASE WHEN MatchEntry.Outcome = ? THEN 1 ELSE 0 END) AS Losses,
                SUM(CASE WHEN MatchEntry.Outcome = ? THEN 1 ELSE 0 END) AS Draws,
                SUM(CASE WHEN MatchEntry.Status  = ? THEN 1 ELSE 0 END) AS Forfeits
            FROM MatchEntry
                LEFT JOIN Match ON(Match.ID = MatchEntry.MatchID)
            WHERE MatchEntry.Status != ? AND MatchEntry.PlayerID = ?
            GROUP BY Match.LeagueID
            `,
			MatchEntryOutcomeWin,
			MatchEntryOutcomeLoss,
			MatchEntryOutcomeDraw,
			MatchEntryStatusForfeit,
			MatchEntryStatusInProgress,
			playerID,
		); err != nil {
			return err
		}

		var ratings []PlayerRating
		query := `SELECT * FROM PlayerRating WHERE PlayerID = ?`
		if err := tx.Select(&ratings, query, playerID); err != nil {
			return err
		}

		for k := range stats.Performances {
			stats.Performances[k].WLChart, err = generateWLChart(
				float64(stats.Performances[k].Wins),
				float64(stats.Performances[k].Losses),
			)
			if err != nil {
				return err
			}

			stats.Performances[k].RRDChart, err = generateRRDChart(tx, playerID, stats.Performances[k].LeagueID)
			if err != nil {
				return err
			}

			for l := range ratings {
				if stats.Performances[k].LeagueID == ratings[l].LeagueID {
					stats.Performances[k].Rating = ratings[l]
				}
			}
		}

		return nil
	}); err != nil {
		return PlayerStats{}, err
	}

	return stats, nil
}

func generateWLChart(wins, losses float64) (template.HTML, error) {
	pie := chart.PieChart{
		Width:      200,
		Height:     200,
		Canvas:     chart.Style{FillColor: chart.ColorTransparent},
		Background: chart.Style{FillColor: chart.ColorTransparent},
		Values: []chart.Value{
			{
				Value: wins,
				Label: fmt.Sprintf("Wins (%.0f %%)", (wins/(losses+wins))*100.0),
				Style: chart.Style{FillColor: drawing.ColorFromHex("FDF1DC")},
			},
			{
				Value: losses,
				Label: fmt.Sprintf("Losses (%.0f %%)", (losses/(losses+wins))*100.0),
				Style: chart.Style{FillColor: drawing.ColorFromHex("F2E1D7")},
			},
		},
	}

	var buf bytes.Buffer
	if err := pie.Render(chart.SVG, &buf); err != nil {
		return template.HTML(""), err
	}

	return template.HTML(buf.Bytes()), nil // nolint:gosec
}

func generateRRDChart(tx *sqlx.Tx, playerID, leagueID util.UUIDAsBlob) (template.HTML, error) {
	var history []struct {
		RatingPeriodStartedAt int64
		Rating, Deviation     float64
	}
	if err := tx.Select(&history, `
        SELECT RatingPeriodStartedAt, Rating, Deviation FROM PlayerRatingHistory
        WHERE PlayerID = ? AND LeagueID = ? ORDER BY RatingPeriodStartedAt ASC`,
		playerID, leagueID,
	); err != nil {
		return template.HTML(""), err
	}

	r := make([]float64, len(history))
	rd := make([]float64, len(history))
	period := make([]float64, len(history))

	for i := range history {
		period[i] = float64(history[i].RatingPeriodStartedAt)
		r[i] = history[i].Rating
		rd[i] = history[i].Deviation
	}

	graph := chart.Chart{
		Width:      864,
		Height:     256,
		Canvas:     chart.Style{FillColor: chart.ColorTransparent},
		Background: chart.Style{FillColor: chart.ColorTransparent},
		XAxis: chart.XAxis{
			TickPosition: chart.TickPositionBetweenTicks,
			ValueFormatter: func(v interface{}) string {
				y, w := time.Unix(int64(v.(float64)), 0).ISOWeek()
				return fmt.Sprintf("%d w%d", y, w)
			},
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				Name:    "Rating",
				XValues: period,
				YValues: r,
			},
			chart.ContinuousSeries{
				Name:    "Deviation",
				YAxis:   chart.YAxisSecondary,
				XValues: period,
				YValues: rd,
			},
		},
	}

	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
	}

	var buf bytes.Buffer
	if err := graph.Render(chart.SVG, &buf); err != nil {
		return template.HTML(""), err
	}

	return template.HTML(buf.Bytes()), nil // nolint:gosec
}
