.PHONY: build install clean

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

build:
	@echo "Building musing $(VERSION)..."
	@go build $(LDFLAGS) -o musing ./cmd/musing

install: build
	@echo "Installing musing to /usr/local/bin/..."
	@sudo cp musing /usr/local/bin/

clean:
	@rm -f musing
	@rm -rf dist/

.DEFAULT_GOAL := build
