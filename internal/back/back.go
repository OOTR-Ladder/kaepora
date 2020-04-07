package back

import (
	"fmt"
	"kaepora/internal/util"
	"log"
	"sync"
	"time"

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

	return &Back{
		db: db,
	}, nil
}

func (b *Back) Run(wg *sync.WaitGroup, done <-chan struct{}) {
	wg.Add(1)
	defer wg.Done()
	log.Print("info: starting Back dæmon")

	for {
		e := []error{
			b.prepareScheduledSessions(),
		}

		if err := util.ConcatErrors(e); err != nil {
			log.Printf("errors: %s", err)
		}

		select {
		case <-time.After(1 * time.Minute):
		case <-done:
			return
		}
	}
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

func (b *Back) LoadFixtures() error {
	game := NewGame("The Legend of Zelda: Ocarina of Time", "OoT-Randomizer:v5.2")
	leagues := []League{
		NewLeague("Standard", "std", game.ID, "AJWGAJARB2BCAAJWAAJBASAGJBHNTHA3EA2UTVAFAA"),
		NewLeague("Random rules", "rand", game.ID, "A2WGAJARB2BCAAJWAAJBASAGJBHNTHA3EA2UTVAFAA"),
	}

	leagues[0].Schedule.SetAll([]string{"21:00 Europe/Paris", "21:00 Asia/Tokyo"})
	leagues[1].Schedule.SetAll([]string{"19:00 America/Mexico_City", "20:00 Europe/Berlin"})

	return b.transaction(func(tx *sqlx.Tx) error {
		if err := game.Insert(tx); err != nil {
			return err
		}

		for _, v := range leagues {
			if err := v.Insert(tx); err != nil {
				return err
			}
		}

		return nil
	})
}
