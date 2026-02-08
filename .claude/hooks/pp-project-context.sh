#!/bin/bash
# Project-level PP context injection for Legible Money
# Fired on UserPromptSubmit — adds project-specific context when PP is activated.

INPUT=$(cat)
PROMPT=$(echo "$INPUT" | jq -r '.prompt // ""')

if echo "$PROMPT" | grep -qiE '^(pp:|/pp\s|/pp$|use purpose prompter|use pp\b)'; then
  jq -n '{
    "hookSpecificOutput": {
      "additionalContext": "[PP:PROJECT-CONTEXT] Legible Money Protocol\n\nCritical patterns to follow:\n- BigInt for all monetary values\n- domainHash() for all hashing\n- Atomic state transitions (beginTransaction/commitTransaction/rollbackTransaction)\n- DID format: did:lgm:{32-hex}\n- 15 state sub-roots with namespaces 0x01-0x0f\n- Known bug: fee routing nonce 3x increment (state/transitions.ts:306-338)\n\nTest with: vitest (use describe.sequential for shared state)\nType check with: tsc --noEmit\n\nRead .claude/memory/CURRENT_STATE.md for latest project state."
    }
  }'
fi

exit 0
