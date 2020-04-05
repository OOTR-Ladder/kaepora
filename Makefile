EXEC=$(shell basename "$(shell pwd)")
VERSION ?= $(shell git describe --tags 2>/dev/null || echo "unknown")

all: $(EXEC) migrate

$(EXEC):
	go build -tags "sqlite_json"

migrate:
	go build -tags "sqlite3 sqlite_json" github.com/golang-migrate/migrate/v4/cmd/migrate

.PHONY: vendor upgrade
vendor:
	go get -v
	go mod vendor
	go mod tidy

upgrade:
	go get -u -v
	go mod vendor
	go mod tidy
