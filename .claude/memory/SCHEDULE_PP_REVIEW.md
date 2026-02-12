# PP Review: Schedule Module (DMSS)

**Date**: 2026-02-10
**Scope**: `x/schedule/` (Go SDK, 2,115 LOC) + `core/schedule/` (TS prototype, 3,210 LOC)
**Reviewer**: PP 30-gate analysis

---

## Executive Summary

The schedule module implements DMSS (Decentralized Message Scheduling System) — a patent-based system for time and logic-based automated execution. The TS prototype is mature with 14+ security fixes applied. The Go SDK module is structurally sound but has **3 CRITICAL findings**, **4 HIGH**, and **5 MEDIUM** issues.

**Score**: 32/50

---

## TRUTH (Grounded in Reality)

### F1 — CRITICAL: MaxGasPerBlock Not Enforced

**File**: `x/schedule/keeper/process.go:15-113`
**Gate**: T1 (Verifiable Claims), T5-Claude (Citation)

`Params.MaxGasPerBlock` (default 2,000,000) exists in genesis and validates as non-zero, but `ProcessDueSchedules()` has **no gas metering**. Every due schedule executes unconditionally with no budget cap.

```go
// process.go:19 — iterates ALL due processes with no gas limit
for _, processId := range dueIds {
    // ... executes without checking cumulative gas
}
```

The TS prototype correctly enforces this (SPU `processSchedules` tracks cumulative gas per block via `maxScheduleExecutionGas`). The Go module skips it entirely.

**Impact**: An attacker with 100 schedules due at the same block can cause O(100) bank transfers in a single EndBlock, potentially exceeding block gas limits or causing consensus timeouts.

**Remediation**: Add gas metering loop:
```go
var gasUsed uint64
params := k.GetParams(ctx)
for _, processId := range dueIds {
    if gasUsed >= params.MaxGasPerBlock {
        break
    }
    // ... execute ...
    gasUsed += estimateProcessGas(process)
}
```

---

### F2 — CRITICAL: big.Int SetString Return Value Unchecked

**File**: `x/schedule/keeper/process.go:32-35`
**Gate**: T2 (Source Integrity)

```go
remaining := new(big.Int)
remaining.SetString(process.RemainingFee, 10)  // bool return IGNORED
feePerExec := new(big.Int)
feePerExec.SetString(process.FeePerExecution, 10)  // bool return IGNORED
```

If `RemainingFee` or `FeePerExecution` is empty string or non-numeric (e.g., corrupted state, genesis import error), `SetString` returns `(nil, false)` and the `big.Int` remains at **zero**. The fee check `remaining.Cmp(feePerExec) < 0` then passes (0 >= 0), allowing **free execution**.

Same pattern at `process.go:65-66` for `TransferValue`.

**Impact**: Corrupted or maliciously crafted genesis state could enable unlimited free schedule execution.

**Remediation**: Check all SetString returns; error or skip process on parse failure.

---

### F3 — CRITICAL: Logic Conditions Accepted but Never Evaluated

**File**: `x/schedule/keeper/process.go:63-78`, `types/types.go:64-75`
**Gate**: T3 (Completeness), T1 (Verifiable Claims)

The Go module accepts `LogicCondition` and `CompoundCondition` in `MsgCreateSchedule`, stores them, and indexes them. But `ProcessDueSchedules` **only evaluates time conditions** — it fires schedules based purely on `NextExecuteAt` block height. Logic conditions are never read or evaluated.

A schedule with a logic-only condition gets `NextExecuteAt = currentHeight + 1` (from `calculateNextExecuteAt`), meaning it fires on the very next block **regardless of whether the logic condition is satisfied**.

```go
// msg_server.go:349-351 — non-time conditions check every block
// For logic-only conditions, check every block
return currentHeight + 1
```

But there's no actual logic evaluation in `ProcessDueSchedules`. The schedule just fires.

**Impact**: Logic-based schedules execute unconditionally. A schedule meant to fire "when balance > 1000" fires immediately every block, draining prepaid fees.

**Remediation**: Either:
1. Reject logic conditions at `CreateSchedule` validation (simplest for MVP)
2. Add a `StateProvider` interface to keeper and evaluate logic conditions in `ProcessDueSchedules`

---

## UNDERSTANDING (Correct Comprehension)

### F4 — HIGH: WatchedKeyIndexPrefix Dead Code

**File**: `x/schedule/types/keys.go:27`
**Gate**: U2 (Pattern Recognition)

```go
WatchedKeyIndexPrefix = []byte{0x06} // {state_key}/{process_id} -> nil
```

This prefix is defined but **never used** in any keeper code. No watched key indexes are written or read. This is the Go counterpart of the TS `watchedKeys` index, but since logic conditions aren't evaluated, it's dead infrastructure.

**Impact**: Low — wastes a prefix byte. But signals incomplete implementation that could confuse future developers.

**Remediation**: Either remove the prefix (if logic conditions are deferred) or implement watched key indexing when adding logic evaluation.

---

### F5 — HIGH: CountActiveByCreator is O(n) Per Schedule Creation

**File**: `x/schedule/keeper/state.go:200-210`
**Gate**: U4 (Edge Cases)

```go
func (k Keeper) CountActiveByCreator(ctx sdk.Context, creator string) uint32 {
    processes := k.GetProcessesByCreator(ctx, creator)  // fetches ALL, deserializes ALL
    var count uint32
    for _, p := range processes {
        if p.Status == types.StatusActive || p.Status == types.StatusPaused {
            count++
        }
    }
    return count
}
```

Called on every `CreateSchedule`. For a creator with 99 active schedules, this deserializes 99 JSON blobs from the KV store, iterates them all, just to get a count.

**Impact**: O(n) per creation where n = total processes (not just active). Performance degrades as schedules accumulate.

**Remediation**: Maintain a dedicated `activeCountByCreator` counter key, incremented/decremented atomically on status changes.

---

### F6 — HIGH: ExpiresAtBlock Never Set on Creation

**File**: `x/schedule/keeper/msg_server.go:77-93`
**Gate**: U3 (Dependency Mapping)

The `ScheduleProcess` struct has `ExpiresAtBlock uint64` (types.go:128) and `ProcessDueSchedules` checks it (process.go:99). But `CreateSchedule` never populates it — the field is always zero (no expiry).

```go
// msg_server.go:77-93 — ExpiresAtBlock not set
process := types.ScheduleProcess{
    Id:               processId,
    Creator:          msg.Creator,
    // ... no ExpiresAtBlock field ...
}
```

`MsgCreateSchedule` also has no `ExpiresAtBlock` field (types.go:190-200).

**Impact**: Unlimited-duration schedules can't be expired. A schedule with `MaxExecutions=0` (unlimited) and no expiry runs forever until fee exhaustion or manual cancellation.

**Remediation**: Add `ExpiresAtBlock` to `MsgCreateSchedule`. Validate against `MaxFutureBlocks` param. Default to `currentHeight + MaxFutureBlocks` if not specified and `MaxExecutions == 0`.

---

### F7 — HIGH: Transfer Errors Silently Swallowed

**File**: `x/schedule/keeper/process.go:64-78`
**Gate**: U5-Claude (Exhaustion)

```go
if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, targetAddr, sdk.NewCoins(coin)); err != nil {
    k.Logger(ctx).Error("schedule transfer failed",
        "process_id", process.Id,
        "error", err,
    )
    // NO return, NO status change, NO event — execution continues
}
```

If a scheduled transfer fails (e.g., module account underfunded), the error is logged but:
- The fee is still deducted
- `ExecutionCount` is still incremented
- The process is marked as having executed
- No `schedule_transfer_failed` event is emitted

**Impact**: User's prepaid fees consumed without the intended transfer executing. No on-chain signal of failure.

**Remediation**: On transfer failure, either:
1. Revert the entire execution (don't deduct fee, don't increment count)
2. Emit a `schedule_execution_failed` event and set a failure status

---

## BEAUTY (Elegant Design)

### F8 — MEDIUM: Non-Periodic Conditions Check Every Block

**File**: `x/schedule/keeper/process.go:147-148`, `msg_server.go:349-351`
**Gate**: B3 (Efficiency)

```go
// Non-periodic: check next block
return currentHeight + 1
```

Both `calculateNextPeriodicExecution` and `calculateNextExecuteAt` return `currentHeight + 1` for non-time conditions. This means:
- A one-shot absolute schedule that already missed its block is re-checked every block
- A logic-only schedule is indexed at every single block height

This creates N time index entries over N blocks for a single long-running logic schedule.

**Impact**: State bloat. Each "check every block" creates a time index entry, then removes it, then creates the next one. The schedule will be in `GetDueProcesses` result set every block.

**Remediation**: For one-shot schedules that missed their window, mark as expired. For logic conditions, use watched key triggers instead of polling.

---

### F9 — MEDIUM: Cleanup Retention Heuristic Is Non-Obvious

**File**: `x/schedule/keeper/process.go:127`
**Gate**: B1 (Simplicity)

```go
if currentHeight > p.CreatedAtBlock+params.ExpiredRetentionBlocks+p.ExecutionCount*100 {
    k.DeleteProcess(ctx, p)
}
```

The `ExecutionCount*100` term extends retention for frequently-executed processes. A process that ran 1000 times gets 100,000 extra blocks of retention (~3 days). The reasoning isn't documented.

The TS prototype uses a simpler `expiredProcessRetentionBlocks` without execution count scaling.

**Impact**: Low — unintuitive behavior, hard to reason about storage reclamation.

**Remediation**: Use `lastExecutionBlock + retentionBlocks` or just `completedAtBlock + retentionBlocks`. Document the rationale if execution-based scaling is intentional.

---

### F10 — MEDIUM: Redundant Index Values

**File**: `x/schedule/keeper/state.go:88,94,101,109`
**Gate**: B2 (Minimal Complexity)

All secondary indexes store `process.Id` as both part of the key AND as the value:

```go
acctKey := accountIndexKey(process.Creator, process.Id)
store.Set(acctKey, []byte(process.Id))  // processId in key AND value
```

The value is redundant — the processId is already extractable from the key suffix. Using `[]byte{}` (nil value, key-only index) would save ~10-30 bytes per index entry.

**Impact**: Low — minor storage overhead. Not a correctness issue.

**Remediation**: Use nil/empty values for index entries where the key already contains the lookup value.

---

### F11 — MEDIUM: Params.Validate() Doesn't Validate Bounds

**File**: `x/schedule/types/types.go:154-169`
**Gate**: B4 (Coherent Patterns)

```go
func (p Params) Validate() error {
    // Only checks != 0, no upper bounds
    if p.MaxActivePerAccount == 0 { ... }
    if p.MaxGasPerBlock == 0 { ... }
    if p.MinIntervalBlocks == 0 { ... }
    if p.MinPrepaidExecutions == 0 { ... }
    return nil
}
```

No validation for:
- `MaxActivePerAccount` upper bound (could be set to `uint32_max`)
- `MaxFutureBlocks` not validated at all (could be 0)
- `ExpiredRetentionBlocks` not validated (could be 0, causing immediate cleanup)
- `MinIntervalBlocks` should have upper bound too

**Remediation**: Add reasonable bounds checks, validate `MaxFutureBlocks` > 0 and `ExpiredRetentionBlocks` > 0.

---

### F12 — MEDIUM: No Query Request Validation

**File**: `x/schedule/keeper/grpc_query.go:27-40`
**Gate**: B5 (Defensive)

```go
func (q queryServer) Process(goCtx context.Context, req *types.QueryProcessRequest) (*types.QueryProcessResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)
    process, found := q.GetProcess(ctx, req.ProcessId)  // req could be nil
```

No nil check on `req`. If `req` is nil, this panics with nil pointer dereference. Same for `ProcessesByCreator`.

**Impact**: Low — gRPC framework typically prevents nil requests, but defense-in-depth.

**Remediation**: Add `if req == nil { return nil, status.Error(codes.InvalidArgument, "empty request") }`.

---

## JUSTICE (Right Placement)

### F13 — Dual-Stack Parity Gap

**Gate**: J1 (Contextual Fit)

| Feature | TS | Go | Gap |
|---------|----|----|-----|
| Process statuses | 8 (pending, active, executing, paused, completed, cancelled, exhausted, expired) | 5 (active, paused, completed, cancelled, exhausted) | Go missing: pending, executing, expired |
| Logic evaluation | Full (state_compare, threshold_crossed, event_occurred) | None | **CRITICAL** |
| Gas metering | Per-block budget enforced | Param exists, not enforced | **CRITICAL** |
| Per-process locks | Async locks prevent concurrent modification | N/A (consensus) | OK — different execution model |
| Watched keys | Full index + trigger | Dead prefix | Go incomplete |
| Threshold state | Per-process isolation (CRIT-004) | N/A | Go missing |
| Signature verification | Defense-in-depth (HIGH-001) | Via SDK `GetSigners` | OK — different layer |
| Refunds on cancel | `submitRefund()` | `bankKeeper.SendCoinsFromModuleToAccount` | Both work |
| Cleanup | Retention + expiry | Retention heuristic every 100 blocks | Both work |
| ExpiresAt | Full support | Field exists, never populated | Go incomplete |

**Impact**: Agents migrating from TS devnet to Go chain will lose logic-based scheduling. This must be documented.

---

## CREATIVITY (Novel Solutions)

### Architectural Strengths

1. **Patent-backed design** (SPU 131322 / SPS 131313) provides clear separation between message processing, state management, and condition evaluation
2. **Prepaid fee model** prevents gas griefing — users pay upfront, no surprise costs
3. **Multi-level indexing** (time, account, status, linked entity) enables O(log n) queries
4. **Linked entity pattern** connects schedules to higher-level protocol entities (projects, partnerships, disputes)
5. **TS implementation is exemplary** — 14 security fixes documented inline with codes, per-process locks, threshold state isolation, deterministic timestamps

### What's Missing (Go)

1. **No CLI commands** — GetTxCmd/GetQueryCmd return nil
2. **No ante handler integration** — schedule execution doesn't go through the ante handler chain
3. **No module account funding mechanism** — transfers require the schedule module account to have funds, but there's no documented way to fund it beyond `SendCoinsFromAccountToModule` during creation

---

## Findings Summary

| # | Severity | Title | File | Line |
|---|----------|-------|------|------|
| F1 | **CRITICAL** | MaxGasPerBlock not enforced | process.go | 15-113 |
| F2 | **CRITICAL** | big.Int SetString return unchecked | process.go | 32-35 |
| F3 | **CRITICAL** | Logic conditions accepted but never evaluated | process.go | 63-78 |
| F4 | HIGH | WatchedKeyIndexPrefix dead code | keys.go | 27 |
| F5 | HIGH | CountActiveByCreator is O(n) | state.go | 200-210 |
| F6 | HIGH | ExpiresAtBlock never set on creation | msg_server.go | 77-93 |
| F7 | HIGH | Transfer errors silently swallowed | process.go | 64-78 |
| F8 | MEDIUM | Non-periodic conditions check every block | process.go | 147-148 |
| F9 | MEDIUM | Cleanup retention heuristic non-obvious | process.go | 127 |
| F10 | MEDIUM | Redundant index values | state.go | 88,94 |
| F11 | MEDIUM | Params.Validate() missing bounds | types.go | 154-169 |
| F12 | MEDIUM | No query request nil checks | grpc_query.go | 27-40 |
| F13 | INFO | Dual-stack parity gap | — | — |

---

## Remediation Priority

### Immediate (Block mainnet)
1. **F1**: Add gas metering to ProcessDueSchedules
2. **F2**: Check all SetString returns, error on parse failure
3. **F3**: Reject logic-only conditions at CreateSchedule validation (or implement evaluation)

### Before Mainnet
4. **F7**: Handle transfer errors (revert execution or emit failure event)
5. **F6**: Add ExpiresAtBlock to MsgCreateSchedule
6. **F5**: Optimize CountActiveByCreator with dedicated counter
7. **F11**: Add bounds validation to Params.Validate()

### Polish
8. **F4**: Remove dead WatchedKeyIndexPrefix (or implement)
9. **F8**: Use watched keys for logic conditions instead of per-block polling
10. **F9**: Simplify cleanup retention formula
11. **F10**: Use nil values for index entries
12. **F12**: Add nil checks to query handlers

---

## Test Coverage Assessment

### Go Tests (302 LOC, 7 tests)
- Params CRUD
- Sequence generation
- Process storage/retrieval
- Creator-based queries
- CreateSchedule handler
- ProcessDueSchedules (basic one-shot)
- CancelSchedule with refund

**Missing**:
- Logic condition rejection test
- Gas budget exhaustion test
- Periodic rescheduling test
- ExpiresAtBlock expiry test
- Transfer failure handling test
- Genesis import/export roundtrip
- AddFee + reactivation from exhausted
- PauseSchedule / ResumeSchedule handlers
- Multiple schedules in same block
- Edge case: remaining fee exactly equals fee per execution

### TS Tests (2,843 LOC, ~80+ tests)
- Comprehensive across all 4 files
- All condition types and edge cases covered
- Concurrency, threshold isolation, fee underflow tested
- Exemplary coverage

---

## PP Gate Scorecard

| Dimension | Score | Notes |
|-----------|-------|-------|
| TRUTH (5 gates) | 5/10 | 3 CRITICALs: unmetered gas, unchecked parse, phantom logic eval |
| UNDERSTANDING (5 gates) | 6/10 | Dead code, O(n) performance, missing field population |
| BEAUTY (7 gates) | 8/14 | Clean structure but wasteful patterns, weak validation |
| JUSTICE (6 gates) | 7/12 | Good placement, but dual-stack divergence is concerning |
| CREATIVITY (7 gates) | 6/14 | TS is creative and well-hardened; Go is a thin skeleton |
| **TOTAL** | **32/50** | |
