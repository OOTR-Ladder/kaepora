package main

import (
	"flag"
	"fmt"
	"kaepora/internal/back"
	"kaepora/internal/bot"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
)

// Version holds the build-time version string.
var Version = "unknown" // nolint:gochecknoglobals

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()

	switch flag.Arg(0) { // commands not requiring a back
	case "version":
		fmt.Fprintf(os.Stdout, "Kaepora %s\n", Version)
		return
	case "help":
		fmt.Fprint(os.Stdout, help())
		return
	}

	back, err := back.New("sqlite3", "./kaepora.db")
	if err != nil {
		log.Fatal(err)
	}

	switch flag.Arg(0) {
	case "fixtures":
		if err := back.LoadFixtures(); err != nil {
			log.Fatal(err)
		}
	case "serve":
		if err := serve(back); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Fprint(os.Stderr, help())
		os.Exit(1)
	}
}

func help() string {
	return fmt.Sprintf(`
Kaepora is a tool to manage the "Ocarina of Time: Randomizer"
competitive ladder.

Usage: %[1]s COMMAND [ARGS…]

COMMANDS
    fixtures    create default data for quick testing during development
    help        display this help
    serve       start the Discord bot
    version     display the current version
`,
		os.Args[0],
	)
}

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
