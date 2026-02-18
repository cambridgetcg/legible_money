# Legible Money - Go Chain Development Makefile

BINARY := legbled
BUILD_DIR := build

# Version metadata (injected via ldflags)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0-dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TAGS := netgo,ledger

LDFLAGS := -X github.com/cosmos/cosmos-sdk/version.Name=legible \
           -X github.com/cosmos/cosmos-sdk/version.AppName=legbled \
           -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
           -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
           -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(BUILD_TAGS)"

# Colors
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m

.PHONY: all build test vet clean help pr-check devnet devnet-reset devnet-stop devnet-health test-tx-lifecycle test-e2e test-openclaw test-openclaw-devnet test-openclaw-testnet test-openclaw-report test-openclaw-phase-a test-openclaw-phase-b test-openclaw-phase-c test-openclaw-phase-d test-claw test-claw-oc test-claw-nc test-claw-zc test-claw-catalog

# Default target
all: build test

# === Build ===

build: ## Build legbled binary
	@echo "$(GREEN)Building $(BINARY)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags "$(LDFLAGS) -s -w" -o $(BUILD_DIR)/$(BINARY) ./cmd/legbled

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

# === OpenClaw Agent Tests ===

test-openclaw: build ## Run all OpenClaw agent integration tests (alias for devnet)
	@echo "$(GREEN)Running OpenClaw agent tests...$(NC)"
	@go test ./tests/openclaw/... -tags openclaw -v -count=1 -timeout 600s

test-openclaw-devnet: build ## Run OpenClaw on single-validator devnet
	@echo "$(GREEN)Running OpenClaw tests (devnet)...$(NC)"
	@LGM_ENV=devnet go test ./tests/openclaw/... -tags openclaw -v -count=1 -timeout 600s

test-openclaw-testnet: build ## Run OpenClaw on 3-validator testnet
	@echo "$(GREEN)Running OpenClaw tests (testnet, 3 validators)...$(NC)"
	@LGM_ENV=testnet go test ./tests/openclaw/... -tags openclaw -v -count=1 -timeout 900s

test-openclaw-report: build ## Run OpenClaw with JSON report
	@echo "$(GREEN)Running OpenClaw tests with JSON report...$(NC)"
	@LGM_REPORT_PATH=tests/openclaw/report.json go test ./tests/openclaw/... \
		-tags openclaw -v -count=1 -timeout 600s

test-openclaw-phase-a: build ## Run OpenClaw Phase A (lifecycle) only
	@go test ./tests/openclaw/... -tags openclaw -run TestPhaseA -v -count=1 -timeout 300s

test-openclaw-phase-b: build ## Run OpenClaw Phase B (adversarial) only
	@go test ./tests/openclaw/... -tags openclaw -run TestPhaseB -v -count=1 -timeout 300s

test-openclaw-phase-c: build ## Run OpenClaw Phase C (coordination) only
	@go test ./tests/openclaw/... -tags openclaw -run TestPhaseC -v -count=1 -timeout 300s

test-openclaw-phase-d: build ## Run OpenClaw Phase D (agent) only
	@go test ./tests/openclaw/... -tags openclaw -run TestPhaseD -v -count=1 -timeout 600s

# === Claw Adversarial Tests (unit-level, no devnet) ===

test-claw: ## Run all Claw adversarial probes
	@echo "$(GREEN)Running all Claw probes...$(NC)"
	@go test ./x/... -run "TestOC|TestNC|TestZC|TestXM|TestOCP|TestPSY|TestFarm|TestOCA|TestMetaLoop" -v -count=1

test-claw-oc: ## Run OpenClaw (macro game theory) probes
	@go test ./x/... -run "TestOC|TestOCP|TestOCA|TestMetaLoop" -v -count=1

test-claw-nc: ## Run NanoClaw (precision attacks) probes
	@go test ./x/... -run "TestNC" -v -count=1

test-claw-zc: ## Run ZeroClaw (zero-cost attacks) probes
	@go test ./x/... -run "TestZC|TestFarm" -v -count=1

test-claw-catalog: ## Run Claw catalog validation
	@go test ./x/testutil/claw/... -v -count=1

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
