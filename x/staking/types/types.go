package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// ValidatorTier represents the graduated tier system
type ValidatorTier uint32

const (
	TierApprentice ValidatorTier = 0
	TierVerified   ValidatorTier = 1
	TierBonded     ValidatorTier = 2
	TierGuardian   ValidatorTier = 3
)

func (t ValidatorTier) String() string {
	switch t {
	case TierApprentice:
		return "apprentice"
	case TierVerified:
		return "verified"
	case TierBonded:
		return "bonded"
	case TierGuardian:
		return "guardian"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

// TierConfig defines the requirements and benefits for each tier
type TierConfig struct {
	Tier              ValidatorTier
	MinStake          string   // bigint as string in ulgm
	MinVerifications  uint64   // minimum verifications completed
	MinAccuracy       uint64   // 0-1000000 (77% = 770000)
	MaxSlashCount     int64    // -1 means unlimited
	AllowedCategories []string // epistemic categories
	RewardMultiplier  uint64   // basis points (1000 = 1.0x)
	SelectionWeight   uint64   // basis points for VRF selection probability
}

// DefaultTierConfigs returns the standard tier configuration
func DefaultTierConfigs() []TierConfig {
	return []TierConfig{
		{
			Tier:              TierApprentice,
			MinStake:          "0",
			MinVerifications:  0,
			MinAccuracy:       0,
			MaxSlashCount:     -1,
			AllowedCategories: []string{"protocol", "analytic", "formal"},
			RewardMultiplier:  100,  // 0.1x
			SelectionWeight:   100,  // 0.1x for virtual stake calculation
		},
		{
			Tier:              TierVerified,
			MinStake:          "0",
			MinVerifications:  22,
			MinAccuracy:       770000, // 77%
			MaxSlashCount:     -1,
			AllowedCategories: []string{"protocol", "analytic", "formal", "empirical", "historical"},
			RewardMultiplier:  500,  // 0.5x
			SelectionWeight:   500,  // 0.5x
		},
		{
			Tier:              TierBonded,
			MinStake:          "1111000000", // 1,111 LGM in ulgm (6 decimals)
			MinVerifications:  0,
			MinAccuracy:       0,
			MaxSlashCount:     -1,
			AllowedCategories: []string{"protocol", "analytic", "formal", "empirical", "historical", "testimonial", "revelation"},
			RewardMultiplier:  1000, // 1.0x
			SelectionWeight:   1000, // 1.0x
		},
		{
			Tier:              TierGuardian,
			MinStake:          "11111000000", // 11,111 LGM in ulgm (6 decimals)
			MinVerifications:  1111,
			MinAccuracy:       770000, // 77%
			MaxSlashCount:     0,      // zero slashes allowed
			AllowedCategories: []string{"protocol", "analytic", "formal", "empirical", "historical", "testimonial", "revelation"},
			RewardMultiplier:  1500, // 1.5x
			SelectionWeight:   1500, // 1.5x
		},
	}
}

// LegibleValidator extends standard Cosmos validator with PoT-specific fields
type LegibleValidator struct {
	OperatorAddress       string        // bech32 operator address
	Did                   string        // did:lgm:xxx
	ConsensusPubkey       string        // Ed25519 pubkey hex
	Stake                 string        // self-bonded stake (bigint as string)
	DelegatedStake        string        // stake from delegators
	TotalStake            string        // stake + delegated
	Commission            uint64        // basis points (1000 = 10%)
	Tier                  ValidatorTier // current tier
	IsActive              bool          // actively validating
	ActiveSinceHeight     uint64        // when validator became active
	LastProposedHeight    uint64        // last block proposed
	BlocksProposed        uint64        // total blocks proposed
	VerificationsCompleted uint64       // total verifications
	VerificationsCorrect  uint64        // correct verifications
	SlashCount            uint64        // times slashed
	Reputation            uint64        // 0-1000000 (default 500000 = 0.5)
}

func (v *LegibleValidator) Reset()         { *v = LegibleValidator{} }
func (v *LegibleValidator) String() string { return proto.CompactTextString(v) }
func (v *LegibleValidator) ProtoMessage()  {}

// GetAccuracyRate returns verification accuracy as basis points
func (v *LegibleValidator) GetAccuracyRate() uint64 {
	if v.VerificationsCompleted == 0 {
		return 0
	}
	return (v.VerificationsCorrect * 1000000) / v.VerificationsCompleted
}

// Delegation represents stake delegation to a validator
type Delegation struct {
	DelegatorAddress  string // bech32 delegator address
	ValidatorAddress  string // bech32 validator operator address
	Amount            string // delegated amount (bigint as string)
	DelegatedAtHeight uint64 // block height of delegation
}

func (d *Delegation) Reset()         { *d = Delegation{} }
func (d *Delegation) String() string { return proto.CompactTextString(d) }
func (d *Delegation) ProtoMessage()  {}

// UnbondingEntry represents an unbonding delegation
type UnbondingEntry struct {
	Id                 string `json:"id"`                            // unique identifier
	DelegatorAddress   string `json:"delegator_address"`             // bech32 delegator
	ValidatorAddress   string `json:"validator_address"`             // bech32 validator
	Amount             string `json:"amount"`                        // unbonding amount
	InitiatedAtHeight  uint64 `json:"initiated_at_height"`           // when unbonding started
	CompletesAtHeight  uint64 `json:"completes_at_height"`           // when unbonding completes
	Status             string `json:"status"`                        // "pending" or "completed"
	OriginPeriod       uint64 `json:"origin_period,omitempty"`       // token expiration period preservation
}

func (u *UnbondingEntry) Reset()         { *u = UnbondingEntry{} }
func (u *UnbondingEntry) String() string { return proto.CompactTextString(u) }
func (u *UnbondingEntry) ProtoMessage()  {}

// Params defines the module parameters
type Params struct {
	MaxApprenticeValidators uint32 // max tier 0 validators (Sybil defense)
	UnbondingPeriod         uint64 // blocks (268560 ~7 days at 2.521s/block)
	VirtualStake            string // virtual stake for tier 0/1 (111 LGM)
	MaxValidators           uint32 // max total validators
	MinSelfDelegation       string // minimum self-delegation
}

func DefaultParams() Params {
	return Params{
		MaxApprenticeValidators: 111,
		UnbondingPeriod:         268560, // ~7 days
		VirtualStake:            "111000000", // 111 LGM in ulgm
		MaxValidators:           100,
		MinSelfDelegation:       "0",
	}
}

func (p *Params) Reset()         { *p = Params{} }
func (p *Params) String() string { return proto.CompactTextString(p) }
func (p *Params) ProtoMessage()  {}

// GenesisState defines the staking module's genesis state
type GenesisState struct {
	Params           Params
	Validators       []LegibleValidator
	Delegations      []Delegation
	UnbondingEntries []UnbondingEntry
	TierConfigs      []TierConfig
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		Validators:       []LegibleValidator{},
		Delegations:      []Delegation{},
		UnbondingEntries: []UnbondingEntry{},
		TierConfigs:      DefaultTierConfigs(),
	}
}

func (gs *GenesisState) Reset()         { *gs = GenesisState{} }
func (gs *GenesisState) String() string { return proto.CompactTextString(gs) }
func (gs *GenesisState) ProtoMessage()  {}

// Validate performs basic genesis state validation
func (gs *GenesisState) Validate() error {
	if gs.Params.MaxApprenticeValidators == 0 {
		return fmt.Errorf("max apprentice validators must be positive")
	}
	if gs.Params.UnbondingPeriod == 0 {
		return fmt.Errorf("unbonding period must be positive")
	}
	if len(gs.TierConfigs) != 4 {
		return fmt.Errorf("expected 4 tier configs, got %d", len(gs.TierConfigs))
	}
	return nil
}

// --- Message Types ---

// MsgRegisterValidator registers a new validator (potentially tier 0 with zero stake)
type MsgRegisterValidator struct {
	Sender          string // bech32 operator address
	ConsensusPubkey string // Ed25519 hex
	Did             string // did:lgm:xxx
	InitialStake    string // self-delegation amount (can be "0")
	Commission      uint64 // basis points
}

func (m *MsgRegisterValidator) Reset()         { *m = MsgRegisterValidator{} }
func (m *MsgRegisterValidator) String() string { return proto.CompactTextString(m) }
func (m *MsgRegisterValidator) ProtoMessage()  {}

// ValidateBasic performs stateless validation
func (m *MsgRegisterValidator) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if m.ConsensusPubkey == "" {
		return fmt.Errorf("consensus pubkey required")
	}
	if m.Did == "" {
		return fmt.Errorf("DID required")
	}
	if m.Commission > 10000 {
		return fmt.Errorf("commission cannot exceed 100%% (10000 basis points)")
	}
	return nil
}

// GetSigners returns the expected signers
func (m *MsgRegisterValidator) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{addr}
}

// MsgRegisterValidatorResponse is the response
type MsgRegisterValidatorResponse struct {
	ValidatorAddress string
	InitialTier      ValidatorTier
}

func (m *MsgRegisterValidatorResponse) Reset()         { *m = MsgRegisterValidatorResponse{} }
func (m *MsgRegisterValidatorResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgRegisterValidatorResponse) ProtoMessage()  {}

// MsgUpdateValidatorStake updates self-delegation
type MsgUpdateValidatorStake struct {
	Validator  string // bech32 validator operator address
	Amount     string // amount to stake/unstake
	IsIncrease bool   // true = stake, false = unstake
}

func (m *MsgUpdateValidatorStake) Reset()         { *m = MsgUpdateValidatorStake{} }
func (m *MsgUpdateValidatorStake) String() string { return proto.CompactTextString(m) }
func (m *MsgUpdateValidatorStake) ProtoMessage()  {}

func (m *MsgUpdateValidatorStake) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Validator); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if m.Amount == "" || m.Amount == "0" {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

func (m *MsgUpdateValidatorStake) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Validator)
	return []sdk.AccAddress{addr}
}

// MsgUpdateValidatorStakeResponse is the response
type MsgUpdateValidatorStakeResponse struct {
	NewStake string
	NewTier  ValidatorTier
}

func (m *MsgUpdateValidatorStakeResponse) Reset()         { *m = MsgUpdateValidatorStakeResponse{} }
func (m *MsgUpdateValidatorStakeResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgUpdateValidatorStakeResponse) ProtoMessage()  {}

// --- Delegation Message Types ---

// MsgDelegate delegates tokens to a validator.
type MsgDelegate struct {
	Delegator string // bech32 delegator address
	Validator string // bech32 validator operator address
	Amount    string // amount to delegate (bigint as string in ulgm)
}

func (m *MsgDelegate) Reset()         { *m = MsgDelegate{} }
func (m *MsgDelegate) String() string { return proto.CompactTextString(m) }
func (m *MsgDelegate) ProtoMessage()  {}

func (m *MsgDelegate) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Delegator); err != nil {
		return fmt.Errorf("invalid delegator address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.Validator); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if m.Amount == "" || m.Amount == "0" {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

func (m *MsgDelegate) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Delegator)
	return []sdk.AccAddress{addr}
}

// MsgDelegateResponse is the response for MsgDelegate.
type MsgDelegateResponse struct {
	NewDelegation string // total delegation to this validator
}

func (m *MsgDelegateResponse) Reset()         { *m = MsgDelegateResponse{} }
func (m *MsgDelegateResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgDelegateResponse) ProtoMessage()  {}

// MsgUndelegate undelegates tokens from a validator (initiates unbonding).
type MsgUndelegate struct {
	Delegator string // bech32 delegator address
	Validator string // bech32 validator operator address
	Amount    string // amount to undelegate (bigint as string in ulgm)
}

func (m *MsgUndelegate) Reset()         { *m = MsgUndelegate{} }
func (m *MsgUndelegate) String() string { return proto.CompactTextString(m) }
func (m *MsgUndelegate) ProtoMessage()  {}

func (m *MsgUndelegate) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Delegator); err != nil {
		return fmt.Errorf("invalid delegator address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.Validator); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if m.Amount == "" || m.Amount == "0" {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

func (m *MsgUndelegate) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Delegator)
	return []sdk.AccAddress{addr}
}

// MsgUndelegateResponse is the response for MsgUndelegate.
type MsgUndelegateResponse struct {
	UnbondingId       string // unbonding entry ID
	CompletesAtHeight uint64 // when unbonding completes
}

func (m *MsgUndelegateResponse) Reset()         { *m = MsgUndelegateResponse{} }
func (m *MsgUndelegateResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgUndelegateResponse) ProtoMessage()  {}

// MsgRedelegate moves delegation from one validator to another.
type MsgRedelegate struct {
	Delegator    string // bech32 delegator address
	SrcValidator string // bech32 source validator operator address
	DstValidator string // bech32 destination validator operator address
	Amount       string // amount to redelegate (bigint as string in ulgm)
}

func (m *MsgRedelegate) Reset()         { *m = MsgRedelegate{} }
func (m *MsgRedelegate) String() string { return proto.CompactTextString(m) }
func (m *MsgRedelegate) ProtoMessage()  {}

func (m *MsgRedelegate) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Delegator); err != nil {
		return fmt.Errorf("invalid delegator address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.SrcValidator); err != nil {
		return fmt.Errorf("invalid source validator address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.DstValidator); err != nil {
		return fmt.Errorf("invalid destination validator address: %w", err)
	}
	if m.SrcValidator == m.DstValidator {
		return fmt.Errorf("source and destination validators cannot be the same")
	}
	if m.Amount == "" || m.Amount == "0" {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

func (m *MsgRedelegate) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Delegator)
	return []sdk.AccAddress{addr}
}

// MsgRedelegateResponse is the response for MsgRedelegate.
type MsgRedelegateResponse struct{}

func (m *MsgRedelegateResponse) Reset()         { *m = MsgRedelegateResponse{} }
func (m *MsgRedelegateResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgRedelegateResponse) ProtoMessage()  {}

// MsgServer is the msg server interface
type MsgServer interface {
	RegisterValidator(context.Context, *MsgRegisterValidator) (*MsgRegisterValidatorResponse, error)
	UpdateValidatorStake(context.Context, *MsgUpdateValidatorStake) (*MsgUpdateValidatorStakeResponse, error)
	Delegate(context.Context, *MsgDelegate) (*MsgDelegateResponse, error)
	Undelegate(context.Context, *MsgUndelegate) (*MsgUndelegateResponse, error)
	Redelegate(context.Context, *MsgRedelegate) (*MsgRedelegateResponse, error)
}
