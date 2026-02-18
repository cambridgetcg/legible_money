# KNOWLEDGE-KEEPER: 11 Missing Query Server Methods Implemented

**Date**: 2026-02-18
**Modules**: x/knowledge (keeper, types), x/ontology (keeper)
**Status**: All 11 methods implemented, builds clean, tests pass

---

## Summary

The knowledge module's CLI had 20 query commands (created by KNOWLEDGE-CLI task). 13 had working keeper implementations; 11 returned gRPC "Unimplemented" because the `QueryServer` methods were missing. All 11 are now implemented.

## Architecture

### Cross-Module Data Flow

The 11 missing queries are ontology-related (strata, domain proposals, dependency validation). The data lives in the ontology module, but queries are served through the knowledge module's gRPC server.

**Approach**: Expanded the `OntologyKeeper` interface in `x/knowledge/types/expected_keepers.go` with 6 query-support methods that return knowledge proto types directly. The ontology keeper converts its own types (StratumProperties, Domain, DomainProposal) to knowledge types (StratumDefinition, OntologicalDomain, DomainProposal) internally.

**Dependency graph** (no cycles):
```
x/knowledge/types ← standalone (defines OntologyKeeper interface + proto types)
x/knowledge/keeper → x/knowledge/types
x/ontology/keeper → x/ontology/types + x/knowledge/types (NEW import)
```

### Nil Safety

All query methods check `q.ontologyKeeper != nil` before calling ontology methods. When nil (e.g., in tests or before init), methods return empty responses or appropriate errors.

---

## Files Changed

### x/knowledge/types/expected_keepers.go (MODIFIED)
Added 6 methods to `OntologyKeeper` interface:
- `GetStratumDefinition(ctx, level uint32) (*StratumDefinition, bool)`
- `GetAllStratumDefinitions(ctx) []*StratumDefinition`
- `GetOntologyDomains(ctx) []*OntologicalDomain`
- `GetOntologyDomainsByStratum(ctx, level uint32) []*OntologicalDomain`
- `GetDomainProposalById(ctx, id string) (*DomainProposal, bool)`
- `GetAllDomainProposals(ctx) []*DomainProposal`

### x/knowledge/types/errors.go (MODIFIED)
Added 2 error types:
- `ErrStratumNotFound` (code 65)
- `ErrProposalNotFound` (code 66)

### x/ontology/keeper/query_support.go (NEW — 147 lines)
Implements the 6 new OntologyKeeper interface methods, converting:
- `StratumProperties` → `StratumDefinition` (level→id mapping, uint64→uint32 confidence ceiling)
- `Domain` → `OntologicalDomain` (ProposedBy→Proposer, CreatedAt→CreatedAtHeight)
- ontology `DomainProposal` → knowledge `DomainProposal` (status string→enum, VotesFor/Against→EndorsementStake/ChallengeStake)

### x/knowledge/keeper/grpc_query.go (MODIFIED — 11 methods + 5 helpers added)
All 11 query methods + helper functions added after existing code.

---

## Per-Method Implementation Details

### 1. Domains (fully implemented)
- **Approach**: If stratum filter set + ontology keeper available → `GetOntologyDomainsByStratum`; else iterate local knowledge domains
- **Filters**: stratum (parsed via `parseStratumID`), parent_domain (`isChildOf`), pattern (`matchPattern` — glob-like with `*`)
- **Pagination**: offset/limit, default limit=100, returns total count

### 2. DomainsByStratum (fully implemented)
- **Approach**: Parse stratum_id (numeric or name) → `GetOntologyDomainsByStratum`
- **Store access**: Ontology keeper's stratum→domain index

### 3. DomainChildren (fully implemented)
- **Approach**: Dot-separated naming convention ("physics.mechanics" is child of "physics")
- **include_descendants=false**: Only direct children (no dots in remaining suffix)
- **include_descendants=true**: All descendants with prefix match
- **Fallback**: Uses local knowledge domains if ontology keeper unavailable

### 4. DomainPath (fully implemented)
- **Approach**: Split domain_id on ".", build path from root components
- For "physics.mechanics.quantum" → ["physics", "physics.mechanics", "physics.mechanics.quantum"]
- Looks up each segment in knowledge store; creates placeholder with status="unregistered" if not found

### 5. Stratum (fully implemented)
- **Approach**: Parse stratum_id → `GetStratumDefinition`
- Returns `StratumDefinition` with id, name, level, confidence_ceiling, description

### 6. Strata (fully implemented)
- **Approach**: `GetAllStratumDefinitions` + optional min_level/max_level filter
- Default max_level=6 (Testimonial) when only min_level specified

### 7. StratumRegistry (fully implemented)
- **Approach**: `GetAllStratumDefinitions` → build `StratumRegistry` with schema_version=1 and last_modified=current block
- Empty registry returned if ontology keeper unavailable

### 8. ValidateDependency (fully implemented)
- **Approach**: Parse both stratum IDs, verify both exist, validate dependency direction
- **Rules**: Self-dependency invalid; lower level cannot depend on higher level
- Returns `{valid, reason}` — never errors, always returns a response

### 9. CalculateConfidence (fully implemented with dependency discount)
- **Approach**: Start with base_confidence, apply stratum ceiling from `GetConfidenceCeiling`, then apply dependency discounts
- **Discount formula**: Per-dependency discount = `1 - (1-strength/1M)/10` — weak dependencies reduce confidence by up to 10% each
- **Fields populated**: base_confidence, stratum_ceiling, inherited_ceiling (default 1M), stratum_discount, effective_confidence, dependencies
- Gracefully handles missing ontology keeper (uses defaults)

### 10. DomainProposal (fully implemented)
- **Approach**: `GetDomainProposalById` → convert to knowledge DomainProposal
- Endorsements/challenges arrays returned empty (individual records not tracked in ontology module — noted for future work)

### 11. DomainProposals (fully implemented)
- **Approach**: `GetAllDomainProposals` + status filter + pagination
- Status filter uses knowledge DomainProposalStatus enum (UNSPECIFIED means no filter)
- Pagination with offset/limit, default limit=100

---

## Helper Functions

| Function | Purpose |
|----------|---------|
| `parseStratumID(id)` | Parse "empirical" or "4" → uint32 level |
| `matchPattern(name, pattern)` | Glob-like: "physics.*" matches "physics.mechanics" |
| `isChildOf(name, parent)` | "a.b" is child of "a" |
| `isDirectChild(name, parent)` | "a.b" is direct child; "a.b.c" is not |

---

## Type Conversion Summary

| Ontology Type | Knowledge Type | Key Mappings |
|---------------|---------------|--------------|
| `StratumProperties` | `StratumDefinition` | level→Level, Name→Name, MaxConfidence(uint64)→ConfidenceCeiling(uint32) |
| `Domain` | `OntologicalDomain` | Name, Description, ProposedBy→Proposer, Status, CreatedAt→CreatedAtHeight |
| `DomainProposal` | `DomainProposal` | Id, Domain→ProposedDomain, status string→DomainProposalStatus enum |

Status mapping: "active"→PENDING, "passed"→APPROVED, "rejected"→REJECTED, "expired"→EXPIRED

---

## Build + Test Output

```
$ go build ./x/knowledge/...   # clean
$ go build ./x/ontology/...    # clean
$ go build ./...               # clean (full build)

$ go test ./x/knowledge/... -count=1
ok  github.com/legible-money/legible/x/knowledge/crypto        1.102s
ok  github.com/legible-money/legible/x/knowledge/evaluation     0.873s
ok  github.com/legible-money/legible/x/knowledge/keeper         0.865s

$ go test ./x/ontology/... -count=1
ok  github.com/legible-money/legible/x/ontology/keeper          0.461s

$ go vet ./x/knowledge/... ./x/ontology/...
# Pre-existing lock-copy warnings only (proto MessageState passed by value)
# No new warnings from this change
```

---

## Limitations & Future Work

1. **DomainProposal endorsements/challenges**: The ontology module tracks aggregate VotesFor/VotesAgainst but not individual endorsement/challenge records. The `DomainProposalResponse.endorsements` and `challenges` fields are returned as empty slices.

2. **CalculateConfidence dependency discount**: The current formula is a reasonable approximation. A more sophisticated version would:
   - Look up each dependency's stratum ceiling
   - Apply the ontology module's cross-stratum link discount factors
   - Account for transitive dependencies

3. **DomainPath path segments**: Unregistered intermediate path segments are returned with status="unregistered". The ontology module doesn't enforce hierarchical registration.

4. **Pattern matching**: Only supports trailing wildcards (`physics.*`, `phys*`). Full glob or regex would require additional implementation.
