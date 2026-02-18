# Tokens Module P1 Cleanup Report

## Fix 1: Add UpdateParams Handler

### Proto Update
**File**: `proto/legible/tokens/v1/tx.proto`

Added `UpdateParams` RPC to the Msg service and `MsgUpdateParams`/`MsgUpdateParamsResponse` message types,
following the pattern from `x/research` and `x/tree`.

```diff
+ import "legible/tokens/v1/genesis.proto";
  service Msg {
    ...
+   rpc UpdateParams(MsgUpdateParams) returns (MsgUpdateParamsResponse);
  }

+ message MsgUpdateParams {
+   option (cosmos.msg.v1.signer) = "authority";
+   string authority = 1;
+   Params params = 2;
+ }
+ message MsgUpdateParamsResponse {}
```

Proto regenerated with `buf generate` -- only `x/tokens/types/tx.pb.go` and `tx_grpc.pb.go` changed.

### Handler Implementation
**File**: `x/tokens/keeper/msg_server.go:776-797`

```go
func (s *msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
    if msg.Authority != s.GetAuthority() {
        return nil, fmt.Errorf("unauthorized: expected %s, got %s", s.GetAuthority(), msg.Authority)
    }
    ctx := sdk.UnwrapSDKContext(goCtx)
    if msg.Params != nil {
        if err := msg.Params.Validate(); err != nil {
            return nil, fmt.Errorf("invalid params: %w", err)
        }
        s.SetParams(ctx, *msg.Params)
    }
    ctx.EventManager().EmitEvent(sdk.NewEvent(
        "tokens_params_updated",
        sdk.NewAttribute("authority", msg.Authority),
    ))
    return &types.MsgUpdateParamsResponse{}, nil
}
```

### Type Registration
**File**: `x/tokens/types/types.go` -- Added `GetSigners()` and `ValidateBasic()` for `MsgUpdateParams`
**File**: `x/tokens/types/codec.go` -- Registered in both `RegisterCodec` (Amino) and `RegisterInterfaces`

### CLI Command
`update-params` does **NOT** have a CLI command. This is expected -- UpdateParams is typically
invoked via governance proposals (`legbled tx gov submit-proposal`), not directly from CLI.
Adding a direct CLI command is a separate task if needed.

---

## Fix 2: Stale Comment

**File**: `x/tokens/keeper/msg_server.go:19-22`

Before:
```go
// Phase 1 implements 8 core LGM-20 handlers; the remaining 6 return Unimplemented.
```

After:
```go
// Phase 1 implements 12 handlers: 8 core LGM-20 token operations (Create, Mint,
// Burn, Transfer, Approve, TransferFrom, Pause, Unpause), 3 liquidity pool
// operations (AddLiquidity, RemoveLiquidity, Swap), and UpdateParams for governance.
// The remaining 3 (DelegatePower, WrapToken, UnwrapToken) return Unimplemented.
```

---

## Fix 3: Phase 2 Stub Documentation

**File**: `x/tokens/keeper/msg_server.go:799-812`

Added block comment documenting the 3 Phase 2 stubs that return `codes.Unimplemented`
via the embedded `UnimplementedMsgServer`:

- **DelegatePower** -- Governance voting power delegation without token transfer (liquid democracy)
- **WrapToken** -- Wraps LGM-20 token into IBC-compatible sdk.Coin denomination
- **UnwrapToken** -- Unwraps IBC-compatible sdk.Coin back to LGM-20 representation

Note: The task description mentioned 6 stubs (DelegatePower, UndelegatePower, WrapToken,
UnwrapToken, SetTokenURI, UpdateTokenMetadata), but only 3 exist in the proto definition.
UndelegatePower, SetTokenURI, and UpdateTokenMetadata are not in the proto service or
generated code -- they would need to be added to the proto before implementation.

---

## Build Verification

```
$ go build ./...
(clean -- no errors)

$ go test ./x/tokens/... -count=1
ok  github.com/legible-money/legible/x/tokens/keeper  1.144s

$ /tmp/legbled tx tokens --help
Available Commands:
  add-liquidity    Add liquidity to a token pair pool
  approve-token    Approve a spender to transfer tokens on your behalf
  burn-token       Burn tokens from your balance
  create-token     Create a new token definition
  mint-token       Mint new tokens to an address
  pause-token      Pause a token (creator only)
  remove-liquidity Remove liquidity from a pool
  swap             Swap tokens through a liquidity pool
  transfer-from    Transfer tokens using an allowance
  transfer-token   Transfer tokens to another address
  unpause-token    Unpause a token (creator only)
```

No `update-params` in CLI (governance-only, as expected).

---

## Files Changed

| File | Change |
|------|--------|
| `proto/legible/tokens/v1/tx.proto` | Added UpdateParams RPC + message types |
| `x/tokens/types/tx.pb.go` | Regenerated (proto) |
| `x/tokens/types/tx_grpc.pb.go` | Regenerated (proto) |
| `x/tokens/keeper/msg_server.go` | UpdateParams handler, comment fix, Phase 2 docs |
| `x/tokens/types/types.go` | GetSigners + ValidateBasic for MsgUpdateParams |
| `x/tokens/types/codec.go` | Registered MsgUpdateParams in Amino + InterfaceRegistry |
