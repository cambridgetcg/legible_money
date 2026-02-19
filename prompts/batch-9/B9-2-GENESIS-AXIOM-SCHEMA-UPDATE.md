# B9-2: Update Genesis Axiom Schema for Option C Layered System

## Context
The genesis axiom system was updated AFTER B8-3 ran. The new schema (in `genesis-axioms/SCHEMA.md`) defines 7 claim types:
- `axiom`, `empirical_axiom`, `definition`, `regime_declaration`, `derived_claim`, `measurement_fact`, `meta`

Plus additional fields: `confidence`, `regime`, `approximation_tags`, `derivation_ref`, `evidence_ref`.

B8-3's `GenesisAxiom` struct and template only support the original simpler schema.

## Task

### 1. Read the updated schema
Read `genesis-axioms/SCHEMA.md` for the full type system.

### 2. Update `x/knowledge/types/axiom.go`

The `GenesisAxiom` struct needs:
```go
type GenesisAxiom struct {
    AxiomID           string   `json:"axiom_id"`
    Statement         string   `json:"statement"`
    FormalExpression   string   `json:"formal_expression,omitempty"`
    ClaimType         string   `json:"type"`              // NEW: axiom|empirical_axiom|definition|...
    Domain            string   `json:"domain"`
    EpistemicCategory string   `json:"epistemic_category"`
    Regime            string   `json:"regime,omitempty"`   // NEW: classical|SR|GR|QM|...
    Confidence        float64  `json:"confidence"`         // NEW: 0.0-1.0
    ApproxTags        []string `json:"approximation_tags,omitempty"` // NEW
    DerivationRef     string   `json:"derivation_ref,omitempty"`     // NEW
    EvidenceRef       string   `json:"evidence_ref,omitempty"`       // NEW
    Dependencies      []string `json:"dependencies"`
    SourceTradition   string   `json:"source_tradition,omitempty"`   // NEW
    References        []string `json:"references,omitempty"`
    Notes             string   `json:"notes,omitempty"`              // NEW
    TierRequired      int      `json:"tier_required,omitempty"`      // NEW: for agent rights
}
```

### 3. Update ValidAxiomCategories
Currently only allows `analytic` and `formal`. Needs to also allow `empirical` and `protocol` for physics axioms.

### 4. Update the loader tool
`tools/axiom-loader/main.go` needs to handle the new fields.

### 5. Update genesis template
`scripts/genesis-axioms-template.json` — add examples with the new fields.

### 6. Update genesis.go
Ensure `InitGenesis` properly maps the new claim types to appropriate on-chain representations.

### 7. Update tests
`x/knowledge/types/axiom_test.go` — add tests for new types.

### 8. Verify
```bash
go build ./...
go test ./x/knowledge/... -count=1
```

## File Ownership
- `x/knowledge/types/axiom.go`
- `x/knowledge/types/axiom_test.go`
- `x/knowledge/types/genesis.go`
- `tools/axiom-loader/main.go`
- `scripts/genesis-axioms-template.json`

## Report
Write `reports/batch-9/SESSION-B9-2-AXIOM-SCHEMA.md`.
