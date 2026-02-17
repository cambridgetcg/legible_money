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
		runtime.NewKVStoreService(storeKey),
		"authority",
		nil, // stakingKeeper
		nil, // bankKeeper
		nil, // upgradeKeeper
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

// extendedMsgServer extends the proto-generated MsgServer with hand-written
// methods that are not part of the proto definition.
type extendedMsgServer interface {
	types.MsgServer
	UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error)
	AttachUpgradePlan(ctx context.Context, msg *types.MsgAttachUpgradePlan) (*types.MsgAttachUpgradePlanResponse, error)
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
		LipStakeConsensus:          "500000000",
		LipStakeEconomics:          "400000000",
		LipStakeInterface:          "200000000",
		LipStakeProcess:            "100000000",
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
		Proposer:     testAddr("alice"),
		Title:        "Test LIP",
		Description:  "A test improvement proposal",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
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
	if lip.Stage != string(types.StageDraft) {
		t.Errorf("expected draft stage, got %s", lip.Stage)
	}
	if lip.Proposer != testAddr("alice") {
		t.Errorf("wrong proposer")
	}
}

func TestSubmitLIP_IncrementingIds(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	resp1, _ := ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "First LIP",
		Description:  "First",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})
	resp2, _ := ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("bob"),
		Title:        "Second LIP",
		Description:  "Second",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})

	if resp1.LipId != "LIP-1" {
		t.Errorf("expected LIP-1, got %s", resp1.LipId)
	}
	if resp2.LipId != "LIP-2" {
		t.Errorf("expected LIP-2, got %s", resp2.LipId)
	}
}

func TestStakeLIP_AdvancesToReview(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Submit a LIP with process category
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Process LIP",
		Description:  "Process improvement",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})

	// Set category manually to "process" for stake test compatibility
	lip, _ := k.GetLIP(ctx, "LIP-1")
	lip.Category = string(types.CategoryProcess)
	k.SetLIP(ctx, lip)

	// Stake enough to advance to review
	_, err := ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Staker: testAddr("bob"),
		LipId:  "LIP-1",
		Amount: "200000000", // 200 LGM
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lip, _ = k.GetLIP(ctx, "LIP-1")
	if lip.Stage != string(types.StageReview) {
		t.Errorf("expected review stage, got %s", lip.Stage)
	}
}

func TestStakeLIP_InsufficientStake(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Consensus LIP",
		Description:  "Big stake needed",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})

	// Set category manually to "consensus"
	lip, _ := k.GetLIP(ctx, "LIP-1")
	lip.Category = string(types.CategoryConsensus)
	k.SetLIP(ctx, lip)

	// Stake 100 LGM - not enough for consensus (needs 1000)
	ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Staker: testAddr("bob"),
		LipId:  "LIP-1",
		Amount: "100000000", // 100 LGM
	})

	lip, _ = k.GetLIP(ctx, "LIP-1")
	if lip.Stage != string(types.StageDraft) {
		t.Errorf("should remain in draft with insufficient stake, got %s", lip.Stage)
	}
}

func TestStakeLIP_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Staker: testAddr("bob"),
		LipId:  "LIP-999",
		Amount: "1000000000",
	})
	if err == nil {
		t.Error("expected error for non-existent LIP")
	}
}

// helper to create a LIP and advance it to voting stage via category manipulation
func createAndAdvanceToVoting(t *testing.T, k keeper.Keeper, ms types.MsgServer, ctx sdk.Context) {
	t.Helper()

	// Set review period to 0 so we can advance immediately in tests
	params := k.GetParams(ctx)
	params.ReviewPeriodBlocks = 0
	k.SetParams(ctx, params)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Test",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})
	// Set category to process for stake threshold
	lip, _ := k.GetLIP(ctx, "LIP-1")
	lip.Category = string(types.CategoryProcess)
	k.SetLIP(ctx, lip)

	ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Staker: testAddr("bob"),
		LipId:  "LIP-1",
		Amount: "200000000",
	})
	if _, err := ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Authority: testAddr("alice"), LipId: "LIP-1"}); err != nil {
		t.Fatalf("advance review->last_call failed: %v", err)
	}
	if _, err := ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Authority: testAddr("alice"), LipId: "LIP-1"}); err != nil {
		t.Fatalf("advance last_call->voting failed: %v", err)
	}
}

func TestAdvanceLIPStage_ReviewToLastCall(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Set review period to 0 for immediate advancement
	params := k.GetParams(ctx)
	params.ReviewPeriodBlocks = 0
	k.SetParams(ctx, params)

	// Create and advance to review
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Test",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})
	lip, _ := k.GetLIP(ctx, "LIP-1")
	lip.Category = string(types.CategoryProcess)
	k.SetLIP(ctx, lip)
	ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Staker: testAddr("bob"),
		LipId:  "LIP-1",
		Amount: "200000000",
	})

	// Advance review -> last_call
	_, err := ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{
		Authority: testAddr("alice"),
		LipId:     "LIP-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lip, _ = k.GetLIP(ctx, "LIP-1")
	if lip.Stage != string(types.StageLastCall) {
		t.Errorf("expected last_call, got %s", lip.Stage)
	}
}

func TestAdvanceLIPStage_LastCallToVoting(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Set review period to 0 for immediate advancement
	params := k.GetParams(ctx)
	params.ReviewPeriodBlocks = 0
	k.SetParams(ctx, params)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Test",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})
	lip, _ := k.GetLIP(ctx, "LIP-1")
	lip.Category = string(types.CategoryProcess)
	k.SetLIP(ctx, lip)
	ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Staker: testAddr("bob"),
		LipId:  "LIP-1",
		Amount: "200000000",
	})
	_, err := ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{
		Authority: testAddr("alice"),
		LipId:     "LIP-1",
	})
	if err != nil {
		t.Fatalf("advance review->last_call failed: %v", err)
	}

	// Advance last_call -> voting
	_, err = ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{
		Authority: testAddr("alice"),
		LipId:     "LIP-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lip, _ = k.GetLIP(ctx, "LIP-1")
	if lip.Stage != string(types.StageVoting) {
		t.Errorf("expected voting, got %s", lip.Stage)
	}
	if lip.VotingEndsAt != 100+100000 {
		t.Errorf("expected voting_ends_at=%d, got %d", 100+100000, lip.VotingEndsAt)
	}
}

func TestAdvanceLIPStage_FromDraft_Fails(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Test",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})

	_, err := ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{
		Authority: testAddr("alice"),
		LipId:     "LIP-1",
	})
	if err == nil {
		t.Error("expected error when advancing from draft stage")
	}
}

func TestWithdrawLIP(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Withdraw Me",
		Description:  "Will be withdrawn",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})

	_, err := ms.WithdrawLIP(ctx, &types.MsgWithdrawLIP{
		Proposer: testAddr("alice"),
		LipId:    "LIP-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lip, _ := k.GetLIP(ctx, "LIP-1")
	if lip.Stage != string(types.StageWithdrawn) {
		t.Errorf("expected withdrawn, got %s", lip.Stage)
	}
}

func TestWithdrawLIP_NotProposer(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Test",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
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
		Proposer:     testAddr("alice"),
		Title:        "Test",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
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

func TestCastVote(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Create LIP and advance to voting
	createAndAdvanceToVoting(t, k, ms, ctx)

	// Cast vote with stake (ZD-E05: quadratic voting, power = sqrt(stake))
	resp, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter:      testAddr("voter1"),
		ProposalId: "LIP-1",
		Option:     types.VoteOption_VOTE_OPTION_YES,
		Stake:      "9000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// sqrt(9000000) = 3000
	if resp.EffectiveVotingPower != "3000" {
		t.Errorf("expected effective_voting_power=3000 (sqrt(9000000)), got %s", resp.EffectiveVotingPower)
	}

	// Verify tally updated with quadratic voting power
	lip, _ := k.GetLIP(ctx, "LIP-1")
	if lip.SupportWeight != "3000" {
		t.Errorf("expected support_weight=3000, got %s", lip.SupportWeight)
	}
	if lip.UniqueVoters != 1 {
		t.Errorf("expected 1 unique voter, got %d", lip.UniqueVoters)
	}
}

func TestCastVote_DuplicateVote(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	createAndAdvanceToVoting(t, k, ms, ctx)

	ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter1"), ProposalId: "LIP-1", Option: types.VoteOption_VOTE_OPTION_YES, Stake: "1",
	})

	_, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter1"), ProposalId: "LIP-1", Option: types.VoteOption_VOTE_OPTION_NO, Stake: "1",
	})
	if err == nil {
		t.Error("expected error for duplicate vote")
	}
}

func TestCastVote_NotInVotingStage(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Test",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})

	// LIP is in draft stage
	_, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter1"), ProposalId: "LIP-1", Option: types.VoteOption_VOTE_OPTION_YES, Stake: "1",
	})
	if err == nil {
		t.Error("expected error for voting on draft LIP")
	}
}

func TestCastVote_MultipleVoters(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	createAndAdvanceToVoting(t, k, ms, ctx)

	// ZD-E05: quadratic voting — power = sqrt(stake)
	// sqrt(5)=2, sqrt(2)=1, sqrt(1)=1 (integer sqrt)
	ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter1"), ProposalId: "LIP-1", Option: types.VoteOption_VOTE_OPTION_YES, Stake: "5",
	})
	ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter2"), ProposalId: "LIP-1", Option: types.VoteOption_VOTE_OPTION_NO, Stake: "2",
	})
	ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter3"), ProposalId: "LIP-1", Option: types.VoteOption_VOTE_OPTION_ABSTAIN, Stake: "1",
	})

	lip, _ := k.GetLIP(ctx, "LIP-1")
	if lip.SupportWeight != "2" {
		t.Errorf("expected support=2 (sqrt(5)), got %s", lip.SupportWeight)
	}
	if lip.OpposeWeight != "1" {
		t.Errorf("expected oppose=1 (sqrt(2)), got %s", lip.OpposeWeight)
	}
	if lip.AbstainWeight != "1" {
		t.Errorf("expected abstain=1 (sqrt(1)), got %s", lip.AbstainWeight)
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
		Submitter:   testAddr("alice"),
		Title:       "Fund Infrastructure",
		Description: "Build new nodes",
		Recipient:   testAddr("treasury"),
		Amount:      "50000000000", // 50K LGM
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
	if disb.Stage != string(types.DisbStageVoting) {
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
		Stage:           string(types.DisbStageApproved),
		ApprovedAtBlock: 100,
		TimelockExpiry:  200, // expires at block 200
		SupportWeight:   "100",
		OpposeWeight:    "0",
		AbstainWeight:   "0",
	}
	k.SetDisbursement(ctx, disb)
	k.SetNextDisbNumber(ctx, 2)

	// Try to execute at block 100 - timelock hasn't expired
	_, err := ms.ExecuteDisbursement(ctx, &types.MsgExecuteDisbursement{
		Authority:      testAddr("alice"),
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
		Stage:           string(types.DisbStageApproved),
		ApprovedAtBlock: 50,
		TimelockExpiry:  90, // expires at block 90
	}
	k.SetDisbursement(ctx, disb)

	// Execute at block 100 - timelock expired
	_, err := ms.ExecuteDisbursement(ctx, &types.MsgExecuteDisbursement{
		Authority:      testAddr("alice"),
		DisbursementId: "DISB-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := k.GetDisbursement(ctx, "DISB-1")
	if got.Stage != string(types.DisbStageExecuted) {
		t.Errorf("expected executed, got %s", got.Stage)
	}
}

func TestCancelDisbursement(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitDisbursement(ctx, &types.MsgSubmitDisbursement{
		Submitter:   testAddr("alice"),
		Title:       "Cancel Me",
		Description: "Will be cancelled",
		Recipient:   testAddr("treasury"),
		Amount:      "10000000000",
	})

	_, err := ms.CancelDisbursement(ctx, &types.MsgCancelDisbursement{
		Authority:      testAddr("alice"),
		DisbursementId: "DISB-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	disb, _ := k.GetDisbursement(ctx, "DISB-1")
	if disb.Stage != string(types.DisbStageCancelled) {
		t.Errorf("expected cancelled, got %s", disb.Stage)
	}
}

func TestCancelDisbursement_NotProposer(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitDisbursement(ctx, &types.MsgSubmitDisbursement{
		Submitter:   testAddr("alice"),
		Title:       "Test",
		Description: "Test",
		Recipient:   testAddr("treasury"),
		Amount:      "10000000000",
	})

	_, err := ms.CancelDisbursement(ctx, &types.MsgCancelDisbursement{
		Authority:      testAddr("bob"),
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
		Submitter:   testAddr("alice"),
		Title:       "Test",
		Description: "Test",
		Recipient:   testAddr("treasury"),
		Amount:      "10000000000",
	})

	// ZD-E05: quadratic voting — sqrt(100000000) = 10000
	resp, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter:      testAddr("voter1"),
		ProposalId: "DISB-1",
		Option:     types.VoteOption_VOTE_OPTION_YES,
		Stake:      "100000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.EffectiveVotingPower != "10000" {
		t.Errorf("expected effective_voting_power=10000 (sqrt(100000000)), got %s", resp.EffectiveVotingPower)
	}

	disb, _ := k.GetDisbursement(ctx, "DISB-1")
	if disb.SupportWeight != "10000" {
		t.Errorf("expected support=10000, got %s", disb.SupportWeight)
	}
}

// ---------- UpdateParams Tests ----------

func TestUpdateParams_Authority(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k).(extendedMsgServer)

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
	ms := keeper.NewMsgServerImpl(k).(extendedMsgServer)

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
		Proposer:     testAddr("alice"),
		Title:        "Genesis LIP",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})
	ms.SubmitDisbursement(ctx, &types.MsgSubmitDisbursement{
		Submitter:   testAddr("bob"),
		Title:       "Genesis Disb",
		Description: "Test",
		Recipient:   testAddr("treasury"),
		Amount:      "10000000000",
	})

	// Export
	gs := k.ExportGenesis(ctx)
	if len(gs.Lips) != 1 {
		t.Errorf("expected 1 LIP, got %d", len(gs.Lips))
	}
	if len(gs.Disbursements) != 1 {
		t.Errorf("expected 1 disbursement, got %d", len(gs.Disbursements))
	}
	if gs.NextLipNumber != 2 {
		t.Errorf("expected next_lip=2, got %d", gs.NextLipNumber)
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
		Proposer:     testAddr("alice"),
		Title:        "Query Test",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_ECONOMIC_POLICY,
		InitialStake: "1000000",
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
		Proposer:     testAddr("alice"),
		Title:        "Draft 1",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Draft 2",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})

	// Set first LIP to process category and advance to review
	lip, _ := k.GetLIP(ctx, "LIP-1")
	lip.Category = string(types.CategoryProcess)
	k.SetLIP(ctx, lip)
	ms.StakeLIP(ctx, &types.MsgStakeLIP{Staker: testAddr("bob"), LipId: "LIP-1", Amount: "200000000"})

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
		Submitter:   testAddr("alice"),
		Title:       "Query Disb",
		Description: "Test",
		Recipient:   testAddr("treasury"),
		Amount:      "5000000000",
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
		{"valid", types.MsgSubmitLIP{Proposer: testAddr("alice"), Title: "Test", Description: "Test", LipType: types.LIPType_LIP_TYPE_COMMUNITY, InitialStake: "1000000"}, false},
		{"empty title", types.MsgSubmitLIP{Proposer: testAddr("alice"), Title: "", Description: "Test", LipType: types.LIPType_LIP_TYPE_COMMUNITY, InitialStake: "1000000"}, true},
		{"invalid addr", types.MsgSubmitLIP{Proposer: "bad", Title: "Test", Description: "Test", LipType: types.LIPType_LIP_TYPE_COMMUNITY, InitialStake: "1000000"}, true},
		{"zero stake", types.MsgSubmitLIP{Proposer: testAddr("alice"), Title: "Test", Description: "Test", LipType: types.LIPType_LIP_TYPE_COMMUNITY, InitialStake: "0"}, true},
		{"empty stake", types.MsgSubmitLIP{Proposer: testAddr("alice"), Title: "Test", Description: "Test", LipType: types.LIPType_LIP_TYPE_COMMUNITY, InitialStake: ""}, true},
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
		{"valid", types.MsgCastVote{Voter: testAddr("v"), ProposalId: "LIP-1", Option: types.VoteOption_VOTE_OPTION_YES, Stake: "1"}, false},
		{"invalid addr", types.MsgCastVote{Voter: "bad", ProposalId: "LIP-1", Option: types.VoteOption_VOTE_OPTION_YES, Stake: "1"}, true},
		{"empty proposal", types.MsgCastVote{Voter: testAddr("v"), ProposalId: "", Option: types.VoteOption_VOTE_OPTION_YES, Stake: "1"}, true},
		{"unspecified option", types.MsgCastVote{Voter: testAddr("v"), ProposalId: "LIP-1", Option: types.VoteOption_VOTE_OPTION_UNSPECIFIED, Stake: "1"}, true},
		{"empty stake", types.MsgCastVote{Voter: testAddr("v"), ProposalId: "LIP-1", Option: types.VoteOption_VOTE_OPTION_YES, Stake: ""}, true},
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

// ---------- Upgrade Plan Tests ----------

func TestAttachUpgradePlan(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k).(extendedMsgServer)

	// Submit a LIP and set to consensus category
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Upgrade v2",
		Description:  "Software upgrade to v2",
		LipType:      types.LIPType_LIP_TYPE_SOFTWARE_UPGRADE,
		InitialStake: "1000000",
	})
	lip, _ := k.GetLIP(ctx, "LIP-1")
	lip.Category = string(types.CategoryConsensus)
	k.SetLIP(ctx, lip)

	// Attach upgrade plan
	_, err := ms.AttachUpgradePlan(ctx, &types.MsgAttachUpgradePlan{
		Proposer:    testAddr("alice"),
		LipId:       "LIP-1",
		UpgradeName: "v2",
		Height:      500000,
		Info:        "https://example.com/v2-binaries",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify plan stored
	plan, found := k.GetUpgradePlan(ctx, "LIP-1")
	if !found {
		t.Fatal("upgrade plan not found")
	}
	if plan.Name != "v2" {
		t.Errorf("expected name=v2, got %s", plan.Name)
	}
	if plan.Height != 500000 {
		t.Errorf("expected height=500000, got %d", plan.Height)
	}
}

func TestAttachUpgradePlan_NotConsensus(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k).(extendedMsgServer)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Process LIP",
		Description:  "Not a consensus LIP",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})
	// Category defaults to LipType int, not "consensus"

	_, err := ms.AttachUpgradePlan(ctx, &types.MsgAttachUpgradePlan{
		Proposer:    testAddr("alice"),
		LipId:       "LIP-1",
		UpgradeName: "v2",
		Height:      500000,
	})
	if err == nil {
		t.Error("expected error for non-consensus category")
	}
}

func TestAttachUpgradePlan_NotProposer(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k).(extendedMsgServer)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Upgrade v2",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_SOFTWARE_UPGRADE,
		InitialStake: "1000000",
	})
	lip, _ := k.GetLIP(ctx, "LIP-1")
	lip.Category = string(types.CategoryConsensus)
	k.SetLIP(ctx, lip)

	_, err := ms.AttachUpgradePlan(ctx, &types.MsgAttachUpgradePlan{
		Proposer:    testAddr("bob"),
		LipId:       "LIP-1",
		UpgradeName: "v2",
		Height:      500000,
	})
	if err == nil {
		t.Error("expected error for non-proposer")
	}
}

func TestAttachUpgradePlan_DuplicatePlan(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k).(extendedMsgServer)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Upgrade v2",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_SOFTWARE_UPGRADE,
		InitialStake: "1000000",
	})
	lip, _ := k.GetLIP(ctx, "LIP-1")
	lip.Category = string(types.CategoryConsensus)
	k.SetLIP(ctx, lip)

	ms.AttachUpgradePlan(ctx, &types.MsgAttachUpgradePlan{
		Proposer:    testAddr("alice"),
		LipId:       "LIP-1",
		UpgradeName: "v2",
		Height:      500000,
	})

	_, err := ms.AttachUpgradePlan(ctx, &types.MsgAttachUpgradePlan{
		Proposer:    testAddr("alice"),
		LipId:       "LIP-1",
		UpgradeName: "v3",
		Height:      600000,
	})
	if err == nil {
		t.Error("expected error for duplicate upgrade plan")
	}
}

func TestAttachUpgradePlan_TerminalLIP(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k).(extendedMsgServer)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Upgrade v2",
		Description:  "Test",
		LipType:      types.LIPType_LIP_TYPE_SOFTWARE_UPGRADE,
		InitialStake: "1000000",
	})
	lip, _ := k.GetLIP(ctx, "LIP-1")
	lip.Category = string(types.CategoryConsensus)
	k.SetLIP(ctx, lip)
	ms.WithdrawLIP(ctx, &types.MsgWithdrawLIP{
		Proposer: testAddr("alice"),
		LipId:    "LIP-1",
	})

	_, err := ms.AttachUpgradePlan(ctx, &types.MsgAttachUpgradePlan{
		Proposer:    testAddr("alice"),
		LipId:       "LIP-1",
		UpgradeName: "v2",
		Height:      500000,
	})
	if err == nil {
		t.Error("expected error for terminal LIP")
	}
}

func TestDeleteUpgradePlan(t *testing.T) {
	k, ctx := setupKeeper(t)

	plan := types.UpgradePlan{Name: "v2", Height: 100000, Info: "test"}
	k.SetUpgradePlan(ctx, "LIP-1", plan)

	_, found := k.GetUpgradePlan(ctx, "LIP-1")
	if !found {
		t.Fatal("plan should exist")
	}

	k.DeleteUpgradePlan(ctx, "LIP-1")

	_, found = k.GetUpgradePlan(ctx, "LIP-1")
	if found {
		t.Error("plan should be deleted")
	}
}

func TestValidateBasic_MsgAttachUpgradePlan(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgAttachUpgradePlan
		wantErr bool
	}{
		{"valid", types.MsgAttachUpgradePlan{Proposer: testAddr("alice"), LipId: "LIP-1", UpgradeName: "v2", Height: 500000}, false},
		{"invalid addr", types.MsgAttachUpgradePlan{Proposer: "bad", LipId: "LIP-1", UpgradeName: "v2", Height: 500000}, true},
		{"empty lip_id", types.MsgAttachUpgradePlan{Proposer: testAddr("alice"), LipId: "", UpgradeName: "v2", Height: 500000}, true},
		{"empty name", types.MsgAttachUpgradePlan{Proposer: testAddr("alice"), LipId: "LIP-1", UpgradeName: "", Height: 500000}, true},
		{"zero height", types.MsgAttachUpgradePlan{Proposer: testAddr("alice"), LipId: "LIP-1", UpgradeName: "v2", Height: 0}, true},
		{"negative height", types.MsgAttachUpgradePlan{Proposer: testAddr("alice"), LipId: "LIP-1", UpgradeName: "v2", Height: -1}, true},
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

// ---------- Genesis Roundtrip Tests ----------

func TestGenesisRoundtrip_LIPFields(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Create a LIP and advance it
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Roundtrip LIP",
		Description:  "Tests field preservation",
		LipType:      types.LIPType_LIP_TYPE_SOFTWARE_UPGRADE,
		InitialStake: "1000000",
	})
	// Set category for stake test
	lip, _ := k.GetLIP(ctx, "LIP-1")
	lip.Category = string(types.CategoryConsensus)
	k.SetLIP(ctx, lip)
	ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Staker: testAddr("bob"),
		LipId:  "LIP-1",
		Amount: "1000000000",
	})

	// Export
	gs := k.ExportGenesis(ctx)

	// Create a fresh keeper and import
	k2, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, *gs)

	// Verify fields survived roundtrip
	lip, found := k2.GetLIP(ctx2, "LIP-1")
	if !found {
		t.Fatal("LIP-1 not found after roundtrip")
	}
	if lip.Title != "Roundtrip LIP" {
		t.Errorf("title: got %q, want %q", lip.Title, "Roundtrip LIP")
	}
	if lip.Abstract != "Tests field preservation" {
		t.Errorf("abstract mismatch")
	}
	if lip.Category != string(types.CategoryConsensus) {
		t.Errorf("category: got %s, want %s", lip.Category, types.CategoryConsensus)
	}
	if lip.Proposer != testAddr("alice") {
		t.Errorf("proposer mismatch")
	}
	if lip.Stage != string(types.StageReview) {
		t.Errorf("stage: got %s, want review", lip.Stage)
	}
	// 1000000 (initial deposit) + 1000000000 (StakeLIP) = 1001000000
	if lip.StakedAmount != "1001000000" {
		t.Errorf("staked_amount: got %s, want 1001000000", lip.StakedAmount)
	}
}

func TestGenesisRoundtrip_VoteFields(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Create LIP and advance to voting
	createAndAdvanceToVoting(t, k, ms, ctx)

	// Cast votes
	ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("voter1"), ProposalId: "LIP-1", Option: types.VoteOption_VOTE_OPTION_YES, Stake: "5",
	})

	// Export and reimport
	gs := k.ExportGenesis(ctx)
	k2, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, *gs)

	// Verify votes survived
	allVotes := k2.GetAllVotes(ctx2)
	if len(allVotes) != 1 {
		t.Fatalf("expected 1 vote, got %d", len(allVotes))
	}
	if allVotes[0].Voter != testAddr("voter1") {
		t.Errorf("voter mismatch")
	}
	if allVotes[0].Direction != string(types.VoteSupport) {
		t.Errorf("direction: got %s, want support", allVotes[0].Direction)
	}
}

func TestGenesisRoundtrip_Disbursement(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitDisbursement(ctx, &types.MsgSubmitDisbursement{
		Submitter:   testAddr("alice"),
		Title:       "Fund Nodes",
		Description: "Infrastructure funding",
		Recipient:   testAddr("treasury"),
		Amount:      "50000000000",
	})

	gs := k.ExportGenesis(ctx)
	k2, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, *gs)

	disb, found := k2.GetDisbursement(ctx2, "DISB-1")
	if !found {
		t.Fatal("DISB-1 not found after roundtrip")
	}
	if disb.Title != "Fund Nodes" {
		t.Errorf("title mismatch")
	}
	if disb.Amount != "50000000000" {
		t.Errorf("amount: got %s, want 50000000000", disb.Amount)
	}
	if disb.RecipientAddr != testAddr("treasury") {
		t.Errorf("recipient mismatch")
	}
}

func TestGenesisRoundtrip_Counters(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Create several LIPs and disbursements to advance counters
	for i := 0; i < 5; i++ {
		ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
			Proposer:     testAddr("alice"),
			Title:        "LIP",
			Description:  "Test",
			LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
			InitialStake: "1000000",
		})
	}
	for i := 0; i < 3; i++ {
		ms.SubmitDisbursement(ctx, &types.MsgSubmitDisbursement{
			Submitter:   testAddr("alice"),
			Title:       "Disb",
			Description: "Test",
			Recipient:   testAddr("treasury"),
			Amount:      "1000000",
		})
	}

	gs := k.ExportGenesis(ctx)
	k2, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, *gs)

	if k2.GetNextLIPNumber(ctx2) != 6 {
		t.Errorf("LIP counter: got %d, want 6", k2.GetNextLIPNumber(ctx2))
	}
	if k2.GetNextDisbNumber(ctx2) != 4 {
		t.Errorf("disb counter: got %d, want 4", k2.GetNextDisbNumber(ctx2))
	}
}

func TestGenesisRoundtrip_Params(t *testing.T) {
	k, ctx := setupKeeper(t)

	custom := types.Params{
		VotingPeriodBlocks:         7777,
		ReviewPeriodBlocks:         3333,
		LastCallPeriodBlocks:       1111,
		BaseCostPerVote:            "2000000",
		MaxVotesPerVoter:           77,
		SupportThresholdBps:        6500,
		QuorumThresholdBps:         1200,
		DisbursementStake:          "3000000000",
		DisbursementTimelockBlocks: 22222,
		DisbursementSupportBps:     8500,
		DisbursementQuorumBps:      2500,
		LipStakeConsensus:          "500000000",
		LipStakeEconomics:          "300000000",
		LipStakeInterface:          "200000000",
		LipStakeProcess:            "100000000",
	}
	k.SetParams(ctx, custom)

	gs := k.ExportGenesis(ctx)
	k2, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, *gs)

	got := k2.GetParams(ctx2)
	if got.VotingPeriodBlocks != 7777 {
		t.Errorf("VotingPeriodBlocks: got %d, want 7777", got.VotingPeriodBlocks)
	}
	if got.MaxVotesPerVoter != 77 {
		t.Errorf("MaxVotesPerVoter: got %d, want 77", got.MaxVotesPerVoter)
	}
	if got.SupportThresholdBps != 6500 {
		t.Errorf("SupportThresholdBps: got %d, want 6500", got.SupportThresholdBps)
	}
	if got.DisbursementTimelockBlocks != 22222 {
		t.Errorf("DisbursementTimelockBlocks: got %d, want 22222", got.DisbursementTimelockBlocks)
	}
}

func TestGenesisRoundtrip_UpgradePlan(t *testing.T) {
	k, ctx := setupKeeper(t)

	plan := types.UpgradePlan{
		Name:   "v3-upgrade",
		Height: 1000000,
		Info:   "https://example.com/v3",
	}
	k.SetUpgradePlan(ctx, "LIP-42", plan)

	got, found := k.GetUpgradePlan(ctx, "LIP-42")
	if !found {
		t.Fatal("upgrade plan not found")
	}
	if got.Name != "v3-upgrade" {
		t.Errorf("name: got %s, want v3-upgrade", got.Name)
	}
	if got.Height != 1000000 {
		t.Errorf("height: got %d, want 1000000", got.Height)
	}
	if got.Info != "https://example.com/v3" {
		t.Errorf("info mismatch")
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
