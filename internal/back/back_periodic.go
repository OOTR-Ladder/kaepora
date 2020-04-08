package back

import (
	"database/sql"
	"errors"
	"kaepora/internal/util"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

// createNextScheduledMatchSessions look for leagues with scheduled races and
// creates the MatchSession for the very next race.
func (b *Back) createNextScheduledMatchSessions() error {
	return b.transaction(func(tx *sqlx.Tx) error {
		leagues, err := getLeagues(tx)
		if err != nil {
			return err
		}

		// Create MatchSession
		for _, league := range leagues {
			next := league.Schedule.Next()
			if next.IsZero() {
				continue
			}

			if _, err := getMatchSessionByStartDate(tx, league.ID, next); err != sql.ErrNoRows {
				if err == nil {
					continue // MatchSession already exists
				}

				return err
			}

			log.Printf("info: creating MatchSession for League %s at %s", league.ShortCode, next)
			sess := NewMatchSession(league.ID, next)
			if err := sess.Insert(tx); err != nil {
				return err
			}
		}

		return nil
	})
}

// makeMatchSessionsJoinable looks for races that have reached the time at
// which they can be joined by players and update their status.
func (b *Back) makeMatchSessionsJoinable() error {
	return b.transaction(func(tx *sqlx.Tx) error {
		min := time.Now().Add(-MatchSessionPreparationOffset)
		max := time.Now().Add(-MatchSessionJoinableAfterOffset)
		res, err := tx.Exec(`
            UPDATE MatchSession SET Status = ?
            WHERE DATETIME(StartDate) > DATETIME(?)
              AND DATETIME(StartDate) <= DATETIME(?)
              AND Status = ?
        `,
			MatchSessionStatusJoinable,
			util.TimeAsDateTimeTZ(min),
			util.TimeAsDateTimeTZ(max),
			MatchSessionStatusWaiting,
		)
		if err != nil {
			return err
		}

		if cnt, err := res.RowsAffected(); cnt > 0 && err == nil {
			log.Printf("info: marked %d MatchSession as joinable", cnt)
		}

		return nil
	})
}

// makeMatchSessionsPreparing takes races that are past their "joinable" state
// and put them in the "preparing" state
func (b *Back) makeMatchSessionsPreparing() error {
	return b.transaction(func(tx *sqlx.Tx) error {
		min := time.Now()
		max := time.Now().Add(-MatchSessionPreparationOffset)
		res, err := tx.Exec(`
            UPDATE MatchSession SET Status = ?
            WHERE DATETIME(StartDate) > DATETIME(?)
              AND DATETIME(StartDate) <= DATETIME(?)
              AND Status = ?
        `,
			MatchSessionStatusPreparing,
			util.TimeAsDateTimeTZ(min),
			util.TimeAsDateTimeTZ(max),
			MatchSessionStatusJoinable,
		)
		if err != nil {
			return err
		}

		if cnt, err := res.RowsAffected(); cnt > 0 && err == nil {
			log.Printf("info: marked %d MatchSession as preparing", cnt)
		}

		return nil
	})
}

// doMatchMaking creates all Match and MatchEntry on Matches that reached the
// preparing state, and dispatches seeds to the players.
// This is done in a different transaction than makeMatchSessionsPreparing to
// ensure no one can join when we matchmake/generate the seeds.
func (b *Back) doMatchMaking() error {
	return b.transaction(func(tx *sqlx.Tx) error {
		return errors.New("not implemented: actually creating the matches")
	})
}
