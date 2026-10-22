# Budgeteer: Go API + embedded Vite/React frontend.

WEB := web
BIN := bin/budgeteer
DEV_DB := ./dev.db

.PHONY: dev dev-api dev-web reset-dev build test test-go test-web fmt clean

## dev: run the Go API ($(DEV_DB)) and the Vite dev server together; open http://localhost:5173
dev: $(WEB)/node_modules
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
build: $(WEB)/node_modules
	npm --prefix $(WEB) run build
	go build -trimpath -o $(BIN) ./cmd/budgeteer

## test: Go and frontend tests
test: test-go test-web

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

clean:
	rm -rf bin
	find $(WEB)/dist -mindepth 1 ! -name .gitkeep -delete

$(WEB)/node_modules: $(WEB)/package.json $(WEB)/package-lock.json
	npm --prefix $(WEB) ci
	@touch $@
