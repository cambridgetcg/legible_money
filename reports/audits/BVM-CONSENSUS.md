# BVM-CONSENSUS: BVM Integration with Consensus Layer

**Date**: 2026-02-18
**Files audited**: `x/bvm/keeper/msg_server.go`, `x/bvm/keeper/keeper.go`, `x/bvm/keeper/state.go`, `x/bvm/module.go`, `x/bvm/vm/interpreter.go`, `x/bvm/vm/gas.go`, `x/bvm/vm/context.go`, `x/bvm/vm/calls.go`, `x/bvm/types/types.go`, `x/bvm/types/genesis.go`, `app/gas.go`, `app/ante.go`, `app/ante_legible.go`
**Build**: `go build ./...` — PASS
**Tests**: `go test ./x/bvm/... -count=1` — PASS (keeper + vm packages)

---

## Executive Summary

The BVM (Belief Virtual Machine) integrates with the Cosmos SDK transaction processing pipeline through 7 message handlers (3 proto + 4 hand-written) and a BeginBlock hook for scheduled execution. The audit found **3 P0 vulnerabilities**, **3 P1 issues**, and **2 P2 items**.

**Critical finding**: The BVM operates a completely independent gas meter from the SDK gas meter. A `CallContract` transaction pays only 21,000 SDK gas regardless of how much VM gas it consumes, and `msg.GasLimit` has no upper bound — a user can set it to `2^64-1` and execute arbitrarily expensive computation.

| Severity | Count | Summary |
|----------|-------|---------|
| **P0** | 3 | No gas bridge, no panic recovery, unbounded scheduled execution gas |
| **P1** | 3 | Unbounded msg.GasLimit, scheduled privilege escalation, off-by-one callDepth |
| **P2** | 2 | Silent refund failure, wasted storage from invalid bytecode |

---

## Transaction Flow Diagram

```
User submits MsgCallContract
       │
       ▼
┌─────────────────────────────────────────────────┐
│  Ante Handler Chain (app/ante.go)                │
│  1. SetUpContext (init SDK gas meter)            │
│  2. BootstrapGasFree (waive if bootstrap)        │
│  3. EmergencyHalt (block if halted)              │
│  4. LGMGasDecorator ─── charges 21,000 SDK gas   │
│  5. FeeRouter (log fee split)                    │
│  6. Standard Cosmos (sig verify, fee deduct...)  │
│  7. LegibleDID / Account / Capability            │
└─────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────┐
│  msg_server.CallContract                         │
│  1. Verify contract exists                       │
│  2. Get bytecode from code store                 │
│  3. Parse value, transfer to module account      │
│  4. Resolve caller DID, load session caps        │
│  5. Build governance gas schedule overrides       │
│  6. Create ExecutionContext with msg.GasLimit     │  ◄── VM gas meter
│  7. Create stateDB (kvStoreAdapter)              │
│  8. Build host functions (agent/treasury/know)   │
│  9. interp.Execute(bytecode, ctx, stateDB, host) │
│  10. Apply state changes on success              │
│  11. Refund value on failure                     │
└─────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────┐
│  VM Interpreter (vm/interpreter.go)              │
│  - Own GasMeter(msg.GasLimit) ◄─ INDEPENDENT    │
│  - Opcode loop with per-opcode gas costs         │
│  - EIP-2929 cold/warm access tracking            │
│  - Dynamic gas (SHA3, SSTORE, LOG, memory)       │
│  - 63/64 gas forwarding for CALL/CREATE          │
│  - Agent opcodes with capability checks          │
│  - Returns ExecutionResult{GasUsed, StateChanges}│
└─────────────────────────────────────────────────┘
       │
       ▼  (SDK gas meter unchanged — only 21K consumed)
  DeliverTx completes
```

---

## Gas Boundary Analysis

### Dual Gas System

| Layer | Meter | Source | Enforcement |
|-------|-------|--------|-------------|
| **SDK** | `sdk.Context.GasMeter()` | `ante.SetUpContextDecorator` | Block gas limit (33.3M), tx limit (11.1M) |
| **VM** | `vm.GasMeter` | `vm.NewGasMeter(ctx.GasLimit)` | Only `msg.GasLimit` field |

### P0-1: No Gas Bridge — SDK Gas is Disconnected from VM Gas

**Location**: `msg_server.go:589-590`, `app/ante.go:294`

The SDK ante handler (`LGMGasDecorator`) charges a flat **21,000 SDK gas** for `MsgCallContract` (via `TransactionGasCosts["call_contract"]`). The VM interpreter creates its own `GasMeter(msg.GasLimit)` and consumes gas independently. **The VM's gas consumption is never reported to the SDK gas meter.**

```
SDK gas consumed:   21,000 (flat, regardless of execution)
VM gas consumed:    0 to msg.GasLimit (up to 2^64-1)
Gap:                Potentially 18,446,744,073,709,530,615 gas units unmetered
```

**Attack vector**: A user submits `MsgCallContract` with `GasLimit: 2^64-1`. The SDK ante handler deducts 21,000 gas. The VM runs with a 2^64-1 gas meter. The contract could execute billions of opcodes (each costing 2-10 gas), consuming seconds or minutes of wall-clock time, while only paying for 21K SDK gas. In a block with 33M gas budget, ~1,587 such transactions could fit, each executing for potentially seconds.

**Impact**: DoS — block production stalls as validators spend excessive time executing contract calls that pay minimal gas.

**Fix required**: Either:
1. Bridge VM gas to SDK gas: `ctx.GasMeter().ConsumeGas(result.GasUsed, "bvm_execution")` after execution
2. Cap `msg.GasLimit` to `TxGasLimit - ante_gas_overhead`
3. Use the SDK gas meter directly as the VM gas meter (preferred)

### P0-2: Scheduled Execution Consumes Zero SDK Gas

**Location**: `msg_server.go:840-955`, `module.go:129-133`

`ExecutePendingSchedules` runs in `BeginBlock` — outside the ante handler chain. Scheduled executions:
- Use `schedule.MaxGas` as the VM gas limit (user-controlled via `MsgScheduleContract.MaxGas`)
- Do NOT consume any SDK gas (no `ctx.GasMeter()` calls)
- Have no per-block execution budget

For `ScheduleContract` (hand-written), `msg.MaxGas` is user-provided with no upper bound check (only `MaxGas == 0` is rejected in `ValidateBasic`).

For `ScheduleExecution` (proto), `MaxGas` is set to `params.CallBaseGasCost` (21,000) — bounded but still unmetered.

**Attack vector**: Deploy a contract, schedule 100 executions (the limit per contract) each with `MaxGas: 2^64-1`. At the target block, BeginBlock spends unbounded time executing all 100 schedules. Validators might fail to produce a block within the block time (2521ms).

**Fix required**: Add per-block gas budget for scheduled executions in `ExecutePendingSchedules`:
```go
blockGasBudget := uint64(5_000_000) // e.g., 15% of block gas limit
for _, schedule := range schedules {
    if totalGasUsed + schedule.MaxGas > blockGasBudget {
        break // defer remaining to next block
    }
    // ... execute ...
    totalGasUsed += result.GasUsed
}
```

### P0-3: No Panic Recovery in VM Execution

**Location**: `msg_server.go:589-590` (CallContract), `msg_server.go:921` (scheduled)

Neither `CallContract` nor `ExecutePendingSchedules` has a `defer recover()` around `interp.Execute()`. If the interpreter panics (e.g., from an edge case in state access, stack operations, or host functions), the panic propagates up to:
- `DeliverTx` for CallContract → Cosmos SDK catches this, but the tx state is undefined
- `BeginBlock` for scheduled execution → **chain halts for ALL validators** (no recovery)

```go
// Current (DANGEROUS):
result := interp.Execute(code.Bytecode, execCtx, stateDB, host)

// Required:
var result *vm.ExecutionResult
func() {
    defer func() {
        if r := recover(); r != nil {
            result = &vm.ExecutionResult{Success: false, Error: ...}
        }
    }()
    result = interp.Execute(code.Bytecode, execCtx, stateDB, host)
}()
```

The keeper `state.go` uses `panic()` for KV store errors (32 locations). If a KV store operation fails during VM execution's host function calls, it panics. In `BeginBlock`, this is chain-halting.

---

## Contract Lifecycle State Machine

```
                   DeployContract
                        │
                        ▼
┌──────────────────────────────────────────┐
│  DEPLOYED                                 │
│  - Address: sha256(deployer/block/nonce)  │
│  - Code: sha256(bytecode) → dedup        │
│  - Creator: msg.Deployer                  │
│  - BvmVersion: params.CurrentBvmVersion   │
└──────────────────────────────────────────┘
        │                          │
        │ CallContract             │ ScheduleContract / ScheduleExecution
        ▼                          ▼
┌──────────────┐    ┌─────────────────────────┐
│  EXECUTING    │    │  SCHEDULED               │
│  - VM runs    │    │  - ExecuteAtBlock: N     │
│  - Gas meter  │    │  - MaxGas: user/params   │
│  - State DB   │    │  - Cancelled: false      │
│  - Host funcs │    │  - Executed: false        │
└──────────────┘    └─────────────────────────┘
   │        │              │           │
   │ success│ failure      │ Block N   │ CancelSchedule
   ▼        ▼              ▼           ▼
┌─────┐  ┌─────┐   ┌──────────┐  ┌───────────┐
│STATE│  │REVERT│   │ EXECUTING│  │ CANCELLED  │
│SAVED│  │+refund│  │(BeginBlk)│  │           │
└─────┘  └─────┘   └──────────┘  └───────────┘
                       │     │
                    success  failure
                       ▼     ▼
                   ┌─────┐ ┌──────────┐
                   │STATE│ │EXECUTED   │
                   │SAVED│ │(no state) │
                   └─────┘ └──────────┘
```

### Deploy
1. **ValidateBasic**: deployer is valid bech32, bytecode non-empty
2. **Handler**: check `len(bytecode) <= params.MaxBytecodeSize` (default 1MB)
3. **Code dedup**: SHA-256 hash → if code exists, increment `RefCount`
4. **Address**: `sha256(deployer/blockHeight/nonce)` → truncate to 20 bytes → `lgm1contract{hex}`
5. **Store**: contract metadata + code (if new) + increment nonce counter

### Call
1. **ValidateBasic**: caller valid, contract non-empty, GasLimit > 0
2. **Handler**: verify contract exists, load bytecode
3. **Value transfer**: if `msg.Value > 0`, transfer from caller to BVM module account
4. **DID resolution**: resolve caller's DID from x/auth (for agent opcodes)
5. **Capability loading**: session key → restricted caps; identity key → full caps; no DID → nil (deny all)
6. **VM execution**: build context, stateDB, host functions; `interp.Execute()`
7. **State commit**: on success, iterate `result.StateChanges`, write via `SetContractState`
8. **Refund**: on failure with value > 0, transfer back from module to caller

### Schedule
1. **Target block**: `BlockInterval.StartHeight`, `TimeCondition → next block`, `StateCondition → next block`
2. **Limits**: `MaxSchedulesPerContract` (default 100)
3. **Execution**: `BeginBlock` → `ExecutePendingSchedules` → iterate block index → execute each
4. **Capabilities**: Scheduled executions always get **full capabilities** (P1 issue, see below)

---

## Schedule Execution Flow

```
BeginBlock (module.go:129-133)
    │
    ▼
ExecutePendingSchedules(ctx) (msg_server.go:840-955)
    │
    ├─ GetPendingSchedules(ctx, currentBlock)
    │     └─ Iterate: ScheduleBlockIndexPrefix + [0x00..00] → [currentBlock+1]
    │
    ├─ For each schedule:
    │     ├─ Delete block index entry
    │     ├─ Load contract and bytecode
    │     ├─ Build caller (bech32 → 20 bytes)
    │     ├─ Resolve caller DID
    │     ├─ Set fullCaps (all true) ◄── P1: privilege escalation
    │     ├─ Build gas schedule overrides
    │     ├─ Create ExecutionContext(GasLimit: schedule.MaxGas)
    │     ├─ Create stateDB + host functions
    │     ├─ interp.Execute() ◄── NO panic recovery (P0)
    │     ├─ Apply state changes on success
    │     ├─ Mark schedule.Executed = true
    │     └─ Emit event
    │
    └─ (no gas budget enforcement)
```

---

## Handler Analysis

### DeployContract (msg_server.go:399-453)

| Aspect | Status |
|--------|--------|
| Validation | Size check only (`MaxBytecodeSize`). No bytecode structural validation. |
| Gas | 200,000 SDK gas (ante). No VM execution. |
| State changes | SetCode, SetContract, increment nonce counter. |
| Panic recovery | None needed (no VM execution). |
| Auth | Deployer must sign. No further restrictions. |

### CallContract (msg_server.go:456-631)

| Aspect | Status |
|--------|--------|
| Validation | Contract exists, code exists. GasLimit > 0 (ValidateBasic). **No GasLimit upper bound.** |
| Gas | 21,000 SDK gas (ante). VM runs independently. **No bridge.** |
| State changes | On success: iterate StateChanges, SetContractState. On failure: refund value. |
| Panic recovery | **NONE — P0.** |
| Auth | Caller must sign. Session key caps enforced via VM context. |
| Value handling | Transfer to module account before execution, refund on failure. |

### ScheduleExecution (msg_server.go:633-709)

| Aspect | Status |
|--------|--------|
| Validation | Contract exists, scheduler == creator. Schedule in future. Count < limit. |
| Gas | 40,000 SDK gas (ante). MaxGas = `params.CallBaseGasCost` (21,000). |
| State changes | SetSchedule + block index. |
| Auth | Only contract creator can schedule. |

### ScheduleContract (msg_server.go:711-767)

| Aspect | Status |
|--------|--------|
| Validation | Contract exists, caller == creator. ExecuteAtBlock in future. MaxGas > 0 (ValidateBasic). **No MaxGas upper bound.** |
| Gas | 40,000 SDK gas (ante). MaxGas is user-provided. |
| Auth | Only contract creator. |

### CancelSchedule (msg_server.go:769-809)

| Aspect | Status |
|--------|--------|
| Validation | Schedule exists, not executed, not cancelled, caller == owner. |
| Gas | 30,000 SDK gas (ante). No VM execution. |
| State changes | Delete block index, set Cancelled=true. |

### UpdateContractState (msg_server.go:811-836)

| Aspect | Status |
|--------|--------|
| Validation | Authority check (governance only). Contract exists. |
| Gas | 40,000 SDK gas (ante). No VM execution. |

### UpdateParams (msg_server.go:958-972)

| Aspect | Status |
|--------|--------|
| Validation | Authority check (governance only). |
| Gas | Governance-gated. |

---

## Determinism Audit

| Check | Result | Evidence |
|-------|--------|----------|
| **Floating point** | PASS — none found | `grep float32\|float64 x/bvm/vm/` → no matches |
| **math/big arithmetic** | PASS — all integer | All arithmetic uses `*big.Int` with `Mod256()` for 256-bit word semantics |
| **Random values** | PASS — none | No `math/rand`, `crypto/rand`, or `time.Now()` in VM |
| **Map iteration** | PASS (borderline) | `msg_server.go:544`: `range govSchedule` iterates `map[string]uint64` but only builds a lookup table `map[byte]uint64`. Iteration order doesn't affect result. |
| **Contract addresses** | PASS — deterministic | `sha256(deployer/blockHeight/nonce)` — all inputs deterministic |
| **Schedule IDs** | PASS — deterministic | `sched-{blockHeight}-{counter}` — counter is store-backed |
| **Timestamps** | PASS — from context | `ctx.BlockTime().Unix()` — CometBFT consensus timestamp |
| **Block numbers** | PASS — from context | `ctx.BlockHeight()` — consensus block height |
| **Hash functions** | PASS — SHA-256, Keccak-256 | No non-deterministic hash functions |
| **Code dedup** | PASS — SHA-256 hash key | Same bytecode always produces same code hash |

**Verdict**: The VM execution path is **deterministic**. All operations use integer arithmetic, deterministic hashing, and consensus-derived inputs. No floating point, randomness, or non-deterministic iteration in consensus-critical paths.

---

## P0 Findings (Consensus/Security Critical)

### P0-1: No Gas Bridge Between SDK and VM

**Files**: `msg_server.go:589-590`, `app/ante.go:294`

The SDK charges a flat 21,000 gas for `CallContract` via the ante handler. The VM creates an independent `GasMeter(msg.GasLimit)`. VM gas consumed is never reflected in the SDK gas meter.

**Exploit**: Set `msg.GasLimit = 2^64-1`, pay 21K SDK gas, execute billions of opcodes.

**Fix**: After `interp.Execute()`, consume VM gas on the SDK meter:
```go
result := interp.Execute(...)
ctx.GasMeter().ConsumeGas(result.GasUsed, "bvm_execution")
```
And cap `msg.GasLimit` in `ValidateBasic`:
```go
if m.GasLimit > 11_000_000 { // TxGasLimit minus overhead
    return fmt.Errorf("gas limit too high")
}
```

### P0-2: No Panic Recovery in VM Execution

**Files**: `msg_server.go:590` (CallContract), `msg_server.go:921` (scheduled)

The `interp.Execute()` call has no `defer recover()`. A panic from a VM edge case, host function error, or KV store failure in `keeper/state.go` (which uses `panic()` for all store errors) propagates uncaught.

- In `CallContract` (DeliverTx): Cosmos SDK may partially catch it, but state could be corrupted.
- In `ExecutePendingSchedules` (BeginBlock): **Chain-halting.** All validators panic on the same block.

**Fix**: Wrap both execution sites in panic recovery:
```go
func (k Keeper) safeExecute(interp *vm.Interpreter, ...) *vm.ExecutionResult {
    var result *vm.ExecutionResult
    func() {
        defer func() {
            if r := recover(); r != nil {
                result = &vm.ExecutionResult{Success: false, GasUsed: ctx.GasLimit}
            }
        }()
        result = interp.Execute(bytecode, execCtx, stateDB, host)
    }()
    return result
}
```

### P0-3: Unbounded Scheduled Execution Gas in BeginBlock

**File**: `msg_server.go:840-955`

`ExecutePendingSchedules` executes ALL pending schedules for the current block with no aggregate gas budget. Each schedule uses its own VM gas meter but there's no block-level cap. With 100 schedules per contract and many contracts, BeginBlock could run for minutes.

**Fix**: Add a per-block gas budget:
```go
const ScheduleBlockGasBudget = 5_000_000
var totalGasUsed uint64
for _, schedule := range schedules {
    if schedule.MaxGas > ScheduleBlockGasBudget - totalGasUsed {
        schedule.MaxGas = ScheduleBlockGasBudget - totalGasUsed
    }
    if totalGasUsed >= ScheduleBlockGasBudget {
        break // defer to next block
    }
    // ... execute ...
    totalGasUsed += result.GasUsed
}
```

---

## P1 Findings

### P1-1: No Upper Bound on msg.GasLimit

**File**: `x/bvm/types/types.go:49`

`MsgCallContract.ValidateBasic()` only checks `m.GasLimit == 0` — no upper bound. A user can set `GasLimit: 18446744073709551615` (max uint64).

Similarly, `MsgScheduleContract.ValidateBasic()` (types.go:109) only checks `m.MaxGas == 0`.

**Fix**: Add upper bound checks:
```go
if m.GasLimit > 11_000_000 { // TxGasLimit
    return fmt.Errorf("gas limit exceeds maximum")
}
```

### P1-2: Scheduled Execution Privilege Escalation

**File**: `msg_server.go:877-882`

Scheduled executions always receive `fullCaps` (all capabilities true), regardless of the original caller's session key restrictions. A session key with only `CanVote` could schedule a contract execution that runs with `CanTransfer`, `CanStake`, `CanSubmitClaims`, etc.

**Mitigated by**: Only the contract creator can schedule (msg_server.go:724). But the creator might use a restricted session key.

**Fix**: Store the caller's capabilities at schedule creation time and use them during execution:
```go
schedule := types.ContractSchedule{
    // ... existing fields ...
    CallerCapabilities: encodeCaps(currentSessionCaps),
}
```

### P1-3: Off-by-One in Call Depth Check

**Files**: `vm/interpreter.go:63`, `vm/calls.go:23`

```go
// interpreter.go:63 — allows callDepth == MaxCallDepth (1024)
if interp.callDepth > MaxCallDepth { ... }

// calls.go:23 — rejects callDepth >= MaxCallDepth (1024)
if callDepth >= MaxCallDepth { ... }
```

The entry-point check uses `>` while the recursive check uses `>=`. This means `interp.callDepth = 1024` passes the interpreter check but would fail if it tried to make a sub-call. Not directly exploitable (confirmed by security_audit_test.go), but inconsistent.

**Fix**: Align to `>=` in interpreter.go:63:
```go
if interp.callDepth >= MaxCallDepth {
```

---

## P2 Findings

### P2-1: Silent Refund Failure

**File**: `msg_server.go:610`

```go
_ = m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, callerAddr, refundCoins)
```

The refund error is silently ignored. If the module account lacks funds (e.g., due to a prior bug), the caller permanently loses their value transfer with no error reported.

**Fix**: Return the error:
```go
if err := m.bankKeeper.SendCoinsFromModuleToAccount(...); err != nil {
    return nil, fmt.Errorf("refund failed: %w", err)
}
```

### P2-2: No Bytecode Structural Validation on Deploy

**File**: `msg_server.go:399-453`

`DeployContract` validates bytecode size only. Invalid opcodes, malformed PUSH data, or entirely random bytes are accepted and stored. They fail at execution time, but waste permanent on-chain storage.

**Fix** (optional): Add a bytecode pre-scan:
```go
if err := vm.ValidateBytecode(msg.Bytecode); err != nil {
    return nil, fmt.Errorf("invalid bytecode: %w", err)
}
```

---

## Gas Cost Table (SDK Layer)

| Message | Gas Key | SDK Gas Cost |
|---------|---------|-------------|
| `MsgDeployContract` | `deploy_contract` | 200,000 |
| `MsgCallContract` | `call_contract` | **21,000** (P0: too low without gas bridge) |
| `MsgScheduleExecution` | `schedule_execution` | 40,000 |
| `MsgScheduleContract` | `schedule_contract` | 40,000 |
| `MsgCancelSchedule` | `cancel_bvm_schedule` | 30,000 |
| `MsgUpdateContractState` | `update_contract_state` | 40,000 |

## VM Gas Cost Table (Internal)

| Tier | Gas | Opcodes |
|------|-----|---------|
| Zero | 0 | STOP, JUMPDEST |
| Base (2) | 2 | ADDRESS, ORIGIN, CALLER, CALLVALUE, CALLDATASIZE, CODESIZE, GASPRICE |
| VeryLow (3) | 3 | ADD, SUB, LT, GT, SLT, SGT, EQ, ISZERO, AND, OR, XOR, NOT, BYTE, SHL, SHR, POP, MLOAD, MSTORE, PUSH0-PUSH32 |
| Low (5) | 5 | MUL, DIV, SDIV, MOD, SMOD |
| Mid (8) | 8 | ADDMOD, MULMOD, SIGNEXTEND |
| High (10) | 10 | EXP (+ 50/byte dynamic) |
| SHA3 | 30 + 6/word | KECCAK256 |
| SLOAD | 2100 (cold) / 100 (warm) | EIP-2929 |
| SSTORE | 20000 (set) / 5000 (reset/clear) | EIP-2200 |
| LOG | 375 + 375/topic + 8/byte | LOG0-LOG4 |
| CALL | 700 + 63/64 forwarding | CALL, CALLCODE, DELEGATECALL, STATICCALL |
| CREATE | 32000 | CREATE, CREATE2 |
| Agent opcodes | Per OpcodeTable entry | 0xB0-0xF9 (governance-adjustable via GasSchedule) |

---

## Security Mechanisms Already Present

| Mechanism | Location | Description |
|-----------|----------|-------------|
| Call depth limit | `vm/context.go:234` | `MaxCallDepth = 1024` |
| Bytecode size limit | `vm/context.go:241` | `MaxBytecodeSize = 24576` (EIP-170) |
| Storage slot limit | `vm/context.go:238` | `MaxStorageSlotsPerContract = 16384` |
| EIP-2929 cold/warm | `vm/gas.go:107-166` | Tracks address/storage access, charges cold gas on first access |
| 63/64 gas forwarding | `vm/calls.go:94-101` | EIP-150 gas cap on sub-calls |
| Static call restriction | `vm/interpreter.go:109-111` | Rejects state-modifying opcodes in STATICCALL |
| Reentrancy guard | `vm/interpreter.go:1465-1489` | `ctx.InHostCall` prevents agent opcode → host → agent opcode |
| Session key caps | `vm/interpreter.go:1417-1443` | `checkCapability()` with nil=deny default (C-1 fix) |
| Schedule limits | `types/genesis.go:10` | `MaxSchedulesPerContract = 100` |
| Creator-only scheduling | `msg_server.go:647,724` | Only contract creator can schedule |
| Governance gas overrides | `vm/gas.go:88-104` | Per-opcode gas adjustable via governance |
| BVM version gating | `vm/interpreter.go:85-86` | Opcodes require minimum BVM version |
| Opcode activation blocks | `vm/interpreter.go:90-91` | Opcodes can be scheduled for future activation |
| Deprecation handling | `vm/interpreter.go:95-106` | Deprecated opcodes become no-ops |
| FARM-4 anti-herding | `vm/interpreter.go:1707-1708` | COMPARE_BELIEF returns 0 during commit phase |

---

## Build + Test Output

```
$ go build ./...
(exit 0, no errors)

$ go test ./x/bvm/... -count=1
ok  github.com/legible-money/legible/x/bvm/keeper   0.880s
ok  github.com/legible-money/legible/x/bvm/vm        0.448s
```

---

## Recommended Fix Priority

| ID | Severity | Effort | Fix |
|----|----------|--------|-----|
| P0-1 | Critical | Medium | Add SDK gas bridge + cap msg.GasLimit |
| P0-2 | Critical | Low | Add `defer recover()` around both Execute calls |
| P0-3 | Critical | Medium | Add per-block gas budget for scheduled execution |
| P1-1 | High | Low | Add GasLimit/MaxGas upper bounds in ValidateBasic |
| P1-2 | High | Medium | Store+restore caller capabilities in schedules |
| P1-3 | Medium | Low | Change `>` to `>=` in interpreter.go:63 |
| P2-1 | Low | Low | Return refund error instead of ignoring |
| P2-2 | Low | Medium | Optional bytecode pre-scan |
