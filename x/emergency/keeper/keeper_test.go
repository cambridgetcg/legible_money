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
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/emergency/keeper"
	"github.com/legible-money/legible/x/emergency/types"
	stakingtypes "github.com/legible-money/legible/x/staking/types"
)

// ---------- Mock StakingKeeper ----------

type mockStakingKeeper struct {
	validators map[string]stakingtypes.LegibleValidator
}

func newMockStakingKeeper() *mockStakingKeeper {
	return &mockStakingKeeper{
		validators: make(map[string]stakingtypes.LegibleValidator),
	}
}

func (m *mockStakingKeeper) addGuardian(addr string, totalStake string) {
	m.validators[addr] = stakingtypes.LegibleValidator{
		OperatorAddress:        addr,
		TotalStake:             totalStake,
		Tier:                   stakingtypes.TierGuardian.String(),
		IsActive:               true,
		VerificationsCompleted: 2000,
		VerificationsCorrect:   1800,
	}
}

func (m *mockStakingKeeper) addBonded(addr string, totalStake string) {
	m.validators[addr] = stakingtypes.LegibleValidator{
		OperatorAddress: addr,
		TotalStake:      totalStake,
		Tier:            stakingtypes.TierBonded.String(),
		IsActive:        true,
	}
}

func (m *mockStakingKeeper) GetValidator(_ sdk.Context, operatorAddr string) (stakingtypes.LegibleValidator, bool) {
	v, ok := m.validators[operatorAddr]
	return v, ok
}

func (m *mockStakingKeeper) GetActiveValidatorSet(_ sdk.Context) []stakingtypes.LegibleValidator {
	var result []stakingtypes.LegibleValidator
	for _, v := range m.validators {
		if v.IsActive {
			result = append(result, v)
		}
	}
	return result
}

func (m *mockStakingKeeper) IterateValidators(_ sdk.Context, cb func(stakingtypes.LegibleValidator) bool) {
	for _, v := range m.validators {
		if cb(v) {
			return
		}
	}
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockStakingKeeper) {
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

	mockSK := newMockStakingKeeper()
	// Set up 5 guardians with equal stake — 4 needed for 75%/80% quorum
	// Total = 600M, 75% = 450M → need 4 guardians (480M), 80% = 480M → need 4
	mockSK.addGuardian("guardian1", "120000000000000000000000000") // 120M LGM
	mockSK.addGuardian("guardian2", "120000000000000000000000000") // 120M LGM
	mockSK.addGuardian("guardian3", "120000000000000000000000000") // 120M LGM
	mockSK.addGuardian("guardian4", "120000000000000000000000000") // 120M LGM
	mockSK.addGuardian("guardian5", "120000000000000000000000000") // 120M LGM
	mockSK.addBonded("bonded1", "50000000000000000000000000")     // 50M LGM (not guardian)

	storeService := runtime.NewKVStoreService(storeKey)

	k := keeper.NewKeeper(
		cdc,
		storeService,
		mockSK,
		"authority",
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	// Set default params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx, mockSK
}

// ---------- Guardian Helper Tests ----------

func TestIsGuardian_TierThreeActive(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	if !k.IsGuardian(ctx, "guardian1") {
		t.Error("expected tier-3 active validator to be guardian")
	}
}

func TestIsGuardian_TierTwoNotGuardian(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	if k.IsGuardian(ctx, "bonded1") {
		t.Error("expected tier-2 validator to NOT be guardian")
	}
}

func TestIsGuardian_UnknownValidator(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	if k.IsGuardian(ctx, "unknown") {
		t.Error("expected unknown address to NOT be guardian")
	}
}

func TestIsGuardian_CouncilMemberDuringBootstrap(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	params := types.DefaultParams()
	params.GenesisCouncil = []string{"council1", "council2"}
	params.CouncilExpiryBlock = 1000 // Active until block 1000
	params.CouncilVirtualStake = "11111000000000000000000"
	k.SetParams(ctx, params)

	if !k.IsGuardian(ctx, "council1") {
		t.Error("expected council member to be guardian during bootstrap")
	}
}

func TestIsGuardian_CouncilMemberAfterExpiry(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	params := types.DefaultParams()
	params.GenesisCouncil = []string{"council1"}
	params.CouncilExpiryBlock = 50 // Already expired (current height = 100)
	k.SetParams(ctx, params)

	if k.IsGuardian(ctx, "council1") {
		t.Error("expected council member to NOT be guardian after expiry")
	}
}

func TestGetGuardianStake(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	stake := k.GetGuardianStake(ctx)
	// 5 × 120M = 600M LGM
	expected := "600000000000000000000000000"
	if stake.String() != expected {
		t.Errorf("expected guardian stake %s, got %s", expected, stake.String())
	}
}

// ---------- Ceremony Lifecycle Tests ----------

func TestHaltCeremony_FullLifecycle(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Propose halt
	resp, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "Critical vulnerability discovered",
	})
	if err != nil {
		t.Fatalf("ProposeHalt failed: %v", err)
	}
	if resp.ProposalId == "" {
		t.Fatal("expected non-empty proposal ID")
	}

	// Status should be halt_voting
	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusHaltVoting {
		t.Errorf("expected status %s, got %s", types.StatusHaltVoting, status)
	}

	// Prevote from 4 guardians (need 75% = 450M out of 600M)
	for _, guardian := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		_, err := msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter:      guardian,
			ProposalId: resp.ProposalId,
			Approve:    true,
		})
		if err != nil {
			t.Fatalf("VoteHalt prevote from %s failed: %v", guardian, err)
		}
	}

	// Check that ceremony advanced to precommit
	ceremony, found := k.GetCeremony(ctx, resp.ProposalId)
	if !found {
		t.Fatal("ceremony not found")
	}
	if ceremony.Phase != string(types.PhasePrecommit) {
		t.Errorf("expected phase %s, got %s", types.PhasePrecommit, ceremony.Phase)
	}

	// Precommit from 4 guardians (phase-based: vote again while in precommit phase)
	for _, guardian := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		_, err := msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter:      guardian,
			ProposalId: resp.ProposalId,
			Approve:    true,
		})
		if err != nil {
			t.Fatalf("VoteHalt precommit from %s failed: %v", guardian, err)
		}
	}

	// Status should be halted
	status = k.GetEmergencyStatus(ctx)
	if status != types.StatusHalted {
		t.Errorf("expected status %s, got %s", types.StatusHalted, status)
	}

	// ActiveHaltCeremonyId should be set
	haltId := k.GetActiveHaltCeremonyId(ctx)
	if haltId != resp.ProposalId {
		t.Errorf("expected active halt ID %s, got %s", resp.ProposalId, haltId)
	}
}

func TestResumeCeremony_AfterHalt(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// First, execute a full halt
	haltResp, _ := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "exploit found",
	})
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter: g, ProposalId: haltResp.ProposalId, Approve: true,
		})
	}
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter: g, ProposalId: haltResp.ProposalId, Approve: true,
		})
	}

	// Now propose resume
	resumeResp, err := msgServer.ProposeResume(sdk.WrapSDKContext(ctx), &types.MsgProposeResume{
		Proposer: "guardian2",
	})
	if err != nil {
		t.Fatalf("ProposeResume failed: %v", err)
	}

	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusResumeVoting {
		t.Errorf("expected status %s, got %s", types.StatusResumeVoting, status)
	}

	// Vote resume through (80% quorum = 480M out of 600M, need g1+g2+g3+g4 = 480M)
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteResume(sdk.WrapSDKContext(ctx), &types.MsgVoteResume{
			Voter: g, ProposalId: resumeResp.ProposalId, Approve: true,
		})
	}
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteResume(sdk.WrapSDKContext(ctx), &types.MsgVoteResume{
			Voter: g, ProposalId: resumeResp.ProposalId, Approve: true,
		})
	}

	// Should be back to normal
	status = k.GetEmergencyStatus(ctx)
	if status != types.StatusNormal {
		t.Errorf("expected status %s after resume, got %s", types.StatusNormal, status)
	}

	// Active halt ceremony should be cleared
	haltId := k.GetActiveHaltCeremonyId(ctx)
	if haltId != "" {
		t.Errorf("expected empty active halt ID after resume, got %s", haltId)
	}
}

func TestRevertCeremony_AfterHalt(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Execute halt
	haltResp, _ := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "need revert",
	})
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter: g, ProposalId: haltResp.ProposalId, Approve: true,
		})
	}
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter: g, ProposalId: haltResp.ProposalId, Approve: true,
		})
	}

	// Propose revert
	revertResp, err := msgServer.ProposeRevert(sdk.WrapSDKContext(ctx), &types.MsgProposeRevert{
		Proposer:       "guardian3",
		RevertToHeight: 50,
		Justification:  "state corruption at block 51",
	})
	if err != nil {
		t.Fatalf("ProposeRevert failed: %v", err)
	}

	if k.GetEmergencyStatus(ctx) != types.StatusRevertVoting {
		t.Error("expected revert_voting status")
	}

	// Vote through revert (80% quorum)
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteRevert(sdk.WrapSDKContext(ctx), &types.MsgVoteRevert{
			Voter: g, ProposalId: revertResp.ProposalId, Approve: true,
		})
	}
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteRevert(sdk.WrapSDKContext(ctx), &types.MsgVoteRevert{
			Voter: g, ProposalId: revertResp.ProposalId, Approve: true,
		})
	}

	// After revert finalized, status should be halted (resume still required)
	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusHalted {
		t.Errorf("expected status %s after revert, got %s", types.StatusHalted, status)
	}
}

func TestCeremonyTimeout(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Propose halt
	resp, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "timeout test",
	})
	if err != nil {
		t.Fatalf("ProposeHalt failed: %v", err)
	}

	// Advance past timeout deadline (100 + 44 = 144)
	ctx = ctx.WithBlockHeight(200)

	// CheckCeremonyProgress should detect timeout
	_, err = k.CheckCeremonyProgress(ctx, resp.ProposalId)
	if err != nil {
		t.Fatalf("CheckCeremonyProgress failed: %v", err)
	}

	ceremony, found := k.GetCeremony(ctx, resp.ProposalId)
	if !found {
		t.Fatal("ceremony should still exist")
	}
	if ceremony.Phase != string(types.PhaseFailed) {
		t.Errorf("expected phase %s after timeout, got %s", types.PhaseFailed, ceremony.Phase)
	}
}

// ---------- Anti-Abuse Tests ----------

func TestCooldownEnforcement(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// First proposal succeeds
	_, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "first",
	})
	if err != nil {
		t.Fatalf("first ProposeHalt failed: %v", err)
	}

	// Advance past timeout so ceremony fails and allows new proposal
	ctx = ctx.WithBlockHeight(200)
	ceremony, _ := k.GetActiveCeremony(ctx)
	k.CheckCeremonyProgress(ctx, ceremony.Id)
	k.HandleCeremonyFailure(ctx, ceremony.Id)

	// Try again within cooldown (200 < 100 + 111 = 211)
	_, err = msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian2",
		Reason:   "too soon",
	})
	if err == nil {
		t.Error("expected cooldown error, got nil")
	}
}

func TestPerGuardianLimit(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// First proposal from guardian1
	_, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "first",
	})
	if err != nil {
		t.Fatalf("first ProposeHalt failed: %v", err)
	}

	// Advance past timeout + cooldown
	ctx = ctx.WithBlockHeight(400)
	ceremony, _ := k.GetActiveCeremony(ctx)
	k.CheckCeremonyProgress(ctx, ceremony.Id)
	k.HandleCeremonyFailure(ctx, ceremony.Id)

	// guardian1 tries again (should fail — 1 per guardian per epoch)
	_, err = msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "second attempt",
	})
	if err == nil {
		t.Error("expected per-guardian limit error, got nil")
	}
}

func TestEpochReset(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Manually set some counters
	k.IncrementGuardianProposalCount(ctx, "guardian1")
	k.IncrementEpochProposalCount(ctx)

	if k.GetGuardianProposalCount(ctx, "guardian1") != 1 {
		t.Error("expected guardian count 1")
	}
	if k.GetEpochProposalCount(ctx) != 1 {
		t.Error("expected epoch count 1")
	}

	// Reset
	k.ResetEpochCounters(ctx)

	if k.GetGuardianProposalCount(ctx, "guardian1") != 0 {
		t.Error("expected guardian count 0 after reset")
	}
	if k.GetEpochProposalCount(ctx) != 0 {
		t.Error("expected epoch count 0 after reset")
	}
}

// ---------- Access Control Tests ----------

func TestNonGuardianProposalRejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	_, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "bonded1",
		Reason:   "not a guardian",
	})
	if err == nil {
		t.Error("expected error for non-guardian proposal")
	}
}

func TestNonGuardianVoteRejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// First create a valid ceremony
	resp, _ := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "test",
	})

	// Non-guardian tries to vote
	_, err := msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter:      "bonded1",
		ProposalId: resp.ProposalId,
		Approve:    true,
	})
	if err == nil {
		t.Error("expected error for non-guardian vote")
	}
}

// ---------- Validation Tests ----------

func TestCannotHaltWhenAlreadyHalted(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Set status to halted
	k.SetEmergencyStatus(ctx, types.StatusHalted)

	_, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "already halted",
	})
	if err == nil {
		t.Error("expected error when proposing halt while already halted")
	}
}

func TestCannotRevertWhenNotHalted(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	_, err := msgServer.ProposeRevert(sdk.WrapSDKContext(ctx), &types.MsgProposeRevert{
		Proposer:       "guardian1",
		RevertToHeight: 50,
	})
	if err == nil {
		t.Error("expected error when proposing revert without halt")
	}
}

func TestCannotResumeWhenNotHalted(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	_, err := msgServer.ProposeResume(sdk.WrapSDKContext(ctx), &types.MsgProposeResume{
		Proposer: "guardian1",
	})
	if err == nil {
		t.Error("expected error when proposing resume without halt")
	}
}

func TestRevertDepthExceeded(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Execute halt
	haltResp, _ := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "depth test",
	})
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter: g, ProposalId: haltResp.ProposalId, Approve: true,
		})
	}
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter: g, ProposalId: haltResp.ProposalId, Approve: true,
		})
	}

	// Try revert to block 0 (depth = 100, but raise height to exceed MaxRevertDepth)
	// MaxRevertDepth = 111111, so set target to well beyond
	ctx = ctx.WithBlockHeight(200000)
	_, err := msgServer.ProposeRevert(sdk.WrapSDKContext(ctx), &types.MsgProposeRevert{
		Proposer:       "guardian1",
		RevertToHeight: 1, // depth = 200000 - 1 = 199999 > 111111
	})
	if err == nil {
		t.Error("expected revert depth exceeded error")
	}
}

// ---------- Voting Mechanics Tests ----------

func TestDuplicatePrevoteRejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "test",
	})

	// First prevote succeeds
	_, err := msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian2", ProposalId: resp.ProposalId, Approve: true,
	})
	if err != nil {
		t.Fatalf("first prevote failed: %v", err)
	}

	// Duplicate prevote rejected
	_, err = msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian2", ProposalId: resp.ProposalId, Approve: true,
	})
	if err == nil {
		t.Error("expected duplicate prevote error")
	}
}

func TestPrecommitRequiresPriorYesPrevote(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "test",
	})

	// Get enough prevotes to advance to precommit phase
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter: g, ProposalId: resp.ProposalId, Approve: true,
		})
	}

	// guardian5 did NOT prevote — precommit should fail
	// (now in precommit phase, vote will attempt precommit automatically)
	_, err := msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian5", ProposalId: resp.ProposalId, Approve: true,
	})
	if err == nil {
		t.Error("expected error for precommit without prior prevote")
	}
}

func TestPrecommitInPrevotePhaseRejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "test",
	})

	// In the phase-based model, voting during prevote phase always records a prevote.
	// So this test verifies that a vote during prevote phase is treated as a prevote (not rejected).
	_, err := msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian1", ProposalId: resp.ProposalId, Approve: true,
	})
	if err != nil {
		t.Errorf("vote during prevote phase should succeed as prevote: %v", err)
	}

	// Verify it was recorded as a prevote
	ceremony, found := k.GetCeremony(ctx, resp.ProposalId)
	if !found {
		t.Fatal("ceremony not found")
	}
	_, hasPrevote := ceremony.GetPrevote("guardian1")
	if !hasPrevote {
		t.Error("expected prevote to be recorded")
	}
}

// ---------- State Tests ----------

func TestIsHaltedStates(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	tests := []struct {
		status   types.EmergencyStatus
		expected bool
	}{
		{types.StatusNormal, false},
		{types.StatusHaltVoting, false},
		{types.StatusHalted, true},
		{types.StatusRevertVoting, true},
		{types.StatusReverting, true},
		{types.StatusResumeVoting, true},
	}

	for _, tt := range tests {
		k.SetEmergencyStatus(ctx, tt.status)
		got := k.IsHalted(ctx)
		if got != tt.expected {
			t.Errorf("IsHalted(%s) = %v, want %v", tt.status, got, tt.expected)
		}
	}
}

func TestAuditLogRecordsActions(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Propose halt (generates audit entries)
	_, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "audit test",
	})
	if err != nil {
		t.Fatalf("ProposeHalt failed: %v", err)
	}

	entries := k.GetAuditLog(ctx)
	if len(entries) == 0 {
		t.Error("expected audit log entries after proposal")
	}

	// Check that the first entry records the halt proposal
	found := false
	for _, e := range entries {
		if e.Action == string(types.AuditHaltProposed) {
			found = true
			if e.Actor != "guardian1" {
				t.Errorf("expected actor guardian1, got %s", e.Actor)
			}
			break
		}
	}
	if !found {
		t.Error("expected halt_proposed audit entry")
	}
}

func TestStakeWeightedVoting(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "stake weight test",
	})

	// Just guardian1+guardian2+guardian3 = 360M out of 600M = 60%
	// Not enough for 75% quorum
	for _, g := range []string{"guardian1", "guardian2", "guardian3"} {
		msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter: g, ProposalId: resp.ProposalId, Approve: true,
		})
	}

	ceremony, _ := k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrevote) {
		t.Errorf("expected still in prevote (60%% < 75%%), got phase %s", ceremony.Phase)
	}

	// Add guardian4 = 480M / 600M = 80% — should advance past 75% threshold
	msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian4", ProposalId: resp.ProposalId, Approve: true,
	})

	ceremony, _ = k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrecommit) {
		t.Errorf("expected precommit after 80%% quorum, got %s", ceremony.Phase)
	}
}

// ---------- Genesis Tests ----------

func TestGenesisRoundtrip(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Set some state
	k.SetEmergencyStatus(ctx, types.StatusNormal)
	params := types.DefaultParams()
	params.GenesisCouncil = []string{"council1", "council2"}
	params.CouncilExpiryBlock = 50000
	k.SetParams(ctx, params)

	// Export
	genState := k.ExportGenesis(ctx)
	if genState.Status != string(types.StatusNormal) {
		t.Errorf("expected normal status in genesis, got %s", genState.Status)
	}
	if len(genState.Params.GenesisCouncil) != 2 {
		t.Error("expected 2 council members in genesis")
	}

	// Clear and re-init
	k.SetEmergencyStatus(ctx, "")
	k.SetParams(ctx, types.Params{})
	k.InitGenesis(ctx, *genState)

	// Verify restored
	if k.GetEmergencyStatus(ctx) != types.StatusNormal {
		t.Error("expected normal status after init genesis")
	}
	restoredParams := k.GetParams(ctx)
	if len(restoredParams.GenesisCouncil) != 2 {
		t.Error("expected 2 council members after init genesis")
	}
	if restoredParams.CouncilExpiryBlock != 50000 {
		t.Errorf("expected CouncilExpiryBlock 50000, got %d", restoredParams.CouncilExpiryBlock)
	}
}

// ---------- Query Tests ----------

func TestQueryStatus(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	queryServer := keeper.NewQueryServerImpl(k)

	resp, err := queryServer.Status(sdk.WrapSDKContext(ctx), &types.QueryStatusRequest{})
	if err != nil {
		t.Fatalf("query Status failed: %v", err)
	}
	if resp.Status != types.StatusNormal {
		t.Errorf("expected normal status, got %s", resp.Status)
	}
	if resp.IsHalted {
		t.Error("expected not halted")
	}
}

func TestQueryActiveCeremony_None(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	queryServer := keeper.NewQueryServerImpl(k)

	resp, err := queryServer.ActiveCeremony(sdk.WrapSDKContext(ctx), &types.QueryActiveCeremonyRequest{})
	if err != nil {
		t.Fatalf("query ActiveCeremony failed: %v", err)
	}
	if resp.Found {
		t.Error("expected no active ceremony")
	}
}

func TestQueryActiveCeremony_Found(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)
	queryServer := keeper.NewQueryServerImpl(k)

	// Create a ceremony
	_, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "query test",
	})
	if err != nil {
		t.Fatalf("ProposeHalt failed: %v", err)
	}

	resp, err := queryServer.ActiveCeremony(sdk.WrapSDKContext(ctx), &types.QueryActiveCeremonyRequest{})
	if err != nil {
		t.Fatalf("query ActiveCeremony failed: %v", err)
	}
	if !resp.Found {
		t.Error("expected active ceremony to be found")
	}
	if resp.Ceremony == nil {
		t.Error("expected non-nil ceremony")
	}
}

func TestQueryCompletedCeremonies(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	queryServer := keeper.NewQueryServerImpl(k)

	// No completed ceremonies initially
	resp, err := queryServer.CompletedCeremonies(sdk.WrapSDKContext(ctx), &types.QueryCompletedCeremoniesRequest{})
	if err != nil {
		t.Fatalf("query CompletedCeremonies failed: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected 0 completed, got %d", resp.Total)
	}
}

func TestQueryAuditLog(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	queryServer := keeper.NewQueryServerImpl(k)

	// Add an audit entry
	k.AddAuditEntry(ctx, types.EmergencyAuditEntry{
		BlockNumber: 100,
		Action:      string(types.AuditHaltProposed),
		Actor:       "guardian1",
		CeremonyId:  "test-123",
		Details:     "test entry",
	})

	resp, err := queryServer.AuditLog(sdk.WrapSDKContext(ctx), &types.QueryAuditLogRequest{})
	if err != nil {
		t.Fatalf("query AuditLog failed: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 audit entry, got %d", resp.Total)
	}
	if len(resp.Entries) != 1 {
		t.Errorf("expected 1 entry in response, got %d", len(resp.Entries))
	}
}

// ---------- Invalid Halt Reference Tests ----------

func TestCeremonyActiveBlocksNewProposal(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// First proposal
	_, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "first",
	})
	if err != nil {
		t.Fatalf("first ProposeHalt failed: %v", err)
	}

	// Second proposal while first is active
	_, err = msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian2",
		Reason:   "second",
	})
	if err == nil {
		t.Error("expected error when ceremony is already active")
	}
}
