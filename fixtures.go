package main

import (
	"context"
	"kaepora/internal/back"
	"kaepora/internal/util"

	"github.com/jmoiron/sqlx"
)

func loadFixtures() error {
	db, err := sqlx.Connect("sqlite3", "./kaepora.db")
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(1)

	game := back.NewGame("The Legend of Zelda: Ocarina of Time", "OoT-Randomizer:v5.2")
	leagues := []back.League{
		back.NewLeague("Standard", "std", game.ID, "AJWGAJARB2BCAAJWAAJBASAGJBHNTHA3EA2UTVAFAA"),
		back.NewLeague("Random rules", "rand", game.ID, "A2WGAJARB2BCAAJWAAJBASAGJBHNTHA3EA2UTVAFAA"),
	}

	return util.Transaction(context.Background(), db, func(tx *sqlx.Tx) error {
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
