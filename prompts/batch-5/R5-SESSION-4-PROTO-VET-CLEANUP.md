# R5 Session 4: Proto Mutex Copy Cleanup + go vet Clean

## Context
R4-5 documented ~150+ `go vet` "copies lock value" warnings. These are caused by protobuf-generated types containing `sync.Mutex` via `protoimpl.MessageState` being passed by value instead of by pointer. While not runtime bugs, they:

1. Make `go vet` unusable for catching real issues (signal lost in noise)
2. Are technically undefined behavior per Go spec
3. Would fail CI with `go vet` gate

For testnet readiness, `go vet ./...` must be clean (or only have documented, accepted warnings).

## Tasks

### 1. Categorize the Warnings
```bash
go vet ./... 2>&1 | grep "copies lock" | sed 's/:.*//' | sort -u | head -30
```

Group by:
- **A. Function signatures** — `func Foo(claim Claim)` should be `func Foo(claim *Claim)`
- **B. Range copies** — `for _, v := range claims` should be `for i := range claims { v := &claims[i] }`
- **C. Struct literal copies** — assigning proto types by value
- **D. Test-only** — warnings only in `_test.go` files

### 2. Fix Category A (Function Signatures)
These are the highest impact. Affected areas from R4-5:
- `app/rest_api.go` (~60 warnings)
- `app/alignment_adapters.go` (~10 warnings)
- `app/vote_extensions.go` (~10 warnings)
- `app/staking_adapter.go` (~5 warnings)

Change function params from value to pointer:
```go
// Before
func (a *Adapter) SetClaim(ctx sdk.Context, claim types.Claim) error
// After
func (a *Adapter) SetClaim(ctx sdk.Context, claim *types.Claim) error
```

**Critical**: Update all callers and interface definitions when changing signatures. Use the compiler to find them — if you change a signature, anything that doesn't compile is a caller that needs updating.

### 3. Fix Category B (Range Copies)
```go
// Before
for _, claim := range claims {
    doSomething(claim)
}
// After
for i := range claims {
    claim := &claims[i]
    doSomething(claim)
}
```

### 4. Fix Category C (Struct Literals)
Where proto types are constructed as values and passed around:
```go
// Before
claim := types.Claim{...}
k.SetClaim(ctx, claim)
// After
claim := &types.Claim{...}
k.SetClaim(ctx, claim)
```

### 5. Fix Category D (Test Files)
- `x/knowledge/keeper/*_test.go` (~50 warnings)
- Same patterns as above but in test code

### 6. Verify
```bash
go vet ./... 2>&1 | grep -c "copies lock"
# Target: 0
go build ./...
go test ./...
```

## Constraints
- Do NOT change proto-generated files (*.pb.go)
- Do NOT change function signatures in interfaces that are defined by Cosmos SDK
- Focus on our own code that passes proto types by value

## Report
Write to `/reports/batch-5/SESSION-R5-4-VET-CLEANUP.md` with:
- Warning count before/after
- Category breakdown (A/B/C/D counts)
- Files modified
- Any warnings intentionally left (with justification)
