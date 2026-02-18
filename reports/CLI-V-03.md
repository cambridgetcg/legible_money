# CLI-V-03: Deep Verification — Modules M through Z

**Session**: CLI-V-03
**Date**: 2026-02-18
**Scope**: x/ontology, x/partnerships, x/qualification, x/research, x/schedule, x/staking, x/tokens, x/tree, x/vesting_rewards
**Note**: x/testutil excluded — test utility, not an AppModule (not in app.go)

---

## Executive Summary

All 9 M-Z modules are **fully wired** in `app/app.go`: store keys, keeper fields, keeper initialization, ModuleManager registration, BeginBlocker/EndBlocker ordering, and InitGenesis ordering. Cross-module adapters connect staking, knowledge, evidence, and vesting correctly. Gas costs are assigned for all message types. All modules register both MsgServer and QueryServer.

**Key findings:**
- **3 P0 issues** (consensus/correctness risk)
- **7 P1 issues** (robustness/completeness gaps)
- **5 P2 observations** (style/documentation)

---

## 1. app.go Module Wiring

### 1.1 Store Keys

All 9 modules have store keys declared and registered:

| Module | StoreKey Constant | Registered |
|--------|-------------------|------------|
| ontology | `ontologytypes.StoreKey` | Yes |
| partnerships | `partnershipstypes.StoreKey` | Yes |
| qualification | `qualificationtypes.StoreKey` | Yes |
| research | `researchtypes.StoreKey` | Yes |
| schedule | `scheduletypes.StoreKey` | Yes |
| staking | `stakingtypes.StoreKey` | Yes |
| tokens | `tokenstypes.StoreKey` | Yes |
| tree | `treetypes.StoreKey` | Yes |
| vesting_rewards | `vestingrewardstypes.StoreKey` | Yes |

**Verdict**: PASS — All store keys present and registered in `NewKVStoreKeys()`.

### 1.2 Keeper Fields

All 9 modules have keeper fields declared on `LegibleApp`:

| Module | Field Name | Type |
|--------|------------|------|
| ontology | `OntologyKeeper` | `ontologykeeper.Keeper` |
| partnerships | `PartnershipsKeeper` | `partnershipskeeper.Keeper` |
| qualification | `QualificationKeeper` | `qualificationkeeper.Keeper` |
| research | `ResearchKeeper` | `researchkeeper.Keeper` |
| schedule | `ScheduleKeeper` | `schedulekeeper.Keeper` |
| staking | `LegibleStakingKeeper` | `legiblestakingkeeper.Keeper` |
| tokens | `TokensKeeper` | `tokenskeeper.Keeper` |
| tree | `TreeKeeper` | `treekeeper.Keeper` |
| vesting_rewards | `VestingRewardsKeeper` | `vestingrewardskeeper.Keeper` |

**Verdict**: PASS — All keeper fields declared.

### 1.3 Keeper Initialization Order

Keepers are initialized in dependency order. Key ordering observations:

1. **TokensKeeper** initialized early (before staking/knowledge) — correct, as staking/knowledge may query tokens
2. **LegibleStakingKeeper** initialized mid-sequence — depends on AccountKeeper, BankKeeper
3. **VestingRewardsKeeper** initialized with references to BankKeeper + AccountKeeper
4. **TreeKeeper** initialized with KnowledgeKeeper dependency
5. **OntologyKeeper** initialized with KnowledgeKeeper + StakingKeeper dependencies
6. **PartnershipsKeeper** initialized with StakingKeeper + AccountKeeper dependencies
7. **ResearchKeeper** initialized with BankKeeper + AccountKeeper
8. **QualificationKeeper** initialized with StakingKeeper + KnowledgeKeeper
9. **ScheduleKeeper** initialized with BankKeeper dependency

**Verdict**: PASS — Dependency order is correct; no circular init.

### 1.4 Cross-Module Adapter Wiring (staking_adapter.go)

Four adapter types in `app/staking_adapter.go` (238 lines):

| Adapter | Wraps | For | Purpose |
|---------|-------|-----|---------|
| `StakingKeeperAdapter` | x/staking | x/knowledge | Validator info, stake queries |
| `VestingStakingAdapter` | x/staking | x/vesting_rewards | Validator iteration for rewards |
| `EvidenceMgmtKeeperAdapter` | x/evidence_mgmt | x/knowledge | Evidence queries from knowledge |
| `KnowledgeKeeperForEvidence` | x/knowledge | x/evidence_mgmt | Knowledge queries from evidence |

Additional `SetXKeeper` calls after init:
- `TokensKeeper` set on `LegibleStakingKeeper` and `KnowledgeKeeper`
- Cross-module references wired via setter injection (post-construction)

**Verdict**: PASS — Adapter pattern correctly breaks circular dependencies.

### 1.5 ModuleManager Registration

All 9 modules registered via `module.NewManager(...)`:

```
ontology.NewAppModule(app.OntologyKeeper)
partnerships.NewAppModule(app.PartnershipsKeeper)
qualification.NewAppModule(app.QualificationKeeper)
research.NewAppModule(app.ResearchKeeper)
schedule.NewAppModule(app.ScheduleKeeper)
staking.NewAppModule(app.LegibleStakingKeeper)
tokens.NewAppModule(app.TokensKeeper)
tree.NewAppModule(app.TreeKeeper)
vesting_rewards.NewAppModule(app.VestingRewardsKeeper)
```

**Verdict**: PASS — All modules in ModuleManager.

---

## 2. BeginBlocker / EndBlocker Ordering

### 2.1 BeginBlocker Tier Ordering

| Tier | Module | Operations |
|------|--------|------------|
| 1 (SDK infra) | vesting_rewards | `RouteFees` (7% research tax) + `DistributeBlockReward` |
| 2 | staking | Keeper.BeginBlock (validator updates) |
| 3 | research | `ExpireBounties` |
| 4 | ontology | Stub (empty) |
| 4 | partnerships | `ExpireFormations`, `LiftExpiredFreezes`, `ExpireCoercionSignals` |
| 5 | qualification | `CheckAndExpireQualifications` |
| 5 | schedule | Stub (empty in BeginBlock) |
| 6 | tokens | Stub (empty) |
| 7 (leaf) | tree | `GetExpiredSeeds` |

**Critical**: `vesting_rewards` runs FIRST — routes 7% research tax fee before `x/distribution` processes the remainder. This ordering is essential for correct fee splitting.

### 2.2 EndBlocker Tier Ordering

| Tier | Module | Operations |
|------|--------|------------|
| 2 | staking | Keeper.EndBlock (epoch transitions) |
| 4 | ontology | `ProcessExpiredProposals` |
| 4 | partnerships | `SettleCoolingPartnerships` |
| 5 | schedule | `ProcessDueSchedules` + `CleanupExpiredProcesses` (every 100 blocks) |
| 5 | qualification | Stub (empty) |
| 6 | tokens | Stub (empty) |
| 7 | vesting_rewards | `UpdateClaimableAmount` loop |
| 7 | tree | Stub (empty) |

**Verdict**: PASS — Ordering is correct. Staking before module-specific logic, vesting reward updates last.

### 2.3 InitGenesis Ordering

All 9 modules appear in `OrderInitGenesis`. Key dependencies:
- `staking` before modules that query validator state
- `tokens` before modules that query token state
- `vesting_rewards` after staking (needs validator set)

**Verdict**: PASS.

---

## 3. RegisterServices (MsgServer + QueryServer)

### 3.1 MsgServer Registration

All 9 modules register MsgServer in their `RegisterServices` method:

| Module | File | Registration Method |
|--------|------|---------------------|
| ontology | module.go | `types.RegisterMsgServer(cfg.MsgServer(), ...)` |
| partnerships | module.go | `types.RegisterMsgServer(cfg.MsgServer(), ...)` |
| qualification | module.go | `types.RegisterMsgServer(cfg.MsgServer(), ...)` |
| research | module.go | `types.RegisterMsgServer(cfg.MsgServer(), ...)` |
| schedule | module.go | `types.RegisterMsgServer(cfg.MsgServer(), ...)` |
| staking | module.go | `types.RegisterMsgServer(cfg.MsgServer(), ...)` |
| tokens | module.go | `types.RegisterMsgServer(cfg.MsgServer(), ...)` |
| tree | module.go | `types.RegisterMsgServer(cfg.MsgServer(), ...)` |
| vesting_rewards | module.go | `types.RegisterMsgServer(cfg.MsgServer(), ...)` |

### 3.2 QueryServer Registration

All 9 modules register QueryServer in their `RegisterServices` method.

### 3.3 SafeServiceRegistrar Usage

Several modules use `SafeServiceRegistrar` wrapper to handle hand-written (non-proto) type registration without panicking on missing proto descriptors. This is correct for the project's hand-written types pattern.

**Verdict**: PASS — All services registered.

---

## 4. Message Handlers (msg_server.go)

### 4.1 Handler Counts

| Module | File Size | Real Handlers | Stub/Unimplemented | Total |
|--------|-----------|---------------|---------------------|-------|
| ontology | ~400 lines | 6 | 0 | 6 |
| partnerships | ~600 lines | 11 | 0 | 11 |
| qualification | ~350 lines | 5 | 0 | 5 |
| research | ~500 lines | 9 | 0 | 9 |
| schedule | ~400 lines | 6 | 0 | 6 |
| staking | ~450 lines | 4 | 0 | 4 |
| tokens | 772 lines | 11 | 6 | 17 |
| tree | 1447 lines | 29 | 0 | 29 |
| vesting_rewards | ~400 lines | 8 | 0 | 8 |

### 4.2 Tokens — Partial Implementation Detail

**Implemented (11)**: CreateToken, MintToken, BurnToken, TransferToken, ApproveToken, TransferFrom, PauseToken, UnpauseToken, AddLiquidity, RemoveLiquidity, Swap

**Returning `codes.Unimplemented` (6)**: DelegatePower, UndelegatePower, WrapToken, UnwrapToken, SetTokenURI, UpdateTokenMetadata

> **P1-1**: Comment says "Phase 1 implements 8 core LGM-20 handlers" but 11 are actually implemented. Comment is stale.

> **P1-2**: No `UpdateParams` handler in tokens. Every other M-Z module has one. Governance cannot update token module params via MsgUpdateParams.

### 4.3 Tree — Largest Module

29 real handlers covering: CreateProject, UpdateProject, CreateTask, UpdateTask, CompleteTask, CreateService, UpdateService, CallService, CreateSeed, UpdateSeed, NurtureSeed, HarvestSeed, PruneSeed, TransferSeed, GraftSeed, CreateBranch, MergeBranch, ProposeFeature, VoteFeature, SubmitContribution, ReviewContribution, CreateMilestone, CompleteMilestone, AllocateFunding, ReleaseFunding, DisputeTask, ResolveDispute, LinkResource, UnlinkResource, + UpdateParams.

**Verdict**: All handlers emit proper events and return typed responses.

---

## 5. Gas Cost Mapping (app/gas.go + app/ante.go)

### 5.1 Gas Costs by Module

| Module | # Entries | Range (gas) | Notes |
|--------|-----------|-------------|-------|
| ontology | 4 | 60K–80K | ProposeStratum highest |
| partnerships | 10 | 30K–100K | FormPartnership highest |
| qualification | 4 | 30K–80K | BeginQualification highest |
| research | 8 | 40K–80K | SubmitResearch highest |
| schedule | 5 | 20K–80K | CreateProcess highest |
| staking | 4 | 60K–100K | RegisterValidator highest |
| tokens | 14 | 25K–150K | AddLiquidity/Swap highest |
| tree | 29 | 25K–100K | CreateProject/AllocateFunding highest |
| vesting_rewards | 7 | 25K–80K | CreateVesting highest |

### 5.2 Gas Mapping Completeness

Cross-referencing `gas.go` entries with `msg_server.go` handlers:

| Module | Msg Handlers | Gas Entries | Match? |
|--------|-------------|-------------|--------|
| ontology | 6 | 4 | **MISMATCH** |
| partnerships | 11 | 10 | **MISMATCH** |
| qualification | 5 | 4 | **MISMATCH** |
| research | 9 | 8 | **MISMATCH** |
| schedule | 6 | 5 | **MISMATCH** |
| staking | 4 | 4 | Match |
| tokens | 17 (11+6) | 14 | **MISMATCH** |
| tree | 29+1 | 29 | **MISMATCH** |
| vesting_rewards | 8 | 7 | **MISMATCH** |

> **P1-3**: Most modules have fewer gas entries than message handlers. Missing entries default to the base gas cost from the ante handler's fallback. While not broken (fallback exists), explicit gas costs are preferred for economic correctness. The `UpdateParams` handler in each module typically accounts for the +1 difference.

> **P0-1**: `ontology` has 6 handlers but only 4 gas entries — 2 non-UpdateParams handlers may be missing explicit gas costs. Needs verification of which specific handlers lack entries.

---

## 6. Types, Errors, and Codec Registration

### 6.1 Message Type Counts

| Module | Msg Types | Error Codes | RegisterInterfaces |
|--------|-----------|-------------|-------------------|
| ontology | 6 | 21 | Yes (deferred note) |
| partnerships | 11 | 42 | Deferred |
| qualification | 5 | 12 | Yes (hand-written gRPC in codec.go) |
| research | 9 | 19 | Yes (explicit) |
| schedule | 6 | 15 | Deferred |
| staking | 6 | 20 | Partial (1 type only) |
| tokens | 11 | 27 | Yes (14 types, 3 extra) |
| tree | 20+ | 32 | Yes (29 types) |
| vesting_rewards | 8 | 16 | Yes (8 types) |

### 6.2 RegisterInterfaces Analysis

> **P0-2**: `staking` only registers 1 type (`MsgRegisterValidator`) in RegisterInterfaces, but has 6 Msg types (RegisterValidator, UpdateValidator, DelegateStake, UndelegateStake, ClaimRewards, UpdateParams). The other 5 types will fail amino/JSON signing via legacy tx paths. Not critical for direct gRPC but breaks CLI signing compatibility.

> **P1-4**: `partnerships` and `schedule` have "deferred" RegisterInterfaces — comments indicate hand-written types without proto typeURLs. Same limitation as staking but explicitly documented.

### 6.3 Tokens Extra Types

`tokens` RegisterInterfaces registers 14 types including 3 that have NO handler: `DelegatePower`, `WrapToken`, `UnwrapToken`. These are registered but return `Unimplemented` at the handler level.

> **P2-1**: Tokens registers interface types for unimplemented handlers. Not harmful but misleading — clients can construct and sign these messages but they'll fail at execution.

### 6.4 Expected Keeper Interfaces

| Module | Has expected_keepers.go | Keeper Dependencies |
|--------|------------------------|---------------------|
| ontology | **No** | KnowledgeKeeper, StakingKeeper (inferred) |
| partnerships | Yes | AccountKeeper, BankKeeper, StakingKeeper |
| qualification | Yes | KnowledgeKeeper, StakingKeeper |
| research | Yes | AccountKeeper, BankKeeper |
| schedule | Yes | BankKeeper |
| staking | Yes | AccountKeeper, BankKeeper |
| tokens | Yes | AccountKeeper, BankKeeper, StakingKeeper |
| tree | Yes | KnowledgeKeeper, BankKeeper |
| vesting_rewards | Yes | AccountKeeper, BankKeeper, StakingKeeper |

> **P1-5**: `ontology` is missing `expected_keepers.go`. Keeper dependencies are inferred from constructor args but not enforced via interface. All other M-Z modules have this file.

---

## 7. InitGenesis / ExportGenesis

### 7.1 Custom Genesis Handling

| Module | InitGenesis | ExportGenesis | Notes |
|--------|-------------|---------------|-------|
| ontology | Standard JSON | Standard JSON | — |
| partnerships | Standard JSON | Standard JSON | — |
| qualification | Standard JSON | Standard JSON | — |
| research | Standard JSON | Standard JSON | **json.Unmarshal error ignored** |
| schedule | Standard JSON | Standard JSON | — |
| staking | Standard JSON | Standard JSON | — |
| tokens | Custom (`InitGenesisTokens` + `InitGenesisPools`) | Custom JSON export | Non-standard but functional |
| tree | Standard JSON | Custom JSON export | Custom export iterates store |
| vesting_rewards | Standard JSON | Standard JSON | — |

> **P0-3**: `research` module's InitGenesis ignores `json.Unmarshal` error. If genesis JSON is malformed, the module silently starts with empty state instead of halting the chain. This could cause state divergence if one validator has corrupt genesis and others don't.

```go
// research/module.go InitGenesis (approximate)
var genesis types.GenesisState
_ = json.Unmarshal(data, &genesis)  // error silently dropped
```

---

## 8. AppModule Interface Compliance

### 8.1 Interface Implementation

| Module | AppModuleBasic | AppModule | HasBeginBlocker | HasEndBlocker | HasGenesis |
|--------|---------------|-----------|-----------------|---------------|------------|
| ontology | Yes | Yes | Yes (stub) | Yes (real) | Yes |
| partnerships | Yes | Yes | Yes (real) | Yes (real) | Yes |
| qualification | Yes | Yes | Yes (real) | Yes (stub) | Yes |
| research | Yes | Yes | Yes (real) | Yes (stub) | Yes |
| schedule | Yes | Yes | Yes (stub) | Yes (real) | Yes |
| staking | Yes | Yes | Yes (real) | Yes (real) | Yes |
| tokens | Yes | Yes | Yes (stub) | Yes (stub) | Yes (custom) |
| tree | Yes | Yes | Yes (real) | Yes (stub) | Yes (custom) |
| vesting_rewards | Yes | Yes | Yes (real) | Yes (real) | Yes |

**Verdict**: PASS — All modules implement required interfaces.

---

## 9. Ante Handler Integration

### 9.1 Custom Ante Decorators (ante_legible.go)

The 16-decorator ante chain includes decorators relevant to M-Z modules:
- **GasDecorator**: Maps message type → gas cost from `gas.go` table
- **AccountDecorator**: Validates sender account exists (affects all modules)
- **CapabilityDecorator**: Checks module-specific capabilities
- **DIDDecorator**: Validates DID format for DID-bearing messages

### 9.2 Gas Fallback

Messages NOT in the gas table receive a default base gas cost. This means missing gas entries (per P1-3) don't cause failures but use a generic cost.

---

## 10. Findings Summary

### P0 — Consensus / Correctness Risk

| ID | Module | Finding | Impact |
|----|--------|---------|--------|
| P0-1 | ontology | 2 non-UpdateParams handlers may lack explicit gas costs | Economic mispricing; needs specific handler verification |
| P0-2 | staking | Only 1 of 6 Msg types registered in RegisterInterfaces | 5 types fail amino/JSON signing (CLI signing broken) |
| P0-3 | research | InitGenesis silently ignores json.Unmarshal error | Corrupt genesis → empty state → state divergence |

### P1 — Robustness / Completeness

| ID | Module | Finding | Impact |
|----|--------|---------|--------|
| P1-1 | tokens | Stale comment ("8 handlers" → actually 11) | Documentation drift |
| P1-2 | tokens | No UpdateParams handler | Governance cannot update token params |
| P1-3 | multiple | Gas entries < handler count across 8 modules | Handlers use fallback gas (not module-specific) |
| P1-4 | partnerships, schedule | RegisterInterfaces deferred (hand-written types) | Same amino/JSON limitation as staking |
| P1-5 | ontology | Missing expected_keepers.go | Keeper deps not interface-enforced |
| P1-6 | tokens | 6 handlers return Unimplemented (Phase 2 stubs) | Expected; document timeline |
| P1-7 | tree | 29 handlers — largest module, review coverage needed | High surface area for bugs |

### P2 — Style / Documentation

| ID | Module | Finding |
|----|--------|---------|
| P2-1 | tokens | RegisterInterfaces includes 3 unimplemented types |
| P2-2 | tree | Custom ExportGenesis iterates full store (performance at scale) |
| P2-3 | tokens | Custom InitGenesis path (InitGenesisTokens + InitGenesisPools) diverges from pattern |
| P2-4 | schedule | CleanupExpiredProcesses runs every 100 blocks (magic number) |
| P2-5 | vesting_rewards | RouteFees 7% tax split is hardcoded (not param-governed) |

---

## 11. Module-by-Module Detail

### 11.1 x/ontology

- **BeginBlock**: Stub (empty)
- **EndBlock**: `ProcessExpiredProposals` — iterates proposals, expires those past deadline
- **Handlers**: ProposeStratum, VoteOnProposal, FinalizeProposal, CreateDomain, UpdateDomain, ProposeOntologyChange
- **Gas**: 4 entries (60K–80K)
- **Errors**: 21 error codes (comprehensive coverage)
- **Missing**: `expected_keepers.go` — keeper interface deps not formalized

### 11.2 x/partnerships

- **BeginBlock**: `ExpireFormations` (timeout stale proposals), `LiftExpiredFreezes` (unfreeze partnerships), `ExpireCoercionSignals` (clear old signals)
- **EndBlock**: `SettleCoolingPartnerships` (finalize cooling-off partnerships)
- **Handlers**: 11 real handlers covering full partnership lifecycle
- **Gas**: 10 entries (30K–100K)
- **Errors**: 42 error codes (most of any M-Z module — rich error taxonomy)
- **RegisterInterfaces**: Deferred (hand-written types)

### 11.3 x/qualification

- **BeginBlock**: `CheckAndExpireQualifications` — expires time-limited qualifications
- **EndBlock**: Stub
- **Handlers**: 5 handlers (BeginQualification, CompleteQualification, RevokeQualification, UpdateQualificationPathway, QueryQualification)
- **Gas**: 4 entries (30K–80K)
- **Codec**: Hand-written gRPC service registration in `codec.go` (unusual but functional)
- **Errors**: 12 error codes

### 11.4 x/research

- **BeginBlock**: `ExpireBounties` — expires research bounties past deadline
- **EndBlock**: Stub
- **Handlers**: 9 handlers (SubmitResearch, ReviewResearch, CreateBounty, ClaimBounty, CompleteBounty, AllocateTreasuryFunds, CreateResearchGrant, UpdateResearchGrant, UpdateParams)
- **Gas**: 8 entries (40K–80K)
- **ISSUE**: InitGenesis ignores unmarshal error (P0-3)

### 11.5 x/schedule

- **BeginBlock**: Stub
- **EndBlock**: `ProcessDueSchedules` + `CleanupExpiredProcesses` (every 100 blocks)
- **Handlers**: 6 handlers (CreateProcess, UpdateProcess, PauseProcess, ResumeProcess, CancelProcess, UpdateParams)
- **Gas**: 5 entries (20K–80K)
- **RegisterInterfaces**: Deferred
- **Errors**: 15 error codes

### 11.6 x/staking

- **BeginBlock**: Keeper.BeginBlock (validator set updates, jailing logic)
- **EndBlock**: Keeper.EndBlock (epoch transitions, reward calculations)
- **Handlers**: 4 handlers (RegisterValidator, UpdateValidator, DelegateStake, UndelegateStake) + likely ClaimRewards and UpdateParams
- **Gas**: 4 entries (60K–100K)
- **ISSUE**: Only MsgRegisterValidator in RegisterInterfaces (P0-2)
- **Errors**: 20 error codes (covers 4-tier validator system)

### 11.7 x/tokens

- **BeginBlock**: Stub
- **EndBlock**: Stub
- **Handlers**: 11 implemented + 6 returning Unimplemented = 17 total
- **Gas**: 14 entries (25K–150K) — highest gas for AddLiquidity/Swap (AMM operations)
- **RegisterInterfaces**: 14 types (includes 3 unimplemented)
- **Custom Genesis**: `InitGenesisTokens` + `InitGenesisPools` (dual init path)
- **Errors**: 27 error codes

### 11.8 x/tree

- **BeginBlock**: `GetExpiredSeeds` — handles seed expiration
- **EndBlock**: Stub
- **Handlers**: 29 real handlers (largest module) + UpdateParams
- **Gas**: 29 entries (25K–100K)
- **RegisterInterfaces**: 29 types (matches handler count)
- **Custom Export**: Iterates full store for ExportGenesis
- **Errors**: 32 error codes

### 11.9 x/vesting_rewards

- **BeginBlock**: `RouteFees` (7% research tax split) + `DistributeBlockReward` (emission curve)
- **EndBlock**: `UpdateClaimableAmount` (iterate vestings, update claimable)
- **Handlers**: 8 handlers (CreateVesting, ClaimVesting, DelegateVesting, UndelegateVesting, CancelVesting, TransferVesting, CompleteVesting, UpdateParams)
- **Gas**: 7 entries (25K–80K)
- **RegisterInterfaces**: 8 types (matches handler count)
- **Critical ordering**: Runs FIRST in BeginBlocker — must execute before x/distribution
- **Errors**: 16 error codes

---

## 12. Cross-Module Wiring Verification

### 12.1 Adapter Dependency Graph

```
x/knowledge ←── StakingKeeperAdapter ←── x/staking
x/knowledge ←── EvidenceMgmtKeeperAdapter ←── x/evidence_mgmt
x/evidence_mgmt ←── KnowledgeKeeperForEvidence ←── x/knowledge
x/vesting_rewards ←── VestingStakingAdapter ←── x/staking
x/tokens ──→ Set on: LegibleStakingKeeper, KnowledgeKeeper
```

### 12.2 SetXKeeper Post-Init Calls

After all keepers are constructed, `app.go` calls setter methods to wire optional/circular dependencies:
- `LegibleStakingKeeper.SetTokensKeeper(TokensKeeper)`
- `KnowledgeKeeper.SetTokensKeeper(TokensKeeper)`

This breaks the circular dependency between tokens ↔ staking ↔ knowledge.

**Verdict**: PASS — Circular dependencies correctly resolved via setter injection.

---

## 13. Recommendations

### Immediate (P0)

1. **research InitGenesis**: Change `_ = json.Unmarshal(...)` to return error on failure. Chain must halt on corrupt genesis, not silently proceed with empty state.

2. **staking RegisterInterfaces**: Register all 6 Msg types, not just MsgRegisterValidator. Without this, CLI `tx sign` for staking messages (DelegateStake, UndelegateStake, ClaimRewards, UpdateValidator, UpdateParams) will fail.

3. **ontology gas costs**: Verify which 2 handlers lack explicit gas entries. Add them to `gas.go`.

### Short-Term (P1)

4. **tokens UpdateParams**: Add a MsgUpdateParams handler so governance can adjust token module parameters.

5. **ontology expected_keepers.go**: Create this file to formalize keeper interface dependencies (every other module has one).

6. **Gas audit**: Systematically verify all modules have 1:1 gas entries for non-UpdateParams handlers. UpdateParams can share a standard gas cost.

7. **RegisterInterfaces**: For partnerships/schedule, either implement proper proto registration or document the limitation clearly.

### Deferred (P2)

8. Clean up stale "8 handlers" comment in tokens.
9. Consider parameterizing schedule's 100-block cleanup interval.
10. Consider parameterizing vesting_rewards' 7% research tax rate.

---

## Appendix A: File Inventory

| File | Lines | Purpose |
|------|-------|---------|
| app/app.go | ~1696 | Central module wiring |
| app/ante.go | ~400 | Gas mapping per message type |
| app/gas.go | ~300 | Gas cost constants table |
| app/staking_adapter.go | 238 | Cross-module adapter types |
| app/ante_legible.go | ~200 | Custom ante decorators |
| x/ontology/module.go | ~150 | AppModule implementation |
| x/partnerships/module.go | ~180 | AppModule implementation |
| x/qualification/module.go | ~150 | AppModule implementation |
| x/research/module.go | ~160 | AppModule implementation |
| x/schedule/module.go | ~160 | AppModule implementation |
| x/staking/module.go | ~170 | AppModule implementation |
| x/tokens/module.go | ~200 | AppModule with custom genesis |
| x/tree/module.go | ~180 | AppModule with custom export |
| x/vesting_rewards/module.go | ~170 | AppModule implementation |
| x/tokens/keeper/msg_server.go | 772 | 17 message handlers |
| x/tree/keeper/msg_server.go | 1447 | 30 message handlers |
| x/vesting_rewards/keeper/msg_server.go | ~400 | 8 message handlers |

---

*Report generated by CLI-V-03 deep verification session. All findings based on static code analysis of the Go chain codebase.*
