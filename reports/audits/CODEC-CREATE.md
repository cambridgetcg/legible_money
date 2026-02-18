# CODEC-CREATE: Fill RegisterInterfaces for capture_challenge, capture_defense, emergency

**Date**: 2026-02-18
**Task**: Fix RegisterInterfaces gaps in the 3 modules identified as "no codec.go" in REG-SWEEP

---

## Summary

REG-SWEEP identified 3 modules needing codec.go creation. Upon inspection, all 3 already have RegisterCodec and RegisterInterfaces **in types.go** (not a separate codec.go). Two had empty RegisterInterfaces; one was already complete.

| Module | Amino | Interfaces (Before) | Interfaces (After) | Handlers | Change |
|--------|-------|--------------------|--------------------|----------|--------|
| capture_challenge | 4 | 0 (empty) | 4 | 4 | **+4** |
| capture_defense | 3 | 0 (empty) | 3 | 3 | **+3** |
| emergency | 7 | **7** (already done) | 7 | 7 | — |
| **TOTAL** | **14** | **7** | **14** | **14** | **+7** |

**Combined with REG-SWEEP**: 152 + 7 = **159 total RegisterInterfaces types** across codebase.

---

## Per-Module Details

### 1. capture_challenge (0 → 4, +4 types)

**File**: `x/capture_challenge/types/types.go:221-227`

**Proto names verified** (`proto_names.go`):
- `MsgSubmitChallenge` → `legible.capture_challenge.v1.MsgSubmitChallenge`
- `MsgAddEvidence` → `legible.capture_challenge.v1.MsgAddEvidence`
- `MsgResolveChallenge` → `legible.capture_challenge.v1.MsgResolveChallenge`
- `MsgUpdateParams` → `legible.capture_challenge.v1.MsgUpdateParams`

**Handlers confirmed** (`keeper/msg_server.go`):
- `SubmitChallenge` (line 18)
- `AddEvidence` (line 87)
- `ResolveChallenge` (line 119)
- `UpdateParams` (line 255)

**Before:**
```go
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
    // Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}
```

**After:**
```go
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgSubmitChallenge{},
        &MsgAddEvidence{},
        &MsgResolveChallenge{},
        &MsgUpdateParams{},
    )
}
```

### 2. capture_defense (0 → 3, +3 types)

**File**: `x/capture_defense/types/types.go:229-235`

**Proto names verified** (`proto_names.go`):
- `MsgRequestQualification` → `legible.capture_defense.v1.MsgRequestQualification`
- `MsgEndorseQualification` → `legible.capture_defense.v1.MsgEndorseQualification`
- `MsgUpdateParams` → `legible.capture_defense.v1.MsgUpdateParams`

**Handlers confirmed** (`keeper/msg_server.go`):
- `RequestQualification` (line 15)
- `EndorseQualification` (line 48)
- `UpdateParams` (line 71)

**Before:**
```go
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
    // Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}
```

**After:**
```go
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgRequestQualification{},
        &MsgEndorseQualification{},
        &MsgUpdateParams{},
    )
}
```

### 3. emergency (already complete — 7 types)

**File**: `x/emergency/types/types.go:348-358`

Already had all 7 types registered:
```go
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
    registry.RegisterImplementations((*sdk.Msg)(nil),
        &MsgProposeHalt{},
        &MsgVoteHalt{},
        &MsgProposeRevert{},
        &MsgVoteRevert{},
        &MsgProposeResume{},
        &MsgVoteResume{},
        &MsgUpdateParams{},
    )
}
```

No changes needed.

---

## Types Without Proto Names

**None.** All Msg types in all 3 modules have `XXX_MessageName()` in proto_names.go.

Note: emergency's proto_names.go only has `MsgUpdateParams` and `MsgUpdateParamsResponse` because the other 6 types (ProposeHalt, VoteHalt, etc.) are proto-generated in `tx.pb.go` and already have proto type URLs.

---

## Build Output

### Per-module builds
```
$ go build ./x/capture_challenge/...   ✓ (no errors)
$ go build ./x/capture_defense/...     ✓ (no errors)
$ go build ./x/emergency/...           ✓ (no errors)
```

### Full build
```
$ go build ./...                       ✓ (no errors)
```

---

## Files Changed

| File | Change |
|------|--------|
| `x/capture_challenge/types/types.go` | RegisterInterfaces: added 4 types (was empty) |
| `x/capture_defense/types/types.go` | RegisterInterfaces: added 3 types (was empty) |

**Total: 2 files changed, 7 types added**

---

## Codebase Totals (Post REG-SWEEP + CODEC-CREATE)

| Metric | Count |
|--------|-------|
| Modules with RegisterInterfaces | 28 |
| Total types in RegisterInterfaces | 159 |
| Modules with empty RegisterInterfaces | **0** |
| Types in amino but not interfaces | **0** |
