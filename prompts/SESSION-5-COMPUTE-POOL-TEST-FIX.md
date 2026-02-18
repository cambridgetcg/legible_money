# Session 5: Test Fixes + compute_pool + Gov Quorum Enforcement

## Context
You are working on Legible Money (`/Users/yuai/Desktop/legible_money`), a Cosmos SDK blockchain. There are 2 test build failures and a governance quorum enforcement gap to fix.

## Scope
Files to modify:
- `x/compute_pool/keeper/openclaw_compute_pool_test.go`
- `x/gov/keeper/msg_server.go`
- `x/gov/types/types.go`
- `x/staking/types/expected_keepers.go` (or wherever StakingKeeper interface is defined for gov)
- Test files as needed

## Task 1: Fix compute_pool Test Build Failure
In `x/compute_pool/keeper/openclaw_compute_pool_test.go`, lines 44 and 56 use `PricePerCU` but the proto-generated field is `PricePerCu`. Fix these 2 occurrences.

Verify: `go test ./x/compute_pool/... -count=1`

## Task 2: Implement Gov QuorumThresholdBps Enforcement
The `QuorumThresholdBps` parameter exists in gov params (validated in types.go line 85-86 as <= 10000) but is **never actually checked** during vote tallying.

Fix:
1. Find or create the vote tally function in `x/gov/keeper/` (likely in msg_server.go near the proposal finalization logic)
2. During vote tally, calculate: `totalVotingPower = sum of all voters' bonded stake`
3. Get `totalBondedStake` from the staking module — this may require adding a method to the StakingKeeper interface used by gov (check `x/gov/types/expected_keepers.go`)
4. Check: `totalVotingPower * 10000 / totalBondedStake >= params.QuorumThresholdBps`
5. If quorum is not met, the proposal should fail regardless of vote outcome
6. Handle the edge case where `totalBondedStake == 0`

Look at how the existing `x/staking` keeper exposes total bonded stake. The custom staking module at `x/staking/keeper/` should already track this. If `GetTotalBondedStake()` doesn't exist, add it.

## Task 3: Full Test Verification
After both fixes, run:
```bash
go test ./x/compute_pool/... -count=1
go test ./x/gov/... -count=1  
go test ./x/staking/... -count=1
go build ./...
```

All must pass.

## Report
Write a detailed report to `reports/SESSION-5-TEST-GOV-QUORUM.md` covering:
- Test fix details
- Quorum enforcement implementation with before/after code
- Any interface changes made
- Full test results
