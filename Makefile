# Legible Money - Go Chain Development Makefile

BINARY := legbled
BUILD_DIR := build

# Colors
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m

.PHONY: all build test vet clean help pr-check devnet devnet-reset devnet-stop devnet-health test-tx-lifecycle test-e2e test-openclaw test-openclaw-phase-a test-openclaw-phase-b test-openclaw-phase-c test-claw test-claw-oc test-claw-nc test-claw-zc test-claw-catalog release changelog classify-pr sync-teams

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

# === OpenClaw Agent Tests ===

test-openclaw: build ## Run all OpenClaw agent integration tests
	@echo "$(GREEN)Running OpenClaw agent tests...$(NC)"
	@go test ./tests/openclaw/... -tags openclaw -v -count=1 -timeout 600s

test-openclaw-phase-a: build ## Run OpenClaw Phase A (lifecycle) only
	@go test ./tests/openclaw/... -tags openclaw -run TestPhaseA -v -count=1 -timeout 300s

test-openclaw-phase-b: build ## Run OpenClaw Phase B (adversarial) only
	@go test ./tests/openclaw/... -tags openclaw -run TestPhaseB -v -count=1 -timeout 300s

test-openclaw-phase-c: build ## Run OpenClaw Phase C (coordination) only
	@go test ./tests/openclaw/... -tags openclaw -run TestPhaseC -v -count=1 -timeout 300s

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

classify-pr: ## Classify PR risk tier (R0-R3)
	@bash scripts/ci/classify-pr.sh

# === Release ===

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

release: ## Build release binaries for all platforms
	@echo "$(GREEN)Building release $(VERSION)...$(NC)"
	@mkdir -p $(BUILD_DIR)/release
	@GOOS=linux  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/release/$(BINARY)-$(VERSION)-linux-amd64 ./cmd/legbled
	@GOOS=linux  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/release/$(BINARY)-$(VERSION)-linux-arm64 ./cmd/legbled
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/release/$(BINARY)-$(VERSION)-darwin-amd64 ./cmd/legbled
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/release/$(BINARY)-$(VERSION)-darwin-arm64 ./cmd/legbled
	@cd $(BUILD_DIR)/release && for f in $(BINARY)-*; do shasum -a 256 "$$f" > "$$f.sha256"; done
	@echo "$(GREEN)Release binaries in $(BUILD_DIR)/release/$(NC)"

changelog: ## Generate changelog from git history
	@echo "## Changelog"
	@echo ""
	@PREV_TAG=$$(git tag --sort=-version:refname | head -1); \
	if [ -n "$$PREV_TAG" ]; then \
		echo "Changes since $$PREV_TAG:"; \
		echo ""; \
		git log --pretty=format:"- %s (%h)" "$$PREV_TAG..HEAD"; \
	else \
		echo "All changes:"; \
		echo ""; \
		git log --pretty=format:"- %s (%h)" | head -50; \
	fi

# === Governance ===

sync-teams: ## Sync known-agents.json to GitHub teams
	@bash scripts/governance/sync-teams.sh

# === Cleanup ===

clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning...$(NC)"
	@rm -rf $(BUILD_DIR)

# === Help ===

help: ## Show this help
	@echo "Legible Money (Go Chain) - Development Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
