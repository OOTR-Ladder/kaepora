package back

import (
	"kaepora/internal/util"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type Game struct {
	ID        util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp
	Name      string
}

func NewGame(name string) Game {
	return Game{
		ID:        util.NewUUIDAsBlob(),
		CreatedAt: util.TimeAsTimestamp(time.Now()),
		Name:      name,
	}
}

func (g *Game) insert(tx *sqlx.Tx) error {
	query, args, err := squirrel.Insert("Game").SetMap(squirrel.Eq{
		"ID":        g.ID,
		"CreatedAt": g.CreatedAt,
		"Name":      g.Name,
	}).ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func getGames(tx *sqlx.Tx) ([]Game, error) {
	var ret []Game
	if err := tx.Select(&ret, "SELECT * FROM Game ORDER BY Name ASC"); err != nil {
		return nil, err
	}

	return ret, nil
}

func getGameByID(tx *sqlx.Tx, id util.UUIDAsBlob) (Game, error) {
	var ret Game
	query := `SELECT * FROM Game WHERE Game.ID = ? LIMIT 1`
	if err := tx.Get(&ret, query, id); err != nil {
		return Game{}, err
	}

	return ret, nil
}
