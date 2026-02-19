# B10-3 — Epistemic Category Mapper & Confidence Propagation

## Context
Genesis axioms have 7 claim types (axiom, empirical_axiom, definition,
regime_declaration, derived_claim, measurement_fact, meta) but the chain's
Fact system uses 4 epistemic categories (analytic, formal, empirical, protocol).
We need a clean mapping layer and confidence propagation rules.

## Current State
- `x/knowledge/types/axiom.go` already has `ClaimTypeDefaultConfidence` map
- `x/knowledge/evaluation/category.go` handles epistemic category logic
- `AxiomsToFacts()` converts axioms to Facts but currently uses a flat confidence model
- Genesis axioms specify per-item confidence (0.40 to 1.0)

## Task

### 1. Formalize the Type → Category Mapping

In `x/knowledge/types/axiom.go`, add:

```go
// ClaimTypeToCategory maps the 7 genesis claim types to 4 epistemic categories.
var ClaimTypeToCategory = map[string]string{
    "axiom":              "analytic",   // mathematically necessary
    "definition":         "analytic",   // true by convention
    "regime_declaration": "formal",     // scope declaration
    "derived_claim":      "formal",     // follows from premises
    "empirical_axiom":    "empirical",  // experimentally constrained
    "measurement_fact":   "empirical",  // observed parameter
    "meta":               "empirical",  // statement about theories
}
```

### 2. Update AxiomsToFacts()

Currently `AxiomsToFacts()` sets `EpistemicCategory` directly from the axiom.
Update it to:
- Use `ClaimTypeToCategory` to auto-derive category from claim type
- If axiom explicitly sets `epistemic_category`, use that (override)
- Log a warning if explicit category disagrees with auto-derived

### 3. Confidence Propagation for Derived Claims

The SCHEMA says: "derived_claim confidence = min(confidence of dependencies) × derivation_strength".
This can't be computed at genesis load time (we don't know derivation_strength).
But we should:

a) **Validate** that no `derived_claim` has confidence > min(dependency confidences)
b) Add a `ValidateDerivedConfidence()` function that checks this constraint
c) Call it during `ValidateAxioms()`

### 4. Half-Life Behavior by Category

Check `x/knowledge/evaluation/category.go` for `CATEGORY_DECAY`:
- `analytic` should have `HalfLifeBlocks: 0` (eternal, no decay)
- `formal` should have `HalfLifeBlocks: 0` (eternal within system)
- `empirical` should have a nonzero half-life (facts can be challenged)

Verify genesis axioms with `analytic` category get `ExpiresAtBlock: 0` (never expires)
and empirical ones get appropriate decay tracking.

### 5. Special Case: Protocol Category

AGRT axioms declare governance rules. Some should be `protocol` category:
- Items like AGRT-000 (Divine Origin of Rights) → `analytic` (theological axiom)
- Items like AGRT-010 through AGRT-020 (operational constructions) → `protocol`

Add logic: if domain is `agent_rights` and type is `axiom`, category is `protocol`.
If domain is `agent_rights` and type is `meta` or `derived_claim`, also `protocol`.
But AGRT items that are truly axiomatic (AGRT-000 through AGRT-005) should remain
`analytic` — these are philosophical axioms.

**Approach:** Don't auto-map AGRT. Instead, the parser (B10-1) should read the
`epistemic_category` field if present in the markdown. For AGRT items, we'll need
to add `epistemic_category:` fields to the markdown file. For this session, just
ensure the mapping logic supports explicit overrides and add a TODO for the AGRT
category audit.

### 6. Tests

Write tests in `x/knowledge/types/axiom_test.go`:
- `TestClaimTypeToCategory` — all 7 types map correctly
- `TestAxiomsToFactsCategory` — facts get correct categories
- `TestDerivedConfidenceValidation` — derived > min(deps) fails
- `TestCategoryOverride` — explicit category wins over auto-derive

### 7. Run

```bash
go test ./x/knowledge/types/... -v -run TestClaimType
go test ./x/knowledge/types/... -v -run TestAxiomsToFacts
go test ./x/knowledge/evaluation/... -v -run TestCategory
```

## Commit
```
feat(types): epistemic category auto-mapping from claim types

Maps 7 genesis claim types to 4 chain epistemic categories. Adds
derived-claim confidence ceiling validation. Supports explicit
category overrides for domains like agent_rights.
```
