APP_PORT ?= 8080
POSTGRES_DSN ?= postgres://app_user:app_password@localhost:5432/orderdb?sslmode=disable

.PHONY: up down logs run tidy test loadtest loadtest-smoke

up:
	docker compose up --build -d db api

down:
	docker compose down

logs:
	docker compose logs -f api db

run:
	APP_PORT=$(APP_PORT) POSTGRES_DSN=$(POSTGRES_DSN) go run ./cmd/api

tidy:
	go mod tidy

test:
	go test ./...

loadtest:
	docker compose --profile loadtest run --rm --no-deps k6 run /scripts/scenario.js

loadtest-smoke:
	docker compose --profile loadtest run --rm --no-deps -e TEST_PROFILE=smoke k6 run /scripts/scenario.js
