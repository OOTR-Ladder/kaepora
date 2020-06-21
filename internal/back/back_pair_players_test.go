package back // nolint:testpackage

import (
	"fmt"
	"io/ioutil"
	"kaepora/internal/util"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	glicko "github.com/zelenin/go-glicko2"
)

type pairFun func([]Player) []pair

// createRandomPlayerDistribution outputs a list of randomly generated players
// with semi-realistic ratings, sorted by rating, with the first byte of their
// ID as their index in the list.
func createRandomPlayerDistribution() []Player {
	players := make([]Player, 8)
	for k := range players {
		players[k] = NewPlayer("player#" + strconv.Itoa(k))
		players[k].ID = util.UUIDAsBlob(uuid.New())
		players[k].Rating = PlayerRating{
			Rating:     glicko.RATING_BASE_R + float64(randomInt(-500, 500)),
			Deviation:  glicko.RATING_BASE_RD + float64(randomInt(-300, 0)),
			Volatility: glicko.RATING_BASE_SIGMA,
		}
	}

	sort.Sort(byRating(players))
	for k := range players {
		players[k].ID[0] = byte(k & 0xFF)
		players[k].ID[1] = byte(k & 0xFF00)
		players[k].Rating.PlayerID = players[k].ID
	}

	return players
}

func doMM(t *testing.T, fn pairFun) []pair {
	t.Helper()

	players := createRandomPlayerDistribution()
	if len(players) == 0 {
		t.Fatal("empty players")
	}
	pairs := fn(players)
	if len(pairs) == 0 {
		t.Fatal("empty pairs")
	}
	if len(pairs) != len(players)/2 {
		t.Errorf("expected %d pairs, got %d", len(players)/2, len(pairs))
	}

	return pairs
}

func displayRatingDistanceDistribution(t *testing.T, fn pairFun) {
	t.Helper()

	distrib := make(map[int]int, 20)
	for repeat := 0; repeat < 1000; repeat++ {
		pairs := doMM(t, fn)
		for _, v := range pairs {
			dist := v.p1.Rating.Rating - v.p2.Rating.Rating
			if dist < 0 {
				dist = -dist
			}

			dist = math.Round(dist/50) * 50
			distrib[int(dist)]++
		}
	}

	var total, max int
	keys := make([]int, 0, len(distrib))
	for k := range distrib {
		keys = append(keys, k)
		total += distrib[k]
		if distrib[k] > max {
			max = distrib[k]
		}
	}
	sort.Ints(keys)

	fmt.Println("rating distances distribution")
	for _, k := range keys {
		v := float64(distrib[k]) / float64(total)
		vMax := (float64(max) / float64(total))
		width := int((v * 120.0) / vMax) // 120 chars output max

		fmt.Printf("%4d %s\n", k, strings.Repeat("*", width))
	}
}

func TestPairPlayersDistance(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	fmt.Println("\norderedRandomPairPlayers")
	displayRatingDistanceDistribution(t, orderedRandomPairPlayers)

	fmt.Println("\nrangedPairPlayers")
	displayRatingDistanceDistribution(t, rangedPairPlayers)
}
