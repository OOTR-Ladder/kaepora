package bot

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) cmdLeagues(_ *discordgo.Message, args []string, out io.Writer) error {
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

		table := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

		fmt.Fprintln(table, "shortcode\tname\tnext race\t\tsettings")
		fmt.Fprintln(table, "\t\t\t\t")
		now := time.Now()
		for _, league := range leagues {
			var nextStr, nextDeltaStr string
			next, err := bot.back.GetNextMatchSessionForLeague(league.ID)
			if err == nil {
				nextStr = next.StartDate.Time().Format("2006-01-02 15:04 MST")
				delta := next.StartDate.Time().Sub(now).Truncate(time.Minute)
				nextDeltaStr = "(in " + strings.TrimSuffix(delta.String(), "0s") + ")"
			}

			fmt.Fprintf(
				table, "%s\t%s\t%s\t%s\t%.64s\n",
				league.ShortCode, league.Name,
				nextStr, nextDeltaStr, league.Settings,
			)
		}
		table.Flush()

		fmt.Fprint(out, "```\n")
	}

	return nil
}
