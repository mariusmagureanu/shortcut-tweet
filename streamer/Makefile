# Makefile
SHELL := /bin/sh

REVISION := $(shell git rev-parse HEAD)
VERSION := $(shell git rev-parse --abbrev-ref HEAD)

.PHONY: all
all: build

.PHONY: build
build:
	@go generate
	@go build -o bin/streamer -ldflags "-s -w -X main.revision=${REVISION} -X main.version=${VERSION}"

osxbuild:
	@go generate
	@GOOS=darwin GOARCH=amd64 go build -o bin/streamer -ldflags "-s -w -X main.revision=${REVISION} -X main.version=${VERSION}"

clean:
	@rm -rf bin/
