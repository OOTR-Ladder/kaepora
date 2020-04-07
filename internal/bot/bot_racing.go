package bot

import (
	"database/sql"
	"errors"
	"io"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) cmdJoin(m *discordgo.Message, args []string, w io.Writer) error {
	player, err := bot.back.GetPlayerByDiscordID(m.Author.ID)
	if err != nil {
		return errPublic("you need to `!register` first")
	}

	league, err := bot.back.GetLeagueByShortcode(strings.Join(args[1:], " "))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errPublic("invalid short code, try `!leagues`")
		}
		return err
	}

	session, err := bot.back.GetNextJoinableMatchSessionForLeague(league.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errPublic("could not find a scheduled race")
		}
		return err
	}

	session.AddPlayerID(player.ID.UUID())
	return bot.back.UpdateMatchSession(session)
}

func (bot *Bot) cmdCancel(_ *discordgo.Message, _ []string, w io.Writer) error {
	return errPublic("not implemented")
}

func (bot *Bot) cmdStop(_ *discordgo.Message, _ []string, w io.Writer) error {
	return errPublic("not implemented")
}

func (bot *Bot) cmdForfeit(_ *discordgo.Message, _ []string, w io.Writer) error {
	return errPublic("not implemented")
}
