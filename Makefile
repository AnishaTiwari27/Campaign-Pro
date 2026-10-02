SHELL := /bin/bash

# Local Postgres is the default target because this repo was built and
# verified on a machine without Docker. `make dev-docker` runs the same
# stack through docker-compose where Docker is available.
DATABASE_URL ?= postgres://campaign_tracker_pro:campaign_tracker_pro_dev_pw@localhost:5432/campaign_tracker_pro?sslmode=disable
PORT ?= 8090
MIGRATIONS := db/migrations

.PHONY: help dev dev-api dev-web dev-docker db-create migrate migrate-down seed logos test test-go test-web test-e2e lint fmt generate

help:
	@echo "make dev          - run api (:$(PORT)) and web (:5173) together"
	@echo "make db-create    - create the local database + role"
	@echo "make migrate      - apply migrations"
	@echo "make seed         - truncate and reseed demo data"
	@echo "make logos        - fetch brand logos into web/public/logos"
	@echo "make test         - go tests + vitest + playwright"
	@echo "make lint         - go vet + gofmt check + eslint"
	@echo "make generate     - regenerate sqlc code"
	@echo "make dev-docker   - run the whole stack via docker-compose"

dev:
	@trap 'kill 0' EXIT; \
	$(MAKE) dev-api & \
	$(MAKE) dev-web & \
	wait

dev-api:
	DATABASE_URL="$(DATABASE_URL)" PORT=$(PORT) go run ./cmd/server

dev-web:
	cd frontend && npm run dev

dev-docker:
	docker compose up --build

db-create:
	createdb campaign_tracker_pro || true
	psql -d campaign_tracker_pro -c "DO \$$\$$ BEGIN CREATE ROLE campaign_tracker_pro LOGIN PASSWORD 'campaign_tracker_pro_dev_pw'; EXCEPTION WHEN duplicate_object THEN NULL; END \$$\$$;"
	psql -d campaign_tracker_pro -c "GRANT ALL ON SCHEMA public TO campaign_tracker_pro;"
	psql -d campaign_tracker_pro -c "ALTER DATABASE campaign_tracker_pro OWNER TO campaign_tracker_pro;"

migrate:
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" down 1

logos:
	./scripts/fetch-logos.sh

seed:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/seed

generate:
	sqlc generate

test: test-go test-web test-e2e

test-go:
	DATABASE_URL="$(DATABASE_URL)" go test ./...

test-web:
	cd frontend && npm run test

# Playwright drives the running dev servers, so seed first for a known state.
test-e2e: seed
	cd frontend && npx playwright test

lint:
	go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)
	cd frontend && npm run lint

fmt:
	gofmt -w .
