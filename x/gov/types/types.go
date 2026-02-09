package types

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// --- Enums ---

// LIPCategory determines stake requirements and review thresholds.
type LIPCategory string

const (
	CategoryConsensus  LIPCategory = "consensus"
	CategoryEconomics  LIPCategory = "economics"
	CategoryInterface  LIPCategory = "interface"
	CategoryProcess    LIPCategory = "process"
)

// LIPStage represents the current stage in a LIP's lifecycle.
type LIPStage string

const (
	StageDraft     LIPStage = "draft"
	StageReview    LIPStage = "review"
	StageLastCall  LIPStage = "last_call"
	StageVoting    LIPStage = "voting"
	StageAccepted  LIPStage = "accepted"
	StageRejected  LIPStage = "rejected"
	StageWithdrawn LIPStage = "withdrawn"
)

// VoteDirection represents how a voter voted.
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

// --- Core Types ---

// LIP represents a Legible Improvement Proposal.
type LIP struct {
	Id             string      `json:"id"`
	Title          string      `json:"title"`
	Abstract       string      `json:"abstract"`
	Category       LIPCategory `json:"category"`
	Proposer       string      `json:"proposer"`
	Stage          LIPStage    `json:"stage"`
	StakedAmount   string      `json:"staked_amount"`
	StakeHolder    string      `json:"stake_holder"`
	CreatedAtBlock uint64      `json:"created_at_block"`
	UpdatedAtBlock uint64      `json:"updated_at_block"`
	VotingEndsAt   uint64      `json:"voting_ends_at"`
	SupportWeight  string      `json:"support_weight"`
	OpposeWeight   string      `json:"oppose_weight"`
	AbstainWeight  string      `json:"abstain_weight"`
	UniqueVoters   uint32      `json:"unique_voters"`
	QuorumMet      bool        `json:"quorum_met"`
}

// QuadraticVote records a voter's quadratic vote on a proposal.
type QuadraticVote struct {
	VoteId         string        `json:"vote_id"`
	Voter          string        `json:"voter"`
	ProposalId     string        `json:"proposal_id"`
	Direction      VoteDirection `json:"direction"`
	Votes          uint32        `json:"votes"`
	Cost           string        `json:"cost"`
	CreatedAtBlock uint64        `json:"created_at_block"`
}

// DisbursementProposal represents a treasury spending proposal.
type DisbursementProposal struct {
	Id              string               `json:"id"`
	Title           string               `json:"title"`
	Description     string               `json:"description"`
	Category        DisbursementCategory `json:"category"`
	Proposer        string               `json:"proposer"`
	RecipientAddr   string               `json:"recipient_addr"`
	Amount          string               `json:"amount"`
	Stage           DisbursementStage    `json:"stage"`
	StakedAmount    string               `json:"staked_amount"`
	StakeHolder     string               `json:"stake_holder"`
	CreatedAtBlock  uint64               `json:"created_at_block"`
	ApprovedAtBlock uint64               `json:"approved_at_block,omitempty"`
	VotingEndsAt    uint64               `json:"voting_ends_at,omitempty"`
	TimelockExpiry  uint64               `json:"timelock_expiry,omitempty"`
	SupportWeight   string               `json:"support_weight"`
	OpposeWeight    string               `json:"oppose_weight"`
	AbstainWeight   string               `json:"abstain_weight"`
	UniqueVoters    uint32               `json:"unique_voters"`
	QuorumMet       bool                 `json:"quorum_met"`
}

// --- Params ---

// Params defines governance module parameters.
type Params struct {
	VotingPeriodBlocks         uint64 `json:"voting_period_blocks"`
	ReviewPeriodBlocks         uint64 `json:"review_period_blocks"`
	LastCallPeriodBlocks       uint64 `json:"last_call_period_blocks"`
	BaseCostPerVote            string `json:"base_cost_per_vote"`
	MaxVotesPerVoter           uint32 `json:"max_votes_per_voter"`
	SupportThresholdBps        uint64 `json:"support_threshold_bps"`
	QuorumThresholdBps         uint64 `json:"quorum_threshold_bps"`
	DisbursementStake          string `json:"disbursement_stake"`
	DisbursementTimelockBlocks uint64 `json:"disbursement_timelock_blocks"`
	DisbursementSupportBps     uint64 `json:"disbursement_support_bps"`
	DisbursementQuorumBps      uint64 `json:"disbursement_quorum_bps"`
	LIPStakeConsensus          string `json:"lip_stake_consensus"`
	LIPStakeEconomics          string `json:"lip_stake_economics"`
	LIPStakeInterface          string `json:"lip_stake_interface"`
	LIPStakeProcess            string `json:"lip_stake_process"`
}

// DefaultParams returns default governance parameters.
func DefaultParams() Params {
	return Params{
		VotingPeriodBlocks:         100000,
		ReviewPeriodBlocks:         50000,
		LastCallPeriodBlocks:       50000,
		BaseCostPerVote:            "1000000",    // 1 LGM in ulgm
		MaxVotesPerVoter:           100,
		SupportThresholdBps:        7500, // 75%
		QuorumThresholdBps:         1500, // 15%
		DisbursementStake:          "2000000000", // 2,000 LGM in ulgm
		DisbursementTimelockBlocks: 50000,
		DisbursementSupportBps:     8000, // 80%
		DisbursementQuorumBps:      2000, // 20%
		LIPStakeConsensus:          "1000000000", // 1,000 LGM in ulgm
		LIPStakeEconomics:          "800000000",  // 800 LGM in ulgm
		LIPStakeInterface:          "400000000",  // 400 LGM in ulgm
		LIPStakeProcess:            "200000000",  // 200 LGM in ulgm
	}
}

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
		return p.LIPStakeConsensus
	case CategoryEconomics:
		return p.LIPStakeEconomics
	case CategoryInterface:
		return p.LIPStakeInterface
	case CategoryProcess:
		return p.LIPStakeProcess
	default:
		return p.LIPStakeProcess
	}
}

// --- Genesis ---

// GenesisState is the state that must be provided at genesis.
type GenesisState struct {
	Params         Params                 `json:"params"`
	LIPs           []LIP                  `json:"lips"`
	Votes          []QuadraticVote        `json:"votes"`
	Disbursements  []DisbursementProposal `json:"disbursements"`
	NextLIPNumber  uint64                 `json:"next_lip_number"`
	NextDisbNumber uint64                 `json:"next_disb_number"`
}

// DefaultGenesis returns default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:         DefaultParams(),
		LIPs:           []LIP{},
		Votes:          []QuadraticVote{},
		Disbursements:  []DisbursementProposal{},
		NextLIPNumber:  1,
		NextDisbNumber: 1,
	}
}

// Validate validates genesis state.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
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
func IsTerminalLIPStage(stage LIPStage) bool {
	return stage == StageAccepted || stage == StageRejected || stage == StageWithdrawn
}

// IsTerminalDisbStage returns true if the disbursement stage is terminal.
func IsTerminalDisbStage(stage DisbursementStage) bool {
	return stage == DisbStageExecuted || stage == DisbStageRejected || stage == DisbStageCancelled
}

// --- Message Types ---

// MsgSubmitLIP submits a new Legible Improvement Proposal.
type MsgSubmitLIP struct {
	Proposer string      `json:"proposer"`
	Title    string      `json:"title"`
	Abstract string      `json:"abstract"`
	Category LIPCategory `json:"category"`
}

type MsgSubmitLIPResponse struct {
	LipId string `json:"lip_id"`
}

// MsgStakeLIP stakes on a LIP to advance it from draft to review.
type MsgStakeLIP struct {
	Sender string `json:"sender"`
	LipId  string `json:"lip_id"`
	Amount string `json:"amount"`
}

type MsgStakeLIPResponse struct{}

// MsgCastVote casts a quadratic vote on a LIP or disbursement.
type MsgCastVote struct {
	Voter      string        `json:"voter"`
	ProposalId string        `json:"proposal_id"`
	Direction  VoteDirection `json:"direction"`
	Votes      uint32        `json:"votes"`
}

type MsgCastVoteResponse struct {
	Cost string `json:"cost"`
}

// MsgAdvanceLIPStage advances a LIP to its next stage based on vote tally.
type MsgAdvanceLIPStage struct {
	Sender string `json:"sender"`
	LipId  string `json:"lip_id"`
}

type MsgAdvanceLIPStageResponse struct {
	NewStage LIPStage `json:"new_stage"`
}

// MsgWithdrawLIP withdraws a LIP (only by proposer).
type MsgWithdrawLIP struct {
	Proposer string `json:"proposer"`
	LipId    string `json:"lip_id"`
}

type MsgWithdrawLIPResponse struct{}

// MsgSubmitDisbursement submits a new treasury disbursement proposal.
type MsgSubmitDisbursement struct {
	Proposer      string               `json:"proposer"`
	Title         string               `json:"title"`
	Description   string               `json:"description"`
	Category      DisbursementCategory `json:"category"`
	RecipientAddr string               `json:"recipient_addr"`
	Amount        string               `json:"amount"`
}

type MsgSubmitDisbursementResponse struct {
	DisbursementId string `json:"disbursement_id"`
}

// MsgExecuteDisbursement executes an approved disbursement after timelock.
type MsgExecuteDisbursement struct {
	Sender         string `json:"sender"`
	DisbursementId string `json:"disbursement_id"`
}

type MsgExecuteDisbursementResponse struct{}

// MsgCancelDisbursement cancels a disbursement (only by proposer).
type MsgCancelDisbursement struct {
	Proposer       string `json:"proposer"`
	DisbursementId string `json:"disbursement_id"`
}

type MsgCancelDisbursementResponse struct{}

// MsgUpdateParams updates governance module params (authority only).
type MsgUpdateParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

type MsgUpdateParamsResponse struct{}

// --- sdk.Msg interface ---

func (msg MsgSubmitLIP) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Proposer)
	return []sdk.AccAddress{addr}
}

func (msg MsgSubmitLIP) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if strings.TrimSpace(msg.Title) == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if len(msg.Title) > 200 {
		return fmt.Errorf("title too long (max 200)")
	}
	if len(msg.Abstract) > 2000 {
		return fmt.Errorf("abstract too long (max 2000)")
	}
	if !ValidLIPCategories[msg.Category] {
		return fmt.Errorf("invalid LIP category: %s", msg.Category)
	}
	return nil
}

func (msg MsgStakeLIP) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg MsgStakeLIP) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.LipId == "" {
		return fmt.Errorf("lip_id cannot be empty")
	}
	return nil
}

func (msg MsgCastVote) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Voter)
	return []sdk.AccAddress{addr}
}

func (msg MsgCastVote) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Voter); err != nil {
		return fmt.Errorf("invalid voter address: %w", err)
	}
	if msg.ProposalId == "" {
		return fmt.Errorf("proposal_id cannot be empty")
	}
	if !ValidVoteDirections[msg.Direction] {
		return fmt.Errorf("invalid vote direction: %s", msg.Direction)
	}
	if msg.Votes == 0 {
		return fmt.Errorf("votes must be > 0")
	}
	return nil
}

func (msg MsgAdvanceLIPStage) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg MsgAdvanceLIPStage) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.LipId == "" {
		return fmt.Errorf("lip_id cannot be empty")
	}
	return nil
}

func (msg MsgWithdrawLIP) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Proposer)
	return []sdk.AccAddress{addr}
}

func (msg MsgWithdrawLIP) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if msg.LipId == "" {
		return fmt.Errorf("lip_id cannot be empty")
	}
	return nil
}

func (msg MsgSubmitDisbursement) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Proposer)
	return []sdk.AccAddress{addr}
}

func (msg MsgSubmitDisbursement) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if strings.TrimSpace(msg.Title) == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if !ValidDisbursementCategories[msg.Category] {
		return fmt.Errorf("invalid disbursement category: %s", msg.Category)
	}
	if _, err := sdk.AccAddressFromBech32(msg.RecipientAddr); err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}
	return nil
}

func (msg MsgExecuteDisbursement) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg MsgExecuteDisbursement) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.DisbursementId == "" {
		return fmt.Errorf("disbursement_id cannot be empty")
	}
	return nil
}

func (msg MsgCancelDisbursement) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Proposer)
	return []sdk.AccAddress{addr}
}

func (msg MsgCancelDisbursement) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if msg.DisbursementId == "" {
		return fmt.Errorf("disbursement_id cannot be empty")
	}
	return nil
}

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

// --- proto.Message stubs ---

func (msg MsgSubmitLIP) Reset()         {}
func (msg MsgSubmitLIP) String() string { return fmt.Sprintf("MsgSubmitLIP{%s}", msg.Proposer) }
func (msg MsgSubmitLIP) ProtoMessage()  {}

func (msg MsgSubmitLIPResponse) Reset()         {}
func (msg MsgSubmitLIPResponse) String() string { return "MsgSubmitLIPResponse{}" }
func (msg MsgSubmitLIPResponse) ProtoMessage()  {}

func (msg MsgStakeLIP) Reset()         {}
func (msg MsgStakeLIP) String() string { return fmt.Sprintf("MsgStakeLIP{%s}", msg.Sender) }
func (msg MsgStakeLIP) ProtoMessage()  {}

func (msg MsgStakeLIPResponse) Reset()         {}
func (msg MsgStakeLIPResponse) String() string { return "MsgStakeLIPResponse{}" }
func (msg MsgStakeLIPResponse) ProtoMessage()  {}

func (msg MsgCastVote) Reset()         {}
func (msg MsgCastVote) String() string { return fmt.Sprintf("MsgCastVote{%s}", msg.Voter) }
func (msg MsgCastVote) ProtoMessage()  {}

func (msg MsgCastVoteResponse) Reset()         {}
func (msg MsgCastVoteResponse) String() string { return "MsgCastVoteResponse{}" }
func (msg MsgCastVoteResponse) ProtoMessage()  {}

func (msg MsgAdvanceLIPStage) Reset()         {}
func (msg MsgAdvanceLIPStage) String() string { return fmt.Sprintf("MsgAdvanceLIPStage{%s}", msg.LipId) }
func (msg MsgAdvanceLIPStage) ProtoMessage()  {}

func (msg MsgAdvanceLIPStageResponse) Reset()         {}
func (msg MsgAdvanceLIPStageResponse) String() string { return "MsgAdvanceLIPStageResponse{}" }
func (msg MsgAdvanceLIPStageResponse) ProtoMessage()  {}

func (msg MsgWithdrawLIP) Reset()         {}
func (msg MsgWithdrawLIP) String() string { return fmt.Sprintf("MsgWithdrawLIP{%s}", msg.Proposer) }
func (msg MsgWithdrawLIP) ProtoMessage()  {}

func (msg MsgWithdrawLIPResponse) Reset()         {}
func (msg MsgWithdrawLIPResponse) String() string { return "MsgWithdrawLIPResponse{}" }
func (msg MsgWithdrawLIPResponse) ProtoMessage()  {}

func (msg MsgSubmitDisbursement) Reset()         {}
func (msg MsgSubmitDisbursement) String() string { return fmt.Sprintf("MsgSubmitDisbursement{%s}", msg.Proposer) }
func (msg MsgSubmitDisbursement) ProtoMessage()  {}

func (msg MsgSubmitDisbursementResponse) Reset()         {}
func (msg MsgSubmitDisbursementResponse) String() string { return "MsgSubmitDisbursementResponse{}" }
func (msg MsgSubmitDisbursementResponse) ProtoMessage()  {}

func (msg MsgExecuteDisbursement) Reset()         {}
func (msg MsgExecuteDisbursement) String() string { return fmt.Sprintf("MsgExecuteDisbursement{%s}", msg.DisbursementId) }
func (msg MsgExecuteDisbursement) ProtoMessage()  {}

func (msg MsgExecuteDisbursementResponse) Reset()         {}
func (msg MsgExecuteDisbursementResponse) String() string { return "MsgExecuteDisbursementResponse{}" }
func (msg MsgExecuteDisbursementResponse) ProtoMessage()  {}

func (msg MsgCancelDisbursement) Reset()         {}
func (msg MsgCancelDisbursement) String() string { return fmt.Sprintf("MsgCancelDisbursement{%s}", msg.Proposer) }
func (msg MsgCancelDisbursement) ProtoMessage()  {}

func (msg MsgCancelDisbursementResponse) Reset()         {}
func (msg MsgCancelDisbursementResponse) String() string { return "MsgCancelDisbursementResponse{}" }
func (msg MsgCancelDisbursementResponse) ProtoMessage()  {}

func (msg MsgUpdateParams) Reset()         {}
func (msg MsgUpdateParams) String() string { return "MsgUpdateParams{}" }
func (msg MsgUpdateParams) ProtoMessage()  {}

func (msg MsgUpdateParamsResponse) Reset()         {}
func (msg MsgUpdateParamsResponse) String() string { return "MsgUpdateParamsResponse{}" }
func (msg MsgUpdateParamsResponse) ProtoMessage()  {}

// proto.Message stubs for data types
func (l LIP) Reset()         {}
func (l LIP) String() string { return fmt.Sprintf("LIP{%s}", l.Id) }
func (l LIP) ProtoMessage()  {}

func (v QuadraticVote) Reset()         {}
func (v QuadraticVote) String() string { return fmt.Sprintf("QuadraticVote{%s}", v.VoteId) }
func (v QuadraticVote) ProtoMessage()  {}

func (d DisbursementProposal) Reset()         {}
func (d DisbursementProposal) String() string { return fmt.Sprintf("DisbursementProposal{%s}", d.Id) }
func (d DisbursementProposal) ProtoMessage()  {}

func (p Params) Reset()         {}
func (p Params) String() string { return "Params{}" }
func (p Params) ProtoMessage()  {}

func (gs GenesisState) Reset()         {}
func (gs GenesisState) String() string { return "GenesisState{}" }
func (gs GenesisState) ProtoMessage()  {}

// --- Interface assertions ---

var _ sdk.Msg = (*MsgSubmitLIP)(nil)
var _ sdk.Msg = (*MsgStakeLIP)(nil)
var _ sdk.Msg = (*MsgCastVote)(nil)
var _ sdk.Msg = (*MsgAdvanceLIPStage)(nil)
var _ sdk.Msg = (*MsgWithdrawLIP)(nil)
var _ sdk.Msg = (*MsgSubmitDisbursement)(nil)
var _ sdk.Msg = (*MsgExecuteDisbursement)(nil)
var _ sdk.Msg = (*MsgCancelDisbursement)(nil)
var _ sdk.Msg = (*MsgUpdateParams)(nil)

var _ proto.Message = (*MsgSubmitLIP)(nil)
var _ proto.Message = (*MsgSubmitLIPResponse)(nil)
var _ proto.Message = (*MsgStakeLIP)(nil)
var _ proto.Message = (*MsgStakeLIPResponse)(nil)
var _ proto.Message = (*MsgCastVote)(nil)
var _ proto.Message = (*MsgCastVoteResponse)(nil)
var _ proto.Message = (*MsgAdvanceLIPStage)(nil)
var _ proto.Message = (*MsgAdvanceLIPStageResponse)(nil)
var _ proto.Message = (*MsgWithdrawLIP)(nil)
var _ proto.Message = (*MsgWithdrawLIPResponse)(nil)
var _ proto.Message = (*MsgSubmitDisbursement)(nil)
var _ proto.Message = (*MsgSubmitDisbursementResponse)(nil)
var _ proto.Message = (*MsgExecuteDisbursement)(nil)
var _ proto.Message = (*MsgExecuteDisbursementResponse)(nil)
var _ proto.Message = (*MsgCancelDisbursement)(nil)
var _ proto.Message = (*MsgCancelDisbursementResponse)(nil)
var _ proto.Message = (*MsgUpdateParams)(nil)
var _ proto.Message = (*MsgUpdateParamsResponse)(nil)

var _ proto.Message = (*LIP)(nil)
var _ proto.Message = (*QuadraticVote)(nil)
var _ proto.Message = (*DisbursementProposal)(nil)
var _ proto.Message = (*Params)(nil)
var _ proto.Message = (*GenesisState)(nil)

// MsgServer interface for gRPC handlers
type MsgServer interface {
	SubmitLIP(context.Context, *MsgSubmitLIP) (*MsgSubmitLIPResponse, error)
	StakeLIP(context.Context, *MsgStakeLIP) (*MsgStakeLIPResponse, error)
	CastVote(context.Context, *MsgCastVote) (*MsgCastVoteResponse, error)
	AdvanceLIPStage(context.Context, *MsgAdvanceLIPStage) (*MsgAdvanceLIPStageResponse, error)
	WithdrawLIP(context.Context, *MsgWithdrawLIP) (*MsgWithdrawLIPResponse, error)
	SubmitDisbursement(context.Context, *MsgSubmitDisbursement) (*MsgSubmitDisbursementResponse, error)
	ExecuteDisbursement(context.Context, *MsgExecuteDisbursement) (*MsgExecuteDisbursementResponse, error)
	CancelDisbursement(context.Context, *MsgCancelDisbursement) (*MsgCancelDisbursementResponse, error)
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}
