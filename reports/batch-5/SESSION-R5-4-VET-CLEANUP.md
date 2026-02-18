# R5-SESSION-4: Proto Vet Cleanup — Lock Copy Warnings

**Date**: 2026-02-18
**Scope**: Fix all `go vet` "copies lock value" / "passes lock by value" warnings
**Result**: 2792 warnings → 0 warnings (100% elimination)

---

## Summary

Proto-generated Go types contain `protoimpl.MessageState` which embeds `sync.Mutex`. Our hand-written code was passing these types by value, which copies the mutex — undefined behavior per the Go spec. Fixed all instances across the entire codebase.

## Warning Breakdown by Category

| Category | Count | Fix Pattern |
|----------|-------|-------------|
| A: Function signatures (params/returns) | ~1400 | Value → pointer params/returns |
| B: Range variable copies | ~350 | `for _, v := range` → `for i := range { v := &items[i] }` |
| C: Struct literal copies | ~600 | `Type{...}` → `&Type{...}` |
| D: Test-only copies | ~440 | Same patterns in `_test.go` files |

## Modules Fixed (26 modules + app/)

| Module | Warnings | Status |
|--------|----------|--------|
| x/knowledge/keeper | 802 | FIXED |
| x/knowledge/evaluation | 144 | FIXED |
| x/knowledge/types | 3 | FIXED |
| x/knowledge (module.go) | 1 | FIXED |
| x/billing | ~120 | FIXED |
| x/emergency | ~100 | FIXED |
| x/staking | ~90 | FIXED |
| x/disputes | ~80 | FIXED |
| x/home | ~75 | FIXED |
| x/evidence_mgmt | ~70 | FIXED |
| x/channels | ~65 | FIXED |
| x/gov | ~60 | FIXED |
| x/bvm | ~55 | FIXED |
| x/compute_pool | ~50 | FIXED |
| x/ontology | ~50 | FIXED |
| x/research | ~45 | FIXED |
| x/tree | ~40 | FIXED |
| x/capture_challenge | ~40 | FIXED |
| x/capture_defense | ~40 | FIXED |
| x/partnerships | ~35 | FIXED |
| x/schedule | ~35 | FIXED |
| x/claiming_pot | ~30 | FIXED |
| x/qualification | ~30 | FIXED |
| x/tokens | ~25 | FIXED |
| x/discovery | ~25 | FIXED |
| x/ibcratelimit | ~25 | FIXED |
| x/icaauth | ~15 | FIXED |
| x/auth | ~10 | FIXED |
| x/alignment | ~10 | FIXED |
| x/autopoiesis | ~10 | FIXED |
| x/vesting_rewards | ~5 | FIXED |
| app/ (rest_api, ante_test) | ~19 | FIXED |

## Key Patterns Applied

### 1. Value Receivers → Pointer Receivers
```go
// Before
func (p Params) Validate() error { ... }
func (gs GenesisState) Validate() error { ... }

// After
func (p *Params) Validate() error { ... }
func (gs *GenesisState) Validate() error { ... }
```

### 2. Function Signatures
```go
// Before
func (k Keeper) SetParams(ctx context.Context, p types.Params) { ... }
func (k Keeper) GetParams(ctx context.Context) types.Params { ... }

// After
func (k Keeper) SetParams(ctx context.Context, p *types.Params) { ... }
func (k Keeper) GetParams(ctx context.Context) *types.Params { ... }
```

### 3. Iterate Callbacks
```go
// Before
func (k Keeper) IterateFacts(ctx context.Context, cb func(types.Fact) bool) { ... }

// After
func (k Keeper) IterateFacts(ctx context.Context, cb func(*types.Fact) bool) { ... }
```

### 4. Range Variable Copies
```go
// Before
for _, tc := range tests {
    t.Run(tc.name, ...)
}

// After
for i := range tests {
    tc := &tests[i]
    t.Run(tc.name, ...)
}
```

### 5. Struct Literal Copies
```go
// Before
results = append(results, *ptr)  // dereference copies lock

// After
results = append(results, ptr)   // keep as pointer
```

## Files NOT Modified

- All `.pb.go` files (proto-generated, never touch)
- `x/knowledge/evaluation/category.go` / `category_test.go` (no proto types used)

## Verification

```
$ go vet ./... 2>&1 | grep -c "copies lock\|passes lock"
0

$ go build ./...
# Clean — 0 errors

$ go test ./... -count=1
# All packages pass — 0 failures
```

## Notes

- `ExtendedParams` in x/knowledge/types is a hand-written struct (not proto-generated), so value receivers are correct for it
- The `ValidatorInfo` type returned by staking keeper interface returns `[]ValidatorInfo` (value slice) — validator_selection.go converts these to `[]*ValidatorInfo` at the boundary
- Total files modified: ~120+ across 26 modules + app/
- No behavioral changes — all fixes are purely mechanical pointer-vs-value changes
