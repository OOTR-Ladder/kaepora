package web

import (
	"encoding/hex"
	"html/template"
	"kaepora/internal/back"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi"
)

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	shortcode := chi.URLParam(r, "shortcode")
	misc, err := s.back.GetMiscStats(shortcode)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}
	log.Printf("info: computed misc stats in %s", time.Since(start))

	seed, err := s.getSeedStats(shortcode)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	attendance, err := s.getAttendanceStats(shortcode)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	seedTime, err := s.back.GetLeagueSeedTimesGraph(shortcode)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	ratings, err := s.back.GetRatingsDistributionGraph(shortcode)
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 1*time.Hour)
	s.response(w, r, http.StatusOK, "stats.html", struct {
		Misc                           back.StatsMisc
		Attendance                     []attendanceEntry
		Seed                           statsSeed
		ShortCode                      string
		ExtendedStats                  bool
		SeedTimes, RatingsDistribution template.HTML
	}{
		misc, attendance, seed, shortcode, shortcode == "shu",
		template.HTML(seedTime), template.HTML(ratings), // nolint:gosec
	})
}

type attendanceEntry struct {
	From, To    string    // HH:MM, always UTC
	Color       [7]string // color to use in heatmap
	Average     [7]int    // average player count per dow
	Accumulated [7]int    // total player count per dow
	Counted     [7]int    // sessions counted in this slot
}

func (s *Server) getAttendanceStats(shortcode string) ([]attendanceEntry, error) {
	start := time.Now()
	defer func() { log.Printf("info: computed attendance in %s", time.Since(start)) }()

	bins := 3
	ret := make([]attendanceEntry, bins)
	for i := 0; i < len(ret); i++ {
		t := time.Unix(int64(24/bins*i*3600), 0).UTC()
		ret[i].From = t.Format("15")
		ret[i].To = t.Add(time.Duration(24/bins) * time.Hour).Format("15")
	}

	max := math.MinInt64
	if err := s.back.MapMatchSessions(shortcode, func(m back.MatchSession) error {
		players := len(m.PlayerIDs)
		if players > max {
			max = players
		}

		t := m.StartDate.Time().UTC()
		bin := t.Hour() / (24 / bins)
		dow := t.Weekday() - 1
		if dow < 0 {
			dow = 6
		}

		ret[bin].Accumulated[dow] += players
		ret[bin].Counted[dow]++

		return nil
	}); err != nil {
		return nil, err
	}

	for i := 0; i < len(ret); i++ {
		for dow := 0; dow < 7; dow++ {
			if ret[i].Counted[dow] > 0 {
				ret[i].Average[dow] = ret[i].Accumulated[dow] / ret[i].Counted[dow]
			}

			if max > 0 {
				ret[i].Color[dow] = lerpColor(float64(ret[i].Average[dow]) / float64(max))
			} else {
				ret[i].Color[dow] = lerpColor(0)
			}
		}
	}

	return ret, nil
}

func lerpColor(r float64) string {
	lerp := func(v0, v1, r float64) float64 {
		return v0*(1-r) + v1*r
	}

	// lerp white to red
	a := [3]float64{1, 1, 1}
	b := [3]float64{1, 0, 0}

	return "#" + hex.EncodeToString([]byte{
		byte(lerp(a[0], b[0], r) * 255.0),
		byte(lerp(a[1], b[1], r) * 255.0),
		byte(lerp(a[2], b[2], r) * 255.0),
	})
}
