# R2 Session 6: 4-Tier Validator Lifecycle — Guardian Path Integration Test

**Date**: 2026-02-18
**Branch**: `feat-cicd-opensource-flow`
**Scope**: `x/staking/keeper/tiers.go`, `x/staking/keeper/keeper_test.go`

## Summary

Verified the complete Apprentice → Verified → Bonded → Guardian lifecycle with proper tier demotion on slash. Fixed a bug where the Verified tier did not enforce its MinStake requirement. Total: **1 bug fix** + **14 new tests**.

## Bug Fix: Verified Tier MinStake Not Enforced

**File**: `x/staking/keeper/tiers.go` (line 74)
**Root Cause**: `ComputeValidatorTier` checked `MinStake` for Guardian (Tier 3) and Bonded (Tier 2) but NOT for Verified (Tier 1). The `TierConfig` for Verified defines `MinStake: "1110000"` (1.11 LGM) but this was never evaluated. A validator with 0.111 LGM and 22 correct verifications at 77%+ accuracy would be promoted to Verified without meeting the stake requirement.

**Fix**: Added `MinStake` comparison for the Verified tier check, matching the pattern used by Bonded and Guardian:
```go
// Before:
if verifications >= verifiedConfig.MinVerifications &&
    accuracy >= verifiedConfig.MinAccuracy {
    return types.TierVerified
}

// After:
minStakeVerified := new(big.Int)
minStakeVerified.SetString(verifiedConfig.MinStake, 10)
if stakeInt.Cmp(minStakeVerified) >= 0 &&
    verifications >= verifiedConfig.MinVerifications &&
    accuracy >= verifiedConfig.MinAccuracy {
    return types.TierVerified
}
```

**Impact**: Verified tier now requires both 1.11 LGM stake AND 22 verifications at 77% accuracy. Without this fix, the Guardian path integration test would incorrectly promote validators with insufficient stake.

## New Tests (14 total)

### Task 1: Full Tier Progression (1 test)

| Test | Description |
|------|-------------|
| `TestTierProgression_ApprenticeToGuardian` | Full lifecycle: register at 0.111 LGM → 22 verifications (still Apprentice due to stake) → increase to 1.11 LGM (Verified) → increase to 1,111 LGM (Bonded) → increase to 11,111 LGM + 267 total verifications (212 normal + 33 contested + 22 prior) → Guardian |

### Task 2: Tier Demotion (4 tests)

| Test | Description |
|------|-------------|
| `TestTierDemotion_GuardianSlash_ImmediateDemotion` | Guardian slashed → immediate demotion to Bonded (MaxSlashCount=0 in window) |
| `TestTierDemotion_StakeDropsBelowThreshold` | Bonded validator reduces stake below 1,111 LGM → demoted to Apprentice (accuracy 75% < 77% for Verified) |
| `TestTierDemotion_AccuracyDropsBelowThreshold` | Verified validator records 8 incorrect verifications → accuracy drops to 73.3% < 77% → demoted to Apprentice |
| `TestTierDemotion_SlashDecayWindow` | Slash within 10-epoch window blocks Guardian; after window expires (342,721+ blocks), Guardian eligible again |

### Task 3: Contested Verification Flow (4 tests)

| Test | Description |
|------|-------------|
| `TestContestedVerification_ChallengeRound` | `RecordVerification(correct=true, contested=true)` increments `ContestedVerificationsCorrect` |
| `TestContestedVerification_ContradictionRound` | Same as challenge — `contested=true` increments counter |
| `TestContestedVerification_NormalRound` | `RecordVerification(correct=true, contested=false)` does NOT increment `ContestedVerificationsCorrect` |
| `TestContestedVerification_3xMultiplier` | 267 total + 33 contested × 3x = 333 effective → Guardian. 266 total + 33 contested = 332 effective → NOT Guardian. 32 contested (< 33 MinContestedVerifications) → NOT Guardian regardless of total. |

### Task 4: VRF Selection Weight by Tier (2 tests)

| Test | Description |
|------|-------------|
| `TestVRFSelection_TierWeighting` | Apprentice/Verified use virtual stake (11 LGM). Bonded/Guardian use real stake. Reward multipliers: 0.1x/0.5x/1.0x/2.0x. |
| `TestVRFSelection_OnlyEligibleCategories` | Apprentice: protocol/computational/formal only. Verified: adds empirical. Bonded: adds oracle/attestation. Guardian: all 9 categories including predictive/social/contested. |

### Task 5: Edge Cases (3 tests)

| Test | Description |
|------|-------------|
| `TestTier_ExactThresholdStake` | Exact minimum stake for each tier qualifies (Verified=1,110,000, Bonded=1,111,000,000, Guardian=11,111,000,000) |
| `TestTier_OneBelow` | One ulgm below threshold fails (1,109,999 → Apprentice, 1,110,999,999 → Verified not Bonded, 11,110,999,999 → Bonded not Guardian) |
| `TestTier_ZeroVerifications` | Zero verifications → Apprentice regardless of stake (even with 11,111 LGM) |

## Test Results

```
go build ./...                       ✅ Clean
go test ./x/staking/... -count=1     ✅ All PASS (0.38s)
go test ./x/knowledge/... -count=1   ✅ All PASS (0.38s)
go test ./x/... -count=1             ✅ All PASS except pre-existing x/bvm flaky test
```

Pre-existing failures (not related to this session):
- `x/bvm/keeper:TestGasBridge_NestedCalls` — BVM gas bridge test
- `app/:ante_integration_test.go` — proto API mismatch (build failure)

## Files Changed

| File | Lines Changed | Description |
|------|--------------|-------------|
| `x/staking/keeper/tiers.go` | +4 | Bug fix: Added MinStake check for Verified tier |
| `x/staking/keeper/keeper_test.go` | +350 | 14 new tests for tier lifecycle |

## Key Findings

1. **Verified tier MinStake was never enforced** — TierConfig defined 1.11 LGM but ComputeValidatorTier skipped the check. Guardian/Bonded both checked MinStake. This was the same class of bug as the Guardian unreachability (hooks not wiring through) — a requirement defined in config but not enforced in logic.

2. **MinContestedVerifications is raw, not multiplied** — The 3x `ContestedVerificationMultiplier` boosts the effective verification count for `MinVerifications` (333), but `MinContestedVerifications` (33) checks the raw `ContestedVerificationsCorrect` counter. This is correct behavior: you need 33 actual contested correct verifications, and their 3x multiplier reduces the total verification count needed from 333 to 267.

3. **Slash window math** — Guardian eligibility uses `SlashWindowEpochs * SlashDecayPeriodBlocks` (10 × 34,272 = 342,720 blocks ≈ 10 days). A slash at height H blocks Guardian until height H + 342,720. The P0-3 fix from R1 Session 7 correctly reads epoch length from params.

## Tier Requirements Summary (verified by tests)

| Tier | MinStake | MinVerifications | MinAccuracy | MaxSlashCount | Categories | Reward | VRF Stake |
|------|----------|-----------------|-------------|---------------|------------|--------|-----------|
| Apprentice | 0.111 LGM | 0 | 0% | - | 3 | 0.1x | Virtual (11 LGM) |
| Verified | 1.11 LGM | 22 | 77% | - | 4 | 0.5x | Virtual (11 LGM) |
| Bonded | 1,111 LGM | 11 | 50% | - | 6 | 1.0x | Real stake |
| Guardian | 11,111 LGM | 333* | 77% | 0 (10-epoch window) | 9 (all) | 2.0x | Real stake |

*333 effective = raw count + contested × (multiplier − 1). Needs 33 raw contested correct.
