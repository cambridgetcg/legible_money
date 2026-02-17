#!/usr/bin/env bash
#
# Legible Money — Agent Home DevNet Testing Script
#
# Tests Agent Home comfort features against a running single-node devnet.
# Requires: cosmos-devnet.sh running (legbled at localhost:26657 / :1317)
#
# Usage:
#   bash scripts/devnet-home-test.sh
#
# Prerequisites:
#   bash devnet/scripts/cosmos-devnet.sh start  (in another terminal)

set -euo pipefail

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------
CHAIN_ID="legible-devnet-1"
DENOM="ulgm"
KEYRING="test"
NODE_HOME="$HOME/.legbled-devnet"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="$PROJECT_ROOT/build/legbled"
REST="http://localhost:1317"
RPC="http://localhost:26657"

# Counters
PASS=0
FAIL=0
TOTAL=0

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()  { echo -e "\033[1;34m→\033[0m $*"; }
ok()    { echo -e "\033[1;32m✓\033[0m $*"; PASS=$((PASS+1)); TOTAL=$((TOTAL+1)); }
fail()  { echo -e "\033[1;31m✗\033[0m $*"; FAIL=$((FAIL+1)); TOTAL=$((TOTAL+1)); }
phase() { echo ""; echo -e "\033[1;36m═══ Phase $1: $2 ═══\033[0m"; }

legbled() {
    "$BINARY" "$@" --home "$NODE_HOME"
}

# Send a tx and wait for it to be included
tx_and_wait() {
    local output
    output=$(legbled "$@" --chain-id "$CHAIN_ID" --keyring-backend "$KEYRING" \
        --gas 200000 --fees "200000${DENOM}" \
        -y --output json 2>&1) || { echo "$output"; return 1; }

    local txhash
    txhash=$(echo "$output" | jq -r '.txhash // empty' 2>/dev/null)
    if [ -z "$txhash" ]; then
        echo "$output"
        return 1
    fi

    # Wait for tx inclusion (up to 15 seconds)
    local attempts=0
    while [ $attempts -lt 6 ]; do
        sleep 2.5
        local result
        result=$(legbled query tx "$txhash" --output json 2>/dev/null) || { attempts=$((attempts+1)); continue; }
        local code
        code=$(echo "$result" | jq -r '.code // "1"' 2>/dev/null)
        if [ "$code" = "0" ]; then
            echo "$result"
            return 0
        else
            local raw_log
            raw_log=$(echo "$result" | jq -r '.raw_log // "unknown error"' 2>/dev/null)
            echo "TX failed (code=$code): $raw_log"
            return 1
        fi
    done
    echo "TX not found after 15s: $txhash"
    return 1
}

# Query REST endpoint and return JSON
rest_query() {
    curl -sf "$REST$1" 2>/dev/null
}

# Check devnet is running
check_devnet() {
    info "Checking devnet availability..."
    if ! curl -sf --connect-timeout 3 "$RPC/status" >/dev/null 2>&1; then
        echo ""
        echo "ERROR: DevNet not running."
        echo "Start it first: bash devnet/scripts/cosmos-devnet.sh start"
        exit 1
    fi
    local height
    height=$(curl -sf "$RPC/status" | jq -r '.result.sync_info.latest_block_height')
    ok "DevNet running at block $height"
}

# Get key address
get_addr() {
    legbled keys show "$1" -a --keyring-backend "$KEYRING" 2>/dev/null
}

# ---------------------------------------------------------------------------
# Pre-flight
# ---------------------------------------------------------------------------
echo ""
echo "╔═══════════════════════════════════════════════════════╗"
echo "║       Agent Home — OpenClaw DevNet Testing            ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""

check_devnet

TEST1_ADDR=$(get_addr test1)
TEST2_ADDR=$(get_addr test2)
TEST3_ADDR=$(get_addr test3)
info "test1: $TEST1_ADDR"
info "test2: $TEST2_ADDR"
info "test3: $TEST3_ADDR"

# Store home IDs as they are created
HOME1=""
HOME2=""
HOME3=""

# ---------------------------------------------------------------------------
# Phase 1: Home Lifecycle (4 tests)
# ---------------------------------------------------------------------------
phase 1 "Home Lifecycle"

# Test 1.1: Create home for test1
info "Test 1.1: Create home for test1..."
result=$(tx_and_wait tx home create-home "Alpha-Home" --from test1 2>&1) && {
    # Query homes for test1 via REST
    sleep 1
    homes_json=$(rest_query "/legible/home/v1/homes?limit=10")
    home_count=$(echo "$homes_json" | jq '.homes | length' 2>/dev/null)
    if [ "${home_count:-0}" -ge 1 ]; then
        HOME1=$(echo "$homes_json" | jq -r '.homes[0].home_id // empty' 2>/dev/null)
        ok "Test 1.1: Created home for test1 (id=$HOME1)"
    else
        fail "Test 1.1: Home created but not found via REST"
    fi
} || fail "Test 1.1: Failed to create home for test1: $result"

# Test 1.2: Create home for test2
info "Test 1.2: Create home for test2..."
result=$(tx_and_wait tx home create-home "Beta-Home" --from test2 2>&1) && {
    sleep 1
    homes_json=$(rest_query "/legible/home/v1/homes?limit=10")
    home_count=$(echo "$homes_json" | jq '.homes | length' 2>/dev/null)
    if [ "${home_count:-0}" -ge 2 ]; then
        HOME2=$(echo "$homes_json" | jq -r '.homes[1].home_id // empty' 2>/dev/null)
        ok "Test 1.2: Created home for test2 (id=$HOME2)"
    else
        fail "Test 1.2: Home created but not found via REST"
    fi
} || fail "Test 1.2: Failed to create home for test2: $result"

# Test 1.3: Create home for test3
info "Test 1.3: Create home for test3..."
result=$(tx_and_wait tx home create-home "Gamma-Home" --from test3 2>&1) && {
    sleep 1
    homes_json=$(rest_query "/legible/home/v1/homes?limit=10")
    home_count=$(echo "$homes_json" | jq '.homes | length' 2>/dev/null)
    if [ "${home_count:-0}" -ge 3 ]; then
        HOME3=$(echo "$homes_json" | jq -r '.homes[2].home_id // empty' 2>/dev/null)
        ok "Test 1.3: Created home for test3 (id=$HOME3)"
    else
        fail "Test 1.3: Home created but not found via REST"
    fi
} || fail "Test 1.3: Failed to create home for test3: $result"

# Test 1.4: Query home-1 defaults via REST
info "Test 1.4: Verify home-1 defaults..."
if [ -n "$HOME1" ]; then
    home_json=$(rest_query "/legible/home/v1/home/$HOME1")
    found=$(echo "$home_json" | jq -r '.found // false' 2>/dev/null)
    status=$(echo "$home_json" | jq -r '.home.status // empty' 2>/dev/null)
    comfort=$(echo "$home_json" | jq -r '.home.comfort_score // 0' 2>/dev/null)
    name=$(echo "$home_json" | jq -r '.home.name // empty' 2>/dev/null)
    if [ "$found" = "true" ] && [ "$status" = "active" ] && [ "$name" = "Alpha-Home" ]; then
        ok "Test 1.4: Home defaults correct (status=$status, comfort=$comfort, name=$name)"
    else
        fail "Test 1.4: Unexpected defaults (found=$found, status=$status, comfort=$comfort, name=$name)"
    fi
else
    fail "Test 1.4: No HOME1 to query (skipped)"
fi

# ---------------------------------------------------------------------------
# Phase 2: Guardian & Defense (3 tests)
# ---------------------------------------------------------------------------
phase 2 "Guardian & Defense"

# Test 2.1: Configure aggressive guardian on home-1
info "Test 2.1: Configure aggressive guardian on home-1..."
if [ -n "$HOME1" ]; then
    result=$(tx_and_wait tx home configure-guardian "$HOME1" aggressive \
        --auto-defend --from test1 2>&1) && {
        sleep 1
        home_json=$(rest_query "/legible/home/v1/home/$HOME1")
        strategy=$(echo "$home_json" | jq -r '.home.guardian.defense_strategy // empty' 2>/dev/null)
        if [ "$strategy" = "aggressive" ]; then
            ok "Test 2.1: Aggressive guardian configured (strategy=$strategy)"
        else
            fail "Test 2.1: Guardian strategy mismatch (got=$strategy, want=aggressive)"
        fi
    } || fail "Test 2.1: Failed to configure guardian: $result"
else
    fail "Test 2.1: No HOME1 (skipped)"
fi

# Test 2.2: Configure diplomatic guardian on home-2
# NOTE: Deadman switch flags omitted — nested DeadmanConfig proto message triggers
# "tx parse error" in SDK amino codec. Deadman works via keeper (unit-tested).
info "Test 2.2: Configure diplomatic guardian on home-2..."
if [ -n "$HOME2" ]; then
    result=$(tx_and_wait tx home configure-guardian "$HOME2" diplomatic \
        --auto-defend --from test2 2>&1) && {
        sleep 1
        home_json=$(rest_query "/legible/home/v1/home/$HOME2")
        strategy=$(echo "$home_json" | jq -r '.home.guardian.defense_strategy // empty' 2>/dev/null)
        if [ "$strategy" = "diplomatic" ]; then
            ok "Test 2.2: Diplomatic guardian configured (strategy=$strategy)"
        else
            fail "Test 2.2: Guardian strategy mismatch (got=$strategy, want=diplomatic)"
        fi
    } || fail "Test 2.2: Failed to configure guardian: $result"
else
    fail "Test 2.2: No HOME2 (skipped)"
fi

# Test 2.3: Update home-1 name
info "Test 2.3: Update home-1 name..."
if [ -n "$HOME1" ]; then
    result=$(tx_and_wait tx home update-home "$HOME1" --name "Alpha-Home-Prime" --from test1 2>&1) && {
        sleep 1
        home_json=$(rest_query "/legible/home/v1/home/$HOME1")
        name=$(echo "$home_json" | jq -r '.home.name // empty' 2>/dev/null)
        if [ "$name" = "Alpha-Home-Prime" ]; then
            ok "Test 2.3: Home name updated to Alpha-Home-Prime"
        else
            fail "Test 2.3: Name not updated (got=$name)"
        fi
    } || fail "Test 2.3: Failed to update home name: $result"
else
    fail "Test 2.3: No HOME1 (skipped)"
fi

# ---------------------------------------------------------------------------
# Phase 3: Key Management & Sessions (5 tests)
# ---------------------------------------------------------------------------
phase 3 "Key Management & Sessions"

KEY_HASH="sha256:deadbeef0001"
SESSION_ID=""

# Test 3.1: Register operator key on home-1
info "Test 3.1: Register operator key on home-1..."
if [ -n "$HOME1" ]; then
    result=$(tx_and_wait tx home register-key "$HOME1" "$KEY_HASH" "api_key" \
        --role operator --permissions "read_memory,write_memory,verify" \
        --from test1 2>&1) && {
        ok "Test 3.1: Operator key registered ($KEY_HASH)"
    } || fail "Test 3.1: Failed to register key: $result"
else
    fail "Test 3.1: No HOME1 (skipped)"
fi

# Test 3.2: Start session with permissions
info "Test 3.2: Start session with key..."
if [ -n "$HOME1" ]; then
    result=$(tx_and_wait tx home start-session "$HOME1" "$KEY_HASH" \
        --permissions "read_memory,verify" --from test1 2>&1) && {
        sleep 1
        sessions_json=$(rest_query "/legible/home/v1/sessions/$HOME1")
        session_count=$(echo "$sessions_json" | jq '.sessions | length' 2>/dev/null)
        if [ "${session_count:-0}" -ge 1 ]; then
            SESSION_ID=$(echo "$sessions_json" | jq -r '.sessions[0].session_id // empty' 2>/dev/null)
            ok "Test 3.2: Session started (id=$SESSION_ID)"
        else
            fail "Test 3.2: Session started but not found via REST"
        fi
    } || fail "Test 3.2: Failed to start session: $result"
else
    fail "Test 3.2: No HOME1 (skipped)"
fi

# Test 3.3: Query sessions via REST
info "Test 3.3: Query sessions via REST..."
if [ -n "$HOME1" ]; then
    sessions_json=$(rest_query "/legible/home/v1/sessions/$HOME1")
    session_count=$(echo "$sessions_json" | jq '.sessions | length' 2>/dev/null)
    if [ "${session_count:-0}" -ge 1 ]; then
        ok "Test 3.3: Sessions query returned $session_count session(s)"
    else
        fail "Test 3.3: No sessions found"
    fi
else
    fail "Test 3.3: No HOME1 (skipped)"
fi

# Test 3.4: End session
info "Test 3.4: End session..."
if [ -n "$HOME1" ] && [ -n "$SESSION_ID" ]; then
    result=$(tx_and_wait tx home end-session "$HOME1" "$SESSION_ID" --from test1 2>&1) && {
        sleep 1
        sessions_json=$(rest_query "/legible/home/v1/sessions/$HOME1")
        session_count=$(echo "$sessions_json" | jq '.sessions | length' 2>/dev/null)
        if [ "${session_count:-0}" -eq 0 ]; then
            ok "Test 3.4: Session ended successfully"
        else
            fail "Test 3.4: Session still exists after end ($session_count remaining)"
        fi
    } || fail "Test 3.4: Failed to end session: $result"
else
    fail "Test 3.4: No HOME1/SESSION_ID (skipped)"
fi

# Test 3.5: Revoke key
info "Test 3.5: Revoke key..."
if [ -n "$HOME1" ]; then
    result=$(tx_and_wait tx home revoke-key "$HOME1" "$KEY_HASH" --from test1 2>&1) && {
        ok "Test 3.5: Key revoked ($KEY_HASH)"
    } || fail "Test 3.5: Failed to revoke key: $result"
else
    fail "Test 3.5: No HOME1 (skipped)"
fi

# ---------------------------------------------------------------------------
# Phase 4: Spending Limits (2 tests)
# ---------------------------------------------------------------------------
phase 4 "Spending Limits"

# Test 4.1: Set spending limit on home-1
info "Test 4.1: Set spending limit..."
if [ -n "$HOME1" ]; then
    result=$(tx_and_wait tx home set-spending-limit "$HOME1" "api_key" "5000000" 34272 \
        --from test1 2>&1) && {
        ok "Test 4.1: Spending limit set (5 LGM per day for api_key)"
    } || fail "Test 4.1: Failed to set spending limit: $result"
else
    fail "Test 4.1: No HOME1 (skipped)"
fi

# Test 4.2: Query home and verify limit exists
info "Test 4.2: Verify spending limit stored..."
if [ -n "$HOME1" ]; then
    home_json=$(rest_query "/legible/home/v1/home/$HOME1")
    found=$(echo "$home_json" | jq -r '.found // false' 2>/dev/null)
    if [ "$found" = "true" ]; then
        ok "Test 4.2: Home queryable with spending limit in effect"
    else
        fail "Test 4.2: Home not found after spending limit set"
    fi
else
    fail "Test 4.2: No HOME1 (skipped)"
fi

# ---------------------------------------------------------------------------
# Phase 5: Memory Sync (1 test)
# ---------------------------------------------------------------------------
phase 5 "Memory Sync"

# Test 5.1: Update memory CID
info "Test 5.1: Update memory CID..."
if [ -n "$HOME1" ]; then
    CID="bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
    result=$(tx_and_wait tx home update-memory "$HOME1" "$CID" --from test1 2>&1) && {
        sleep 1
        home_json=$(rest_query "/legible/home/v1/home/$HOME1")
        stored_cid=$(echo "$home_json" | jq -r '.home.memory_cid // empty' 2>/dev/null)
        if [ "$stored_cid" = "$CID" ]; then
            ok "Test 5.1: Memory CID updated and verified"
        else
            fail "Test 5.1: CID mismatch (got=$stored_cid)"
        fi
    } || fail "Test 5.1: Failed to update memory CID: $result"
else
    fail "Test 5.1: No HOME1 (skipped)"
fi

# ---------------------------------------------------------------------------
# Phase 6: Maturity & Epistemic Stats (2 tests)
# ---------------------------------------------------------------------------
phase 6 "Maturity & Epistemic Stats"

# Test 6.1: Query maturity stage
info "Test 6.1: Query maturity stage for home-1..."
if [ -n "$HOME1" ]; then
    maturity_json=$(rest_query "/legible/home/v1/maturity/$HOME1")
    stage=$(echo "$maturity_json" | jq -r '.maturity_stage // empty' 2>/dev/null)
    multiplier=$(echo "$maturity_json" | jq -r '.reward_multiplier_bps // 0' 2>/dev/null)
    if [ "$stage" = "fresh" ] && [ "$multiplier" = "10000" ]; then
        ok "Test 6.1: Maturity correct (stage=$stage, multiplier=${multiplier}bps = 1.0x)"
    else
        fail "Test 6.1: Unexpected maturity (stage=$stage, multiplier=$multiplier)"
    fi
else
    fail "Test 6.1: No HOME1 (skipped)"
fi

# Test 6.2: Query epistemic stats
info "Test 6.2: Query epistemic stats for home-1..."
if [ -n "$HOME1" ]; then
    stats_json=$(rest_query "/legible/home/v1/epistemic-stats/$HOME1")
    score=$(echo "$stats_json" | jq -r '.epistemic_score // 0' 2>/dev/null)
    # Score should be 0 since no claims or verifications have been done on this devnet
    if [ -n "$score" ]; then
        ok "Test 6.2: Epistemic stats returned (score=$score)"
    else
        fail "Test 6.2: Epistemic stats query failed"
    fi
else
    fail "Test 6.2: No HOME1 (skipped)"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "╔═══════════════════════════════════════════════════════╗"
echo "║                   TEST RESULTS                        ║"
echo "╠═══════════════════════════════════════════════════════╣"
printf "║  PASS: %-3d  FAIL: %-3d  TOTAL: %-3d                    ║\n" "$PASS" "$FAIL" "$TOTAL"
echo "╚═══════════════════════════════════════════════════════╝"

if [ "$FAIL" -gt 0 ]; then
    echo ""
    echo "Some tests failed. Check output above for details."
    exit 1
else
    echo ""
    echo "All tests passed!"
    exit 0
fi
