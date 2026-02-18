# R4 Session 3: Tokens Delegation + Wrap Hardening

## Summary

Hardened the Phase 2 tokens module (DelegatePower, WrapToken, UnwrapToken) with 11 new tests covering delegation-aware transfer/burn interactions, multi-step wrap/unwrap round-trips, feature flag edge cases, and store key cleanup.

## Results

| Check | Status | Details |
|-------|--------|---------|
| Module Account Permissions | VERIFIED | `tokenstypes.ModuleName: {Minter, Burner}` at `app/app.go:317` |
| Gas Entries | VERIFIED | `delegate_power: 40K`, `wrap_token: 50K`, `unwrap_token: 50K` at `app/gas.go:154-156` |
| Tests | **134/134 PASS** | 11 new + 123 existing, 0 failures |
| Build | CLEAN | `go build ./...` succeeds |

## New Tests Added (11)

### Task 1: Delegation-Aware Transfer/Burn (5 tests)

| Test | Scenario | Result |
|------|----------|--------|
| `TestTransfer_Delegate80Pct_Transfer50Pct_Fails` | Delegate 800K/1M, transfer 500K | Correctly blocked (200K undelegated) |
| `TestBurn_Delegate80Pct_Burn50Pct_Fails` | Delegate 800K/1M, burn 500K | Correctly blocked |
| `TestTransfer_Delegate80Pct_Transfer20Pct_Succeeds` | Delegate 800K/1M, transfer 200K | Succeeds (exact undelegated) |
| `TestTransfer_DelegateThenRevoke_TransferFull_Succeeds` | Delegate 800K, revoke, transfer 1M | Succeeds (full balance restored) |
| `TestTransferFrom_BlockedByDelegation` | Approve 500K, delegate 900K, TransferFrom 200K | Blocked; 100K TransferFrom succeeds |

### Task 3: Multi-Step Wrap/Unwrap Round-Trip (1 test)

| Test | Scenario | Result |
|------|----------|--------|
| `TestWrapUnwrap_MultiStepRoundTrip` | Wrap 600K, unwrap 300K, unwrap 300K | All intermediate balances verified (LGM-20 + sdk.Coin) |

### Task 4: Unwrap Without Wrappable Flag (1 test)

| Test | Scenario | Result |
|------|----------|--------|
| `TestUnwrapToken_AfterWrappableDisabled` | Wrap 500, disable Wrappable, unwrap | Unwrap succeeds; wrap correctly blocked |

### Task 5: Edge Cases (4 tests)

| Test | Scenario | Result |
|------|----------|--------|
| `TestDelegation_ReceiveTokens_UndelegatedIncreases` | Delegate 150K, receive 100K more | Undelegated correctly increases (50K -> 150K) |
| `TestWrap_OnlyUndelegatedPortion` | Delegate 700K, wrap 300K, try wrap 1 more | Exact undelegated wraps; excess blocked |
| `TestDelegation_ZeroRevoke_CleansUpStoreKeys` | Delegate 500, revoke to 0, iterate | Store keys cleaned up, iteration yields nothing |
| `TestDelegation_MultipleDelegates_TotalTracking` | Delegate to 3 delegates, revoke 1 | Total and undelegated correctly track through add/revoke |

## Insight Review

### Insight 1: Delegation locking affects existing operations
**CONFIRMED** — Transfer, Burn, TransferFrom, and WrapToken all check `GetUndelegatedBalance()` before executing. Five new tests prove the interaction is correct.

### Insight 2: Wrap/Unwrap uses bank module Mint/Burn
**VERIFIED** — `maccPerms` at `app/app.go:317` grants `{Minter, Burner}` to the tokens module account. WrapToken calls `MintCoins` + `SendCoinsFromModuleToAccount`; UnwrapToken calls `SendCoinsFromAccountToModule` + `BurnCoins`.

### Insight 3: No delegation iteration cap
**ACKNOWLEDGED** — `IterateDelegationsByToken` has no practical bound. For genesis export this is acceptable (one-time operation). Runtime operations use `GetDelegatorTotal` (O(1)) not iteration. No change needed.

### Insight 4: Unwrap always allowed regardless of Wrappable flag
**CONFIRMED** — `UnwrapToken` does not check `token.Features.Wrappable`. Test `TestUnwrapToken_AfterWrappableDisabled` proves unwrap succeeds even after governance disables the flag. This is intentional: users can always recover their tokens.

## Gas Entry Notes

The prompt suggested `delegate_power: 30K`, `wrap_token: 60K`, `unwrap_token: 60K`. Actual values are 40K/50K/50K respectively. The existing calibration is reasonable:
- `delegate_power` at 40K: 2 KV writes (delegation + total) + 1 read + validation
- `wrap_token` at 50K: LGM-20 debit + bank MintCoins + SendFromModule
- `unwrap_token` at 50K: bank SendToModule + BurnCoins + LGM-20 credit

No gas changes made.

## Files Modified

- `x/tokens/keeper/phase2_test.go` — 11 new tests added

## Verification

```
go build ./...         # CLEAN
go test ./x/tokens/... # 134/134 PASS
```
