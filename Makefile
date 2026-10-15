GO ?= /opt/homebrew/bin/go
export PATH := /opt/homebrew/bin:$(PATH)

.PHONY: dev build test test-pg tidy lint-sr

TEST_DATABASE_URL ?= postgres://localhost/srpski_test?sslmode=disable

dev:
	@echo "frontend: http://localhost:5173  (proxies /api to :8080)"
	@( cd web && npm run dev ) & \
	 APP_BASE_URL=http://localhost:5173 $(GO) run ./server -addr :8080 ; \
	 kill %1 2>/dev/null || true

build:
	cd web && npm ci && npm run build
	rm -rf server/web/dist && cp -r web/dist server/web/dist
	cd landing && npm ci && npm run generate
	rm -rf server/landing/dist && cp -r landing/.output/public server/landing/dist
	$(GO) build -o serbian-app ./server

test:
	$(GO) test ./server/...
	cd web && npm run test

# store tests against a real Postgres (needs a running server + `createdb srpski_test`)
test-pg:
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" $(GO) test ./server/internal/store/ -count=1

tidy:
	$(GO) mod tidy

# Проверка сербского текста курса: орфография/экавица/алфавиты (lint_serbian.py)
# и сверка ударений со словарём (check_stress.py, только отчёт). Нужно:
# pip3 install spylls pyyaml; при первом запуске качает словари в scripts/.cache/.
lint-sr:
	python3 scripts/lint_serbian.py --self-test
	python3 scripts/lint_serbian.py
	python3 scripts/check_stress.py
	python3 scripts/check_stress_tokens.py
