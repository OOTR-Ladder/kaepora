package util

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Storable interface {
	Store(*sqlx.Tx) error
}

// NullString is a shorthand to create a valid sql.NullString or an invalid one
// if the given string is empty.
func NullString(v string) sql.NullString {
	return sql.NullString{
		String: v,
		Valid:  v != "",
	}
}

type TransactionCallback func(*sqlx.Tx) error

func Transaction(ctx context.Context, db *sqlx.DB, cb TransactionCallback) error {
	tx, err := db.BeginTxx(ctx, nil)
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
