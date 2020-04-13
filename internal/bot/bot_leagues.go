package bot

import (
	"fmt"
	"io"
	"kaepora/internal/util"
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
		return util.ErrPublic("bad arguments count")
	}
}

func (bot *Bot) displayLeagues(out io.Writer) error {
	games, leagues, times, err := bot.back.GetGamesLeaguesAndTheirNextSessionStartDate()
	if err != nil {
		return err
	}

	for k, game := range games {
		fmt.Fprintf(out, "%d. Leagues for _%s_:\n", k+1, game.Name)
		fmt.Fprint(out, "```\n")

		table := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

		fmt.Fprintln(table, "shortcode\tname\tnext race\t")
		fmt.Fprintln(table, "\t\t\t")
		now := time.Now()

		for _, league := range leagues {
			if league.GameID != game.ID { // Hello O(n²) my old friend.
				continue
			}

			var nextStr, nextDeltaStr string
			if next, ok := times[league.ID]; ok {
				nextStr = next.Format("2006-01-02 15:04 MST")
				delta := next.Sub(now).Truncate(time.Minute)
				nextDeltaStr = "(in " + strings.TrimSuffix(delta.String(), "0s") + ")"
			}

			fmt.Fprintf(
				table, "%s\t%s\t%s\t%s\n",
				league.ShortCode, league.Name,
				nextStr, nextDeltaStr,
			)
		}
		table.Flush()

		fmt.Fprint(out, "```\n")
	}

	return nil
}
