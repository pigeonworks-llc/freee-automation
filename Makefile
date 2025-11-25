.PHONY: help emulator-build emulator-openapi emulator-dev emulator-clean \
        freee-sync-build freee-sync-run freee-sync-clean go-build-all go-clean \
        ts-install ts-build ts-clean generate-types build-all clean-all

# Default target
help:
	@echo "Accounting System - Makefile"
	@echo ""
	@echo "Go:"
	@echo "  make emulator-build      - Build freee-emulator binary"
	@echo "  make emulator-openapi    - Generate OpenAPI specification"
	@echo "  make emulator-dev        - Run emulator in development mode"
	@echo "  make freee-sync-build    - Build freee-sync binary"
	@echo "  make freee-sync-run      - Run freee-sync"
	@echo "  make go-build-all        - Build all Go binaries"
	@echo "  make go-clean            - Clean all Go build artifacts"
	@echo ""
	@echo "TypeScript:"
	@echo "  make ts-install          - Install npm dependencies"
	@echo "  make ts-build            - Build all TypeScript workspaces"
	@echo "  make ts-clean            - Clean TypeScript build artifacts"
	@echo ""
	@echo "Type Generation:"
	@echo "  make generate-types      - Generate TypeScript types from OpenAPI"
	@echo ""
	@echo "All:"
	@echo "  make build-all           - Build everything"
	@echo "  make clean-all           - Clean everything"

# =============================================================================
# Emulator (Go)
# =============================================================================

emulator-build:
	@echo "🔨 Building freee-emulator..."
	cd emulator && go build -o bin/freee-emulator ./cmd/server
	@echo "✅ Built: emulator/bin/freee-emulator"

emulator-openapi:
	@echo "📝 Generating OpenAPI specification..."
	cd emulator && swag init -g cmd/server/main.go -o docs/openapi --parseDependency --parseInternal
	@echo "✅ Generated: emulator/docs/openapi/swagger.yaml"

emulator-dev: emulator-openapi
	@echo "🚀 Running emulator in development mode..."
	cd emulator && go run ./cmd/server

emulator-clean:
	@echo "🧹 Cleaning emulator build artifacts..."
	rm -rf emulator/bin
	@echo "✅ Cleaned emulator artifacts"

freee-sync-build:
	@echo "🔨 Building freee-sync..."
	go build -o bin/freee-sync ./cmd/freee-sync
	@echo "✅ Built: bin/freee-sync"

freee-sync-run:
	@echo "🚀 Running freee-sync..."
	./bin/freee-sync

freee-sync-clean:
	@echo "🧹 Cleaning freee-sync build artifacts..."
	rm -f bin/freee-sync
	@echo "✅ Cleaned freee-sync artifacts"

go-build-all: emulator-build freee-sync-build
	@echo "✅ All Go binaries built successfully"

go-clean: emulator-clean freee-sync-clean
	@echo "✅ All Go artifacts cleaned"

# =============================================================================
# TypeScript
# =============================================================================

ts-install:
	@echo "📦 Installing npm dependencies..."
	npm install
	@echo "✅ Dependencies installed"

ts-build:
	@echo "🔨 Building TypeScript workspaces..."
	npm run build
	@echo "✅ Built all TypeScript workspaces"

ts-clean:
	@echo "🧹 Cleaning TypeScript build artifacts..."
	npm run clean
	rm -rf shared/dist freee-sync/dist amazon-receipt-processor/dist
	rm -rf node_modules shared/node_modules freee-sync/node_modules amazon-receipt-processor/node_modules
	@echo "✅ Cleaned TypeScript artifacts"

# =============================================================================
# Type Generation
# =============================================================================

generate-types: emulator-openapi
	@echo "🔄 Generating TypeScript types from OpenAPI..."
	npm run generate-types
	@echo "✅ Types generated"

# =============================================================================
# Build All
# =============================================================================

build-all: go-build-all ts-install ts-build
	@echo "✅ All components built successfully"

# =============================================================================
# Clean All
# =============================================================================

clean-all: go-clean ts-clean
	@echo "✅ All components cleaned"
