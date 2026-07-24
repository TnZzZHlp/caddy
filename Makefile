.PHONY: build test

BUILD_DIR ?= build
CADDY_BINARY ?= $(BUILD_DIR)/caddy

build:
	mkdir -p $(BUILD_DIR)
	go build -trimpath -ldflags='-s -w' -o $(CADDY_BINARY) ./cmd/caddy

test:
	go test ./...
