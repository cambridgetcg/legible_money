package keeper_test

import (
	"context"
	"math"
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

	"github.com/legible-money/legible/x/capture_defense/keeper"
	"github.com/legible-money/legible/x/capture_defense/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

// ---------- Mock StakingKeeper ----------

type mockStakingKeeper struct {
	effectiveStakes map[string]uint64
	totalStake      uint64
	slashCalls      []slashCall
}

type slashCall struct {
	operatorAddr string
	amount       string
	reason       string
}

func newMockStakingKeeper() *mockStakingKeeper {
	return &mockStakingKeeper{
		effectiveStakes: make(map[string]uint64),
	}
}

func (m *mockStakingKeeper) GetEffectiveStake(_ context.Context, operatorAddr string) uint64 {
	return m.effectiveStakes[operatorAddr]
}

func (m *mockStakingKeeper) GetTotalStake(_ context.Context) uint64 {
	return m.totalStake
}

func (m *mockStakingKeeper) SlashValidator(_ context.Context, operatorAddr string, amount string, reason string) error {
	m.slashCalls = append(m.slashCalls, slashCall{operatorAddr, amount, reason})
	return nil
}

// ---------- Mock OntologyKeeper ----------

type mockOntologyKeeper struct {
	domainDepths  map[string]uint32
	domainStrata  map[string]uint32
	domainExists  map[string]bool
	parentDomains map[string]string
	stratumNames  map[uint32]string
}

func newMockOntologyKeeper() *mockOntologyKeeper {
	return &mockOntologyKeeper{
		domainDepths:  make(map[string]uint32),
		domainStrata:  make(map[string]uint32),
		domainExists:  make(map[string]bool),
		parentDomains: make(map[string]string),
		stratumNames:  make(map[uint32]string),
	}
}

func (m *mockOntologyKeeper) GetDomainDepth(_ sdk.Context, domain string) (uint32, bool) {
	d, ok := m.domainDepths[domain]
	return d, ok
}

func (m *mockOntologyKeeper) GetDomainStratum(_ sdk.Context, domain string) (uint32, bool) {
	s, ok := m.domainStrata[domain]
	return s, ok
}

func (m *mockOntologyKeeper) DomainExists(_ sdk.Context, domain string) bool {
	return m.domainExists[domain]
}

func (m *mockOntologyKeeper) GetParentDomain(_ sdk.Context, domain string) (string, bool) {
	p, ok := m.parentDomains[domain]
	return p, ok
}

func (m *mockOntologyKeeper) GetStratumName(_ sdk.Context, stratum uint32) string {
	return m.stratumNames[stratum]
}

// ---------- Mock BankKeeper ----------

type mockBankKeeper struct{}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, _ sdk.AccAddress, _ string, _ sdk.Coins) error {
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
	return nil
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockStakingKeeper, *mockOntologyKeeper) {
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
	mockOK := newMockOntologyKeeper()
	mockBK := &mockBankKeeper{}

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		mockSK,
		mockOK,
		mockBK,
		"authority",
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	k.SetParams(ctx, *types.DefaultParams())

	return k, ctx, mockSK, mockOK
}

func testAddr(name string) string {
	addr := sdk.AccAddress([]byte("addr_" + name + "_______________")[:20])
	return addr.String()
}

// =============================================================================
// Serialization Tests (3)
// =============================================================================

func TestDomainQualificationRoundtrip(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	qual := types.DomainQualification{
		ValidatorAddr:       testAddr("val1"),
		Domain:              "mathematics",
		Stratum:             3,
		Pathway:             string(types.PathwayStakeCommitment),
		Status:              string(types.StatusActive),
		EarnedAtBlock:       50,
		ExpiresAtBlock:      10050,
		LockedStake:         "22000000",
		StakeUnlocksAtBlock: 57777,
		InheritanceDiscount: 1_000_000,
		Metrics: &types.QualificationMetrics{
			TotalVerifications:    100,
			CorrectVerifications:  90,
			Accuracy:              900_000,
			LastVerificationBlock: 99,
			ConsecutiveCorrect:    5,
			ConsecutiveIncorrect:  0,
			ChallengesReceived:    2,
			ChallengesLost:        0,
			EffectiveWeight:       1_000_000,
		},
		Endorsers:     []string{testAddr("e1"), testAddr("e2")},
		InheritedFrom: "physics",
	}

	k.SetQualification(ctx, qual)

	got, found := k.GetQualification(ctx, testAddr("val1"), "mathematics")
	if !found {
		t.Fatal("qualification not found after set")
	}

	if got.ValidatorAddr != qual.ValidatorAddr {
		t.Errorf("ValidatorAddr: got %s, want %s", got.ValidatorAddr, qual.ValidatorAddr)
	}
	if got.Domain != qual.Domain {
		t.Errorf("Domain: got %s, want %s", got.Domain, qual.Domain)
	}
	if got.Stratum != qual.Stratum {
		t.Errorf("Stratum: got %d, want %d", got.Stratum, qual.Stratum)
	}
	if got.Pathway != qual.Pathway {
		t.Errorf("Pathway: got %s, want %s", got.Pathway, qual.Pathway)
	}
	if got.Status != qual.Status {
		t.Errorf("Status: got %s, want %s", got.Status, qual.Status)
	}
	if got.EarnedAtBlock != qual.EarnedAtBlock {
		t.Errorf("EarnedAtBlock: got %d, want %d", got.EarnedAtBlock, qual.EarnedAtBlock)
	}
	if got.ExpiresAtBlock != qual.ExpiresAtBlock {
		t.Errorf("ExpiresAtBlock: got %d, want %d", got.ExpiresAtBlock, qual.ExpiresAtBlock)
	}
	if got.LockedStake != qual.LockedStake {
		t.Errorf("LockedStake: got %s, want %s", got.LockedStake, qual.LockedStake)
	}
	if got.StakeUnlocksAtBlock != qual.StakeUnlocksAtBlock {
		t.Errorf("StakeUnlocksAtBlock: got %d, want %d", got.StakeUnlocksAtBlock, qual.StakeUnlocksAtBlock)
	}
	if got.InheritanceDiscount != qual.InheritanceDiscount {
		t.Errorf("InheritanceDiscount: got %d, want %d", got.InheritanceDiscount, qual.InheritanceDiscount)
	}
	if got.Metrics.TotalVerifications != qual.Metrics.TotalVerifications {
		t.Errorf("Metrics.TotalVerifications: got %d, want %d", got.Metrics.TotalVerifications, qual.Metrics.TotalVerifications)
	}
	if got.Metrics.CorrectVerifications != qual.Metrics.CorrectVerifications {
		t.Errorf("Metrics.CorrectVerifications: got %d, want %d", got.Metrics.CorrectVerifications, qual.Metrics.CorrectVerifications)
	}
	if got.Metrics.Accuracy != qual.Metrics.Accuracy {
		t.Errorf("Metrics.Accuracy: got %d, want %d", got.Metrics.Accuracy, qual.Metrics.Accuracy)
	}
	if got.Metrics.EffectiveWeight != qual.Metrics.EffectiveWeight {
		t.Errorf("Metrics.EffectiveWeight: got %d, want %d", got.Metrics.EffectiveWeight, qual.Metrics.EffectiveWeight)
	}
	if got.InheritedFrom != qual.InheritedFrom {
		t.Errorf("InheritedFrom: got %s, want %s", got.InheritedFrom, qual.InheritedFrom)
	}
	if len(got.Endorsers) != 2 {
		t.Errorf("Endorsers length: got %d, want 2", len(got.Endorsers))
	}
}

func TestIsolatedReputationRoundtrip(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	rep := types.IsolatedReputation{
		ValidatorAddr: testAddr("val1"),
		GlobalReputation: &types.GlobalReputation{
			Score:                800,
			TotalVerifications:   50,
			CorrectVerifications: 40,
			Accuracy:             800_000,
		},
		StratumReputations: []*types.StratumReputationEntry{
			{
				Key: 3,
				Value: &types.StratumReputation{
					Stratum:                3,
					Score:                  750,
					TotalVerifications:     30,
					CorrectVerifications:   24,
					Accuracy:               800_000,
					DomainsVerified:        2,
					FirstVerificationBlock: 10,
					LastActivityBlock:      90,
					LastDecayBlock:         80,
				},
			},
		},
		DomainReputations: []*types.DomainReputationEntry{
			{
				Key: "mathematics",
				Value: &types.DomainReputation{
					Domain:                 "mathematics",
					Stratum:                3,
					Score:                  850,
					TotalVerifications:     25,
					CorrectVerifications:   22,
					Accuracy:               880_000,
					VerdictConsistency:     700_000,
					ConfidenceCalibration:  600_000,
					ChallengesReceived:     3,
					ChallengesLost:         1,
					ChallengeDefenseRate:   666_666,
					ScoreConfidence:        250_000,
					FirstVerificationBlock: 15,
					LastVerificationBlock:  85,
					DecayedScore:           820,
					LastDecayBlock:         80,
				},
			},
		},
		CreatedAtBlock:   5,
		LastUpdatedBlock: 90,
	}

	k.SetIsolatedReputation(ctx, rep)

	got, found := k.GetIsolatedReputation(ctx, testAddr("val1"))
	if !found {
		t.Fatal("reputation not found after set")
	}

	if got.ValidatorAddr != rep.ValidatorAddr {
		t.Errorf("ValidatorAddr: got %s, want %s", got.ValidatorAddr, rep.ValidatorAddr)
	}
	if got.GlobalReputation.Score != rep.GlobalReputation.Score {
		t.Errorf("GlobalReputation.Score: got %d, want %d", got.GlobalReputation.Score, rep.GlobalReputation.Score)
	}
	if got.GlobalReputation.TotalVerifications != rep.GlobalReputation.TotalVerifications {
		t.Errorf("GlobalReputation.TotalVerifications: got %d, want %d", got.GlobalReputation.TotalVerifications, rep.GlobalReputation.TotalVerifications)
	}
	if got.GlobalReputation.Accuracy != rep.GlobalReputation.Accuracy {
		t.Errorf("GlobalReputation.Accuracy: got %d, want %d", got.GlobalReputation.Accuracy, rep.GlobalReputation.Accuracy)
	}
	if got.CreatedAtBlock != rep.CreatedAtBlock {
		t.Errorf("CreatedAtBlock: got %d, want %d", got.CreatedAtBlock, rep.CreatedAtBlock)
	}
	if got.LastUpdatedBlock != rep.LastUpdatedBlock {
		t.Errorf("LastUpdatedBlock: got %d, want %d", got.LastUpdatedBlock, rep.LastUpdatedBlock)
	}

	sr := findTestStratumRep(got.StratumReputations, 3)
	if sr == nil {
		t.Fatal("stratum reputation for stratum 3 not found")
	}
	if sr.Score != 750 {
		t.Errorf("StratumReputation[3].Score: got %d, want 750", sr.Score)
	}
	if sr.TotalVerifications != 30 {
		t.Errorf("StratumReputation[3].TotalVerifications: got %d, want 30", sr.TotalVerifications)
	}

	dr := findTestDomainRep(got.DomainReputations, "mathematics")
	if dr == nil {
		t.Fatal("domain reputation for mathematics not found")
	}
	if dr.Score != 850 {
		t.Errorf("DomainReputation[mathematics].Score: got %d, want 850", dr.Score)
	}
	if dr.VerdictConsistency != 700_000 {
		t.Errorf("DomainReputation[mathematics].VerdictConsistency: got %d, want 700000", dr.VerdictConsistency)
	}
	if dr.ConfidenceCalibration != 600_000 {
		t.Errorf("DomainReputation[mathematics].ConfidenceCalibration: got %d, want 600000", dr.ConfidenceCalibration)
	}
	if dr.DecayedScore != 820 {
		t.Errorf("DomainReputation[mathematics].DecayedScore: got %d, want 820", dr.DecayedScore)
	}
}

func TestParamsValidation(t *testing.T) {
	params := types.DefaultParams()

	if err := params.Validate(); err != nil {
		t.Fatalf("default params should be valid, got error: %v", err)
	}

	// Verify default values match constants
	if params.StakeLockPeriod != types.StakeLockPeriod {
		t.Errorf("StakeLockPeriod: got %d, want %d", params.StakeLockPeriod, types.StakeLockPeriod)
	}
	if params.MinVerificationsForQual != types.MinVerificationsForQual {
		t.Errorf("MinVerificationsForQual: got %d, want %d", params.MinVerificationsForQual, types.MinVerificationsForQual)
	}
	if params.MinAccuracyForQual != types.MinAccuracyForQual {
		t.Errorf("MinAccuracyForQual: got %d, want %d", params.MinAccuracyForQual, types.MinAccuracyForQual)
	}
	if params.ReputationHalfLifeBlocks != types.ReputationHalfLifeBlocks {
		t.Errorf("ReputationHalfLifeBlocks: got %d, want %d", params.ReputationHalfLifeBlocks, types.ReputationHalfLifeBlocks)
	}
	if params.HerfindahlTriggerThreshold != types.HerfindahlTriggerThreshold {
		t.Errorf("HerfindahlTriggerThreshold: got %d, want %d", params.HerfindahlTriggerThreshold, types.HerfindahlTriggerThreshold)
	}

	// Invalid: zero stake lock period
	invalid := params
	invalid.StakeLockPeriod = 0
	if err := invalid.Validate(); err == nil {
		t.Error("params with zero StakeLockPeriod should be invalid")
	}

	// Invalid: accuracy > 1,000,000
	invalid2 := params
	invalid2.MinAccuracyForQual = 1_000_001
	if err := invalid2.Validate(); err == nil {
		t.Error("params with MinAccuracyForQual > 1,000,000 should be invalid")
	}

	// Invalid: HHI threshold > 1,000,000
	invalid3 := params
	invalid3.HerfindahlTriggerThreshold = 1_000_001
	if err := invalid3.Validate(); err == nil {
		t.Error("params with HerfindahlTriggerThreshold > 1,000,000 should be invalid")
	}
}

// =============================================================================
// Qualification Tests (7)
// =============================================================================

func TestStakeCommitmentQualificationSuccess(t *testing.T) {
	k, ctx, _, mockOK := setupKeeper(t)

	domain := "thermodynamics"
	validatorAddr := testAddr("val1")

	// Set up ontology: leaf domain (depth=3), stratum=5
	mockOK.domainExists[domain] = true
	mockOK.domainDepths[domain] = 3
	mockOK.domainStrata[domain] = 5

	// Leaf domain requires StakeLeafDomain = "22000000"
	// Provide 30M stake (above 22M threshold)
	qual, err := k.ProcessQualificationRequest(
		ctx, validatorAddr, domain,
		types.PathwayStakeCommitment,
		"30000000", // 30M stake
		nil,        // no endorsers
		"",         // no inheritance
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Stake commitment starts as probationary
	if qual.Status != string(types.StatusProbationary) {
		t.Errorf("Status: got %s, want %s", qual.Status, types.StatusProbationary)
	}
	if qual.Pathway != string(types.PathwayStakeCommitment) {
		t.Errorf("Pathway: got %s, want %s", qual.Pathway, types.PathwayStakeCommitment)
	}
	if qual.LockedStake != "30000000" {
		t.Errorf("LockedStake: got %s, want 30000000", qual.LockedStake)
	}
	if qual.Stratum != 5 {
		t.Errorf("Stratum: got %d, want 5", qual.Stratum)
	}
	if qual.InheritanceDiscount != 1_000_000 {
		t.Errorf("InheritanceDiscount: got %d, want 1000000", qual.InheritanceDiscount)
	}
	if qual.ExpiresAtBlock != 0 {
		t.Errorf("ExpiresAtBlock: got %d, want 0 (no expiry for stake)", qual.ExpiresAtBlock)
	}

	// Verify it was persisted
	stored, found := k.GetQualification(ctx, validatorAddr, domain)
	if !found {
		t.Fatal("qualification should be stored")
	}
	if stored.ValidatorAddr != validatorAddr {
		t.Errorf("stored ValidatorAddr mismatch")
	}
}

func TestStakeCommitmentInsufficientStake(t *testing.T) {
	k, ctx, _, mockOK := setupKeeper(t)

	domain := "thermodynamics"
	validatorAddr := testAddr("val1")

	mockOK.domainExists[domain] = true
	mockOK.domainDepths[domain] = 3 // leaf, requires 22M
	mockOK.domainStrata[domain] = 5

	// Provide only 10M (below 22M)
	_, err := k.ProcessQualificationRequest(
		ctx, validatorAddr, domain,
		types.PathwayStakeCommitment,
		"10000000", // 10M (insufficient)
		nil, "",
	)
	if err == nil {
		t.Fatal("expected error for insufficient stake")
	}
	if err != types.ErrInsufficientStake {
		t.Errorf("expected ErrInsufficientStake, got: %v", err)
	}
}

func TestTrackRecordQualificationSuccess(t *testing.T) {
	k, ctx, _, mockOK := setupKeeper(t)

	domain := "mathematics"
	validatorAddr := testAddr("val1")

	mockOK.domainExists[domain] = true
	mockOK.domainDepths[domain] = 3
	mockOK.domainStrata[domain] = 3

	// Set up domain reputation: 19 verifications, 77% accuracy (15 correct out of 19)
	// MinVerificationsForQual = 19, MinAccuracyForQual = 770_000 (77%)
	rep := types.IsolatedReputation{
		ValidatorAddr:      validatorAddr,
		GlobalReputation:   &types.GlobalReputation{Score: 700, TotalVerifications: 19, CorrectVerifications: 15, Accuracy: 789_473},
		StratumReputations: nil,
		DomainReputations: []*types.DomainReputationEntry{
			{
				Key: domain,
				Value: &types.DomainReputation{
					Domain:               domain,
					Stratum:              3,
					Score:                750,
					TotalVerifications:   19,
					CorrectVerifications: 15,
					Accuracy:             789_473, // 15/19 * 1_000_000 = 789_473
				},
			},
		},
		CreatedAtBlock:   10,
		LastUpdatedBlock: 90,
	}
	k.SetIsolatedReputation(ctx, rep)

	qual, err := k.ProcessQualificationRequest(
		ctx, validatorAddr, domain,
		types.PathwayTrackRecord,
		"", nil, "",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Track record grants active status directly
	if qual.Status != string(types.StatusActive) {
		t.Errorf("Status: got %s, want %s", qual.Status, types.StatusActive)
	}
	if qual.Pathway != string(types.PathwayTrackRecord) {
		t.Errorf("Pathway: got %s, want %s", qual.Pathway, types.PathwayTrackRecord)
	}
	if qual.Metrics.TotalVerifications != 19 {
		t.Errorf("Metrics.TotalVerifications: got %d, want 19", qual.Metrics.TotalVerifications)
	}
	if qual.Metrics.CorrectVerifications != 15 {
		t.Errorf("Metrics.CorrectVerifications: got %d, want 15", qual.Metrics.CorrectVerifications)
	}
	if qual.Metrics.EffectiveWeight != 1_000_000 {
		t.Errorf("Metrics.EffectiveWeight: got %d, want 1000000", qual.Metrics.EffectiveWeight)
	}
	if qual.ExpiresAtBlock == 0 {
		t.Error("track record qualification should have expiration")
	}
}

func TestTrackRecordInsufficientVerifications(t *testing.T) {
	k, ctx, _, mockOK := setupKeeper(t)

	domain := "mathematics"
	validatorAddr := testAddr("val1")

	mockOK.domainExists[domain] = true
	mockOK.domainDepths[domain] = 3
	mockOK.domainStrata[domain] = 3

	// Only 10 verifications (below 19 required)
	rep := types.IsolatedReputation{
		ValidatorAddr:      validatorAddr,
		GlobalReputation:   &types.GlobalReputation{Score: 700, TotalVerifications: 10, CorrectVerifications: 8, Accuracy: 800_000},
		StratumReputations: nil,
		DomainReputations: []*types.DomainReputationEntry{
			{
				Key: domain,
				Value: &types.DomainReputation{
					Domain:               domain,
					TotalVerifications:   10,
					CorrectVerifications: 8,
					Accuracy:             800_000,
				},
			},
		},
		CreatedAtBlock:   10,
		LastUpdatedBlock: 90,
	}
	k.SetIsolatedReputation(ctx, rep)

	_, err := k.ProcessQualificationRequest(
		ctx, validatorAddr, domain,
		types.PathwayTrackRecord,
		"", nil, "",
	)
	if err == nil {
		t.Fatal("expected error for insufficient verifications")
	}
	if err != types.ErrInsufficientTrackRecord {
		t.Errorf("expected ErrInsufficientTrackRecord, got: %v", err)
	}
}

func TestCrossReferenceQualificationSuccess(t *testing.T) {
	k, ctx, _, mockOK := setupKeeper(t)

	domain := "physics"
	validatorAddr := testAddr("new_val")

	mockOK.domainExists[domain] = true
	mockOK.domainDepths[domain] = 3
	mockOK.domainStrata[domain] = 5

	// Create 3 existing qualified validators as endorsers (MinEndorsersForQual = 3)
	endorser1 := testAddr("endorser1")
	endorser2 := testAddr("endorser2")
	endorser3 := testAddr("endorser3")

	for _, e := range []string{endorser1, endorser2, endorser3} {
		k.SetQualification(ctx, types.DomainQualification{
			ValidatorAddr: e,
			Domain:        domain,
			Pathway:       string(types.PathwayStakeCommitment),
			Status:        string(types.StatusActive),
		})
	}

	qual, err := k.ProcessQualificationRequest(
		ctx, validatorAddr, domain,
		types.PathwayCrossReference,
		"",
		[]string{endorser1, endorser2, endorser3},
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cross-reference grants probationary status
	if qual.Status != string(types.StatusProbationary) {
		t.Errorf("Status: got %s, want %s", qual.Status, types.StatusProbationary)
	}
	if qual.Pathway != string(types.PathwayCrossReference) {
		t.Errorf("Pathway: got %s, want %s", qual.Pathway, types.PathwayCrossReference)
	}
	if len(qual.Endorsers) != 3 {
		t.Errorf("Endorsers length: got %d, want 3", len(qual.Endorsers))
	}
	if qual.ExpiresAtBlock == 0 {
		t.Error("cross-reference qualification should have expiration")
	}
}

func TestStratumInheritanceSuccess(t *testing.T) {
	k, ctx, _, mockOK := setupKeeper(t)

	parentDomain := "physics"
	childDomain := "thermodynamics"
	validatorAddr := testAddr("val1")

	mockOK.domainExists[parentDomain] = true
	mockOK.domainExists[childDomain] = true
	mockOK.domainDepths[childDomain] = 4
	mockOK.domainStrata[childDomain] = 5
	mockOK.parentDomains[childDomain] = parentDomain

	// Create active qualification in parent domain
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr:       validatorAddr,
		Domain:              parentDomain,
		Pathway:             string(types.PathwayStakeCommitment),
		Status:              string(types.StatusActive),
		InheritanceDiscount: 1_000_000,
	})

	qual, err := k.ProcessQualificationRequest(
		ctx, validatorAddr, childDomain,
		types.PathwayStratumInheritance,
		"", nil,
		parentDomain,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if qual.Status != string(types.StatusProbationary) {
		t.Errorf("Status: got %s, want %s", qual.Status, types.StatusProbationary)
	}
	if qual.Pathway != string(types.PathwayStratumInheritance) {
		t.Errorf("Pathway: got %s, want %s", qual.Pathway, types.PathwayStratumInheritance)
	}
	if qual.InheritedFrom != parentDomain {
		t.Errorf("InheritedFrom: got %s, want %s", qual.InheritedFrom, parentDomain)
	}
	// StratumInheritanceDiscount = 700_000 (70% of 1,000,000)
	if qual.InheritanceDiscount != types.StratumInheritanceDiscount {
		t.Errorf("InheritanceDiscount: got %d, want %d", qual.InheritanceDiscount, types.StratumInheritanceDiscount)
	}
	if qual.Metrics.EffectiveWeight != types.StratumInheritanceDiscount {
		t.Errorf("Metrics.EffectiveWeight: got %d, want %d (70%% of full weight)", qual.Metrics.EffectiveWeight, types.StratumInheritanceDiscount)
	}
}

func TestQualificationExpiration(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	validatorAddr := testAddr("val1")
	domain := "mathematics"
	currentBlock := uint64(ctx.BlockHeight()) // 100

	// Create qualification that expires at block 200
	qual := types.DomainQualification{
		ValidatorAddr:       validatorAddr,
		Domain:              domain,
		Pathway:             string(types.PathwayTrackRecord),
		Status:              string(types.StatusActive),
		EarnedAtBlock:       currentBlock,
		ExpiresAtBlock:      currentBlock + 100, // expires at 200
		InheritanceDiscount: 1_000_000,
		Metrics:             &types.QualificationMetrics{EffectiveWeight: 1_000_000},
	}
	k.SetQualification(ctx, qual)

	// Verify it's active at current block
	if !k.IsQualifiedForDomain(ctx, validatorAddr, domain, currentBlock) {
		t.Error("should be qualified at current block")
	}

	// Advance context past expiration (block 201)
	expiredCtx := ctx.WithBlockHeight(201)

	// IsQualifiedForDomain should return false for expired
	if k.IsQualifiedForDomain(expiredCtx, validatorAddr, domain, 201) {
		t.Error("should NOT be qualified past expiration block")
	}

	// Simulate BeginBlock expiration logic: iterate and expire
	k.IterateQualifications(expiredCtx, func(q types.DomainQualification) bool {
		if q.ExpiresAtBlock > 0 && uint64(201) > q.ExpiresAtBlock {
			if q.Status == string(types.StatusActive) || q.Status == string(types.StatusProbationary) {
				q.Status = string(types.StatusExpired)
				q.Metrics.EffectiveWeight = 0
				k.SetQualification(expiredCtx, q)
			}
		}
		return false
	})

	// Verify status changed to expired
	stored, found := k.GetQualification(expiredCtx, validatorAddr, domain)
	if !found {
		t.Fatal("qualification should still exist")
	}
	if stored.Status != string(types.StatusExpired) {
		t.Errorf("Status: got %s, want %s", stored.Status, types.StatusExpired)
	}
	if stored.Metrics.EffectiveWeight != 0 {
		t.Errorf("EffectiveWeight: got %d, want 0", stored.Metrics.EffectiveWeight)
	}
}

// =============================================================================
// Reputation Tests (5)
// =============================================================================

func TestReputationScoreCalculation(t *testing.T) {
	// CalculateDomainReputationScore uses:
	// accuracy (50%) + consistency (25%) + challenge_defense (15%) + calibration (10%)
	// Each component is scaled to MaxScore (1000), then weighted.

	// Setup: accuracy=800000 basis pts, consistency=700000, challengeDefense=600000, calibration=500000
	// We need to provide raw values that produce the target component scores.
	//
	// Accuracy: Laplace smoothing: (correct+1)/(total+2) * MaxScore
	//   For 80% raw: 80 correct out of 100. (81/102)*1000 = 794 (Laplace adjusted)
	//   But we want to test the weighted composite, so let's pick values that produce known outputs.
	//
	// For simplicity: 100 total, 80 correct
	// accuracyScore = (80+1)*1000/(100+2) = 81000/102 = 794
	// consistencyScore = 700000 * 1000 / 1_000_000 = 700
	// challengeScore: 0 challenges = 1000 (perfect defense)
	// calibrationScore = 500000 * 1000 / 1_000_000 = 500
	//
	// Weighted: (794*500000 + 700*250000 + 1000*150000 + 500*100000) / 1_000_000
	//         = (397_000_000 + 175_000_000 + 150_000_000 + 50_000_000) / 1_000_000
	//         = 772_000_000 / 1_000_000 = 772

	score, _, _ := keeper.CalculateDomainReputationScore(
		100, 80, // totalVerifications, correctVerifications
		700_000, // verdictConsistency
		0, 0,    // challengesReceived, challengesLost (0 challenges = perfect defense)
		500_000,  // confidenceCalibration
		100, 100, // currentBlock, lastActivityBlock (no decay)
	)

	expected := uint64(772)
	if score != expected {
		t.Errorf("reputation score: got %d, want %d", score, expected)
	}
}

func TestEffectiveReputationFallback(t *testing.T) {
	k, ctx, _, mockOK := setupKeeper(t)

	domain := "mathematics"
	mockOK.domainStrata[domain] = 3

	validator1 := testAddr("v1_domain")
	validator2 := testAddr("v2_stratum")
	validator3 := testAddr("v3_global")

	currentBlock := uint64(ctx.BlockHeight())

	// Validator 1: has domain reputation (should use domain source)
	rep1 := types.IsolatedReputation{
		ValidatorAddr:      validator1,
		GlobalReputation:   &types.GlobalReputation{Score: 500, TotalVerifications: 10, CorrectVerifications: 5, Accuracy: 500_000},
		StratumReputations: nil,
		DomainReputations: []*types.DomainReputationEntry{
			{
				Key: domain,
				Value: &types.DomainReputation{
					Domain:                domain,
					Stratum:               3,
					Score:                 800,
					TotalVerifications:    25, // above MinVerificationsForDomainRep (19)
					CorrectVerifications:  20,
					Accuracy:              800_000,
					DecayedScore:          780,
					ScoreConfidence:       250_000,
					LastVerificationBlock: currentBlock,
				},
			},
		},
		CreatedAtBlock:   10,
		LastUpdatedBlock: currentBlock,
	}
	k.SetIsolatedReputation(ctx, rep1)

	result1 := k.GetEffectiveReputation(ctx, validator1, domain)
	if result1.Source != string(types.SourceDomain) {
		t.Errorf("validator1 should use domain source, got %s", result1.Source)
	}
	if result1.Discount != 1_000_000 {
		t.Errorf("domain discount should be 1_000_000 (no discount), got %d", result1.Discount)
	}

	// Validator 2: no domain rep but has stratum reputation
	rep2 := types.IsolatedReputation{
		ValidatorAddr:    validator2,
		GlobalReputation: &types.GlobalReputation{Score: 500, TotalVerifications: 10, CorrectVerifications: 5, Accuracy: 500_000},
		StratumReputations: []*types.StratumReputationEntry{
			{
				Key: 3,
				Value: &types.StratumReputation{
					Stratum:            3,
					Score:              700,
					TotalVerifications: 25, // above threshold
					LastActivityBlock:  currentBlock,
				},
			},
		},
		DomainReputations: nil,
		CreatedAtBlock:    10,
		LastUpdatedBlock:  currentBlock,
	}
	k.SetIsolatedReputation(ctx, rep2)

	result2 := k.GetEffectiveReputation(ctx, validator2, domain)
	if result2.Source != string(types.SourceStratum) {
		t.Errorf("validator2 should fallback to stratum source, got %s", result2.Source)
	}
	if result2.Discount != types.StratumReputationDiscount {
		t.Errorf("stratum discount should be %d, got %d", types.StratumReputationDiscount, result2.Discount)
	}

	// Validator 3: no domain or stratum rep — falls back to global
	rep3 := types.IsolatedReputation{
		ValidatorAddr:      validator3,
		GlobalReputation:   &types.GlobalReputation{Score: 600, TotalVerifications: 30, CorrectVerifications: 18, Accuracy: 600_000},
		StratumReputations: nil,
		DomainReputations:  nil,
		CreatedAtBlock:     10,
		LastUpdatedBlock:   currentBlock,
	}
	k.SetIsolatedReputation(ctx, rep3)

	result3 := k.GetEffectiveReputation(ctx, validator3, domain)
	if result3.Source != string(types.SourceGlobal) {
		t.Errorf("validator3 should fallback to global source, got %s", result3.Source)
	}
	if result3.Discount != types.GlobalReputationDiscount {
		t.Errorf("global discount should be %d, got %d", types.GlobalReputationDiscount, result3.Discount)
	}
}

func TestReputationTimeDecay(t *testing.T) {
	// applyDecay: decayedScore = baseScore + (score - baseScore) * 0.5^(age/halfLife)
	// BaseScore = 500, score = 1000, halfLife = 77777
	// At age = halfLife: factor = 0.5
	// decayed = 500 + (1000-500)*0.5 = 500 + 250 = 750

	k, ctx, _, mockOK := setupKeeper(t)
	domain := "mathematics"
	mockOK.domainStrata[domain] = 3

	validatorAddr := testAddr("decay_val")

	// Create reputation at block 0 with score 1000
	rep := types.IsolatedReputation{
		ValidatorAddr: validatorAddr,
		GlobalReputation: &types.GlobalReputation{
			Score:                1000,
			TotalVerifications:   100,
			CorrectVerifications: 100,
			Accuracy:             1_000_000,
		},
		StratumReputations: nil,
		DomainReputations:  nil,
		CreatedAtBlock:     0,
		LastUpdatedBlock:   0,
	}
	k.SetIsolatedReputation(ctx, rep)

	// At block = halfLife (77777), global reputation falls back and applies decay
	decayCtx := ctx.WithBlockHeight(int64(types.ReputationHalfLifeBlocks))
	result := k.GetEffectiveReputation(decayCtx, validatorAddr, domain)

	// Uses global fallback: applyDecay(1000, 77777, 0)
	// age = 77777, factor = 0.5^(77777/77777) = 0.5
	// decayed = 500 + (1000-500)*0.5 = 750
	expected := uint64(750)
	tolerance := uint64(5) // Allow small floating point drift

	if result.Score < expected-tolerance || result.Score > expected+tolerance {
		t.Errorf("decayed score at half-life: got %d, want ~%d (tolerance %d)", result.Score, expected, tolerance)
	}
}

func TestUpdateAfterVerification(t *testing.T) {
	k, ctx, _, mockOK := setupKeeper(t)

	domain := "mathematics"
	validatorAddr := testAddr("val1")
	mockOK.domainExists[domain] = true
	mockOK.domainStrata[domain] = 3

	// Create a probationary qualification
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr:       validatorAddr,
		Domain:              domain,
		Pathway:             string(types.PathwayStakeCommitment),
		Status:              string(types.StatusProbationary),
		EarnedAtBlock:       100,
		ExpiresAtBlock:      0,
		InheritanceDiscount: 1_000_000,
		Metrics:             &types.QualificationMetrics{EffectiveWeight: types.ProbationaryWeight},
	})

	currentBlock := uint64(ctx.BlockHeight())

	// Perform a correct verification
	err := k.UpdateAfterVerification(ctx, validatorAddr, domain, 3, true, 800_000, 850_000, currentBlock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check qualification metrics updated
	qual, found := k.GetQualification(ctx, validatorAddr, domain)
	if !found {
		t.Fatal("qualification should exist")
	}
	if qual.Metrics.TotalVerifications != 1 {
		t.Errorf("TotalVerifications: got %d, want 1", qual.Metrics.TotalVerifications)
	}
	if qual.Metrics.CorrectVerifications != 1 {
		t.Errorf("CorrectVerifications: got %d, want 1", qual.Metrics.CorrectVerifications)
	}
	if qual.Metrics.ConsecutiveCorrect != 1 {
		t.Errorf("ConsecutiveCorrect: got %d, want 1", qual.Metrics.ConsecutiveCorrect)
	}
	if qual.Metrics.Accuracy != 1_000_000 {
		t.Errorf("Accuracy: got %d, want 1_000_000", qual.Metrics.Accuracy)
	}

	// Check reputation was created/updated
	rep, found := k.GetIsolatedReputation(ctx, validatorAddr)
	if !found {
		t.Fatal("reputation should exist after verification")
	}
	if rep.GlobalReputation.TotalVerifications != 1 {
		t.Errorf("GlobalReputation.TotalVerifications: got %d, want 1", rep.GlobalReputation.TotalVerifications)
	}
	if rep.GlobalReputation.CorrectVerifications != 1 {
		t.Errorf("GlobalReputation.CorrectVerifications: got %d, want 1", rep.GlobalReputation.CorrectVerifications)
	}

	dr := findTestDomainRep(rep.DomainReputations, domain)
	if dr == nil {
		t.Fatal("domain reputation should be created")
	}
	if dr.TotalVerifications != 1 {
		t.Errorf("DomainReputation.TotalVerifications: got %d, want 1", dr.TotalVerifications)
	}
	// Score should be > BaseScore since we had a correct verification
	if dr.Score <= types.BaseScore {
		t.Errorf("DomainReputation.Score: got %d, should be > BaseScore (%d)", dr.Score, types.BaseScore)
	}
}

func TestGetReputationScore(t *testing.T) {
	k, ctx, _, mockOK := setupKeeper(t)

	domain := "mathematics"
	validatorAddr := testAddr("val1")
	mockOK.domainStrata[domain] = 3

	currentBlock := uint64(ctx.BlockHeight())

	// Set up reputation with known global score (no domain/stratum rep -> global fallback)
	rep := types.IsolatedReputation{
		ValidatorAddr: validatorAddr,
		GlobalReputation: &types.GlobalReputation{
			Score:                800,
			TotalVerifications:   50,
			CorrectVerifications: 40,
			Accuracy:             800_000,
		},
		StratumReputations: nil,
		DomainReputations:  nil,
		CreatedAtBlock:     currentBlock,
		LastUpdatedBlock:   currentBlock,
	}
	k.SetIsolatedReputation(ctx, rep)

	// GetReputationScore: (score * discount) / 1_000_000
	// Global fallback: score=800 (no decay since lastUpdatedBlock == currentBlock), discount=GlobalReputationDiscount=500_000
	// Expected: (800 * 500_000) / 1_000_000 = 400
	result := k.GetReputationScore(ctx, validatorAddr, domain)

	expected := uint64(800) * types.GlobalReputationDiscount / 1_000_000
	if result != expected {
		t.Errorf("GetReputationScore: got %d, want %d", result, expected)
	}
}

// =============================================================================
// Cross-Stratum Tests (3)
// =============================================================================

func TestCrossStratumCoverageValid(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Stratum 4 (chemical) requires strata {4, 5} with min {4:2, 5:2}
	validatorStrata := map[string]uint32{
		testAddr("v1"): 4, // chemical
		testAddr("v2"): 4, // chemical
		testAddr("v3"): 5, // physical
		testAddr("v4"): 5, // physical
	}

	valid, confAdjust := k.ValidateCrossStratumCoverage(ctx, 4, validatorStrata)
	if !valid {
		t.Error("coverage should be valid")
	}
	if confAdjust != 1_000_000 {
		t.Errorf("confidence adjustment: got %d, want 1_000_000 (full coverage)", confAdjust)
	}
}

func TestCrossStratumCoverageConfidenceBoost(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Stratum 6 (fundamental) requires only {6:3} — self-contained
	validatorStrata := map[string]uint32{
		testAddr("v1"): 6,
		testAddr("v2"): 6,
		testAddr("v3"): 6,
	}

	valid, confAdjust := k.ValidateCrossStratumCoverage(ctx, 6, validatorStrata)
	if !valid {
		t.Error("fundamental stratum coverage should be valid")
	}
	if confAdjust != 1_000_000 {
		t.Errorf("confidence adjustment: got %d, want 1_000_000", confAdjust)
	}
}

func TestCrossStratumPartialPenalty(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Stratum 4 (chemical) requires strata {4, 5} with min {4:2, 5:2}
	// Only provide stratum 4 validators, missing stratum 5
	validatorStrata := map[string]uint32{
		testAddr("v1"): 4,
		testAddr("v2"): 4,
	}

	valid, confAdjust := k.ValidateCrossStratumCoverage(ctx, 4, validatorStrata)
	if !valid {
		t.Error("partial coverage should still return valid=true (with penalty)")
	}

	// PartialCoverageConfPenalty = 200_000
	expectedConf := uint64(1_000_000) - types.PartialCoverageConfPenalty
	if confAdjust != expectedConf {
		t.Errorf("confidence adjustment: got %d, want %d (partial penalty)", confAdjust, expectedConf)
	}
}

// =============================================================================
// Detection Tests (4)
// =============================================================================

func TestHHICalculation(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	domain := "captured_domain"

	// Create verification history where one validator has ~60% of participations.
	// 20 entries: validator "dominant" in all 20, validators "minor1"-"minor4" in 8 each
	// Total participations: 20 + 8*4 = 52 (but we put all in the same entries)
	//
	// Alternative: 20 entries where "dominant" participates in all 20, and
	// "minor1" and "minor2" participate in 10 each.
	// Total participations: 20 + 10 + 10 = 40
	// dominant share = 20/40 = 0.5 -> share^2 = 0.25
	// minor1 share = 10/40 = 0.25 -> share^2 = 0.0625
	// minor2 share = 10/40 = 0.25 -> share^2 = 0.0625
	// HHI = 0.25 + 0.0625 + 0.0625 = 0.375 = 375_000 basis points

	// Let's make it more concentrated: dominant gets 12/20 entries alone
	// 20 entries where dominant is in 12 entries with 1 validator each,
	// and 8 entries with 2 validators (dominant + minor)
	// Simpler: create entries with controlled participation counts

	entries := make([]*types.VerificationHistoryEntry, 20)
	dominant := testAddr("dominant")
	minor1 := testAddr("minor1")

	// 12 entries: only dominant
	for i := 0; i < 12; i++ {
		entries[i] = &types.VerificationHistoryEntry{
			ClaimId:     "claim_" + string(rune('A'+i)),
			BlockNumber: uint64(10 + i),
			Validators:  []string{dominant},
			Verdicts:    []string{"confirmed"},
		}
	}
	// 8 entries: dominant + minor1
	for i := 12; i < 20; i++ {
		entries[i] = &types.VerificationHistoryEntry{
			ClaimId:     "claim_" + string(rune('A'+i)),
			BlockNumber: uint64(10 + i),
			Validators:  []string{dominant, minor1},
			Verdicts:    []string{"confirmed", "confirmed"},
		}
	}

	history := types.VerificationHistory{Domain: domain, Entries: entries}
	k.SetVerificationHistory(ctx, history)

	// Total participations: 12*1 + 8*2 = 28
	// dominant: 20/28 = 0.714..., share^2 = 0.5102
	// minor1: 8/28 = 0.2857, share^2 = 0.0816
	// HHI = 0.5102 + 0.0816 = 0.5918 -> ~591_800 basis points
	metrics := k.AnalyzeCaptureRisk(ctx, domain, 100)

	// HHI should be > 0.36 (360_000)
	if metrics.HerfindahlIndex < 360_000 {
		t.Errorf("HerfindahlIndex: got %d, want > 360_000", metrics.HerfindahlIndex)
	}
	// More precisely, should be around 591_000
	if metrics.HerfindahlIndex < 500_000 {
		t.Errorf("HerfindahlIndex should indicate high concentration: got %d, want > 500_000", metrics.HerfindahlIndex)
	}
	if metrics.UniqueValidators != 2 {
		t.Errorf("UniqueValidators: got %d, want 2", metrics.UniqueValidators)
	}
}

func TestTimingCorrelation(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	domain := "timing_domain"

	// Create 20 entries with very similar submission timestamps (stddev < 2)
	entries := make([]*types.VerificationHistoryEntry, 20)
	for i := 0; i < 20; i++ {
		baseTimestamp := uint64(1000 + i*10)
		entries[i] = &types.VerificationHistoryEntry{
			ClaimId:     "claim_timing_" + string(rune('A'+i)),
			BlockNumber: uint64(10 + i),
			Validators:  []string{testAddr("v1"), testAddr("v2"), testAddr("v3")},
			Verdicts:    []string{"confirmed", "confirmed", "confirmed"},
			// All timestamps within 1 block of each other (stddev < 2)
			SubmissionTimestamps: []uint64{baseTimestamp, baseTimestamp, baseTimestamp + 1},
		}
	}

	history := types.VerificationHistory{Domain: domain, Entries: entries}
	k.SetVerificationHistory(ctx, history)

	metrics := k.AnalyzeCaptureRisk(ctx, domain, 100)

	// With stddev < 2 in every entry, timing correlation should be very high
	if metrics.TimingCorrelation < 800_000 {
		t.Errorf("TimingCorrelation: got %d, want > 800_000 (suspicious timing)", metrics.TimingCorrelation)
	}
}

func TestVerdictCorrelation(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	domain := "verdict_domain"

	// Create 20 entries where all validators give the same verdict every time
	entries := make([]*types.VerificationHistoryEntry, 20)
	for i := 0; i < 20; i++ {
		entries[i] = &types.VerificationHistoryEntry{
			ClaimId:     "claim_verdict_" + string(rune('A'+i)),
			BlockNumber: uint64(10 + i),
			Validators:  []string{testAddr("v1"), testAddr("v2"), testAddr("v3")},
			Verdicts:    []string{"confirmed", "confirmed", "confirmed"},
			SubmissionTimestamps: []uint64{
				uint64(1000 + i*100),
				uint64(1000 + i*100 + 50),
				uint64(1000 + i*100 + 80),
			},
		}
	}

	history := types.VerificationHistory{Domain: domain, Entries: entries}
	k.SetVerificationHistory(ctx, history)

	metrics := k.AnalyzeCaptureRisk(ctx, domain, 100)

	// All 20 entries have unanimous verdicts -> verdict correlation = 1_000_000
	if metrics.VerdictCorrelation != 1_000_000 {
		t.Errorf("VerdictCorrelation: got %d, want 1_000_000 (all unanimous)", metrics.VerdictCorrelation)
	}
}

func TestAutoBountyThreshold(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	domain := "monopoly_domain"

	// Create history where a single validator dominates: HHI = 1.0 = 1_000_000
	// (monopoly scenario)
	entries := make([]*types.VerificationHistoryEntry, 20)
	monopolist := testAddr("monopolist")
	for i := 0; i < 20; i++ {
		entries[i] = &types.VerificationHistoryEntry{
			ClaimId:              "claim_mono_" + string(rune('A'+i)),
			BlockNumber:          uint64(10 + i),
			Validators:           []string{monopolist},
			Verdicts:             []string{"confirmed"},
			SubmissionTimestamps: []uint64{uint64(1000 + i)},
		}
	}

	history := types.VerificationHistory{Domain: domain, Entries: entries}
	k.SetVerificationHistory(ctx, history)

	metrics := k.AnalyzeCaptureRisk(ctx, domain, 100)

	// HHI for monopoly = 1.0 = 1_000_000
	// This should be > HerfindahlTriggerThreshold (400_000)
	if metrics.HerfindahlIndex < types.HerfindahlTriggerThreshold {
		t.Errorf("HerfindahlIndex: got %d, should exceed trigger threshold %d", metrics.HerfindahlIndex, types.HerfindahlTriggerThreshold)
	}

	// Verify metrics are stored
	stored, found := k.GetCaptureMetrics(ctx, domain)
	if !found {
		t.Fatal("capture metrics should be stored after analysis")
	}
	if stored.Domain != domain {
		t.Errorf("stored domain: got %s, want %s", stored.Domain, domain)
	}
	if stored.HerfindahlIndex != metrics.HerfindahlIndex {
		t.Errorf("stored HHI: got %d, want %d", stored.HerfindahlIndex, metrics.HerfindahlIndex)
	}
	if stored.RiskScore == 0 {
		t.Error("risk score should be non-zero for monopoly domain")
	}
	if stored.UniqueValidators != 1 {
		t.Errorf("UniqueValidators: got %d, want 1", stored.UniqueValidators)
	}
}

// ---------- Test Helpers ----------

func findTestStratumRep(entries []*types.StratumReputationEntry, stratum uint32) *types.StratumReputation {
	for _, e := range entries {
		if e != nil && e.Key == stratum {
			return e.Value
		}
	}
	return nil
}

func findTestDomainRep(entries []*types.DomainReputationEntry, domain string) *types.DomainReputation {
	for _, e := range entries {
		if e != nil && e.Key == domain {
			return e.Value
		}
	}
	return nil
}

// Ensure math import is used (for tolerance comparison in decay test)
var _ = math.Abs
