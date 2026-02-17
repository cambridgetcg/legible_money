package orchestrator

import (
	"context"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// Instrument is the intelligence interface for specialized agent behaviors.
// Each orchestral section has a corresponding instrument type that provides
// domain-aware reasoning before actions are dispatched to the chain.
type Instrument interface {
	// Name returns the instrument's display name.
	Name() string

	// Section returns which orchestral section this instrument belongs to.
	Section() SectionName

	// PrepareAction is called before ActionExecutor.Execute().
	// It may modify params, inject computed values, or return an error to skip.
	PrepareAction(ctx context.Context, agent *agentsdk.Agent, action ActionType, params map[string]string) (map[string]string, error)
}

// InstrumentConfig holds common configuration for instruments.
type InstrumentConfig struct {
	// Domains this instrument specializes in.
	Domains []string

	// Verbose enables detailed logging.
	Verbose bool
}

// ClaimResolver fetches claim details for evaluation.
type ClaimResolver interface {
	ResolveClaim(ctx context.Context, claimID string) (*agentsdk.ClaimInfo, error)
}

// FactScanner fetches facts in a domain for challenge detection.
type FactScanner interface {
	ScanFacts(ctx context.Context, domain string) ([]agentsdk.FactInfo, error)
}

// categoryBaseConfidence maps epistemic categories to base confidence values.
// Duplicated from x/knowledge/evaluation/category.go because off-chain packages
// cannot import on-chain modules.
var categoryBaseConfidence = map[string]uint64{
	"protocol":      800_000,
	"analytic":      700_000,
	"formal":        700_000,
	"computational": 650_000,
	"replicated":    550_000,
	"empirical":     500_000,
	"historical":    450_000,
	"social":        400_000,
	"predictive":    350_000,
}

const defaultCategoryBase = 400_000

// lookupCategoryBase returns the base confidence for an epistemic category.
func lookupCategoryBase(category string) uint64 {
	if base, ok := categoryBaseConfidence[category]; ok {
		return base
	}
	return defaultCategoryBase
}

// Challenge economics constants (from x/knowledge/types/types.go).
const (
	MinChallengeStakeUlgm       = 11_000_000 // 11 LGM
	FailedChallengeSlashBps     = 220_000    // 22%
	SuccessfulChallengeRewardBps = 300_000    // 30%
)

// containsDomain checks if a domain list contains a specific domain.
func containsDomain(domains []string, target string) bool {
	for _, d := range domains {
		if d == target {
			return true
		}
	}
	return false
}
