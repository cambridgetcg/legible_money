# TYPE-REG: Interface Registration Gaps Fixed

**Date**: 2026-02-18
**Modules**: partnerships, schedule, ontology
**Status**: All fixes applied, builds clean, CLI verified

---

## Fix 1: x/partnerships/types/codec.go

### Files Changed
- `x/partnerships/types/codec.go`

### Before
```go
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
	_ = registry
}
```

### After
```go
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgInitiatePartnership{},
		&MsgAcceptPartnership{},
		&MsgDepositToPot{},
		&MsgDistributeReward{},
		&MsgProposeOperation{},
		&MsgApproveOperation{},
		&MsgRejectOperation{},
		&MsgSafetyFreeze{},
		&MsgRaiseCoercionSignal{},
		&MsgInitiateExit{},
		&MsgUpdateParams{},
	)
}
```

Added `sdk "github.com/cosmos/cosmos-sdk/types"` to imports.

### Types Registered (11)
1. MsgInitiatePartnership
2. MsgAcceptPartnership
3. MsgDepositToPot
4. MsgDistributeReward
5. MsgProposeOperation
6. MsgApproveOperation
7. MsgRejectOperation
8. MsgSafetyFreeze
9. MsgRaiseCoercionSignal
10. MsgInitiateExit
11. MsgUpdateParams

### Types NOT Registered (5) — Reason: No MsgServer handler, not in RegisterCodec
- MsgInitiateSeedPartnership
- MsgGraduateSeedPartnership
- MsgRegisterInPool
- MsgDeclareIntent
- MsgSponsorPartnership

These types exist in types.go but have no corresponding handler in msg_server.go and are not registered in RegisterCodec. They appear to be future/placeholder types.

---

## Fix 2: x/schedule/types/codec.go

### Files Changed
- `x/schedule/types/codec.go`

### Before
```go
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
	_ = registry
}
```

### After
```go
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateSchedule{},
		&MsgPauseSchedule{},
		&MsgResumeSchedule{},
		&MsgCancelSchedule{},
		&MsgAddFee{},
		&MsgUpdateParams{},
	)
}
```

Added `sdk "github.com/cosmos/cosmos-sdk/types"` to imports.

### Types Registered (6)
1. MsgCreateSchedule
2. MsgPauseSchedule
3. MsgResumeSchedule
4. MsgCancelSchedule
5. MsgAddFee
6. MsgUpdateParams

---

## Fix 3: x/ontology — Extract BankKeeper to types/expected_keepers.go

### Files Changed
- `x/ontology/types/expected_keepers.go` (NEW)
- `x/ontology/keeper/keeper.go` (MODIFIED)

### New File: x/ontology/types/expected_keepers.go
```go
package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank keeper interface for the ontology module.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}
```

### keeper.go Changes
- Removed local `BankKeeper` interface definition (was inline in keeper package)
- Changed `bankKeeper BankKeeper` → `bankKeeper types.BankKeeper` in Keeper struct and NewKeeper
- Removed unused `"context"` import

---

## Build Verification

```
$ go build ./x/partnerships/...   # clean
$ go build ./x/schedule/...       # clean
$ go build ./x/ontology/...       # clean
$ go build ./...                  # clean (full build)
$ go build -o /tmp/legbled ./cmd/legbled/   # binary built
```

## CLI Verification

### `legbled tx partnerships --help`
```
Available Commands:
  accept            Accept a pending partnership formation
  approve-operation Approve a pending consensus operation
  deposit           Deposit funds into a partnership common pot
  distribute-reward Distribute a reward through a partnership
  initiate          Initiate a partnership formation
  initiate-exit     Initiate the exit/dissolution process
  propose-operation Propose a consensus operation
  raise-coercion    Raise a coercion/duress flag
  reject-operation  Reject a pending operation with optional counter-proposal
  safety-freeze     Trigger a unilateral safety freeze
```

10 subcommands listed (UpdateParams is governance-only, not exposed as CLI subcommand).

### `legbled tx schedule --help`
```
Available Commands:
  add-fee         Add prepaid fee to an existing schedule
  cancel-schedule Cancel a scheduled process and refund remaining fee
  create-schedule Create a new scheduled process
  pause-schedule  Pause an active scheduled process
  resume-schedule Resume a paused scheduled process
```

5 subcommands listed (UpdateParams is governance-only, not exposed as CLI subcommand).

---

## Pattern Reference

The `RegisterImplementations` approach was validated by 5 existing modules in the codebase that already use it with hand-written (non-proto-generated) types:
- `x/channels/types/codec.go`
- `x/tokens/types/codec.go`
- `x/claiming_pot/types/codec.go`
- `x/icaauth/types/codec.go`
- `x/bvm/types/codec.go`

Both partnerships and schedule already had hand-written gRPC service descriptors (`_Msg_serviceDesc`) and `RegisterMsgServer` in their types.go files — the missing piece was `RegisterInterfaces` for the `InterfaceRegistry` path used by `ProtoCodec` / CLI tx encoding.
