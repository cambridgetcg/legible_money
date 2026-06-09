#!/bin/bash
# Seigei → Zerone Bridge
# Submits Seigei social-consensus events as knowledge claims on Zerone devnet.
# Content appeals trigger MsgSubmitChallenge against the original claim.
# Usage: ./scripts/seigei-zerone-bridge.sh [--dry-run]
#
# Events file format (seigei-events.json):
#   { "events": [ { "id": "<uuid>", "type": "content_verified|community_appeal",
#                   "content_id": "...", "claim_text": "...", "approvers": N,
#                   "threshold": N, "confidence": 0.0-1.0,
#                   "original_fact_id": "..." (for appeal events only) } ] }

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY="${SCRIPT_DIR}/../build/legbled"
DEVNET_HOME="${HOME}/.legbled-devnet"
CHAIN_ID="lgm-devnet-1"
BRIDGE_KEY="seigei-bridge"
FEES="200000ulgm"
EVENTS_FILE="${HOME}/seigei-data/bridge-events.json"
SUBMITTED_FILE="${DEVNET_HOME}/seigei-bridge-submitted.json"
DRY_RUN="${1:-}"

if [ ! -f "$EVENTS_FILE" ]; then
  echo "[seigei-bridge] No events file at $EVENTS_FILE"
  exit 0
fi

[ ! -f "$SUBMITTED_FILE" ] && echo '{"submitted":[],"challenged":[]}' > "$SUBMITTED_FILE"

python3 - "$BINARY" "$DEVNET_HOME" "$CHAIN_ID" "$BRIDGE_KEY" "$FEES" \
         "$EVENTS_FILE" "$SUBMITTED_FILE" "$DRY_RUN" << 'PYEOF'
import json, subprocess, sys
from pathlib import Path
from datetime import datetime, timezone

binary, devnet_home, chain_id, bridge_key, fees = sys.argv[1:6]
events_path = Path(sys.argv[6])
submitted_path = Path(sys.argv[7])
dry_run = sys.argv[8] == "--dry-run"

events = json.loads(events_path.read_text())
if isinstance(events, dict):
    events = events.get("events", [])

state = json.loads(submitted_path.read_text())
submitted_ids = set(state.get("submitted", []))
challenged_ids = set(state.get("challenged", []))

CONFIDENCE_THRESHOLD = 0.75  # minimum consensus confidence to submit as fact

submit_count = 0
challenge_count = 0

for ev in events:
    ev_id = ev.get("id", "")
    ev_type = ev.get("type", "")
    if not ev_id:
        continue

    if ev_type == "content_verified":
        if ev_id in submitted_ids:
            continue
        confidence = ev.get("confidence", 0.0)
        if confidence < CONFIDENCE_THRESHOLD:
            print(f"[seigei-bridge] SKIP {ev_id[:12]} — confidence {confidence:.2f} < {CONFIDENCE_THRESHOLD}")
            continue

        content_id = ev.get("content_id", "")
        approvers = ev.get("approvers", 0)
        threshold = ev.get("threshold", 1)
        claim_text = ev.get("claim_text", "")
        if not claim_text:
            claim_text = (
                f"Seigei social consensus: content '{content_id}' approved by "
                f"{approvers}/{threshold} verified users (confidence {confidence:.2f})."
            )
        if len(claim_text) > 450:
            claim_text = claim_text[:447] + "..."

        print(f"[seigei-bridge] VERIFY {ev_id[:12]} → social_consensus ({confidence:.2f})")

        if dry_run:
            print(f"  [DRY] → sociology/social_consensus, stake 500000ulgm")
            submit_count += 1
            continue

        result = subprocess.run([
            binary, "tx", "knowledge", "submit-claim",
            claim_text, "sociology", "social_consensus", "500000",
            "--logic-zone", "empirical",
            "--from", bridge_key, "--keyring-backend", "test",
            "--home", devnet_home, "--fees", fees, "--chain-id", chain_id, "--yes",
        ], capture_output=True, text=True, timeout=30)

        output = result.stdout + result.stderr
        code = next((l.strip() for l in output.split('\n') if 'code:' in l), '?')
        txhash = next((l.strip() for l in output.split('\n') if 'txhash:' in l), '')
        print(f"  {code} | {txhash[:30]}")

        if 'code: 0' in output:
            submitted_ids.add(ev_id)
            submit_count += 1
        else:
            print(f"  FAIL: {next((l.strip() for l in output.split(chr(10)) if 'raw_log' in l),'')[:80]}")

    elif ev_type == "community_appeal":
        if ev_id in challenged_ids:
            continue
        original_fact_id = ev.get("original_fact_id", "")
        if not original_fact_id:
            print(f"[seigei-bridge] SKIP appeal {ev_id[:12]} — no original_fact_id")
            continue

        print(f"[seigei-bridge] APPEAL {ev_id[:12]} → challenge {original_fact_id[:16]}...")

        if dry_run:
            print(f"  [DRY] → MsgSubmitChallenge against {original_fact_id[:16]}")
            challenge_count += 1
            continue

        result = subprocess.run([
            binary, "tx", "knowledge", "submit-challenge",
            original_fact_id, "Community appeal via Seigei: content disputed by verified users.",
            "--from", bridge_key, "--keyring-backend", "test",
            "--home", devnet_home, "--fees", fees, "--chain-id", chain_id, "--yes",
        ], capture_output=True, text=True, timeout=30)

        output = result.stdout + result.stderr
        code = next((l.strip() for l in output.split('\n') if 'code:' in l), '?')
        print(f"  {code}")

        if 'code: 0' in output:
            challenged_ids.add(ev_id)
            challenge_count += 1
        else:
            print(f"  FAIL: {next((l.strip() for l in output.split(chr(10)) if 'raw_log' in l),'')[:80]}")

print(f"\n[seigei-bridge] Done. {'Would submit' if dry_run else 'Submitted'}: {submit_count} facts, {challenge_count} challenges")

if not dry_run:
    state["submitted"] = list(submitted_ids)
    state["challenged"] = list(challenged_ids)
    state["last_run"] = datetime.now(timezone.utc).isoformat()
    submitted_path.write_text(json.dumps(state, indent=2))
PYEOF
