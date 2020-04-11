package back

import (
	"database/sql"
	"errors"
	"fmt"
	"kaepora/internal/util"
	"time"

	"github.com/jmoiron/sqlx"
)

// Put bot-specific oddities here

func (b *Back) GetGamesLeaguesAndTheirNextSessionStartDate() (
	[]Game,
	[]League,
	map[util.UUIDAsBlob]time.Time,
	error,
) {
	var (
		games   []Game
		leagues []League
		times   map[util.UUIDAsBlob]time.Time
	)

	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		games, err = getGames(tx)
		if err != nil {
			return err
		}

		leagues, err = getLeagues(tx)
		if err != nil {
			return err
		}
		times = make(map[util.UUIDAsBlob]time.Time, len(leagues))

		for _, league := range leagues {
			session, err := getNextMatchSessionForLeague(tx, league.ID)
			if err != nil {
				return err
			}
			times[league.ID] = session.StartDate.Time()
		}

		return nil
	}); err != nil {
		return nil, nil, nil, err
	}

	return games, leagues, times, nil
}

func getPlayerByDiscordID(tx *sqlx.Tx, discordID string) (Player, error) {
	var ret Player
	query := `SELECT * FROM Player WHERE Player.DiscordID = ? LIMIT 1`
	if err := tx.Get(&ret, query, discordID); err != nil {
		return Player{}, err
	}

	return ret, nil
}

func (b *Back) SetLeagueAnnounceChannel(shortcode, channelID string) error {
	return b.transaction(func(tx *sqlx.Tx) error {
		league, err := getLeagueByShortCode(tx, shortcode)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return util.ErrPublic(fmt.Sprintf("invalid shortcode '%s'", shortcode))
			}

			return err
		}

		league.AnnounceDiscordChannelID = channelID
		return league.update(tx)
	})
}

// TODO remove the need for this
func (b *Back) GetPlayerByDiscordID(discordID string) (player Player, _ error) {
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		player, err = getPlayerByDiscordID(tx, discordID)
		return err
	}); err != nil {
		return Player{}, err
	}

	return player, nil
}
