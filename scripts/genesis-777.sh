#!/usr/bin/env bash
# Genesis 777 — Full axiom pipeline: parse → validate → inject → launch
#
# Builds the chain binary, parses 15 domain markdown files into JSON,
# validates the axiom DAG, injects axioms as pre-verified facts into
# genesis.json, and initializes a single-validator devnet.
#
# Usage:
#   bash scripts/genesis-777.sh
#
# Environment:
#   NODE_HOME   — chain data directory (default: ~/.legbled-genesis777)
#   CHAIN_ID    — chain ID (default: lgm-genesis777-1)
#
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="$PROJECT_ROOT/build/legbled"
NODE_HOME="${NODE_HOME:-$HOME/.legbled-genesis777}"
CHAIN_ID="${CHAIN_ID:-lgm-genesis777-1}"
AXIOM_DIR="$PROJECT_ROOT/genesis-axioms"
AXIOM_JSON="$PROJECT_ROOT/scripts/axioms-777.json"

info()  { echo -e "\033[1;34m→\033[0m $*"; }
ok()    { echo -e "\033[1;32m✓\033[0m $*"; }
err()   { echo -e "\033[1;31m✗\033[0m $*" >&2; exit 1; }

echo ""
echo "╔═══════════════════════════════════════════════════════╗"
echo "║         Genesis 777 — Axiom Launch Pipeline           ║"
echo "╠═══════════════════════════════════════════════════════╣"
echo "║  Chain ID:   $CHAIN_ID                       ║"
echo "║  Home:       $NODE_HOME  ║"
echo "║  Axiom dir:  $AXIOM_DIR               ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""

# ──── Step 1: Build ────
info "Building legbled..."
cd "$PROJECT_ROOT" && make build
ok "Binary built: $BINARY"

# ──── Step 2: Parse MD → JSON ────
info "Parsing genesis axioms from $AXIOM_DIR..."
go run "$PROJECT_ROOT/tools/axiom-parser/main.go" \
    -dir "$AXIOM_DIR" -o "$AXIOM_JSON"

COUNT=$(python3 -c "import json; print(len(json.load(open('$AXIOM_JSON'))))")
if [ "$COUNT" -lt 700 ]; then
    err "Expected 700+ axioms, got $COUNT — check genesis-axioms/ directory"
fi
ok "Parsed $COUNT axioms → $AXIOM_JSON"

# ──── Step 3: Validate axiom DAG ────
info "Validating axiom DAG..."
go run "$PROJECT_ROOT/tools/axiom-loader/main.go" validate "$AXIOM_JSON"
ok "Axiom validation passed"

# ──── Step 4: Init chain ────
info "Initializing chain (chain-id=$CHAIN_ID)..."
rm -rf "$NODE_HOME"
$BINARY init genesis-node --chain-id "$CHAIN_ID" --home "$NODE_HOME" 2>/dev/null
ok "Chain initialized"

# ──── Step 5: Create validator key ────
info "Creating validator key..."
$BINARY keys add validator --keyring-backend test --home "$NODE_HOME" 2>/dev/null
VALIDATOR_ADDR=$($BINARY keys show validator -a --keyring-backend test --home "$NODE_HOME")
ok "Validator: $VALIDATOR_ADDR"

# ──── Step 6: Fund genesis account ────
info "Adding genesis account..."
$BINARY add-genesis-account "$VALIDATOR_ADDR" \
    1000000000000ulgm --home "$NODE_HOME"
ok "Account funded with 1,000,000 LGM"

# ──── Step 7: Create gentx ────
info "Creating gentx..."
$BINARY gentx validator 100000000000ulgm \
    --chain-id "$CHAIN_ID" --keyring-backend test --home "$NODE_HOME" 2>/dev/null
ok "Gentx created (100,000 LGM self-delegation)"

# ──── Step 8: Collect gentxs ────
info "Collecting gentxs..."
$BINARY collect-gentxs --home "$NODE_HOME" 2>/dev/null
ok "Gentxs collected"

# ──── Step 9: Configure node ────
info "Configuring node..."
GENESIS="$NODE_HOME/config/genesis.json"
CONFIG="$NODE_HOME/config/config.toml"

# Fast block times for testing
sed -i.bak 's/"timeout_commit": "5s"/"timeout_commit": "1s"/' "$CONFIG" 2>/dev/null || true

# Enable vote extensions at block 1 (required for PoT)
python3 -c "
import json
g = json.load(open('$GENESIS'))
g['consensus']['params']['abci']['vote_extensions_enable_height'] = '1'
g['app_state']['knowledge']['params']['min_verifiers'] = 1
json.dump(g, open('$GENESIS', 'w'), indent=2)
"
rm -f "$CONFIG.bak"
ok "Node configured (fast blocks, vote extensions)"

# ──── Step 10: Inject axioms into genesis ────
info "Injecting $COUNT axioms into genesis..."
go run "$PROJECT_ROOT/tools/axiom-loader/main.go" inject "$AXIOM_JSON" "$GENESIS"
ok "Axioms injected"

# ──── Step 11: Validate final genesis ────
info "Validating genesis..."
$BINARY validate-genesis --home "$NODE_HOME"
ok "Genesis valid"

# ──── Step 12: Report ────
GENESIS_SIZE=$(du -h "$GENESIS" | cut -f1)

echo ""
echo "╔═══════════════════════════════════════════════════════╗"
echo "║              Genesis 777 — Ready                      ║"
echo "╠═══════════════════════════════════════════════════════╣"
echo "║                                                       ║"
echo "║  Chain ID:      $CHAIN_ID                    ║"
echo "║  Home:          $NODE_HOME ║"
echo "║  Validator:     $VALIDATOR_ADDR  ║"
echo "║  Axioms:        $COUNT                               ║"
echo "║  Genesis size:  $GENESIS_SIZE                            ║"
echo "║                                                       ║"
echo "║  To start:                                            ║"
echo "║    $BINARY start --home $NODE_HOME  ║"
echo "║                                                       ║"
echo "║  To query axioms:                                     ║"
echo "║    $BINARY query knowledge list-facts \\  ║"
echo "║      --home $NODE_HOME                ║"
echo "║                                                       ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""
