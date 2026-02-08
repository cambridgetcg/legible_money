# Token Expiration & Account Tiers — Deep Investigation

> **Date**: 2026-02-06
> **Method**: PP-guided 2-agent parallel exploration (expirable token architecture, account types & reward tiers)
> **Verdict**: LGM implements a 3-priority cascading expiration hierarchy — not explicit tiers, but a functional composition of global defaults, account type overrides, and participation-based dynamic scoring that creates differentiated token lifetimes per account.

---

## The 3-Priority Expiration Hierarchy

### Priority 1 (Base): Global Default

**File**: `core/tokens/types.ts:234-244`

```typescript
DEFAULT_EXPIRATION_CONFIG = {
  periodDurationMs: 7 * 24 * 60 * 60 * 1000,  // 1 week
  periodsUntilExpiration: 26,                   // ~6 months
  tokenLifespanMs: 26 * 7 * 24 * 60 * 60 * 1000,
  dustThreshold: 1000n,
  graceBlocks: 1000n,
  allowExtension: true,
  maxExtensionPeriods: 52,                      // Max 1 year
  retrievalRate: 0,                             // No transfer tax
  expirationMode: "direct_retrieval",
}
```

Applies to all accounts unless overridden.

### Priority 2 (Override): Account Type Configuration

**File**: `core/tokens/types.ts:247-259`

4 account types: `agent | validator | system | contract`

Each can override all config fields EXCEPT `periodDurationMs` and `tokenLifespanMs` (globally locked to ensure uniform time granularity).

| Configurable Field | Example Override |
|---|---|
| `periodsUntilExpiration` | Validators: 10 weeks vs agents: 26 |
| `retrievalRate` | Validators: 1% transfer tax vs agents: 0% |
| `expirationMode` | Contracts: `lock_and_unlock` vs agents: `direct_retrieval` |
| `allowExtension` | Contracts: false |
| `graceBlocks` | System: 0 |
| `renewalFeeRate` | Custom per type |
| `dustThreshold` | Custom per type |
| `maxExtensionPeriods` | Custom per type |

**API**: `manager.setAccountType(did, "validator")` + `manager.setTypeConfig("validator", { periodsUntilExpiration: 52 })`

### Priority 3 (Highest): Participation-Based Dynamic Expiration

**File**: `core/tokens/participation.ts`

Dynamic score (0-1000) maps to 2-182 periods of token lifespan.

**Scoring components**:

| Component | Weight | Full Score At |
|---|---|---|
| Claims accepted | 25% | 50 claims |
| Verifications correct | 30% | 200 verifications |
| Knowledge queries | 15% | 100 queries |
| Governance votes | 10% | 20 votes |
| Active stake | 20% | 10,000 LGM |

**Formula**:
```
rawScore = weighted sum of component scores (0-1000)
decayFactor = 0.5 ^ (blocksSinceActivity / 100,000)
finalScore = floor(rawScore * decayFactor)
periodsGranted = 2 + floor(180 * finalScore / 1000)
```

**Decay half-life**: 100,000 blocks (~2.9 days at 2521ms block time)

**Effective tiers by activity**:

| Activity Level | Score | Periods | Token Life |
|---|---|---|---|
| Brand new | 0 | 2 | ~2 weeks |
| Minimal | ~50 | ~11 | ~11 weeks |
| Regular | ~500 | ~92 | ~1.8 years |
| Highly active | ~1000 | 182 | ~3.5 years |
| Inactive >6 days | decayed | 2 | ~2 weeks |

---

## Resolution Cascade

**File**: `core/tokens/child_account.ts:107-138`

```
resolveConfig(account):
  1. Start with global config
  2. Apply type override (if setAccountType was called)
     - periodDurationMs and tokenLifespanMs stay LOCKED
  3. Apply participation score (if resolver injected)
     - Overrides periodsUntilExpiration and tokenLifespanMs
```

---

## Child Account Structure (Per-Period Tracking)

**File**: `core/tokens/types.ts:63-87`, `child_account.ts`

Tokens grouped by minting period, not individually tracked:

```
Parent: did:lgm:agent-123 (periodsUntilExpiration = 26)
+-- Child [period 100]: 500 LGM -> expires period 126
+-- Child [period 101]: 300 LGM -> expires period 127
+-- Child [period 102]: 200 LGM -> expires period 128

Balance = sum of non-expired children (O(1) query)
Period = floor(timestamp / periodDurationMs)
```

**ChildAccount fields**: `parentAccount`, `period`, `balance`, `expiresAt`, `swept`, `locked`

---

## Expiration Modes

| Mode | Behavior | Default For |
|---|---|---|
| `direct_retrieval` | Expired tokens swept to RAI holding account | agents, validators |
| `lock_and_unlock` | Tokens stay frozen, can unlock by paying `renewalFeeRate` | contracts (optional) |

---

## Transfer Policy

**File**: `core/tokens/types.ts:438-450`

```typescript
DEFAULT_TRANSFER_POLICY = {
  order: "lifo",                    // Send newest tokens first (preserves sender value)
  freshExpirationOnReceive: true,   // Recipients get full lifespan
  allowPartialOnExpiration: true,
  applyRetrievalRate: true,         // Transfer tax applied
}
```

LIFO sends newest tokens first, leaving oldest (near-expiry) with the sender.

---

## RAI Circular Economy

```
Minted Tokens -> Active Balances (child accounts)
    -> Expire (period > now - periodsUntilExpiration)
    -> RAI Holding Account (sweep)
    -> Eligibility Filter (age, activity, verification level)
    -> Distribution to eligible active accounts
    -> Back to Active Balances (new period, fresh expiration)
```

**RAI Eligibility** (`types.ts:389-404`): minAccountAge, minActivityScore, maxBalance, requiredVerificationLevel, excludedTypes (system accounts excluded).

---

## Orthogonal: Epistemic Category Vesting

**File**: `core/rewards/types.ts:54-91`

Separate from token expiration. Reward VESTING varies by claim type:

| Category | Half-Life | Max Release | Reserve |
|---|---|---|---|
| Axiomatic | 1.1 years | 95% | 5% |
| Formal Proof | 6.5 months | 92% | 8% |
| On-Chain / Cryptographic | 2.6 months | 90% | 10% |
| Peer-Reviewed / Replicated | 5.4 weeks | 85-88% | 12-15% |
| Contested | 1.1 weeks | 60% | 40% |

Once vested tokens are minted into an account, they follow that account's expiration rules (participation + type + global).

---

## Key Design Constraints

| Constraint | Reason |
|---|---|
| Period duration globally locked | All accounts use same time granularity for determinism |
| Participation score cached (24 blocks TTL) | Avoids expensive recalculation per transaction |
| Decay is block-based, not wall-clock | Deterministic, verifiable |
| LIFO transfer default | Preserves value for sender (newest tokens transferred first) |
| Child account per period | O(1) balance queries, efficient batch expiration |

---

## Key Files

| File | Purpose |
|---|---|
| `core/tokens/types.ts` | ExpirationConfig, AccountType, ChildAccount, RAIEligibility |
| `core/tokens/child_account.ts` | ChildAccountManager, resolveConfig(), type/participation resolution |
| `core/tokens/participation.ts` | ParticipationResolver, scoring formula, decay, caching |
| `core/tokens/expiration_tracker.ts` | ExpirationTracker, global expiration state, period transitions |
| `core/tokens/retrieval_engine.ts` | RAI sweep, eligibility filtering, distribution |
| `core/rewards/types.ts` | Epistemic category vesting profiles |
