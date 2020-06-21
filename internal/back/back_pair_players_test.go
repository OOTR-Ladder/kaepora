package back

import (
	"fmt"
	"kaepora/internal/util"
	"strconv"
	"strings"
	"testing"
)

func TestPairPlayersDistance(t *testing.T) {
	players := make([]Player, 256)
	for k := range players {
		players[k] = NewPlayer("player#" + strconv.Itoa(k))
		players[k].ID = util.UUIDAsBlob{}
		players[k].ID[0] = byte(k)
	}
	if len(players) == 0 {
		t.Fatal("empty players")
	}

	pairs := orderedRandomPairPlayers(players)
	if len(pairs) == 0 {
		t.Fatal("empty pairs")
	}
	if len(pairs) != len(players)/2 {
		t.Errorf("expected %d pairs, got %d", len(players)/2, len(pairs))
	}

	distrib := make(map[int]int)
	for _, v := range pairs {
		delta := int(v.p1.ID[0]) - int(v.p2.ID[0])
		if delta < 0 {
			delta = -delta
		}
		distrib[delta]++
	}

	fmt.Println("index distance distribution:")
	for dist := 1; dist <= 32; dist++ {
		fmt.Printf("%-3d %s\n", dist, strings.Repeat("*", distrib[dist]))
	}
}
