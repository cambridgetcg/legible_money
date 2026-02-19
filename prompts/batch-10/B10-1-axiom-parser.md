# B10-1 — Markdown-to-JSON Axiom Parser

## Context
We have 15 domain files in `genesis-axioms/` totaling 777 items written in a
structured markdown/code-block format. The existing `tools/axiom-loader/main.go`
expects a JSON array of `GenesisAxiom` structs. We need a parser that reads all
15 `.md` files and emits a single `axioms-777.json` ready for the loader.

## Current State
- `scripts/genesis-axioms-template.json` has 75 sample items (stale)
- `x/knowledge/types/axiom.go` defines `GenesisAxiom` struct with fields:
  `axiom_id`, `statement`, `formal_expression`, `type`, `domain`,
  `epistemic_category`, `regime`, `confidence`, `approximation_tags`,
  `derivation_ref`, `evidence_ref`, `dependencies`, `source_tradition`,
  `references`, `notes`, `tier_required`
- Domain files use this format inside code blocks:
  ```
  MATH-001 (Axiom, Peano Successor)
    Statement text here spanning multiple lines.
    type: axiom
    domain: mathematics
    confidence: 1.0
    depends: [MATH-002, PHYS-001]
    source_tradition: "Peano (1889)"
    notes: Optional context.
  ```

## Task

### 1. Build `tools/axiom-parser/main.go`
A Go tool that:
- Reads all `.md` files from a directory (default: `genesis-axioms/`)
- Parses the code-block format extracting all fields
- Maps each item to a `GenesisAxiom` JSON struct
- Handles the field mapping:
  - `axiom_id` ← the `PREFIX-NNN` at start of block
  - `statement` ← all text lines before the first `key: value` field
  - `type` ← `type:` field (already matches `ClaimType` values)
  - `domain` ← `domain:` field
  - `confidence` ← `confidence:` field (float)
  - `depends` ← `depends: [...]` field → `dependencies` array
  - `source_tradition` ← `source_tradition:` field (strip quotes)
  - `notes` ← `notes:` field (can be multi-line until next field or end of block)
  - `formal` or `formal_expression` ← `formal:` field if present
  - `regime` ← `regime:` field if present
  - `approximation_tags` ← `approximation_tags:` field if present
  - `evidence_ref` ← `evidence_ref:` field if present
  - `derivation_ref` ← `derivation_ref:` field if present
- Auto-maps `epistemic_category` from `type`:
  - `axiom` → `analytic`
  - `definition` → `analytic`
  - `empirical_axiom` → `empirical`
  - `regime_declaration` → `formal`
  - `derived_claim` → `formal`
  - `measurement_fact` → `empirical`
  - `meta` → `empirical`
- Outputs sorted by axiom_id (AGRT before BIO before CHEM...)
- Writes to stdout or `-o <file>`

### 2. Usage
```bash
go run tools/axiom-parser/main.go -dir genesis-axioms/ -o scripts/axioms-777.json
```

### 3. Validation
After generating JSON:
```bash
go run tools/axiom-loader/main.go validate scripts/axioms-777.json
```
This will fail until B10-2 fixes the prefix map. That's OK — verify the parser
output has exactly 777 items and the JSON is well-formed.

### 4. Edge Cases
- Some items span many lines (AGRT items can be 10+ lines of statement)
- `notes:` can be multi-line (ends at next recognized field or block boundary)
- `depends: []` means empty array (not omitted)
- Some items have `formal:` with symbolic expressions
- Code blocks are delimited by triple backticks
- Items are separated by blank lines within code blocks
- The `(Type, Name)` in the header line is informational — use the `type:` field

### 5. Tests
Write `tools/axiom-parser/main_test.go` with:
- Test parsing a single item
- Test parsing multiple items from one block
- Test multi-line statement extraction
- Test multi-line notes extraction
- Test empty depends
- Test formal expression extraction
- Test regime and approximation_tags extraction
- Integration test: parse actual `genesis-axioms/MATH.md` and verify count = 111

## Output
- `tools/axiom-parser/main.go`
- `tools/axiom-parser/main_test.go`
- `scripts/axioms-777.json` (generated output)

## Commit
```
feat(tools): add axiom-parser — MD-to-JSON converter for 777 genesis axioms

Parses structured markdown domain files in genesis-axioms/ and emits
a JSON array compatible with axiom-loader. Handles all 7 claim types,
multi-line statements, dependencies, and epistemic category auto-mapping.
```
