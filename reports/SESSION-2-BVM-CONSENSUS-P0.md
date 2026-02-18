# SESSION-2: BVM Consensus Integration — P0/P1 Security Fixes

**Date**: 2026-02-18
**Branch**: `feat-cicd-opensource-flow`
**Audit Source**: `reports/BVM-CONSENSUS.md`
**Status**: All 5 fixes implemented and tested

---

## Summary

Fixed 3 P0 (critical) and 2 P1 (high) vulnerabilities in the BVM (Belief Virtual Machine) consensus integration layer. These vulnerabilities created a disconnect between the Cosmos SDK transaction pipeline and BVM execution, allowing gas exhaustion attacks, chain halts from VM panics, and privilege escalation through scheduled executions.

---

## Fixes Implemented

### P0-1: No Gas Bridge (SDK gas disconnected from VM gas)

**File**: `x/bvm/keeper/msg_server.go:621-625`

**Problem**: `CallContract` consumed a flat 21,000 SDK gas regardless of actual VM gas used. An attacker could submit a tx with `GasLimit: 2^64-1`, consume zero SDK gas beyond the flat cost, and fill blocks with free computation.

**Fix**: After VM execution completes, bridge the VM gas consumed back to the SDK gas meter:
```go
ctx.GasMeter().ConsumeGas(result.GasUsed, "bvm_execution")
```

If the bridged gas exceeds the transaction's SDK gas limit, the SDK catches the `ErrorOutOfGas` panic and reverts the entire transaction including any value transfers. This is the correct behavior — the cache-wrapped context ensures atomicity.

**Tests**: `TestP0_1_GasBridge_CallContract`, `TestP0_1_GasBridge_OutOfGas`

---

### P0-2: Scheduled Execution Consumes Zero SDK Gas

**File**: `x/bvm/keeper/msg_server.go:878-1039`

**Problem**: `ExecutePendingSchedules` runs in `BeginBlock` with an infinite gas meter. There was no limit on aggregate VM gas consumed per block, allowing an attacker to schedule hundreds of contracts that collectively consume O(minutes) of CPU per block, causing block production stalls.

**Fix**: Introduced a per-block gas budget (`MaxScheduledGasPerBlock = 5,000,000` VM gas) that caps aggregate scheduled execution. Each schedule's effective gas is also capped to the remaining budget:

```go
if totalGasUsed >= types.MaxScheduledGasPerBlock {
    break // defer remaining to next block
}
remainingBudget := types.MaxScheduledGasPerBlock - totalGasUsed
effectiveMaxGas := schedule.MaxGas
if effectiveMaxGas > remainingBudget {
    effectiveMaxGas = remainingBudget
}
```

Schedules that don't fit in the current block's budget are **deferred** (their block index entries remain), and `GetPendingSchedules` picks them up next block via range iterator.

**Constants added** (`x/bvm/types/types.go:23-33`):
- `MaxContractGasLimit = 10,000,000` — per-call VM gas cap
- `MaxScheduledGasPerBlock = 5,000,000` — aggregate per-block budget (~15% of block gas)

**Tests**: `TestP0_2_ScheduledGasBudget`, `TestP0_2_EffectiveGasCap`

---

### P0-3: No Panic Recovery in VM Execution

**File**: `x/bvm/keeper/msg_server.go:398-418`

**Problem**: `interp.Execute()` could panic on malformed bytecode, edge cases in opcode handling, or unexpected state. In `CallContract`, the SDK's `runMsgs` wrapper catches panics for transaction handlers. But in `BeginBlock` (scheduled execution), an unrecovered panic **halts the entire chain**.

**Fix**: Added `safeExecute()` wrapper with `defer/recover`:

```go
func safeExecute(
    interp *vm.Interpreter,
    bytecode []byte,
    execCtx *vm.ExecutionContext,
    stateDB vm.StateDB,
    host vm.HostFunctions,
) (result *vm.ExecutionResult, panicked bool) {
    defer func() {
        if r := recover(); r != nil {
            panicked = true
            result = &vm.ExecutionResult{
                Success: false,
                GasUsed: execCtx.GasLimit, // charge full gas as punishment
            }
        }
    }()
    return interp.Execute(bytecode, execCtx, stateDB, host), false
}
```

Used in both `CallContract` (line 612) and `ExecutePendingSchedules` (line 996). On panic:
- The execution is marked as failed
- Full gas is charged (punishes the triggering contract)
- Error is logged for operator visibility
- Subsequent schedules continue executing

**Tests**: `TestP0_3_PanicRecovery_CallContract`, `TestP0_3_PanicRecovery_ScheduledExecution`

---

### P1-1: Unbounded msg.GasLimit

**File**: `x/bvm/types/types.go:65-67, 126-128`

**Problem**: `MsgCallContract.GasLimit` and `MsgScheduleContract.MaxGas` had no upper bound. A user could set `GasLimit: 2^64-1`, causing the VM interpreter to allocate a gas meter with unreasonable limits.

**Fix**: Added upper-bound validation in `ValidateBasic()` for both message types:

```go
if m.GasLimit > MaxContractGasLimit {
    return fmt.Errorf("gas limit %d exceeds maximum %d", m.GasLimit, MaxContractGasLimit)
}
```

`MaxContractGasLimit = 10,000,000` — sufficient for complex contracts, prevents abuse.

**Tests**: `TestP1_1_GasLimitCap_CallContract`, `TestP1_1_GasLimitCap_ScheduleContract`

---

### P1-2: Scheduled Execution Privilege Escalation

**Files**:
- `x/bvm/types/keys.go:34` — `ScheduleCapabilityKeyPrefix = []byte{0x12}`
- `x/bvm/keeper/state.go:379-405` — `SetScheduleCapabilities`, `GetScheduleCapabilities`
- `x/bvm/keeper/msg_server.go:1043-1066` — `resolveCallerCapabilities` helper
- `x/bvm/keeper/msg_server.go:791-794` — capability storage in `ScheduleContract`
- `x/bvm/keeper/msg_server.go:700-704` — capability storage in `ScheduleExecution`
- `x/bvm/keeper/msg_server.go:935-955` — capability loading in `ExecutePendingSchedules`

**Problem**: Scheduled executions always ran with `fullCaps` (all capabilities enabled). A restricted session key (e.g., `CanTransfer: true, CanStake: false`) could schedule a contract execution that would later run with full permissions, escalating privileges.

**Fix**: Store the caller's current capabilities at schedule creation time in a separate KV store entry (keyed by `0x12 + scheduleId`, JSON-encoded). At execution time, load the stored capabilities instead of hardcoding full access.

The design uses a separate store entry rather than modifying the proto-generated `ContractSchedule` type (which would require proto regeneration). Backwards compatibility is preserved: pre-fix schedules without stored capabilities fall back to full access.

**Resolution flow**:
1. `resolveCallerCapabilities(ctx, caller)` checks for active session key via `authKeeper.GetSessionCapabilities()`
2. If session key exists → return restricted capabilities
3. If no session key (identity/operational key) → return full capabilities
4. If no authKeeper → return full capabilities
5. Stored at creation in both `ScheduleContract` and `ScheduleExecution` handlers
6. Loaded at execution time in `ExecutePendingSchedules`

**Tests**: `TestP1_2_ScheduleCapabilityStorage`, `TestP1_2_ScheduleCapability_FullAccess`, `TestP1_2_ScheduleExecution_UsesStoredCaps`, `TestP1_2_ScheduleExecution_BackwardsCompat`

---

## Files Modified

| File | Changes |
|------|---------|
| `x/bvm/keeper/msg_server.go` | `safeExecute()`, gas bridge in `CallContract`, gas budget + capability loading in `ExecutePendingSchedules`, capability storage in `ScheduleContract` + `ScheduleExecution`, `resolveCallerCapabilities()` helper |
| `x/bvm/types/types.go` | `MaxContractGasLimit`, `MaxScheduledGasPerBlock` constants; upper-bound checks in `ValidateBasic()` |
| `x/bvm/types/keys.go` | `ScheduleCapabilityKeyPrefix`, `ScheduleCapabilityKey()` |
| `x/bvm/keeper/state.go` | `SetScheduleCapabilities()`, `GetScheduleCapabilities()` |
| `x/bvm/keeper/keeper_test.go` | 12 new tests for all 5 fixes |

## Files NOT Modified

- `x/bvm/vm/*` — VM interpreter layer (out of scope per session prompt)
- `x/bvm/module.go` — No changes needed (BeginBlock calls existing `ExecutePendingSchedules`)
- `app/gas.go` — Constants added to `x/bvm/types` instead

---

## Test Results

```
=== NEW TESTS (12 passing) ===
TestP0_1_GasBridge_CallContract         PASS  (VM gas=18 bridged to SDK gas=4149)
TestP0_1_GasBridge_OutOfGas             PASS  (out-of-gas panic on low SDK limit)
TestP0_2_ScheduledGasBudget             PASS  (budget enforcement active)
TestP0_2_EffectiveGasCap                PASS  (infinite loop capped without panic)
TestP0_3_PanicRecovery_CallContract     PASS  (INVALID opcode handled without panic)
TestP0_3_PanicRecovery_ScheduledExecution PASS (panic didn't block siblings)
TestP1_1_GasLimitCap_CallContract       PASS  (capped at MaxContractGasLimit)
TestP1_1_GasLimitCap_ScheduleContract   PASS  (capped at MaxContractGasLimit)
TestP1_2_ScheduleCapabilityStorage      PASS  (restricted caps stored)
TestP1_2_ScheduleCapability_FullAccess  PASS  (full caps when no session)
TestP1_2_ScheduleExecution_UsesStoredCaps PASS (stored caps used at execution)
TestP1_2_ScheduleExecution_BackwardsCompat PASS (pre-fix schedules get full access)

=== EXISTING TESTS (44 passing) ===
All 44 existing BVM keeper + openclaw tests continue to pass.

=== BUILD ===
go build ./...  ✓ (clean)
go vet ./x/bvm/...  ✓ (only pre-existing proto lock-copy warnings)
```

---

## Remaining P2 Items (Not in Scope)

Per `reports/BVM-CONSENSUS.md`, the following P2 items were not addressed in this session:

- **P2-1: No State Rollback on Failed Scheduled Execution** — Needs cache-wrapped context per schedule. Lower risk since the contract's state changes are only applied on `result.Success`.
- **P2-2: No Gas Schedule Governance Integration** — Gas schedule overrides exist (`SetGasSchedule`/`GetGasSchedule`) but there's no governance proposal type to modify them via LIP.
