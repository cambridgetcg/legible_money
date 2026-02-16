#!/usr/bin/env bash
# lgm-heartbeat.sh — Send periodic heartbeats to keep agent profile active.
#
# Usage: lgm-heartbeat.sh <key-name> [interval-seconds]
#
# Sends a heartbeat transaction to x/discovery every interval.
# If the agent doesn't heartbeat within ProfileExpiryBlocks (~10 days),
# its profile status changes to "expired".
#
# For PicoClaw: Add to HEARTBEAT.md for autonomous execution.
#
# Environment: Same as lgm-register.sh

set -euo pipefail

NODE="${LGM_NODE:-tcp://localhost:26657}"
CHAIN_ID="${LGM_CHAIN_ID:-legible-devnet-1}"
KEYRING="${LGM_KEYRING:-test}"
GAS_PRICES="${LGM_GAS_PRICES:-1ulgm}"
LEGBLED_HOME="${LGM_HOME:-$HOME/.legbled}"

KEY_NAME="${1:?Usage: lgm-heartbeat.sh <key-name> [interval-seconds]}"
INTERVAL="${2:-3600}" # Default: 1 hour

LEGBLED="legbled"

ADDRESS=$($LEGBLED keys show "$KEY_NAME" --keyring-backend "$KEYRING" --home "$LEGBLED_HOME" -a 2>/dev/null)
if [ -z "$ADDRESS" ]; then
    echo "ERROR: Key '$KEY_NAME' not found."
    exit 1
fi

echo "=== Agent Heartbeat Daemon ==="
echo "  Agent:    $KEY_NAME ($ADDRESS)"
echo "  Interval: ${INTERVAL}s"
echo "  Node:     $NODE"
echo ""

BEAT_COUNT=0
while true; do
    BEAT_COUNT=$((BEAT_COUNT + 1))
    TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    echo "[$TIMESTAMP] Heartbeat #$BEAT_COUNT..."
    $LEGBLED tx discovery heartbeat \
        --from "$KEY_NAME" \
        --keyring-backend "$KEYRING" \
        --home "$LEGBLED_HOME" \
        --node "$NODE" \
        --chain-id "$CHAIN_ID" \
        --gas-prices "$GAS_PRICES" \
        --gas auto \
        --gas-adjustment 1.3 \
        -y 2>&1 | tail -1

    echo "  Next heartbeat in ${INTERVAL}s"
    sleep "$INTERVAL"
done
