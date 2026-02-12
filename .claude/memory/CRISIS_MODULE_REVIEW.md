# PP Review: Crisis Management System

> **Review Date**: 2026-02-10
> **Reviewer**: PP (Full 30-gate review)
> **Territory**: 4 modules, ~6,600 LOC across Go + TypeScript
> **Verdict**: Architecturally sound, but 1 CRITICAL bug + 3 HIGH gaps need resolution

---

## Territory Map

| Module | Language | Files | LOC | Purpose |
|--------|----------|-------|-----|---------|
| `x/emergency/` | Go | 6 files | ~2,919 | BFT halt/revert/resume ceremonies |
| `x/capture_defense/` | Go | 5 files | ~2,561 | HHI detection, reputation, qualification |
| `x/capture_challenge/` | Go | 4 files | ~1,030 | Bounty system for capture reporting |
| `consensus/emergency/` | TypeScript | 7 files | ~1,438 | Parallel TS implementation (DevNet) |
| **Total** | | **22 files** | **~7,948** | |

### Architecture

```
            x/emergency (BFT ceremonies)
           ┌───────────────────────────────┐
           │ NORMAL → halt_voting → HALTED │
           │   ↑         ↓ fail            │
           │   │    ← normal               │
           │   │                           │
           │ HALTED → revert_voting → HALTED (post-revert)
           │   │         ↓ fail            │
           │   │    ← halted               │
           │   │                           │
           │ HALTED → resume_voting → NORMAL
           │             ↓ fail            │
           │        ← halted               │
           └───────────────────────────────┘

   x/capture_defense          x/capture_challenge
  ┌─────────────────┐        ┌──────────────────┐
  │ HHI analysis    │        │ Bounty system    │
  │ Timing corr.    │  ←──?  │ Evidence submit  │
  │ Verdict corr.   │        │ Governance resolve│
  │ Reputation      │        │ Slash/reward     │
  │ Qualification   │        └──────────────────┘
  └─────────────────┘
         ↓ events (unhandled)
         ??? (no wiring to x/emergency)
```

---

## Findings

### FINDING-1: Number() Overflow in TS Ceremony Quorum | CRITICAL

**Files**: `consensus/emergency/ceremony.ts:394-395, 402, 420`

The TypeScript emergency ceremony uses `Number()` on `bigint` stake values for quorum calculations:

```typescript
// ceremony.ts:394
const yesRatio = Number(ceremony.yesPrevoteStake) / Number(guardianStake);
// ceremony.ts:402
const noRatio = Number(ceremony.noPrevoteStake) / Number(guardianStake);
// ceremony.ts:420
const precommitRatio = Number(ceremony.precommitStake) / Number(guardianStake);
```

Guardian stakes use 18-decimal precision (e.g., `120000000000000000000000000` = 120M LGM). This exceeds `Number.MAX_SAFE_INTEGER` (2^53 ≈ 9×10^15), causing silent precision loss.

**Impact**: Quorum checks may produce incorrect results — ceremonies could finalize with insufficient quorum, or valid quorums could be rejected.

**This is the same bug pattern fixed in PP Consensus Review Session 8 (Finding #1)**, where `verification_round.ts` and `confidence.ts` used `Number(bigint)` on stakes. The emergency module was not included in that fix sweep.

**Go side is correct** — uses `math/big` basis-point arithmetic (`ceremony.go:263-265`):
```go
yesRatio := new(big.Int).Mul(yesStake, big.NewInt(1000000))
yesRatio.Div(yesRatio, guardianStake)
```

**Fix**: Convert to bigint basis-point arithmetic matching the Go pattern:
```typescript
const yesRatioBP = (ceremony.yesPrevoteStake * 1_000_000n) / guardianStake;
if (yesRatioBP >= BigInt(Math.floor(threshold * 1_000_000))) {
```

---

### FINDING-2: No Ante Handler Halt Enforcement | HIGH

**Files**: `x/emergency/keeper/state.go:33-41`, `app/ante.go` (missing)

The emergency module tracks halt status via `IsHalted()`:
```go
func (k Keeper) IsHalted(ctx sdk.Context) bool {
    switch status {
    case types.StatusHalted, types.StatusRevertVoting, types.StatusReverting, types.StatusResumeVoting:
        return true
    ...
```

**But nothing in the ante handler chain calls `IsHalted()` to block non-emergency transactions.** When the chain enters `halted` status, normal transactions (transfers, stakes, claims, governance) continue to be processed.

The TS implementation has the same issue — `EmergencyCeremonyManager.isHalted()` is available but never gated at the transaction processing level.

**Impact**: The halt ceremony has no enforcement — it's a status flag that nothing reads. The chain continues operating normally during an emergency halt.

**Fix**: Add `EmergencyHaltDecorator` to the ante handler chain:
```go
type EmergencyHaltDecorator struct {
    emergencyKeeper emergencykeeper.Keeper
}

func (ehd EmergencyHaltDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
    if ehd.emergencyKeeper.IsHalted(ctx) {
        for _, msg := range tx.GetMsgs() {
            if !emergencytypes.IsEmergencyMsg(msg) {
                return ctx, emergencytypes.ErrChainHalted
            }
        }
    }
    return next(ctx, tx, simulate)
}
```

Place it early in the chain (after SetUpContext, before gas/fee processing).

---

### FINDING-3: Vote Signatures Accepted but Never Verified | HIGH

**Files**: `x/emergency/types/types.go:183-184`, `x/emergency/keeper/ceremony.go:129,183`

All vote message types (`MsgVoteHalt`, `MsgVoteRevert`, `MsgVoteResume`) include a `Signature string` field. This signature is stored in the ceremony's `EmergencyVote` and `EmergencyPrecommit` structs, but **never verified anywhere**.

```go
type MsgVoteHalt struct {
    Voter       string `json:"voter"`
    Signature   string `json:"signature"`  // Stored but never checked
    ...
}
```

The `Voter` address is authenticated by the Cosmos SDK's `SigVerificationDecorator` via the transaction's outer signature. The inner `Signature` field is redundant.

**Impact**: The field creates a false sense of cryptographic verification. It also wastes storage in ceremony state.

**Options**:
- **Option A** (Recommended): Remove the `Signature` field from vote messages — the tx signature suffices for authentication. Keep it in types only if inner signatures are needed for non-repudiation audit trails (and then actually verify them).
- **Option B**: Verify the inner signature against the voter's public key and a canonical vote payload (ceremony_id + vote + block_height).

---

### FINDING-4: TS/Go Implementation Divergence | HIGH

**Files**: All emergency module files across both stacks

The TypeScript and Go emergency modules implement the same logic but diverge in critical details:

| Aspect | TypeScript | Go |
|--------|-----------|-----|
| Quorum math | `Number()` float64 **(broken)** | `big.Int` basis points (correct) |
| Quorum thresholds | Floating point (0.75, 0.8) | Basis points (750000, 800000) |
| State persistence | In-memory only | KVStore-backed (persistent) |
| Vote storage | `Map<string, Vote>` | Proto slice + helpers |
| Min distinct voters | 4 (hardcoded constant) | `params.MinDistinctVoters` (configurable) |
| Council virtual stake | 11,111 LGM (hardcoded) | `params.CouncilVirtualStake` (configurable) |
| Revert max depth | 111,111 blocks (hardcoded) | `params.MaxRevertDepth` (configurable) |

**Impact**: DevNet testing with the TS implementation may not catch bugs that would manifest in the Go production implementation, and vice versa. The quorum math divergence means DevNet ceremonies operate under different mathematical guarantees than production.

**Fix**: Align TS implementation to match Go patterns — use bigint basis points for quorum math, make constants configurable via consensus params.

---

### FINDING-5: Revert Ceremony Is a Signal, Not an Action | MEDIUM

**Files**: `x/emergency/keeper/ceremony.go:353-363`

When a revert ceremony is finalized, the handler only updates status:
```go
case types.CeremonyRevert:
    k.SetEmergencyStatus(ctx, types.StatusHalted)
    k.AddAuditEntry(ctx, ...)
```

The `TargetBlockNumber`, `TargetBlockHash`, and `TargetStateRoot` fields from the proposal are validated during submission but **never used during execution**. No actual state rollback occurs. Similarly, the TS `EmergencyCeremonyManager.processBlock()` just sets status to "halted" on revert finalization.

**Context**: Actual state reversion requires ABCI-level coordination (e.g., Cosmos SDK `x/upgrade` with a fork plan, or CometBFT block rollback). A module message handler cannot roll back committed state. This is an inherent architectural limitation.

**Impact**: The revert ceremony gives the impression of functionality that doesn't exist. Guardians may believe voting through a revert will actually restore state to the target block.

**Fix**: Document this clearly in the types and in operator docs. Consider renaming to "VoteRevertIntent" to signal that it's a governance decision, not an automated execution. The actual revert would require validator coordination out-of-band (stop nodes → rollback → restart with new genesis).

---

### FINDING-6: Capture Detection Not Wired to Emergency Module | MEDIUM

**Files**: `x/capture_defense/keeper/detection.go:192-211`, `x/emergency/` (no imports)

The capture defense module's `RunCaptureDetection()` emits events when HHI thresholds are crossed:
```go
if metrics.RiskScore >= types.HerfindahlTriggerThreshold {
    ctx.EventManager().EmitEvent(sdk.NewEvent("capture_risk_detected", ...))
}
if metrics.RiskScore >= types.HerfindahlPauseThreshold {
    ctx.EventManager().EmitEvent(sdk.NewEvent("capture_risk_critical", ...))
}
```

These events are never consumed. There is **no integration between the three crisis modules**:
- `x/capture_defense` detects capture → emits events → nothing happens
- `x/capture_challenge` resolves challenges → slashes validators → doesn't notify emergency
- `x/emergency` manages halt ceremonies → doesn't respond to capture signals

**Impact**: Automated capture detection doesn't feed into any enforcement mechanism. The three modules are independent islands. A critical HHI score doesn't trigger a guardian alert, let alone an auto-halt.

**Fix**: Wire at minimum:
1. `capture_risk_critical` event → auto-notify guardians (emit structured alert)
2. Upheld capture challenge → auto-pause domain (already partially implemented via `DomainPausedUntil`)
3. Consider: critical HHI score → auto-propose halt (requires keeper cross-dependency)

---

### FINDING-7: Challenge Resolution is Centralized | MEDIUM

**Files**: `x/capture_challenge/keeper/msg_server.go:109`

```go
if msg.Resolver != k.authority {
    return nil, types.ErrUnauthorizedResolver
}
```

Challenge resolution requires the governance authority address. There is no:
- Decentralized jury mechanism
- Automated resolution from on-chain evidence metrics
- Escalation path for disputed resolutions
- Time-based auto-resolution for expired reviews

For a protocol built around decentralized truth-finding, having centralized challenge resolution is incongruent.

**Impact**: The capture challenge system's credibility depends on a single authority address. If that address is compromised or unresponsive, challenges can be permanently stuck in `under_review`.

**Fix**: Implement at least time-based auto-expiry (if review deadline passes with no resolution, challenge expires and stake is returned). For production, consider guardian-jury resolution using the same ceremony pattern from `x/emergency`.

---

### FINDING-8: Panics in State Operations | MEDIUM

**Files**: `x/emergency/keeper/state.go:28,51,53,288,290`

```go
panic("failed to set emergency status: " + err.Error())
panic("failed to marshal emergency ceremony: " + err.Error())
panic("failed to set emergency ceremony: " + err.Error())
panic("failed to marshal emergency params: " + err.Error())
panic("failed to set emergency params: " + err.Error())
```

Five `panic()` calls in state operations. While this follows Cosmos SDK conventions (KV store operations shouldn't fail in normal operation), these panics will crash the entire node. In the emergency module specifically, this is ironic — the module designed to handle crises can itself cause a crisis via node crash.

**Impact**: If JSON serialization ever fails (e.g., due to a malformed ceremony struct from a bug), the emergency module panics and takes down the validator.

**Fix**: LOW priority — follows SDK convention. But consider logging and returning errors for marshal operations rather than panicking, especially in the module that handles emergencies.

---

### FINDING-9: Ceremony Phase String Inconsistency | LOW

**Files**: `x/emergency/keeper/ceremony.go` (throughout)

Phase comparisons mix styles:
```go
// Style A: explicit string conversion
if ceremony.Phase != string(types.PhasePrevote) {   // line 111
ceremony.Phase = string(types.PhasePrecommit)        // line 268

// Style B: direct comparison (works because CeremonyPhase is a string type)
if ceremony.Phase != types.PhaseFinalized {           // line 336 (HandleCeremonyFinalization)
```

Both work — `CeremonyPhase` is a named `string` type. But the inconsistency suggests the code was written by copy-paste with partial cleanup.

**Fix**: Pick one style and apply uniformly. Recommendation: drop all `string()` casts since `CeremonyPhase` is already a string type.

---

### FINDING-10: `prefixEndBytes` Duplicated | LOW

**Files**: `x/emergency/keeper/keeper.go:42-55` (and ~22 other modules)

The `prefixEndBytes()` helper function is copy-pasted identically across 23+ modules. This is a codebase-wide issue documented in the Store Architecture Review.

**Fix**: Extract to a shared utility package. Not specific to crisis management.

---

### FINDING-11: Capture Defense Uses float64 for HHI | LOW

**Files**: `x/capture_defense/keeper/detection.go:78-79, 130, 137`

```go
var hhi float64
for _, count := range validatorCounts {
    share := float64(count) / float64(totalParticipations)
    hhi += share * share
}
return uint64(hhi * 1_000_000)
```

HHI computation uses `float64` arithmetic before converting to `uint64` basis points. For verification count data (not monetary), float64 precision is adequate — participation counts are small integers (hundreds, not 10^18). This is acceptable but inconsistent with the project's general preference for integer arithmetic.

Similarly, `computeTimingCorrelation` uses `math.Sqrt(variance)` — appropriate for statistical analysis.

**Impact**: Negligible — the inputs are small enough that float64 is precise.

---

### FINDING-12: Good Test Coverage | INFO (PASS)

The Go emergency module has **25+ test functions** in `keeper_test.go` (959 lines) covering:
- Guardian tier checks (tier 3 active, tier 2 rejected, unknown, council bootstrap, council expiry)
- Full ceremony lifecycles (halt, resume-after-halt, revert-after-halt)
- Timeouts and anti-abuse (cooldown, per-guardian limit, epoch reset)
- Access control (non-guardian proposal/vote rejected)
- Status validation (can't halt when halted, can't revert when not halted, etc.)
- Voting mechanics (duplicate prevote, precommit-requires-prevote, wrong phase)
- State queries (status, active ceremony, completed ceremonies, audit log)
- Genesis roundtrip

The `x/capture_defense` has 1,271 lines of tests, `x/capture_challenge` has 448 lines.

---

## Gate Summary

### TRUTH (5 gates)
| Gate | Status | Notes |
|------|--------|-------|
| T-1 Ground Truth | FAIL | FINDING-2: IsHalted() never enforced |
| T-2 Data Fidelity | PASS | Ceremony state faithfully represents protocol |
| T-3 Precision | FAIL | FINDING-1: Number() overflow in TS quorum math |
| T-4 Completeness | WARN | FINDING-5: Revert is signal-only, not action |
| T-5 Citation | PASS | Types are well-documented with comments |

### UNDERSTANDING (6 gates)
| Gate | Status | Notes |
|------|--------|-------|
| U-1 Purpose Clarity | PASS | Clear module separation and documentation |
| U-2 Interaction Model | FAIL | FINDING-4: TS/Go divergence on key details |
| U-3 State Space | PASS | 6 states, all transitions handled correctly |
| U-4 Invariants | PASS | Anti-abuse counters, quorum thresholds maintained |
| U-5 Failure Modes | PASS | Timeouts, quorum impossible, ceremony failures all handled |
| U-6 Exhaustion | FAIL | FINDING-6: Capture detection → emergency not wired |

### BEAUTY (7 gates)
| Gate | Status | Notes |
|------|--------|-------|
| B-1 Naming | PASS | Consistent naming across modules |
| B-2 Structure | PASS | keeper/types/module separation |
| B-3 Symmetry | PASS | halt/revert/resume follow identical patterns |
| B-4 Minimalism | PASS | No unnecessary complexity |
| B-5 Clarity | PASS | Intent clear from code structure |
| B-6 Cohesion | WARN | Three crisis modules not interconnected (FINDING-6) |
| B-7 Conformance | PASS | Follows Cosmos SDK module patterns |

### JUSTICE (6 gates)
| Gate | Status | Notes |
|------|--------|-------|
| J-1 Access Control | WARN | FINDING-3: Signatures not verified; FINDING-7: Centralized challenge resolution |
| J-2 Resource Allocation | PASS | Proper bounty pool mechanics, stake-weighted voting |
| J-3 Accountability | PASS | FINDING-12: Comprehensive audit logging |
| J-4 Error Handling | WARN | FINDING-8: Panics in state operations |
| J-5 Precedent | PASS | Follows Cosmos SDK patterns (BFT voting, keeper/module) |
| J-6 Evidence | PASS | Evidence-based challenge system with 7 evidence types |

### CREATIVITY (6 gates)
| Gate | Status | Notes |
|------|--------|-------|
| C-1 Innovation | PASS | Novel 3-phase BFT ceremony for emergency ops |
| C-2 Extension Points | PASS | Configurable params, genesis council bootstrap |
| C-3 Adaptability | PASS | Council expiry, epoch-based counter reset |
| C-4 Integration | FAIL | FINDING-2: No ante handler enforcement |
| C-5 Future-Proofing | WARN | FINDING-7: Resolution model needs decentralization path |
| C-6 Proof | PASS | FINDING-12: 25+ tests, 2,677 lines of test code |

### Score: 22/30 PASS, 4 WARN, 4 FAIL

---

## Priority Fixes

| # | Finding | Severity | Effort | Impact |
|---|---------|----------|--------|--------|
| 1 | FINDING-1: TS Number() overflow | CRITICAL | Low | Quorum math broken |
| 2 | FINDING-2: No halt enforcement in ante handler | HIGH | Medium | Halt has no effect |
| 3 | FINDING-3: Vote signatures never verified | HIGH | Low | Remove or verify |
| 4 | FINDING-4: TS/Go divergence | HIGH | Medium | DevNet ≠ production |
| 5 | FINDING-6: Capture detection not wired | MEDIUM | Medium | No automated response |
| 6 | FINDING-7: Centralized challenge resolution | MEDIUM | High | Single point of failure |
| 7 | FINDING-5: Revert is signal-only | MEDIUM | Low (doc) | Misleading functionality |

---

## Previous Review: x/crisis Module Removal

The earlier review (2026-02-09) recommended removing the Cosmos SDK `x/crisis` module from `app/app.go`. That recommendation stands — `x/crisis` is deprecated, broken, and superseded by `x/emergency`. See the original review for full details on the 10 removal points in `app/app.go`.
