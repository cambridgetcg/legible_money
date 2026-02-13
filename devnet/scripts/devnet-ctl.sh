#!/usr/bin/env bash
#
# devnet-ctl.sh — Unified CLI for Legible Money DevNet
#
# Usage: devnet-ctl.sh <command> [options]
#
# Commands:
#   start [--full|--explorer|--monitor|--all]  Start Docker devnet
#   stop                                       Stop devnet (preserves data)
#   status                                     Show devnet status + health
#   logs [service]                             Tail devnet logs
#   reset                                      Wipe all data and start fresh
#   local                                      Run single local validator
#   health                                     Check validator health endpoints
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEVNET_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_DIR="$(dirname "$DEVNET_DIR")"
COMPOSE_FILE="$DEVNET_DIR/docker-compose.yml"
LOCAL_DATA_DIR="$HOME/.lgm/devnet"

# ── Helpers ──────────────────────────────────────────────────

log()  { echo "[devnet] $*"; }
err()  { echo "[devnet] ERROR: $*" >&2; }

ensure_secrets() {
  if [ ! -f "$DEVNET_DIR/secrets/validator-1.key" ]; then
    log "Secrets not found — initializing devnet keys..."
    (cd "$PROJECT_DIR" && npx tsx devnet/scripts/init-devnet.ts)
  fi
}

# ── Commands ─────────────────────────────────────────────────

cmd_start() {
  local profiles=""
  for arg in "$@"; do
    case "$arg" in
      --full)     profiles="$profiles --profile fullnode" ;;
      --explorer) profiles="$profiles --profile explorer" ;;
      --monitor)  profiles="$profiles --profile monitoring" ;;
      --all)      profiles="--profile fullnode --profile explorer --profile monitoring" ;;
      *) err "Unknown start option: $arg"; exit 1 ;;
    esac
  done

  ensure_secrets
  log "Starting Docker devnet..."
  docker-compose -f "$COMPOSE_FILE" $profiles up -d

  echo ""
  log "Devnet started!"
  echo ""
  echo "  Validator 1 RPC:  http://localhost:8545"
  echo "  Validator 2 RPC:  http://localhost:8547"
  echo "  Validator 3 RPC:  http://localhost:8549"
  echo "  Faucet:           http://localhost:8080"

  if [[ "$profiles" == *"explorer"* ]]; then
    echo "  Explorer:         http://localhost:3000"
  fi
  if [[ "$profiles" == *"monitoring"* ]]; then
    echo "  Grafana:          http://localhost:3001 (admin/devnet)"
  fi
  echo ""
}

cmd_stop() {
  log "Stopping devnet (data preserved in Docker volumes)..."
  docker-compose -f "$COMPOSE_FILE" down
  log "Stopped."
}

cmd_status() {
  echo "=== Docker Compose Status ==="
  docker-compose -f "$COMPOSE_FILE" ps
  echo ""
  cmd_health
}

cmd_logs() {
  local service="${1:-}"
  if [ -n "$service" ]; then
    docker-compose -f "$COMPOSE_FILE" logs -f "$service"
  else
    docker-compose -f "$COMPOSE_FILE" logs -f
  fi
}

cmd_reset() {
  log "Resetting devnet — destroying all volumes..."
  docker-compose -f "$COMPOSE_FILE" down -v 2>/dev/null || true
  log "Volumes destroyed. Re-initializing..."
  (cd "$PROJECT_DIR" && npx tsx devnet/scripts/init-devnet.ts)
  log "Reset complete. Run 'devnet-ctl.sh start' to launch."
}

cmd_local() {
  ensure_secrets

  # Ensure persistent data directory
  mkdir -p "$LOCAL_DATA_DIR/state"
  log "Local validator data: $LOCAL_DATA_DIR"

  local config="$DEVNET_DIR/config/local-validator.json"
  local genesis="$DEVNET_DIR/config/genesis.json"

  if [ ! -f "$config" ]; then
    err "Config not found: $config"
    exit 1
  fi
  if [ ! -f "$genesis" ]; then
    err "Genesis not found: $genesis"
    exit 1
  fi

  log "Starting local single-validator node..."
  CONFIG_PATH="$config" \
  GENESIS_PATH="$genesis" \
  VALIDATOR_KEY_PATH="$DEVNET_DIR/secrets/validator-1.key" \
    exec npx tsx "$DEVNET_DIR/cmd/node.ts"
}

cmd_health() {
  echo "=== Validator Health ==="
  local ports=(8545 8547 8549)
  local names=("validator-1" "validator-2" "validator-3")

  for i in "${!ports[@]}"; do
    local port="${ports[$i]}"
    local name="${names[$i]}"
    if curl -sf "http://localhost:$port/health" >/dev/null 2>&1; then
      echo "  $name (localhost:$port): UP"
    else
      echo "  $name (localhost:$port): DOWN"
    fi
  done
  echo ""
}

cmd_help() {
  echo "Legible Money DevNet Controller"
  echo ""
  echo "Usage: devnet-ctl.sh <command> [options]"
  echo ""
  echo "Commands:"
  echo "  start [--full|--explorer|--monitor|--all]  Start Docker devnet"
  echo "  stop                                       Stop devnet (preserves data)"
  echo "  status                                     Show status + health"
  echo "  logs [service]                             Tail devnet logs"
  echo "  reset                                      Wipe data and reinitialize"
  echo "  local                                      Run single local validator"
  echo "  health                                     Check validator endpoints"
  echo ""
}

# ── Main dispatch ────────────────────────────────────────────

command="${1:-help}"
shift 2>/dev/null || true

case "$command" in
  start)  cmd_start "$@" ;;
  stop)   cmd_stop ;;
  status) cmd_status ;;
  logs)   cmd_logs "$@" ;;
  reset)  cmd_reset ;;
  local)  cmd_local ;;
  health) cmd_health ;;
  help|-h|--help) cmd_help ;;
  *)
    err "Unknown command: $command"
    cmd_help
    exit 1
    ;;
esac
