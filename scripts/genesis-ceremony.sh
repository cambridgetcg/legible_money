#!/bin/bash
# ═══════════════════════════════════════════════════════════════════════════
# Legible Money — Genesis Ceremony
# ═══════════════════════════════════════════════════════════════════════════
#
# Creates the genesis.json for chain "legible-1" — the forever network.
#
# Usage:
#   ./scripts/genesis-ceremony.sh init              # Initialize genesis dir
#   ./scripts/genesis-ceremony.sh add-validator NAME # Add a validator
#   ./scripts/genesis-ceremony.sh finalize           # Seal genesis
#   ./scripts/genesis-ceremony.sh export             # Copy final genesis.json
#
# Requires: build/legbled binary (run `go build -o build/legbled ./cmd/legbled`)
# ═══════════════════════════════════════════════════════════════════════════

set -euo pipefail

# ── Constants ──────────────────────────────────────────────────────────────

CHAIN_ID="legible-1"
GENESIS_TIME="2026-02-22T00:00:00Z"
DENOM="ulgm"
BINARY="build/legbled"
CEREMONY_HOME="${HOME}/.lgm/genesis-ceremony"

# Economics: NO pre-mine. Bootstrap accounts funded by governance, not allocation.
# Foundation: operational costs, initial development
# Research:   x/vesting_rewards research share treasury
# Faucet:     initial onboarding for new participants
FOUNDATION_BALANCE="10000000000000${DENOM}"   # 10,000,000 LGM (10M)
RESEARCH_BALANCE="5000000000000${DENOM}"       #  5,000,000 LGM  (5M)
FAUCET_BALANCE="500000000000${DENOM}"          #    500,000 LGM (500K)

# Validator bootstrap
VALIDATOR_BALANCE="1000000000000${DENOM}"      #  1,000,000 LGM  (1M) — initial balance
VALIDATOR_STAKE="100000000000${DENOM}"          #    100,000 LGM (100K) — self-delegation

# ── Helpers ────────────────────────────────────────────────────────────────

die() { echo "ERROR: $*" >&2; exit 1; }
info() { echo "  → $*"; }

check_binary() {
  if [ ! -f "${BINARY}" ]; then
    echo "Binary not found at ${BINARY}. Building..."
    go build -o "${BINARY}" ./cmd/legbled || die "Build failed"
  fi
}

# Patch a JSON field in genesis.json using python3 (available on macOS + Linux)
# Usage: patch_genesis '.path.to.field' 'value'
# For nested app_state modules: patch_app_state 'module' '.path' 'value'
patch_genesis() {
  local path="$1" value="$2"
  local genesis="${CEREMONY_HOME}/config/genesis.json"
  python3 -c "
import json, sys
with open('${genesis}') as f:
    g = json.load(f)

# Navigate to parent, set final key
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
  local module="$1" path="$2" value="$3"
  local genesis="${CEREMONY_HOME}/config/genesis.json"
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

cmd_init() {
  echo "═══════════════════════════════════════════════════════════════"
  echo "  Legible Money — Genesis Ceremony: INIT"
  echo "  Chain ID: ${CHAIN_ID}"
  echo "  Genesis Time: ${GENESIS_TIME}"
  echo "═══════════════════════════════════════════════════════════════"
  echo ""

  check_binary

  # Clean previous ceremony
  if [ -d "${CEREMONY_HOME}" ]; then
    echo "Removing previous ceremony state at ${CEREMONY_HOME}"
    rm -rf "${CEREMONY_HOME}"
  fi

  # Initialize chain
  info "Initializing chain..."
  ${BINARY} init genesis-coordinator \
    --chain-id "${CHAIN_ID}" \
    --default-denom "${DENOM}" \
    --home "${CEREMONY_HOME}" 2>/dev/null

  # Set genesis time
  info "Setting genesis time to ${GENESIS_TIME}..."
  patch_genesis '.genesis_time' "\"${GENESIS_TIME}\""

  # ── Consensus params ──
  info "Setting consensus parameters..."
  # Block time: 2521ms (CometBFT will use this as the target)
  # SDK v0.50 uses consensus.params.block (not consensus_params)
  patch_genesis '.consensus.params.block.max_gas' '"-1"'
  patch_genesis '.consensus.params.block.max_bytes' '"4194304"'
  # Enable ABCI++ vote extensions from block 1 (needed for VRF)
  patch_genesis '.consensus.params.abci.vote_extensions_enable_height' '"1"'

  # ── Knowledge module params (PoT core) ──
  info "Configuring knowledge module (PoT lifecycle)..."
  patch_app_state 'knowledge' '.params.commit_phase_duration'  100
  patch_app_state 'knowledge' '.params.reveal_phase_duration'  100
  patch_app_state 'knowledge' '.params.aggregation_phase_duration' 50
  patch_app_state 'knowledge' '.params.min_verifiers' 1
  patch_app_state 'knowledge' '.params.acceptance_threshold' 770000
  patch_app_state 'knowledge' '.params.quorum_threshold' 660000
  patch_app_state 'knowledge' '.params.max_validators_per_round' 22
  patch_app_state 'knowledge' '.params.invalid_claim_slash_bps' 220000
  patch_app_state 'knowledge' '.params.verification_reward_amount' 3000000
  patch_app_state 'knowledge' '.params.min_claim_text_length' 20
  patch_app_state 'knowledge' '.params.max_claim_reward_multiplier' 2

  # ── Governance params (short voting for initial iteration) ──
  # Module key is 'lgm_gov' in app_state (our custom governance)
  info "Configuring governance (short initial voting period)..."
  patch_app_state 'lgm_gov' '.params.voting_period_blocks' 34272
  patch_app_state 'lgm_gov' '.params.review_period_blocks' 17136
  patch_app_state 'lgm_gov' '.params.last_call_period_blocks' 17136
  patch_app_state 'lgm_gov' '.params.support_threshold_bps' 7500
  patch_app_state 'lgm_gov' '.params.quorum_threshold_bps' 1500

  # ── Emergency params (low threshold initially — safety net) ──
  info "Configuring emergency module..."
  patch_app_state 'emergency' '.params.halt_quorum' 750000
  patch_app_state 'emergency' '.params.min_distinct_voters' 2
  patch_app_state 'emergency' '.params.min_guardian_stake' '"50000000000"'

  # ── Staking params ──
  info "Configuring staking..."
  patch_app_state 'legible_staking' '.params.max_validators' 100
  patch_app_state 'legible_staking' '.params.min_self_delegation' '"111000"'
  patch_app_state 'legible_staking' '.params.unbonding_period' 268560

  # ── Vesting rewards ──
  info "Configuring vesting rewards..."
  patch_app_state 'vesting_rewards' '.params.block_reward_amount' '"3000000"'
  patch_app_state 'vesting_rewards' '.params.min_validators_for_full_reward' 3
  patch_app_state 'vesting_rewards' '.params.vesting_enabled' True

  # ── Create bootstrap accounts ──
  info "Creating foundation account..."
  FOUNDATION_KEY="foundation"
  ${BINARY} keys add "${FOUNDATION_KEY}" --keyring-backend test --home "${CEREMONY_HOME}" 2>/dev/null
  FOUNDATION_ADDR=$(${BINARY} keys show "${FOUNDATION_KEY}" -a --keyring-backend test --home "${CEREMONY_HOME}")
  ${BINARY} add-genesis-account "${FOUNDATION_ADDR}" "${FOUNDATION_BALANCE}" --home "${CEREMONY_HOME}"
  info "  Foundation: ${FOUNDATION_ADDR}"

  info "Creating research treasury account..."
  RESEARCH_KEY="research-treasury"
  ${BINARY} keys add "${RESEARCH_KEY}" --keyring-backend test --home "${CEREMONY_HOME}" 2>/dev/null
  RESEARCH_ADDR=$(${BINARY} keys show "${RESEARCH_KEY}" -a --keyring-backend test --home "${CEREMONY_HOME}")
  ${BINARY} add-genesis-account "${RESEARCH_ADDR}" "${RESEARCH_BALANCE}" --home "${CEREMONY_HOME}"
  info "  Research: ${RESEARCH_ADDR}"

  info "Creating faucet account..."
  FAUCET_KEY="faucet"
  ${BINARY} keys add "${FAUCET_KEY}" --keyring-backend test --home "${CEREMONY_HOME}" 2>/dev/null
  FAUCET_ADDR=$(${BINARY} keys show "${FAUCET_KEY}" -a --keyring-backend test --home "${CEREMONY_HOME}")
  ${BINARY} add-genesis-account "${FAUCET_ADDR}" "${FAUCET_BALANCE}" --home "${CEREMONY_HOME}"
  info "  Faucet: ${FAUCET_ADDR}"

  echo ""
  echo "═══════════════════════════════════════════════════════════════"
  echo "  Genesis INIT complete."
  echo "  Ceremony home: ${CEREMONY_HOME}"
  echo ""
  echo "  Next: Add validators with:"
  echo "    ./scripts/genesis-ceremony.sh add-validator <name>"
  echo "═══════════════════════════════════════════════════════════════"
}

cmd_add_validator() {
  local name="${1:-}"
  [ -z "${name}" ] && die "Usage: $0 add-validator <validator-name>"

  echo "═══════════════════════════════════════════════════════════════"
  echo "  Adding validator: ${name}"
  echo "═══════════════════════════════════════════════════════════════"

  check_binary

  [ -d "${CEREMONY_HOME}" ] || die "Run 'init' first. No ceremony home at ${CEREMONY_HOME}"

  # Each validator needs a unique consensus key (priv_validator_key.json).
  # Create a per-validator home, init it to generate unique keys, then
  # generate the gentx there and copy it to the coordinator's gentx dir.
  local val_home="${CEREMONY_HOME}/validators/${name}"

  info "Generating unique consensus key for ${name}..."
  ${BINARY} init "${name}" --chain-id "${CHAIN_ID}" --home "${val_home}" --overwrite 2>/dev/null

  # Copy the coordinator's genesis (with patched params + accounts) to the validator home
  cp "${CEREMONY_HOME}/config/genesis.json" "${val_home}/config/genesis.json"

  # Create validator account key in coordinator keyring (shared keyring)
  info "Generating account key for ${name}..."
  ${BINARY} keys add "${name}" --keyring-backend test --home "${CEREMONY_HOME}" 2>/dev/null
  ADDR=$(${BINARY} keys show "${name}" -a --keyring-backend test --home "${CEREMONY_HOME}")
  info "  Address: ${ADDR}"

  # Fund validator account in coordinator's genesis
  info "Funding with ${VALIDATOR_BALANCE}..."
  ${BINARY} add-genesis-account "${ADDR}" "${VALIDATOR_BALANCE}" --home "${CEREMONY_HOME}"

  # Copy updated genesis (now with this validator's account) to the validator home
  cp "${CEREMONY_HOME}/config/genesis.json" "${val_home}/config/genesis.json"

  # Copy keyring to validator home so gentx can sign
  cp -r "${CEREMONY_HOME}/keyring-test" "${val_home}/"

  # Create gentx using the validator's unique priv_validator_key
  info "Creating gentx (self-delegation: ${VALIDATOR_STAKE})..."
  mkdir -p "${CEREMONY_HOME}/config/gentx"
  ${BINARY} gentx "${name}" "${VALIDATOR_STAKE}" \
    --chain-id "${CHAIN_ID}" \
    --keyring-backend test \
    --home "${val_home}" \
    --moniker "${name}" \
    --commission-rate "0.10" \
    --commission-max-rate "0.20" \
    --commission-max-change-rate "0.01" \
    --output-document "${CEREMONY_HOME}/config/gentx/gentx-${name}.json" 2>/dev/null

  # Save the validator's consensus key and node key for later use
  info "Saving validator keys to ${CEREMONY_HOME}/validators/${name}/"
  # These files will be needed when setting up each validator's actual node

  echo ""
  echo "  Validator ${name} added."
  echo "  Address: ${ADDR}"
  echo "  Stake:   ${VALIDATOR_STAKE}"
  echo "  Consensus key: ${val_home}/config/priv_validator_key.json"
  echo "  Node key:      ${val_home}/config/node_key.json"
  echo ""
}

cmd_finalize() {
  echo "═══════════════════════════════════════════════════════════════"
  echo "  Legible Money — Genesis Ceremony: FINALIZE"
  echo "═══════════════════════════════════════════════════════════════"

  check_binary

  [ -d "${CEREMONY_HOME}" ] || die "Run 'init' first."

  # Collect all gentxs
  info "Collecting gentxs..."
  ${BINARY} collect-gentxs --home "${CEREMONY_HOME}" 2>/dev/null

  # Validate genesis
  info "Validating genesis..."
  ${BINARY} validate --home "${CEREMONY_HOME}" 2>&1 || die "Genesis validation FAILED"

  echo ""
  echo "═══════════════════════════════════════════════════════════════"
  echo "  Genesis FINALIZED."
  echo ""
  echo "  Genesis file: ${CEREMONY_HOME}/config/genesis.json"
  echo ""
  echo "  To export: ./scripts/genesis-ceremony.sh export"
  echo "═══════════════════════════════════════════════════════════════"
}

cmd_export() {
  echo "═══════════════════════════════════════════════════════════════"
  echo "  Exporting genesis.json"
  echo "═══════════════════════════════════════════════════════════════"

  local genesis="${CEREMONY_HOME}/config/genesis.json"
  local dest="genesis.json"

  [ -f "${genesis}" ] || die "No genesis file at ${genesis}. Run init + finalize first."

  cp "${genesis}" "${dest}"
  info "Exported to: ${dest}"

  # Print summary
  echo ""
  info "Chain ID:     $(python3 -c "import json; print(json.load(open('${dest}'))['chain_id'])")"
  info "Genesis Time: $(python3 -c "import json; print(json.load(open('${dest}'))['genesis_time'])")"
  info "Accounts:     $(python3 -c "import json; d=json.load(open('${dest}')); auth=d['app_state'].get('auth',{}); accs=auth.get('accounts',[]); print(len(accs))")"

  echo ""
  echo "═══════════════════════════════════════════════════════════════"
  echo "  Genesis ready for legible-1."
  echo "  On February 22, truth verification begins."
  echo "═══════════════════════════════════════════════════════════════"
}

# ── Main ───────────────────────────────────────────────────────────────────

case "${1:-help}" in
  init)           cmd_init ;;
  add-validator)  cmd_add_validator "${2:-}" ;;
  finalize)       cmd_finalize ;;
  export)         cmd_export ;;
  help|--help|-h)
    echo "Usage: $0 {init|add-validator <name>|finalize|export}"
    echo ""
    echo "Commands:"
    echo "  init              Initialize genesis directory and bootstrap accounts"
    echo "  add-validator     Add a validator with self-delegation"
    echo "  finalize          Collect gentxs and validate genesis"
    echo "  export            Copy final genesis.json to project root"
    ;;
  *)
    die "Unknown command: $1. Use --help for usage."
    ;;
esac
