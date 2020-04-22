package bot

import (
	"fmt"
	"io"
	"kaepora/internal/back"
	"log"
)

func (bot *Bot) sendNotification(notif back.Notification) error {
	w, err := bot.getWriterForNotification(notif)
	if err != nil {
		return err
	}
	if w == nil {
		log.Printf("info: not sent: %s", notif.String())
		return nil
	}

	if len(notif.Files) > 0 {
		for k := range notif.Files {
			w.addFile(notif.Files[k])
		}
	}

	if _, err := io.Copy(w, &notif); err != nil {
		return fmt.Errorf("error: unable to copy notification body buffer: %w", err)
	}

	return w.Flush()
}

// getWriterForNotification returns a writer to the recipient in the given
// notification, the returned writer might be null with no error if the
// recipient is empty, meaning we should just ignore it.
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
