# Build variables
APP_NAME := web-security-proxy
MAIN_PKG := ./cmd/server
BUILD_DIR := bin
COVERAGE_DIR := coverage

.PHONY: all build run test test-cover clean docker-build docker-up docker-down lint fmt help

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PKG)

run: build
	./$(BUILD_DIR)/$(APP_NAME)

test:
	go test ./... -count=1

test-cover:
	@mkdir -p $(COVERAGE_DIR)
	go test ./... -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic
	go tool cover -func=$(COVERAGE_DIR)/coverage.out

clean:
	rm -rf $(BUILD_DIR) $(COVERAGE_DIR) data

fmt:
	go fmt ./...

lint:
	go vet ./...

docker-build:
	docker compose build

docker-up:
	docker compose up --build

docker-down:
	docker compose down

help:
	@echo "Targets:"
	@echo "  build       - Build binary"
	@echo "  run         - Build and run locally"
	@echo "  test        - Run unit tests"
	@echo "  test-cover  - Run tests with coverage report"
	@echo "  docker-up   - Start with Docker Compose"
	@echo "  docker-down - Stop Docker Compose"
