# Session 7: Gov QuorumThresholdBps + Gas Coverage Gaps

## Objective
Two tasks: (1) Wire the unused `QuorumThresholdBps` in x/gov, and (2) fill gas coverage gaps across the 9 modules with only partial gas entries.

## Part A: Gov QuorumThresholdBps (DEFER-005)

### Context
- `x/gov` has `QuorumThresholdBps` in its Params but it's **never checked** during vote tallying
- The quorum check needs `StakingKeeper` to know total bonded stake
- This was deferred because it needs StakingKeeper interface expansion

### Tasks
1. **Expand StakingKeeper interface** in `x/gov/types/expected_keepers.go`:
   - Add `GetTotalBondedStake(ctx) math.Int` (or equivalent)
2. **Implement in x/staking keeper**:
   - Add the method to satisfy the interface
   - Sum bonded stake across all active validators
3. **Wire quorum check in x/gov**:
   - In the vote tallying logic (likely `keeper/tally.go` or similar)
   - After counting votes, check: `total_voted_stake >= total_bonded_stake * QuorumThresholdBps / 10000`
   - If quorum not met, proposal fails with `StatusQuorumNotMet` or equivalent
4. **Update app.go adapter** if needed:
   - Ensure the gov keeper receives a staking keeper that satisfies the expanded interface
5. **Tests**:
   - Test quorum met → proposal passes (if votes sufficient)
   - Test quorum not met → proposal fails
   - Test edge: exactly at quorum threshold
   - Test with QuorumThresholdBps = 0 (no quorum requirement)

## Part B: Gas Coverage Gaps

### Context
9 modules have only partial (hand-written) gas entries. They need complete coverage for all handler operations.

### Modules to Audit
Check each module's handlers against `app/gas.go` entries:

1. **x/autopoiesis** — 4 handlers, check gas
2. **x/billing** — 5 handlers
3. **x/capture_challenge** — 4 handlers
4. **x/capture_defense** — 3 handlers
5. **x/discovery** — 7 handlers
6. **x/ontology** — 6 handlers
7. **x/partnerships** — 11 handlers
8. **x/qualification** — 5 handlers
9. **x/schedule** — 6 handlers

### For Each Module
1. List all Msg handlers
2. Check if a gas entry exists in `app/gas.go`
3. For missing entries, add appropriate gas costs following the existing pattern:
   - Read-heavy operations: lower gas
   - Write operations: medium gas
   - Complex operations (multi-store writes, iterations): higher gas
4. Follow the naming convention: `Gas<Module><Operation>`

### Gas Cost Guidelines (from existing entries)
- Simple state write: 10,000-20,000
- Complex state write: 30,000-60,000
- Operations with iteration: 50,000-100,000
- Reference `app/gas.go` for the established scale

## Verification
```bash
go build ./...
go test ./x/gov/...
go test ./x/staking/...
go test ./...
go vet ./...
```

## Report
Write completion report to `/reports/SESSION-R3-7-GOV-QUORUM-GAS.md` with:
- Part A: QuorumThresholdBps implementation details, interface changes, test results
- Part B: Table of gas entries added per module (operation name, gas cost, rationale)
- Total new gas entries added
- Full test results
