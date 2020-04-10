EXEC=./$(shell basename "$(shell pwd)")
VERSION ?= $(shell git describe --tags 2>/dev/null || echo "unknown")
GOLANGCI=./golangci-lint
BUILDFLAGS=-tags 'sqlite_json' -ldflags '-X main.Version=${VERSION}'

all: $(EXEC) $(GOLANGCI) migrate

$(EXEC):
	go build $(BUILDFLAGS)

migrate:
	go build -tags "sqlite3 sqlite_json" github.com/golang-migrate/migrate/v4/cmd/migrate

.PHONY: $(EXEC) vendor upgrade lint test coverage randomizer


randomizer:
	cp docker/.dockerignore docker/OoT-Randomizer/
	docker build docker/OoT-Randomizer \
		-f docker/OoT-Randomizer.dockerfile \
		-t oot-randomizer:$(shell git -C docker/OoT-Randomizer describe --tags)
	rm docker/OoT-Randomizer/.dockerignore

coverage:
	go test -covermode=count -coverprofile=coverage.cov --timeout=6s ./...
	go tool cover -html=coverage.cov -o coverage.html
	rm coverage.cov
	sensible-browser coverage.html

test:
	go test --timeout=6s ./...

vendor:
	go get -v
	go mod vendor
	go mod tidy

upgrade:
	go get -u -v
	go mod vendor
	go mod tidy

$(GOLANGCI):
	go build github.com/golangci/golangci-lint/cmd/golangci-lint

lint: $(GOLANGCI)
	$(GOLANGCI) run
