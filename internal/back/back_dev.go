package back

import (
	"fmt"
	"kaepora/internal/generator/oot"
	"kaepora/internal/util"

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

		gen, err := b.generatorFactory.NewGenerator(league.Generator)
		if err != nil {
			return err
		}

		out, err := gen.Generate(league.Settings, seed)
		if err != nil {
			return err
		}

		if err := gen.UnlockSpoilerLog(out.State); err != nil {
			return err
		}

		zlibLog, err := util.NewZLIBBlob(out.SpoilerLog)
		if err != nil {
			return err
		}

		player := Player{DiscordID: null.NewString(discordID, true)}
		b.sendMatchSeedNotification(
			MatchSession{},
			gen.GetDownloadURL(out.State), out,
			player, Player{},
		)
		b.sendSpoilerLogNotification(player, seed, zlibLog)

		return nil
	})
}

func (b *Back) LoadFixtures() error {
	game := NewGame("The Legend of Zelda: Ocarina of Time")
	leagues := []League{
		NewLeague("Standard", "std", game.ID, oot.RandomizerAPIName+":5.2.0", "s3.json"),
		NewLeague("Debug", "debug", game.ID, oot.RandomizerName+":5.2.13", "s3.json"),
		NewLeague("Random", "random", game.ID, oot.SettingsRandomizerName+":5.2.13", "s3.json"),
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

		return nil
	})
}
