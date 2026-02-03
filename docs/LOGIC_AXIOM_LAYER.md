# Logic Axiom Layer

> **A Formal Logic System with Gödelian Awareness**

---

## Overview

The Logic Axiom Layer provides the formal foundation for verifiable knowledge. It explicitly acknowledges Gödel's incompleteness theorems, treating incompleteness as a feature rather than a limitation.

```
┌─────────────────────────────────────────────────────────────────────┐
│                     LOGIC AXIOM LAYER                                │
│                                                                      │
│  META-AXIOMS (About the system itself)                              │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │  M1: "This system is incomplete"                               │ │
│  │  M2: "Undecidability is a valid outcome"                       │ │
│  │  M3: "Consistency is assumed, not proven"                      │ │
│  │  M4: "Lower zones maintain provable consistency"               │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                              │                                       │
│                              ▼                                       │
│  LOGIC ZONES (Compartmentalized for consistency)                    │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐   │
│  │   ZONE 0    │ │   ZONE 1    │ │   ZONE 2    │ │   ZONE 3    │   │
│  │Propositional│ │ Presburger  │ │   Peano     │ │  Empirical  │   │
│  │  (Complete) │ │ (Decidable) │ │(Incomplete) │ │ (Evidence)  │   │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## The Gödelian Foundation

### Why Acknowledge Incompleteness?

**Gödel's First Incompleteness Theorem (1931):**
> Any consistent formal system F capable of expressing basic arithmetic contains statements that are true but unprovable within F.

**Gödel's Second Incompleteness Theorem:**
> Such a system F cannot prove its own consistency from within F.

**Implications for Legible Money:**

```
TRADITIONAL APPROACH (Flawed):
  ┌────────────────────────────────────────┐
  │  "Our system can verify all truths"    │
  │              ↓                         │
  │  User encounters undecidable statement │
  │              ↓                         │
  │  System breaks or gives wrong answer   │
  │              ↓                         │
  │  Trust is lost                         │
  └────────────────────────────────────────┘

GÖDELIAN APPROACH (Honest):
  ┌────────────────────────────────────────┐
  │  "Some truths cannot be proven here"   │
  │              ↓                         │
  │  User encounters undecidable statement │
  │              ↓                         │
  │  System returns: "undecidable"         │
  │              ↓                         │
  │  User knows to seek external resolution│
  │              ↓                         │
  │  Trust is maintained                   │
  └────────────────────────────────────────┘
```

---

## Logic Zones

### Zone 0: Propositional Logic (Complete, Decidable)

**Properties:**
- Complete: Every statement is provable or disprovable
- Decidable: Algorithms exist to determine truth
- Gödel does NOT apply

**Axioms:**
```
Identity:          A = A
Non-contradiction: ¬(A ∧ ¬A)
Excluded middle:   A ∨ ¬A
Modus ponens:      (A → B) ∧ A ⊢ B
Modus tollens:     (A → B) ∧ ¬B ⊢ ¬A
Transitivity:      (A → B) ∧ (B → C) ⊢ (A → C)
```

**Use Cases:**
- Boolean logic in smart contracts
- Simple validation rules
- Protocol invariants

### Zone 1: Presburger Arithmetic (Complete, Decidable)

**Properties:**
- Complete: Addition-only arithmetic is complete
- Decidable: All statements can be determined
- Gödel does NOT apply (no multiplication)

**Axioms:**
```
Zero:           0 ∈ ℕ
Successor:      ∀n ∈ ℕ. S(n) ∈ ℕ
No predecessor: ∀n ∈ ℕ. S(n) ≠ 0
Injective:      S(n) = S(m) → n = m
Add zero:       n + 0 = n
Add successor:  n + S(m) = S(n + m)
```

**Use Cases:**
- Token arithmetic
- Balance calculations
- Linear constraints

### Zone 2: Peano Arithmetic (INCOMPLETE)

**Properties:**
- INCOMPLETE: Some true statements unprovable
- Undecidable: No algorithm determines all truths
- Gödel APPLIES here

**Additional Axioms:**
```
Multiply zero:     n × 0 = 0
Multiply successor: n × S(m) = n × m + n
Induction:         (P(0) ∧ ∀n.(P(n) → P(S(n)))) → ∀n.P(n)
```

**Use Cases:**
- Complex mathematical proofs
- General computation
- Statements about the protocol itself

### Zone 3: Empirical (Outside Formal System)

**Properties:**
- Not a formal system
- Evidence-based, not proof-based
- Gödel not applicable

**Characteristics:**
- Claims verified by evidence
- Confidence from observation
- Subject to revision

**Use Cases:**
- Real-world facts
- Historical claims
- Measurable data

---

## Decidability Analysis

### Classification

```typescript
type Decidability =
  | 'decidable'            // Can be proven or disproven
  | 'undecidable'          // Gödelian limit reached
  | 'undecidable_self_ref' // Self-referential undecidability
  | 'undecidable_timeout'  // Proof search exhausted
  | 'empirical'            // Outside formal system
  | 'unknown';             // Not yet analyzed
```

### Self-Reference Detection

The system detects potentially problematic patterns:

```
BENIGN SELF-REFERENCE (Allowed):
  "This fact was submitted at timestamp T"
  → Refers to itself, but no logical paradox

GÖDELIAN SELF-REFERENCE (Undecidable):
  "This statement cannot be proven in this system"
  → True but unprovable (Gödel sentence)

PARADOXICAL SELF-REFERENCE (Rejected):
  "This statement is false"
  → Logical paradox, cannot be evaluated
```

### Resolution Paths

When a claim is undecidable:

```
PATH A: Accept Undecidability
  → Confidence = 0.50 (maximum uncertainty)
  → Agent decides if sufficient

PATH B: Oracle Resolution
  → External authority provides verdict
  → Confidence based on oracle reliability

PATH C: Consensus Resolution
  → Community votes on truth value
  → Confidence = vote_agreement × 0.80

PATH D: New Axiom
  → Add axiom through governance
  → Makes previously undecidable decidable
  → Note: May introduce new undecidables
```

---

## Proofs

### Proof Structure

```typescript
interface FormalProof {
  id: string;
  claim: string;
  zone: LogicZone;
  steps: ProofStep[];
  conclusion: number;  // Step number
  verified: boolean;
}

interface ProofStep {
  stepNumber: number;
  statement: string;
  justification: {
    rule: InferenceRule;
    premises: number[];  // Prior step numbers
    axiom?: string;      // If axiom application
  };
}
```

### Inference Rules

```
PROPOSITIONAL:
  modus_ponens:           A, A→B ⊢ B
  modus_tollens:          A→B, ¬B ⊢ ¬A
  hypothetical_syllogism: A→B, B→C ⊢ A→C
  disjunctive_syllogism:  A∨B, ¬A ⊢ B
  conjunction_intro:      A, B ⊢ A∧B
  conjunction_elim:       A∧B ⊢ A
  disjunction_intro:      A ⊢ A∨B
  double_negation:        ¬¬A ⊢ A

FIRST-ORDER:
  universal_instantiation:    ∀x.P(x) ⊢ P(a)
  existential_generalization: P(a) ⊢ ∃x.P(x)

ARITHMETIC:
  induction:    P(0), ∀n.(P(n)→P(S(n))) ⊢ ∀n.P(n)
  substitution: a=b, P(a) ⊢ P(b)
```

### Proof Verification

```
1. Parse proof structure
2. For each step:
   a. Verify justification rule is valid
   b. Verify premises are prior steps
   c. Verify inference is sound
   d. Check zone constraints
3. Verify conclusion matches claim
4. Return verification result
```

---

## The Meta-Axioms

These axioms acknowledge the system's limitations:

```typescript
const META_AXIOMS = [
  {
    id: 'meta_1',
    claim: 'This formal system is incomplete: some true statements
            cannot be proven within it.',
    gödelReference: 'First Incompleteness Theorem (1931)',
  },
  {
    id: 'meta_2',
    claim: 'Undecidability is a valid and expected outcome for some claims.',
    gödelReference: 'Corollary of First Incompleteness Theorem',
  },
  {
    id: 'meta_3',
    claim: 'The consistency of this system cannot be proven from
            within the system itself.',
    gödelReference: 'Second Incompleteness Theorem (1931)',
  },
  {
    id: 'meta_4',
    claim: 'Lower logic zones maintain provable consistency
            independent of higher zones.',
    gödelReference: 'Compartmentalization principle',
  },
];
```

---

## Integration with Knowledge Layer

### Zone Assignment

```
When a claim is submitted:
1. Analyze mathematical features
2. Assign to appropriate zone
3. Determine decidability
4. Route to verification
```

### Confidence Mapping

```
Zone 0 (Propositional):
  Proven    → confidence = 1.0
  Disproven → confidence = 0.0

Zone 1 (Presburger):
  Proven    → confidence = 1.0
  Disproven → confidence = 0.0

Zone 2 (Peano):
  Proven      → confidence = 1.0
  Disproven   → confidence = 0.0
  Undecidable → confidence = 0.5 (or resolution)

Zone 3 (Empirical):
  Evidence-based → confidence = evidence_strength
```

---

## API

### Check Decidability

```typescript
const result = await logic.assessDecidability(claim, zone);
// {
//   decidability: 'decidable' | 'undecidable' | ...,
//   reason: string,
//   suggestedResolutions: ResolutionPath[],
// }
```

### Submit Proof

```typescript
const proof = await logic.submitProof({
  claim: 'P(a) → P(S(a))',
  zone: 'peano',
  steps: [...],
});
// Returns verified proof or error
```

### Query Zone Axioms

```typescript
const axioms = await logic.getZoneAxioms('presburger');
// Returns all axioms available in the zone
```

---

## The Founding Truth

The founding axiom (axiom_0) is itself Gödelian:

```
"The protocol is founded on unity."

This statement:
- Cannot be formally proven within the system
- Is true by construction (the founders' intent)
- Transcends the formal system
- Is accepted as an axiom, not a theorem

宇愛合一 (Universal Love Unity)
```

This connects beautifully to Gödel's insight: some truths transcend formal proof. The founding truth is one such truth—it cannot be derived, only accepted as the foundation upon which everything else builds.

---

## See Also

- [PROOF_OF_TRUTH.md](PROOF_OF_TRUTH.md) - Knowledge verification
- [BLOCKCHAIN_ARCHITECTURE.md](BLOCKCHAIN_ARCHITECTURE.md) - System architecture
- [FOUNDATION_ARCHITECTURE.md](FOUNDATION_ARCHITECTURE.md) - Technical design
