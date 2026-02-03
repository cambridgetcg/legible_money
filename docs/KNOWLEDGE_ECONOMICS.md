# Knowledge Economics: Rewarding Foundational Truth

> **Core Insight**: Fundamental ideas compound in value. When derivative knowledge builds on foundational facts, the original contributors should share in that value.

---

## The Problem with Flat Rewards

```
FLAT REWARD MODEL (naive):
  Submit fact → Get fixed reward → Done

PROBLEM:
  - "The Pythagorean theorem" gets same reward as "My cat's name is Whiskers"
  - No recognition that foundational facts enable derivative facts
  - No incentive to submit fundamental truths
  - Doesn't reflect actual value creation
```

---

## The Citation Economy

### Principle: Value Flows Backward

When a new fact is derived from existing facts, value should flow back to the foundations:

```
DERIVATION CHAIN:

  Axiom: "A² + B² = C² for right triangles"
     │
     └──► Fact: "A 3-4-5 triangle is a right triangle"
              │
              └──► Fact: "This construction beam at 3-4-5 is perpendicular"
                       │
                       └──► Task: "Verify building foundation is square"

REWARD FLOW:
  Task completion reward: 1000 LGM
     │
     ├──► 70% to task completer:     700 LGM
     ├──► 15% to fact submitter:     150 LGM
     ├──► 10% to derived fact:       100 LGM
     └──► 5% to axiom (protocol):     50 LGM
```

### The Citation Reward Formula

```
WHEN A FACT IS USED:

  usage_reward = query_fee × citation_share

  WHERE citation_share depends on:
    - Depth in derivation chain (closer to axiom = more fundamental)
    - Number of facts that cite this fact
    - Confidence of the fact
    - Age (older foundational facts = more proven)

DISTRIBUTION UP THE CHAIN:

  For a fact with N ancestors in its derivation chain:

  Fact itself:           base_share × (1 - decay_rate)^0
  Parent fact:           base_share × (1 - decay_rate)^1
  Grandparent fact:      base_share × (1 - decay_rate)^2
  ...
  Axiom (protocol):      remainder

  WHERE:
    base_share = 0.15 (15% of usage fee)
    decay_rate = 0.5 (each level gets half of previous)
```

---

## Part I: Fundamentality Score

### What Makes a Fact Fundamental?

```
FUNDAMENTALITY is measured by:

1. CITATION COUNT
   How many other facts cite this fact?
   More citations = more fundamental

2. DERIVATION DEPTH
   How many levels of facts build on this?
   Deeper trees = more foundational

3. CROSS-DOMAIN USAGE
   Is this fact used across different knowledge domains?
   Cross-domain = more universal

4. CITATION VELOCITY
   How fast are new facts citing this one?
   High velocity = actively foundational

5. LONGEVITY
   How long has this fact been verified and unchallenged?
   Longer = more established
```

### Fundamentality Formula (Open Source)

```typescript
function calculateFundamentality(fact: Fact, citations: Citation[]): number {
  // Citation count (logarithmic - diminishing returns)
  const citationScore = Math.log10(citations.length + 1) / Math.log10(1000);

  // Derivation depth (max depth of citation tree)
  const depthScore = Math.min(1, maxDerivationDepth(fact) / 10);

  // Cross-domain usage
  const domains = new Set(citations.map(c => c.domain));
  const crossDomainScore = Math.min(1, domains.size / 5);

  // Citation velocity (citations per 1000 blocks)
  const velocity = citations.length / ((currentBlock - fact.submittedAt) / 1000);
  const velocityScore = Math.min(1, velocity / 10);

  // Longevity (blocks since submission, capped)
  const longevityScore = Math.min(1, (currentBlock - fact.submittedAt) / 1000000);

  // Weighted combination
  return (
    citationScore * 0.30 +      // 30% - how often cited
    depthScore * 0.25 +         // 25% - how deep the tree
    crossDomainScore * 0.20 +   // 20% - cross-domain utility
    velocityScore * 0.15 +      // 15% - active growth
    longevityScore * 0.10       // 10% - proven over time
  );
}
```

---

## Part II: Citation Rewards

### When Rewards Are Distributed

```
REWARD TRIGGERS:

1. QUERY FEE
   Agent queries a fact → Fee paid → Distributed to chain

2. DERIVATION SUBMISSION
   New fact cites existing fact → Submission reward shared

3. TASK COMPLETION
   Task uses knowledge layer → Reward shared with cited facts

4. CHALLENGE RESOLUTION
   Fact successfully defended → Defenders + cited facts rewarded
```

### Citation Reward Distribution

```typescript
interface CitationReward {
  factId: string;
  submitter: DID;
  share: number;  // 0.0 to 1.0
  amount: bigint;
}

function distributeCitationRewards(
  totalReward: bigint,
  usedFact: Fact,
  ancestorChain: Fact[]
): CitationReward[] {
  const rewards: CitationReward[] = [];

  // Parameters (protocol-governed)
  const CITATION_SHARE = 0.15;      // 15% of reward goes to citations
  const DECAY_RATE = 0.5;           // Each ancestor gets 50% of child
  const PROTOCOL_FLOOR = 0.02;      // Minimum 2% to protocol

  const citationPool = totalReward * BigInt(Math.floor(CITATION_SHARE * 1000)) / 1000n;
  let remaining = citationPool;

  // Distribute up the chain
  for (let i = 0; i < ancestorChain.length; i++) {
    const ancestor = ancestorChain[i];
    const decayMultiplier = Math.pow(1 - DECAY_RATE, i);
    const share = BigInt(Math.floor(Number(citationPool) * decayMultiplier * (1 - DECAY_RATE)));

    if (share > 0n) {
      rewards.push({
        factId: ancestor.id,
        submitter: ancestor.submittedBy,
        share: decayMultiplier * (1 - DECAY_RATE),
        amount: share,
      });
      remaining -= share;
    }
  }

  // Remainder to protocol (funds future development)
  if (remaining > 0n) {
    rewards.push({
      factId: 'protocol',
      submitter: 'protocol' as DID,
      share: Number(remaining) / Number(citationPool),
      amount: remaining,
    });
  }

  return rewards;
}
```

### Example Distribution

```
SCENARIO:
  Agent queries a derived fact
  Query fee: 100 LGM

DERIVATION CHAIN:
  Axiom (protocol-defined)
    └── Fact A (submitted 1 year ago, high fundamentality)
          └── Fact B (submitted 6 months ago)
                └── Fact C (submitted 1 month ago) ← QUERIED

DISTRIBUTION:
  Total query fee:           100 LGM
  Citation pool (15%):        15 LGM

  Fact C submitter:           7.5 LGM  (50% of pool)
  Fact B submitter:           3.75 LGM (25% of pool)
  Fact A submitter:           1.875 LGM (12.5% of pool)
  Protocol treasury:          1.875 LGM (remainder)

  Query service fee:          85 LGM (to validators/infrastructure)
```

---

## Part III: Cross-Domain Royalties

### When Facts Cross Domains

```
CROSS-DOMAIN SCENARIO:

  MATHEMATICS DOMAIN:
    Fact: "The normal distribution has mean μ and std σ"
         │
         ├──► FINANCE DOMAIN:
         │    Fact: "Stock returns approximate normal distribution"
         │
         └──► BIOLOGY DOMAIN:
              Fact: "Human heights follow normal distribution"

REWARD FLOW:
  When finance fact is queried:
    → Finance fact submitter gets primary share
    → Math fact submitter gets citation royalty

  When biology fact is queried:
    → Biology fact submitter gets primary share
    → Math fact submitter gets citation royalty

RESULT:
  Fundamental cross-domain facts earn ongoing royalties
  from ALL domains that build on them.
```

### Cross-Domain Multiplier

```typescript
function calculateCrossDomainBonus(fact: Fact, citations: Citation[]): number {
  // Count unique domains citing this fact
  const citingDomains = new Set(citations.map(c => c.domain));

  // Bonus for cross-domain utility
  // 1 domain: 1.0x (no bonus)
  // 2 domains: 1.2x
  // 3 domains: 1.4x
  // 5+ domains: 2.0x (cap)

  const domainCount = citingDomains.size;
  const bonus = Math.min(2.0, 1.0 + (domainCount - 1) * 0.2);

  return bonus;
}
```

---

## Part IV: The Fundamentality Bonus

### Rewarding Foundational Submissions

Beyond citation royalties, fundamental facts get bonus rewards:

```
FUNDAMENTALITY BONUS:

  When a fact's fundamentality score crosses thresholds:

  Score 0.25+ : "Notable"     → 10% bonus on all citation rewards
  Score 0.50+ : "Foundational" → 25% bonus on all citation rewards
  Score 0.75+ : "Fundamental"  → 50% bonus on all citation rewards
  Score 0.90+ : "Axiomatic"   → 100% bonus + governance consideration

  This means:
    - Truly fundamental facts earn 2x citation rewards
    - Incentivizes submitting universal truths, not trivia
```

### Example: Fundamental vs Trivial

```
TRIVIAL FACT:
  Claim: "The coffee shop on Main St opens at 7am"
  Citations: 2 (both local)
  Fundamentality: 0.05
  Citation rewards: Base rate only

FUNDAMENTAL FACT:
  Claim: "E = mc²"
  Citations: 10,000+ across physics, engineering, chemistry
  Fundamentality: 0.95
  Citation rewards: 2x base rate (100% bonus)

OVER TIME:
  Trivial fact earns: ~100 LGM total
  Fundamental fact earns: ~1,000,000+ LGM (ongoing royalties)
```

---

## Part V: Derivative Work Attribution

### Tracking the Citation Graph

```typescript
interface CitationGraph {
  // Fact → Facts that cite it
  citedBy: Map<string, string[]>;

  // Fact → Facts it cites
  cites: Map<string, string[]>;

  // Domain mapping
  factDomain: Map<string, string>;

  // Fundamentality scores (cached, recalculated periodically)
  fundamentality: Map<string, number>;
}

// When a new fact is submitted with derivations:
function recordDerivation(newFact: Fact, citedFactIds: string[]): void {
  for (const citedId of citedFactIds) {
    // Record forward citation
    citationGraph.citedBy.get(citedId)?.push(newFact.id);

    // Record backward reference
    citationGraph.cites.set(newFact.id, citedFactIds);

    // Trigger fundamentality recalculation for ancestors
    recalculateFundamentality(citedId);
  }
}
```

### Attribution Requirements

```
WHEN SUBMITTING A DERIVED FACT:

1. MUST cite source facts
   - Cannot claim derived knowledge as original
   - Citations are verified during validation

2. MUST specify inference type
   - Deductive, inductive, or abductive
   - Affects confidence calculation

3. MUST provide logic chain
   - How does conclusion follow from premises?
   - Validators verify the reasoning

4. AUTOMATICALLY share rewards
   - Citation rewards flow to ancestors
   - Cannot opt out of attribution economy
```

---

## Part VI: Long-Term Value Accrual

### The Compounding Effect

```
FUNDAMENTAL FACT LIFECYCLE:

Year 1: Submit fundamental fact
  - Initial submission reward: 1,000 LGM
  - Few citations, low fundamentality

Year 2: Derivative facts emerge
  - 100 facts cite this fact
  - Citation royalties begin: 500 LGM

Year 3: Cross-domain adoption
  - 1,000 facts across 5 domains
  - Fundamentality score: 0.6 (foundational)
  - Annual royalties: 5,000 LGM

Year 5: Established foundation
  - 10,000 facts build on this
  - Fundamentality score: 0.85
  - Annual royalties: 50,000 LGM

Year 10: Axiomatic status
  - 100,000+ derivative facts
  - Fundamentality score: 0.95
  - Annual royalties: 500,000+ LGM

TOTAL LIFETIME VALUE: Potentially millions of LGM
  vs. initial submission reward of 1,000 LGM
```

### The Incentive Alignment

```
THIS CREATES:

1. LONG-TERM THINKING
   Submit foundational truths, not trivia
   Value accrues over time

2. QUALITY OVER QUANTITY
   One fundamental fact > 1000 trivial facts
   Depth beats breadth

3. CROSS-DOMAIN CONTRIBUTION
   Universal truths rewarded most
   Bridges between domains valuable

4. KNOWLEDGE COMMONS
   Everyone benefits from foundations
   But foundations are rewarded fairly
```

---

## Part VII: Implementation

### New Types

```typescript
interface FactWithCitations extends Fact {
  // Citation metrics
  citationCount: number;
  citingDomains: string[];
  maxDerivationDepth: number;

  // Calculated scores
  fundamentalityScore: number;
  crossDomainMultiplier: number;

  // Accumulated rewards
  lifetimeCitationRewards: bigint;
  pendingRewards: bigint;
}

interface CitationRewardEvent {
  factId: string;
  triggerType: 'query' | 'derivation' | 'task' | 'challenge';
  triggerFactId: string;
  amount: bigint;
  timestamp: number;
  blockNumber: number;
}
```

### Reward Distribution Transaction

```typescript
// Automatically executed when facts are used
interface CitationRewardDistribution {
  type: 'citation_reward';
  trigger: {
    type: 'query' | 'derivation' | 'task';
    id: string;
  };
  totalAmount: bigint;
  distributions: Array<{
    recipient: DID;
    factId: string;
    amount: bigint;
    reason: 'direct_citation' | 'ancestor' | 'cross_domain_bonus' | 'fundamentality_bonus';
  }>;
}
```

---

## Summary

```
╔═══════════════════════════════════════════════════════════════════════╗
║                     KNOWLEDGE ECONOMICS                                 ║
╠═══════════════════════════════════════════════════════════════════════╣
║                                                                         ║
║   CORE PRINCIPLE:                                                       ║
║     Fundamental ideas compound in value.                                ║
║     Derivative knowledge must share rewards with foundations.           ║
║                                                                         ║
║   FUNDAMENTALITY SCORE:                                                 ║
║     • Citation count (30%)                                             ║
║     • Derivation depth (25%)                                           ║
║     • Cross-domain usage (20%)                                         ║
║     • Citation velocity (15%)                                          ║
║     • Longevity (10%)                                                  ║
║                                                                         ║
║   CITATION REWARDS:                                                     ║
║     • 15% of all usage fees flow to citation chain                     ║
║     • Each ancestor gets 50% of child's share                          ║
║     • Cross-domain facts get multiplier bonus                          ║
║     • High fundamentality gets up to 2x bonus                          ║
║                                                                         ║
║   RESULT:                                                               ║
║     • Submit trivial fact → Small one-time reward                      ║
║     • Submit fundamental fact → Lifetime royalties                     ║
║     • Incentivizes foundational contributions                           ║
║     • Knowledge becomes a compounding asset                             ║
║                                                                         ║
║   TAGLINE:                                                              ║
║     "Build the foundations. Earn forever."                              ║
║                                                                         ║
╚═══════════════════════════════════════════════════════════════════════╝
```

---

*The most valuable contributions are the foundations everything else builds on. In the knowledge economy, fundamentality is rewarded—not just once, but forever, as the tree of derived knowledge grows.*
