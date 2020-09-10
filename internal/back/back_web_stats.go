package back

import (
	"bytes"
	"fmt"
	"io"
	"kaepora/internal/util"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

const emptySVG = `<svg xmlns="http://www.w3.org/2000/svg"/>`

func (b *Back) GetRatingsDistributionGraph(shortcode string) ([]byte, error) {
	start := time.Now()
	defer func() { log.Printf("info: computed ratings stats in %s", time.Since(start)) }()

	bars, maxValue, err := b.getRatingsStats(shortcode, chart.Style{
		FontColor:   drawing.ColorBlack,
		FillColor:   drawing.ColorFromHex("285577"),
		StrokeColor: drawing.ColorFromHex("4c7899"),
		StrokeWidth: 1,
	})
	if err != nil {
		return nil, err
	}

	graph := chart.BarChart{
		Height: 300,
		Width:  600,
		Canvas: chart.Style{FillColor: chart.ColorTransparent},
		Background: chart.Style{
			FillColor: chart.ColorTransparent,
		},
		YAxis: chart.YAxis{
			Ticks: []chart.Tick{
				{Value: 0},
				{Value: maxValue},
			},
		},
		Bars: bars,
	}
	graph.BarWidth = (graph.Width - (len(bars) * graph.BarSpacing)) / len(bars)

	svg, err := renderChart(graph)
	if err != nil {
		return nil, err
	}

	return svg, nil
}

func (b *Back) getRatingsStats(
	shortcode string,
	barStyle chart.Style,
) (
	[]chart.Value, float64, error,
) {
	ratings, err := b.GetPlayerRatings(shortcode)
	if err != nil {
		return nil, 0, err
	}

	binWidth := 100 // width in rating units

	bins := make(map[int]int, 20)
	minBin, maxBin := math.MaxInt64, math.MinInt64
	maxValue := math.MinInt64
	valuesCount := 0

	for k := range ratings {
		valuesCount++

		r := int(math.Round(ratings[k].Rating/float64(binWidth)) * float64(binWidth))
		bins[r]++
		if r < minBin {
			minBin = r
		}
		if r > maxBin {
			maxBin = r
		}

		if bins[r] > maxValue {
			maxValue = bins[r]
		}
	}

	bars := make([]chart.Value, 0, len(bins))
	for i := minBin; i <= maxBin; i += binWidth {
		bars = append(bars, chart.Value{
			Value: float64(bins[i]) / float64(valuesCount),
			Label: strconv.Itoa(i),
			Style: barStyle,
		})
	}

	return bars, float64(maxValue) / float64(valuesCount), nil
}

func (b *Back) GetLeagueSeedTimesGraph(shortcode string) ([]byte, error) {
	var svg []byte
	start := time.Now()
	defer func() { log.Printf("info: computed seed completion stats in %s", time.Since(start)) }()

	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		league, err := getLeagueByShortCode(tx, shortcode)
		if err != nil {
			return err
		}

		svg, err = generateLeagueSeedTimesGraph(tx, league.ID)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return svg, nil
}

func generatePlayerSeedTimesGraph(tx *sqlx.Tx, playerID, leagueID util.UUIDAsBlob) ([]byte, error) {
	var times []int
	if err := tx.Select(
		&times,
		`SELECT (MatchEntry.EndedAt-MatchEntry.StartedAt) AS time
        FROM MatchEntry
        INNER JOIN Match ON(Match.ID = MatchEntry.MatchID)
        INNER JOIN MatchSession ON(MatchSession.ID = Match.MatchSessionID)
        WHERE
            MatchEntry.PlayerID = ?
            AND Match.LeagueID = ?
            AND MatchEntry.Status = ?
            AND MatchSession.Status = ?
        `,
		playerID, leagueID,
		MatchEntryStatusFinished, MatchSessionStatusClosed,
	); err != nil {
		return nil, err
	}

	return generateSeedTimesGraph(times)
}

func generateLeagueSeedTimesGraph(tx *sqlx.Tx, leagueID util.UUIDAsBlob) ([]byte, error) {
	var times []int
	if err := tx.Select(
		&times,
		`SELECT (MatchEntry.EndedAt-MatchEntry.StartedAt) AS time
        FROM MatchEntry
        INNER JOIN Match ON(Match.ID = MatchEntry.MatchID)
        INNER JOIN MatchSession ON(MatchSession.ID = Match.MatchSessionID)
        WHERE
            Match.LeagueID = ?
            AND MatchEntry.Status = ?
            AND MatchSession.Status = ?
        `,
		leagueID,
		MatchEntryStatusFinished, MatchSessionStatusClosed,
	); err != nil {
		return nil, err
	}

	return generateSeedTimesGraph(times)
}

func generateSeedTimesGraph(times []int) ([]byte, error) {
	style := chart.Style{
		FontColor:   drawing.ColorBlack,
		FillColor:   drawing.ColorFromHex("1d72aa"),
		StrokeColor: drawing.ColorFromHex("363636"),
		StrokeWidth: 1,
	}

	bars := []chart.Value{
		{Style: style, Label: "&lt; 2:00"},
		{Style: style, Label: "2:00"},
		{Style: style, Label: "2:30"},
		{Style: style, Label: "3:00"},
		{Style: style, Label: "3:30"},
		{Style: style, Label: "4:00"},
		{Style: style, Label: "4:30"},
		{Style: style, Label: "≥ 5:00"},
	}

	var hasValue bool
	for _, v := range times {
		i := ((v - (2 * 3600)) / (30 * 60)) + 1
		if i < 0 {
			i = 0
		}
		if i >= len(bars) {
			i = len(bars) - 1
		}

		bars[i].Value++
		hasValue = true
	}

	if !hasValue {
		// go-chart does not like it when all values are 0
		return []byte(emptySVG), nil
	}

	graph := chart.BarChart{
		Width:      620,
		Height:     200,
		Canvas:     chart.Style{FillColor: chart.ColorTransparent},
		Background: chart.Style{FillColor: chart.ColorTransparent},
		Bars:       bars,
	}

	graph.BarWidth = (graph.Width - (len(bars) * graph.BarSpacing)) / len(bars)

	return renderChart(graph)
}

func (b *Back) GetPlayerRatingGraph(playerName, shortcode string) ([]byte, error) {
	var ret []byte
	if err := b.transaction(func(tx *sqlx.Tx) error {
		player, err := getPlayerByName(tx, playerName)
		if err != nil {
			return err
		}

		league, err := getLeagueByShortCode(tx, shortcode)
		if err != nil {
			return err
		}

		ret, err = generateRRDGraph(tx, player.ID, league.ID)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (b *Back) GetPlayerSeedTimeGraph(playerName, shortcode string) ([]byte, error) {
	var ret []byte
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		player, err := getPlayerByName(tx, playerName)
		if err != nil {
			return err
		}

		league, err := getLeagueByShortCode(tx, shortcode)
		if err != nil {
			return err
		}

		ret, err = generatePlayerSeedTimesGraph(tx, player.ID, league.ID)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func generateRRDGraph(tx *sqlx.Tx, playerID, leagueID util.UUIDAsBlob) ([]byte, error) {
	var history []struct {
		RatingPeriodStartedAt int64
		Rating, Deviation     float64
	}
	if err := tx.Select(&history, `
        SELECT RatingPeriodStartedAt, Rating, Deviation FROM PlayerRatingHistory
        WHERE PlayerID = ? AND LeagueID = ? ORDER BY RatingPeriodStartedAt ASC`,
		playerID, leagueID,
	); err != nil {
		return nil, err
	}

	if len(history) < 2 {
		// Not enough data, return nothing.
		return []byte(emptySVG), nil
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
		Width:      620,
		Height:     200,
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

	return renderChart(graph)
}

type renderable interface {
	Render(chart.RendererProvider, io.Writer) error
}

func renderChart(r renderable) ([]byte, error) {
	var buf bytes.Buffer
	if err := r.Render(chart.SVG, &buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
