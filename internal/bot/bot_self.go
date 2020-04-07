package bot

import (
	"fmt"
	"io"
	"kaepora/internal/back"
	"kaepora/internal/util"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) cmdRename(m *discordgo.Message, args []string, out io.Writer) error {
	if len(args) < 1 {
		return errPublic("your forgot to tell me your desired name")
	}

	name := strings.Trim(strings.Join(args, " "), "  \t\n")
	return bot.setDiscordPlayerName(m.Author.ID, name, out)
}

func (bot *Bot) setDiscordPlayerName(discordID string, name string, out io.Writer) error {
	player, err := bot.back.GetPlayerByDiscordID(discordID)
	if err != nil {
		return errPublic("you need to `!register` first")
	}
	if player.Name == name {
		return errPublic("that's your name already")
	}

	if len(name) < 3 || len(name) > 32 {
		return errPublic("your name must be between 3 and 32 characters")
	}

	if _, err := bot.back.GetPlayerByName(name); err == nil {
		return errPublic("this name is taken already")
	}

	player.Name = name
	if err := bot.back.UpdatePlayer(player); err != nil {
		return err
	}

	fmt.Fprintf(out, "You'll be henceforth known as `%s` on the leaderboards.", player.Name)
	return err
}

func (bot *Bot) cmdRegister(m *discordgo.Message, _ []string, out io.Writer) error {
	user := m.Author
	if _, err := bot.back.GetPlayerByDiscordID(user.ID); err == nil {
		return errPublic("you are already registered")
	}

	player := back.NewPlayer(user.Username)
	player.DiscordID = util.NullString(user.ID)
	if err := bot.back.RegisterPlayer(player); err != nil {
		return err
	}

	fmt.Fprintf(out, "You have been registered as `%s`, see you on the leaderboards.", player.Name)

	return nil
}
