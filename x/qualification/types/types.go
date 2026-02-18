package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Enums ---

type QualificationPathway string

const (
	PathwayStakeCommitment    QualificationPathway = "stake_commitment"
	PathwayTrackRecord        QualificationPathway = "track_record"
	PathwayCrossReference     QualificationPathway = "cross_reference"
	PathwayStratumInheritance QualificationPathway = "stratum_inheritance"
	PathwayGenesis            QualificationPathway = "genesis"
)

func (p QualificationPathway) IsValid() bool {
	switch p {
	case PathwayStakeCommitment, PathwayTrackRecord, PathwayCrossReference, PathwayStratumInheritance, PathwayGenesis:
		return true
	}
	return false
}

type QualificationStatus string

const (
	StatusActive       QualificationStatus = "active"
	StatusProbationary QualificationStatus = "probationary"
	StatusSuspended    QualificationStatus = "suspended"
	StatusRevoked      QualificationStatus = "revoked"
	StatusExpired      QualificationStatus = "expired"
)

// --- Params Validation ---

func (p Params) Validate() error {
	if p.MinVerifications == 0 {
		return fmt.Errorf("min verifications must be positive")
	}
	if p.MinAccuracyBps == 0 || p.MinAccuracyBps > 10000 {
		return fmt.Errorf("min accuracy must be between 1 and 10000 bps")
	}
	if p.MinEndorsers == 0 {
		return fmt.Errorf("min endorsers must be positive")
	}
	if p.SuspensionAccuracyBps > 10000 {
		return fmt.Errorf("suspension accuracy must be <= 10000 bps")
	}
	if p.MaxQualificationDepth == 0 {
		return fmt.Errorf("max qualification depth must be positive")
	}
	return nil
}

// --- ValidateBasic and GetSigners for proto-generated Msg types ---

func (m *MsgRequestQualification) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.ValidatorAddr); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if m.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	pathway := QualificationPathway(m.Pathway)
	if !pathway.IsValid() {
		return fmt.Errorf("invalid pathway: %s", m.Pathway)
	}
	if pathway == PathwayGenesis {
		return fmt.Errorf("genesis pathway cannot be requested via message")
	}
	return nil
}

func (m *MsgRequestQualification) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.ValidatorAddr)
	return []sdk.AccAddress{addr}
}

func (m *MsgEndorseValidator) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.EndorserAddr); err != nil {
		return fmt.Errorf("invalid endorser address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.TargetValidator); err != nil {
		return fmt.Errorf("invalid target validator address: %w", err)
	}
	if m.EndorserAddr == m.TargetValidator {
		return fmt.Errorf("cannot endorse yourself")
	}
	if m.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	return nil
}

func (m *MsgEndorseValidator) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.EndorserAddr)
	return []sdk.AccAddress{addr}
}

func (m *MsgRenewQualification) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.ValidatorAddr); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if m.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	return nil
}

func (m *MsgRenewQualification) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.ValidatorAddr)
	return []sdk.AccAddress{addr}
}

func (m *MsgWithdrawQualification) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.ValidatorAddr); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if m.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	return nil
}

func (m *MsgWithdrawQualification) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.ValidatorAddr)
	return []sdk.AccAddress{addr}
}

func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if m.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return m.Params.Validate()
}

func (m *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}
