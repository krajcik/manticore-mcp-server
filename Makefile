.PHONY: test test-unit test-integration test-up test-down build build-all lint fmt generate check clean run deps release help

BINARY_NAME=manticore-mcp-server
VERSION?=latest
BUILD_DIR=build

# Run all tests
test: test-unit test-integration

# Run unit tests only
test-unit:
	go test -v -short ./...

# Run integration tests with Manticore
test-integration: test-up
	@echo "Waiting for Manticore to be ready..."
	@timeout 60 sh -c 'until docker compose -f docker-compose.test.yml exec manticore curl -f http://localhost:9308/sql -d "{\"query\":\"SHOW STATUS\"}" > /dev/null 2>&1; do sleep 1; done' || (echo "Manticore failed to start" && exit 1)
	go test -v ./client -run TestClientIntegration
	@$(MAKE) test-down

# Start test containers
test-up:
	docker compose -f docker-compose.test.yml up -d --wait

# Stop test containers
test-down:
	docker compose -f docker-compose.test.yml down -v

# Build for current platform
build:
	go build -o $(BINARY_NAME) .

# Build for all platforms
build-all: clean
	mkdir -p $(BUILD_DIR)
	# Linux
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	# macOS
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	# Windows
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe .

# Run linter
lint:
	golangci-lint run ./... --fix

# Run linter (CI mode - no fixes)
lint-ci:
	golangci-lint run ./...

# Format code
fmt:
	gofmt -s -w .

# Check formatting (CI mode - no fixes)
fmt-check:
	@if [ -n "$$(gofmt -s -l .)" ]; then \
		echo "Code is not formatted. Run 'make fmt' to fix:"; \
		gofmt -s -l .; \
		exit 1; \
	fi

# Generate code (mocks, etc.)
generate:
	go generate ./...

# Run all checks
check: fmt generate lint test

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)

# Development run
run:
	go run .

# Install dependencies
deps:
	go mod download
	go mod tidy

# Release builds with compression
release: build-all
	cd $(BUILD_DIR) && \
	for file in *; do \
		if [[ $$file == *.exe ]]; then \
			zip $${file%.exe}.zip $$file; \
		else \
			tar -czf $$file.tar.gz $$file; \
		fi; \
	done

# Run tests with coverage
test-coverage:
	go test -cover ./...

help:
	@echo "Available commands:"
	@echo "  build        - Build for current platform"
	@echo "  build-all    - Build for all platforms (Linux, macOS, Windows)"
	@echo "  test         - Run all tests (unit + integration)"
	@echo "  test-unit    - Run unit tests only"
	@echo "  test-integration - Run integration tests with Manticore"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter with --fix"
	@echo "  generate     - Generate mocks"
	@echo "  check        - Run all quality checks (fmt, generate, lint, test)"
	@echo "  clean        - Clean build artifacts"
	@echo "  run          - Run in development mode"
	@echo "  deps         - Install and tidy dependencies"
	@echo "  release      - Build all platforms and create archives"
	@echo "  help         - Show this help"