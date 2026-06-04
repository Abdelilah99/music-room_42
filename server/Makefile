.PHONY: install run build test migrate migrate-down docker-up docker-down deps docs

install:
	go mod download

run:
	go run ./cmd/main.go

build:
	go build -o bin/server ./cmd/main.go

test:
	go test ./...

migrate:
	go run ./cmd/migrate/main.go up

migrate-down:
	go run ./cmd/migrate/main.go down

docker-up:
	docker compose up --build

docker-down:
	docker compose down

deps:
	docker compose run --rm server go get $(pkg) && docker compose run --rm server go mod tidy

docs:
	go run github.com/swaggo/swag/cmd/swag@v1.8.12 init -g cmd/main.go -o docs --parseDependency --parseInternal
