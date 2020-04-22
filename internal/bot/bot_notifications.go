package bot

import (
	"fmt"
	"io"
	"kaepora/internal/back"
)

func (bot *Bot) sendNotification(notif back.Notification) error {
	w, err := bot.getWriterForNotification(notif)
	if err != nil {
		return err
	}
	defer w.Flush()

	if len(notif.Files) > 0 {
		for k := range notif.Files {
			w.addFile(notif.Files[k])
		}
	}

	if _, err := io.Copy(w, &notif); err != nil {
		return fmt.Errorf("error: unable to copy notification body buffer: %w", err)
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
