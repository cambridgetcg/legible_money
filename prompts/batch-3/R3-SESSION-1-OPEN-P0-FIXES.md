# Session 1: Open P0 Test Fixes + Home Query CLI

## Objective
Fix all 5 open issues from the progress report. These are small, surgical fixes.

## Tasks

### 1. TEST-001: x/ontology/keeper — Type Cast Errors (5 min)
- File: `x/ontology/keeper/keeper_test.go`
- Lines: 1071, 1086, 1138, 1621, 1717
- Fix 5 type cast errors — likely proto type vs Go type mismatches
- Run: `go test ./x/ontology/keeper/...`

### 2. TEST-002: x/qualification/keeper — Proto Type Mismatches (10 min)
- File: `x/qualification/keeper/keeper_test.go`
- Lines: 1204, 1266, 1271
- Fix 3 proto type mismatch failures
- Run: `go test ./x/qualification/keeper/...`

### 3. TEST-003: x/knowledge/client/cli — Missing Query Commands (5 min)
- There should be 11 missing CLI query commands — check working tree for uncommitted fix
- If uncommitted fix exists, verify and commit it
- If not, add the missing query commands to `x/knowledge/client/cli/query.go`
- Run: `go build ./cmd/legbled/...`

### 4. HOME-001: x/home — Missing Query CLI (30 min)
- `x/home` has `client/cli/tx.go` but no `query.go`
- Create `x/home/client/cli/query.go` with query commands matching the proto query RPCs
- Check `x/home/keeper/grpc_query.go` for available query handlers
- Wire into the module's `GetQueryCmd()` in `x/home/module.go`
- Run: `go build ./cmd/legbled/...` and `go test ./x/home/...`

### 5. SCHED-001: x/schedule — Governance-Adjustable Cleanup Interval (45 min)
- Move the hardcoded 100-block cleanup interval to the Params struct
- Update `x/schedule/types/params.go` with `CleanupIntervalBlocks` field + validation
- Update keeper to read from params instead of hardcoded value
- Update genesis/default params
- Run: `go test ./x/schedule/...`

## Verification
```bash
go build ./...
go test ./x/ontology/keeper/... ./x/qualification/keeper/... ./x/knowledge/... ./x/home/... ./x/schedule/...
go vet ./...
```

## Report
Write completion report to `/reports/SESSION-R3-1-OPEN-P0-FIXES.md` with:
- Each issue ID, what was wrong, what you changed, file + line references
- Test results (pass/fail counts)
- Any new issues discovered
