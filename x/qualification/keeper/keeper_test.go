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

	ontologytypes "github.com/legible-money/legible/x/ontology/types"
	"github.com/legible-money/legible/x/qualification/keeper"
	"github.com/legible-money/legible/x/qualification/types"
	stakingtypes "github.com/legible-money/legible/x/staking/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.Seal()
}

// ---------- Mock OntologyKeeper ----------

type mockOntologyKeeper struct {
	domains map[string]ontologytypes.Domain
}

func newMockOntologyKeeper() *mockOntologyKeeper {
	return &mockOntologyKeeper{
		domains: make(map[string]ontologytypes.Domain),
	}
}

func (m *mockOntologyKeeper) addDomain(name string, stratum ontologytypes.Stratum) {
	m.domains[name] = ontologytypes.Domain{
		Name:    name,
		Stratum: uint32(stratum),
		Status:  "active",
	}
}

func (m *mockOntologyKeeper) GetDomain(_ sdk.Context, name string) (ontologytypes.Domain, bool) {
	d, ok := m.domains[name]
	return d, ok
}

// ---------- Mock StakingKeeper ----------

type mockStakingKeeper struct {
	validators map[string]stakingtypes.LegibleValidator
}

func newMockStakingKeeper() *mockStakingKeeper {
	return &mockStakingKeeper{
		validators: make(map[string]stakingtypes.LegibleValidator),
	}
}

func (m *mockStakingKeeper) addValidator(addr string, tier stakingtypes.ValidatorTier, verifications, correct uint64) {
	m.validators[addr] = stakingtypes.LegibleValidator{
		OperatorAddress:        addr,
		TotalStake:             "5000000000000000000000",
		Tier:                   tier.String(),
		IsActive:               true,
		VerificationsCompleted: verifications,
		VerificationsCorrect:   correct,
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

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockOntologyKeeper, *mockStakingKeeper) {
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

	mockOK := newMockOntologyKeeper()
	mockOK.addDomain("mathematics", ontologytypes.StratumAxiomatic) // stratum 0
	mockOK.addDomain("physics", ontologytypes.StratumEmpirical)     // stratum 4
	mockOK.addDomain("history", ontologytypes.StratumHistorical)    // stratum 5
	mockOK.addDomain("formal_logic", ontologytypes.StratumFormal)   // stratum 1

	mockSK := newMockStakingKeeper()
	mockSK.addValidator("lgmvaloper1abc", stakingtypes.TierBonded, 100, 90)
	mockSK.addValidator("lgmvaloper1def", stakingtypes.TierVerified, 50, 45)
	mockSK.addValidator("lgmvaloper1ghi", stakingtypes.TierGuardian, 200, 180)
	mockSK.addValidator("lgmvaloper1jkl", stakingtypes.TierApprentice, 5, 3)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		mockOK,
		mockSK,
		nil, // bankKeeper — nil-safe, skips bank operations in tests
		"authority",
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx, mockOK, mockSK
}

// ==================== State CRUD Tests ====================

func TestSetGetQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	q := types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "mathematics",
		Pathway:       string(types.PathwayStakeCommitment),
		Status:        string(types.StatusActive),
		EarnedAtBlock: 100,
	}

	k.SetQualification(ctx, q)
	got, found := k.GetQualification(ctx, "lgmvaloper1abc", "mathematics")
	if !found {
		t.Fatal("expected qualification to be found")
	}
	if got.ValidatorAddr != "lgmvaloper1abc" {
		t.Errorf("expected validator lgmvaloper1abc, got %s", got.ValidatorAddr)
	}
	if got.Domain != "mathematics" {
		t.Errorf("expected domain mathematics, got %s", got.Domain)
	}
	if got.Status != string(types.StatusActive) {
		t.Errorf("expected status active, got %s", got.Status)
	}
}

func TestGetQualification_NotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	_, found := k.GetQualification(ctx, "lgmvaloper1abc", "nonexistent")
	if found {
		t.Error("expected qualification to not be found")
	}
}

func TestDeleteQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	q := types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "mathematics",
		Status:        string(types.StatusActive),
	}
	k.SetQualification(ctx, q)

	k.DeleteQualification(ctx, "lgmvaloper1abc", "mathematics")
	_, found := k.GetQualification(ctx, "lgmvaloper1abc", "mathematics")
	if found {
		t.Error("expected qualification to be deleted")
	}
}

func TestGetQualificationsByValidator(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "mathematics",
		Status:        string(types.StatusActive),
	})
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusProbationary),
	})
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1def",
		Domain:        "mathematics",
		Status:        string(types.StatusActive),
	})

	qs := k.GetQualificationsByValidator(ctx, "lgmvaloper1abc")
	if len(qs) != 2 {
		t.Errorf("expected 2 qualifications for lgmvaloper1abc, got %d", len(qs))
	}
}

func TestSetGetEndorsement(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	e := types.QualificationEndorsement{
		Id:              "endorse_1",
		EndorserAddr:    "lgmvaloper1abc",
		TargetValidator: "lgmvaloper1def",
		Domain:          "mathematics",
		Stake:           "22000000",
		EndorsedAtBlock: 100,
		ExpiresAtBlock:  100000,
	}

	k.SetEndorsement(ctx, e)
	got, found := k.GetEndorsement(ctx, "endorse_1")
	if !found {
		t.Fatal("expected endorsement to be found")
	}
	if got.EndorserAddr != "lgmvaloper1abc" {
		t.Errorf("expected endorser lgmvaloper1abc, got %s", got.EndorserAddr)
	}
	if got.TargetValidator != "lgmvaloper1def" {
		t.Errorf("expected target lgmvaloper1def, got %s", got.TargetValidator)
	}
}

func TestGetEndorsementsForValidator(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetEndorsement(ctx, types.QualificationEndorsement{
		Id: "e1", EndorserAddr: "lgmvaloper1abc", TargetValidator: "lgmvaloper1def",
		Domain: "mathematics", EndorsedAtBlock: 100, ExpiresAtBlock: 200000,
	})
	k.SetEndorsement(ctx, types.QualificationEndorsement{
		Id: "e2", EndorserAddr: "lgmvaloper1ghi", TargetValidator: "lgmvaloper1def",
		Domain: "mathematics", EndorsedAtBlock: 100, ExpiresAtBlock: 200000,
	})
	k.SetEndorsement(ctx, types.QualificationEndorsement{
		Id: "e3", EndorserAddr: "lgmvaloper1abc", TargetValidator: "lgmvaloper1ghi",
		Domain: "physics", EndorsedAtBlock: 100, ExpiresAtBlock: 200000,
	})

	endorsements := k.GetEndorsementsForValidator(ctx, "lgmvaloper1def")
	if len(endorsements) != 2 {
		t.Errorf("expected 2 endorsements for lgmvaloper1def, got %d", len(endorsements))
	}
}

func TestGetNextEndorsementId(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	id1 := k.GetNextEndorsementId(ctx)
	id2 := k.GetNextEndorsementId(ctx)

	if id1 == id2 {
		t.Error("expected different endorsement IDs")
	}
	if id1 != "endorse_1" {
		t.Errorf("expected endorse_1, got %s", id1)
	}
	if id2 != "endorse_2" {
		t.Errorf("expected endorse_2, got %s", id2)
	}
}

func TestParamsSetGet(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	params := types.DefaultParams()
	params.MinVerifications = 42
	k.SetParams(ctx, params)

	got := k.GetParams(ctx)
	if got.MinVerifications != 42 {
		t.Errorf("expected MinVerifications 42, got %d", got.MinVerifications)
	}
}

// ==================== Pathway Tests ====================

func TestQualifyViaStakeCommitment_Success(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Physics is stratum 4 → StakeLeaf required (22 LGM)
	q, err := k.QualifyViaStakeCommitment(ctx, "lgmvaloper1abc", "physics", "22000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Status != string(types.StatusProbationary) {
		t.Errorf("expected probationary status, got %s", q.Status)
	}
	if q.Pathway != string(types.PathwayStakeCommitment) {
		t.Errorf("expected stake_commitment pathway, got %s", q.Pathway)
	}
	if q.LockedStake != "22000000" {
		t.Errorf("expected locked stake 22000000000000000000, got %s", q.LockedStake)
	}
}

func TestQualifyViaStakeCommitment_InsufficientStake(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Mathematics is stratum 0 → StakeTop required (2222 LGM)
	_, err := k.QualifyViaStakeCommitment(ctx, "lgmvaloper1abc", "mathematics", "22000000")
	if err == nil {
		t.Fatal("expected error for insufficient stake")
	}
}

func TestQualifyViaStakeCommitment_AlreadyQualified(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	_, err := k.QualifyViaStakeCommitment(ctx, "lgmvaloper1abc", "physics", "22000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = k.QualifyViaStakeCommitment(ctx, "lgmvaloper1abc", "physics", "22000000")
	if err == nil {
		t.Fatal("expected error for already qualified")
	}
}

func TestQualifyViaStakeCommitment_ValidatorNotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	_, err := k.QualifyViaStakeCommitment(ctx, "lgmvaloper1zzz", "physics", "22000000")
	if err == nil {
		t.Fatal("expected error for nonexistent validator")
	}
}

func TestQualifyViaStakeCommitment_DomainNotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	_, err := k.QualifyViaStakeCommitment(ctx, "lgmvaloper1abc", "nonexistent", "22000000")
	if err == nil {
		t.Fatal("expected error for nonexistent domain")
	}
}

func TestQualifyViaTrackRecord_Success(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// lgmvaloper1abc has 100 verifications, 90 correct (90% accuracy) > 77%
	q, err := k.QualifyViaTrackRecord(ctx, "lgmvaloper1abc", "physics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Status != string(types.StatusActive) {
		t.Errorf("expected active status, got %s", q.Status)
	}
	if q.Pathway != string(types.PathwayTrackRecord) {
		t.Errorf("expected track_record pathway, got %s", q.Pathway)
	}
}

func TestQualifyViaTrackRecord_InsufficientVerifications(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// lgmvaloper1jkl has only 5 verifications, need 19
	_, err := k.QualifyViaTrackRecord(ctx, "lgmvaloper1jkl", "physics")
	if err == nil {
		t.Fatal("expected error for insufficient verifications")
	}
}

func TestQualifyViaTrackRecord_InsufficientAccuracy(t *testing.T) {
	k, ctx, _, mockSK := setupKeeper(t)

	// Add a validator with enough verifications but low accuracy (50%)
	mockSK.addValidator("lgmvaloper1low", stakingtypes.TierBonded, 100, 50)

	_, err := k.QualifyViaTrackRecord(ctx, "lgmvaloper1low", "physics")
	if err == nil {
		t.Fatal("expected error for insufficient accuracy")
	}
}

func TestQualifyViaCrossReference_Success(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// First, qualify 3 endorsers in the domain
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusActive),
	})
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1ghi",
		Domain:        "physics",
		Status:        string(types.StatusActive),
	})
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1jkl",
		Domain:        "physics",
		Status:        string(types.StatusActive),
	})

	// Create endorsements
	k.SetEndorsement(ctx, types.QualificationEndorsement{
		Id: "e1", EndorserAddr: "lgmvaloper1abc", TargetValidator: "lgmvaloper1def",
		Domain: "physics", EndorsedAtBlock: 50, ExpiresAtBlock: 200000,
	})
	k.SetEndorsement(ctx, types.QualificationEndorsement{
		Id: "e2", EndorserAddr: "lgmvaloper1ghi", TargetValidator: "lgmvaloper1def",
		Domain: "physics", EndorsedAtBlock: 50, ExpiresAtBlock: 200000,
	})
	k.SetEndorsement(ctx, types.QualificationEndorsement{
		Id: "e3", EndorserAddr: "lgmvaloper1jkl", TargetValidator: "lgmvaloper1def",
		Domain: "physics", EndorsedAtBlock: 50, ExpiresAtBlock: 200000,
	})

	q, err := k.QualifyViaCrossReference(ctx, "lgmvaloper1def", "physics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Status != string(types.StatusProbationary) {
		t.Errorf("expected probationary, got %s", q.Status)
	}
	if len(q.Endorsers) != 3 {
		t.Errorf("expected 3 endorsers, got %d", len(q.Endorsers))
	}
}

func TestQualifyViaCrossReference_InsufficientEndorsements(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Only 1 qualified endorser (need 3)
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusActive),
	})
	k.SetEndorsement(ctx, types.QualificationEndorsement{
		Id: "e1", EndorserAddr: "lgmvaloper1abc", TargetValidator: "lgmvaloper1def",
		Domain: "physics", EndorsedAtBlock: 50, ExpiresAtBlock: 200000,
	})

	_, err := k.QualifyViaCrossReference(ctx, "lgmvaloper1def", "physics")
	if err == nil {
		t.Fatal("expected error for insufficient endorsements")
	}
}

func TestQualifyViaCrossReference_ExpiredEndorsementsNotCounted(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Qualify 3 endorsers
	for _, addr := range []string{"lgmvaloper1abc", "lgmvaloper1ghi", "lgmvaloper1jkl"} {
		k.SetQualification(ctx, types.DomainQualification{
			ValidatorAddr: addr, Domain: "physics", Status: string(types.StatusActive),
		})
	}

	// 2 valid + 1 expired endorsement
	k.SetEndorsement(ctx, types.QualificationEndorsement{
		Id: "e1", EndorserAddr: "lgmvaloper1abc", TargetValidator: "lgmvaloper1def",
		Domain: "physics", EndorsedAtBlock: 50, ExpiresAtBlock: 200000,
	})
	k.SetEndorsement(ctx, types.QualificationEndorsement{
		Id: "e2", EndorserAddr: "lgmvaloper1ghi", TargetValidator: "lgmvaloper1def",
		Domain: "physics", EndorsedAtBlock: 50, ExpiresAtBlock: 200000,
	})
	k.SetEndorsement(ctx, types.QualificationEndorsement{
		Id: "e3", EndorserAddr: "lgmvaloper1jkl", TargetValidator: "lgmvaloper1def",
		Domain: "physics", EndorsedAtBlock: 50, ExpiresAtBlock: 50, // already expired at block 100
	})

	_, err := k.QualifyViaCrossReference(ctx, "lgmvaloper1def", "physics")
	if err == nil {
		t.Fatal("expected error: expired endorsement should not count")
	}
}

func TestQualifyViaStratumInheritance_Success(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Qualify in lower stratum domain first (mathematics = stratum 0)
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "mathematics",
		Status:        string(types.StatusActive),
	})

	// Inherit to higher stratum domain (physics = stratum 4)
	q, err := k.QualifyViaStratumInheritance(ctx, "lgmvaloper1abc", "physics", "mathematics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Status != string(types.StatusProbationary) {
		t.Errorf("expected probationary, got %s", q.Status)
	}
	if q.InheritedFrom != "mathematics" {
		t.Errorf("expected inherited from mathematics, got %s", q.InheritedFrom)
	}
	if q.InheritanceDiscountBps != 7000 {
		t.Errorf("expected inheritance discount 7000, got %d", q.InheritanceDiscountBps)
	}
}

func TestQualifyViaStratumInheritance_WrongDirection(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Qualify in higher stratum domain first (physics = stratum 4)
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusActive),
	})

	// Try to inherit to lower stratum domain (mathematics = stratum 0) — should fail
	_, err := k.QualifyViaStratumInheritance(ctx, "lgmvaloper1abc", "mathematics", "physics")
	if err == nil {
		t.Fatal("expected error: cannot inherit from higher stratum to lower")
	}
}

func TestQualifyViaStratumInheritance_NoSourceQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Try to inherit without a qualification in the source domain
	_, err := k.QualifyViaStratumInheritance(ctx, "lgmvaloper1abc", "physics", "mathematics")
	if err == nil {
		t.Fatal("expected error for no source qualification")
	}
}

// ==================== Cross-Module Interface Tests ====================

func TestIsQualified_ActiveQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr:  "lgmvaloper1abc",
		Domain:         "mathematics",
		Status:         string(types.StatusActive),
		ExpiresAtBlock: 0, // no expiry
	})

	if !k.IsQualified(ctx, "lgmvaloper1abc", "mathematics") {
		t.Error("expected validator to be qualified")
	}
}

func TestIsQualified_ProbationaryQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr:  "lgmvaloper1abc",
		Domain:         "mathematics",
		Status:         string(types.StatusProbationary),
		ExpiresAtBlock: 0,
	})

	if !k.IsQualified(ctx, "lgmvaloper1abc", "mathematics") {
		t.Error("expected probationary validator to be qualified")
	}
}

func TestIsQualified_SuspendedNotQualified(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "mathematics",
		Status:        string(types.StatusSuspended),
	})

	if k.IsQualified(ctx, "lgmvaloper1abc", "mathematics") {
		t.Error("expected suspended validator to not be qualified")
	}
}

func TestIsQualified_ExpiredByBlock(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr:  "lgmvaloper1abc",
		Domain:         "mathematics",
		Status:         string(types.StatusActive),
		ExpiresAtBlock: 50, // expired at block 50, we're at block 100
	})

	if k.IsQualified(ctx, "lgmvaloper1abc", "mathematics") {
		t.Error("expected expired validator to not be qualified")
	}
}

func TestIsQualified_NotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	if k.IsQualified(ctx, "lgmvaloper1abc", "mathematics") {
		t.Error("expected non-existent qualification to return false")
	}
}

func TestGetQualificationWeight_Active(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr:          "lgmvaloper1abc",
		Domain:                 "mathematics",
		Status:                 string(types.StatusActive),
		InheritanceDiscountBps: 10000, // no discount
		Metrics: &types.QualificationMetrics{
			EffectiveWeightBps: 10000,
		},
	})

	weight := k.GetQualificationWeight(ctx, "lgmvaloper1abc", "mathematics")
	if weight == 0 {
		t.Error("expected non-zero weight for active qualification")
	}
}

func TestGetQualificationWeight_Suspended(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "mathematics",
		Status:        string(types.StatusSuspended),
	})

	weight := k.GetQualificationWeight(ctx, "lgmvaloper1abc", "mathematics")
	if weight != 0 {
		t.Errorf("expected zero weight for suspended, got %d", weight)
	}
}

func TestGetQualifiedValidators(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusActive),
	})
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1def",
		Domain:        "physics",
		Status:        string(types.StatusProbationary),
	})
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1ghi",
		Domain:        "physics",
		Status:        string(types.StatusSuspended), // should NOT be counted
	})

	validators := k.GetQualifiedValidators(ctx, "physics")
	if len(validators) != 2 {
		t.Errorf("expected 2 qualified validators, got %d", len(validators))
	}
}

func TestRecordVerificationOutcome_Correct(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusActive),
		Metrics:       &types.QualificationMetrics{EffectiveWeightBps: 10000},
	})

	err := k.RecordVerificationOutcome(ctx, "lgmvaloper1abc", "physics", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q, _ := k.GetQualification(ctx, "lgmvaloper1abc", "physics")
	if q.Metrics.TotalVerifications != 1 {
		t.Errorf("expected 1 total verification, got %d", q.Metrics.TotalVerifications)
	}
	if q.Metrics.CorrectVerifications != 1 {
		t.Errorf("expected 1 correct verification, got %d", q.Metrics.CorrectVerifications)
	}
	if q.Metrics.ConsecutiveCorrect != 1 {
		t.Errorf("expected 1 consecutive correct, got %d", q.Metrics.ConsecutiveCorrect)
	}
}

func TestRecordVerificationOutcome_Incorrect(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusActive),
		Metrics: &types.QualificationMetrics{
			TotalVerifications:   10,
			CorrectVerifications: 9,
			ConsecutiveCorrect:   5,
			EffectiveWeightBps:   10000,
		},
	})

	err := k.RecordVerificationOutcome(ctx, "lgmvaloper1abc", "physics", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q, _ := k.GetQualification(ctx, "lgmvaloper1abc", "physics")
	if q.Metrics.ConsecutiveCorrect != 0 {
		t.Errorf("expected consecutive correct reset to 0, got %d", q.Metrics.ConsecutiveCorrect)
	}
	if q.Metrics.ConsecutiveIncorrect != 1 {
		t.Errorf("expected 1 consecutive incorrect, got %d", q.Metrics.ConsecutiveIncorrect)
	}
}

func TestRecordVerificationOutcome_SuspensionByConsecutive(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusActive),
		Metrics: &types.QualificationMetrics{
			TotalVerifications:   10,
			CorrectVerifications: 8,
			ConsecutiveIncorrect: 4, // default SuspensionConsecutive = 5
			EffectiveWeightBps:   10000,
		},
	})

	// 5th consecutive incorrect should trigger suspension
	_ = k.RecordVerificationOutcome(ctx, "lgmvaloper1abc", "physics", false)

	q, _ := k.GetQualification(ctx, "lgmvaloper1abc", "physics")
	if q.Status != string(types.StatusSuspended) {
		t.Errorf("expected suspended after 5 consecutive incorrect, got %s", q.Status)
	}
}

func TestRecordVerificationOutcome_NoQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Should be a no-op, not an error
	err := k.RecordVerificationOutcome(ctx, "lgmvaloper1abc", "physics", true)
	if err != nil {
		t.Fatalf("expected no error for missing qualification, got: %v", err)
	}
}

// ==================== Lifecycle Tests ====================

func TestPromoteProbationary(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	q := types.DomainQualification{
		ValidatorAddr:          "lgmvaloper1abc",
		Domain:                 "physics",
		Status:                 string(types.StatusProbationary),
		EarnedAtBlock:          10,
		InheritanceDiscountBps: 10000,
		Metrics:                &types.QualificationMetrics{EffectiveWeightBps: 10000},
	}
	k.SetQualification(ctx, q)

	// At block 100, earned at block 10, probationary period = 777
	// 100 < 10 + 777 = 787, so should NOT promote
	promoted := k.PromoteProbationary(ctx, q)
	if promoted {
		t.Error("should not promote before probationary period ends")
	}

	// Advance context to block 1000 (past 10 + 777 = 787)
	ctx = ctx.WithBlockHeight(1000)
	q, _ = k.GetQualification(ctx, "lgmvaloper1abc", "physics")
	promoted = k.PromoteProbationary(ctx, q)
	if !promoted {
		t.Error("should promote after probationary period")
	}

	q, _ = k.GetQualification(ctx, "lgmvaloper1abc", "physics")
	if q.Status != string(types.StatusActive) {
		t.Errorf("expected active after promotion, got %s", q.Status)
	}
}

func TestRenewQualification_Active(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr:  "lgmvaloper1abc",
		Domain:         "physics",
		Status:         string(types.StatusActive),
		ExpiresAtBlock: 500,
	})

	err := k.RenewQualification(ctx, "lgmvaloper1abc", "physics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q, _ := k.GetQualification(ctx, "lgmvaloper1abc", "physics")
	// Should extend expiration: current block (100) + InactivityExpiration (777777)
	if q.ExpiresAtBlock != 100+777777 {
		t.Errorf("expected expiration at %d, got %d", 100+777777, q.ExpiresAtBlock)
	}
}

func TestRenewQualification_Expired(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr:  "lgmvaloper1abc",
		Domain:         "physics",
		Status:         string(types.StatusExpired),
		ExpiresAtBlock: 50,
	})

	err := k.RenewQualification(ctx, "lgmvaloper1abc", "physics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q, _ := k.GetQualification(ctx, "lgmvaloper1abc", "physics")
	if q.Status != string(types.StatusProbationary) {
		t.Errorf("expected probationary after renewing expired, got %s", q.Status)
	}
}

func TestRenewQualification_Revoked(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusRevoked),
	})

	err := k.RenewQualification(ctx, "lgmvaloper1abc", "physics")
	if err == nil {
		t.Fatal("expected error when renewing revoked qualification")
	}
}

func TestRenewQualification_NotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	err := k.RenewQualification(ctx, "lgmvaloper1abc", "physics")
	if err == nil {
		t.Fatal("expected error for non-existent qualification")
	}
}

func TestWithdrawQualification_Success(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr:  "lgmvaloper1abc",
		Domain:         "physics",
		Status:         string(types.StatusActive),
		LockedStake:    "22000000",
		StakeUnlocksAt: 50, // already unlocked at block 100
	})

	err := k.WithdrawQualification(ctx, "lgmvaloper1abc", "physics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, found := k.GetQualification(ctx, "lgmvaloper1abc", "physics")
	if found {
		t.Error("expected qualification to be deleted after withdrawal")
	}
}

func TestWithdrawQualification_NotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	err := k.WithdrawQualification(ctx, "lgmvaloper1abc", "physics")
	if err == nil {
		t.Fatal("expected error for non-existent qualification")
	}
}

func TestRecordChallenge_Revocation(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusActive),
		Metrics: &types.QualificationMetrics{
			ChallengesReceived: 2,
			ChallengesLost:     2, // default RevocationChallenges = 3
		},
	})

	// 3rd challenge loss should trigger revocation
	err := k.RecordChallenge(ctx, "lgmvaloper1abc", "physics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q, _ := k.GetQualification(ctx, "lgmvaloper1abc", "physics")
	if q.Status != string(types.StatusRevoked) {
		t.Errorf("expected revoked after 3 challenge losses, got %s", q.Status)
	}
}

func TestCheckAndExpireQualifications(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Active but expired
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr:  "lgmvaloper1abc",
		Domain:         "physics",
		Status:         string(types.StatusActive),
		ExpiresAtBlock: 50, // expired at 50, we're at 100
	})

	// Probationary, should promote (earned at block 0, period = 777, now at 1000)
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr:          "lgmvaloper1def",
		Domain:                 "mathematics",
		Status:                 string(types.StatusProbationary),
		EarnedAtBlock:          0,
		InheritanceDiscountBps: 10000,
		Metrics:                &types.QualificationMetrics{EffectiveWeightBps: 10000},
	})

	// Run at block 1000
	ctx = ctx.WithBlockHeight(1000)
	k.CheckAndExpireQualifications(ctx)

	q1, _ := k.GetQualification(ctx, "lgmvaloper1abc", "physics")
	if q1.Status != string(types.StatusExpired) {
		t.Errorf("expected expired, got %s", q1.Status)
	}

	q2, _ := k.GetQualification(ctx, "lgmvaloper1def", "mathematics")
	if q2.Status != string(types.StatusActive) {
		t.Errorf("expected promoted to active, got %s", q2.Status)
	}
}

// ==================== Genesis Tests ====================

func TestInitExportGenesis(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	defaultParams := types.DefaultParams()
	genState := types.GenesisState{
		Params: &defaultParams,
		Qualifications: []*types.DomainQualification{
			{
				ValidatorAddr: "lgmvaloper1abc",
				Domain:        "mathematics",
				Status:        string(types.StatusActive),
				Pathway:       string(types.PathwayGenesis),
			},
		},
		Endorsements: []*types.QualificationEndorsement{
			{
				Id:              "endorse_1",
				EndorserAddr:    "lgmvaloper1ghi",
				TargetValidator: "lgmvaloper1def",
				Domain:          "physics",
			},
		},
	}

	k.InitGenesis(ctx, genState)

	// Verify qualification was stored
	q, found := k.GetQualification(ctx, "lgmvaloper1abc", "mathematics")
	if !found {
		t.Fatal("expected genesis qualification to be stored")
	}
	if q.Pathway != string(types.PathwayGenesis) {
		t.Errorf("expected genesis pathway, got %s", q.Pathway)
	}

	// Verify endorsement was stored
	e, found := k.GetEndorsement(ctx, "endorse_1")
	if !found {
		t.Fatal("expected genesis endorsement to be stored")
	}
	if e.Domain != "physics" {
		t.Errorf("expected physics domain, got %s", e.Domain)
	}

	// Export and verify roundtrip
	exported := k.ExportGenesis(ctx)
	if len(exported.Qualifications) != 1 {
		t.Errorf("expected 1 exported qualification, got %d", len(exported.Qualifications))
	}
	if len(exported.Endorsements) != 1 {
		t.Errorf("expected 1 exported endorsement, got %d", len(exported.Endorsements))
	}
}

// ==================== Msg Server Tests ====================

func TestMsgRequestQualification_StakeCommitment(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	ms := keeper.NewMsgServerImpl(k)
	resp, err := ms.RequestQualification(ctx, &types.MsgRequestQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Pathway:       string(types.PathwayStakeCommitment),
		Stake:         "22000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != string(types.StatusProbationary) {
		t.Errorf("expected probationary, got %s", resp.Status)
	}
}

func TestMsgRequestQualification_TrackRecord(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	ms := keeper.NewMsgServerImpl(k)
	resp, err := ms.RequestQualification(ctx, &types.MsgRequestQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Pathway:       string(types.PathwayTrackRecord),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != string(types.StatusActive) {
		t.Errorf("expected active for track record, got %s", resp.Status)
	}
}

func TestMsgRequestQualification_UnsupportedPathway(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.RequestQualification(ctx, &types.MsgRequestQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Pathway:       "unknown_pathway",
	})
	if err == nil {
		t.Fatal("expected error for unsupported pathway")
	}
}

func TestMsgEndorseValidator(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Qualify endorser first
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusActive),
	})

	ms := keeper.NewMsgServerImpl(k)
	resp, err := ms.EndorseValidator(ctx, &types.MsgEndorseValidator{
		EndorserAddr:    "lgmvaloper1abc",
		TargetValidator: "lgmvaloper1def",
		Domain:          "physics",
		Stake:           "22000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.EndorsementId == "" {
		t.Error("expected non-empty endorsement ID")
	}
}

func TestMsgEndorseValidator_NotQualified(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.EndorseValidator(ctx, &types.MsgEndorseValidator{
		EndorserAddr:    "lgmvaloper1abc",
		TargetValidator: "lgmvaloper1def",
		Domain:          "physics",
	})
	if err == nil {
		t.Fatal("expected error: endorser not qualified")
	}
}

func TestMsgRenewQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr:  "lgmvaloper1abc",
		Domain:         "physics",
		Status:         string(types.StatusActive),
		ExpiresAtBlock: 500,
	})

	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.RenewQualification(ctx, &types.MsgRenewQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMsgWithdrawQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusActive),
	})

	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.WithdrawQualification(ctx, &types.MsgWithdrawQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, found := k.GetQualification(ctx, "lgmvaloper1abc", "physics")
	if found {
		t.Error("expected qualification to be deleted")
	}
}

func TestMsgUpdateParams(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	ms := keeper.NewMsgServerImpl(k)
	newParams := types.DefaultParams()
	newParams.MinVerifications = 50

	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "authority",
		Params:    &newParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := k.GetParams(ctx)
	if got.MinVerifications != 50 {
		t.Errorf("expected MinVerifications 50, got %d", got.MinVerifications)
	}
}

func TestMsgUpdateParams_Unauthorized(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	ms := keeper.NewMsgServerImpl(k)
	defaultParams := types.DefaultParams()
	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "not_authority",
		Params:    &defaultParams,
	})
	if err == nil {
		t.Fatal("expected error for unauthorized params update")
	}
}

// ==================== Query Server Tests ====================

func TestQueryQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusActive),
	})

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Qualification(ctx, &types.QueryQualificationRequest{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Qualification == nil {
		t.Fatal("expected qualification in response")
	}
	if resp.Qualification.Status != string(types.StatusActive) {
		t.Errorf("expected active, got %s", resp.Qualification.Status)
	}
}

func TestQueryQualificationsByValidator(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc", Domain: "physics", Status: string(types.StatusActive),
	})
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc", Domain: "mathematics", Status: string(types.StatusActive),
	})

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.QualificationsByValidator(ctx, &types.QueryQualificationsByValidatorRequest{
		ValidatorAddr: "lgmvaloper1abc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Qualifications) != 2 {
		t.Errorf("expected 2 qualifications, got %d", len(resp.Qualifications))
	}
}

func TestQueryQualifiedValidators(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc", Domain: "physics", Status: string(types.StatusActive),
	})
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1def", Domain: "physics", Status: string(types.StatusActive),
	})

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.QualifiedValidators(ctx, &types.QueryQualifiedValidatorsRequest{
		Domain: "physics",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Validators) != 2 {
		t.Errorf("expected 2 validators, got %d", len(resp.Validators))
	}
}

func TestQueryEndorsement(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	k.SetEndorsement(ctx, types.QualificationEndorsement{
		Id: "lgmvaloper1abc/lgmvaloper1def/physics", EndorserAddr: "lgmvaloper1abc", TargetValidator: "lgmvaloper1def",
		Domain: "physics",
	})

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Endorsement(ctx, &types.QueryEndorsementRequest{
		Endorser:        "lgmvaloper1abc",
		TargetValidator: "lgmvaloper1def",
		Domain:          "physics",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Endorsement == nil {
		t.Error("expected endorsement in response")
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Params.MinVerifications != 19 {
		t.Errorf("expected default MinVerifications 19, got %d", resp.Params.MinVerifications)
	}
}

// ==================== Stratum Stake Tier Tests ====================

func TestStakeRequirement_TopStratum(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Mathematics is stratum 0 → StakeTop (2222 LGM)
	// Providing exactly StakeTop should succeed
	q, err := k.QualifyViaStakeCommitment(ctx, "lgmvaloper1abc", "mathematics", "2222000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.LockedStake != "2222000000" {
		t.Errorf("expected locked stake 2222000000000000000000, got %s", q.LockedStake)
	}
}

func TestStakeRequirement_MidStratum(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// formal_logic is stratum 1 → StakeTop (2222 LGM) — strata 0-1 map to top
	_, err := k.QualifyViaStakeCommitment(ctx, "lgmvaloper1abc", "formal_logic", "222000000")
	if err == nil {
		t.Fatal("expected error: stratum 1 requires StakeTop, not StakeMid")
	}

	// Provide StakeTop for stratum 1
	_, err = k.QualifyViaStakeCommitment(ctx, "lgmvaloper1abc", "formal_logic", "2222000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
