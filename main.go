package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// Version holds the build-time version string.
var Version = "unknown" // nolint:gochecknoglobals

func main() {
	flag.Parse()

	var err error

	switch flag.Arg(0) {
	case "version":
		fmt.Fprintf(os.Stdout, "Kaepora %s\n", Version)
	case "dev:fixtures":
		err = loadFixtures()
	case "help":
		fmt.Fprint(os.Stdout, help())
		return
	default:
		fmt.Fprint(os.Stderr, help())
		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func help() string {
	return fmt.Sprintf(`
Kaepora is a tool to manage the "Ocarina of Time: Randomizer"
competitive ladder.

Usage: %[1]s COMMAND [ARGS…]

COMMANDS
    dev:fixtures create default data for quick testing during development
    help         display this help
    version      display the current version
`,
		os.Args[0],
	)
}
