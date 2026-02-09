# Legible Money — Cosmos SDK Module Overview

> This document provides a high-level map of all Go chain modules, their dependencies, and the application wiring layer.

## Module Inventory

| Module | StoreKey | Status | Purpose |
|--------|----------|--------|---------|
| `x/knowledge` | `knowledge` | **Complete** | Proof of Truth consensus — claims, verification rounds, facts |
| `x/auth` (legible) | `legible_auth` | **Complete** | 4-layer key architecture, DID, session keys |
| `x/staking` (legible) | `legible_staking` | **Complete** | Graduated 4-tier validators, delegation, unbonding |
| `x/tokens` | `tokens` | **Complete** | Period-based expirable balances, RAI pool, sweep |
| `x/emergency` | `emergency` | **Complete** | 3-phase BFT kill switch (Halt/Revert/Resume) |
| `x/gov` (lgm_gov) | `lgm_gov` | **Complete** | LIP lifecycle, quadratic voting, treasury disbursements |
| `x/research` | `research` | **Complete** | Peer-reviewed research, bounties, treasury |
| `x/disputes` | `disputes` | **Complete** | Multi-tier fact arbitration with escalation |
| `x/evidence_mgmt` | `evid_mgmt` | **Complete** | Evidence auditing, oracle registration, consensus scoring |
| `x/ontology` | `ontology` | **Complete** | Gödelian strata, domain governance, cross-stratum links |
| `x/vesting_rewards` | `vesting_rewards` | **Partial** | Time-locked reward release (EndBlocker works, msgs deferred) |
| `x/bvm` | `bvm` | **Stub** | Blockchain VM for self-invoking contracts |
| `x/channels` | `channels` | **Stub** | Off-chain payment channels |
| `x/claiming_pot` | `claiming_pot` | **Stub** | Collaborative reward pooling |
| `x/tree` | `tree` | **Stub** | Project/task hierarchy (Tree of Life) |

Plus standard Cosmos SDK modules: `auth`, `bank`, `staking`, `distribution`, `gov`, `crisis`, `mint`, `slashing`, `params`, `capability`, `ibc`, `transfer`.

---

## Dependency Graph

```
                    ┌──────────────┐
                    │   x/tokens   │
                    │ (expirable   │
                    │  balances)   │
                    └──────┬───────┘
                           │ DebitBalance / CreditBalance
                    ┌──────┴───────┐
                    │  x/staking   │◄─────────────────────────────┐
                    │ (validators, │                               │
                    │  tiers)      │                               │
                    └──┬───┬───┬───┘                               │
         GetValidator  │   │   │  GetEffectiveStake               │
        ┌──────────────┘   │   └──────────────────┐               │
        ▼                  ▼                       ▼               │
 ┌──────────────┐  ┌──────────────┐       ┌──────────────┐        │
 │ x/knowledge  │  │ x/emergency  │       │  x/disputes  │        │
 │ (PoT engine) │  │ (kill switch)│       │ (arbitration)│        │
 └──────┬───────┘  └──────────────┘       └──────┬───────┘        │
        │                                         │ GetFact/SetFact│
        │ GetFact/GetClaim                        │                │
        ├─────────────────────────────────────────┘                │
        ▼                                                          │
 ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
 │  x/ontology  │  │  x/research  │  │  x/gov       │─────────────┘
 │ (strata)     │  │ (labs/bounty)│  │ (LIP/disb.)  │  GetValidatorStake
 └──────────────┘  └──────────────┘  └──────────────┘

 ┌──────────────┐  ┌──────────────┐
 │ x/auth       │  │ x/evidence   │
 │ (DID, keys)  │  │ (audits)     │
 └──────────────┘  └──────────────┘

All modules depend on Cosmos SDK BankKeeper for coin transfers.
Auth module depends on Cosmos SDK AccountKeeper for BaseAccount sync.
```

---

## Application Wiring (`app/app.go`)

### Keeper Initialization Order

Keepers are initialized in dependency order in `NewLegibleApp()`:

1. Standard SDK keepers (AccountKeeper, BankKeeper, etc.)
2. IBC keepers (CapabilityKeeper, IBCKeeper, TransferKeeper)
3. `TokensKeeper` — no custom dependencies
4. `LegibleStakingKeeper` — receives TokensKeeper
5. `LegibleAuthKeeper` — receives Cosmos AccountKeeper
6. `KnowledgeKeeper` — receives StakingKeeperAdapter + TokensKeeper
7. `EmergencyKeeper` — receives LegibleStakingKeeper directly
8. `LegibleGovKeeper` — receives StakingKeeper + BankKeeper
9. `ResearchKeeper` — receives BankKeeper + StakingKeeper
10. `OntologyKeeper` — receives BankKeeper
11. `DisputesKeeper` — receives DisputeKnowledgeAdapter + DisputeStakingAdapter + BankKeeper
12. `EvidenceMgmtKeeper` — receives BankKeeper

### Module Manager Ordering

**BeginBlockers** (executed every block start):
```
emergency → tokens → legible_staking → lgm_gov → research →
ontology → disputes → evidence_mgmt → knowledge
```
Emergency runs first (halt guard). Tokens sweeps expired balances. Staking processes mature unbondings. Knowledge advances verification round phases last.

**EndBlockers** (executed every block end):
```
legible_staking → knowledge → tokens → emergency → lgm_gov →
research → ontology → disputes → evidence_mgmt
```
Staking safety tier checks. Knowledge cleans up expired rounds. Emergency resets per-epoch counters.

**InitGenesis** (chain initialization):
```
Standard SDK modules first (auth → bank → staking → distribution → ...)
then: legible_auth → tokens → emergency → knowledge → legible_staking →
lgm_gov → research → ontology → disputes → evidence_mgmt → stubs...
```

---

## Adapter Pattern

Three adapter structs bridge type differences between modules:

| Adapter | Location | Wraps | Satisfies |
|---------|----------|-------|-----------|
| `StakingKeeperAdapter` | `app/staking_adapter.go` | `x/staking` Keeper | `x/knowledge` StakingKeeper interface |
| `DisputeKnowledgeAdapter` | `app/dispute_adapters.go` | `x/knowledge` Keeper | `x/disputes` KnowledgeKeeper interface |
| `DisputeStakingAdapter` | `app/dispute_adapters.go` | `x/staking` Keeper | `x/disputes` StakingKeeper interface |

The emergency module needs no adapter — `LegibleStakingKeeper` satisfies its `StakingKeeper` interface directly.

---

## Transaction Validation Pipeline (`app/ante.go` + `app/ante_legible.go`)

The ante handler chain has 20 decorators in 4 phases:

### Phase 1: IBC
1. `RedundantRelayDecorator` — deduplicate IBC relays

### Phase 2: Legible Pre-Auth (before signature verification)
2. `LGMGasDecorator` — validate gas limits (block: 33.3M, tx: 11.1M, min: 22K), lookup per-message gas costs
3. `FeeRouterDecorator` — log 22% research / 78% validator fee split
4. `LegibleDIDDecorator` — parse `did:lgm:` from tx memo, resolve to address, emit event

### Phase 3: Standard SDK
5-16. SetUpContext, ExtensionOptions, ValidateBasic, TxTimeout, Memo, TxSize, DeductFee, SetPubKey, SigCount, SigGas, SigVerification, IncrementSequence

### Phase 4: Legible Post-Auth (after signature verification)
17. `LegibleAccountDecorator` — frozen account check, update `LastActiveBlock`
18. `LegibleCapabilityDecorator` — session key capability enforcement (transfer/stake/claim/vote)

---

## ABCI++ Integration (`app/abci.go` + `app/vote_extensions.go`)

### PrepareProposal
1. Emergency halt guard — filter non-emergency txs if chain halted
2. Process vote extensions from previous block (aggregate reveals per round)
3. Generate system txs for completed verification rounds
4. Fill remaining space with mempool txs (respecting 33.3M block gas limit)

### ProcessProposal
1. Decode and validate all txs
2. Emergency halt guard — reject proposals with non-emergency txs during halt

### ExtendVote (per validator)
1. Query active verification rounds at current height
2. **Commit phase**: Generate VRF → check selection → compute commitment hash → store salt locally
3. **Reveal phase**: Retrieve local salt → emit reveal with verdict + confidence

### VerifyVoteExtension
1. Validate commitment format (64-hex hash, non-empty round ID)
2. Verify VRF proof if present
3. Validate reveal (verdict valid, confidence ≤ 1M, salt present)
4. Cross-check reveal hash matches stored commitment

---

## Gas Costs (`app/gas.go`)

### Limits
| Constant | Value | Notes |
|----------|-------|-------|
| `BlockGasLimit` | 33,333,333 | Max gas per block |
| `TxGasLimit` | 11,111,111 | Max gas per transaction |
| `SelfInvocationLimit` | 5,555,555 | Max gas for BVM contract calls |
| `MinGasLimit` | 22,222 | Minimum gas per tx |
| `MinGasPrice` | 1 ulgm/gas | Floor price |

### Fee Routing (basis points)
| Route | BPS | Percentage |
|-------|-----|------------|
| Research tax | 2200 | 22% |
| Validator fees | 7800 | 78% |
| Burn rate (penalties) | 500 | 5% |

### Transaction Gas Costs (38 types)

| Category | Transaction | Gas |
|----------|-------------|-----|
| **Core** | transfer | 21,000 |
| **Core** | stake / unstake | 50,000 |
| **Knowledge** | claim_submit | 100,000 |
| **Knowledge** | verification_commit | 30,000 |
| **Knowledge** | verification_reveal | 40,000 |
| **Knowledge** | challenge_fact | 80,000 |
| **Knowledge** | add_fact | 120,000 |
| **Staking** | delegate / undelegate | 60,000 |
| **Staking** | register_validator | 100,000 |
| **Staking** | redelegate | 80,000 |
| **Auth** | register_account | 50,000 |
| **Auth** | rotate_key | 30,000 |
| **Auth** | create_session | 40,000 |
| **Auth** | revoke_session | 25,000 |
| **Auth** | recover_account | 80,000 |
| **Governance** | governance_propose / submit_lip | 100,000 |
| **Governance** | governance_vote / cast_vote | 30,000 |
| **Governance** | submit_disbursement | 80,000 |
| **Governance** | execute_disbursement | 100,000 |
| **Disputes** | initiate_dispute | 100,000 |
| **Disputes** | arbiter_vote | 30,000 |
| **Research** | submit_research | 80,000 |
| **Research** | create_bounty | 80,000 |
| **Emergency** | propose_halt/revert/resume | 50,000 |
| **Emergency** | vote_halt/revert/resume | 25,000 |
| **System** | verification_reward (50K), slash, epoch | fee-exempt (gas defined but not charged) |

---

## In-Memory State (`app/local_state.go`)

`LocalCommitmentStore` — thread-safe (RWMutex) map storing validator commitment salts between the commit and reveal phases of PoT verification rounds.

- **Store**: Called during `ExtendVote` commit phase
- **Get**: Called during `ExtendVote` reveal phase
- **CleanupExpired**: Removes commitments older than 100 blocks
- **Phase 2 only**: Validator restart between phases forfeits rewards (file-based persistence planned for Phase 3)

---

## Review Documents

| Document | Covers |
|----------|--------|
| [01 — Knowledge + Staking + Tokens](./01_KNOWLEDGE_STAKING_TOKENS.md) | Core PoT engine, validator tiers, expirable balances |
| [02 — Disputes + Evidence](./02_DISPUTES_EVIDENCE.md) | Fact arbitration, evidence auditing, oracles |
| [03 — Auth & Identity](./03_AUTH_IDENTITY.md) | DID system, 4-layer keys, session keys, ante decorators |
| [04 — Emergency + Gov + Research](./04_EMERGENCY_GOV_RESEARCH.md) | Kill switch, LIP governance, research labs |
| [05 — Ontology](./05_ONTOLOGY.md) | Gödelian strata, confidence ceilings, domain governance |
| [06 — Stub Modules](./06_STUB_MODULES.md) | BVM, channels, claiming pot, tree, vesting (Phase 2 roadmap) |
