package back

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

type Back struct {
	db            *sqlx.DB
	notifications chan Notification
}

func New(sqlDriver string, sqlDSN string) (*Back, error) {
	// Why even bother converting names? A single greppable string across all
	// your source code is better than any odd conversion scheme you could ever
	// come up with.
	// HACK: This is global but putting this in init() makes test ugly.
	// As only the Back relies on the DB, this seems like an okay-ish place.
	sqlx.NameMapper = func(v string) string { return v }

	db, err := sqlx.Connect("sqlite3", sqlDSN)
	if err != nil {
		return nil, err
	}

	return &Back{
		db:            db,
		notifications: make(chan Notification, 32),
	}, nil
}

func (b *Back) GetNotificationsChan() <-chan Notification {
	return b.notifications
}

func (b *Back) Run(wg *sync.WaitGroup, done <-chan struct{}) {
	wg.Add(1)
	defer wg.Done()
	log.Print("info: starting Back dæmon")

	for {
		if err := b.runPeriodicTasks(); err != nil {
			log.Printf("error: runPeriodicTasks: %s", err)
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
	game := NewGame("The Legend of Zelda: Ocarina of Time", "oot-randomizer:5.2.12")
	leagues := []League{
		NewLeague("Standard", "std", game.ID, "s3.json"),
		NewLeague("Random rules", "rand", game.ID, "s3.json"),
	}

	leagues[0].Schedule.SetAll([]string{"21:00 Europe/Paris"})
	leagues[1].Schedule.SetAll([]string{"21:00 Europe/Paris"})

	return b.transaction(func(tx *sqlx.Tx) error {
		if err := game.insert(tx); err != nil {
			return err
		}

		for _, v := range leagues {
			if err := v.insert(tx); err != nil {
				return err
			}
		}

		return nil
	})
}
