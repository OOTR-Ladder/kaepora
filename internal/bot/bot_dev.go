package bot

import (
	"fmt"
	"io"
	"kaepora/internal/back"
	"kaepora/internal/util"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// nolint:funlen
func (bot *Bot) cmdDev(m *discordgo.Message, args []string, out io.Writer) error {
	if !bot.config.IsDiscordIDAdmin(m.Author.ID) {
		return fmt.Errorf("!dev command ran by a non-admin: %v", args)
	}
	if len(args) < 1 {
		fmt.Fprintf(out, `
**Admin-only commands**:
%[1]s
!dev as NAME PARAMS          # send bot command as another user
!dev to NAME PARAMS          # send message to another user via the bot
!dev error                   # error out
!dev panic                   # panic and abort
!dev rerank SHORTCODE        # erase and recompute all the ranking history for a league
!dev setannounce SHORTCODE   # configure a league to post its announcements in the channel the command was sent in
!dev uptime                  # display for how long the server has been running
!dev url                     # display the link to use when adding the bot to a new server
!dev token                   # get an authenticated URL to the website
%[1]s`,
			"```",
		)
		return nil
	}

	switch args[0] {
	case "token":
		return bot.cmdDevToken(m, args, out)
	case "as":
		return bot.cmdDevAs(m, args, out)
	case "to":
		return bot.cmdDevTo(m, args, out)
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
			discordgo.PermissionViewChannel|discordgo.PermissionSendMessages|
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
	for k, v := range bot.config.Discord.ListenIDs {
		if v == m.ChannelID {
			i = k
		}
	}

	if i < 0 {
		return util.ErrPublic("channel was not being listened on")
	}

	bot.config.Discord.ListenIDs = append(
		bot.config.Discord.ListenIDs[:i],
		bot.config.Discord.ListenIDs[i+1:]...,
	)

	return bot.config.Write()
}

func (bot *Bot) cmdDevAddListen(m *discordgo.Message, _ []string, _ io.Writer) (err error) {
	for _, v := range bot.config.Discord.ListenIDs {
		if v == m.ChannelID {
			return util.ErrPublic("channel is already being listened on")
		}
	}

	bot.config.Discord.ListenIDs = append(bot.config.Discord.ListenIDs, m.ChannelID)
	return bot.config.Write()
}

func (bot *Bot) cmdDevRerank(_ *discordgo.Message, args []string, _ io.Writer) error {
	shortcode := argsAsName(args[1:])
	return bot.back.Rerank(shortcode)
}

func (bot *Bot) cmdDevAs(m *discordgo.Message, args []string, _ io.Writer) error {
	if len(args) < 3 {
		return util.ErrPublic("expected a name and a command")
	}

	player, err := bot.back.GetPlayerByName(args[1])
	if err != nil {
		return util.ErrPublic("no player with this name")
	}

	m.Author.ID = player.DiscordID.String
	m.Content = strings.Join(args[2:], " ")

	bot.createWriterAndDispatch(bot.dg, m, m.Author.ID)

	return nil
}

func (bot *Bot) cmdDevTo(_ *discordgo.Message, args []string, _ io.Writer) error {
	if len(args) < 3 {
		return util.ErrPublic("expected a name and a message")
	}

	player, err := bot.back.GetPlayerByName(args[1])
	if err != nil {
		return util.ErrPublic("no player with this name")
	}

	message := strings.Join(args[2:], " ")
	out, err := newUserChannelWriter(bot.dg, player.DiscordID.String)
	if err != nil {
		log.Printf("error: could not create channel writer: %s", err)
	}

	fmt.Fprint(out, message)

	if err := out.Flush(); err != nil {
		log.Printf("error: could not send message: %s", err)
	}

	return nil
}

func (bot *Bot) cmdDevToken(m *discordgo.Message, _ []string, w io.Writer) error {
	player, err := bot.back.GetPlayerByDiscordID(m.Author.ID)
	if err != nil {
		return err
	}

	token, err := bot.back.CreateToken(player.ID, back.DefaultTokenLifetime)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "`?t=%s`", token)
	return nil
}
