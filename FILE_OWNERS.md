# File Ownership & Parallel Work Rules

> Guidelines for multiple agents working on Legible Money simultaneously.

---

## Safe Zones (No Coordination Needed)

These areas can be modified by any agent without coordination:

### Adding New Files

| Location | Safe to Add |
|----------|-------------|
| `tests/unit/{module}/` | New test files |
| `tests/integration/` | New integration test files |
| `docs/examples/` | New examples |
| `docs/` | New documentation |

### Adding to Existing Files

| File Type | Safe Addition |
|-----------|---------------|
| `types.ts` | New types at END of file (after existing types) |
| `index.ts` | New exports in designated "Add Features" section |
| Implementation files | New functions (not modifying existing) |

### Creating New Modules

```
Safe: Create entirely new directory with new files
      Example: core/new-feature/
               ├── types.ts
               ├── index.ts
               └── implementation.ts

Coordinate: Adding exports to parent index.ts
```

---

## Conflict Zones (Coordinate First)

These files are modified frequently and require coordination:

### High-Traffic Files

| File | Risk | Mitigation |
|------|------|------------|
| `consensus/pot/types.ts` | Central types | Claim before editing |
| `state/transitions.ts` | All tx execution | Claim before editing |
| `protocol/rpc/handlers/index.ts` | Handler registration | Claim before editing |
| `foundation/index.ts` | Core exports | Claim before editing |

### Structural Changes

| Change Type | Coordination Required |
|-------------|----------------------|
| Reordering exports in index.ts | Yes |
| Renaming types | Yes |
| Moving files between directories | Yes |
| Changing function signatures | Yes |

---

## Claiming Files

When you need exclusive access to a conflict zone:

### Claim Format

Add this comment at the top of the file:

```typescript
// EDITING: {agent-id} until {HH:MM UTC}
// Task: {brief description}
```

### Example

```typescript
// EDITING: claude-7a3b until 15:30 UTC
// Task: Adding StakeTransaction type

export type Transaction =
  | TransferTransaction
  | StakeTransaction  // Adding this
  | ...
```

### Claim Rules

1. **Duration**: Max 30 minutes per claim
2. **Release**: Remove comment when done
3. **Conflict**: If claimed, work on something else
4. **Expired**: If claim is >1 hour old, consider it abandoned

---

## Merge Conflict Prevention

### Pattern 1: Append-Only Types

```typescript
// types.ts - Add new types at END

// ============ Core Types ============
export interface ExistingType { ... }

// ============ Add New Types Below ============
// (Safe zone for parallel additions)
export interface NewTypeA { ... }  // Agent 1
export interface NewTypeB { ... }  // Agent 2 (no conflict)
```

### Pattern 2: Sectioned Index

```typescript
// index.ts - Each section is independent

// ============ Types (alphabetical) ============
export type { TypeA, TypeB, TypeC } from './types.js';

// ============ Core Functions ============
export { coreFunction } from './core.js';

// ============ Feature A (Agent 1) ============
export { featureA } from './feature_a.js';

// ============ Feature B (Agent 2) ============
export { featureB } from './feature_b.js';

// ============ Add New Features Below ============
```

### Pattern 3: One Function Per File

For high-churn areas, split into multiple files:

```
handlers/
├── index.ts           # Just re-exports
├── transfer.ts        # Agent 1 owns
├── stake.ts           # Agent 2 owns
└── verify.ts          # Agent 3 owns
```

---

## Test File Ownership

Test files are naturally conflict-free:

```
tests/unit/
├── agents/
│   ├── account.test.ts     # Mirrors agents/identity/account.ts
│   └── session.test.ts     # Mirrors agents/identity/session_keys.ts
├── state/
│   ├── transitions.test.ts # Mirrors state/transitions.ts
│   └── accounts.test.ts    # Mirrors state/accounts.ts
```

**Rule**: One test file per source file = no conflicts

---

## Coordination Channels

When you need to coordinate:

### For Quick Questions

Check `.claude/memory/BLOCKED.md` for known issues.

### For Design Decisions

Add to `.claude/memory/DECISIONS.md` with:

```markdown
## {Date} - {Decision Title}

**Context**: What situation prompted this?
**Decision**: What was decided?
**Rationale**: Why this choice?
**Alternatives**: What was considered?
```

### For Work In Progress

Add to `.claude/memory/CURRENT_STATE.md`:

```markdown
### In Progress
- {Agent}: Working on {feature} in {files}
```

---

## Emergency Conflict Resolution

If you discover a conflict:

1. **Don't force push** - preserves history
2. **Read both changes** - understand intent
3. **Prefer newer logic** - if functionally equivalent
4. **Ask for help** - if unclear which to keep
5. **Document** - note resolution in commit message

---

## Quick Reference

| Situation | Action |
|-----------|--------|
| Adding new test | Just create file |
| Adding new type | Add at END of types.ts |
| Adding new function | Create new file, add export |
| Modifying existing function | Claim file first |
| Restructuring | Full coordination required |

---

## See Also

- `ARCHITECTURE.md` - Directory structure
- `AGENT_INDEX.md` - Navigation decision trees
- `CONTRIBUTING.md` - Full contribution guide
