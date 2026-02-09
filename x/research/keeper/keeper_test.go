package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/research/keeper"
	"github.com/legible-money/legible/x/research/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

// ---------- Mock Bank Keeper ----------

type mockBankKeeper struct {
	balances map[string]map[string]sdkmath.Int // addr -> denom -> amount
	module   map[string]map[string]sdkmath.Int // module -> denom -> amount
	burned   sdkmath.Int
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances: make(map[string]map[string]sdkmath.Int),
		module:   make(map[string]map[string]sdkmath.Int),
		burned:   sdkmath.ZeroInt(),
	}
}

func (m *mockBankKeeper) fundAccount(addr string, denom string, amount sdkmath.Int) {
	if m.balances[addr] == nil {
		m.balances[addr] = make(map[string]sdkmath.Int)
	}
	cur, ok := m.balances[addr][denom]
	if !ok {
		cur = sdkmath.ZeroInt()
	}
	m.balances[addr][denom] = cur.Add(amount)
}

func (m *mockBankKeeper) fundModule(mod string, denom string, amount sdkmath.Int) {
	if m.module[mod] == nil {
		m.module[mod] = make(map[string]sdkmath.Int)
	}
	cur, ok := m.module[mod][denom]
	if !ok {
		cur = sdkmath.ZeroInt()
	}
	m.module[mod][denom] = cur.Add(amount)
}

func (m *mockBankKeeper) SendCoins(_ context.Context, _, _ sdk.AccAddress, _ sdk.Coins) error {
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, sender sdk.AccAddress, moduleName string, amt sdk.Coins) error {
	addr := sender.String()
	for _, coin := range amt {
		if m.balances[addr] == nil {
			m.balances[addr] = make(map[string]sdkmath.Int)
		}
		cur, ok := m.balances[addr][coin.Denom]
		if !ok {
			cur = sdkmath.ZeroInt()
		}
		m.balances[addr][coin.Denom] = cur.Sub(coin.Amount)

		if m.module[moduleName] == nil {
			m.module[moduleName] = make(map[string]sdkmath.Int)
		}
		mcur, ok := m.module[moduleName][coin.Denom]
		if !ok {
			mcur = sdkmath.ZeroInt()
		}
		m.module[moduleName][coin.Denom] = mcur.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, moduleName string, recipient sdk.AccAddress, amt sdk.Coins) error {
	addr := recipient.String()
	for _, coin := range amt {
		if m.module[moduleName] == nil {
			m.module[moduleName] = make(map[string]sdkmath.Int)
		}
		mcur, ok := m.module[moduleName][coin.Denom]
		if !ok {
			mcur = sdkmath.ZeroInt()
		}
		m.module[moduleName][coin.Denom] = mcur.Sub(coin.Amount)

		if m.balances[addr] == nil {
			m.balances[addr] = make(map[string]sdkmath.Int)
		}
		cur, ok := m.balances[addr][coin.Denom]
		if !ok {
			cur = sdkmath.ZeroInt()
		}
		m.balances[addr][coin.Denom] = cur.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) BurnCoins(_ context.Context, moduleName string, amt sdk.Coins) error {
	for _, coin := range amt {
		if m.module[moduleName] == nil {
			m.module[moduleName] = make(map[string]sdkmath.Int)
		}
		mcur, ok := m.module[moduleName][coin.Denom]
		if !ok {
			mcur = sdkmath.ZeroInt()
		}
		m.module[moduleName][coin.Denom] = mcur.Sub(coin.Amount)
		m.burned = m.burned.Add(coin.Amount)
	}
	return nil
}

// ---------- Mock Staking Keeper ----------

type mockStakingKeeper struct{}

func (m *mockStakingKeeper) GetValidatorStake(_ sdk.Context, _ string) (string, error) {
	return "1000000000", nil // 1000 LGM
}

// ---------- Test Setup ----------

const authority = "lgm10d07y265gmmuvt4z0w9aw880jnsr700jq7e2dg" // deterministic

// largeBalance is a large amount for funding test accounts (avoids Int64 overflow).
var largeBalance = sdkmath.NewIntWithDecimal(1, 24) // 10^24 ulgm = 1M LGM

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper) {
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

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		bk,
		&mockStakingKeeper{},
		authority,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	k.InitGenesis(ctx, *types.DefaultGenesis())

	return k, ctx, bk
}

func testAddr(name string) string {
	addr := sdk.AccAddress([]byte("addr_" + name + "_______________")[:20])
	return addr.String()
}

func newMsgServer(k keeper.Keeper) types.MsgServer {
	return keeper.NewMsgServerImpl(k)
}

func newQueryServer(k keeper.Keeper) types.QueryServer {
	return keeper.NewQueryServerImpl(k)
}

// ---------- Params Tests ----------

func TestSetGetParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	params := k.GetParams(ctx)

	if params.MinReviewerCount != 3 {
		t.Errorf("expected min_reviewer_count=3, got %d", params.MinReviewerCount)
	}
	if params.AcceptanceScoreThreshold != 60 {
		t.Errorf("expected acceptance_score_threshold=60, got %d", params.AcceptanceScoreThreshold)
	}
	if params.RejectionSlashBps != 3300 {
		t.Errorf("expected rejection_slash_bps=3300, got %d", params.RejectionSlashBps)
	}
	if params.ReviewPeriodBlocks != 68544 {
		t.Errorf("expected review_period_blocks=68544, got %d", params.ReviewPeriodBlocks)
	}
	if params.BountyMinDeadlineBlocks != 34272 {
		t.Errorf("expected bounty_min_deadline_blocks=34272, got %d", params.BountyMinDeadlineBlocks)
	}
}

func TestUpdateParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := newMsgServer(k)

	newParams := types.DefaultParams()
	newParams.MinReviewerCount = 5
	newParams.AcceptanceScoreThreshold = 70

	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    newParams,
	})
	if err != nil {
		t.Fatalf("UpdateParams failed: %v", err)
	}

	got := k.GetParams(ctx)
	if got.MinReviewerCount != 5 {
		t.Errorf("expected min_reviewer_count=5, got %d", got.MinReviewerCount)
	}
	if got.AcceptanceScoreThreshold != 70 {
		t.Errorf("expected acceptance_score_threshold=70, got %d", got.AcceptanceScoreThreshold)
	}
}

func TestUpdateParamsUnauthorized(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := newMsgServer(k)

	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAddr("rando"),
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

// ---------- Submit Research Tests ----------

func TestSubmitResearch(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	resp, err := ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter:    submitter,
		Title:        "Test Research",
		Description:  "A test research submission",
		Domain:       "physics",
		ResearchType: types.ResearchTypeReplication,
		TargetFactId: "FACT-1",
		Stake:        "100000000", // 100 LGM in ulgm
	})
	if err != nil {
		t.Fatalf("SubmitResearch failed: %v", err)
	}

	if resp.ResearchId != "RES-1" {
		t.Errorf("expected RES-1, got %s", resp.ResearchId)
	}

	// Verify stored
	r, found := k.GetResearch(ctx, "RES-1")
	if !found {
		t.Fatal("research not found after submit")
	}
	if r.Status != types.ResearchStatusSubmitted {
		t.Errorf("expected status=submitted, got %s", r.Status)
	}
	if r.Submitter != submitter {
		t.Errorf("expected submitter=%s, got %s", submitter, r.Submitter)
	}
	if r.Domain != "physics" {
		t.Errorf("expected domain=physics, got %s", r.Domain)
	}
	if r.CreatedAt != 100 {
		t.Errorf("expected created_at=100, got %d", r.CreatedAt)
	}
}

func TestSubmitResearchInsufficientStake(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	_, err := ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter:    submitter,
		Title:        "Test",
		Description:  "Test",
		Domain:       "physics",
		ResearchType: types.ResearchTypeReplication,
		Stake:        "50000000", // 50 LGM (below 100 min)
	})
	if err == nil {
		t.Fatal("expected insufficient stake error")
	}
}

func TestSubmitResearchSequentialIds(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	msg := &types.MsgSubmitResearch{
		Submitter:    submitter,
		Title:        "Research",
		Description:  "Desc",
		Domain:       "physics",
		ResearchType: types.ResearchTypeReplication,
		Stake:        "100000000",
	}

	resp1, _ := ms.SubmitResearch(ctx, msg)
	resp2, _ := ms.SubmitResearch(ctx, msg)
	resp3, _ := ms.SubmitResearch(ctx, msg)

	if resp1.ResearchId != "RES-1" || resp2.ResearchId != "RES-2" || resp3.ResearchId != "RES-3" {
		t.Errorf("expected sequential IDs RES-1,2,3, got %s,%s,%s",
			resp1.ResearchId, resp2.ResearchId, resp3.ResearchId)
	}
}

// ---------- Challenge Research Tests ----------

func TestChallengeResearch(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	challenger := testAddr("bob")
	bk.fundAccount(submitter, "ulgm", largeBalance)
	bk.fundAccount(challenger, "ulgm", largeBalance)

	// Submit first
	ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter:    submitter,
		Title:        "Research",
		Description:  "Desc",
		Domain:       "physics",
		ResearchType: types.ResearchTypeReplication,
		Stake:        "100000000",
	})

	// Challenge
	_, err := ms.ChallengeResearch(ctx, &types.MsgChallengeResearch{
		Challenger: challenger,
		ResearchId: "RES-1",
		Reason:     "Methodology flawed",
		Stake:      "100000000",
	})
	if err != nil {
		t.Fatalf("ChallengeResearch failed: %v", err)
	}

	r, _ := k.GetResearch(ctx, "RES-1")
	if r.Status != types.ResearchStatusChallenged {
		t.Errorf("expected status=challenged, got %s", r.Status)
	}
}

func TestChallengeResearchNotFound(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	challenger := testAddr("bob")
	bk.fundAccount(challenger, "ulgm", largeBalance)

	_, err := ms.ChallengeResearch(ctx, &types.MsgChallengeResearch{
		Challenger: challenger,
		ResearchId: "RES-999",
		Reason:     "Does not exist",
		Stake:      "100000000",
	})
	if err == nil {
		t.Fatal("expected error for non-existent research")
	}
}

func TestChallengeResearchAlreadyAccepted(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	challenger := testAddr("bob")
	bk.fundAccount(submitter, "ulgm", largeBalance)
	bk.fundAccount(challenger, "ulgm", largeBalance)

	// Create and manually accept
	ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter: submitter, Title: "Research", Description: "Desc",
		Domain: "physics", ResearchType: types.ResearchTypeReplication,
		Stake: "100000000",
	})

	r, _ := k.GetResearch(ctx, "RES-1")
	r.Status = types.ResearchStatusAccepted
	k.SetResearch(ctx, r)

	_, err := ms.ChallengeResearch(ctx, &types.MsgChallengeResearch{
		Challenger: challenger, ResearchId: "RES-1",
		Reason: "Too late", Stake: "100000000",
	})
	if err == nil {
		t.Fatal("expected error challenging accepted research")
	}
}

// ---------- Review Research Tests ----------

func TestReviewResearch(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	reviewer := testAddr("reviewer1")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter: submitter, Title: "Research", Description: "Desc",
		Domain: "physics", ResearchType: types.ResearchTypeReplication,
		Stake: "100000000",
	})

	_, err := ms.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer:   reviewer,
		ResearchId: "RES-1",
		Verdict:    types.ReviewVerdictApprove,
		Score:      80,
		Reasoning:  "Good methodology",
	})
	if err != nil {
		t.Fatalf("ReviewResearch failed: %v", err)
	}

	// Check status transitioned to under_review
	r, _ := k.GetResearch(ctx, "RES-1")
	if r.Status != types.ResearchStatusUnderReview {
		t.Errorf("expected status=under_review, got %s", r.Status)
	}
	if r.ReviewCount != 1 {
		t.Errorf("expected review_count=1, got %d", r.ReviewCount)
	}
	if r.ApproveCount != 1 {
		t.Errorf("expected approve_count=1, got %d", r.ApproveCount)
	}
	if r.AggregateScore != 80 {
		t.Errorf("expected aggregate_score=80, got %d", r.AggregateScore)
	}
}

func TestReviewResearchMultipleReviewers(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter: submitter, Title: "Research", Description: "Desc",
		Domain: "physics", ResearchType: types.ResearchTypeReplication,
		Stake: "100000000",
	})

	// Three reviews
	ms.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer: testAddr("rev1"), ResearchId: "RES-1",
		Verdict: types.ReviewVerdictApprove, Score: 80, Reasoning: "Good",
	})
	ms.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer: testAddr("rev2"), ResearchId: "RES-1",
		Verdict: types.ReviewVerdictApprove, Score: 70, Reasoning: "Ok",
	})
	ms.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer: testAddr("rev3"), ResearchId: "RES-1",
		Verdict: types.ReviewVerdictReject, Score: 40, Reasoning: "Weak",
	})

	r, _ := k.GetResearch(ctx, "RES-1")
	if r.ReviewCount != 3 {
		t.Errorf("expected review_count=3, got %d", r.ReviewCount)
	}
	if r.ApproveCount != 2 {
		t.Errorf("expected approve_count=2, got %d", r.ApproveCount)
	}
	if r.RejectCount != 1 {
		t.Errorf("expected reject_count=1, got %d", r.RejectCount)
	}
	// (80+70+40)/3 = 63
	if r.AggregateScore != 63 {
		t.Errorf("expected aggregate_score=63, got %d", r.AggregateScore)
	}
}

func TestReviewResearchDuplicateReviewer(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	reviewer := testAddr("rev1")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter: submitter, Title: "Research", Description: "Desc",
		Domain: "physics", ResearchType: types.ResearchTypeReplication,
		Stake: "100000000",
	})

	ms.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer: reviewer, ResearchId: "RES-1",
		Verdict: types.ReviewVerdictApprove, Score: 80, Reasoning: "Good",
	})

	// Duplicate review should fail
	_, err := ms.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer: reviewer, ResearchId: "RES-1",
		Verdict: types.ReviewVerdictReject, Score: 30, Reasoning: "Changed mind",
	})
	if err == nil {
		t.Fatal("expected duplicate reviewer error")
	}
}

// ---------- Resolve Research Tests ----------

func TestResolveResearchAccepted(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter: submitter, Title: "Research", Description: "Desc",
		Domain: "physics", ResearchType: types.ResearchTypeReplication,
		Stake: "100000000",
	})

	// Add 3 positive reviews to meet minimum
	for i := 0; i < 3; i++ {
		ms.ReviewResearch(ctx, &types.MsgReviewResearch{
			Reviewer: testAddr("rev" + string(rune('a'+i))), ResearchId: "RES-1",
			Verdict: types.ReviewVerdictApprove, Score: 80, Reasoning: "Great",
		})
	}

	// Resolve
	resp, err := ms.ResolveResearch(ctx, &types.MsgResolveResearch{
		Authority:  authority,
		ResearchId: "RES-1",
	})
	if err != nil {
		t.Fatalf("ResolveResearch failed: %v", err)
	}
	if resp.Outcome != "accepted" {
		t.Errorf("expected outcome=accepted, got %s", resp.Outcome)
	}

	r, _ := k.GetResearch(ctx, "RES-1")
	if r.Status != types.ResearchStatusAccepted {
		t.Errorf("expected status=accepted, got %s", r.Status)
	}
}

func TestResolveResearchRejected(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter: submitter, Title: "Research", Description: "Desc",
		Domain: "physics", ResearchType: types.ResearchTypeReplication,
		Stake: "100000000",
	})

	// Add 3 negative reviews (score below 60)
	for i := 0; i < 3; i++ {
		ms.ReviewResearch(ctx, &types.MsgReviewResearch{
			Reviewer: testAddr("rev" + string(rune('a'+i))), ResearchId: "RES-1",
			Verdict: types.ReviewVerdictReject, Score: 30, Reasoning: "Flawed",
		})
	}

	resp, err := ms.ResolveResearch(ctx, &types.MsgResolveResearch{
		Authority:  authority,
		ResearchId: "RES-1",
	})
	if err != nil {
		t.Fatalf("ResolveResearch failed: %v", err)
	}
	if resp.Outcome != "rejected" {
		t.Errorf("expected outcome=rejected, got %s", resp.Outcome)
	}

	r, _ := k.GetResearch(ctx, "RES-1")
	if r.Status != types.ResearchStatusRejected {
		t.Errorf("expected status=rejected, got %s", r.Status)
	}
}

func TestResolveResearchInsufficientReviews(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter: submitter, Title: "Research", Description: "Desc",
		Domain: "physics", ResearchType: types.ResearchTypeReplication,
		Stake: "100000000",
	})

	// Only 1 review (need 3)
	ms.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer: testAddr("rev1"), ResearchId: "RES-1",
		Verdict: types.ReviewVerdictApprove, Score: 80, Reasoning: "Good",
	})

	_, err := ms.ResolveResearch(ctx, &types.MsgResolveResearch{
		Authority:  authority,
		ResearchId: "RES-1",
	})
	if err == nil {
		t.Fatal("expected insufficient reviews error")
	}
}

func TestResolveResearchUnauthorized(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter: submitter, Title: "Research", Description: "Desc",
		Domain: "physics", ResearchType: types.ResearchTypeReplication,
		Stake: "100000000",
	})

	_, err := ms.ResolveResearch(ctx, &types.MsgResolveResearch{
		Authority:  testAddr("rando"),
		ResearchId: "RES-1",
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

// ---------- Bounty Tests ----------

func TestCreateBounty(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	creator := testAddr("alice")
	bk.fundAccount(creator, "ulgm", largeBalance)

	resp, err := ms.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator:        creator,
		Title:          "Find error in proof",
		Description:    "Bounty description",
		Domain:         "mathematics",
		Reward:         "1000000000", // 1000 LGM
		DeadlineHeight: 100 + 34272 + 1,         // current(100) + min(34272) + 1
	})
	if err != nil {
		t.Fatalf("CreateBounty failed: %v", err)
	}
	if resp.BountyId != "BOUNTY-1" {
		t.Errorf("expected BOUNTY-1, got %s", resp.BountyId)
	}

	b, found := k.GetBounty(ctx, "BOUNTY-1")
	if !found {
		t.Fatal("bounty not found")
	}
	if b.Status != types.BountyStatusOpen {
		t.Errorf("expected status=open, got %s", b.Status)
	}
	if b.Creator != creator {
		t.Errorf("expected creator=%s, got %s", creator, b.Creator)
	}
}

func TestCreateBountyDeadlineTooSoon(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	creator := testAddr("alice")
	bk.fundAccount(creator, "ulgm", largeBalance)

	_, err := ms.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator:        creator,
		Title:          "Too soon",
		Description:    "Desc",
		Domain:         "physics",
		Reward:         "1000000000",
		DeadlineHeight: 200, // not far enough (need 100 + 34272)
	})
	if err == nil {
		t.Fatal("expected deadline too soon error")
	}
}

func TestCreateBountyExceedsMaxReward(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	creator := testAddr("alice")
	bk.fundAccount(creator, "ulgm", largeBalance)

	_, err := ms.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator:        creator,
		Title:          "Too rich",
		Description:    "Desc",
		Domain:         "physics",
		Reward:         "99999000000000", // way over max
		DeadlineHeight: 100 + 34272 + 1,
	})
	if err == nil {
		t.Fatal("expected exceeds max reward error")
	}
}

func TestClaimBounty(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	creator := testAddr("alice")
	claimer := testAddr("bob")
	bk.fundAccount(creator, "ulgm", largeBalance)

	ms.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator: creator, Title: "Bounty", Description: "Desc",
		Domain: "physics", Reward: "1000000000",
		DeadlineHeight: 100 + 34272 + 1,
	})

	_, err := ms.ClaimBounty(ctx, &types.MsgClaimBounty{
		Claimer:  claimer,
		BountyId: "BOUNTY-1",
	})
	if err != nil {
		t.Fatalf("ClaimBounty failed: %v", err)
	}

	b, _ := k.GetBounty(ctx, "BOUNTY-1")
	if b.Status != types.BountyStatusClaimed {
		t.Errorf("expected status=claimed, got %s", b.Status)
	}
	if b.ClaimedBy != claimer {
		t.Errorf("expected claimed_by=%s, got %s", claimer, b.ClaimedBy)
	}
}

func TestClaimBountyNotOpen(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	creator := testAddr("alice")
	bk.fundAccount(creator, "ulgm", largeBalance)

	ms.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator: creator, Title: "Bounty", Description: "Desc",
		Domain: "physics", Reward: "1000000000",
		DeadlineHeight: 100 + 34272 + 1,
	})

	// Claim once
	ms.ClaimBounty(ctx, &types.MsgClaimBounty{
		Claimer: testAddr("bob"), BountyId: "BOUNTY-1",
	})

	// Try to claim again
	_, err := ms.ClaimBounty(ctx, &types.MsgClaimBounty{
		Claimer: testAddr("charlie"), BountyId: "BOUNTY-1",
	})
	if err == nil {
		t.Fatal("expected bounty not open error")
	}
}

func TestClaimBountyExpired(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	creator := testAddr("alice")
	bk.fundAccount(creator, "ulgm", largeBalance)

	ms.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator: creator, Title: "Bounty", Description: "Desc",
		Domain: "physics", Reward: "1000000000",
		DeadlineHeight: 100 + 34272 + 1,
	})

	// Advance past deadline
	ctx = ctx.WithBlockHeight(100 + 34272 + 2)

	_, err := ms.ClaimBounty(ctx, &types.MsgClaimBounty{
		Claimer: testAddr("bob"), BountyId: "BOUNTY-1",
	})
	if err == nil {
		t.Fatal("expected bounty expired error")
	}
}

func TestFulfillBounty(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	creator := testAddr("alice")
	claimer := testAddr("bob")
	bk.fundAccount(creator, "ulgm", largeBalance)
	bk.fundModule(types.ModuleName, "ulgm", largeBalance) // fund module for payout

	ms.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator: creator, Title: "Bounty", Description: "Desc",
		Domain: "physics", Reward: "1000000000",
		DeadlineHeight: 100 + 34272 + 1,
	})

	ms.ClaimBounty(ctx, &types.MsgClaimBounty{
		Claimer: claimer, BountyId: "BOUNTY-1",
	})

	resp, err := ms.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Authority: authority,
		BountyId:  "BOUNTY-1",
	})
	if err != nil {
		t.Fatalf("FulfillBounty failed: %v", err)
	}
	if resp.Reward != "1000000000" {
		t.Errorf("expected reward=1000000000, got %s", resp.Reward)
	}

	b, _ := k.GetBounty(ctx, "BOUNTY-1")
	if b.Status != types.BountyStatusFulfilled {
		t.Errorf("expected status=fulfilled, got %s", b.Status)
	}
	if b.FulfilledBy != claimer {
		t.Errorf("expected fulfilled_by=%s, got %s", claimer, b.FulfilledBy)
	}
}

func TestFulfillBountyNotClaimed(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	creator := testAddr("alice")
	bk.fundAccount(creator, "ulgm", largeBalance)

	ms.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator: creator, Title: "Bounty", Description: "Desc",
		Domain: "physics", Reward: "1000000000",
		DeadlineHeight: 100 + 34272 + 1,
	})

	// Try to fulfill without claiming
	_, err := ms.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Authority: authority,
		BountyId:  "BOUNTY-1",
	})
	if err == nil {
		t.Fatal("expected bounty not claimed error")
	}
}

func TestFulfillBountyUnauthorized(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	creator := testAddr("alice")
	claimer := testAddr("bob")
	bk.fundAccount(creator, "ulgm", largeBalance)

	ms.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator: creator, Title: "Bounty", Description: "Desc",
		Domain: "physics", Reward: "1000000000",
		DeadlineHeight: 100 + 34272 + 1,
	})

	ms.ClaimBounty(ctx, &types.MsgClaimBounty{
		Claimer: claimer, BountyId: "BOUNTY-1",
	})

	_, err := ms.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Authority: testAddr("rando"),
		BountyId:  "BOUNTY-1",
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

// ---------- Fund Research Treasury Tests ----------

func TestFundResearch(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	funder := testAddr("alice")
	bk.fundAccount(funder, "ulgm", largeBalance)

	resp, err := ms.FundResearch(ctx, &types.MsgFundResearch{
		Funder: funder,
		Amount: "5000000000", // 5000 LGM
	})
	if err != nil {
		t.Fatalf("FundResearch failed: %v", err)
	}
	if resp.TreasuryBalance != "5000000000" {
		t.Errorf("expected treasury=5000000000, got %s", resp.TreasuryBalance)
	}

	// Fund again
	resp2, _ := ms.FundResearch(ctx, &types.MsgFundResearch{
		Funder: funder,
		Amount: "3000000000",
	})
	if resp2.TreasuryBalance != "8000000000" {
		t.Errorf("expected treasury=8000000000, got %s", resp2.TreasuryBalance)
	}
}

// ---------- Genesis Tests ----------

func TestInitExportGenesis(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Manually set some state
	k.SetResearch(ctx, types.Research{
		Id: "RES-42", Submitter: testAddr("alice"),
		Title: "Genesis Research", Domain: "physics",
		Status: types.ResearchStatusSubmitted,
	})
	k.SetBounty(ctx, types.Bounty{
		Id: "BOUNTY-7", Creator: testAddr("bob"),
		Title: "Genesis Bounty", Status: types.BountyStatusOpen,
	})
	k.SetTreasuryBalance(ctx, "999")

	// Export
	gs := k.ExportGenesis(ctx)
	if len(gs.Submissions) != 1 {
		t.Errorf("expected 1 submission, got %d", len(gs.Submissions))
	}
	if len(gs.Bounties) != 1 {
		t.Errorf("expected 1 bounty, got %d", len(gs.Bounties))
	}
	if gs.TreasuryBalance != "999" {
		t.Errorf("expected treasury=999, got %s", gs.TreasuryBalance)
	}
}

// ---------- Query Tests ----------

func TestQueryResearch(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)
	qs := newQueryServer(k)

	submitter := testAddr("alice")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter: submitter, Title: "Test", Description: "Desc",
		Domain: "physics", ResearchType: types.ResearchTypeReplication,
		Stake: "100000000",
	})

	// Add a review
	ms.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer: testAddr("rev1"), ResearchId: "RES-1",
		Verdict: types.ReviewVerdictApprove, Score: 75, Reasoning: "Good",
	})

	resp, err := qs.Research(ctx, &types.QueryResearchRequest{ResearchId: "RES-1"})
	if err != nil {
		t.Fatalf("Query Research failed: %v", err)
	}
	if resp.Research.Id != "RES-1" {
		t.Errorf("expected id=RES-1, got %s", resp.Research.Id)
	}
	if len(resp.Reviews) != 1 {
		t.Errorf("expected 1 review, got %d", len(resp.Reviews))
	}
}

func TestQueryResearchNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := newQueryServer(k)

	_, err := qs.Research(ctx, &types.QueryResearchRequest{ResearchId: "RES-999"})
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestQuerySubmissions(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)
	qs := newQueryServer(k)

	submitter := testAddr("alice")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	// Submit 3 in different domains
	for _, domain := range []string{"physics", "physics", "math"} {
		ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
			Submitter: submitter, Title: "Research " + domain, Description: "Desc",
			Domain: domain, ResearchType: types.ResearchTypeReplication,
			Stake: "100000000",
		})
	}

	// Query all
	resp, _ := qs.Submissions(ctx, &types.QuerySubmissionsRequest{})
	if len(resp.Submissions) != 3 {
		t.Errorf("expected 3 submissions, got %d", len(resp.Submissions))
	}

	// Filter by domain
	resp2, _ := qs.Submissions(ctx, &types.QuerySubmissionsRequest{Domain: "physics"})
	if len(resp2.Submissions) != 2 {
		t.Errorf("expected 2 physics submissions, got %d", len(resp2.Submissions))
	}

	// Filter by status
	resp3, _ := qs.Submissions(ctx, &types.QuerySubmissionsRequest{Status: "submitted"})
	if len(resp3.Submissions) != 3 {
		t.Errorf("expected 3 submitted, got %d", len(resp3.Submissions))
	}
}

func TestQueryBounty(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)
	qs := newQueryServer(k)

	creator := testAddr("alice")
	bk.fundAccount(creator, "ulgm", largeBalance)

	ms.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator: creator, Title: "Query Bounty", Description: "Desc",
		Domain: "physics", Reward: "1000000000",
		DeadlineHeight: 100 + 34272 + 1,
	})

	resp, err := qs.Bounty(ctx, &types.QueryBountyRequest{BountyId: "BOUNTY-1"})
	if err != nil {
		t.Fatalf("Query Bounty failed: %v", err)
	}
	if resp.Bounty.Title != "Query Bounty" {
		t.Errorf("expected title='Query Bounty', got %s", resp.Bounty.Title)
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := newQueryServer(k)

	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("Query Params failed: %v", err)
	}
	if resp.Params.MinReviewerCount != 3 {
		t.Errorf("expected min_reviewer_count=3, got %d", resp.Params.MinReviewerCount)
	}
	if resp.TreasuryBalance != "0" {
		t.Errorf("expected treasury_balance=0, got %s", resp.TreasuryBalance)
	}
}

// ---------- Full Lifecycle Test ----------

func TestFullResearchLifecycle(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	// 1. Submit
	resp, err := ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter: submitter, Title: "Full Lifecycle",
		Description: "Complete test", Domain: "physics",
		ResearchType: types.ResearchTypeReplication,
		Stake:        "100000000",
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	resId := resp.ResearchId

	// 2. Review (3 reviewers)
	for i := 0; i < 3; i++ {
		_, err := ms.ReviewResearch(ctx, &types.MsgReviewResearch{
			Reviewer: testAddr("rev" + string(rune('a'+i))), ResearchId: resId,
			Verdict: types.ReviewVerdictApprove, Score: 85, Reasoning: "Excellent",
		})
		if err != nil {
			t.Fatalf("review %d failed: %v", i, err)
		}
	}

	// 3. Verify under review status
	r, _ := k.GetResearch(ctx, resId)
	if r.Status != types.ResearchStatusUnderReview {
		t.Fatalf("expected under_review, got %s", r.Status)
	}

	// 4. Resolve (should accept with score 85 >= 60)
	resolveResp, err := ms.ResolveResearch(ctx, &types.MsgResolveResearch{
		Authority: authority, ResearchId: resId,
	})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if resolveResp.Outcome != "accepted" {
		t.Errorf("expected accepted, got %s", resolveResp.Outcome)
	}

	// 5. Verify final state
	r, _ = k.GetResearch(ctx, resId)
	if r.Status != types.ResearchStatusAccepted {
		t.Errorf("expected status=accepted, got %s", r.Status)
	}
}

func TestFullBountyLifecycle(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	creator := testAddr("alice")
	claimer := testAddr("bob")
	bk.fundAccount(creator, "ulgm", largeBalance)
	bk.fundModule(types.ModuleName, "ulgm", largeBalance)

	// 1. Create
	resp, err := ms.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator: creator, Title: "Full Bounty", Description: "Complete test",
		Domain: "physics", Reward: "500000000",
		DeadlineHeight: 100 + 34272 + 1,
	})
	if err != nil {
		t.Fatalf("create bounty failed: %v", err)
	}
	bountyId := resp.BountyId

	// 2. Claim
	_, err = ms.ClaimBounty(ctx, &types.MsgClaimBounty{
		Claimer: claimer, BountyId: bountyId,
	})
	if err != nil {
		t.Fatalf("claim bounty failed: %v", err)
	}

	// 3. Fulfill
	fulfillResp, err := ms.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Authority: authority, BountyId: bountyId,
	})
	if err != nil {
		t.Fatalf("fulfill bounty failed: %v", err)
	}
	if fulfillResp.Reward != "500000000" {
		t.Errorf("expected reward=500000000, got %s", fulfillResp.Reward)
	}

	// 4. Verify final state
	b, _ := k.GetBounty(ctx, bountyId)
	if b.Status != types.BountyStatusFulfilled {
		t.Errorf("expected fulfilled, got %s", b.Status)
	}
}

// ---------- State CRUD Tests ----------

func TestIterateResearches(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Add 5 research submissions
	for i := 0; i < 5; i++ {
		k.SetResearch(ctx, types.Research{
			Id:     "RES-" + string(rune('A'+i)),
			Domain: "test",
			Status: types.ResearchStatusSubmitted,
		})
	}

	var count int
	k.IterateResearches(ctx, func(r types.Research) bool {
		count++
		return false
	})
	if count != 5 {
		t.Errorf("expected 5 researches, iterated %d", count)
	}
}

func TestGetResearchesByStatus(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetResearch(ctx, types.Research{Id: "RES-1", Status: types.ResearchStatusSubmitted})
	k.SetResearch(ctx, types.Research{Id: "RES-2", Status: types.ResearchStatusAccepted})
	k.SetResearch(ctx, types.Research{Id: "RES-3", Status: types.ResearchStatusSubmitted})

	submitted := k.GetResearchesByStatus(ctx, types.ResearchStatusSubmitted)
	if len(submitted) != 2 {
		t.Errorf("expected 2 submitted, got %d", len(submitted))
	}

	accepted := k.GetResearchesByStatus(ctx, types.ResearchStatusAccepted)
	if len(accepted) != 1 {
		t.Errorf("expected 1 accepted, got %d", len(accepted))
	}
}

func TestGetResearchesByDomain(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetResearch(ctx, types.Research{Id: "RES-1", Domain: "physics"})
	k.SetResearch(ctx, types.Research{Id: "RES-2", Domain: "math"})
	k.SetResearch(ctx, types.Research{Id: "RES-3", Domain: "physics"})

	physics := k.GetResearchesByDomain(ctx, "physics")
	if len(physics) != 2 {
		t.Errorf("expected 2 physics, got %d", len(physics))
	}
}

func TestGetActiveBounties(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetBounty(ctx, types.Bounty{Id: "B-1", Status: types.BountyStatusOpen})
	k.SetBounty(ctx, types.Bounty{Id: "B-2", Status: types.BountyStatusClaimed})
	k.SetBounty(ctx, types.Bounty{Id: "B-3", Status: types.BountyStatusExpired})
	k.SetBounty(ctx, types.Bounty{Id: "B-4", Status: types.BountyStatusFulfilled})

	active := k.GetActiveBounties(ctx)
	if len(active) != 2 {
		t.Errorf("expected 2 active bounties (open+claimed), got %d", len(active))
	}
}

func TestDeleteResearch(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetResearch(ctx, types.Research{Id: "RES-1", Domain: "test"})
	_, found := k.GetResearch(ctx, "RES-1")
	if !found {
		t.Fatal("expected research to exist")
	}

	k.DeleteResearch(ctx, "RES-1")
	_, found = k.GetResearch(ctx, "RES-1")
	if found {
		t.Fatal("expected research to be deleted")
	}
}

func TestTreasuryBalance(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	bal := k.GetTreasuryBalance(ctx)
	if bal != "0" {
		t.Errorf("expected initial treasury=0, got %s", bal)
	}

	k.SetTreasuryBalance(ctx, "12345")
	bal = k.GetTreasuryBalance(ctx)
	if bal != "12345" {
		t.Errorf("expected treasury=12345, got %s", bal)
	}
}
