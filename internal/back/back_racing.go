package back

import (
	"database/sql"
	"errors"
	"fmt"
	"kaepora/internal/util"
	"log"

	"github.com/jmoiron/sqlx"
)

// JoinCurrentMatchSessionByShortcode adds a player to the currently joinable
// session of a league. It returns the session and league that were joined.
func (b *Back) JoinCurrentMatchSessionByShortcode(player Player, shortcode string) (
	session MatchSession,
	league League,
	_ error,
) {
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		league, err = getLeagueByShortCode(tx, shortcode)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return util.ErrPublic("could not find a league with this shortcode, try `!leagues`")
			}
			return err
		}

		session, err = joinCurrentMatchSessionTx(tx, player, league)
		return err
	}); err != nil {
		return MatchSession{}, League{}, err
	}

	return session, league, nil
}

func (b *Back) JoinMatchSessionByID(sessionID, playerID util.UUIDAsBlob) error {
	return b.transaction(func(tx *sqlx.Tx) error {
		session, err := getMatchSessionByID(tx, sessionID)
		if err != nil {
			return err
		}

		league, err := getLeagueByID(tx, session.LeagueID)
		if err != nil {
			return err
		}

		return joinMatchSessionTx(tx, session, playerID, league)
	})
}

func joinCurrentMatchSessionTx(
	tx *sqlx.Tx, player Player, league League,
) (MatchSession, error) {
	session, err := getNextJoinableMatchSessionForLeague(tx, league.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MatchSession{},
				util.ErrPublic("could not find a joinable race for the given league")
		}
		return MatchSession{}, err
	}

	if err := joinMatchSessionTx(tx, session, player.ID, league); err != nil {
		return MatchSession{}, err
	}

	return session, nil
}

func joinMatchSessionTx(
	tx *sqlx.Tx,
	session MatchSession,
	playerID util.UUIDAsBlob,
	league League,
) error {
	if session.HasPlayerID(playerID.UUID()) {
		return util.ErrPublic(fmt.Sprintf(
			"you are already registered for the next %s race", league.Name,
		))
	}

	if err := ensurePlayerHasNoActiveMatch(tx, playerID); err != nil {
		return err
	}

	session.AddPlayerID(playerID.UUID())
	if err := session.update(tx); err != nil {
		return err
	}

	return nil
}

// ensurePlayerHasNoActiveMatch returns an error if the player is currently in
// an active race (ie. he did not start or did not complete the race).
func ensurePlayerHasNoActiveMatch(tx *sqlx.Tx, playerID util.UUIDAsBlob) error {
	_, err := getPlayerActiveSession(tx, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	return util.ErrPublic("you already have a race in progress")
}

// CompleteActiveMatch ends the timer of the current player match entry.
func (b *Back) CompleteActiveMatch(player Player) (Match, error) {
	return b.endActiveMatch(player, false)
}

// ForfeitActiveMatch forfeits the current player match entry.
func (b *Back) ForfeitActiveMatch(player Player) (Match, error) {
	return b.endActiveMatch(player, true)
}

func (b *Back) endActiveMatch(player Player, forfeit bool) (Match, error) {
	var ret Match
	if err := b.transaction(func(tx *sqlx.Tx) error {
		match, self, against, err := getActiveMatchAndEntriesForPlayer(tx, player)
		if err != nil {
			return err
		}

		if forfeit {
			self.forfeit(&against, &match)
		} else {
			if self.Status != MatchEntryStatusInProgress {
				return util.ErrPublic("you can't complete a race that has not started")
			}

			self.complete(&against, &match)
		}

		if err := util.ConcatErrors([]error{
			self.update(tx),
			against.update(tx),
			match.update(tx),
			b.maybeSendMatchEndNotifications(tx, player, self, against, against.PlayerID),
		}); err != nil {
			return err
		}

		ret = match
		return nil
	}); err != nil {
		return Match{}, err
	}

	if err := b.sendPrivateRecapForSessionID(ret.MatchSessionID, player); err != nil {
		return Match{}, err
	}
	b.sendSpoilerLogNotification(player, ret.ID)

	if ret.HasEnded() {
		go func() {
			if err := b.maybeUnlockSpoilerLogs(ret); err != nil {
				log.Printf("error: unable to unlock spoiler log: %s", err)
			}
		}()
	}

	return ret, nil
}

func (b *Back) sendPrivateRecapForSessionID(sessionID util.UUIDAsBlob, player Player) error {
	return b.transaction(func(tx *sqlx.Tx) error {
		session, err := getMatchSessionByID(tx, sessionID)
		if err != nil {
			return err
		}

		matches, err := getMatchesBySessionID(tx, sessionID)
		if err != nil {
			return err
		}

		return b.sendSessionRecapNotification(
			tx, session, matches,
			RecapScopeRunner, &player.DiscordID.String,
		)
	})
}

// CancelActiveMatchSession removes the player from the currently joinable
// session, it cannot be called if the session has begun its preparation
// phase.
func (b *Back) CancelActiveMatchSession(playerID util.UUIDAsBlob) (MatchSession, error) {
	var ret MatchSession

	if err := b.transaction(func(tx *sqlx.Tx) error {
		session, err := getPlayerActiveSession(tx, playerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return util.ErrPublic("you are not in any active race right now")
			}
			return err
		}

		if err := session.canCancel(); err != nil {
			return err
		}

		session.RemovePlayerID(playerID.UUID())
		if err := session.update(tx); err != nil {
			return err
		}

		ret = session
		return nil
	}); err != nil {
		return MatchSession{}, err
	}

	return ret, nil
}

func (b *Back) maybeSendMatchEndNotifications(
	tx *sqlx.Tx,
	player Player,
	selfEntry MatchEntry, againstEntry MatchEntry,
	opponentID util.UUIDAsBlob,
) error {
	if selfEntry.HasEnded() {
		if err := b.sendMatchEndNotification(tx, selfEntry, againstEntry, player); err != nil {
			return err
		}
	}

	if againstEntry.HasEnded() {
		opponent, err := getPlayerByID(tx, opponentID)
		if err != nil {
			return err
		}

		if err := b.sendMatchEndNotification(tx, againstEntry, selfEntry, opponent); err != nil {
			return err
		}
	}

	return nil
}
