package main

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func loadFixtures() {
	db, err := sqlx.Connect("sqlite3", "./kaepora.db")
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(1)

	gameID := uuid.New()

	tx := db.MustBegin()
	tx.MustExec(
		`INSERT INTO Game (ID, Name, Generator) VALUES
        (?, "The Legend of Zelda: Ocarina of Time", "OoT-Randomizer:v5.2")`,
		gameID,
	)
	tx.MustExec(
		`INSERT INTO League (ID, GameID, Name, Settings) VALUES
        (:stdID, :gameID, "Standard", "AJWGAJARB2BCAAJWAAJBASAGJBHNTHA3EA2UTVAFAA"),
        (:randomID, :gameID, "Random rules", "A2WGAJARB2BCAAJWAAJBASAGJBHNTHA3EA2UTVAFAA")`,

		sql.Named("stdID", uuid.New()),
		sql.Named("randomID", uuid.New()),
		sql.Named("gameID", gameID),
	)

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}
