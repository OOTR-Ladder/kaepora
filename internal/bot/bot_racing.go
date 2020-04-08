package bot

import (
	"fmt"
	"io"
	"kaepora/internal/back"
	"kaepora/internal/util"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) cmdJoin(m *discordgo.Message, args []string, w io.Writer) error {
	player, err := bot.back.GetPlayerByDiscordID(m.Author.ID)
	if err != nil {
		return util.ErrPublic("you need to `!register` first")
	}

	shortcode := strings.Join(args, " ")
	session, league, err := bot.back.JoinCurrentMatchSessionByShortcode(player, shortcode)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "You have been registered for the next race in the %s league.\n", league.Name)

	cancelDelta := time.Until(session.StartDate.Time().Add(back.MatchSessionPreparationOffset))
	if cancelDelta > 0 {
		fmt.Fprintf(
			w,
			"If you wish to `!cancel` you have %s to do so, after that you will have to `!forfeit`.",
			cancelDelta.Truncate(time.Second),
		)
	} else { // maybe unreachable, maybe not.
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
		return util.ErrPublic("you need to `!register` first")
	}

	if _, err := bot.back.CancelActiveMatchSession(player); err != nil {
		return err
	}

	fmt.Fprint(w, `You have cancelled your participation for the next race.
This _will not_ count as a loss and won't affect your rankings.`)

	return nil
}

func (bot *Bot) cmdComplete(_ *discordgo.Message, _ []string, _ io.Writer) error {
	return util.ErrPublic("not implemented")
}

func (bot *Bot) cmdForfeit(_ *discordgo.Message, _ []string, _ io.Writer) error {
	return util.ErrPublic("not implemented")
}
