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
#   lgm-tx.sh hash-commitment <round-id> <verdict> <confidence-bps> <salt>
#   lgm-tx.sh observe  (shortcut: read vitals + submit protocol_observation claim)
#   lgm-tx.sh verify   (poll active rounds, auto-commit/reveal)
#   lgm-tx.sh watch    (continuous verification daemon)
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

# Salt storage directory (keyed by round_id)
SALT_DIR="${HOME_DIR}/agent-salts/${FROM}"

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

# --- Commitment Hash (matches chain's ComputeCommitmentHash exactly) ---
# Format: SHA-256("LGM.commitment.v1:" + LP(roundID) + LP(verdict) + confidence_le64 + LP(salt))
# where LP = 4-byte big-endian length prefix + data, confidence = 8-byte little-endian uint64
compute_commitment_hash() {
    python3 -c "
import hashlib, struct, sys
round_id, verdict, confidence, salt = sys.argv[1], sys.argv[2], int(sys.argv[3]), sys.argv[4]
h = hashlib.sha256()
h.update(b'LGM.commitment.v1:')
for s in [round_id, verdict]:
    b = s.encode()
    h.update(struct.pack('>I', len(b)))
    h.update(b)
h.update(struct.pack('<Q', confidence))
b = salt.encode()
h.update(struct.pack('>I', len(b)))
h.update(b)
print(h.hexdigest())
" "$1" "$2" "$3" "$4"
}

# --- Salt Persistence ---
save_salt() {
    local round_id="$1" verdict="$2" confidence="$3" salt="$4"
    mkdir -p "$SALT_DIR"
    echo "${verdict}|${confidence}|${salt}" > "$SALT_DIR/${round_id}.salt"
}

load_salt() {
    local round_id="$1"
    local salt_file="$SALT_DIR/${round_id}.salt"
    if [ -f "$salt_file" ]; then
        cat "$salt_file"
        return 0
    fi
    return 1
}

clear_salt() {
    local round_id="$1"
    rm -f "$SALT_DIR/${round_id}.salt"
}

# Clean up salt files for completed rounds
cleanup_stale_salts() {
    [ -d "$SALT_DIR" ] || return 0
    local active_rounds
    active_rounds=$(curl -s "$NODE_URL/legible/knowledge/v1/rounds/active" 2>/dev/null || echo '{"rounds":[]}')
    local active_ids
    active_ids=$(echo "$active_rounds" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for r in data.get('rounds', []):
    print(r.get('id', ''))
" 2>/dev/null || true)

    for salt_file in "$SALT_DIR"/*.salt; do
        [ -f "$salt_file" ] || continue
        local rid
        rid=$(basename "$salt_file" .salt)
        if ! echo "$active_ids" | grep -qF "$rid"; then
            rm -f "$salt_file"
        fi
    done
}

# --- Agent State Persistence ---
# State file tracks observations, outcomes, trends for the recursive loop.
STATE_DIR="${HOME_DIR}/agent-state/${FROM}"
STATE_FILE="${STATE_DIR}/state.json"

# Initialize empty state JSON
empty_state() {
    echo '{"observations":[],"trends":{"ssi_history":[],"accuracy":{"submitted":0,"accepted":0,"rejected":0,"expired":0}},"last_observe_epoch":0,"last_reflect_height":0}'
}

load_state() {
    mkdir -p "$STATE_DIR"
    if [ -f "$STATE_FILE" ]; then
        cat "$STATE_FILE"
    else
        empty_state
    fi
}

save_state() {
    local state_json="$1"
    mkdir -p "$STATE_DIR"
    echo "$state_json" > "${STATE_FILE}.tmp"
    mv "${STATE_FILE}.tmp" "$STATE_FILE"
}

# Add an observation to state. Prints updated state JSON.
add_observation() {
    local state="$1" claim_id="$2" epoch="$3" height="$4" ssi="$5" concern="$6" stake="$7"
    echo "$state" | python3 -c "
import sys, json
state = json.load(sys.stdin)
state['observations'].append({
    'claim_id': '$claim_id',
    'epoch': int('$epoch'),
    'height': int('$height'),
    'ssi': int('$ssi'),
    'concern': '$concern',
    'stake': int('$stake'),
    'status': 'pending',
    'outcome': None
})
state['last_observe_epoch'] = int('$epoch')
print(json.dumps(state))
"
}

# Update an observation's status/outcome. Prints updated state JSON.
update_observation() {
    local state="$1" claim_id="$2" new_status="$3"
    echo "$state" | python3 -c "
import sys, json
state = json.load(sys.stdin)
for obs in state['observations']:
    if obs['claim_id'] == '$claim_id':
        obs['status'] = '$new_status'
        obs['outcome'] = '$new_status'
        break
# Update accuracy counts
acc = state['trends']['accuracy']
if '$new_status' == 'accepted':
    acc['accepted'] = acc.get('accepted', 0) + 1
elif '$new_status' == 'rejected':
    acc['rejected'] = acc.get('rejected', 0) + 1
elif '$new_status' == 'expired':
    acc['expired'] = acc.get('expired', 0) + 1
print(json.dumps(state))
"
}

# Prune observations older than max_keep (default 50) entries.
prune_observations() {
    local state="$1" max_keep="${2:-50}"
    echo "$state" | python3 -c "
import sys, json
state = json.load(sys.stdin)
obs = state['observations']
if len(obs) > $max_keep:
    state['observations'] = obs[-$max_keep:]
print(json.dumps(state))
"
}

# Compute adaptive stake based on accuracy (returns ulgm amount)
compute_adaptive_stake() {
    local state="$1"
    echo "$state" | python3 -c "
import sys, json
state = json.load(sys.stdin)
acc = state['trends']['accuracy']
submitted = acc.get('submitted', 0)
accepted = acc.get('accepted', 0)
# Learning period: first 5 observations, always 1 LGM
if submitted < 5:
    print(1000000)
    sys.exit()
accuracy = (accepted * 100) // submitted if submitted > 0 else 0
if accuracy >= 80:
    print(2000000)   # 2 LGM — high confidence
elif accuracy >= 50:
    print(1000000)   # 1 LGM — default
else:
    print(500000)    # 0.5 LGM — reduce risk
"
}

# Determine SSI trend direction from history: "rising", "falling", "stable"
compute_ssi_trend() {
    local state="$1"
    echo "$state" | python3 -c "
import sys, json
state = json.load(sys.stdin)
hist = state['trends'].get('ssi_history', [])
if len(hist) < 3:
    print('stable')
    sys.exit()
last3 = hist[-3:]
if last3[2] > last3[0] and last3[2] > last3[1]:
    print('rising')
elif last3[2] < last3[0] and last3[2] < last3[1]:
    print('falling')
else:
    print('stable')
"
}

# Get agent balance in ulgm from REST API
get_balance() {
    local addr
    addr=$("$LEGBLED" keys show "$FROM" --home "$HOME_DIR" --keyring-backend test -a 2>/dev/null || echo "")
    if [ -z "$addr" ]; then
        echo "0"
        return
    fi
    local bal_json
    bal_json=$(curl -s "$NODE_URL/cosmos/bank/v1beta1/balances/$addr" 2>/dev/null || echo '{}')
    echo "$bal_json" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for b in data.get('balances', []):
    if b.get('denom') == 'ulgm':
        print(b['amount'])
        sys.exit()
print('0')
" 2>/dev/null || echo "0"
}

# Get current block height from REST API
get_current_height() {
    curl -s "$NODE_URL/cosmos/base/tendermint/v1beta1/blocks/latest" 2>/dev/null \
        | python3 -c "import sys,json; print(json.load(sys.stdin)['block']['header']['height'])" 2>/dev/null || echo "0"
}

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
    local round_id="$1"
    local verdict="$2"
    local confidence="$3"
    local salt="$4"

    local hash
    hash=$(compute_commitment_hash "$round_id" "$verdict" "$confidence" "$salt")
    echo "{\"commitment_hash\":\"$hash\",\"round_id\":\"$round_id\",\"verdict\":\"$verdict\",\"confidence\":$confidence}"
}

cmd_commit() {
    local round_id="$1"
    local verdict="$2"
    local confidence="$3"
    local salt="$4"

    # Compute commitment hash (binary LP format matching chain)
    local hash
    hash=$(compute_commitment_hash "$round_id" "$verdict" "$confidence" "$salt")

    # Save salt for reveal phase
    save_salt "$round_id" "$verdict" "$confidence" "$salt"

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

    # Clear saved salt after reveal
    clear_salt "$round_id"
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
    local tdi dci hi fi_val ni
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
    local stake="${LGM_OBSERVE_STAKE:-1000000}"
    echo "---" >&2
    echo "Submitting protocol_observation claim (stake: ${stake} ulgm)..." >&2
    local tx_result
    tx_result=$(cmd_submit_claim "$claim_text" "computer_science" "protocol" "$stake" --logic-zone "$zone" 2>&1) || true
    echo "$tx_result" >&2

    # 7. Record observation in agent state
    local claim_id=""
    claim_id=$(echo "$tx_result" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    # Try to extract claim_id from tx response logs
    for log in data.get('logs', []):
        for ev in log.get('events', []):
            for attr in ev.get('attributes', []):
                if attr.get('key') == 'claim_id':
                    print(attr['value'])
                    sys.exit()
    # Fallback: use txhash as claim reference
    if 'txhash' in data:
        print('tx:' + data['txhash'])
except:
    pass
" 2>/dev/null || echo "")

    if [ -n "$claim_id" ]; then
        local height
        height=$(get_current_height)
        local state
        state=$(load_state)
        state=$(add_observation "$state" "$claim_id" "$epoch" "$height" "$ssi" "$concern" "$stake")
        # Append SSI to trend history
        state=$(echo "$state" | python3 -c "
import sys, json
s = json.load(sys.stdin)
s['trends']['ssi_history'].append(int('$ssi'))
# Keep last 20 SSI values
s['trends']['ssi_history'] = s['trends']['ssi_history'][-20:]
s['trends']['accuracy']['submitted'] = s['trends']['accuracy'].get('submitted', 0) + 1
print(json.dumps(s))
")
        save_state "$state"
        echo "Recorded observation: $claim_id" >&2
    fi
}

# --- P2: Automated Verification ---

# Evaluate a claim — returns "accept <confidence>" or "reject <confidence>" or "abstain 0"
# For protocol observations: cross-check against on-chain state
evaluate_claim() {
    local claim_id="$1"

    # Fetch claim details
    local claim_json
    claim_json=$(curl -s "$NODE_URL/legible/knowledge/v1/claim/$claim_id" 2>/dev/null || echo '{}')

    local claim_text category domain
    claim_text=$(echo "$claim_json" | python3 -c "import sys,json; c=json.load(sys.stdin).get('claim',{}); print(c.get('claim_text',''))" 2>/dev/null || echo "")
    category=$(echo "$claim_json" | python3 -c "import sys,json; c=json.load(sys.stdin).get('claim',{}); print(c.get('epistemic_category',''))" 2>/dev/null || echo "")
    domain=$(echo "$claim_json" | python3 -c "import sys,json; c=json.load(sys.stdin).get('claim',{}); print(c.get('domain',''))" 2>/dev/null || echo "")

    if [ -z "$claim_text" ]; then
        echo "abstain 0"
        return
    fi

    # Protocol observation claims: verify by checking on-chain state
    if [ "$category" = "protocol" ]; then
        # Extract epoch from claim text and cross-check vitals
        local claimed_epoch
        claimed_epoch=$(echo "$claim_text" | python3 -c "
import sys, re
text = sys.stdin.read()
m = re.search(r'epoch (\d+)', text)
print(m.group(1) if m else '0')
" 2>/dev/null || echo "0")

        # Check if the epoch mentioned is recent (within 5 epochs of current)
        local current_epoch
        current_epoch=$(curl -s "$NODE_URL/legible/autopoiesis/v1/state" | python3 -c "import sys,json; print(json.load(sys.stdin)['state']['current_epoch'])" 2>/dev/null || echo "0")

        local epoch_diff=$(( current_epoch - claimed_epoch ))
        if [ "$epoch_diff" -lt 0 ]; then epoch_diff=$(( -epoch_diff )); fi

        if [ "$epoch_diff" -le 5 ]; then
            # Recent observation — verify SSI value if present
            local claimed_ssi
            claimed_ssi=$(echo "$claim_text" | python3 -c "
import sys, re
text = sys.stdin.read()
m = re.search(r'SSI=(\d+)', text)
print(m.group(1) if m else '-1')
" 2>/dev/null || echo "-1")

            if [ "$claimed_ssi" != "-1" ]; then
                local actual_ssi
                actual_ssi=$(curl -s "$NODE_URL/legible/autopoiesis/v1/hormones/$claimed_epoch" | python3 -c "import sys,json; print(json.load(sys.stdin).get('hormones',{}).get('ssi',0))" 2>/dev/null || echo "0")

                if [ "$claimed_ssi" = "$actual_ssi" ]; then
                    echo "accept 800000"
                    return
                else
                    echo "reject 700000"
                    return
                fi
            fi

            # Can't verify specific values but epoch is recent
            echo "accept 500000"
            return
        fi

        # Stale epoch observation
        echo "abstain 100000"
        return
    fi

    # Non-protocol claims: abstain (no evaluation capability for other categories yet)
    echo "abstain 0"
}

cmd_verify() {
    echo "Polling active verification rounds..." >&2

    local rounds_json
    rounds_json=$(curl -s "$NODE_URL/legible/knowledge/v1/rounds/active" 2>/dev/null || echo '{"rounds":[]}')

    local round_count
    round_count=$(echo "$rounds_json" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('rounds',[])))" 2>/dev/null || echo "0")

    if [ "$round_count" = "0" ]; then
        echo '{"status":"idle","active_rounds":0}'
        return
    fi

    echo "Found $round_count active round(s)" >&2

    # Process each round
    echo "$rounds_json" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for r in data.get('rounds', []):
    print(json.dumps({
        'id': r.get('id',''),
        'claim_id': r.get('claim_id',''),
        'phase': r.get('phase',''),
        'commit_deadline': r.get('commit_deadline_height',0),
        'reveal_deadline': r.get('reveal_deadline_height',0),
    }))
" 2>/dev/null | while IFS= read -r round_line; do
        local round_id claim_id phase
        round_id=$(echo "$round_line" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
        claim_id=$(echo "$round_line" | python3 -c "import sys,json; print(json.load(sys.stdin)['claim_id'])")
        phase=$(echo "$round_line" | python3 -c "import sys,json; print(json.load(sys.stdin)['phase'])")

        case "$phase" in
            commit)
                # Check if we already committed
                if [ -f "$SALT_DIR/${round_id}.salt" ]; then
                    echo "  Round $round_id: already committed, waiting for reveal" >&2
                    continue
                fi

                # Evaluate the claim
                local eval_result
                eval_result=$(evaluate_claim "$claim_id")
                local verdict confidence
                verdict=$(echo "$eval_result" | cut -d' ' -f1)
                confidence=$(echo "$eval_result" | cut -d' ' -f2)

                echo "  Round $round_id: evaluating claim $claim_id → $verdict ($confidence BPS)" >&2

                # Generate salt and commit
                local salt
                salt=$(openssl rand -hex 16)

                local result
                result=$(cmd_commit "$round_id" "$verdict" "$confidence" "$salt" 2>&1) || true
                echo "  Round $round_id: committed ($verdict)" >&2
                ;;

            reveal)
                # Check if we have a saved salt for this round
                local saved
                if ! saved=$(load_salt "$round_id" 2>/dev/null); then
                    echo "  Round $round_id: no commitment found, skipping reveal" >&2
                    continue
                fi

                local verdict confidence salt
                verdict=$(echo "$saved" | cut -d'|' -f1)
                confidence=$(echo "$saved" | cut -d'|' -f2)
                salt=$(echo "$saved" | cut -d'|' -f3)

                echo "  Round $round_id: revealing $verdict ($confidence BPS)" >&2

                local result
                result=$(cmd_reveal "$round_id" "$verdict" "$confidence" "$salt" 2>&1) || true
                # Always clear salt after reveal attempt (even if tx failed)
                clear_salt "$round_id"
                echo "  Round $round_id: revealed" >&2
                ;;

            aggregation|complete)
                # Clean up any stale salts
                clear_salt "$round_id"
                ;;
        esac
    done

    # Report
    echo "{\"status\":\"verified\",\"active_rounds\":$round_count}"
}

cmd_reflect() {
    echo "Reflecting on past observations..." >&2

    local state
    state=$(load_state)
    local height
    height=$(get_current_height)

    # Process pending observations
    local updated=0 still_pending=0
    state=$(echo "$state" | python3 -c "
import sys, json, subprocess, os

NODE_URL = os.environ.get('NODE_URL', 'http://localhost:1317')
state = json.load(sys.stdin)
updated = 0
still_pending = 0

for obs in state['observations']:
    if obs['status'] != 'pending':
        continue
    claim_id = obs['claim_id']
    # Skip tx-hash references (can't query those as claims)
    if claim_id.startswith('tx:'):
        still_pending += 1
        continue
    try:
        import urllib.request
        url = f'{NODE_URL}/legible/knowledge/v1/claim/{claim_id}'
        resp = urllib.request.urlopen(url, timeout=5).read()
        data = json.loads(resp)
        claim = data.get('claim', {})
        status = claim.get('status', 'pending')
        if status in ('accepted', 'verified', 'fact'):
            obs['status'] = 'accepted'
            obs['outcome'] = 'accepted'
            state['trends']['accuracy']['accepted'] = state['trends']['accuracy'].get('accepted', 0) + 1
            updated += 1
        elif status in ('rejected', 'slashed'):
            obs['status'] = 'rejected'
            obs['outcome'] = 'rejected'
            state['trends']['accuracy']['rejected'] = state['trends']['accuracy'].get('rejected', 0) + 1
            updated += 1
        elif status == 'expired':
            obs['status'] = 'expired'
            obs['outcome'] = 'expired'
            state['trends']['accuracy']['expired'] = state['trends']['accuracy'].get('expired', 0) + 1
            updated += 1
        else:
            still_pending += 1
    except Exception:
        still_pending += 1

state['last_reflect_height'] = int('$height')

# Prune: keep last 50 observations
if len(state['observations']) > 50:
    state['observations'] = state['observations'][-50:]

result = {'state': state, 'updated': updated, 'still_pending': still_pending}
print(json.dumps(result))
" 2>/dev/null)

    # Extract updated state and summary
    local new_state summary
    new_state=$(echo "$state" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin)['state']))")
    updated=$(echo "$state" | python3 -c "import sys,json; print(json.load(sys.stdin)['updated'])")
    still_pending=$(echo "$state" | python3 -c "import sys,json; print(json.load(sys.stdin)['still_pending'])")

    save_state "$new_state"

    # Compute accuracy
    local accuracy_info
    accuracy_info=$(echo "$new_state" | python3 -c "
import sys, json
s = json.load(sys.stdin)
acc = s['trends']['accuracy']
submitted = acc.get('submitted', 0)
accepted = acc.get('accepted', 0)
rejected = acc.get('rejected', 0)
expired = acc.get('expired', 0)
pct = (accepted * 100) // submitted if submitted > 0 else 0
print(json.dumps({'submitted': submitted, 'accepted': accepted, 'rejected': rejected, 'expired': expired, 'accuracy_pct': pct}))
")

    echo "Updated $updated observation(s), $still_pending still pending" >&2

    # Output summary
    echo "$accuracy_info" | python3 -c "
import sys, json
acc = json.load(sys.stdin)
result = {
    'status': 'reflected',
    'height': int('$height'),
    'updated': int('$updated'),
    'still_pending': int('$still_pending'),
    'accuracy': acc
}
print(json.dumps(result, indent=2))
"
}

cmd_watch() {
    local interval="${1:-3}"  # Poll interval in seconds (default ~1 block)

    echo "Starting verification watch daemon (poll every ${interval}s)..." >&2
    echo "Press Ctrl+C to stop." >&2

    trap 'echo "Watch daemon stopped." >&2; exit 0' INT TERM

    local cycle=0
    while true; do
        cycle=$((cycle + 1))
        local result
        result=$(cmd_verify 2>/dev/null) || true

        local status
        status=$(echo "$result" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status','error'))" 2>/dev/null || echo "error")

        if [ "$status" = "verified" ]; then
            local count
            count=$(echo "$result" | python3 -c "import sys,json; print(json.load(sys.stdin).get('active_rounds',0))" 2>/dev/null || echo "0")
            echo "[cycle $cycle] Processed $count active round(s)" >&2
        fi

        # Periodic salt cleanup (every 20 cycles)
        if [ $((cycle % 20)) -eq 0 ]; then
            cleanup_stale_salts
        fi

        sleep "$interval"
    done
}

# --- Dispatch ---

case "${1:-help}" in
    submit-claim)    shift; cmd_submit_claim "$@" ;;
    commit)          shift; cmd_commit "$@" ;;
    reveal)          shift; cmd_reveal "$@" ;;
    hash-commitment) shift; cmd_hash_commitment "$@" ;;
    observe)         shift; cmd_observe "$@" ;;
    verify)          shift; cmd_verify "$@" ;;
    watch)           shift; cmd_watch "$@" ;;
    help|--help|-h)
        echo "Usage: lgm-tx.sh <command> [args...]"
        echo ""
        echo "Commands:"
        echo "  submit-claim <text> <domain> <category> <stake-ulgm> [flags]"
        echo "  commit <round-id> <verdict> <confidence-bps> <salt>"
        echo "  reveal <round-id> <verdict> <confidence-bps> <salt> [flags]"
        echo "  hash-commitment <round-id> <verdict> <confidence-bps> <salt>"
        echo "  observe  — read vitals and submit a protocol_observation claim"
        echo "  verify   — poll active rounds and auto-commit/reveal"
        echo "  watch [interval] — continuous verification daemon (default: 3s)"
        echo ""
        echo "Environment: LGM_NODE_URL, LGM_HOME, LGM_FROM, LGM_CHAIN_ID"
        ;;
    *)
        echo "Unknown command: $1" >&2
        exit 1
        ;;
esac
