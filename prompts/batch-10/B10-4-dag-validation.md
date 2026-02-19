# B10-4 — DAG Validation Hardening & Cross-Domain Reference Integrity

## Context
The 777 genesis axioms form a directed acyclic graph (DAG) with ~2000+
cross-references both within and across 15 domains. The existing
`ValidateAxiomDAG()` in `x/knowledge/types/axiom.go` checks for cycles
using Kahn's algorithm but doesn't check cross-domain reference integrity.

## Current State
- `ValidateAxiomDAG()` exists and works for basic cycle detection
- `ValidateAxioms()` checks: unique IDs, format, domain exists, prefix matches, categories, DAG
- Dependencies reference axiom IDs that may be in OTHER domain files
- The parser (B10-1) will produce a single JSON with all 777 items
- All cross-domain deps have been manually verified to exist

## Task

### 1. Enhanced DAG Validation

Add to `x/knowledge/types/axiom.go`:

#### a) Topological Order Report
After Kahn's algorithm succeeds, compute:
- Number of root nodes (no dependencies) — these are true axioms
- Maximum depth of the DAG
- Items at each depth level (stratum 0, 1, 2, ...)
- Orphan detection: items that nothing depends on AND have no deps themselves

Add a `DAGStats` struct and `ComputeDAGStats()` function.

#### b) Cross-Domain Dependency Matrix
Build a matrix showing which domains depend on which:
```
From → To matrix:
MATH → PHYS: 23 refs
PHYS → MATH: 45 refs
COSM → PHYS: 18 refs
PSYCH → BIO: 8 refs
...
```
This helps visualize the knowledge graph structure.

#### c) Dependency Completeness Check
Verify that ALL items referenced in `depends:` arrays exist in the full 777 set.
Currently `ValidateAxiomDAG` checks this, but add a clearer error message showing
which file the missing dependency should be in based on prefix.

#### d) Stratum Consistency Check
If item A depends on item B, and B is a `derived_claim`, then A should not be
typed as `axiom` (axioms shouldn't depend on derived claims). Add this check.

### 2. CLI Reporting Tool

Extend `tools/axiom-loader/main.go` with a `stats` command:
```bash
go run tools/axiom-loader/main.go stats <axioms.json>
```
Output:
- Total items by type
- Total items by domain
- DAG depth
- Root nodes (true axioms with no deps)
- Most-referenced items (hub nodes)
- Cross-domain dependency matrix
- Any warnings (high-depth items, orphans)

### 3. Visualization Data

Output a `dag-edges.csv` for external visualization:
```csv
from,to,from_domain,to_domain
COSM-005,PHYS-078,cosmology,physics
COSM-005,PHYS-079,cosmology,physics
...
```
Add as a subcommand: `go run tools/axiom-loader/main.go edges <axioms.json> -o dag-edges.csv`

### 4. Tests

In `x/knowledge/types/axiom_test.go`:
- `TestDAGStatsComputation` — verify stats for a known small DAG
- `TestCrossDomainMatrix` — verify matrix counts
- `TestStratumConsistency` — axiom depending on derived_claim detected
- `TestDAGWithAllDomains` — integration test loading all 777 (requires
  axioms-777.json from B10-1; skip if file doesn't exist)

### 5. Run

```bash
go test ./x/knowledge/types/... -v -run TestDAG
go run tools/axiom-loader/main.go stats scripts/axioms-777.json
```

## Commit
```
feat(tools): DAG stats, cross-domain matrix, and stratum consistency checks

Adds dag stats computation, cross-domain dependency visualization,
and stratum consistency validation. axiom-loader gains stats and
edges subcommands for knowledge graph analysis.
```
