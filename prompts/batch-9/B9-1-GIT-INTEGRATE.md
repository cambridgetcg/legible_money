# B9-1: Git Integration — Commit All Batch-8 Work

## Context
Batch 8 ran 7 parallel sessions. Due to concurrency, B8-4 had to revert some WIP changes. There are now ~70+ uncommitted files in the working tree spanning:
- E2E test fixes (helpers, race fixes)
- gRPC gateway stubs (26 `query.pb.gw.go` files + module wiring)
- Knowledge module indexes (`indices.go`, `keys.go`, state changes)
- Genesis axiom infrastructure (`axiom.go`, `axiom_test.go`, genesis changes, loader tool)
- Config changes

## Task

### 1. Assess the working tree
```bash
git status
git diff --stat
```

### 2. Check if everything compiles
```bash
go build ./...
go vet ./...
```

If it doesn't, fix compilation errors. The most likely issues:
- `x/knowledge/keeper/state.go` has `k.maintainFactIndexes undefined` — need to check if `indices.go` provides it
- gRPC gateway files may need a dependency added to `go.mod`

### 3. Run tests
```bash
go test ./... -count=1 -timeout 600s
```
Fix any regressions.

### 4. Organize into clean commits
Group changes logically:
1. `fix(e2e): add test helpers and fix nonce race conditions`
2. `feat(api): enable gRPC gateway REST endpoints for all modules`
3. `feat(knowledge): add secondary indexes for BeginBlocker performance`
4. `feat(knowledge): genesis axiom infrastructure (types, loader, template)`
5. `chore(reports): batch-8 session reports`

### 5. Verify clean state
```bash
go build ./...
go test ./... -count=1 -timeout 600s
git status  # should be clean
```

### 6. Update tag if needed
If the tag `v0.1.0-alpha` is on an older commit, consider whether to move it or create `v0.1.0-alpha.2`.

## File Ownership
All uncommitted files in working tree.

## Report
Write `reports/batch-9/SESSION-B9-1-GIT-INTEGRATE.md`.
