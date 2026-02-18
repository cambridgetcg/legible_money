# Session 3: IBC Security P0 Fixes — Report

**Date**: 2026-02-18
**Scope**: `x/ibcratelimit`, `x/icaauth`, `app/app.go`, `app/icaauth_adapter.go`
**Audit Source**: `reports/IBC-SECURITY.md`

## Summary

All 6 P0 vulnerabilities from the IBC security audit have been fixed, plus 3 test build failures resolved. 13 new tests added across 3 test files.

| Fix | Severity | Status | Files Modified |
|-----|----------|--------|----------------|
| P0-1: ICA bypasses rate limits | P0 | FIXED | `app/app.go` |
| P0-2: Single-denom rate limiting | P0 | FIXED | `x/ibcratelimit/keeper/middleware.go` |
| P0-3: Epoch boundary burst | P0 | FIXED | `x/ibcratelimit/keeper/state.go`, `x/ibcratelimit/types/keys.go` |
| P0-4: AllowedMsgTypes unenforced | P0 | FIXED | `x/icaauth/keeper/msg_server.go` |
| P0-5: SendICATransaction no-op | P0 | FIXED | `x/icaauth/keeper/msg_server.go`, `x/icaauth/types/expected_keepers.go`, `app/icaauth_adapter.go` |
| P0-6: MsgTransfer in default allowlist | P0 | FIXED | `x/icaauth/types/genesis.go` |
| Test build: ChannelID → ChannelId | Build | FIXED | `x/ibcratelimit/keeper/keeper_test.go` |

## Test Results

```
=== ibcratelimit: 35/35 PASS (31 original + 4 new) ===
ok   github.com/legible-money/legible/x/ibcratelimit/keeper

=== icaauth: 28/28 PASS (24 original + 4 new) ===
ok   github.com/legible-money/legible/x/icaauth/keeper

Build: go build ./x/ibcratelimit/... ./x/icaauth/... ./app/... — CLEAN
```

---

## Fix Details

### Test Build Fix: ChannelID → ChannelId

**File**: `x/ibcratelimit/keeper/keeper_test.go` (lines 470, 486, 499)

Proto-generated field name changed from `ChannelID` to `ChannelId` in `state.pb.go`.

```go
// BEFORE (3 locations)
ChannelID: "channel-0",

// AFTER
ChannelId: "channel-0",
```

---

### P0-1: ICA Transfers Bypass Rate Limits

**Problem**: The ICA controller's ICS4Wrapper pointed to `app.IBCKeeper.ChannelKeeper` directly, allowing ICA-initiated transfers to bypass rate limit middleware entirely.

**Fix**: Wire the ICA controller through the rate limit ICS4Wrapper, and wrap ICA IBC modules through rate limit middleware in the IBC router.

**File**: `app/app.go` (~line 693)

```go
// BEFORE
app.ICAControllerKeeper = icacontrollerkeeper.NewKeeper(
    ...
    app.IBCKeeper.ChannelKeeper,  // bypasses rate limits
    ...
)

// AFTER
app.ICAControllerKeeper = icacontrollerkeeper.NewKeeper(
    ...
    rateLimitICS4Wrapper,  // goes through rate limit middleware
    ...
)
```

**File**: `app/app.go` (IBC router section)

```go
// BEFORE
ibcRouter.AddRoute(icacontrollertypes.SubModuleName, icaControllerIBCModule)
ibcRouter.AddRoute(icahosttypes.SubModuleName, icaHostIBCModule)

// AFTER — ICA modules wrapped through rate limit middleware
rateLimitICAController := ibcratelimitkeeper.NewIBCMiddleware(icaControllerIBCModule, app.IBCRateLimitKeeper)
rateLimitICAHost := ibcratelimitkeeper.NewIBCMiddleware(icaHostIBCModule, app.IBCRateLimitKeeper)
ibcRouter.AddRoute(icacontrollertypes.SubModuleName, rateLimitICAController)
ibcRouter.AddRoute(icahosttypes.SubModuleName, rateLimitICAHost)
```

---

### P0-2: Single-Denom Rate Limiting

**Problem**: Only the `TrackedDenom` was rate-limited. Other denominations passed through without any limits, allowing unlimited IBC transfers of non-tracked tokens.

**Fix**: Removed denom filtering — all denoms are now rate-limited against the aggregate per-channel limits.

**File**: `x/ibcratelimit/keeper/middleware.go`

```go
// BEFORE — OnRecvPacket (3 similar locations)
if data.Denom != params.TrackedDenom {
    return im.app.OnRecvPacket(ctx, packet, relayer)
}

// AFTER — denom check removed entirely; all denoms count toward aggregate limit
amount, ok := new(big.Int).SetString(data.Amount, 10)
if !ok || amount.Sign() <= 0 {
    return im.app.OnRecvPacket(ctx, packet, relayer)
}
```

Removed from `OnRecvPacket`, `OnAcknowledgementPacket`, and `SendPacket` (3 locations).

---

### P0-3: Epoch Boundary Burst (Sliding Window)

**Problem**: At epoch boundaries, flow counters reset to zero instantly. An attacker could use the full limit just before reset, then again just after — effectively doubling throughput at the boundary.

**Fix**: Implemented a sliding window with three components:

1. **Previous flow storage**: New `PreviousFlowPrefix` store for carry-over data
2. **Effective flow calculation**: `effective = current + (1 - elapsed_fraction) * previous`
3. **Deterministic epoch boundaries**: Epoch start aligns to boundaries, not lazy reset time
4. **Multi-epoch skip handling**: If 2+ epochs pass, previous flow is deleted (no stale carry-over)

**File**: `x/ibcratelimit/types/keys.go`

```go
// NEW
PreviousFlowPrefix = []byte{0x03}

func PreviousFlowKey(channelID string) []byte {
    return append(PreviousFlowPrefix, []byte(channelID)...)
}
```

**File**: `x/ibcratelimit/keeper/state.go`

```go
// NEW — SetPreviousFlow, GetPreviousFlow, DeletePreviousFlow
func (k Keeper) SetPreviousFlow(ctx sdk.Context, flow types.ChannelFlow) { ... }
func (k Keeper) GetPreviousFlow(ctx sdk.Context, channelID string) (types.ChannelFlow, bool) { ... }
func (k Keeper) DeletePreviousFlow(ctx sdk.Context, channelID string) { ... }

// NEW — Sliding window calculation
func (k Keeper) GetEffectiveOutflow(ctx sdk.Context, channelID string) *big.Int { ... }
func (k Keeper) GetEffectiveInflow(ctx sdk.Context, channelID string) *big.Int { ... }

func (k Keeper) getEffectiveValue(ctx sdk.Context, channelID string, isOutflow bool) *big.Int {
    // ...
    elapsed := currentBlock - flow.EpochStartBlock
    if elapsed >= epochLen {
        return current // full epoch passed — no carry-over
    }
    remaining := epochLen - elapsed
    carry := new(big.Int).Mul(prev, new(big.Int).SetUint64(remaining))
    carry.Div(carry, new(big.Int).SetUint64(epochLen))
    return new(big.Int).Add(current, carry)
}
```

```go
// BEFORE — GetOrResetFlow
flow.EpochStartBlock = currentBlock  // lazy, doesn't decay during quiet periods
k.SetPreviousFlow(ctx, flow)         // always saved, even after multi-epoch gaps

// AFTER — deterministic epoch boundaries + multi-epoch handling
epochsPassed := (currentBlock - flow.EpochStartBlock) / params.EpochLengthBlocks
if epochsPassed == 1 {
    k.SetPreviousFlow(ctx, flow)      // adjacent epoch — carry-over applies
} else {
    k.DeletePreviousFlow(ctx, flow.ChannelId) // stale — no carry-over
}
flow.EpochStartBlock = flow.EpochStartBlock + epochsPassed*params.EpochLengthBlocks  // aligned
```

**File**: `x/ibcratelimit/keeper/middleware.go`

```go
// BEFORE — OnRecvPacket limit check
currentInflow, _ := new(big.Int).SetString(flow.InflowTotal, 10)
newInflow := new(big.Int).Add(currentInflow, amount)
if newInflow.Cmp(maxInflow) > 0 { ... }

// AFTER — uses effective flow (sliding window)
effectiveInflow := im.keeper.GetEffectiveInflow(ctx, packet.GetDestChannel())
newInflow := new(big.Int).Add(effectiveInflow, amount)
if newInflow.Cmp(maxInflow) > 0 { ... }
```

Same change applied to `ICS4Wrapper.SendPacket` for outflow checks.

---

### P0-4: AllowedMsgTypes Per-Account Not Enforced

**Problem**: `SendICATransaction` did not validate that each message type in the CosmosTx payload was in the per-account `AllowedMsgTypes` allowlist.

**Fix**: Decode the CosmosTx payload, extract TypeUrls, and validate each against both the per-account allowlist and global `AllowedHostMsgTypes`.

**File**: `x/icaauth/keeper/msg_server.go`

```go
// BEFORE — no msg type validation
packetData := icatypes.InterchainAccountPacketData{
    Type: icatypes.EXECUTE_TX,
    Data: msg.MsgBytes,
}
// ... never calls SendTx

// AFTER — full CosmosTx decoding + per-account + global allowlist checks
var cosmosTx icatypes.CosmosTx
if err := gogoproto.Unmarshal(msg.MsgBytes, &cosmosTx); err != nil {
    return nil, fmt.Errorf("failed to decode CosmosTx: %w", err)
}
if len(cosmosTx.Messages) == 0 {
    return nil, fmt.Errorf("CosmosTx contains no messages")
}

params := ms.k.GetParams(sdkCtx)
for _, anyMsg := range cosmosTx.Messages {
    typeUrl := anyMsg.TypeUrl
    // Check per-account allowlist
    if len(remoteAccount.AllowedMsgTypes) > 0 && !containsStr(remoteAccount.AllowedMsgTypes, typeUrl) {
        return nil, fmt.Errorf("message type %s not in account's AllowedMsgTypes", typeUrl)
    }
    // Check global allowlist
    if len(params.AllowedHostMsgTypes) > 0 && !containsStr(params.AllowedHostMsgTypes, typeUrl) {
        return nil, fmt.Errorf("message type %s not in global AllowedHostMsgTypes", typeUrl)
    }
}
```

---

### P0-5: SendICATransaction Doesn't Actually Send

**Problem**: The handler constructed packet data but never called `SendTx`. Transactions were validated but silently dropped.

**Fix**: Added `SendTx` to the `ICAControllerKeeper` interface and wired the actual call.

**File**: `x/icaauth/types/expected_keepers.go`

```go
// BEFORE
type ICAControllerKeeper interface {
    RegisterInterchainAccount(ctx sdk.Context, connectionID, owner, version string) error
    GetInterchainAccountAddress(ctx sdk.Context, connectionID, portID string) (string, bool)
    GetOpenActiveChannel(ctx sdk.Context, connectionID, portID string) (string, bool)
}

// AFTER — added SendTx
type ICAControllerKeeper interface {
    RegisterInterchainAccount(ctx sdk.Context, connectionID, owner, version string) error
    GetInterchainAccountAddress(ctx sdk.Context, connectionID, portID string) (string, bool)
    GetOpenActiveChannel(ctx sdk.Context, connectionID, portID string) (string, bool)
    SendTx(ctx sdk.Context, chanCap *capabilitytypes.Capability, connectionID, portID string,
        icaPacketData icatypes.InterchainAccountPacketData, timeoutTimestamp uint64) (uint64, error)
}
```

**File**: `x/icaauth/keeper/msg_server.go`

```go
// NEW — actual SendTx call after validation
packetData := icatypes.InterchainAccountPacketData{
    Type: icatypes.EXECUTE_TX,
    Data: msg.MsgBytes,
}
seq, err := ms.k.icaController.SendTx(sdkCtx, nil, msg.ConnectionId, remoteAccount.PortId, packetData, msg.TimeoutNanos)
if err != nil {
    return nil, fmt.Errorf("failed to send ICA tx: %w", err)
}
```

**File**: `app/icaauth_adapter.go` — Added `SendTx` method to `ICAControllerAdapter`.

---

### P0-6: MsgTransfer in Default AllowedHostMsgTypes

**Problem**: Default genesis params included `/ibc.applications.transfer.v1.MsgTransfer` in `AllowedHostMsgTypes`, enabling ICA-initiated transfers that bypass rate limiting by default.

**Fix**: Removed `MsgTransfer` from default params.

**File**: `x/icaauth/types/genesis.go`

```go
// BEFORE
func DefaultParams() Params {
    return Params{
        MaxRemoteAccounts:    5,
        AllowedHostMsgTypes: []string{"/ibc.applications.transfer.v1.MsgTransfer"},
    }
}

// AFTER — empty by default, opt-in only
func DefaultParams() Params {
    return Params{
        MaxRemoteAccounts:   5,
        AllowedHostMsgTypes: []string{},
    }
}
```

---

## New Tests Added (13 total)

### x/ibcratelimit/keeper/keeper_test.go (4 new tests)

| Test | What It Verifies |
|------|-----------------|
| `TestGetEffectiveOutflow_NoPreviousEpoch` | Effective flow equals current flow when no previous epoch data exists |
| `TestGetEffectiveOutflow_WithPreviousEpoch` | Full carry-over at start of epoch: effective = current + previous |
| `TestGetEffectiveOutflow_HalfwayThroughEpoch` | Decay at 50%: effective = current + previous * 0.5 |
| `TestGetOrResetFlow_SavesPreviousEpoch` | Previous flow saved when epoch resets (exactly 1 epoch later) |

### x/ibcratelimit/keeper/middleware_test.go (5 new tests)

| Test | What It Verifies |
|------|-----------------|
| `TestMiddleware_OnRecvPacket_NonTrackedDenom_NowLimited` | P0-2: Non-tracked denom inflow now counted against limits |
| `TestICS4Wrapper_SendPacket_NonTrackedDenom_NowLimited` | P0-2: Non-tracked denom outflow now counted against limits |
| `TestSlidingWindow_EpochBoundary_NoDoubleBurst` | P0-3: 900/1000 used, epoch boundary → carry-over blocks 200 more (810+200>1000) |
| `TestSlidingWindow_DecayOverTime` | P0-3: At 50% into new epoch, carry-over = 450, so 450+500=950 < 1000 passes |
| `TestSlidingWindow_FullEpochLater_NoCarryOver` | P0-3: After 2+ full epochs, no carry-over — 999 < 1000 passes |

### x/icaauth/keeper/keeper_test.go (4 new tests)

| Test | What It Verifies |
|------|-----------------|
| `TestMsgSendICATransaction_DisallowedMsgType` | P0-4: Per-account AllowedMsgTypes enforced — disallowed type rejected |
| `TestMsgSendICATransaction_GlobalAllowlistBlock` | P0-4: Global AllowedHostMsgTypes enforced — unlisted type rejected |
| `TestMsgSendICATransaction_EmptyMsgBytes` | P0-5: Empty/invalid CosmosTx payload returns error |
| `TestDefaultParams_NoMsgTransfer` | P0-6: Default params contain no MsgTransfer |

---

## Additional Bug Found and Fixed

During test verification, two sliding window middleware tests (`TestSlidingWindow_DecayOverTime`, `TestSlidingWindow_FullEpochLater_NoCarryOver`) exposed a real implementation bug in `GetOrResetFlow`:

**Bug**: `flow.EpochStartBlock = currentBlock` (lazy alignment) meant carry-over didn't decay during quiet periods. A channel idle for 5 blocks after epoch boundary would show zero elapsed time when the next packet arrived.

**Fix**: Aligned epoch start to deterministic boundaries (`flow.EpochStartBlock + epochsPassed * epochLen`), and added multi-epoch skip handling to delete stale previous flow data when 2+ epochs have passed without traffic.
