package back

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"kaepora/internal/util"
	"log"
	"math/big"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// doMatchMaking creates all Match and MatchEntry on Matches that reached the
// preparing state, and dispatches seeds to the players.
func (b *Back) doMatchMaking(sessions []MatchSession) error {
	return b.transaction(func(tx *sqlx.Tx) error {
		for k := range sessions {
			if err := b.matchMakeSession(tx, sessions[k]); err != nil {
				return err
			}

			if err := b.generateAndSendSeeds(tx, sessions[k]); err != nil {
				return err
			}
		}

		return nil
	})
}

// generateAndSendSeeds creates the seeds for all matches in a given session.
func (b *Back) generateAndSendSeeds(tx *sqlx.Tx, session MatchSession) error {
	matches, err := getMatchesBySessionID(tx, session.ID)
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		return errors.New("attempted to generate seeds for 0 matches")
	}

	players := make(map[util.UUIDAsBlob]Player, len(matches)*2)
	for k := range matches {
		for i := 0; i <= 1; i++ {
			p, err := getPlayerByID(tx, matches[k].Entries[i].PlayerID)
			if err != nil {
				return err
			}
			players[p.ID] = p
		}
	}

	// Run in a routine to release the transaction early and write the entries.
	go b.doParallelSeedGeneration(session, matches, players)

	return nil
}

// doParallelSeedGeneration generates the seeds for the given matches of the
// given MatchSession in parallel using one worker per available CPU core.
// players must be a prefetched Player.ID->Player map for the given matches.
func (b *Back) doParallelSeedGeneration(
	session MatchSession,
	matches []Match,
	players map[util.UUIDAsBlob]Player,
) {
	type pl struct {
		match   Match
		session MatchSession
		p1, p2  Player
	}

	start := time.Now()
	worker := func(ch <-chan pl, wg *sync.WaitGroup) {
		defer wg.Done()
		for v := range ch {
			curSeedStart := time.Now()
			log.Printf("debug: generating seed %s for match %s", v.match.Seed, v.match.ID)
			if err := b.generateAndSendMatchSeed(v.match, v.session, v.p1, v.p2); err != nil {
				log.Printf("unable to generate and send seed: %s", err)
			}
			log.Printf("info: generated seed %s in %s (%s)", v.match.Seed, time.Since(curSeedStart), time.Since(start))
		}
	}

	cpus := runtime.NumCPU()
	// HACK, arbitrary rate limit for external services, let the API client do
	// the rate-limiting.
	if gen, err := b.generatorFactory.NewGenerator(matches[0].Generator); err == nil {
		if gen.IsExternal() {
			cpus = 10
			log.Printf("debug: external seedgen, setting limit to %d", cpus)
		}
	}

	pool := make(chan pl, cpus)
	log.Printf("debug: limiting seedgen to %d at a time", cpus)

	var wg sync.WaitGroup
	for i := 0; i < cpus; i++ {
		wg.Add(1)
		go worker(pool, &wg)
	}

	for k := range matches {
		pool <- pl{
			match:   matches[k],
			session: session,
			p1:      players[matches[k].Entries[0].PlayerID],
			p2:      players[matches[k].Entries[1].PlayerID],
		}
	}
	close(pool)

	wg.Wait()
	log.Printf("info: generated %d seeds in %s", len(matches), time.Since(start))
}

// generateAndSendMatchSeed synchronously generates the seed and then sends the
// binary patch to the players via a notification.
func (b *Back) generateAndSendMatchSeed(
	match Match,
	session MatchSession,
	p1, p2 Player,
) error {
	gen, err := b.generatorFactory.NewGenerator(match.Generator)
	if err != nil {
		return err
	}

	out, err := gen.Generate(match.Settings, match.Seed)
	if err != nil {
		return err
	}

	match.SpoilerLog, err = util.NewZLIBBlob(out.SpoilerLog)
	if err != nil {
		return err
	}

	match.SeedPatch = out.SeedPatch
	match.GeneratorState = out.State

	if err := b.transaction(match.update); err != nil {
		return err
	}

	b.sendMatchSeedNotification(
		session,
		gen.GetDownloadURL(out.State),
		out.SeedPatch, hashFromSpoilerLog(out.SpoilerLog),
		p1, p2,
	)

	return nil
}

// hashFromSpoilerLog extracts the "seed hash" from a OoT-Randomizer spoiler log.
// A seed hash is a short list of items that ±uniquely identifies a generated
// patch and can be verified in-game.
func hashFromSpoilerLog(spoilerLog []byte) string {
	spoil := struct {
		Hash []string `json:"file_hash"`
	}{}

	if err := json.Unmarshal(spoilerLog, &spoil); err != nil {
		log.Println("error: unable to extract hash from spoiler log")
		return ""
	}

	return strings.Join(spoil.Hash, ", ")
}

type pair struct {
	p1, p2 Player
}

// matchMakeSession takes a session, pairs registered players, and creates the
// resulting matches. The actual MM algorithm is in the pairPlayers function.
func (b *Back) matchMakeSession(tx *sqlx.Tx, session MatchSession) error {
	if session.Status != MatchSessionStatusPreparing {
		log.Printf("warning: attempted to matchmake session %s at status %d", session.ID, session.Status)
		return nil
	}

	session, ok, err := b.ensureSessionIsValidForMatchMaking(tx, session)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	players, err := getSessionPlayersSortedByRating(tx, session)
	if err != nil {
		return err
	}

	pairs := pairPlayers(players)
	log.Printf("debug: got %d players in the pool (%d pairs)", len(players), len(pairs))

	for k := range pairs {
		// google/uuid.v4 are generated using a CSPRNG
		match, err := NewMatch(tx, session, uuid.New().String())
		if err != nil {
			return err
		}
		if err := match.insert(tx); err != nil {
			return err
		}

		e1 := NewMatchEntry(match.ID, pairs[k].p1.ID)
		if err := e1.insert(tx); err != nil {
			return err
		}
		e2 := NewMatchEntry(match.ID, pairs[k].p2.ID)
		if err := e2.insert(tx); err != nil {
			return err
		}
	}

	return nil
}

func clamp(v, min, max int) int {
	if v > max {
		return max
	}

	if v < min {
		return min
	}

	return v
}

// pairPlayers randomly pairs close players together.
// It takes list of players sorted by their tank and matches two players close
// enough in the list until there is no player left.
// TODO: Heuristics, if both shared their last match: go one neighbor down/up.
func pairPlayers(players []Player) []pair {
	if len(players) < 2 {
		return nil
	}
	if len(players)%2 != 0 {
		panic("fed an odd number of players to pairPlayers")
	}

	pairs := make([]pair, 0, len(players)/2)
	maxDelta := 3

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

func getSessionPlayersSortedByRating(tx *sqlx.Tx, session MatchSession) ([]Player, error) {
	ids := session.GetPlayerIDs()
	players := make([]Player, 0, len(ids))

	for _, playerID := range ids {
		player, err := getPlayerByID(tx, util.UUIDAsBlob(playerID))
		if err != nil {
			return nil, err
		}
		player.Rating, err = getPlayerRating(tx, player.ID, session.LeagueID)
		if err != nil {
			return nil, err
		}

		players = append(players, player)
	}

	sort.Sort(byRating(players))
	return players, nil
}

// ensureSessionIsValidForMatchMaking ensures a MatchSession is in the required
// state for MM to occur and returns true if the MM can proceed.
// If there is an odd number of players, the last to join will be kicked.
func (b *Back) ensureSessionIsValidForMatchMaking(tx *sqlx.Tx, session MatchSession) (MatchSession, bool, error) {
	players := session.GetPlayerIDs()

	// No players / closed session
	if len(players) < 2 || session.Status != MatchSessionStatusPreparing {
		log.Printf("debug: not enough players or closed session %s", session.ID)
		return MatchSession{}, false, nil
	}

	// Ditch the one player we can't match with anyone.
	// The last player to join gets removed per community request.
	// (They did not like the idea of joining early and be kicked randomly 45
	// minutes later, can't fathom why.)
	if len(players)%2 == 1 {
		toRemove := players[len(players)-1]
		session.RemovePlayerID(toRemove)
		player, err := getPlayerByID(tx, util.UUIDAsBlob(toRemove))
		if err != nil {
			log.Printf("info: removed odd player %s (%s) from session %s", player.ID, player.Name, session.ID.UUID())
			return MatchSession{}, false, fmt.Errorf("unable to fetch odd player: %w", err)
		}
		b.sendOddKickNotification(player)
	}

	if err := session.update(tx); err != nil {
		return MatchSession{}, false, err
	}

	return session, true, nil
}

func randomIndex(length int) int {
	if length == 0 {
		panic("calling randomIndex with a length of zero")
	}

	return randomInt(0, length-1)
}

func randomInt(iMin, iMax int) int {
	if iMin > iMax {
		panic("iMin > iMax")
	}

	max := big.NewInt(int64(iMax - iMin + 1))
	offset, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(err)
	}

	return int(offset.Int64() + int64(iMin))
}

// getActiveMatchAndEntriesForPlayer returns the Match and the two
// corresponding MatchEntry for the Match the given player is currently running
// (ie. session is neither closed nor waiting).
func getActiveMatchAndEntriesForPlayer(tx *sqlx.Tx, player Player) (
	match Match, self MatchEntry, opponent MatchEntry, _ error,
) {
	session, err := getPlayerActiveSession(tx, player.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return match, self, opponent, util.ErrPublic("you are not in any active race right now")
		}
		return match, self, opponent, fmt.Errorf("unable to get active session: %w", err)
	}

	if err := session.CanForfeit(); err != nil {
		return match, self, opponent, err
	}

	match, err = getMatchByPlayerAndSession(tx, player.ID, session.ID)
	if err != nil {
		return match, self, opponent, fmt.Errorf("cannot find Match: %w", err)
	}

	self, opponent, err = match.getPlayerAndOpponentEntries(player.ID)
	if err != nil {
		return match, self, opponent, err
	}

	return match, self, opponent, nil
}
