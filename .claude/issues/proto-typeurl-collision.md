# ISSUE: Proto TypeURL Collision — Binary Panics on Startup

**Severity:** BLOCKER — `legbled` cannot start at all
**Found:** 2026-02-17
**Status:** OPEN

## Symptom

```
panic: concrete type *types.MsgUpdateParams has already been registered under typeURL /,
cannot register *types.MsgUpdateParams under same typeURL
```

The binary builds (`go build`) but panics immediately on any CLI command (`legbled version`, `legbled init`, etc.).

## Root Cause

Multiple custom modules define hand-written Go message types (e.g. `MsgUpdateParams`, `MsgRegisterAgent`) that implement `proto.Message` with empty `ProtoMessage()` stubs. Without proper protobuf registration, `proto.MessageName()` returns `""` for all of them, giving them all typeURL `"/"`.

When `RegisterImplementations((*sdk.Msg)(nil), &MsgUpdateParams{})` is called by each module, the Cosmos SDK interface registry sees duplicate registrations under the same typeURL and panics.

## Scope

**87 hand-written message types across 22 modules** need unique proto names.

Some modules ALSO have protobuf-generated types (`.pb.go` files) that already have proper names — those are fine. The issue is ONLY with hand-written types in `types.go` or `msgs.go`.

### Affected Modules (hand-written types needing fix)

| Module | File | Types Needing Fix |
|--------|------|-------------------|
| auth | types.go | MsgRegisterAccount, MsgFreezeAccount, MsgUnfreezeAccount, MsgSetRecoveryConfig, MsgInitiateRecovery, MsgSubmitRecoveryShard, MsgChallengeRecovery, MsgExecuteRecovery |
| autopoiesis | msgs.go | MsgUpdateParams, MsgActivateAutopoiesis, MsgOverrideMultiplier, MsgFreezeMultiplier |
| billing | types.go | MsgRegisterProvider, MsgDeregisterProvider, MsgRequestQuote, MsgExecutePayment, MsgUpdateParams |
| bvm | types.go | MsgScheduleContract, MsgCancelSchedule, MsgUpdateContractState, MsgUpdateParams |
| channels | types.go | MsgClaimExpired, MsgDepositChannel, MsgUpdateState, MsgUpdateParams |
| claiming_pot | types.go | MsgUpdateParams |
| compute_pool | types.go | MsgRegisterComputeProvider, MsgUpdateProvider, MsgDeregisterProvider, MsgProviderHeartbeat, MsgRedeemCredits, MsgConfirmUsage, MsgDisputeUsage, MsgUpdateComputeParams, MsgUpdateParams |
| discovery | types.go | MsgRegisterAgent, MsgUpdateProfile, MsgAddCapability, MsgRemoveCapability, MsgDeregisterAgent, MsgHeartbeat, MsgUpdateParams |
| disputes | types.go | MsgOpenDispute, MsgSubmitEvidence, MsgVoteDispute, MsgUpdateParams |
| gov | types.go | MsgAttachUpgradePlan, MsgUpdateParams |
| home | types.go | MsgUpdateParams |
| ibcratelimit | types.go | MsgUpdateParams |
| icaauth | types.go | MsgRegisterInterchainAccount, MsgSendICATransaction, MsgUpdateParams |
| knowledge | types.go | MsgUpdateExtendedParams |
| ontology | types.go | MsgProposeDomain, MsgVoteDomainProposal, MsgUpdateDomain, MsgUpdateParams, MsgRegisterLogicZone, MsgAcknowledgeIncompleteness |
| partnerships | types.go | MsgInitiatePartnership, MsgAcceptPartnership, MsgDepositToPot, MsgDistributeReward, MsgProposeOperation, MsgApproveOperation, MsgRejectOperation, MsgSafetyFreeze, MsgRaiseCoercionSignal, MsgInitiateExit, MsgUpdateParams |
| qualification | types.go | MsgRequestQualification, MsgEndorseValidator, MsgRenewQualification, MsgWithdrawQualification, MsgUpdateParams |
| schedule | types.go | MsgCreateSchedule, MsgPauseSchedule, MsgResumeSchedule, MsgCancelSchedule, MsgAddFee, MsgUpdateParams |
| staking | types.go | MsgUpdateValidatorStake, MsgDelegate, MsgUndelegate, MsgRedelegate, MsgUpdateParams |
| tokens | types.go | MsgUpdateParams |
| tree | types.go | MsgUpdateParams |

## Fix

For each hand-written message type, add `XXX_MessageName()` returning a unique fully-qualified name:

```go
func (msg *MsgUpdateParams) XXX_MessageName() string {
    return "legible.<module>.v1.MsgUpdateParams"
}
```

**IMPORTANT:** Do NOT touch `.pb.go` files — those already have proper proto names from codegen.

Some modules already have `proto_names.go` files (from a prior partial fix attempt by delta-atlas) covering a few modules: autopoiesis, disputes, discovery, emergency, staking, billing. Check those first — they may already have the needed methods, but the types they cover may not match the full set needed.

## Verification

```bash
go build -o legbled ./cmd/legbled && ./legbled version
```

Should print version info without panicking. Then:

```bash
go test ./x/... -count=1
```

All existing tests should still pass.
