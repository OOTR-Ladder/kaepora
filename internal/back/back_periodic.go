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

	if err := b.startMatchSessions(); err != nil {
		return err
	}

	if err := b.endMatchSessionsAndUpdateRanks(); err != nil {
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

			sess := NewMatchSession(league.ID, next)
			if err := sess.insert(tx); err != nil {
				return err
			}
			if err := b.sendSessionCountdownNotification(tx, sess); err != nil {
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
		var sessions []MatchSession
		if err := tx.Select(&sessions, `
            SELECT * FROM MatchSession
            WHERE DATETIME(StartDate) > DATETIME(?)
              AND DATETIME(StartDate) <= DATETIME(?)
              AND Status = ?
        `,
			util.TimeAsDateTimeTZ(min),
			util.TimeAsDateTimeTZ(max),
			MatchSessionStatusWaiting,
		); err != nil {
			return err
		}

		for k := range sessions {
			log.Printf("debug: put session %s in MatchSessionStatusJoinable", sessions[k].ID)
			sessions[k].Status = MatchSessionStatusJoinable
			if err := sessions[k].update(tx); err != nil {
				return err
			}

			if err := b.sendSessionCountdownNotification(tx, sessions[k]); err != nil {
				return err
			}
		}

		return nil
	})
}

// makeMatchSessionsPreparing takes races that are past their "joinable" state
// and put them in the "preparing" state. It returns the modified sessions.
func (b *Back) makeMatchSessionsPreparing() ([]MatchSession, error) {
	var sessions []MatchSession

	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		sessions, err = getMatchSessionsToPrepare(tx)
		if err != nil {
			return err
		}

		for k := range sessions {
			if len(sessions[k].GetPlayerIDs()) < 2 {
				sessions[k].Status = MatchSessionStatusClosed
				log.Printf("info: no players for session %s", sessions[k].ID.UUID())
				if err := sessions[k].update(tx); err != nil {
					return err
				}
				if err := b.sendMatchSessionEmptyNotification(tx, sessions[k]); err != nil {
					return err
				}
				continue
			}

			log.Printf("debug: put session %s in MatchSessionStatusPreparing", sessions[k].ID)
			sessions[k].Status = MatchSessionStatusPreparing
			if err := sessions[k].update(tx); err != nil {
				return err
			}

			if err := b.sendSessionCountdownNotification(tx, sessions[k]); err != nil {
				return err
			}
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

func (b *Back) startMatchSessions() error {
	if err := b.transaction(func(tx *sqlx.Tx) error {
		sessions, err := getMatchSessionsToStart(tx)
		if err != nil {
			return err
		}

		for k := range sessions {
			if err := sessions[k].start(tx); err != nil {
				return err
			}

			if err := sessions[k].update(tx); err != nil {
				return err
			}

			if err := b.sendSessionCountdownNotification(tx, sessions[k]); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func getMatchSessionsToStart(tx *sqlx.Tx) ([]MatchSession, error) {
	query := `SELECT * FROM MatchSession
    WHERE DATETIME(StartDate) <= DATETIME(?) AND Status = ?`
	var sessions []MatchSession
	if err := tx.Select(
		&sessions, query,
		util.TimeAsDateTimeTZ(time.Now()),
		MatchSessionStatusPreparing,
	); err != nil {
		return nil, err
	}

	return sessions, nil
}

func getMatchSessionsToEnd(tx *sqlx.Tx) ([]MatchSession, map[util.UUIDAsBlob][]Match, error) {
	query := `
    SELECT MatchSession.* FROM MatchSession
    WHERE
        DATETIME(MatchSession.StartDate) < DATETIME(?)
        AND MatchSession.Status = ?`
	var sessions []MatchSession
	if err := tx.Select(
		&sessions, query,
		util.TimeAsDateTimeTZ(time.Now()),
		MatchSessionStatusInProgress,
	); err != nil {
		return nil, nil, err
	}

	var ret []MatchSession
	sessMatches := map[util.UUIDAsBlob][]Match{}

loop:
	for k := range sessions {
		matches, err := getMatchesBySessionID(tx, sessions[k].ID)
		if err != nil {
			return nil, nil, err
		}

		// Should not happen, but a session with no match cannot even be in progress so close it.
		if len(matches) == 0 {
			ret = append(ret, sessions[k])
			continue
		}

		for j := range matches {
			// HACK: A Match has no status but a date that is written only when
			// closed, check match status with that date.
			if !matches[j].EndedAt.Valid {
				continue loop
			}
		}

		ret = append(ret, sessions[k])
		sessMatches[sessions[k].ID] = matches
	}

	return ret, sessMatches, nil
}

func (b *Back) endMatchSessionsAndUpdateRanks() error {
	return b.transaction(func(tx *sqlx.Tx) error {
		sessions, matches, err := getMatchSessionsToEnd(tx)
		if err != nil {
			return err
		}

		for k := range sessions {
			sessions[k].Status = MatchSessionStatusClosed
			if err := sessions[k].update(tx); err != nil {
				return err
			}

			if err := b.updateRankingsForMatchSession(tx, sessions[k], matches[sessions[k].ID]); err != nil {
				return err
			}
		}

		return nil
	})
}

func getMatchSessionsToPrepare(tx *sqlx.Tx) ([]MatchSession, error) {
	var sessions []MatchSession

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
		return nil, err
	}

	return sessions, nil
}
