# Session 6: Implement x/tokens Stub Handlers

## Objective
x/tokens has 3 Phase 2 stub handlers that return `Unimplemented`: DelegatePower, WrapToken, UnwrapToken. Implement them fully.

## Reference
- `x/tokens/keeper/msg_server.go` — stub handlers
- `x/tokens/types/` — message types, expected keepers
- `proto/legible/tokens/v1/tx.proto` — proto definitions
- `docs/TRUSTPAPER.md` — token design philosophy

## Context
Legible Money has a dual-ledger system. These 3 operations are core token mechanics:
- **DelegatePower**: Delegate voting/staking power from token holdings
- **WrapToken**: Wrap external tokens into the LGM ledger
- **UnwrapToken**: Unwrap LGM-wrapped tokens back to their origin

## Tasks

### 1. Understand the Design
- Read `x/tokens/types/` thoroughly — understand the dual-ledger model
- Read any design docs referencing token wrapping/unwrapping
- Check `x/tokens/keeper/keeper.go` for existing helper methods
- Check `x/staking/types/expected_keepers.go` for how staking expects to interact with tokens

### 2. Implement DelegatePower
- Validate sender has sufficient undelegated balance
- Create/update delegation record in store
- Emit event with delegator, amount, target
- Update any aggregate power tracking
- Return delegation ID or confirmation

### 3. Implement WrapToken
- Validate the wrap request (source token, amount, recipient)
- Lock/escrow the source tokens (via bank keeper)
- Mint or credit wrapped token representation on the LGM ledger
- Create wrap record for audit trail
- Emit wrap event

### 4. Implement UnwrapToken
- Validate the unwrap request (wrapped token ID/denom, amount)
- Verify sender holds sufficient wrapped tokens
- Burn/debit the wrapped representation
- Release/unescrow the source tokens (via bank keeper)
- Remove or update wrap record
- Emit unwrap event

### 5. Add Tests
- Add comprehensive tests in `x/tokens/keeper/msg_server_test.go` or `keeper_test.go`
- Test: successful delegate/wrap/unwrap
- Test: insufficient balance errors
- Test: double-unwrap prevention
- Test: delegation power updates correctly
- Test: events emitted correctly

### 6. Update Gas Entries
- Check `app/gas.go` for existing tokens gas entries
- Add gas entries for the 3 new operations if missing

## Verification
```bash
go build ./...
go test ./x/tokens/...
go vet ./x/tokens/...
```

## Report
Write completion report to `/reports/SESSION-R3-6-TOKENS-IMPLEMENTATION.md` with:
- Implementation approach for each handler
- Store schema decisions (keys, prefixes)
- New test count and coverage
- Gas entries added
- Any design decisions or assumptions made
- Interaction with x/staking delegation system
