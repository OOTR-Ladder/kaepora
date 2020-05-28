package web

import (
	"encoding/json"
	"io"
	"kaepora/internal/back"
	"kaepora/internal/generator/oot"
	"log"
	"net/http"
	"sort"
	"time"
)

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	misc, err := s.back.GetMiscStats()
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	seed, err := s.getSeedStats()
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	s.cache(w, "public", 1*time.Hour)
	s.response(w, r, http.StatusOK, "seed_stats.html", struct {
		Misc back.StatsMisc
		Seed statsSeed
	}{misc, seed})
}

func (s *Server) getSeedStats() (stats statsSeed, _ error) {
	start := time.Now()
	seedTotal := 0
	wothLocations := map[string]int{}
	wothItems := map[string]int{}
	barrenRegions := map[string]int{}

	if err := s.back.MapSpoilerLogs(func(raw io.Reader) error {
		seedTotal++

		var l oot.SpoilerLog
		dec := json.NewDecoder(raw)
		if err := dec.Decode(&l); err != nil {
			return err
		}

		var force, hookshot int
		for location, item := range l.WOTHLocations {
			wothLocations[location]++

			if item == "Progressive Strength Upgrade" {
				force++
				switch force {
				case 1:
					item = "Goron's Bracelet"
				case 2:
					item = "Silver Gauntlets"
				case 3:
					item = "Golden Gauntlets"
				}
			} else if item == "Progressive Hookshot" {
				hookshot++
				switch hookshot {
				case 1:
					item = "Hookshot"
				case 2:
					item = "Longshot"
				}
			}

			wothItems[string(item)]++
		}

		for _, name := range l.BarrenRegions {
			barrenRegions[name]++
		}

		return nil
	}); err != nil {
		return statsSeed{}, err
	}

	stats.Barren = locationPctFromMap(barrenRegions, seedTotal)
	stats.WOTH = locationPctFromMap(wothLocations, seedTotal)
	stats.WOTHItems = locationPctFromMap(wothItems, seedTotal)

	log.Printf("debug: computed stats for %d seeds in %s", seedTotal, time.Since(start))

	return stats, nil
}

func locationPctFromMap(m map[string]int, total int) (ret []locationPct) {
	for k, v := range m {
		ret = append(ret, locationPct{
			Name: k,
			Pct:  100.0 * (float64(v) / float64(total)),
		})
	}

	sort.Sort(byPctDesc(ret))

	return ret
}

type statsSeed struct {
	WOTH, WOTHItems, Barren []locationPct
}

type locationPct struct {
	Name string
	Pct  float64
}

type byPctDesc []locationPct

func (a byPctDesc) Len() int {
	return len([]locationPct(a))
}

func (a byPctDesc) Less(i, j int) bool {
	if a[i].Pct == a[j].Pct {
		return a[i].Name < a[j].Name
	}

	return a[i].Pct > a[j].Pct
}

func (a byPctDesc) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
