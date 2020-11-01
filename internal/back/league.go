package back

import (
	"kaepora/internal/back/schedule"
	"kaepora/internal/util"
	"log"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4"
)

// A League is a set of rules, randomizer settings, and scheduling parameters.
// All match sessions belong to a League which announces them in a specific
// discord channel.
type League struct {
	ID        util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp
	Name      string
	ShortCode string
	GameID    util.UUIDAsBlob
	Generator string
	Settings  string
	Schedule  schedule.Config

	AnnounceDiscordChannelID null.String
}

func NewLeague(name string, shortCode string, gameID util.UUIDAsBlob, generator, settings string) League {
	return League{
		ID:        util.NewUUIDAsBlob(),
		CreatedAt: util.TimeAsTimestamp(time.Now()),
		GameID:    gameID,
		Generator: generator,
		Name:      name,
		ShortCode: shortCode,
		Settings:  settings,
		Schedule:  schedule.Config{},
	}
}

func (l *League) Scheduler() schedule.Scheduler {
	s, err := schedule.New(l.Schedule)
	if err != nil { // HACK accommodate tests
		log.Printf("warning: %s", err)
	}

	return s
}

func (l *League) insert(tx *sqlx.Tx) error {
	query, args, err := squirrel.Insert("League").SetMap(squirrel.Eq{
		"ID":        l.ID,
		"CreatedAt": l.CreatedAt,
		"GameID":    l.GameID,
		"Generator": l.Generator,
		"Name":      l.Name,
		"ShortCode": l.ShortCode,
		"Settings":  l.Settings,
		"Schedule":  l.Schedule,

		"AnnounceDiscordChannelID": l.AnnounceDiscordChannelID,
	}).ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func (l *League) update(tx *sqlx.Tx) error {
	query, args, err := squirrel.Update("League").SetMap(squirrel.Eq{
		"GameID":    l.GameID,
		"Generator": l.Generator,
		"Name":      l.Name,
		"ShortCode": l.ShortCode,
		"Settings":  l.Settings,
		"Schedule":  l.Schedule,

		"AnnounceDiscordChannelID": l.AnnounceDiscordChannelID,
	}).Where("League.ID = ?", l.ID).ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func getLeagues(tx *sqlx.Tx) ([]League, error) {
	var ret []League
	// HACK, order is DESC to show Standard first, custom ordering should be
	// implemented if there's more than two leagues.
	if err := tx.Select(&ret, "SELECT * FROM League ORDER BY League.Name DESC"); err != nil {
		return nil, err
	}

	return ret, nil
}

func getLeagueByShortCode(tx *sqlx.Tx, shortCode string) (League, error) {
	if shortCode == "" {
		return League{}, util.ErrPublic("you need to give me a league shortcode, see `!leagues` and `!help`")
	}

	var ret League
	query := `SELECT * FROM League WHERE League.ShortCode = ? LIMIT 1`
	if err := tx.Get(&ret, query, shortCode); err != nil {
		return League{}, err
	}

	return ret, nil
}

func getLeagueByID(tx *sqlx.Tx, id util.UUIDAsBlob) (League, error) {
	var ret League
	query := `SELECT * FROM League WHERE League.ID = ? LIMIT 1`
	if err := tx.Get(&ret, query, id); err != nil {
		return League{}, err
	}

	return ret, nil
}
