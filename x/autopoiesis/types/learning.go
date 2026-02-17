package types

// ============ Meta-Learning Types (Loop 2 — Level 3 Recursion) ============
//
// The meta-learning system tracks which adaptation rules actually improve their
// target metrics over a learning window (LearningIntervalEpochs = 10 epochs).
// Effective rules have their rate multipliers increased; ineffective rules are
// dampened. This creates a second-order feedback loop on top of the hormonal
// adaptation system.

// LearningRecord tracks one adaptation rule's effectiveness over learning windows.
type LearningRecord struct {
	HormoneType      string `json:"hormone_type"`      // SSI, CRI, DVI, VCF, GRM
	MultiplierPath   string `json:"multiplier_path"`    // e.g. "slash.system"
	ApplicationCount uint64 `json:"application_count"`  // times this rule fired
	ImprovementCount uint64 `json:"improvement_count"`  // times target metric improved
	DegradationCount uint64 `json:"degradation_count"`  // times target metric worsened
	EffectivenessBps uint64 `json:"effectiveness_bps"`  // improvement / applications (BPS)
	RateMultiplier   uint64 `json:"rate_multiplier"`    // current adaptation rate (BPS, 1M = 1.0x)
	LastUpdatedEpoch uint64 `json:"last_updated_epoch"` // epoch of last meta-learning update
}

// DefaultRateMultiplier is the starting rate multiplier for new rules.
const DefaultRateMultiplier = BasisPoints // 1.0x

// LearningWindow holds metrics over a learning interval for before/after comparison.
type LearningWindow struct {
	StartEpoch         uint64                `json:"start_epoch"`
	EndEpoch           uint64                `json:"end_epoch"`
	InitialMetrics     map[string]uint64     `json:"initial_metrics"`     // path → metric value at start
	FinalMetrics       map[string]uint64     `json:"final_metrics"`       // path → metric value at end
	AdaptationsApplied []AdaptationSummary   `json:"adaptations_applied"` // rules that fired in this window
}

// AdaptationSummary is a lightweight record of one rule's activity in a window.
type AdaptationSummary struct {
	Path          string `json:"path"`
	HormoneType   string `json:"hormone_type"`
	TotalDelta    int64  `json:"total_delta"`    // sum of deltas over the window
	FireCount     uint64 `json:"fire_count"`     // how many epochs the rule fired
}

// MetaLearningState tracks the current learning window boundaries.
type MetaLearningState struct {
	WindowStartEpoch uint64 `json:"window_start_epoch"`
	LastLearningEpoch uint64 `json:"last_learning_epoch"`
}

// RateAdjustment constants for meta-learning (BPS).
const (
	// When a rule is effective, boost its rate by 10% (multiply by 1.1)
	EffectiveBoostBps = 100_000 // +10%
	// When a rule is ineffective, dampen its rate by 10% (multiply by 0.9)
	IneffectiveDampenBps = 100_000 // -10%
	// Neutral threshold: effectiveness between 40% and 60% is considered neutral
	NeutralLowBps  = 400_000 // 40%
	NeutralHighBps = 600_000 // 60%
	// Maximum proposals per 100 epochs (for Loop 4 rate limiting)
	MaxAutoProposalsPerWindow = 1
	AutoProposalCooldownEpochs = 100
)

// HormoneTargetMetric maps each hormone to its primary "health" metric.
// Used by meta-learning to determine if an adaptation rule improved things.
// The metric key corresponds to a field derivable from EpochObservation.
var HormoneTargetMetric = map[string]string{
	"SSI": "system_stress",        // SSI should decrease (lower is better)
	"CRI": "retention",            // CRI should increase (higher is better)
	"DVI": "domain_vitality",      // DVI should increase (higher is better)
	"VCF": "cascade_factor",       // VCF should increase (higher is better)
	"GRM": "growth_utilization",   // GRM should stay in healthy range (30-70%)
}

// MetricDirectionLowerIsBetter returns true if the metric improves by decreasing.
func MetricDirectionLowerIsBetter(metricKey string) bool {
	return metricKey == "system_stress"
}
