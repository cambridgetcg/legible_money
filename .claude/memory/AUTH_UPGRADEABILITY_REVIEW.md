# PP Review: Auth Module Upgradeability

**Date**: 2026-02-10
**Scope**: `x/auth/` module (14 Go files + 3 proto files)
**Reviewer**: PP (Purpose Prompter) — Full 30-gate analysis
**Grade**: D (Deficient) — 12/30 gates PASS

---

## Executive Summary

The auth module has comprehensive functionality (registration, 4-layer keys, session management, recovery) with strong test coverage (2,390 lines of tests). However, it has **zero upgrade infrastructure** — no state migration, no schema versioning, no parameter governance, and unused protobuf definitions while all state is JSON-serialized. Any schema change requires a hard fork.

---

## Findings

### CRITICAL-1: No State Migration Infrastructure
**Location**: Entire module
**PP Dimension**: TRUTH (grounded in reality?)

- No `migrations/` directory
- No `RegisterMigration()` calls in `module.go:107-110`
- No `MigrateStore` functions in keeper
- `ConsensusVersion` hardcoded to `1` at `module.go:148-149`
- **Impact**: Schema changes (adding fields to LegibleAccount, changing Params) have no upgrade path. State becomes incompatible between versions.

### CRITICAL-2: Proto/Go Type Divergence
**Location**: `proto/legible/auth/v1/genesis.proto` vs `types/genesis.go`; `state.proto` vs keeper usage
**PP Dimension**: TRUTH (accurate representation?)

5 fields exist in Go but NOT in proto:
1. `BootstrapEnabled` (genesis.go:13) — missing from genesis.proto Params
2. `BootstrapAmount` (genesis.go:14) — missing from genesis.proto Params
3. `AccountType` (msg_server.go:62) — missing from state.proto LegibleAccount
4. `OperationalPublicKey` (msg_server.go:68,169) — missing from state.proto (only hash exists)
5. `Address` field stored in Go but not as proto field in LegibleAccount

**Impact**: Proto-based clients (IBC relayers, light clients, external tools) cannot deserialize full state. Genesis import/export via proto loses data.

### HIGH-1: JSON Serialization Bypasses Protobuf
**Location**: `keeper/keeper.go:65-71` (SetAccount), all state methods
**PP Dimension**: UNDERSTANDING (deeper comprehension)

All state read/write uses `json.Marshal/Unmarshal` despite having full protobuf definitions in `state.pb.go` and `genesis.pb.go`. This means:
- No schema versioning (proto field numbers provide this)
- No forward compatibility (proto preserves unknown fields; JSON drops them)
- No type safety enforcement beyond Go struct tags
- Proto definitions are effectively dead code for state persistence

### HIGH-2: Genesis Recovery Configs Not Loaded
**Location**: `keeper/keeper.go:445-467` (InitGenesis)
**PP Dimension**: JUSTICE (correct placement?)

`ExportGenesis()` at line 511 exports recovery configs, but `InitGenesis()` at line 445-467 **never loads them**. Chain restart via genesis export/import silently loses all recovery configurations.

```go
// InitGenesis loads accounts, DIDs, session keys — but NOT recovery configs
func (k Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) error {
    k.SetParams(ctx, data.Params)
    for _, account := range data.Accounts { k.SetAccount(ctx, account) }
    for _, mapping := range data.DIDMappings { k.SetDIDMapping(ctx, mapping) }
    for _, session := range data.SessionKeys { k.SetSessionKey(ctx, session) }
    return nil  // <-- recovery_configs MISSING
}
```

### HIGH-3: No Parameter Governance (MsgUpdateParams)
**Location**: Module-wide
**PP Dimension**: CREATIVITY (adaptive paths?)

No `MsgUpdateParams` message type or handler. Parameters (`MaxSessionKeys`, `KeyRotationCooldown`, `RecoveryDelayBlocks`, etc.) can only be changed via:
- Hard fork with new binary
- Manual genesis export, modify, reimport

Standard Cosmos SDK pattern (MsgUpdateParams with authority check against governance module address) is not implemented.

### MEDIUM-1: No Schema Evolution Tests
**Location**: `keeper/keeper_test.go` (2,390 lines)
**PP Dimension**: TRUTH (verified?)

Comprehensive functional tests but zero upgradeability tests:
- No test for reading state written by older version
- No test for handling unknown JSON fields
- No test for adding new fields to LegibleAccount/Params
- No genesis roundtrip fidelity test (export === import)

### MEDIUM-2: BeginBlocker Session Cleanup is O(N)
**Location**: `module.go:130-140` (BeginBlock)
**PP Dimension**: BEAUTY (elegant?)

`CleanupExpiredSessions()` iterates ALL session keys every block. As session count grows, this becomes a performance bottleneck. Upgrade to indexed expiry queue would require state migration (which doesn't exist).

### MEDIUM-3: Recovery `executable` Status Has No Timeout
**Location**: `keeper/msg_server.go` (recovery flow)
**PP Dimension**: JUSTICE (fair placement?)

Once a recovery request reaches `executable` status after the delay period, it stays executable indefinitely. No timeout means a forgotten/abandoned recovery can be executed at any future time. Adding a timeout field would require schema migration.

### MEDIUM-4: Invariants Not Implemented
**Location**: `module.go:113-115`
**PP Dimension**: TRUTH (verifiable?)

```go
func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
    // Phase 2: Register invariants
}
```

No invariants to detect state corruption during upgrades. Key invariants missing:
- Account count matches DID mapping count
- All session keys reference valid accounts
- Recovery configs reference valid accounts

### LOW-1: prefixEndBytes Duplicated
**Location**: `keeper/keeper.go:42-55`
**PP Dimension**: BEAUTY (DRY?)

Same 14-line helper function duplicated across 23+ modules. Not directly an upgradeability issue, but indicates no shared utility library — making coordinated upgrades across modules harder.

### LOW-2: Session Key Block Heights are Upgrade-Safe
**Location**: All session structures
**PP Dimension**: BEAUTY (consistent?)

Positive finding: Uses `uint64` block heights (not timestamps) for all expiry and cooldown logic. This is deterministic across nodes and upgrade-safe.

### LOW-3: No Snapshot/IAVL Migration Hooks
**Location**: Module-wide
**PP Dimension**: CREATIVITY (future paths?)

No `SnapshotExtension` implementation. If state format changes, snapshot restore from older state would silently produce corrupted data.

---

## PP Gate Scorecard

| Dimension | Gates | Pass | Warn | Fail |
|-----------|-------|------|------|------|
| TRUTH | 5 | 2 | 1 | 2 |
| UNDERSTANDING | 5 | 2 | 2 | 1 |
| BEAUTY | 7 | 3 | 2 | 2 |
| JUSTICE | 6 | 3 | 1 | 2 |
| CREATIVITY | 7 | 2 | 1 | 4 |
| **Total** | **30** | **12** | **7** | **11** |

---

## Backwards Compatibility Assessment

| Question | Answer |
|----------|--------|
| Can old node read new state? | NO — JSON field additions may break |
| Can new node read old state? | MAYBE — additive fields with defaults might work, untested |
| Can mixed-version nodes coexist? | NO — consensus breaks on state divergence |
| Is there an upgrade coordinator? | NO — no x/upgrade integration |

---

## Relationship to System-Wide Upgradeability

Per `.claude/memory/UPGRADEABILITY_PP_REVIEW.md`:
- **All 23 modules** are at ConsensusVersion=1 with no migrations
- The auth module is **representative** of the system-wide pattern, not an outlier
- The **CRITICAL gap** (LIP governance disconnected from x/upgrade) compounds: even if auth had migrations, there's no governance path to trigger them
- Auth module is **higher risk** than most because it manages identity state (DIDs, keys, sessions) — data loss here is catastrophic

---

## Priority Remediation

1. **P0 FIXED**: Fix `InitGenesis` to load `recovery_configs` — loads and exports with `AccountAddress` field
2. **P0 FIXED**: Sync proto definitions with Go types — rewrote state.pb.go + genesis.pb.go as plain Go structs, updated protos
3. **P1 DEFERRED**: Switch state serialization from JSON to protobuf (system-wide pattern, not auth-specific)
4. **P1 FIXED**: Add `migrations/v2/` with `MigrateStore` skeleton — `x/auth/migrations/v2/migrate.go` + `keeper/migrator.go`, ConsensusVersion bumped to 2
5. **P1 FIXED**: Implement `MsgUpdateParams` with authority check — full gRPC handler, 3 tests
6. **P2 FIXED**: Add schema evolution tests — genesis roundtrip with recovery configs test
7. **P2 FIXED**: Implement invariants — 3 invariants (account-did-parity, session-count-consistency, params-valid) + 5 invariant tests
8. **P3**: Add session expiry index (replaces O(N) scan)
9. **P3**: Add recovery execution timeout

**Post-remediation test count**: 79 tests, all passing
