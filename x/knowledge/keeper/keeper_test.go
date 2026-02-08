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
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/knowledge/keeper"
	"github.com/legible-money/legible/x/knowledge/types"
)

// ---------- Mock StakingKeeper ----------

type mockStakingKeeper struct {
	validators map[string]types.ValidatorInfo
}

func newMockStakingKeeper() *mockStakingKeeper {
	return &mockStakingKeeper{
		validators: make(map[string]types.ValidatorInfo),
	}
}

func (m *mockStakingKeeper) addValidator(addr string, stake uint64, tier uint32) {
	m.validators[addr] = types.ValidatorInfo{
		Address:    addr,
		Stake:      stake,
		Tier:       tier,
		PublicKey:  []byte("pubkey_" + addr),
		Reputation: 800000,
	}
}

func (m *mockStakingKeeper) GetActiveValidatorInfos(_ context.Context) []types.ValidatorInfo {
	infos := make([]types.ValidatorInfo, 0, len(m.validators))
	for _, v := range m.validators {
		infos = append(infos, v)
	}
	return infos
}

func (m *mockStakingKeeper) GetValidatorInfo(_ context.Context, addr string) (types.ValidatorInfo, bool) {
	v, ok := m.validators[addr]
	return v, ok
}

func (m *mockStakingKeeper) GetEffectiveStake(_ context.Context, addr string) uint64 {
	if v, ok := m.validators[addr]; ok {
		return v.Stake
	}
	return 0
}

func (m *mockStakingKeeper) GetTotalStake(_ context.Context) uint64 {
	var total uint64
	for _, v := range m.validators {
		total += v.Stake
	}
	return total
}

func (m *mockStakingKeeper) RecordVerification(_ context.Context, _ string, _ bool) error {
	return nil
}

func (m *mockStakingKeeper) SlashValidator(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

// ---------- Mock BankKeeper ----------

type mockBankKeeper struct{}

func (m mockBankKeeper) SendCoins(_ context.Context, _, _ sdk.AccAddress, _ sdk.Coins) error {
	return nil
}

func (m mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
	return nil
}

func (m mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, _ sdk.AccAddress, _ string, _ sdk.Coins) error {
	return nil
}

// ---------- Mock AccountKeeper ----------

type mockAccountKeeper struct{}

func (m mockAccountKeeper) GetAccount(_ context.Context, _ sdk.AccAddress) sdk.AccountI {
	return nil
}

func (m mockAccountKeeper) GetModuleAddress(_ string) sdk.AccAddress {
	return nil
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockStakingKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	memKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memKey, storetypes.StoreTypeMemory, nil)
	err := stateStore.LoadLatestVersion()
	if err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mockSK := newMockStakingKeeper()
	mockSK.addValidator("lgmvaloper1abc", 5_000_000, 2) // Bonded tier, 5 LGM
	mockSK.addValidator("lgmvaloper1def", 3_000_000, 2) // Bonded tier, 3 LGM
	mockSK.addValidator("lgmvaloper1ghi", 2_000_000, 1) // Verified tier, 2 LGM

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memKey,
		mockAccountKeeper{},
		mockBankKeeper{},
		mockSK,
		"lgm10d07y265gmmuvt4z0w9aw880jnsr700jefnezl", // authority
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 10}, false, log.NewNopLogger())

	// Initialize params and default domains
	k.SetParams(ctx, types.DefaultParams())
	for _, domain := range types.DefaultDomains() {
		k.SetDomainInState(ctx, domain)
	}

	return k, ctx, mockSK
}

// ---------- Tests ----------

func TestSetGetParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	params := k.GetParams(ctx)
	if params.MinVerifiers != 3 {
		t.Errorf("expected MinVerifiers=3, got %d", params.MinVerifiers)
	}
	if params.CommitPhaseDuration != 4 {
		t.Errorf("expected CommitPhaseDuration=4, got %d", params.CommitPhaseDuration)
	}
	if params.AcceptanceThreshold != 770000 {
		t.Errorf("expected AcceptanceThreshold=770000, got %d", params.AcceptanceThreshold)
	}
}

func TestDefaultDomains(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Check all 4 default domains exist
	for _, name := range []string{"mathematics", "physics", "computer_science", "general"} {
		domain, found := k.GetDomainFromState(ctx, name)
		if !found {
			t.Errorf("expected domain %q to exist", name)
			continue
		}
		if domain.Status != "active" {
			t.Errorf("expected domain %q status=active, got %s", name, domain.Status)
		}
	}

	// Non-existent domain
	_, found := k.GetDomainFromState(ctx, "nonexistent")
	if found {
		t.Error("expected nonexistent domain to not be found")
	}
}

func TestSubmitClaimCreatesRound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{
		Id:                "test-claim-1",
		ClaimText:         "Water boils at 100 degrees Celsius at sea level",
		ContentHash:       "abc123",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         "lgm1submitter",
		StakeAmount:       "1000000",
		Status:            "pending",
		SubmittedAtHeight: 10,
	}
	k.SetClaim(ctx, claim)

	// Create verification round
	round, err := k.CreateVerificationRound(ctx, claim)
	if err != nil {
		t.Fatalf("CreateVerificationRound failed: %v", err)
	}

	if round.ClaimId != claim.Id {
		t.Errorf("expected round.ClaimId=%s, got %s", claim.Id, round.ClaimId)
	}
	if round.Phase != "commit" {
		t.Errorf("expected round.Phase=commit, got %s", round.Phase)
	}
	if round.StartHeight != 10 {
		t.Errorf("expected round.StartHeight=10, got %d", round.StartHeight)
	}
	if round.CommitDeadlineHeight != 14 { // 10 + 4
		t.Errorf("expected CommitDeadlineHeight=14, got %d", round.CommitDeadlineHeight)
	}
	if round.RevealDeadlineHeight != 18 { // 10 + 4 + 4
		t.Errorf("expected RevealDeadlineHeight=18, got %d", round.RevealDeadlineHeight)
	}

	// Verify round is persisted
	stored, found := k.GetVerificationRound(ctx, round.Id)
	if !found {
		t.Fatal("expected round to be stored")
	}
	if stored.ClaimId != claim.Id {
		t.Errorf("stored round has wrong claim ID")
	}
}

func TestCommitRevealLifecycle(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create claim and round
	claim := types.Claim{
		Id:        "lifecycle-claim",
		ClaimText: "E equals mc squared in special relativity",
		Domain:    "physics",
		Status:    "pending",
	}
	k.SetClaim(ctx, claim)

	round, err := k.CreateVerificationRound(ctx, claim)
	if err != nil {
		t.Fatalf("CreateVerificationRound failed: %v", err)
	}

	// Submit commitments (commit phase)
	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		err := k.StoreCommitmentInRound(ctx, round.Id, types.VerificationCommitment{
			Validator:         v,
			CommitmentHash:    "deadbeef" + v + "deadbeefdeadbeefdeadbeefdeadbeef",
			SubmittedAtHeight: 11,
		})
		if err != nil {
			t.Fatalf("StoreCommitmentInRound failed for %s: %v", v, err)
		}
	}

	// Verify commitments stored
	commitments, _ := k.GetCommitmentsForRound(ctx, round.Id)
	if len(commitments) != 3 {
		t.Fatalf("expected 3 commitments, got %d", len(commitments))
	}

	// Transition to reveal phase
	updatedRound, _ := k.GetVerificationRound(ctx, round.Id)
	updatedRound.Phase = "reveal"
	k.SetVerificationRound(ctx, updatedRound)

	// Submit reveals
	for _, v := range validators {
		err := k.StoreRevealInRound(ctx, round.Id, types.VerificationReveal{
			Validator:         v,
			Verdict:           "accept",
			Confidence:        800000,
			SubmittedAtHeight: 15,
		})
		if err != nil {
			t.Fatalf("StoreRevealInRound failed for %s: %v", v, err)
		}
	}

	// Verify reveals stored
	reveals, _ := k.GetRevealsForRound(ctx, round.Id)
	if len(reveals) != 3 {
		t.Fatalf("expected 3 reveals, got %d", len(reveals))
	}
}

func TestDuplicateCommitmentRejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{Id: "dup-commit-claim", ClaimText: "Test claim for dedup", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// First commitment succeeds
	err := k.StoreCommitmentInRound(ctx, round.Id, types.VerificationCommitment{
		Validator:      "lgmvaloper1abc",
		CommitmentHash: "aaaa" + "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	})
	if err != nil {
		t.Fatalf("first commitment should succeed: %v", err)
	}

	// Duplicate commitment rejected
	err = k.StoreCommitmentInRound(ctx, round.Id, types.VerificationCommitment{
		Validator:      "lgmvaloper1abc",
		CommitmentHash: "cccc" + "dddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	})
	if err == nil {
		t.Error("expected duplicate commitment to be rejected")
	}
}

func TestRevealRequiresCommitment(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{Id: "no-commit-claim", ClaimText: "Claim without commitment", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// Move to reveal phase
	round.Phase = "reveal"
	k.SetVerificationRound(ctx, round)

	// Reveal without commitment should fail
	err := k.StoreRevealInRound(ctx, round.Id, types.VerificationReveal{
		Validator:  "lgmvaloper1abc",
		Verdict:    "accept",
		Confidence: 800000,
	})
	if err == nil {
		t.Error("expected reveal without commitment to fail")
	}
}

func TestConfidenceAggregation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{Id: "agg-claim", ClaimText: "Test aggregation logic", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// Add commitments and reveals (all accept with high confidence)
	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v, CommitmentHash: "hash"}
		round.Reveals[v] = types.VerificationReveal{
			Validator:  v,
			Verdict:    "accept",
			Confidence: 900000, // 90%
		}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	// Aggregate
	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	if result.Verdict != "accepted" {
		t.Errorf("expected verdict=accepted, got %s", result.Verdict)
	}
	if result.Confidence < 770000 {
		t.Errorf("expected confidence >= 770000, got %d", result.Confidence)
	}
	if result.ParticipantCount != 3 {
		t.Errorf("expected 3 participants, got %d", result.ParticipantCount)
	}
}

func TestRejectedOutcome(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{Id: "reject-claim", ClaimText: "False claim to reject", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// All validators reject
	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v, CommitmentHash: "hash"}
		round.Reveals[v] = types.VerificationReveal{
			Validator:  v,
			Verdict:    "reject",
			Confidence: 850000,
		}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	if result.Verdict != "rejected" {
		t.Errorf("expected verdict=rejected, got %s", result.Verdict)
	}
}

func TestContestedOutcome(t *testing.T) {
	k, ctx, mockSK := setupKeeper(t)
	// Add more validators for a contested scenario
	mockSK.addValidator("lgmvaloper1jkl", 2_000_000, 2)

	claim := types.Claim{Id: "contested-claim", ClaimText: "Controversial claim for debate", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// Split verdict: 2 accept, 2 reject (below 77% threshold)
	round.Commitments["lgmvaloper1abc"] = types.VerificationCommitment{Validator: "lgmvaloper1abc"}
	round.Commitments["lgmvaloper1def"] = types.VerificationCommitment{Validator: "lgmvaloper1def"}
	round.Commitments["lgmvaloper1ghi"] = types.VerificationCommitment{Validator: "lgmvaloper1ghi"}
	round.Commitments["lgmvaloper1jkl"] = types.VerificationCommitment{Validator: "lgmvaloper1jkl"}

	round.Reveals["lgmvaloper1abc"] = types.VerificationReveal{Validator: "lgmvaloper1abc", Verdict: "accept", Confidence: 800000}
	round.Reveals["lgmvaloper1def"] = types.VerificationReveal{Validator: "lgmvaloper1def", Verdict: "accept", Confidence: 800000}
	round.Reveals["lgmvaloper1ghi"] = types.VerificationReveal{Validator: "lgmvaloper1ghi", Verdict: "reject", Confidence: 800000}
	round.Reveals["lgmvaloper1jkl"] = types.VerificationReveal{Validator: "lgmvaloper1jkl", Verdict: "reject", Confidence: 800000}

	round.SelectedValidators = []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi", "lgmvaloper1jkl"}
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	// With equal stake split, agreement ratio should be ~58% (accept has more stake: 5+3=8 vs 2+2=4)
	// lgmvaloper1abc: 5M stake, accept
	// lgmvaloper1def: 3M stake, accept
	// lgmvaloper1ghi: 2M stake, reject
	// lgmvaloper1jkl: 2M stake, reject
	// Accept weight > reject weight, but may be below 77% threshold
	if result.Verdict == "accepted" && result.AgreementRatio < 770000 {
		t.Errorf("verdict should not be 'accepted' when agreement < 77%%")
	}
}

func TestFactCreationOnAcceptance(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{
		Id:                "fact-claim",
		ClaimText:         "The speed of light is approximately 299,792,458 m/s",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         "lgm1submitter",
		SubmitterDid:      "did:lgm:abc123",
		Status:            "pending",
		LogicZone:         "empirical",
	}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// All validators accept
	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 900000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	// Aggregate and complete
	result, _ := k.AggregateVerificationResult(ctx, round)
	err := k.CompleteRound(ctx, round, result)
	if err != nil {
		t.Fatalf("CompleteRound failed: %v", err)
	}

	// Verify claim status updated
	updatedClaim, found := k.GetClaim(ctx, claim.Id)
	if !found {
		t.Fatal("claim should still exist")
	}
	if updatedClaim.Status != "accepted" {
		t.Errorf("expected claim status=accepted, got %s", updatedClaim.Status)
	}

	// Verify fact was created
	factCount := k.GetFactCount(ctx)
	if factCount == 0 {
		t.Error("expected at least 1 fact after acceptance")
	}

	// Verify fact content
	var createdFact types.Fact
	k.IterateFacts(ctx, func(f types.Fact) bool {
		if f.ClaimText == claim.ClaimText {
			createdFact = f
			return true
		}
		return false
	})

	if createdFact.Id == "" {
		t.Fatal("expected to find created fact")
	}
	if createdFact.Domain != "physics" {
		t.Errorf("expected fact domain=physics, got %s", createdFact.Domain)
	}
	if createdFact.Status != "verified" {
		t.Errorf("expected fact status=verified, got %s", createdFact.Status)
	}
	if createdFact.VerifierCount != 3 {
		t.Errorf("expected 3 verifiers, got %d", createdFact.VerifierCount)
	}
}

func TestInsufficientParticipation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{Id: "insuff-claim", ClaimText: "Insufficient participation test", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// Only 1 reveal out of 3 selected validators (below 66% quorum)
	round.SelectedValidators = []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	round.Commitments["lgmvaloper1abc"] = types.VerificationCommitment{Validator: "lgmvaloper1abc"}
	round.Reveals["lgmvaloper1abc"] = types.VerificationReveal{
		Validator:  "lgmvaloper1abc",
		Verdict:    "accept",
		Confidence: 900000,
	}
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	// 1/3 = 33% participation, below 66% quorum
	if result.Verdict != "insufficient" {
		t.Errorf("expected verdict=insufficient, got %s", result.Verdict)
	}
}

func TestPhaseTransitions(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{Id: "phase-claim", ClaimText: "Phase transition test claim text", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	params := k.GetParams(ctx)

	// At start: commit phase
	phase := keeper.GetRoundPhase(round, 10, params)
	if phase != "commit" {
		t.Errorf("at height 10: expected commit, got %s", phase)
	}

	// At commit deadline: still commit
	phase = keeper.GetRoundPhase(round, round.CommitDeadlineHeight, params)
	if phase != "commit" {
		t.Errorf("at commit deadline: expected commit, got %s", phase)
	}

	// After commit deadline: reveal
	phase = keeper.GetRoundPhase(round, round.CommitDeadlineHeight+1, params)
	if phase != "reveal" {
		t.Errorf("after commit deadline: expected reveal, got %s", phase)
	}

	// At reveal deadline: still reveal
	phase = keeper.GetRoundPhase(round, round.RevealDeadlineHeight, params)
	if phase != "reveal" {
		t.Errorf("at reveal deadline: expected reveal, got %s", phase)
	}

	// After reveal deadline: aggregation
	phase = keeper.GetRoundPhase(round, round.RevealDeadlineHeight+1, params)
	if phase != "aggregation" {
		t.Errorf("after reveal deadline: expected aggregation, got %s", phase)
	}

	// After aggregation deadline: expired
	aggDeadline := round.RevealDeadlineHeight + params.AggregationPhaseDuration
	phase = keeper.GetRoundPhase(round, aggDeadline+1, params)
	if phase != "expired" {
		t.Errorf("after aggregation deadline: expected expired, got %s", phase)
	}
}

func TestQueryFact(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create a fact directly
	fact := types.Fact{
		Id:                "query-fact-1",
		ClaimText:         "Gravity is approximately 9.81 m/s^2 on Earth",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Confidence:        850000,
		Status:            "verified",
		Submitter:         "lgm1submitter",
		VerifierCount:     3,
	}
	k.SetFact(ctx, fact)

	// Query via query server
	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Fact(ctx, &types.QueryFactRequest{FactId: "query-fact-1"})
	if err != nil {
		t.Fatalf("query Fact failed: %v", err)
	}
	if resp.Fact.ClaimText != fact.ClaimText {
		t.Errorf("expected claim text match")
	}
	if resp.Fact.Confidence != 850000 {
		t.Errorf("expected confidence=850000, got %d", resp.Fact.Confidence)
	}

	// Query non-existent fact
	_, err = qs.Fact(ctx, &types.QueryFactRequest{FactId: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent fact")
	}
}

func TestQueryActiveRounds(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create 2 active rounds + 1 complete
	for i, status := range []string{"commit", "reveal", "complete"} {
		claim := types.Claim{
			Id:        "active-claim-" + string(rune('0'+i)),
			ClaimText: "Active round test " + string(rune('0'+i)),
			Domain:    "general",
			Status:    "pending",
		}
		k.SetClaim(ctx, claim)
		round, _ := k.CreateVerificationRound(ctx, claim)
		round.Phase = status
		k.SetVerificationRound(ctx, round)
	}

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.ActiveRounds(ctx, &types.QueryActiveRoundsRequest{})
	if err != nil {
		t.Fatalf("query ActiveRounds failed: %v", err)
	}

	if len(resp.Rounds) != 2 {
		t.Errorf("expected 2 active rounds, got %d", len(resp.Rounds))
	}
}

func TestSlashOnMissedReveal(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{Id: "slash-claim", ClaimText: "Slash test for missed reveal", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// 3 validators commit, but only 2 reveal
	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v, CommitmentHash: "hash"}
	}
	// Only first two reveal
	round.Reveals["lgmvaloper1abc"] = types.VerificationReveal{Validator: "lgmvaloper1abc", Verdict: "accept", Confidence: 900000}
	round.Reveals["lgmvaloper1def"] = types.VerificationReveal{Validator: "lgmvaloper1def", Verdict: "accept", Confidence: 900000}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	// Aggregate — lgmvaloper1ghi committed but didn't reveal
	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	// Check that lgmvaloper1ghi got slashed for missed reveal
	if _, slashed := result.Slashes["lgmvaloper1ghi"]; !slashed {
		t.Error("expected lgmvaloper1ghi to be slashed for missed reveal")
	}
}

func TestCleanupExpiredRounds(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create a completed round at height 5
	claim := types.Claim{Id: "cleanup-claim", ClaimText: "Cleanup test completed round", Domain: "general", Status: "accepted"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)
	round.Phase = "complete"
	round.CompletedAtHeight = 5
	k.SetVerificationRound(ctx, round)

	// At height 10 with retention=100, round should NOT be cleaned
	k.CleanupExpiredRounds(ctx, 100)
	_, found := k.GetVerificationRound(ctx, round.Id)
	if !found {
		t.Error("round should not be cleaned up yet (within retention)")
	}

	// Move to height 200 to trigger cleanup (5 + 100 = 105 < 200)
	ctx = ctx.WithBlockHeight(200)
	k.CleanupExpiredRounds(ctx, 100)
	_, found = k.GetVerificationRound(ctx, round.Id)
	if found {
		t.Error("round should have been cleaned up")
	}
}
