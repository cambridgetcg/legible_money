# R4 Session 7: Full Build Green + Commit All R3+R4 Work

## Context
R3 sessions ran in parallel and may have introduced conflicts. R3-5's report explicitly noted `go build ./...` was broken. All R3+R4 work is uncommitted. This session's job: make everything build and pass, then commit.

## Tasks

### 1. Full Build
```bash
cd /Users/yournameisai/Desktop/legible-money
go build ./...
```
If it fails, fix ALL errors. Common expected issues:
- x/knowledge ExtendedParams signature changes breaking external callers
- Proto-generated type conflicts from R3-3's mass migration
- Import cycles from new cross-module interfaces

### 2. Full Test Suite
```bash
go test ./... 2>&1 | tee /tmp/test-results.txt
```
Track: total pass, total fail, total skip. Fix any failures introduced by R3/R4 sessions.

### 3. Go Vet
```bash
go vet ./... 2>&1 | grep -v "copies lock" | head -20
```
Fix any new warnings (ignore pre-existing proto lock-copy warnings).

### 4. Dependency Check
```bash
go mod tidy
go mod verify
```

### 5. Diff Summary
```bash
git diff --stat HEAD
git diff HEAD --name-only | wc -l
```
Document total files changed.

### 6. Commit Strategy
Create a single comprehensive commit with all R3+R4 changes:

```bash
git add -A
git commit -m "feat: R3+R4 batch — parameterization, proto tx, tokens phase 2, fixes

Round 3:
- P0 fixes: home query CLI, schedule cleanup param (R3-1)
- compute_pool verified fully wired (R3-2)
- Proto tx.proto for 12 modules, 72 new gRPC RPCs (R3-3)
- x/knowledge: 60+ constants → ExtendedParams (R3-4)
- x/staking: tier configs + slash/reputation → Params (R3-5)
- x/tokens: DelegatePower, WrapToken, UnwrapToken implemented (R3-6)

Round 4:
- Gov quorum enforcement (R4-1)
- Gas coverage for 9 modules (R4-2)
- Tokens delegation hardening + tests (R4-3)
- Knowledge integration fixes (R4-4)
- Proto codec integrity verification (R4-5)
- Staking migration safety tests (R4-6)

All tests pass. Build clean."
```

### 7. Updated Progress Report
Update `/reports/PROGRESS-REPORT.md`:
- Refresh all statistics (handler count, gas coverage, test count, etc.)
- Move fixed issues from OPEN to FIXED
- Add any new issues discovered in R4
- Update coverage percentages

## Verification
```bash
go build ./...
go test ./...
go vet ./...
go mod verify
```
**ALL must pass with zero failures before committing.**

## Report
Write to `/reports/batch-4/SESSION-R4-7-FINAL-GREEN.md` with:
- Build status
- Test results (total pass/fail/skip per package)
- Files changed count
- Commit hash
- Updated issue tracker (open/fixed/deferred counts)
- Any remaining known issues
