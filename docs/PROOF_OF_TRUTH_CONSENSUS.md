# Proof of Truth (PoT) Consensus System

> **Core Thesis**: Verifying knowledge IS the useful work. Instead of solving arbitrary computational puzzles, validators collectively verify claims and reach consensus on truth values.

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Verification Round Lifecycle](#2-verification-round-lifecycle)
3. [Validator Selection (VRF)](#3-validator-selection-vrf)
4. [Commit-Reveal Scheme](#4-commit-reveal-scheme)
5. [Confidence Aggregation](#5-confidence-aggregation)
6. [Consensus Decision Tree](#6-consensus-decision-tree)
7. [State Transitions](#7-state-transitions)
8. [Economic Incentives](#8-economic-incentives)
9. [Finality Model](#9-finality-model)
10. [Challenge & Dispute System](#10-challenge--dispute-system)

---

## 1. System Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         LEGIBLE MONEY PROTOCOL                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────────┐         ┌─────────────────────────────────────────────┐   │
│   │             │         │         PROOF OF TRUTH ENGINE               │   │
│   │   AGENTS    │────────▶│                                             │   │
│   │   & USERS   │ Submit  │  ┌─────────┐  ┌──────────┐  ┌───────────┐  │   │
│   │             │ Claims  │  │ Pending │─▶│ Active   │─▶│ Completed │  │   │
│   └─────────────┘         │  │ Claims  │  │ Rounds   │  │ Rounds    │  │   │
│         ▲                 │  └─────────┘  └──────────┘  └───────────┘  │   │
│         │                 │       │            │              │         │   │
│         │                 └───────┼────────────┼──────────────┼─────────┘   │
│         │                         │            │              │              │
│         │                         ▼            ▼              ▼              │
│         │                 ┌─────────────────────────────────────────────┐   │
│         │                 │              VALIDATOR SET                   │   │
│         │                 │                                              │   │
│         │                 │   ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐    ┌───┐   │   │
│         │                 │   │V1 │ │V2 │ │V3 │ │V4 │ │V5 │....│Vn │   │   │
│         │                 │   └─┬─┘ └─┬─┘ └─┬─┘ └─┬─┘ └─┬─┘    └─┬─┘   │   │
│         │                 │     │     │     │     │     │        │      │   │
│         │                 │     └─────┴─────┴──┬──┴─────┴────────┘      │   │
│         │                 │                    │                         │   │
│         │                 │              VRF Selection                   │   │
│         │                 │            (5-21 validators)                 │   │
│         │                 └────────────────────┬────────────────────────┘   │
│         │                                      │                             │
│         │                                      ▼                             │
│         │                 ┌─────────────────────────────────────────────┐   │
│         │                 │            STATE MANAGER                     │   │
│    Query│                 │                                              │   │
│   Facts │                 │  ┌──────────┐ ┌──────────┐ ┌──────────────┐ │   │
│         │                 │  │ Account  │ │Validator │ │  Knowledge   │ │   │
│         │                 │  │  Tree    │ │  Tree    │ │    Tree      │ │   │
│         │                 │  │(balances)│ │ (stakes) │ │   (facts)    │ │   │
│         │                 │  └──────────┘ └──────────┘ └──────────────┘ │   │
│         │                 └─────────────────────────────────────────────┘   │
│         │                                      │                             │
│         └──────────────────────────────────────┘                             │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
                    ┌─────────────────────────────────────┐
                    │           CLAIM SUBMISSION          │
                    │                                     │
                    │  claim: "2 + 2 = 4"                 │
                    │  domain: "math.arithmetic"          │
                    │  stake: 100 LGM                     │
                    │  references: ["axiom:math:zero"]    │
                    └──────────────┬──────────────────────┘
                                   │
                                   ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                                                                          │
│                        VERIFICATION PIPELINE                             │
│                                                                          │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐    ┌────────────┐   │
│  │            │    │            │    │            │    │            │   │
│  │  VALIDATOR │───▶│   COMMIT   │───▶│   REVEAL   │───▶│ AGGREGATE  │   │
│  │  SELECTION │    │   PHASE    │    │   PHASE    │    │  RESULTS   │   │
│  │            │    │            │    │            │    │            │   │
│  │  VRF-based │    │  4 blocks  │    │  4 blocks  │    │  2 blocks  │   │
│  │  5-21 vals │    │  submit    │    │  reveal    │    │  calculate │   │
│  │            │    │  hashes    │    │  verdicts  │    │ confidence │   │
│  └────────────┘    └────────────┘    └────────────┘    └────────────┘   │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
                    ┌──────────────┴──────────────┐
                    │                             │
                    ▼                             ▼
         ┌─────────────────┐           ┌─────────────────┐
         │    ACCEPTED     │           │    REJECTED     │
         │                 │           │                 │
         │ ≥80% confirmed  │           │ ≥80% rejected   │
         │ Create FACT     │           │ No fact created │
         │ Reward verifiers│           │ Slash submitter │
         └─────────────────┘           └─────────────────┘
```

---

## 2. Verification Round Lifecycle

### Timeline

```
Block:    N      N+1     N+2     N+3     N+4     N+5     N+6     N+7     N+8     N+9    N+10
          │       │       │       │       │       │       │       │       │       │       │
          ▼       ▼       ▼       ▼       ▼       ▼       ▼       ▼       ▼       ▼       ▼
    ┌─────┴───────┴───────┴───────┴───────┴───────┴───────┴───────┴───────┴───────┴───────┴─────┐
    │                                                                                            │
    │  ◀────────── COMMIT PHASE ──────────▶◀────────── REVEAL PHASE ──────────▶◀─ AGGREGATE ─▶  │
    │              (4 blocks)                          (4 blocks)                (2 blocks)      │
    │                                                                                            │
    │  Validators submit                   Validators reveal                   System computes   │
    │  commitment hashes                   actual verdicts                     final confidence  │
    │                                                                                            │
    └───────────────────────────────────────────────────────────────────────────────────────────┘
          │                                       │                                       │
          │                                       │                                       │
    START BLOCK                            COMMIT DEADLINE                         COMPLETION
    Round created                          No more commits                         Round closed
    Validators selected                    Reveal phase begins                     Results final
```

### Phase State Machine

```
                              ┌─────────────┐
                              │   PENDING   │
                              │             │
                              │ Round just  │
                              │ created     │
                              └──────┬──────┘
                                     │
                                     │ Block N arrives
                                     ▼
                              ┌─────────────┐
                              │   COMMIT    │
                              │             │
                              │ Validators  │◀─────────────────┐
                              │ submit      │                  │
                              │ commitments │──────────────────┤
                              └──────┬──────┘  Commitment      │
                                     │         received        │
                                     │
                                     │ Block N+4 (deadline)
                                     ▼
                              ┌─────────────┐
                              │   REVEAL    │
                              │             │
                              │ Validators  │◀─────────────────┐
                              │ reveal      │                  │
                              │ verdicts    │──────────────────┤
                              └──────┬──────┘  Reveal          │
                                     │         received        │
                                     │
                                     │ Block N+8 (deadline)
                                     ▼
                              ┌─────────────┐
                              │ AGGREGATION │
                              │             │
                              │ Calculate   │
                              │ confidence  │
                              │ & consensus │
                              └──────┬──────┘
                                     │
                                     │ Block N+10
                                     ▼
                              ┌─────────────┐
                              │  COMPLETE   │
                              │             │
                              │ Results     │
                              │ finalized   │
                              └─────────────┘
```

---

## 3. Validator Selection (VRF)

### VRF-Based Selection Process

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        VRF VALIDATOR SELECTION                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  INPUT SEED (deterministic but unpredictable):                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  seed = SHA512_256("verification:{claimId}:{blockNum}:{prevHash}")  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│                              │                                              │
│                              ▼                                              │
│                                                                             │
│  FOR EACH VALIDATOR:                                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │   ┌──────────────┐      ┌──────────────┐      ┌──────────────┐     │   │
│  │   │  Validator   │      │     VRF      │      │   Selection  │     │   │
│  │   │  Private Key │─────▶│   Compute    │─────▶│   Threshold  │     │   │
│  │   │              │      │              │      │              │     │   │
│  │   └──────────────┘      └──────────────┘      └──────────────┘     │   │
│  │                               │                      │              │   │
│  │                               ▼                      ▼              │   │
│  │                         ┌──────────┐          ┌──────────────┐     │   │
│  │                         │VRF Output│          │   Selected?  │     │   │
│  │                         │VRF Proof │          │              │     │   │
│  │                         └──────────┘          │ threshold =  │     │   │
│  │                                               │ f(stake/     │     │   │
│  │                                               │   totalStake)│     │   │
│  │                                               └──────────────┘     │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Selection Probability by Stake

```
Selection Probability vs Stake

  100% ┤                                                    ●────────
       │                                                 ●●●
       │                                              ●●●
   80% ┤                                           ●●●
       │                                        ●●●
       │                                     ●●●
   60% ┤                                  ●●●
       │                               ●●●
       │                            ●●●
   40% ┤                         ●●●
       │                      ●●●
       │                   ●●●
   20% ┤                ●●●
       │             ●●●
       │          ●●●
    0% ┼───●●●●●●●───────────────────────────────────────────────────
       0%      5%      10%     15%     20%     25%     30%     35%
                    Validator Stake / Total Stake

  Note: Higher stake = higher selection probability
        But selection is NOT deterministic (VRF randomness)
        Even small validators can be selected
```

### Selection Distribution Example

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    VALIDATOR SET (10 validators)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Validator    Stake      % of Total    Avg Selection    This Round          │
│  ─────────    ─────      ──────────    ─────────────    ──────────          │
│  V1           5000 LGM     25%            25%           ✓ SELECTED          │
│  V2           4000 LGM     20%            20%           ✓ SELECTED          │
│  V3           3000 LGM     15%            15%           ✓ SELECTED          │
│  V4           2000 LGM     10%            10%           ✓ SELECTED          │
│  V5           2000 LGM     10%            10%           ✗ not selected      │
│  V6           1500 LGM     7.5%           7.5%          ✓ SELECTED          │
│  V7           1000 LGM     5%             5%            ✗ not selected      │
│  V8            700 LGM     3.5%           3.5%          ✗ not selected      │
│  V9            500 LGM     2.5%           2.5%          ✓ SELECTED          │
│  V10           300 LGM     1.5%           1.5%          ✗ not selected      │
│  ─────────    ─────────   ─────────                                         │
│  TOTAL       20000 LGM    100%           Selected: 6 validators             │
│                                                                             │
│  Note: V9 (2.5% stake) was selected while V5 (10% stake) was not           │
│        This is the VRF randomness at work - unpredictable but fair         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Commit-Reveal Scheme

### Why Commit-Reveal?

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    WITHOUT COMMIT-REVEAL (BAD)                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Time ──────────────────────────────────────────────────────────────────▶   │
│                                                                             │
│  V1: "I vote CONFIRMED"                                                     │
│           │                                                                 │
│           ▼                                                                 │
│  V2: "V1 voted confirmed... I'll vote CONFIRMED too"  ◀── HERDING          │
│           │                                                                 │
│           ▼                                                                 │
│  V3: "Everyone's confirming... CONFIRMED"  ◀── NO INDEPENDENT THOUGHT      │
│           │                                                                 │
│           ▼                                                                 │
│  V4: "I'll sell my vote - pay me and I'll prove I voted your way"          │
│                                              ◀── VOTE BUYING                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                     WITH COMMIT-REVEAL (GOOD)                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  COMMIT PHASE:                                                              │
│  V1: commits hash("confirmed" + secret_nonce_1) = 0x7a3f...                │
│  V2: commits hash("rejected" + secret_nonce_2)  = 0x9c2b...                │
│  V3: commits hash("confirmed" + secret_nonce_3) = 0x1d8e...                │
│  V4: commits hash("uncertain" + secret_nonce_4) = 0x4f7a...                │
│                                                                             │
│  ◀── No one knows how others voted! Can't copy, can't prove vote.          │
│                                                                             │
│  REVEAL PHASE:                                                              │
│  V1: reveals "confirmed" + secret_nonce_1  ✓ matches commitment            │
│  V2: reveals "rejected" + secret_nonce_2   ✓ matches commitment            │
│  V3: reveals "confirmed" + secret_nonce_3  ✓ matches commitment            │
│  V4: reveals "uncertain" + secret_nonce_4  ✓ matches commitment            │
│                                                                             │
│  ◀── All votes were independent! No herding possible.                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Commitment Structure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         COMMITMENT GENERATION                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  VALIDATOR'S PRIVATE DATA:                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  verdict: "confirmed"                                               │   │
│  │  confidence: 0.85                                                   │   │
│  │  nonce: "a7b3c9d2e1f0..."  (random 32 bytes - KEEP SECRET!)        │   │
│  │  referencesChecked: ["axiom:math:zero", "axiom:math:successor"]    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                              │                                              │
│                              ▼                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      HASH FUNCTION                                  │   │
│  │                                                                     │   │
│  │  commitment = SHA512_256(                                          │   │
│  │    domain_hash("LGM.verification.commitment.v1",                   │   │
│  │      {roundId, verdict, confidence, nonce, referencesChecked}      │   │
│  │    )                                                               │   │
│  │  )                                                                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                              │                                              │
│                              ▼                                              │
│  PUBLIC COMMITMENT:                                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  commitment: "0x7a3f9b2c1d8e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c..."   │   │
│  │  signature: validator's ED25519 signature                          │   │
│  │  vrfProof: proof of valid selection                                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ◀── Only the hash is public. Actual vote is hidden until reveal.          │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Reveal Verification

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          REVEAL VERIFICATION                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  VALIDATOR REVEALS:                                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  verdict: "confirmed"                                               │   │
│  │  confidence: 0.85                                                   │   │
│  │  nonce: "a7b3c9d2e1f0..."                                          │   │
│  │  referencesChecked: ["axiom:math:zero", "axiom:math:successor"]    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                              │                                              │
│                              ▼                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    SYSTEM VERIFICATION                              │   │
│  │                                                                     │   │
│  │  1. Recompute: expected = SHA512_256(domain_hash(...))             │   │
│  │                                                                     │   │
│  │  2. Compare:   expected === stored_commitment ?                    │   │
│  │                                                                     │   │
│  │     ┌─────────────┐              ┌─────────────┐                   │   │
│  │     │  MATCH ✓    │              │ MISMATCH ✗  │                   │   │
│  │     │             │              │             │                   │   │
│  │     │ Accept vote │              │ SLASH       │                   │   │
│  │     │ Include in  │              │ validator   │                   │   │
│  │     │ aggregation │              │ (cheating)  │                   │   │
│  │     └─────────────┘              └─────────────┘                   │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Confidence Aggregation

### The Master Formula

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      CONFIDENCE CALCULATION                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│                                                                             │
│   C_final = min( C_ceiling,  C_base × M_temporal × M_adversarial )         │
│                                                                             │
│                                                                             │
│   Where:                                                                    │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  C_ceiling = Epistemic maximum for knowledge category               │  │
│   │                                                                     │  │
│   │    analytic (math):     1.00  ─────────────────────────────── 100%  │  │
│   │    formal (proofs):     1.00  ─────────────────────────────── 100%  │  │
│   │    protocol:            1.00  ─────────────────────────────── 100%  │  │
│   │    empirical:           0.99  ──────────────────────────────  99%  │  │
│   │    historical:          0.95  ────────────────────────────    95%  │  │
│   │    predictive:          0.90  ──────────────────────          90%  │  │
│   │    social:              0.85  ────────────────                85%  │  │
│   │                                                                     │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  C_base = Combine(verification_confidence, evidence_confidence)    │  │
│   │                                                                     │  │
│   │    = (verification × 0.60) + (evidence × 0.40)                     │  │
│   │                                                                     │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  M_temporal = Decay factor (only for non-eternal truths)           │  │
│   │                                                                     │  │
│   │    For analytic/formal: 1.0 (no decay - eternal truths)            │  │
│   │    For empirical: 0.5^(blocks_since / half_life)                   │  │
│   │                                                                     │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  M_adversarial = Sybil adjustment × Manipulation adjustment        │  │
│   │                                                                     │  │
│   │    Range: 0.25 to 1.0 (reduces confidence if attack detected)      │  │
│   │                                                                     │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Verification Confidence Calculation

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                   VERIFICATION CONFIDENCE                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  INPUTS: Validator reveals with verdicts and stakes                         │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  V1: verdict=confirmed, confidence=0.90, stake=1000                 │   │
│  │  V2: verdict=confirmed, confidence=0.85, stake=800                  │   │
│  │  V3: verdict=confirmed, confidence=0.80, stake=600                  │   │
│  │  V4: verdict=rejected,  confidence=0.70, stake=400                  │   │
│  │  V5: verdict=uncertain, confidence=0.50, stake=200                  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                              │                                              │
│                              ▼                                              │
│  CALCULATION:                                                               │
│                                                                             │
│  1. STAKE-WEIGHTED CONFIDENCE (50% weight):                                 │
│     ┌───────────────────────────────────────────────────────────────────┐  │
│     │  Confirmed stake: 1000 + 800 + 600 = 2400                         │  │
│     │  Total stake: 3000                                                │  │
│     │                                                                   │  │
│     │  Weighted avg of confirmed:                                       │  │
│     │    (0.90×1000 + 0.85×800 + 0.80×600) / 2400 = 0.858               │  │
│     │                                                                   │  │
│     │  stake_weighted = 0.858 × 0.50 = 0.429                           │  │
│     └───────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  2. REPUTATION-WEIGHTED (30% weight):                                       │
│     ┌───────────────────────────────────────────────────────────────────┐  │
│     │  (Similar calculation using reputation scores)                    │  │
│     │  reputation_weighted = 0.82 × 0.30 = 0.246                       │  │
│     └───────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  3. AGREEMENT BONUS (20% weight):                                           │
│     ┌───────────────────────────────────────────────────────────────────┐  │
│     │  Agreement level = 2400/3000 = 80%                                │  │
│     │  Bonus = max(0, (0.80 - 0.50) × 2) = 0.60                        │  │
│     │  agreement_contribution = 0.60 × 0.20 = 0.12                     │  │
│     └───────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  FINAL: 0.429 + 0.246 + 0.12 = 0.795 (79.5% verification confidence)       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Temporal Decay (for Empirical Facts)

```
Confidence Decay Over Time (Empirical Facts)

  1.0 ┤●
      │ ●●
      │   ●●
  0.8 ┤     ●●
      │       ●●
      │         ●●
  0.6 ┤           ●●
      │             ●●
      │               ●●
  0.4 ┤                 ●●●
      │                    ●●●
      │                       ●●●
  0.2 ┤                          ●●●●
      │                              ●●●●●
      │                                   ●●●●●●●●
  0.0 ┼────────────────────────────────────────────────────────────────▶
      0        50k       100k      150k      200k      250k      300k
                              Blocks Since Verification

      Half-life = 100,000 blocks (~3 days)

      At 100k blocks: 50% retained
      At 200k blocks: 25% retained
      At 300k blocks: 12.5% retained

      NOTE: Analytic/Formal facts do NOT decay (eternal truths)
```

---

## 6. Consensus Decision Tree

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        CONSENSUS DECISION TREE                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│                    ┌───────────────────────┐                                │
│                    │  Aggregation Phase    │                                │
│                    │  Complete             │                                │
│                    └───────────┬───────────┘                                │
│                                │                                            │
│                                ▼                                            │
│                    ┌───────────────────────┐                                │
│                    │  Participation ≥ 66%? │                                │
│                    └───────────┬───────────┘                                │
│                                │                                            │
│              ┌─────────────────┼─────────────────┐                          │
│              │ NO              │                 │ YES                      │
│              ▼                 │                 ▼                          │
│    ┌─────────────────┐        │       ┌─────────────────┐                  │
│    │  INSUFFICIENT   │        │       │ Confirmed ≥ 80%?│                  │
│    │                 │        │       └────────┬────────┘                  │
│    │ Not enough      │        │                │                           │
│    │ validators      │        │    ┌───────────┼───────────┐               │
│    │ participated    │        │    │ YES       │           │ NO            │
│    │                 │        │    ▼           │           ▼               │
│    │ • No fact       │        │ ┌──────────┐   │    ┌─────────────────┐    │
│    │ • No rewards    │        │ │ ACCEPTED │   │    │ Rejected ≥ 80%? │    │
│    │ • Can resubmit  │        │ └──────────┘   │    └────────┬────────┘    │
│    └─────────────────┘        │      │        │             │              │
│                               │      ▼        │    ┌────────┼────────┐     │
│                               │ • Create FACT │    │ YES    │        │ NO  │
│                               │ • confidence  │    ▼        │        ▼     │
│                               │   calculated  │ ┌──────────┐│ ┌──────────┐ │
│                               │ • Reward      │ │ REJECTED ││ │ CONTESTED│ │
│                               │   correct     │ └──────────┘│ └──────────┘ │
│                               │   validators  │      │      │      │       │
│                               │ • Slash wrong │      ▼      │      ▼       │
│                               │   validators  │ • No fact   │ • No fact    │
│                               │ • Return      │ • Slash     │ • No slash   │
│                               │   submitter   │   submitter │ • Disputed   │
│                               │   stake       │ • Reward    │   knowledge  │
│                               │               │   correct   │ • Can        │
│                               │               │   validators│   resubmit   │
│                               │               │             │              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Threshold Parameters

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         CONSENSUS THRESHOLDS                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Parameter                 Value     Description                            │
│  ─────────────────────────────────────────────────────────────────────────  │
│  validatorQuorum           66%       Min participation required             │
│  acceptanceThreshold       80%       Confirmed stake for acceptance         │
│  rejectionThreshold        80%       Rejected stake for rejection           │
│  contestedThreshold        60%       Below this = contested                 │
│                                                                             │
│                                                                             │
│  VISUAL REPRESENTATION:                                                     │
│                                                                             │
│  0%        20%        40%        60%        80%        100%                 │
│  │          │          │          │          │          │                   │
│  ├──────────┴──────────┴──────────┼──────────┼──────────┤                   │
│  │                                │          │          │                   │
│  │◀────── CONTESTED ─────────────▶│          │◀ACCEPTED▶│                   │
│  │         (no decision)          │          │          │                   │
│  │                                │          │          │                   │
│  │◀REJECTED▶│                     │          │          │                   │
│  │(if 80%+  │                     │          │          │                   │
│  │ rejected)│                     │          │          │                   │
│  │          │                     │          │          │                   │
│  └──────────┴─────────────────────┴──────────┴──────────┘                   │
│                                                                             │
│  Example scenarios:                                                         │
│    85% confirmed, 15% rejected  → ACCEPTED                                  │
│    85% rejected, 15% confirmed  → REJECTED                                  │
│    65% confirmed, 35% rejected  → CONTESTED                                 │
│    55% confirmed, 45% rejected  → CONTESTED                                 │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. State Transitions

### Claim Lifecycle State Machine

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        CLAIM STATE MACHINE                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│                         ┌───────────────┐                                   │
│                         │   SUBMITTED   │                                   │
│                         │               │                                   │
│                         │ Stake locked  │                                   │
│                         │ Pending queue │                                   │
│                         └───────┬───────┘                                   │
│                                 │                                           │
│                                 │ Round created                             │
│                                 ▼                                           │
│                         ┌───────────────┐                                   │
│                         │  VERIFYING    │                                   │
│                         │               │                                   │
│                         │ Active round  │                                   │
│                         │ Validators    │                                   │
│                         │ voting        │                                   │
│                         └───────┬───────┘                                   │
│                                 │                                           │
│              ┌──────────────────┼──────────────────┐                        │
│              │                  │                  │                        │
│              ▼                  ▼                  ▼                        │
│     ┌────────────────┐ ┌────────────────┐ ┌────────────────┐               │
│     │    ACCEPTED    │ │    REJECTED    │ │   CONTESTED    │               │
│     │                │ │                │ │                │               │
│     │ • Fact created │ │ • No fact      │ │ • No fact      │               │
│     │ • Stake return │ │ • Stake slash  │ │ • Stake return │               │
│     │ • Rewards vest │ │   (20%)        │ │ • Can resubmit │               │
│     └───────┬────────┘ └────────────────┘ └────────┬───────┘               │
│             │                                      │                        │
│             │                                      │ Resubmit               │
│             │                                      └───────▶ SUBMITTED      │
│             │                                                               │
│             │ Challenge filed                                               │
│             ▼                                                               │
│     ┌────────────────┐                                                      │
│     │   CHALLENGED   │                                                      │
│     │                │                                                      │
│     │ Dispute system │                                                      │
│     │ activated      │                                                      │
│     └───────┬────────┘                                                      │
│             │                                                               │
│      ┌──────┴──────┐                                                        │
│      │             │                                                        │
│      ▼             ▼                                                        │
│ ┌─────────┐  ┌─────────────┐                                               │
│ │DEFENDED │  │ FALSIFIED   │                                               │
│ │         │  │             │                                               │
│ │Fact     │  │ Fact marked │                                               │
│ │stands   │  │ invalid     │                                               │
│ │         │  │ Rewards     │                                               │
│ │         │  │ clawed back │                                               │
│ └─────────┘  └─────────────┘                                               │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### State Tree Updates

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    STATE UPDATES ON FACT ACCEPTANCE                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  BEFORE:                              AFTER:                                │
│                                                                             │
│  ┌─────────────────────┐              ┌─────────────────────┐              │
│  │   ACCOUNT TREE      │              │   ACCOUNT TREE      │              │
│  ├─────────────────────┤              ├─────────────────────┤              │
│  │ Submitter:          │              │ Submitter:          │              │
│  │   balance: 1000 LGM │              │   balance: 900 LGM  │◀─stake locked│
│  │   staked: 0         │              │   staked: 100 LGM   │              │
│  │                     │              │                     │              │
│  │ Validator1:         │              │ Validator1:         │              │
│  │   balance: 500 LGM  │              │   balance: 500.78   │◀─reward      │
│  │                     │              │                     │              │
│  │ Research Treasury:  │              │ Research Treasury:  │              │
│  │   balance: 10000    │              │   balance: 10000.22 │◀─22% share   │
│  └─────────────────────┘              └─────────────────────┘              │
│                                                                             │
│  ┌─────────────────────┐              ┌─────────────────────┐              │
│  │  KNOWLEDGE TREE     │              │  KNOWLEDGE TREE     │              │
│  ├─────────────────────┤              ├─────────────────────┤              │
│  │ (empty for domain)  │              │ fact_abc123:        │◀─NEW FACT    │
│  │                     │              │   claim: "2+2=4"    │              │
│  │                     │              │   domain: math      │              │
│  │                     │              │   confidence: 0.87  │              │
│  │                     │              │   verifiedAt: 1000  │              │
│  │                     │              │   status: active    │              │
│  └─────────────────────┘              └─────────────────────┘              │
│                                                                             │
│  ┌─────────────────────┐              ┌─────────────────────┐              │
│  │  VALIDATOR TREE     │              │  VALIDATOR TREE     │              │
│  ├─────────────────────┤              ├─────────────────────┤              │
│  │ Validator1:         │              │ Validator1:         │              │
│  │   correctVotes: 10  │              │   correctVotes: 11  │◀─incremented │
│  │   reputation: 0.8   │              │   reputation: 0.82  │◀─improved    │
│  │                     │              │                     │              │
│  │ Validator2 (wrong): │              │ Validator2:         │              │
│  │   stake: 500 LGM    │              │   stake: 450 LGM    │◀─slashed 10% │
│  │   slashCount: 0     │              │   slashCount: 1     │              │
│  └─────────────────────┘              └─────────────────────┘              │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Economic Incentives

### Reward Distribution

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         REWARD DISTRIBUTION                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│                    ┌─────────────────────────────┐                          │
│                    │    VERIFICATION REWARD      │                          │
│                    │         1 LGM               │                          │
│                    └──────────────┬──────────────┘                          │
│                                   │                                         │
│                    ┌──────────────┴──────────────┐                          │
│                    │                             │                          │
│                    ▼                             ▼                          │
│           ┌────────────────┐           ┌────────────────┐                  │
│           │   VALIDATOR    │           │    RESEARCH    │                  │
│           │     78%        │           │    TREASURY    │                  │
│           │   0.78 LGM     │           │      22%       │                  │
│           └────────────────┘           │   0.22 LGM     │                  │
│                                        └────────────────┘                  │
│                                                                             │
│                                                                             │
│  CLAIM SUBMITTER REWARDS (on acceptance):                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │    Initial stake returned + Bonus rewards                           │   │
│  │                                                                     │   │
│  │    ┌─────────────────────────────────────────────────────────┐     │   │
│  │    │              VESTING SCHEDULE                           │     │   │
│  │    │                                                         │     │   │
│  │    │  Cliff: 1000 blocks (~42 minutes)                       │     │   │
│  │    │  Duration: 10000 blocks (~7 hours)                      │     │   │
│  │    │                                                         │     │   │
│  │    │  100% ┤                              ●●●●●●●●●●●●●●     │     │   │
│  │    │      │                         ●●●●●                    │     │   │
│  │    │  75% ┤                    ●●●●●                         │     │   │
│  │    │      │               ●●●●●                              │     │   │
│  │    │  50% ┤          ●●●●●                                   │     │   │
│  │    │      │     ●●●●●                                        │     │   │
│  │    │  25% ┤●●●●●                                             │     │   │
│  │    │      │                                                  │     │   │
│  │    │   0% ┼───────┬─────────────────────────────────────────▶│     │   │
│  │    │      0    1000                                    10000 │     │   │
│  │    │           cliff                                 vested  │     │   │
│  │    │                        Blocks                           │     │   │
│  │    └─────────────────────────────────────────────────────────┘     │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Slashing Conditions

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         SLASHING CONDITIONS                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  OFFENSE                          PENALTY           CONSEQUENCE             │
│  ────────────────────────────────────────────────────────────────────────   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  VALIDATOR OFFENSES                                                 │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  Wrong verification         10% of stake       Reputation damage    │   │
│  │  (voted against consensus)                     +1 slash count       │   │
│  │                                                                     │   │
│  │  Missed verification        5% of stake        Reputation damage    │   │
│  │  (committed but no reveal)                     +1 slash count       │   │
│  │                                                                     │   │
│  │  Commitment mismatch        20% of stake       Severe reputation    │   │
│  │  (reveal doesn't match)                        damage, +2 slash     │   │
│  │                                                                     │   │
│  │  Double voting              50% of stake       Deactivation         │   │
│  │  (two commitments)                                                  │   │
│  │                                                                     │   │
│  │  3+ slash counts            Deactivation       Must re-register     │   │
│  │                                                with full stake      │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  CLAIM SUBMITTER OFFENSES                                           │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  Invalid claim              20% of stake       Reputation damage    │   │
│  │  (rejected by consensus)                                            │   │
│  │                                                                     │   │
│  │  Duplicate claim            50% of stake       Severe reputation    │   │
│  │  (already verified)                            damage               │   │
│  │                                                                     │   │
│  │  Spam claims                100% of stake      Account flagged      │   │
│  │  (repeated rejections)                                              │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  CHALLENGE OUTCOMES                                                 │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  Challenger loses           30% of challenge   Reputation damage    │   │
│  │  (fact defended)            stake                                   │   │
│  │                                                                     │   │
│  │  Defender loses             Full clawback of   Fact marked          │   │
│  │  (fact falsified)           original rewards   falsified            │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Game Theory Equilibrium

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      INCENTIVE ALIGNMENT                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  HONEST VERIFICATION IS THE DOMINANT STRATEGY:                              │
│                                                                             │
│                        Consensus Outcome                                    │
│                    ┌────────────┬────────────┐                              │
│                    │  Confirmed │  Rejected  │                              │
│  ┌─────────────────┼────────────┼────────────┤                              │
│  │ You vote        │            │            │                              │
│  │ CONFIRMED       │  +1 LGM ✓  │  -10% ✗    │                              │
│  │                 │  (reward)  │  (slash)   │                              │
│  ├─────────────────┼────────────┼────────────┤                              │
│  │ You vote        │            │            │                              │
│  │ REJECTED        │  -10% ✗    │  +1 LGM ✓  │                              │
│  │                 │  (slash)   │  (reward)  │                              │
│  ├─────────────────┼────────────┼────────────┤                              │
│  │ You vote        │            │            │                              │
│  │ UNCERTAIN       │   0 →      │   0 →      │                              │
│  │                 │  (safe)    │  (safe)    │                              │
│  └─────────────────┴────────────┴────────────┘                              │
│                                                                             │
│  ANALYSIS:                                                                  │
│  • If you KNOW the truth: Vote honestly → maximize reward                   │
│  • If you DON'T know: Vote "uncertain" → avoid slash                        │
│  • Trying to game: High risk of -10% slash                                  │
│  • Following the crowd: Commit-reveal prevents this!                        │
│                                                                             │
│  NASH EQUILIBRIUM: All validators vote honestly based on their              │
│                    independent assessment of the claim's truth.             │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 9. Finality Model

### Block Finality

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         FINALITY MODEL                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  PROBABILISTIC FINALITY (19 confirmations):                                 │
│                                                                             │
│  Block   Confirmations   P(Reorg)        Finality                           │
│  ─────   ─────────────   ────────        ────────                           │
│  N       0               ~33%            ~67%                               │
│  N-1     1               ~11%            ~89%                               │
│  N-2     2               ~3.7%           ~96%                               │
│  N-5     5               ~0.39%          ~99.6%                             │
│  N-10    10              ~0.0017%        ~99.998%                           │
│  N-19    19              ~0.0000001%     ~99.9999999%  ◀── FINALIZED        │
│                                                                             │
│                                                                             │
│  TIMELINE:                                                                  │
│                                                                             │
│  Block time: 2521ms                                                        │
│  Finality depth: 19 blocks                                                 │
│  Time to finality: ~48 seconds                                              │
│                                                                             │
│                                                                             │
│  ┌─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┐            │
│  │  1  │  2  │  3  │  4  │  5  │ ... │ 17  │ 18  │ 19  │ 20  │            │
│  └──┬──┴──┬──┴──┬──┴──┬──┴──┬──┴─────┴──┬──┴──┬──┴──┬──┴──┬──┘            │
│     │     │     │     │     │           │     │     │     │                │
│     │     │     │     │     │           │     │     │     │                │
│  ◀──┴─────┴─────┴─────┴─────┴───────────┴─────┴─────┴─────┴──▶             │
│                         ~48 seconds                                         │
│                                                                             │
│  Block 1: PROPOSED (not final)                                              │
│  Block 19: FINALIZED (immutable)                                            │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Checkpoint Ceremony

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       CHECKPOINT CEREMONY                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Checkpoints provide INSTANT finality for historical blocks:                │
│                                                                             │
│                                                                             │
│       Block 0        Block 10000      Block 20000      Block 30000          │
│          │               │                │                │                │
│          ▼               ▼                ▼                ▼                │
│       ┌─────┐         ┌─────┐          ┌─────┐          ┌─────┐            │
│       │ CP1 │─────────│ CP2 │──────────│ CP3 │──────────│ CP4 │            │
│       └─────┘         └─────┘          └─────┘          └─────┘            │
│          │               │                │                │                │
│     Genesis          Checkpoint       Checkpoint       Checkpoint           │
│                      Ceremony 1       Ceremony 2       Ceremony 3           │
│                                                                             │
│                                                                             │
│  CHECKPOINT STRUCTURE:                                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  blockNumber: 10000                                                 │   │
│  │  blockHash: "0x7a3f..."                                             │   │
│  │  stateRoot: "0x9c2b..."                                             │   │
│  │  validatorSetHash: "0x1d8e..."                                      │   │
│  │  timestamp: 1704067200                                              │   │
│  │  signatures: [                                                      │   │
│  │    { validator: V1, sig: "0x..." },   ◀── 2/3+ validator signatures │   │
│  │    { validator: V2, sig: "0x..." },                                 │   │
│  │    { validator: V3, sig: "0x..." },                                 │   │
│  │    ...                                                              │   │
│  │  ]                                                                  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  BENEFITS:                                                                  │
│  • Light clients can sync from checkpoints                                  │
│  • Instant finality for blocks before checkpoint                            │
│  • Reduces storage for historical verification                              │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 10. Challenge & Dispute System

### Challenge Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         CHALLENGE SYSTEM                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│                    ┌─────────────────────────┐                              │
│                    │     ACCEPTED FACT       │                              │
│                    │                         │                              │
│                    │  "The Earth is flat"    │                              │
│                    │  confidence: 0.85       │                              │
│                    │  status: active         │                              │
│                    └───────────┬─────────────┘                              │
│                                │                                            │
│                                │ Challenge filed                            │
│                                │ (stake: 200 LGM)                           │
│                                ▼                                            │
│                    ┌─────────────────────────┐                              │
│                    │    DISPUTE INITIATED    │                              │
│                    │                         │                              │
│                    │  Fact status: challenged│                              │
│                    │  Challenge stake locked │                              │
│                    └───────────┬─────────────┘                              │
│                                │                                            │
│         ┌──────────────────────┼──────────────────────┐                     │
│         │                      │                      │                     │
│         ▼                      ▼                      ▼                     │
│  ┌─────────────┐       ┌─────────────┐       ┌─────────────┐               │
│  │   TIER 1    │       │   TIER 2    │       │   TIER 3    │               │
│  │  (5 arbiters)│──────▶│ (11 arbiters)│──────▶│ (21 arbiters)│              │
│  │             │Appeal │             │Appeal │             │               │
│  │ Quick       │       │ Standard    │       │ Full        │               │
│  │ resolution  │       │ review      │       │ tribunal    │               │
│  └─────────────┘       └─────────────┘       └─────────────┘               │
│         │                      │                      │                     │
│         ▼                      ▼                      ▼                     │
│  ┌─────────────────────────────────────────────────────────┐               │
│  │                    RESOLUTION                           │               │
│  ├─────────────────────────────────────────────────────────┤               │
│  │                                                         │               │
│  │  CHALLENGER WINS:              DEFENDER WINS:           │               │
│  │  • Fact → falsified            • Fact remains active    │               │
│  │  • Original rewards clawed     • Challenger loses 30%   │               │
│  │  • Challenger gets 60% of      • Defender keeps rewards │               │
│  │    defender stake + bonus      • Arbiters rewarded      │               │
│  │  • Arbiters rewarded                                    │               │
│  │                                                         │               │
│  └─────────────────────────────────────────────────────────┘               │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Dispute State Machine

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      DISPUTE STATE MACHINE                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────┐                                                           │
│  │   OPEN       │  Challenge filed, waiting for response                    │
│  └──────┬───────┘                                                           │
│         │                                                                   │
│         │ Defender responds OR timeout                                      │
│         ▼                                                                   │
│  ┌──────────────┐                                                           │
│  │  EVIDENCE    │  Both parties submit evidence                             │
│  │  SUBMISSION  │  Commit-reveal for evidence                               │
│  └──────┬───────┘                                                           │
│         │                                                                   │
│         │ Evidence deadline                                                 │
│         ▼                                                                   │
│  ┌──────────────┐                                                           │
│  │   VOTING     │  Arbiters vote on outcome                                 │
│  │              │  Commit-reveal for votes                                  │
│  └──────┬───────┘                                                           │
│         │                                                                   │
│         │ Voting complete                                                   │
│         ▼                                                                   │
│  ┌──────────────┐                                                           │
│  │  RESOLUTION  │  Winner determined, stakes distributed                    │
│  └──────┬───────┘                                                           │
│         │                                                                   │
│    ┌────┴────┐                                                              │
│    │         │                                                              │
│    ▼         ▼                                                              │
│ ┌──────┐  ┌──────┐                                                          │
│ │FINAL │  │APPEAL│ ───▶ Escalate to next tier                              │
│ └──────┘  └──────┘      (requires additional stake)                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Summary

The Proof of Truth consensus system creates a virtuous cycle where:

1. **Truth verification becomes economically valuable** - Validators earn rewards for honest assessment
2. **Dishonesty is punished** - Slashing creates strong deterrents against lying
3. **Knowledge accumulates** - Accepted facts form a growing knowledge base
4. **Uncertainty is acknowledged** - The "uncertain" vote allows validators to admit when they don't know
5. **Challenges keep facts honest** - Even accepted facts can be falsified if proven wrong

The system's elegance comes from aligning economic incentives with epistemic virtue: **the best way to make money is to tell the truth**.

---

*Document Version: 1.0*
*Last Updated: 2026-02-04*
*Protocol Version: 0.1.0*
