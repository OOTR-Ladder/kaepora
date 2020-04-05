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
}

func NewLeague(name string, shortCode string, gameID util.UUIDAsBlob, settings string) League {
	return League{
		ID:        util.NewUUIDAsBlob(),
		CreatedAt: util.TimeAsTimestamp(time.Now()),
		GameID:    gameID,
		Name:      name,
		ShortCode: shortCode,
		Settings:  settings,
	}
}

func (l *League) Insert(tx *sqlx.Tx) error {
	query, args, err := squirrel.Insert("League").SetMap(squirrel.Eq{
		"ID":        l.ID,
		"CreatedAt": l.CreatedAt,
		"GameID":    l.GameID,
		"Name":      l.Name,
		"ShortCode": l.ShortCode,
		"Settings":  l.Settings,
	}).ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}
