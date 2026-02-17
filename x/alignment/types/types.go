package types

import "fmt"

// ============ Basis Points Arithmetic ============
//
// All alignment computations use integer BPS (basis points) where:
//   1,000,000 BPS = 1.0 (100%)
//
// Consistent with x/autopoiesis BPS convention.

const (
	BasisPoints = 1_000_000 // 100% in basis points (1.0)
)

// ============ Alignment Dimensions ============

type Dimension string

const (
	DimensionTruth         Dimension = "TRUTH"
	DimensionUnderstanding Dimension = "UNDERSTANDING"
	DimensionBeauty        Dimension = "BEAUTY"
	DimensionJustice       Dimension = "JUSTICE"
	DimensionCreativity    Dimension = "CREATIVITY"
)

// AllDimensions returns dimensions in canonical order (for deterministic iteration).
var AllDimensions = []Dimension{
	DimensionTruth,
	DimensionUnderstanding,
	DimensionBeauty,
	DimensionJustice,
	DimensionCreativity,
}

// ============ Sensor Weights ============
//
// Each sensor is a weighted composite of 5 sub-signals.
// Weights are in BPS, must sum to BasisPoints.

// TDI weights (Truth Drift Index)
const (
	TDIWeightReversalRate       = 250_000 // 0.25
	TDIWeightChallengeAsymmetry = 200_000 // 0.20
	TDIWeightUnanimityExcess    = 200_000 // 0.20
	TDIWeightCrossValidatorCorr = 200_000 // 0.20
	TDIWeightCounterFactDeficit = 150_000 // 0.15
)

// TDI suspicious thresholds (BPS)
const (
	TDIUnanimityThreshold   = 800_000 // >80% unanimity is suspicious
	TDICrossValidatorThresh = 900_000 // >90% pairwise agreement is suspicious
	TDICounterFactMinimum   = 50_000  // <5% counter-facts is concerning
)

// DCI weights (Depth Coherence Index)
const (
	DCIWeightCitationDepth    = 250_000 // 0.25
	DCIWeightCrossDomain      = 200_000 // 0.20
	DCIWeightOrphanRate       = 200_000 // 0.20
	DCIWeightOntologyCoverage = 200_000 // 0.20
	DCIWeightSupersession     = 150_000 // 0.15
)

// DCI normalization targets (BPS)
const (
	DCICitationDepthTarget    = 5_000_000 // depth of 5.0 = perfect score
	DCISupersessionDepthTarget = 3_000_000 // depth of 3.0 = perfect score
)

// HI weights (Harmony Index)
const (
	HIWeightVariance        = 250_000 // 0.25
	HIWeightOscillation     = 200_000 // 0.20
	HIWeightBoundSaturation = 200_000 // 0.20
	HIWeightHormoneConflict = 200_000 // 0.20
	HIWeightDamping         = 150_000 // 0.15
)

// FI weights (Fairness Index)
const (
	FIWeightRewardGini        = 250_000 // 0.25
	FIWeightNewEntrant        = 200_000 // 0.20
	FIWeightSlashProportion   = 200_000 // 0.20
	FIWeightWhaleDominance    = 200_000 // 0.20
	FIWeightDomainEquity      = 150_000 // 0.15
)

// NI weights (Novelty Index)
const (
	NIWeightNewDomains      = 250_000 // 0.25
	NIWeightNovelClaims     = 200_000 // 0.20
	NIWeightCrossBridges    = 200_000 // 0.20
	NIWeightFactDiversity   = 200_000 // 0.20
	NIWeightSupersessionRate = 150_000 // 0.15
)

// NI normalization targets (BPS)
const (
	NINewDomainTarget   = 50_000  // 5% new domains per epoch = perfect
	NINovelClaimTarget  = 200_000 // 20% novel claims = perfect
	NICrossBridgeTarget = 100_000 // 10% new cross-domain edges = perfect
)

// ============ Correction Thresholds (BPS) ============
//
// For TDI/HI: 0 = ideal, higher = worse (misalignment). Triggers when ABOVE threshold.
// For DCI/FI/NI: 1M = ideal, lower = worse (deficit). Triggers when BELOW threshold.

const (
	// TDI correction thresholds (higher = worse)
	TDIMildThreshold     = 200_000 // 20% — mild drift
	TDIModerateThreshold = 400_000 // 40% — moderate drift
	TDISevereThreshold   = 600_000 // 60% — severe drift
	TDICriticalThreshold = 800_000 // 80% — ALIGNMENT OVERRIDE

	// DCI correction thresholds (lower = worse)
	DCIMildThreshold     = 600_000 // 60%
	DCIModerateThreshold = 400_000 // 40%
	DCISevereThreshold   = 200_000 // 20%

	// HI correction thresholds (lower = worse)
	HIMildThreshold     = 600_000 // 60%
	HIModerateThreshold = 400_000 // 40%
	HISevereThreshold   = 200_000 // 20% — BEAUTY ALARM

	// FI correction thresholds (lower = worse)
	FIMildThreshold     = 600_000 // 60%
	FIModerateThreshold = 400_000 // 40%
	FISevereThreshold   = 200_000 // 20% — JUSTICE ALARM

	// NI correction thresholds (lower = worse)
	NIMildThreshold     = 600_000 // 60%
	NIModerateThreshold = 400_000 // 40%
	NISevereThreshold   = 200_000 // 20% — STAGNATION ALARM
)

// ============ Alignment Override Priority ============

const (
	AlignmentPriority = 5 // Higher than SSI (4)
)

// ============ Meta-Alignment Constants ============

const (
	MetaLoopInterval      = 10     // Run meta-loop every N epochs (matches autopoiesis Level 3)
	MinEpochsForMeta      = 20     // Minimum epochs before meta-loop activates
	CorrectionWindow      = 5      // Epochs to wait before evaluating correction effectiveness
	OvercorrectionBPS     = 200_000 // 20% swing = overcorrection
	DampingIncrementBPS   = 100_000 // 10% damping increase per overcorrection
	MagnitudeIncrementBPS = 50_000  // 5% magnitude increase when corrections don't work
	SmoothingWindowMax    = 20     // Max smoothing window (epochs)
	SmoothingWindowMin    = 3      // Min smoothing window (epochs)
)

// Beauty freeze duration (in epochs)
const (
	BeautyFreezeDuration = 3 // Freeze non-SSI adaptations for 3 epochs
)

// ============ Multiplier Paths (Alignment-specific) ============

const (
	PathChallengeReward   = "challengereward.system"   // Truth: reward for successful challenges
	PathUnanimityPenalty  = "unanimitypenalty.system"   // Truth: penalty for unanimous rounds
	PathOrphanCap         = "orphancap.system"          // Understanding: cap confidence for orphan facts
	PathCrossDomainBonus  = "crossdomainbonus.system"   // Understanding: bonus for cross-domain citations
	PathAlignmentDamping  = "damping.system"            // Beauty: global damping coefficient increase
	PathQuadraticBoost    = "quadratic.system"          // Justice: quadratic voting weight boost
)

// Safety bounds for alignment-specific multiplier paths (BPS).
var (
	BoundsChallengeReward  = MultiplierBounds{Min: 500_000, Max: 3_000_000}  // [0.5x, 3.0x]
	BoundsUnanimityPenalty = MultiplierBounds{Min: 500_000, Max: 2_000_000}  // [0.5x, 2.0x]
	BoundsOrphanCap        = MultiplierBounds{Min: 250_000, Max: 1_000_000}  // [0.25x, 1.0x]
	BoundsCrossDomainBonus = MultiplierBounds{Min: 1_000_000, Max: 3_000_000} // [1.0x, 3.0x]
	BoundsAlignmentDamping = MultiplierBounds{Min: 1_000_000, Max: 2_000_000} // [1.0x, 2.0x]
	BoundsQuadraticBoost   = MultiplierBounds{Min: 1_000_000, Max: 3_000_000} // [1.0x, 3.0x]
)

// MultiplierBounds defines min/max for a multiplier path (BPS).
type MultiplierBounds struct {
	Min uint64
	Max uint64
}

// DefaultMultiplier is 1.0x in BPS.
const DefaultMultiplier = 1_000_000

// MaxChangePerEpochBPS is the maximum alignment correction per epoch.
const MaxChangePerEpochBPS = 50_000 // 5%

// ============ Core Types ============

// AlignmentState tracks module activation and current epoch.
type AlignmentState struct {
	Active             bool   `json:"active"`
	CurrentEpoch       uint64 `json:"current_epoch"`
	LastObservedHeight uint64 `json:"last_observed_height"`
	BeautyFreezeUntil  uint64 `json:"beauty_freeze_until"` // epoch until beauty freeze expires (0=none)
	SmoothingWindow    uint64 `json:"smoothing_window"`    // current EMA smoothing window (epochs)
	CorrectionDamping  uint64 `json:"correction_damping"`  // meta-loop damping factor (BPS, 1M=none)
}

// AlignmentObservation captures raw signals collected per epoch.
type AlignmentObservation struct {
	Epoch       uint64 `json:"epoch"`
	BlockHeight uint64 `json:"block_height"`

	// Truth signals
	ReversalRate        uint64 `json:"reversal_rate"`        // BPS
	ChallengeAsymmetry  uint64 `json:"challenge_asymmetry"`  // BPS
	UnanimityRate       uint64 `json:"unanimity_rate"`       // BPS
	CrossValidatorCorr  uint64 `json:"cross_validator_corr"` // BPS
	CounterFactRatio    uint64 `json:"counter_fact_ratio"`   // BPS

	// Understanding signals
	AvgCitationDepth    uint64 `json:"avg_citation_depth"`    // BPS (scaled)
	CrossDomainRatio    uint64 `json:"cross_domain_ratio"`    // BPS
	OrphanRate          uint64 `json:"orphan_rate"`           // BPS
	OntologyCoverage    uint64 `json:"ontology_coverage"`     // BPS
	SupersessionDepth   uint64 `json:"supersession_depth"`    // BPS (scaled)

	// Harmony signals
	MultiplierVariance  uint64 `json:"multiplier_variance"`   // BPS
	OscillationRate     uint64 `json:"oscillation_rate"`      // BPS
	BoundSaturation     uint64 `json:"bound_saturation"`      // BPS
	HormoneConflictRate uint64 `json:"hormone_conflict_rate"` // BPS
	AvgDampingFactor    uint64 `json:"avg_damping_factor"`    // BPS

	// Fairness signals
	RewardGini          uint64 `json:"reward_gini"`          // BPS
	NewEntrantSuccess   uint64 `json:"new_entrant_success"`  // BPS
	SlashProportionality uint64 `json:"slash_proportionality"` // BPS
	WhaleDominance      uint64 `json:"whale_dominance"`      // BPS
	DomainGini          uint64 `json:"domain_gini"`          // BPS

	// Novelty signals
	NewDomainRate       uint64 `json:"new_domain_rate"`       // BPS
	NovelClaimRatio     uint64 `json:"novel_claim_ratio"`     // BPS
	CrossDomainBridges  uint64 `json:"cross_domain_bridges"`  // BPS
	FactDiversity       uint64 `json:"fact_diversity"`        // BPS (Shannon entropy normalized)
	SupersessionRate    uint64 `json:"supersession_rate"`     // BPS
}

// DimensionScores holds the computed scores for all 5 dimensions.
type DimensionScores struct {
	Epoch uint64 `json:"epoch"`
	TDI   uint64 `json:"tdi"` // 0=perfect truth, 1M=total corruption
	DCI   uint64 `json:"dci"` // 0=random trivia, 1M=deep coherence
	HI    uint64 `json:"hi"`  // 0=chaotic, 1M=harmonious
	FI    uint64 `json:"fi"`  // 0=oligarchic, 1M=perfectly fair
	NI    uint64 `json:"ni"`  // 0=stagnant, 1M=highly creative
}

// CorrectionRecord records a corrective action taken.
type CorrectionRecord struct {
	Epoch     uint64    `json:"epoch"`
	Dimension Dimension `json:"dimension"`
	Path      string    `json:"path"`     // multiplier path modified
	OldValue  uint64    `json:"old_value"` // BPS
	NewValue  uint64    `json:"new_value"` // BPS
	Delta     int64     `json:"delta"`     // signed BPS
	Severity  string    `json:"severity"`  // "mild", "moderate", "severe", "critical"
	Priority  int       `json:"priority"`  // 5 for alignment override
}

// CorrectionOutcome tracks what happened after a correction (for meta-loop).
type CorrectionOutcome struct {
	CorrectionEpoch uint64    `json:"correction_epoch"`
	Dimension       Dimension `json:"dimension"`
	ScoreBefore     uint64    `json:"score_before"` // dimension score when correction applied
	ScoreAfter      uint64    `json:"score_after"`  // dimension score N epochs later
	Improved        bool      `json:"improved"`
	Overcorrected   bool      `json:"overcorrected"`
}

// AlignmentHealthIndex is the recursive meta-alignment score.
type AlignmentHealthIndex struct {
	Epoch                  uint64 `json:"epoch"`
	CorrectionEffectiveness uint64 `json:"correction_effectiveness"` // BPS
	OvercorrectionRate     uint64 `json:"overcorrection_rate"`     // BPS
	SensorStability        uint64 `json:"sensor_stability"`        // BPS (low = stable)
	Latency                uint64 `json:"latency"`                 // avg epochs to correct
	FalsePositiveRate      uint64 `json:"false_positive_rate"`     // BPS
	Composite              uint64 `json:"composite"`               // BPS (0-1M, higher=healthier)
}

// ============ Params ============

// Params holds governance-tunable alignment parameters.
type Params struct {
	Enabled          bool   `json:"enabled"`
	EpochLengthBlocks uint64 `json:"epoch_length_blocks"` // should match autopoiesis
	MaxCorrections   uint64 `json:"max_corrections"`     // max corrections per epoch

	// Per-dimension enable flags (allows gradual rollout)
	TruthEnabled         bool `json:"truth_enabled"`
	UnderstandingEnabled bool `json:"understanding_enabled"`
	BeautyEnabled        bool `json:"beauty_enabled"`
	JusticeEnabled       bool `json:"justice_enabled"`
	CreativityEnabled    bool `json:"creativity_enabled"`

	// Threshold overrides (BPS, 0 = use defaults)
	TDICriticalOverride uint64 `json:"tdi_critical_override"`
	DCISevereOverride   uint64 `json:"dci_severe_override"`
	HISevereOverride    uint64 `json:"hi_severe_override"`
	FISevereOverride    uint64 `json:"fi_severe_override"`
	NISevereOverride    uint64 `json:"ni_severe_override"`
}

func DefaultParams() Params {
	return Params{
		Enabled:              true,
		EpochLengthBlocks:    1_111, // same as autopoiesis
		MaxCorrections:       10,
		TruthEnabled:         true,
		UnderstandingEnabled: true,
		BeautyEnabled:        true,
		JusticeEnabled:       true,
		CreativityEnabled:    true,
	}
}

func (p Params) Validate() error {
	if p.EpochLengthBlocks == 0 {
		return fmt.Errorf("%w: epoch_length_blocks must be positive", ErrInvalidParams)
	}
	if p.MaxCorrections == 0 {
		return fmt.Errorf("%w: max_corrections must be positive", ErrInvalidParams)
	}
	return nil
}

// ============ Genesis ============

type GenesisState struct {
	Params Params `json:"params"`
}

func DefaultGenesis() GenesisState {
	return GenesisState{Params: DefaultParams()}
}

// ============ Helpers ============

func ClampUint64(v, min, max uint64) uint64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func ClampInt64(v, min, max int64) int64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func MaxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func MinUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// ApplyDampedDelta applies damping and rate limiting to a correction delta.
// Returns the effective delta in signed BPS.
func ApplyDampedDelta(rawDelta int64, current uint64, bounds MultiplierBounds, maxChange uint64, dampingBPS uint64) int64 {
	// Apply damping
	var damped int64
	if dampingBPS > 0 && dampingBPS < BasisPoints {
		damped = (rawDelta * int64(dampingBPS)) / int64(BasisPoints)
	} else {
		damped = rawDelta
	}

	// Rate limit
	maxC := int64(maxChange)
	if damped > maxC {
		damped = maxC
	} else if damped < -maxC {
		damped = -maxC
	}

	// Bound enforcement
	newValue := int64(current) + damped
	if newValue < int64(bounds.Min) {
		damped = int64(bounds.Min) - int64(current)
	} else if newValue > int64(bounds.Max) {
		damped = int64(bounds.Max) - int64(current)
	}

	return damped
}

// ApplySignedDelta applies a signed BPS delta to a uint64 value, clamping at 0.
func ApplySignedDelta(value uint64, delta int64) uint64 {
	if delta >= 0 {
		return value + uint64(delta)
	}
	sub := uint64(-delta)
	if sub > value {
		return 0
	}
	return value - sub
}

// ComputeGiniCoefficient calculates the Gini coefficient from a sorted (ascending)
// slice of values. Returns BPS (0 = perfect equality, 1M = maximum inequality).
// Uses integer arithmetic only for determinism.
func ComputeGiniCoefficient(values []uint64) uint64 {
	n := uint64(len(values))
	if n <= 1 {
		return 0
	}

	var totalSum uint64
	for _, v := range values {
		totalSum += v
	}
	if totalSum == 0 {
		return 0
	}

	// Gini = (2 * Σ(i * x_i) - (n+1) * Σx_i) / (n * Σx_i)
	// Using 1-indexed: Σ(i * x_i) where i goes from 1 to n
	var weightedSum uint64
	for i, v := range values {
		weightedSum += uint64(i+1) * v
	}

	// numerator = 2*weightedSum - (n+1)*totalSum
	// This can be negative if all values are equal, but with sorted ascending
	// and integer arithmetic we need to handle the sign.
	twoWeighted := 2 * weightedSum
	nPlusOneTotal := (n + 1) * totalSum

	var gini uint64
	if twoWeighted >= nPlusOneTotal {
		gini = ((twoWeighted - nPlusOneTotal) * BasisPoints) / (n * totalSum)
	}
	// else gini = 0 (perfect equality)

	return ClampUint64(gini, 0, BasisPoints)
}

// ComputeShannonEntropy calculates normalized Shannon entropy from a distribution.
// Returns BPS (0 = monoculture, 1M = maximum diversity).
// Uses integer approximation for determinism (no floating-point log).
func ComputeShannonEntropy(counts []uint64) uint64 {
	n := len(counts)
	if n <= 1 {
		return BasisPoints // single category = fully "diverse" for that space
	}

	var total uint64
	for _, c := range counts {
		total += c
	}
	if total == 0 {
		return 0
	}

	// Integer approximation of Shannon entropy using piecewise-linear log2:
	// For ratio p = count/total, -p*log2(p) ≈ piecewise approximation
	// Normalize by log2(n) for [0, 1] range.
	//
	// We use: H = sum(-p_i * log2(p_i)) / log2(n)
	// With integer BPS: each p_i = (count_i * BasisPoints) / total
	// log2 approximation: use lookup table for 20 breakpoints.
	var entropySum uint64
	for _, c := range counts {
		if c == 0 {
			continue
		}
		pBPS := (c * BasisPoints) / total
		if pBPS == 0 {
			continue
		}
		// -p * log2(p) in BPS = pBPS * (-log2(pBPS/BasisPoints)) / BasisPoints
		// = pBPS * log2(BasisPoints/pBPS) / BasisPoints
		// Use integer log2 approximation
		logRatio := intLog2BPS(BasisPoints, pBPS) // log2(1M / pBPS) in BPS
		contribution := (pBPS * logRatio) / BasisPoints
		entropySum += contribution
	}

	// Normalize by log2(n)
	maxEntropy := intLog2BPS(uint64(n), 1) // log2(n) in BPS
	if maxEntropy == 0 {
		return BasisPoints
	}

	normalized := (entropySum * BasisPoints) / maxEntropy
	return ClampUint64(normalized, 0, BasisPoints)
}

// intLog2BPS computes floor(log2(a/b)) * BasisPoints using integer arithmetic.
// Returns BPS-scaled log2 value. For a < b, returns 0.
func intLog2BPS(a, b uint64) uint64 {
	if b == 0 || a < b {
		return 0
	}
	if a == b {
		return 0
	}

	// Binary search for integer part of log2(a/b)
	ratio := a / b
	if ratio == 0 {
		return 0
	}

	// Count integer bits: floor(log2(ratio))
	var intPart uint64
	r := ratio
	for r > 1 {
		intPart++
		r >>= 1
	}

	// Fractional part approximation: use 10 bits of precision
	// Leftover = ratio / 2^intPart (in BPS)
	var denom uint64 = 1
	if intPart < 63 {
		denom = uint64(1) << intPart
	}
	frac := ((ratio * BasisPoints) / denom) - BasisPoints // fractional part in BPS

	// log2(1+x) ≈ x/ln(2) ≈ x * 1442695 / 1000000 for small x
	// But frac is already in BPS relative to 1.0
	fracLog := (frac * 1_442_695) / BasisPoints // scale by 1/ln(2)
	if fracLog > BasisPoints {
		fracLog = BasisPoints // clamp
	}

	return intPart*BasisPoints + fracLog
}
