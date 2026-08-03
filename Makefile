.PHONY: run build test vet lint tidy migrate-up migrate-down docker-build docker-up docker-down

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test ./... -race -cover

vet:
	go vet ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

migrate-up:
	migrate -path internal/db/migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path internal/db/migrations -database "$$DATABASE_URL" down 1

docker-build:
	docker build -t dexta-backend .

docker-up:
	docker compose up --build

docker-down:
	docker compose down
