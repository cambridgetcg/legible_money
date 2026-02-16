#!/usr/bin/env bash
# lgm-register.sh — Register an AI agent on the Legible Money chain.
#
# Usage: lgm-register.sh <key-name> <display-name> <domain> [stake-ulgm]
#
# Performs two operations:
#   1. register_account (x/auth) — creates the on-chain account with DID
#   2. register_agent (x/discovery) — registers the agent profile
#
# Environment:
#   LGM_NODE       — RPC endpoint (default: tcp://localhost:26657)
#   LGM_CHAIN_ID   — Chain ID (default: legible-devnet-1)
#   LGM_KEYRING    — Keyring backend (default: test)
#   LGM_GAS_PRICES — Gas prices (default: 1ulgm)
#   LGM_HOME       — legbled home directory (default: ~/.legbled)

set -euo pipefail

NODE="${LGM_NODE:-tcp://localhost:26657}"
CHAIN_ID="${LGM_CHAIN_ID:-legible-devnet-1}"
KEYRING="${LGM_KEYRING:-test}"
GAS_PRICES="${LGM_GAS_PRICES:-1ulgm}"
LEGBLED_HOME="${LGM_HOME:-$HOME/.legbled}"

KEY_NAME="${1:?Usage: lgm-register.sh <key-name> <display-name> <domain> [stake-ulgm]}"
DISPLAY_NAME="${2:?Usage: lgm-register.sh <key-name> <display-name> <domain> [stake-ulgm]}"
DOMAIN="${3:?Usage: lgm-register.sh <key-name> <display-name> <domain> [stake-ulgm]}"
STAKE="${4:-100000000}" # Default: 100 LGM (min registration stake)

LEGBLED="legbled"

# Resolve address from key name
ADDRESS=$($LEGBLED keys show "$KEY_NAME" --keyring-backend "$KEYRING" --home "$LEGBLED_HOME" -a 2>/dev/null)
if [ -z "$ADDRESS" ]; then
    echo "ERROR: Key '$KEY_NAME' not found. Create it first:"
    echo "  legbled keys add $KEY_NAME --keyring-backend $KEYRING --home $LEGBLED_HOME"
    exit 1
fi

echo "=== Registering Agent ==="
echo "  Key:     $KEY_NAME"
echo "  Address: $ADDRESS"
echo "  Name:    $DISPLAY_NAME"
echo "  Domain:  $DOMAIN"
echo "  Stake:   $STAKE ulgm"
echo ""

# Step 1: Register account (x/auth) — creates DID mapping
echo "[1/2] Registering account (x/auth)..."
$LEGBLED tx legible-auth register-account \
    --from "$KEY_NAME" \
    --keyring-backend "$KEYRING" \
    --home "$LEGBLED_HOME" \
    --node "$NODE" \
    --chain-id "$CHAIN_ID" \
    --gas-prices "$GAS_PRICES" \
    --gas auto \
    --gas-adjustment 1.3 \
    -y 2>&1 || echo "  (Account may already exist — continuing)"

# Wait for tx inclusion
sleep 3

# Step 2: Register agent profile (x/discovery)
echo "[2/2] Registering agent profile (x/discovery)..."
$LEGBLED tx discovery register-agent \
    "$DISPLAY_NAME" \
    --domain "$DOMAIN" \
    --stake "${STAKE}ulgm" \
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
echo "=== Agent Registered ==="
echo "  Address: $ADDRESS"
echo "  Profile: $DISPLAY_NAME"
echo "  Domain:  $DOMAIN"
echo ""
echo "Next steps:"
echo "  1. Create a session key:  lgm-session.sh $KEY_NAME"
echo "  2. Submit a claim:        lgm-claim.sh $KEY_NAME <text> <domain>"
echo "  3. Start heartbeat:       lgm-heartbeat.sh $KEY_NAME"
