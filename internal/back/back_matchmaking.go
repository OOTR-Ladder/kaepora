package back

import (
	"database/sql"
	"errors"
	"fmt"
	"kaepora/internal/util"

	"github.com/jmoiron/sqlx"
)

func (b *Back) CancelActiveMatchSession(player Player) (MatchSession, error) {
	var ret MatchSession

	if err := b.transaction(func(tx *sqlx.Tx) error {
		session, err := getPlayerActiveSession(tx, player.ID.UUID())
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return util.ErrPublic("you are not in any active race right now")
			}
			return err
		}

		if err := session.CanCancel(); err != nil {
			return err
		}

		session.RemovePlayerID(player.ID.UUID())
		if err := session.Update(tx); err != nil {
			return err
		}

		ret = session
		return nil
	}); err != nil {
		return MatchSession{}, err
	}

	return ret, nil
}

func (b *Back) ForfeitActiveMatchSession(player Player) (MatchSession, error) {
	var session MatchSession
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		session, err = getPlayerActiveSession(tx, player.ID.UUID())
		if err != nil {
			if err == sql.ErrNoRows {
				return util.ErrPublic("you are not in any active race right now")
			}
			return err
		}

		if err := session.CanForfeit(); err != nil {
			return err
		}

		match, err := getMatchByPlayerAndSession(tx, player, session)
		if err != nil {
			return err
		}

		entry, err := match.GetPlayerEntry(player.ID)
		if err != nil {
			return err
		}

		entry.forfeit()
		if err := entry.update(tx); err != nil {
			return err
		}

		// TODO end match

		return util.ErrPublic("ForfeitActiveMatchSession not implemented")
	}); err != nil {
		return MatchSession{}, err
	}

	return session, nil
}

func (b *Back) JoinCurrentMatchSession(player Player, league League) (MatchSession, error) {
	var ret MatchSession
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		ret, err = joinCurrentMatchSessionTx(tx, player, league)
		return err
	}); err != nil {
		return MatchSession{}, err
	}

	return ret, nil
}

func (b *Back) JoinCurrentMatchSessionByShortcode(player Player, shortcode string) (
	session MatchSession,
	league League,
	_ error,
) {
	if err := b.transaction(func(tx *sqlx.Tx) (err error) {
		league, err = getLeagueByShortCode(tx, shortcode)
		if err != nil {
			return err
		}

		session, err = joinCurrentMatchSessionTx(tx, player, league)
		return err
	}); err != nil {
		return MatchSession{}, League{}, err
	}

	return session, league, nil
}

func joinCurrentMatchSessionTx(tx *sqlx.Tx, player Player, league League) (MatchSession, error) {
	session, err := getCurrentJoinableSession(tx, league.ID)
	if err != nil {
		return MatchSession{}, err
	}

	if session.HasPlayerID(player.ID.UUID()) {
		return MatchSession{}, util.ErrPublic(fmt.Sprintf(
			"you are already registered for the next %s race", league.Name,
		))
	}

	if active, err := getPlayerActiveSession(tx, player.ID.UUID()); err == nil {
		activeLeague, err := getLeagueByID(tx, active.LeagueID)
		if err != nil {
			return MatchSession{}, err
		}

		if active.ID != session.ID {
			return MatchSession{},
				util.ErrPublic(fmt.Sprintf(
					"you are already registered for another race on the %s league",
					activeLeague.Name,
				))
		}
	}

	session.AddPlayerID(player.ID.UUID())
	if err := session.Update(tx); err != nil {
		return MatchSession{}, err
	}

	return session, nil
}

func getCurrentJoinableSession(tx *sqlx.Tx, leagueID util.UUIDAsBlob) (MatchSession, error) {
	session, err := getNextJoinableMatchSessionForLeague(tx, leagueID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MatchSession{},
				util.ErrPublic("could not find a joinable race for the given league")
		}
		return MatchSession{}, err
	}

	return session, nil
}
