package main

import (
	"flag"
	"fmt"
	"io"
	"kaepora/internal/back"
	"kaepora/internal/bot"
	"kaepora/internal/generator"
	"kaepora/internal/web"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// Version holds the build-time version string.
var Version = "unknown" // nolint:gochecknoglobals

func main() {
	log.SetFlags(0) // we syslog in prod so we don't care about time here
	flag.Parse()

	switch flag.Arg(0) { // commands not requiring a back
	case "version":
		fmt.Fprintf(os.Stdout, "Kaepora %s\n", Version)
		return
	case "help":
		fmt.Fprint(os.Stdout, help())
		return
	case "spoilers":
		if err := generateSpoilerLogs(); err != nil {
			log.Fatal(err)
		}
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
    spoilers    generate a spoiler log in $cwd/spoilers
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

func generateSpoilerLogs() error {
	generator, err := generator.NewGenerator("oot-randomizer:5.2.12")
	if err != nil {
		return err
	}

	seed := uuid.New().String()
	_, spoiler, err := generator.Generate("s3.json", seed)
	if err != nil {
		return err
	}

	dir := filepath.Join("spoilers", seed[0:2], seed[2:4])
	if err := os.MkdirAll(dir, 0o775); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(dir, seed+".json"), os.O_RDWR|os.O_CREATE, 0o664)
	if err != nil {
		return err
	}

	if _, err := io.WriteString(f, spoiler); err != nil {
		f.Close()
		return err
	}

	return f.Close()
}
