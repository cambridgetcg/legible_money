#!/bin/bash
# Auto-verifier for Zerone devnet — polls for active rounds and submits commit+reveal
# Usage: ./scripts/auto-verifier.sh [--watch]
# Without --watch: one-shot, handles any current active rounds
# With --watch: continuous loop, waits for new rounds

set -euo pipefail

BINARY="build/legbled"
DEVNET_HOME="${HOME}/.legbled-devnet"
CHAIN_ID="lgm-devnet-1"
VERIFIER_KEY="validator"
FEES="200000ulgm"
VERDICT="accept"
CONFIDENCE="850000"  # 85% confidence

# Compute commitment hash
compute_hash() {
  local verdict="$1"
  local confidence="$2"
  local salt="$3"
  echo -n "LGM.commitment.v1:${verdict}|${confidence}|${salt}" | sha256sum | awk '{print $1}'
}

# Submit commit for a round
submit_commit() {
  local round_id="$1"
  local salt="$2"
  local hash
  hash=$(compute_hash "$VERDICT" "$CONFIDENCE" "$salt")
  
  echo "[auto-verifier] Committing to round ${round_id:0:12}... (hash: ${hash:0:16}...)"
  
  "$BINARY" tx knowledge submit-commitment \
    "$round_id" "$hash" \
    --from "$VERIFIER_KEY" \
    --keyring-backend test \
    --home "$DEVNET_HOME" \
    --fees "$FEES" \
    --chain-id "$CHAIN_ID" \
    --yes 2>&1 | grep -E "txhash|code:|raw_log" || true
}

# Submit reveal for a round
submit_reveal() {
  local round_id="$1"
  local salt="$2"
  
  echo "[auto-verifier] Revealing for round ${round_id:0:12}..."
  
  "$BINARY" tx knowledge submit-reveal \
    "$round_id" "$VERDICT" "$CONFIDENCE" "$salt" \
    --from "$VERIFIER_KEY" \
    --keyring-backend test \
    --home "$DEVNET_HOME" \
    --fees "$FEES" \
    --chain-id "$CHAIN_ID" \
    --yes 2>&1 | grep -E "txhash|code:|raw_log" || true
}

# Get current block height
get_height() {
  "$BINARY" status --home "$DEVNET_HOME" 2>/dev/null | python3 -c \
    "import sys,json; d=json.load(sys.stdin); print(d['sync_info']['latest_block_height'])" 2>/dev/null || \
    curl -s http://localhost:26657/status | python3 -c \
    "import sys,json; d=json.load(sys.stdin); print(d['result']['sync_info']['latest_block_height'])" 2>/dev/null
}

process_rounds() {
  local rounds
  rounds=$("$BINARY" query knowledge active-rounds --home "$DEVNET_HOME" 2>/dev/null)
  
  if echo "$rounds" | grep -q "round_id:" ; then
    local current_height
    current_height=$(get_height)
    echo "[auto-verifier] Block $current_height — active rounds found"
    
    # Extract round IDs and commit deadlines
    echo "$rounds" | python3 << 'PYEOF'
import sys, re

content = sys.stdin.read()
# Parse round blocks
rounds_raw = content.split('- assignments:') if '- assignments:' in content else content.split('  round_id:')

import subprocess, hashlib, os

binary = "./build/legbled"
devnet_home = os.path.expanduser("~/.legbled-devnet")
chain_id = "lgm-devnet-1"
verifier_key = "validator"
fees = "200000ulgm"
verdict = "accept"
confidence = "850000"
salt = "auto-verifier-salt-001"

def compute_hash(v, c, s):
    payload = f"LGM.commitment.v1:{v}|{c}|{s}"
    return hashlib.sha256(payload.encode()).hexdigest()

# Get all round_ids and their commit_deadline_block
round_ids = re.findall(r'round_id: ([a-f0-9]{64})', content)
commit_ends = re.findall(r'commit_phase_end_block: "(\d+)"', content)
reveal_ends = re.findall(r'reveal_phase_end_block: "(\d+)"', content)

current_height_out = subprocess.run(
    ["curl", "-s", "http://localhost:26657/status"],
    capture_output=True, text=True
)
import json
current_height = int(json.loads(current_height_out.stdout)['result']['sync_info']['latest_block_height'])

print(f"Current block: {current_height}")
print(f"Rounds found: {len(round_ids)}")

for i, (rid, ce, re_) in enumerate(zip(round_ids, commit_ends, reveal_ends)):
    ce_int = int(ce)
    re_int = int(re_)
    print(f"\nRound {rid[:16]}...")
    print(f"  Commit ends: {ce} (in {ce_int - current_height} blocks)")
    print(f"  Reveal ends: {re_} (in {re_int - current_height} blocks)")
    
    if current_height <= ce_int:
        print(f"  → In COMMIT phase, submitting...")
        h = compute_hash(verdict, confidence, salt)
        r = subprocess.run([
            binary, "tx", "knowledge", "submit-commitment",
            rid, h,
            "--from", verifier_key, "--keyring-backend", "test",
            "--home", devnet_home, "--fees", fees,
            "--chain-id", chain_id, "--yes"
        ], capture_output=True, text=True)
        out = r.stdout + r.stderr
        for line in out.split('\n'):
            if any(k in line for k in ['txhash', 'code:', 'raw_log']):
                print(f"  {line.strip()}")
    elif current_height <= re_int:
        print(f"  → In REVEAL phase, submitting...")
        r = subprocess.run([
            binary, "tx", "knowledge", "submit-reveal",
            rid, verdict, confidence, salt,
            "--from", verifier_key, "--keyring-backend", "test",
            "--home", devnet_home, "--fees", fees,
            "--chain-id", chain_id, "--yes"
        ], capture_output=True, text=True)
        out = r.stdout + r.stderr
        for line in out.split('\n'):
            if any(k in line for k in ['txhash', 'code:', 'raw_log']):
                print(f"  {line.strip()}")
    else:
        print(f"  → Round expired (too late)")
PYEOF
  else
    echo "[auto-verifier] No active rounds"
  fi
}

# Main
if [[ "${1:-}" == "--watch" ]]; then
  echo "[auto-verifier] Watching for rounds (poll every 2s)..."
  while true; do
    process_rounds
    sleep 2
  done
else
  process_rounds
fi
