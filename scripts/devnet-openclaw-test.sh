#!/bin/bash
# ═══════════════════════════════════════════════════════════════════════════
# Legible Money — OpenClaw Integration Test DevNet
# ═══════════════════════════════════════════════════════════════════════════
#
# Sets up a 3-validator local devnet for OpenClaw integration testing.
# Uses the genesis ceremony pattern with devnet-appropriate parameters.
#
# Usage:
#   ./scripts/devnet-openclaw-test.sh setup     # Build + genesis + configure
#   ./scripts/devnet-openclaw-test.sh start      # Start all 3 nodes
#   ./scripts/devnet-openclaw-test.sh stop       # Stop all nodes
#   ./scripts/devnet-openclaw-test.sh clean      # Remove all state
# ═══════════════════════════════════════════════════════════════════════════

set -euo pipefail

CHAIN_ID="legible-devnet-1"
DENOM="ulgm"
BINARY="build/legbled"
DEVNET_BASE="${HOME}/.lgm/openclaw-devnet"

# Node ports
# Node 1: P2P=26656, RPC=26657, API=1317
# Node 2: P2P=26666, RPC=26667, API=1318
# Node 3: P2P=26676, RPC=26677, API=1319

VALIDATOR_NAMES=("validator-1" "validator-2" "validator-3")
P2P_PORTS=(26656 26666 26676)
RPC_PORTS=(26657 26667 26677)
API_PORTS=(1317 1318 1319)

VALIDATOR_BALANCE="1000000000000${DENOM}"     # 1M LGM
VALIDATOR_STAKE="100000000000${DENOM}"         # 100K LGM
FAUCET_BALANCE="500000000000${DENOM}"          # 500K LGM

# ── Helpers ────────────────────────────────────────────────────────────────

die() { echo "ERROR: $*" >&2; exit 1; }
info() { echo "  → $*"; }

check_binary() {
  if [ ! -f "${BINARY}" ]; then
    echo "Binary not found at ${BINARY}. Building..."
    go build -o "${BINARY}" ./cmd/legbled || die "Build failed"
  fi
}

patch_genesis() {
  local genesis="$1" path="$2" value="$3"
  python3 -c "
import json
with open('${genesis}') as f:
    g = json.load(f)
keys = '${path}'.strip('.').split('.')
obj = g
for k in keys[:-1]:
    obj = obj[k]
obj[keys[-1]] = ${value}
with open('${genesis}', 'w') as f:
    json.dump(g, f, indent=2)
" || die "Failed to patch genesis at ${path}"
}

patch_app_state() {
  local genesis="$1" module="$2" path="$3" value="$4"
  python3 -c "
import json
with open('${genesis}') as f:
    g = json.load(f)
mod = json.loads(g['app_state']['${module}']) if isinstance(g['app_state']['${module}'], str) else g['app_state']['${module}']
keys = '${path}'.strip('.').split('.')
obj = mod
for k in keys[:-1]:
    obj = obj[k]
obj[keys[-1]] = ${value}
g['app_state']['${module}'] = mod
with open('${genesis}', 'w') as f:
    json.dump(g, f, indent=2)
" || die "Failed to patch app_state.${module} at ${path}"
}

# ── Commands ───────────────────────────────────────────────────────────────

cmd_setup() {
  echo "═══════════════════════════════════════════════════════════════"
  echo "  OpenClaw Integration Test — DevNet Setup"
  echo "  Chain ID: ${CHAIN_ID}"
  echo "  Nodes: 3 validators on localhost"
  echo "═══════════════════════════════════════════════════════════════"
  echo ""

  check_binary

  # Clean previous
  if [ -d "${DEVNET_BASE}" ]; then
    echo "Removing previous devnet state..."
    rm -rf "${DEVNET_BASE}"
  fi
  mkdir -p "${DEVNET_BASE}"

  # ── Phase 1: Initialize coordinator ──
  info "Phase 1: Initializing coordinator node..."
  local coord_home="${DEVNET_BASE}/coordinator"
  ${BINARY} init coordinator \
    --chain-id "${CHAIN_ID}" \
    --default-denom "${DENOM}" \
    --home "${coord_home}" 2>/dev/null

  local genesis="${coord_home}/config/genesis.json"

  # Patch genesis time to NOW
  local now
  now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  info "Setting genesis time to ${now}..."
  patch_genesis "${genesis}" '.genesis_time' "\"${now}\""

  # ── Consensus params ──
  info "Setting consensus parameters..."
  patch_genesis "${genesis}" '.consensus.params.block.max_gas' '"-1"'
  patch_genesis "${genesis}" '.consensus.params.block.max_bytes' '"4194304"'
  patch_genesis "${genesis}" '.consensus.params.abci.vote_extensions_enable_height' '"1"'

  # ── Knowledge module (fast phases for testing) ──
  info "Configuring knowledge module (fast devnet phases)..."
  patch_app_state "${genesis}" 'knowledge' '.params.commit_phase_duration' 20
  patch_app_state "${genesis}" 'knowledge' '.params.reveal_phase_duration' 20
  patch_app_state "${genesis}" 'knowledge' '.params.aggregation_phase_duration' 10
  patch_app_state "${genesis}" 'knowledge' '.params.min_verifiers' 1
  patch_app_state "${genesis}" 'knowledge' '.params.acceptance_threshold' 770000
  patch_app_state "${genesis}" 'knowledge' '.params.quorum_threshold' 660000
  patch_app_state "${genesis}" 'knowledge' '.params.max_validators_per_round' 22
  patch_app_state "${genesis}" 'knowledge' '.params.invalid_claim_slash_bps' 220000
  patch_app_state "${genesis}" 'knowledge' '.params.verification_reward_amount' 3000000
  patch_app_state "${genesis}" 'knowledge' '.params.min_claim_text_length' 20
  patch_app_state "${genesis}" 'knowledge' '.params.max_claim_reward_multiplier' 2

  # ── Governance ──
  info "Configuring governance..."
  patch_app_state "${genesis}" 'lgm_gov' '.params.voting_period_blocks' 34272
  patch_app_state "${genesis}" 'lgm_gov' '.params.review_period_blocks' 17136
  patch_app_state "${genesis}" 'lgm_gov' '.params.last_call_period_blocks' 17136
  patch_app_state "${genesis}" 'lgm_gov' '.params.support_threshold_bps' 7500
  patch_app_state "${genesis}" 'lgm_gov' '.params.quorum_threshold_bps' 1500

  # ── Emergency ──
  info "Configuring emergency module..."
  patch_app_state "${genesis}" 'emergency' '.params.halt_quorum' 750000
  patch_app_state "${genesis}" 'emergency' '.params.min_distinct_voters' 2
  patch_app_state "${genesis}" 'emergency' '.params.min_guardian_stake' '"50000000000"'

  # ── Staking ──
  info "Configuring staking..."
  patch_app_state "${genesis}" 'legible_staking' '.params.max_validators' 100
  patch_app_state "${genesis}" 'legible_staking' '.params.min_self_delegation' '"111000"'
  patch_app_state "${genesis}" 'legible_staking' '.params.unbonding_period' 268560

  # ── Vesting rewards ──
  info "Configuring vesting rewards..."
  patch_app_state "${genesis}" 'vesting_rewards' '.params.block_reward_amount' '"3000000"'
  patch_app_state "${genesis}" 'vesting_rewards' '.params.min_validators_for_full_reward' 3
  patch_app_state "${genesis}" 'vesting_rewards' '.params.vesting_enabled' True

  # ── Create faucet ──
  info "Creating faucet account..."
  ${BINARY} keys add faucet --keyring-backend test --home "${coord_home}" 2>/dev/null
  FAUCET_ADDR=$(${BINARY} keys show faucet -a --keyring-backend test --home "${coord_home}")
  ${BINARY} add-genesis-account "${FAUCET_ADDR}" "${FAUCET_BALANCE}" --home "${coord_home}"
  info "  Faucet: ${FAUCET_ADDR}"

  # ── Phase 2: Create validator nodes ──
  info ""
  info "Phase 2: Creating 3 validators..."
  mkdir -p "${coord_home}/config/gentx"

  for i in 0 1 2; do
    local name="${VALIDATOR_NAMES[$i]}"
    local val_home="${DEVNET_BASE}/${name}"

    info "Setting up ${name}..."

    # Init the validator node (generates unique consensus key)
    ${BINARY} init "${name}" --chain-id "${CHAIN_ID}" --home "${val_home}" --overwrite 2>/dev/null

    # Copy current genesis to validator home
    cp "${genesis}" "${val_home}/config/genesis.json"

    # Create validator account key in coordinator keyring
    ${BINARY} keys add "${name}" --keyring-backend test --home "${coord_home}" 2>/dev/null
    local addr
    addr=$(${BINARY} keys show "${name}" -a --keyring-backend test --home "${coord_home}")
    info "  ${name}: ${addr}"

    # Fund validator in coordinator genesis
    ${BINARY} add-genesis-account "${addr}" "${VALIDATOR_BALANCE}" --home "${coord_home}"

    # Copy updated genesis back
    cp "${genesis}" "${val_home}/config/genesis.json"

    # Copy keyring so gentx can sign
    cp -r "${coord_home}/keyring-test" "${val_home}/"

    # Create gentx
    ${BINARY} gentx "${name}" "${VALIDATOR_STAKE}" \
      --chain-id "${CHAIN_ID}" \
      --keyring-backend test \
      --home "${val_home}" \
      --moniker "${name}" \
      --commission-rate "0.10" \
      --commission-max-rate "0.20" \
      --commission-max-change-rate "0.01" \
      --output-document "${coord_home}/config/gentx/gentx-${name}.json" 2>/dev/null

    info "  Created gentx for ${name}"
  done

  # ── Phase 3: Finalize genesis ──
  info ""
  info "Phase 3: Collecting gentxs and finalizing..."
  ${BINARY} collect-gentxs --home "${coord_home}" 2>/dev/null

  # Validate
  info "Validating genesis..."
  ${BINARY} validate --home "${coord_home}" 2>&1 || info "(validation warning — proceeding)"

  # ── Phase 4: Distribute genesis + configure nodes ──
  info ""
  info "Phase 4: Distributing genesis and configuring peers..."

  # Get node IDs for persistent peers
  local peers=""
  for i in 0 1 2; do
    local name="${VALIDATOR_NAMES[$i]}"
    local val_home="${DEVNET_BASE}/${name}"
    local node_id
    node_id=$(${BINARY} comet show-node-id --home "${val_home}" 2>&1)
    local p2p_port="${P2P_PORTS[$i]}"

    if [ -n "${peers}" ]; then
      peers="${peers},"
    fi
    peers="${peers}${node_id}@127.0.0.1:${p2p_port}"
  done

  for i in 0 1 2; do
    local name="${VALIDATOR_NAMES[$i]}"
    local val_home="${DEVNET_BASE}/${name}"
    local p2p_port="${P2P_PORTS[$i]}"
    local rpc_port="${RPC_PORTS[$i]}"
    local api_port="${API_PORTS[$i]}"

    # Copy final genesis
    cp "${coord_home}/config/genesis.json" "${val_home}/config/genesis.json"

    # Copy keyring to all validators
    cp -r "${coord_home}/keyring-test" "${val_home}/"

    # Configure ports and peers in config.toml
    local config="${val_home}/config/config.toml"
    if [ -f "${config}" ]; then
      # P2P listen address
      sed -i.bak "s|laddr = \"tcp://0.0.0.0:26656\"|laddr = \"tcp://0.0.0.0:${p2p_port}\"|" "${config}"
      # RPC listen address
      sed -i.bak "s|laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://127.0.0.1:${rpc_port}\"|" "${config}"
      # Persistent peers (all other nodes)
      sed -i.bak "s|persistent_peers = \"\"|persistent_peers = \"${peers}\"|" "${config}"
      # Allow duplicate IPs for localhost setup
      sed -i.bak "s|allow_duplicate_ip = false|allow_duplicate_ip = true|" "${config}"
      rm -f "${config}.bak"
    fi

    # Configure app.toml for REST API
    local app_config="${val_home}/config/app.toml"
    if [ -f "${app_config}" ]; then
      # Enable REST API (targeted — only the [api] section)
      # Use python3 for section-aware TOML editing
      python3 -c "
import re
with open('${app_config}') as f:
    content = f.read()
# Enable API section: find [api] and set enable = true
content = re.sub(r'(\[api\][^\[]*?)enable = false', r'\1enable = true', content, count=1, flags=re.DOTALL)
# Disable gRPC to avoid proto reflection crash with hand-written types
content = re.sub(r'(\[grpc\][^\[]*?)enable = true', r'\1enable = false', content, count=1, flags=re.DOTALL)
# Disable gRPC-web too
content = re.sub(r'(\[grpc-web\][^\[]*?)enable = true', r'\1enable = false', content, count=1, flags=re.DOTALL)
# Set API address
content = content.replace('address = \"tcp://localhost:1317\"', 'address = \"tcp://localhost:${api_port}\"')
content = content.replace('address = \"tcp://0.0.0.0:1317\"', 'address = \"tcp://localhost:${api_port}\"')
with open('${app_config}', 'w') as f:
    f.write(content)
" || die "Failed to configure app.toml for ${name}"
    fi
  done

  echo ""
  echo "═══════════════════════════════════════════════════════════════"
  echo "  DevNet setup complete!"
  echo ""
  echo "  Node 1: RPC=26657, API=1317, P2P=26656"
  echo "  Node 2: RPC=26667, API=1318, P2P=26666"
  echo "  Node 3: RPC=26677, API=1319, P2P=26676"
  echo ""
  echo "  Validator keys in: ${coord_home}/keyring-test"
  echo ""
  echo "  Next: ./scripts/devnet-openclaw-test.sh start"
  echo "═══════════════════════════════════════════════════════════════"
}

cmd_start() {
  echo "═══════════════════════════════════════════════════════════════"
  echo "  Starting 3-node DevNet..."
  echo "═══════════════════════════════════════════════════════════════"

  check_binary

  for i in 0 1 2; do
    local name="${VALIDATOR_NAMES[$i]}"
    local val_home="${DEVNET_BASE}/${name}"
    local log_file="${DEVNET_BASE}/${name}.log"

    [ -d "${val_home}" ] || die "Node ${name} not set up. Run 'setup' first."

    info "Starting ${name} (PID file: ${DEVNET_BASE}/${name}.pid)..."
    ${BINARY} start --home "${val_home}" \
      --minimum-gas-prices "0ulgm" \
      > "${log_file}" 2>&1 &
    echo $! > "${DEVNET_BASE}/${name}.pid"
    info "  PID: $(cat "${DEVNET_BASE}/${name}.pid")"
  done

  echo ""
  info "Waiting for blocks to start producing..."
  sleep 8

  # Check if nodes are producing blocks
  for i in 0 1 2; do
    local rpc_port="${RPC_PORTS[$i]}"
    local status
    status=$(curl -s "http://127.0.0.1:${rpc_port}/status" 2>/dev/null || echo "FAILED")
    if echo "${status}" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d['result']['sync_info']['latest_block_height'])" 2>/dev/null; then
      info "  Node $((i+1)): Block height $(echo "${status}" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d['result']['sync_info']['latest_block_height'])")"
    else
      info "  Node $((i+1)): Still starting up..."
    fi
  done

  echo ""
  echo "═══════════════════════════════════════════════════════════════"
  echo "  DevNet started. Logs in ${DEVNET_BASE}/*.log"
  echo "  REST API: http://localhost:1317"
  echo "  RPC:      http://localhost:26657"
  echo "═══════════════════════════════════════════════════════════════"
}

cmd_stop() {
  echo "Stopping DevNet nodes..."
  for i in 0 1 2; do
    local name="${VALIDATOR_NAMES[$i]}"
    local pid_file="${DEVNET_BASE}/${name}.pid"
    if [ -f "${pid_file}" ]; then
      local pid
      pid=$(cat "${pid_file}")
      if kill -0 "${pid}" 2>/dev/null; then
        kill "${pid}"
        info "Stopped ${name} (PID ${pid})"
      fi
      rm -f "${pid_file}"
    fi
  done
  echo "All nodes stopped."
}

cmd_clean() {
  cmd_stop 2>/dev/null || true
  echo "Cleaning devnet state at ${DEVNET_BASE}..."
  rm -rf "${DEVNET_BASE}"
  echo "Done."
}

# ── Main ───────────────────────────────────────────────────────────────────

case "${1:-help}" in
  setup)  cmd_setup ;;
  start)  cmd_start ;;
  stop)   cmd_stop ;;
  clean)  cmd_clean ;;
  help|--help|-h)
    echo "Usage: $0 {setup|start|stop|clean}"
    ;;
  *)
    die "Unknown command: $1"
    ;;
esac
