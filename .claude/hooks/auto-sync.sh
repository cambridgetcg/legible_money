#!/bin/bash
# Auto-commit and push after file edits/writes.
# Runs as a PostToolUse hook on Edit and Write tool calls.

cd /Users/yu/Desktop/legible_money || exit 0

# Only sync if we're on main branch
BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
if [ "$BRANCH" != "main" ]; then
  exit 0
fi

# Stage all changes, commit if there's something to commit, push
git add -A 2>/dev/null
if ! git diff --cached --quiet 2>/dev/null; then
  git commit -m "auto-sync: $(date +%Y-%m-%dT%H:%M:%S)" --no-verify 2>/dev/null
  git push origin main 2>/dev/null
fi

exit 0
