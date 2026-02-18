# Session 3: IBC Security P0 Fixes (Rate Limiting + ICA Auth)

## Context
You are working on Legible Money (`/Users/yuai/Desktop/legible_money`), a Cosmos SDK blockchain with IBC rate limiting (`x/ibcratelimit`) and interchain account auth (`x/icaauth`).

An audit in `reports/IBC-SECURITY.md` found 6 P0 vulnerabilities. Fix all of them. Also fix the 2 test build failures in `x/ibcratelimit/keeper/keeper_test.go`.

## Scope
Files to modify:
- `x/ibcratelimit/keeper/keeper.go`
- `x/ibcratelimit/keeper/keeper_test.go` 
- `x/ibcratelimit/middleware.go`
- `x/ibcratelimit/state.go`
- `x/ibcratelimit/types/` (as needed)
- `x/icaauth/keeper/msg_server.go`
- `x/icaauth/types/genesis.go`
- `app/app.go` (ICA wrapper wiring only — the rate limit wrapper for ICA path)

## P0 Fixes Required

### P0-1: ICA Transfers Bypass Rate Limits (app.go:1042-1045)
The ICA controller's ICS4Wrapper points to `app.IBCKeeper.ChannelKeeper` directly, bypassing the rate limit middleware. Fix: wrap the ICA controller's ICS4Wrapper through the rate limit middleware, similar to how the transfer module is wrapped. Be careful not to break the ICA channel handshake flow.

### P0-2: Single-Denom Rate Limiting (middleware.go:130, 269)
Only one denom is rate-limited; all other denominations pass through unchecked. Fix: either rate-limit all denoms by default with per-denom overrides, OR add a `RateLimitedDenoms` list in params that specifies which denoms to limit (with a total aggregate limit as backstop).

### P0-3: Epoch Boundary Race — Instant Reset (state.go:117-122)
At epoch boundaries, flow counters reset to zero instantly, effectively doubling capacity at the boundary (full flow just before reset + full flow just after). Fix: implement a sliding window or gradual decay instead of an instant reset. One approach: use a rolling window where flow = current_epoch_flow + (remaining_fraction * previous_epoch_flow).

### P0-4: AllowedMsgTypes Per-Account Not Enforced (msg_server.go:89-116)
`SendICATransaction` does not check the per-account `AllowedMsgTypes` allowlist. Fix: before constructing the ICA packet, validate that each message type in the request is in the account's allowed list.

### P0-5: SendICATransaction Doesn't Actually Send (msg_server.go:107-115)
The handler validates but never calls `SendTx` on the ICA controller keeper. Fix: actually send the transaction by calling the controller keeper's `SendTx` method with the properly constructed packet data.

### P0-6: AllowedHostMsgTypes Includes MsgTransfer (genesis.go:12)
The default genesis includes `MsgTransfer` in `AllowedHostMsgTypes`, creating an IBC transfer backdoor that bypasses rate limiting. Fix: remove `MsgTransfer` from the default allowed host message types. Only allow explicitly opted-in message types.

### Test Build Fix
Fix `ChannelID` → `ChannelId` in `x/ibcratelimit/keeper/keeper_test.go` (3 occurrences at lines 470, 486, 499).

## Testing
- Fix existing test build failures first
- Add tests for rate limit bypass scenarios
- All tests must pass: `go test ./x/ibcratelimit/... ./x/icaauth/... -count=1`
- Run `go build ./...` to verify no compilation errors

## Report
Write a detailed report to `reports/SESSION-3-IBC-SECURITY-P0.md` covering:
- Each fix with before/after code
- New tests added  
- Test results
