package types

import (
	"context"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ---------- Enums ----------

type ResearchType string

const (
	ResearchTypeReplication        ResearchType = "replication"
	ResearchTypeFraudInvestigation ResearchType = "fraud_investigation"
	ResearchTypeMethodologyAudit   ResearchType = "methodology_audit"
	ResearchTypeDataValidation     ResearchType = "data_validation"
)

func IsValidResearchType(rt ResearchType) bool {
	switch rt {
	case ResearchTypeReplication, ResearchTypeFraudInvestigation,
		ResearchTypeMethodologyAudit, ResearchTypeDataValidation:
		return true
	}
	return false
}

type ResearchStatus string

const (
	ResearchStatusSubmitted   ResearchStatus = "submitted"
	ResearchStatusUnderReview ResearchStatus = "under_review"
	ResearchStatusAccepted    ResearchStatus = "accepted"
	ResearchStatusRejected    ResearchStatus = "rejected"
	ResearchStatusChallenged  ResearchStatus = "challenged"
)

type BountyStatus string

const (
	BountyStatusOpen      BountyStatus = "open"
	BountyStatusClaimed   BountyStatus = "claimed"
	BountyStatusFulfilled BountyStatus = "fulfilled"
	BountyStatusExpired   BountyStatus = "expired"
)

type ReviewVerdict string

const (
	ReviewVerdictApprove ReviewVerdict = "approve"
	ReviewVerdictReject  ReviewVerdict = "reject"
	ReviewVerdictRevise  ReviewVerdict = "revise"
)

func IsValidReviewVerdict(v ReviewVerdict) bool {
	switch v {
	case ReviewVerdictApprove, ReviewVerdictReject, ReviewVerdictRevise:
		return true
	}
	return false
}

// ---------- State Types ----------

// Research represents a research submission on-chain.
type Research struct {
	Id             string         `json:"id"`
	Submitter      string         `json:"submitter"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Domain         string         `json:"domain"`
	ResearchType   ResearchType   `json:"research_type"`
	TargetFactId   string         `json:"target_fact_id"`
	Stake          string         `json:"stake"`
	Status         ResearchStatus `json:"status"`
	ReviewCount    uint32         `json:"review_count"`
	ApproveCount   uint32         `json:"approve_count"`
	RejectCount    uint32         `json:"reject_count"`
	ReviseCount    uint32         `json:"revise_count"`
	AggregateScore uint32         `json:"aggregate_score"`
	CreatedAt      uint64         `json:"created_at"`
	UpdatedAt      uint64         `json:"updated_at"`
}

// Bounty represents a research bounty.
type Bounty struct {
	Id             string       `json:"id"`
	Creator        string       `json:"creator"`
	Title          string       `json:"title"`
	Description    string       `json:"description"`
	Domain         string       `json:"domain"`
	Reward         string       `json:"reward"`
	DeadlineHeight uint64       `json:"deadline_height"`
	Status         BountyStatus `json:"status"`
	ClaimedBy      string       `json:"claimed_by,omitempty"`
	FulfilledBy    string       `json:"fulfilled_by,omitempty"`
	CreatedAt      uint64       `json:"created_at"`
}

// PeerReview represents a review of a research submission.
type PeerReview struct {
	Id         string        `json:"id"`
	ResearchId string        `json:"research_id"`
	Reviewer   string        `json:"reviewer"`
	Verdict    ReviewVerdict `json:"verdict"`
	Score      uint32        `json:"score"`
	Reasoning  string        `json:"reasoning"`
	ReviewedAt uint64        `json:"reviewed_at"`
}

// TreasuryBalance tracks the research treasury balance.
type TreasuryBalance struct {
	Balance string `json:"balance"`
}

// ---------- Params ----------

type Params struct {
	MinResearchStake         string `json:"min_research_stake"`
	MinChallengeStake        string `json:"min_challenge_stake"`
	ReviewPeriodBlocks       uint64 `json:"review_period_blocks"`
	MinReviewerCount         uint32 `json:"min_reviewer_count"`
	AcceptanceScoreThreshold uint32 `json:"acceptance_score_threshold"`
	RejectionSlashBps        uint64 `json:"rejection_slash_bps"`
	MaxBountyReward          string `json:"max_bounty_reward"`
	BountyMinDeadlineBlocks  uint64 `json:"bounty_min_deadline_blocks"`
}

func DefaultParams() Params {
	return Params{
		MinResearchStake:         "100000000",   // 100 LGM in ulgm
		MinChallengeStake:        "100000000",   // 100 LGM in ulgm
		ReviewPeriodBlocks:       68544,          // ~2 days
		MinReviewerCount:         3,
		AcceptanceScoreThreshold: 60,             // 60/100
		RejectionSlashBps:        3300,           // 33%
		MaxBountyReward:          "10000000000",  // 10,000 LGM in ulgm
		BountyMinDeadlineBlocks:  34272,          // ~1 day
	}
}

// ---------- Messages ----------

// MsgSubmitResearch submits a new research submission.
type MsgSubmitResearch struct {
	Submitter    string       `json:"submitter"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Domain       string       `json:"domain"`
	ResearchType ResearchType `json:"research_type"`
	TargetFactId string       `json:"target_fact_id"`
	Stake        string       `json:"stake"`
}

func (m *MsgSubmitResearch) Reset()         {}
func (m *MsgSubmitResearch) String() string { return fmt.Sprintf("MsgSubmitResearch{%s}", m.Title) }
func (m *MsgSubmitResearch) ProtoMessage()  {}

func (m *MsgSubmitResearch) ValidateBasic() error {
	if m.Submitter == "" {
		return fmt.Errorf("submitter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(m.Submitter); err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	if m.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if m.Domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	if !IsValidResearchType(m.ResearchType) {
		return fmt.Errorf("invalid research type: %s", m.ResearchType)
	}
	amount := new(big.Int)
	if _, ok := amount.SetString(m.Stake, 10); !ok || amount.Sign() <= 0 {
		return fmt.Errorf("invalid stake amount")
	}
	return nil
}

func (m *MsgSubmitResearch) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Submitter)
	return []sdk.AccAddress{addr}
}

type MsgSubmitResearchResponse struct {
	ResearchId string `json:"research_id"`
}

func (m *MsgSubmitResearchResponse) Reset()         {}
func (m *MsgSubmitResearchResponse) String() string { return "MsgSubmitResearchResponse" }
func (m *MsgSubmitResearchResponse) ProtoMessage()  {}

// MsgChallengeResearch challenges an existing research submission.
type MsgChallengeResearch struct {
	Challenger string `json:"challenger"`
	ResearchId string `json:"research_id"`
	Reason     string `json:"reason"`
	Stake      string `json:"stake"`
}

func (m *MsgChallengeResearch) Reset()         {}
func (m *MsgChallengeResearch) String() string { return "MsgChallengeResearch" }
func (m *MsgChallengeResearch) ProtoMessage()  {}

func (m *MsgChallengeResearch) ValidateBasic() error {
	if m.Challenger == "" {
		return fmt.Errorf("challenger cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(m.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	if m.ResearchId == "" {
		return fmt.Errorf("research id cannot be empty")
	}
	if m.Reason == "" {
		return fmt.Errorf("reason cannot be empty")
	}
	amount := new(big.Int)
	if _, ok := amount.SetString(m.Stake, 10); !ok || amount.Sign() <= 0 {
		return fmt.Errorf("invalid stake amount")
	}
	return nil
}

func (m *MsgChallengeResearch) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Challenger)
	return []sdk.AccAddress{addr}
}

type MsgChallengeResearchResponse struct{}

func (m *MsgChallengeResearchResponse) Reset()         {}
func (m *MsgChallengeResearchResponse) String() string { return "MsgChallengeResearchResponse" }
func (m *MsgChallengeResearchResponse) ProtoMessage()  {}

// MsgReviewResearch submits a peer review for a research submission.
type MsgReviewResearch struct {
	Reviewer   string        `json:"reviewer"`
	ResearchId string        `json:"research_id"`
	Verdict    ReviewVerdict `json:"verdict"`
	Score      uint32        `json:"score"`
	Reasoning  string        `json:"reasoning"`
}

func (m *MsgReviewResearch) Reset()         {}
func (m *MsgReviewResearch) String() string { return "MsgReviewResearch" }
func (m *MsgReviewResearch) ProtoMessage()  {}

func (m *MsgReviewResearch) ValidateBasic() error {
	if m.Reviewer == "" {
		return fmt.Errorf("reviewer cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(m.Reviewer); err != nil {
		return fmt.Errorf("invalid reviewer address: %w", err)
	}
	if m.ResearchId == "" {
		return fmt.Errorf("research id cannot be empty")
	}
	if !IsValidReviewVerdict(m.Verdict) {
		return fmt.Errorf("invalid verdict: %s", m.Verdict)
	}
	if m.Score > 100 {
		return fmt.Errorf("score must be 0-100, got %d", m.Score)
	}
	return nil
}

func (m *MsgReviewResearch) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Reviewer)
	return []sdk.AccAddress{addr}
}

type MsgReviewResearchResponse struct{}

func (m *MsgReviewResearchResponse) Reset()         {}
func (m *MsgReviewResearchResponse) String() string { return "MsgReviewResearchResponse" }
func (m *MsgReviewResearchResponse) ProtoMessage()  {}

// MsgResolveResearch resolves a research submission after reviews are complete.
type MsgResolveResearch struct {
	Authority  string `json:"authority"`
	ResearchId string `json:"research_id"`
}

func (m *MsgResolveResearch) Reset()         {}
func (m *MsgResolveResearch) String() string { return "MsgResolveResearch" }
func (m *MsgResolveResearch) ProtoMessage()  {}

func (m *MsgResolveResearch) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if m.ResearchId == "" {
		return fmt.Errorf("research id cannot be empty")
	}
	return nil
}

func (m *MsgResolveResearch) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

type MsgResolveResearchResponse struct {
	Outcome string `json:"outcome"`
}

func (m *MsgResolveResearchResponse) Reset()         {}
func (m *MsgResolveResearchResponse) String() string { return "MsgResolveResearchResponse" }
func (m *MsgResolveResearchResponse) ProtoMessage()  {}

// MsgCreateBounty creates a research bounty.
type MsgCreateBounty struct {
	Creator        string `json:"creator"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	Domain         string `json:"domain"`
	Reward         string `json:"reward"`
	DeadlineHeight uint64 `json:"deadline_height"`
}

func (m *MsgCreateBounty) Reset()         {}
func (m *MsgCreateBounty) String() string { return "MsgCreateBounty" }
func (m *MsgCreateBounty) ProtoMessage()  {}

func (m *MsgCreateBounty) ValidateBasic() error {
	if m.Creator == "" {
		return fmt.Errorf("creator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	reward := new(big.Int)
	if _, ok := reward.SetString(m.Reward, 10); !ok || reward.Sign() <= 0 {
		return fmt.Errorf("invalid reward amount")
	}
	if m.DeadlineHeight == 0 {
		return fmt.Errorf("deadline height must be > 0")
	}
	return nil
}

func (m *MsgCreateBounty) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Creator)
	return []sdk.AccAddress{addr}
}

type MsgCreateBountyResponse struct {
	BountyId string `json:"bounty_id"`
}

func (m *MsgCreateBountyResponse) Reset()         {}
func (m *MsgCreateBountyResponse) String() string { return "MsgCreateBountyResponse" }
func (m *MsgCreateBountyResponse) ProtoMessage()  {}

// MsgClaimBounty claims an open bounty.
type MsgClaimBounty struct {
	Claimer  string `json:"claimer"`
	BountyId string `json:"bounty_id"`
}

func (m *MsgClaimBounty) Reset()         {}
func (m *MsgClaimBounty) String() string { return "MsgClaimBounty" }
func (m *MsgClaimBounty) ProtoMessage()  {}

func (m *MsgClaimBounty) ValidateBasic() error {
	if m.Claimer == "" {
		return fmt.Errorf("claimer cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(m.Claimer); err != nil {
		return fmt.Errorf("invalid claimer address: %w", err)
	}
	if m.BountyId == "" {
		return fmt.Errorf("bounty id cannot be empty")
	}
	return nil
}

func (m *MsgClaimBounty) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Claimer)
	return []sdk.AccAddress{addr}
}

type MsgClaimBountyResponse struct{}

func (m *MsgClaimBountyResponse) Reset()         {}
func (m *MsgClaimBountyResponse) String() string { return "MsgClaimBountyResponse" }
func (m *MsgClaimBountyResponse) ProtoMessage()  {}

// MsgFulfillBounty marks a bounty as fulfilled.
type MsgFulfillBounty struct {
	Authority string `json:"authority"`
	BountyId  string `json:"bounty_id"`
}

func (m *MsgFulfillBounty) Reset()         {}
func (m *MsgFulfillBounty) String() string { return "MsgFulfillBounty" }
func (m *MsgFulfillBounty) ProtoMessage()  {}

func (m *MsgFulfillBounty) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if m.BountyId == "" {
		return fmt.Errorf("bounty id cannot be empty")
	}
	return nil
}

func (m *MsgFulfillBounty) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

type MsgFulfillBountyResponse struct {
	Reward string `json:"reward"`
}

func (m *MsgFulfillBountyResponse) Reset()         {}
func (m *MsgFulfillBountyResponse) String() string { return "MsgFulfillBountyResponse" }
func (m *MsgFulfillBountyResponse) ProtoMessage()  {}

// MsgFundResearch adds funds to the research treasury.
type MsgFundResearch struct {
	Funder string `json:"funder"`
	Amount string `json:"amount"`
}

func (m *MsgFundResearch) Reset()         {}
func (m *MsgFundResearch) String() string { return "MsgFundResearch" }
func (m *MsgFundResearch) ProtoMessage()  {}

func (m *MsgFundResearch) ValidateBasic() error {
	if m.Funder == "" {
		return fmt.Errorf("funder cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(m.Funder); err != nil {
		return fmt.Errorf("invalid funder address: %w", err)
	}
	amount := new(big.Int)
	if _, ok := amount.SetString(m.Amount, 10); !ok || amount.Sign() <= 0 {
		return fmt.Errorf("invalid fund amount")
	}
	return nil
}

func (m *MsgFundResearch) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Funder)
	return []sdk.AccAddress{addr}
}

type MsgFundResearchResponse struct {
	TreasuryBalance string `json:"treasury_balance"`
}

func (m *MsgFundResearchResponse) Reset()         {}
func (m *MsgFundResearchResponse) String() string { return "MsgFundResearchResponse" }
func (m *MsgFundResearchResponse) ProtoMessage()  {}

// MsgUpdateParams updates module parameters.
type MsgUpdateParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

func (m *MsgUpdateParams) Reset()         {}
func (m *MsgUpdateParams) String() string { return "MsgUpdateParams" }
func (m *MsgUpdateParams) ProtoMessage()  {}

func (m *MsgUpdateParams) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	return nil
}

func (m *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

type MsgUpdateParamsResponse struct{}

func (m *MsgUpdateParamsResponse) Reset()         {}
func (m *MsgUpdateParamsResponse) String() string { return "MsgUpdateParamsResponse" }
func (m *MsgUpdateParamsResponse) ProtoMessage()  {}

// ---------- MsgServer Interface ----------

type MsgServer interface {
	SubmitResearch(context.Context, *MsgSubmitResearch) (*MsgSubmitResearchResponse, error)
	ChallengeResearch(context.Context, *MsgChallengeResearch) (*MsgChallengeResearchResponse, error)
	ReviewResearch(context.Context, *MsgReviewResearch) (*MsgReviewResearchResponse, error)
	ResolveResearch(context.Context, *MsgResolveResearch) (*MsgResolveResearchResponse, error)
	CreateBounty(context.Context, *MsgCreateBounty) (*MsgCreateBountyResponse, error)
	ClaimBounty(context.Context, *MsgClaimBounty) (*MsgClaimBountyResponse, error)
	FulfillBounty(context.Context, *MsgFulfillBounty) (*MsgFulfillBountyResponse, error)
	FundResearch(context.Context, *MsgFundResearch) (*MsgFundResearchResponse, error)
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}
