package web

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

func (s *Server) statsRatings(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() { log.Printf("info: computed ratings stats in %s", time.Since(start)) }()

	bars, maxValue, err := s.getRatingsStats("std", chart.Style{
		FontColor:   drawing.ColorBlack,
		FillColor:   drawing.ColorFromHex("285577"),
		StrokeColor: drawing.ColorFromHex("4c7899"),
		StrokeWidth: 1,
	})

	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
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

	s.cache(w, "public", 1*time.Hour)
	w.Header().Set("Content-Type", "image/svg+xml")
	if err := graph.Render(chart.SVG, w); err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}
}

func (s *Server) getRatingsStats(
	shortcode string,
	barStyle chart.Style,
) (
	[]chart.Value, float64, error,
) {
	ratings, err := s.back.GetPlayerRatings(shortcode)
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
