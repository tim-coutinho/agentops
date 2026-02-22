.PHONY: build test lint clean install sync-hooks

# Build variables
BINARY_NAME=ao
BUILD_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default target
all: build

# Sync hooks, scripts, and skills into cli/embedded/ for Go embed
sync-hooks:
	@mkdir -p embedded/hooks embedded/lib embedded/skills/standards/references embedded/skills/using-agentops
	@cp ../hooks/*.sh ../hooks/hooks.json embedded/hooks/
	@cp ../lib/hook-helpers.sh embedded/lib/
	@cp ../skills/standards/references/* embedded/skills/standards/references/
	@cp ../skills/using-agentops/SKILL.md embedded/skills/using-agentops/SKILL.md

# Build the binary
build: sync-hooks
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ao

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Install to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/ao

# Run the binary
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Tidy go modules
tidy:
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Generate (for future code generation)
generate:
	go generate ./...
