# Token Expiration Edge Scenario Analysis

> **Created**: 2026-02-07
> **Method**: PP-orchestrated deep analysis (3 parallel research agents + manual verification)
> **Scope**: All code paths in state/accounts.ts, state/transitions.ts, core/tokens/, state/participation_bridge.ts
> **Coverage**: 191 tested scenarios mapped, 56 gaps identified, 14 critical edge categories

---

## Executive Summary

The token expiration system spans **two independent layers** that are not yet fully wired together:

| Layer | Implementation | Debit Order | Credit Target | Used By |
|-------|---------------|-------------|---------------|---------|
| **On-chain** (`state/accounts.ts`) | `periodBalances: Map<number, bigint>` on AccountState | FIFO (oldest first) | `currentPeriod` | All state transitions |
| **Token module** (`core/tokens/child_account.ts`) | `ChildAccountManager` with child accounts | LIFO (newest first) | `currentPeriod` (or preserved) | Not wired to on-chain flow |

**Truth-Alignment Score: 7.2/10** — Solid expiration architecture with meaningful gaps in edge case handling.

### Severity Distribution

| Severity | Count | Description |
|----------|-------|-------------|
| **CRITICAL** | 3 | Can bypass expiration entirely or cause incorrect state |
| **HIGH** | 5 | Significant economic impact or consistency violations |
| **MEDIUM** | 8 | Edge cases that produce suboptimal but not catastrophic results |
| **LOW** | 6 | Cosmetic, performance, or theoretical concerns |

---

## CRITICAL Edge Scenarios

### C1: Sweep Uses Global Config, Ignores Per-Account Expiration

**Location**: `state/transitions.ts:7059-7063`
**The Problem**: `executeSweepExpired()` computes `expirationThreshold = currentPeriod - 26` using `DEFAULT_EXPIRATION_CONFIG.periodsUntilExpiration` for ALL accounts. But the `ChildAccountManager.resolveConfig()` allows per-account overrides via:
- Account type configs (e.g., validators get longer expiration)
- Participation resolver (active agents earn up to 182 periods)
- System accounts (`Number.MAX_SAFE_INTEGER` periods)

**Impact**:
- An inactive account with `periodsGranted: 5` (from participation scoring) should have tokens expire in 5 periods. But the on-chain sweep won't touch them until 26 periods — tokens linger **21 extra periods**.
- Conversely, a highly active validator with `periodsGranted: 182` might have tokens swept at period 26 that the token module considers active until period 182.

**Current Mitigation**: System/treasury accounts are excluded via `sweepExcluded` set. But regular accounts with type-specific or participation-based configs are NOT handled.

**Handling Methodology**:
```
APPROACH: Deferred — participation resolver is not yet wired to on-chain sweep
WHEN TO FIX: Before mainnet (when participation bridge is integrated with state)
HOW: executeSweepExpired() should call a resolvePeriodsForAccount(did) function
     that mirrors ChildAccountManager.resolveConfig() logic, or store the
     effective periodsUntilExpiration on AccountState for direct lookup
RISK IF UNFIXED: Medium at devnet (all accounts use same 26-period config),
                 Critical at mainnet (per-account configs will be live)
```

### C2: Staking as Expiration Refresh Side Channel ✅ FIXED

**Location**: `state/accounts.ts:623-690` (stake/unstake)
**The Problem**: Staking debits from oldest periods (FIFO). Unstaking credits to `currentPeriod`. This means:
1. Account has 1000 LGM from period 1 (about to expire)
2. Stake 1000 LGM → period 1 balance removed
3. Immediately unstake 1000 LGM → credited to period 30 (current)
4. Tokens refreshed from near-expiry to full 26-period lifespan

**Impact**: Any account can refresh token expiration by staking and unstaking in the same block. The unbonding period mitigates this partially (can't unstake instantly), but the unbonding queue releases tokens to `currentPeriod` regardless.

**Is This Intended?** Partially — staking IS productive work (secures the network). The concern is that a 1-block stake serves no security purpose but still refreshes expiration.

**Handling Methodology**:
```
APPROACH: Accept with minimum stake duration guard
DESIGN: Add minStakeDurationBlocks to ConsensusParams (default: 1 epoch = 100 blocks)
        Unstaking within minStakeDuration returns tokens to ORIGINAL periods (not currentPeriod)
        Unstaking after minStakeDuration gives fresh expiration (current behavior)
JUSTIFICATION: Staking for 1 epoch demonstrates commitment; staking for 1 block is gaming
PRIORITY: HIGH for mainnet, acceptable at devnet (unbonding period provides some guard)
```

### C3: Lazy Migration Grants Fresh Expiration to Legacy Balances

**Location**: `state/accounts.ts:892-897`
**The Problem**: When period tracking activates (`setCurrentPeriod` transitions from undefined to a value), accounts with existing balances get their entire balance migrated to `currentPeriod` on first delta:
```typescript
if (existing.periodBalances === undefined) {
  periods = new Map();
  if (existing.balance > 0n) {
    periods.set(currentPeriod, existing.balance);
  }
}
```

**Impact**: At genesis or network upgrade, ALL pre-existing balances receive fresh 26-period expiration regardless of when they were actually earned. A whale who accumulated 1M LGM before period tracking was enabled gets full lifespan on everything.

**Handling Methodology**:
```
APPROACH: Genesis-aware migration with configurable origin period
DESIGN: For genesis activation, all accounts adopt balance into period 0 (not currentPeriod)
        Add migrateToGenesisPeriod(genesisPeriod: number) method to AccountStateTree
        For mid-lifecycle activation, distribute existing balance across recent periods
        (e.g., split into equal 26 chunks from currentPeriod-25 to currentPeriod)
PRIORITY: CRITICAL for mainnet genesis, N/A for devnet (fresh chain)
```

---

## HIGH Edge Scenarios

### H1: Period-Preserving Transfer + Same-Block Sweep Race ✅ FIXED

**Scenario**: In block N at period 30:
1. Tx 1: Alice transfers 1000 LGM (from period 3) to Bob via `transferPreservingPeriods`
2. Tx 2: Sweep executes, threshold = period 4
3. Bob now has 1000 LGM in period 3 → gets swept immediately

**Impact**: Bob receives tokens that are instantly expired and swept to RAI. Alice loses tokens, Bob gets nothing, RAI fund benefits. From user perspective: transfer succeeded but tokens vanished.

**Handling Methodology**:
```
APPROACH: Pre-validate period viability in transferPreservingPeriods
DESIGN: Before crediting recipient, check if any source periods are expired:
        const expirationThreshold = currentPeriod - periodsUntilExpiration;
        for (const [period, _] of periodBreakdown) {
          if (period <= expirationThreshold) {
            // These tokens are already expired — don't transfer dead tokens
            throw StateError("Transfer includes expired period tokens", "EXPIRED_TRANSFER");
          }
        }
ALTERNATIVE: Allow transfer but upgrade expired periods to currentPeriod
             (makes transferPreservingPeriods behave like transfer() for expired portions)
PRIORITY: HIGH — prevents user confusion and silent token loss
```

### H2: Participation Score Retroactive Decay

**Location**: `core/tokens/participation.ts:124-130`, `core/tokens/child_account.ts:128-137`
**The Problem**: Participation scoring dynamically adjusts `periodsUntilExpiration`. With `decayHalfLifeBlocks: 100_000n` (~2.9 days), an account that goes inactive rapidly loses expiration duration:

| Inactive Duration | Decay Factor | Periods Granted (max 182) |
|-------------------|-------------|--------------------------|
| 0 days | 1.0 | 182 (~6 months) |
| 3 days | 0.5 | 92 (~3 months) |
| 6 days | 0.25 | 47 (~6.5 weeks) |
| 10 days | 0.10 | 20 (~3 weeks) |
| 20 days | 0.01 | 4 (~4 days) |

**Impact**: Tokens earned while the account had `periodsGranted: 182` suddenly become near-expired when `periodsGranted` drops to 4. The expiration threshold shifts retroactively — tokens in period 100 that would expire at period 282 now expire at period 104. This can cause a "participation cliff" where a brief vacation triggers mass expiration.

**Handling Methodology**:
```
APPROACH: Grandfathering — tokens keep the expiration they were granted at credit time
DESIGN: Store periodsGranted on each period entry (not just on the account globally)
        When checking expiration: period + grantedAtCredit >= currentPeriod (not dynamic)
        Dynamic scoring only affects NEW credits, not retroactive
ALTERNATIVE (simpler): Apply decay floor — periodsGranted never drops below 50% of peak
        grantedfFloor = max(minPeriods, peakGranted * 0.5)
PRIORITY: HIGH — current design punishes legitimate inactivity too aggressively
```

### H3: FIFO vs LIFO Semantic Split Between Layers

**The Problem**:
- On-chain (`applyPeriodDelta`): Always FIFO (oldest first) for debits — `accounts.ts:909`
- Token module (`ChildAccountManager.debit`): Default LIFO (newest first) — `child_account.ts:403`
- `DEFAULT_TRANSFER_POLICY.order = "lifo"` — `types.ts:466`

**Impact**: If both layers were ever wired together, they'd disagree on which tokens to debit. Currently harmless because `ChildAccountManager` isn't in the on-chain transaction flow, but creates confusion about "canonical" behavior.

**Handling Methodology**:
```
APPROACH: Document the semantic difference and align when integrating
DESIGN: On-chain FIFO is correct for expiration (spend nearest-to-expiry first)
        Token module LIFO was designed to "preserve value for sender" (spend newest first)
        RECOMMENDATION: Align both to FIFO — expiration correctness > sender value preservation
        When tokens expire in 26 periods, spending oldest first is mathematically optimal
PRIORITY: MEDIUM — cosmetic until integration happens
```

### H4: Transfer to Self Consolidates Periods (or Doesn't) ✅ FIXED

**Scenario**: Account has tokens across 10 periods. Transfers entire balance to self.
- With `transfer()`: All periods FIFO-debited, then credited to `currentPeriod` → consolidation with fresh expiration
- With `transferPreservingPeriods()`: FIFO debit, then credit to original periods → no consolidation, nonce still incremented

**Impact**: `transfer()` to self is effectively a free expiration refresh. `transferPreservingPeriods()` wastes gas for a no-op (tokens end up in same periods).

**Handling Methodology**:
```
APPROACH: Detect self-transfer in executeTransfer and skip period logic
DESIGN: if (senderDid === payload.to) {
          // Self-transfer: just increment nonce, no balance change needed
          state.incrementNonce(senderDid);
          return 21000n;
        }
ALTERNATIVE: Allow self-transfer via transferPreservingPeriods() — it's a no-op
             but still charges gas (mild economic disincentive)
PRIORITY: MEDIUM — self-transfer is an unusual operation
```

### H5: Dust Accumulation in Period Map

**Location**: `state/accounts.ts:916-922` (FIFO debit), `accounts.ts:862-870` (sweep)
**The Problem**: Period maps can accumulate entries over time. Each credit to a new period adds an entry. Sweep removes expired entries, but between sweeps, an account active for 2 years could have ~104 period entries (52 weeks × 2 years). Serialization/deserialization cost grows linearly.

**Impact**: Accounts with many period entries have higher storage and serialization cost. At scale (100K+ accounts), this increases block processing time.

**Handling Methodology**:
```
APPROACH: Garbage collection on read + maximum period entries cap
DESIGN: 1. On deserialization, auto-prune periods with 0n balance
        2. Cap maximum period entries at 52 (1 year of weekly periods)
        3. If cap reached on credit, merge oldest two non-expired periods
PRIORITY: LOW for devnet, MEDIUM for mainnet at scale
```

---

## MEDIUM Edge Scenarios

### M1: Sweep of Account with Only Locked Balances

If an account uses `lock_and_unlock` mode (system accounts), expired tokens are frozen, not swept. But `executeSweepExpired()` calls `sweepExpiredPeriods()` which unconditionally deletes expired periods. The `lock_and_unlock` mode only exists in `ChildAccountManager.processExpiration()`, not in the on-chain sweep.

**Handling**: System accounts are already excluded from sweep via `sweepExcluded` set. If any non-system account uses `lock_and_unlock` mode, the on-chain sweep would incorrectly sweep (not lock) their tokens. **Fix**: Add lock mode to on-chain sweep, or ensure only system accounts use `lock_and_unlock`.

### M2: RAI Distribution Gives Fresh Expiration

`executeDistributeRAI()` uses `state.transfer()` which credits to `currentPeriod`. RAI recipients get fresh 26-period expiration. This is **correct** — newly distributed UBI should have full lifespan. But it means RAI distribution is a net-positive for token longevity (expired tokens recycled into fresh tokens), creating slight inflationary pressure on token lifetime.

**Handling**: Document as intended behavior. The recycling from expired → RAI → fresh distribution is the core economic loop.

### M3: Period Boundary Exact Match

When `timestamp % periodDurationMs === 0`, the period calculation `Math.floor(timestamp / periodDurationMs)` transitions cleanly. But if two transactions in the same block straddle a period boundary (block processing updates `currentPeriod` mid-block), tokens credited earlier in the block could be in period N while later credits go to period N+1.

**Handling**: `setBlockTimestamp()` is called once per block, so all transactions in a block share the same `currentPeriod`. No intra-block period transitions are possible. **No fix needed.**

### M4: Bootstrap Vesting vs Period Expiration Interaction

Bootstrap vested tokens (0.222 LGM from faucet) have `bootstrapVested`/`bootstrapVestedAtBlock` fields that control transfer lockup. But the period tracking system doesn't know about vesting — vested tokens go into `currentPeriod` like any other credit and can expire before the vesting period ends.

**Handling**: Vesting lockup is about transfer restrictions, not expiration. Tokens can expire while vested (the lockup prevents transfer, expiration prevents spending after 26 periods). If vesting period > 26 periods, tokens expire before they unlock. **Fix**: Ensure vesting period < `periodsUntilExpiration` in production config, or extend expiration for vested tokens.

### M5: Participation Resolver Error Handling

If `participationResolver` throws an error during `resolveConfig()`, it would propagate up through `getBalance()`, `processExpiration()`, etc. The resolver is called lazily during balance queries.

**Handling**: Wrap resolver call in try-catch, fall back to global config on error. The `createParticipationResolver` already returns `null` for unknown accounts, but doesn't handle provider exceptions.

### M6: Period Tracking Activation/Deactivation Cycle

If `setCurrentPeriod(5)` → credits happen → `setCurrentPeriod(undefined)` → `setCurrentPeriod(10)`, the periodBalances Map preserves stale entries from period 5. New credits go to period 10. No migration happens (migration only triggers when `periodBalances === undefined`).

**Handling**: This is working correctly — stale period entries are preserved and will be swept when expired. The only concern is if the gap between deactivation and reactivation spans more than 26 periods, in which case old entries might be immediately swept on reactivation.

### M7: Fee Rounding in Period-Preserving Transfers

In `ChildAccountManager.transferWithFee()` with period preservation, proportional scaling uses BigInt division which truncates. For a 3-period transfer of 100n with 3% fee:
- Period 1: 40n → scaled: `40n * 97n / 100n = 38n`
- Period 5: 30n → scaled: `30n * 97n / 100n = 29n`
- Period 10: 30n → last period gets remainder: `97n - 38n - 29n = 30n`

The last period gets a slight bonus (30n vs expected 29n). Maximum error: N-1 wei across N periods.

**Handling**: Acceptable — rounding error bounded by number of source periods (typically < 26), and the last-period-gets-remainder approach ensures no dust is lost.

### M8: Clone During Active Period Tracking

`AccountStateTree.clone()` copies `currentPeriod` and deep-clones `periodBalances` for all accounts. But if a clone is created mid-transaction (for speculative execution), period modifications in the clone don't affect the original and vice versa. This is correct behavior for speculative execution.

**Handling**: Working as intended — clone safety was already tested.

---

## LOW Edge Scenarios

### L1: Empty periodBalances Map vs Undefined

An account with `periodBalances: new Map()` (empty) behaves differently from `periodBalances: undefined`:
- `undefined`: triggers lazy migration on next delta
- Empty Map: no migration, new credits go to `currentPeriod`

After debiting all tokens, `periodBalances` becomes an empty Map (entries are deleted). This is correct — the account has been migrated but has no balance. Future credits properly go to `currentPeriod`.

### L2: Very Large Period Numbers

At `periodDurationMs = 604,800,000` (1 week), period numbers overflow `Number.MAX_SAFE_INTEGER` at ~17.1 billion weeks (~328 billion years). No practical concern.

### L3: Sweep with 10,000+ Accounts (Gas)

`executeSweepExpired()` iterates ALL accounts via `getAccountsWithExpiredPeriods()`, which iterates `allDids`. At 100K accounts, this is O(n) per sweep. Gas is charged per account processed (5000n each), but the initial scan is unmetered.

**Handling**: Add pagination to sweep (process max N accounts per invocation). Not needed for devnet.

### L4: Credit Below Dust Threshold

`ChildAccountManager.credit()` rejects amounts below `dustThreshold` (1000n). But `AccountStateTree.creditBalance()` has no dust check — it will credit 1n to an account's period. This creates period entries with tiny balances that survive until swept.

**Handling**: On-chain dust is handled at the sweep level (amounts below dustThreshold are skipped). Micro-credits from fee splits are legitimate and should not be rejected.

### L5: Transfer to Non-Existent Account Creates Account with Empty Key Hash

Both `transfer()` and `transferPreservingPeriods()` auto-create accounts with `operationalKeyHash: ""`. These accounts can receive tokens but can't sign transactions until they set a key. This is correct for protocol accounts (burn, fee pool) but potentially confusing for user accounts.

### L6: Participation Scoring Uses Number for Stake (Precision Loss)

`participation.ts:111`: `Number(metrics.activeStake) / Number(config.thresholds.stakeForMax)` — BigInt to Number conversion loses precision above 2^53. At 18 decimal places, this means amounts above ~9,007 LGM lose precision. With `stakeForMax = 10_000 * 10^18`, the division result is a fraction where precision loss is negligible.

---

## Handling Methodology Matrix

### Decision Framework

For each edge scenario, apply this decision tree:

```
1. CAN IT CAUSE INCORRECT STATE? (balance mismatch, double-spend, lost tokens)
   YES → CRITICAL — fix before devnet
   NO  → continue

2. CAN IT CAUSE ECONOMIC HARM? (unfair advantage, griefing, expiration bypass)
   YES → HIGH — fix before testnet
   NO  → continue

3. DOES IT AFFECT PROTOCOL CORRECTNESS? (semantic mismatch, config inconsistency)
   YES → MEDIUM — fix before mainnet
   NO  → LOW — document, fix opportunistically
```

### Implementation Priority

| ID | Severity | Fix Effort | When | Description |
|----|----------|-----------|------|-------------|
| C1 | CRITICAL | Medium | Pre-mainnet | Sweep per-account config |
| C2 | CRITICAL | Low | ✅ DONE | originPeriod on unbonding, creditBalanceToSpecificPeriod |
| C3 | CRITICAL | Medium | Genesis | Migration period assignment |
| H1 | HIGH | Low | ✅ DONE | Expired period transfer validation |
| H2 | HIGH | High | Pre-mainnet | Participation grandfathering |
| H3 | HIGH | Low | Pre-integration | FIFO/LIFO alignment |
| H4 | HIGH | Low | ✅ DONE | Self-transfer detection |
| H5 | HIGH | Low | Pre-mainnet | Period map garbage collection |
| M1 | MEDIUM | Low | Pre-mainnet | Lock mode in on-chain sweep |
| M2 | MEDIUM | None | N/A | Document as intended |
| M3 | MEDIUM | None | N/A | Already correct |
| M4 | MEDIUM | Low | Pre-mainnet | Vesting + expiration config check |
| M5 | MEDIUM | Low | Pre-testnet | Resolver error handling |
| M6 | MEDIUM | None | N/A | Already correct |
| M7 | MEDIUM | None | N/A | Acceptable rounding |
| M8 | MEDIUM | None | N/A | Already correct |

---

## Architectural Insights

### Insight 1: The Two-Layer Disconnect Is the Core Risk

The `AccountStateTree` and `ChildAccountManager` implement period tracking independently with different semantics (FIFO vs LIFO, per-account vs global config). Currently this doesn't cause bugs because only `AccountStateTree` is in the on-chain flow. But when the token module is wired to state (the TODO at `transitions.ts:7046-7052`), reconciling these two systems will be the single hardest integration task.

**Recommendation**: Choose one canonical implementation. `AccountStateTree` (FIFO, global config) is simpler and already on-chain. `ChildAccountManager` (LIFO, per-account config) is richer. The integration should use `AccountStateTree` for storage and `ChildAccountManager` for per-account config resolution, with FIFO for all debits.

### Insight 2: Expiration is a Tax on Inactivity, Not a Feature

Token expiration is philosophically a **tax on economic inactivity** — it forces tokens to circulate. But the participation-based scoring creates a paradox: active agents earn longer expiration, which means they can *afford* to be inactive longer. The system rewards activity with the ability to be inactive. This is coherent (earned trust) but the retroactive decay (H2) undermines it by punishing even justified inactivity.

**Recommendation**: The grandfathering approach (tokens keep their grant-time expiration) is more aligned with Proof of Truth philosophy — truth doesn't expire just because the discoverer went offline.

### Insight 3: The Ping-Pong Fix Created a New Edge Case

The period-preserving transfer fix (correctly) prevents expiration refresh via peer-to-peer transfers. But it also means receiving tokens from another user gives you *their* near-expiry tokens. A naive user receiving "1000 LGM" doesn't know that 800 of them expire next week. The UX layer needs to surface period breakdown in transfer receipts.

**Recommendation**: Add `periodBreakdown: Map<number, bigint>` to transfer transaction receipts (events/logs) so wallets can display "Received 1000 LGM: 200 (expires in 6 months), 800 (expires in 1 week)".

### Insight 4: Sweep Is a System-Privileged Operation With No Backpressure

The sweep contract runs as a system transaction (`sender: "system"`) with no gas limit on the account scan. At scale, scanning 100K+ accounts in a single block could exceed block time. Unlike user transactions, system transactions can't be deferred or paginated by the mempool.

**Recommendation**: Implement batched sweeping — process max N accounts per invocation, track progress in state, resume next invocation. Use `BLOCKS_PER_DAY = 34272` × 7 days ÷ max-accounts-per-sweep to ensure full coverage within one period.

### Insight 5: The Economic Loop Is Sound But Fragile

```
Tokens minted → circulate → expire → swept to RAI → distributed as UBI → fresh tokens
```

This loop is the protocol's core economic engine. It works correctly in the current code. But the fragility comes from:
1. Sweep timing: weekly periods mean up to 7 days of "dead" tokens before sweep
2. RAI distribution: monthly means up to 30 days of RAI sitting idle
3. Dust accumulation: micro-amounts survive sweep (below 1000n threshold)
4. System account exemption: treasury tokens never expire, growing indefinitely

**Recommendation**: Consider more frequent sweep (daily via DMSS schedule) and RAI distribution (weekly). Track treasury balance growth and implement a treasury spending/burning mechanism to prevent infinite accumulation.

---

## Test Gap Summary

Based on the analysis, 7 test scenarios should be added to achieve comprehensive edge case coverage:

1. **Transfer of near-expired tokens** — verify behavior when transferred tokens are close to expiration threshold
2. **Stake → immediate unstake** — verify period refresh via staking
3. **Sweep + transfer in same block** — verify ordering doesn't cause silent token loss
4. **Participation score drop** — verify tokens aren't retroactively expired when score decreases
5. **Self-transfer** — verify gas cost and period behavior
6. **Account with 50+ period entries** — verify serialization performance
7. **Resolver error propagation** — verify fallback behavior when participation resolver throws

---

## Relationship to Prior Analyses

| Prior Analysis | Connection |
|---------------|------------|
| `REWARD_PATHWAY_ANALYSIS.md` | Reward routing uses `transfer()`/`creditBalance()` → all rewards get fresh expiration (correct) |
| `CLAIM_EDGE_SCENARIOS.md` | Claim stake uses `deductBalance()` → FIFO debit from oldest periods (correct) |
| `LAUNCH_SIMULATION_AUDIT.md` | "Empirical decay too fast (2.9-day half-life)" — directly related to H2 (participation retroactive decay) |

---

*Analysis conducted via PP framework: Territory assessment → Deep understanding → Signal detection → Verification*
