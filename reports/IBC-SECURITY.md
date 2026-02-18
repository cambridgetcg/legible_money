# IBC Security Audit Report — Rate Limiting & Interchain Accounts

**Date**: 2026-02-18
**Branch**: `fix/phase-d-proto-json-agent-sdk`
**Scope**: `x/ibcratelimit` (319 lines middleware + 141 lines state + 98 lines keeper)
         `x/icaauth` (231 lines keeper + 202 lines types + 136 lines msg_server)
**Auditor**: Claude Code (PP-Lite [C] tier)

---

## Executive Summary

| Metric | ibcratelimit | icaauth |
|--------|-------------|---------|
| Test files | 2 | 1 |
| Tests | 31 (3 fail — build) | 24 (all pass) |
| P0 findings | **3** | **3** |
| P1 findings | 4 | 3 |
| P2 findings | 2 | 1 |

**Overall Assessment**: The architecture is sound and follows established Cosmos SDK IBC
middleware patterns (similar to Osmosis's rate limiter). However, there are **6 P0
security findings** across the two modules — 3 in rate limiting bypass attack surface,
3 in ICA authentication gaps.

---

## Section 1: Rate Limiting Architecture

### 1.1 Architecture Diagram

```
                            OUTBOUND PATH
┌──────────────┐    ┌──────────────┐    ┌────────────────┐    ┌──────────────┐
│  Transfer    │───▶│  IBCFee      │───▶│  RateLimit     │───▶│  IBC Core    │
│  Module      │    │  Keeper      │    │  ICS4Wrapper   │    │  Channel     │
│  (SendPacket)│    │  (ICS4)      │    │  (outflow chk) │    │  Keeper      │
└──────────────┘    └──────────────┘    └────────────────┘    └──────────────┘

                            INBOUND PATH
┌──────────────┐    ┌────────────────┐    ┌──────────────┐    ┌──────────────┐
│  IBC Core    │───▶│  RateLimit     │───▶│  IBCFee      │───▶│  Transfer    │
│  Router      │    │  IBCMiddleware │    │  Middleware   │    │  Module      │
│              │    │  (inflow chk)  │    │              │    │  (OnRecvPkt) │
└──────────────┘    └────────────────┘    └──────────────┘    └──────────────┘

                         ICA PATH (NO RATE LIMITING!)
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  ICA         │───▶│  IBC Core    │───▶│  ICA Host    │
│  Controller  │    │  Channel     │    │  (remote)    │
│  (direct)    │    │  Keeper      │    │              │
└──────────────┘    └──────────────┘    └──────────────┘
```

**Key observations**:
- Rate limiting wraps the **transfer module** path only (app.go:1038)
- ICA controller and ICA host are routed **directly** to IBC core (app.go:1042-1045)
- The ICA controller's ICS4Wrapper is `app.IBCKeeper.ChannelKeeper` — NOT the rate limit wrapper

### 1.2 Rate Limit Parameters

| Parameter | Default | Description | Governance-changeable |
|-----------|---------|-------------|----------------------|
| `EpochLengthBlocks` | 34,272 | ~1 day at 2521ms blocks | Yes (MsgUpdateParams) |
| `MaxOutflowPerEpoch` | "10000000000" | 10,000 LGM (in ulgm) | Yes |
| `MaxInflowPerEpoch` | "10000000000" | 10,000 LGM (in ulgm) | Yes |
| `TrackedDenom` | "ulgm" | Only this denom is rate-limited | Yes |

### 1.3 What Is Rate-Limited

| Direction | Hook Point | Behavior |
|-----------|-----------|----------|
| **Outbound** (SendPacket) | `ICS4Wrapper.SendPacket` (middleware.go:250) | Pre-send check: rejects if cumulative outflow > limit |
| **Outbound** (track) | `OnAcknowledgementPacket` (middleware.go:173) | Post-ack: records confirmed outflow on success ack only |
| **Inbound** | `OnRecvPacket` (middleware.go:115) | Pre-recv check: returns error ack if cumulative inflow > limit |
| **Timeout** | `OnTimeoutPacket` (middleware.go:225) | Passthrough — no tracking (correct: funds returned to sender) |

### 1.4 Sliding Window Implementation

- **Block-based epochs**, NOT time-based. Epoch resets when `currentBlock >= epochStartBlock + epochLengthBlocks` (state.go:117)
- **Per-channel** flow tracking — each IBC channel has independent inflow/outflow counters
- **Lazy reset**: epochs reset on next access via `GetOrResetFlow()`, plus a bulk `ResetExpiredFlows()` in `BeginBlock` (module.go:127-130)
- **Not a sliding window** — it's a fixed-epoch counter. All flow is forgotten at epoch boundary.

---

## Section 2: Rate Limit Bypass Analysis

### P0-1: ICA Transfers Bypass Rate Limits Entirely (CRITICAL)

**Severity**: P0 — Bridge drain attack vector
**Location**: app.go:1042-1045

The ICA controller module is wired directly to IBC core **without** the rate limit middleware:

```go
// app.go:1042 — NO rate limiting on ICA channels
icaControllerIBCModule := icacontroller.NewIBCMiddleware(nil, app.ICAControllerKeeper)
icaHostIBCModule := icahost.NewIBCModule(app.ICAHostKeeper)
ibcRouter.AddRoute(icacontrollertypes.SubModuleName, icaControllerIBCModule)
ibcRouter.AddRoute(icahosttypes.SubModuleName, icaHostIBCModule)
```

**Attack**: A Legible account holder can:
1. Register an ICA on a remote chain via `MsgRegisterInterchainAccount`
2. Use `MsgSendICATransaction` to send `MsgTransfer` back to Legible from the remote chain
3. These incoming transfers arrive on the ICA host channel, not the transfer channel
4. The rate limit middleware only wraps the transfer module route — ICA packets bypass it

**Impact**: Full drain of bridged assets by routing through ICA relay instead of direct transfer.

**Fix**: Either:
- (a) Apply rate limiting to the ICA route as well
- (b) Track ALL incoming transfers at the `x/bank` level regardless of IBC channel
- (c) Rate-limit outflows at the SendPacket level for ALL channels, not just transfer channels

### P0-2: Single-Denom Rate Limiting — All Other Denoms Unprotected

**Severity**: P0 — Design limitation
**Location**: middleware.go:130, middleware.go:269

Only `TrackedDenom` ("ulgm") is rate-limited. All other IBC-enabled tokens pass through without limits:

```go
// middleware.go:130 — only ulgm is checked
if data.Denom != params.TrackedDenom {
    return im.app.OnRecvPacket(ctx, packet, relayer)
}
```

**Impact**: If any secondary token (LGM-20, wrapped assets, LP tokens) is IBC-enabled, they can be drained without limits. The tokens module supports wrapping (`MsgWrapToken`) which creates IBC-compatible denoms.

**Comparison**: Osmosis rate-limits **all denoms** with per-denom configurable limits via a `RateLimitPaths` list. Noble uses per-denom caps. Single-denom is weaker.

**Recommendation**: Extend params to support a list of tracked denoms, or track all denoms with configurable per-denom limits.

### P0-3: Epoch Boundary Race — Instantaneous Reset Doubles Available Capacity

**Severity**: P0 — Timing attack
**Location**: state.go:117-122

The epoch resets **instantly** to zero when the boundary is crossed:

```go
// state.go:117
if currentBlock >= flow.EpochStartBlock+params.EpochLengthBlocks {
    flow.OutflowTotal = "0"
    flow.InflowTotal = "0"
    flow.EpochStartBlock = currentBlock
}
```

**Attack**: An attacker who has used 9,999 LGM of the 10,000 LGM outflow limit can:
1. Wait until block N (last block of epoch)
2. Submit a 1 LGM transfer in block N (uses remaining limit)
3. Submit a 10,000 LGM transfer in block N+1 (new epoch, full limit restored)
4. Net: 10,001 LGM outflow in 2 consecutive blocks

**Impact**: Over any 2-block window straddling an epoch boundary, the effective limit is **2x** the configured limit.

**Comparison**: Osmosis uses a **sliding window** approach where flow decays over time rather than resetting. This prevents the 2x-burst attack.

**Recommendation**: Implement exponential decay or sliding window instead of hard reset. Or reduce the per-epoch limit to account for the 2x worst case.

### P1-1: Multi-Channel Amplification — Limits Are Per-Channel

**Severity**: P1 — Design limitation
**Location**: middleware.go:141 (dest channel), middleware.go:279 (source channel)

Flow tracking uses the **channel ID** as the key. If multiple channels exist to the same chain (or different chains), each has an independent limit:

```go
flow := im.keeper.GetOrResetFlow(ctx, packet.GetDestChannel()) // per-channel
```

**Impact**: With N transfer channels open, the effective limit is N × MaxOutflowPerEpoch. Any user can establish new channels via relayer infrastructure.

**Mitigation**: This is partially mitigated by IBC's one-channel-per-connection model for transfer, but a determined attacker could establish multiple connections to the same chain. Consider adding a **global aggregate limit** across all channels.

### P1-2: Outflow Pre-Check Does Not Track Pending Packets

**Severity**: P1 — Accounting gap
**Location**: middleware.go:250-299

The `ICS4Wrapper.SendPacket` checks the outflow limit but does NOT update the flow record:

```go
// middleware.go:297-298
// Delegate to inner wrapper — the actual outflow tracking is done in OnAcknowledgementPacket
// when the counterparty confirms receipt. Here we only do the pre-send limit check.
return w.channel.SendPacket(...)
```

The actual outflow increment happens later in `OnAcknowledgementPacket` (middleware.go:218). This creates a window where multiple outflow packets can be sent in the same block, all passing the pre-check against the same (stale) outflow total.

**Attack**: In a single block, submit 10 separate 1,000 LGM transfers. Each pre-check sees outflow=0 (no ack received yet), so all 10 pass. When acks arrive, total tracked outflow = 10,000 LGM — equals the limit but wasn't caught.

**Recommendation**: Track "pending outflow" at SendPacket time, not just confirmed outflow. The pre-check should include pending amounts.

### P1-3: Malformed Packet Data Silently Passes Through

**Severity**: P1 — Bypass via encoding
**Location**: middleware.go:124-127, middleware.go:263-265

If packet data fails JSON unmarshalling, the middleware silently passes through:

```go
if err := json.Unmarshal(packet.GetData(), &data); err != nil {
    return im.app.OnRecvPacket(ctx, packet, relayer) // bypass
}
```

While standard ICS-20 transfers always use JSON, a custom IBC module or malformed relayer could craft packets that fail JSON parsing but are accepted by the transfer module via a different encoding path (e.g., protobuf).

**Recommendation**: Log a warning when packet data can't be parsed as transfer data on the transfer port. Consider using the transfer module's official packet data parser.

### P1-4: BeginBlock `ResetExpiredFlows` Iterates All Flows Every Block

**Severity**: P1 — Performance
**Location**: module.go:127-130, state.go:128-141

`ResetExpiredFlows()` is called in every BeginBlock and iterates ALL stored flow records:

```go
func (am AppModule) BeginBlock(goCtx context.Context) error {
    ctx := sdk.UnwrapSDKContext(goCtx)
    am.keeper.ResetExpiredFlows(ctx)
    return nil
}
```

With many IBC channels (or historical channels that were closed but records remain), this becomes O(n) per block. Since `GetOrResetFlow` already handles lazy resets, the BeginBlock sweep is redundant.

**Recommendation**: Remove the BeginBlock sweep and rely entirely on lazy resets in `GetOrResetFlow()`. Or only run the sweep every N blocks.

### P2-1: `big.Int` String Parsing Without Validation

**Severity**: P2 — Defensive coding
**Location**: middleware.go:135-138, middleware.go:273-275

When amount parsing fails, the middleware silently passes through instead of rejecting:

```go
amount, ok := new(big.Int).SetString(data.Amount, 10)
if !ok || amount.Sign() <= 0 {
    return im.app.OnRecvPacket(ctx, packet, relayer) // bypass
}
```

A negative or zero amount shouldn't bypass — it should be explicitly rejected or delegated with logging.

### P2-2: ibcratelimit Test Build Failure — Proto Field Name Regression

**Severity**: P2 — Test regression
**Location**: keeper_test.go:470, 486, 499

Proto codegen changed `ChannelID` to `ChannelId` in `QueryChannelFlowRequest`. Tests still use old field name:

```go
// Current (fails):
qs.ChannelFlow(ctx, &types.QueryChannelFlowRequest{ChannelID: "channel-0"})
// Should be:
qs.ChannelFlow(ctx, &types.QueryChannelFlowRequest{ChannelId: "channel-0"})
```

Fix: 3 locations at lines 470, 486, 499 — change `ChannelID` → `ChannelId`.

---

## Section 3: Interchain Account Authentication

### 3.1 Authentication Flow

```
┌─────────────┐     ┌─────────────────┐     ┌──────────────┐     ┌─────────────┐
│ User signs  │────▶│ Ante Handler    │────▶│ x/icaauth    │────▶│ ICA         │
│ MsgRegister │     │ Chain:          │     │ MsgServer    │     │ Controller  │
│ ICA         │     │ 1. SigVerify   │     │              │     │ Keeper      │
│             │     │ 2. EmergHalt   │     │ - Legible    │     │ (ibc-go)    │
│             │     │ 3. LgmAccount  │     │   account?   │     │             │
│             │     │ 4. Capability  │     │ - DID valid? │     │             │
│             │     │    (deny sess) │     │ - Max accts? │     │             │
│             │     │ 5. DIDDecorator│     │ - Duplicate? │     │             │
└─────────────┘     └─────────────────┘     └──────────────┘     └─────────────┘
```

### 3.2 Who Can Register

| Requirement | Location | Enforced |
|-------------|----------|----------|
| Valid Cosmos signature | ante.go (SigVerification) | Yes |
| Non-frozen Legible account | ante_legible.go:135 | Yes |
| Operational key (NOT session key) | ante_legible.go:274 | Yes |
| Valid DID on account | msg_server.go:35 | Yes |
| Under max accounts limit | msg_server.go:42 | Yes |
| No duplicate connection | msg_server.go:47 | Yes |

### 3.3 What ICAs Can Do

| Message Type | Allowed by Default | Source |
|--------------|-------------------|--------|
| `/cosmos.bank.v1beta1.MsgSend` | Yes | genesis.go:8 |
| `/cosmos.staking.v1beta1.MsgDelegate` | Yes | genesis.go:9 |
| `/cosmos.staking.v1beta1.MsgUndelegate` | Yes | genesis.go:10 |
| `/cosmos.gov.v1.MsgVote` | Yes | genesis.go:11 |
| `/ibc.applications.transfer.v1.MsgTransfer` | Yes | genesis.go:12 |

### 3.4 Session Key Restrictions

The `LegibleCapabilityDecorator` (ante_legible.go:268-276) enforces:
- Session keys with `CanIca=true` can only call `SendICATransaction`
- `RegisterInterchainAccount` is **always denied** for session keys — operational key required
- This is a strong security control: ICA registration (a privileged operation) requires the full key

---

## Section 4: ICA Security Findings

### P0-4: AllowedMsgTypes Per-Account Allowlist Is Not Enforced

**Severity**: P0 — Authorization bypass
**Location**: msg_server.go:89-116 (SendICATransaction)

The `MsgRegisterInterchainAccount` stores an `AllowedMsgTypes` list per account (msg_server.go:69):

```go
remoteAccount := &types.RemoteAccount{
    ...
    AllowedMsgTypes: msg.AllowedMsgs,  // stored but never checked
}
```

But `SendICATransaction` **never validates** the `MsgBytes` against this allowlist:

```go
func (ms *msgServer) SendICATransaction(...) (...) {
    // ...
    // Check the ICA exists and is active
    remoteAccount, found := ms.k.GetRemoteAccountByConnection(...)
    if !found { return nil, types.ErrNotRegistered }
    if !remoteAccount.Active { return nil, fmt.Errorf("not yet active") }
    // NO VALIDATION OF msg.MsgBytes AGAINST remoteAccount.AllowedMsgTypes
    return &types.MsgSendICATransactionResponse{}, nil
}
```

**Impact**: Any allowed message type (from the global `AllowedHostMsgTypes` param) can be sent through any ICA, regardless of the per-account allowlist set during registration. The per-account restriction is entirely decorative.

**Fix**: Deserialize `msg.MsgBytes` as a `CosmosTx`, iterate messages, check each type URL against `remoteAccount.AllowedMsgTypes`. Reject if any message type is not in the allowlist.

### P0-5: SendICATransaction Does Not Actually Send the Transaction

**Severity**: P0 — Incomplete implementation
**Location**: msg_server.go:107-115

The `SendICATransaction` handler emits an event but **never actually sends the ICA packet**:

```go
func (ms *msgServer) SendICATransaction(...) (...) {
    // ... auth checks ...
    ctx.EventManager().EmitEvent(...)
    return &types.MsgSendICATransactionResponse{}, nil  // <-- just returns, no packet sent
}
```

The handler is missing the actual ICA packet submission call. It should:
1. Deserialize `msg.MsgBytes` into an ICA `CosmosTx`
2. Call the ICA controller keeper's `SendTx` method
3. Return the packet sequence in the response

**Impact**: ICA functionality is non-functional — users can register ICAs but cannot actually use them. This is not a security vulnerability per se (it fails safe), but it means the ICA system is incomplete.

**Note**: The `MsgSendICATransactionResponse.Sequence` field is defined in types.go:101 but always returns 0.

### P0-6: Global AllowedHostMsgTypes Includes MsgTransfer — ICA Rate Limit Bypass

**Severity**: P0 — Links to P0-1
**Location**: genesis.go:12

The default `AllowedHostMsgTypes` includes `/ibc.applications.transfer.v1.MsgTransfer`:

```go
AllowedHostMsgTypes: []string{
    "/cosmos.bank.v1beta1.MsgSend",
    "/cosmos.staking.v1beta1.MsgDelegate",
    "/cosmos.staking.v1beta1.MsgUndelegate",
    "/cosmos.gov.v1.MsgVote",
    "/ibc.applications.transfer.v1.MsgTransfer",  // <-- IBC transfer from ICA
}
```

This means ICAs can initiate IBC transfers from remote chains back to Legible. Combined with P0-1 (ICA channels bypass rate limits), this creates a concrete exploit path for draining bridged assets.

**Recommendation**: Remove `MsgTransfer` from the default allowed types, or ensure ICA-originated transfers are rate-limited.

### P1-5: No Registration Cooldown Enforcement

**Severity**: P1 — Missing validation
**Location**: msg_server.go:25-87 (RegisterInterchainAccount)

The `Params.RegistrationCooldown` (default: 100 blocks ≈ 4 min) is defined and stored but **never checked** during registration:

```go
// genesis.go:14 — parameter exists
RegistrationCooldown: 100,

// msg_server.go:25-87 — but never referenced
func (ms *msgServer) RegisterInterchainAccount(...) {
    // NO cooldown check anywhere in this function
}
```

**Impact**: A user can register ICAs on multiple connections in rapid succession, potentially creating many channels quickly.

**Fix**: Track `LastRegistrationBlock` on the record and check:
```go
if currentBlock < lastRegistration + params.RegistrationCooldown {
    return nil, types.ErrRegistrationCooldown
}
```

### P1-6: No Reverse Index Used for Account Lookup

**Severity**: P1 — Dead code / incomplete implementation
**Location**: keys.go:20-38

The `RemoteAccountPrefix` (0x02) and `RemoteAccountKey()` function are defined but never used:

```go
// keys.go:20 — defined
RemoteAccountPrefix = []byte{0x02}

// keys.go:34 — function defined
func RemoteAccountKey(connectionId, portId string) []byte { ... }
```

No code reads or writes to this prefix. The keeper only uses the forward index (`InterchainAccountPrefix` → DID → record). This means:
- No way to look up which DID owns a given connection+port
- No way for IBC callbacks to update account state without knowing the DID

This is needed for the ICA channel open callback to activate the account (set `Active=true`).

### P1-7: ICA Channel Activation Has No Callback Handler

**Severity**: P1 — Incomplete lifecycle
**Location**: app.go:1042

The ICA controller route is wired with `nil` as the first argument:

```go
icaControllerIBCModule := icacontroller.NewIBCMiddleware(nil, app.ICAControllerKeeper)
```

When `nil` is passed as the app module, the ICA controller has no way to notify `x/icaauth` when a channel opens and the remote address is assigned. This means:
- Accounts are registered with `Active: false`
- There's no automated callback to set `Active: true` and record the remote address
- The `UpdateRemoteAccountAddress` function exists (keeper.go:146) but nothing calls it

**Fix**: Implement an ICA callback handler module and wire it as the first argument to `NewIBCMiddleware`.

### P2-3: `MsgSendICATransaction.TimeoutNanos` Not Validated

**Severity**: P2 — Input validation
**Location**: types.go:82-93

`ValidateBasic()` does not check `TimeoutNanos`:

```go
func (msg MsgSendICATransaction) ValidateBasic() error {
    // ... checks Owner, ConnectionId, MsgBytes
    // NO check on TimeoutNanos — 0 means no timeout, infinite hang
}
```

A zero timeout could cause packets to hang indefinitely. Should validate a minimum timeout.

---

## Section 5: Test Coverage Assessment

### 5.1 ibcratelimit Test Coverage

| Category | Tests | Status | Notes |
|----------|-------|--------|-------|
| OnRecvPacket — within limit | 1 | PASS | |
| OnRecvPacket — exceeds limit | 1 | PASS | |
| OnRecvPacket — non-tracked denom | 1 | PASS | |
| OnRecvPacket — non-transfer packet | 1 | PASS | |
| OnRecvPacket — cumulative inflow | 1 | PASS | |
| SendPacket — within limit | 1 | PASS | |
| SendPacket — exceeds limit | 1 | PASS | |
| SendPacket — non-tracked denom | 1 | PASS | |
| SendPacket — non-transfer data | 1 | PASS | |
| OnAckPacket — success tracks outflow | 1 | PASS | |
| OnAckPacket — inner module error | 1 | PASS | |
| OnAckPacket — malformed ack | 1 | PASS | |
| Channel lifecycle delegation | 2 | PASS | |
| Params CRUD | 3 | PASS | |
| Flow CRUD | 4 | PASS | |
| Epoch management | 4 | PASS | |
| Genesis init/export | 2 | PASS | |
| MsgServer (UpdateParams) | 3 | PASS | |
| QueryServer | 3 | **BUILD FAIL** | `ChannelID` → `ChannelId` regression |
| **Total** | **31** | **28 pass, 3 build fail** | |

### 5.2 icaauth Test Coverage

| Category | Tests | Status | Notes |
|----------|-------|--------|-------|
| Record CRUD | 2 | PASS | |
| AddRemoteAccount | 3 | PASS | |
| GetRemoteAccountByConnection | 2 | PASS | |
| UpdateRemoteAccountAddress | 1 | PASS | |
| Params CRUD | 3 | PASS | |
| Genesis init/export | 2 | PASS | |
| MsgRegister — success | 1 | PASS | |
| MsgRegister — no account | 1 | PASS | |
| MsgRegister — max reached | 1 | PASS | |
| MsgRegister — duplicate | 1 | PASS | |
| MsgRegister — controller error | 1 | PASS | |
| MsgSendICA — success | 1 | PASS | |
| MsgSendICA — no account | 1 | PASS | |
| MsgSendICA — not registered | 1 | PASS | |
| MsgSendICA — not active | 1 | PASS | |
| QueryServer | 3 | PASS | |
| **Total** | **24** | **All pass** | |

### 5.3 Missing Test Coverage

| Gap | Priority | Notes |
|-----|----------|-------|
| Rate limit boundary — exactly at limit | High | Test with amount == limit exactly |
| Rate limit — epoch reset timing | High | Test transfers straddling epoch boundary |
| Rate limit — concurrent outflow in same block | High | Multiple SendPacket in single block |
| ICA + rate limit integration | **Critical** | No test verifying ICA bypasses rate limits |
| ICA activation callback flow | High | No test for channel-open → Active=true |
| ICA AllowedMsgTypes enforcement | High | No test since enforcement is missing |
| Session key → ICA registration denied | Medium | Tested in ante handler, not in icaauth |
| Multi-channel rate amplification | Medium | Multiple channels to same chain |
| Negative/zero amount handling | Low | Ensure consistent reject behavior |

### 5.4 No Integration Tests

There are **zero** cross-module integration tests combining IBC + rate limiting + ICA in `tests/cross_stack/`. This is the highest-priority test gap.

---

## Section 6: Comparison with Other Chain Rate Limiters

### Osmosis Rate Limiting (ibc-rate-limit)

| Feature | Osmosis | Legible | Notes |
|---------|---------|---------|-------|
| Tracked denoms | **All denoms** (per-path config) | Single denom only | Osmosis much stronger |
| Limit type | % of channel value | Absolute amount | Osmosis adapts to total value |
| Window type | **Sliding window** (gradual decay) | Fixed epoch (hard reset) | Osmosis prevents 2x burst |
| Per-channel | Yes + **global aggregate** | Per-channel only | Osmosis prevents multi-channel amplification |
| SendPacket tracking | Pre-send + pending tracking | Pre-check only (no pending) | Osmosis prevents same-block burst |
| ICA coverage | N/A (separate concern) | **Not covered** | Both leave ICA unprotected |
| CosmWasm integration | Contract-based | Native Go | Legible is simpler/faster |

### Noble Rate Limiting

| Feature | Noble | Legible | Notes |
|---------|-------|---------|-------|
| Per-denom caps | Yes | No | Noble covers all assets |
| Minting rate limit | Yes (USDC-specific) | No | N/A for Legible |

### Recommendations from Comparison

1. **Multi-denom tracking** — follow Osmosis pattern with per-denom configurable limits
2. **Sliding window** — replace fixed epoch with gradual decay to prevent 2x burst
3. **Pending outflow tracking** — track at SendPacket time, not just on ack
4. **Global aggregate** — add cross-channel limit in addition to per-channel

---

## Section 7: Summary of Findings

### P0 — Must Fix Before Production

| ID | Module | Finding | Impact |
|----|--------|---------|--------|
| P0-1 | ibcratelimit | ICA transfers bypass rate limits entirely | Bridge drain via ICA relay |
| P0-2 | ibcratelimit | Single-denom limiting — all other denoms unprotected | Drain of non-ulgm bridged assets |
| P0-3 | ibcratelimit | Epoch boundary instant reset doubles capacity | 2x burst at epoch transitions |
| P0-4 | icaauth | AllowedMsgTypes per-account allowlist never enforced | Per-account restrictions decorative |
| P0-5 | icaauth | SendICATransaction doesn't actually send the packet | ICA non-functional (fails safe) |
| P0-6 | icaauth | Default AllowedHostMsgTypes includes MsgTransfer | Concrete ICA→rate limit bypass path |

### P1 — Should Fix Before Production

| ID | Module | Finding |
|----|--------|---------|
| P1-1 | ibcratelimit | Multi-channel amplification (per-channel limits) |
| P1-2 | ibcratelimit | Outflow pre-check doesn't track pending packets |
| P1-3 | ibcratelimit | Malformed packet data silently passes through |
| P1-4 | ibcratelimit | BeginBlock iterates all flows every block |
| P1-5 | icaauth | Registration cooldown defined but not enforced |
| P1-6 | icaauth | Reverse index (RemoteAccountPrefix) never used |
| P1-7 | icaauth | No ICA channel activation callback handler |

### P2 — Nice to Have

| ID | Module | Finding |
|----|--------|---------|
| P2-1 | ibcratelimit | Negative/zero amount parsing bypasses silently |
| P2-2 | ibcratelimit | Test build failure — `ChannelID` → `ChannelId` |
| P2-3 | icaauth | `TimeoutNanos` not validated in ValidateBasic |

---

## Section 8: Recommended Fix Priority

### Immediate (Block Release)

1. **P0-5**: Complete `SendICATransaction` implementation or disable the endpoint
   - Without this, ICA is non-functional anyway — P0-1/P0-4/P0-6 are theoretical until this is fixed
   - But P0-1 must be addressed BEFORE P0-5 is fixed, or fixing P0-5 creates the drain vector

2. **P0-1**: Wire rate limiting on ICA channels OR remove `MsgTransfer` from allowed ICA types
   - Fastest fix: Remove `/ibc.applications.transfer.v1.MsgTransfer` from default `AllowedHostMsgTypes`
   - Proper fix: Implement rate limiting at a lower level (bank module hooks)

3. **P2-2**: Fix 3 test field name regressions in ibcratelimit keeper_test.go

### Before Mainnet

4. **P0-2**: Multi-denom rate limiting
5. **P0-3**: Sliding window or decay-based epoch transitions
6. **P0-4**: Enforce AllowedMsgTypes in SendICATransaction
7. **P1-2**: Track pending outflow at SendPacket time
8. **P1-5**: Enforce registration cooldown
9. **P1-7**: Implement ICA channel activation callback

### Deferred (Design Decisions Needed)

10. **P1-1**: Global aggregate rate limit (needs economic modeling)
11. **P1-4**: BeginBlock optimization (lazy reset may be sufficient)
12. **P1-6**: Reverse index for ICA account lookup

---

## Appendix A: File Index

| File | Lines | Purpose |
|------|-------|---------|
| `x/ibcratelimit/keeper/middleware.go` | 319 | IBCMiddleware + ICS4Wrapper |
| `x/ibcratelimit/keeper/state.go` | 141 | Flow CRUD + epoch management |
| `x/ibcratelimit/keeper/keeper.go` | 98 | Keeper + genesis + params |
| `x/ibcratelimit/keeper/msg_server.go` | 48 | UpdateParams handler |
| `x/ibcratelimit/keeper/grpc_query.go` | 46 | Params + ChannelFlow queries |
| `x/ibcratelimit/module.go` | 137 | Module wiring + BeginBlock |
| `x/ibcratelimit/types/types.go` | 86 | Params, MsgUpdateParams |
| `x/ibcratelimit/types/keys.go` | 28 | Store key prefixes |
| `x/ibcratelimit/types/errors.go` | 9 | Error sentinels |
| `x/ibcratelimit/keeper/middleware_test.go` | 487 | Middleware tests (15) |
| `x/ibcratelimit/keeper/keeper_test.go` | 503 | Keeper tests (16, 3 fail) |
| `x/icaauth/keeper/keeper.go` | 231 | Keeper + state ops + genesis |
| `x/icaauth/keeper/msg_server.go` | 136 | Register + Send + UpdateParams |
| `x/icaauth/keeper/grpc_query.go` | 40 | Accounts + Params queries |
| `x/icaauth/module.go` | 121 | Module wiring |
| `x/icaauth/types/types.go` | 202 | Msg types + gRPC descriptors |
| `x/icaauth/types/genesis.go` | 33 | Default params + genesis |
| `x/icaauth/types/expected_keepers.go` | 21 | Keeper interfaces |
| `x/icaauth/types/keys.go` | 39 | Store key prefixes |
| `x/icaauth/types/errors.go` | 13 | Error sentinels |
| `x/icaauth/keeper/keeper_test.go` | 700 | Full keeper test suite (24) |
| `app/app.go` | ~1700 | Module wiring (IBC routes: 1031-1049) |
| `app/ante_legible.go` | 397 | Session key capability enforcement |

## Appendix B: IBC Middleware Stack (app.go)

```
Keeper creation order (app.go:640-712):
  1. IBCKeeper (640)
  2. IBCRateLimitKeeper (652)
  3. rateLimitICS4Wrapper = NewICS4Wrapper(ChannelKeeper, IBCRateLimitKeeper) (659)
  4. IBCFeeKeeper(ics4Wrapper=rateLimitICS4Wrapper) (664)
  5. TransferKeeper(ics4Wrapper=IBCFeeKeeper) (674)
  6. ICAControllerKeeper(channelKeeper=ChannelKeeper) (689) ← NOT rate-limited
  7. ICAHostKeeper (701)

IBC Router (app.go:1031-1049):
  transfer → rateLimitMiddleware(feeMiddleware(transferModule))
  icacontroller → icaControllerMiddleware(nil)  ← NO rate limiting
  icahost → icaHostModule  ← NO rate limiting
```
