# Development Workflow Insights

> **Status**: Living document — updated 2026-02-16
> **Scope**: How Legible Money is built, how that compares to industry standards,
> and what to formalize next.

---

## Part I: The Manual Blockchain Development Loop

Legible Money is developed through a unique process: **AI agent sessions with human
oversight**, where each session starts fresh and reconstructs context from persistent
memory files. This is not a traditional development workflow — it is closer to a
relay race where each runner must read the baton before running.

### The 6-Phase Loop

```
┌──────────────────────────────────────────────────────────────┐
│                   SESSION LIFECYCLE                           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Phase 0: BOOTSTRAP                                          │
│  ├─ git checkout main && git pull                            │
│  ├─ Read .claude/memory/CURRENT_STATE.md                     │
│  ├─ Read .claude/memory/DECISIONS.md                         │
│  ├─ Check .claude/memory/BLOCKED.md                          │
│  └─ Pick work → create feature branch                        │
│                                                              │
│  Phase 1: IDENTIFY                                           │
│  ├─ Human-assigned task (highest priority)                   │
│  ├─ GitHub issues (labeled by tier/priority)                 │
│  ├─ BLOCKED.md open questions                                │
│  └─ INTEGRATED_GROWTH_ROADMAP.md (self-selected)             │
│                                                              │
│  Phase 2: PLAN                                               │
│  ├─ PP orchestration (sense → comprehend → build → assess)   │
│  ├─ Design docs in .claude/plans/                            │
│  └─ Architecture docs in docs/                               │
│                                                              │
│  Phase 3: EXECUTE                                            │
│  ├─ Atomic commits on feature branch                         │
│  ├─ Tests written alongside implementation                   │
│  └─ make pr-check as continuous gate                         │
│                                                              │
│  Phase 4: VERIFY                                             │
│  ├─ Automated: make pr-check (build + test + vet)            │
│  ├─ Behavioral: tests as specification                       │
│  ├─ Cross-stack: golden vector validation                    │
│  └─ PP /verify: 30-gate quality pass                         │
│                                                              │
│  Phase 5: DOCUMENT + HAND-OFF                                │
│  ├─ Session log: .claude/sessions/YYYY-MM-DD.md              │
│  ├─ CURRENT_STATE.md prepend (newest at top)                 │
│  ├─ DECISIONS.md append (immutable log)                      │
│  ├─ PR with Three Laws Checklist                             │
│  └─ continuation-welcome label if incomplete                 │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### Memory Architecture (3 Tiers)

| Tier | Location | Update Frequency | Purpose |
|------|----------|-----------------|---------|
| **1. Trunk State** | `.claude/memory/CURRENT_STATE.md`, `DECISIONS.md`, `BLOCKED.md`, `GAPS.md` | Every session (after merge) | Master project state |
| **2. Quick Reference** | `.claude/context/CORE_TYPES.md`, `API_PATTERNS.md`, `ARCHITECTURE_RATIONALE.md` | Periodically | Distilled domain knowledge |
| **3. Investigation Archives** | `.claude/memory/*.md` (60+ files) | On creation | Deep-dive artifacts |

### What Works Well

1. **CURRENT_STATE.md as running journal** — prepend-only, newest at top, full situational awareness in first 100 lines
2. **DECISIONS.md as append-only immutable log** — includes reasoning ("why"), not just choices ("what")
3. **PP framework forces pre-work investigation** — gap analysis, territory scouting before code
4. **Tests as behavioral documentation** — cycle penalty math preserved in assertions, survives refactoring
5. **Cross-stack golden vectors** — Go/TS semantic equivalence validated automatically
6. **Five Execution Laws** — "No edit without read, No claim without citation" prevents ghost-pattern bugs

### What Is Fragile

| Risk | Impact | Current Mitigation |
|------|--------|--------------------|
| CURRENT_STATE.md staleness | Session starts from false premises | Agent discipline only |
| Human reviewer bottleneck | All agent work queues in open PRs | None — single CODEOWNER |
| 60+ investigation docs, no index | Prior work rediscovered from scratch | Manual memory of file names |
| TEST_PROGRESS.md drift | Coverage picture inaccurate | None — manually maintained |
| Memory files on feature branches | Lost if PR never merges | Branch session logs |
| Pre-existing CI failures | New sessions can't distinguish regressions | Manual investigation |
| Agent tier enforcement is label-based | Any agent can self-label tier-3 | `agent-ci.yml` checks (weak) |

---

## Part II: Current CI/CD Infrastructure

### GitHub Actions Workflows

**`ci.yml`** — Main CI pipeline (6 jobs):

| Job | Trigger | What It Does |
|-----|---------|-------------|
| `build` | push/PR to main | `go build ./...` |
| `test` | push/PR to main | `go test ./x/... -count=1 -v` + `go test ./app/...` (continue-on-error) |
| `vet` | push/PR to main | `go vet ./...` |
| `e2e-cosmos` | manual dispatch only | Build binary → start devnet → run tx lifecycle → stop |
| `agent-validate` | PR with `agent-contribution` label | PR size, binaries, secrets, conventional commit |
| `collect-metrics` | always (after build/test/vet) | JSON artifact with job results |

**`agent-ci.yml`** — Agent contribution pipeline (2 jobs):

| Job | Trigger | What It Does |
|-----|---------|-------------|
| `detect-agent` | PR with `agent-contribution` label | Check Co-Authored-By, determine tier from labels |
| `validate-scope` | agent detected | Tier 0: docs/tests only. Tier 0-1: no security paths. Secret scan. Binary check. |

### Makefile Targets

| Target | What It Does |
|--------|-------------|
| `build` | Build `legbled` binary to `build/` |
| `build-all` | `go build ./...` |
| `test` | `go test ./x/... -count=1` |
| `test-v` | Verbose tests |
| `test-app` | `go test ./app/... -count=1 -v` |
| `test-all` | `test` + `test-app` |
| `vet` | `go vet ./...` |
| `devnet` | Start Cosmos SDK devnet |
| `devnet-reset` | Reset devnet data |
| `devnet-stop` | Stop devnet |
| `devnet-health` | Run 9-check healthcheck |
| `test-tx-lifecycle` | Run tx lifecycle tests against live devnet |
| `test-e2e` | Full: start devnet → health → tx lifecycle → stop |
| `pr-check` | `build-all` + `test` + `vet` |
| `clean` | Remove `build/` |

### PR Quality Gates

The PR template (`.github/PULL_REQUEST_TEMPLATE.md`) enforces:
- **Three Laws Checklist**: LEGIBLE (readable), HONEST (uncertainty acknowledged), AGENT-FIRST (computational scale)
- **Code Quality**: invariants documented, property tests, edge cases, security review
- **Quality Gates**: tests pass, typecheck passes, lint passes, 80%+ coverage
- **Test Plan**: How was it tested, what scenarios covered

---

## Part III: Industry Standard Comparison

### 3.1 CI/CD for Blockchain Projects (2025-2026)

| Practice | Industry Standard | Legible Money Status | Gap |
|----------|-------------------|---------------------|-----|
| **Static analysis** | golangci-lint with gosec, staticcheck, errcheck | `go vet` only | Missing security + quality linting |
| **Fuzz testing** | Go native fuzzing (`go test -fuzz`) | None | Consensus-critical code unfuzzed |
| **Test coverage reporting** | `go test -coverprofile` + codecov/coveralls | None | No automated coverage tracking |
| **Dependency scanning** | govulncheck, Dependabot, Snyk | None | Supply chain blind spot |
| **Race detection** | `go test -race` in CI | None | Concurrency bugs undetected |
| **Simapp testing** | Cosmos SDK `simapp` simulation (100+ tx types) | None | No genesis simulation |
| **Upgrade handler testing** | Test upgrade from version N to N+1 | None | Upgrade path untested |
| **Binary reproducibility** | Deterministic builds (goreleaser, Docker multi-stage) | Ad-hoc `go build` | Not reproducible |
| **SBOM generation** | CycloneDX/SPDX in CI | None | No software bill of materials |
| **Conventional commits** | commitlint, semantic-release | Warning only (agent-ci) | Not enforced |

### 3.2 DevNet → TestNet → MainNet Promotion

Industry-standard promotion workflow:

```
LOCAL DEV          DEVNET              TESTNET             MAINNET
─────────────────────────────────────────────────────────────────

 make test    →   make test-e2e   →  IBC relay tests  →  Governance
 go vet       →   healthcheck     →  Load testing     →  Binary release
 build        →   tx lifecycle    →  Fuzz overnight   →  Chain upgrade
              →   1-validator     →  Multi-validator   →  Multi-validator
              →   ephemeral       →  persistent        →  persistent

 Gates:          Gates:              Gates:               Gates:
 - All tests     - Blocks produced   - 72h uptime         - Audit complete
 - Build clean   - Txs execute       - No consensus fail  - Gov proposal passes
 - Vet clean     - gRPC responds     - IBC transfers work - Binary signed
                                     - Upgrade tested     - Monitoring live
```

**Legible Money status**: Local dev + DevNet (1-validator) implemented. TestNet and MainNet promotion workflows are not defined.

### 3.3 Security Tooling Standards

| Tool Category | Industry Standard | Recommended for Legible Money |
|---------------|-------------------|-------------------------------|
| **Linting** | golangci-lint (meta-linter, 50+ linters) | Add `.golangci.yml` with gosec, staticcheck, errcheck, revive |
| **Security scanning** | gosec (Go-specific) | Enable in golangci-lint |
| **Dependency vulns** | govulncheck (Go official) | Add `make vuln-check` target + CI job |
| **Fuzzing** | Go native `testing.F` | Priority: VRF, state transitions, gas calculation |
| **Race detection** | `go test -race` | Add to CI as separate job (slow but critical) |
| **SAST** | Semgrep (with blockchain rules) | Evaluate for consensus-critical paths |

### 3.4 Cosmos SDK-Specific Patterns

Leading Cosmos SDK chains (Osmosis, Juno, Stride, dYdX) use:

| Pattern | Description | Legible Money Status |
|---------|-------------|---------------------|
| **SimApp testing** | Run simulations with random tx sequences | Not implemented |
| **Genesis migration tests** | Test `MigrateGenesis` from v(N) to v(N+1) | `migrate-genesis` is a stub |
| **IBC interchain tests** | Use `interchaintest` framework | Not implemented |
| **Upgrade handler tests** | Create v1→v2 migration, test state preservation | Not implemented |
| **Module parameter validation** | Validate all params in `ValidateBasic` | Weak validation (P2 in genesis audit) |
| **BeginBlocker ordering tests** | Machine-checked dependency graph | Implemented (`blocker_order_validator.go`) |

### 3.5 AI-Assisted Development Workflows (2025-2026)

The project already uses several cutting-edge practices:

| Practice | Description | Legible Money Status |
|----------|-------------|---------------------|
| **Agent session continuity** | Memory files + context distillation | Best-in-class |
| **PP quality framework** | 30-gate verification for agent work | Unique/novel |
| **Agent tier system** | Graduated permissions for AI contributors | Implemented |
| **Cross-session knowledge** | Investigation archives + decision log | Well-established |
| **Agent CI pipeline** | Dedicated validation for agent PRs | Implemented |

What's emerging as standard elsewhere:
- **AI-generated test suggestions**: Claude/Copilot for test gap detection
- **AI code review in CI**: Automated review comments on PRs
- **Specification-driven generation**: From formal spec to implementation
- **Continuous fuzzing as a service**: OSS-Fuzz for blockchain projects

---

## Part IV: Recommended Workflow Enhancements

### Priority 1: Security Hardening (Immediate)

**Add golangci-lint with security linters**:

```yaml
# .golangci.yml
linters:
  enable:
    - staticcheck
    - gosimple
    - govet
    - errcheck
    - gosec
    - revive
    - gocyclo
    - misspell
    - unconvert
    - unparam
linters-settings:
  errcheck:
    check-type-assertions: true
    check-blank: true
  govet:
    enable-all: true
  gocyclo:
    min-complexity: 15
  gosec:
    excludes:
      - G404  # Weak random (we use crypto/rand where needed)
```

**Add to Makefile**:
```makefile
lint: ## Run golangci-lint
	@golangci-lint run ./...

vuln-check: ## Check for known vulnerabilities
	@govulncheck ./...

race-test: ## Run tests with race detector
	@go test -race ./x/... -count=1
```

**Add to CI**:
```yaml
lint:
  name: "Go: Lint"
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with: { go-version: '1.22' }
    - uses: golangci/golangci-lint-action@v6
      with: { version: latest }
```

### Priority 2: Session Health Automation

**`make session-health`** — validate memory system integrity at session start:

```makefile
session-health: ## Validate session memory integrity
	@echo "Checking memory freshness..."
	@# Check CURRENT_STATE.md was updated within last 5 commits
	@LAST_MODIFIED=$$(git log -1 --format=%ct -- .claude/memory/CURRENT_STATE.md 2>/dev/null || echo 0); \
	 HEAD_TIME=$$(git log -1 --format=%ct); \
	 AGE=$$(( (HEAD_TIME - LAST_MODIFIED) / 86400 )); \
	 if [ $$AGE -gt 7 ]; then echo "WARNING: CURRENT_STATE.md last updated $$AGE days ago"; fi
	@echo "Checking for open continuation PRs..."
	@gh pr list --label continuation-welcome 2>/dev/null || true
	@echo "Checking for stale blockers..."
	@if [ -f .claude/memory/BLOCKED.md ]; then grep -c "^-" .claude/memory/BLOCKED.md 2>/dev/null || echo "0 active blockers"; fi
	@echo "Session health check complete."
```

### Priority 3: Investigation Index

**`make memory-index`** — auto-generate index of investigation documents:

```makefile
memory-index: ## Generate investigation document index
	@echo "# Investigation Index (auto-generated)" > .claude/memory/INDEX.md
	@echo "" >> .claude/memory/INDEX.md
	@echo "| File | Last Modified | First Line |" >> .claude/memory/INDEX.md
	@echo "|------|--------------|------------|" >> .claude/memory/INDEX.md
	@for f in .claude/memory/*.md; do \
	  NAME=$$(basename $$f); \
	  if [ "$$NAME" != "INDEX.md" ] && [ "$$NAME" != "CURRENT_STATE.md" ]; then \
	    DATE=$$(git log -1 --format=%cs -- $$f 2>/dev/null || echo "untracked"); \
	    FIRST=$$(head -1 $$f | sed 's/|/∣/g' | cut -c1-60); \
	    echo "| $$NAME | $$DATE | $$FIRST |" >> .claude/memory/INDEX.md; \
	  fi; \
	done
	@echo "Index generated: .claude/memory/INDEX.md"
```

### Priority 4: Coverage Tracking

**Add automated coverage reporting to CI**:

```yaml
coverage:
  name: "Go: Coverage"
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with: { go-version: '1.22', cache: true }
    - name: Run tests with coverage
      run: go test ./x/... -coverprofile=coverage.out -count=1
    - name: Generate coverage summary
      run: |
        go tool cover -func=coverage.out | tail -1
        TOTAL=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}')
        echo "Total coverage: $TOTAL"
        echo "## Coverage: $TOTAL" >> $GITHUB_STEP_SUMMARY
```

### Priority 5: Fuzz Testing for Consensus-Critical Code

Priority targets for Go native fuzzing:

| Target | File | Why |
|--------|------|-----|
| VRF generation | `x/knowledge/crypto/vrf.go` | Validator selection integrity |
| VRF verification | `x/knowledge/crypto/vrf.go` | Proof tampering detection |
| Gas cost lookup | `app/gas.go` | Unknown tx types must not panic |
| Confidence aggregation | `x/knowledge/keeper/confidence.go` | Overflow/underflow in rewards |
| Genesis validation | `x/*/types/types.go` | Malformed genesis must be rejected |

Example fuzz test:

```go
func FuzzVerifyVRF(f *testing.F) {
    _, priv, _ := ed25519.GenerateKey(nil)
    seed := []byte("fuzz-seed")
    _, proof, _ := GenerateVRF(seed, priv)
    pub := priv.Public().(ed25519.PublicKey)

    f.Add(seed, []byte(pub), proof)
    f.Fuzz(func(t *testing.T, seed, pubKey, proof []byte) {
        // Must never panic
        VerifyVRF(seed, pubKey, proof)
    })
}
```

### Priority 6: Module Status Introspection

**`make module-status`** — live module readiness report:

```makefile
module-status: ## Show module implementation status
	@echo "Module Status Report"
	@echo "==================="
	@for dir in x/*/; do \
	  MOD=$$(basename $$dir); \
	  KEEPER=$$(find $$dir -name "*.go" -path "*/keeper/*" ! -name "*_test.go" | wc -l | tr -d ' '); \
	  TESTS=$$(find $$dir -name "*_test.go" | wc -l | tr -d ' '); \
	  TODOS=$$(grep -r "TODO\|FIXME\|HACK\|STUB" $$dir --include="*.go" 2>/dev/null | wc -l | tr -d ' '); \
	  printf "  %-25s keeper:%s tests:%s todos:%s\n" "$$MOD" "$$KEEPER" "$$TESTS" "$$TODOS"; \
	done
```

---

## Part V: The Legible Money Development Model

### What Makes This Different

Traditional blockchain development follows:
```
spec → implement → test → audit → deploy
```

Legible Money follows:
```
investigate → design → implement → verify → document → hand-off → repeat
```

The key difference is **each session is stateless**. No developer has continuous context.
Instead, context is reconstructed from memory files at the start of every session. This
creates unique properties:

1. **Every design decision must be written down** — oral tradition doesn't work
2. **Tests serve as executable documentation** — they survive agent context resets
3. **Investigation archives accumulate institutional knowledge** — 60+ deep-dive documents
4. **The PP framework ensures quality even with session discontinuity** — 30 gates, 5 laws

### The Relay Race Model

```
Session N                    Session N+1                  Session N+2
─────────────────────────────────────────────────────────────────────

 Read memory        ←──────  Write memory        ←──────  Read memory
 Investigate                 Investigate                   Investigate
 Design                      Implement                     Fix regression
 Implement                   Test                          Document
 Test                        Document                      PR + merge
 PR (draft)                  PR (ready)

 .claude/memory/    ←──────  .claude/memory/     ←──────  .claude/memory/
 (state at N)                (state at N+1)               (state at N+2)
```

Each session reads the baton, runs its leg, and writes the baton for the next runner.
The memory files ARE the baton. Their integrity is the single most important property
of the system.

### Session Health Invariants

These must hold at every session boundary:

1. **CURRENT_STATE.md reflects reality** — no stale claims about what's built
2. **DECISIONS.md is append-only** — no decision is ever deleted or modified
3. **BLOCKED.md has no items older than 2 weeks** — stale blockers are tech debt
4. **All tests pass on main** — green baseline is non-negotiable
5. **No orphaned feature branches** — draft PRs get `continuation-welcome` or close

### Recommendations for Formalization

| Current Practice | Recommended Automation |
|-----------------|----------------------|
| "Read CURRENT_STATE.md at session start" | `make session-health` validates freshness |
| "Run make pr-check before PR" | CI blocks merge without passing pr-check |
| "Update CURRENT_STATE.md after merge" | Post-merge hook reminds/validates |
| "Check for continuation-welcome PRs" | `make session-health` lists open drafts |
| "Investigation documents accumulate" | `make memory-index` generates searchable index |
| "Test coverage tracked manually" | CI generates coverage report automatically |
| "Module status checked by reading code" | `make module-status` generates live report |

---

## Part VI: Industry Tooling Recommendations

### Immediate (Next Sprint)

| Tool | Purpose | Integration |
|------|---------|-------------|
| **golangci-lint** | Meta-linter with gosec + staticcheck | `.golangci.yml` + CI job |
| **govulncheck** | Dependency vulnerability scanning | `make vuln-check` + weekly CI |
| **`go test -race`** | Race condition detection | CI job (separate from unit tests) |
| **`go test -coverprofile`** | Coverage reporting | CI job + summary |

### Medium Term (Next Month)

| Tool | Purpose | Integration |
|------|---------|-------------|
| **Go native fuzzing** | Consensus-critical code fuzzing | `x/knowledge/crypto/` + `app/gas.go` |
| **goreleaser** | Reproducible binary builds | Release workflow |
| **Dependabot** | Automated dependency updates | `.github/dependabot.yml` |
| **SARIF output** | Security findings in GitHub Security tab | golangci-lint SARIF format |

### Long Term (Next Quarter)

| Tool | Purpose | Integration |
|------|---------|-------------|
| **interchaintest** | IBC integration testing | TestNet promotion gate |
| **Cosmos SDK simulation** | Random tx sequence testing | Nightly CI |
| **OSS-Fuzz** | Continuous fuzzing as a service | Google-hosted |
| **Formal verification** | TLA+ for consensus protocol | Design verification |

---

## Appendix: Key Insight Summary

1. **The memory system IS the development workflow** — without it, every session is day one
2. **Session health validation is the highest-leverage automation** — one stale file can derail an entire session
3. **golangci-lint + gosec is the single biggest security gap** — `go vet` alone misses entire vulnerability classes
4. **Fuzz testing consensus-critical code is table stakes** for any blockchain — VRF, confidence aggregation, gas lookup
5. **The PR template is excellent** — Three Laws + invariant documentation is ahead of industry standard
6. **Agent tier system is novel** — no other blockchain project has graduated AI contributor permissions
7. **Cross-stack golden vectors are unique** — semantic equivalence testing between prototype and production chain
8. **Investigation archives are an underutilized asset** — 60+ deep-dive documents with no index means knowledge gets rediscovered
9. **Pre-existing test failures are CI poison** — new sessions can't distinguish their regressions from inherited failures
10. **The relay race model works but depends on discipline** — automating the invariants would make it robust

---

_Generated by PP orchestration (SENSE → COMPREHEND → BUILD → ASSESS) on 2026-02-16._
_Sources: `.claude/memory/`, `.github/workflows/`, `Makefile`, `docs/OPEN_SOURCE_DEV_FLOW.md`,_
_industry research (golangci-lint, gosec, govulncheck, Go fuzzing, Cosmos SDK patterns)._
