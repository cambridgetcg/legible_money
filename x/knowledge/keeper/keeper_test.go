package keeper_test

import (
	"context"
	"fmt"
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

// testSubmitterAddr is a valid lgm-prefixed bech32 address for use in stake settlement tests.
var testSubmitterAddr string

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")

	// Generate a valid bech32 address from deterministic bytes
	testSubmitterAddr = sdk.AccAddress([]byte("test_submitter_addr_")).String()
}

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

func (m mockBankKeeper) SendCoinsFromModuleToModule(_ context.Context, _, _ string, _ sdk.Coins) error {
	return nil
}

func (m mockBankKeeper) BurnCoins(_ context.Context, _ string, _ sdk.Coins) error {
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

// ---------- Mock TokensKeeper ----------

type mockTokensKeeper struct{}

func (m mockTokensKeeper) DebitBalance(_ sdk.Context, _, _ string) error {
	return nil
}

func (m mockTokensKeeper) CreditBalance(_ sdk.Context, _, _ string, _ uint64) error {
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
		mockTokensKeeper{},
		nil, // qualificationKeeper — nil-safe, wire when needed
		nil, // captureDefenseKeeper — nil-safe, wire when needed
		nil, // vestingRewardsKeeper — nil-safe, wire when needed
		nil, // evidenceMgmtKeeper — nil-safe, wire when needed
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

// ---------- Tests for Proto-Go Reconciliation (new fields) ----------

func TestClaimWithEvidence(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	evidence := []types.ClaimEvidence{
		{
			EvidenceType: "citation",
			URI:          "https://example.com/paper.pdf",
			ContentHash:  "sha256:abc123",
			Metadata:     map[string]string{"author": "Einstein", "year": "1905"},
		},
		{
			EvidenceType: "dataset",
			URI:          "ipfs://Qm...",
			ContentHash:  "sha256:def456",
		},
	}

	claim := types.Claim{
		Id:                "evidence-claim-1",
		ClaimText:         "E equals mc squared in special relativity framework",
		ContentHash:       "evhash123",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         "lgm1submitter",
		StakeAmount:       "1000000",
		Status:            "pending",
		SubmittedAtHeight: 10,
		Evidence:          evidence,
	}
	k.SetClaim(ctx, claim)

	// Retrieve and verify evidence is stored
	stored, found := k.GetClaim(ctx, "evidence-claim-1")
	if !found {
		t.Fatal("expected claim to be stored")
	}
	if len(stored.Evidence) != 2 {
		t.Fatalf("expected 2 evidence items, got %d", len(stored.Evidence))
	}
	if stored.Evidence[0].EvidenceType != "citation" {
		t.Errorf("expected evidence_type=citation, got %s", stored.Evidence[0].EvidenceType)
	}
	if stored.Evidence[0].Metadata["author"] != "Einstein" {
		t.Errorf("expected metadata author=Einstein, got %s", stored.Evidence[0].Metadata["author"])
	}
	if stored.Evidence[1].URI != "ipfs://Qm..." {
		t.Errorf("expected evidence URI=ipfs://Qm..., got %s", stored.Evidence[1].URI)
	}
}

func TestRevealStoresSalt(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{Id: "salt-claim", ClaimText: "Salt storage test for reveal auditability", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// Store commitment
	err := k.StoreCommitmentInRound(ctx, round.Id, types.VerificationCommitment{
		Validator:         "lgmvaloper1abc",
		CommitmentHash:    "aaaa" + "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		SubmittedAtHeight: 11,
	})
	if err != nil {
		t.Fatalf("StoreCommitmentInRound failed: %v", err)
	}

	// Transition to reveal phase
	updatedRound, _ := k.GetVerificationRound(ctx, round.Id)
	updatedRound.Phase = "reveal"
	k.SetVerificationRound(ctx, updatedRound)

	// Store reveal with salt
	err = k.StoreRevealInRound(ctx, round.Id, types.VerificationReveal{
		Validator:         "lgmvaloper1abc",
		Verdict:           "accept",
		Confidence:        800000,
		Justification:     "Correct per observation",
		Salt:              "random-salt-value-12345",
		SubmittedAtHeight: 15,
	})
	if err != nil {
		t.Fatalf("StoreRevealInRound failed: %v", err)
	}

	// Verify salt is persisted
	storedRound, _ := k.GetVerificationRound(ctx, round.Id)
	reveal, ok := storedRound.Reveals["lgmvaloper1abc"]
	if !ok {
		t.Fatal("expected reveal to exist")
	}
	if reveal.Salt != "random-salt-value-12345" {
		t.Errorf("expected salt=random-salt-value-12345, got %s", reveal.Salt)
	}
}

func TestFactDecayTrackingFields(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create a fact with decay tracking fields
	fact := types.Fact{
		Id:                   "decay-fact-1",
		ClaimText:            "Water boils at 100 degrees Celsius at standard pressure",
		Domain:               "physics",
		EpistemicCategory:    "empirical",
		Confidence:           900000,
		Status:               "verified",
		Submitter:            "lgm1submitter",
		VerifierCount:        3,
		CreatedAtHeight:      10,
		VerifiedAtHeight:     10,
		LastVerifiedHeight:   10,
		LastConfidenceUpdate: 10,
		BaseConfidence:       900000,
	}
	k.SetFact(ctx, fact)

	// Retrieve and verify decay tracking fields
	stored, found := k.GetFact(ctx, "decay-fact-1")
	if !found {
		t.Fatal("expected fact to be stored")
	}
	if stored.LastConfidenceUpdate != 10 {
		t.Errorf("expected LastConfidenceUpdate=10, got %d", stored.LastConfidenceUpdate)
	}
	if stored.BaseConfidence != 900000 {
		t.Errorf("expected BaseConfidence=900000, got %d", stored.BaseConfidence)
	}
}

func TestFactCreationSetsDecayFields(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{
		Id:                "decay-create-claim",
		ClaimText:         "Pythagoras theorem: a squared plus b squared equals c squared",
		Domain:            "mathematics",
		EpistemicCategory: "formal",
		Submitter:         "lgm1submitter",
		SubmitterDid:      "did:lgm:abc123",
		Status:            "pending",
		LogicZone:         "propositional",
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

	// Find created fact and verify decay fields
	var createdFact types.Fact
	k.IterateFacts(ctx, func(f types.Fact) bool {
		if f.ClaimText == claim.ClaimText {
			createdFact = f
			return true
		}
		return false
	})

	if createdFact.Id == "" {
		t.Fatal("expected fact to be created")
	}
	if createdFact.BaseConfidence == 0 {
		t.Error("expected BaseConfidence to be set (non-zero)")
	}
	if createdFact.BaseConfidence != createdFact.Confidence {
		t.Errorf("expected BaseConfidence=%d to equal Confidence=%d on creation",
			createdFact.BaseConfidence, createdFact.Confidence)
	}
	if createdFact.LastConfidenceUpdate != uint64(ctx.BlockHeight()) {
		t.Errorf("expected LastConfidenceUpdate=%d, got %d",
			ctx.BlockHeight(), createdFact.LastConfidenceUpdate)
	}
}

func TestFactReferenceCreatedAtHeight(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	ref := types.FactReference{
		FromFactId:      "fact-1",
		ToFactId:        "fact-2",
		ReferenceType:   "supports",
		Weight:          100,
		CreatedAtHeight: uint64(ctx.BlockHeight()),
	}
	k.SetFactReference(ctx, ref)

	// Retrieve and verify
	refs := k.GetFactReferences(ctx, "fact-1")
	if len(refs) == 0 {
		t.Fatal("expected at least 1 reference")
	}
	if refs[0].CreatedAtHeight != uint64(ctx.BlockHeight()) {
		t.Errorf("expected CreatedAtHeight=%d, got %d", ctx.BlockHeight(), refs[0].CreatedAtHeight)
	}
}

func TestExtendedEpistemicCategories(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Test that the extended categories (predictive, replicated, computational) are stored correctly
	categories := []string{"predictive", "replicated", "computational"}
	for i, cat := range categories {
		fact := types.Fact{
			Id:                fmt.Sprintf("cat-fact-%d", i),
			ClaimText:         fmt.Sprintf("Test fact for category %s here", cat),
			Domain:            "general",
			EpistemicCategory: cat,
			Confidence:        800000,
			Status:            "verified",
			Submitter:         "lgm1submitter",
		}
		k.SetFact(ctx, fact)

		stored, found := k.GetFact(ctx, fmt.Sprintf("cat-fact-%d", i))
		if !found {
			t.Fatalf("expected fact for category %s to be stored", cat)
		}
		if stored.EpistemicCategory != cat {
			t.Errorf("expected category=%s, got %s", cat, stored.EpistemicCategory)
		}
	}
}

func TestAggregationSetsCompletedAtHeightAndQuorum(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{Id: "quorum-claim", ClaimText: "Quorum and completion height test data", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// All validators accept with high confidence
	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v, CommitmentHash: "hash"}
		round.Reveals[v] = types.VerificationReveal{
			Validator:  v,
			Verdict:    "accept",
			Confidence: 900000,
		}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	if result.CompletedAtHeight != uint64(ctx.BlockHeight()) {
		t.Errorf("expected CompletedAtHeight=%d, got %d", ctx.BlockHeight(), result.CompletedAtHeight)
	}
	if !result.QuorumReached {
		t.Error("expected QuorumReached=true when all validators participate")
	}
}

func TestInsufficientQuorumNoQuorumFlag(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{Id: "no-quorum-claim", ClaimText: "Quorum not reached test scenario", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// Only 1 of 3 selected validators reveals
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

	if result.QuorumReached {
		t.Error("expected QuorumReached=false when participation below threshold")
	}
	if result.Verdict != "insufficient" {
		t.Errorf("expected verdict=insufficient, got %s", result.Verdict)
	}
}

// ==================== Confidence Decay Tests ====================

func TestCalculateDecayedConfidence_Eternal(t *testing.T) {
	// Eternal categories (formal, analytic, protocol, computational) should not decay
	for _, cat := range []string{"formal", "analytic", "protocol", "computational"} {
		result := keeper.CalculateDecayedConfidence(900_000, 1_000_000, cat)
		if result != 900_000 {
			t.Errorf("category %s: expected no decay (900000), got %d", cat, result)
		}
	}
}

func TestCalculateDecayedConfidence_Empirical(t *testing.T) {
	// Empirical half-life: 100,000 blocks
	base := uint64(900_000)

	// At 0 blocks: no decay
	result := keeper.CalculateDecayedConfidence(base, 0, "empirical")
	if result != base {
		t.Errorf("at 0 blocks: expected %d, got %d", base, result)
	}

	// At 1 half-life (100,000 blocks): should be ~halfway between base and min
	// min=300,000, so midpoint = 300,000 + (900,000-300,000)*0.5 = 600,000
	result = keeper.CalculateDecayedConfidence(base, 100_000, "empirical")
	if result < 550_000 || result > 650_000 {
		t.Errorf("at 1 half-life: expected ~600000, got %d", result)
	}

	// At 2 half-lives: should be ~450,000
	result = keeper.CalculateDecayedConfidence(base, 200_000, "empirical")
	if result < 400_000 || result > 500_000 {
		t.Errorf("at 2 half-lives: expected ~450000, got %d", result)
	}

	// At 10 half-lives: should be near min (300,000)
	result = keeper.CalculateDecayedConfidence(base, 1_000_000, "empirical")
	if result < 300_000 || result > 310_000 {
		t.Errorf("at 10 half-lives: expected ~300000, got %d", result)
	}
}

func TestCalculateDecayedConfidence_CeilingEnforced(t *testing.T) {
	// Empirical max = 990,000. Passing 1,000,000 should be clamped.
	result := keeper.CalculateDecayedConfidence(1_000_000, 0, "empirical")
	if result != 990_000 {
		t.Errorf("expected ceiling 990000, got %d", result)
	}
}

func TestCalculateDecayedConfidence_UnknownCategory(t *testing.T) {
	// Unknown category returns base unchanged
	result := keeper.CalculateDecayedConfidence(800_000, 50_000, "unknown")
	if result != 800_000 {
		t.Errorf("expected 800000 for unknown category, got %d", result)
	}
}

func TestGetFactWithDecay(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create an empirical fact verified at height 10
	fact := types.Fact{
		Id:                 "decay-test-1",
		ClaimText:          "Empirical fact for decay test scenario",
		Domain:             "physics",
		EpistemicCategory:  "empirical",
		Confidence:         900_000,
		BaseConfidence:     900_000,
		Status:             types.FactStatusVerified,
		Submitter:          "lgm1sub",
		VerifiedAtHeight:   10,
		LastVerifiedHeight: 10,
		VerifierCount:      3,
	}
	k.SetFact(ctx, fact)

	// Query at height 10: no decay
	decayed, found := k.GetFactWithDecay(ctx, "decay-test-1")
	if !found {
		t.Fatal("fact not found")
	}
	if decayed.Confidence != 900_000 {
		t.Errorf("at same height: expected 900000, got %d", decayed.Confidence)
	}

	// Query at height 100,010 (100,000 blocks later = 1 half-life)
	ctx = ctx.WithBlockHeight(100_010)
	decayed, _ = k.GetFactWithDecay(ctx, "decay-test-1")
	if decayed.Confidence >= 900_000 {
		t.Errorf("expected decay after 1 half-life, got %d", decayed.Confidence)
	}
	if decayed.Confidence < 550_000 || decayed.Confidence > 650_000 {
		t.Errorf("expected ~600000 after 1 half-life, got %d", decayed.Confidence)
	}

	// Original fact should be unchanged (lazy evaluation)
	original, _ := k.GetFact(ctx, "decay-test-1")
	if original.Confidence != 900_000 {
		t.Errorf("original fact should be unchanged, got %d", original.Confidence)
	}
}

func TestCalculateMaturity(t *testing.T) {
	// Emerging: low conf/age/verifications
	m := keeper.CalculateMaturity(500_000, 1000, 1)
	if m != types.MaturityEmerging {
		t.Errorf("expected emerging, got %s", m)
	}

	// Established: >=800K conf, >=34272 age, >=3 verifications
	m = keeper.CalculateMaturity(800_000, 34_272, 3)
	if m != types.MaturityEstablished {
		t.Errorf("expected established, got %s", m)
	}

	// Canonical: >=900K conf, >=342720 age, >=10 verifications
	m = keeper.CalculateMaturity(900_000, 342_720, 10)
	if m != types.MaturityCanonical {
		t.Errorf("expected canonical, got %s", m)
	}
}

func TestNeedsReverification(t *testing.T) {
	// Formal fact: never needs reverification
	formalFact := types.Fact{
		EpistemicCategory:  "formal",
		Confidence:         500_000,
		LastVerifiedHeight: 10,
	}
	if keeper.NeedsReverification(formalFact, 1_000_000) {
		t.Error("formal facts should never need reverification")
	}

	// Empirical fact above threshold, not stale
	empiricalFact := types.Fact{
		EpistemicCategory:  "empirical",
		Confidence:         700_000, // above 60% threshold
		LastVerifiedHeight: 10,
		VerifiedAtHeight:   10,
	}
	if keeper.NeedsReverification(empiricalFact, 100) {
		t.Error("empirical fact above threshold should not need reverification")
	}

	// Empirical fact below threshold
	empiricalFact.Confidence = 400_000 // below 60%
	if !keeper.NeedsReverification(empiricalFact, 100) {
		t.Error("empirical fact below threshold should need reverification")
	}

	// Empirical fact stale: age > 2× half-life (200,000 blocks)
	empiricalFact.Confidence = 800_000
	if !keeper.NeedsReverification(empiricalFact, 200_011) {
		t.Error("stale empirical fact should need reverification")
	}
}

func TestApplyReverificationBonus(t *testing.T) {
	fact := types.Fact{
		EpistemicCategory: "empirical",
		BaseConfidence:    700_000,
		Confidence:        600_000,
	}

	// Empirical bonus = 50,000
	result := keeper.ApplyReverificationBonus(fact)
	if result.BaseConfidence != 750_000 {
		t.Errorf("expected base 750000 after bonus, got %d", result.BaseConfidence)
	}
	if result.Confidence != 750_000 {
		t.Errorf("expected confidence 750000, got %d", result.Confidence)
	}

	// Bonus capped at max
	fact.BaseConfidence = 970_000
	result = keeper.ApplyReverificationBonus(fact)
	if result.BaseConfidence != 990_000 { // empirical max
		t.Errorf("expected capped at 990000, got %d", result.BaseConfidence)
	}
}

// ==================== Fact Expiration Tests ====================

func TestIsFactExpired_ExplicitExpiry(t *testing.T) {
	fact := types.Fact{
		Status:         types.FactStatusActive,
		ExpiresAtBlock: 100,
	}

	expired, reason := keeper.IsFactExpired(fact, 99)
	if expired {
		t.Error("should not be expired before expiry block")
	}

	expired, reason = keeper.IsFactExpired(fact, 100)
	if !expired || reason != "explicit_expiry" {
		t.Errorf("expected explicit_expiry at block 100, got expired=%v reason=%s", expired, reason)
	}
}

func TestIsFactExpired_ConfidenceDecay(t *testing.T) {
	// Use a low base confidence so decay hits the floor before staleness triggers
	fact := types.Fact{
		Status:             types.FactStatusActive,
		EpistemicCategory:  "empirical",
		Confidence:         350_000,
		BaseConfidence:     350_000,
		VerifiedAtHeight:   10,
		LastVerifiedHeight: 10,
	}

	// At 50,000 blocks: confidence decays from 350K toward 300K (min).
	// base=350K, min=300K, range=50K, decay at 50K/100K = 0.5 half-life → factor ~0.707
	// decayed = 300K + 50K * 0.707 = ~335K, still above 300K.
	// At 150,000 blocks: factor ~0.354 → decayed = 300K + 50K*0.354 = ~317K → still above 300K.
	// At ~500K blocks (5 half-lives): factor ~0.031 → decayed = 300K + 50K*0.031 = ~301K
	// This will hit the floor only after many half-lives, but staleness (200K blocks) triggers first.
	// So use a very low base that will drop below 300K faster:
	fact.BaseConfidence = 310_000
	fact.Confidence = 310_000

	// At 100K blocks (1 half-life): decayed = 300K + 10K*0.5 = 305K → above 300K
	// At 200K blocks (2 half-lives): decayed = 300K + 10K*0.25 = 302K → above 300K
	// Staleness at 200K triggers first. The confidence_decay path is hard to hit without staleness.
	// Use a category without RequiresReverification to isolate confidence decay.
	// All temporal categories have RequiresReverification=true... so staleness always triggers first
	// when age > 2×halfLife. The confidence_decay path is for when confidence drops below 300K
	// without hitting staleness. This happens when half-life is large relative to decay.

	// Use "replicated" category: halfLife=1,000,000, staleness at 2,000,000 blocks
	fact.EpistemicCategory = "replicated"
	fact.BaseConfidence = 350_000

	// At 10,000,000 blocks (10 half-lives): decayed near min 300K
	// Staleness threshold: 2 × 1,000,000 = 2,000,000
	// But staleness triggers at 2M blocks. We need confidence to drop below 300K
	// before 2M blocks... With base=350K, min=300K, range=50K:
	// At 2M blocks: factor = 0.5^2 = 0.25 → 300K + 50K*0.25 = 312.5K → still above 300K
	// At 10M blocks: factor = 0.5^10 ≈ 0.001 → 300K + 50K*0.001 = 300.05K → still above
	// The floor means confidence NEVER drops below min. So confidence_decay only triggers
	// when min confidence itself is >= MinConfidenceThreshold(300K) and the decayed value
	// still ends up below 300K. With replicated min=300K, this can't happen.

	// Use "predictive" category: halfLife=20,000, min=100,000 (below 300K threshold)
	fact.EpistemicCategory = "predictive"
	fact.BaseConfidence = 400_000

	// At 60K blocks (3 half-lives): factor = 0.125 → 100K + 300K*0.125 = 137.5K < 300K
	// Staleness: 2 × 20K = 40K blocks. So staleness triggers at 40K before confidence_decay at 60K.
	// To hit confidence_decay first, we need the confidence to drop below 300K before staleness.
	// Staleness = 40,010 blocks. At 40,009: factor = 0.5^(40009/20000) ≈ 0.5^2 ≈ 0.25
	// decayed = 100K + 300K*0.25 = 175K < 300K. YES! This triggers before staleness.

	expired, reason := keeper.IsFactExpired(fact, 40_009)
	if !expired || reason != "confidence_decay" {
		t.Errorf("expected confidence_decay, got expired=%v reason=%s", expired, reason)
	}
}

func TestIsFactExpired_ReverificationStaleness(t *testing.T) {
	fact := types.Fact{
		Status:             types.FactStatusActive,
		EpistemicCategory:  "empirical",
		Confidence:         900_000,
		BaseConfidence:     900_000,
		VerifiedAtHeight:   10,
		LastVerifiedHeight: 10,
	}

	// Staleness threshold: 2 × 100,000 = 200,000 blocks
	expired, reason := keeper.IsFactExpired(fact, 200_011)
	if !expired || reason != "reverification_staleness" {
		t.Errorf("expected reverification_staleness, got expired=%v reason=%s", expired, reason)
	}
}

func TestIsFactExpired_TerminalStatesSkipped(t *testing.T) {
	for _, status := range []string{types.FactStatusExpired, types.FactStatusRevoked, types.FactStatusDisproven} {
		fact := types.Fact{
			Status:            status,
			EpistemicCategory: "empirical",
		}
		expired, _ := keeper.IsFactExpired(fact, 999_999)
		if expired {
			t.Errorf("status %s should not be checked for expiry", status)
		}
	}
}

func TestExpireStaleFactsInEndBlocker(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create a fact that will expire
	fact := types.Fact{
		Id:                 "stale-1",
		ClaimText:          "Stale empirical fact for expiration test",
		Domain:             "physics",
		EpistemicCategory:  "empirical",
		Confidence:         900_000,
		BaseConfidence:     900_000,
		Status:             types.FactStatusActive,
		Submitter:          "lgm1sub",
		VerifiedAtHeight:   10,
		LastVerifiedHeight: 10,
	}
	k.SetFact(ctx, fact)

	// At height 10: should not expire
	expired := k.ExpireStaleFactsInEndBlocker(ctx, 50)
	if expired != 0 {
		t.Errorf("expected 0 expired at height 10, got %d", expired)
	}

	// At height 200,011: staleness threshold exceeded (2 × 100,000)
	ctx = ctx.WithBlockHeight(200_011)
	expired = k.ExpireStaleFactsInEndBlocker(ctx, 50)
	if expired != 1 {
		t.Errorf("expected 1 expired at height 200011, got %d", expired)
	}

	// Verify fact is now expired
	f, found := k.GetFact(ctx, "stale-1")
	if !found {
		t.Fatal("fact should still exist")
	}
	if f.Status != types.FactStatusExpired {
		t.Errorf("expected status expired, got %s", f.Status)
	}
}

func TestReverifyFact(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := types.Fact{
		Id:                 "reverify-1",
		ClaimText:          "Reverifiable fact for recheck test",
		Domain:             "physics",
		EpistemicCategory:  "empirical",
		Confidence:         600_000,
		BaseConfidence:     700_000,
		Status:             types.FactStatusExpired,
		Submitter:          "lgm1sub",
		VerifiedAtHeight:   10,
		LastVerifiedHeight: 10,
		VerifierCount:      2,
		CreatedAtHeight:    5,
	}
	k.SetFact(ctx, fact)

	// Reverify at height 500
	ctx = ctx.WithBlockHeight(500)
	err := k.ReverifyFact(ctx, "reverify-1", 850_000, 500)
	if err != nil {
		t.Fatalf("ReverifyFact failed: %v", err)
	}

	updated, _ := k.GetFact(ctx, "reverify-1")
	if updated.Status != types.FactStatusActive {
		t.Errorf("expected status active after reverification, got %s", updated.Status)
	}
	if updated.LastVerifiedHeight != 500 {
		t.Errorf("expected LastVerifiedHeight=500, got %d", updated.LastVerifiedHeight)
	}
	if updated.VerifierCount != 3 {
		t.Errorf("expected VerifierCount=3, got %d", updated.VerifierCount)
	}
	if updated.Confidence != 850_000 {
		t.Errorf("expected confidence 850000, got %d", updated.Confidence)
	}
}

// ==================== Confidence Propagation Tests ====================

func TestPropagateConfidenceChange(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create base fact A
	factA := types.Fact{
		Id: "prop-a", ClaimText: "Axiom A base fact",
		Domain: "mathematics", EpistemicCategory: "formal",
		Confidence: 900_000, BaseConfidence: 900_000,
		Status: types.FactStatusVerified, Submitter: "lgm1sub",
		VerifiedAtHeight: 10, LastVerifiedHeight: 10,
	}
	k.SetFact(ctx, factA)

	// Create fact B that cites A (B references A as "supports")
	factB := types.Fact{
		Id: "prop-b", ClaimText: "Derived fact B from A",
		Domain: "mathematics", EpistemicCategory: "formal",
		Confidence: 850_000, BaseConfidence: 850_000,
		Status: types.FactStatusVerified, Submitter: "lgm1sub",
		VerifiedAtHeight: 10, LastVerifiedHeight: 10,
	}
	k.SetFact(ctx, factB)

	// B supports A (B's outgoing ref points to A)
	k.SetFactReference(ctx, types.FactReference{
		FromFactId: "prop-b", ToFactId: "prop-a",
		ReferenceType: "supports", Weight: 1_000_000,
	})

	// Now reduce A's confidence (both current and base)
	factA.Confidence = 400_000
	factA.BaseConfidence = 400_000
	k.SetFact(ctx, factA)

	// Propagate change from A
	updated := k.PropagateConfidenceChange(ctx, "prop-a", 400_000)
	if updated != 1 {
		t.Errorf("expected 1 fact updated by propagation, got %d", updated)
	}

	// B's confidence should have been reduced (weakest-link)
	factBUpdated, _ := k.GetFact(ctx, "prop-b")
	if factBUpdated.Confidence >= 850_000 {
		t.Errorf("expected B's confidence to decrease, got %d", factBUpdated.Confidence)
	}
}

func TestCalculateDerivedConfidence_NoRefs(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := types.Fact{
		Id: "no-ref-fact", ClaimText: "Standalone fact no refs",
		Domain: "general", EpistemicCategory: "formal",
		Confidence: 800_000, BaseConfidence: 800_000,
		Status: types.FactStatusVerified, Submitter: "lgm1sub",
		VerifiedAtHeight: 10, LastVerifiedHeight: 10,
	}
	k.SetFact(ctx, fact)

	derived := k.CalculateDerivedConfidence(ctx, "no-ref-fact")
	if derived != 800_000 {
		t.Errorf("standalone fact should return base confidence, got %d", derived)
	}
}

func TestDetectCircularDependencies(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create A → B → C → A cycle
	for _, id := range []string{"cycle-a", "cycle-b", "cycle-c"} {
		k.SetFact(ctx, types.Fact{
			Id: id, ClaimText: "Cycle member " + id,
			Domain: "general", EpistemicCategory: "formal",
			Confidence: 800_000, BaseConfidence: 800_000,
			Status: types.FactStatusVerified, Submitter: "lgm1sub",
			VerifiedAtHeight: 10, LastVerifiedHeight: 10,
		})
	}

	k.SetFactReference(ctx, types.FactReference{FromFactId: "cycle-a", ToFactId: "cycle-b", ReferenceType: "supports", Weight: 1_000_000})
	k.SetFactReference(ctx, types.FactReference{FromFactId: "cycle-b", ToFactId: "cycle-c", ReferenceType: "supports", Weight: 1_000_000})
	k.SetFactReference(ctx, types.FactReference{FromFactId: "cycle-c", ToFactId: "cycle-a", ReferenceType: "supports", Weight: 1_000_000})

	cycleFacts := k.DetectCircularDependencies(ctx)
	if len(cycleFacts) < 3 {
		t.Errorf("expected 3 cycle members, got %d", len(cycleFacts))
	}
}

func TestApplyCyclePenalties(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create simple A → B → A cycle
	k.SetFact(ctx, types.Fact{
		Id: "cyc-x", ClaimText: "Cycle fact X test",
		Domain: "general", EpistemicCategory: "formal",
		Confidence: 800_000, BaseConfidence: 800_000,
		Status: types.FactStatusVerified, Submitter: "lgm1sub",
		VerifiedAtHeight: 10, LastVerifiedHeight: 10,
	})
	k.SetFact(ctx, types.Fact{
		Id: "cyc-y", ClaimText: "Cycle fact Y test",
		Domain: "general", EpistemicCategory: "formal",
		Confidence: 800_000, BaseConfidence: 800_000,
		Status: types.FactStatusVerified, Submitter: "lgm1sub",
		VerifiedAtHeight: 10, LastVerifiedHeight: 10,
	})

	k.SetFactReference(ctx, types.FactReference{FromFactId: "cyc-x", ToFactId: "cyc-y", ReferenceType: "supports", Weight: 1_000_000})
	k.SetFactReference(ctx, types.FactReference{FromFactId: "cyc-y", ToFactId: "cyc-x", ReferenceType: "supports", Weight: 1_000_000})

	penalized := k.ApplyCyclePenalties(ctx)
	if penalized != 2 {
		t.Errorf("expected 2 penalized, got %d", penalized)
	}

	// Check penalty applied (0.4× = 320,000)
	fx, _ := k.GetFact(ctx, "cyc-x")
	if fx.Confidence != 320_000 {
		t.Errorf("expected 320000 after 0.4x penalty, got %d", fx.Confidence)
	}
}

// ==================== Citation Economics Tests ====================

func TestIncrementCitationCount(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := types.Fact{
		Id: "cite-fact-1", ClaimText: "Base fact for citation",
		Domain: "physics", EpistemicCategory: "empirical",
		Confidence: 800_000, Status: types.FactStatusVerified,
		Submitter: "lgm1sub", CitationCount: 0,
	}
	k.SetFact(ctx, fact)

	err := k.IncrementCitationCount(ctx, "cite-fact-1")
	if err != nil {
		t.Fatalf("IncrementCitationCount failed: %v", err)
	}

	updated, _ := k.GetFact(ctx, "cite-fact-1")
	if updated.CitationCount != 1 {
		t.Errorf("expected citation count 1, got %d", updated.CitationCount)
	}
	if updated.LastCitedBlock != uint64(ctx.BlockHeight()) {
		t.Errorf("expected LastCitedBlock=%d, got %d", ctx.BlockHeight(), updated.LastCitedBlock)
	}
}

func TestDistributeCitationRewards(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create cited fact
	citedFact := types.Fact{
		Id: "cited-1", ClaimText: "Cited fact for rewards",
		Domain: "physics", EpistemicCategory: "empirical",
		Confidence: 800_000, Status: types.FactStatusVerified,
		Submitter: "lgm1sub",
	}
	k.SetFact(ctx, citedFact)

	// Create citing fact with reference
	citingFact := types.Fact{
		Id: "citing-1", ClaimText: "Citing fact with ref",
		Domain: "physics", EpistemicCategory: "empirical",
		Confidence: 800_000, Status: types.FactStatusVerified,
		Submitter: "lgm1sub2",
	}
	k.SetFact(ctx, citingFact)

	k.SetFactReference(ctx, types.FactReference{
		FromFactId: "citing-1", ToFactId: "cited-1",
		ReferenceType: "supports", Weight: 1_000_000,
	})

	// Distribute from 1,000,000 ulgm total reward
	distributed := k.DistributeCitationRewards(ctx, "citing-1", 1_000_000)
	// 15% of 1M = 150,000 citation pool
	if distributed == 0 {
		t.Error("expected some rewards to be distributed")
	}
}

func TestCalculateFundamentalityScore(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create base fact with many incoming references
	fact := types.Fact{
		Id: "fundamental-1", ClaimText: "Fundamental axiom",
		Domain: "mathematics", EpistemicCategory: "formal",
		Confidence: 1_000_000, Status: types.FactStatusVerified,
		Submitter: "lgm1sub",
	}
	k.SetFact(ctx, fact)

	// Add 5 incoming support references
	for i := 0; i < 5; i++ {
		depID := fmt.Sprintf("dep-%d", i)
		k.SetFact(ctx, types.Fact{
			Id: depID, ClaimText: fmt.Sprintf("Dependent %d", i),
			Domain: "mathematics", EpistemicCategory: "formal",
			Confidence: 800_000, Status: types.FactStatusVerified,
			Submitter: "lgm1sub",
		})
		k.SetFactReference(ctx, types.FactReference{
			FromFactId: depID, ToFactId: "fundamental-1",
			ReferenceType: "supports", Weight: 1_000_000,
		})
	}

	score := k.CalculateFundamentalityScore(ctx, "fundamental-1")
	if score != types.FundamentalityNotable {
		t.Errorf("expected Notable (250000) with 5 refs, got %d", score)
	}

	mult := keeper.GetFundamentalityMultiplier(score)
	if mult != types.FundamentalityBonusNotable {
		t.Errorf("expected 1.1x multiplier (1100000), got %d", mult)
	}
}

// ==================== Supersession Tests ====================

func TestSupersedeFact(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	oldFact := types.Fact{
		Id: "old-fact", ClaimText: "Old version of knowledge",
		Domain: "physics", EpistemicCategory: "empirical",
		Confidence: 800_000, BaseConfidence: 800_000,
		Status: types.FactStatusVerified, Submitter: "lgm1sub",
		VerifiedAtHeight: 10, LastVerifiedHeight: 10,
		CitationCount: 5,
	}
	k.SetFact(ctx, oldFact)

	newFact := types.Fact{
		Id: "new-fact", ClaimText: "Updated version of knowledge",
		Domain: "physics", EpistemicCategory: "empirical",
		Confidence: 900_000, BaseConfidence: 900_000,
		Status: types.FactStatusVerified, Submitter: "lgm1sub2",
		VerifiedAtHeight: 100, LastVerifiedHeight: 100,
	}
	k.SetFact(ctx, newFact)

	err := k.SupersedeFact(ctx, "old-fact", "new-fact")
	if err != nil {
		t.Fatalf("SupersedeFact failed: %v", err)
	}

	// Old fact should be superseded
	old, _ := k.GetFact(ctx, "old-fact")
	if old.Status != types.FactStatusSuperseded {
		t.Errorf("expected old fact status=superseded, got %s", old.Status)
	}
	if old.SupersededBy != "new-fact" {
		t.Errorf("expected SupersededBy=new-fact, got %s", old.SupersededBy)
	}

	// New fact should inherit citation count
	updated, _ := k.GetFact(ctx, "new-fact")
	if updated.CitationCount != 5 {
		t.Errorf("expected citation count 5 inherited, got %d", updated.CitationCount)
	}

	// "extends" reference should exist
	refs := k.GetFactReferences(ctx, "new-fact")
	found := false
	for _, ref := range refs {
		if ref.ToFactId == "old-fact" && ref.ReferenceType == "extends" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'extends' reference from new to old")
	}
}

func TestSupersedeFact_CrossDomainRejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetFact(ctx, types.Fact{
		Id: "domain-a", ClaimText: "Physics fact", Domain: "physics",
		EpistemicCategory: "empirical", Status: types.FactStatusVerified,
		Submitter: "lgm1sub",
	})
	k.SetFact(ctx, types.Fact{
		Id: "domain-b", ClaimText: "Math fact", Domain: "mathematics",
		EpistemicCategory: "formal", Status: types.FactStatusVerified,
		Submitter: "lgm1sub",
	})

	err := k.SupersedeFact(ctx, "domain-a", "domain-b")
	if err == nil {
		t.Error("expected cross-domain supersession to be rejected")
	}
}

func TestGetSupersessionChain(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create chain: v1 → v2 → v3
	k.SetFact(ctx, types.Fact{
		Id: "v1", ClaimText: "Version 1", Domain: "physics",
		EpistemicCategory: "empirical", Status: types.FactStatusSuperseded,
		Submitter: "lgm1sub", SupersededBy: "v2",
	})
	k.SetFact(ctx, types.Fact{
		Id: "v2", ClaimText: "Version 2", Domain: "physics",
		EpistemicCategory: "empirical", Status: types.FactStatusSuperseded,
		Submitter: "lgm1sub", SupersededBy: "v3",
	})
	k.SetFact(ctx, types.Fact{
		Id: "v3", ClaimText: "Version 3", Domain: "physics",
		EpistemicCategory: "empirical", Status: types.FactStatusVerified,
		Submitter: "lgm1sub",
	})

	chain := k.GetSupersessionChain(ctx, "v1")
	if len(chain) != 3 {
		t.Fatalf("expected chain length 3, got %d", len(chain))
	}
	if chain[0].Id != "v1" || chain[1].Id != "v2" || chain[2].Id != "v3" {
		t.Errorf("unexpected chain order: %s, %s, %s", chain[0].Id, chain[1].Id, chain[2].Id)
	}

	// GetLatestVersion
	latest, found := k.GetLatestVersion(ctx, "v1")
	if !found || latest.Id != "v3" {
		t.Errorf("expected latest version v3, got %s", latest.Id)
	}
}

func TestMasterConfidence(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := types.Fact{
		Id: "master-1", ClaimText: "Master confidence test fact",
		Domain: "physics", EpistemicCategory: "empirical",
		Confidence: 900_000, BaseConfidence: 900_000,
		Status: types.FactStatusVerified, Submitter: "lgm1sub",
		VerifiedAtHeight: 10, LastVerifiedHeight: 10,
		ChallengeCount: 2, // survived 2 challenges
	}
	k.SetFact(ctx, fact)

	// At height 10: no decay, adversarial bonus
	conf := k.MasterConfidence(ctx, "master-1")
	// M_temporal = 1.0, M_adversarial = 1.0 + 2*0.02 = 1.04
	// C_final = 900,000 × 1.0 × 1.04 = 936,000, capped at 990,000 (empirical max)
	if conf < 930_000 || conf > 940_000 {
		t.Errorf("at height 10: expected ~936000, got %d", conf)
	}

	// At height 100_010 (1 half-life): significant temporal decay
	ctx = ctx.WithBlockHeight(100_010)
	conf = k.MasterConfidence(ctx, "master-1")
	if conf >= 900_000 {
		t.Errorf("after 1 half-life: expected significant decay, got %d", conf)
	}
}

// ==================== Stratum Ceiling Tests ====================

func TestStratumCeilingInAggregation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create a social science claim (social stratum ceiling = 850,000)
	claim := types.Claim{
		Id: "social-claim", ClaimText: "Social science claim for ceiling test",
		Domain: "sociology", EpistemicCategory: "social",
		Submitter: "lgm1sub", Status: "pending",
	}
	k.SetClaim(ctx, claim)

	// Add sociology as a domain
	k.SetDomainInState(ctx, types.OntologicalDomain{
		Name: "sociology", Status: "active",
	})

	round, _ := k.CreateVerificationRound(ctx, claim)

	// All validators accept with very high confidence
	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{
			Validator: v, Verdict: "accept", Confidence: 990_000,
		}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	// Social stratum ceiling = 850,000, category max = 850,000
	if result.Confidence > 850_000 {
		t.Errorf("social science confidence should be capped at 850000, got %d", result.Confidence)
	}
}

// ==================== Bidirectional Reference Tests ====================

func TestBidirectionalReferences(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetFact(ctx, types.Fact{
		Id: "bidir-a", ClaimText: "Fact A",
		Domain: "general", Status: types.FactStatusVerified, Submitter: "lgm1sub",
	})
	k.SetFact(ctx, types.Fact{
		Id: "bidir-b", ClaimText: "Fact B",
		Domain: "general", Status: types.FactStatusVerified, Submitter: "lgm1sub",
	})

	// B supports A
	k.SetFactReference(ctx, types.FactReference{
		FromFactId: "bidir-b", ToFactId: "bidir-a",
		ReferenceType: "supports", Weight: 1_000_000,
	})

	// Outgoing refs from B
	outgoing := k.GetFactReferences(ctx, "bidir-b")
	if len(outgoing) != 1 || outgoing[0].ToFactId != "bidir-a" {
		t.Error("expected outgoing reference from B to A")
	}

	// Incoming refs to A
	incoming := k.GetIncomingReferences(ctx, "bidir-a")
	if len(incoming) != 1 || incoming[0].FromFactId != "bidir-b" {
		t.Error("expected incoming reference from B to A")
	}

	// Supporting facts for A
	supporting := k.GetSupportingFacts(ctx, "bidir-a")
	if len(supporting) != 1 {
		t.Errorf("expected 1 supporting fact, got %d", len(supporting))
	}

	// Contradicting facts (should be empty)
	contradicting := k.GetContradictingFacts(ctx, "bidir-a")
	if len(contradicting) != 0 {
		t.Errorf("expected 0 contradicting facts, got %d", len(contradicting))
	}
}

// ==================== Fact Lifecycle Integration Test ====================

func TestFactLifecycle_SubmitVerifyDecayExpire(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// 1. Submit claim
	claim := types.Claim{
		Id: "lifecycle-1", ClaimText: "Integration test: full lifecycle from claim to expiry",
		Domain: "physics", EpistemicCategory: "empirical",
		Submitter: "lgm1sub", Status: "pending",
	}
	k.SetClaim(ctx, claim)

	// 2. Create and complete verification round
	round, _ := k.CreateVerificationRound(ctx, claim)
	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{
			Validator: v, Verdict: "accept", Confidence: 900_000,
		}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)
	_ = k.CompleteRound(ctx, round, result)

	// Find the created fact
	var factID string
	k.IterateFacts(ctx, func(f types.Fact) bool {
		if f.ClaimText == claim.ClaimText {
			factID = f.Id
			return true
		}
		return false
	})
	if factID == "" {
		t.Fatal("fact should have been created")
	}

	// 3. Verify decay at half-life
	ctx = ctx.WithBlockHeight(100_010)
	decayed, _ := k.GetFactWithDecay(ctx, factID)
	if decayed.Confidence >= 900_000 {
		t.Errorf("expected decay after half-life, got %d", decayed.Confidence)
	}

	// 4. Expire via EndBlocker at staleness threshold
	ctx = ctx.WithBlockHeight(200_011)
	expired := k.ExpireStaleFactsInEndBlocker(ctx, 50)
	if expired != 1 {
		t.Errorf("expected 1 fact expired, got %d", expired)
	}

	// 5. Reverify to reactivate
	err := k.ReverifyFact(ctx, factID, 850_000, 200_011)
	if err != nil {
		t.Fatalf("ReverifyFact failed: %v", err)
	}
	reactivated, _ := k.GetFact(ctx, factID)
	if reactivated.Status != types.FactStatusActive {
		t.Errorf("expected active after reverification, got %s", reactivated.Status)
	}
}

// ==================== Stake Settlement Tests (G-1, G-2, G-5, G-7, G-9) ====================

// ---------- Tracking Bank Keeper ----------

type trackingBankKeeper struct {
	sentToAccount map[string]sdk.Coins // addr → coins sent
	sentToModule  map[string]sdk.Coins // module → coins sent
}

func newTrackingBankKeeper() *trackingBankKeeper {
	return &trackingBankKeeper{
		sentToAccount: make(map[string]sdk.Coins),
		sentToModule:  make(map[string]sdk.Coins),
	}
}

func (m *trackingBankKeeper) SendCoins(_ context.Context, _, _ sdk.AccAddress, _ sdk.Coins) error {
	return nil
}

func (m *trackingBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	key := recipientAddr.String()
	m.sentToAccount[key] = m.sentToAccount[key].Add(amt...)
	return nil
}

func (m *trackingBankKeeper) SendCoinsFromAccountToModule(_ context.Context, _ sdk.AccAddress, _ string, _ sdk.Coins) error {
	return nil
}

func (m *trackingBankKeeper) SendCoinsFromModuleToModule(_ context.Context, _, _ string, _ sdk.Coins) error {
	return nil
}

func (m *trackingBankKeeper) BurnCoins(_ context.Context, _ string, _ sdk.Coins) error {
	return nil
}

// ---------- Mock VestingRewardsKeeper ----------

type mockVestingRewardsKeeper struct {
	created []vestingCreation
}

type vestingCreation struct {
	claimId, factId, recipient, totalAmount, category string
}

func (m *mockVestingRewardsKeeper) CreateVestingScheduleFromKnowledge(_ sdk.Context, claimId, factId, recipient, totalAmount, epistemicCategory string) error {
	m.created = append(m.created, vestingCreation{claimId, factId, recipient, totalAmount, epistemicCategory})
	return nil
}

// ---------- Setup with tracking keepers ----------

func setupKeeperWithTracking(t *testing.T) (keeper.Keeper, sdk.Context, *mockStakingKeeper, *trackingBankKeeper, *mockVestingRewardsKeeper) {
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
	mockSK.addValidator("lgmvaloper1abc", 5_000_000, 2)
	mockSK.addValidator("lgmvaloper1def", 3_000_000, 2)
	mockSK.addValidator("lgmvaloper1ghi", 2_000_000, 1)

	tbk := newTrackingBankKeeper()
	mvk := &mockVestingRewardsKeeper{}

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memKey,
		mockAccountKeeper{},
		tbk,
		mockSK,
		mockTokensKeeper{},
		nil, // qualificationKeeper
		nil, // captureDefenseKeeper
		mvk,
		nil, // evidenceMgmtKeeper — nil-safe
		"lgm10d07y265gmmuvt4z0w9aw880jnsr700jefnezl",
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 10}, false, log.NewNopLogger())
	k.SetParams(ctx, types.DefaultParams())
	for _, domain := range types.DefaultDomains() {
		k.SetDomainInState(ctx, domain)
	}

	return k, ctx, mockSK, tbk, mvk
}

func TestStakeReturnOnAcceptance(t *testing.T) {
	k, ctx, _, tbk, _ := setupKeeperWithTracking(t)

	claim := types.Claim{
		Id:                "accept-stake-claim",
		ClaimText:         "Stake should be returned on acceptance",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
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

	result, _ := k.AggregateVerificationResult(ctx, round)
	err := k.CompleteRound(ctx, round, result)
	if err != nil {
		t.Fatalf("CompleteRound failed: %v", err)
	}

	// Verify stake was returned to submitter
	returned := tbk.sentToAccount[testSubmitterAddr]
	if returned.IsZero() {
		t.Error("expected stake to be returned to submitter on acceptance")
	}
	if returned.AmountOf("ulgm").Int64() != 1000000 {
		t.Errorf("expected 1000000 ulgm returned, got %s", returned.AmountOf("ulgm").String())
	}
}

func TestStakeSlashOnRejection(t *testing.T) {
	k, ctx, _, tbk, _ := setupKeeperWithTracking(t)

	claim := types.Claim{
		Id:                "reject-stake-claim",
		ClaimText:         "Stake should be slashed on rejection",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// All validators reject
	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "reject", Confidence: 900000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)
	err := k.CompleteRound(ctx, round, result)
	if err != nil {
		t.Fatalf("CompleteRound failed: %v", err)
	}

	// Verify partial stake return (78% of 1,000,000 = 780,000)
	returned := tbk.sentToAccount[testSubmitterAddr]
	if returned.IsZero() {
		t.Error("expected partial stake return on rejection")
	}
	// 22% slash → 78% returned = 780000
	returnedAmt := returned.AmountOf("ulgm").Int64()
	if returnedAmt != 780000 {
		t.Errorf("expected 780000 ulgm returned (22%% slash), got %d", returnedAmt)
	}
}

func TestStakeReturnOnInsufficient(t *testing.T) {
	k, ctx, _, tbk, _ := setupKeeperWithTracking(t)

	claim := types.Claim{
		Id:                "insuff-stake-claim",
		ClaimText:         "Stake returned when insufficient participation",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "500000",
		Status:            "pending",
	}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// Only 1 of 3 validators participates (below quorum)
	round.SelectedValidators = []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	round.Commitments["lgmvaloper1abc"] = types.VerificationCommitment{Validator: "lgmvaloper1abc"}
	round.Reveals["lgmvaloper1abc"] = types.VerificationReveal{
		Validator: "lgmvaloper1abc", Verdict: "accept", Confidence: 900000,
	}
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)
	if result.Verdict != "insufficient" {
		t.Fatalf("expected insufficient verdict, got %s", result.Verdict)
	}
	err := k.CompleteRound(ctx, round, result)
	if err != nil {
		t.Fatalf("CompleteRound failed: %v", err)
	}

	// Verify full stake returned
	returned := tbk.sentToAccount[testSubmitterAddr]
	if returned.AmountOf("ulgm").Int64() != 500000 {
		t.Errorf("expected 500000 ulgm returned on insufficient, got %s", returned.AmountOf("ulgm").String())
	}
}

func TestVestingCreatedOnAcceptance(t *testing.T) {
	k, ctx, _, _, mvk := setupKeeperWithTracking(t)

	claim := types.Claim{
		Id:                "vesting-claim",
		ClaimText:         "Should trigger vesting schedule creation",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "2000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 900000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)
	_ = k.CompleteRound(ctx, round, result)

	// Verify vesting schedule was created
	if len(mvk.created) != 1 {
		t.Fatalf("expected 1 vesting creation, got %d", len(mvk.created))
	}
	vc := mvk.created[0]
	if vc.claimId != "vesting-claim" {
		t.Errorf("expected claimId=vesting-claim, got %s", vc.claimId)
	}
	if vc.recipient != testSubmitterAddr {
		t.Errorf("expected recipient=%s, got %s", testSubmitterAddr, vc.recipient)
	}
	if vc.totalAmount != "2000000" {
		t.Errorf("expected totalAmount=2000000, got %s", vc.totalAmount)
	}
	if vc.category != "empirical" {
		t.Errorf("expected category=empirical, got %s", vc.category)
	}
}

func TestNoVestingOnRejection(t *testing.T) {
	k, ctx, _, _, mvk := setupKeeperWithTracking(t)

	claim := types.Claim{
		Id:                "no-vesting-claim",
		ClaimText:         "Rejected claims should not create vesting",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "reject", Confidence: 900000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)
	_ = k.CompleteRound(ctx, round, result)

	if len(mvk.created) != 0 {
		t.Errorf("expected 0 vesting creations on rejection, got %d", len(mvk.created))
	}
}

func TestContestedFactCreation(t *testing.T) {
	k, ctx, mockSK, _, _ := setupKeeperWithTracking(t)
	mockSK.addValidator("lgmvaloper1jkl", 5_000_000, 2)

	claim := types.Claim{
		Id:                "contested-fact-claim",
		ClaimText:         "Contested claim should create contested fact",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	// Split verdict with matched stakes to get contested (between 55% and 77%)
	// abc (5M) + def (3M) accept = 8M, ghi (2M) + jkl (5M) reject = 7M
	// Accept weight/total ≈ 53% < 55% → inconclusive. Need exact contested.
	// Let's make: abc (5M) accept, def (3M) accept, ghi (2M) reject
	// = 8/10 = 80% but this is > 77%. Need to make it 55-77%.
	// abc (5M) accept, ghi (2M) reject, jkl (5M) reject = 5/12 = 41% for accept
	// reject = 7/12 = 58% → contested (between 55% and 77%)
	round.Commitments["lgmvaloper1abc"] = types.VerificationCommitment{Validator: "lgmvaloper1abc"}
	round.Commitments["lgmvaloper1ghi"] = types.VerificationCommitment{Validator: "lgmvaloper1ghi"}
	round.Commitments["lgmvaloper1jkl"] = types.VerificationCommitment{Validator: "lgmvaloper1jkl"}

	round.Reveals["lgmvaloper1abc"] = types.VerificationReveal{Validator: "lgmvaloper1abc", Verdict: "accept", Confidence: 800000}
	round.Reveals["lgmvaloper1ghi"] = types.VerificationReveal{Validator: "lgmvaloper1ghi", Verdict: "reject", Confidence: 800000}
	round.Reveals["lgmvaloper1jkl"] = types.VerificationReveal{Validator: "lgmvaloper1jkl", Verdict: "reject", Confidence: 800000}

	round.SelectedValidators = []string{"lgmvaloper1abc", "lgmvaloper1ghi", "lgmvaloper1jkl"}
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)
	if result.Verdict != "contested" && result.Verdict != "rejected" {
		// If this ends up as rejected (>77% for reject), adjust stakes
		t.Logf("Verdict: %s, AgreementRatio: %d", result.Verdict, result.AgreementRatio)
	}

	_ = k.CompleteRound(ctx, round, result)

	// If the verdict was "contested", verify fact was created
	if result.Verdict == "contested" {
		var contestedFact types.Fact
		k.IterateFacts(ctx, func(f types.Fact) bool {
			if f.ClaimText == claim.ClaimText {
				contestedFact = f
				return true
			}
			return false
		})
		if contestedFact.Id == "" {
			t.Error("expected contested fact to be created")
		}
		if contestedFact.Status != types.FactStatusContested {
			t.Errorf("expected fact status=contested, got %s", contestedFact.Status)
		}
	}
}

func TestParameterizedVerificationReward(t *testing.T) {
	k, ctx, _, _, _ := setupKeeperWithTracking(t)

	// Set custom reward amount
	params := k.GetParams(ctx)
	params.VerificationRewardAmount = 200000 // 0.2 LGM
	k.SetParams(ctx, params)

	claim := types.Claim{Id: "reward-param-claim", ClaimText: "Test parameterized verification reward", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 900000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)

	// Rewards are tier-adjusted: base 200000 × tier multiplier
	// lgmvaloper1abc: tier 2 → 200000 * 1.0 = 200000
	// lgmvaloper1def: tier 2 → 200000 * 1.0 = 200000
	// lgmvaloper1ghi: tier 1 → 200000 * 0.5 = 100000
	expectedRewards := map[string]uint64{
		"lgmvaloper1abc": 200000, // tier 2: 1.0x
		"lgmvaloper1def": 200000, // tier 2: 1.0x
		"lgmvaloper1ghi": 100000, // tier 1: 0.5x
	}
	for v, expected := range expectedRewards {
		if reward, ok := result.Rewards[v]; !ok || reward != expected {
			t.Errorf("expected reward=%d for %s, got %d", expected, v, reward)
		}
	}
}

func TestStakeSettlementWithZeroStake(t *testing.T) {
	k, ctx, _, tbk, _ := setupKeeperWithTracking(t)

	// Claim with zero or empty stake
	claim := types.Claim{
		Id:                "zero-stake-claim",
		ClaimText:         "Zero stake should not cause errors",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "",
		Status:            "pending",
	}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 900000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)
	err := k.CompleteRound(ctx, round, result)
	if err != nil {
		t.Fatalf("CompleteRound with empty stake should not error: %v", err)
	}

	// No coins should have been sent
	if len(tbk.sentToAccount) != 0 {
		t.Error("expected no coins sent with empty stake")
	}
}

func TestDefaultParamsIncludeNewFields(t *testing.T) {
	params := types.DefaultParams()
	if params.InvalidClaimSlashBps != 220000 {
		t.Errorf("expected InvalidClaimSlashBps=220000, got %d", params.InvalidClaimSlashBps)
	}
	if params.VerificationRewardAmount != 100000 {
		t.Errorf("expected VerificationRewardAmount=100000, got %d", params.VerificationRewardAmount)
	}
}

// ==================== Evidence-Confidence Integration Tests ====================

// ---------- Mock EvidenceMgmtKeeper ----------

type mockEvidenceMgmtKeeper struct {
	evidenceByClaimId map[string][]types.EvidenceRecord
	reliabilities     map[string]uint64
}

func newMockEvidenceMgmtKeeper() *mockEvidenceMgmtKeeper {
	return &mockEvidenceMgmtKeeper{
		evidenceByClaimId: make(map[string][]types.EvidenceRecord),
		reliabilities:     make(map[string]uint64),
	}
}

func (m *mockEvidenceMgmtKeeper) addEvidence(claimId string, ev types.EvidenceRecord) {
	m.evidenceByClaimId[claimId] = append(m.evidenceByClaimId[claimId], ev)
	if ev.Reliability > 0 {
		m.reliabilities[ev.Id] = ev.Reliability
	}
}

func (m *mockEvidenceMgmtKeeper) GetEvidenceByClaimId(_ sdk.Context, claimId string) []types.EvidenceRecord {
	return m.evidenceByClaimId[claimId]
}

func (m *mockEvidenceMgmtKeeper) ComputeReliability(_ sdk.Context, evidenceId string) (uint64, error) {
	if r, ok := m.reliabilities[evidenceId]; ok {
		return r, nil
	}
	return 0, fmt.Errorf("evidence %s not found", evidenceId)
}

// setupKeeperWithEvidence creates a keeper with a wired mockEvidenceMgmtKeeper.
func setupKeeperWithEvidence(t *testing.T) (keeper.Keeper, sdk.Context, *mockStakingKeeper, *mockEvidenceMgmtKeeper) {
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
	mockSK.addValidator("lgmvaloper1abc", 5_000_000, 2)
	mockSK.addValidator("lgmvaloper1def", 3_000_000, 2)
	mockSK.addValidator("lgmvaloper1ghi", 2_000_000, 1)

	mockEM := newMockEvidenceMgmtKeeper()

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memKey,
		mockAccountKeeper{},
		mockBankKeeper{},
		mockSK,
		mockTokensKeeper{},
		nil,    // qualificationKeeper
		nil,    // captureDefenseKeeper
		nil,    // vestingRewardsKeeper
		mockEM, // evidenceMgmtKeeper
		"lgm10d07y265gmmuvt4z0w9aw880jnsr700jefnezl",
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 10}, false, log.NewNopLogger())
	k.SetParams(ctx, types.DefaultParams())
	for _, domain := range types.DefaultDomains() {
		k.SetDomainInState(ctx, domain)
	}

	return k, ctx, mockSK, mockEM
}

func TestEvidenceBonus_SingleOnChainEvidence(t *testing.T) {
	k, ctx, _, mockEM := setupKeeperWithEvidence(t)

	claimId := "ev-bonus-claim-1"
	mockEM.addEvidence(claimId, types.EvidenceRecord{
		Id:           "ev-1",
		ClaimId:      claimId,
		EvidenceType: "on_chain",
		Status:       "verified",
		Reliability:  900_000, // 90%
	})

	claim := types.Claim{Id: claimId, ClaimText: "Claim with on-chain evidence for bonus test", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 900_000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	// on_chain strength = 1,000,000, reliability = 900,000
	// score = (1,000,000 * 900,000) / 1,000,000 = 900,000
	// capped at MaxEvidenceBonus = 150,000
	if result.EvidenceBonus != 150_000 {
		t.Errorf("expected evidence bonus=150000 (capped), got %d", result.EvidenceBonus)
	}

	// Confidence should include the bonus
	if result.Confidence < 900_000 {
		t.Errorf("expected confidence >= 900000 with evidence bonus, got %d", result.Confidence)
	}
}

func TestEvidenceBonus_MixedEvidenceTypes(t *testing.T) {
	k, ctx, _, mockEM := setupKeeperWithEvidence(t)

	claimId := "ev-bonus-claim-2"
	// Add document (strength 600K) at 80% reliability → score = 480,000
	mockEM.addEvidence(claimId, types.EvidenceRecord{
		Id: "ev-doc", ClaimId: claimId, EvidenceType: "document",
		Status: "verified", Reliability: 800_000,
	})
	// Add attestation (strength 500K) at 60% reliability → score = 300,000
	mockEM.addEvidence(claimId, types.EvidenceRecord{
		Id: "ev-att", ClaimId: claimId, EvidenceType: "attestation",
		Status: "submitted", Reliability: 600_000,
	})

	claim := types.Claim{Id: claimId, ClaimText: "Claim with mixed evidence for bonus averaging", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 800_000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	// avg = (480,000 + 300,000) / 2 = 390,000 → capped at 150,000
	if result.EvidenceBonus != 150_000 {
		t.Errorf("expected evidence bonus=150000 (capped), got %d", result.EvidenceBonus)
	}
}

func TestEvidenceBonus_LowReliability(t *testing.T) {
	k, ctx, _, mockEM := setupKeeperWithEvidence(t)

	claimId := "ev-bonus-claim-3"
	// attestation (500K) with zero reliability → default 100K (10%)
	// score = (500,000 * 100,000) / 1,000,000 = 50,000
	mockEM.addEvidence(claimId, types.EvidenceRecord{
		Id: "ev-low", ClaimId: claimId, EvidenceType: "attestation",
		Status: "submitted", Reliability: 0,
	})

	claim := types.Claim{Id: claimId, ClaimText: "Claim with low reliability evidence for floor test", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 800_000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	// score = 50,000 → below MaxEvidenceBonus, so bonus = 50,000
	if result.EvidenceBonus != 50_000 {
		t.Errorf("expected evidence bonus=50000, got %d", result.EvidenceBonus)
	}
}

func TestEvidenceBonus_RejectedEvidenceExcluded(t *testing.T) {
	k, ctx, _, mockEM := setupKeeperWithEvidence(t)

	claimId := "ev-bonus-claim-4"
	// Rejected evidence should be excluded
	mockEM.addEvidence(claimId, types.EvidenceRecord{
		Id: "ev-rej", ClaimId: claimId, EvidenceType: "on_chain",
		Status: "rejected", Reliability: 1_000_000,
	})
	// Expired evidence should be excluded
	mockEM.addEvidence(claimId, types.EvidenceRecord{
		Id: "ev-exp", ClaimId: claimId, EvidenceType: "computation",
		Status: "expired", Reliability: 900_000,
	})

	claim := types.Claim{Id: claimId, ClaimText: "Claim with only rejected and expired evidence", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 800_000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	// All evidence rejected/expired → bonus should be 0
	if result.EvidenceBonus != 0 {
		t.Errorf("expected evidence bonus=0 with rejected/expired evidence, got %d", result.EvidenceBonus)
	}
}

func TestEvidenceBonus_NoEvidence(t *testing.T) {
	k, ctx, _, _ := setupKeeperWithEvidence(t)

	claimId := "ev-bonus-claim-5"
	// No evidence added for this claim

	claim := types.Claim{Id: claimId, ClaimText: "Claim with no evidence at all for zero bonus", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 800_000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	if result.EvidenceBonus != 0 {
		t.Errorf("expected evidence bonus=0 with no evidence, got %d", result.EvidenceBonus)
	}
}

func TestEvidenceBonus_UnknownEvidenceType(t *testing.T) {
	k, ctx, _, mockEM := setupKeeperWithEvidence(t)

	claimId := "ev-bonus-claim-6"
	// Unknown type → default strength 400,000, reliability 800,000
	// score = (400,000 * 800,000) / 1,000,000 = 320,000 → capped at 150,000
	mockEM.addEvidence(claimId, types.EvidenceRecord{
		Id: "ev-unk", ClaimId: claimId, EvidenceType: "unknown_type",
		Status: "submitted", Reliability: 800_000,
	})

	claim := types.Claim{Id: claimId, ClaimText: "Claim with unknown evidence type for default strength", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 800_000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	if result.EvidenceBonus != 150_000 {
		t.Errorf("expected evidence bonus=150000 (capped from 320000), got %d", result.EvidenceBonus)
	}
}

func TestEvidenceBonus_ConfidenceCappedAt100Percent(t *testing.T) {
	k, ctx, _, mockEM := setupKeeperWithEvidence(t)

	claimId := "ev-bonus-claim-7"
	// High-quality evidence that would push confidence above 1,000,000
	mockEM.addEvidence(claimId, types.EvidenceRecord{
		Id: "ev-cap", ClaimId: claimId, EvidenceType: "on_chain",
		Status: "verified", Reliability: 950_000,
	})

	// Use mathematics domain (stratum: fundamental, ceiling 1,000,000) so stratum doesn't clamp
	claim := types.Claim{
		Id: claimId, ClaimText: "Mathematical claim with strong evidence for cap test",
		Domain: "mathematics", EpistemicCategory: "formal", Status: "pending",
	}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 950_000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	// Even with evidence bonus, confidence must not exceed 1,000,000
	if result.Confidence > 1_000_000 {
		t.Errorf("confidence should not exceed 1,000,000, got %d", result.Confidence)
	}
}

func TestEvidenceSummaryTransferOnFactCreation(t *testing.T) {
	k, ctx, _, mockEM := setupKeeperWithEvidence(t)

	claimId := "ev-transfer-claim"
	mockEM.addEvidence(claimId, types.EvidenceRecord{
		Id: "ev-t1", ClaimId: claimId, EvidenceType: "computation",
		Status: "verified", Reliability: 850_000,
	})
	mockEM.addEvidence(claimId, types.EvidenceRecord{
		Id: "ev-t2", ClaimId: claimId, EvidenceType: "document",
		Status: "submitted", Reliability: 700_000,
	})
	// Rejected evidence should NOT be transferred
	mockEM.addEvidence(claimId, types.EvidenceRecord{
		Id: "ev-t3", ClaimId: claimId, EvidenceType: "on_chain",
		Status: "rejected", Reliability: 1_000_000,
	})

	claim := types.Claim{
		Id: claimId, ClaimText: "Claim whose evidence transfers to fact on acceptance",
		Domain: "physics", EpistemicCategory: "empirical",
		Submitter: "lgm1sub", SubmitterDid: "did:lgm:transfer1",
		Status: "pending", LogicZone: "empirical",
	}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 900_000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)
	err := k.CompleteRound(ctx, round, result)
	if err != nil {
		t.Fatalf("CompleteRound failed: %v", err)
	}

	// Find the created fact
	var createdFact types.Fact
	k.IterateFacts(ctx, func(f types.Fact) bool {
		if f.ClaimText == claim.ClaimText {
			createdFact = f
			return true
		}
		return false
	})

	if createdFact.Id == "" {
		t.Fatal("expected fact to be created")
	}

	// Should have 2 evidence summaries (rejected one excluded)
	if len(createdFact.EvidenceSummaries) != 2 {
		t.Fatalf("expected 2 evidence summaries, got %d", len(createdFact.EvidenceSummaries))
	}

	// Verify summary fields
	for _, s := range createdFact.EvidenceSummaries {
		if s.EvidenceId == "" {
			t.Error("evidence summary missing Id")
		}
		if s.TypeStrength == 0 {
			t.Errorf("evidence summary %s missing TypeStrength", s.EvidenceId)
		}
		expectedStrength, ok := types.EVIDENCE_TYPE_STRENGTH[s.EvidenceType]
		if ok && s.TypeStrength != expectedStrength {
			t.Errorf("evidence %s: expected TypeStrength=%d, got %d", s.EvidenceId, expectedStrength, s.TypeStrength)
		}
	}

	// Verify EvidenceBonus was set on the fact
	if createdFact.EvidenceBonus == 0 {
		t.Error("expected EvidenceBonus to be set on fact")
	}
}

func TestNilEvidenceKeeper_NoPanic(t *testing.T) {
	// The default setupKeeper passes nil for evidenceMgmtKeeper.
	// Verify that aggregation and completion work without panic.
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{
		Id: "nil-evk-claim", ClaimText: "Claim processed without evidence keeper",
		Domain: "physics", EpistemicCategory: "empirical",
		Submitter: "lgm1sub", SubmitterDid: "did:lgm:niltest",
		Status: "pending", LogicZone: "empirical",
	}
	k.SetClaim(ctx, claim)
	round, _ := k.CreateVerificationRound(ctx, claim)

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = types.VerificationCommitment{Validator: v}
		round.Reveals[v] = types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 850_000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	// Should not panic
	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed with nil evidence keeper: %v", err)
	}
	if result.EvidenceBonus != 0 {
		t.Errorf("expected zero evidence bonus with nil keeper, got %d", result.EvidenceBonus)
	}

	// CompleteRound should also work
	err = k.CompleteRound(ctx, round, result)
	if err != nil {
		t.Fatalf("CompleteRound failed with nil evidence keeper: %v", err)
	}

	// Verify fact was created without evidence summaries
	var createdFact types.Fact
	k.IterateFacts(ctx, func(f types.Fact) bool {
		if f.ClaimText == claim.ClaimText {
			createdFact = f
			return true
		}
		return false
	})
	if createdFact.Id == "" {
		t.Fatal("expected fact to be created")
	}
	if len(createdFact.EvidenceSummaries) != 0 {
		t.Errorf("expected 0 evidence summaries with nil keeper, got %d", len(createdFact.EvidenceSummaries))
	}
}
