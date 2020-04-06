package back

import (
	"context"
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

type TransactionCallback func(*sqlx.Tx) error

func (b *Back) Transaction(ctx context.Context, cb TransactionCallback) error {
	tx, err := b.db.BeginTxx(ctx, nil)
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
