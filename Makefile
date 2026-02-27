.PHONY: build install test clean fmt lint run help

# Binary name
BINARY_NAME=keel

# Build variables
VERSION?=dev
LDFLAGS=-ldflags "-X github.com/slice-soft/ss-keel-cli/cmd.version=$(VERSION)"

## help: Display this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "✓ Build complete: ./$(BINARY_NAME)"

## install: Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) .
	@echo "✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

## test: Run tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "✓ Tests complete"

## coverage: Show test coverage
coverage: test
	@go tool cover -html=coverage.out

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Formatting complete"

## lint: Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run
	@echo "✓ Linting complete"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out
	@rm -rf dist/
	@echo "✓ Clean complete"

## run: Run the application
run: build
	@./$(BINARY_NAME)

## dev: Run in development mode
dev:
	@go run . $(ARGS)
