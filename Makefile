.PHONY: run build test seeder docker-up docker-down clean

run:
	go run cmd/api/main.go

seeder:
	go run cmd/seeder/main.go

build:
	go build -o bin/dexta-api cmd/api/main.go
	go build -o bin/dexta-seeder cmd/seeder/main.go

test:
	go test -v -race ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down

clean:
	rm -rf bin/

