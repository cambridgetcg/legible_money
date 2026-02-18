# Session R3-3: Proto TX Generation for 12 Modules

**Date**: 2026-02-18
**Scope**: Add `tx.proto` for 11 modules missing gRPC TX surface + expand staking `tx.proto`

## Summary

Created `tx.proto` definitions for 11 modules and expanded 1 existing proto, generating proper gRPC transaction endpoints. Removed all hand-written Msg structs, MsgServer interfaces, gRPC descriptors, and proto_names.go files that conflicted with generated code. Preserved GetSigners/ValidateBasic validation logic.

**Total new gRPC transaction RPCs: 72** (67 new + 5 expanded on staking)

## Module Results

| Module | Proto Created | RPCs | msg_server Updated | proto_names Deleted | Tests Pass |
|--------|:---:|:---:|:---:|:---:|:---:|
| autopoiesis | NEW | 4 | YES | YES | YES |
| billing | NEW | 5 | YES | YES | YES |
| compute_pool | NEW | 8 | YES | YES | YES |
| discovery | NEW | 7 | YES | YES | YES |
| ontology | NEW | 6 | YES | YES | YES |
| partnerships | NEW | 16 | YES | YES | YES |
| qualification | NEW | 5 | YES | YES | YES |
| schedule | NEW | 6 | YES | YES | YES |
| capture_challenge | NEW | 4 | YES | YES | YES |
| capture_defense | NEW | 3 | YES | YES | YES |
| icaauth | NEW | 3 | YES | YES | YES |
| staking | EXPANDED | 6 (was 1) | YES | YES | YES |

## Per-Module Details

### Phase 1: HIGH Priority (17 RPCs)

**autopoiesis** (4 RPCs) — First module to get proto dir (`proto/legible/autopoiesis/v1/`)
- `UpdateParams`, `Activate`, `OverrideMultiplier`, `FreezeMultiplier`
- Params message defined inline in tx.proto (no pre-existing state.proto)
- Removed: `msgs.go` struct defs, `service.go`, `msg_server.go` (interface), `proto_names.go`

**billing** (5 RPCs)
- `RegisterProvider`, `DeregisterProvider`, `RequestQuote`, `ExecutePayment`, `UpdateParams`
- Imports `state.proto` for `APIQuote` and `Params`
- Removed: Msg structs + gRPC from `types.go`, `proto_names.go`

**compute_pool** (8 RPCs)
- `RegisterComputeProvider`, `UpdateProvider`, `DeregisterProvider`, `ProviderHeartbeat`, `RedeemCredits`, `ConfirmUsage`, `DisputeUsage`, `UpdateComputeParams`
- Removed dead `MsgUpdateParams` (duplicate of `MsgUpdateComputeParams`)
- Removed: Msg structs + gRPC from `types.go`, `proto_names.go`

### Phase 2: MEDIUM Priority (42 RPCs)

**discovery** (7 RPCs)
- `RegisterAgent`, `UpdateProfile`, `AddCapability`, `RemoveCapability`, `DeregisterAgent`, `Heartbeat`, `UpdateParams`
- Imports `state.proto` for `AgentCapability` and `Params`

**ontology** (6 RPCs)
- `ProposeDomain`, `VoteDomainProposal`, `UpdateDomain`, `RegisterLogicZone`, `AcknowledgeIncompleteness`, `UpdateParams`
- Imports both `state.proto` (LogicZoneProperties) and `genesis.proto` (Params)

**partnerships** (16 RPCs) — Largest module
- Cold-start: `InitiateSeedPartnership`, `GraduateSeedPartnership`, `RegisterInPool`, `DeclareIntent`, `SponsorPartnership`
- Core: `InitiatePartnership`, `AcceptPartnership`, `DepositToPot`, `DistributeReward`
- Deliberation: `ProposeOperation`, `ApproveOperation`, `RejectOperation`
- Safety: `SafetyFreeze`, `RaiseCoercionSignal`, `InitiateExit`
- Governance: `UpdateParams`

**qualification** (5 RPCs)
- `RequestQualification`, `EndorseValidator`, `RenewQualification`, `WithdrawQualification`, `UpdateParams`

**schedule** (6 RPCs)
- `CreateSchedule`, `PauseSchedule`, `ResumeSchedule`, `CancelSchedule`, `AddFee`, `UpdateParams`
- Imports `state.proto` for `ScheduleCondition`

**capture_challenge** (4 RPCs)
- `SubmitChallenge`, `AddEvidence`, `ResolveChallenge`, `UpdateParams`
- Imports `state.proto` for `CaptureEvidence`

**capture_defense** (3 RPCs)
- `RequestQualification`, `EndorseQualification`, `UpdateParams`

### Phase 3: LOW Priority (8 RPCs)

**icaauth** (3 RPCs)
- `RegisterInterchainAccount`, `SendICATransaction`, `UpdateParams`

**staking** (expanded: 1 → 6 RPCs, +5 new)
- Existing: `RegisterValidator`
- Added: `UpdateValidatorStake`, `Delegate`, `Undelegate`, `Redelegate`, `UpdateParams`
- Removed `FullMsgServer` interface (proto-generated MsgServer now has all 6 methods)

## Cleanup Summary

For each module, the following was performed:
1. **Deleted**: `proto_names.go` (XXX_MessageName methods — proto-generated types have proper names)
2. **Removed from types.go/msgs.go**: Hand-written Msg struct definitions, Response structs, Reset/String/ProtoMessage methods, MsgServer interface, gRPC service descriptor + handlers, RegisterMsgServer
3. **Preserved**: GetSigners() and ValidateBasic() methods (moved to msgs_validate.go or kept in types.go)
4. **Updated**: `keeper/msg_server.go` — embedded `types.UnimplementedMsgServer` in msgServer struct
5. **Verified**: `codec.go` RegisterInterfaces works with proto-generated types

## Test Results

```
go build ./...     → PASS (clean)
go vet ./...       → PASS (pre-existing warnings in auth/emergency only)
go test ./app/...  → PASS
go test ./tests/...→ PASS
go test ./x/...    → 1 FAIL (pre-existing: x/knowledge/evaluation — unrelated to this session)
```

All 12 modified modules pass their tests individually. The single failure in `x/knowledge/evaluation` is pre-existing (files were already modified before this session).

## Files Changed

- **12 tx.proto files** created/expanded
- **12 tx.pb.go + 12 tx_grpc.pb.go** generated by `buf generate`
- **11 proto_names.go** deleted
- **11 types.go/msgs.go** cleaned (hand-written Msg/gRPC removed, validation kept)
- **2 service.go** deleted (autopoiesis)
- **2 msg_server.go** deleted (autopoiesis type-level interface)
- **12 keeper/msg_server.go** updated (UnimplementedMsgServer embed)
