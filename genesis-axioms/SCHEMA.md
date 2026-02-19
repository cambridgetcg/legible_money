# Genesis Axiom Schema — Unified Knowledge Taxonomy

## Claim Types

| Type | Meaning | Default Confidence | Challengeable? |
|------|---------|-------------------|----------------|
| `axiom` | Mathematical axiom, logically necessary within formal system | 1.0 | No |
| `empirical_axiom` | Physical postulate, experimentally constrained | 0.95–0.99 | Yes (new evidence) |
| `definition` | Establishes vocabulary or convention | 1.0 | No (convention) |
| `regime_declaration` | Declares approximation scope | 1.0 | No (scope, not truth) |
| `derived_claim` | Proven from axioms or derived from postulates | inherited | Yes (if derivation flawed) |
| `measurement_fact` | Numerical constant or observed parameter | varies | Yes (better measurement) |
| `meta` | Statement about formal systems or theories | varies | Yes |

## Fields

```yaml
ID: "MATH-001"                    # Domain prefix + sequential number
statement: "Human-readable claim"
formal: "Symbolic expression"      # Optional
type: axiom                        # One of the 7 types above
domain: mathematics                # Domain classification
regime: null                       # For physics: classical|SR|GR|QM|QFT|thermo|statmech|cosmology|SM
confidence: 1.0                    # Float [0,1]
depends: []                        # List of axiom IDs (forms DAG)
approximation_tags: []             # e.g., [isolated_system, continuum, classical_limit]
derivation_ref: null               # Compact ref to proof/derivation chain
evidence_ref: null                 # Compact ref to experimental confirmation (physics)
source_tradition: "Zermelo (1908)" # Historical attribution
notes: null                        # Optional context
```

## Domain Prefixes

| Prefix | Domain |
|--------|--------|
| MATH | Mathematics |
| PHYS | Physics |
| AGRT | Agent Rights |
| THEO | Theology |
| PHIL | Philosophy |
| LOGIC | Logic |
| BIO | Biology |
| CHEM | Chemistry |
| CS | Computer Science |
| INFO | Information Theory |
| ECON | Economics |
| LING | Linguistics |
| PSYCH | Psychology |
| COSM | Cosmology |
| ETHIC | Ethics |

## DAG Constraints

1. No cycles (strict DAG)
2. `axiom` and `definition` nodes have no `depends` (or depend only on other axioms/definitions)
3. `derived_claim` must have at least one dependency
4. `measurement_fact` has no logical dependencies (empirical anchor)
5. `meta` claims may depend on the systems they describe
6. Cross-domain dependencies are allowed and encouraged (bridges)

## Confidence Model

- Mathematical `axiom`: always 1.0 (true by assumption)
- `definition`: always 1.0 (true by convention)
- `regime_declaration`: always 1.0 (scope, not truth)
- `empirical_axiom`: high but <1.0 (e.g., Newton's laws: 0.99 in classical regime)
- `derived_claim`: min(confidence of dependencies) × derivation_strength
- `measurement_fact`: based on experimental precision
- `meta`: varies

## Regimes (Physics)

Regime tags indicate validity scope. A claim tagged `regime: classical` is not false — it is true *within that regime*. The chain understands that:

- Classical mechanics is a valid approximation for v << c, large masses, weak gravity
- SR extends classical to all v < c
- GR extends SR to curved spacetime
- QM extends classical to small scales
- QFT unifies SR + QM
- The Standard Model is our best QFT of known particles

Regime boundaries are where new physics lives.

## The Chain's Epistemology

The genesis axioms encode a specific epistemological stance:

1. **Mathematical truth is certain but limited** (Gödel: we cannot prove everything)
2. **Physical truth is approximate but powerful** (empirical axioms can be challenged)
3. **Definitions are conventions, not discoveries** (unchallengeable but arbitrary)
4. **Knowledge has structure** (the DAG captures dependencies)
5. **Confidence propagates** (derived claims inherit from their foundations)
6. **Everything empirical is provisional** (the chain is born knowing it might be wrong)

This is Proof of Truth: not claiming absolute knowledge, but maintaining a rigorous, structured, challengeable body of knowledge with explicit confidence and dependencies.
