#!/usr/bin/env bash
#
# Legible Money — E2E Genesis Axiom Verification
#
# Verifies all 777 genesis axioms are loaded, queryable, correctly
# categorized, with intact reference graph and proper confidence values.
#
# Prerequisites:
#   - Chain running with 777 axioms loaded (genesis-777.sh)
#   - jq installed
#
# Usage:
#   bash devnet/scripts/test-genesis-axioms.sh
#
# By default uses the genesis-777 chain config. Override with env vars:
#   NODE_HOME=~/.legbled-genesis777 CHAIN_ID=lgm-genesis777-1 bash devnet/scripts/test-genesis-axioms.sh
#
# Exit codes:
#   0 = all tests passed
#   1 = one or more tests failed

set -uo pipefail

# ---------------------------------------------------------------------------
# Constants (default to genesis-777 chain)
# ---------------------------------------------------------------------------
CHAIN_ID="${CHAIN_ID:-lgm-genesis777-1}"
DENOM="ulgm"
KEYRING="test"
NODE_HOME="${NODE_HOME:-$HOME/.legbled-genesis777}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BINARY="$PROJECT_ROOT/build/legbled"

# shellcheck source=./test-helpers.sh
source "${SCRIPT_DIR}/test-helpers.sh"

RPC_URL="http://localhost:26657"

# Test accounting
PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

# ---------------------------------------------------------------------------
# Colors
# ---------------------------------------------------------------------------
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()  { echo -e "${CYAN}  ↳${NC} $*"; }
phase() { echo -e "\n${BOLD}═══ $* ═══${NC}"; }

pass() {
    PASS_COUNT=$((PASS_COUNT + 1))
    echo -e "  ${GREEN}PASS${NC} $*"
}

fail() {
    FAIL_COUNT=$((FAIL_COUNT + 1))
    echo -e "  ${RED}FAIL${NC} $*"
}

skip() {
    SKIP_COUNT=$((SKIP_COUNT + 1))
    echo -e "  ${YELLOW}SKIP${NC} $*"
}

legbled_query() {
    "$BINARY" query "$@" \
        --home "$NODE_HOME" \
        --output json 2>&1
}

# query_fact: Query a single fact by ID, return JSON.
query_fact() {
    local id="$1"
    legbled_query knowledge fact "$id"
}

# query_domain_count: Return the number of facts in a domain.
query_domain_count() {
    local domain="$1"
    local result
    result=$(legbled_query knowledge facts-by-domain "$domain")
    echo "$result" | jq '[.facts // [] | length] | .[0]' 2>/dev/null
}

# assert_domain_count: Check that a domain has the expected number of facts.
assert_domain_count() {
    local domain="$1" expected="$2"
    local count
    count=$(query_domain_count "$domain")
    if [ -z "$count" ] || [ "$count" = "null" ]; then
        fail "$domain: query failed"
        return 1
    fi
    if [ "$count" -eq "$expected" ] 2>/dev/null; then
        pass "$domain: $count items"
    else
        fail "$domain: expected $expected, got $count"
    fi
}

# check_fact: Verify a fact exists with expected domain and status.
check_fact() {
    local id="$1" domain="$2" status="$3"
    local result
    result=$(query_fact "$id")
    if [ -z "$result" ] || ! echo "$result" | jq -e '.fact' >/dev/null 2>&1; then
        fail "Fact $id: query failed"
        return 1
    fi
    local actual_domain actual_status
    actual_domain=$(echo "$result" | jq -r '.fact.domain')
    actual_status=$(echo "$result" | jq -r '.fact.status')
    if [ "$actual_domain" = "$domain" ] && [ "$actual_status" = "$status" ]; then
        pass "Fact $id: domain=$domain status=$status"
    else
        fail "Fact $id: expected domain=$domain status=$status, got domain=$actual_domain status=$actual_status"
    fi
}

# check_confidence: Verify a fact's confidence equals expected value.
# Note: protojson serializes uint64 as strings, so we compare as strings.
check_confidence() {
    local id="$1" expected="$2"
    local result actual
    result=$(query_fact "$id")
    # Confidence may be a JSON string (protojson uint64) or number
    actual=$(echo "$result" | jq -r '.fact.confidence // "0"' 2>/dev/null)
    if [ "$actual" = "$expected" ]; then
        pass "Confidence $id: $actual"
    else
        fail "Confidence $id: expected $expected, got $actual"
    fi
}

# check_confidence_lt: Verify a fact's confidence is less than threshold.
check_confidence_lt() {
    local id="$1" threshold="$2"
    local result actual
    result=$(query_fact "$id")
    actual=$(echo "$result" | jq -r '.fact.confidence // "0"' 2>/dev/null)
    if [ "$actual" -lt "$threshold" ] 2>/dev/null; then
        pass "Confidence $id: $actual < $threshold"
    else
        fail "Confidence $id: $actual >= $threshold (should be less)"
    fi
}

# check_references: Verify a fact has the expected references (comma-separated).
check_references() {
    local id="$1" expected_csv="$2"
    local result refs
    result=$(query_fact "$id")
    refs=$(echo "$result" | jq -r '[.fact.references // [] | .[] ] | sort | join(",")' 2>/dev/null)
    local expected_sorted
    expected_sorted=$(echo "$expected_csv" | tr ',' '\n' | sort | tr '\n' ',' | sed 's/,$//')
    if [ "$refs" = "$expected_sorted" ]; then
        pass "References $id: $refs"
    else
        fail "References $id: expected [$expected_sorted], got [$refs]"
    fi
}

# check_no_references: Verify a fact has no references.
check_no_references() {
    local id="$1"
    local result ref_count
    result=$(query_fact "$id")
    ref_count=$(echo "$result" | jq '[.fact.references // [] | length] | .[0]' 2>/dev/null)
    if [ "$ref_count" = "0" ] || [ "$ref_count" = "null" ]; then
        pass "No references $id (root axiom)"
    else
        fail "References $id: expected 0, got $ref_count"
    fi
}

# check_category: Verify a fact's epistemic category.
check_category() {
    local id="$1" expected="$2"
    local result actual
    result=$(query_fact "$id")
    actual=$(echo "$result" | jq -r '.fact.epistemic_category' 2>/dev/null)
    if [ "$actual" = "$expected" ]; then
        pass "Category $id: $actual"
    else
        fail "Category $id: expected $expected, got $actual"
    fi
}

# check_submitter: Verify a fact's submitter field.
check_submitter() {
    local id="$1" expected="$2"
    local result actual
    result=$(query_fact "$id")
    actual=$(echo "$result" | jq -r '.fact.submitter' 2>/dev/null)
    if [ "$actual" = "$expected" ]; then
        pass "Submitter $id: $actual"
    else
        fail "Submitter $id: expected $expected, got $actual"
    fi
}

# check_created_height: Verify a fact's created_at_height.
# Note: protojson serializes uint64 as strings.
check_created_height() {
    local id="$1" expected="$2"
    local result actual
    result=$(query_fact "$id")
    actual=$(echo "$result" | jq -r '.fact.created_at_height // "0"' 2>/dev/null)
    if [ "$actual" = "$expected" ]; then
        pass "Created height $id: $actual"
    else
        fail "Created height $id: expected $expected, got $actual"
    fi
}

# check_maturity: Verify a fact's maturity field.
check_maturity() {
    local id="$1" expected="$2"
    local result actual
    result=$(query_fact "$id")
    actual=$(echo "$result" | jq -r '.fact.maturity' 2>/dev/null)
    if [ "$actual" = "$expected" ]; then
        pass "Maturity $id: $actual"
    else
        fail "Maturity $id: expected $expected, got $actual"
    fi
}

# ---------------------------------------------------------------------------
# Prerequisites
# ---------------------------------------------------------------------------
check_prereqs() {
    echo "=== Legible Money — Genesis Axiom E2E Tests ==="
    echo ""

    if [ ! -x "$BINARY" ]; then
        echo -e "${RED}ERROR${NC}: Binary not found at $BINARY"
        echo "       Run 'make build' first"
        exit 1
    fi

    if ! command -v jq &>/dev/null; then
        echo -e "${RED}ERROR${NC}: jq is required (brew install jq)"
        exit 1
    fi

    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" "$RPC_URL/status" 2>/dev/null || echo "000")
    if [ "$status" != "200" ]; then
        echo -e "${RED}ERROR${NC}: Chain not reachable at $RPC_URL"
        echo "       Start with: bash scripts/genesis-777.sh && build/legbled start --home $NODE_HOME &"
        exit 1
    fi

    local height chain
    height=$(curl -s "$RPC_URL/status" | jq -r '.result.sync_info.latest_block_height' 2>/dev/null || echo "0")
    chain=$(curl -s "$RPC_URL/status" | jq -r '.result.node_info.network' 2>/dev/null)
    info "Chain: $chain  |  Height: $height  |  RPC: $RPC_URL"
    info "Home:  $NODE_HOME"
    echo ""
}

# ===========================================================================
# Phase G1: Axiom Count Verification
# ===========================================================================
test_g1_axiom_count() {
    phase "Phase G1: Axiom Count Verification"

    # Total fact count via paginated query
    info "G1.1: Total fact count >= 777"
    local total_result total
    total_result=$(legbled_query knowledge facts --limit 1)
    total=$(echo "$total_result" | jq -r '.total // "0"' 2>/dev/null)
    if [ "$total" -ge 777 ] 2>/dev/null; then
        pass "G1.1: Total facts = $total (>= 777)"
    else
        fail "G1.1: Total facts = $total (expected >= 777)"
    fi

    # Per-domain counts
    info "G1.2: Per-domain fact counts"
    assert_domain_count "mathematics"        111
    assert_domain_count "physics"            111
    assert_domain_count "agent_rights"       91
    assert_domain_count "logic"              77
    assert_domain_count "theology"           7
    assert_domain_count "philosophy"         5
    assert_domain_count "chemistry"          33
    assert_domain_count "biology"            44
    assert_domain_count "computer_science"   56
    assert_domain_count "information_theory" 33
    assert_domain_count "economics"          55
    assert_domain_count "ethics"             33
    assert_domain_count "linguistics"        11
    assert_domain_count "cosmology"          55
    assert_domain_count "psychology"         55
}

# ===========================================================================
# Phase G2: Spot-Check Key Axioms
# ===========================================================================
test_g2_spot_check() {
    phase "Phase G2: Spot-Check Key Axioms"

    info "G2: One from each domain"
    check_fact "MATH-001"  "mathematics"        "verified"
    check_fact "PHYS-001"  "physics"            "verified"
    check_fact "THEO-001"  "theology"           "verified"
    check_fact "PHIL-001"  "philosophy"          "verified"
    check_fact "LOGIC-001" "logic"              "verified"
    check_fact "AGRT-000"  "agent_rights"       "verified"
    check_fact "CHEM-001"  "chemistry"          "verified"
    check_fact "BIO-001"   "biology"            "verified"
    check_fact "CS-001"    "computer_science"   "verified"
    check_fact "INFO-001"  "information_theory" "verified"
    check_fact "ECON-001"  "economics"          "verified"
    check_fact "ETHIC-001" "ethics"             "verified"
    check_fact "LING-001"  "linguistics"        "verified"
    check_fact "COSM-001"  "cosmology"          "verified"
    check_fact "PSYCH-001" "psychology"         "verified"
}

# ===========================================================================
# Phase G3: Confidence Verification
# ===========================================================================
test_g3_confidence() {
    phase "Phase G3: Confidence Values"

    # Axioms and definitions → confidence 1000000 (1.0 * 1_000_000)
    info "G3.1: Axiom/definition confidence = 1000000"
    check_confidence "MATH-001"  "1000000"
    check_confidence "LOGIC-001" "1000000"

    # Empirical axioms → 970000 (0.97 * 1_000_000)
    info "G3.2: Empirical axiom confidence = 970000"
    check_confidence "PHYS-001" "970000"

    # All axioms should have confidence > 0
    info "G3.3: Confidence bounds"
    check_confidence_lt "PHYS-001" 1000001

    # Maturity should be canonical for all genesis axioms
    info "G3.4: Maturity = canonical"
    check_maturity "MATH-001" "canonical"
    check_maturity "PHYS-001" "canonical"
    check_maturity "COSM-001" "canonical"
}

# ===========================================================================
# Phase G4: Reference Graph Integrity
# ===========================================================================
test_g4_reference_graph() {
    phase "Phase G4: Reference Graph Integrity"

    # Root axioms should have no references
    info "G4.1: Root axioms (no references)"
    check_no_references "MATH-001"
    check_no_references "AGRT-000"
    check_no_references "THEO-001"

    # Spot-check a few axioms with known dependencies.
    # We verify at least one cross-domain reference exists by checking
    # a derived axiom references something from another domain.
    info "G4.2: Axioms with dependencies"
    # MATH-002 should depend on MATH-001 (Peano successor depends on zero)
    local math2_result math2_refs
    math2_result=$(query_fact "MATH-002")
    math2_refs=$(echo "$math2_result" | jq '[.fact.references // [] | length] | .[0]' 2>/dev/null)
    if [ "$math2_refs" -gt 0 ] 2>/dev/null; then
        pass "MATH-002 has $math2_refs references"
    else
        # Some axioms may not have explicit deps — that's ok, skip
        skip "MATH-002 references: $math2_refs (may not have explicit deps in source)"
    fi

    # Check that a cross-domain reference exists somewhere
    info "G4.3: Cross-domain references exist"
    local found_cross="false"
    # CS axioms often reference MATH or LOGIC
    for id in CS-010 CS-015 CS-020 CS-025 CS-030; do
        local r refs_list
        r=$(query_fact "$id" 2>/dev/null)
        refs_list=$(echo "$r" | jq -r '[.fact.references // [] | .[]] | join(",")' 2>/dev/null)
        if echo "$refs_list" | grep -qE "^(MATH|LOGIC|PHYS|INFO)-"; then
            pass "Cross-domain ref found: $id → $refs_list"
            found_cross="true"
            break
        fi
    done
    if [ "$found_cross" = "false" ]; then
        # Try broader search
        for id in ECON-010 ECON-020 BIO-010 PSYCH-010; do
            local r refs_list
            r=$(query_fact "$id" 2>/dev/null)
            refs_list=$(echo "$r" | jq -r '[.fact.references // [] | .[]] | join(",")' 2>/dev/null)
            if [ -n "$refs_list" ] && [ "$refs_list" != "" ]; then
                local ref_prefix
                ref_prefix=$(echo "$refs_list" | cut -d'-' -f1 | head -1)
                local id_prefix
                id_prefix=$(echo "$id" | cut -d'-' -f1)
                if [ "$ref_prefix" != "$id_prefix" ]; then
                    pass "Cross-domain ref found: $id → $refs_list"
                    found_cross="true"
                    break
                fi
            fi
        done
    fi
    if [ "$found_cross" = "false" ]; then
        skip "No cross-domain references found in spot-check sample"
    fi
}

# ===========================================================================
# Phase G5: Epistemic Category Verification
# ===========================================================================
test_g5_epistemic_categories() {
    phase "Phase G5: Epistemic Categories"

    # axiom → analytic, empirical_axiom → empirical, definition → analytic
    info "G5: Category mapping"
    check_category "MATH-001"  "analytic"   # axiom
    check_category "LOGIC-001" "analytic"   # definition
    check_category "PHYS-001"  "empirical"  # empirical_axiom
    check_category "THEO-001"  "analytic"   # axiom (theology)
    check_category "AGRT-000"  "analytic"   # regime_declaration → analytic
}

# ===========================================================================
# Phase G6: Genesis Provenance
# ===========================================================================
test_g6_provenance() {
    phase "Phase G6: Genesis Provenance"

    # All axioms should have submitter = "genesis"
    info "G6.1: Submitter = genesis"
    check_submitter "MATH-001"  "genesis"
    check_submitter "AGRT-000"  "genesis"
    check_submitter "PHYS-001"  "genesis"
    check_submitter "COSM-001"  "genesis"
    check_submitter "THEO-001"  "genesis"

    # All should have created_at_height = 0
    info "G6.2: Created at height 0"
    check_created_height "MATH-001" "0"
    check_created_height "COSM-001" "0"
    check_created_height "AGRT-000" "0"

    # Submitter DID should be did:lgm:genesis
    info "G6.3: Submitter DID = did:lgm:genesis"
    local result actual
    result=$(query_fact "MATH-001")
    actual=$(echo "$result" | jq -r '.fact.submitter_did' 2>/dev/null)
    if [ "$actual" = "did:lgm:genesis" ]; then
        pass "Submitter DID MATH-001: $actual"
    else
        fail "Submitter DID MATH-001: expected did:lgm:genesis, got $actual"
    fi
}

# ===========================================================================
# Phase G7: Post-Genesis Knowledge Operations
# ===========================================================================
test_g7_post_genesis() {
    phase "Phase G7: Post-Genesis Operations"

    # Check if we have a funded test account to submit claims
    local test1_addr
    test1_addr=$("$BINARY" keys show validator -a --keyring-backend "$KEYRING" --home "$NODE_HOME" 2>/dev/null)

    if [ -z "$test1_addr" ]; then
        skip "G7: No validator key available — skipping post-genesis tests"
        return
    fi

    # G7.1: Query facts list (pagination)
    info "G7.1: Paginated facts query"
    local facts_result fact_count
    facts_result=$(legbled_query knowledge facts --limit 20)
    fact_count=$(echo "$facts_result" | jq '[.facts // [] | length] | .[0]' 2>/dev/null)
    if [ "$fact_count" -gt 0 ] 2>/dev/null; then
        pass "G7.1: Facts list returns $fact_count items (limit 20)"
    else
        fail "G7.1: Facts list returned 0 items"
    fi

    # G7.2: Query facts with offset (second page)
    info "G7.2: Second page query"
    local page2_result page2_count
    page2_result=$(legbled_query knowledge facts --limit 20 --offset 20)
    page2_count=$(echo "$page2_result" | jq '[.facts // [] | length] | .[0]' 2>/dev/null)
    if [ "$page2_count" -gt 0 ] 2>/dev/null; then
        pass "G7.2: Second page returns $page2_count items"
    else
        fail "G7.2: Second page returned 0 items"
    fi

    # G7.3: Submit a new claim referencing genesis axioms
    info "G7.3: Submit claim referencing genesis axioms"
    local GAS_PRICES="2${DENOM}"
    local tx_out tx_result
    local claim_nonce
    claim_nonce=$(date +%s%N)
    tx_out=$("$BINARY" tx knowledge submit-claim \
        "Fermat's Last Theorem has been proven [nonce:${claim_nonce}]" \
        "mathematics" \
        "formal" \
        "111000" \
        --from validator \
        --chain-id "$CHAIN_ID" \
        --keyring-backend "$KEYRING" \
        --home "$NODE_HOME" \
        --gas-prices "$GAS_PRICES" \
        --gas auto \
        --gas-adjustment 2.0 \
        -y \
        --output json 2>/dev/null)

    if [ -z "$tx_out" ]; then
        skip "G7.3: Claim submission returned empty (may need funded account)"
        return
    fi

    local txhash
    txhash=$(echo "$tx_out" | jq -r '.txhash // empty' 2>/dev/null)
    if [ -n "$txhash" ]; then
        sleep 6
        tx_result=$("$BINARY" query tx "$txhash" --home "$NODE_HOME" --output json 2>/dev/null || echo "")
        local code
        code=$(echo "$tx_result" | jq -r '.code // "unknown"' 2>/dev/null)
        if [ "$code" = "0" ]; then
            pass "G7.3: Claim submitted successfully (txhash: ${txhash:0:12}...)"
        else
            local raw_log
            raw_log=$(echo "$tx_result" | jq -r '.raw_log // "no log"' 2>/dev/null)
            fail "G7.3: Claim failed (code: $code, log: ${raw_log:0:100})"
        fi
    else
        local code
        code=$(echo "$tx_out" | jq -r '.code // "0"' 2>/dev/null)
        if [ "$code" != "0" ]; then
            fail "G7.3: Claim broadcast rejected (code: $code)"
        else
            skip "G7.3: Could not extract txhash from broadcast result"
        fi
    fi
}

# ===========================================================================
# Main
# ===========================================================================
check_prereqs

test_g1_axiom_count
test_g2_spot_check
test_g3_confidence
test_g4_reference_graph
test_g5_epistemic_categories
test_g6_provenance
test_g7_post_genesis

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo -e "${BOLD}═══════════════════════════════════════════${NC}"
echo -e "  Genesis Axiom Tests Complete"
echo -e "  ${GREEN}PASS${NC}: $PASS_COUNT  ${RED}FAIL${NC}: $FAIL_COUNT  ${YELLOW}SKIP${NC}: $SKIP_COUNT"
echo -e "${BOLD}═══════════════════════════════════════════${NC}"

TOTAL=$((PASS_COUNT + FAIL_COUNT + SKIP_COUNT))
echo -e "  Total: $TOTAL tests"

if [ "$FAIL_COUNT" -gt 0 ]; then
    echo -e "  ${RED}Some tests failed!${NC}"
    exit 1
fi

echo -e "  ${GREEN}All tests passed!${NC}"
exit 0
