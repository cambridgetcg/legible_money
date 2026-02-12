package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Enums ---

type QualificationPathway string

const (
	PathwayStakeCommitment   QualificationPathway = "stake_commitment"
	PathwayTrackRecord       QualificationPathway = "track_record"
	PathwayCrossReference    QualificationPathway = "cross_reference"
	PathwayStratumInheritance QualificationPathway = "stratum_inheritance"
	PathwayGenesis           QualificationPathway = "genesis"
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

// --- Message Types ---

// MsgRequestQualification requests qualification in a domain via a specific pathway.
type MsgRequestQualification struct {
	ValidatorAddr string               `json:"validator_addr"`
	Domain        string               `json:"domain"`
	Pathway       QualificationPathway `json:"pathway"`
	Stake         string               `json:"stake,omitempty"`         // for stake_commitment
	InheritFrom   string               `json:"inherit_from,omitempty"`  // for stratum_inheritance
}

func (m *MsgRequestQualification) Reset()         { *m = MsgRequestQualification{} }
func (m *MsgRequestQualification) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MsgRequestQualification) ProtoMessage()  {}

func (m *MsgRequestQualification) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.ValidatorAddr); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if m.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if !m.Pathway.IsValid() {
		return fmt.Errorf("invalid pathway: %s", m.Pathway)
	}
	if m.Pathway == PathwayGenesis {
		return fmt.Errorf("genesis pathway cannot be requested via message")
	}
	return nil
}

func (m *MsgRequestQualification) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.ValidatorAddr)
	return []sdk.AccAddress{addr}
}

type MsgRequestQualificationResponse struct {
	Status QualificationStatus `json:"status"`
}

func (m *MsgRequestQualificationResponse) Reset()         { *m = MsgRequestQualificationResponse{} }
func (m *MsgRequestQualificationResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MsgRequestQualificationResponse) ProtoMessage()  {}

// MsgEndorseValidator endorses another validator for a domain (cross_reference pathway).
type MsgEndorseValidator struct {
	EndorserAddr    string `json:"endorser_addr"`
	TargetValidator string `json:"target_validator"`
	Domain          string `json:"domain"`
	Stake           string `json:"stake"`
}

func (m *MsgEndorseValidator) Reset()         { *m = MsgEndorseValidator{} }
func (m *MsgEndorseValidator) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MsgEndorseValidator) ProtoMessage()  {}

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

type MsgEndorseValidatorResponse struct {
	EndorsementId string `json:"endorsement_id"`
}

func (m *MsgEndorseValidatorResponse) Reset()         { *m = MsgEndorseValidatorResponse{} }
func (m *MsgEndorseValidatorResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MsgEndorseValidatorResponse) ProtoMessage()  {}

// MsgRenewQualification renews an expiring qualification.
type MsgRenewQualification struct {
	ValidatorAddr string `json:"validator_addr"`
	Domain        string `json:"domain"`
}

func (m *MsgRenewQualification) Reset()         { *m = MsgRenewQualification{} }
func (m *MsgRenewQualification) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MsgRenewQualification) ProtoMessage()  {}

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

type MsgRenewQualificationResponse struct{}

func (m *MsgRenewQualificationResponse) Reset()         { *m = MsgRenewQualificationResponse{} }
func (m *MsgRenewQualificationResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MsgRenewQualificationResponse) ProtoMessage()  {}

// MsgWithdrawQualification voluntarily withdraws qualification and unlocks stake.
type MsgWithdrawQualification struct {
	ValidatorAddr string `json:"validator_addr"`
	Domain        string `json:"domain"`
}

func (m *MsgWithdrawQualification) Reset()         { *m = MsgWithdrawQualification{} }
func (m *MsgWithdrawQualification) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MsgWithdrawQualification) ProtoMessage()  {}

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

type MsgWithdrawQualificationResponse struct{}

func (m *MsgWithdrawQualificationResponse) Reset()         { *m = MsgWithdrawQualificationResponse{} }
func (m *MsgWithdrawQualificationResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MsgWithdrawQualificationResponse) ProtoMessage()  {}

// MsgUpdateParams updates module parameters (authority only).
type MsgUpdateParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

func (m *MsgUpdateParams) Reset()         { *m = MsgUpdateParams{} }
func (m *MsgUpdateParams) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MsgUpdateParams) ProtoMessage()  {}

func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	return m.Params.Validate()
}

func (m *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

type MsgUpdateParamsResponse struct{}

func (m *MsgUpdateParamsResponse) Reset()         { *m = MsgUpdateParamsResponse{} }
func (m *MsgUpdateParamsResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MsgUpdateParamsResponse) ProtoMessage()  {}

// --- Service Interfaces ---

type MsgServer interface {
	RequestQualification(context.Context, *MsgRequestQualification) (*MsgRequestQualificationResponse, error)
	EndorseValidator(context.Context, *MsgEndorseValidator) (*MsgEndorseValidatorResponse, error)
	RenewQualification(context.Context, *MsgRenewQualification) (*MsgRenewQualificationResponse, error)
	WithdrawQualification(context.Context, *MsgWithdrawQualification) (*MsgWithdrawQualificationResponse, error)
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}
