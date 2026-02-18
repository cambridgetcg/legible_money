# P0-SAFETY: Consensus/Correctness Fixes

**Date**: 2026-02-18

---

## Fix 1: research InitGenesis silently swallows errors

### File
`x/research/module.go`

### Problem
`InitGenesis` at line 107 called `json.Unmarshal(data, &gs)` without checking the error. If genesis JSON is malformed, the module starts with empty state. On a multi-validator network, one node with corrupt genesis proceeds silently while others may not — causing state divergence at the first research-related transaction.

Additionally, `ExportGenesis` (line 114) and `DefaultGenesis` (line 51) had the same pattern: `bz, _ := json.Marshal(gs)` — silently dropping marshal errors.

### Before (3 locations)

**InitGenesis (line 105-109):**
```go
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var gs types.GenesisState
	json.Unmarshal(data, &gs)
	am.keeper.InitGenesis(ctx, gs)
}
```

**ExportGenesis (line 112-116):**
```go
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	bz, _ := json.Marshal(gs)
	return bz
}
```

**DefaultGenesis (line 48-52):**
```go
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	gs := types.DefaultGenesis()
	bz, _ := json.Marshal(gs)
	return bz
}
```

### After

**InitGenesis:**
```go
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var gs types.GenesisState
	if err := json.Unmarshal(data, &gs); err != nil {
		panic(fmt.Sprintf("failed to unmarshal %s genesis: %v", types.ModuleName, err))
	}
	am.keeper.InitGenesis(ctx, gs)
}
```

**ExportGenesis:**
```go
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal %s genesis: %v", types.ModuleName, err))
	}
	return bz
}
```

**DefaultGenesis:**
```go
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	gs := types.DefaultGenesis()
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal default %s genesis: %v", types.ModuleName, err))
	}
	return bz
}
```

### Why
Cosmos SDK convention is to `panic` on genesis failures — the chain must not start with corrupt state. This matches the pattern in `x/tokens/module.go:109-111` and `x/tree/module.go:114-116`. The `research` module was the **only** module in the codebase with this silent swallowing pattern — all 27 other modules already panic on unmarshal failure.

### Verification
Confirmed by grepping all `module.go` files: every other module uses `if err := json.Unmarshal(...); err != nil { panic(...) }` in `InitGenesis`. Research was the sole exception.

---

## Fix 2: staking RegisterInterfaces only registers 1 of 6 Msg types

### File
`x/staking/types/codec.go`

### Problem
`RegisterInterfaces` (line 20-23) only registered `MsgRegisterValidator`. The module defines 6 Msg types that implement `sdk.Msg`:

| Type | Source | Was registered? |
|------|--------|-----------------|
| `MsgRegisterValidator` | `tx.pb.go` (proto-generated) | YES |
| `MsgUpdateValidatorStake` | `types.go` (hand-written) | NO |
| `MsgDelegate` | `types.go` (hand-written) | NO |
| `MsgUndelegate` | `types.go` (hand-written) | NO |
| `MsgRedelegate` | `types.go` (hand-written) | NO |
| `MsgUpdateParams` | `types.go` (hand-written) | NO |

Note: `RegisterCodec` (amino, line 10-17) already registered all 6 types correctly.

The 5 missing `RegisterInterfaces` registrations mean the interface registry doesn't know about these types. This breaks amino/JSON signing for those messages via CLI — users cannot sign delegation, undelegation, redelegation, stake updates, or param updates.

### Before (line 20-24)
```go
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterValidator{},
	)
}
```

### After
```go
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterValidator{},
		&MsgUpdateValidatorStake{},
		&MsgDelegate{},
		&MsgUndelegate{},
		&MsgRedelegate{},
		&MsgUpdateParams{},
	)
}
```

### Why
The types list matches `RegisterCodec` exactly. Reference pattern: `x/tokens/types/codec.go:23-39` and `x/tree/types/codec.go:42-55` both register all their Msg types in `RegisterInterfaces`.

### Verification
All hand-written Msg types implement the required `proto.Message` interface (`ProtoMessage()`, `Reset()`, `String()` methods verified in `types.go`). Build succeeds. CLI confirms all tx subcommands are available.

---

## Build Output

```
$ go build -o /tmp/legbled ./cmd/legbled
(exit 0, no output)
```

**PASS** — clean build, zero errors, zero warnings.

---

## CLI Verification

```
$ /tmp/legbled tx legible_staking --help

Legible staking module transaction commands

Available Commands:
  delegate           Delegate tokens to a validator
  redelegate         Redelegate tokens from one validator to another
  register-validator Register a new validator with PoT consensus
  undelegate         Undelegate tokens from a validator (initiates unbonding)
  update-stake       Increase or decrease validator self-delegation
```

All 5 tx subcommands present. (`update-params` not exposed as a CLI subcommand — it's authority-only, consistent with other modules.)

---

## Related Issues Discovered (Not Fixed)

### 1. `compute_pool` RegisterInterfaces: same pattern as staking (P1)
**File**: `x/compute_pool/types/codec.go:23-27`
`RegisterCodec` registers 9 Msg types. `RegisterInterfaces` only registers `MsgUpdateParams`. The other 8 types (`MsgRegisterComputeProvider`, `MsgUpdateProvider`, `MsgDeregisterProvider`, `MsgProviderHeartbeat`, `MsgRedeemCredits`, `MsgConfirmUsage`, `MsgDisputeUsage`, `MsgUpdateComputeParams`) are missing from interface registration.

### 2. `icaauth` RegisterInterfaces: missing 2 types (P2)
**File**: `x/icaauth/types/codec.go:17-23`
`RegisterCodec` registers `MsgRegisterInterchainAccount`, `MsgSubmitTx`, `MsgUpdateParams`. `RegisterInterfaces` only registers `MsgUpdateParams`. Missing: `MsgRegisterInterchainAccount`, `MsgSubmitTx`.

### 3. Seven modules have empty RegisterInterfaces (P2)
The following modules have `RegisterInterfaces` that registers nothing, with comments like "Hand-written types lack proto typeURLs; registration deferred to proto codegen":
- `billing`, `qualification`, `ibcratelimit`, `ontology`, `partnerships`, `schedule`, `discovery`

These modules have hand-written Msg types that lack proto codegen. Their types ARE registered in `RegisterCodec` (amino) but not in the interface registry. This means JSON/amino signing works via legacy amino but not via the newer `SIGN_MODE_DIRECT` path. Since the chain currently uses hand-written types with `json.Marshal` (not proto codec), this is lower severity — but it will block migration to proto-based signing.

### 4. `alignment` module has no codec.go at all (P3)
**File**: `x/alignment/types/codec.go` exists but `RegisterInterfaces` is a no-op. The module also has no `client/cli/` directory (no CLI commands). The alignment module's Msg types cannot be signed or submitted via CLI.
