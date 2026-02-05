# Session Startup Protocol

> Run this at the start of every session for maximum continuity.

---

## Quick Start (Copy-Paste)

```
Read these files to restore context:
1. .claude/memory/CURRENT_STATE.md
2. .claude/memory/BLOCKED.md
3. .claude/sessions/[latest].md
```

---

## Full Context Restore

For complex work, also read:
```
4. .claude/memory/DECISIONS.md
5. .claude/context/CORE_TYPES.md
6. .claude/context/API_PATTERNS.md
```

---

## Session Start Checklist

1. [ ] Read CURRENT_STATE.md - know what's built
2. [ ] Read BLOCKED.md - know what's stuck
3. [ ] Read latest session log - know recent work
4. [ ] Check for new files in project since last session
5. [ ] Identify today's goals

---

## If Context Was Lost

If starting fresh with no prior context:

1. Read `.claude/CLAUDE.md` - full project overview
2. Read `docs/BLOCKCHAIN_ARCHITECTURE.md` - technical design
3. Read `consensus/pot/types.ts` - core types
4. Read `state/types.ts` - state layer types

---

## Directory Quick Reference

```
legible-money/
├── core/           # Crypto, logic, knowledge primitives
├── agents/         # Agent identity (4-layer keys)
├── network/        # libp2p networking
├── consensus/      # Proof of Truth engine
├── state/          # State layer (SMTs)
├── docs/           # Documentation
└── .claude/        # Session continuity
```

---

## Key Files by Topic

**Consensus**: `consensus/pot/types.ts`, `consensus/pot/engine.ts`
**State**: `state/types.ts`, `state/state_manager.ts`
**Identity**: `agents/identity/account.ts`
**Crypto**: `core/crypto/merkle.ts`, `core/crypto/vrf.ts`
**Logic**: `core/logic/zones.ts`

---

## Common Tasks

### Continue development
```
"Continue work on Legible Money. Read .claude/memory/CURRENT_STATE.md to see project status."
```

### Implement next phase
```
"Read .claude/memory/CURRENT_STATE.md and implement the next pending phase."
```

### Fix a bug
```
"Read relevant module files and fix [describe bug]."
```

### Add a feature
```
"Read .claude/CLAUDE.md for context, then implement [feature]."
```
