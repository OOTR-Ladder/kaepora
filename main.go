package main

import (
	"context"
	"flag"
	"fmt"
	"kaepora/internal/back"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// Version holds the build-time version string.
var Version = "unknown" // nolint:gochecknoglobals

func main() {
	flag.Parse()

	switch flag.Arg(0) { // commands not requiring a back
	case "version":
		fmt.Fprintf(os.Stdout, "Kaepora %s\n", Version)
		return
	case "help":
		fmt.Fprint(os.Stdout, help())
		return
	}

	back, err := back.New("sqlite3", "./kaepora.db")
	if err != nil {
		log.Fatal(err)
	}

	switch flag.Arg(0) {
	case "fixtures":
		if err := loadFixtures(back); err != nil {
			log.Fatal(err)
		}
	case "serve":
		if err := serve(back); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Fprint(os.Stderr, help())
		os.Exit(1)
	}
}

func help() string {
	return fmt.Sprintf(`
Kaepora is a tool to manage the "Ocarina of Time: Randomizer"
competitive ladder.

Usage: %[1]s COMMAND [ARGS…]

COMMANDS
    fixtures    create default data for quick testing during development
    help        display this help
    serve       start the Discord bot
    version     display the current version
`,
		os.Args[0],
	)
}

func loadFixtures(b *back.Back) error {
	game := back.NewGame("The Legend of Zelda: Ocarina of Time", "OoT-Randomizer:v5.2")
	leagues := []back.League{
		back.NewLeague("Standard", "std", game.ID, "AJWGAJARB2BCAAJWAAJBASAGJBHNTHA3EA2UTVAFAA"),
		back.NewLeague("Random rules", "rand", game.ID, "A2WGAJARB2BCAAJWAAJBASAGJBHNTHA3EA2UTVAFAA"),
	}

	return b.Transaction(context.Background(), func(tx *sqlx.Tx) error {
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
