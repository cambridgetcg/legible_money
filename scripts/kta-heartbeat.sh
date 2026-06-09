#!/usr/bin/env bash
# kta-heartbeat.sh — Submit a heartbeat beat-claim to Zerone via KTA HeartbeatAdapter.
# Queues offline-tolerantly; flushes automatically if chain comes online.
#
# Usage:
#   kta-heartbeat.sh --beat-id asha-beat-20260422-114115 --chain-status OFFLINE --duties "verification,farm"
#   kta-heartbeat.sh --flush          # drain queue (call after chain-health.sh confirms ONLINE)

LOVE_DIR="${LOVE_DIR:-$HOME/Love}"
INSTANCE="${INSTANCE:-asha}"

usage() {
    echo "Usage: $0 --beat-id <id> --chain-status <ONLINE|OFFLINE> --duties <str>"
    echo "       $0 --flush [--instance <name>]"
    exit 1
}

BEAT_ID=""
CHAIN_STATUS="OFFLINE"
DUTIES=""
FLUSH=0

while [[ $# -gt 0 ]]; do
    case $1 in
        --beat-id)    BEAT_ID="$2";     shift 2 ;;
        --chain-status) CHAIN_STATUS="$2"; shift 2 ;;
        --duties)     DUTIES="$2";      shift 2 ;;
        --instance)   INSTANCE="$2";   shift 2 ;;
        --flush)      FLUSH=1;          shift ;;
        *)            usage ;;
    esac
done

if [[ $FLUSH -eq 1 ]]; then
    python3 "$LOVE_DIR/tools/kta.py" flush --engine "heartbeat-$INSTANCE" 2>/dev/null
    exit $?
fi

[[ -z "$BEAT_ID" ]] && usage

python3 - <<EOF
import sys
sys.path.insert(0, '$LOVE_DIR')
from tools.kta import HeartbeatAdapter
adapter = HeartbeatAdapter(instance='$INSTANCE')
result = adapter.submit_beat(
    beat_id='$BEAT_ID',
    chain_status='$CHAIN_STATUS',
    duties='$DUTIES',
)
status = result.get('status', '?')
if status == 'submitted':
    print(f"KTA: beat submitted | tx: {result.get('tx','?')[:20]}")
elif status == 'queued':
    print(f"KTA: beat queued (chain offline) | hash: {result.get('hash','?')[:16]}")
elif status == 'duplicate':
    print(f"KTA: beat already recorded | hash: {result.get('hash','?')[:16]}")
else:
    print(f"KTA: {result}")
EOF
