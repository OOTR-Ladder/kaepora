package bot

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) dispatchLeagues(s *discordgo.Session, m *discordgo.Message, args []string) error {
	switch len(args) {
	case 0:
		return bot.displayLeagues(s, m.ChannelID)
	default:
		return errPublic("bad arguments count")
	}
}

func (bot *Bot) displayLeagues(s *discordgo.Session, channelID string) error {
	games, err := bot.back.GetGames()
	if err != nil {
		return err
	}

	var buf strings.Builder

	for k, game := range games {
		fmt.Fprintf(&buf, "%d. Leagues for _%s_:\n", k+1, game.Name)

		leagues, err := bot.back.GetLeaguesByGameID(game.ID)
		if err != nil {
			return err
		}

		buf.WriteString("```\n")

		table := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)

		fmt.Fprintln(table, "shortcode\tname\tsettings")
		fmt.Fprintln(table, "\t\t")
		for _, league := range leagues {
			fmt.Fprintf(table, "%s\t%s\t%.64s\n", league.ShortCode, league.Name, league.Settings)
		}
		table.Flush()

		buf.WriteString("```\n")
	}

	_, err = s.ChannelMessageSend(channelID, buf.String())
	return err
}
