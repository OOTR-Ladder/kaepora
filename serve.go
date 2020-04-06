package main

import (
	"kaepora/internal/back"
	"kaepora/internal/bot"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func serve(b *back.Back) error {
	closer := make(chan struct{})
	signaled := make(chan os.Signal, 1)
	signal.Notify(signaled, syscall.SIGINT, syscall.SIGTERM)

	bot, err := bot.New(b, os.Getenv("KAEPORA_DISCORD_TOKEN"), closer)
	if err != nil {
		return err
	}

	bot.Serve()

	select {
	case sig := <-signaled:
		log.Printf("received signal %d", sig)
	case <-closer:
		log.Print("bot killed itself")
	}

	if err := bot.Close(); err != nil {
		return err
	}

	log.Print("shutdown complete")

	return nil
}
