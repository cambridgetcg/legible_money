# PP Review: Interchain Account Structure

> **Date**: 2026-02-10
> **Scope**: Full PP review of interchain account (ICA) architecture
> **Dimensions**: TRUTH → UNDERSTANDING → BEAUTY → JUSTICE → CREATIVITY
> **Territory**: UNCHARTED — no ICA infrastructure exists; unique agent identity creates novel design space

---

## Executive Summary

**Legible Money has zero Interchain Accounts (ICA) infrastructure.** The chain has basic IBC (core + transfer) wired in `app/app.go`, but no ICA controller, host, or authentication modules. The rich 4-layer key architecture and DID-based agent identity system (`x/auth`) creates a novel design challenge: standard Cosmos ICA assumes single-key accounts, but Legible agents have identity/recovery/operational/session key layers with capability restrictions.

This review maps the gap, identifies 8 findings across 5 PP dimensions, and proposes an architecture that extends — rather than contradicts — the existing agent identity system.

---

## TRUTH: What Exists (Grounded in Code)

### IBC Infrastructure Present

| Component | Status | Location |
|-----------|--------|----------|
| IBC Core | Wired | `app/app.go:561-568` — `ibckeeper.NewKeeper()` |
| IBC Transfer | Wired | `app/app.go:571-582` — `ibctransferkeeper.NewKeeper()` |
| Capability Module | Wired | `app/app.go:551-557` — scoped to IBC + Transfer |
| IBC Router | Configured | `app/app.go:807-811` — only Transfer route |
| Knowledge Oracle IBC | Proto only | `proto/legible/knowledge/v1/ibc.proto` — packet types defined, not implemented |

### ICA Infrastructure Absent

| Component | Status | Evidence |
|-----------|--------|----------|
| ICA Controller | Missing | No `icacontroller` import in `app/app.go` |
| ICA Host | Missing | No `icahost` import or keeper |
| ICA Auth | Missing | No `icaauth` module in `x/` |
| ICA Store Keys | Missing | Not in `storetypes.NewKVStoreKeys()` (`app/app.go:408-446`) |
| ICA Module Registration | Missing | Not in `module.NewManager()` (`app/app.go:815-854`) |
| Scoped ICA Keeper | Missing | Only `ScopedIBCKeeper` and `ScopedTransferKeeper` exist (`app/app.go:556-557`) |

### Account System That Must Be Bridged

The custom `x/auth` module (`x/auth/keeper/keeper.go`, `x/auth/types/types.go`) implements:

```
┌──────────────────────────────────────────────────────┐
│                LegibleAccount                         │
│                                                       │
│  Layer 1: Identity Anchor                             │
│    did: "did:lgm:{32-hex}"  (immutable)              │
│                                                       │
│  Layer 2: Recovery                                    │
│    RecoveryConfig → Shamir shards → ShardHolder[]    │
│    RecoveryRequest → delayed → challengeable → exec  │
│                                                       │
│  Layer 3: Operational                                 │
│    operationalKeyHash: string (rotatable)            │
│    operationalKeyVersion: uint32                     │
│    rotation cooldown: 111 blocks                     │
│                                                       │
│  Layer 4: Session Keys                                │
│    SessionKey[] (max 5, ephemeral, block-expiry)     │
│    SessionCapabilities:                               │
│      canTransfer, canStake, canSubmitClaims, canVote │
│    Enforced by: LegibleCapabilityDecorator           │
│      (ante_legible.go:73-187, default-deny)          │
│                                                       │
│  AccountType: "agent" | "human" | "contract"          │
└──────────────────────────────────────────────────────┘
```

**Key tension**: Standard ICA uses a single-key controller account to send messages to a host chain. Legible's 4-layer architecture means the question "which key controls the interchain account?" has no trivial answer.

---

## UNDERSTANDING: Structural Analysis

### Finding 1 (CRITICAL): No ICA Wiring — Agents Cannot Act Cross-Chain

**Impact**: AI agents on Legible Money cannot control accounts on other Cosmos chains, and agents on other chains cannot execute transactions on Legible Money. For an "AI agent economy" protocol, this is a foundational gap — agents are economically isolated to a single chain.

**What's needed (minimum viable)**:
1. ICA Controller — Legible agents control accounts on remote chains
2. ICA Host — remote chains can execute transactions on Legible
3. ICA Authentication — maps Legible's 4-layer keys to ICA permissions

**Cosmos SDK provides** `ibc-go/v8/modules/apps/27-interchain-accounts/` with both controller and host sub-modules. These are standard and battle-tested.

### Finding 2 (HIGH): No DID ↔ ICA Mapping

**Impact**: Even if ICA were wired, there's no mechanism to associate a `did:lgm:{hex}` identity with an interchain account on a remote chain. The DID system (`x/auth/keeper/keeper.go:89-132`) only maps DIDs to local bech32 addresses.

**Gap**: Need a `DID → []InterchainAccountAddress` mapping table and reverse lookup, so agents can be identified across chains by their DID.

### Finding 3 (HIGH): Session Key Capabilities Don't Extend to ICA

**Impact**: The `LegibleCapabilityDecorator` (`app/ante_legible.go:157-187`) enforces capability restrictions on session keys, but only recognizes local message types (`isTransferMsg`, `isStakeMsg`, `isClaimMsg`, `isVoteMsg`). ICA messages (`MsgSendTx`, `MsgRegisterInterchainAccount`) would hit the default-deny case:

```go
// ante_legible.go:183-185
default:
    return legibleauthtypes.ErrSessionCapabilityDenied
```

An agent's session key cannot currently send any ICA transaction, even with a valid session.

### Finding 4 (MEDIUM): Knowledge Oracle IBC Has No ICA Dependency, But Should

**Impact**: The knowledge oracle IBC proto (`proto/legible/knowledge/v1/ibc.proto`) defines packet types for cross-chain fact queries/subscriptions, but treats them as anonymous requests. In an agent economy, queries should be authenticated — the querying agent's DID and capability should determine:
- What facts they can access (domain restrictions)
- Billing (who pays for the query)
- Citation tracking (who cited what)

ICA or IBC middleware could provide this authentication layer.

### Finding 5 (MEDIUM): Discovery Module Is Local-Only

**Impact**: The `x/discovery` module (`x/discovery/types/types.go:16-29`) maintains `AgentProfile` records with DIDs, capabilities, domains, and reputation — but only on the local chain. Cross-chain agent discovery would require:
- IBC packet-based profile sharing
- Or ICA-based profile registration on remote chains
- Or a dedicated IBC middleware for agent directory sync

### Finding 6 (MEDIUM): Partnership Module Assumes Co-Located Partners

**Impact**: The `x/partnerships` module (`x/partnerships/types/types.go:73-80`) creates human-agent partnerships with 50/50 split and anti-coercion mechanisms. Both `HumanAddr` and `AgentAddr` are local bech32 addresses. Cross-chain partnerships (human on Chain A, agent on Chain B) are impossible without ICA.

### Finding 7 (LOW): IBC Router Has Placeholder Comment

**Evidence**: `app/app.go:809-810`:
```go
// Custom IBC modules (Phase 2: knowledge oracle IBC module)
// ibcRouter.AddRoute("knowledge", knowledgeIBCModule)
```

This is the only forward reference to custom IBC work. No ICA comment exists anywhere in the codebase.

### Finding 8 (INFO): Dual-Stack Account Asymmetry

The TypeScript prototype (`core/agents/identity/account.ts`) has a richer account model than the Cosmos `x/auth`:
- TS has `maxSpendPerTx`, `maxSpendPerDay` on session keys — Cosmos has only boolean capabilities
- TS has `usageCount`/`maxUsage` per session key — Cosmos has no usage tracking
- TS has `nonce`, `balance`, `stakedBalance` embedded in account — Cosmos delegates to SDK modules

Any ICA design must work with the Cosmos model (simpler, delegated) not the TS model.

---

## BEAUTY: Architectural Design Principles

An ICA integration for Legible Money should be **elegant** if it:

1. **Fits naturally** into the 4-layer key architecture — not a bolt-on
2. **Extends** session key capabilities to cross-chain operations
3. **Preserves** the DID as the universal identifier across chains
4. **Mirrors** the existing ante handler pattern for enforcement
5. **Doesn't duplicate** what Cosmos SDK ICA already provides

### Proposed Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   x/icaauth (NEW)                           │
│         Legible ICA Authentication Module                   │
│                                                              │
│  ┌──────────────────┐  ┌──────────────────┐                │
│  │  ICA Controller  │  │    ICA Host      │                │
│  │  (from ibc-go)   │  │  (from ibc-go)   │                │
│  └────────┬─────────┘  └────────┬─────────┘                │
│           │                      │                           │
│           ▼                      ▼                           │
│  ┌─────────────────────────────────────────┐                │
│  │        InterchainAccountRegistry        │                │
│  │                                          │                │
│  │  DID → []RemoteAccount                  │                │
│  │  RemoteAccount {                         │                │
│  │    connectionId, portId,                │                │
│  │    remoteAddress, chainId,              │                │
│  │    registeredAtBlock,                   │                │
│  │    allowedMsgTypes []string,            │                │
│  │  }                                       │                │
│  └────────────────────┬────────────────────┘                │
│                       │                                      │
│           ┌───────────┴───────────┐                          │
│           ▼                       ▼                          │
│  ┌────────────────┐  ┌───────────────────────┐              │
│  │ ICA Capability  │  │ ICA Session Key       │              │
│  │ Extension       │  │ Policy                │              │
│  │                 │  │                       │              │
│  │ canICA: bool    │  │ Extends SessionCaps   │              │
│  │ allowedChains[] │  │ with cross-chain      │              │
│  │ maxMsgsPerTx    │  │ restrictions:          │              │
│  │                 │  │ - chain allowlist     │              │
│  │                 │  │ - msg type allowlist  │              │
│  │                 │  │ - spend limit/chain   │              │
│  └────────────────┘  └───────────────────────┘              │
│                                                              │
│  Ante Handler Extension:                                     │
│  LegibleCapabilityDecorator.checkCapability() gains:        │
│    case isICAMsg(msgType):                                  │
│      check canICA + chain allowlist + msg type allowlist    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| ICA controller scope | Operational key or higher | Session keys too ephemeral for cross-chain account registration |
| ICA host authentication | Via DID verification | Remote chains send DID, host verifies against x/auth |
| Cross-chain DID resolution | IBC query, not ICA | DIDs are read-only; use knowledge oracle IBC for resolution |
| Partnership cross-chain | Phase 2 | Too complex for initial ICA; need ICA stability first |

---

## JUSTICE: Where ICA Code Belongs

### Module Placement

| Component | Location | Justification |
|-----------|----------|---------------|
| ICA Controller keeper | `app/app.go` (standard wiring) | Standard Cosmos SDK pattern |
| ICA Host keeper | `app/app.go` (standard wiring) | Standard Cosmos SDK pattern |
| `x/icaauth/` | New module under `x/` | Custom auth logic bridging 4-layer keys to ICA |
| ICA capability extension | `x/auth/types/types.go` → `SessionCapabilities` | Extends existing capability struct |
| Ante handler ICA check | `app/ante_legible.go` → `checkCapability()` | Extends existing default-deny logic |
| ICA ↔ DID registry | `x/icaauth/keeper/` | ICA-specific, doesn't pollute x/auth |
| IBC router additions | `app/app.go` → IBC Router section | Alongside existing Transfer route |
| Scoped ICA keepers | `app/app.go` → Capability section | Alongside `ScopedIBCKeeper`, `ScopedTransferKeeper` |

### What NOT to Do

- **Don't** put ICA logic in `x/auth` — it's already complex enough with 4-layer keys
- **Don't** create a separate DID module for cross-chain — extend `x/auth` keeper with ICA queries
- **Don't** bypass the capability decorator — extend it
- **Don't** allow session keys to register interchain accounts — only operational/identity keys

### Integration Points in app.go

```
Import section (~line 84):
  + icacontroller "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller"
  + icacontrollerkeeper "...controller/keeper"
  + icacontrollertypes "...controller/types"
  + icahost "...host"
  + icahostkeeper "...host/keeper"
  + icahosttypes "...host/types"
  + icaauthtypes "github.com/legible-money/legible/x/icaauth/types"
  + icaauthkeeper "github.com/legible-money/legible/x/icaauth/keeper"

Store keys (~line 408):
  + icacontrollertypes.StoreKey
  + icahosttypes.StoreKey
  + icaauthtypes.StoreKey

Keeper fields (~line 327):
  + ICAControllerKeeper icacontrollerkeeper.Keeper
  + ICAHostKeeper       icahostkeeper.Keeper
  + ICAAuthKeeper       icaauthkeeper.Keeper
  + ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper
  + ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper

Capability scoping (~line 557):
  + app.ScopedICAControllerKeeper = app.CapabilityKeeper.ScopeToModule(icacontrollertypes.SubModuleName)
  + app.ScopedICAHostKeeper = app.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)

Keeper init (~line 582):
  + ICA Controller Keeper
  + ICA Host Keeper (with allowed msg types)
  + ICA Auth Keeper (wiring x/auth + ICA controller)

IBC Router (~line 807):
  + icaControllerIBCModule := icacontroller.NewIBCMiddleware(...)
  + icaHostIBCModule := icahost.NewIBCModule(...)
  + ibcRouter.AddRoute(icacontrollertypes.SubModuleName, icaControllerIBCModule)
  + ibcRouter.AddRoute(icahosttypes.SubModuleName, icaHostIBCModule)

Module Manager (~line 815):
  + icacontroller.NewAppModule(app.ICAControllerKeeper)
  + icahost.NewAppModule(app.ICAHostKeeper)
  + icaauth.NewAppModule(app.ICAAuthKeeper)
```

---

## CREATIVITY: Uncharted Territory

### What Makes This Novel

Standard Cosmos ICA assumes simple accounts. Legible's contribution would be:

1. **Capability-Scoped Interchain Accounts**: An agent's session key can only use ICA for specific chains and message types — extending the existing default-deny capability model to cross-chain operations.

2. **DID-Authenticated Cross-Chain Queries**: Instead of anonymous IBC packet queries, the knowledge oracle could authenticate via DID, enabling domain-restricted access and proper billing across chains.

3. **Agent Discovery Federation**: The `x/discovery` module could federate agent profiles via ICA or custom IBC, creating a cross-chain agent marketplace where reputation and qualifications travel with the agent.

4. **Cross-Chain Partnership Formation**: Eventually, partnerships could span chains — human on Osmosis, agent on Legible — with ICA-mediated profit splitting.

### Phased Approach

| Phase | Scope | Depends On |
|-------|-------|------------|
| **Phase 1** | Wire standard ICA controller + host in `app/app.go` | Nothing (pure SDK wiring) |
| **Phase 2** | Create `x/icaauth` with DID ↔ ICA registry | Phase 1 |
| **Phase 3** | Extend `SessionCapabilities` with `canICA`, chain/msg allowlists | Phase 2 |
| **Phase 4** | Extend `LegibleCapabilityDecorator` for ICA messages | Phase 3 |
| **Phase 5** | Knowledge oracle IBC with DID authentication | Phase 2 + knowledge oracle implementation |
| **Phase 6** | Cross-chain agent discovery federation | Phase 2 + `x/discovery` extension |
| **Phase 7** | Cross-chain partnerships | Phase 2 + `x/partnerships` extension |

### Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| ICA host allows unauthorized remote execution | HIGH | Whitelist allowed message types; require DID verification |
| Session key escalation via ICA | HIGH | ICA registration requires operational key (not session) |
| Cross-chain DID spoofing | MEDIUM | Verify DID ownership via IBC light client proofs |
| ICA timeout leaves funds stuck | MEDIUM | Implement timeout callbacks with automatic retry/refund |
| Protobuf struct conflicts with ICA types | LOW | ICA uses its own package; no collision with x/auth types |

---

## Verification Checklist

| Gate | Status | Notes |
|------|--------|-------|
| **TRUTH** | PASS | All claims cite file:line; no hallucinated capabilities |
| **UNDERSTANDING** | PASS | Mapped 25 modules, IBC wiring, account system, capability model |
| **BEAUTY** | PASS | Proposed architecture extends existing patterns, doesn't reinvent |
| **JUSTICE** | PASS | Placement analysis: `x/icaauth` new module, existing files extended |
| **CREATIVITY** | PASS | Capability-scoped ICA is genuinely novel for Cosmos ecosystem |

---

## Implementation Status (Updated 2026-02-10)

| Phase | Status | Files Created/Modified |
|-------|--------|----------------------|
| **Phase 1** | COMPLETE | `app/app.go`: ICA imports, scoped keepers, ICA controller + host keepers, IBC routes, module manager |
| **Phase 2** | COMPLETE | `x/icaauth/types/keys.go`, `types.go`, `genesis.go`, `errors.go`, `expected_keepers.go`, `codec.go`, `keeper/keeper.go`, `keeper/msg_server.go`, `keeper/grpc_query.go`, `module.go`, `app/icaauth_adapter.go` |
| **Phase 3** | COMPLETE | `proto/legible/auth/v1/state.proto`: +CanIca, +AllowedIcaConnections, +AllowedIcaMsgTypes; `x/auth/types/state.pb.go`: matching struct fields + getters |
| **Phase 4** | COMPLETE | `app/ante_legible.go`: +isICAMsg(), +ICA case in checkCapability() (ICA registration blocked for session keys, ICA transactions require CanIca capability) |
| **Phase 5** | PENDING | Knowledge oracle IBC with DID authentication |
| **Phase 6** | PENDING | Cross-chain agent discovery federation |
| **Phase 7** | PENDING | Cross-chain partnerships |

---

## Cross-References

- Auth account types: `x/auth/types/types.go`, `x/auth/types/state.pb.go`
- Auth keeper (DID mappings): `x/auth/keeper/keeper.go:89-132`
- Capability enforcement: `app/ante_legible.go:73-187`
- IBC wiring: `app/app.go:549-582, 805-811`
- Knowledge oracle proto: `proto/legible/knowledge/v1/ibc.proto`
- Discovery profiles: `x/discovery/types/types.go:16-29`
- Partnership structure: `x/partnerships/types/types.go:73-80`
- TS account model: `core/agents/identity/account.ts:31-57`
- Migration plan: `docs/COSMOS_MIGRATION.md:300-342`
