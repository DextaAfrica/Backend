.PHONY: run build test vet lint tidy migrate-up migrate-down \
        dev dev-down prod prod-down docker-build-dev docker-build-prod

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

# Local development: hot reload via air, source bind-mounted, Postgres port exposed.
dev:
	docker compose -f docker-compose.dev.yml up --build

dev-down:
	docker compose -f docker-compose.dev.yml down

# Production-shaped local run: the same distroless image used in deployment.
prod:
	docker compose up --build

prod-down:
	docker compose down

docker-build-dev:
	docker build -f Dockerfile.dev -t dexta-backend:dev .

docker-build-prod:
	docker build -f Dockerfile -t dexta-backend:latest .
