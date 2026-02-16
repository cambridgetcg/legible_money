#!/usr/bin/env bash
#
# Entrypoint for legbled Docker container.
#
# Environment variables:
#   MONIKER          - Node moniker (default: legible-node)
#   CHAIN_ID         - Chain ID (default: lgm-devnet-1)
#   GENESIS_URL      - URL to download genesis.json from (optional)
#   SEEDS            - Seed nodes (comma-separated)
#   PERSISTENT_PEERS - Persistent peers (comma-separated)
#   MINIMUM_GAS      - Minimum gas prices (default: 0.025ulgm)
#   ENABLE_API       - Enable REST API (default: true)
#   ENABLE_GRPC      - Enable gRPC (default: true)
#   UNSAFE_CORS      - Enable unsafe CORS (default: false)
#   PRUNING          - Pruning strategy: default|nothing|everything (default: default)
#   SNAPSHOT_INTERVAL - Snapshot interval in blocks (default: 0 = disabled)
#

set -euo pipefail

CHAIN_ID="${CHAIN_ID:-lgm-devnet-1}"
MONIKER="${MONIKER:-legible-node}"
MINIMUM_GAS="${MINIMUM_GAS:-0.025ulgm}"
ENABLE_API="${ENABLE_API:-true}"
ENABLE_GRPC="${ENABLE_GRPC:-true}"
UNSAFE_CORS="${UNSAFE_CORS:-false}"
PRUNING="${PRUNING:-default}"
SNAPSHOT_INTERVAL="${SNAPSHOT_INTERVAL:-0}"
HOME_DIR="${LGM_HOME:-/root/.legbled}"

# ──── Initialize if needed ────
if [ ! -f "$HOME_DIR/config/genesis.json" ]; then
  echo "[entrypoint] Initializing node: moniker=$MONIKER chain-id=$CHAIN_ID"
  legbled init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR" 2>/dev/null

  # Download genesis if URL provided
  if [ -n "${GENESIS_URL:-}" ]; then
    echo "[entrypoint] Downloading genesis from $GENESIS_URL"
    curl -sSL "$GENESIS_URL" > "$HOME_DIR/config/genesis.json"
  fi
fi

# ──── Patch configs ────
CONFIG_TOML="$HOME_DIR/config/config.toml"
APP_TOML="$HOME_DIR/config/app.toml"

# Seeds and persistent peers
if [ -n "${SEEDS:-}" ]; then
  sed -i "s/^seeds = .*/seeds = \"$SEEDS\"/" "$CONFIG_TOML"
fi

if [ -n "${PERSISTENT_PEERS:-}" ]; then
  sed -i "s/^persistent_peers = .*/persistent_peers = \"$PERSISTENT_PEERS\"/" "$CONFIG_TOML"
fi

# Listen on all interfaces inside container
sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|' "$CONFIG_TOML"
sed -i 's|laddr = "tcp://127.0.0.1:26656"|laddr = "tcp://0.0.0.0:26656"|' "$CONFIG_TOML"

# CORS for devnet
if [ "$UNSAFE_CORS" = "true" ]; then
  sed -i 's/cors_allowed_origins = \[\]/cors_allowed_origins = ["*"]/' "$CONFIG_TOML"
  sed -i 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/' "$APP_TOML"
fi

# Prometheus metrics
sed -i 's/^prometheus = false/prometheus = true/' "$CONFIG_TOML"

# REST API
if [ "$ENABLE_API" = "true" ]; then
  sed -i '/^\[api\]/,/^\[/{s/^enable = false/enable = true/}' "$APP_TOML"
  sed -i 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' "$APP_TOML"
fi

# gRPC
if [ "$ENABLE_GRPC" = "true" ]; then
  sed -i '/^\[grpc\]/,/^\[/{s/^enable = false/enable = true/}' "$APP_TOML"
  sed -i 's|address = "localhost:9090"|address = "0.0.0.0:9090"|' "$APP_TOML"
fi

# Pruning
sed -i "s/^pruning = .*/pruning = \"$PRUNING\"/" "$APP_TOML"

# Snapshot interval
if [ "$SNAPSHOT_INTERVAL" != "0" ]; then
  sed -i "s/^snapshot-interval = .*/snapshot-interval = $SNAPSHOT_INTERVAL/" "$APP_TOML"
fi

# ──── Run command ────
case "${1:-start}" in
  start)
    echo "[entrypoint] Starting legbled (chain=$CHAIN_ID, gas=$MINIMUM_GAS)"
    exec legbled start \
      --home "$HOME_DIR" \
      --minimum-gas-prices "$MINIMUM_GAS"
    ;;
  *)
    exec legbled "$@" --home "$HOME_DIR"
    ;;
esac
