.PHONY: build install test clean

BINARY=ali
GO=go
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo v1.1.0)
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS=-ldflags "-s -w -X github.com/philmehew/ali/internal/version.Version=$(VERSION) -X github.com/philmehew/ali/internal/version.Commit=$(COMMIT) -X github.com/philmehew/ali/internal/version.BuildDate=$(BUILD_DATE)"

build:
	$(GO) build $(LDFLAGS) -o $(BINARY) ./cmd/ali

install: build
	cp $(BINARY) /usr/local/bin/

test:
	$(GO) test ./... -v

clean:
	rm -f $(BINARY)
