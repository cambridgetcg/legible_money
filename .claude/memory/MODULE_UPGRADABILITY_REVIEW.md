# Module Upgradability & Openness Review

> **Date**: 2026-02-13
> **Scope**: All 25 custom Go/Cosmos SDK modules under `x/`
> **Goal**: Ensure smooth post-launch upgrades without service interruption

---

## Executive Summary

The Legible Money chain has **solid upgrade infrastructure** (x/upgrade wired, governance bridge, upgrade handler skeleton). ~~Critical gaps in per-module migration readiness~~ **RESOLVED (2026-02-13)**: All 25 modules now have migration scaffolding (migrator.go + RegisterMigration). The remaining major risk is the universal use of `json.Marshal`/`json.Unmarshal` for state persistence instead of protobuf codec — making schema evolution fragile.

### P0 Actions Status (2026-02-13)

| # | Action | Status |
|---|--------|--------|
| 1 | Add migration template to every module | **DONE** — 24 `migrator.go` files + RegisterMigration in all module.go |
| 2 | Create migration test harness | **DONE** — `x/testutil/migration_harness.go` with 6 passing tests |
| 3 | Document BeginBlocker ordering rationale | **DONE** — Inline comments in `app/app.go` (Begin, End, Init ordering) |

### Risk Summary

| Risk Level | Count | Description |
|------------|-------|-------------|
| **CRITICAL** | 2 | JSON serialization fragility, missing migration templates |
| **HIGH** | 3 | Cross-module dependency chains, BeginBlocker ordering, adapter interface lock-in |
| **MEDIUM** | 4 | Proto/JSON type divergence, genesis export gaps, no rollback testing, parameter governance |
| **LOW** | 2 | Stub module activation, unused proto types |

---

## 1. Upgrade Infrastructure Assessment

### 1.1 What's Working

| Component | File | Status |
|-----------|------|--------|
| x/upgrade keeper | `app/app.go:573` | **Wired** — PreBlocker halts chain at scheduled height |
| Upgrade handler skeleton | `app/upgrades.go` | **Ready** — v2 placeholder with `RunMigrations` call |
| StoreUpgrades pattern | `app/upgrades.go:68-79` | **Documented** — commented example for adding/removing stores |
| Governance bridge | `app/gov_upgrade_adapter.go` | **Working** — LIP consensus proposals can schedule upgrades |
| Module configurator | `app/app.go:1154` | **Wired** — `RegisterServices` called for all modules |
| BeginBlocker ordering | `app/app.go:1023-1061` | **Explicit** — `upgradetypes.ModuleName` runs first |

### 1.2 What's Missing

| Gap | Severity | Description |
|-----|----------|-------------|
| **Migration templates** | CRITICAL | Only x/auth has `RegisterMigration`. 24 modules have no migration path. |
| **Migration test harness** | HIGH | No framework for testing state migrations (before/after schema snapshots) |
| **Version map snapshot** | MEDIUM | No tooling to dump current `VersionMap` for comparison during upgrades |
| **Rollback plan** | MEDIUM | No documented rollback procedure if upgrade handler panics |
| **Multi-step upgrades** | LOW | No pattern for chaining migrations (v1→v2→v3) if a module skips a version |

---

## 2. Serialization Risk: JSON vs Protobuf

### The Core Problem

**Every keeper** in the chain uses `json.Marshal`/`json.Unmarshal` for KV store persistence:

| Keeper File | json.Marshal/Unmarshal calls |
|-------------|------------------------------|
| `x/knowledge/keeper/state.go` | 15 |
| `x/ontology/keeper/state.go` | 6 |
| `x/research/keeper/state.go` | 6 |
| `x/ibcratelimit/keeper/middleware.go` | 4 |
| `x/auth/keeper/keeper.go` | 3 |
| `x/home/keeper/state.go` | 3 |
| `x/gov/keeper/state.go` | 2 |
| `x/knowledge/keeper/provisional.go` | 3 |
| Others (5 files) | 1 each |
| **Total** | **47 calls across 13 files** |

**Zero** `cdc.Marshal` / `cdc.MustMarshal` calls in any keeper.

### Why This Is Dangerous

1. **No schema evolution guarantees**: JSON has no field numbering. Renaming a field breaks deserialization of existing state.
2. **No unknown field preservation**: Adding a field to a Go struct causes `json.Unmarshal` to silently ignore the new field in old data, but **removing** a field drops data permanently.
3. **No backward/forward compatibility contract**: Proto guarantees additive-only changes are safe. JSON provides no such guarantee.
4. **Numeric precision**: `json.Marshal` converts `int64`/`uint64` to JSON numbers, which can lose precision above 2^53 when round-tripping through JS clients.

### Why It Happened

The types are hand-written Go structs (not proto-generated), and proto registration panics with `gogoproto` codec when types aren't generated from `.proto` files. The `json.Marshal` path was a pragmatic workaround.

### Proto Files Exist But Are Unused for State

All 25 modules have `.pb.go` files (83 total proto-generated files), but keepers don't use them for state persistence:
- `genesis.pb.go` — exists for all modules (used for genesis JSON export)
- `state.pb.go` — exists for all modules (defines state types but not used by keeper CRUD)
- `tx.pb.go` / `tx_grpc.pb.go` — exists for 16 modules (used for gRPC message routing)

### Migration Path: JSON → Proto

**Recommended strategy**: Dual-read migration per module:

```go
// Phase 1: Read JSON, write Proto
func migrateStateV1toV2(store sdk.KVStore) error {
    // Read old JSON-encoded value
    bz := store.Get(key)
    var oldState OldTypeJSON
    json.Unmarshal(bz, &oldState)

    // Convert to proto type
    newState := &NewTypeProto{...} // map fields

    // Write proto-encoded value
    store.Set(key, cdc.MustMarshal(newState))
}
```

**Effort estimate**: ~200 LOC per module for the most complex (knowledge: ~15 conversions), ~20 LOC for simple ones.

---

## 3. Per-Module Upgrade Risk Rating

### Rating Scale

- **GREEN**: Safe to upgrade with minimal work
- **YELLOW**: Upgradable but needs migration scaffolding first
- **RED**: Requires careful planning — high dependency count or complex state

### 3.1 Core Modules (RED — upgrade carefully)

| Module | Version | Dependencies | Dependents | BeginBlocker | State Complexity | Rating |
|--------|---------|-------------|------------|--------------|------------------|--------|
| **x/knowledge** | 1 | 9 keepers | 4 modules | `BeginBlockerAdversarial`, `ExpireStaleFactsInEndBlocker` | 15 json calls, facts/claims/rounds/domains | **RED** |
| **x/staking** | 1 | 4 keepers | 9+ modules | Credits with OriginPeriod | Validators, tiers, unbonding entries | **RED** |
| **x/auth** | **2** | 2 keepers | 4+ modules | `CleanupExpiredSessions`, `ProcessRecoveryTimeouts` | Accounts, sessions, recovery configs | **YELLOW** (has migration) |

**x/knowledge** is the most dangerous module to upgrade — 9 dependencies, 4 dependents, 15 JSON serialization sites, and complex BeginBlocker logic. Any schema change here ripples across the entire chain.

**x/staking** is equally critical — 9+ modules depend on its keeper interface. Changing the `LegibleValidator` type breaks every adapter.

### 3.2 Economic Modules (YELLOW — need migration scaffolding)

| Module | Version | Dependencies | Dependents | BeginBlocker | State Complexity | Rating |
|--------|---------|-------------|------------|--------------|------------------|--------|
| **x/tokens** | 1 | 1 (BankKeeper) | 3 modules | Sweep expired, RAI distribution | Periods, balances, allowances | **YELLOW** |
| **x/vesting_rewards** | 1 | 2 keepers | 2 modules | No-op | Schedules, periods | **YELLOW** |
| **x/gov** | 1 | 3 keepers | 0 | Voting deadline check + tally | LIPs, votes, disbursements (2 json calls) | **YELLOW** |
| **x/billing** | 1 | 3 keepers | 0 | Expire quotes | Quotes, pricing | **YELLOW** |
| **x/claiming_pot** | 1 | 1 (BankKeeper) | 0 | Expire old pots | Pots, claims | **GREEN** |
| **x/research** | 1 | 2 keepers | 0 | Expire bounties | Bounties, proposals (6 json calls) | **YELLOW** |

### 3.3 Safety Modules (YELLOW — non-trivial state + dependency chains)

| Module | Version | Dependencies | Dependents | BeginBlocker | State Complexity | Rating |
|--------|---------|-------------|------------|--------------|------------------|--------|
| **x/emergency** | 1 | 1 (StakingKeeper) | 1 (capture_defense) | Ceremony progress | Ceremonies, audit log, counters | **YELLOW** |
| **x/capture_defense** | 1 | 4 keepers | 2 modules | Reputation decay, qualification expiry | Scores, domains, thresholds | **YELLOW** |
| **x/capture_challenge** | 1 | 3 keepers | 0 | Advance phases, expire challenges | Challenges, bounty pools | **YELLOW** |
| **x/evidence_mgmt** | 1 | 3 keepers | 1 (knowledge) | No-op | Evidence records, reliability scores | **YELLOW** |
| **x/disputes** | 1 | 4 keepers | 0 | No-op | Disputes, votes, clawbacks | **YELLOW** |

### 3.4 Domain Modules (GREEN — low risk, few dependents)

| Module | Version | Dependencies | Dependents | BeginBlocker | State Complexity | Rating |
|--------|---------|-------------|------------|--------------|------------------|--------|
| **x/ontology** | 1 | 1 (BankKeeper) | 3 modules | No-op | Domains, strata, properties (6 json calls) | **YELLOW** |
| **x/qualification** | 1 | 3 keepers | 1 (knowledge) | Promote/expire qualifications | Qualifications, domain caps | **GREEN** |
| **x/tree** | 1 | 1 (BankKeeper) | 0 | Expire seeds, task deadlines | Seeds, branches, tasks | **GREEN** |

### 3.5 Agent Modules (GREEN — mostly isolated)

| Module | Version | Dependencies | Dependents | BeginBlocker | State Complexity | Rating |
|--------|---------|-------------|------------|--------------|------------------|--------|
| **x/home** | 1 | 2 keepers | 0 | Deadman, sessions, treasury, aging | Homes, keys, sessions, alerts (3 json calls) | **GREEN** |
| **x/discovery** | 1 | 1 (BankKeeper) | 0 | No-op | Profiles, capabilities | **GREEN** |
| **x/partnerships** | 1 | 2 keepers | 0 | No-op | Partnerships, tiers | **GREEN** |

### 3.6 Execution & Infrastructure (YELLOW — complex internal state)

| Module | Version | Dependencies | Dependents | BeginBlocker | State Complexity | Rating |
|--------|---------|-------------|------------|--------------|------------------|--------|
| **x/bvm** | 1 | 4 keepers (via adapters) | 0 | Execute pending schedules | Contracts, code, storage, nonces | **YELLOW** |
| **x/schedule** | 1 | 2 keepers | 0 | No-op in Begin; process due schedules in End | Schedules, processes | **GREEN** |
| **x/channels** | 1 | 1 (BankKeeper) | 0 | Resolve expired disputes | Channels, balances | **GREEN** |

### 3.7 IBC & Cross-Chain (GREEN — standard patterns)

| Module | Version | Dependencies | Dependents | BeginBlocker | State Complexity | Rating |
|--------|---------|-------------|------------|--------------|------------------|--------|
| **x/icaauth** | 1 | 2 keepers | 0 | No-op | Auth entries | **GREEN** |
| **x/ibcratelimit** | 1 | 0 | 0 | Reset expired flow records | Flow records, limits | **GREEN** |

---

## 4. Cross-Module Dependency Graph

### Critical Dependency Chains

```
                      ┌─────────────────┐
                      │  SDK BankKeeper  │ ← ALL 25 modules depend on this
                      └────────┬────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
        ┌─────┴─────┐  ┌──────┴──────┐  ┌──────┴──────┐
        │  x/tokens  │  │  x/auth     │  │ (13 leaf    │
        └─────┬─────┘  └──────┬──────┘  │  modules)   │
              │                │         └─────────────┘
        ┌─────┴─────────┬─────┘
        │               │
   ┌────┴────┐    ┌─────┴──────────────────────────────┐
   │x/staking│    │                                     │
   └────┬────┘    │  x/knowledge  (CENTRAL HUB)        │
        │         │  9 deps: account, bank, staking,    │
        │         │  tokens, ontology, qualification,   │
        │         │  capture_defense, vesting_rewards,  │
        │         │  evidence_mgmt                      │
        │         └─────┬──────────────────────────────┘
        │               │
   ┌────┴────┐    ┌─────┴──────┐
   │ 9+ deps │    │  billing,  │
   │ via      │    │  disputes, │
   │ adapters │    │  bvm,      │
   └─────────┘    │  evidence  │
                  └────────────┘
```

### Adapter Layer (11 adapters in app/adapters.go)

Adapters break circular dependencies but **lock interface shapes**:

| Adapter | Wraps | Used By | Interface Width |
|---------|-------|---------|-----------------|
| StakingKeeperAdapter | x/staking | knowledge, capture_defense, capture_challenge, disputes | ~6 methods |
| StakeQueryAdapter | x/staking | x/gov, x/research | ~3 methods |
| VestingStakingAdapter | x/staking | x/vesting_rewards | ~2 methods |
| CaptureDefenseOntologyAdapter | x/ontology | x/capture_defense | ~3 methods |
| KnowledgeKeeperForEvidence | x/knowledge | x/evidence_mgmt | ~4 methods |
| BillingKnowledgeAdapter | x/knowledge | x/billing | ~2 methods |
| BVM{Knowledge,Staking,Auth}Adapter | Various | x/bvm | ~3 methods each |
| Dispute{Knowledge,Staking,Vesting}Adapter | Various | x/disputes | ~3 methods each |
| LGMGovUpgradeAdapter | x/upgrade | x/gov | 1 method |
| PartnershipsStakingAdapter | x/staking | x/partnerships | ~2 methods |
| ICAControllerAdapter | ICA controller | x/icaauth | ~2 methods |

**Upgrade risk**: Changing a keeper's method signature requires updating **every adapter** that wraps it. The x/staking keeper has **5 different adapters** — any method signature change touches 5 files.

### Post-Init Cross-Wiring (7 hooks in app/app.go:875-913)

These bidirectional references are set after all keepers are constructed:

```go
EvidenceMgmtKeeper.SetKnowledgeKeeper()      // line 875
KnowledgeKeeper.SetOntologyKeeper()            // line 887
KnowledgeKeeper.SetVerificationHooks()         // line 890-895
BVMKeeper.SetKnowledgeKeeper()                 // line 898
BVMKeeper.SetStakingKeeper()                   // line 899
BVMKeeper.SetAuthKeeper()                      // line 900
DisputesKeeper.SetVestingKeeper()              // line 910
CaptureDefenseKeeper.SetEmergencyKeeper()      // line 913
```

**Upgrade risk**: These `Set*Keeper` methods accept interface types. If the interface changes, the set-up fails at app construction time — which means the chain won't start after a binary swap.

---

## 5. BeginBlocker / EndBlocker Ordering

The execution order of block-lifecycle hooks affects upgrade safety. If a module's BeginBlocker depends on another module's state being updated first, reordering during an upgrade could cause state inconsistencies.

### BeginBlocker Order (from app/app.go:1023-1061)

```
1.  upgradetypes.ModuleName        ← MUST be first (halts at upgrade height)
2.  capabilitykeeper.ModuleName    ← SDK modules
3-12. [other SDK modules]
13. lgm_auth                       ← Session cleanup, recovery timeouts
14. lgm_staking                    ← Credits with OriginPeriod
15. lgm_tokens                     ← Period sweep, RAI distribution
16. lgm_emergency                  ← Ceremony progress
17. lgm_knowledge                  ← Adversarial BeginBlocker, round phases
18. lgm_evidence_mgmt
19. lgm_ontology
20. lgm_qualification              ← Promote/expire qualifications
21. lgm_capture_defense            ← Reputation decay
22. lgm_capture_challenge          ← Phase advancement
23. lgm_research                   ← Bounty expiration
24. lgm_gov                        ← Voting deadline check
25. lgm_channels                   ← Expire disputes
26. lgm_claiming_pot               ← Expire old pots
27. lgm_tree                       ← Expire seeds, task deadlines
28. lgm_home                       ← Deadman switch, session cleanup, aging
29. lgm_bvm                        ← Execute pending schedules
30. lgm_schedule                   ← (EndBlocker processes due schedules)
31. lgm_vesting_rewards
32. lgm_billing
33. lgm_disputes
34. lgm_discovery, lgm_partnerships, lgm_icaauth, lgm_ibcratelimit
```

### Critical Ordering Dependencies

| Dependency | Why It Matters |
|------------|---------------|
| `upgrade` before ALL | Chain halts before any module touches state |
| `auth` before `staking` | Session cleanup before stake operations |
| `staking` before `knowledge` | Validator set must be current for round phases |
| `tokens` before `knowledge` | Period data must be current for debit operations |
| `emergency` before `capture_defense` | Halt status must be known for reputation decay |
| `ontology` before `qualification` | Domain structure must be current for qualification checks |
| `qualification` before `knowledge` | Validator eligibility must be current for round selection |

**Upgrade recommendation**: Never reorder BeginBlockers without verifying all dependency chains. Document the ordering rationale in `app/app.go` as inline comments.

---

## 6. Module Openness Assessment

"Openness" = how easy it is to extend a module without modifying its core code.

### 6.1 Extension Points

| Extension Type | Modules That Support It | Pattern |
|----------------|------------------------|---------|
| **Keeper interfaces** | All 25 (via `expected_keepers.go`) | Interface-based dependency injection |
| **Post-init hooks** | knowledge, bvm, disputes, evidence_mgmt, capture_defense | `Set*Keeper()` methods |
| **BeginBlock/EndBlock** | All 25 | Override module.go hooks |
| **Msg handlers** | All 25 (via `RegisterServices`) | Add new message types |
| **gRPC queries** | All 25 (via `RegisterServices`) | Add new query types |
| **Genesis import/export** | All 25 | `InitGenesis`/`ExportGenesis` |
| **Parameters** | ~15 modules | Params type in state |
| **Governance-driven upgrade** | Chain-wide | LIP → x/upgrade bridge |

### 6.2 Module Extensibility Ratings

| Rating | Module | Notes |
|--------|--------|-------|
| **Highly Extensible** | x/knowledge | 9 keeper interfaces, verification hooks, ontology hooks, capture hooks |
| **Highly Extensible** | x/bvm | 3 host interfaces (agent, knowledge, treasury), opcode registry |
| **Extensible** | x/staking | TokensKeeper optional, tier system parameterized |
| **Extensible** | x/home | TokensKeeper nil-safe, guardian strategies pluggable |
| **Extensible** | x/gov | UpgradeKeeper interface, category system parameterized |
| **Moderate** | x/emergency | StakingKeeper interface, but ceremony logic is monolithic |
| **Moderate** | x/tokens | Period system is generic, but RAI logic is hardcoded |
| **Limited** | x/ontology | Domain/stratum structure is rigid, no plugin points |
| **Limited** | x/schedule | Process execution is generic, but condition types are fixed |
| **Minimal** | Leaf modules (discovery, partnerships, icaauth, ibcratelimit) | Simple CRUD, little need for extension |

---

## 7. Recommended Actions (Prioritized)

### P0 — Before Any Upgrade (Pre-Launch Hardening)

| # | Action | Effort | Impact |
|---|--------|--------|--------|
| 1 | **Add migration template to every module** | M (1-2 days) | Every module needs `RegisterMigration` in `RegisterServices`, a `Migrator` struct, and a stub `Migrate1to2` function. Without this, `RunMigrations` silently skips the module. |
| 2 | **Create migration test harness** | M (1 day) | Test helper that: (a) seeds KV store with v1 JSON data, (b) runs migration, (c) verifies v2 data reads correctly. Pattern from x/auth's Migrate1to2 should be generalized. |
| 3 | **Document BeginBlocker ordering rationale** | S (2 hours) | Add inline comments in `app/app.go:1023-1061` explaining WHY each module is ordered where it is. This prevents accidental reordering during upgrades. |

### P1 — First Upgrade Cycle (v2)

| # | Action | Effort | Impact |
|---|--------|--------|--------|
| 4 | **Migrate x/knowledge state to proto codec** | L (3-5 days) | 15 JSON serialization sites → proto. This is the highest-state-complexity module and the riskiest to leave on JSON. |
| 5 | **Migrate x/staking state to proto codec** | M (2-3 days) | 9+ dependent modules means any staking schema change must be rock-solid. Proto gives backward compatibility guarantees. |
| 6 | **Lock adapter interface versions** | S (4 hours) | Add compile-time interface assertions for all 11 adapters. If a keeper method changes, compilation fails immediately rather than at runtime. |
| 7 | **Add UpgradeStoreLoader for new modules** | S (1 hour) | Uncomment the storeUpgrades block in `app/upgrades.go` and populate it whenever a new module store key is added in a future upgrade. |
| 8 | **Implement rollback documentation** | S (2 hours) | Document the procedure for reverting a failed upgrade: (a) restore pre-upgrade binary, (b) `--unsafe-skip-upgrades` flag, (c) state export/import. |

### P2 — Medium-Term (v3+)

| # | Action | Effort | Impact |
|---|--------|--------|--------|
| 9 | **Migrate remaining modules to proto codec** | L (1-2 weeks) | Cover all 13 keeper files (47 JSON calls). Priority order: ontology (6), research (6), ibcratelimit (4), auth (3), home (3), gov (2), then remaining. |
| 10 | **Add module version snapshot tooling** | M (2 days) | CLI command that dumps `{module: version}` map for comparison before/after upgrades. |
| 11 | **Implement multi-version migration chaining** | M (2 days) | Support `Migrate1to2`, `Migrate2to3` chains so modules can skip intermediate versions. |
| 12 | **Governance parameter migration** | M (3 days) | Formalize how params change across versions — some modules store params in custom KV (not x/params), which needs its own migration path. |

### P3 — Long-Term

| # | Action | Effort | Impact |
|---|--------|--------|--------|
| 13 | **Module hot-loading** | XL | Cosmos SDK doesn't support module hot-swap. For zero-downtime upgrades, rely on the standard halt-at-height + binary swap pattern. Consider a module registry that can enable/disable modules via governance. |
| 14 | **State export/import versioning** | L | Tag genesis exports with module versions so import can detect version mismatches and apply migrations automatically. |
| 15 | **Canary upgrade testing** | L | CI pipeline that runs upgrade simulations: seed testnet with v1 state, apply v2 binary, verify state integrity. |

---

## 8. Zero-Downtime Upgrade Flow

The Cosmos SDK halt-at-height pattern provides **near-zero-downtime** upgrades (only blocks during the binary swap are missed, typically <30 seconds):

```
┌──────────────────────────────────────────────────────┐
│  1. Governance: Accept consensus LIP with upgrade    │
│     plan (name="v2", height=500000)                  │
│                                                      │
│  2. LGMGovUpgradeAdapter.ScheduleUpgrade()           │
│     → x/upgrade stores plan on-chain                 │
│                                                      │
│  3. Chain runs normally until block 500000            │
│                                                      │
│  4. At block 500000:                                 │
│     x/upgrade PreBlocker detects plan                │
│     → Chain HALTS (all validators stop)              │
│                                                      │
│  5. Validators swap binary (v1 → v2)                 │
│     → Restart nodes with new binary                  │
│                                                      │
│  6. On first block of v2:                            │
│     registerUpgradeHandlers() runs v2 handler        │
│     → RunMigrations() iterates all modules           │
│     → Modules with bumped ConsensusVersion get       │
│       their registered migration function called     │
│     → StoreUpgrades adds/removes store keys          │
│                                                      │
│  7. Chain resumes normal operation                    │
│     Downtime = binary swap time (~10-30 seconds)     │
└──────────────────────────────────────────────────────┘
```

### What Each Module Needs for This Flow

```go
// In x/<module>/module.go:
func (AppModule) ConsensusVersion() uint64 { return 2 } // Bump from 1

func (am AppModule) RegisterServices(cfg module.Configurator) {
    // Existing service registration...

    migrator := keeper.NewMigrator(am.keeper)
    if err := cfg.RegisterMigration(types.ModuleName, 1, migrator.Migrate1to2); err != nil {
        panic(fmt.Sprintf("failed to register migration: %v", err))
    }
}

// In x/<module>/keeper/migrator.go:
type Migrator struct {
    keeper Keeper
}

func NewMigrator(k Keeper) Migrator {
    return Migrator{keeper: k}
}

func (m Migrator) Migrate1to2(ctx context.Context) error {
    // Iterate KV store, re-encode values, add new fields, etc.
    return nil
}
```

---

## 9. Module-Specific Upgrade Notes

### x/knowledge (RED — most complex)

- **9 keeper dependencies** means any upgrade must coordinate with auth, staking, tokens, ontology, qualification, capture_defense, vesting_rewards, evidence_mgmt, and bank.
- **15 JSON serialization sites** in `state.go` and `provisional.go`. Schema changes to `Fact`, `Claim`, `VerificationRound`, or `Domain` types require migration of all stored instances.
- **BeginBlockerAdversarial** runs complex adversarial-resistance logic — any behavior change here needs thorough testing under adversarial conditions.
- **Verification hooks** (staking + qualification) are set post-init. Changing hook signatures breaks the app at construction time.
- **Recommendation**: Migrate to proto codec first. Create a dedicated `x/knowledge/migrations/v2/` package with comprehensive tests.

### x/staking (RED — most dependents)

- **9+ dependent modules** via 5 different adapters. The `LegibleStakingKeeper` interface is the most-consumed interface in the chain.
- Changing any method signature on the keeper requires updating: `StakingKeeperAdapter`, `StakeQueryAdapter`, `VestingStakingAdapter`, `PartnershipsStakingAdapter`, and all expected_keepers.go files that reference it.
- **Tier system** (Apprentice→Verified→Bonded→Guardian) has constants that affect validator economics. Changing tier thresholds requires a governance proposal + migration to recalculate all validators.
- **Recommendation**: Freeze the keeper interface and add new methods rather than modifying existing ones. Use compile-time assertions.

### x/bvm (YELLOW — complex internal state)

- **Contract code + storage** is the most latent-risk state. If `CosmosStateDB` encoding changes, all deployed contracts lose their storage.
- **Host interfaces** (AgentHost, KnowledgeHost, TreasuryHost) are consumed by the interpreter. Changing host method signatures requires updating ~283 delegation methods in `prodCompositeHost`.
- **Recommendation**: Version the host interface (HostV1, HostV2) so contracts compiled against HostV1 continue to work after an upgrade.

### x/tokens (YELLOW — period system complexity)

- **Period-based expiration** means state layout is time-dependent. Migrating balances requires understanding which period each balance belongs to.
- **RAI distribution** in EndBlocker has economic effects. Changes must be coordinated with tokenomics analysis.
- **Recommendation**: Add period metadata to the migration (which period was current at migration time) to avoid expiration edge cases.

### x/emergency (YELLOW — safety-critical)

- **Must remain functional during upgrades**. If the emergency module fails after a binary swap, the chain loses its kill switch.
- **Ceremony state** (active ceremonies, audit log) must survive migration intact. A partial migration could leave a ceremony in an inconsistent state.
- **Recommendation**: Emergency module should be the LAST module migrated in any upgrade, and its migration should be the simplest possible (ideally no-op unless absolutely necessary).

---

## 10. Testing Checklist for Any Upgrade

```
Pre-Upgrade:
  [ ] All modules have RegisterMigration for the target version
  [ ] Migration functions have unit tests (old state → new state)
  [ ] UpgradeStoreLoader configured if adding/removing stores
  [ ] Genesis export works on current version (verify with `legbled export`)
  [ ] Adapter compile-time assertions pass

Upgrade Execution:
  [ ] Test on local devnet first (single validator)
  [ ] Test on multi-validator testnet
  [ ] Verify all BeginBlocker/EndBlocker hooks execute without error
  [ ] Verify all gRPC queries return correct data
  [ ] Verify IBC channels remain functional
  [ ] Verify governance proposals can be submitted

Post-Upgrade:
  [ ] All module ConsensusVersions match expected values
  [ ] Genesis export + re-import produces identical state
  [ ] Emergency kill switch still functional
  [ ] BVM contracts execute correctly with stored state
  [ ] Token balances and periods are correct
```

---

## Appendix A: Full Module Inventory

| # | Module | Store Key | ConsensusVersion | Migrations | JSON Calls | Proto Files | BeginBlocker | EndBlocker |
|---|--------|-----------|-----------------|------------|------------|-------------|-------------|------------|
| 1 | x/auth | lgm_auth | **2** | **Migrate1to2** | 3 | genesis, state, tx, tx_grpc | Session cleanup, recovery timeouts | No-op |
| 2 | x/staking | lgm_staking | 1 | None | 0 | genesis, state, tx, tx_grpc | Credit with OriginPeriod | No-op |
| 3 | x/tokens | lgm_tokens | 1 | None | 0 | genesis, state, tx, tx_grpc | Sweep expired, RAI | No-op |
| 4 | x/knowledge | lgm_knowledge | 1 | None | 18 | genesis, claim, counter_fact, domain, ibc, query, stratum, tx, tx_grpc | Adversarial, round phases | Expire stale facts |
| 5 | x/emergency | lgm_emergency | 1 | None | 0 | genesis, state, tx, tx_grpc | Ceremony progress | Epoch reset |
| 6 | x/gov | lgm_gov | 1 | None | 2 | genesis, state, tx, tx_grpc | Voting deadline + tally | No-op |
| 7 | x/research | lgm_research | 1 | None | 6 | genesis, state, tx, tx_grpc | Expire bounties | No-op |
| 8 | x/vesting_rewards | lgm_vesting_rewards | 1 | None | 0 | genesis, state, tx, tx_grpc | No-op | No-op |
| 9 | x/ontology | lgm_ontology | 1 | None | 6 | genesis, state | No-op | Compute dynamic ceilings |
| 10 | x/evidence_mgmt | lgm_evidence_mgmt | 1 | None | 0 | genesis, state, tx, tx_grpc | No-op | Reliability scoring |
| 11 | x/qualification | lgm_qualification | 1 | None | 0 | genesis, state | Promote/expire | No-op |
| 12 | x/capture_defense | lgm_capture_defense | 1 | None | 0 | genesis, state | Reputation decay, qualify expiry | Capture detection |
| 13 | x/capture_challenge | lgm_capture_challenge | 1 | None | 0 | genesis, state | Phase advance, expire | Auto-fund bounties |
| 14 | x/billing | lgm_billing | 1 | None | 0 | genesis, state | No-op | Expire quotes |
| 15 | x/bvm | lgm_bvm | 1 | None | 1 | genesis, state, tx, tx_grpc | Execute schedules | No-op |
| 16 | x/home | lgm_home | 1 | None | 3 | genesis, state | Deadman, sessions, treasury, aging | Cleanup alerts |
| 17 | x/schedule | lgm_schedule | 1 | None | 0 | genesis, state | No-op | Process due schedules |
| 18 | x/tree | lgm_tree | 1 | None | 0 | genesis, state, tx, tx_grpc | Expire seeds, tasks | No-op |
| 19 | x/channels | lgm_channels | 1 | None | 1 | genesis, state, tx, tx_grpc | Resolve expired | No-op |
| 20 | x/claiming_pot | lgm_claiming_pot | 1 | None | 0 | genesis, state, tx, tx_grpc | Expire old pots | No-op |
| 21 | x/disputes | lgm_disputes | 1 | None | 0 | genesis, state, tx, tx_grpc | No-op | No-op |
| 22 | x/discovery | lgm_discovery | 1 | None | 0 | genesis, state | No-op | No-op |
| 23 | x/partnerships | lgm_partnerships | 1 | None | 0 | genesis, state | No-op | No-op |
| 24 | x/icaauth | lgm_icaauth | 1 | None | 0 | genesis, state | No-op | No-op |
| 25 | x/ibcratelimit | lgm_ibcratelimit | 1 | None | 5 | genesis, state | Reset flow records | No-op |

---

## Appendix B: Adapter Interface Lock

To prevent accidental interface breakage, add compile-time assertions for all 11 adapters:

```go
// In app/adapters.go — add at the bottom:
var (
    _ knowledgetypes.StakingKeeper          = (*StakingKeeperAdapter)(nil)
    _ lgmgovtypes.StakingKeeper             = (*StakeQueryAdapter)(nil)
    _ vestingtypes.StakingKeeper            = (*VestingStakingAdapter)(nil)
    _ capturedefensetypes.OntologyKeeper    = (*CaptureDefenseOntologyAdapter)(nil)
    _ evidencetypes.KnowledgeKeeper         = (*KnowledgeKeeperForEvidence)(nil)
    _ billingtypes.KnowledgeKeeper          = (*BillingKnowledgeAdapter)(nil)
    _ bvmtypes.KnowledgeKeeper             = (*BVMKnowledgeAdapter)(nil)
    _ bvmtypes.StakingKeeper               = (*BVMStakingAdapter)(nil)
    _ bvmtypes.AuthKeeper                  = (*BVMAuthAdapter)(nil)
    _ disputetypes.KnowledgeKeeper         = (*DisputeKnowledgeAdapter)(nil)
    _ disputetypes.StakingKeeper           = (*DisputeStakingAdapter)(nil)
    _ disputetypes.VestingKeeper           = (*DisputeVestingAdapter)(nil)
    _ lgmgovtypes.UpgradeKeeper            = (*LGMGovUpgradeAdapter)(nil)  // already exists
)
```

---

## Appendix C: Migration Template (Copy-Paste Ready)

For each module that currently has `ConsensusVersion() = 1` and no migration:

```go
// x/<module>/keeper/migrator.go
package keeper

import "context"

// Migrator handles state migrations between module versions.
type Migrator struct {
    keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(k Keeper) Migrator {
    return Migrator{keeper: k}
}

// Migrate1to2 migrates the module state from version 1 to version 2.
// Called by x/upgrade RunMigrations when ConsensusVersion bumps from 1 to 2.
func (m Migrator) Migrate1to2(ctx context.Context) error {
    // TODO: Implement when v2 state changes are needed.
    // Typical operations:
    //   - Re-encode KV values from JSON to proto
    //   - Add new fields with default values
    //   - Rename/restructure keys
    //   - Migrate params from custom KV to x/params
    return nil
}
```

```go
// In x/<module>/module.go RegisterServices():
migrator := keeper.NewMigrator(am.keeper)
if err := cfg.RegisterMigration(types.ModuleName, 1, migrator.Migrate1to2); err != nil {
    panic(fmt.Sprintf("failed to register %s migration: %v", types.ModuleName, err))
}
```
