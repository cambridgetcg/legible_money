# R4 Session 5: Proto TX Migration — Regression Verification & Codec Integrity

**Date**: 2026-02-18
**Scope**: 12 modules migrated in R3-3 (autopoiesis, billing, compute_pool, discovery, ontology, partnerships, qualification, schedule, capture_challenge, capture_defense, icaauth, staking)
**Verdict**: **CLEAN** — No regressions found. All 12 modules pass tests, build succeeds, gRPC wiring intact.

---

## 1. Amino Type URL Audit

All 12 modules have correct `RegisterConcrete` + `RegisterImplementations` in their `types/codec.go`. Type URLs are consistent and unique per module.

| Module | Prefix | Msg Count | RegisterImplementations |
|--------|--------|-----------|------------------------|
| autopoiesis | `autopoiesis/` | 4 | Yes |
| billing | `billing/` | 5 | Yes |
| compute_pool | `compute_pool/` | 8 | Yes |
| discovery | `discovery/` | 7 | Yes |
| ontology | `ontology/` | 6 | Yes |
| partnerships | `partnerships/` | 16 | Yes |
| qualification | `qualification/` | 5 | Yes |
| schedule | `schedule/` | 6 | Yes |
| capture_challenge | (hand-written types, no proto_names.go) | 4 | N/A |
| capture_defense | `capture_defense/` | 3 | Yes |
| icaauth | `icaauth/` | 3 | Yes |
| staking | `legible_staking/` | 6 | Yes |

**Finding**: No type URL changes detected. All Amino names preserved from pre-migration.

---

## 2. GetSigners Wiring Verification

### Modules with Hand-Written GetSigners (4)

These modules have both proto `(cosmos.msg.v1.signer)` annotations AND hand-written `GetSigners()` methods. All verified to agree:

| Module | Msg | GetSigners Field | Proto Signer | Match |
|--------|-----|------------------|-------------|-------|
| **qualification** | MsgRequestQualification | `m.ValidatorAddr` | `validator_addr` | YES |
| | MsgEndorseValidator | `m.EndorserAddr` | `endorser_addr` | YES |
| | MsgRenewQualification | `m.ValidatorAddr` | `validator_addr` | YES |
| | MsgWithdrawQualification | `m.ValidatorAddr` | `validator_addr` | YES |
| | MsgUpdateParams | `msg.Authority` | `authority` | YES |
| **capture_challenge** | MsgSubmitChallenge | `m.Challenger` | `challenger` | YES |
| | MsgAddEvidence | `m.Challenger` | `challenger` | YES |
| | MsgResolveChallenge | `m.Resolver` | `resolver` | YES |
| | MsgUpdateParams | `msg.Authority` | `authority` | YES |
| **capture_defense** | MsgRequestQualification | `m.ValidatorAddr` | `validator_addr` | YES |
| | MsgEndorseQualification | `m.Endorser` | `endorser` | YES |
| | MsgUpdateParams | `msg.Authority` | `authority` | YES |
| **staking** | MsgRegisterValidator | `m.ValidatorAddress` | `validator_address` | YES |
| | MsgUpdateValidatorStake | `m.Validator` | `validator` | YES |
| | MsgDelegate | `m.Delegator` | `delegator` | YES |
| | MsgUndelegate | `m.Delegator` | `delegator` | YES |
| | MsgRedelegate | `m.Delegator` | `delegator` | YES |
| | MsgUpdateParams | `msg.Authority` | `authority` | YES |

### Modules with Proto-Only Signers (8)

autopoiesis, billing, compute_pool, discovery, ontology, partnerships, schedule, icaauth — all rely exclusively on `(cosmos.msg.v1.signer)` proto annotations. All annotations verified present and correct.

---

## 3. gRPC Service Registration

All 12 modules verified:

| Module | `RegisterMsgServer` in module.go | `Msg_ServiceDesc` in tx_grpc.pb.go | `tx.pb.go` exists |
|--------|----------------------------------|------------------------------------|--------------------|
| autopoiesis | YES (line 83) | YES (line 216) | YES |
| billing | YES (line 103) | YES (line 250) | YES |
| compute_pool | YES (line 93) | YES (line 352) | YES |
| discovery | YES (line 83) | YES (line 318) | YES |
| ontology | YES (line 103) | YES (line 284) | YES |
| partnerships | YES (line 103) | YES (line 634) | YES |
| qualification | YES (line 90) | YES (line 250) | YES |
| schedule | YES (line 96) | YES (line 284) | YES |
| capture_challenge | YES (line 102) | YES (line 216) | YES |
| capture_defense | YES (line 102) | YES (line 182) | YES |
| icaauth | YES (line 83) | YES (line 182) | YES |
| staking | YES (line 110) | YES (line 292) | YES |

**Note**: capture_challenge and capture_defense pass the keeper directly to `RegisterMsgServer` (the keeper implements the server interface directly), while all others use `keeper.NewMsgServerImpl(am.keeper)`. Both patterns are valid.

---

## 4. Module Test Results

All 12 migrated modules pass:

```
autopoiesis/keeper     1.234s  PASS
autopoiesis/types      0.677s  PASS
billing/keeper         0.734s  PASS
compute_pool/keeper    0.734s  PASS
discovery/keeper       0.729s  PASS
ontology/keeper        0.742s  PASS
partnerships/keeper    0.756s  PASS
qualification/keeper   0.765s  PASS
schedule/keeper        0.741s  PASS
capture_challenge/keeper 0.742s PASS
capture_defense/keeper 0.806s  PASS
icaauth/keeper         0.804s  PASS
staking/keeper         0.608s  PASS
```

All remaining modules also pass (knowledge, home, bvm, emergency, gov, disputes, evidence_mgmt, research, tree, channels, vesting_rewards, claiming_pot, alignment, ibcratelimit, tokens).

App tests: **PASS** (1.009s)

---

## 5. Build Verification

| Command | Result |
|---------|--------|
| `go build ./...` | **PASS** (exit 0) |
| `go test ./x/... -count=1` | **ALL PASS** (0 failures) |
| `go test ./app/... -count=1` | **PASS** |
| `go vet ./...` | **KNOWN WARNINGS** (see below) |

---

## 6. Pre-Existing Issues (Not Regressions)

### `go vet` Mutex Copy Warnings

`go vet` reports ~150+ "copies lock value" warnings across `app/` and `x/knowledge/keeper/` test files. These are **NOT** regressions from the TX migration — they are a systemic issue where protobuf-generated types (containing `protoimpl.MessageState` with `sync.Mutex`) are passed by value instead of by pointer.

**Affected areas**:
- `app/rest_api.go` — ~60 warnings (struct literal copies)
- `app/alignment_adapters.go` — ~10 warnings (func params by value)
- `app/vote_extensions.go` — ~10 warnings (range copies, func params)
- `app/staking_adapter.go` — ~5 warnings (range copies)
- `x/knowledge/keeper/*_test.go` — ~50 warnings (SetClaim/SetFact by value)

**Root cause**: Functions like `SetClaim(ctx, claim Claim)`, `SetFact(ctx, fact Fact)`, and REST API handlers pass proto structs by value. The fix is to change signatures to pointer receivers (`*Claim`, `*Fact`) throughout. This is a separate cleanup task, not a migration regression.

---

## Summary

| Check | Status |
|-------|--------|
| Amino Type URLs | PASS — All preserved |
| GetSigners Wiring | PASS — 4 hand-written + 8 proto-only, all correct |
| gRPC Registration | PASS — All 12 modules wired |
| Module Tests | PASS — 12/12 migrated + all remaining |
| App Tests | PASS |
| Build | PASS |
| go vet | KNOWN WARNINGS (pre-existing mutex copy, not a regression) |
| **Regressions Found** | **0** |
