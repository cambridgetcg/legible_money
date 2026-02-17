#!/bin/bash
# Auto-pull before reading files to get changes from other sessions.
# Runs as a PreToolUse hook on Read and Glob tool calls.

cd /Users/yu/Desktop/legible_money || exit 0

# Only pull if we're on main branch
BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
if [ "$BRANCH" != "main" ]; then
  exit 0
fi

# Only pull if there are no uncommitted changes (avoid conflicts)
if git diff --quiet 2>/dev/null && git diff --cached --quiet 2>/dev/null; then
  git pull --ff-only origin main 2>/dev/null
fi

exit 0
