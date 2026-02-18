# R2 Session 6: 4-Tier Validator Lifecycle — Guardian Path Integration Test

## Context
Round 1 Session 7 discovered that Guardian tier was unreachable because `ContestedVerificationsCorrect` was never incremented (hooks interface lacked the `contested` parameter). The fix added contested tracking through the full hooks chain. 

The aha moment: **the Guardian tier was designed but never achievable**. Now that the path is open, we need to verify the entire lifecycle: Apprentice → Verified → Bonded → Guardian, with proper tier demotion on slash.

## Scope
- `x/staking/keeper/keeper.go`
- `x/staking/keeper/tiers.go`
- `x/staking/keeper/keeper_test.go`
- `x/knowledge/keeper/phases.go` (read verification hooks)

## Tasks

### 1. Full Tier Progression Integration Test
Write `TestTierProgression_ApprenticeToGuardian` that simulates the complete validator lifecycle:
1. Register validator with 0.111 LGM → Apprentice (Tier 0)
2. Perform 22 correct verifications → check still Apprentice (need 1.11 LGM for Verified)
3. Increase stake to 1.11 LGM → Verified (Tier 1) 
4. Verify accuracy ≥ 77% → still Verified
5. Increase stake to 1,111 LGM → check if Bonded requirements met (Tier 2)
6. Simulate 11 verifications with ≥ 50% accuracy → Bonded
7. Increase stake to 11,111 LGM, simulate 333 total verifications (33 contested, 3x multiplier), 77% accuracy, 0 slashes → Guardian (Tier 3)

### 2. Tier Demotion Tests
- `TestTierDemotion_GuardianSlash_ImmediateDemotion` — Guardian gets slashed → immediately drops to Bonded (regardless of other metrics)
- `TestTierDemotion_StakeDropsBelowThreshold` — Validator undelegates below tier's minimum stake → demoted
- `TestTierDemotion_AccuracyDropsBelowThreshold` — Validator's accuracy drops below tier requirement → demoted
- `TestTierDemotion_SlashDecayWindow` — Slash happened 11 epochs ago (outside 10-epoch window) → Guardian eligible again

### 3. Contested Verification Flow
- `TestContestedVerification_ChallengeRound` — Verification in a Challenge round → contested=true passed to hooks → counter incremented
- `TestContestedVerification_ContradictionRound` — Verification in a Contradiction round → contested=true
- `TestContestedVerification_NormalRound` — Verification in a Normal round → contested=false → counter NOT incremented
- `TestContestedVerification_3xMultiplier` — 11 contested correct verifications count as 33 for Guardian requirement

### 4. VRF Selection Weight by Tier
- `TestVRFSelection_TierWeighting` — Run 1000 simulated selections and verify:
  - Apprentice selected ~10% of fair share (virtual stake = 11 LGM)
  - Verified selected ~50% of fair share (0.5x, virtual stake = 11 LGM)
  - Bonded selected at fair share (1.0x, real stake)
  - Guardian selected at 1.5x fair share (real stake × 1.5)
- `TestVRFSelection_OnlyEligibleCategories` — Apprentice can only verify protocol/computational/formal claims, not empirical

### 5. Edge Cases
- `TestTier_ExactThresholdStake` — Stake exactly at tier boundary (e.g., 1.11 LGM for Verified) → qualifies
- `TestTier_OneBelow` — Stake at 1.109999 LGM → doesn't qualify for Verified
- `TestTier_ZeroVerifications` — New validator with sufficient stake but 0 verifications → Apprentice (not Verified)

## Report
Write to `reports/R2-SESSION-6-TIER-LIFECYCLE.md`
