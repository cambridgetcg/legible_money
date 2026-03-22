package keeper_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/legible-money/legible/x/bvm/types"
	"github.com/legible-money/legible/x/bvm/vm"
)

// ---------- Mock KnowledgeKeeper ----------

type mockKnowledgeKeeper struct {
	facts map[string]*mockFact
}

type mockFact struct {
	id         string
	confidence uint64
	domain     string
}

func (f *mockFact) GetConfidence() uint64 { return f.confidence }
func (f *mockFact) GetDomain() string     { return f.domain }
func (f *mockFact) GetId() string         { return f.id }

func newMockKnowledgeKeeper() *mockKnowledgeKeeper {
	return &mockKnowledgeKeeper{facts: make(map[string]*mockFact)}
}

func (m *mockKnowledgeKeeper) GetFact(_ context.Context, factId string) (interface{}, bool) {
	f, ok := m.facts[factId]
	if !ok {
		return nil, false
	}
	return f, true
}

func (m *mockKnowledgeKeeper) GetFactsByDomain(_ context.Context, domain string) []interface{} {
	var result []interface{}
	for _, f := range m.facts {
		if f.domain == domain {
			result = append(result, f)
		}
	}
	return result
}

func (m *mockKnowledgeKeeper) SubmitClaimFromBVM(_ context.Context, _, _, _, _, _ string, _ uint64) (string, error) {
	return "mock-claim-id", nil
}

func (m *mockKnowledgeKeeper) CommitVoteFromBVM(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockKnowledgeKeeper) RevealVoteFromBVM(_ context.Context, _, _, _ string, _ uint64, _ string) error {
	return nil
}

func (m *mockKnowledgeKeeper) ChallengeFactFromBVM(_ context.Context, _, _, _ string, _ uint64) (string, error) {
	return "mock-round-id", nil
}

var _ types.KnowledgeKeeper = (*mockKnowledgeKeeper)(nil)

// ---------- Mock StakingKeeper ----------

type mockStakingKeeper struct {
	stakes map[string]uint64
}

func newMockStakingKeeper() *mockStakingKeeper {
	return &mockStakingKeeper{stakes: make(map[string]uint64)}
}

func (m *mockStakingKeeper) GetValidatorStake(_ context.Context, validatorAddr string) (uint64, bool) {
	s, ok := m.stakes[validatorAddr]
	return s, ok
}

var _ types.StakingKeeper = (*mockStakingKeeper)(nil)

// ---------- Mock AuthKeeper with addressable accounts ----------

type mockAuthKeeperWithAddr struct {
	mockAuthKeeper
	// DID -> address mapping
	didToAddr map[string]string
}

type mockAccount struct {
	address string
	did     string
}

func (a *mockAccount) GetAddress() string { return a.address }

func newMockAuthKeeperWithAddr() *mockAuthKeeperWithAddr {
	return &mockAuthKeeperWithAddr{
		mockAuthKeeper: mockAuthKeeper{
			dids:     make(map[string]string),
			sessions: make(map[string]mockSession),
		},
		didToAddr: make(map[string]string),
	}
}

func (m *mockAuthKeeperWithAddr) GetAccountByDID(_ context.Context, did string) (interface{}, bool) {
	addr, ok := m.didToAddr[did]
	if !ok {
		return nil, false
	}
	return &mockAccount{address: addr, did: did}, true
}

func (m *mockAuthKeeperWithAddr) register(address, did string) {
	m.dids[address] = did
	m.didToAddr[did] = address
}

var _ types.AuthKeeper = (*mockAuthKeeperWithAddr)(nil)

// ---------- VerifyIdentity Tests ----------

func TestAgentHost_VerifyIdentity_WithAuthKeeper(t *testing.T) {
	_, ctx, _ := setupKeeper(t)

	stateDB := vm.NewMemoryStateDB()
	agent := vm.NewAgentHostWithSchedules(ctx, stateDB, nil)

	// Without auth keeper — should fall back to contract state (which is empty = false)
	targetDID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	if agent.VerifyIdentity(targetDID) {
		t.Fatal("expected VerifyIdentity to return false without auth keeper and empty state")
	}

	// Wire auth keeper with a registered DID
	ak := newMockAuthKeeperWithAddr()
	// DID format for VerifyIdentity: "did:lgm:" + hex(targetDID)
	didStr := "did:lgm:0102030405060708090a0b0c0d0e0f10"
	ak.register(testAddr("target"), didStr)
	agent.SetAuthKeeper(ak)

	// Now VerifyIdentity should return true via auth keeper
	if !agent.VerifyIdentity(targetDID) {
		t.Fatal("expected VerifyIdentity to return true with registered DID in auth keeper")
	}

	// Unregistered DID should still return false
	unknownDID := []byte{0xff, 0xff, 0xff, 0xff}
	if agent.VerifyIdentity(unknownDID) {
		t.Fatal("expected VerifyIdentity to return false for unregistered DID")
	}
}

// ---------- CompareBelief Tests ----------

func TestAgentHost_CompareBelief_WithKnowledgeKeeper(t *testing.T) {
	_, ctx, _ := setupKeeper(t)

	stateDB := vm.NewMemoryStateDB()
	agent := vm.NewAgentHostWithSchedules(ctx, stateDB, nil)

	callerDID := "did:lgm:caller"
	factId := []byte{0xaa, 0xbb, 0xcc, 0xdd}

	// Set agent's belief to 600000 * 1e12 = 6e17
	agent.SetBelief(callerDID, factId, 600_000_000_000_000_000, nil)

	// Without knowledge keeper — defaults to 0.5e18 network confidence
	diff := agent.CompareBelief(callerDID, factId)
	expected := int64(600_000_000_000_000_000) - int64(500_000_000_000_000_000)
	if diff != expected {
		t.Fatalf("expected diff %d without keeper, got %d", expected, diff)
	}

	// Wire knowledge keeper with a fact at confidence 800,000 (80% on 1M scale)
	kk := newMockKnowledgeKeeper()
	factIdHex := "aabbccdd"
	kk.facts[factIdHex] = &mockFact{
		id:         factIdHex,
		confidence: 800_000,
		domain:     "science",
	}
	agent.SetKnowledgeKeeper(kk)

	// Network confidence = 800,000 * 1e12 = 8e17
	diff = agent.CompareBelief(callerDID, factId)
	expected = int64(600_000_000_000_000_000) - int64(800_000_000_000_000_000)
	if diff != expected {
		t.Fatalf("expected diff %d with real keeper, got %d", expected, diff)
	}

	// Fact not found — should fall back to default 0.5e18
	unknownFact := []byte{0x11, 0x22}
	agent.SetBelief(callerDID, unknownFact, 500_000_000_000_000_000, nil)
	diff = agent.CompareBelief(callerDID, unknownFact)
	if diff != 0 {
		t.Fatalf("expected diff 0 for unknown fact (agent == default), got %d", diff)
	}
}

// ---------- SelfBalance Tests ----------

func TestTreasuryHost_SelfBalance_WithBankKeeper(t *testing.T) {
	_, ctx, bk := setupKeeper(t)

	stateDB := vm.NewMemoryStateDB()
	treasury := vm.NewTreasuryHost(stateDB)

	callerDID := "did:lgm:agent1"

	// Without querier — returns 0 (empty contract state)
	bal := treasury.SelfBalance(callerDID)
	if bal.Sign() != 0 {
		t.Fatalf("expected zero balance without querier, got %s", bal)
	}

	// Wire a mock balance querier
	callerAddr := testAddr("agent001")
	bk.balances[callerAddr] = map[string]int64{"ulgm": 42_000_000}

	ak := newMockAuthKeeperWithAddr()
	ak.register(callerAddr, callerDID)

	// Create a VMBalanceQuerier mock that uses our mock bank+auth
	bq := &testBalanceQuerier{ak: ak, bk: bk, ctx: ctx}
	treasury.SetBalanceQuerier(bq)

	bal = treasury.SelfBalance(callerDID)
	if bal.Int64() != 42_000_000 {
		t.Fatalf("expected balance 42000000, got %s", bal)
	}

	// Unknown DID — querier returns nil, falls back to contract state (0)
	bal = treasury.SelfBalance("did:lgm:unknown")
	if bal.Sign() != 0 {
		t.Fatalf("expected zero for unknown DID, got %s", bal)
	}
}

// ---------- SelfStake Tests ----------

func TestTreasuryHost_SelfStake_WithStakingKeeper(t *testing.T) {
	_, ctx, _ := setupKeeper(t)

	stateDB := vm.NewMemoryStateDB()
	treasury := vm.NewTreasuryHost(stateDB)

	callerDID := "did:lgm:validator1"

	// Without querier — returns 0
	stake := treasury.SelfStake(callerDID)
	if stake.Sign() != 0 {
		t.Fatalf("expected zero stake without querier, got %s", stake)
	}

	// Wire a mock stake querier
	callerAddr := testAddr("val00001")
	ak := newMockAuthKeeperWithAddr()
	ak.register(callerAddr, callerDID)

	sk := newMockStakingKeeper()
	sk.stakes[callerAddr] = 1_000_000_000

	sq := &testStakeQuerier{ak: ak, sk: sk, ctx: ctx}
	treasury.SetStakeQuerier(sq)

	stake = treasury.SelfStake(callerDID)
	if stake.Uint64() != 1_000_000_000 {
		t.Fatalf("expected stake 1000000000, got %s", stake)
	}
}

// ---------- QueryTreasury Tests ----------

func TestTreasuryHost_QueryTreasury_WithKeepers(t *testing.T) {
	_, ctx, bk := setupKeeper(t)

	stateDB := vm.NewMemoryStateDB()
	treasury := vm.NewTreasuryHost(stateDB)

	callerDID := "did:lgm:agent2"
	callerAddr := testAddr("agent002")

	ak := newMockAuthKeeperWithAddr()
	ak.register(callerAddr, callerDID)

	bk.balances[callerAddr] = map[string]int64{"ulgm": 99_000}

	sk := newMockStakingKeeper()
	sk.stakes[callerAddr] = 500_000

	treasury.SetBalanceQuerier(&testBalanceQuerier{ak: ak, bk: bk, ctx: ctx})
	treasury.SetStakeQuerier(&testStakeQuerier{ak: ak, sk: sk, ctx: ctx})

	available, reserved, staked, delegated := treasury.QueryTreasury(callerDID)

	if available.Int64() != 99_000 {
		t.Fatalf("expected available 99000, got %s", available)
	}
	if reserved.Sign() != 0 {
		t.Fatalf("expected reserved 0, got %s", reserved)
	}
	if staked.Uint64() != 500_000 {
		t.Fatalf("expected staked 500000, got %s", staked)
	}
	if delegated.Sign() != 0 {
		t.Fatalf("expected delegated 0, got %s", delegated)
	}
}

// ---------- Nil keeper fallback tests ----------

func TestAgentHost_VerifyIdentity_NilKeeper_FallsBack(t *testing.T) {
	_, ctx, _ := setupKeeper(t)

	stateDB := vm.NewMemoryStateDB()
	agent := vm.NewAgentHostWithSchedules(ctx, stateDB, nil)

	// No auth keeper set — should use contract state (empty = false)
	if agent.VerifyIdentity([]byte{0x01, 0x02}) {
		t.Fatal("expected false from contract state fallback")
	}
}

func TestTreasuryHost_SelfBalance_NilQuerier_FallsBack(t *testing.T) {
	stateDB := vm.NewMemoryStateDB()
	treasury := vm.NewTreasuryHost(stateDB)

	bal := treasury.SelfBalance("did:lgm:nobody")
	if bal.Sign() != 0 {
		t.Fatalf("expected zero from contract state fallback, got %s", bal)
	}
}

// ---------- Test helpers: VMBalanceQuerier / VMStakeQuerier adapters ----------

type testBalanceQuerier struct {
	ak  *mockAuthKeeperWithAddr
	bk  *mockBankKeeper
	ctx context.Context
}

func (q *testBalanceQuerier) GetBalanceByDID(_ context.Context, did string) *big.Int {
	acct, ok := q.ak.GetAccountByDID(q.ctx, did)
	if !ok {
		return nil
	}
	type addressable interface {
		GetAddress() string
	}
	if aa, ok := acct.(addressable); ok {
		addr := aa.GetAddress()
		if bals, ok := q.bk.balances[addr]; ok {
			if amt, ok := bals["ulgm"]; ok {
				return big.NewInt(amt)
			}
		}
	}
	return big.NewInt(0)
}

type testStakeQuerier struct {
	ak  *mockAuthKeeperWithAddr
	sk  *mockStakingKeeper
	ctx context.Context
}

func (q *testStakeQuerier) GetStakeByDID(_ context.Context, did string) (uint64, bool) {
	acct, ok := q.ak.GetAccountByDID(q.ctx, did)
	if !ok {
		return 0, false
	}
	type addressable interface {
		GetAddress() string
	}
	if aa, ok := acct.(addressable); ok {
		return q.sk.GetValidatorStake(q.ctx, aa.GetAddress())
	}
	return 0, false
}
