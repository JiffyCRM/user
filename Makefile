.PHONY: generate migrate test run build docker-build version

VERSION ?= $(shell cat VERSION)
LDFLAGS := -X github.com/JiffyCRM/user/internal/version.Version=$(VERSION)
DATABASE_URL ?= postgres://user:user@localhost:5435/user?sslmode=disable

generate:
	./scripts/generate.sh

migrate:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate up

test:
	go test ./...

run:
	go run -ldflags "$(LDFLAGS)" ./cmd/server

build:
	go build -ldflags "$(LDFLAGS)" -o bin/user ./cmd/server

docker-build:
	docker build --build-arg VERSION="$(VERSION)" -t user:$(VERSION) .

version:
	@echo "$(VERSION)"