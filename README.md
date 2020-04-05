# Kaepora
The _Ocarina of Time Randomizer_ leagues.

## Build
```shell
$ # Install Go: https://golang.org/dl/
$ make
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
