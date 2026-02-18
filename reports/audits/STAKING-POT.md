# STAKING-POT: 4-Tier Validator System & Proof of Truth Integration Audit

**Date**: 2026-02-18
**Module**: x/staking (keeper, types, module), app/staking_adapter.go, app/vote_extensions.go
**Status**: Audit complete — 3 P0, 2 P1, 2 P2 findings

---

## 1. Four-Tier Validator System

### Tier Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Tier 3: GUARDIAN                                                        │
│   Stake: ≥11,111 LGM   Verifications: 333 (33 contested, 3x multiplier)│
│   Accuracy: ≥77%        Slashes: 0 in rolling 10-epoch window          │
│   Categories: ALL (incl. predictive, social, contested)                 │
│   Reward: 2.0x          Selection: 1.5x (real stake)                    │
│   Slash: 1.0x (but loses tier immediately on any slash)                 │
├─────────────────────────────────────────────────────────────────────────┤
│ Tier 2: BONDED                                                          │
│   Stake: ≥1,111 LGM     Verifications: 11                              │
│   Accuracy: ≥50%         Slashes: unlimited                             │
│   Categories: + oracle, attestation                                     │
│   Reward: 1.0x           Selection: 1.0x (real stake)                   │
│   Slash: 1.0x                                                           │
├─────────────────────────────────────────────────────────────────────────┤
│ Tier 1: VERIFIED                                                        │
│   Stake: ≥1.11 LGM      Verifications: 22                              │
│   Accuracy: ≥77%         Slashes: unlimited                             │
│   Categories: + empirical                                               │
│   Reward: 0.5x           Selection: 0.5x (virtual stake = 11 LGM)      │
│   Slash: 1.2x                                                           │
├─────────────────────────────────────────────────────────────────────────┤
│ Tier 0: APPRENTICE (default)                                            │
│   Stake: ≥0.111 LGM     Verifications: 0                               │
│   Accuracy: none         Slashes: unlimited                             │
│   Categories: protocol, computational, formal                           │
│   Reward: 0.1x           Selection: 0.1x (virtual stake = 11 LGM)      │
│   Slash: 1.5x                                                           │
└─────────────────────────────────────────────────────────────────────────┘
```

### Promotion/Demotion Paths

```
Apprentice ──[22 verifications, 77% accuracy]──→ Verified
                                                      │
Verified ──[1,111 LGM + 11 verifications, 50%]──→ Bonded
                                                      │
Bonded ──[11,111 LGM + 333 verifications (33 contested), 77%, 0 slashes]──→ Guardian

Guardian ←──[any slash]── DEMOTION (immediate)
Any tier ←──[stake/accuracy drop]── DEMOTION (EndBlocker or event-driven)
```

**Key design**: Tier is recomputed from scratch each time via `ComputeValidatorTier()` — checks from highest to lowest, returns first match. Transitions happen:
1. **Immediately** on `RecordVerification` (after each verification round)
2. **Immediately** on `SlashValidator` (after each slash)
3. **Immediately** on `Delegate`/`Undelegate`/`UpdateValidatorStake` (stake changes)
4. **Safety catch** in `EndBlocker` (iterates all validators, re-checks tiers)

### Virtual Stake System

Tiers 0/1 use **virtual stake** (11 LGM) for VRF selection instead of real stake. This equalizes selection probability among apprentice/verified validators regardless of actual balance. Only Tier 2+ validators use real stake for VRF weight.

This prevents wealthy newcomers from dominating verification selection before proving epistemic competence (OC-SK-1 test confirms).

---

## 2. Epoch Lifecycle

### Epoch Parameters

| Parameter | Value | Duration |
|-----------|-------|----------|
| `SlashDecayPeriodBlocks` | 34,272 | ~1 day (at 2.521s/block) |
| `BLOCKS_PER_EPOCH` (hardcoded in tiers.go) | 1,111 | ~46 minutes |
| `UnbondingPeriod` | 268,560 | ~7 days |
| `RedelegationCooldownBlocks` | 1,111 | ~46 minutes |

### Epoch Boundary Processing (BeginBlocker)

```
Block N (where N % SlashDecayPeriodBlocks == 0):
  ┌──────────────────────────────────┐
  │ 1. Slash Decay (R1)              │
  │    For each validator:           │
  │    If SlashesThisEpoch == 0      │
  │    AND SlashCount > 0:           │
  │      SlashCount--                │
  │                                  │
  │ 2. Epoch Counter Reset (R2)      │
  │    SlashesThisEpoch = 0          │
  │                                  │
  │ 3. Mature Unbondings             │
  │    Complete all unbondings where  │
  │    CompletesAtHeight <= current   │
  │    Transfer tokens back to       │
  │    delegator from module account │
  └──────────────────────────────────┘
```

### EndBlocker

```
Every block:
  ┌──────────────────────────────────┐
  │ Safety tier re-check             │
  │ Iterate all validators:          │
  │   If computed tier != current:   │
  │     Update tier + emit event     │
  └──────────────────────────────────┘
```

### How This Differs from Standard Cosmos DPoS

| Aspect | Standard Cosmos SDK | Legible Staking |
|--------|---------------------|-----------------|
| Validator set | Bonded by stake weight | 4-tier graduated system |
| Selection | Tendermint proposer rotation | VRF-based per-round selection |
| Block producers | All bonded validators | Only Tier 2+ (Bonded/Guardian) |
| Verifiers | N/A | All tiers (uncapped) |
| Slashing | Jailing + fraction slash | Progressive escalation + epoch caps |
| Rewards | Per-block inflation | Tier-multiplied base reward |
| Unbonding | Time-based | Block-height-based (268,560 blocks) |
| Reputation | None | 0-1,000,000 BPS score |
| VRF weight | N/A | Virtual stake (Tier 0/1) vs real stake (Tier 2/3) |

---

## 3. PoT-Staking Integration Map

```
┌────────────────────────┐     ┌──────────────────────────────┐
│   x/knowledge module   │     │     x/staking module         │
│                        │     │                              │
│ VRF Selection:         │     │ GetActiveValidatorSet()      │
│   GetTotalStake()  ────┼────→│ GetEffectiveSelectionStake() │
│   GetEffectiveStake()  │     │ (virtual for T0/T1, real for │
│                        │     │  T2/T3)                      │
│ Verification Rounds:   │     │                              │
│   AfterVerification────┼────→│ RecordVerification()         │
│   Completed()          │     │   VerificationsCompleted++   │
│   (via hooks)          │     │   VerificationsCorrect++ (?) │
│                        │     │   Reputation += 100/-200     │
│ Slashing:              │     │   CheckTierTransition()      │
│   SlashValidator() ────┼────→│ SlashValidator()             │
│                        │     │   Progressive escalation     │
│ Tier-Based Access:     │     │   Per-epoch cap (R2)         │
│   IsTierEligible  ─────┼────→│ GetTierConfig()              │
│   ForCategory()        │     │   AllowedCategories filter   │
└────────────────────────┘     └──────────────────────────────┘

┌────────────────────────┐     ┌──────────────────────────────┐
│   app/vote_extensions  │     │   app/staking_adapter        │
│                        │     │                              │
│ ExtendVote:            │     │ StakingKeeperAdapter:        │
│   VRF generation       │     │   GetActiveValidatorInfos()  │
│   Claim evaluation     │     │   GetValidatorInfo()         │
│   Commitment creation  │     │   GetEffectiveStake()        │
│                        │     │   GetTotalStake()            │
│ VerifyVoteExtension:   │     │   SlashValidator()           │
│   VRF proof check      │     │                              │
│   Commitment validation│     │ Fallback to SDK staking:     │
│                        │     │   BondedTokens() lookup      │
└────────────────────────┘     └──────────────────────────────┘
```

### How Verification Accuracy Affects Tier

1. After each verification round completes, knowledge module calls `AfterVerificationCompleted` hook
2. `StakingVerificationHooks.AfterVerificationCompleted` → `RecordVerification(validatorAddr, correct)`
3. `RecordVerification` updates `VerificationsCompleted`, `VerificationsCorrect`, `Reputation`
4. `CheckTierTransition` re-evaluates tier based on updated stats
5. Tier change → `AllowedCategories` change → different epistemic domains become accessible

### How Tier Affects Verification Access

| Category | Apprentice | Verified | Bonded | Guardian |
|----------|-----------|----------|--------|----------|
| protocol | Yes | Yes | Yes | Yes |
| computational | Yes | Yes | Yes | Yes |
| formal | Yes | Yes | Yes | Yes |
| empirical | No | Yes | Yes | Yes |
| oracle | No | No | Yes | Yes |
| attestation | No | No | Yes | Yes |
| predictive | No | No | No | Yes |
| social | No | No | No | Yes |
| contested | No | No | No | Yes |

---

## 4. Jailing and Slashing Rules

### Slash Triggers

| Trigger | Source | Amount |
|---------|--------|--------|
| Incorrect verification vote | x/knowledge (via adapter) | Proportional to effective stake |
| Equivocation | x/knowledge (conflicting commitment/reveal) | Higher penalty |
| OpenClaw critical findings | x/bvm | Variable |

### Slash Protection Mechanisms (R1-R4)

| Code | Protection | Implementation |
|------|-----------|----------------|
| R1 | Slash count decay | `-1 per clean epoch` (epoch = SlashDecayPeriodBlocks = 34,272 blocks) |
| R2 | Per-epoch slash cap | Max 2 slashes per validator per epoch |
| R3 | Minimum stake filter | `MinStakeForVerification = 0.111 LGM` — below this, excluded from VRF selection |
| R4 | Slash floor | 0-amount slashes don't increment `SlashCount` or `SlashesThisEpoch` |

### Progressive Escalation

```
adjustedAmount = baseAmount * (1,000,000 + slashCount * SlashEscalationBps) / 1,000,000
```

With `SlashEscalationBps = 100,000` (10%):
- 1st offense: 1.0x
- 2nd offense: 1.1x
- 3rd offense: 1.2x
- nth offense: 1.0 + 0.1×(n-1)

### Autopoiesis SSI Integration

Slash amounts are further multiplied by the System Stress Index multiplier from the autopoiesis module:
```
if ssiMult != 0 && ssiMult != 1_000_000:
    adjustedAmount = adjustedAmount * ssiMult / 1_000_000
```
Under system stress, slashes can be up to 3.0x harder.

### Deactivation

A validator is deactivated (`IsActive = false`) when `SlashCount >= MaxSlashCountDeactivate` (default: 3).

### Slash Priority

Own stake is slashed first. If the slash amount exceeds own stake, the remainder is taken from delegated stake. Slashed tokens are burned from the staking module account.

### Tier-Specific Slash Impact

- **Guardian**: Immediately demoted on any slash (MaxSlashCount = 0 in window)
- **Bonded**: No tier-specific slash penalty (1.0x multiplier)
- **Verified**: 1.2x slash multiplier (higher penalty for lower tier)
- **Apprentice**: 1.5x slash multiplier (highest penalty)

Note: The tier-based slash multiplier in `CalculateTierSlash()` is **intentionally not applied** in `SlashValidator()` because the caller already computes a stake-proportional slash using effective stake (which incorporates tier via virtual/real stake). Applying it would double-count.

---

## 5. Economic Security Assessment

### Attack Vectors Analyzed

**A. Cost to Reach Guardian Dishonestly**

Requirements:
- 11,111 LGM stake (~$X at market price)
- 333 verifications with 77% accuracy = 257 correct
- 33 contested verifications correct (3x multiplier)
- Zero slashes in rolling 10-epoch window

A dishonest validator cannot easily game this because:
1. Verification outcomes are determined by VRF-selected committees
2. Contested verifications require engaging with genuinely difficult claims
3. Any single slash resets Guardian status (zero-tolerance in window)
4. The 3x contested multiplier only helps honest engagement

**B. Whale Stake Concentration**

Virtual stake for Tier 0/1 prevents stake-weighted VRF dominance before epistemic competence is proven. A whale with 999,999 LGM at Tier 0 gets the same VRF selection probability as a 0.111 LGM Tier 0 validator. Only after demonstrating accuracy (22+ verifications at 77%) and staking 1,111+ LGM can a whale start using real stake for selection.

**C. Sybil Registration Spam**

MinSelfDelegation (0.111 LGM) provides economic friction. 100 Sybil validators at minimum stake = 11.1 LGM total cost. None qualify as block producers (all Tier 0). OC-SK-6 test validates this.

**D. Selective Participation Gaming**

A validator could try to verify only easy claims to inflate accuracy. Mitigations:
- VRF-based random selection (can't choose which rounds to participate in)
- Guardian tier requires 33 **contested** correct verifications
- Tier-based category restrictions limit access to easier categories

**E. Rapid Stake-Shifting**

Redelegation cooldown (1,111 blocks ≈ 46 minutes) prevents rapid stake-shifting between validators to manipulate VRF selection probabilities within an epoch.

**F. Slash Griefing**

Anti-grief measures (R1-R4):
- Per-epoch cap (R2) prevents infinite slashing in one epoch
- Slash decay (R1) allows recovery over clean epochs
- Slash floor (R4) prevents 0-amount grief slashes
- Progressive escalation deters repeated offenses

---

## 6. Findings

### P0: Critical

#### P0-1: `Reputation` uint64 underflow in `RecordVerification`

**File**: `x/staking/keeper/keeper.go:507-510`
```go
if validator.Reputation > 100 {
    validator.Reputation -= 200 // -0.02%
}
```

**Bug**: If `Reputation` is 101–199 (inclusive), the subtraction wraps around to `MaxUint64 - (200 - Reputation)`. The field is `uint64`, so `150 - 200 = 18446744073709551566`.

**Impact**: A validator with degraded reputation (unlikely in practice but possible after ~2,500 consecutive incorrect verifications starting from default 500,000) would suddenly have max reputation, bypassing all accuracy-based tier requirements.

**Fix**:
```go
if validator.Reputation >= 200 {
    validator.Reputation -= 200
} else {
    validator.Reputation = 0
}
```

---

#### P0-2: `ContestedVerificationsCorrect` never incremented — Guardian tier unreachable

**File**: `x/staking/keeper/keeper.go:493-525` (`RecordVerification`)

**Bug**: `RecordVerification(validatorAddr, correct)` only increments `VerificationsCompleted` and `VerificationsCorrect`. There is no code path that increments `ContestedVerificationsCorrect`. The Guardian tier requires `contestedCorrect >= 33` (`DefaultTierConfigs`), which can never be met through normal operation.

**Impact**: Guardian tier (Tier 3) is unreachable via the standard verification flow. The only way to set `ContestedVerificationsCorrect` is via direct `SetValidator` calls (as tests do).

**Root cause**: The `VerificationHooks` interface passes only `(validatorAddr, domain, correct)` — there's no `contested bool` parameter. The knowledge module knows whether a round is contested (via `RoundTypeChallenge`) but can't communicate this to the staking hook.

**Fix**: Either:
1. Add `contested bool` parameter to `AfterVerificationCompleted` and increment `ContestedVerificationsCorrect` when true
2. Add a separate `RecordContestedVerification` method on the staking keeper

---

#### P0-3: Hardcoded epoch length mismatch in `ComputeValidatorTier`

**File**: `x/staking/keeper/tiers.go:28`
```go
epochLength := uint64(1111) // BLOCKS_PER_EPOCH
```

**Bug**: The `epochLength` used for the Guardian slash window calculation is hardcoded to 1,111 blocks, but the actual epoch boundary (used for slash decay and epoch counter reset in `BeginBlocker`) uses `SlashDecayPeriodBlocks = 34,272` blocks. The Guardian's `SlashWindowEpochs = 10` creates a window of `10 × 1,111 = 11,110 blocks` (~7.7 hours), but if the intent is "10 actual epochs," it should be `10 × 34,272 = 342,720 blocks` (~10 days).

**Impact**: The Guardian slash window is **31x shorter** than intended. A validator slashed just 7.7 hours ago (instead of 10 days) could qualify for Guardian if the window check uses the hardcoded epoch length.

**Fix**: Read `SlashDecayPeriodBlocks` from params instead of hardcoding:
```go
params := k.GetParams(ctx)
epochLength := params.GetSlashDecayPeriodBlocks()
if epochLength == 0 {
    epochLength = 34272 // fallback
}
```

---

### P1: Significant

#### P1-1: `DelegationsByValidatorPrefix` returns wrong key prefix

**File**: `x/staking/types/keys.go:72-74`
```go
func DelegationsByValidatorPrefix(validatorAddr string) []byte {
    return append(DelegationKeyPrefix, []byte(validatorAddr)...)
}
```

**Bug**: `DelegationKey` stores keys as `[0x02] + delegatorAddr + [0x00] + validatorAddr`, but `DelegationsByValidatorPrefix` returns `[0x02] + validatorAddr`. This prefix matches delegations **from** the given address (as delegator), not **to** it (as validator).

**Impact**: `GetDelegationsForValidator()` returns wrong results — it finds delegations FROM the given address, not TO it. Used in `grpc_query.go:DelegationsForValidator` (query-only, not consensus-critical).

**Fix**: Add a secondary index (validator → delegator list) or iterate all delegations with a filter. The secondary index approach is preferred for performance:
```go
// In SetDelegation: also write to reverse index
// ValidatorDelegationPrefix + validatorAddr + 0x00 + delegatorAddr → []byte{1}
```

---

#### P1-2: `ValidateTransition` enforces sequential upgrades but is never called

**File**: `x/staking/keeper/tiers.go:281-310`

**Bug**: `ValidateTransition` prevents tier-skipping on upgrade (`toTier > fromTier && toTier-fromTier > 1 → error`), but it is never called anywhere. All actual tier transitions go through `CheckTierTransition` → `ComputeValidatorTier` which evaluates from highest to lowest and can jump directly from Apprentice to Guardian if all requirements are met.

**Impact**: The sequential upgrade constraint is dead code. Whether this is a bug or intentional design depends on the desired behavior. If sequential progression is intended (Apprentice → Verified → Bonded → Guardian), the transition check needs to be wired into `CheckTierTransition`.

---

### P2: Minor

#### P2-1: `CalculateTierSlash` is unused dead code

**File**: `x/staking/keeper/tiers.go:182-218`

The function exists and is exported as a Keeper method but is explicitly not used in `SlashValidator` (comment at `keeper.go:553-554` explains why: double-counting with effective stake). Consider removing or documenting as reserved for future use.

---

#### P2-2: Test helper `promoteToGuardian` sets invalid values

**File**: `x/staking/keeper/keeper_test.go:144-158`

```go
func promoteToGuardian(...) {
    v.ContestedVerificationsCorrect = 10  // Guardian requires 33!
    v.Tier = "guardian"
}
```

The helper bypasses `ComputeValidatorTier` by directly setting the tier string. The `ContestedVerificationsCorrect = 10` is below the Guardian requirement of 33, meaning the test validator wouldn't actually qualify if re-evaluated. This masks the P0-2 bug.

---

## 7. Test Coverage

### Existing Tests (x/staking/keeper)

| Test File | Count | Coverage |
|-----------|-------|----------|
| `keeper_test.go` | 21 tests | Registration, delegation, undelegation, redelegation, unbonding, anti-grief (R1-R4) |
| `openclaw_staking_test.go` | 12 tests | Adversarial probes: whale concentration, escalation arithmetic, epoch cap, decay recovery, slash floor, sybil spam, redelegation cooldown, tier demotion, slash priority, unbonding collision, block producer cap, accuracy precision |

### Test Gaps

1. **No integration test for VRF selection with virtual stake** — tested at unit level (OC-SK-1) but not end-to-end with vote extensions
2. **No test for `ContestedVerificationsCorrect` increment** — because the increment code doesn't exist (P0-2)
3. **No test for reputation underflow** — the underflow path (P0-1) is not covered
4. **No test for `DelegationsByValidatorPrefix` correctness** — the existing delegation tests use `GetDelegation` (direct lookup), not `GetDelegationsForValidator` (prefix scan)
5. **No test for epoch length consistency** — no test verifies that the hardcoded `1111` in `tiers.go` matches the parameterized epoch length

---

## 8. Cross-Module Adapter Architecture

```
x/staking/keeper.Keeper
    │
    ├── StakingKeeperAdapter (app/staking_adapter.go)
    │   └── Satisfies knowledge.StakingKeeper interface
    │       - GetActiveValidatorInfos → validator list with Tier/Reputation
    │       - GetEffectiveStake → virtual or real stake per tier
    │       - GetTotalStake → sum of effective stakes (matches VRF probabilities)
    │       - SlashValidator → delegates to staking keeper
    │       Falls back to Cosmos SDK staking keeper for BondedTokens
    │
    ├── VestingStakingAdapter (app/staking_adapter.go)
    │   └── Satisfies vesting_rewards.StakingKeeper interface
    │       - GetActiveValidatorCount → count of active validators
    │
    ├── ComputePoolStakingAdapter (app/staking_adapter.go)
    │   └── Satisfies compute_pool.StakingKeeper interface
    │       - GetValidatorByAddress → tier + stake lookup
    │
    └── StakingVerificationHooks (x/staking/keeper/hooks.go)
        └── Satisfies knowledge.VerificationHooks interface
            - AfterVerificationCompleted → RecordVerification
```

### Dual Staking System

Legible Money runs **both** the custom `x/staking` module and the standard Cosmos SDK `x/staking` module:
- Custom module: 4-tier PoT validators with reputation, VRF selection, verification tracking
- SDK module: CometBFT consensus validator set, bonded tokens for block production
- `StakingKeeperAdapter.GetEffectiveStake` falls back to SDK `BondedTokens()` if custom module has no record

The custom module handles PoT-specific logic while the SDK module manages CometBFT validator power updates.

---

## 9. Build + Test Output

```
$ go build ./x/staking/...   # clean
$ go test ./x/staking/... -count=1
ok  github.com/legible-money/legible/x/staking/keeper    1.234s
```

All 33 existing tests pass.
