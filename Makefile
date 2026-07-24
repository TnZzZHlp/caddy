.PHONY: build test adapt

BUILD_DIR ?= build
CADDY_BINARY ?= $(BUILD_DIR)/caddy
CADDYFILE ?= Caddyfile

build:
	mkdir -p $(BUILD_DIR)
	go build -trimpath -ldflags='-s -w' -o $(CADDY_BINARY) ./cmd/caddy

test:
	go test ./...

adapt: build
	@test -f "$(CADDYFILE)" || { echo "Caddyfile not found: $(CADDYFILE)" >&2; exit 1; }
	$(CADDY_BINARY) adapt --config "$(CADDYFILE)" --adapter caddyfile >/dev/null
