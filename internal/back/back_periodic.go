package back

import (
	"database/sql"
	"kaepora/internal/util"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

func (b *Back) runPeriodicTasks() error {
	if err := b.createNextScheduledMatchSessions(); err != nil {
		return err
	}

	if err := b.makeMatchSessionsJoinable(); err != nil {
		return err
	}

	sessions, err := b.makeMatchSessionsPreparing()
	if err != nil {
		return err
	}

	if err := b.doMatchMaking(sessions); err != nil {
		return err
	}

	return nil
}

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
			if err := sess.insert(tx); err != nil {
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
// and put them in the "preparing" state. It returns the modified sessions.
func (b *Back) makeMatchSessionsPreparing() ([]MatchSession, error) {
	var sessions []MatchSession

	if err := b.transaction(func(tx *sqlx.Tx) error {
		min := time.Now()
		max := time.Now().Add(-MatchSessionPreparationOffset)
		err := tx.Select(&sessions, `
            SELECT * FROM MatchSession
            WHERE DATETIME(StartDate) > DATETIME(?)
              AND DATETIME(StartDate) <= DATETIME(?)
              AND Status = ?
        `,
			util.TimeAsDateTimeTZ(min),
			util.TimeAsDateTimeTZ(max),
			MatchSessionStatusJoinable,
		)
		if err != nil {
			return err
		}

		for k := range sessions {
			sessions[k].Status = MatchSessionStatusPreparing
			sessions[k].update(tx)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if cnt := len(sessions); cnt > 0 {
		log.Printf("info: marked %d MatchSession as preparing", cnt)
	}

	return sessions, nil
}
