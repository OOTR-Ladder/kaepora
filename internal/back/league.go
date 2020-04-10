package back

import (
	"kaepora/internal/util"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type League struct {
	ID        util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp
	Name      string
	ShortCode string
	GameID    util.UUIDAsBlob
	Settings  string
	Schedule  Schedule

	AnnounceDiscordChannelID string
}

func NewLeague(name string, shortCode string, gameID util.UUIDAsBlob, settings string) League {
	return League{
		ID:        util.NewUUIDAsBlob(),
		CreatedAt: util.TimeAsTimestamp(time.Now()),
		GameID:    gameID,
		Name:      name,
		ShortCode: shortCode,
		Settings:  settings,
		Schedule:  NewSchedule(),
	}
}

func (l *League) insert(tx *sqlx.Tx) error {
	query, args, err := squirrel.Insert("League").SetMap(squirrel.Eq{
		"ID":        l.ID,
		"CreatedAt": l.CreatedAt,
		"GameID":    l.GameID,
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
	if err := tx.Select(&ret, "SELECT * FROM League ORDER BY League.Name ASC"); err != nil {
		return nil, err
	}

	return ret, nil
}

func getLeagueByShortCode(tx *sqlx.Tx, shortCode string) (League, error) {
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
