# Session 7: Staking & Knowledge Module Security Hardening

**Date**: 2026-02-18
**Branch**: `feat-cicd-opensource-flow`
**Source Audits**: `reports/STAKING-POT.md`, `reports/KNOWLEDGE-KEEPER.md`

---

## Summary

Fixed 5 security findings (3 P0, 1 P1, 1 P2) from the staking/PoT audit, plus 1 overflow vulnerability in the knowledge module's confidence calculation. Added 9 new tests covering all fixes. All 54 staking tests pass; all modified modules build cleanly.

---

## Findings Addressed

### P0-1: Reputation uint64 Underflow in RecordVerification

**File**: `x/staking/keeper/keeper.go:549-555`
**Severity**: P0 (data corruption / potential panic on subsequent math)

The `RecordVerification` function decremented reputation by 200 on incorrect verdicts, but the guard only checked `> 100`, leaving values 101-199 vulnerable to uint64 underflow (wrapping to ~2^64).

**Before**:
```go
if validator.Reputation > 100 {
    validator.Reputation -= 200
}
```

**After**:
```go
if validator.Reputation >= 200 {
    validator.Reputation -= 200
} else {
    validator.Reputation = 0
}
```

**Tests added**: `TestRecordVerification_ReputationUnderflowEdge` — covers values 200 (exact boundary), 199, 1, and 0.

---

### P0-2: ContestedVerificationsCorrect Never Incremented

**Files**: 6 files modified across `x/knowledge` and `x/staking`
**Severity**: P0 (Guardian tier unreachable — requires MinContestedVerifications=33)

The `VerificationHooks.AfterVerificationCompleted` interface had no `contested` parameter, so `RecordVerification` could never know whether a verification was contested. Guardian tier requires 33 contested correct verifications, making it permanently unreachable.

**Changes**:
1. `x/knowledge/types/hooks.go` — Added `contested bool` to `VerificationHooks` interface and `MultiVerificationHooks`
2. `x/staking/keeper/hooks.go` — `StakingVerificationHooks.AfterVerificationCompleted` passes contested to `RecordVerification`
3. `x/staking/keeper/keeper.go` — `RecordVerification(ctx, addr, correct, contested bool)` increments `ContestedVerificationsCorrect` when `correct && contested`
4. `x/knowledge/keeper/confidence.go` — `ApplyRewardsAndSlashes` accepts `contested bool`, passes to hooks
5. `x/knowledge/keeper/phases.go` — `performAggregation` derives contested from round type (`Challenge` or `Contradiction`)
6. `x/qualification/keeper/hooks.go` — Updated signature (accepts but ignores contested param)

**Tests added**: `TestRecordVerification_ContestedTracking` — verifies correct+contested increments counter, correct+non-contested does not, incorrect+contested does not.

---

### P0-3: Hardcoded Epoch Length in ComputeValidatorTier

**File**: `x/staking/keeper/tiers.go:28-35`
**Severity**: P0 (Guardian slash window 31x shorter than intended)

`ComputeValidatorTier` used a hardcoded `epochLength = 1111` (BLOCKS_PER_EPOCH) for the Guardian slash window, but the actual `SlashDecayPeriodBlocks` param is 34272. This made the Guardian 10-epoch window ~31x shorter (9.3 hours instead of 10 days), allowing slashed validators to regain Guardian status far too quickly.

**Before**:
```go
epochLength := uint64(1111) // BLOCKS_PER_EPOCH
```

**After**:
```go
params := k.GetParams(ctx)
epochLength := params.GetSlashDecayPeriodBlocks()
if epochLength == 0 {
    epochLength = 34272 // fallback default
}
```

**Tests added**: `TestComputeValidatorTier_GuardianSlashWindow` — verifies recent slash (within window) disqualifies Guardian, old slash (outside window) allows qualification.

---

### P1-1: DelegationsByValidatorPrefix Wrong Key Prefix

**Files**: `x/staking/types/keys.go`, `x/staking/keeper/keeper.go`
**Severity**: P1 (incorrect query results)

`GetDelegationsForValidator` iterated over `DelegationKeyPrefix` (0x02), which is keyed by delegator address, not validator address. This could return delegations for the wrong validator when addresses share a byte prefix.

**Fix**: Added a reverse index (`ValidatorDelegationIndexPrefix = 0x09`) maintained by `SetDelegation` and cleaned up by `DeleteDelegation`. `GetDelegationsForValidator` now iterates the reverse index to find delegator addresses, then looks up the actual delegation records.

**New store keys**:
- `ValidatorDelegationIndexPrefix` (0x09): validator → delegator reverse index
- `ValidatorDelegationIndexKey(validatorAddr, delegatorAddr)`: full index key
- `DelegationsByValidatorPrefix(validatorAddr)`: prefix for iteration

**Tests added**:
- `TestGetDelegationsForValidator_ReverseIndex` — 2 validators, 3 delegators, verifies correct delegation assignment including self-delegations
- `TestGetDelegationsForValidator_UndelegateCleanup` — verifies reverse index entry removed on full undelegation

---

### Knowledge Module: uint64 Overflow in Confidence Calculation

**File**: `x/knowledge/keeper/confidence.go`
**Severity**: P1 (incorrect slash amounts for large stakes)

The slash calculation `stake * wrongSlashBps / 1_000_000` could overflow uint64 when stake exceeds ~10^18 (1 billion LGM). Replaced with existing `safeMulDiv` which uses `math/big.Int` internally.

**Before**:
```go
slashAmount := stake * wrongSlashBps / 1_000_000
```

**After**:
```go
slashAmount := safeMulDiv(stake, wrongSlashBps, 1_000_000)
```

Applied at both slash sites (wrong verdict and missed round).

---

### P2-2: promoteToGuardian Test Helper Incorrect

**File**: `x/staking/keeper/keeper_test.go:152`
**Severity**: P2 (test correctness)

The `promoteToGuardian` helper set `ContestedVerificationsCorrect = 10`, but Guardian tier requires `MinContestedVerifications = 33`. This meant the helper's output didn't actually compute as Guardian tier, masking potential tier calculation bugs.

**Fix**: Changed to `ContestedVerificationsCorrect = 33`.

**Tests added**: `TestPromoteToGuardian_MeetsAllRequirements` — verifies the helper produces a validator that `ComputeValidatorTier` evaluates as Guardian.

---

## Findings NOT Fixed (with rationale)

### P1-2: ValidateTransition Dead Code
The `ValidateTransition` function in `tiers.go` is not called from any message handler. This is intentional dead code — it's a utility function available for future use when tier transition messages are added. No fix needed; leaving as-is for forward compatibility.

### P2-1: CalculateTierSlash Unused
Similar to P1-2, `CalculateTierSlash` is a utility available but not yet wired into `SlashValidator`. The current slash logic uses progressive escalation (R3) which is separate from tier-based multipliers. Wiring this requires a design decision on whether tier multipliers stack with escalation. Deferred to a future session.

---

## Additional Tests Added

| Test Name | Finding | Validates |
|-----------|---------|-----------|
| `TestRecordVerification_ReputationUnderflowEdge` | P0-1 | Reputation clamps to 0 at values 200, 199, 1, 0 |
| `TestRecordVerification_ContestedTracking` | P0-2 | Contested flag correctly increments counter |
| `TestComputeValidatorTier_GuardianSlashWindow` | P0-3 | Slash window uses params, not hardcoded value |
| `TestGetDelegationsForValidator_ReverseIndex` | P1-1 | Reverse index returns correct delegations per validator |
| `TestGetDelegationsForValidator_UndelegateCleanup` | P1-1 | Reverse index cleaned on full undelegation |
| `TestComputeValidatorTier_GuardianContestedRequired` | P0-2/P2-2 | Guardian requires exactly 33 contested verifications |
| `TestComputeValidatorTier_ContestedMultiplier` | P0-2 | 3x multiplier for contested verifications in Guardian tier |
| `TestPromoteToGuardian_MeetsAllRequirements` | P2-2 | Test helper produces a valid Guardian validator |

---

## Test Results

```
go test ./x/staking/... -count=1    → 54 PASS (was 33 + 12 prior + 9 new)
go build ./x/staking/...            → OK
go build ./x/knowledge/...          → OK
go build ./x/qualification/...      → OK
```

**Pre-existing knowledge test failures** (not from this session): 3 tests in `x/knowledge/keeper` fail due to uncommitted changes in `rounds.go` that added `VerifyCommitmentHash` without updating test fixtures. Verified by stashing session changes — failures persist on base.

---

## VRF Implementation Review

Reviewed `x/knowledge/crypto/vrf.go`:
- ECVRF with Ed25519, domain-separated seeds via `GenerateVRFSeed`
- `IsValidatorSelected` uses `math/big.Int` for overflow-safe comparison
- Structured binary payload with length prefixes prevents hash collision attacks
- **No issues found** — implementation is sound

## Phase Transition Safety Review

Reviewed `x/knowledge/keeper/rounds.go`:
- `StoreCommitmentInRound` rejects if `round.Phase != "commit"`
- `StoreRevealInRound` rejects if `round.Phase != "reveal"` and verifies commitment exists
- Equivocation detection for both commitments and reveals
- `CompleteRound` has idempotency guard
- **No issues found** — phase transitions are properly guarded

---

## Files Modified

| File | Changes |
|------|---------|
| `x/staking/keeper/keeper.go` | P0-1 underflow fix, P0-2 contested param, P1-1 reverse index |
| `x/staking/keeper/tiers.go` | P0-3 epoch length from params |
| `x/staking/keeper/hooks.go` | P0-2 contested param passthrough |
| `x/staking/types/keys.go` | P1-1 reverse index prefix + key functions |
| `x/staking/keeper/keeper_test.go` | P2-2 helper fix + 9 new tests |
| `x/knowledge/types/hooks.go` | P0-2 contested param in interface |
| `x/knowledge/keeper/confidence.go` | P0-2 contested param + overflow fix |
| `x/knowledge/keeper/phases.go` | P0-2 contested derivation from round type |
| `x/qualification/keeper/hooks.go` | P0-2 interface compatibility |
