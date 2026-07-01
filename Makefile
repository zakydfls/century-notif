SHELL := /bin/sh

# Load .env for migrate targets (app uses viper; make does not unless included here).
ifneq (,$(wildcard .env))
include .env
endif

GOOSE := go run github.com/pressly/goose/v3/cmd/goose@latest
MIGRATION_DIR := migrations
DB_DRIVER := postgres

.PHONY: test vet staticcheck build verify dev migrate-up migrate-down migrate-status migrate-redo migrate-create

dev:
	air

test:
	go test ./...

vet:
	go vet ./...

staticcheck:
	staticcheck ./...

build:
	go build ./cmd/http ./cmd/mq ./cmd/migration

verify: test vet staticcheck build

migrate-up:
	@test -n "$(POSTGRES_DSN)" || (echo "POSTGRES_DSN is required (.env or export)"; exit 1)
	$(GOOSE) -dir $(MIGRATION_DIR) $(DB_DRIVER) "$(POSTGRES_DSN)" up

migrate-down:
	@test -n "$(POSTGRES_DSN)" || (echo "POSTGRES_DSN is required (.env or export)"; exit 1)
	$(GOOSE) -dir $(MIGRATION_DIR) $(DB_DRIVER) "$(POSTGRES_DSN)" down

migrate-status:
	@test -n "$(POSTGRES_DSN)" || (echo "POSTGRES_DSN is required (.env or export)"; exit 1)
	$(GOOSE) -dir $(MIGRATION_DIR) $(DB_DRIVER) "$(POSTGRES_DSN)" status

migrate-redo:
	@test -n "$(POSTGRES_DSN)" || (echo "POSTGRES_DSN is required (.env or export)"; exit 1)
	$(GOOSE) -dir $(MIGRATION_DIR) $(DB_DRIVER) "$(POSTGRES_DSN)" redo

migrate-create:
	@test -n "$(name)" || (echo "usage: make migrate-create name=create_users"; exit 1)
	$(GOOSE) -dir $(MIGRATION_DIR) create $(name) sql
