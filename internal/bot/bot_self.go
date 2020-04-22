package bot

import (
	"fmt"
	"io"
	"kaepora/internal/util"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) cmdRename(m *discordgo.Message, args []string, out io.Writer) error {
	if len(args) < 1 {
		return util.ErrPublic("your forgot to tell me your desired name")
	}

	return bot.setDiscordPlayerName(m.Author.ID, argsAsName(args), out)
}

func (bot *Bot) setDiscordPlayerName(discordID string, name string, out io.Writer) error {
	if err := bot.back.UpdateDiscordPlayerName(discordID, name); err != nil {
		return err
	}

	fmt.Fprintf(out, "You'll be henceforth known as `%s` on the leaderboards.", name)
	return nil
}

func (bot *Bot) cmdRegister(m *discordgo.Message, args []string, out io.Writer) error {
	name := argsAsName(args)
	if name == "" {
		name = m.Author.Username
	}

	if err := bot.back.RegisterDiscordPlayer(m.Author.ID, name); err != nil {
		return err
	}

	fmt.Fprintf(out, "You have been registered as `%s`, see you on the leaderboards.", name)
	return nil
}

func (bot *Bot) cmdLeaderboards(m *discordgo.Message, args []string, w io.Writer) error {
	shortcode := argsAsName(args)
	top, around, err := bot.back.GetLeaderboardsForDiscordUser(m.Author.ID, shortcode)
	if err != nil {
		return err
	}

	if len(top) == 0 && len(around) == 0 {
		fmt.Fprintf(w, "The leaderboard for league `%s` is empty, join the next race!", shortcode)
		return nil
	}

	fmt.Fprintf(w, "Top players for league `%s`:\n```\n", shortcode)
	for i := range top {
		fmt.Fprintf(w, " %2.d. %s\n", i+1, top[i].PlayerName)
	}
	fmt.Fprint(w, "```\n")

	if len(around) > 0 {
		fmt.Fprint(w, "Players around you:\n```\n")
		for i := range around {
			fmt.Fprintf(w, "  - %s\n", around[i].PlayerName)
		}
		fmt.Fprint(w, "```")
	}

	return nil
}
