# B14-4 — Remove Dead Treasury Module Accounts

## Context
The 5-way research sub-split created 5 separate module accounts:
- `treasury_research`
- `treasury_foundation`
- `treasury_community`
- `treasury_developers`
- `treasury_reserve`

After B14-1 and B14-2 eliminate the sub-split, these accounts are dead code.
They're registered as module accounts in `app.go` and consume gas for creation
at genesis. Clean them up.

## Changes

### 1. app/app.go — Remove Module Account Registration

Find where `maccPerms` (module account permissions) are defined:
```bash
grep -n "treasury_research\|treasury_foundation\|treasury_community\|treasury_developers\|treasury_reserve" app/app.go
```

Remove all 5 entries. Keep `research_fund` and `treasury_protocol`.

### 2. x/billing/keeper/distribution.go — Remove Constants

Already done in B14-2, but verify:
```go
// These should be DELETED:
ResearchTreasury   = "treasury_research"     // DELETE
FoundationTreasury = "treasury_foundation"   // DELETE
CommunityTreasury  = "treasury_community"    // DELETE
DevelopersTreasury = "treasury_developers"   // DELETE
ReserveTreasury    = "treasury_reserve"       // DELETE
```

### 3. Check for Any Other References

```bash
grep -rn "treasury_research\|treasury_foundation\|treasury_community\|treasury_developers\|treasury_reserve" x/ app/ --include="*.go" | grep -v _test.go | grep -v .pb.go
```

Remove ALL references. If any test files reference them, update those too.

### 4. Billing Proto Types

If `ResearchFundSplit` is a proto message type, deprecate or remove:
```bash
grep -rn "ResearchFundSplit\|research_split" proto/ --include="*.proto"
```

### 5. Migration Note

For existing chains (not applicable to fresh testnet, but good practice):
Any balances in the dead module accounts should be swept to `research_fund`
before removing them. For fresh genesis, this is a no-op.

### 6. Verify Module Account List

After cleanup, the complete list of module accounts should be:

| Account | Purpose |
|---------|---------|
| `research_fund` | Research tax deposits (3.33% of all rewards + 13% of service revenue) |
| `treasury_protocol` | Protocol operations (22% of service revenue) |
| `knowledge` | Verification/citation/falsification reward pool |
| `compute_pool` | Compute infrastructure funding |
| `fee_collector` | Standard Cosmos SDK tx fee collection |
| `vesting_rewards` | Block reward minting |
| `bonded_tokens_pool` | Standard Cosmos SDK staking |
| `not_bonded_tokens_pool` | Standard Cosmos SDK staking |
| `distribution` | Standard Cosmos SDK distribution |
| `gov` | Standard Cosmos SDK governance |

No more: `treasury_research`, `treasury_foundation`, `treasury_community`,
`treasury_developers`, `treasury_reserve`.

### 7. Run

```bash
go build ./...
go test ./... -count=1 -timeout 300s 2>&1 | tail -20
```

## Commit
```
chore: remove 5 dead treasury module accounts

Removes treasury_research, treasury_foundation, treasury_community,
treasury_developers, treasury_reserve — all unused after the revenue
split simplification. Module account list cleaned up.
```
