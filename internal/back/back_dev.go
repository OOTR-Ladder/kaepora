package back

import (
	"fmt"
	"kaepora/internal/generator"
	"kaepora/internal/util"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4"
)

func (b *Back) SendDevSeed(
	discordID string,
	leagueShortCode string,
	seed string,
) error {
	return b.transaction(func(tx *sqlx.Tx) error {
		league, err := getLeagueByShortCode(tx, leagueShortCode)
		if err != nil {
			return fmt.Errorf("could not find League: %w", err)
		}

		gen, err := generator.NewGenerator(league.Generator)
		if err != nil {
			return err
		}

		patch, spoilerLog, err := gen.Generate(league.Settings, seed)
		if err != nil {
			return err
		}

		player := Player{DiscordID: null.NewString(discordID, true)}
		b.sendMatchSeedNotification(MatchSession{}, patch, hashFromSpoilerLog(spoilerLog), player, Player{})
		b.sendSpoilerLogNotification(player, seed, spoilerLog)

		return nil
	})
}

func (b *Back) CreateDevMatchSession(leagueShortCode string) error {
	return b.transaction(func(tx *sqlx.Tx) error {
		league, err := getLeagueByShortCode(tx, leagueShortCode)
		if err != nil {
			return fmt.Errorf("could not find League: %w", err)
		}

		// Create joinable
		session := NewMatchSession(league.ID, time.Now().Add(-MatchSessionJoinableAfterOffset))
		session.Status = MatchSessionStatusJoinable
		if err := session.insert(tx); err != nil {
			return err
		}

		// Add players
		for _, playerID := range debugPlayerIDs {
			player, err := getPlayerByID(tx, playerID)
			if err != nil {
				return err
			}
			if session, err = joinCurrentMatchSessionTx(tx, player, league); err != nil {
				return err
			}
		}

		// Matchmake and start countdown, skipping seed generation
		session.Status = MatchSessionStatusPreparing
		session.StartDate = util.TimeAsDateTimeTZ(time.Now().Add(15 * time.Second))
		if err := b.matchMakeSession(tx, session); err != nil {
			return err
		}
		return session.update(tx)
	})
}

func (b *Back) CloseDevMatchSession() error {
	players := make([]Player, 0, len(debugPlayerIDs))
	if err := b.transaction(func(tx *sqlx.Tx) error {
		for _, v := range debugPlayerIDs {
			player, err := getPlayerByID(tx, v)
			if err != nil {
				return err
			}
			players = append(players, player)
		}

		return nil
	}); err != nil {
		return err
	}

	for k := range players {
		if k == len(players)-1 { // skip last player who has no active match
			continue
		}

		if _, err := b.CompleteActiveMatch(players[k]); err != nil {
			return err
		}
	}

	return nil
}

var debugPlayerIDs = []util.UUIDAsBlob{ // nolint:gochecknoglobals
	util.UUIDAsBlob(uuid.MustParse("00000000-1111-0000-0000-000000000000")),
	util.UUIDAsBlob(uuid.MustParse("00000000-2222-0000-0000-000000000000")),
	util.UUIDAsBlob(uuid.MustParse("00000000-3333-0000-0000-000000000000")),
	util.UUIDAsBlob(uuid.MustParse("00000000-4444-0000-0000-000000000000")),
	util.UUIDAsBlob(uuid.MustParse("00000000-5555-0000-0000-000000000000")),
	util.UUIDAsBlob(uuid.MustParse("00000000-6666-0000-0000-000000000000")),
	util.UUIDAsBlob(uuid.MustParse("00000000-7777-0000-0000-000000000000")),
}

// sames indices as debugPlayerIDs
var debugPlayerNames = []string{ // nolint:gochecknoglobals
	"Darunia", "Nabooru", "Rauru",
	"Ruto", "Saria", "Zelda",
	"Impa",
}

func (b *Back) LoadFixtures() error {
	game := NewGame("The Legend of Zelda: Ocarina of Time")
	leagues := []League{
		NewLeague("Standard", "std", game.ID, "oot-randomizer:5.2.12", "s3.json"),
		NewLeague("Debug", "debug", game.ID, "oot-randomizer:5.2.12", "s3.json"),
		NewLeague("Random", "random", game.ID, "oot-settings-randomizer:5.2.12", "s3.json"),
	}

	// 20h PST is 05h CEST, Los Angeles was chosen because it observes DST
	leagues[0].Schedule.SetAll([]string{"14:00 Europe/Paris", "20:00 Europe/Paris"})
	leagues[0].Schedule.Mon = []string{"15:00 Europe/Paris", "21:00 Europe/Paris"}
	leagues[0].Schedule.Wed = []string{"15:00 Europe/Paris", "21:00 Europe/Paris"}
	leagues[0].Schedule.Fri = []string{"15:00 Europe/Paris", "21:00 Europe/Paris"}
	leagues[0].Schedule.Sat = []string{"15:00 Europe/Paris", "21:00 Europe/Paris"}

	return b.transaction(func(tx *sqlx.Tx) error {
		if err := game.insert(tx); err != nil {
			return err
		}

		for _, v := range leagues {
			if err := v.insert(tx); err != nil {
				return err
			}
		}

		for k, v := range debugPlayerNames {
			player := NewPlayer(v)
			player.ID = debugPlayerIDs[k]
			if err := player.insert(tx); err != nil {
				return err
			}
		}

		return nil
	})
}
