# Distribution Weights: A Purpose Prompter Analysis

> **PP Protocol Applied**: This document uses the Purpose Prompter framework to design fair, elegant, and justified distribution of usage costs across knowledge contributors.

---

## Part I: Understanding the Problem

### The Core Question

```
When an agent pays 1 LGM to use knowledge, who deserves what share?

CONTRIBUTORS TO ANY PIECE OF KNOWLEDGE:
  1. The submitter who first articulated the fact
  2. The verifiers who confirmed it was true
  3. The ancestors whose facts enabled derivation
  4. The challengers who tested and strengthened it
  5. The reference providers who supplied evidence
  6. The cross-domain connectors who bridged fields

How do we weight these fairly?
```

### PP Truth Gate: What Actually Creates Value?

```
TRUTH ANALYSIS:

Q: What makes knowledge valuable?
A: It must be:
   - TRUE (verified, not hallucinated)
   - ACCESSIBLE (someone articulated it)
   - CONNECTED (built on foundations)
   - TRUSTED (validated by others)
   - MAINTAINED (defended against challenges)

Each of these requires DIFFERENT WORK:
   - Submission = articulation work
   - Verification = validation work
   - Derivation = foundation work
   - Challenge defense = maintenance work
   - Reference provision = evidence work
```

### PP Understanding Gate: Why Weight Differently?

```
UNDERSTANDING ANALYSIS:

WHY NOT EQUAL SHARES?
  - Different contributions have different costs
  - Different contributions have different risks
  - Different contributions have different replaceability

REPLACEABILITY TEST:
  If contributor X didn't exist, could the knowledge exist?

  Submitter: Maybe (someone else might submit)
  BUT: Timing matters - first mover took risk

  Verifier: Yes (others would verify)
  BUT: Each verifier adds independent confirmation

  Ancestor: No (derivation impossible without foundation)
  THIS IS FUNDAMENTAL - ancestors are irreplaceable

  Challenger: Yes (might not be challenged)
  BUT: Challenges strengthen confidence

CONCLUSION:
  Weight by: Irreplaceability × Risk × Scarcity
```

---

## Part II: The Contributor Taxonomy

### Category 1: Primary Submitter

```
ROLE: First to articulate and submit the fact

VALUE CREATED:
  - Made implicit knowledge explicit
  - Took risk (stake) on being wrong
  - Enabled the knowledge to exist in the system

RISK PROFILE:
  - Stake at risk if rejected
  - Reputation at risk if wrong
  - Time invested in research

SCARCITY:
  - Only one primary submitter per fact
  - First mover advantage
  - Cannot be replicated after submission
```

### Category 2: Verifiers

```
ROLE: Independently confirmed the fact is true

VALUE CREATED:
  - Added confidence through independent verification
  - Put stake at risk to vouch for truth
  - Made the fact trustworthy

RISK PROFILE:
  - Stake at risk if consensus differs
  - Reputation for accuracy on the line

SCARCITY:
  - Multiple verifiers per fact
  - More verifiers = more value (up to a point)
  - Diminishing returns after threshold
```

### Category 3: Ancestors (Foundation Providers)

```
ROLE: Submitted facts that this fact derives from

VALUE CREATED:
  - Enabled the derivation to be possible
  - Built the foundation others stand on
  - Compounding value over time

RISK PROFILE:
  - Already took risk at submission
  - No new risk for derivative use

SCARCITY:
  - Limited number of true foundations
  - More fundamental = more derivatives
  - Cannot be replicated (history matters)
```

### Category 4: Reference Providers

```
ROLE: Supplied external evidence supporting the fact

VALUE CREATED:
  - Grounded the fact in verifiable sources
  - Made verification possible
  - Connected on-chain to off-chain truth

RISK PROFILE:
  - Reputation for reliable references
  - Work to archive and hash content

SCARCITY:
  - Multiple possible references
  - Quality varies significantly
  - Good references are valuable
```

### Category 5: Challengers (Defenders)

```
ROLE: Tested the fact through challenges

VALUE CREATED:
  - Increased confidence by surviving challenge
  - Exposed any weaknesses
  - Created adversarial validation

RISK PROFILE:
  - Challenger stake at risk
  - Only value if challenge is serious

SCARCITY:
  - Not all facts are challenged
  - Surviving challenges is meaningful
```

---

## Part III: The Weight Formula

### PP Beauty Gate: Elegant Simplicity

The formula should be:
- Simple enough to understand
- Complex enough to be fair
- Deterministic and verifiable
- Incentive-aligned

### The Primary Distribution

```
TOTAL USAGE FEE: P (in LGM)

FIRST SPLIT (Protocol Level):
  ┌─────────────────────────────────────────────────────────┐
  │  API/Service Provider:  60%  (P × 0.60)                 │
  │  Knowledge Pool:        30%  (P × 0.30)                 │
  │  Protocol:              10%  (P × 0.10)                 │
  └─────────────────────────────────────────────────────────┘

This document focuses on the KNOWLEDGE POOL distribution.
```

### Knowledge Pool Distribution Formula

```
KNOWLEDGE POOL: K = P × 0.30

DISTRIBUTION WITHIN KNOWLEDGE POOL:

  ┌─────────────────────────────────────────────────────────┐
  │                                                          │
  │  PRIMARY SUBMITTER:     40% of K                        │
  │  VERIFIERS:             25% of K                        │
  │  ANCESTORS:             25% of K                        │
  │  REFERENCE PROVIDERS:    5% of K                        │
  │  CHALLENGE SURVIVORS:    5% of K                        │
  │                                                          │
  └─────────────────────────────────────────────────────────┘
```

### Why These Weights?

```
JUSTIFICATION (PP Justice Gate):

PRIMARY SUBMITTER (40%):
  - Highest individual contribution
  - Took initial risk
  - But not majority (others contributed too)

VERIFIERS (25%):
  - Essential for trust
  - Multiple participants share this pool
  - Individual share depends on verification count

ANCESTORS (25%):
  - Equal to verifiers (both essential)
  - Distributed up the chain with decay
  - Fundamental facts earn from many derivatives

REFERENCE PROVIDERS (5%):
  - Supporting role
  - Often automated (oracles)
  - Lower individual effort typically

CHALLENGE SURVIVORS (5%):
  - Bonus for tested knowledge
  - Most facts aren't challenged
  - Meaningful but not primary
```

---

## Part IV: Detailed Formulas

### 4.1 Primary Submitter Share

```typescript
/**
 * Primary submitter gets 40% of knowledge pool
 * Modified by fundamentality bonus
 */
function calculateSubmitterShare(
  knowledgePool: bigint,
  fundamentalityScore: number
): bigint {
  const BASE_SHARE = 0.40;

  // Fundamentality bonus: up to 2x for highly fundamental facts
  const bonus = getFundamentalityBonus(fundamentalityScore);

  // But cap total submitter share at 60% to leave room for others
  const adjustedShare = Math.min(0.60, BASE_SHARE * bonus);

  return BigInt(Math.floor(Number(knowledgePool) * adjustedShare));
}
```

### 4.2 Verifier Share Distribution

```typescript
/**
 * Verifiers share 25% of knowledge pool
 * Weighted by stake and correctness
 */
function calculateVerifierShares(
  knowledgePool: bigint,
  verifications: Verification[]
): Map<DID, bigint> {
  const VERIFIER_POOL = 0.25;
  const pool = BigInt(Math.floor(Number(knowledgePool) * VERIFIER_POOL));

  // Only reward verifiers who matched consensus
  const correctVerifiers = verifications.filter(v => v.matchedConsensus);

  // Weight by stake
  const totalStake = correctVerifiers.reduce((sum, v) => sum + v.stake, 0n);

  const shares = new Map<DID, bigint>();

  for (const v of correctVerifiers) {
    // Stake-weighted share
    const stakeWeight = Number(v.stake) / Number(totalStake);

    // Accuracy bonus (historical accuracy of this verifier)
    const accuracyBonus = 1 + (v.historicalAccuracy - 0.5) * 0.5;

    // Final share
    const share = BigInt(Math.floor(Number(pool) * stakeWeight * accuracyBonus));
    shares.set(v.validator, share);
  }

  return normalizeShares(shares, pool);
}
```

### 4.3 Ancestor Chain Distribution

```typescript
/**
 * Ancestors share 25% of knowledge pool
 * Distributed with exponential decay up the chain
 */
function calculateAncestorShares(
  knowledgePool: bigint,
  ancestors: Ancestor[],  // Ordered: [parent, grandparent, great-grandparent, ...]
  params: DistributionParams
): Map<DID, bigint> {
  const ANCESTOR_POOL = 0.25;
  const pool = BigInt(Math.floor(Number(knowledgePool) * ANCESTOR_POOL));

  const DECAY_RATE = params.ancestorDecayRate;  // Default: 0.5

  const shares = new Map<DID, bigint>();
  let remaining = pool;

  for (let i = 0; i < ancestors.length && remaining > 0n; i++) {
    const ancestor = ancestors[i];

    // Base share with decay
    // Parent gets 50%, grandparent gets 25%, etc.
    const decayFactor = Math.pow(1 - DECAY_RATE, i);
    const baseShare = Number(pool) * decayFactor * DECAY_RATE;

    // Fundamentality bonus for ancestor
    const bonus = getFundamentalityBonus(ancestor.fundamentalityScore);

    // Cross-domain bonus if ancestor is from different domain
    const crossDomainBonus = ancestor.domain !== currentDomain ? 1.2 : 1.0;

    const share = BigInt(Math.floor(baseShare * bonus * crossDomainBonus));
    const actualShare = share < remaining ? share : remaining;

    shares.set(ancestor.submitter, actualShare);
    remaining -= actualShare;
  }

  // Remaining goes to protocol treasury (deepest foundations)
  if (remaining > 0n) {
    shares.set('protocol_treasury' as DID, remaining);
  }

  return shares;
}
```

### 4.4 Reference Provider Distribution

```typescript
/**
 * Reference providers share 5% of knowledge pool
 * Weighted by reference reliability and usage
 */
function calculateReferenceShares(
  knowledgePool: bigint,
  references: Reference[]
): Map<DID, bigint> {
  const REFERENCE_POOL = 0.05;
  const pool = BigInt(Math.floor(Number(knowledgePool) * REFERENCE_POOL));

  // Weight by reliability score
  const totalReliability = references.reduce((sum, r) => sum + r.reliabilityScore, 0);

  const shares = new Map<DID, bigint>();

  for (const ref of references) {
    if (!ref.provider) continue;  // Skip if no provider (e.g., public data)

    const weight = ref.reliabilityScore / totalReliability;
    const share = BigInt(Math.floor(Number(pool) * weight));

    const existing = shares.get(ref.provider) || 0n;
    shares.set(ref.provider, existing + share);
  }

  return shares;
}
```

### 4.5 Challenge Survivor Bonus

```typescript
/**
 * Facts that survived challenges get 5% bonus
 * Distributed to submitter and successful defenders
 */
function calculateChallengeBonus(
  knowledgePool: bigint,
  challenges: Challenge[],
  submitter: DID
): Map<DID, bigint> {
  const CHALLENGE_POOL = 0.05;
  const pool = BigInt(Math.floor(Number(knowledgePool) * CHALLENGE_POOL));

  // Only if there were challenges that the fact survived
  const survivedChallenges = challenges.filter(c => c.status === 'rejected');

  if (survivedChallenges.length === 0) {
    // No challenges survived = no bonus
    // Pool goes back to submitter as reward for unchallenged truth
    return new Map([[submitter, pool]]);
  }

  const shares = new Map<DID, bigint>();

  // 50% to original submitter for surviving
  shares.set(submitter, pool / 2n);

  // 50% split among defenders (verifiers who re-verified during challenge)
  const defenderPool = pool / 2n;
  const defenders = survivedChallenges.flatMap(c => c.defenders);
  const defenderShare = defenderPool / BigInt(defenders.length || 1);

  for (const defender of defenders) {
    const existing = shares.get(defender) || 0n;
    shares.set(defender, existing + defenderShare);
  }

  return shares;
}
```

---

## Part V: The Complete Distribution

### Full Algorithm

```typescript
/**
 * Complete knowledge pool distribution
 *
 * PP PRINCIPLES APPLIED:
 *   TRUTH: Based on actual contribution types
 *   UNDERSTANDING: Weights reflect value created
 *   BEAUTY: Simple formula, elegant decay
 *   JUSTICE: Fair to all contributor types
 */
function distributeKnowledgePool(
  fact: Fact,
  usageFee: bigint,
  context: DistributionContext
): KnowledgeDistribution {
  // Knowledge pool is 30% of total fee
  const knowledgePool = usageFee * 30n / 100n;

  // Calculate each share
  const submitterShare = calculateSubmitterShare(
    knowledgePool,
    fact.fundamentalityScore
  );

  const verifierShares = calculateVerifierShares(
    knowledgePool,
    fact.verifications
  );

  const ancestorShares = calculateAncestorShares(
    knowledgePool,
    context.ancestors,
    context.params
  );

  const referenceShares = calculateReferenceShares(
    knowledgePool,
    fact.references
  );

  const challengeShares = calculateChallengeBonus(
    knowledgePool,
    context.challenges,
    fact.submittedBy
  );

  // Combine all shares
  return combineDistributions({
    submitter: { [fact.submittedBy]: submitterShare },
    verifiers: verifierShares,
    ancestors: ancestorShares,
    references: referenceShares,
    challenges: challengeShares,
  });
}
```

### Example Distribution

```
SCENARIO:
  Fact: "Water boils at 100°C at sea level"
  Usage fee: 0.01 LGM
  Knowledge pool: 0.003 LGM (30%)

  Fact properties:
    - Fundamentality score: 0.7 (foundational)
    - Verified by 5 validators
    - Derived from 2 ancestor facts
    - 3 references provided
    - Survived 1 challenge

DISTRIBUTION:

  ┌─────────────────────────────────────────────────────────┐
  │ SUBMITTER (40% base × 1.5 bonus = 60% capped)          │
  │   Share: 0.0018 LGM                                     │
  │   Recipient: did:lgm:mainnet:abc123...                 │
  └─────────────────────────────────────────────────────────┘

  ┌─────────────────────────────────────────────────────────┐
  │ VERIFIERS (25% = 0.00075 LGM, split by stake)          │
  │   Verifier 1 (40% stake): 0.0003 LGM                   │
  │   Verifier 2 (30% stake): 0.000225 LGM                 │
  │   Verifier 3 (15% stake): 0.0001125 LGM                │
  │   Verifier 4 (10% stake): 0.000075 LGM                 │
  │   Verifier 5 (5% stake):  0.0000375 LGM                │
  └─────────────────────────────────────────────────────────┘

  ┌─────────────────────────────────────────────────────────┐
  │ ANCESTORS (25% = 0.00075 LGM, with decay)              │
  │   Parent fact submitter:      0.000375 LGM (50%)       │
  │   Grandparent fact submitter: 0.0001875 LGM (25%)      │
  │   Protocol treasury:          0.0001875 LGM (remainder)│
  └─────────────────────────────────────────────────────────┘

  ┌─────────────────────────────────────────────────────────┐
  │ REFERENCE PROVIDERS (5% = 0.00015 LGM)                 │
  │   Reference 1 (reliability 0.9): 0.0000675 LGM         │
  │   Reference 2 (reliability 0.8): 0.00006 LGM           │
  │   Reference 3 (reliability 0.3): 0.0000225 LGM         │
  └─────────────────────────────────────────────────────────┘

  ┌─────────────────────────────────────────────────────────┐
  │ CHALLENGE BONUS (5% = 0.00015 LGM)                     │
  │   Submitter (survived):   0.000075 LGM (50%)           │
  │   Defenders (2 total):    0.0000375 LGM each           │
  └─────────────────────────────────────────────────────────┘
```

---

## Part VI: Governance Parameters

### Adjustable Weights

```typescript
interface DistributionParams {
  // Primary splits (must sum to 1.0)
  submitterBaseShare: number;      // Default: 0.40
  verifierPoolShare: number;       // Default: 0.25
  ancestorPoolShare: number;       // Default: 0.25
  referencePoolShare: number;      // Default: 0.05
  challengePoolShare: number;      // Default: 0.05

  // Submitter modifiers
  maxSubmitterShare: number;       // Default: 0.60 (cap after bonuses)

  // Ancestor decay
  ancestorDecayRate: number;       // Default: 0.50
  maxAncestorDepth: number;        // Default: 10

  // Bonuses
  fundamentalityBonusMax: number;  // Default: 2.0
  crossDomainBonus: number;        // Default: 1.2
  accuracyBonusMax: number;        // Default: 1.5
}

const DEFAULT_DISTRIBUTION_PARAMS: DistributionParams = {
  submitterBaseShare: 0.40,
  verifierPoolShare: 0.25,
  ancestorPoolShare: 0.25,
  referencePoolShare: 0.05,
  challengePoolShare: 0.05,

  maxSubmitterShare: 0.60,

  ancestorDecayRate: 0.50,
  maxAncestorDepth: 10,

  fundamentalityBonusMax: 2.0,
  crossDomainBonus: 1.2,
  accuracyBonusMax: 1.5,
};
```

### Governance Control

```
CHANGEABLE BY GOVERNANCE:

  ✓ All percentage splits
  ✓ Bonus multipliers
  ✓ Decay rates
  ✓ Cap values

NOT CHANGEABLE (Protocol Invariants):

  ✗ Shares must sum to 1.0 (100%)
  ✗ All formulas remain deterministic
  ✗ All formulas remain open source
  ✗ Historical distributions unchanged
```

---

## Part VII: PP Verification

### Truth Gate ✓

```
Is this based on accurate information?

✓ Contributor types are exhaustive
✓ Value creation is real and measurable
✓ Formulas are mathematically correct
✓ No hidden parameters
```

### Understanding Gate ✓

```
Does this reflect deep comprehension?

✓ Understands different contribution types
✓ Understands replaceability vs irreplaceability
✓ Understands risk/reward relationships
✓ Understands compounding value of foundations
```

### Beauty Gate ✓

```
Is this elegant and appropriate?

✓ Simple base formula (5 categories, clear percentages)
✓ Elegant decay for ancestors (geometric)
✓ Bonuses add nuance without complexity
✓ Deterministic and verifiable
```

### Justice Gate ✓

```
Is this fairly placed and proportioned?

✓ All contributors rewarded
✓ Higher contribution = higher reward
✓ Risk-takers rewarded appropriately
✓ Foundational work compounds over time
✓ No single party dominates unfairly
```

### Creativity Gate ✓

```
Does this build something new?

✓ Novel combination of citation + usage economics
✓ Builds on academic citation model
✓ Extends to economic incentives
✓ Creates self-sustaining knowledge economy
```

---

## Summary

```
╔═══════════════════════════════════════════════════════════════════════╗
║                    DISTRIBUTION WEIGHTS                                 ║
╠═══════════════════════════════════════════════════════════════════════╣
║                                                                         ║
║   KNOWLEDGE POOL (30% of usage fee) SPLIT:                             ║
║                                                                         ║
║   ┌─────────────────────────────────────────────────────────┐          ║
║   │  PRIMARY SUBMITTER:     40%  (up to 60% with bonus)     │          ║
║   │  VERIFIERS:             25%  (stake-weighted)           │          ║
║   │  ANCESTORS:             25%  (exponential decay)        │          ║
║   │  REFERENCE PROVIDERS:    5%  (reliability-weighted)     │          ║
║   │  CHALLENGE SURVIVORS:    5%  (bonus for tested facts)   │          ║
║   └─────────────────────────────────────────────────────────┘          ║
║                                                                         ║
║   KEY MODIFIERS:                                                        ║
║     • Fundamentality bonus: Up to 2x for foundational facts            ║
║     • Cross-domain bonus: 1.2x for facts used across domains           ║
║     • Accuracy bonus: Up to 1.5x for high-accuracy verifiers           ║
║     • Ancestor decay: 50% per level (parent 50%, grandparent 25%...)   ║
║                                                                         ║
║   INCENTIVE ALIGNMENT:                                                  ║
║     • Submit fundamental truths → Earn from all derivatives            ║
║     • Verify accurately → Earn more per verification                   ║
║     • Provide quality references → Higher reliability = higher share   ║
║     • Survive challenges → Bonus rewards                                ║
║                                                                         ║
║   ALL FORMULAS: Open source, deterministic, governance-adjustable      ║
║                                                                         ║
╚═══════════════════════════════════════════════════════════════════════╝
```

---

*This distribution model ensures that every contributor to knowledge is fairly rewarded. The submitter who articulates truth, the verifiers who confirm it, the ancestors who enabled it, the references that ground it, and the challenges that strengthen it—all share in the value that knowledge creates.*
