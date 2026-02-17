package vm

import (
	"context"
	"fmt"
	"math/big"
	"testing"
)

// mockFact implements the FactData interface for testing.
type mockFact struct {
	confidence uint64
	domain     string
}

func (f *mockFact) GetConfidence() uint64 { return f.confidence }
func (f *mockFact) GetDomain() string     { return f.domain }

// mockKnowledgeKeeper implements KnowledgeKeeper for testing.
type mockKnowledgeKeeper struct {
	facts  map[string]interface{}
	domain map[string][]interface{}
}

func newMockKnowledgeKeeper() *mockKnowledgeKeeper {
	return &mockKnowledgeKeeper{
		facts: map[string]interface{}{
			"fact1": &mockFact{confidence: 850000, domain: "general"},
		},
		domain: map[string][]interface{}{
			"mathematics": {&mockFact{confidence: 500000, domain: "mathematics"}, &mockFact{confidence: 600000, domain: "mathematics"}},
		},
	}
}

func (m *mockKnowledgeKeeper) GetFact(_ context.Context, factId string) (interface{}, bool) {
	f, ok := m.facts[factId]
	return f, ok
}

func (m *mockKnowledgeKeeper) GetFactsByDomain(_ context.Context, domain string) []interface{} {
	return m.domain[domain]
}

func (m *mockKnowledgeKeeper) SubmitClaimFromBVM(_ context.Context, submitterDID, domain, contentHash, evidenceHash, logicZone string, stakeAmount uint64) (string, error) {
	return fmt.Sprintf("claim-%s-%s", submitterDID, domain), nil
}

func (m *mockKnowledgeKeeper) CommitVoteFromBVM(_ context.Context, validatorDID, roundId, commitmentHash string) error {
	return nil
}

func (m *mockKnowledgeKeeper) RevealVoteFromBVM(_ context.Context, validatorDID, roundId, verdict string, confidence uint64, salt string) error {
	return nil
}

func (m *mockKnowledgeKeeper) ChallengeFactFromBVM(_ context.Context, challengerDID, factId, evidenceHash string, stakeAmount uint64) (string, error) {
	return fmt.Sprintf("round-%s-%s", challengerDID, factId), nil
}

// mockFullHost implements HostFunctions backed by KnowledgeHost for knowledge ops.
type mockFullHost struct {
	kh *KnowledgeHost
}

func newMockFullHost(mk *mockKnowledgeKeeper) *mockFullHost {
	return &mockFullHost{
		kh: NewKnowledgeHost(mk, context.Background()),
	}
}

func (h *mockFullHost) QueryFact(factId []byte) (bool, uint64, []byte) {
	return h.kh.QueryFact(factId)
}
func (h *mockFullHost) QueryDomain(domain []byte, limit uint64) (uint64, [][]byte) {
	return h.kh.QueryDomain(domain, limit)
}
func (h *mockFullHost) SubmitClaim(callerDID string, domain, dataHash, evidenceHash []byte, blockNumber uint64) []byte {
	return h.kh.SubmitClaim(callerDID, domain, dataHash, evidenceHash, blockNumber)
}
func (h *mockFullHost) CommitVote(callerDID string, claimId, voteHash []byte, blockNumber uint64) bool {
	return h.kh.CommitVote(callerDID, claimId, voteHash, blockNumber)
}
func (h *mockFullHost) RevealVote(callerDID string, claimId []byte, vote, salt *big.Int, blockNumber uint64) bool {
	return h.kh.RevealVote(callerDID, claimId, vote, salt, blockNumber)
}
func (h *mockFullHost) InitiateChallenge(callerDID string, factId, evidenceHash []byte, stake *big.Int, blockNumber uint64) []byte {
	return h.kh.InitiateChallenge(callerDID, factId, evidenceHash, stake, blockNumber)
}

// Stub implementations for non-knowledge host functions
func (h *mockFullHost) AgentDID([]byte) []byte                                 { return nil }
func (h *mockFullHost) AgentHome(string) []byte                                { return nil }
func (h *mockFullHost) VerifyIdentity([]byte) bool                             { return false }
func (h *mockFullHost) AttestIdentity(string, []byte, []byte, []byte) []byte   { return nil }
func (h *mockFullHost) GetBelief(string, []byte) uint64                        { return 0 }
func (h *mockFullHost) SetBelief(string, []byte, uint64, []byte)               {}
func (h *mockFullHost) CompareBelief(string, []byte) int64                     { return 0 }
func (h *mockFullHost) BeliefCount(string, []byte) uint64                      { return 0 }
func (h *mockFullHost) GetTrust(string, []byte) uint64                         { return 0 }
func (h *mockFullHost) SetTrust(string, []byte, uint64, []byte)                {}
func (h *mockFullHost) IsTrusted(string, []byte, uint64) bool                  { return false }
func (h *mockFullHost) Delegate(string, []byte, []byte, []byte) []byte         { return nil }
func (h *mockFullHost) ScheduleSelf([]byte, []byte, uint64, uint64) []byte     { return nil }
func (h *mockFullHost) CancelScheduleHost([]byte) bool                         { return false }
func (h *mockFullHost) ModifySchedule([]byte, uint64) bool                     { return false }
func (h *mockFullHost) GetScheduleCount([]byte) uint64                         { return 0 }
func (h *mockFullHost) ListTrusted(string, uint64, uint64) (uint64, [][]byte)  { return 0, nil }
func (h *mockFullHost) BlockAgent(string, []byte, []byte) bool                 { return false }
func (h *mockFullHost) UnblockAgent(string, []byte) bool                       { return false }
func (h *mockFullHost) RevokeDelegation(string, []byte) bool                   { return false }
func (h *mockFullHost) ClaimVesting(string, []byte) *big.Int                   { return nil }
func (h *mockFullHost) QueryVesting(string, uint64, uint64) (uint64, [][]byte) { return 0, nil }
func (h *mockFullHost) StakeTokens(string, *big.Int, []byte) []byte            { return nil }
func (h *mockFullHost) UnstakeTokens(string, []byte) *big.Int                  { return nil }
func (h *mockFullHost) QueryExpiring(string, []byte) (*big.Int, uint64)        { return nil, 0 }
func (h *mockFullHost) ExtendExpiring(string, []byte, uint64) bool             { return false }
func (h *mockFullHost) SetSpendLimit(string, []byte, []byte, uint64) bool      { return false }
func (h *mockFullHost) QueryTreasury(string) (*big.Int, *big.Int, *big.Int, *big.Int) {
	return nil, nil, nil, nil
}
func (h *mockFullHost) QueryAlerts(string, uint64, uint64) (uint64, [][]byte)   { return 0, nil }
func (h *mockFullHost) DismissAlert(string, []byte) bool                        { return false }
func (h *mockFullHost) SetDefense(string, []byte, []byte, []byte) bool          { return false }
func (h *mockFullHost) QueryDefenses(string) (uint64, [][]byte)                 { return 0, nil }
func (h *mockFullHost) RegisterKey(string, []byte, []byte, []byte, []byte) bool { return false }
func (h *mockFullHost) RevokeKey(string, []byte) bool                           { return false }
func (h *mockFullHost) QueryPermissions(string, []byte) *big.Int                { return nil }
func (h *mockFullHost) QuerySession(string) ([]byte, []byte, uint64, uint64)    { return nil, nil, 0, 0 }
func (h *mockFullHost) SelfBalance(string) *big.Int                             { return nil }
func (h *mockFullHost) SelfStake(string) *big.Int                               { return nil }
func (h *mockFullHost) SelfReputation(string) uint64                            { return 0 }
func (h *mockFullHost) SelfAccuracy(string) uint64                              { return 0 }

func TestKnowledge_QueryFact(t *testing.T) {
	// Set up mock with a fact keyed by hex of the 32-byte value we'll push
	factKey := WordToBytes32(big.NewInt(42))
	factIdHex := fmt.Sprintf("%x", factKey[:])

	mk := &mockKnowledgeKeeper{
		facts: map[string]interface{}{
			factIdHex: &mockFact{confidence: 850000, domain: "general"},
		},
		domain: map[string][]interface{}{},
	}
	host := newMockFullHost(mk)

	// QUERY_FACT opcode: push factId hash, execute QUERY_FACT
	// The opcode pops 1, pushes 3 (exists, confidence, domain)
	bytecode := make([]byte, 0, 64)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, factKey[:]...)
	bytecode = append(bytecode, QUERY_FACT)
	// Stack now: [exists, confidence, domain]
	// Store domain (TOS) at memory 64
	bytecode = append(bytecode, PUSH1, 64, MSTORE)
	// Store confidence at memory 32
	bytecode = append(bytecode, PUSH1, 32, MSTORE)
	// Store exists at memory 0
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	// Return 96 bytes
	bytecode = append(bytecode, PUSH1, 96, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.ReturnData) != 96 {
		t.Fatalf("expected 96 bytes, got %d", len(result.ReturnData))
	}
	// Check exists (first 32 bytes) = 1 (true)
	exists := new(big.Int).SetBytes(result.ReturnData[:32])
	if exists.Int64() != 1 {
		t.Fatalf("expected exists=1, got %s", exists)
	}
	// Check confidence (second 32 bytes) = 850000 * 1e12 = 850000000000000000
	confidence := new(big.Int).SetBytes(result.ReturnData[32:64])
	if confidence.Uint64() != 850000000000000000 {
		t.Fatalf("expected confidence=850000000000000000, got %s", confidence)
	}
}

func TestKnowledge_QueryFactNotFound(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	host := newMockFullHost(mk)

	// Query a fact that doesn't exist
	factIdBytes := WordToBytes32(big.NewInt(999))

	bytecode := make([]byte, 0, 64)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, factIdBytes[:]...)
	bytecode = append(bytecode, QUERY_FACT)
	// Pop domain and confidence, keep exists
	bytecode = append(bytecode, POP, POP)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	exists := new(big.Int).SetBytes(result.ReturnData)
	if exists.Sign() != 0 {
		t.Fatalf("expected exists=0, got %s", exists)
	}
}

func TestKnowledge_SubmitClaim(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	host := newMockFullHost(mk)

	// SUBMIT_CLAIM pops 3 (domain, dataHash, evidenceHash), pushes 1 (claimId)
	bytecode := make([]byte, 0, 128)
	// Push evidenceHash
	bytecode = append(bytecode, PUSH1, 0xEE)
	// Push dataHash
	bytecode = append(bytecode, PUSH1, 0xDD)
	// Push domain
	bytecode = append(bytecode, PUSH1, 0xAA)
	bytecode = append(bytecode, SUBMIT_CLAIM)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:test123"
	ctx.Capabilities = allCaps()
	ctx.BlockNumber = 100
	ctx.Capabilities = &SessionCapabilities{CanTransfer: true, CanStake: true, CanSubmitClaims: true, CanVote: true, CanManageTrust: true, CanManageIdentity: true, CanManageSchedules: true, CanManageDefenses: true, CanManageBeliefs: true}
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	claimId := new(big.Int).SetBytes(result.ReturnData)
	if claimId.Sign() == 0 {
		t.Fatal("expected non-zero claim ID")
	}
}

func TestKnowledge_StaticCallRejectsSubmitClaim(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	host := newMockFullHost(mk)

	bytecode := []byte{
		PUSH1, 0xEE,
		PUSH1, 0xDD,
		PUSH1, 0xAA,
		SUBMIT_CLAIM,
		STOP,
	}
	ctx := makeCtx(100000)
	ctx.Mode = ModeStaticCall
	ctx.CallerDID = "did:lgm:test123"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if result.Success {
		t.Fatal("expected failure for SUBMIT_CLAIM in static call")
	}
}

func TestKnowledge_CommitVote(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	host := newMockFullHost(mk)

	// COMMIT_VOTE pops 2 (claimId, voteHash), pushes 1 (success)
	bytecode := []byte{
		PUSH1, 0xBB, // voteHash
		PUSH1, 0xAA, // claimId
		COMMIT_VOTE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:validator1"
	ctx.Capabilities = &SessionCapabilities{CanTransfer: true, CanStake: true, CanSubmitClaims: true, CanVote: true, CanManageTrust: true, CanManageIdentity: true, CanManageSchedules: true, CanManageDefenses: true, CanManageBeliefs: true}
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected 1 (success), got %s", val)
	}
}

func TestKnowledge_RevealVote(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	host := newMockFullHost(mk)

	// REVEAL_VOTE pops 3 (claimId, vote, salt), pushes 1 (success)
	bytecode := []byte{
		PUSH1, 0x42, // salt
		PUSH1, 0x01, // vote
		PUSH1, 0xAA, // claimId
		REVEAL_VOTE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:validator1"
	ctx.Capabilities = &SessionCapabilities{CanTransfer: true, CanStake: true, CanSubmitClaims: true, CanVote: true, CanManageTrust: true, CanManageIdentity: true, CanManageSchedules: true, CanManageDefenses: true, CanManageBeliefs: true}
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected 1 (success), got %s", val)
	}
}

func TestKnowledge_InitiateChallenge(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	host := newMockFullHost(mk)

	// INITIATE_CHALLENGE pops 3 (factId, evidenceHash, stake), pushes 1 (challengeId)
	bytecode := make([]byte, 0, 32)
	bytecode = append(bytecode, PUSH1, 100)  // stake
	bytecode = append(bytecode, PUSH1, 0xEE) // evidenceHash
	bytecode = append(bytecode, PUSH1, 0xAA) // factId
	bytecode = append(bytecode, INITIATE_CHALLENGE)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:challenger1"
	ctx.Capabilities = allCaps()
	ctx.BlockNumber = 200
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	challengeId := new(big.Int).SetBytes(result.ReturnData)
	if challengeId.Sign() == 0 {
		t.Fatal("expected non-zero challenge ID")
	}
}
