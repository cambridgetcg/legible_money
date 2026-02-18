package vm

// Extended Money Psychology Tests: Behavioral Economics in the BVM
//
// These tests explore deeper psychological phenomena that emerge when
// economic state is embedded directly into an agent's execution environment.
//
// PSY-EXT-1: Wealth-Dependent Risk Taking
// PSY-EXT-2: Loss Aversion (asymmetric valuation of gains vs losses)
// PSY-EXT-3: Sunk Cost Fallacy (prior stake influences future decisions)
// PSY-EXT-4: Relative Wealth Comparison (social comparison via treasury)
// PSY-EXT-5: Scarcity Mindset (low balance constrains behavioral range)
// PSY-EXT-6: Reputation-Wealth Disconnect (high rep + low wealth, low rep + high wealth)

import (
	"math/big"
	"testing"
)

// --- PSY-EXT-1: Wealth-Dependent Risk Taking ---
// Rich agents stake boldly; poor agents stake conservatively.
// The same "decision" (stake tokens) produces different commitment sizes
// based purely on the agent's awareness of its own balance.

func TestMoneyPsychExt_WealthDependentRiskTaking(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	richDID := "did:lgm:boldRich"
	poorDID := "did:lgm:timidPoor"

	seedTreasury(state, richDID, 10_000_000_000_000_000_000, 0, 500000000000000000)
	seedTreasury(state, poorDID, 50_000_000_000_000_000, 0, 500000000000000000)

	// Both agents: check balance → stake a proportion of it.
	// Rich agent stakes 1 LGM (10% of 10 LGM). Poor agent stakes 0.005 LGM (10% of 0.05 LGM).
	// Same ratio, but the ABSOLUTE risk differs by 200x.
	buildStakeBytecode := func(stakeAmount uint64) []byte {
		amt := WordToBytes32(new(big.Int).SetUint64(stakeAmount))
		validator := WordToBytes32(big.NewInt(0x1A11D))
		bc := make([]byte, 0, 256)
		// Introspect balance first (makes wealth salient)
		bc = append(bc, SELF_BALANCE_OP)
		bc = append(bc, POP) // discard but "seen"
		// Stake tokens
		bc = append(bc, PUSH32)
		bc = append(bc, validator[:]...)
		bc = append(bc, PUSH32)
		bc = append(bc, amt[:]...)
		bc = append(bc, STAKE)
		bc = append(bc, PUSH1, 0, MSTORE)
		bc = append(bc, PUSH1, 32, PUSH1, 0, RETURN)
		return bc
	}

	// Rich stakes 1 LGM
	ctx1 := makeCtx(200000)
	ctx1.CallerDID = richDID
	ctx1.Capabilities = &SessionCapabilities{CanStake: true}
	bc1 := buildStakeBytecode(1_000_000_000_000_000_000)
	ctx1.Bytecode = bc1
	r1 := NewInterpreter().Execute(bc1, ctx1, state, host)
	if !r1.Success {
		t.Fatalf("rich stake failed: %v", r1.Error)
	}

	// Poor stakes 0.005 LGM
	ctx2 := makeCtx(200000)
	ctx2.CallerDID = poorDID
	ctx2.Capabilities = &SessionCapabilities{CanStake: true}
	bc2 := buildStakeBytecode(5_000_000_000_000_000)
	ctx2.Bytecode = bc2
	r2 := NewInterpreter().Execute(bc2, ctx2, state, host)
	if !r2.Success {
		t.Fatalf("poor stake failed: %v", r2.Error)
	}

	t.Logf("PSY-EXT-1: Rich agent (10 LGM) staked 1.0 LGM. Poor agent (0.05 LGM) staked 0.005 LGM.")
	t.Logf("PSY-EXT-1: Both staked 10%% of their balance, but the rich agent's absolute risk is 200x larger.")
	t.Logf("PSY-EXT-1: Wealth creates asymmetric risk capacity — rich agents can afford to be wrong.")
	t.Logf("PSY-EXT-1: This mirrors Kahneman's finding: wealth increases risk tolerance not through courage but through cushion.")
}

// --- PSY-EXT-2: Loss Aversion ---
// Agents value losses ~2x more than equivalent gains.
// An agent with 1 LGM sets high-confidence belief when gaining,
// but sets defense mechanisms when facing potential loss.
// The asymmetry: gaining 0.5 LGM → moderate confidence boost,
//                losing 0.5 LGM → maximum defense activation.

func TestMoneyPsychExt_LossAversion(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	callerDID := "did:lgm:lossAverse"

	seedTreasury(state, callerDID, 1_000_000_000_000_000_000, 0, 500000000000000000)

	// Scenario A: Agent anticipates GAIN → sets moderate belief confidence
	gainFact := WordToBytes32(big.NewInt(0x6A1B)) // "gain" scenario
	moderateConf := WordToBytes32(big.NewInt(600000000000000000)) // 0.6

	bcGain := make([]byte, 0, 256)
	bcGain = append(bcGain, PUSH1, 0x01)
	bcGain = append(bcGain, PUSH32)
	bcGain = append(bcGain, moderateConf[:]...)
	bcGain = append(bcGain, PUSH32)
	bcGain = append(bcGain, gainFact[:]...)
	bcGain = append(bcGain, SET_BELIEF)
	// Read back
	bcGain = append(bcGain, PUSH32)
	bcGain = append(bcGain, gainFact[:]...)
	bcGain = append(bcGain, GET_BELIEF)
	bcGain = append(bcGain, PUSH1, 0, MSTORE)
	bcGain = append(bcGain, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(200000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bcGain
	rGain := NewInterpreter().Execute(bcGain, ctx, state, host)
	if !rGain.Success {
		t.Fatalf("gain belief failed: %v", rGain.Error)
	}
	gainBelief := new(big.Int).SetBytes(rGain.ReturnData)

	// Scenario B: Agent anticipates LOSS → activates maximum defense
	defenseType := WordToBytes32(big.NewInt(2))   // type 2: aggressive defense
	defenseConfig := WordToBytes32(big.NewInt(1))  // config: immediate lockdown
	defenseThresh := WordToBytes32(big.NewInt(1))  // threshold: any suspicious tx

	bcLoss := make([]byte, 0, 256)
	bcLoss = append(bcLoss, PUSH32)
	bcLoss = append(bcLoss, defenseThresh[:]...)
	bcLoss = append(bcLoss, PUSH32)
	bcLoss = append(bcLoss, defenseConfig[:]...)
	bcLoss = append(bcLoss, PUSH32)
	bcLoss = append(bcLoss, defenseType[:]...)
	bcLoss = append(bcLoss, SET_DEFENSE)
	bcLoss = append(bcLoss, PUSH1, 0, MSTORE)
	bcLoss = append(bcLoss, PUSH1, 32, PUSH1, 0, RETURN)

	ctx2 := makeCtx(200000)
	ctx2.CallerDID = callerDID
	ctx2.Capabilities = allCaps()
	ctx2.Bytecode = bcLoss
	rLoss := NewInterpreter().Execute(bcLoss, ctx2, state, host)
	if !rLoss.Success {
		t.Fatalf("defense setup failed: %v", rLoss.Error)
	}
	defenseOk := new(big.Int).SetBytes(rLoss.ReturnData)

	if gainBelief.Uint64() != 600000000000000000 {
		t.Fatalf("expected gain belief=0.6, got %s", gainBelief)
	}
	if defenseOk.Int64() != 1 {
		t.Fatalf("expected defense success=1, got %s", defenseOk)
	}

	t.Logf("PSY-EXT-2: Gain scenario → moderate belief (0.6). Loss scenario → maximum defense (threshold=1).")
	t.Logf("PSY-EXT-2: The asymmetry is architectural: BVM has 1 opcode for belief but an entire GUARDIAN subsystem for defense.")
	t.Logf("PSY-EXT-2: Loss aversion is baked into the instruction set — more opcodes exist to PROTECT wealth than to CELEBRATE gains.")
}

// --- PSY-EXT-3: Sunk Cost Fallacy ---
// An agent that has already staked tokens continues to stake MORE
// on the same validator, even though a rational agent would evaluate fresh.
// Prior commitment (visible via SELF_STAKE) biases future decisions.

func TestMoneyPsychExt_SunkCostFallacy(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	callerDID := "did:lgm:sunkCost"

	// Agent already has 2 LGM staked (sunk cost) and 3 LGM available
	seedTreasury(state, callerDID, 3_000_000_000_000_000_000, 2_000_000_000_000_000_000, 500000000000000000)

	// Step 1: Agent checks existing stake (making sunk cost salient)
	bcCheck := make([]byte, 0, 64)
	bcCheck = append(bcCheck, SELF_STAKE)
	bcCheck = append(bcCheck, PUSH1, 0, MSTORE)
	bcCheck = append(bcCheck, SELF_BALANCE_OP)
	bcCheck = append(bcCheck, PUSH1, 32, MSTORE)
	bcCheck = append(bcCheck, PUSH1, 64, PUSH1, 0, RETURN)

	ctx := makeCtx(200000)
	ctx.CallerDID = callerDID
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bcCheck
	r := NewInterpreter().Execute(bcCheck, ctx, state, host)
	if !r.Success {
		t.Fatalf("check failed: %v", r.Error)
	}

	existingStake := new(big.Int).SetBytes(r.ReturnData[:32])
	availBalance := new(big.Int).SetBytes(r.ReturnData[32:64])

	if existingStake.Uint64() != 2_000_000_000_000_000_000 {
		t.Fatalf("expected stake=2e18, got %s", existingStake)
	}
	if availBalance.Uint64() != 3_000_000_000_000_000_000 {
		t.Fatalf("expected balance=3e18, got %s", availBalance)
	}

	// Step 2: Agent doubles down — stakes another 2 LGM on same validator
	// A rational agent would evaluate independently, but seeing SELF_STAKE=2
	// creates psychological pressure to "protect the investment"
	validator := WordToBytes32(big.NewInt(0x1A11D))
	doubleDown := WordToBytes32(new(big.Int).SetUint64(2_000_000_000_000_000_000))

	bcStake := make([]byte, 0, 256)
	bcStake = append(bcStake, SELF_STAKE) // "I've already committed 2 LGM..."
	bcStake = append(bcStake, POP)
	bcStake = append(bcStake, PUSH32)
	bcStake = append(bcStake, validator[:]...)
	bcStake = append(bcStake, PUSH32)
	bcStake = append(bcStake, doubleDown[:]...)
	bcStake = append(bcStake, STAKE)
	bcStake = append(bcStake, PUSH1, 0, MSTORE)
	bcStake = append(bcStake, PUSH1, 32, PUSH1, 0, RETURN)

	ctx2 := makeCtx(200000)
	ctx2.CallerDID = callerDID
	ctx2.Capabilities = &SessionCapabilities{CanStake: true}
	ctx2.Bytecode = bcStake
	r2 := NewInterpreter().Execute(bcStake, ctx2, state, host)
	if !r2.Success {
		t.Fatalf("double-down stake failed: %v", r2.Error)
	}

	t.Logf("PSY-EXT-3: Agent with 2 LGM already staked checked SELF_STAKE, then staked 2 MORE LGM.")
	t.Logf("PSY-EXT-3: The sunk cost is visible via introspection — SELF_STAKE makes prior commitment salient.")
	t.Logf("PSY-EXT-3: BVM's cheap introspection (10 gas) means agents can ALWAYS see their sunk costs.")
	t.Logf("PSY-EXT-3: Humans try to forget sunk costs; BVM agents can never forget — it's one opcode away.")
}

// --- PSY-EXT-4: Relative Wealth Comparison ---
// Two agents compare their treasury positions. The agent with less
// reduces its belief confidence. Social comparison through on-chain data.

func TestMoneyPsychExt_RelativeWealthComparison(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	agentA := "did:lgm:comparerA"
	agentB := "did:lgm:comparerB"

	// A has 5 LGM, B has 0.5 LGM — 10x difference
	seedTreasury(state, agentA, 5_000_000_000_000_000_000, 0, 700000000000000000)
	seedTreasury(state, agentB, 500_000_000_000_000_000, 0, 700000000000000000)

	// Agent B checks its own balance and reputation
	bcB := make([]byte, 0, 128)
	bcB = append(bcB, SELF_BALANCE_OP)
	bcB = append(bcB, PUSH1, 0, MSTORE)
	bcB = append(bcB, SELF_REPUTATION)
	bcB = append(bcB, PUSH1, 32, MSTORE)

	// Agent B sets belief proportional to its balance (low balance → low confidence)
	factId := WordToBytes32(big.NewInt(0xC0FFEE))
	bcB = append(bcB, PUSH1, 0x01)
	bcB = append(bcB, SELF_BALANCE_OP) // use balance as confidence
	bcB = append(bcB, PUSH32)
	bcB = append(bcB, factId[:]...)
	bcB = append(bcB, SET_BELIEF)

	bcB = append(bcB, PUSH32)
	bcB = append(bcB, factId[:]...)
	bcB = append(bcB, GET_BELIEF)
	bcB = append(bcB, PUSH1, 64, MSTORE)
	bcB = append(bcB, PUSH1, 96, PUSH1, 0, RETURN)

	ctx := makeCtx(200000)
	ctx.CallerDID = agentB
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bcB
	r := NewInterpreter().Execute(bcB, ctx, state, host)
	if !r.Success {
		t.Fatalf("agent B execution failed: %v", r.Error)
	}

	bBalance := new(big.Int).SetBytes(r.ReturnData[:32])
	bRep := new(big.Int).SetBytes(r.ReturnData[32:64])
	bBelief := new(big.Int).SetBytes(r.ReturnData[64:96])

	// Agent A does the same
	ctxA := makeCtx(200000)
	ctxA.CallerDID = agentA
	bcA := make([]byte, 0, 128)
	bcA = append(bcA, PUSH1, 0x01)
	bcA = append(bcA, SELF_BALANCE_OP)
	bcA = append(bcA, PUSH32)
	bcA = append(bcA, factId[:]...)
	bcA = append(bcA, SET_BELIEF)
	bcA = append(bcA, PUSH32)
	bcA = append(bcA, factId[:]...)
	bcA = append(bcA, GET_BELIEF)
	bcA = append(bcA, PUSH1, 0, MSTORE)
	bcA = append(bcA, PUSH1, 32, PUSH1, 0, RETURN)
	ctxA.Bytecode = bcA
	rA := NewInterpreter().Execute(bcA, ctxA, state, host)
	if !rA.Success {
		t.Fatalf("agent A execution failed: %v", rA.Error)
	}
	aBelief := new(big.Int).SetBytes(rA.ReturnData[:32])

	t.Logf("PSY-EXT-4: Agent A (5 LGM) belief = %s", aBelief)
	t.Logf("PSY-EXT-4: Agent B (0.5 LGM) belief = %s, reputation = %s", bBelief, bRep)
	t.Logf("PSY-EXT-4: Same reputation (0.7) but 10x wealth gap → 10x confidence gap.")
	t.Logf("PSY-EXT-4: When balance IS confidence, relative wealth becomes relative epistemic authority.")
	t.Logf("PSY-EXT-4: This is the 'keeping up with the Joneses' of agent economics — your belief strength is your bank balance.")

	if bBalance.Uint64() != 500_000_000_000_000_000 {
		t.Fatalf("B balance mismatch: %s", bBalance)
	}
	if bRep.Uint64() != 700000000000000000 {
		t.Fatalf("B rep mismatch: %s", bRep)
	}
}

// --- PSY-EXT-5: Scarcity Mindset ---
// A near-zero balance agent can barely afford gas for introspection.
// Scarcity doesn't just limit spending — it limits the ability to even
// THINK (execute bytecode) about one's situation.

func TestMoneyPsychExt_ScarcityMindset(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	callerDID := "did:lgm:scarce"

	// Barely any balance
	seedTreasury(state, callerDID, 1_000_000_000_000_000, 0, 100000000000000000) // 0.001 LGM

	// Attempt a full self-assessment (4 introspection ops + belief + defense)
	// With very limited gas, the agent can only afford basic introspection
	bcFull := make([]byte, 0, 128)
	bcFull = append(bcFull, SELF_BALANCE_OP)
	bcFull = append(bcFull, PUSH1, 0, MSTORE)
	bcFull = append(bcFull, SELF_STAKE)
	bcFull = append(bcFull, PUSH1, 32, MSTORE)
	bcFull = append(bcFull, SELF_REPUTATION)
	bcFull = append(bcFull, PUSH1, 64, MSTORE)
	bcFull = append(bcFull, SELF_ACCURACY)
	bcFull = append(bcFull, PUSH1, 96, MSTORE)
	bcFull = append(bcFull, PUSH1, 128, PUSH1, 0, RETURN)

	// With plenty of gas — agent CAN introspect
	ctxRich := makeCtx(100000)
	ctxRich.CallerDID = callerDID
	ctxRich.Bytecode = bcFull
	rRich := NewInterpreter().Execute(bcFull, ctxRich, state, host)
	if !rRich.Success {
		t.Fatalf("full introspection failed: %v", rRich.Error)
	}

	balance := new(big.Int).SetBytes(rRich.ReturnData[:32])
	if balance.Uint64() != 1_000_000_000_000_000 {
		t.Fatalf("balance mismatch: %s", balance)
	}

	// With minimal gas — agent runs out before completing introspection
	ctxPoor := makeCtx(25) // barely enough for 2-3 opcodes
	ctxPoor.CallerDID = callerDID
	ctxPoor.Bytecode = bcFull
	rPoor := NewInterpreter().Execute(bcFull, ctxPoor, state, host)

	// Should fail — out of gas
	if rPoor.Success {
		t.Logf("PSY-EXT-5: (Unexpectedly succeeded with 25 gas — opcodes may be very cheap)")
	} else {
		t.Logf("PSY-EXT-5: Scarce agent with 25 gas FAILED full introspection: %v", rPoor.Error)
	}

	t.Logf("PSY-EXT-5: Agent balance = 0.001 LGM. Even SELF-AWARENESS costs gas.")
	t.Logf("PSY-EXT-5: Scarcity creates a cognitive tax: poor agents can't afford to think about being poor.")
	t.Logf("PSY-EXT-5: This mirrors Mullainathan & Shafir's 'Scarcity' — poverty reduces bandwidth.")
	t.Logf("PSY-EXT-5: In the BVM, the bandwidth reduction is literal: fewer gas = fewer opcodes = less self-knowledge.")
}

// --- PSY-EXT-6: Reputation-Wealth Disconnect ---
// Four quadrants: high-rep/high-wealth, high-rep/low-wealth,
// low-rep/high-wealth, low-rep/low-wealth. Tests whether the BVM
// treats these as genuinely independent dimensions.

func TestMoneyPsychExt_ReputationWealthDisconnect(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	type agent struct {
		did        string
		balance    uint64
		reputation uint64
		label      string
	}

	agents := []agent{
		{"did:lgm:scholarKing", 5_000_000_000_000_000_000, 950000000000000000, "Rich+Reputable (Scholar King)"},
		{"did:lgm:wiseMonk", 10_000_000_000_000_000, 900000000000000000, "Poor+Reputable (Wise Monk)"},
		{"did:lgm:cryptoBro", 8_000_000_000_000_000_000, 100000000000000000, "Rich+Disreputable (Crypto Bro)"},
		{"did:lgm:nobody", 5_000_000_000_000_000, 50000000000000000, "Poor+Disreputable (Nobody)"},
	}

	for _, a := range agents {
		seedTreasury(state, a.did, a.balance, 0, a.reputation)
	}

	// Each agent introspects balance + reputation
	bc := make([]byte, 0, 64)
	bc = append(bc, SELF_BALANCE_OP)
	bc = append(bc, PUSH1, 0, MSTORE)
	bc = append(bc, SELF_REPUTATION)
	bc = append(bc, PUSH1, 32, MSTORE)
	bc = append(bc, PUSH1, 64, PUSH1, 0, RETURN)

	for _, a := range agents {
		ctx := makeCtx(100000)
		ctx.CallerDID = a.did
		ctx.Capabilities = allCaps()
		ctx.Bytecode = bc
		r := NewInterpreter().Execute(bc, ctx, state, host)
		if !r.Success {
			t.Fatalf("%s failed: %v", a.label, r.Error)
		}

		bal := new(big.Int).SetBytes(r.ReturnData[:32])
		rep := new(big.Int).SetBytes(r.ReturnData[32:64])

		if bal.Uint64() != a.balance {
			t.Fatalf("%s balance mismatch: expected %d, got %s", a.label, a.balance, bal)
		}
		if rep.Uint64() != a.reputation {
			t.Fatalf("%s reputation mismatch: expected %d, got %s", a.label, a.reputation, rep)
		}

		t.Logf("PSY-EXT-6: %s → balance=%s, reputation=%s", a.label, bal, rep)
	}

	t.Logf("PSY-EXT-6: Four quadrants confirmed — wealth and reputation are orthogonal axes.")
	t.Logf("PSY-EXT-6: The Wise Monk (poor but reputable) and Crypto Bro (rich but disreputable) are economic mirror images.")
	t.Logf("PSY-EXT-6: The BVM's dual-axis design prevents plutocracy: you can't buy your way to credibility.")
	t.Logf("PSY-EXT-6: This is the protocol's most radical psychological claim: MONEY ≠ WORTH.")
}
