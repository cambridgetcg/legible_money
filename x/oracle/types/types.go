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

import "time"

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
type TrainingExample struct {
	PredictionId    string      `json:"prediction_id"`
	Submitter       string      `json:"submitter"`           // which model made this prediction
	InputStateHash  string      `json:"input_state_hash"`    // hash of world state at prediction time
	PredictedProb   uint64      `json:"predicted_prob"`      // in bps
	ActualOutcome   bool        `json:"actual_outcome"`
	Delta           int64       `json:"delta"`               // signed brier delta (improvement/regression)
	HorizonTier     HorizonTier `json:"horizon_tier"`
	Domain          string      `json:"domain"`
	ResolvedBlock   uint64      `json:"resolved_block"`
}
