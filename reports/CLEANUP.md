# CLEANUP: Dead Gas Keys, Alignment Assessment, P2 Assessments

**Date**: 2026-02-18
**Status**: 1 code change (gas keys), 3 assessments (no code changes)

---

## Fix 1: Dead Gas Key Removal

### Keys Removed (5)

| Key | Gas | Reason |
|-----|-----|--------|
| `challenge_claim` | 80,000 | No `MsgChallengeClaim` handler; `challenge_fact` serves this purpose |
| `revoke_fact` | 60,000 | No `MsgRevokeFact` handler exists |
| `archive_evidence` | 40,000 | No `MsgArchiveEvidence` handler exists |
| `audit_submit` | 50,000 | Replaced by `audit_evidence` (which maps to `MsgAuditEvidence`) |
| `audit_resolve` | 60,000 | Was paired with `audit_submit`; no real msg type |

### Grep Confirmation: Orphan Status

All 5 keys appear ONLY in:
- `app/gas.go` `TransactionGasCosts` map (the source being cleaned)
- Documentation files (`docs/PROTOCOL_PARAMETERS.md`, `docs/ZEROCLAW_INTEGRATION_SPEC.md`)
- Memory/planning files (`.claude/memory/`, `.claude/plans/`, `reports/GAS-AUDIT.md`)
- MCP tool name references (`lgm_challenge_claim` in docs — uses `challenge_fact` under the hood)

None of the 5 keys appear in `msgTypeURLToGas` (app/ante.go). No proto message type URL maps to any of them. They are truly dead entries in the gas cost table.

### File Changed
- `app/gas.go`: Removed 5 entries from `TransactionGasCosts` map

### Build Output
```
$ go build ./app/...
# Pre-existing error only: x/knowledge/client/cli (undefined ontology query commands)
# No new errors from gas key removal
```

---

## Fix 2: Alignment Module Assessment

### What It Does

The alignment module (`x/alignment/`) is a **fully functional autonomous feedback system** that monitors protocol health across 5 dimensions and applies corrective actions through the autopoiesis multiplier system.

**Architecture**: SENSE → JUDGE → CORRECT → REFLECT (meta-loop)

Called from `EndBlock` via `SenseJudgeCorrectReflect()` at epoch boundaries (default 1,111 blocks per epoch).

### The 5 Sensors

| Sensor | Index | Direction | Signals Collected From |
|--------|-------|-----------|----------------------|
| Truth Drift (TDI) | 0=perfect, 1M=corrupted | Higher=worse | knowledge, staking |
| Depth Coherence (DCI) | 0=random, 1M=deep | Higher=better | knowledge, ontology |
| Harmony (HI) | 0=chaotic, 1M=harmonious | Higher=better | autopoiesis |
| Fairness (FI) | 0=oligarchic, 1M=fair | Higher=better | staking, knowledge |
| Novelty (NI) | 0=stagnant, 1M=creative | Higher=better | knowledge, ontology |

Each sensor is a weighted composite of 5 sub-signals (weights sum to 1M BPS):
- **TDI**: reversal rate (25%), challenge asymmetry (20%), unanimity excess (20%), cross-validator correlation (20%), counter-fact deficit (15%)
- **DCI**: citation depth (25%), cross-domain ratio (20%), orphan rate (20%), ontology coverage (20%), supersession depth (15%)
- **HI**: multiplier variance (25%), oscillation rate (20%), bound saturation (20%), hormone conflict rate (20%), avg damping (15%)
- **FI**: reward Gini (25%), new entrant success (20%), slash proportionality (20%), whale dominance (20%), domain equity (15%)
- **NI**: new domain rate (25%), novel claims (20%), cross-domain bridges (20%), fact diversity (20%), supersession rate (15%)

### Correction Engine

Each sensor has a 4-level severity ladder: mild (20%) → moderate (40%) → severe (60%) → critical (80%).

Corrections flow through autopoiesis multiplier paths at priority 5 (above SSI priority 4):
- `challengereward.system` — Truth: reward for successful challenges
- `unanimitypenalty.system` — Truth: penalty for unanimous rounds
- `orphancap.system` — Understanding: cap confidence for orphan facts
- `crossdomainbonus.system` — Understanding/Creativity: bonus for cross-domain citations
- `damping.system` — Beauty: global damping coefficient
- `quadratic.system` — Justice: quadratic voting weight boost

All corrections are damped, rate-limited (5% max per epoch), and bounded.

### Meta-Loop (Recursive Self-Assessment)

Every 10 epochs, the alignment module evaluates whether its own corrections were effective:
- Computes `AlignmentHealthIndex` (effectiveness, overcorrection rate, sensor stability)
- Adjusts `CorrectionDamping` (stronger if ineffective, weaker if overcorrecting)
- Adjusts `SmoothingWindow` (wider if sensors noisy, narrower if stable)

### Module Structure

| File | Purpose | Lines |
|------|---------|-------|
| `keeper/keeper.go` | State management (params, observations, scores, corrections, meta-health) | 312 |
| `keeper/sensors.go` | Signal collection + score computation (5 sensors, 25 sub-signals) | 446 |
| `keeper/corrections.go` | Correction engine (5 dimensions × 4 severity levels) | 413 |
| `keeper/meta.go` | Meta-loop (evaluateOutcomes → computeAHI → applyMetaCorrections) | 209 |
| `module.go` | AppModule with EndBlock calling `SenseJudgeCorrectReflect` | 111 |
| `types/types.go` | All types, constants, BPS arithmetic, Gini, Shannon entropy | 557 |
| `types/expected_keepers.go` | 6 cross-module keeper interfaces | 106 |
| `types/keys.go` | Store key prefixes (0x01-0x07) | 18 |
| `types/errors.go` | Error types | 11 |
| `types/codec.go` | No-op codec registration (no Msg types) | 11 |

### Is It Functional or a Skeleton?

**Fully functional.** 48 tests pass (per memory). The module has:
- Complete state management with JSON serialization
- Full sensor implementation reading real cross-module data
- Damped correction engine with safety bounds
- Recursive meta-loop for self-assessment
- Event emission for observability
- Genesis import/export
- Per-dimension enable flags for gradual rollout
- Emergency halt integration (disables when chain halted)

### Should It Have a CLI?

**No — internal-only module is the correct design.** Reasons:

1. **No user-facing transactions**: The module has no `MsgServer` and no `Msg` types. It doesn't accept transactions from users. It's an autonomous system that runs in EndBlock.

2. **No CLI commands needed**: There are no transaction types to submit. The only interaction is governance-adjustable params (via standard param-change proposals).

3. **Query endpoints would be useful but not required**: Adding gRPC query endpoints for `GetScores`, `GetObservation`, `GetMetaHealth` would enable dashboards and monitoring. This is a nice-to-have, not a blocker.

### What Would It Take to Make It Externally Accessible?

If observability queries are desired:
1. Add `query.proto` with `QueryScoresRequest/Response`, `QueryObservationRequest/Response`, `QueryMetaHealthRequest/Response`
2. Add `keeper/grpc_query.go` implementing the query server
3. Add `client/cli/query.go` with CLI commands
4. Register query server in `module.go` `RegisterServices`
5. Estimated effort: ~200 lines of new code

No transaction CLI is needed — the module should remain autonomous.

---

## Fix 3: Schedule Cleanup Interval Assessment

### Location of Magic Number

**File**: `x/schedule/module.go:145`
```go
// Cleanup expired processes every 100 blocks
if sdkCtx.BlockHeight()%100 == 0 {
    cleaned := am.keeper.CleanupExpiredProcesses(sdkCtx)
```

The `100` is a hardcoded literal with no corresponding module parameter.

### Secondary Magic Number

**File**: `x/schedule/keeper/process.go:139`
```go
if currentHeight > p.CreatedAtBlock+params.ExpiredRetentionBlocks+p.ExecutionCount*100 {
```

The `*100` multiplier scales retention by execution count — also hardcoded.

### Existing Params

The schedule module params (from `state.pb.go`) include:
- `ExpiredRetentionBlocks` (default: 1000) — how long to keep completed/cancelled processes

But there is NO `CleanupIntervalBlocks` param.

### What the Change Would Look Like

1. **Add to proto**: `uint64 cleanup_interval_blocks = 7;` in `Params` message
2. **Default**: 100 (preserves current behavior)
3. **Validation**: Must be > 0
4. **Code change in module.go:145**:
   ```go
   params := am.keeper.GetParams(sdkCtx)
   if params.CleanupIntervalBlocks > 0 && sdkCtx.BlockHeight()%int64(params.CleanupIntervalBlocks) == 0 {
   ```
5. **For process.go:139**: Consider a `RetentionScalePerExecution` param, or leave it — the `*100` there is less concerning since it's a retention heuristic, not a frequency gate.

### Impact Assessment

- **P2 severity is correct**: The hardcoded 100 blocks (~252 seconds) is reasonable for cleanup frequency. Making it governance-adjustable is good practice but not urgent.
- No consensus-breaking risk from the current hardcoding.
- The `ExpiredRetentionBlocks` param already governs the retention period; only the cleanup check frequency is hardcoded.

---

## Fix 4: Vesting Rewards Tax Assessment

### Is the 7% Hardcoded?

**No — it is already a governance-adjustable parameter.**

**File**: `x/vesting_rewards/types/genesis.go:10`
```go
func DefaultParams() *Params {
    return &Params{
        ResearchShareBps: 70000, // 7%
```

The param `ResearchShareBps` is defined in the proto-generated `Params` message (`genesis.pb.go:88`):
```go
ResearchShareBps uint64 `protobuf:"varint,1,opt,name=research_share_bps,..."`
```

### Usage in Code

Both reward routing functions use the param, not a hardcoded value:

1. **`RouteFees`** (`rewards.go:75`): `researchBps := params.ResearchShareBps`
2. **`RouteReward`** (`rewards.go:45`): `researchShare := new(big.Int).Mul(amountBig, big.NewInt(int64(params.ResearchShareBps)))`

### The "7%" Constants in gas.go

```go
// app/gas.go:337
ResearchTaxBPS  uint64 = 70000  // 7% to research fund
ValidatorFeeBPS uint64 = 930000 // 93% to block producer / validators
```

These constants are used ONLY by the `FeeRouterDecorator` for **logging** (`ante.go:472`). The actual routing is done by `RouteFees()` which reads from params. The gas.go constants are documentation artifacts — they don't control actual routing.

### Assessment

**No code change needed.** The P2 concern is already addressed:
- Research tax rate IS governance-adjustable via `ResearchShareBps` param
- Default is 70000 BPS (7%), can be changed via param-change governance proposal
- Validation exists: `ResearchShareBps > 1000000` is rejected (genesis.go:55)
- The gas.go constants are cosmetic (logging only) and should be updated if the param ever changes, but they don't affect behavior

### Minor Recommendation (Deferred)

The gas.go constants `ResearchTaxBPS` and `ValidatorFeeBPS` could be removed or replaced with a comment pointing to the actual param. Currently they create a false impression that the rate is hardcoded. Not urgent since the FeeRouterDecorator only logs them.

---

## Build Output Summary

```
$ go build ./x/...
# ONLY pre-existing error: x/knowledge/client/cli (undefined ontology query commands)
# No new errors from gas key removal

$ go vet ./app/
# Same pre-existing errors plus x/tokens lock-copy warnings (proto types by value)
# No new issues from gas.go changes
```

Pre-existing build issues (not introduced by this cleanup):
1. `x/knowledge/client/cli/query.go` — 10+ undefined ontology query functions (stub references)
2. `x/tokens` — lock-copy vet warnings (proto MessageState passed by value)
