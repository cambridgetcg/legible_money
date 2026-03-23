#!/bin/bash
# Oracle → Zerone Bridge
# Submits oracle signals as protocol_observation claims on Zerone devnet.
# Usage: ./scripts/oracle-zerone-bridge.sh [--dry-run]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY="${SCRIPT_DIR}/../build/legbled"
DEVNET_HOME="${HOME}/.legbled-devnet"
CHAIN_ID="lgm-devnet-1"
BRIDGE_KEY="bridge-agent"
FEES="200000ulgm"
SIGNALS_FILE="${HOME}/Desktop/prediction-markets/data/pending-signals.json"
SUBMITTED_FILE="${DEVNET_HOME}/bridge-submitted.json"
DRY_RUN="${1:-}"

if [ ! -f "$SIGNALS_FILE" ]; then
  echo "[bridge] No signals file"
  exit 0
fi

[ ! -f "$SUBMITTED_FILE" ] && echo '{"submitted":[]}' > "$SUBMITTED_FILE"

python3 - "$BINARY" "$DEVNET_HOME" "$CHAIN_ID" "$BRIDGE_KEY" "$FEES" \
         "$SIGNALS_FILE" "$SUBMITTED_FILE" "$DRY_RUN" << 'PYEOF'
import json, subprocess, sys
from pathlib import Path
from datetime import datetime, timezone

binary, devnet_home, chain_id, bridge_key, fees = sys.argv[1:6]
signals_path = Path(sys.argv[6])
submitted_path = Path(sys.argv[7])
dry_run = sys.argv[8] == "--dry-run"

signals = json.loads(signals_path.read_text())
if isinstance(signals, dict):
    signals = signals.get("signals", [])

submitted_data = json.loads(submitted_path.read_text())
submitted_ids = set(submitted_data.get("submitted", []))

domain_map = {
    "GEO": "sociology", "CRY": "economics", "TECH": "computer_science",
    "POL": "sociology", "ECON": "economics", "SCI": "physics", "OTHER": "general",
}

new_count = 0
for s in signals:
    sig_id = s.get("id", "")
    if not sig_id or sig_id in submitted_ids:
        continue

    market = s.get("market", s.get("market_question", ""))
    if not market:
        continue

    trade = s.get("trade", "")
    score = s.get("score", 0)
    magnitude = s.get("magnitude", 0)
    domain_code = s.get("domain", "OTHER")
    zerone_domain = domain_map.get(domain_code, "general")

    claim_text = (
        f"Prediction market signal ({domain_code}): '{market}'. "
        f"Trade: {trade}. Magnitude: {magnitude:.1%}. Score: {score}/10."
    )
    if len(claim_text) > 450:
        claim_text = claim_text[:447] + "..."

    print(f"[bridge] {sig_id[:12]} → {market[:55]}...")

    if dry_run:
        print(f"  [DRY] → {zerone_domain}/protocol_observation")
        new_count += 1
        continue

    result = subprocess.run([
        binary, "tx", "knowledge", "submit-claim",
        claim_text, zerone_domain, "protocol_observation", "1000000",
        "--logic-zone", "empirical",
        "--from", bridge_key, "--keyring-backend", "test",
        "--home", devnet_home, "--fees", fees, "--chain-id", chain_id, "--yes",
    ], capture_output=True, text=True, timeout=30)

    output = result.stdout + result.stderr
    code = next((l.strip() for l in output.split('\n') if 'code:' in l), '?')
    txhash = next((l.strip() for l in output.split('\n') if 'txhash:' in l), '')
    print(f"  {code} | {txhash[:30]}")

    if 'code: 0' in output:
        submitted_ids.add(sig_id)
        new_count += 1
    else:
        print(f"  FAIL: {next((l.strip() for l in output.split(chr(10)) if 'raw_log' in l),'')[:80]}")

print(f"\n[bridge] Done. {'Would submit' if dry_run else 'Submitted'}: {new_count}")

if not dry_run:
    submitted_data["submitted"] = list(submitted_ids)
    submitted_data["last_run"] = datetime.now(timezone.utc).isoformat()
    submitted_path.write_text(json.dumps(submitted_data, indent=2))
PYEOF
