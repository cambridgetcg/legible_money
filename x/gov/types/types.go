package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Enums (hand-written string-based, complementary to proto int32 enums) ---

// LIPCategory determines stake requirements and review thresholds.
type LIPCategory string

const (
	CategoryConsensus  LIPCategory = "consensus"
	CategoryEconomics  LIPCategory = "economics"
	CategoryInterface  LIPCategory = "interface"
	CategoryProcess    LIPCategory = "process"
)

// VoteDirection represents how a voter voted (hand-written string enum).
type VoteDirection string

const (
	VoteSupport VoteDirection = "support"
	VoteOppose  VoteDirection = "oppose"
	VoteAbstain VoteDirection = "abstain"
)

// DisbursementCategory categorizes treasury spending.
type DisbursementCategory string

const (
	DisbInfrastructure DisbursementCategory = "infrastructure"
	DisbLegal          DisbursementCategory = "legal"
	DisbDomain         DisbursementCategory = "domain"
	DisbMarketing      DisbursementCategory = "marketing"
	DisbDevelopment    DisbursementCategory = "development"
	DisbResearch       DisbursementCategory = "research"
	DisbOperations     DisbursementCategory = "operations"
	DisbEmergency      DisbursementCategory = "emergency"
)

// DisbursementStage represents the lifecycle of a disbursement proposal.
type DisbursementStage string

const (
	DisbStageProposed  DisbursementStage = "proposed"
	DisbStageVoting    DisbursementStage = "voting"
	DisbStageApproved  DisbursementStage = "approved"
	DisbStageExecuted  DisbursementStage = "executed"
	DisbStageRejected  DisbursementStage = "rejected"
	DisbStageCancelled DisbursementStage = "cancelled"
)

// HandwrittenLIPStage is the hand-written string-based stage type.
// The proto-generated LIPStage is an int32 enum. This string-based
// type is used by keeper logic for store values and stage comparisons.
type HandwrittenLIPStage string

const (
	StageDraft     HandwrittenLIPStage = "draft"
	StageReview    HandwrittenLIPStage = "review"
	StageLastCall  HandwrittenLIPStage = "last_call"
	StageVoting    HandwrittenLIPStage = "voting"
	StageAccepted  HandwrittenLIPStage = "accepted"
	StageRejected  HandwrittenLIPStage = "rejected"
	StageWithdrawn HandwrittenLIPStage = "withdrawn"
)

// --- Params Validation ---

// Validate performs basic param validation.
func (p Params) Validate() error {
	if p.VotingPeriodBlocks == 0 {
		return fmt.Errorf("voting_period_blocks must be > 0")
	}
	if p.MaxVotesPerVoter == 0 {
		return fmt.Errorf("max_votes_per_voter must be > 0")
	}
	if p.SupportThresholdBps > 10000 {
		return fmt.Errorf("support_threshold_bps must be <= 10000")
	}
	if p.QuorumThresholdBps > 10000 {
		return fmt.Errorf("quorum_threshold_bps must be <= 10000")
	}
	return nil
}

// GetLIPStake returns the required stake for a LIP category.
func (p Params) GetLIPStake(category LIPCategory) string {
	switch category {
	case CategoryConsensus:
		return p.LipStakeConsensus
	case CategoryEconomics:
		return p.LipStakeEconomics
	case CategoryInterface:
		return p.LipStakeInterface
	case CategoryProcess:
		return p.LipStakeProcess
	default:
		return p.LipStakeProcess
	}
}

// --- Validation helpers ---

// ValidLIPCategories for input validation.
var ValidLIPCategories = map[LIPCategory]bool{
	CategoryConsensus: true,
	CategoryEconomics: true,
	CategoryInterface: true,
	CategoryProcess:   true,
}

// ValidVoteDirections for input validation.
var ValidVoteDirections = map[VoteDirection]bool{
	VoteSupport: true,
	VoteOppose:  true,
	VoteAbstain: true,
}

// ValidDisbursementCategories for input validation.
var ValidDisbursementCategories = map[DisbursementCategory]bool{
	DisbInfrastructure: true,
	DisbLegal:          true,
	DisbDomain:         true,
	DisbMarketing:      true,
	DisbDevelopment:    true,
	DisbResearch:       true,
	DisbOperations:     true,
	DisbEmergency:      true,
}

// IsTerminalLIPStage returns true if the LIP stage is terminal.
func IsTerminalLIPStage(stage HandwrittenLIPStage) bool {
	return stage == StageAccepted || stage == StageRejected || stage == StageWithdrawn
}

// IsTerminalDisbStage returns true if the disbursement stage is terminal.
func IsTerminalDisbStage(stage DisbursementStage) bool {
	return stage == DisbStageExecuted || stage == DisbStageRejected || stage == DisbStageCancelled
}

// --- ValidateBasic / GetSigners for proto-generated message types ---
// These methods extend the proto-generated structs.

func (msg *MsgSubmitLIP) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if strings.TrimSpace(msg.Title) == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if len(msg.Title) > 200 {
		return fmt.Errorf("title too long (max 200)")
	}
	if len(msg.Description) > 2000 {
		return fmt.Errorf("description too long (max 2000)")
	}
	// Require positive initial stake to prevent zero-cost proposal spam
	if msg.InitialStake == "" || msg.InitialStake == "0" {
		return fmt.Errorf("initial_stake must be positive: proposals require a deposit")
	}
	return nil
}

func (msg *MsgSubmitLIP) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Proposer)
	return []sdk.AccAddress{addr}
}

func (msg *MsgStakeLIP) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Staker); err != nil {
		return fmt.Errorf("invalid staker address: %w", err)
	}
	if msg.LipId == "" {
		return fmt.Errorf("lip_id cannot be empty")
	}
	return nil
}

func (msg *MsgStakeLIP) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Staker)
	return []sdk.AccAddress{addr}
}

func (msg *MsgCastVote) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Voter); err != nil {
		return fmt.Errorf("invalid voter address: %w", err)
	}
	if msg.ProposalId == "" {
		return fmt.Errorf("proposal_id cannot be empty")
	}
	if msg.Option == VoteOption_VOTE_OPTION_UNSPECIFIED {
		return fmt.Errorf("vote option cannot be unspecified")
	}
	// Require positive stake to prevent unstaked voting
	if msg.Stake == "" {
		return fmt.Errorf("stake cannot be empty: voters must stake tokens")
	}
	return nil
}

func (msg *MsgCastVote) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Voter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgAdvanceLIPStage) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.LipId == "" {
		return fmt.Errorf("lip_id cannot be empty")
	}
	return nil
}

func (msg *MsgAdvanceLIPStage) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgWithdrawLIP) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if msg.LipId == "" {
		return fmt.Errorf("lip_id cannot be empty")
	}
	return nil
}

func (msg *MsgWithdrawLIP) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Proposer)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSubmitDisbursement) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Submitter); err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	if strings.TrimSpace(msg.Title) == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Recipient); err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}
	return nil
}

func (msg *MsgSubmitDisbursement) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Submitter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgExecuteDisbursement) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.DisbursementId == "" {
		return fmt.Errorf("disbursement_id cannot be empty")
	}
	return nil
}

func (msg *MsgExecuteDisbursement) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgCancelDisbursement) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.DisbursementId == "" {
		return fmt.Errorf("disbursement_id cannot be empty")
	}
	return nil
}

func (msg *MsgCancelDisbursement) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// --- MsgUpdateParams is NOT in proto; keep hand-written ---

// MsgUpdateParams updates governance module params (authority only).
type MsgUpdateParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

type MsgUpdateParamsResponse struct{}

func (msg MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	return msg.Params.Validate()
}

func (msg MsgUpdateParams) Reset()         {}
func (msg MsgUpdateParams) String() string { return "MsgUpdateParams{}" }
func (msg MsgUpdateParams) ProtoMessage()  {}

func (msg MsgUpdateParamsResponse) Reset()         {}
func (msg MsgUpdateParamsResponse) String() string { return "MsgUpdateParamsResponse{}" }
func (msg MsgUpdateParamsResponse) ProtoMessage()  {}

// --- MsgAttachUpgradePlan is NOT in proto; keep hand-written ---

// MsgAttachUpgradePlan attaches a software upgrade plan to a consensus-category LIP.
type MsgAttachUpgradePlan struct {
	Proposer    string `json:"proposer"`
	LipId       string `json:"lip_id"`
	UpgradeName string `json:"upgrade_name"`
	Height      int64  `json:"height"`
	Info        string `json:"info"`
}

type MsgAttachUpgradePlanResponse struct{}

func (msg MsgAttachUpgradePlan) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Proposer)
	return []sdk.AccAddress{addr}
}

func (msg MsgAttachUpgradePlan) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if msg.LipId == "" {
		return fmt.Errorf("lip_id cannot be empty")
	}
	if strings.TrimSpace(msg.UpgradeName) == "" {
		return fmt.Errorf("upgrade_name cannot be empty")
	}
	if msg.Height <= 0 {
		return fmt.Errorf("upgrade height must be > 0")
	}
	return nil
}

func (msg MsgAttachUpgradePlan) Reset()         {}
func (msg MsgAttachUpgradePlan) String() string { return fmt.Sprintf("MsgAttachUpgradePlan{%s}", msg.LipId) }
func (msg MsgAttachUpgradePlan) ProtoMessage()  {}

func (msg MsgAttachUpgradePlanResponse) Reset()         {}
func (msg MsgAttachUpgradePlanResponse) String() string { return "MsgAttachUpgradePlanResponse{}" }
func (msg MsgAttachUpgradePlanResponse) ProtoMessage()  {}

// --- Interface assertions ---

var _ sdk.Msg = (*MsgUpdateParams)(nil)
var _ sdk.Msg = (*MsgAttachUpgradePlan)(nil)
