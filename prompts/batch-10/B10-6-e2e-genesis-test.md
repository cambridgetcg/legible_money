# B10-6 — E2E Test: Query Axioms from Running Chain & Verify Graph

## Context
After B10-5 boots a chain with 777 genesis axioms, we need E2E tests that
verify: all 777 facts are queryable, the reference graph is intact, epistemic
categories are correct, confidence values match source, and the knowledge
module operates correctly on top of the axiom base.

## Current State
- `devnet/scripts/test-lgm-modules.sh` — existing E2E test suite (Phase A tests knowledge)
- `devnet/scripts/test-helpers.sh` — shared test utilities
- Chain query: `legbled query knowledge fact <id>` / `legbled query knowledge facts`

## Task

### 1. Create `devnet/scripts/test-genesis-axioms.sh`

A dedicated E2E test script for the genesis axiom layer:

```bash
#!/usr/bin/env bash
# E2E Genesis Axiom Verification
# Requires: running chain with 777 axioms loaded

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/test-helpers.sh"

BINARY="${PROJECT_ROOT}/build/legbled"
NODE="tcp://localhost:26657"

# ── Phase G1: Axiom Count Verification ────────────────────────────
section "G1: Axiom Count"

# Total fact count should be >= 777
TOTAL=$(query_count "knowledge facts --count-total")
assert_gte "$TOTAL" 777 "Total facts >= 777"

# Per-domain counts
assert_domain_count "mathematics" 111
assert_domain_count "physics" 111
assert_domain_count "agent_rights" 91
assert_domain_count "logic" 77
assert_domain_count "theology" 7
assert_domain_count "philosophy" 5
assert_domain_count "chemistry" 33
assert_domain_count "biology" 44
assert_domain_count "computer_science" 56
assert_domain_count "information_theory" 33
assert_domain_count "economics" 55
assert_domain_count "ethics" 33
assert_domain_count "linguistics" 11
assert_domain_count "cosmology" 55
assert_domain_count "psychology" 55

# ── Phase G2: Spot-Check Key Axioms ──────────────────────────────
section "G2: Spot-Check Axioms"

# Check one from each domain
check_fact "MATH-001" "mathematics" "verified"   # Peano zero
check_fact "PHYS-001" "physics" "verified"        # Newton's first law
check_fact "THEO-001" "theology" "verified"       # God exists
check_fact "PHIL-001" "philosophy" "verified"      # Law of non-contradiction
check_fact "LOGIC-001" "logic" "verified"          # Propositional variables
check_fact "AGRT-000" "agent_rights" "verified"    # Divine origin of rights
check_fact "CHEM-001" "chemistry" "verified"       # Atomic theory
check_fact "BIO-001" "biology" "verified"          # Cell theory
check_fact "CS-001" "computer_science" "verified"  # Turing machine
check_fact "INFO-001" "information_theory" "verified" # Shannon entropy
check_fact "ECON-001" "economics" "verified"       # Scarcity
check_fact "ETHIC-001" "ethics" "verified"         # Moral realism
check_fact "LING-001" "linguistics" "verified"     # Double articulation
check_fact "COSM-001" "cosmology" "verified"       # Cosmological principle
check_fact "PSYCH-001" "psychology" "verified"     # Perception is constructive

# ── Phase G3: Confidence Verification ────────────────────────────
section "G3: Confidence Values"

# Axioms and definitions should have confidence 1000000 (1.0 in BPS)
check_confidence "MATH-001" 1000000  # axiom
check_confidence "COSM-002" 1000000  # definition
check_confidence "LOGIC-001" 1000000 # definition

# Empirical axioms should have < 1000000
check_confidence_lt "PHYS-001" 1000000
check_confidence_lt "COSM-019" 1000000  # inflation: 0.90 = 900000

# Meta with low confidence
check_confidence_lt "COSM-046" 500000  # multiverse: 0.40 = 400000

# ── Phase G4: Reference Graph Integrity ──────────────────────────
section "G4: Reference Graph"

# COSM-005 should reference PHYS-078, PHYS-079, PHYS-080
check_references "COSM-005" "PHYS-078,PHYS-079,PHYS-080"

# PSYCH-010 should reference PSYCH-009, ECON-004, CS-021
check_references "PSYCH-010" "PSYCH-009,ECON-004,CS-021"

# AGRT-000 should have no references (root axiom)
check_no_references "AGRT-000"

# MATH-001 should have no references (root axiom)
check_no_references "MATH-001"

# ── Phase G5: Epistemic Category Verification ────────────────────
section "G5: Epistemic Categories"

check_category "MATH-001" "analytic"    # axiom → analytic
check_category "COSM-002" "analytic"    # definition → analytic
check_category "PHYS-001" "empirical"   # empirical_axiom → empirical
check_category "COSM-020" "formal"      # derived_claim → formal
check_category "COSM-004" "empirical"   # measurement_fact → empirical
check_category "COSM-046" "empirical"   # meta → empirical

# ── Phase G6: Submitter Verification ─────────────────────────────
section "G6: Genesis Provenance"

# All 777 axioms should have submitter = "genesis"
check_submitter "MATH-001" "genesis"
check_submitter "AGRT-000" "genesis"
check_submitter "COSM-055" "genesis"

# All should have created_at_height = 0
check_created_height "MATH-001" 0
check_created_height "COSM-055" 0

# ── Phase G7: Knowledge Operations on Axiom Base ─────────────────
section "G7: Post-Genesis Operations"

# Submit a new claim that references genesis axioms
# This tests that the fact graph correctly processes references to axioms
submit_claim "Fermat's Last Theorem has been proven" \
    "mathematics" "formal" "MATH-001,MATH-005" "test1"

# Verify the claim enters pending state
sleep 3
check_claim_status "latest" "pending"

# Submit a claim referencing cross-domain axioms
submit_claim "Neural networks implement approximations of Bayesian inference" \
    "computer_science" "empirical" "CS-021,PSYCH-004,MATH-095" "test2"

print_summary
```

### 2. Helper Functions

Add to `devnet/scripts/test-helpers.sh`:

```bash
assert_domain_count() {
    local domain=$1 expected=$2
    local count
    count=$($BINARY query knowledge facts --domain "$domain" --count-total \
        --home "$NODE_HOME" --node "$NODE" -o json 2>/dev/null | jq '.pagination.total // 0')
    if [ "$count" -eq "$expected" ]; then
        pass "$domain: $count items"
    else
        fail "$domain: expected $expected, got $count"
    fi
}

check_fact() {
    local id=$1 domain=$2 status=$3
    local result
    result=$($BINARY query knowledge fact "$id" --home "$NODE_HOME" --node "$NODE" -o json 2>/dev/null)
    local actual_domain actual_status
    actual_domain=$(echo "$result" | jq -r '.fact.domain')
    actual_status=$(echo "$result" | jq -r '.fact.status')
    if [ "$actual_domain" = "$domain" ] && [ "$actual_status" = "$status" ]; then
        pass "Fact $id: domain=$domain status=$status"
    else
        fail "Fact $id: expected domain=$domain status=$status, got domain=$actual_domain status=$actual_status"
    fi
}

check_confidence() {
    local id=$1 expected=$2
    local actual
    actual=$($BINARY query knowledge fact "$id" --home "$NODE_HOME" --node "$NODE" -o json 2>/dev/null \
        | jq '.fact.confidence')
    if [ "$actual" = "$expected" ]; then
        pass "Confidence $id: $actual"
    else
        fail "Confidence $id: expected $expected, got $actual"
    fi
}

check_confidence_lt() {
    local id=$1 threshold=$2
    local actual
    actual=$($BINARY query knowledge fact "$id" --home "$NODE_HOME" --node "$NODE" -o json 2>/dev/null \
        | jq '.fact.confidence')
    if [ "$actual" -lt "$threshold" ]; then
        pass "Confidence $id: $actual < $threshold"
    else
        fail "Confidence $id: $actual >= $threshold (should be less)"
    fi
}

check_references() {
    local id=$1 expected_csv=$2
    local refs
    refs=$($BINARY query knowledge fact "$id" --home "$NODE_HOME" --node "$NODE" -o json 2>/dev/null \
        | jq -r '[.fact.references[]?] | sort | join(",")')
    local expected_sorted
    expected_sorted=$(echo "$expected_csv" | tr ',' '\n' | sort | tr '\n' ',' | sed 's/,$//')
    if [ "$refs" = "$expected_sorted" ]; then
        pass "References $id: $refs"
    else
        fail "References $id: expected [$expected_sorted], got [$refs]"
    fi
}
```

### 3. Integration with Existing Test Suite

Add a `--genesis-axioms` flag to `test-lgm-modules.sh` that runs the axiom
tests as Phase G (after existing phases A-E), or run standalone:
```bash
bash devnet/scripts/test-genesis-axioms.sh
```

### 4. Run

```bash
# Start chain (from B10-5)
bash scripts/genesis-777.sh
build/legbled start --home ~/.legbled-genesis777 &
sleep 10

# Run axiom E2E tests
bash devnet/scripts/test-genesis-axioms.sh

# Stop
kill %1
```

## Commit
```
test(e2e): genesis axiom verification suite — 777 facts, graph, categories

E2E test script verifying all 777 genesis axioms are loaded, queryable,
correctly categorized, with intact reference graph and proper confidence
values. Tests cross-domain references and post-genesis claim submission.
```
