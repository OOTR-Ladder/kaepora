package back

import (
	"database/sql"
	"kaepora/internal/util"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

func (b *Back) prepareScheduledSessions() error {
	leagues, err := b.GetLeagues()
	if err != nil {
		return err
	}

	return b.transaction(func(tx *sqlx.Tx) error {
		// Create MatchSession
		for _, league := range leagues {
			next := league.Schedule.Next()
			if next.IsZero() {
				continue
			}

			if _, err := b.GetMatchSessionByStartDate(league.ID, next); err != sql.ErrNoRows {
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

		// Mark sessions as joinable when they reach the proper offset
		max := time.Now().Add(-MatchSessionJoinableAfterOffset)
		min := time.Now().Add(-MatchSessionCancellableUntilOffset)
		res, err := tx.Exec(`
            UPDATE MatchSession SET Status = ?
            WHERE DATETIME(StartDate) > DATETIME(?)
              AND DATETIME(StartDate) < DATETIME(?)
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
