package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Enums ---

// QualificationPathway represents how a validator became qualified for a domain.
type QualificationPathway string

const (
	PathwayStakeCommitment    QualificationPathway = "stake_commitment"
	PathwayTrackRecord        QualificationPathway = "track_record"
	PathwayCrossReference     QualificationPathway = "cross_reference"
	PathwayStratumInheritance QualificationPathway = "stratum_inheritance"
	PathwayGenesis            QualificationPathway = "genesis"
)

// QualificationStatus represents the current status of a domain qualification.
type QualificationStatus string

const (
	StatusActive       QualificationStatus = "active"
	StatusProbationary QualificationStatus = "probationary"
	StatusSuspended    QualificationStatus = "suspended"
	StatusRevoked      QualificationStatus = "revoked"
	StatusExpired      QualificationStatus = "expired"
)

// ReputationSource indicates which reputation layer was used.
type ReputationSource string

const (
	SourceDomain  ReputationSource = "domain"
	SourceStratum ReputationSource = "stratum"
	SourceGlobal  ReputationSource = "global"
)

// --- Configuration Constants ---
// All floating-point values stored as uint64 basis points (1,000,000 = 100%).

const (
	// Stake requirements (in ulgm = micro-LGM)
	StakeLeafDomain = "22000000"   // 22 LGM for leaf domains (depth > 2)
	StakeMidDomain  = "222000000"  // 222 LGM for mid-level (depth 2)
	StakeTopDomain  = "2222000000" // 2222 LGM for top-level (depth 1)

	// Lock and timing (in blocks)
	StakeLockPeriod            uint64 = 7_777
	MinVerificationsForQual    uint64 = 19
	MinAccuracyForQual         uint64 = 770_000 // 77% in basis points
	MinEndorsersForQual        uint64 = 3
	EndorsementDecayBlocks     uint64 = 77_777
	InactivityExpirationBlocks uint64 = 777_777
	ProbationaryPeriodBlocks   uint64 = 777
	MaxQualificationDepth      uint64 = 7

	// Weights (basis points, 1,000,000 = 100%)
	ProbationaryWeight          uint64 = 500_000 // 50%
	StratumInheritanceDiscount  uint64 = 700_000 // 70%
	AccuracyThresholdSuspension uint64 = 500_000 // 50%
	ConsecutiveIncorrectSuspend uint64 = 5

	// Reputation config
	ReputationHalfLifeBlocks     uint64 = 77_777
	BaseScore                    uint64 = 500
	MaxScore                     uint64 = 1000
	MinDecayedScore              uint64 = 300
	GlobalReputationDiscount     uint64 = 500_000 // 50%
	StratumReputationDiscount    uint64 = 700_000 // 70%
	NewDomainDiscount            uint64 = 300_000 // 30%
	MinVerificationsForDomainRep uint64 = 19

	// Score component weights (must sum to 1,000,000)
	AccuracyWeight         uint64 = 500_000 // 50%
	ConsistencyWeight      uint64 = 250_000 // 25%
	ChallengeDefenseWeight uint64 = 150_000 // 15%
	CalibrationWeight      uint64 = 100_000 // 10%

	// Capture detection thresholds
	HerfindahlTriggerThreshold       uint64 = 400_000 // 0.4
	HerfindahlPauseThreshold         uint64 = 600_000 // 0.6
	MinVerificationsForAnalysis      uint64 = 19
	DetectionIntervalBlocks          uint64 = 777
	DecayIntervalBlocks              uint64 = 7_777
	MaxExpirationsPerBlock           uint64 = 100
	DefaultDomainPauseDurationBlocks uint64 = 777 // ~32 min auto-pause on critical HHI

	// Cross-stratum
	CrossStratumAgreementThreshold uint64 = 700_000 // 70%
	LowerStratumVetoWeight         uint64 = 300_000 // 30%
	PartialCoverageConfPenalty     uint64 = 200_000 // 20%
	MaxStratumDistance             uint64 = 2
)

// --- Params Validation ---

// Validate performs basic param validation.
func (p *Params) Validate() error {
	if p.StakeLockPeriod == 0 {
		return fmt.Errorf("stake lock period must be positive")
	}
	if p.MinAccuracyForQual > 1_000_000 {
		return fmt.Errorf("min accuracy must be <= 1,000,000")
	}
	if p.HerfindahlTriggerThreshold > 1_000_000 {
		return fmt.Errorf("herfindahl trigger must be <= 1,000,000")
	}
	return nil
}

// --- Validation & Signers for proto-generated Msg types ---

func (m *MsgRequestQualification) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.ValidatorAddr); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if m.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	switch QualificationPathway(m.Pathway) {
	case PathwayStakeCommitment, PathwayTrackRecord, PathwayCrossReference, PathwayStratumInheritance:
	default:
		return fmt.Errorf("invalid pathway: %s", m.Pathway)
	}
	return nil
}

func (m *MsgRequestQualification) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.ValidatorAddr)
	return []sdk.AccAddress{addr}
}

func (m *MsgEndorseQualification) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Endorser); err != nil {
		return fmt.Errorf("invalid endorser address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.TargetValidator); err != nil {
		return fmt.Errorf("invalid target validator address: %w", err)
	}
	if m.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if m.Endorser == m.TargetValidator {
		return ErrSelfEndorsement
	}
	return nil
}

func (m *MsgEndorseQualification) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Endorser)
	return []sdk.AccAddress{addr}
}

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
