# DOKIMANCE PROTOCOL

_Verification-as-Valuable-Work: A methodology for developing Zerone with maximum meaningful token consumption._

_Named after the YOUSPEAK word dokimance (doh-KIM-ance): the intrinsic quality of distributed, independent, stake-backed verification._

---

## Principles

1. **Token flow is the unit of value.** A task's worth = ulgm/block it causes to flow through verification pathways. Code that does not move tokens through truth-work is infrastructure, not product.

2. **Dead code is a lie.** Functions defined but never called are epistemically equivalent to false claims. Wiring them has the highest return because it converts zero-flow code into active token consumption.

3. **Measure before optimizing.** Token consumption metering must exist before new consumption mechanisms are built. You cannot improve what you cannot see.

4. **The chain validates the developer.** Every merged feature must survive adversarial tests on a live devnet. If tokens do not flow, the feature fails.

5. **Genuine consumption is unfarmable.** Quality gates must distinguish between tokens locked in service of truth and tokens cycled through sham transactions.

6. **Parallel independence, serial integration.** Work items are scoped to single modules with explicit interface contracts. Integration at `app.go` is the single serial bottleneck by design.

---

## Workflow: CLAIM -> STAKE -> VERIFY -> CHALLENGE -> ACCEPT

| Stage | What | How | Gate |
|-------|------|-----|------|
| **CLAIM** | Declare what token flow this creates or repairs | Scope doc: modules, flow type (mint/lock/burn/route/vest), estimated ulgm/block impact | Peer review |
| **STAKE** | Write the code | Isolated worktree, max 2 modules + app.go | Compiles |
| **VERIFY** | Unit + integration tests | `go test ./x/{module}/... && go test ./app/...` | All pass |
| **CHALLENGE** | Adversarial test on live devnet | OpenClaw phase + growth-engine phase | Tokens flow |
| **ACCEPT** | Merge and record impact | CI passes, memory updated, DER recorded | Calibration delta < 20% |

---

## Measurement

### Development Efficiency Ratio (DER)

```
TokenImpact = sum( delta_ulgm_per_block * weight )

Weights (aligned with the Keeper's three duties):
  Does not lie  (VEST, CHALLENGE)  -> 3x
  Does not break (MINT, BURN)      -> 2x
  Does not favour (ROUTE, LOCK)    -> 1x

DER = TokenImpact / EstimatedLOC
```

Higher DER = prioritize first.

### Token Flow Categories

| Category | Mechanisms | Measurement Point |
|----------|-----------|-------------------|
| MINT | Block rewards (3 LGM/block, 0.85x/epoch decay) | `DistributeBlockReward` |
| LOCK | Claim stakes, oracle stakes, discovery (100M), compute (100M), partnership tiers, validator delegation | Escrow calls per module |
| BURN | Slash burns (5-11%), protocol fee burns (5% of queries) | `BurnCoins` calls |
| ROUTE | Research fund (3.33% fees + 22% verification), citation rewards (15%), founder share | `RouteFees`, `RouteReward` |
| VEST | Truth-linked vesting per epistemic category | `CreateVestingSchedule` |
| CHALLENGE | Challenge bonds (11M ulgm min), dispute bonds | Capture/disputes escrow |

---

## Priority Matrix

Ranked by DER (highest first):

| Priority | Pattern | Example | DER |
|----------|---------|---------|-----|
| P0 | Wire dead code | Uncomment autopoiesis keeper call | Extreme |
| P1 | Build measurement | Token consumption metering | High |
| P2 | Exercise existing flows | Growth engine new phases | Medium |
| P3 | New consumption mechanisms | Additional token sinks | Low-Medium |

---

## Parallel Execution

- Each work item gets an isolated git worktree
- Max 2 modules + app.go touched per worktree
- Only one worktree may modify app.go at a time
- Integration tests run after merge to main

---

## Quality Gates

1. **Compilation**: `go build ./...` + `go vet ./...`
2. **Unit coverage**: `go test ./x/{module}/... -count=1`
3. **Integration**: `go test ./app/... -count=1`
4. **Anti-gaming**: OpenClaw Phase B adversarial tests
5. **Devnet smoke**: `./scripts/growth-engine.sh full`
6. **CI pipeline**: All `.github/workflows/ci.yml` jobs pass

---

## Verification Cycle

After each sprint:
1. Compute total TokenImpact of merged work
2. Run `./scripts/growth-engine.sh status` to measure actual consumption
3. Compare estimated vs actual -- delta measures methodology calibration
4. Feed delta into DER weights for next sprint
5. Record in `.claude/memory/DOKIMANCE_SPRINT_N.md`

The chain validates the developer. Not just tests validating code.

---

_The chain must not lie, must not break, must not favour. Dokimance: the fire that constitutes gold-status._
