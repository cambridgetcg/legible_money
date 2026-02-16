#!/usr/bin/env bash
#
# Legible Money — DevNet Health Check
#
# Verifies that the Cosmos SDK devnet is operational.
# Used by CI/CD to validate devnet before running integration tests.
#
# Usage:
#   devnet-healthcheck.sh                   Check default (localhost:26657)
#   devnet-healthcheck.sh <rpc_url>         Check specific endpoint
#   devnet-healthcheck.sh --full            Run all checks including tx test
#
# Exit codes:
#   0 = healthy
#   1 = unhealthy

set -euo pipefail

RPC_URL="${1:-http://localhost:26657}"
REST_URL="${2:-http://localhost:1317}"
FULL_CHECK=false

if [ "${1:-}" = "--full" ]; then
  FULL_CHECK=true
  RPC_URL="http://localhost:26657"
  REST_URL="http://localhost:1317"
fi

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}PASS${NC} $*"; }
fail() { echo -e "${RED}FAIL${NC} $*"; FAILURES=$((FAILURES + 1)); }
warn() { echo -e "${YELLOW}WARN${NC} $*"; }
info() { echo -e "     $*"; }

FAILURES=0
CHECKS=0

echo "=== Legible Money DevNet Health Check ==="
echo "RPC:  $RPC_URL"
echo "REST: $REST_URL"
echo ""

# --- Check 1: Node reachable ---
CHECKS=$((CHECKS + 1))
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$RPC_URL/status" 2>/dev/null || echo "000")
if [ "$STATUS" = "200" ]; then
  pass "Node reachable (HTTP $STATUS)"
else
  fail "Node unreachable (HTTP $STATUS)"
  echo ""
  echo "Result: $FAILURES/$CHECKS checks failed"
  exit 1
fi

# --- Check 2: Block production ---
CHECKS=$((CHECKS + 1))
HEIGHT=$(curl -s "$RPC_URL/status" | python3 -c "import sys,json; print(json.load(sys.stdin)['result']['sync_info']['latest_block_height'])" 2>/dev/null || echo "0")
if [ "$HEIGHT" -gt 0 ] 2>/dev/null; then
  pass "Block production active (height: $HEIGHT)"
else
  fail "No blocks produced (height: $HEIGHT)"
fi

# --- Check 3: Not catching up ---
CHECKS=$((CHECKS + 1))
CATCHING_UP=$(curl -s "$RPC_URL/status" | python3 -c "import sys,json; print(json.load(sys.stdin)['result']['sync_info']['catching_up'])" 2>/dev/null || echo "True")
if [ "$CATCHING_UP" = "False" ]; then
  pass "Node synced (not catching up)"
else
  fail "Node still catching up"
fi

# --- Check 4: Chain ID ---
CHECKS=$((CHECKS + 1))
CHAIN_ID=$(curl -s "$RPC_URL/status" | python3 -c "import sys,json; print(json.load(sys.stdin)['result']['node_info']['network'])" 2>/dev/null || echo "unknown")
if echo "$CHAIN_ID" | grep -q "lgm"; then
  pass "Chain ID: $CHAIN_ID"
else
  fail "Unexpected chain ID: $CHAIN_ID"
fi

# --- Check 5: REST API ---
CHECKS=$((CHECKS + 1))
REST_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$REST_URL/cosmos/base/tendermint/v1beta1/node_info" 2>/dev/null || echo "000")
if [ "$REST_STATUS" = "200" ]; then
  pass "REST API reachable (HTTP $REST_STATUS)"
else
  warn "REST API not reachable (HTTP $REST_STATUS) — may need enabling"
fi

# --- Check 6: Validators ---
CHECKS=$((CHECKS + 1))
VALIDATORS=$(curl -s "$RPC_URL/validators" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['result']['validators']))" 2>/dev/null || echo "0")
if [ "$VALIDATORS" -gt 0 ] 2>/dev/null; then
  pass "Active validators: $VALIDATORS"
else
  fail "No active validators"
fi

# --- Check 7: Block time ---
CHECKS=$((CHECKS + 1))
if [ "$HEIGHT" -gt 2 ] 2>/dev/null; then
  BLOCK_TIME=$(curl -s "$RPC_URL/status" | python3 -c "
import sys,json
from datetime import datetime
si = json.load(sys.stdin)['result']['sync_info']
t = si['latest_block_time'][:26] + 'Z'
print(t)
" 2>/dev/null || echo "unknown")
  if [ "$BLOCK_TIME" != "unknown" ]; then
    pass "Latest block time: $BLOCK_TIME"
  else
    warn "Could not parse block time"
  fi
else
  warn "Not enough blocks to check timing"
fi

# --- Full checks (tx test) ---
if [ "$FULL_CHECK" = true ]; then
  echo ""
  echo "--- Extended Checks ---"

  # Check 8: Bank module query
  CHECKS=$((CHECKS + 1))
  SUPPLY=$(curl -s "$REST_URL/cosmos/bank/v1beta1/supply" 2>/dev/null | python3 -c "
import sys,json
try:
    d=json.load(sys.stdin)
    for s in d.get('supply',[]):
        if s['denom'] == 'ulgm':
            print(int(s['amount'])/1000000)
except: print('0')
" 2>/dev/null || echo "0")
  if [ "$SUPPLY" != "0" ]; then
    pass "Total supply: ${SUPPLY} LGM"
  else
    fail "Could not query total supply"
  fi

  # Check 9: Staking module
  CHECKS=$((CHECKS + 1))
  BONDED=$(curl -s "$REST_URL/cosmos/staking/v1beta1/pool" 2>/dev/null | python3 -c "
import sys,json
try:
    d=json.load(sys.stdin)
    print(int(d['pool']['bonded_tokens'])/1000000)
except: print('0')
" 2>/dev/null || echo "0")
  if [ "$BONDED" != "0" ]; then
    pass "Bonded tokens: ${BONDED} LGM"
  else
    fail "No bonded tokens"
  fi
fi

# --- Summary ---
echo ""
PASSED=$((CHECKS - FAILURES))
if [ "$FAILURES" -eq 0 ]; then
  echo -e "${GREEN}All $CHECKS checks passed${NC}"
  exit 0
else
  echo -e "${RED}$FAILURES/$CHECKS checks failed${NC}"
  exit 1
fi
