# Legible Money - Go Chain Development Makefile

BINARY := legbled
BUILD_DIR := build

# Colors
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m

.PHONY: all build test vet clean help pr-check devnet devnet-reset devnet-stop devnet-health test-tx-lifecycle test-e2e

# Default target
all: build test

# === Build ===

build: ## Build legbled binary
	@echo "$(GREEN)Building $(BINARY)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY) ./cmd/legbled

build-all: ## Build all packages
	@echo "$(GREEN)Building all packages...$(NC)"
	@go build ./...

# === Test ===

test: ## Run all module tests
	@echo "$(GREEN)Running Go tests...$(NC)"
	@go test ./x/... -count=1

test-v: ## Run tests with verbose output
	@go test ./x/... -count=1 -v

test-app: ## Run app-level tests
	@go test ./app/... -count=1 -v

test-all: test test-app ## Run all tests (modules + app)

# === Code Quality ===

vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(NC)"
	@go vet ./...

# === DevNet ===

devnet: ## Start Cosmos SDK devnet
	@bash devnet/scripts/cosmos-devnet.sh start

devnet-reset: ## Reset devnet data
	@bash devnet/scripts/cosmos-devnet.sh reset

devnet-stop: ## Stop devnet
	@bash devnet/scripts/cosmos-devnet.sh stop

devnet-health: ## Run devnet health check
	@bash devnet/scripts/devnet-healthcheck.sh --full

test-tx-lifecycle: ## Run tx lifecycle tests against live devnet
	@bash devnet/scripts/test-tx-lifecycle.sh

test-e2e: devnet devnet-health test-tx-lifecycle devnet-stop ## Full E2E: start devnet, test, stop

# === PR Validation ===

pr-check: build-all test vet ## Run full PR validation
	@echo "$(GREEN)All PR checks passed!$(NC)"

# === Cleanup ===

clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning...$(NC)"
	@rm -rf $(BUILD_DIR)

# === Help ===

help: ## Show this help
	@echo "Legible Money (Go Chain) - Development Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
