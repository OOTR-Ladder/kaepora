package bot

import (
	"context"
	"kaepora/internal/back"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	back  *back.Back
	token string
	dg    *discordgo.Session

	closed bool
	closer chan<- struct{}
}

func New(back *back.Back, token string, closer chan<- struct{}) (*Bot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		back:   back,
		closer: closer,
		token:  token,
		dg:     dg,
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
	if m.Author == nil || m.Author.ID == s.State.User.ID {
		return
	}

	if !strings.HasPrefix(m.Content, "!") {
		return
	}

	log.Printf(
		"<%s#%s(%s)@%s#%s> %s",
		m.Author.Username, m.Author.Discriminator,
		m.Author.ID,
		m.GuildID, m.ChannelID,
		m.Content,
	)

	bot.dispatch(s, m.Message)
}

func (bot *Bot) dispatch(s *discordgo.Session, m *discordgo.Message) {
	command := strings.SplitN(m.Content, " ", 2)
	switch command[0] { // nolint:gocritic, TODO
	case "!dev":
		bot.dispatchDev(s, m, command[1:])
	default:
		log.Printf("error: invalid command: %v", m.Content)
	}
}

func (bot *Bot) dispatchDev(_ *discordgo.Session, m *discordgo.Message, args []string) {
	if m.Author.ID != os.Getenv("KAEPORA_ADMIN_USER") {
		log.Printf("error: !dev command ran by a non-admin: %v", args)
		return
	}

	if len(args) < 1 {
		log.Printf("error: !dev command has no arguments")
		return
	}

	switch args[0] { // nolint:gocritic, TODO
	case "down":
		bot.Close()
	}
}
