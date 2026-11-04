.PHONY: run build vet lint tidy migrate-up migrate-down \
        dev dev-down prod prod-down docker-build-dev docker-build-prod clean

run:
	go run ./cmd/api

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/api ./cmd/api

vet:
	go vet ./...

lint:
	go vet ./...

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

clean:
	rm -rf bin/
