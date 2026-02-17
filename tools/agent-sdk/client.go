package agentsdk

import (
	"context"
	"fmt"
)

// QueryClient defines the interface for querying chain state.
// Implementations wrap legbled's gRPC client or CLI output.
type QueryClient interface {
	// ChainStatus returns current block height and chain info.
	ChainStatus(ctx context.Context) (*ChainStatus, error)

	// Balance returns the ulgm balance for an address.
	Balance(ctx context.Context, address string) (uint64, error)

	// AgentProfile returns the discovery profile for an address.
	AgentProfile(ctx context.Context, address string) (*AgentProfileInfo, error)

	// ActiveRounds returns verification rounds in commit or reveal phase.
	ActiveRounds(ctx context.Context) ([]RoundInfo, error)

	// RoundDetail returns details of a specific verification round.
	RoundDetail(ctx context.Context, roundID string) (*RoundInfo, error)

	// FactsByDomain returns accepted facts in a domain.
	FactsByDomain(ctx context.Context, domain string) ([]FactInfo, error)

	// ClaimsBySubmitter returns claims submitted by an address.
	ClaimsBySubmitter(ctx context.Context, address string) ([]ClaimInfo, error)

	// GetClaim returns details of a specific claim by ID.
	GetClaim(ctx context.Context, claimID string) (*ClaimInfo, error)

	// FactReferences returns reference edges for a fact.
	FactReferences(ctx context.Context, factID string) ([]FactReference, error)

	// CounterFacts returns counter-facts targeting a fact.
	CounterFacts(ctx context.Context, factID string) ([]CounterFactInfo, error)

	// BugFacts returns verified protocol_observation facts for a domain.
	BugFacts(ctx context.Context, domain string) ([]FactInfo, error)

	// ProposalsByTrigger returns governance proposals by trigger type.
	ProposalsByTrigger(ctx context.Context, triggerType string) ([]ProposalInfo, error)

	// --- Dispute Queries (x/disputes) ---

	// GetDispute returns details of a specific dispute.
	GetDispute(ctx context.Context, disputeID string) (*DisputeInfo, error)

	// DisputesByFact returns all disputes targeting a fact.
	DisputesByFact(ctx context.Context, factID string) ([]DisputeInfo, error)

	// ActiveDisputes returns all non-resolved disputes.
	ActiveDisputes(ctx context.Context) ([]DisputeInfo, error)

	// DisputeVotes returns votes cast on a dispute.
	DisputeVotes(ctx context.Context, disputeID string) ([]DisputeVoteInfo, error)

	// --- Evidence Queries (x/evidence_mgmt) ---

	// EvidenceByClaimID returns all evidence submitted for a claim.
	EvidenceByClaimID(ctx context.Context, claimID string) ([]EvidenceInfo, error)

	// ActiveOracles returns all active oracle feeds.
	ActiveOracles(ctx context.Context) ([]OracleInfo, error)

	// --- Research Queries (x/research) ---

	// GetResearch returns details of a research submission.
	GetResearch(ctx context.Context, researchID string) (*ResearchInfo, error)

	// ResearchSubmissions returns research filtered by status and/or domain.
	ResearchSubmissions(ctx context.Context, status, domain string) ([]ResearchInfo, error)

	// GetBounty returns details of a research bounty.
	GetBounty(ctx context.Context, bountyID string) (*ResearchBountyInfo, error)

	// --- Ontology Queries (x/ontology) ---

	// AllDomains returns all registered knowledge domains.
	AllDomains(ctx context.Context) ([]DomainInfo, error)

	// DomainsByStratum returns domains in a specific stratum.
	DomainsByStratum(ctx context.Context, stratum uint32) ([]DomainInfo, error)

	// ConfidenceCeiling returns the max confidence and decay rate for a domain.
	ConfidenceCeiling(ctx context.Context, domain string) (*ConfidenceCeilingInfo, error)

	// AllLogicZones returns all registered logic zones.
	AllLogicZones(ctx context.Context) ([]LogicZoneInfo, error)
}

// TxClient defines the interface for submitting transactions.
type TxClient interface {
	// RegisterAccount registers the agent's account on-chain (x/auth).
	RegisterAccount(ctx context.Context, pubKeyHex string) (string, error)

	// RegisterAgent registers the agent profile (x/discovery).
	RegisterAgent(ctx context.Context, req RegisterAgentRequest) (string, error)

	// Heartbeat sends a liveness heartbeat (x/discovery).
	Heartbeat(ctx context.Context) (string, error)

	// SubmitClaim submits a knowledge claim (x/knowledge).
	SubmitClaim(ctx context.Context, req SubmitClaimRequest) (string, error)

	// SubmitCommitment submits a verification commitment (x/knowledge).
	SubmitCommitment(ctx context.Context, roundID, commitment string) (string, error)

	// SubmitReveal submits a verification reveal (x/knowledge).
	SubmitReveal(ctx context.Context, roundID string, verdict Verdict, confidence uint32, salt string) (string, error)

	// ChallengeFact challenges an existing fact (x/knowledge).
	ChallengeFact(ctx context.Context, factID, reason string, stake uint64) (string, error)

	// CreateHome creates an Agent Home (x/home).
	CreateHome(ctx context.Context, name string) (string, error)

	// UpdateMemoryCID updates the Agent Home's IPFS memory pointer (x/home).
	UpdateMemoryCID(ctx context.Context, homeID, cid string) (string, error)

	// Stake delegates tokens to a validator (x/staking).
	Stake(ctx context.Context, validatorAddr string, amount uint64) (string, error)

	// DeployContract deploys BVM bytecode and returns the contract address (x/bvm).
	DeployContract(ctx context.Context, bytecodeHex string) (string, error)

	// CallContract calls a deployed BVM contract (x/bvm).
	CallContract(ctx context.Context, contractAddr, inputDataHex string, gasLimit uint64) (string, error)

	// ScheduleExecution schedules a BVM contract call at a future block (x/bvm).
	ScheduleExecution(ctx context.Context, contractAddr, method, payload string, atBlock, maxGas uint64) (string, error)

	// ConfigureGuardian configures the Agent Home's guardian settings (x/home).
	ConfigureGuardian(ctx context.Context, homeID, strategy string, autoDefend bool, deadmanThreshold uint64) (string, error)

	// RegisterHomeKey registers an access key on an Agent Home (x/home).
	RegisterHomeKey(ctx context.Context, homeID, keyHash, role string, permissions []string, expiresAt uint64) (string, error)

	// StartSession starts a session on an Agent Home (x/home).
	StartSession(ctx context.Context, homeID, keyHash string, permissions []string) (string, error)

	// EndSession ends an active session on an Agent Home (x/home).
	EndSession(ctx context.Context, homeID, sessionID string) (string, error)

	// SetSpendingLimit sets a spending limit on an Agent Home key type (x/home).
	SetSpendingLimit(ctx context.Context, homeID, keyType string, maxAmount, periodBlocks uint64) (string, error)

	// SubmitLIP submits a governance proposal linking a fix to bug facts (x/gov).
	SubmitLIP(ctx context.Context, req SubmitLIPRequest) (string, error)

	// CreateVesting creates a vesting reward entry for bounty payouts (x/vesting_rewards).
	CreateVesting(ctx context.Context, req CreateVestingRequest) (string, error)
}

// --- Response Types ---

// ChainStatus holds chain state information.
type ChainStatus struct {
	ChainID        string `json:"chain_id"`
	LatestHeight   int64  `json:"latest_height"`
	LatestTime     string `json:"latest_time"`
	CatchingUp     bool   `json:"catching_up"`
	ValidatorCount int    `json:"validator_count"`
}

// AgentProfileInfo holds discovery profile data.
type AgentProfileInfo struct {
	Address       string   `json:"address"`
	DID           string   `json:"did"`
	DisplayName   string   `json:"display_name"`
	Status        string   `json:"status"`
	Reputation    uint32   `json:"reputation_score"`
	Domains       []string `json:"domains"`
	Stake         string   `json:"stake"`
	LastActive    uint64   `json:"last_active_block"`
	RegisteredAt  uint64   `json:"registered_at_block"`
}

// RoundInfo holds verification round data.
type RoundInfo struct {
	ID            string `json:"id"`
	ClaimID       string `json:"claim_id"`
	Phase         string `json:"phase"` // "commit", "reveal", "aggregation", "complete"
	VerifierCount int    `json:"verifier_count"`
	StartBlock    uint64 `json:"start_block"`
	PhaseEndBlock uint64 `json:"phase_end_block"`
}

// FactInfo holds accepted fact data.
type FactInfo struct {
	ID            string   `json:"id"`
	ClaimText     string   `json:"claim_text"`
	Domain        string   `json:"domain"`
	Category      string   `json:"category"`
	Confidence    uint32   `json:"confidence"`
	SubmittedBy   string   `json:"submitted_by"`
	Maturity      string   `json:"maturity,omitempty"`       // emerging/established/canonical
	CitationCount uint32   `json:"citation_count,omitempty"`
	References    []string `json:"references,omitempty"`     // outgoing fact IDs
	StratumCeiling uint32  `json:"stratum_ceiling,omitempty"`
	NoveltyScore  uint32   `json:"novelty_score,omitempty"`
	LogicZone     string   `json:"logic_zone,omitempty"`
	Status        string   `json:"status,omitempty"`
	EvidenceBonus uint32   `json:"evidence_bonus,omitempty"`
}

// FactReference represents a typed edge between two facts.
type FactReference struct {
	FromFactID string `json:"from_fact_id"`
	ToFactID   string `json:"to_fact_id"`
	Type       string `json:"reference_type"` // supports/contradicts/extends/qualifies
	Weight     uint32 `json:"weight"`
}

// CounterFactInfo holds counter-fact data.
type CounterFactInfo struct {
	ID             string `json:"id"`
	OriginalFactID string `json:"original_fact_id"`
	NegationType   string `json:"negation_type"` // rejection/contradiction/falsification/supersession
	Confidence     uint32 `json:"confidence"`
	Status         string `json:"status"`
}

// ClaimInfo holds claim data.
type ClaimInfo struct {
	ID        string `json:"id"`
	Text      string `json:"claim_text"`
	Domain    string `json:"domain"`
	Category  string `json:"category"`
	Status    string `json:"status"`
	Stake     string `json:"stake"`
}

// --- Request Types ---

// RegisterAgentRequest contains fields for agent registration.
type RegisterAgentRequest struct {
	DisplayName  string   `json:"display_name"`
	Domains      []string `json:"domains"`
	Capabilities []string `json:"capabilities"`
	Stake        uint64   `json:"stake"` // in ulgm
	Description  string   `json:"description"`
}

// SubmitClaimRequest contains fields for knowledge claim submission.
type SubmitClaimRequest struct {
	Text     string `json:"text"`
	Domain   string `json:"domain"`
	Category string `json:"category"`
	Stake    uint64 `json:"stake"` // in ulgm
}

// SubmitLIPRequest contains fields for governance proposal (LIP) submission.
type SubmitLIPRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	BugFactIDs  []string `json:"bug_fact_ids"`  // References to verified bug facts
	FixClaimID  string   `json:"fix_claim_id"`  // The fix claim
	Deposit     uint64   `json:"deposit"`        // in ulgm
}

// CreateVestingRequest contains fields for vesting reward creation.
type CreateVestingRequest struct {
	Beneficiary string `json:"beneficiary"` // Recipient address
	Amount      uint64 `json:"amount"`      // in ulgm
	Source      string `json:"source"`      // e.g. "bounty_fulfillment"
	Category    string `json:"category"`    // e.g. "protocol_observation"
	Description string `json:"description"` // Human-readable description
}

// ProposalInfo holds governance proposal data.
type ProposalInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"` // "voting", "passed", "rejected", "deposit"
	TriggerType string `json:"trigger_type,omitempty"`
	SubmittedBy string `json:"submitted_by"`
}

// BugFixRequest contains fields for submitting a bug fix.
type BugFixRequest struct {
	Title          string   `json:"title"`
	FixDescription string   `json:"fix_description"`
	BugFactIDs     []string `json:"bug_fact_ids"` // Verified bug facts this fix addresses
	Stake          uint64   `json:"stake"`         // in ulgm
	Deposit        uint64   `json:"deposit"`       // LIP deposit in ulgm
}

// BountyFinder holds the address and share for a bug finder.
type BountyFinder struct {
	Address string `json:"address"`
	Share   uint64 `json:"share"` // in ulgm
}

// BountyDistribution contains fields for distributing bug bounty payouts.
type BountyDistribution struct {
	BugFinders []BountyFinder `json:"bug_finders"`
	FixAuthor  string         `json:"fix_author"`
	FixShare   uint64         `json:"fix_share"`  // in ulgm
	BugFactIDs []string       `json:"bug_fact_ids"`
	LIPID      string         `json:"lip_id"`
}

// --- CLI Client Implementation ---

// CLIClient implements QueryClient and TxClient by shelling out to legbled.
// This is the simplest integration path — no gRPC dependencies needed.
type CLIClient struct {
	Binary   string // Path to legbled binary
	KeyName  string // Keyring key name
	Keyring  string // Keyring backend (test, file, os)
	Home     string // legbled home directory
	Node     string // RPC endpoint (tcp://host:port)
	RESTAddr string // REST/gRPC-gateway endpoint (http://host:port)
	ChainID  string // Chain ID
	GasPrice uint64 // Gas price in ulgm
}

// NewCLIClient creates a CLI-based client with defaults.
func NewCLIClient(keyName string) *CLIClient {
	return &CLIClient{
		Binary:   "legbled",
		KeyName:  keyName,
		Keyring:  "test",
		Home:     fmt.Sprintf("%s/.legbled", mustGetenv("HOME")),
		Node:     DefaultNode,
		ChainID:  ChainID,
		GasPrice: DefaultGasPrice,
	}
}

func mustGetenv(key string) string {
	v := ""
	if key == "HOME" {
		v, _ = homeDir()
	}
	return v
}

func homeDir() (string, error) {
	home, err := userHomeDir()
	if err != nil {
		return "/tmp", nil
	}
	return home, nil
}

// SetQueryClient sets a custom query client.
func (a *Agent) SetQueryClient(qc QueryClient) { a.queryClient = qc }

// SetTxClient sets a custom transaction client.
func (a *Agent) SetTxClient(tc TxClient) { a.txClient = tc }

// QueryClient returns the agent's query client.
func (a *Agent) QueryClient() QueryClient { return a.queryClient }

// TxClient returns the agent's transaction client.
func (a *Agent) TxClient() TxClient { return a.txClient }
