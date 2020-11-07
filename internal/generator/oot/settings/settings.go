package settings

import (
	"encoding/json"
	"hash/fnv"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
)

// Settings holds every setting we want to randomize, along with their possible
// values, cost, and a probability weight.
// The cost is an arbitrary cost out of a budget of an arbitrary number of
// points, the idea is to avoid having too much chaos-inducing settings applied
// at the same time.
// The probability is there to ensure some values are scarcely or never used.
// It is an integer that only has meaning relative to the sum of all
// probabilities.
type Settings map[string]Setting // name (json key) => possible values

// Load loads shuffled settings parameters from file.
func Load(path string) (Settings, error) {
	var ret Settings
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	if err := dec.Decode(&ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func (s Settings) weightSum() float64 {
	var weightSum float64
	for k := range s {
		for i := range s[k] {
			weightSum += s[k][i].Weight
		}
	}

	return weightSum
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func int64SeedFromString(str string) int64 {
	h := fnv.New64a()
	if _, err := h.Write([]byte(str)); err != nil {
		panic(err)
	}

	return int64(h.Sum64())
}

// Shuffle is a probably broken adaptation of M. T. Chao "general purpose
// unequal probability sampling plan" algorithm.
// Biometrika Vol. 69, No. 3 (Dec., 1982), pp. 653-656
// DOI: 10.2307/2336002
// There is an hardcoded maximum iterations count to avoid inifite loops, and
// and a tolerance for going under or over the cost budget if we reach enough
// iterations.
func (s Settings) Shuffle(seedStr string, costMax int) map[string]interface{} { // nolint:funlen
	r := rand.New(rand.NewSource(int64SeedFromString(seedStr))) // nolint:gosec

	var costSum, iterations, tolerance int
	weightSum := s.weightSum()
	maxIterations := 1000 // arbitrary
	ret := map[string]interface{}{}

	// Make map iteration deterministic
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// iterate until we matched our budget or we failed to match it
	for abs(costMax-costSum) > tolerance && iterations < maxIterations {
		// Shuffle before each step to ensure first items are not picked more than the rest.
		r.Shuffle(len(keys), func(i, j int) {
			keys[i], keys[j] = keys[j], keys[i]
		})

		for _, k := range keys { // iterate over all settings
			// Already decided on a value for this setting on a previous iteration.
			if _, ok := ret[k]; ok {
				continue
			}

			for i := range s[k] { // iterate over all possible values
				if s[k][i].Weight <= 0 {
					continue
				}

				p := s[k][i].Weight / weightSum
				if r.Float64() > p { // not selected, ignore
					continue
				}

				// Over cost budget, ignore
				newCost := costSum + s[k][i].Cost
				if newCost > (costMax - tolerance) {
					continue
				}

				// Selected, set value and update cost.
				costSum = newCost
				ret[k] = s[k][i].Value

				// Avoid conflicting settings
				for impliedKey, impliedValue := range s[k][i].Implies {
					ret[impliedKey] = impliedValue
				}

				// Disable other bridge settings if we picked one.
				handleBridgeSettings(k, s[k][i].Value, ret)

				break
			}
		}

		tolerance = iterations / 100
		iterations++
	}

	return ret
}

// handleBridgeSettings is a magic/auto Implies statement to disable all other
// bridge settings once one has been picked.
func handleBridgeSettings(name string, value interface{}, ret map[string]interface{}) {
	if !strings.HasPrefix(name, "bridge") {
		return
	}

	// If we manually set bridge, ensure we're not picking any other bridge setting.
	if name == "bridge" {
		vStr, ok := value.(string)
		if !ok {
			log.Printf("error: 'bridge' set with non-string value '%v'", value)
			return
		}
		if vStr != "open" && vStr != "vanilla" {
			log.Printf("error: 'bridge' can only be set to 'open' or 'vanilla', '%s' provided", vStr)
			return
		}
		ret["bridge_tokens"] = 0
		ret["bridge_stones"] = 0
		ret["bridge_rewards"] = 0
		ret["bridge_medallions"] = 0
		return
	}

	typ := strings.Split(name, "_")[1] // will panic on bad value, 'tis ok.
	if typ == "rewards" {
		ret["bridge"] = "dungeons" // OOTR and consistency…
	} else {
		ret["bridge"] = typ
	}

	// Set all other settings to 0 to ensure they cannot be picked later.
	for _, v := range []string{"stones", "medallions", "rewards", "tokens"} {
		if v != typ {
			ret["bridge_"+v] = 0
		}
	}
}

// A Setting is a collection of values that can be given to a setting key.
type Setting []PossibleSettingValue

// A PossibleSettingValue has a cost that represents its impact on routing, a
// weight to make it appear less or more often, and a list of implied settings
// k/v to avoid impossible settings combo or force interesting combinations.
type PossibleSettingValue struct {
	Value   interface{}
	Cost    int
	Weight  float64
	Implies map[string]interface{}
}
