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
}

// EmergencyKeeper checks whether the chain is in emergency halt.
type EmergencyKeeper interface {
	IsHalted(ctx context.Context) bool
}
