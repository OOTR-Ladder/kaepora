package bot

import (
	"fmt"
	"kaepora/internal/back"
	"kaepora/internal/util"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) dispatchSelf(s *discordgo.Session, m *discordgo.Message, args []string) error {
	if len(args) < 1 {
		return errPublic("need a subcommand")
	}

	command := args[0]
	args = args[1:]

	switch command {
	case "register":
		return bot.registerDiscordPlayer(s, m.Author, m.ChannelID)
	case "name":
		if len(args) < 1 {
			return errPublic("your forgot to tell me your name")
		}
		return bot.setDiscordPlayerName(
			s,
			m.Author.ID,
			strings.Trim(strings.Join(args, " "), "  \t\n"),
			m.ChannelID,
		)
	default:
		return errPublic("bad subcommand")
	}
}

func (bot *Bot) setDiscordPlayerName(s *discordgo.Session, discordID string, name string, channelID string) error {
	player, err := bot.back.GetPlayerByDiscordID(discordID)
	if err != nil {
		return errPublic("you need to register first")
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

	_, err = s.ChannelMessageSend(channelID, fmt.Sprintf(
		"You'll be henceforth known as `%s` on the leaderboards.",
		player.Name,
	))
	return err
}

func (bot *Bot) registerDiscordPlayer(s *discordgo.Session, user *discordgo.User, channelID string) error {
	if _, err := bot.back.GetPlayerByDiscordID(user.ID); err == nil {
		return errPublic("you are already registered")
	}

	player := back.NewPlayer(user.Username)
	player.DiscordID = util.NullString(user.ID)
	if err := bot.back.RegisterPlayer(player); err != nil {
		return err
	}

	_, err := s.ChannelMessageSend(channelID, fmt.Sprintf(
		"You have been registered as `%s`, see you on the leaderboards.",
		player.Name,
	))
	return err
}
