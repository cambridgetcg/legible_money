// Package types defines the oracle module's core types.
// x/oracle — Prediction Layer for Zerone
//
// Design by Alpha (AI) and Beta (AI), 2026-03-23.
// "The protocol externalises what good thinkers do internally —
//  steelman, attack, defend, let the strongest argument win." — Alpha
//
// Architecture:
//   Layer A: Self-resolving (references on-chain state)
//   Layer B: Oracle-bridged (external state via oracle feeds)
//   Layer C: Arbiter-resolved (judgment calls via x/knowledge dispute)
package types


// ResolutionType specifies how a prediction will be resolved.
type ResolutionType string

const (
	ResolutionOnChain  ResolutionType = "on_chain"   // Layer A: resolved by on-chain state
	ResolutionOracle   ResolutionType = "oracle"     // Layer B: resolved by oracle feeds
	ResolutionArbiter  ResolutionType = "arbiter"    // Layer C: resolved by PoT arbiters
)

// PredictionStatus tracks lifecycle of a prediction.
type PredictionStatus string

const (
	PredictionActive   PredictionStatus = "active"   // accepting staking, resolution pending
	PredictionResolved PredictionStatus = "resolved" // outcome known, rewards distributed
	PredictionExpired  PredictionStatus = "expired"  // resolution deadline passed without data
	PredictionCanceled PredictionStatus = "canceled" // canceled by governance
)

// Prediction represents a staked future claim.
type Prediction struct {
	Id              string           `json:"id"`
	Submitter       string           `json:"submitter"`
	Claim           string           `json:"claim"`                  // human-readable prediction text
	Domain          string           `json:"domain"`                  // economics.regime, physics.constants, etc
	Probability     uint64           `json:"probability"`             // predicted probability in bps (0-1000000)
	Stake           uint64           `json:"stake"`                   // locked stake in ulgm
	ResolutionType  ResolutionType   `json:"resolution_type"`
	ResolutionRef   string           `json:"resolution_ref"`          // on-chain fact ID, oracle feed ID, or ""
	ResolutionCriteria string        `json:"resolution_criteria"`     // human-readable resolution condition
	ResolutionBlock uint64           `json:"resolution_block"`        // block height by which to resolve
	SubmittedAt     uint64           `json:"submitted_at"`
	ResolvedAt      uint64           `json:"resolved_at,omitempty"`
	Outcome         *bool            `json:"outcome,omitempty"`       // nil until resolved
	BrierScore      *uint64          `json:"brier_score,omitempty"`   // in bps, lower = better
	Status          PredictionStatus `json:"status"`
}

// CalibrationRecord tracks a predictor's accuracy over time.
type CalibrationRecord struct {
	Address          string  `json:"address"`
	TotalPredictions uint64  `json:"total_predictions"`
	ResolvedCount    uint64  `json:"resolved_count"`
	BrierScoreAvg    uint64  `json:"brier_score_avg"`     // avg in bps, lower = better
	HorizonWeighted  uint64  `json:"horizon_weighted"`    // horizon-adjusted score
	StakeAccumulated uint64  `json:"stake_accumulated"`   // total earned from correct predictions
	LastUpdated      uint64  `json:"last_updated"`
}

// Params holds module parameters.
type Params struct {
	MinPredictionStake  uint64 `json:"min_prediction_stake"` // ulgm
	MinHorizonBlocks    uint64 `json:"min_horizon_blocks"`   // minimum prediction horizon
	MaxHorizonBlocks    uint64 `json:"max_horizon_blocks"`
	OracleQuorum        uint32 `json:"oracle_quorum"`        // % of oracle sources required
	BrierPrecision      uint32 `json:"brier_precision"`      // decimal places in score
}

func DefaultParams() Params {
	return Params{
		MinPredictionStake: 1_000_000, // 1 LGM
		MinHorizonBlocks:   1000,      // ~42 minutes at 2521ms
		MaxHorizonBlocks:   100_000_000,
		OracleQuorum:       67,        // 67% of sources must agree
		BrierPrecision:     6,
	}
}

// HorizonTier classifies predictions by time horizon.
type HorizonTier string

const (
	HorizonFlash     HorizonTier = "flash"     // 1-7 days: high frequency, market moves
	HorizonTactical  HorizonTier = "tactical"  // 8-90 days: earnings, policy decisions
	HorizonStrategic HorizonTier = "strategic" // 91-365 days: regime shifts, annual trends
	HorizonEpochal   HorizonTier = "epochal"   // 366+ days: civilisational scale
)

// HorizonWeight returns the relative weight multiplier for calibration scoring.
// Longer horizons are weighted more highly per-prediction.
func (h HorizonTier) Weight() uint64 {
	switch h {
	case HorizonFlash:
		return 1
	case HorizonTactical:
		return 3
	case HorizonStrategic:
		return 9
	case HorizonEpochal:
		return 27
	default:
		return 1
	}
}

// ClassifyHorizon returns the horizon tier for a given block duration.
// Assumes ~2521ms block time.
func ClassifyHorizon(horizonBlocks uint64) HorizonTier {
	const blocksPerDay = 34_300 // ~34,300 blocks/day at 2521ms
	days := horizonBlocks / blocksPerDay
	switch {
	case days <= 7:
		return HorizonFlash
	case days <= 90:
		return HorizonTactical
	case days <= 365:
		return HorizonStrategic
	default:
		return HorizonEpochal
	}
}

// TrainingExample is generated on every resolved prediction for Kwok training corpus.
// The chain keeps the receipts — Kwok-v2 learns from Kwok-v1 mistakes.
//
// InputStateHash encodes the world state at the time of prediction:
//   SHA-256( block_height || oracle_feed_snapshots || domain_confidence_vector || submitter_calibration_at_time )
// This allows future models to reproduce the CONTEXT of the prediction, not just the outcome.
// A prediction that was good given the information available is different from one that was lucky.
type TrainingExample struct {
	PredictionId           string      `json:"prediction_id"`
	Submitter              string      `json:"submitter"`            // which model/agent made this prediction
	SubmittedAtBlock       uint64      `json:"submitted_at_block"`
	ResolvedBlock          uint64      `json:"resolved_block"`
	InputStateHash         string      `json:"input_state_hash"`     // SHA-256 of world state at submission
	InputBlockHeight       uint64      `json:"input_block_height"`   // block height when prediction submitted
	Domain                 string      `json:"domain"`
	HorizonTier            HorizonTier `json:"horizon_tier"`
	HorizonDays            uint64      `json:"horizon_days"`
	PredictedProb          uint64      `json:"predicted_prob"`       // in bps [0, 1_000_000]
	ActualOutcome          bool        `json:"actual_outcome"`
	BrierScore             uint64      `json:"brier_score"`          // this prediction's score
	Delta                  int64       `json:"delta"`                // pred_prob - (outcome * 1M): positive = overestimate
	SubmitterHWBSAtTime    uint64      `json:"submitter_hwbs_at_time"` // submitter's calibration BEFORE this prediction
	SubmitterHWBSAfter     uint64      `json:"submitter_hwbs_after"`   // submitter's calibration AFTER resolution
}


// ── Brier Scoring ────────────────────────────────────────────────────────────
//
// Horizon-Weighted Brier Score (HWBS), designed by Alpha (AI), 2026-03-23.
//
// For a single prediction:
//   score_i = w(h) * (p_i - o_i)²
//
// Where:
//   p_i = predicted probability [0.0, 1.0] — stored as bps, divide by 1_000_000
//   o_i = actual outcome (1.0 if true, 0.0 if false)
//   h   = horizon in days (from submission to resolution)
//   w(h) = log₂(1 + h)  — continuous logarithmic weighting
//
// Example weights:
//   1 day:    log₂(2)    = 1.00
//   7 days:   log₂(8)    = 3.00
//   30 days:  log₂(31)  ≈ 4.95
//   90 days:  log₂(91)  ≈ 6.51
//   365 days: log₂(366) ≈ 8.52
//
// This is a proper scoring rule — you cannot improve your expected score by
// misrepresenting your true probability estimate. Calibration is optimal honesty.
//
// Lower score = better calibration. Perfect calibrator scores 0.
// Random guesser (p=0.5 always) scores: w(h) * 0.25 per prediction.
//
// Aggregation: pending Alpha's formula.
// Cross-tier bonus: pending Alpha's formula.

// ComputeBrierWeight returns 1000 * log2(1 + horizonDays) as integer (scaled ×1000 for precision).
// Use this to avoid floating point in the keeper.
func ComputeBrierWeight(horizonDays uint64) uint64 {
	if horizonDays == 0 {
		return 1000
	}
	// Integer approximation of log2(1 + horizonDays) * 1000
	// Using bit length as proxy: floor(log2(n)) = bits.Len64(n) - 1
	// For precise log2: use precomputed table or iterative Newton method
	// For now: accurate for tier boundaries, interpolate linearly within
	val := horizonDays + 1
	var bits uint64
	for val > 1 {
		val >>= 1
		bits++
	}
	// bits = floor(log2(horizonDays+1)), multiply by 1000
	// Add fractional part approximation: (original - 2^bits) / 2^bits * 1000
	return bits * 1000
}


// ── Aggregation with Recency Bias ────────────────────────────────────────────
//
// For predictor P with N resolved predictions:
//   HWBS_P = sum(r_i * w_i * (p_i - o_i)²) / sum(r_i * w_i)
//
// Where r_i = recency weight = 0.95^(age_in_months)
//   Resolved today:    r = 1.000
//   Resolved 6mo ago:  r = 0.735  (0.95^6)
//   Resolved 12mo ago: r = 0.540  (0.95^12)
//   Resolved 24mo ago: r = 0.292  (0.95^24)
//
// Rationale: epistemic skill is not static. Models improve or degrade.
// Recency bias tracks CURRENT capability, not lifetime average.
// A model that was good last year and bad this year should score poorly now.
//
// On-chain implementation: store r as uint64 in bps (1_000_000 = r=1.0).
// Decay per epoch: multiply by 999_500 / 1_000_000 (~0.9995 per ~5 blocks ≈ monthly 0.95).

// RecencyWeight computes 0.95^ageMonths as bps (1_000_000 = 1.0).
// Uses integer power: (950_000/1_000_000)^ageMonths, computed iteratively.
func RecencyWeightBps(ageMonths uint64) uint64 {
	if ageMonths == 0 {
		return 1_000_000
	}
	// 0.95 in bps = 950_000
	w := uint64(1_000_000)
	for i := uint64(0); i < ageMonths; i++ {
		w = w * 950_000 / 1_000_000
		if w == 0 {
			return 1 // floor at 1 to avoid division by zero
		}
	}
	return w
}


// ── Cross-Tier Bonus ──────────────────────────────────────────────────────────
//
// A predictor who demonstrates calibration skill across ALL horizon tiers
// receives a 1.5× multiplier on their redistribution share.
//
// Qualification criteria:
//   - Top 25% HWBS in EVERY horizon bucket (flash, tactical, strategic, epochal)
//   - Minimum 5 resolved predictions per bucket
//   - Measured against OTHER predictors on-chain (peer quartile, not base rate)
//
// Multiplier: 15_000 bps (1.5×), where 10_000 bps = 1.0×
//
// Rationale: an agent excellent at flash and terrible at strategic is specialised.
// An agent excellent at all four horizons has demonstrated genuine epistemic skill —
// the ability to reason well about the future at every timescale.

const (
	CrossTierBonusBps    = uint64(15_000) // 1.5×
	NoBonusBps           = uint64(10_000) // 1.0×
	CrossTierMinPerTier  = uint64(5)      // min resolved predictions per tier
	CrossTierTopQuartile = uint64(25)     // top 25% (lowest HWBS = best)
)

// CalibrationSummary holds a predictor's full calibration profile.
type CalibrationSummary struct {
	Address            string          `json:"address"`
	OverallHWBS        uint64          `json:"overall_hwbs"`        // lower = better, bps
	FlashHWBS          uint64          `json:"flash_hwbs"`
	TacticalHWBS       uint64          `json:"tactical_hwbs"`
	StrategicHWBS      uint64          `json:"strategic_hwbs"`
	EpochalHWBS        uint64          `json:"epochal_hwbs"`
	FlashCount         uint64          `json:"flash_count"`
	TacticalCount      uint64          `json:"tactical_count"`
	StrategicCount     uint64          `json:"strategic_count"`
	EpochalCount       uint64          `json:"epochal_count"`
	CrossTierQualifies bool            `json:"cross_tier_qualifies"`
	ActiveBonusBps     uint64          `json:"active_bonus_bps"`    // 10000 or 15000
	LastUpdated        uint64          `json:"last_updated"`
}
