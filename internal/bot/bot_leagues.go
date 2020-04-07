package bot

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"
)

func (bot *Bot) dispatchLeagues(args []string, out io.Writer) error {
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
			next := league.Schedule.Next()
			var nextStr, nextDeltaStr string
			if !next.IsZero() {
				nextStr = next.Format("2006-01-02 15:04 MST")
				nextDeltaStr = strings.TrimSuffix(next.Sub(now).Truncate(time.Minute).String(), "0s")
			}

			fmt.Fprintf(
				table, "%s\t%s\t%s\t(in %s)\t%.64s\n",
				league.ShortCode, league.Name,
				nextStr, nextDeltaStr, league.Settings,
			)
		}
		table.Flush()

		fmt.Fprint(out, "```\n")
	}

	return nil
}
