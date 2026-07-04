DATABASE_URL ?= postgres://relay:relay@localhost:5433/relay?sslmode=disable
BINARY = bin/relay

.PHONY: build run test test-v test-race docker-up docker-down migrate-up migrate-down migrate-reset dev sqlc

## Build
build:
	go build -o $(BINARY) ./cmd/relay

## Test
test:
	go test ./...

test-v:
	go test -v ./...

test-race:
	go test -race ./...

## Docker
docker-up:
	docker-compose up -d db

docker-down:
	docker-compose down

## Migrations
migrate-up:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DATABASE_URL)" down 1

migrate-reset:
	migrate -path db/migrations -database "$(DATABASE_URL)" drop -f
	migrate -path db/migrations -database "$(DATABASE_URL)" up

## Generate sqlc
sqlc:
	sqlc generate

## Dev: start DB → migrate → run server
dev: docker-up migrate-up
	DATABASE_URL=$(DATABASE_URL) go run ./cmd/relay
