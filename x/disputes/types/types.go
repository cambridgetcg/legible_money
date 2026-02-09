package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ---------- Dispute Status & Outcome Enums ----------

// DisputeStatus tracks dispute lifecycle.
type DisputeStatus string

const (
	DisputeStatusOpen      DisputeStatus = "open"
	DisputeStatusEvidence  DisputeStatus = "evidence"
	DisputeStatusVoting    DisputeStatus = "voting"
	DisputeStatusResolved  DisputeStatus = "resolved"
	DisputeStatusExpired   DisputeStatus = "expired"
	DisputeStatusEscalated DisputeStatus = "escalated"
)

// DisputeOutcome is the resolution result.
type DisputeOutcome string

const (
	OutcomeUpheld       DisputeOutcome = "upheld"       // original fact stands
	OutcomeOverturned   DisputeOutcome = "overturned"   // challenger wins
	OutcomeInconclusive DisputeOutcome = "inconclusive" // no clear winner
)

// EscalationTier defines dispute complexity levels.
type EscalationTier uint32

const (
	TierBasic     EscalationTier = 1 // 3 arbiters, 111 LGM, 2 days
	TierEscalated EscalationTier = 2 // 7 arbiters, 555 LGM, 4 days
	TierFinal     EscalationTier = 3 // 15 arbiters, 1111 LGM, 7 days
)

// ---------- Core Data Types ----------

// Dispute represents a challenge to a verified fact.
type Dispute struct {
	Id               string         `json:"id"`
	FactId           string         `json:"fact_id"`
	ClaimId          string         `json:"claim_id"`
	Challenger       string         `json:"challenger"`        // bech32 address
	Defender         string         `json:"defender"`          // bech32 address of original claim submitter
	Reason           string         `json:"reason"`
	Bond             string         `json:"bond"`              // stake locked by challenger (bigint string)
	Tier             EscalationTier `json:"tier"`              // current escalation tier
	Status           DisputeStatus  `json:"status"`            // current status
	Outcome          DisputeOutcome `json:"outcome"`           // final outcome (if resolved)
	CreatedAt        uint64         `json:"created_at"`        // block height
	EvidenceDeadline uint64         `json:"evidence_deadline"` // block height deadline for evidence
	VotingDeadline   uint64         `json:"voting_deadline"`   // block height deadline for voting
	Arbiters         []string       `json:"arbiters"`          // selected arbiter addresses
	EvidenceCount    uint32         `json:"evidence_count"`    // total evidence submissions
}

func (d *Dispute) Reset()        {}
func (d *Dispute) String() string { return fmt.Sprintf("%+v", *d) }
func (d *Dispute) ProtoMessage()  {}

// DisputeEvidence represents evidence submitted for a dispute.
type DisputeEvidence struct {
	Id          string `json:"id"`
	DisputeId   string `json:"dispute_id"`
	Submitter   string `json:"submitter"` // bech32 address
	Side        string `json:"side"`      // "challenger" or "defender"
	Content     string `json:"content"`   // evidence content/hash
	SubmittedAt uint64 `json:"submitted_at"` // block height
}

func (e *DisputeEvidence) Reset()        {}
func (e *DisputeEvidence) String() string { return fmt.Sprintf("%+v", *e) }
func (e *DisputeEvidence) ProtoMessage()  {}

// EvidenceCommitment represents a committed (but not yet revealed) evidence hash.
type EvidenceCommitment struct {
	DisputeId   string `json:"dispute_id"`
	Submitter   string `json:"submitter"`    // bech32 address
	Side        string `json:"side"`         // "challenger" or "defender"
	ContentHash string `json:"content_hash"` // SHA-256 of evidence content
	CommittedAt uint64 `json:"committed_at"` // block height
	Revealed    bool   `json:"revealed"`     // whether the reveal has been submitted
}

func (c *EvidenceCommitment) Reset()        {}
func (c *EvidenceCommitment) String() string { return fmt.Sprintf("%+v", *c) }
func (c *EvidenceCommitment) ProtoMessage()  {}

// DisputeVote represents an arbiter's vote.
type DisputeVote struct {
	DisputeId string `json:"dispute_id"`
	Arbiter   string `json:"arbiter"` // bech32 address
	Vote      string `json:"vote"`    // "uphold", "overturn", "abstain"
	Stake     string `json:"stake"`   // arbiter's stake weight (bigint string)
	Rationale string `json:"rationale"`
	VotedAt   uint64 `json:"voted_at"` // block height
}

func (v *DisputeVote) Reset()        {}
func (v *DisputeVote) String() string { return fmt.Sprintf("%+v", *v) }
func (v *DisputeVote) ProtoMessage()  {}

// TierConfig defines requirements for each escalation tier.
type TierConfig struct {
	Tier              EscalationTier `json:"tier"`
	ArbiterCount      uint32         `json:"arbiter_count"`      // number of arbiters
	MinBond           string         `json:"min_bond"`           // minimum challenger bond (bigint string)
	EvidencePeriod    uint64         `json:"evidence_period"`    // blocks for evidence submission
	VotingPeriod      uint64         `json:"voting_period"`      // blocks for voting
	QuorumThreshold   uint64         `json:"quorum_threshold"`   // basis points (660000 = 66%)
	MajorityThreshold uint64         `json:"majority_threshold"` // basis points (770000 = 77%)
}

func (t *TierConfig) Reset()        {}
func (t *TierConfig) String() string { return fmt.Sprintf("%+v", *t) }
func (t *TierConfig) ProtoMessage()  {}

// Params for the disputes module.
type Params struct {
	TierConfigs       []TierConfig `json:"tier_configs"`
	MaxActiveDisputes uint32       `json:"max_active_disputes"` // max active disputes per fact
	EscalationDelay   uint64       `json:"escalation_delay"`    // blocks before escalation allowed
	SlashRateLoser    uint64       `json:"slash_rate_loser"`    // basis points of bond slashed from loser
	RewardRateWinner  uint64       `json:"reward_rate_winner"`  // basis points of slashed bond given to winner
	ArbiterRewardRate uint64       `json:"arbiter_reward_rate"` // basis points of bond given to arbiters
}

func (p *Params) Reset()        {}
func (p *Params) String() string { return fmt.Sprintf("%+v", *p) }
func (p *Params) ProtoMessage()  {}

// Validate performs parameter validation.
func (p Params) Validate() error {
	if len(p.TierConfigs) == 0 {
		return fmt.Errorf("tier configs cannot be empty")
	}
	if p.SlashRateLoser > 1000000 {
		return fmt.Errorf("slash rate cannot exceed 1000000 basis points")
	}
	if p.RewardRateWinner > 1000000 {
		return fmt.Errorf("reward rate cannot exceed 1000000 basis points")
	}
	if p.ArbiterRewardRate > 1000000 {
		return fmt.Errorf("arbiter reward rate cannot exceed 1000000 basis points")
	}
	return nil
}

// ---------- Message Types ----------

// MsgOpenDispute opens a new dispute against a verified fact.
type MsgOpenDispute struct {
	Challenger string `json:"challenger"` // bech32 address
	FactId     string `json:"fact_id"`
	Reason     string `json:"reason"`
	Bond       string `json:"bond"` // bigint as string
}

func (msg *MsgOpenDispute) Reset()        {}
func (msg *MsgOpenDispute) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgOpenDispute) ProtoMessage()  {}

func (msg *MsgOpenDispute) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Challenger)
	return []sdk.AccAddress{addr}
}

func (msg *MsgOpenDispute) ValidateBasic() error {
	if msg.Challenger == "" {
		return fmt.Errorf("challenger cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	if msg.FactId == "" {
		return fmt.Errorf("fact ID cannot be empty")
	}
	if msg.Reason == "" {
		return fmt.Errorf("reason cannot be empty")
	}
	if msg.Bond == "" {
		return fmt.Errorf("bond cannot be empty")
	}
	return nil
}

// MsgOpenDisputeResponse is the response to MsgOpenDispute.
type MsgOpenDisputeResponse struct {
	DisputeId string `json:"dispute_id"`
}

func (msg *MsgOpenDisputeResponse) Reset()        {}
func (msg *MsgOpenDisputeResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgOpenDisputeResponse) ProtoMessage()  {}

// MsgCommitEvidence commits a hash of evidence (phase 1 of commit-reveal).
type MsgCommitEvidence struct {
	Submitter   string `json:"submitter"`    // bech32 address
	DisputeId   string `json:"dispute_id"`
	Side        string `json:"side"`         // "challenger" or "defender"
	ContentHash string `json:"content_hash"` // SHA-256 hex of evidence content
}

func (msg *MsgCommitEvidence) Reset()        {}
func (msg *MsgCommitEvidence) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgCommitEvidence) ProtoMessage()  {}

func (msg *MsgCommitEvidence) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Submitter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgCommitEvidence) ValidateBasic() error {
	if msg.Submitter == "" {
		return fmt.Errorf("submitter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Submitter); err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	if msg.DisputeId == "" {
		return fmt.Errorf("dispute ID cannot be empty")
	}
	if msg.Side != "challenger" && msg.Side != "defender" {
		return fmt.Errorf("side must be 'challenger' or 'defender'")
	}
	if msg.ContentHash == "" {
		return fmt.Errorf("content hash cannot be empty")
	}
	if len(msg.ContentHash) != 64 {
		return fmt.Errorf("content hash must be 64 hex characters (SHA-256)")
	}
	return nil
}

// MsgCommitEvidenceResponse is the response to MsgCommitEvidence.
type MsgCommitEvidenceResponse struct{}

func (msg *MsgCommitEvidenceResponse) Reset()        {}
func (msg *MsgCommitEvidenceResponse) String() string { return "" }
func (msg *MsgCommitEvidenceResponse) ProtoMessage()  {}

// MsgSubmitEvidence reveals evidence for a dispute (phase 2 of commit-reveal).
type MsgSubmitEvidence struct {
	Submitter string `json:"submitter"` // bech32 address
	DisputeId string `json:"dispute_id"`
	Side      string `json:"side"`    // "challenger" or "defender"
	Content   string `json:"content"` // evidence content/hash
}

func (msg *MsgSubmitEvidence) Reset()        {}
func (msg *MsgSubmitEvidence) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSubmitEvidence) ProtoMessage()  {}

func (msg *MsgSubmitEvidence) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Submitter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSubmitEvidence) ValidateBasic() error {
	if msg.Submitter == "" {
		return fmt.Errorf("submitter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Submitter); err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	if msg.DisputeId == "" {
		return fmt.Errorf("dispute ID cannot be empty")
	}
	if msg.Side != "challenger" && msg.Side != "defender" {
		return fmt.Errorf("side must be 'challenger' or 'defender'")
	}
	if msg.Content == "" {
		return fmt.Errorf("content cannot be empty")
	}
	return nil
}

// MsgSubmitEvidenceResponse is the response to MsgSubmitEvidence.
type MsgSubmitEvidenceResponse struct {
	EvidenceId string `json:"evidence_id"`
}

func (msg *MsgSubmitEvidenceResponse) Reset()        {}
func (msg *MsgSubmitEvidenceResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSubmitEvidenceResponse) ProtoMessage()  {}

// MsgVoteDispute casts a vote on a dispute.
type MsgVoteDispute struct {
	Arbiter   string `json:"arbiter"` // bech32 address
	DisputeId string `json:"dispute_id"`
	Vote      string `json:"vote"`      // "uphold", "overturn", "abstain"
	Rationale string `json:"rationale"` // brief rationale
}

func (msg *MsgVoteDispute) Reset()        {}
func (msg *MsgVoteDispute) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgVoteDispute) ProtoMessage()  {}

func (msg *MsgVoteDispute) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Arbiter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgVoteDispute) ValidateBasic() error {
	if msg.Arbiter == "" {
		return fmt.Errorf("arbiter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Arbiter); err != nil {
		return fmt.Errorf("invalid arbiter address: %w", err)
	}
	if msg.DisputeId == "" {
		return fmt.Errorf("dispute ID cannot be empty")
	}
	if msg.Vote != "uphold" && msg.Vote != "overturn" && msg.Vote != "abstain" {
		return fmt.Errorf("vote must be 'uphold', 'overturn', or 'abstain'")
	}
	return nil
}

// MsgVoteDisputeResponse is the response to MsgVoteDispute.
type MsgVoteDisputeResponse struct{}

func (msg *MsgVoteDisputeResponse) Reset()        {}
func (msg *MsgVoteDisputeResponse) String() string { return "" }
func (msg *MsgVoteDisputeResponse) ProtoMessage()  {}

// MsgEscalateDispute escalates a dispute to the next tier.
type MsgEscalateDispute struct {
	Requester      string `json:"requester"`       // bech32 address
	DisputeId      string `json:"dispute_id"`
	AdditionalBond string `json:"additional_bond"` // bigint as string
}

func (msg *MsgEscalateDispute) Reset()        {}
func (msg *MsgEscalateDispute) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgEscalateDispute) ProtoMessage()  {}

func (msg *MsgEscalateDispute) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Requester)
	return []sdk.AccAddress{addr}
}

func (msg *MsgEscalateDispute) ValidateBasic() error {
	if msg.Requester == "" {
		return fmt.Errorf("requester cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Requester); err != nil {
		return fmt.Errorf("invalid requester address: %w", err)
	}
	if msg.DisputeId == "" {
		return fmt.Errorf("dispute ID cannot be empty")
	}
	if msg.AdditionalBond == "" {
		return fmt.Errorf("additional bond cannot be empty")
	}
	return nil
}

// MsgEscalateDisputeResponse is the response to MsgEscalateDispute.
type MsgEscalateDisputeResponse struct{}

func (msg *MsgEscalateDisputeResponse) Reset()        {}
func (msg *MsgEscalateDisputeResponse) String() string { return "" }
func (msg *MsgEscalateDisputeResponse) ProtoMessage()  {}

// MsgSettleDispute triggers settlement of a dispute that's past its voting deadline.
// Any account can submit this — it's a permissionless trigger for tally + resolution.
type MsgSettleDispute struct {
	Sender    string `json:"sender"`     // bech32 address of the trigger sender (pays gas)
	DisputeId string `json:"dispute_id"`
}

func (msg *MsgSettleDispute) Reset()        {}
func (msg *MsgSettleDispute) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSettleDispute) ProtoMessage()  {}

func (msg *MsgSettleDispute) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSettleDispute) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.DisputeId == "" {
		return fmt.Errorf("dispute ID cannot be empty")
	}
	return nil
}

// MsgSettleDisputeResponse is the response to MsgSettleDispute.
type MsgSettleDisputeResponse struct {
	Outcome string `json:"outcome"` // "upheld", "overturned", or "inconclusive"
}

func (msg *MsgSettleDisputeResponse) Reset()        {}
func (msg *MsgSettleDisputeResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSettleDisputeResponse) ProtoMessage()  {}

// MsgTimeoutDispute triggers timeout processing of an expired dispute.
// Any account can submit this — permissionless trigger for evidence→voting transition or expiration.
type MsgTimeoutDispute struct {
	Sender    string `json:"sender"`     // bech32 address of the trigger sender (pays gas)
	DisputeId string `json:"dispute_id"`
}

func (msg *MsgTimeoutDispute) Reset()        {}
func (msg *MsgTimeoutDispute) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgTimeoutDispute) ProtoMessage()  {}

func (msg *MsgTimeoutDispute) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgTimeoutDispute) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.DisputeId == "" {
		return fmt.Errorf("dispute ID cannot be empty")
	}
	return nil
}

// MsgTimeoutDisputeResponse is the response to MsgTimeoutDispute.
type MsgTimeoutDisputeResponse struct {
	NewStatus string `json:"new_status"` // new dispute status after timeout processing
}

func (msg *MsgTimeoutDisputeResponse) Reset()        {}
func (msg *MsgTimeoutDisputeResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgTimeoutDisputeResponse) ProtoMessage()  {}

// ---------- MsgServer Interface ----------

// MsgServer is the service interface for the disputes module.
type MsgServer interface {
	OpenDispute(context.Context, *MsgOpenDispute) (*MsgOpenDisputeResponse, error)
	CommitEvidence(context.Context, *MsgCommitEvidence) (*MsgCommitEvidenceResponse, error)
	SubmitEvidence(context.Context, *MsgSubmitEvidence) (*MsgSubmitEvidenceResponse, error)
	VoteDispute(context.Context, *MsgVoteDispute) (*MsgVoteDisputeResponse, error)
	EscalateDispute(context.Context, *MsgEscalateDispute) (*MsgEscalateDisputeResponse, error)
	SettleDispute(context.Context, *MsgSettleDispute) (*MsgSettleDisputeResponse, error)
	TimeoutDispute(context.Context, *MsgTimeoutDispute) (*MsgTimeoutDisputeResponse, error)
}

