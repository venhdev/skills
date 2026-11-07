.PHONY: all build build-web build-daemon test test-web test-daemon test-all clean dev

all: test-all build

# Web & Daemon Delegation
build-web:
	@if [ -d "apps/web" ]; then \
		echo "Building web dashboard in apps/web..."; \
		(cd apps/web && pnpm build); \
		mkdir -p apps/daemon/internal/gateway/web_dist; \
		find apps/daemon/internal/gateway/web_dist -mindepth 1 ! -name '.keep' -delete; \
		cp -r apps/web/dist/* apps/daemon/internal/gateway/web_dist/; \
		touch apps/daemon/internal/gateway/web_dist/.keep; \
		echo "Copied web bundle to apps/daemon/internal/gateway/web_dist"; \
	fi

build-daemon:
	@echo "Building zenfd daemon..."
	@$(MAKE) -C apps/daemon build

build: build-daemon

# Testing
test-web:
	@if [ -d "apps/web" ]; then \
		echo "Testing apps/web..."; \
		(cd apps/web && pnpm test); \
	fi

test-daemon:
	@echo "Testing apps/daemon..."
	@$(MAKE) -C apps/daemon test

test: test-daemon

test-all: test-web test-daemon

# Development
dev:
	@echo "Starting zenfd in dev mode..."
	@$(MAKE) -C apps/daemon dev

# Cleanup
clean:
	@echo "Cleaning artifacts..."
	@$(MAKE) -C apps/daemon clean
	@rm -rf apps/web/dist
