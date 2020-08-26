package back

import (
	"fmt"
	"kaepora/internal/util"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

const DefaultTokenLifetime = 24 * time.Hour

// A token is an expirable random string used to authenticate players.
type token struct {
	ID        util.UUIDAsBlob
	CreatedAt util.TimeAsTimestamp
	ExpiresAt util.TimeAsTimestamp
	PlayerID  util.UUIDAsBlob
}

func (b *Back) GetPlayerFromTokenID(tokenID util.UUIDAsBlob) (*Player, error) {
	var ret *Player
	if err := b.transaction(func(tx *sqlx.Tx) error {
		token, err := getTokenByID(tx, tokenID)
		if err != nil {
			return fmt.Errorf("token not found: %s", err)
		}

		player, err := getPlayerByID(tx, token.PlayerID)
		if err != nil {
			return fmt.Errorf("player not found: %s", err)
		}

		if b.config.IsDiscordIDBanned(player.DiscordID.String) {
			return fmt.Errorf("user %s is banned", player.Name)
		}

		ret = &player
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func newToken(playerID util.UUIDAsBlob, lifetime time.Duration) token {
	now := time.Now()

	return token{
		ID:        util.NewUUIDAsBlob(),
		CreatedAt: util.TimeAsTimestamp(now),
		ExpiresAt: util.TimeAsTimestamp(now.Add(lifetime)),
		PlayerID:  playerID,
	}
}

// CreateToken creates a new authentication token and returns its ID as a string.
func (b *Back) CreateToken(playerID util.UUIDAsBlob, lifetime time.Duration) (string, error) {
	t := newToken(playerID, lifetime)
	if err := b.transaction(t.insert); err != nil {
		return "", err
	}

	return t.ID.String(), nil
}

func (b *Back) CreateTokenForPlayerName(name string, lifetime time.Duration) (string, error) {
	var ret string
	if err := b.transaction(func(tx *sqlx.Tx) error {
		player, err := getPlayerByName(tx, name)
		if err != nil {
			return err
		}

		t := newToken(player.ID, lifetime)
		ret = t.ID.String()
		if err := t.insert(tx); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return "", err
	}

	return ret, nil
}

func getTokenByID(tx *sqlx.Tx, id util.UUIDAsBlob) (token, error) {
	query := `SELECT * FROM "Token" WHERE ID = ? LIMIT 1`

	var ret token
	if err := tx.Get(&ret, query, id); err != nil {
		return token{}, err
	}

	return ret, nil
}

// insert inserts the token and remove any previous token associated with the player.
func (t token) insert(tx *sqlx.Tx) error {
	_, err := tx.Exec(
		`DELETE FROM "Token" WHERE "PlayerID" = ?`,
		t.PlayerID,
	)
	if err != nil {
		return err
	}

	query, args, err := squirrel.Insert("Token").SetMap(squirrel.Eq{
		"ID":        t.ID,
		"CreatedAt": t.CreatedAt,
		"ExpiresAt": t.ExpiresAt,
		"PlayerID":  t.PlayerID,
	}).ToSql()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

func (b *Back) pruneExpiredTokens() error {
	return b.transaction(func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`DELETE FROM "Token" WHERE "ExpiresAt" < ?`, time.Now().Unix())
		return err
	})
}
