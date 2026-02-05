# Legible Money - Development Makefile
# Optimized for Apple M4 Max (14 cores, 36GB RAM)

# Configuration
CORES := 10
MEMORY_MB := 27648

# Colors
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m

.PHONY: all build test lint format clean dev help

# Default target
all: build test

# === Build Targets ===

build: ## Build with TypeScript (incremental)
	@echo "$(GREEN)Building with TypeScript (incremental)...$(NC)"
	@npm run build

build-fast: ## Build with tsup (esbuild, fastest)
	@echo "$(GREEN)Building with tsup (esbuild)...$(NC)"
	@npm run build:fast

build-clean: ## Clean build (removes dist first)
	@echo "$(YELLOW)Clean build...$(NC)"
	@npm run build:clean

build-watch: ## Watch mode build
	@npm run build:watch

# === Test Targets ===

test: ## Run all tests in parallel (10 cores)
	@echo "$(GREEN)Running tests on $(CORES) cores...$(NC)"
	@NODE_OPTIONS="--max-old-space-size=$(MEMORY_MB)" npm run test:run

test-watch: ## Run tests in watch mode
	@npm run test:watch

test-coverage: ## Run tests with coverage
	@NODE_OPTIONS="--max-old-space-size=$(MEMORY_MB)" npm run test:coverage

test-unit: ## Run unit tests only
	@npm run test:unit

test-integration: ## Run integration tests only
	@npm run test:integration

# === Code Quality ===

lint: ## Lint code with caching
	@echo "$(GREEN)Linting with cache...$(NC)"
	@npm run lint

lint-fix: ## Lint and auto-fix
	@npm run lint:fix

format: ## Format code with caching
	@npm run format

format-check: ## Check formatting
	@npm run format:check

typecheck: ## Type check only (no emit)
	@npm run typecheck

typecheck-watch: ## Type check in watch mode
	@npm run typecheck:watch

# === Development ===

dev: ## Start development (build + watch + test watch)
	@npx concurrently -n "build,test" -c "blue,green" \
		"npm run build:watch" \
		"npm run test:watch"

devnet: ## Start single devnet node
	@npm run devnet

devnet-multi: ## Start 10-node devnet (uses all P-cores)
	@echo "$(GREEN)Starting 10-node devnet on $(CORES) cores...$(NC)"
	@npm run devnet:multi

# === Performance ===

benchmark: ## Run performance benchmark
	@echo "$(GREEN)Running benchmark on M4 Max...$(NC)"
	@npm run benchmark

perf: ## Performance check (benchmark + typecheck)
	@npm run perf:check

# === Cleanup ===

clean: ## Clean all build artifacts and caches
	@echo "$(YELLOW)Cleaning...$(NC)"
	@npm run clean

clean-all: clean ## Deep clean including node_modules
	@rm -rf node_modules
	@echo "$(YELLOW)Run 'npm install' to reinstall dependencies$(NC)"

# === Setup ===

install: ## Install dependencies
	@npm install

setup: install build ## Full setup (install + build)
	@echo "$(GREEN)Setup complete!$(NC)"

# === Help ===

help: ## Show this help
	@echo "Legible Money Development Commands (M4 Max Optimized)"
	@echo ""
	@echo "Build:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^build' | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Test:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^test' | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Code Quality:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^(lint|format|typecheck)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Development:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^(dev|benchmark|perf)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Other:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^(clean|install|setup|help)' | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
