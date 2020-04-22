package bot

import (
	"bytes"
	"fmt"
	"kaepora/internal/back"
	"log"

	"github.com/bwmarrin/discordgo"
)

// channelWriter outputs messages to a Discord channel (or private message)
// when flushed, it can be reused right after flushing to send a new message.
type channelWriter struct {
	channelID string
	dg        *discordgo.Session
	buf       bytes.Buffer
	files     []*discordgo.File

	debugInfo string
}

func (w *channelWriter) addFile(file back.NotificationFile) {
	w.files = append(w.files, &discordgo.File{
		Name:        file.Name,
		ContentType: file.ContentType,
		Reader:      file.Reader,
	})
}

func newUserChannelWriter(dg *discordgo.Session, userID string) (*channelWriter, error) {
	if userID == "" {
		log.Print("warning: skipping creating writer for empty Discord user ID")
		return nil, nil
	}

	channel, err := dg.UserChannelCreate(userID)
	if err != nil {
		return nil, fmt.Errorf("unable to create user channel: %w", err)
	}

	ret := newChannelWriter(dg, channel.ID)
	ret.debugInfo = fmt.Sprintf("<to user %s (chan %s)>", userID, channel.ID)

	return ret, nil
}

func newChannelWriter(dg *discordgo.Session, channelID string) *channelWriter {
	if channelID == "" {
		log.Print("warning: skipping creating writer for empty Discord channel ID")
		return nil
	}

	return &channelWriter{
		dg:        dg,
		channelID: channelID,
		debugInfo: fmt.Sprintf("<to chan %s>", channelID),
	}
}

func (w *channelWriter) Write(p []byte) (int, error) {
	if w == nil {
		return 0, nil
	}

	return w.buf.Write(p)
}

func (w *channelWriter) Reset() {
	if w == nil {
		return
	}

	w.buf.Reset()
	w.files = nil
}

func (w *channelWriter) Flush() error {
	if w == nil || w.buf.Len() <= 0 {
		return nil
	}

	msg := discordgo.MessageSend{
		Content: w.buf.String(),
		Files:   w.files,
	}

	_, err := w.dg.ChannelMessageSendComplex(w.channelID, &msg)
	log.Printf("info: %s: %s", w.debugInfo, msg.Content)

	w.buf.Reset()
	return err
}
