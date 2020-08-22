// package back contains all the business logic.
package back

import (
	"fmt"
	"kaepora/internal/config"
	"kaepora/internal/generator"
	"kaepora/internal/generator/factory"
	"kaepora/internal/util"
	"kaepora/pkg/ootrapi"
	"log"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// Back holds the global state of the application and all the business logic.
// It is the back-end of the web and bot front-ends.
type Back struct {
	db               *sqlx.DB
	generatorFactory factory.Factory
	config           *config.Config

	// notifications receives content from the Back and MUST be consumed externally.
	notifications chan Notification

	// It is possible to fetch the same session twice to count it down, this
	// cache avoid starting the same session twice.  This is only used in
	// countdownAndStartMatchSession which is _not_ run concurrently.
	countingDown map[util.UUIDAsBlob]struct{}
}

func New(sqlDriver, sqlDSN string, config *config.Config) (*Back, error) {
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
		db:               db,
		config:           config,
		notifications:    make(chan Notification, 32),
		countingDown:     map[util.UUIDAsBlob]struct{}{},
		generatorFactory: factory.New(ootrapi.New(config.OOTRAPIKey)),
	}, nil
}

func (b *Back) GetGenerator(name string) (generator.Generator, error) {
	return b.generatorFactory.NewGenerator(name)
}

// GetNotificationsChan returns the channel on which the Back sends the textual
// notifications destined to either a specific player or a whole channel.
func (b *Back) GetNotificationsChan() <-chan Notification {
	return b.notifications
}

// Run performs all the matchmaking business until the done channel is closed.
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

// transaction runs the given callback in a SQL transaction and either COMMIT
// if the returned error is nil, or ROLLBACK if the error is non-nil.
// All interactions with the DB should go through this function.
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
