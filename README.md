# Kaepora [![Build Status](https://travis-ci.org/OOTR-Ladder/kaepora.svg?branch=master)](https://travis-ci.org/OOTR-Ladder/kaepora) [![go report](https://goreportcard.com/badge/github.com/OOTR-Ladder/kaepora)](https://goreportcard.com/report/github.com/OOTR-Ladder/kaepora)

[The _Ocarina of Time Randomizer_ leagues](https://ootrladder.com).

Goals:
  - Provide a 1vs1 ladder for OoT randomizer races.
  - Provide multiple leagues to compete with different settings.
  - Web-first, Discord-second, easy to port on other messaging platforms.

Side-goals:
  - Provide a framework for other randomized games to use for their ladder.
  - Provide an OoT-specific randomizer settings randomizer (sic).

Non-goals and out of scope:
  - Providing a generic tournament bot.

## Configuration (env vars)
  - `KAEPORA_DISCORD_TOKEN`, can be omitted if you don't want to run the bot.
  - `KAEPORA_OOTR_API_KEY`, can be omitted if you don't use remote seedgen.

## Configuration file
Located at: `$XDG_CONFIG_HOME/kaepora/config.json`
```
{
	"DiscordAdminUserIDs": ["<discord user id>"]
}
```

Having at least one user ID is mandatory to make the bot listen to a channel
and not only to PMs.

## Build and run
```shell
$ # Install Go: https://golang.org/dl/
$ make
$ ./migrate -database sqlite3://kaepora.db -path resources/migrations up
$ ./kaepora fixtures
$ # Place ZOOTDEC.z64 and ARCHIVE.bin in the resources/oot-randomizer directory.
$ ./kaepora serve
```

To allow/disallow the bot to listen to a channel, send `!dev addlisten` or
`!dev removelisten` in said channel.

## Migrations
- Running migrations:
```shell
$ # Go to latest version:
$ ./migrate -database sqlite3://kaepora.db -path resources/migrations up
$ # Revert last migration:
$ ./migrate -database sqlite3://kaepora.db -path resources/migrations down 1
```
- Adding a migration:
```shell
$ ./migrate create -ext sql -dir resources/migrations -seq NAME
```
