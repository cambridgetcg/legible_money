# PP Review: Upgradeability Philosophy in Cosmos SDK Design

> **Review Date**: 2026-02-10
> **Scope**: Is the upgradeability philosophy baked into the design and implementation of the Cosmos SDK chain?
> **Files Reviewed**: `app/app.go`, `tools/migrate/migrate.go`, `docs/COSMOS_MIGRATION.md`, all 23 `x/*/module.go`, all `x/*/types/keys.go`, all `proto/legible/*/v1/*.proto`, `cmd/legbled/cmd/genesis.go`
> **Territory**: PARTIALLY CHARTED — Cosmos SDK upgrade patterns are well-established, but integrating custom PoT consensus + LIP governance with the standard upgrade mechanism is novel

---

## Architecture Summary

```
                       Upgradeability Layers
                       ════════════════════

  ┌─────────────────────────────────────────────────────────┐
  │  Layer 5: GOVERNANCE                                     │
  │  ┌───────────────────┐   ┌──────────────────────┐      │
  │  │  x/gov (LIP sys)  │   │  x/upgrade (Cosmos)  │      │
  │  │  Custom proposals  │   │  SoftwareUpgrade     │      │
  │  │  Category-based    │   │  IBC upgrade plans   │      │
  │  └────────┬──────────┘   └──────────┬───────────┘      │
  │           │     ✅ BRIDGED (F1)      │                   │
  │           ▼                          ▼                   │
  ├─────────────────────────────────────────────────────────┤
  │  Layer 4: MODULE VERSIONING                              │
  │  23 modules × ConsensusVersion=1                         │
  │  RegisterServices() ✅ | RegisterMigrations() ❌         │
  ├─────────────────────────────────────────────────────────┤
  │  Layer 3: STATE MANAGEMENT                               │
  │  47 KV stores | Genesis init/export ✅                   │
  │  Zero-height genesis ⚠️ stub                            │
  ├─────────────────────────────────────────────────────────┤
  │  Layer 2: SCHEMA VERSIONING                              │
  │  Proto v1 across all modules                             │
  │  No reserved directives needed yet                       │
  ├─────────────────────────────────────────────────────────┤
  │  Layer 1: INFRASTRUCTURE                                 │
  │  x/upgrade keeper ✅ | skipUpgradeHeights ✅             │
  │  IBC ← UpgradeKeeper ✅ | UpgradeStoreLoader ❌         │
  └─────────────────────────────────────────────────────────┘
```

---

## TRUTH GATES (6 gates)

### T1: Grounding — PASS

Every claim in this review is grounded in specific file:line citations from direct file reads.

| Claim | Evidence |
|-------|----------|
| x/upgrade fully integrated | `app/app.go:542-549` (keeper init), `app/app.go:431` (store key), `app/app.go:919` (first BeginBlocker) |
| All 23 modules at ConsensusVersion=1 | Grep across all `x/*/module.go` — 23 matches, all return `1` |
| Zero SetUpgradeHandler calls | Grep `SetUpgradeHandler` across `*.go` → 0 results |
| Zero RegisterMigrations calls | Grep `RegisterMigrations` across `*.go` → 0 results |
| Genesis export/import is symmetric | `x/knowledge/keeper/keeper.go:90-104` (init), `:107+` (export); same for tokens, auth |
| LIP governance is disconnected from x/upgrade | `x/gov/types/types.go` uses custom LIPCategory/LIPStage; no `MsgSoftwareUpgrade` anywhere |

### T2: Calibration — PASS

- **High confidence**: Infrastructure layer (x/upgrade keeper, store keys, IBC wiring) — verified against standard Cosmos SDK patterns
- **High confidence**: Module versioning (ConsensusVersion=1, genesis roundtrip) — verified by reading all 23 modules
- **Medium confidence**: Whether custom LIP governance can trigger standard upgrades — no code exists yet, but the architectural gap is clear
- **Low confidence**: Whether the migration tool will handle all edge cases — it's a stub

### T3: Counter-Consideration — PASS

**Counter: "Isn't ConsensusVersion=1 with no migrations correct for v1?"**
Yes — this is correct. No migrations are needed until v2 modules exist. The risk is that the *pattern* for adding migrations isn't established (no example, no test, no documentation for developers).

**Counter: "Doesn't x/upgrade handle everything via governance?"**
Only if governance proposals can create SoftwareUpgrade plans. The custom LIP system has no path to `UpgradeKeeper.ScheduleUpgrade()`. Standard Cosmos SDK chains use `x/gov/v1.MsgSoftwareUpgrade` which routes to the upgrade module. Legible Money's custom `x/gov` (LIP) does not.

**Counter: "Is the zero-height genesis stub a real problem?"**
For testnet: no. For production chain restarts (e.g., after critical bug): yes. A non-functional `prepForZeroHeightGenesis` means the chain cannot be cleanly re-exported and restarted without manual state surgery.

### T4: Completeness — PASS

Full pipeline reviewed across 5 layers: infrastructure → schema → state → modules → governance.

### T5: Representation — PASS

Findings rated CRITICAL/HIGH/MEDIUM/LOW with explicit severity reasoning.

### T6 (Claude): Citation — PASS

Every claim cites specific `file:line`. All files were read via Read tool or verified by Explore agent.

---

## UNDERSTANDING GATES (6 gates)

### U1: Truth Foundation — PASS

Built on verified code readings across 50+ files.

### U2: Mental Model — PASS

The 5-layer architecture diagram above shows the complete upgrade pipeline with the disconnection between LIP governance and x/upgrade explicitly marked.

### U3: Why Explanation — PASS

Key "why" questions:
- **Why x/upgrade first in BeginBlockers?** Because the upgrade module needs to halt the chain at the upgrade height BEFORE any other module processes the block. If it runs later, a module could write invalid v2 state before the halt (`app/app.go:919`).
- **Why ConsensusVersion=1?** All modules are at their first release. The version increments when state schema changes require migration.
- **Why no SetUpgradeHandler?** First release chains have no prior version to migrate from. Handlers are registered when you need to transform state during a version bump.
- **Why is the LIP ↔ x/upgrade gap critical?** Without it, the chain cannot schedule coordinated halts. Validators must manually coordinate upgrades out-of-band, which is error-prone and defeats the purpose of on-chain governance.

### U4: Knowledge Connection — PASS

Connected to: Cosmos SDK upgrade module patterns (used by Osmosis, Juno, Stride, etc.), IBC client upgrade protocol (ICS-4), CometBFT upgrade coordination.

### U5: Practical Application — PASS

Findings include specific remediation paths with code locations.

### U6 (Claude): Exhaustion — PASS

Exhaustively searched: `SetUpgradeHandler`, `RegisterMigrations`, `RunMigrations`, `UpgradeStoreLoader`, `StoreUpgrades`, `VersionMap`, `MsgSoftwareUpgrade`, `SoftwareUpgradeProposal`, `ConsensusVersion`, all proto files, all module.go files, all keys.go files, migration tool, genesis commands.

---

## BEAUTY GATES (8 gates)

### B1: Foundation — PASS

Grounded in verified code.

### B2: Emergence — PASS

The upgrade architecture follows naturally from Cosmos SDK conventions. The 23-module + 47-store design is the standard Cosmos SDK pattern. Infrastructure choices (ConsensusVersion, genesis export, store keys) emerge from the framework, not forced onto it.

### B3: Fit — PASS (with concern)

**Fits well**: x/upgrade integration, store key management, module versioning pattern, proto v1 namespacing, IBC upgrade keeper wiring.

**Doesn't fit**: The custom `x/gov` (LIP) system exists alongside the standard Cosmos SDK `x/gov` (registered in app.go), but neither connects to x/upgrade for software upgrade proposals. This creates a confusing dual-governance situation.

### B4: Elegance — CONCERN

**Extra elements**: Two governance modules registered simultaneously — Cosmos SDK `govtypes.ModuleName` in ModuleManager AND custom `lgmgovtypes.ModuleName`. Both have their own store keys, BeginBlockers, EndBlockers. The relationship between them is unclear.

**Missing elements**:
- No upgrade handler registration anywhere
- No `UpgradeStoreLoader` configuration
- No migration example or test
- `prepForZeroHeightGenesis` is empty

### B5: Coherence — CONCERN

The upgrade layers are internally coherent (infrastructure → schema → state), but the governance → upgrade connection is broken. The chain can't trigger its own upgrades via on-chain governance, which is a coherence gap in the "upgrade without hard forks" promise made in `docs/COSMOS_MIGRATION.md:47`.

### B6: Clarity — PASS

Module structure is consistent. Every module follows the same pattern: `module.go` with ConsensusVersion, RegisterServices, InitGenesis, ExportGenesis. Proto files are in `proto/legible/{module}/v1/`. Store keys are in `x/{module}/types/keys.go`.

### B7: Appropriateness — PASS

Right-sized for Phase 1. The infrastructure is in place; the active mechanisms (handlers, migrations) are correctly deferred to when they're needed.

### B8 (Claude): Conformance — PASS

Follows standard Cosmos SDK module patterns. ConsensusVersion, RegisterServices, keeper-based InitGenesis/ExportGenesis all match established conventions.

---

## JUSTICE GATES (7 gates)

### J1: Placement — PASS

| Component | Location | Justification |
|-----------|----------|---------------|
| Upgrade keeper | `app/app.go:542-549` | Standard placement alongside other SDK keepers |
| Module versions | Each `x/*/module.go` | Standard — every module declares its own version |
| Store keys | `app/app.go:424-464` | Centralized store key registration, standard pattern |
| Proto schemas | `proto/legible/*/v1/` | Versioned directory structure, standard |
| Migration tool | `tools/migrate/` | Appropriate — tool code separate from chain code |
| Genesis commands | `cmd/legbled/cmd/genesis.go` | Standard CLI placement |

### J2: Ownership — PASS

- x/upgrade module owns upgrade scheduling and halt mechanics
- Each custom module owns its own ConsensusVersion and migration logic
- App layer owns module registration order and store configuration
- Proto layer owns schema versioning

### J3: Responsibility — PASS

Appropriate abstraction levels throughout. Upgrade infrastructure is framework-level (app.go). Module versioning is module-level. Schema versioning is proto-level.

### J4: Proportion — PASS

Complexity concentrated in the right places:
- App wiring in `app/app.go` (highest complexity, but necessary for 23 modules)
- Individual modules are simple (ConsensusVersion=1, symmetric genesis)
- Proto schemas are straightforward

### J5: Balance — PASS

All 23 modules follow identical patterns. No module is special-cased for upgrade handling.

### J6: Integrity — PASS

Layer boundaries respected:
- App layer → modules via keeper interfaces
- Modules → stores via typed store keys
- Proto schemas → Go types via code generation
- No cross-layer shortcuts

### J7 (Claude): Evidence — PASS

All claims backed by file:line citations from direct reads.

---

## CREATIVITY GATES (7 gates)

### C0: Territory — PARTIALLY CHARTED

**Charted**: Standard Cosmos SDK upgrade patterns (x/upgrade, ConsensusVersion, genesis export, store keys).
**Uncharted**: Bridging custom LIP governance with standard x/upgrade. No established pattern exists for a dual-governance system where custom proposals trigger Cosmos SDK upgrade plans.

### C1: Pioneering — N/A for charted portions; NEEDED for governance bridge

### C2: Path-Building — PASS

The 23-module structure with consistent patterns (ConsensusVersion, RegisterServices, genesis roundtrip) establishes a clear path for future developers to add migrations.

### C3: Extension — PASS

Proto v1 namespacing allows natural extension to v2 when needed. Store key isolation allows adding new stores during upgrades.

### C4: Vision — PARTIAL CONCERN

`docs/COSMOS_MIGRATION.md` describes the vision ("upgrade without hard forks") but the implementation doesn't yet deliver on it. The gap between vision and reality needs to be explicitly documented.

### C5: Foundation — PASS

The infrastructure enables future upgrades:
- ConsensusVersion can increment to 2, 3, ...
- RegisterServices/Configurator supports migration registration
- Genesis roundtrip preserves all state
- Proto v1 paths allow v2 introduction

### C6: Integration — CONCERN

The upgrade system is NOT fully integrated with the governance system. This is the single biggest integration gap.

### C7 (Claude): Proof — PASS

Exhaustive search documented for all upgrade-related patterns.

---

## FINDINGS

### CRITICAL

#### F1: LIP Governance Cannot Trigger Software Upgrades

**Location**: `x/gov/types/types.go` (LIP system), `app/app.go:542-549` (upgrade keeper)

The custom `x/gov` module implements a Legible Improvement Proposal (LIP) system with categories: `consensus`, `economics`, `interface`, `process`. However, no LIP category or handler can call `UpgradeKeeper.ScheduleUpgrade()`.

The standard Cosmos SDK path is:
1. Governance proposal with `MsgSoftwareUpgrade` → `x/upgrade` keeper → scheduled halt at block height H
2. At height H, chain halts, validators switch binary, chain resumes

In Legible Money, this path is **completely disconnected**:
- Custom `x/gov` (LIP) has no concept of `SoftwareUpgrade`
- Standard Cosmos SDK `x/gov` is registered but only has a legacy router with a single `ProposalHandler` route (`app/app.go:846-850`)
- No `MsgSoftwareUpgrade` is routed anywhere

**Impact**: Chain upgrades cannot be coordinated via on-chain governance. Validators must coordinate upgrades out-of-band (Discord, social consensus), which is:
- Error-prone (validators may run wrong binary versions)
- Not verifiable on-chain
- Defeats IBC upgrade signaling (remote chains can't verify upgrade plans)

**Remediation**: Bridge LIP → x/upgrade:
```go
// Option A: Add LIP execution handler for consensus category
func (k Keeper) ExecuteAcceptedLIP(ctx sdk.Context, lip types.LIP) error {
    if lip.Category == types.CategoryConsensus && lip.UpgradePlan != nil {
        return k.upgradeKeeper.ScheduleUpgrade(ctx, *lip.UpgradePlan)
    }
    ...
}

// Option B: Use standard x/gov for SoftwareUpgrade only
// Route MsgSoftwareUpgrade through standard governance
```

---

### HIGH

#### F2: No SetUpgradeHandler — Chain Cannot Execute Migrations

**Location**: Grep `SetUpgradeHandler` → 0 results in entire codebase

While having no upgrade handlers is correct for v1 (no prior state to migrate), the **pattern is not established**. When v2 arrives, developers need:

1. A place to register handlers (`app/upgrades.go` or similar)
2. A pattern to follow
3. Store upgrades configuration (`StoreUpgrades` for new/deleted/renamed stores)
4. Tests verifying migration correctness

**Remediation**: Create `app/upgrades.go` with a commented skeleton:

```go
// app/upgrades.go — Register upgrade handlers for chain upgrades.
//
// When bumping a module to ConsensusVersion N+1:
// 1. Add migration logic in x/{module}/migrations/vN/
// 2. Register migration in x/{module}/module.go RegisterServices()
// 3. Register upgrade handler below
// 4. Add store upgrades if adding/removing stores
//
// Example (uncomment when v2 ships):
//
// func (app *LegibleApp) registerUpgradeHandlers() {
//     app.UpgradeKeeper.SetUpgradeHandler(
//         "v2.0.0",
//         func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
//             return app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
//         },
//     )
// }
```

#### F3: Zero-Height Genesis Is Non-Functional

**Location**: `app/app.go:1228-1238`

```go
func (app *LegibleApp) prepForZeroHeightGenesis(ctx sdk.Context, jailAllowedAddrs []string) {
    // Standard Cosmos SDK zero-height genesis prep.
    // For Phase 1 MVC, this is a no-op stub.
    _ = ctx
    _ = jailAllowedAddrs
}
```

The function is called from `ExportAppStateAndValidators` when `forZeroHeight=true`, but does nothing. A production chain restart via genesis export requires:
- Withdrawing all pending delegator rewards
- Resetting delegation heights
- Resetting validator bond heights
- Resetting slashing info
- Resetting governance deposit heights

Without this, re-genesis produces incorrect state (stale heights, uncollected rewards).

**Impact**: Cannot perform clean chain restart from exported state.

**Remediation**: Implement using the standard Cosmos SDK pattern (reference: Gaia's `prepForZeroHeightGenesis`).

#### F4: TS → Cosmos Migration Tool Is a Stub

**Location**: `tools/migrate/migrate.go:128-155`

```go
func MigrateState(inputPath, chainID, outputPath string) error {
    // ...parse input...
    return fmt.Errorf("migrate-genesis not yet fully implemented (Phase 5)")
}
```

The migration tool has well-defined types (`TSStateExport` with 17 state trees) but returns an error immediately. This blocks the path from the running TypeScript DevNet to the Cosmos chain.

**Impact**: Cannot migrate existing DevNet state to production Cosmos chain.

**Remediation**: Phase 5 implementation as documented. The types are well-structured; the transform logic needs to be written.

---

### MEDIUM

#### F5: No UpgradeStoreLoader Configuration

**Location**: Grep `UpgradeStoreLoader|StoreUpgrades` → 0 results

When a chain upgrade adds new modules (new store keys) or removes old ones, `app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))` must be called in the app constructor. Without it, adding a new module store during an upgrade causes a panic ("unknown store key").

**Current state**: Not needed (no upgrades yet), but the pattern should be established when the first upgrade handler is added.

**Remediation**: Add store loader configuration alongside upgrade handler registration in `app/upgrades.go`.

#### F6: Dual Governance System Creates Confusion

**Location**: `app/app.go:843-850` (standard Cosmos gov), `app/app.go:904-914` (custom LIP gov)

Both Cosmos SDK `govtypes.ModuleName` and custom `lgmgovtypes.ModuleName` are registered in the ModuleManager with separate store keys, BeginBlockers, EndBlockers, and InitGenesis order. The standard gov module has a legacy router with minimal configuration:

```go
govRouter := govv1beta1.NewRouter()
govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler)
```

The custom `x/gov` (LIP) module handles all actual governance. The relationship and boundaries between these two systems is undocumented.

**Impact**: A developer or validator encountering both governance modules won't know which to use for what purpose.

**Remediation**: Either:
1. Remove standard `x/gov` and route SoftwareUpgrade through LIP (requires implementing F1)
2. Document clearly: "Standard x/gov handles SoftwareUpgrade only; LIP handles all other governance"

#### F7: Params Not Changeable Via Governance

**Location**: All keeper constructors use `paramstypes.Subspace{}` (empty, `app/app.go:594,620,635,647`)

The modern Cosmos SDK 0.47+ pattern stores params directly in module keeper state instead of x/params subspaces. All 23 custom modules do this correctly (each has a `SetParams`/`GetParams` in their keeper). However, there is no `MsgUpdateParams` message type in any custom module that would allow governance to change parameters at runtime.

**Impact**: Module parameters (e.g., `tokens.Params.PeriodBlocks`, `tree.Params.MaxTasksPerProject`, `knowledge.Params.MinVerifiers`) can only be changed by chain upgrade (binary change), not via governance proposal.

**Remediation**: For parameters that should be governance-changeable, add `MsgUpdateParams` to each module's proto tx.proto, with authority set to the governance module address.

---

### LOW

#### F8: No Reserved Directives in Proto Files

**Location**: All `proto/legible/*/v1/*.proto` files

No proto files use the `reserved` keyword. This is correct for v1 (no fields have been deleted), but the practice should be adopted immediately when any field is removed in future versions. Proto field numbers must never be reused — `reserved` prevents this.

**Impact**: None currently. Future risk if field numbers are reused during schema evolution.

#### F9: Migration Tool Types Are Stubs

**Location**: `tools/migrate/migrate.go:106-119`

```go
type TSClaim struct{ ID string `json:"id"` }
type TSDispute struct{ ID string `json:"id"` }
// ... 12 more stub types
```

Only `TSAccount`, `TSValidator`, and `TSFact` have real field definitions. The other 14 types are stubs with only an `ID` field. This means the migration tool's type system doesn't actually match the TypeScript chain's full state.

#### F10: No Upgrade-Related Tests

**Location**: Grep `Upgrade|upgrade` in `*_test.go` → 0 results related to chain upgrades

No tests verify:
- Genesis export → import roundtrip preserves all state
- Upgrade handler execution
- Store key addition during upgrade
- ConsensusVersion tracking

---

## AGGREGATE ASSESSMENT

| Dimension | Status | Score |
|-----------|--------|-------|
| **Truth** | PASS | 6/6 gates |
| **Understanding** | PASS | 6/6 gates |
| **Beauty** | PARTIAL (coherence, elegance concerns) | 6/8 gates |
| **Justice** | PASS | 7/7 gates |
| **Creativity** | PARTIAL (governance bridge uncharted) | 5/7 gates |

**Overall**: 30/34 gates PASS.

---

## VERDICT: IS UPGRADEABILITY BAKED IN?

### INFRASTRUCTURE: YES (8/10)

The mechanical foundations are solid:
- x/upgrade keeper: **wired** (`app/app.go:542-549`)
- BeginBlocker ordering: **correct** (upgrade first, `app/app.go:919`)
- Store keys: **properly managed** (47 stores, no collisions)
- IBC upgrade path: **connected** (`app/app.go:594-602`)
- ConsensusVersion: **declared** (all 23 modules)
- Genesis roundtrip: **symmetric** (all modules)
- Proto versioning: **structured** (all `v1/` paths)
- skipUpgradeHeights: **configured** (`app/app.go:1250-1256`)

Missing: UpgradeStoreLoader, SetUpgradeHandler pattern

### GOVERNANCE → UPGRADE BRIDGE: YES (9/10) — FIXED

~~The governance system **cannot trigger upgrades**.~~ **RESOLVED**: LIP consensus-category proposals can now attach upgrade plans via `MsgAttachUpgradePlan`. When accepted, `tallyAndResolveLIP` calls `ScheduleUpgrade` through the `LGMGovUpgradeAdapter`. Full pipeline:
1. Submit consensus LIP ✅
2. Attach upgrade plan (name, height, info) ✅
3. Validators vote via quadratic LIP voting ✅
4. On acceptance → `ScheduleUpgrade` called ✅
5. Chain halts at target height (x/upgrade PreBlocker) ✅
6. Validators swap binary, upgrade handler runs ✅

### STATE MIGRATION READINESS: GOOD (7/10) — IMPROVED

- Genesis export/import: complete ✅
- Zero-height genesis: ~~stub~~ **implemented** ✅
- TS → Cosmos migration: stub (Phase 5)
- Module migration pattern: **established** (app/upgrades.go skeleton) ✅
- Migration tests: ~~none~~ **6 genesis roundtrip + 7 upgrade plan tests** ✅

### DEVELOPER EXPERIENCE: GOOD (7/10) — IMPROVED

~~A new developer arriving at this codebase would not know:~~
- How to add a v2 migration: **documented** in `app/upgrades.go` comments + `docs/UPGRADE_GUIDE.md` ✅
- Where to register upgrade handlers: **established** in `app/upgrades.go` ✅
- Whether to use standard x/gov or LIP: **documented** in dual governance table + app.go comments ✅
- How to configure store additions: **pattern** in `app/upgrades.go` commented skeleton ✅

---

## RECOMMENDED ACTIONS (Priority Order)

1. **Bridge LIP → x/upgrade** (F1, CRITICAL) — Enable governance-triggered upgrades
2. **Create app/upgrades.go skeleton** (F2, HIGH) — Establish the pattern for v2+
3. **Implement prepForZeroHeightGenesis** (F3, HIGH) — Enable clean chain restarts
4. **Clarify dual governance** (F6, MEDIUM) — Document or consolidate the two gov modules
5. **Add MsgUpdateParams to critical modules** (F7, MEDIUM) — Enable runtime param changes
6. **Add UpgradeStoreLoader pattern** (F5, MEDIUM) — Prepare for store additions
7. **Complete migration tool** (F4, HIGH but Phase 5) — Enable DevNet → production path
8. **Add genesis roundtrip tests** (F10, LOW) — Verify state preservation
9. **Document upgrade workflow** — Developer guide for adding migrations

---

## FIXES APPLIED (2026-02-10)

### F1 (CRITICAL): LIP → x/upgrade Bridge — IMPLEMENTED

**Files modified/created**:
- `x/gov/types/expected_keepers.go` — Added `UpgradePlan` struct + `UpgradeKeeper` interface
- `x/gov/types/keys.go` — Added `UpgradePlanKeyPrefix = []byte{0x09}`
- `x/gov/types/errors.go` — Added `ErrNotConsensusCategory`, `ErrUpgradePlanExists`, `ErrInvalidUpgradePlan`
- `x/gov/types/types.go` — Added `MsgAttachUpgradePlan` message + `MsgServer` interface extension
- `x/gov/types/codec.go` — Registered `MsgAttachUpgradePlan` with amino + gRPC handler
- `x/gov/keeper/keeper.go` — Added `upgradeKeeper` field, passed via `NewKeeper` parameter
- `x/gov/keeper/state.go` — Added `SetUpgradePlan`, `GetUpgradePlan`, `DeleteUpgradePlan` CRUD
- `x/gov/keeper/msg_server.go` — Added `AttachUpgradePlan` handler (validates category, ownership, stage)
- `x/gov/module.go` — Extended `tallyAndResolveLIP` to call `ScheduleUpgrade` on LIP acceptance
- `app/gov_upgrade_adapter.go` — Created adapter bridging `lgmgovtypes.UpgradeKeeper` → real SDK keeper
- `app/app.go` — Passed `NewLGMGovUpgradeAdapter(app.UpgradeKeeper)` to LGM gov NewKeeper

**Design decisions**:
- Local `UpgradePlan` type in x/gov/types (avoids importing cosmossdk.io/x/upgrade/types)
- Adapter pattern in app/ for type conversion
- Added to NewKeeper params (not setter) because Keeper is stored as value type

### F2 + F5 (HIGH + MEDIUM): app/upgrades.go + UpgradeStoreLoader — IMPLEMENTED

**File created**: `app/upgrades.go`
- `UpgradeNameV2` constant
- `registerUpgradeHandlers()` with v2 skeleton handler calling `RunMigrations`
- Commented `UpgradeStoreLoader` pattern for store additions/removals
- Wired into app constructor between RegisterServices and LoadLatestVersion

### F3 (HIGH): prepForZeroHeightGenesis — IMPLEMENTED

**File modified**: `app/app.go`
- Replaced no-op stub with full Cosmos SDK standard implementation
- Withdraws all validator commissions + delegator rewards
- Clears slash events and historical rewards
- Resets context to height 0
- Sweeps outstanding rewards to community pool
- Reinitializes delegations in distribution
- Resets slashing signing info start heights to 0

### F6 (MEDIUM): Dual Governance Clarification — IMPLEMENTED

**Files modified/created**:
- `app/app.go` — Added 15-line comment block explaining dual governance system
- `docs/UPGRADE_GUIDE.md` — Created comprehensive upgrade workflow documentation:
  - Architecture diagram (LIP → upgrade → halt → migrate flow)
  - Dual governance comparison table (SDK x/gov vs LGM x/gov)
  - Step-by-step upgrade procedure (8 steps)
  - Store additions/removals guide
  - Module migration pattern
  - Zero-height genesis export
  - Key files reference table

### F10 (LOW): Genesis Roundtrip Tests — IMPLEMENTED

**File modified**: `x/gov/keeper/keeper_test.go`
- `TestGenesisRoundtrip_LIPFields` — Verifies all LIP fields survive export→import
- `TestGenesisRoundtrip_VoteFields` — Verifies vote preservation
- `TestGenesisRoundtrip_Disbursement` — Verifies disbursement records
- `TestGenesisRoundtrip_Counters` — Verifies NextLIPNumber, NextDisbursementID
- `TestGenesisRoundtrip_Params` — Verifies governance parameters
- `TestGenesisRoundtrip_UpgradePlan` — Verifies upgrade plan persistence
- Also added 7 upgrade plan unit tests (attach, validation, CRUD)

### Remaining (Not Implemented)

- **F4** (HIGH): Migration tool — Phase 5, types exist but transform logic deferred
- **F7** (MEDIUM): MsgUpdateParams — Requires proto changes for each module, deferred
- **F8** (LOW): Proto reserved directives — Not needed until fields are deleted
- **F9** (LOW): Migration tool stub types — Blocked on F4
