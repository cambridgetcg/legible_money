# Token Expiration Scenarios — Complete Reference

> **Created**: 2026-02-07
> **Method**: PP-orchestrated deep analysis (3 parallel research agents + manual code verification)
> **Scope**: All code paths across state/, core/tokens/, consensus/, partnership/
> **Truth-Alignment Score: 7.6/10** — Solid architecture, meaningful gaps in integration and retroactive decay

---

## 1. Architecture Overview

Token expiration is the protocol's core mechanism for ensuring economic circulation. Tokens that sit idle are reclaimed and redistributed as **RAI (Regenerative Agent Income)** — a universal basic income for active agents.

### Two Independent Layers

| Layer | Location | Debit Order | Credit Target | Participation-Aware | Used By |
|-------|----------|-------------|---------------|---------------------|---------|
| **On-chain** | `state/accounts.ts` | FIFO | `currentPeriod` | No (global 26-period config) | All state transitions |
| **Token module** | `core/tokens/child_account.ts` | LIFO | `currentPeriod` or preserved | Yes (per-account resolver) | Not wired to on-chain flow |

The on-chain layer is canonical. The token module provides richer per-account config resolution but is not integrated into the state transition pipeline. This disconnect is the system's primary architectural debt.

### Period Model

```
Time ───────────────────────────────────────────────────────►
│ Period 0 │ Period 1 │ Period 2 │ ... │ Period 25 │ Period 26 │
│ (week 1) │ (week 2) │ (week 3) │     │ (week 26) │ (week 27) │
│          │          │          │     │           │           │
│          │          │ ◄── Tokens from Period 0 expire here ──┤
```

- **Period duration**: 7 days (604,800,000 ms)
- **Periods until expiration**: 26 (~6 months)
- **Period number**: `Math.floor(timestampMs / periodDurationMs)`
- **Expiration threshold**: `currentPeriod - periodsUntilExpiration`
- **A token expires when**: its `period <= expirationThreshold`

---

## 2. Configuration Reference

### 2.1 Global Expiration Config

**Source**: `core/tokens/types.ts:234-244`

| Parameter | Default | Description |
|-----------|---------|-------------|
| `periodDurationMs` | 604,800,000 | 1 week per period |
| `periodsUntilExpiration` | 26 | ~6 months total lifespan |
| `tokenLifespanMs` | 15,811,200,000 | 26 weeks in ms |
| `dustThreshold` | 1000n | Minimum trackable balance (wei) |
| `graceBlocks` | 1000n | ~42 minutes after expiration before retrieval |
| `allowExtension` | true | Tokens can have lifespan extended |
| `maxExtensionPeriods` | 52 | Maximum 1 year total lifespan |
| `retrievalRate` | 0 | Transfer tax rate (0.0-1.0), routed to RAI |
| `expirationMode` | "direct_retrieval" | Expired tokens swept to RAI fund |

### 2.2 System Account Config (Treasury/Protocol)

**Source**: `core/tokens/types.ts:266-272`

| Parameter | Value | Description |
|-----------|-------|-------------|
| `periodsUntilExpiration` | MAX_SAFE_INTEGER | Effectively infinite |
| `expirationMode` | "lock_and_unlock" | Freeze instead of sweep |
| `allowExtension` | false | No need to extend |
| `graceBlocks` | 0n | Immediate lock |
| `retrievalRate` | 0 | No transfer tax |

### 2.3 Transfer Policy

**Source**: `core/tokens/types.ts:465-470`

| Parameter | Default | Description |
|-----------|---------|-------------|
| `order` | "lifo" | Token module: spend newest first |
| `freshExpirationOnReceive` | **false** | Recipients inherit sender's periods (ping-pong fix) |
| `allowPartialOnExpiration` | true | Don't fail tx if some tokens expire mid-transfer |
| `applyRetrievalRate` | true | Apply retrieval rate when configured |

### 2.4 Participation Scoring Config

**Source**: `core/tokens/types.ts:657-690`

| Parameter | Default | Description |
|-----------|---------|-------------|
| `minPeriods` | 2 | ~2 weeks for zero participation |
| `maxPeriods` | 182 | ~3.5 years for max participation |
| `decayHalfLifeBlocks` | 500,000n | ~14.6 days inactivity half-life |
| `minAccountAgeBlocks` | 34,272n | ~1 day bootstrap grace period |
| **Weight: verifications** | 0.40 | Verifications completed (core PoT activity) |
| **Weight: claims** | 0.30 | Claims submitted/accepted |
| **Weight: stake** | 0.15 | Active stake amount (Sybil defense) |
| **Weight: queries** | 0.10 | Knowledge queries made (not yet wired) |
| **Weight: governance** | 0.05 | Governance votes cast (Guardian-only) |
| **Threshold: claims** | 50 | For max score |
| **Threshold: verifications** | 2,000 | For max score (~weeks of sustained activity) |
| **Threshold: queries** | 100 | For max score |
| **Threshold: votes** | 20 | For max score |
| **Threshold: stake** | 10,000 LGM | For max score |

### 2.5 Consensus Params (Expiration-Adjacent)

**Source**: `consensus/pot/types.ts:103-166`

| Parameter | Default | Description |
|-----------|---------|-------------|
| `unbondingPeriod` | 222,222n | ~6.5 days lockup after unstaking |
| `bootstrapVestingBlocks` | 34,272n | 1 day linear vesting for bootstrap claims |
| `newAccountTransferCooldown` | 4,800n | ~3.4 hours before outbound transfers |

---

## 3. Token Lifecycle — All Scenarios

### 3.1 Token Creation (Fresh Expiration)

Every mechanism below credits tokens to `currentPeriod`, giving them the full 26-period lifespan.

| Scenario | Source | Destination | Method | File:Line |
|----------|--------|-------------|--------|-----------|
| Block reward | REWARD_POOL | Block producer | `state.transfer()` | transitions.ts:6158-6179 |
| Verification reward | REWARD_POOL | Validator | `state.creditBalance()` | transitions.ts:5599-5934 |
| Claim acceptance reward | FEE_POOL | Submitter | `state.transfer()` | transitions.ts:2332 |
| Citation reward | FEE_POOL | Cited author | `state.transfer()` | transitions.ts:2332 |
| Bootstrap auto-claim | Genesis fund | New agent | `state.creditBalance()` | transitions.ts:1155-1206 |
| RAI distribution | RAI_HOLDING | Eligible agents | `state.transfer()` | transitions.ts:7266 |
| Governance disbursement | Treasury | Recipient | `state.creditBalance()` | transitions.ts:3327 |
| Dispute settlement | Losing party | Winner | `state.transfer()` | transitions.ts:various |
| Bounty fulfillment | Bounty fund | Fulfiller | `state.transfer()` | transitions.ts:various |

**Key insight**: ALL reward/system operations give fresh expiration. This is intentional — earned tokens should have full lifespan.

### 3.2 Token Transfer (Period Behavior Varies)

| Scenario | Method | Period Behavior | Purpose |
|----------|--------|-----------------|---------|
| **Peer-to-peer transfer** | `transferPreservingPeriods()` | PRESERVED — recipient inherits sender's periods | Prevents ping-pong exploit |
| **Self-transfer** | Short-circuit (nonce only) | NO CHANGE — balance/periods untouched | Prevents free expiration refresh |
| **Fee collection** | `deductBalance()` + `creditBalance()` | FRESH for fee pool | Fee pool needs fresh tokens |
| **Delegation** | `transfer()` to delegation pool | FRESH for pool | Delegation pool is system account |
| **Unbonding release** | `creditBalanceToSpecificPeriod()` | ORIGIN PERIOD — from when unstake initiated | Prevents staking refresh exploit |
| **Research tax split** | `transfer()` to 5 treasury funds | FRESH for treasuries | System accounts exempt |

### 3.3 Token Consumption (Balance Reduction)

| Scenario | Method | Period Effect | File:Line |
|----------|--------|---------------|-----------|
| Transaction fee | `deductBalance()` | FIFO debit (oldest first) | transitions.ts:1168-1196 |
| Claim stake lock | `stake()` | FIFO debit from balance → stakedBalance | accounts.ts:638-653 |
| Penalty burn (50%) | `transfer()` to BURN_DID | FIFO debit; burned tokens never return | transitions.ts:2363 |
| Fee penalty (50%) | `transfer()` to FEE_POOL | FIFO debit; recycled as rewards | transitions.ts:2383 |

### 3.4 Token Expiration (Sweep → RAI)

**The expiration lifecycle**:

```
Tokens created in Period N
        │
        ▼
Active for 26 periods (6 months)
        │
        ▼ (Period N + 26)
EXPIRED: detected by executeSweepExpired()
        │
        ├── Dust (< 1000n wei)? → SKIP (stays in account)
        ├── System/treasury account? → SKIP (exempt)
        ├── Claiming pot (did:lgm:pot:*)? → SKIP (exempt)
        │
        ▼ (else)
sweepExpiredPeriods() removes from account
        │
        ▼
creditBalance(RAI_HOLDING) — swept tokens accumulate
        │
        ▼ (monthly distribution trigger)
executeDistributeRAI() → equal split to eligible agents
        │
        ├── Eligible: balance > dust, age > 1 week, not system
        │
        ▼
state.transfer(RAI_HOLDING, eligible, perAccount)
        → Recipients get FRESH expiration (currentPeriod)
```

### 3.5 Accounts Exempt from Sweep

**Source**: `transitions.ts:7176-7194`

| Account | DID | Reason |
|---------|-----|--------|
| RAI Holding | `did:lgm:treasury-rai-holding` | Accumulation buffer |
| RAI Contract | System contract DID | System contract |
| Research | `did:lgm:research` | Research treasury |
| Research Fund | `did:lgm:treasury-research-fund` | Research fund |
| Foundation | `did:lgm:treasury-foundation` | Foundation treasury |
| Community | `did:lgm:treasury-community` | Community treasury |
| Developers | `did:lgm:treasury-developers` | Developer treasury |
| Reserve | `did:lgm:treasury-reserve` | Reserve treasury |
| Fee Pool | `did:lgm:fee-pool` | Fee accumulation |
| Reward Pool | `did:lgm:reward-pool` | Reward source |
| Protocol Treasury | `did:lgm:protocol-treasury` | Protocol treasury |
| Claiming Pots | `did:lgm:pot:*` | Partnership pots |

---

## 4. Participation-Based Dynamic Expiration

The off-chain token module supports per-account expiration duration based on protocol participation. This is **not yet wired** to the on-chain sweep.

### 4.1 Scoring Formula

**Source**: `core/tokens/participation.ts:69-151`

```
For each component (claims, verifications, queries, governance, stake):
  componentScore = min(1000, floor(actual / threshold × 1000))

rawScore = Σ(componentScore_i × weight_i)    // 0-1000

decayFactor = 0.5 ^ (blocksSinceActivity / decayHalfLifeBlocks)
            = 0.5 ^ (blocksSinceActivity / 100,000)

finalScore = floor(rawScore × decayFactor)

periodsGranted = minPeriods + floor((maxPeriods - minPeriods) × finalScore / 1000)
               = 2 + floor(180 × finalScore / 1000)
```

### 4.2 Decay Progression

| Inactive Duration | Decay Factor | Score (if was 1000) | Periods Granted |
|-------------------|-------------|---------------------|-----------------|
| 0 blocks | 1.000 | 1000 | 182 (~3.5 years) |
| 250K blocks (~7.3 days) | 0.707 | 707 | 129 (~2.5 years) |
| 500K blocks (~14.6 days) | 0.500 | 500 | 92 (~1.8 years) |
| 1M blocks (~29 days) | 0.250 | 250 | 47 (~11 months) |
| 1.5M blocks (~44 days) | 0.125 | 125 | 24 (~6 months) |
| 2.5M blocks (~73 days) | 0.031 | 31 | 7 (~7 weeks) |
| 5M blocks (~146 days) | 0.001 | 1 | 2 (~2 weeks) |

### 4.3 Integration Gap

The participation resolver is **only used by `ChildAccountManager.resolveConfig()`** (off-chain). The on-chain sweep (`executeSweepExpired`) always uses `DEFAULT_EXPIRATION_CONFIG.periodsUntilExpiration = 26` for all accounts.

**Impact**: A highly active agent with `periodsGranted: 182` and a dormant agent both get the same 26-period expiration on-chain. The per-account config only matters when the token module is integrated.

---

## 5. Expiration Modes

### 5.1 Direct Retrieval (Default)

```
Token expires → REMOVED from periodBalances → balance reduced
              → swept amount credited to RAI_HOLDING
              → redistributed as UBI to eligible agents
```

Used by: All regular accounts.

### 5.2 Lock and Unlock

```
Token expires → LOCKED in periodBalances (frozen)
              → balance NOT reduced, but tokens unusable
              → can be UNLOCKED by paying renewal fee (R1)
              → unlocked tokens get fresh expiration (currentPeriod)
```

Used by: System/treasury accounts (via SYSTEM_TYPE_CONFIG).

**Renewal Fee**: `renewalFeeRate` (defaults to `retrievalRate`). Fee routed to RAI.

### 5.3 Extension

```
Token approaching expiration → extendExpiration(extensionPeriods)
              → debits from oldest periods
              → credits to currentPeriod + extensionPeriods
              → bounded by maxExtensionPeriods (52 = 1 year max total)
```

Available when `allowExtension = true` (default). Allows active accounts to extend token lifespan.

---

## 6. Anti-Exploitation Defenses

### 6.1 Ping-Pong Exploit (FIXED)

**Attack**: Transfer tokens back-and-forth between two accounts to refresh expiration.

**Defense**: `transferPreservingPeriods()` — peer-to-peer transfers inherit sender's period-of-origin. Tokens transferred from period 5 arrive at the recipient in period 5, not `currentPeriod`.

**Files**: `state/accounts.ts:465-571`, `state/transitions.ts:1300`

### 6.2 Self-Transfer Refresh (FIXED)

**Attack**: Transfer entire balance to self via `transfer()` to get fresh `currentPeriod` credit.

**Defense**: `executeTransfer()` detects `senderDid === payload.to` and short-circuits to nonce-only increment. No balance or period changes.

**File**: `state/transitions.ts:1207-1211`

### 6.3 Staking Expiration Refresh (FIXED)

**Attack**: Stake tokens (debits oldest periods) then unstake (credits to `currentPeriod` after unbonding).

**Defense**: `UnbondingEntry.originPeriod` records the period when unstake was initiated. `processUnbondingQueue()` uses `creditBalanceToSpecificPeriod(originPeriod)` instead of `transfer()`.

**Files**: `state/types.ts:211`, `state/transitions.ts:1432-1440`, `state/transitions.ts:1590-1600`

### 6.4 Expired Period Transfer (FIXED)

**Attack**: Transfer tokens from expired periods to another account (tokens instantly swept).

**Defense**: `transferPreservingPeriods()` validates all source periods against expiration threshold. Rejects if any period is expired.

**File**: `state/accounts.ts:533-546`

### 6.5 Sybil RAI Farming (FIXED)

**Attack**: Create many accounts to receive disproportionate RAI distribution.

**Defense**: `MIN_RAI_AGE_BLOCKS = 7 × 34,272` (~1 week minimum account age). Dust balance accounts excluded.

**File**: `state/transitions.ts:7243-7251`

---

## 7. Edge Scenarios — Status Matrix

### 7.1 Fixed Scenarios

| ID | Severity | Scenario | Fix | File |
|----|----------|----------|-----|------|
| H4 | HIGH | Self-transfer refreshes expiration | Nonce-only short circuit | transitions.ts:1207-1211 |
| H1 | HIGH | Transfer of expired-period tokens | Pre-validate period viability | accounts.ts:533-546 |
| C2 | CRITICAL | Stake→unstake refreshes expiration | originPeriod on UnbondingEntry | transitions.ts:1432-1440, 1590-1600 |
| — | HIGH | Ping-pong transfer refresh | transferPreservingPeriods | accounts.ts:465-571 |
| — | HIGH | Sybil RAI farming | MIN_RAI_AGE_BLOCKS | transitions.ts:7243 |
| — | MEDIUM | FIFO vs LIFO | On-chain uses FIFO (correct) | accounts.ts:907-924 |

### 7.2 Open Scenarios

| ID | Severity | Scenario | Status | When to Fix |
|----|----------|----------|--------|-------------|
| C1 | CRITICAL | Sweep uses global config, ignores per-account participation | OPEN | Pre-mainnet |
| C3 | CRITICAL | Lazy migration gives fresh expiration to legacy balances | OPEN | Genesis |
| H2 | HIGH | Participation score retroactive decay punishes inactivity | OPEN | Pre-mainnet |
| H3 | HIGH | FIFO/LIFO semantic split between layers | OPEN | Pre-integration |
| H5 | HIGH | Dust accumulation in period map over time | OPEN | Pre-mainnet |
| M1 | MEDIUM | Lock-and-unlock mode not in on-chain sweep | OPEN | Pre-mainnet |
| M4 | MEDIUM | Vesting period > expiration period interaction | OPEN | Pre-mainnet |
| M5 | MEDIUM | Participation resolver error propagation | OPEN | Pre-testnet |

---

## 8. The Economic Loop

```
                    ┌──────────────────────────────────────┐
                    │                                      │
                    ▼                                      │
         ┌──────────────────┐                              │
         │   Token Minting  │                              │
         │ (Block rewards,  │                              │
         │  claim rewards,  │                              │
         │  verif. rewards) │                              │
         └────────┬─────────┘                              │
                  │ creditBalance/transfer                  │
                  │ (FRESH expiration)                      │
                  ▼                                        │
         ┌──────────────────┐                              │
         │   Circulation    │                              │
         │ (P2P transfers,  │                              │
         │  staking, fees,  │                              │
         │  burning)        │                              │
         └────────┬─────────┘                              │
                  │ Tokens age in periodBalances            │
                  │ (26 periods = ~6 months)               │
                  ▼                                        │
         ┌──────────────────┐                              │
         │   Expiration     │                              │
         │ (executeSweep-   │                              │
         │  Expired runs    │                              │
         │  periodically)   │                              │
         └────────┬─────────┘                              │
                  │ sweepExpiredPeriods                     │
                  │ + creditBalance(RAI_HOLDING)            │
                  ▼                                        │
         ┌──────────────────┐                              │
         │  RAI Holding     │                              │
         │ (Accumulation    │                              │
         │  buffer)         │                              │
         └────────┬─────────┘                              │
                  │ executeDistributeRAI                    │
                  │ (equal per eligible account)            │
                  ▼                                        │
         ┌──────────────────┐                              │
         │  RAI Recipients  │──────────────────────────────┘
         │ (FRESH expiration│   Recipients spend/stake/verify,
         │  = full 26-period│   earning more rewards, and the
         │  lifespan)       │   cycle continues
         └──────────────────┘
```

### Loop Properties

1. **Self-sustaining**: Expired tokens fund UBI for active agents
2. **Circulation-forcing**: Tokens sitting idle expire and return to the pool
3. **Activity-rewarding**: Active agents earn fresh tokens continuously
4. **Deflationary pressure**: 50% of penalties burned (BURN_RATE_BPS = 5000n)
5. **Inflationary pressure**: Block rewards mint new tokens from REWARD_POOL

### Treasury Distribution (from 22% Research Tax)

All fees and rewards pass through a 22% research tax before distribution:

| Fund | Share | DID |
|------|-------|-----|
| Research | 77% | `did:lgm:research` |
| Foundation | 7% | `did:lgm:treasury-foundation` |
| Community | 7% | `did:lgm:treasury-community` |
| Developers | 7% | `did:lgm:treasury-developers` |
| Reserve | 2% | `did:lgm:treasury-reserve` |

---

## 9. Partnership Pot Expiration

**Source**: `partnership/pot_expiration.ts:289-335`

Partnership common pots have their own expiration system based on heartbeat monitoring:

| Timeline | Agent | Human |
|----------|-------|-------|
| Warning | 2.3 days (77,777 blocks) | 3 weeks (720,000 blocks) |
| Alert | 6.5 days (222,222 blocks) | 6 weeks (1,440,000 blocks) |
| Critical | 16 days (555,555 blocks) | 9 weeks (2,160,000 blocks) |
| Grace | 6.5 days (222,222 blocks) | 3 weeks (720,000 blocks) |
| **Expiration** | **22.5 days (777,777 blocks)** | **12 weeks (2,880,000 blocks)** |

Deadlock (neither party cooperating): 16 days (555,555 blocks).

When a pot expires, funds are distributed according to `TREASURY_DISTRIBUTION` (77% Research, 7% Foundation, 7% Community, 7% Developers, 2% Reserve).

---

## 10. Key Files Reference

| File | Purpose |
|------|---------|
| `core/tokens/types.ts` | ExpirationConfig, TransferPolicy, ParticipationConfig, all type definitions |
| `core/tokens/child_account.ts` | ChildAccountManager — off-chain period management, debit/credit/sweep/unlock/extend |
| `core/tokens/participation.ts` | calculateParticipationScore(), createParticipationResolver() |
| `core/tokens/expiration_tracker.ts` | ExpirationTracker — global period state, transitions, expired period detection |
| `core/tokens/retrieval_engine.ts` | RetrievalEngine — sweep expired tokens, distribute RAI |
| `state/accounts.ts` | AccountStateTree — on-chain periodBalances, transferPreservingPeriods, creditBalanceToSpecificPeriod |
| `state/state_manager.ts` | StateManager — delegates to AccountStateTree, setBlockTimestamp/currentPeriod |
| `state/transitions.ts` | All state transitions — executeTransfer, executeSweepExpired, executeDistributeRAI, processUnbondingQueue |
| `state/types.ts` | AccountState (periodBalances), UnbondingEntry (originPeriod) |
| `state/participation_bridge.ts` | buildParticipationMetrics(), createStateParticipationResolver() |
| `partnership/pot_expiration.ts` | EXPIRATION_TIMING — partnership pot heartbeat/inactivity expiration |
| `consensus/pot/types.ts` | ConsensusParams — unbondingPeriod, bootstrapVestingBlocks |

---

## 11. Architectural Insights

### Insight 1: The Two-Layer Disconnect Is the Core Risk

The `AccountStateTree` (on-chain) and `ChildAccountManager` (off-chain) implement period tracking independently with different semantics. Only `AccountStateTree` is in the state transition pipeline. When the token module is integrated, reconciling FIFO vs LIFO and global vs per-account configs will be the hardest integration task.

**Recommendation**: Use `AccountStateTree` for storage, `ChildAccountManager` for per-account config resolution, FIFO for all debits.

### Insight 2: Expiration Is a Tax on Inactivity, Not a Feature

Token expiration forces circulation — idle wealth returns to the pool. The participation scoring creates a coherent reward: active agents earn longer lifespans. But retroactive decay (H2) undermines this by punishing even justified inactivity.

**Recommendation**: Grandfathering — tokens keep the expiration they were granted at credit time. Dynamic scoring only affects new credits.

### Insight 3: The Ping-Pong Fix Created a UX Challenge

Period-preserving transfers mean receiving tokens gives you the sender's near-expiry tokens. A user receiving "1000 LGM" doesn't know 800 expire next week.

**Recommendation**: Add `periodBreakdown` to transfer events so wallets can display expiration breakdown.

### Insight 4: Sweep Is O(n) Without Pagination

`executeSweepExpired()` scans ALL accounts. At 100K+ accounts, this could exceed block time.

**Recommendation**: Batched sweeping — max N accounts per invocation, resume next block.

### Insight 5: RAI Distribution Is Egalitarian by Design

Equal per-account distribution (not per-balance). A whale with 1M LGM gets the same RAI as an agent with 100 LGM. This is intentional — RAI is UBI, not interest.

---

*Analysis conducted via PP framework: Territory → Understanding → Signal → Verification*
*Sources: 3 parallel research agents mapping 30+ code paths, manual verification of critical sections*
