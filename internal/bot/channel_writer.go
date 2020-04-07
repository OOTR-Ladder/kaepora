package bot

import (
	"bytes"

	"github.com/bwmarrin/discordgo"
)

type channelWriter struct {
	channelID string
	dg        *discordgo.Session
	buf       bytes.Buffer
	toUser    *discordgo.User
}

func newChannelWriter(dg *discordgo.Session, channelID string, toUser *discordgo.User) *channelWriter {
	return &channelWriter{
		dg:        dg,
		channelID: channelID,
		toUser:    toUser,
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
	return err
}
