DATABASE_URL ?= postgres://acoustic:acoustic@localhost:5432/ufo_observation_command?sslmode=disable

.PHONY: test race vet build run compose-up compose-down measure

test:
	DATABASE_URL=$(DATABASE_URL) go test ./... -count=1

race:
	DATABASE_URL=$(DATABASE_URL) go test -race ./... -count=1

vet:
	go vet ./...

build:
	go build ./...

run:
	DATABASE_URL=$(DATABASE_URL) go run ./cmd/server

compose-up:
	docker compose up -d postgres

compose-down:
	docker compose down

measure:
	go run ../../.agents/skills/go-base-project-create/scripts/measure_project.go -root . -enforce
