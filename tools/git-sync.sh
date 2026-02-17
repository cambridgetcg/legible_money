#!/usr/bin/env bash
#
# git-sync.sh — Background auto-pull for legible_money
#
# Usage:
#   ./tools/git-sync.sh              # run in foreground
#   ./tools/git-sync.sh &            # run in background
#   ./tools/git-sync.sh --once       # single check, no loop
#
# Environment:
#   GIT_SYNC_INTERVAL=30             # seconds between checks (default: 30)
#   GIT_SYNC_BRANCH=main             # branch to track (default: main)
#   GIT_SYNC_REMOTE=origin           # remote name (default: origin)
#   GIT_SYNC_LOG=/dev/stderr         # log destination (default: stderr)
#   GIT_SYNC_STASH=0                 # set to 1 to auto-stash dirty trees

set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
INTERVAL="${GIT_SYNC_INTERVAL:-30}"
BRANCH="${GIT_SYNC_BRANCH:-main}"
REMOTE="${GIT_SYNC_REMOTE:-origin}"
LOG="${GIT_SYNC_LOG:-/dev/stderr}"
AUTO_STASH="${GIT_SYNC_STASH:-0}"
ONCE=0

[ "${1:-}" = "--once" ] && ONCE=1

log() {
  echo "[git-sync $(date +%H:%M:%S)] $*" >> "$LOG"
}

sync_once() {
  cd "$REPO_DIR"

  # Ensure we're on the right branch
  current_branch=$(git branch --show-current 2>/dev/null)
  if [ "$current_branch" != "$BRANCH" ]; then
    log "On branch '$current_branch', not '$BRANCH' — skipping"
    return 0
  fi

  # Fetch latest
  if ! git fetch "$REMOTE" "$BRANCH" --quiet 2>/dev/null; then
    log "Fetch failed (offline or remote unreachable)"
    return 0
  fi

  # Compare
  local_head=$(git rev-parse HEAD)
  remote_head=$(git rev-parse "$REMOTE/$BRANCH")

  if [ "$local_head" = "$remote_head" ]; then
    return 0  # nothing to do, stay silent
  fi

  # Check if we're behind, ahead, or diverged
  local_only=$(git rev-list "$REMOTE/$BRANCH..HEAD" --count)
  remote_only=$(git rev-list "HEAD..$REMOTE/$BRANCH" --count)

  if [ "$remote_only" -eq 0 ]; then
    log "Local is $local_only commit(s) ahead — nothing to pull"
    return 0
  fi

  if [ "$local_only" -gt 0 ]; then
    log "Diverged: $local_only local, $remote_only remote — manual merge needed"
    return 1
  fi

  # We're behind. Check working tree.
  if ! git diff --quiet || ! git diff --cached --quiet; then
    if [ "$AUTO_STASH" = "1" ]; then
      log "Stashing local changes before pull"
      git stash push -m "git-sync auto-stash $(date +%Y%m%d-%H%M%S)" --quiet
      git rebase "$REMOTE/$BRANCH" --quiet
      log "Pulled $remote_only commit(s), restoring stash"
      git stash pop --quiet || {
        log "WARNING: Stash pop had conflicts — resolve manually"
        return 1
      }
    else
      log "Behind by $remote_only commit(s) but working tree is dirty — skipping (set GIT_SYNC_STASH=1 to auto-stash)"
      return 0
    fi
  else
    # Clean tree — safe to rebase
    git rebase "$REMOTE/$BRANCH" --quiet
    log "Pulled $remote_only commit(s): $(git log --oneline -1)"
  fi
}

# Main
log "Watching $REMOTE/$BRANCH in $REPO_DIR (interval: ${INTERVAL}s)"

if [ "$ONCE" = "1" ]; then
  sync_once
  exit $?
fi

while true; do
  sync_once || true
  sleep "$INTERVAL"
done
