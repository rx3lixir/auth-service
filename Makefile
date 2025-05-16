# ============================================================================
# CONFIGURATION
# ============================================================================

BINARY_NAME=auth-service

# Auto-load .env file if it exists
ifneq (,$(wildcard .env))
	include .env
	export
endif

# ============================================================================
# DEFAULT TARGET
# ============================================================================

all: build ## Default build command

# ============================================================================
# BUILD & RUN
# ============================================================================

build: ## Build the binary
	@echo "🔨 Building..."
	go build -o ./bin/$(BINARY_NAME) ./cmd/auth/main.go

run: build ## Build and run the app
	@echo "🚀 Running..."
	./bin/$(BINARY_NAME)

clean: ## Clean binary
	@echo "🧹 Cleaning..."
	go clean
	rm -f ./bin/$(BINARY_NAME)

# ============================================================================
# PROTO
# ============================================================================
proto-gen: ## Generate protobuf
	@echo "🧪  Generating protobuf..."
	protoc \
  --proto_path=auth-grpc/proto \
  --go_out=auth-grpc/gen/go \
  --go_opt=paths=source_relative \
  --go-grpc_out=auth-grpc/gen/go \
  --go-grpc_opt=paths=source_relative \
  auth-grpc/proto/auth-service.proto

# ============================================================================
# TESTING
# ============================================================================

test: ## Run all tests with coverage
	@echo "🧪 Running tests..."
	go test -cover ./...

# ============================================================================
# UTILITIES
# ============================================================================

help: ## Show help
	@echo "📖 Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}'
