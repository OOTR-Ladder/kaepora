package bot

import (
	"bytes"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// channelWriter outputs messages to a Discord channel (or private message)
// when flushed, it can be reused right after flushing to send a new message.
type channelWriter struct {
	channelID string
	dg        *discordgo.Session
	buf       bytes.Buffer
}

func newUserChannelWriter(dg *discordgo.Session, user *discordgo.User) (*channelWriter, error) {
	channel, err := dg.UserChannelCreate(user.ID)
	if err != nil {
		return nil, fmt.Errorf("unable to create user channel: %s", err)
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
}

func (w *channelWriter) Flush() error {
	if w.buf.Len() <= 0 {
		return nil
	}

	_, err := w.dg.ChannelMessageSend(w.channelID, w.buf.String())
	w.buf.Reset()
	return err
}
