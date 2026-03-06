.PHONY: build test lint clean run install

# Go commands
GO := go
GOFLAGS := 
BINARY_NAME := relay
BUILD_DIR := bin

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/relay

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Lint the code
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || echo "golangci-lint not installed. Run: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"
	@which golangci-lint > /dev/null && golangci-lint run ./... || echo "Skipping lint (golangci-lint not found)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Run the server (for development)
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) server --port 8080 --admin-token dev-token

# Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install ./cmd/relay

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	$(GO) vet ./...

# Run all checks
check: fmt vet test

# Help
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary"
	@echo "  test       - Run tests"
	@echo "  coverage   - Run tests with coverage report"
	@echo "  lint       - Lint the code"
	@echo "  clean      - Clean build artifacts"
	@echo "  run        - Build and run the server (dev mode)"
	@echo "  install    - Install binary to GOPATH/bin"
	@echo "  deps       - Download dependencies"
	@echo "  fmt        - Format code"
	@echo "  vet        - Vet code"
	@echo "  check      - Run fmt, vet, and test"
	@echo "  help       - Show this help message"
