#!/usr/bin/env bash
# lgm-status.sh — Query chain and agent status.
#
# Usage:
#   lgm-status.sh chain              — Chain height, block time, validators
#   lgm-status.sh claims <address>   — Pending/active claims for an agent
#   lgm-status.sh rounds             — Active verification rounds
#   lgm-status.sh facts <domain>     — Facts in a domain
#   lgm-status.sh rewards <address>  — Pending rewards
#
# Environment: Same as lgm-register.sh

set -euo pipefail

NODE="${LGM_NODE:-tcp://localhost:26657}"
CHAIN_ID="${LGM_CHAIN_ID:-legible-devnet-1}"
LEGBLED_HOME="${LGM_HOME:-$HOME/.legbled}"

COMMAND="${1:?Usage: lgm-status.sh <chain|claims|rounds|facts|rewards> [args...]}"
shift

LEGBLED="legbled"

case "$COMMAND" in
    chain)
        echo "=== Chain Status ==="
        # Node status
        STATUS=$($LEGBLED status --node "$NODE" 2>/dev/null || echo '{}')
        echo "$STATUS" | python3 -c "
import json, sys
data = json.load(sys.stdin)
si = data.get('sync_info', data.get('SyncInfo', {}))
ni = data.get('node_info', data.get('NodeInfo', {}))
if si:
    print(f'  Chain ID:     {ni.get(\"network\", \"unknown\")}')
    print(f'  Height:       {si.get(\"latest_block_height\", \"?\")}')
    print(f'  Block Time:   {si.get(\"latest_block_time\", \"?\")}')
    print(f'  Catching Up:  {si.get(\"catching_up\", \"?\")}')
else:
    print('  (no sync info)')
" 2>/dev/null || echo "  (query failed — is the node running?)"
        ;;

    claims)
        ADDRESS="${1:?Usage: lgm-status.sh claims <address>}"
        echo "=== Claims for $ADDRESS ==="
        $LEGBLED query knowledge claims-by-submitter "$ADDRESS" \
            --node "$NODE" --output json 2>/dev/null | python3 -c "
import json, sys
data = json.load(sys.stdin)
claims = data.get('claims', [])
for c in claims:
    print(f'  [{c.get(\"id\",\"?\")}] {c.get(\"status\",\"?\")} — {c.get(\"claim_text\",\"\")[:60]}...')
if not claims:
    print('  No claims found')
print(f'  Total: {len(claims)}')
" 2>/dev/null || echo "  (query failed)"
        ;;

    rounds)
        echo "=== Active Verification Rounds ==="
        $LEGBLED query knowledge active-rounds \
            --node "$NODE" --output json 2>/dev/null | python3 -c "
import json, sys
data = json.load(sys.stdin)
rounds = data.get('rounds', [])
for r in rounds:
    print(f'  [{r.get(\"id\",\"?\")}] Phase: {r.get(\"phase\",\"?\")} Claim: {r.get(\"claim_id\",\"?\")} Verifiers: {r.get(\"verifier_count\",0)}')
if not rounds:
    print('  No active rounds')
print(f'  Total active: {len(rounds)}')
" 2>/dev/null || echo "  (query failed)"
        ;;

    facts)
        DOMAIN="${1:?Usage: lgm-status.sh facts <domain>}"
        echo "=== Facts in domain '$DOMAIN' ==="
        $LEGBLED query knowledge facts-by-domain "$DOMAIN" \
            --node "$NODE" --output json 2>/dev/null | python3 -c "
import json, sys
data = json.load(sys.stdin)
facts = data.get('facts', [])
for f in facts:
    conf = int(f.get('confidence', 0)) / 10000
    print(f'  [{f.get(\"id\",\"?\")}] {conf:.1f}% — {f.get(\"claim_text\",\"\")[:60]}')
if not facts:
    print('  No facts found')
print(f'  Total: {len(facts)}')
" 2>/dev/null || echo "  (query failed)"
        ;;

    rewards)
        ADDRESS="${1:?Usage: lgm-status.sh rewards <address>}"
        echo "=== Rewards for $ADDRESS ==="
        # Check distribution rewards
        $LEGBLED query distribution rewards "$ADDRESS" \
            --node "$NODE" --output json 2>/dev/null | python3 -c "
import json, sys
data = json.load(sys.stdin)
total = data.get('total', [])
for t in total:
    if t['denom'] == 'ulgm':
        amt = float(t['amount'])
        print(f'  Pending: {amt:,.0f} ulgm ({amt / 1_000_000:.6f} LGM)')
        sys.exit(0)
print('  No pending rewards')
" 2>/dev/null || echo "  (query failed)"
        ;;

    *)
        echo "Unknown command: $COMMAND"
        echo "Usage: lgm-status.sh <chain|claims|rounds|facts|rewards> [args...]"
        exit 1
        ;;
esac
