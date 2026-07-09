# Grok Proxy Plus — terminal edition build helpers.
# The desktop (Wails) build still works via `wails build` — see README.md.

GO            ?= go
GOFLAGS       ?= -trimpath -ldflags="-s -w"
CLI_PKG       := ./cmd/grok-proxy-cli
CLI_BIN       := grok-proxy-cli
BUILD_DIR     := build/bin

.PHONY: all cli build-cli install clean test selftest fmt vet help

all: cli

# Build the terminal-only binary into build/bin.
cli: build-cli
build-cli:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(CLI_BIN) $(CLI_PKG)
	@echo "built $(BUILD_DIR)/$(CLI_BIN)"

# Install the CLI to $$GOBIN (or $$HOME/.local/bin).
install: build-cli
	@dest="$(shell go env GOBIN 2>/dev/null || echo $$HOME/.local/bin)"; \
	  mkdir -p $$dest; \
	  cp $(BUILD_DIR)/$(CLI_BIN) $$dest/$(CLI_BIN); \
	  echo "installed $$dest/$(CLI_BIN)"

# Quick smoke test of the CLI's help output.
test: build-cli
	@./$(BUILD_DIR)/$(CLI_BIN) --help >/dev/null && echo "cli help OK"
	@./$(BUILD_DIR)/$(CLI_BIN) accounts >/dev/null 2>&1 || true

# Original self-test (needs at least one logged-in account).
selftest:
	$(GO) run ./cmd/selftest

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf $(BUILD_DIR)/*

help:
	@echo "targets:"
	@echo "  make cli        build the terminal CLI into build/bin"
	@echo "  make install    build and install to $$GOBIN or ~/.local/bin"
	@echo "  make test       quick CLI smoke test"
	@echo "  make selftest   integration self-test (needs an account)"
	@echo "  make fmt/vet    format / lint"
	@echo "  make clean      remove build/bin"
