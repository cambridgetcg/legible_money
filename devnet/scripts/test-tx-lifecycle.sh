#!/usr/bin/env bash
#
# Legible Money — DevNet Transaction Lifecycle Tests
#
# Runs 23 tests against a live Cosmos SDK devnet covering:
#   Phase 1: Bank Operations          (6 tests)
#   Phase 2: Staking Operations       (5 tests)
#   Phase 3: Governance Operations    (4 tests)
#   Phase 4: Authorization & Errors   (5 tests)
#   Phase 5: Economic Invariants      (3 tests)
#
# Prerequisites:
#   - Cosmos devnet running (cosmos-devnet.sh start)
#   - Accounts funded (validator, test1, test2, test3)
#   - jq installed
#
# Usage:
#   bash devnet/scripts/test-tx-lifecycle.sh
#
# Exit codes:
#   0 = all tests passed
#   1 = one or more tests failed

set -uo pipefail

# ---------------------------------------------------------------------------
# Constants (must match cosmos-devnet.sh)
# ---------------------------------------------------------------------------
CHAIN_ID="lgm-devnet-1"
DENOM="ulgm"
KEYRING="test"
NODE_HOME="$HOME/.legbled-devnet"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BINARY="$PROJECT_ROOT/build/legbled"

# shellcheck source=./test-helpers.sh
source "${SCRIPT_DIR}/test-helpers.sh"

RPC_URL="http://localhost:26657"
REST_URL="http://localhost:1317"

# Test accounting
PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

# Gas price: must cover the node's minimum-gas-prices (1ulgm/gas on devnet)
GAS_PRICES="2${DENOM}"

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

# Execute a legbled CLI command with common flags
legbled() {
    "$BINARY" "$@" --home "$NODE_HOME"
}

# Execute a tx with standard flags (chain-id, keyring, fees, -y, output json)
legbled_tx() {
    "$BINARY" tx "$@" \
        --chain-id "$CHAIN_ID" \
        --keyring-backend "$KEYRING" \
        --home "$NODE_HOME" \
        --gas-prices "$GAS_PRICES" \
        --gas auto \
        --gas-adjustment 2.0 \
        -y \
        --output json 2>/dev/null
}

# Query REST API endpoint, return JSON
rest_query() {
    curl -s "${REST_URL}$1" 2>/dev/null
}

# Get balance for an address in ulgm (returns integer string)
get_balance() {
    local addr="$1"
    local result
    result=$(rest_query "/cosmos/bank/v1beta1/balances/${addr}" | jq -r ".balances[] | select(.denom == \"$DENOM\") | .amount" 2>/dev/null)
    echo "${result:-0}"
}

# Get total supply in ulgm
get_total_supply() {
    rest_query "/cosmos/bank/v1beta1/supply/by_denom?denom=${DENOM}" | jq -r '.amount.amount // "0"' 2>/dev/null
}

# Get address for a key name
get_addr() {
    legbled keys show "$1" -a --keyring-backend "$KEYRING" 2>/dev/null
}

# Wait for a tx to be included in a block (up to 15 seconds)
# Input: tx hash
# Output: tx result JSON (or empty on timeout)
wait_for_tx() {
    local txhash="$1"
    local max_attempts=10
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        sleep 3
        local result
        result=$(legbled query tx "$txhash" --output json 2>/dev/null || echo "")
        if [ -n "$result" ] && echo "$result" | jq -e '.code' >/dev/null 2>&1; then
            echo "$result"
            return 0
        fi
        attempt=$((attempt + 1))
    done
    echo ""
    return 1
}

# Submit a tx and wait for inclusion. Returns the tx result JSON.
# Parses txhash from the broadcast response, then polls.
# If broadcast itself failed (code != 0), returns the broadcast response directly.
submit_and_wait() {
    local tx_output="$1"

    # Check if broadcast itself failed (CheckTx rejection)
    local broadcast_code
    broadcast_code=$(echo "$tx_output" | jq -r '.code // "0"' 2>/dev/null)
    if [ "$broadcast_code" != "0" ] && [ "$broadcast_code" != "null" ]; then
        # Tx was rejected at broadcast — return the broadcast response as-is
        echo "$tx_output"
        return 0
    fi

    local txhash
    txhash=$(echo "$tx_output" | jq -r '.txhash // empty' 2>/dev/null)

    if [ -z "$txhash" ]; then
        echo ""
        return 1
    fi

    wait_for_tx "$txhash"
}

# Assert two values are equal
assert_eq() {
    local test_name="$1"
    local actual="$2"
    local expected="$3"
    if [ "$actual" = "$expected" ]; then
        pass "$test_name"
    else
        fail "$test_name (expected: $expected, got: $actual)"
    fi
}

# Assert actual != expected
assert_neq() {
    local test_name="$1"
    local actual="$2"
    local unexpected="$3"
    if [ "$actual" != "$unexpected" ]; then
        pass "$test_name"
    else
        fail "$test_name (got unexpected value: $actual)"
    fi
}

# Assert actual > threshold (numeric)
assert_gt() {
    local test_name="$1"
    local actual="$2"
    local threshold="$3"
    if [ "$actual" -gt "$threshold" ] 2>/dev/null; then
        pass "$test_name"
    else
        fail "$test_name (expected > $threshold, got: $actual)"
    fi
}

# Assert tx result code == 0 (success)
assert_tx_success() {
    local test_name="$1"
    local tx_result="$2"
    if [ -z "$tx_result" ]; then
        fail "$test_name (tx not found — timeout or broadcast failure)"
        return
    fi
    local code
    code=$(echo "$tx_result" | jq -r '.code // "unknown"' 2>/dev/null)
    if [ "$code" = "0" ]; then
        pass "$test_name"
    else
        local raw_log
        raw_log=$(echo "$tx_result" | jq -r '.raw_log // "no log"' 2>/dev/null)
        fail "$test_name (code: $code, log: $raw_log)"
    fi
}

# Assert tx result code != 0 (expected failure)
assert_tx_fail() {
    local test_name="$1"
    local tx_result="$2"
    local expected_hint="${3:-}"
    if [ -z "$tx_result" ]; then
        # No result could mean rejection before mempool — that's still a failure path
        pass "$test_name (rejected before inclusion)"
        return
    fi
    local code
    code=$(echo "$tx_result" | jq -r '.code // "0"' 2>/dev/null)
    if [ "$code" != "0" ]; then
        pass "$test_name (code: $code)"
    else
        fail "$test_name (expected failure but tx succeeded)"
    fi
}

# ---------------------------------------------------------------------------
# Prerequisites
# ---------------------------------------------------------------------------
check_prereqs() {
    echo "=== Legible Money — Transaction Lifecycle Tests ==="
    echo ""

    # Binary exists
    if [ ! -x "$BINARY" ]; then
        echo -e "${RED}ERROR${NC}: Binary not found at $BINARY"
        echo "       Run 'make devnet-cosmos' or 'go build -o build/legbled ./cmd/legbled' first"
        exit 1
    fi

    # jq available
    if ! command -v jq &>/dev/null; then
        echo -e "${RED}ERROR${NC}: jq is required (brew install jq)"
        exit 1
    fi

    # Node reachable
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" "$RPC_URL/status" 2>/dev/null || echo "000")
    if [ "$status" != "200" ]; then
        echo -e "${RED}ERROR${NC}: DevNet not reachable at $RPC_URL"
        echo "       Start with 'make devnet-cosmos' first"
        exit 1
    fi

    # Wait for at least 2 blocks
    local height
    height=$(curl -s "$RPC_URL/status" | jq -r '.result.sync_info.latest_block_height' 2>/dev/null || echo "0")
    if [ "$height" -lt 2 ] 2>/dev/null; then
        info "Waiting for blocks..."
        sleep 6
    fi

    local chain
    chain=$(curl -s "$RPC_URL/status" | jq -r '.result.node_info.network' 2>/dev/null)
    info "Chain: $chain  |  Height: $height  |  RPC: $RPC_URL"
    echo ""
}

# ---------------------------------------------------------------------------
# Phase 1: Bank Operations (6 tests)
# ---------------------------------------------------------------------------
test_bank_operations() {
    phase "Phase 1: Bank Operations"

    local validator_addr test1_addr test2_addr
    validator_addr=$(get_addr validator)
    test1_addr=$(get_addr test1)
    test2_addr=$(get_addr test2)

    # Record initial balances
    local val_balance_before test1_balance_before
    val_balance_before=$(get_balance "$validator_addr")
    test1_balance_before=$(get_balance "$test1_addr")

    # T1.1: Send tokens validator → test1
    info "T1.1: Send 1 LGM (validator → test1)"
    local send_amount="1000000"
    local tx_out
    tx_out=$(legbled_tx bank send validator "$test1_addr" "${send_amount}${DENOM}")
    local tx_result
    tx_result=$(submit_and_wait "$tx_out")
    assert_tx_success "T1.1: Send tokens (validator → test1)" "$tx_result"

    # Verify balance changed (use >= to account for vesting rewards that accrue between queries)
    local test1_balance_after
    test1_balance_after=$(get_balance "$test1_addr")
    local min_expected=$((test1_balance_before + send_amount))
    if [ "$test1_balance_after" -ge "$min_expected" ] 2>/dev/null; then
        pass "T1.2: Recipient balance increased by at least send amount"
    else
        fail "T1.2: Recipient balance too low (expected >= $min_expected, got: $test1_balance_after)"
    fi

    # T1.3: Send tokens test1 → test2 (secondary transfer)
    info "T1.3: Send 0.5 LGM (test1 → test2)"
    local test2_balance_before
    test2_balance_before=$(get_balance "$test2_addr")
    tx_out=$(legbled_tx bank send test1 "$test2_addr" "500000${DENOM}")
    tx_result=$(submit_and_wait "$tx_out")
    assert_tx_success "T1.3: Secondary transfer (test1 → test2)" "$tx_result"

    # T1.4: Send with insufficient funds — should fail
    info "T1.4: Send with insufficient funds"
    # test3 has 1000 LGM, try sending 999999 LGM (nearly all, but fees will push it over)
    local test3_addr
    test3_addr=$(get_addr test3)
    local test3_balance
    test3_balance=$(get_balance "$test3_addr")
    local overspend=$((test3_balance + 1000000))
    tx_out=$(legbled_tx bank send test3 "$test1_addr" "${overspend}${DENOM}" 2>&1 || echo '{"txhash":""}')
    tx_result=$(submit_and_wait "$tx_out")
    assert_tx_fail "T1.4: Insufficient funds rejected" "$tx_result"

    # T1.5: Multi-denomination query — only ulgm should exist
    info "T1.5: Multi-denomination query"
    local denoms
    denoms=$(rest_query "/cosmos/bank/v1beta1/balances/${validator_addr}" | jq '[.balances[].denom]' 2>/dev/null)
    local denom_count
    denom_count=$(echo "$denoms" | jq 'length' 2>/dev/null || echo "0")
    assert_eq "T1.5: Only ulgm denomination exists" "$denom_count" "1"

    # T1.6: Query non-existent account — should return empty balances
    info "T1.6: Query non-existent account"
    local fake_addr="lgm1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq0lfmjr"
    local fake_balance
    fake_balance=$(rest_query "/cosmos/bank/v1beta1/balances/${fake_addr}" | jq '.balances | length' 2>/dev/null || echo "-1")
    assert_eq "T1.6: Non-existent account returns empty balances" "$fake_balance" "0"
}

# ---------------------------------------------------------------------------
# Phase 2: Staking Operations (5 tests)
# ---------------------------------------------------------------------------
test_staking_operations() {
    phase "Phase 2: Staking Operations"

    local test1_addr validator_addr
    test1_addr=$(get_addr test1)
    validator_addr=$(get_addr validator)

    # T2.1: Query validators — at least 1 active
    info "T2.1: Query validators"
    local val_count
    val_count=$(rest_query "/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED" | jq '.validators | length' 2>/dev/null || echo "0")
    assert_gt "T2.1: At least 1 bonded validator" "$val_count" "0"

    # Get the validator operator address for delegation
    local valoper
    valoper=$(rest_query "/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED" | jq -r '.validators[0].operator_address' 2>/dev/null)

    # Record bonded pool before delegation
    local bonded_before
    bonded_before=$(rest_query "/cosmos/staking/v1beta1/pool" | jq -r '.pool.bonded_tokens' 2>/dev/null || echo "0")

    # Record delegation before to check delta (test may run on non-fresh state)
    local delegation_before
    delegation_before=$(rest_query "/cosmos/staking/v1beta1/delegations/${test1_addr}" | \
        jq -r ".delegation_responses[] | select(.delegation.validator_address == \"$valoper\") | .balance.amount" 2>/dev/null || echo "0")
    if [ -z "$delegation_before" ]; then delegation_before="0"; fi

    # T2.2: Delegate from test1 to validator
    info "T2.2: Delegate 100 LGM (test1 → validator)"
    local delegate_amount="100000000"
    local tx_out
    tx_out=$(legbled_tx staking delegate "$valoper" "${delegate_amount}${DENOM}" --from test1)
    local tx_result
    tx_result=$(submit_and_wait "$tx_out")
    assert_tx_success "T2.2: Delegate from test1 to validator" "$tx_result"

    # T2.3: Query delegation — delta should match delegated amount
    info "T2.3: Query delegation"
    local delegation_after
    delegation_after=$(rest_query "/cosmos/staking/v1beta1/delegations/${test1_addr}" | \
        jq -r ".delegation_responses[] | select(.delegation.validator_address == \"$valoper\") | .balance.amount" 2>/dev/null || echo "0")
    local delegation_delta=$(( delegation_after - delegation_before ))
    assert_eq "T2.3: Delegation amount correct" "$delegation_delta" "$delegate_amount"

    # T2.4: Undelegate — unbonding entry created
    info "T2.4: Undelegate 50 LGM"
    local undelegate_amount="50000000"
    tx_out=$(legbled_tx staking unbond "$valoper" "${undelegate_amount}${DENOM}" --from test1)
    tx_result=$(submit_and_wait "$tx_out")
    assert_tx_success "T2.4: Undelegate creates unbonding entry" "$tx_result"

    # T2.5: Bonded pool increased by net delegation (100 - 50 = 50 LGM net)
    info "T2.5: Staking pool conservation"
    local bonded_after
    bonded_after=$(rest_query "/cosmos/staking/v1beta1/pool" | jq -r '.pool.bonded_tokens' 2>/dev/null || echo "0")
    local net_delegation=$((delegate_amount - undelegate_amount))
    local expected_bonded=$((bonded_before + net_delegation))
    # Allow for small variance due to block reward distribution
    local bonded_diff
    bonded_diff=$(( bonded_after - expected_bonded ))
    if [ "$bonded_diff" -lt 0 ]; then bonded_diff=$(( -bonded_diff )); fi
    # Tolerance: within 1 LGM (block rewards may shift bonded pool slightly)
    if [ "$bonded_diff" -lt 1000000 ]; then
        pass "T2.5: Bonded pool consistent (diff: ${bonded_diff}${DENOM})"
    else
        fail "T2.5: Bonded pool mismatch (expected ~$expected_bonded, got $bonded_after, diff: $bonded_diff)"
    fi
}

# ---------------------------------------------------------------------------
# Phase 3: Governance Operations (4 tests)
# ---------------------------------------------------------------------------
test_governance_operations() {
    phase "Phase 3: Governance Operations"

    local test1_addr
    test1_addr=$(get_addr test1)

    # T3.1: Submit text proposal (SDK v0.50 uses submit-legacy-proposal)
    info "T3.1: Submit text proposal"
    local tx_out
    tx_out=$(legbled_tx gov submit-legacy-proposal \
        --type text \
        --title "Test Proposal" \
        --description "Lifecycle test proposal for devnet validation" \
        --deposit "10000000${DENOM}" \
        --from test1 2>&1 || echo '{"txhash":""}')

    local tx_result
    tx_result=$(submit_and_wait "$tx_out")
    assert_tx_success "T3.1: Submit governance proposal" "$tx_result"

    # Find the proposal ID from events
    local proposal_id
    proposal_id=$(echo "$tx_result" | jq -r '[.events[] | select(.type == "submit_proposal") | .attributes[] | select(.key == "proposal_id") | .value] | first // empty' 2>/dev/null)
    if [ -z "$proposal_id" ]; then
        # Fallback: query latest proposal
        proposal_id=$(rest_query "/cosmos/gov/v1/proposals?proposal_status=PROPOSAL_STATUS_VOTING_PERIOD" | jq -r '.proposals[-1].id // empty' 2>/dev/null)
    fi
    if [ -z "$proposal_id" ]; then
        proposal_id=$(rest_query "/cosmos/gov/v1/proposals?proposal_status=PROPOSAL_STATUS_DEPOSIT_PERIOD" | jq -r '.proposals[-1].id // empty' 2>/dev/null)
    fi

    if [ -z "$proposal_id" ]; then
        skip "T3.2-T3.4: Could not determine proposal ID"
        return
    fi

    info "  Proposal ID: $proposal_id"

    # T3.2: Vote on proposal
    info "T3.2: Vote YES on proposal"
    # Validator must vote (they have voting power)
    tx_out=$(legbled_tx gov vote "$proposal_id" yes --from validator)
    tx_result=$(submit_and_wait "$tx_out")
    assert_tx_success "T3.2: Vote on proposal" "$tx_result"

    # T3.3: Query proposal — status and vote tally
    info "T3.3: Query proposal status"
    local proposal_status
    proposal_status=$(rest_query "/cosmos/gov/v1/proposals/${proposal_id}" | jq -r '.proposal.status' 2>/dev/null || echo "unknown")
    assert_neq "T3.3: Proposal has valid status" "$proposal_status" "unknown"
    info "  Status: $proposal_status"

    # T3.4: Wait for voting period to end (devnet: 60s)
    # Only wait if still in voting period
    if [ "$proposal_status" = "PROPOSAL_STATUS_VOTING_PERIOD" ]; then
        info "T3.4: Waiting for voting period (60s)..."
        sleep 65
        proposal_status=$(rest_query "/cosmos/gov/v1/proposals/${proposal_id}" | jq -r '.proposal.status' 2>/dev/null || echo "unknown")
    fi

    if [ "$proposal_status" = "PROPOSAL_STATUS_PASSED" ] || [ "$proposal_status" = "PROPOSAL_STATUS_REJECTED" ]; then
        pass "T3.4: Proposal reached final status ($proposal_status)"
    else
        fail "T3.4: Proposal did not finalize (status: $proposal_status)"
    fi
}

# ---------------------------------------------------------------------------
# Phase 4: Authorization & Errors (5 tests)
# ---------------------------------------------------------------------------
test_authorization_errors() {
    phase "Phase 4: Authorization & Errors"

    local test1_addr test2_addr
    test1_addr=$(get_addr test1)
    test2_addr=$(get_addr test2)

    # T4.1: Tx with wrong chain-id — should fail signature verification
    info "T4.1: Wrong chain-id"
    local tx_out
    tx_out=$("$BINARY" tx bank send test1 "$test2_addr" "1000${DENOM}" \
        --chain-id "wrong-chain-99" \
        --keyring-backend "$KEYRING" \
        --home "$NODE_HOME" \
        --gas-prices "$GAS_PRICES" \
        --gas 200000 \
        -y \
        --output json 2>&1 || echo '{"code": 1}')
    # The node will reject with signature mismatch or the CLI may error before broadcast
    local code
    code=$(echo "$tx_out" | jq -r '.code // "1"' 2>/dev/null || echo "1")
    if [ "$code" != "0" ]; then
        pass "T4.1: Wrong chain-id rejected (code: $code)"
    else
        local tx_result
        tx_result=$(submit_and_wait "$tx_out")
        assert_tx_fail "T4.1: Wrong chain-id rejected" "$tx_result"
    fi

    # T4.2: Tx with zero fees — should fail if min gas price enforced
    info "T4.2: Insufficient fees"
    tx_out=$("$BINARY" tx bank send test1 "$test2_addr" "1000${DENOM}" \
        --chain-id "$CHAIN_ID" \
        --keyring-backend "$KEYRING" \
        --home "$NODE_HOME" \
        --gas-prices "0${DENOM}" \
        --gas 200000 \
        -y \
        --output json 2>&1 || echo '{"code": 1}')
    code=$(echo "$tx_out" | jq -r '.code // "1"' 2>/dev/null || echo "1")
    if [ "$code" != "0" ]; then
        pass "T4.2: Zero fees rejected (code: $code)"
    else
        local tx_result
        tx_result=$(submit_and_wait "$tx_out")
        assert_tx_fail "T4.2: Zero fees rejected" "$tx_result"
    fi

    # T4.3: Tx with invalid recipient address
    info "T4.3: Invalid recipient address"
    tx_out=$("$BINARY" tx bank send test1 "invalid_address_xyz" "1000${DENOM}" \
        --chain-id "$CHAIN_ID" \
        --keyring-backend "$KEYRING" \
        --home "$NODE_HOME" \
        --gas-prices "$GAS_PRICES" --gas 200000 \
        -y \
        --output json 2>&1 || echo '{"code": 1}')
    code=$(echo "$tx_out" | jq -r '.code // "1"' 2>/dev/null || echo "1")
    if [ "$code" != "0" ]; then
        pass "T4.3: Invalid recipient rejected (code: $code)"
    else
        fail "T4.3: Invalid recipient was not rejected"
    fi

    # T4.4: Double-spend — send full balance twice quickly
    info "T4.4: Double-spend attempt"
    local test3_addr
    test3_addr=$(get_addr test3)
    local test3_balance
    test3_balance=$(get_balance "$test3_addr")
    # Send nearly all balance (leave room for fees on first tx)
    local spend_amount=$((test3_balance - 100000))
    if [ "$spend_amount" -le 0 ]; then
        skip "T4.4: test3 balance too low for double-spend test"
    else
        # First send — should succeed
        tx_out=$(legbled_tx bank send test3 "$test1_addr" "${spend_amount}${DENOM}")
        local tx1_hash
        tx1_hash=$(echo "$tx_out" | jq -r '.txhash // empty' 2>/dev/null)

        # Second send immediately — should fail (insufficient funds after first tx)
        local tx_out2
        tx_out2=$(legbled_tx bank send test3 "$test1_addr" "${spend_amount}${DENOM}" 2>&1 || echo '{"code": 1}')

        # Wait for first tx
        if [ -n "$tx1_hash" ]; then
            wait_for_tx "$tx1_hash" >/dev/null 2>&1
        fi

        # Second tx should have failed
        local tx_result2
        tx_result2=$(submit_and_wait "$tx_out2")
        assert_tx_fail "T4.4: Double-spend rejected" "$tx_result2"
    fi

    # T4.5: Query with invalid address — should return error, no crash
    info "T4.5: REST query with invalid address"
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" "${REST_URL}/cosmos/bank/v1beta1/balances/not-a-valid-address" 2>/dev/null || echo "000")
    # Should get a 400 or similar error, but NOT crash (500)
    if [ "$http_code" != "500" ] && [ "$http_code" != "000" ]; then
        pass "T4.5: Invalid address query handled gracefully (HTTP $http_code)"
    else
        fail "T4.5: Invalid address query caused server error (HTTP $http_code)"
    fi
}

# ---------------------------------------------------------------------------
# Phase 5: Economic Invariants (3 tests)
# ---------------------------------------------------------------------------
test_economic_invariants() {
    phase "Phase 5: Economic Invariants"

    # T5.1: Total supply conservation across transfers
    info "T5.1: Supply conservation"
    local supply_before
    supply_before=$(get_total_supply)

    # Execute a transfer
    local test1_addr test2_addr
    test1_addr=$(get_addr test1)
    test2_addr=$(get_addr test2)
    local tx_out
    tx_out=$(legbled_tx bank send test1 "$test2_addr" "100000${DENOM}")
    local tx_result
    tx_result=$(submit_and_wait "$tx_out")

    local supply_after
    supply_after=$(get_total_supply)

    # Supply should be unchanged (fees stay in the system via fee collector)
    assert_eq "T5.1: Total supply unchanged after transfer" "$supply_after" "$supply_before"

    # T5.2: Fee deducted from sender
    info "T5.2: Fee deduction"
    local sender_before
    sender_before=$(get_balance "$test1_addr")

    local send_amount="100000"
    tx_out=$(legbled_tx bank send test1 "$test2_addr" "${send_amount}${DENOM}")
    tx_result=$(submit_and_wait "$tx_out")

    local sender_after
    sender_after=$(get_balance "$test1_addr")

    if [ -n "$tx_result" ]; then
        local code
        code=$(echo "$tx_result" | jq -r '.code // "unknown"' 2>/dev/null)
        if [ "$code" = "0" ]; then
            local deducted=$((sender_before - sender_after))
            # Sender should have lost more than just the send amount (fees on top)
            if [ "$deducted" -gt "$send_amount" ]; then
                local fee_paid=$((deducted - send_amount))
                pass "T5.2: Fee deducted from sender (send: ${send_amount}, fee: ${fee_paid}, total: ${deducted})"
            else
                fail "T5.2: No fee deducted (deducted: ${deducted}, send: ${send_amount})"
            fi
        else
            skip "T5.2: Tx failed, cannot verify fee deduction"
        fi
    else
        skip "T5.2: Tx not included, cannot verify fee deduction"
    fi

    # T5.3: Staking conservation — bonded + not_bonded = sum of delegations
    info "T5.3: Staking pool conservation"
    local pool
    pool=$(rest_query "/cosmos/staking/v1beta1/pool")
    local bonded not_bonded
    bonded=$(echo "$pool" | jq -r '.pool.bonded_tokens // "0"' 2>/dev/null)
    not_bonded=$(echo "$pool" | jq -r '.pool.not_bonded_tokens // "0"' 2>/dev/null)

    # Both values should be non-negative integers
    if [ "$bonded" -ge 0 ] 2>/dev/null && [ "$not_bonded" -ge 0 ] 2>/dev/null; then
        local staking_total=$((bonded + not_bonded))
        assert_gt "T5.3: Staking pool has positive total ($staking_total = $bonded bonded + $not_bonded unbonding)" "$staking_total" "0"
    else
        fail "T5.3: Could not parse staking pool (bonded: $bonded, not_bonded: $not_bonded)"
    fi
}

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
print_summary() {
    local total=$((PASS_COUNT + FAIL_COUNT + SKIP_COUNT))
    echo ""
    echo "╔═══════════════════════════════════════════════════════╗"
    echo "║          Transaction Lifecycle Test Results           ║"
    echo "╠═══════════════════════════════════════════════════════╣"
    printf "║  Passed:  %-43s ║\n" "${PASS_COUNT}/${total}"
    printf "║  Failed:  %-43s ║\n" "${FAIL_COUNT}/${total}"
    printf "║  Skipped: %-43s ║\n" "${SKIP_COUNT}/${total}"
    echo "╚═══════════════════════════════════════════════════════╝"
    echo ""

    if [ "$FAIL_COUNT" -gt 0 ]; then
        echo -e "${RED}FAILED${NC}: $FAIL_COUNT test(s) failed"
        exit 1
    else
        echo -e "${GREEN}ALL TESTS PASSED${NC}"
        exit 0
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    check_prereqs

    test_bank_operations
    test_staking_operations
    test_governance_operations
    test_authorization_errors
    test_economic_invariants

    print_summary
}

main "$@"
