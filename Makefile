EXEC=./$(shell basename "$(shell pwd)")
VERSION ?= $(shell git describe --tags 2>/dev/null || echo "unknown")
GOLANGCI=./golangci-lint
BUILDFLAGS=-tags 'sqlite_json' -ldflags '-X kaepora/internal/global.Version=${VERSION}'
SRC=$(shell find internal pkg -type f -name "*.go")

all: $(EXEC) migrate

$(EXEC):
	go build $(BUILDFLAGS)

tags: $(SRC)
	ctags-universal -R internal pkg

migrate:
	go build -tags "sqlite3 sqlite_json" github.com/golang-migrate/migrate/v4/cmd/migrate

.PHONY: $(EXEC) vendor upgrade lint test coverage randomizer docker run extract

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
	go test -tags docker,api -covermode=count -coverprofile=coverage.cov --timeout=30s ./...
	go tool cover -html=coverage.cov -o coverage.html
	rm coverage.cov
	sensible-browser coverage.html

test:
	go test -tags docker,api --timeout=30s ./...

run:
	gin --all --bin "$(EXEC)" -x kaepora.db --buildArgs "${BUILDFLAGS}" -- serve

runi:
	gin --immediate --bin "$(EXEC)" --buildArgs "${BUILDFLAGS}" -- serve

extract:
	./extract > out.po
	msgmerge --lang=en --backup=off -F -U resources/web/locales/en/default.po out.po
	msgmerge --lang=fr --backup=off -F -U resources/web/locales/fr/default.po out.po
	rm out.po

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
