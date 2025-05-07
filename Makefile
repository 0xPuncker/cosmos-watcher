.PHONY: build run clean deps test test-coverage test-integration lint mod-tidy setup test-slack help

# Default target
.DEFAULT_GOAL := help

# Go related variables
BINARY_NAME=cosmos-watcher
GO_FILES=$(shell find . -type f -name '*.go')

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) ./cmd/server

# Run the application
run:
	@go run cmd/server/main.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@go clean

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download

# Run all tests
test:
	@echo "Running tests..."
	@go test ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -tags=integration ./...

# Run linters
lint:
	@echo "Running linters..."
	@golangci-lint run
	@go vet ./...
	@staticcheck ./...

# Go mod tidy
mod-tidy:
	@echo "Tidying up Go modules..."
	@go mod tidy

# Setup initial configuration
setup:
	@echo "Setting up configuration..."
	@if [ ! -f config/config.json ]; then \
		cp config/config.json.example config/config.json; \
		echo "Created config/config.json from example"; \
	fi

# Run manual Slack notification test
test-slack:
	@echo "Testing Slack notifications..."
	@go test -v ./internal/notifications -run TestSlackNotificationManual

# Help target
help:
	@echo "Available targets:"
	@echo "  build           	- Build the application"
	@echo "  run             	- Run the application"
	@echo "  clean           	- Clean build artifacts"
	@echo "  deps            	- Install dependencies"
	@echo "  test            	- Run all tests"
	@echo "  test-coverage   	- Run tests with coverage"
	@echo "  test-integration	- Run integration tests"
	@echo "  lint            	- Run linters"
	@echo "  setup           	- Setup initial configuration"
	@echo "  mod-tidy        	- Tidy up Go modules"
	@echo "  test-slack      	- Test Slack notifications"
	@echo "  help            	- Show this help message" 