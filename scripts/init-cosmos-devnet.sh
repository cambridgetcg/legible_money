#!/bin/bash
# Initialize a 3-node Cosmos devnet for Legible Money.
#
# Usage:
#   ./scripts/init-cosmos-devnet.sh
#
# This creates 3 validator nodes with:
#   - Chain ID: lgm-devnet-1
#   - Bond denom: ulgm
#   - Block time: 2521ms
#   - Validators with initial stakes

set -euo pipefail

CHAIN_ID="lgm-devnet-1"
BINARY="build/legbled"
DENOM="ulgm"
NUM_VALIDATORS=3
BASE_PORT=26600
INITIAL_BALANCE="1000000000000${DENOM}"  # 1,000,000 LGM
STAKE_AMOUNT="100000000000${DENOM}"       # 100,000 LGM

echo "=== Initializing Legible Money Cosmos Devnet ==="
echo "Chain ID:    ${CHAIN_ID}"
echo "Validators:  ${NUM_VALIDATORS}"
echo "Bond Denom:  ${DENOM}"
echo ""

# Build binary if not present
if [ ! -f "${BINARY}" ]; then
  echo "Building legbled binary..."
  go build -o "${BINARY}" ./cmd/legbled
fi

# Clean previous devnet state
rm -rf ~/.legbled-devnet

for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
  NODE_HOME="${HOME}/.legbled-devnet/node${i}"
  MONIKER="validator-${i}"
  P2P_PORT=$((BASE_PORT + i * 10))
  RPC_PORT=$((P2P_PORT + 1))

  echo "--- Initializing node ${i} (${MONIKER}) ---"

  # Initialize node
  ${BINARY} init ${MONIKER} --chain-id ${CHAIN_ID} --home ${NODE_HOME} 2>/dev/null

  # Create validator key
  ${BINARY} keys add ${MONIKER} --keyring-backend test --home ${NODE_HOME} 2>/dev/null

  # Get address
  ADDR=$(${BINARY} keys show ${MONIKER} -a --keyring-backend test --home ${NODE_HOME})
  echo "  Address: ${ADDR}"

  # Add genesis account
  ${BINARY} add-genesis-account ${ADDR} ${INITIAL_BALANCE} --home ${NODE_HOME}

  if [ ${i} -eq 0 ]; then
    # First node: create gentx
    ${BINARY} gentx ${MONIKER} ${STAKE_AMOUNT} \
      --chain-id ${CHAIN_ID} \
      --keyring-backend test \
      --home ${NODE_HOME} \
      --moniker ${MONIKER} 2>/dev/null
  fi

  echo "  Node ${i} initialized at ${NODE_HOME}"
done

echo ""
echo "=== Devnet Initialization Complete ==="
echo ""
echo "To start the chain:"
echo "  ${BINARY} start --home ~/.legbled-devnet/node0"
echo ""
echo "To query balances:"
echo "  ${BINARY} query bank balances <address> --node tcp://localhost:$((BASE_PORT + 1))"
echo ""
echo "To send LGM via IBC (Phase 1):"
echo "  ${BINARY} tx ibc-transfer transfer transfer channel-0 <receiver> 1000000${DENOM} --from validator-0"
