package back

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

type NotificationRecipientType int

const (
	NotificationRecipientTypeDiscordChannel NotificationRecipientType = 0
	NotificationRecipientTypeDiscordUser    NotificationRecipientType = 1
)

type NotificationType int

const (
	NotificationTypeSessionCountdown NotificationType = 0
	NotificationTypeMatchSeed        NotificationType = 1
	NotificationTypeMatchEnd         NotificationType = 2
)

type Notification struct {
	RecipientType NotificationRecipientType
	Recipient     string
	Type          NotificationType
	Payload       map[string]interface{}
}

// For debugging purposes only.
func (n *Notification) String() string {
	return fmt.Sprintf("notif type %d for recipient of type %d (%s)", n.Type, n.RecipientType, n.Recipient)
}

func (b *Back) sendSessionCountdownNotification(tx *sqlx.Tx, session MatchSession) error {
	league, err := getLeagueByID(tx, session.LeagueID)
	if err != nil {
		return err
	}

	if league.AnnounceDiscordChannelID == "" {
		log.Printf(
			"ignored insertSessionCountdownNotification for league '%s' without a channel",
			league.ShortCode,
		)
		return nil
	}

	notif := Notification{
		RecipientType: NotificationRecipientTypeDiscordChannel,
		Recipient:     league.AnnounceDiscordChannelID,
		Type:          NotificationTypeSessionCountdown,
		Payload: map[string]interface{}{
			"MatchSession": session,
			"League":       league,
		},
	}
	b.notifications <- notif

	return nil
}
