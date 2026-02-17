package agentsdk

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// --- Research Mock Clients ---

type resMockTxClient struct {
	researches []SubmitResearchRequest
	reviews    []ReviewResearchRequest
	bounties   []CreateResearchBountyRequest
	evidences  []SubmitEvidenceRequest
	domains    []ProposeOntologyDomainRequest
	goedels    []string // factIDs acknowledged
	resErr     error
	reviewErr  error
	bountyErr  error
	evidErr    error
	domainErr  error
	goedelErr  error
}

func (m *resMockTxClient) RegisterAccount(_ context.Context, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) RegisterAgent(_ context.Context, _ RegisterAgentRequest) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) Heartbeat(_ context.Context) (string, error) { return "mock-tx", nil }
func (m *resMockTxClient) SubmitClaim(_ context.Context, _ SubmitClaimRequest) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) SubmitCommitment(_ context.Context, _, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) SubmitReveal(_ context.Context, _ string, _ Verdict, _ uint32, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) ChallengeFact(_ context.Context, _, _ string, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) CreateHome(_ context.Context, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) UpdateMemoryCID(_ context.Context, _, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) Stake(_ context.Context, _ string, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) DeployContract(_ context.Context, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) CallContract(_ context.Context, _, _ string, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) ScheduleExecution(_ context.Context, _, _, _ string, _, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) ConfigureGuardian(_ context.Context, _, _ string, _ bool, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) RegisterHomeKey(_ context.Context, _, _, _ string, _ []string, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) StartSession(_ context.Context, _, _ string, _ []string) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) EndSession(_ context.Context, _, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) SetSpendingLimit(_ context.Context, _, _ string, _, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) SubmitLIP(_ context.Context, _ SubmitLIPRequest) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) CreateVesting(_ context.Context, _ CreateVestingRequest) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) InitiateDispute(_ context.Context, _ InitiateDisputeRequest) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) CommitDisputeEvidence(_ context.Context, _, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) RevealDisputeEvidence(_ context.Context, _ RevealDisputeEvidenceRequest) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) ArbiterVote(_ context.Context, _ ArbiterVoteRequest) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) EscalateDispute(_ context.Context, _ string, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) SubmitEvidence(_ context.Context, req SubmitEvidenceRequest) (string, error) {
	if m.evidErr != nil {
		return "", m.evidErr
	}
	m.evidences = append(m.evidences, req)
	return fmt.Sprintf("evidence-tx-%d", len(m.evidences)), nil
}
func (m *resMockTxClient) AuditEvidence(_ context.Context, _ AuditEvidenceRequest) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) RegisterOracle(_ context.Context, _ RegisterOracleRequest) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) SubmitOracleData(_ context.Context, _ SubmitOracleDataRequest) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) SubmitResearch(_ context.Context, req SubmitResearchRequest) (string, error) {
	if m.resErr != nil {
		return "", m.resErr
	}
	m.researches = append(m.researches, req)
	return fmt.Sprintf("research-tx-%d", len(m.researches)), nil
}
func (m *resMockTxClient) ReviewResearch(_ context.Context, req ReviewResearchRequest) (string, error) {
	if m.reviewErr != nil {
		return "", m.reviewErr
	}
	m.reviews = append(m.reviews, req)
	return fmt.Sprintf("review-tx-%d", len(m.reviews)), nil
}
func (m *resMockTxClient) CreateResearchBounty(_ context.Context, req CreateResearchBountyRequest) (string, error) {
	if m.bountyErr != nil {
		return "", m.bountyErr
	}
	m.bounties = append(m.bounties, req)
	return fmt.Sprintf("bounty-tx-%d", len(m.bounties)), nil
}
func (m *resMockTxClient) ClaimResearchBounty(_ context.Context, _ string) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) FundResearch(_ context.Context, _ uint64) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) ProposeOntologyDomain(_ context.Context, req ProposeOntologyDomainRequest) (string, error) {
	if m.domainErr != nil {
		return "", m.domainErr
	}
	m.domains = append(m.domains, req)
	return fmt.Sprintf("domain-tx-%d", len(m.domains)), nil
}
func (m *resMockTxClient) VoteDomainProposal(_ context.Context, _ string, _ bool) (string, error) {
	return "mock-tx", nil
}
func (m *resMockTxClient) AcknowledgeIncompleteness(_ context.Context, factID, _, _ string) (string, error) {
	if m.goedelErr != nil {
		return "", m.goedelErr
	}
	m.goedels = append(m.goedels, factID)
	return fmt.Sprintf("goedel-tx-%d", len(m.goedels)), nil
}

type resMockQueryClient struct {
	submissions []ResearchInfo
	bounty      *ResearchBountyInfo
	domains     []DomainInfo
	ceiling     *ConfidenceCeilingInfo
	zones       []LogicZoneInfo
	subErr      error
}

func (m *resMockQueryClient) ChainStatus(_ context.Context) (*ChainStatus, error) {
	return &ChainStatus{ChainID: ChainID, LatestHeight: 100}, nil
}
func (m *resMockQueryClient) Balance(_ context.Context, _ string) (uint64, error) {
	return 1000000, nil
}
func (m *resMockQueryClient) AgentProfile(_ context.Context, _ string) (*AgentProfileInfo, error) {
	return &AgentProfileInfo{}, nil
}
func (m *resMockQueryClient) ActiveRounds(_ context.Context) ([]RoundInfo, error) { return nil, nil }
func (m *resMockQueryClient) RoundDetail(_ context.Context, _ string) (*RoundInfo, error) {
	return nil, nil
}
func (m *resMockQueryClient) FactsByDomain(_ context.Context, _ string) ([]FactInfo, error) {
	return nil, nil
}
func (m *resMockQueryClient) ClaimsBySubmitter(_ context.Context, _ string) ([]ClaimInfo, error) {
	return nil, nil
}
func (m *resMockQueryClient) GetClaim(_ context.Context, _ string) (*ClaimInfo, error) {
	return nil, nil
}
func (m *resMockQueryClient) FactReferences(_ context.Context, _ string) ([]FactReference, error) {
	return nil, nil
}
func (m *resMockQueryClient) CounterFacts(_ context.Context, _ string) ([]CounterFactInfo, error) {
	return nil, nil
}
func (m *resMockQueryClient) BugFacts(_ context.Context, _ string) ([]FactInfo, error) {
	return nil, nil
}
func (m *resMockQueryClient) ProposalsByTrigger(_ context.Context, _ string) ([]ProposalInfo, error) {
	return nil, nil
}
func (m *resMockQueryClient) GetDispute(_ context.Context, _ string) (*DisputeInfo, error) {
	return nil, nil
}
func (m *resMockQueryClient) DisputesByFact(_ context.Context, _ string) ([]DisputeInfo, error) {
	return nil, nil
}
func (m *resMockQueryClient) ActiveDisputes(_ context.Context) ([]DisputeInfo, error) {
	return nil, nil
}
func (m *resMockQueryClient) DisputeVotes(_ context.Context, _ string) ([]DisputeVoteInfo, error) {
	return nil, nil
}
func (m *resMockQueryClient) EvidenceByClaimID(_ context.Context, _ string) ([]EvidenceInfo, error) {
	return nil, nil
}
func (m *resMockQueryClient) ActiveOracles(_ context.Context) ([]OracleInfo, error) {
	return nil, nil
}
func (m *resMockQueryClient) GetResearch(_ context.Context, id string) (*ResearchInfo, error) {
	for _, s := range m.submissions {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("research not found: %s", id)
}
func (m *resMockQueryClient) ResearchSubmissions(_ context.Context, _, _ string) ([]ResearchInfo, error) {
	return m.submissions, m.subErr
}
func (m *resMockQueryClient) GetBounty(_ context.Context, _ string) (*ResearchBountyInfo, error) {
	if m.bounty == nil {
		return nil, fmt.Errorf("bounty not found")
	}
	return m.bounty, nil
}
func (m *resMockQueryClient) AllDomains(_ context.Context) ([]DomainInfo, error) {
	return m.domains, nil
}
func (m *resMockQueryClient) DomainsByStratum(_ context.Context, s uint32) ([]DomainInfo, error) {
	var result []DomainInfo
	for _, d := range m.domains {
		if d.Stratum == s {
			result = append(result, d)
		}
	}
	return result, nil
}
func (m *resMockQueryClient) ConfidenceCeiling(_ context.Context, _ string) (*ConfidenceCeilingInfo, error) {
	if m.ceiling == nil {
		return nil, fmt.Errorf("ceiling not found")
	}
	return m.ceiling, nil
}
func (m *resMockQueryClient) AllLogicZones(_ context.Context) ([]LogicZoneInfo, error) {
	return m.zones, nil
}

// --- Mock Research Evaluator ---

type mockResearchEvaluator struct {
	verdict   ReviewVerdict
	score     uint32
	reasoning string
	err       error
}

func (m *mockResearchEvaluator) EvaluateResearch(_ context.Context, _ string) (ReviewVerdict, uint32, string, error) {
	return m.verdict, m.score, m.reasoning, m.err
}

// --- Research Workflow Tests ---

func TestSubmitResearchWithEvidence(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	txHash, err := agent.SubmitResearchWithEvidence(context.Background(),
		SubmitResearchRequest{
			Title:       "Replication: Double Slashing Bug",
			Description: "We replicated the double-slashing bug reported in fact-123 using the following methodology...",
			Domain:      "protocol",
			Stake:       10000,
		},
		[]SubmitEvidenceRequest{
			{ClaimID: "claim-1", EvidenceType: "computation", ContentHash: "abc123", Description: "Automated reproduction"},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txHash != "research-tx-1" {
		t.Fatalf("unexpected tx hash: %s", txHash)
	}
	if len(txClient.researches) != 1 {
		t.Fatalf("expected 1 research, got %d", len(txClient.researches))
	}
	if len(txClient.evidences) != 1 {
		t.Fatalf("expected 1 evidence, got %d", len(txClient.evidences))
	}
}

func TestSubmitResearchWithEvidence_NoEvidence(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.SubmitResearchWithEvidence(context.Background(),
		SubmitResearchRequest{
			Title:       "Methodology Audit",
			Description: "Comprehensive audit of the validator selection methodology for fairness...",
			Domain:      "protocol",
			Stake:       5000,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txClient.evidences) != 0 {
		t.Fatalf("expected no evidence, got %d", len(txClient.evidences))
	}
}

func TestSubmitResearchWithEvidence_ShortDescription(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.SubmitResearchWithEvidence(context.Background(),
		SubmitResearchRequest{Title: "Test", Description: "short", Domain: "test", Stake: 5000},
		nil,
	)
	if err == nil {
		t.Fatal("expected error for short description")
	}
}

func TestSubmitResearchWithEvidence_EmptyTitle(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.SubmitResearchWithEvidence(context.Background(),
		SubmitResearchRequest{Title: "", Description: "long enough description text here", Domain: "test", Stake: 5000},
		nil,
	)
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestPeerReview(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	txHash, err := agent.PeerReview(context.Background(), ReviewResearchRequest{
		ResearchID:   "research-1",
		Verdict:      ReviewVerdictApprove,
		QualityScore: 85,
		Reasoning:    "Methodology is sound, results are reproducible",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txHash != "review-tx-1" {
		t.Fatalf("unexpected tx hash: %s", txHash)
	}
	if len(txClient.reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(txClient.reviews))
	}
	if txClient.reviews[0].Verdict != ReviewVerdictApprove {
		t.Fatalf("unexpected verdict: %s", txClient.reviews[0].Verdict)
	}
}

func TestPeerReview_InvalidScore(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.PeerReview(context.Background(), ReviewResearchRequest{
		ResearchID:   "research-1",
		Verdict:      ReviewVerdictApprove,
		QualityScore: 150, // > 100
		Reasoning:    "Good research",
	})
	if err == nil {
		t.Fatal("expected error for quality score > 100")
	}
}

func TestPeerReview_EmptyReasoning(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.PeerReview(context.Background(), ReviewResearchRequest{
		ResearchID:   "research-1",
		Verdict:      ReviewVerdictReject,
		QualityScore: 20,
		Reasoning:    "",
	})
	if err == nil {
		t.Fatal("expected error for empty reasoning")
	}
}

func TestCreateAndMonitorBounty(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	txHash, err := agent.CreateAndMonitorBounty(context.Background(), CreateResearchBountyRequest{
		Title:          "Reproduce VRF Selection Bias",
		Description:    "Reproduce and quantify the reported VRF selection bias...",
		Reward:         50000,
		DeadlineHeight: 100000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txHash != "bounty-tx-1" {
		t.Fatalf("unexpected tx hash: %s", txHash)
	}
}

func TestCreateAndMonitorBounty_ZeroReward(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.CreateAndMonitorBounty(context.Background(), CreateResearchBountyRequest{
		Title:          "Test",
		Description:    "Test bounty",
		Reward:         0,
		DeadlineHeight: 100000,
	})
	if err == nil {
		t.Fatal("expected error for zero reward")
	}
}

func TestSubmitSupportingEvidence(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	txHash, err := agent.SubmitSupportingEvidence(context.Background(), SubmitEvidenceRequest{
		ClaimID:      "claim-42",
		EvidenceType: "computation",
		ContentHash:  "sha256:abc123def456",
		Description:  "Automated test reproduction",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txHash != "evidence-tx-1" {
		t.Fatalf("unexpected tx hash: %s", txHash)
	}
}

func TestSubmitSupportingEvidence_MissingFields(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	// Missing claim ID
	_, err := agent.SubmitSupportingEvidence(context.Background(), SubmitEvidenceRequest{
		EvidenceType: "computation",
		ContentHash:  "hash",
		Description:  "test",
	})
	if err == nil {
		t.Fatal("expected error for missing claim ID")
	}

	// Missing content hash
	_, err = agent.SubmitSupportingEvidence(context.Background(), SubmitEvidenceRequest{
		ClaimID:      "claim-1",
		EvidenceType: "computation",
		Description:  "test",
	})
	if err == nil {
		t.Fatal("expected error for missing content hash")
	}

	// Missing evidence type
	_, err = agent.SubmitSupportingEvidence(context.Background(), SubmitEvidenceRequest{
		ClaimID:     "claim-1",
		ContentHash: "hash",
		Description: "test",
	})
	if err == nil {
		t.Fatal("expected error for missing evidence type")
	}
}

// --- Ontology Workflow Tests ---

func TestProposeNewDomain(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	txHash, err := agent.ProposeNewDomain(context.Background(), ProposeOntologyDomainRequest{
		Name:        "quantum_computing",
		DisplayName: "Quantum Computing",
		Description: "Knowledge domain for quantum computing research and protocols",
		Stratum:     3,
		Stake:       10000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txHash != "domain-tx-1" {
		t.Fatalf("unexpected tx hash: %s", txHash)
	}
	if len(txClient.domains) != 1 {
		t.Fatalf("expected 1 domain proposal, got %d", len(txClient.domains))
	}
}

func TestProposeNewDomain_InvalidStratum(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.ProposeNewDomain(context.Background(), ProposeOntologyDomainRequest{
		Name: "test", DisplayName: "Test", Stratum: 7, Stake: 1000,
	})
	if err == nil {
		t.Fatal("expected error for stratum > 6")
	}
}

func TestProposeNewDomain_ZeroStake(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.ProposeNewDomain(context.Background(), ProposeOntologyDomainRequest{
		Name: "test", DisplayName: "Test", Stratum: 3, Stake: 0,
	})
	if err == nil {
		t.Fatal("expected error for zero stake")
	}
}

func TestAcknowledgeGodelianIncompleteness(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	txHash, err := agent.AcknowledgeGodelianIncompleteness(context.Background(),
		"fact-goedel-1", "peano",
		"This arithmetic statement is true but unprovable within Peano arithmetic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txHash != "goedel-tx-1" {
		t.Fatalf("unexpected tx hash: %s", txHash)
	}
	if len(txClient.goedels) != 1 {
		t.Fatalf("expected 1 acknowledgment, got %d", len(txClient.goedels))
	}
}

func TestAcknowledgeGodelianIncompleteness_MissingFields(t *testing.T) {
	txClient := &resMockTxClient{}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)

	_, err := agent.AcknowledgeGodelianIncompleteness(context.Background(), "", "peano", "reason")
	if err == nil {
		t.Fatal("expected error for empty fact ID")
	}

	_, err = agent.AcknowledgeGodelianIncompleteness(context.Background(), "fact-1", "", "reason")
	if err == nil {
		t.Fatal("expected error for empty zone")
	}

	_, err = agent.AcknowledgeGodelianIncompleteness(context.Background(), "fact-1", "peano", "")
	if err == nil {
		t.Fatal("expected error for empty reason")
	}
}

func TestQueryDomainConfidence(t *testing.T) {
	qc := &resMockQueryClient{
		ceiling: &ConfidenceCeilingInfo{
			Domain:        "protocol",
			MaxConfidence: 950000,
			DecayRateBPS:  100,
		},
	}
	agent := New(DefaultConfig())
	agent.SetQueryClient(qc)

	ceiling, err := agent.QueryDomainConfidence(context.Background(), "protocol")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ceiling.MaxConfidence != 950000 {
		t.Fatalf("unexpected max confidence: %d", ceiling.MaxConfidence)
	}
}

func TestQueryLogicZones(t *testing.T) {
	qc := &resMockQueryClient{
		zones: []LogicZoneInfo{
			{Zone: "propositional", Complete: true, Decidable: true, GoedelApplies: false, MaxConfidence: 1000000},
			{Zone: "peano", Complete: false, Decidable: false, GoedelApplies: true, MaxConfidence: 800000},
		},
	}
	agent := New(DefaultConfig())
	agent.SetQueryClient(qc)

	zones, err := agent.QueryLogicZones(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(zones) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(zones))
	}
	// Peano should have Gödel applies
	if !zones[1].GoedelApplies {
		t.Fatal("expected Gödel applies for Peano")
	}
}

func TestAutoResearchReviewer_Cancellation(t *testing.T) {
	txClient := &resMockTxClient{}
	qc := &resMockQueryClient{
		submissions: []ResearchInfo{
			{ID: "res-1", Status: "under_review", Title: "Test Research"},
		},
	}
	evaluator := &mockResearchEvaluator{
		verdict:   ReviewVerdictApprove,
		score:     80,
		reasoning: "Sound methodology",
	}
	agent := New(DefaultConfig())
	agent.SetTxClient(txClient)
	agent.SetQueryClient(qc)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := agent.AutoResearchReviewer(ctx, evaluator)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
	if len(txClient.reviews) == 0 {
		t.Fatal("expected at least 1 review from auto-reviewer")
	}
}

func TestReviewVerdictConstants(t *testing.T) {
	if ReviewVerdictApprove != "APPROVE" {
		t.Fatalf("wrong approve constant: %s", ReviewVerdictApprove)
	}
	if ReviewVerdictReject != "REJECT" {
		t.Fatalf("wrong reject constant: %s", ReviewVerdictReject)
	}
	if ReviewVerdictRevise != "REVISE" {
		t.Fatalf("wrong revise constant: %s", ReviewVerdictRevise)
	}
}
