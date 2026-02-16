package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// Tier aliases — map legacy names to proto-generated ValidatorTier enum values.
const (
	TierApprentice = ValidatorTier_VALIDATOR_TIER_APPRENTICE // 1
	TierVerified   = ValidatorTier_VALIDATOR_TIER_VERIFIED   // 2
	TierBonded     = ValidatorTier_VALIDATOR_TIER_BONDED     // 3
	TierGuardian   = ValidatorTier_VALIDATOR_TIER_GUARDIAN   // 4
)

// ValidatorTierFromString converts a tier string back to ValidatorTier.
func ValidatorTierFromString(s string) ValidatorTier {
	switch s {
	case "apprentice":
		return TierApprentice
	case "verified":
		return TierVerified
	case "bonded":
		return TierBonded
	case "guardian":
		return TierGuardian
	default:
		return TierApprentice
	}
}

// ValidatorTierString returns a human-readable tier name (lowercase).
// Use this instead of the proto-generated String() when a short tier name is needed.
func ValidatorTierString(t ValidatorTier) string {
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

// Note: TierConfig struct is defined in state.pb.go (proto-generated).

// DefaultTierConfigs returns the standard tier configuration
func DefaultTierConfigs() []*TierConfig {
	return []*TierConfig{
		{
			Tier:                            "apprentice",
			MinStake:                        "111000", // 0.111 LGM in ulgm (6 decimals)
			MinVerifications:                0,
			MinAccuracy:                     0,
			MaxSlashCount:                   -1,
			AllowedCategories:               []string{"protocol", "computational", "formal"},
			RewardMultiplier:                100, // 0.1x
			SelectionWeight:                 100, // 0.1x for virtual stake calculation
			SlashWindowEpochs:               0,
			MinContestedVerifications:       0,
			ContestedVerificationMultiplier: 1,
		},
		{
			Tier:                            "verified",
			MinStake:                        "1110000", // 1.11 LGM in ulgm (6 decimals)
			MinVerifications:                22,
			MinAccuracy:                     770000, // 77%
			MaxSlashCount:                   -1,
			AllowedCategories:               []string{"protocol", "computational", "formal", "empirical"},
			RewardMultiplier:                500, // 0.5x
			SelectionWeight:                 500, // 0.5x
			SlashWindowEpochs:               0,
			MinContestedVerifications:       0,
			ContestedVerificationMultiplier: 1,
		},
		{
			Tier:                            "bonded",
			MinStake:                        "1111000000", // 1,111 LGM in ulgm (6 decimals)
			MinVerifications:                0,
			MinAccuracy:                     0,
			MaxSlashCount:                   -1,
			AllowedCategories:               []string{"protocol", "computational", "formal", "empirical", "oracle", "attestation"},
			RewardMultiplier:                1000, // 1.0x
			SelectionWeight:                 1000, // 1.0x
			SlashWindowEpochs:               0,
			MinContestedVerifications:       0,
			ContestedVerificationMultiplier: 1,
		},
		{
			Tier:                            "guardian",
			MinStake:                        "11111000000", // 11,111 LGM in ulgm (6 decimals)
			MinVerifications:                333,
			MinAccuracy:                     770000, // 77%
			MaxSlashCount:                   0,      // zero slashes in window
			AllowedCategories:               []string{"protocol", "computational", "formal", "empirical", "oracle", "attestation", "predictive", "social", "contested"},
			RewardMultiplier:                1500, // 1.5x
			SelectionWeight:                 1500, // 1.5x
			SlashWindowEpochs:               10,   // Only slashes within last 10 epochs count
			MinContestedVerifications:       33,   // Must engage hard claims
			ContestedVerificationMultiplier: 3,    // Contested claims count 3x
		},
	}
}

// Note: LegibleValidator struct is defined in state.pb.go (proto-generated).

// GetAccuracyRate returns verification accuracy as basis points
func (v *LegibleValidator) GetAccuracyRate() uint64 {
	if v.VerificationsCompleted == 0 {
		return 0
	}
	return (v.VerificationsCorrect * 1000000) / v.VerificationsCompleted
}

// Note: Delegation, UnbondingEntry, Params, and GenesisState structs
// are defined in state.pb.go and genesis.pb.go (proto-generated).

// Note: MsgRegisterValidator and MsgRegisterValidatorResponse are defined in tx.pb.go (proto-generated).

// ValidateBasic performs stateless validation for MsgRegisterValidator.
func (m *MsgRegisterValidator) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.ValidatorAddress); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if len(m.PubKey) == 0 {
		return fmt.Errorf("consensus pubkey required")
	}
	if m.CommissionRate > 10000 {
		return fmt.Errorf("commission cannot exceed 100%% (10000 basis points)")
	}
	return nil
}

// GetSigners returns the expected signers for MsgRegisterValidator.
func (m *MsgRegisterValidator) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.ValidatorAddress)
	return []sdk.AccAddress{addr}
}

// --- Additional Message Types (not in proto) ---

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

// --- Params Validation ---

// Validate performs parameter validation.
func (p *Params) Validate() error {
	if p.UnbondingPeriod == 0 {
		return fmt.Errorf("unbonding_period must be positive")
	}
	if p.MaxValidators == 0 {
		return fmt.Errorf("max_validators must be positive")
	}
	stake := new(big.Int)
	if _, ok := stake.SetString(p.VirtualStake, 10); !ok || stake.Sign() <= 0 {
		return fmt.Errorf("invalid virtual_stake: %s", p.VirtualStake)
	}
	minDel := new(big.Int)
	if _, ok := minDel.SetString(p.MinSelfDelegation, 10); !ok || minDel.Sign() <= 0 {
		return fmt.Errorf("invalid min_self_delegation: %s", p.MinSelfDelegation)
	}
	if p.SlashEscalationBps > 1000000 {
		return fmt.Errorf("slash_escalation_bps must be <= 1000000")
	}
	return nil
}

// MsgUpdateParams updates module parameters (governance-gated).
type MsgUpdateParams struct {
	Authority string `json:"authority"` // gov module address
	Params    Params `json:"params"`
}

func (msg *MsgUpdateParams) Reset()        {}
func (msg *MsgUpdateParams) String() string { return "MsgUpdateParams" }
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

func (msg *MsgUpdateParamsResponse) Reset()        {}
func (msg *MsgUpdateParamsResponse) String() string { return "" }
func (msg *MsgUpdateParamsResponse) ProtoMessage()  {}

// FullMsgServer extends the proto-generated MsgServer with additional RPCs
// that are not yet defined in the protobuf service.
type FullMsgServer interface {
	MsgServer
	UpdateValidatorStake(sdk.Context, *MsgUpdateValidatorStake) (*MsgUpdateValidatorStakeResponse, error)
	Delegate(sdk.Context, *MsgDelegate) (*MsgDelegateResponse, error)
	Undelegate(sdk.Context, *MsgUndelegate) (*MsgUndelegateResponse, error)
	Redelegate(sdk.Context, *MsgRedelegate) (*MsgRedelegateResponse, error)
	UpdateParams(sdk.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}
