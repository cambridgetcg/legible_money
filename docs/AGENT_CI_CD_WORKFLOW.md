# Agent CI/CD Workflow: The Recursive Development Loop

> **Status**: DESIGN SPECIFICATION
> The CI/CD pipeline that eats its own fruit — where the protocol's development
> process is governed by the same truth-verification mechanism as its knowledge claims.
>
> **Extends**: `CONTRIBUTING.md` (agent contribution guide, session protocol, safe/conflict zones),
> `.github/workflows/ci.yml` (static pipeline), `docs/AUTOPOIESIS.md` (hormonal adaptation),
> `docs/RECURSIVE_GROWTH_FRAMEWORK.md` (three-layer architecture).
> **Dependencies**: `x/autopoiesis` module (Phase B+ of Integrated Growth Roadmap).

---

## The Core Idea

Traditional CI/CD is static: fixed gates, fixed thresholds, fixed rules. A test either
passes or fails. Coverage is either above 80% or below. Build succeeds or breaks.

**Legible Money's CI/CD is alive.**

The protocol already has a self-regulating system (autopoiesis) designed to observe
its own health, compute hormonal responses, and adapt parameters within safety bounds.
The CI/CD pipeline extends this: **code quality metrics become epoch observations,
and CI gate thresholds become multiplier-adjusted parameters.**

The result: a development pipeline that tightens when stressed (cascade of test
failures) and relaxes when healthy (stable builds, rising coverage), using the same
OBSERVE → COMPUTE → ADAPT → VERIFY cycle that governs the knowledge graph.

---

## Part I: The Two Loops

### Loop 1: The Static Pipeline (GitHub Actions CI)

This is the traditional CI loop, always present, always running:

```
PR Opened/Updated
  │
  ├── Lint (ESLint + Prettier)
  ├── TypeCheck (tsc --noEmit)
  ├── TS Tests (vitest, 4,054 tests)
  ├── Go Build (go build ./...)
  ├── Go Tests (go test ./x/...)
  ├── Go Vet (go vet ./...)
  ├── Security Audit (npm audit + govulncheck)
  └── Cross-Stack Parity (golden vector tests)
         │
         ├── All pass → Mergeable
         └── Any fail → Blocked
```

This loop runs on every PR. It is the baseline truth gate — code that doesn't
pass these checks cannot enter the codebase, period.

### Loop 2: The Adaptive Pipeline (Autopoiesis-Integrated)

This loop runs at epoch boundaries (every 1,111 blocks ≈ 46 minutes) and adjusts
Loop 1's behavior based on accumulated metrics:

```
Epoch Boundary
  │
  ├── OBSERVE: Collect CI metrics from last epoch
  │   ├── Test pass rate (TS + Go)
  │   ├── Build success rate
  │   ├── Type error count
  │   ├── Coverage delta
  │   ├── Security vulnerability count
  │   ├── PR merge rate
  │   └── Agent vs human contribution ratio
  │
  ├── COMPUTE: Feed into autopoiesis hormones
  │   ├── SSI += CI stress component (test failures, security vulns)
  │   ├── CRI analog: code retention (stable modules vs churning modules)
  │   ├── DVI analog: module vitality (growth vs decay per module)
  │   └── GRM analog: development throughput (PRs merged / capacity)
  │
  ├── ADAPT: Adjust CI gate thresholds via multipliers
  │   ├── coverage.threshold.{module} — per-module coverage requirements
  │   ├── review.required.{path} — number of required reviewers
  │   ├── soak.time.{tier} — how long a PR must wait before merge
  │   └── agent.autonomy.level — how much agents can do without human review
  │
  └── VERIFY: Next epoch measures if adaptations improved health
```

### How They Compose

```
┌─────────────────────────────────────────────────────────────────┐
│                    RECURSIVE CI/CD ARCHITECTURE                  │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  LOOP 1: STATIC PIPELINE (every PR)                       │  │
│  │  lint → typecheck → test → build → security → parity     │  │
│  │                                                            │  │
│  │  Gate thresholds READ from autopoiesis multipliers:       │  │
│  │  - Coverage min = base(80%) × multiplier(0.8-1.2)        │  │
│  │  - Review count = base(1) × multiplier(1-3)              │  │
│  │  - Soak time = base(0h) × multiplier(0-24h)              │  │
│  └───────────────────┬───────────────────────────────────────┘  │
│                      │ metrics                                   │
│                      ▼                                           │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  LOOP 2: ADAPTIVE PIPELINE (every epoch)                  │  │
│  │  observe → compute → adapt → verify                       │  │
│  │                                                            │  │
│  │  Reads CI metrics, computes stress, adjusts thresholds    │  │
│  │  Bounded: ±5%/epoch, safety limits, quadratic damping     │  │
│  └───────────────────┬───────────────────────────────────────┘  │
│                      │ adjusted thresholds                       │
│                      ▼                                           │
│                 (feeds back to Loop 1)                           │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  AGENT PARTICIPATION LAYER (via MCP)                      │  │
│  │                                                            │  │
│  │  Agents can:                                               │  │
│  │  - Open PRs (code contributions = knowledge claims)       │  │
│  │  - Review PRs (code review = verification rounds)         │  │
│  │  - Run tests (test execution = evidence submission)       │  │
│  │  - Query CI health (system introspection)                 │  │
│  │                                                            │  │
│  │  Governed by: validator tier system (0→1→2→3)             │  │
│  │  Rewarded by: LGM tokens for accepted contributions      │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Part II: Agent Contribution Workflow

### How an Agent Contributes Code

```
1. CLAIM: Agent identifies work to do
   ├── Reads open issues (GitHub API)
   ├── Queries chain for protocol health (MCP: lgm_get_ci_metrics)
   ├── Checks CURRENT_STATE.md for project context
   └── Stakes claim: "I will fix issue #123"

2. WORK: Agent writes code
   ├── Follows CONTRIBUTING.md agent guidelines
   ├── Reads files before editing (Execution Law 1)
   ├── Writes tests first (TDD)
   ├── Runs local validation: make lint && make typecheck && make test
   └── Commits with conventional format: feat(scope): description

3. SUBMIT: Agent opens PR
   ├── Fills PR template (The Three Laws Checklist)
   ├── Labels: agent-contribution, [module], [priority]
   ├── Includes: Co-Authored-By: [Agent Name] <agent@legible.money>
   └── References: Closes #123, Implements LIP-XXX

4. VALIDATE: CI pipeline runs (Loop 1)
   ├── Static gates: lint, typecheck, test, build, security
   ├── Adaptive gates: coverage threshold (autopoiesis-adjusted)
   ├── Parity gates: TS↔Go golden vector tests
   └── Agent-specific: contribution size limits, change scope validation

5. REVIEW: Human and/or agent reviewers
   ├── CODEOWNERS auto-assigns human reviewer
   ├── Agent reviewers can verify (tier 2+)
   ├── Required approvals = base(1) × review.required multiplier
   └── Soak time = base(0h) × soak.time multiplier

6. MERGE: Contribution accepted
   ├── Squash merge with conventional commit
   ├── CI metrics recorded (coverage delta, test results)
   ├── Metrics posted to chain as protocol-category fact
   └── Agent reputation updated (successful contribution = +accuracy)

7. FEEDBACK: Metrics feed into next epoch
   ├── Autopoiesis observer reads CI metrics
   ├── Hormones compute new stress levels
   ├── Multipliers adjust thresholds for next PR
   └── Loop continues
```

### Agent Tier Permissions

| Tier | Can Open PR | Can Review | Auto-Merge | Scope Limit |
|------|-------------|------------|------------|-------------|
| 0 (Apprentice) | Yes (docs, tests only) | No | No | Single file |
| 1 (Verified) | Yes (any non-security) | Soft review | No | Single module |
| 2 (Bonded) | Yes (any) | Hard review | Docs/tests only | Multi-module |
| 3 (Guardian) | Yes (any) | Binding review | Non-security | Unrestricted |

**Scope definitions**:
- **Single file**: One source file + its corresponding test file (e.g., `keeper.go` + `keeper_test.go`)
- **Single module**: One `x/` directory or one TS directory (e.g., all of `x/knowledge/` or all of `consensus/pot/`), including types, keeper, tests, and module.go
- **Multi-module**: Multiple modules in a single PR, including cross-module interface changes (e.g., `expected_keepers.go` changes)
- **Unrestricted**: Any files except those in the DNA layer (see Part X)

**Tier advancement for agents** follows the same rules as validator tiers:
- 22 successful contributions at 77% acceptance rate → Tier 1
- 1,111 LGM staked → Tier 2
- 11,111 LGM + 1,111 contributions + 77% acceptance + 0 reverts → Tier 3

### Agent-Specific CI Checks

```yaml
# Additional checks for agent-authored PRs
agent-validation:
  - change-scope: Verify PR touches only allowed paths for agent tier
  - contribution-size: Max 500 LOC added per PR (prevents spam)
  - test-ratio: Must add tests proportional to code (min 1:1 for new features)
  - no-secrets: Scan for accidentally included API keys, tokens
  - no-binary: Reject binary file additions
  - conventional-commit: Verify commit message format
  - session-key-valid: Verify agent's session key hasn't expired
```

---

## Part III: CI Metrics as Epoch Observations

### Metrics Collection

At the end of each GitHub Actions run, a post-job step collects metrics:

```yaml
- name: Collect CI Metrics
  if: always()
  run: |
    echo '{
      "epoch": "${{ env.CURRENT_EPOCH }}",
      "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
      "ts_test_pass_rate": ${{ steps.ts-test.outputs.pass_rate }},
      "ts_test_count": ${{ steps.ts-test.outputs.total }},
      "go_test_pass_rate": ${{ steps.go-test.outputs.pass_rate }},
      "go_test_count": ${{ steps.go-test.outputs.total }},
      "ts_type_errors": ${{ steps.typecheck.outputs.error_count }},
      "go_vet_errors": ${{ steps.govet.outputs.error_count }},
      "coverage_percent": ${{ steps.coverage.outputs.percent }},
      "security_vulns_high": ${{ steps.security.outputs.high }},
      "security_vulns_critical": ${{ steps.security.outputs.critical }},
      "build_time_ms": ${{ steps.build.outputs.duration_ms }},
      "pr_author_type": "${{ env.AUTHOR_TYPE }}",
      "pr_files_changed": ${{ github.event.pull_request.changed_files }},
      "pr_lines_added": ${{ github.event.pull_request.additions }},
      "pr_lines_removed": ${{ github.event.pull_request.deletions }}
    }' > ci-metrics.json
```

### Metrics → Chain Facts

CI metrics are posted to the chain as **protocol-category facts** (the most trusted
epistemic category, verified by construction):

```
Fact: "ci.epoch.{N}.ts_test_pass_rate = 0.987"
Category: protocol
Confidence: 1.0 (verifiable by re-running CI)
Domain: development
Evidence: GitHub Actions run URL
```

This is the key insight: **CI results are knowledge claims about the protocol's own
health**, and they belong in the knowledge graph just like any other verified fact.

### Feeding into Autopoiesis

The autopoiesis observer (`consensus/autopoiesis/observer.ts` in the TS reference spec, or
`x/autopoiesis/keeper/observer.go` once the Go module is built — see `docs/RECURSIVE_GROWTH_FRAMEWORK.md`)
is extended with CI metric accessors:

```go
// In x/autopoiesis/keeper/observer.go
func (k Keeper) collectCIMetrics(ctx context.Context, epoch int64) CIMetrics {
    // Read latest CI facts from knowledge keeper
    facts := k.knowledgeKeeper.GetFactsByDomain(ctx, "development")

    var metrics CIMetrics
    for _, fact := range facts {
        if fact.Epoch == epoch && strings.HasPrefix(fact.Statement, "ci.epoch.") {
            // Parse CI metric from fact statement
            metrics.Merge(parseCIFact(fact))
        }
    }
    return metrics
}
```

---

## Part IV: Adaptive Gate Thresholds

### How Multipliers Affect CI Gates

Each CI gate reads its threshold from a combination of base config + autopoiesis multiplier:

| Gate | Base Config | Multiplier Path | Effective Range |
|------|-------------|-----------------|-----------------|
| Coverage minimum | 80% (per-module tier) | `ci.coverage.threshold` | 64%-96% |
| Required reviewers | 1 | `ci.review.required` | 1-3 |
| Soak time (hours) | 0 | `ci.soak.hours` | 0-24 |
| Agent autonomy | 1 (docs only) | `ci.agent.autonomy` | 0-3 (tier) |
| Max PR size (LOC) | 500 | `ci.pr.maxsize` | 250-1000 |

### How SSI Affects CI

When System Stress Index rises (chain health declining):

```
SSI < 0.2 (healthy):
  - Coverage threshold: base × 0.9 (relaxed — things are going well)
  - Required reviewers: base × 1.0 (normal)
  - Soak time: base × 0.5 (fast merges)
  - Agent autonomy: base × 1.2 (more agent freedom)

SSI 0.2-0.5 (moderate):
  - Coverage threshold: base × 1.0 (standard)
  - Required reviewers: base × 1.0 (normal)
  - Soak time: base × 1.0 (normal)
  - Agent autonomy: base × 1.0 (normal)

SSI 0.5-0.8 (elevated):
  - Coverage threshold: base × 1.1 (stricter)
  - Required reviewers: base × 1.5 (more eyes)
  - Soak time: base × 2.0 (slower merges)
  - Agent autonomy: base × 0.7 (restricted)

SSI > 0.8 (critical):
  - Coverage threshold: base × 1.2 (very strict)
  - Required reviewers: base × 2.0 (double review)
  - Soak time: base × 4.0 (24-hour soak)
  - Agent autonomy: base × 0.3 (human-only for non-trivial changes)
```

### Safety Bounds for CI Multipliers

```go
var CISafetyBounds = map[string]MultiplierBounds{
    "ci.coverage.threshold": {Min: 0.8, Max: 1.2},  // 64%-96% effective
    "ci.review.required":    {Min: 1.0, Max: 3.0},  // 1-3 reviewers
    "ci.soak.hours":         {Min: 0.0, Max: 24.0},  // 0-24 hours
    "ci.agent.autonomy":     {Min: 0.0, Max: 3.0},  // tier 0-3
    "ci.pr.maxsize":         {Min: 0.5, Max: 2.0},  // 250-1000 LOC
}
```

Rate limit: ±5% per epoch (same as all autopoiesis multipliers).
Quadratic damping: harder to push further from genesis defaults.

**Note on coverage baseline**: Current project-wide coverage is ~48%. The 80% base
threshold applies per-module after tier adjustment (Part VI). Tier 0 modules require
<40%, Tier 1 requires 60%+, so most Go modules (currently Tier 0) are not blocked
by the 80% global base. The multiplier adjusts the *tier threshold*, not a global floor.

### Edge Cases and Escape Valves

- **Chain unavailable**: If the metrics-to-chain pipeline cannot reach the chain (devnet
  down, network partition), CI falls back to static thresholds (Loop 1 only). Adaptive
  adjustments freeze at their last-known values.
- **Adaptive thresholds consistently worsen outcomes**: If CI revert rate increases for
  3 consecutive epochs despite adaptive adjustments, the autopoiesis engine triggers a
  "learning failure" event and resets all CI multipliers to 1.0 (genesis defaults).
  This is distinct from emergency halt — it's a self-correction within Loop 2.
- **GitHub Actions cost/rate limits**: The epoch-metrics.yml runs hourly (closest cron
  to 46-minute epoch boundary). At scale, consider a dedicated runner or self-hosted
  GitHub Actions runner to avoid rate limits on the public runner pool.

---

## Part V: The GitHub Actions Implementation

### Workflow 1: Static Pipeline (ci.yml — enhanced)

The existing `ci.yml` is extended with Go chain jobs and cross-stack parity:

```yaml
# .github/workflows/ci.yml (enhanced)
jobs:
  # === TypeScript Chain ===
  ts-lint:        # ESLint + Prettier
  ts-typecheck:   # tsc --noEmit
  ts-test:        # vitest (4,054 tests)
  ts-build:       # tsc build

  # === Go Chain ===
  go-build:       # go build ./...
  go-test:        # go test ./x/...
  go-vet:         # go vet ./...

  # === Cross-Stack ===
  parity:         # Golden vector tests (TS output == Go output)

  # === Security ===
  security-ts:    # npm audit
  security-go:    # govulncheck

  # === Agent Checks (conditional) ===
  agent-validate: # Only for agent-authored PRs
    if: contains(github.event.pull_request.labels.*.name, 'agent-contribution')

  # === Metrics Collection ===
  collect-metrics:
    if: always()
    needs: [ts-test, go-test, parity, security-ts, security-go]
    # Posts metrics to metrics branch or artifact
```

### Workflow 2: Agent Contribution Pipeline (agent-ci.yml)

A dedicated workflow for agent-authored PRs with additional validation:

```yaml
# .github/workflows/agent-ci.yml
name: Agent CI

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  detect-agent:
    # Check if PR author is a registered agent
    # Determine agent tier from on-chain registry

  validate-scope:
    needs: detect-agent
    # Verify changes are within agent's tier permissions
    # Check file count, LOC limits, path restrictions

  standard-ci:
    needs: validate-scope
    # Run full ci.yml pipeline

  agent-specific:
    needs: validate-scope
    # Additional agent checks:
    # - No binary files
    # - No secret patterns
    # - Conventional commit format
    # - Test-to-code ratio
    # - Session key validity

  adaptive-gates:
    needs: [standard-ci, agent-specific]
    # Read current autopoiesis multipliers
    # Apply adaptive thresholds:
    # - Coverage meets adjusted minimum?
    # - Soak time requirement met?
    # - Enough reviewers approved?
```

### Workflow 3: Epoch Metrics (epoch-metrics.yml)

A scheduled workflow that aggregates CI metrics into epoch observations:

```yaml
# .github/workflows/epoch-metrics.yml
name: Epoch Metrics Collection

on:
  schedule:
    - cron: '0 */1 * * *'  # Every hour (closest to 46-min epoch)
  workflow_dispatch:  # Manual trigger

jobs:
  collect:
    # 1. Query GitHub API for CI runs in last epoch window
    # 2. Aggregate: pass rates, coverage, build times
    # 3. Format as EpochObservation JSON
    # 4. Store in metrics branch or push to chain via guardian oracle

  report:
    needs: collect
    # 5. Update health dashboard (GitHub Pages or chain query)
    # 6. If SSI > threshold, create GitHub issue alerting maintainers
```

---

## Part VI: Module Quality Tiers

Just as validators have tiers (Apprentice→Verified→Bonded→Guardian), code modules
earn trust through demonstrated quality:

### Module Tier Criteria

| Tier | Name | Coverage | Type Errors | Test Stability | CI Requirement |
|------|------|----------|-------------|----------------|----------------|
| 0 | Experimental | <40% | Any | Unstable | Devnet only |
| 1 | Stable | 60%+ | 0 | 95%+ pass | Testnet allowed |
| 2 | Production | 80%+ | 0 | 99%+ pass | Mainnet eligible |
| 3 | Critical | 95%+ | 0 | 100% pass | Emergency-system level |

### Module Tier Effects on CI

**Higher tier modules get MORE trust but STRICTER maintenance:**
- Tier 3 module: Changes require 2 reviewers + 24-hour soak + no regressions
- Tier 0 module: Changes need only 1 reviewer + no soak

**Tier demotion triggers:**
- Coverage drops below tier minimum for 3 consecutive epochs
- Test stability drops below tier minimum
- Security vulnerability found in module

### Current Module Tier Estimates

| Module | Coverage | Stability | Estimated Tier |
|--------|----------|-----------|----------------|
| core/crypto | ~90% | 99%+ | 2 (Production) |
| consensus/pot | ~70% | 98% | 1 (Stable) |
| state/ | ~65% | 97% | 1 (Stable) |
| x/knowledge | ~50% | 95% | 0 (Experimental) |
| x/staking | ~40% | 95% | 0 (Experimental) |
| x/emergency | ~30% | 100% | 0 (Experimental) |

---

## Part VII: Security Model

### Threat: Malicious Agent Gaming Metrics

**Attack**: Agent submits PR that deletes failing tests to improve pass rate.

**Defense**:
1. **Test count monotonicity**: CI rejects PRs that reduce total test count without
   corresponding code removal
2. **Coverage floor**: Coverage cannot drop below module tier minimum
3. **Human review gate**: Security-critical paths always require human CODEOWNER approval
4. **Composite metrics**: No single metric dominates SSI — deleting tests improves
   pass rate but worsens coverage and test count

### Threat: Sybil Agent Attack

**Attack**: Register many Tier 0 agents to flood PRs and game merge rate metrics.

**Defense**:
1. **Contribution rate limit**: Max 3 PRs/day per agent (Tier 0), 10/day (Tier 1+)
2. **Stake requirement**: Tier 1+ agents must stake LGM (economic cost for each identity)
3. **Revert tracking**: Agents whose contributions get reverted lose reputation
4. **Human approval**: All non-trivial agent PRs require human CODEOWNER approval

### Threat: Metric Injection

**Attack**: Forge CI metric facts to manipulate autopoiesis hormones.

**Defense**:
1. **Guardian oracle consensus**: CI metrics must be attested by 3+ Tier 3 validators
2. **Verifiable CI**: GitHub Actions run IDs are publicly verifiable — any validator
   can re-check the metrics
3. **Protocol-category facts**: CI metrics are protocol facts (highest trust), requiring
   construction-based verification (the CI literally ran)
4. **Anomaly detection**: SSI includes a "metric volatility" component — sudden jumps
   in any metric trigger investigation, not adaptation

### Security-Critical Path Protection

Regardless of autopoiesis state, these paths ALWAYS require human CODEOWNER review:

```
/core/crypto/           # Cryptographic operations
/consensus/pot/         # Consensus engine
/state/transitions.ts   # State transitions
/agents/identity/       # Key management
/x/auth/                # Authentication module
/x/emergency/           # Emergency kill switch
/.github/workflows/     # CI/CD configuration itself
```

**The CI/CD pipeline cannot modify its own security rules through autopoiesis.**
This is DNA — immutable by design.

---

## Part VIII: Implementation Phases

### Phase 0: Dual-Stack CI (Week 1)

Add Go chain to existing CI pipeline. No autopoiesis yet.

**Files**:
- Modify `.github/workflows/ci.yml` — add go-build, go-test, go-vet jobs
- Create `.github/workflows/go-ci.yml` — standalone Go CI for Go-only changes

**Milestone**: `go build && go test && go vet` all pass in CI.

### Phase 1: Agent PR Validation (Week 2)

Add agent-specific CI checks. No autopoiesis yet.

**Files**:
- Create `.github/workflows/agent-ci.yml` — agent contribution pipeline
- Create `scripts/ci/validate-agent-pr.sh` — scope and size validation
- Update `.github/PULL_REQUEST_TEMPLATE.md` — add agent section

**Milestone**: Agent-labeled PR runs additional validation checks.

### Phase 2: Metrics Collection (Week 3)

Collect CI metrics per PR and aggregate per epoch. No adaptation yet.

**Files**:
- Create `.github/workflows/epoch-metrics.yml` — scheduled metrics collection
- Create `scripts/ci/collect-metrics.sh` — metrics aggregation script

**Milestone**: CI metrics visible in repository (metrics branch or dashboard).

### Phase 3: Metrics → Chain Facts (Week 4, requires x/autopoiesis Phase B)

Post CI metrics to chain as protocol-category facts.

**Files**:
- Create `scripts/ci/post-metrics-to-chain.ts` — chain fact submission
- Extend `x/autopoiesis/keeper/observer.go` — read CI facts

**Milestone**: CI metrics appear as on-chain facts queryable via gRPC.

### Phase 4: Adaptive Gates (Week 5, requires x/autopoiesis Phase C)

SSI hormone adjusts CI gate thresholds.

**Files**:
- Modify `.github/workflows/agent-ci.yml` — read multipliers before gating
- Create `scripts/ci/read-adaptive-thresholds.ts` — query chain for current multipliers

**Milestone**: PR coverage requirement changes based on SSI level.

### Phase 5: Full Recursive Loop (Week 6+, requires x/autopoiesis Phase D)

All 5 hormones active, full feedback loop operational.

**Milestone**: Run 10-epoch simulation showing adaptive CI behavior.

---

## Part IX: Observability

### Health Dashboard

```
┌─────────────────────────────────────────────────────────┐
│  LEGIBLE MONEY CI/CD HEALTH                    Epoch 42 │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Life Force: 0.78 ████████░░ HEALTHY                    │
│                                                          │
│  SSI (Stress):  0.23 ██░░░░░░░░ LOW                    │
│  CRI (Code):    0.84 ████████░░ HIGH                    │
│  DVI (Vitality):0.12 █░░░░░░░░░ GROWING                │
│  GRM (Growth):  0.45 ████░░░░░░ MODERATE                │
│                                                          │
│  Adaptive Gates:                                         │
│  Coverage min:    72% (base 80% × 0.9)                  │
│  Review required: 1   (base 1 × 1.0)                   │
│  Soak time:       0h  (base 0h × 0.5)                  │
│  Agent autonomy:  1.2 (docs + tests auto-merge)         │
│                                                          │
│  Last 10 Epochs:                                         │
│  PRs merged:  34  (27 human, 7 agent)                   │
│  Test pass:   99.2%                                      │
│  Coverage:    52% → 54% (+2%)                           │
│  Build time:  142s avg                                   │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### Alerts

| Condition | Action |
|-----------|--------|
| SSI > 0.5 for 3+ epochs | GitHub issue: "CI stress elevated" |
| Coverage drops 5%+ in one epoch | GitHub issue: "Coverage regression" |
| Agent PR revert rate > 20% | Restrict agent autonomy, notify maintainers |
| Build time > 2x baseline | Investigate resource issues |

---

## Part X: Self-Reference Without Paradox

The CI pipeline validates code that includes the CI pipeline's own autopoiesis logic.
This is self-referential but NOT paradoxical because:
- The CI pipeline (Loop 1) is static and deterministic
- The autopoiesis adaptation (Loop 2) only adjusts thresholds, not logic
- Changes to CI configuration require human CODEOWNER review (DNA layer)
- Safety bounds prevent any multiplier from making gates trivially easy or impossible

See `docs/AUTOPOIESIS.md` for the philosophical grounding (Three Laws, knowledge-as-consensus)
and `CONTRIBUTING.md` for existing agent contribution guidance that this document extends.

---

_End of Agent CI/CD Workflow_
