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
	NotificationTypeMatchSessionCountdown NotificationType = iota
	NotificationTypeMatchSessionEmpty
	NotificationTypeMatchSessionOddKick
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

func (b *Back) sendOddKickNotification(player Player) {
	if !player.DiscordID.Valid {
		log.Printf(
			"ignored sendOddKickNotification to Player '%s' without a DiscordID",
			player.ID,
		)
		return
	}

	notif := Notification{
		RecipientType: NotificationRecipientTypeDiscordUser,
		Recipient:     player.DiscordID.String,
		Type:          NotificationTypeMatchSessionOddKick,
		Payload:       nil,
	}
	b.notifications <- notif
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
		Type:          NotificationTypeMatchSessionCountdown,
		Payload: map[string]interface{}{
			"MatchSession": session,
			"League":       league,
		},
	}
	b.notifications <- notif

	return nil
}

func (b *Back) sendMatchSessionEmptyNotification(tx *sqlx.Tx, session MatchSession) error {
	league, err := getLeagueByID(tx, session.LeagueID)
	if err != nil {
		return err
	}

	if league.AnnounceDiscordChannelID == "" {
		log.Printf(
			"ignored sendMatchSessionEmptyNotification for league '%s' without a channel",
			league.ShortCode,
		)
		return nil
	}

	notif := Notification{
		RecipientType: NotificationRecipientTypeDiscordChannel,
		Recipient:     league.AnnounceDiscordChannelID,
		Type:          NotificationTypeMatchSessionEmpty,
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
		Type:          NotificationTypeMatchEnd,
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

func (b *Back) sendMatchSeedNotification(
	match Match,
	session MatchSession,
	patch []byte,
	p1, p2 Player,
) {
	send := func(player Player) {
		if !player.DiscordID.Valid {
			log.Printf("ignored sendMatchSeedNotification to Player '%s' without a DiscordID", player.ID)
			return
		}

		b.notifications <- Notification{
			RecipientType: NotificationRecipientTypeDiscordUser,
			Recipient:     player.DiscordID.String,
			Type:          NotificationTypeMatchSeed,
			Payload: map[string]interface{}{
				"patch":        patch,
				"MatchSession": session,
				"Match":        match,
			},
		}
	}

	send(p1)
	send(p2)
}
