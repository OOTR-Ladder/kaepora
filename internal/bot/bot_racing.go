package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"kaepora/internal/back"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) cmdJoin(m *discordgo.Message, args []string, w io.Writer) error {
	player, err := bot.back.GetPlayerByDiscordID(m.Author.ID)
	if err != nil {
		return errPublic("you need to `!register` first")
	}

	session, league, err := bot.getCurJoinable(strings.Join(args, " "))
	if err != nil {
		return fmt.Errorf("could not fetch joinable MatchSession: %s", err)
	}

	if session.HasPlayerID(player.ID.UUID()) {
		return errPublic(fmt.Sprintf("you are already registered for the next %s race", league.Name))
	}

	if active, err := bot.back.GetPlayerActiveSession(player.ID.UUID()); err == nil {
		activeLeague, err := bot.back.GetLeagueByID(active.LeagueID)
		if err != nil {
			return fmt.Errorf("could not fetch active league: %s", err)
		}
		if active.ID != session.ID {
			return errPublic(fmt.Sprintf("you are already registered for another race on the %s league", activeLeague.Name))
		}
	}

	session.AddPlayerID(player.ID.UUID())
	if err := bot.back.UpdateMatchSession(session); err != nil {
		return fmt.Errorf("could not update MatchSession: %s", err)
	}

	fmt.Fprintf(w, "You have been registered for the next race in the %s league.\n", league.Name)

	cancelDelta := time.Until(session.StartDate.Time().Add(back.MatchSessionCancellableUntilOffset))
	if cancelDelta > 0 {
		fmt.Fprintf(
			w,
			"If you wish to `!cancel` you have %s to do so, after that you will have to `!forfeit`.",
			cancelDelta.Truncate(time.Second),
		)
	} else {
		raceDelta := time.Until(session.StartDate.Time())
		fmt.Fprintf(w,
			"The race begins in %s, you will soon receive your _seed_ details.",
			raceDelta.Truncate(time.Second),
		)
	}

	return nil
}

func (bot *Bot) cmdCancel(m *discordgo.Message, _ []string, w io.Writer) error {
	player, err := bot.back.GetPlayerByDiscordID(m.Author.ID)
	if err != nil {
		return errPublic("you need to `!register` first")
	}

	session, err := bot.back.GetPlayerActiveSession(player.ID.UUID())
	if err != nil {
		if err == sql.ErrNoRows {
			return errPublic("you are not in any active race right now")
		}
		return err
	}

	if !session.CanCancel() {
		return errPublic("you can't cancel your current race, you either have to `!forfeit` or  `!complete` it.")
	}

	league, err := bot.back.GetLeagueByID(session.LeagueID)
	if err != nil {
		return err
	}

	session.RemovePlayerID(player.ID.UUID())
	if err := bot.back.UpdateMatchSession(session); err != nil {
		return err
	}

	fmt.Fprintf(w, `You have cancelled your participation for the next race in the %[1]s league.
This _will not_ count as a loss and won't affect your rankings.`,
		league.Name,
	)

	return nil
}

func (bot *Bot) cmdStop(_ *discordgo.Message, _ []string, w io.Writer) error {
	return errPublic("not implemented")
}

func (bot *Bot) cmdForfeit(_ *discordgo.Message, _ []string, w io.Writer) error {
	return errPublic("not implemented")
}

func (bot *Bot) getCurJoinable(leagueShortCode string) (back.MatchSession, back.League, error) {
	league, err := bot.back.GetLeagueByShortcode(leagueShortCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return back.MatchSession{}, back.League{},
				errPublic("invalid short code, try `!leagues`")
		}
		return back.MatchSession{}, back.League{}, err
	}

	session, err := bot.back.GetNextJoinableMatchSessionForLeague(league.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return back.MatchSession{}, back.League{},
				errPublic(fmt.Sprintf("could not find a joinable race for the %s league", league.Name))
		}
		return back.MatchSession{}, back.League{}, err
	}

	return session, league, nil
}
