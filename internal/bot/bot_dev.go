package bot

import (
	"fmt"
	"io"
	"kaepora/internal/util"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) cmdDev(m *discordgo.Message, args []string, out io.Writer) error {
	if m.Author.ID != bot.adminUserID {
		return fmt.Errorf("!dev command ran by a non-admin: %v", args)
	}
	if len(args) < 1 {
		return util.ErrPublic("need a subcommand")
	}

	switch args[0] { // nolint:gocritic, TODO
	case "panic":
		panic("an admin asked me to panic")
	case "uptime":
		fmt.Fprintf(out, "The bot has been online for %s", time.Since(bot.startedAt))
	case "error":
		return util.ErrPublic("here's your error")
	case "url":
		fmt.Fprintf(
			out,
			"https://discordapp.com/api/oauth2/authorize?client_id=%s&scope=bot&permissions=%d",
			bot.dg.State.User.ID,
			discordgo.PermissionReadMessages|discordgo.PermissionSendMessages|
				discordgo.PermissionEmbedLinks|discordgo.PermissionAttachFiles|
				discordgo.PermissionManageMessages|discordgo.PermissionMentionEveryone,
		)
	case "setannounce": // SHORTCODE
		shortcode := strings.Join(args[1:], " ")
		err := bot.back.SetLeagueAnnounceChannel(shortcode, m.ChannelID)
		if err != nil {
			return err
		}

		channel := newChannelWriter(bot.dg, m.ChannelID)
		defer channel.Flush()
		fmt.Fprintf(channel, "Announcements for league `%s` now will now happen in this channel.", shortcode)
	case "seed": // SHORTCODE SEED
		if len(args) != 3 {
			return util.ErrPublic("expected 2 arguments: SHORTCODE SEED")
		}
		return bot.back.SendDevSeed(m.Author.ID, args[1], args[2])
	}

	return nil
}
