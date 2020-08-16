package main

import (
	"flag"
	"fmt"
	"kaepora/internal/back"
	"kaepora/internal/bot"
	"kaepora/internal/config"
	"kaepora/internal/global"
	"kaepora/internal/web"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.SetFlags(0) // we syslog in prod so we don't care about time here
	flag.Parse()

	switch flag.Arg(0) { // commands not requiring a back
	case "version":
		fmt.Fprintf(os.Stdout, "Kaepora %s\n", global.Version)
		return
	case "help":
		fmt.Fprint(os.Stdout, help())
		return
	}

	conf, err := config.NewFromUserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("info: Starting Kaepora %s", global.Version)
	back, err := back.New(
		"sqlite3", "./kaepora.db",
		os.Getenv("KAEPORA_OOTR_API_KEY"),
		conf,
	)
	if err != nil {
		log.Fatal(err)
	}

	switch flag.Arg(0) {
	case "fixtures":
		if err := back.LoadFixtures(); err != nil {
			log.Fatal(err)
		}
	case "serve":
		if err := serve(back, conf); err != nil {
			log.Fatal(err)
		}
	case "rerank":
		if err := back.Rerank(flag.Arg(1)); err != nil {
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

    rerank SHORTCODE  recompute all rankings in a league
`,
		os.Args[0],
	)
}

func serve(b *back.Back, conf *config.Config) error {
	done := make(chan struct{})
	signaled := make(chan os.Signal, 1)
	signal.Notify(signaled, syscall.SIGINT, syscall.SIGTERM)

	bot, err := bot.New(b, os.Getenv("KAEPORA_DISCORD_TOKEN"), conf)
	if err != nil {
		return err
	}

	server, err := web.NewServer(b, os.Getenv("KAEPORA_WEB_TOKEN_KEY"))
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	go b.Run(&wg, done)
	go bot.Serve(&wg, done)
	go server.Serve(&wg, done)

	sig := <-signaled
	log.Printf("warning: received signal %d", sig)
	close(done)

	log.Print("info: waiting for complete shutdown")
	wg.Wait()
	log.Print("info: shutdown complete")

	return nil
}
