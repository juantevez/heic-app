# Makefile
.PHONY: help build run test clean docker-build docker-run docker-down deps

# Default target
help:
	@echo "Available commands:"
	@echo "  build        - Build the application"
	@echo "  run          - Run the application locally"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run with Docker Compose"
	@echo "  docker-down  - Stop Docker Compose"
	@echo "  deps         - Download dependencies"
	@echo "  lint         - Run linter"

# Build the application
build:
	@echo "Building HEIC Photo Processor..."
	CGO_ENABLED=1 go build -o bin/github.com/juantevez/heic-app cmd/server/main.go

# Run the application locally
run:
	@echo "Running HEIC Photo Processor..."
	go run cmd/server/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf uploads/
	go clean

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -t github.com/juantevez/heic-app .

docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

docker-down:
	@echo "Stopping Docker Compose services..."
	docker-compose down

docker-logs:
	@echo "Viewing logs..."
	docker-compose logs -f

# Development commands
dev-setup:
	@echo "Setting up development environment..."
	cp .env.example .env
	mkdir -p uploads
	go mod download

dev-run:
	@echo "Running in development mode..."
	go run cmd/server/main.go

# Database commands
db-migrate:
	@echo "Running database migrations..."
	psql -h localhost -U postgres -d photo_db -f scripts/init.sql

---

