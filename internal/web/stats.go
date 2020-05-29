package web

import (
	"encoding/json"
	"io"
	"kaepora/internal/back"
	"kaepora/internal/generator/oot"
	"log"
	"net/http"
	"sort"
	"strings"
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
	s.response(w, r, http.StatusOK, "stats.html", struct {
		Misc back.StatsMisc
		Seed statsSeed
	}{misc, seed})
}

func (s *Server) getSeedStats() (statsSeed, error) { // nolint:funlen
	start := time.Now()
	seedTotal := 0

	wothLocations := map[string]int{}
	wothItems := map[string]int{}
	barrenRegions := map[string]int{}

	locationItems := map[string]int{}
	locationSmallKeys := map[string]int{}
	locationBossKeys := map[string]int{}
	locationJunk := map[string]int{}
	locationIceTraps := map[string]int{}

	if err := s.back.MapSpoilerLogs("std", func(raw io.Reader) error {
		seedTotal++

		var l oot.SpoilerLog
		dec := json.NewDecoder(raw)
		if err := dec.Decode(&l); err != nil {
			return err
		}

		progressive := map[string]int{}
		for location, item := range l.WOTHLocations {
			wothLocations[location]++

			if strings.HasPrefix(string(item), "Progressive") {
				wothItems[progressiveItemName(progressive, string(item))]++
			} else {
				wothItems[string(item)]++
			}
		}

		for _, name := range l.BarrenRegions {
			barrenRegions[name]++
		}

		for name, item := range l.Locations {
			switch getItemCategory(string(item)) {
			case itemCategoryItem:
				locationItems[name]++
			case itemCategoryJunk:
				locationJunk[name]++
			case itemCategorySmallKey:
				locationSmallKeys[name]++
			case itemCategoryBossKey:
				locationBossKeys[name]++
			case itemCategoryIceTrap:
				locationIceTraps[name]++
			}
		}

		return nil
	}); err != nil {
		return statsSeed{}, err
	}

	defer log.Printf("debug: computed stats for %d seeds in %s", seedTotal, time.Since(start))
	return statsSeed{
		Barren:            locationPctFromMap(barrenRegions, seedTotal),
		WOTH:              locationPctFromMap(wothLocations, seedTotal),
		WOTHItems:         locationPctFromMap(wothItems, seedTotal),
		ItemLocations:     locationPctFromMap(locationItems, seedTotal),
		JunkLocations:     locationPctFromMap(locationJunk, seedTotal),
		SmallKeyLocations: locationPctFromMap(locationSmallKeys, seedTotal),
		BossKeyLocations:  locationPctFromMap(locationBossKeys, seedTotal),
		IceTrapLocations:  locationPctFromMap(locationIceTraps, seedTotal),
	}, nil
}

func progressiveItemName(cache map[string]int, item string) string {
	cache[item]++
	switch item {
	case "Progressive Strength Upgrade":
		switch cache[item] {
		case 1:
			return "Goron's Bracelet"
		case 2:
			return "Silver Gauntlets"
		case 3:
			return "Golden Gauntlets"
		}
	case "Progressive Hookshot":
		switch cache[item] {
		case 1:
			return "Hookshot"
		case 2:
			return "Longshot"
		}
	case "Progressive Scale":
		switch cache[item] {
		case 1:
			return "Silver Scale"
		case 2:
			return "Golden Scale"
		}
	case "Progressive Wallet":
		switch cache[item] {
		case 1:
			return "Adult's Wallet"
		case 2:
			return "Giant's Wallet"
		}
	}

	return item
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

	ItemLocations, JunkLocations, IceTrapLocations []locationPct
	SmallKeyLocations, BossKeyLocations            []locationPct
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

type itemCategory int

const (
	itemCategoryItem = iota
	itemCategoryBossKey
	itemCategoryIceTrap
	itemCategoryJunk
	itemCategoryMedallion
	itemCategoryPoH
	itemCategorySmallKey
	itemCategorySong
)

func getItemCategory(item string) itemCategory {
	if strings.HasPrefix(item, "Small Key") {
		return itemCategorySmallKey
	}
	if strings.HasPrefix(item, "Boss Key") {
		return itemCategoryBossKey
	}
	if strings.HasSuffix(item, "Medallion") {
		return itemCategoryMedallion
	}

	switch item {
	case
		"Arrows (10)", "Arrows (30)", "Arrows (5)",
		"Bombs (10)", "Bombs (20)", "Bombs (5)",
		"Deku Nuts (10)", "Deku Nuts (5)",
		"Deku Seeds (30)", "Deku Stick (1)",
		"Recovery Heart",
		"Rupee (1)", "Rupees (5)", "Rupees (50)",
		"Rupees (20)", "Rupees (200)",
		// might deserve its own category
		"Bombchus (5)", "Bombchus (10)", "Bombchus (20)":
		return itemCategoryJunk

	case
		"Zeldas Lullaby", "Eponas Song", "Sarias Song",
		"Suns Song", "Song of Time", "Song of Storms",
		"Minuet of Forest", "Bolero of Fire", "Serenade of Water",
		"Nocturne of Shadow", "Requiem of Spirit", "Prelude of Light":
		return itemCategorySong

	case
		"Kokiri Emerald", "Goron Ruby", "Zora Sapphire":
		return itemCategoryMedallion
	case
		"Piece of Heart", "Piece of Heart (Treasure Chest Game)",
		"Heart Container", "Double Defense":
		return itemCategoryPoH
	case "Ice Trap":
		return itemCategoryIceTrap
	default:
		return itemCategoryItem
	}
}
