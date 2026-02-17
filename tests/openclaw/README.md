# OpenClaw Agent Testing Framework

Live-chain integration tests for Legible Money's Proof of Truth (PoT) protocol.

## Test Phases

| Phase | Tests | Focus |
|-------|-------|-------|
| **A** | 10 | Lifecycle — chain boot, claims, verification rounds, facts |
| **B** | 15 | Adversarial — whale attacks, spam, equivocation, DID spoofing |
| **C** | 8 | Coordination — multi-agent workflows, pipelines, heartbeats |
| **D** | 4 | Agent — AI gateway integration (requires OpenClaw gateway) |

**Total: 37 tests**

## Environment Modes

Tests auto-detect the backend via environment variables:

| Mode | Env Var | Validators | Use Case |
|------|---------|------------|----------|
| **devnet** (default) | none | 1 | Fast local development |
| **testnet** | `LGM_ENV=testnet` | 3 | Multi-validator scenarios |
| **external** | `LGM_TESTNET_RPC=...` | N | Pre-existing chain |

### Running Tests

```bash
# Default: single-validator devnet
make test-openclaw

# Explicit devnet
make test-openclaw-devnet

# 3-validator testnet (slower, enables multi-val tests)
make test-openclaw-testnet

# With JSON report
make test-openclaw-report

# Individual phases
make test-openclaw-phase-a
make test-openclaw-phase-b
make test-openclaw-phase-c
make test-openclaw-phase-d  # requires OpenClaw gateway
```

### External Chain

```bash
export LGM_TESTNET_RPC=tcp://localhost:26657
export LGM_TESTNET_REST=http://localhost:1317
export LGM_TESTNET_CHAIN_ID=legible-testnet-1
export LGM_TESTNET_BINARY=/path/to/legbled
export LGM_TESTNET_HOME=/path/to/node-home
go test ./tests/openclaw/... -tags openclaw -v -count=1 -timeout 600s
```

## Multi-Validator Tests

These tests require `LGM_ENV=testnet` (or an external chain with 3+ validators):

| ID | Name | Why Multi-Validator |
|----|------|---------------------|
| B-03 | Abstain-to-Avoid-Slash | Needs verifiers who commit but don't reveal |
| B-06 | Split Verdict Lock | 50/50 split requires independent verifiers |
| B-09 | Quorum Manipulation | Quorum blocking requires colluding minority |
| C-04 | Mixed Verdicts | 3 verifiers with different verdicts |

On single-validator devnet, these tests are automatically skipped.

## Phase D Prerequisites

Phase D tests require a running [OpenClaw](https://github.com/anthropics/openclaw) gateway:

```bash
openclaw serve  # Start gateway in another terminal
make test-openclaw-phase-d
```

The gateway health check runs automatically; tests skip if unavailable.

## JSON Report Format

When `LGM_REPORT_PATH` is set, a structured report is written:

```json
{
  "run_id": "openclaw-1708200000000000000",
  "timestamp": "2026-02-17T20:00:00Z",
  "environment": "devnet",
  "validators": 1,
  "duration": "2m30s",
  "results": [
    {"id": "OCA-A-01", "name": "Chain Boot", "phase": "A", "status": "PASS", "duration": "5s"},
    {"id": "OCA-B-03", "name": "Abstain-to-Avoid-Slash", "phase": "B", "status": "SKIP"}
  ],
  "summary": {"total": 37, "passed": 30, "failed": 0, "skipped": 7}
}
```

## Adding New Tests

1. Add entry to `catalog/catalog.go` with env requirements
2. Add subtest in the appropriate `phase_X_test.go`
3. Use `env.RequireMultiValidator(t, e, N)` for multi-validator tests
4. Use `claw.RequireGateway(t)` for gateway tests
5. Log results via `helpers.Result(t, "X-NN", "outcome")`

## Directory Structure

```
tests/openclaw/
├── main_test.go              # TestMain: build binary, init reporter
├── doc.go                    # Package documentation
├── env/                      # Environment abstraction
│   ├── env.go                # Interface + Detect() + Boot()
│   ├── devnet.go             # SingleNodeEnv (wraps devnet.Harness)
│   ├── testnet.go            # MultiNodeEnv (wraps testnet.MultiHarness)
│   └── external.go           # ExternalEnv (connects to running chain)
├── devnet/                   # Single-validator harness
├── testnet/                  # Multi-validator harness
│   ├── harness.go            # MultiHarness: 3 legbled processes
│   ├── genesis.go            # PatchGenesisMultiValidator
│   └── peer.go               # Peer config helpers
├── agents/                   # Agent SDK wrappers
├── claw/                     # OpenClaw gateway client
├── helpers/                  # Test utilities
│   ├── assert.go, wait.go    # Assertions and polling
│   ├── log.go                # Structured logging
│   └── report.go             # JSON reporter
├── catalog/                  # Test catalog
│   └── catalog.go            # 37 entries with env requirements
├── phase_a_lifecycle_test.go
├── phase_b_adversarial_test.go
├── phase_c_coordination_test.go
└── phase_d_agent_test.go
```
