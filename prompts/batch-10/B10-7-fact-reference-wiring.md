# B10-7 — Fact Graph Reference Wiring: Axiom Dependencies → FactReference Edges

## Context
Genesis axioms have `dependencies` (the DAG edges). When converted to Facts by
`AxiomsToFacts()`, these become `fact.References` — the links in the knowledge
graph. But the chain also has a separate `FactReference` object type that tracks
richer metadata about relationships between facts. We need to wire the axiom
DAG into both systems.

## Current State

### Fact.References
`AxiomsToFacts()` already sets `fact.References = a.Dependencies`. This gives
each fact a list of IDs it depends on. ✓

### FactReference Objects
The genesis state (`GenesisState`) has a `FactReferences` field:
```go
type GenesisState struct {
    Params         *Params
    Domains        []*OntologicalDomain
    Facts          []*Fact
    Claims         []*Claim
    FactReferences []*FactReference
}
```

`FactReference` tracks:
- Source fact ID (the referencer)
- Target fact ID (the referenced)
- Reference type (supports, contradicts, extends, etc.)
- Strength/weight

The fact graph evaluation (`evaluation/fact_graph.go`) uses these to compute
support bonuses and discredit penalties during claim evaluation.

### The Gap
`AxiomsToFacts()` sets `fact.References` but does NOT create `FactReference`
objects. So the genesis state has 777 facts but 0 `FactReference` entries.
The fact graph evaluation will see these facts as having no structural
relationships.

## Task

### 1. Create FactReference Objects from Axiom Dependencies

In `x/knowledge/types/axiom.go`, add:

```go
// AxiomsToFactReferences creates FactReference edges from axiom dependencies.
// Each dependency edge (A depends on B) becomes a "supports" reference:
// fact A references fact B as supporting evidence.
func AxiomsToFactReferences(axioms []*GenesisAxiom) []*FactReference {
    var refs []*FactReference
    for _, a := range axioms {
        for _, dep := range a.Dependencies {
            ref := &FactReference{
                SourceFactId: a.AxiomID,
                TargetFactId: dep,
                ReferenceType: "supports",     // dependency = supported by
                Strength:      1_000_000,       // full strength for genesis
                CreatedAtHeight: 0,             // genesis block
            }
            refs = append(refs, ref)
        }
    }
    return refs
}
```

### 2. Update axiom-loader inject Command

In `tools/axiom-loader/main.go`, the `cmdInject` function currently only injects
facts. Update it to also inject FactReferences:

```go
// After creating facts...
factRefs := types.AxiomsToFactReferences(axioms)

// Get existing fact references
var existingRefs []*types.FactReference
if refsRaw, ok := knowledgeState["fact_references"]; ok && string(refsRaw) != "null" {
    json.Unmarshal(refsRaw, &existingRefs)
}

allRefs := append(existingRefs, factRefs...)
refsJSON, _ := json.Marshal(allRefs)
knowledgeState["fact_references"] = refsJSON

fmt.Printf("Injected %d fact references into %s\n", len(factRefs), genesisPath)
```

### 3. Verify Reference Type Schema

Check `x/knowledge/types/` for the FactReference proto/type definition.
Ensure the `ReferenceType` field accepts "supports". Common reference types:
- `"supports"` — evidence for
- `"contradicts"` — evidence against
- `"extends"` — builds upon
- `"supersedes"` — replaces
- `"defines"` — definitional dependency

For genesis axiom dependencies, use `"supports"` as the default. But consider:
- If A is a `definition` and B depends on A, B→A could be `"defines"` instead
- If A is a `derived_claim` and depends on B, A→B is `"supports"` (B supports A)

**Direction matters:** In the axiom DAG, `depends: [B]` on item A means "A depends
on B" = "B supports A". So the FactReference should be:
- `SourceFactId: A` (the dependent)
- `TargetFactId: B` (the dependency)
- `ReferenceType: "supports"` (B supports A; the reference is "A references B
  as supporting evidence")

Verify this matches how `evaluateFactGraph()` processes references.

### 4. Reference Count Stats

Add reporting to the inject command:
```
Injected 777 axiom facts into genesis.json
Injected 2134 fact references into genesis.json
  Total references: 2134
  Reference types: supports=2134
  Cross-domain references: 847
  Intra-domain references: 1287
```

### 5. Verify fact_graph Evaluation Handles Genesis Facts

Check `x/knowledge/evaluation/fact_graph.go`:
- When a new claim references a genesis axiom (status=verified), it should get
  the `supportBonus` (30,000 per reference, max 150,000)
- Verify that the evaluation engine correctly looks up genesis facts via
  `evalCtx.GetFact(ctx, refId)`

### 6. Tests

In `x/knowledge/types/axiom_test.go`:
- `TestAxiomsToFactReferences` — verify reference count and direction
- `TestFactReferenceDirection` — A depends on B → source=A, target=B
- `TestFactReferenceTypes` — all genesis refs are "supports"

In `x/knowledge/evaluation/fact_graph_test.go`:
- Test that a claim referencing a genesis axiom (verified, confidence 1M) gets
  the support bonus

### 7. Run

```bash
go test ./x/knowledge/types/... -v -run TestAxiomsToFact
go test ./x/knowledge/evaluation/... -v -run TestFactGraph
```

## Commit
```
feat(knowledge): wire axiom dependency DAG into FactReference graph

Creates FactReference edges from genesis axiom dependencies during
injection. Each dependency becomes a "supports" reference at full
strength. axiom-loader inject now outputs both facts and references.
Enables fact graph evaluation to leverage the 777-item knowledge DAG.
```
