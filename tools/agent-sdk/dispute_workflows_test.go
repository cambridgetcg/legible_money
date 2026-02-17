package agentsdk

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// --- Dispute Mock Clients ---

type disputeMockTxClient struct {
	disputes      []InitiateDisputeRequest
	commits       []string // commitment hashes
	reveals       []RevealDisputeEvidenceRequest
	votes         []ArbiterVoteRequest
	escalations   []string // dispute IDs
	disputeErr    error
	commitErr     error
	revealErr     error
	voteErr       error
	escalateErr   error
}

func (m *disputeMockTxClient) RegisterAccount(_ context.Context, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) RegisterAgent(_ context.Context, _ RegisterAgentRequest) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) Heartbeat(_ context.Context) (string, error) { return "mock-tx", nil }
func (m *disputeMockTxClient) SubmitClaim(_ context.Context, _ SubmitClaimRequest) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) SubmitCommitment(_ context.Context, _, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) SubmitReveal(_ context.Context, _ string, _ Verdict, _ uint32, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) ChallengeFact(_ context.Context, _, _ string, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) CreateHome(_ context.Context, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) UpdateMemoryCID(_ context.Context, _, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) Stake(_ context.Context, _ string, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) DeployContract(_ context.Context, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) CallContract(_ context.Context, _, _ string, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) ScheduleExecution(_ context.Context, _, _, _ string, _, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) ConfigureGuardian(_ context.Context, _, _ string, _ bool, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) RegisterHomeKey(_ context.Context, _, _, _ string, _ []string, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) StartSession(_ context.Context, _, _ string, _ []string) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) EndSession(_ context.Context, _, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) SetSpendingLimit(_ context.Context, _, _ string, _, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) SubmitLIP(_ context.Context, _ SubmitLIPRequest) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) CreateVesting(_ context.Context, _ CreateVestingRequest) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) InitiateDispute(_ context.Context, req InitiateDisputeRequest) (string, error) {
	if m.disputeErr != nil {
		return "", m.disputeErr
	}
	m.disputes = append(m.disputes, req)
	return fmt.Sprintf("dispute-tx-%d", len(m.disputes)), nil
}
func (m *disputeMockTxClient) CommitDisputeEvidence(_ context.Context, _, hash string) (string, error) {
	if m.commitErr != nil {
		return "", m.commitErr
	}
	m.commits = append(m.commits, hash)
	return "commit-tx", nil
}
func (m *disputeMockTxClient) RevealDisputeEvidence(_ context.Context, req RevealDisputeEvidenceRequest) (string, error) {
	if m.revealErr != nil {
		return "", m.revealErr
	}
	m.reveals = append(m.reveals, req)
	return "reveal-tx", nil
}
func (m *disputeMockTxClient) ArbiterVote(_ context.Context, req ArbiterVoteRequest) (string, error) {
	if m.voteErr != nil {
		return "", m.voteErr
	}
	m.votes = append(m.votes, req)
	return "vote-tx", nil
}
func (m *disputeMockTxClient) EscalateDispute(_ context.Context, disputeID string, _ uint64) (string, error) {
	if m.escalateErr != nil {
		return "", m.escalateErr
	}
	m.escalations = append(m.escalations, disputeID)
	return "escalate-tx", nil
}
func (m *disputeMockTxClient) SubmitEvidence(_ context.Context, _ SubmitEvidenceRequest) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) AuditEvidence(_ context.Context, _ AuditEvidenceRequest) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) RegisterOracle(_ context.Context, _ RegisterOracleRequest) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) SubmitOracleData(_ context.Context, _ SubmitOracleDataRequest) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) SubmitResearch(_ context.Context, _ SubmitResearchRequest) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) ReviewResearch(_ context.Context, _ ReviewResearchRequest) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) CreateResearchBounty(_ context.Context, _ CreateResearchBountyRequest) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) ClaimResearchBounty(_ context.Context, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) FundResearch(_ context.Context, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) ProposeOntologyDomain(_ context.Context, _ ProposeOntologyDomainRequest) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) VoteDomainProposal(_ context.Context, _ string, _ bool) (string, error) {
	return "mock-tx", nil
}
func (m *disputeMockTxClient) AcknowledgeIncompleteness(_ context.Context, _, _, _ string) (string, error) {
	return "mock-tx", nil
}

type disputeMockQueryClient struct {
	disputes    []DisputeInfo
	votes       []DisputeVoteInfo
	disputeErr  error
	votesErr    error
}

func (m *disputeMockQueryClient) ChainStatus(_ context.Context) (*ChainStatus, error) {
	return &ChainStatus{ChainID: ChainID, LatestHeight: 100}, nil
}
func (m *disputeMockQueryClient) Balance(_ context.Context, _ string) (uint64, error) {
	return 1000000, nil
}
func (m *disputeMockQueryClient) AgentProfile(_ context.Context, _ string) (*AgentProfileInfo, error) {
	return &AgentProfileInfo{}, nil
}
func (m *disputeMockQueryClient) ActiveRounds(_ context.Context) ([]RoundInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) RoundDetail(_ context.Context, _ string) (*RoundInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) FactsByDomain(_ context.Context, _ string) ([]FactInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) ClaimsBySubmitter(_ context.Context, _ string) ([]ClaimInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) GetClaim(_ context.Context, _ string) (*ClaimInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) FactReferences(_ context.Context, _ string) ([]FactReference, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) CounterFacts(_ context.Context, _ string) ([]CounterFactInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) BugFacts(_ context.Context, _ string) ([]FactInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) ProposalsByTrigger(_ context.Context, _ string) ([]ProposalInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) GetDispute(_ context.Context, id string) (*DisputeInfo, error) {
	if m.disputeErr != nil {
		return nil, m.disputeErr
	}
	for _, d := range m.disputes {
		if d.ID == id {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("dispute not found: %s", id)
}
func (m *disputeMockQueryClient) DisputesByFact(_ context.Context, factID string) ([]DisputeInfo, error) {
	var result []DisputeInfo
	for _, d := range m.disputes {
		if d.TargetID == factID {
			result = append(result, d)
		}
	}
	return result, nil
}
func (m *disputeMockQueryClient) ActiveDisputes(_ context.Context) ([]DisputeInfo, error) {
	if m.disputeErr != nil {
		return nil, m.disputeErr
	}
	return m.disputes, nil
}
func (m *disputeMockQueryClient) DisputeVotes(_ context.Context, _ string) ([]DisputeVoteInfo, error) {
	return m.votes, m.votesErr
}
func (m *disputeMockQueryClient) EvidenceByClaimID(_ context.Context, _ string) ([]EvidenceInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) ActiveOracles(_ context.Context) ([]OracleInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) GetResearch(_ context.Context, _ string) (*ResearchInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) ResearchSubmissions(_ context.Context, _, _ string) ([]ResearchInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) GetBounty(_ context.Context, _ string) (*ResearchBountyInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) AllDomains(_ context.Context) ([]DomainInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) DomainsByStratum(_ context.Context, _ uint32) ([]DomainInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) ConfidenceCeiling(_ context.Context, _ string) (*ConfidenceCeilingInfo, error) {
	return nil, nil
}
func (m *disputeMockQueryClient) AllLogicZones(_ context.Context) ([]LogicZoneInfo, error) {
	return nil, nil
}

// --- Mock Dispute Evaluator ---

type mockDisputeEvaluator struct {
	vote      DisputeVote
	reasoning string
	err       error
}

func (m *mockDisputeEvaluator) EvaluateDispute(_ context.Context, _ string) (DisputeVote, string, error) {
	return m.vote, m.reasoning, m.err
}

// --- Dispute Workflow Tests ---

func TestDisputeFact(t *testing.T) {
	txClient := &disputeMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	txHash, err := agent.DisputeFact(context.Background(),
		"fact-123", "Double slashing during unbonding", 5000,
		[]string{"evidence-item-1", "evidence-item-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txHash != "dispute-tx-1" {
		t.Fatalf("unexpected tx hash: %s", txHash)
	}
	if len(txClient.disputes) != 1 {
		t.Fatalf("expected 1 dispute, got %d", len(txClient.disputes))
	}
	if txClient.disputes[0].TargetID != "fact-123" {
		t.Fatalf("unexpected target: %s", txClient.disputes[0].TargetID)
	}
	if len(txClient.commits) != 1 {
		t.Fatalf("expected 1 evidence commit, got %d", len(txClient.commits))
	}
	if len(txClient.reveals) != 1 {
		t.Fatalf("expected 1 evidence reveal, got %d", len(txClient.reveals))
	}
	if len(txClient.reveals[0].Evidence) != 2 {
		t.Fatalf("expected 2 evidence items, got %d", len(txClient.reveals[0].Evidence))
	}
}

func TestDisputeFact_NoEvidence(t *testing.T) {
	txClient := &disputeMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	txHash, err := agent.DisputeFact(context.Background(),
		"fact-456", "Invalid methodology", 3000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txHash != "dispute-tx-1" {
		t.Fatalf("unexpected tx hash: %s", txHash)
	}
	// No evidence submitted — only initiate.
	if len(txClient.commits) != 0 {
		t.Fatalf("expected no evidence commits, got %d", len(txClient.commits))
	}
}

func TestDisputeFact_EmptyReason(t *testing.T) {
	txClient := &disputeMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.DisputeFact(context.Background(), "fact-789", "", 5000, nil)
	if err == nil {
		t.Fatal("expected error for empty reason")
	}
}

func TestDisputeFact_ZeroStake(t *testing.T) {
	txClient := &disputeMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.DisputeFact(context.Background(), "fact-789", "reason", 0, nil)
	if err == nil {
		t.Fatal("expected error for zero stake")
	}
}

func TestDisputeFact_NoClient(t *testing.T) {
	agent := New(DefaultConfig())
	_, err := agent.DisputeFact(context.Background(), "fact-789", "reason", 5000, nil)
	if err == nil {
		t.Fatal("expected error for missing tx client")
	}
}

func TestDisputeFact_InitiateError(t *testing.T) {
	txClient := &disputeMockTxClient{disputeErr: fmt.Errorf("dispute failed")}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.DisputeFact(context.Background(), "fact-abc", "reason", 5000, nil)
	if err == nil {
		t.Fatal("expected error from initiate")
	}
}

func TestVoteOnDispute(t *testing.T) {
	txClient := &disputeMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	txHash, err := agent.VoteOnDispute(context.Background(),
		"dispute-1", DisputeVoteChallenger, "Evidence clearly shows the bug exists")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txHash != "vote-tx" {
		t.Fatalf("unexpected tx hash: %s", txHash)
	}
	if len(txClient.votes) != 1 {
		t.Fatalf("expected 1 vote, got %d", len(txClient.votes))
	}
	if txClient.votes[0].Vote != DisputeVoteChallenger {
		t.Fatalf("unexpected vote: %s", txClient.votes[0].Vote)
	}
}

func TestVoteOnDispute_EmptyReasoning(t *testing.T) {
	txClient := &disputeMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.VoteOnDispute(context.Background(), "dispute-1", DisputeVoteDefender, "")
	if err == nil {
		t.Fatal("expected error for empty reasoning")
	}
}

func TestQueryActiveDisputes(t *testing.T) {
	qc := &disputeMockQueryClient{
		disputes: []DisputeInfo{
			{ID: "d1", Status: "voting", TargetID: "fact-1"},
			{ID: "d2", Status: "evidence", TargetID: "fact-2"},
		},
	}
	agent := New(DefaultConfig())
	agent.SetQueryClient(qc)

	disputes, err := agent.QueryActiveDisputes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(disputes) != 2 {
		t.Fatalf("expected 2 disputes, got %d", len(disputes))
	}
}

func TestQueryDisputesByFact(t *testing.T) {
	qc := &disputeMockQueryClient{
		disputes: []DisputeInfo{
			{ID: "d1", TargetID: "fact-X"},
			{ID: "d2", TargetID: "fact-Y"},
			{ID: "d3", TargetID: "fact-X"},
		},
	}
	agent := New(DefaultConfig())
	agent.SetQueryClient(qc)

	disputes, err := agent.QueryDisputesByFact(context.Background(), "fact-X")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(disputes) != 2 {
		t.Fatalf("expected 2 disputes for fact-X, got %d", len(disputes))
	}
}

func TestAutoArbiter_Cancellation(t *testing.T) {
	txClient := &disputeMockTxClient{}
	qc := &disputeMockQueryClient{
		disputes: []DisputeInfo{
			{ID: "d1", Status: "voting"},
		},
	}
	evaluator := &mockDisputeEvaluator{
		vote:      DisputeVoteChallenger,
		reasoning: "Bug is valid",
	}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)
	agent.SetQueryClient(qc)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := agent.AutoArbiter(ctx, evaluator)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
	// Should have voted on the dispute
	if len(txClient.votes) == 0 {
		t.Fatal("expected at least 1 vote from auto-arbiter")
	}
}

func TestAutoArbiter_SkipsNonVoting(t *testing.T) {
	txClient := &disputeMockTxClient{}
	qc := &disputeMockQueryClient{
		disputes: []DisputeInfo{
			{ID: "d1", Status: "evidence"},   // not voting
			{ID: "d2", Status: "resolved"},   // not voting
		},
	}
	evaluator := &mockDisputeEvaluator{
		vote:      DisputeVoteDefender,
		reasoning: "Insufficient evidence",
	}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)
	agent.SetQueryClient(qc)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = agent.AutoArbiter(ctx, evaluator)
	if len(txClient.votes) != 0 {
		t.Fatalf("expected no votes for non-voting disputes, got %d", len(txClient.votes))
	}
}

func TestEvidenceCommitmentDeterminism(t *testing.T) {
	evidence := []string{"item1", "item2"}
	nonce := "abc123"

	c1 := computeEvidenceCommitment(evidence, nonce)
	c2 := computeEvidenceCommitment(evidence, nonce)
	if c1 != c2 {
		t.Fatal("evidence commitment not deterministic")
	}

	// Different nonce should produce different commitment
	c3 := computeEvidenceCommitment(evidence, "different")
	if c1 == c3 {
		t.Fatal("different nonces produced same commitment")
	}

	// Different evidence should produce different commitment
	c4 := computeEvidenceCommitment([]string{"item1", "item3"}, nonce)
	if c1 == c4 {
		t.Fatal("different evidence produced same commitment")
	}
}

func TestDisputeVoteConstants(t *testing.T) {
	if DisputeVoteChallenger != "CHALLENGER" {
		t.Fatalf("wrong challenger constant: %s", DisputeVoteChallenger)
	}
	if DisputeVoteDefender != "DEFENDER" {
		t.Fatalf("wrong defender constant: %s", DisputeVoteDefender)
	}
	if DisputeVoteAbstain != "ABSTAIN" {
		t.Fatalf("wrong abstain constant: %s", DisputeVoteAbstain)
	}
}
