VERSION?=
BINARY?=dedupe
COMMIT?=$(shell git rev-parse --short HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
MODBASE=github.com/alexgQQ/dedupe

all: fmt clean test build

build:
	go build -ldflags " \
		-s -w \
		-X ${MODBASE}/utils.Version=${VERSION} \
		-X ${MODBASE}/utils.Branch=${BRANCH} \
		-X ${MODBASE}/utils.Commit=${COMMIT} \
		" \
		-o dist/${BINARY} cmd/dedupe/main.go

test:
	go test --count 5 ./...

fmt:
	go fmt ./...

clean:
	-rm -f dist/*

.PHONY: build test fmt clean