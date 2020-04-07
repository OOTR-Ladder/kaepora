package bot

import (
	"fmt"
	"io"
)

func (bot *Bot) dispatchGames(args []string, out io.Writer) error {
	if len(args) > 0 {
		return errPublic("this command takes no argument")
	}

	return bot.displayGames(out)
}

func (bot *Bot) displayGames(out io.Writer) error {
	games, err := bot.back.GetGames()
	if err != nil {
		return err
	}

	if len(games) == 0 {
		fmt.Fprint(out, "There is no registered game yet.")
		return nil
	}

	fmt.Fprint(out, "Here are the available games:\n\n")
	for k, v := range games {
		fmt.Fprintf(out, "%d. %s\n", k+1, v.Name)
	}

	return nil
}
