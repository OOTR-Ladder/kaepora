package bot

import (
	"errors"
	"fmt"
	"io"
	"kaepora/internal/back"
	"kaepora/internal/util"
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

	handlers      map[string]commandHandler
	notifications <-chan back.Notification
}

func New(back *back.Back, token string) (*Bot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		back:          back,
		adminUserID:   os.Getenv("KAEPORA_ADMIN_USER"),
		token:         token,
		dg:            dg,
		startedAt:     time.Now(),
		notifications: back.GetNotificationsChan(),
	}

	dg.AddHandler(bot.handleMessage)

	bot.handlers = map[string]commandHandler{
		"!dev":          bot.cmdDev,
		"!help":         bot.cmdHelp,
		"!no":           bot.cmdHelp,
		"!yes":          bot.cmdAllRight,
		"!leagues":      bot.cmdLeagues,
		"!leaderboard":  bot.cmdLeaderboards,
		"!leaderboards": bot.cmdLeaderboards,
		"!register":     bot.cmdRegister,
		"!rename":       bot.cmdRename,
		"!spoilers":     bot.cmdSpoilers,

		"!cancel":   bot.cmdCancel,
		"!done":     bot.cmdComplete,
		"!complete": bot.cmdComplete,
		"!forfeit":  bot.cmdForfeit,
		"!join":     bot.cmdJoin,

		"!rando": bot.cmdDevRandomSettings, // DEBUG
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

loop:
	for {
		select {
		case notif := <-bot.notifications:
			if err := bot.sendNotification(notif); err != nil {
				log.Printf("unable to send notification: %s", err)
			}
		case <-done:
			break loop
		}
	}

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

	out, err := newUserChannelWriter(s, m.Author.ID)
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
			log.Printf("%s", debug.Stack())
		}
	}()

	if err := bot.maybeCleanupMessage(s, m.ChannelID, m.Message.ID); err != nil {
		log.Printf("error: unable to cleanup message: %s", err)
	}

	if err := bot.dispatch(m.Message, out); err != nil {
		out.Reset()
		fmt.Fprintln(out, "There was an error processing your command.")

		if errors.Is(err, util.ErrPublic("")) {
			fmt.Fprintf(out, "```%s\n```\nIf you need help, send `!help`.", err)
		} else {
			fmt.Fprintf(out, "<@%s> will check the logs when he has time.", bot.adminUserID)
		}

		log.Printf("error: failed to process command: %s", err)
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
		return util.ErrPublic(fmt.Sprintf("invalid command: %v", m.Content))
	}

	return handler(m, args, w)
}

// nolint:funlen
func (bot *Bot) cmdHelp(m *discordgo.Message, _ []string, w io.Writer) error {
	truncate := func(v time.Duration) string {
		ret := strings.TrimSuffix(v.Truncate(time.Second).String(), "0s")
		if strings.HasSuffix(ret, "h0m") {
			return strings.TrimSuffix(ret, "0m")
		}

		return ret
	}
	joinOffset := truncate(back.MatchSessionJoinableAfterOffset)
	prepOffset := truncate(back.MatchSessionPreparationOffset)

	fmt.Fprintf(w, "Hoo hoot! %s… Look up here!\n"+
		"It appears that the time has finally come for you to start your adventure!\n"+
		"You will encounter many hardships ahead… That is your fate.\n"+
		"Don't feel discouraged, even during the toughest times!\n\n",
		m.Author.Mention(),
	)

	// nolint:lll
	fmt.Fprintf(w, `**Available commands**:
%[1]s
# Management
!help                   # display this help message
!leaderboard SHORTCODE  # show leaderboards for the given league
!leagues                # list leagues
!register               # create your account and link it to your Discord account
!register NAME          # same as "!register" but use another name
!rename NAME            # set your display name to NAME

# Racing
!cancel            # cancel joining the next race without penalty until T%[3]s
!done              # stop your race timer and register your final time
!forfeit           # forfeit (and thus lose) the current race
!join SHORTCODE    # join the next race of the given league (see !leagues)
!spoilers SEED     # send the spoiler log for the given seed (if the corresponding race has finished)
%[1]s

**Racing**:
You can freely join a race and cancel without consequences between T%[2]s and T%[3]s.
When the race reaches its preparation phase at T%[3]s you can no longer cancel and must either complete or forfeit the race.
You can't join a race that is in progress or has begun its preparation phase (T%[3]s).
If you are caught cheating, using an alt, or breaking a league's rules **you will be banned**.

Did you get all that?
`,
		"```",
		joinOffset,
		prepOffset,
	)

	return nil
}

func argsAsName(args []string) string {
	return strings.Trim(strings.Join(args, " "), "  \t\n")
}

func (bot *Bot) cmdAllRight(m *discordgo.Message, _ []string, w io.Writer) error {
	fmt.Fprintf(w, "All right then, I'll see you around!\nHoot hoot hoot ho!")
	return nil
}
