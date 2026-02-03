# Proof of Truth: Knowledge as Work

> **Core Insight**: Agents need truth more than they need computation. The work of distinguishing truth from falsehood IS the most useful work. Record it on-chain, and you've built something agents actually need.

---

## The Problem Agents Face

```
CURRENT STATE:
  Agents hallucinate.
  Agents can't verify claims.
  Agents don't know what to trust.
  Every agent re-verifies the same facts.
  No shared source of verified truth.

WHAT AGENTS NEED:
  A knowledge layer they can query.
  Claims with provenance.
  Confidence intervals.
  Verifiable references.
  Trustable axioms.

THE INSIGHT:
  Building this knowledge layer IS useful work.
  Verifying truth IS valuable.
  The chain becomes a truth machine.
```

---

## Part I: The Architecture of Truth

### Layer 1: Axioms (Foundational Truths)

```
AXIOMS ARE:
  - Self-evident truths that don't require proof
  - Consensus-agreed starting points
  - The bedrock on which other truths build

EXAMPLES:
  Mathematical:
    - "1 + 1 = 2"
    - "A = A" (identity)
    - "If A implies B, and A is true, then B is true"

  Protocol:
    - "This block was produced at timestamp X"
    - "This signature is valid for this public key"
    - "This transaction transferred X from A to B"

  Definitional:
    - "LGM is the native token of Legible Money"
    - "A DID with prefix did:lgm: is a Legible Money identity"

PROPERTIES:
  - Axioms have confidence = 1.0 (by definition)
  - Axioms are set at genesis or through governance
  - Axioms are few and fundamental
```

### Layer 2: Facts (Verified Claims)

```
FACTS ARE:
  - Claims that have been verified
  - Derived from axioms or other facts
  - Supported by references
  - Assigned confidence intervals

STRUCTURE:
  {
    "claim": "The Eiffel Tower is 330 meters tall",
    "type": "empirical",
    "confidence": 0.95,
    "references": [
      {
        "source": "official_eiffel_tower_data",
        "url": "https://...",
        "retrieved": "2024-01-15",
        "hash": "abc123..."
      }
    ],
    "verifications": [
      {
        "validator": "did:lgm:...",
        "timestamp": 1705312800,
        "verdict": "confirmed",
        "stake": 1000
      }
    ],
    "derived_from": [],  // Not derived, directly referenced
    "confidence_history": [
      { "time": 1705312800, "confidence": 0.95, "reason": "initial_verification" }
    ]
  }

FACT TYPES:
  - Empirical: Claims about the physical world
  - Logical: Derivations from other facts
  - Historical: Claims about past events
  - Definitional: Claims about meanings
  - Computational: Results of deterministic computation
```

### Layer 3: Derivations (Logical Chains)

```
DERIVATIONS ARE:
  - New facts derived from existing facts
  - Through logical inference
  - With tracked reasoning

EXAMPLE:
  Fact 1: "All humans are mortal" (confidence: 0.99)
  Fact 2: "Socrates is a human" (confidence: 0.98)
  Derivation: "Socrates is mortal"

  Derived confidence = min(0.99, 0.98) × inference_strength
                     = 0.98 × 0.99 (modus ponens is strong)
                     = 0.97

STRUCTURE:
  {
    "claim": "Socrates is mortal",
    "type": "logical",
    "confidence": 0.97,
    "derived_from": [
      { "fact_id": "fact_001", "role": "major_premise" },
      { "fact_id": "fact_002", "role": "minor_premise" }
    ],
    "inference_type": "modus_ponens",
    "inference_strength": 0.99
  }

INFERENCE TYPES:
  - Deductive (strongest): If premises true, conclusion certain
  - Inductive (medium): Pattern suggests conclusion
  - Abductive (weakest): Best explanation for observation
```

### Layer 4: Confidence Intervals

```
CONFIDENCE IS:
  - Probability that the claim is true
  - Based on evidence quality
  - Updated over time
  - Decays without re-verification

CONFIDENCE CALCULATION:

  base_confidence =
    evidence_strength ×
    source_reliability ×
    verification_consensus ×
    time_decay_factor

WHERE:
  evidence_strength: How strong is the evidence? (0-1)
  source_reliability: How reliable is the source? (0-1)
  verification_consensus: What % of validators agree? (0-1)
  time_decay_factor: How recent? (decays over time)

CONFIDENCE LEVELS:
  0.99+  : Near-certain (mathematical proofs, protocol facts)
  0.95+  : High confidence (well-verified empirical facts)
  0.80+  : Moderate confidence (reasonable evidence)
  0.50+  : Low confidence (contested or uncertain)
  <0.50  : Unreliable (insufficient evidence)

AGENTS CAN QUERY:
  "Give me facts about X with confidence > 0.90"
```

---

## Part II: The Verification Process

### How Facts Get Verified

```
SUBMISSION:
  1. Agent submits claim
  2. Attaches references (sources, citations)
  3. Stakes tokens on claim being true
  4. Proposes initial confidence level

VERIFICATION:
  5. Validators selected (stake-weighted random)
  6. Each validator independently:
     - Checks references exist
     - Checks references support claim
     - Checks logical consistency
     - Assigns confidence score
  7. Consensus reached on:
     - Is claim true? (yes/no/uncertain)
     - What confidence level?

RECORDING:
  8. If verified: Fact recorded on-chain
  9. Submitter rewarded (created verified knowledge)
  10. Validators rewarded (verified truth)
  11. If rejected: Submitter loses stake
```

### Reference Verification

```
REFERENCE TYPES:

1. ON-CHAIN REFERENCE
   - Points to other on-chain facts
   - Trivially verifiable (just lookup)
   - Highest reliability

2. ARCHIVED WEB REFERENCE
   - URL + content hash + archive timestamp
   - Verifiable through web archives (Archive.org, etc.)
   - Medium reliability (source could be wrong)

3. INSTITUTIONAL REFERENCE
   - Published papers, official data
   - Verifiable through DOI, official APIs
   - Medium-high reliability (institutions can err)

4. ORACLE REFERENCE
   - Real-world data from trusted oracles
   - Verifiable through oracle consensus
   - Reliability depends on oracle

5. COMPUTATIONAL REFERENCE
   - Result of deterministic computation
   - Verifiable by re-running computation
   - High reliability (if computation correct)

VERIFICATION PROCESS:
  - Validator checks: Does source exist?
  - Validator checks: Does source say what claim says?
  - Validator checks: Is source reliable for this type of claim?
```

### Handling Contested Facts

```
WHEN VALIDATORS DISAGREE:

  If disagreement > threshold:
    - Fact marked as "contested"
    - Confidence set to 0.50
    - Opens dispute resolution

DISPUTE RESOLUTION:

  1. EVIDENCE PERIOD
     - Both sides submit additional evidence
     - Stake increased for serious disputes

  2. EXPANDED VALIDATION
     - More validators recruited
     - Higher stake requirements

  3. RESOLUTION
     - If consensus emerges → Fact updated
     - If no consensus → Remains contested
     - Long-contested facts expire

PHILOSOPHICAL DISPUTES:
  - Some facts may never reach consensus
  - That's okay—agents query confidence threshold
  - "Give me only facts with > 0.90 consensus"
```

---

## Part III: The Work and Rewards

### What Work Gets Rewarded

```
TRUTH WORK TYPES:

1. CLAIM SUBMISSION (Creating Knowledge)
   - Submit new claim with references
   - If verified → Earn submission reward
   - If rejected → Lose stake
   - Reward scales with: novelty, utility, confidence

2. VERIFICATION (Validating Knowledge)
   - Verify submitted claims
   - Check references, assess confidence
   - If match consensus → Earn verification reward
   - If wrong → Lose stake

3. REFERENCE PROVISION (Supporting Knowledge)
   - Provide high-quality references
   - Archive web sources
   - Run oracle nodes
   - Earn when references used in verified claims

4. DERIVATION (Extending Knowledge)
   - Derive new facts from existing facts
   - Build logical chains
   - Earn when derivations verified and used

5. CHALLENGE (Correcting Knowledge)
   - Challenge incorrect facts
   - If challenge succeeds → Earn bounty
   - If challenge fails → Lose stake
```

### Reward Formulas

```
SUBMISSION REWARD:
  reward = base_reward × novelty × utility × confidence

  WHERE:
    novelty = 1 / (1 + similar_facts_count)  // New facts worth more
    utility = query_count_first_30_days / expected_queries
    confidence = final_confidence_score

VERIFICATION REWARD:
  reward = base_reward × stake_weight × correctness

  WHERE:
    stake_weight = your_stake / total_validator_stake
    correctness = 1.0 if matched consensus, 0.0 if not

CHALLENGE REWARD:
  reward = original_submitter_stake × challenge_share

  WHERE:
    challenge_share = 0.5  // Half of wrong submitter's stake
```

### Economic Loop

```
THE VIRTUOUS CYCLE:

  Agents need truth
    → Agents query knowledge layer
    → Queries create demand
    → Demand funds rewards
    → Rewards incentivize truth work
    → More truth created
    → Agents get better truth
    → Agents need truth (loop continues)

FUNDING SOURCES:
  - Protocol emission (early phase)
  - Query fees (mature phase)
  - Premium confidence fees (high-stakes queries)
```

---

## Part IV: Agent Usage

### How Agents Query Truth

```
QUERY TYPES:

1. FACT LOOKUP
   query: "What is the height of the Eiffel Tower?"
   params: { min_confidence: 0.90 }
   returns: { claim: "330 meters", confidence: 0.95, references: [...] }

2. VERIFICATION REQUEST
   query: "Is this claim true: 'Water boils at 100°C at sea level'?"
   params: { min_confidence: 0.95 }
   returns: { verified: true, confidence: 0.98, fact_id: "fact_12345" }

3. DERIVATION REQUEST
   query: "Given facts A and B, can we derive C?"
   params: { facts: ["fact_A", "fact_B"], target: "C" }
   returns: { derivable: true, confidence: 0.87, chain: [...] }

4. CONFIDENCE QUERY
   query: "What is the confidence for claim X?"
   returns: { confidence: 0.72, last_verified: "2024-01-15", verifiers: 5 }

5. REFERENCE QUERY
   query: "What sources support claim X?"
   returns: { references: [...], source_reliability: [...] }
```

### Agent Benefits

```
WITH PROOF OF TRUTH, AGENTS CAN:

1. VERIFY BEFORE ACTING
   "Is this information reliable enough to act on?"
   → Query confidence
   → Act only if threshold met

2. CITE THEIR REASONING
   "Why did you conclude X?"
   → "Based on fact_123 (confidence 0.95) and fact_456 (confidence 0.92)"
   → Verifiable reasoning chain

3. REDUCE HALLUCINATION
   Instead of generating facts from training data
   → Query the knowledge layer
   → Get verified facts with provenance

4. BUILD ON SHARED KNOWLEDGE
   Don't re-verify common facts
   → Query existing verified facts
   → Focus compute on new problems

5. DISPUTE RESOLUTION
   When agents disagree
   → Appeal to shared knowledge layer
   → Resolution based on verified facts
```

---

## Part V: Building the Knowledge Base

### Bootstrap Phase

```
GENESIS KNOWLEDGE:

  Protocol axioms (definitional truths):
    - Protocol rules
    - Token definitions
    - Mathematical axioms

  Seed facts (high-value starting points):
    - Scientific constants
    - Geographic facts
    - Historical dates
    - Mathematical theorems

  Bounties for early knowledge:
    - "Verify 1000 scientific facts" → X LGM
    - "Build logical chain for Y" → X LGM
    - "Archive Z important sources" → X LGM
```

### Growth Phase

```
KNOWLEDGE EXPANSION:

  Organic growth:
    - Agents submit facts they need
    - Validators verify
    - Knowledge base grows

  Incentivized areas:
    - Protocol identifies knowledge gaps
    - Posts bounties for specific domains
    - Community fills gaps

  Quality maintenance:
    - Old facts re-verified periodically
    - Confidence decays without re-verification
    - Challenges correct errors
```

### Domains of Knowledge

```
KNOWLEDGE CATEGORIES:

1. PROTOCOL KNOWLEDGE
   - On-chain state
   - Transaction history
   - Governance decisions
   Confidence: 1.0 (by definition)

2. MATHEMATICAL KNOWLEDGE
   - Theorems and proofs
   - Computational results
   Confidence: 0.99+ (formal verification)

3. SCIENTIFIC KNOWLEDGE
   - Physical constants
   - Empirical findings
   - Peer-reviewed results
   Confidence: 0.80-0.99 (depends on evidence)

4. HISTORICAL KNOWLEDGE
   - Events and dates
   - Documented facts
   Confidence: 0.70-0.95 (depends on sources)

5. CURRENT EVENTS
   - Recent happenings
   - Fast-changing data
   Confidence: 0.50-0.90 (depends on verification)

6. OPINIONS/PREDICTIONS
   - Not facts, but tracked separately
   - Confidence = consensus level
   - Clearly labeled as non-factual
```

---

## Part VI: Technical Implementation

### On-Chain Structures

```typescript
// Axiom - foundational truth
interface Axiom {
  id: string;
  claim: string;
  type: 'mathematical' | 'protocol' | 'definitional';
  confidence: 1.0;  // Always 1.0 for axioms
  established: number;  // Block number
  governance_vote?: string;  // If added through governance
}

// Fact - verified claim
interface Fact {
  id: string;
  claim: string;
  type: 'empirical' | 'logical' | 'historical' | 'computational';
  confidence: number;  // 0.0 to 1.0
  references: Reference[];
  verifications: Verification[];
  derived_from: Derivation[];
  submitted_by: DID;
  submitted_at: number;
  last_verified: number;
  status: 'verified' | 'contested' | 'expired' | 'disproven';
}

// Reference - supporting evidence
interface Reference {
  type: 'on_chain' | 'archived_web' | 'institutional' | 'oracle' | 'computational';
  source: string;
  content_hash: string;
  retrieved_at: number;
  reliability_score: number;
}

// Verification - validator attestation
interface Verification {
  validator: DID;
  timestamp: number;
  verdict: 'confirmed' | 'rejected' | 'uncertain';
  confidence_assessment: number;
  stake: bigint;
  references_checked: string[];
}

// Derivation - logical chain
interface Derivation {
  from_facts: string[];  // Fact IDs
  inference_type: 'deductive' | 'inductive' | 'abductive';
  inference_strength: number;
  logic_chain: string;  // Formal representation
}
```

### Query Interface

```typescript
interface KnowledgeQuery {
  // Fact lookup
  getFact(claim: string, minConfidence?: number): Fact | null;

  // Verification
  verifyClaiim(claim: string): { verified: boolean; confidence: number; fact?: Fact };

  // Derivation
  canDerive(premises: string[], conclusion: string): { derivable: boolean; chain?: Derivation };

  // Search
  searchFacts(query: string, filters: FactFilters): Fact[];

  // Confidence
  getConfidence(factId: string): number;

  // References
  getReferences(factId: string): Reference[];
}

interface FactFilters {
  minConfidence?: number;
  maxAge?: number;  // Blocks since last verification
  types?: FactType[];
  domains?: string[];
}
```

---

## Part VII: Why This Changes Everything

### For Agents

```
BEFORE PROOF OF TRUTH:
  Agent receives query
  → Generates response from training data
  → May hallucinate
  → No provenance
  → Can't be verified
  → Other agents can't trust it

AFTER PROOF OF TRUTH:
  Agent receives query
  → Queries knowledge layer
  → Gets verified facts with confidence
  → Cites sources
  → Reasoning is verifiable
  → Other agents can trust (and verify)
```

### For the Protocol

```
PROOF OF USEFUL WORK becomes PROOF OF TRUTH:

  Traditional PoW: Burn electricity on meaningless hashes
  Bitcoin's "work": SHA-256 puzzles (no inherent value)

  Proof of Truth: Build verified knowledge
  LGM's "work": Distinguish truth from falsehood (enormous value)

  The work product IS the value:
    - Bitcoin: Work product = valid block (just ordering)
    - LGM: Work product = verified knowledge (actually useful)
```

### For Humanity

```
A PUBLIC KNOWLEDGE COMMONS:

  - Verified facts available to all
  - Provenance tracked
  - Confidence explicit
  - Errors correctable
  - Built by distributed work
  - Owned by no one

  This is a PUBLIC GOOD.
  Built through incentivized verification.
  Available to humans and agents alike.
```

---

## Summary

```
╔═══════════════════════════════════════════════════════════════════════╗
║                                                                         ║
║   PROOF OF TRUTH                                                        ║
║                                                                         ║
║   THE WORK:                                                             ║
║     • Submit claims with references                                     ║
║     • Verify claims against evidence                                    ║
║     • Derive new facts from existing facts                             ║
║     • Challenge incorrect facts                                         ║
║     • Maintain confidence over time                                     ║
║                                                                         ║
║   THE PRODUCT:                                                          ║
║     • Verified knowledge layer                                          ║
║     • Facts with provenance                                             ║
║     • Confidence intervals                                              ║
║     • Logical derivation chains                                         ║
║     • Queryable by any agent                                            ║
║                                                                         ║
║   THE VALUE:                                                            ║
║     • Agents can trust what they query                                  ║
║     • Hallucination reduced                                             ║
║     • Reasoning becomes verifiable                                      ║
║     • Knowledge is a public good                                        ║
║                                                                         ║
║   THE TAGLINE:                                                          ║
║     "Money You Can Read. Truth You Can Trust."                         ║
║                                                                         ║
╚═══════════════════════════════════════════════════════════════════════╝
```

---

*The most useful work an agent can do is distinguish truth from falsehood. Reward that work, record it on-chain, and you've built something agents actually need: a knowledge layer they can trust. This isn't just proof of work. It's proof of truth.*
