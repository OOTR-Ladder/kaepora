package back

import (
	"database/sql"
	"kaepora/internal/util"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type Game struct {
	ID        util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp
	Name      string
	Generator sql.NullString
}

func NewGame(name string, generator string) Game {
	return Game{
		ID:        util.NewUUIDAsBlob(),
		CreatedAt: util.TimeAsTimestamp(time.Now()),
		Name:      name,
		Generator: util.NullString(generator),
	}
}

func (g *Game) Insert(tx *sqlx.Tx) error {
	query, args, err := squirrel.Insert("Game").SetMap(squirrel.Eq{
		"ID":        g.ID,
		"CreatedAt": g.CreatedAt,
		"Name":      g.Name,
		"Generator": g.Generator,
	}).ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}
