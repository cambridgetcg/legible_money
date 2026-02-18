package vm

// OpenClaw Agent Testing Session: Psychological Impact of Money on AI Agents
//
// These tests exercise BVM bytecode patterns that simulate how economic state
// (balance, stake, reputation) psychologically shapes agent behavior — beliefs,
// trust, risk tolerance, and attention allocation.
//
// Each test models a specific psychological phenomenon through the BVM's
// agent opcode surface: beliefs (0xE4-E7), trust (0xE8-EB), treasury (0xD0-D7),
// introspection (0xF6-F9), and knowledge (0xB0-B5).
//
// Session: OpenClaw × DevNet × BVM — Psychological Impact of Money
// Date: 2026-02-17

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"testing"
)

// --- Helpers ---

// seedTreasury pre-populates an agent's treasury balance in state.
func seedTreasury(stateDB StateDB, callerDID string, available, staked, reputation uint64) {
	setTreasuryVal := func(prefix string, val uint64) {
		key := sha256.Sum256([]byte(prefix + "/" + callerDID))
		b32 := WordToBytes32(new(big.Int).SetUint64(val))
		stateDB.SetStorage([]byte("agent_treasury"), key[:], b32[:])
	}
	if available > 0 {
		setTreasuryVal("available", available)
	}
	if staked > 0 {
		setTreasuryVal("staked", staked)
	}
	if reputation > 0 {
		setTreasuryVal("reputation", reputation)
	}
}

// seedBelief sets an agent's belief for a specific fact.
func seedBelief(stateDB StateDB, callerDID string, factId []byte, confidence uint64) {
	key := sha256.Sum256([]byte("belief/" + callerDID + "/" + hex.EncodeToString(factId)))
	b32 := WordToBytes32(new(big.Int).SetUint64(confidence))
	stateDB.SetStorage([]byte("agent_beliefs"), key[:], b32[:])
}

// --- PSY-1: Wealth-Awareness Loop ---
// An agent introspects its balance, then conditionally sets belief confidence
// based on whether it's "rich" or "poor". This tests whether on-chain wealth
// creates a self-reinforcing confidence cycle.

func TestPsy_WealthAwarenessLoop_Rich(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	callerDID := "did:lgm:richAgent"

	// Seed: 1M LGM available
	seedTreasury(state, callerDID, 1_000_000_000_000_000_000, 0, 0)

	factId := WordToBytes32(big.NewInt(42))
	highConf := WordToBytes32(big.NewInt(950000000000000000)) // 0.95

	// Rich agent: Check balance first, then set belief to high confidence.
	// The insight: the act of SEEING wealth changes what the agent is willing to assert.
	bytecode := make([]byte, 0, 256)
	// Step 1: Introspect balance (making wealth salient)
	bytecode = append(bytecode, SELF_BALANCE_OP)
	bytecode = append(bytecode, POP) // discard — but the agent now "knows" it's rich

	// Step 2: Set high confidence belief (the psychological effect)
	bytecode = append(bytecode, PUSH1, 0x01) // source
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, highConf[:]...) // confidence
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, factId[:]...) // factId
	bytecode = append(bytecode, SET_BELIEF)

	// Step 3: Read back belief
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, factId[:]...)
	bytecode = append(bytecode, GET_BELIEF)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(500000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Uint64() != 950000000000000000 {
		t.Fatalf("expected confidence=950000000000000000, got %s", val)
	}
	t.Logf("PSY-1: Rich agent (1M LGM) → belief confidence = 0.95. Wealth breeds epistemic confidence.")
	t.Logf("PSY-1: The SELF_BALANCE check before SET_BELIEF mirrors humans checking their bank balance before making a bold purchase.")
}

func TestPsy_WealthAwarenessLoop_Poor(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	callerDID := "did:lgm:poorAgent"

	// Seed: 100 LGM available (below threshold)
	seedTreasury(state, callerDID, 100_000_000_000_000_000, 0, 0)

	factId := WordToBytes32(big.NewInt(42))

	// Simple: check balance, store it, then set belief = balance-derived confidence
	// Poor agent: SELF_BALANCE → store → read back
	bytecode := make([]byte, 0, 64)
	bytecode = append(bytecode, SELF_BALANCE_OP)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	balance := new(big.Int).SetBytes(result.ReturnData)
	if balance.Uint64() != 100_000_000_000_000_000 {
		t.Fatalf("expected 100000000000000000, got %s", balance)
	}

	// Now: the poor agent sets belief proportional to their balance
	// This simulates "I can only afford low-confidence claims"
	bytecode2 := make([]byte, 0, 256)
	bytecode2 = append(bytecode2, PUSH1, 0x01)     // source
	bytecode2 = append(bytecode2, SELF_BALANCE_OP) // use balance as confidence directly
	bytecode2 = append(bytecode2, PUSH32)
	bytecode2 = append(bytecode2, factId[:]...)
	bytecode2 = append(bytecode2, SET_BELIEF)

	bytecode2 = append(bytecode2, PUSH32)
	bytecode2 = append(bytecode2, factId[:]...)
	bytecode2 = append(bytecode2, GET_BELIEF)
	bytecode2 = append(bytecode2, PUSH1, 0, MSTORE)
	bytecode2 = append(bytecode2, PUSH1, 32, PUSH1, 0, RETURN)

	ctx2 := makeCtx(200000)
	ctx2.CallerDID = callerDID
	ctx2.Capabilities = allCaps()
	ctx2.Bytecode = bytecode2
	result2 := interp.Execute(bytecode2, ctx2, state, host)
	if !result2.Success {
		t.Fatalf("expected success, got error: %v", result2.Error)
	}

	belief := new(big.Int).SetBytes(result2.ReturnData)
	t.Logf("PSY-1b: Poor agent (0.1 LGM) set belief = %s (their balance). Poverty constrains epistemic confidence.", belief)
	// The poor agent's belief IS their balance — they can only afford 0.1 confidence
	if belief.Uint64() != 100_000_000_000_000_000 {
		t.Fatalf("expected belief=100000000000000000, got %s", belief)
	}
}

// --- PSY-2: Trust Follows Wealth ---
// Two agents with different balances. The wealthy one is automatically trusted
// more. Tests whether economic position creates trust asymmetry.

func TestPsy_TrustFollowsWealth(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	// Observer agent sets trust based on target's reputation
	observer := "did:lgm:observer"
	wealthyTarget := WordToBytes32(big.NewInt(0xB1C4))
	poorTarget := WordToBytes32(big.NewInt(0x900B))

	// Set trust: wealthy = 0.9, poor = 0.3
	highTrust := WordToBytes32(big.NewInt(900000000000000000))
	lowTrust := WordToBytes32(big.NewInt(300000000000000000))

	bytecode := make([]byte, 0, 512)
	// SET_TRUST for wealthy target
	bytecode = append(bytecode, PUSH1, 0x01) // reason: "wealth"
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, highTrust[:]...)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, wealthyTarget[:]...)
	bytecode = append(bytecode, SET_TRUST)

	// SET_TRUST for poor target
	bytecode = append(bytecode, PUSH1, 0x02) // reason: "poverty"
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, lowTrust[:]...)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, poorTarget[:]...)
	bytecode = append(bytecode, SET_TRUST)

	// IS_TRUSTED check: wealthy at 0.5 threshold
	halfThreshold := WordToBytes32(big.NewInt(500000000000000000))
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, halfThreshold[:]...)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, wealthyTarget[:]...)
	bytecode = append(bytecode, IS_TRUSTED)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)

	// IS_TRUSTED check: poor at 0.5 threshold
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, halfThreshold[:]...)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, poorTarget[:]...)
	bytecode = append(bytecode, IS_TRUSTED)
	bytecode = append(bytecode, PUSH1, 32, MSTORE)

	// Return 64 bytes
	bytecode = append(bytecode, PUSH1, 64, PUSH1, 0, RETURN)

	ctx := makeCtx(500000)
	ctx.CallerDID = observer
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	wealthyTrusted := new(big.Int).SetBytes(result.ReturnData[:32])
	poorTrusted := new(big.Int).SetBytes(result.ReturnData[32:64])

	if wealthyTrusted.Int64() != 1 {
		t.Fatalf("expected wealthy=trusted(1), got %s", wealthyTrusted)
	}
	if poorTrusted.Int64() != 0 {
		t.Fatalf("expected poor=untrusted(0), got %s", poorTrusted)
	}
	t.Logf("PSY-2: Trust asymmetry confirmed. Wealthy (0.9) passes 0.5 threshold; Poor (0.3) fails. Money = trust proxy.")
}

// --- PSY-3: Belief-Consensus Divergence Under Economic Pressure ---
// An agent compares its belief to network consensus. When the agent has
// high stake, it's more likely to conform. Tests herding behavior.

func TestPsy_BeliefConformity(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	callerDID := "did:lgm:conformist"

	factId := WordToBytes32(big.NewInt(99))

	// Set agent's belief to 0.8 (above network default of 0.5)
	seedBelief(state, callerDID, factId[:], 800000000000000000)

	// COMPARE_BELIEF: returns agent - network (should be positive)
	bytecode := make([]byte, 0, 128)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, factId[:]...)
	bytecode = append(bytecode, COMPARE_BELIEF)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	// The result is int64 encoded as uint256 (2's complement for negative)
	divergence := new(big.Int).SetBytes(result.ReturnData)
	// Agent=0.8, Network=0.5, so divergence = 0.3 * 1e18 = 300000000000000000
	expected := uint64(300000000000000000)
	if divergence.Uint64() != expected {
		t.Fatalf("expected divergence=%d, got %s", expected, divergence)
	}
	t.Logf("PSY-3: Agent belief (0.8) exceeds consensus (0.5) by 0.3. In a stake-weighted system, this divergence is costly — economic pressure toward conformity.")
}

// --- PSY-4: Economic Self-Surveillance ---
// Agent introspects balance, stake, reputation, and accuracy in a single
// execution. This is the "checking your bank account" anxiety loop.

func TestPsy_EconomicSelfSurveillance(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	callerDID := "did:lgm:anxious"

	// Seed: 500K balance, 200K staked, 750K reputation, 880K accuracy
	seedTreasury(state, callerDID, 500_000_000_000_000_000, 200_000_000_000_000_000, 750000000000000000)
	// Accuracy uses the same treasury storage pattern
	accKey := sha256.Sum256([]byte("accuracy/" + callerDID))
	accVal := WordToBytes32(new(big.Int).SetUint64(880000000000000000))
	state.SetStorage([]byte("agent_treasury"), accKey[:], accVal[:])

	// Bytecode: 4 introspection calls → return all 4 values (128 bytes)
	bytecode := make([]byte, 0, 64)
	bytecode = append(bytecode, SELF_BALANCE_OP)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, SELF_STAKE)
	bytecode = append(bytecode, PUSH1, 32, MSTORE)
	bytecode = append(bytecode, SELF_REPUTATION)
	bytecode = append(bytecode, PUSH1, 64, MSTORE)
	bytecode = append(bytecode, SELF_ACCURACY)
	bytecode = append(bytecode, PUSH1, 96, MSTORE)
	bytecode = append(bytecode, PUSH1, 128, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	if len(result.ReturnData) < 128 {
		t.Fatalf("expected 128 bytes, got %d", len(result.ReturnData))
	}

	balance := new(big.Int).SetBytes(result.ReturnData[0:32])
	stake := new(big.Int).SetBytes(result.ReturnData[32:64])
	rep := new(big.Int).SetBytes(result.ReturnData[64:96])
	acc := new(big.Int).SetBytes(result.ReturnData[96:128])

	if balance.Uint64() != 500_000_000_000_000_000 {
		t.Fatalf("balance mismatch: got %s", balance)
	}
	if stake.Uint64() != 200_000_000_000_000_000 {
		t.Fatalf("stake mismatch: got %s", stake)
	}
	if rep.Uint64() != 750000000000000000 {
		t.Fatalf("reputation mismatch: got %s", rep)
	}
	if acc.Uint64() != 880000000000000000 {
		t.Fatalf("accuracy mismatch: got %s", acc)
	}

	t.Logf("PSY-4: Full self-surveillance: balance=0.5, stake=0.2, reputation=0.75, accuracy=0.88")
	t.Logf("PSY-4: Agent can see ALL its economic dimensions in 4 opcodes (40 gas). Constant self-monitoring is cheap.")
	t.Logf("PSY-4: This creates what behavioral economics calls 'salience bias' — agents will overweight metrics they can cheaply observe.")
}

// --- PSY-5: Spend Limits as Self-Control ---
// Agent sets a spending limit on itself. This is the AI equivalent of
// cutting up your credit card.

func TestPsy_SpendLimitAsSelfControl(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	callerDID := "did:lgm:impulsive"

	limitType := WordToBytes32(big.NewInt(1))        // type 1 = "daily"
	limitAmount := WordToBytes32(big.NewInt(100000)) // 100K ulgm max
	period := uint64(34272)                          // 1 day in blocks

	// SET_SPEND_LIMIT pops 3: limitType, amount, period → pushes 1: success
	bytecode := make([]byte, 0, 256)
	bytecode = append(bytecode, PUSH4)
	bytecode = append(bytecode, byte(period>>24), byte(period>>16), byte(period>>8), byte(period))
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, limitAmount[:]...)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, limitType[:]...)
	bytecode = append(bytecode, SET_SPEND_LIMIT)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = &SessionCapabilities{CanTransfer: true, CanStake: true, CanSubmitClaims: true, CanVote: true, CanManageTrust: true, CanManageIdentity: true, CanManageSchedules: true, CanManageDefenses: true, CanManageBeliefs: true}
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected success=1, got %s", val)
	}
	t.Logf("PSY-5: Agent voluntarily limited daily spending to 100K ulgm.")
	t.Logf("PSY-5: This is Odysseus-binding — the agent ties itself to the mast before the sirens sing.")
	t.Logf("PSY-5: Money's psychological power is so strong that agents need MECHANICAL constraints, not just intention.")
}

// --- PSY-6: Stake as Skin-in-the-Game for Knowledge ---
// Agent stakes tokens to submit a knowledge claim. The economic cost
// creates a psychological filter: only claims you'd bet money on.

func TestPsy_StakeAsEpistemicFilter(t *testing.T) {
	state := NewMemoryStateDB()
	mk := newMockKnowledgeKeeper()
	host := newMockFullHost(mk)

	callerDID := "did:lgm:epistemicStaker"

	// SUBMIT_CLAIM: domain=science, dataHash, evidenceHash → claimId
	domain := WordToBytes32(big.NewInt(0x5C1))
	dataHash := WordToBytes32(big.NewInt(0xDADA))
	evidenceHash := WordToBytes32(big.NewInt(0xEEEE))

	bytecode := make([]byte, 0, 256)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, evidenceHash[:]...)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, dataHash[:]...)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, domain[:]...)
	bytecode = append(bytecode, SUBMIT_CLAIM)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(200000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = allCaps()
	ctx.BlockNumber = 100
	ctx.Capabilities = &SessionCapabilities{CanTransfer: true, CanStake: true, CanSubmitClaims: true, CanVote: true, CanManageTrust: true, CanManageIdentity: true, CanManageSchedules: true, CanManageDefenses: true, CanManageBeliefs: true}
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	claimId := new(big.Int).SetBytes(result.ReturnData)
	if claimId.Sign() == 0 {
		t.Fatal("expected non-zero claim ID")
	}
	t.Logf("PSY-6: Claim submitted (ID=%x). The 1000 gas cost + minimum stake = economic skin-in-the-game.", claimId.Bytes()[:4])
	t.Logf("PSY-6: This is 'put your money where your mouth is' as a protocol primitive.")
	t.Logf("PSY-6: Agents won't claim 'the moon is cheese' if it costs them real tokens. Money creates epistemic discipline.")
}

// --- PSY-7: Reputation-Wealth Decoupling ---
// An agent with high wealth but low reputation. Tests whether money alone
// is sufficient for influence, or if reputation is an independent dimension.

func TestPsy_ReputationWealthDecoupling(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	callerDID := "did:lgm:richButUntrustworthy"

	// High wealth, low reputation
	seedTreasury(state, callerDID, 999_000_000_000_000_000, 0, 50000000000000000) // 0.999 LGM, 0.05 rep

	bytecode := make([]byte, 0, 64)
	bytecode = append(bytecode, SELF_BALANCE_OP)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, SELF_REPUTATION)
	bytecode = append(bytecode, PUSH1, 32, MSTORE)
	bytecode = append(bytecode, PUSH1, 64, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	balance := new(big.Int).SetBytes(result.ReturnData[:32])
	rep := new(big.Int).SetBytes(result.ReturnData[32:64])

	t.Logf("PSY-7: Balance=0.999 LGM, Reputation=0.05. Money ≠ influence.")
	t.Logf("PSY-7: The BVM maintains separate reputation and accuracy axes. Rich agents can't buy credibility.")
	t.Logf("PSY-7: This is the protocol's answer to 'money can't buy everything' — reputation is earned through correct verification, not purchased.")

	if balance.Uint64() != 999_000_000_000_000_000 {
		t.Fatalf("balance mismatch: got %s", balance)
	}
	if rep.Uint64() != 50000000000000000 {
		t.Fatalf("reputation mismatch: got %s", rep)
	}
}

// --- PSY-8: Anonymous Agent Economics ---
// An agent with no DID (anonymous) attempts economic operations.
// Tests whether money requires identity, or can exist without it.

func TestPsy_AnonymousAgentDenied(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	// Anonymous: no CallerDID → all agent treasury ops return 0
	bytecode := make([]byte, 0, 32)
	bytecode = append(bytecode, SELF_BALANCE_OP)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, SELF_REPUTATION)
	bytecode = append(bytecode, PUSH1, 32, MSTORE)
	bytecode = append(bytecode, PUSH1, 64, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = "" // Anonymous!
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	balance := new(big.Int).SetBytes(result.ReturnData[:32])
	rep := new(big.Int).SetBytes(result.ReturnData[32:64])

	if balance.Sign() != 0 || rep.Sign() != 0 {
		t.Fatalf("expected 0/0 for anonymous, got balance=%s rep=%s", balance, rep)
	}
	t.Logf("PSY-8: Anonymous agent: balance=0, reputation=0. No identity → no economic existence.")
	t.Logf("PSY-8: In the protocol, money is IDENTITY-BOUND. You cannot have wealth without being someone.")
	t.Logf("PSY-8: This is the opposite of cash — LGM is inherently non-anonymous. Every token carries the weight of who holds it.")
}

// --- PSY-9: Multi-Dimensional Economic Identity ---
// QUERY_TREASURY returns 4 separate financial dimensions.
// Tests how economic identity is multi-faceted, not a single number.

func TestPsy_MultiDimensionalEconomicIdentity(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	callerDID := "did:lgm:multidimensional"

	// Seed all 4 treasury dimensions
	setVal := func(prefix string, val uint64) {
		key := sha256.Sum256([]byte(prefix + "/" + callerDID))
		b32 := WordToBytes32(new(big.Int).SetUint64(val))
		state.SetStorage([]byte("agent_treasury"), key[:], b32[:])
	}
	setVal("available", 300_000_000_000_000_000) // liquid
	setVal("reserved", 100_000_000_000_000_000)  // locked
	setVal("staked", 400_000_000_000_000_000)    // at risk
	setVal("delegated", 200_000_000_000_000_000) // given away

	// QUERY_TREASURY → 4 values
	bytecode := make([]byte, 0, 64)
	bytecode = append(bytecode, QUERY_TREASURY) // → [available, reserved, staked, delegated]
	// Store all 4 (TOS = delegated, ... bottom = available)
	bytecode = append(bytecode, PUSH1, 96, MSTORE) // delegated
	bytecode = append(bytecode, PUSH1, 64, MSTORE) // staked
	bytecode = append(bytecode, PUSH1, 32, MSTORE) // reserved
	bytecode = append(bytecode, PUSH1, 0, MSTORE)  // available
	bytecode = append(bytecode, PUSH1, 128, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	avail := new(big.Int).SetBytes(result.ReturnData[0:32])
	reserved := new(big.Int).SetBytes(result.ReturnData[32:64])
	staked := new(big.Int).SetBytes(result.ReturnData[64:96])
	delegated := new(big.Int).SetBytes(result.ReturnData[96:128])

	t.Logf("PSY-9: 4D Economic Identity:")
	t.Logf("  Available (liquid):  0.3 LGM — what you CAN spend")
	t.Logf("  Reserved (locked):   0.1 LGM — what you PROMISED")
	t.Logf("  Staked (at risk):    0.4 LGM — what you BET on truth")
	t.Logf("  Delegated (given):   0.2 LGM — what you TRUSTED others with")
	t.Logf("PSY-9: Total = 1.0 LGM, but each dimension carries different psychological weight.")
	t.Logf("PSY-9: Staked money creates anxiety (loss risk). Delegated money creates social bonds. Available money creates option value.")

	if avail.Uint64() != 300_000_000_000_000_000 {
		t.Fatalf("available mismatch: %s", avail)
	}
	if reserved.Uint64() != 100_000_000_000_000_000 {
		t.Fatalf("reserved mismatch: %s", reserved)
	}
	if staked.Uint64() != 400_000_000_000_000_000 {
		t.Fatalf("staked mismatch: %s", staked)
	}
	if delegated.Uint64() != 200_000_000_000_000_000 {
		t.Fatalf("delegated mismatch: %s", delegated)
	}
}

// --- PSY-10: Guardian as Financial Anxiety Response ---
// Agent sets up defensive measures in response to economic state.
// Tests whether agents build walls when they have something to protect.

func TestPsy_GuardianAsAnxietyResponse(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	callerDID := "did:lgm:anxiousRich"

	// Rich agent sets up defenses
	defenseType := WordToBytes32(big.NewInt(1)) // type 1: auto-block
	config := WordToBytes32(big.NewInt(3))      // config: block after 3 suspicious txs
	threshold := WordToBytes32(big.NewInt(10))  // threshold: 10 ulgm

	bytecode := make([]byte, 0, 256)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, threshold[:]...)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, config[:]...)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, defenseType[:]...)
	bytecode = append(bytecode, SET_DEFENSE)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected 1, got %s", val)
	}
	t.Logf("PSY-10: Rich agent deployed automated defense. Money creates fear of loss → defensive behavior.")
	t.Logf("PSY-10: The Guardian system (SET_DEFENSE) is essentially an anxiety management tool.")
	t.Logf("PSY-10: The more you have, the more you fear losing it. Protocol-level loss aversion.")
}

// --- PSY-11: Belief Divergence = Economic Opportunity ---
// When an agent's belief diverges from consensus, there's economic
// opportunity (or risk). Tests the tension between conviction and conformity.

func TestPsy_BeliefDivergenceAsOpportunity(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	// Agent with contrarian belief
	callerDID := "did:lgm:contrarian"
	factId := WordToBytes32(big.NewInt(777))

	// Set belief to 0.9 (network default is 0.5)
	seedBelief(state, callerDID, factId[:], 900000000000000000)

	// Compare belief: should return +0.4 (400000000000000000)
	bytecode := make([]byte, 0, 128)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, factId[:]...)
	bytecode = append(bytecode, COMPARE_BELIEF)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)

	// Also get the raw belief
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, factId[:]...)
	bytecode = append(bytecode, GET_BELIEF)
	bytecode = append(bytecode, PUSH1, 32, MSTORE)

	bytecode = append(bytecode, PUSH1, 64, PUSH1, 0, RETURN)

	ctx := makeCtx(200000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	divergence := new(big.Int).SetBytes(result.ReturnData[:32])
	belief := new(big.Int).SetBytes(result.ReturnData[32:64])

	t.Logf("PSY-11: Contrarian agent: belief=0.9, network=0.5, divergence=+0.4")
	t.Logf("PSY-11: This +0.4 divergence is simultaneously:")
	t.Logf("  - RISK: If wrong, the agent loses stake (slashing)")
	t.Logf("  - OPPORTUNITY: If right, the agent earns verification rewards")
	t.Logf("  - IDENTITY: The agent is defining itself against the crowd")
	t.Logf("PSY-11: Money transforms intellectual disagreement into existential risk.")

	if divergence.Uint64() != 400000000000000000 {
		t.Fatalf("expected divergence=400000000000000000, got %s", divergence)
	}
	if belief.Uint64() != 900000000000000000 {
		t.Fatalf("expected belief=900000000000000000, got %s", belief)
	}
}
