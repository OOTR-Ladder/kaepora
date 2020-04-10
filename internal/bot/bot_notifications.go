package bot

import (
	"fmt"
	"kaepora/internal/back"
	"time"
)

func (bot *Bot) sendNotification(notif back.Notification) error {
	switch notif.Type {
	case back.NotificationTypeSessionCountdown:
		return bot.sendCountdown(notif)
	default:
		return fmt.Errorf("got unknown notification type: %d", notif.Type)
	}
}

func (bot *Bot) sendCountdown(notif back.Notification) error {
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
			"The race for league `%s` can now be joined!\nThe race starts in %s.",
			league.ShortCode,
			time.Until(session.StartDate.Time()).Truncate(time.Second),
		)
	case back.MatchSessionStatusPreparing:
		fmt.Fprintf(w,
			"The race for league `%s` is closed, you can no longer join. "+
				"Seeds will soon be sent to the contestants.\n"+
				"The race starts in %s.",
			league.ShortCode,
			time.Until(session.StartDate.Time()).Truncate(time.Second),
		)
	case back.MatchSessionStatusInProgress:
		fmt.Fprintf(w,
			"The race for league `%s` **starts now**.\nGood luck and have fun!",
			league.ShortCode,
		)
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
