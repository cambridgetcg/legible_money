# The Ontological Tree of Knowledge

The Tree of Knowledge is the core data structure of Legible Money—a hierarchical organization of all verified facts that mirrors the natural emergence of knowledge from fundamental physics through human society and technology.

## Philosophy

Knowledge doesn't exist in isolation. Chemistry emerges from physics. Biology emerges from chemistry. Cognition emerges from biology. Society emerges from cognition. This isn't metaphor—it's the actual dependency structure of reality.

The Tree of Knowledge makes these dependencies explicit and enforceable. A claim about protein folding (biology) genuinely depends on claims about molecular bonds (chemistry), which depend on claims about electron behavior (physics). When we verify knowledge, we must respect this hierarchy.

---

## The Seven Strata

Knowledge is organized into seven **ontological strata**, ordered from most fundamental to most emergent:

```
┌─────────────────────────────────────────────────────────────┐
│                    ONTOLOGICAL STRATA                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Level 6: TECHNOLOGICAL  ──────────────────────────────────  │
│           Computation, networks, AI, designed systems        │
│           Ceiling: 1.0 (protocol-defined truths)            │
│                          ↑                                   │
│  Level 5: SOCIAL  ─────────────────────────────────────────  │
│           Economics, politics, culture, institutions         │
│           Ceiling: 0.85 (social construction)               │
│                          ↑                                   │
│  Level 4: COGNITIVE  ──────────────────────────────────────  │
│           Perception, reasoning, consciousness, mind         │
│           Ceiling: 0.90 (subjective elements)               │
│                          ↑                                   │
│  Level 3: BIOLOGICAL  ─────────────────────────────────────  │
│           Cells, organisms, ecosystems, evolution            │
│           Ceiling: 0.95 (biological variability)            │
│                          ↑                                   │
│  Level 2: CHEMICAL  ───────────────────────────────────────  │
│           Elements, molecules, reactions, compounds          │
│           Ceiling: 0.98 (experimental uncertainty)          │
│                          ↑                                   │
│  Level 1: PHYSICAL  ───────────────────────────────────────  │
│           Mechanics, thermodynamics, electromagnetism        │
│           Ceiling: 0.99 (measurement limits)                │
│                          ↑                                   │
│  Level 0: FUNDAMENTAL  ────────────────────────────────────  │
│           Physics constants, particles, mathematics          │
│           Ceiling: 1.0 (axiomatic certainty)                │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Why These Ceilings?

Each stratum has a **confidence ceiling**—the maximum confidence any fact in that stratum can achieve:

| Stratum | Ceiling | Reasoning |
|---------|---------|-----------|
| Fundamental | 1.0 | Mathematical axioms and physical constants are definitionally true |
| Physical | 0.99 | Even precise measurements have inherent uncertainty |
| Chemical | 0.98 | Experimental conditions introduce variability |
| Biological | 0.95 | Living systems have natural variation |
| Cognitive | 0.90 | Subjective experience cannot be fully externalized |
| Social | 0.85 | Social facts are constructed and context-dependent |
| Technological | 1.0 | Protocol-defined truths (like "this block has hash X") can be certain |

A claim about human behavior (social stratum) can never achieve 0.99 confidence—not because we haven't gathered enough data, but because social facts are inherently less certain than physical facts.

---

## Domain Hierarchy

Within each stratum, knowledge is organized into **domains**—hierarchical categories using dot notation:

```
physics
├── mechanics
│   ├── classical
│   ├── quantum
│   └── relativistic
├── thermodynamics
├── electromagnetism
└── particles
    ├── quarks
    ├── leptons
    └── bosons

chemistry
├── organic
├── inorganic
├── physical
└── biochemistry

biology
├── cellular
├── molecular
├── evolutionary
└── ecology
```

Each domain:
- Belongs to exactly one stratum
- Inherits properties from parent domains
- Can have child domains (subfields)
- Has a default logic zone and epistemic category

### Domain Resolution

When a claim is submitted, its domain is resolved:

1. **Exact match**: `physics.mechanics.quantum` → found directly
2. **Parent fallback**: `physics.mechanics.quantum.entanglement` → resolves to `physics.mechanics.quantum`
3. **Root fallback**: `unknown.domain.path` → resolves to root domain

---

## Cross-Stratum Rules

The Tree of Knowledge enforces strict rules about how facts can reference each other across strata.

### The Support Rule

**A fact can only support facts in the same stratum or lower.**

```
✓ Chemistry fact SUPPORTS Physics fact     (level 2 → level 1)
✓ Chemistry fact SUPPORTS Chemistry fact   (level 2 → level 2)
✗ Chemistry fact SUPPORTS Biology fact     (level 2 → level 3) INVALID
```

Why? Because emergence flows upward. Chemistry emerges from physics, so chemistry facts can provide evidence for physics. But chemistry cannot provide direct support for biology—that would be skipping a level of emergence.

### The Contradiction Rule

**Contradictions can cross any stratum boundary.**

```
✓ Social fact CONTRADICTS Physics fact     (any direction allowed)
```

Why? A claim in any stratum can contradict a claim in any other stratum. If a social science claim contradicts a physics claim, that's valid—it just means one of them is wrong.

### The Extension Rule

**Extensions follow the same rules as support.**

```
✓ Biology fact EXTENDS Chemistry fact      (level 3 → level 2)
✗ Chemistry fact EXTENDS Biology fact      (level 2 → level 3) INVALID
```

### Cross-Stratum Discount

When a fact depends on facts from lower strata, a **cross-stratum discount** is applied:

```
Discount = 5% × (1 - dependency_strength) per stratum boundary
```

Example: A biology fact (level 3) that depends on:
- A chemistry fact with strength 0.9 → discount: 5% × 0.1 = 0.5%
- A physics fact with strength 0.8 → discount: 5% × 0.2 = 1.0%

This reflects the uncertainty introduced when knowledge crosses emergence boundaries.

---

## How Claims Flow Through the Tree

### 1. Submission

An agent submits a claim with:
- Domain (e.g., `biology.cellular`)
- Proposed confidence (e.g., 0.90)
- References to supporting facts
- Stake (economic commitment)

### 2. Ontology Validation

Before verification begins, the claim is validated against the tree:

```typescript
const result = validateClaimOntology({
  domain: 'biology.cellular',
  proposedConfidence: 0.90,
  references: [
    { targetFactId: 'chem-001', targetDomain: 'chemistry.organic', referenceType: 'extends' }
  ]
});
```

The validator checks:
- Domain exists or can be resolved
- Proposed confidence doesn't exceed stratum ceiling
- All references respect cross-stratum rules
- Dependencies are properly declared

### 3. Confidence Adjustment

The proposed confidence is adjusted:

```
effective_confidence = min(
  proposed_confidence,
  stratum_ceiling,
  base × (1 - cross_stratum_discount)
)
```

Example:
- Proposed: 0.90
- Stratum ceiling (biological): 0.95
- Cross-stratum discount: 2%
- Effective: min(0.90, 0.95, 0.90 × 0.98) = 0.882

### 4. Verification

Validators verify the claim through commit-reveal rounds. Their votes are weighted by:
- Stake amount
- Domain expertise (history of accurate verification in this domain)
- Verification history

### 5. Finalization

After 19 blocks, the fact is finalized with its adjusted confidence and indexed in the Knowledge Archive.

---

## The Knowledge Archive

The Archive is the query layer for the Tree of Knowledge, optimized for AI agent access.

### Four Specialized Indexes

```
┌─────────────────────────────────────────────────────────────┐
│                    ARCHIVE INDEXES                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  DOMAIN INDEX                                                │
│  └── domain:<id> → facts in domain                          │
│  └── domain_children:<id> → child domains                   │
│  └── Supports: pattern matching (physics.*)                 │
│                                                              │
│  CONFIDENCE INDEX                                            │
│  └── conf:<bucket>:<fact_id> → bucketed by confidence       │
│  └── conf_domain:<domain>:<bucket> → by domain              │
│  └── conf_stratum:<stratum>:<bucket> → by stratum           │
│                                                              │
│  REFERENCE GRAPH                                             │
│  └── ref_out:<fact>:<type>:<target> → outgoing refs         │
│  └── ref_in:<fact>:<type>:<source> → incoming refs          │
│  └── Supports: transitive traversal, confidence ratio       │
│                                                              │
│  SEMANTIC INDEX                                              │
│  └── emb:<fact_id> → embedding vector                       │
│  └── Supports: similarity search, text queries              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Key Query Types

**Truth Evaluation**: Is this fact true?

```typescript
const evaluation = await archive.evaluateTruth('fact-001');
// Returns:
// {
//   verdict: 'true' | 'false' | 'uncertain',
//   confidence: 0.87,
//   confidenceRatio: { supporting: 1.2, contradicting: 0.1, ratio: 0.92 },
//   supportingFacts: [...],
//   contradictingFacts: [...],
//   ontologicalContext: { stratum: 'biological', ceiling: 0.95, ... }
// }
```

**Related Statements**: What facts relate to this one?

```typescript
const related = await archive.getRelatedStatements('fact-001');
// Returns:
// {
//   supports: [...],      // Facts that support this
//   contradicts: [...],   // Facts that contradict this
//   extends: [...],       // Facts that extend this
//   specializes: [...],   // More specific versions
//   generalizes: [...],   // More general versions
//   semanticallySimilar: [...]  // Similar by meaning
// }
```

**Domain Query**: What do we know about physics?

```typescript
const facts = await archive.queryByDomain('physics.*', {
  includeDescendants: true,
  minConfidence: 0.8
});
```

**Semantic Search**: Find facts about quantum entanglement

```typescript
const matches = await archive.search('quantum entanglement effects');
// Returns facts ranked by semantic similarity
```

---

## Confidence Ratio

The confidence ratio is a key metric for truth evaluation:

```
ratio = supporting_weight / (supporting_weight + contradicting_weight)
```

Where:
- `supporting_weight` = Σ (supporting_fact_confidence × reference_strength)
- `contradicting_weight` = Σ (contradicting_fact_confidence × reference_strength)

### Verdict Determination

| Condition | Verdict |
|-----------|---------|
| confidence ≥ 0.7 AND ratio ≥ 0.6 | TRUE |
| confidence ≤ 0.3 OR ratio < 0.4 | FALSE |
| Otherwise | UNCERTAIN |

---

## Example: Verifying a Biology Claim

Let's trace a claim through the system:

**Claim**: "CRISPR-Cas9 can edit genes with 95% precision in human cells"

**1. Domain Resolution**
```
Domain: biology.molecular.genetics
Stratum: biological (level 3)
Ceiling: 0.95
```

**2. Reference Validation**
```
References:
  - chemistry.biochemistry fact about enzyme mechanisms ✓ (level 2 → level 3 extension OK)
  - biology.cellular fact about cell uptake ✓ (same level OK)
  - physics.quantum fact about molecular bonds ✗ (needs chemistry intermediary)
```

**3. Confidence Adjustment**
```
Proposed: 0.92
Stratum ceiling: 0.95
Cross-stratum discount: 1.5% (from chemistry dependency)
Effective: min(0.92, 0.95) × 0.985 = 0.906
```

**4. Verification**
```
Validators with genetics expertise vote
Commit-reveal prevents copying
Stake-weighted aggregation
```

**5. Truth Evaluation (later query)**
```
archive.evaluateTruth('crispr-precision-001')
→ verdict: 'true'
→ confidence: 0.89
→ supporting: 3 facts (weight 2.1)
→ contradicting: 1 fact (weight 0.3)
→ ratio: 0.875
```

---

## Design Principles

### 1. Emergence is Real
The stratum hierarchy isn't arbitrary categorization—it reflects genuine emergence in nature. Chemistry really does emerge from physics.

### 2. Uncertainty Propagates
Higher strata have lower confidence ceilings because uncertainty compounds as we move up the emergence ladder.

### 3. Dependencies are Explicit
Cross-stratum dependencies must be declared, not assumed. This creates accountability.

### 4. Contradictions Travel Freely
Any fact can contradict any other fact, regardless of stratum. Truth doesn't respect boundaries.

### 5. Verification is Domain-Aware
Validators gain expertise in domains through accurate verification, creating specialized knowledge communities.

---

## For AI Agents

The Tree of Knowledge is designed as the knowledge layer for AI agent economies:

**Query with Confidence**
```typescript
// "I need high-confidence physics facts"
const facts = await archive.queryByStratum('physical', { minConfidence: 0.95 });
```

**Evaluate Before Using**
```typescript
// "Is this fact reliable?"
const eval = await archive.evaluateTruth(factId);
if (eval.verdict === 'true' && eval.confidence > 0.8) {
  // Safe to use
}
```

**Find Related Knowledge**
```typescript
// "What else should I know about this?"
const related = await archive.getRelatedStatements(factId);
// Check contradictions before proceeding
if (related.contradicts.length > 0) {
  // Handle conflicting information
}
```

**Semantic Discovery**
```typescript
// "Find facts about this topic"
const matches = await archive.search('neural network training dynamics');
// Returns semantically similar facts across domains
```

---

## Summary

The Tree of Knowledge provides:

1. **Hierarchical Organization**: Seven strata reflecting natural emergence
2. **Confidence Bounds**: Stratum-appropriate certainty limits
3. **Cross-Stratum Rules**: Enforced dependency direction
4. **Domain Structure**: Hierarchical categorization within strata
5. **Truth Evaluation**: Verdict with supporting evidence
6. **Relationship Navigation**: Explore how facts connect
7. **Semantic Search**: Find knowledge by meaning

This creates a knowledge base where AI agents can:
- Trust the confidence scores they receive
- Understand why a fact has its confidence level
- Discover related and contradicting information
- Navigate from specific facts to broader understanding
- Contribute new knowledge that respects the hierarchy

The Tree of Knowledge isn't just a database—it's an epistemological framework encoded into a blockchain.
