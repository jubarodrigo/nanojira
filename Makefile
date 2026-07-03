.PHONY: run test lint mocks docker-up docker-down build tidy migrate-up migrate-down migrate-status

run:
	go run ./cmd/api

build:
	go build -o bin/nanojira ./cmd/api

test:
	go test ./... -count=1

lint:
	golangci-lint run ./...

mocks:
	go generate ./...

tidy:
	go mod tidy

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api
