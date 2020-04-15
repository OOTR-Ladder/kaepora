package bot

import (
	"bytes"
	"fmt"
	"kaepora/internal/back"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) sendNotification(notif back.Notification) error {
	switch notif.Type {
	case back.NotificationTypeMatchSessionCountdown:
		return bot.sendMatchSessionCountdown(notif)
	case back.NotificationTypeMatchEnd:
		return bot.sendMatchEndNotification(notif)
	case back.NotificationTypeMatchSessionEmpty:
		return bot.sendMatchSessionEmptyNotification(notif)
	case back.NotificationTypeMatchSeed:
		return bot.sendMatchSeedNotification(notif)
	default:
		return fmt.Errorf("got unknown notification type: %d", notif.Type)
	}
}

func (bot *Bot) sendMatchSessionCountdown(notif back.Notification) error {
	w, err := bot.getWriterForNotification(notif)
	if err != nil {
		return err
	}
	defer w.Flush()

	session := notif.Payload["MatchSession"].(back.MatchSession)
	league := notif.Payload["League"].(back.League)

	switch session.Status {
	case back.MatchSessionStatusWaiting:
		fmt.Fprintf(w,
			"The next race for league `%s` has been scheduled for %s (in %s)",
			league.ShortCode,
			session.StartDate.Time(),
			time.Until(session.StartDate.Time()).Truncate(time.Second),
		)
	case back.MatchSessionStatusJoinable:
		fmt.Fprintf(w,
			"The race for league `%s` can now be joined! The race starts at %s (in %s).\n"+
				"You can join using `!join %s`",
			league.ShortCode,
			session.StartDate.Time(),
			time.Until(session.StartDate.Time()).Truncate(time.Second),
			league.ShortCode,
		)
	case back.MatchSessionStatusPreparing:
		fmt.Fprintf(w,
			"The race for league `%s` is closed, you can no longer join. "+
				"Seeds will soon be sent to the %d contestants.\n"+
				"The race starts at %s (in %s). Watch this channel for the official go.",
			league.ShortCode,
			len(session.PlayerIDs)-(len(session.PlayerIDs)%2),
			session.StartDate.Time(),
			time.Until(session.StartDate.Time()).Truncate(time.Second),
		)
	case back.MatchSessionStatusInProgress:
		fmt.Fprintf(w,
			"The race for league `%s` **starts now**. Good luck and have fun!",
			league.ShortCode,
		)
	}

	return nil
}

func (bot *Bot) sendMatchSessionEmptyNotification(notif back.Notification) error {
	w, err := bot.getWriterForNotification(notif)
	if err != nil {
		return err
	}
	defer w.Flush()

	league := notif.Payload["League"].(back.League)

	fmt.Fprintf(w,
		"The race for league `%s` is closed, you can no longer join.\n"+
			"There was not enough players to start the race.",
		league.ShortCode,
	)

	return nil
}

func (bot *Bot) sendMatchEndNotification(notif back.Notification) error {
	w, err := bot.getWriterForNotification(notif)
	if err != nil {
		return err
	}
	defer w.Flush()

	opponent := notif.Payload["Opponent"].(back.Player)
	selfEntry := notif.Payload["PlayerMatchEntry"].(back.MatchEntry)
	opponentEntry := notif.Payload["OpponentMatchEntry"].(back.MatchEntry)

	fmt.Fprintf(w, "Your race against %s has ended.\n", opponent.Name)

	start, end := selfEntry.StartedAt.Time.Time(), selfEntry.EndedAt.Time.Time()
	delta := end.Sub(start).Truncate(time.Second)
	if selfEntry.Status == back.MatchEntryStatusForfeit {
		fmt.Fprintf(w, "You forfeited your race after %s.\n", delta)
	} else if selfEntry.Status == back.MatchEntryStatusFinished {
		fmt.Fprintf(w, "You completed your race in %s.\n", delta)
	}

	start, end = opponentEntry.StartedAt.Time.Time(), opponentEntry.EndedAt.Time.Time()
	delta = end.Sub(start).Truncate(time.Second)
	if opponentEntry.Status == back.MatchEntryStatusForfeit {
		fmt.Fprintf(w, "%s forfeited after %s.\n", opponent.Name, delta)
	} else if opponentEntry.Status == back.MatchEntryStatusFinished {
		fmt.Fprintf(w, "%s completed his/her race in %s.\n", opponent.Name, delta)
	}

	switch selfEntry.Outcome {
	case back.MatchEntryOutcomeWin:
		fmt.Fprint(w, "**You won!**")
	case back.MatchEntryOutcomeDraw:
		fmt.Fprint(w, "**The race is a draw.**")
	case back.MatchEntryOutcomeLoss:
		fmt.Fprintf(w, "**%s wins.**", opponent.Name)
	}

	return nil
}

func (bot *Bot) getWriterForNotification(notif back.Notification) (*channelWriter, error) {
	switch notif.RecipientType {
	case back.NotificationRecipientTypeDiscordUser:
		return newUserChannelWriter(bot.dg, notif.Recipient)
	case back.NotificationRecipientTypeDiscordChannel:
		return newChannelWriter(bot.dg, notif.Recipient), nil
	default:
		return nil, fmt.Errorf("cannot handle recipient type: %d", notif.RecipientType)
	}
}

func (bot *Bot) sendMatchSeedNotification(notif back.Notification) error {
	w, err := bot.getWriterForNotification(notif)
	if err != nil {
		return err
	}
	defer w.Flush()

	session := notif.Payload["MatchSession"].(back.MatchSession)
	name := fmt.Sprintf(
		"seed_%s.zpf",
		session.StartDate.Time().Format("2006-01-02_15h04"),
	)

	w.files = append(w.files, &discordgo.File{
		Name:        name,
		ContentType: "application/zlib",
		Reader:      bytes.NewReader(notif.Payload["patch"].([]byte)),
	})

	fmt.Fprintf(w,
		"Here is your seed in _Patch_ format. "+
			"You can use https://ootrandomizer.com/generator to patch your ROM.\n"+
			"Your race starts in %s, **do not explore the seed before the match starts**.",
		time.Until(session.StartDate.Time()).Truncate(time.Second),
	)

	return nil
}
