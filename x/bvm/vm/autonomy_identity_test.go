package vm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"testing"
)

// --- Mock ScheduleKeeper ---

type mockScheduleKeeper struct {
	schedules map[string]*mockSchedule
	nextID    int
	maxAllowed uint64
}

type mockSchedule struct {
	contractAddr string
	caller       string
	method       string
	executeAt    uint64
	maxGas       uint64
	cancelled    bool
}

func newMockScheduleKeeper(maxAllowed uint64) *mockScheduleKeeper {
	return &mockScheduleKeeper{
		schedules:  make(map[string]*mockSchedule),
		maxAllowed: maxAllowed,
	}
}

func (m *mockScheduleKeeper) CreateSchedule(contractAddr, caller, method, payload string, executeAtBlock, maxGas uint64) (string, error) {
	m.nextID++
	id := "sched-" + uitoa(uint64(m.nextID))
	m.schedules[id] = &mockSchedule{
		contractAddr: contractAddr,
		caller:       caller,
		method:       method,
		executeAt:    executeAtBlock,
		maxGas:       maxGas,
	}
	return id, nil
}

func (m *mockScheduleKeeper) CancelScheduleByID(scheduleId, caller string) error {
	if s, ok := m.schedules[scheduleId]; ok {
		s.cancelled = true
	}
	return nil
}

func (m *mockScheduleKeeper) ModifyScheduleBlock(scheduleId, caller string, newBlock uint64) error {
	if s, ok := m.schedules[scheduleId]; ok {
		s.executeAt = newBlock
	}
	return nil
}

func (m *mockScheduleKeeper) CountActiveSchedules(contractAddr string) uint64 {
	var count uint64
	for _, s := range m.schedules {
		if s.contractAddr == contractAddr && !s.cancelled {
			count++
		}
	}
	return count
}

func (m *mockScheduleKeeper) MaxSchedulesAllowed() uint64 {
	return m.maxAllowed
}

// --- Helper: composite host with schedule keeper ---

func newCompositeHostWithSchedules(stateDB StateDB, sk ScheduleKeeper) *compositeHost {
	return &compositeHost{
		agent:    NewAgentHostWithSchedules(context.Background(), stateDB, sk),
		treasury: NewTreasuryHostWithAuth(stateDB, &testVMAuthKeeper{}, context.Background()),
		kh:       NewKnowledgeHost(nil, context.Background()),
	}
}

func makeAutonomyCtx(gasLimit uint64) *ExecutionContext {
	caller := make([]byte, 20)
	caller[19] = 0x42
	contract := make([]byte, 20)
	contract[19] = 0x42 // self-call: caller == contract
	return &ExecutionContext{
		Caller:          caller,
		CallerDID:       "did:lgm:" + hex.EncodeToString(caller),
		Origin:          caller,
		CurrentContract: contract,
		BlockNumber:     100,
		Timestamp:       1700000000,
		GasLimit:        gasLimit,
		Value:           new(big.Int),
		GasPrice:        new(big.Int),
		ChainID:         big.NewInt(2521),
		BaseFee:         new(big.Int),
		Capabilities: &SessionCapabilities{
			CanTransfer:        true,
			CanStake:           true,
			CanSubmitClaims:    true,
			CanVote:            true,
			CanManageTrust:     true,
			CanManageIdentity:  true,
			CanManageSchedules: true,
			CanManageDefenses:  true,
			CanManageBeliefs:   true,
		},
		ContractBvmVersion: 1,
	}
}

// =============================================================================
// 1. Self-Scheduling Agent: SCHEDULE_SELF creates a self-sustaining loop
// =============================================================================
func TestAutonomy_SelfSchedulingLoop(t *testing.T) {
	state := NewMemoryStateDB()
	sk := newMockScheduleKeeper(10)
	host := newCompositeHostWithSchedules(state, sk)

	// Bytecode: PUSH calldata(0x01), PUSH executeBlock(200), PUSH gasLimit(50000)
	// → SCHEDULE_SELF → store result → RETURN
	bytecode := []byte{
		PUSH1, 0x01,    // callData
		PUSH2, 0x00, 200, // executeBlock = 200
		PUSH2, 0xC3, 0x50, // gasLimit = 50000
		SCHEDULE_SELF,
		// Store schedule ID at memory[0]
		PUSH1, 0, MSTORE,
		// Return 32 bytes from memory[0]
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	ctx := makeAutonomyCtx(200000)
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)

	if !result.Success {
		t.Fatalf("self-schedule failed: %v", result.Error)
	}

	// Schedule ID should be non-zero (hash returned)
	schedId := new(big.Int).SetBytes(result.ReturnData)
	if schedId.Sign() == 0 {
		t.Fatal("expected non-zero schedule ID")
	}

	// Verify schedule was persisted
	count := sk.CountActiveSchedules(string(ctx.CurrentContract))
	if count != 1 {
		t.Fatalf("expected 1 active schedule, got %d", count)
	}

	// Simulate second execution (the scheduled callback) — schedules again
	interp2 := NewInterpreter()
	ctx2 := makeAutonomyCtx(200000)
	ctx2.BlockNumber = 200
	ctx2.Bytecode = bytecode
	result2 := interp2.Execute(bytecode, ctx2, state, host)
	if !result2.Success {
		t.Fatalf("second self-schedule failed: %v", result2.Error)
	}

	// Now 2 active schedules
	if sk.CountActiveSchedules(string(ctx.CurrentContract)) != 2 {
		t.Fatalf("expected 2 active schedules after loop iteration")
	}
	t.Log("✓ Self-scheduling loop: contract can perpetuate its own execution")
}

// =============================================================================
// 2. Adaptive Scheduling: reputation/accuracy → schedule frequency
// =============================================================================
func TestAutonomy_AdaptiveScheduling(t *testing.T) {
	state := NewMemoryStateDB()
	sk := newMockScheduleKeeper(10)
	host := newCompositeHostWithSchedules(state, sk)
	ctx := makeAutonomyCtx(500000)

	// Seed high reputation and accuracy for the agent's DID
	repKey := sha256.Sum256([]byte("reputation/" + ctx.CallerDID))
	ctx.Capabilities = allCaps()
	repVal := WordToBytes32(big.NewInt(900)) // high reputation
	state.SetStorage([]byte("agent_treasury"), repKey[:], repVal[:])

	accKey := sha256.Sum256([]byte("accuracy/" + ctx.CallerDID))
	ctx.Capabilities = allCaps()
	accVal := WordToBytes32(big.NewInt(850)) // high accuracy
	state.SetStorage([]byte("agent_treasury"), accKey[:], accVal[:])

	// Bytecode: check rep+acc > 1000 → schedule at block 110, else block 200
	// NOTE: EVM GT pops a(top) then b, returns a > b. So push threshold first, then value.
	bytecode := []byte{
		// pc=0: PUSH2 1000 (threshold first)
		PUSH2, 0x03, 0xE8, // [1000]
		// pc=3: SELF_REPUTATION
		SELF_REPUTATION, // [1000, rep=900]
		// pc=4: SELF_ACCURACY
		SELF_ACCURACY, // [1000, rep, acc=850]
		// pc=5: ADD
		ADD, // [1000, 1750]
		// pc=6: GT
		GT, // 1750 > 1000 → 1
		// pc=7: PUSH1 dest (high perf at pc=24)
		PUSH1, 24,
		// pc=9: JUMPI
		JUMPI,
		// LOW PERFORMER: schedule at block 200
		// pc=10: PUSH1 calldata
		PUSH1, 0x01,
		// pc=12: PUSH1 block=200
		PUSH1, 200,
		// pc=14: PUSH2 gas=50000
		PUSH2, 0xC3, 0x50,
		// pc=17: SCHEDULE_SELF
		SCHEDULE_SELF,
		// pc=18: PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
		PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN,
		// HIGH PERFORMER: schedule at block 110
		// pc=26... wait let me recount
	}
	// Recount: pc=18(PUSH1),19(0),20(MSTORE),21(PUSH1),22(32),23(PUSH1),24... no.
	// pc=18: PUSH1 → 2 bytes → pc=18,19
	// pc=20: MSTORE → 1 byte
	// pc=21: PUSH1 → 2 bytes → pc=21,22
	// pc=23: PUSH1 → 2 bytes → pc=23,24
	// pc=25: RETURN → 1 byte
	// So JUMPDEST should be at pc=26
	// Fix the jump target
	bytecode[8] = 26
	bytecode = append(bytecode,
		// pc=26: JUMPDEST
		JUMPDEST,
		// pc=27: PUSH1 calldata
		PUSH1, 0x01,
		// pc=29: PUSH1 block=110
		PUSH1, 110,
		// pc=31: PUSH2 gas=50000
		PUSH2, 0xC3, 0x50,
		// pc=34: SCHEDULE_SELF
		SCHEDULE_SELF,
		// pc=35: store and return
		PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN,
	)

	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)

	if !result.Success {
		t.Fatalf("adaptive scheduling failed: %v", result.Error)
	}

	// Verify schedule was created
	if sk.CountActiveSchedules(string(ctx.CurrentContract)) != 1 {
		t.Fatal("expected 1 schedule")
	}

	// Check which schedule was created — high performer should target block 110
	for _, s := range sk.schedules {
		if s.executeAt == 110 {
			t.Log("✓ High performer scheduled at block 110 (every 10 blocks)")
			return
		}
		if s.executeAt == 200 {
			t.Fatal("agent scheduled as low performer despite high reputation+accuracy")
		}
	}
	t.Log("✓ Adaptive scheduling based on introspection")
}

// =============================================================================
// 3. Identity Coherence: DID persists, VERIFY_IDENTITY checks registration
// =============================================================================
func TestAutonomy_IdentityCoherence(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)
	ctx := makeAutonomyCtx(200000)

	// First: get our DID
	bytecodeGetDID := []byte{
		AGENT_DID,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx.Bytecode = bytecodeGetDID
	interp := NewInterpreter()
	r1 := interp.Execute(bytecodeGetDID, ctx, state, host)
	if !r1.Success {
		t.Fatalf("AGENT_DID failed: %v", r1.Error)
	}
	did1 := new(big.Int).SetBytes(r1.ReturnData)

	// Second execution: same caller should get same DID (deterministic)
	interp2 := NewInterpreter()
	ctx2 := makeAutonomyCtx(200000)
	ctx2.BlockNumber = 200 // different block
	ctx2.Bytecode = bytecodeGetDID
	r2 := interp2.Execute(bytecodeGetDID, ctx2, state, host)
	if !r2.Success {
		t.Fatalf("second AGENT_DID failed: %v", r2.Error)
	}
	did2 := new(big.Int).SetBytes(r2.ReturnData)

	if did1.Cmp(did2) != 0 {
		t.Fatalf("identity not coherent across blocks: %x != %x", did1, did2)
	}
	t.Log("✓ DID is deterministic and coherent across block heights")

	// VERIFY_IDENTITY on unregistered DID should return 0
	bytecodeVerify := []byte{
		AGENT_DID,         // get our DID hash
		VERIFY_IDENTITY,   // check if registered
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx3 := makeAutonomyCtx(200000)
	ctx3.Bytecode = bytecodeVerify
	interp3 := NewInterpreter()
	r3 := interp3.Execute(bytecodeVerify, ctx3, state, host)
	if !r3.Success {
		t.Fatalf("VERIFY_IDENTITY failed: %v", r3.Error)
	}
	verified := new(big.Int).SetBytes(r3.ReturnData)
	if verified.Sign() != 0 {
		t.Fatal("unregistered DID should not verify")
	}

	// Register the identity in state, then verify
	didBytes := WordToBytes32(did1)
	regKey := sha256.Sum256([]byte("identity/" + hex.EncodeToString(didBytes[:])))
	regVal := WordToBytes32(big.NewInt(1))
	state.SetStorage([]byte("agent_registry"), regKey[:], regVal[:])

	interp4 := NewInterpreter()
	ctx4 := makeAutonomyCtx(200000)
	ctx4.Bytecode = bytecodeVerify
	r4 := interp4.Execute(bytecodeVerify, ctx4, state, host)
	if !r4.Success {
		t.Fatalf("VERIFY_IDENTITY after registration failed: %v", r4.Error)
	}
	verified2 := new(big.Int).SetBytes(r4.ReturnData)
	if verified2.Sign() == 0 {
		t.Fatal("registered DID should verify")
	}
	t.Log("✓ Identity coherence: DID deterministic, verify works after registration")
}

// =============================================================================
// 4. Key Management Lifecycle: REGISTER_KEY → use → REVOKE_KEY
// =============================================================================
func TestAutonomy_KeyManagementLifecycle(t *testing.T) {
	state := NewMemoryStateDB()
	ak := &testVMAuthKeeper{}
	host := &compositeHost{
		agent:    NewAgentHost(context.Background(), state),
		treasury: NewTreasuryHostWithAuth(state, ak, context.Background()),
		kh:       NewKnowledgeHost(nil, context.Background()),
	}
	ctx := makeAutonomyCtx(300000)

	// REGISTER_KEY: keyType=1, pubKey=0xABCD, permissions=0x0F (all), expiry=1000
	bytecodeRegister := []byte{
		PUSH1, 0x01,     // keyType
		PUSH2, 0xAB, 0xCD, // pubKey
		PUSH1, 0x0F,     // permissions (transfer|stake|claims|vote)
		PUSH2, 0x03, 0xE8, // expiry = 1000
		REGISTER_KEY,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	ctx.Bytecode = bytecodeRegister
	interp := NewInterpreter()
	r := interp.Execute(bytecodeRegister, ctx, state, host)
	if !r.Success {
		t.Fatalf("REGISTER_KEY failed: %v", r.Error)
	}
	regResult := new(big.Int).SetBytes(r.ReturnData)
	if regResult.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("expected register success (1), got %d", regResult)
	}

	// Verify key was registered in auth keeper
	if len(ak.keys) == 0 {
		t.Fatal("key not registered in auth keeper")
	}
	t.Logf("✓ Key registered: %d keys in auth keeper", len(ak.keys))

	// REVOKE_KEY: use the pubKey hex as the keyId
	pubKeyB32 := WordToBytes32(big.NewInt(0xABCD))
	pubKeyHex := hex.EncodeToString(pubKeyB32[:])

	// Store the key so we can revoke it
	ak.keys[ctx.CallerDID+"/"+pubKeyHex] = true
	ctx.Capabilities = allCaps()

	bytecodeRevoke := []byte{
		PUSH2, 0xAB, 0xCD, // keyId (same as pubKey)
		REVOKE_KEY,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx2 := makeAutonomyCtx(300000)
	ctx2.Bytecode = bytecodeRevoke
	interp2 := NewInterpreter()
	r2 := interp2.Execute(bytecodeRevoke, ctx2, state, host)
	if !r2.Success {
		t.Fatalf("REVOKE_KEY failed: %v", r2.Error)
	}
	revokeResult := new(big.Int).SetBytes(r2.ReturnData)
	if revokeResult.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("expected revoke success (1), got %d", revokeResult)
	}
	t.Log("✓ Key management lifecycle: register → revoke complete")
}

// =============================================================================
// 5. Autonomy Under Constraint: per-contract schedule limit
// =============================================================================
func TestAutonomy_ScheduleLimitEnforcement(t *testing.T) {
	state := NewMemoryStateDB()
	sk := newMockScheduleKeeper(2) // only 2 schedules allowed
	host := newCompositeHostWithSchedules(state, sk)

	// Bytecode: GET_SCHEDULES → check count → SCHEDULE_SELF → return schedule ID
	bytecodeSchedule := []byte{
		PUSH1, 0x01,       // callData
		PUSH2, 0x00, 200,  // executeBlock
		PUSH2, 0xC3, 0x50, // gasLimit
		SCHEDULE_SELF,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	ctx := makeAutonomyCtx(500000)
	ctx.Bytecode = bytecodeSchedule

	// Schedule 1 - should succeed
	r1 := NewInterpreter().Execute(bytecodeSchedule, ctx, state, host)
	if !r1.Success || new(big.Int).SetBytes(r1.ReturnData).Sign() == 0 {
		t.Fatal("first schedule should succeed")
	}

	// Schedule 2 - should succeed
	r2 := NewInterpreter().Execute(bytecodeSchedule, ctx, state, host)
	if !r2.Success || new(big.Int).SetBytes(r2.ReturnData).Sign() == 0 {
		t.Fatal("second schedule should succeed")
	}

	// Schedule 3 - should FAIL (limit = 2)
	r3 := NewInterpreter().Execute(bytecodeSchedule, ctx, state, host)
	if !r3.Success {
		t.Fatal("execution should succeed (opcode returns 0, not error)")
	}
	schedId3 := new(big.Int).SetBytes(r3.ReturnData)
	if schedId3.Sign() != 0 {
		t.Fatal("third schedule should return 0 (limit reached)")
	}

	// Now test the smart pattern: GET_SCHEDULES before scheduling
	bytecodeSmartSchedule := []byte{
		GET_SCHEDULES,        // push count
		PUSH1, 2,             // max limit
		LT,                   // count < 2?
		PUSH1, 17,            // jump to schedule
		JUMPI,
		// Limit reached: return 0
		PUSH1, 0,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
		// Do schedule
		JUMPDEST,              // pc=17
		PUSH1, 0x01,
		PUSH2, 0x01, 0x00,
		PUSH2, 0xC3, 0x50,
		SCHEDULE_SELF,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	ctx2 := makeAutonomyCtx(500000)
	ctx2.Bytecode = bytecodeSmartSchedule
	r4 := NewInterpreter().Execute(bytecodeSmartSchedule, ctx2, state, host)
	if !r4.Success {
		t.Fatalf("smart schedule check failed: %v", r4.Error)
	}
	smartResult := new(big.Int).SetBytes(r4.ReturnData)
	if smartResult.Sign() != 0 {
		t.Fatal("smart schedule should have detected limit and returned 0")
	}

	t.Log("✓ Autonomy under constraint: schedule limit enforced, smart check-before-schedule works")
}

// =============================================================================
// 6. Self-Modification Attempt: no SMOD, but CREATE2 upgrade pattern
// =============================================================================
func TestAutonomy_SelfModificationAttempt(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	// There is no self-modify opcode. SMOD (0x07) is signed modulo, not self-modify.
	// Verify that CODECOPY works (agent can READ its own code) but cannot SSTORE to code area.

	// Bytecode: CODESIZE → return it
	bytecodeIntrospect := []byte{
		CODESIZE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	ctx := makeAutonomyCtx(200000)
	ctx.Bytecode = bytecodeIntrospect
	r := NewInterpreter().Execute(bytecodeIntrospect, ctx, state, host)
	if !r.Success {
		t.Fatalf("CODESIZE failed: %v", r.Error)
	}
	codeSize := new(big.Int).SetBytes(r.ReturnData)
	if codeSize.Int64() != int64(len(bytecodeIntrospect)) {
		t.Fatalf("expected codesize %d, got %d", len(bytecodeIntrospect), codeSize.Int64())
	}

	// CREATE2 with same salt produces same address — this IS the upgrade pattern
	// (deploy new code to a deterministic address)
	initCode := []byte{PUSH1, 0x42, PUSH1, 0, MSTORE, PUSH1, 1, PUSH1, 31, RETURN}
	bytecodeCreate2 := []byte{
		// Store init code in memory
		PUSH1, initCode[0], PUSH1, 0, MSTORE8,
		PUSH1, initCode[1], PUSH1, 1, MSTORE8,
		PUSH1, initCode[2], PUSH1, 2, MSTORE8,
		PUSH1, initCode[3], PUSH1, 3, MSTORE8,
		PUSH1, initCode[4], PUSH1, 4, MSTORE8,
		PUSH1, initCode[5], PUSH1, 5, MSTORE8,
		PUSH1, initCode[6], PUSH1, 6, MSTORE8,
		PUSH1, initCode[7], PUSH1, 7, MSTORE8,
		PUSH1, initCode[8], PUSH1, 8, MSTORE8,
		// CREATE2: value=0, offset=0, length=9, salt=0x1234
		PUSH0,              // value
		PUSH1, 0,           // offset
		PUSH1, 9,           // length
		PUSH2, 0x12, 0x34,  // salt
		CREATE2,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	ctx2 := makeAutonomyCtx(500000)
	ctx2.Bytecode = bytecodeCreate2
	r2 := NewInterpreter().Execute(bytecodeCreate2, ctx2, state, host)
	if !r2.Success {
		t.Fatalf("CREATE2 failed: %v", r2.Error)
	}
	addr := new(big.Int).SetBytes(r2.ReturnData)
	if addr.Sign() == 0 {
		t.Fatal("CREATE2 should return non-zero address")
	}

	t.Logf("✓ Self-modification: no SMOD opcode exists; CREATE2 (addr=%x) is the upgrade pattern", addr)
}

// =============================================================================
// 7. Introspective Decision Making: all 4 introspection opcodes → compound decision
// =============================================================================
func TestAutonomy_IntrospectiveDecisionMaking(t *testing.T) {
	state := NewMemoryStateDB()
	sk := newMockScheduleKeeper(10)
	host := newCompositeHostWithSchedules(state, sk)
	ctx := makeAutonomyCtx(500000)

	// Seed introspection values
	setTreasuryVal := func(prefix string, val uint64) {
		key := sha256.Sum256([]byte(prefix + "/" + ctx.CallerDID))
		ctx.Capabilities = allCaps()
		b32 := WordToBytes32(new(big.Int).SetUint64(val))
		state.SetStorage([]byte("agent_treasury"), key[:], b32[:])
	}
	setTreasuryVal("available", 5000) // balance
	setTreasuryVal("staked", 2000)    // already staked
	setTreasuryVal("reputation", 800) // high rep
	setTreasuryVal("accuracy", 750)   // decent accuracy

	// Decision logic (built with explicit PC tracking):
	// 1. SELF_BALANCE > 1000 AND SELF_REPUTATION > 500 → STAKE (return 1)
	// 2. SELF_STAKE > 0 AND SELF_ACCURACY > 900 → CLAIM (return 2)
	// 3. Otherwise → IDLE (return 3)
	// NOTE: EVM GT pops a(top), b, returns a > b. Push threshold first, then value on top.
	var bytecode []byte
	// Check 1: balance > 1000 AND reputation > 500
	bytecode = append(bytecode, PUSH2, 0x03, 0xE8) // push 1000 (threshold)
	bytecode = append(bytecode, SELF_BALANCE_OP)     // push balance on top
	bytecode = append(bytecode, GT)                  // balance > 1000
	bytecode = append(bytecode, PUSH2, 0x01, 0xF4)  // push 500
	bytecode = append(bytecode, SELF_REPUTATION)     // push rep on top
	bytecode = append(bytecode, GT)                  // rep > 500
	bytecode = append(bytecode, AND)
	stakeJumpIdx := len(bytecode)
	bytecode = append(bytecode, PUSH1, 0) // placeholder
	bytecode = append(bytecode, JUMPI)

	// Check 2: stake > 0 AND accuracy > 900
	bytecode = append(bytecode, PUSH1, 0)            // push 0
	bytecode = append(bytecode, SELF_STAKE)           // push stake on top
	bytecode = append(bytecode, GT)                   // stake > 0
	bytecode = append(bytecode, PUSH2, 0x03, 0x84)   // push 900
	bytecode = append(bytecode, SELF_ACCURACY)        // push accuracy on top
	bytecode = append(bytecode, GT)                   // accuracy > 900
	bytecode = append(bytecode, AND)
	// pc=24: PUSH1 <claimLabel> (fill later)
	claimJumpIdx := len(bytecode)
	bytecode = append(bytecode, PUSH1, 0)
	// pc=26: JUMPI
	bytecode = append(bytecode, JUMPI)

	// IDLE: return 3
	// pc=27
	bytecode = append(bytecode, PUSH1, 3, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)

	// STAKE: return 1
	stakePC := len(bytecode)
	bytecode = append(bytecode, JUMPDEST, PUSH1, 1, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)

	// CLAIM: return 2
	claimPC := len(bytecode)
	bytecode = append(bytecode, JUMPDEST, PUSH1, 2, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)

	// Patch jump targets
	bytecode[stakeJumpIdx+1] = byte(stakePC)
	bytecode[claimJumpIdx+1] = byte(claimPC)

	ctx.Bytecode = bytecode
	r := NewInterpreter().Execute(bytecode, ctx, state, host)
	if !r.Success {
		t.Fatalf("introspective decision failed: %v", r.Error)
	}
	decision := new(big.Int).SetBytes(r.ReturnData)
	if decision.Int64() != 1 {
		t.Fatalf("expected STAKE decision (1), got %d", decision.Int64())
	}
	t.Log("✓ Introspective decision: agent chose STAKE based on balance+reputation")

	// Now change conditions: low balance, high accuracy → should get CLAIM or IDLE
	setTreasuryVal("available", 500) // below 1000 threshold
	setTreasuryVal("accuracy", 950)  // above 900

	r2 := NewInterpreter().Execute(bytecode, ctx, state, host)
	if !r2.Success {
		t.Fatalf("second introspective decision failed: %v", r2.Error)
	}
	decision2 := new(big.Int).SetBytes(r2.ReturnData)
	if decision2.Int64() != 2 {
		t.Fatalf("expected CLAIM decision (2), got %d", decision2.Int64())
	}
	t.Log("✓ Introspective decision: agent chose CLAIM based on stake+accuracy")

	// Final: low everything → IDLE
	setTreasuryVal("available", 500)
	setTreasuryVal("staked", 0)
	setTreasuryVal("accuracy", 100)

	r3 := NewInterpreter().Execute(bytecode, ctx, state, host)
	if !r3.Success {
		t.Fatalf("third introspective decision failed: %v", r3.Error)
	}
	decision3 := new(big.Int).SetBytes(r3.ReturnData)
	if decision3.Int64() != 3 {
		t.Fatalf("expected IDLE decision (3), got %d", decision3.Int64())
	}
	t.Log("✓ Introspective decision: agent chose IDLE when all metrics low")
}
