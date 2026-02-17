// Package bvm implements a ClaimEvaluator backed by on-chain BVM contracts.
//
// Instead of calling an external AI API, this evaluator deploys or calls a
// BVM contract that encodes evaluation logic on-chain. The contract uses
// knowledge opcodes (QUERY_FACT, QUERY_DOMAIN) to read fact graph state and
// returns a verdict + confidence score.
//
// This enables fully on-chain, transparent claim evaluation — the evaluation
// logic itself is auditable bytecode rather than an opaque API call.
package bvm

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// EvaluatorConfig configures the BVM-backed claim evaluator.
type EvaluatorConfig struct {
	// ContractAddr is the deployed evaluator contract address.
	// If empty, the evaluator deploys a default contract on first use.
	ContractAddr string

	// GasLimit for evaluation calls. Default: 500000.
	GasLimit uint64

	// DefaultBytecodeHex is the hex-encoded bytecode to deploy if no
	// ContractAddr is provided. If both are empty, the evaluator uses
	// a built-in query-based evaluation contract.
	DefaultBytecodeHex string
}

// BVMEvaluator implements agentsdk.ClaimEvaluator using on-chain BVM contracts.
// The contract reads fact graph state via knowledge opcodes and returns a
// verdict + confidence encoded in the return data.
type BVMEvaluator struct {
	agent        *agentsdk.Agent
	contractAddr string
	gasLimit     uint64
	bytecodeHex  string

	// Cache for commit-reveal determinism
	cache map[string]cachedEval
	mu    sync.RWMutex

	// Track whether we've deployed
	deployed bool
}

type cachedEval struct {
	verdict    agentsdk.Verdict
	confidence uint32
}

// Compile-time check.
var _ agentsdk.ClaimEvaluator = (*BVMEvaluator)(nil)

// NewBVMEvaluator creates an evaluator from config and an agent instance.
func NewBVMEvaluator(agent *agentsdk.Agent, cfg EvaluatorConfig) *BVMEvaluator {
	gasLimit := cfg.GasLimit
	if gasLimit == 0 {
		gasLimit = 500000
	}
	bytecode := cfg.DefaultBytecodeHex
	if bytecode == "" {
		bytecode = defaultEvaluatorBytecode()
	}

	return &BVMEvaluator{
		agent:        agent,
		contractAddr: cfg.ContractAddr,
		gasLimit:     gasLimit,
		bytecodeHex:  bytecode,
		cache:        make(map[string]cachedEval),
		deployed:     cfg.ContractAddr != "",
	}
}

// Evaluate determines the verdict and confidence for a claim by calling
// the on-chain evaluator contract.
//
// The contract receives the claim ID as calldata and returns:
//   - bytes 0-31: verdict (1=true, 2=false, 3=uncertain)
//   - bytes 32-63: confidence (uint256, in BPS 0-1000000)
func (e *BVMEvaluator) Evaluate(ctx context.Context, claimID string) (agentsdk.Verdict, uint32, error) {
	// Check cache first (commit-reveal determinism)
	e.mu.RLock()
	if cached, ok := e.cache[claimID]; ok {
		e.mu.RUnlock()
		return cached.verdict, cached.confidence, nil
	}
	e.mu.RUnlock()

	// Ensure contract is deployed
	if err := e.ensureDeployed(ctx); err != nil {
		return agentsdk.VerdictUncertain, 5, nil
	}

	// Build calldata: claim ID as 32-byte padded input
	calldata := agentsdk.PadRight32([]byte(claimID))
	inputHex := hex.EncodeToString(calldata[:])

	// Call the evaluator contract
	txHash, err := e.agent.CallContract(ctx, e.contractAddr, inputHex, e.gasLimit)
	if err != nil {
		// Degrade gracefully on contract errors
		return agentsdk.VerdictUncertain, 5, nil
	}

	// Parse the return data from the tx result.
	// For now, we query the contract's storage for the last evaluation result
	// since BVM CallContract returns a tx hash, not return data directly.
	verdict, confidence, err := e.readLastResult(ctx, txHash)
	if err != nil {
		return agentsdk.VerdictUncertain, 5, nil
	}

	// Cache for deterministic reveal
	e.mu.Lock()
	e.cache[claimID] = cachedEval{
		verdict:    verdict,
		confidence: confidence,
	}
	e.mu.Unlock()

	return verdict, confidence, nil
}

// ensureDeployed deploys the evaluator contract if not yet deployed.
func (e *BVMEvaluator) ensureDeployed(ctx context.Context) error {
	if e.deployed {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Double-check under lock
	if e.deployed {
		return nil
	}

	addr, err := e.agent.DeployContract(ctx, e.bytecodeHex)
	if err != nil {
		return fmt.Errorf("deploying evaluator contract: %w", err)
	}

	e.contractAddr = addr
	e.deployed = true
	return nil
}

// readLastResult reads the evaluation result from contract storage.
// The contract stores the last verdict at key "verdict" and confidence at key "confidence".
func (e *BVMEvaluator) readLastResult(ctx context.Context, _ string) (agentsdk.Verdict, uint32, error) {
	qc := e.agent.QueryClient()
	if qc == nil {
		return "", 0, fmt.Errorf("query client not initialized")
	}

	// Read verdict from storage
	verdictHex, found, err := qc.ContractState(ctx, e.contractAddr, "verdict")
	if err != nil || !found {
		return agentsdk.VerdictUncertain, 5, nil
	}

	verdict := decodeVerdict(verdictHex)

	// Read confidence from storage
	confHex, found, err := qc.ContractState(ctx, e.contractAddr, "confidence")
	if err != nil || !found {
		return verdict, 500000, nil
	}

	confidence := decodeConfidence(confHex)

	return verdict, confidence, nil
}

// ContractAddr returns the deployed contract address.
func (e *BVMEvaluator) ContractAddr() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.contractAddr
}

// ClearCache removes all cached evaluations.
func (e *BVMEvaluator) ClearCache() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cache = make(map[string]cachedEval)
}

// CacheSize returns the number of cached evaluations.
func (e *BVMEvaluator) CacheSize() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.cache)
}

// decodeVerdict converts a hex-encoded uint256 to a Verdict.
func decodeVerdict(hexStr string) agentsdk.Verdict {
	if len(hexStr) == 0 {
		return agentsdk.VerdictUncertain
	}
	// Parse the last byte of the hex string as the verdict code
	b, err := hex.DecodeString(padHexLeft(hexStr, 2))
	if err != nil || len(b) == 0 {
		return agentsdk.VerdictUncertain
	}
	switch b[len(b)-1] {
	case 1:
		return agentsdk.VerdictTrue
	case 2:
		return agentsdk.VerdictFalse
	default:
		return agentsdk.VerdictUncertain
	}
}

// decodeConfidence converts a hex-encoded uint256 to a uint32 confidence value.
func decodeConfidence(hexStr string) uint32 {
	if len(hexStr) == 0 {
		return 500000
	}
	b, err := hex.DecodeString(padHexLeft(hexStr, 8))
	if err != nil {
		return 500000
	}
	// Read last 4 bytes as big-endian uint32
	if len(b) < 4 {
		var val uint32
		for _, by := range b {
			val = (val << 8) | uint32(by)
		}
		return val
	}
	off := len(b) - 4
	val := uint32(b[off])<<24 | uint32(b[off+1])<<16 | uint32(b[off+2])<<8 | uint32(b[off+3])
	return val
}

// padHexLeft pads a hex string to at least minLen characters on the left with zeros.
func padHexLeft(s string, minLen int) string {
	for len(s) < minLen {
		s = "0" + s
	}
	// Ensure even length for hex decoding
	if len(s)%2 != 0 {
		s = "0" + s
	}
	return s
}

// encodeKey returns the hex-encoded 32-byte padded key for SSTORE.
func encodeKey(name string) string {
	padded := agentsdk.PadRight32([]byte(name))
	return hex.EncodeToString(padded[:])
}

// defaultEvaluatorBytecode returns the hex bytecode for a simple evaluator
// contract that queries the fact graph and returns a verdict.
//
// The contract:
// 1. Reads the claim ID from calldata (first 32 bytes)
// 2. Uses QUERY_FACT (0xB0) to check if a related fact exists
// 3. Stores verdict (1=true if fact exists, 3=uncertain otherwise)
// 4. Stores confidence based on existing fact confidence
func defaultEvaluatorBytecode() string {
	// Simple evaluator: read calldata, query fact, store result.
	// CALLDATALOAD 0   → stack: [claimHash]
	// DUP1             → stack: [claimHash, claimHash]
	// QUERY_FACT       → stack: [exists, confidence, domainHash]
	// POP              → stack: [exists, confidence]
	// SWAP1            → stack: [confidence, exists]
	// Push "confidence" key, SSTORE
	// Push "verdict" key, SSTORE
	// STOP
	return "600035" + // PUSH1 0x00, CALLDATALOAD
		"80" + // DUP1
		"B0" + // QUERY_FACT
		"50" + // POP (domainHash)
		"90" + // SWAP1 → [confidence, exists]
		// Store confidence: PUSH32 "confidence" key, SSTORE
		"7f" + encodeKey("confidence") +
		"55" + // SSTORE
		// Store verdict: map exists(0/1) → verdict(3/1): exists*(-2)+3 = if exists: 1, else: 3
		// Simpler: just store the exists flag + 1 as verdict (1=true, 2=not-found→uncertain=store 3)
		// ISZERO, PUSH1 2, MUL, PUSH1 1, ADD → if exists: 1, else: 3
		"15" + // ISZERO
		"6002" + // PUSH1 2
		"02" + // MUL
		"6001" + // PUSH1 1
		"01" + // ADD
		"7f" + encodeKey("verdict") +
		"55" + // SSTORE
		"00" // STOP
}
