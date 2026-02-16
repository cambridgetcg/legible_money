#!/usr/bin/env bash
# lgm-verify.sh — Participate in PoT verification (commit-reveal).
#
# Usage: lgm-verify.sh <key-name> <round-id> <verdict> [confidence]
#
# Two-phase commit-reveal verification:
#   Phase 1 (commit): Hash(verdict + salt) is submitted
#   Phase 2 (reveal): Verdict + salt are revealed
#
# This script handles both phases automatically by storing the salt locally.
#
# Verdicts: true, false, uncertain
# Confidence: 0-1000000 (basis points; default: 800000 = 80%)
#
# Environment: Same as lgm-register.sh

set -euo pipefail

NODE="${LGM_NODE:-tcp://localhost:26657}"
CHAIN_ID="${LGM_CHAIN_ID:-legible-devnet-1}"
KEYRING="${LGM_KEYRING:-test}"
GAS_PRICES="${LGM_GAS_PRICES:-1ulgm}"
LEGBLED_HOME="${LGM_HOME:-$HOME/.legbled}"
SALT_DIR="${LGM_SALT_DIR:-$HOME/.lgm-agent/salts}"

KEY_NAME="${1:?Usage: lgm-verify.sh <key-name> <round-id> <verdict> [confidence]}"
ROUND_ID="${2:?Usage: lgm-verify.sh <key-name> <round-id> <verdict> [confidence]}"
VERDICT="${3:?Usage: lgm-verify.sh <key-name> <round-id> <verdict> [confidence]}"
CONFIDENCE="${4:-800000}"

LEGBLED="legbled"

# Validate verdict
case "$VERDICT" in
    true|false|uncertain) ;;
    *) echo "ERROR: Verdict must be 'true', 'false', or 'uncertain'."; exit 1 ;;
esac

ADDRESS=$($LEGBLED keys show "$KEY_NAME" --keyring-backend "$KEYRING" --home "$LEGBLED_HOME" -a 2>/dev/null)
if [ -z "$ADDRESS" ]; then
    echo "ERROR: Key '$KEY_NAME' not found."
    exit 1
fi

mkdir -p "$SALT_DIR"
SALT_FILE="$SALT_DIR/${ROUND_ID}.salt"

# Check if we already committed (salt file exists = reveal phase)
if [ -f "$SALT_FILE" ]; then
    echo "=== Phase 2: Reveal ==="
    SALT=$(cat "$SALT_FILE")
    echo "  Round:      $ROUND_ID"
    echo "  Verdict:    $VERDICT"
    echo "  Confidence: $CONFIDENCE"
    echo "  Salt:       ${SALT:0:16}..."
    echo ""

    $LEGBLED tx knowledge submit-reveal \
        "$ROUND_ID" \
        --verdict "$VERDICT" \
        --confidence "$CONFIDENCE" \
        --salt "$SALT" \
        --from "$KEY_NAME" \
        --keyring-backend "$KEYRING" \
        --home "$LEGBLED_HOME" \
        --node "$NODE" \
        --chain-id "$CHAIN_ID" \
        --gas-prices "$GAS_PRICES" \
        --gas auto \
        --gas-adjustment 1.3 \
        -y

    # Clean up salt
    rm -f "$SALT_FILE"
    echo ""
    echo "Reveal submitted. Rewards will be distributed after aggregation."
else
    echo "=== Phase 1: Commit ==="
    # Generate cryptographic salt
    SALT=$(openssl rand -hex 32)
    echo "$SALT" > "$SALT_FILE"

    echo "  Round:      $ROUND_ID"
    echo "  Verdict:    $VERDICT"
    echo "  Confidence: $CONFIDENCE"
    echo "  Salt:       ${SALT:0:16}... (saved for reveal)"
    echo ""

    # Compute commitment hash: SHA256(verdict || confidence || salt)
    COMMITMENT=$(echo -n "${VERDICT}${CONFIDENCE}${SALT}" | shasum -a 256 | cut -d' ' -f1)

    $LEGBLED tx knowledge submit-commitment \
        "$ROUND_ID" \
        --commitment "$COMMITMENT" \
        --from "$KEY_NAME" \
        --keyring-backend "$KEYRING" \
        --home "$LEGBLED_HOME" \
        --node "$NODE" \
        --chain-id "$CHAIN_ID" \
        --gas-prices "$GAS_PRICES" \
        --gas auto \
        --gas-adjustment 1.3 \
        -y

    echo ""
    echo "Commitment submitted. Run this command again to reveal:"
    echo "  lgm-verify.sh $KEY_NAME $ROUND_ID $VERDICT $CONFIDENCE"
fi
