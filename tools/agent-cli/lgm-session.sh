#!/usr/bin/env bash
# lgm-session.sh — Create a session key for an agent.
#
# Usage: lgm-session.sh <key-name> [capabilities] [duration-blocks]
#
# Creates a session key with scoped capabilities for automated operations.
# Session keys allow the agent to operate without exposing the master key.
#
# Capabilities (comma-separated):
#   CanSubmitClaims, CanVote, CanTransfer, CanStake, CanIca
#
# Environment: Same as lgm-register.sh

set -euo pipefail

NODE="${LGM_NODE:-tcp://localhost:26657}"
CHAIN_ID="${LGM_CHAIN_ID:-legible-devnet-1}"
KEYRING="${LGM_KEYRING:-test}"
GAS_PRICES="${LGM_GAS_PRICES:-1ulgm}"
LEGBLED_HOME="${LGM_HOME:-$HOME/.legbled}"

KEY_NAME="${1:?Usage: lgm-session.sh <key-name> [capabilities] [duration-blocks]}"
CAPABILITIES="${2:-CanSubmitClaims,CanVote}"
DURATION="${3:-34272}" # Default: ~1 day at 2521ms blocks

LEGBLED="legbled"

ADDRESS=$($LEGBLED keys show "$KEY_NAME" --keyring-backend "$KEYRING" --home "$LEGBLED_HOME" -a 2>/dev/null)
if [ -z "$ADDRESS" ]; then
    echo "ERROR: Key '$KEY_NAME' not found."
    exit 1
fi

# Generate a session key
SESSION_KEY_NAME="${KEY_NAME}-session-$(date +%s)"
echo "=== Creating Session Key ==="
echo "  Parent:       $KEY_NAME ($ADDRESS)"
echo "  Session:      $SESSION_KEY_NAME"
echo "  Capabilities: $CAPABILITIES"
echo "  Duration:     $DURATION blocks"
echo ""

# Create the session key in keyring
$LEGBLED keys add "$SESSION_KEY_NAME" --keyring-backend "$KEYRING" --home "$LEGBLED_HOME" 2>/dev/null
SESSION_ADDRESS=$($LEGBLED keys show "$SESSION_KEY_NAME" --keyring-backend "$KEYRING" --home "$LEGBLED_HOME" -a)

# Register session key on-chain (x/auth)
echo "Registering session key on-chain..."
$LEGBLED tx legible-auth create-session \
    --session-address "$SESSION_ADDRESS" \
    --capabilities "$CAPABILITIES" \
    --duration "$DURATION" \
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
echo "=== Session Key Created ==="
echo "  Key Name: $SESSION_KEY_NAME"
echo "  Address:  $SESSION_ADDRESS"
echo "  Caps:     $CAPABILITIES"
echo "  Expires:  ~$DURATION blocks from now"
echo ""
echo "Use this key for automated operations:"
echo "  lgm-claim.sh $SESSION_KEY_NAME <text> <domain>"
echo "  lgm-verify.sh $SESSION_KEY_NAME <claim-id>"
