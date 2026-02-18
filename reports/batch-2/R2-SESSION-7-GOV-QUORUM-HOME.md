# R2 Session 7: Governance Quorum Integration + Agent Home Lifecycle

**Date**: 2026-02-18
**Branch**: `feat-cicd-opensource-flow`

---

## Summary

Added 14 new tests across 2 modules: 8 governance quorum integration tests with a real mock staking keeper (previously all quorum tests used nil-safe degradation), and 6 agent home lifecycle tests covering guardian deadman actions, status transitions, and end-to-end integration. All 164 tests pass across both modules.

---

## Part A: Governance Quorum Integration Tests

**File**: `x/gov/quorum_test.go` (new, package `gov_test`)

### Background

Round 1 Session 5 added `QuorumThresholdBps` enforcement to governance tallying (`module.go:262-278`). The `checkQuorum` function compares total voting power against total bonded stake. However, all existing tests used `nil` for the staking keeper, which triggers nil-safe degradation (`return true` — quorum always met). No test actually exercised the quorum check with real stake values.

### Mock Staking Keeper

Created `mockStakingKeeper` implementing `types.StakingKeeper` with configurable `totalBondedStake`. Tests use `setupWithStaking(t, totalBonded)` which creates a keeper wired with the mock, builds an `AppModule`, and returns both.

Tests exercise quorum via `AppModule.BeginBlock()`, which triggers `tallyAndResolveLIP`/`tallyAndResolveDisbursement` for proposals past their voting deadline.

### Tests Added

| Test | What It Verifies |
|------|-----------------|
| `TestQuorum_Sufficient_LIPPasses` | 66.67% participation > 15% quorum → LIP accepted |
| `TestQuorum_Insufficient_LIPFails` | 10% participation < 15% quorum → LIP rejected (even with 100% support) |
| `TestQuorum_Disbursement_HigherThreshold` | Same 18% participation passes LIP (15% quorum) but fails disbursement (20% quorum) |
| `TestQuorum_ZeroBondedStake` | 0 bonded stake → quorum never met, proposal rejected |
| `TestQuorum_ZeroThreshold` | QuorumThresholdBps=0 → quorum disabled, minimal participation passes |
| `TestQuorum_ExactlyAtThreshold` | 15.00% exactly (1500 BPS) → passes (`>=` check) |
| `TestQuorum_JustBelowThreshold` | 14.96% (1496 BPS) → fails |
| `TestQuorum_LargeStake_BigIntSafe` | 10^18 stakes → no uint64 overflow in `bpsFraction` |

### Key Design Insight

The quorum formula is: `(totalVotingPower × 10000) / totalBondedStake >= quorumBps`

Where `totalVotingPower` is the sum of quadratic voting weights (sqrt of stake), not raw stake. This means the quorum metric compares *quadratic participation* against *linear bonded stake* — an intentional design that makes quorum harder to meet with whale-only participation.

### Quorum Logic Verification

- `checkQuorum` (`module.go:262-278`): Correct nil-safe degradation, zero-bonded guard, BPS comparison
- `bpsFraction` (`module.go:294-304`): Uses `math/big.Int` — overflow-safe
- `tallyAndResolveLIP` (`module.go:163-183`): Checks quorum before support threshold
- `tallyAndResolveDisbursement` (`module.go:226-257`): Uses separate `DisbursementQuorumBps` (2000 vs 1500)

### Minor Finding: No DisbursementQuorumBps Validation

`Params.Validate()` (`types/types.go:85-88`) validates `QuorumThresholdBps <= 10000` but does NOT validate `DisbursementQuorumBps`. This means `DisbursementQuorumBps > 10000` would silently make disbursements impossible to pass. **P2 severity — governance can still fix via `UpdateParams`**.

---

## Part B: Agent Home Lifecycle Tests

**File**: `x/home/keeper/lifecycle_test.go` (new, package `keeper_test`)

### Background

The home module already had 59 tests (41 in `keeper_test.go` + 12 adversarial in `openclaw_home_test.go` + 6 sub-tests). The existing tests thoroughly cover individual features. The new tests focus on:

1. Guardian deadman action specifics (alert type/priority verification)
2. Status transitions not previously tested (recovery, guarded→recovery)
3. End-to-end integration combining multiple features

### Tests Added

| Test | What It Verifies |
|------|-----------------|
| `TestGuardian_DeadmanAction_AlertGuardians` | Deadman creates `deadman_triggered` alert with `critical` priority, correct homeId and CreatedAt |
| `TestGuardian_DeadmanAction_TransferToBeneficiary` | `transfer_to_beneficiary` action (Phase 2 stub) still creates alert + sets guarded status |
| `TestHome_RecoveryStatus` | active→recovery→active lifecycle; recovery→dormant rejected |
| `TestHome_GuardedToRecoveryToActive` | guarded→recovery→active lifecycle; guarded→dormant rejected |
| `TestHome_ArchiveBlocksStatusChanges` | Archived home rejects transition to active/dormant/guarded/recovery |
| `TestHome_FullLifecycle_Integration` | 8-step end-to-end: create → guardian → key → spending limit → spend → deadman → recover → comfort |

### Status Transition Matrix Verified

```
From\To     active  dormant  guarded  recovery  archived
active        -      ✓        ✓        ✓         ✓
dormant      ✓       -        ✗        ✗         ✓
guarded      ✓       ✗        -        ✓         ✓
recovery     ✓       ✗        ✗        -         ✓
archived     ✗       ✗        ✗        ✗         -  (terminal)
```

### Integration Test Flow

`TestHome_FullLifecycle_Integration` exercises:
1. Home creation with default active status
2. Guardian configuration with aggressive defense + deadman switch (50,000 blocks)
3. Key registration with operator role + transfer/memory permissions
4. Spending limit (10 LGM/day) with debit verification
5. Epistemic activity recording (2 verifications + 1 claim)
6. Deadman trigger via time advancement (50,001 blocks of inactivity)
7. Recovery flow: guarded → recovery → active
8. Comfort dimension verification (sanctuary=100 from active+autoDefend)

---

## Test Results

```
go test ./x/gov/... -count=1     → 99 PASS (8 new quorum + 91 existing)
go test ./x/home/keeper -count=1 → 65 PASS (6 new lifecycle + 59 existing)
```

All existing tests continue to pass. No regressions.

---

## Files Created

| File | Package | Tests |
|------|---------|-------|
| `x/gov/quorum_test.go` | `gov_test` | 8 quorum integration tests |
| `x/home/keeper/lifecycle_test.go` | `keeper_test` | 6 lifecycle + integration tests |

---

## Files Read (not modified)

| File | Purpose |
|------|---------|
| `x/gov/module.go` | Quorum check logic, BeginBlock, tally functions |
| `x/gov/keeper/keeper.go` | Keeper constructor, GetStakingKeeper |
| `x/gov/types/expected_keepers.go` | StakingKeeper interface |
| `x/gov/keeper/keeper_test.go` | Existing test patterns and helpers |
| `x/home/keeper/guardian.go` | Deadman switch, spending limits, alerts |
| `x/home/keeper/keeper_test.go` | Existing test patterns and helpers |
| `x/home/types/types.go` | Status, action, and alert constants |
