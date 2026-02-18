# R2 Session 3: BVM Gas Bridge — Integration Testing + Scheduled Execution

## Context
Round 1 Sessions 1+2 fixed the BVM's gas disconnect. Session 1 fixed the VM engine internals (memory overflow, CALLCODE, capabilities, call depth). Session 2 bridged VM gas to SDK gas, added panic recovery, capped gas limits, and fixed scheduled execution capabilities.

The aha moment: **the gas bridge and scheduled execution budget need integration testing** to verify they work correctly under realistic conditions, not just unit tests.

## Scope
- `x/bvm/keeper/msg_server.go`
- `x/bvm/keeper/keeper_test.go` or new integration test file
- `x/bvm/vm/` (read-only — don't modify, verify behavior)

## Tasks

### 1. Gas Bridge Integration Tests
Write tests that verify the SDK gas meter correctly reflects VM gas consumption:
- `TestGasBridge_SimpleContract` — Deploy and call a simple contract, verify SDK gas consumed ≥ VM gas consumed
- `TestGasBridge_OutOfGas_SDKLimit` — Set SDK gas limit lower than VM will consume → transaction reverts atomically (value transfer rolled back)
- `TestGasBridge_OutOfGas_VMLimit` — Set VM gas limit lower than needed → VM returns out of gas, SDK reflects it
- `TestGasBridge_NestedCalls` — Contract A calls contract B calls contract C → total VM gas correctly bridged

### 2. Scheduled Execution Budget Tests
Write tests that verify the per-block gas budget for scheduled execution:
- `TestScheduleBudget_SingleSchedule_FitsInBudget` — One schedule within budget → executes
- `TestScheduleBudget_MultipleSchedules_ExceedsBudget` — 3 schedules totaling > MaxScheduledGasPerBlock → first 2 execute, third deferred
- `TestScheduleBudget_DeferredScheduleRunsNextBlock` — Verify deferred schedule runs in next block's BeginBlock
- `TestScheduleBudget_PanicInSchedule_DoesntBlockOthers` — Schedule 1 panics → schedule 2 still executes

### 3. Capability Inheritance Tests
Write tests that verify session key capabilities are correctly stored and loaded for scheduled execution:
- `TestScheduleCapability_RestrictedKey_RestrictedExecution` — Schedule created with CanTransfer-only key → execution only has CanTransfer
- `TestScheduleCapability_FullKey_FullExecution` — Schedule created with operational key → execution has full capabilities
- `TestScheduleCapability_CapabilityExpires` — If the session key that created the schedule expires, does the stored capability still work? (It should — capabilities are snapshotted at creation)

### 4. Panic Recovery Stress Test
- `TestPanicRecovery_INVALID_Opcode` — Contract with INVALID opcode → recovered, error returned
- `TestPanicRecovery_StackOverflow` — Deeply recursive contract → recovered
- `TestPanicRecovery_MultipleSchedules_OnePanics` — Mix of valid and panicking schedules → valid ones complete

## Report
Write to `reports/R2-SESSION-3-BVM-GAS-BRIDGE.md`
