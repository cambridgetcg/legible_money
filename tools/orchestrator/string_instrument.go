package orchestrator

import (
	"context"
	"fmt"
	"sync"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// VerificationStrategy controls how aggressively the verifier evaluates claims.
type VerificationStrategy int

const (
	// StrategyConservative applies high thresholds — fewer acceptances.
	StrategyConservative VerificationStrategy = iota
	// StrategyAggressive applies lower thresholds — more acceptances.
	StrategyAggressive
	// StrategySpecialist boosts confidence for claims in specialist domains.
	StrategySpecialist
)

// Strategy adjustment factors (basis points, 10000 = 1.0x).
const (
	conservativeMultiplierBps = 8000  // 0.8x
	aggressiveMultiplierBps   = 12000 // 1.2x
	specialistMultiplierBps   = 11000 // 1.1x
	maxConfidenceBps          = 1_000_000
)

// Verdict thresholds.
const (
	verdictTrueThreshold    = 500_000
	verdictFalseThreshold   = 300_000
)

// cachedEval holds a cached evaluation result for commit-reveal determinism.
type cachedEval struct {
	verdict    agentsdk.Verdict
	confidence uint32
}

// StringInstrument implements domain-aware verification (the "strings" section).
// It satisfies agentsdk.ClaimEvaluator for use with Agent.AutoVerify().
type StringInstrument struct {
	strategy VerificationStrategy
	domains  []string // specialist domains (only used with StrategySpecialist)
	resolver ClaimResolver
	cache    map[string]cachedEval // claimID → cached result
	mu       sync.RWMutex
}

// NewStringInstrument creates a verifier instrument.
func NewStringInstrument(strategy VerificationStrategy, domains []string, resolver ClaimResolver) *StringInstrument {
	return &StringInstrument{
		strategy: strategy,
		domains:  domains,
		resolver: resolver,
		cache:    make(map[string]cachedEval),
	}
}

// Name returns the instrument name.
func (s *StringInstrument) Name() string { return "string" }

// Section returns the strings section.
func (s *StringInstrument) Section() SectionName { return SectionStrings }

// PrepareAction injects verification parameters before execution.
func (s *StringInstrument) PrepareAction(ctx context.Context, agent *agentsdk.Agent, action ActionType, params map[string]string) (map[string]string, error) {
	if action != ActionVerify && action != ActionAutoVerify {
		return params, nil // pass through non-verify actions
	}

	// For ActionVerify with a specific round_id, evaluate and inject verdict/confidence
	if action == ActionVerify {
		claimID := params["claim_id"]
		if claimID == "" {
			return params, nil
		}

		verdict, confidence, err := s.Evaluate(ctx, claimID)
		if err != nil {
			return nil, fmt.Errorf("evaluating claim %s: %w", claimID, err)
		}

		result := copyParams(params)
		result["verdict"] = string(verdict)
		result["confidence"] = fmt.Sprintf("%d", confidence)
		return result, nil
	}

	return params, nil
}

// Evaluate implements agentsdk.ClaimEvaluator.
// It is deterministic between commit and reveal phases via caching.
func (s *StringInstrument) Evaluate(ctx context.Context, claimID string) (agentsdk.Verdict, uint32, error) {
	// Check cache first (commit-reveal determinism)
	s.mu.RLock()
	if cached, ok := s.cache[claimID]; ok {
		s.mu.RUnlock()
		return cached.verdict, cached.confidence, nil
	}
	s.mu.RUnlock()

	// Resolve the claim
	var category string
	var domain string
	if s.resolver != nil {
		info, err := s.resolver.ResolveClaim(ctx, claimID)
		if err != nil {
			// Can't resolve — return uncertain
			return agentsdk.VerdictUncertain, 400_000, nil
		}
		category = info.Category
		domain = info.Domain
	}

	// Look up base confidence for the category
	base := lookupCategoryBase(category)

	// Apply strategy multiplier
	adjusted := applyStrategy(base, s.strategy, s.domains, domain)

	// Determine verdict
	verdict := confidenceToVerdict(adjusted)

	// Cache for deterministic commit-reveal
	s.mu.Lock()
	s.cache[claimID] = cachedEval{
		verdict:    verdict,
		confidence: uint32(adjusted),
	}
	s.mu.Unlock()

	return verdict, uint32(adjusted), nil
}

// ClearCache removes all cached evaluations.
func (s *StringInstrument) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]cachedEval)
}

// CacheSize returns the number of cached evaluations.
func (s *StringInstrument) CacheSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.cache)
}

// applyStrategy adjusts confidence using integer BPS arithmetic.
func applyStrategy(base uint64, strategy VerificationStrategy, specialistDomains []string, claimDomain string) uint64 {
	var multiplier uint64
	switch strategy {
	case StrategyConservative:
		multiplier = conservativeMultiplierBps
	case StrategyAggressive:
		multiplier = aggressiveMultiplierBps
	case StrategySpecialist:
		if containsDomain(specialistDomains, claimDomain) {
			multiplier = specialistMultiplierBps
		} else {
			multiplier = 10000 // 1.0x for non-specialist domains
		}
	default:
		multiplier = 10000
	}

	adjusted := (base * multiplier) / 10000
	if adjusted > maxConfidenceBps {
		adjusted = maxConfidenceBps
	}
	return adjusted
}

// confidenceToVerdict maps a confidence value to a verdict.
func confidenceToVerdict(confidence uint64) agentsdk.Verdict {
	if confidence >= verdictTrueThreshold {
		return agentsdk.VerdictTrue
	}
	if confidence < verdictFalseThreshold {
		return agentsdk.VerdictFalse
	}
	return agentsdk.VerdictUncertain
}

// Compile-time check: StringInstrument satisfies ClaimEvaluator.
var _ agentsdk.ClaimEvaluator = (*StringInstrument)(nil)

// copyParams creates a shallow copy of params.
func copyParams(params map[string]string) map[string]string {
	if params == nil {
		return make(map[string]string)
	}
	result := make(map[string]string, len(params))
	for k, v := range params {
		result[k] = v
	}
	return result
}
