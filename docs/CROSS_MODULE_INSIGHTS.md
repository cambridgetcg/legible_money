# Cross-Module Keeper Call Insights

> **Legible Money Cosmos SDK Chain** | 24 custom modules + 15 standard SDK modules
> Analysis of 400+ cross-module keeper calls across 340 Go files

---

## 1. Module Dependency Graph

```
                          ┌─────────────────────┐
                          │   Cosmos SDK Base    │
                          │ (auth, bank, staking │
                          │  distr, slashing,    │
                          │  gov, ibc, evidence) │
                          └──────────┬──────────┘
                                     │
              ┌──────────────────────┼──────────────────────┐
              ▼                      ▼                      ▼
     ┌────────────────┐    ┌────────────────┐    ┌────────────────┐
     │  legible_auth   │    │    tokens      │    │   ontology     │
     │ (DID, sessions, │    │ (expirable     │    │ (domains,      │
     │  recovery, keys)│    │  balances)     │    │  strata)       │
     └───────┬────────┘    └───────┬────────┘    └───────┬────────┘
             │                     │                     │
             ▼                     ▼                     ▼
     ┌────────────────┐    ┌────────────────┐    ┌────────────────┐
     │ legible_staking │◄──│   billing      │    │ qualification  │
     │ (validators,    │    │ (API pricing,  │    │ (domain certs, │
     │  tiers, VRF)    │    │  citations)    │    │  pathways)     │
     └───────┬────────┘    └────────────────┘    └───────┬────────┘
             │                                           │
    ┌────────┼────────────────────────┬──────────────────┤
    ▼        ▼                        ▼                  ▼
┌────────┐ ┌───────────────┐  ┌──────────────┐  ┌──────────────────┐
│emergency│ │vesting_rewards│  │capture_defense│  │ capture_challenge│
└────────┘ └──────┬────────┘  └──────┬───────┘  └──────────────────┘
                  │                  │
                  ▼                  ▼
          ┌────────────────────────────────┐
          │         knowledge              │
          │  (facts, claims, verification  │
          │   rounds, confidence, VRF)     │
          │  DEPENDS ON: 8 keepers         │
          └──────────────┬─────────────────┘
                         │
                ┌────────┼────────┐
                ▼        ▼        ▼
          ┌─────────┐ ┌──────┐ ┌────────┐
          │disputes  │ │ bvm  │ │billing │
          │(arbiter, │ │(VM,  │ │(price, │
          │ slash)   │ │sched)│ │distrib)│
          └─────────┘ └──────┘ └────────┘

Leaf modules (minimal dependencies):
  partnership, partnerships, channels, claiming_pot,
  schedule, discovery, tree, home, research, gov
```

---

## 2. Keeper Dependency Heatmap

| Keeper | Modules Using It | Total Calls | Risk Level |
|--------|-----------------|-------------|------------|
| **bankKeeper** | 18+ modules | 45+ | CRITICAL |
| **stakingKeeper** | 8 modules | 25+ | HIGH |
| **tokensKeeper** | 4 modules | 12+ | MEDIUM |
| **knowledgeKeeper** | 3 modules | 12+ | MEDIUM |
| **ontologyKeeper** | 5 modules | 8+ | MEDIUM |
| **authKeeper** | 2 modules | 6+ | LOW |
| **vestingRewardsKeeper** | 2 modules | 4+ | LOW |
| **schedKeeper** | 1 module | 5 | LOW |
| **evidenceMgmtKeeper** | 1 module | 2 | LOW |
| **captureDefenseKeeper** | 1 module | 2 | LOW |

### bankKeeper: The Central Hub

Every module that moves funds routes through `bankKeeper`. The 5 primary methods:

```
SendCoinsFromAccountToModule()  ← 17 calls (stake deposits, fees)
SendCoinsFromModuleToAccount()  ← 17 calls (rewards, withdrawals)
SendCoins()                     ← 3 calls  (peer-to-peer)
BurnCoins()                     ← 8 calls  (slash penalties)
MintCoins()                     ← 1 call   (vesting_rewards only)
```

Only `vesting_rewards` has `Minter` permission. 13 modules have `Burner`. This is correct — minting is tightly controlled.

---

## 3. Adapter/Bridge Pattern Analysis

Cross-module type incompatibilities are resolved via **10 adapter structs** in `app/`:

| Adapter | Source | Target | File |
|---------|--------|--------|------|
| `StakingKeeperAdapter` | x/staking | x/knowledge | staking_adapter.go |
| `VestingStakingAdapter` | x/staking | x/vesting_rewards | staking_adapter.go |
| `EvidenceMgmtKeeperAdapter` | x/evidence_mgmt | x/knowledge | staking_adapter.go |
| `BVMKnowledgeAdapter` | x/knowledge | x/bvm | bvm_adapters.go |
| `BVMStakingAdapter` | x/staking | x/bvm | bvm_adapters.go |
| `BVMAuthAdapter` | x/auth | x/bvm | bvm_adapters.go |
| `DisputeKnowledgeAdapter` | x/knowledge | x/disputes | dispute_adapters.go |
| `DisputeStakingAdapter` | x/staking | x/disputes | dispute_adapters.go |
| `StakeQueryAdapter` | x/staking | x/gov, x/research | dispute_adapters.go |
| `DisputeVestingAdapter` | x/vesting_rewards | x/disputes | dispute_adapters.go |
| `CaptureDefenseOntologyAdapter` | x/ontology | x/capture_defense | capture_adapters.go |

All adapters follow the same pattern: thin wrappers with zero business logic, only type conversion and keeper method delegation. Compile-time interface satisfaction checks (`var _ Interface = (*Adapter)(nil)`) ensure correctness.

---

## 4. Ante Decorator Chain

```
[IBC Redundant Relay]
        ↓
[LGMGasDecorator]           ← Maps 65+ msg types to gas costs
[FeeRouterDecorator]        ← 22% research fund, 78% validators
[LegibleDIDDecorator]       ← DID memo validation + resolution
        ↓
[Standard Cosmos SDK × 11]  ← SetUpContext through IncrementSequence
        ↓
[LegibleAccountDecorator]   ← Frozen account enforcement + LastActiveBlock
[LegibleCapabilityDecorator]← Session key permission bits (4 capabilities)
```

**Session key capabilities** enforced at ante level:
- `CanTransfer` — MsgSend, MsgMultiSend, MsgTransfer
- `CanStake` — MsgDelegate, MsgUndelegate, MsgBeginRedelegate, MsgRegisterValidator
- `CanSubmitClaims` — MsgSubmitClaim, MsgSubmitCommitment, MsgSubmitReveal, MsgChallengeFact
- `CanVote` — MsgVote, MsgCastVote, MsgVoteHalt, MsgVoteRevert, MsgVoteResume

---

## 5. Block Lifecycle Ordering

### BeginBlock (27 modules)

```
Phase 1: Cosmos SDK infrastructure
  upgrade → distr → slashing → evidence → staking → ibc → ibc-transfer

Phase 2: Core Legible infrastructure
  emergency → vesting_rewards → tokens → legible_staking → gov → research

Phase 3: Knowledge ecosystem
  ontology → disputes → evidence_mgmt → claiming_pot → channels

Phase 4: Execution & coordination
  schedule → partnership → tree → bvm → home

Phase 5: Quality assurance (runs last)
  qualification → capture_defense → capture_challenge → knowledge
```

### EndBlock (25 modules)

```
Phase 1: Cosmos SDK finalization
  gov → staking → ibc → ibc-transfer → feegrant

Phase 2: Legible finalization
  legible_staking → knowledge → tokens → emergency → gov → research
  → vesting_rewards → ontology → disputes → evidence_mgmt
  → claiming_pot → channels → schedule → partnership → tree
  → bvm → home → qualification → capture_defense → capture_challenge
```

### InitGenesis (34 modules)

Knowledge module initializes **last** — correct, since it depends on 8 other keepers that must be initialized first.

---

## 6. Critical Dependency Chains

### The Knowledge Verification Path (deepest)

```
knowledge ← staking (VRF selection, stake weights)
         ← tokens (period-based locking)
         ← ontology (confidence ceilings)
         ← qualification (domain filtering)
         ← capture_defense (reputation weights)
         ← vesting_rewards (truth-linked vesting)
         ← evidence_mgmt (evidence quality)
         ← bank (fund movements)
```

**8 keeper dependencies** — highest in the codebase. This is architecturally justified: knowledge verification is the core consensus operation and must integrate truth assessment, economic incentives, reputation, and domain expertise.

### The Dispute Resolution Path

```
disputes ← knowledge (fact/claim state)
         ← staking (arbiter selection)
         ← vesting_rewards (clawback)
         ← bank (bond/reward transfers)
```

### The BVM Execution Path

```
bvm ← knowledge (fact access)
    ← staking (validator info)
    ← auth (session keys, DID)
    ← schedule (timed execution)
```

---

## 7. Findings & Risks

### FINDING-1: Stake Type Inconsistency (MEDIUM)

`DisputeStakingAdapter` returns stake as `uint64`:
```go
// dispute_adapters.go
func parseStakeToUint64(stakeStr string) uint64 { ... }
```

`StakeQueryAdapter` returns stake as `string`:
```go
// dispute_adapters.go - used by gov + research
func (a StakeQueryAdapter) GetStake(...) string { return validator.TotalStake }
```

**Risk**: Consumers of `StakeQueryAdapter` must parse strings themselves. If one module expects uint64 and gets string, silent zero-value bugs can occur.

**Recommendation**: Unify on a single return type. Since LGM uses bigint math, `math.Int` (Cosmos SDK math type) is the safest choice.

### FINDING-2: Incomplete Ontology Adapter (LOW)

`CaptureDefenseOntologyAdapter` has stub methods:
```go
func (a CaptureDefenseOntologyAdapter) GetDomainDepth(...) int { return 1 }
func (a CaptureDefenseOntologyAdapter) GetParentDomain(...) (string, bool) { return "", false }
```

Domain hierarchy is not yet tracked. Capture defense reputation scoring defaults to flat domain depth, which means:
- All domains treated as leaf nodes (depth=1)
- No parent domain reputation propagation
- Stratum-based scoring works, hierarchy-based doesn't

**Risk**: Low immediate risk. Capture defense functions correctly with flat domains. Full hierarchy wiring needed before mainnet.

### FINDING-3: bankKeeper as Single Point of Failure (HIGH)

18+ modules depend on `bankKeeper` for all fund movements. While this is standard Cosmos SDK architecture, the breadth is notable:
- A misconfigured module account permission could freeze an entire subsystem
- Module account balance exhaustion cascades to all dependent operations

**Mitigation already in place**: `maccPerms` correctly separates Minter (only vesting_rewards) from Burner (13 modules). Module account registration in `app.go` is complete.

### FINDING-4: Potential Circular Dependency Path (MEDIUM)

```
knowledge → evidence_mgmt (via EvidenceMgmtKeeperAdapter)
evidence_mgmt → ??? (does evidence_mgmt call back into knowledge?)
```

Currently `evidence_mgmt` only depends on `bankKeeper` — no circular dependency exists. But if evidence management ever needs to query fact state, the cycle would need an adapter or event-based decoupling.

### FINDING-5: Knowledge Module BeginBlock Ordering (VERIFIED CORRECT)

Knowledge runs **last** in both BeginBlock and InitGenesis. This is correct because:
- Verification rounds need validator set (staking must run first)
- Confidence aggregation needs reputation scores (capture_defense must run first)
- Evidence linking needs evidence state (evidence_mgmt must run first)

### FINDING-6: Module Account Permissions Audit

| Module | Permissions | Assessment |
|--------|------------|------------|
| vesting_rewards | Minter, Burner | Correct — only minter in system |
| legible_staking | Burner, Staking | Correct — manages bonded pools |
| tokens | Burner | Correct — burns expired tokens |
| legible_auth | nil | Correct — holds bootstrap fund, sends only |
| 10 other modules | Burner | Correct — can burn slashed/penalized funds |
| research_fund | Burner | Correct — can burn unused bounties |
| burn | Burner | Correct — dedicated burn sink |

No module has unauthorized permissions. The permission model is sound.

### FINDING-7: Post-Instantiation Wiring (INFO)

Three keepers use post-instantiation `Set*` methods to break circular construction order:

```go
// app.go lines 734-749
app.KnowledgeKeeper.SetOntologyKeeper(app.OntologyKeeper)      // line 734
app.BVMKeeper.SetKnowledgeKeeper(NewBVMKnowledgeAdapter(...))   // line 737
app.BVMKeeper.SetStakingKeeper(NewBVMStakingAdapter(...))       // line 738
app.BVMKeeper.SetAuthKeeper(NewBVMAuthAdapter(...))             // line 739
app.DisputesKeeper.SetVestingKeeper(NewDisputeVestingAdapter(...)) // line 749
```

This is standard Cosmos SDK practice for breaking keeper construction cycles. These keepers are nil-safe until wired.

---

## 8. Cross-Module Call Inventory by Module

| Module | External Keepers | Calls | Key Operations |
|--------|-----------------|-------|----------------|
| **knowledge** | bank, tokens, ontology, evidence_mgmt, vesting_rewards, capture_defense, staking | 30+ | Claims, verification, confidence |
| **disputes** | bank, staking, knowledge, vesting_rewards | 15+ | Dispute resolution, slashing |
| **staking** | bank, tokens | 11+ | Delegation, unbonding |
| **bvm** | auth, schedule | 10+ | Smart contracts, sessions |
| **billing** | bank, knowledge, tokens | 9+ | API billing, citations |
| **qualification** | bank, staking, ontology | 9+ | Domain certifications |
| **research** | bank | 8+ | Bounties, fund management |
| **tree** | bank | 7+ | Project lifecycle, rewards |
| **claiming_pot** | bank | 7+ | Pot creation, eligibility |
| **channels** | bank | 6+ | Payment channels |
| **auth** | bank, cosmos-auth | 5+ | Recovery, bootstrap |
| **partnership** | bank | 5+ | Formation, profit splits |
| **schedule** | bank | 5+ | Scheduled execution |
| **evidence_mgmt** | bank | 4+ | Evidence submission |
| **vesting_rewards** | bank | 4+ | Vesting distribution |
| **capture_defense** | ontology | 4+ | Reputation scoring |
| **tokens** | bank | 3+ | Transfers, burning |
| **emergency** | staking | 3+ | Emergency state |
| **discovery** | bank | 2+ | Provider registration |
| **capture_challenge** | staking | 2+ | Challenge enforcement |
| **ontology** | bank | 1+ | Domain registration |
| **partnerships** | (internal) | 1+ | Profit distribution |
| **home** | (none) | 0 | Fully self-contained |

---

## 9. Module Isolation Tiers

```
Tier 0 — Fully Isolated (0 external keepers)
  └─ home

Tier 1 — Minimally Coupled (bankKeeper only)
  └─ channels, tokens, discovery, research, ontology,
     tree, schedule, partnership, claiming_pot, evidence_mgmt,
     vesting_rewards

Tier 2 — Moderately Coupled (2-3 external keepers)
  └─ staking (bank + tokens)
     auth (bank + cosmos-auth)
     gov (staking + bank)
     capture_defense (staking + ontology + bank)
     capture_challenge (capture_defense + staking + bank)
     emergency (staking)

Tier 3 — Highly Connected (4+ external keepers)
  └─ knowledge (8 keepers — heaviest)
     disputes (4 keepers)
     bvm (4 keepers via adapters)
     billing (3 keepers)
     qualification (3 keepers)
```

---

## 10. Architectural Strengths

1. **Clean adapter pattern**: All cross-module type conversions happen in dedicated adapter files in `app/`, not scattered through module code
2. **Compile-time interface checks**: Every adapter uses `var _ Interface = (*Adapter)(nil)` for safety
3. **Correct module ordering**: BeginBlock/EndBlock/InitGenesis ordering respects dependency chains
4. **Tight minting control**: Only `vesting_rewards` can mint — all other token creation flows through genesis or existing supply
5. **4-layer ante enforcement**: Session key capabilities checked before any transaction reaches a keeper
6. **Nil-safe post-wiring**: Keepers with circular dependencies use setter injection, safely nil until wired

---

*Generated 2026-02-10 via PP cross-module review*
