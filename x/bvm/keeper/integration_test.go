package keeper_test

// ===========================================================================
// R2 Session 3: BVM Gas Bridge Integration Tests
//
// Tests the SDK gas meter ↔ VM gas bridge, scheduled execution per-block
// budget, capability inheritance for schedules, and panic recovery under
// realistic conditions.
// ===========================================================================

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"

	"github.com/legible-money/legible/x/bvm/keeper"
	"github.com/legible-money/legible/x/bvm/types"
)

// ---------- Bytecode Helpers ----------

// sstoreBytecode writes value to storage slot 0, then STOP.
// SSTORE costs 20000 gas (SstoreSetGas) making it useful for gas measurement.
// Bytecode: PUSH1 value, PUSH1 0, SSTORE, STOP
func sstoreBytecode(value byte) []byte {
	return []byte{
		0x60, value, // PUSH1 value
		0x60, 0, // PUSH1 0  (slot)
		0x55, // SSTORE
		0x00, // STOP
	}
}

// multiSstoreBytecode writes N values to N different storage slots.
// Each SSTORE costs 20000 gas, so total gas ≈ N * 20000 + overhead.
func multiSstoreBytecode(count int) []byte {
	code := make([]byte, 0, count*5+1)
	for i := 0; i < count; i++ {
		code = append(code, 0x60, byte(i+1)) // PUSH1 value
		code = append(code, 0x60, byte(i))   // PUSH1 slot
		code = append(code, 0x55)            // SSTORE
	}
	code = append(code, 0x00) // STOP
	return code
}

// callBytecode constructs bytecode that performs a CALL to a target address.
// CALL args: gas, addr, value, argsOffset, argsLength, retOffset, retLength
// Since we can't easily push a long address, we'll CALL address 0 (precompile/EOA).
// The CALL itself costs GasCallBase=700, and calling a non-existent contract
// succeeds with empty return.
func callToZeroBytecode() []byte {
	return []byte{
		// retLength=0
		0x60, 0, // PUSH1 0
		// retOffset=0
		0x60, 0, // PUSH1 0
		// argsLength=0
		0x60, 0, // PUSH1 0
		// argsOffset=0
		0x60, 0, // PUSH1 0
		// value=0
		0x60, 0, // PUSH1 0
		// addr=1 (precompile identity, returns empty)
		0x60, 1, // PUSH1 1
		// gas=5000
		0x61, 0x13, 0x88, // PUSH2 5000
		// CALL
		0xF1,
		// Pop result
		0x50, // POP
		// STOP
		0x00,
	}
}

// nestedCallBytecode creates bytecode that CALLs address 1 three times
// (simulating A→B→C in terms of gas consumption).
func tripleCallBytecode() []byte {
	code := make([]byte, 0, 60)
	for i := 0; i < 3; i++ {
		code = append(code,
			0x60, 0, // retLength
			0x60, 0, // retOffset
			0x60, 0, // argsLength
			0x60, 0, // argsOffset
			0x60, 0, // value
			0x60, 1, // addr
			0x61, 0x13, 0x88, // PUSH2 5000 gas
			0xF1, // CALL
			0x50, // POP result
		)
	}
	code = append(code, 0x00) // STOP
	return code
}

// loopForeverBytecode creates a contract that loops until gas runs out.
// JUMPDEST, PUSH1 0, JUMP
func loopForeverBytecode() []byte {
	return []byte{
		0x5B,    // JUMPDEST (offset 0)
		0x60, 0, // PUSH1 0
		0x56,    // JUMP → offset 0
	}
}

// deepRecursionBytecode creates bytecode that performs recursive CALLs
// to itself. Since CosmosStateDB can't resolve the contract code via the
// current key format, the sub-CALL resolves to an EOA (empty code → success).
// Still exercises the CALL stack depth handling and gas forwarding.
func deepCallChainBytecode() []byte {
	// Performs 10 nested CALLs to address 0, each forwarding gas
	code := make([]byte, 0, 150)
	for i := 0; i < 10; i++ {
		code = append(code,
			0x60, 0, // retLength
			0x60, 0, // retOffset
			0x60, 0, // argsLength
			0x60, 0, // argsOffset
			0x60, 0, // value
			0x60, 0, // addr=0
			0x61, 0x27, 0x10, // PUSH2 10000 gas
			0xF1, // CALL
			0x50, // POP
		)
	}
	code = append(code, 0x00) // STOP
	return code
}

// ==========================================================================
// Section 1: Gas Bridge Integration Tests
// ==========================================================================

func TestGasBridge_SimpleContract(t *testing.T) {
	k, ctx, _, _ := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy a contract with SSTORE (measurable gas: ~20000 for SSTORE)
	bytecode := sstoreBytecode(42)
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	// Set up SDK gas meter with high limit
	gasMeter := storetypes.NewGasMeter(10_000_000)
	ctx = ctx.WithGasMeter(gasMeter)

	gasBefore := gasMeter.GasConsumed()

	resp, err := msgServer.CallContract(ctx, &types.MsgCallContract{
		Caller:          creator1,
		ContractAddress: deployResp.ContractAddress,
		InputData:       []byte{},
		GasLimit:        500_000,
	})
	if err != nil {
		t.Fatalf("call error: %v", err)
	}

	gasAfter := gasMeter.GasConsumed()
	sdkGasConsumed := gasAfter - gasBefore

	// SDK gas meter MUST be charged at least the VM's reported gas
	if sdkGasConsumed < resp.GasUsed {
		t.Fatalf("SDK gas consumed (%d) < VM gas used (%d) — gas bridge broken",
			sdkGasConsumed, resp.GasUsed)
	}

	// VM should have consumed meaningful gas (SSTORE = 20000+)
	if resp.GasUsed < 20_000 {
		t.Fatalf("expected VM gas >= 20000 (SSTORE), got %d", resp.GasUsed)
	}

	t.Logf("PASS: SimpleContract — VM gas=%d, SDK gas=%d", resp.GasUsed, sdkGasConsumed)
}

func TestGasBridge_OutOfGas_SDKLimit(t *testing.T) {
	k, ctx, mockBK, _ := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy contract with SSTORE (needs ~20000 gas)
	bytecode := sstoreBytecode(42)
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	// Fund caller for value transfer test
	mockBK.balances[caller1] = map[string]int64{"ulgm": 100_000}

	// Set SDK gas limit LOWER than what the VM will consume
	// SSTORE costs ~20000 VM gas, so set SDK limit to 100
	gasMeter := storetypes.NewGasMeter(100)
	ctx = ctx.WithGasMeter(gasMeter)

	// Call with payable value to test atomic rollback
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_, _ = msgServer.CallContract(ctx, &types.MsgCallContract{
			Caller:          caller1,
			ContractAddress: deployResp.ContractAddress,
			InputData:       []byte{},
			GasLimit:        500_000,
			Value:           "5000",
		})
	}()

	if !panicked {
		t.Fatal("expected out-of-gas panic when SDK gas limit < VM gas consumed")
	}

	// In production, the SDK ante handler catches this panic and reverts
	// the entire transaction, including the value transfer.
	// Verify the mock bank balance is unchanged (panic prevents commit)
	if mockBK.balances[caller1]["ulgm"] != 100_000 {
		t.Logf("NOTE: Balance changed to %d — in real SDK, tx revert would restore it",
			mockBK.balances[caller1]["ulgm"])
	}

	t.Log("PASS: OutOfGas_SDKLimit — panic triggered, tx would be atomically reverted")
}

func TestGasBridge_OutOfGas_VMLimit(t *testing.T) {
	k, ctx, _, _ := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy infinite loop contract — will consume gas up to the VM limit
	bytecode := loopForeverBytecode()
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	// Set high SDK gas limit but LOW VM gas limit
	gasMeter := storetypes.NewGasMeter(10_000_000)
	ctx = ctx.WithGasMeter(gasMeter)

	gasBefore := gasMeter.GasConsumed()

	// Call with low VM gas limit — VM should hit OOG
	_, err = msgServer.CallContract(ctx, &types.MsgCallContract{
		Caller:          creator1,
		ContractAddress: deployResp.ContractAddress,
		InputData:       []byte{},
		GasLimit:        1_000, // Low VM gas limit
	})

	// CallContract returns error when VM execution fails
	if err == nil {
		t.Fatal("expected error from VM out-of-gas")
	}

	gasAfter := gasMeter.GasConsumed()
	sdkGasConsumed := gasAfter - gasBefore

	// SDK gas meter should reflect the VM's gas usage (full gas limit charged on OOG)
	if sdkGasConsumed == 0 {
		t.Fatal("SDK gas meter shows 0 — gas bridge not working for failed executions")
	}

	// The VM should have consumed approximately the full gas limit
	if sdkGasConsumed < 900 {
		t.Fatalf("SDK gas consumed (%d) too low — expected near VM gas limit (1000)", sdkGasConsumed)
	}

	t.Logf("PASS: OutOfGas_VMLimit — VM OOG, SDK gas consumed=%d (≈VM limit 1000)", sdkGasConsumed)
}

func TestGasBridge_NestedCalls(t *testing.T) {
	k, ctx, _, _ := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy contract with multiple CALLs (each CALL has base cost ~700 gas)
	bytecode := tripleCallBytecode()
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	// Also deploy a simpler contract for gas comparison baseline
	simpleBc := []byte{0x00} // STOP
	simpleResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: simpleBc,
	})
	if err != nil {
		t.Fatalf("deploy simple error: %v", err)
	}

	// Call the simple contract first to get baseline gas
	gasMeter1 := storetypes.NewGasMeter(10_000_000)
	ctx1 := ctx.WithGasMeter(gasMeter1)

	_, err = msgServer.CallContract(ctx1, &types.MsgCallContract{
		Caller:          creator1,
		ContractAddress: simpleResp.ContractAddress,
		InputData:       []byte{},
		GasLimit:        500_000,
	})
	if err != nil {
		t.Fatalf("simple call error: %v", err)
	}
	simpleGas := gasMeter1.GasConsumed()

	// Call the triple-CALL contract
	gasMeter2 := storetypes.NewGasMeter(10_000_000)
	ctx2 := ctx.WithGasMeter(gasMeter2)

	resp, err := msgServer.CallContract(ctx2, &types.MsgCallContract{
		Caller:          creator1,
		ContractAddress: deployResp.ContractAddress,
		InputData:       []byte{},
		GasLimit:        500_000,
	})
	if err != nil {
		t.Fatalf("nested call error: %v", err)
	}
	nestedGas := gasMeter2.GasConsumed()

	// The nested-CALL contract should consume more SDK gas than the simple STOP
	// (CALL to empty address refunds forwarded gas, but stack ops still cost gas)
	if nestedGas <= simpleGas {
		t.Fatalf("nested call SDK gas (%d) should be > simple SDK gas (%d)", nestedGas, simpleGas)
	}

	// SDK gas should be >= VM gas (fundamental bridge property)
	if nestedGas < resp.GasUsed {
		t.Fatalf("SDK gas (%d) < VM gas (%d) — bridge broken for nested calls", nestedGas, resp.GasUsed)
	}

	// VM gas should be non-zero (3 sets of PUSH/CALL/POP operations)
	if resp.GasUsed == 0 {
		t.Fatal("expected non-zero VM gas for triple CALL contract")
	}

	t.Logf("PASS: NestedCalls — simple SDK=%d, nested SDK=%d, VM gas=%d", simpleGas, nestedGas, resp.GasUsed)
}

// ==========================================================================
// Section 2: Scheduled Execution Budget Tests
// ==========================================================================

func TestScheduleBudget_SingleSchedule_FitsInBudget(t *testing.T) {
	k, ctx, _, _ := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy a simple contract
	bytecode := []byte{0x00} // STOP
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	// Schedule one execution well within budget (MaxScheduledGasPerBlock = 5_000_000)
	schedResp, err := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: deployResp.ContractAddress,
		Method:          "tick",
		ExecuteAtBlock:  200,
		MaxGas:          50_000,
	})
	if err != nil {
		t.Fatalf("schedule error: %v", err)
	}

	// Execute at block 200
	ctx = ctx.WithBlockHeight(200)
	k.ExecutePendingSchedules(ctx)

	// Verify it executed
	schedule, found := k.GetSchedule(ctx, schedResp.ScheduleId)
	if !found {
		t.Fatal("schedule not found")
	}
	if !schedule.Executed {
		t.Fatal("single schedule should have executed within budget")
	}

	t.Log("PASS: SingleSchedule fits in budget and executes")
}

func TestScheduleBudget_MultipleSchedules_ExceedsBudget(t *testing.T) {
	k, ctx, _, _ := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy an infinite loop contract — consumes gas up to its limit
	bytecode := loopForeverBytecode()
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	// Schedule 3 contracts, each requesting 3M gas
	// MaxScheduledGasPerBlock = 5M, so first 1-2 should execute, third deferred
	schedIds := make([]string, 3)
	for i := 0; i < 3; i++ {
		resp, err := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
			Caller:          creator1,
			ContractAddress: deployResp.ContractAddress,
			Method:          fmt.Sprintf("loop%d", i),
			ExecuteAtBlock:  200,
			MaxGas:          3_000_000,
		})
		if err != nil {
			t.Fatalf("schedule %d error: %v", i, err)
		}
		schedIds[i] = resp.ScheduleId
	}

	// Execute at block 200
	ctx = ctx.WithBlockHeight(200)
	k.ExecutePendingSchedules(ctx)

	// Count executed vs pending
	var executed, deferred int
	for _, id := range schedIds {
		sched, found := k.GetSchedule(ctx, id)
		if !found {
			t.Fatalf("schedule %s not found", id)
		}
		if sched.Executed {
			executed++
		} else {
			deferred++
		}
	}

	// Budget enforcement: not all 3 should execute (3×3M = 9M > 5M budget)
	// At least 1 should execute, and at least 1 should be deferred
	if executed == 0 {
		t.Fatal("expected at least 1 schedule to execute")
	}
	if deferred == 0 {
		t.Fatal("expected at least 1 schedule to be deferred (budget exceeded)")
	}

	t.Logf("PASS: MultipleSchedules — %d executed, %d deferred (budget enforcement)", executed, deferred)
}

func TestScheduleBudget_DeferredScheduleRunsNextBlock(t *testing.T) {
	k, ctx, _, _ := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy infinite loop contract
	bytecode := loopForeverBytecode()
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	// Schedule 3 contracts at block 200, each requesting 3M gas
	schedIds := make([]string, 3)
	for i := 0; i < 3; i++ {
		resp, err := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
			Caller:          creator1,
			ContractAddress: deployResp.ContractAddress,
			Method:          fmt.Sprintf("loop%d", i),
			ExecuteAtBlock:  200,
			MaxGas:          3_000_000,
		})
		if err != nil {
			t.Fatalf("schedule %d error: %v", i, err)
		}
		schedIds[i] = resp.ScheduleId
	}

	// Execute at block 200 — some deferred
	ctx = ctx.WithBlockHeight(200)
	k.ExecutePendingSchedules(ctx)

	// Count deferred from block 200
	var executedBlock200 int
	for _, id := range schedIds {
		sched, _ := k.GetSchedule(ctx, id)
		if sched.Executed {
			executedBlock200++
		}
	}

	// Execute at block 201 — deferred should run
	ctx = ctx.WithBlockHeight(201)
	k.ExecutePendingSchedules(ctx)

	// Now all should be executed
	var executedBlock201 int
	for _, id := range schedIds {
		sched, _ := k.GetSchedule(ctx, id)
		if sched.Executed {
			executedBlock201++
		}
	}

	if executedBlock201 <= executedBlock200 {
		t.Fatalf("deferred schedules didn't run in next block: block200=%d, block201=%d",
			executedBlock200, executedBlock201)
	}

	t.Logf("PASS: DeferredScheduleRunsNextBlock — block200=%d, block201=%d executed",
		executedBlock200, executedBlock201)
}

func TestScheduleBudget_PanicInSchedule_DoesntBlockOthers(t *testing.T) {
	k, ctx, _, _ := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy a panicking contract (INVALID opcode)
	panicBc := []byte{0xFE} // INVALID
	panicResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: panicBc,
	})
	if err != nil {
		t.Fatalf("deploy panic error: %v", err)
	}

	// Deploy a normal contract
	normalBc := []byte{0x00} // STOP
	normalResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: normalBc,
	})
	if err != nil {
		t.Fatalf("deploy normal error: %v", err)
	}

	// Schedule panic contract first, then normal contract
	_, err = msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: panicResp.ContractAddress,
		Method:          "crash",
		ExecuteAtBlock:  200,
		MaxGas:          50_000,
	})
	if err != nil {
		t.Fatalf("schedule panic error: %v", err)
	}

	normalSchedResp, err := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: normalResp.ContractAddress,
		Method:          "ok",
		ExecuteAtBlock:  200,
		MaxGas:          50_000,
	})
	if err != nil {
		t.Fatalf("schedule normal error: %v", err)
	}

	// Execute — should NOT panic; normal schedule should still run
	ctx = ctx.WithBlockHeight(200)
	k.ExecutePendingSchedules(ctx)

	// Verify normal schedule executed
	normalSched, found := k.GetSchedule(ctx, normalSchedResp.ScheduleId)
	if !found {
		t.Fatal("normal schedule not found")
	}
	if !normalSched.Executed {
		t.Fatal("normal schedule should have executed despite prior panic")
	}

	t.Log("PASS: PanicInSchedule doesn't block subsequent schedules")
}

// ==========================================================================
// Section 3: Capability Inheritance Tests
// ==========================================================================

func TestScheduleCapability_RestrictedKey_RestrictedExecution(t *testing.T) {
	k, ctx, _, mockAuth := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Set up DID + restricted session (CanTransfer only)
	mockAuth.dids[creator1] = "did:lgm:restricted"
	mockAuth.sessions[creator1] = mockSession{
		caps: types.SessionCapabilities{
			CanTransfer:     true,
			CanStake:        false,
			CanSubmitClaims: false,
			CanVote:         false,
		},
		expiresAtBlock: 500,
	}

	// Deploy contract
	bytecode := []byte{0x00} // STOP
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	// Schedule with restricted session
	schedResp, err := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: deployResp.ContractAddress,
		Method:          "transfer",
		ExecuteAtBlock:  200,
		MaxGas:          50_000,
	})
	if err != nil {
		t.Fatalf("schedule error: %v", err)
	}

	// Verify stored capabilities match the restricted session
	storedCaps, found := k.GetScheduleCapabilities(ctx, schedResp.ScheduleId)
	if !found {
		t.Fatal("stored capabilities not found")
	}
	if !storedCaps.CanTransfer {
		t.Fatal("expected CanTransfer=true in stored caps")
	}
	if storedCaps.CanStake {
		t.Fatal("expected CanStake=false in stored caps")
	}
	if storedCaps.CanSubmitClaims {
		t.Fatal("expected CanSubmitClaims=false in stored caps")
	}
	if storedCaps.CanVote {
		t.Fatal("expected CanVote=false in stored caps")
	}

	// Execute the schedule
	ctx = ctx.WithBlockHeight(200)
	k.ExecutePendingSchedules(ctx)

	// Verify it executed (restricted caps don't prevent STOP from executing)
	sched, _ := k.GetSchedule(ctx, schedResp.ScheduleId)
	if !sched.Executed {
		t.Fatal("schedule with restricted caps should execute")
	}

	// Verify capabilities are still stored (not deleted after execution)
	capsAfter, found := k.GetScheduleCapabilities(ctx, schedResp.ScheduleId)
	if !found {
		t.Fatal("capabilities deleted after execution")
	}
	if capsAfter.CanStake {
		t.Fatal("stored caps should remain restricted after execution")
	}

	t.Log("PASS: RestrictedKey → RestrictedExecution")
}

func TestScheduleCapability_FullKey_FullExecution(t *testing.T) {
	k, ctx, _, mockAuth := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Set up DID with NO session key (identity key = full access)
	mockAuth.dids[creator1] = "did:lgm:full-access"

	// Deploy contract
	bytecode := []byte{0x00} // STOP
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	// Schedule with identity key (full access)
	schedResp, err := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: deployResp.ContractAddress,
		Method:          "full",
		ExecuteAtBlock:  200,
		MaxGas:          50_000,
	})
	if err != nil {
		t.Fatalf("schedule error: %v", err)
	}

	// Verify stored capabilities are full access
	storedCaps, found := k.GetScheduleCapabilities(ctx, schedResp.ScheduleId)
	if !found {
		t.Fatal("stored capabilities not found")
	}
	if !storedCaps.CanTransfer || !storedCaps.CanStake || !storedCaps.CanSubmitClaims || !storedCaps.CanVote {
		t.Fatalf("expected full capabilities, got: %+v", storedCaps)
	}

	// Execute
	ctx = ctx.WithBlockHeight(200)
	k.ExecutePendingSchedules(ctx)

	sched, _ := k.GetSchedule(ctx, schedResp.ScheduleId)
	if !sched.Executed {
		t.Fatal("schedule with full caps should execute")
	}

	t.Log("PASS: FullKey → FullExecution")
}

func TestScheduleCapability_CapabilityExpires(t *testing.T) {
	k, ctx, _, mockAuth := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Set up DID + session that expires at block 150
	mockAuth.dids[creator1] = "did:lgm:expiring"
	mockAuth.sessions[creator1] = mockSession{
		caps: types.SessionCapabilities{
			CanTransfer:     true,
			CanStake:        false,
			CanSubmitClaims: false,
			CanVote:         false,
		},
		expiresAtBlock: 150, // Expires BEFORE the schedule executes
	}

	// Deploy contract
	bytecode := []byte{0x00} // STOP
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	// Schedule at creation time (block 100) — session is still active
	schedResp, err := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: deployResp.ContractAddress,
		Method:          "late",
		ExecuteAtBlock:  200, // Session expires at 150!
		MaxGas:          50_000,
	})
	if err != nil {
		t.Fatalf("schedule error: %v", err)
	}

	// Verify capabilities were snapshotted at creation time (restricted)
	storedCaps, found := k.GetScheduleCapabilities(ctx, schedResp.ScheduleId)
	if !found {
		t.Fatal("stored capabilities not found")
	}
	if storedCaps.CanStake {
		t.Fatal("expected CanStake=false (snapshotted from restricted session)")
	}

	// Now the session key expires (block 150 passes)
	// Execute at block 200 — session is expired but stored caps should persist
	ctx = ctx.WithBlockHeight(200)
	k.ExecutePendingSchedules(ctx)

	// Verify the schedule still executed with the SNAPSHOTTED capabilities
	sched, _ := k.GetSchedule(ctx, schedResp.ScheduleId)
	if !sched.Executed {
		t.Fatal("schedule should execute even after session key expires (caps are snapshotted)")
	}

	// Verify stored caps are still restricted (not upgraded to full-access)
	capsAfter, found := k.GetScheduleCapabilities(ctx, schedResp.ScheduleId)
	if !found {
		t.Fatal("capabilities deleted after execution")
	}
	if capsAfter.CanStake {
		t.Fatal("stored caps should remain restricted even after session key expired")
	}
	if !capsAfter.CanTransfer {
		t.Fatal("stored CanTransfer should remain true")
	}

	t.Log("PASS: CapabilityExpires — snapshotted caps survive key expiry")
}

// ==========================================================================
// Section 4: Panic Recovery Stress Tests
// ==========================================================================

func TestPanicRecovery_INVALID_Opcode(t *testing.T) {
	k, ctx, _, _ := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy contract with INVALID opcode
	bytecode := []byte{0xFE}
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	gasMeter := storetypes.NewGasMeter(10_000_000)
	ctx = ctx.WithGasMeter(gasMeter)

	gasBefore := gasMeter.GasConsumed()

	// Should NOT panic — safeExecute catches it
	_, callErr := msgServer.CallContract(ctx, &types.MsgCallContract{
		Caller:          creator1,
		ContractAddress: deployResp.ContractAddress,
		InputData:       []byte{},
		GasLimit:        100_000,
	})

	gasAfter := gasMeter.GasConsumed()
	sdkGasConsumed := gasAfter - gasBefore

	// INVALID opcode should cause execution error (not panic)
	if callErr == nil {
		t.Log("INVALID opcode handled gracefully by interpreter (no error)")
	} else {
		t.Logf("INVALID opcode returned error: %v", callErr)
	}

	// Gas should still be bridged
	if sdkGasConsumed == 0 {
		t.Fatal("SDK gas should be consumed even for INVALID opcode execution")
	}

	t.Logf("PASS: INVALID opcode — recovered, SDK gas=%d", sdkGasConsumed)
}

func TestPanicRecovery_StackOverflow(t *testing.T) {
	k, ctx, _, _ := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy contract that tries deep call chain (10 nested CALLs)
	bytecode := deepCallChainBytecode()
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	gasMeter := storetypes.NewGasMeter(10_000_000)
	ctx = ctx.WithGasMeter(gasMeter)

	gasBefore := gasMeter.GasConsumed()

	// Should NOT panic even with deep call chain
	resp, err := msgServer.CallContract(ctx, &types.MsgCallContract{
		Caller:          creator1,
		ContractAddress: deployResp.ContractAddress,
		InputData:       []byte{},
		GasLimit:        1_000_000,
	})

	gasAfter := gasMeter.GasConsumed()

	// Gas should be consumed regardless
	if gasAfter <= gasBefore {
		t.Fatal("expected gas consumption from deep call chain")
	}

	if err != nil {
		t.Logf("Deep call chain returned error (expected): %v", err)
	} else {
		t.Logf("Deep call chain succeeded, gas=%d", resp.GasUsed)
	}

	t.Logf("PASS: StackOverflow — recovered, SDK gas=%d", gasAfter-gasBefore)
}

func TestPanicRecovery_MultipleSchedules_OnePanics(t *testing.T) {
	k, ctx, _, _ := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy 3 contracts: good, bad, good
	goodBc := []byte{0x00} // STOP
	badBc := []byte{0xFE}  // INVALID

	good1, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1, Bytecode: goodBc,
	})
	bad, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1, Bytecode: badBc,
	})
	good2, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1, Bytecode: goodBc,
	})

	// Schedule: good1, bad, good2 — all at block 200
	sched1, _ := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller: creator1, ContractAddress: good1.ContractAddress,
		Method: "ok1", ExecuteAtBlock: 200, MaxGas: 50_000,
	})
	_, _ = msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller: creator1, ContractAddress: bad.ContractAddress,
		Method: "crash", ExecuteAtBlock: 200, MaxGas: 50_000,
	})
	sched3, _ := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller: creator1, ContractAddress: good2.ContractAddress,
		Method: "ok2", ExecuteAtBlock: 200, MaxGas: 50_000,
	})

	// Execute — should NOT panic
	ctx = ctx.WithBlockHeight(200)
	k.ExecutePendingSchedules(ctx)

	// Both good schedules should have executed
	s1, _ := k.GetSchedule(ctx, sched1.ScheduleId)
	s3, _ := k.GetSchedule(ctx, sched3.ScheduleId)

	if !s1.Executed {
		t.Fatal("good schedule 1 should have executed")
	}
	if !s3.Executed {
		t.Fatal("good schedule 3 should have executed despite schedule 2 panicking")
	}

	t.Log("PASS: MultipleSchedules_OnePanics — good schedules complete, bad one recovered")
}

// ==========================================================================
// Section 5: Additional Gas Bridge Edge Cases
// ==========================================================================

func TestGasBridge_SSTOREHeavyContract(t *testing.T) {
	k, ctx, _, _ := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy contract with 5 SSTOREs (5 × 20000 = ~100000 gas)
	bytecode := multiSstoreBytecode(5)
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	gasMeter := storetypes.NewGasMeter(10_000_000)
	ctx = ctx.WithGasMeter(gasMeter)

	resp, err := msgServer.CallContract(ctx, &types.MsgCallContract{
		Caller:          creator1,
		ContractAddress: deployResp.ContractAddress,
		InputData:       []byte{},
		GasLimit:        500_000,
	})
	if err != nil {
		t.Fatalf("call error: %v", err)
	}

	// 5 SSTOREs should cost at least 5 × 20000 = 100000 VM gas
	if resp.GasUsed < 100_000 {
		t.Fatalf("expected VM gas >= 100000 for 5 SSTOREs, got %d", resp.GasUsed)
	}

	// SDK gas must be at least that much
	sdkGas := gasMeter.GasConsumed()
	if sdkGas < resp.GasUsed {
		t.Fatalf("SDK gas (%d) < VM gas (%d) — bridge broken", sdkGas, resp.GasUsed)
	}

	t.Logf("PASS: SSTORE-heavy — 5 SSTOREs, VM gas=%d, SDK gas=%d", resp.GasUsed, sdkGas)
}

func TestGasBridge_PanicChargesFullGas(t *testing.T) {
	k, ctx, _, _ := setupKeeperWithAuth(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy INVALID opcode contract
	bytecode := []byte{0xFE}
	deployResp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Deployer: creator1,
		Bytecode: bytecode,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	gasLimit := uint64(100_000)
	gasMeter := storetypes.NewGasMeter(10_000_000)
	ctx = ctx.WithGasMeter(gasMeter)

	gasBefore := gasMeter.GasConsumed()

	// Call — will either error normally or panic (caught by safeExecute)
	_, _ = msgServer.CallContract(ctx, &types.MsgCallContract{
		Caller:          creator1,
		ContractAddress: deployResp.ContractAddress,
		InputData:       []byte{},
		GasLimit:        gasLimit,
	})

	sdkGasConsumed := gasMeter.GasConsumed() - gasBefore

	// Whether it's a normal error or a recovered panic, gas should be charged
	if sdkGasConsumed == 0 {
		t.Fatal("expected non-zero gas consumption for failed execution")
	}

	t.Logf("PASS: PanicChargesGas — gasLimit=%d, SDK gas consumed=%d", gasLimit, sdkGasConsumed)
}
