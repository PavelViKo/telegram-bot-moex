.PHONY: build run test clean deps lint docker-build docker-run docker-stop

# Go параметры
BINARY_NAME=bot
BUILD_DIR=build
CONFIG_DIR=configs

# Цели разработки
build:
	@echo "Building..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/bot

run: build
	@echo "Running..."
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	@echo "Running tests..."
	go test ./... -v

clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	go clean

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

lint:
	@echo "Linting..."
	golangci-lint run

# Docker команды
docker-build:
	@echo "Building Docker image..."
	docker build -t moex-telegram-bot:latest .

docker-run:
	@echo "Running with Docker Compose..."
	docker-compose up -d

docker-stop:
	@echo "Stopping Docker Compose..."
	docker-compose down

docker-logs:
	@echo "Showing logs..."
	docker-compose logs -f

# Утилиты
generate-config:
	@echo "Generating config from example..."
	cp $(CONFIG_DIR)/config.yaml.example $(CONFIG_DIR)/config.yaml
	@echo "Please edit $(CONFIG_DIR)/config.yaml with your settings"

setup:
	@echo "Setting up project..."
	make deps
	make generate-config
	@echo "Setup complete! Don't forget to:"
	@echo "1. Edit configs/config.yaml"
	@echo "2. Set TELEGRAM_TOKEN environment variable"
	@echo "3. Run 'make docker-run' to start"

help:
	@echo "Available commands:"
	@echo "  build          - Build the application"
	@echo "  run            - Build and run locally"
	@echo "  test           - Run tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download dependencies"
	@echo "  lint           - Run linter"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run with Docker Compose"
	@echo "  docker-stop    - Stop Docker Compose"
	@echo "  docker-logs    - Show Docker logs"
	@echo "  generate-config - Generate config from example"
	@echo "  setup          - Setup project"