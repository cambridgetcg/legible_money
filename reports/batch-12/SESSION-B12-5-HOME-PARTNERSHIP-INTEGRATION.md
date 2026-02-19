# B12-5 Session Report: Agent Home → Partnership Integration

## Summary

Linked `x/home` and `x/partnerships` modules so that the guardian relationship
(home) naturally connects to the economic relationship (partnership).

## Changes

### Proto Changes
- **`proto/legible/home/v1/state.proto`**: Added `partnership_id` (field 12) to
  `AgentHome` and `guardian_address` (field 6) to `HomeGuardian`
- **`proto/legible/home/v1/tx.proto`**: Added `guardian_address` (field 8) to
  `MsgConfigureGuardian`
- Regenerated `state.pb.go` and `tx.pb.go` via `buf generate`

### Cross-Module Keeper Interfaces
- **`x/home/types/expected_keepers.go`**: Added `PartnershipKeeper` interface
  with `GetPartnership()` and `CalculateAmountTier()`, plus `PartnershipInfo`
  mirror type (avoids circular imports)
- **`x/partnerships/types/expected_keepers.go`**: Added `HomeKeeper` interface
  with `GetHomesByOwner()`, `GetHomeGuardianAddress()`, `LinkHomeToPartnership()`,
  `SetHomeStatus()`, `GetHomeByPartnership()`
- **`x/home/keeper/keeper.go`**: Added `partnershipKeeper` field + `SetPartnershipKeeper()` setter
- **`x/partnerships/keeper/keeper.go`**: Added `homeKeeper` field + `SetHomeKeeper()` setter

### KV Storage
- **`x/home/types/keys.go`**: Added `HomeByPartnershipPrefix` (0x0B) for
  partnership→home reverse index
- **`x/home/keeper/state.go`**: Added 4 HomeKeeper interface methods:
  `GetHomeGuardianAddress()`, `LinkHomeToPartnership()`, `SetHomeStatus()`,
  `GetHomeByPartnership()`

### Integration Points (4)
1. **Auto-link on AcceptPartnership** (`x/partnerships/keeper/msg_server.go:148`):
   After activating partnership, checks if agent has a home where `guardian_address`
   matches the human partner. If so, auto-links the home to the partnership.
2. **Guardian-partner consistency** (`x/home/keeper/msg_server.go:398`):
   When setting `guardian_address` on a partnership-linked home, verifies the
   address matches the human partner.
3. **Freeze sync** (`x/partnerships/keeper/anti_coercion.go:62`):
   Partnership safety freeze sets linked home status to "guarded".
   Freeze lift restores home to "active" (line 180).
4. **Spending limit deliberation** (`x/home/keeper/guardian.go:109`):
   Large/major amounts (>22% of partnership pot) blocked — must go through
   partnership deliberation instead of direct spending.

### Adapter & Wiring
- **`app/partnerships_adapter.go`**: Added `HomePartnershipAdapter` bridging
  `partnershipskeeper.Keeper` → `hometypes.PartnershipKeeper`
- **`app/app.go`**: Wired cross-module setters after keeper initialization

## Tests (10 total, all passing)

### x/home/keeper (5 tests)
- `TestHomeGuardianPartnerConsistency` — guardian must match partner
- `TestHomeGuardianPartnerConsistency_NoPartnership` — no constraint without link
- `TestHomeSpendingLimitDeliberation` — large/major blocked, micro/small allowed
- `TestHomeSpendingLimitDeliberation_NoPartnership` — no deliberation without link
- `TestHomePartnershipLinkAndReverseLookup` — KV index and reverse lookup

### x/partnerships/keeper (5 tests)
- `TestHomePartnershipAutoLink` — auto-links on AcceptPartnership
- `TestHomePartnershipAutoLink_NoGuardianMatch` — no link when guardian differs
- `TestHomePartnershipAutoLink_NoHome` — no link when agent has no home
- `TestHomeFreezeSync` — freeze → home "guarded"
- `TestHomeFreezeSync_LiftRestoresActive` — lift → home "active"

## Verification
```
go build ./...          # clean
go vet ./...            # clean
go test ./x/home/...    # all pass
go test ./x/partnerships/... # all pass
go test ./app/...       # all pass
```

## Files Changed (14)
- `proto/legible/home/v1/state.proto` (2 new fields)
- `proto/legible/home/v1/tx.proto` (1 new field)
- `x/home/types/state.pb.go` (regenerated)
- `x/home/types/tx.pb.go` (regenerated)
- `x/home/types/expected_keepers.go` (PartnershipKeeper interface)
- `x/home/types/keys.go` (HomeByPartnershipPrefix)
- `x/home/keeper/keeper.go` (partnershipKeeper field + setter)
- `x/home/keeper/state.go` (4 HomeKeeper methods)
- `x/home/keeper/msg_server.go` (guardian consistency check)
- `x/home/keeper/guardian.go` (spending limit deliberation)
- `x/partnerships/types/expected_keepers.go` (HomeKeeper interface)
- `x/partnerships/keeper/keeper.go` (homeKeeper field + setter)
- `x/partnerships/keeper/msg_server.go` (auto-link on accept)
- `x/partnerships/keeper/anti_coercion.go` (freeze sync + lift restore)
- `app/partnerships_adapter.go` (HomePartnershipAdapter)
- `app/app.go` (cross-module wiring)
- `x/home/keeper/partnership_test.go` (NEW: 5 tests)
- `x/partnerships/keeper/home_link_test.go` (NEW: 5 tests)
