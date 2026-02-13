package vm

import (
	"math/big"
	"testing"
)

func makeCtx(gasLimit uint64) *ExecutionContext {
	return &ExecutionContext{
		Caller:          make([]byte, 20),
		Origin:          make([]byte, 20),
		CurrentContract: make([]byte, 20),
		BlockNumber:     100,
		Timestamp:       1700000000,
		GasLimit:        gasLimit,
		Value:           new(big.Int),
		GasPrice:        new(big.Int),
		ChainID:         big.NewInt(2521),
		BaseFee:         new(big.Int),
	}
}

func TestInterp_AddSimple(t *testing.T) {
	// PUSH1 3 PUSH1 5 ADD PUSH1 0 MSTORE PUSH1 32 PUSH1 0 RETURN
	bytecode := []byte{
		PUSH1, 3,
		PUSH1, 5,
		ADD,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 8 {
		t.Fatalf("expected 8, got %d", val.Int64())
	}
}

func TestInterp_SubWrap(t *testing.T) {
	// PUSH1 5 PUSH1 3 SUB → 5-3=2 (but stack order: first push=bottom)
	// Actually: SUB pops a then b, returns a-b
	// PUSH1 3 PUSH1 5 SUB → 5-3 = 2
	bytecode := []byte{
		PUSH1, 3,
		PUSH1, 5,
		SUB,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 2 {
		t.Fatalf("expected 2, got %s", val)
	}
}

func TestInterp_SubUnderflow(t *testing.T) {
	// PUSH1 5 PUSH1 3 SUB → 3-5 wraps to MaxUint256-1
	bytecode := []byte{
		PUSH1, 5,
		PUSH1, 3,
		SUB,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	// 3 - 5 = -2, which wraps to 2^256 - 2
	expected := new(big.Int).Sub(WordMod, big.NewInt(2))
	if val.Cmp(expected) != 0 {
		t.Fatalf("expected %s, got %s", expected, val)
	}
}

func TestInterp_MulOverflow(t *testing.T) {
	// MaxUint256 * 2 should overflow to MaxUint256-1
	bytecode := []byte{
		PUSH32,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		PUSH1, 2,
		MUL,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	// MaxUint256 * 2 mod 2^256 = (2^256-1)*2 mod 2^256 = 2^257-2 mod 2^256 = 2^256-2
	expected := new(big.Int).Sub(MaxUint256, big.NewInt(1))
	if val.Cmp(expected) != 0 {
		t.Fatalf("expected %s, got %s", expected, val)
	}
}

func TestInterp_DivByZero(t *testing.T) {
	// 10 / 0 = 0
	bytecode := []byte{
		PUSH1, 0,
		PUSH1, 10,
		DIV,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatalf("expected 0 for div by zero, got %s", val)
	}
}

func TestInterp_MstoreMload(t *testing.T) {
	// Store 42 at offset 0, load it back
	bytecode := []byte{
		PUSH1, 42,
		PUSH1, 0,
		MSTORE,
		PUSH1, 0,
		MLOAD,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 42 {
		t.Fatalf("expected 42, got %d", val.Int64())
	}
}

func TestInterp_SstoreSload(t *testing.T) {
	// Store 99 at slot 1, load it back
	bytecode := []byte{
		PUSH1, 99,  // value
		PUSH1, 1,   // key
		SSTORE,
		PUSH1, 1,   // key
		SLOAD,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatalf("expected success, got %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 99 {
		t.Fatalf("expected 99, got %d", val.Int64())
	}
	// Verify state change recorded
	if len(result.StateChanges) != 1 {
		t.Fatalf("expected 1 state change, got %d", len(result.StateChanges))
	}
}

func TestInterp_JumpJumpdest(t *testing.T) {
	// PUSH1 4 JUMP STOP JUMPDEST PUSH1 42 PUSH1 0 MSTORE PUSH1 32 PUSH1 0 RETURN
	bytecode := []byte{
		PUSH1, 4,   // Push jump dest (byte 4)
		JUMP,       // Jump to byte 4
		STOP,       // Should be skipped
		JUMPDEST,   // Byte 4: landing pad
		PUSH1, 42,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatalf("expected success, got %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 42 {
		t.Fatalf("expected 42, got %d", val.Int64())
	}
}

func TestInterp_BadJump(t *testing.T) {
	// Jump to non-JUMPDEST location
	bytecode := []byte{
		PUSH1, 3, // byte 3 is PUSH1, not JUMPDEST
		JUMP,
		PUSH1, 0,
		STOP,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if result.Success {
		t.Fatal("expected failure for bad jump")
	}
}

func TestInterp_JumpiTrue(t *testing.T) {
	// PUSH1 1 (condition true) PUSH1 6 JUMPI STOP JUMPDEST PUSH1 99 ...
	bytecode := []byte{
		PUSH1, 1,   // condition (true)
		PUSH1, 6,   // dest
		JUMPI,      // byte 4: conditional jump
		STOP,       // byte 5: skipped
		JUMPDEST,   // byte 6: target
		PUSH1, 99,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 99 {
		t.Fatalf("expected 99, got %d", val.Int64())
	}
}

func TestInterp_JumpiFalse(t *testing.T) {
	// Condition 0 → falls through to PUSH1 77
	bytecode := []byte{
		PUSH1, 0,   // condition (false)
		PUSH1, 8,   // dest (doesn't matter)
		JUMPI,      // falls through
		PUSH1, 77,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 77 {
		t.Fatalf("expected 77, got %d", val.Int64())
	}
}

func TestInterp_Push32(t *testing.T) {
	val := make([]byte, 33)
	val[0] = PUSH32
	for i := 1; i <= 32; i++ {
		val[i] = 0xAB
	}
	bytecode := append(val, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	for _, b := range result.ReturnData {
		if b != 0xAB {
			t.Fatal("expected all 0xAB bytes")
		}
	}
}

func TestInterp_DupSwap(t *testing.T) {
	// PUSH1 10 PUSH1 20 DUP2 → stack: [10, 20, 10]
	// SWAP1 → stack: [10, 10, 20]
	// ADD → stack: [10, 30]
	// ADD → stack: [40]
	bytecode := []byte{
		PUSH1, 10,
		PUSH1, 20,
		DUP2,
		SWAP1,
		ADD,
		ADD,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 40 {
		t.Fatalf("expected 40, got %d", val.Int64())
	}
}

func TestInterp_LogEmission(t *testing.T) {
	// LOG1 with one topic
	bytecode := []byte{
		PUSH1, 0xAA, // topic
		PUSH1, 0,    // length
		PUSH1, 0,    // offset
		LOG1,
		STOP,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	if len(result.Logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(result.Logs))
	}
	if len(result.Logs[0].Topics) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(result.Logs[0].Topics))
	}
}

func TestInterp_Return(t *testing.T) {
	// Store 0xBEEF in memory, return 2 bytes
	bytecode := []byte{
		PUSH2, 0xBE, 0xEF,
		PUSH1, 0,
		MSTORE,
		PUSH1, 2,  // length
		PUSH1, 30, // offset (last 2 bytes of 32-byte word)
		RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	if len(result.ReturnData) != 2 {
		t.Fatalf("expected 2 bytes return data, got %d", len(result.ReturnData))
	}
	if result.ReturnData[0] != 0xBE || result.ReturnData[1] != 0xEF {
		t.Fatalf("expected 0xBEEF, got %X", result.ReturnData)
	}
}

func TestInterp_Revert(t *testing.T) {
	// Revert with error data
	bytecode := []byte{
		PUSH1, 42,
		PUSH1, 0,
		MSTORE,
		PUSH1, 32,
		PUSH1, 0,
		REVERT,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if result.Success {
		t.Fatal("expected revert")
	}
	if result.Error == nil || result.Error.Code != ErrCodeRevert {
		t.Fatal("expected revert error code")
	}
	if len(result.ReturnData) != 32 {
		t.Fatalf("expected 32 bytes return data, got %d", len(result.ReturnData))
	}
	// State changes should be empty on revert
	if len(result.StateChanges) != 0 {
		t.Fatal("expected no state changes after revert")
	}
}

func TestInterp_RevertClearsStateChanges(t *testing.T) {
	// SSTORE then REVERT — changes should be discarded
	bytecode := []byte{
		PUSH1, 99,  // value
		PUSH1, 1,   // key
		SSTORE,
		PUSH1, 0,
		PUSH1, 0,
		REVERT,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if result.Success {
		t.Fatal("expected revert")
	}
	if len(result.StateChanges) != 0 {
		t.Fatalf("expected 0 state changes after revert, got %d", len(result.StateChanges))
	}
}

func TestInterp_OutOfGas(t *testing.T) {
	// Give very little gas
	bytecode := []byte{
		PUSH1, 1,
		PUSH1, 2,
		ADD,
		PUSH1, 3,
		ADD,
	}
	ctx := makeCtx(10) // Only 10 gas
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if result.Success {
		t.Fatal("expected out of gas")
	}
}

func TestInterp_StaticCallRejectsSstore(t *testing.T) {
	bytecode := []byte{
		PUSH1, 42,
		PUSH1, 0,
		SSTORE,
		STOP,
	}
	ctx := makeCtx(100000)
	ctx.Mode = ModeStaticCall
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if result.Success {
		t.Fatal("expected failure for SSTORE in static call")
	}
	if result.Error == nil || result.Error.Code != ErrCodeStaticStateChange {
		t.Fatal("expected static state change error")
	}
}

func TestInterp_StaticCallRejectsLog(t *testing.T) {
	bytecode := []byte{
		PUSH1, 0,
		PUSH1, 0,
		LOG0,
		STOP,
	}
	ctx := makeCtx(100000)
	ctx.Mode = ModeStaticCall
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if result.Success {
		t.Fatal("expected failure for LOG in static call")
	}
}

func TestInterp_InvalidOpcode(t *testing.T) {
	bytecode := []byte{0x0C} // Not a valid opcode
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if result.Success {
		t.Fatal("expected failure for invalid opcode")
	}
}

func TestInterp_Stop(t *testing.T) {
	bytecode := []byte{PUSH1, 42, STOP, PUSH1, 99}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
}

func TestInterp_CallerOrigin(t *testing.T) {
	caller := make([]byte, 20)
	caller[19] = 0x42
	origin := make([]byte, 20)
	origin[19] = 0x99

	// CALLER, ORIGIN
	bytecode := []byte{
		CALLER,
		PUSH1, 0, MSTORE,
		ORIGIN,
		PUSH1, 32, MSTORE,
		PUSH1, 64, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Caller = caller
	ctx.Origin = origin
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	callerVal := new(big.Int).SetBytes(result.ReturnData[:32])
	originVal := new(big.Int).SetBytes(result.ReturnData[32:])
	if callerVal.Int64() != 0x42 {
		t.Fatalf("expected caller 0x42, got %x", callerVal.Int64())
	}
	if originVal.Int64() != 0x99 {
		t.Fatalf("expected origin 0x99, got %x", originVal.Int64())
	}
}

func TestInterp_CallDataLoad(t *testing.T) {
	bytecode := []byte{
		PUSH1, 0,
		CALLDATALOAD,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	callData := make([]byte, 32)
	callData[31] = 0x42

	ctx := makeCtx(100000)
	ctx.CallData = callData
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 0x42 {
		t.Fatalf("expected 0x42, got %x", val.Int64())
	}
}

func TestInterp_CallDataSize(t *testing.T) {
	bytecode := []byte{
		CALLDATASIZE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.CallData = make([]byte, 100)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 100 {
		t.Fatalf("expected 100, got %d", val.Int64())
	}
}

func TestInterp_BlockNumber(t *testing.T) {
	bytecode := []byte{
		NUMBER,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.BlockNumber = 12345
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 12345 {
		t.Fatalf("expected 12345, got %d", val.Int64())
	}
}

func TestInterp_ChainID(t *testing.T) {
	bytecode := []byte{
		CHAINID,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.ChainID = big.NewInt(2521)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 2521 {
		t.Fatalf("expected 2521, got %d", val.Int64())
	}
}

func TestInterp_SHA3(t *testing.T) {
	// Store 0x01 in memory, hash 1 byte
	bytecode := []byte{
		PUSH1, 1,
		PUSH1, 0, MSTORE8,
		PUSH1, 1,  // length
		PUSH1, 0,  // offset
		SHA3,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	if len(result.ReturnData) != 32 {
		t.Fatalf("expected 32-byte hash, got %d", len(result.ReturnData))
	}
	// The hash should be non-zero
	all_zero := true
	for _, b := range result.ReturnData {
		if b != 0 {
			all_zero = false
			break
		}
	}
	if all_zero {
		t.Fatal("hash should not be all zeros")
	}
}

func TestInterp_Balance(t *testing.T) {
	addr := make([]byte, 20)
	addr[19] = 0x01

	bytecode := []byte{
		PUSH1, 0x01,
		BALANCE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	state.SetBalance(addr, big.NewInt(1000))
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1000 {
		t.Fatalf("expected 1000, got %d", val.Int64())
	}
}

func TestInterp_Comparison(t *testing.T) {
	// Test LT: 3 < 5 = 1
	bytecode := []byte{
		PUSH1, 5,
		PUSH1, 3,
		LT,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected 1 (3 < 5), got %d", val.Int64())
	}
}

func TestInterp_IsZero(t *testing.T) {
	bytecode := []byte{
		PUSH1, 0,
		ISZERO,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected 1, got %d", val.Int64())
	}
}

func TestInterp_Bitwise(t *testing.T) {
	// AND: 0xFF & 0x0F = 0x0F
	bytecode := []byte{
		PUSH1, 0x0F,
		PUSH1, 0xFF,
		AND,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 0x0F {
		t.Fatalf("expected 0x0F, got 0x%X", val.Int64())
	}
}

func TestInterp_Push0(t *testing.T) {
	bytecode := []byte{
		PUSH0,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatalf("expected 0, got %s", val)
	}
}

func TestInterp_GasOpcode(t *testing.T) {
	// GAS opcode returns remaining gas
	bytecode := []byte{
		GAS,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	// GAS returns remaining after GAS opcode itself consumed 2 gas
	if val.Uint64() >= ctx.GasLimit {
		t.Fatal("remaining gas should be less than limit")
	}
}

func TestInterp_Exp(t *testing.T) {
	// 2^10 = 1024
	bytecode := []byte{
		PUSH1, 10,
		PUSH1, 2,
		EXP,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1024 {
		t.Fatalf("expected 1024, got %d", val.Int64())
	}
}

func TestInterp_Mstore8(t *testing.T) {
	bytecode := []byte{
		PUSH1, 0xAB,
		PUSH1, 0,
		MSTORE8,
		PUSH1, 1,
		PUSH1, 0,
		RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	if len(result.ReturnData) != 1 || result.ReturnData[0] != 0xAB {
		t.Fatalf("expected 0xAB, got %X", result.ReturnData)
	}
}

func TestInterp_PC(t *testing.T) {
	// PC at byte 0 should return 0
	bytecode := []byte{
		PC,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 0 {
		t.Fatalf("expected PC=0, got %d", val.Int64())
	}
}

func TestInterp_Msize(t *testing.T) {
	bytecode := []byte{
		PUSH1, 42,
		PUSH1, 0,
		MSTORE,    // Expands memory to 32 bytes
		MSIZE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() < 32 {
		t.Fatalf("expected MSIZE >= 32, got %d", val.Int64())
	}
}

func TestInterp_CodeSize(t *testing.T) {
	bytecode := []byte{
		CODESIZE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != int64(len(bytecode)) {
		t.Fatalf("expected %d, got %d", len(bytecode), val.Int64())
	}
}

// --- Capability check tests ---

func TestCheckCapability_NilCaps(t *testing.T) {
	ctx := makeCtx(100000)
	// C-1 FIX: nil Capabilities = DENY all (secure default)
	if checkCapability(ctx, "stake") {
		t.Fatal("expected stake DENIED with nil capabilities")
	}
	if checkCapability(ctx, "transfer") {
		t.Fatal("expected transfer DENIED with nil capabilities")
	}
	if checkCapability(ctx, "submit_claims") {
		t.Fatal("expected submit_claims DENIED with nil capabilities")
	}
	if checkCapability(ctx, "vote") {
		t.Fatal("expected vote DENIED with nil capabilities")
	}
	if checkCapability(ctx, "unknown") {
		t.Fatal("expected unknown capability DENIED with nil capabilities")
	}
}

func TestCheckCapability_AllGranted(t *testing.T) {
	ctx := makeCtx(100000)
	// Explicit all-true capabilities (set by keeper for identity/operational key callers)
	ctx.Capabilities = &SessionCapabilities{
		CanTransfer:    true,
		CanStake:       true,
		CanSubmitClaims: true,
		CanVote:        true,
	}
	if !checkCapability(ctx, "stake") {
		t.Fatal("expected stake allowed with all capabilities granted")
	}
	if !checkCapability(ctx, "transfer") {
		t.Fatal("expected transfer allowed with all capabilities granted")
	}
	if !checkCapability(ctx, "submit_claims") {
		t.Fatal("expected submit_claims allowed with all capabilities granted")
	}
	if !checkCapability(ctx, "vote") {
		t.Fatal("expected vote allowed with all capabilities granted")
	}
	// Unknown capability should still be denied
	if checkCapability(ctx, "unknown") {
		t.Fatal("expected unknown capability DENIED even with all capabilities granted")
	}
}

func TestCheckCapability_StakeBlocked(t *testing.T) {
	ctx := makeCtx(100000)
	ctx.Capabilities = &SessionCapabilities{
		CanTransfer:    true,
		CanStake:       false,
		CanSubmitClaims: true,
		CanVote:        true,
	}
	if checkCapability(ctx, "stake") {
		t.Fatal("expected stake blocked")
	}
	if !checkCapability(ctx, "transfer") {
		t.Fatal("expected transfer allowed")
	}
	if !checkCapability(ctx, "submit_claims") {
		t.Fatal("expected submit_claims allowed")
	}
	if !checkCapability(ctx, "vote") {
		t.Fatal("expected vote allowed")
	}
}

func TestCheckCapability_TransferBlocked(t *testing.T) {
	ctx := makeCtx(100000)
	ctx.Capabilities = &SessionCapabilities{
		CanTransfer:    false,
		CanStake:       true,
		CanSubmitClaims: true,
		CanVote:        true,
	}
	if checkCapability(ctx, "transfer") {
		t.Fatal("expected transfer blocked")
	}
	if !checkCapability(ctx, "stake") {
		t.Fatal("expected stake allowed")
	}
}

func TestCheckCapability_VoteBlocked(t *testing.T) {
	ctx := makeCtx(100000)
	ctx.Capabilities = &SessionCapabilities{
		CanTransfer:    true,
		CanStake:       true,
		CanSubmitClaims: true,
		CanVote:        false,
	}
	if checkCapability(ctx, "vote") {
		t.Fatal("expected vote blocked")
	}
}

func TestCheckCapability_SubmitClaimsBlocked(t *testing.T) {
	ctx := makeCtx(100000)
	ctx.Capabilities = &SessionCapabilities{
		CanTransfer:    true,
		CanStake:       true,
		CanSubmitClaims: false,
		CanVote:        true,
	}
	if checkCapability(ctx, "submit_claims") {
		t.Fatal("expected submit_claims blocked")
	}
}

func TestCheckCapability_AllBlocked(t *testing.T) {
	ctx := makeCtx(100000)
	ctx.Capabilities = &SessionCapabilities{
		CanTransfer:    false,
		CanStake:       false,
		CanSubmitClaims: false,
		CanVote:        false,
	}
	if checkCapability(ctx, "stake") {
		t.Fatal("expected stake blocked")
	}
	if checkCapability(ctx, "transfer") {
		t.Fatal("expected transfer blocked")
	}
	if checkCapability(ctx, "submit_claims") {
		t.Fatal("expected submit_claims blocked")
	}
	if checkCapability(ctx, "vote") {
		t.Fatal("expected vote blocked")
	}
	// C-1 FIX: Unknown capabilities now default to DENIED
	if checkCapability(ctx, "unknown_cap") {
		t.Fatal("expected unknown capability DENIED")
	}
}

func TestInterp_AgentOpcodeNilHost(t *testing.T) {
	// Agent opcodes with nil host should gracefully push 0
	bytecode := []byte{
		AGENT_DID,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success with nil host")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatalf("expected 0 from agent opcode with nil host, got %s", val)
	}
}

func TestReentrancyGuard_RejectsNestedAgentOpcode(t *testing.T) {
	// H-1: When InHostCall is true, agent opcodes should fail (reentrancy blocked)
	bytecode := []byte{
		AGENT_DID, // This should fail because InHostCall = true
		STOP,
	}
	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:test1234567890abcdef"
	ctx.Capabilities = &SessionCapabilities{
		CanTransfer: true, CanStake: true,
		CanSubmitClaims: true, CanVote: true,
	}
	ctx.InHostCall = true // Simulate being inside a host call
	ctx.Bytecode = bytecode

	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)

	if result.Success {
		t.Fatal("expected failure: agent opcode should be rejected during host call (reentrancy guard)")
	}
}

func TestReentrancyGuard_AllowsNormalExecution(t *testing.T) {
	// H-1: When InHostCall is false, agent opcodes should work normally
	bytecode := []byte{
		AGENT_DID,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:test1234567890abcdef"
	ctx.InHostCall = false // Normal execution
	ctx.Bytecode = bytecode

	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)

	if !result.Success {
		t.Fatalf("expected success when InHostCall is false, got error: %v", result.Error)
	}
}

func TestInterp_SstoreDynamicGas(t *testing.T) {
	// New slot write costs 20000
	bytecode := []byte{
		PUSH1, 42,
		PUSH1, 0,
		SSTORE,
		STOP,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	// Gas used should include: PUSH1(3) + PUSH1(3) + SSTORE(0 base + 20000 dynamic) + STOP(0)
	// = 20006
	if result.GasUsed < 20006 {
		t.Fatalf("expected >= 20006 gas used for new SSTORE, got %d", result.GasUsed)
	}
}

func TestInterp_EmptyBytecode(t *testing.T) {
	bytecode := []byte{}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("empty bytecode should succeed")
	}
	if result.GasUsed != 0 {
		t.Fatalf("empty bytecode should use 0 gas, got %d", result.GasUsed)
	}
}

func TestInterp_SelfBalance(t *testing.T) {
	contract := make([]byte, 20)
	contract[19] = 0x42

	bytecode := []byte{
		SELFBALANCE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.CurrentContract = contract
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	state.SetBalance(contract, big.NewInt(5000))
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 5000 {
		t.Fatalf("expected 5000, got %d", val.Int64())
	}
}

func TestInterp_SHL(t *testing.T) {
	// 1 << 4 = 16
	bytecode := []byte{
		PUSH1, 1,
		PUSH1, 4,
		SHL,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 16 {
		t.Fatalf("expected 16, got %d", val.Int64())
	}
}

func TestInterp_SHR(t *testing.T) {
	// 16 >> 2 = 4
	bytecode := []byte{
		PUSH1, 16,
		PUSH1, 2,
		SHR,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 4 {
		t.Fatalf("expected 4, got %d", val.Int64())
	}
}

func TestInterp_ExtCodeSize(t *testing.T) {
	addr := make([]byte, 20)
	addr[19] = 0x01

	bytecode := []byte{
		PUSH1, 0x01,
		EXTCODESIZE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	state.SetCode(addr, []byte{0x01, 0x02, 0x03})
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 3 {
		t.Fatalf("expected 3, got %d", val.Int64())
	}
}

func TestInterp_Eq(t *testing.T) {
	bytecode := []byte{
		PUSH1, 5,
		PUSH1, 5,
		EQ,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected 1, got %d", val.Int64())
	}
}

func TestInterp_Not(t *testing.T) {
	// NOT(0) = MaxUint256
	bytecode := []byte{
		PUSH1, 0,
		NOT,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Cmp(MaxUint256) != 0 {
		t.Fatalf("expected MaxUint256, got %s", val)
	}
}

func TestInterp_Addmod(t *testing.T) {
	// (10 + 10) % 8 = 4
	bytecode := []byte{
		PUSH1, 8,
		PUSH1, 10,
		PUSH1, 10,
		ADDMOD,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 4 {
		t.Fatalf("expected 4, got %d", val.Int64())
	}
}

func TestInterp_Mulmod(t *testing.T) {
	// (10 * 10) % 8 = 4
	bytecode := []byte{
		PUSH1, 8,
		PUSH1, 10,
		PUSH1, 10,
		MULMOD,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 4 {
		t.Fatalf("expected 4, got %d", val.Int64())
	}
}

func TestInterp_Pop(t *testing.T) {
	bytecode := []byte{
		PUSH1, 42,
		PUSH1, 99,
		POP,  // Discard 99
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 42 {
		t.Fatalf("expected 42, got %d", val.Int64())
	}
}

func TestInterp_CallValue(t *testing.T) {
	bytecode := []byte{
		CALLVALUE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Value = big.NewInt(1000000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1000000 {
		t.Fatalf("expected 1000000, got %d", val.Int64())
	}
}

func TestInterp_ByteExtract(t *testing.T) {
	// BYTE(31, 0xFF) = 0xFF (byte 31 is the least significant byte)
	bytecode := []byte{
		PUSH1, 0xFF,
		PUSH1, 31,
		BYTE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 0xFF {
		t.Fatalf("expected 0xFF, got 0x%X", val.Int64())
	}
}

func TestInterp_Timestamp(t *testing.T) {
	bytecode := []byte{
		TIMESTAMP,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Timestamp = 1700000000
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatal("expected success")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Uint64() != 1700000000 {
		t.Fatalf("expected 1700000000, got %d", val.Uint64())
	}
}

// --- Safety Bounds Tests ---

func TestInterp_CallDepthExceeded(t *testing.T) {
	bytecode := []byte{STOP}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	interp.callDepth = MaxCallDepth + 1

	result := interp.Execute(bytecode, ctx, state, nil)
	if result.Success {
		t.Fatal("expected call depth exceeded error")
	}
	if result.Error.Code != ErrCodeCallDepthExceeded {
		t.Fatalf("expected ErrCodeCallDepthExceeded, got %d", result.Error.Code)
	}
}

func TestInterp_BytecodeSizeExceeded(t *testing.T) {
	// Create bytecode larger than MaxBytecodeSize
	bytecode := make([]byte, MaxBytecodeSize+1)
	bytecode[0] = STOP
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()

	result := interp.Execute(bytecode, ctx, state, nil)
	if result.Success {
		t.Fatal("expected bytecode size exceeded error")
	}
}

func TestInterp_BytecodeSizeAtLimit(t *testing.T) {
	// Bytecode exactly at MaxBytecodeSize should be OK
	bytecode := make([]byte, MaxBytecodeSize)
	bytecode[0] = STOP
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()

	result := interp.Execute(bytecode, ctx, state, nil)
	if !result.Success {
		t.Fatalf("bytecode at max size should succeed, got error: %v", result.Error)
	}
}
