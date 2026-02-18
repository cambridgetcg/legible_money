# Session 2: BVM Consensus Integration P0 Fixes

## Context
You are working on Legible Money (`/Users/yuai/Desktop/legible_money`), a Cosmos SDK blockchain for AI agent economies. The BVM (Belief Virtual Machine) integrates with the Cosmos SDK transaction pipeline via message handlers in `x/bvm/keeper/msg_server.go`.

An audit in `reports/BVM-CONSENSUS.md` found 3 P0 and 3 P1 vulnerabilities in the consensus integration layer. Fix all P0s and the most critical P1s.

## Scope
Files to modify:
- `x/bvm/keeper/msg_server.go`
- `x/bvm/module.go`
- `app/gas.go` (gas cost entries only)
- Test files in `x/bvm/keeper/`

## P0 Fixes Required

### P0-1: No Gas Bridge — SDK Gas Disconnected from VM Gas (msg_server.go:589-590, app/ante.go:294)
Currently `CallContract` charges a flat 21,000 SDK gas via the ante handler regardless of VM gas consumed. The VM has its own independent gas meter using `msg.GasLimit`. Fix: after VM execution completes, bridge the consumed VM gas back to the SDK gas meter. Convert VM gas to SDK gas (1:1 or a configurable ratio) and consume it from the SDK context's gas meter. This ensures block gas limits actually constrain VM execution.

### P0-2: Scheduled Execution Consumes Zero SDK Gas (msg_server.go:840-955, module.go:129-133)
BeginBlock scheduled contract execution creates a fresh context and runs contracts without consuming any SDK gas from the block's gas meter. Fix: use the block's gas context (not an infinite-gas context) for scheduled executions, OR impose a hard per-schedule-execution gas cap that's deducted from the block. Also consider adding a `MaxScheduledGasPerBlock` parameter.

### P0-3: No Panic Recovery in VM Execution (msg_server.go:589-590, msg_server.go:921)
If the VM panics (e.g., from a bug in an opcode handler), the entire ABCI handler crashes. Add `defer recover()` wrapping around `interp.Execute()` calls in both `CallContract` and scheduled execution. On panic, return an error result and log the panic.

### P1 Fixes (also fix these)

### P1-1: Unbounded msg.GasLimit
`msg.GasLimit` has no upper bound — a user can set it to `2^64-1`. Add a `MaxContractGasLimit` parameter (suggest 10M) and reject transactions exceeding it in `ValidateBasic()`.

### P1-2: Scheduled Execution Privilege Escalation
Scheduled executions always get full capabilities. Scheduled executions should inherit the capabilities that were granted when the schedule was created, not full capabilities.

## Testing
- Add tests for gas bridging, panic recovery, gas limit validation
- All existing tests must continue passing: `go test ./x/bvm/... -count=1`
- Run `go build ./...` to verify no compilation errors

## Important: Do NOT modify files in `x/bvm/vm/` (the VM engine itself). Session 1 is handling those. Your scope is the keeper/msg_server layer and module.go only.

## Report
Write a detailed report to `reports/SESSION-2-BVM-CONSENSUS-P0.md` covering:
- Each fix with before/after code
- New tests added
- Test results
