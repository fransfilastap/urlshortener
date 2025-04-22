# Makefile for URL Shortener

# Variables
BINARY_NAME=urlshortener
MAIN_PACKAGE=.
GO=go
DOCKER=docker
DOCKER_COMPOSE=docker-compose
DOCKER_IMAGE=urlshortener
COVERAGE_FILE=coverage.out
GOARCH=amd64

# Go build flags
LDFLAGS=-ldflags "-s -w"

# Default target
.PHONY: all
all: clean build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Build for multiple platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BINARY_NAME)_linux_$(GOARCH) $(MAIN_PACKAGE)

.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BINARY_NAME)_darwin_$(GOARCH) $(MAIN_PACKAGE)

.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BINARY_NAME)_windows_$(GOARCH).exe $(MAIN_PACKAGE)

# Run the application
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GO) clean
	rm -f $(BINARY_NAME) $(BINARY_NAME)_linux_$(GOARCH) $(BINARY_NAME)_darwin_$(GOARCH) $(BINARY_NAME)_windows_$(GOARCH).exe $(COVERAGE_FILE)

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	$(GO) test -race -v ./...

# Generate test coverage
.PHONY: coverage
coverage:
	@echo "Generating test coverage..."
	$(GO) test -coverprofile=$(COVERAGE_FILE) ./...
	$(GO) tool cover -html=$(COVERAGE_FILE)

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, installing..."; \
		$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy

# Verify dependencies
.PHONY: verify
verify:
	@echo "Verifying dependencies..."
	$(GO) mod verify

# Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	$(DOCKER) build -t $(DOCKER_IMAGE) .

# Run Docker container
.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	$(DOCKER) run -p 8080:8080 $(DOCKER_IMAGE)

# Start Docker Compose services
.PHONY: docker-up
docker-up:
	@echo "Starting Docker Compose services..."
	$(DOCKER_COMPOSE) up -d

# Stop Docker Compose services
.PHONY: docker-down
docker-down:
	@echo "Stopping Docker Compose services..."
	$(DOCKER_COMPOSE) down

# Show help
.PHONY: help
help:
	@echo "URL Shortener Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all             Clean and build the application"
	@echo "  build           Build the application"
	@echo "  build-all       Build for multiple platforms (Linux, macOS, Windows)"
	@echo "  build-linux     Build for Linux"
	@echo "  build-darwin    Build for macOS"
	@echo "  build-windows   Build for Windows"
	@echo "  run             Run the application"
	@echo "  clean           Clean build artifacts"
	@echo "  test            Run tests"
	@echo "  test-race       Run tests with race detection"
	@echo "  coverage        Generate test coverage"
	@echo "  lint            Run linter"
	@echo "  fmt             Format code"
	@echo "  tidy            Tidy dependencies"
	@echo "  verify          Verify dependencies"
	@echo "  docker-build    Build Docker image"
	@echo "  docker-run      Run Docker container"
	@echo "  docker-up       Start Docker Compose services"
	@echo "  docker-down     Stop Docker Compose services"
	@echo "  help            Show this help"