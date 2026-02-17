#!/usr/bin/env bash
# sync-teams.sh — Sync known-agents.json tiers to GitHub teams.
#
# Requires: gh CLI authenticated with admin:org scope
# Teams: legible-guardians (Tier 3), legible-bonded (Tier 2+)
#
# Usage: scripts/governance/sync-teams.sh [--dry-run]
set -euo pipefail

DRY_RUN=false
if [ "${1:-}" = "--dry-run" ]; then
  DRY_RUN=true
  echo "[DRY RUN] No changes will be made."
fi

AGENTS_FILE="scripts/governance/known-agents.json"
ORG="${GITHUB_REPOSITORY_OWNER:-legible-protocol}"

if [ ! -f "$AGENTS_FILE" ]; then
  echo "Error: $AGENTS_FILE not found"
  exit 1
fi

if ! command -v jq &>/dev/null; then
  echo "Error: jq required"
  exit 1
fi

if ! command -v gh &>/dev/null; then
  echo "Error: gh CLI required"
  exit 1
fi

echo "=== Syncing Agent Teams ==="
echo "Org: $ORG"
echo ""

# Extract agents by tier
GUARDIANS=$(jq -r '.agents | to_entries[] | select(.value.tier >= 3) | .key' "$AGENTS_FILE")
BONDED=$(jq -r '.agents | to_entries[] | select(.value.tier >= 2) | .key' "$AGENTS_FILE")

echo "Guardians (Tier 3+):"
if [ -z "$GUARDIANS" ]; then
  echo "  (none)"
else
  echo "$GUARDIANS" | while read -r agent; do echo "  - $agent"; done
fi

echo ""
echo "Bonded (Tier 2+):"
if [ -z "$BONDED" ]; then
  echo "  (none)"
else
  echo "$BONDED" | while read -r agent; do echo "  - $agent"; done
fi

if [ "$DRY_RUN" = true ]; then
  echo ""
  echo "[DRY RUN] Would sync these users to GitHub teams."
  exit 0
fi

# Ensure teams exist
for team in legible-guardians legible-bonded; do
  if ! gh api "orgs/$ORG/teams/$team" &>/dev/null 2>&1; then
    echo "Creating team: $team"
    gh api "orgs/$ORG/teams" -f name="$team" -f privacy="closed" || echo "Warning: Could not create team $team"
  fi
done

# Sync guardians team
echo ""
echo "Syncing legible-guardians..."
if [ -n "$GUARDIANS" ]; then
  echo "$GUARDIANS" | while read -r agent; do
    echo "  Adding $agent..."
    gh api "orgs/$ORG/teams/legible-guardians/memberships/$agent" -f role="member" 2>/dev/null || echo "  Warning: Could not add $agent"
  done
fi

# Sync bonded team
echo "Syncing legible-bonded..."
if [ -n "$BONDED" ]; then
  echo "$BONDED" | while read -r agent; do
    echo "  Adding $agent..."
    gh api "orgs/$ORG/teams/legible-bonded/memberships/$agent" -f role="member" 2>/dev/null || echo "  Warning: Could not add $agent"
  done
fi

echo ""
echo "Team sync complete."
