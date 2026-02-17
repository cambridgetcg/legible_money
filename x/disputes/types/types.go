package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ---------- Dispute Status Enum (not in proto) ----------

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

// ---------- Dispute Outcome (string aliases — proto DisputeOutcome is an int32 enum) ----------
// These string constants are used by keeper logic for dispute resolution.

const (
	OutcomeUpheld               = "upheld"         // original fact stands
	OutcomeOverturned           = "overturned"     // challenger wins
	OutcomeInconclusive         = "inconclusive"   // no clear winner
	OutcomeWeakened             = "weakened"       // partial success — confidence reduced
	OutcomeUndecidabilityProven = "undecidability" // Gödel zones — provably unprovable
)

// DisputeType distinguishes dispute origins.
const (
	DisputeTypeStandard             = "standard"              // traditional fact challenge
	DisputeTypeProvisionalChallenge = "provisional_challenge" // adversarial verification challenge
)

// EscalationTier defines dispute complexity levels.
type EscalationTier uint32

const (
	TierBasic     EscalationTier = 1 // 3 arbiters, 111 LGM, 2 days
	TierEscalated EscalationTier = 2 // 7 arbiters, 555 LGM, 4 days
	TierFinal     EscalationTier = 3 // 15 arbiters, 1111 LGM, 7 days
)

// ---------- Params Validation (method on proto-generated Params) ----------

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

// ---------- Hand-Written Message Types (no proto equivalent with same name) ----------

// MsgOpenDispute opens a new dispute against a verified fact.
// Proto service uses MsgInitiateDispute; this is the hand-written legacy type
// retained for backward compatibility with existing keeper logic.
type MsgOpenDispute struct {
	Challenger string `json:"challenger"` // bech32 address
	FactId     string `json:"fact_id"`
	Reason     string `json:"reason"`
	Bond       string `json:"bond"` // bigint as string
}

func (msg *MsgOpenDispute) Reset()         {}
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

func (msg *MsgOpenDisputeResponse) Reset()         {}
func (msg *MsgOpenDisputeResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgOpenDisputeResponse) ProtoMessage()  {}

// ---------- ValidateBasic / GetSigners for proto-generated Msg types ----------
// These methods extend the proto structs with SDK validation.

// ValidateBasic validates MsgCommitEvidence fields.
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
	if msg.CommitmentHash == "" {
		return fmt.Errorf("commitment hash cannot be empty")
	}
	return nil
}

// GetSigners returns the signers of MsgCommitEvidence.
func (msg *MsgCommitEvidence) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Submitter)
	return []sdk.AccAddress{addr}
}

// MsgSubmitEvidence reveals evidence for a dispute (phase 2 of commit-reveal).
// Proto service uses MsgRevealEvidence; this is the hand-written legacy type.
type MsgSubmitEvidence struct {
	Submitter string `json:"submitter"` // bech32 address
	DisputeId string `json:"dispute_id"`
	Side      string `json:"side"`    // "challenger" or "defender"
	Content   string `json:"content"` // evidence content/hash
}

func (msg *MsgSubmitEvidence) Reset()         {}
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

func (msg *MsgSubmitEvidenceResponse) Reset()         {}
func (msg *MsgSubmitEvidenceResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSubmitEvidenceResponse) ProtoMessage()  {}

// MsgVoteDispute casts a vote on a dispute.
// Proto service uses MsgArbiterVote; this is the hand-written legacy type.
type MsgVoteDispute struct {
	Arbiter   string `json:"arbiter"` // bech32 address
	DisputeId string `json:"dispute_id"`
	Vote      string `json:"vote"`      // "uphold", "overturn", "abstain"
	Rationale string `json:"rationale"` // brief rationale
}

func (msg *MsgVoteDispute) Reset()         {}
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

func (msg *MsgVoteDisputeResponse) Reset()         {}
func (msg *MsgVoteDisputeResponse) String() string { return "" }
func (msg *MsgVoteDisputeResponse) ProtoMessage()  {}

// ---------- ValidateBasic / GetSigners for proto MsgEscalateDispute ----------

// ValidateBasic validates MsgEscalateDispute fields.
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
	if msg.AdditionalStake == "" {
		return fmt.Errorf("additional stake cannot be empty")
	}
	return nil
}

// GetSigners returns the signers of MsgEscalateDispute.
func (msg *MsgEscalateDispute) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Requester)
	return []sdk.AccAddress{addr}
}

// ---------- ValidateBasic / GetSigners for proto MsgSettleDispute ----------

// ValidateBasic validates MsgSettleDispute fields.
func (msg *MsgSettleDispute) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.DisputeId == "" {
		return fmt.Errorf("dispute ID cannot be empty")
	}
	return nil
}

// GetSigners returns the signers of MsgSettleDispute.
func (msg *MsgSettleDispute) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// ---------- ValidateBasic / GetSigners for proto MsgTimeoutDispute ----------

// ValidateBasic validates MsgTimeoutDispute fields.
func (msg *MsgTimeoutDispute) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.DisputeId == "" {
		return fmt.Errorf("dispute ID cannot be empty")
	}
	return nil
}

// GetSigners returns the signers of MsgTimeoutDispute.
func (msg *MsgTimeoutDispute) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// ---------- Legacy MsgServer Interface ----------
// LegacyMsgServer extends the proto MsgServer with hand-written message handlers
// that use non-proto message types (MsgOpenDispute, MsgSubmitEvidence, MsgVoteDispute).
// Tests and internal code use this to access both proto and legacy methods.

type LegacyMsgServer interface {
	MsgServer
	OpenDispute(context.Context, *MsgOpenDispute) (*MsgOpenDisputeResponse, error)
	SubmitEvidence(context.Context, *MsgSubmitEvidence) (*MsgSubmitEvidenceResponse, error)
	VoteDispute(context.Context, *MsgVoteDispute) (*MsgVoteDisputeResponse, error)
}
