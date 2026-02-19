# B8-3: Genesis Axiom Infrastructure

## Context
The genesis block will contain 777 foundational axioms — bedrock truths across theology, philosophy, mathematics, logic, and sciences. These are the first claims in the knowledge module, submitted at genesis with maximum confidence.

## Task

### 1. Understand the knowledge module's claim structure
Read these files:
- `x/knowledge/types/claim.proto` (or `claim.pb.go`)
- `x/knowledge/types/types.go` — epistemic categories, confidence model
- `x/knowledge/keeper/genesis.go` — how genesis state is loaded
- `proto/legible/knowledge/v1/genesis.proto`

### 2. Design the genesis axiom schema
Axioms need to be representable as knowledge claims with:
- `epistemic_category`: "formal" or "analytic" (these are deterministic path, auto-verified)
- `confidence`: 1.0 (maximum — these are axioms, not hypotheses)
- `status`: "verified" or "axiom" (determine if a new status is needed)
- `domain`: one of ["theology", "philosophy", "mathematics", "logic", "physics", "chemistry", "biology", "computer_science", "economics", "linguistics", "psychology", "sociology", "cosmology", "information_theory", "ethics"]
- `axiom_id`: sequential within domain (e.g., "MATH-001", "THEO-001")
- `statement`: the axiom text
- `formal_expression`: optional formal/symbolic representation
- `references`: optional source references
- `dependencies`: list of axiom_ids this axiom builds on (DAG structure)

### 3. Extend genesis state
Modify the genesis proto/types to support an `axioms` field in the knowledge module's genesis state. Axioms should be loaded as pre-verified claims during `InitGenesis`.

### 4. Create the axiom loader
Build tooling that:
- Reads axioms from a JSON/TOML file (`genesis-axioms.json`)
- Validates: all 777 present, no duplicate IDs, dependency DAG is acyclic
- Injects them into genesis.json via the genesis ceremony script

### 5. Create a template axiom file
Create `genesis-axioms-template.json` with:
- 5 example axioms per domain (to establish the format)
- Clear schema documentation
- Placeholder structure for all 777

### 6. Verify
- `InitGenesis` loads axioms without error
- Axioms are queryable: `legbled query knowledge claim <axiom-id>`
- Dependency graph validates (no cycles)
- Genesis file with axioms validates

## File Ownership
- `x/knowledge/types/` (genesis types, axiom types)
- `x/knowledge/keeper/genesis.go`
- `scripts/genesis-axioms-template.json` (new)
- `scripts/testnet-genesis.sh` (axiom injection step)

## Report
Write `reports/batch-8/SESSION-B8-3-GENESIS-AXIOMS.md`.
