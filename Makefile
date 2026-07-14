# RepWire — developer Makefile.
# Requires: Go 1.26+, Docker, and (for migrate targets) the `migrate` CLI or Docker.

DATABASE_URL ?= postgres://repwire:repwire@localhost:5432/repwire?sslmode=disable
MIGRATE_IMAGE := migrate/migrate:v4.17.1

.PHONY: help
help:
	@echo "RepWire targets:"
	@echo "  build          Build api + worker binaries into ./bin"
	@echo "  run-api        Run the API server locally"
	@echo "  run-worker     Run the worker locally"
	@echo "  vet            go vet ./..."
	@echo "  test           go test ./..."
	@echo "  tidy           go mod tidy"
	@echo "  migrate-up     Apply all migrations (via Docker)"
	@echo "  migrate-down   Roll back one migration (via Docker)"
	@echo "  migrate-new    Create a new migration pair: make migrate-new NAME=add_x"
	@echo "  db-up          Start only Postgres via Docker Compose"
	@echo "  up             Build + start the full stack (Docker Compose)"
	@echo "  down           Stop the stack"
	@echo "  logs           Tail api + worker logs"
	@echo "  admin          Promote a user to admin: make admin EMAIL=you@example.com"

# ---- Go ----

.PHONY: build
build:
	go build -o bin/api ./apps/api
	go build -o bin/worker ./apps/worker

.PHONY: run-api
run-api:
	go run ./apps/api

.PHONY: run-worker
run-worker:
	go run ./apps/worker

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test ./...

.PHONY: tidy
tidy:
	go mod tidy

# ---- Migrations (Dockerised migrate CLI; no local install needed) ----

.PHONY: migrate-up
migrate-up:
	docker run --rm --network host -v "$(CURDIR)/migrations:/migrations" $(MIGRATE_IMAGE) \
		-path=/migrations -database "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down:
	docker run --rm --network host -v "$(CURDIR)/migrations:/migrations" $(MIGRATE_IMAGE) \
		-path=/migrations -database "$(DATABASE_URL)" down 1

.PHONY: migrate-new
migrate-new:
	docker run --rm -v "$(CURDIR)/migrations:/migrations" $(MIGRATE_IMAGE) \
		create -ext sql -dir /migrations -seq $(NAME)

# ---- Docker Compose ----

.PHONY: db-up
db-up:
	docker compose -f deploy/docker-compose.yml --env-file .env up -d postgres

.PHONY: up
up:
	docker compose -f deploy/docker-compose.yml --env-file .env up -d --build

.PHONY: down
down:
	docker compose -f deploy/docker-compose.yml down

.PHONY: logs
logs:
	docker compose -f deploy/docker-compose.yml logs -f api worker

# ---- Ops helpers ----

.PHONY: admin
admin:
	@test -n "$(EMAIL)" || (echo "usage: make admin EMAIL=you@example.com" && exit 1)
	docker compose -f deploy/docker-compose.yml exec -T postgres \
		psql -U repwire -d repwire -c "UPDATE users SET role='admin' WHERE email='$(EMAIL)';"
