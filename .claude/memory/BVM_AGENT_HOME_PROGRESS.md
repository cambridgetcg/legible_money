# BVM & Agent Home — Go Chain Progress Check

> Updated 2026-02-13. Go/Cosmos SDK is the sole development target. TS is reference only.

---

## BVM — 100% Complete ✅ (2026-03-22)

The Go BVM has a **production-grade opcode interpreter** with full host integrations.
**Phase 4 complete (2026-02-13)**: Inter-contract calls, contract creation, and REVERT rollback all working.
**Phase 5 complete (2026-03-22)**: LGM-20 token opcodes (0xA5-0xAC) — TOKEN_BALANCE/TRANSFER/MINT/BURN/APPROVE/ALLOWANCE/PAUSE/UNPAUSE.
**Phase 6 complete (2026-03-22)**: Real TokensKeeper wired — stubs replaced, on-chain token state live.

### What's Done

| Component | File | LOC | Tests | Status |
|-----------|------|-----|-------|--------|
| Interpreter | `x/bvm/vm/interpreter.go` | ~39K | 66 functions | **Complete** — CALL/CREATE wired |
| Opcode defs | `x/bvm/vm/opcodes.go` | ~13K | Covered | **Complete** |
| Stack + Memory | `x/bvm/vm/stack.go`, `memory.go` | ~2K | vm_test.go | **Complete** |
| 256-bit Word | `x/bvm/vm/word.go` | ~1K | Covered | **Complete** |
| Call handling | `x/bvm/vm/calls.go` | ~4K | Covered | **Complete** — ExecuteCall + address derivation |
| Snapshot/Revert | `x/bvm/vm/snapshot_statedb.go` | ~200 | 27 tests | **NEW — Phase 4** |
| Execution context | `x/bvm/vm/context.go` | ~8K | Covered | **Complete** |
| Agent host | `x/bvm/vm/host_agent.go` | ~7K | agent_test.go | **Complete** |
| Knowledge host | `x/bvm/vm/host_knowledge.go` | ~3K | knowledge_test.go | **Complete** |
| Treasury host | `x/bvm/vm/host_treasury.go` | ~10K | Covered | **Complete** |
| State: in-memory | `x/bvm/vm/state_memory.go` | — | Covered | **Complete** |
| State: Cosmos KV | `x/bvm/vm/state_cosmos.go` | — | Covered | **Complete** |
| Gas metering | `x/bvm/vm/gas.go` | — | Covered | **Complete** |
| Keeper CRUD | `x/bvm/keeper/` | ~4K | 40 functions | **Complete** |
| Deploy/Call/Schedule | `x/bvm/keeper/msg_server.go` | ~350 | Covered | **Complete** |
| gRPC queries | `x/bvm/keeper/grpc_query.go` | — | Covered | **Complete** |
| Module wiring | `x/bvm/module.go` + `app/app.go` | — | — | **Wired** |

**Total**: ~87K LOC, 106+ test functions

### Opcodes: 78+ Implemented

**EVM base layer**: Full compatibility (0x00-0xFF)
- Arithmetic, comparison, bitwise, SHA3
- Environmental, block info, stack/memory/storage
- PUSH0-32, DUP1-16, SWAP1-16, LOG0-4
- CREATE, CALL, CALLCODE, RETURN, DELEGATECALL, CREATE2, STATICCALL, REVERT

**Legible extensions**:
- Agent Identity (0xE0-0xE3): DID, home, verify, attest
- Agent Belief (0xE4-0xE7): get/set/compare belief, count
- Agent Relationship (0xE8-0xEB): trust, delegate
- Agent Autonomy (0xEC-0xEF): schedule self, cancel, modify
- Knowledge (0xB0-0xB5): query fact/domain, submit claim, vote, challenge
- Treasury (0xD0-0xD7): vesting, stake, expiring tokens, spending limits
- Guardian (0xD8-0xDB): alerts, defense
- Access (0xDC-0xDF): keys, permissions, sessions
- Introspection (0xF6-0xF9): self-balance, self-stake, self-reputation, self-accuracy

### Host Architecture

```
Interpreter
  └── prodCompositeHost (283+ delegation methods)
       ├── AgentHost      → x/auth (DID, identity)
       ├── KnowledgeHost  → x/knowledge (facts, claims, votes)
       ├── TreasuryHost   → x/bank + x/staking + x/tokens
       └── ScheduleKeeper → x/schedule adapter
```

### Msg Types

| Msg | Source | Handler |
|-----|--------|---------|
| MsgDeployContract | proto | Bytecode validation, SHA256 hash, deterministic address, code dedup |
| MsgCallContract | proto | Full interpreter execution, state persistence |
| MsgScheduleExecution | proto | Block-indexed future execution |
| MsgScheduleContract | hand-written | Method-name scheduling |
| MsgCancelSchedule | hand-written | Schedule removal |
| MsgUpdateContractState | hand-written | Manual state update |

### Completed in Phase 4 (2026-02-13)

| Item | What Was Done |
|------|---------------|
| SnapshotStateDB | Copy-on-write wrapper for REVERT rollback (snapshot_statedb.go, ~200 LOC) |
| CALL/CALLCODE | Full ExecuteCall wiring with gas forwarding, snapshot, return data copy |
| DELEGATECALL | Caller storage context preserved, static mode inheritance |
| STATICCALL | Read-only enforcement with sub-call inheritance (**fixes TS bug**) |
| CREATE | keccak256(sender\|\|nonce) address derivation, nonce tracking, code storage |
| CREATE2 | keccak256(0xff\|\|sender\|\|salt\|\|codeHash), salt-based deterministic addresses |
| GasMeter.Refund() | Post-call gas return with overflow protection |
| NonceDB/CodeWriter | Optional interfaces for StateDB nonce + code write support |
| CosmosStateDB nonce | GetNonce/SetNonce with big-endian 8-byte encoding |
| CosmosStateDB SetCode | SHA256 code dedup + contract record update |
| WordToBytes20 | 20-byte EVM address extraction from 256-bit word |
| 27 new tests | Snapshot CRUD, CALL/STATICCALL/DELEGATECALL, CREATE/CREATE2, gas |

### Remaining Work

| Item | Severity | Effort | Notes |
|------|----------|--------|-------|
| LGM-20 token opcodes (0xB6-0xBF) | MEDIUM | S | TOKEN_BALANCE through TOKEN_EXISTS — needs x/tokens wiring |
| Extended relationship opcodes (0xC0-0xC3) | LOW | S | LIST_TRUSTED, BLOCK/UNBLOCK_AGENT, REVOKE_DELEGATION |
| Gas cost production audit | LOW | M | Current costs are placeholders, need economic modeling |
| Contract upgrade mechanism | LOW | M | Proxy pattern or code hash migration |
| Real host function integration | MEDIUM | M | Replace stubs in host_knowledge.go with actual keeper calls |

---

## Agent Home — 75% Complete

Core home lifecycle, guardian, key management, sessions, spending limits, treasury automation,
aging/maturity system, memory encryption, and defense strategy enactment all working.
Missing social layer (voice, neighborhoods), multi-sig, and cross-module wiring.

### What's Done

| Component | File | Status | Notes |
|-----------|------|--------|-------|
| Proto types | `x/home/types/state.pb.go` | **Complete** | AgentHome, HomeTreasury, HomeGuardian, KeyRegistration, Alert, SpendingLimit |
| Hand-written types | `x/home/types/types.go` | **Complete** | Enums, validation, 5 statuses, 5 roles, 10 permissions, 10 msg types |
| Codec + service desc | `x/home/types/codec.go` | **Complete** | RegisterInterfaces, Msg + Query descriptors |
| Error codes | `x/home/types/errors.go` | **Complete** | 18 error types including ErrSpendingLimitExceeded |
| Expected keepers | `x/home/types/expected_keepers.go` | **Complete** | BankKeeper, TokensKeeper interfaces |
| Keeper | `x/home/keeper/keeper.go` | **Complete** | CRUD, genesis, params |
| State store | `x/home/keeper/state.go` | **Complete** | Homes, keys, sessions, alerts, spending limits (~344 LOC) |
| Msg server | `x/home/keeper/msg_server.go` | **Complete** | 10 handlers (~506 LOC) |
| Guardian logic | `x/home/keeper/guardian.go` | **Complete** | Deadman, session/alert cleanup, spending limits, treasury automation |
| Aging/maturity | `x/home/keeper/aging.go` | **Complete** | 5 stages, patina formula, reward multipliers, BeginBlocker hook |
| gRPC queries | `x/home/keeper/grpc_query.go` | **Complete** | Home, HomesByOwner, Sessions, Alerts, Params |
| Module | `x/home/module.go` | **Complete** | RegisterServices, genesis, BeginBlock (4 hooks) / EndBlock (1 hook) |
| App wiring | `app/app.go` | **Wired** | HomeKeeper field, store key, module manager |
| Tests | `x/home/keeper/keeper_test.go` | **36 tests** | Full coverage of handlers, spending limits, automation, aging |

### What Works Now

1. **Home lifecycle**: Create → active ↔ dormant ↔ guarded → archived (status transitions validated)
2. **10 msg handlers**: CreateHome, UpdateHome, UpdateMemoryCID, StartSession, EndSession, RegisterKey, RevokeKey, ConfigureGuardian, AcknowledgeAlert, SetSpendingLimit
3. **Deadman switch**: BeginBlocker scans for inactive homes, transitions to guarded, creates alert
4. **Session management**: Create with TTL, permission intersection, auto-expire in BeginBlocker
5. **Alert system**: Create, acknowledge, auto-cleanup of old acknowledged alerts in EndBlocker
6. **Key registration**: Per-home key storage with role assignment, max keys enforced, revocation cascades sessions
7. **Spending limits**: Per-key-type limits with period-based reset, enforcement (CheckAndDebitSpendingLimit), remaining capacity query
8. **Treasury automation**: Auto-compound rewards (event emission), min liquid balance alerts (bankKeeper query)
9. **Aging/maturity**: 5 stages (fresh→established→mature→venerable→ancient), reward multipliers (1.0x→1.2x), patina score (0-100), periodic BeginBlocker processing (~1hr)
10. **Memory CID**: Off-chain memory persistence via IPFS CID

### What's Missing

| Feature | Priority | Effort | TS Reference | Description |
|---------|----------|--------|-------------|-------------|
| ~~Memory encryption~~ | ~~**P1**~~ | ~~S~~ | ~~`home/memory.ts`~~ | **DONE** — AES-256-GCM encrypt/decrypt CIDs, 5 tests |
| ~~Defense strategies~~ | ~~**P2**~~ | ~~M~~ | ~~`home/guardian.ts`~~ | **DONE** — 4 strategies enacted with rate_limit, cooldown, 2FA, whitelist, blacklist, 7 tests |
| Multi-sig approval | **P2** | S | `home/access.ts` | M-of-N approval for operations above threshold |
| Full memory manager | **P2** | L | `home/memory.ts` | Beliefs, relationships, contributions, milestones, FIFO eviction |
| Agent registry | **P2** | M | `home/registry.ts` | Capabilities (7 types, 3 levels), reputation, vouch system |
| Wire Home → x/schedule | **P2** | S | — | Register guardian maintenance as scheduled process |
| Wire Home → x/knowledge | **P2** | S | — | Track contributions, record beliefs |
| Voice messaging | **P3** | M | `home/voice.ts` | Inter-home messages, inbox, auto-respond, TTL |
| Neighborhoods | **P3** | M | `home/neighborhood.ts` | Domain clusters (3-22 members), defense pools, 5% bonus |
| Comfort system | **P4** | L | `home/comfort.ts` | 10 psychological dimensions — nice-to-have |

### Msg Handlers: All 10 Done

| Handler | Status | Notes |
|---------|--------|-------|
| CreateHome | **Done** | Generates unique ID, default guardian, owner index |
| UpdateHome | **Done** | Name/status updates with valid transition check |
| UpdateMemoryCID | **Done** | IPFS CID for off-chain memory |
| StartSession | **Done** | Permission intersection, max sessions enforced |
| EndSession | **Done** | Session deletion |
| RegisterKey | **Done** | Role + permissions, max keys enforced, expiration |
| RevokeKey | **Done** | Cascade-terminates active sessions using that key |
| ConfigureGuardian | **Done** | Defense strategy, deadman config, recovery DIDs |
| AcknowledgeAlert | **Done** | Mark alert as acknowledged |
| SetSpendingLimit | **Done** | Per-key-type with period and max amount |

---

## Cross-Module Wiring

```
x/bvm (95%) ──host_agent──→ x/home (70%)
    │                            │
    ├──host_knowledge──→ x/knowledge (70%)
    │                            │
    ├──host_treasury──→ x/bank + x/staking + x/tokens
    │
    └──schedule_adapter──→ x/schedule (50%)
                               │
                          x/home guardian ──→ x/schedule (NOT WIRED)
                          x/home contrib ──→ x/knowledge (NOT WIRED)
```

| Integration | Status | What's Needed |
|-------------|--------|---------------|
| BVM → Home (agent opcodes) | **Working** | `host_agent.go` reads/writes home state |
| BVM → Knowledge | **Working** | `host_knowledge.go` queries facts, submits claims |
| BVM → Treasury | **Working** | `host_treasury.go` manages balances, stakes |
| BVM → Schedule | **Working** | Adapter bridges to x/schedule |
| Home → Auth | **Wired** | DID resolution for ownership |
| Home → Bank | **Wired** | Treasury balance ops, min liquid balance alerts |
| Home → Tokens | **Wired** | Period overlay (nil-safe) |
| Home → Schedule | **Not wired** | Guardian maintenance needs scheduled execution |
| Home → Knowledge | **Not wired** | Contribution tracking, belief recording |

---

## Recommended Build Order (Updated 2026-02-13)

### ~~Phase 1: Security & Automation~~ — DONE

1. ~~**Spending limit enforcement**~~ — **DONE**: `CheckAndDebitSpendingLimit`, `GetRemainingSpendingLimit` in guardian.go
2. ~~**Treasury automation**~~ — **DONE**: `ProcessTreasuryAutomation` (auto-compound + min balance alerts) in guardian.go
3. ~~**Aging/maturity system**~~ — **DONE**: 5 stages, patina formula, reward multipliers in aging.go
4. ~~**All msg handlers**~~ — **DONE**: 10 handlers in msg_server.go
5. ~~**Tests**~~ — **DONE**: 36 tests covering all features

### Phase 2: Remaining P1-P2 (next)

1. ~~**Memory encryption**~~ — **DONE** (2026-03-22): AES-256-GCM in keeper.go, EncryptMemoryCID/DecryptMemoryCID, 12-byte random nonce, base64(nonce||ciphertext), 5 tests
2. ~~**Defense strategies**~~ — **DONE** (2026-03-22): EnforceDefenseStrategy in guardian.go, 4 strategies (aggressive/moderate/conservative/diplomatic) with rate_limit, cooldown, require_2fa, whitelist, blacklist. Stateless checks first, stateful second. 7 tests
3. **Multi-sig** — Approval flow for operations above threshold
4. **Wire Home → Schedule** — Register guardian maintenance as scheduled process
5. **Wire Home → Knowledge** — Track contributions when claims accepted

### Phase 3: Social Layer (1-2 weeks)

6. **Agent registry** — Capabilities, reputation, vouch system
7. **Voice messaging** — Inter-home messages with inbox and TTL
8. **Neighborhoods** — Domain clusters with defense pools and reward bonuses

### ~~Phase 4: BVM Core Completion~~ — DONE (2026-02-13)

9. ~~**CALL/STATICCALL/DELEGATECALL/CALLCODE**~~ — **DONE**: Full ExecuteCall wiring with snapshot rollback
10. ~~**CREATE/CREATE2**~~ — **DONE**: keccak256 address derivation, code storage, nonce tracking
11. ~~**REVERT state rollback**~~ — **DONE**: SnapshotStateDB copy-on-write wrapper
12. ~~**STATICCALL inheritance**~~ — **DONE**: Sub-calls inherit read-only mode (verified by test)
13. ~~**BeginBlocker schedule execution**~~ — **Already worked** (ExecutePendingSchedules)
14. ~~**27 new tests**~~ — **DONE**: Full coverage of calls, snapshots, gas

### Phase 5: BVM Polish (remaining)

15. **LGM-20 opcodes** — Wire token secondary system to BVM
16. **Real host function integration** — Replace stubs with actual keeper calls
17. **Gas cost review** — Production-ready gas table
18. **Contract upgrade mechanism** — Proxy pattern or code hash migration

---

## Test Status

| Module | Tests | Status |
|--------|-------|--------|
| BVM interpreter | 66 | Complete |
| BVM keeper | 40 | Complete |
| BVM agent host | ~10 | Complete |
| BVM knowledge host | ~15 | Complete |
| **BVM calls/snapshot** | **27** | **NEW — Phase 4**: snapshot CRUD, CALL/STATICCALL/DELEGATECALL, CREATE/CREATE2, gas |
| Home keeper | **48** | **Complete** — full handler, spending limit, automation, aging, encryption, defense coverage |

### Home Test Coverage (36 tests)

**Existing** (16): CreateHome, CreateHomeDuplicate, UpdateHome, StatusTransitions, UpdateMemoryCID, StartEndSession, SessionExpiry, RegisterKey, RevokeKey, MaxKeysEnforced, ConfigureGuardian, DeadmanSwitch, AcknowledgeAlert, SetSpendingLimit, ParamsGenesis

**New — Spending Limits** (6): WithinLimit, ExceedsLimit (with alert verification), PeriodReset, NoLimit, InvalidAmount, GetRemainingSpendingLimit

**New — Treasury Automation** (3): AutoCompound (event verification), SkipsDormant, NoAutomation

**New — Aging/Maturity** (9): ComputeMaturityStage (12 boundary cases), GetRewardMultiplierBps (6 cases), ComputePatina (5 scenarios), UpdateHomeMaturity, AllStages, GetHomeRewardMultiplierBps, ProcessHomeAging_Periodic, SkipsArchived, MaturityKVStore, MaturityConstants

**New — Guardian Logic** (2): CleanupExpiredSessions, CleanupOldAlerts
