package back

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"kaepora/internal/util"
	"log"
	"math/big"
	"sort"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func (b *Back) CancelActiveMatchSession(player Player) (MatchSession, error) {
	var ret MatchSession

	if err := b.transaction(func(tx *sqlx.Tx) error {
		session, err := getPlayerActiveSession(tx, player.ID.UUID())
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return util.ErrPublic("you are not in any active race right now")
			}
			return err
		}

		if err := session.CanCancel(); err != nil {
			return err
		}

		session.RemovePlayerID(player.ID.UUID())
		if err := session.update(tx); err != nil {
			return err
		}

		ret = session
		return nil
	}); err != nil {
		return MatchSession{}, err
	}

	return ret, nil
}

func (b *Back) ForfeitActiveMatch(player Player) (Match, error) {
	var match Match
	if err := b.transaction(func(tx *sqlx.Tx) error {
		match, self, against, err := getActiveMatchAndEntriesForPlayer(tx, player)
		if err != nil {
			return err
		}

		self.forfeit(&against, &match)

		e := []error{
			self.update(tx),
			against.update(tx),
			match.update(tx),
		}

		return util.ConcatErrors(e)
	}); err != nil {
		return Match{}, err
	}

	return match, nil
}

func (b *Back) CompleteActiveMatch(player Player) (Match, error) {
	var match Match
	if err := b.transaction(func(tx *sqlx.Tx) error {
		match, self, against, err := getActiveMatchAndEntriesForPlayer(tx, player)
		if err != nil {
			return err
		}

		self.complete(&against, &match)

		e := []error{
			self.update(tx),
			against.update(tx),
			match.update(tx),
		}

		return util.ConcatErrors(e)
	}); err != nil {
		return Match{}, err
	}

	return match, nil
}

func (b *Back) JoinCurrentMatchSession(
	player Player, league League,
) (MatchSession, error) {
	var ret MatchSession
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		ret, err = joinCurrentMatchSessionTx(tx, player, league)
		return err
	}); err != nil {
		return MatchSession{}, err
	}

	return ret, nil
}

func (b *Back) JoinCurrentMatchSessionByShortcode(player Player, shortcode string) (
	session MatchSession,
	league League,
	_ error,
) {
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		league, err = getLeagueByShortCode(tx, shortcode)
		if err != nil {
			return err
		}

		session, err = joinCurrentMatchSessionTx(tx, player, league)
		return err
	}); err != nil {
		return MatchSession{}, League{}, err
	}

	return session, league, nil
}

func joinCurrentMatchSessionTx(
	tx *sqlx.Tx, player Player, league League,
) (MatchSession, error) {
	session, err := getNextJoinableMatchSessionForLeague(tx, league.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MatchSession{},
				util.ErrPublic("could not find a joinable race for the given league")
		}
		return MatchSession{}, err
	}

	if session.HasPlayerID(player.ID.UUID()) {
		return MatchSession{}, util.ErrPublic(fmt.Sprintf(
			"you are already registered for the next %s race", league.Name,
		))
	}

	if active, err := getPlayerActiveSession(tx, player.ID.UUID()); err == nil {
		activeLeague, err := getLeagueByID(tx, active.LeagueID)
		if err != nil {
			return MatchSession{}, err
		}

		if active.ID != session.ID {
			return MatchSession{},
				util.ErrPublic(fmt.Sprintf(
					"you are already registered for another race on the %s league",
					activeLeague.Name,
				))
		}
	}

	session.AddPlayerID(player.ID.UUID())
	if err := session.update(tx); err != nil {
		return MatchSession{}, err
	}

	return session, nil
}

// doMatchMaking creates all Match and MatchEntry on Matches that reached the
// preparing state, and dispatches seeds to the players.
// This is done in a different transaction than makeMatchSessionsPreparing to
// ensure no one can join when we matchmake/generate the seeds.
func (b *Back) doMatchMaking(sessions []MatchSession) error {
	return b.transaction(func(tx *sqlx.Tx) error {
		for k := range sessions {
			if err := b.matchMakeSession(tx, sessions[k]); err != nil {
				return err
			}

			// TODO generate & send seeds
		}

		return nil
	})
}

type byRating []Player

func (a byRating) Len() int {
	return len(a)
}

func (a byRating) Less(i, j int) bool {
	return a[i].Rating.GlickoRating().R() < a[j].Rating.GlickoRating().R()
}

func (a byRating) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

type pair struct {
	p1, p2 Player
}

// I'm going to do things the sqlite way and JOIN nothing here, don't be afraid.
func (b *Back) matchMakeSession(tx *sqlx.Tx, session MatchSession) error {
	session, err := ensureSessionIsValidForMatchMaking(tx, session)
	if err != nil {
		return err
	}
	if session.Status != MatchSessionStatusInProgress {
		return nil
	}
	if err := b.sendSessionCountdownNotification(tx, session); err != nil {
		return err
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

// pairPlayers randomly pair close players together.
func pairPlayers(players []Player) []pair {
	// TODO: Heuristics, if both shared their last match: go one neighbor down/up
	pairs := make([]pair, 0, len(players)/2)
	for len(players) > 2 {
		i1 := randomIndex(len(players))
		p := pair{p1: players[i1]}
		players = players[:i1+copy(players[i1:], players[i1+1:])]

		minIndex := i1 - 5
		if minIndex < 0 {
			minIndex = 0
		}
		maxIndex := i1 + 5
		if maxIndex > len(players)-1 {
			maxIndex = len(players) - 1
		}
		if minIndex == maxIndex {
			panic("unreachable")
		}

		var i2 int
		for i2 == 0 {
			i2 = randomInt(minIndex, maxIndex)
		}
		p.p2 = players[i2]
		players = players[:i2+copy(players[i2:], players[i2+1:])]

		pairs = append(pairs, p)
	}
	pairs = append(pairs, pair{players[0], players[1]})

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

func ensureSessionIsValidForMatchMaking(tx *sqlx.Tx, session MatchSession) (MatchSession, error) {
	players := session.GetPlayerIDs()
	// No one wants to play =(
	if len(players) <= 1 {
		// TODO announce empty race
		session.Status = MatchSessionStatusClosed
		log.Printf("info: no players for session %s", session.ID.UUID())
		return session, session.update(tx)
	}

	// Ditch the one player we can't match with anyone.
	if len(players)%2 == 1 {
		toRemove := players[randomIndex(len(players))]
		log.Printf("info: removed odd player %s from session %s", toRemove, session.ID.UUID())
		session.RemovePlayerID(toRemove)
	}

	session.Status = MatchSessionStatusInProgress

	if err := session.update(tx); err != nil {
		return MatchSession{}, err
	}

	return session, nil
}

func randomIndex(length int) int {
	return randomInt(0, length-1)
}

func randomInt(iMin, iMax int) int {
	if iMin > iMax {
		panic("iMin > iMax")
	}

	max := big.NewInt(int64(iMax - iMin))
	offset, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(err)
	}

	return int(offset.Int64() - int64(iMin))
}

func getActiveMatchAndEntriesForPlayer(tx *sqlx.Tx, player Player) (
	match Match, self MatchEntry, opponent MatchEntry, _ error,
) {
	session, err := getPlayerActiveSession(tx, player.ID.UUID())
	if err != nil {
		if err == sql.ErrNoRows {
			return match, self, opponent, util.ErrPublic("you are not in any active race right now")
		}
		return match, self, opponent, fmt.Errorf("unable to get active session: %w", err)
	}

	if err := session.CanForfeit(); err != nil {
		return match, self, opponent, err
	}

	match, err = getMatchByPlayerAndSession(tx, player, session)
	if err != nil {
		return match, self, opponent, fmt.Errorf("cannot find Match: %w", err)
	}

	self, opponent, err = match.getPlayerAndOpponentEntries(player.ID)
	if err != nil {
		return match, self, opponent, err
	}

	return match, self, opponent, nil
}
