SHELL := /bin/bash

# Local Postgres is the default target because this repo was built and
# verified on a machine without Docker. `make dev-docker` runs the same
# stack through docker-compose where Docker is available.
DATABASE_URL ?= postgres://campaign_tracker_pro:campaign_tracker_pro_dev_pw@localhost:5432/campaign_tracker_pro?sslmode=disable
PORT ?= 8090
MIGRATIONS := backend/db/migrations

.PHONY: help dev dev-api dev-web dev-docker db-create migrate migrate-down seed logos build build-web test test-go test-web test-e2e lint fmt generate

help:
	@echo "make dev          - run api (:$(PORT)) and web (:5173) together"
	@echo "make db-create    - create the local database + role"
	@echo "make migrate      - apply migrations"
	@echo "make seed         - truncate and reseed demo data"
	@echo "make build        - frontend + single binary with the app embedded"
	@echo "make logos        - fetch brand logos into frontend/public/logos"
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
	cd backend && DATABASE_URL="$(DATABASE_URL)" PORT=$(PORT) go run ./cmd/server

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
	cd backend && DATABASE_URL="$(DATABASE_URL)" go run ./cmd/seed

# Build the frontend and copy it where the Go binary embeds it from, so the
# API port serves the real app. This is what the Docker build does, done
# locally — useful for verifying the production shape without Docker.
build-web:
	cd frontend && npm run build
	rm -rf backend/web/dist
	mkdir -p backend/web/dist
	cp -R frontend/dist/. backend/web/dist/
	touch backend/web/dist/.gitkeep
	@echo "embedded $$(find backend/web/dist -type f | wc -l | tr -d ' ') files; rebuild the server to pick them up"

# Everything the production image contains: frontend embedded, one binary.
build: build-web
	cd backend && CGO_ENABLED=0 go build -o ../bin/server ./cmd/server
	@echo "built ./bin/server"

generate:
	cd backend && sqlc generate

test: test-go test-web test-e2e

test-go:
	cd backend && DATABASE_URL="$(DATABASE_URL)" go test ./...

test-web:
	cd frontend && npm run test

# Playwright drives the running dev servers, so seed first for a known state.
# The e2e suite signs in as fixture users, so it needs a password it knows.
# This is the only place that value exists; an ordinary `make seed` gets a
# generated one, so seeding a deployment never installs a published
# credential.
test-e2e:
	cd backend && SEED_PASSWORD=demo-password-change-me DATABASE_URL="$(DATABASE_URL)" go run ./cmd/seed
	cd frontend && npx playwright test

lint:
	cd backend && go vet ./...
	@test -z "$$(cd backend && gofmt -l .)" || (echo "gofmt needed:"; cd backend && gofmt -l .; exit 1)
	cd frontend && npm run lint

fmt:
	cd backend && gofmt -w .
