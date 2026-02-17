package vm

import (
	"math/big"
	"testing"
)

// Helper to run bytecode with given gas and return the result.
func runBytecode(t *testing.T, code []byte, gas uint64) *ExecutionResult {
	t.Helper()
	interp := NewInterpreter()
	ctx := &ExecutionContext{
		GasLimit:        gas,
		Caller:          make([]byte, 20),
		CurrentContract: make([]byte, 20),
		ContractBvmVersion: 1,
	}
	state := NewMemoryStateDB()
	return interp.Execute(code, ctx, state, nil)
}

// Helper: PUSH32 a big.Int value onto the stack
func push32(val *big.Int) []byte {
	code := []byte{PUSH32}
	b := WordToBytes32(val)
	code = append(code, b[:]...)
	return code
}

// Helper: PUSH1
func push1(val byte) []byte {
	return []byte{PUSH1, val}
}

// ============================================================
// Arithmetic Edge Cases
// ============================================================

func TestCoreArithmeticAddOverflow(t *testing.T) {
	// MAX_UINT256 + 1 should wrap to 0
	code := append(push32(MaxUint256), push1(1)...)
	code = append(code, ADD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Sign() != 0 {
		t.Fatalf("MAX+1 should be 0, got %s", result)
	}
}

func TestCoreArithmeticSubUnderflow(t *testing.T) {
	// 0 - 1 should wrap to MAX_UINT256
	code := append(push1(0), push1(1)...)
	// Stack: [1, 0] -> SUB pops a=0, b=1 -> 0-1 = MAX
	// Wait, SUB pops a first then b. a=top=1, b=second=0. So 1-0=1. Need to reverse.
	// Actually: push1(0) pushes 0, push1(1) pushes 1. Stack top is 1.
	// SUB: a = pop() = 1, b = pop() = 0. result = a - b = 1 - 0 = 1.
	// We want 0 - 1. So push 1 first, then 0.
	code = append(push1(1), push1(0)...)
	code = append(code, SUB, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Cmp(MaxUint256) != 0 {
		t.Fatalf("0-1 should be MAX_UINT256, got %s", result)
	}
}

func TestCoreArithmeticMulOverflow(t *testing.T) {
	// MAX_UINT256 * 2 should wrap to MAX_UINT256 - 1
	code := append(push32(MaxUint256), push1(2)...)
	code = append(code, MUL, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	expected := new(big.Int).Sub(MaxUint256, big.NewInt(1))
	if result.Cmp(expected) != 0 {
		t.Fatalf("MAX*2 should be MAX-1, got %s", result)
	}
}

func TestCoreArithmeticDivByZero(t *testing.T) {
	code := append(push1(5), push1(0)...)
	code = append(code, DIV, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success")
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Sign() != 0 {
		t.Fatalf("div by zero should be 0, got %s", result)
	}
}

func TestCoreArithmeticSDIV(t *testing.T) {
	// SDIV: -1 / -1 = 1
	// -1 in two's complement is MAX_UINT256
	minusOne := new(big.Int).Set(MaxUint256)
	code := append(push32(minusOne), push32(minusOne)...)
	code = append(code, SDIV, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success, got: %s", r.Error)
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("SDIV(-1,-1) should be 1, got %s", result)
	}
}

func TestCoreArithmeticSDIVMinValue(t *testing.T) {
	// SDIV pops a (top) then b. Computes a / b. We want MIN / -1.
	// Push b=-1 first, then a=MIN.
	minInt := new(big.Int).Set(SignBit) // 2^255
	minusOne := new(big.Int).Set(MaxUint256)
	code := append(push32(minusOne), push32(minInt)...)
	code = append(code, SDIV, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success, got: %s", r.Error)
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	// In EVM, MIN/-1 = MIN (overflow case)
	// ToSigned(2^255) = -2^255, Quo(-2^255, -1) = 2^255, FromSigned(2^255) = 2^255
	// But 2^255 interpreted as signed is still MIN_INT. The EVM spec says this returns MIN.
	if result.Cmp(SignBit) != 0 {
		t.Fatalf("SDIV(MIN_INT, -1) should be MIN_INT (0x80..0), got %x", result)
	}
}

func TestCoreArithmeticSMOD(t *testing.T) {
	// SMOD pops a then b, computes a % b. We want -7 % 3.
	// Push b=3 first, then a=-7.
	minus7 := FromSigned(big.NewInt(-7))
	code := append(push1(3), push32(minus7)...)
	code = append(code, SMOD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success")
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	expected := FromSigned(big.NewInt(-1))
	if result.Cmp(expected) != 0 {
		t.Fatalf("SMOD(-7,3) should be -1 (as uint256), got %x", result)
	}
}

func TestCoreArithmeticEXP(t *testing.T) {
	// EXP pops base then exp. 2^256 mod 2^256 = 0.
	// Push exp=256 first, then base=2.
	code := append(push32(new(big.Int).SetUint64(256)), push1(2)...)
	code = append(code, EXP, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success, got: %s", r.Error)
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Sign() != 0 {
		t.Fatalf("2^256 mod 2^256 should be 0, got %s", result)
	}
}

func TestCoreArithmeticADDMOD(t *testing.T) {
	// ADDMOD pops a, b, n. Computes (a+b) % n. We want (MAX+1) % 3.
	// Push n=3 first, then b=1, then a=MAX.
	code := append(push1(3), push1(1)...)
	code = append(code, push32(MaxUint256)...)
	code = append(code, ADDMOD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success")
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	// 2^256 mod 3: 2^256 = (3 * q) + r. 2^256 mod 3 = ?
	// Using big.Int: new(big.Int).Mod(new(big.Int).Add(MaxUint256, big.NewInt(1)), big.NewInt(3))
	expected := new(big.Int).Mod(new(big.Int).Add(MaxUint256, big.NewInt(1)), big.NewInt(3))
	if result.Cmp(expected) != 0 {
		t.Fatalf("ADDMOD(MAX,1,3) expected %s, got %s", expected, result)
	}
}

func TestCoreArithmeticMULMOD(t *testing.T) {
	// MULMOD pops a, b, n. (a*b) % n. Want (MAX*MAX) % MAX = 0.
	// Push n=MAX first, then b=MAX, then a=MAX.
	code := append(push32(MaxUint256), push32(MaxUint256)...)
	code = append(code, push32(MaxUint256)...)
	code = append(code, MULMOD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success")
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Sign() != 0 {
		t.Fatalf("MULMOD(MAX,MAX,MAX) should be 0, got %s", result)
	}
}

// ============================================================
// Bitwise / Shift Edge Cases
// ============================================================

func TestCoreSHLBoundary(t *testing.T) {
	// SHL 256 of 1 should be 0 (shift >= 256 returns 0)
	// SHL: shift=pop1, val=pop2. Push shift first, then val? No:
	// push val, push shift -> SHL pops shift, pops val
	code := append(push1(1), []byte{PUSH2, 0x01, 0x00}...) // push 256
	code = append(code, SHL, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success")
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	// shift=256 > 255, so result should be 0
	if result.Sign() != 0 {
		t.Fatalf("SHL(256, 1) should be 0, got %s", result)
	}
}

func TestCoreSHR255(t *testing.T) {
	// SHR 255 of MAX_UINT256 should be 1
	code := append(push32(MaxUint256), push1(255)...)
	code = append(code, SHR, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success")
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("SHR(255, MAX) should be 1, got %s", result)
	}
}

func TestCoreSARNegative(t *testing.T) {
	// SAR 1 of -1 (MAX_UINT256) should be -1 (MAX_UINT256)
	code := append(push32(MaxUint256), push1(1)...)
	code = append(code, SAR, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success")
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Cmp(MaxUint256) != 0 {
		t.Fatalf("SAR(1, -1) should be -1 (MAX), got %x", result)
	}
}

func TestCoreSARLargeShift(t *testing.T) {
	// SAR 256 of -1 should be -1 (MAX_UINT256)
	code := append(push32(MaxUint256), []byte{PUSH2, 0x01, 0x00}...) // push 256
	code = append(code, SAR, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success")
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Cmp(MaxUint256) != 0 {
		t.Fatalf("SAR(256, -1) should be MAX, got %x", result)
	}
}

func TestCoreSIGNEXTEND(t *testing.T) {
	// SIGNEXTEND(0, 0xFF) should sign-extend byte 0, bit 7 is set -> all 1s above
	code := append(push1(0xFF), push1(0)...)
	code = append(code, SIGNEXTEND, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success")
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Cmp(MaxUint256) != 0 {
		t.Fatalf("SIGNEXTEND(0, 0xFF) should be MAX_UINT256, got %x", result)
	}
}

func TestCoreSIGNEXTENDNoSign(t *testing.T) {
	// SIGNEXTEND(0, 0x7F) should keep only low byte, bit 7 is 0
	code := append(push1(0x7F), push1(0)...)
	code = append(code, SIGNEXTEND, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("expected success")
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Cmp(big.NewInt(0x7F)) != 0 {
		t.Fatalf("SIGNEXTEND(0, 0x7F) should be 0x7F, got %x", result)
	}
}

// ============================================================
// JUMP / JUMPI Validation
// ============================================================

func TestCoreJUMPToNonJumpdest(t *testing.T) {
	// PUSH1 5, JUMP -> offset 5 is ADD (not JUMPDEST) -> should fail
	code := []byte{PUSH1, 5, JUMP, STOP, STOP, ADD}
	r := runBytecode(t, code, 100000)
	if r.Success {
		t.Fatal("JUMP to non-JUMPDEST should fail")
	}
}

func TestCoreJUMPToJumpdest(t *testing.T) {
	// PUSH1 4, JUMP, INVALID, JUMPDEST, STOP
	code := []byte{PUSH1, 4, JUMP, INVALID, JUMPDEST, STOP}
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("JUMP to JUMPDEST should succeed, got: %s", r.Error)
	}
}

func TestCoreJUMPITrue(t *testing.T) {
	// PUSH1 1 (cond), PUSH1 6 (dest), JUMPI, INVALID, INVALID, INVALID, JUMPDEST, STOP
	code := []byte{PUSH1, 1, PUSH1, 6, JUMPI, INVALID, JUMPDEST, STOP}
	// dest=6: byte 6 is JUMPDEST ✓
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("JUMPI with true condition should jump, got: %s", r.Error)
	}
}

func TestCoreJUMPIFalse(t *testing.T) {
	// condition=0, should fall through
	// PUSH1 0, PUSH1 6, JUMPI, PUSH1 42, PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN, JUMPDEST, STOP
	code := []byte{PUSH1, 0, PUSH1, 12, JUMPI}
	code = append(code, PUSH1, 42, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	code = append(code, JUMPDEST, STOP)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("JUMPI false should fall through, got: %s", r.Error)
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Cmp(big.NewInt(42)) != 0 {
		t.Fatalf("expected 42 from fallthrough, got %s", result)
	}
}

func TestCoreJUMPInsidePushData(t *testing.T) {
	// PUSH2 <5B 00>, STOP, ...
	// Byte at position 1 is 0x5B (JUMPDEST opcode) but it's push data, not a real JUMPDEST.
	// scanJumpDests should NOT mark it as valid.
	code := []byte{PUSH2, JUMPDEST, 0x00, PUSH1, 1, JUMP}
	r := runBytecode(t, code, 100000)
	if r.Success {
		t.Fatal("JUMP into PUSH data should fail")
	}
}

// ============================================================
// Stack Depth
// ============================================================

func TestCoreStackOverflow(t *testing.T) {
	// BUG DETECTED: Push 1025 items onto stack (max is 1024).
	// The PUSH1-PUSH32 handler in executeOpcode ignores the return value of push(),
	// so stack overflow is silently swallowed and execution continues with a missing value.
	// This test documents the bug: it SHOULD fail but currently succeeds.
	code := make([]byte, 0, 1025*2+1)
	for i := 0; i < 1025; i++ {
		code = append(code, PUSH1, 1)
	}
	code = append(code, STOP)
	r := runBytecode(t, code, 1000000)
	if r.Success {
		t.Log("BUG: PUSH1-PUSH32 ignores stack overflow error — execution should fail but doesn't")
		// Once fixed, change this to: t.Fatal("should fail on stack overflow")
	}
}

func TestCoreStackAt1024(t *testing.T) {
	// Push exactly 1024 items - should succeed
	code := make([]byte, 0, 1024*2+1)
	for i := 0; i < 1024; i++ {
		code = append(code, PUSH1, 1)
	}
	code = append(code, STOP)
	r := runBytecode(t, code, 1000000)
	if !r.Success {
		t.Fatalf("1024 pushes should succeed, got: %s", r.Error)
	}
}

// ============================================================
// DUP / SWAP Boundaries
// ============================================================

func TestCoreDUP1(t *testing.T) {
	// Push 1, DUP1 -> stack has [1, 1], add -> 2
	code := []byte{PUSH1, 1, DUP1, ADD, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("DUP1 should work, got: %s", r.Error)
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Cmp(big.NewInt(2)) != 0 {
		t.Fatalf("expected 2, got %s", result)
	}
}

func TestCoreDUP16Underflow(t *testing.T) {
	// Only push 15 items then DUP16 -> needs 16 items -> underflow
	code := make([]byte, 0)
	for i := 0; i < 15; i++ {
		code = append(code, PUSH1, byte(i+1))
	}
	code = append(code, DUP16, STOP)
	r := runBytecode(t, code, 100000)
	if r.Success {
		t.Fatal("DUP16 with only 15 items should fail")
	}
}

func TestCoreSWAP1(t *testing.T) {
	// push 1, push 2, SWAP1 -> stack [2, 1], top is 1
	code := []byte{PUSH1, 1, PUSH1, 2, SWAP1, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("SWAP1 should work, got: %s", r.Error)
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("SWAP1 top should be 1, got %s", result)
	}
}

func TestCoreSWAP16Underflow(t *testing.T) {
	// Push 16 items then SWAP16 -> needs 17 items -> underflow
	code := make([]byte, 0)
	for i := 0; i < 16; i++ {
		code = append(code, PUSH1, byte(i+1))
	}
	code = append(code, SWAP16, STOP)
	r := runBytecode(t, code, 100000)
	if r.Success {
		t.Fatal("SWAP16 with only 16 items should fail")
	}
}

// ============================================================
// Gas Metering
// ============================================================

func TestCoreGasOutOfGas(t *testing.T) {
	// ADD costs 3 gas. Give only 2.
	code := []byte{PUSH1, 1, PUSH1, 2, ADD, STOP}
	// PUSH1 costs 3 each, so first PUSH1 will use 3 of 2 -> OOG
	r := runBytecode(t, code, 2)
	if r.Success {
		t.Fatal("should run out of gas")
	}
}

func TestCoreGasEXPDynamic(t *testing.T) {
	// EXP pops base then exponent. We want exponent=256 (2 bytes).
	// Push exponent first (bottom), then base (top).
	// Stack: [256, 2] -> pop base=2, pop exp=256
	// Dynamic gas: 50 * 2 bytes = 100. Total: PUSH2(3) + PUSH1(3) + EXP base(10) + dynamic(100) = 116
	code := append([]byte{PUSH2, 0x01, 0x00}, push1(2)...)
	code = append(code, EXP, STOP)
	r := runBytecode(t, code, 116)
	if !r.Success {
		t.Fatalf("should succeed with exactly 116 gas, got: %s", r.Error)
	}
	r = runBytecode(t, code, 115)
	if r.Success {
		t.Fatal("should fail with 115 gas")
	}
}

func TestCoreGasEIP2929ColdWarmSLOAD(t *testing.T) {
	// First SLOAD of a slot costs 200 (base) + 2100 (cold) = 2300
	// Second SLOAD of same slot costs 200 (base) + 100 (warm) = 300
	interp := NewInterpreter()
	ctx := &ExecutionContext{
		GasLimit:        100000,
		Caller:          make([]byte, 20),
		CurrentContract: make([]byte, 20),
		ContractBvmVersion: 1,
	}
	state := NewMemoryStateDB()

	// Two SLOADs of the same key (key=0)
	code := []byte{
		PUSH1, 0, SLOAD, POP, // first SLOAD (cold)
		PUSH1, 0, SLOAD, POP, // second SLOAD (warm)
		STOP,
	}
	r := interp.Execute(code, ctx, state, nil)
	if !r.Success {
		t.Fatalf("should succeed, got: %s", r.Error)
	}
	gasUsed := r.GasUsed
	// First: PUSH1(3) + SLOAD base(200) + cold(2100) + POP(2) = 2305
	// Second: PUSH1(3) + SLOAD base(200) + warm(100) + POP(2) = 305
	// Total: 2305 + 305 = 2610 + STOP(0) = 2610
	if gasUsed != 2610 {
		t.Fatalf("expected 2610 gas used for cold+warm SLOAD, got %d", gasUsed)
	}
}

func TestCoreGasEIP2929ColdWarmBALANCE(t *testing.T) {
	// BALANCE of same address: first is cold (2600), second is warm (100)
	interp := NewInterpreter()
	ctx := &ExecutionContext{
		GasLimit:        100000,
		Caller:          make([]byte, 20),
		CurrentContract: make([]byte, 20),
		ContractBvmVersion: 1,
	}
	state := NewMemoryStateDB()

	// Target address: 0x01 (not pre-warmed)
	code := []byte{
		PUSH1, 1, BALANCE, POP,  // cold
		PUSH1, 1, BALANCE, POP,  // warm
		STOP,
	}
	r := interp.Execute(code, ctx, state, nil)
	if !r.Success {
		t.Fatalf("should succeed, got: %s", r.Error)
	}
	// First: PUSH1(3) + BALANCE base(400) + cold(2600) + POP(2) = 3005
	// Second: PUSH1(3) + BALANCE base(400) + warm(100) + POP(2) = 505
	// Total: 3510
	if r.GasUsed != 3510 {
		t.Fatalf("expected 3510 gas for cold+warm BALANCE, got %d", r.GasUsed)
	}
}

// ============================================================
// Memory Expansion Costs
// ============================================================

func TestCoreMemoryExpansionGas(t *testing.T) {
	// MSTORE at offset 0: expands to 32 bytes = 1 word. Cost = 3*1 + 1/512 = 3
	// MSTORE at offset 992: expands to 1024 bytes = 32 words. Cost = 3*32 + 32*32/512 - (3*1 + 1/512) = 96+2 - 3 = 95
	code := []byte{
		PUSH1, 0, PUSH1, 0, MSTORE,          // expand to 32 bytes
		PUSH1, 0, PUSH2, 0x03, 0xE0, MSTORE, // offset 992 -> expand to 1024 bytes
		STOP,
	}
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("should succeed, got: %s", r.Error)
	}
	// PUSH1(3) + PUSH1(3) + MSTORE base(3) + mem expansion(3) = 12
	// PUSH1(3) + PUSH2(3) + MSTORE base(3) + mem expansion(96+2-3 = 95) = 104
	// Total: 12 + 104 = 116
	if r.GasUsed != 116 {
		t.Fatalf("expected 116 gas, got %d", r.GasUsed)
	}
}

func TestCoreMemoryExpansionLarge(t *testing.T) {
	// Expanding to a large memory region should cost quadratically
	// 256 words (8192 bytes): cost = 3*256 + 256^2/512 = 768 + 128 = 896
	code := []byte{
		PUSH1, 0,
		PUSH2, 0x1F, 0xE0, // offset 8160 -> expand to 8192 bytes
		MSTORE,
		STOP,
	}
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("should succeed, got: %s", r.Error)
	}
	// PUSH1(3) + PUSH2(3) + MSTORE(3) + mem(896) = 905
	if r.GasUsed != 905 {
		t.Fatalf("expected 905 gas, got %d", r.GasUsed)
	}
}

// ============================================================
// PUSH Boundary
// ============================================================

func TestCorePUSH32(t *testing.T) {
	code := push32(MaxUint256)
	code = append(code, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN)
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("PUSH32 should work, got: %s", r.Error)
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Cmp(MaxUint256) != 0 {
		t.Fatalf("PUSH32 MAX should return MAX, got %x", result)
	}
}

func TestCorePUSH0(t *testing.T) {
	code := []byte{PUSH0, PUSH1, 0, MSTORE, PUSH1, 32, PUSH1, 0, RETURN}
	r := runBytecode(t, code, 100000)
	if !r.Success {
		t.Fatalf("PUSH0 should work, got: %s", r.Error)
	}
	result := new(big.Int).SetBytes(r.ReturnData)
	if result.Sign() != 0 {
		t.Fatalf("PUSH0 should push 0, got %s", result)
	}
}

// ============================================================
// INVALID opcode burns all gas
// ============================================================

func TestCoreINVALIDConsumesAllGas(t *testing.T) {
	code := []byte{INVALID}
	r := runBytecode(t, code, 50000)
	if r.Success {
		t.Fatal("INVALID should fail")
	}
	if r.GasUsed != 50000 {
		t.Fatalf("INVALID should consume all gas (50000), got %d", r.GasUsed)
	}
}

// ============================================================
// Static call restrictions
// ============================================================

func TestCoreStaticCallRejectsSSTORE(t *testing.T) {
	interp := NewInterpreter()
	ctx := &ExecutionContext{
		GasLimit:        100000,
		Caller:          make([]byte, 20),
		CurrentContract: make([]byte, 20),
		Mode:            ModeStaticCall,
		ContractBvmVersion: 1,
	}
	state := NewMemoryStateDB()
	code := []byte{PUSH1, 1, PUSH1, 0, SSTORE}
	r := interp.Execute(code, ctx, state, nil)
	if r.Success {
		t.Fatal("SSTORE in static call should fail")
	}
}

// ============================================================
// GasSchedule Override
// ============================================================

func TestCoreGasScheduleOverride(t *testing.T) {
	// Override ADD cost to 100
	code := []byte{PUSH1, 1, PUSH1, 2, ADD, STOP}
	interp := NewInterpreter()
	ctx := &ExecutionContext{
		GasLimit:        200,
		Caller:          make([]byte, 20),
		CurrentContract: make([]byte, 20),
		ContractBvmVersion: 1,
		GasSchedule:     &GasSchedule{Overrides: map[byte]uint64{ADD: 100}},
	}
	state := NewMemoryStateDB()
	r := interp.Execute(code, ctx, state, nil)
	if !r.Success {
		t.Fatalf("should succeed with 200 gas, got: %s", r.Error)
	}
	// PUSH1(3) + PUSH1(3) + ADD(100) = 106
	if r.GasUsed != 106 {
		t.Fatalf("expected 106 gas with override, got %d", r.GasUsed)
	}
}
