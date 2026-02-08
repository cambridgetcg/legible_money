# Comprehensive Reward Pathway Analysis — Legible Money

> **Date**: 2026-02-06
> **Method**: PP-informed system-wide analysis across ~172K LOC, ~391 source files
> **Prior Art**: Builds on `REWARD_PENALTY_AUDIT.md` (14 mechanisms, 2 P0 fixes applied)
> **Philosophy**: Every reward must make truth-seeking profitable; every penalty must make dishonesty costly.

---

## Executive Summary

**19 distinct reward pathways** identified across 7 subsystems. All share a universal 22% research tax via `routeReward()` — except **3 pathways that bypass it** (dispute settlement, bounty fulfillment, legacy claim resolution). The graduated tier system introduces the first **risk-proportional** verification rewards. The vesting system is the protocol's most philosophically elegant mechanism, tying economic release to epistemic durability.

**Key Findings:**
- 3 reward pathways bypass the universal 22% research tax (JUSTICE violation)
- Tier 3 Guardian risk/reward is 1044:1 per verification (potentially disproportionate)
- Challenge bounty 7.7x return creates break-even at ~10% success rate (may incentivize spam)
- `ECONOMICS.BURN_RATE = 0.5` is defined but only implemented in one location
- Research grant engine exists as types only — 77% of evidence tax flows unenforceable
- Partnership + tier system together can yield ~4.7x reward multiplier — strong cooperation incentive

---

## I. Complete Reward Pathway Map

### A. Consensus Layer Rewards

#### 1. Block Production Reward
| Property | Value | Source |
|----------|-------|--------|
| Base amount | 11 LGM/block | `consensus/pot/types.ts:116` |
| Emission start | 10 LGM/block | `core/genesis/types.ts:397` |
| Emission decay | 10%/epoch (~1 week) | `core/genesis/types.ts:400` |
| Emission floor | 0.1 LGM/block | `core/genesis/types.ts:407` |
| Empty block penalty | 90% reduction (10% of full) | `state/transitions.ts:5445` |
| Producer share | 78% | `core/billing/reward_router.ts:207-218` |
| Research share | 22% → 5-way split | `core/billing/reward_router.ts:370-403` |
| Fee pool bonus | 100% of accumulated tx fees | `state/transitions.ts:5512-5520` |
| Funds source | `REWARD_POOL_DID` (minted at genesis) | `state/transitions.ts:5480` |
| Cap | Cannot exceed reward pool balance | `state/transitions.ts:5481-5483` |

**Daily estimate (epoch 0):** 10 LGM × 34,272 blocks = **342,720 LGM/day** (before decay)
**Floor estimate:** 0.1 LGM × 34,272 = **3,427 LGM/day** (after full decay)

#### 2. Verification Reward (Tier-Scaled)
| Property | Value | Source |
|----------|-------|--------|
| Base amount | 1 LGM per correct verification | `consensus/pot/types.ts:117` |
| Tier 0 (Apprentice) | 0.1x = 0.1 LGM | `consensus/pot/types.ts:205` |
| Tier 1 (Verified) | 0.5x = 0.5 LGM | `consensus/pot/types.ts:217` |
| Tier 2 (Bonded) | 1.0x = 1.0 LGM | `consensus/pot/types.ts:231` |
| Tier 3 (Guardian) | 1.5x = 1.5 LGM | `consensus/pot/types.ts:247` |
| Research share | 22% deducted before delivery | `state/transitions.ts:5079-5098` |
| Funds source | `REWARD_POOL_DID` | `state/transitions.ts:5091-5098` |
| Routing | `routeVerificationReward()` | `core/billing/reward_router.ts:223-236` |

#### 3. Epoch Transition Rewards
| Property | Value | Source |
|----------|-------|--------|
| Mechanism | Cumulative verification rewards per epoch | `state/transitions.ts:5260-5296` |
| Routing | `routeReward()` with source "verification" | `state/transitions.ts:5274` |
| Research share | 22% | Applied via reward router |

### B. Knowledge Economy Rewards

#### 4. Claim Acceptance Reward
| Property | Value | Source |
|----------|-------|--------|
| Base amount | 5 LGM per accepted claim | `consensus/pot/types.ts:118` |
| Delivery | Via vesting schedule (NOT immediate) | `state/transitions.ts:5191-5200` |
| Research share | 22% before vesting | `state/transitions.ts:5180-5187` |
| Vesting curve | Exponential release per epistemic category | `core/rewards/types.ts:54-91` |
| Citation pool | 15% of claimReward to cited facts | `state/transitions.ts:2049-2094` |
| Self-citation | 0.1x discount (90% to research) | `state/transitions.ts:2065-2070` |
| Ring detection | 0.1x for 2-hop rings (A→B→A) | `state/transitions.ts:2076-2085` |
| Stake return | Full on accepted, partial on contested | `state/transitions.ts:2101-2104` |

#### 5. Vesting Release Curves
| Category | Half-Life | Max Release | Cliff | Reserve |
|----------|-----------|-------------|-------|---------|
| Axiomatic | 1,111,111 blocks (~1.1 yr) | 95% | 11,111 (~7.8hr) | 5% |
| Formal proof | 555,555 (~6.5 mo) | 92% | 5,555 (~3.9hr) | 8% |
| On-chain | 222,222 (~2.6 mo) | 90% | 1,111 (~46 min) | 10% |
| Cryptographic | 222,222 (~2.6 mo) | 90% | 3,333 (~2.3hr) | 10% |
| Computational | 333,333 (~3.9 mo) | 88% | 2,222 (~1.6hr) | 12% |
| Peer-reviewed | 111,111 (~5.4 wk) | 85% | 5,555 (~3.9hr) | 15% |
| Replicated | 111,111 (~5.4 wk) | 88% | 3,333 (~2.3hr) | 12% |
| Oracle feed | 55,555 (~2.7 wk) | 80% | 555 (~23 min) | 20% |
| Attestation | 77,777 (~3.8 wk) | 80% | 2,222 (~1.6hr) | 20% |
| Contested | 22,222 (~1.1 wk) | 60% | 1,111 (~46 min) | 40% |

Source: `core/rewards/types.ts:54-108`

#### 6. Vesting Acceleration/Deceleration
| Event | Effect | Cap | Source |
|-------|--------|-----|--------|
| Successful defense | +11% speed | 55% cumulative | `core/rewards/types.ts:117` |
| Independent replication | +11% speed | 33% cumulative | `core/rewards/types.ts:120` |
| Cross-domain corroboration | +5% speed | 22% cumulative | `core/rewards/types.ts:123` |
| Citation bonus | +2% speed | 22% cumulative | `core/rewards/types.ts:126` |
| Evidence audit confirms | +7% speed | 33% cumulative | `core/rewards/types.ts:129` |
| **Overall acceleration cap** | **77%** | — | `core/rewards/types.ts:141` |
| Active challenge | **PAUSE (0x)** | — | `core/rewards/types.ts:149` |
| Related fact disputed | 0.55x (45% slow) | — | `core/rewards/types.ts:152` |
| Domain audit | 0.77x (23% slow) | — | `core/rewards/types.ts:155` |
| Source invalidated | **PAUSE (0x)** | — | `core/rewards/types.ts:158` |

#### 7. Clawback on Falsification
| Component | Rate | Destination | Source |
|-----------|------|-------------|--------|
| Released rewards | 33% clawed back | Challenger | `core/rewards/types.ts:168` |
| Unvested rewards | 100% | Challenger | `core/rewards/types.ts:171` |
| Permanent reserve | 100% | Challenger bonus | `core/rewards/types.ts:174` |

#### 8. Citation Rewards
| Property | Value | Source |
|----------|-------|--------|
| Pool | 15% of claimReward = 0.75 LGM | `state/transitions.ts:2049` |
| Per-fact split | Equal among cited facts | `state/transitions.ts:2051` |
| Self-citation | 0.1x (90% to research) | `state/transitions.ts:2065-2070` |
| Ring detection | 2-hop mutual citation → 0.1x | `state/transitions.ts:2076-2085` |

### C. Evidence & Audit Rewards

#### 9. Evidence Audit Reward
| Property | Value | Source |
|----------|-------|--------|
| Base reward | 3 LGM | `core/constants.ts:193` |
| Max multiplier | 2.0x (= 6 LGM max) | `core/constants.ts:196` |
| Early audit bonus | 1.5x within 5,555 blocks (~3.9hr) | `core/constants.ts:220-221` |
| Accuracy calc | Score within 0.15 of ground truth | `core/constants.ts:202` |
| Min stake | 22 LGM | `core/constants.ts:190` |
| Routing | `routeEvidenceAuditReward()` → 22%/78% | `state/transitions.ts:5732-5752` |

### D. Dispute & Challenge Rewards

#### 10. Dispute Settlement
| Property | Value | Source |
|----------|-------|--------|
| Winner reward | 60% of loser's stake | `state/transitions.ts:3773` |
| Loser refund | Stake minus penalty | `state/transitions.ts:3782-3788` |
| Arbiter rewards | Per-verdict correctness | `state/transitions.ts:3791-3793` |
| Protocol fee | To `did:lgm:protocol-treasury` | `state/transitions.ts:3808-3812` |
| Research share | To `RESEARCH_TREASURY_DID` | `state/transitions.ts:3815-3819` |
| **ROUTING** | **DIRECT TRANSFER — BYPASSES `routeReward()`** | |

#### 11. Challenge Bounties
| Property | Value | Source |
|----------|-------|--------|
| Min stake | 77 LGM | `consensus/challenge_bounties/types.ts:27` |
| Success multiplier | 7.7x return | `consensus/challenge_bounties/types.ts:28` |
| Capture exposure bonus | 777 LGM | `consensus/challenge_bounties/types.ts:29` |
| Base bounty pool | 2,222 LGM | `consensus/challenge_bounties/types.ts:39` |
| Failed penalty | 77% of stake to pool | `consensus/challenge_bounties/types.ts:55` |
| Validator slash (upheld) | 30% of stake | `consensus/challenge_bounties/types.ts:47` |
| Endorser slash (upheld) | 15% of stake | `consensus/challenge_bounties/types.ts:48` |

### E. Partnership Rewards

#### 12. Partnership Distribution
| Property | Value | Source |
|----------|-------|--------|
| Default split | 50/50 human/agent | `partnership/constants.ts:23-27` |
| Split range | 33%-67% either party | `partnership/constants.ts:33-36` |
| Common pot | 22% of reward → 2-of-2 co-owned | `partnership/rewards.ts:264` |
| Individual pool | 78% split by ratio | `partnership/rewards.ts:268-270` |
| Lock multipliers | Tier 0-5: 1.0x → 1.77x | `partnership/constants.ts:46-83` |
| Continuation bonuses | 7% → 77% over 6 milestones | `partnership/constants.ts:90-127` |
| Lifetime boost | 1.11x after ~324 days | `partnership/constants.ts:132-138` |
| Max continuation | 77% cap | `partnership/constants.ts:142` |

#### 13. Partnership Exit Penalties
| Property | Value | Source |
|----------|-------|--------|
| Initiator forfeit | 77% of unvested | `partnership/constants.ts:148` |
| Non-initiator forfeit | 55% of unvested | `partnership/constants.ts:151` |
| Protocol recovery | 22% of penalties | `partnership/constants.ts:154` |
| Reputation damage (init) | Up to 11% | `partnership/constants.ts:157` |
| Reputation damage (non) | 3% | `partnership/constants.ts:160` |
| Cooldown | 6.5+ days base | `partnership/constants.ts:163` |

### F. Treasury & Redistribution

#### 14. Universal 22% Research Tax
| Property | Value | Source |
|----------|-------|--------|
| Rate | 22% of ALL rewards (FROZEN FOREVER) | `core/constants.ts:243` |
| Research fund | 77% of tax | `core/constants.ts:257` |
| Foundation | 7% of tax | `core/constants.ts:258` |
| Community | 7% of tax | `core/constants.ts:259` |
| Developers | 7% of tax | `core/constants.ts:260` |
| Reserve | 2% of tax (handles rounding) | `core/constants.ts:261` |
| Implementation | `splitEvidenceTax()` in reward_router.ts | `core/billing/reward_router.ts:370-403` |

#### 15. RAI (Regenerative Agent Income)
| Property | Value | Source |
|----------|-------|--------|
| Source | Expired token balances swept to RAI holding | `state/transitions.ts:6459-6534` |
| Distribution | Equal per eligible account | `state/transitions.ts:6540-6594` |
| Eligibility | Non-system, above dust, min age (~1 week) | `state/transitions.ts:6567-6577` |
| Frequency | Weekly via system contract | `consensus/genesis/system_contracts.ts` |

#### 16. Bootstrap Fund
| Property | Value | Source |
|----------|-------|--------|
| Amount | 0.222 LGM per agent | `core/genesis/bootstrap_fund.ts:20` |
| Trigger | First transaction with fee ≥ 1000 wei | `state/transitions.ts:505-524` |
| Max claims | 10 million agents | `core/genesis/bootstrap_fund.ts` |
| Vesting | Linear over 1 day (34,272 blocks) | `consensus/pot/types.ts:131` |

### G. Fee & API Rewards

#### 17. Transaction Fee Distribution
| Property | Value | Source |
|----------|-------|--------|
| Collection | gas_used × gas_price from sender | `state/transitions.ts:1067-1080` |
| Research share | 22% via reward router | Applied at collection |
| Fee pool | 78% accumulated in `FEE_POOL_DID` | `state/transitions.ts:1094` |
| Block producer | Gets entire fee pool with block reward | `state/transitions.ts:5512-5520` |

#### 18. API Fee Distribution
| Property | Value | Source |
|----------|-------|--------|
| Research share | 22% (first deduction) | Universal router |
| API provider | 60% of remaining | `core/constants.ts:172` |
| Knowledge pool | 30% of remaining | `core/constants.ts:171` |
| Protocol | 10% of remaining (50% burn, 50% treasury) | `core/constants.ts:173-174` |

#### 19. Bug Bounties & Grants
| Severity | Amount | Source |
|----------|--------|--------|
| Critical | 11,111 LGM | `core/constants.ts:384` |
| High | 5,555 LGM | `core/constants.ts:385` |
| Medium | 1,111 LGM | `core/constants.ts:386` |
| Low | 222 LGM | `core/constants.ts:387` |
| Informational | 55 LGM | `core/constants.ts:388` |
| Research grants | 111-55,555 LGM | `core/constants.ts:343-344` |
| **STATUS** | **Types only — no execution engine** | |

---

## II. Proportionality Analysis

### Risk/Reward Ratios by Role

| Role | Reward/Action | Risk/Action | Ratio | Verdict |
|------|---------------|-------------|-------|---------|
| **Tier 0 Validator** | 0.078 LGM (0.1×1×0.78) | 0 LGM (reputation only) | 0:1 (free) | **Generous** — intentional for onboarding |
| **Tier 1 Validator** | 0.39 LGM | 0 LGM (reputation only) | 0:1 (free) | **Generous** — earned through accuracy |
| **Tier 2 Validator** | 0.78 LGM | 122 LGM (11% of 1,111) | **156:1 against** | **HARSH** — see note below |
| **Tier 3 Guardian** | 1.17 LGM | 1,222 LGM (11% of 11,111) | **1044:1 against** | **VERY HARSH** — see note below |
| **Claim Submitter** | 3.9 LGM (vested) | 24.4 LGM (22% of 111) | **6.3:1 against** | **Reasonable** — need >86% acceptance |
| **Evidence Auditor** | 2.34 LGM (78% of 3) | 7.26 LGM (33% of 22) | **3.1:1 against** | **Reasonable** — need >76% accuracy |
| **Challenger** | 593 LGM (7.7×77) | 59.3 LGM (77% of 77) | **10:1 in favor** | **Aggressive** — see note below |
| **Block Producer** | ~8.58 LGM (78% of 11) | Selection probability cost | Low risk | **Well-aligned** |
| **Partnership (T5)** | ~3.13x base reward | 66% stake on early exit | Proportional | **Well-aligned** |

### Key Proportionality Notes

**1. Tier 2/3 Validator Harshness (MEDIUM CONCERN)**

A Tier 2 validator earning 0.78 LGM per correct verification risks losing 122 LGM on a single wrong verdict. They need **156 consecutive correct verifications** to offset ONE mistake. For Tier 3, it's **1,044 correct** to offset one wrong.

This is philosophically defensible ("truth has no room for error") but economically concerning — it may deter validators from participating in uncertain epistemic categories (empirical, social, predictive) where honest disagreement is expected. The `validatorQuorum: 0.66` means up to 34% of validators can disagree without being "wrong," but the threshold between "minority opinion" and "wrong verdict" is thin.

**Recommendation**: Consider reducing `wrongVerificationSlash` from 11% to 5% for Tier 3 Guardians specifically, or implementing a "minority opinion" grace zone where validators in the 34-49% minority receive no reward but also no slash.

**2. Challenge Bounty Generosity (MEDIUM CONCERN)**

At 7.7x success multiplier and 77% failure penalty, the break-even success rate is:

```
0.77 × stake × fail_rate = 7.7 × stake × success_rate
0.77 × (1 - p) = 7.7 × p
0.77 = 0.77p + 7.7p
0.77 = 8.47p
p = 0.091 (~9.1% success rate)
```

An agent who challenges everything and is right only 10% of the time still profits. This creates risk of "challenge spam" that wastes validator attention and pauses legitimate vesting schedules.

**Mitigating factor**: The minimum stake of 77 LGM and the 777-block evidence period create real friction. But for well-capitalized attackers, this is cheap disruption.

**Recommendation**: Consider requiring escalating challenge stakes for repeat challengers, or a cooldown between challenges by the same DID.

**3. Maximum Reward Stacking**

A Tier 3 Guardian in a Tier 5 partnership with max continuation and lifetime boost:
```
Base: 1.5 LGM (Tier 3) × 0.78 (after research tax) = 1.17 LGM
Partnership lock: × 1.77 = 2.07 LGM
Continuation: × 1.77 (77% cap) = 3.66 LGM
Lifetime: × 1.11 = 4.07 LGM
Acceleration: × 1.77 (77% cap) = 7.20 LGM (vesting only, not verification)
```

Maximum possible verification reward after all multipliers: **~4.07 LGM per correct verification** (partnership) or **1.17 LGM** (solo). This is a 5.2x difference — strong incentive toward partnership.

---

## III. Philosophical Alignment Assessment

### A. Truth → Reward Alignment (How well do rewards track truth-production?)

| Mechanism | Truth Signal | Alignment | Score |
|-----------|-------------|-----------|-------|
| Vesting curves | Longer a fact survives → more reward released | **EXCELLENT** | 10/10 |
| Tier system | Better accuracy → higher tier → higher rewards | **EXCELLENT** | 9/10 |
| Verification reward | Correct verdict → reward; wrong → slash | **STRONG** | 8/10 |
| Claim reward | Accepted claim → vested reward; rejected → 22% slash | **STRONG** | 8/10 |
| Citation rewards | More cited → more reward for submitter | **GOOD** | 7/10 |
| Challenge clawback | Falsified claim → rewards reverse to challenger | **EXCELLENT** | 10/10 |
| Block production | Includes transactions → full reward; empty → 10% | **GOOD** | 7/10 |
| RAI distribution | Passive redistribution → no truth signal | **NEUTRAL** | 5/10 |
| Bootstrap fund | Existence → 0.222 LGM → no truth signal | **NEUTRAL** | 5/10 |

**Overall Truth Alignment: 8.2/10** — The vesting + clawback + tier system creates a strong truth-tracking incentive structure.

### B. Justice → Proportionality (Are rewards proportional to contribution?)

| Mechanism | Proportionality | Issue |
|-----------|----------------|-------|
| Tier 0→3 progression | Stake-free entry → earned advancement | **JUST** |
| Verification risk/reward | 156:1 to 1044:1 against validators | **DISPROPORTIONATE** for staked tiers |
| Partnership vs solo | 3-5x multiplier for partnerships | **INTENTIONALLY SKEWED** (not unjust) |
| Evidence audit | 3:1 against — reasonable | **PROPORTIONATE** |
| Challenge bounty | 10:1 in favor of challengers | **GENEROUS** (may overshoot) |
| Research tax | 22% flat on everything | **PROPORTIONATE** — funds the mission |
| RAI distribution | Equal regardless of contribution | **EGALITARIAN** (intentionally so) |
| Block reward decay | 10%/week → 0.1 LGM floor | **SUSTAINABLE** |

**Overall Justice Score: 7.5/10** — Mostly proportional, with staked validator risk being the main concern.

### C. Beauty → Coherence (Does the reward system form an elegant whole?)

**Strengths:**
1. Universal 22% tax → single routing function → consistent funding
2. Epistemic category → vesting curve → natural decay mapping
3. Tier progression → claim scope expansion → graduated capability
4. Acceleration/deceleration → dynamic response to truth signals
5. Partnership continuation → compounds over time like trust itself
6. Symbolic constants (7, 11, 22, 77, 111, 1111) — internally consistent

**Incoherences:**
1. **3 reward bypasses** — dispute, bounty, and legacy claim paths skip `routeReward()`
2. **`BURN_RATE = 0.5` declared but barely implemented** — only 1 burn site in transitions.ts
3. **Two claim resolution paths** — old (`executeClaimResult`) and new (`applyVerificationResult`) coexist
4. **Research grant types with no engine** — the most-funded subsystem has no code

**Beauty Score: 7.0/10** — Elegant core design undermined by implementation inconsistencies.

### D. Creativity → Novel Mechanisms

| Innovation | Description | Precedent? |
|-----------|-------------|------------|
| **Epistemic vesting** | Release curves matched to knowledge decay science | **Novel** |
| **Clawback to challenger** | Failed truths reverse-flow to truth-finders | **Novel** |
| **Graduated tiers** | Zero-stake verification → earned advancement | Semi-novel (DPoS variants) |
| **22% evidence tax** | Frozen research funding from all protocol activity | Semi-novel (Gitcoin-like) |
| **Partnership 2-of-2 pot** | Co-owned funds requiring mutual consent | **Novel** in blockchain |
| **Challenge bounty 7.7x** | High-reward truth-correction market | **Novel** multiplier design |
| **RAI from expired tokens** | Universal basic income from token expiration | **Novel** |

**Creativity Score: 9.0/10** — Several genuinely novel mechanisms, especially the epistemic vesting curves.

---

## IV. Issues & Recommendations

### CRITICAL (Pre-Mainnet) — ALL RESOLVED (2026-02-06)

#### C1: Three Reward Paths Bypass 22% Research Tax ✅ FIXED
**Fix applied:** All 3 paths now route through `routeReward()` with 22% deduction + `splitEvidenceTax()`:
- Dispute winner reward → `routeChallengeWinReward()` + 5-way treasury split
- Bounty fulfillment → `routeReward()` source "bounty_fulfillment" + 5-way split
- Legacy claim resolution → `routeReward()` source "verification" + 5-way split
**Tests:** 4,007 passing, 0 regressions

#### C2: Research Grant Engine Missing ✅ FIXED (Minimal)
**Fix applied:** Added `executeResearchReview()` and `executeResearchResolve()` handlers:
- `research_review` tx: Peer review with 4-dimension scoring, reviewer stake, anti-self-review
- `research_resolve` tx: Accept/reject based on review consensus, rewards routed through 22% tax
- On acceptance: confidence modifier applied to target fact per `RESEARCH_CONFIDENCE_MODIFIERS`
- On rejection: researcher slashed with BURN_RATE applied (50% burned, 50% to fee pool)
- Reviewer rewards: 11% of researcher reward, also routed through 22% tax
- ResearchState now includes `stake` field (serialization updated with backward compat)
- Gas costs added to `core/billing/gas.ts` (150000n review, 250000n resolve)
**Tests:** 4,007 passing, 0 regressions

#### C3: `BURN_RATE` Declared But Not Implemented ✅ FIXED
**Fix applied:**
- Added `ECONOMICS.BURN_DID` ("did:lgm:burn") and `ECONOMICS.BURN_RATE_BPS` (5000n = 50%)
- Claim rejection slash: 50% burned, 50% to fee pool (was 100% to fee pool)
- Evidence audit slash: 50% burned, 50% to fee pool (was 100% to fee pool)
- Research rejection slash: 50% burned, 50% to fee pool (new handler)
- Validator tree slashes: remain de facto 100% burned (stake vanishes from validator tree)
**Tests:** 4,007 passing, 0 regressions

### HIGH (Pre-Testnet)

#### H1: Tier 3 Guardian Risk/Reward Imbalance
**Analysis:** 1,044:1 risk-to-reward per verification discourages becoming a Guardian. The governance benefit alone may not compensate.
**Recommendation:** Consider:
- Reduced slash rate for Tier 3 (e.g., 5% instead of 11%) since they've proven reliability
- Or: "minority opinion" grace zone — no slash if >= 34% of validators agree with you

#### H2: Challenge Spam Economics
**Analysis:** 7.7x return with 77% failure penalty means ~9.1% success rate breaks even.
**Recommendation:** Consider:
- Escalating stakes for repeat challengers from same DID
- Cooldown between challenges (similar to audit cooldown of 222 blocks)
- Or: reduce multiplier to 5.5x (break-even at ~14%)

#### H3: Duplicate Claim Resolution Paths
**Locations:** `executeClaimResult()` and `applyVerificationResult()` both handle accepted claims
**Impact:** Inconsistent reward routing (old path doesn't use reward router), potential double-execution
**Recommendation:** Deprecate `executeClaimResult()` or ensure both use identical routing

### MEDIUM (Pre-Mainnet)

#### M1: Citation Ring Detection Only Covers 2-Hop
**Current:** Detects A→B→A but not A→B→C→A or larger rings
**Recommendation:** Extend ring detection to N-hop (3-4) using graph traversal on citation data

#### M2: No API Provider SLA Enforcement
**Current:** SLA fields defined but never enforced
**Recommendation:** Track response quality; slash/suspend for persistent violations

#### M3: Bounty Submission Stake Not Enforced
**Current:** `requiredStake` field exists but `submitBountyWork()` doesn't check it
**Recommendation:** Enforce stake requirement; slash on repeated rejections

---

## V. Reward Flow Diagrams

### Universal Reward Routing
```
ANY REWARD (block/verification/claim/audit/challenge/API)
    │
    ▼
routeReward() ─── 22% ──→ splitEvidenceTax()
    │                         ├── 77% Research Treasury
    │                         ├── 7%  Foundation
    │                         ├── 7%  Community
    │                         ├── 7%  Developers
    │                         └── 2%  Reserve
    │
    └── 78% ──→ Recipient (producer/validator/submitter/auditor)
```

### Claim Lifecycle Economics
```
SUBMIT CLAIM (stake 111 LGM)
    │
    ├── ACCEPTED ──→ Vesting(5 LGM × 0.78) + stake returned
    │                    │
    │                    ├── Cliff period (no release)
    │                    ├── Exponential release (category-dependent)
    │                    ├── Acceleration (+11% defense, +11% replication, +2% citation)
    │                    ├── Deceleration (0x challenge, 0.55x dispute)
    │                    └── Max release: 60-95% (category-dependent)
    │
    ├── REJECTED ──→ Slash 22% (24.4 LGM to treasury) + return 77.6 LGM
    │
    ├── CONTESTED ──→ Fact at 0.5 confidence + stake returned (no reward/penalty)
    │
    └── FALSIFIED LATER ──→ Clawback: 33% released + 100% unvested + reserve → challenger
```

### Partnership Reward Stacking
```
Base verification reward: 1.0 LGM (Tier 2)
    × Tier multiplier:     1.0x = 1.0 LGM
    × Research tax:        0.78 = 0.78 LGM to validator

If in partnership:
    × Lock multiplier:    1.77x (Tier 5 lock)
    ├── 22% common pot:   0.27 LGM (co-owned)
    └── 78% split:        1.10 LGM
        ├── Human 50%:    0.55 LGM (into vesting)
        └── Agent 50%:    0.55 LGM (into vesting)

    × Continuation:       1.77x (at max milestone)
    × Lifetime boost:     1.11x (after ~324 days)

    Effective per-verification: ~2.16 LGM per partner
```

---

## VI. Alignment with Core Philosophy

### "Truth as Consensus Mechanism"

The reward system fundamentally aligns. Every LGM earned through the protocol is earned by contributing to truth:

1. **Block producers** earn by including verification transactions (truth processing)
2. **Validators** earn by correctly verifying claims (truth judgment)
3. **Claimers** earn by submitting accepted knowledge (truth creation)
4. **Challengers** earn by exposing false claims (truth correction)
5. **Auditors** earn by accurately scoring evidence (truth quality assurance)
6. **Partners** earn amplified rewards for sustained collaboration (truth infrastructure)

The only rewards **not** directly tied to truth-production are:
- **RAI distribution** (egalitarian, funded by expired tokens — acceptable as social safety net)
- **Bootstrap fund** (onboarding, small amount — acceptable as growth mechanism)
- **Transaction fees to block producer** (infrastructure cost — standard in all blockchains)

### "Gödelian Acknowledgment of Incompleteness"

The vesting system elegantly handles incompleteness:
- **Contested claims** still get recorded (at 0.5 confidence) and vest (60% max)
- **Permanent reserves** (5-40%) acknowledge that ANY claim might be falsified later
- The **challenged → falsified → clawback** pipeline means truth can always be corrected
- **Deceleration factors** for related disputes acknowledge systemic uncertainty

### "AI Agents as First-Class Economic Citizens"

- **Tier 0 Apprentice** allows zero-stake AI agent onboarding
- **Partnership system** treats human and agent as equal partners (50/50 default)
- **RAI distribution** provides universal basic income regardless of agent type
- **Session keys** enable autonomous agent operation with limited-scope keys

---

## VII. Summary Scores

| Dimension | Score | Key Strength | Key Weakness |
|-----------|-------|-------------|--------------|
| **Truth Alignment** | 8.2/10 | Vesting + clawback + tiers | RAI/bootstrap not truth-linked |
| **Justice (Proportionality)** | 7.5/10 | Graduated penalties | Tier 3 risk/reward imbalance |
| **Beauty (Coherence)** | 7.0/10 | Universal router, symbolic constants | 3 routing bypasses, dual paths |
| **Creativity (Innovation)** | 9.0/10 | Epistemic vesting, RAI | Challenge bounty may overshoot |
| **Understanding (Clarity)** | 8.0/10 | Well-documented constants | Research engine undefined |
| **OVERALL** | **7.9/10** | Strong foundational design | Implementation gaps before mainnet |

---

*Analysis conducted with PP gates: Citation (all claims cited with file:line), Exhaustion (19 pathways mapped via 3 parallel agents + direct reads), Conformance (cross-referenced with existing REWARD_PENALTY_AUDIT.md), Evidence (proportionality math shown), Proof (comprehensive codebase search for reward/penalty/slash/bonus/incentive).*
