#!/usr/bin/env bash
# lgm-balance.sh — Query agent balances and staking info.
#
# Usage: lgm-balance.sh <key-name-or-address>
#
# Shows:
#   - Available balance (ulgm and LGM)
#   - Staked amount
#   - Discovery registration stake
#   - Pending rewards
#
# Environment: Same as lgm-register.sh

set -euo pipefail

NODE="${LGM_NODE:-tcp://localhost:26657}"
CHAIN_ID="${LGM_CHAIN_ID:-legible-devnet-1}"
KEYRING="${LGM_KEYRING:-test}"
LEGBLED_HOME="${LGM_HOME:-$HOME/.legbled}"

INPUT="${1:?Usage: lgm-balance.sh <key-name-or-address>}"

LEGBLED="legbled"

# Resolve address — input can be a key name or bech32 address
if [[ "$INPUT" == lgm1* ]]; then
    ADDRESS="$INPUT"
else
    ADDRESS=$($LEGBLED keys show "$INPUT" --keyring-backend "$KEYRING" --home "$LEGBLED_HOME" -a 2>/dev/null)
    if [ -z "$ADDRESS" ]; then
        echo "ERROR: Key '$INPUT' not found and not a valid address."
        exit 1
    fi
fi

echo "=== Agent Balance ==="
echo "  Address: $ADDRESS"
echo ""

# Bank balance
echo "--- Bank Balance ---"
BALANCE_JSON=$($LEGBLED query bank balances "$ADDRESS" --node "$NODE" --output json 2>/dev/null || echo '{"balances":[]}')
ULGM=$(echo "$BALANCE_JSON" | python3 -c "
import json, sys
data = json.load(sys.stdin)
for b in data.get('balances', []):
    if b['denom'] == 'ulgm':
        amt = int(b['amount'])
        print(f'  ulgm: {amt:,}')
        print(f'  LGM:  {amt / 1_000_000:.6f}')
        sys.exit(0)
print('  ulgm: 0')
print('  LGM:  0.000000')
" 2>/dev/null || echo "  (query failed)")
echo "$ULGM"

# Staking delegations
echo ""
echo "--- Staking ---"
$LEGBLED query staking delegations "$ADDRESS" --node "$NODE" --output json 2>/dev/null | python3 -c "
import json, sys
data = json.load(sys.stdin)
delegations = data.get('delegation_responses', [])
total = 0
for d in delegations:
    amt = int(d['balance']['amount'])
    total += amt
    val = d['delegation']['validator_address']
    print(f'  {val[:20]}... : {amt:,} ulgm')
if not delegations:
    print('  No delegations')
print(f'  Total staked: {total:,} ulgm ({total / 1_000_000:.6f} LGM)')
" 2>/dev/null || echo "  (query failed)"

# Discovery profile
echo ""
echo "--- Discovery Profile ---"
$LEGBLED query discovery agent "$ADDRESS" --node "$NODE" --output json 2>/dev/null | python3 -c "
import json, sys
data = json.load(sys.stdin)
p = data.get('profile', {})
if p:
    print(f'  Name:       {p.get(\"display_name\", \"N/A\")}')
    print(f'  Status:     {p.get(\"status\", \"N/A\")}')
    print(f'  Reputation: {int(p.get(\"reputation_score\", 0)) / 10000:.2f}%')
    print(f'  Stake:      {p.get(\"stake\", \"0\")} ulgm')
    domains = p.get('domains', [])
    print(f'  Domains:    {\", \".join(domains) if domains else \"none\"}')
else:
    print('  Not registered as agent')
" 2>/dev/null || echo "  (query failed or not registered)"

echo ""
