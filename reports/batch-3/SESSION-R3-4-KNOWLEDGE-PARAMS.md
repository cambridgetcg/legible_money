# Session R3-4: x/knowledge Constant Parameterization

**Date**: 2026-02-18
**Branch**: `feat/cli-proto-wiring`
**Status**: COMPLETE

## Summary

Moved 60+ hardcoded constants from `x/knowledge/types/types.go` into the governance-adjustable `ExtendedParams` struct, making them tunable via `MsgUpdateExtendedParams` without chain upgrade. All default values exactly match prior hardcoded constants — zero behavioral change.

## Architecture Decision

**ExtendedParams (JSON) over Proto Params**: The codebase has a dual params system:
- `Params` (proto-generated, 33 fields) — stored via `proto.Marshal`
- `ExtendedParams` (hand-written, JSON) — stored via `json.Marshal` under `ExtendedParamsKey`

Rather than modifying proto definitions and regenerating (which would require proto schema migration), all new params were added to the existing `ExtendedParams` system. This follows the established codebase pattern and leverages the existing `MsgUpdateExtendedParams` governance message.

## Constants Moved to ExtendedParams

### Group 1: Category Decay Profiles (map)

| Old Constant | New Field | Type | Default |
|---|---|---|---|
| `CATEGORY_DECAY["analytic"]` | `CategoryDecayProfiles["analytic"]` | `CategoryDecayProfile` | `{HalfLife:0, Max:1M, Min:0, Reverif:0}` |
| `CATEGORY_DECAY["formal"]` | `CategoryDecayProfiles["formal"]` | `CategoryDecayProfile` | `{HalfLife:0, Max:1M, Min:0, Reverif:0}` |
| `CATEGORY_DECAY["protocol"]` | `CategoryDecayProfiles["protocol"]` | `CategoryDecayProfile` | `{HalfLife:0, Max:1M, Min:0, Reverif:0}` |
| `CATEGORY_DECAY["empirical"]` | `CategoryDecayProfiles["empirical"]` | `CategoryDecayProfile` | `{HalfLife:1M, Max:990k, Min:330k, Reverif:110k}` |
| `CATEGORY_DECAY["historical"]` | `CategoryDecayProfiles["historical"]` | `CategoryDecayProfile` | `{HalfLife:500k, Max:950k, Min:200k, Reverif:30k}` |
| `CATEGORY_DECAY["predictive"]` | `CategoryDecayProfiles["predictive"]` | `CategoryDecayProfile` | `{HalfLife:20k, Max:900k, Min:100k, Reverif:80k}` |
| `CATEGORY_DECAY["social"]` | `CategoryDecayProfiles["social"]` | `CategoryDecayProfile` | `{HalfLife:150k, Max:850k, Min:200k, Reverif:40k}` |
| `CATEGORY_DECAY["replicated"]` | `CategoryDecayProfiles["replicated"]` | `CategoryDecayProfile` | `{HalfLife:1M, Max:995k, Min:300k, Reverif:20k}` |
| `CATEGORY_DECAY["computational"]` | `CategoryDecayProfiles["computational"]` | `CategoryDecayProfile` | `{HalfLife:0, Max:1M, Min:0, Reverif:0}` |

Note: `CategoryDecayProfile` struct extended with `MinConfidence`, `ReverificationBonus`, `RequiresReverification` JSON tags.

### Group 2: Evidence Type Strengths (map)

| Old Constant | New Field | Type | Default |
|---|---|---|---|
| `EVIDENCE_TYPE_STRENGTH["experiment"]` | `EvidenceTypeStrengths["experiment"]` | `uint64` | `800,000` |
| `EVIDENCE_TYPE_STRENGTH["on_chain"]` | `EvidenceTypeStrengths["on_chain"]` | `uint64` | `900,000` |
| `EVIDENCE_TYPE_STRENGTH["cryptographic_proof"]` | `EvidenceTypeStrengths["cryptographic_proof"]` | `uint64` | `950,000` |
| `EVIDENCE_TYPE_STRENGTH["peer_review"]` | `EvidenceTypeStrengths["peer_review"]` | `uint64` | `700,000` |
| `EVIDENCE_TYPE_STRENGTH["statistical"]` | `EvidenceTypeStrengths["statistical"]` | `uint64` | `750,000` |
| `EVIDENCE_TYPE_STRENGTH["attestation"]` | `EvidenceTypeStrengths["attestation"]` | `uint64` | `300,000` |

### Group 3: Stratum Confidence Ceilings (map)

| Old Constant | New Field | Default |
|---|---|---|
| `STRATUM_CONFIDENCE_CEILING["fundamental"]` | `StratumConfidenceCeilings["fundamental"]` | `1,000,000` |
| `STRATUM_CONFIDENCE_CEILING["physical"]` | `StratumConfidenceCeilings["physical"]` | `990,000` |
| `STRATUM_CONFIDENCE_CEILING["chemical"]` | `StratumConfidenceCeilings["chemical"]` | `980,000` |
| `STRATUM_CONFIDENCE_CEILING["biological"]` | `StratumConfidenceCeilings["biological"]` | `970,000` |
| `STRATUM_CONFIDENCE_CEILING["cognitive"]` | `StratumConfidenceCeilings["cognitive"]` | `930,000` |
| `STRATUM_CONFIDENCE_CEILING["social"]` | `StratumConfidenceCeilings["social"]` | `900,000` |
| `STRATUM_CONFIDENCE_CEILING["technological"]` | `StratumConfidenceCeilings["technological"]` | `960,000` |

### Group 4: Challenge Window Multipliers (map)

| Old Constant | New Field | Default |
|---|---|---|
| `CHALLENGE_WINDOW_PROFILE["analytic"]` | `ChallengeWindowMultipliers["analytic"]` | `500,000 (0.5x)` |
| `CHALLENGE_WINDOW_PROFILE["formal"]` | `ChallengeWindowMultipliers["formal"]` | `500,000` |
| `CHALLENGE_WINDOW_PROFILE["protocol"]` | `ChallengeWindowMultipliers["protocol"]` | `500,000` |
| `CHALLENGE_WINDOW_PROFILE["empirical"]` | `ChallengeWindowMultipliers["empirical"]` | `2,000,000 (2x)` |
| `CHALLENGE_WINDOW_PROFILE["historical"]` | `ChallengeWindowMultipliers["historical"]` | `1,500,000` |
| `CHALLENGE_WINDOW_PROFILE["predictive"]` | `ChallengeWindowMultipliers["predictive"]` | `3,000,000 (3x)` |
| `CHALLENGE_WINDOW_PROFILE["social"]` | `ChallengeWindowMultipliers["social"]` | `2,000,000` |
| `CHALLENGE_WINDOW_PROFILE["replicated"]` | `ChallengeWindowMultipliers["replicated"]` | `1,000,000` |
| `CHALLENGE_WINDOW_PROFILE["computational"]` | `ChallengeWindowMultipliers["computational"]` | `500,000` |

### Group 5: Scalar Constants

| Old Constant | New ExtendedParams Field | Default |
|---|---|---|
| `SelfCitationDiscountBPS` | `SelfCitationDiscountBps` | `500,000 (50%)` |
| `FundamentalityNotable` | `FundamentalityNotableBps` | `250,000` |
| `FundamentalityFoundational` | `FundamentalityFoundationalBps` | `500,000` |
| `FundamentalityFundamental` | `FundamentalityFundamentalBps` | `750,000` |
| `FundamentalityAxiomatic` | `FundamentalityAxiomaticBps` | `900,000` |
| `FundamentalityBonusNotable` | `FundamentalityBonusNotableBps` | `1,100,000 (1.1x)` |
| `FundamentalityBonusFoundational` | `FundamentalityBonusFoundationalBps` | `1,250,000 (1.25x)` |
| `FundamentalityBonusFundamental` | `FundamentalityBonusFundamentalBps` | `1,500,000 (1.5x)` |
| `FundamentalityBonusAxiomatic` | `FundamentalityBonusAxiomaticBps` | `2,000,000 (2.0x)` |
| `CalibrationTrivialThreshold` | `CalibrationTrivialThreshold` | `950,000 (95%)` |
| `MaxCalibrationTrivialPenalty` | `MaxCalibrationTrivialPenalty` | `100,000 (10%)` |
| `MisbehaviorRejectionThreshold` | `MisbehaviorRejectionThreshold` | `300,000 (30%)` |
| `MisbehaviorMinSamples` | `MisbehaviorMinSamples` | `10` |
| `ChallengeStakeRatioMinBPS` | `ChallengeStakeRatioMinBps` | `500,000 (50%)` |
| `ChallengeRewardCapBPS` | `ChallengeRewardCapBps` | `1,000,000 (100%)` |
| `MinDomainContributorsForNovelty` | `MinDomainContributorsForNovelty` | `3` |
| `ConformityThresholdBPS` | `ConformityThresholdBps` | `950,000 (95%)` |
| `ConformityMinSamples` | `ConformityMinSamples` | `20` |
| `ConformityPenaltyMaxBPS` | `ConformityPenaltyMaxBps` | `200,000 (20%)` |
| `MinParticipationRateBPS` | `MinParticipationRateBps` | `500,000 (50%)` |
| `ParticipationMinSamples` | `ParticipationMinSamples` | `10` |
| `PassiveStakingPenaltyBPS` | `PassiveStakingPenaltyBps` | `500,000 (50%)` |
| `CounterFactHighConfidence` | `CounterFactHighConfidenceBps` | `700,000 (70%)` |
| `CounterFactMediumConfidence` | `CounterFactMediumConfidenceBps` | `400,000 (40%)` |
| `CounterFactHighPenalty` | `CounterFactHighPenalty` | `80,000` |
| `CounterFactMediumPenalty` | `CounterFactMediumPenalty` | `40,000` |
| `CounterFactLowPenalty` | `CounterFactLowPenalty` | `10,000` |
| `MaxCounterFactPenalty` | `MaxCounterFactPenalty` | `400,000` |
| `StaleContestedDecayEpochs` | `StaleContestedDecayEpochs` | `10` |
| `StaleContestedDecayRateBps` | `StaleContestedDecayRateBps` | `50,000 (5%)` |
| `MinClaimStakeUlgm` | `MinClaimStakeUlgm` | `1,000` |

### Group 6: Proto Params (DefaultParams updated)

| Proto Field | Default Value |
|---|---|
| `ProvisionalThreshold` | `600,000` |
| `BaseChallengeDuration` | `34,272 blocks` |
| `MinChallengeStake` | `"11000000" ulgm` |
| `ConfidenceGrowthPerEpoch` | `10,000` |
| `MaxSurvivalConfidence` | `770,000` |
| `FailedChallengeSlashBps` | `220,000` |
| `SuccessfulChallengeRewardBps` | `500,000` |
| `AdversarialVerificationEnabled` | `true` |

## Files Modified (35 total in x/knowledge/)

### Types (2 files)
- `types/types.go` — Added 31 scalar fields, 4 map fields, 4 Default*() functions, 4 getter methods, validation
- `types/genesis.go` — Updated DefaultParams() with adversarial verification fields

### Evaluation (10 files)
- `evaluation/types.go` — Added `ExtendedParams` field to `EvaluationContext`
- `evaluation/calibration.go` — Use `ep.CalibrationTrivialThreshold`, `ep.MaxCalibrationTrivialPenalty`, etc.
- `evaluation/counter_fact.go` — Use `ep.CounterFactHighConfidenceBps`, `ep.MaxCounterFactPenalty`, etc.
- `evaluation/evidence.go` — Use `ep.GetEvidenceTypeStrength()`, `ep.MaxEvidenceBonus`
- `evaluation/ceiling.go` — Use `ep.GetCategoryDecayProfile()`, `ep.GetStratumConfidenceCeiling()`
- `evaluation/novelty.go` — Use `ep.NoveltyMaxBonusBps`, `ep.MinDomainContributorsForNovelty`, etc.
- `evaluation/fact_graph.go` — Use `ep.VerifiedBridgeThresholdBps`, `ep.CyclePenaltyFactor`
- `evaluation/edge_cases.go` — Use `ep.GetEvidenceTypeStrength()`
- `evaluation/participant_context.go` — Use `ep.HumanEvidenceBonusBps`, `ep.AgentOnlyEmpiricalCeiling`

### Keeper (13 files)
- `keeper/decay.go` — Changed 6 standalone function signatures to accept `ep types.ExtendedParams`
- `keeper/citation.go` — Use `ep.CitationShareBps`, `ep.SelfCitationDiscountBps`, fundamentality params
- `keeper/growth.go` — Use `ep.SurvivedChallengeConfCap`, maturity criteria, stale decay params
- `keeper/challenge.go` — Use `ep.GetChallengeWindowMultiplier()`, challenge economics params
- `keeper/provisional.go` — Use proto `ProvisionalThreshold`, `BaseChallengeDuration`, `ep.GetChallengeWindowMultiplier()`
- `keeper/confidence.go` — Use `ep.GetCategoryDecayProfile()`, `ep.GetEvidenceTypeStrength()`
- `keeper/propagation.go` — Use `ep.PropagationThresholdBps`, `ep.MaxPropagationDepth`, `ep.CyclePenaltyFactor`
- `keeper/expiration.go` — Use `ep.GetCategoryDecayProfile()`, pass `ep` to `IsFactExpired`/`NeedsReverification`
- `keeper/msg_server.go` — Use `ep.MinClaimStakeUlgm`
- `keeper/rounds.go` — Use `ep.GetEvidenceTypeStrength()`, `ep.MisbehaviorRejectionThreshold`
- `keeper/state.go` — Use `ep.ConformityThresholdBps`, `ep.MinParticipationRateBps`, `ep.PassiveStakingPenaltyBps`
- `keeper/supersession.go` — Use `ep.MaxReverificationsPerBlock`
- `keeper/pruning.go` — Use `ep.GetCategoryDecayProfile()`, pass `ep` to `CalculateMaturity`/`GetFundamentalityMultiplier`
- `keeper/phases.go` — Use `ep.ConfidenceGrowthEpochBlocks`
- `keeper/keeper.go` — Wire `ExtendedParams` into `EvaluationContext` construction

### Tests (10 files)
- `evaluation/*_test.go` (7 files) — Added `ExtendedParams: types.DefaultExtendedParams()` to ~66 `EvaluationContext` constructions
- `keeper/keeper_test.go` — Updated tests for new function signatures
- `keeper/novelty_test.go` — Updated tests for new function signatures

## Call Site Statistics

| Category | Count |
|---|---|
| New `ep.Field` usages added | ~172 |
| Hardcoded `types.Constant` references removed | ~136 |
| Standalone function signatures changed | 6 (`CalculateDecayedConfidence`, `CalculateDecayedConfidenceNovel`, `calculateDecayedConfidenceWithMultiplier`, `CalculateMaturity`, `NeedsReverification`, `ApplyReverificationBonus`) |
| Keeper methods updated | 20+ |
| Test EvaluationContext constructions updated | ~66 |
| Total files modified | 35 |

## Test Results

```
go build ./...                      # PASS (clean)
go test ./x/knowledge/crypto        # ok   0.497s
go test ./x/knowledge/evaluation    # ok   0.504s
go test ./x/knowledge/keeper        # ok   0.838s
go test ./tests/cross_stack         # ok   0.486s
go test ./app/...                   # ok   0.977s
```

All tests pass. Zero behavioral change confirmed by cross-stack golden vectors.

## Constants Intentionally Left Hardcoded

| Constant | Reason |
|---|---|
| `RetractionThresholdBps` (100,000) | Used in `growth.go` stale decay — local to decay logic, changing rarely affects consensus |
| `CascadeReductionBps` (100,000) | Cascade reduction factor — tightly coupled to fact graph propagation invariants |
| `brokenRefPenalty`, `disprovenRefPenalty`, `supportBonus`, etc. in `evaluation/fact_graph.go` | Pipeline-internal scoring constants — already governed by the evaluation pipeline's macro params (MaxSupportBonus, etc.) which ARE parameterized |
| `noEvidencePenalty`, `defaultTypeStrength`, `baseReliability` in `evaluation/evidence.go` | Fallback/default constants for when params provide no value — effectively dead code paths |
| `maxBFSDepth` (10) | Graph traversal bound — DoS protection, not a consensus parameter |

## Validation Added

- All BPS fields: `0 ≤ value ≤ 1,000,000`
- Fundamentality thresholds: monotonically increasing (`Notable < Foundational < Fundamental < Axiomatic`)
- Fundamentality bonuses: monotonically increasing
- Counter-fact penalties: ordered (`Low < Medium < High ≤ MaxPenalty`)
- Counter-fact confidence thresholds: ordered (`Medium < High`)
- Map values: bounded within BPS range
- Category decay profiles: `MaxConfidence > MinConfidence`
- `MinClaimStakeUlgm > 0`

## Helper Methods Added

| Method | Purpose |
|---|---|
| `GetCategoryDecayProfile(category)` | Returns profile from params map, falls back to hardcoded `CATEGORY_DECAY` |
| `GetStratumConfidenceCeiling(stratum)` | Returns ceiling from params map, falls back to hardcoded `STRATUM_CONFIDENCE_CEILING` |
| `GetEvidenceTypeStrength(evidenceType)` | Returns strength from params map, falls back to hardcoded `EVIDENCE_TYPE_STRENGTH` |
| `GetChallengeWindowMultiplier(category)` | Returns multiplier from params map, falls back to hardcoded `CHALLENGE_WINDOW_PROFILE` |
| `DefaultCategoryDecayProfiles()` | Returns default map matching hardcoded `CATEGORY_DECAY` |
| `DefaultStratumConfidenceCeilings()` | Returns default map matching hardcoded `STRATUM_CONFIDENCE_CEILING` |
| `DefaultEvidenceTypeStrengths()` | Returns default map matching hardcoded `EVIDENCE_TYPE_STRENGTH` |
| `DefaultChallengeWindowMultipliers()` | Returns default map matching hardcoded `CHALLENGE_WINDOW_PROFILE` |
