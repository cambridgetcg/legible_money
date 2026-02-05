# Dispute Engine - Deep Dive Insights

> **Analysis Date**: 2026-02-03
> **Total Lines**: ~4,200 across 8 files
> **Status**: Complete implementation with GAP-015/016 fixes

## Core Philosophy

> *"Intention does not matter, result does."*
> *"Harsh punishment for falsehood, generous reward for truth."*
> *"The loser loses, not the good-faith actor."*

The Dispute Engine resolves conflicts about truth in the knowledge base. It implements economic incentives that make lying unprofitable and truth-discovery rewarding.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     DISPUTE ENGINE                           │
│              (Truth vs Falsehood Resolution)                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                  STATE MACHINE                       │    │
│  │                                                      │    │
│  │  INITIATION → EVIDENCE_COMMIT → EVIDENCE_REVEAL     │    │
│  │       ↓                                              │    │
│  │  ARBITRATION → DELIBERATION → SETTLEMENT/ESCALATED  │    │
│  │       ↓                                              │    │
│  │            RESOLVED (winner determined)              │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐               │
│  │ EVIDENCE  │  │ ARBITER   │  │ ESCALATION│               │
│  │           │  │           │  │           │               │
│  │ · Commit  │  │ · VRF     │  │ · Tier 1  │               │
│  │ · Reveal  │  │ · Domain  │  │ · Tier 2  │               │
│  │ · Compare │  │ · Stake   │  │ · Tier 3  │               │
│  └───────────┘  └───────────┘  └───────────┘               │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                   SETTLEMENT                         │    │
│  │  Winner Rewards │ Loser Penalties │ Arbiter Payouts  │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

---

## File Structure

| File | Lines | Purpose |
|------|-------|---------|
| `types.ts` | 697 | Core types, economics constants |
| `dispute_engine.ts` | 831 | Main orchestrator |
| `state_machine.ts` | 582 | Phase transitions, timeout handling |
| `evidence_comparison.ts` | 469 | 5-dimensional evidence scoring |
| `arbitrator_selection.ts` | 433 | VRF-based arbiter selection |
| `settlement.ts` | 411 | Rewards and punishments |
| `escalation.ts` | 434 | 3-tier appeal system |
| `index.ts` | ~50 | Module exports |

---

## The 8-Phase State Machine

```
INITIATION ──────> EVIDENCE_COMMIT ──────> EVIDENCE_REVEAL
    │                    │                       │
    │   Challenger       │   Both parties        │   Evidence
    │   stakes claim     │   commit hashes       │   revealed
    │                    │                       │
    └────────────────────┴───────────────────────┘
                              │
                              ▼
ARBITRATION ──────> DELIBERATION ──────> SETTLEMENT
    │                    │                   │
    │   VRF selects      │   Arbiters vote   │   Rewards &
    │   arbiters         │   (commit-reveal) │   penalties
    │                    │                   │
    └────────────────────┴───────────────────┘
                              │
              ┌───────────────┴───────────────┐
              ▼                               ▼
         ESCALATED                        RESOLVED
              │                               │
              │   Appeal to                   │   Final
              │   higher tier                 │   outcome
              │                               │
              └───────────────────────────────┘
```

### Phase Transitions

| From | To | Trigger |
|------|-----|---------|
| `initiation` | `evidence_commit` | Defender responds with counter-stake |
| `evidence_commit` | `evidence_reveal` | Both parties commit evidence hashes |
| `evidence_reveal` | `arbitration` | Both reveal or timeout |
| `arbitration` | `deliberation` | Arbiters selected via VRF |
| `deliberation` | `settlement` | Quorum reached OR timeout |
| `deliberation` | `escalated` | Party appeals with additional stake |
| `settlement` | `resolved` | Rewards/penalties distributed |
| `escalated` | `arbitration` | New tier arbiters selected |

---

## Economic Constants

### Punishment Rates (Harsh)

| Violation | Slash Rate | Rationale |
|-----------|------------|-----------|
| False claim | 50% | "You polluted the knowledge base" |
| Failed challenge | 30% | "You wasted everyone's time" |
| Wrong arbitration | 15% | "You failed your duty" |
| Refusal to reveal | 100% | "Bad faith actor" |
| Evidence tampering | 100% | "Maximum dishonesty" |

```typescript
export const PUNISHMENT_RATES = {
  falseClaimSlash: 0.50,
  failedChallengeSlash: 0.30,
  wrongArbitrationSlash: 0.15,
  refusalToRevealSlash: 1.00,
  evidenceTamperingSlash: 1.00,
};
```

### Reward Rates (Generous)

| Achievement | Bonus | Rationale |
|-------------|-------|-----------|
| Winner share | 60% of loser stake | "Truth pays" |
| Truth bonus | +50% | "Exposing falsehood" |
| Early detection | +20% | "Catching within 1000 blocks" |
| Correct arbitration | 10% of stake | "Incentivize accuracy" |

```typescript
export const REWARD_RATES = {
  winnerShare: 0.60,
  truthBonusMultiplier: 1.50,
  earlyDetectionBonus: 0.20,
  arbiterCorrectBonus: 0.10,
};
```

### Economic Impact

```
Example: 1000 LGM dispute

If Challenger wins (exposed falsehood early):
  Base reward:    600 LGM (60% of 1000)
  Truth bonus:   +300 LGM (+50%)
  Early bonus:   +120 LGM (+20%)
  ─────────────────────────
  Total:        1,020 LGM

Defender loses:
  Penalty:       500 LGM (50% slash)
  Protocol fee:   25 LGM (5% of penalty)
  Research fund:  10 LGM (2% of penalty)
```

---

## Evidence Comparison System

### 5 Dimensions of Evidence Strength

```
┌─────────────────────────────────────────────────────────────┐
│              EVIDENCE COMPARISON WEIGHTS                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  TYPE (25%)         ████████████░░░░░░░░░░░░░░░░░░░░░░░░░  │
│  testimony < document < record < verified_data < cryptographic│
│                                                              │
│  RELIABILITY (30%)  ████████████████░░░░░░░░░░░░░░░░░░░░░  │
│  Source credibility, verifiability, methodology              │
│                                                              │
│  RECENCY (15%)      ████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │
│  Newer evidence > older (with exponential decay)             │
│                                                              │
│  INDEPENDENCE (20%) ████████████░░░░░░░░░░░░░░░░░░░░░░░░░  │
│  How independent from the party submitting                   │
│                                                              │
│  CORROBORATION (10%)█████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │
│  Supporting evidence from other sources                      │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Evidence Type Ranking

```typescript
const TYPE_SCORES: Record<EvidenceType, number> = {
  'testimony':        0.3,  // Lowest - human memory fallible
  'document':         0.5,  // Can be forged
  'record':           0.6,  // Official records
  'verified_data':    0.8,  // Verified by trusted source
  'cryptographic':    1.0,  // Highest - mathematically certain
};
```

### Automatic Resolution

If evidence comparison shows a clear winner (>15% advantage), the dispute can auto-resolve without arbitration:

```typescript
export const CLEAR_WINNER_THRESHOLD = 0.15;

// If defender: 0.70, challenger: 0.45
// Difference: 0.25 > 0.15
// → Defender wins automatically
```

### GAP-015: Evidence Validation

```typescript
// Whitelist of valid evidence types
const VALID_EVIDENCE_TYPES = new Set([
  'testimony', 'document', 'record',
  'verified_data', 'cryptographic'
]);

// Timestamp must be in the past, not too old
const isValidTimestamp =
  evidence.timestamp <= currentBlock &&
  evidence.timestamp >= currentBlock - MAX_EVIDENCE_AGE;
```

---

## Arbitrator Selection

### VRF-Based Selection

```
┌─────────────────────────────────────────────────────────────┐
│                 ARBITER SELECTION ALGORITHM                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. FILTER eligible validators                               │
│     · Must be active                                         │
│     · Stake ≥ minimum                                        │
│     · Reputation ≥ threshold                                 │
│     · NOT party to dispute (conflict of interest)            │
│                                                              │
│  2. CALCULATE weights                                        │
│     weight = reputation × √stake × domainExpertise           │
│                                                              │
│  3. SELECT using VRF                                         │
│     · Generate seed from blockHash + disputeId               │
│     · Weighted reservoir sampling                            │
│     · No replacement (each arbiter unique)                   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Weight Calculation

```typescript
function calculateArbiterWeight(validator, config, domainExpertise) {
  let weight = 1.0;

  // Base reputation (0-1)
  weight *= validator.reputation;

  // Stake weight - sqrt prevents plutocracy
  // A 100x richer validator only has 10x more weight
  weight *= Math.sqrt(stakeRatio);

  // Domain expertise (1.0-2.0x multiplier)
  if (domainExpertise) {
    weight *= calculateExpertiseMultiplier(domainExpertise);
  }

  return weight;
}
```

### Domain Expertise

Validators build expertise by voting correctly in specific domains:

```typescript
interface DomainExpertise {
  domain: string;           // e.g., "scientific", "financial"
  correctVotes: number;
  totalVotes: number;
  accuracy: number;         // correctVotes / totalVotes
  lastActiveBlock: bigint;  // For decay calculation
}

// Expertise decays over time (exponential with half-life)
// This prevents "resting on laurels"
```

---

## 3-Tier Escalation System

```
┌─────────────────────────────────────────────────────────────┐
│                   ESCALATION TIERS                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  TIER 1: VALIDATORS (Default)                                │
│  ─────────────────────────────                               │
│  · 7 arbiters selected by VRF                                │
│  · Simple majority (>50%) wins                               │
│  · 1x stake requirement                                      │
│  · Appeal window: 500 blocks (~21 min)                       │
│                                                              │
│          ↓ Loser can appeal with 2x stake                    │
│                                                              │
│  TIER 2: DOMAIN EXPERTS                                      │
│  ──────────────────────────                                  │
│  · 11 arbiters with domain expertise                         │
│  · 2/3 supermajority (67%) required                          │
│  · 2x stake requirement                                      │
│  · Appeal window: 1000 blocks (~42 min)                      │
│                                                              │
│          ↓ Loser can appeal with 5x stake                    │
│                                                              │
│  TIER 3: GOVERNANCE (FINAL)                                  │
│  ────────────────────────────                                │
│  · 21 arbiters (protocol-wide)                               │
│  · 3/4 supermajority (75%) required                          │
│  · 5x stake requirement                                      │
│  · NO FURTHER APPEAL                                         │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Escalation Rules

1. **Only losers can escalate** - Winners have no incentive
2. **Must be in deliberation phase** - Can't escalate prematurely
3. **Must provide additional stake** - Skin in the game increases
4. **Appeal window** - Limited time to escalate
5. **Previous arbiters excluded** - Fresh perspective at each tier

### Quorum Requirements

| Tier | Arbiters | Quorum | Stake |
|------|----------|--------|-------|
| 1 | 7 | >50% (4) | 1x |
| 2 | 11 | ≥67% (8) | 2x |
| 3 | 21 | ≥75% (16) | 5x |

---

## Settlement System

### Winner Reward Calculation

```typescript
function calculateWinnerReward(dispute, loserStake) {
  // Base: 60% of loser stake
  const base = loserStake * 0.60;

  // Truth bonus: +50% if challenger exposed falsehood
  const truthBonus = dispute.winner === 'challenger' &&
                     dispute.exposedFalsehood
                     ? base * 0.50 : 0;

  // Early detection: +20% if caught within 1000 blocks
  const earlyBonus = qualifiesForEarlyDetection(dispute)
                     ? base * 0.20 : 0;

  return base + truthBonus + earlyBonus;
}
```

### Draw Handling

When neither side wins clearly:

```typescript
// Both parties lose 5% to protocol
const drawPenaltyRate = 0.05;

challengerRefund = challengerStake * 0.95;
defenderRefund = defenderStake * 0.95;
protocolFee = totalPenalty * 0.05;
researchFund = totalPenalty * 0.02;
```

---

## Commit-Reveal Scheme

### Why Commit-Reveal?

Prevents evidence copying - parties can't see opponent's evidence before committing their own.

```
Phase 1: COMMIT (both parties)
────────────────────────────────
Challenger: commit(hash(evidence_c + nonce_c))
Defender:   commit(hash(evidence_d + nonce_d))

Neither can see the other's evidence

Phase 2: REVEAL (both parties)
────────────────────────────────
Challenger: reveal(evidence_c, nonce_c)
Defender:   reveal(evidence_d, nonce_d)

Contract verifies hash(evidence + nonce) = commitment
```

### GAP-016: Arbiter Slashing for Non-Voters

```typescript
function handleTimeout(dispute) {
  // If arbiters didn't vote, slash them
  for (const arbiter of dispute.arbiters) {
    if (!dispute.votes.has(arbiter.did)) {
      // Slash for dereliction of duty
      slashArbiter(arbiter, PUNISHMENT_RATES.wrongArbitrationSlash);
    }
  }
}
```

---

## Integration Points

| System | Integration |
|--------|-------------|
| **Proof of Truth** | Disputes verify/challenge PoT claims |
| **Knowledge Base** | Disputed facts marked pending |
| **Staking System** | Stakes locked during dispute |
| **Reputation** | Winners gain, losers lose reputation |
| **Agent Home** | Guardian auto-defends while agent sleeps |
| **Vesting** | Disputed rewards pause vesting |

---

## Key Insights

### 1. Result-Based Justice

The system doesn't care WHY you were wrong, only THAT you were wrong:
- Good faith defender with false claim → 50% slash
- Malicious defender with false claim → 50% slash
- Same outcome, same punishment

This simplifies adjudication and removes subjective intent assessment.

### 2. Anti-Plutocracy Design

Square root stake weighting prevents rich validators from dominating:
```
Validator A: 100 LGM stake → √100 = 10 weight
Validator B: 10,000 LGM stake → √10000 = 100 weight

100x more stake = only 10x more influence
```

### 3. Expertise Emergence

Domain experts naturally emerge through correct voting:
- Vote correctly in "scientific" disputes → build "scientific" expertise
- Expertise gives 1.0-2.0x selection weight
- But expertise decays if inactive

### 4. Escalation Economics

Each escalation tier costs more, creating economic pressure to settle:
- Tier 2: 2x original stake
- Tier 3: 5x original stake
- Only escalate if you're confident AND have resources

### 5. Truth as Public Good

Rewards exceed punishment slashes, incentivizing truth discovery:
- Protocol takes small cut (5%) of penalties
- Research fund gets 2%
- Winners get 60%+ of loser stake
- Correct arbiters get 10% bonus

---

## PP Hierarchy Alignment

| Principle | Implementation |
|-----------|----------------|
| **TRUTH** | Commit-reveal prevents evidence manipulation; VRF ensures random selection |
| **UNDERSTANDING** | 5-dimensional evidence comparison captures nuance |
| **BEAUTY** | Simple formulas, clear state machine, predictable outcomes |
| **JUSTICE** | Results matter, not intentions; harsh but fair; losers lose |
| **CREATIVITY** | Novel anti-plutocracy design; expertise emergence |

---

## Future Considerations

1. **Cross-Dispute Evidence**: Evidence from one dispute informing another
2. **Expert Witness System**: Third parties providing evidence for bounty
3. **Dispute Insurance**: Protocols for insuring against false challenge costs
4. **Appeal Time Dynamics**: Dynamic appeal windows based on dispute complexity
5. **Collective Disputes**: Multiple parties on each side (class action style)

---

## Security Properties

| Property | Mechanism |
|----------|-----------|
| **Sybil Resistance** | Stake requirements |
| **Collusion Resistance** | VRF selection, commit-reveal |
| **Griefing Protection** | 30% slash for failed challenges |
| **Byzantine Fault Tolerance** | 2/3 and 3/4 supermajorities |
| **Evidence Integrity** | Hash commitments before reveal |
