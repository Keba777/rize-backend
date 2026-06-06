.PHONY: dev build tidy docker-up docker-down

dev:
	go run ./cmd/server

build:
	go build -o server ./cmd/server

tidy:
	go mod tidy

docker-up:
	docker compose up -d

docker-down:
	docker compose down
