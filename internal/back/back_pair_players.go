package back

import (
	"log"
	"sort"
)

// orderedRandomPairPlayers randomly pairs close players together.
// It takes a list of players and matches two players close enough in the list
// until there is no player left.
// TODO: Heuristics, if both shared their last match: go one neighbor down/up.
func orderedRandomPairPlayers(players []Player) []pair {
	if len(players) < 2 {
		return nil
	}
	if len(players)%2 != 0 {
		panic("fed an odd number of players to orderedRandomPairPlayers")
	}

	sort.Sort(byRating(players))
	pairs := make([]pair, 0, len(players)/2)
	maxDelta := getMaxDelta(len(players))

	for len(players) > 0 {
		i1 := randomIndex(len(players))
		p := pair{p1: players[i1]}
		players = removePlayer(players, i1)

		minIndex := clamp(i1-maxDelta, 0, len(players)-1)
		maxIndex := clamp(i1+maxDelta, 0, len(players)-1)

		i2 := randomInt(minIndex, maxIndex)
		p.p2 = players[i2]
		players = removePlayer(players, i2)

		pairs = append(pairs, p)
	}

	return pairs
}

func removePlayer(players []Player, i int) []Player {
	return players[:i+copy(players[i:], players[i+1:])]
}

// getMaxDelta calculates the max distance between opponents for a given
// session size. This reduces the probability of smaller sessions producing
// completely arbitrary pairings.
func getMaxDelta(sessionSize int) int {
	switch {
	case sessionSize <= 8:
		return 1
	case sessionSize <= 16:
		return 2
	default:
		return 3
	}
}

// sensible pairings based on skill range (R±(2*RD)).
func rangedPairPlayers(players []Player) []pair {
	if len(players) < 2 {
		return nil
	}
	if len(players)%2 != 0 {
		panic("fed an odd number of players to rangedPairPlayers")
	}

	ranges := getOverlappingRanges(players)
	matched := make(map[int]struct{}, len(players))
	pairs := make([]pair, 0, len(players)/2)

	// We iterate over the generated overlapping ranges, as the players with
	// the lowest available matches are picked first, we lower the probability
	// of ending up with no proper matches.
	for rangeID := range ranges {
		playerIndex := ranges[rangeID].playerIndex
		if _, ok := matched[playerIndex]; ok {
			continue
		}

		// Get available (not yet matched) players
		available := make([]rangeEntry, 0, len(ranges[rangeID].entries))
		for i := range ranges[rangeID].entries {
			candidateIndex := ranges[rangeID].entries[i].playerIndex
			if _, ok := matched[candidateIndex]; !ok {
				available = append(available, ranges[rangeID].entries[i])
			}
		}

		// There is no matchable players, bail on this player.
		if len(available) == 0 {
			continue
		}

		// Pick one available player at random, preferably from the top of the
		// list (closer matchup as it's ordered by rating distance)
		pick := randomLowIndex(len(available))
		pairs = append(pairs, pair{
			p1: players[playerIndex],
			p2: players[available[pick].playerIndex],
		})

		// Mark as matched
		matched[playerIndex] = struct{}{}
		matched[available[pick].playerIndex] = struct{}{}
	}

	// Match the remaining players, these will be pretty much random.
	unmatched := make([]Player, 0, len(players)-len(matched))
	for playerIndex := range players {
		if _, ok := matched[playerIndex]; !ok {
			unmatched = append(unmatched, players[playerIndex])
		}
	}
	if len(unmatched) > 0 {
		log.Printf("debug: got %d unmatched players via range, using ordered random", len(unmatched))
		pairs = append(pairs, orderedRandomPairPlayers(unmatched)...)
	}

	return pairs
}

// getOverlappingRanges returns the list of indices of acceptable matchings
// along with the distance between the ratings of the two players. The result
// is sorted by available matches and the matches are sorted by rating
// distance.
func getOverlappingRanges(players []Player) []playerRange {
	// player index in players slice => [player index in players slice,]
	// filling this is O(n²), call me when we have > 1k players.
	ret := make([]playerRange, len(players))
	for i := range players {
		min0, max0 := players[i].Rating.Range()
		for j := range players {
			if i == j {
				continue
			}

			min1, max1 := players[j].Rating.Range()
			if max0 < min1 || min0 > max1 {
				continue
			}

			dist := players[i].Rating.Rating - players[j].Rating.Rating
			if dist < 0 {
				dist = -dist
			}

			ret[i].playerIndex = i
			ret[i].entries = append(ret[i].entries, rangeEntry{
				playerIndex: j,
				dist:        int(dist),
			})
		}

		sort.Sort(byEntryRating(ret[i].entries))
	}

	sort.Sort(byLen(ret))
	return ret
}

type playerRange struct {
	playerIndex int
	entries     []rangeEntry
}

type rangeEntry struct {
	playerIndex int
	dist        int // distance between ratings
}

type byLen []playerRange

func (a byLen) Len() int {
	return len(a)
}

func (a byLen) Less(i, j int) bool {
	return len(a[i].entries) < len(a[j].entries)
}

func (a byLen) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

type byEntryRating []rangeEntry

func (a byEntryRating) Len() int {
	return len(a)
}

func (a byEntryRating) Less(i, j int) bool {
	return a[i].dist < a[j].dist
}

func (a byEntryRating) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// randomLowIndex returns a random index with a higher chance of having a value
// close to 0.
func randomLowIndex(ln int) int {
	a, b := randomIndex(ln), randomIndex(ln)
	if a < b {
		return a
	}

	return b
}
