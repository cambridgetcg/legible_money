package vm

import (
	"math/big"
	"testing"
)

// --- Word Tests ---

func TestNewWord(t *testing.T) {
	w := NewWord(42)
	if w.Int64() != 42 {
		t.Fatalf("expected 42, got %d", w.Int64())
	}
}

func TestWordFromBytes(t *testing.T) {
	b := []byte{0x01, 0x00}
	w := WordFromBytes(b)
	if w.Int64() != 256 {
		t.Fatalf("expected 256, got %d", w.Int64())
	}
}

func TestWordToBytes32_RoundTrip(t *testing.T) {
	original := big.NewInt(0xDEADBEEF)
	b32 := WordToBytes32(original)
	recovered := WordFromBytes(b32[:])
	if original.Cmp(recovered) != 0 {
		t.Fatalf("round-trip failed: %s != %s", original, recovered)
	}
}

func TestWordToBytes32_MaxUint256(t *testing.T) {
	b32 := WordToBytes32(MaxUint256)
	for _, b := range b32 {
		if b != 0xFF {
			t.Fatalf("expected all 0xFF bytes for MaxUint256")
		}
	}
}

func TestToSigned_Positive(t *testing.T) {
	w := big.NewInt(100)
	s := ToSigned(w)
	if s.Int64() != 100 {
		t.Fatalf("expected 100, got %d", s.Int64())
	}
}

func TestToSigned_Negative(t *testing.T) {
	// MaxUint256 should be -1 in two's complement
	s := ToSigned(new(big.Int).Set(MaxUint256))
	if s.Int64() != -1 {
		t.Fatalf("expected -1, got %s", s)
	}
}

func TestFromSigned_Negative(t *testing.T) {
	neg := big.NewInt(-1)
	u := FromSigned(neg)
	if u.Cmp(MaxUint256) != 0 {
		t.Fatalf("expected MaxUint256, got %s", u)
	}
}

func TestMod256_Overflow(t *testing.T) {
	// 2^256 should wrap to 0
	val := new(big.Int).Set(WordMod)
	m := Mod256(val)
	if m.Sign() != 0 {
		t.Fatalf("expected 0, got %s", m)
	}
}

func TestMod256_MaxUint256PlusOne(t *testing.T) {
	val := new(big.Int).Add(MaxUint256, big.NewInt(1))
	m := Mod256(val)
	if m.Sign() != 0 {
		t.Fatalf("expected 0, got %s", m)
	}
}

// --- Stack Tests ---

func TestStack_PushPop(t *testing.T) {
	s := NewStack()
	if err := s.Push(big.NewInt(42)); err != nil {
		t.Fatal(err)
	}
	v, err := s.Pop()
	if err != nil {
		t.Fatal(err)
	}
	if v.Int64() != 42 {
		t.Fatalf("expected 42, got %d", v.Int64())
	}
}

func TestStack_PopEmpty(t *testing.T) {
	s := NewStack()
	_, err := s.Pop()
	if err != ErrStackUnderflow {
		t.Fatalf("expected underflow, got %v", err)
	}
}

func TestStack_Overflow(t *testing.T) {
	s := NewStack()
	for i := 0; i < DefaultMaxStackDepth; i++ {
		if err := s.Push(big.NewInt(int64(i))); err != nil {
			t.Fatalf("push %d failed: %v", i, err)
		}
	}
	if err := s.Push(big.NewInt(0)); err != ErrStackOverflow {
		t.Fatalf("expected overflow, got %v", err)
	}
}

func TestStack_Depth(t *testing.T) {
	s := NewStack()
	s.Push(big.NewInt(1))
	s.Push(big.NewInt(2))
	if s.Depth() != 2 {
		t.Fatalf("expected depth 2, got %d", s.Depth())
	}
}

func TestStack_Peek(t *testing.T) {
	s := NewStack()
	s.Push(big.NewInt(10))
	s.Push(big.NewInt(20))
	v, err := s.Peek(0) // top
	if err != nil {
		t.Fatal(err)
	}
	if v.Int64() != 20 {
		t.Fatalf("expected 20, got %d", v.Int64())
	}
	v, err = s.Peek(1)
	if err != nil {
		t.Fatal(err)
	}
	if v.Int64() != 10 {
		t.Fatalf("expected 10, got %d", v.Int64())
	}
}

func TestStack_PeekUnderflow(t *testing.T) {
	s := NewStack()
	s.Push(big.NewInt(1))
	_, err := s.Peek(1)
	if err != ErrStackUnderflow {
		t.Fatalf("expected underflow, got %v", err)
	}
}

func TestStack_Swap(t *testing.T) {
	s := NewStack()
	s.Push(big.NewInt(1)) // bottom
	s.Push(big.NewInt(2))
	s.Push(big.NewInt(3)) // top

	err := s.Swap(2) // swap top with 3rd from top
	if err != nil {
		t.Fatal(err)
	}
	v, _ := s.Pop()
	if v.Int64() != 1 {
		t.Fatalf("expected 1 at top after swap, got %d", v.Int64())
	}
}

func TestStack_SwapUnderflow(t *testing.T) {
	s := NewStack()
	s.Push(big.NewInt(1))
	err := s.Swap(1) // Only 1 item, can't swap with depth 1
	if err != ErrStackUnderflow {
		t.Fatalf("expected underflow, got %v", err)
	}
}

func TestStack_Dup(t *testing.T) {
	s := NewStack()
	s.Push(big.NewInt(10))
	s.Push(big.NewInt(20))

	err := s.Dup(2) // Dup the 2nd item (10)
	if err != nil {
		t.Fatal(err)
	}
	if s.Depth() != 3 {
		t.Fatalf("expected depth 3, got %d", s.Depth())
	}
	v, _ := s.Pop()
	if v.Int64() != 10 {
		t.Fatalf("expected 10, got %d", v.Int64())
	}
}

func TestStack_DupTop(t *testing.T) {
	s := NewStack()
	s.Push(big.NewInt(42))
	s.Dup(1) // Dup top
	if s.Depth() != 2 {
		t.Fatalf("expected depth 2, got %d", s.Depth())
	}
	v, _ := s.Pop()
	if v.Int64() != 42 {
		t.Fatalf("expected 42, got %d", v.Int64())
	}
}

func TestStack_PushPopIsolation(t *testing.T) {
	// Verify push copies value (modifying original doesn't affect stack)
	s := NewStack()
	val := big.NewInt(100)
	s.Push(val)
	val.SetInt64(999)
	v, _ := s.Pop()
	if v.Int64() != 100 {
		t.Fatalf("expected 100 (push should copy), got %d", v.Int64())
	}
}

// --- Memory Tests ---

func TestMemory_ReadWriteRoundTrip(t *testing.T) {
	m := NewMemory()
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	if err := m.Write(0, data); err != nil {
		t.Fatal(err)
	}
	result := m.Read(0, 4)
	for i, b := range result {
		if b != data[i] {
			t.Fatalf("byte %d: expected 0x%02X, got 0x%02X", i, data[i], b)
		}
	}
}

func TestMemory_ReadZeroPad(t *testing.T) {
	m := NewMemory()
	m.Write(0, []byte{0xFF})
	result := m.Read(0, 4)
	if result[0] != 0xFF || result[1] != 0 || result[2] != 0 || result[3] != 0 {
		t.Fatalf("expected zero-padded read, got %v", result)
	}
}

func TestMemory_ReadBeyondSize(t *testing.T) {
	m := NewMemory()
	result := m.Read(100, 4)
	for _, b := range result {
		if b != 0 {
			t.Fatal("reading beyond size should return zeros")
		}
	}
}

func TestMemory_AutoExpand(t *testing.T) {
	m := NewMemory()
	data := []byte{0x42}
	m.Write(100, data)
	if m.Size() < 101 {
		t.Fatalf("expected size >= 101, got %d", m.Size())
	}
	if m.GetByte(100) != 0x42 {
		t.Fatal("expected 0x42 at offset 100")
	}
}

func TestMemory_ExpandGasCost(t *testing.T) {
	m := NewMemory()
	gas1, err := m.Expand(32)
	if err != nil {
		t.Fatal(err)
	}
	if gas1 != 3 { // 3*1 + 1^2/512 = 3
		t.Fatalf("expected gas cost 3 for first word, got %d", gas1)
	}

	gas2, err := m.Expand(64)
	if err != nil {
		t.Fatal(err)
	}
	// memCost(2) - memCost(1) = (3*2 + 4/512) - (3*1 + 1/512) = 6 - 3 = 3
	if gas2 != 3 {
		t.Fatalf("expected gas cost 3 for second word, got %d", gas2)
	}
}

func TestMemory_ExpandNoOpIfSmaller(t *testing.T) {
	m := NewMemory()
	m.Expand(64)
	size := m.Size()
	gas, _ := m.Expand(32)
	if gas != 0 {
		t.Fatal("expanding to smaller size should cost 0 gas")
	}
	if m.Size() != size {
		t.Fatal("size should not change")
	}
}

func TestMemory_MStoreMLoad(t *testing.T) {
	m := NewMemory()
	val := big.NewInt(0xCAFEBABE)
	m.MStore(0, val)
	result := m.MLoad(0)
	if result.Cmp(val) != 0 {
		t.Fatalf("MLoad/MStore round-trip failed: %s != %s", result, val)
	}
}

func TestMemory_MStore8(t *testing.T) {
	m := NewMemory()
	m.SetByte(10, 0xAB)
	if m.GetByte(10) != 0xAB {
		t.Fatalf("expected 0xAB at offset 10")
	}
}

func TestMemory_MaxSizeLimit(t *testing.T) {
	m := NewMemory()
	_, err := m.Expand(DefaultMaxMemorySize + 1)
	if err != ErrMemoryLimit {
		t.Fatalf("expected memory limit error, got %v", err)
	}
}

func TestMemory_ReadEmptyLength(t *testing.T) {
	m := NewMemory()
	result := m.Read(0, 0)
	if result != nil {
		t.Fatalf("expected nil for zero-length read, got %v", result)
	}
}

// --- Gas Meter Tests ---

func TestGasMeter_Consume(t *testing.T) {
	g := NewGasMeter(1000)
	if err := g.Consume(500); err != nil {
		t.Fatal(err)
	}
	if g.Used() != 500 {
		t.Fatalf("expected 500 used, got %d", g.Used())
	}
	if g.Remaining() != 500 {
		t.Fatalf("expected 500 remaining, got %d", g.Remaining())
	}
}

func TestGasMeter_OutOfGas(t *testing.T) {
	g := NewGasMeter(100)
	if err := g.Consume(101); err != ErrOutOfGas {
		t.Fatalf("expected out of gas, got %v", err)
	}
	// After OOG, used should be at limit
	if g.Used() != 100 {
		t.Fatalf("expected used at limit (100), got %d", g.Used())
	}
	if g.Remaining() != 0 {
		t.Fatalf("expected 0 remaining, got %d", g.Remaining())
	}
}

func TestGasMeter_ExactLimit(t *testing.T) {
	g := NewGasMeter(100)
	if err := g.Consume(100); err != nil {
		t.Fatalf("consuming exact limit should succeed, got %v", err)
	}
	if err := g.Consume(1); err != ErrOutOfGas {
		t.Fatalf("expected out of gas after exact limit")
	}
}

func TestGasMeter_MultipleConsume(t *testing.T) {
	g := NewGasMeter(100)
	g.Consume(30)
	g.Consume(30)
	g.Consume(30)
	if g.Remaining() != 10 {
		t.Fatalf("expected 10 remaining, got %d", g.Remaining())
	}
	if err := g.Consume(11); err != ErrOutOfGas {
		t.Fatalf("expected out of gas")
	}
}

// --- Opcode Table Tests ---

func TestOpcodeTable_ADD(t *testing.T) {
	info := OpcodeTable[ADD]
	if info == nil {
		t.Fatal("ADD should be registered")
	}
	if info.Mnemonic != "ADD" {
		t.Fatalf("expected ADD, got %s", info.Mnemonic)
	}
	if info.StackInputs != 2 || info.StackOutputs != 1 {
		t.Fatalf("ADD: expected 2 in / 1 out, got %d/%d", info.StackInputs, info.StackOutputs)
	}
	if info.GasCost != 3 {
		t.Fatalf("ADD: expected gas 3, got %d", info.GasCost)
	}
}

func TestOpcodeTable_PUSH32(t *testing.T) {
	info := OpcodeTable[PUSH32]
	if info == nil {
		t.Fatal("PUSH32 should be registered")
	}
	if info.Mnemonic != "PUSH32" {
		t.Fatalf("expected PUSH32, got %s", info.Mnemonic)
	}
}

func TestOpcodeTable_SSTORE_IsStateModifier(t *testing.T) {
	info := OpcodeTable[SSTORE]
	if !info.IsStateModifier {
		t.Fatal("SSTORE should be a state modifier")
	}
}

func TestOpcodeTable_SLOAD_NotStateModifier(t *testing.T) {
	info := OpcodeTable[SLOAD]
	if info.IsStateModifier {
		t.Fatal("SLOAD should not be a state modifier")
	}
}

func TestIsValidOpcode(t *testing.T) {
	if !IsValidOpcode(ADD) {
		t.Fatal("ADD should be valid")
	}
	if IsValidOpcode(0x0C) {
		t.Fatal("0x0C should be invalid")
	}
}

func TestIsAgentOpcode(t *testing.T) {
	if !IsAgentOpcode(QUERY_FACT) {
		t.Fatal("QUERY_FACT should be agent opcode")
	}
	if !IsAgentOpcode(AGENT_DID) {
		t.Fatal("AGENT_DID should be agent opcode")
	}
	if !IsAgentOpcode(SELF_BALANCE_OP) {
		t.Fatal("SELF_BALANCE should be agent opcode")
	}
	if IsAgentOpcode(ADD) {
		t.Fatal("ADD should not be agent opcode")
	}
}

func TestGetMnemonic(t *testing.T) {
	if GetMnemonic(STOP) != "STOP" {
		t.Fatalf("expected STOP, got %s", GetMnemonic(STOP))
	}
	if GetMnemonic(0x0C) != "UNDEFINED" {
		t.Fatalf("expected UNDEFINED for invalid opcode")
	}
}

func TestOpcodeTable_AllLOGs(t *testing.T) {
	for i := byte(0); i <= 4; i++ {
		info := OpcodeTable[LOG0+i]
		if info == nil {
			t.Fatalf("LOG%d should be registered", i)
		}
		if !info.IsStateModifier {
			t.Fatalf("LOG%d should be state modifier", i)
		}
		expectedGas := uint64(375 + 375*int(i))
		if info.GasCost != expectedGas {
			t.Fatalf("LOG%d: expected gas %d, got %d", i, expectedGas, info.GasCost)
		}
	}
}

func TestOpcodeTable_AllDUPs(t *testing.T) {
	for i := byte(0); i < 16; i++ {
		info := OpcodeTable[DUP1+i]
		if info == nil {
			t.Fatalf("DUP%d should be registered", i+1)
		}
	}
}

func TestOpcodeTable_AllSWAPs(t *testing.T) {
	for i := byte(0); i < 16; i++ {
		info := OpcodeTable[SWAP1+i]
		if info == nil {
			t.Fatalf("SWAP%d should be registered", i+1)
		}
	}
}

func TestOpcodeTable_KnowledgeOpcodes(t *testing.T) {
	cases := []struct {
		op       byte
		mnemonic string
		stateMod bool
	}{
		{QUERY_FACT, "QUERY_FACT", false},
		{QUERY_DOMAIN, "QUERY_DOMAIN", false},
		{SUBMIT_CLAIM, "SUBMIT_CLAIM", true},
		{COMMIT_VOTE, "COMMIT_VOTE", true},
		{REVEAL_VOTE, "REVEAL_VOTE", true},
		{INITIATE_CHALLENGE, "INITIATE_CHALLENGE", true},
	}
	for _, tc := range cases {
		info := OpcodeTable[tc.op]
		if info == nil {
			t.Fatalf("%s should be registered", tc.mnemonic)
		}
		if info.Mnemonic != tc.mnemonic {
			t.Fatalf("expected %s, got %s", tc.mnemonic, info.Mnemonic)
		}
		if info.IsStateModifier != tc.stateMod {
			t.Fatalf("%s: stateMod expected %v", tc.mnemonic, tc.stateMod)
		}
	}
}
