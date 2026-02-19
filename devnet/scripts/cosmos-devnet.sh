#!/usr/bin/env bash
#
# Legible Money — Cosmos SDK DevNet Launch Script
#
# Usage:
#   cosmos-devnet.sh              Build + init + start (default)
#   cosmos-devnet.sh start        Same as above
#   cosmos-devnet.sh reset        Remove devnet state
#   cosmos-devnet.sh stop         Stop the node
#   cosmos-devnet.sh status       Query node status
#
# Requirements: go (1.22+), jq

set -euo pipefail

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------
CHAIN_ID="lgm-devnet-1"
DENOM="ulgm"
MONIKER="lgm-devnet-validator"
KEYRING="test"
NODE_HOME="$HOME/.legbled-devnet"
BINARY="build/legbled"

# 1 LGM = 1,000,000 ulgm
LGM=1000000

# Genesis account balances (in ulgm)
VALIDATOR_BALANCE=$(( 50000 * LGM ))       # 50,000 LGM
FOUNDATION_BALANCE=$(( 22222222 * LGM ))   # ~22.2M LGM
RESEARCH_BALANCE=$(( 17111111 * LGM ))     # ~17.1M LGM
COMMUNITY_BALANCE=$(( 11111111 * LGM ))    # ~11.1M LGM
TEST_BALANCE=$(( 1000 * LGM ))             # 1,000 LGM each

# Validator self-delegation
SELF_DELEGATION=$(( 10000 * LGM ))         # 10,000 LGM

# Project root (script is in devnet/scripts/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()  { echo -e "\033[1;34m→\033[0m $*"; }
ok()    { echo -e "\033[1;32m✓\033[0m $*"; }
warn()  { echo -e "\033[1;33m!\033[0m $*"; }
err()   { echo -e "\033[1;31m✗\033[0m $*" >&2; }

legbled() {
    "$PROJECT_ROOT/$BINARY" "$@" --home "$NODE_HOME"
}

check_prereqs() {
    local missing=0
    if ! command -v go &>/dev/null; then
        err "go is required but not found"
        missing=1
    fi
    if ! command -v jq &>/dev/null; then
        err "jq is required but not found (brew install jq)"
        missing=1
    fi
    if [ $missing -ne 0 ]; then
        exit 1
    fi
}

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
build_binary() {
    info "Building legbled binary..."
    cd "$PROJECT_ROOT"
    mkdir -p build
    go build -o "$BINARY" ./cmd/legbled
    ok "Binary built: $BINARY ($(du -h "$BINARY" | cut -f1))"
}

# ---------------------------------------------------------------------------
# Init chain
# ---------------------------------------------------------------------------
init_chain() {
    info "Initializing chain ($CHAIN_ID)..."

    # Clean previous state
    if [ -d "$NODE_HOME" ]; then
        warn "Removing previous devnet state at $NODE_HOME"
        rm -rf "$NODE_HOME"
    fi

    legbled init "$MONIKER" --chain-id "$CHAIN_ID" --default-denom "$DENOM" >/dev/null 2>&1
    ok "Chain initialized"
}

# ---------------------------------------------------------------------------
# Create keys
# ---------------------------------------------------------------------------
create_keys() {
    info "Creating keys (keyring=$KEYRING)..."

    local keys=("validator" "foundation" "research" "community" "test1" "test2" "test3")
    for name in "${keys[@]}"; do
        legbled keys add "$name" --keyring-backend "$KEYRING" >/dev/null 2>&1
    done
    ok "Keys created: validator, foundation, research, community, test1, test2, test3"
}

# ---------------------------------------------------------------------------
# Add genesis accounts
# ---------------------------------------------------------------------------
add_genesis_accounts() {
    info "Adding genesis accounts..."

    local validator_addr
    validator_addr=$(legbled keys show validator -a --keyring-backend "$KEYRING")

    # Named accounts
    legbled add-genesis-account "$validator_addr" "${VALIDATOR_BALANCE}${DENOM}"

    # Foundation, research, community (keys already created in create_keys)
    for name in foundation research community; do
        local addr
        addr=$(legbled keys show "$name" -a --keyring-backend "$KEYRING")
        case $name in
            foundation) legbled add-genesis-account "$addr" "${FOUNDATION_BALANCE}${DENOM}" ;;
            research)   legbled add-genesis-account "$addr" "${RESEARCH_BALANCE}${DENOM}" ;;
            community)  legbled add-genesis-account "$addr" "${COMMUNITY_BALANCE}${DENOM}" ;;
        esac
    done

    # Test accounts
    for name in test1 test2 test3; do
        local addr
        addr=$(legbled keys show "$name" -a --keyring-backend "$KEYRING")
        legbled add-genesis-account "$addr" "${TEST_BALANCE}${DENOM}"
    done

    ok "Genesis accounts added"
}

# ---------------------------------------------------------------------------
# Patch genesis.json
# ---------------------------------------------------------------------------
patch_genesis() {
    info "Patching genesis.json..."

    local genesis="$NODE_HOME/config/genesis.json"

    # Single jq pass for all app_state + consensus patches
    jq '
    # --- CRITICAL: Enable vote extensions for PoT at height 1 ---
    .consensus.params.abci.vote_extensions_enable_height = "1" |

    # --- Staking bond denom ---
    .app_state.staking.params.bond_denom = "ulgm" |

    # --- Knowledge module: single-node compatible ---
    .app_state.knowledge.params.min_verifiers = 1 |

    # --- Bank: denom metadata ---
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

    # --- Governance: fast voting for devnet ---
    .app_state.gov.params.voting_period = "60s" |
    .app_state.gov.params.expedited_voting_period = "30s" |
    .app_state.gov.params.max_deposit_period = "60s" |
    .app_state.gov.params.min_deposit = [{"denom": "ulgm", "amount": "10000000"}] |

    # --- Slashing: lenient for devnet ---
    .app_state.slashing.params.signed_blocks_window = "100" |
    .app_state.slashing.params.slash_fraction_downtime = "0.010000000000000000"
    ' "$genesis" > "${genesis}.tmp" && mv "${genesis}.tmp" "$genesis"

    ok "Genesis patched"
}

# ---------------------------------------------------------------------------
# Generate gentx + collect
# ---------------------------------------------------------------------------
create_gentx() {
    info "Creating gentx (self-delegation: $(( SELF_DELEGATION / LGM )) LGM)..."

    legbled gentx validator "${SELF_DELEGATION}${DENOM}" \
        --chain-id "$CHAIN_ID" \
        --keyring-backend "$KEYRING" \
        --moniker "$MONIKER" >/dev/null 2>&1

    legbled collect-gentxs >/dev/null 2>&1
    ok "Gentx collected"
}

# ---------------------------------------------------------------------------
# Inject genesis axioms (optional — if template file exists)
# ---------------------------------------------------------------------------
inject_axioms() {
    local axiom_file="$PROJECT_ROOT/scripts/genesis-axioms-template.json"
    local genesis="$NODE_HOME/config/genesis.json"
    if [ -f "$axiom_file" ]; then
        info "Injecting genesis axioms..."
        go run "$PROJECT_ROOT/tools/axiom-loader/main.go" inject "$axiom_file" "$genesis" >/dev/null 2>&1
        ok "Axioms injected ($(jq '.app_state.knowledge.facts | length' "$genesis") facts)"
    fi
}

# ---------------------------------------------------------------------------
# Validate genesis
# ---------------------------------------------------------------------------
validate_genesis() {
    info "Validating genesis..."
    legbled validate-genesis >/dev/null 2>&1
    ok "Genesis valid"
}

# ---------------------------------------------------------------------------
# Patch CometBFT + app config
# ---------------------------------------------------------------------------
patch_configs() {
    info "Patching node config..."

    local config_toml="$NODE_HOME/config/config.toml"
    local app_toml="$NODE_HOME/config/app.toml"

    # config.toml — enable CORS, unsafe RPC for devnet, prometheus
    sed -i.bak 's/cors_allowed_origins = \[\]/cors_allowed_origins = ["*"]/' "$config_toml"
    sed -i.bak 's/^unsafe = false/unsafe = true/' "$config_toml"
    sed -i.bak 's/^prometheus = false/prometheus = true/' "$config_toml"

    # app.toml — enable unsafe CORS for REST API
    sed -i.bak 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/' "$app_toml"

    # Clean up sed backup files
    rm -f "${config_toml}.bak" "${app_toml}.bak"

    ok "Config patched (CORS, unsafe RPC, prometheus)"
}

# ---------------------------------------------------------------------------
# Print summary + start
# ---------------------------------------------------------------------------
print_summary() {
    local validator_addr
    validator_addr=$(legbled keys show validator -a --keyring-backend "$KEYRING" 2>/dev/null)

    echo ""
    echo "╔═══════════════════════════════════════════════════════╗"
    echo "║          Legible Money — Cosmos SDK DevNet            ║"
    echo "╠═══════════════════════════════════════════════════════╣"
    echo "║                                                       ║"
    echo "║  Chain ID:     $CHAIN_ID                        ║"
    echo "║  Block time:   2521ms                                 ║"
    echo "║  Denom:        ulgm (1 LGM = 10^6 ulgm)              ║"
    echo "║  Validator:    $validator_addr  ║"
    echo "║                                                       ║"
    echo "║  Endpoints:                                           ║"
    echo "║    RPC:        http://localhost:26657                  ║"
    echo "║    REST API:   http://localhost:1317                   ║"
    echo "║    gRPC:       localhost:9090                          ║"
    echo "║                                                       ║"
    echo "║  Key accounts: validator, foundation, research,       ║"
    echo "║                 community, test1, test2, test3         ║"
    echo "║                                                       ║"
    echo "╚═══════════════════════════════════════════════════════╝"
    echo ""
    echo "Quick test commands:"
    echo "  # Query status"
    echo "  curl -s localhost:26657/status | jq .result.sync_info.latest_block_height"
    echo ""
    echo "  # Query validator balance"
    echo "  $PROJECT_ROOT/$BINARY query bank balances $validator_addr --home $NODE_HOME"
    echo ""
    echo "  # Send tokens"
    echo "  $PROJECT_ROOT/$BINARY tx bank send validator test1 1000000ulgm \\"
    echo "    --chain-id $CHAIN_ID --keyring-backend test --home $NODE_HOME -y"
    echo ""
}

start_node() {
    info "Starting node..."
    echo ""
    legbled start --minimum-gas-prices "0.025${DENOM}"
}

# ---------------------------------------------------------------------------
# Commands: reset, stop, status
# ---------------------------------------------------------------------------
do_reset() {
    if [ -d "$NODE_HOME" ]; then
        info "Removing devnet state at $NODE_HOME..."
        rm -rf "$NODE_HOME"
        ok "Devnet state removed"
    else
        warn "No devnet state found at $NODE_HOME"
    fi
}

do_stop() {
    if pgrep -f "legbled.*start.*legbled-devnet" >/dev/null 2>&1; then
        info "Stopping legbled..."
        pkill -f "legbled.*start.*legbled-devnet" || true
        ok "Node stopped"
    else
        warn "No running legbled devnet process found"
    fi
}

do_status() {
    if ! curl -s --connect-timeout 2 http://localhost:26657/status >/dev/null 2>&1; then
        err "Node is not running (RPC not reachable at localhost:26657)"
        exit 1
    fi
    echo ""
    info "Node Status:"
    curl -s http://localhost:26657/status | jq '{
        node_info: .result.node_info.moniker,
        chain_id: .result.node_info.network,
        latest_block_height: .result.sync_info.latest_block_height,
        latest_block_time: .result.sync_info.latest_block_time,
        catching_up: .result.sync_info.catching_up,
        voting_power: .result.validator_info.voting_power
    }'
    echo ""
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    local cmd="${1:-start}"

    case "$cmd" in
        start|"")
            check_prereqs
            build_binary
            init_chain
            create_keys
            add_genesis_accounts
            patch_genesis
            create_gentx
            inject_axioms
            validate_genesis
            patch_configs
            print_summary
            start_node
            ;;
        reset)
            do_reset
            ;;
        stop)
            do_stop
            ;;
        status)
            do_status
            ;;
        *)
            echo "Usage: $0 {start|reset|stop|status}"
            exit 1
            ;;
    esac
}

main "$@"
