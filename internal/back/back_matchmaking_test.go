package back

import (
	"errors"
	"fmt"
	"io/ioutil"
	"kaepora/internal/util"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func TestMatchMaking(t *testing.T) {
	back := createFixturedTestBack(t)

	session, err := createSessionAndJoin(back)
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := prepareSession(back, session)
	if err != nil {
		t.Fatal(err)
	}

	if err := back.doMatchMaking(sessions); err != nil {
		t.Fatal(err)
	}

	// Drops after being able to cancel: forfeit and loss
	// There was a random player kicked out of the race already so we can't
	// hardcode a name to forfeit.
	if err := haveSomeoneForfeit(back, sessions); err != nil {
		t.Error(err)
	}
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

func haveSomeoneForfeit(back *Back, sessions []MatchSession) error {
	playerID := util.UUIDAsBlob(sessions[0].GetPlayerIDs()[0])
	var player Player
	if err := back.transaction(func(tx *sqlx.Tx) (err error) {
		player, err = getPlayerByID(tx, playerID)
		return err
	}); err != nil {
		return fmt.Errorf("can get player: %s", err)
	}

	if _, err := back.CancelActiveMatchSession(player); err == nil {
		return errors.New("expected an error when cancelling after MatchSessionCancellableUntilOffset")
	}

	match, err := back.ForfeitActiveMatch(player)
	if err != nil {
		return fmt.Errorf("can't forfeit: %s", err)
	}
	if match.EndedAt.Valid {
		return errors.New("match should not have ended")
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
			getPlayer("Impa"),
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

	back, err := New("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}

	if err := back.transaction(fixtures); err != nil {
		t.Fatal(err)
	}

	return back
}

func fixtures(tx *sqlx.Tx) error {
	game := NewGame("The Test Game", "test:v0")
	leagues := []League{
		NewLeague("The A League", "testa", game.ID, "AJWGAJARB2BGATTACAJBASAGJBHNTHA3EA2UTVAFAA"),
		NewLeague("The B League", "testb", game.ID, "A2WGAJARB2BCAAJWAAJBASAGJBHNTWAKEUPNEOAFAA"),
	}
	playerNames := []string{
		"Darunia", "Nabooru", "Rauru", "Ruto", "Saria", "Zelda", "Impa",
		"Our Lord and Savior ZFG",
	}

	if err := game.insert(tx); err != nil {
		return err
	}

	for _, v := range leagues {
		if err := v.insert(tx); err != nil {
			return err
		}
	}

	for _, v := range playerNames {
		player := NewPlayer(v)
		if err := player.insert(tx); err != nil {
			return err
		}
	}

	return nil
}
