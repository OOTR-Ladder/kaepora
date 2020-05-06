package bot

import (
	"fmt"
	"io"
	"kaepora/internal/generator/oot/settings"
	"kaepora/internal/util"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// nolint:funlen
func (bot *Bot) cmdDev(m *discordgo.Message, args []string, out io.Writer) error {
	if !bot.isAdmin(m.Author.ID) {
		return fmt.Errorf("!dev command ran by a non-admin: %v", args)
	}
	if len(args) < 1 {
		fmt.Fprintf(out, `
**Admin-only commands**:
%[1]s
!dev closesession            # close the debug race
!dev createsession SHORTCODE # create a new debug race starting immediately
!dev error                   # error out
!dev panic                   # panic and abort
!dev seed SHORTCODE [SEED]   # generate a seed valid for the given league
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
	case "seed": // SHORTCODE
		if len(args) < 2 || len(args) > 3 {
			return util.ErrPublic("expected 2 arguments: SHORTCODE [SEED]")
		}

		seed := uuid.New().String()
		if len(args) == 3 {
			seed = args[2]
		}
		return bot.back.SendDevSeed(m.Author.ID, args[1], seed)
	case "createsession": // SHORTCODE
		if len(args) != 2 {
			return util.ErrPublic("expected 1 argument: SHORTCODE")
		}
		return bot.back.CreateDevMatchSession(args[1])
	case "closesession":
		return bot.back.CloseDevMatchSession()
	default:
		return util.ErrPublic("invalid command")
	}

	return nil
}

// cmdDevRandomSettings is a temporary DEBUG command to demonstrate randomized settings
func (bot *Bot) cmdDevRandomSettings(m *discordgo.Message, args []string, w io.Writer) (err error) {
	cost := 20
	seed := uuid.New().String()

	if len(args) > 0 {
		cost, err = strconv.Atoi(args[0])
		if err != nil {
			return util.ErrPublic(err.Error())
		}
	}

	if len(args) > 1 {
		seed = args[2]
	}

	s, err := settings.Load("resources/oot-randomizer/" + settings.DefaultName)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Generated settings for seed %s and cost %d:\n```\n", seed, cost)
	shuffled := s.Shuffle(seed, cost)
	for k, v := range shuffled {
		fmt.Fprintf(w, "%s = %v\n", k, v)
	}
	fmt.Fprint(w, "\n```")

	return nil
}
