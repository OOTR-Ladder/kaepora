package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) dispatchGames(s *discordgo.Session, m *discordgo.Message, args []string) error {
	if len(args) > 0 {
		return errPublic("this command takes no argument")
	}

	return bot.displayGames(s, m.ChannelID)
}

func (bot *Bot) displayGames(s *discordgo.Session, channelID string) error {
	games, err := bot.back.GetGames()
	if err != nil {
		return err
	}

	if len(games) == 0 {
		_, err := s.ChannelMessageSend(channelID, "There is no registered game yet.")
		return err
	}

	var buf strings.Builder

	buf.WriteString("Here are the available games:\n\n")
	for k, v := range games {
		fmt.Fprintf(&buf, "%d. %s\n", k+1, v.Name)
	}

	_, err = s.ChannelMessageSend(channelID, buf.String())
	return err
}
