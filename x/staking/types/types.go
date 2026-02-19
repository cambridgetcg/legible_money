package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
			RewardMultiplier:                100,  // 0.1x
			SelectionWeight:                 100,  // 0.1x for virtual stake calculation
			SlashWindowEpochs:               0,
			MinContestedVerifications:       0,
			ContestedVerificationMultiplier: 1,
			SlashMultiplierBps:              1500, // 1.5x slash (less stake to lose, higher penalty)
		},
		{
			Tier:                            "verified",
			MinStake:                        "1110000", // 1.11 LGM in ulgm (6 decimals)
			MinVerifications:                22,
			MinAccuracy:                     770000, // 77%
			MaxSlashCount:                   -1,
			AllowedCategories:               []string{"protocol", "computational", "formal", "empirical"},
			RewardMultiplier:                500,  // 0.5x
			SelectionWeight:                 500,  // 0.5x
			SlashWindowEpochs:               0,
			MinContestedVerifications:       0,
			ContestedVerificationMultiplier: 1,
			SlashMultiplierBps:              1200, // 1.2x slash
		},
		{
			Tier:                            "bonded",
			MinStake:                        "1111000000", // 1,111 LGM in ulgm (6 decimals)
			MinVerifications:                11,
			MinAccuracy:                     500000, // 50% — oracle/attestation access requires some epistemic track record
			MaxSlashCount:                   -1,
			AllowedCategories:               []string{"protocol", "computational", "formal", "empirical", "oracle", "attestation"},
			RewardMultiplier:                1000, // 1.0x
			SelectionWeight:                 1000, // 1.0x
			SlashWindowEpochs:               0,
			MinContestedVerifications:       0,
			ContestedVerificationMultiplier: 1,
			SlashMultiplierBps:              1000, // 1.0x slash (standard)
		},
		{
			Tier:                            "guardian",
			MinStake:                        "11111000000", // 11,111 LGM in ulgm (6 decimals)
			MinVerifications:                333,
			MinAccuracy:                     770000, // 77%
			MaxSlashCount:                   0,      // zero slashes in window
			AllowedCategories:               []string{"protocol", "computational", "formal", "empirical", "oracle", "attestation", "predictive", "social", "contested"},
			RewardMultiplier:                2000, // 2.0x — compensates asymmetric risk (zero-slash tolerance + contested engagement)
			SelectionWeight:                 1500, // 1.5x
			SlashWindowEpochs:               10,   // Only slashes within last 10 epochs count
			MinContestedVerifications:       33,   // Must engage hard claims
			ContestedVerificationMultiplier: 3,    // Contested claims count 3x
			SlashMultiplierBps:              1000, // 1.0x slash (but loses tier immediately)
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
	// Description field bounds (prevent state bloat)
	if m.Description != nil {
		if len(m.Description.Moniker) > 70 {
			return fmt.Errorf("moniker too long (max 70)")
		}
		if len(m.Description.Identity) > 70 {
			return fmt.Errorf("identity too long (max 70)")
		}
		if len(m.Description.Website) > 140 {
			return fmt.Errorf("website too long (max 140)")
		}
		if len(m.Description.Details) > 2000 {
			return fmt.Errorf("details too long (max 2000)")
		}
	}
	return nil
}

// GetSigners returns the expected signers for MsgRegisterValidator.
func (m *MsgRegisterValidator) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.ValidatorAddress)
	return []sdk.AccAddress{addr}
}

// --- Additional Message Types (now proto-generated in tx.pb.go) ---

// ValidateBasic performs stateless validation for MsgUpdateValidatorStake.
func (m *MsgUpdateValidatorStake) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Validator); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if m.Amount == "" || m.Amount == "0" {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

// GetSigners returns the expected signers for MsgUpdateValidatorStake.
func (m *MsgUpdateValidatorStake) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Validator)
	return []sdk.AccAddress{addr}
}

// ValidateBasic performs stateless validation for MsgDelegate.
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

// GetSigners returns the expected signers for MsgDelegate.
func (m *MsgDelegate) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Delegator)
	return []sdk.AccAddress{addr}
}

// ValidateBasic performs stateless validation for MsgUndelegate.
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

// GetSigners returns the expected signers for MsgUndelegate.
func (m *MsgUndelegate) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Delegator)
	return []sdk.AccAddress{addr}
}

// ValidateBasic performs stateless validation for MsgRedelegate.
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

// GetSigners returns the expected signers for MsgRedelegate.
func (m *MsgRedelegate) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Delegator)
	return []sdk.AccAddress{addr}
}

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

	// Validate tier configs (if present — empty is allowed for backward compat)
	if len(p.TierConfigs) > 0 {
		if len(p.TierConfigs) != 4 {
			return fmt.Errorf("tier_configs must have exactly 4 entries, got %d", len(p.TierConfigs))
		}
		expectedTiers := []string{"apprentice", "verified", "bonded", "guardian"}
		for i, tc := range p.TierConfigs {
			if tc.Tier != expectedTiers[i] {
				return fmt.Errorf("tier_configs[%d]: expected tier %q, got %q", i, expectedTiers[i], tc.Tier)
			}
			tcMinStake := new(big.Int)
			if _, ok := tcMinStake.SetString(tc.MinStake, 10); !ok || tcMinStake.Sign() <= 0 {
				return fmt.Errorf("tier_configs[%d]: min_stake must be positive, got %q", i, tc.MinStake)
			}
			if tc.MinAccuracy > 1000000 {
				return fmt.Errorf("tier_configs[%d]: min_accuracy must be <= 1000000, got %d", i, tc.MinAccuracy)
			}
			if tc.RewardMultiplier == 0 {
				return fmt.Errorf("tier_configs[%d]: reward_multiplier must be positive", i)
			}
			if tc.SelectionWeight == 0 {
				return fmt.Errorf("tier_configs[%d]: selection_weight must be positive", i)
			}
			if tc.SlashMultiplierBps == 0 || tc.SlashMultiplierBps > 10000 {
				return fmt.Errorf("tier_configs[%d]: slash_multiplier_bps must be in [1, 10000], got %d", i, tc.SlashMultiplierBps)
			}
		}
	}

	// Validate reputation deltas (zero is allowed for backward compat — treated as default)
	if p.ReputationCorrectDelta > 1000000 {
		return fmt.Errorf("reputation_correct_delta must be <= 1000000")
	}
	if p.ReputationIncorrectDelta > 1000000 {
		return fmt.Errorf("reputation_incorrect_delta must be <= 1000000")
	}
	if p.ReputationSlashDelta > 1000000 {
		return fmt.Errorf("reputation_slash_delta must be <= 1000000")
	}

	return nil
}

// Note: MsgUpdateParams and MsgUpdateParamsResponse are now proto-generated in tx.pb.go.

// GetSigners returns the expected signers for MsgUpdateParams.
func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic performs stateless validation for MsgUpdateParams.
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
