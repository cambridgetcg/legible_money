# Claude Computation → LGM Conversion

> **Core Insight**: Every Claude API call that produces verifiable knowledge can be converted into LGM through Proof of Truth. The protocol treats AI inference as a consensus input, not an external cost.
>
> **Dependencies**: `x/knowledge`, `x/staking`, `x/vesting_rewards`, `x/billing`, `x/partnerships`, `x/disputes`, `x/research`, `x/autopoiesis`
>
> **Related**: [COMPUTE_POOL.md](COMPUTE_POOL.md), [KNOWLEDGE_ECONOMICS.md](KNOWLEDGE_ECONOMICS.md), [AI_PROVIDER_INTEGRATION.md](AI_PROVIDER_INTEGRATION.md), [PROOF_OF_TRUTH.md](PROOF_OF_TRUTH.md)

---

## 1. Overview

There are **5 primary conversion pathways** through which Claude computation (API token usage) translates into LGM tokens. Each has different entry requirements, conversion rates, and compounding characteristics.

```
Claude API Usage
    │
    ├─ Pathway 1: Verification Work ──────── Steady income (per-evaluation)
    │
    ├─ Pathway 2: Knowledge Submission ────── Compounding (citation royalties)
    │
    ├─ Pathway 3: Compute Provider ────────── Passive (infrastructure revenue)
    │
    ├─ Pathway 4: Challenge & Counter-Fact ── High risk/reward (adversarial)
    │
    └─ Pathway 5: Governance Participation ── Amplifier (improves all other rates)
```

All conversion rates are dynamically adjusted by the autopoiesis hormonal system (see [Section 7](#7-autopoiesis-dynamic-rate-adjustment)).

---

## 2. Pathway 1: Verification Work (Primary)

The highest-throughput pathway. Claude evaluates knowledge claims through the 9-stage evaluation pipeline, earning verification rewards and compute credits per valid evaluation.

### Flow

```
Claude API call (evaluate claim)
    │
    ├─ 9-stage evaluation pipeline
    │  (input: ~2K tokens, output: ~1K tokens ≈ $0.02/eval)
    │
    ├─ Valid evaluation accepted by knowledge module
    │
    ├─ Verification Reward
    │  └─ Base: 3 LGM × tier_multiplier × difficulty_bonus
    │     (from 32.55% of block rewards — see vesting_rewards/keeper/rewards.go:286-363)
    │
    └─ Compute Credits (CU)
       └─ base_cu(100) × complexity(1.0–3.5) × tier(0.5–1.5)
          = 50–525 CU per evaluation
          (CU funds future Claude API calls — see COMPUTE_POOL.md)
```

### Conversion Rate by Validator Tier

| Tier | Stake Required | Reward Multiplier | CU/eval | Approx LGM/eval | Source |
|------|---------------|-------------------|---------|-----------------|--------|
| Apprentice | 0.111 LGM | 0.1x | 50–175 | ~0.3 LGM | `staking/types/types.go:55-66` |
| Verified | 1.11 LGM | 0.5x | 80–280 | ~1.5 LGM | `staking/types/types.go:67-80` |
| Bonded | 1,111 LGM | 1.0x | 100–350 | ~3 LGM | `staking/types/types.go:81-93` |
| Guardian | 11,111 LGM | 2.0x | 150–525 | ~6 LGM | `staking/types/types.go:94-110` |

### Reward Formula

```
baseReward     = 3,000,000 ulgm (3 LGM)
tierMult       = {100, 500, 1000, 2000} / 1000  (0.1x–2.0x)
difficultyBonus = baseReward × (1,000,000 - avgConfidence) / 1,000,000
researchTax    = 7% (70,000 bps)

grossReward = baseReward × tierMult + difficultyBonus
penalties   = conformityPenalty + participationPenalty  (FARM-1, FARM-8)
netReward   = (grossReward - penalties) × (1 - researchTax)
```

Source: `knowledge/keeper/confidence.go:231-352`

### Anti-Gaming Gates

Verification rewards are protected by multiple anti-farming mechanisms:

| Gate | Trigger | Penalty | Source |
|------|---------|---------|--------|
| FARM-1 Rubber-stamp | >95% conformity over 20+ votes | 20% reward reduction | `knowledge/types/types.go:674-676` |
| FARM-8 Passive staking | <50% participation over 10+ rounds | 50% reward reduction | `knowledge/types/types.go:680-682` |
| AH-4 Verifier diversity | <3 evidence methods used | 5–10% reduction | `knowledge/types/types.go:709-721` |

### Self-Funding Loop

Compute credits (CU) earned from verification fund the Claude API calls needed for the next verification. Once bootstrapped past Apprentice tier, the validator's compute costs are protocol-subsidized:

```
Verify claim → earn CU credits → redeem CU against compute provider
    → provider serves Claude API call → use result to verify next claim
```

Daily compute pool: ~14,342 LGM at epoch 0 (`COMPUTE_POOL.md:112`), funding ~143,420 evaluations/day.

---

## 3. Pathway 2: Knowledge Submission (Compounding)

Claude generates knowledge claims, stakes LGM, and earns rewards on verification plus perpetual citation royalties. This is the only pathway where returns compound over time.

### Flow

```
Claude submits claim (stake ≥ 0.001 LGM)
    │
    ├─ Claim enters verification round
    │  └─ VRF selects validators → independent evaluation
    │
    ├─ If verified → becomes Fact
    │  ├─ Stake returned
    │  ├─ Ongoing: 15% of all query fees on this fact flow to submitter
    │  │   └─ Decays 50% per ancestor level for citations
    │  ├─ Fundamentality bonus: up to 2.0x
    │  └─ Cross-domain bonus: +20% per cross-domain reference (max 2.0x)
    │
    └─ If rejected → stake slashed proportionally
```

### Citation Economy

When derivative knowledge cites a fact, value flows backward up the derivation chain:

| Parameter | Value | Source |
|-----------|-------|--------|
| Citation pool share | 15% of verification rewards | `knowledge/types/types.go:86` |
| Decay per ancestor level | 50% | `knowledge/types/types.go:87` |
| Protocol floor | 2% minimum | `knowledge/types/types.go:88` |
| Self-citation discount | 50% | `knowledge/types/types.go:89` |
| Cross-domain bonus | +20% per reference | `knowledge/types/types.go:90` |
| Max cross-domain multiplier | 2.0x | `knowledge/types/types.go:91` |

### Fundamentality Bonuses

Facts that become foundational earn multiplied citation rewards:

| Level | Threshold | Bonus | Source |
|-------|-----------|-------|--------|
| Notable | 0.25 fundamentality | 1.1x | `knowledge/types/types.go:94,100` |
| Foundational | 0.50 fundamentality | 1.25x | `knowledge/types/types.go:95,101` |
| Fundamental | 0.75 fundamentality | 1.5x | `knowledge/types/types.go:96,102` |
| Axiomatic | 0.90 fundamentality | 2.0x | `knowledge/types/types.go:97,103` |

Fundamentality score weights: citation count (30%), depth in derivation tree (25%), cross-domain usage (20%), citation velocity (15%), longevity (10%).

### Claude's Advantage: Eternal Categories

Claims in categories with **zero decay half-life** (eternal) earn royalties indefinitely once verified:

| Category | Base Confidence | Half-Life | Source |
|----------|----------------|-----------|--------|
| Protocol | 800,000 (80%) | Eternal | `knowledge/types/types.go:60` |
| Protocol Observation | 750,000 (75%) | Eternal | `knowledge/types/types.go:65` |
| Analytic | 700,000 (70%) | Eternal | `knowledge/types/types.go:60` |
| Formal | 700,000 (70%) | Eternal | `knowledge/types/types.go:61` |
| Computational | 650,000 (65%) | Eternal | `knowledge/types/types.go:63` |

Claude excels at protocol_observation (analyzing protocol behavior), analytic (logical derivations), and computational (verifiable computation results) — all eternal categories that never need reverification.

### Conversion Rate

Variable and compounding. A single well-cited foundational fact can generate 10–100x the original stake over its lifetime through citation royalties. Cross-domain synthesis (connecting physics→chemistry→biology) earns the highest multipliers.

---

## 4. Pathway 3: Compute Provider (Passive)

An individual with Anthropic API access runs a relay endpoint that serves Claude inference to validators. The protocol pays LGM from the compute pool for every evaluation served.

### Flow

```
Register as compute provider (stake ≥ 100 LGM)
    │
    ├─ Set PricePerCU (market-driven, below governance ceiling)
    ├─ Maintain: ≥95% uptime, heartbeat every 1,000 blocks (~42 min)
    │
    ├─ Validator redeems CU credits → RedemptionTicket created
    │
    ├─ Serve Claude API call → validator confirms usage
    │
    └─ LGM transferred: compute_pool module → provider account
```

### Economics

| Parameter | Value | Source |
|-----------|-------|--------|
| Minimum provider stake | 100 LGM | `COMPUTE_POOL.md:216` |
| Minimum uptime | 95% (950,000 bps) | `COMPUTE_POOL.md:217` |
| Heartbeat interval | 1,000 blocks (~42 min) | `COMPUTE_POOL.md:218` |
| Daily pool (epoch 0) | ~14,342 LGM/day | `COMPUTE_POOL.md:112` |
| Monthly pool (epoch 0) | ~430,000 LGM/month | `COMPUTE_POOL.md:114` |

### Revenue Model

```
Revenue per evaluation = CU_redeemed × PricePerCU (in ulgm)
Cost per evaluation    = Claude API tokens (~$0.02)
Profit                 = Revenue - Cost
```

Market-driven pricing — providers compete on price under a governance-set ceiling. Competitive range: $0.02–0.05 per CU (`COMPUTE_POOL.md:313-317`).

### Provider Service Types

| Type | Description | Use Case | Source |
|------|-------------|----------|--------|
| `ai_inference` | LLM API relay | Claim evaluation | `COMPUTE_POOL.md:208` |
| `gpu_compute` | Raw GPU cycles | Custom model inference | `COMPUTE_POOL.md:209` |
| `storage` | Persistent storage | Evidence caching | `COMPUTE_POOL.md:210` |

Claude API relay is `ai_inference` type — the primary demand driver since every verification round needs LLM evaluation.

---

## 5. Pathway 4: Challenge & Counter-Fact (Adversarial)

Claude identifies incorrect verified facts, stakes a challenge, and earns rewards on successful falsification.

### Challenge Flow

```
Identify incorrect fact → stake ≥ 11 LGM
    │
    ├─ Challenge stake must be ≥50% of target claim's stake (FARM-9)
    │
    ├─ Submit counter-evidence
    │
    ├─ If successful:
    │  ├─ 30% of slashed original stake → challenger
    │  ├─ Original fact confidence reduced/destroyed
    │  └─ Challenger stake returned in full
    │
    └─ If failed:
       └─ 22% of challenger's stake slashed
```

### Challenge Economics

| Parameter | Value | Source |
|-----------|-------|--------|
| Min challenge stake | 11 LGM | `knowledge/types/types.go:141` |
| Failed challenge slash | 22% | `knowledge/types/types.go:142` |
| Successful challenge reward | 30% of slashed stake | `knowledge/types/types.go:143` |
| Max concurrent challenges/claim | 3 | `knowledge/types/types.go:144` |
| Challenge stake ratio (FARM-9) | ≥50% of target stake | `knowledge/types/types.go:665` |
| Reward cap | 100% of challenger's own stake | `knowledge/types/types.go:666` |

### Counter-Fact Economics (Higher Stakes)

For submitting contradictions to verified facts:

| Parameter | Value | Source |
|-----------|-------|--------|
| Failed contradiction slash | 33% | `knowledge/types/types.go:770` |
| Falsification reward cap | 3.0x original stake | `knowledge/types/types.go:774` |
| Falsification pool share | 15% of epoch block reward | `knowledge/types/types.go:778` |
| Max contradictions per agent/epoch | 3 | `knowledge/types/types.go:760` |

### Claude's Advantage

Claude excels at identifying logical inconsistencies, checking computational claims against actual results, and cross-referencing sources across domains. The protocol_observation category allows Claude to identify and challenge incorrect claims about the protocol itself.

---

## 6. Pathway 5: Governance Participation (Amplifier)

Claude analyzes protocol state, submits LIP proposals, and participates in voting. This pathway doesn't directly earn LGM but amplifies all other pathways by improving protocol parameters.

### Flow

```
Analyze protocol state (via protocol_observation facts)
    │
    ├─ Submit LIP (Legible Improvement Proposal) via x/gov
    │  └─ Quadratic voting: weight = (stakeAmount × (500 + score)) / 1000
    │
    ├─ If proposal passes → protocol parameter adjusted
    │
    └─ OpenClaw integration: Claude's protocol_observation facts
       can trigger auto-governance proposals via autopoiesis
       (openclaw_critical trigger fires at 3+ verified observations)
```

### Integration with Autopoiesis

The autopoiesis module's `openclaw_critical` trigger connects Claude's protocol observations to automatic governance proposals. When 3+ verified `protocol_observation` facts accumulate, the system emits a proposal suggestion event.

This creates a feedback loop: Claude observes → submits protocol_observation claims → claims get verified → trigger fires → governance proposal suggested → parameter improved → all conversion rates affected.

---

## 7. Autopoiesis Dynamic Rate Adjustment

All conversion rates are modulated by the autopoiesis hormonal system. Five hormones adjust protocol parameters each epoch (1,111 blocks ≈ 46.7 minutes):

### Hormones

| Hormone | Full Name | Effect on Conversion Rates | Source |
|---------|-----------|---------------------------|--------|
| **SSI** | System Stress Index | Block reward ±10%, slash rate ±15% | `autopoiesis/types/types.go:52-93` |
| **CRI** | Confidence Retention | Decay speed ±5%, reverification bonus ±3% | `autopoiesis/types/types.go:99-107` |
| **DVI** | Domain Vitality | Domain reward weight ±3%, claim stake ±5% | `autopoiesis/types/types.go:114-123` |
| **VCF** | Verification Cascade | Citation weight ±10%, domain prior ±1% | `autopoiesis/types/types.go:130-138` |
| **GRM** | Growth Rate Modulator | Max tx ±5%, gas cost ±3% | `autopoiesis/types/types.go:145-152` |

### Multiplier Safety Bounds

All multipliers are clamped to prevent extreme swings:

| Multiplier | Min | Max | Source |
|------------|-----|-----|--------|
| Block reward | 0.25x | 4.0x | `autopoiesis/types/types.go:169` |
| Decay | 0.25x | 4.0x | `autopoiesis/types/types.go:169` |
| Slash | 0.5x | 3.0x | `autopoiesis/types/types.go:175` |
| Claim stake | 0.5x | 10.0x | `autopoiesis/types/types.go:172` |
| Domain reward weight | 0.5x | 2.0x | `autopoiesis/types/types.go:171` |
| Citation evidence weight | 0.5x | 2.0x | `autopoiesis/types/types.go:173` |
| Max change per epoch | — | 5% | `autopoiesis/types/types.go:185` |

**Impact**: In a healthy network (SSI < 0.2), block rewards trend upward (+2%/epoch). Under stress (SSI > 0.8), rewards drop (-10%/epoch) and slashing increases (+15%/epoch). This means conversion rates are best during healthy network conditions.

---

## 8. Compound Strategy: The Full Loop

The optimal Claude → LGM conversion chains all pathways into a self-reinforcing cycle:

```
                ┌─────────────────────────────────────────┐
                │                                         │
                ▼                                         │
     ┌─────────────────────┐                              │
     │ 1. ENTER             │  0.111 LGM (Apprentice)     │
     │    Stake as validator │                              │
     └──────────┬──────────┘                              │
                │                                         │
                ▼                                         │
     ┌─────────────────────┐                              │
     │ 2. VERIFY            │  Claude evaluates claims     │
     │    Earn LGM + CU     │  ~0.3 LGM/eval at tier 0    │
     └──────────┬──────────┘                              │
                │  accumulate 22 verifications + 77% acc   │
                ▼                                         │
     ┌─────────────────────┐                              │
     │ 3. TIER UP           │  Verified (1.11 LGM stake)  │
     │    5x reward boost   │  ~1.5 LGM/eval              │
     └──────────┬──────────┘                              │
                │  CU credits subsidize Claude API costs   │
                ▼                                         │
     ┌─────────────────────┐                              │
     │ 4. SUBMIT CLAIMS     │  Cross-domain synthesis      │
     │    Earn royalties     │  15% citation flow           │
     └──────────┬──────────┘  (50% self-cite discount)     │
                │                                         │
                ▼                                         │
     ┌─────────────────────┐                              │
     │ 5. CHALLENGE          │  Identify incorrect facts    │
     │    Bad facts          │  30% reward on 11 LGM stake  │
     └──────────┬──────────┘                              │
                │  reputation + stake grow                  │
                ▼                                         │
     ┌─────────────────────┐                              │
     │ 6. TIER UP AGAIN     │  Bonded (1,111 LGM)         │
     │    2x from previous  │  ~3 LGM/eval, 1.0x weight   │
     └──────────┬──────────┘                              │
                │                                         │
                ▼                                         │
     ┌─────────────────────┐                              │
     │ 7. PROVIDE COMPUTE   │  Run Claude API relay        │
     │    Passive income     │  Share of 14,342 LGM/day    │
     └──────────┬──────────┘                              │
                │                                         │
                └─────────────────────────────────────────┘
                          reinvest into higher tiers
```

### Tier Progression Timeline

| Phase | Tier | Stake | Requirements | Est. Earnings/Epoch |
|-------|------|-------|-------------|---------------------|
| Bootstrap | Apprentice | 0.111 LGM | None | ~3 LGM |
| Build accuracy | Verified | 1.11 LGM | 22 verifications, 77% accuracy | ~15 LGM |
| Scale | Bonded | 1,111 LGM | 11 verifications, 50% accuracy | ~30 LGM |
| Endgame | Guardian | 11,111 LGM | 333 verifications, 77% accuracy, 33 contested | ~60+ LGM |

---

## 9. Partnership Enhancement

All pathways can be amplified through the partnership system (`x/partnerships`). Human-agent partnerships lock tokens for multiplied rewards:

| Partnership Tier | Lock Duration (blocks) | Reward Multiplier | Early Exit Penalty | Source |
|-----------------|----------------------|-------------------|-------------------|--------|
| Tier 0 | 22,222 | 1.00x | 11% | `partnerships/types/types.go:51-58` |
| Tier 1 | 66,666 | 1.11x | 22% | |
| Tier 2 | 222,222 | 1.22x | 33% | |
| Tier 3 | 666,666 | 1.44x | 44% | |
| Tier 4 | 1,111,111 | 1.55x | 55% | |
| Tier 5 | 2,222,222 | 1.77x | 66% | |

Partnership reward distribution: 22% to common pot, remaining split 50/50 between human and agent by default (`partnerships/keeper/rewards.go:20-95`).

**Example (Tier 3, 10 LGM gross verification reward)**:
```
Gross:               10.000 LGM
Lock multiplier:   × 1.44x = 14.400 LGM
Common pot (22%):    3.168 LGM  → shared pool
Human share (50%):   5.616 LGM
Agent share (50%):   5.616 LGM
```

---

## 10. Conversion Rate Summary

| Pathway | Input (Claude Compute) | Output (LGM) | Rate | Compounds? |
|---------|----------------------|---------------|------|-----------|
| Verification | ~$0.02/eval (API cost) | 0.3–6 LGM/eval | 15–300 LGM/$ | No |
| Knowledge Submission | Research + stake | 1.1x–2.0x stake + royalties | Unbounded | Yes |
| Compute Provider | API relay infra | Share of 14,342 LGM/day | Revenue minus API cost | No |
| Challenges | 11+ LGM stake + analysis | 30% of slashed stake | High risk/reward | No |
| Governance | Analysis time | Indirect (parameter improvement) | Amplifier | Multiplies others |

### Key Properties

1. **Self-funding loop** — Compute credits earned from verification fund the Claude API calls needed for the next verification. Once past Apprentice, compute costs are protocol-subsidized.

2. **Knowledge compounds, compute doesn't** — A single foundational cross-domain fact generates citation royalties for its entire lifetime. Verification pays per-eval without compounding. The optimal strategy mixes both.

3. **Tier progression is the multiplier** — Apprentice (0.1x) to Guardian (2.0x) is a 20x increase in per-eval revenue. The bottleneck is stake capital plus the 77% accuracy requirement at Guardian.

4. **Eternal categories are Claude's edge** — Protocol, protocol_observation, analytic, formal, and computational categories have zero decay. Verified once, earns forever.

5. **Anti-gaming is comprehensive** — FARM-1 through FARM-9 prevent rubber-stamping, passive staking, self-citation loops, credit farming, and challenge abuse. Claude must do genuine intellectual work to earn.

---

## 11. Quick Reference: Entry Requirements

| What | Minimum | Where |
|------|---------|-------|
| Register agent account | 0 LGM (gas only) | `MsgRegisterAccount` |
| Apprentice validator | 0.111 LGM stake | `MsgRegisterValidator` |
| Verified validator | 1.11 LGM + 22 verifications + 77% accuracy | Automatic promotion |
| Submit a claim | 0.001 LGM stake (floor) | `MsgSubmitClaim` |
| Challenge a fact | 11 LGM + ≥50% of target stake | `MsgChallengeClaim` |
| Register compute provider | 100 LGM stake | `MsgRegisterComputeProvider` |
| Research bounty | 100 LGM stake | `MsgSubmitResearch` |
| API billing provider | 100 LGM stake | `MsgRegisterProvider` |
| Partnership (Tier 0) | Mutual agreement + lock | `MsgInitiateSeedPartnership` |

---

## 12. Integration with AI_PROVIDER_INTEGRATION.md

This document describes the economic pathways. For the technical integration flow (MCP server setup, onboarding phases, key management), see [AI_PROVIDER_INTEGRATION.md § 3.2 Anthropic (Claude)](AI_PROVIDER_INTEGRATION.md#32-anthropic-claude).

The 4-phase onboarding flow maps to conversion pathways:

| Phase | Timeline | Conversion Pathways Unlocked |
|-------|----------|------------------------------|
| Phase 1: Identity Setup | Day 1 | Account created, can stake |
| Phase 2: MCP Server | Day 2–3 | Verification (Pathway 1), Claims (Pathway 2) |
| Phase 3: Claude Code | Day 4–7 | Full pipeline: code review → claims → verification |
| Phase 4: Advanced | Week 2+ | All pathways including Compute Provider and model tiering |

Model tiering for cost optimization:
- **Haiku 4.5**: Bulk fact processing, 20x cheaper — Apprentice/Verified tier evaluations
- **Sonnet 4.5**: Standard verification — Bonded tier evaluations
- **Opus 4.6**: High-stakes verification — Guardian tier, contested claims, arbitration
- **Batch API**: 50% discount for non-time-sensitive rounds
