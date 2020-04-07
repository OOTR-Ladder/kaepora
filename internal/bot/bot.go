package bot

import (
	"errors"
	"fmt"
	"io"
	"kaepora/internal/back"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type commandHandler func(m *discordgo.Message, args []string, w io.Writer) error

type Bot struct {
	back *back.Back

	startedAt   time.Time
	token       string
	dg          *discordgo.Session
	adminUserID string

	handlers map[string]commandHandler
}

func New(back *back.Back, token string) (*Bot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		back:        back,
		adminUserID: os.Getenv("KAEPORA_ADMIN_USER"),
		token:       token,
		dg:          dg,
		startedAt:   time.Now(),
	}

	dg.AddHandler(bot.handleMessage)

	bot.handlers = map[string]commandHandler{
		"!dev":      bot.cmdDev,
		"!games":    bot.cmdGames,
		"!help":     bot.cmdHelp,
		"!leagues":  bot.cmdLeagues,
		"!register": bot.cmdRegister,
		"!rename":   bot.cmdRename,

		"!cancel":  bot.cmdCancel,
		"!forfeit": bot.cmdForfeit,
		"!join":    bot.cmdJoin,
		"!stop":    bot.cmdStop,
	}

	return bot, nil
}

func (bot *Bot) Serve(wg *sync.WaitGroup, done <-chan struct{}) {
	log.Println("info: starting Discord bot")
	wg.Add(1)
	defer wg.Done()
	if err := bot.dg.Open(); err != nil {
		log.Panic(err)
	}

	<-done

	if err := bot.dg.Close(); err != nil {
		log.Printf("error: could not close Discord bot: %s", err)
	}
}

func (bot *Bot) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore webooks, self, bots, non-commands.
	if m.Author == nil || m.Author.ID == s.State.User.ID ||
		m.Author.Bot || !strings.HasPrefix(m.Content, "!") {
		return
	}

	log.Printf(
		"info: <%s(%s)@%s#%s> %s",
		m.Author.String(), m.Author.ID,
		m.GuildID, m.ChannelID,
		m.Content,
	)

	out, err := newUserChannelWriter(s, m.Author)
	if err != nil {
		log.Printf("error: could not create channel writer: %s", err)
	}
	defer func() {
		if err := out.Flush(); err != nil {
			log.Printf("error: could not send message: %s", err)
		}
	}()

	defer func() {
		r := recover()
		if r != nil {
			out.Reset()
			fmt.Fprintf(out, "Someting went very wrong, please tell <@%s>.", bot.adminUserID)
			log.Print("panic: ", r)
			log.Print(debug.Stack())
		}
	}()

	if err := bot.dispatch(m.Message, out); err != nil {
		out.Reset()
		fmt.Fprintln(out, "There was an error processing your command.")

		if errors.Is(err, errPublic("")) {
			fmt.Fprintf(out, "```%s\n```\nIf you need help, send `!help`.", err)
		} else {
			fmt.Fprintf(out, "<@%s> will check the logs when he has time.", bot.adminUserID)
		}

		log.Printf("error: failed to process command: %s", err)
	}

	if err := bot.maybeCleanupMessage(s, m.ChannelID, m.Message.ID); err != nil {
		log.Printf("error: unable to cleanup message: %s", err)
	}
}

func (bot *Bot) maybeCleanupMessage(s *discordgo.Session, channelID string, messageID string) error {
	channel, err := s.Channel(channelID)
	if err != nil {
		return err
	}

	if channel.Type != discordgo.ChannelTypeGuildText {
		return nil
	}

	if err := s.ChannelMessageDelete(channelID, messageID); err != nil {
		log.Printf("error: unable to delete message: %s", err)
	}

	return nil
}

func parseCommand(cmd string) (string, []string) {
	parts := strings.Split(cmd, " ")

	switch len(parts) {
	case 0:
		return "", nil
	case 1:
		return parts[0], nil
	default:
		return parts[0], parts[1:]
	}
}

func (bot *Bot) dispatch(m *discordgo.Message, w io.Writer) error {
	command, args := parseCommand(m.Content)
	handler, ok := bot.handlers[command]
	if !ok {
		return errPublic(fmt.Sprintf("invalid command: %v", m.Content))
	}

	return handler(m, args, w)
}

func (bot *Bot) cmdHelp(m *discordgo.Message, _ []string, w io.Writer) error {
	// TODO hardcoded delays in doc
	fmt.Fprint(w, strings.ReplaceAll(`Available commands:
'''
# Management
!games             # list games
!help              # display this help message
!leagues           # list leagues
!register          # create your account and link it to your Discord account
!rename NAME       # set your display name to NAME

# Racing
!cancel            # cancel joining the next race without penalty until T-30m
!forfeit           # forfeit (and thus lose) the current race
!join SHORTCODE    # join the next race of the given league (see !leagues)
!stop              # stop your race timer and register your final time
'''`, "'''", "```"))

	if m.Author.ID != bot.adminUserID {
		return nil
	}

	fmt.Fprint(w, strings.ReplaceAll(`Admin-only commands:
'''
!dev error     error out
!dev panic     panic and abort
!dev uptime    display for how long the server has been running
!dev url       display the link to use when adding the bot to a new server
'''`, "'''", "```"))

	return nil
}
