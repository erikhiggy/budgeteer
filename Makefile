# Budgeteer: Go API + embedded Vite/React frontend.

WEB := web
BIN := bin/budgeteer

.PHONY: dev dev-api dev-web build test test-go test-web clean

## dev: run the Go API (./dev.db) and the Vite dev server together; open http://localhost:5173
dev: $(WEB)/node_modules
	$(MAKE) -j2 dev-api dev-web

dev-api:
	go run ./cmd/budgeteer --db ./dev.db

dev-web:
	npm --prefix $(WEB) run dev

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
	npm --prefix $(WEB) run typecheck
	npm --prefix $(WEB) test

clean:
	rm -rf bin
	find $(WEB)/dist -mindepth 1 ! -name .gitkeep -delete

$(WEB)/node_modules: $(WEB)/package.json $(WEB)/package-lock.json
	npm --prefix $(WEB) ci
	@touch $@
