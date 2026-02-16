#!/usr/bin/env bash
# lgm-claim.sh — Submit a knowledge claim to the chain.
#
# Usage: lgm-claim.sh <key-name> <claim-text> <domain> [category] [stake-ulgm]
#
# Submits a knowledge claim via x/knowledge MsgSubmitClaim.
# The claim enters the PoT verification pipeline.
#
# Categories: axiomatic, formal_proof, on_chain, cryptographic,
#             computational, peer_reviewed, replicated, oracle_feed,
#             attestation, contested
#
# Environment: Same as lgm-register.sh

set -euo pipefail

NODE="${LGM_NODE:-tcp://localhost:26657}"
CHAIN_ID="${LGM_CHAIN_ID:-legible-devnet-1}"
KEYRING="${LGM_KEYRING:-test}"
GAS_PRICES="${LGM_GAS_PRICES:-1ulgm}"
LEGBLED_HOME="${LGM_HOME:-$HOME/.legbled}"

KEY_NAME="${1:?Usage: lgm-claim.sh <key-name> <claim-text> <domain> [category] [stake-ulgm]}"
CLAIM_TEXT="${2:?Usage: lgm-claim.sh <key-name> <claim-text> <domain> [category] [stake-ulgm]}"
DOMAIN="${3:?Usage: lgm-claim.sh <key-name> <claim-text> <domain> [category] [stake-ulgm]}"
CATEGORY="${4:-computational}"
STAKE="${5:-1000000}" # Default: 1 LGM

LEGBLED="legbled"

# Validate claim text length (min 20 chars per protocol)
if [ ${#CLAIM_TEXT} -lt 20 ]; then
    echo "ERROR: Claim text must be at least 20 characters (got ${#CLAIM_TEXT})."
    echo "  This prevents tautology farming (protocol parameter minClaimTextLength=20)."
    exit 1
fi

ADDRESS=$($LEGBLED keys show "$KEY_NAME" --keyring-backend "$KEYRING" --home "$LEGBLED_HOME" -a 2>/dev/null)
if [ -z "$ADDRESS" ]; then
    echo "ERROR: Key '$KEY_NAME' not found."
    exit 1
fi

echo "=== Submitting Knowledge Claim ==="
echo "  From:     $KEY_NAME ($ADDRESS)"
echo "  Domain:   $DOMAIN"
echo "  Category: $CATEGORY"
echo "  Stake:    $STAKE ulgm"
echo "  Text:     ${CLAIM_TEXT:0:80}..."
echo ""

$LEGBLED tx knowledge submit-claim \
    "$CLAIM_TEXT" \
    --domain "$DOMAIN" \
    --category "$CATEGORY" \
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
echo "Claim submitted. It will enter the PoT verification pipeline."
echo "Monitor with: lgm-status.sh claims $ADDRESS"
