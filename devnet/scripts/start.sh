#!/bin/bash
#
# Legible Money Devnet Quick Start (backward-compat wrapper)
#
# Delegates to devnet-ctl.sh. Kept for backward compatibility.
#
# Usage: ./start.sh [options]
#   --init      Initialize devnet first (generate keys)
#   --full      Include fullnode
#   --explorer  Include block explorer
#   --monitor   Include Prometheus + Grafana
#   --all       Include all optional services
#   --reset     Reset all data and start fresh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CTL="$SCRIPT_DIR/devnet-ctl.sh"

# Translate flags to devnet-ctl commands
RESET=false
INIT=false
START_ARGS=()

for arg in "$@"; do
  case "$arg" in
    --reset)    RESET=true ;;
    --init)     INIT=true ;;
    --full)     START_ARGS+=("--full") ;;
    --explorer) START_ARGS+=("--explorer") ;;
    --monitor)  START_ARGS+=("--monitor") ;;
    --all)      START_ARGS+=("--all") ;;
    *)
      echo "Unknown option: $arg"
      exit 1
      ;;
  esac
done

if [ "$RESET" = true ]; then
  "$CTL" reset
fi

if [ "$INIT" = true ]; then
  # Force re-initialization by removing existing secrets
  DEVNET_DIR="$(dirname "$SCRIPT_DIR")"
  PROJECT_DIR="$(dirname "$DEVNET_DIR")"
  (cd "$PROJECT_DIR" && npx tsx devnet/scripts/init-devnet.ts)
fi

exec "$CTL" start "${START_ARGS[@]}"
