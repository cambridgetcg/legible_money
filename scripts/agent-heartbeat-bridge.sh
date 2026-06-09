#!/usr/bin/env bash
# agent-heartbeat-bridge.sh — Zerone identity attestation per heartbeat beat
# Wraps tools/kta.py HeartbeatAdapter. One activity claim per invocation.
# Usage: agent-heartbeat-bridge.sh [--instance INSTANCE] [--beat-id ID] [--chain-status STATUS] [--duties DUTIES] [--dry-run] [--flush]

set -euo pipefail

LOVE_DIR="${LOVE_DIR:-$HOME/Love}"
INSTANCE="asha"
BEAT_ID="${BEAT_ID:-unknown}"
CHAIN_STATUS="${CHAIN_STATUS:-UNKNOWN}"
DUTIES="${DUTIES:-heartbeat}"
DRY_RUN=0
FLUSH=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --instance)   INSTANCE="$2"; shift 2 ;;
        --beat-id)    BEAT_ID="$2"; shift 2 ;;
        --chain-status) CHAIN_STATUS="$2"; shift 2 ;;
        --duties)     DUTIES="$2"; shift 2 ;;
        --dry-run)    DRY_RUN=1; shift ;;
        --flush)      FLUSH=1; shift ;;
        *) echo "Unknown arg: $1"; exit 1 ;;
    esac
done

PYTHON="$(command -v python3)"
KTA="$LOVE_DIR/tools/kta.py"

if [[ ! -f "$KTA" ]]; then
    echo "BRIDGE: kta.py not found at $KTA" >&2
    exit 1
fi

if [[ "$FLUSH" -eq 1 ]]; then
    echo "BRIDGE: flushing queued heartbeat claims for $INSTANCE..."
    "$PYTHON" - <<EOF
import sys
sys.path.insert(0, "$LOVE_DIR")
from tools.kta import HeartbeatAdapter
kta = HeartbeatAdapter("$INSTANCE")
result = kta.flush_queue()
print(f"[bridge/{$INSTANCE}] flush: {result}")
EOF
    exit 0
fi

if [[ "$DRY_RUN" -eq 1 ]]; then
    echo "BRIDGE: dry-run — would submit heartbeat claim for $INSTANCE beat $BEAT_ID (chain: $CHAIN_STATUS, duties: $DUTIES)"
    exit 0
fi

"$PYTHON" - <<EOF
import sys
sys.path.insert(0, "$LOVE_DIR")
from tools.kta import HeartbeatAdapter
kta = HeartbeatAdapter("$INSTANCE")
result = kta.submit_beat("$BEAT_ID", "$CHAIN_STATUS", "$DUTIES")
status = result.get("status", "unknown")
if status == "submitted":
    print(f"BRIDGE: heartbeat $BEAT_ID submitted | tx: {result.get('tx','?')[:20]}")
elif status == "queued":
    print(f"BRIDGE: chain offline, heartbeat $BEAT_ID queued | hash: {result.get('hash','?')[:12]}")
elif status == "duplicate":
    print(f"BRIDGE: heartbeat $BEAT_ID already submitted (dedup) | hash: {result.get('hash','?')[:12]}")
elif status == "rejected":
    print(f"BRIDGE: heartbeat $BEAT_ID rejected | reason: {result.get('reason','?')}", file=sys.stderr)
    sys.exit(1)
else:
    print(f"BRIDGE: heartbeat $BEAT_ID status={status}")
EOF
