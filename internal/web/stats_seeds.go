package web

import (
	"encoding/json"
	"fmt"
	"io"
	"kaepora/internal/generator/oot"
	"log"
	"sort"
	"strings"
	"time"
)

type locationPct struct {
	Name                     string
	Items, Junk, IceTraps    float64
	SmallKeys, BossKeys, PoH float64
	Triforces, Songs, Chus   float64
}

func locationPctFromMap(
	m map[string]map[oot.SpoilerLogItemCategory]int,
	totalInt int,
) (ret []locationPct) {
	total := float64(totalInt)
	for name, v := range m {
		ret = append(ret, locationPct{
			Name:      name,
			Items:     100.0 * (float64(v[oot.SpoilerLogItemCategoryItem]) / total),
			Junk:      100.0 * (float64(v[oot.SpoilerLogItemCategoryJunk]) / total),
			IceTraps:  100.0 * (float64(v[oot.SpoilerLogItemCategoryIceTrap]) / total),
			SmallKeys: 100.0 * (float64(v[oot.SpoilerLogItemCategorySmallKey]) / total),
			BossKeys:  100.0 * (float64(v[oot.SpoilerLogItemCategoryBossKey]) / total),
			PoH:       100.0 * (float64(v[oot.SpoilerLogItemCategoryPoH]) / total),
			Chus:      100.0 * (float64(v[oot.SpoilerLogItemCategoryBombchu]) / total),
			Songs:     100.0 * (float64(v[oot.SpoilerLogItemCategorySong]) / total),
			Triforces: 100.0 * (float64(v[oot.SpoilerLogItemCategoryTriforce]) / total),
		})
	}

	sort.Sort(byName(ret))

	return ret
}

type byName []locationPct

func (a byName) Len() int {
	return len([]locationPct(a))
}

func (a byName) Less(i, j int) bool {
	return a[i].Name < a[j].Name
}

func (a byName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (s *Server) getSeedStats(shortcode string) (statsSeed, error) {
	start := time.Now()
	seedTotal := 0

	wothLocations := map[string]int{}
	wothItems := map[string]int{}
	barrenRegions := map[string]int{}
	settings := map[string]map[string]int{} // name => value => count
	locationsAcc := map[string]map[oot.SpoilerLogItemCategory]int{}
	sphereSum := map[string]int{}   // total sphere sum by item
	sphereCount := map[string]int{} // item occurrence count

	if err := s.back.MapSpoilerLogs(shortcode, func(raw io.Reader) error {
		seedTotal++

		var l oot.SpoilerLog
		dec := json.NewDecoder(raw)
		if err := dec.Decode(&l); err != nil {
			return err
		}

		for _, name := range l.BarrenRegions {
			barrenRegions[name]++
		}

		computeWOTHStats(l, wothLocations, wothItems)
		computeLocationStats(l, locationsAcc)
		computeSettingsStats(l, settings)
		computeSphereStats(l, sphereSum, sphereCount)

		return nil
	}); err != nil {
		return statsSeed{}, err
	}

	defer func() { log.Printf("info: computed stats for %d seeds in %s", seedTotal, time.Since(start)) }()
	return statsSeed{
		Barren:    namedPctFromMap(barrenRegions, seedTotal),
		WOTH:      namedPctFromMap(wothLocations, seedTotal),
		WOTHItems: namedPctFromMap(wothItems, seedTotal),
		Locations: locationPctFromMap(locationsAcc, seedTotal),
		Settings:  NamedPct2DFrom2DMap(settings, seedTotal),
		Spheres:   namedAvgFromSumAndCount(sphereSum, sphereCount),
	}, nil
}

func computeWOTHStats(l oot.SpoilerLog, wothLocations, wothItems map[string]int) {
	progressive := map[string]int{}
	for location, item := range l.WOTHLocations {
		wothLocations[string(location)]++

		if strings.HasPrefix(string(item), "Progressive") {
			wothItems[progressiveItemName(progressive, string(item))]++
		} else {
			wothItems[string(item)]++
		}
	}
}

func computeLocationStats(l oot.SpoilerLog, locationsAcc map[string]map[oot.SpoilerLogItemCategory]int) {
	for name, item := range l.Locations {
		if _, ok := locationsAcc[string(name)]; !ok {
			locationsAcc[string(name)] = make(
				map[oot.SpoilerLogItemCategory]int,
				oot.SpoilerLogItemCategoryCount,
			)
		}

		locationsAcc[string(name)][item.GetCategory()]++
	}
}

func computeSettingsStats(l oot.SpoilerLog, settings map[string]map[string]int) {
	for name, value := range l.Settings {
		// HACK, ugly array we don't care about.
		if name == "allowed_tricks" {
			continue
		}

		if _, ok := settings[name]; !ok {
			settings[name] = map[string]int{}
		}

		settings[name][fmt.Sprintf("%v", value)]++
	}
}

func computeSphereStats(l oot.SpoilerLog, sphereSum, sphereCount map[string]int) {
	progressive := map[string]int{}

	for k, sphere := range l.Spheres() {
		for _, item := range sphere {
			itemName := progressiveItemName(progressive, string(item))
			switch item.GetCategory() { // nolint:exhaustive
			case oot.SpoilerLogItemCategorySmallKey, oot.SpoilerLogItemCategoryBossKey:
				continue
			}

			sphereSum[itemName] += k
			sphereCount[itemName]++
		}
	}
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

func namedPctFromMap(m map[string]int, totalInt int) (ret []namedPct) {
	total := float64(totalInt)

	for k, v := range m {
		ret = append(ret, namedPct{
			Name: k,
			Pct:  100.0 * (float64(v) / total),
		})
	}

	sort.Sort(byPctDesc(ret))

	return ret
}

func namedAvgFromSumAndCount(sum, count map[string]int) (ret []namedPct) {
	for k, v := range sum {
		ret = append(ret, namedPct{
			Name: k,
			Pct:  (float64(v) / float64(count[k])),
		})
	}

	sort.Sort(byPctDesc(ret))

	return ret
}

func NamedPct2DFrom2DMap(m map[string]map[string]int, totalInt int) (ret []NamedPct2D) {
	total := float64(totalInt)

	for name := range m {
		for value, count := range m[name] {
			ret = append(ret, NamedPct2D{
				Name:  name,
				Value: value,
				Pct:   100.0 * (float64(count) / total),
			})
		}
	}

	sort.Sort(byPct2DDesc(ret))

	return ret
}

type statsSeed struct {
	WOTH, WOTHItems, Barren []namedPct
	Spheres                 []namedPct
	Locations               []locationPct
	Settings                []NamedPct2D
}

type namedPct struct {
	Name string
	Pct  float64
}

type NamedPct2D struct {
	Name  string
	Value string
	Pct   float64
}

type byPctDesc []namedPct

func (a byPctDesc) Len() int {
	return len([]namedPct(a))
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

type byPct2DDesc []NamedPct2D

func (a byPct2DDesc) Len() int {
	return len([]NamedPct2D(a))
}

func (a byPct2DDesc) Less(i, j int) bool {
	if a[i].Pct == a[j].Pct {
		return a[i].Name < a[j].Name
	}

	return a[i].Pct > a[j].Pct
}

func (a byPct2DDesc) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
