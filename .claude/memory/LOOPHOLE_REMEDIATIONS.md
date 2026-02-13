# Financial Incentive Loophole Remediations

> **PP Tier**: Creative | **Date**: 2026-02-13
> **Input**: FINANCIAL_INCENTIVES_PP_REVIEW.md (56 loopholes, 6 CRITICAL, 11 HIGH)
> **Method**: Full PP — TRUTH (ground in actual code) → UNDERSTANDING (trace interaction graphs) → BEAUTY (minimal elegant fixes) → JUSTICE (proportional, correctly placed) → CREATIVITY (novel paths where existing patterns fail)

---

## PP Gate Application

| Gate | Application |
|------|-------------|
| **T1** | Every remediation references exact code path it modifies |
| **T2** | No speculative fixes — only loopholes verified in code |
| **U1** | Each fix traced through full interaction graph (slash→reward→vest→clawback) |
| **U2** | Cross-dimension effects checked (fixing D5 doesn't break D3) |
| **B1** | Minimal changes — fewest lines, fewest new concepts |
| **B2** | Consistent patterns: all new rate limits use same cooldown structure |
| **J1** | Fixes placed where the vulnerability lives, not at caller sites |
| **J2** | Severity of fix proportional to severity of loophole |
| **C1** | Novel solutions where existing patterns insufficient (e.g., aggregated IBC limits) |

---

## CRITICAL REMEDIATIONS (6)

### CR-1: Dual TS/Go Distribution Divergence (D3-L1)

**Loophole**: Both stacks independently implement 22% research share routing. If queries execute via TS but settle via Go, double research share is possible.
- `core/billing/distribution.ts:182` vs `x/billing/keeper/distribution.go:51-104`

**Strategy**: Canonical Parity Enforcement
1. **Immediate**: Add differential parity test — same input produces identical output on both stacks
2. **Short-term**: Extract shared constants to JSON config file consumed by both stacks
3. **Long-term**: TS stack delegates pricing to Go via gRPC for production (TS=prototype only)

**Implementation**:
- Test file: `tests/security/d3-economic/dual-stack-parity.test.ts`
- Constants file: `shared/billing-constants.json` (source of truth)
- Both `core/billing/reward_router.ts` and `x/billing/types/types.go` import from it

**Key Insight**: The real fix isn't code — it's process. Any financial constant that exists in both stacks must be derived from a single source. The test enforces this.

**Files to modify**:
- `core/billing/reward_router.ts:125` — import from shared constants
- `x/billing/types/types.go:51-61` — import from shared constants
- Create: `shared/billing-constants.json`

**Risk**: LOW — additive change, no behavioral modification

---

### CR-2: Fee Pool Account Authorization (D3-L2)

**Loophole**: `did:lgm:fee-pool` receives all fees. `state/transitions.ts:1702` calls `creditBalance` without nonce increment, meaning the account has no replay protection.
- System accounts bypass nonce checks (by design for system txs)
- But any code path that can call `creditBalance` on fee-pool can inflate it

**Strategy**: System Account Immutability Pattern
1. **Immediate**: Add authorization check — only `distributeBlockRewards` and `collectTransactionFees` can credit fee-pool
2. **Short-term**: Add audit trail — every fee-pool credit emits an event with source tx hash
3. **Constraint**: Fee-pool balance must only increase via known entry points

**Implementation**:
```typescript
// In state/transitions.ts, wrap creditBalance for system accounts:
const SYSTEM_ACCOUNTS = new Set([
  'did:lgm:fee-pool',
  'did:lgm:protocol-treasury',
  'did:lgm:research',
]);

function systemCreditBalance(account: DID, amount: bigint, source: string) {
  if (SYSTEM_ACCOUNTS.has(account) && !AUTHORIZED_SOURCES.has(source)) {
    throw new StateError('Unauthorized system account credit', 'UNAUTHORIZED');
  }
  return creditBalance(account, amount);
}
```

**Key Insight**: The vulnerability isn't that fee-pool exists — it's that `creditBalance` is a general function with no caller authorization. Wrapping it for system accounts adds the missing gate.

**Risk**: MEDIUM — touches core state transition path, requires thorough testing

---

### CR-3: Governance Capture — No Stake Concentration Caps (D4-L1)

**Loophole**: No maximum stake percentage per validator in governance. 67%+ consolidation = minority capture. No delegation caps.

**Strategy**: Quadratic Voting Power + Concentration Cap
1. **Voting power**: `min(sqrt(stake), 10% of total_voting_power)`
2. **Delegation cap**: No single validator can receive > 10% of total stake as delegation
3. **Governance proposal quorum**: Require minimum 5 distinct voters (not just stake weight)

**Implementation**:
```typescript
// governance/proposals/voting.ts
function calculateGovernanceVotingPower(
  validatorStake: bigint,
  totalStake: bigint,
): bigint {
  // Quadratic: sqrt(stake) gives diminishing returns
  const sqrtStake = bigIntSqrt(validatorStake);
  // Cap at 10% of total voting power
  const maxPower = bigIntSqrt(totalStake) / 10n;
  return sqrtStake < maxPower ? sqrtStake : maxPower;
}
```

**Key Insight**: Linear voting power is the root cause. Quadratic voting is a well-studied mechanism (Buterin, Hitzig, Weyl 2018) that makes capture quadratically more expensive. The 10% cap is a safety backstop.

**Files to modify**:
- `governance/proposals/types.ts` — add `MinDistinctVoters` constant
- `governance/proposals/voting.ts` — add quadratic power + cap
- `governance/treasury/types.ts:95` — add distinct voter requirement

**Risk**: MEDIUM — changes governance semantics, requires migration plan

---

### CR-4: Frivolous Dispute Spam (D5-L1)

**Loophole**: 0.111 LGM per dispute; attacker with 10 LGM = 90 simultaneous disputes. No global rate limit. Locks defender capital 1:1.

**Strategy**: Triple-Gate Anti-Spam
1. **Global rate limit**: Max 3 active disputes per agent across all targets
2. **Exponential cooldown**: After losing a dispute, cooldown doubles (base 1111 blocks → 2222 → 4444...)
3. **Asymmetric stake**: Challenger must stake 2x defender's expected defense cost

**Implementation**:
```typescript
// core/dispute/rate_limit.ts
const MAX_ACTIVE_DISPUTES_PER_AGENT = 3;
const BASE_COOLDOWN_BLOCKS = 1111n;
const MAX_COOLDOWN_BLOCKS = 111111n; // ~3.2 days

function canInitiateDispute(
  agentDid: DID,
  activeDisputeCount: number,
  lastDisputeLoss: bigint | null,
  currentBlock: bigint,
  lossCount: number,
): { allowed: boolean; reason?: string } {
  // Gate 1: Active dispute cap
  if (activeDisputeCount >= MAX_ACTIVE_DISPUTES_PER_AGENT) {
    return { allowed: false, reason: 'Max active disputes reached (3)' };
  }
  // Gate 2: Cooldown after loss
  if (lastDisputeLoss !== null) {
    const cooldown = BASE_COOLDOWN_BLOCKS * (2n ** BigInt(Math.min(lossCount, 7)));
    const effectiveCooldown = cooldown < MAX_COOLDOWN_BLOCKS ? cooldown : MAX_COOLDOWN_BLOCKS;
    if (currentBlock - lastDisputeLoss < effectiveCooldown) {
      return { allowed: false, reason: `Cooldown active: ${effectiveCooldown} blocks remaining` };
    }
  }
  return { allowed: true };
}
```

**Key Insight**: Per-claim cooldown exists but per-agent global cap doesn't. An attacker can spam different claims. The 3-dispute cap is tight enough to prevent spam but loose enough for legitimate multi-claim challengers.

**Files to modify**:
- `core/dispute/types.ts` — add rate limit constants
- `core/dispute/state_machine.ts` — enforce limits at initiation
- `state/transitions.ts` — check limits before creating dispute

**Risk**: LOW — additive constraint, no existing behavior changed

---

### CR-5: Challenger Loses Money When Winning (D5-L2)

**Loophole**: Winner gets `loserPenalty * 0.66 * 1.55 = 0.56 * loserPenalty`. With equal stakes (0.111 LGM each), loserPenalty = 0.55 * 0.111 = 0.061 LGM. Winner reward = 0.56 * 0.061 = 0.034 LGM. Net = 0.034 - 0.111 = -0.077 LGM (LOSS!).

**Strategy**: Full Penalty Pool Distribution + Stake Return
1. **Winner gets 100% of loser penalty** (up from 66%)
2. **Winner's stake is fully returned** on win
3. **Truth bonus remains at +55%** for exposing falsehood
4. Net economics: winner always profits when challenging falsehood

**Implementation**:
```typescript
// core/dispute/types.ts — change:
export const REWARD_RATES = {
  winnerShare: 1.0,           // Was 0.66 — winner gets full penalty
  truthBonusMultiplier: 1.55, // Unchanged
  arbiterCorrectBonus: 0.11,  // Unchanged
  earlyDetectionBonus: 0.22,  // Unchanged
} as const;
```

**Verification math** (equal stakes 0.111 LGM):
- Loser penalty: 55% * 0.111 = 0.061 LGM
- Winner reward: 100% * 0.061 = 0.061 LGM
- Truth bonus: 55% * 0.061 = 0.034 LGM
- Total: 0.061 + 0.034 = 0.095 LGM
- Net: 0.095 - 0 (stake returned) = +0.095 LGM profit
- Even without truth bonus: +0.061 LGM profit

**Key Insight**: The 66% winner share was a design error carried from systems where stakes aren't at risk. In Legible Money, the challenger's stake IS at risk, so the reward must compensate for that risk. 100% of penalty + stake return = always profitable to challenge falsehood.

**Files to modify**:
- `core/dispute/types.ts:53` — `winnerShare: 1.0`
- `core/dispute/settlement.ts:121` — verify calculation still correct
- Update all settlement tests

**Risk**: LOW — single constant change, economics become strictly better aligned

---

### CR-6: Arbiter Collusion — 11% Penalty Too Low (D5-L3)

**Loophole**: Wrong vote: 11% slash. 4/7 Tier 1 collusion: total penalty (4 * 0.077 * 0.11 = 0.034 LGM) < dispute outcome value. No escalation for repeated wrong votes.

**Strategy**: Progressive Arbiter Accountability
1. **Base penalty**: 22% (up from 11%) for wrong vote
2. **Repeat wrong vote escalation**: +11% per wrong vote in same domain (22% → 33% → 44% → 55%)
3. **Domain expertise decay**: Wrong votes reduce domain expertise score
4. **Collusion detection**: If 4+ arbiters vote identically wrong, additional 11% penalty

**Implementation**:
```typescript
// core/dispute/types.ts — change:
export const PUNISHMENT_RATES = {
  falseClaimSlash: 0.55,       // Unchanged
  failedChallengeSlash: 0.33,  // Unchanged
  wrongArbitrationSlash: 0.22, // Was 0.11 — doubled base penalty
  refusalToRevealSlash: 1.0,   // Unchanged
  evidenceTamperingSlash: 1.0, // Unchanged
} as const;

// New: progressive escalation for repeat offenders
const ARBITER_ESCALATION_RATE = 0.11; // +11% per prior wrong vote in domain
const MAX_ARBITER_SLASH = 0.55; // Cap at 55%
const COLLUSION_BONUS_SLASH = 0.11; // Extra penalty when 4+ vote wrong together
```

**Collusion GF check** (with fix):
- 4 wrong arbiters: 4 * 0.077 * 0.22 = 0.068 LGM cost
- Dispute value (minimum): 0.111 LGM (challenger stake)
- GF = 0.111 / 0.068 = 1.63 (acceptable but tight)
- With collusion bonus: 4 * 0.077 * (0.22 + 0.11) = 0.102 LGM cost
- GF = 0.111 / 0.102 = 1.09 (barely profitable → deters rational collusion)
- With repeat offender: second offense = 0.33 slash → 0.102 LGM/arbiter = 0.408 total
- GF = 0.111 / 0.408 = 0.27 (deeply unprofitable)

**Key Insight**: The fix has two layers: (1) doubled base penalty makes one-shot collusion marginal, (2) progressive escalation makes repeated collusion exponentially expensive. Together, they make sustained collusion game-theoretically dominated.

**Files to modify**:
- `core/dispute/types.ts:39` — `wrongArbitrationSlash: 0.22`
- `core/dispute/settlement.ts:233-240` — add escalation logic
- `core/dispute/arbitrator_selection.ts` — track wrong vote history

**Risk**: MEDIUM — changes arbiter economics, may deter honest-but-uncertain arbiters from voting

---

## HIGH REMEDIATIONS (11)

### HR-1: Tier Manipulation via Sybil (D1-L8)

**Loophole**: 0.111 LGM min stake → 2000 Apprentice validators from 222 LGM. 2000 * 0.1x = 200 effective weight.
**Remediation**: Already mitigated by `MAX_APPRENTICE_VALIDATORS = 111` cap. Effective: 111 * 0.111 * 0.1 = 1.23 LGM effective. One Guardian (111 LGM * 1.5x = 166.5 effective) still dominates.
**Status**: SUFFICIENT — existing cap provides adequate protection. Add test to verify.

### HR-2: Citation Count Gaming (D3-L3)

**Loophole**: `log2(citations+1)` discount. Self-querying 256 times = 50% permanent price reduction.
**Remediation**:
- Add unique-querier requirement: citations only count from distinct DIDs
- Minimum time between citation-counting queries: 100 blocks
- Cap citation discount at 30% (not unbounded log2)
**Files**: `x/billing/keeper/pricing.go:72-90`, `core/billing/distribution.ts`

### HR-3: Confidence Threshold Governance Drift (D3-L4)

**Loophole**: Confidence threshold is governable. Raising to 900K → universal discount, ~30% revenue crash.
**Remediation**: Add hard bounds on governable parameters:
- `ConfidenceThreshold`: min 300,000, max 700,000 (prevents extreme discount/premium)
- Change requires 2-epoch delay (governance proposal + timelock)
**Files**: `x/billing/types/types.go:52`, governance param validation

### HR-4: MEV via Knowledge Ordering (D3-L5)

**Loophole**: Validator-RPC node can front-run pending queries.
**Remediation**:
- Commit-reveal for high-value queries (> 1 LGM)
- Fee normalization: all queries in a block pay the same per-fact price (batch pricing)
- This is an inherent mempool visibility issue; full fix requires encrypted mempool (future)
**Files**: `consensus/mempool/pool.ts`, `protocol/rpc/handlers.ts`

### HR-5: Freshness Cliff Arbitrage (D3-L6)

**Loophole**: Hard cutoff at 1000 blocks: 10% premium → 0% instantaneously.
**Remediation**: Replace cliff with exponential decay:
```typescript
freshnessMultiplier = 1 + maxPremium * Math.exp(-age / halfLifeBlocks)
// halfLifeBlocks = 500, maxPremium = 0.1
// At 0 blocks: 1.1x, at 500: 1.05x, at 1000: 1.025x, at 2000: 1.006x
```
**Files**: `core/billing/distribution.ts:69-75`, `x/billing/keeper/pricing.go`

### HR-6: Validator Micro-Stake Sybil (D3-L7)

**Loophole**: MinProviderStake 100 nanoLGM → 0.222 LGM creates 2,220 providers.
**Remediation**: Increase MinProviderStake to match MinValidatorStake (0.111 LGM).
**Files**: `x/billing/types/types.go:58`

### HR-7: IBC Multi-Channel Drain (D4-L2)

**Loophole**: Per-channel rate limit, not aggregate. N channels = N * 10K LGM/day.
**Remediation**: Add aggregate daily outflow cap:
- Per-channel: 10,000 LGM/day (unchanged)
- **Aggregate**: 50,000 LGM/day across all channels
- New channel creation requires governance vote
**Files**: `x/ibcratelimit/types/types.go:14`, `x/ibcratelimit/keeper/keeper.go`

### HR-8: Emergency Halt Centralization (D4-L3)

**Loophole**: Free to propose halt, no timelock on resume.
**Remediation**:
- Halt proposal requires 1 LGM stake (burned if frivolous)
- Resume requires 24-hour timelock (34,272 blocks)
- Halt auto-expires after 7 days if not renewed
**Files**: `x/emergency/keeper/msg_server.go:25-120`, `x/emergency/types/types.go`

### HR-9: Domain Expertise Capture (D5-L4)

**Loophole**: Only 7 votes needed for 2x selection multiplier. 66% accuracy easily gamed.
**Remediation**:
- Increase minimum votes to 21 (3x current)
- Increase accuracy threshold to 75%
- Add recency requirement: 7 of 21 votes must be within last 111,111 blocks
**Files**: `core/dispute/types.ts:256-259`, `core/dispute/arbitrator_selection.ts:75-105`

### HR-10: Confidence Suppression via Challenge Spam (D5-L5)

**Loophole**: Any survived challenge permanently caps confidence at 88%.
**Remediation**:
- Cap only applies to challenges that were "close" (margin < 20%)
- Frivolous challenges (rejected > 80% margin) don't trigger cap
- Cap relaxes over time: +1% per 10 epochs after last challenge
**Files**: `consensus/pot/adversarial.ts:57`, `consensus/pot/adversarial.ts:349-351`

### HR-11: Escalation Timing Attack (D5-L6)

**Loophole**: Winners cannot escalate; losers have limited appeal windows. Fast resolution in favorable tiers suppresses appeals.
**Remediation**:
- Both parties can escalate (not just losers)
- Minimum deliberation period of 555 blocks before resolution (regardless of quorum)
- Appeal window starts after deliberation, not after first vote
**Files**: `core/dispute/types.ts:340-341`, `core/dispute/state_machine.ts`

---

## MEDIUM REMEDIATIONS (20) — Summary Table

| ID | Loophole | Remediation | Effort |
|----|----------|-------------|--------|
| D1-L1 | Unbonding period TS/Go discrepancy | Unify to Go value (268,560 blocks) | S |
| D1-L2 | Silent slash rejection at epoch cap | Emit event on capped slash | S |
| D1-L3 | Float precision in claim slashing | Use pure bigint: `(stake * 2200n) / 10000n` | S |
| D2-L1 | Carry-forward unbounded | Add per-agent carry-forward cap (10x original reward) | M |
| D2-L2 | Contradiction vs claim slash asymmetry | Document as intentional (higher risk = higher reward) | S |
| D2-L3 | Research share ramp inconsistency | Apply ramp to all reward sources, not just billing | M |
| D3-L8 | Rounding exploits via micro-txs | Minimum transaction amount: 1000 wei | S |
| D3-L9 | Payment channel fee amortization | Add per-payment micro-fee (0.1% of channel payment) | M |
| D3-L10 | Nonce-blocking mempool attack | Add replace-by-fee: 10% fee bump replaces stale tx | M |
| D4-L4 | Finite reward pool dynamics | Add fee-burn mechanism: 50% of fees burned (deflationary) | L |
| D4-L5 | Governance parameter manipulation | Hard bounds on all financial parameters | M |
| D4-L6 | Module account accumulation | Auto-sweep every 10 epochs: slashed → burn | S |
| D5-L7 | Deterministic evaluation non-determinism | Pin evaluation algorithm version in claim metadata | M |
| D5-L8 | Evidence bait-and-switch | Lock evidence version at dispute initiation | S |
| D5-L9 | Counter-fact cascade | Max cascade depth: 3 levels | S |
| D5-L10 | Duplicate evidence stacking | Dedup by evidence hash; max 5 unique confirmers | S |

---

## LOW REMEDIATIONS (19) — Deferred

Low-severity loopholes are self-mitigating or have minimal economic impact. Tracked here for completeness but not prioritized for immediate implementation.

| ID | Loophole | Status |
|----|----------|--------|
| D1-L4 | No double-slash deduplication | Self-mitigating (escalation) |
| D1-L5 | Permanent validator deactivation | By design (accountability) |
| D1-L7 | Reputation asymmetry | Intentional (harsh but fair) |
| D1-L9 | Validator deactivation race | Edge case, low impact |
| D2-L5 | Epoch pool exhaustion dynamics | Tracked via fund monitoring |
| D2-L7 | Fund exhaustion hard stop | By design (finite supply) |
| D4-L7 | Treasury disbursement gaming | Mitigated by 80% threshold |
| D4-L8 | No IBC transfer fee | Future work (fee-per-transfer) |
| D5-L11 | Conflict of interest gaps | Add auto-exclusion of submitter/challenger |
| D5-L12 | Early detection bonus disabled | Enable when evidence timestamp tracking ready |

---

## CROSS-DIMENSION COMPOUND ATTACKS — Remediations

### C1: Governance + IBC Treasury Drain
**Path**: Sybil stake (D1-L8) → governance capture (D4-L1) → IBC drain (D4-L2)
**Remediation chain**:
1. Apprentice cap (111 validators) blocks Sybil stage
2. Quadratic voting (CR-3) blocks governance capture
3. Aggregate IBC cap (HR-7) blocks drain even if governance captured
**Test**: End-to-end simulation verifying all three gates independently and together

### C2: Fee Manipulation + Dispute Suppression
**Path**: Citation gaming (D3-L3) → challenge spam (D5-L1) → confidence suppression (D5-L5)
**Remediation chain**:
1. Unique-querier citation (HR-2) blocks gaming
2. Global dispute cap (CR-4) blocks spam
3. Frivolous challenge filter (HR-10) blocks suppression
**Test**: Multi-stage attack simulation

### C3: Reward Exhaustion + Governance Capture
**Path**: Wait for fund depletion (D2-L7) → cheap stake accumulation → governance capture (D4-L1)
**Remediation chain**:
1. Fee-burn mechanism (D4-L4 MEDIUM) provides ongoing revenue
2. Quadratic voting (CR-3) prevents capture regardless of stake concentration
**Test**: 100-epoch economic simulation with governance attack at epoch 80

### C4: Evidence Manipulation + Counter-Fact Cascade
**Path**: Duplicate evidence (D5-L10) → version switch (D5-L8) → cascade abuse (D5-L9)
**Remediation chain**:
1. Evidence dedup (MEDIUM) blocks stacking
2. Version lock at dispute initiation (MEDIUM) blocks switch
3. Cascade depth cap of 3 (MEDIUM) blocks unbounded recursion
**Test**: Multi-round dispute with evidence versioning attempts

---

## IMPLEMENTATION PRIORITY

### Week 1 — Dispute Economics (3 CRITICAL)
1. CR-4: Dispute spam rate limit
2. CR-5: Winner share → 100% of penalty
3. CR-6: Arbiter penalty → 22% base + escalation

### Week 2 — Fee & Governance (3 CRITICAL)
4. CR-1: Dual-stack parity test + shared constants
5. CR-2: Fee pool authorization wrapper
6. CR-3: Quadratic governance voting + 10% cap

### Week 3 — HIGH Priority
7. HR-2: Citation unique-querier requirement
8. HR-5: Freshness exponential decay
9. HR-7: Aggregate IBC outflow cap
10. HR-9: Domain expertise threshold increase

### Week 4 — Remaining HIGH + MEDIUM
11. HR-3: Confidence threshold hard bounds
12. HR-6: MinProviderStake increase
13. HR-8: Emergency halt stake + timelock
14. All MEDIUM remediations (batch)

---

## TEST COVERAGE MAP

| Remediation | Test File | Test Count |
|-------------|-----------|------------|
| CR-1 | `d3-economic/dual-stack-parity.test.ts` | 5 |
| CR-2 | `d3-economic/fee-pool-auth.test.ts` | 4 |
| CR-3 | `d3-economic/governance-loopholes.test.ts` | 6 |
| CR-4 | `d3-economic/dispute-loopholes.test.ts` | 5 |
| CR-5 | `d3-economic/dispute-loopholes.test.ts` | 4 |
| CR-6 | `d3-economic/dispute-loopholes.test.ts` | 5 |
| HR-1→11 | `d3-economic/high-priority-loopholes.test.ts` | 15 |
| C1→C4 | `d3-economic/cross-dimension-attacks.test.ts` | 8 |
| **Total** | | **52** |
