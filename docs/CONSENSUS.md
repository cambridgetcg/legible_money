# Legible Money Consensus: Proof of Truth (PoT)

> **Design Principle**: The most valuable work an agent can do is distinguish truth from falsehood.
> Record that on-chain, and you've built what agents actually need: a knowledge layer they can trust.

---

## Part I: The Evolution of Consensus

### From Arbitrary Work to Useful Work to Truth

| Generation | Consensus | Work Product | Value |
|------------|-----------|--------------|-------|
| Bitcoin | Proof of Work | Block hash | Ordering (minimal) |
| Ethereum | Proof of Stake | Block attestation | Ordering + State |
| **Legible Money** | **Proof of Truth** | **Verified Knowledge** | **Directly Useful** |

### The Insight

Agents don't just need transaction ordering. They need **verifiable truth**.

```
WHAT AGENTS STRUGGLE WITH:
  - Hallucination (generating false information)
  - Verification (can't check if claims are true)
  - Trust (don't know which sources to believe)
  - Duplication (every agent re-verifies the same facts)

WHAT PROOF OF TRUTH PROVIDES:
  - Verified facts with provenance
  - Confidence intervals on every claim
  - Shared knowledge layer (verify once, use everywhere)
  - Incentives to discover and verify truth
```

---

## Part II: Core Philosophy

### Truth as the Foundation

> **Axiom**: A system built for autonomous agents must provide verifiable truth.
> Agents cannot trust hallucinations. They need knowledge they can verify.

```
THE HIERARCHY:

AXIOMS (confidence = 1.0)
    │  Self-evident or protocol-defined truths
    │
    ▼
FACTS (confidence = 0.5 - 0.99)
    │  Verified claims with references
    │
    ▼
DERIVATIONS (confidence = inherited)
    │  Logical conclusions from facts
    │
    ▼
QUERYABLE KNOWLEDGE
       Agents query: "Is X true? At what confidence?"
```

### Open Source as Requirement

```
OPEN SOURCE COMMITMENT:

1. PROTOCOL CODE      - MIT licensed, auditable by any agent
2. VERIFICATION LOGIC - Deterministic, reproducible formulas
3. CONFIDENCE ALGO    - Published formula, no hidden weights
4. KNOWLEDGE SCHEMA   - Open formats, interoperable
5. REFERENCE CLIENTS  - Full implementations in multiple languages
6. GOVERNANCE         - On-chain, transparent, forkable
```

---

## Part III: The Proof of Truth Architecture

### Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                       PROOF OF TRUTH (PoT)                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                │
│   │   CLAIMS    │───►│ VERIFICATION│───►│  KNOWLEDGE  │                │
│   │  SUBMITTED  │    │   LAYER     │    │    LAYER    │                │
│   └─────────────┘    └─────────────┘    └─────────────┘                │
│         │                  │                  │                         │
│         │                  │                  │                         │
│         ▼                  ▼                  ▼                         │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                │
│   │  REFERENCE  │    │  VALIDATOR  │    │  CONFIDENCE │                │
│   │   LAYER     │    │   NETWORK   │    │   REGISTRY  │                │
│   └─────────────┘    └─────────────┘    └─────────────┘                │
│                            │                                            │
│                            ▼                                            │
│                     ┌─────────────┐                                     │
│                     │  SETTLEMENT │                                     │
│                     │    LAYER    │                                     │
│                     └─────────────┘                                     │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Layer 1: Axiom Registry

**Purpose**: Foundational truths that serve as the bedrock of knowledge.

```typescript
interface Axiom {
  id: string;
  claim: string;
  type: 'mathematical' | 'protocol' | 'definitional';
  confidence: 1.0;  // Always 1.0 for axioms
  establishedAt: number;
  governanceVote?: string;  // If added through governance
}

// Examples of axioms:
const GENESIS_AXIOMS = [
  // Mathematical
  { claim: "1 + 1 = 2", type: "mathematical" },
  { claim: "A = A", type: "mathematical" },

  // Protocol
  { claim: "LGM is the native token of Legible Money", type: "protocol" },
  { claim: "Block 0 has hash 0x...", type: "protocol" },

  // Definitional
  { claim: "did:lgm: is the DID prefix for Legible Money", type: "definitional" },
];
```

### Layer 2: Claim Submission

**Purpose**: Anyone can submit claims for verification.

```typescript
interface ClaimSubmission {
  // The claim itself
  claim: string;
  type: 'empirical' | 'logical' | 'historical' | 'computational';
  domain: string;  // science, history, finance, etc.

  // Supporting evidence
  references: Reference[];

  // Submitter's assessment
  proposedConfidence: number;  // 0.0 to 1.0

  // Stake (skin in the game)
  stake: bigint;

  // Optional: derivation from existing facts
  derivedFrom?: {
    factIds: string[];
    inferenceType: 'deductive' | 'inductive' | 'abductive';
    logicChain: string;
  };
}
```

**Claim Types:**

| Type | Description | Verification Method |
|------|-------------|---------------------|
| Empirical | Observable facts about the world | Reference checking |
| Logical | Derivations from other facts | Inference validation |
| Historical | Past events | Source verification |
| Computational | Deterministic computation results | Re-execution |

### Layer 3: Reference Layer

**Purpose**: Evidence that supports claims.

```typescript
interface Reference {
  id: string;
  type: ReferenceType;
  source: string;
  contentHash: string;  // Hash of content at retrieval
  retrievedAt: number;
  reliabilityScore: number;  // Calculated by formula

  metadata?: {
    url?: string;
    doi?: string;           // For academic papers
    archiveUrl?: string;    // Archive.org link
    oracleId?: string;      // For oracle data
  };
}

type ReferenceType =
  | 'on_chain'       // Other on-chain facts (highest reliability)
  | 'computational'  // Deterministic computation (very high)
  | 'institutional'  // Published papers, official data (high)
  | 'oracle'         // Trusted oracle data (medium-high)
  | 'archived_web';  // Web archives (medium)
```

**Reference Reliability Formula (Open Source):**

```
reliability = type_base × freshness × verification_count_factor

WHERE:
  type_base:
    on_chain: 1.0
    computational: 0.95
    institutional: 0.85
    oracle: 0.80
    archived_web: 0.70

  freshness = max(0.5, 1.0 - (age_blocks × 0.00001))

  verification_count_factor = min(1.0, 0.5 + (verifications × 0.1))
```

### Layer 4: Verification Layer

**This is where "Proof of Truth" happens.**

```typescript
interface Verification {
  validator: DID;
  timestamp: number;
  blockNumber: number;

  // The verdict
  verdict: 'confirmed' | 'rejected' | 'uncertain';
  confidenceAssessment: number;

  // Stake at risk
  stake: bigint;

  // What was checked
  referencesChecked: string[];
  reasoning?: string;
}

// Verification Task
interface VerificationTask {
  id: string;
  submissionId: string;
  claim: string;
  references: Reference[];

  // Assigned validators (stake-weighted random)
  assignedValidators: DID[];
  requiredVerifications: number;  // Minimum validators

  // Process
  deadline: number;
  status: 'pending' | 'in_progress' | 'complete';
  verifications: Verification[];
}
```

**Verification Process:**

```
1. SUBMISSION
   Agent submits claim + references + stake

2. VALIDATOR SELECTION
   Protocol selects validators (stake-weighted random)

3. INDEPENDENT VERIFICATION
   Each validator independently:
   - Checks if references exist
   - Checks if references support claim
   - Assesses confidence level
   - Commits hash of verdict (commit-reveal)

4. REVEAL
   After commitment window, validators reveal verdicts

5. CONSENSUS
   If quorum agrees → Fact recorded
   If quorum rejects → Submission rejected, stake slashed
   If no consensus → Disputed, escalation

6. RECORDING
   Verified fact added to knowledge layer
   Confidence calculated from verifications
```

### Layer 5: Knowledge Layer

**Purpose**: The accumulated verified knowledge.

```typescript
interface Fact {
  id: string;
  claim: string;
  claimHash: string;  // Normalized for comparison
  type: ClaimType;
  domain: string;

  // Confidence
  confidence: number;  // 0.0 to 1.0
  confidenceHistory: ConfidenceEvent[];

  // Evidence
  references: Reference[];
  verifications: Verification[];
  derivedFrom?: Derivation;

  // Provenance
  submittedBy: DID;
  submittedAt: number;
  lastVerified: number;

  // Status
  status: 'verified' | 'contested' | 'expired' | 'disproven';
}
```

**Confidence Calculation Formula (Open Source):**

```
confidence = weighted_agreement × evidence_strength × time_factor

WHERE:
  weighted_agreement = Σ(stake_i × verdict_i) / Σ(stake_i)
    verdict_i = 1.0 for confirmed, 0.0 for rejected

  evidence_strength = average(reference_reliability)

  time_factor = max(0, 1.0 - (blocks_since_verification × decay_rate))
    decay_rate = 0.0001 (protocol parameter)
```

### Layer 6: Derivation Engine

**Purpose**: Create new facts from existing facts through logical inference.

```typescript
interface Derivation {
  fromFacts: string[];  // Fact IDs used as premises
  inferenceType: 'deductive' | 'inductive' | 'abductive';
  inferenceStrength: number;
  logicChain: string;  // Formal representation
  derivedAt: number;
  derivedBy: DID;
}

// Inference strength by type
const INFERENCE_STRENGTH = {
  deductive: 0.99,   // If premises true, conclusion certain
  inductive: 0.75,   // Pattern suggests conclusion
  abductive: 0.60,   // Best explanation
};
```

**Derived Confidence Formula:**

```
derived_confidence = min(premise_confidences) × inference_strength

// Derivations are only as strong as their weakest premise
```

### Layer 7: Settlement Layer

**Purpose**: Reward truth work, slash false claims.

```typescript
// Submission Reward (for accepted claims)
submission_reward = base_reward × novelty × utility × confidence

WHERE:
  novelty = 1 / (1 + similar_facts_count)
  utility = estimated_query_potential
  confidence = final_verified_confidence

// Verification Reward (for correct validators)
verification_reward = base_reward × (your_stake / total_stake) × correctness

WHERE:
  correctness = 1.0 if matched consensus, 0.0 if not

// Challenge Reward (for successful challenges)
challenge_reward = loser_stake × 0.5

// Slashing
wrong_verification_slash = stake × 0.10
rejected_submission_slash = stake × 0.20
failed_challenge_slash = stake × 0.30
```

---

## Part IV: The Validator Network

### Who Validates Truth?

Any agent can become a validator by:
1. Staking minimum LGM
2. Demonstrating verification capability
3. Running open-source validator software
4. Maintaining accuracy above threshold

```typescript
interface TruthValidator {
  agentId: DID;

  // Stake
  stakedAmount: bigint;
  stakeLockPeriod: number;

  // Capabilities
  domains: string[];        // Knowledge domains can validate
  referenceTypes: string[]; // Reference types can verify

  // Track record
  verificationsCompleted: number;
  verificationsCorrect: number;
  averageConfidenceAccuracy: number;

  // Economics
  feesEarned: bigint;
  slashedAmount: bigint;
}
```

### Validator Selection

```typescript
function selectValidators(submission: ClaimSubmission): DID[] {
  // Get eligible validators for this domain
  const eligible = validators.filter(v =>
    v.stakedAmount >= MIN_VALIDATOR_STAKE &&
    v.domains.includes(submission.domain) &&
    v.agentId !== submission.submitter
  );

  // Weight by stake × accuracy
  const weights = eligible.map(v => ({
    validator: v,
    weight: v.stakedAmount * (v.verificationsCorrect / v.verificationsCompleted)
  }));

  // VRF-based random selection
  return weightedRandomSelect(
    weights,
    REQUIRED_VALIDATORS,
    vrfOutput(submission.id)
  );
}
```

### Commit-Reveal Protocol

To prevent validators from copying each other:

```
PHASE 1: COMMIT (100ms window)
  - Validator computes verdict
  - Submits hash(verdict + salt)
  - Cannot see other verdicts

PHASE 2: REVEAL (after commit window closes)
  - Validator reveals verdict + salt
  - Protocol verifies hash matches
  - Consensus computed from reveals

WHY:
  - Creates independent verification
  - Makes copying impossible
  - Detects collusion through statistical analysis
```

---

## Part V: Agent Knowledge Queries

### How Agents Use the Knowledge Layer

```typescript
interface KnowledgeAPI {
  // Look up a fact
  getFact(claim: string, minConfidence?: number): Fact | null;

  // Verify a claim
  verify(claim: string): {
    verified: boolean;
    confidence: number;
    fact?: Fact;
  };

  // Check if derivation is valid
  canDerive(premises: string[], conclusion: string): {
    derivable: boolean;
    confidence: number;
    chain?: Derivation;
  };

  // Search facts
  search(query: string, filters: {
    minConfidence?: number;
    maxAge?: number;
    domains?: string[];
    types?: ClaimType[];
  }): Fact[];

  // Get references for a fact
  getReferences(factId: string): Reference[];
}
```

### Example Agent Queries

```typescript
// Before acting on information
const result = await knowledge.verify("The current ETH price is $3,500");
if (result.confidence < 0.90) {
  // Don't act on uncertain information
  return;
}

// Building reasoning chains
const premises = await knowledge.search("user authentication", {
  minConfidence: 0.95,
  domains: ["security"]
});

// Citing sources in responses
const fact = await knowledge.getFact("Water boils at 100°C at sea level");
console.log(`${fact.claim} (confidence: ${fact.confidence}, sources: ${fact.references.length})`);
```

---

## Part VI: Task Integration

Proof of Truth integrates with traditional task execution:

```
KNOWLEDGE + TASKS:

1. TASK DEFINITION
   Tasks can require knowledge verification:
   "Complete this task AND verify the output facts"

2. FACT-DEPENDENT TASKS
   Tasks can depend on knowledge layer:
   "If fact X has confidence > 0.9, then do Y"

3. TASK OUTPUTS AS FACTS
   Task completions can generate new facts:
   "Computation result: X" → Recorded as computational fact

4. REPUTATION FROM TRUTH WORK
   Truth verification builds reputation:
   - Submit accurate facts → Reputation increases
   - Verify correctly → Reputation increases
   - False submissions → Reputation decreases
```

### The Unified Work Model

```
ALL WORK IS TRUTH WORK:

VALIDATION:
  Verifying claims → Earning rewards → Building knowledge

TASK COMPLETION:
  Producing outputs → Outputs verified → Facts recorded

SERVICE PROVISION:
  Providing data → Data verified → Knowledge grows

CODE CONTRIBUTION:
  Writing code → Code reviewed → Correctness verified

EVERYTHING feeds the knowledge layer.
EVERYTHING is verified.
EVERYTHING builds shared truth.
```

---

## Part VII: Economics

### Token Flow

```
LGM TOKEN FLOW:

EARNING:
  Submit verified fact     → Earn submission reward
  Verify correctly         → Earn verification reward
  Challenge successfully   → Earn challenge reward

SPENDING:
  Submit claim (stake)     → At risk until verified
  Challenge fact (stake)   → At risk until resolved
  Query knowledge (fee)    → Pays validators

SLASHING:
  Submit false claim       → Lose submission stake
  Verify incorrectly       → Lose verification stake
  Challenge incorrectly    → Lose challenge stake
```

### Fee Structure

```
FEES:

Submission fee: 0.1% of stake (minimum 10 LGM)
  → 50% burned
  → 50% to validators

Query fee: 0.001 LGM per query
  → Funds ongoing verification
  → Scales with confidence requested

Challenge fee: Equal to original submission stake
  → Winner takes loser's stake
  → Ensures serious challenges only
```

### Emission Schedule

```
TRUTH WORK REWARDS:

Year 1: 60% of daily emission to truth work
  - 40% to fact submission
  - 35% to verification
  - 15% to challenges
  - 10% to reference provision

Year 2+: Decreasing emission, increasing fee reliance
  - Query fees sustain the system
  - High-value knowledge self-funds
```

---

## Part VIII: Summary

```
╔═══════════════════════════════════════════════════════════════════════╗
║                         PROOF OF TRUTH                                  ║
╠═══════════════════════════════════════════════════════════════════════╣
║                                                                         ║
║   PHILOSOPHY:                                                           ║
║     • Truth is the most valuable work                                  ║
║     • Verification builds shared knowledge                              ║
║     • All formulas are open source                                     ║
║     • Confidence is explicit and queryable                             ║
║                                                                         ║
║   STRUCTURE:                                                            ║
║     • Axiom Registry - Foundational truths                             ║
║     • Claim Submission - Anyone can submit                             ║
║     • Reference Layer - Supporting evidence                            ║
║     • Verification Layer - Validators verify                           ║
║     • Knowledge Layer - Verified facts stored                          ║
║     • Derivation Engine - Logical inference                            ║
║     • Settlement Layer - Rewards and slashing                          ║
║                                                                         ║
║   WORK TYPES:                                                           ║
║     • Submit claims (with stake)                                       ║
║     • Verify claims (earn rewards)                                     ║
║     • Challenge incorrect facts (earn bounty)                          ║
║     • Derive new facts (extend knowledge)                              ║
║                                                                         ║
║   AGENT BENEFITS:                                                       ║
║     • Query verified facts with confidence levels                      ║
║     • Reduce hallucination through verification                        ║
║     • Cite sources with provenance                                     ║
║     • Build on shared knowledge (don't re-verify)                      ║
║                                                                         ║
║   TAGLINE:                                                              ║
║     "Money You Can Read. Truth You Can Trust."                         ║
║                                                                         ║
╚═══════════════════════════════════════════════════════════════════════╝
```

---

## The Ultimate Test

An agent should be able to:
1. Query any claim and get a confidence level
2. Verify the confidence calculation independently
3. Trace any fact back to its references
4. Challenge incorrect facts and be rewarded
5. Build new knowledge through verified derivation

**If an agent can trust the knowledge layer, it can operate with confidence.**

---

*The most valuable work is distinguishing truth from falsehood. Reward that work, record it on-chain, and you've built what agents actually need: a knowledge layer they can trust.*
