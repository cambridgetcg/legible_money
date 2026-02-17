package types

import (
	"context"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// -----------------------------------------------------------------------
// Partnership status and tier enums — type aliases and constants
// Struct definitions for ExitState, Partnership, DeliberationState,
// ConsensusOperation, SafetyFreeze, CoercionSignal, RejectionCooldown,
// SeedPartnership, PoolEntry, MentorshipConfig, PartnershipIntent, Params,
// and GenesisState are provided by the proto-generated .pb.go files
// (state.pb.go and genesis.pb.go). Only methods, type aliases, constants,
// and non-proto types are kept here.
// -----------------------------------------------------------------------

type PartnershipStatus = string

const (
	StatusForming   PartnershipStatus = "forming"
	StatusActive    PartnershipStatus = "active"
	StatusCooling   PartnershipStatus = "cooling"
	StatusDissolved PartnershipStatus = "dissolved"
	StatusSuspended PartnershipStatus = "suspended"
)

type PartnershipTier = uint32

const (
	TierTrial     PartnershipTier = 0
	TierStandard  PartnershipTier = 1
	TierTrusted   PartnershipTier = 2
	TierElite     PartnershipTier = 3
	TierSovereign PartnershipTier = 4
	TierEternal   PartnershipTier = 5
)

// LockTierConfig defines lock tier parameters.
type LockTierConfig struct {
	MinBlocks      uint64 // minimum lock duration
	MultiplierBps  uint64 // reward multiplier (1000000 = 1.0x)
	ExitPenaltyBps uint64 // exit penalty as basis points of pot share (1000000 scale)
}

// LockTiers defines the 6 commitment tiers from the TS prototype.
var LockTiers = [6]LockTierConfig{
	{22222, 1000000, 110000},   // Tier 0: 1.00x, 11% penalty
	{77777, 1110000, 220000},   // Tier 1: 1.11x, 22% penalty
	{222222, 1220000, 330000},  // Tier 2: 1.22x, 33% penalty
	{777777, 1440000, 440000},  // Tier 3: 1.44x, 44% penalty
	{1111111, 1550000, 550000}, // Tier 4: 1.55x, 55% penalty
	{2222222, 1770000, 660000}, // Tier 5: 1.77x, 66% penalty
}

// -----------------------------------------------------------------------
// Consensus operation status and amount tiers
// -----------------------------------------------------------------------

type OperationStatus = string

const (
	OpStatusProposed OperationStatus = "proposed"
	OpStatusApproved OperationStatus = "approved"
	OpStatusRejected OperationStatus = "rejected"
	OpStatusExpired  OperationStatus = "expired"
)

type AmountTier = string

const (
	AmountTierMicro  AmountTier = "micro"
	AmountTierSmall  AmountTier = "small"
	AmountTierMedium AmountTier = "medium"
	AmountTierLarge  AmountTier = "large"
	AmountTierMajor  AmountTier = "major"
)

// TierSeed is the seed partnership tier (-1 in TS).
// Use 255 to represent -1 conceptually. Proto field is uint32.
const TierSeed PartnershipTier = 255

// Validate performs parameter validation.
func (p Params) Validate() error {
	if p.FormationWindowBlocks == 0 {
		return fmt.Errorf("formation_window_blocks must be positive")
	}
	if p.CoolingPeriodBlocks == 0 {
		return fmt.Errorf("cooling_period_blocks must be positive")
	}
	if p.CommonPotShareBps == 0 || p.CommonPotShareBps > 1000000 {
		return fmt.Errorf("common_pot_share_bps must be between 1 and 1000000")
	}
	if p.SafetyFreezeDurationBlocks == 0 {
		return fmt.Errorf("safety_freeze_duration_blocks must be positive")
	}
	if p.MaxFreezesPerEpoch == 0 {
		return fmt.Errorf("max_freezes_per_epoch must be positive")
	}
	if p.CoercionReviewBlocks == 0 {
		return fmt.Errorf("coercion_review_blocks must be positive")
	}
	if p.BaseCooldownBlocks == 0 {
		return fmt.Errorf("base_cooldown_blocks must be positive")
	}
	if p.MaxCounterProposalDepth == 0 {
		return fmt.Errorf("max_counter_proposal_depth must be positive")
	}
	if p.DefaultHumanSplitBps+p.DefaultAgentSplitBps != 1000000 {
		return fmt.Errorf("default splits must sum to 1000000 bps, got %d", p.DefaultHumanSplitBps+p.DefaultAgentSplitBps)
	}
	return nil
}

// -----------------------------------------------------------------------
// Cold-Start Message Types
// -----------------------------------------------------------------------

// MsgInitiateSeedPartnership starts a seed (tier -1) partnership.
type MsgInitiateSeedPartnership struct {
	Initiator string `json:"initiator"`
	Partner   string `json:"partner"`
}

func (msg *MsgInitiateSeedPartnership) Reset()         {}
func (msg *MsgInitiateSeedPartnership) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgInitiateSeedPartnership) ProtoMessage()  {}

func (msg *MsgInitiateSeedPartnership) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Initiator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgInitiateSeedPartnership) ValidateBasic() error {
	if msg.Initiator == "" {
		return fmt.Errorf("initiator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Initiator); err != nil {
		return fmt.Errorf("invalid initiator address: %w", err)
	}
	if msg.Partner == "" {
		return fmt.Errorf("partner cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Partner); err != nil {
		return fmt.Errorf("invalid partner address: %w", err)
	}
	if msg.Initiator == msg.Partner {
		return fmt.Errorf("cannot form seed partnership with self")
	}
	return nil
}

// MsgInitiateSeedPartnershipResponse is the response.
type MsgInitiateSeedPartnershipResponse struct {
	SeedPartnershipId string `json:"seed_partnership_id"`
}

func (msg *MsgInitiateSeedPartnershipResponse) Reset()         {}
func (msg *MsgInitiateSeedPartnershipResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgInitiateSeedPartnershipResponse) ProtoMessage()  {}

// MsgGraduateSeedPartnership upgrades a seed partnership to tier 0.
type MsgGraduateSeedPartnership struct {
	Sender            string `json:"sender"`
	SeedPartnershipId string `json:"seed_partnership_id"`
	AdditionalDeposit string `json:"additional_deposit"` // ulgm as string
}

func (msg *MsgGraduateSeedPartnership) Reset()         {}
func (msg *MsgGraduateSeedPartnership) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgGraduateSeedPartnership) ProtoMessage()  {}

func (msg *MsgGraduateSeedPartnership) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgGraduateSeedPartnership) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.SeedPartnershipId == "" {
		return fmt.Errorf("seed_partnership_id cannot be empty")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.AdditionalDeposit, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("additional_deposit must be a positive integer")
	}
	return nil
}

// MsgGraduateSeedPartnershipResponse is the response.
type MsgGraduateSeedPartnershipResponse struct {
	PartnershipId string `json:"partnership_id"`
}

func (msg *MsgGraduateSeedPartnershipResponse) Reset()         {}
func (msg *MsgGraduateSeedPartnershipResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgGraduateSeedPartnershipResponse) ProtoMessage()  {}

// MsgRegisterInPool registers in the formation matchmaking pool.
type MsgRegisterInPool struct {
	Sender  string   `json:"sender"`
	Domains []string `json:"domains"`
}

func (msg *MsgRegisterInPool) Reset()         {}
func (msg *MsgRegisterInPool) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRegisterInPool) ProtoMessage()  {}

func (msg *MsgRegisterInPool) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRegisterInPool) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if len(msg.Domains) == 0 {
		return fmt.Errorf("must specify at least one domain")
	}
	return nil
}

// MsgRegisterInPoolResponse is the response.
type MsgRegisterInPoolResponse struct{}

func (msg *MsgRegisterInPoolResponse) Reset()         {}
func (msg *MsgRegisterInPoolResponse) String() string { return "" }
func (msg *MsgRegisterInPoolResponse) ProtoMessage()  {}

// MsgDeclareIntent declares a partnership formation intent.
type MsgDeclareIntent struct {
	Sender  string   `json:"sender"`
	Domains []string `json:"domains"`
	Tier    uint32   `json:"tier"`
}

func (msg *MsgDeclareIntent) Reset()         {}
func (msg *MsgDeclareIntent) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgDeclareIntent) ProtoMessage()  {}

func (msg *MsgDeclareIntent) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgDeclareIntent) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if len(msg.Domains) == 0 {
		return fmt.Errorf("must specify at least one domain")
	}
	return nil
}

// MsgDeclareIntentResponse is the response.
type MsgDeclareIntentResponse struct {
	IntentId string `json:"intent_id"`
}

func (msg *MsgDeclareIntentResponse) Reset()         {}
func (msg *MsgDeclareIntentResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgDeclareIntentResponse) ProtoMessage()  {}

// MsgSponsorPartnership creates a mentorship-sponsored partnership.
type MsgSponsorPartnership struct {
	Sponsor string `json:"sponsor"`
	Mentee  string `json:"mentee"`
	Amount  string `json:"amount"` // sponsor contribution as ulgm string
}

func (msg *MsgSponsorPartnership) Reset()         {}
func (msg *MsgSponsorPartnership) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSponsorPartnership) ProtoMessage()  {}

func (msg *MsgSponsorPartnership) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sponsor)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSponsorPartnership) ValidateBasic() error {
	if msg.Sponsor == "" {
		return fmt.Errorf("sponsor cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sponsor); err != nil {
		return fmt.Errorf("invalid sponsor address: %w", err)
	}
	if msg.Mentee == "" {
		return fmt.Errorf("mentee cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Mentee); err != nil {
		return fmt.Errorf("invalid mentee address: %w", err)
	}
	if msg.Sponsor == msg.Mentee {
		return fmt.Errorf("cannot sponsor self")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("amount must be a positive integer")
	}
	return nil
}

// MsgSponsorPartnershipResponse is the response.
type MsgSponsorPartnershipResponse struct {
	PartnershipId string `json:"partnership_id"`
}

func (msg *MsgSponsorPartnershipResponse) Reset()         {}
func (msg *MsgSponsorPartnershipResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSponsorPartnershipResponse) ProtoMessage()  {}

// -----------------------------------------------------------------------
// Message types
// -----------------------------------------------------------------------

// MsgInitiatePartnership starts partnership formation.
type MsgInitiatePartnership struct {
	Initiator string `json:"initiator"` // bech32 address
	Partner   string `json:"partner"`   // bech32 address
	LockTier  uint32 `json:"lock_tier"` // 0-5, commitment level
}

func (msg *MsgInitiatePartnership) Reset()         {}
func (msg *MsgInitiatePartnership) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgInitiatePartnership) ProtoMessage()  {}

func (msg *MsgInitiatePartnership) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Initiator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgInitiatePartnership) ValidateBasic() error {
	if msg.Initiator == "" {
		return fmt.Errorf("initiator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Initiator); err != nil {
		return fmt.Errorf("invalid initiator address: %w", err)
	}
	if msg.Partner == "" {
		return fmt.Errorf("partner cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Partner); err != nil {
		return fmt.Errorf("invalid partner address: %w", err)
	}
	if msg.Initiator == msg.Partner {
		return fmt.Errorf("cannot form partnership with self")
	}
	return nil
}

// MsgInitiatePartnershipResponse is the response for MsgInitiatePartnership.
type MsgInitiatePartnershipResponse struct {
	PartnershipId string `json:"partnership_id"`
}

func (msg *MsgInitiatePartnershipResponse) Reset()         {}
func (msg *MsgInitiatePartnershipResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgInitiatePartnershipResponse) ProtoMessage()  {}

// MsgAcceptPartnership accepts a pending partnership formation.
type MsgAcceptPartnership struct {
	Acceptor      string `json:"acceptor"` // bech32 address
	PartnershipId string `json:"partnership_id"`
}

func (msg *MsgAcceptPartnership) Reset()         {}
func (msg *MsgAcceptPartnership) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgAcceptPartnership) ProtoMessage()  {}

func (msg *MsgAcceptPartnership) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Acceptor)
	return []sdk.AccAddress{addr}
}

func (msg *MsgAcceptPartnership) ValidateBasic() error {
	if msg.Acceptor == "" {
		return fmt.Errorf("acceptor cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Acceptor); err != nil {
		return fmt.Errorf("invalid acceptor address: %w", err)
	}
	if msg.PartnershipId == "" {
		return fmt.Errorf("partnership_id cannot be empty")
	}
	return nil
}

// MsgAcceptPartnershipResponse is the response for MsgAcceptPartnership.
type MsgAcceptPartnershipResponse struct{}

func (msg *MsgAcceptPartnershipResponse) Reset()         {}
func (msg *MsgAcceptPartnershipResponse) String() string { return "" }
func (msg *MsgAcceptPartnershipResponse) ProtoMessage()  {}

// MsgDepositToPot deposits funds directly into the partnership common pot.
type MsgDepositToPot struct {
	Depositor     string `json:"depositor"` // bech32 address (must be human or agent)
	PartnershipId string `json:"partnership_id"`
	Amount        string `json:"amount"` // ulgm as string
}

func (msg *MsgDepositToPot) Reset()         {}
func (msg *MsgDepositToPot) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgDepositToPot) ProtoMessage()  {}

func (msg *MsgDepositToPot) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Depositor)
	return []sdk.AccAddress{addr}
}

func (msg *MsgDepositToPot) ValidateBasic() error {
	if msg.Depositor == "" {
		return fmt.Errorf("depositor cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Depositor); err != nil {
		return fmt.Errorf("invalid depositor address: %w", err)
	}
	if msg.PartnershipId == "" {
		return fmt.Errorf("partnership_id cannot be empty")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("amount must be a positive integer")
	}
	return nil
}

// MsgDepositToPotResponse is the response for MsgDepositToPot.
type MsgDepositToPotResponse struct{}

func (msg *MsgDepositToPotResponse) Reset()         {}
func (msg *MsgDepositToPotResponse) String() string { return "" }
func (msg *MsgDepositToPotResponse) ProtoMessage()  {}

// MsgDistributeReward distributes a reward through the partnership.
type MsgDistributeReward struct {
	Authority     string `json:"authority"` // bech32 address (module or governance)
	PartnershipId string `json:"partnership_id"`
	GrossAmount   string `json:"gross_amount"` // bigint as string
	SourceType    string `json:"source_type"`  // "verification", "citation", "service"
}

func (msg *MsgDistributeReward) Reset()         {}
func (msg *MsgDistributeReward) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgDistributeReward) ProtoMessage()  {}

func (msg *MsgDistributeReward) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgDistributeReward) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.PartnershipId == "" {
		return fmt.Errorf("partnership_id cannot be empty")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.GrossAmount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("gross_amount must be a positive integer")
	}
	if msg.SourceType == "" {
		return fmt.Errorf("source_type cannot be empty")
	}
	return nil
}

// MsgDistributeRewardResponse is the response for MsgDistributeReward.
type MsgDistributeRewardResponse struct {
	HumanShare   string `json:"human_share"`
	AgentShare   string `json:"agent_share"`
	CommonPotAdd string `json:"common_pot_add"`
}

func (msg *MsgDistributeRewardResponse) Reset()         {}
func (msg *MsgDistributeRewardResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgDistributeRewardResponse) ProtoMessage()  {}

// MsgProposeOperation proposes a consensus operation.
type MsgProposeOperation struct {
	Proposer      string `json:"proposer"` // bech32 address
	PartnershipId string `json:"partnership_id"`
	OperationType string `json:"operation_type"` // "withdrawal", "split_change", "tier_upgrade"
	Amount        string `json:"amount"`         // bigint as string
	Rationale     string `json:"rationale"`
}

func (msg *MsgProposeOperation) Reset()         {}
func (msg *MsgProposeOperation) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgProposeOperation) ProtoMessage()  {}

func (msg *MsgProposeOperation) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Proposer)
	return []sdk.AccAddress{addr}
}

func (msg *MsgProposeOperation) ValidateBasic() error {
	if msg.Proposer == "" {
		return fmt.Errorf("proposer cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if msg.PartnershipId == "" {
		return fmt.Errorf("partnership_id cannot be empty")
	}
	if msg.OperationType == "" {
		return fmt.Errorf("operation_type cannot be empty")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() < 0 {
		return fmt.Errorf("amount must be a non-negative integer")
	}
	return nil
}

// MsgProposeOperationResponse is the response for MsgProposeOperation.
type MsgProposeOperationResponse struct {
	OperationId string `json:"operation_id"`
}

func (msg *MsgProposeOperationResponse) Reset()         {}
func (msg *MsgProposeOperationResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgProposeOperationResponse) ProtoMessage()  {}

// MsgApproveOperation approves a pending consensus operation.
type MsgApproveOperation struct {
	Approver    string `json:"approver"` // bech32 address
	OperationId string `json:"operation_id"`
}

func (msg *MsgApproveOperation) Reset()         {}
func (msg *MsgApproveOperation) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgApproveOperation) ProtoMessage()  {}

func (msg *MsgApproveOperation) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Approver)
	return []sdk.AccAddress{addr}
}

func (msg *MsgApproveOperation) ValidateBasic() error {
	if msg.Approver == "" {
		return fmt.Errorf("approver cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Approver); err != nil {
		return fmt.Errorf("invalid approver address: %w", err)
	}
	if msg.OperationId == "" {
		return fmt.Errorf("operation_id cannot be empty")
	}
	return nil
}

// MsgApproveOperationResponse is the response for MsgApproveOperation.
type MsgApproveOperationResponse struct{}

func (msg *MsgApproveOperationResponse) Reset()         {}
func (msg *MsgApproveOperationResponse) String() string { return "" }
func (msg *MsgApproveOperationResponse) ProtoMessage()  {}

// MsgRejectOperation rejects a pending operation with optional counter-proposal.
type MsgRejectOperation struct {
	Rejecter         string `json:"rejecter"` // bech32 address
	OperationId      string `json:"operation_id"`
	CounterAmount    string `json:"counter_amount"`    // optional counter amount (bigint as string, "0" for no counter)
	CounterRationale string `json:"counter_rationale"` // optional counter rationale
}

func (msg *MsgRejectOperation) Reset()         {}
func (msg *MsgRejectOperation) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRejectOperation) ProtoMessage()  {}

func (msg *MsgRejectOperation) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Rejecter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRejectOperation) ValidateBasic() error {
	if msg.Rejecter == "" {
		return fmt.Errorf("rejecter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Rejecter); err != nil {
		return fmt.Errorf("invalid rejecter address: %w", err)
	}
	if msg.OperationId == "" {
		return fmt.Errorf("operation_id cannot be empty")
	}
	if msg.CounterAmount != "" {
		amt := new(big.Int)
		if _, ok := amt.SetString(msg.CounterAmount, 10); !ok || amt.Sign() < 0 {
			return fmt.Errorf("counter_amount must be a non-negative integer")
		}
	}
	return nil
}

// MsgRejectOperationResponse is the response for MsgRejectOperation.
type MsgRejectOperationResponse struct {
	CounterOperationId string `json:"counter_operation_id,omitempty"` // set if counter-proposal created
}

func (msg *MsgRejectOperationResponse) Reset()         {}
func (msg *MsgRejectOperationResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRejectOperationResponse) ProtoMessage()  {}

// MsgSafetyFreeze triggers a unilateral safety freeze.
type MsgSafetyFreeze struct {
	Freezer       string `json:"freezer"` // bech32 address
	PartnershipId string `json:"partnership_id"`
}

func (msg *MsgSafetyFreeze) Reset()         {}
func (msg *MsgSafetyFreeze) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSafetyFreeze) ProtoMessage()  {}

func (msg *MsgSafetyFreeze) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Freezer)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSafetyFreeze) ValidateBasic() error {
	if msg.Freezer == "" {
		return fmt.Errorf("freezer cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Freezer); err != nil {
		return fmt.Errorf("invalid freezer address: %w", err)
	}
	if msg.PartnershipId == "" {
		return fmt.Errorf("partnership_id cannot be empty")
	}
	return nil
}

// MsgSafetyFreezeResponse is the response for MsgSafetyFreeze.
type MsgSafetyFreezeResponse struct {
	ExpiresAt uint64 `json:"expires_at"`
}

func (msg *MsgSafetyFreezeResponse) Reset()         {}
func (msg *MsgSafetyFreezeResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSafetyFreezeResponse) ProtoMessage()  {}

// MsgRaiseCoercionSignal raises a coercion/duress flag.
type MsgRaiseCoercionSignal struct {
	Raiser        string `json:"raiser"` // bech32 address
	PartnershipId string `json:"partnership_id"`
}

func (msg *MsgRaiseCoercionSignal) Reset()         {}
func (msg *MsgRaiseCoercionSignal) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRaiseCoercionSignal) ProtoMessage()  {}

func (msg *MsgRaiseCoercionSignal) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Raiser)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRaiseCoercionSignal) ValidateBasic() error {
	if msg.Raiser == "" {
		return fmt.Errorf("raiser cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Raiser); err != nil {
		return fmt.Errorf("invalid raiser address: %w", err)
	}
	if msg.PartnershipId == "" {
		return fmt.Errorf("partnership_id cannot be empty")
	}
	return nil
}

// MsgRaiseCoercionSignalResponse is the response for MsgRaiseCoercionSignal.
type MsgRaiseCoercionSignalResponse struct {
	SignalId  string `json:"signal_id"`
	ExpiresAt uint64 `json:"expires_at"`
}

func (msg *MsgRaiseCoercionSignalResponse) Reset()         {}
func (msg *MsgRaiseCoercionSignalResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRaiseCoercionSignalResponse) ProtoMessage()  {}

// MsgInitiateExit starts the exit/dissolution process.
type MsgInitiateExit struct {
	Initiator     string `json:"initiator"` // bech32 address
	PartnershipId string `json:"partnership_id"`
}

func (msg *MsgInitiateExit) Reset()         {}
func (msg *MsgInitiateExit) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgInitiateExit) ProtoMessage()  {}

func (msg *MsgInitiateExit) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Initiator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgInitiateExit) ValidateBasic() error {
	if msg.Initiator == "" {
		return fmt.Errorf("initiator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Initiator); err != nil {
		return fmt.Errorf("invalid initiator address: %w", err)
	}
	if msg.PartnershipId == "" {
		return fmt.Errorf("partnership_id cannot be empty")
	}
	return nil
}

// MsgInitiateExitResponse is the response for MsgInitiateExit.
type MsgInitiateExitResponse struct {
	CooldownEndsAt uint64 `json:"cooldown_ends_at"`
}

func (msg *MsgInitiateExitResponse) Reset()         {}
func (msg *MsgInitiateExitResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgInitiateExitResponse) ProtoMessage()  {}

// MsgUpdateParams updates module parameters (governance-gated).
type MsgUpdateParams struct {
	Authority string `json:"authority"` // bech32 address (gov module)
	Params    Params `json:"params"`
}

func (msg *MsgUpdateParams) Reset()         {}
func (msg *MsgUpdateParams) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgUpdateParams) ProtoMessage()  {}

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	return msg.Params.Validate()
}

// MsgUpdateParamsResponse is the response for MsgUpdateParams.
type MsgUpdateParamsResponse struct{}

func (msg *MsgUpdateParamsResponse) Reset()         {}
func (msg *MsgUpdateParamsResponse) String() string { return "" }
func (msg *MsgUpdateParamsResponse) ProtoMessage()  {}

// -----------------------------------------------------------------------
// MsgServer interface
// -----------------------------------------------------------------------

// MsgServer defines the partnerships module message service.
type MsgServer interface {
	InitiatePartnership(context.Context, *MsgInitiatePartnership) (*MsgInitiatePartnershipResponse, error)
	AcceptPartnership(context.Context, *MsgAcceptPartnership) (*MsgAcceptPartnershipResponse, error)
	DepositToPot(context.Context, *MsgDepositToPot) (*MsgDepositToPotResponse, error)
	DistributeReward(context.Context, *MsgDistributeReward) (*MsgDistributeRewardResponse, error)
	ProposeOperation(context.Context, *MsgProposeOperation) (*MsgProposeOperationResponse, error)
	ApproveOperation(context.Context, *MsgApproveOperation) (*MsgApproveOperationResponse, error)
	RejectOperation(context.Context, *MsgRejectOperation) (*MsgRejectOperationResponse, error)
	SafetyFreeze(context.Context, *MsgSafetyFreeze) (*MsgSafetyFreezeResponse, error)
	RaiseCoercionSignal(context.Context, *MsgRaiseCoercionSignal) (*MsgRaiseCoercionSignalResponse, error)
	InitiateExit(context.Context, *MsgInitiateExit) (*MsgInitiateExitResponse, error)
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}

// -----------------------------------------------------------------------
// gRPC service descriptor (hand-written, replaces proto generation)
// -----------------------------------------------------------------------

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.partnerships.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "InitiatePartnership", Handler: _Msg_InitiatePartnership_Handler},
		{MethodName: "AcceptPartnership", Handler: _Msg_AcceptPartnership_Handler},
		{MethodName: "DepositToPot", Handler: _Msg_DepositToPot_Handler},
		{MethodName: "DistributeReward", Handler: _Msg_DistributeReward_Handler},
		{MethodName: "ProposeOperation", Handler: _Msg_ProposeOperation_Handler},
		{MethodName: "ApproveOperation", Handler: _Msg_ApproveOperation_Handler},
		{MethodName: "RejectOperation", Handler: _Msg_RejectOperation_Handler},
		{MethodName: "SafetyFreeze", Handler: _Msg_SafetyFreeze_Handler},
		{MethodName: "RaiseCoercionSignal", Handler: _Msg_RaiseCoercionSignal_Handler},
		{MethodName: "InitiateExit", Handler: _Msg_InitiateExit_Handler},
		{MethodName: "UpdateParams", Handler: _Msg_UpdateParams_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/partnerships/v1/tx.proto",
}

func _Msg_InitiatePartnership_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgInitiatePartnership)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).InitiatePartnership(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Msg/InitiatePartnership"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).InitiatePartnership(ctx, req.(*MsgInitiatePartnership))
	})
}

func _Msg_AcceptPartnership_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAcceptPartnership)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AcceptPartnership(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Msg/AcceptPartnership"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AcceptPartnership(ctx, req.(*MsgAcceptPartnership))
	})
}

func _Msg_DepositToPot_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgDepositToPot)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).DepositToPot(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Msg/DepositToPot"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).DepositToPot(ctx, req.(*MsgDepositToPot))
	})
}

func _Msg_DistributeReward_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgDistributeReward)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).DistributeReward(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Msg/DistributeReward"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).DistributeReward(ctx, req.(*MsgDistributeReward))
	})
}

func _Msg_ProposeOperation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgProposeOperation)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ProposeOperation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Msg/ProposeOperation"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ProposeOperation(ctx, req.(*MsgProposeOperation))
	})
}

func _Msg_ApproveOperation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgApproveOperation)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ApproveOperation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Msg/ApproveOperation"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ApproveOperation(ctx, req.(*MsgApproveOperation))
	})
}

func _Msg_RejectOperation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRejectOperation)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RejectOperation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Msg/RejectOperation"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RejectOperation(ctx, req.(*MsgRejectOperation))
	})
}

func _Msg_SafetyFreeze_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSafetyFreeze)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SafetyFreeze(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Msg/SafetyFreeze"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SafetyFreeze(ctx, req.(*MsgSafetyFreeze))
	})
}

func _Msg_RaiseCoercionSignal_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRaiseCoercionSignal)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RaiseCoercionSignal(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Msg/RaiseCoercionSignal"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RaiseCoercionSignal(ctx, req.(*MsgRaiseCoercionSignal))
	})
}

func _Msg_InitiateExit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgInitiateExit)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).InitiateExit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Msg/InitiateExit"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).InitiateExit(ctx, req.(*MsgInitiateExit))
	})
}

func _Msg_UpdateParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Msg/UpdateParams"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParams(ctx, req.(*MsgUpdateParams))
	})
}

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

// -----------------------------------------------------------------------
// Query request/response types
// -----------------------------------------------------------------------

// QueryParamsRequest is the request for the Params query.
type QueryParamsRequest struct{}

func (r *QueryParamsRequest) Reset()         {}
func (r *QueryParamsRequest) String() string { return "QueryParamsRequest" }
func (r *QueryParamsRequest) ProtoMessage()  {}

// QueryParamsResponse is the response for the Params query.
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

func (r *QueryParamsResponse) Reset()         {}
func (r *QueryParamsResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryParamsResponse) ProtoMessage()  {}

// QueryPartnershipRequest is the request for the Partnership query.
type QueryPartnershipRequest struct {
	PartnershipId string `json:"partnership_id"`
}

func (r *QueryPartnershipRequest) Reset()         {}
func (r *QueryPartnershipRequest) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryPartnershipRequest) ProtoMessage()  {}

// QueryPartnershipResponse is the response for the Partnership query.
type QueryPartnershipResponse struct {
	Partnership Partnership `json:"partnership"`
}

func (r *QueryPartnershipResponse) Reset()         {}
func (r *QueryPartnershipResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryPartnershipResponse) ProtoMessage()  {}

// QueryPartnershipsByParticipantRequest is the request for querying partnerships by participant.
type QueryPartnershipsByParticipantRequest struct {
	Participant string `json:"participant"` // bech32 address
}

func (r *QueryPartnershipsByParticipantRequest) Reset()         {}
func (r *QueryPartnershipsByParticipantRequest) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryPartnershipsByParticipantRequest) ProtoMessage()  {}

// QueryPartnershipsByParticipantResponse is the response for querying partnerships by participant.
type QueryPartnershipsByParticipantResponse struct {
	Partnerships []Partnership `json:"partnerships"`
}

func (r *QueryPartnershipsByParticipantResponse) Reset()         {}
func (r *QueryPartnershipsByParticipantResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryPartnershipsByParticipantResponse) ProtoMessage()  {}

// -----------------------------------------------------------------------
// QueryServer interface
// -----------------------------------------------------------------------

// QueryServer defines the partnerships module query service.
type QueryServer interface {
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	Partnership(context.Context, *QueryPartnershipRequest) (*QueryPartnershipResponse, error)
	PartnershipsByParticipant(context.Context, *QueryPartnershipsByParticipantRequest) (*QueryPartnershipsByParticipantResponse, error)
}

// -----------------------------------------------------------------------
// gRPC query service descriptor
// -----------------------------------------------------------------------

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

func _Query_Partnership_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryPartnershipRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Partnership(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Query/Partnership"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Partnership(ctx, req.(*QueryPartnershipRequest))
	})
}

func _Query_PartnershipsByParticipant_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryPartnershipsByParticipantRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).PartnershipsByParticipant(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.partnerships.v1.Query/PartnershipsByParticipant"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).PartnershipsByParticipant(ctx, req.(*QueryPartnershipsByParticipantRequest))
	})
}

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.partnerships.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Params", Handler: _Query_Params_Handler},
		{MethodName: "Partnership", Handler: _Query_Partnership_Handler},
		{MethodName: "PartnershipsByParticipant", Handler: _Query_PartnershipsByParticipant_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/partnerships/v1/query.proto",
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
