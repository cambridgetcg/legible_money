# B8-6: Query Performance + Indexing Audit

## Context
With 777 genesis axioms plus real-time claims, the knowledge module needs efficient queries. Testnet validators will run queries constantly. Need to verify store key design and add secondary indices where missing.

## Task

### 1. Audit all KV store key designs
For each of the 28 modules in `x/`, read:
- `types/keys.go` — store key prefixes
- `keeper/` — how data is stored and retrieved

Check for:
- **Missing indices**: Can all advertised queries be answered without full-store iteration?
- **Prefix collisions**: Any store key prefixes that could overlap?
- **Unbounded iterations**: Any `Iterator` calls without proper bounds?

### 2. Focus on knowledge module
The knowledge module is the heaviest — claims, challenges, rounds, evidence, confidence scores.

Read:
- `x/knowledge/types/keys.go`
- `x/knowledge/keeper/claim.go` (or similar)
- `x/knowledge/keeper/grpc_query.go`

Verify these queries are indexed (not full-scan):
- Claims by domain
- Claims by epistemic category
- Claims by status
- Claims by submitter
- Active verification rounds
- Axioms (the 777 genesis claims)

### 3. Add missing indices
For any query that currently requires iteration:
- Add a secondary index key (e.g., `DomainClaimPrefix + domain + claimId`)
- Update Set/Delete to maintain the index
- Update query to use the index

### 4. Benchmark
If possible, write a simple benchmark:
- Insert 1000 claims across 15 domains
- Time: query all claims in "mathematics" domain
- Compare indexed vs non-indexed path

### 5. Verify
- All existing tests still pass
- New index queries return correct results
- No store key prefix collisions

## File Ownership
- `x/knowledge/types/keys.go`
- `x/knowledge/keeper/` (query and store files)
- Any other module with missing indices

## Report
Write `reports/batch-8/SESSION-B8-6-QUERY-INDEXING.md`.
