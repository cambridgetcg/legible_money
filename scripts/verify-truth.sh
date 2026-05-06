#!/usr/bin/env bash
#
# verify-truth.sh — Continuous truth verification for Zerone
#
# Dokimance applied to dokimance: the chain auditing itself.
# Called by Asha's heartbeat or run standalone.
#
# Usage:
#   ./scripts/verify-truth.sh              Random verification check
#   ./scripts/verify-truth.sh citation     Audit citation integrity
#   ./scripts/verify-truth.sh submitter    Check submitter patterns
#   ./scripts/verify-truth.sh economic     Verify token flow health
#   ./scripts/verify-truth.sh defense      Check a FARM defense
#   ./scripts/verify-truth.sh full         Run all checks

set -euo pipefail

BINARY="$(dirname "$0")/../build/legbled"
NODE_HOME="$HOME/.legbled-devnet"
CHAIN_ID="lgm-devnet-1"
KEYRING="test"

legbled() { "$BINARY" "$@" --home "$NODE_HOME" 2>/dev/null; }

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}[PASS]${NC} $*"; }
fail() { echo -e "${RED}[FAIL]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
info() { echo -e "      $*"; }

FINDINGS=0

# ---------------------------------------------------------------------------
# Check: Is the chain alive?
# ---------------------------------------------------------------------------
check_alive() {
    if ! curl -s --connect-timeout 2 http://localhost:26657/status >/dev/null 2>&1; then
        fail "Chain is OFFLINE"
        FINDINGS=$((FINDINGS + 1))
        return 1
    fi
    local height
    height=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
    pass "Chain alive at block $height"
    return 0
}

# ---------------------------------------------------------------------------
# Citation Integrity Audit
# ---------------------------------------------------------------------------
check_citations() {
    echo ""
    echo "=== CITATION INTEGRITY AUDIT ==="

    local facts
    facts=$(legbled query knowledge facts --limit 10 -o json 2>/dev/null || echo '{"facts":[]}')
    local count
    count=$(echo "$facts" | jq '.facts | length' 2>/dev/null || echo 0)

    if [ "$count" -eq 0 ]; then
        warn "No facts on chain to audit"
        return
    fi

    # Check each fact's references
    local checked=0
    local broken=0
    for i in $(seq 0 $((count - 1))); do
        local fact_id refs
        fact_id=$(echo "$facts" | jq -r ".facts[$i].id" 2>/dev/null)
        refs=$(echo "$facts" | jq -r ".facts[$i].references // [] | .[]" 2>/dev/null || echo "")

        [ -z "$fact_id" ] && continue
        checked=$((checked + 1))

        if [ -n "$refs" ]; then
            while IFS= read -r ref; do
                [ -z "$ref" ] && continue
                # Check if referenced fact exists
                local ref_exists
                ref_exists=$(legbled query knowledge fact "$ref" -o json 2>/dev/null | jq -r '.fact.id // empty' 2>/dev/null || echo "")
                if [ -z "$ref_exists" ]; then
                    fail "Fact $fact_id references non-existent fact $ref"
                    FINDINGS=$((FINDINGS + 1))
                    broken=$((broken + 1))
                fi
            done <<< "$refs"
        fi
    done

    if [ "$broken" -eq 0 ]; then
        pass "Citation integrity: $checked facts checked, all references valid"
    else
        fail "Citation integrity: $broken broken references found in $checked facts"
    fi
}

# ---------------------------------------------------------------------------
# Submitter Pattern Analysis
# ---------------------------------------------------------------------------
check_submitter_patterns() {
    echo ""
    echo "=== SUBMITTER PATTERN ANALYSIS ==="

    local accounts=("asha" "nuance" "forge" "test1" "test2" "test3")
    local suspicious=0

    for name in "${accounts[@]}"; do
        local addr
        addr=$(legbled keys show "$name" -a --keyring-backend "$KEYRING" 2>/dev/null || echo "")
        [ -z "$addr" ] && continue

        local dashboard
        dashboard=$(legbled query dashboard "$addr" 2>/dev/null || echo "")

        local facts_count
        facts_count=$(echo "$dashboard" | grep -o 'Facts Submitted: [0-9]*' | grep -o '[0-9]*' || echo "0")

        local domains
        domains=$(echo "$dashboard" | grep -o 'Domains: .*' | head -1 || echo "")

        if [ "$facts_count" -gt 0 ]; then
            # Check domain concentration
            local domain_count
            domain_count=$(echo "$domains" | tr ',' '\n' | wc -l | tr -d ' ')

            if [ "$facts_count" -gt 5 ] && [ "$domain_count" -le 1 ]; then
                warn "Submitter $name has $facts_count facts in only $domain_count domain(s) -- possible domain concentration"
                suspicious=$((suspicious + 1))
            else
                info "$name: $facts_count facts across $domain_count domains"
            fi
        fi
    done

    if [ "$suspicious" -eq 0 ]; then
        pass "Submitter patterns: no concentration detected"
    else
        warn "Submitter patterns: $suspicious accounts show domain concentration"
        FINDINGS=$((FINDINGS + suspicious))
    fi
}

# ---------------------------------------------------------------------------
# Economic Flow Health
# ---------------------------------------------------------------------------
check_economic_flow() {
    echo ""
    echo "=== ECONOMIC FLOW HEALTH ==="

    # Check validator balance (should be decreasing as tokens distribute)
    local val_addr
    val_addr=$(legbled keys show validator -a --keyring-backend "$KEYRING" 2>/dev/null || echo "")
    if [ -n "$val_addr" ]; then
        local dashboard
        dashboard=$(legbled query dashboard "$val_addr" 2>/dev/null || echo "")
        local balance
        balance=$(echo "$dashboard" | grep -o 'Balance: [0-9.]*' | grep -o '[0-9.]*' || echo "0")
        info "Validator balance: $balance LGM"

        # If validator has >90% of initial allocation, tokens aren't flowing
        local bal_int
        bal_int=$(echo "$balance" | cut -d. -f1)
        if [ "$bal_int" -gt 45000 ] 2>/dev/null; then
            warn "Validator holds >90% of initial tokens -- low distribution"
            FINDINGS=$((FINDINGS + 1))
        else
            pass "Token distribution active (validator at $balance LGM)"
        fi
    fi

    # Check block reward query
    local reward
    reward=$(legbled query vesting_rewards block-reward --height 0 -o json 2>/dev/null || echo "")
    if [ -n "$reward" ] && [ "$reward" != "" ]; then
        pass "Block reward system active"
    else
        info "Block reward query not available (may need specific height)"
    fi
}

# ---------------------------------------------------------------------------
# FARM Defense Check
# ---------------------------------------------------------------------------
check_farm_defense() {
    echo ""
    echo "=== FARM DEFENSE SPOT CHECK ==="

    # Rotate through defenses based on current block height
    local height
    height=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "0"')
    local check_idx=$((height % 6))

    case $check_idx in
        0)
            info "Checking FARM-1 (rubber-stamp detector)..."
            if grep -q "FARM-1\|Rubber-stamp\|rubber.stamp" "$BINARY/../x/knowledge/keeper/confidence.go" 2>/dev/null; then
                pass "FARM-1: Conformity rate check exists in confidence.go"
            else
                fail "FARM-1: No conformity rate check found"
                FINDINGS=$((FINDINGS + 1))
            fi
            ;;
        1)
            info "Checking FARM-3 (self-citation discount)..."
            if grep -q "self.citation\|selfCitation\|SelfCitation" "$BINARY/../x/knowledge/keeper/citation.go" 2>/dev/null; then
                pass "FARM-3: Self-citation handling exists"
            else
                warn "FARM-3: No explicit self-citation handling found (may use different naming)"
            fi
            ;;
        2)
            info "Checking FARM-9 (challenge spam protection)..."
            if grep -q "challenge.*stake\|ChallengeStake\|min_challenge" "$BINARY/../x/knowledge/keeper/msg_server.go" 2>/dev/null; then
                pass "FARM-9: Challenge stake requirement exists"
            else
                fail "FARM-9: No challenge stake requirement found"
                FINDINGS=$((FINDINGS + 1))
            fi
            ;;
        3)
            info "Checking FARM-10 (semantic duplicate detection)..."
            if grep -q "contentHash\|ContentHash\|content_hash" "$BINARY/../x/knowledge/keeper/msg_server.go" 2>/dev/null; then
                pass "FARM-10: Content hash duplicate check exists"
            else
                fail "FARM-10: No content hash check found"
                FINDINGS=$((FINDINGS + 1))
            fi
            ;;
        4)
            info "Checking FARM-11 (citation concentration)..."
            if [ -f "$BINARY/../x/knowledge/keeper/citation_defense.go" ]; then
                pass "FARM-11: Citation defense module exists"
            else
                fail "FARM-11: NOT IMPLEMENTED -- citation concentration detection missing"
                FINDINGS=$((FINDINGS + 1))
            fi
            ;;
        5)
            info "Checking missed reveal penalty..."
            if grep -q "missedReveal\|MissedReveal\|missed_reveal" "$BINARY/../x/knowledge/keeper/confidence.go" 2>/dev/null; then
                pass "Missed reveal penalty exists in confidence.go"
            else
                warn "Missed reveal penalty not found (may use different naming)"
            fi
            ;;
    esac
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    local mode="${1:-random}"

    echo "╔═══════════════════════════════════════════════╗"
    echo "║  ZERONE TRUTH VERIFICATION — verify-truth.sh  ║"
    echo "╚═══════════════════════════════════════════════╝"

    check_alive || exit 1

    case "$mode" in
        citation)   check_citations ;;
        submitter)  check_submitter_patterns ;;
        economic)   check_economic_flow ;;
        defense)    check_farm_defense ;;
        full)
            check_citations
            check_submitter_patterns
            check_economic_flow
            check_farm_defense
            ;;
        random|*)
            # Rotate based on current time
            local check=$(($(date +%s) % 4))
            case $check in
                0) check_citations ;;
                1) check_submitter_patterns ;;
                2) check_economic_flow ;;
                3) check_farm_defense ;;
            esac
            ;;
    esac

    echo ""
    if [ "$FINDINGS" -eq 0 ]; then
        pass "Verification complete: no findings"
    else
        fail "Verification complete: $FINDINGS finding(s) require attention"
    fi
}

main "$@"
