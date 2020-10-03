package back

import (
	"database/sql"
	"kaepora/internal/util"
	"log"
	"runtime/debug"
	"time"

	"github.com/jmoiron/sqlx"
)

func (b *Back) runPeriodicTasks() error {
	defer func() {
		r := recover()
		if r != nil {
			log.Print("panic: ", r)
			log.Printf("%s", debug.Stack())
		}
	}()

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

	// This is done in a different transaction than makeMatchSessionsPreparing
	// to ensure no one can join when we matchmake/generate the seeds.
	if err := b.doMatchMaking(sessions); err != nil {
		return err
	}

	if err := b.startMatchSessions(); err != nil {
		return err
	}

	if err := b.closeMatchSessionsAndUpdateRanks(); err != nil {
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
			next := league.Scheduler().Next()
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
			if err := b.sendSessionStatusUpdateNotification(tx, sess); err != nil {
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

			if err := b.sendSessionStatusUpdateNotification(tx, sessions[k]); err != nil {
				return err
			}
		}

		return nil
	})
}

// makeMatchSessionsPreparing takes races that are past their "joinable" state
// and put them in the "preparing" state. It returns the modified sessions.
func (b *Back) makeMatchSessionsPreparing() ([]MatchSession, error) {
	var ret []MatchSession

	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		sessions, err := getMatchSessionsToPrepare(tx)
		if err != nil {
			return err
		}

		for k := range sessions {
			if playerIDs := sessions[k].GetPlayerIDs(); len(playerIDs) < 2 {
				sessions[k].Status = MatchSessionStatusClosed
				log.Printf("info: no players for session %s", sessions[k].ID.UUID())
				if err := sessions[k].update(tx); err != nil {
					return err
				}
				if err := b.sendMatchSessionEmptyNotification(tx, sessions[k], playerIDs); err != nil {
					return err
				}

				continue
			}

			log.Printf("debug: put session %s in MatchSessionStatusPreparing", sessions[k].ID)
			sessions[k].Status = MatchSessionStatusPreparing
			if err := sessions[k].update(tx); err != nil {
				return err
			}

			if err := b.sendSessionStatusUpdateNotification(tx, sessions[k]); err != nil {
				return err
			}

			ret = append(ret, sessions[k])
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if cnt := len(ret); cnt > 0 {
		log.Printf("info: marked %d MatchSession as preparing", cnt)
	}

	return ret, nil
}

func (b *Back) startMatchSessions() error {
	var sessions []MatchSession
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		sessions, err = getMatchSessionsToStart(tx)
		return err
	}); err != nil {
		return err
	}

	for k := range sessions {
		go b.countdownAndStartMatchSession(sessions[k])
	}

	return nil
}

// for tests only, we don't want to wait 90s per test.
func (b *Back) instantlyStartMatchSessions() error {
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
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (b *Back) countdownAndStartMatchSession(session MatchSession) {
	if _, ok := b.countingDown[session.ID]; ok {
		log.Printf("debug: not counting down session %s twice", session.ID)
		return
	}
	log.Printf("info: starting countdown for session %s", session.ID)
	b.countingDown[session.ID] = struct{}{}

	countdowns := []time.Duration{ // order matters
		time.Minute, 30 * time.Second, 10 * time.Second,
		5 * time.Second, 4 * time.Second, 3 * time.Second,
		2 * time.Second, 1 * time.Second,
	}
	min := countdowns[0] + 1

	for {
		delta := time.Until(session.StartDate.Time())
		if delta <= 0 {
			// HACK, send the notification ASAP, avoid delays.
			go func() {
				if err := b.transaction(func(tx *sqlx.Tx) error {
					sessionCopy := session
					sessionCopy.Status = MatchSessionStatusInProgress
					return b.sendSessionStatusUpdateNotification(tx, sessionCopy)
				}); err != nil {
					log.Printf("error: unable to send status update: %s", err)
				}
			}()

			break
		}

		for _, v := range countdowns {
			if v < min && v >= delta {
				if err := b.transaction(func(tx *sqlx.Tx) error {
					return b.sendSessionCountdownNotification(tx, session)
				}); err != nil {
					log.Printf("error: unable to send countdown notification: %s", err)
				}

				min = v
				break
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	if err := b.transaction(func(tx *sqlx.Tx) error {
		if err := session.start(tx); err != nil {
			return err
		}

		if err := session.update(tx); err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.Printf("error: unable to start MatchSession %s", err)
	}
}

func getMatchSessionsToStart(tx *sqlx.Tx) ([]MatchSession, error) {
	query := `SELECT * FROM MatchSession
    WHERE DATETIME(StartDate) <= DATETIME(?) AND Status = ?`
	var sessions []MatchSession
	if err := tx.Select(
		&sessions, query,
		// ensure we can start notifying at exactly T-60s by using 1.5× the update rate
		util.TimeAsDateTimeTZ(time.Now().Add(90*time.Second)),
		MatchSessionStatusPreparing,
	); err != nil {
		return nil, err
	}

	return sessions, nil
}

// getMatchSessionsToEnd returns all non-closed sessions that are both stale
// (we are past their StartDate) and have no matches in progress.
func getMatchSessionsToEnd(tx *sqlx.Tx) ([]MatchSession, map[util.UUIDAsBlob][]Match, error) {
	query := `
        SELECT MatchSession.* FROM MatchSession
        WHERE DATETIME(MatchSession.StartDate) < DATETIME(?)
              AND MatchSession.Status != ?`
	var sessions []MatchSession
	if err := tx.Select(
		&sessions, query,
		util.TimeAsDateTimeTZ(time.Now()),
		MatchSessionStatusClosed,
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

		// A stale session with no match should only happen in dev.
		if len(matches) == 0 {
			log.Printf("warning: closing stale session with no match:  %s", sessions[k].ID)
			ret = append(ret, sessions[k])
			continue
		}

		for j := range matches {
			if !matches[j].HasEnded() {
				continue loop
			}
		}

		ret = append(ret, sessions[k])
		sessMatches[sessions[k].ID] = matches
	}

	return ret, sessMatches, nil
}

func (b *Back) closeMatchSessionsAndUpdateRanks() error {
	var sessions []MatchSession
	var matches map[util.UUIDAsBlob][]Match

	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		sessions, matches, err = getMatchSessionsToEnd(tx)
		if err != nil {
			return err
		}

		for k := range sessions {
			sessions[k].Status = MatchSessionStatusClosed
			if err := sessions[k].update(tx); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		now := time.Now()
		for k := range sessions {
			if err := b.updateLeagueRankings(tx, sessions[k].LeagueID, now); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	// In a separate transaction to avoid delaying ranking updates and working with stale data.
	return b.transaction(func(tx *sqlx.Tx) error {
		for k := range sessions {
			if err := b.sendSessionStatusUpdateNotification(tx, sessions[k]); err != nil {
				return err
			}

			if err := b.sendSessionRecapNotification(
				tx, sessions[k], matches[sessions[k].ID],
				RecapScopePublic, nil,
			); err != nil {
				return err
			}

			if err := b.sendLeaderboardUpdateNotification(tx, sessions[k].LeagueID); err != nil {
				return err
			}
		}

		return nil
	})
}

// maybeUnlockSpoilerLogs tells ootrandomizer.com to unlock the spoiler log.
func (b *Back) maybeUnlockSpoilerLogs(match Match) error {
	gen, err := b.generatorFactory.NewGenerator(match.Generator)
	if err != nil {
		return err
	}

	if err := gen.UnlockSpoilerLog(match.GeneratorState); err != nil {
		return err
	}

	return nil
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
