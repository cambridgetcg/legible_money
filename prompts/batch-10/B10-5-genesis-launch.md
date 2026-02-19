# B10-5 — Genesis Launch Script: Parse → Validate → Inject → Boot

## Context
We need an end-to-end script that takes the 15 domain markdown files,
converts them to JSON, validates, injects into genesis.json, and boots
a single-validator devnet with all 777 axioms loaded at block 0.

## Current State
- `devnet/scripts/cosmos-devnet.sh` — existing devnet launcher
- `scripts/testnet-genesis.sh` — existing genesis config script (calls axiom-loader inject)
- `tools/axiom-parser/main.go` — (from B10-1) converts MD → JSON
- `tools/axiom-loader/main.go` — validates and injects JSON into genesis
- Build binary: `make build` → `build/legbled`

## Task

### 1. Create `scripts/genesis-777.sh`

An all-in-one script that:

```bash
#!/usr/bin/env bash
# Genesis 777 — Full axiom pipeline: parse → validate → inject → launch
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="$PROJECT_ROOT/build/legbled"
NODE_HOME="${NODE_HOME:-$HOME/.legbled-genesis777}"
CHAIN_ID="${CHAIN_ID:-lgm-genesis777-1}"
AXIOM_DIR="$PROJECT_ROOT/genesis-axioms"
AXIOM_JSON="$PROJECT_ROOT/scripts/axioms-777.json"

# Step 1: Build
echo "=== Building legbled ==="
cd "$PROJECT_ROOT" && make build

# Step 2: Parse MD → JSON
echo "=== Parsing 777 genesis axioms ==="
go run "$PROJECT_ROOT/tools/axiom-parser/main.go" \
    -dir "$AXIOM_DIR" -o "$AXIOM_JSON"

# Verify count
COUNT=$(jq length "$AXIOM_JSON")
if [ "$COUNT" -ne 777 ]; then
    echo "ERROR: Expected 777 axioms, got $COUNT"
    exit 1
fi
echo "✓ Parsed $COUNT axioms"

# Step 3: Init chain
echo "=== Initializing chain ==="
rm -rf "$NODE_HOME"
$BINARY init genesis-node --chain-id "$CHAIN_ID" --home "$NODE_HOME"

# Step 4: Create validator key
$BINARY keys add validator --keyring-backend test --home "$NODE_HOME"
VALIDATOR_ADDR=$($BINARY keys show validator -a --keyring-backend test --home "$NODE_HOME")

# Step 5: Add genesis account with tokens
$BINARY genesis add-genesis-account "$VALIDATOR_ADDR" \
    1000000000000ulgm --home "$NODE_HOME"

# Step 6: Create gentx
$BINARY genesis gentx validator 100000000000ulgm \
    --chain-id "$CHAIN_ID" --keyring-backend test --home "$NODE_HOME"

# Step 7: Collect gentxs
$BINARY genesis collect-gentxs --home "$NODE_HOME"

# Step 8: Configure genesis.json with module params
GENESIS="$NODE_HOME/config/genesis.json"
# Set fast block times for testing
sed -i.bak 's/"timeout_commit": "5s"/"timeout_commit": "1s"/' \
    "$NODE_HOME/config/config.toml" 2>/dev/null || true

# Step 9: Validate axioms
echo "=== Validating axiom DAG ==="
go run "$PROJECT_ROOT/tools/axiom-loader/main.go" validate "$AXIOM_JSON"

# Step 10: Inject axioms into genesis
echo "=== Injecting 777 axioms into genesis ==="
go run "$PROJECT_ROOT/tools/axiom-loader/main.go" inject "$AXIOM_JSON" "$GENESIS"

# Step 11: Validate genesis
echo "=== Validating genesis ==="
$BINARY genesis validate-genesis --home "$NODE_HOME"

# Step 12: Report
echo ""
echo "=== Genesis 777 Ready ==="
echo "  Chain ID:    $CHAIN_ID"
echo "  Home:        $NODE_HOME"
echo "  Validator:   $VALIDATOR_ADDR"
echo "  Axioms:      $COUNT"
echo ""
echo "To start: $BINARY start --home $NODE_HOME"
echo "To query axioms: $BINARY query knowledge facts --home $NODE_HOME --node tcp://localhost:26657"
```

### 2. Launch and Verify Boot

After the script succeeds, start the chain and verify:
```bash
# Start in background
$BINARY start --home $NODE_HOME &
sleep 5

# Check block production
$BINARY status --home $NODE_HOME --node tcp://localhost:26657

# Query facts count
$BINARY query knowledge facts --count-total --home $NODE_HOME --node tcp://localhost:26657

# Query specific axioms
$BINARY query knowledge fact MATH-001 --home $NODE_HOME --node tcp://localhost:26657
$BINARY query knowledge fact THEO-001 --home $NODE_HOME --node tcp://localhost:26657
$BINARY query knowledge fact AGRT-000 --home $NODE_HOME --node tcp://localhost:26657

# Stop
kill %1
```

### 3. Handle Potential Issues

#### Genesis too large
777 items with full text may make genesis.json large. Check size after injection.
If > 50MB, we may need to compress statement text or load axioms via InitGenesis
code rather than genesis.json.

#### Module InitGenesis ordering
The knowledge module's InitGenesis needs to process domains before facts
(facts reference domains). Verify `DefaultDomains()` includes all 15+ domains
before any facts are loaded.

#### Fact reference validation at genesis
`GenesisState.Validate()` checks that all fact references point to existing facts.
With 777 axioms that cross-reference each other, the insertion order matters.
Currently `InitGenesis` calls `SetFact` sequentially — references to
not-yet-inserted facts will fail validation if checked during insertion.

**Fix if needed:** Either:
a) Disable reference checking during genesis (add a `isGenesis` flag)
b) Sort facts in topological order before insertion
c) Do a two-pass: insert all facts first, then validate references

The parser (B10-1) should output in topological order. Verify this.

### 4. Documentation

Add a section to `docs/GENESIS-AXIOMS.md`:
```markdown
## Loading Genesis Axioms

### Quick Start
```bash
bash scripts/genesis-777.sh
```

### Manual Pipeline
1. Parse: `go run tools/axiom-parser/main.go -dir genesis-axioms/ -o axioms.json`
2. Validate: `go run tools/axiom-loader/main.go validate axioms.json`
3. Inject: `go run tools/axiom-loader/main.go inject axioms.json genesis.json`
4. Boot: `legbled start --home ~/.legbled`
```

## Commit
```
feat(scripts): genesis-777.sh — full pipeline from MD to running chain

End-to-end script: parse 15 domain files → validate 777-item DAG →
inject into genesis.json → boot single-validator devnet. Includes
genesis validation and axiom count verification.
```
