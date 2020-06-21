package back

// orderedRandomPairPlayers randomly pairs close players together.
// It takes a list of players sorted by their rank and matches two players
// close enough in the list until there is no player left.
// TODO: Heuristics, if both shared their last match: go one neighbor down/up.
func orderedRandomPairPlayers(players []Player) []pair {
	if len(players) < 2 {
		return nil
	}
	if len(players)%2 != 0 {
		panic("fed an odd number of players to pairPlayers")
	}

	pairs := make([]pair, 0, len(players)/2)
	maxDelta := getMaxDelta(len(players))

	for len(players) > 0 {
		i1 := randomIndex(len(players))
		p := pair{p1: players[i1]}
		players = players[:i1+copy(players[i1:], players[i1+1:])]

		minIndex := clamp(i1-maxDelta, 0, len(players)-1)
		maxIndex := clamp(i1+maxDelta, 0, len(players)-1)

		i2 := randomInt(minIndex, maxIndex)
		p.p2 = players[i2]
		players = players[:i2+copy(players[i2:], players[i2+1:])]

		pairs = append(pairs, p)
	}

	return pairs
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
