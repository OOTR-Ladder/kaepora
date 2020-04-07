package bot

import (
	"fmt"
	"io"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) cmdDev(m *discordgo.Message, args []string, out io.Writer) error {
	if m.Author.ID != bot.adminUserID {
		return fmt.Errorf("!dev command ran by a non-admin: %v", args)
	}

	if len(args) < 1 {
		return errPublic("need a subcommand")
	}

	switch args[0] { // nolint:gocritic, TODO
	case "panic":
		panic("an admin asked me to panic")
	case "uptime":
		fmt.Fprintf(out, "The bot has been online for %s", time.Since(bot.startedAt))
	case "error":
		return errPublic("here's your error")
	case "url":
		fmt.Fprintf(
			out,
			"https://discordapp.com/api/oauth2/authorize?client_id=%s&scope=bot&permissions=%d",
			bot.dg.State.User.ID,
			discordgo.PermissionReadMessages|discordgo.PermissionSendMessages|
				discordgo.PermissionEmbedLinks|discordgo.PermissionAttachFiles|
				discordgo.PermissionManageMessages|discordgo.PermissionMentionEveryone,
		)
	}

	return nil
}
