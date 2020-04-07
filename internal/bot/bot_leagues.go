package bot

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) dispatchLeagues(m *discordgo.Message, args []string, out io.Writer) error {
	switch len(args) {
	case 0:
		return bot.displayLeagues(out)
	default:
		return errPublic("bad arguments count")
	}
}

func (bot *Bot) displayLeagues(out io.Writer) error {
	games, err := bot.back.GetGames()
	if err != nil {
		return err
	}

	for k, game := range games {
		fmt.Fprintf(out, "%d. Leagues for _%s_:\n", k+1, game.Name)

		leagues, err := bot.back.GetLeaguesByGameID(game.ID)
		if err != nil {
			return err
		}

		fmt.Fprint(out, "```\n")

		table := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)

		fmt.Fprintln(table, "shortcode\tname\tsettings")
		fmt.Fprintln(table, "\t\t")
		for _, league := range leagues {
			fmt.Fprintf(table, "%s\t%s\t%.64s\n", league.ShortCode, league.Name, league.Settings)
		}
		table.Flush()

		fmt.Fprint(out, "```\n")
	}

	return nil
}
