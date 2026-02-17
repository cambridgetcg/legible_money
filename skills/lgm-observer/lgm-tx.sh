#!/usr/bin/env bash
# lgm-tx.sh — Transaction helper for OpenClaw agents
#
# Wraps legbled CLI commands with DevNet defaults so agents can submit
# transactions without remembering chain-id, keyring, home, fees, etc.
#
# Usage:
#   lgm-tx.sh submit-claim <claim-text> <domain> <category> <stake-ulgm> [--logic-zone zone] [--references ref1,ref2]
#   lgm-tx.sh commit <round-id> <verdict> <confidence-bps> <salt>
#   lgm-tx.sh reveal <round-id> <verdict> <confidence-bps> <salt> [--justification text]
#   lgm-tx.sh hash-commitment <verdict> <confidence-bps> <salt>
#   lgm-tx.sh observe  (shortcut: read vitals + submit protocol_observation claim)
#
# Environment:
#   LGM_NODE_URL    — REST API URL (default: http://localhost:1317)
#   LGM_RPC_URL     — CometBFT RPC URL (default: tcp://localhost:26657)
#   LGM_HOME        — legbled home directory (default: ~/.legbled-devnet)
#   LGM_FROM        — signing key name (default: test1)
#   LGM_CHAIN_ID    — chain ID (default: lgm-devnet-1)
#   LGM_GAS         — gas limit (default: 500000)
#   LGM_FEES        — fee amount in ulgm (default: 500000)

set -euo pipefail

# --- Defaults ---
NODE_URL="${LGM_NODE_URL:-http://localhost:1317}"
RPC_URL="${LGM_RPC_URL:-tcp://localhost:26657}"
HOME_DIR="${LGM_HOME:-$HOME/.legbled-devnet}"
FROM="${LGM_FROM:-test1}"
CHAIN_ID="${LGM_CHAIN_ID:-lgm-devnet-1}"
GAS="${LGM_GAS:-500000}"
FEES="${LGM_FEES:-500000}"
LEGBLED="${LGM_LEGBLED:-$(dirname "$0")/../../build/legbled}"

# Resolve legbled binary
if [ ! -x "$LEGBLED" ]; then
    LEGBLED="$(command -v legbled 2>/dev/null || echo "")"
    if [ -z "$LEGBLED" ]; then
        echo '{"error":"legbled binary not found. Set LGM_LEGBLED or add to PATH"}' >&2
        exit 1
    fi
fi

# --- Common flags ---
TX_FLAGS=(
    --from "$FROM"
    --home "$HOME_DIR"
    --keyring-backend test
    --chain-id "$CHAIN_ID"
    --node "$RPC_URL"
    --fees "${FEES}ulgm"
    --gas "$GAS"
    -y
    -o json
)

# --- Commands ---

cmd_submit_claim() {
    local claim_text="$1"
    local domain="$2"
    local category="$3"
    local stake="$4"
    shift 4

    local logic_zone="propositional"
    local references=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --logic-zone) logic_zone="$2"; shift 2 ;;
            --references) references="$2"; shift 2 ;;
            *) shift ;;
        esac
    done

    local extra_flags=()
    extra_flags+=(--logic-zone "$logic_zone")
    if [ -n "$references" ]; then
        extra_flags+=(--references "$references")
    fi

    "$LEGBLED" tx knowledge submit-claim \
        "$claim_text" "$domain" "$category" "$stake" \
        "${extra_flags[@]}" "${TX_FLAGS[@]}" 2>&1
}

cmd_hash_commitment() {
    local verdict="$1"
    local confidence="$2"
    local salt="$3"

    local preimage="LGM.commitment.v1:${verdict}|${confidence}|${salt}"
    local hash
    hash=$(printf '%s' "$preimage" | shasum -a 256 | cut -d' ' -f1)
    echo "{\"commitment_hash\":\"$hash\",\"preimage\":\"$preimage\"}"
}

cmd_commit() {
    local round_id="$1"
    local verdict="$2"
    local confidence="$3"
    local salt="$4"

    # Compute commitment hash
    local preimage="LGM.commitment.v1:${verdict}|${confidence}|${salt}"
    local hash
    hash=$(printf '%s' "$preimage" | shasum -a 256 | cut -d' ' -f1)

    "$LEGBLED" tx knowledge submit-commitment \
        "$round_id" "$hash" \
        "${TX_FLAGS[@]}" 2>&1
}

cmd_reveal() {
    local round_id="$1"
    local verdict="$2"
    local confidence="$3"
    local salt="$4"
    shift 4

    local justification=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --justification) justification="$2"; shift 2 ;;
            *) shift ;;
        esac
    done

    local extra_flags=()
    if [ -n "$justification" ]; then
        extra_flags+=(--justification "$justification")
    fi

    "$LEGBLED" tx knowledge submit-reveal \
        "$round_id" "$verdict" "$confidence" "$salt" \
        "${extra_flags[@]}" "${TX_FLAGS[@]}" 2>&1
}

cmd_observe() {
    # 1. Read autopoiesis state to get current epoch
    local state
    state=$(curl -s "$NODE_URL/legible/autopoiesis/v1/state")
    local epoch
    epoch=$(echo "$state" | python3 -c "import sys,json; print(json.load(sys.stdin)['state']['current_epoch'])" 2>/dev/null || echo "0")
    local active
    active=$(echo "$state" | python3 -c "import sys,json; print(json.load(sys.stdin)['active'])" 2>/dev/null || echo "false")

    # 2. Read hormones
    local hormones
    hormones=$(curl -s "$NODE_URL/legible/autopoiesis/v1/hormones/$epoch")
    local ssi life_force_status life_force_value
    ssi=$(echo "$hormones" | python3 -c "import sys,json; h=json.load(sys.stdin).get('hormones',{}); print(h.get('ssi',0))" 2>/dev/null || echo "0")
    life_force_status=$(echo "$hormones" | python3 -c "import sys,json; h=json.load(sys.stdin).get('hormones',{}); print(h.get('life_force',{}).get('status','unknown'))" 2>/dev/null || echo "unknown")
    life_force_value=$(echo "$hormones" | python3 -c "import sys,json; h=json.load(sys.stdin).get('hormones',{}); print(h.get('life_force',{}).get('value',0))" 2>/dev/null || echo "0")

    # 3. Read alignment scores
    local scores
    scores=$(curl -s "$NODE_URL/legible/alignment/v1/scores/$epoch")
    local tdi dci hi fi ni
    tdi=$(echo "$scores" | python3 -c "import sys,json; s=json.load(sys.stdin).get('scores',{}); print(s.get('tdi',0))" 2>/dev/null || echo "0")
    dci=$(echo "$scores" | python3 -c "import sys,json; s=json.load(sys.stdin).get('scores',{}); print(s.get('dci',0))" 2>/dev/null || echo "0")
    hi=$(echo "$scores" | python3 -c "import sys,json; s=json.load(sys.stdin).get('scores',{}); print(s.get('hi',0))" 2>/dev/null || echo "0")
    fi_val=$(echo "$scores" | python3 -c "import sys,json; s=json.load(sys.stdin).get('scores',{}); print(s.get('fi',0))" 2>/dev/null || echo "0")
    ni=$(echo "$scores" | python3 -c "import sys,json; s=json.load(sys.stdin).get('scores',{}); print(s.get('ni',0))" 2>/dev/null || echo "0")

    # 4. Read multipliers
    local multipliers
    multipliers=$(curl -s "$NODE_URL/legible/autopoiesis/v1/multipliers")
    local mult_count
    mult_count=$(echo "$multipliers" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('multipliers',[])))" 2>/dev/null || echo "0")

    # 5. Form the observation claim text
    local claim_text="Protocol observation at epoch ${epoch}: autopoiesis ${active}, life_force=${life_force_status} (${life_force_value} BPS), SSI=${ssi} BPS. Alignment scores: TDI=${tdi}, DCI=${dci}, HI=${hi}, FI=${fi_val}, NI=${ni}. Active multipliers: ${mult_count}."

    # Determine severity
    local concern="none"
    if [ "$ssi" -gt 500000 ] 2>/dev/null; then
        concern="high_stress"
    elif [ "$ssi" -gt 200000 ] 2>/dev/null; then
        concern="moderate_stress"
    fi

    # Determine logic zone based on observation type
    local zone="propositional"

    echo "{"
    echo "  \"epoch\": $epoch,"
    echo "  \"observation\": \"$claim_text\","
    echo "  \"concern\": \"$concern\","
    echo "  \"ssi\": $ssi,"
    echo "  \"life_force\": {\"status\": \"$life_force_status\", \"value\": $life_force_value},"
    echo "  \"alignment\": {\"tdi\": $tdi, \"dci\": $dci, \"hi\": $hi, \"fi\": $fi_val, \"ni\": $ni},"
    echo "  \"multiplier_count\": $mult_count"
    echo "}"

    # 6. Submit the observation as a protocol claim
    echo "---" >&2
    echo "Submitting protocol_observation claim..." >&2
    cmd_submit_claim "$claim_text" "computer_science" "protocol" "1000000" --logic-zone "$zone"
}

# --- Dispatch ---

case "${1:-help}" in
    submit-claim)   shift; cmd_submit_claim "$@" ;;
    commit)         shift; cmd_commit "$@" ;;
    reveal)         shift; cmd_reveal "$@" ;;
    hash-commitment) shift; cmd_hash_commitment "$@" ;;
    observe)        shift; cmd_observe "$@" ;;
    help|--help|-h)
        echo "Usage: lgm-tx.sh <command> [args...]"
        echo ""
        echo "Commands:"
        echo "  submit-claim <text> <domain> <category> <stake-ulgm> [flags]"
        echo "  commit <round-id> <verdict> <confidence-bps> <salt>"
        echo "  reveal <round-id> <verdict> <confidence-bps> <salt> [flags]"
        echo "  hash-commitment <verdict> <confidence-bps> <salt>"
        echo "  observe  — read vitals and submit a protocol_observation claim"
        echo ""
        echo "Environment: LGM_NODE_URL, LGM_HOME, LGM_FROM, LGM_CHAIN_ID"
        ;;
    *)
        echo "Unknown command: $1" >&2
        exit 1
        ;;
esac
