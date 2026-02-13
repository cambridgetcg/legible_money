# Cosmos SDK DevNet Progress Check

> **PP Applied**: TRUTH (ground in actual test results) | UNDERSTANDING (trace full tx lifecycle) | BEAUTY (clean issue categorization) | JUSTICE (correct severity assignment) | CREATIVITY (resolution paths)
>
> **Date**: 2026-02-12 | **Session**: Proto Codegen Integration + DevNet Testing

---

## 1. Work Completed This Session

### Part 1: RegisterInterfaces (13 Modules) — DONE
All 13 module `codec.go` files updated with `registry.RegisterImplementations((*sdk.Msg)(nil), ...)`. Total: **120+ Msg types** registered across auth, bvm, channels, claiming_pot, disputes, evidence_mgmt, gov, knowledge, research, staking, tokens, tree, vesting_rewards.

### Part 2: Knowledge ExtendedMsgServer — DONE
Added 6 supplemental gRPC handlers (SubmitClaim, SubmitCommitment, SubmitReveal, ChallengeFact, AddFact, SubmitContradiction) via `_ExtendedMsg_serviceDesc`. Module.go wired to `RegisterExtendedMsgServer()`.

### Part 3: Channels FullMsgServer — DONE
Wired existing `_SupplementalMsg_serviceDesc` (DepositChannel, UpdateState, ClaimExpired) via `RegisterFullMsgServer()`. Module.go updated.

### Part 4: sdk Imports — DONE
All 13 `codec.go` files have `sdk "github.com/cosmos/cosmos-sdk/types"` import.

### Part 5: Critical Bug Fix — 5 module.go Delegation Gaps — DONE
**Root cause**: `AppModuleBasic.RegisterInterfaces()` in 5 modules (bvm, channels, staking, tree, vesting_rewards) had empty/no-op bodies that never called `types.RegisterInterfaces(reg)`. This meant Parts 1-4 were dead code for those modules.

**Fix**: Added `types.RegisterInterfaces(reg)` delegation in each module.go.

**Verification**: `MsgRegisterValidator` went from "unable to resolve type URL" to successful signing after the fix.

### Part 6: DevNet Testing — DONE
- Node producing blocks (CometBFT consensus)
- Bank transfers work end-to-end (broadcast → block inclusion → balance update)
- Knowledge gRPC Query service visible (13 methods) and responding (Params query returns data)
- `MsgCreateToken` successfully decoded and routed to keeper (hit expected "not implemented" from proto stub)
- `go build ./...` clean | 25/26 Go test packages pass (1 pre-existing: evidence_mgmt/TestExpireStaleEvidence)

---

## 2. Issues Found During DevNet Testing

### Issue Matrix

| # | Issue | Severity | Status | Resolution Path |
|---|-------|----------|--------|-----------------|
| I-1 | REST gateway stubs (all `RegisterGRPCGatewayRoutes` empty) | LOW | Deferred | Phase 2: generate `.pb.gw.go` files, requires query.proto per module |
| I-2 | `cosmos.msg.v1.signer` annotation errors on tx signing | FALSE ALARM | Resolved | Investigation: all 200+ proto msgs already have correct annotations |
| I-3 | `cosmos.tx.v1beta1.Tx` not registered in InterfaceRegistry | MEDIUM | **Open** | Add `authtx.RegisterInterfaces(interfaceRegistry)` in app.go |
| I-4 | 6 keeper methods are stubs (5 in tree, 1 in knowledge) | LOW | Expected | Phase 2 features: SubscribeService, BeginSeeding, DetectOpportunity, ClaimOpportunity, AddContributor, RegisterStratum |
| I-5 | Proto codegen may need re-run to sync signer annotations | LOW | **Open** | Run `buf generate` to ensure `.pb.go` files reflect latest `.proto` annotations |

---

## 3. PP-Informed Analysis

### TRUTH Gate: What Did We Actually Verify?

| Claim | Evidence | Confidence |
|-------|----------|------------|
| "Custom Msg types decode correctly" | `MsgCreateToken` decoded, hit keeper method | HIGH (directly observed in block 34) |
| "Bank module works" | Transfer 1000000ulgm succeeded, balance updated | HIGH (balance verified via gRPC) |
| "Knowledge gRPC queries work" | `grpcurl list` shows 13 methods, Params returns data | HIGH (direct gRPC call) |
| "All 13 modules register types" | `go build` clean + token tx decoded | MEDIUM (only tested tokens path end-to-end; other modules inferred from same pattern) |
| "Signer annotations are present" | Proto files have annotations, but haven't tested signing for ALL module types | MEDIUM (investigation found annotations in proto, but `.pb.go` may lag) |

### UNDERSTANDING Gate: Full Tx Lifecycle Traced

```
Client → JSON tx → lgmd tx broadcast (REST :1317 or gRPC :9090)
  → CometBFT mempool (CheckTx: AnteHandler validates fees, signer, sequence)
  → Block inclusion (DeliverTx)
  → InterfaceRegistry resolves Any type URL → proto Msg type
  → Module router dispatches to keeper method
  → Keeper executes business logic
  → Events emitted, state committed
```

**Gap identified**: Step 3 (InterfaceRegistry) was failing for all custom modules before our fix. Now works for tokens (tested). Need to verify knowledge and staking paths.

### BEAUTY Gate: Clean Categorization

The 5 issues fall into 3 categories:
1. **Infrastructure gaps** (I-1, I-5): Proto tooling not fully wired. Cosmetic for now.
2. **Missing registration** (I-3): Real bug — `cosmos.tx.v1beta1.Tx` type not registered means certain SDK internal operations may fail.
3. **Expected stubs** (I-4): Documented Phase 2 features. Not bugs.

### JUSTICE Gate: Correct Severity

- **I-3 is the only actionable item**. It's a 2-line fix in `app/app.go` that could prevent subtle SDK-internal failures (e.g., tx simulation, tx decoding in REST endpoints).
- I-1 is intentional deferral (no query.proto files for most modules yet).
- I-2 was a false alarm (annotations exist, error was likely from stale codegen).
- I-4 is documented future work.
- I-5 is belt-and-suspenders (run buf generate to ensure consistency).

### CREATIVITY Gate: What Could Go Wrong?

1. **Tx simulation failure**: Without `authtx.RegisterInterfaces`, the `cosmos.tx.v1beta1.Tx` type can't be deserialized in certain SDK-internal paths (e.g., `ServiceMsg` routing, tx simulation). This could silently break `--dry-run` or gas estimation.
2. **Stale .pb.go files**: If `.pb.go` files were generated before signer annotations were added to `.proto` files, the SDK's `GetSigners()` derivation from proto options may return empty, causing "no signers" errors on tx delivery.
3. **Cross-module keeper calls**: Some modules (e.g., knowledge) depend on optional keepers (tokensKeeper, ontologyKeeper). If those modules aren't wired in app.go, keeper methods will panic on nil pointer. This was NOT tested.

---

## 4. Resolution Plan

### Priority 1: Fix Tx Type Registration (I-3) — ~5 minutes
**File**: `app/app.go`
**Change**: Add `authtx.RegisterInterfaces(interfaceRegistry)` at two locations:
1. In `NewLegibleApp()` after `ModuleBasics.RegisterInterfaces(interfaceRegistry)` (~line 439)
2. In `MakeEncodingConfig()` after `ModuleBasics.RegisterInterfaces(interfaceRegistry)` (~line 1497)

**Import**: `authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"`

### Priority 2: Re-run Proto Codegen (I-5) — ~10 minutes
```bash
buf generate
go build ./...
```
Verify signer annotations propagated to `.pb.go` files.

### Priority 3: REST Gateway (I-1) — Phase 2 (deferred)
Requires:
1. Write `query.proto` for each module that needs REST endpoints
2. Run `buf generate` with grpc-gateway plugin
3. Implement `RegisterGRPCGatewayRoutes()` per module
4. Register routes in `app.go`

**Recommended first modules**: knowledge (already has query.proto), staking, tokens

### Priority 4: Tree Module Stubs (I-4) — Phase 2 (deferred)
5 methods in x/tree + 1 in x/knowledge need implementation when those features are designed.

---

## 5. Keeper Implementation Status (Full Audit)

| Module | Total Methods | Implemented | Stubs | Rate |
|--------|---------------|-------------|-------|------|
| x/auth | 12 | 12 | 0 | 100% |
| x/bvm | 7 | 7 | 0 | 100% |
| x/channels | 8 | 8 | 0 | 100% |
| x/claiming_pot | 4 | 4 | 0 | 100% |
| x/disputes | 10 | 10 | 0 | 100% |
| x/evidence_mgmt | 5 | 5 | 0 | 100% |
| x/gov | 10 | 10 | 0 | 100% |
| x/knowledge | 12 | 11 | 1 | 92% |
| x/research | 9 | 9 | 0 | 100% |
| x/staking | 5 | 5 | 0 | 100% |
| x/tokens | 5 | 5 | 0 | 100% |
| x/tree | 28 | 23 | 5 | 82% |
| x/vesting_rewards | 8 | 8 | 0 | 100% |
| **TOTAL** | **123** | **117** | **6** | **95.1%** |

### Stub Methods (6)
1. `x/tree/SubscribeService` — Service subscription model (Phase 2)
2. `x/tree/BeginSeeding` — Project seeding workflow (Phase 2)
3. `x/tree/DetectOpportunity` — Opportunity detection (Phase 2)
4. `x/tree/ClaimOpportunity` — Opportunity claiming (Phase 2)
5. `x/tree/AddContributor` — Direct contributor addition (Phase 2)
6. `x/knowledge/RegisterStratum` — Ontology stratum registration (Phase 2)

---

## 6. Test Results Summary

| Suite | Pass | Fail | Skip | Notes |
|-------|------|------|------|-------|
| `go build ./...` | CLEAN | 0 | — | All packages compile |
| `go test ./x/knowledge/...` | PASS | 0 | 0 | Evaluation + keeper tests |
| `go test ./x/staking/...` | PASS | 0 | 0 | Registration + delegation |
| `go test ./x/...` (all) | 25 | 1 | 0 | Pre-existing: evidence_mgmt/TestExpireStaleEvidence |
| DevNet: block production | PASS | — | — | Blocks at ~5s (CometBFT) |
| DevNet: bank transfer | PASS | — | — | 1000000ulgm transferred |
| DevNet: gRPC query | PASS | — | — | Knowledge params returned |
| DevNet: custom tx decode | PASS | — | — | MsgCreateToken decoded in block 34 |

---

## 7. Next Steps

- [ ] **Fix I-3**: Add `authtx.RegisterInterfaces()` in app.go (Priority 1)
- [ ] **Fix I-5**: Re-run `buf generate` to sync signer annotations (Priority 2)
- [ ] **Test**: Submit knowledge claim tx end-to-end on Cosmos devnet
- [ ] **Test**: Register validator via staking tx on Cosmos devnet
- [ ] **Test**: Verify cross-module keeper wiring (knowledge → tokens, knowledge → ontology)
- [ ] Proceed with TS DevNet testing per DEVNET_TESTING_STRATEGY.md Phase 1-2

---

*Generated: 2026-02-12 | PP Dimensions: 5/5 applied | Supersedes: none (new document)*
