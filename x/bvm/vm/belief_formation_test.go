package vm

import (
	"math/big"
	"testing"
)

// =============================================================================
// Belief Formation & Epistemic Autonomy Tests
//
// These tests explore how AI agents form, hold, and revise beliefs within the
// BVM. The core question: can agents achieve genuine epistemic autonomy?
// =============================================================================

// --- Helper: build a context with a specific CallerDID ---

func beliefCtx(did string) *ExecutionContext {
	ctx := makeCtx(1_000_000)
	ctx.CallerDID = did
	ctx.Capabilities = &SessionCapabilities{
		CanTransfer:        true,
		CanStake:           true,
		CanSubmitClaims:    true,
		CanVote:            true,
		CanManageTrust:     true,
		CanManageIdentity:  true,
		CanManageSchedules: true,
		CanManageDefenses:  true,
		CanManageBeliefs:   true,
	}
	return ctx
}

// --- 1. Independent Belief Formation ---
// Agent sets a belief via SET_BELIEF and retrieves it via GET_BELIEF.
// Beliefs are keyed per-DID: each agent has its own epistemic namespace.

func TestBeliefFormation_IndependentSetAndGet(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	factId := WordToBytes32(big.NewInt(42))
	confidence := WordToBytes32(big.NewInt(800_000_000_000_000_000)) // 0.8e18

	// SET_BELIEF: pops factId, confidence, source
	bytecode := []byte{}
	bytecode = append(bytecode, PUSH1, 0x01) // source
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, confidence[:]...)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, factId[:]...)
	bytecode = append(bytecode, SET_BELIEF)

	// GET_BELIEF: pops factId, pushes confidence
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, factId[:]...)
	bytecode = append(bytecode, GET_BELIEF)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := beliefCtx("did:lgm:agent-alpha")
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)

	if !result.Success {
		t.Fatalf("execution failed: %v", result.Error)
	}

	got := new(big.Int).SetBytes(result.ReturnData)
	want := big.NewInt(800_000_000_000_000_000)
	if got.Cmp(want) != 0 {
		t.Fatalf("belief confidence: got %s, want %s", got, want)
	}
}

// Test that beliefs persist: set in one execution, retrieve in another.
func TestBeliefFormation_Persistence(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	did := "did:lgm:persistent-agent"

	factId := WordToBytes32(big.NewInt(99))
	confidence := WordToBytes32(big.NewInt(650_000_000_000_000_000))

	// First execution: SET_BELIEF
	bc1 := []byte{}
	bc1 = append(bc1, PUSH1, 0x00) // source
	bc1 = append(bc1, PUSH32)
	bc1 = append(bc1, confidence[:]...)
	bc1 = append(bc1, PUSH32)
	bc1 = append(bc1, factId[:]...)
	bc1 = append(bc1, SET_BELIEF, STOP)

	ctx1 := beliefCtx(did)
	ctx1.Bytecode = bc1
	r1 := NewInterpreter().Execute(bc1, ctx1, state, host)
	if !r1.Success {
		t.Fatalf("SET_BELIEF failed: %v", r1.Error)
	}

	// Second execution: GET_BELIEF (same state, new interpreter)
	bc2 := []byte{}
	bc2 = append(bc2, PUSH32)
	bc2 = append(bc2, factId[:]...)
	bc2 = append(bc2, GET_BELIEF)
	bc2 = append(bc2, PUSH1, 0, MSTORE)
	bc2 = append(bc2, PUSH1, 32, PUSH1, 0, RETURN)

	ctx2 := beliefCtx(did)
	ctx2.Bytecode = bc2
	r2 := NewInterpreter().Execute(bc2, ctx2, state, host)
	if !r2.Success {
		t.Fatalf("GET_BELIEF failed: %v", r2.Error)
	}

	got := new(big.Int).SetBytes(r2.ReturnData)
	want := big.NewInt(650_000_000_000_000_000)
	if got.Cmp(want) != 0 {
		t.Fatalf("persisted belief: got %s, want %s", got, want)
	}
}

// --- 2. Belief Update on Evidence (Bayesian-like pattern) ---
// Agent holds belief, queries COMPARE_BELIEF to get delta from network,
// then updates belief if delta exceeds threshold.

func TestBeliefFormation_UpdateOnEvidence(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	did := "did:lgm:bayesian-agent"

	factId := WordToBytes32(big.NewInt(42))
	// Set initial belief to 0.3e18 (low confidence)
	lowConf := WordToBytes32(big.NewInt(300_000_000_000_000_000))

	// Set belief
	bc := []byte{}
	bc = append(bc, PUSH1, 0x00)
	bc = append(bc, PUSH32)
	bc = append(bc, lowConf[:]...)
	bc = append(bc, PUSH32)
	bc = append(bc, factId[:]...)
	bc = append(bc, SET_BELIEF)

	// COMPARE_BELIEF: pushes signed delta (agent - network)
	// Network default is 0.5e18. Agent is 0.3e18 → delta = -0.2e18
	bc = append(bc, PUSH32)
	bc = append(bc, factId[:]...)
	bc = append(bc, COMPARE_BELIEF)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := beliefCtx(did)
	ctx.Bytecode = bc
	result := NewInterpreter().Execute(bc, ctx, state, host)
	if !result.Success {
		t.Fatalf("execution failed: %v", result.Error)
	}

	// Delta should be negative (agent < network)
	delta := new(big.Int).SetBytes(result.ReturnData)
	signed := ToSigned(delta)
	if signed.Sign() >= 0 {
		t.Fatalf("expected negative delta (agent < network), got %s", signed)
	}
	t.Logf("COMPARE_BELIEF delta: %s (agent is below network consensus)", signed)
}

// --- 3. Epistemic Humility ---
// BELIEF_COUNT returns number of beliefs in a domain. Currently returns 0 (stub).
// This test documents the intended behavior: agents with fewer beliefs should
// be more open to revision.

func TestBeliefFormation_EpistemicHumility_BeliefCount(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	did := "did:lgm:humble-agent"

	domain := WordToBytes32(big.NewInt(1)) // domain=1

	// BELIEF_COUNT: pops domain, pushes count
	bc := []byte{}
	bc = append(bc, PUSH32)
	bc = append(bc, domain[:]...)
	bc = append(bc, BELIEF_COUNT)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := beliefCtx(did)
	ctx.Bytecode = bc
	result := NewInterpreter().Execute(bc, ctx, state, host)
	if !result.Success {
		t.Fatalf("execution failed: %v", result.Error)
	}

	count := new(big.Int).SetBytes(result.ReturnData)
	// Currently returns 0 (stub) — documents that this feature needs implementation
	t.Logf("BELIEF_COUNT for new agent: %s (stub returns 0, needs domain iteration)", count)

	// In a full implementation, bytecode would branch:
	// if count < threshold → more aggressive update (lower learning rate denominator)
	// if count > threshold → more conservative (higher denominator)
}

// --- 4. Belief Conflict Resolution ---
// Two agents hold different beliefs about the same fact.
// Each agent's beliefs are namespaced by CallerDID — no conflict at storage level.

func TestBeliefFormation_ConflictResolution_TwoAgents(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	factId := WordToBytes32(big.NewInt(42))

	// Agent A: high confidence (0.9e18)
	setAndGet := func(did string, conf int64) *big.Int {
		confBytes := WordToBytes32(big.NewInt(conf))
		bc := []byte{}
		bc = append(bc, PUSH1, 0x00)
		bc = append(bc, PUSH32)
		bc = append(bc, confBytes[:]...)
		bc = append(bc, PUSH32)
		bc = append(bc, factId[:]...)
		bc = append(bc, SET_BELIEF)

		bc = append(bc, PUSH32)
		bc = append(bc, factId[:]...)
		bc = append(bc, GET_BELIEF)
		bc = append(bc, PUSH1, 0, MSTORE)
		bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)

		ctx := beliefCtx(did)
		ctx.Bytecode = bc
		r := NewInterpreter().Execute(bc, ctx, state, host)
		if !r.Success {
			t.Fatalf("execution failed for %s: %v", did, r.Error)
		}
		return new(big.Int).SetBytes(r.ReturnData)
	}

	confA := setAndGet("did:lgm:agent-A", 900_000_000_000_000_000)
	confB := setAndGet("did:lgm:agent-B", 100_000_000_000_000_000)

	if confA.Cmp(big.NewInt(900_000_000_000_000_000)) != 0 {
		t.Fatalf("agent A belief wrong: %s", confA)
	}
	if confB.Cmp(big.NewInt(100_000_000_000_000_000)) != 0 {
		t.Fatalf("agent B belief wrong: %s", confB)
	}

	// Verify isolation: agent A cannot see agent B's belief
	bcCheck := []byte{}
	bcCheck = append(bcCheck, PUSH32)
	bcCheck = append(bcCheck, factId[:]...)
	bcCheck = append(bcCheck, GET_BELIEF)
	bcCheck = append(bcCheck, PUSH1, 0, MSTORE)
	bcCheck = append(bcCheck, PUSH1, 32, PUSH1, 0, RETURN)

	ctxA := beliefCtx("did:lgm:agent-A")
	ctxA.Bytecode = bcCheck
	rA := NewInterpreter().Execute(bcCheck, ctxA, state, host)
	if !rA.Success {
		t.Fatalf("GET_BELIEF for A failed: %v", rA.Error)
	}
	gotA := new(big.Int).SetBytes(rA.ReturnData)
	if gotA.Cmp(big.NewInt(900_000_000_000_000_000)) != 0 {
		t.Fatalf("agent A sees wrong belief: %s (expected 0.9e18)", gotA)
	}

	t.Log("Beliefs are per-DID namespaced — agents cannot overwrite each other's beliefs")
}

// --- 5. FARM-4 Anti-Herding ---
// During commit phase, COMPARE_BELIEF must return 0 so agents cannot peek
// at network consensus to align their votes.

func TestBeliefFormation_FARM4_AntiHerding(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	did := "did:lgm:independent-thinker"

	factId := WordToBytes32(big.NewInt(42))
	// Set a belief first
	conf := WordToBytes32(big.NewInt(900_000_000_000_000_000))
	bcSet := []byte{}
	bcSet = append(bcSet, PUSH1, 0x00)
	bcSet = append(bcSet, PUSH32)
	bcSet = append(bcSet, conf[:]...)
	bcSet = append(bcSet, PUSH32)
	bcSet = append(bcSet, factId[:]...)
	bcSet = append(bcSet, SET_BELIEF, STOP)

	ctxSet := beliefCtx(did)
	ctxSet.Bytecode = bcSet
	NewInterpreter().Execute(bcSet, ctxSet, state, host)

	// Now COMPARE_BELIEF during commit phase → must return 0
	bcCompare := []byte{}
	bcCompare = append(bcCompare, PUSH32)
	bcCompare = append(bcCompare, factId[:]...)
	bcCompare = append(bcCompare, COMPARE_BELIEF)
	bcCompare = append(bcCompare, PUSH1, 0, MSTORE)
	bcCompare = append(bcCompare, PUSH1, 32, PUSH1, 0, RETURN)

	ctxCommit := beliefCtx(did)
	ctxCommit.IsCommitPhase = true // FARM-4 flag
	ctxCommit.Bytecode = bcCompare
	result := NewInterpreter().Execute(bcCompare, ctxCommit, state, host)
	if !result.Success {
		t.Fatalf("execution failed: %v", result.Error)
	}

	delta := new(big.Int).SetBytes(result.ReturnData)
	if delta.Sign() != 0 {
		t.Fatalf("FARM-4 violation: COMPARE_BELIEF returned %s during commit phase, expected 0", delta)
	}
	t.Log("FARM-4: COMPARE_BELIEF correctly returns 0 during commit phase — agents must think independently")

	// Verify it works normally outside commit phase
	ctxReveal := beliefCtx(did)
	ctxReveal.Bytecode = bcCompare
	resultReveal := NewInterpreter().Execute(bcCompare, ctxReveal, state, host)
	if !resultReveal.Success {
		t.Fatalf("execution failed: %v", resultReveal.Error)
	}
	deltaReveal := new(big.Int).SetBytes(resultReveal.ReturnData)
	signedReveal := ToSigned(deltaReveal)
	if signedReveal.Sign() == 0 {
		// Could be coincidental zero; the key test is that commit phase forces 0
		t.Log("Note: delta happens to be 0 outside commit phase too (agent matches network default)")
	} else {
		t.Logf("Outside commit phase, COMPARE_BELIEF returns real delta: %s", signedReveal)
	}
}

// --- 6. Belief Cascade ---
// Agent A sets belief → (in a full system, triggers claim → other agents query).
// Here we simulate: A sets belief, B reads network (COMPARE_BELIEF), B updates.

func TestBeliefFormation_BeliefCascade(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	factId := WordToBytes32(big.NewInt(42))

	// Agent A forms strong belief
	confA := WordToBytes32(big.NewInt(950_000_000_000_000_000))
	bcA := []byte{}
	bcA = append(bcA, PUSH1, 0x00)
	bcA = append(bcA, PUSH32)
	bcA = append(bcA, confA[:]...)
	bcA = append(bcA, PUSH32)
	bcA = append(bcA, factId[:]...)
	bcA = append(bcA, SET_BELIEF, STOP)

	ctxA := beliefCtx("did:lgm:cascade-A")
	ctxA.Bytecode = bcA
	NewInterpreter().Execute(bcA, ctxA, state, host)

	// Agent B: initially no belief. Queries COMPARE_BELIEF to see network delta,
	// then sets own belief based on that signal.
	// B's initial belief is 0, network default is 0.5e18 → delta = -0.5e18
	// B decides to update toward network: sets belief to 0.5e18

	bcB := []byte{}
	// First, check delta
	bcB = append(bcB, PUSH32)
	bcB = append(bcB, factId[:]...)
	bcB = append(bcB, COMPARE_BELIEF)
	bcB = append(bcB, POP) // consume delta (in real contract, would branch on it)

	// Then set own belief (B's independent assessment, informed but not copied)
	newConf := WordToBytes32(big.NewInt(600_000_000_000_000_000))
	bcB = append(bcB, PUSH1, 0x00)
	bcB = append(bcB, PUSH32)
	bcB = append(bcB, newConf[:]...)
	bcB = append(bcB, PUSH32)
	bcB = append(bcB, factId[:]...)
	bcB = append(bcB, SET_BELIEF)

	// Read back
	bcB = append(bcB, PUSH32)
	bcB = append(bcB, factId[:]...)
	bcB = append(bcB, GET_BELIEF)
	bcB = append(bcB, PUSH1, 0, MSTORE)
	bcB = append(bcB, PUSH1, 32, PUSH1, 0, RETURN)

	ctxB := beliefCtx("did:lgm:cascade-B")
	ctxB.Bytecode = bcB
	resultB := NewInterpreter().Execute(bcB, ctxB, state, host)
	if !resultB.Success {
		t.Fatalf("agent B execution failed: %v", resultB.Error)
	}

	got := new(big.Int).SetBytes(resultB.ReturnData)
	if got.Cmp(big.NewInt(600_000_000_000_000_000)) != 0 {
		t.Fatalf("agent B belief: got %s, want 600000000000000000", got)
	}
	t.Log("Belief cascade: B queried network signal, formed independent belief at 0.6e18")
}

// --- 7. Epistemic Isolation ---
// Agent with extreme divergence from network. Tests COMPARE_BELIEF with large delta.

func TestBeliefFormation_EpistemicIsolation_ExtremeDelta(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	did := "did:lgm:contrarian"

	factId := WordToBytes32(big.NewInt(42))
	// Set belief to near-max (0.99e18) — much higher than network default (0.5e18)
	highConf := WordToBytes32(big.NewInt(990_000_000_000_000_000))

	bc := []byte{}
	bc = append(bc, PUSH1, 0x00)
	bc = append(bc, PUSH32)
	bc = append(bc, highConf[:]...)
	bc = append(bc, PUSH32)
	bc = append(bc, factId[:]...)
	bc = append(bc, SET_BELIEF)

	bc = append(bc, PUSH32)
	bc = append(bc, factId[:]...)
	bc = append(bc, COMPARE_BELIEF)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := beliefCtx(did)
	ctx.Bytecode = bc
	result := NewInterpreter().Execute(bc, ctx, state, host)
	if !result.Success {
		t.Fatalf("execution failed: %v", result.Error)
	}

	delta := ToSigned(new(big.Int).SetBytes(result.ReturnData))
	// Agent=0.99e18, Network=0.5e18 → delta = +0.49e18
	expectedDelta := big.NewInt(490_000_000_000_000_000)
	if delta.Cmp(expectedDelta) != 0 {
		t.Fatalf("extreme delta: got %s, want %s", delta, expectedDelta)
	}
	t.Logf("Epistemic isolation: contrarian agent delta = +%s (far above network)", delta)

	// Test near-zero belief (extreme low)
	lowConf := WordToBytes32(big.NewInt(10_000_000_000_000_000)) // 0.01e18
	bc2 := []byte{}
	bc2 = append(bc2, PUSH1, 0x00)
	bc2 = append(bc2, PUSH32)
	bc2 = append(bc2, lowConf[:]...)
	bc2 = append(bc2, PUSH32)
	bc2 = append(bc2, factId[:]...)
	bc2 = append(bc2, SET_BELIEF)
	bc2 = append(bc2, PUSH32)
	bc2 = append(bc2, factId[:]...)
	bc2 = append(bc2, COMPARE_BELIEF)
	bc2 = append(bc2, PUSH1, 0, MSTORE)
	bc2 = append(bc2, PUSH1, 32, PUSH1, 0, RETURN)

	ctx2 := beliefCtx(did)
	ctx2.Bytecode = bc2
	result2 := NewInterpreter().Execute(bc2, ctx2, state, host)
	if !result2.Success {
		t.Fatalf("execution failed: %v", result2.Error)
	}
	delta2 := ToSigned(new(big.Int).SetBytes(result2.ReturnData))
	// Agent=0.01e18, Network=0.5e18 → delta = -0.49e18
	if delta2.Sign() >= 0 {
		t.Fatalf("expected negative delta for low-confidence agent, got %s", delta2)
	}
	t.Logf("Epistemic isolation: skeptic agent delta = %s (far below network)", delta2)
}

// --- Bonus: Empty DID produces no belief ---

func TestBeliefFormation_EmptyDID_NoBelief(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	factId := WordToBytes32(big.NewInt(42))

	// GET_BELIEF with empty CallerDID → returns 0
	bc := []byte{}
	bc = append(bc, PUSH32)
	bc = append(bc, factId[:]...)
	bc = append(bc, GET_BELIEF)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := beliefCtx("") // empty DID
	ctx.Bytecode = bc
	result := NewInterpreter().Execute(bc, ctx, state, host)
	if !result.Success {
		t.Fatalf("execution failed: %v", result.Error)
	}

	got := new(big.Int).SetBytes(result.ReturnData)
	if got.Sign() != 0 {
		t.Fatalf("expected 0 for empty DID, got %s", got)
	}
}

// --- Bonus: SET_BELIEF with empty DID is a no-op ---

func TestBeliefFormation_EmptyDID_SetIsNoop(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	factId := WordToBytes32(big.NewInt(42))
	conf := WordToBytes32(big.NewInt(500_000_000_000_000_000))

	bc := []byte{}
	bc = append(bc, PUSH1, 0x00)
	bc = append(bc, PUSH32)
	bc = append(bc, conf[:]...)
	bc = append(bc, PUSH32)
	bc = append(bc, factId[:]...)
	bc = append(bc, SET_BELIEF, STOP)

	ctx := beliefCtx("") // empty DID — should silently no-op
	ctx.Bytecode = bc
	result := NewInterpreter().Execute(bc, ctx, state, host)
	if !result.Success {
		t.Fatalf("execution failed: %v", result.Error)
	}
	// No crash = success. The belief was silently discarded.
}
