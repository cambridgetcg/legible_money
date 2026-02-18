# R2 Session 7: Governance Quorum Integration + Agent Home Lifecycle

## Context
Round 1 Session 5 implemented `QuorumThresholdBps` enforcement in gov, but the existing tests all use `nil` for the staking keeper (nil-safe degradation = quorum always met). We need tests with a REAL staking keeper to verify quorum is actually enforced.

Also, `x/home` is central to the agent economy vision. The home module has aging, epistemic stats, comfort dimensions, guardian system, deadman switches, and spending limits — but we need to verify these features work together.

## Scope
- `x/gov/keeper/` and `x/gov/module.go`
- `x/gov/keeper/keeper_test.go` or new test file
- `x/home/keeper/`
- `x/home/keeper/keeper_test.go` or new test file

## Tasks — Part A: Governance Quorum

### 1. Quorum Integration Tests with Real Staking Keeper
Create tests that wire gov keeper with a mock staking keeper that returns actual stake values:

- `TestQuorum_Sufficient_LIPPasses` — 3 validators (1000 LGM each, total bonded = 3000), 2 vote yes (2000/3000 = 66.7% > 15% quorum) → proposal passes
- `TestQuorum_Insufficient_LIPFails` — 10 validators (1000 LGM each), 1 votes yes (1000/10000 = 10% < 15% quorum) → proposal fails EVEN IF vote is 100% yes
- `TestQuorum_Disbursement_HigherThreshold` — Disbursement requires 20% quorum, LIP requires 15% → same vote passes LIP but fails disbursement
- `TestQuorum_ZeroBondedStake` — No validators bonded → quorum never met → proposal fails
- `TestQuorum_ZeroThreshold` — QuorumThresholdBps=0 → quorum always met

### 2. Edge Cases
- `TestQuorum_ExactlyAtThreshold` — Voting power exactly equals quorum threshold (e.g., 1500 BPS = 15.00%) → passes
- `TestQuorum_JustBelowThreshold` — 14.99% → fails
- `TestQuorum_LargeStake_BigIntSafe` — Validators with 10^18 stake → no overflow in quorum calculation

## Tasks — Part B: Agent Home Lifecycle

### 3. Home Creation + Management Tests
- `TestHome_CreateAndQuery` — Create a home, query it by DID, verify all fields
- `TestHome_MemoryCIDUpdate` — Update memory CID, verify new CID stored
- `TestHome_MultipleHomesPerAccount` — Create up to MaxHomesPerAccount, verify limit enforced

### 4. Guardian System Tests
- `TestGuardian_Configure` — Set up guardian for a home, verify stored
- `TestGuardian_DeadmanSwitch` — Home goes dormant (no activity for DeadmanTimeout blocks), verify deadman switch triggers alert
- `TestGuardian_DeadmanAction_AlertGuardians` — Deadman fires → alert created with priority critical
- `TestGuardian_DeadmanAction_TransferToBeneficiary` — Deadman fires with transfer action → treasury sent to beneficiary

### 5. Session Key + Spending Limit Integration
- `TestHome_SessionKey_SpendingLimit` — Register session key → set spending limit → spend within limit → success
- `TestHome_SessionKey_ExceedsSpendingLimit` — Spend exceeding limit → rejected
- `TestHome_SessionKey_PeriodReset` — Spending limit period expires → counter resets → can spend again

### 6. Aging + Epistemic Stats
- `TestHome_Aging_BeginBlock` — Run BeginBlock multiple times → home age increases
- `TestHome_EpistemicStats` — Record verification activity → epistemic stats updated
- `TestHome_ComfortDimensions` — Set and query comfort dimensions for a home

### 7. Home Status Lifecycle
- `TestHome_StatusTransitions` — active → dormant (no activity) → active (new activity)
- `TestHome_GuardedStatus` — Guardian activates protection → status changes to guarded
- `TestHome_RecoveryStatus` — Recovery initiated → status changes to recovery
- `TestHome_ArchiveStatus` — Home archived → cannot perform operations

## Report
Write to `reports/R2-SESSION-7-GOV-QUORUM-HOME.md`
