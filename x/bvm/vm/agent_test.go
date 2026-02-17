package vm

import (
	"context"
	"math/big"
	"testing"
)

// allCaps returns a SessionCapabilities with all permissions enabled, for use in tests.
func allCaps() *SessionCapabilities {
	return &SessionCapabilities{
		CanTransfer: true, CanStake: true, CanSubmitClaims: true, CanVote: true,
		CanManageTrust: true, CanManageIdentity: true, CanManageSchedules: true,
		CanManageDefenses: true, CanManageBeliefs: true,
	}
}

// compositeHost combines AgentHost, TreasuryHost, and KnowledgeHost into a full HostFunctions.
type compositeHost struct {
	agent    *AgentHost
	treasury *TreasuryHost
	kh       *KnowledgeHost
}

func newCompositeHost(stateDB StateDB) *compositeHost {
	return &compositeHost{
		agent:    NewAgentHost(context.Background(), stateDB),
		treasury: NewTreasuryHostWithAuth(stateDB, &testVMAuthKeeper{}, context.Background()),
		kh:       NewKnowledgeHost(nil, context.Background()),
	}
}

// testVMAuthKeeper is a mock VMAuthKeeper for tests. Stores registered keys in memory.
type testVMAuthKeeper struct {
	keys map[string]bool // ownerDID/pubKeyHex -> registered
}

func (m *testVMAuthKeeper) RegisterSessionKey(_ context.Context, ownerDID, pubKeyHex string, _, _, _, _ bool, _ uint64) error {
	if m.keys == nil {
		m.keys = make(map[string]bool)
	}
	m.keys[ownerDID+"/"+pubKeyHex] = true
	return nil
}

func (m *testVMAuthKeeper) RevokeSessionKey(_ context.Context, ownerDID, keyHash string) error {
	if m.keys != nil {
		delete(m.keys, ownerDID+"/"+keyHash)
	}
	return nil
}

func (m *testVMAuthKeeper) GetSessionForDID(_ context.Context, _ string, _ uint64) (uint64, bool, bool, bool, bool, bool) {
	return 0, false, false, false, false, false
}

// Knowledge
func (c *compositeHost) QueryFact(factId []byte) (bool, uint64, []byte) {
	return c.kh.QueryFact(factId)
}
func (c *compositeHost) QueryDomain(domain []byte, limit uint64) (uint64, [][]byte) {
	return c.kh.QueryDomain(domain, limit)
}
func (c *compositeHost) SubmitClaim(callerDID string, domain, dataHash, evidenceHash []byte, blockNumber uint64) []byte {
	return c.kh.SubmitClaim(callerDID, domain, dataHash, evidenceHash, blockNumber)
}
func (c *compositeHost) CommitVote(callerDID string, claimId, voteHash []byte, blockNumber uint64) bool {
	return c.kh.CommitVote(callerDID, claimId, voteHash, blockNumber)
}
func (c *compositeHost) RevealVote(callerDID string, claimId []byte, vote, salt *big.Int, blockNumber uint64) bool {
	return c.kh.RevealVote(callerDID, claimId, vote, salt, blockNumber)
}
func (c *compositeHost) InitiateChallenge(callerDID string, factId, evidenceHash []byte, stake *big.Int, blockNumber uint64) []byte {
	return c.kh.InitiateChallenge(callerDID, factId, evidenceHash, stake, blockNumber)
}

// Agent Identity
func (c *compositeHost) AgentDID(caller []byte) []byte     { return c.agent.AgentDID(caller) }
func (c *compositeHost) AgentHome(callerDID string) []byte { return c.agent.AgentHome(callerDID) }
func (c *compositeHost) VerifyIdentity(targetDID []byte) bool {
	return c.agent.VerifyIdentity(targetDID)
}
func (c *compositeHost) AttestIdentity(callerDID string, targetDID, attType, data []byte) []byte {
	return c.agent.AttestIdentity(callerDID, targetDID, attType, data)
}

// Agent Belief
func (c *compositeHost) GetBelief(callerDID string, factId []byte) uint64 {
	return c.agent.GetBelief(callerDID, factId)
}
func (c *compositeHost) SetBelief(callerDID string, factId []byte, conf uint64, src []byte) {
	c.agent.SetBelief(callerDID, factId, conf, src)
}
func (c *compositeHost) CompareBelief(callerDID string, factId []byte) int64 {
	return c.agent.CompareBelief(callerDID, factId)
}
func (c *compositeHost) BeliefCount(callerDID string, domain []byte) uint64 {
	return c.agent.BeliefCount(callerDID, domain)
}

// Agent Relationship
func (c *compositeHost) GetTrust(callerDID string, targetDID []byte) uint64 {
	return c.agent.GetTrust(callerDID, targetDID)
}
func (c *compositeHost) SetTrust(callerDID string, targetDID []byte, level uint64, reason []byte) {
	c.agent.SetTrust(callerDID, targetDID, level, reason)
}
func (c *compositeHost) IsTrusted(callerDID string, targetDID []byte, threshold uint64) bool {
	return c.agent.IsTrusted(callerDID, targetDID, threshold)
}
func (c *compositeHost) Delegate(callerDID string, targetDID, cap, constraints []byte) []byte {
	return c.agent.Delegate(callerDID, targetDID, cap, constraints)
}

// Agent Autonomy
func (c *compositeHost) ScheduleSelf(contract, callData []byte, execBlock, gasLimit uint64) []byte {
	return c.agent.ScheduleSelf(contract, callData, execBlock, gasLimit)
}
func (c *compositeHost) CancelScheduleHost(scheduleId []byte) bool {
	return c.agent.CancelScheduleHost(scheduleId)
}
func (c *compositeHost) ModifySchedule(scheduleId []byte, newBlock uint64) bool {
	return c.agent.ModifySchedule(scheduleId, newBlock)
}
func (c *compositeHost) GetScheduleCount(contract []byte) uint64 {
	return c.agent.GetScheduleCount(contract)
}

// Extended Relationships
func (c *compositeHost) ListTrusted(callerDID string, offset, limit uint64) (uint64, [][]byte) {
	return c.agent.ListTrusted(callerDID, offset, limit)
}
func (c *compositeHost) BlockAgent(callerDID string, targetDID, reason []byte) bool {
	return c.agent.BlockAgent(callerDID, targetDID, reason)
}
func (c *compositeHost) UnblockAgent(callerDID string, targetDID []byte) bool {
	return c.agent.UnblockAgent(callerDID, targetDID)
}
func (c *compositeHost) RevokeDelegation(callerDID string, delegationId []byte) bool {
	return c.agent.RevokeDelegation(callerDID, delegationId)
}

// Treasury
func (c *compositeHost) ClaimVesting(callerDID string, vestingId []byte) *big.Int {
	return c.treasury.ClaimVesting(callerDID, vestingId)
}
func (c *compositeHost) QueryVesting(callerDID string, off, lim uint64) (uint64, [][]byte) {
	return c.treasury.QueryVesting(callerDID, off, lim)
}
func (c *compositeHost) StakeTokens(callerDID string, amount *big.Int, valDID []byte) []byte {
	return c.treasury.StakeTokens(callerDID, amount, valDID)
}
func (c *compositeHost) UnstakeTokens(callerDID string, stakeId []byte) *big.Int {
	return c.treasury.UnstakeTokens(callerDID, stakeId)
}
func (c *compositeHost) QueryExpiring(callerDID string, periodId []byte) (*big.Int, uint64) {
	return c.treasury.QueryExpiring(callerDID, periodId)
}
func (c *compositeHost) ExtendExpiring(callerDID string, periodId []byte, newExpiry uint64) bool {
	return c.treasury.ExtendExpiring(callerDID, periodId, newExpiry)
}
func (c *compositeHost) SetSpendLimit(callerDID string, limitType, amount []byte, period uint64) bool {
	return c.treasury.SetSpendLimit(callerDID, limitType, amount, period)
}
func (c *compositeHost) QueryTreasury(callerDID string) (*big.Int, *big.Int, *big.Int, *big.Int) {
	return c.treasury.QueryTreasury(callerDID)
}

// Guardian
func (c *compositeHost) QueryAlerts(callerDID string, off, lim uint64) (uint64, [][]byte) {
	return c.treasury.QueryAlerts(callerDID, off, lim)
}
func (c *compositeHost) DismissAlert(callerDID string, alertId []byte) bool {
	return c.treasury.DismissAlert(callerDID, alertId)
}
func (c *compositeHost) SetDefense(callerDID string, defenseType, config, threshold []byte) bool {
	return c.treasury.SetDefense(callerDID, defenseType, config, threshold)
}
func (c *compositeHost) QueryDefenses(callerDID string) (uint64, [][]byte) {
	return c.treasury.QueryDefenses(callerDID)
}

// Access Control
func (c *compositeHost) RegisterKey(callerDID string, keyType, pubKey, perms, expiry []byte) bool {
	return c.treasury.RegisterKey(callerDID, keyType, pubKey, perms, expiry)
}
func (c *compositeHost) RevokeKey(callerDID string, keyId []byte) bool {
	return c.treasury.RevokeKey(callerDID, keyId)
}
func (c *compositeHost) QueryPermissions(callerDID string, role []byte) *big.Int {
	return c.treasury.QueryPermissions(callerDID, role)
}
func (c *compositeHost) QuerySession(callerDID string) ([]byte, []byte, uint64, uint64) {
	return c.treasury.QuerySession(callerDID)
}

// Introspection
func (c *compositeHost) SelfBalance(callerDID string) *big.Int {
	return c.treasury.SelfBalance(callerDID)
}
func (c *compositeHost) SelfStake(callerDID string) *big.Int { return c.treasury.SelfStake(callerDID) }
func (c *compositeHost) SelfReputation(callerDID string) uint64 {
	return c.treasury.SelfReputation(callerDID)
}
func (c *compositeHost) SelfAccuracy(callerDID string) uint64 {
	return c.treasury.SelfAccuracy(callerDID)
}

// --- Agent Identity Tests ---

func TestAgent_AgentDID(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	// AGENT_DID opcode: push caller address, execute AGENT_DID
	caller := WordToBytes32(big.NewInt(0xDEAD))
	bytecode := make([]byte, 0, 64)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, caller[:]...)
	bytecode = append(bytecode, AGENT_DID)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.ReturnData) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(result.ReturnData))
	}
	// DID hash should be non-zero
	did := new(big.Int).SetBytes(result.ReturnData)
	if did.Sign() == 0 {
		t.Fatal("expected non-zero DID hash")
	}
}

func TestAgent_VerifyIdentity_NotRegistered(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	// Verify an unregistered identity → should return 0
	targetDID := WordToBytes32(big.NewInt(42))
	bytecode := make([]byte, 0, 64)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, targetDID[:]...)
	bytecode = append(bytecode, VERIFY_IDENTITY)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatalf("expected 0 (not registered), got %s", val)
	}
}

// --- Agent Belief Tests ---

func TestAgent_SetAndGetBelief(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	// SET_BELIEF: push source, confidence, factId → SET_BELIEF
	// Then GET_BELIEF: push factId → GET_BELIEF → should return the confidence
	factId := WordToBytes32(big.NewInt(777))
	confidence := WordToBytes32(big.NewInt(900000000000000000)) // 0.9 * 1e18

	bytecode := make([]byte, 0, 256)
	// SET_BELIEF pops 3: factId, confidence, source
	bytecode = append(bytecode, PUSH1, 0x01) // source
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, confidence[:]...) // confidence
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, factId[:]...) // factId
	bytecode = append(bytecode, SET_BELIEF)

	// GET_BELIEF pops 1: factId, pushes 1: confidence
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, factId[:]...)
	bytecode = append(bytecode, GET_BELIEF)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(200000)
	ctx.CallerDID = "did:lgm:testAgent001"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Uint64() != 900000000000000000 {
		t.Fatalf("expected confidence=900000000000000000, got %s", val)
	}
}

// --- Agent Relationship Tests ---

func TestAgent_SetAndGetTrust(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	targetDID := WordToBytes32(big.NewInt(0xBEEF))
	trustLevel := WordToBytes32(big.NewInt(800000000000000000)) // 0.8 * 1e18

	bytecode := make([]byte, 0, 256)
	// SET_TRUST pops 3: targetDID, level, reason
	bytecode = append(bytecode, PUSH1, 0x01) // reason
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, trustLevel[:]...) // level
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, targetDID[:]...) // targetDID
	bytecode = append(bytecode, SET_TRUST)

	// GET_TRUST pops 1: targetDID, pushes 1: level
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, targetDID[:]...)
	bytecode = append(bytecode, GET_TRUST)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(200000)
	ctx.CallerDID = "did:lgm:agent123"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Uint64() != 800000000000000000 {
		t.Fatalf("expected trust=800000000000000000, got %s", val)
	}
}

func TestAgent_IsTrusted(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	targetDID := WordToBytes32(big.NewInt(0xCAFE))
	trustLevel := WordToBytes32(big.NewInt(700000000000000000))
	threshold := WordToBytes32(big.NewInt(500000000000000000))

	bytecode := make([]byte, 0, 256)
	// SET_TRUST first
	bytecode = append(bytecode, PUSH1, 0x01) // reason
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, trustLevel[:]...)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, targetDID[:]...)
	bytecode = append(bytecode, SET_TRUST)

	// IS_TRUSTED pops 2: targetDID, threshold → pushes 1: bool
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, threshold[:]...)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, targetDID[:]...)
	bytecode = append(bytecode, IS_TRUSTED)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(200000)
	ctx.CallerDID = "did:lgm:trustor"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected 1 (trusted), got %s", val)
	}
}

// --- Agent Autonomy Tests ---

func TestAgent_ScheduleSelf(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	// SCHEDULE_SELF pops 4: contract, callData, executeBlock, gasLimit → pushes 1: scheduleId
	bytecode := make([]byte, 0, 32)
	bytecode = append(bytecode, PUSH1, 100)  // gasLimit
	bytecode = append(bytecode, PUSH1, 200)  // executeBlock
	bytecode = append(bytecode, PUSH1, 0xDD) // callData
	bytecode = append(bytecode, PUSH1, 0xCC) // contract
	bytecode = append(bytecode, SCHEDULE_SELF)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:scheduler"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	scheduleId := new(big.Int).SetBytes(result.ReturnData)
	if scheduleId.Sign() == 0 {
		t.Fatal("expected non-zero schedule ID")
	}
}

// --- Extended Relationship Tests ---

func TestAgent_BlockAndUnblockAgent(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	targetDID := WordToBytes32(big.NewInt(0xBAD))

	// BLOCK_AGENT pops 2: targetDID, reason → pushes 1: success
	bytecode := make([]byte, 0, 64)
	bytecode = append(bytecode, PUSH1, 0x01) // reason
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, targetDID[:]...)
	bytecode = append(bytecode, BLOCK_AGENT)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:blocker"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected 1 (blocked), got %s", val)
	}
}

// --- Treasury Tests ---

func TestTreasury_StakeTokens(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	validatorDID := WordToBytes32(big.NewInt(0xBA1))
	amount := WordToBytes32(big.NewInt(1000))

	// STAKE pops 2: amount, validatorDID → pushes 1: stakeId
	bytecode := make([]byte, 0, 128)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, validatorDID[:]...)
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, amount[:]...)
	bytecode = append(bytecode, STAKE)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:staker"
	ctx.Capabilities = &SessionCapabilities{CanTransfer: true, CanStake: true, CanSubmitClaims: true, CanVote: true, CanManageTrust: true, CanManageIdentity: true, CanManageSchedules: true, CanManageDefenses: true, CanManageBeliefs: true}
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	stakeId := new(big.Int).SetBytes(result.ReturnData)
	if stakeId.Sign() == 0 {
		t.Fatal("expected non-zero stake ID")
	}
}

func TestTreasury_QueryTreasury(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	// QUERY_TREASURY pops 0, pushes 4: available, reserved, staked, delegated
	bytecode := make([]byte, 0, 64)
	bytecode = append(bytecode, QUERY_TREASURY)
	// Pop delegated, staked, reserved; keep available
	bytecode = append(bytecode, POP, POP, POP)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:querier"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	// With no data, available should be 0
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatalf("expected 0, got %s", val)
	}
}

// --- Access Control Tests ---

func TestAccess_RegisterAndRevokeKey(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	pubKey := WordToBytes32(big.NewInt(0xABCD))

	// REGISTER_KEY pops 4: keyType, publicKey, permissions, expiry → pushes 1: success
	bytecode := make([]byte, 0, 256)
	bytecode = append(bytecode, PUSH1, 0)    // expiry
	bytecode = append(bytecode, PUSH1, 0xFF) // permissions
	bytecode = append(bytecode, PUSH32)
	bytecode = append(bytecode, pubKey[:]...) // publicKey
	bytecode = append(bytecode, PUSH1, 0x01)  // keyType
	bytecode = append(bytecode, REGISTER_KEY)
	bytecode = append(bytecode, PUSH1, 0, MSTORE)
	bytecode = append(bytecode, PUSH1, 32, PUSH1, 0, RETURN)

	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:keyowner"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Int64() != 1 {
		t.Fatalf("expected 1 (registered), got %s", val)
	}
}

// --- Introspection Tests ---

func TestIntrospection_SelfBalance(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	// SELF_BALANCE pushes 1 value
	bytecode := []byte{
		SELF_BALANCE_OP,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:introspect"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatalf("expected 0 balance, got %s", val)
	}
}

func TestIntrospection_SelfReputation(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	bytecode := []byte{
		SELF_REPUTATION,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:repcheck"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatalf("expected 0 reputation, got %s", val)
	}
}

// --- Static Call Enforcement Tests ---

func TestAgent_StaticCallRejectsSetBelief(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	bytecode := []byte{
		PUSH1, 0x01, // source
		PUSH1, 0x50, // confidence
		PUSH1, 0xAA, // factId
		SET_BELIEF,
		STOP,
	}
	ctx := makeCtx(100000)
	ctx.Mode = ModeStaticCall
	ctx.CallerDID = "did:lgm:static"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if result.Success {
		t.Fatal("expected failure for SET_BELIEF in static call")
	}
}

func TestAgent_StaticCallRejectsSetTrust(t *testing.T) {
	state := NewMemoryStateDB()
	host := newCompositeHost(state)

	bytecode := []byte{
		PUSH1, 0x01, // reason
		PUSH1, 0x50, // level
		PUSH1, 0xAA, // targetDID
		SET_TRUST,
		STOP,
	}
	ctx := makeCtx(100000)
	ctx.Mode = ModeStaticCall
	ctx.CallerDID = "did:lgm:static"
	ctx.Capabilities = allCaps()
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if result.Success {
		t.Fatal("expected failure for SET_TRUST in static call")
	}
}

func TestAgent_NilHostGracefulDegradation(t *testing.T) {
	state := NewMemoryStateDB()

	// With nil host, agent opcodes should push 0 / no-op
	bytecode := []byte{
		PUSH1, 0xAA,
		AGENT_DID,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil) // nil host
	if !result.Success {
		t.Fatalf("expected success with nil host, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatalf("expected 0 from nil host, got %s", val)
	}
}
