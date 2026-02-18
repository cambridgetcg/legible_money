# R4 Session 1: Gov Quorum Enforcement — Wire QuorumThresholdBps into Tally

## Context
`QuorumThresholdBps` exists in x/gov Params (default 1500 = 15%), has validation, has tests in `x/gov/quorum_test.go` — but is **never enforced** during actual proposal tallying. Session R3-7 was supposed to fix this but produced no changes.

`x/gov/quorum_test.go` already has tests for a `checkQuorum` function and quorum scenarios. The function and/or its call site in the tally path need to be verified or created.

## Tasks

### 1. Trace the Tally Path
- Read `x/gov/keeper/msg_server.go` — find where proposals get tallied (likely in a FinalizeProposal or TallyVotes method)
- Read `x/gov/keeper/keeper.go` — find vote counting logic
- Identify exactly where the quorum check should gate proposal passage

### 2. Expand StakingKeeper Interface
In `x/gov/types/expected_keepers.go`, add:
```go
GetTotalBondedStake(ctx sdk.Context) math.Int
```

### 3. Implement in x/staking
Add the method to x/staking's keeper. Sum the BondedTokens across all active validators (Bonded + Guardian tiers). This method may already exist under a different name — check first.

### 4. Wire Quorum Check into Tally
In the tally path:
- After counting total voted stake, compute: `quorumRequired = totalBondedStake * QuorumThresholdBps / 10000`
- If `totalVotedStake < quorumRequired` → proposal fails with a quorum-not-met status
- If `QuorumThresholdBps == 0` → quorum is disabled (always passes this check)

### 5. Update app.go Adapter
If the gov keeper receives staking through an adapter, update it to satisfy the expanded interface.

### 6. Tests
- Verify existing `x/gov/quorum_test.go` tests pass
- Add integration test: proposal with insufficient voter participation → fails
- Add test: QuorumThresholdBps=0 → quorum disabled
- Add test: exact threshold boundary

## Verification
```bash
go build ./...
go test ./x/gov/... ./x/staking/...
```

## Report
Write to `/reports/batch-4/SESSION-R4-1-GOV-QUORUM.md`
