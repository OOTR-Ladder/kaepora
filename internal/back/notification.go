package back

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"kaepora/internal/generator"
	"kaepora/internal/generator/oot"
	"kaepora/internal/util"
	"log"
	"text/tabwriter"
	"time"

	"github.com/jmoiron/sqlx"
)

type NotificationRecipientType int

const (
	NotificationRecipientTypeDiscordChannel NotificationRecipientType = 0
	NotificationRecipientTypeDiscordUser    NotificationRecipientType = 1
)

type NotificationType int

const (
	NotificationTypeMatchSessionStatusUpdate NotificationType = iota
	NotificationTypeMatchEnd
	NotificationTypeMatchSeed
	NotificationTypeMatchSessionCountdown
	NotificationTypeMatchSessionEmpty
	NotificationTypeMatchSessionOddKick
	NotificationTypeMatchSessionRecap
	NotificationTypeSpoilerLog
	NotificationTypeLeagueLeaderboardUpdate
)

type NotificationFile struct {
	Name        string
	ContentType string
	Reader      io.Reader
}

type Notification struct {
	RecipientType NotificationRecipientType
	Recipient     string
	Type          NotificationType
	Files         []NotificationFile

	body bytes.Buffer
}

func (n *Notification) SetDiscordUserRecipient(userID string) {
	n.Recipient = userID
	n.RecipientType = NotificationRecipientTypeDiscordUser
}

func (n *Notification) Printf(str string, args ...interface{}) (int, error) {
	return fmt.Fprintf(&n.body, str, args...)
}

func (n *Notification) Print(args ...interface{}) (int, error) {
	return fmt.Fprint(&n.body, args...)
}

func (n *Notification) Read(p []byte) (int, error) {
	return n.body.Read(p)
}

func (n *Notification) Write(p []byte) (int, error) {
	return n.body.Write(p)
}

func (n *Notification) Reset() {
	n.body.Reset()
	n.Files = nil
}

func NotificationTypeName(typ NotificationType) string {
	switch typ {
	case NotificationTypeMatchSessionStatusUpdate:
		return "MatchSessionStatusUpdate"
	case NotificationTypeMatchSessionEmpty:
		return "MatchSessionEmpty"
	case NotificationTypeMatchSessionOddKick:
		return "MatchSessionOddKick"
	case NotificationTypeMatchSessionRecap:
		return "MatchSessionRecap"
	case NotificationTypeMatchSeed:
		return "MatchSeed"
	case NotificationTypeMatchEnd:
		return "MatchEnd"
	case NotificationTypeSpoilerLog:
		return "SpoilerLog"
	default:
		return "invalid"
	}
}

func NotificationRecipientTypeName(typ NotificationRecipientType) string {
	switch typ {
	case NotificationRecipientTypeDiscordChannel:
		return "DiscordChannel"
	case NotificationRecipientTypeDiscordUser:
		return "DiscordUser"
	default:
		return "invalid"
	}
}

// For debugging purposes only.
func (n *Notification) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(
		&buf,
		"type %s, recipient type %s \"%s\"",
		NotificationTypeName(n.Type),
		NotificationRecipientTypeName(n.RecipientType),
		n.Recipient,
	)

	if len := len(n.Files); len > 0 {
		fmt.Fprintf(&buf, ", %d file(s)", len)
	}

	// HACK: Ensure its on one line (and safe to print)
	content, _ := json.Marshal(n.body.String())
	fmt.Fprintf(&buf, ", contents: %s", string(content))

	return buf.String()
}

func (b *Back) sendOddKickNotification(player Player) {
	notif := Notification{
		RecipientType: NotificationRecipientTypeDiscordUser,
		Recipient:     player.DiscordID.String,
		Type:          NotificationTypeMatchSessionOddKick,
	}

	notif.Printf(
		"Sorry %s, but there was an odd number of players and you were the last person to join.\n"+
			"You have been kicked out of the race, don't worry this won't affect your ranking.\n",
		player.Name,
	)

	b.notifications <- notif
}

func (b *Back) sendMatchSessionEmptyNotification(tx *sqlx.Tx, session MatchSession) error {
	league, err := getLeagueByID(tx, session.LeagueID)
	if err != nil {
		return err
	}

	notif := Notification{
		RecipientType: NotificationRecipientTypeDiscordChannel,
		Recipient:     league.AnnounceDiscordChannelID.String,
		Type:          NotificationTypeMatchSessionEmpty,
	}

	notif.Printf(
		"The race for league `%s` is closed, you can no longer join.\n"+
			"There was not enough players to start the race.\n",
		league.ShortCode,
	)

	b.notifications <- notif
	return nil
}

func (b *Back) sendMatchEndNotification(
	tx *sqlx.Tx,
	selfEntry MatchEntry,
	opponentEntry MatchEntry,
	player Player,
) error {
	opponent, err := getPlayerByID(tx, opponentEntry.PlayerID)
	if err != nil {
		return err
	}

	notif := Notification{
		RecipientType: NotificationRecipientTypeDiscordUser,
		Recipient:     player.DiscordID.String,
		Type:          NotificationTypeMatchEnd,
	}

	notif.Printf("%s, your race against %s has ended.\n", player.Name, opponent.Name)

	start, end := selfEntry.StartedAt.Time.Time(), selfEntry.EndedAt.Time.Time()
	if !start.IsZero() {
		delta := end.Sub(start).Round(time.Second)
		if selfEntry.Status == MatchEntryStatusForfeit {
			notif.Printf("You forfeited your race after %s.\n", delta)
		} else if selfEntry.Status == MatchEntryStatusFinished {
			notif.Printf("You completed your race in %s.\n", delta)
		}
	} else if selfEntry.Status == MatchEntryStatusForfeit {
		notif.Print("You forfeited before the race started.\n")
	}

	start, end = opponentEntry.StartedAt.Time.Time(), opponentEntry.EndedAt.Time.Time()
	if !start.IsZero() {
		delta := end.Sub(start).Round(time.Second)
		if opponentEntry.Status == MatchEntryStatusForfeit {
			notif.Printf("%s forfeited after %s.\n", opponent.Name, delta)
		} else if opponentEntry.Status == MatchEntryStatusFinished {
			notif.Printf("%s completed their race in %s.\n", opponent.Name, delta)
		}
	} else if opponentEntry.Status == MatchEntryStatusForfeit {
		notif.Printf("%s forfeited before the race started.\n", opponent.Name)
	}

	switch selfEntry.Outcome {
	case MatchEntryOutcomeWin:
		notif.Print("**You won!**\n")
	case MatchEntryOutcomeDraw:
		notif.Print("**The race is a draw.**\n")
	case MatchEntryOutcomeLoss:
		notif.Printf("**%s wins.**\n", opponent.Name)
	}

	if opponent.StreamURL != "" {
		notif.Printf("Your opponent stream: %s\n", opponent.StreamURL)
	}

	b.notifications <- notif
	return nil
}

func (b *Back) sendMatchSeedNotification(
	session MatchSession,
	url string,
	out generator.Output,
	p1, p2 Player,
) {
	name := fmt.Sprintf(
		"seed_%s.zpf",
		session.StartDate.Time().Format("2006-01-02_15h04"),
	)

	send := func(player Player) {
		notif := Notification{
			RecipientType: NotificationRecipientTypeDiscordUser,
			Recipient:     player.DiscordID.String,
			Type:          NotificationTypeMatchSeed,
		}

		if url == "" {
			notif.Files = []NotificationFile{{
				Name:        name,
				ContentType: "application/zlib",
				Reader:      bytes.NewReader(out.SeedPatch),
			}}

			notif.Print(
				"Here is your seed in _Patch_ format. " +
					"You can use https://ootrandomizer.com/generator to patch your ROM.\n",
			)
		} else {
			notif.Printf("Here is your seed: %s\n", url)
		}

		if hash := hashFromSpoilerLog(out.SpoilerLog); hash != "" {
			notif.Print("Your seed hash is: **", hash, "**\n")
		}

		if !session.StartDate.Time().IsZero() {
			notif.Printf(
				"Your race starts in %s, **do not explore the seed before the match starts**.\n",
				time.Until(session.StartDate.Time()).Round(time.Second),
			)
		}

		if err := maybeWriteSettingsPatchInfo(&notif, out.State); err != nil {
			log.Printf("warning: unable to send settings info: %v", err)
		}

		b.notifications <- notif
	}

	send(p1)
	send(p2)
}

// maybeWriteSettingsPatchInfo sends OOTR-specific documentation about the
// settings used to generate the seed (used in shuffled settings).
func maybeWriteSettingsPatchInfo(w io.Writer, stateJSON []byte) error {
	var state oot.State
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return err
	}
	if len(state.SettingsPatch) == 0 {
		return nil
	}

	doc, err := LoadSettingsDocumentation("en") // HARDCODED every bot message is in English.
	if err != nil {
		return err
	}

	fmt.Fprint(w, "Seed settings:\n")

	for k, v := range state.SettingsPatch {
		setting := doc[k]
		value := setting.GetValueEntry(v)

		if setting.Title == "" {
			log.Printf("warning: no title for setting %s", k)
			continue
		}
		if value.Title == "" {
			log.Printf("warning: no title for value %s = %v", k, v)
			continue
		}

		fmt.Fprintf(w, "  - %s: %s\n", setting.Title, value.Title)
	}

	return nil
}

func (b *Back) sendSessionStatusUpdateNotification(tx *sqlx.Tx, session MatchSession) error {
	league, err := getLeagueByID(tx, session.LeagueID)
	if err != nil {
		return err
	}

	notif := Notification{
		RecipientType: NotificationRecipientTypeDiscordChannel,
		Recipient:     league.AnnounceDiscordChannelID.String,
		Type:          NotificationTypeMatchSessionStatusUpdate,
	}

	switch session.Status {
	case MatchSessionStatusWaiting:
		notif.Printf(
			"The next race for league `%s` has been scheduled for %s (in %s)",
			league.ShortCode,
			util.Datetime(session.StartDate),
			time.Until(session.StartDate.Time()).Round(time.Second),
		)
	case MatchSessionStatusJoinable:
		notif.Printf(
			"The race for league `%s` can now be joined! The race starts at %s (in %s).\n"+
				"You can join using `!join %s`.",
			league.ShortCode,
			util.Datetime(session.StartDate),
			time.Until(session.StartDate.Time()).Round(time.Second),
			league.ShortCode,
		)
	case MatchSessionStatusPreparing:
		notif.Printf(
			"The race for league `%s` has begun preparations, you can no longer join. "+
				"Seeds will soon be sent to the %d contestants.\n"+
				"The race starts at %s (in %s). Watch this channel for the official go.",
			league.ShortCode,
			len(session.PlayerIDs)-(len(session.PlayerIDs)%2),
			util.Datetime(session.StartDate),
			time.Until(session.StartDate.Time()).Round(time.Second),
		)
	case MatchSessionStatusInProgress:
		notif.Printf(
			"The race for league `%s` **starts now**. Good luck and have fun!",
			league.ShortCode,
		)
	case MatchSessionStatusClosed:
		notif.Printf(
			"All players have finished their last `%s` race, rankings have been updated.",
			league.ShortCode,
		)
	}

	b.notifications <- notif
	return nil
}

func (b *Back) sendSessionCountdownNotification(tx *sqlx.Tx, session MatchSession) error {
	league, err := getLeagueByID(tx, session.LeagueID)
	if err != nil {
		return err
	}

	notif := Notification{
		RecipientType: NotificationRecipientTypeDiscordChannel,
		Recipient:     league.AnnounceDiscordChannelID.String,
		Type:          NotificationTypeMatchSessionStatusUpdate,
	}

	notif.Printf(
		"The next race for league `%s` starts in %s.",
		league.ShortCode,
		time.Until(session.StartDate.Time()).Round(time.Second),
	)

	b.notifications <- notif
	return nil
}

func (b *Back) sendLeaderboardUpdateNotification(
	tx *sqlx.Tx,
	leagueID util.UUIDAsBlob,
) error {
	league, err := getLeagueByID(tx, leagueID)
	if err != nil {
		return err
	}

	notif := Notification{
		RecipientType: NotificationRecipientTypeDiscordChannel,
		Recipient:     league.AnnounceDiscordChannelID.String,
		Type:          NotificationTypeLeagueLeaderboardUpdate,
	}

	top, err := getTop20(tx, league.ID, DeviationThreshold)
	if err != nil {
		return err
	}

	if len(top) == 0 {
		return nil
	}

	notif.Printf("Top players for league `%s`:\n```\n", league.ShortCode)
	for i := range top {
		notif.Printf(" %2.d. %s\n", i+1, top[i].PlayerName)
	}
	notif.Print("```\n")

	b.notifications <- notif
	return nil
}

type RecapScope int

const (
	RecapScopePublic RecapScope = iota // only show completed races
	RecapScopeRunner                   // also show races with one finisher
	RecapScopeAdmin                    // show everything
)

func (b *Back) sendSessionRecapNotification(
	tx *sqlx.Tx,
	session MatchSession,
	matches []Match,
	scope RecapScope, // private recap, don't send to announce channel
	toDiscordUserID *string, // can be nil for public recaps: will send to announce channel
) error {
	league, err := getLeagueByID(tx, session.LeagueID)
	if err != nil {
		return err
	}

	notif := Notification{
		RecipientType: NotificationRecipientTypeDiscordChannel,
		Recipient:     league.AnnounceDiscordChannelID.String,
		Type:          NotificationTypeMatchSessionRecap,
	}
	if toDiscordUserID != nil {
		notif.SetDiscordUserRecipient(*toDiscordUserID)
	}

	notif.Printf("Results for `%s` race started at %s:\n```\n", league.ShortCode, util.Datetime(session.StartDate))
	known, unknown := writeResultsTable(tx, &notif, matches, scope)
	notif.Print("```\n")

	if known == 0 {
		notif.Reset()
	}

	if unknown > 0 {
		notif.Printf("There are still %d race(s) in progress.\n", unknown)
		notif.Printf("You can get an up to date recap with `!recap %s`.", league.ShortCode)
	} else {
		notif.Printf("Get the seeds and spoiler logs on https://ootrladder.com/en/sessions/%s", session.ID)
	}

	b.notifications <- notif
	return nil
}

// writeResultsTable is an helper for sendSessionRecapNotification.
func writeResultsTable(
	tx *sqlx.Tx,
	w io.Writer,
	matches []Match,
	scope RecapScope,
) (known, unknown int) {
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "Player 1\t\tvs\tPlayer 2\t")

	for _, match := range matches {
		if scope != RecapScopeAdmin {
			if !match.Entries[0].HasEnded() && !match.Entries[1].HasEnded() {
				unknown++
				continue
			}

			if scope == RecapScopePublic && (!match.Entries[0].HasEnded() || !match.Entries[1].HasEnded()) {
				unknown++
				continue
			}
		}

		wrap0, name0, duration0 := entryDetails(tx, match.Entries[0])
		wrap1, name1, duration1 := entryDetails(tx, match.Entries[1])
		fmt.Fprint(
			table,
			wrap0, name0, wrap0, "\t", duration0, "\t\t",
			wrap1, name1, wrap1, "\t", duration1, "\n",
		)
		known++
	}

	table.Flush()
	return known, unknown
}

// entryDetails is a formatting helper for sendSessionRecapNotification.
func entryDetails(tx *sqlx.Tx, entry MatchEntry) (wrap string, name string, duration string) {
	if entry.HasWon() {
		wrap = "*"
	}

	player, _ := getPlayerByID(tx, entry.PlayerID)
	name = player.Name

	switch entry.Status {
	case MatchEntryStatusWaiting:
		duration = "not started"
	case MatchEntryStatusInProgress:
		duration = "in progress"
	case MatchEntryStatusForfeit:
		if entry.StartedAt.Time.Time().IsZero() {
			duration = "forfeit (before start)"
			break
		}

		delta := entry.EndedAt.Time.Time().Sub(entry.StartedAt.Time.Time()).Round(time.Second)
		duration = "forfeit (" + delta.String() + ")"
	case MatchEntryStatusFinished:
		delta := entry.EndedAt.Time.Time().Sub(entry.StartedAt.Time.Time()).Round(time.Second)
		duration = delta.String()
	}

	return
}

func (b *Back) sendSpoilerLogNotification(player Player, seed string, spoilerLog util.ZLIBBlob) {
	notif := Notification{
		RecipientType: NotificationRecipientTypeDiscordUser,
		Recipient:     player.DiscordID.String,
		Type:          NotificationTypeSpoilerLog,
	}

	if len(spoilerLog) == 0 {
		notif.Printf("There is no spoiler log available for seed `%s`.", seed)
	} else {
		notif.Files = []NotificationFile{{
			Name:        fmt.Sprintf("%s.spoilers.json", seed),
			ContentType: "application/json",
			Reader:      spoilerLog.Uncompressed(),
		}}
		notif.Printf("Here is the spoiler log for seed `%s`.", seed)
	}

	b.notifications <- notif
}
