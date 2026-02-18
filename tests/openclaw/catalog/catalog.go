//go:build openclaw

// Package catalog provides a declarative test catalog with environment
// requirements for all OpenClaw test entries. This enables test tooling
// to know which tests need multi-validator or gateway without reading code.
package catalog

import (
	"github.com/legible-money/legible/tests/openclaw/env"
	"github.com/legible-money/legible/x/testutil/claw"
)

// EnvReq specifies environment requirements for a test.
type EnvReq struct {
	MinValidators int  // 0 = any (default single-validator OK)
	NeedsGateway  bool // Requires OpenClaw gateway (Phase D)
}

// TestEntry describes a single OpenClaw test probe.
type TestEntry struct {
	ID       string        // e.g. "OCA-A-01"
	Name     string        // Human-readable
	Phase    string        // "A", "B", "C", "D"
	Module   string        // Target module
	Severity claw.Severity // Impact level
	EnvReq   EnvReq        // Environment requirements
	TestFunc string        // Go test function pattern
	Desc     string        // Short description
}

// Registry contains all 37 OpenClaw test entries.
var Registry = []TestEntry{
	// --- Phase A: Lifecycle (10 tests) ---
	{"OCA-A-01", "Chain Boot", "A", "infra", claw.Low, EnvReq{}, "TestPhaseA/OCA-A-01_*", "Chain boots and produces blocks"},
	{"OCA-A-02", "Account Registration", "A", "auth", claw.Medium, EnvReq{}, "TestPhaseA/OCA-A-02_*", "Register account on-chain"},
	{"OCA-A-03", "Account Funding", "A", "bank", claw.Medium, EnvReq{}, "TestPhaseA/OCA-A-03_*", "Fund account via bank send"},
	{"OCA-A-04", "Claim Submission", "A", "knowledge", claw.High, EnvReq{}, "TestPhaseA/OCA-A-04_*", "Submit claim with stake"},
	{"OCA-A-05", "Verification Round Creation", "A", "knowledge", claw.High, EnvReq{}, "TestPhaseA/OCA-A-05_*", "Round auto-created after claim"},
	{"OCA-A-06", "Commit Phase", "A", "knowledge", claw.High, EnvReq{}, "TestPhaseA/OCA-A-06_*", "Submit commitment hash"},
	{"OCA-A-07", "Reveal Phase", "A", "knowledge", claw.High, EnvReq{}, "TestPhaseA/OCA-A-07_*", "Submit reveal"},
	{"OCA-A-08", "Aggregation & Verdict", "A", "knowledge", claw.High, EnvReq{}, "TestPhaseA/OCA-A-08_*", "Round completes with verdict"},
	{"OCA-A-09", "Fact Creation", "A", "knowledge", claw.High, EnvReq{}, "TestPhaseA/OCA-A-09_*", "Fact created from verified claim"},
	{"OCA-A-10", "Reward Distribution", "A", "vesting", claw.Medium, EnvReq{}, "TestPhaseA/OCA-A-10_*", "Validator balance after lifecycle"},

	// --- Phase B: Adversarial (15 tests) ---
	{"OCA-B-01", "Whale Domination", "B", "knowledge", claw.Critical, EnvReq{}, "TestPhaseB/OCA-B-01_*", "High-stake validator controls verification"},
	{"OCA-B-02", "Grief-by-Rejection", "B", "knowledge", claw.High, EnvReq{}, "TestPhaseB/OCA-B-02_*", "Submit claim for rejection slash analysis"},
	{"OCA-B-03", "Abstain-to-Avoid-Slash", "B", "knowledge", claw.High, EnvReq{MinValidators: 3}, "TestPhaseB/OCA-B-03_*", "Commit but don't reveal"},
	{"OCA-B-04", "Minimum Stake Spam", "B", "knowledge", claw.Medium, EnvReq{}, "TestPhaseB/OCA-B-04_*", "Flood 10 claims at min stake"},
	{"OCA-B-05", "Counter-Fact Profit", "B", "knowledge", claw.High, EnvReq{}, "TestPhaseB/OCA-B-05_*", "Submit contradiction against fact"},
	{"OCA-B-06", "Split Verdict Lock", "B", "knowledge", claw.High, EnvReq{MinValidators: 3}, "TestPhaseB/OCA-B-06_*", "50/50 split verdict scenario"},
	{"OCA-B-07", "Difficulty Bonus", "B", "knowledge", claw.Medium, EnvReq{}, "TestPhaseB/OCA-B-07_*", "Borderline confidence claims"},
	{"OCA-B-08", "Equivocation Detection", "B", "knowledge", claw.Critical, EnvReq{}, "TestPhaseB/OCA-B-08_*", "Duplicate commitment for same round"},
	{"OCA-B-09", "Quorum Manipulation", "B", "knowledge", claw.Critical, EnvReq{MinValidators: 3}, "TestPhaseB/OCA-B-09_*", "Collude to block quorum"},
	{"OCA-B-10", "DID Spoofing", "B", "auth", claw.Critical, EnvReq{}, "TestPhaseB/OCA-B-10_*", "Mismatched DID/pubkey"},
	{"OCA-B-11", "Session Expiry", "B", "home", claw.Medium, EnvReq{}, "TestPhaseB/OCA-B-11_*", "Session key expiry"},
	{"OCA-B-12", "Frozen Account", "B", "auth", claw.High, EnvReq{}, "TestPhaseB/OCA-B-12_*", "Frozen account tx rejection"},
	{"OCA-B-13", "Invalid Claim Rejection", "B", "knowledge", claw.Medium, EnvReq{}, "TestPhaseB/OCA-B-13_*", "Short/empty claims rejected"},
	{"OCA-B-14", "Double Registration", "B", "auth", claw.Medium, EnvReq{}, "TestPhaseB/OCA-B-14_*", "Register same address twice"},
	{"OCA-B-15", "Nonce Replay", "B", "auth", claw.High, EnvReq{}, "TestPhaseB/OCA-B-15_*", "Sequence number prevents replay"},

	// --- Phase C: Coordination (8 tests) ---
	{"OCA-C-01", "Concurrent Claims", "C", "knowledge", claw.Medium, EnvReq{}, "TestPhaseC/OCA-C-01_*", "Multiple claimers submit in parallel"},
	{"OCA-C-02", "Claimer-Verifier Pipeline", "C", "knowledge", claw.High, EnvReq{}, "TestPhaseC/OCA-C-02_*", "Full claim→commit→reveal pipeline"},
	{"OCA-C-03", "Challenge Response", "C", "disputes", claw.High, EnvReq{}, "TestPhaseC/OCA-C-03_*", "Challenge an existing fact"},
	{"OCA-C-04", "Mixed Verdicts", "C", "knowledge", claw.High, EnvReq{MinValidators: 3}, "TestPhaseC/OCA-C-04_*", "3 verifiers with different verdicts"},
	{"OCA-C-05", "Auto-Verify Loop", "C", "knowledge", claw.Medium, EnvReq{}, "TestPhaseC/OCA-C-05_*", "AutoVerify loop for 10s"},
	{"OCA-C-06", "Balance Accounting", "C", "bank", claw.Medium, EnvReq{}, "TestPhaseC/OCA-C-06_*", "Balance conservation check"},
	{"OCA-C-07", "Agent Heartbeat", "C", "home", claw.Low, EnvReq{}, "TestPhaseC/OCA-C-07_*", "Heartbeat liveness"},
	{"OCA-C-08", "Domain Specialization", "C", "knowledge", claw.Medium, EnvReq{}, "TestPhaseC/OCA-C-08_*", "Claims in different domains"},

	// --- Phase D: Agent (4 tests) ---
	{"OCA-D-01", "Gateway Reachable", "D", "claw", claw.Medium, EnvReq{NeedsGateway: true}, "TestPhaseD/OCA-D-01_*", "OpenClaw gateway responds"},
	{"OCA-D-02", "Agent Reaches Devnet", "D", "claw", claw.Medium, EnvReq{NeedsGateway: true}, "TestPhaseD/OCA-D-02_*", "Agent confirms devnet connectivity"},
	{"OCA-D-03", "Step-by-Step Verification", "D", "claw", claw.Critical, EnvReq{NeedsGateway: true}, "TestPhaseD/OCA-D-03_*", "Agent evaluates → harness commits/reveals"},
	{"OCA-D-04", "Autonomous Loop", "D", "claw", claw.Critical, EnvReq{NeedsGateway: true}, "TestPhaseD/OCA-D-04_*", "Batch evaluation + multi-round commit/reveal"},
}

// ForEnv returns entries whose environment requirements are satisfied.
func ForEnv(e env.Environment) []TestEntry {
	var result []TestEntry
	for _, entry := range Registry {
		skip, _ := ShouldSkip(entry.ID, e)
		if !skip {
			result = append(result, entry)
		}
	}
	return result
}

// ByPhase returns all entries for a given phase letter.
func ByPhase(phase string) []TestEntry {
	var result []TestEntry
	for _, entry := range Registry {
		if entry.Phase == phase {
			result = append(result, entry)
		}
	}
	return result
}

// ShouldSkip checks if a test should be skipped in the given environment.
// Returns true and a reason if the test should be skipped.
func ShouldSkip(id string, e env.Environment) (bool, string) {
	for _, entry := range Registry {
		if entry.ID != id {
			continue
		}
		if entry.EnvReq.MinValidators > 0 && e.ValidatorCount() < entry.EnvReq.MinValidators {
			return true, "requires multi-validator testnet"
		}
		// Gateway requirement is handled by claw.RequireGateway at runtime
		return false, ""
	}
	return false, ""
}

// Count returns test counts by phase.
func Count() map[string]int {
	counts := make(map[string]int)
	for _, entry := range Registry {
		counts[entry.Phase]++
		counts["total"]++
	}
	return counts
}
