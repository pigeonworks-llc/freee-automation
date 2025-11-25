.PHONY: all build test openapi openapi-fmt clean

# Build configuration
BINARY_NAME=freee-emulator
BUILD_DIR=bin
DOCS_DIR=docs/openapi

all: build

# Build the application
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

# Run tests
test:
	go test -v -race ./...

# Generate OpenAPI specification
openapi:
	@echo "Generating OpenAPI specification..."
	swag init -g cmd/server/main.go -o $(DOCS_DIR) --parseDependency --parseInternal
	@echo "✓ OpenAPI spec generated at $(DOCS_DIR)/swagger.json"

# Format Swagger comments
openapi-fmt:
	swag fmt -g cmd/server/main.go

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(DOCS_DIR)

# Development workflow: format -> generate -> build
dev: openapi-fmt openapi build

# Install dependencies
deps:
	go mod download
	go install github.com/swaggo/swag/cmd/swag@latest
