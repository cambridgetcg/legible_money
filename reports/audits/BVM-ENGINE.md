# BVM-ENGINE: Deep Security & Correctness Audit

**Date**: 2026-02-18
**Scope**: `x/bvm/vm/` — 4,921 lines core + 10,162 lines tests (15 test files, 315 test functions)
**Test Result**: ALL PASS (0.357s)

---

## Architecture

```
                    ┌──────────────────────────────────┐
                    │        ExecutionContext           │
                    │ Caller, Origin, DID, BlockNumber  │
                    │ GasLimit, Value, CallData, Mode   │
                    │ Capabilities, GasSchedule         │
                    └───────────────┬──────────────────┘
                                    │
                    ┌───────────────▼──────────────────┐
                    │          Interpreter              │
                    │   Stack (1024) │ Memory (1MB)     │
                    │   GasMeter    │ AccessTracker     │
                    │   PC          │ JumpDests         │
                    │   callDepth   │ returnData        │
                    └───┬───────────┼──────────────────┘
                        │           │
        ┌───────────────▼───┐   ┌──▼──────────────────┐
        │  EVM Opcodes      │   │  Agent Opcodes      │
        │  (0x00-0xAF,      │   │  (0xB0-0xEF,        │
        │   0xF0-0xFF)      │   │   0xF6-0xF9)        │
        └───────┬───────────┘   └──────┬──────────────┘
                │                      │
        ┌───────▼──────────┐   ┌───────▼──────────────┐
        │    StateDB       │   │   HostFunctions      │
        │  ┌────────────┐  │   │  ┌─────────────────┐ │
        │  │ CosmosState│  │   │  │ KnowledgeHost   │ │
        │  │ MemoryState│  │   │  │ TreasuryHost    │ │
        │  │ Snapshot   │  │   │  │ AgentHost       │ │
        │  └────────────┘  │   │  └─────────────────┘ │
        └──────────────────┘   └──────────────────────┘
```

### Components

| Component | File | Lines | Purpose |
|-----------|------|-------|---------|
| **Word** | `word.go` | 69 | 256-bit word type (big.Int), signed/unsigned, Mod256 |
| **Stack** | `stack.go` | 80 | Max 1024 depth, overflow/underflow protection |
| **Memory** | `memory.go` | 143 | 1MB max, 32-byte word aligned, quadratic gas cost |
| **Opcodes** | `opcodes.go` | 542 | 142 opcodes (EVM-compat + 56 Legible extensions) |
| **Context** | `context.go` | 243 | Execution params, capabilities, interfaces |
| **Gas** | `gas.go` | 167 | GasMeter, EIP-2929 access tracking, GasSchedule |
| **Interpreter** | `interpreter.go` | 2296 | Main fetch-decode-execute loop |
| **Calls** | `calls.go` | 161 | Sub-call dispatch, CREATE/CREATE2 addresses |
| **Host Knowledge** | `host_knowledge.go` | 178 | Knowledge keeper bridge (6 RPCs) |
| **Host Treasury** | `host_treasury.go` | 312 | Treasury/Guardian/AccessControl bridge |
| **Host Agent** | `host_agent.go` | 237 | Agent identity/belief/relationship/autonomy |
| **State Cosmos** | `state_cosmos.go` | 203 | On-chain KV store backend |
| **State Memory** | `state_memory.go` | 106 | In-memory test backend |
| **Snapshot** | `snapshot_statedb.go` | 198 | Copy-on-write overlay for atomic rollback |

---

## Instruction Set (142 opcodes)

### EVM-Compatible (86 opcodes)

| Category | Range | Count | Examples |
|----------|-------|-------|---------|
| Arithmetic | 0x00-0x0B | 12 | ADD, MUL, SUB, DIV, EXP, SIGNEXTEND |
| Comparison/Bitwise | 0x10-0x1D | 14 | LT, GT, SLT, EQ, AND, OR, SHL, SAR |
| SHA3 | 0x20 | 1 | SHA3 (keccak256) |
| Environmental | 0x30-0x3F | 16 | ADDRESS, BALANCE, CALLER, CALLDATALOAD |
| Block Info | 0x40-0x48 | 9 | BLOCKHASH, TIMESTAMP, NUMBER, CHAINID |
| Stack/Memory/Storage | 0x50-0x5F | 14 | POP, MLOAD, MSTORE, SLOAD, SSTORE, JUMP |
| PUSH0-PUSH32 | 0x5F-0x7F | 33 | PUSH0 through PUSH32 |
| DUP1-DUP16 | 0x80-0x8F | 16 | Stack duplication |
| SWAP1-SWAP16 | 0x90-0x9F | 16 | Stack swap |
| LOG0-LOG4 | 0xA0-0xA4 | 5 | Event logging |
| System | 0xF0-0xFF | 10 | CREATE, CALL, RETURN, REVERT, SELFDESTRUCT |

### Legible Money Extensions (56 opcodes)

| Category | Range | Count | State-Mod? | Gas Range |
|----------|-------|-------|------------|-----------|
| Knowledge | 0xB0-0xB5 | 6 | 3 write | 100-1000 |
| Extended Relationships | 0xC0-0xC3 | 4 | 3 write | 50-500 |
| Treasury | 0xD0-0xD7 | 8 | 4 write | 10-1000 |
| Guardian | 0xD8-0xDB | 4 | 2 write | 10-300 |
| Access Control | 0xDC-0xDF | 4 | 2 write | 10-500 |
| Agent Identity | 0xE0-0xE3 | 4 | 1 write | 10-500 |
| Agent Belief | 0xE4-0xE7 | 4 | 1 write | 50-500 |
| Agent Relationship | 0xE8-0xEB | 4 | 2 write | 50-500 |
| Agent Autonomy | 0xEC-0xEF | 4 | 3 write | 50-1000 |
| Introspection | 0xF6-0xF9 | 4 | 0 read | 10-50 |

---

## Host Function Security Assessment

### KnowledgeHost (host_knowledge.go)

| Function | Keeper Method | Capability | Risk |
|----------|--------------|------------|------|
| QueryFact | GetFact | none (read) | **OK** — read-only |
| QueryDomain | GetFactsByDomain | none (read) | **OK** — read-only |
| SubmitClaim | SubmitClaimFromBVM | `submit_claims` | **OK** — capability-gated |
| CommitVote | CommitVoteFromBVM | `vote` | **OK** — capability-gated |
| RevealVote | RevealVoteFromBVM | `vote` | **OK** — capability-gated |
| InitiateChallenge | ChallengeFactFromBVM | `submit_claims` | **OK** — capability-gated |

**Notes**:
- `host_knowledge.go:57`: Confidence scaled by 1e12 for BVM precision — deterministic, no overflow risk (uint64 max = 1e6 × 1e12 = 1e18 < 2^64)
- `host_knowledge.go:97-105`: SubmitClaim hardcodes `logicZone: "empirical"` and `stakeAmount: 0` — may need future parameterization
- `host_knowledge.go:139-145`: RevealVote confidence clamped to 1,000,000 — good boundary check

### TreasuryHost (host_treasury.go)

| Function | Keeper Method | Capability | Risk |
|----------|--------------|------------|------|
| ClaimVesting | contract state | `transfer` | **P1** — reads from `agent_treasury` but no validation |
| StakeTokens | contract state | `stake` | **P2** — returns deterministic hash, no actual staking |
| UnstakeTokens | contract state | `stake` | **P2** — similar stub behavior |
| ExtendExpiring | SetStorage | `transfer` | **OK** — validates inputs |
| SetSpendLimit | SetStorage | `transfer` | **OK** — validates inputs |
| RegisterKey | VMAuthKeeper | `manage_identity` | **OK** — delegated to x/auth (H-3) |
| RevokeKey | VMAuthKeeper | `manage_identity` | **OK** — delegated to x/auth (H-3) |
| QuerySession | VMAuthKeeper or state | none (read) | **OK** — read-only |

**Notes**:
- Treasury host is partially implemented — `StakeTokens` returns a hash but doesn't actually interact with staking module
- `QueryVesting`, `QueryAlerts`, `QueryDefenses` return 0/nil — iteration not implemented
- RegisterKey/RevokeKey properly delegate to x/auth's canonical store (H-3 fix)

### AgentHost (host_agent.go)

| Function | Keeper Method | Capability | Risk |
|----------|--------------|------------|------|
| AgentDID | deterministic hash | none | **OK** |
| AgentHome | contract state read | none | **OK** |
| VerifyIdentity | contract state read | none | **OK** |
| AttestIdentity | deterministic hash | `manage_identity` | **P2** — fire-and-forget, no on-chain record |
| GetBelief | contract state read | none | **OK** |
| SetBelief | SetStorage | `manage_beliefs` | **OK** |
| CompareBelief | GetBelief + hardcoded | none | **P1** — hardcoded network confidence |
| SetTrust | SetStorage | `manage_trust` | **OK** |
| BlockAgent | SetStorage | `manage_trust` | **OK** |
| ScheduleSelf | ScheduleKeeper | `manage_schedules` | **OK** — enforces per-contract limit |
| CancelScheduleHost | ScheduleKeeper | `manage_schedules` | **P1** — empty caller auth |
| ModifySchedule | ScheduleKeeper | `manage_schedules` | **P1** — empty caller auth |

**Notes**:
- `host_agent.go:104`: `CompareBelief` hardcodes `networkConf = 0.5 * 1e18` — should query knowledge keeper
- `host_agent.go:189,198`: `CancelScheduleHost` and `ModifySchedule` pass empty `""` as caller to ScheduleKeeper — the keeper must enforce authorization internally or any contract could cancel any schedule

---

## State Management Flow

### Normal Execution

```
Tx arrives → keeper creates ExecutionContext → Interpreter.Execute()
  → Pre-scan JUMPDEST → Main loop: fetch → gas check → dispatch
  → On SSTORE: StateDB.SetStorage (direct write)
  → On success: return result with GasUsed + Logs + StateChanges
```

### Snapshot/Revert (CALL/CREATE)

```
Parent CALL:
  1. Create SnapshotStateDB wrapping base StateDB
  2. ExecuteCall() with snapshot as StateDB
  3. Sub-interpreter reads/writes through snapshot overlay
  4. On success: snapshot.Commit() → flushes to base
  5. On failure/REVERT: snapshot discarded → base untouched
```

### Nested Calls

```
Base StateDB
  └─ Snapshot L1 (CALL A→B)
       └─ Snapshot L2 (CALL B→C)
            └─ Snapshot L3 (CALL C→D)
                 - If D reverts: L3 discarded, L2 sees its own state
                 - If D succeeds: L3.Commit() → writes to L2
```

**Verified**: Nested snapshot layering works correctly (tested in `TestCrossCall_NestedSnapshotRollback`).

---

## Vulnerability Assessment

### P0 — Critical (Fix Required)

#### P0-1: uint64 truncation in memory offsets creates potential for wrap-around

**Location**: `interpreter.go:501-502`, `interpreter.go:583-603`, and 20+ similar sites

```go
off := int(offset.Uint64())  // 256-bit value truncated to uint64 then int
ln := int(length.Uint64())   // Same truncation
```

**Risk**: A 256-bit value like `0x1_0000_0000_0000_0000` truncates to `0` via `.Uint64()`, allowing attackers to bypass memory expansion gas charges while targeting valid memory. On 64-bit systems, `int(maxUint64)` wraps to `-1`, and `offset + length` can wrap around.

**Severity**: Medium-high. The 1MB memory cap in `memory.go` limits actual damage, but gas accounting is incorrect for crafted inputs. A contract could hash memory at offset=0 but pay no expansion gas if the offset overflows to 0.

**Mitigation**: Check `offset.BitLen() > 63` or `length.BitLen() > 63` before truncating. Return `actionError` (out-of-gas or invalid) for values that exceed `int` range.

#### P0-2: CALLCODE uses ModeCall — wrong storage context

**Location**: `interpreter.go:1110-1113`

```go
mode := ModeCall
if opcode == CALLCODE {
    mode = ModeCall // CALLCODE uses caller's storage (similar to DELEGATECALL...)
}
```

**Risk**: The comment says "caller's storage" but `ModeCall` uses **target's** storage. CALLCODE per EVM spec should execute the target's code but read/write the **caller's** storage. This is confirmed as a bug in `TestCrossCall_CallCodeStorageContext` (line 172: "BUG CONFIRMED").

**Severity**: Medium. CALLCODE is deprecated in favor of DELEGATECALL, but contracts relying on CALLCODE semantics will behave incorrectly. Storage writes from CALLCODE go to the wrong contract's storage.

**Fix**: Set `mode = ModeDelegateCall` for CALLCODE (with value transfer semantics, unlike DELEGATECALL).

#### P0-3: Capabilities not propagated to sub-calls

**Location**: `calls.go:68-86` — `subCtx` construction omits `Capabilities`, `GasSchedule`, `IsCommitPhase`, `InHostCall`, `ContractBvmVersion`

```go
subCtx := &ExecutionContext{
    Caller:          caller,
    // ... 14 fields copied
    BaseFee:         callerCtx.BaseFee,
    // MISSING: Capabilities, GasSchedule, IsCommitPhase, ContractBvmVersion
}
```

**Risk**: Sub-calls get `Capabilities: nil` → secure default (C-1) denies all agent opcodes. This means **legitimate contract chains cannot use agent opcodes** through CALL/DELEGATECALL. A contract A that CALL's contract B cannot have B execute SUBMIT_CLAIM even if A has the capability.

**Severity**: High (functional). The secure default is safe but breaks composability. Contract-to-contract agent opcode delegation is completely non-functional.

**Fix**: Propagate `Capabilities`, `GasSchedule`, `IsCommitPhase`, and `ContractBvmVersion` to `subCtx`. For DELEGATECALL, inherit all. For CALL/STATICCALL, policy decision needed (propagate? downscope? deny?).

#### P0-4: Call depth off-by-one between Execute and ExecuteCall

**Location**: `interpreter.go:63` vs `calls.go:23`

```go
// interpreter.go:63
if interp.callDepth > MaxCallDepth {  // allows 1024

// calls.go:23
if callDepth >= MaxCallDepth {        // rejects 1024
```

**Risk**: Execute allows `callDepth == 1024` (one more level than ExecuteCall). An attacker can create a contract that reaches depth 1024 through the interpreter path, one level beyond the intended limit.

**Severity**: Low. The practical difference is 1 extra call frame. Not exploitable for resource exhaustion since gas still limits depth.

**Fix**: Change `interpreter.go:63` from `>` to `>=`.

### P1 — Important (Should Fix)

#### P1-1: Balance not isolated in snapshots — CALL reverts don't rollback balance changes

**Location**: `snapshot_statedb.go:74-77`

```go
func (s *SnapshotStateDB) GetBalance(addr []byte) *big.Int {
    return s.base.GetBalance(addr)  // Always reads through — no overlay
}
```

**Risk**: If a host function or bank keeper operation modifies a balance during a sub-call, and that sub-call REVERTs, the balance change persists. This violates the atomicity guarantee of REVERT for balance-related operations.

**Severity**: Medium. Currently treasury host functions read from contract state (not bank), so the immediate risk is limited. But if CALL ever implements actual ETH-style value transfer via bank keeper, reverts won't rollback balance.

#### P1-2: CancelScheduleHost and ModifySchedule pass empty caller

**Location**: `host_agent.go:189,198`

```go
err := h.schedKeeper.CancelScheduleByID(id, "")  // empty caller
err := h.schedKeeper.ModifyScheduleBlock(id, "", newBlock)  // empty caller
```

**Risk**: If the ScheduleKeeper trusts the caller parameter for authorization, any contract can cancel/modify any schedule. Authorization is only safe if the keeper independently validates ownership.

#### P1-3: CompareBelief uses hardcoded network confidence

**Location**: `host_agent.go:104`

```go
networkConf := uint64(500000000000000000) // 0.5 * 1e18 default
```

**Risk**: COMPARE_BELIEF always compares against a fixed value rather than querying the actual network consensus confidence. This makes the opcode meaningless for belief comparison.

#### P1-4: GasSchedule allows governance to set any opcode to gas=0

**Location**: `gas.go:97-104`

```go
func (gs *GasSchedule) GasCostFor(opcode byte, defaultCost uint64) uint64 {
    if gs != nil && gs.Overrides != nil {
        if override, ok := gs.Overrides[opcode]; ok {
            return override  // Could be 0 — free opcode
        }
    }
```

**Risk**: A governance proposal could set SSTORE gas to 0, enabling unbounded state writes. No minimum floor enforced.

**Mitigation**: Add a minimum gas floor check: `if override < MinGasFloor { return MinGasFloor }`.

#### P1-5: EXP opcode allows unbounded big.Int computation

**Location**: `interpreter.go:305`

```go
result := new(big.Int).Exp(base, exp, WordMod)
```

**Risk**: While `big.Int.Exp` with modulus is polynomial, the gas cost charges only `50 * bytes_of_exponent`. A 32-byte exponent (max gas: 50*32=1600) can require significant CPU for modular exponentiation. The EVM also has this issue but Ethereum's gas prices were specifically tuned for it.

**Severity**: Low-medium. Gas is charged, but the ratio of gas cost to CPU work may be exploitable for CPU-bound DoS.

### P2 — Low Priority

#### P2-1: SELFDESTRUCT is a no-op that still consumes gas

**Location**: `interpreter.go:1012-1017` — Pops beneficiary, returns `actionStop`. No state destruction occurs. This is fine (matches post-EIP-6780 behavior) but documented here for completeness.

#### P2-2: Multiple host functions are stubs returning deterministic hashes

`StakeTokens`, `Delegate`, `AttestIdentity` return SHA256 hashes rather than performing real operations. This is documented as partial implementation.

#### P2-3: `itoa` function only handles 1-2 digit numbers

**Location**: `opcodes.go:536-541` — Used only for opcode mnemonic generation (1-32), so this is safe but fragile.

### OK — Verified Secure

| Class | Status | Evidence |
|-------|--------|----------|
| **Integer overflow in arithmetic** | **OK** | All arithmetic uses `big.Int` with `Mod256()` — 256-bit wraparound is EVM-correct |
| **Unbounded memory growth** | **OK** | `DefaultMaxMemorySize = 1MB` enforced in `memory.go:10`, quadratic gas cost in `memoryGasCost()` |
| **Gas exhaustion (unmetered ops)** | **OK** | All opcodes metered. Dynamic gas for SHA3, EXP, SSTORE, CALL, memory expansion. EIP-2929 cold/warm tracking |
| **Re-entrancy (host calls)** | **OK** | H-1 guard: `ctx.InHostCall` flag checked at `interpreter.go:1465`, set/cleared with defer at `interpreter.go:1488-1489` |
| **Stack depth attacks** | **OK** | `MaxCallDepth = 1024` enforced in both `Execute()` and `ExecuteCall()` (off-by-one noted in P0-4) |
| **Determinism** | **OK** | No floating point, no system time, no randomness. All operations use `big.Int`. `TIMESTAMP` and `BLOCKHASH` come from deterministic context |
| **Panic paths** | **OK** | All stack/memory operations return errors, not panics. `big.Int` operations don't panic. State access nil-guarded |
| **Jump validation** | **OK** | `scanJumpDests()` pre-scans bytecode. JUMP/JUMPI validate against the map. PUSH data bytes correctly skipped during scan |
| **Static call enforcement** | **OK** | `ModeStaticCall` checked at `interpreter.go:109` — all `IsStateModifier=true` opcodes rejected. Static mode inherited by sub-calls |
| **Capability gating** | **OK** | All state-modifying agent opcodes have `checkCapability()` gates. `nil` capabilities = deny all (C-1 secure default) |
| **FARM-4: Belief herding** | **OK** | `COMPARE_BELIEF` returns 0 during commit phase (`interpreter.go:1707`). Gas raised to 500 to deter herding |
| **Storage slot limits** | **OK** | `MaxStorageSlotsPerContract = 16384` enforced at `interpreter.go:889-894` via `StorageCounter` interface |
| **Bytecode size limit** | **OK** | `MaxBytecodeSize = 24576` (EIP-170) enforced at `interpreter.go:68` and `interpreter.go:1275` |
| **Create address derivation** | **OK** | `CreateAddress` uses keccak256(sender||nonce), `Create2Address` uses keccak256(0xff||sender||salt||keccak256(initCode)) |
| **Snapshot correctness** | **OK** | Copy-on-write overlay with proper read-through. Commit flushes storage, code, and nonces. Nested snapshots tested |
| **Deprecated opcode handling** | **OK** | `DeprecatedAtBlock > 0` opcodes become no-ops: pop inputs, push zeros (`interpreter.go:95-106`) |

---

## Test Coverage Assessment

### 15 test files, 315 test functions

| File | Tests | Lines | Coverage Area |
|------|-------|-------|---------------|
| `interpreter_test.go` | 66 | 1,189 | Core opcodes: arithmetic, comparison, bitwise, SHA3, env, block info, stack/memory/storage, PUSH/DUP/SWAP |
| `core_correctness_test.go` | 38 | 755 | Edge cases: SIGNEXTEND, SDIV, SMOD, SAR, 256-bit wraparound, CALLDATALOAD boundaries |
| `vm_test.go` | 45 | 780 | Integration: contract patterns, loops, branching, CREATE/CREATE2, REVERT |
| `calls_test.go` | 27 | 430 | Sub-calls, gas forwarding, return data, CREATE with init code |
| `crosscall_audit_test.go` | 20 | 738 | DELEGATECALL semantics, CALLCODE bug, snapshot rollback, gas refund, nested snapshots, value transfer |
| `security_audit_test.go` | 14 | 453 | Capability checks, nil-caps deny-all, reentrancy guard, static call enforcement, call depth off-by-one |
| `knowledge_audit_test.go` | 21 | 400 | Knowledge opcodes, FARM-4 commit-phase restriction, capability isolation |
| `knowledge_test.go` | 7 | 370 | Knowledge host: QueryFact, QueryDomain, SubmitClaim, CommitVote, RevealVote |
| `agent_test.go` | 15 | 680 | All agent opcodes: identity, belief, relationship, autonomy, guardian, treasury, introspection |
| `trust_dynamics_test.go` | 16 | 679 | Trust bootstrap, transitivity, betrayal/recovery, delegation chains, social exclusion, coalitions |
| `belief_formation_test.go` | 10 | 720 | Belief CRD, COMPARE_BELIEF, multi-agent belief formation |
| `money_psychology_test.go` | 12 | 690 | Treasury lifecycle, staking, vesting, spend limits, reputation, accuracy |
| `money_psychology_extended_test.go` | 6 | 441 | Extended treasury: stake races, reputation evolution, wealth/reputation quadrants |
| `autonomy_identity_test.go` | 7 | 636 | Scheduling, identity lifecycle, key management, session queries |
| `empty_work_test.go` | 11 | 770 | Host functions with empty/nil inputs, graceful degradation |

### Coverage Gaps

| Gap | Severity | Description |
|-----|----------|-------------|
| **uint64 overflow in memory offsets** | P0 | No test for `MSTORE` with offset > 2^64 — the truncation described in P0-1 is untested |
| **GasSchedule override edge cases** | P1 | No test for governance setting gas to 0 (P1-4) |
| **EXP CPU DoS** | P1 | No test measuring wall-clock time for max-size exponents |
| **Snapshot balance isolation** | Documented | Tested in `TestCrossCall_BalanceNotIsolatedInSnapshot` — confirms the limitation but doesn't fix it |
| **CosmosStateDB integration** | P2 | `state_cosmos.go` has no dedicated test file — tested indirectly through keeper tests |
| **GasSchedule propagation to sub-calls** | P1 | GasSchedule not propagated (P0-3), but no test verifies the behavior |
| **LOG gas for large data** | P2 | No test for LOG with max memory-size data to verify gas accounting |
| **AccessTracker persistence across snapshots** | P2 | AccessTracker is per-interpreter, not per-snapshot. Cold/warm state doesn't roll back on sub-call revert. This is EVM-correct but untested |

### Well-Tested Areas

- **Arithmetic correctness**: Comprehensive edge cases for all 12 arithmetic opcodes, including division by zero, 256-bit wraparound, signed operations
- **Cross-call semantics**: DELEGATECALL caller/storage preservation, STATICCALL inheritance, gas forwarding, log collection on success/discard on revert
- **Capability gating**: Every capability-gated opcode tested with both allow and deny scenarios
- **Reentrancy protection**: H-1 guard tested with direct flag setting
- **Snapshot rollback**: Storage, nonce, and code all verified to rollback on sub-call revert
- **Agent opcode lifecycle**: Full CRUD cycles for beliefs, trust, schedules

### Test Quality Notes

- `security_audit_test.go` tests are labeled "VULNERABILITY CONFIRMED" but all the vulnerabilities were **already fixed** — the tests now verify the fixes work. The labels are misleading but the assertions are correct.
- Tests use `t.Log()` extensively with findings/recommendations — good documentation practice.
- Test coverage for EVM-compatible opcodes is thorough. Coverage for Legible extensions is comprehensive at the API level but light on edge cases.

---

## Top 5 Security Recommendations

### 1. Fix uint64 truncation in memory/calldata offsets (P0-1)

Add bounds checking before `.Uint64()` truncation on all memory offset/length values:

```go
if offset.BitLen() > 63 {
    return actionError  // Offset too large for memory addressing
}
off := int(offset.Uint64())
```

This affects ~25 locations in `interpreter.go`.

### 2. Fix CALLCODE storage context (P0-2)

```go
// interpreter.go:1110-1113 — change to:
mode := ModeCall
if opcode == CALLCODE {
    mode = ModeDelegateCall  // CALLCODE = target code + caller storage
}
```

Note: CALLCODE is deprecated; consider disabling it entirely via `DeprecateOpcode(CALLCODE, 0)`.

### 3. Propagate context fields to sub-calls (P0-3)

Add missing fields to `subCtx` in `calls.go:68-86`:

```go
subCtx := &ExecutionContext{
    // ... existing fields ...
    Capabilities:       callerCtx.Capabilities,
    GasSchedule:        callerCtx.GasSchedule,
    IsCommitPhase:      callerCtx.IsCommitPhase,
    ContractBvmVersion: callerCtx.ContractBvmVersion,
}
```

Design decision needed: should Capabilities downscope through CALL? Options:
- **Propagate fully** (composable but risky)
- **Propagate for DELEGATECALL only** (matches EVM msg.sender semantics)
- **Never propagate** (current behavior — safe but breaks composability)

### 4. Add minimum gas floor to GasSchedule overrides (P1-4)

```go
const MinGasFloor uint64 = 1  // Minimum 1 gas for any opcode

func (gs *GasSchedule) GasCostFor(opcode byte, defaultCost uint64) uint64 {
    if gs != nil && gs.Overrides != nil {
        if override, ok := gs.Overrides[opcode]; ok {
            if override < MinGasFloor {
                return MinGasFloor
            }
            return override
        }
    }
    return defaultCost
}
```

### 5. Wire CancelSchedule/ModifySchedule caller authorization (P1-2)

Pass the contract address (from ctx.CurrentContract) as the caller to ScheduleKeeper methods:

```go
func (h *AgentHost) CancelScheduleHost(scheduleId []byte) bool {
    // ... existing nil check ...
    id := hex.EncodeToString(scheduleId)
    contractAddr := hex.EncodeToString(h.currentContract)  // Need to add field
    err := h.schedKeeper.CancelScheduleByID(id, contractAddr)
    return err == nil
}
```

---

## Summary

The BVM is a well-structured EVM-compatible stack machine with thoughtful Legible Money extensions. The architecture follows established patterns (EVM opcodes, big.Int arithmetic, snapshot state, gas metering) with good security defaults:

- **Capability gating** with secure nil-deny default (C-1)
- **Reentrancy protection** via InHostCall flag (H-1)
- **Static call enforcement** for all state-modifying opcodes
- **Memory and storage limits** prevent resource exhaustion
- **EIP-2929 cold/warm tracking** for realistic gas pricing

The 4 P0 issues (uint64 truncation, CALLCODE semantics, context propagation, off-by-one) are all fixable with targeted changes. The test suite is comprehensive at 315 tests covering all major subsystems.

**Overall assessment**: Production-ready for the current partial implementation scope. The P0 items should be fixed before mainnet. The partially-implemented host functions (treasury, delegation) should be completed or explicitly disabled before production use.
