APP_NAME=stress-test
MODULE=github.com/JeanGrijp/stress-test
LDFLAGS=-X $(MODULE)/internal/version.Version=$(VERSION) -X $(MODULE)/internal/version.Commit=$(COMMIT) -X $(MODULE)/internal/version.Date=$(DATE)

VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

.PHONY: build install clean

build:
	go build -ldflags '$(LDFLAGS)' -o bin/$(APP_NAME) ./cmd/$(APP_NAME)

install:
	go install -ldflags '$(LDFLAGS)' ./cmd/$(APP_NAME)

clean:
	rm -rf bin

.PHONY: docs
docs: build
	./bin/$(APP_NAME) docs --format markdown --out-dir ./docs/cli
