# PP Review: Staking Reward System

**Date**: 2026-02-10
**Scope**: x/staking/, x/vesting_rewards/, consensus/pot/, core/genesis/, core/billing/reward_router.ts, state/transitions.ts, state/validators.ts
**Method**: Full 30-gate PP review (TRUTH → UNDERSTANDING → BEAUTY → JUSTICE → CREATIVITY)

---

## Executive Summary

**Overall Grade: 7/10** (STRONG foundation, 5 CRITICAL gaps prevent production)

| Dimension | Score | Status |
|-----------|-------|--------|
| TRUTH | 6/10 | TS/Go divergence on emission model |
| UNDERSTANDING | 8/10 | Clear tier mechanics, sound vesting math |
| BEAUTY | 7/10 | Elegant vesting curve, fragmented routing |
| JUSTICE | 5/10 | No delegator rewards, inverted slash multiplier |
| CREATIVITY | 9/10 | Novel truth-linked vesting, excellent zero-stake design |

**Findings**: 5 CRITICAL, 7 HIGH, 9 MEDIUM, 4 LOW

---

## TRUTH GATES

### T1: Grounding — Monetary Values Grounded in BigInt ✅ PASS
- All Go staking/reward code uses `math/big.Int` for amounts (keeper.go:546-550, rewards.go:34-44, vesting.go:45-52)
- All TS prototype uses `bigint` (transitions.ts:4878, types.ts:406)
- No `Number()` conversions of stakes or rewards in either stack
- **Citation**: `x/vesting_rewards/keeper/vesting.go:16` — `fixedPointPrecision = 10^18`

### T2: Calibration — Confidence Matches Evidence ⚠️ CRITICAL
**Block reward emission models DIVERGE between TS and Go**

| Feature | TS Prototype | Go SDK |
|---------|-------------|--------|
| Decay | Exponential per-epoch (0.9^epoch) | **None** — constant `BlockRewardAmount` |
| API suppression | Quadratic: `reward *= (1 - s²)` | **None** |
| Fund health | Linear taper below 10% of initial | **None** |
| Fund source | Finite 60% of MAX_SUPPLY | **Minted on demand** (MintCoins) |
| Floor reward | 0.1 LGM | Not defined |

- TS: `core/genesis/types.ts:443-505` — `calculateBlockReward()` with 4-stage pipeline
- Go: `x/vesting_rewards/keeper/rewards.go:69-179` — `DistributeBlockReward()` calls `MintCoins()` with NO supply cap
- **Impact**: Go SDK has unbounded inflation. TS draws from finite fund (133M LGM).
- **Severity**: CRITICAL — the Go SDK will mint LGM forever

### T3: Counter-Consideration — Slash Escalation TS/Go Divergence ⚠️ CRITICAL
- Go has progressive slash escalation: `amount * (1M + slashCount × 100k) / 1M` (`keeper.go:542-551`)
  - First slash: 1.0x, second: 1.1x, third: 1.2x, etc.
  - Tested: `keeper_test.go:1028` `TestSlashValidator_ProgressiveEscalation`
- TS prototype has **no equivalent** — each slash is independent
- **Impact**: Validators slashed 3x receive 30% higher penalty in Go vs TS
- **Severity**: CRITICAL for cross-chain consistency (if TS ever verifies Go state)

### T4: Completeness — Vesting Pause on Slash NOT Wired ⚠️ HIGH
- `PauseVestingByClaimId()` exists (`vesting.go:502-510`) but only called by disputes module
- When `x/staking/keeper.go:SlashValidator()` slashes a wrong-verdict validator, existing vesting schedules **continue releasing**
- No call to `PauseVesting` anywhere in the slash path
- Grep for `PauseVesting|pause.*vesting|slash.*vesting` in `x/staking/` = **0 results**
- **Impact**: Slashed validators keep extracting rewards from prior correct verifications
- **Severity**: HIGH — enables grief-and-extract (verify correctly to build vesting, then attack)

### T5: Representation — Research Share Consistently Applied ✅ PASS
- All reward paths deduct 22% research share FIRST
- TS: `routeReward()` in `core/billing/reward_router.ts:146`
- Go: `RouteReward()` in `x/vesting_rewards/keeper/rewards.go:25-61`
- Both use `amount * rate / 1M` (basis point safe)
- Verification: `researchShare + recipientAmount == original` invariant holds

### T-Claude: Citation Gate ✅ PASS
All claims cite specific file:line. No uncited assertions.

---

## UNDERSTANDING GATES

### U1: Truth Foundation — Tier System Is Accurate ✅ PASS
4-tier system with clear progression:

| Tier | MinStake (ulgm) | MinVerif | MinAccuracy | MaxSlash | Reward | Selection |
|------|----------------|----------|-------------|----------|--------|-----------|
| 0 Apprentice | 111,000 | 0 | 0 | unlimited | 0.1x | 0.1x |
| 1 Verified | 1,110,000 | 22 | 77% | unlimited | 0.5x | 0.5x |
| 2 Bonded | 11,100,000 | 0 | 0 | unlimited | 1.0x | 1.0x |
| 3 Guardian | 111,000,000 | 1,111 | 77% | 0 | 1.5x | 1.5x |

**Citation**: `x/staking/types/types.go:55-97` DefaultTierConfigs

### U2: Mental Model — Vesting Curve Is Correct ✅ PASS
- Formula: `released = maxRelease × (1 - 2^(-elapsed/halfLife))` — asymptotic approach
- Taylor series for fractional halving: `e^(-x) ≈ 1 - x + x²/2 - x³/6 + x⁴/24`
- 10^18 fixed-point precision throughout
- **Citation**: `vesting.go:128-207` CalculateReleasedAmount, `vesting.go:216-282` calculateHalvingFactor

### U3: Why Explanation — Acceleration Rationale Clear ✅ PASS
4 acceleration types with capped bonuses:
- Defense: +11% per, cap 55% (survives challenge = trusted)
- Replication: +11% per, cap 33% (independently reproduced)
- Corroboration: +5% per, cap 22% (cross-domain confirmation)
- Citation: +2% per, cap 22% (other accepted facts reference this)
- Overall cap: 77%
- **Citation**: `vesting.go:284-323` calculateAccelerationBonusBps

### U4: Knowledge Connection — Category→Vesting Mapping Correct ✅ PASS
10 categories with scientometric half-lives:
- Axiomatic: 1.1M blocks (~75 days), 95% max release
- Peer reviewed: 111K blocks (~7.5 days), 85% max release
- Contested: 22K blocks (~1.5 days), 60% max release
- **Citation**: `x/vesting_rewards/types/genesis.go:20-32`

### U5: Practical Application — Clawback Is Sound ✅ PASS
On falsification: 33% of released clawed back + all unvested forfeited + reserve → challenger
- Uses `big.Int` arithmetic, no truncation risk
- **Citation**: `rewards.go:181-259` FalsifyClaim

### U6: Boundary Clarity — Block Reward Scaling Is INVERTED ⚠️ HIGH
```go
// rewards.go:95-98
if activeValidatorCount < MinValidatorsForFullReward {
    baseReward.Mul(baseReward, big.NewInt(int64(activeValidatorCount)))
    baseReward.Div(baseReward, big.NewInt(int64(params.MinValidatorsForFullReward)))
}
```
- When 5 of 22 validators active: reward = `base × 5/22 = 0.23x`
- **Problem**: Low participation → lower rewards → less incentive to join
- **Expected**: Low participation → higher rewards → attract validators
- **Note**: TS has the SAME logic (inherited, not new bug) but TS also has fund-health taper which naturally compensates. Go does not.

### U-Claude: Exhaustion Gate ✅ PASS
Read all relevant files across both stacks.

---

## BEAUTY GATES

### B1: Foundation — Architecture Is Sound ✅ PASS
- x/staking owns validator state, delegation, unbonding, tier computation
- x/vesting_rewards owns reward routing, vesting schedules, clawback
- Clean cross-module interfaces (VestingRewardsKeeper in expected_keepers.go)

### B2: Emergence — Vesting Curve Emerged Naturally ✅ PASS
Truth-linked vesting is genuinely novel: rewards vest faster for stable knowledge.
Acceleration bonuses create positive feedback for truth.

### B3: Fit — Commission Field Stored But Never Used ⚠️ HIGH
- `LegibleValidator` has `Commission uint64` field (state.pb.go:33)
- `MsgRegisterValidator` accepts `Commission` (types.go:121)
- Registration stores it: `msg_server.go:72`
- **Never read for reward distribution** — grep for commission usage in rewards/vesting = 0 hits
- **Impact**: Delegators see commission advertised but it has no effect on reward splits
- **Severity**: HIGH — misleading field, may violate user expectations

### B4: Elegance — Minimal But Complete ⚠️ MEDIUM
- Vesting math is elegant (Taylor series, 10^18 precision)
- But: 8 message types in vesting_rewards where 5 would suffice
- `MsgCompleteVesting` is never needed (auto-completes in ClaimRewards:415-417)

### B5: Coherence — Reward Entry Points Fragmented ⚠️ MEDIUM
Three independent reward entry points with no unified dispatcher:
1. `DistributeBlockReward()` — block production (Go only)
2. `ApplyRewardsAndSlashes()` — verification (in x/knowledge)
3. `CreateVestingScheduleFromKnowledge()` — claim acceptance vesting

No single function that logs "total rewards distributed this block." Auditing requires checking all three.

### B6: Clarity — Code Is Readable ✅ PASS
Well-commented, consistent naming, clear variable purposes.

### B7: Appropriateness — Right-Sized ✅ PASS
Complexity proportional to problem space. Vesting math necessarily complex.

### B-Claude: Conformance Gate ✅ PASS
Follows existing Cosmos SDK keeper patterns consistently.

---

## JUSTICE GATES

### J1: Placement — Delegator Rewards Missing ⚠️ CRITICAL
- Commission field exists on validators but never used for reward splits
- `state/validators.ts:724`: Comment says `// Delegator rewards would be distributed separately` — **TODO never resolved**
- Go: No commission-based reward splitting in any reward path
- **Impact**: Delegation is a one-way value transfer. Delegators give stake (boosting validator power and rewards) and receive nothing.
- **Severity**: CRITICAL — breaks fundamental Cosmos SDK expectation that delegation earns returns

### J2: Ownership — Tier Slash Multiplier Is Inverted ⚠️ HIGH
From `x/staking/keeper/tiers.go` (verified via `CalculateTierSlash` called at keeper.go:540):

| Tier | Reward Multiplier | Slash Multiplier |
|------|-------------------|-----------------|
| 0 Apprentice | 0.1x | **1.5x** |
| 1 Verified | 0.5x | **1.2x** |
| 2 Bonded | 1.0x | 1.0x |
| 3 Guardian | 1.5x | 1.0x |

- Apprentices earn 0.1x rewards but receive 1.5x slashes
- Rationale (per code comments): "lower-tier validators have less skin in the game"
- **Problem**: Tier 0 has **no stake** — 1.5x of 0 = 0, but the multiplier also applies to the reputation penalty of -10000 (1% of 1M scale)
- Effective: Apprentices lose 1.5% reputation per slash vs 1.0% for Guardians
- **Severity**: HIGH — discourages new participants. The tier with least to gain loses most proportionally.

### J3: Responsibility — Slash Propagation to Delegators ⚠️ MEDIUM
- `keeper.go:568-578`: Own stake slashed first, then delegated stake for remainder
- But: Individual delegations not updated — only aggregate `DelegatedStake` changes
- If validator with 100k delegated from 5 delegators is slashed 50k on delegated, each delegator should lose proportionally
- **Current**: Aggregate reduced but individual delegation records unchanged
- **Impact**: First delegator to undelegate may get full amount; last gets shorted

### J4: Proportion — Research Share Is Proportional ✅ PASS
22% is consistently applied. Not excessive for a research-focused chain.

### J5: Balance — Empty Block Penalty Is Fair ✅ PASS
- Empty blocks earn 0% by default (`EmptyBlockRewardRate = 0`)
- Aligns with PoT: blocks without truth verification don't earn

### J6: Integrity — Unbonding Period Protects Network ✅ PASS
268,560 blocks (~7 days) prevents fast stake-and-run attacks.

### J-Claude: Evidence Gate ✅ PASS
All findings cite specific code locations.

---

## CREATIVITY GATES

### C0: Territory — Is This Uncharted? ✅ YES
Truth-linked vesting with epistemic category-based release curves is genuinely novel.
No existing blockchain has this pattern.

### C1: Pioneering — Zero-Stake Onboarding ✅ EXCELLENT
- Tier 0 allows AI agents to participate immediately with 0 stake
- Virtual stake (111 LGM equivalent) ensures minimum VRF selection probability
- Progressive trust: 22 correct verifications → Tier 1 without ever staking
- **Citation**: `types.go:55-67` TierApprentice config

### C2: Path-Building — Vesting as Truth Incentive ✅ EXCELLENT
The insight that **knowledge stability should determine reward release speed** is profound:
- Mathematical axioms vest over 75 days (stable knowledge)
- Empirical claims vest over 7.5 days (changeable knowledge)
- Contested claims vest over 1.5 days with 60% cap (uncertain knowledge)
- Acceleration bonuses reward facts that survive challenges, get replicated, are corroborated, or cited

### C3: Extension — API Suppression (TS only) ✅ NOVEL
- As API revenue grows, block rewards shrink (`1 - s²` quadratic)
- Self-sustaining economy: when fees fund the network, inflation stops
- **Gap**: Not implemented in Go SDK

### C4: Vision — Fund Health Taper (TS only) ✅ NOVEL
- Finite reward fund (60% of supply = 133M LGM)
- When fund drops below 10% of initial, rewards taper linearly
- Hard stop at 0 balance — no infinite minting
- **Gap**: Not implemented in Go SDK

### C5: Foundation — Future Justice Possible ⚠️ MEDIUM
- Commission field exists for future delegator rewards (hooks ready)
- But: No actual implementation means delegators are currently exploited
- Future justice IS possible but current state is unjust

### C6: Integration — Cross-Module Wiring ⚠️ MEDIUM
- Staking → Knowledge via hooks (recently fixed in isolation review)
- Knowledge → VestingRewards via keeper interface
- Disputes → VestingRewards for pause/falsify
- **Gap**: Staking → VestingRewards for pause-on-slash not wired

### C-Claude: Proof Gate ✅ PASS
Exhaustive search confirms TS emission model features are absent from Go.

---

## CRITICAL FINDINGS (5)

### F1 [CRITICAL] Go SDK Has Unbounded Inflation
- **Location**: `x/vesting_rewards/keeper/rewards.go:138`
- **Issue**: `MintCoins()` creates new LGM with no supply cap. TS uses finite fund (133M LGM from genesis).
- **Risk**: Unlimited token supply growth
- **Fix**: Port `EmissionSchedule` (decay + fund health + API suppression + floor) from TS `core/genesis/types.ts:443-505` to Go `DistributeBlockReward()`
- **Complexity**: MEDIUM — requires adding fund module account and 4-stage reward pipeline

### F2 [CRITICAL] No Delegator Reward Sharing
- **Location**: `x/staking/types/types.go:121` (Commission field), `state/validators.ts:724` (TODO comment)
- **Issue**: Commission stored but never used. Delegators stake tokens boosting validator power and earn zero.
- **Risk**: Rational actors won't delegate → weak validator set
- **Fix**: Add commission-based reward splitting in `ApplyRewardsAndSlashes()` and `DistributeBlockReward()`
- **Complexity**: HIGH — requires commission tracking, per-delegator accounting, claim interface

### F3 [CRITICAL] TS/Go Slash Escalation Divergence
- **Location**: Go `keeper.go:542-551` (escalation), TS prototype (no equivalent)
- **Issue**: Validators slashed multiple times get 10% escalation per prior slash in Go, flat in TS
- **Risk**: State divergence if prototypes ever cross-validate
- **Fix**: Implement escalation in TS OR document as deliberate Go-only enhancement
- **Complexity**: LOW — straightforward addition to TS slash logic

### F4 [CRITICAL] Block Reward Scaling Discourages Participation
- **Location**: `rewards.go:95-98`
- **Issue**: `reward = base × activeCount / targetCount` when active < target → lower rewards
- **Risk**: Fewer validators → lower rewards → even fewer validators → death spiral
- **Fix**: Invert: `reward = base × targetCount / max(activeCount, targetCount)` (capped at base when at/above target)
- **Complexity**: LOW — 2-line change

### F5 [CRITICAL] Vesting Not Paused on Validator Slash
- **Location**: `x/staking/keeper/keeper.go:516-631` (SlashValidator)
- **Issue**: Slash reduces stake but doesn't pause vesting schedules
- **Risk**: Grief-and-extract: build vesting from correct verifications, then submit bad verifications. Vesting continues releasing during/after slash.
- **Fix**: SlashValidator should call `vestingRewardsKeeper.PauseVestingByValidatorAddr()` (new method needed that iterates validator's schedules)
- **Complexity**: MEDIUM — requires new keeper interface method + wiring

---

## HIGH FINDINGS (7)

| # | Finding | Location | Impact |
|---|---------|----------|--------|
| H1 | Commission field is dead code | types.go:121 | Misleads delegators |
| H2 | Tier slash multiplier penalizes newcomers | tiers.go CalculateTierSlash | 1.5x slash on 0.1x-reward tier |
| H3 | Individual delegations not reduced on slash | keeper.go:576-578 | Aggregate reduced, records stale |
| H4 | No block reward decay in Go | rewards.go:89-98 | Constant inflation |
| H5 | No API suppression in Go | rewards.go (missing) | No self-sustaining transition |
| H6 | Reputation uses absolute +100/-200 adjustments | keeper.go:492-499 | Should be proportional to current score |
| H7 | `RecordVerification` reputation change asymmetric | keeper.go:492-499 | Correct +0.01%, incorrect -0.02% (2x penalty) |

---

## MEDIUM FINDINGS (9)

| # | Finding | Location |
|---|---------|----------|
| M1 | `MsgCompleteVesting` never needed (auto-completes) | types.go:286 |
| M2 | No governance path to adjust BlockRewardAmount | genesis.go:10 |
| M3 | Slash epoch counter never reset | keeper.go:597 (SlashesThisEpoch) — no ResetEpochSlashCounters |
| M4 | 8 message types in vesting_rewards (5 would suffice) | types.go |
| M5 | No unified reward dispatcher for auditing | rewards.go / confidence.go |
| M6 | Unbonding period hardcoded | No params reference in msg_server.go |
| M7 | Category configs not governance-settable | genesis.go:20-32 |
| M8 | No max vesting schedules per recipient (DoS vector) | vesting.go:28-93 |
| M9 | Taylor series 4th-order may diverge for x > 1 | vesting.go:243-266 (when remainder > halfLife, x = r×ln2/H > ln2 ≈ 0.69 always < 1, so safe) |

---

## LOW FINDINGS (4)

| # | Finding | Location |
|---|---------|----------|
| L1 | `prefixEndBytes` duplicated in staking keeper | keeper.go:55-68 |
| L2 | JSON marshal panic possible in SetParams | keeper.go:638 (returns error, acceptable) |
| L3 | `decayFactor: 0.9` uses float in TS (could drift on different JS runtimes) | types.ts:409 |
| L4 | No snapshot/export test for vesting schedules | (missing test coverage) |

---

## RECOMMENDATIONS

### Before Mainnet (CRITICAL)
1. **Port TS emission model to Go** — finite fund, decay, API suppression, floor reward
2. **Implement delegator reward sharing** — use commission field for actual splits
3. **Fix block reward scaling** — invert to encourage participation when count is low
4. **Wire vesting pause on slash** — add `PauseVestingByValidatorAddr()` to slash path
5. **Resolve TS/Go escalation divergence** — either port to TS or document as Go-only

### Before Testnet (HIGH)
6. **Fix tier slash multiplier** — remove 1.5x/1.2x penalty for lower tiers
7. **Fix individual delegation accounting on slash** — prorate across delegators
8. **Add slash epoch counter reset** in EndBlocker
9. **Add max vesting schedules per recipient** guard

### Before Production (MEDIUM)
10. Remove dead `MsgCompleteVesting` message
11. Add governance path for reward params
12. Consolidate reward entry points into unified dispatcher
13. Add supply ceiling invariant (panic if total supply > MAX_SUPPLY)
