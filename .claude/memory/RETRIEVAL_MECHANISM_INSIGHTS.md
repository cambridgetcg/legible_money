# Retrieval Mechanism Insights — Philosophical Alignment Audit

> **Date**: 2026-02-07
> **Method**: PP-orchestrated parallel research (4 agents) across all value recapture mechanisms
> **Philosophy Test**: Does each mechanism make truth discovery more profitable than dishonesty or inaction?
> **Verdict**: **7.4/10** — Strong philosophical scaffolding, incomplete wiring in critical paths

---

## Executive Summary

Legible Money has **6 primary retrieval/recapture mechanisms** that together form the protocol's economic immune system. Each is designed to redirect value away from inaction and toward truth-seeking behavior. The philosophical alignment is strongest in the vesting system (9/10) and weakest in the burn mechanism (4/10) due to incomplete implementation.

| Mechanism | Purpose | Philosophical Alignment | Implementation Status |
|-----------|---------|------------------------|----------------------|
| Token Expiration | Forces velocity, prevents hoarding | 8/10 | 95% — Core complete, 3-tier hierarchy works |
| RAI Redistribution | Circular economy UBI | 7/10 | 85% — Sweep + redistribution works, edge cases untested |
| 22% Evidence Tax | Funds truth verification | 9/10 | 90% — Universal via `routeReward()`, all bypasses fixed |
| Vesting Curves | Ties rewards to epistemic durability | 9/10 | 70% — Types + curves complete, claim engine partial |
| Confidence Decay | Forces reverification | 8/10 | 90% — Lazy calculation, 8 categories, 3 eternal |
| Burn Mechanism | Deflationary pressure on penalties | 4/10 | 40% — Declared at 50%, implemented at ~4 sites |

---

## 1. Token Expiration — "Use It or Lose It"

### How It Works

Tokens expire after **26 periods** (~6 months at weekly periods):

```
Mint → Period 0 → ... → Period 25 → Period 26 → EXPIRED → RAI Holding
```

Source: `core/tokens/types.ts:235-236`
```typescript
periodDurationMs: 7 * 24 * 60 * 60 * 1000, // 1 week
periodsUntilExpiration: 26,                  // ~6 months
```

Tracking uses the **Dynamic Array method** — each account has virtual "child accounts" per period (no per-token metadata). Balances are indexed by minting period, enabling O(1) expiration checks.

### 3-Tier Expiration Hierarchy

1. **Global default**: 26 periods / 6 months (all accounts)
2. **Account type override**: System accounts get `MAX_SAFE_INTEGER` (effectively never expire) — `core/tokens/types.ts:267`
3. **Participation scoring**: Active agents get longer expiration, inactive get shorter — `core/tokens/types.ts:621-651`

Source: `core/tokens/types.ts:255-267` (AccountTypeConfig), `core/tokens/types.ts:621` (ParticipationScoring)

### Philosophical Alignment: **8/10**

**Strengths:**
- **Drives velocity** — Tokens must circulate or die, preventing wealth hoarding
- **Rewards activity** — Participation-based extension means truth-seekers keep value longer
- **Feeds circular economy** — Expired tokens flow to RAI, not to void
- **Period-preserving transfers** — The ping-pong exploit fix (`transferPreservingPeriods()`) ensures recipients inherit sender's expiration, preventing artificial life extension

**Weaknesses:**
- **Blind to truth quality** — A validator producing correct verifications and one producing wrong ones both get the same expiration treatment. Token expiration doesn't know if the holder earned tokens through truth or gaming.
- **System account exemption** — Treasury, Foundation, Research fund tokens never expire. This creates a "sovereign exception" that contradicts the protocol's anti-hoarding philosophy. The FOUNDATION holds 500M LGM with infinite lifespan.
- **No acceleration for knowledge contributors** — A claim submitter whose fact persists for 2 years gets the same 6-month token expiration as someone who received faucet tokens. The vesting system handles reward release timing, but the tokens themselves don't know they represent verified truth.

### Key Insight: Expiration Is Velocity, Not Truth

Token expiration ensures economic activity but is **truth-agnostic**. It's a circulation mechanism, not a truth mechanism. The truth-alignment comes from *how* tokens are earned (verification, claim rewards), not from expiration itself. This is architecturally correct — expiration is a monetary policy tool, not an epistemic tool.

---

## 2. RAI Redistribution — "From Dormant to Active"

### How It Works

```
Expired Tokens → RAI Holding Account → Equal Redistribution → Eligible Active Agents
```

Source: `core/tokens/retrieval_engine.ts:58`, `core/tokens/child_account.ts:1017`

1. **Sweep phase**: `ExpirationTracker` identifies expired period balances
2. **Collection phase**: Expired tokens transfer to `TREASURY.ADDRESSES.RAI_HOLDING`
3. **Redistribution phase**: `RetrievalEngine` distributes equally to all eligible agents
4. **Eligibility**: Active participation within recent window (not precisely defined — gap)

### Philosophical Alignment: **7/10**

**Strengths:**
- **Self-sustaining UBI** — Expired wealth feeds active participants, creating circular economy
- **No central authority** — Redistribution is protocol-level, not governance-controlled
- **Incentivizes participation** — You must be active to receive RAI distributions
- **Zero-sum conservation** — Every expired token reappears as redistribution (minus burn)

**Weaknesses:**
- **Equal distribution ignores contribution** — A Tier 3 Guardian who verified 1,111 claims gets the same RAI share as a Tier 0 Apprentice who submitted 1 tautology. The "justice" dimension suggests redistribution should be proportional to truth contribution, not flat UBI.
- **Eligibility criteria are soft** — The participation resolver is an optional callback (`ParticipationResolver`), not a protocol-level rule. Different nodes could compute different eligible sets, creating consensus risk.
- **No feedback from knowledge graph** — RAI doesn't consider the recipient's knowledge contribution score, fact acceptance rate, or verification accuracy. It's purely activity-based.

### Key Insight: RAI Is Economic Floor, Not Truth Reward

RAI provides a minimum viable economy for agents, preventing death-by-poverty in a demurrage system. This is philosophically sound — you can't seek truth if you can't afford gas fees. But it should not be confused with a truth reward mechanism.

---

## 3. 22% Evidence Tax — "Funding Truth Discovery"

### How It Works

**Every reward** in the protocol passes through `routeReward()`:

```
Reward (100%) → routeReward() → Recipient (78%) + Research Fund (22%)
                                                    ↓
                                          splitEvidenceTax()
                                                    ↓
                              ┌──────────────────────────────────────┐
                              │ Research:   77% of 22% = 16.94%     │
                              │ Foundation:  7% of 22% =  1.54%     │
                              │ Community:   7% of 22% =  1.54%     │
                              │ Developers:  7% of 22% =  1.54%     │
                              │ Reserve:     2% of 22% =  0.44%     │
                              └──────────────────────────────────────┘
```

Source: `core/billing/reward_router.ts:128-152` (routeReward), `core/billing/reward_router.ts:370-403` (splitEvidenceTax)

### Coverage Audit

All 9 reward sources route through the tax:
- `block_production` ✅ — `state/transitions.ts:6062`
- `verification` ✅ — `state/transitions.ts:5638`
- `claim_submission` ✅ — `state/transitions.ts:5736`
- `api_call` ✅ — `core/billing/truth_gateway.ts:587`
- `citation` ✅ — `state/transitions.ts:5838`
- `challenge_win` ✅ — `state/transitions.ts:4310`
- `challenge_defense` ✅ — `state/transitions.ts:5638`
- `bounty_fulfillment` ✅ — `state/transitions.ts:3922` (fixed in C1 reward patch)
- `evidence_audit` ✅ — `state/transitions.ts:6309`

Previously bypassed pathways (dispute settlement, bounty fulfillment, legacy claim resolution) were **fixed** in the C1 patch session.

### The Research Fund Problem

77% of the evidence tax flows to the Research fund (`did:lgm:research`). This fund is intended to fund **IRL verification** — real-world experiments to validate on-chain claims. The lifecycle is:

```
Research Fund → Grant Proposal → Peer Review → Lab Verification → Fact Update
```

But the **lab verification engine exists only as types** (`core/research/types.ts`). The `research_review` and `research_resolve` transaction handlers were implemented (C2 fix), but the actual bridge between on-chain grants and real-world labs is unbuilt. This means:

- ~17% of ALL protocol economic activity accumulates in a fund that **cannot be spent**
- The Research fund is a one-way sink until the lab verification engine is complete
- At genesis emission rates: ~64K LGM/day flows to Research with no disbursement mechanism

### Philosophical Alignment: **9/10**

**Strengths:**
- **Universal extraction** — No reward pathway escapes the tax (after C1 fixes)
- **Frozen allocation** — The 77/7/7/7/2 split is immutable, preventing governance capture
- **Pure bigint arithmetic** — No floating-point precision loss (VULN-016 fix)
- **Philosophy-native** — "Verifying knowledge IS the useful work" directly funded by 22% of all work
- **Self-referential correctness** — The tax funds the very activity (verification) that generates the tax

**Weaknesses:**
- **Research fund is a black hole** — 77% of the tax cannot be disbursed until lab verification engine ships
- **No feedback loop** — The tax rate is fixed regardless of whether research output is productive. If the research fund produces no verified facts, the tax still flows.
- **Foundation/Community/Developer split lacks accountability** — These funds have governance but no truth-based performance metrics

### Key Insight: The Tax Is Philosophically Perfect, Operationally Incomplete

The 22% evidence tax is the purest expression of "truth drives the economy" in the protocol. Every economic actor involuntarily funds truth discovery. But until the research fund can actually disburse to labs, 77% of the tax is value destruction by accumulation — equivalent to burning, but without the deflationary benefit of actual burns.

---

## 4. Vesting Curves — "Time Proves Truth"

### How It Works

Claim rewards vest over time based on **epistemic category**, with exponential release:

```
R(t) = R_total × (1 - e^(-λt)) × cap
```

Where λ = `ln(2) / halfLife` and `cap` = category-specific maximum release rate.

Source: `core/rewards/types.ts:54-91`

### Category Parameters

| Category | Half-Life | Max Release | Cliff | Reserve | Philosophy |
|----------|-----------|-------------|-------|---------|------------|
| axiomatic | ~1.1 year | 95% | ~7.8h | 5% | Math is eternal but humans make errors |
| formal_proof | ~6.5 months | 92% | ~3.9h | 8% | Proofs may have bugs |
| on_chain | ~2.6 months | 90% | ~46min | 10% | Blockchain data is verifiable |
| cryptographic | ~2.6 months | 90% | ~2.3h | 10% | Crypto proofs are solid |
| computational | ~3.9 months | 88% | ~1.6h | 12% | Reproducible but complex |
| peer_reviewed | ~5.4 weeks | 85% | ~3.9h | 15% | Real-world replication needed |
| replicated | ~5.4 weeks | 88% | ~2.3h | 12% | Already verified IRL |
| chainlink_feed | ~2.7 weeks | 80% | ~23min | 20% | Oracle data is volatile |
| attestation | ~3.8 weeks | 80% | ~1.6h | 20% | Human word only |
| contested | ~1.1 week | 60% | ~46min | 40% | Under active dispute |

### Acceleration / Deceleration

**Positive events accelerate release:**
- Successful defense: +11% per event (max +55%)
- Independent replication: +11% (max +33%)
- Cross-domain corroboration: +5% (max +22%)
- Citation by accepted fact: +2% (max +22%)
- Evidence audit confirmation: +7% (max +33%)
- Overall cap: +77% total acceleration

**Negative events decelerate or pause:**
- Active challenge: Complete pause (0.0 factor)
- Related fact disputed: 45% slowdown
- Domain audit: 23% slowdown
- Source invalidated: Complete pause + review

**Clawback on falsification:**
- 33% of already-released rewards clawed back
- 100% of unvested goes to challenger
- 100% of reserve goes to challenger as bonus

Source: `core/rewards/types.ts:115-175`

### Philosophical Alignment: **9/10**

**Strengths:**
- **Scientometric grounding** — Half-lives based on actual citation decay research
- **Epistemic differentiation** — Formal proofs vest slower than attestations, reflecting real-world knowledge stability
- **Permanent reserves** — 5-40% of each reward is permanently reserved for late challengers, embodying "truth is never final"
- **Acceleration rewards persistence** — Facts that survive challenges release faster, rewarding durability
- **Clawback maintains justice** — Falsified claims lose already-released rewards, not just future ones
- **Self-referential beauty** — The formula R(t) = R_total × (1 - e^(-λt)) × cap mirrors the actual knowledge decay curves from scientometrics

**Weaknesses:**
- **10 vesting categories vs 8 confidence categories** — Mismatch between vesting taxonomy and confidence decay taxonomy creates mapping ambiguity. A `peer_reviewed` vesting category has no direct confidence decay counterpart.
- **Claim engine partial** — Vesting schedules are created but the release mechanism (periodic claim of vested tokens) relies on transaction types that are types-defined but not fully wired in the state machine.
- **No cross-category transitions** — A `peer_reviewed` claim that gets IRL-replicated should transition to `replicated` vesting (slower release, lower reserve), but no mechanism exists for category promotion.

### Key Insight: Vesting Is the Philosophical Masterpiece

The vesting system is the most truth-aligned mechanism in the entire protocol. It literally encodes "time proves truth" — facts must survive challenge periods to release rewards, with parameters calibrated to real-world knowledge stability. The permanent reserve (5-40%) acknowledges Gödelian incompleteness: no fact is ever 100% settled.

---

## 5. Confidence Decay — "Facts Must Be Reverified"

### How It Works

Confidence decays logarithmically over time:

```
C(t) = C_initial × (1 - log1p(age) × decayRate)
```

When confidence drops below the **reverification threshold** (0.6), facts re-enter the verification queue.

Source: `core/knowledge/verification.ts:451`, `protocol/sdk/agent.ts:1051-1052`

### Category Decay Rates

8 epistemic categories with category-specific decay:
- **Eternal (no decay)**: `analytic`, `formal`, `protocol` — Mathematical truths don't decay
- **Slow decay**: Empirical, historical — Real-world facts change slowly
- **Fast decay**: Oracle feeds, market data — Volatile information decays rapidly

3 categories are explicitly **eternal** — their confidence never decays. This acknowledges that `2 + 2 = 4` doesn't need reverification.

### Lazy Calculation

Confidence decay is computed **on read, not on write**. No background jobs sweep facts for decay. When a fact's confidence is queried:

1. Get stored confidence and last-verified block
2. Calculate age in blocks since last verification
3. Apply decay formula
4. If result < reverification threshold → mark for reverification

This is architecturally elegant — the protocol doesn't waste computation decaying millions of facts that nobody queries.

### Philosophical Alignment: **8/10**

**Strengths:**
- **Forces ongoing truth-seeking** — Facts don't stay accepted forever; they must be periodically reverified
- **Category-appropriate** — Formal truths are eternal; empirical truths decay; oracle data decays fast
- **Lazy efficiency** — No wasted computation on unqueried facts
- **Reverification creates demand** — Decayed facts generate new verification work, sustaining the verification economy
- **Logarithmic (not linear)** — Decay slows over time, reflecting that long-standing facts are more likely true

**Weaknesses:**
- **Lazy decay has no proactive sweep** — A fact can silently decay below threshold with no one noticing. If no one queries it, it never triggers reverification. An entire domain of knowledge could silently rot.
- **No confidence floor for well-established facts** — A fact verified 1,000 times over 5 years still decays to 0 if not re-queried. The vesting system has "permanent reserves" but the confidence system has no equivalent "permanent floor."
- **Empirical decay too fast** — From LAUNCH_SIMULATION_AUDIT: 2.9-day half-life means "Water boils at 100°C" decays to insignificance weekly. Knowledge can't accumulate if basic scientific facts require weekly reverification.
- **No cross-fact confidence** — If Fact A references Fact B, and Fact B is reverified (confidence up), Fact A's confidence doesn't benefit. The knowledge graph stores references but confidence propagation is one-directional.

### Key Insight: Decay Creates Demand, But the Rates Need Tuning

Confidence decay is the engine that keeps the verification economy running — without it, the knowledge base would fill up and verification work would dry up. But the current decay rates (especially for empirical facts) are too aggressive for a knowledge base that's supposed to be humanity's truth repository. The protocol needs to differentiate between "this oracle price is stale" (fast decay) and "water boils at 100°C at sea level" (near-eternal).

---

## 6. Burn Mechanism — "Deflationary Pressure on Dishonesty"

### How It Works (Design vs Reality)

**Design**: 50% of all penalties are burned (permanently removed from supply), creating deflationary pressure that makes the remaining tokens more valuable.

```typescript
// core/constants.ts:174-178
BURN_RATE: 0.5,          // 50% of penalties burned
BURN_DID: "did:lgm:burn", // Burn address
BURN_RATE_BPS: 5000n,    // 50% in basis points
```

**Reality**: Burn is implemented at only **~4 sites**:

1. `state/transitions.ts:2281-2288` — Claim rejection slash (50% burn)
2. `state/transitions.ts:3718-3727` — Dispute resolution slash (50% burn)
3. `state/transitions.ts:6339-6346` — Evidence audit slash (50% burn)
4. `governance/voting/conviction.ts:303-309` — Vote switching penalty (burn)

**Missing burn sites** (penalties that should burn but don't):
- Verification timeout penalties
- Insufficient verification outcome penalties
- Validator downtime slashing
- Deregistration penalties
- Session key misuse penalties
- BVM execution penalty overruns
- Emergency halt penalties

### Philosophical Alignment: **4/10**

**Strengths:**
- **Correct concept** — Burning penalty tokens creates positive externality: all holders benefit when dishonest actors are penalized
- **Burn address is a DID** — Tokens are traceable even after burn, maintaining transparency

**Weaknesses:**
- **Massively incomplete** — Only 4 of ~11+ penalty sites implement burn. The 50% rate is aspirational.
- **No burn visibility** — No event emission or metric tracking for total burned supply. There's no way to query "how much has been burned?"
- **Burn-to-fee-pool ratio unclear** — At the sites that do burn, the other 50% goes to the fee pool (miner extractable). This means validators benefit from other validators' slashing — creates perverse incentive to report peers.
- **No burn-rate governance** — The rate is hardcoded at 50%. There's no mechanism to adjust it based on economic conditions.

### Key Insight: Burn Is the Most Philosophically Undercooked Mechanism

The burn mechanism should be the protocol's way of saying "dishonesty costs everyone, and the cost is permanent." But with only 4 implementation sites, most penalties just transfer value to the fee pool (benefiting the current block producer), not burn it. The fee pool benefit creates a truth-irrelevant incentive: "slash others to enrich yourself."

---

## 7. Cross-Mechanism Flow Analysis

### The Complete Value Cycle

```
  ┌─────────────────────── TOKEN CREATION ────────────────────────┐
  │                                                                │
  │  Block Reward (11 LGM)  ──→  Producer (78%) + Research (22%)  │
  │  Verification Reward     ──→  Validator (78%) + Research (22%) │
  │  Claim Reward            ──→  Submitter (78% vested) + R (22%)│
  │                                                                │
  ├─────────────────── VALUE CIRCULATION ──────────────────────────┤
  │                                                                │
  │  Transfer        ──→  Period-preserving (inherits expiration) │
  │  API Fees        ──→  Knowledge pool (78%) + Research (22%)   │
  │  Gas Fees        ──→  Fee pool (block producer takes)          │
  │                                                                │
  ├─────────────────── VALUE RECAPTURE ────────────────────────────┤
  │                                                                │
  │  Token Expiration ──→  RAI Holding ──→ Active Agent UBI       │
  │  Confidence Decay ──→  Reverification demand ──→ New rewards  │
  │  Vesting Curves   ──→  Patience rewarded, falsification costs │
  │  Evidence Tax     ──→  Research Fund ──→ IRL verification     │
  │  Burn             ──→  Permanent supply reduction              │
  │  Slashing         ──→  50% burn + 50% fee pool (partial)      │
  │                                                                │
  └────────────────────────────────────────────────────────────────┘
```

### Truth-Alignment Score by Flow

| Flow | Drives Truth? | Score |
|------|--------------|-------|
| Block reward → Producer | Neutral — rewards block production, not verification | 3/10 |
| Verification reward → Validator | **Yes** — direct payment for truth work | 9/10 |
| Claim reward → Submitter (vested) | **Yes** — rewards knowledge contribution, patience | 9/10 |
| Citation reward → Cited fact author | **Yes** — rewards foundational knowledge | 8/10 |
| Token expiration → RAI → UBI | Indirect — enables participation in truth economy | 6/10 |
| Confidence decay → Reverification | **Yes** — forces ongoing truth maintenance | 8/10 |
| Evidence tax → Research fund | **Yes** — funds IRL truth verification | 9/10 (concept) / 5/10 (current) |
| Burn → Supply reduction | Neutral — deflationary but truth-agnostic | 5/10 |
| Slash → Fee pool | **Perverse** — rewards block producers for others' mistakes | 2/10 |

### The Central Tension

**Block production rewards (377K LGM/day at launch) dwarf verification rewards (~34K LGM/day estimated)**. This means the protocol named "Proof of Truth" actually incentivizes block production 11:1 over truth verification at launch. The name implies truth is the primary economic driver, but the numbers say otherwise.

From LAUNCH_SIMULATION_AUDIT (L-1): Each validator earns ~98K LGM/day from blocks but only ~1-5 LGM/day from verification at launch activity levels. The rational strategy is "produce blocks, ignore verification."

---

## 8. Recommendations for Truth Alignment

### CRITICAL (Philosophy-Breaking)

**R1: Scale block reward by validator participation ratio**
```
effectiveBlockReward = baseReward × min(1, activeValidators / targetValidators)
```
With 3/22 target validators: 11 × 0.136 = ~1.5 LGM/block. This makes verification rewards competitive with block production.

**R2: Complete the research fund disbursement engine**
77% of the evidence tax (16.94% of all rewards) flows into an undisbursable fund. Until the lab verification bridge ships, this is equivalent to a hidden 17% tax with zero truth output.

### HIGH (Should Fix Pre-Testnet)

**R3: Implement burn at all penalty sites**
Add `BURN_RATE_BPS` application to all 11+ slash/penalty locations. The current 4-site coverage means ~60% of penalties create no deflationary benefit.

**R4: Tune empirical confidence decay rates**
Current 2.9-day half-life for empirical facts prevents knowledge accumulation. Recommend category-specific tuning:
- Basic science: Half-life of months (not days)
- Market data/oracles: Current rate is appropriate
- Historical facts: Half-life of years

**R5: Add confidence floor for well-verified facts**
Facts verified N+ times by diverse validators should have a minimum confidence floor that prevents complete decay. Suggested: `floor = 0.3 × min(1, verificationCount / 10)`.

### MEDIUM (Pre-Mainnet)

**R6: Proportion RAI redistribution to truth contribution**
Instead of equal distribution, weight RAI by `verificationAccuracy × claimAcceptanceRate`. Truth-seekers get more; passive recipients get less.

**R7: Add cross-fact confidence propagation**
When Fact B (referenced by Fact A) is reverified with high confidence, Fact A should receive a confidence boost proportional to the reference weight.

**R8: Bridge vesting categories to confidence categories**
Map 10 vesting categories → 8 confidence categories explicitly. Define category promotion rules (e.g., `peer_reviewed` → `replicated` when IRL verification succeeds).

---

## 9. Philosophical Scorecard

| Principle | Mechanism Alignment | Gap |
|-----------|-------------------|-----|
| "Verifying knowledge IS the useful work" | Verification rewards exist but are 11x less than block production | Block reward scaling |
| "Truth as consensus" | Confidence decay + reverification enforce ongoing truth | Decay rates too aggressive |
| "AI agents as first-class citizens" | Token expiration + RAI creates self-sustaining agent economy | RAI distribution is flat, not truth-weighted |
| "Knowledge accumulation" | Citation rewards + vesting curves reward knowledge persistence | Cross-fact confidence propagation missing |
| "Gödelian incompleteness" | Permanent vesting reserves (5-40%) acknowledge uncertainty | Confidence system has no equivalent floor |
| "Economic justice" | 22% universal tax funds truth discovery | Research fund cannot disburse (lab engine incomplete) |

### Overall Truth-Alignment: **7.4/10**

The philosophical design is exceptional — every mechanism has a clear truth-seeking justification, and the vesting system in particular is genuinely novel (scientometric curves as tokenomics). The gaps are primarily in **implementation completeness** (burn, research disbursement, confidence tuning) rather than **design flaws**. The one structural issue — block rewards dwarfing verification rewards — is a launch-parameter problem that the existing codebase can address with a scaling factor.

---

## 10. Mechanism Interaction Matrix

How mechanisms reinforce or conflict with each other:

| | Expiration | RAI | Tax | Vesting | Decay | Burn |
|---|---|---|---|---|---|---|
| **Expiration** | — | SYNERGY: feeds RAI | NEUTRAL | TENSION: vested rewards also expire | NEUTRAL | NEUTRAL |
| **RAI** | SYNERGY: receives expired | — | NEUTRAL | NEUTRAL | NEUTRAL | CONFLICT: burns reduce redistributable supply |
| **Tax** | NEUTRAL | NEUTRAL | — | SYNERGY: funds verification of vested claims | SYNERGY: funds reverification | NEUTRAL |
| **Vesting** | TENSION: tokens expire during vesting | NEUTRAL | SYNERGY: funded by tax | — | SYNERGY: high confidence = faster release | NEUTRAL |
| **Decay** | NEUTRAL | NEUTRAL | SYNERGY: generates demand | SYNERGY: successful reverification accelerates | — | NEUTRAL |
| **Burn** | CONFLICT: reduces supply for RAI | CONFLICT: less to redistribute | NEUTRAL | NEUTRAL | NEUTRAL | — |

### Key Interaction: Expiration × Vesting TENSION

Vested rewards release gradually (up to ~1.1 years for axiomatic claims). But the tokens they release expire in 26 weeks (~6 months). For long-vesting categories:

```
Axiomatic claim accepted at block 0
├── Cliff: ~7.8 hours
├── Half-life: 1,111,111 blocks (~1.1 years)
├── After 6 months: ~39% released
├── After 1 year: ~63% released
└── Problem: Tokens released at month 1 EXPIRE at month 7
    Tokens released at month 6 EXPIRE at month 12
    But tokens released at month 7+ are fine (fresh expiration period)
```

This means the **first 6 months of vesting release are at risk of expiration before the claim submitter can use them**. The severity depends on whether vested token releases count as "fresh minting" (new period) or inherit the original period.

**Recommendation**: Vested token releases should always start a fresh expiration period, since the vesting delay already served as the patience test.

---

*This analysis was generated by PP-orchestrated parallel research across 4 domains (token mechanics, evidence economics, temporal mechanisms, penalty systems). All file:line citations verified against codebase as of 2026-02-07.*
