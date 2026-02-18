# R4 Session 3: Tokens Delegation + Wrap Hardening

## Context
Session R3-6 implemented DelegatePower, WrapToken, UnwrapToken. The implementation is solid but needs hardening. Key insights from reviewing the diff:

### Insight 1: Delegation locking affects existing operations
Burns, transfers, and TransferFrom now check `GetUndelegatedBalance` instead of raw balance. This is correct, but the **existing tests don't cover the interaction**. A user who delegates 80% of their tokens should fail to transfer 50% — is this tested?

### Insight 2: Wrap/Unwrap uses bank module Mint/Burn
The wrap path mints `lgm20/{tokenId}` coins via bank keeper. This means:
- The tokens module account needs `Burner` + `Minter` permissions in app.go
- Verify `maccPerms` in app.go includes these for the tokens module

### Insight 3: No delegation iteration cap
`IterateDelegationsByToken` iterates all delegations. In a pathological case with thousands of delegations per token, genesis export could be expensive. Consider: is there a practical bound?

### Insight 4: No wrap feature flag enforcement on unwrap
The code comment says "Unwrap is always allowed regardless of the Wrappable feature flag." This is intentional (you can always get your tokens back), but should have a test proving this.

## Tasks

### 1. Delegation-Aware Transfer/Burn Tests
Add tests in `x/tokens/keeper/` proving:
- Delegate 80%, try transfer 50% → fails with ErrInsufficientUndelegatedBalance
- Delegate 80%, try burn 50% → fails
- Delegate 80%, try transfer 20% → succeeds
- Delegate then un-delegate (amount=0), transfer full balance → succeeds
- TransferFrom with delegated tokens → correctly limited

### 2. Verify Module Account Permissions
Check `app/app.go` maccPerms for the tokens module. It should have `{Burner, Minter}`. If not, add it. Without Minter, WrapToken will panic at runtime.

### 3. Wrap/Unwrap Round-Trip Test
- Create token with Wrappable=true
- Mint 1000 to user
- Wrap 600 → verify LGM-20 balance is 400, sdk.Coin balance is 600
- Unwrap 300 → verify LGM-20 balance is 700, sdk.Coin balance is 300
- Unwrap remaining 300 → verify LGM-20 balance is 1000, sdk.Coin balance is 0

### 4. Unwrap Without Wrappable Flag Test
- Wrap tokens while Wrappable=true
- If governance disables Wrappable → unwrap should still work

### 5. Edge Cases
- Delegate to multiple delegates, verify total tracking is correct
- Delegate, then receive more tokens → undelegated balance increases
- Wrap with delegated tokens → should only wrap undelegated portion
- Zero-amount delegation (revoke) cleans up store keys

### 6. Gas Entries
Verify `app/gas.go` has entries for:
- `delegate_power`: 30,000 (moderate write, 2 KV ops)
- `wrap_token`: 60,000 (cross-module: LGM-20 debit + bank Mint + Send)
- `unwrap_token`: 60,000 (cross-module: bank Send + Burn + LGM-20 credit)

## Verification
```bash
go build ./...
go test ./x/tokens/...
go test ./app/...
```

## Report
Write to `/reports/batch-4/SESSION-R4-3-TOKENS-HARDENING.md`
