package back

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Back struct {
	db *sqlx.DB
}

func New(sqlDriver string, sqlDSN string) (*Back, error) {
	db, err := sqlx.Connect("sqlite3", "./kaepora.db")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	return &Back{
		db: db,
	}, nil
}

type transactionCallback func(*sqlx.Tx) error

func (b *Back) transaction(cb transactionCallback) error {
	tx, err := b.db.Beginx()
	if err != nil {
		return err
	}

	if err := cb(tx); err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			return fmt.Errorf("rollback error: %s\noriginal error: %s", err2, err)
		}

		return err
	}

	return tx.Commit()
}

type Storable interface {
	Store(*sqlx.Tx) error
}

func (b *Back) LoadFixtures() error {
	game := NewGame("The Legend of Zelda: Ocarina of Time", "OoT-Randomizer:v5.2")
	leagues := []League{
		NewLeague("Standard", "std", game.ID, "AJWGAJARB2BCAAJWAAJBASAGJBHNTHA3EA2UTVAFAA"),
		NewLeague("Random rules", "rand", game.ID, "A2WGAJARB2BCAAJWAAJBASAGJBHNTHA3EA2UTVAFAA"),
	}

	return b.transaction(func(tx *sqlx.Tx) error {
		if err := game.Insert(tx); err != nil {
			return err
		}

		for _, v := range leagues {
			if err := v.Insert(tx); err != nil {
				return nil
			}
		}

		return nil
	})
}
