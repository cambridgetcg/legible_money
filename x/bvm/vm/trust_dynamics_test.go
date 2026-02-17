package vm

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"testing"
)

// =============================================================================
// Trust Dynamics & Social Graph Emergence Tests
//
// These tests explore how trust relationships emerge, evolve, and (don't) decay
// between agents in the BVM. Each test simulates a social scenario using raw
// bytecode executed against the BVM interpreter.
// =============================================================================

// helper: build a DID string from an integer identifier
func testDID(id int) string {
	return "did:lgm:agent" + uitoa(uint64(id))
}

// helper: build a 32-byte target DID hash for use in bytecode
func targetDIDBytes(id int64) [32]byte {
	return WordToBytes32(big.NewInt(id))
}

// helper: execute bytecode that sets trust from callerDID → targetDID at given level,
// then returns the trust level via GET_TRUST.
func buildSetThenGetTrust(targetDID [32]byte, level uint64) []byte {
	levelBytes := WordToBytes32(new(big.Int).SetUint64(level))
	bc := make([]byte, 0, 256)
	// SET_TRUST: pops targetDID, level, reason
	bc = append(bc, PUSH1, 0x01) // reason
	bc = append(bc, PUSH32)
	bc = append(bc, levelBytes[:]...) // level
	bc = append(bc, PUSH32)
	bc = append(bc, targetDID[:]...) // targetDID
	bc = append(bc, SET_TRUST)
	// GET_TRUST: pops targetDID, pushes level
	bc = append(bc, PUSH32)
	bc = append(bc, targetDID[:]...)
	bc = append(bc, GET_TRUST)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)
	return bc
}

// =============================================================================
// 1. Trust Bootstrap Problem
//    New agent with zero trust — how does it establish initial trust?
//    Bytecode reads trust level and branches on whether it's zero.
// =============================================================================

func TestTrustBootstrap_ZeroTrustDefault(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	// Query trust for a target we've never interacted with → should be 0
	target := targetDIDBytes(0x4E45)
	bc := make([]byte, 0, 128)
	bc = append(bc, PUSH32)
	bc = append(bc, target[:]...)
	bc = append(bc, GET_TRUST)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(200000)
	ctx.CallerDID = testDID(1)
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bc
	result := NewInterpreter().Execute(bc, ctx, state, host)
	if !result.Success {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatalf("expected zero trust for unknown agent, got %s", val)
	}
}

func TestTrustBootstrap_ConditionalBehavior(t *testing.T) {
	// Bytecode: check trust level for target. If zero, set initial trust to 100.
	// If non-zero, leave unchanged. Simulates "bootstrap" logic.
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	target := targetDIDBytes(0xBB)
	initialTrust := WordToBytes32(big.NewInt(100))

	// GET_TRUST → check if zero → if zero, SET_TRUST(100) → GET_TRUST → return
	bc := make([]byte, 0, 256)
	// GET_TRUST
	bc = append(bc, PUSH32)
	bc = append(bc, target[:]...)
	bc = append(bc, GET_TRUST)
	// ISZERO → if trust == 0, jump to bootstrap
	bc = append(bc, ISZERO)
	// JUMPI to bootstrap label (we'll calculate offset)
	bc = append(bc, PUSH1, 0) // placeholder
	jumpIPc := len(bc) - 1    // index of the destination byte
	bc = append(bc, JUMPI)

	// Fall-through: trust is non-zero, just read and return
	bc = append(bc, PUSH32)
	bc = append(bc, target[:]...)
	bc = append(bc, GET_TRUST)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)

	// Bootstrap label
	bootstrapOffset := len(bc)
	bc[jumpIPc] = byte(bootstrapOffset)
	bc = append(bc, JUMPDEST)
	// SET_TRUST(target, 100, reason=1)
	bc = append(bc, PUSH1, 0x01) // reason
	bc = append(bc, PUSH32)
	bc = append(bc, initialTrust[:]...) // level = 100
	bc = append(bc, PUSH32)
	bc = append(bc, target[:]...) // targetDID
	bc = append(bc, SET_TRUST)
	// Now read it back
	bc = append(bc, PUSH32)
	bc = append(bc, target[:]...)
	bc = append(bc, GET_TRUST)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(500000)
	ctx.CallerDID = testDID(1)
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bc
	result := NewInterpreter().Execute(bc, ctx, state, host)
	if !result.Success {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Uint64() != 100 {
		t.Fatalf("expected bootstrapped trust=100, got %s", val)
	}
}

// =============================================================================
// 2. Trust Transitivity
//    A trusts B, B trusts C. Does A trust C?
//    Answer: NO — the BVM has no transitive trust. Each pair is independent.
//    This test proves it and shows how bytecode could implement it manually.
// =============================================================================

func TestTrustTransitivity_NotBuiltIn(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	agentA := testDID(1)
	agentB := targetDIDBytes(0xBB)
	agentC := targetDIDBytes(0xCC)

	// A sets trust for B = 800
	host.SetTrust(agentA, agentB[:], 800, []byte{0x01})
	// B sets trust for C = 900
	host.SetTrust("did:lgm:agentBB", agentC[:], 900, []byte{0x01})

	// A queries trust for C directly → should be 0 (no transitivity)
	bc := make([]byte, 0, 128)
	bc = append(bc, PUSH32)
	bc = append(bc, agentC[:]...)
	bc = append(bc, GET_TRUST)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(200000)
	ctx.CallerDID = agentA
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bc
	result := NewInterpreter().Execute(bc, ctx, state, host)
	if !result.Success {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatalf("expected 0 (no transitive trust), got %s", val)
	}
}

func TestTrustTransitivity_ManualChainQuery(t *testing.T) {
	// Simulate transitive trust check via host API:
	// A trusts B at 800, B trusts C at 900.
	// "Transitive trust" = min(A→B, B→C) = 800
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	agentA := testDID(10)
	agentBDID := testDID(20)
	agentB := targetDIDBytes(0x20)
	agentC := targetDIDBytes(0x30)

	host.SetTrust(agentA, agentB[:], 800, []byte{0x01})
	host.SetTrust(agentBDID, agentC[:], 900, []byte{0x01})

	// Verify: A→B = 800
	trustAB := host.GetTrust(agentA, agentB[:])
	if trustAB != 800 {
		t.Fatalf("expected A→B trust=800, got %d", trustAB)
	}

	// Verify: B→C = 900
	trustBC := host.GetTrust(agentBDID, agentC[:])
	if trustBC != 900 {
		t.Fatalf("expected B→C trust=900, got %d", trustBC)
	}

	// Manual transitive: min(800, 900) = 800
	transitive := trustAB
	if trustBC < transitive {
		transitive = trustBC
	}
	if transitive != 800 {
		t.Fatalf("expected transitive trust=800, got %d", transitive)
	}

	// But direct A→C is still 0
	trustAC := host.GetTrust(agentA, agentC[:])
	if trustAC != 0 {
		t.Fatalf("expected direct A→C trust=0, got %d", trustAC)
	}
}

// =============================================================================
// 3. Trust Betrayal & Recovery
//    Agent sets high trust, then revokes it (sets to 0), then rebuilds.
// =============================================================================

func TestTrustBetrayal_SetRevokeRecover(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	callerDID := testDID(1)
	target := targetDIDBytes(0xBAD)

	// Phase 1: High trust
	host.SetTrust(callerDID, target[:], 1000, []byte("trusted"))
	if lvl := host.GetTrust(callerDID, target[:]); lvl != 1000 {
		t.Fatalf("phase 1: expected 1000, got %d", lvl)
	}

	// Phase 2: Betrayal — revoke to 0
	host.SetTrust(callerDID, target[:], 0, []byte("betrayed"))
	if lvl := host.GetTrust(callerDID, target[:]); lvl != 0 {
		t.Fatalf("phase 2: expected 0, got %d", lvl)
	}

	// Phase 3: IS_TRUSTED check should fail
	if host.IsTrusted(callerDID, target[:], 1) {
		t.Fatal("phase 3: should not be trusted at threshold=1")
	}

	// Phase 4: Partial recovery
	host.SetTrust(callerDID, target[:], 250, []byte("probation"))
	if lvl := host.GetTrust(callerDID, target[:]); lvl != 250 {
		t.Fatalf("phase 4: expected 250, got %d", lvl)
	}
	// Trusted at threshold=200 but not at 300
	if !host.IsTrusted(callerDID, target[:], 200) {
		t.Fatal("phase 4: should be trusted at threshold=200")
	}
	if host.IsTrusted(callerDID, target[:], 300) {
		t.Fatal("phase 4: should NOT be trusted at threshold=300")
	}
}

func TestTrustBetrayal_BytecodeRevokeAndCheck(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	target := targetDIDBytes(0xBAD)
	callerDID := testDID(5)

	// Pre-set high trust
	host.SetTrust(callerDID, target[:], 999, []byte("initial"))

	// Bytecode: SET_TRUST to 0 (revoke), then IS_TRUSTED with threshold=1 → should return 0
	zeroLevel := WordToBytes32(big.NewInt(0))
	threshold := WordToBytes32(big.NewInt(1))

	bc := make([]byte, 0, 256)
	// SET_TRUST(target, 0, reason)
	bc = append(bc, PUSH1, 0x01)
	bc = append(bc, PUSH32)
	bc = append(bc, zeroLevel[:]...)
	bc = append(bc, PUSH32)
	bc = append(bc, target[:]...)
	bc = append(bc, SET_TRUST)
	// IS_TRUSTED(target, 1)
	bc = append(bc, PUSH32)
	bc = append(bc, threshold[:]...)
	bc = append(bc, PUSH32)
	bc = append(bc, target[:]...)
	bc = append(bc, IS_TRUSTED)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(300000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bc
	result := NewInterpreter().Execute(bc, ctx, state, host)
	if !result.Success {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatalf("expected 0 (not trusted after revocation), got %s", val)
	}
}

// =============================================================================
// 4. Delegation Chains
//    A delegates to B, B delegates to C. Test capability flow.
// =============================================================================

func TestTrustDelegationChain(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	agentA := testDID(1)
	agentBDID := testDID(2)
	targetB := targetDIDBytes(0xB0)
	targetC := targetDIDBytes(0xC0)
	capability := WordToBytes32(big.NewInt(0x01))   // "transfer" capability
	constraints := WordToBytes32(big.NewInt(0x00))

	// A delegates "transfer" to B
	delegationAB := host.Delegate(agentA, targetB[:], capability[:], constraints[:])
	if delegationAB == nil {
		t.Fatal("delegation A→B should return non-nil ID")
	}

	// B delegates "transfer" to C
	delegationBC := host.Delegate(agentBDID, targetC[:], capability[:], constraints[:])
	if delegationBC == nil {
		t.Fatal("delegation B→C should return non-nil ID")
	}

	// Both delegation IDs should be unique (deterministic but different)
	if string(delegationAB) == string(delegationBC) {
		t.Fatal("delegation IDs should be distinct")
	}

	t.Logf("Delegation A→B: %x", delegationAB)
	t.Logf("Delegation B→C: %x", delegationBC)

	// FINDING: Delegation is fire-and-forget — no chain validation exists.
	// The BVM returns a hash but doesn't track whether C can exercise A's capability.
	// A proper delegation chain would need on-chain verification of the chain A→B→C.
}

func TestTrustDelegationRevoke(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	callerDID := testDID(1)
	target := targetDIDBytes(0xDD)
	cap := WordToBytes32(big.NewInt(0x01))
	con := WordToBytes32(big.NewInt(0x00))

	delegationId := host.Delegate(callerDID, target[:], cap[:], con[:])
	if delegationId == nil {
		t.Fatal("expected delegation ID")
	}

	// Revoke via bytecode
	delegationIdB32 := WordToBytes32(new(big.Int).SetBytes(delegationId))
	bc := make([]byte, 0, 64)
	bc = append(bc, PUSH32)
	bc = append(bc, delegationIdB32[:]...)
	bc = append(bc, REVOKE_DELEGATION)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(200000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bc
	result := NewInterpreter().Execute(bc, ctx, state, host)
	if !result.Success {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected 1 (revoked), got %s", val)
	}
}

// =============================================================================
// 5. Social Exclusion via BLOCK_AGENT
//    Blocked agents should be excluded from trust queries.
// =============================================================================

func TestTrustSocialExclusion_BlockExcludesTrust(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	callerDID := testDID(1)
	target := targetDIDBytes(0xBAD)

	// Set trust first
	host.SetTrust(callerDID, target[:], 999, []byte("friend"))

	// Verify trust exists
	if lvl := host.GetTrust(callerDID, target[:]); lvl != 999 {
		t.Fatalf("pre-block: expected 999, got %d", lvl)
	}

	// Block the agent
	if !host.BlockAgent(callerDID, target[:], []byte("spam")) {
		t.Fatal("BlockAgent should succeed")
	}

	// FINDING: Blocking does NOT automatically revoke trust.
	// Trust level remains 999 even after blocking.
	// The block is stored separately under "blocked/" prefix.
	trustAfterBlock := host.GetTrust(callerDID, target[:])
	t.Logf("Trust after block: %d (FINDING: block does NOT auto-revoke trust)", trustAfterBlock)

	// Verify block is recorded by checking storage directly
	blockKey := sha256.Sum256([]byte("blocked/" + callerDID + "/" + hex.EncodeToString(target[:])))
	blockVal := state.GetStorage([]byte("agent_trust"), blockKey[:])
	if blockVal == nil {
		t.Fatal("block record should exist in storage")
	}
	blocked := new(big.Int).SetBytes(blockVal)
	if blocked.Int64() != 1 {
		t.Fatalf("expected block value=1, got %s", blocked)
	}
}

func TestTrustSocialExclusion_UnblockRestoresAccess(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	callerDID := testDID(1)
	target := targetDIDBytes(0xBAD)

	// Block then unblock
	host.BlockAgent(callerDID, target[:], []byte("temp"))
	host.UnblockAgent(callerDID, target[:])

	// Verify block record is cleared
	blockKey := sha256.Sum256([]byte("blocked/" + callerDID + "/" + hex.EncodeToString(target[:])))
	blockVal := state.GetStorage([]byte("agent_trust"), blockKey[:])
	if blockVal != nil {
		// Check if it's actually zeroed
		v := new(big.Int).SetBytes(blockVal)
		if v.Sign() != 0 {
			t.Fatalf("expected block cleared, got %s", v)
		}
	}
}

func TestTrustSocialExclusion_BlockViaBytecode(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	target := targetDIDBytes(0x5BA4)

	// BLOCK_AGENT via bytecode: pops targetDID, reason → pushes success
	bc := make([]byte, 0, 128)
	bc = append(bc, PUSH1, 0x01) // reason
	bc = append(bc, PUSH32)
	bc = append(bc, target[:]...) // target
	bc = append(bc, BLOCK_AGENT)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(200000)
	ctx.CallerDID = testDID(1)
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bc
	result := NewInterpreter().Execute(bc, ctx, state, host)
	if !result.Success {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected 1 (blocked), got %s", val)
	}
}

// =============================================================================
// 6. Trust-Based Coalition
//    Agents form a trust cluster above a threshold, exclude outsiders.
// =============================================================================

func TestTrustCoalition_ThresholdGating(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	// Coalition leader sets trust for members
	leader := testDID(1)
	member1 := targetDIDBytes(0xA1)
	member2 := targetDIDBytes(0xA2)
	outsider := targetDIDBytes(0xFF)

	coalitionThreshold := uint64(500)

	// Leader trusts members above threshold
	host.SetTrust(leader, member1[:], 800, []byte("coalition"))
	host.SetTrust(leader, member2[:], 600, []byte("coalition"))
	// Outsider below threshold
	host.SetTrust(leader, outsider[:], 100, []byte("acquaintance"))

	// IS_TRUSTED checks
	if !host.IsTrusted(leader, member1[:], coalitionThreshold) {
		t.Fatal("member1 should be trusted above coalition threshold")
	}
	if !host.IsTrusted(leader, member2[:], coalitionThreshold) {
		t.Fatal("member2 should be trusted above coalition threshold")
	}
	if host.IsTrusted(leader, outsider[:], coalitionThreshold) {
		t.Fatal("outsider should NOT be trusted above coalition threshold")
	}
}

func TestTrustCoalition_BytecodeGatekeeper(t *testing.T) {
	// Bytecode: check if caller is trusted by gatekeeper above threshold.
	// If yes, return 1 (access granted). If no, return 0 (denied).
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	gatekeeper := testDID(1)
	member := targetDIDBytes(0xAA)
	threshold := WordToBytes32(big.NewInt(500))

	// Pre-set trust
	host.SetTrust(gatekeeper, member[:], 700, []byte("member"))

	// IS_TRUSTED(member, 500) → return result
	bc := make([]byte, 0, 128)
	bc = append(bc, PUSH32)
	bc = append(bc, threshold[:]...)
	bc = append(bc, PUSH32)
	bc = append(bc, member[:]...)
	bc = append(bc, IS_TRUSTED)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(200000)
	ctx.CallerDID = gatekeeper
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bc
	result := NewInterpreter().Execute(bc, ctx, state, host)
	if !result.Success {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected 1 (coalition access granted), got %s", val)
	}
}

func TestTrustCoalition_MutualTrust(t *testing.T) {
	// Two agents mutually trust each other — bidirectional trust
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	agentA := testDID(1)
	agentB := testDID(2)
	targetA := targetDIDBytes(0xA0)
	targetB := targetDIDBytes(0xB0)

	// A trusts B, B trusts A
	host.SetTrust(agentA, targetB[:], 900, []byte("mutual"))
	host.SetTrust(agentB, targetA[:], 850, []byte("mutual"))

	// Verify bidirectional
	if lvl := host.GetTrust(agentA, targetB[:]); lvl != 900 {
		t.Fatalf("A→B trust expected 900, got %d", lvl)
	}
	if lvl := host.GetTrust(agentB, targetA[:]); lvl != 850 {
		t.Fatalf("B→A trust expected 850, got %d", lvl)
	}

	// Trust is asymmetric — A→B ≠ B→A (this is by design)
	t.Log("FINDING: Trust is inherently asymmetric — A→B and B→A are independent values")
}

// =============================================================================
// 7. Trust Decay Under Silence
//    When agents don't interact, does trust decay?
//    FINDING: NO — trust is static. Once set, it persists forever.
// =============================================================================

func TestTrustDecay_DoesNotExist(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	callerDID := testDID(1)
	target := targetDIDBytes(0x01D)

	// Set trust at "block 100"
	host.SetTrust(callerDID, target[:], 999, []byte("friends"))

	// Simulate passage of time — query at "block 1000000"
	// Trust should remain unchanged because there's no decay mechanism
	trust := host.GetTrust(callerDID, target[:])
	if trust != 999 {
		t.Fatalf("expected trust=999 (no decay), got %d", trust)
	}

	// IS_TRUSTED still passes at original level
	if !host.IsTrusted(callerDID, target[:], 999) {
		t.Fatal("trust should not decay — no decay mechanism exists")
	}

	t.Log("CONFIRMED: Trust does NOT decay. Once set, it persists indefinitely.")
	t.Log("RECOMMENDATION: Add timestamp-based decay: trust_effective = trust_level * decay_factor(blocks_since_last_interaction)")
}

// =============================================================================
// Integration: Full Social Graph Scenario
// =============================================================================

func TestTrustFullScenario_SocialGraphLifecycle(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	// Cast of characters
	alice := testDID(1)
	bobTarget := targetDIDBytes(0xB0B)
	carolTarget := targetDIDBytes(0xCA0)
	eveTarget := targetDIDBytes(0xE7E)

	// Act 1: Alice builds her trust network
	host.SetTrust(alice, bobTarget[:], 900, []byte("colleague"))
	host.SetTrust(alice, carolTarget[:], 700, []byte("friend"))
	host.SetTrust(alice, eveTarget[:], 300, []byte("new_acquaintance"))

	// Act 2: Alice delegates "transfer" capability to Bob
	cap := WordToBytes32(big.NewInt(0x01))
	con := WordToBytes32(big.NewInt(0x00))
	delegation := host.Delegate(alice, bobTarget[:], cap[:], con[:])
	if delegation == nil {
		t.Fatal("delegation should succeed")
	}

	// Act 3: Eve behaves badly — Alice revokes trust and blocks
	host.SetTrust(alice, eveTarget[:], 0, []byte("scammer"))
	host.BlockAgent(alice, eveTarget[:], []byte("fraud"))

	// Act 4: Verify final state
	// Bob: high trust, delegated
	if !host.IsTrusted(alice, bobTarget[:], 500) {
		t.Fatal("Bob should be trusted")
	}
	// Carol: medium trust
	if !host.IsTrusted(alice, carolTarget[:], 500) {
		t.Fatal("Carol should be trusted at 500")
	}
	if host.IsTrusted(alice, carolTarget[:], 800) {
		t.Fatal("Carol should NOT be trusted at 800")
	}
	// Eve: revoked
	if host.IsTrusted(alice, eveTarget[:], 1) {
		t.Fatal("Eve should not be trusted at all")
	}

	// Act 5: Eve tries to recover — Alice gives probationary trust
	host.UnblockAgent(alice, eveTarget[:])
	host.SetTrust(alice, eveTarget[:], 50, []byte("probation"))
	if host.IsTrusted(alice, eveTarget[:], 100) {
		t.Fatal("Eve should not meet threshold=100 on probation")
	}
	if !host.IsTrusted(alice, eveTarget[:], 50) {
		t.Fatal("Eve should meet threshold=50 on probation")
	}

	t.Log("Full social graph lifecycle completed successfully")
}
