#!/bin/bash

# Legible Money Devnet Quick Start
# Usage: ./start.sh [options]
#
# Options:
#   --init      Initialize devnet first (generate keys)
#   --full      Include fullnode
#   --explorer  Include block explorer
#   --monitor   Include Prometheus + Grafana
#   --all       Include all optional services
#   --reset     Reset all data and start fresh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEVNET_DIR="$(dirname "$SCRIPT_DIR")"

cd "$DEVNET_DIR"

# Parse arguments
INIT=false
PROFILES=""
RESET=false

for arg in "$@"; do
  case $arg in
    --init)
      INIT=true
      ;;
    --full)
      PROFILES="$PROFILES --profile fullnode"
      ;;
    --explorer)
      PROFILES="$PROFILES --profile explorer"
      ;;
    --monitor)
      PROFILES="$PROFILES --profile monitoring"
      ;;
    --all)
      PROFILES="--profile fullnode --profile explorer --profile monitoring"
      ;;
    --reset)
      RESET=true
      ;;
    *)
      echo "Unknown option: $arg"
      exit 1
      ;;
  esac
done

# Reset if requested
if [ "$RESET" = true ]; then
  echo "Resetting devnet..."
  docker-compose down -v 2>/dev/null || true
  INIT=true
fi

# Initialize if requested or secrets don't exist
if [ "$INIT" = true ] || [ ! -f "secrets/validator-1.key" ]; then
  echo "Initializing devnet..."
  cd ..
  npx ts-node devnet/scripts/init-devnet.ts
  cd devnet
fi

# Start services
echo ""
echo "Starting Legible Money Devnet..."
echo ""

docker-compose $PROFILES up -d

echo ""
echo "Devnet started!"
echo ""
echo "Endpoints:"
echo "  Validator 1 RPC:  http://localhost:8545"
echo "  Validator 2 RPC:  http://localhost:8547"
echo "  Validator 3 RPC:  http://localhost:8549"
echo "  Faucet:           http://localhost:8080"

if [[ "$PROFILES" == *"explorer"* ]] || [[ "$PROFILES" == *"fullnode --profile explorer --profile monitoring"* ]]; then
  echo "  Explorer:         http://localhost:3000"
fi

if [[ "$PROFILES" == *"monitoring"* ]] || [[ "$PROFILES" == *"fullnode --profile explorer --profile monitoring"* ]]; then
  echo "  Grafana:          http://localhost:3001 (admin/devnet)"
fi

echo ""
echo "Commands:"
echo "  Logs:   docker-compose logs -f"
echo "  Stop:   docker-compose down"
echo "  Reset:  ./scripts/start.sh --reset"
