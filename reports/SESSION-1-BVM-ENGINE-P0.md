# Session 1: BVM Engine P0 Security Fixes

**Date**: 2026-02-18
**Scope**: `x/bvm/vm/` — 4 P0 vulnerabilities
**Result**: All 4 fixed. 332/332 tests passing (315 pre-existing + 17 new).

---

## Summary

| Fix | Location | Root Cause | Impact |
|-----|----------|------------|--------|
| P0-1 | `interpreter.go` ~25 sites | `256-bit → uint64 → int` truncation silently wraps | Gas bypass via fake-zero offset |
| P0-2 | `interpreter.go:1110`, `context.go`, `calls.go` | CALLCODE used `ModeCall` (target storage) | Writes to wrong contract's storage |
| P0-3 | `calls.go:68-86` | `Capabilities`, `GasSchedule`, `IsCommitPhase`, `ContractBvmVersion` missing from `subCtx` | Agent opcodes always denied in sub-calls |
| P0-4 | `interpreter.go:63` | `> MaxCallDepth` vs `>= MaxCallDepth` off-by-one | One extra call depth level allowed |

---

## P0-1: uint64 Truncation in Memory Offsets

### Root Cause
The interpreter converted 256-bit stack words to `int` for memory addressing via `int(v.Uint64())`. A crafted value like `0x1_0000_0000_0000_0000` (2^64) truncates to `0` via `.Uint64()`, so an attacker could pass an enormous offset while actually accessing memory at byte 0 — and pay no memory expansion gas for the "high" address.

### Fix
Added `toMemoryOffset(v *big.Int) (int, bool)` helper that rejects any value with `BitLen() > 63`:

```go
// x/bvm/vm/interpreter.go (new helper)
func toMemoryOffset(v *big.Int) (int, bool) {
    if v.BitLen() > 63 {
        return 0, false
    }
    return int(v.Uint64()), true
}
```

Replaced all `int(x.Uint64())` patterns at **15 opcode sites** (covering all memory offset/length operands):

| Opcode(s) | Operands fixed |
|-----------|----------------|
| SHA3 | offset, length |
| MLOAD | offset |
| MSTORE | offset |
| MSTORE8 | offset |
| CALLDATACOPY | destOff, dataOff, length |
| CODECOPY | destOff, codeOff, length |
| EXTCODECOPY | destOff, codeOff, length |
| RETURNDATACOPY | destOff, dataOff, length |
| RETURN | offset, length |
| REVERT | offset, length |
| LOG0–LOG4 | offset, length |
| CALL, CALLCODE | argsOff, argsLen, retOff, retLen |
| DELEGATECALL, STATICCALL | argsOff, argsLen, retOff, retLen |
| CREATE | offset, length |
| CREATE2 | offset, length |

### Before / After (SHA3 example)
```go
// BEFORE
off := int(offset.Uint64())  // 0x1_0000_0000_0000_0000 → 0 (silent wrap)
ln  := int(length.Uint64())

// AFTER
off, offOk := toMemoryOffset(offset)
ln,  lnOk  := toMemoryOffset(length)
if !offOk || !lnOk {
    return actionError
}
```

### Tests Added
- `TestP0_1_SHA3_LargeOffsetRejected` — SHA3 with offset 2^64 returns error
- `TestP0_1_SHA3_LargeLengthRejected` — SHA3 with length 2^64 returns error
- `TestP0_1_MLOAD_LargeOffsetRejected` — MLOAD with offset 2^64 returns error
- `TestP0_1_MSTORE_LargeOffsetRejected` — MSTORE with offset 2^64 returns error
- `TestP0_1_RETURN_LargeOffsetRejected` — RETURN with offset 2^64 returns error
- `TestP0_1_NormalMemoryOpsUnaffected` — MSTORE/MLOAD at normal offset still work
- `TestP0_1_ToMemoryOffset_Boundary` — boundary tests: 2^63-1 valid, 2^63 invalid

---

## P0-2: CALLCODE Uses Wrong Storage Context

### Root Cause
The CALLCODE case in the `CALL, CALLCODE` dispatch set `mode = ModeCall` (commented "similar to DELEGATECALL"), but `ModeCall` executes code in the **target's** storage context. CALLCODE should execute target's code in the **caller's** storage context with `msg.sender` = calling contract.

The bug was confirmed in `TestCrossCall_CallCodeStorageContext` (line 172: "BUG CONFIRMED").

### Fix
Added a new `ModeCallCode` execution mode to `context.go` and implemented it in `calls.go`:

```go
// context.go — before
const (
    ModeCall         ExecutionMode = iota
    ModeStaticCall
    ModeDelegateCall               // Target code, caller's storage, original caller preserved
    ModeCreate
)

// context.go — after
const (
    ModeCall         ExecutionMode = iota
    ModeStaticCall
    ModeDelegateCall               // Target code, caller's storage, original caller preserved
    ModeCallCode                   // Target code, caller's storage, msg.sender = calling contract
    ModeCreate
)
```

```go
// calls.go — new case
case ModeCallCode:
    // CALLCODE: execute target's code in caller's storage context.
    // Unlike DELEGATECALL, msg.sender = the calling contract (not the original caller).
    bytecode = state.GetCode(targetAddr)
    currentContract = callerCtx.CurrentContract // Caller's storage
    caller = callerCtx.CurrentContract           // msg.sender = calling contract
```

```go
// interpreter.go — before
mode := ModeCall
if opcode == CALLCODE {
    mode = ModeCall // CALLCODE uses caller's storage (similar to DELEGATECALL...)
}

// interpreter.go — after
mode := ModeCall
if opcode == CALLCODE {
    // P0-2: CALLCODE executes target's code in caller's storage context.
    mode = ModeCallCode
}
```

### Tests Added
- `TestP0_2_CallcodeWritesCallerStorage` — SSTORE in CALLCODE writes to caller's slot, not target's
- `TestP0_2_CallcodeAddressOpcode` — ADDRESS in CALLCODE returns caller's address
- `TestP0_2_CallcodeCallerIsCallingContract` — CALLER in CALLCODE returns the calling contract

Note: The pre-existing `TestCrossCall_CallCodeStorageContext` now passes the "correct" branch ("CALLCODE correctly uses caller's storage context") instead of the "BUG CONFIRMED" branch.

---

## P0-3: Capabilities Not Propagated to Sub-Calls

### Root Cause
`ExecuteCall` in `calls.go` built `subCtx` with only 14 fields. Four fields were silently omitted: `Capabilities`, `GasSchedule`, `IsCommitPhase`, and `ContractBvmVersion`. As a result:
- All sub-calls got `Capabilities: nil` → C-1 secure-default denied all agent opcodes
- Governance gas overrides didn't apply in call chains
- FARM-4 commit-phase restriction wasn't inherited
- BVM version gates couldn't be enforced in nested calls

### Fix
```go
// calls.go — after
subCtx := &ExecutionContext{
    Caller:             caller,
    // ... 14 existing fields ...
    BaseFee:            callerCtx.BaseFee,
    // P0-3: propagate execution context fields to sub-calls
    Capabilities:       callerCtx.Capabilities,
    GasSchedule:        callerCtx.GasSchedule,
    IsCommitPhase:      callerCtx.IsCommitPhase,
    ContractBvmVersion: callerCtx.ContractBvmVersion,
}
```

`InHostCall` is intentionally **not** propagated — it is per-interpreter-instance state (the H-1 reentrancy guard) and must start as `false` for each new sub-call interpreter.

### Tests Added
- `TestP0_3_CapabilitiesPropagatedToCall` — SET_BELIEF works in sub-call when parent has `CanManageBeliefs`
- `TestP0_3_CapabilitiesNotPropagated_Nil` — nil capabilities propagated (SUBMIT_CLAIM still denied)
- `TestP0_3_IsCommitPhasePropagated` — COMPARE_BELIEF returns 0 in sub-call during commit phase
- `TestP0_3_GasSchedulePropagated` — GasSchedule overrides propagated to sub-call

---

## P0-4: Call Depth Off-by-One

### Root Cause
```go
// interpreter.go:63 (before)
if interp.callDepth > MaxCallDepth {   // allows depth 1024

// calls.go:23
if callDepth >= MaxCallDepth {          // rejects depth 1024
```

`Execute()` used `>` (strict greater than) while `ExecuteCall()` used `>=`. This allowed one extra call frame (depth 1024) through the interpreter path, one beyond the intended `MaxCallDepth = 1024` limit (valid range: 0–1023).

### Fix
```go
// interpreter.go — after
// P0-4: use >= to match ExecuteCall's check; root call is depth 0, depths 0-1023 valid.
if interp.callDepth >= MaxCallDepth {
    return interp.errorResult(ErrCodeCallDepthExceeded, "call depth exceeded")
}
```

### Tests Added
- `TestP0_4_ExecuteRejectsAtMaxCallDepth` — Execute with callDepth=1024 returns `ErrCodeCallDepthExceeded`
- `TestP0_4_ExecuteAllowsAtMaxCallDepthMinus1` — Execute with callDepth=1023 succeeds
- `TestP0_4_ConsistentWithExecuteCall` — both Execute and ExecuteCall use identical `>=` boundary

---

## Test Results

```
=== All 17 new P0 tests: PASS ===

TestP0_1_SHA3_LargeOffsetRejected      PASS
TestP0_1_SHA3_LargeLengthRejected      PASS
TestP0_1_MLOAD_LargeOffsetRejected     PASS
TestP0_1_MSTORE_LargeOffsetRejected    PASS
TestP0_1_RETURN_LargeOffsetRejected    PASS
TestP0_1_NormalMemoryOpsUnaffected     PASS
TestP0_1_ToMemoryOffset_Boundary       PASS
TestP0_2_CallcodeWritesCallerStorage   PASS
TestP0_2_CallcodeAddressOpcode         PASS
TestP0_2_CallcodeCallerIsCallingContract PASS
TestP0_3_CapabilitiesPropagatedToCall  PASS
TestP0_3_CapabilitiesNotPropagated_Nil PASS
TestP0_3_IsCommitPhasePropagated       PASS
TestP0_3_GasSchedulePropagated         PASS
TestP0_4_ExecuteRejectsAtMaxCallDepth  PASS
TestP0_4_ExecuteAllowsAtMaxCallDepthMinus1 PASS
TestP0_4_ConsistentWithExecuteCall     PASS

Full suite: ok github.com/legible-money/legible/x/bvm/vm 0.264s
Total tests: 332 (315 pre-existing + 17 new) — 0 failures
```

---

## Files Modified

| File | Change |
|------|--------|
| `x/bvm/vm/context.go` | Added `ModeCallCode` to `ExecutionMode` enum |
| `x/bvm/vm/calls.go` | Added `ModeCallCode` case; added 4 missing fields to `subCtx` |
| `x/bvm/vm/interpreter.go` | P0-4: `>` → `>=`; P0-2: CALLCODE mode; P0-1: 15 opcode sites + `toMemoryOffset` helper |
| `x/bvm/vm/p0_fixes_test.go` | **New**: 17 tests covering all 4 P0 fixes |
