package bot

import (
	"bytes"
	"fmt"
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
}

func newUserChannelWriter(dg *discordgo.Session, userID string) (*channelWriter, error) {
	channel, err := dg.UserChannelCreate(userID)
	if err != nil {
		return nil, fmt.Errorf("unable to create user channel: %w", err)
	}

	return newChannelWriter(dg, channel.ID), nil
}

func newChannelWriter(dg *discordgo.Session, channelID string) *channelWriter {
	return &channelWriter{
		dg:        dg,
		channelID: channelID,
	}
}

func (w *channelWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *channelWriter) Reset() {
	w.buf.Reset()
	w.files = nil
}

func (w *channelWriter) Flush() error {
	if w.buf.Len() <= 0 {
		return nil
	}

	msg := discordgo.MessageSend{
		Content: w.buf.String(),
		Files:   w.files,
	}

	_, err := w.dg.ChannelMessageSendComplex(w.channelID, &msg)
	log.Print("info: <self> " + msg.Content)

	w.buf.Reset()
	return err
}
