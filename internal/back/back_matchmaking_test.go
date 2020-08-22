package back // nolint:testpackage

import (
	"errors"
	"fmt"
	"io/ioutil"
	"kaepora/internal/config"
	"kaepora/internal/util"
	"log"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/guregu/null.v4"
)

func TestMatchMaking(t *testing.T) {
	back := createFixturedTestBack(t)

	innerTestMatchMaking(t, back)
	innerTestMatchMaking(t, back)
	innerTestMatchMaking(t, back)
}

// nolint:funlen
func innerTestMatchMaking(t *testing.T, back *Back) {
	notifs := make(map[NotificationType]int)

	// Consume notifications to avoid filling and blocking the chan
	notifsDone := make(chan struct{})
	go func(c <-chan Notification, done chan struct{}) {
		for {
			select {
			case notif := <-c:
				log.Printf("test: got notification: %s", notif.String())
				notifs[notif.Type]++
			case <-done:
				return
			}
		}
	}(back.GetNotificationsChan(), notifsDone)

	session, err := createSessionAndJoin(back)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkSessionStatus(back, session.ID, MatchSessionStatusJoinable); err != nil {
		t.Error(err)
	}
	sessions, err := prepareSession(back, session)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkSessionStatus(back, session.ID, MatchSessionStatusPreparing); err != nil {
		t.Error(err)
	}

	if err := back.doMatchMaking(sessions); err != nil {
		t.Fatal(err)
	}

	time.Sleep(200 * time.Millisecond) // HACK: wait for fake seed generation

	// Drops after being able to cancel: forfeit and loss
	if err := haveZeldaForfeit(back); err != nil {
		t.Error(err)
	}

	if err := fakeSessionStart(back, sessions[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := back.instantlyStartMatchSessions(); err != nil {
		t.Fatal(err)
	}
	if err := checkSessionStatus(back, session.ID, MatchSessionStatusInProgress); err != nil {
		t.Error(err)
	}

	if err := makeEveryoneComplete(back); err != nil {
		t.Error(err)
	}

	if err := back.runPeriodicTasks(); err != nil {
		t.Fatal(err)
	}
	if err := checkSessionStatus(back, session.ID, MatchSessionStatusClosed); err != nil {
		t.Error(err)
	}

	time.Sleep(50 * time.Millisecond) // HACK: wait for async recap notif

	expected := map[NotificationType]int{
		NotificationTypeMatchSessionStatusUpdate: 3, // /* TODO "created" when using schedule */, joinable, preparing, closed.
		NotificationTypeMatchSessionOddKick:      1, // that one unlucky runner
		NotificationTypeMatchSessionRecap:        7, // 1 in announce channel + 1 per joined player
		// NotificationTypeLeagueLeaderboardUpdate:  1, // not enough races to warrant a leaderboard update
		NotificationTypeMatchSeed:  6,     // 1 per joined player
		NotificationTypeMatchEnd:   6 + 3, // 1 per joined player, plus one for those who finish first
		NotificationTypeSpoilerLog: 6,     // 1 per joined player
	}
	close(notifsDone)
	if !reflect.DeepEqual(expected, notifs) {
		t.Errorf("notifications count does not match\nexpected: %#v\nactual  : %#v", expected, notifs)
	}
}

func checkSessionStatus(back *Back, sessionID util.UUIDAsBlob, status MatchSessionStatus) error {
	return back.transaction(func(tx *sqlx.Tx) error {
		session, err := getMatchSessionByID(tx, sessionID)
		if err != nil {
			return err
		}

		if session.Status != status {
			return fmt.Errorf("MatchSession %s expected status %d got %d", sessionID, status, session.Status)
		}

		return nil
	})
}

func haveRauruCancel(back *Back) error {
	var player Player
	if err := back.transaction(func(tx *sqlx.Tx) (err error) {
		player, err = getPlayerByName(tx, "Rauru")
		return err
	}); err != nil {
		return err
	}
	if _, err := back.ForfeitActiveMatch(player); err == nil {
		return errors.New("expected an error when forfeiting a cancellable MatchSession")
	}
	if _, err := back.CancelActiveMatchSession(player); err != nil {
		return err
	}

	return nil
}

func fakeSessionStart(back *Back, sessionID util.UUIDAsBlob) error {
	return back.transaction(func(tx *sqlx.Tx) (err error) {
		session, err := getMatchSessionByID(tx, sessionID)
		if err != nil {
			return fmt.Errorf("unable to get session: %s", err)
		}

		if session.Status != MatchSessionStatusPreparing {
			return fmt.Errorf("session %s should be preparing", sessionID)
		}

		// Fake match start
		session.StartDate = util.TimeAsDateTimeTZ(time.Now().Add(-time.Minute))
		if err := session.update(tx); err != nil {
			return fmt.Errorf("unable to update session: %s", err)
		}

		return nil
	})
}

func makeEveryoneComplete(back *Back) error {
	var players []Player
	if err := back.transaction(func(tx *sqlx.Tx) (err error) {
		var ids []util.UUIDAsBlob
		query := `SELECT PlayerID FROM MatchEntry WHERE Status = ?`
		if err := tx.Select(&ids, query, MatchEntryStatusInProgress); err != nil {
			return fmt.Errorf("unable to fetch player ids from MatchEntry: %s", err)
		}
		if len(ids) == 0 {
			return errors.New("no ids found")
		}

		for _, id := range ids {
			player, err := getPlayerByID(tx, id)
			if err != nil {
				return fmt.Errorf("unable to fetch player: %s", err)
			}
			players = append(players, player)
		}

		return nil
	}); err != nil {
		return err
	}

	for _, player := range players {
		log.Printf("test: completing %s", player.Name)

		if _, err := back.CompleteActiveMatch(player); err != nil {
			return fmt.Errorf("unable to complete match: %s", err)
		}
	}

	return nil
}

func haveZeldaForfeit(back *Back) error {
	var player Player
	if err := back.transaction(func(tx *sqlx.Tx) (err error) {
		player, err = getPlayerByName(tx, "Zelda")
		return err
	}); err != nil {
		return fmt.Errorf("cannot get player: %s", err)
	}

	if _, err := back.CancelActiveMatchSession(player); err == nil {
		return errors.New("expected an error when cancelling after MatchSessionCancellableUntilOffset")
	}

	log.Printf("test: forfeiting %s", player.Name)
	match, err := back.ForfeitActiveMatch(player)
	if err != nil {
		return fmt.Errorf("can't forfeit: %s", err)
	}
	if match.HasEnded() {
		return errors.New("match should not have ended")
	}

	var opponent Player
	if err := back.transaction(func(tx *sqlx.Tx) (err error) {
		_, against, err := match.getPlayerAndOpponentEntries(player.ID)
		if err != nil {
			return fmt.Errorf("cannot get MatchEntry: %s", err)
		}

		opponent, err = getPlayerByID(tx, against.PlayerID)
		return err
	}); err != nil {
		return fmt.Errorf("cannot get opponent: %s", err)
	}

	match, err = back.CompleteActiveMatch(opponent)
	if err == nil {
		return errors.New("match has not started, opponent should not have been able to complete")
	}

	if match.HasEnded() {
		return errors.New("the match should not have ended")
	}

	return nil
}

func haveZFGBeLate(back *Back, session MatchSession) error {
	var (
		player Player
		league League
	)

	if err := back.transaction(func(tx *sqlx.Tx) (err error) {
		player, err = getPlayerByName(tx, "Our Lord and Savior ZFG")
		if err != nil {
			return err
		}

		league, err = getLeagueByID(tx, session.LeagueID)
		return err
	}); err != nil {
		return err
	}

	if _, err := back.JoinCurrentMatchSession(player, league); err == nil {
		return errors.New("expected an error when joining MatchSessionStatusPreparing")
	}

	return nil
}

func prepareSession(back *Back, session MatchSession) ([]MatchSession, error) {
	var ret []MatchSession

	// He drops _before_ the race. No loss.
	if err := haveRauruCancel(back); err != nil {
		return nil, fmt.Errorf("could not cancel: %s", err)
	}

	// Fake being in the "preparing" time slice
	session.StartDate = util.TimeAsDateTimeTZ(time.Now().Add(-MatchSessionPreparationOffset))
	if err := back.transaction(session.update); err != nil {
		return nil, err
	}

	var err error
	ret, err = back.makeMatchSessionsPreparing()
	if err != nil {
		return nil, fmt.Errorf("unable to put MatchSession in preparing state: %w", err)
	}

	// Too slow buddy, should have joined before.
	if err := haveZFGBeLate(back, session); err != nil {
		return nil, fmt.Errorf("could not fail to join: %s", err)
	}

	return ret, nil
}

func createSessionAndJoin(back *Back) (MatchSession, error) {
	session, league, err := createJoinableSession(back)
	if err != nil {
		return MatchSession{}, fmt.Errorf("unable to create joinable session: %s", err)
	}

	if err := back.transaction(func(tx *sqlx.Tx) (err error) {
		getPlayer := func(name string) Player {
			player, err := getPlayerByName(tx, name)
			if err != nil {
				panic(err)
			}
			return player
		}

		players := []Player{
			getPlayer("Darunia"), getPlayer("Nabooru"), getPlayer("Rauru"),
			getPlayer("Ruto"), getPlayer("Saria"), getPlayer("Zelda"),
			getPlayer("Impa"), // last player, odd, will be kicked
		}
		for _, v := range players {
			found, err := joinCurrentMatchSessionTx(tx, v, league)
			if err != nil {
				return err
			} else if found.ID != session.ID {
				return errors.New("got the wrong session")
			}
		}

		session, err = getMatchSessionByID(tx, session.ID)
		if err != nil {
			return err
		}

		if len(session.GetPlayerIDs()) != 7 {
			return errors.New("expected 7 players in session")
		}

		return nil
	}); err != nil {
		return MatchSession{}, err
	}

	return session, nil
}

func createJoinableSession(back *Back) (MatchSession, League, error) {
	var (
		league  League
		session MatchSession
	)

	if err := back.transaction(func(tx *sqlx.Tx) (err error) {
		league, err = getLeagueByShortCode(tx, "testa")
		if err != nil {
			return fmt.Errorf("failed to fetch League: %w", err)
		}

		// TODO: use the schedule to create the session
		session = NewMatchSession(league.ID, time.Now().Add(-MatchSessionJoinableAfterOffset))
		if err := session.insert(tx); err != nil {
			return fmt.Errorf("failed to insert MatchSession: %w", err)
		}

		return nil
	}); err != nil {
		return MatchSession{}, League{}, err
	}

	if err := back.makeMatchSessionsJoinable(); err != nil {
		return MatchSession{}, League{}, fmt.Errorf("unable to make MatchSession joinable: %w", err)
	}

	if err := back.transaction(func(tx *sqlx.Tx) (err error) {
		session, err = getMatchSessionByID(tx, session.ID)
		if err != nil {
			return fmt.Errorf("could not fetch MatchSession back: %w", err)
		}

		if session.Status != MatchSessionStatusJoinable {
			return errors.New("expected Status == MatchSessionStatusJoinable")
		}
		return nil
	}); err != nil {
		return MatchSession{}, League{}, err
	}

	return session, league, nil
}

func createFixturedTestBack(t *testing.T) *Back {
	f, err := ioutil.TempFile("", "*.db")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()
	t.Cleanup(func() {
		os.Remove(path)
	})

	migrator, err := migrate.New(
		"file://../../resources/migrations",
		"sqlite3://"+path,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrator.Up(); err != nil {
		t.Fatal(err)
	}
	migrator.Close()

	back, err := New("sqlite3", path, &config.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := back.transaction(fixtures); err != nil {
		t.Fatal(err)
	}

	return back
}

func fixtures(tx *sqlx.Tx) error {
	game := NewGame("The Test Game")
	leagues := []League{
		NewLeague("The A League", "testa", game.ID, "test:v0", "s3.json"),
		NewLeague("The B League", "testb", game.ID, "test:v0", "s3.json"),
	}
	playerNames := []string{
		"Darunia", "Nabooru", "Rauru", "Ruto", "Saria", "Zelda", "Impa",
		"Our Lord and Savior ZFG",
	}

	if err := game.insert(tx); err != nil {
		return err
	}

	for k, v := range leagues {
		v.ID[0] = byte(k)
		v.AnnounceDiscordChannelID = util.NullString(fmt.Sprintf("league#%d", k))
		if err := v.insert(tx); err != nil {
			return err
		}
	}

	for k, v := range playerNames {
		player := NewPlayer(v)
		player.ID[0] = byte(k)
		player.DiscordID = null.NewString(fmt.Sprintf("player#%d %s", k, v), true)
		if err := player.insert(tx); err != nil {
			return err
		}
	}

	return nil
}

func TestRandomInt(t *testing.T) {
	ranges := [][]int{
		{-10, 10},
		{0, 10},
		{-1, 1},
		{-20, -10},
		{0, 1},
		{0, 0},
	}

	for _, r := range ranges {
		min, max := r[0], r[1]
		distrib := make(map[int]int, max-min)

		for i := 0; i < 1000; i++ {
			v := randomInt(min, max)
			distrib[v]++
			if v < min {
				t.Fatalf("%d < min", v)
			}
			if v > max {
				t.Fatalf("%d > max", v)
			}
		}

		for i := min; i <= max; i++ {
			if distrib[i] <= 0 {
				t.Errorf("it is _highly_ improbable not to have %d as a random value", i)
			}
		}
	}
}

func TestRandomIndex(t *testing.T) {
	a := make([]struct{}, 10)
	min := 0
	max := len(a) - 1
	distrib := make(map[int]int, max-min)

	for i := 0; i < 1000; i++ {
		v := randomIndex(len(a))
		distrib[v]++
		if v < min {
			t.Fatalf("%d < min", v)
		}
		if v > max {
			t.Fatalf("%d > max", v)
		}
	}

	for i := min; i <= max; i++ {
		if distrib[i] <= 0 {
			t.Errorf("it is _highly_ improbable not to have %d as a random value", i)
		}
	}
}

func TestMaxDelta(t *testing.T) {
	for i := 2; i <= 24; i++ {
		maxDelta := getMaxDelta(i)
		expectedResult := ((i + 7) / 8)
		if getMaxDelta(i) != expectedResult {
			t.Errorf("expected %d max delta for session size %d, got %d", expectedResult, i, maxDelta)
		}
	}
}
