package orchestrator

import (
	"context"
	"sync"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// Tree-aware confidence adjustments (all BPS integer arithmetic).
const (
	refChainBonusPerFact uint32 = 500     // +500 BPS per verified supporting fact
	refChainBonusMax     uint32 = 2000    // max +2000 BPS from reference chain
	counterFactPenalty   uint32 = 1000    // -1000 BPS per active contradiction
	domainDensityBonus   uint32 = 200     // +200 BPS if domain density > densityThreshold
	densityThreshold     uint32 = 50      // facts in domain before bonus applies
	openclawTreeWeight   uint32 = 4000    // 40% weight for OpenClaw in blend
	treeAlgoWeight       uint32 = 6000    // 60% weight for algorithmic in blend
)

// TreeAwareEvaluator wraps StringInstrument with knowledge tree context.
// It adjusts confidence based on ontological strata, logic zones, reference
// chains, counter-facts, and domain density — all using integer BPS arithmetic.
type TreeAwareEvaluator struct {
	inner       *StringInstrument
	ctxProvider KnowledgeContextProvider
	navigator   KnowledgeNavigator
	openclaw    *StructuredOpenClawManager // nil = fall back to algorithmic only
	sessionID   string                     // OpenClaw session for this evaluator
	cache       map[string]cachedEval
	mu          sync.RWMutex
}

// TreeAwareEvaluatorConfig holds configuration for creating a TreeAwareEvaluator.
type TreeAwareEvaluatorConfig struct {
	Inner       *StringInstrument
	CtxProvider KnowledgeContextProvider
	Navigator   KnowledgeNavigator
	OpenClaw    *StructuredOpenClawManager
	SessionID   string
}

// NewTreeAwareEvaluator creates an evaluator that combines StringInstrument
// base evaluation with knowledge tree context adjustments.
func NewTreeAwareEvaluator(cfg TreeAwareEvaluatorConfig) *TreeAwareEvaluator {
	return &TreeAwareEvaluator{
		inner:       cfg.Inner,
		ctxProvider: cfg.CtxProvider,
		navigator:   cfg.Navigator,
		openclaw:    cfg.OpenClaw,
		sessionID:   cfg.SessionID,
		cache:       make(map[string]cachedEval),
	}
}

// Evaluate implements agentsdk.ClaimEvaluator.
// It enriches the base StringInstrument evaluation with knowledge tree context.
func (t *TreeAwareEvaluator) Evaluate(ctx context.Context, claimID string) (agentsdk.Verdict, uint32, error) {
	// Check cache first (commit-reveal determinism)
	t.mu.RLock()
	if cached, ok := t.cache[claimID]; ok {
		t.mu.RUnlock()
		return cached.verdict, cached.confidence, nil
	}
	t.mu.RUnlock()

	// Start with base evaluation from StringInstrument
	_, baseConf, err := t.inner.Evaluate(ctx, claimID)
	if err != nil {
		return agentsdk.VerdictUncertain, 400_000, nil
	}

	confidence := uint32(baseConf)

	// Build knowledge context (graceful degradation: skip tree adjustments on error)
	if t.ctxProvider != nil {
		kctx, err := t.ctxProvider.BuildContext(ctx, claimID)
		if err == nil {
			confidence = t.applyTreeAdjustments(confidence, kctx)
		}
	}

	// Blend with OpenClaw if available
	if t.openclaw != nil && t.sessionID != "" {
		ocConf, err := t.evaluateWithOpenClaw(ctx, claimID)
		if err == nil && ocConf > 0 {
			// Blend: 60% tree + 40% openclaw
			confidence = (confidence*treeAlgoWeight + ocConf*openclawTreeWeight) / 10000
		}
	}

	// Clamp to valid range
	if confidence > maxConfidenceBps {
		confidence = uint32(maxConfidenceBps)
	}

	verdict := confidenceToVerdict(uint64(confidence))

	// Cache for deterministic commit-reveal
	t.mu.Lock()
	t.cache[claimID] = cachedEval{
		verdict:    verdict,
		confidence: confidence,
	}
	t.mu.Unlock()

	return verdict, confidence, nil
}

// applyTreeAdjustments modifies confidence based on knowledge tree context.
// All arithmetic is integer BPS — no floats.
func (t *TreeAwareEvaluator) applyTreeAdjustments(confidence uint32, kctx *KnowledgeContext) uint32 {
	// 1. Stratum ceiling: confidence cannot exceed the ontological stratum cap
	if kctx.StratumCeiling > 0 && confidence > kctx.StratumCeiling {
		confidence = kctx.StratumCeiling
	}

	// 2. Logic zone cap: confidence cannot exceed the logic zone cap
	if kctx.LogicZoneCap > 0 && confidence > kctx.LogicZoneCap {
		confidence = kctx.LogicZoneCap
	}

	// 3. Reference chain bonus: +500 BPS per verified supporting fact (max +2000)
	supportCount := uint32(len(kctx.SupportingFacts))
	bonus := supportCount * refChainBonusPerFact
	if bonus > refChainBonusMax {
		bonus = refChainBonusMax
	}
	confidence += bonus

	// 4. Counter-fact penalty: -1000 BPS per active contradiction
	penalty := uint32(len(kctx.CounterFacts)) * counterFactPenalty
	if penalty > confidence {
		confidence = 0
	} else {
		confidence -= penalty
	}

	// 5. Domain density bonus: +200 BPS if domain has > threshold facts
	if kctx.DomainDensity > densityThreshold {
		confidence += domainDensityBonus
	}

	// 6. Evidence bonus pass-through (from on-chain fact data)
	// This is already encoded in FactInfo; we don't double-count it here.

	return confidence
}

// evaluateWithOpenClaw sends the claim to OpenClaw for AI-powered evaluation.
func (t *TreeAwareEvaluator) evaluateWithOpenClaw(ctx context.Context, claimID string) (uint32, error) {
	if t.ctxProvider == nil {
		return 0, nil
	}

	kctx, err := t.ctxProvider.BuildContext(ctx, claimID)
	if err != nil {
		return 0, err
	}

	// Build supporting fact summaries
	supportFacts := make([]FactSummary, 0, len(kctx.SupportingFacts))
	for _, f := range kctx.SupportingFacts {
		supportFacts = append(supportFacts, FactSummary{
			ID:         f.ID,
			Text:       f.ClaimText,
			Confidence: f.Confidence,
			Maturity:   f.Maturity,
			RefType:    "supports",
		})
	}

	counterFacts := make([]CounterFactSummary, 0, len(kctx.CounterFacts))
	for _, cf := range kctx.CounterFacts {
		counterFacts = append(counterFacts, CounterFactSummary{
			ID:           cf.ID,
			NegationType: cf.NegationType,
			Confidence:   cf.Confidence,
		})
	}

	payload := EvaluateClaimPayload{
		ClaimID:         claimID,
		ClaimText:       kctx.Claim.Text,
		Domain:          kctx.Claim.Domain,
		Category:        kctx.Claim.Category,
		SupportingFacts: supportFacts,
		CounterFacts:    counterFacts,
		DomainDensity:   kctx.DomainDensity,
		StratumCeiling:  kctx.StratumCeiling,
		LogicZoneCap:    kctx.LogicZoneCap,
	}

	_, conf, err := t.openclaw.EvaluateClaim(ctx, t.sessionID, payload)
	return conf, err
}

// ClearCache removes all cached evaluations.
func (t *TreeAwareEvaluator) ClearCache() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cache = make(map[string]cachedEval)
}

// CacheSize returns the number of cached evaluations.
func (t *TreeAwareEvaluator) CacheSize() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.cache)
}

// Compile-time check: TreeAwareEvaluator satisfies ClaimEvaluator.
var _ agentsdk.ClaimEvaluator = (*TreeAwareEvaluator)(nil)
