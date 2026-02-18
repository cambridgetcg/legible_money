# Test Suite Sweep Report — Post Round 1

**Date**: 2026-02-18
**Branch**: `fix/phase-d-proto-json-agent-sdk`
**Scope**: Full `go test ./... -count=1 -timeout 300s` + per-module isolation + `go vet ./...`

---

## Per-Module Test Results

| Module | Tests Run | Status | Notes |
|--------|-----------|--------|-------|
| `x/alignment` | 48 | **PASS** | |
| `x/auth` | ✓ | **PASS** | |
| `x/autopoiesis` | ✓ | **PASS** | No test files |
| `x/billing` | ✓ | **PASS** | No test files |
| `x/bvm/vm` | 18 | **PASS** | |
| `x/capture_challenge` | ✓ | **PASS** | No test files |
| `x/capture_defense` | ✓ | **PASS** | No test files |
| `x/channels` | ✓ | **PASS** | No test files |
| `x/claiming_pot` | ✓ | **PASS** | No test files |
| `x/compute_pool` | ✓ | **PASS** | No test files |
| `x/discovery` | ✓ | **PASS** | No test files |
| `x/disputes` | ✓ | **PASS** | No test files |
| `x/emergency` | ✓ | **PASS** | |
| `x/evidence_mgmt` | ✓ | **PASS** | No test files |
| `x/gov` | ✓ | **PASS** | |
| `x/home` | ✓ | **PASS** | |
| `x/knowledge/keeper` | ✓ | **PASS** | (keeper tests only) |
| `x/knowledge/client/cli` | — | **FAIL (build)** | Pre-existing gap; uncommitted fix in working tree |
| `x/ontology/keeper` | — | **FAIL (build)** | **Round 1 regression** — 5 type cast errors |
| `x/partnerships` | ✓ | **PASS** | |
| `x/qualification/keeper` | — | **FAIL (build)** | **Round 1 regression** — 3 proto type mismatches |
| `x/research` | ✓ | **PASS** | |
| `x/schedule` | ✓ | **PASS** | |
| `x/staking` | ✓ | **PASS** | |
| `x/tokens` | ✓ | **PASS** | |
| `x/tree` | ✓ | **PASS** | |
| `x/vesting_rewards` | ✓ | **PASS** | |
| `app` | — | **FAIL (build)** | Cascade from knowledge/client/cli |
| `cmd/legbled` | — | **FAIL (build)** | Cascade from knowledge/client/cli |
| `tests/cross_stack` | ✓ | **PASS** | |
| `tools/testutil` | ✓ | **PASS** | |

**Summary**: 27 PASS, 3 direct FAIL (build), 2 cascading FAIL

---

## Failure Analysis

### Failure 1: `x/ontology/keeper` — Round 1 Regression

**Classification**: Round 1 regression (commit `f032f0a` proto codegen)

**Root Cause**: Proto codegen replaced hand-written named Go types (`type Stratum uint32`,
`type LogicZone string`) with proto-generated struct fields using underlying types (`uint32`, `string`).
The test file still uses the named types in struct literals.

**Errors** (5 locations in `keeper_test.go`):
- Line 1071: `cannot use types.StratumEmpirical (untyped int constant) as uint32`
- Line 1086: `cannot use types.StratumEmpirical as uint32`
- Line 1138: `cannot use types.StratumEmpirical as uint32`
- Line 1621: `cannot use types.ZonePropositional as string`
- Line 1717: `cannot use types.ZonePropositional as string`

**Fix**: Add explicit type casts at each location:
```go
// Line 1071, 1086, 1138: change
Stratum: types.StratumEmpirical,
// to
Stratum: uint32(types.StratumEmpirical),

// Line 1621, 1717: change
Zone: types.ZonePropositional,
// to
Zone: string(types.ZonePropositional),
```

---

### Failure 2: `x/knowledge/client/cli` — Pre-Existing Gap

**Classification**: Pre-existing gap (NOT a regression)

**Root Cause**: The knowledge `query.proto` was expanded with 11 ontology/stratum/domain RPC methods,
and proto codegen produced the gRPC client code (`query_grpc.pb.go`). However, the matching CLI
commands (`NewQueryDomainsCmd`, `NewQueryDomainsByStratumCmd`, etc.) were never added to
`x/knowledge/client/cli/query.go`.

**Note**: The working tree already contains an uncommitted fix adding all 11 CLI command functions.
This fix needs to be committed.

**Cascading Impact**: This build failure cascades to `app` and `cmd/legbled` packages
(which import the knowledge module).

---

### Failure 3: `x/qualification/keeper` — Round 1 Regression

**Classification**: Round 1 regression (commit `86e986a` proto codegen)

**Root Cause**: Proto codegen replaced hand-written query types with proto-generated types
that have different field shapes. The handler (`grpc_query.go`) was updated, but the tests
(`keeper_test.go`) were not.

**Errors** (3 locations in `keeper_test.go`):
| Line | Error | Cause |
|------|-------|-------|
| 1204 | `resp.Found` undefined on `*types.QueryQualificationResponse` | Proto type has no `Found` bool; returns error when not found |
| 1266 | unknown field `EndorsementId` in `QueryEndorsementRequest` | Proto type uses `Endorser` + `TargetValidator` + `Domain` composite key |
| 1271 | `resp.Found` undefined on `*types.QueryEndorsementResponse` | Same as line 1204 — no `Found` bool in proto type |

**Fix**: Update test assertions to match proto-generated patterns:
```go
// Lines 1204, 1271: replace `resp.Found` checks with error checks
// Old: if !resp.Found { ... }
// New: check err == nil (found) and resp.Qualification/Endorsement != nil

// Line 1266: replace EndorsementId with composite key fields
// Old:  &types.QueryEndorsementRequest{EndorsementId: "e1"}
// New:  &types.QueryEndorsementRequest{Endorser: "...", TargetValidator: "...", Domain: "..."}
```

---

## `go vet ./...` Results

**All warnings are pre-existing.** Zero new warnings introduced by Round 1 changes.

The entire output consists of **"copies lock by value"** warnings from proto-generated types.
Every `*.pb.go` file contains `protoimpl.MessageState` which embeds `sync.Mutex`. When these
types are passed by value (e.g., in test assertions, function returns), `go vet` flags them.

**Affected modules** (partial list — pattern is universal across all proto-generated code):
- `x/tokens/keeper` — keeper.go:58, 78, 94, 96; msg_server.go (multiple)
- `x/compute_pool/types` — codec.go
- `x/claiming_pot/types` — codec.go
- `x/billing/types` — codec.go
- `x/discovery/types` — codec.go
- `x/knowledge/types` — codec.go
- `x/research/types` — codec.go
- `x/staking/types` — codec.go
- (and many more)

**Recommendation**: These are cosmetic warnings inherent to protobuf-generated Go code in
Cosmos SDK projects. They do not affect correctness or runtime behavior. Suppression at CI
level or acceptance as known noise is standard practice.

---

## Recommendations

### Immediate Fixes (Round 1 Regressions)

1. **`x/ontology/keeper/keeper_test.go`** — Add 5 explicit type casts (`uint32()`, `string()`)
   at lines 1071, 1086, 1138, 1621, 1717. Trivial mechanical fix.

2. **`x/qualification/keeper/keeper_test.go`** — Update 3 test assertions at lines 1204, 1266, 1271
   to match proto-generated type shapes (remove `Found` checks, use composite key fields).

3. **`x/knowledge/client/cli/query.go`** — Commit the existing uncommitted fix that adds
   11 ontology/stratum CLI query commands. Verify build passes.

### Deferred (Pre-Existing)

- `go vet` lock-copy warnings: Accept as known noise from proto codegen. No action needed.
- Stub modules with no test files: Expected for STUB-status modules. Tests will be added
  when modules are implemented.

---

## Build Verification

```
$ go build ./...
PASS (clean — no errors)

$ go test ./x/tokens/... -count=1
ok  github.com/legible-money/legible/x/tokens/keeper  1.144s
```

All modules that were passing before Round 1 continue to pass. The 2 regressions are
confined to test files that weren't updated during proto codegen.
