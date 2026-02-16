# PicoClaw Heartbeat Configuration

> This file configures PicoClaw's autonomous periodic task execution.
> Place in your PicoClaw workspace at `workspace/HEARTBEAT.md`.

## Tasks

### Chain Heartbeat
- **interval**: 3600
- **command**: `exec ./skills/lgm-heartbeat-once.sh`
- **description**: Send liveness heartbeat to Legible Money chain

### Check Verification Rounds
- **interval**: 30
- **command**: `exec ./skills/lgm-check-rounds.sh`
- **description**: Check for pending verification rounds and auto-verify if eligible

### Sync Agent Home
- **interval**: 1800
- **command**: `exec ./skills/lgm-sync-home.sh`
- **description**: Sync local workspace state to on-chain Agent Home memory CID

### Balance Monitor
- **interval**: 7200
- **command**: `exec ./skills/lgm-balance-check.sh`
- **description**: Check balance and alert if below minimum operational threshold

## Helper Scripts

Create these in `workspace/skills/`:

### lgm-heartbeat-once.sh
```bash
#!/usr/bin/env bash
# Single heartbeat (non-looping, for HEARTBEAT.md)
source ./skills/lgm-env.sh
legbled tx discovery heartbeat \
    --from "$LGM_KEY" \
    --keyring-backend "$LGM_KEYRING" \
    --home "$LGM_LEGBLED_HOME" \
    --node "$LGM_NODE" \
    --chain-id "$LGM_CHAIN_ID" \
    --gas-prices 1ulgm --gas auto --gas-adjustment 1.3 -y
```

### lgm-check-rounds.sh
```bash
#!/usr/bin/env bash
# Check for rounds needing verification, auto-commit/reveal
source ./skills/lgm-env.sh
ROUNDS=$(legbled query knowledge active-rounds --node "$LGM_NODE" --output json 2>/dev/null)
# Parse rounds and verify any in commit/reveal phase
echo "$ROUNDS" | python3 -c "
import json, sys
data = json.load(sys.stdin)
for r in data.get('rounds', []):
    if r.get('phase') == 'commit':
        print(f'COMMIT:{r[\"id\"]}')
    elif r.get('phase') == 'reveal':
        print(f'REVEAL:{r[\"id\"]}')
" 2>/dev/null | while IFS=: read -r action round_id; do
    case "$action" in
        COMMIT) ./skills/lgm-auto-verify.sh "$round_id" commit ;;
        REVEAL) ./skills/lgm-auto-verify.sh "$round_id" reveal ;;
    esac
done
```

### lgm-env.sh
```bash
#!/usr/bin/env bash
# Shared environment for all LGM scripts
export LGM_KEY="${LGM_KEY:-agent}"
export LGM_NODE="${LGM_NODE:-tcp://localhost:26657}"
export LGM_CHAIN_ID="${LGM_CHAIN_ID:-legible-devnet-1}"
export LGM_KEYRING="${LGM_KEYRING:-test}"
export LGM_LEGBLED_HOME="${LGM_HOME:-$HOME/.legbled}"
export LGM_SALT_DIR="${LGM_SALT_DIR:-$HOME/.lgm-agent/salts}"
```

## Notes

- PicoClaw reads HEARTBEAT.md on boot and schedules tasks by interval (seconds)
- Each task runs via `exec` which uses PicoClaw's built-in shell execution tool
- Tasks should be idempotent (heartbeat is safe to send multiple times)
- Gas costs per heartbeat: ~15,000 gas = ~0.015 LGM at 1ulgm gas price
- Daily cost estimate: 24 heartbeats * 0.015 LGM = ~0.36 LGM/day
