# Genesis Axioms

The Legible Money chain launches with **772 pre-verified axioms** across 15 knowledge domains, seeded directly into `genesis.json` at block 0. These form the foundational knowledge graph that validators extend through Proof of Truth consensus.

## Domains (15)

| Prefix | Domain | Count | Description |
|--------|--------|-------|-------------|
| MATH | Mathematics | ~100 | Foundations, algebra, analysis, topology, probability |
| LOGIC | Logic | ~50 | Propositional, predicate, modal, Godel incompleteness |
| PHYS | Physics | ~80 | Classical mechanics, EM, QM, relativity, cosmology |
| CHEM | Chemistry | ~50 | Atomic structure, bonding, thermodynamics, organic |
| BIO | Biology | ~50 | Cell biology, genetics, evolution, ecology |
| CS | Computer Science | ~60 | Computability, complexity, algorithms, information |
| INFO | Information Theory | ~40 | Shannon entropy, coding, channels, compression |
| ECON | Economics | ~50 | Microeconomics, game theory, markets, welfare |
| LING | Linguistics | ~40 | Phonology, syntax, semantics, pragmatics |
| PHIL | Philosophy | ~50 | Epistemology, metaphysics, ethics, mind |
| PSYCH | Psychology | ~40 | Cognition, perception, development, social |
| ETHIC | Ethics | ~40 | Normative theory, metaethics, applied ethics |
| COSM | Cosmology | ~30 | Big Bang, dark matter/energy, CMB, structure |
| THEO | Theology | ~30 | Philosophical theology, comparative religion |
| AGRT | Agreement Theory | ~90 | Social contracts, coordination, trust, institutions |

## Axiom Types

- **axiom** — Self-evident foundational truth (highest stratum)
- **definition** — Formal definition establishing terminology
- **empirical_axiom** — Well-established empirical fact
- **regime_declaration** — Declares a physical/mathematical regime
- **derived_claim** — Derived from other axioms (confidence clamped to weakest dependency)
- **measurement_fact** — Established measurement or observation
- **meta** — Meta-level statement about knowledge itself

## Epistemic Categories

- **analytic** — True by definition or logical necessity
- **formal** — True within a formal system
- **empirical** — Established by observation/experiment

## Loading Genesis Axioms

### Quick Start

```bash
bash scripts/genesis-777.sh
```

This runs the full pipeline: build → parse → validate → inject → boot.

### Manual Pipeline

```bash
# 1. Parse markdown → JSON
go run tools/axiom-parser/main.go -dir genesis-axioms/ -o axioms.json

# 2. Validate axiom DAG (acyclicity, stratum, confidence, domains)
go run tools/axiom-loader/main.go validate axioms.json

# 3. Inject into genesis.json
go run tools/axiom-loader/main.go inject axioms.json path/to/genesis.json

# 4. Validate genesis
legbled validate-genesis --home ~/.legbled

# 5. Start chain
legbled start --home ~/.legbled
```

### Pipeline Details

**Parser** (`tools/axiom-parser/main.go`):
- Reads structured markdown from `genesis-axioms/*.md`
- Extracts axiom ID, statement, type, domain, confidence, dependencies, regime
- Normalizes compound claim types (e.g., `measurement_fact (asymmetry) + derived_claim`)
- Auto-clamps derived claim confidence to weakest dependency
- Outputs topologically-sorted JSON array

**Validator** (`tools/axiom-loader/main.go validate`):
- Validates axiom ID format (`PREFIX-NNN[suffix]`)
- Checks DAG acyclicity (no circular dependencies)
- Verifies stratum consistency (axioms cannot depend on derived claims)
- Validates derived confidence constraints
- Checks domain assignments against known domains

**Injector** (`tools/axiom-loader/main.go inject`):
- Converts axioms to `Fact` entries (confidence float → BPS, status=verified, submitter=genesis)
- Creates `FactReference` edges from dependency declarations (type=supports)
- Merges into existing genesis.json knowledge module state
- Checks for ID collisions with existing facts

### Validation Rules

1. **DAG Acyclicity**: No axiom may transitively depend on itself
2. **Stratum Consistency**: Axioms (type=axiom) must not depend on derived_claims or measurement_facts. Definitions may reference derived results.
3. **Derived Confidence**: Any derived_claim's confidence must be ≤ min(dependency confidences)
4. **Domain Validity**: All domain assignments must match a registered domain
5. **ID Format**: Must match `^[A-Z]+-\d{3}[a-z0-9-]*$` (e.g., MATH-001, AGRT-004a)

## Genesis Size

With 772 axioms and ~1282 fact references, the genesis.json is approximately **728KB** — well within practical limits.
