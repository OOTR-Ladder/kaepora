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
	NotificationTypeSessionCountdown NotificationType = iota
	NotificationTypeSessionOddKick
	NotificationTypeMatchSeed
	NotificationTypeMatchEnd
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

func (b *Back) sendOddKickNotification(player Player) error {
	if !player.DiscordID.Valid {
		log.Printf(
			"ignored sendMatchEndNotification to Player '%s' without a DiscordID",
			player.ID,
		)
		return nil
	}

	notif := Notification{
		RecipientType: NotificationRecipientTypeDiscordUser,
		Recipient:     player.DiscordID.String,
		Type:          NotificationTypeSessionOddKick,
		Payload:       nil,
	}
	b.notifications <- notif

	return nil
}

func (b *Back) sendSessionCountdownNotification(tx *sqlx.Tx, session MatchSession) error {
	league, err := getLeagueByID(tx, session.LeagueID)
	if err != nil {
		return err
	}

	if league.AnnounceDiscordChannelID == "" {
		log.Printf(
			"ignored sendSessionCountdownNotification for league '%s' without a channel",
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

func (b *Back) sendMatchEndNotification(tx *sqlx.Tx, match Match, player Player) error {
	if !player.DiscordID.Valid {
		log.Printf(
			"ignored sendMatchEndNotification to Player '%s' without a DiscordID",
			player.ID,
		)
		return nil
	}

	playerEntry, opponentEntry, err := match.getPlayerAndOpponentEntries(player.ID)
	if err != nil {
		return err
	}

	opponent, err := getPlayerByID(tx, opponentEntry.PlayerID)
	if err != nil {
		return err
	}

	notif := Notification{
		RecipientType: NotificationRecipientTypeDiscordUser,
		Recipient:     player.DiscordID.String,
		Type:          NotificationTypeSessionCountdown,
		Payload: map[string]interface{}{
			"Player":             player,
			"Opponent":           opponent,
			"PlayerMatchEntry":   playerEntry,
			"OpponentMatchEntry": opponentEntry,
		},
	}
	b.notifications <- notif

	return nil
}
