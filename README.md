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

## Build and run
```shell
$ # Install Go: https://golang.org/dl/
$ make
$ ./migrate -database sqlite3://kaepora.db -path resources/migrations up
$ ./kaepora fixtures
$ # Place ZOOTDEC.z64 and ARCHIVE.bin in the `resources/oot-randomizer` directory.
$ KAEPORA_DISCORD_TOKEN=$YOUR_BOT_TOKEN \
  KAEPORA_ADMIN_USER=$YOUR_DISCORD_USER_ID \
  ./kaepora serve
```

## Hacking
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
