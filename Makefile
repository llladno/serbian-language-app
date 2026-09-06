GO ?= /opt/homebrew/bin/go
export PATH := /opt/homebrew/bin:$(PATH)

.PHONY: dev build test test-pg tidy

TEST_DATABASE_URL ?= postgres://localhost/srpski_test?sslmode=disable

dev:
	@echo "frontend: http://localhost:5173  (proxies /api to :8080)"
	@( cd web && npm run dev ) & \
	 $(GO) run ./server -addr :8080 ; \
	 kill %1 2>/dev/null || true

build:
	cd web && npm ci && npm run build
	rm -rf server/web/dist && cp -r web/dist server/web/dist
	$(GO) build -o serbian-app ./server

test:
	$(GO) test ./server/...
	cd web && npm run test

# store tests against a real Postgres (needs a running server + `createdb srpski_test`)
test-pg:
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" $(GO) test ./server/internal/store/ -count=1

tidy:
	$(GO) mod tidy
