#!/usr/bin/env bash
#
# Initialize a 3-validator Cosmos devnet for Docker deployment.
#
# This script runs OUTSIDE Docker to prepare volumes with:
#   - Shared genesis.json (all 3 validators + funded accounts)
#   - Per-node configs (unique ports, persistent peers)
#   - Gentxs from all validators
#
# Usage:
#   # Build the image first
#   docker build -f deployment/docker/Dockerfile.cosmos -t legbled .
#
#   # Initialize the devnet
#   ./deployment/docker/init-cosmos-docker.sh
#
#   # Start
#   docker compose -f deployment/docker/docker-compose.cosmos.yml up -d
#
# Requirements: docker, jq

set -euo pipefail

CHAIN_ID="lgm-devnet-1"
DENOM="ulgm"
IMAGE="legbled:latest"
NUM_VALIDATORS=3
COMPOSE_PROJECT="lgm-devnet"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="$SCRIPT_DIR/.devnet-data"

# 1 LGM = 1,000,000 ulgm
LGM=1000000
VALIDATOR_BALANCE=$(( 100000 * LGM ))  # 100,000 LGM per validator
SELF_DELEGATION=$(( 10000 * LGM ))     # 10,000 LGM staked
FAUCET_BALANCE=$(( 1000000000 * LGM )) # 1B LGM faucet
TEST_BALANCE=$(( 10000 * LGM ))        # 10,000 LGM per test account

info()  { echo -e "\033[1;34m→\033[0m $*"; }
ok()    { echo -e "\033[1;32m✓\033[0m $*"; }
err()   { echo -e "\033[1;31m✗\033[0m $*" >&2; exit 1; }

# Check prerequisites
command -v docker &>/dev/null || err "docker is required"
command -v jq &>/dev/null || err "jq is required"
docker image inspect "$IMAGE" &>/dev/null || err "Image '$IMAGE' not found. Build it first: docker build -f deployment/docker/Dockerfile.cosmos -t legbled ."

# Shorthand: run legbled inside a temp container, bypassing entrypoint
run_legbled() {
  local vol="$1"; shift
  docker run --rm --entrypoint legbled -v "$vol:/root/.legbled" "$IMAGE" "$@" --home /root/.legbled
}

echo ""
echo "╔═══════════════════════════════════════════════════════╗"
echo "║     Legible Money — Docker DevNet Init                ║"
echo "╠═══════════════════════════════════════════════════════╣"
echo "║  Chain ID:    $CHAIN_ID                          ║"
echo "║  Validators:  $NUM_VALIDATORS                                       ║"
echo "║  Image:       $IMAGE                        ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""

# ──── Clean previous state ────
if [ -d "$DATA_DIR" ]; then
  info "Removing previous devnet data..."
  rm -rf "$DATA_DIR"
fi
mkdir -p "$DATA_DIR"

# Also remove Docker volumes from previous runs
for i in $(seq 1 $NUM_VALIDATORS); do
  docker volume rm "${COMPOSE_PROJECT}_validator-${i}-data" 2>/dev/null || true
done

# ──── 1. Initialize each validator node ────
PEER_IDS=()
ADDRESSES=()

for i in $(seq 1 $NUM_VALIDATORS); do
  NODE_DIR="$DATA_DIR/validator-$i"
  mkdir -p "$NODE_DIR"
  MONIKER="validator-$i"

  info "Initializing $MONIKER..."

  # Init node
  run_legbled "$NODE_DIR" init "$MONIKER" --chain-id "$CHAIN_ID" 2>/dev/null

  # Create validator key
  run_legbled "$NODE_DIR" keys add "$MONIKER" --keyring-backend test 2>/dev/null

  # Get address
  ADDR=$(docker run --rm --entrypoint legbled -v "$NODE_DIR:/root/.legbled" "$IMAGE" \
    keys show "$MONIKER" -a --keyring-backend test --home /root/.legbled 2>/dev/null)
  ADDRESSES+=("$ADDR")

  # Get node ID
  NODE_ID=$(docker run --rm --entrypoint legbled -v "$NODE_DIR:/root/.legbled" "$IMAGE" \
    tendermint show-node-id --home /root/.legbled 2>&1 | tail -1)
  PEER_IDS+=("$NODE_ID")

  ok "$MONIKER: addr=$ADDR node_id=$NODE_ID"
done

# ──── 2. Build shared genesis on validator-1 ────
info "Building shared genesis..."
NODE1_DIR="$DATA_DIR/validator-1"
GENESIS="$NODE1_DIR/config/genesis.json"

# Add all validator accounts to genesis
for i in $(seq 1 $NUM_VALIDATORS); do
  run_legbled "$NODE1_DIR" add-genesis-account "${ADDRESSES[$((i-1))]}" "${VALIDATOR_BALANCE}${DENOM}"
done

# Add faucet + test accounts (create keys on node1)
run_legbled "$NODE1_DIR" keys add faucet --keyring-backend test 2>/dev/null
FAUCET_ADDR=$(docker run --rm --entrypoint legbled -v "$NODE1_DIR:/root/.legbled" "$IMAGE" \
  keys show faucet -a --keyring-backend test --home /root/.legbled 2>/dev/null)
run_legbled "$NODE1_DIR" add-genesis-account "$FAUCET_ADDR" "${FAUCET_BALANCE}${DENOM}"

for name in test1 test2 test3; do
  run_legbled "$NODE1_DIR" keys add "$name" --keyring-backend test 2>/dev/null
  TADDR=$(docker run --rm --entrypoint legbled -v "$NODE1_DIR:/root/.legbled" "$IMAGE" \
    keys show "$name" -a --keyring-backend test --home /root/.legbled 2>/dev/null)
  run_legbled "$NODE1_DIR" add-genesis-account "$TADDR" "${TEST_BALANCE}${DENOM}"
done

# Patch genesis
info "Patching genesis (vote extensions, PoT params, fast governance)..."
jq '
  .consensus.params.abci.vote_extensions_enable_height = "1" |
  .app_state.staking.params.bond_denom = "ulgm" |
  .app_state.knowledge.params.min_verifiers = 1 |
  .app_state.bank.denom_metadata = [{
    "description": "The native staking token of Legible Money",
    "denom_units": [
      {"denom": "ulgm",  "exponent": 0, "aliases": ["microlegible"]},
      {"denom": "mlgm",  "exponent": 3, "aliases": ["millilegible"]},
      {"denom": "lgm",   "exponent": 6, "aliases": ["legible"]}
    ],
    "base": "ulgm",
    "display": "lgm",
    "name": "Legible Money",
    "symbol": "LGM"
  }] |
  .app_state.gov.params.voting_period = "60s" |
  .app_state.gov.params.expedited_voting_period = "30s" |
  .app_state.gov.params.max_deposit_period = "60s" |
  .app_state.gov.params.min_deposit = [{"denom": "ulgm", "amount": "10000000"}] |
  .app_state.slashing.params.signed_blocks_window = "100" |
  .app_state.slashing.params.slash_fraction_downtime = "0.010000000000000000"
' "$GENESIS" > "${GENESIS}.tmp" && mv "${GENESIS}.tmp" "$GENESIS"

ok "Genesis patched"

# ──── 3. Generate gentxs from each validator ────
info "Generating gentxs..."

# Copy shared genesis to all nodes first
for i in $(seq 2 $NUM_VALIDATORS); do
  cp "$GENESIS" "$DATA_DIR/validator-$i/config/genesis.json"
  # Also copy the accounts from node1's keyring so each node can create its gentx
done

# Create gentx on each node
for i in $(seq 1 $NUM_VALIDATORS); do
  NODE_DIR="$DATA_DIR/validator-$i"
  MONIKER="validator-$i"
  run_legbled "$NODE_DIR" gentx "$MONIKER" "${SELF_DELEGATION}${DENOM}" \
    --chain-id "$CHAIN_ID" \
    --keyring-backend test \
    --moniker "$MONIKER" 2>/dev/null
  ok "Gentx created for $MONIKER"
done

# Collect all gentxs into node1
for i in $(seq 2 $NUM_VALIDATORS); do
  cp "$DATA_DIR/validator-$i/config/gentx/"*.json "$NODE1_DIR/config/gentx/"
done

run_legbled "$NODE1_DIR" collect-gentxs 2>/dev/null
ok "Gentxs collected"

# Validate final genesis
run_legbled "$NODE1_DIR" validate-genesis 2>/dev/null
ok "Genesis valid"

# ──── 4. Distribute final genesis + configure peers ────
info "Configuring peer connections..."

# Build persistent_peers string
# In Docker, validators connect via container names on the lgm-net network
PEERS=""
DOCKER_NAMES=("lgm-validator-1" "lgm-validator-2" "lgm-validator-3")
for i in $(seq 1 $NUM_VALIDATORS); do
  IDX=$((i-1))
  if [ -n "$PEERS" ]; then PEERS="$PEERS,"; fi
  PEERS="${PEERS}${PEER_IDS[$IDX]}@${DOCKER_NAMES[$IDX]}:26656"
done

for i in $(seq 1 $NUM_VALIDATORS); do
  NODE_DIR="$DATA_DIR/validator-$i"

  # Copy final genesis to all nodes (skip self-copy for validator-1)
  if [ "$NODE_DIR" != "$NODE1_DIR" ]; then
    cp "$NODE1_DIR/config/genesis.json" "$NODE_DIR/config/genesis.json"
  fi

  # Set persistent peers (exclude self)
  SELF_PEERS=""
  for j in $(seq 1 $NUM_VALIDATORS); do
    if [ $j -ne $i ]; then
      JDXI=$((j-1))
      if [ -n "$SELF_PEERS" ]; then SELF_PEERS="$SELF_PEERS,"; fi
      SELF_PEERS="${SELF_PEERS}${PEER_IDS[$JDXI]}@${DOCKER_NAMES[$JDXI]}:26656"
    fi
  done

  CONFIG="$NODE_DIR/config/config.toml"
  sed -i.bak "s/^persistent_peers = .*/persistent_peers = \"$SELF_PEERS\"/" "$CONFIG"

  # Listen on all interfaces (for Docker networking)
  sed -i.bak 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|' "$CONFIG"
  sed -i.bak 's|laddr = "tcp://127.0.0.1:26656"|laddr = "tcp://0.0.0.0:26656"|' "$CONFIG"

  # Prometheus
  sed -i.bak 's/^prometheus = false/prometheus = true/' "$CONFIG"

  # CORS for devnet
  sed -i.bak 's/cors_allowed_origins = \[\]/cors_allowed_origins = ["*"]/' "$CONFIG"

  # app.toml: API + gRPC
  APP="$NODE_DIR/config/app.toml"
  sed -i.bak '/^\[api\]/,/^\[/s/^enable = false/enable = true/' "$APP"
  sed -i.bak 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' "$APP"
  sed -i.bak '/^\[grpc\]/,/^\[/s/^enable = false/enable = true/' "$APP"
  sed -i.bak 's|address = "localhost:9090"|address = "0.0.0.0:9090"|' "$APP"
  sed -i.bak 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/' "$APP"

  # Clean sed backups
  rm -f "$CONFIG.bak" "$APP.bak"

  ok "validator-$i configured (peers: $SELF_PEERS)"
done

# ──── 5. Copy data into Docker volumes ────
info "Copying data into Docker volumes..."

for i in $(seq 1 $NUM_VALIDATORS); do
  VOL_NAME="${COMPOSE_PROJECT}_validator-${i}-data"
  NODE_DIR="$DATA_DIR/validator-$i"

  # Create volume and copy data
  docker volume create "$VOL_NAME" > /dev/null
  docker run --rm \
    -v "$NODE_DIR:/src" \
    -v "$VOL_NAME:/dest" \
    alpine sh -c "cp -a /src/. /dest/"

  ok "Volume $VOL_NAME populated"
done

# ──── 6. Create Ping.pub explorer config ────
info "Creating Ping.pub explorer config..."
EXPLORER_DIR="$SCRIPT_DIR/ping-pub-config"
mkdir -p "$EXPLORER_DIR"

cat > "$EXPLORER_DIR/lgm-devnet.json" << EOF
{
  "chain_name": "lgm-devnet",
  "api": ["http://lgm-validator-1:1317"],
  "rpc": ["http://lgm-validator-1:26657"],
  "grpc": ["http://lgm-validator-1:9090"],
  "sdk_version": "0.50.10",
  "coin_type": "118",
  "min_tx_fee": "5000",
  "addr_prefix": "lgm",
  "logo": "",
  "assets": [
    {
      "base": "ulgm",
      "symbol": "LGM",
      "exponent": "6",
      "coingecko_id": "",
      "logo": ""
    }
  ]
}
EOF
ok "Ping.pub config created"

# ──── Done ────
echo ""
echo "╔═══════════════════════════════════════════════════════╗"
echo "║     DevNet Initialization Complete                    ║"
echo "╠═══════════════════════════════════════════════════════╣"
echo "║                                                       ║"
echo "║  Validators:                                          ║"
for i in $(seq 1 $NUM_VALIDATORS); do
  printf "║    validator-%-2d: %-37s  ║\n" "$i" "${ADDRESSES[$((i-1))]}"
done
echo "║                                                       ║"
echo "║  Faucet: $FAUCET_ADDR  ║"
echo "║                                                       ║"
echo "║  Start the network:                                   ║"
echo "║    cd deployment/docker                                ║"
echo "║    docker compose -f docker-compose.cosmos.yml up -d   ║"
echo "║                                                       ║"
echo "║  With explorer:                                       ║"
echo "║    docker compose -f docker-compose.cosmos.yml \\       ║"
echo "║      --profile explorer up -d                          ║"
echo "║                                                       ║"
echo "║  Check status:                                        ║"
echo "║    curl -s localhost:26657/status | jq                 ║"
echo "║                                                       ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""
