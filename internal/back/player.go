package back

import (
	"database/sql"
	"kaepora/internal/util"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type Player struct {
	ID        util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp
	Name      string
	DiscordID sql.NullString
}

func NewPlayer(name string) Player {
	return Player{
		ID:        util.NewUUIDAsBlob(),
		CreatedAt: util.TimeAsTimestamp(time.Now()),
		Name:      name,
	}
}

func (p *Player) Insert(tx *sqlx.Tx) error {
	query, args, err := squirrel.Insert("Player").SetMap(squirrel.Eq{
		"ID":        p.ID,
		"CreatedAt": p.CreatedAt,
		"Name":      p.Name,
		"DiscordID": p.DiscordID,
	}).ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func (p *Player) Update(tx *sqlx.Tx) error {
	query, args, err := squirrel.Update("Player").SetMap(squirrel.Eq{
		"Name":      p.Name,
		"DiscordID": p.DiscordID,
	}).
		Where("Player.ID = ?", p.ID).
		ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func (b *Back) UpdatePlayer(p Player) error {
	return b.transaction(p.Update)
}

func (b *Back) RegisterPlayer(p Player) error {
	return b.transaction(p.Insert)
}

func (b *Back) GetPlayerByDiscordID(discordID string) (Player, error) {
	var ret Player
	query := `SELECT * FROM Player WHERE Player.DiscordID = ? LIMIT 1`
	if err := b.db.Get(&ret, query, discordID); err != nil {
		return Player{}, err
	}

	return ret, nil
}

func (b *Back) GetPlayerByName(name string) (Player, error) {
	var ret Player
	query := `SELECT * FROM Player WHERE Player.Name = ? LIMIT 1`
	if err := b.db.Get(&ret, query, name); err != nil {
		return Player{}, err
	}

	return ret, nil
}
