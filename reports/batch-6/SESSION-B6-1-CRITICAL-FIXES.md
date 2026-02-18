# SESSION B6-1: CRITICAL FIXES — Module Account, CLI Wiring, Query Path

**Date**: 2026-02-18
**Scope**: 3 testnet launch blockers from batch-5 E2E testing
**Result**: All 3 fixed. Build clean, all tests pass (0 failures across 35 test packages).

---

## Fix 1: F-2 CRITICAL — lgm_gov module account not registered

**Symptom**: `MsgSubmitLIP` panics with `module account lgm_gov does not exist` (code 111222)

**Root cause confirmed**: `maccPerms` in `app/app.go:303-331` did NOT include `lgmgovtypes.ModuleName` (`"lgm_gov"`). The existing `govtypes.ModuleName` at line 308 is the standard SDK `"gov"` module, not the custom Legible governance module.

Both `SubmitLIP` (msg_server.go:49) and `CastVote` (msg_server.go:206) call `SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, coins)` where `types.ModuleName = "lgm_gov"`. Without the module account registered, the bank keeper panics.

**Fix**: Added `lgmgovtypes.ModuleName: {authtypes.Burner}` to `maccPerms` (app/app.go:331).

**Before**:
```go
computepooltypes.ModuleName:      {authtypes.Burner},
}
```

**After**:
```go
computepooltypes.ModuleName:      {authtypes.Burner},
lgmgovtypes.ModuleName:           {authtypes.Burner}, // LIP deposit escrow + vote stake escrow
}
```

The import `lgmgovtypes` was already present at app/app.go:140. Burner permission is needed because rejected LIP deposits should be burned.

---

## Fix 2: F-1 HIGH — Staking module CLI + amino not wired

**Symptom**: `register-validator` returns code 2 (tx parse error)

**Prompt diagnosis**: Wrong proto generator (protoc-gen-go vs gogoproto).
**Actual root cause**: The proto generation is correct — ALL modules in this project use `protoc-gen-go` (Google protov2) via `buf.gen.yaml`, not gogoproto. The real issues were:

1. **`GetTxCmd()` returned nil** (module.go:70-72) — CLI tx commands existed in `x/staking/client/cli/tx.go` but weren't wired to the module
2. **`GetQueryCmd()` returned nil** (module.go:76-78) — CLI query commands existed in `x/staking/client/cli/query.go` but weren't wired
3. **`RegisterLegacyAminoCodec` was empty** (module.go:37-39) — Amino types not registered for legacy signing mode

**Fix** (x/staking/module.go):

```go
// Before:
import (
    "github.com/legible-money/legible/x/staking/keeper"
    "github.com/legible-money/legible/x/staking/types"
)

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
    // Register types if using amino
}
func (AppModuleBasic) GetTxCmd() *cobra.Command {
    // Return tx commands when CLI is implemented
    return nil
}
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
    // Return query commands when CLI is implemented
    return nil
}

// After:
import (
    "github.com/legible-money/legible/x/staking/client/cli"
    "github.com/legible-money/legible/x/staking/keeper"
    "github.com/legible-money/legible/x/staking/types"
)

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
    types.RegisterCodec(cdc)
}
func (AppModuleBasic) GetTxCmd() *cobra.Command {
    return cli.NewTxCmd()
}
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
    return cli.NewQueryCmd()
}
```

The CLI files were complete and well-implemented (5 tx commands, 6 query commands) — they just weren't connected to the module registration.

---

## Fix 3: F-4 MEDIUM — lgm_gov query CLI gRPC path mismatch

**Symptom**: `legbled query lgm_gov params` fails with `unknown query path`

**Prompt diagnosis**: Missing `RegisterQueryServer` call.
**Actual root cause**: The query server WAS registered (module.go:104 calls `types.RegisterQueryServer`), but the **CLI used the wrong gRPC service path**:

- **Proto-generated service**: `legible.gov.v1.Query` (from `query_grpc.pb.go:217`)
- **CLI Invoke path**: `/legible.lgm_gov.v1.Query/*` (4 locations in `x/gov/client/cli/query.go`)

The proto package is `legible.gov.v1`, not `legible.lgm_gov.v1`. The module name (`lgm_gov`) and proto package name (`gov`) differ.

**Fix**: Replaced all 4 occurrences of `/legible.lgm_gov.v1.Query/` with `/legible.gov.v1.Query/` in `x/gov/client/cli/query.go`.

**Additionally**: Wired `GetTxCmd()` and `GetQueryCmd()` in `x/gov/module.go` (same pattern as Fix 2 — they both returned nil).

---

## Files Changed

| File | Change |
|------|--------|
| `app/app.go` | Added `lgmgovtypes.ModuleName: {authtypes.Burner}` to maccPerms |
| `x/staking/module.go` | Added `cli` import, wired GetTxCmd/GetQueryCmd/RegisterLegacyAminoCodec |
| `x/gov/module.go` | Added `cli` import, wired GetTxCmd/GetQueryCmd |
| `x/gov/client/cli/query.go` | Fixed gRPC path: `lgm_gov` → `gov` (4 occurrences) |

---

## Verification

```
$ go build ./...    # Clean — no errors
$ go test ./... -count=1
ok   github.com/legible-money/legible/app                    1.245s
ok   github.com/legible-money/legible/x/gov                  5.036s
ok   github.com/legible-money/legible/x/gov/keeper           5.385s
ok   github.com/legible-money/legible/x/staking/keeper       5.573s
... (35 test packages, 0 FAIL)
```

---

## Diagnosis Corrections

The prompt's root cause analysis was partially incorrect:

1. **F-2**: Correctly diagnosed. Missing module account in `maccPerms`.
2. **F-1**: Incorrectly attributed to `protoc-gen-go` vs `protoc-gen-gogo`. The entire project uses `protoc-gen-go` (Google protov2) via `buf.gen.yaml` — this is intentional and consistent. The actual root cause was that the staking module's `GetTxCmd()`, `GetQueryCmd()`, and `RegisterLegacyAminoCodec()` were stubbed out with nil/empty implementations, despite the CLI code existing in `x/staking/client/cli/`.
3. **F-4**: Incorrectly attributed to missing `RegisterQueryServer`. The query server was properly registered. The actual root cause was a gRPC service path mismatch: the CLI used `legible.lgm_gov.v1.Query` while the proto-generated service descriptor uses `legible.gov.v1.Query`.

---

## New Issues Discovered

1. **lgm_gov `GetTxCmd()`/`GetQueryCmd()` were also nil** — Fixed alongside F-4. The same stub pattern as staking.
2. **Proto package naming inconsistency**: The module is named `lgm_gov` (in Go/store keys) but the proto package is `legible.gov.v1`. This will be a recurring source of confusion. Consider aligning them in a future proto refactor.
