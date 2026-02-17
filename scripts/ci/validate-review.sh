#!/usr/bin/env bash
# validate-review.sh — Validate that a reviewer has sufficient authority for a PR's risk tier.
#
# Usage: validate-review.sh <reviewer_username> <risk_number> <pr_author>
#
# Exits 0 if review is valid (binding), 1 if invalid.
# Outputs review_type={binding|soft|rejected} to GITHUB_OUTPUT.
set -euo pipefail

REVIEWER="${1:?Usage: validate-review.sh <reviewer> <risk_number> <pr_author>}"
RISK_NUMBER="${2:?Missing risk number (0-3)}"
PR_AUTHOR="${3:?Missing PR author}"

AGENTS_FILE="scripts/governance/known-agents.json"

echo "=== Review Validation ==="
echo "Reviewer: $REVIEWER"
echo "PR Risk:  R${RISK_NUMBER}"
echo "Author:   $PR_AUTHOR"

# Self-review check
if [ "$REVIEWER" = "$PR_AUTHOR" ]; then
  echo "::error::Self-review not allowed: $REVIEWER is both author and reviewer."
  if [ -n "${GITHUB_OUTPUT:-}" ]; then
    echo "review_type=rejected" >> "$GITHUB_OUTPUT"
    echo "review_reason=self-review" >> "$GITHUB_OUTPUT"
  fi
  exit 1
fi

# Look up reviewer in known-agents.json
REVIEWER_TIER=-1
REVIEWER_TYPE="unknown"

if [ -f "$AGENTS_FILE" ] && command -v jq &>/dev/null; then
  # Check if reviewer is a human
  HUMAN_ROLE=$(jq -r ".humans[\"$REVIEWER\"].role // \"\"" "$AGENTS_FILE" 2>/dev/null || echo "")
  if [ -n "$HUMAN_ROLE" ]; then
    REVIEWER_TYPE="human"
    REVIEWER_TIER=99
    echo "Reviewer is human (role: $HUMAN_ROLE)"
  else
    # Check if reviewer is a registered agent
    AGENT_TIER=$(jq -r ".agents[\"$REVIEWER\"].tier // \"unknown\"" "$AGENTS_FILE" 2>/dev/null || echo "unknown")
    if [ "$AGENT_TIER" != "unknown" ] && [ "$AGENT_TIER" != "null" ]; then
      REVIEWER_TYPE="agent"
      REVIEWER_TIER="$AGENT_TIER"
      echo "Reviewer is registered agent (tier: $REVIEWER_TIER)"

      # Check review accuracy if available
      ACCURACY=$(jq -r ".agents[\"$REVIEWER\"].review_stats.review_accuracy_bps // 10000" "$AGENTS_FILE" 2>/dev/null || echo "10000")
      if [ "$ACCURACY" -lt 5000 ]; then
        echo "::error::Reviewer $REVIEWER has review accuracy ${ACCURACY}bps (< 50%). Binding review authority revoked."
        if [ -n "${GITHUB_OUTPUT:-}" ]; then
          echo "review_type=rejected" >> "$GITHUB_OUTPUT"
          echo "review_reason=low-accuracy" >> "$GITHUB_OUTPUT"
        fi
        exit 1
      fi
    else
      echo "Reviewer not in agent registry — treating as unverified"
      REVIEWER_TYPE="unregistered"
      REVIEWER_TIER=-1
    fi
  fi
else
  echo "Warning: Cannot read $AGENTS_FILE or jq not available"
  # If we can't verify, assume human (GitHub handles human identity)
  REVIEWER_TYPE="human"
  REVIEWER_TIER=99
fi

# Determine review authority
# Binding review matrix:
#   Human:          R0-R3 (all)
#   Tier 3 (Guardian): R0-R1
#   Tier 2 (Bonded):   R0-R1
#   Tier 1 (Verified): R0
#   Tier 0 (Apprentice): none
#   Unregistered:       none

REVIEW_TYPE="rejected"

if [ "$REVIEWER_TYPE" = "human" ]; then
  REVIEW_TYPE="binding"
elif [ "$REVIEWER_TYPE" = "agent" ]; then
  case "$REVIEWER_TIER" in
    3)
      if [ "$RISK_NUMBER" -le 1 ]; then
        REVIEW_TYPE="binding"
      elif [ "$RISK_NUMBER" -le 2 ]; then
        REVIEW_TYPE="soft"
      else
        REVIEW_TYPE="soft"
      fi
      ;;
    2)
      if [ "$RISK_NUMBER" -le 1 ]; then
        REVIEW_TYPE="binding"
      else
        REVIEW_TYPE="soft"
      fi
      ;;
    1)
      if [ "$RISK_NUMBER" -eq 0 ]; then
        REVIEW_TYPE="binding"
      else
        REVIEW_TYPE="soft"
      fi
      ;;
    0)
      REVIEW_TYPE="soft"
      ;;
  esac
fi

echo ""
echo "Review verdict: $REVIEW_TYPE"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "review_type=$REVIEW_TYPE" >> "$GITHUB_OUTPUT"
  echo "reviewer_tier=$REVIEWER_TIER" >> "$GITHUB_OUTPUT"
  echo "reviewer_type=$REVIEWER_TYPE" >> "$GITHUB_OUTPUT"
fi

if [ "$REVIEW_TYPE" = "rejected" ]; then
  echo "::error::Reviewer $REVIEWER ($REVIEWER_TYPE, tier $REVIEWER_TIER) cannot provide any review for R${RISK_NUMBER} PRs."
  exit 1
fi

if [ "$REVIEW_TYPE" = "soft" ]; then
  echo "::warning::Review from $REVIEWER is advisory only (soft review) for R${RISK_NUMBER} PRs."
fi

echo "Review validation complete."
