package vm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

// AgentKeeper provides agent identity/relationship/autonomy data.
type AgentKeeper interface {
	GetAccountByDID(ctx context.Context, did string) (interface{}, bool)
}

// AgentHost bridges BVM agent opcodes to Cosmos keeper state.
// Agent beliefs/relationships are stored in contract state under reserved key prefixes.
type AgentHost struct {
	ctx         context.Context
	stateDB     StateDB
	schedKeeper ScheduleKeeper // nil-safe: methods degrade gracefully
}

func NewAgentHost(ctx context.Context, stateDB StateDB) *AgentHost {
	return &AgentHost{ctx: ctx, stateDB: stateDB}
}

// NewAgentHostWithSchedules creates an AgentHost wired to persist schedules.
func NewAgentHostWithSchedules(ctx context.Context, stateDB StateDB, sk ScheduleKeeper) *AgentHost {
	return &AgentHost{ctx: ctx, stateDB: stateDB, schedKeeper: sk}
}

// --- Agent Identity (0xE0-0xE3) ---

func (h *AgentHost) AgentDID(caller []byte) []byte {
	// Return caller address as a DID hash
	if len(caller) == 0 {
		return nil
	}
	didStr := "did:lgm:" + hex.EncodeToString(caller)
	hash := sha256.Sum256([]byte(didStr))
	return hash[:]
}

func (h *AgentHost) AgentHome(callerDID string) []byte {
	if callerDID == "" {
		return nil
	}
	// Home address is stored in contract state under "home/" prefix
	key := sha256.Sum256([]byte("home/" + callerDID))
	val := h.stateDB.GetStorage([]byte("agent_registry"), key[:])
	if val == nil {
		return nil
	}
	return val
}

func (h *AgentHost) VerifyIdentity(targetDID []byte) bool {
	// Check if the DID is registered
	didStr := hex.EncodeToString(targetDID)
	key := sha256.Sum256([]byte("identity/" + didStr))
	val := h.stateDB.GetStorage([]byte("agent_registry"), key[:])
	return val != nil
}

func (h *AgentHost) AttestIdentity(callerDID string, targetDID, attestationType, data []byte) []byte {
	if callerDID == "" {
		return nil
	}
	input := fmt.Sprintf("attest/%s/%x/%x", callerDID, targetDID, attestationType)
	hash := sha256.Sum256([]byte(input))
	return hash[:]
}

// --- Agent Belief (0xE4-0xE7) ---

func (h *AgentHost) GetBelief(callerDID string, factId []byte) uint64 {
	if callerDID == "" {
		return 0
	}
	key := sha256.Sum256([]byte("belief/" + callerDID + "/" + hex.EncodeToString(factId)))
	val := h.stateDB.GetStorage([]byte("agent_beliefs"), key[:])
	if val == nil {
		return 0
	}
	return new(big.Int).SetBytes(val).Uint64()
}

func (h *AgentHost) SetBelief(callerDID string, factId []byte, confidence uint64, source []byte) {
	if callerDID == "" {
		return
	}
	key := sha256.Sum256([]byte("belief/" + callerDID + "/" + hex.EncodeToString(factId)))
	val := new(big.Int).SetUint64(confidence)
	b32 := WordToBytes32(val)
	h.stateDB.SetStorage([]byte("agent_beliefs"), key[:], b32[:])
}

func (h *AgentHost) CompareBelief(callerDID string, factId []byte) int64 {
	// Compare agent's belief with network confidence
	// Returns positive if agent > network, negative if agent < network
	agentConf := h.GetBelief(callerDID, factId)
	// Network confidence would come from knowledge keeper (simplified)
	networkConf := uint64(500000000000000000) // 0.5 * 1e18 default
	return int64(agentConf) - int64(networkConf)
}

func (h *AgentHost) BeliefCount(callerDID string, domain []byte) uint64 {
	// In a full implementation, iterate over beliefs in domain.
	// For now, return 0 (can be computed from contract state later).
	return 0
}

// --- Agent Relationship (0xE8-0xEB) ---

func (h *AgentHost) GetTrust(callerDID string, targetDID []byte) uint64 {
	if callerDID == "" {
		return 0
	}
	key := sha256.Sum256([]byte("trust/" + callerDID + "/" + hex.EncodeToString(targetDID)))
	val := h.stateDB.GetStorage([]byte("agent_trust"), key[:])
	if val == nil {
		return 0
	}
	return new(big.Int).SetBytes(val).Uint64()
}

func (h *AgentHost) SetTrust(callerDID string, targetDID []byte, level uint64, reason []byte) {
	if callerDID == "" {
		return
	}
	key := sha256.Sum256([]byte("trust/" + callerDID + "/" + hex.EncodeToString(targetDID)))
	val := new(big.Int).SetUint64(level)
	b32 := WordToBytes32(val)
	h.stateDB.SetStorage([]byte("agent_trust"), key[:], b32[:])
}

func (h *AgentHost) IsTrusted(callerDID string, targetDID []byte, threshold uint64) bool {
	level := h.GetTrust(callerDID, targetDID)
	return level >= threshold
}

func (h *AgentHost) Delegate(callerDID string, targetDID, capability, constraints []byte) []byte {
	if callerDID == "" {
		return nil
	}
	input := fmt.Sprintf("delegate/%s/%x/%x", callerDID, targetDID, capability)
	hash := sha256.Sum256([]byte(input))
	return hash[:]
}

// --- Agent Autonomy (0xEC-0xEF) ---

func (h *AgentHost) ScheduleSelf(contract []byte, callData []byte, executeBlock uint64, gasLimit uint64) []byte {
	if h.schedKeeper == nil {
		// Graceful degradation: return deterministic ID but don't persist
		input := fmt.Sprintf("schedule/%x/%x/%d", contract, callData, executeBlock)
		hash := sha256.Sum256([]byte(input))
		return hash[:]
	}

	// Enforce per-contract schedule limit
	contractAddr := string(contract)
	if h.schedKeeper.CountActiveSchedules(contractAddr) >= h.schedKeeper.MaxSchedulesAllowed() {
		return nil // Signal failure to the contract
	}

	scheduleId, err := h.schedKeeper.CreateSchedule(
		contractAddr,
		contractAddr,                 // self-invocation: caller is the contract itself
		hex.EncodeToString(callData), // method stored as hex of calldata
		"",                           // no separate payload
		executeBlock,
		gasLimit,
	)
	if err != nil {
		return nil
	}

	hash := sha256.Sum256([]byte(scheduleId))
	return hash[:]
}

func (h *AgentHost) CancelScheduleHost(scheduleId []byte) bool {
	if h.schedKeeper == nil {
		return false
	}
	id := hex.EncodeToString(scheduleId)
	err := h.schedKeeper.CancelScheduleByID(id, "")
	return err == nil
}

func (h *AgentHost) ModifySchedule(scheduleId []byte, newBlock uint64) bool {
	if h.schedKeeper == nil || newBlock == 0 {
		return false
	}
	id := hex.EncodeToString(scheduleId)
	err := h.schedKeeper.ModifyScheduleBlock(id, "", newBlock)
	return err == nil
}

func (h *AgentHost) GetScheduleCount(contract []byte) uint64 {
	if h.schedKeeper == nil {
		return 0
	}
	return h.schedKeeper.CountActiveSchedules(string(contract))
}

// --- Extended Relationships (0xC0-0xC3) ---

func (h *AgentHost) ListTrusted(callerDID string, offset, limit uint64) (uint64, [][]byte) {
	return 0, nil
}

func (h *AgentHost) BlockAgent(callerDID string, targetDID, reason []byte) bool {
	if callerDID == "" {
		return false
	}
	key := sha256.Sum256([]byte("blocked/" + callerDID + "/" + hex.EncodeToString(targetDID)))
	b32 := WordToBytes32(big.NewInt(1))
	h.stateDB.SetStorage([]byte("agent_trust"), key[:], b32[:])
	return true
}

func (h *AgentHost) UnblockAgent(callerDID string, targetDID []byte) bool {
	if callerDID == "" {
		return false
	}
	key := sha256.Sum256([]byte("blocked/" + callerDID + "/" + hex.EncodeToString(targetDID)))
	h.stateDB.SetStorage([]byte("agent_trust"), key[:], nil)
	return true
}

func (h *AgentHost) RevokeDelegation(callerDID string, delegationId []byte) bool {
	return callerDID != "" && len(delegationId) > 0
}
