# Session 1: BVM Engine P0 Security Fixes

## Context
You are working on Legible Money (`/Users/yuai/Desktop/legible_money`), a Cosmos SDK blockchain for AI agent economies. The BVM (Belief Virtual Machine) is a custom EVM-like VM in `x/bvm/vm/`.

An audit in `reports/BVM-ENGINE.md` found 4 P0 vulnerabilities in the VM engine. Fix all 4.

## Scope
Files to modify:
- `x/bvm/vm/interpreter.go`
- `x/bvm/vm/calls.go`
- Test files in `x/bvm/vm/`

## P0 Fixes Required

### P0-1: uint64 truncation in memory offsets (interpreter.go:501-502, 583-603, ~20 sites)
Memory offset + size calculations use `uint64` and can silently wrap around. Add overflow checks before any memory offset arithmetic. If `offset + size` would overflow uint64, return an out-of-gas or invalid memory access error instead of wrapping.

### P0-2: CALLCODE uses ModeCall — wrong storage context (interpreter.go:1110-1113)
CALLCODE should execute the called contract's code but use the CALLER's storage. Currently it uses `ModeCall` which writes to the callee's storage. Fix by using the correct storage context — CALLCODE should behave like DELEGATECALL but with `msg.sender` set to the caller. Check how DELEGATECALL is implemented and mirror the storage behavior for CALLCODE.

### P0-3: Capabilities not propagated to sub-calls (calls.go:68-86)
When creating `subCtx` for sub-calls (CALL, CALLCODE, DELEGATECALL, STATICCALL), the following fields are omitted: `Capabilities`, `GasSchedule`, `IsCommitPhase`, `InHostCall`, `ContractBvmVersion`. Copy all of these from the parent context into the sub-call context.

### P0-4: Call depth off-by-one (interpreter.go:63 vs calls.go:23)
`Execute()` checks `callDepth >= MaxCallDepth` and `ExecuteCall()` also checks the same condition, but they increment at different points. Ensure consistent call depth tracking: the root call should be depth 0, sub-calls increment before the check, and `MaxCallDepth = 1024` means depths 0-1023 are valid (1024 is rejected).

## Testing
- Add tests for each P0 fix
- All existing tests must continue passing: `go test ./x/bvm/vm/... -count=1`
- Run `go build ./...` to verify no compilation errors

## Report
Write a detailed report to `reports/SESSION-1-BVM-ENGINE-P0.md` covering:
- Each fix with before/after code
- New tests added
- Test results
