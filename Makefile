EXEC=./$(shell basename "$(shell pwd)")
VERSION ?= $(shell git describe --tags 2>/dev/null || echo "unknown")
GOLANGCI=./golangci-lint
BUILDFLAGS=-tags 'sqlite_json' -ldflags '-X main.Version=${VERSION}'

all: $(EXEC) migrate

$(EXEC):
	go build $(BUILDFLAGS)

migrate:
	go build -tags "sqlite3 sqlite_json" github.com/golang-migrate/migrate/v4/cmd/migrate

.PHONY: $(EXEC) vendor upgrade lint test coverage randomizer docker

docker:
	docker build . \
		-f "docker/kaepora.dockerfile" \
		-t "kaepora:${VERSION}" \
		--build-arg "VERSION=${VERSION}"

randomizer:
	cp docker/.dockerignore docker/OoT-Randomizer/
	docker build docker/OoT-Randomizer \
		-f "docker/OoT-Randomizer.dockerfile" \
		-t "lp042/oot-randomizer:$(shell git -C docker/OoT-Randomizer describe --tags)"
	rm docker/OoT-Randomizer/.dockerignore

push-randomizer: randomizer
	docker push "lp042/oot-randomizer:$(shell git -C docker/OoT-Randomizer describe --tags)"

coverage:
	go test -covermode=count -coverprofile=coverage.cov --timeout=10s ./...
	go tool cover -html=coverage.cov -o coverage.html
	rm coverage.cov
	sensible-browser coverage.html

test:
	go test --timeout=10s ./...

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
