# Task Manager - Makefile

.PHONY: all build run test clean docker-up docker-down migrate generate lint fmt help

# Variables
APP_NAME := task-manager
BUILD_DIR := ./bin
MAIN_PATH := ./cmd/server
DOCKER_COMPOSE := docker-compose

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

# Run the application
run:
	@echo "Running $(APP_NAME)..."
	go run $(MAIN_PATH)

# Run with hot reload
dev:
	@echo "Starting development server with hot reload..."
	air

# Run all tests
test:
	@echo "Running tests..."
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Start Docker services
docker-up:
	@echo "Starting Docker services..."
	$(DOCKER_COMPOSE) up -d

# Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	$(DOCKER_COMPOSE) down

# Run database migrations
migrate-up:
	@echo "Running migrations..."
	$(DOCKER_COMPOSE) run --rm migrate

# Rollback database migrations
migrate-down:
	@echo "Rolling back migrations..."
	migrate -database "postgres://postgres:postgres@localhost:5433/taskmanager?sslmode=disable" -path migrations down 1

# Generate SQLC code
generate-sqlc:
	@echo "Generating SQLC code..."
	sqlc generate

# Generate GraphQL code
generate-graphql:
	@echo "Generating GraphQL code..."
	go run github.com/99designs/gqlgen generate

# Generate gRPC code
generate-grpc:
	@echo "Generating gRPC code..."
	protoc --go_out=. --go-grpc_out=. internal/transport/grpc/proto/*.proto

# Generate Swagger documentation
generate-swagger:
	@echo "Generating Swagger documentation..."
	swag init -g cmd/server/main.go -o docs

# Generate all code
generate: generate-sqlc generate-swagger
	@echo "Code generation complete!"

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/air-verse/air@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/99designs/gqlgen@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Full setup for new developers
setup: install-tools docker-up migrate-up generate
	@echo "Setup complete! Run 'make dev' to start development server."

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build the application"
	@echo "  run             - Run the application"
	@echo "  dev             - Run with hot reload (air)"
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  clean           - Clean build artifacts"
	@echo "  docker-up       - Start Docker services"
	@echo "  docker-down     - Stop Docker services"
	@echo "  migrate-up      - Run database migrations"
	@echo "  migrate-down    - Rollback last migration"
	@echo "  generate-sqlc   - Generate SQLC code"
	@echo "  generate-graphql- Generate GraphQL code"
	@echo "  generate-grpc   - Generate gRPC code"
	@echo "  generate-swagger- Generate Swagger docs"
	@echo "  generate        - Generate all code"
	@echo "  lint            - Run linter"
	@echo "  fmt             - Format code"
	@echo "  install-tools   - Install dev tools"
	@echo "  setup           - Full setup for new developers"
	@echo "  help            - Show this help"

# Default target
all: build
