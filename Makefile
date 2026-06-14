.PHONY: run build test test-short cover lint sqlc-gen redis-up redis-down

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./... -v

test-short:
	go test ./... -short -v

# cover: 全体カバレッジを計測し 80% 未満なら fail する。
# sqlc 生成コード（db/generated）はテスト対象外（rules/sqlc.md）のため計測から除外する。
cover:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	@grep -v "db/generated" coverage.out > coverage.filtered.out
	@go tool cover -func=coverage.filtered.out | tail -1
	@COV=$$(go tool cover -func=coverage.filtered.out | tail -1 | awk '{print $$3}' | tr -d %); \
		awk -v c="$$COV" 'BEGIN{exit !(c+0<80)}' && { echo "coverage $$COV% < 80%"; exit 1; } || echo "coverage $$COV% OK (>= 80%)"

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
