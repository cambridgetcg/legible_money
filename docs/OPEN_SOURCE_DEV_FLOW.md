# Open-Source Development Flow

> **Status**: ACTIVE — All Claude Code sessions follow this workflow.
>
> **Scope**: How independent Claude Code sessions contribute to Legible Money
> using branches, PRs, and reviews — even on the private repo.
>
> **Extends**: `CONTRIBUTING.md` (general guide), `FILE_OWNERS.md` (parallel work rules),
> `.claude/CLAUDE.md` (session continuity protocol).

---

## The Model

Each Claude Code session is an **independent agent contributor**. Sessions don't
share working memory. They coordinate through git (branches, PRs) and the memory
system (`.claude/memory/`).

```
                    ┌─────────────────────┐
                    │   main (trunk)      │
                    │                     │
                    │  .claude/memory/    │  ← Shared truth
                    │  CURRENT_STATE.md   │
                    │  DECISIONS.md       │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
         ┌────▼────┐     ┌────▼────┐     ┌────▼────┐
         │Session A│     │Session B│     │Session C│
         │ branch  │     │ branch  │     │ branch  │
         └────┬────┘     └────┬────┘     └────┬────┘
              │                │                │
              │  PR #1         │  PR #2         │  PR #3
              └────────────────┼────────────────┘
                               │
                          Human review
                               │
                          Merge to main
```

**Rule**: No session commits directly to `main`. All work goes through branches and PRs.

---

## Session Lifecycle

### 1. Start: Read Trunk, Pick Work, Branch

```bash
# Pull latest main
git checkout main && git pull origin main

# Read project state (Claude does this automatically via .claude/CLAUDE.md)
# - .claude/memory/CURRENT_STATE.md
# - .claude/memory/DECISIONS.md
# - .claude/memory/BLOCKED.md
```

**Pick work** from one of:
- GitHub Issues (labeled `good-first-issue`, `agent-tier-0`, etc.)
- `.claude/memory/BLOCKED.md` open questions
- `docs/INTEGRATED_GROWTH_ROADMAP.md` next phase tasks
- Human-assigned task in the session prompt

**Create a feature branch**:

```bash
# Format: {type}-{scope}-{slug}
git checkout -b feat-schedule-add-tests
git checkout -b fix-knowledge-decay-bug
git checkout -b docs-update-api-reference
```

Branch naming extends `CONTRIBUTING.md:113-120` with scope qualifiers:
- `lip-XXX-description` — LIP implementations
- `fix-{scope}-{slug}` — Bug fixes (scope = module name)
- `feat-{scope}-{slug}` — Features
- `refactor-{scope}-{slug}` — Refactoring
- `docs-{slug}` — Documentation
- `test-{scope}-{slug}` — Tests only

> Note: `CONTRIBUTING.md` uses simpler names (`fix-issue-XXX`, `feat-description`).
> This document adds `{scope}` for multi-session disambiguation.

### 2. Claim: Prevent Collisions

Before starting work, check for collisions and claim:

**File-level claiming** (for conflict zones in `FILE_OWNERS.md:42-63`):

```typescript
// EDITING: session-{date}-{id} until {HH:MM UTC}
// Task: {brief description}
```

**Issue-level claiming** (for GitHub Issues):

```
Comment on issue: "Claiming this. Working on branch {branch-name}."
```

**Check before claiming**:
1. Is anyone else's branch touching the same module? (`git branch -r | grep {module}`)
2. Is the file claimed? (Check for `// EDITING:` comments)
3. Is the issue assigned? (Check GitHub)

**Claim duration**: 4 hours max for issue-level claims. File-level edit claims
(`// EDITING:` comments) follow `FILE_OWNERS.md` rules (30 min max).
If you can't finish, open a draft PR and note the state. Another session
can continue from your branch.

### 3. Work: Follow the Laws

**The 5 Execution Laws** (from `.claude/CLAUDE.md`):
1. No edit without read
2. No claim without citation (file:line)
3. No pattern without source
4. No placement without evidence
5. No completion without verification

**During work**:
- Commit frequently to your branch (small, atomic commits)
- Run `make pr-check` before each commit (see below)
- Follow `CONTRIBUTING.md` code patterns (BigInt, domain hashing, atomic state)
- Stay within your claimed scope

**Conflict zone rules** (from `FILE_OWNERS.md`):

| Situation | Action |
|-----------|--------|
| Adding new test file | Just create it (safe zone) |
| Adding new type | Add at END of types file (safe zone) |
| Modifying existing function | Claim file first (conflict zone) |
| Changing function signatures | Full coordination required |
| Touching `state/transitions.ts` | Always claim |

### 4. Validate: Local PR Check

Before opening a PR, run the full validation locally:

```bash
make pr-check
```

This runs (in order):
1. `npm run lint` — ESLint
2. `npm run typecheck` — TypeScript strict mode
3. `npm run test:run` — All vitest tests
4. `go build ./...` — Go chain compiles
5. `go test ./x/...` — All Go tests
6. `go vet ./x/...` — Go static analysis

**All must pass before opening a PR.** If any fail, fix before submitting.

### 5. Submit: Open a PR

```bash
# Push your branch
git push origin {branch-name}

# Open PR via gh CLI
gh pr create \
  --title "{type}({scope}): {description}" \
  --body "$(cat <<'EOF'
## Description

{What this PR does and why}

## Related Issues

Closes #{issue-number}

---

## The Three Laws Checklist

### LEGIBLE
- [x] Code purpose is clear without extensive comments
- [x] Confidence derivations are traceable

### HONEST
- [x] Uncertainty is acknowledged appropriately
- [x] No false certainty claims

### AGENT-FIRST
- [x] Design assumes computational scale
- [x] Works for autonomous operation

---

## Session Notes

**Branch**: {branch-name}
**Session**: {date}
**Work completed**: {summary}
**Work remaining**: {if any — for draft PRs}
**Tests**: {pass count} passing, {fail count} failing

## Test Plan

- [x] `make pr-check` passes locally
- [x] {specific test scenarios}

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

**PR labels** (add as applicable):
- `agent-contribution` — marks this as agent-authored
- `x/{module}` — which module is affected
- `tests` / `docs` / `feat` / `fix` — change type
- `draft` — if work is incomplete and another session should continue

### 6. Review: Human Gate

All PRs require human review (CODEOWNERS assigns `@yournameisai`).

**What the reviewer checks**:
1. Three Laws compliance (from PR template)
2. Tests pass and are meaningful
3. No security issues (especially for paths in `.github/CODEOWNERS:7-37`)
4. Code fits existing patterns
5. Scope matches the claimed issue

**Reviewer workflow**:
- Review PRs in submission order (oldest first) unless priority labels override
- If changes needed but no session is active: leave detailed comments with
  file:line pointers so any future session can address them
- For conflicting PRs: comment on the later PR noting the dependency
  (e.g., "Blocked by #42 — rebase after it merges")
- Batch-review related PRs when possible (same module, sequential changes)

**Review outcomes**:
- **Approved** → Merge (human merges, not agent)
- **Changes requested** → Next session picks up the PR and addresses feedback
- **Rejected** → Close PR, document why in issue

### 7. End: Clean Up and Hand Off

Whether the PR is merged or still in review:

```bash
# If PR is merged, switch back to main
git checkout main && git pull origin main

# If PR is still open (incomplete work), mark as draft
gh pr ready --undo  # marks as draft
```

**Update trunk memory** (only after merge to main):
- Prepend session summary to `.claude/memory/CURRENT_STATE.md`
- Log any decisions to `.claude/memory/DECISIONS.md`
- Clear resolved items from `.claude/memory/BLOCKED.md`

**If work is incomplete** (draft PR):
- Write "Session End Notes" in PR description:
  - What was completed
  - What remains (with file:line pointers)
  - Current blockers
- Label PR: `continuation-welcome`
- Any session can pick up the branch and continue

---

## Continuing Another Session's Work

When you find a draft PR labeled `continuation-welcome`:

```bash
# Fetch and checkout the branch
git fetch origin
git checkout {branch-name}

# Read the PR description for context ("Session End Notes")
gh pr view {pr-number}

# Continue working, commit to the same branch
# When done, mark PR as ready
gh pr ready
```

**Rules for continuation**:
- Read the full PR description before starting
- Don't rewrite completed work unless it's broken
- Add your own "Session Notes" section to the PR description
- Credit the original session: `Continued from session {date}`

---

## Memory System

### Trunk Memory (shared, on `main`)

| File | Purpose | Update When |
|------|---------|-------------|
| `.claude/memory/CURRENT_STATE.md` | Project state, session summaries | After PR merge |
| `.claude/memory/DECISIONS.md` | Technical decisions (append-only) | When decisions are made |
| `.claude/memory/BLOCKED.md` | Active blockers, open questions | When blockers change |

**Rule**: Only update trunk memory files on `main` after a PR merge.
Never modify them on a feature branch (they'll conflict).

### Branch-Local Notes

On your feature branch, you can create temporary notes:

```
.claude/sessions/{date}-{branch-slug}.md
```

These are branch-local session logs. They get merged to main with the PR
and serve as historical record.

### Reading Memory on Branch

At session start on a feature branch, you still read trunk memory:

```bash
# Your branch has trunk memory from when you branched
# If trunk has been updated since, rebase:
git fetch origin main
git rebase origin/main
```

This keeps your branch's memory files current with trunk.

---

## Scope Rules

Borrowed from the agent tier system (`docs/AGENT_CI_CD_WORKFLOW.md:184-203`).
These apply to **all sessions**, not just external agents:

| Scope | Definition | Example |
|-------|------------|---------|
| Single file | One source file + its test | `keeper.go` + `keeper_test.go` |
| Single module | One `x/` dir or TS directory | All of `x/knowledge/` |
| Multi-module | Multiple modules + interfaces | `x/knowledge/` + `x/staking/` + `expected_keepers.go` |

**PR size guideline**: Keep PRs under 500 LOC of non-test code. Tests can be
longer. If the work requires more, split into sequential PRs.

**Security-critical paths** (always require explicit human review, regardless of
scope):

```
core/crypto/            # Cryptographic operations
consensus/pot/          # Consensus engine
state/transitions.ts    # All state transitions
agents/identity/        # Key management
app/ante*.go            # Transaction validation
x/auth/                 # Authentication module
```

---

## Conflict Prevention

### Safe Zones (no coordination needed)

From `FILE_OWNERS.md:7-38`:
- Adding new test files
- Adding new types at END of types files
- Creating new implementation files
- Adding new documentation

### Conflict Zones (claim first)

From `FILE_OWNERS.md:42-63`:
- `consensus/pot/types.ts` — central types
- `state/transitions.ts` — all tx execution
- `app/app.go` — module wiring
- Any `index.ts` / `module.go` reordering

### Branch Collision Detection

Before creating a branch, check what's in flight:

```bash
# See all open branches
git branch -r | grep -v main

# See all open PRs
gh pr list

# Check if anyone is working on your target module
gh pr list --label "x/{module}"
```

If another branch touches the same files, coordinate:
- If the other PR is close to merging, wait for it
- If it's a draft, check PR notes for overlap
- If there's no overlap, proceed (git will catch real conflicts)

---

## Quick Reference

### Session Start Checklist

```
[ ] git checkout main && git pull
[ ] Read .claude/memory/CURRENT_STATE.md
[ ] Pick work (issue, roadmap task, or human-assigned)
[ ] Check for branch collisions (gh pr list)
[ ] Create feature branch
[ ] Claim issue/files if needed
```

### Session End Checklist

```
[ ] make pr-check passes
[ ] All changes committed to branch
[ ] PR opened (or updated if continuing)
[ ] Three Laws Checklist filled in PR
[ ] Session notes in PR description
[ ] If incomplete: mark draft, add continuation-welcome label
[ ] If merged: update trunk memory on main
```

### Common Commands

```bash
# Local validation
make pr-check

# Open PR
gh pr create --title "feat(scope): description"

# Check open PRs
gh pr list

# Continue someone's branch
git checkout {branch-name}

# Rebase on latest main
git fetch origin main && git rebase origin/main

# Mark PR as draft
gh pr ready --undo

# Mark PR as ready
gh pr ready
```

---

## Relationship to Other Documents

| Document | Role |
|----------|------|
| This file | **How** sessions work together (operational) |
| `CONTRIBUTING.md` | **How** to contribute in general (onboarding) |
| `FILE_OWNERS.md` | **What** files need coordination (reference) |
| `.claude/CLAUDE.md` | **What** Claude reads at session start (config) |
| `docs/AGENT_CI_CD_WORKFLOW.md` | **Future** adaptive CI pipeline (design spec) |
| `docs/INTEGRATED_GROWTH_ROADMAP.md` | **What** to build next (roadmap) |

---

_End of Open-Source Development Flow_
