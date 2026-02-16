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

	"github.com/legible-money/legible/x/autopoiesis/keeper"
	"github.com/legible-money/legible/x/autopoiesis/types"
)

// ---------- Mock Keepers ----------

type mockStakingKeeper struct {
	activeValidators uint64
	totalValidators  uint64
	avgAccuracy      uint64
	slashEvents      uint64
}

func (m *mockStakingKeeper) GetActiveValidatorCount(_ context.Context) uint64 {
	return m.activeValidators
}
func (m *mockStakingKeeper) GetTotalValidatorCount(_ context.Context) uint64 {
	return m.totalValidators
}
func (m *mockStakingKeeper) GetAverageValidatorAccuracy(_ context.Context) uint64 {
	return m.avgAccuracy
}
func (m *mockStakingKeeper) GetSlashEventsInRange(_ context.Context, _, _ uint64) uint64 {
	return m.slashEvents
}

type mockKnowledgeKeeper struct {
	activeFacts        uint64
	factsCreated       uint64
	factsExpired       uint64
	challengeSuccess   uint64
	domains            []string
	domainObservations map[string]types.DomainObservation
	retentions         []types.CategoryRetention
}

func (m *mockKnowledgeKeeper) GetActiveFactCount(_ context.Context) uint64 {
	return m.activeFacts
}
func (m *mockKnowledgeKeeper) GetFactsCreatedInRange(_ context.Context, _, _ uint64) uint64 {
	return m.factsCreated
}
func (m *mockKnowledgeKeeper) GetFactsExpiredInRange(_ context.Context, _, _ uint64) uint64 {
	return m.factsExpired
}
func (m *mockKnowledgeKeeper) GetChallengeSuccessRate(_ context.Context) uint64 {
	return m.challengeSuccess
}
func (m *mockKnowledgeKeeper) GetActiveDomains(_ context.Context) []string {
	return m.domains
}
func (m *mockKnowledgeKeeper) GetDomainObservation(_ context.Context, domain string, _, _ uint64) types.DomainObservation {
	if m.domainObservations != nil {
		if obs, ok := m.domainObservations[domain]; ok {
			return obs
		}
	}
	return types.DomainObservation{Domain: domain}
}
func (m *mockKnowledgeKeeper) GetCategoryRetentions(_ context.Context, _, _ uint64) []types.CategoryRetention {
	return m.retentions
}

type mockEmergencyKeeper struct {
	halted bool
}

func (m *mockEmergencyKeeper) IsHalted(_ context.Context) bool {
	return m.halted
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockStakingKeeper, *mockKnowledgeKeeper, *mockEmergencyKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	mockSK := &mockStakingKeeper{
		activeValidators: 20,
		totalValidators:  22,
		avgAccuracy:      900_000, // 90%
		slashEvents:      0,
	}
	mockKK := &mockKnowledgeKeeper{
		activeFacts:  100,
		factsCreated: 10,
		factsExpired: 2,
	}
	mockEK := &mockEmergencyKeeper{halted: false}

	storeService := runtime.NewKVStoreService(storeKey)

	k := keeper.NewKeeper(
		storeService,
		log.NewNopLogger(),
		mockSK,
		mockKK,
		mockEK,
		"authority",
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
	k.InitGenesis(ctx, types.DefaultGenesis())

	return k, ctx, mockSK, mockKK, mockEK
}

// ---------- SSI Computation Tests ----------

func TestComputeSSI_Healthy(t *testing.T) {
	obs := types.EpochObservation{
		ActiveValidators:     20,
		TotalValidators:      22,
		ValidatorAccuracyBps: 950_000, // 95% accuracy
		FactsCreated:         10,
		FactsExpired:         1,
		TotalActiveFacts:     100,
	}
	ssi := types.ComputeSSI(obs)
	if ssi > 200_000 {
		t.Errorf("expected healthy SSI (<0.2), got %d", ssi)
	}
}

func TestComputeSSI_Moderate(t *testing.T) {
	obs := types.EpochObservation{
		ActiveValidators:        10,
		TotalValidators:         22,
		ValidatorAccuracyBps:    700_000, // 70% accuracy
		ChallengeSuccessRateBps: 500_000, // 50% challenges succeed
		FactsCreated:            5,
		FactsExpired:            4,
		TotalActiveFacts:        100,
		AvgConfidenceDeclineBps: 300_000,
	}
	ssi := types.ComputeSSI(obs)
	if ssi < 300_000 || ssi > 700_000 {
		t.Errorf("expected moderate SSI (0.3-0.7), got %d", ssi)
	}
}

func TestComputeSSI_Critical(t *testing.T) {
	obs := types.EpochObservation{
		ActiveValidators:        3,
		TotalValidators:         22,
		ValidatorAccuracyBps:    300_000, // 30% accuracy
		ChallengeSuccessRateBps: 800_000, // 80% challenges succeed
		FactsCreated:            1,
		FactsExpired:            10,
		TotalActiveFacts:        100,
		AvgConfidenceDeclineBps: 700_000,
	}
	ssi := types.ComputeSSI(obs)
	if ssi < 700_000 {
		t.Errorf("expected critical SSI (>0.7), got %d", ssi)
	}
}

func TestComputeSSI_NoValidators(t *testing.T) {
	obs := types.EpochObservation{
		ActiveValidators: 0,
		TotalValidators:  0,
	}
	ssi := types.ComputeSSI(obs)
	// No validators = no participation signal = some stress
	if ssi > 500_000 {
		t.Errorf("expected moderate SSI with no validators, got %d", ssi)
	}
}

func TestComputeSSI_NoFactsCreated(t *testing.T) {
	obs := types.EpochObservation{
		ActiveValidators:     10,
		TotalValidators:      10,
		ValidatorAccuracyBps: 1_000_000,
		FactsCreated:         0,
		FactsExpired:         5, // expiring with no creation
	}
	ssi := types.ComputeSSI(obs)
	if ssi < 200_000 {
		t.Errorf("expected elevated SSI when facts expire without creation, got %d", ssi)
	}
}

func TestComputeSSI_ClampsToMax(t *testing.T) {
	obs := types.EpochObservation{
		ActiveValidators:        0,
		TotalValidators:         100,
		ValidatorAccuracyBps:    0,
		ChallengeSuccessRateBps: 2_000_000, // over max — should clamp
		FactsCreated:            0,
		FactsExpired:            100,
		AvgConfidenceDeclineBps: 2_000_000, // over max — should clamp
	}
	ssi := types.ComputeSSI(obs)
	if ssi > 1_000_000 {
		t.Errorf("SSI should be clamped to 1.0 (1M bps), got %d", ssi)
	}
}

// ---------- SSI Adaptation Tests ----------

func TestAdaptation_HealthyRelaxes(t *testing.T) {
	params := types.DefaultParams()
	// Start with elevated multiplier
	current := uint64(1_200_000) // 1.2x
	ssi := uint64(100_000)      // 0.1 — healthy
	newMult, delta := types.ComputeSSIAdaptation(ssi, current, params)
	if delta >= 0 {
		t.Errorf("expected negative delta for healthy SSI, got %d", delta)
	}
	if newMult >= current {
		t.Errorf("expected multiplier to decrease, old=%d new=%d", current, newMult)
	}
}

func TestAdaptation_HealthyDoesNotGoBelow1x(t *testing.T) {
	params := types.DefaultParams()
	current := uint64(1_000_000) // 1.0x — already at default
	ssi := uint64(50_000)        // very healthy
	newMult, delta := types.ComputeSSIAdaptation(ssi, current, params)
	if delta != 0 {
		t.Errorf("expected no change when already at 1.0x, got delta=%d", delta)
	}
	if newMult != types.DefaultMultiplier {
		t.Errorf("expected default multiplier, got %d", newMult)
	}
}

func TestAdaptation_ModerateIncreasesSlash(t *testing.T) {
	params := types.DefaultParams()
	current := uint64(1_000_000) // 1.0x
	ssi := uint64(600_000)       // 0.6 — moderate
	newMult, delta := types.ComputeSSIAdaptation(ssi, current, params)
	if delta <= 0 {
		t.Errorf("expected positive delta for moderate SSI, got %d", delta)
	}
	if newMult <= current {
		t.Errorf("expected multiplier to increase, old=%d new=%d", current, newMult)
	}
}

func TestAdaptation_CriticalIncreasesMore(t *testing.T) {
	// Single epoch both hit 5% rate limit — accumulate over 4 epochs to see divergence.
	// Critical target is +15% per epoch (clamped to +5%), moderate is +5% (at cap already).
	// After 4 epochs critical catches up because target keeps growing,
	// while moderate's smaller delta means the absolute cap also grows proportionally.
	// Since both hit the per-epoch cap each time the outputs are identical.
	// Instead, test by raising the rate limit so the raw deltas are visible.
	params := types.DefaultParams()
	params.MaxChangePerEpochBps = 200_000 // raise to 20% so both deltas pass unclamped
	current := uint64(1_000_000)          // 1.0x
	ssiMod := uint64(600_000)             // moderate
	ssiCrit := uint64(900_000)            // critical
	newMod, _ := types.ComputeSSIAdaptation(ssiMod, current, params)
	newCrit, _ := types.ComputeSSIAdaptation(ssiCrit, current, params)
	if newCrit <= newMod {
		t.Errorf("critical SSI should produce larger increase than moderate: critical=%d moderate=%d", newCrit, newMod)
	}
}

func TestAdaptation_RateLimiting(t *testing.T) {
	params := types.DefaultParams()
	current := uint64(1_000_000) // 1.0x
	ssi := uint64(900_000)       // critical — wants +15%
	newMult, _ := types.ComputeSSIAdaptation(ssi, current, params)
	maxAllowed := current + (current * params.MaxChangePerEpochBps / types.BasisPoints)
	if newMult > maxAllowed {
		t.Errorf("rate limit violated: new=%d max_allowed=%d", newMult, maxAllowed)
	}
}

func TestAdaptation_BoundsEnforced(t *testing.T) {
	params := types.DefaultParams()

	// Try to go above max
	current := uint64(2_900_000) // 2.9x — near max of 3.0x
	ssi := uint64(900_000)       // critical
	newMult, _ := types.ComputeSSIAdaptation(ssi, current, params)
	if newMult > params.SlashMultiplierMax {
		t.Errorf("upper bound violated: new=%d max=%d", newMult, params.SlashMultiplierMax)
	}

	// Try to go below min
	current2 := uint64(510_000) // 0.51x — near min of 0.5x
	ssi2 := uint64(50_000)      // healthy — wants to relax
	newMult2, _ := types.ComputeSSIAdaptation(ssi2, current2, params)
	if newMult2 < params.SlashMultiplierMin {
		t.Errorf("lower bound violated: new=%d min=%d", newMult2, params.SlashMultiplierMin)
	}
}

func TestAdaptation_MultiEpochAccumulation(t *testing.T) {
	params := types.DefaultParams()
	var mult uint64 = types.DefaultMultiplier
	ssi := uint64(700_000) // moderate-high

	// Run 10 epochs of moderate stress
	for i := 0; i < 10; i++ {
		mult, _ = types.ComputeSSIAdaptation(ssi, mult, params)
	}

	if mult <= types.DefaultMultiplier {
		t.Errorf("10 epochs of stress should accumulate: got %d", mult)
	}
	if mult > params.SlashMultiplierMax {
		t.Errorf("accumulated multiplier exceeds max: %d > %d", mult, params.SlashMultiplierMax)
	}
}

// ---------- Keeper State Tests ----------

func TestGetSetParams(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	params := k.GetParams(ctx)
	if params.EpochLengthBlocks != types.EpochLengthBlocks {
		t.Errorf("expected default epoch length %d, got %d", types.EpochLengthBlocks, params.EpochLengthBlocks)
	}

	params.EpochLengthBlocks = 2222
	k.SetParams(ctx, params)
	got := k.GetParams(ctx)
	if got.EpochLengthBlocks != 2222 {
		t.Errorf("expected 2222, got %d", got.EpochLengthBlocks)
	}
}

func TestGetMultiplier_DefaultIs1x(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	mult := k.GetMultiplier(ctx, types.PathSlashSystem)
	if mult != types.DefaultMultiplier {
		t.Errorf("expected default multiplier %d, got %d", types.DefaultMultiplier, mult)
	}
}

func TestSetGetMultiplier(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	entry := types.MultiplierEntry{
		Path:    types.PathSlashSystem,
		Value:   1_500_000,
		Hormone: "SSI",
		Epoch:   5,
	}
	k.SetMultiplier(ctx, entry)
	got := k.GetMultiplier(ctx, types.PathSlashSystem)
	if got != 1_500_000 {
		t.Errorf("expected 1500000, got %d", got)
	}
}

func TestSetGetObservation(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	obs := types.EpochObservation{
		Epoch:            3,
		BlockHeight:      3333,
		ActiveValidators: 20,
		TotalValidators:  22,
	}
	k.SetObservation(ctx, obs)
	got := k.GetObservation(ctx, 3)
	if got == nil {
		t.Fatal("expected observation, got nil")
	}
	if got.ActiveValidators != 20 {
		t.Errorf("expected 20, got %d", got.ActiveValidators)
	}
}

func TestGetObservation_NonExistent(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	got := k.GetObservation(ctx, 999)
	if got != nil {
		t.Errorf("expected nil for non-existent epoch, got %+v", got)
	}
}

func TestIsActive_DefaultEnabled(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	if !k.IsActive(ctx) {
		t.Error("expected active after init with enabled=true")
	}
}

func TestIsActive_EmergencyFreeze(t *testing.T) {
	k, ctx, _, _, ek := setupKeeper(t)
	if !k.IsActive(ctx) {
		t.Error("expected active before emergency")
	}
	ek.halted = true
	if k.IsActive(ctx) {
		t.Error("expected inactive during emergency halt")
	}
}

func TestIsActive_DisabledByParams(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	params := k.GetParams(ctx)
	params.Enabled = false
	k.SetParams(ctx, params)
	if k.IsActive(ctx) {
		t.Error("expected inactive when params.Enabled=false")
	}
}

// ---------- CollectAndAdapt Integration Tests ----------

func TestCollectAndAdapt_HealthyChain(t *testing.T) {
	k, ctx, sk, kk, _ := setupKeeper(t)

	// Healthy chain metrics
	sk.activeValidators = 20
	sk.totalValidators = 22
	sk.avgAccuracy = 950_000
	kk.activeFacts = 100
	kk.factsCreated = 10
	kk.factsExpired = 1

	// Advance past epoch boundary
	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)

	// Should stay at default (healthy = no increase)
	mult := k.GetMultiplier(ctx, types.PathSlashSystem)
	if mult != types.DefaultMultiplier {
		t.Errorf("expected default multiplier for healthy chain, got %d", mult)
	}
}

func TestCollectAndAdapt_StressedChain(t *testing.T) {
	k, ctx, sk, kk, _ := setupKeeper(t)

	// Stressed chain metrics
	sk.activeValidators = 5
	sk.totalValidators = 22
	sk.avgAccuracy = 500_000 // 50%
	kk.activeFacts = 100
	kk.factsCreated = 2
	kk.factsExpired = 10
	kk.challengeSuccess = 600_000

	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)

	mult := k.GetMultiplier(ctx, types.PathSlashSystem)
	if mult <= types.DefaultMultiplier {
		t.Errorf("expected increased multiplier for stressed chain, got %d", mult)
	}
}

func TestCollectAndAdapt_EmergencyFreezeSkips(t *testing.T) {
	k, ctx, _, _, ek := setupKeeper(t)

	ek.halted = true
	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)

	mult := k.GetMultiplier(ctx, types.PathSlashSystem)
	if mult != types.DefaultMultiplier {
		t.Errorf("expected no change during emergency halt, got %d", mult)
	}
}

func TestCollectAndAdapt_NoChangeBeforeFirstEpoch(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Block 500 — within first epoch (epoch 0)
	ctx = ctx.WithBlockHeight(500)
	k.CollectAndAdapt(ctx)

	// State should not advance — epoch 0 stays
	state := k.GetState(ctx)
	if state.CurrentEpoch != 0 {
		t.Errorf("expected epoch 0, got %d", state.CurrentEpoch)
	}
}

func TestCollectAndAdapt_EpochProgression(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// First epoch boundary
	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)
	state := k.GetState(ctx)
	if state.CurrentEpoch != 1 {
		t.Errorf("expected epoch 1, got %d", state.CurrentEpoch)
	}

	// Second epoch boundary
	ctx = ctx.WithBlockHeight(int64(2*types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)
	state = k.GetState(ctx)
	if state.CurrentEpoch != 2 {
		t.Errorf("expected epoch 2, got %d", state.CurrentEpoch)
	}
}

func TestCollectAndAdapt_SameEpochNoRepeat(t *testing.T) {
	k, ctx, sk, _, _ := setupKeeper(t)

	sk.avgAccuracy = 500_000 // stressed
	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)
	mult1 := k.GetMultiplier(ctx, types.PathSlashSystem)

	// Same epoch, different block
	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 10))
	k.CollectAndAdapt(ctx)
	mult2 := k.GetMultiplier(ctx, types.PathSlashSystem)

	if mult1 != mult2 {
		t.Errorf("expected no change within same epoch: first=%d second=%d", mult1, mult2)
	}
}

func TestCollectAndAdapt_ObservationStored(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)

	obs := k.GetObservation(ctx, 1)
	if obs == nil {
		t.Fatal("expected observation stored for epoch 1")
	}
	if obs.ActiveValidators != 20 {
		t.Errorf("expected 20 active validators in observation, got %d", obs.ActiveValidators)
	}
}

func TestCollectAndAdapt_SSIStored(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)

	ssi := k.GetSSI(ctx, 1)
	if ssi == nil {
		t.Fatal("expected SSI stored for epoch 1")
	}
}

// ---------- Params Validation Tests ----------

func TestParams_Validate(t *testing.T) {
	p := types.DefaultParams()
	if err := p.Validate(); err != nil {
		t.Errorf("default params should be valid: %v", err)
	}

	p.EpochLengthBlocks = 0
	if err := p.Validate(); err == nil {
		t.Error("expected error for zero epoch length")
	}

	p = types.DefaultParams()
	p.SlashMultiplierMin = p.SlashMultiplierMax + 1
	if err := p.Validate(); err == nil {
		t.Error("expected error for min > max")
	}
}

func TestGetMultiplierEntry(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Not set yet
	entry := k.GetMultiplierEntry(ctx, types.PathSlashSystem)
	if entry != nil {
		t.Errorf("expected nil for unset multiplier, got %+v", entry)
	}

	// Set and retrieve
	k.SetMultiplier(ctx, types.MultiplierEntry{
		Path:    types.PathSlashSystem,
		Value:   1_100_000,
		Hormone: "SSI",
		Epoch:   2,
	})
	entry = k.GetMultiplierEntry(ctx, types.PathSlashSystem)
	if entry == nil {
		t.Fatal("expected entry after set")
	}
	if entry.Hormone != "SSI" {
		t.Errorf("expected SSI hormone, got %s", entry.Hormone)
	}
}

// ---------- Hormone Values Storage Tests ----------

func TestSetGetHormoneValues(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	hv := types.HormoneValues{
		Epoch: 5,
		SSI:   300_000,
		GRM:   600_000,
		CRI:   []types.CRIValue{{Domain: "math", Category: "formal", Value: 800_000}},
		DVI:   []types.DVIValue{{Domain: "math", Value: 100_000}},
		VCF:   []types.VCFValue{{Domain: "math", Value: 500_000}},
		LifeForce: types.LifeForce{Value: 400_000, Status: "healthy"},
	}
	k.SetHormoneValues(ctx, hv)

	got := k.GetHormoneValues(ctx, 5)
	if got == nil {
		t.Fatal("expected hormone values, got nil")
	}
	if got.SSI != 300_000 {
		t.Errorf("expected SSI 300000, got %d", got.SSI)
	}
	if got.GRM != 600_000 {
		t.Errorf("expected GRM 600000, got %d", got.GRM)
	}
	if len(got.CRI) != 1 || got.CRI[0].Value != 800_000 {
		t.Errorf("unexpected CRI: %+v", got.CRI)
	}
	if got.LifeForce.Status != "healthy" {
		t.Errorf("expected healthy life force, got %s", got.LifeForce.Status)
	}
}

func TestSSIBackwardCompatibility(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// SetSSI should create HormoneValues with SSI field
	k.SetSSI(ctx, types.SSIValue{Epoch: 3, Value: 500_000})
	ssi := k.GetSSI(ctx, 3)
	if ssi == nil || ssi.Value != 500_000 {
		t.Fatalf("expected SSI 500000, got %+v", ssi)
	}

	// Should also be accessible via GetHormoneValues
	hv := k.GetHormoneValues(ctx, 3)
	if hv == nil || hv.SSI != 500_000 {
		t.Fatalf("expected HormoneValues.SSI 500000, got %+v", hv)
	}
}

// ---------- DVI Tracking Tests ----------

func TestSetGetDVITracking(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dvi := types.DVIValue{Domain: "physics", Value: -50_000, ConsecutiveNegativeEpochs: 3}
	k.SetDVITracking(ctx, dvi)

	got := k.GetDVITracking(ctx, "physics")
	if got == nil {
		t.Fatal("expected DVI tracking, got nil")
	}
	if got.ConsecutiveNegativeEpochs != 3 {
		t.Errorf("expected 3 consecutive negative epochs, got %d", got.ConsecutiveNegativeEpochs)
	}
	if got.Value != -50_000 {
		t.Errorf("expected value -50000, got %d", got.Value)
	}
}

func TestGetDVITracking_NonExistent(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)
	got := k.GetDVITracking(ctx, "nonexistent")
	if got != nil {
		t.Errorf("expected nil for untracked domain, got %+v", got)
	}
}

// ---------- Multi-Hormone Integration Tests ----------

func TestCollectAndAdapt_WithDomains(t *testing.T) {
	k, ctx, sk, kk, _ := setupKeeper(t)

	// Setup domain-level data
	sk.activeValidators = 20
	sk.totalValidators = 22
	sk.avgAccuracy = 950_000
	kk.activeFacts = 200
	kk.factsCreated = 20
	kk.factsExpired = 2
	kk.domains = []string{"mathematics", "physics"}
	kk.domainObservations = map[string]types.DomainObservation{
		"mathematics": {
			Domain:           "mathematics",
			FactsCreated:     15,
			FactsExpired:     1,
			TotalActiveFacts: 120,
			NewCitations:     30,
		},
		"physics": {
			Domain:           "physics",
			FactsCreated:     5,
			FactsExpired:     1,
			TotalActiveFacts: 80,
			NewCitations:     10,
		},
	}
	kk.retentions = []types.CategoryRetention{
		{Domain: "mathematics", Category: "empirical", RetentionRatioBps: 900_000, FactCount: 50},
		{Domain: "physics", Category: "empirical", RetentionRatioBps: 700_000, FactCount: 30},
	}

	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)

	// Should store hormone values with CRI, DVI, VCF
	hv := k.GetHormoneValues(ctx, 1)
	if hv == nil {
		t.Fatal("expected hormone values stored")
	}
	if len(hv.CRI) == 0 {
		t.Error("expected CRI values for domain retentions")
	}
	if len(hv.DVI) == 0 {
		t.Error("expected DVI values for domains")
	}
	if len(hv.VCF) == 0 {
		t.Error("expected VCF values for domains")
	}
	if hv.LifeForce.Status == "" {
		t.Error("expected life force status")
	}
}

func TestCollectAndAdapt_CRIAdaptation(t *testing.T) {
	k, ctx, sk, kk, _ := setupKeeper(t)

	sk.activeValidators = 20
	sk.totalValidators = 22
	sk.avgAccuracy = 950_000
	kk.activeFacts = 100
	kk.factsCreated = 10
	kk.factsExpired = 2
	// Very low retention → CRI should trigger decay acceleration
	kk.retentions = []types.CategoryRetention{
		{Domain: "physics", Category: "empirical", RetentionRatioBps: 100_000, FactCount: 50},
	}

	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)

	// decay.empirical.physics should be adapted (decreased = faster decay)
	decayMult := k.GetMultiplier(ctx, "decay.empirical.physics")
	if decayMult >= types.DefaultMultiplier {
		t.Errorf("expected decreased decay multiplier for low retention, got %d", decayMult)
	}
}

func TestCollectAndAdapt_DVITracking(t *testing.T) {
	k, ctx, sk, kk, _ := setupKeeper(t)

	sk.activeValidators = 20
	sk.totalValidators = 22
	sk.avgAccuracy = 950_000
	kk.activeFacts = 100
	kk.factsCreated = 10
	kk.factsExpired = 2

	// Shrinking domain (more expired than created)
	kk.domains = []string{"biology"}
	kk.domainObservations = map[string]types.DomainObservation{
		"biology": {
			Domain:           "biology",
			FactsCreated:     1,
			FactsExpired:     10,
			TotalActiveFacts: 50,
		},
	}

	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)

	// DVI tracking should be stored
	dvi := k.GetDVITracking(ctx, "biology")
	if dvi == nil {
		t.Fatal("expected DVI tracking for biology")
	}
	if dvi.Value >= 0 {
		t.Errorf("expected negative DVI for shrinking domain, got %d", dvi.Value)
	}
	if dvi.ConsecutiveNegativeEpochs != 1 {
		t.Errorf("expected 1 consecutive negative epoch, got %d", dvi.ConsecutiveNegativeEpochs)
	}
}

func TestCollectAndAdapt_VCFHighCascade(t *testing.T) {
	k, ctx, sk, kk, _ := setupKeeper(t)

	sk.activeValidators = 20
	sk.totalValidators = 22
	sk.avgAccuracy = 950_000
	kk.activeFacts = 100
	kk.factsCreated = 10
	kk.factsExpired = 2

	// High cascade domain
	kk.domains = []string{"mathematics"}
	kk.domainObservations = map[string]types.DomainObservation{
		"mathematics": {
			Domain:           "mathematics",
			FactsCreated:     10,
			TotalActiveFacts: 50,
			NewCitations:     100, // lots of citations
		},
	}

	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)

	hv := k.GetHormoneValues(ctx, 1)
	if hv == nil {
		t.Fatal("expected hormone values")
	}
	// VCF should be high for mathematics
	foundHigh := false
	for _, v := range hv.VCF {
		if v.Domain == "mathematics" && v.Value > types.VCFHighCascade {
			foundHigh = true
		}
	}
	if !foundHigh {
		t.Errorf("expected high VCF for mathematics with many citations, got %+v", hv.VCF)
	}
}

func TestCollectAndAdapt_PriorityConflictResolution(t *testing.T) {
	k, ctx, sk, kk, _ := setupKeeper(t)

	// Stressed chain (SSI will claim slash.system)
	sk.activeValidators = 5
	sk.totalValidators = 22
	sk.avgAccuracy = 500_000
	sk.slashEvents = 5
	kk.activeFacts = 100
	kk.factsCreated = 2
	kk.factsExpired = 10
	kk.challengeSuccess = 600_000

	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)

	// SSI should have claimed slash.system
	entry := k.GetMultiplierEntry(ctx, types.PathSlashSystem)
	if entry == nil {
		t.Fatal("expected slash.system entry")
	}
	if entry.Hormone != "SSI" {
		t.Errorf("expected SSI to own slash.system, got %s", entry.Hormone)
	}
}

func TestCollectAndAdapt_GRMAdaptation(t *testing.T) {
	k, ctx, sk, kk, _ := setupKeeper(t)

	sk.activeValidators = 20
	sk.totalValidators = 22
	sk.avgAccuracy = 950_000
	kk.activeFacts = 100
	kk.factsCreated = 10
	kk.factsExpired = 2
	// High GRM: lots of verified facts relative to max possible
	kk.activeFacts = 100

	ctx = ctx.WithBlockHeight(int64(types.EpochLengthBlocks + 1))
	k.CollectAndAdapt(ctx)

	// GRM computation depends on VerifiedFactsCreated / MaxPossibleVerifications
	// With default metrics these are 0, so GRM will be 0 (low utilization)
	// GRM < GRMLow → should decrease maxtx, increase gas
	hv := k.GetHormoneValues(ctx, 1)
	if hv == nil {
		t.Fatal("expected hormone values")
	}
	// GRM = 0 which is below GRMLow, so maxtx.system should decrease
	maxtx := k.GetMultiplier(ctx, types.PathMaxTxSystem)
	if maxtx > types.DefaultMultiplier {
		t.Errorf("expected decreased maxtx for low GRM, got %d", maxtx)
	}
}
