package types

import (
	"context"
)

// StakingKeeper provides validator metrics for observation.
type StakingKeeper interface {
	GetActiveValidatorCount(ctx context.Context) uint64
	GetTotalValidatorCount(ctx context.Context) uint64
	GetAverageValidatorAccuracy(ctx context.Context) uint64 // basis points (0-1M)
	GetSlashEventsInRange(ctx context.Context, startHeight, endHeight uint64) uint64
}

// KnowledgeKeeper provides fact/claim metrics for observation.
type KnowledgeKeeper interface {
	GetActiveFactCount(ctx context.Context) uint64
	GetFactsCreatedInRange(ctx context.Context, startHeight, endHeight uint64) uint64
	GetFactsExpiredInRange(ctx context.Context, startHeight, endHeight uint64) uint64
	GetChallengeSuccessRate(ctx context.Context) uint64 // basis points (0-1M)

	// Domain-level metrics (for CRI/DVI/VCF hormones)
	GetActiveDomains(ctx context.Context) []string
	GetDomainObservation(ctx context.Context, domain string, startHeight, endHeight uint64) DomainObservation
	GetCategoryRetentions(ctx context.Context, startHeight, endHeight uint64) []CategoryRetention

	// OpenClaw critical trigger: count verified protocol_observation facts since a given block.
	GetVerifiedProtocolObservationCount(ctx context.Context, sinceBlock uint64) uint64
}

// EmergencyKeeper checks whether the chain is in emergency halt.
type EmergencyKeeper interface {
	IsHalted(ctx context.Context) bool
}

// GovKeeper provides governance metrics for auto-proposal rate limiting.
// The autopoiesis module emits proposal suggestions as events — it does NOT
// submit proposals directly. This interface provides observability only.
type GovKeeper interface {
	GetActiveProposalCount(ctx context.Context) uint64
}
