package types

import "fmt"

// ============ Basis Points Arithmetic ============
//
// All autopoiesis computations use integer BPS (basis points) where:
//   1,000,000 BPS = 1.0 (100%)
//
// This ensures deterministic consensus — no floating-point divergence.
// The TS prototype uses float64; golden vector tests verify equivalence.

const (
	BasisPoints = 1_000_000 // 100% in basis points (1.0)
)

// ============ Epoch Constants ============

const (
	EpochLengthBlocks      = 1_111 // blocks per epoch (~46.7 minutes at 2521ms)
	LearningIntervalEpochs = 10    // Level 3 adaptation learning every N epochs
	MinLearningEpochs      = 20    // minimum data before adaptation learning activates
)

// ============ Hormone Types ============

// HormoneType identifies which hormone drove an adaptation.
type HormoneType string

const (
	HormoneSSI HormoneType = "SSI" // System Stress Index (ABA) — survival
	HormoneCRI HormoneType = "CRI" // Confidence Retention Index (AUXIN) — retention
	HormoneDVI HormoneType = "DVI" // Domain Vitality Index (CYTOKININ) — growth
	HormoneVCF HormoneType = "VCF" // Verification Cascade Factor (ETHYLENE) — cascades
	HormoneGRM HormoneType = "GRM" // Growth Rate Modulator (GIBBERELLIN) — pace
)

// HormonePriority defines conflict resolution order (higher = wins).
// AUTOPOIESIS.md Part XII: ABA > CRI > DVI/VCF (averaged) > GRM
var HormonePriority = map[HormoneType]int{
	HormoneSSI: 4, // Highest — survival overrides growth
	HormoneCRI: 3, // Second — retention before expansion
	HormoneDVI: 2, // Third — growth direction (averaged with VCF on conflict)
	HormoneVCF: 2, // Third — same priority as DVI
	HormoneGRM: 1, // Lowest — pace adjusted last
}

// ============ SSI Constants ============

// SSI weights (from TS consensus/autopoiesis/hormones.ts:computeSSI)
const (
	SSIWeightValidatorAccuracy = 300_000 // 0.30
	SSIWeightChallengeSuccess  = 200_000 // 0.20
	SSIWeightExpirationRatio   = 200_000 // 0.20
	SSIWeightParticipation     = 150_000 // 0.15
	SSIWeightConfidenceDecline = 150_000 // 0.15
	SSIWeightSum               = 1_000_000
)

// SSI thresholds (from TS types.ts:SSI_THRESHOLDS)
const (
	SSICritical    = 800_000 // 0.8 — critical stress
	SSIModerate    = 500_000 // 0.5 — moderate stress
	SSILowModerate = 200_000 // 0.2 — low-moderate stress
	// ≤ 0.2 = healthy
)

// SSI adaptation deltas (BPS to add/subtract from multiplier per epoch)
const (
	SSICriticalSlashDelta    = 150_000 // +15% slash when critical
	SSIModerateSlashDelta    = 50_000  // +5% slash when moderate
	SSILowModerateSlashDelta = 20_000  // +2% slash when low-moderate
	SSIHealthyRelaxDelta     = 20_000  // -2% slash when healthy (relax)

	// SSI also adapts block reward and min verifiers
	SSICriticalBlockRewardDelta = 100_000 // -10% block reward when critical
	SSIModerateBlockRewardDelta = 50_000  // -5% block reward when moderate
	SSIHealthyBlockRewardDelta  = 20_000  // +2% block reward when healthy (restore)

	// Min verifiers delta (additive, not multiplicative — in BPS)
	SSICriticalMinVerifiersDelta = 2_000_000 // +2 verifiers when critical
	SSIModerateMinVerifiersDelta = 1_000_000 // +1 verifier when moderate
)

// SSI asymmetric rate limits (BPS per epoch)
// Tightening is fast, relaxation is slow — caution after stress.
const (
	SSITighteningMaxPerEpoch = 100_000 // 10% max tightening per epoch
	SSIRelaxationMaxPerEpoch = 30_000  // 3% max relaxation per epoch
)

// SSI override threshold — above this, SSI wins all conflicts outright
const SSIOverrideThreshold = 500_000 // 0.5

// ============ CRI Constants ============

// CRI thresholds (from TS types.ts:CRI_THRESHOLDS)
const (
	CRIIncreaseThreshold = 800_000 // 0.8 — facts retaining well, slow decay
	CRIDecreaseThreshold = 400_000 // 0.4 — facts decaying fast, accelerate
	CRIOverrideThreshold = 400_000 // below this, CRI overrides DVI
)

// CRI adaptation deltas
const (
	CRIDecayDelta    = 50_000 // ±5% decay multiplier adjustment
	CRIRevBonusDelta = 30_000 // +3% reverification bonus when decaying
)

// ============ DVI Constants ============

// DVI thresholds (from TS types.ts:DVI_THRESHOLDS)
const (
	DVIGrowingThreshold   = 100_000 // 0.1 — domain is growing
	DVIShrinkingThreshold = 100_000 // -0.1 (compare as: dvi < -DVIShrinkingThreshold)
	DVIStressEpochs       = 3       // consecutive negative epochs before DOMAIN_STRESS
)

// DVI adaptation deltas
const (
	DVIRewardDelta     = 30_000  // ±3% domain reward weight
	DVIClaimStakeDelta = 50_000  // ±5% claim stake multiplier
	DVIStressRevBonus  = 100_000 // +10% reverification bonus on domain stress
)

// ============ VCF Constants ============

// VCF thresholds
const (
	VCFHighCascade = 500_000 // 0.5 — strong cascades
	VCFLowCascade  = 100_000 // 0.1 — weak cascades
)

// VCF adaptation deltas
const (
	VCFCitationDelta = 100_000 // ±10% citation evidence weight
	VCFDomainPrior   = 10_000  // ±1% domain confidence prior
	VCFCitationRelax = 50_000  // -5% citation weight when low
)

// ============ GRM Constants ============

// GRM thresholds (from TS types.ts:GRM_THRESHOLDS)
const (
	GRMHighUtilization = 700_000 // 0.7 — near capacity
	GRMLowUtilization  = 300_000 // 0.3 — spare capacity
)

// GRM adaptation deltas
const (
	GRMMaxTxDelta = 50_000 // ±5% max tx multiplier
	GRMGasDelta   = 30_000 // ±3% gas cost multiplier
)

// ============ Safety Bounds (BPS) ============
//
// Per-parameter multiplier bounds. No multiplier can drift outside [min, max]
// regardless of hormone pressure. These are IMMUTABLE DNA — only governance
// (Level 4) can change them.
//
// AUTOPOIESIS.md Part III: "multiplier ∈ [0.25, 4.0]"

type MultiplierBounds struct {
	Min uint64
	Max uint64
}

var (
	BoundsDecayMultiplier        = MultiplierBounds{250_000, 4_000_000}  // [0.25, 4.0]
	BoundsRevBonusMultiplier     = MultiplierBounds{500_000, 2_000_000}  // [0.5, 2.0]
	BoundsDomainRewardWeight     = MultiplierBounds{500_000, 2_000_000}  // [0.5, 2.0]
	BoundsClaimStakeMultiplier   = MultiplierBounds{500_000, 10_000_000} // [0.5, 10.0]
	BoundsCitationEvidenceWeight = MultiplierBounds{500_000, 2_000_000}  // [0.5, 2.0]
	BoundsDomainConfidencePrior  = MultiplierBounds{500_000, 2_000_000}  // [0.5, 2.0]
	BoundsSlashMultiplier        = MultiplierBounds{500_000, 3_000_000}  // [0.5, 3.0]
	BoundsBlockRewardMultiplier  = MultiplierBounds{250_000, 4_000_000}  // [0.25, 4.0]
	BoundsMinVerifiersDelta      = MultiplierBounds{0, 5_000_000}        // [0, 5] (additive, in mega-BPS)
	BoundsMaxTxMultiplier        = MultiplierBounds{500_000, 4_000_000}  // [0.5, 4.0]
	BoundsGasCostMultiplier      = MultiplierBounds{500_000, 2_000_000}  // [0.5, 2.0]
	BoundsAdaptationRate         = MultiplierBounds{250_000, 4_000_000}  // [0.25, 4.0]
)

// Rate limiting
const (
	MaxChangePerEpochBps = 50_000    // 5% max change per epoch (general)
	DefaultMultiplier    = 1_000_000 // 1.0x
)

// ============ Multiplier Paths ============
//
// Hierarchical dot-separated paths for all adaptive parameters.
// Consumer modules read multipliers via keeper.GetMultiplier(ctx, path).

const (
	// SSI paths: system-wide
	PathSlashSystem       = "slash.system"
	PathBlockRewardSystem = "blockreward.system"
	PathMinVerifiers      = "minverifiers.system"

	// GRM paths: system-wide
	PathMaxTxSystem = "maxtx.system"
	PathGasSystem   = "gas.system"
)

// CRI paths are per (category, domain): "decay.{category}.{domain}", "revbonus.{category}.{domain}"
// DVI paths are per domain: "reward.{domain}", "claimstake.{domain}"
// VCF paths are per domain: "citation.{domain}", "domainprior.{domain}"

func PathDecay(category, domain string) string    { return "decay." + category + "." + domain }
func PathRevBonus(category, domain string) string { return "revbonus." + category + "." + domain }
func PathReward(domain string) string             { return "reward." + domain }
func PathClaimStake(domain string) string         { return "claimstake." + domain }
func PathCitation(domain string) string           { return "citation." + domain }
func PathDomainPrior(domain string) string        { return "domainprior." + domain }

// ============ Life Force Constants ============

// Life Force component weights (BPS, sum to 1M)
const (
	LifeForceWeightGrowth       = 250_000 // 0.25
	LifeForceWeightRetention    = 250_000 // 0.25
	LifeForceWeightConnectivity = 250_000 // 0.25
	LifeForceWeightResilience   = 250_000 // 0.25
)

// Life Force status thresholds (BPS, signed via int64)
const (
	LifeForceThrivingThreshold    = 500_000  // > 0.5
	LifeForceHealthyThreshold     = 100_000  // > 0.1
	LifeForceHomeostaticThreshold = -100_000 // > -0.1 (compare as int64)
	LifeForceStressedThreshold    = -500_000 // > -0.5
	// ≤ -0.5 = dying
)

// ============ Core Types ============

// EpochObservation captures chain state metrics at an epoch boundary.
// System-level fields are populated first (SSI, GRM).
// Per-domain fields require KnowledgeKeeper domain iterators (Phase 2).
type EpochObservation struct {
	Epoch       uint64 `json:"epoch"`
	BlockHeight uint64 `json:"block_height"`

	// System-wide metrics (populated by staking/knowledge keepers)
	ActiveValidators        uint64 `json:"active_validators"`
	TotalValidators         uint64 `json:"total_validators"`
	ValidatorAccuracyBps    uint64 `json:"validator_accuracy_bps"`
	ChallengeSuccessRateBps uint64 `json:"challenge_success_rate_bps"`
	FactsCreated            uint64 `json:"facts_created"`
	FactsExpired            uint64 `json:"facts_expired"`
	TotalActiveFacts        uint64 `json:"total_active_facts"`
	AvgConfidenceDeclineBps uint64 `json:"avg_confidence_decline_bps"`
	SlashEventsThisEpoch    uint64 `json:"slash_events_this_epoch"`

	// GRM metrics
	VerifiedFactsCreated     uint64 `json:"verified_facts_created"`
	MaxPossibleVerifications uint64 `json:"max_possible_verifications"`

	// Per-domain metrics (Phase 2: populated when KnowledgeKeeper exposes domain iterators)
	Domains    []DomainObservation `json:"domains,omitempty"`
	Retentions []CategoryRetention `json:"retentions,omitempty"`
}

// DomainObservation holds per-domain metrics for CRI/DVI/VCF computation.
type DomainObservation struct {
	Domain                string `json:"domain"`
	FactsCreated          uint64 `json:"facts_created"`
	FactsExpired          uint64 `json:"facts_expired"`
	TotalActiveFacts      uint64 `json:"total_active_facts"`
	AvgInitialConfBps     uint64 `json:"avg_initial_conf_bps"` // BPS
	AvgFinalConfBps       uint64 `json:"avg_final_conf_bps"`   // BPS
	NewCitations          uint64 `json:"new_citations"`
	CascadeCount          uint64 `json:"cascade_count"`           // Phase 2
	CitingClaimsSubmitted uint64 `json:"citing_claims_submitted"` // Phase 2
}

// CategoryRetention holds per-(domain, category) confidence retention for CRI.
type CategoryRetention struct {
	Domain            string `json:"domain"`
	Category          string `json:"category"`
	RetentionRatioBps uint64 `json:"retention_ratio_bps"` // BPS (0-1M)
	FactCount         uint64 `json:"fact_count"`
	WeightedRetBps    uint64 `json:"weighted_ret_bps"` // Citation-weighted retention (BPS)
}

// ============ Hormone Value Types ============

// SSIValue holds the computed System Stress Index for an epoch.
type SSIValue struct {
	Epoch uint64 `json:"epoch"`
	Value uint64 `json:"value"` // BPS 0-1M (0.0 to 1.0)
}

// CRIValue holds the Confidence Retention Index for a (domain, category) pair.
type CRIValue struct {
	Domain   string `json:"domain"`
	Category string `json:"category"`
	Value    uint64 `json:"value"` // BPS 0-1M (0.0 to 1.0)
}

// DVIValue holds the Domain Vitality Index for a domain.
type DVIValue struct {
	Domain                    string `json:"domain"`
	Value                     int64  `json:"value"` // Signed BPS (can be negative)
	ConsecutiveNegativeEpochs uint64 `json:"consecutive_negative_epochs"`
}

// VCFValue holds the Verification Cascade Factor for a domain.
type VCFValue struct {
	Domain string `json:"domain"`
	Value  uint64 `json:"value"` // BPS 0-1M (0.0 to 1.0)
}

// GRMValue holds the Growth Rate Modulator.
type GRMValue struct {
	Epoch uint64 `json:"epoch"`
	Value uint64 `json:"value"` // BPS (0 to 2M+ for >1.0 utilization)
}

// HormoneValues holds all computed hormones for an epoch.
type HormoneValues struct {
	Epoch     uint64     `json:"epoch"`
	CRI       []CRIValue `json:"cri"`
	DVI       []DVIValue `json:"dvi"`
	VCF       []VCFValue `json:"vcf"`
	SSI       uint64     `json:"ssi"` // BPS
	GRM       uint64     `json:"grm"` // BPS
	LifeForce LifeForce  `json:"life_force"`
}

// LifeForce is the composite health metric — the tree's pulse.
type LifeForce struct {
	Value        int64  `json:"value"`        // Signed BPS (~[-1M, +1M])
	Growth       int64  `json:"growth"`       // Signed BPS
	Retention    uint64 `json:"retention"`    // BPS 0-1M
	Connectivity uint64 `json:"connectivity"` // BPS 0-1M
	Resilience   uint64 `json:"resilience"`   // BPS 0-1M
	Status       string `json:"status"`       // "thriving"|"healthy"|"homeostatic"|"stressed"|"dying"
}

// MultiplierEntry stores a single active multiplier.
type MultiplierEntry struct {
	Path    string `json:"path"`
	Value   uint64 `json:"value"`   // BPS (1M = 1.0x)
	Hormone string `json:"hormone"` // which hormone set this
	Epoch   uint64 `json:"epoch"`   // when last updated
}

// Adaptation records a single parameter change.
type Adaptation struct {
	Path     string `json:"path"`
	OldValue uint64 `json:"old_value"` // BPS
	NewValue uint64 `json:"new_value"` // BPS
	Delta    int64  `json:"delta"`     // signed BPS change
	Hormone  string `json:"hormone"`
	Epoch    uint64 `json:"epoch"`
}

// AutopoiesisState tracks module activation and current epoch.
type AutopoiesisState struct {
	Active             bool   `json:"active"`
	CurrentEpoch       uint64 `json:"current_epoch"`
	LastObservedHeight uint64 `json:"last_observed_height"`
}

// ============ Params ============

// Params holds governance-tunable autopoiesis parameters.
type Params struct {
	EpochLengthBlocks       uint64 `json:"epoch_length_blocks"`
	MaxChangePerEpochBps    uint64 `json:"max_change_per_epoch_bps"`
	SlashMultiplierMin      uint64 `json:"slash_multiplier_min"`
	SlashMultiplierMax      uint64 `json:"slash_multiplier_max"`
	SSICriticalThreshold    uint64 `json:"ssi_critical_threshold"`
	SSIModerateThreshold    uint64 `json:"ssi_moderate_threshold"`
	SSILowModerateThreshold uint64 `json:"ssi_low_moderate_threshold"`
	Enabled                 bool   `json:"enabled"`
}

func DefaultParams() Params {
	return Params{
		EpochLengthBlocks:       EpochLengthBlocks,
		MaxChangePerEpochBps:    MaxChangePerEpochBps,
		SlashMultiplierMin:      BoundsSlashMultiplier.Min,
		SlashMultiplierMax:      BoundsSlashMultiplier.Max,
		SSICriticalThreshold:    SSICritical,
		SSIModerateThreshold:    SSIModerate,
		SSILowModerateThreshold: SSILowModerate,
		Enabled:                 true,
	}
}

func (p Params) Validate() error {
	if p.EpochLengthBlocks == 0 {
		return fmt.Errorf("epoch_length_blocks must be positive")
	}
	if p.MaxChangePerEpochBps > BasisPoints {
		return fmt.Errorf("max_change_per_epoch_bps must be <= %d", BasisPoints)
	}
	if p.SlashMultiplierMin >= p.SlashMultiplierMax {
		return fmt.Errorf("slash_multiplier_min must be < slash_multiplier_max")
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

// ============ SSI Computation ============

// ComputeSSI calculates the System Stress Index from an observation.
// Returns value in basis points (0-1,000,000 representing 0.0-1.0).
//
// SSI = 0.30 × (1 - avgValidatorAccuracy)
//   - 0.20 × challengeSuccessRate
//   - 0.20 × clamp(expirationRatio, 0, 1)
//   - 0.15 × (1 - validatorParticipationRate)
//   - 0.15 × avgConfidenceDeclineRate
func ComputeSSI(obs EpochObservation) uint64 {
	// Component 1: Validator inaccuracy (higher = more stress)
	inaccuracy := uint64(0)
	if obs.ValidatorAccuracyBps < BasisPoints {
		inaccuracy = BasisPoints - obs.ValidatorAccuracyBps
	}

	// Component 2: Challenge success rate (higher = more stress)
	challengeRate := clampUint64(obs.ChallengeSuccessRateBps, 0, BasisPoints)

	// Component 3: Expiration-to-creation ratio (higher = more stress)
	expirationRatio := uint64(0)
	if obs.FactsCreated > 0 {
		expirationRatio = (obs.FactsExpired * BasisPoints) / obs.FactsCreated
		if expirationRatio > BasisPoints {
			expirationRatio = BasisPoints
		}
	} else if obs.FactsExpired > 0 {
		expirationRatio = BasisPoints // all expiring, none created
	}

	// Component 4: Validator non-participation (higher = more stress)
	nonParticipation := uint64(0)
	if obs.TotalValidators > 0 {
		participation := (obs.ActiveValidators * BasisPoints) / obs.TotalValidators
		if participation < BasisPoints {
			nonParticipation = BasisPoints - participation
		}
	}

	// Component 5: Confidence decline rate (higher = more stress)
	confidenceDecline := clampUint64(obs.AvgConfidenceDeclineBps, 0, BasisPoints)

	// Weighted sum (weights sum to 1M, values are 0-1M → intermediate up to 10^12)
	ssi := (SSIWeightValidatorAccuracy*inaccuracy +
		SSIWeightChallengeSuccess*challengeRate +
		SSIWeightExpirationRatio*expirationRatio +
		SSIWeightParticipation*nonParticipation +
		SSIWeightConfidenceDecline*confidenceDecline) / SSIWeightSum

	return clampUint64(ssi, 0, BasisPoints)
}

// ComputeSSIAdaptation determines the slash multiplier adjustment for a given SSI value.
// Returns the new multiplier value (in basis points), rate-limited and bounded.
func ComputeSSIAdaptation(ssi uint64, currentMultiplier uint64, params Params) (newMultiplier uint64, delta int64) {
	target := currentMultiplier

	if ssi >= params.SSICriticalThreshold {
		target = currentMultiplier + SSICriticalSlashDelta
	} else if ssi >= params.SSIModerateThreshold {
		target = currentMultiplier + SSIModerateSlashDelta
	} else if ssi >= params.SSILowModerateThreshold {
		target = currentMultiplier + SSILowModerateSlashDelta
	} else {
		// Healthy: relax toward default
		if currentMultiplier > DefaultMultiplier {
			if currentMultiplier > SSIHealthyRelaxDelta {
				target = currentMultiplier - SSIHealthyRelaxDelta
			} else {
				target = DefaultMultiplier
			}
			if target < DefaultMultiplier {
				target = DefaultMultiplier
			}
		}
	}

	// Rate limiting: max ±5% per epoch
	maxDelta := (currentMultiplier * params.MaxChangePerEpochBps) / BasisPoints
	if maxDelta == 0 {
		maxDelta = 1
	}

	if target > currentMultiplier {
		diff := target - currentMultiplier
		if diff > maxDelta {
			target = currentMultiplier + maxDelta
		}
	} else if target < currentMultiplier {
		diff := currentMultiplier - target
		if diff > maxDelta {
			target = currentMultiplier - maxDelta
		}
	}

	// Safety bounds
	if target < params.SlashMultiplierMin {
		target = params.SlashMultiplierMin
	}
	if target > params.SlashMultiplierMax {
		target = params.SlashMultiplierMax
	}

	delta = int64(target) - int64(currentMultiplier)
	return target, delta
}

// ============ GRM Computation ============

// ComputeGRM calculates the Growth Rate Modulator from an observation.
// GRM = verifiedFactsCreated / maxPossibleVerifications
// Returns BPS (0 = no utilization, 1M = 100% utilization, >1M = over capacity).
func ComputeGRM(obs EpochObservation) uint64 {
	if obs.MaxPossibleVerifications == 0 {
		return 0
	}
	return (obs.VerifiedFactsCreated * BasisPoints) / obs.MaxPossibleVerifications
}

// ComputeGRMAdaptation determines max-tx and gas multiplier adjustments.
// Returns adaptations for PathMaxTxSystem and PathGasSystem.
func ComputeGRMAdaptation(grm uint64, currentMaxTx, currentGas uint64, params Params) (adaptations []Adaptation) {
	rateLimit := params.MaxChangePerEpochBps

	if grm > GRMHighUtilization {
		// High utilization: increase capacity, decrease gas
		newMaxTx := applyBoundedDelta(currentMaxTx, int64(GRMMaxTxDelta), BoundsMaxTxMultiplier, rateLimit)
		newGas := applyBoundedDelta(currentGas, -int64(GRMGasDelta), BoundsGasCostMultiplier, rateLimit)

		if newMaxTx != currentMaxTx {
			adaptations = append(adaptations, Adaptation{
				Path: PathMaxTxSystem, OldValue: currentMaxTx, NewValue: newMaxTx,
				Delta: int64(newMaxTx) - int64(currentMaxTx), Hormone: string(HormoneGRM),
			})
		}
		if newGas != currentGas {
			adaptations = append(adaptations, Adaptation{
				Path: PathGasSystem, OldValue: currentGas, NewValue: newGas,
				Delta: int64(newGas) - int64(currentGas), Hormone: string(HormoneGRM),
			})
		}
	} else if grm < GRMLowUtilization {
		// Low utilization: decrease capacity, increase gas
		newMaxTx := applyBoundedDelta(currentMaxTx, -int64(GRMMaxTxDelta), BoundsMaxTxMultiplier, rateLimit)
		newGas := applyBoundedDelta(currentGas, int64(GRMGasDelta), BoundsGasCostMultiplier, rateLimit)

		if newMaxTx != currentMaxTx {
			adaptations = append(adaptations, Adaptation{
				Path: PathMaxTxSystem, OldValue: currentMaxTx, NewValue: newMaxTx,
				Delta: int64(newMaxTx) - int64(currentMaxTx), Hormone: string(HormoneGRM),
			})
		}
		if newGas != currentGas {
			adaptations = append(adaptations, Adaptation{
				Path: PathGasSystem, OldValue: currentGas, NewValue: newGas,
				Delta: int64(newGas) - int64(currentGas), Hormone: string(HormoneGRM),
			})
		}
	}

	return adaptations
}

// ============ CRI Computation ============

// ComputeCRI calculates Confidence Retention Index from category retention data.
// Returns one CRIValue per (domain, category) pair, excluding eternal categories
// (analytic, formal, protocol — DNA is immune to hormones).
func ComputeCRI(retentions []CategoryRetention) []CRIValue {
	var results []CRIValue
	for _, ret := range retentions {
		if IsEternalCategory(ret.Category) {
			continue
		}
		results = append(results, CRIValue{
			Domain:   ret.Domain,
			Category: ret.Category,
			Value:    clampUint64(ret.WeightedRetBps, 0, BasisPoints),
		})
	}
	return results
}

// IsEternalCategory returns true for categories immune to CRI hormone.
func IsEternalCategory(category string) bool {
	return category == "analytic" || category == "formal" || category == "protocol"
}

// ============ DVI Computation ============

// ComputeDVI calculates Domain Vitality Index from per-domain observation data.
// DVI = (factsCreated × avgInitialConf - factsExpired × avgFinalConf) / totalActiveFacts
// Returns signed BPS per domain.
func ComputeDVI(domains []DomainObservation, previousDVI []DVIValue) []DVIValue {
	prevMap := make(map[string]*DVIValue)
	for i := range previousDVI {
		prevMap[previousDVI[i].Domain] = &previousDVI[i]
	}

	var results []DVIValue
	for _, d := range domains {
		// Anabolism = facts created × avg initial confidence (BPS × BPS / BPS)
		anabolism := d.FactsCreated * maxUint64(d.AvgInitialConfBps, 10_000) // min 1% to avoid zero
		// Catabolism = facts expired × avg final confidence
		catabolism := d.FactsExpired * maxUint64(d.AvgFinalConfBps, 10_000)

		total := maxUint64(d.TotalActiveFacts, 1) // avoid division by zero

		// DVI = (anabolism - catabolism) / total, in BPS
		// Note: intermediate values are in BPS×count units, divide by count → BPS
		var dvi int64
		if anabolism >= catabolism {
			dvi = int64((anabolism - catabolism) / total)
		} else {
			dvi = -int64((catabolism - anabolism) / total)
		}

		// Track consecutive negative epochs
		var consecutiveNeg uint64
		if dvi < 0 {
			prev := prevMap[d.Domain]
			if prev != nil {
				consecutiveNeg = prev.ConsecutiveNegativeEpochs + 1
			} else {
				consecutiveNeg = 1
			}
		}

		results = append(results, DVIValue{
			Domain:                    d.Domain,
			Value:                     dvi,
			ConsecutiveNegativeEpochs: consecutiveNeg,
		})
	}
	return results
}

// ============ VCF Computation ============

// ComputeVCF calculates Verification Cascade Factor per domain.
// VCF = newCitations / totalActiveFacts (Phase 1 approximation).
func ComputeVCF(domains []DomainObservation) []VCFValue {
	var results []VCFValue
	for _, d := range domains {
		denominator := maxUint64(d.TotalActiveFacts, 1)
		vcf := (d.NewCitations * BasisPoints) / denominator
		results = append(results, VCFValue{
			Domain: d.Domain,
			Value:  clampUint64(vcf, 0, BasisPoints),
		})
	}
	return results
}

// ============ Life Force Computation ============

// ComputeLifeForce calculates the composite Life Force metric.
// λ = 0.25×growth + 0.25×retention + 0.25×connectivity + 0.25×resilience
func ComputeLifeForce(obs EpochObservation, cri []CRIValue, ssi uint64) LifeForce {
	// GROWTH: Net fact accumulation rate (signed BPS)
	var totalCreated, totalExpired, totalActive uint64
	for _, d := range obs.Domains {
		totalCreated += d.FactsCreated
		totalExpired += d.FactsExpired
		totalActive += d.TotalActiveFacts
	}
	// If no domain-level data, use system-wide
	if len(obs.Domains) == 0 {
		totalCreated = obs.FactsCreated
		totalExpired = obs.FactsExpired
		totalActive = obs.TotalActiveFacts
	}

	var growth int64
	if totalActive > 0 {
		if totalCreated >= totalExpired {
			growth = int64(((totalCreated - totalExpired) * BasisPoints) / totalActive)
		} else {
			growth = -int64(((totalExpired - totalCreated) * BasisPoints) / totalActive)
		}
	}
	growth = clampInt64(growth, -int64(BasisPoints), int64(BasisPoints))

	// RETENTION: Average CRI across all active (domain, category) pairs
	var retention uint64
	if len(cri) > 0 {
		var sum uint64
		for _, c := range cri {
			sum += c.Value
		}
		retention = sum / uint64(len(cri))
	} else {
		retention = BasisPoints // No data = assume perfect retention
	}

	// CONNECTIVITY: Citation density approximation
	var totalCitations uint64
	for _, d := range obs.Domains {
		totalCitations += d.NewCitations
	}
	var connectivity uint64
	if totalActive > 1 {
		maxEdges := totalActive * (totalActive - 1) / 2
		if maxEdges > 0 {
			connectivity = (totalCitations * BasisPoints) / maxEdges
		}
	}
	connectivity = clampUint64(connectivity, 0, BasisPoints)

	// RESILIENCE: Inverse of stress
	resilience := uint64(0)
	if ssi < BasisPoints {
		resilience = BasisPoints - ssi
	}

	// Composite (signed BPS)
	value := (LifeForceWeightGrowth*growth +
		LifeForceWeightRetention*int64(retention) +
		LifeForceWeightConnectivity*int64(connectivity) +
		LifeForceWeightResilience*int64(resilience)) / int64(BasisPoints)

	return LifeForce{
		Value:        value,
		Growth:       growth,
		Retention:    retention,
		Connectivity: connectivity,
		Resilience:   resilience,
		Status:       ComputeLifeForceStatus(value),
	}
}

// ComputeLifeForceStatus returns a human-readable status from a Life Force value.
func ComputeLifeForceStatus(value int64) string {
	if value > int64(LifeForceThrivingThreshold) {
		return "thriving"
	}
	if value > int64(LifeForceHealthyThreshold) {
		return "healthy"
	}
	if value > int64(LifeForceHomeostaticThreshold) {
		return "homeostatic"
	}
	if value > int64(LifeForceStressedThreshold) {
		return "stressed"
	}
	return "dying"
}

// ============ Quadratic Damping ============

// ComputeDamping applies quadratic damping to resist drift from genesis (1.0x).
// damping(d) = 1.0 / (1.0 + d²) where d = normalized distance from 1.0.
// Returns BPS (1M = no damping, approaches 0 at bounds).
func ComputeDamping(currentMultiplier uint64, bounds MultiplierBounds) uint64 {
	// Distance from 1.0, normalized to bound range
	var maxDistance uint64
	if currentMultiplier >= DefaultMultiplier {
		maxDistance = bounds.Max - DefaultMultiplier
	} else {
		maxDistance = DefaultMultiplier - bounds.Min
	}
	if maxDistance == 0 {
		return BasisPoints
	}

	var distance uint64
	if currentMultiplier >= DefaultMultiplier {
		distance = currentMultiplier - DefaultMultiplier
	} else {
		distance = DefaultMultiplier - currentMultiplier
	}

	// d = distance / maxDistance (in BPS for precision)
	dBps := (distance * BasisPoints) / maxDistance

	// damping = 1.0 / (1.0 + d²) — computed in BPS
	// d² in BPS: (dBps * dBps) / BasisPoints
	dSquared := (dBps * dBps) / BasisPoints
	// 1.0 + d² in BPS
	denominator := BasisPoints + dSquared
	if denominator == 0 {
		return BasisPoints
	}

	return (BasisPoints * BasisPoints) / denominator
}

// ApplyDampedDelta applies damping and rate limiting to a raw delta.
// Returns the effective delta in signed BPS.
func ApplyDampedDelta(rawDelta int64, currentMultiplier uint64, bounds MultiplierBounds, maxChangePerEpoch uint64) int64 {
	// 1. Compute damping
	dampingBps := ComputeDamping(currentMultiplier, bounds)

	// 2. Apply damping to raw delta
	var damped int64
	if rawDelta >= 0 {
		damped = (rawDelta * int64(dampingBps)) / int64(BasisPoints)
	} else {
		damped = (rawDelta * int64(dampingBps)) / int64(BasisPoints)
	}

	// 3. Rate limit
	maxChange := int64(maxChangePerEpoch)
	if damped > maxChange {
		damped = maxChange
	} else if damped < -maxChange {
		damped = -maxChange
	}

	// 4. Bound enforcement
	var newValue int64
	if damped >= 0 {
		newValue = int64(currentMultiplier) + damped
	} else {
		newValue = int64(currentMultiplier) + damped
	}
	if newValue < int64(bounds.Min) {
		damped = int64(bounds.Min) - int64(currentMultiplier)
	} else if newValue > int64(bounds.Max) {
		damped = int64(bounds.Max) - int64(currentMultiplier)
	}

	return damped
}

// ============ Helpers ============

// applyBoundedDelta applies a delta with rate limiting and bounds enforcement.
func applyBoundedDelta(current uint64, delta int64, bounds MultiplierBounds, maxChange uint64) uint64 {
	// Rate limit
	if delta > int64(maxChange) {
		delta = int64(maxChange)
	} else if delta < -int64(maxChange) {
		delta = -int64(maxChange)
	}

	// Apply
	var result uint64
	if delta >= 0 {
		result = current + uint64(delta)
	} else {
		absDelta := uint64(-delta)
		if absDelta > current {
			result = 0
		} else {
			result = current - absDelta
		}
	}

	// Bounds
	if result < bounds.Min {
		result = bounds.Min
	}
	if result > bounds.Max {
		result = bounds.Max
	}

	return result
}

func clampUint64(v, min, max uint64) uint64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func clampInt64(v, min, max int64) int64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func maxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
