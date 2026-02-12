# PP Review: Cosmos SDK x/params Module in Legible Money

> **Review Date**: 2026-02-09
> **Verdict**: KEEP (for now) — deprecated but structurally entangled; removal requires coordinated refactor

---

## 1. What x/params Does

The x/params module provides a **centralized, namespaced parameter store** with isolated
Subspaces for each module. Key design:

- **Keeper**: Master controller that creates/retrieves Subspaces
- **Subspace**: Isolated key-value namespace per module (e.g. `auth/MaxMemoCharacters`)
- **Security**: Modules receive their own Subspace only; x/gov receives the full Keeper
- **ParamChangeProposal**: Legacy governance mechanism to change any module's params

---

## 2. Deprecation Status

| Version | Status |
|---------|--------|
| v0.47 | Initially deprecated; all core SDK modules migrated to MsgUpdateParams |
| v0.50 | Deprecated; x/consensus replaces consensus params |
| v0.53 | Deprecated with removal notice — **will be removed in next release** |

**Replacement**: ADR-046 — each module self-manages params via `MsgUpdateParams` message
handled through standard governance. All core SDK modules (auth, bank, staking, distr,
slashing, gov) completed migration by v0.47.

---

## 3. Current Wiring in Legible Money

### Direct References (app/app.go)

| Location | Line(s) | What |
|----------|---------|------|
| Imports | 67-71 | `params`, `paramsclient`, `paramskeeper`, `paramstypes`, `paramproposal` |
| ModuleBasics | 209 | `paramsclient.ProposalHandler` in gov's proposal handlers |
| ModuleBasics | 212 | `params.AppModuleBasic{}` |
| Struct field | 307 | `ParamsKeeper paramskeeper.Keeper` |
| Store key (KV) | 397 | `paramstypes.StoreKey` |
| Store key (Transient) | 426 | `paramstypes.TStoreKey` |
| Keeper init | 442 | `initParamsKeeper(...)` |
| IBC keeper | 545 | `app.GetSubspace(ibcexported.ModuleName)` |
| IBC transfer keeper | 555 | `app.GetSubspace(ibctransfertypes.ModuleName)` |
| Module manager (6x) | 768-775 | `app.GetSubspace(...)` for auth, bank, gov, slashing, distr, staking |
| Module manager | 778 | `params.NewAppModule(app.ParamsKeeper)` |
| Gov legacy router | 752 | `params.NewParamChangeProposalHandler(app.ParamsKeeper)` |
| GetSubspace helper | 1008-1012 | `app.GetSubspace()` method |
| initParamsKeeper | 1116-1132 | 10 Subspace registrations |

### Test References (app/store_test.go)

| Line(s) | What |
|---------|------|
| 36, 433 | `"params"` in store key arrays |
| 44, 442 | `"params_tstore"` in transient store key arrays |

### Subspaces Registered

| Module | Used By |
|--------|---------|
| auth | `auth.NewAppModule(...)` |
| bank | `bank.NewAppModule(...)` |
| staking | `staking.NewAppModule(...)` |
| distribution | `distr.NewAppModule(...)` |
| slashing | `slashing.NewAppModule(...)` |
| gov | `gov.NewAppModule(...)` |
| ibc | `ibckeeper.NewKeeper(...)` |
| ibc-transfer | `ibctransferkeeper.NewKeeper(...)` |
| knowledge | Registered but **not consumed** (custom module uses MsgUpdateParams) |
| legible_staking | Registered but **not consumed** (custom module uses MsgUpdateParams) |

---

## 4. Cross-Module Interaction Map

```
                    ┌────────────────────┐
                    │   x/params Keeper  │
                    │   (ParamsKeeper)   │
                    └────────┬───────────┘
                             │
              ┌──────────────┼──────────────────┐
              │              │                  │
              ▼              ▼                  ▼
    ┌─────────────┐  ┌──────────────┐  ┌──────────────┐
    │ GetSubspace │  │  GetSubspace │  │ ParamChange  │
    │ (SDK mods)  │  │ (IBC mods)   │  │ Proposal     │
    ├─────────────┤  ├──────────────┤  │ Handler      │
    │ auth        │  │ ibc          │  │ (legacy gov) │
    │ bank        │  │ ibc-transfer │  └──────────────┘
    │ staking     │  └──────────────┘
    │ distr       │
    │ slashing    │
    │ gov         │
    └─────────────┘

Custom LGM modules: Do NOT use x/params
(knowledge, legible_staking, tokens, research, etc. all use MsgUpdateParams)
```

---

## 5. Why It Can't Be Simply Removed (Unlike mint/crisis)

### 5.1 SDK Module Constructor Signatures

Standard SDK modules in v0.50 still accept a `paramstypes.Subspace` parameter in their
`NewAppModule()` constructors. While these modules have migrated internally to
MsgUpdateParams, the constructor signature remains for backward compatibility:

```go
// auth.NewAppModule signature includes subspace parameter
auth.NewAppModule(appCodec, app.AccountKeeper, nil, app.GetSubspace(authtypes.ModuleName))
//                                                     ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
```

Passing an empty/zero-value `paramstypes.Subspace{}` may work since modules don't read
from it anymore, but this is **untested in LGM** and could cause panics during:
- Legacy proposal processing
- Module migration hooks
- Store iteration

### 5.2 IBC Keeper Dependency

Both IBC keepers require a Subspace:
```go
ibckeeper.NewKeeper(..., app.GetSubspace(ibcexported.ModuleName), ...)
ibctransferkeeper.NewKeeper(..., app.GetSubspace(ibctransfertypes.ModuleName), ...)
```

IBC-go v8 may still read from the Subspace for certain parameters. This needs
verification before removal.

### 5.3 Legacy Gov Router

The `ParamChangeProposal` handler is wired into the legacy governance router:
```go
govRouter := govv1beta1.NewRouter()
govRouter.
    AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
    AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper))
```

LGM's custom x/gov module uses MsgUpdateParams, but the standard x/gov module's legacy
router still exists. Removing params means removing this route — which is fine for a new
chain (no existing proposals), but the router setup may need adjustment.

### 5.4 The `paramsclient.ProposalHandler` in Gov Basic

```go
gov.NewAppModuleBasic(
    []govclient.ProposalHandler{
        paramsclient.ProposalHandler,
    },
),
```

This registers the CLI command for legacy param-change proposals. Removing it changes
the gov module's CLI behavior.

---

## 6. Comparison: What LGM Custom Modules Do Instead

All 11+ custom LGM modules already follow the modern pattern (ADR-046):

```
proto:   MsgUpdateParams { authority, params }
keeper:  func (k msgServer) UpdateParams(ctx, msg) { k.SetParams(ctx, msg.Params) }
auth:    authority == govModuleAddress
storage: module's own KVStore
```

Modules with MsgUpdateParams: knowledge, research, tokens, evidence_mgmt, qualification,
lgm_gov, ontology, legible_staking, tree, disputes, vesting_rewards, emergency, bvm,
channels, claiming_pot, home, capture_defense, capture_challenge.

The two custom Subspace registrations (`knowledge`, `legible_staking`) in `initParamsKeeper`
are **dead code** — registered but never consumed by those modules.

---

## 7. Recommendation: Staged Removal

### Phase 1: NOW — Clean up dead code
- Remove the 2 unused custom Subspace registrations (knowledge, legible_staking)
- Document that x/params is legacy-only

### Phase 2: NEXT SESSION — Full removal (requires testing)
- Replace `app.GetSubspace(...)` calls with `paramstypes.Subspace{}` (empty value)
- Remove `ParamChangeProposal` handler from gov legacy router
- Remove `paramsclient.ProposalHandler` from gov basic
- Remove `params.AppModuleBasic{}` and `params.NewAppModule(...)`
- Remove `ParamsKeeper` field, store keys, imports
- Remove `initParamsKeeper()` helper and `GetSubspace()` method
- **Test**: Run full test suite + devnet to verify no panics

### Why Staged?
Unlike mint (zero consumers) and crisis (broken + replaced by emergency), params is
**structurally entangled** with 8 SDK module constructors and 2 IBC keepers. A new chain
like LGM doesn't need any of the legacy param infrastructure, but the SDK module
constructors still accept the parameter. Safe removal requires verification that
empty subspaces don't cause runtime panics in SDK module initialization or IBC.

---

## 8. Risk Assessment

| Action | Risk | Mitigation |
|--------|------|------------|
| Remove dead custom subspaces | None | Subspaces registered but never read |
| Pass empty subspace to SDK modules | Low-Medium | SDK modules migrated in v0.47, but untested in LGM |
| Remove ParamChangeProposal | None | New chain, no existing proposals |
| Remove IBC subspaces | Medium | IBC-go v8 may still read from subspace |
| Remove entire params module | Low (after testing) | Subtractive change, but needs full test pass |

---

## 9. Key Insight: New Chain Advantage

LGM has never run a pre-v0.47 version. There is no legacy parameter state to migrate.
The entire x/params Subspace system exists only because SDK module constructors still
accept the parameter. For a **greenfield chain**, x/params provides zero value:

- No legacy params to read from Subspace stores
- No legacy ParamChangeProposals in governance history
- All custom modules already use MsgUpdateParams
- x/consensus already handles consensus params

The only question is whether SDK v0.50 module constructors tolerate empty subspaces.
