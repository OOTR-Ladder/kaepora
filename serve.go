package main

import (
	"kaepora/internal/back"
	"kaepora/internal/bot"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func serve(b *back.Back) error {
	done := make(chan struct{})
	signaled := make(chan os.Signal, 1)
	signal.Notify(signaled, syscall.SIGINT, syscall.SIGTERM)

	bot, err := bot.New(b, os.Getenv("KAEPORA_DISCORD_TOKEN"))
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	go b.Run(&wg, done)
	go bot.Serve(&wg, done)

	sig := <-signaled
	log.Printf("warning: received signal %d", sig)
	close(done)

	log.Print("info: waiting for complete shutdown")
	wg.Wait()
	log.Print("info: shutdown complete")

	return nil
}
