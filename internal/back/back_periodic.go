package back

import (
	"database/sql"
	"log"

	"github.com/jmoiron/sqlx"
)

func (b *Back) prepareScheduledSessions() error {
	leagues, err := b.GetLeagues()
	if err != nil {
		return err
	}

	return b.transaction(func(tx *sqlx.Tx) error {
		for _, league := range leagues {
			next := league.Schedule.Next()
			if next.IsZero() {
				continue
			}

			if _, err := b.GetMatchSessionByStartDate(next); err != sql.ErrNoRows {
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
