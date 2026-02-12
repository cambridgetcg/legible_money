package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/capture_challenge/keeper"
	"github.com/legible-money/legible/x/capture_challenge/types"
	capturedefensetypes "github.com/legible-money/legible/x/capture_defense/types"
)

var testChallengerAddr sdk.AccAddress

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
	testChallengerAddr = sdk.AccAddress([]byte("capture_test_addr___"))
}

// ---------- Mock CaptureDefenseKeeper ----------

type mockCaptureDefenseKeeper struct {
	metrics map[string]capturedefensetypes.CaptureMetrics
}

func newMockCaptureDefenseKeeper() *mockCaptureDefenseKeeper {
	return &mockCaptureDefenseKeeper{
		metrics: make(map[string]capturedefensetypes.CaptureMetrics),
	}
}

func (m *mockCaptureDefenseKeeper) GetCaptureMetrics(_ sdk.Context, domain string) (capturedefensetypes.CaptureMetrics, bool) {
	cm, ok := m.metrics[domain]
	return cm, ok
}

func (m *mockCaptureDefenseKeeper) AnalyzeCaptureRisk(_ sdk.Context, domain string, _ uint64) capturedefensetypes.CaptureMetrics {
	cm, ok := m.metrics[domain]
	if ok {
		return cm
	}
	return capturedefensetypes.CaptureMetrics{Domain: domain}
}

// ---------- Mock StakingKeeper ----------

type mockStakingKeeper struct {
	stakes  map[string]uint64
	slashes map[string][]slashRecord
}

type slashRecord struct {
	Amount string
	Reason string
}

func newMockStakingKeeper() *mockStakingKeeper {
	return &mockStakingKeeper{
		stakes:  make(map[string]uint64),
		slashes: make(map[string][]slashRecord),
	}
}

func (m *mockStakingKeeper) GetEffectiveStake(_ context.Context, operatorAddr string) uint64 {
	return m.stakes[operatorAddr]
}

func (m *mockStakingKeeper) SlashValidator(_ context.Context, operatorAddr string, amount string, reason string) error {
	m.slashes[operatorAddr] = append(m.slashes[operatorAddr], slashRecord{Amount: amount, Reason: reason})
	return nil
}

// ---------- Mock BankKeeper ----------

type mockBankKeeper struct{}

func (m mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, _ sdk.AccAddress, _ string, _ sdk.Coins) error {
	return nil
}

func (m mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
	return nil
}

// ---------- Test Setup ----------

var testAuthority string

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockCaptureDefenseKeeper, *mockStakingKeeper) {
	testAuthority = testChallengerAddr.String()
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

	mockCDK := newMockCaptureDefenseKeeper()
	mockSK := newMockStakingKeeper()

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		mockCDK,
		mockSK,
		mockBankKeeper{},
		testAuthority,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 10}, false, log.NewNopLogger())

	// Set default params
	k.SetParams(ctx, *types.DefaultParams())

	return k, ctx, mockCDK, mockSK
}

// makeEvidence creates N CaptureEvidence pieces for testing.
func makeEvidence(n int) []*types.CaptureEvidence {
	evidence := make([]*types.CaptureEvidence, n)
	for i := 0; i < n; i++ {
		evidence[i] = &types.CaptureEvidence{
			Id:          "ev-" + string(rune('a'+i)),
			Type:        string(types.EvidenceCoordinatedVoting),
			Description: "Evidence piece " + string(rune('a'+i)),
			DataHash:    "hash_" + string(rune('a'+i)),
			SubmittedAt: 10,
			SubmittedBy: testAuthority,
		}
	}
	return evidence
}

// ---------- Tests: Submit ----------

func TestSubmitChallengeSuccess(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	msg := &types.MsgSubmitChallenge{
		Challenger:        testAuthority,
		Domain:            "physics",
		Evidence:          makeEvidence(3),
		Stake:             "77000000", // exactly MinChallengeStake (77 LGM)
		AccusedValidators: []string{"lgmvaloper1abc", "lgmvaloper1def"},
	}

	resp, err := k.SubmitChallenge(ctx, msg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ChallengeId == "" {
		t.Fatal("expected non-empty challenge ID")
	}

	// Verify challenge was persisted
	ch, found := k.GetChallenge(ctx, resp.ChallengeId)
	if !found {
		t.Fatal("expected challenge to be found in store")
	}

	// Verify status is evidence_period
	if ch.Status != string(types.ChallengeEvidencePeriod) {
		t.Errorf("expected status=%s, got %s", types.ChallengeEvidencePeriod, ch.Status)
	}

	// Verify deadlines
	currentBlock := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)
	expectedEvidenceDeadline := currentBlock + params.EvidencePeriodBlocks
	expectedReviewDeadline := currentBlock + params.EvidencePeriodBlocks + params.ReviewPeriodBlocks
	if ch.EvidenceDeadline != expectedEvidenceDeadline {
		t.Errorf("expected evidence deadline=%d, got %d", expectedEvidenceDeadline, ch.EvidenceDeadline)
	}
	if ch.ReviewDeadline != expectedReviewDeadline {
		t.Errorf("expected review deadline=%d, got %d", expectedReviewDeadline, ch.ReviewDeadline)
	}

	// Verify bounty pool updated
	pool, found := k.GetBountyPool(ctx, "physics")
	if !found {
		t.Fatal("expected bounty pool to be created")
	}
	if pool.ChallengesSubmitted != 1 {
		t.Errorf("expected ChallengesSubmitted=1, got %d", pool.ChallengesSubmitted)
	}
	if pool.Domain != "physics" {
		t.Errorf("expected pool domain=physics, got %s", pool.Domain)
	}
}

func TestSubmitChallengeInsufficientStake(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	msg := &types.MsgSubmitChallenge{
		Challenger:        testAuthority,
		Domain:            "physics",
		Evidence:          makeEvidence(3),
		Stake:             "50000000", // 50 LGM, below 77 LGM minimum
		AccusedValidators: []string{"lgmvaloper1abc"},
	}

	_, err := k.SubmitChallenge(ctx, msg)
	if err == nil {
		t.Fatal("expected error for insufficient stake")
	}
	if err != types.ErrInsufficientChallengeStake {
		t.Errorf("expected ErrInsufficientChallengeStake, got %v", err)
	}
}

func TestSubmitChallengeInsufficientEvidence(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	msg := &types.MsgSubmitChallenge{
		Challenger:        testAuthority,
		Domain:            "physics",
		Evidence:          makeEvidence(2), // only 2, minimum is 3
		Stake:             "77000000",
		AccusedValidators: []string{"lgmvaloper1abc"},
	}

	_, err := k.SubmitChallenge(ctx, msg)
	if err == nil {
		t.Fatal("expected error for insufficient evidence")
	}
	if err != types.ErrInsufficientEvidence {
		t.Errorf("expected ErrInsufficientEvidence, got %v", err)
	}
}

// ---------- Tests: Lifecycle ----------

// submitAndResolve is a helper that creates a challenge and resolves it.
func submitAndResolve(t *testing.T, k keeper.Keeper, ctx sdk.Context, outcome string) (types.CaptureChallenge, *types.MsgResolveChallengeResponse) {
	t.Helper()

	// Submit the challenge
	submitMsg := &types.MsgSubmitChallenge{
		Challenger:        testAuthority,
		Domain:            "mathematics",
		Evidence:          makeEvidence(4),
		Stake:             "77000000",
		AccusedValidators: []string{"lgmvaloper1abc", "lgmvaloper1def"},
	}
	submitResp, err := k.SubmitChallenge(ctx, submitMsg)
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	// Resolve the challenge
	resolveMsg := &types.MsgResolveChallenge{
		Resolver:      testAuthority,
		ChallengeId:   submitResp.ChallengeId,
		Outcome:       outcome,
		Justification: "test resolution: " + outcome,
	}
	resolveResp, err := k.ResolveChallenge(ctx, resolveMsg)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	ch, found := k.GetChallenge(ctx, submitResp.ChallengeId)
	if !found {
		t.Fatal("expected challenge to exist after resolution")
	}

	return ch, resolveResp
}

func TestResolveChallengeUpheld(t *testing.T) {
	k, ctx, _, mockSK := setupKeeper(t)

	// Set up validator stakes for slashing
	mockSK.stakes["lgmvaloper1abc"] = 100_000_000 // 100 LGM
	mockSK.stakes["lgmvaloper1def"] = 200_000_000 // 200 LGM

	ch, resp := submitAndResolve(t, k, ctx, "upheld")

	// Verify status
	if ch.Status != string(types.ChallengeUpheld) {
		t.Errorf("expected status=%s, got %s", types.ChallengeUpheld, ch.Status)
	}

	// Verify challenger reward: 7.7x stake + 777 LGM bonus
	// 77000000 * 77 / 10 = 592900000
	// 592900000 + 777000000 = 1369900000
	expectedReward := "1369900000"
	if resp.ChallengerReward != expectedReward {
		t.Errorf("expected challenger reward=%s, got %s", expectedReward, resp.ChallengerReward)
	}

	// Verify DomainPausedUntil set
	currentBlock := uint64(ctx.BlockHeight())
	expectedPause := currentBlock + types.DefaultDomainPauseDurationBlocks
	if ch.DomainPausedUntil != expectedPause {
		t.Errorf("expected DomainPausedUntil=%d, got %d", expectedPause, ch.DomainPausedUntil)
	}

	// Verify bounty pool ChallengesUpheld incremented
	pool, found := k.GetBountyPool(ctx, "mathematics")
	if !found {
		t.Fatal("expected bounty pool to exist")
	}
	if pool.ChallengesUpheld != 1 {
		t.Errorf("expected ChallengesUpheld=1, got %d", pool.ChallengesUpheld)
	}

	// Verify slashes were recorded for accused validators
	// lgmvaloper1abc: 100000000 * 300000 / 1000000 = 30000000
	if slashes, ok := mockSK.slashes["lgmvaloper1abc"]; !ok || len(slashes) == 0 {
		t.Error("expected slash for lgmvaloper1abc")
	} else if slashes[0].Amount != "30000000" {
		t.Errorf("expected slash amount=30000000 for lgmvaloper1abc, got %s", slashes[0].Amount)
	}
	// lgmvaloper1def: 200000000 * 300000 / 1000000 = 60000000
	if slashes, ok := mockSK.slashes["lgmvaloper1def"]; !ok || len(slashes) == 0 {
		t.Error("expected slash for lgmvaloper1def")
	} else if slashes[0].Amount != "60000000" {
		t.Errorf("expected slash amount=60000000 for lgmvaloper1def, got %s", slashes[0].Amount)
	}

	// Verify resolution is attached
	if ch.Resolution == nil {
		t.Fatal("expected resolution to be attached")
	}
	if ch.Resolution.Outcome != "upheld" {
		t.Errorf("expected resolution outcome=upheld, got %s", ch.Resolution.Outcome)
	}
}

func TestResolveChallengeRejected(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	ch, _ := submitAndResolve(t, k, ctx, "rejected")

	// Verify status
	if ch.Status != string(types.ChallengeRejected) {
		t.Errorf("expected status=%s, got %s", types.ChallengeRejected, ch.Status)
	}

	// Verify 77% of stake added to bounty pool balance
	// 77000000 * 770000 / 1000000 = 59290000
	pool, found := k.GetBountyPool(ctx, "mathematics")
	if !found {
		t.Fatal("expected bounty pool to exist")
	}
	if pool.Balance != "59290000" {
		t.Errorf("expected pool balance=59290000, got %s", pool.Balance)
	}
	if pool.ChallengesRejected != 1 {
		t.Errorf("expected ChallengesRejected=1, got %d", pool.ChallengesRejected)
	}
}

func TestResolveChallengePartial(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	ch, resp := submitAndResolve(t, k, ctx, "partial")

	// Verify status
	if ch.Status != string(types.ChallengePartial) {
		t.Errorf("expected status=%s, got %s", types.ChallengePartial, ch.Status)
	}

	// Verify reward: 3.85x stake = 77000000 * 385 / 100 = 296450000
	expectedReward := "296450000"
	if resp.ChallengerReward != expectedReward {
		t.Errorf("expected challenger reward=%s, got %s", expectedReward, resp.ChallengerReward)
	}
}

// ---------- Tests: Phase Advancement ----------

func TestAdvanceChallengePhases(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Create a challenge manually in evidence_period with known deadlines
	ch := types.CaptureChallenge{
		Id:               "test-phase-challenge",
		Domain:           "physics",
		Challenger:       testAuthority,
		ChallengeStake:   "77000000",
		Evidence:         makeEvidence(3),
		SubmittedAtBlock: 10,
		EvidenceDeadline: 100,  // evidence deadline at block 100
		ReviewDeadline:   8000, // review deadline at block 8000
		Status:           string(types.ChallengeEvidencePeriod),
	}
	k.SetChallenge(ctx, ch)

	// At block 50, still in evidence period
	k.AdvanceChallengePhases(ctx, 50)
	updated, _ := k.GetChallenge(ctx, "test-phase-challenge")
	if updated.Status != string(types.ChallengeEvidencePeriod) {
		t.Errorf("expected status=%s at block 50, got %s", types.ChallengeEvidencePeriod, updated.Status)
	}

	// At block 101, past evidence deadline -- should advance to under_review
	k.AdvanceChallengePhases(ctx, 101)
	updated, _ = k.GetChallenge(ctx, "test-phase-challenge")
	if updated.Status != string(types.ChallengeUnderReview) {
		t.Errorf("expected status=%s at block 101, got %s", types.ChallengeUnderReview, updated.Status)
	}

	// At block 8001, past review deadline -- should advance to expired
	k.AdvanceChallengePhases(ctx, 8001)
	updated, _ = k.GetChallenge(ctx, "test-phase-challenge")
	if updated.Status != string(types.ChallengeExpired) {
		t.Errorf("expected status=%s at block 8001, got %s", types.ChallengeExpired, updated.Status)
	}
}

// ---------- Tests: Auto-Fund Bounty Pools ----------

func TestAutoFundBountyPools(t *testing.T) {
	k, ctx, mockCDK, _ := setupKeeper(t)

	// Set up a bounty pool for a domain
	pool := types.DomainBountyPool{
		Domain:           "physics",
		Balance:          "1000",
		TotalRewardsPaid: "0",
		CreatedAtBlock:   5,
		LastUpdatedBlock: 5,
	}
	k.SetBountyPool(ctx, pool)

	// Set up capture metrics with high risk score (>= 400000)
	mockCDK.metrics["physics"] = capturedefensetypes.CaptureMetrics{
		Domain:    "physics",
		RiskScore: 500_000, // 50%, above the 40% threshold
	}

	// Run auto-fund at block 777
	k.AutoFundBountyPools(ctx, 777)

	// Verify pool was updated
	updatedPool, found := k.GetBountyPool(ctx, "physics")
	if !found {
		t.Fatal("expected bounty pool to exist after auto-fund")
	}
	if updatedPool.CaptureRiskScore != 500_000 {
		t.Errorf("expected CaptureRiskScore=500000, got %d", updatedPool.CaptureRiskScore)
	}
	if !updatedPool.AutoGenerated {
		t.Error("expected AutoGenerated=true after auto-fund")
	}
	if updatedPool.LastAnalyzedBlock != 777 {
		t.Errorf("expected LastAnalyzedBlock=777, got %d", updatedPool.LastAnalyzedBlock)
	}
	if updatedPool.LastUpdatedBlock != 777 {
		t.Errorf("expected LastUpdatedBlock=777, got %d", updatedPool.LastUpdatedBlock)
	}

	// Test that low-risk domains are NOT auto-funded
	lowRiskPool := types.DomainBountyPool{
		Domain:           "mathematics",
		Balance:          "500",
		TotalRewardsPaid: "0",
		CreatedAtBlock:   5,
		LastUpdatedBlock: 5,
	}
	k.SetBountyPool(ctx, lowRiskPool)

	mockCDK.metrics["mathematics"] = capturedefensetypes.CaptureMetrics{
		Domain:    "mathematics",
		RiskScore: 200_000, // 20%, below the 40% threshold
	}

	k.AutoFundBountyPools(ctx, 800)

	lowPool, _ := k.GetBountyPool(ctx, "mathematics")
	if lowPool.AutoGenerated {
		t.Error("expected AutoGenerated=false for low-risk domain")
	}
	if lowPool.LastAnalyzedBlock != 0 {
		t.Errorf("expected LastAnalyzedBlock=0 for low-risk domain, got %d", lowPool.LastAnalyzedBlock)
	}
}
