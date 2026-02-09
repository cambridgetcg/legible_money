package keeper_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/gov/keeper"
	"github.com/legible-money/legible/x/gov/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
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

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		"authority",
		nil, // stakingKeeper
		nil, // bankKeeper
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}

func testAddr(name string) string {
	// Generate deterministic bech32 addresses from name
	addr := sdk.AccAddress([]byte("addr_" + name + "_______________")[:20])
	return addr.String()
}

// ---------- Params Tests ----------

func TestSetGetParams(t *testing.T) {
	k, ctx := setupKeeper(t)
	params := k.GetParams(ctx)

	if params.VotingPeriodBlocks != 100000 {
		t.Errorf("expected voting_period_blocks=100000, got %d", params.VotingPeriodBlocks)
	}
	if params.MaxVotesPerVoter != 100 {
		t.Errorf("expected max_votes_per_voter=100, got %d", params.MaxVotesPerVoter)
	}
	if params.SupportThresholdBps != 7500 {
		t.Errorf("expected support_threshold_bps=7500, got %d", params.SupportThresholdBps)
	}
}

func TestSetParams_Custom(t *testing.T) {
	k, ctx := setupKeeper(t)
	custom := types.Params{
		VotingPeriodBlocks:         5000,
		ReviewPeriodBlocks:         2500,
		LastCallPeriodBlocks:       2500,
		BaseCostPerVote:            "500000",
		MaxVotesPerVoter:           50,
		SupportThresholdBps:        6000,
		QuorumThresholdBps:         1000,
		DisbursementStake:          "1000000000",
		DisbursementTimelockBlocks: 25000,
		DisbursementSupportBps:     7000,
		DisbursementQuorumBps:      1500,
		LIPStakeConsensus:          "500000000",
		LIPStakeEconomics:          "400000000",
		LIPStakeInterface:          "200000000",
		LIPStakeProcess:            "100000000",
	}
	k.SetParams(ctx, custom)
	got := k.GetParams(ctx)
	if got.VotingPeriodBlocks != 5000 {
		t.Errorf("expected 5000, got %d", got.VotingPeriodBlocks)
	}
	if got.MaxVotesPerVoter != 50 {
		t.Errorf("expected 50, got %d", got.MaxVotesPerVoter)
	}
}

// ---------- LIP Lifecycle Tests ----------

func TestSubmitLIP(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	resp, err := ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Test LIP",
		Abstract: "A test improvement proposal",
		Category: types.CategoryConsensus,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.LipId != "LIP-1" {
		t.Errorf("expected LIP-1, got %s", resp.LipId)
	}

	lip, found := k.GetLIP(ctx, "LIP-1")
	if !found {
		t.Fatal("LIP-1 not found")
	}
	if lip.Stage != types.StageDraft {
		t.Errorf("expected draft stage, got %s", lip.Stage)
	}
	if lip.Proposer != testAddr("alice") {
		t.Errorf("wrong proposer")
	}
	if lip.Category != types.CategoryConsensus {
		t.Errorf("wrong category")
	}
}

func TestSubmitLIP_IncrementingIds(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	resp1, _ := ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "First LIP",
		Abstract: "First",
		Category: types.CategoryProcess,
	})
	resp2, _ := ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("bob"),
		Title:    "Second LIP",
		Abstract: "Second",
		Category: types.CategoryInterface,
	})

	if resp1.LipId != "LIP-1" {
		t.Errorf("expected LIP-1, got %s", resp1.LipId)
	}
	if resp2.LipId != "LIP-2" {
		t.Errorf("expected LIP-2, got %s", resp2.LipId)
	}
}

func TestSubmitLIP_InvalidCategory(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Bad LIP",
		Abstract: "Invalid category",
		Category: "nonexistent",
	})
	if err == nil {
		t.Error("expected error for invalid category")
	}
}

func TestStakeLIP_AdvancesToReview(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Submit a process LIP (lowest stake: 200 LGM)
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Process LIP",
		Abstract: "Process improvement",
		Category: types.CategoryProcess,
	})

	// Stake enough to advance to review
	_, err := ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Sender: testAddr("bob"),
		LipId:  "LIP-1",
		Amount: "200000000", // 200 LGM
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lip, _ := k.GetLIP(ctx, "LIP-1")
	if lip.Stage != types.StageReview {
		t.Errorf("expected review stage, got %s", lip.Stage)
	}
}

func TestStakeLIP_InsufficientStake(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Consensus LIP",
		Abstract: "Big stake needed",
		Category: types.CategoryConsensus,
	})

	// Stake 100 LGM — not enough for consensus (needs 1000)
	ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Sender: testAddr("bob"),
		LipId:  "LIP-1",
		Amount: "100000000", // 100 LGM
	})

	lip, _ := k.GetLIP(ctx, "LIP-1")
	if lip.Stage != types.StageDraft {
		t.Errorf("should remain in draft with insufficient stake, got %s", lip.Stage)
	}
}

func TestStakeLIP_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Sender: testAddr("bob"),
		LipId:  "LIP-999",
		Amount: "1000000000",
	})
	if err == nil {
		t.Error("expected error for non-existent LIP")
	}
}

func TestAdvanceLIPStage_ReviewToLastCall(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Create and advance to review
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Test",
		Abstract: "Test",
		Category: types.CategoryProcess,
	})
	ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Sender: testAddr("bob"),
		LipId:  "LIP-1",
		Amount: "200000000",
	})

	// Advance review -> last_call
	resp, err := ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{
		Sender: testAddr("alice"),
		LipId:  "LIP-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NewStage != types.StageLastCall {
		t.Errorf("expected last_call, got %s", resp.NewStage)
	}
}

func TestAdvanceLIPStage_LastCallToVoting(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Test",
		Abstract: "Test",
		Category: types.CategoryProcess,
	})
	ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Sender: testAddr("bob"),
		LipId:  "LIP-1",
		Amount: "200000000",
	})
	ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{
		Sender: testAddr("alice"),
		LipId:  "LIP-1",
	})

	// Advance last_call -> voting
	resp, err := ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{
		Sender: testAddr("alice"),
		LipId:  "LIP-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NewStage != types.StageVoting {
		t.Errorf("expected voting, got %s", resp.NewStage)
	}

	lip, _ := k.GetLIP(ctx, "LIP-1")
	if lip.VotingEndsAt != 100+100000 {
		t.Errorf("expected voting_ends_at=%d, got %d", 100+100000, lip.VotingEndsAt)
	}
}

func TestAdvanceLIPStage_FromDraft_Fails(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Test",
		Abstract: "Test",
		Category: types.CategoryProcess,
	})

	_, err := ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{
		Sender: testAddr("alice"),
		LipId:  "LIP-1",
	})
	if err == nil {
		t.Error("expected error when advancing from draft stage")
	}
}

func TestWithdrawLIP(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Withdraw Me",
		Abstract: "Will be withdrawn",
		Category: types.CategoryProcess,
	})

	_, err := ms.WithdrawLIP(ctx, &types.MsgWithdrawLIP{
		Proposer: testAddr("alice"),
		LipId:    "LIP-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lip, _ := k.GetLIP(ctx, "LIP-1")
	if lip.Stage != types.StageWithdrawn {
		t.Errorf("expected withdrawn, got %s", lip.Stage)
	}
}

func TestWithdrawLIP_NotProposer(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Test",
		Abstract: "Test",
		Category: types.CategoryProcess,
	})

	_, err := ms.WithdrawLIP(ctx, &types.MsgWithdrawLIP{
		Proposer: testAddr("bob"),
		LipId:    "LIP-1",
	})
	if err == nil {
		t.Error("expected error when non-proposer withdraws")
	}
}

func TestWithdrawLIP_AlreadyTerminal(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Test",
		Abstract: "Test",
		Category: types.CategoryProcess,
	})
	ms.WithdrawLIP(ctx, &types.MsgWithdrawLIP{
		Proposer: testAddr("alice"),
		LipId:    "LIP-1",
	})

	_, err := ms.WithdrawLIP(ctx, &types.MsgWithdrawLIP{
		Proposer: testAddr("alice"),
		LipId:    "LIP-1",
	})
	if err == nil {
		t.Error("expected error when withdrawing already withdrawn LIP")
	}
}

// ---------- Quadratic Voting Tests ----------

func TestCastVote_QuadraticCost(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Create LIP and advance to voting
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Voting Test",
		Abstract: "Test",
		Category: types.CategoryProcess,
	})
	ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Sender: testAddr("bob"),
		LipId:  "LIP-1",
		Amount: "200000000",
	})
	ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Sender: testAddr("alice"), LipId: "LIP-1"})
	ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Sender: testAddr("alice"), LipId: "LIP-1"})

	// Cast 3 votes → cost = 3^2 * 1e6 = 9e6
	resp, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter:      testAddr("voter1"),
		ProposalId: "LIP-1",
		Direction:  types.VoteSupport,
		Votes:      3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Cost != "9000000" {
		t.Errorf("expected cost=9000000, got %s", resp.Cost)
	}

	// Verify tally updated
	lip, _ := k.GetLIP(ctx, "LIP-1")
	if lip.SupportWeight != "3" {
		t.Errorf("expected support_weight=3, got %s", lip.SupportWeight)
	}
	if lip.UniqueVoters != 1 {
		t.Errorf("expected 1 unique voter, got %d", lip.UniqueVoters)
	}
}

func TestCastVote_DuplicateVote(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Test",
		Abstract: "Test",
		Category: types.CategoryProcess,
	})
	ms.StakeLIP(ctx, &types.MsgStakeLIP{Sender: testAddr("bob"), LipId: "LIP-1", Amount: "200000000"})
	ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Sender: testAddr("alice"), LipId: "LIP-1"})
	ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Sender: testAddr("alice"), LipId: "LIP-1"})

	ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter1"), ProposalId: "LIP-1", Direction: types.VoteSupport, Votes: 1,
	})

	_, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter1"), ProposalId: "LIP-1", Direction: types.VoteOppose, Votes: 1,
	})
	if err == nil {
		t.Error("expected error for duplicate vote")
	}
}

func TestCastVote_ExceedsMaxVotes(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Test",
		Abstract: "Test",
		Category: types.CategoryProcess,
	})
	ms.StakeLIP(ctx, &types.MsgStakeLIP{Sender: testAddr("bob"), LipId: "LIP-1", Amount: "200000000"})
	ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Sender: testAddr("alice"), LipId: "LIP-1"})
	ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Sender: testAddr("alice"), LipId: "LIP-1"})

	_, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter1"), ProposalId: "LIP-1", Direction: types.VoteSupport, Votes: 101,
	})
	if err == nil {
		t.Error("expected error for exceeding max votes")
	}
}

func TestCastVote_NotInVotingStage(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Test",
		Abstract: "Test",
		Category: types.CategoryProcess,
	})

	// LIP is in draft stage
	_, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter1"), ProposalId: "LIP-1", Direction: types.VoteSupport, Votes: 1,
	})
	if err == nil {
		t.Error("expected error for voting on draft LIP")
	}
}

func TestCastVote_MultipleVoters(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Test",
		Abstract: "Test",
		Category: types.CategoryProcess,
	})
	ms.StakeLIP(ctx, &types.MsgStakeLIP{Sender: testAddr("bob"), LipId: "LIP-1", Amount: "200000000"})
	ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Sender: testAddr("alice"), LipId: "LIP-1"})
	ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Sender: testAddr("alice"), LipId: "LIP-1"})

	ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter1"), ProposalId: "LIP-1", Direction: types.VoteSupport, Votes: 5,
	})
	ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter2"), ProposalId: "LIP-1", Direction: types.VoteOppose, Votes: 2,
	})
	ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter3"), ProposalId: "LIP-1", Direction: types.VoteAbstain, Votes: 1,
	})

	lip, _ := k.GetLIP(ctx, "LIP-1")
	if lip.SupportWeight != "5" {
		t.Errorf("expected support=5, got %s", lip.SupportWeight)
	}
	if lip.OpposeWeight != "2" {
		t.Errorf("expected oppose=2, got %s", lip.OpposeWeight)
	}
	if lip.AbstainWeight != "1" {
		t.Errorf("expected abstain=1, got %s", lip.AbstainWeight)
	}
	if lip.UniqueVoters != 3 {
		t.Errorf("expected 3 voters, got %d", lip.UniqueVoters)
	}
}

// ---------- Disbursement Tests ----------

func TestSubmitDisbursement(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	resp, err := ms.SubmitDisbursement(ctx, &types.MsgSubmitDisbursement{
		Proposer:      testAddr("alice"),
		Title:         "Fund Infrastructure",
		Description:   "Build new nodes",
		Category:      types.DisbInfrastructure,
		RecipientAddr: testAddr("treasury"),
		Amount:        "50000000000", // 50K LGM
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.DisbursementId != "DISB-1" {
		t.Errorf("expected DISB-1, got %s", resp.DisbursementId)
	}

	disb, found := k.GetDisbursement(ctx, "DISB-1")
	if !found {
		t.Fatal("DISB-1 not found")
	}
	if disb.Stage != types.DisbStageVoting {
		t.Errorf("expected voting stage, got %s", disb.Stage)
	}
	if disb.VotingEndsAt != 100+100000 {
		t.Errorf("expected voting_ends_at=%d, got %d", 100+100000, disb.VotingEndsAt)
	}
}

func TestExecuteDisbursement_TimelockNotExpired(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Manually create an approved disbursement with future timelock
	disb := types.DisbursementProposal{
		Id:              "DISB-1",
		Title:           "Test",
		Proposer:        testAddr("alice"),
		RecipientAddr:   testAddr("treasury"),
		Amount:          "10000000000",
		Stage:           types.DisbStageApproved,
		ApprovedAtBlock: 100,
		TimelockExpiry:  200, // expires at block 200
		SupportWeight:   "100",
		OpposeWeight:    "0",
		AbstainWeight:   "0",
	}
	k.SetDisbursement(ctx, disb)
	k.SetNextDisbNumber(ctx, 2)

	// Try to execute at block 100 — timelock hasn't expired
	_, err := ms.ExecuteDisbursement(ctx, &types.MsgExecuteDisbursement{
		Sender:         testAddr("alice"),
		DisbursementId: "DISB-1",
	})
	if err == nil {
		t.Error("expected timelock error")
	}
}

func TestExecuteDisbursement_AfterTimelock(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	disb := types.DisbursementProposal{
		Id:              "DISB-1",
		Title:           "Test",
		Proposer:        testAddr("alice"),
		RecipientAddr:   testAddr("treasury"),
		Amount:          "10000000000",
		Stage:           types.DisbStageApproved,
		ApprovedAtBlock: 50,
		TimelockExpiry:  90, // expires at block 90
	}
	k.SetDisbursement(ctx, disb)

	// Execute at block 100 — timelock expired
	_, err := ms.ExecuteDisbursement(ctx, &types.MsgExecuteDisbursement{
		Sender:         testAddr("alice"),
		DisbursementId: "DISB-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := k.GetDisbursement(ctx, "DISB-1")
	if got.Stage != types.DisbStageExecuted {
		t.Errorf("expected executed, got %s", got.Stage)
	}
}

func TestCancelDisbursement(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitDisbursement(ctx, &types.MsgSubmitDisbursement{
		Proposer:      testAddr("alice"),
		Title:         "Cancel Me",
		Description:   "Will be cancelled",
		Category:      types.DisbResearch,
		RecipientAddr: testAddr("treasury"),
		Amount:        "10000000000",
	})

	_, err := ms.CancelDisbursement(ctx, &types.MsgCancelDisbursement{
		Proposer:       testAddr("alice"),
		DisbursementId: "DISB-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	disb, _ := k.GetDisbursement(ctx, "DISB-1")
	if disb.Stage != types.DisbStageCancelled {
		t.Errorf("expected cancelled, got %s", disb.Stage)
	}
}

func TestCancelDisbursement_NotProposer(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitDisbursement(ctx, &types.MsgSubmitDisbursement{
		Proposer:      testAddr("alice"),
		Title:         "Test",
		Description:   "Test",
		Category:      types.DisbDevelopment,
		RecipientAddr: testAddr("treasury"),
		Amount:        "10000000000",
	})

	_, err := ms.CancelDisbursement(ctx, &types.MsgCancelDisbursement{
		Proposer:       testAddr("bob"),
		DisbursementId: "DISB-1",
	})
	if err == nil {
		t.Error("expected error for non-proposer cancel")
	}
}

func TestCastVote_OnDisbursement(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitDisbursement(ctx, &types.MsgSubmitDisbursement{
		Proposer:      testAddr("alice"),
		Title:         "Test",
		Description:   "Test",
		Category:      types.DisbDevelopment,
		RecipientAddr: testAddr("treasury"),
		Amount:        "10000000000",
	})

	resp, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter:      testAddr("voter1"),
		ProposalId: "DISB-1",
		Direction:  types.VoteSupport,
		Votes:      10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// cost = 10^2 * 1e6 = 100e6
	if resp.Cost != "100000000" {
		t.Errorf("expected cost=100000000, got %s", resp.Cost)
	}

	disb, _ := k.GetDisbursement(ctx, "DISB-1")
	if disb.SupportWeight != "10" {
		t.Errorf("expected support=10, got %s", disb.SupportWeight)
	}
}

// ---------- UpdateParams Tests ----------

func TestUpdateParams_Authority(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	newParams := types.DefaultParams()
	newParams.MaxVotesPerVoter = 50

	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "authority",
		Params:    newParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := k.GetParams(ctx)
	if got.MaxVotesPerVoter != 50 {
		t.Errorf("expected 50, got %d", got.MaxVotesPerVoter)
	}
}

func TestUpdateParams_Unauthorized(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "not_authority",
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Error("expected unauthorized error")
	}
}

// ---------- Genesis Tests ----------

func TestInitExportGenesis(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Create some state
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Genesis LIP",
		Abstract: "Test",
		Category: types.CategoryProcess,
	})
	ms.SubmitDisbursement(ctx, &types.MsgSubmitDisbursement{
		Proposer:      testAddr("bob"),
		Title:         "Genesis Disb",
		Description:   "Test",
		Category:      types.DisbResearch,
		RecipientAddr: testAddr("treasury"),
		Amount:        "10000000000",
	})

	// Export
	gs := k.ExportGenesis(ctx)
	if len(gs.LIPs) != 1 {
		t.Errorf("expected 1 LIP, got %d", len(gs.LIPs))
	}
	if len(gs.Disbursements) != 1 {
		t.Errorf("expected 1 disbursement, got %d", len(gs.Disbursements))
	}
	if gs.NextLIPNumber != 2 {
		t.Errorf("expected next_lip=2, got %d", gs.NextLIPNumber)
	}
	if gs.NextDisbNumber != 2 {
		t.Errorf("expected next_disb=2, got %d", gs.NextDisbNumber)
	}
}

// ---------- Query Tests ----------

func TestQueryLIP(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Query Test",
		Abstract: "Test",
		Category: types.CategoryEconomics,
	})

	resp, err := qs.LIP(ctx, &types.QueryLIPRequest{LipId: "LIP-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Lip.Title != "Query Test" {
		t.Errorf("expected 'Query Test', got '%s'", resp.Lip.Title)
	}
}

func TestQueryLIP_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	_, err := qs.LIP(ctx, &types.QueryLIPRequest{LipId: "LIP-999"})
	if err == nil {
		t.Error("expected error for non-existent LIP")
	}
}

func TestQueryLIPs_ByStage(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Draft 1",
		Abstract: "Test",
		Category: types.CategoryProcess,
	})
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer: testAddr("alice"),
		Title:    "Draft 2",
		Abstract: "Test",
		Category: types.CategoryProcess,
	})

	// Advance first to review
	ms.StakeLIP(ctx, &types.MsgStakeLIP{Sender: testAddr("bob"), LipId: "LIP-1", Amount: "200000000"})

	// Query drafts
	drafts, _ := qs.LIPs(ctx, &types.QueryLIPsRequest{Stage: "draft"})
	if drafts.Total != 1 {
		t.Errorf("expected 1 draft, got %d", drafts.Total)
	}

	// Query reviews
	reviews, _ := qs.LIPs(ctx, &types.QueryLIPsRequest{Stage: "review"})
	if reviews.Total != 1 {
		t.Errorf("expected 1 review, got %d", reviews.Total)
	}

	// Query all
	all, _ := qs.LIPs(ctx, &types.QueryLIPsRequest{})
	if all.Total != 2 {
		t.Errorf("expected 2 total, got %d", all.Total)
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Params.VotingPeriodBlocks != 100000 {
		t.Errorf("expected 100000, got %d", resp.Params.VotingPeriodBlocks)
	}
}

func TestQueryDisbursement(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	ms.SubmitDisbursement(ctx, &types.MsgSubmitDisbursement{
		Proposer:      testAddr("alice"),
		Title:         "Query Disb",
		Description:   "Test",
		Category:      types.DisbDomain,
		RecipientAddr: testAddr("treasury"),
		Amount:        "5000000000",
	})

	resp, err := qs.Disbursement(ctx, &types.QueryDisbursementRequest{DisbursementId: "DISB-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Disbursement.Title != "Query Disb" {
		t.Errorf("expected 'Query Disb', got '%s'", resp.Disbursement.Title)
	}
}

// ---------- ValidateBasic Tests ----------

func TestValidateBasic_MsgSubmitLIP(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgSubmitLIP
		wantErr bool
	}{
		{"valid", types.MsgSubmitLIP{Proposer: testAddr("alice"), Title: "Test", Abstract: "Test", Category: types.CategoryProcess}, false},
		{"empty title", types.MsgSubmitLIP{Proposer: testAddr("alice"), Title: "", Abstract: "Test", Category: types.CategoryProcess}, true},
		{"invalid addr", types.MsgSubmitLIP{Proposer: "bad", Title: "Test", Abstract: "Test", Category: types.CategoryProcess}, true},
		{"invalid category", types.MsgSubmitLIP{Proposer: testAddr("alice"), Title: "Test", Abstract: "Test", Category: "bad"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBasic_MsgCastVote(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgCastVote
		wantErr bool
	}{
		{"valid", types.MsgCastVote{Voter: testAddr("v"), ProposalId: "LIP-1", Direction: types.VoteSupport, Votes: 1}, false},
		{"invalid addr", types.MsgCastVote{Voter: "bad", ProposalId: "LIP-1", Direction: types.VoteSupport, Votes: 1}, true},
		{"empty proposal", types.MsgCastVote{Voter: testAddr("v"), ProposalId: "", Direction: types.VoteSupport, Votes: 1}, true},
		{"invalid direction", types.MsgCastVote{Voter: testAddr("v"), ProposalId: "LIP-1", Direction: "bad", Votes: 1}, true},
		{"zero votes", types.MsgCastVote{Voter: testAddr("v"), ProposalId: "LIP-1", Direction: types.VoteSupport, Votes: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ---------- State CRUD Tests ----------

func TestLIPCounter(t *testing.T) {
	k, ctx := setupKeeper(t)

	n := k.GetNextLIPNumber(ctx)
	if n != 1 {
		t.Errorf("expected initial counter=1, got %d", n)
	}

	k.SetNextLIPNumber(ctx, 42)
	n = k.GetNextLIPNumber(ctx)
	if n != 42 {
		t.Errorf("expected 42, got %d", n)
	}
}

func TestDisbCounter(t *testing.T) {
	k, ctx := setupKeeper(t)

	n := k.GetNextDisbNumber(ctx)
	if n != 1 {
		t.Errorf("expected initial counter=1, got %d", n)
	}

	k.SetNextDisbNumber(ctx, 7)
	n = k.GetNextDisbNumber(ctx)
	if n != 7 {
		t.Errorf("expected 7, got %d", n)
	}
}

func TestGetLIPStake_PerCategory(t *testing.T) {
	params := types.DefaultParams()
	if params.GetLIPStake(types.CategoryConsensus) != "1000000000" {
		t.Error("wrong consensus stake")
	}
	if params.GetLIPStake(types.CategoryEconomics) != "800000000" {
		t.Error("wrong economics stake")
	}
	if params.GetLIPStake(types.CategoryInterface) != "400000000" {
		t.Error("wrong interface stake")
	}
	if params.GetLIPStake(types.CategoryProcess) != "200000000" {
		t.Error("wrong process stake")
	}
}
