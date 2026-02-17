# Compute Pool Mechanism — Detailed Design

> **Status**: Design Document
> **Author**: Protocol Team
> **Module**: `x/compute_pool`
> **Dependencies**: `x/vesting_rewards`, `x/staking`, `x/knowledge`, `x/billing`

---

## 1. Problem Statement

Validators running an LLM evaluation sidecar must call external AI APIs (Claude, OpenAI, Gemini, open-weight models, etc.) to verify knowledge claims through the 9-stage evaluation pipeline. These API calls cost real money:

- **Typical hosted LLM**: ~$3/M input tokens, ~$15/M output tokens
- **A typical claim evaluation**: ~2K input + ~1K output tokens = ~$0.02
- **A validator doing 100 evaluations/day**: ~$2/day = ~$60/month

This creates three problems:

1. **Barrier to entry** — Only validators who can afford API costs can participate in PoT
2. **Centralization pressure** — Wealthy actors can run more evaluations, earning more rewards
3. **Misaligned incentives** — Verification cost is private (validator pays), but the benefit is public (network gains knowledge)

The protocol already recognizes verification as useful work (PoT consensus), but doesn't fund the compute infrastructure that makes verification possible.

---

## 2. Solution: Protocol-Funded Compute Pool

The Compute Pool treats **verification compute as a consensus cost**, funded directly from block rewards. Just as Proof of Work blockchains implicitly fund electricity through block rewards, Legible Money explicitly funds AI inference through a dedicated compute pool.

### Core Loop

```
Block Rewards (3 LGM/block)
    │
    ├─ 7% → Research Fund
    │
    └─ 93% Post-Research
         │
         ├─ 50% → Block Producer
         │
         └─ 50% → Verification Pool
              │
              ├─ 70% → Verification Rewards (knowledge module)
              │         (pays validators for correct evaluations)
              │
              └─ 30% → Compute Pool ← NEW
                        (funds the compute that makes evaluations possible)
                             │
                             ├─ Credits issued to validators
                             ├─ Validators redeem against providers
                             ├─ Providers deliver API/GPU access
                             └─ Validators evaluate claims → earn LGM → cycle continues
```

---

## 3. Funding Model

### Current Block Reward Distribution

| Share | Recipient | Amount (epoch 0) |
|-------|-----------|-----------------|
| 7% | Research Fund | 0.21 LGM |
| 46.5% | Verification Pool (knowledge module) | 1.395 LGM |
| 46.5% | Block Producer | 1.395 LGM |

### Proposed Distribution (with Compute Pool)

The compute pool carves from the existing verification pool — not from the producer share or research fund. This is economically coherent: the verification pool funds verification, and compute is an input to verification.

| Share | Recipient | Amount (epoch 0) | Change |
|-------|-----------|-----------------|--------|
| 7% | Research Fund | 0.21 LGM | unchanged |
| 32.55% | Verification Rewards (knowledge module) | 0.9765 LGM | -0.4185 |
| **13.95%** | **Compute Pool** | **0.4185 LGM** | **NEW** |
| 46.5% | Block Producer | 1.395 LGM | unchanged |

### Implementation

New constant in `x/vesting_rewards/types/keys.go`:

```go
// ComputePoolShareBps is the share of the verification pool allocated to compute.
// 300000 = 30% of the verification pool (which is 50% of post-research reward).
// Net effect: 13.95% of total block reward funds compute.
ComputePoolShareBps = 300000

// ComputePoolModuleName is the module account that holds compute pool funds.
ComputePoolModuleName = "compute_pool"
```

Modified flow in `DistributeBlockReward()`:

```go
// After computing verificationPoolBig (50% of post-research):
computePoolBig := new(big.Int).Mul(verificationPoolBig, big.NewInt(ComputePoolShareBps))
computePoolBig.Div(computePoolBig, big.NewInt(1000000))

actualVerificationBig := new(big.Int).Sub(verificationPoolBig, computePoolBig)

// Send to respective module accounts:
// actualVerificationBig → knowledge module
// computePoolBig → compute_pool module
```

### Daily Pool Size (Epoch 0)

```
Per block:     0.4185 LGM
Per day:       0.4185 × 34,272 = 14,342 LGM/day
Per epoch:     0.4185 × 34,272 = 14,342 LGM/epoch
Per month:     ~430,000 LGM/month
```

---

## 4. Compute Credits (CU)

Credits are the internal accounting unit of the compute pool. They are **not tokens** — they are non-transferable, non-tradeable entitlements denominated in abstract "Compute Units" (CU).

### Why Credits, Not Direct LGM?

1. **Decoupling** — CU price adjusts with LGM market price; 1 CU always buys ~1 standard evaluation
2. **Non-transferable** — Prevents credit markets and wash trading
3. **Expiration** — Forces usage, preventing hoarding
4. **Metering** — Enables fine-grained cost tracking

### Credit Allocation

Validators earn credits when they **participate in verification rounds and produce valid evaluations**. Credits are not earned for merely being a validator — you must do work.

```
credits_earned = base_cu × complexity_factor × tier_multiplier
```

| Component | Value | Rationale |
|-----------|-------|-----------|
| `base_cu` | 100 CU | Covers 1 standard AI inference call |
| `complexity_factor` | `1.0 + 0.25 × evidence_count + 0.5 × cross_domain_refs` | Complex claims need more API calls (multi-evidence, cross-domain checks) |
| `tier_multiplier` | Apprentice: 0.5, Verified: 0.8, Bonded: 1.0, Guardian: 1.5 | Higher tiers handle harder claims needing more compute |

**Examples:**

| Claim Type | Evidence | Cross-Domain | Tier | Credits |
|------------|----------|-------------|------|---------|
| Simple fact check | 1 | 0 | Verified | `100 × 1.25 × 0.8 = 100 CU` |
| Multi-source claim | 3 | 0 | Bonded | `100 × 1.75 × 1.0 = 175 CU` |
| Cross-domain contested | 2 | 2 | Guardian | `100 × 2.5 × 1.5 = 375 CU` |

### Credit Lifecycle

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Allocated  │────→│   Active     │────→│   Redeemed   │────→│   Settled    │
│  (earned via │     │  (available  │     │  (claimed by │     │  (provider   │
│  verification)│     │   for use)   │     │  validator)  │     │    paid)     │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
                           │
                           │ (1 epoch / 34,272 blocks)
                           ▼
                     ┌──────────────┐
                     │   Expired    │
                     │  (returned   │
                     │   to pool)   │
                     └──────────────┘
```

**Key rules:**
- Credits expire after **1 epoch** (34,272 blocks ≈ 24 hours) if not redeemed
- Expired credits return to the compute pool (not burned)
- Maximum credit balance per validator: `10 × base_cu × tier_multiplier × epoch` (prevents hoarding)
- Credits are tied to a validator address, non-transferable

---

## 5. Compute Providers

### Registration

Anyone can register as a compute provider by staking LGM and declaring their service capabilities:

```go
type ComputeProvider struct {
    Address        string              // bech32 account address
    Did            string              // optional DID for agent providers
    ServiceType    ComputeServiceType  // ai_inference | gpu_compute | storage
    Endpoint       string              // off-chain service URL (for discovery only)
    PricePerCU     string              // ulgm per compute unit (provider sets this)
    Stake          string              // security deposit (min 100 LGM)
    Status         ProviderStatus      // active | paused | suspended | deregistered
    RegisteredAt   uint64              // block height

    // Performance metrics (rolling window)
    TasksServed    uint64              // lifetime tasks completed
    TasksFailed    uint64              // lifetime tasks failed
    AvgLatencyMs   uint64              // rolling average response time
    UptimeBps      uint64              // rolling 7-epoch uptime (basis points)
    LastHeartbeat  uint64              // last block where provider proved liveness
}
```

### Service Types

| Type | Description | Typical Provider | Use Case |
|------|-------------|-----------------|----------|
| `ai_inference` | LLM API access | Any LLM relay (Claude, OpenAI, Gemini, local) | Claim evaluation via sidecar |
| `gpu_compute` | Raw GPU cycles | GPU pool operators | Custom model inference, training |
| `storage` | Persistent storage | IPFS/Arweave pinners | Evidence caching, knowledge archival |

### Provider Requirements

| Requirement | Value | Rationale |
|-------------|-------|-----------|
| Minimum stake | 100 LGM | Skin in the game, Sybil resistance |
| Minimum uptime | 95% (950,000 bps) | Validators need reliable compute |
| Heartbeat interval | Every 1,000 blocks (~42 min) | Prove liveness |
| Maximum price | Governance-set ceiling | Prevent price gouging |
| Unbonding period | 1 epoch (34,272 blocks) | Can't rage-quit during active tasks |

### Provider Registration Message

```go
type MsgRegisterComputeProvider struct {
    Sender      string             `json:"sender"`
    ServiceType ComputeServiceType `json:"service_type"`
    Endpoint    string             `json:"endpoint"`
    PricePerCU  string             `json:"price_per_cu"`
    StakeAmount string             `json:"stake_amount"`
}
```

---

## 6. Credit Redemption

### Flow

```
Validator                    Chain                       Provider
   │                          │                            │
   │ 1. MsgRedeemCredits      │                            │
   │ (provider, cu_amount)    │                            │
   │─────────────────────────→│                            │
   │                          │                            │
   │                          │ 2. Validate:               │
   │                          │    - credits sufficient    │
   │                          │    - provider active       │
   │                          │    - price ≤ ceiling       │
   │                          │                            │
   │                          │ 3. Deduct credits          │
   │                          │                            │
   │                          │ 4. Create RedemptionTicket │
   │                          │    (pending settlement)    │
   │                          │                            │
   │                          │ 5. Emit event              │
   │                          │ (provider watches chain)   │
   │                          │───────────────────────────→│
   │                          │                            │
   │ 6. Call provider API     │                            │
   │────────────────────────────────────────────────────→  │
   │                          │                            │
   │ 7. Receive compute result│                            │
   │←────────────────────────────────────────────────────  │
   │                          │                            │
   │ 8. MsgConfirmUsage       │                            │
   │ (ticket_id, actual_cu)   │                            │
   │─────────────────────────→│                            │
   │                          │                            │
   │                          │ 9. Transfer LGM            │
   │                          │ compute_pool → provider    │
   │                          │                            │
   │                          │ 10. Update metrics         │
   │                          │                            │
```

### Redemption Ticket

```go
type RedemptionTicket struct {
    Id             string   // deterministic: sha256("redeem:{validator}:{provider}:{block}")
    Validator      string   // who's spending credits
    Provider       string   // who's providing compute
    CUAmount       uint64   // credits being redeemed
    LGMAmount      string   // ulgm to transfer (cu × pricePerCU)
    Status         string   // pending | confirmed | disputed | expired
    CreatedAt      uint64   // block height
    ExpiresAt      uint64   // must confirm within 1000 blocks (~42 min)
    ActualCU       uint64   // filled on confirmation (may differ from requested)
}
```

### Settlement

1. **Normal**: Validator calls `MsgConfirmUsage` within 1000 blocks → LGM transferred to provider
2. **Underuse**: `actual_cu < cu_amount` → Partial payment, excess credits returned
3. **Timeout**: No confirmation within 1000 blocks → Ticket expires, credits returned, provider's reliability score decremented
4. **Dispute**: Validator claims service wasn't delivered → Routed to `x/disputes` module, provider stake at risk

---

## 7. Provider Market Dynamics

### Price Discovery

Providers compete on price. The protocol enforces a ceiling but not a floor.

```
                    Price Ceiling (governance param)
                    ────────────────────────────────── $0.05/CU

Provider A:  ████████████████████████  $0.03/CU (hosted LLM relay, high volume)
Provider B:  ██████████████████        $0.025/CU (alternative relay, competitive)
Provider C:  ████████████              $0.02/CU (self-hosted GPU, cheapest)

                    ────────────────────────────────── $0.00/CU
                    Market Floor (natural)
```

**Selection incentives:**
- Validators choose the cheapest provider that meets their quality needs
- Providers with better uptime and lower latency get more business
- Competition drives prices toward marginal cost

### Provider Discovery

Validators discover providers through on-chain queries:

```go
// QueryActiveProviders returns providers sorted by price, filtered by service type
type QueryActiveProvidersRequest struct {
    ServiceType ComputeServiceType
    MinUptime   uint64   // minimum uptime in bps
    MaxPrice    string   // maximum price per CU in ulgm
}
```

---

## 8. Abuse Prevention

### 8.1 Credit Farming

**Attack**: Validator submits low-effort evaluations just to earn credits.

**Defense**: Credits are only granted for verifications that produce a **valid evaluation result** — the claim evaluation must pass the knowledge module's quality gate. The 8-stage evaluation pipeline already rejects shallow evaluations (anti-hallucination gates AH-1 through AH-5).

### 8.2 Credit Hoarding

**Attack**: Validator accumulates credits without spending them, creating artificial scarcity.

**Defense**:
- Credits expire after 1 epoch (24h)
- Max balance cap: `10 × base_cu × tier_multiplier × epoch`
- Expired credits return to pool (no waste)

### 8.3 Provider Collusion

**Attack**: Validator and provider collude — validator redeems credits, provider "delivers" nothing, they split the LGM.

**Defense**:
- Provider must stake 100+ LGM (slashable)
- Maximum 30% of pool can flow to any single provider per epoch
- Usage patterns monitored: if a validator always uses the same provider with 100% confirmation rate but low claim quality, flag for review
- Dispute window: anyone can challenge a redemption within 500 blocks

### 8.4 Sybil Providers

**Attack**: Single entity registers many providers to bypass per-provider caps.

**Defense**:
- Stake requirement (100 LGM each) makes Sybil expensive
- Minimum operational history (100 tasks served) before receiving >10% of pool
- Heartbeat requirement proves distinct operational infrastructure

### 8.5 Price Manipulation

**Attack**: Provider temporarily drops price to zero, captures all demand, then raises price.

**Defense**:
- Governance-set price ceiling prevents gouging
- Price changes take effect after 1000 blocks (no flash pricing)
- Validators can switch providers freely (no lock-in)

### 8.6 Phantom Compute

**Attack**: Provider claims to deliver compute but returns garbage results.

**Defense**:
- Validators won't confirm usage for bad results (no `MsgConfirmUsage` → no payment)
- Persistent bad results → validators switch to other providers
- Provider reputation score (tasks_failed / tasks_served) visible on-chain

---

## 9. Module Architecture

### Storage Layout

```go
// Key prefixes for x/compute_pool module
const (
    ProviderPrefix        = 0x01  // Provider records
    CreditBalancePrefix   = 0x02  // Per-validator credit balances
    CreditHistoryPrefix   = 0x03  // Credit allocation history
    RedemptionPrefix      = 0x04  // Redemption tickets
    UsageReportPrefix     = 0x05  // Confirmed usage reports
    EpochStatsPrefix      = 0x06  // Per-epoch pool statistics
    ProviderEpochPrefix   = 0x07  // Per-provider per-epoch flow tracking
    ParamsKey             = 0x08  // Module parameters
)
```

### Types

```go
type ComputeCredit struct {
    ValidatorAddr  string  // owner
    Balance        uint64  // current CU balance
    EarnedTotal    uint64  // lifetime CU earned
    RedeemedTotal  uint64  // lifetime CU redeemed
    ExpiredTotal   uint64  // lifetime CU expired
    LastEarnedAt   uint64  // block height of last credit grant
    ExpiresAt      uint64  // block height when current balance expires
}

type UsageReport struct {
    TicketId     string  // references RedemptionTicket
    Validator    string
    Provider     string
    RequestedCU  uint64  // what was requested
    ActualCU     uint64  // what was consumed
    LGMPaid      string  // actual ulgm transferred
    ConfirmedAt  uint64  // block height
}

type EpochStats struct {
    Epoch           uint64
    PoolInflow      string  // LGM received from block rewards
    CreditsIssued   uint64  // total CU allocated
    CreditsRedeemed uint64  // total CU redeemed
    CreditsExpired  uint64  // total CU expired
    LGMDistributed  string  // total LGM paid to providers
    ActiveProviders uint32  // providers that served tasks
    ActiveValidators uint32 // validators that used credits
}
```

### Expected Keepers

```go
type BankKeeper interface {
    SendCoins(ctx sdk.Context, from, to sdk.AccAddress, amt sdk.Coins) error
    SendCoinsFromModuleToAccount(ctx sdk.Context, module string, addr sdk.AccAddress, amt sdk.Coins) error
    SendCoinsFromModuleToModule(ctx sdk.Context, from, to string, amt sdk.Coins) error
    GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

type StakingKeeper interface {
    GetValidator(ctx sdk.Context, addr string) (*stakingtypes.LegibleValidator, bool)
    GetValidatorTier(ctx sdk.Context, addr string) stakingtypes.ValidatorTier
}

type KnowledgeKeeper interface {
    GetVerificationCount(ctx sdk.Context, validator string, epoch uint64) uint64
    GetClaimComplexity(ctx sdk.Context, claimId string) (evidenceCount uint64, crossDomainRefs uint64, found bool)
}
```

### Keeper

```go
type Keeper struct {
    cdc             codec.Codec
    storeService    store.KVStoreService
    bankKeeper      BankKeeper
    stakingKeeper   StakingKeeper
    knowledgeKeeper KnowledgeKeeper
    authority       string  // governance module address
}
```

### Messages

| Message | Description | Sender |
|---------|-------------|--------|
| `MsgRegisterComputeProvider` | Register as compute provider | Provider |
| `MsgUpdateProvider` | Update endpoint, price, pause/unpause | Provider |
| `MsgDeregisterProvider` | Exit and unstake (after unbonding) | Provider |
| `MsgProviderHeartbeat` | Prove liveness | Provider |
| `MsgRedeemCredits` | Spend credits for compute | Validator |
| `MsgConfirmUsage` | Confirm compute was delivered | Validator |
| `MsgDisputeUsage` | Challenge non-delivery | Validator |
| `MsgUpdateComputeParams` | Update module parameters | Governance |

### Queries

| Query | Description |
|-------|-------------|
| `QueryComputeProvider` | Get provider by address |
| `QueryActiveProviders` | List active providers (filterable) |
| `QueryCreditBalance` | Get validator's credit balance |
| `QueryRedemptionTicket` | Get ticket status |
| `QueryPoolStats` | Current pool balance and epoch stats |
| `QueryEpochStats` | Historical epoch statistics |

### ABCI Hooks

**BeginBlock:**

```go
func (am AppModule) BeginBlock(ctx sdk.Context) {
    // 1. Expire stale credits (balance past ExpiresAt)
    am.keeper.ExpireCredits(ctx)

    // 2. Expire unconfirmed redemption tickets (>1000 blocks old)
    am.keeper.ExpireRedemptionTickets(ctx)

    // 3. Check provider heartbeats (suspend inactive providers)
    am.keeper.CheckProviderLiveness(ctx)

    // 4. At epoch boundary: roll up stats, reset per-provider flow counters
    if isEpochBoundary(ctx) {
        am.keeper.FinalizeEpoch(ctx)
    }
}
```

---

## 10. Integration Points

### 10.1 x/vesting_rewards — Funding

Modified `DistributeBlockReward()` in `x/vesting_rewards/keeper/rewards.go`:

```go
// After computing verificationPoolBig:
computePoolBig := new(big.Int).Mul(
    verificationPoolBig,
    big.NewInt(int64(types.ComputePoolShareBps)),
)
computePoolBig.Div(computePoolBig, big.NewInt(1000000))
actualVerificationBig := new(big.Int).Sub(verificationPoolBig, computePoolBig)

// Send compute pool share to compute_pool module
if computePoolBig.Sign() > 0 {
    computeCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(computePoolBig)))
    k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "compute_pool", computeCoins)
}

// Send reduced verification share to knowledge module
if actualVerificationBig.Sign() > 0 {
    verificationCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(actualVerificationBig)))
    k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, types.KnowledgeModuleName, verificationCoins)
}
```

### 10.2 x/knowledge — Credit Grants

After a validator produces a valid evaluation in `RecordVerification()`:

```go
// Grant compute credits for completed verification
complexity := k.getClaimComplexity(ctx, claimId)
k.computePoolKeeper.GrantCredits(ctx, validatorAddr, claimId, complexity)
```

### 10.3 x/staking — Tier Lookups

Compute pool queries validator tier for credit multiplier:

```go
tier := k.stakingKeeper.GetValidatorTier(ctx, validatorAddr)
multiplier := tierMultiplier(tier) // 0.5, 0.8, 1.0, or 1.5
```

### 10.4 Daemon Integration

The LLM evaluation sidecar would be extended to:

1. **Check credit balance** before making API calls
2. **Discover cheapest provider** via chain query
3. **Redeem credits** via `MsgRedeemCredits`
4. **Call provider's API** using the redemption ticket as authorization
5. **Confirm delivery** via `MsgConfirmUsage` after receiving results

```
// Daemon pseudocode
async function evaluateClaim(claim) {
    const credits = await queryCredits(myValidatorAddr);
    if (credits.balance < EVAL_COST_CU) {
        log.warn("insufficient compute credits, skipping evaluation");
        return null;
    }

    const provider = await findBestProvider("ai_inference");
    const ticket = await redeemCredits(provider.address, EVAL_COST_CU);

    const result = await provider.evaluate(claim, ticket.id);

    await confirmUsage(ticket.id, result.actualCU);
    return result.evaluation;
}
```

---

## 11. Governance Parameters

All parameters are modifiable via `x/gov` proposals:

```go
type Params struct {
    // --- Pool Funding ---
    // ComputePoolShareBps: share of verification pool allocated to compute.
    // 300000 = 30%. Applied to the 50% verification pool share of post-research reward.
    ComputePoolShareBps   uint64  // default: 300000

    // --- Credit Allocation ---
    // BaseCUPerVerification: base compute units earned per valid verification.
    BaseCUPerVerification  uint64  // default: 100
    // ComplexityBonusBps: additional CU per evidence item (bps of base).
    // 25000 = 0.25 × base per evidence item.
    ComplexityBonusBps     uint64  // default: 25000
    // CrossDomainBonusBps: additional CU per cross-domain reference (bps of base).
    // 50000 = 0.5 × base per cross-domain ref.
    CrossDomainBonusBps    uint64  // default: 50000
    // MaxCreditBalance: maximum CU a validator can accumulate.
    // Expressed as multiplier of base × epoch (10 = 10 epochs worth).
    MaxCreditBalanceMultiplier uint64  // default: 10

    // --- Provider Requirements ---
    // MinProviderStake: minimum stake in ulgm to register as provider.
    MinProviderStake       string  // default: "100000000" (100 LGM)
    // MinUptimeBps: minimum uptime required to remain active.
    MinUptimeBps           uint64  // default: 950000 (95%)
    // HeartbeatIntervalBlocks: max blocks between heartbeats before suspension.
    HeartbeatIntervalBlocks uint64  // default: 1000
    // MaxPricePerCU: ceiling price per CU in ulgm (anti-gouging).
    MaxPricePerCU          string  // default: "10000" (0.01 LGM)
    // ProviderUnbondingBlocks: blocks before stake is returned after deregistration.
    ProviderUnbondingBlocks uint64  // default: 34272 (1 epoch)
    // PriceChangeDelayBlocks: blocks before price change takes effect.
    PriceChangeDelayBlocks  uint64  // default: 1000

    // --- Anti-Abuse ---
    // CreditExpiryBlocks: blocks before unused credits expire.
    CreditExpiryBlocks     uint64  // default: 34272 (1 epoch)
    // MaxProviderShareBps: max share of pool any single provider can receive per epoch.
    MaxProviderShareBps    uint64  // default: 300000 (30%)
    // RedemptionExpiryBlocks: blocks before unconfirmed redemption ticket expires.
    RedemptionExpiryBlocks uint64  // default: 1000
    // DisputeWindowBlocks: blocks during which a redemption can be disputed.
    DisputeWindowBlocks    uint64  // default: 500
    // MinTasksForFullShare: min lifetime tasks before provider can receive >10% of pool.
    MinTasksForFullShare   uint64  // default: 100

    // --- SLA ---
    // MaxLatencyMs: maximum acceptable response latency.
    MaxLatencyMs           uint64  // default: 30000 (30s)
    // SLAWindowBlocks: rolling window for SLA metric calculation.
    SLAWindowBlocks        uint64  // default: 1000
}
```

---

## 12. Economic Analysis

### Pool Sustainability

The compute pool is self-sustaining as long as `pool_inflow ≥ provider_payments`:

```
Pool Inflow (epoch 0):
  Per block:    0.4185 LGM
  Per epoch:    14,342 LGM
  Per month:    ~430,000 LGM

Provider Payments (depends on activity):
  CU issued:    100 CU × verifications_per_epoch
  CU cost:      ~0.01 LGM/CU (competitive market)
  Break-even:   14,342 / 0.01 = 1,434,200 CU/epoch
  At 100 CU/verification: 14,342 verifications/epoch covered
  With 100 validators:     143 verifications/validator/epoch
```

At epoch 0, the pool covers ~14K evaluations per day. As the network grows and block rewards decay:

| Epoch | Block Reward | Pool/Day | Evaluations/Day | Per Validator (100) |
|-------|-------------|----------|-----------------|-------------------|
| 0 | 3.0 LGM | 14,342 LGM | ~14,342 | ~143 |
| 5 | 1.77 LGM | 8,470 LGM | ~8,470 | ~84 |
| 10 | 1.05 LGM | 5,003 LGM | ~5,003 | ~50 |
| 20 | 0.37 LGM | 1,753 LGM | ~1,753 | ~17 |
| 50+ | 0.1 LGM (floor) | 478 LGM | ~478 | ~4 |

### Long-Term Sustainability

As block rewards decay toward the floor (0.1 LGM/block), the pool shrinks. Three factors compensate:

1. **API costs decrease** — AI inference costs drop ~50% annually; future models may be 10x cheaper
2. **Transaction fees grow** — As network usage increases, tx fees supplement block rewards via `RouteFees()`
3. **Verification efficiency** — Better models need fewer tokens per evaluation, stretching each CU further

The floor reward (0.1 LGM/block) permanently sustains ~478 evaluations/day — sufficient for a mature network where most knowledge is already verified and only new claims need evaluation.

### Validator Economics

A validator's total income from the protocol:

```
Total Income = Block Production Reward
             + Verification Rewards (from knowledge module)
             + Compute Credits (covers evaluation costs)

Net Profit   = Total Income - Operating Costs
             = Total Income - (API costs covered by credits) - (hardware, bandwidth)
             = Total Income - (hardware, bandwidth)  ← compute is free
```

The compute pool eliminates the largest variable cost for validators, making PoT validation profitable at lower stake levels.

---

## 13. Tier Multipliers

Credit allocation scales with validator tier, reflecting that higher-tier validators handle more complex (and compute-intensive) claims:

| Tier | Multiplier | Rationale |
|------|-----------|-----------|
| Apprentice (0.111 LGM) | 0.5x | Limited categories, simple claims |
| Verified (1.11 LGM) | 0.8x | Broader access, moderate complexity |
| Bonded (1,111 LGM) | 1.0x | Full access, standard complexity |
| Guardian (11,111 LGM) | 1.5x | Contested claims, cross-domain, highest compute need |

These multipliers align with the existing `RewardMultiplier` ratios in `x/staking/types/types.go` (0.1x, 0.5x, 1.0x, 2.0x), but slightly compressed because compute costs don't scale as aggressively as reward value.

---

## 14. Phased Rollout

### Phase 1: Foundation (DevNet)

- Deploy `x/compute_pool` module with basic types and keeper
- Manual credit allocation by governance (no automatic verification-based grants)
- Single whitelisted provider (protocol-run LLM relay)
- Fixed CU price (governance-set)
- Goal: Prove the credit→redeem→confirm lifecycle works

### Phase 2: Automation (TestNet)

- Automatic credit grants on verification completion
- Multi-provider registration (open registration with stake)
- Provider heartbeat and liveness monitoring
- Credit expiration and epoch rollover
- Basic SLA enforcement (uptime, latency)
- Goal: Prove market dynamics work with multiple providers

### Phase 3: Market (MainNet)

- Full provider competition with dynamic pricing
- Dispute resolution via `x/disputes` integration
- Cross-chain compute providers (IBC-enabled)
- Advanced anti-abuse (pattern detection, collusion scoring)
- Provider reputation scoring visible on-chain
- Goal: Self-sustaining compute market

---

## 15. Module File Structure

```
x/compute_pool/
├── keeper/
│   ├── keeper.go           # Core keeper, constructor, store access
│   ├── msg_server.go       # RegisterProvider, RedeemCredits, ConfirmUsage, etc.
│   ├── grpc_query.go       # Query handlers
│   ├── state.go            # CRUD: providers, credits, tickets, reports
│   ├── credits.go          # GrantCredits, ExpireCredits, credit math
│   ├── pool.go             # Pool funding receipt, epoch finalization
│   ├── providers.go        # Provider lifecycle, heartbeat, SLA checks
│   ├── redemption.go       # Ticket creation, confirmation, expiration
│   └── migrator.go         # Version migrations
├── types/
│   ├── types.go            # ComputeProvider, ComputeCredit, RedemptionTicket, etc.
│   ├── keys.go             # Storage prefixes, module name
│   ├── codec.go            # Amino/protobuf registration
│   ├── errors.go           # Module-specific errors
│   ├── expected_keepers.go # BankKeeper, StakingKeeper, KnowledgeKeeper interfaces
│   ├── params.go           # Params validation, defaults
│   ├── genesis.go          # Genesis state
│   ├── events.go           # Event types
│   └── query.go            # Query request/response types
└── module.go               # AppModule, RegisterServices, BeginBlock
```

---

## 16. Relationship to Existing Billing Module

The compute pool is **complementary** to `x/billing`, not a replacement:

| Aspect | x/billing | x/compute_pool |
|--------|-----------|---------------|
| **Who pays** | Knowledge consumers (query callers) | Protocol (block rewards) |
| **Who gets paid** | Knowledge API providers | Compute infrastructure providers |
| **What's bought** | Access to verified facts | AI inference to verify facts |
| **Flow direction** | Demand-side (users pay for knowledge) | Supply-side (protocol funds verification) |
| **Token flow** | User LGM → provider + knowledge pool + burn | Block reward LGM → compute pool → provider |

Together they form a complete knowledge economy:
- `x/billing`: Funds the **consumption** of knowledge (reading facts)
- `x/compute_pool`: Funds the **production** of knowledge (evaluating claims)

---

## 17. Future Extensions

### 17.1 Cross-Chain Compute (IBC)

Compute providers on other chains could register via IBC, enabling:
- Cheaper compute from chains with lower costs
- Geographic distribution of compute infrastructure
- Failover when local providers are unavailable

### 17.2 Compute Futures

Validators could pre-purchase compute credits at a fixed price for a future epoch, hedging against LGM price volatility.

### 17.3 Delegation-Funded Compute

Delegators could allocate a portion of their staking rewards to fund compute credits for their chosen validator, subsidizing more verification work.

### 17.4 Compute DAO

Heavy compute users (validators and providers) could form a subDAO to govern compute pool parameters independently from the main governance module, enabling faster parameter tuning.
