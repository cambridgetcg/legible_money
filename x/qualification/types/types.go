package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
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

// --- Core Types ---

type QualificationMetrics struct {
	TotalVerifications    uint32 `json:"total_verifications"`
	CorrectVerifications  uint32 `json:"correct_verifications"`
	AccuracyBps          uint32 `json:"accuracy_bps"`             // basis points (10000 = 100%)
	LastVerificationBlock uint64 `json:"last_verification_block"`
	ConsecutiveCorrect    uint32 `json:"consecutive_correct"`
	ConsecutiveIncorrect  uint32 `json:"consecutive_incorrect"`
	ChallengesReceived    uint32 `json:"challenges_received"`
	ChallengesLost        uint32 `json:"challenges_lost"`
	EffectiveWeightBps    uint32 `json:"effective_weight_bps"`     // basis points (10000 = 100%)
}

type DomainQualification struct {
	ValidatorAddr          string               `json:"validator_addr"`
	Domain                 string               `json:"domain"`
	Stratum                uint32               `json:"stratum"`
	Pathway                QualificationPathway `json:"pathway"`
	Status                 QualificationStatus  `json:"status"`
	EarnedAtBlock          uint64               `json:"earned_at_block"`
	ExpiresAtBlock         uint64               `json:"expires_at_block"`           // 0 = no expiry
	Metrics                QualificationMetrics `json:"metrics"`
	LockedStake            string               `json:"locked_stake,omitempty"`
	StakeUnlocksAt         uint64               `json:"stake_unlocks_at,omitempty"`
	Endorsers              []string             `json:"endorsers,omitempty"`
	InheritedFrom          string               `json:"inherited_from,omitempty"`
	InheritanceDiscountBps uint32               `json:"inheritance_discount_bps"`   // 10000 = 1.0 (no discount)
}

func (q *DomainQualification) Reset()         { *q = DomainQualification{} }
func (q *DomainQualification) String() string { return proto.CompactTextString(q) }
func (q *DomainQualification) ProtoMessage()  {}

type QualificationEndorsement struct {
	Id              string `json:"id"`
	EndorserAddr    string `json:"endorser_addr"`
	TargetValidator string `json:"target_validator"`
	Domain          string `json:"domain"`
	Stake           string `json:"stake"`
	EndorsedAtBlock uint64 `json:"endorsed_at_block"`
	ExpiresAtBlock  uint64 `json:"expires_at_block"`
}

func (e *QualificationEndorsement) Reset()         { *e = QualificationEndorsement{} }
func (e *QualificationEndorsement) String() string { return proto.CompactTextString(e) }
func (e *QualificationEndorsement) ProtoMessage()  {}

// --- Params ---

type Params struct {
	StakeLeaf              string `json:"stake_leaf"`                // 22 LGM in ulgm
	StakeMid               string `json:"stake_mid"`                 // 222 LGM in ulgm
	StakeTop               string `json:"stake_top"`                 // 2222 LGM in ulgm
	StakeLockPeriod        uint64 `json:"stake_lock_period"`         // blocks
	MinVerifications       uint32 `json:"min_verifications"`         // for track_record
	MinAccuracyBps         uint32 `json:"min_accuracy_bps"`          // 7700 = 77%
	MinEndorsers           uint32 `json:"min_endorsers"`             // for cross_reference
	EndorserStake          string `json:"endorser_stake"`            // per endorser
	EndorsementDecayBlocks uint64 `json:"endorsement_decay_blocks"`
	StratumDiscountBps     uint32 `json:"stratum_discount_bps"`      // 7000 = 70%
	InactivityExpiration   uint64 `json:"inactivity_expiration"`     // blocks
	ProbationaryPeriod     uint64 `json:"probationary_period"`       // blocks
	ProbationaryWeightBps  uint32 `json:"probationary_weight_bps"`   // 5000 = 50%
	SuspensionAccuracyBps  uint32 `json:"suspension_accuracy_bps"`   // 5000 = 50%
	SuspensionConsecutive  uint32 `json:"suspension_consecutive"`
	RevocationChallenges   uint32 `json:"revocation_challenges"`
	MaxQualificationDepth  uint32 `json:"max_qualification_depth"`
}

func (p *Params) Reset()         { *p = Params{} }
func (p *Params) String() string { return proto.CompactTextString(p) }
func (p *Params) ProtoMessage()  {}

func DefaultParams() Params {
	return Params{
		StakeLeaf:              "22000000000000000000",    // 22 LGM
		StakeMid:               "222000000000000000000",   // 222 LGM
		StakeTop:               "2222000000000000000000",  // 2222 LGM
		StakeLockPeriod:        7777,
		MinVerifications:       19,
		MinAccuracyBps:         7700,
		MinEndorsers:           3,
		EndorserStake:          "22000000000000000000",    // 22 LGM
		EndorsementDecayBlocks: 77777,
		StratumDiscountBps:     7000,                      // 70%
		InactivityExpiration:   777777,
		ProbationaryPeriod:     777,
		ProbationaryWeightBps:  5000,                      // 50%
		SuspensionAccuracyBps:  5000,                      // 50%
		SuspensionConsecutive:  5,
		RevocationChallenges:   3,
		MaxQualificationDepth:  7,
	}
}

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

// --- Genesis ---

type GenesisState struct {
	Params         Params                `json:"params"`
	Qualifications []DomainQualification `json:"qualifications"`
	Endorsements   []QualificationEndorsement `json:"endorsements"`
}

func (gs *GenesisState) Reset()         { *gs = GenesisState{} }
func (gs *GenesisState) String() string { return proto.CompactTextString(gs) }
func (gs *GenesisState) ProtoMessage()  {}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:         DefaultParams(),
		Qualifications: []DomainQualification{},
		Endorsements:   []QualificationEndorsement{},
	}
}

func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
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
func (m *MsgRequestQualification) String() string { return proto.CompactTextString(m) }
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
func (m *MsgRequestQualificationResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgRequestQualificationResponse) ProtoMessage()  {}

// MsgEndorseValidator endorses another validator for a domain (cross_reference pathway).
type MsgEndorseValidator struct {
	EndorserAddr    string `json:"endorser_addr"`
	TargetValidator string `json:"target_validator"`
	Domain          string `json:"domain"`
	Stake           string `json:"stake"`
}

func (m *MsgEndorseValidator) Reset()         { *m = MsgEndorseValidator{} }
func (m *MsgEndorseValidator) String() string { return proto.CompactTextString(m) }
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
func (m *MsgEndorseValidatorResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgEndorseValidatorResponse) ProtoMessage()  {}

// MsgRenewQualification renews an expiring qualification.
type MsgRenewQualification struct {
	ValidatorAddr string `json:"validator_addr"`
	Domain        string `json:"domain"`
}

func (m *MsgRenewQualification) Reset()         { *m = MsgRenewQualification{} }
func (m *MsgRenewQualification) String() string { return proto.CompactTextString(m) }
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
func (m *MsgRenewQualificationResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgRenewQualificationResponse) ProtoMessage()  {}

// MsgWithdrawQualification voluntarily withdraws qualification and unlocks stake.
type MsgWithdrawQualification struct {
	ValidatorAddr string `json:"validator_addr"`
	Domain        string `json:"domain"`
}

func (m *MsgWithdrawQualification) Reset()         { *m = MsgWithdrawQualification{} }
func (m *MsgWithdrawQualification) String() string { return proto.CompactTextString(m) }
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
func (m *MsgWithdrawQualificationResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgWithdrawQualificationResponse) ProtoMessage()  {}

// MsgUpdateParams updates module parameters (authority only).
type MsgUpdateParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

func (m *MsgUpdateParams) Reset()         { *m = MsgUpdateParams{} }
func (m *MsgUpdateParams) String() string { return proto.CompactTextString(m) }
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
func (m *MsgUpdateParamsResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgUpdateParamsResponse) ProtoMessage()  {}

// --- Service Interfaces ---

type MsgServer interface {
	RequestQualification(context.Context, *MsgRequestQualification) (*MsgRequestQualificationResponse, error)
	EndorseValidator(context.Context, *MsgEndorseValidator) (*MsgEndorseValidatorResponse, error)
	RenewQualification(context.Context, *MsgRenewQualification) (*MsgRenewQualificationResponse, error)
	WithdrawQualification(context.Context, *MsgWithdrawQualification) (*MsgWithdrawQualificationResponse, error)
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}
