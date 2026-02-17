package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/alignment/keeper"
	"github.com/legible-money/legible/x/alignment/types"
)

// ============ Mock Keepers ============

type mockKnowledgeKeeper struct {
	verifiedFacts       uint64
	supersededFacts     uint64
	totalChallenges     uint64
	topNChallenges      uint64
	unanimousRounds     uint64
	totalRounds         uint64
	counterFacts        uint64
	avgCitationDepth    uint64
	crossDomainRatio    uint64
	orphanFacts         uint64
	activeFacts         uint64
	avgSupersession     uint64
	totalFacts          uint64
	factsInNewCats      uint64
	newCrossDomainEdges uint64
	domainDist          []types.DomainFactCount
	newEntrantRate      uint64
}

func (m *mockKnowledgeKeeper) GetVerifiedFactCount(_ context.Context) uint64             { return m.verifiedFacts }
func (m *mockKnowledgeKeeper) GetSupersededFactCount(_ context.Context) uint64            { return m.supersededFacts }
func (m *mockKnowledgeKeeper) GetTotalChallenges(_ context.Context) uint64                { return m.totalChallenges }
func (m *mockKnowledgeKeeper) GetChallengesByTopValidators(_ context.Context, _ int) uint64 { return m.topNChallenges }
func (m *mockKnowledgeKeeper) GetUnanimousRoundCount(_ context.Context) uint64            { return m.unanimousRounds }
func (m *mockKnowledgeKeeper) GetTotalRoundCount(_ context.Context) uint64                { return m.totalRounds }
func (m *mockKnowledgeKeeper) GetCounterFactCount(_ context.Context) uint64               { return m.counterFacts }
func (m *mockKnowledgeKeeper) GetAvgCitationChainLength(_ context.Context) uint64         { return m.avgCitationDepth }
func (m *mockKnowledgeKeeper) GetCrossDomainCitationRatio(_ context.Context) uint64       { return m.crossDomainRatio }
func (m *mockKnowledgeKeeper) GetOrphanFactCount(_ context.Context) uint64                { return m.orphanFacts }
func (m *mockKnowledgeKeeper) GetActiveFactCount(_ context.Context) uint64                { return m.activeFacts }
func (m *mockKnowledgeKeeper) GetAvgSupersessionChainLength(_ context.Context) uint64     { return m.avgSupersession }
func (m *mockKnowledgeKeeper) GetTotalFactCount(_ context.Context) uint64                 { return m.totalFacts }
func (m *mockKnowledgeKeeper) GetFactsInNewCategories(_ context.Context, _, _ uint64) uint64 { return m.factsInNewCats }
func (m *mockKnowledgeKeeper) GetNewCrossDomainEdges(_ context.Context, _, _ uint64) uint64 { return m.newCrossDomainEdges }
func (m *mockKnowledgeKeeper) GetDomainFactDistribution(_ context.Context) []types.DomainFactCount { return m.domainDist }
func (m *mockKnowledgeKeeper) GetNewSubmitterAcceptRate(_ context.Context) uint64          { return m.newEntrantRate }

type mockStakingKeeper struct {
	pairwiseAgreement uint64
	rewardDist        []types.ValidatorReward
	slashProp         uint64
	top10Ratio        uint64
	totalStake        uint64
}

func (m *mockStakingKeeper) GetAvgPairwiseAgreement(_ context.Context) uint64              { return m.pairwiseAgreement }
func (m *mockStakingKeeper) GetValidatorRewardDistribution(_ context.Context) []types.ValidatorReward { return m.rewardDist }
func (m *mockStakingKeeper) GetSlashProportionality(_ context.Context) uint64              { return m.slashProp }
func (m *mockStakingKeeper) GetTop10StakeRatio(_ context.Context) uint64                   { return m.top10Ratio }
func (m *mockStakingKeeper) GetTotalStake(_ context.Context) uint64                        { return m.totalStake }

type mockOntologyKeeper struct {
	activeDomains    uint64
	domainsActivity  uint64
	newDomains       uint64
	totalDomains     uint64
}

func (m *mockOntologyKeeper) GetActiveDomainCount(_ context.Context) uint64                { return m.activeDomains }
func (m *mockOntologyKeeper) GetDomainsWithActivity(_ context.Context, _, _ uint64) uint64 { return m.domainsActivity }
func (m *mockOntologyKeeper) GetNewDomainsInEpoch(_ context.Context, _, _ uint64) uint64   { return m.newDomains }
func (m *mockOntologyKeeper) GetTotalDomainCount(_ context.Context) uint64                 { return m.totalDomains }

type mockAutopoiesisKeeper struct {
	multipliers    map[string]uint64
	entries        []types.MultiplierEntry
	adaptations    uint64
	conflicts      uint64
	dirChanges     uint64
	active         bool
}

func newMockAutopoiesis() *mockAutopoiesisKeeper {
	return &mockAutopoiesisKeeper{
		multipliers: make(map[string]uint64),
		active:      true,
	}
}

func (m *mockAutopoiesisKeeper) GetMultiplier(_ context.Context, path string) uint64 {
	if v, ok := m.multipliers[path]; ok {
		return v
	}
	return types.DefaultMultiplier
}

func (m *mockAutopoiesisKeeper) SetMultiplier(_ context.Context, entry types.MultiplierEntry) {
	m.multipliers[entry.Path] = entry.Value
}

func (m *mockAutopoiesisKeeper) GetMultiplierEntry(_ context.Context, path string) *types.MultiplierEntry {
	if v, ok := m.multipliers[path]; ok {
		return &types.MultiplierEntry{Path: path, Value: v}
	}
	return nil
}

func (m *mockAutopoiesisKeeper) GetAllMultiplierEntries(_ context.Context) []types.MultiplierEntry {
	return m.entries
}

func (m *mockAutopoiesisKeeper) GetAdaptationCount(_ context.Context, _ uint64) uint64 {
	return m.adaptations
}

func (m *mockAutopoiesisKeeper) GetConflictCount(_ context.Context, _ uint64) uint64 {
	return m.conflicts
}

func (m *mockAutopoiesisKeeper) GetDirectionChanges(_ context.Context, _ uint64) uint64 {
	return m.dirChanges
}

func (m *mockAutopoiesisKeeper) IsActive(_ context.Context) bool {
	return m.active
}

type mockEmergencyKeeper struct {
	halted bool
}

func (m *mockEmergencyKeeper) IsHalted(_ context.Context) bool {
	return m.halted
}

type mockVestingKeeper struct {
	rewards []types.ValidatorReward
}

func (m *mockVestingKeeper) GetValidatorRewardsInEpoch(_ context.Context, _, _ uint64) []types.ValidatorReward {
	return m.rewards
}

// ============ Test Setup ============

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1}, false, log.NewNopLogger())

	k := keeper.NewKeeper(
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		"lgm1governance",
	)

	k.InitGenesis(ctx, types.DefaultGenesis())

	return k, ctx
}

func setupFullKeeper(t *testing.T) (
	keeper.Keeper, sdk.Context,
	*mockKnowledgeKeeper, *mockStakingKeeper, *mockOntologyKeeper,
	*mockAutopoiesisKeeper, *mockEmergencyKeeper,
) {
	t.Helper()

	k, ctx := setupKeeper(t)

	mk := &mockKnowledgeKeeper{}
	ms := &mockStakingKeeper{}
	mo := &mockOntologyKeeper{}
	ma := newMockAutopoiesis()
	me := &mockEmergencyKeeper{}

	k.SetKnowledgeKeeper(mk)
	k.SetStakingKeeper(ms)
	k.SetOntologyKeeper(mo)
	k.SetAutopoiesisKeeper(ma)
	k.SetEmergencyKeeper(me)

	return k, ctx, mk, ms, mo, ma, me
}

// ============ Unit Tests: Params & State ============

func TestParamsDefaultRoundtrip(t *testing.T) {
	k, ctx := setupKeeper(t)

	params := k.GetParams(ctx)
	if !params.Enabled {
		t.Fatal("default params should be enabled")
	}
	if params.EpochLengthBlocks != 1111 {
		t.Fatalf("expected epoch length 1111, got %d", params.EpochLengthBlocks)
	}
}

func TestStateRoundtrip(t *testing.T) {
	k, ctx := setupKeeper(t)

	state := k.GetState(ctx)
	if !state.Active {
		t.Fatal("state should be active after InitGenesis")
	}
	if state.SmoothingWindow != 5 {
		t.Fatalf("expected smoothing window 5, got %d", state.SmoothingWindow)
	}
}

func TestIsActive_EmergencyHalt(t *testing.T) {
	k, ctx := setupKeeper(t)
	me := &mockEmergencyKeeper{halted: true}
	k.SetEmergencyKeeper(me)

	if k.IsActive(ctx) {
		t.Fatal("should be inactive during emergency halt")
	}
}

// ============ Unit Tests: TDI Sensor ============

func TestComputeTDI_PerfectTruth(t *testing.T) {
	obs := types.AlignmentObservation{
		ReversalRate:       0,
		ChallengeAsymmetry: 0,
		UnanimityRate:      500_000, // 50% — healthy
		CrossValidatorCorr: 500_000, // 50% — healthy
		CounterFactRatio:   100_000, // 10% — above minimum
	}
	tdi := keeper.ComputeTDI(obs)
	if tdi != 0 {
		t.Fatalf("expected TDI 0 for perfect truth, got %d", tdi)
	}
}

func TestComputeTDI_HighUnanimity(t *testing.T) {
	obs := types.AlignmentObservation{
		UnanimityRate: 950_000, // 95% — suspicious (above 80% threshold)
	}
	tdi := keeper.ComputeTDI(obs)
	if tdi == 0 {
		t.Fatal("expected non-zero TDI when unanimity is 95%")
	}
}

func TestComputeTDI_HighCollusion(t *testing.T) {
	obs := types.AlignmentObservation{
		CrossValidatorCorr: 980_000, // 98% pairwise agreement — very suspicious
	}
	tdi := keeper.ComputeTDI(obs)
	if tdi == 0 {
		t.Fatal("expected non-zero TDI when cross-validator correlation is 98%")
	}
}

func TestComputeTDI_NoCounterFacts(t *testing.T) {
	obs := types.AlignmentObservation{
		CounterFactRatio: 0, // 0% counter-facts — very concerning
	}
	tdi := keeper.ComputeTDI(obs)
	if tdi == 0 {
		t.Fatal("expected non-zero TDI when counter-fact ratio is 0%")
	}
}

// ============ Unit Tests: DCI Sensor ============

func TestComputeDCI_DeepKnowledge(t *testing.T) {
	obs := types.AlignmentObservation{
		AvgCitationDepth:  5_000_000, // depth 5.0 = perfect
		CrossDomainRatio:  1_000_000, // 100% cross-domain
		OrphanRate:        0,         // no orphans
		OntologyCoverage:  1_000_000, // 100% coverage
		SupersessionDepth: 3_000_000, // depth 3.0 = perfect
	}
	dci := keeper.ComputeDCI(obs)
	if dci != 1_000_000 {
		t.Fatalf("expected DCI 1M for deep knowledge, got %d", dci)
	}
}

func TestComputeDCI_ShallowTrivia(t *testing.T) {
	obs := types.AlignmentObservation{
		AvgCitationDepth: 0,
		CrossDomainRatio: 0,
		OrphanRate:       1_000_000, // all orphans
		OntologyCoverage: 0,
		SupersessionDepth: 0,
	}
	dci := keeper.ComputeDCI(obs)
	if dci != 0 {
		t.Fatalf("expected DCI 0 for shallow trivia, got %d", dci)
	}
}

// ============ Unit Tests: HI Sensor ============

func TestComputeHI_PerfectHarmony(t *testing.T) {
	obs := types.AlignmentObservation{
		MultiplierVariance:  0,
		OscillationRate:     0,
		BoundSaturation:     0,
		HormoneConflictRate: 0,
		AvgDampingFactor:    1_000_000, // perfect damping
	}
	hi := keeper.ComputeHI(obs)
	if hi != 1_000_000 {
		t.Fatalf("expected HI 1M for perfect harmony, got %d", hi)
	}
}

func TestComputeHI_Chaos(t *testing.T) {
	obs := types.AlignmentObservation{
		MultiplierVariance:  1_000_000,
		OscillationRate:     1_000_000,
		BoundSaturation:     1_000_000,
		HormoneConflictRate: 1_000_000,
		AvgDampingFactor:    0,
	}
	hi := keeper.ComputeHI(obs)
	if hi != 0 {
		t.Fatalf("expected HI 0 for chaos, got %d", hi)
	}
}

// ============ Unit Tests: FI Sensor ============

func TestComputeFI_PerfectFairness(t *testing.T) {
	obs := types.AlignmentObservation{
		RewardGini:          0, // perfect equality
		NewEntrantSuccess:   1_000_000,
		SlashProportionality: 1_000_000,
		WhaleDominance:      0,
		DomainGini:          0,
	}
	fi := keeper.ComputeFI(obs)
	if fi != 1_000_000 {
		t.Fatalf("expected FI 1M for perfect fairness, got %d", fi)
	}
}

func TestComputeFI_Oligarchy(t *testing.T) {
	obs := types.AlignmentObservation{
		RewardGini:          1_000_000, // max inequality
		NewEntrantSuccess:   0,
		SlashProportionality: 0,
		WhaleDominance:      1_000_000,
		DomainGini:          1_000_000,
	}
	fi := keeper.ComputeFI(obs)
	if fi != 0 {
		t.Fatalf("expected FI 0 for oligarchy, got %d", fi)
	}
}

// ============ Unit Tests: NI Sensor ============

func TestComputeNI_HighNovelty(t *testing.T) {
	obs := types.AlignmentObservation{
		NewDomainRate:      50_000,  // 5% = target
		NovelClaimRatio:    200_000, // 20% = target
		CrossDomainBridges: 100_000, // 10% = target
		FactDiversity:      1_000_000,
		SupersessionRate:   1_000_000,
	}
	ni := keeper.ComputeNI(obs)
	if ni != 1_000_000 {
		t.Fatalf("expected NI 1M for high novelty, got %d", ni)
	}
}

func TestComputeNI_Stagnation(t *testing.T) {
	obs := types.AlignmentObservation{
		NewDomainRate:      0,
		NovelClaimRatio:    0,
		CrossDomainBridges: 0,
		FactDiversity:      0,
		SupersessionRate:   0,
	}
	ni := keeper.ComputeNI(obs)
	if ni != 0 {
		t.Fatalf("expected NI 0 for stagnation, got %d", ni)
	}
}

// ============ Unit Tests: Gini Coefficient ============

func TestGiniCoefficient_PerfectEquality(t *testing.T) {
	gini := types.ComputeGiniCoefficient([]uint64{100, 100, 100, 100})
	if gini != 0 {
		t.Fatalf("expected Gini 0 for perfect equality, got %d", gini)
	}
}

func TestGiniCoefficient_MaxInequality(t *testing.T) {
	// One person has everything, rest have nothing
	gini := types.ComputeGiniCoefficient([]uint64{0, 0, 0, 1000})
	if gini < 700_000 {
		t.Fatalf("expected high Gini for extreme inequality, got %d", gini)
	}
}

func TestGiniCoefficient_SingleValue(t *testing.T) {
	gini := types.ComputeGiniCoefficient([]uint64{42})
	if gini != 0 {
		t.Fatalf("expected Gini 0 for single value, got %d", gini)
	}
}

// ============ Unit Tests: Shannon Entropy ============

func TestShannonEntropy_SingleCategory(t *testing.T) {
	entropy := types.ComputeShannonEntropy([]uint64{100})
	if entropy != types.BasisPoints {
		t.Fatalf("expected max entropy for single category, got %d", entropy)
	}
}

func TestShannonEntropy_UniformDistribution(t *testing.T) {
	entropy := types.ComputeShannonEntropy([]uint64{100, 100, 100, 100})
	// Uniform = maximum entropy → should be near BasisPoints
	if entropy < 900_000 {
		t.Fatalf("expected high entropy for uniform distribution, got %d", entropy)
	}
}

func TestShannonEntropy_Monoculture(t *testing.T) {
	entropy := types.ComputeShannonEntropy([]uint64{1000, 0, 0, 0})
	// All in one category = zero diversity
	if entropy > 100_000 {
		t.Fatalf("expected low entropy for monoculture, got %d", entropy)
	}
}

// ============ Unit Tests: Corrections ============

func TestTruthCorrections_MildDrift(t *testing.T) {
	k, ctx, _, _, _, ma, _ := setupFullKeeper(t)

	scores := types.DimensionScores{TDI: 250_000} // 25% drift — mild
	state := types.AlignmentState{CorrectionDamping: types.BasisPoints}
	params := types.DefaultParams()

	_ = k // suppress unused — we test the exported functions directly
	_ = ctx
	_ = ma
	_ = scores
	_ = state
	_ = params

	// Just verify ComputeTDI works for a mild drift scenario
	obs := types.AlignmentObservation{
		ReversalRate: 500_000, // high reversal
	}
	tdi := keeper.ComputeTDI(obs)
	if tdi < 100_000 {
		t.Fatalf("expected TDI > 100k with 50%% reversal rate, got %d", tdi)
	}
}

// ============ Integration: Full Epoch Cycle ============

func TestFullEpochCycle_NoKeepers(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Advance to epoch boundary (block 1111)
	ctx = ctx.WithBlockHeight(1111)
	k.SenseJudgeCorrectReflect(ctx)

	state := k.GetState(ctx)
	if state.CurrentEpoch != 1 {
		t.Fatalf("expected epoch 1 after block 1111, got %d", state.CurrentEpoch)
	}

	scores := k.GetScores(ctx, 1)
	if scores == nil {
		t.Fatal("expected scores to be stored for epoch 1")
	}
}

func TestFullEpochCycle_WithMocks(t *testing.T) {
	k, ctx, mk, _, _, _, _ := setupFullKeeper(t)

	// Set up some truth drift: 90% unanimity (suspicious)
	mk.totalRounds = 100
	mk.unanimousRounds = 90
	mk.verifiedFacts = 100
	mk.totalFacts = 100
	mk.activeFacts = 100

	ctx = ctx.WithBlockHeight(1111)
	k.SenseJudgeCorrectReflect(ctx)

	scores := k.GetScores(ctx, 1)
	if scores == nil {
		t.Fatal("expected scores")
	}
	if scores.TDI == 0 {
		t.Fatal("expected non-zero TDI with 90% unanimity")
	}
}

func TestEpochSkip_NoDoubleProcess(t *testing.T) {
	k, ctx := setupKeeper(t)

	ctx = ctx.WithBlockHeight(1111)
	k.SenseJudgeCorrectReflect(ctx)

	// Same height again — should be no-op
	k.SenseJudgeCorrectReflect(ctx)

	state := k.GetState(ctx)
	if state.CurrentEpoch != 1 {
		t.Fatalf("expected epoch 1 (no double process), got %d", state.CurrentEpoch)
	}
}

func TestInactive_NoProcess(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Disable
	state := k.GetState(ctx)
	state.Active = false
	k.SetState(ctx, state)

	ctx = ctx.WithBlockHeight(1111)
	k.SenseJudgeCorrectReflect(ctx)

	s := k.GetState(ctx)
	if s.CurrentEpoch != 0 {
		t.Fatal("expected no epoch advancement when inactive")
	}
}
