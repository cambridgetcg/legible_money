#!/usr/bin/env bash
# classify-pr.sh — Classify PR risk tier (R0-R3) based on files changed.
# Usage: scripts/ci/classify-pr.sh [base_ref]
# Output: Sets RISK_TIER env var and prints classification details.
set -euo pipefail

BASE_REF="${1:-origin/main}"

# Get changed files relative to base
CHANGED=$(git diff --name-only "$BASE_REF"...HEAD 2>/dev/null || git diff --name-only "$BASE_REF" HEAD)

if [ -z "$CHANGED" ]; then
  echo "risk_tier=R0"
  echo "risk_label=trivial"
  echo "No files changed."
  exit 0
fi

# Stub modules — test-only changes in these are R0
STUB_MODULES="x/disputes|x/evidence_mgmt|x/research|x/ontology|x/channels|x/tree"

# R3 (Critical) patterns — consensus, auth, app wiring, CI
R3_PATTERNS="^app/(app|ante|ante_legible|abci|vote_extensions)\.go$"
R3_PATTERNS+="|^app/staking_adapter\.go$"
R3_PATTERNS+="|^x/auth/"
R3_PATTERNS+="|^x/staking/"
R3_PATTERNS+="|^x/knowledge/keeper/"
R3_PATTERNS+="|^x/knowledge/evaluation/"
R3_PATTERNS+="|^x/emergency/"
R3_PATTERNS+="|^x/gov/"
R3_PATTERNS+="|^x/bvm/vm/"
R3_PATTERNS+="|^\.github/"
R3_PATTERNS+="|^cmd/"

# R2 (Standard) patterns — module code, proto, non-critical app files
R2_PATTERNS="^x/.*/keeper/.*\.go$"
R2_PATTERNS+="|^x/.*/types/.*\.go$"
R2_PATTERNS+="|^proto/"
R2_PATTERNS+="|^app/gas\.go$"
R2_PATTERNS+="|^app/local_state\.go$"

# R1 (Low) patterns — tests, tools, scripts, devnet
R1_PATTERNS="_test\.go$"
R1_PATTERNS+="|^tools/"
R1_PATTERNS+="|^scripts/"
R1_PATTERNS+="|^devnet/"
R1_PATTERNS+="|^tests/"

# R0 (Trivial) patterns — docs, markdown, memory
R0_PATTERNS="\.md$"
R0_PATTERNS+="|^docs/"
R0_PATTERNS+="|^\.claude/"

max_tier=0

while IFS= read -r file; do
  [ -z "$file" ] && continue
  tier=0

  # Check if it's a test file in a stub module (always R0)
  if echo "$file" | grep -qE "_test\.go$"; then
    if echo "$file" | grep -qE "^($STUB_MODULES)/"; then
      tier=0
    elif echo "$file" | grep -qE "$R3_PATTERNS"; then
      # Test file in a critical module is still R1 (not R3)
      tier=1
    else
      tier=1
    fi
  elif echo "$file" | grep -qE "$R3_PATTERNS"; then
    tier=3
  elif echo "$file" | grep -qE "$R2_PATTERNS"; then
    tier=2
  elif echo "$file" | grep -qE "$R1_PATTERNS"; then
    tier=1
  elif echo "$file" | grep -qE "$R0_PATTERNS"; then
    tier=0
  else
    # Unknown file — default to R2 (standard)
    tier=2
  fi

  if [ "$tier" -gt "$max_tier" ]; then
    max_tier=$tier
  fi
done <<< "$CHANGED"

# Map tier to label
case $max_tier in
  0) label="trivial" ;;
  1) label="low" ;;
  2) label="standard" ;;
  3) label="critical" ;;
esac

echo "risk_tier=R${max_tier}"
echo "risk_label=${label}"
echo ""
echo "Files analyzed:"
echo "$CHANGED" | while IFS= read -r f; do
  [ -z "$f" ] && continue
  # Determine per-file tier for display
  ft=0
  if echo "$f" | grep -qE "_test\.go$"; then
    if echo "$f" | grep -qE "^($STUB_MODULES)/"; then
      ft=0
    else
      ft=1
    fi
  elif echo "$f" | grep -qE "$R3_PATTERNS"; then
    ft=3
  elif echo "$f" | grep -qE "$R2_PATTERNS"; then
    ft=2
  elif echo "$f" | grep -qE "$R1_PATTERNS"; then
    ft=1
  elif echo "$f" | grep -qE "$R0_PATTERNS"; then
    ft=0
  else
    ft=2
  fi
  echo "  R${ft}  ${f}"
done

echo ""
echo "PR Risk Tier: R${max_tier} (${label})"

# GitHub Actions output
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "risk_tier=R${max_tier}" >> "$GITHUB_OUTPUT"
  echo "risk_number=${max_tier}" >> "$GITHUB_OUTPUT"
  echo "risk_label=${label}" >> "$GITHUB_OUTPUT"
fi
