package gov_test

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

	gov "github.com/legible-money/legible/x/gov"
	"github.com/legible-money/legible/x/gov/keeper"
	"github.com/legible-money/legible/x/gov/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

// mockStakingKeeper implements types.StakingKeeper for quorum tests.
// Returns a configurable total bonded stake for quorum calculations.
type mockStakingKeeper struct {
	totalBondedStake string
}

func (m *mockStakingKeeper) GetValidatorStake(_ context.Context, _ string) (string, error) {
	return "0", nil
}

func (m *mockStakingKeeper) GetTotalBondedStake(_ context.Context) (string, error) {
	return m.totalBondedStake, nil
}

// setupWithStaking creates a keeper with a mock staking keeper, an AppModule, and a context.
// The totalBonded parameter sets the total bonded stake returned by the mock.
func setupWithStaking(t *testing.T, totalBonded string) (keeper.Keeper, gov.AppModule, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	sk := &mockStakingKeeper{totalBondedStake: totalBonded}

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		"authority",
		sk,  // mock staking keeper with real stake values
		nil, // bankKeeper
		nil, // upgradeKeeper
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())
	k.SetParams(ctx, types.DefaultParams())

	am := gov.NewAppModule(cdc, k)

	return k, am, ctx
}

func quorumTestAddr(name string) string {
	addr := sdk.AccAddress([]byte("addr_" + name + "_______________")[:20])
	return addr.String()
}

// createVotingLIP inserts a LIP directly into the store in the "voting" stage
// with pre-set vote weights. VotingEndsAt should be <= current block height
// to trigger tally on the next BeginBlock.
func createVotingLIP(t *testing.T, k keeper.Keeper, ctx sdk.Context, id string, support, oppose, abstain string, votingEndsAt uint64) {
	t.Helper()
	lip := types.LIP{
		Id:            id,
		Title:         "Quorum Test LIP " + id,
		Abstract:      "Testing quorum enforcement",
		Category:      string(types.CategoryProcess),
		Proposer:      quorumTestAddr("proposer"),
		Stage:         string(types.StageVoting),
		SupportWeight: support,
		OpposeWeight:  oppose,
		AbstainWeight: abstain,
		VotingEndsAt:  votingEndsAt,
		UniqueVoters:  1,
	}
	k.SetLIP(ctx, lip)
}

// createVotingDisbursement inserts a disbursement directly into the store in the
// "voting" stage with pre-set vote weights.
func createVotingDisbursement(t *testing.T, k keeper.Keeper, ctx sdk.Context, id string, support, oppose, abstain string, votingEndsAt uint64) {
	t.Helper()
	disb := types.DisbursementProposal{
		Id:            id,
		Title:         "Quorum Test Disbursement " + id,
		Proposer:      quorumTestAddr("proposer"),
		RecipientAddr: quorumTestAddr("recipient"),
		Amount:        "10000000",
		Stage:         string(types.DisbStageVoting),
		SupportWeight: support,
		OpposeWeight:  oppose,
		AbstainWeight: abstain,
		VotingEndsAt:  votingEndsAt,
	}
	k.SetDisbursement(ctx, disb)
}

// ==================== Task 1: Quorum Integration Tests ====================

func TestQuorum_Sufficient_LIPPasses(t *testing.T) {
	// 3 validators × 1000 LGM = 3000 total bonded
	// 2 vote yes (weight 2000) → 2000*10000/3000 = 6666 BPS ≈ 66.67% > 15% quorum
	// 100% support (2000/2000) > 75% → accepted
	k, am, ctx := setupWithStaking(t, "3000")

	createVotingLIP(t, k, ctx, "LIP-1", "2000", "0", "0", 999) // voting ends at block 999, current is 1000

	if err := am.BeginBlock(ctx); err != nil {
		t.Fatalf("BeginBlock failed: %v", err)
	}

	lip, found := k.GetLIP(ctx, "LIP-1")
	if !found {
		t.Fatal("LIP-1 not found after BeginBlock")
	}
	if lip.Stage != string(types.StageAccepted) {
		t.Errorf("expected accepted (quorum 66.67%% > 15%%), got %s", lip.Stage)
	}
	if !lip.QuorumMet {
		t.Error("expected QuorumMet=true")
	}
}

func TestQuorum_Insufficient_LIPFails(t *testing.T) {
	// 10 validators × 1000 LGM = 10000 total bonded
	// 1 votes yes (weight 1000) → 1000*10000/10000 = 1000 BPS = 10% < 15% quorum
	// Even though support is 100%, quorum not met → rejected
	k, am, ctx := setupWithStaking(t, "10000")

	createVotingLIP(t, k, ctx, "LIP-1", "1000", "0", "0", 999)

	if err := am.BeginBlock(ctx); err != nil {
		t.Fatalf("BeginBlock failed: %v", err)
	}

	lip, found := k.GetLIP(ctx, "LIP-1")
	if !found {
		t.Fatal("LIP-1 not found")
	}
	if lip.Stage != string(types.StageRejected) {
		t.Errorf("expected rejected (quorum 10%% < 15%%), got %s", lip.Stage)
	}
	if lip.QuorumMet {
		t.Error("expected QuorumMet=false")
	}
}

func TestQuorum_Disbursement_HigherThreshold(t *testing.T) {
	// totalBonded = 10000
	// Vote weight = 1800 → 1800*10000/10000 = 1800 BPS = 18%
	// LIP quorum = 1500 BPS (15%) → 18% >= 15% → passes
	// Disbursement quorum = 2000 BPS (20%) → 18% < 20% → fails
	k, am, ctx := setupWithStaking(t, "10000")

	createVotingLIP(t, k, ctx, "LIP-1", "1800", "0", "0", 999)
	createVotingDisbursement(t, k, ctx, "DISB-1", "1800", "0", "0", 999)

	if err := am.BeginBlock(ctx); err != nil {
		t.Fatalf("BeginBlock failed: %v", err)
	}

	lip, _ := k.GetLIP(ctx, "LIP-1")
	if lip.Stage != string(types.StageAccepted) {
		t.Errorf("LIP should pass (18%% > 15%% quorum), got %s", lip.Stage)
	}
	if !lip.QuorumMet {
		t.Error("LIP QuorumMet should be true")
	}

	disb, _ := k.GetDisbursement(ctx, "DISB-1")
	if disb.Stage != string(types.DisbStageRejected) {
		t.Errorf("disbursement should fail (18%% < 20%% quorum), got %s", disb.Stage)
	}
	if disb.QuorumMet {
		t.Error("disbursement QuorumMet should be false")
	}
}

func TestQuorum_ZeroBondedStake(t *testing.T) {
	// totalBonded = "0" → checkQuorum returns false (no validators bonded)
	k, am, ctx := setupWithStaking(t, "0")

	createVotingLIP(t, k, ctx, "LIP-1", "1000", "0", "0", 999)

	if err := am.BeginBlock(ctx); err != nil {
		t.Fatalf("BeginBlock failed: %v", err)
	}

	lip, _ := k.GetLIP(ctx, "LIP-1")
	if lip.Stage != string(types.StageRejected) {
		t.Errorf("expected rejected with zero bonded stake, got %s", lip.Stage)
	}
	if lip.QuorumMet {
		t.Error("expected QuorumMet=false with zero bonded stake")
	}
}

func TestQuorum_ZeroThreshold(t *testing.T) {
	// QuorumThresholdBps = 0 → checkQuorum returns true (quorum disabled)
	k, am, ctx := setupWithStaking(t, "10000")

	params := k.GetParams(ctx)
	params.QuorumThresholdBps = 0
	k.SetParams(ctx, params)

	// Even minimal participation should pass when quorum is disabled
	createVotingLIP(t, k, ctx, "LIP-1", "1", "0", "0", 999)

	if err := am.BeginBlock(ctx); err != nil {
		t.Fatalf("BeginBlock failed: %v", err)
	}

	lip, _ := k.GetLIP(ctx, "LIP-1")
	if lip.Stage != string(types.StageAccepted) {
		t.Errorf("expected accepted with zero quorum threshold, got %s", lip.Stage)
	}
}

// ==================== Task 2: Edge Cases ====================

func TestQuorum_ExactlyAtThreshold(t *testing.T) {
	// QuorumThresholdBps = 1500 (15%)
	// totalBonded = 3000, weight = 450 → 450*10000/3000 = 1500 BPS = exactly 15%
	// checkQuorum uses >= so exact boundary passes
	k, am, ctx := setupWithStaking(t, "3000")

	createVotingLIP(t, k, ctx, "LIP-1", "450", "0", "0", 999)

	if err := am.BeginBlock(ctx); err != nil {
		t.Fatalf("BeginBlock failed: %v", err)
	}

	lip, _ := k.GetLIP(ctx, "LIP-1")
	if !lip.QuorumMet {
		t.Error("quorum should be met at exactly 15% (1500 BPS)")
	}
	if lip.Stage != string(types.StageAccepted) {
		t.Errorf("expected accepted at exact threshold, got %s", lip.Stage)
	}
}

func TestQuorum_JustBelowThreshold(t *testing.T) {
	// QuorumThresholdBps = 1500 (15%)
	// totalBonded = 3000, weight = 449 → 449*10000/3000 = 1496 BPS < 1500
	// Should fail quorum
	k, am, ctx := setupWithStaking(t, "3000")

	createVotingLIP(t, k, ctx, "LIP-1", "449", "0", "0", 999)

	if err := am.BeginBlock(ctx); err != nil {
		t.Fatalf("BeginBlock failed: %v", err)
	}

	lip, _ := k.GetLIP(ctx, "LIP-1")
	if lip.QuorumMet {
		t.Error("quorum should NOT be met at 14.96% (1496 BPS < 1500)")
	}
	if lip.Stage != string(types.StageRejected) {
		t.Errorf("expected rejected just below threshold, got %s", lip.Stage)
	}
}

func TestQuorum_LargeStake_BigIntSafe(t *testing.T) {
	// Validators with 10^18 stake — verifies math/big.Int prevents uint64 overflow
	// totalBonded = 10^18, weight = 2×10^17 → 20% > 15% → passes
	k, am, ctx := setupWithStaking(t, "1000000000000000000")

	createVotingLIP(t, k, ctx, "LIP-1", "200000000000000000", "0", "0", 999)

	if err := am.BeginBlock(ctx); err != nil {
		t.Fatalf("BeginBlock failed: %v", err)
	}

	lip, _ := k.GetLIP(ctx, "LIP-1")
	if !lip.QuorumMet {
		t.Error("quorum should be met with large stakes (20% > 15%)")
	}
	if lip.Stage != string(types.StageAccepted) {
		t.Errorf("expected accepted with large stakes, got %s", lip.Stage)
	}
}
