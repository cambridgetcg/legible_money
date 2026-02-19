#!/usr/bin/env bash
#
# Legible Money — Join Testnet Script
#
# Usage:
#   ./scripts/join-testnet.sh [options]
#
# Options:
#   --moniker <name>     Node name (required)
#   --chain-id <id>      Chain ID (default: lgm-testnet-1)
#   --peers <peers>      Comma-separated persistent peers (node_id@host:port)
#   --genesis-url <url>  URL to download genesis.json
#   --home <dir>         Data directory (default: ~/.legbled)
#   --key-name <name>    Validator key name (default: validator)
#   --skip-build         Skip building binary
#   --start              Start the node after setup
#
# This script:
#   1. Builds the legbled binary (or downloads it)
#   2. Initializes the node
#   3. Downloads and validates the genesis file
#   4. Configures persistent peers
#   5. Optionally starts syncing

set -euo pipefail

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
CHAIN_ID="lgm-testnet-1"
DENOM="ulgm"
MONIKER=""
PEERS=""
GENESIS_URL=""
NODE_HOME="$HOME/.legbled"
KEY_NAME="validator"
SKIP_BUILD=false
START_NODE=false

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()  { echo -e "\033[1;34m→\033[0m $*"; }
ok()    { echo -e "\033[1;32m✓\033[0m $*"; }
warn()  { echo -e "\033[1;33m!\033[0m $*"; }
err()   { echo -e "\033[1;31m✗\033[0m $*" >&2; }

usage() {
    echo "Usage: $0 --moniker <name> [options]"
    echo ""
    echo "Options:"
    echo "  --moniker <name>     Node name (required)"
    echo "  --chain-id <id>      Chain ID (default: $CHAIN_ID)"
    echo "  --peers <peers>      Persistent peers (node_id@host:port,...)"
    echo "  --genesis-url <url>  URL to download genesis.json"
    echo "  --home <dir>         Data directory (default: $NODE_HOME)"
    echo "  --key-name <name>    Validator key name (default: $KEY_NAME)"
    echo "  --skip-build         Skip building binary"
    echo "  --start              Start the node after setup"
    exit 1
}

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --moniker)     MONIKER="$2";      shift 2 ;;
        --chain-id)    CHAIN_ID="$2";     shift 2 ;;
        --peers)       PEERS="$2";        shift 2 ;;
        --genesis-url) GENESIS_URL="$2";  shift 2 ;;
        --home)        NODE_HOME="$2";    shift 2 ;;
        --key-name)    KEY_NAME="$2";     shift 2 ;;
        --skip-build)  SKIP_BUILD=true;   shift ;;
        --start)       START_NODE=true;   shift ;;
        -h|--help)     usage ;;
        *)             err "Unknown option: $1"; usage ;;
    esac
done

if [ -z "$MONIKER" ]; then
    err "Moniker is required (--moniker <name>)"
    usage
fi

# ---------------------------------------------------------------------------
# Detect project root
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="$PROJECT_ROOT/build/legbled"

# ---------------------------------------------------------------------------
# Step 1: Build binary
# ---------------------------------------------------------------------------
if [ "$SKIP_BUILD" = true ] && [ -f "$BINARY" ]; then
    ok "Skipping build (--skip-build, binary exists)"
else
    info "Building legbled binary..."

    if ! command -v go &>/dev/null; then
        err "Go is required but not found. Install Go 1.23+ from https://go.dev/dl/"
        exit 1
    fi

    cd "$PROJECT_ROOT"
    mkdir -p build

    # Build with version metadata
    VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    LDFLAGS="-X github.com/cosmos/cosmos-sdk/version.Name=legible \
             -X github.com/cosmos/cosmos-sdk/version.AppName=legbled \
             -X github.com/cosmos/cosmos-sdk/version.Version=$VERSION \
             -X github.com/cosmos/cosmos-sdk/version.Commit=$COMMIT"

    go build -ldflags "$LDFLAGS" -o "$BINARY" ./cmd/legbled
    ok "Binary built: $BINARY ($(du -h "$BINARY" | cut -f1))"
fi

# ---------------------------------------------------------------------------
# Step 2: Initialize node
# ---------------------------------------------------------------------------
if [ -d "$NODE_HOME/config" ]; then
    warn "Node home already exists at $NODE_HOME"
    warn "Remove it first if you want a fresh init: rm -rf $NODE_HOME"
else
    info "Initializing node ($MONIKER)..."
    "$BINARY" init "$MONIKER" --chain-id "$CHAIN_ID" --default-denom "$DENOM" --home "$NODE_HOME" >/dev/null 2>&1
    ok "Node initialized at $NODE_HOME"
fi

# ---------------------------------------------------------------------------
# Step 3: Genesis file
# ---------------------------------------------------------------------------
GENESIS="$NODE_HOME/config/genesis.json"

if [ -n "$GENESIS_URL" ]; then
    info "Downloading genesis from $GENESIS_URL..."
    curl -sL "$GENESIS_URL" -o "$GENESIS"
    ok "Genesis downloaded"
fi

info "Validating genesis..."
"$BINARY" validate-genesis --home "$NODE_HOME" 2>&1
ok "Genesis valid"

# ---------------------------------------------------------------------------
# Step 4: Configure node
# ---------------------------------------------------------------------------
CONFIG_TOML="$NODE_HOME/config/config.toml"
APP_TOML="$NODE_HOME/config/app.toml"

info "Configuring node..."

# CometBFT config
if [ -n "$PEERS" ]; then
    sed -i.bak "s/^persistent_peers = .*/persistent_peers = \"$PEERS\"/" "$CONFIG_TOML"
    ok "Persistent peers configured"
fi

# Block time (match network)
sed -i.bak 's/^timeout_commit = .*/timeout_commit = "2521ms"/' "$CONFIG_TOML"
sed -i.bak 's/^timeout_propose = .*/timeout_propose = "2000ms"/' "$CONFIG_TOML"

# Enable prometheus
sed -i.bak 's/^prometheus = false/prometheus = true/' "$CONFIG_TOML"

# App config
sed -i.bak 's/^minimum-gas-prices = .*/minimum-gas-prices = "0.025ulgm"/' "$APP_TOML"

# Clean up sed backup files
rm -f "${CONFIG_TOML}.bak" "${APP_TOML}.bak"

ok "Node configured (block time: 2521ms, prometheus: enabled)"

# ---------------------------------------------------------------------------
# Step 5: Create validator key
# ---------------------------------------------------------------------------
info "Checking for validator key ($KEY_NAME)..."
if "$BINARY" keys show "$KEY_NAME" --keyring-backend file --home "$NODE_HOME" >/dev/null 2>&1; then
    ok "Key '$KEY_NAME' already exists"
else
    warn "No key found. Create one with:"
    echo "  $BINARY keys add $KEY_NAME --keyring-backend file --home $NODE_HOME"
    echo ""
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "╔═════════════════════════════════════════════════════════╗"
echo "║         Legible Money — Node Setup Complete              ║"
echo "╠═════════════════════════════════════════════════════════╣"
echo "║                                                          ║"
echo "║  Chain ID:   $CHAIN_ID                                   ║"
echo "║  Moniker:    $MONIKER"
echo "║  Home:       $NODE_HOME"
echo "║  Binary:     $BINARY"
echo "║                                                          ║"
echo "╚═════════════════════════════════════════════════════════╝"
echo ""
echo "Next steps:"
echo ""
echo "  1. Start syncing:"
echo "     $BINARY start --home $NODE_HOME --minimum-gas-prices 0.025ulgm"
echo ""
echo "  2. Wait for sync (check: curl -s localhost:26657/status | jq '.result.sync_info.catching_up')"
echo ""
echo "  3. Create validator key (if not done):"
echo "     $BINARY keys add $KEY_NAME --keyring-backend file --home $NODE_HOME"
echo ""
echo "  4. Fund your account and register as validator:"
echo "     PUBKEY=\$(cat $NODE_HOME/config/priv_validator_key.json | jq -r '.pub_key.value' | base64 -d | xxd -p -c 32)"
echo "     $BINARY tx legible_staking register-validator \$PUBKEY <stake-in-ulgm> \\"
echo "       --commission 1000 --moniker \"$MONIKER\" \\"
echo "       --from $KEY_NAME --keyring-backend file \\"
echo "       --chain-id $CHAIN_ID --fees 300000ulgm -y"
echo ""
echo "  See docs/VALIDATOR-ONBOARDING.md for full details."
echo ""

# ---------------------------------------------------------------------------
# Optional: start node
# ---------------------------------------------------------------------------
if [ "$START_NODE" = true ]; then
    info "Starting node..."
    exec "$BINARY" start --home "$NODE_HOME" --minimum-gas-prices "0.025$DENOM"
fi
