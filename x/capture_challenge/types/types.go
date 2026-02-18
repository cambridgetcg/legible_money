package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Constants ---

const (
	// Challenge stake and rewards (in ulgm)
	MinChallengeStake           = "77000000"  // 77 LGM
	CaptureExposureBonus        = "777000000" // 777 LGM bonus
	ChallengeStakeMultiplierBPS = 7_700_000   // 7.7x in basis points (/ 1,000,000)
	FailedChallengePenaltyBPS   = 770_000     // 77%
	ValidatorSlashOnUpheldBPS   = 300_000     // 30%

	// Evidence
	MinEvidencePieces                  = 3
	MaxEvidencePieces                  = 50
	DefaultEvidencePeriodBlocks uint64 = 777
	DefaultReviewPeriodBlocks   uint64 = 7_777

	// Bounty pools
	BaseBountyPool             = "2222000000" // 2222 LGM
	BountyReplenishmentRateBPS = 10_000       // 1% of verification rewards

	// Domain recovery
	DefaultDomainPauseDurationBlocks uint64 = 777
)

// --- Enums ---

// ChallengeStatus represents the lifecycle of a capture challenge.
type ChallengeStatus string

const (
	ChallengeSubmitted      ChallengeStatus = "submitted"
	ChallengeEvidencePeriod ChallengeStatus = "evidence_period"
	ChallengeUnderReview    ChallengeStatus = "under_review"
	ChallengeUpheld         ChallengeStatus = "upheld"
	ChallengeRejected       ChallengeStatus = "rejected"
	ChallengePartial        ChallengeStatus = "partial"
	ChallengeExpired        ChallengeStatus = "expired"
)

// EvidenceType categorizes the kind of capture evidence.
type EvidenceType string

const (
	EvidenceCoordinatedVoting    EvidenceType = "coordinated_voting"
	EvidenceTimingCorrelation    EvidenceType = "timing_correlation"
	EvidenceOffChainCoordination EvidenceType = "off_chain_coordination"
	EvidenceFalseFactAccepted    EvidenceType = "false_fact_accepted"
	EvidenceStakeManipulation    EvidenceType = "stake_manipulation"
	EvidenceQualificationFraud   EvidenceType = "qualification_fraud"
	EvidenceSystematicBias       EvidenceType = "systematic_bias"
)

// --- Params Validation ---
// Params struct is defined in genesis.pb.go (proto-generated).

// Validate performs basic param validation.
func (p *Params) Validate() error {
	if p.EvidencePeriodBlocks == 0 {
		return fmt.Errorf("evidence period blocks must be positive")
	}
	if p.ReviewPeriodBlocks == 0 {
		return fmt.Errorf("review period blocks must be positive")
	}
	return nil
}

// --- ValidateBasic & GetSigners for proto-generated Msg types ---

func (m *MsgSubmitChallenge) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	if m.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if len(m.Evidence) < MinEvidencePieces {
		return ErrInsufficientEvidence
	}
	return nil
}

func (m *MsgSubmitChallenge) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Challenger)
	return []sdk.AccAddress{addr}
}

func (m *MsgAddEvidence) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	if m.ChallengeId == "" {
		return fmt.Errorf("challenge_id is required")
	}
	return nil
}

func (m *MsgAddEvidence) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Challenger)
	return []sdk.AccAddress{addr}
}

func (m *MsgResolveChallenge) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Resolver); err != nil {
		return fmt.Errorf("invalid resolver address: %w", err)
	}
	if m.ChallengeId == "" {
		return fmt.Errorf("challenge_id is required")
	}
	switch m.Outcome {
	case "upheld", "rejected", "partial":
	default:
		return fmt.Errorf("invalid outcome: %s", m.Outcome)
	}
	return nil
}

func (m *MsgResolveChallenge) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Resolver)
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

// --- Codec Registration ---

// RegisterCodec registers the capture challenge module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitChallenge{}, "capture_challenge/SubmitChallenge", nil)
	cdc.RegisterConcrete(&MsgAddEvidence{}, "capture_challenge/AddEvidence", nil)
	cdc.RegisterConcrete(&MsgResolveChallenge{}, "capture_challenge/ResolveChallenge", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "capture_challenge/UpdateParams", nil)
}

// RegisterInterfaces registers the module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitChallenge{},
		&MsgAddEvidence{},
		&MsgResolveChallenge{},
		&MsgUpdateParams{},
	)
}
