.PHONY: all build build-web test clean run dev

BIN_DIR := bin
BIN_NAME := zenfd
BINARY := $(BIN_DIR)/$(BIN_NAME)
CUR_DIR := $(shell pwd)
WEB_DIR := $(CUR_DIR)/../web
GATEWAY_WEB_DIST := $(CUR_DIR)/internal/gateway/web_dist
AIR_BIN := $(shell which air 2>/dev/null || echo "$$(go env GOPATH)/bin/air")

all: test build

build-web:
	@if [ -d "$(WEB_DIR)" ]; then \
		echo "Building web dashboard in $(WEB_DIR)..."; \
		(cd $(WEB_DIR) && pnpm build); \
		rm -rf $(GATEWAY_WEB_DIST)/*; \
		mkdir -p $(GATEWAY_WEB_DIST); \
		cp -r $(WEB_DIR)/dist/* $(GATEWAY_WEB_DIST)/; \
		echo "Copied web bundle to $(GATEWAY_WEB_DIST)"; \
	fi

build: build-web
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY) ./cmd/zenfd

test:
	go vet ./...
	go test -v -race -count=1 ./...

clean:
	rm -rf $(BIN_DIR) tmp

dev:
	@if [ -x "$(AIR_BIN)" ]; then \
		echo "Starting zenfd with air live-reload..."; \
		$(AIR_BIN) -c .air.toml; \
	else \
		echo "air not found, running with go run..."; \
		go run ./cmd/zenfd daemon --port 1123 --db ~/.zenforge/state.db; \
	fi

run:
	go run ./cmd/zenfd daemon --port 1123 --db ~/.zenforge/state.db
