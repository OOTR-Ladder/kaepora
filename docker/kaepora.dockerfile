FROM golang:1.14.2-alpine AS builder
ARG VERSION=unkown

WORKDIR $GOPATH/src/kaepora

RUN apk add --no-cache \
    make git gcc musl-dev

COPY vendor vendor
COPY *.go Makefile go.mod go.sum ./
COPY internal internal

RUN VERSION=${VERSION} make

FROM docker:19.03
WORKDIR /opt/kaepora
ENTRYPOINT ["/usr/bin/entrypoint"]
CMD ["/opt/kaepora/kaepora", "serve"]
VOLUME [ \
    "/opt/kaepora/kaepora.db", \
    "/opt/kaepora/resources/oot-randomizer/ARCHIVE.bin", \
    "/opt/kaepora/resources/oot-randomizer/ZOOTDEC.z64", \
]

RUN apk add --no-cache tzdata ca-certificates

COPY resources /opt/kaepora/resources
COPY docker/entrypoint /usr/bin/entrypoint
COPY --from=builder /go/src/kaepora/kaepora /opt/kaepora/kaepora
COPY --from=builder /go/src/kaepora/migrate /opt/kaepora/migrate
