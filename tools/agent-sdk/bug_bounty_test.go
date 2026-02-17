package agentsdk

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// --- Bug Bounty Mock Clients ---

// bbMockTxClient tracks calls for bug bounty workflow testing.
type bbMockTxClient struct {
	claims       []SubmitClaimRequest
	lips         []SubmitLIPRequest
	vestings     []CreateVestingRequest
	submitErr    error
	lipErr       error
	vestingErr   error
}

func (m *bbMockTxClient) RegisterAccount(_ context.Context, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) RegisterAgent(_ context.Context, _ RegisterAgentRequest) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) Heartbeat(_ context.Context) (string, error) { return "mock-tx", nil }
func (m *bbMockTxClient) SubmitClaim(_ context.Context, req SubmitClaimRequest) (string, error) {
	if m.submitErr != nil {
		return "", m.submitErr
	}
	m.claims = append(m.claims, req)
	return fmt.Sprintf("claim-tx-%d", len(m.claims)), nil
}
func (m *bbMockTxClient) SubmitCommitment(_ context.Context, _, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) SubmitReveal(_ context.Context, _ string, _ Verdict, _ uint32, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) ChallengeFact(_ context.Context, _, _ string, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) CreateHome(_ context.Context, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) UpdateMemoryCID(_ context.Context, _, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) Stake(_ context.Context, _ string, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) DeployContract(_ context.Context, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) CallContract(_ context.Context, _, _ string, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) ScheduleExecution(_ context.Context, _, _, _ string, _, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) ConfigureGuardian(_ context.Context, _, _ string, _ bool, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) RegisterHomeKey(_ context.Context, _, _, _ string, _ []string, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) StartSession(_ context.Context, _, _ string, _ []string) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) EndSession(_ context.Context, _, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) SetSpendingLimit(_ context.Context, _, _ string, _, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *bbMockTxClient) SubmitLIP(_ context.Context, req SubmitLIPRequest) (string, error) {
	if m.lipErr != nil {
		return "", m.lipErr
	}
	m.lips = append(m.lips, req)
	return fmt.Sprintf("lip-tx-%d", len(m.lips)), nil
}
func (m *bbMockTxClient) CreateVesting(_ context.Context, req CreateVestingRequest) (string, error) {
	if m.vestingErr != nil {
		return "", m.vestingErr
	}
	m.vestings = append(m.vestings, req)
	return fmt.Sprintf("vesting-tx-%d", len(m.vestings)), nil
}

// bbMockQueryClient provides canned responses for bug bounty queries.
type bbMockQueryClient struct {
	bugFacts  []FactInfo
	proposals []ProposalInfo
	bugErr    error
	propErr   error
}

func (m *bbMockQueryClient) ChainStatus(_ context.Context) (*ChainStatus, error) {
	return &ChainStatus{ChainID: ChainID, LatestHeight: 100}, nil
}
func (m *bbMockQueryClient) Balance(_ context.Context, _ string) (uint64, error) {
	return 1000000, nil
}
func (m *bbMockQueryClient) AgentProfile(_ context.Context, _ string) (*AgentProfileInfo, error) {
	return &AgentProfileInfo{}, nil
}
func (m *bbMockQueryClient) ActiveRounds(_ context.Context) ([]RoundInfo, error) {
	return nil, nil
}
func (m *bbMockQueryClient) RoundDetail(_ context.Context, _ string) (*RoundInfo, error) {
	return nil, nil
}
func (m *bbMockQueryClient) FactsByDomain(_ context.Context, _ string) ([]FactInfo, error) {
	return m.bugFacts, m.bugErr
}
func (m *bbMockQueryClient) ClaimsBySubmitter(_ context.Context, _ string) ([]ClaimInfo, error) {
	return nil, nil
}
func (m *bbMockQueryClient) GetClaim(_ context.Context, _ string) (*ClaimInfo, error) {
	return nil, nil
}
func (m *bbMockQueryClient) FactReferences(_ context.Context, _ string) ([]FactReference, error) {
	return nil, nil
}
func (m *bbMockQueryClient) CounterFacts(_ context.Context, _ string) ([]CounterFactInfo, error) {
	return nil, nil
}
func (m *bbMockQueryClient) BugFacts(_ context.Context, _ string) ([]FactInfo, error) {
	return m.bugFacts, m.bugErr
}
func (m *bbMockQueryClient) ProposalsByTrigger(_ context.Context, _ string) ([]ProposalInfo, error) {
	return m.proposals, m.propErr
}

// --- Tests ---

func TestSubmitBugReport(t *testing.T) {
	txClient := &bbMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	txHash, err := agent.SubmitBugReport(context.Background(),
		"BUG: validator slashing applies penalty twice when slash is called during unbonding period",
		"protocol",
		5000,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txHash != "claim-tx-1" {
		t.Fatalf("unexpected tx hash: %s", txHash)
	}
	if len(txClient.claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(txClient.claims))
	}
	claim := txClient.claims[0]
	if claim.Category != "protocol_observation" {
		t.Fatalf("expected category protocol_observation, got %s", claim.Category)
	}
	if claim.Domain != "protocol" {
		t.Fatalf("expected domain protocol, got %s", claim.Domain)
	}
	if claim.Stake != 5000 {
		t.Fatalf("expected stake 5000, got %d", claim.Stake)
	}
}

func TestSubmitBugReport_TooShort(t *testing.T) {
	txClient := &bbMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.SubmitBugReport(context.Background(), "short", "protocol", 5000)
	if err == nil {
		t.Fatal("expected error for short bug report")
	}
}

func TestSubmitBugReport_NoTxClient(t *testing.T) {
	agent := New(DefaultConfig())
	_, err := agent.SubmitBugReport(context.Background(),
		"BUG: some issue that is long enough to pass validation",
		"protocol", 5000)
	if err == nil {
		t.Fatal("expected error without tx client")
	}
}

func TestSubmitBugFix(t *testing.T) {
	txClient := &bbMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	// Override sleep for tests — we test the logic, not the timing.
	lipTx, err := agent.SubmitBugFix(context.Background(), BugFixRequest{
		Title:          "Fix double slashing",
		FixDescription: "FIX: Deduplicate slashing in keeper.go:340 by checking if slash was already applied",
		BugFactIDs:     []string{"fact-001", "fact-002"},
		Stake:          3000,
		Deposit:        10000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lipTx != "lip-tx-1" {
		t.Fatalf("unexpected LIP tx hash: %s", lipTx)
	}

	// Should have submitted both a claim and a LIP.
	if len(txClient.claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(txClient.claims))
	}
	if len(txClient.lips) != 1 {
		t.Fatalf("expected 1 LIP, got %d", len(txClient.lips))
	}

	// Verify claim fields.
	claim := txClient.claims[0]
	if claim.Category != "protocol_observation" {
		t.Fatalf("expected category protocol_observation, got %s", claim.Category)
	}
	if claim.Stake != 3000 {
		t.Fatalf("expected stake 3000, got %d", claim.Stake)
	}

	// Verify LIP fields.
	lip := txClient.lips[0]
	if lip.Title != "BugFix: Fix double slashing" {
		t.Fatalf("unexpected LIP title: %s", lip.Title)
	}
	if lip.FixClaimID != "claim-tx-1" {
		t.Fatalf("expected fix claim ID claim-tx-1, got %s", lip.FixClaimID)
	}
	if len(lip.BugFactIDs) != 2 {
		t.Fatalf("expected 2 bug fact IDs, got %d", len(lip.BugFactIDs))
	}
	if lip.Deposit != 10000 {
		t.Fatalf("expected deposit 10000, got %d", lip.Deposit)
	}
}

func TestSubmitBugFix_NoBugFacts(t *testing.T) {
	txClient := &bbMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.SubmitBugFix(context.Background(), BugFixRequest{
		Title:          "Fix something",
		FixDescription: "FIX: some fix that is long enough to pass validation check",
		BugFactIDs:     nil, // No bug facts
		Stake:          3000,
		Deposit:        10000,
	})
	if err == nil {
		t.Fatal("expected error when no bug facts referenced")
	}
}

func TestSubmitBugFix_ShortDescription(t *testing.T) {
	txClient := &bbMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.SubmitBugFix(context.Background(), BugFixRequest{
		Title:          "Fix",
		FixDescription: "too short",
		BugFactIDs:     []string{"fact-001"},
		Stake:          3000,
		Deposit:        10000,
	})
	if err == nil {
		t.Fatal("expected error for short description")
	}
}

func TestDistributeBounty(t *testing.T) {
	txClient := &bbMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	err := agent.DistributeBounty(context.Background(), BountyDistribution{
		BugFinders: []BountyFinder{
			{Address: "lgm1scanner1", Share: 4000},
			{Address: "lgm1scanner2", Share: 4000},
		},
		FixAuthor:  "lgm1fixer",
		FixShare:   8000,
		BugFactIDs: []string{"fact-001"},
		LIPID:      "lip-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 finder payouts + 1 fix author payout = 3 vestings.
	if len(txClient.vestings) != 3 {
		t.Fatalf("expected 3 vestings, got %d", len(txClient.vestings))
	}

	// Verify finder payouts.
	v0 := txClient.vestings[0]
	if v0.Beneficiary != "lgm1scanner1" {
		t.Fatalf("expected beneficiary lgm1scanner1, got %s", v0.Beneficiary)
	}
	if v0.Amount != 4000 {
		t.Fatalf("expected amount 4000, got %d", v0.Amount)
	}
	if v0.Source != "bounty_fulfillment" {
		t.Fatalf("expected source bounty_fulfillment, got %s", v0.Source)
	}

	// Verify fix author payout.
	v2 := txClient.vestings[2]
	if v2.Beneficiary != "lgm1fixer" {
		t.Fatalf("expected beneficiary lgm1fixer, got %s", v2.Beneficiary)
	}
	if v2.Amount != 8000 {
		t.Fatalf("expected amount 8000, got %d", v2.Amount)
	}
}

func TestDistributeBounty_SkipsZeroShare(t *testing.T) {
	txClient := &bbMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	err := agent.DistributeBounty(context.Background(), BountyDistribution{
		BugFinders: []BountyFinder{
			{Address: "lgm1scanner1", Share: 0},    // Should be skipped
			{Address: "lgm1scanner2", Share: 5000},
		},
		FixAuthor:  "lgm1fixer",
		FixShare:   5000,
		BugFactIDs: []string{"fact-001"},
		LIPID:      "lip-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only 2 vestings: scanner2 + fixer (scanner1 skipped due to 0 share).
	if len(txClient.vestings) != 2 {
		t.Fatalf("expected 2 vestings (skipping 0 share), got %d", len(txClient.vestings))
	}
}

func TestDistributeBounty_NoRecipients(t *testing.T) {
	txClient := &bbMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	err := agent.DistributeBounty(context.Background(), BountyDistribution{})
	if err == nil {
		t.Fatal("expected error when no recipients")
	}
}

func TestDistributeBounty_VestingError(t *testing.T) {
	txClient := &bbMockTxClient{vestingErr: fmt.Errorf("vesting failed")}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	err := agent.DistributeBounty(context.Background(), BountyDistribution{
		BugFinders: []BountyFinder{
			{Address: "lgm1scanner1", Share: 5000},
		},
		BugFactIDs: []string{"fact-001"},
		LIPID:      "lip-001",
	})
	if err == nil {
		t.Fatal("expected error when vesting fails")
	}
}

func TestQueryBugFacts(t *testing.T) {
	queryClient := &bbMockQueryClient{
		bugFacts: []FactInfo{
			{ID: "fact-001", ClaimText: "BUG: double slash", Category: "protocol_observation"},
			{ID: "fact-002", ClaimText: "BUG: nonce overflow", Category: "protocol_observation"},
		},
	}
	agent := New(DefaultConfig())
	agent.SetQueryClient(queryClient)

	facts, err := agent.QueryBugFacts(context.Background(), "protocol")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(facts) != 2 {
		t.Fatalf("expected 2 bug facts, got %d", len(facts))
	}
}

func TestQueryProposalsByTrigger(t *testing.T) {
	queryClient := &bbMockQueryClient{
		proposals: []ProposalInfo{
			{ID: "prop-001", Title: "LIP: Multiple verified findings", Status: "voting", TriggerType: "openclaw_critical"},
		},
	}
	agent := New(DefaultConfig())
	agent.SetQueryClient(queryClient)

	proposals, err := agent.QueryProposalsByTrigger(context.Background(), "openclaw_critical")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}
	if proposals[0].TriggerType != "openclaw_critical" {
		t.Fatalf("unexpected trigger type: %s", proposals[0].TriggerType)
	}
}

func TestAutoBugBounty_UnknownRole(t *testing.T) {
	agent := New(DefaultConfig())
	err := agent.AutoBugBounty(context.Background(), BountyRole("unknown"), nil)
	if err == nil {
		t.Fatal("expected error for unknown role")
	}
}

func TestAutoBugBounty_Scanner_Cancellation(t *testing.T) {
	queryClient := &bbMockQueryClient{}
	agent := New(DefaultConfig())
	agent.SetQueryClient(queryClient)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := agent.AutoBugBounty(ctx, BountyRoleScanner, nil)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
}

func TestAutoBugBounty_Fixer_Cancellation(t *testing.T) {
	queryClient := &bbMockQueryClient{}
	agent := New(DefaultConfig())
	agent.SetQueryClient(queryClient)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := agent.AutoBugBounty(ctx, BountyRoleFixer, nil)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
}

func TestAutoBugBounty_Guardian_Cancellation(t *testing.T) {
	queryClient := &bbMockQueryClient{}
	txClient := &bbMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetQueryClient(queryClient)
	agent.SetTxClient(txClient)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := agent.AutoBugBounty(ctx, BountyRoleGuardian, nil)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
}

// TestEndToEndBugBountyFlow tests the full lifecycle: report → fix → bounty.
func TestEndToEndBugBountyFlow(t *testing.T) {
	txClient := &bbMockTxClient{}

	queryClient := &bbMockQueryClient{
		bugFacts: []FactInfo{
			{ID: "fact-001", ClaimText: "BUG: double slash", Category: "protocol_observation", Confidence: 85},
		},
		proposals: []ProposalInfo{
			{ID: "prop-001", Title: "LIP: Fix double slash", Status: "passed", TriggerType: "openclaw_critical"},
		},
	}

	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)
	agent.SetQueryClient(queryClient)

	// Phase 1: Scanner reports a bug.
	reportTx, err := agent.SubmitBugReport(context.Background(),
		"BUG: validator slashing applies penalty twice in keeper.go:340 when unbonding is active",
		"protocol", 5000)
	if err != nil {
		t.Fatalf("Phase 1 (report) failed: %v", err)
	}
	if reportTx == "" {
		t.Fatal("Phase 1: expected non-empty tx hash")
	}

	// Phase 2: Verification happens via AutoVerify (tested separately).
	// Assume claim became fact-001.

	// Phase 3: Fixer queries verified bugs and submits fix.
	bugs, err := agent.QueryBugFacts(context.Background(), "protocol")
	if err != nil {
		t.Fatalf("Phase 3 (query bugs) failed: %v", err)
	}
	if len(bugs) == 0 {
		t.Fatal("Phase 3: expected at least 1 verified bug")
	}

	bugFactIDs := make([]string, len(bugs))
	for i, b := range bugs {
		bugFactIDs[i] = b.ID
	}

	lipTx, err := agent.SubmitBugFix(context.Background(), BugFixRequest{
		Title:          "Fix double slashing in validator keeper",
		FixDescription: "FIX: Added idempotency check in keeper.go:340 to prevent double-slash during unbonding",
		BugFactIDs:     bugFactIDs,
		Stake:          3000,
		Deposit:        10000,
	})
	if err != nil {
		t.Fatalf("Phase 3 (submit fix) failed: %v", err)
	}
	if lipTx == "" {
		t.Fatal("Phase 3: expected non-empty LIP tx hash")
	}

	// Phase 4: Guardian distributes bounty after LIP approval.
	err = agent.DistributeBounty(context.Background(), BountyDistribution{
		BugFinders: []BountyFinder{
			{Address: "lgm1scanner", Share: 4000},
		},
		FixAuthor:  "lgm1fixer",
		FixShare:   4000,
		BugFactIDs: bugFactIDs,
		LIPID:      "prop-001",
	})
	if err != nil {
		t.Fatalf("Phase 4 (distribute bounty) failed: %v", err)
	}

	// Verify total transactions: 1 bug report + 1 fix claim + 1 LIP + 2 vestings.
	if len(txClient.claims) != 2 {
		t.Fatalf("expected 2 total claims (report + fix), got %d", len(txClient.claims))
	}
	if len(txClient.lips) != 1 {
		t.Fatalf("expected 1 LIP, got %d", len(txClient.lips))
	}
	if len(txClient.vestings) != 2 {
		t.Fatalf("expected 2 vestings (finder + fixer), got %d", len(txClient.vestings))
	}
}
