package back

import (
	"fmt"
	"kaepora/internal/util"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	glicko "github.com/zelenin/go-glicko2"
	"gopkg.in/guregu/null.v4"
)

// A Player is a competitor that can be registered to MatchSession and have Match and MatchEntry.
type Player struct {
	ID        util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp
	Name      string
	DiscordID null.String
	StreamURL string

	Rating PlayerRating `db:"-"`
}

type byRating []Player

func (a byRating) Len() int {
	return len(a)
}

func (a byRating) Less(i, j int) bool {
	return a[i].Rating.GlickoRating().R() < a[j].Rating.GlickoRating().R()
}

func (a byRating) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func NewPlayer(name string) Player {
	return Player{
		ID:        util.NewUUIDAsBlob(),
		CreatedAt: util.TimeAsTimestamp(time.Now()),
		Name:      name,
	}
}

func (p *Player) GlickoRating() *glicko.Rating {
	return p.Rating.GlickoRating()
}

func (p *Player) insert(tx *sqlx.Tx) error {
	query, args, err := squirrel.Insert("Player").SetMap(squirrel.Eq{
		"ID":        p.ID,
		"CreatedAt": p.CreatedAt,
		"Name":      p.Name,
		"DiscordID": p.DiscordID,
		"StreamURL": p.StreamURL,
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
		"StreamURL": p.StreamURL,
	}).Where("Player.ID = ?", p.ID).ToSql()
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

func getPlayerByName(tx *sqlx.Tx, name string) (Player, error) {
	var ret Player
	query := `SELECT * FROM Player WHERE Player.Name = ? LIMIT 1`
	if err := tx.Get(&ret, query, name); err != nil {
		return Player{}, err
	}

	return ret, nil
}

func getPlayerByID(tx *sqlx.Tx, id util.UUIDAsBlob) (Player, error) {
	var ret Player
	query := `SELECT * FROM Player WHERE Player.ID = ? LIMIT 1`
	if err := tx.Get(&ret, query, id); err != nil {
		return Player{}, err
	}

	return ret, nil
}

func (b *Back) UpdateDiscordPlayerName(discordID string, name string) error {
	return b.transaction(func(tx *sqlx.Tx) error {
		player, err := getPlayerByDiscordID(tx, discordID)
		if err != nil {
			return err
		}

		if player.Name == name {
			return util.ErrPublic("that's your name already")
		}

		if len(name) < 3 || len(name) > 32 {
			return util.ErrPublic("your name must be between 3 and 32 characters")
		}

		if _, err := getPlayerByName(tx, name); err == nil {
			return util.ErrPublic("this name is taken already")
		}

		player.Name = name
		return player.Update(tx)
	})
}

func (b *Back) RegisterDiscordPlayer(discordID, name string) (player Player, _ error) {
	if err := b.transaction(func(tx *sqlx.Tx) error {
		if _, err := getPlayerByDiscordID(tx, discordID); err == nil {
			return util.ErrPublic("you are already registered")
		}

		if _, err := getPlayerByName(tx, name); err == nil {
			return util.ErrPublic(fmt.Sprintf("the name `%s` is taken already, please give me another name", name))
		}

		player = NewPlayer(name)
		player.DiscordID = util.NullString(discordID)
		if err := player.insert(tx); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return Player{}, err
	}

	return player, nil
}

func (b *Back) UpdateDiscordPlayerStreamURL(discordID, streamURL string) (string, error) {
	normalizedURL, err := util.NormalizeStreamURL(streamURL)
	if err != nil {
		return "", err
	}

	if err := b.transaction(func(tx *sqlx.Tx) error {
		player, err := getPlayerByDiscordID(tx, discordID)
		if err != nil {
			return err
		}

		player.StreamURL = normalizedURL
		// TODO, maybe check if the URL does not 404.

		return player.Update(tx)
	}); err != nil {
		return "", err
	}

	return normalizedURL, nil
}

func getPlayersByMatches(tx *sqlx.Tx, matches []Match) (map[util.UUIDAsBlob]Player, error) {
	ids := make([]util.UUIDAsBlob, 0, len(matches)*2)
	for k := range matches {
		ids = append(ids, matches[k].Entries[0].PlayerID, matches[k].Entries[1].PlayerID)
	}

	if len(ids) == 0 {
		return map[util.UUIDAsBlob]Player{}, nil
	}

	query, args, err := sqlx.In(`SELECT * FROM Player WHERE ID IN(?)`, ids)
	if err != nil {
		return nil, err
	}
	query = tx.Rebind(query)

	players := make([]Player, 0, len(ids))
	if err := tx.Select(&players, query, args...); err != nil {
		return nil, err
	}

	ret := make(map[util.UUIDAsBlob]Player, len(players))
	for k := range players {
		ret[players[k].ID] = players[k]
	}

	return ret, nil
}
