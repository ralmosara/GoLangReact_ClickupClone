.PHONY: run build migrate fe tidy seed

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

# Runs migrations only, then exits. No external CLI needed.
migrate:
	go run ./cmd/server -migrate

fe:
	cd frontend && npm install && npm run dev

tidy:
	go mod tidy

seed:
	go run ./scripts/seed.go
