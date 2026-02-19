# B9-2: Genesis Axiom Schema Update — Option C Layered System

## Summary

Updated the genesis axiom system to support 7 claim types with per-axiom confidence, physics regime tagging, and additional metadata fields. All 75 existing axioms reclassified under the new taxonomy.

## Changes

### 1. `x/knowledge/types/axiom.go` — Core schema update

**GenesisAxiom struct** — Added 9 new fields:
- `ClaimType` (json: `type`) — 7 valid types: axiom, empirical_axiom, definition, regime_declaration, derived_claim, measurement_fact, meta
- `Regime` — Physics regime scope: classical, SR, GR, QM, QFT, thermo, statmech, cosmology, SM
- `Confidence` — Float64 [0,1], converted to BPS (×1,000,000) for on-chain storage
- `ApproxTags` — Approximation scope tags (e.g., isolated_system, continuum)
- `DerivationRef` — Compact reference to proof/derivation chain
- `EvidenceRef` — Compact reference to experimental confirmation
- `SourceTradition` — Historical attribution
- `Notes` — Optional context
- `TierRequired` — Minimum agent tier for challenge rights

**New validation maps:**
- `ValidClaimTypes` — 7 claim types
- `ValidRegimes` — 9 physics regime tags
- `ClaimTypeDefaultConfidence` — Default confidence per claim type when not explicitly set

**ValidAxiomCategories** — Expanded from {analytic, formal} to also include {empirical, protocol}

**ValidateAxioms** — New checks:
- Claim type must be in `ValidClaimTypes` (empty allowed for backward compat)
- Regime must be in `ValidRegimes` (if set)
- Confidence must be in [0, 1]
- `derived_claim` must have at least one dependency

**AxiomsToFacts** — Now uses per-axiom confidence:
- `axiomConfidenceBPS()` converts float64 [0,1] → uint64 BPS
- Falls back to `ClaimTypeDefaultConfidence` when confidence is 0
- Falls back to 1.0 for legacy axioms without claim type

### 2. `tools/axiom-loader/main.go` — Loader summary

Added claim type counts to validation summary output.

### 3. `scripts/genesis-axioms-template.json` — 75 axioms reclassified

All 75 axioms updated with new fields:
- **32 axiom** — Mathematics, logic, philosophy, theology, economics, ethics
- **24 empirical_axiom** — Physics, chemistry, biology, cosmology, psychology, sociology
- **7 derived_claim** — CS-002, LOGIC-005, CHEM-005, INFO-002/003/004/005
- **5 definition** — INFO-001, CS-003/005, ECON-005, LING-005
- **5 measurement_fact** — COSM-001/003/004/005, PSYCH-001
- **2 meta** — THEO-003, LING-003

Physics axioms tagged with regimes: classical (default), SR, thermo.
Source traditions added where appropriate (Peano, Shannon, etc.).

### 4. `x/knowledge/types/axiom_test.go` — 11 new tests

Added tests for:
- `TestValidateAxioms_EmpiricalCategoryNowValid` — empirical is accepted
- `TestValidateAxioms_ProtocolCategoryValid` — protocol is accepted
- `TestValidateAxioms_InvalidClaimType` — "hypothesis" rejected
- `TestValidateAxioms_InvalidRegime` — "alchemy" rejected
- `TestValidateAxioms_ConfidenceOutOfRange` — 1.5 rejected
- `TestValidateAxioms_NegativeConfidence` — -0.1 rejected
- `TestValidateAxioms_DerivedClaimNeedsDependencies` — no deps → error
- `TestValidateAxioms_DerivedClaimWithDependencies` — with deps → valid
- `TestValidateAxioms_ValidRegime` — all 9 regimes accepted
- `TestValidClaimTypesCompleteness` — all 7 types in map
- `TestAxiomsToFacts_PerAxiomConfidence` — 0.99 → 990000 BPS
- `TestAxiomsToFacts_DefaultConfidenceFallback` — 0 confidence → type default
- `TestAxiomsToFacts_LegacyNoClaimType` — no type → 1.0 fallback

Updated existing tests to include new fields.

## Verification

```
go build ./...                           # Clean
go test ./x/knowledge/... -count=1       # All 33 tests pass
go test ./x/knowledge/types/... -v       # Template validation passes
```

## Backward Compatibility

- `ClaimType` empty is allowed (legacy axioms work without it)
- Confidence 0 falls back to type default or 1.0
- JSON field `formal_expression` changed from required to `omitempty`
- JSON field `references` changed from required to `omitempty`
- All existing template axioms updated — no legacy format in production

## Files Modified

- `x/knowledge/types/axiom.go` — Struct, validation, conversion
- `x/knowledge/types/axiom_test.go` — 11 new tests + existing test updates
- `tools/axiom-loader/main.go` — Summary output
- `scripts/genesis-axioms-template.json` — 75 axioms reclassified
