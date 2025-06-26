.PHONY: test test-unit test-integration test-up test-down build lint fmt generate

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

# Build the project
build:
	go build ./...

# Run linter
lint:
	golangci-lint run ./...

# Format code
fmt:
	gofmt -s -w .

# Generate code (mocks, etc.)
generate:
	go generate ./...

# Run all checks
check: build test lint