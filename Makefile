.PHONY: run build test lint sqlc-gen redis-up redis-down

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./... -v

test-short:
	go test ./... -short -v

lint:
	golangci-lint run

sqlc-gen:
	sqlc generate

redis-up:
	docker-compose up -d redis

redis-down:
	docker-compose down

migrate:
	psql $$DATABASE_URL -f db/schema.sql
