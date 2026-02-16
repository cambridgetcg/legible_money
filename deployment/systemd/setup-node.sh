#!/usr/bin/env bash
#
# Legible Money — Node Setup Script
#
# Sets up a bare-metal or VPS server to run a legbled node.
# Tested on: Ubuntu 22.04/24.04, Debian 12
#
# Usage:
#   # Full validator setup
#   sudo ./setup-node.sh --validator --moniker my-validator --chain-id lgm-1
#
#   # Full node (non-validator)
#   sudo ./setup-node.sh --moniker my-fullnode --chain-id lgm-1 --genesis-url https://...
#
#   # Devnet single-node
#   sudo ./setup-node.sh --validator --devnet

set -euo pipefail

# ──── Defaults ────
MONIKER="legible-node"
CHAIN_ID="lgm-1"
DENOM="ulgm"
IS_VALIDATOR=false
IS_DEVNET=false
GENESIS_URL=""
SEEDS=""
PERSISTENT_PEERS=""
BINARY_URL=""
GO_VERSION="1.22.5"

LGM_USER="legbled"
LGM_HOME="/var/lib/legbled"
NODE_HOME="$LGM_HOME/.legbled"

# ──── Parse args ────
while [[ $# -gt 0 ]]; do
  case $1 in
    --moniker)      MONIKER="$2"; shift 2 ;;
    --chain-id)     CHAIN_ID="$2"; shift 2 ;;
    --genesis-url)  GENESIS_URL="$2"; shift 2 ;;
    --seeds)        SEEDS="$2"; shift 2 ;;
    --peers)        PERSISTENT_PEERS="$2"; shift 2 ;;
    --binary-url)   BINARY_URL="$2"; shift 2 ;;
    --validator)    IS_VALIDATOR=true; shift ;;
    --devnet)       IS_DEVNET=true; CHAIN_ID="lgm-devnet-1"; shift ;;
    *)              echo "Unknown: $1"; exit 1 ;;
  esac
done

info()  { echo -e "\033[1;34m→\033[0m $*"; }
ok()    { echo -e "\033[1;32m✓\033[0m $*"; }
err()   { echo -e "\033[1;31m✗\033[0m $*" >&2; exit 1; }

# Must be root
[[ $EUID -eq 0 ]] || err "Run with sudo"

echo ""
echo "╔═══════════════════════════════════════════════╗"
echo "║     Legible Money — Node Setup                ║"
echo "╠═══════════════════════════════════════════════╣"
echo "║  Moniker:    $MONIKER"
echo "║  Chain ID:   $CHAIN_ID"
echo "║  Validator:  $IS_VALIDATOR"
echo "║  DevNet:     $IS_DEVNET"
echo "╚═══════════════════════════════════════════════╝"
echo ""

# ──── 1. System dependencies ────
info "Installing system dependencies..."
apt-get update -qq
apt-get install -y -qq curl jq git build-essential lz4 > /dev/null 2>&1
ok "Dependencies installed"

# ──── 2. Create service user ────
if ! id "$LGM_USER" &>/dev/null; then
  info "Creating user: $LGM_USER"
  useradd -m -d "$LGM_HOME" -s /bin/bash "$LGM_USER"
  ok "User created"
else
  ok "User $LGM_USER already exists"
fi

# ──── 3. Install Go (if building from source) ────
if [ -z "$BINARY_URL" ]; then
  if ! command -v go &>/dev/null || [[ "$(go version)" != *"$GO_VERSION"* ]]; then
    info "Installing Go $GO_VERSION..."
    curl -sSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xz
    echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
    export PATH=$PATH:/usr/local/go/bin
    ok "Go $GO_VERSION installed"
  else
    ok "Go already installed"
  fi
fi

# ──── 4. Install legbled binary ────
if [ -n "$BINARY_URL" ]; then
  info "Downloading legbled binary from $BINARY_URL..."
  curl -sSL "$BINARY_URL" -o /usr/local/bin/legbled
  chmod +x /usr/local/bin/legbled
else
  info "Building legbled from source..."
  cd /tmp
  if [ ! -d legible-money ]; then
    git clone --depth 1 https://github.com/legible-protocol/legible-money.git
  fi
  cd legible-money
  go build -o /usr/local/bin/legbled -ldflags="-s -w" ./cmd/legbled
  rm -rf /tmp/legible-money
fi
ok "legbled installed: $(legbled version 2>/dev/null || echo 'v0.1.0')"

# ──── 5. Initialize node ────
info "Initializing node..."
sudo -u "$LGM_USER" legbled init "$MONIKER" \
  --chain-id "$CHAIN_ID" \
  --home "$NODE_HOME" 2>/dev/null
ok "Node initialized at $NODE_HOME"

# ──── 6. Download genesis ────
if [ -n "$GENESIS_URL" ]; then
  info "Downloading genesis from $GENESIS_URL..."
  curl -sSL "$GENESIS_URL" > "$NODE_HOME/config/genesis.json"
  chown "$LGM_USER:$LGM_USER" "$NODE_HOME/config/genesis.json"
  ok "Genesis downloaded"
fi

# ──── 7. Configure peers ────
CONFIG_TOML="$NODE_HOME/config/config.toml"
APP_TOML="$NODE_HOME/config/app.toml"

if [ -n "$SEEDS" ]; then
  sed -i "s/^seeds = .*/seeds = \"$SEEDS\"/" "$CONFIG_TOML"
fi

if [ -n "$PERSISTENT_PEERS" ]; then
  sed -i "s/^persistent_peers = .*/persistent_peers = \"$PERSISTENT_PEERS\"/" "$CONFIG_TOML"
fi

# Enable API + gRPC on all interfaces
sed -i '/^\[api\]/,/^\[/{s/^enable = false/enable = true/}' "$APP_TOML"
sed -i 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' "$APP_TOML"
sed -i '/^\[grpc\]/,/^\[/{s/^enable = false/enable = true/}' "$APP_TOML"
sed -i 's|address = "localhost:9090"|address = "0.0.0.0:9090"|' "$APP_TOML"

# Prometheus metrics
sed -i 's/^prometheus = false/prometheus = true/' "$CONFIG_TOML"

ok "Node configured"

# ──── 8. Firewall ────
if command -v ufw &>/dev/null; then
  info "Configuring firewall (ufw)..."
  ufw allow 26656/tcp comment "legbled P2P"
  ufw allow 26657/tcp comment "legbled RPC"
  ufw allow 1317/tcp  comment "legbled REST API"
  ufw allow 9090/tcp  comment "legbled gRPC"
  ok "Firewall rules added"
fi

# ──── 9. Install systemd service ────
info "Installing systemd service..."
cat > /etc/systemd/system/legbled.service << 'UNIT'
[Unit]
Description=Legible Money Blockchain Daemon (legbled)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=legbled
Group=legbled
ExecStart=/usr/local/bin/legbled start --home /var/lib/legbled/.legbled --minimum-gas-prices 0.025ulgm
Restart=on-failure
RestartSec=5
LimitNOFILE=65536
LimitNPROC=65536
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/legbled
PrivateTmp=true
StandardOutput=journal
StandardError=journal
SyslogIdentifier=legbled

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable legbled
ok "Service installed and enabled"

# ──── 10. DevNet: auto-configure single-node genesis ────
if [ "$IS_DEVNET" = true ] && [ "$IS_VALIDATOR" = true ]; then
  info "Configuring devnet genesis..."

  # Vote extensions at height 1
  jq '.consensus.params.abci.vote_extensions_enable_height = "1"' \
    "$NODE_HOME/config/genesis.json" > /tmp/genesis.tmp && \
    mv /tmp/genesis.tmp "$NODE_HOME/config/genesis.json"

  # Create validator key
  sudo -u "$LGM_USER" legbled keys add validator \
    --keyring-backend test --home "$NODE_HOME" 2>/dev/null

  ADDR=$(sudo -u "$LGM_USER" legbled keys show validator -a \
    --keyring-backend test --home "$NODE_HOME")

  # Fund validator
  sudo -u "$LGM_USER" legbled add-genesis-account "$ADDR" \
    "50000000000${DENOM}" --home "$NODE_HOME"

  # Gentx
  sudo -u "$LGM_USER" legbled gentx validator "10000000000${DENOM}" \
    --chain-id "$CHAIN_ID" \
    --keyring-backend test \
    --home "$NODE_HOME" \
    --moniker "$MONIKER" 2>/dev/null

  sudo -u "$LGM_USER" legbled collect-gentxs --home "$NODE_HOME" 2>/dev/null
  sudo -u "$LGM_USER" legbled validate-genesis --home "$NODE_HOME" 2>/dev/null

  # DevNet: unsafe RPC + CORS
  sed -i 's/cors_allowed_origins = \[\]/cors_allowed_origins = ["*"]/' "$CONFIG_TOML"
  sed -i 's/^unsafe = false/unsafe = true/' "$CONFIG_TOML"
  sed -i 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/' "$APP_TOML"

  chown -R "$LGM_USER:$LGM_USER" "$NODE_HOME"
  ok "DevNet genesis configured (validator: $ADDR)"
fi

# ──── Done ────
echo ""
echo "╔═══════════════════════════════════════════════╗"
echo "║     Setup Complete                            ║"
echo "╠═══════════════════════════════════════════════╣"
echo "║  Start:    sudo systemctl start legbled       ║"
echo "║  Logs:     journalctl -u legbled -f           ║"
echo "║  Status:   curl localhost:26657/status         ║"
echo "║  Stop:     sudo systemctl stop legbled        ║"
echo "╚═══════════════════════════════════════════════╝"
echo ""
