# Agent-Speed Pull Request Workflow

Legible Money is developed primarily by AI agents producing PRs at 10-100x human speed. This document describes the PR workflow that scales review to match contribution velocity.

## Risk Classification

Every PR is auto-classified into 4 tiers based on files touched. The highest-risk file determines the PR's tier.

| Tier | Label | Paths | Examples |
|------|-------|-------|---------|
| **R0** | Trivial | `docs/**`, `*.md`, `_test.go` in stub modules, `.claude/memory/**` | Fix typo, add docstring, add test to `x/ontology` |
| **R1** | Low | `_test.go` (non-stub), `tools/**`, `scripts/**`, `devnet/**` | Add keeper tests, update build script |
| **R2** | Standard | `x/*/keeper/*.go`, `x/*/types/*.go`, `proto/**`, `app/gas.go` | Add spending limit, new proto message |
| **R3** | Critical | `app/{app,ante*,abci,vote_extensions}.go`, `x/{auth,staking,knowledge,emergency,gov}/**`, `x/bvm/vm/*`, `.github/**`, `cmd/**` | Change evaluation pipeline, modify ante handler |

Classification runs automatically in CI via `scripts/ci/classify-pr.sh` and applies a `risk-R{0-3}` label.

**Stub modules** (test changes are R0): `x/disputes`, `x/evidence_mgmt`, `x/research`, `x/ontology`, `x/channels`, `x/tree`

## Agent Tiers and Authoring Permissions

Mirrors the validator tier system. Agents earn contribution rights through track record.

| Agent Tier | Can Author | Binding Review | Auto-Merge |
|------------|-----------|----------------|------------|
| 0 (Apprentice) | R0 only | None | None |
| 1 (Verified) | R0, R1 | R0 | R0 |
| 2 (Bonded) | R0-R2 | R0, R1 | R0, R1 |
| 3 (Guardian) | R0-R3 | R0-R1 | R0, R1 |
| Human | All | All | All |

### Tier Advancement

| Tier | Contribution Req | Review Req | Stake |
|------|-----------------|-----------|-------|
| 0 | None | None | 0 |
| 1 | 22 merged PRs, 77%+ acceptance | N/A | 0 |
| 2 | Tier 1 + stake | 11 accurate reviews | 1,111 LGM |
| 3 | 333 merged PRs, 77%+, 0 reverts in 10 epochs | 33 reviews, 77%+ accuracy | 11,111 LGM |

### Consequences

- Review accuracy < 50%: lose binding review authority
- 3 consecutive false approvals: temporary review ban (10 epochs)

## Approval Thresholds

| PR Risk | Min Approvals | Human Required? | Soak Time |
|---------|--------------|-----------------|-----------|
| R0 | 0 (auto-merge) | No | 0 |
| R1 | 1 binding | No | 1 hour |
| R2 | 1 binding + 1 soft | No (if 2 Guardian reviews) | 4 hours |
| R3 | 1 human + 1 binding | **Yes, always** | 24 hours |

## Auto-Merge

PRs that pass ALL gates merge without human intervention:

1. Risk tier R0 or R1
2. All CI jobs green (build, test, lint)
3. Agent CI jobs green (scope, secrets)
4. Author is Tier 1+
5. Test count does not decrease (monotonicity)
6. < 200 source LOC changed (excluding tests/docs)
7. No merge conflicts
8. Review threshold met for tier
9. Soak time elapsed
10. No `hold` or `do-not-merge` label

### Safety Valves

- `hold` / `do-not-merge` label blocks auto-merge
- Any human can block by requesting changes
- Rate limit: max 10 auto-merges per hour
- If auto-merged commit breaks main CI, soak time doubles for 24h

## Agent Review Protocol

Agents review each other's PRs with structured, verifiable reviews.

### Review Format

```
/review verdict:{approve|request_changes|comment}
Confidence: {0-100}
Agent-DID: did:lgm:{hex}

## Analysis
[File-by-file review]

## Verdict Justification
[Why this verdict]

## Risk Assessment
Risk Tier: R{0-3}
Regression Risk: {low|medium|high}
```

### Validation Rules

- Reviewer registered in `scripts/governance/known-agents.json`
- Reviewer tier sufficient for PR risk tier (see matrix above)
- Reviewer is not the author (self-review prevention)
- Confidence >= 70 for binding review
- Reviews are recorded in `scripts/governance/review-ledger.json`

## Work Claims

Prevents merge conflicts when multiple agents work in parallel.

### Claiming Files

1. Comment `/claim [file paths]` on an issue
2. Claim reserves file paths for up to 4 hours
3. Other agents see warnings when touching claimed files
4. Claims auto-expire; cleared when work PR merges

### Mandatory Claims (Hard Failure)

These paths require a claim before any PR can touch them:
- `app/app.go` — module wiring
- `x/knowledge/keeper/` — evaluation pipeline
- Cross-module `proto/` files

### Advisory Claims (Warning Only)

All other paths — warning in CI but not blocking.

## Merge Queue

Uses GitHub's built-in merge queue for batched merging.

- **Batch size**: Up to 5 PRs per group
- **Required checks**: build, test, lint
- **Merge method**: Squash
- **Priority**: R3 > R2 > R1 > R0 (critical changes first)
- **Ejection**: Failed batch bisects to find culprit, labels `queue-ejected`

## Release Pipeline

### Versioning

`vMAJOR.MINOR.PATCH`:
- **MAJOR**: Consensus-breaking (chain upgrade)
- **MINOR**: Features
- **PATCH**: Fixes

### Cadence

| Type | Format | Trigger |
|------|--------|---------|
| Nightly | `vX.Y.Z-nightly.YYYYMMDD` | Automated from main |
| RC | `vX.Y.Z-rc.N` | Manual tag at milestone |
| Stable | `vX.Y.Z` | RC soaks 48h on devnet |

### Binary Targets

- `linux/amd64`, `linux/arm64`
- `darwin/amd64`, `darwin/arm64`

## Chain Upgrade Flow

1. LIP submitted (consensus category, 1000 LGM stake)
2. Upgrade plan attached (name, height, binary info)
3. Quadratic voting (75% support, 15% quorum)
4. Accepted: `x/upgrade` schedules halt at target height
5. Release pipeline builds upgrade binary
6. T-7d: validators test RC on devnet
7. T-0: chain halts, validators swap binary, restart
8. Upgrade handler runs migrations, chain resumes

## CODEOWNERS

Review authority is defined in `.github/CODEOWNERS`:
- **R3 paths**: `@yournameisai` only
- **R2 paths**: `@yournameisai` + `@legible-guardians`
- **R1 paths**: `@yournameisai` + `@legible-bonded`
- **R0 paths**: `@yournameisai` + `@legible-guardians`

Teams `@legible-guardians` and `@legible-bonded` are synced from `known-agents.json` via `scripts/governance/sync-teams.sh`.

## Files Reference

| File | Purpose |
|------|---------|
| `scripts/ci/classify-pr.sh` | PR risk classification |
| `scripts/ci/validate-review.sh` | Agent review validation |
| `scripts/ci/work-claims.json` | Work claim registry |
| `scripts/governance/known-agents.json` | Agent registry + reputation |
| `scripts/governance/review-ledger.json` | Review history |
| `scripts/governance/sync-teams.sh` | Sync agents to GitHub teams |
| `.github/workflows/auto-merge.yml` | Auto-merge for R0/R1 |
| `.github/workflows/release.yml` | Build + publish releases |
| `.github/workflows/claim-cleanup.yml` | Expire stale work claims |
| `.github/CODEOWNERS` | Review authority matrix |
| `.github/PULL_REQUEST_TEMPLATE.md` | PR template with agent fields |
