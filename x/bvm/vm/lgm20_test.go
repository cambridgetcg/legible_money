package vm

import (
	"math/big"
	"testing"
)

// tokenTestHost implements HostFunctions (stubs) and TokenHost (via StubTokenHost)
// for LGM-20 opcode tests.
type tokenTestHost struct {
	StubTokenHost
}

// --- HostFunctions stubs (all no-op) ---

func (h *tokenTestHost) QueryFact([]byte) (bool, uint64, []byte)                          { return false, 0, nil }
func (h *tokenTestHost) QueryDomain([]byte, uint64) (uint64, [][]byte)                    { return 0, nil }
func (h *tokenTestHost) SubmitClaim(string, []byte, []byte, []byte, uint64) []byte         { return nil }
func (h *tokenTestHost) CommitVote(string, []byte, []byte, uint64) bool                    { return false }
func (h *tokenTestHost) RevealVote(string, []byte, *big.Int, *big.Int, uint64) bool        { return false }
func (h *tokenTestHost) InitiateChallenge(string, []byte, []byte, *big.Int, uint64) []byte { return nil }
func (h *tokenTestHost) AgentDID([]byte) []byte                                            { return nil }
func (h *tokenTestHost) AgentHome(string) []byte                                           { return nil }
func (h *tokenTestHost) VerifyIdentity([]byte) bool                                        { return false }
func (h *tokenTestHost) AttestIdentity(string, []byte, []byte, []byte) []byte              { return nil }
func (h *tokenTestHost) GetBelief(string, []byte) uint64                                   { return 0 }
func (h *tokenTestHost) SetBelief(string, []byte, uint64, []byte)                          {}
func (h *tokenTestHost) CompareBelief(string, []byte) int64                                { return 0 }
func (h *tokenTestHost) BeliefCount(string, []byte) uint64                                 { return 0 }
func (h *tokenTestHost) GetTrust(string, []byte) uint64                                    { return 0 }
func (h *tokenTestHost) SetTrust(string, []byte, uint64, []byte)                           {}
func (h *tokenTestHost) IsTrusted(string, []byte, uint64) bool                             { return false }
func (h *tokenTestHost) Delegate(string, []byte, []byte, []byte) []byte                    { return nil }
func (h *tokenTestHost) ScheduleSelf([]byte, []byte, uint64, uint64) []byte                { return nil }
func (h *tokenTestHost) CancelScheduleHost([]byte) bool                                    { return false }
func (h *tokenTestHost) ModifySchedule([]byte, uint64) bool                                { return false }
func (h *tokenTestHost) GetScheduleCount([]byte) uint64                                    { return 0 }
func (h *tokenTestHost) ListTrusted(string, uint64, uint64) (uint64, [][]byte)             { return 0, nil }
func (h *tokenTestHost) BlockAgent(string, []byte, []byte) bool                            { return false }
func (h *tokenTestHost) UnblockAgent(string, []byte) bool                                  { return false }
func (h *tokenTestHost) RevokeDelegation(string, []byte) bool                              { return false }
func (h *tokenTestHost) ClaimVesting(string, []byte) *big.Int                              { return nil }
func (h *tokenTestHost) QueryVesting(string, uint64, uint64) (uint64, [][]byte)            { return 0, nil }
func (h *tokenTestHost) StakeTokens(string, *big.Int, []byte) []byte                       { return nil }
func (h *tokenTestHost) UnstakeTokens(string, []byte) *big.Int                             { return nil }
func (h *tokenTestHost) QueryExpiring(string, []byte) (*big.Int, uint64)                   { return nil, 0 }
func (h *tokenTestHost) ExtendExpiring(string, []byte, uint64) bool                        { return false }
func (h *tokenTestHost) SetSpendLimit(string, []byte, []byte, uint64) bool                 { return false }
func (h *tokenTestHost) QueryTreasury(string) (*big.Int, *big.Int, *big.Int, *big.Int) {
	return nil, nil, nil, nil
}
func (h *tokenTestHost) QueryAlerts(string, uint64, uint64) (uint64, [][]byte)     { return 0, nil }
func (h *tokenTestHost) DismissAlert(string, []byte) bool                          { return false }
func (h *tokenTestHost) SetDefense(string, []byte, []byte, []byte) bool            { return false }
func (h *tokenTestHost) QueryDefenses(string) (uint64, [][]byte)                   { return 0, nil }
func (h *tokenTestHost) RegisterKey(string, []byte, []byte, []byte, []byte) bool   { return false }
func (h *tokenTestHost) RevokeKey(string, []byte) bool                             { return false }
func (h *tokenTestHost) QueryPermissions(string, []byte) *big.Int                  { return nil }
func (h *tokenTestHost) QuerySession(string) ([]byte, []byte, uint64, uint64)      { return nil, nil, 0, 0 }
func (h *tokenTestHost) SelfBalance(string) *big.Int                               { return nil }
func (h *tokenTestHost) SelfStake(string) *big.Int                                 { return nil }
func (h *tokenTestHost) SelfReputation(string) uint64                              { return 0 }
func (h *tokenTestHost) SelfAccuracy(string) uint64                                { return 0 }

// --- Helper ---

func runTokenOpcode(t *testing.T, bytecode []byte) *ExecutionResult {
	t.Helper()
	host := &tokenTestHost{}
	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:tokentest"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	return interp.Execute(bytecode, ctx, state, host)
}

// --- Tests ---

func TestLGM20_TokenBalance_Stub(t *testing.T) {
	// TOKEN_BALANCE: [tokenID_ptr, tokenID_len, addr_ptr, addr_len] → [balance]
	// Push in reverse: addr_len, addr_ptr, tokenID_len, tokenID_ptr
	bytecode := []byte{
		PUSH1, 0, // addr_len
		PUSH1, 0, // addr_ptr
		PUSH1, 0, // tokenID_len
		PUSH1, 0, // tokenID_ptr
		TOKEN_BALANCE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	result := runTokenOpcode(t, bytecode)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	balance := new(big.Int).SetBytes(result.ReturnData)
	if balance.Sign() != 0 {
		t.Fatalf("expected 0 balance from stub, got %s", balance)
	}
}

func TestLGM20_TokenTransfer_Stub(t *testing.T) {
	// TOKEN_TRANSFER: [tokenID_ptr, tokenID_len, to_ptr, to_len, amount] → [success]
	// Push in reverse: amount, to_len, to_ptr, tokenID_len, tokenID_ptr
	bytecode := []byte{
		PUSH1, 100, // amount
		PUSH1, 0,   // to_len
		PUSH1, 0,   // to_ptr
		PUSH1, 0,   // tokenID_len
		PUSH1, 0,   // tokenID_ptr
		TOKEN_TRANSFER,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	result := runTokenOpcode(t, bytecode)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	success := new(big.Int).SetBytes(result.ReturnData)
	if success.Sign() != 0 {
		t.Fatalf("expected 0 (false) from stub transfer, got %s", success)
	}
}

func TestLGM20_TokenMint_Stub(t *testing.T) {
	// TOKEN_MINT: [tokenID_ptr, tokenID_len, to_ptr, to_len, amount] → [success]
	bytecode := []byte{
		PUSH1, 50, // amount
		PUSH1, 0,  // to_len
		PUSH1, 0,  // to_ptr
		PUSH1, 0,  // tokenID_len
		PUSH1, 0,  // tokenID_ptr
		TOKEN_MINT,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	result := runTokenOpcode(t, bytecode)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	success := new(big.Int).SetBytes(result.ReturnData)
	if success.Sign() != 0 {
		t.Fatalf("expected 0 (false) from stub mint, got %s", success)
	}
}

func TestLGM20_TokenBurn_Stub(t *testing.T) {
	// TOKEN_BURN: [tokenID_ptr, tokenID_len, amount] → [success]
	bytecode := []byte{
		PUSH1, 25, // amount
		PUSH1, 0,  // tokenID_len
		PUSH1, 0,  // tokenID_ptr
		TOKEN_BURN,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	result := runTokenOpcode(t, bytecode)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	success := new(big.Int).SetBytes(result.ReturnData)
	if success.Sign() != 0 {
		t.Fatalf("expected 0 (false) from stub burn, got %s", success)
	}
}

func TestLGM20_TokenApprove_Stub(t *testing.T) {
	// TOKEN_APPROVE: [tokenID_ptr, tokenID_len, spender_ptr, spender_len, amount] → [success]
	bytecode := []byte{
		PUSH1, 200, // amount
		PUSH1, 0,   // spender_len
		PUSH1, 0,   // spender_ptr
		PUSH1, 0,   // tokenID_len
		PUSH1, 0,   // tokenID_ptr
		TOKEN_APPROVE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	result := runTokenOpcode(t, bytecode)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	success := new(big.Int).SetBytes(result.ReturnData)
	if success.Sign() != 0 {
		t.Fatalf("expected 0 (false) from stub approve, got %s", success)
	}
}

func TestLGM20_TokenAllowance_Stub(t *testing.T) {
	// TOKEN_ALLOWANCE: [tokenID_ptr, tokenID_len, spender_ptr, spender_len] → [allowance]
	bytecode := []byte{
		PUSH1, 0, // spender_len
		PUSH1, 0, // spender_ptr
		PUSH1, 0, // tokenID_len
		PUSH1, 0, // tokenID_ptr
		TOKEN_ALLOWANCE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	result := runTokenOpcode(t, bytecode)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	allowance := new(big.Int).SetBytes(result.ReturnData)
	if allowance.Sign() != 0 {
		t.Fatalf("expected 0 allowance from stub, got %s", allowance)
	}
}

func TestLGM20_TokenPause_Stub(t *testing.T) {
	// TOKEN_PAUSE: [tokenID_ptr, tokenID_len] → [success]
	bytecode := []byte{
		PUSH1, 0, // tokenID_len
		PUSH1, 0, // tokenID_ptr
		TOKEN_PAUSE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	result := runTokenOpcode(t, bytecode)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	success := new(big.Int).SetBytes(result.ReturnData)
	if success.Sign() != 0 {
		t.Fatalf("expected 0 (false) from stub pause, got %s", success)
	}
}

func TestLGM20_TokenUnpause_Stub(t *testing.T) {
	// TOKEN_UNPAUSE: [tokenID_ptr, tokenID_len] → [success]
	bytecode := []byte{
		PUSH1, 0, // tokenID_len
		PUSH1, 0, // tokenID_ptr
		TOKEN_UNPAUSE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	result := runTokenOpcode(t, bytecode)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	success := new(big.Int).SetBytes(result.ReturnData)
	if success.Sign() != 0 {
		t.Fatalf("expected 0 (false) from stub unpause, got %s", success)
	}
}

func TestLGM20_StaticCallRejectsTokenTransfer(t *testing.T) {
	// State-modifying TOKEN_TRANSFER should fail in static call mode.
	bytecode := []byte{
		PUSH1, 100,
		PUSH1, 0,
		PUSH1, 0,
		PUSH1, 0,
		PUSH1, 0,
		TOKEN_TRANSFER,
		STOP,
	}
	host := &tokenTestHost{}
	ctx := makeCtx(100000)
	ctx.Mode = ModeStaticCall
	ctx.CallerDID = "did:lgm:tokentest"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if result.Success {
		t.Fatal("expected failure for TOKEN_TRANSFER in static call")
	}
}

func TestLGM20_NoTokenHostGracefulDegradation(t *testing.T) {
	// A host that does NOT implement TokenHost should still succeed
	// with the default behavior (push 0).
	mk := newMockKnowledgeKeeper()
	host := newMockFullHost(mk)

	bytecode := []byte{
		PUSH1, 0,
		PUSH1, 0,
		PUSH1, 0,
		PUSH1, 0,
		TOKEN_BALANCE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:test"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected graceful degradation, got error: %v", result.Error)
	}
	balance := new(big.Int).SetBytes(result.ReturnData)
	if balance.Sign() != 0 {
		t.Fatalf("expected 0 from non-TokenHost, got %s", balance)
	}
}
