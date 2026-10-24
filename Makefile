# Budgeteer: Go API + embedded Vite/React frontend.

WEB := web
BIN := bin/budgeteer
DEV_DB := ./dev.db
# The version `--version` prints: a tag like v1.0.0, or a commit hash before
# the first tag, with -dirty for uncommitted changes. "dev" outside git.
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.DEFAULT_GOAL := help
.PHONY: help tools dev dev-api dev-web reset-dev build test test-go test-web fmt clean

## help: list the targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'

# tools: fail early, with a readable message, when Go or Node is missing or
# Node is too old for Vite and Vitest. (Go itself enforces go.mod's version.)
tools:
	@command -v go >/dev/null || { echo "Go is required: https://go.dev/dl/"; exit 1; }
	@command -v node >/dev/null || { echo "Node.js 22.12 or later is required: https://nodejs.org/"; exit 1; }
	@node -e 'const [a,b]=process.versions.node.split(".").map(Number); process.exit(a>22||(a===22&&b>=12)?0:1)' \
		|| { echo "Node.js 22.12 or later is required; this is $$(node --version)."; exit 1; }

## dev: run the Go API (./dev.db) and the Vite dev server together; open http://localhost:5173
dev: tools $(WEB)/node_modules
	@echo "Open http://localhost:5173 (Ctrl-C stops both)."
	$(MAKE) -j2 dev-api dev-web

dev-api:
	go run ./cmd/budgeteer --db $(DEV_DB)

dev-web:
	npm --prefix $(WEB) run dev

## reset-dev: delete the dev database; the next `make dev` recreates it from the migrations
reset-dev:
	@if lsof -nP -iTCP:7420 -sTCP:LISTEN >/dev/null 2>&1; then \
		echo "Something is listening on :7420. Stop \`make dev\` first, or the server keeps using the deleted file."; \
		exit 1; \
	fi
	rm -f $(DEV_DB) $(DEV_DB)-shm $(DEV_DB)-wal

## build: build the frontend, then a single binary with it embedded
build: tools $(WEB)/node_modules
	npm --prefix $(WEB) run build
	go build -trimpath -ldflags "-X main.version=$(VERSION)" -o $(BIN) ./cmd/budgeteer
	@echo "Built $(BIN) ($(VERSION)). Run it with ./$(BIN)"

## test: Go and frontend tests
test: tools test-go test-web

test-go:
	go vet ./...
	go test ./...

test-web: $(WEB)/node_modules
	npm --prefix $(WEB) run format:check
	npm --prefix $(WEB) run typecheck
	npm --prefix $(WEB) test

## fmt: gofmt the Go code and Prettier the frontend
fmt: $(WEB)/node_modules
	gofmt -w cmd internal web/embed.go
	npm --prefix $(WEB) run format

## clean: delete the binary and the built frontend
clean:
	rm -rf bin
	find $(WEB)/dist -mindepth 1 ! -name .gitkeep -delete

$(WEB)/node_modules: $(WEB)/package.json $(WEB)/package-lock.json
	npm --prefix $(WEB) ci
	@touch $@
