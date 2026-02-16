package vm

import (
	"math/big"
	"testing"
)

// --- Helper: deploy code into memory state and return the address ---

func deployCode(state *MemoryStateDB, addr []byte, code []byte) {
	state.SetCode(addr, code)
}

func makeCallCtx(gasLimit uint64, caller, contract []byte) *ExecutionContext {
	return &ExecutionContext{
		Caller:          caller,
		Origin:          caller,
		CurrentContract: contract,
		BlockNumber:     100,
		Timestamp:       1700000000,
		GasLimit:        gasLimit,
		Value:           new(big.Int),
		GasPrice:        new(big.Int),
		ChainID:         big.NewInt(2521),
		BaseFee:         new(big.Int),
	}
}

// --- SnapshotStateDB Tests ---

func TestSnapshot_ReadThrough(t *testing.T) {
	base := NewMemoryStateDB()
	base.SetStorage([]byte{1}, []byte{2}, []byte{3, 4})
	snap := NewSnapshot(base)

	got := snap.GetStorage([]byte{1}, []byte{2})
	if len(got) != 2 || got[0] != 3 || got[1] != 4 {
		t.Fatalf("expected [3,4], got %v", got)
	}
}

func TestSnapshot_WriteIsolation(t *testing.T) {
	base := NewMemoryStateDB()
	base.SetStorage([]byte{1}, []byte{2}, []byte{3, 4})
	snap := NewSnapshot(base)

	// Write to snapshot — base should be unchanged
	snap.SetStorage([]byte{1}, []byte{2}, []byte{5, 6})

	// Snapshot sees new value
	got := snap.GetStorage([]byte{1}, []byte{2})
	if len(got) != 2 || got[0] != 5 || got[1] != 6 {
		t.Fatalf("snapshot expected [5,6], got %v", got)
	}

	// Base still sees old value
	baseGot := base.GetStorage([]byte{1}, []byte{2})
	if len(baseGot) != 2 || baseGot[0] != 3 || baseGot[1] != 4 {
		t.Fatalf("base expected [3,4], got %v", baseGot)
	}
}

func TestSnapshot_Commit(t *testing.T) {
	base := NewMemoryStateDB()
	snap := NewSnapshot(base)

	snap.SetStorage([]byte{1}, []byte{2}, []byte{7, 8})
	snap.Commit()

	// Base should now have the value
	got := base.GetStorage([]byte{1}, []byte{2})
	if len(got) != 2 || got[0] != 7 || got[1] != 8 {
		t.Fatalf("after commit, base expected [7,8], got %v", got)
	}
}

func TestSnapshot_Discard(t *testing.T) {
	base := NewMemoryStateDB()
	base.SetStorage([]byte{1}, []byte{2}, []byte{3, 4})
	snap := NewSnapshot(base)

	snap.SetStorage([]byte{1}, []byte{2}, []byte{9, 9})
	// Don't commit — discard

	// Base is unchanged
	got := base.GetStorage([]byte{1}, []byte{2})
	if len(got) != 2 || got[0] != 3 || got[1] != 4 {
		t.Fatalf("after discard, base expected [3,4], got %v", got)
	}
}

func TestSnapshot_CodeReadThrough(t *testing.T) {
	base := NewMemoryStateDB()
	code := []byte{PUSH1, 42, STOP}
	base.SetCode([]byte{0xAA}, code)
	snap := NewSnapshot(base)

	got := snap.GetCode([]byte{0xAA})
	if len(got) != len(code) {
		t.Fatalf("expected code length %d, got %d", len(code), len(got))
	}
}

func TestSnapshot_CodeWrite(t *testing.T) {
	base := NewMemoryStateDB()
	snap := NewSnapshot(base)

	code := []byte{PUSH1, 1, STOP}
	snap.SetCode([]byte{0xBB}, code)

	// Visible in snapshot
	got := snap.GetCode([]byte{0xBB})
	if len(got) != len(code) {
		t.Fatalf("expected code in snapshot")
	}

	// Not in base
	baseGot := base.GetCode([]byte{0xBB})
	if baseGot != nil {
		t.Fatalf("expected nil in base before commit")
	}

	snap.Commit()

	// Now in base
	baseGot = base.GetCode([]byte{0xBB})
	if len(baseGot) != len(code) {
		t.Fatalf("expected code in base after commit")
	}
}

func TestSnapshot_NonceReadThrough(t *testing.T) {
	base := NewMemoryStateDB()
	base.SetNonce([]byte{0xCC}, 5)
	snap := NewSnapshot(base)

	got := snap.GetNonce([]byte{0xCC})
	if got != 5 {
		t.Fatalf("expected nonce 5, got %d", got)
	}
}

func TestSnapshot_NonceWrite(t *testing.T) {
	base := NewMemoryStateDB()
	base.SetNonce([]byte{0xDD}, 3)
	snap := NewSnapshot(base)

	snap.SetNonce([]byte{0xDD}, 7)

	// Snapshot sees 7
	if snap.GetNonce([]byte{0xDD}) != 7 {
		t.Fatalf("expected 7 in snapshot")
	}

	// Base still sees 3
	if base.GetNonce([]byte{0xDD}) != 3 {
		t.Fatalf("expected 3 in base")
	}

	snap.Commit()
	if base.GetNonce([]byte{0xDD}) != 7 {
		t.Fatalf("expected 7 in base after commit")
	}
}

func TestSnapshot_ExistsOverlay(t *testing.T) {
	base := NewMemoryStateDB()
	snap := NewSnapshot(base)

	if snap.Exists([]byte{0xEE}) {
		t.Fatalf("expected not exists before code set")
	}

	snap.SetCode([]byte{0xEE}, []byte{STOP})
	if !snap.Exists([]byte{0xEE}) {
		t.Fatalf("expected exists after code set")
	}
}

func TestSnapshot_NestedSnapshots(t *testing.T) {
	base := NewMemoryStateDB()
	base.SetStorage([]byte{1}, []byte{0}, []byte{10})

	snap1 := NewSnapshot(base)
	snap1.SetStorage([]byte{1}, []byte{0}, []byte{20})

	snap2 := NewSnapshot(snap1)
	snap2.SetStorage([]byte{1}, []byte{0}, []byte{30})

	// snap2 sees 30
	got := snap2.GetStorage([]byte{1}, []byte{0})
	if len(got) != 1 || got[0] != 30 {
		t.Fatalf("snap2 expected 30, got %v", got)
	}

	// Commit snap2 → snap1 sees 30
	snap2.Commit()
	got = snap1.GetStorage([]byte{1}, []byte{0})
	if len(got) != 1 || got[0] != 30 {
		t.Fatalf("snap1 after commit expected 30, got %v", got)
	}

	// Base still sees 10
	got = base.GetStorage([]byte{1}, []byte{0})
	if len(got) != 1 || got[0] != 10 {
		t.Fatalf("base expected 10, got %v", got)
	}
}

// --- CALL Tests ---

func TestCall_BasicExecution(t *testing.T) {
	state := NewMemoryStateDB()
	callerAddr := make([]byte, 20)
	callerAddr[19] = 0x01
	targetAddr := make([]byte, 20)
	targetAddr[19] = 0x02

	// Target contract: PUSH1 42, PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
	// Returns 42 as a 32-byte word
	targetCode := []byte{
		PUSH1, 42,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	deployCode(state, targetAddr, targetCode)

	// Caller bytecode: CALL(gas=50000, addr=target, value=0, argsOff=0, argsLen=0, retOff=0, retLen=32)
	// Then return the result from memory
	bytecode := []byte{
		// Push CALL args in reverse order: retLen, retOff, argsLen, argsOff, value, addr, gas
		PUSH1, 32,  // retLen
		PUSH1, 0,   // retOff
		PUSH1, 0,   // argsLen
		PUSH1, 0,   // argsOff
		PUSH1, 0,   // value
		PUSH1, 0x02, // addr (last byte = 0x02)
		PUSH2, 0xC3, 0x50, // gas = 50000
		CALL,
		// Now check success (should be 1 on stack)
		// Return memory[0:32] which has the sub-call result
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	ctx := makeCallCtx(1000000, callerAddr, callerAddr)
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 42 {
		t.Fatalf("expected 42, got %d", val.Int64())
	}
}

func TestCall_RevertRollback(t *testing.T) {
	state := NewMemoryStateDB()
	callerAddr := make([]byte, 20)
	callerAddr[19] = 0x01
	targetAddr := make([]byte, 20)
	targetAddr[19] = 0x02

	// Target contract: SSTORE(slot=0, value=99) then REVERT
	// PUSH1 99, PUSH1 0, SSTORE, PUSH1 0, PUSH1 0, REVERT
	targetCode := []byte{
		PUSH1, 99,
		PUSH1, 0, SSTORE,
		PUSH1, 0, PUSH1, 0, REVERT,
	}
	deployCode(state, targetAddr, targetCode)

	// Call the target
	bytecode := []byte{
		PUSH1, 0,    // retLen
		PUSH1, 0,    // retOff
		PUSH1, 0,    // argsLen
		PUSH1, 0,    // argsOff
		PUSH1, 0,    // value
		PUSH1, 0x02, // addr
		PUSH2, 0xC3, 0x50, // gas = 50000
		CALL,
		// CALL returns 0 (failure) — we don't care about the return value
		POP,
		STOP,
	}

	ctx := makeCallCtx(1000000, callerAddr, callerAddr)
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)

	if !result.Success {
		t.Fatalf("expected parent to succeed, got error: %v", result.Error)
	}

	// Verify state was rolled back — target's storage slot 0 should be nil
	stored := state.GetStorage(targetAddr, []byte{0})
	if stored != nil {
		t.Fatalf("expected nil storage after REVERT, got %v", stored)
	}
}

func TestCall_SuccessCommitsState(t *testing.T) {
	state := NewMemoryStateDB()
	callerAddr := make([]byte, 20)
	callerAddr[19] = 0x01
	targetAddr := make([]byte, 20)
	targetAddr[19] = 0x02

	// Target: SSTORE(slot=0, value=42) then STOP
	targetCode := []byte{
		PUSH1, 42,
		PUSH1, 0, SSTORE,
		STOP,
	}
	deployCode(state, targetAddr, targetCode)

	bytecode := []byte{
		PUSH1, 0,    // retLen
		PUSH1, 0,    // retOff
		PUSH1, 0,    // argsLen
		PUSH1, 0,    // argsOff
		PUSH1, 0,    // value
		PUSH1, 0x02, // addr
		PUSH2, 0xC3, 0x50, // gas
		CALL,
		POP,
		STOP,
	}

	ctx := makeCallCtx(1000000, callerAddr, callerAddr)
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)

	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}

	// Verify state was committed
	stored := state.GetStorage(targetAddr, make([]byte, 32))
	if stored == nil {
		t.Fatalf("expected state to be committed after successful CALL")
	}
	val := new(big.Int).SetBytes(stored)
	if val.Int64() != 42 {
		t.Fatalf("expected 42 in storage, got %d", val.Int64())
	}
}

func TestStaticCall_RejectsStateModification(t *testing.T) {
	state := NewMemoryStateDB()
	callerAddr := make([]byte, 20)
	callerAddr[19] = 0x01
	targetAddr := make([]byte, 20)
	targetAddr[19] = 0x02

	// Target tries to SSTORE — should fail in static mode
	targetCode := []byte{
		PUSH1, 42,
		PUSH1, 0, SSTORE,
		STOP,
	}
	deployCode(state, targetAddr, targetCode)

	// STATICCALL: gas, addr, argsOff, argsLen, retOff, retLen (6 args)
	bytecode := []byte{
		PUSH1, 0,    // retLen
		PUSH1, 0,    // retOff
		PUSH1, 0,    // argsLen
		PUSH1, 0,    // argsOff
		PUSH1, 0x02, // addr
		PUSH2, 0xC3, 0x50, // gas
		STATICCALL,
		// Should return 0 (failure due to state modification attempt)
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	ctx := makeCallCtx(1000000, callerAddr, callerAddr)
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)

	if !result.Success {
		t.Fatalf("parent should succeed, got: %v", result.Error)
	}

	// The return data should contain 0 (STATICCALL failed)
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 0 {
		t.Fatalf("expected 0 (STATICCALL failure), got %d", val.Int64())
	}
}

func TestStaticCall_ReadOnlySucceeds(t *testing.T) {
	state := NewMemoryStateDB()
	callerAddr := make([]byte, 20)
	callerAddr[19] = 0x01
	targetAddr := make([]byte, 20)
	targetAddr[19] = 0x02

	// Pre-populate storage
	state.SetStorage(targetAddr, make([]byte, 32), big.NewInt(77).Bytes())

	// Target: SLOAD(0), PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
	targetCode := []byte{
		PUSH1, 0, SLOAD,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	deployCode(state, targetAddr, targetCode)

	bytecode := []byte{
		PUSH1, 32,   // retLen
		PUSH1, 0,    // retOff
		PUSH1, 0,    // argsLen
		PUSH1, 0,    // argsOff
		PUSH1, 0x02, // addr
		PUSH2, 0xC3, 0x50, // gas
		STATICCALL,
		POP, // pop success flag
		// Return memory[0:32]
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	ctx := makeCallCtx(1000000, callerAddr, callerAddr)
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)

	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}

	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 77 {
		t.Fatalf("expected 77, got %d", val.Int64())
	}
}

func TestStaticCall_InheritedBySubCall(t *testing.T) {
	state := NewMemoryStateDB()
	callerAddr := make([]byte, 20)
	callerAddr[19] = 0x01
	targetAddr := make([]byte, 20)
	targetAddr[19] = 0x02

	// Target tries SSTORE — should fail because parent is in static mode
	targetCode := []byte{
		PUSH1, 42,
		PUSH1, 0, SSTORE,
		STOP,
	}
	deployCode(state, targetAddr, targetCode)

	// Use regular CALL but from a static context — static should inherit
	bytecode := []byte{
		PUSH1, 0,    // retLen
		PUSH1, 0,    // retOff
		PUSH1, 0,    // argsLen
		PUSH1, 0,    // argsOff
		PUSH1, 0,    // value
		PUSH1, 0x02, // addr
		PUSH2, 0xC3, 0x50, // gas
		CALL,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	ctx := makeCallCtx(1000000, callerAddr, callerAddr)
	ctx.Mode = ModeStaticCall // Parent is static
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)

	if !result.Success {
		t.Fatalf("parent should succeed, got: %v", result.Error)
	}

	// Sub-CALL should have failed (0) because static mode inherited
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 0 {
		t.Fatalf("expected 0 (inherited static mode failure), got %d", val.Int64())
	}
}

func TestDelegateCall_UsesCallerStorage(t *testing.T) {
	state := NewMemoryStateDB()
	callerAddr := make([]byte, 20)
	callerAddr[19] = 0x01
	targetAddr := make([]byte, 20)
	targetAddr[19] = 0x02

	// Target code: SSTORE(slot=0, value=55)
	// DELEGATECALL should store in CALLER's storage, not target's
	targetCode := []byte{
		PUSH1, 55,
		PUSH1, 0, SSTORE,
		STOP,
	}
	deployCode(state, targetAddr, targetCode)

	// DELEGATECALL: gas, addr, argsOff, argsLen, retOff, retLen (6 args)
	bytecode := []byte{
		PUSH1, 0,    // retLen
		PUSH1, 0,    // retOff
		PUSH1, 0,    // argsLen
		PUSH1, 0,    // argsOff
		PUSH1, 0x02, // addr
		PUSH2, 0xC3, 0x50, // gas
		DELEGATECALL,
		POP,
		STOP,
	}

	ctx := makeCallCtx(1000000, callerAddr, callerAddr)
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)

	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}

	// Value should be in caller's storage, not target's
	callerStored := state.GetStorage(callerAddr, make([]byte, 32))
	if callerStored == nil {
		t.Fatalf("expected value in caller's storage")
	}
	val := new(big.Int).SetBytes(callerStored)
	if val.Int64() != 55 {
		t.Fatalf("expected 55 in caller storage, got %d", val.Int64())
	}

	// Target storage should be empty
	targetStored := state.GetStorage(targetAddr, make([]byte, 32))
	if targetStored != nil {
		t.Fatalf("expected nil in target storage, got %v", targetStored)
	}
}

func TestCall_NonExistentContract(t *testing.T) {
	state := NewMemoryStateDB()
	callerAddr := make([]byte, 20)
	callerAddr[19] = 0x01
	targetAddr := make([]byte, 20)
	targetAddr[19] = 0xFF // No code deployed here

	bytecode := []byte{
		PUSH1, 0,    // retLen
		PUSH1, 0,    // retOff
		PUSH1, 0,    // argsLen
		PUSH1, 0,    // argsOff
		PUSH1, 0,    // value
		PUSH1, 0xFF, // addr (no code)
		PUSH2, 0xC3, 0x50,
		CALL,
		// CALL to non-existent contract succeeds with empty return
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	ctx := makeCallCtx(1000000, callerAddr, callerAddr)
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)

	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}

	// CALL to empty address returns 1 (success)
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected 1 (success), got %d", val.Int64())
	}
}

// --- CREATE Tests ---

func TestCreate_BasicDeployment(t *testing.T) {
	state := NewMemoryStateDB()
	callerAddr := make([]byte, 20)
	callerAddr[19] = 0x01

	// Init code: returns [PUSH1 99 STOP] as deployed bytecode
	// Init code constructs return data:
	//   PUSH3 <PUSH1, 99, STOP>  →  store 3 bytes at memory[0]
	//   MSTORE8 at positions 0,1,2
	//   RETURN(0, 3)
	// Simpler: just return STOP
	initCode := []byte{
		// Store PUSH1, 99, STOP at memory[0..2]
		PUSH1, PUSH1,
		PUSH1, 0, MSTORE8,
		PUSH1, 99,
		PUSH1, 1, MSTORE8,
		PUSH1, STOP,
		PUSH1, 2, MSTORE8,
		// Return memory[0:3]
		PUSH1, 3, PUSH1, 0, RETURN,
	}

	// Caller: stores init code in memory, then CREATE(value=0, offset=0, length=len(initCode))
	bytecode := make([]byte, 0, 100)
	// Store init code at memory offset 0
	for i, b := range initCode {
		bytecode = append(bytecode, PUSH1, b, PUSH1, byte(i), MSTORE8)
	}
	// CREATE(value=0, offset=0, length=len(initCode))
	bytecode = append(bytecode,
		PUSH1, byte(len(initCode)), // length
		PUSH1, 0,                    // offset
		PUSH1, 0,                    // value
		CREATE,
	)
	// The new address is on the stack — store it at memory[0] and return
	bytecode = append(bytecode,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	)

	ctx := makeCallCtx(10000000, callerAddr, callerAddr)
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)

	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}

	// Return value should be the new contract address (non-zero)
	newAddr := new(big.Int).SetBytes(result.ReturnData)
	if newAddr.Sign() == 0 {
		t.Fatalf("expected non-zero address from CREATE")
	}

	// Verify deployed code exists
	addrBytes := WordToBytes20(newAddr)
	deployed := state.GetCode(addrBytes)
	if deployed == nil {
		t.Fatalf("expected deployed code at new address")
	}
	if len(deployed) != 3 {
		t.Fatalf("expected 3-byte deployed code, got %d bytes", len(deployed))
	}
}

func TestCreate2_DeterministicAddress(t *testing.T) {
	callerAddr := make([]byte, 20)
	callerAddr[19] = 0x01
	initCode := []byte{PUSH1, 42, STOP}

	var salt [32]byte
	salt[31] = 1

	addr1 := Create2Address(callerAddr, salt, initCode)
	addr2 := Create2Address(callerAddr, salt, initCode)

	if len(addr1) != 20 || len(addr2) != 20 {
		t.Fatalf("expected 20-byte addresses")
	}

	for i := range addr1 {
		if addr1[i] != addr2[i] {
			t.Fatalf("CREATE2 addresses not deterministic")
		}
	}

	// Different salt → different address
	var salt2 [32]byte
	salt2[31] = 2
	addr3 := Create2Address(callerAddr, salt2, initCode)
	same := true
	for i := range addr1 {
		if addr1[i] != addr3[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatalf("different salts should produce different addresses")
	}
}

func TestCreateAddress_Deterministic(t *testing.T) {
	sender := make([]byte, 20)
	sender[19] = 0x42

	addr1 := CreateAddress(sender, 0)
	addr2 := CreateAddress(sender, 0)
	addr3 := CreateAddress(sender, 1)

	if len(addr1) != 20 {
		t.Fatalf("expected 20-byte address")
	}

	for i := range addr1 {
		if addr1[i] != addr2[i] {
			t.Fatalf("same inputs should produce same address")
		}
	}

	same := true
	for i := range addr1 {
		if addr1[i] != addr3[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatalf("different nonces should produce different addresses")
	}
}

// --- Gas Tests ---

func TestGasMeter_Refund(t *testing.T) {
	g := NewGasMeter(1000)
	g.Consume(600)
	if g.Remaining() != 400 {
		t.Fatalf("expected 400 remaining, got %d", g.Remaining())
	}

	g.Refund(200)
	if g.Remaining() != 600 {
		t.Fatalf("expected 600 remaining after refund, got %d", g.Remaining())
	}
}

func TestGasMeter_RefundOverflow(t *testing.T) {
	g := NewGasMeter(1000)
	g.Consume(100)

	// Refund more than consumed — should clamp to 0
	g.Refund(500)
	if g.Used() != 0 {
		t.Fatalf("expected 0 used after over-refund, got %d", g.Used())
	}
}

func TestCalculateForwardGas(t *testing.T) {
	// 63/64 rule: max forward = available - available/64
	available := uint64(6400)
	maxForward := available - available/64 // 6400 - 100 = 6300

	got := CalculateForwardGas(available, 0) // 0 means "all"
	if got != maxForward {
		t.Fatalf("expected %d, got %d", maxForward, got)
	}

	// Explicit request less than max
	got = CalculateForwardGas(available, 1000)
	if got != 1000 {
		t.Fatalf("expected 1000, got %d", got)
	}

	// Explicit request more than max → clamped
	got = CalculateForwardGas(available, 100000)
	if got != maxForward {
		t.Fatalf("expected %d, got %d", maxForward, got)
	}
}

// --- WordToBytes20 ---

func TestWordToBytes20(t *testing.T) {
	addr := big.NewInt(0xDEADBEEF)
	b := WordToBytes20(addr)
	if len(b) != 20 {
		t.Fatalf("expected 20 bytes, got %d", len(b))
	}
	// Last 4 bytes should be DEADBEEF
	if b[16] != 0xDE || b[17] != 0xAD || b[18] != 0xBE || b[19] != 0xEF {
		t.Fatalf("unexpected bytes: %x", b)
	}
}

// --- ExecuteCall Direct ---

func TestExecuteCall_CallDepthExceeded(t *testing.T) {
	state := NewMemoryStateDB()
	ctx := makeCallCtx(100000, make([]byte, 20), make([]byte, 20))

	result := ExecuteCall(ModeCall, ctx, make([]byte, 20), nil, 50000, new(big.Int), state, nil, MaxCallDepth)
	if result.Success {
		t.Fatalf("expected failure at max call depth")
	}
	if result.Error.Code != ErrCodeCallDepthExceeded {
		t.Fatalf("expected call depth exceeded error, got: %v", result.Error)
	}
}

func TestExecuteCall_EmptyContract(t *testing.T) {
	state := NewMemoryStateDB()
	ctx := makeCallCtx(100000, make([]byte, 20), make([]byte, 20))

	// Calling an address with no code should succeed with empty return
	result := ExecuteCall(ModeCall, ctx, []byte{0xFF}, nil, 50000, new(big.Int), state, nil, 0)
	if !result.Success {
		t.Fatalf("expected success for empty contract, got: %v", result.Error)
	}
	if result.ReturnData != nil {
		t.Fatalf("expected nil return data, got: %v", result.ReturnData)
	}
}
