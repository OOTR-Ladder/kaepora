package bot

import (
	"fmt"
	"io"
	"kaepora/internal/util"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) cmdRename(m *discordgo.Message, args []string, out io.Writer) error {
	if len(args) < 1 {
		return util.ErrPublic("your forgot to tell me your desired name")
	}

	name := strings.Trim(strings.Join(args, " "), "  \t\n")
	return bot.setDiscordPlayerName(m.Author.ID, name, out)
}

func (bot *Bot) setDiscordPlayerName(discordID string, name string, out io.Writer) error {
	if err := bot.back.UpdateDiscordPlayerName(discordID, name); err != nil {
		return err
	}

	fmt.Fprintf(out, "You'll be henceforth known as `%s` on the leaderboards.", name)
	return nil
}

func (bot *Bot) cmdRegister(m *discordgo.Message, args []string, out io.Writer) error {
	name := strings.Trim(strings.Join(args, " "), "  \t\n")
	if name == "" {
		name = m.Author.Username
	}

	if err := bot.back.RegisterDiscordPlayer(m.Author.ID, name); err != nil {
		return err
	}

	fmt.Fprintf(out, "You have been registered as `%s`, see you on the leaderboards.", name)
	return nil
}
