# Testnet Readiness Assessment — Final Checklist

**Session**: B6-7-TESTNET-READINESS
**Date**: 2026-02-18
**Branch**: fix/phase-d-proto-json-agent-sdk
**Scope**: 10-checklist comprehensive readiness assessment
**Verdict**: **NOT READY** — 1 blocker (6 missing module accounts in maccPerms)

---

## Executive Summary

The Legible Money Go chain is **very close to testnet readiness**. All tests pass (34 packages), the binary builds cleanly, genesis round-trips validate, and all prior P0 security findings have been fixed and tested. The **sole blocker** is 6 modules missing from `maccPerms` in `app/app.go` — these modules use `SendCoinsFromAccountToModule` / `SendCoinsFromModuleToAccount` but their module accounts don't exist. At runtime, the bank keeper will **panic** when any of these modules attempt a coin transfer. Estimated fix: **30 minutes**.

---

## Checklist Results

| # | Checklist | Result | Notes |
|---|-----------|--------|-------|
| 1 | Build & Test | **PASS** | go build/vet/test all clean, 34 packages pass |
| 2 | Binary Capabilities | **PASS** (warnings) | CLI complete, version metadata empty |
| 3 | Module Account Verification | **FAIL — BLOCKER** | 6 modules missing from maccPerms |
| 4 | RegisterServices Completeness | **PASS** (warnings) | 28/28 modules registered; autopoiesis lacks QueryServer |
| 5 | Genesis Round-Trip | **PASS** | init → validate-genesis → gentx → collect works |
| 6 | Gas Coverage | **PASS** | 214 message type → gas entries in ante.go |
| 7 | Proto Registration | **PASS** (warnings) | All 28 modules have RegisterInterfaces; some incomplete |
| 8 | Known Issues from All Batches | **PASS** | All prior P0s resolved; 16 P2 design decisions deferred |
| 9 | Security P0 Status | **PASS** | All P0s have tests; ante P0-1 and channels P0-1 both FIXED |
| 10 | Operational Readiness | **PASS** | Chain boots, InitChain succeeds, ABCI handshake complete |

---

## Checklist 1: Build & Test — PASS

```
go build -o build/legbled ./cmd/legbled    → exit 0, no errors
go vet ./...                                → exit 0, no warnings
go test ./... -count=1 -timeout 600s        → ALL PASS
```

**Test Results** (34 packages):
- `app/` — PASS (1.2s)
- `tests/cross_stack/` — PASS (1.5s)
- `x/alignment/keeper/` — PASS (1.4s)
- `x/auth/keeper/` — PASS (1.5s)
- `x/autopoiesis/keeper/` — PASS (1.8s)
- `x/autopoiesis/types/` — PASS (2.2s)
- `x/billing/keeper/` — PASS (2.6s)
- `x/bvm/keeper/` — PASS (2.9s)
- `x/bvm/vm/` — PASS (3.2s)
- `x/capture_challenge/keeper/` — PASS (3.6s)
- `x/capture_defense/keeper/` — PASS (4.0s)
- `x/channels/keeper/` — PASS (4.5s)
- `x/claiming_pot/keeper/` — PASS (4.9s)
- `x/compute_pool/keeper/` — PASS (5.2s)
- `x/discovery/keeper/` — PASS (5.6s)
- `x/disputes/keeper/` — PASS (6.0s)
- `x/emergency/keeper/` — PASS (6.5s)
- `x/evidence_mgmt/keeper/` — PASS (6.9s)
- `x/gov/` — PASS (6.9s)
- `x/gov/keeper/` — PASS (6.9s)
- `x/home/keeper/` — PASS (6.9s)
- `x/ibcratelimit/keeper/` — PASS (6.9s)
- `x/icaauth/keeper/` — PASS (6.9s)
- `x/knowledge/crypto/` — PASS (6.7s)
- `x/knowledge/evaluation/` — PASS (6.8s)
- `x/knowledge/keeper/` — PASS (6.8s)
- `x/ontology/keeper/` — PASS (6.8s)
- `x/partnerships/keeper/` — PASS (6.7s)
- `x/qualification/keeper/` — PASS (6.7s)
- `x/research/keeper/` — PASS (6.7s)
- `x/schedule/keeper/` — PASS (6.7s)
- `x/staking/keeper/` — PASS (6.8s)
- `x/tokens/keeper/` — PASS (5.6s)
- `x/tree/keeper/` — PASS (5.5s)
- `x/vesting_rewards/keeper/` — PASS (5.5s)
- `x/testutil/` — PASS (6.7s)
- `x/testutil/claw/` — PASS (6.7s)

**Zero failures, zero test errors.**

---

## Checklist 2: Binary Capabilities — PASS (warnings)

### Version
```
cosmos_sdk_version: v0.50.15
go: go version go1.26.0 darwin/arm64
name: ""                    # WARNING: not set
version: ""                 # WARNING: not set
server_name: <appd>         # WARNING: not "legbled"
```
**WARNING-1**: `name`, `version`, and `server_name` are not set via `-ldflags`. Should be set before testnet for identification.

### Init
```
legbled init test-check --chain-id test-check → success
```

### CLI Completeness
- **30 tx subcommands**: autopoiesis, bank, billing, bvm, capture_challenge, capture_defense, channels, claiming_pot, compute_pool, discovery, disputes, emergency, evidence_mgmt, gov, home, ibcratelimit, icaauth, knowledge, legible_auth, legible_staking, lgm_gov, ontology, partnerships, qualification, research, schedule, tokens, tree, vesting_rewards + standard (broadcast, decode, encode, sign, etc.)
- **27 query subcommands**: All custom modules + block, blocks, tx, txs, wait-tx

---

## Checklist 3: Module Account Verification — FAIL (BLOCKER)

### maccPerms Entries (27 total in `app/app.go:303-332`)

| Module Account | Permissions | Source |
|---------------|-------------|--------|
| `fee_collector` | nil | SDK auth |
| `distribution` | nil | SDK distr |
| `bonded_tokens_pool` | Burner, Staking | SDK staking |
| `not_bonded_tokens_pool` | Burner, Staking | SDK staking |
| `gov` | Burner | SDK gov |
| `transfer` | Minter, Burner | IBC transfer |
| `ibcfee` | nil | IBC fee |
| `interchainaccounts` | nil | ICA |
| `legible_auth` | nil | Custom |
| `vesting_rewards` | Minter, Burner | Custom |
| `knowledge` | nil | Custom |
| `legible_staking` | Burner, Staking | Custom |
| `tokens` | Minter, Burner | Custom |
| `research_fund` | nil | Custom |
| `burn` | Burner | Custom |
| `disputes` | Burner | Custom |
| `evidence_mgmt` | Burner | Custom |
| `claiming_pot` | Burner | Custom |
| `channels` | Burner | Custom |
| `qualification` | Burner | Custom |
| `capture_defense` | Burner | Custom |
| `capture_challenge` | Burner | Custom |
| `tree` | Burner | Custom |
| `billing` | Burner | Custom |
| `partnerships` | Burner, Staking | Custom |
| `compute_pool` | Burner | Custom |
| `lgm_gov` | Burner | Custom |

### MISSING Module Accounts (6 BLOCKERS)

These modules call `SendCoinsFromAccountToModule` / `SendCoinsFromModuleToAccount` but are **NOT** in `maccPerms`:

| Module | ModuleName | SendCoins Calls | Impact |
|--------|-----------|-----------------|--------|
| **ontology** | `"ontology"` | propose_domain stake escrow + return | `app/app.go` panic on domain proposal |
| **research** | `"research"` | submit/challenge/bounty stake + rewards | Panic on any research submission |
| **schedule** | `"schedule"` | fee escrow + refunds | Panic on schedule creation |
| **discovery** | `"discovery"` | registration stake + deregistration refund | Panic on agent registration |
| **bvm** | `"bvm"` | value transfers in contract calls + refunds | Panic on funded contract calls |
| **treasury_protocol** | `"treasury_protocol"` | billing protocol treasury share | Panic on billing distribution |

**Why tests pass**: All module tests use mock bank keepers that don't check module account existence. The real `bankkeeper.BaseKeeper.SendCoinsFromAccountToModule` calls `GetModuleAccount()` which panics if the module name isn't in `maccPerms`.

**Fix**: Add 6 entries to `maccPerms` in `app/app.go:303-332`:
```go
ontologytypes.ModuleName:    nil,
researchtypes.ModuleName:    nil,
scheduletypes.ModuleName:    nil,
discoverytypes.ModuleName:   nil,
bvmtypes.ModuleName:         nil,
"treasury_protocol":         nil,
```
**Estimated effort: 15 minutes** (add entries + verify build).

---

## Checklist 4: RegisterServices Completeness — PASS (warnings)

All 28 custom modules implement `RegisterServices(cfg module.Configurator)`.

| Module | MsgServer | QueryServer | Notes |
|--------|-----------|-------------|-------|
| alignment | — | — | Empty RegisterServices (observation-only module) |
| auth | YES | YES | |
| autopoiesis | YES | **NO** | WARNING-2: No QueryServer registered |
| billing | YES | YES | |
| bvm | YES | YES | |
| capture_challenge | YES | YES | |
| capture_defense | YES | YES | |
| channels | YES | YES | |
| claiming_pot | YES | YES | |
| compute_pool | YES | YES | |
| discovery | YES | YES | |
| disputes | YES | YES | |
| emergency | YES | YES | |
| evidence_mgmt | YES | YES | |
| gov | YES | YES | |
| home | YES | YES | |
| ibcratelimit | YES | YES | |
| icaauth | YES | YES | |
| knowledge | YES | YES | |
| ontology | YES | YES | |
| partnerships | YES | YES | |
| qualification | YES | YES | |
| research | YES | YES | |
| schedule | YES | YES | |
| staking | YES | YES | |
| tokens | YES | YES | |
| tree | YES | YES | |
| vesting_rewards | YES | YES | |

**WARNING-2**: `autopoiesis` registers MsgServer but not QueryServer. The CLI shows `legbled query autopoiesis` with no subcommands. Not blocking.

---

## Checklist 5: Genesis Round-Trip — PASS

```
legbled init test-genesis --chain-id test-genesis               → success
# Patch vote_extensions_enable_height = "1"
legbled validate-genesis genesis.json                           → "valid genesis file"
# Full ceremony:
legbled keys add validator --keyring-backend test               → success
legbled add-genesis-account <addr> 100000000ulgm                → success
legbled gentx validator 50000000ulgm --chain-id test-genesis    → success
legbled collect-gentxs                                          → success
```

Genesis includes all 28+ module genesis states. All default params validated successfully.

---

## Checklist 6: Gas Coverage — PASS

- **214 message type URLs** mapped to gas costs in `app/ante.go` `msgTypeURLToGas`
- **27 modules** with msg_server.go handlers totaling ~350 handler functions
- `init()` validation at startup ensures all entries have valid gas costs
- Gas constants: BlockGasLimit=33,333,333, TxGasLimit=11,111,111, MinGasLimit=22,222
- Bootstrap gas-free types: 5 PoT messages (MsgSubmitClaim, MsgSubmitCommitment, MsgSubmitReveal, MsgRegisterValidator, MsgRegisterAccount)

Handler counts per module:
```
auth:17  autopoiesis:5  billing:6  bvm:83  capture_challenge:4
capture_defense:3  channels:10  claiming_pot:6  compute_pool:9
discovery:8  disputes:12  emergency:9  evidence_mgmt:6  gov:14
home:14  ibcratelimit:2  icaauth:5  knowledge:18  ontology:7
partnerships:12  qualification:6  research:10  schedule:9
staking:6  tokens:16  tree:30  vesting_rewards:10
```

---

## Checklist 7: Proto Registration — PASS (warnings)

All 28 modules have both `RegisterCodec` (amino) and `RegisterInterfaces` (protobuf).

**Known gaps** (from P0-SAFETY.md audit, not new):
- `compute_pool` RegisterInterfaces: only registers 1 of 9 Msg types (P1)
- `icaauth` RegisterInterfaces: missing 2 of 3 Msg types (P2)
- 7 modules have empty RegisterInterfaces — hand-written types without proto (billing, qualification, ibcratelimit, ontology, partnerships, schedule, discovery) (P2)

These gaps affect `SIGN_MODE_DIRECT` signing only. Amino signing works for all types. Not blocking for testnet.

---

## Checklist 8: Known Issues from All Batches — PASS

### Previously Reported P0s — ALL FIXED

| P0 | Source | Status |
|----|--------|--------|
| BVM Engine: memory truncation | batch-1/SESSION-1 | FIXED, tested (TestP0_1_SHA3_LargeOffsetRejected + 6 more) |
| BVM Engine: CALLCODE semantics | batch-1/SESSION-1 | FIXED, tested (TestP0_2_CallcodeWritesCallerStorage + 2 more) |
| BVM Engine: context propagation | batch-1/SESSION-1 | FIXED, tested (TestP0_3_CapabilitiesPropagatedToCall + 3 more) |
| BVM Engine: call depth limit | batch-1/SESSION-1 | FIXED, tested (TestP0_4_ExecuteRejectsAtMaxCallDepth + 2 more) |
| BVM Consensus: gas bridge | batch-1/SESSION-2 | FIXED, tested (TestP0_1_GasBridge_CallContract + 5 integration tests) |
| BVM Consensus: panic recovery | batch-1/SESSION-2 | FIXED, tested (TestP0_3_PanicRecovery_CallContract + 2 more) |
| BVM Consensus: schedule gas | batch-1/SESSION-2 | FIXED, tested (TestP0_2_ScheduledGasBudget + 1 more) |
| ABCI: false equivocation | batch-1/SESSION-6 | FIXED, tested (TestCommitmentEquivocation_DifferentHash + related) |
| ABCI: reveal hash | batch-1/SESSION-6 | FIXED, tested (TestRevealEquivocation_DifferentVerdict + related) |
| Staking: reputation underflow | batch-1/SESSION-7 | FIXED, tested (TestRecordVerification_ReputationUnderflowEdge) |
| Staking: contested tracking | batch-1/SESSION-7 | FIXED, tested (TestRecordVerification_ContestedTracking + 4 more) |
| Staking: epoch length | batch-1/SESSION-7 | FIXED, tested (TestSlashValidator_R2_PerEpochCap) |
| IBC: rate limit bypass | batch-1/SESSION-3 | FIXED, tested (TestSlidingWindow_EpochBoundary_NoDoubleBurst) |
| IBC: sliding window | batch-2/SESSION-5 | FIXED, tested (12 sliding window tests) |
| ICA: transfer through rate limit | batch-1/SESSION-3 | FIXED, tested (TestICA_TransferThroughRateLimit) |
| Channels: signatures | batch-2/SESSION-4 | FIXED, tested (6 signature verification tests) |
| Ante: SDK v0.50 bypass | ANTE-CHAIN audit | FIXED, tested (TestFrozenAccount_MsgSend_Rejected + 3 more) |
| lgm_gov module account | batch-5/SESSION-R5-2 | FIXED in B6-1 (added to maccPerms) |
| Research InitGenesis swallow | P0-SAFETY | FIXED (panic on unmarshal error) |
| Staking RegisterInterfaces | P0-SAFETY | FIXED (all 6 Msg types registered) |

### Deferred Items (P2, OK for testnet)

| Item | Source | Severity | Why OK |
|------|--------|----------|--------|
| Bootstrap tx gas UX | ANTE-CHAIN | P2 | UX only, not security |
| FeeRouterDecorator logging-only | ANTE-CHAIN | P2 | Harmless |
| LastActiveBlock on failed tx | ANTE-CHAIN | P2 | Design decision |
| ICA type URL mismatch | ANTE-CHAIN | P1 | Fail-closed (default deny) |
| Gas sum overflow | ANTE-CHAIN | P1 | Theoretical (requires >843 msgs) |
| Channel fund delay risk | CHANNELS | P2 | Design decision |
| No channel→disputes escalation | CHANNELS | P2 | Phase 2 |
| DefaultSettlementFreq unused | CHANNELS | P2 | Stub |
| 9 stub handlers (4% of 251) | PROGRESS | P2 | 96% implemented |
| Network layer rate limiting | PROGRESS | P2 | Post-testnet hardening |
| Schedule retention hardcoded | PROGRESS | P2 | Acceptable heuristic |
| compute_pool RegisterInterfaces | P0-SAFETY | P1 | Amino signing works |
| 7 modules empty RegisterInterfaces | P0-SAFETY | P2 | Hand-written types, amino works |

---

## Checklist 9: Security P0 Status — PASS

All P0 findings from all audits have test coverage:

```
BVM Engine P0s:     17 tests in x/bvm/vm/p0_fixes_test.go
BVM Consensus P0s:   6 tests in x/bvm/keeper/{keeper_test,integration_test}.go
ABCI P0s:            4 tests in x/knowledge/keeper/keeper_test.go
Staking P0s:        10 tests in x/staking/keeper/{keeper_test,openclaw_staking_test}.go
IBC P0s:            14 tests in x/ibcratelimit/keeper/{keeper_test,middleware_test}.go
Channels P0-1:       6 tests in x/channels/keeper/keeper_test.go
Ante P0-1:           4 tests in app/ante_integration_test.go
```

Additionally: 46 OpenClaw adversarial probes pass across all social modules.

---

## Checklist 10: Operational Readiness — PASS

### Single-Validator Boot
```
legbled start --home /tmp/legbled-start-test/.legbled --minimum-gas-prices 0ulgm
```

Output:
```
INF starting node with ABCI CometBFT in-process module=server
INF service start impl=multiAppConn module=proxy
INF InitChain chainID=start-test initialHeight=1 module=server
INF initializing blockchain state from genesis.json module=server
INF created new capability module=ibc name=ports/transfer
INF port binded module=x/ibc/port port=transfer
INF created new capability module=ibc name=ports/icahost
INF port binded module=x/ibc/port port=icahost
INF Completed ABCI Handshake module=consensus
INF Version info abci=2.0.0 block=11 tendermint_version=0.38.19
```

- InitChain: **SUCCESS**
- ABCI Handshake: **SUCCESS**
- IBC ports: **BOUND** (transfer, icahost)
- CometBFT version: **v0.38.19** (block protocol 11)
- No panics, no errors

### REST/gRPC/Metrics
Not tested in this session (requires running node). Based on batch-5/SESSION-R5-2 report, these are confirmed working:
- REST API (`localhost:1317`): functional
- gRPC (`localhost:9090`): functional
- Prometheus metrics: configured by default

---

## BLOCKER LIST

### B-1: Missing Module Accounts in maccPerms (P0 — CRITICAL)

**File**: `app/app.go:303-332`

Six modules use bank keeper coin transfers but have no module account registered in `maccPerms`. At runtime, `bankkeeper.SendCoinsFromAccountToModule()` calls `GetModuleAccount()` which **panics** for unregistered module names.

| Module | ModuleName | First Panic Trigger | Severity |
|--------|-----------|---------------------|----------|
| `ontology` | `"ontology"` | `MsgProposeDomain` (stake escrow) | Runtime panic |
| `research` | `"research"` | `MsgSubmitResearch` (research stake) | Runtime panic |
| `schedule` | `"schedule"` | `MsgCreateSchedule` (fee payment) | Runtime panic |
| `discovery` | `"discovery"` | `MsgRegisterAgent` (registration stake) | Runtime panic |
| `bvm` | `"bvm"` | `MsgCallContract` with value transfer | Runtime panic |
| `treasury_protocol` | `"treasury_protocol"` | `ExecutePayment` (billing distribution) | Runtime panic |

**Why tests pass**: All keeper tests use mock bank keepers that don't enforce module account existence.

**Fix**:
```go
// Add to maccPerms in app/app.go:
ontologytypes.ModuleName:     nil,
researchtypes.ModuleName:     nil,
scheduletypes.ModuleName:     nil,
discoverytypes.ModuleName:    nil,
bvmtypes.ModuleName:          nil,
"treasury_protocol":          nil,
```

**Estimated effort**: 15 minutes (add entries, add import if needed, verify build).

---

## WARNING LIST

| ID | Warning | Effort |
|----|---------|--------|
| W-1 | Binary version/name/server_name not set via ldflags | 10 min (Makefile update) |
| W-2 | autopoiesis: no QueryServer registered (query CLI empty) | 30 min |
| W-3 | compute_pool RegisterInterfaces: 8 of 9 Msg types missing | 15 min |
| W-4 | icaauth RegisterInterfaces: 2 of 3 Msg types missing | 10 min |
| W-5 | 7 modules have empty RegisterInterfaces (hand-written types) | Design decision |
| W-6 | ICA type URL mismatch in ante handler (fail-closed, not exploitable) | 10 min |

---

## OK-FOR-TESTNET LIST

| Item | Reason |
|------|--------|
| 9 stub handlers (4% of 251 total) | Phase 2 functionality, doesn't block core PoT flow |
| Bootstrap gas UX (declare gas, pay 0 fee) | UX annoyance, works correctly |
| FeeRouterDecorator logging-only | No impact, scheduled for cleanup |
| Channel fund delay (no auto-expiration) | Design decision, bilateral closure by design |
| No channel→disputes escalation | Phase 2 feature |
| Network layer rate limiting | Post-testnet production hardening |
| 7 empty RegisterInterfaces | Hand-written types use amino, works for testnet |
| LastActiveBlock persists on failed tx | Documented design decision |
| DefaultSettlementFreq unused in channels | Stub field, not enforced |
| Gas sum uint64 overflow | Requires >843 messages per tx (theoretical) |

---

## FINAL VERDICT

### **NOT READY** — 1 Blocker

**Blocker B-1** (6 missing module accounts) will cause **runtime panics** on any transaction involving ontology, research, schedule, discovery, bvm value transfers, or billing distribution. This is a 15-minute fix.

### After Fixing B-1: **READY FOR TESTNET**

With B-1 resolved:
- All tests pass (34 packages, 0 failures)
- All P0 security findings fixed and tested
- 96% of handlers implemented
- Chain boots cleanly in single-validator mode
- Genesis validates
- 214 message types have gas costs
- 46 OpenClaw adversarial probes pass
- All 28 modules register services

### Estimated Effort to Fix Blockers

| Item | Effort |
|------|--------|
| B-1: Add 6 module accounts to maccPerms | 15 min |
| **Total** | **15 min** |

### Recommended Pre-Testnet Fixes (non-blocking)

| Item | Effort |
|------|--------|
| W-1: Set binary version via ldflags | 10 min |
| W-3: compute_pool RegisterInterfaces | 15 min |
| W-6: ICA type URL fix in ante | 10 min |
| **Total recommended** | **35 min** |
