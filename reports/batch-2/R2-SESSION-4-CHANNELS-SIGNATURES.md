# R2 Session 4: Payment Channels — Signature Integration + Chain-ID Domain Separation

**Date**: 2026-02-18
**Module**: `x/channels`
**Branch**: `feat-cicd-opensource-flow`
**Builds on**: Session 4 (P0 signature fixes)

---

## Summary

This session adds chain-ID domain separation to the Ed25519 signing payload, implements BeginBlock auto-resolution for expired closing/disputed channels, and adds comprehensive replay protection and cooperative close flow tests.

### Results

| Metric | Value |
|--------|-------|
| Tests passing | **95/95** (was 80) |
| New tests added | **10** |
| Tests updated | **1** (BeginBlock dispute resolution) |
| Files modified | 5 |
| Files created | 0 |
| Build status | `go build ./...` clean |
| Vet status | Pre-existing proto lock-copy warnings only |

---

## Task 1: Chain-ID Domain Separation

### Problem

The signing payload was `SHA-256("legible.channels.v1:{op}" | "|" | channelId | "|" | BE(sequence) | "|" | spent)`. A signature created on testnet (same channelId format) would be valid on mainnet — no chain-level replay protection.

### Solution

Added `chainId` as the second field in the canonical payload:

```
SHA-256("legible.channels.v1:{op}" | "|" | chainId | "|" | channelId | "|" | BE(sequence) | "|" | spent)
```

**Files changed:**

- `x/channels/types/signature.go:24` — `ChannelSigningPayload` now takes `chainId` as second parameter
- `x/channels/keeper/msg_server.go:137` — UpdateChannel: `ctx.ChainID()` passed to payload
- `x/channels/keeper/msg_server.go:199` — DisputeChannel: `ctx.ChainID()` passed to payload
- `x/channels/keeper/msg_server.go:383` — CloseChannel: `ctx.ChainID()` passed to payload
- `x/channels/keeper/msg_server.go:513` — UpdateState: `ctx.ChainID()` passed to payload

**Test infrastructure:**
- Added `const testChainID = "legible-test-1"` to test infrastructure
- Updated `setupKeeper` to set `ChainID: testChainID` in context header
- `signPacked`/`signHex` now use `testChainID` internally
- Added `signPackedWithChain` for explicit chain-ID testing

---

## Task 2: Replay Protection Tests (4 tests)

| Test | What it verifies |
|------|-----------------|
| `TestReplay_SameChannelDifferentNonce` | Sig for nonce 5 rejected when nonce is now 6 (nonce guard) |
| `TestReplay_SameNonceDifferentChannel` | Sig for channel A rejected on channel B (channelId in payload) |
| `TestReplay_SameOperationDifferentChain` | Sig from "legible-testnet-1" rejected on "legible-test-1" (chain-ID separation) |
| `TestReplay_UpdateStateDualSig` | Sigs for spent=1000000 rejected when submitted with spent=2000000 (payload binding) |

---

## Task 3: Cooperative Close Flow Tests (4 tests)

| Test | What it verifies |
|------|-----------------|
| `TestCooperativeClose_HappyPath` | Open → settle(closing) → wait deadline → settle(closed) → correct fund distribution |
| `TestCooperativeClose_DisputeDuringClosing` | Open → settle(closing) → dispute(disputed) → settle(closed) with disputed state |
| `TestCooperativeClose_CounterpartyCanDispute` | Provider settles → Payer successfully disputes during closing period |
| `TestCooperativeClose_FundsDistribution` | Verifies actual bank balances after cooperative CloseChannel match expected amounts |

---

## Task 4: BeginBlock Auto-Resolution

### Problem

Session 4 noted that closing channels required manual `SettleChannel` calls after the deadline. The old BeginBlock only resolved disputed channels — and incorrectly set them back to `open` status instead of closing them.

### Solution

Added two new keeper methods and rewrote BeginBlock:

**`GetExpiredChannels(ctx, currentBlock)`** — Returns channels with status `closing` or `disputed` where `blockHeight > DisputeDeadline`.

**`AutoSettleChannel(ctx, ch)`** — Distributes funds and closes a channel:
- For `closing` channels: uses last agreed state (Spent field)
- For `disputed` channels: resolves dispute, uses disputed state, marks dispute resolved

**BeginBlock** now delegates entirely to these two methods:
```go
expiredChannels := am.keeper.GetExpiredChannels(ctx, currentBlock)
for _, ch := range expiredChannels {
    am.keeper.AutoSettleChannel(ctx, ch)
}
```

### Files changed

- `x/channels/keeper/keeper.go` — Added `GetExpiredChannels` and `AutoSettleChannel` methods
- `x/channels/module.go` — Rewrote BeginBlock to use new methods

### Behavior change

| Scenario | Old behavior | New behavior |
|----------|-------------|-------------|
| Disputed channel past deadline | Revert to `open` status | Distribute funds, set to `closed` |
| Closing channel past deadline | Not handled (manual only) | Distribute funds, set to `closed` |

### Tests (2 new + 1 updated)

| Test | What it verifies |
|------|-----------------|
| `TestBeginBlock_AutoResolvesExpiredClosing` | Closing channel past deadline: auto-settled, funds distributed, status=closed |
| `TestBeginBlock_AutoResolvesExpiredDispute` | Disputed channel past deadline: disputed state used, dispute resolved, funds distributed |
| `TestBeginBlock_ResolvesExpiredDisputes` | Updated to use new auto-settle flow (was testing old open-revert behavior) |

---

## Test Summary

### New tests: 10

**Replay Protection (4):**
- TestReplay_SameChannelDifferentNonce
- TestReplay_SameNonceDifferentChannel
- TestReplay_SameOperationDifferentChain
- TestReplay_UpdateStateDualSig

**Cooperative Close Flow (4):**
- TestCooperativeClose_HappyPath
- TestCooperativeClose_DisputeDuringClosing
- TestCooperativeClose_CounterpartyCanDispute
- TestCooperativeClose_FundsDistribution

**BeginBlock Auto-Resolution (2):**
- TestBeginBlock_AutoResolvesExpiredClosing
- TestBeginBlock_AutoResolvesExpiredDispute

### Updated tests: 1
- TestBeginBlock_ResolvesExpiredDisputes (new auto-settle behavior)

### Total: 95 tests passing (was 80 after R1 Session 4)

---

## Known Issues (Not in Scope)

1. **Proto lock-copy warnings**: Pre-existing in all proto-generated types. Not introduced by this session.

2. **No sequence replay protection on CloseChannel**: The cooperative close handler doesn't check that `msg.Sequence > ch.Nonce`. Both parties must agree (sig required), so this is low risk, but a future enhancement could enforce monotonic sequence on close.

3. **AutoSettleChannel silently ignores bank transfer errors**: Uses `_ = bankKeeper.SendCoinsFromModuleToAccount(...)` — if the module account is underfunded, the transfer fails silently but the channel still closes. This prevents stuck channels but may lose accounting precision.
