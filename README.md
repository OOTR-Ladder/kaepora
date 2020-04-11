# Kaepora
The _Ocarina of Time Randomizer_ leagues.

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
