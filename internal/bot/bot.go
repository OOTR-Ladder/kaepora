package bot

import (
	"context"
	"errors"
	"fmt"
	"kaepora/internal/back"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	back        *back.Back
	token       string
	dg          *discordgo.Session
	adminUserID string

	closed bool
	closer chan<- struct{}
}

func New(back *back.Back, token string, closer chan<- struct{}) (*Bot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		back:        back,
		closer:      closer,
		adminUserID: os.Getenv("KAEPORA_ADMIN_USER"),
		token:       token,
		dg:          dg,
	}

	dg.AddHandler(bot.handleMessage)

	return bot, nil
}

func (bot *Bot) Serve() {
	if bot.closed {
		log.Panic("attempted to serve closed bot")
		return
	}

	log.Println("starting Discord bot")

	if err := bot.dg.Open(); err != nil {
		log.Panic(err)
	}
}

func (bot *Bot) Close() error {
	if bot.closed { // don't close twice
		return nil
	}

	log.Println("closing Discord bot")

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	close(bot.closer)
	bot.closed = true

	if err := bot.dg.Close(); err != nil {
		return err
	}

	return nil
}

func (bot *Bot) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	if !strings.HasPrefix(m.Content, "!") {
		return
	}

	log.Printf(
		"<%s(%s)@%s#%s> %s",
		m.Author.String(),
		m.Author.ID,
		m.GuildID, m.ChannelID,
		m.Content,
	)

	defer func() {
		r := recover()
		if r != nil {
			log.Print("panic: ", r)
		}
	}()

	if err := bot.dispatch(s, m.Message); err != nil {
		msg := fmt.Sprintf("%s There was an error processing your command.", m.Author.Mention())

		if errors.Is(err, errPublic("")) {
			msg = fmt.Sprintf("%s\n```%s\n```\nIf you need help, send `!help`.", msg, err)
		} else {
			msg = fmt.Sprintf("%s\n<@%s> will check the logs when he has time.", msg, bot.adminUserID)
		}

		_, _ = s.ChannelMessageSend(m.ChannelID, msg)

		log.Printf("error: %s", err)
	}
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

func (bot *Bot) dispatch(s *discordgo.Session, m *discordgo.Message) error {
	command, args := parseCommand(m.Content)

	switch command { // nolint:gocritic, TODO
	case "!help":
		_, err := s.ChannelMessageSend(m.ChannelID, help())
		return err
	case "!dev":
		return bot.dispatchDev(s, m, args)
	case "!games":
		return bot.dispatchGames(s, m, args)
	case "!leagues":
		return bot.dispatchLeagues(s, m, args)
	case "!self":
		return bot.dispatchSelf(s, m, args)
	default:
		return errPublic(fmt.Sprintf("invalid command: %v", m.Content))
	}
}

func (bot *Bot) dispatchDev(_ *discordgo.Session, m *discordgo.Message, args []string) error {
	if m.Author.ID != bot.adminUserID {
		return fmt.Errorf("!dev command ran by a non-admin: %v", args)
	}

	if len(args) < 1 {
		return fmt.Errorf("error: !dev command has no arguments")
	}

	switch args[0] { // nolint:gocritic, TODO
	case "down":
		bot.Close()
	case "panic":
		panic("an admin asked me to panic")
	}

	return nil
}

func help() string {
	return strings.ReplaceAll(`Available commands:
'''
!games                # list games
!help                 # display this help message
!leagues              # list leagues
!self name NAME       # set your display name to NAME
!self register        # create your account and link it to your Discord account
'''`, "'''", "```")
}
