package keeper_test

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/disputes/keeper"
	"github.com/legible-money/legible/x/disputes/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

// ---------- Mock Keepers ----------

type mockBankKeeper struct {
	balances map[string]sdkmath.Int // address -> balance
	modules  map[string]sdkmath.Int // module -> balance
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances: make(map[string]sdkmath.Int),
		modules:  make(map[string]sdkmath.Int),
	}
}

func (m *mockBankKeeper) SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	addr := senderAddr.String()
	balance, ok := m.balances[addr]
	if !ok {
		return fmt.Errorf("insufficient funds")
	}
	amount := amt.AmountOf("ulgm")
	if balance.LT(amount) {
		return fmt.Errorf("insufficient funds: %s < %s", balance, amount)
	}
	m.balances[addr] = balance.Sub(amount)

	modBal, ok := m.modules[recipientModule]
	if !ok {
		modBal = sdkmath.ZeroInt()
	}
	m.modules[recipientModule] = modBal.Add(amount)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	addr := recipientAddr.String()
	amount := amt.AmountOf("ulgm")
	modBal, ok := m.modules[senderModule]
	if !ok {
		modBal = sdkmath.ZeroInt()
	}
	if modBal.LT(amount) {
		return fmt.Errorf("insufficient module funds")
	}
	m.modules[senderModule] = modBal.Sub(amount)
	balance, ok := m.balances[addr]
	if !ok {
		balance = sdkmath.ZeroInt()
	}
	m.balances[addr] = balance.Add(amount)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error {
	amount := amt.AmountOf("ulgm")
	senderBal, ok := m.modules[senderModule]
	if !ok {
		senderBal = sdkmath.ZeroInt()
	}
	if senderBal.LT(amount) {
		return fmt.Errorf("insufficient module funds")
	}
	m.modules[senderModule] = senderBal.Sub(amount)
	recipBal, ok := m.modules[recipientModule]
	if !ok {
		recipBal = sdkmath.ZeroInt()
	}
	m.modules[recipientModule] = recipBal.Add(amount)
	return nil
}

func (m *mockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	amount := amt.AmountOf("ulgm")
	modBal, ok := m.modules[moduleName]
	if !ok {
		modBal = sdkmath.ZeroInt()
	}
	if modBal.LT(amount) {
		return fmt.Errorf("insufficient module funds to burn")
	}
	m.modules[moduleName] = modBal.Sub(amount)
	return nil
}

func (m *mockBankKeeper) fundAccount(addr string, amount sdkmath.Int) {
	m.balances[addr] = amount
}

func (m *mockBankKeeper) fundModule(moduleName string, amount sdkmath.Int) {
	m.modules[moduleName] = amount
}

type mockKnowledgeKeeper struct {
	facts  map[string]types.Fact
	claims map[string]types.Claim
}

func newMockKnowledgeKeeper() *mockKnowledgeKeeper {
	return &mockKnowledgeKeeper{
		facts:  make(map[string]types.Fact),
		claims: make(map[string]types.Claim),
	}
}

func (m *mockKnowledgeKeeper) GetFact(ctx sdk.Context, factID string) (types.Fact, bool) {
	f, ok := m.facts[factID]
	return f, ok
}

func (m *mockKnowledgeKeeper) GetClaim(ctx sdk.Context, claimID string) (types.Claim, bool) {
	c, ok := m.claims[claimID]
	return c, ok
}

func (m *mockKnowledgeKeeper) MarkFactChallenged(_ sdk.Context, factId string) error {
	f, ok := m.facts[factId]
	if !ok {
		return fmt.Errorf("fact not found: %s", factId)
	}
	if f.Status == "verified" || f.Status == "active" {
		f.Status = "challenged"
		m.facts[factId] = f
	}
	return nil
}

func (m *mockKnowledgeKeeper) ResolveFactDispute(_ sdk.Context, factId, outcome string, reductionBps uint64) (uint64, error) {
	f, ok := m.facts[factId]
	if !ok {
		return 0, fmt.Errorf("fact not found: %s", factId)
	}
	f.ChallengeCount++
	switch outcome {
	case "overturned":
		f.Confidence = f.Confidence * (1_000_000 - reductionBps) / 1_000_000
		if f.Confidence < 100_000 {
			f.Status = "revoked"
		} else {
			f.Status = "contested"
		}
	case "upheld":
		f.Confidence = f.Confidence * (1_000_000 + 110_000) / 1_000_000
		if f.Confidence > 1_000_000 {
			f.Confidence = 1_000_000
		}
		if f.Status == "contested" || f.Status == "challenged" {
			f.Status = "active"
		}
	case "inconclusive":
		if f.Status == "challenged" {
			f.Status = "contested"
		}
	}
	m.facts[factId] = f
	return f.Confidence, nil
}

func (m *mockKnowledgeKeeper) CascadeFactConfidence(_ sdk.Context, _ string, _ uint64) ([]string, error) {
	return nil, nil // No cascade in tests by default
}

type mockStakingKeeper struct {
	validators []types.ValidatorInfo
}

func newMockStakingKeeper() *mockStakingKeeper {
	return &mockStakingKeeper{}
}

func (m *mockStakingKeeper) GetActiveValidatorInfos(ctx context.Context) []types.ValidatorInfo {
	return m.validators
}

func (m *mockStakingKeeper) GetValidatorInfo(ctx context.Context, operatorAddr string) (types.ValidatorInfo, bool) {
	for _, v := range m.validators {
		if v.Address == operatorAddr {
			return v, true
		}
	}
	return types.ValidatorInfo{}, false
}

func (m *mockStakingKeeper) GetEffectiveStake(ctx context.Context, operatorAddr string) uint64 {
	for _, v := range m.validators {
		if v.Address == operatorAddr {
			return v.Stake
		}
	}
	return 0
}

// ---------- Test Setup ----------

func testAddr(i int) string {
	return sdk.AccAddress(fmt.Sprintf("addr%016d", i)).String()
}

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper, *mockKnowledgeKeeper, *mockStakingKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	err := stateStore.LoadLatestVersion()
	if err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	bk := newMockBankKeeper()
	kk := newMockKnowledgeKeeper()
	sk := newMockStakingKeeper()

	storeService := runtime.NewKVStoreService(storeKey)
	k := keeper.NewKeeper(cdc, storeService, bk, kk, sk, testAddr(999))

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	// Initialize genesis
	k.InitGenesis(ctx, *types.DefaultGenesis())

	return k, ctx, bk, kk, sk
}

// Helper to set up validators for arbiter selection
func setupValidators(sk *mockStakingKeeper, challenger, defender string) {
	sk.validators = []types.ValidatorInfo{
		{Address: testAddr(10), Stake: 5000, Tier: 2},
		{Address: testAddr(11), Stake: 4000, Tier: 2},
		{Address: testAddr(12), Stake: 3000, Tier: 1},
		{Address: testAddr(13), Stake: 2000, Tier: 1},
		{Address: testAddr(14), Stake: 1000, Tier: 1},
		{Address: challenger, Stake: 3000, Tier: 2},
		{Address: defender, Stake: 3000, Tier: 2},
	}
}

func setupVerifiedFact(kk *mockKnowledgeKeeper, factId, submitter string) {
	kk.facts[factId] = types.Fact{
		Id:         factId,
		ClaimText:  "Water boils at 100C",
		Domain:     "physics",
		Status:     "verified",
		Confidence: 800_000, // 80% — typical verified fact
		Submitter:  submitter,
	}
}

// ---------- Params Tests ----------

func TestSetGetParams(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	params := k.GetParams(ctx)
	if len(params.TierConfigs) != 3 {
		t.Fatalf("expected 3 tier configs, got %d", len(params.TierConfigs))
	}
	if params.MaxActiveDisputes != 3 {
		t.Fatalf("expected max active disputes 3, got %d", params.MaxActiveDisputes)
	}

	params.MaxActiveDisputes = 5
	k.SetParams(ctx, params)
	got := k.GetParams(ctx)
	if got.MaxActiveDisputes != 5 {
		t.Fatalf("expected 5, got %d", got.MaxActiveDisputes)
	}
}

// ---------- Dispute CRUD Tests ----------

func TestSetGetDispute(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:         "dispute-001",
		FactId:     "fact-001",
		Challenger: testAddr(1),
		Defender:   testAddr(2),
		Status:     string(types.DisputeStatusEvidence),
		Tier:       uint32(types.TierBasic),
		Bond:       "111000000",
	}
	k.SetDispute(ctx, dispute)

	got, found := k.GetDispute(ctx, "dispute-001")
	if !found {
		t.Fatal("dispute not found")
	}
	if got.FactId != "fact-001" || got.Challenger != testAddr(1) {
		t.Fatalf("dispute mismatch: %+v", got)
	}
}

func TestGetDisputeNotFound(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	_, found := k.GetDispute(ctx, "nonexistent")
	if found {
		t.Fatal("should not find nonexistent dispute")
	}
}

func TestDeleteDispute(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:     "dispute-del",
		FactId: "fact-del",
		Status: string(types.DisputeStatusEvidence),
	}
	k.SetDispute(ctx, dispute)
	k.DeleteDispute(ctx, "dispute-del")

	_, found := k.GetDispute(ctx, "dispute-del")
	if found {
		t.Fatal("dispute should be deleted")
	}
}

func TestIterateDisputes(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	for i := 0; i < 5; i++ {
		k.SetDispute(ctx, types.Dispute{
			Id:     fmt.Sprintf("dispute-%03d", i),
			FactId: "fact-001",
			Status: string(types.DisputeStatusEvidence),
		})
	}

	var count int
	k.IterateDisputes(ctx, func(d types.Dispute) bool {
		count++
		return false
	})
	if count != 5 {
		t.Fatalf("expected 5 disputes, got %d", count)
	}
}

func TestGetDisputesByFact(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	k.SetDispute(ctx, types.Dispute{Id: "d1", FactId: "fact-A", Status: string(types.DisputeStatusEvidence)})
	k.SetDispute(ctx, types.Dispute{Id: "d2", FactId: "fact-A", Status: string(types.DisputeStatusVoting)})
	k.SetDispute(ctx, types.Dispute{Id: "d3", FactId: "fact-B", Status: string(types.DisputeStatusEvidence)})

	disputes := k.GetDisputesByFact(ctx, "fact-A")
	if len(disputes) != 2 {
		t.Fatalf("expected 2 disputes for fact-A, got %d", len(disputes))
	}
}

func TestGetActiveDisputes(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	k.SetDispute(ctx, types.Dispute{Id: "active-1", FactId: "f1", Status: string(types.DisputeStatusEvidence)})
	k.SetDispute(ctx, types.Dispute{Id: "active-2", FactId: "f2", Status: string(types.DisputeStatusVoting)})
	k.SetDispute(ctx, types.Dispute{Id: "resolved", FactId: "f3", Status: string(types.DisputeStatusResolved)})
	k.SetDispute(ctx, types.Dispute{Id: "expired", FactId: "f4", Status: string(types.DisputeStatusExpired)})

	active := k.GetActiveDisputes(ctx)
	if len(active) != 2 {
		t.Fatalf("expected 2 active disputes, got %d", len(active))
	}
}

func TestGetActiveDisputeCountForFact(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	k.SetDispute(ctx, types.Dispute{Id: "d1", FactId: "f1", Status: string(types.DisputeStatusEvidence)})
	k.SetDispute(ctx, types.Dispute{Id: "d2", FactId: "f1", Status: string(types.DisputeStatusResolved)})
	k.SetDispute(ctx, types.Dispute{Id: "d3", FactId: "f1", Status: string(types.DisputeStatusVoting)})

	count := k.GetActiveDisputeCountForFact(ctx, "f1")
	if count != 2 {
		t.Fatalf("expected 2 active, got %d", count)
	}
}

// ---------- Evidence CRUD Tests ----------

func TestSetGetEvidence(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	evidence := types.DisputeEvidence{
		Id:        "ev-001",
		DisputeId: "dispute-001",
		Submitter: testAddr(1),
		Side:      "challenger",
		Content:   "Evidence content hash",
	}
	k.SetEvidence(ctx, evidence)

	got, found := k.GetEvidence(ctx, "dispute-001", "ev-001")
	if !found {
		t.Fatal("evidence not found")
	}
	if got.Content != "Evidence content hash" {
		t.Fatalf("evidence mismatch: %+v", got)
	}
}

func TestGetEvidenceByDispute(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	for i := 0; i < 3; i++ {
		k.SetEvidence(ctx, types.DisputeEvidence{
			Id:        fmt.Sprintf("ev-%d", i),
			DisputeId: "dispute-001",
			Submitter: testAddr(i),
		})
	}
	k.SetEvidence(ctx, types.DisputeEvidence{
		Id:        "ev-other",
		DisputeId: "dispute-002",
		Submitter: testAddr(0),
	})

	evidence := k.GetEvidenceByDispute(ctx, "dispute-001")
	if len(evidence) != 3 {
		t.Fatalf("expected 3 evidence items, got %d", len(evidence))
	}
}

// ---------- Vote CRUD Tests ----------

func TestSetGetVote(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	vote := types.DisputeVote{
		DisputeId: "dispute-001",
		Arbiter:   testAddr(10),
		Vote:      "uphold",
		Stake:     "5000",
		Rationale: "Fact is correct",
	}
	k.SetVote(ctx, vote)

	got, found := k.GetVote(ctx, "dispute-001", testAddr(10))
	if !found {
		t.Fatal("vote not found")
	}
	if got.Vote != "uphold" || got.Rationale != "Fact is correct" {
		t.Fatalf("vote mismatch: %+v", got)
	}
}

func TestGetVotes(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	for i := 0; i < 3; i++ {
		k.SetVote(ctx, types.DisputeVote{
			DisputeId: "dispute-001",
			Arbiter:   testAddr(i + 10),
			Vote:      "uphold",
			Stake:     "5000",
		})
	}

	votes := k.GetVotes(ctx, "dispute-001")
	if len(votes) != 3 {
		t.Fatalf("expected 3 votes, got %d", len(votes))
	}
}

// ---------- OpenDispute Tests ----------

func TestOpenDispute(t *testing.T) {
	k, ctx, _, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	factId := "fact-001"
	setupVerifiedFact(kk, factId, defender)
	setupValidators(sk, challenger, defender)

	dispute, err := k.OpenDispute(ctx, factId, challenger, "Incorrect claim", "111000000")
	if err != nil {
		t.Fatalf("failed to open dispute: %v", err)
	}
	if dispute.FactId != factId {
		t.Fatalf("expected fact_id %s, got %s", factId, dispute.FactId)
	}
	if dispute.Status != string(types.DisputeStatusEvidence) {
		t.Fatalf("expected evidence status, got %s", dispute.Status)
	}
	if dispute.Tier != uint32(types.TierBasic) {
		t.Fatalf("expected tier basic, got %d", dispute.Tier)
	}
	if len(dispute.Arbiters) == 0 {
		t.Fatal("expected arbiters to be selected")
	}
	// Verify evidence deadline set
	if dispute.EvidenceDeadline <= 100 {
		t.Fatal("expected evidence deadline > block height")
	}
}

func TestOpenDisputeFactNotFound(t *testing.T) {
	k, ctx, _, _, sk := setupKeeper(t)

	challenger := testAddr(1)
	setupValidators(sk, challenger, testAddr(2))

	_, err := k.OpenDispute(ctx, "nonexistent", challenger, "test", "111000000")
	if err == nil {
		t.Fatal("expected error for nonexistent fact")
	}
}

func TestOpenDisputeFactNotDisputable(t *testing.T) {
	k, ctx, _, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	kk.facts["fact-falsified"] = types.Fact{
		Id:     "fact-falsified",
		Status: "falsified",
	}
	setupValidators(sk, challenger, defender)

	_, err := k.OpenDispute(ctx, "fact-falsified", challenger, "test", "111000000")
	if err == nil {
		t.Fatal("expected error for non-disputable fact")
	}
}

func TestOpenDisputeInsufficientBond(t *testing.T) {
	k, ctx, _, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	setupVerifiedFact(kk, "fact-001", defender)
	setupValidators(sk, challenger, defender)

	_, err := k.OpenDispute(ctx, "fact-001", challenger, "test", "100") // too low
	if err == nil {
		t.Fatal("expected error for insufficient bond")
	}
}

func TestOpenDisputeMaxActive(t *testing.T) {
	k, ctx, _, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	factId := "fact-max"
	setupVerifiedFact(kk, factId, defender)
	setupValidators(sk, challenger, defender)

	// Fill max active disputes (3)
	for i := 0; i < 3; i++ {
		_, err := k.OpenDispute(ctx.WithBlockHeight(int64(100+i)), factId, testAddr(i+20), "reason", "111000000")
		if err != nil {
			t.Fatalf("failed to open dispute %d: %v", i, err)
		}
	}

	// Fourth should fail
	_, err := k.OpenDispute(ctx.WithBlockHeight(200), factId, challenger, "reason", "111000000")
	if err == nil {
		t.Fatal("expected error for max active disputes")
	}
}

// ---------- CastVote Tests ----------

func TestCastVote(t *testing.T) {
	k, ctx, _, _, sk := setupKeeper(t)

	arbiter1 := testAddr(10)
	arbiter2 := testAddr(11)
	sk.validators = []types.ValidatorInfo{
		{Address: arbiter1, Stake: 5000, Tier: 2},
		{Address: arbiter2, Stake: 4000, Tier: 2},
	}

	// Create a dispute in voting phase
	dispute := types.Dispute{
		Id:              "dispute-vote",
		FactId:          "fact-001",
		Status:          string(types.DisputeStatusVoting),
		Tier:            uint32(types.TierBasic),
		VotingDeadline:  200,
		Arbiters:        []string{arbiter1, arbiter2},
	}
	k.SetDispute(ctx, dispute)

	err := k.CastVote(ctx, "dispute-vote", arbiter1, "uphold", "Fact is correct")
	if err != nil {
		t.Fatalf("failed to cast vote: %v", err)
	}

	vote, found := k.GetVote(ctx, "dispute-vote", arbiter1)
	if !found {
		t.Fatal("vote not found")
	}
	if vote.Vote != "uphold" {
		t.Fatalf("expected uphold, got %s", vote.Vote)
	}
}

func TestCastVoteNotArbiter(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:             "dispute-v2",
		Status:         string(types.DisputeStatusVoting),
		VotingDeadline: 200,
		Arbiters:       []string{testAddr(10)},
	}
	k.SetDispute(ctx, dispute)

	err := k.CastVote(ctx, "dispute-v2", testAddr(99), "uphold", "test")
	if err == nil {
		t.Fatal("expected error for non-arbiter")
	}
}

func TestCastVoteAlreadyVoted(t *testing.T) {
	k, ctx, _, _, sk := setupKeeper(t)

	arbiter := testAddr(10)
	sk.validators = []types.ValidatorInfo{{Address: arbiter, Stake: 5000, Tier: 2}}

	dispute := types.Dispute{
		Id:             "dispute-dup",
		Status:         string(types.DisputeStatusVoting),
		VotingDeadline: 200,
		Arbiters:       []string{arbiter},
	}
	k.SetDispute(ctx, dispute)

	_ = k.CastVote(ctx, "dispute-dup", arbiter, "uphold", "test")
	err := k.CastVote(ctx, "dispute-dup", arbiter, "overturn", "changed mind")
	if err == nil {
		t.Fatal("expected error for duplicate vote")
	}
}

func TestCastVoteNotInVotingPhase(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:       "dispute-phase",
		Status:   string(types.DisputeStatusEvidence),
		Arbiters: []string{testAddr(10)},
	}
	k.SetDispute(ctx, dispute)

	err := k.CastVote(ctx, "dispute-phase", testAddr(10), "uphold", "test")
	if err == nil {
		t.Fatal("expected error for wrong phase")
	}
}

func TestCastVoteDeadlinePassed(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	arbiter := testAddr(10)
	dispute := types.Dispute{
		Id:             "dispute-late",
		Status:         string(types.DisputeStatusVoting),
		VotingDeadline: 50, // already passed (block height = 100)
		Arbiters:       []string{arbiter},
	}
	k.SetDispute(ctx, dispute)

	err := k.CastVote(ctx, "dispute-late", arbiter, "uphold", "test")
	if err == nil {
		t.Fatal("expected error for deadline passed")
	}
}

// ---------- TallyVotes Tests ----------

func TestTallyVotesUpheld(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:       "dispute-tally",
		Tier:     uint32(types.TierBasic),
		Arbiters: []string{testAddr(10), testAddr(11), testAddr(12)},
	}
	k.SetDispute(ctx, dispute)

	// All 3 vote uphold with high stake
	for _, addr := range dispute.Arbiters {
		k.SetVote(ctx, types.DisputeVote{
			DisputeId: "dispute-tally",
			Arbiter:   addr,
			Vote:      "uphold",
			Stake:     "5000",
		})
	}

	outcome, err := k.TallyVotes(ctx, dispute)
	if err != nil {
		t.Fatalf("tally failed: %v", err)
	}
	if outcome != types.OutcomeUpheld {
		t.Fatalf("expected upheld, got %s", outcome)
	}
}

func TestTallyVotesOverturned(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:       "dispute-overturn",
		Tier:     uint32(types.TierBasic),
		Arbiters: []string{testAddr(10), testAddr(11), testAddr(12)},
	}
	k.SetDispute(ctx, dispute)

	// All vote overturn
	for _, addr := range dispute.Arbiters {
		k.SetVote(ctx, types.DisputeVote{
			DisputeId: "dispute-overturn",
			Arbiter:   addr,
			Vote:      "overturn",
			Stake:     "5000",
		})
	}

	outcome, err := k.TallyVotes(ctx, dispute)
	if err != nil {
		t.Fatalf("tally failed: %v", err)
	}
	if outcome != types.OutcomeOverturned {
		t.Fatalf("expected overturned, got %s", outcome)
	}
}

func TestTallyVotesInconclusive(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:       "dispute-split",
		Tier:     uint32(types.TierBasic),
		Arbiters: []string{testAddr(10), testAddr(11), testAddr(12)},
	}
	k.SetDispute(ctx, dispute)

	// Split vote: 1 uphold, 1 overturn, 1 abstain (equal stake)
	k.SetVote(ctx, types.DisputeVote{DisputeId: "dispute-split", Arbiter: testAddr(10), Vote: "uphold", Stake: "5000"})
	k.SetVote(ctx, types.DisputeVote{DisputeId: "dispute-split", Arbiter: testAddr(11), Vote: "overturn", Stake: "5000"})
	k.SetVote(ctx, types.DisputeVote{DisputeId: "dispute-split", Arbiter: testAddr(12), Vote: "abstain", Stake: "5000"})

	outcome, err := k.TallyVotes(ctx, dispute)
	if err != nil {
		t.Fatalf("tally failed: %v", err)
	}
	if outcome != types.OutcomeInconclusive {
		t.Fatalf("expected inconclusive, got %s", outcome)
	}
}

func TestTallyVotesQuorumNotMet(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:       "dispute-quorum",
		Tier:     uint32(types.TierBasic),
		Arbiters: []string{testAddr(10), testAddr(11), testAddr(12)},
	}
	k.SetDispute(ctx, dispute)

	// Only 1 of 3 votes (33% < 66% quorum threshold)
	k.SetVote(ctx, types.DisputeVote{DisputeId: "dispute-quorum", Arbiter: testAddr(10), Vote: "uphold", Stake: "5000"})

	outcome, err := k.TallyVotes(ctx, dispute)
	if err != nil {
		t.Fatalf("tally failed: %v", err)
	}
	if outcome != types.OutcomeInconclusive {
		t.Fatalf("expected inconclusive (quorum not met), got %s", outcome)
	}
}

func TestTallyVotesStakeWeighted(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:       "dispute-weighted",
		Tier:     uint32(types.TierBasic),
		Arbiters: []string{testAddr(10), testAddr(11), testAddr(12)},
	}
	k.SetDispute(ctx, dispute)

	// 2 vote uphold with low stake, 1 votes overturn with high stake
	// uphold total: 2*1000 = 2000, overturn total: 8000
	// overturn ratio = 8000/10000 = 80% >= 77% majority
	k.SetVote(ctx, types.DisputeVote{DisputeId: "dispute-weighted", Arbiter: testAddr(10), Vote: "uphold", Stake: "1000"})
	k.SetVote(ctx, types.DisputeVote{DisputeId: "dispute-weighted", Arbiter: testAddr(11), Vote: "uphold", Stake: "1000"})
	k.SetVote(ctx, types.DisputeVote{DisputeId: "dispute-weighted", Arbiter: testAddr(12), Vote: "overturn", Stake: "8000"})

	outcome, err := k.TallyVotes(ctx, dispute)
	if err != nil {
		t.Fatalf("tally failed: %v", err)
	}
	if outcome != types.OutcomeOverturned {
		t.Fatalf("expected overturned (high-stake voter wins), got %s", outcome)
	}
}

// ---------- ResolveDispute Tests ----------

func TestResolveDisputeOverturned(t *testing.T) {
	k, ctx, bk, kk, _ := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	setupVerifiedFact(kk, "fact-res", defender)
	bk.fundModule(types.ModuleName, sdkmath.NewIntWithDecimal(1, 12))

	dispute := types.Dispute{
		Id:         "dispute-resolve-ot",
		FactId:     "fact-res",
		Challenger: challenger,
		Defender:   defender,
		Status:     string(types.DisputeStatusVoting),
		Tier:       uint32(types.TierBasic),
		Bond:       "111000000",
		Arbiters:   []string{testAddr(10), testAddr(11)},
	}
	k.SetDispute(ctx, dispute)

	err := k.ResolveDispute(ctx, dispute, types.OutcomeOverturned)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	resolved, _ := k.GetDispute(ctx, "dispute-resolve-ot")
	if resolved.Status != string(types.DisputeStatusResolved) {
		t.Fatalf("expected resolved status, got %s", resolved.Status)
	}
	if resolved.Outcome != string(types.OutcomeOverturned) {
		t.Fatalf("expected overturned outcome, got %s", resolved.Outcome)

	}

	// Fact status should be updated to contested (confidence drops but above retraction threshold)
	fact := kk.facts["fact-res"]
	if fact.Status != "contested" {
		t.Fatalf("expected fact status contested, got %s", fact.Status)
	}
	// Confidence should be reduced: 800000 * (1000000 - 550000) / 1000000 = 360000
	expectedConf := uint64(800_000) * (1_000_000 - 550_000) / 1_000_000
	if fact.Confidence != expectedConf {
		t.Fatalf("expected confidence %d, got %d", expectedConf, fact.Confidence)
	}
	if fact.ChallengeCount != 1 {
		t.Fatalf("expected challenge count 1, got %d", fact.ChallengeCount)
	}
}

func TestResolveDisputeUpheld(t *testing.T) {
	k, ctx, bk, _, _ := setupKeeper(t)

	bk.fundModule(types.ModuleName, sdkmath.NewIntWithDecimal(1, 12))

	dispute := types.Dispute{
		Id:         "dispute-resolve-up",
		FactId:     "fact-up",
		Challenger: testAddr(1),
		Defender:   testAddr(2),
		Status:     string(types.DisputeStatusVoting),
		Bond:       "111000000",
		Arbiters:   []string{testAddr(10)},
	}
	k.SetDispute(ctx, dispute)

	err := k.ResolveDispute(ctx, dispute, types.OutcomeUpheld)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	resolved, _ := k.GetDispute(ctx, "dispute-resolve-up")
	if resolved.Outcome != string(types.OutcomeUpheld) {
		t.Fatalf("expected upheld, got %s", resolved.Outcome)
	}
}

func TestResolveDisputeInconclusive(t *testing.T) {
	k, ctx, bk, _, _ := setupKeeper(t)

	bk.fundModule(types.ModuleName, sdkmath.NewIntWithDecimal(1, 12))

	dispute := types.Dispute{
		Id:         "dispute-resolve-inc",
		FactId:     "fact-inc",
		Challenger: testAddr(1),
		Status:     string(types.DisputeStatusVoting),
		Bond:       "111000000",
		Arbiters:   []string{testAddr(10)},
	}
	k.SetDispute(ctx, dispute)

	err := k.ResolveDispute(ctx, dispute, types.OutcomeInconclusive)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	resolved, _ := k.GetDispute(ctx, "dispute-resolve-inc")
	if resolved.Outcome != string(types.OutcomeInconclusive) {
		t.Fatalf("expected inconclusive, got %s", resolved.Outcome)
	}
}

func TestResolveDisputeAlreadyResolved(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:     "dispute-done",
		Status: string(types.DisputeStatusResolved),
	}
	k.SetDispute(ctx, dispute)

	err := k.ResolveDispute(ctx, dispute, types.OutcomeUpheld)
	if err == nil {
		t.Fatal("expected error for already-resolved dispute")
	}
}

// ---------- EscalateDispute Tests ----------

func TestEscalateDispute(t *testing.T) {
	k, ctx, bk, _, sk := setupKeeper(t)

	requester := testAddr(5)
	bk.fundAccount(requester, sdkmath.NewIntWithDecimal(1, 12))
	setupValidators(sk, testAddr(1), testAddr(2))

	dispute := types.Dispute{
		Id:         "dispute-esc",
		FactId:     "fact-esc",
		Challenger: testAddr(1),
		Defender:   testAddr(2),
		Status:     string(types.DisputeStatusEvidence),
		Tier:       uint32(types.TierBasic),
		Bond:       "111000000",
		CreatedAt:  50, // old enough for escalation (delay = 17136, but we'll lower it)
	}
	k.SetDispute(ctx, dispute)

	// Set escalation delay to 0 for test
	params := k.GetParams(ctx)
	params.EscalationDelay = 0
	k.SetParams(ctx, params)

	err := k.EscalateDispute(ctx, "dispute-esc", requester, "555000000")
	if err != nil {
		t.Fatalf("escalation failed: %v", err)
	}

	escalated, _ := k.GetDispute(ctx, "dispute-esc")
	if escalated.Tier != uint32(types.TierEscalated) {
		t.Fatalf("expected tier 2, got %d", escalated.Tier)
	}
	if escalated.Status != string(types.DisputeStatusEvidence) {
		t.Fatalf("expected evidence status after escalation, got %s", escalated.Status)
	}
	// Bond should be increased
	expectedBond := new(big.Int)
	expectedBond.SetString("111000000", 10)
	additional := new(big.Int)
	additional.SetString("555000000", 10)
	expectedBond.Add(expectedBond, additional)
	if escalated.Bond != expectedBond.String() {
		t.Fatalf("expected bond %s, got %s", expectedBond.String(), escalated.Bond)
	}
}

func TestEscalateDisputeMaxTier(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:     "dispute-max",
		Status: string(types.DisputeStatusVoting),
		Tier:   uint32(types.TierFinal),
	}
	k.SetDispute(ctx, dispute)

	err := k.EscalateDispute(ctx, "dispute-max", testAddr(5), "1000")
	if err == nil {
		t.Fatal("expected error for max tier escalation")
	}
}

func TestEscalateDisputeTooEarly(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:        "dispute-early",
		Status:    string(types.DisputeStatusEvidence),
		Tier:      uint32(types.TierBasic),
		CreatedAt: 99, // created 1 block ago, delay = 17136 blocks
	}
	k.SetDispute(ctx, dispute)

	err := k.EscalateDispute(ctx, "dispute-early", testAddr(5), "555000000")
	if err == nil {
		t.Fatal("expected error for too-early escalation")
	}
}

// ---------- TransitionToVoting Tests ----------

func TestTransitionToVoting(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:     "dispute-transition",
		Status: string(types.DisputeStatusEvidence),
	}
	k.SetDispute(ctx, dispute)

	err := k.TransitionToVoting(ctx, dispute)
	if err != nil {
		t.Fatalf("transition failed: %v", err)
	}

	got, _ := k.GetDispute(ctx, "dispute-transition")
	if got.Status != string(types.DisputeStatusVoting) {
		t.Fatalf("expected voting, got %s", got.Status)
	}
}

func TestTransitionToVotingWrongPhase(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:     "dispute-wrong",
		Status: string(types.DisputeStatusVoting),
	}
	k.SetDispute(ctx, dispute)

	err := k.TransitionToVoting(ctx, dispute)
	if err == nil {
		t.Fatal("expected error for wrong phase")
	}
}

// ---------- ProcessExpiredDisputes Tests ----------

func TestProcessExpiredDisputesEvidenceToVoting(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:               "dispute-exp-ev",
		FactId:           "f1",
		Status:           string(types.DisputeStatusEvidence),
		Tier:             uint32(types.TierBasic),
		EvidenceDeadline: 80, // already passed (height = 100)
		VotingDeadline:   200,
	}
	k.SetDispute(ctx, dispute)

	k.ProcessExpiredDisputes(ctx)

	got, _ := k.GetDispute(ctx, "dispute-exp-ev")
	if got.Status != string(types.DisputeStatusVoting) {
		t.Fatalf("expected voting after evidence expiry, got %s", got.Status)
	}
}

func TestProcessExpiredDisputesVotingToResolved(t *testing.T) {
	k, ctx, bk, _, _ := setupKeeper(t)

	bk.fundModule(types.ModuleName, sdkmath.NewIntWithDecimal(1, 12))

	dispute := types.Dispute{
		Id:             "dispute-exp-vote",
		FactId:         "f2",
		Challenger:     testAddr(1),
		Status:         string(types.DisputeStatusVoting),
		Tier:           uint32(types.TierBasic),
		VotingDeadline: 80, // already passed
		Bond:           "111000000",
		Arbiters:       []string{testAddr(10), testAddr(11), testAddr(12)},
	}
	k.SetDispute(ctx, dispute)

	// Add votes so tally works (all uphold)
	for _, addr := range dispute.Arbiters {
		k.SetVote(ctx, types.DisputeVote{
			DisputeId: "dispute-exp-vote",
			Arbiter:   addr,
			Vote:      "uphold",
			Stake:     "5000",
		})
	}

	k.ProcessExpiredDisputes(ctx)

	got, _ := k.GetDispute(ctx, "dispute-exp-vote")
	if got.Status != string(types.DisputeStatusResolved) {
		t.Fatalf("expected resolved after voting expiry, got %s", got.Status)
	}
	if got.Outcome != string(types.OutcomeUpheld) {
		t.Fatalf("expected upheld, got %s", got.Outcome)
	}
}

// ---------- SelectArbiters Tests ----------

func TestSelectArbiters(t *testing.T) {
	k, ctx, _, _, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	setupValidators(sk, challenger, defender)

	arbiters, err := k.SelectArbiters(ctx, 3, challenger, defender)
	if err != nil {
		t.Fatalf("select arbiters failed: %v", err)
	}
	if len(arbiters) != 3 {
		t.Fatalf("expected 3 arbiters, got %d", len(arbiters))
	}

	// Verify challenger and defender are excluded
	for _, a := range arbiters {
		if a == challenger || a == defender {
			t.Fatalf("arbiter should not be challenger or defender: %s", a)
		}
	}
}

func TestSelectArbitersExcludesLowTier(t *testing.T) {
	k, ctx, _, _, sk := setupKeeper(t)

	sk.validators = []types.ValidatorInfo{
		{Address: testAddr(10), Stake: 5000, Tier: 0}, // apprentice — excluded
		{Address: testAddr(11), Stake: 4000, Tier: 1}, // verified — eligible
		{Address: testAddr(12), Stake: 3000, Tier: 2}, // bonded — eligible
	}

	arbiters, err := k.SelectArbiters(ctx, 3, testAddr(1), testAddr(2))
	if err != nil {
		t.Fatalf("select arbiters failed: %v", err)
	}
	// Only 2 eligible (tier >= 1)
	if len(arbiters) != 2 {
		t.Fatalf("expected 2 arbiters (tier >= 1 only), got %d", len(arbiters))
	}
}

func TestSelectArbitersNoneAvailable(t *testing.T) {
	k, ctx, _, _, sk := setupKeeper(t)

	sk.validators = []types.ValidatorInfo{
		{Address: testAddr(1), Stake: 5000, Tier: 2}, // this is the challenger
		{Address: testAddr(2), Stake: 4000, Tier: 2}, // this is the defender
	}

	_, err := k.SelectArbiters(ctx, 3, testAddr(1), testAddr(2))
	if err == nil {
		t.Fatal("expected error when no arbiters available")
	}
}

// ---------- ID Generation Tests ----------

func TestGenerateDisputeID(t *testing.T) {
	id1 := keeper.GenerateDisputeID("fact-1", "addr-1", 100)
	id2 := keeper.GenerateDisputeID("fact-1", "addr-1", 100)
	id3 := keeper.GenerateDisputeID("fact-1", "addr-2", 100)

	if id1 != id2 {
		t.Fatal("same inputs should produce same ID")
	}
	if id1 == id3 {
		t.Fatal("different inputs should produce different IDs")
	}
	if len(id1) != 32 {
		t.Fatalf("expected 32-char hex ID, got %d chars", len(id1))
	}
}

func TestGenerateEvidenceID(t *testing.T) {
	id := keeper.GenerateEvidenceID("dispute-1", "addr-1", 100)
	if len(id) != 32 {
		t.Fatalf("expected 32-char hex ID, got %d chars", len(id))
	}
}

// ---------- GetTierConfig Tests ----------

func TestGetTierConfig(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	params := k.GetParams(ctx)

	tc, err := k.GetTierConfig(params, types.TierBasic)
	if err != nil {
		t.Fatalf("failed to get tier config: %v", err)
	}
	if tc.ArbiterCount != 3 {
		t.Fatalf("expected 3 arbiters for basic tier, got %d", tc.ArbiterCount)
	}

	tc2, err := k.GetTierConfig(params, types.TierEscalated)
	if err != nil {
		t.Fatalf("failed to get tier config: %v", err)
	}
	if tc2.ArbiterCount != 7 {
		t.Fatalf("expected 7 arbiters for escalated tier, got %d", tc2.ArbiterCount)
	}

	_, err = k.GetTierConfig(params, types.EscalationTier(99))
	if err == nil {
		t.Fatal("expected error for invalid tier")
	}
}

// ---------- MsgServer Tests ----------

func TestMsgOpenDispute(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	setupVerifiedFact(kk, "fact-msg", defender)
	setupValidators(sk, challenger, defender)
	bk.fundAccount(challenger, sdkmath.NewIntWithDecimal(1, 12))

	srv := keeper.NewMsgServerImpl(k)
	resp, err := srv.OpenDispute(ctx, &types.MsgOpenDispute{
		Challenger: challenger,
		FactId:     "fact-msg",
		Reason:     "Incorrect temperature data",
		Bond:       "111000000",
	})
	if err != nil {
		t.Fatalf("msg open dispute failed: %v", err)
	}
	if resp.DisputeId == "" {
		t.Fatal("expected dispute ID in response")
	}

	// Verify bond was deducted
	bal := bk.balances[challenger]
	expected := sdkmath.NewIntWithDecimal(1, 12).Sub(sdkmath.NewInt(111000000))
	if !bal.Equal(expected) {
		t.Fatalf("expected balance %s, got %s", expected, bal)
	}
}

func TestMsgSubmitEvidence(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	challenger := testAddr(1)
	dispute := types.Dispute{
		Id:               "dispute-ev-msg",
		FactId:           "fact-001",
		Challenger:       challenger,
		Defender:         testAddr(2),
		Status:           string(types.DisputeStatusEvidence),
		EvidenceDeadline: 200,
	}
	k.SetDispute(ctx, dispute)

	srv := keeper.NewMsgServerImpl(k)
	resp, err := srv.SubmitEvidence(ctx, &types.MsgSubmitEvidence{
		Submitter: challenger,
		DisputeId: "dispute-ev-msg",
		Side:      "challenger",
		Content:   "Evidence hash: abc123",
	})
	if err != nil {
		t.Fatalf("submit evidence failed: %v", err)
	}
	if resp.EvidenceId == "" {
		t.Fatal("expected evidence ID")
	}

	// Verify evidence count updated
	updated, _ := k.GetDispute(ctx, "dispute-ev-msg")
	if updated.EvidenceCount != 1 {
		t.Fatalf("expected evidence count 1, got %d", updated.EvidenceCount)
	}
}

func TestMsgSubmitEvidenceWrongSide(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dispute := types.Dispute{
		Id:               "dispute-ev-wrong",
		Challenger:       testAddr(1),
		Defender:         testAddr(2),
		Status:           string(types.DisputeStatusEvidence),
		EvidenceDeadline: 200,
	}
	k.SetDispute(ctx, dispute)

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.SubmitEvidence(ctx, &types.MsgSubmitEvidence{
		Submitter: testAddr(1),
		DisputeId: "dispute-ev-wrong",
		Side:      "defender", // wrong side for this submitter
		Content:   "Evidence",
	})
	if err == nil {
		t.Fatal("expected error for wrong side")
	}
}

func TestMsgVoteDispute(t *testing.T) {
	k, ctx, _, _, sk := setupKeeper(t)

	arbiter := testAddr(10)
	sk.validators = []types.ValidatorInfo{{Address: arbiter, Stake: 5000, Tier: 2}}

	dispute := types.Dispute{
		Id:             "dispute-vote-msg",
		Status:         string(types.DisputeStatusVoting),
		VotingDeadline: 200,
		Arbiters:       []string{arbiter},
	}
	k.SetDispute(ctx, dispute)

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.VoteDispute(ctx, &types.MsgVoteDispute{
		Arbiter:   arbiter,
		DisputeId: "dispute-vote-msg",
		Vote:      "overturn",
		Rationale: "Evidence is compelling",
	})
	if err != nil {
		t.Fatalf("msg vote failed: %v", err)
	}
}

func TestMsgEscalateDispute(t *testing.T) {
	k, ctx, bk, _, sk := setupKeeper(t)

	requester := testAddr(5)
	bk.fundAccount(requester, sdkmath.NewIntWithDecimal(1, 12))
	setupValidators(sk, testAddr(1), testAddr(2))

	dispute := types.Dispute{
		Id:         "dispute-esc-msg",
		Challenger: testAddr(1),
		Defender:   testAddr(2),
		Status:     string(types.DisputeStatusEvidence),
		Tier:       uint32(types.TierBasic),
		Bond:       "111000000",
		CreatedAt:  50,
	}
	k.SetDispute(ctx, dispute)

	params := k.GetParams(ctx)
	params.EscalationDelay = 0
	k.SetParams(ctx, params)

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.EscalateDispute(ctx, &types.MsgEscalateDispute{
		Requester:      requester,
		DisputeId:      "dispute-esc-msg",
		AdditionalStake: "555000000",
	})
	if err != nil {
		t.Fatalf("msg escalate failed: %v", err)
	}
}

// ---------- Query Tests ----------

func TestQueryDispute(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	dispute := types.Dispute{Id: "q-dispute-1", FactId: "f1", Status: string(types.DisputeStatusEvidence)}
	k.SetDispute(ctx, dispute)

	resp, err := qs.Dispute(ctx, &types.QueryDisputeRequest{DisputeId: "q-dispute-1"})
	if err != nil {
		t.Fatalf("query dispute failed: %v", err)
	}
	if resp.Dispute.FactId != "f1" {
		t.Fatalf("expected fact_id f1, got %s", resp.Dispute.FactId)
	}
}

func TestQueryDisputeNotFound(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	_, err := qs.Dispute(ctx, &types.QueryDisputeRequest{DisputeId: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent dispute")
	}
}

func TestQueryDisputesByFact(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	k.SetDispute(ctx, types.Dispute{Id: "qd1", FactId: "qf1", Status: string(types.DisputeStatusEvidence)})
	k.SetDispute(ctx, types.Dispute{Id: "qd2", FactId: "qf1", Status: string(types.DisputeStatusVoting)})
	k.SetDispute(ctx, types.Dispute{Id: "qd3", FactId: "qf2", Status: string(types.DisputeStatusEvidence)})

	resp, err := qs.DisputesByFact(ctx, &types.QueryDisputesByFactRequest{FactId: "qf1"})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(resp.Disputes) != 2 {
		t.Fatalf("expected 2, got %d", len(resp.Disputes))
	}
}

func TestQueryActiveDisputes(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	k.SetDispute(ctx, types.Dispute{Id: "qa1", FactId: "f1", Status: string(types.DisputeStatusEvidence)})
	k.SetDispute(ctx, types.Dispute{Id: "qa2", FactId: "f2", Status: string(types.DisputeStatusResolved)})

	resp, err := qs.ActiveDisputes(ctx, &types.QueryActiveDisputesRequest{})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(resp.Disputes) != 1 {
		t.Fatalf("expected 1 active, got %d", len(resp.Disputes))
	}
}

func TestQueryEvidence(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	k.SetEvidence(ctx, types.DisputeEvidence{Id: "e1", DisputeId: "d1", Content: "ev1"})
	k.SetEvidence(ctx, types.DisputeEvidence{Id: "e2", DisputeId: "d1", Content: "ev2"})

	resp, err := qs.Evidence(ctx, &types.QueryEvidenceRequest{DisputeId: "d1"})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(resp.Evidence) != 2 {
		t.Fatalf("expected 2 evidence, got %d", len(resp.Evidence))
	}
}

func TestQueryVotes(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	k.SetVote(ctx, types.DisputeVote{DisputeId: "dv1", Arbiter: testAddr(10), Vote: "uphold", Stake: "5000"})
	k.SetVote(ctx, types.DisputeVote{DisputeId: "dv1", Arbiter: testAddr(11), Vote: "overturn", Stake: "4000"})

	resp, err := qs.Votes(ctx, &types.QueryVotesRequest{DisputeId: "dv1"})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(resp.Votes) != 2 {
		t.Fatalf("expected 2 votes, got %d", len(resp.Votes))
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(resp.Params.TierConfigs) != 3 {
		t.Fatalf("expected 3 tiers, got %d", len(resp.Params.TierConfigs))
	}
}

// ---------- Genesis Tests ----------

func TestGenesisExportImport(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Create some state
	k.SetDispute(ctx, types.Dispute{
		Id:     "gen-d1",
		FactId: "gen-f1",
		Status: string(types.DisputeStatusEvidence),
		Bond:   "111000000",
	})
	k.SetEvidence(ctx, types.DisputeEvidence{
		Id:        "gen-e1",
		DisputeId: "gen-d1",
		Content:   "test evidence",
	})
	k.SetVote(ctx, types.DisputeVote{
		DisputeId: "gen-d1",
		Arbiter:   testAddr(10),
		Vote:      "uphold",
		Stake:     "5000",
	})

	exported := k.ExportGenesis(ctx)
	if len(exported.Disputes) != 1 {
		t.Fatalf("expected 1 dispute in export, got %d", len(exported.Disputes))
	}
	if len(exported.Evidence) != 1 {
		t.Fatalf("expected 1 evidence in export, got %d", len(exported.Evidence))
	}
	if len(exported.Votes) != 1 {
		t.Fatalf("expected 1 vote in export, got %d", len(exported.Votes))
	}

	// Re-initialize with exported state
	k2, ctx2, _, _, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, *exported)

	d, found := k2.GetDispute(ctx2, "gen-d1")
	if !found || d.Bond != "111000000" {
		t.Fatal("dispute not restored from genesis")
	}
}

func TestDefaultGenesisValidation(t *testing.T) {
	gs := types.DefaultGenesis()
	if err := gs.Validate(); err != nil {
		t.Fatalf("default genesis should be valid: %v", err)
	}
}

// ---------- Full Lifecycle Test ----------

func TestFullDisputeLifecycle(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	factId := "lifecycle-fact"

	setupVerifiedFact(kk, factId, defender)
	setupValidators(sk, challenger, defender)
	bk.fundAccount(challenger, sdkmath.NewIntWithDecimal(1, 12))

	srv := keeper.NewMsgServerImpl(k)

	// 1. Open dispute
	openResp, err := srv.OpenDispute(ctx, &types.MsgOpenDispute{
		Challenger: challenger,
		FactId:     factId,
		Reason:     "Temperature was measured incorrectly",
		Bond:       "111000000",
	})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	disputeId := openResp.DisputeId

	// 2. Submit evidence from both sides
	_, err = srv.SubmitEvidence(ctx, &types.MsgSubmitEvidence{
		Submitter: challenger,
		DisputeId: disputeId,
		Side:      "challenger",
		Content:   "Temperature instrument was miscalibrated",
	})
	if err != nil {
		t.Fatalf("challenger evidence failed: %v", err)
	}

	_, err = srv.SubmitEvidence(ctx, &types.MsgSubmitEvidence{
		Submitter: defender,
		DisputeId: disputeId,
		Side:      "defender",
		Content:   "Calibration records show it was accurate",
	})
	if err != nil {
		t.Fatalf("defender evidence failed: %v", err)
	}

	// 3. Transition to voting by advancing past evidence deadline
	d, _ := k.GetDispute(ctx, disputeId)
	ctx = ctx.WithBlockHeight(int64(d.EvidenceDeadline + 1))
	k.ProcessExpiredDisputes(ctx)

	d, _ = k.GetDispute(ctx, disputeId)
	if d.Status != string(types.DisputeStatusVoting) {
		t.Fatalf("expected voting status, got %s", d.Status)
	}

	// 4. Arbiters vote (all uphold)
	for _, arbiter := range d.Arbiters {
		_, err = srv.VoteDispute(ctx, &types.MsgVoteDispute{
			Arbiter:   arbiter,
			DisputeId: disputeId,
			Vote:      "uphold",
			Rationale: "Defender's evidence is more convincing",
		})
		if err != nil {
			t.Fatalf("vote failed for %s: %v", arbiter, err)
		}
	}

	// 5. Advance past voting deadline to trigger resolution
	bk.fundModule(types.ModuleName, sdkmath.NewIntWithDecimal(1, 12))
	ctx = ctx.WithBlockHeight(int64(d.VotingDeadline + 1))
	k.ProcessExpiredDisputes(ctx)

	// 6. Verify final state
	final, _ := k.GetDispute(ctx, disputeId)
	if final.Status != string(types.DisputeStatusResolved) {
		t.Fatalf("expected resolved, got %s", final.Status)
	}
	if final.Outcome != string(types.OutcomeUpheld) {
		t.Fatalf("expected upheld, got %s", final.Outcome)
	}

	// Verify evidence count
	if final.EvidenceCount != 2 {
		t.Fatalf("expected 2 evidence submissions, got %d", final.EvidenceCount)
	}

	// Verify all votes recorded
	votes := k.GetVotes(ctx, disputeId)
	if len(votes) != len(d.Arbiters) {
		t.Fatalf("expected %d votes, got %d", len(d.Arbiters), len(votes))
	}
}
