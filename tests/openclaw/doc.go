//go:build openclaw

// Package openclaw provides live-chain agent testing for Legible Money.
//
// These tests boot an ephemeral chain and exercise the full PoT verification
// lifecycle using real AI agents (via the Agent SDK). Build tag "openclaw"
// prevents these slow tests from running in regular go test ./... passes.
//
// # Environment Modes
//
// Tests auto-detect the environment via env.Detect():
//
//   - devnet (default): Single-validator ephemeral chain
//   - testnet (LGM_ENV=testnet): 3-validator ephemeral chain
//   - external (LGM_TESTNET_RPC set): Pre-existing chain
//
// # Running Tests
//
//	make test-openclaw-devnet    # single-validator (default)
//	make test-openclaw-testnet   # 3-validator multi-process
//	make test-openclaw-report    # with JSON report output
//
// # Test Phases
//
//	Phase A: Lifecycle (10 tests) — chain boot, claims, verification rounds
//	Phase B: Adversarial (15 tests) — attacks, exploits, edge cases
//	Phase C: Coordination (8 tests) — multi-agent workflows
//	Phase D: Agent (4 tests) — AI agent gateway integration
package openclaw
