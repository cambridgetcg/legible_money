package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// VestingCategoryStr is the string representation of epistemic vesting categories.
// Used by keeper logic to map epistemic categories to vesting schedules.
// The proto-generated VestingCategory is an int32 enum for on-wire representation.
type VestingCategoryStr string

const (
	CategoryAxiomatic     VestingCategoryStr = "axiomatic"     // Mathematical axioms
	CategoryFormalProof   VestingCategoryStr = "formal_proof"  // Verified proofs
	CategoryOnChain       VestingCategoryStr = "on_chain"      // Blockchain-verifiable
	CategoryCryptographic VestingCategoryStr = "cryptographic" // Cryptographic proofs
	CategoryComputational VestingCategoryStr = "computational" // Reproducible computation
	CategoryPeerReviewed  VestingCategoryStr = "peer_reviewed" // Published research
	CategoryReplicated    VestingCategoryStr = "replicated"    // IRL-replicated
	CategoryOracleFeed    VestingCategoryStr = "oracle_feed"   // Oracle price feeds
	CategoryAttestation   VestingCategoryStr = "attestation"   // Human attestations
	CategoryContested     VestingCategoryStr = "contested"     // Claims with active disputes
)

// VestingStatus tracks the lifecycle of a vesting schedule
type VestingStatus string

const (
	VestingStatusActive    VestingStatus = "vesting"   // Active, releasing rewards
	VestingStatusPaused    VestingStatus = "paused"    // Paused due to challenge
	VestingStatusCompleted VestingStatus = "completed" // Reached max release
	VestingStatusFalsified VestingStatus = "falsified" // Claim proven false
	VestingStatusAbandoned VestingStatus = "abandoned" // Recipient abandoned
)

// RewardSource identifies where a reward originated
type RewardSource string

const (
	SourceBlockProduction   RewardSource = "block_production"
	SourceVerification      RewardSource = "verification"
	SourceAPICall           RewardSource = "api_call"
	SourceCitation          RewardSource = "citation"
	SourceChallengeWin      RewardSource = "challenge_win"
	SourceChallengeDefense  RewardSource = "challenge_defense"
	SourceBountyFulfillment RewardSource = "bounty_fulfillment"
	SourceFalsification     RewardSource = "falsification"
)

// --- ValidateBasic methods for proto-generated message types ---

func (m *MsgCreateVesting) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority required")
	}
	if m.Beneficiary == "" {
		return fmt.Errorf("beneficiary required")
	}
	if _, err := sdk.AccAddressFromBech32(m.Beneficiary); err != nil {
		return fmt.Errorf("invalid beneficiary address: %w", err)
	}
	if m.Amount == "" || m.Amount == "0" {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

func (m *MsgCreateVesting) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (m *MsgClaimVesting) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Claimer); err != nil {
		return fmt.Errorf("invalid claimer address: %w", err)
	}
	return nil
}

func (m *MsgClaimVesting) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Claimer)
	return []sdk.AccAddress{addr}
}

func (m *MsgFalsifyVesting) ValidateBasic() error {
	if m.Challenger == "" {
		return fmt.Errorf("challenger address required")
	}
	if m.VestingId == "" {
		return fmt.Errorf("vesting ID required")
	}
	return nil
}

func (m *MsgFalsifyVesting) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Challenger)
	return []sdk.AccAddress{addr}
}

func (m *MsgPauseVesting) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority required")
	}
	if m.VestingId == "" {
		return fmt.Errorf("vesting ID required")
	}
	return nil
}

func (m *MsgPauseVesting) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (m *MsgResumeVesting) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority required")
	}
	if m.VestingId == "" {
		return fmt.Errorf("vesting ID required")
	}
	return nil
}

func (m *MsgResumeVesting) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (m *MsgAccelerateVesting) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority required")
	}
	if m.VestingId == "" {
		return fmt.Errorf("vesting ID required")
	}
	return nil
}

func (m *MsgAccelerateVesting) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (m *MsgCompleteVesting) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority required")
	}
	if m.VestingId == "" {
		return fmt.Errorf("vesting ID required")
	}
	return nil
}

func (m *MsgCompleteVesting) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (m *MsgUpdateParams) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority required")
	}
	return nil
}

func (m *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}
