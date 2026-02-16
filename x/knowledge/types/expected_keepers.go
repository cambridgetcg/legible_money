package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected account keeper interface.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected bank keeper interface.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
}

// OntologyKeeper defines the expected ontology keeper interface.
// The knowledge module uses this to look up stratum-based confidence ceilings for domains
// and to enforce Gödelian logic zone constraints on claims.
type OntologyKeeper interface {
	// GetConfidenceCeiling returns the max confidence and decay rate for a domain
	// based on its ontological stratum. Returns error if domain is not registered.
	GetConfidenceCeiling(ctx sdk.Context, domainName string) (maxConfidence uint64, decayRate uint64, err error)

	// ValidateClaimLogicZone validates that a claim's confidence does not exceed
	// the maximum allowed by its declared logic zone. Returns nil if zone is empty
	// (backward compatible) or valid, error if confidence exceeds ceiling or zone unknown.
	ValidateClaimLogicZone(ctx sdk.Context, zone string, confidenceBps uint64) error

	// GetZoneConfidenceCeiling returns the maximum confidence in basis points
	// (0-1,000,000 scale) for a logic zone. Returns 1000000 if zone is empty or unknown.
	GetZoneConfidenceCeiling(ctx sdk.Context, zone string) uint64

	// RequiresIncompletenessAck returns whether a zone requires Gödelian
	// incompleteness acknowledgment (true for Peano, set_theory).
	RequiresIncompletenessAck(ctx sdk.Context, zone string) bool
}

// DomainQualificationKeeper defines the expected domain qualification keeper interface.
// The knowledge module uses this to filter validators by domain competence.
type DomainQualificationKeeper interface {
	IsQualified(ctx sdk.Context, validatorAddr, domain string) bool
	GetQualificationWeight(ctx sdk.Context, validatorAddr, domain string) uint32
	GetQualifiedValidators(ctx sdk.Context, domain string) []string
	RecordVerificationOutcome(ctx sdk.Context, validatorAddr, domain string, correct bool) error
}

// CaptureDefenseKeeper defines the expected capture defense keeper interface.
// The knowledge module uses this to weight confidence by domain reputation.
type CaptureDefenseKeeper interface {
	IsQualifiedForDomain(ctx sdk.Context, validatorAddr, domain string, currentBlock uint64) bool
	GetQualificationWeight(ctx sdk.Context, validatorAddr, domain string) uint64
	// GetReputationScore returns the effective reputation score for a validator in a domain.
	// The returned score is basis points (0-1,000,000 = 0.0-1.0).
	GetReputationScore(ctx sdk.Context, validatorAddr, domain string) uint64
}

// VestingRewardsKeeper defines the expected vesting rewards keeper interface.
// The knowledge module calls this to create truth-linked vesting schedules on claim acceptance.
type VestingRewardsKeeper interface {
	CreateVestingScheduleFromKnowledge(ctx sdk.Context, claimId, factId, recipient, totalAmount, epistemicCategory string) error
	// DistributeFalsificationReward creates a vesting schedule for a falsification reward.
	// The reward vests with SourceFalsification, enabling clawback if the counter-fact is later overturned.
	DistributeFalsificationReward(ctx sdk.Context, counterFactId, targetFactId, recipient, amount string) error
	// GetEpochBlockRewardPool returns the total block rewards distributed in the given epoch (ulgm).
	// Used by the knowledge module to cap falsification rewards at FalsificationPoolShareBps.
	GetEpochBlockRewardPool(ctx sdk.Context, epoch uint64) uint64
	// PauseVestingByClaimId pauses the vesting schedule associated with a claim.
	// Used when a fact is archived (pruned) to stop further reward vesting.
	PauseVestingByClaimId(ctx sdk.Context, claimId string) error
}

// EvidenceMgmtKeeper defines the expected evidence management keeper interface.
// The knowledge module uses this to query evidence quality during verification
// confidence aggregation, so that evidence actually affects truth determination.
type EvidenceMgmtKeeper interface {
	// GetEvidenceByClaimId returns all evidence records for a given claim.
	GetEvidenceByClaimId(ctx sdk.Context, claimId string) []EvidenceRecord

	// ComputeReliability computes the stake-weighted reliability score for evidence.
	// Returns reliability in basis points (0-1,000,000).
	ComputeReliability(ctx sdk.Context, evidenceId string) (uint64, error)
}

// EvidenceRecord is a minimal view of evidence_mgmt.Evidence for cross-module use.
// Defined here to avoid importing x/evidence_mgmt/types directly (circular dep prevention).
type EvidenceRecord struct {
	Id           string `json:"id"`
	ClaimId      string `json:"claim_id"`
	EvidenceType string `json:"evidence_type"` // on_chain, oracle_feed, attestation, computation, document, experiment
	Status       string `json:"status"`        // submitted, verified, rejected, expired
	Reliability  uint64 `json:"reliability"`   // 0-1,000,000 basis points
}

// StakingKeeper defines the expected staking keeper interface.
// The knowledge module uses this to query validator info for VRF selection.
type StakingKeeper interface {
	// GetActiveValidatorInfos returns all active validators as ValidatorInfo structs.
	GetActiveValidatorInfos(ctx context.Context) []ValidatorInfo

	// GetValidatorInfo returns info for a single validator by operator address.
	GetValidatorInfo(ctx context.Context, operatorAddr string) (ValidatorInfo, bool)

	// GetEffectiveStake returns the effective selection stake for a validator,
	// accounting for virtual stake at lower tiers.
	GetEffectiveStake(ctx context.Context, operatorAddr string) uint64

	// GetTotalStake returns the total stake across all active validators.
	GetTotalStake(ctx context.Context) uint64

	// SlashValidator slashes a validator's stake.
	SlashValidator(ctx context.Context, operatorAddr string, amount string, reason string) error
}
