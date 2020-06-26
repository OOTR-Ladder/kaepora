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
	if !bot.isAdmin(m.Author.ID) {
		return fmt.Errorf("!dev command ran by a non-admin: %v", args)
	}
	if len(args) < 1 {
		fmt.Fprintf(out, `
**Admin-only commands**:
%[1]s
!dev error                   # error out
!dev panic                   # panic and abort
!dev rerank SHORTCODE        # erase and recompute all the ranking history for a league
!dev setannounce SHORTCODE   # configure a league to post its announcements in the channel the command was sent in
!dev uptime                  # display for how long the server has been running
!dev url                     # display the link to use when adding the bot to a new server
%[1]s`,
			"```",
		)
		return nil
	}

	switch args[0] {
	case "panic":
		panic("an admin asked me to panic")
	case "uptime":
		fmt.Fprintf(out, "The bot has been online for %s", time.Since(bot.startedAt).Round(time.Second))
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
	case "addlisten":
		return bot.cmdDevAddListen(m, args, out)
	case "removelisten":
		return bot.cmdDevRemoveListen(m, args, out)
	case "rerank":
		return bot.cmdDevRerank(m, args, out)
	default:
		return util.ErrPublic("invalid command")
	}

	return nil
}

func (bot *Bot) cmdDevRemoveListen(m *discordgo.Message, _ []string, _ io.Writer) (err error) {
	i := -1
	for k, v := range bot.config.DiscordListenIDs {
		if v == m.ChannelID {
			i = k
		}
	}

	if i < 0 {
		return util.ErrPublic("channel was not being listened on")
	}

	bot.config.DiscordListenIDs = append(
		bot.config.DiscordListenIDs[:i],
		bot.config.DiscordListenIDs[i+1:]...,
	)

	return bot.config.Write()
}

func (bot *Bot) cmdDevAddListen(m *discordgo.Message, _ []string, _ io.Writer) (err error) {
	for _, v := range bot.config.DiscordListenIDs {
		if v == m.ChannelID {
			return util.ErrPublic("channel is already being listened on")
		}
	}

	bot.config.DiscordListenIDs = append(bot.config.DiscordListenIDs, m.ChannelID)
	return bot.config.Write()
}

func (bot *Bot) cmdDevRerank(_ *discordgo.Message, args []string, _ io.Writer) error {
	shortcode := argsAsName(args[1:])
	return bot.back.Rerank(shortcode)
}
