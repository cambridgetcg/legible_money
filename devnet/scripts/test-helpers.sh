#!/usr/bin/env bash
# Shared E2E test helpers for Legible Money devnet scripts.
# Source this after setting BINARY, NODE_HOME, KEYRING, GAS_PRICES.
#
# Usage:
#   source "$(dirname "${BASH_SOURCE[0]}")/test-helpers.sh"

# settle: Wait for chain state to propagate after a confirmed transaction.
# Call between back-to-back transactions from the same account to prevent
# nonce races (code 32: account sequence mismatch).
#
# After submit_and_wait confirms a tx is in a block, the account sequence
# should be updated immediately. However, on fast-running test suites the
# CLI's account query can occasionally hit a stale ABCI query cache.
# Sleeping one block period (~2.5s, rounded up to 3s) guarantees the next
# block has been committed and all state is queryable.
settle() {
    sleep 3
}

# wait_for_block: Wait until the chain reaches at least the given height.
# Usage: wait_for_block 5
wait_for_block() {
    local target="${1:-2}"
    local rpc="${RPC_URL:-http://localhost:26657}"
    local max_attempts=20
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        local h
        h=$(curl -s "$rpc/status" | jq -r '.result.sync_info.latest_block_height' 2>/dev/null || echo "0")
        if [ "$h" -ge "$target" ] 2>/dev/null; then
            return 0
        fi
        sleep 1
        attempt=$((attempt + 1))
    done
    return 1
}
