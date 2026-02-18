package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// -----------------------------------------------------------------------
// Partnership status and tier enums — type aliases and constants
// Struct definitions for ExitState, Partnership, DeliberationState,
// ConsensusOperation, SafetyFreeze, CoercionSignal, RejectionCooldown,
// SeedPartnership, PoolEntry, MentorshipConfig, PartnershipIntent, Params,
// and GenesisState are provided by the proto-generated .pb.go files
// (state.pb.go and genesis.pb.go). Only methods, type aliases, constants,
// and non-proto types are kept here.
//
// Msg types, Response types, MsgServer interface, gRPC service descriptors,
// handler functions, and RegisterMsgServer are provided by the proto-generated
// tx.pb.go and tx_grpc.pb.go files. Only GetSigners() and ValidateBasic()
// methods are kept here.
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
func (p *Params) Validate() error {
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
// GetSigners and ValidateBasic methods for proto-generated Msg types
// -----------------------------------------------------------------------

// --- Cold-start messages ---

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

// --- Core messages ---

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

// --- Deliberation messages ---

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

// --- Safety & exit messages ---

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

// --- Governance messages ---

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
	if msg.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return msg.Params.Validate()
}

// Query types (QueryServer, QueryParamsRequest/Response, QueryPartnershipRequest/Response,
// QueryPartnershipsByParticipantRequest/Response), gRPC service descriptor, handlers, and
// RegisterQueryServer are now provided by the proto-generated query.pb.go and query_grpc.pb.go files.
