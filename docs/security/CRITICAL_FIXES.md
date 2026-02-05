# Critical Security Fixes

This document contains the specific code changes required to fix the CRITICAL vulnerabilities
identified in the PoT vulnerability analysis.

---

## Fix 1: Validator Selection Must Use VRF (Currently Broken)

### Problem

In `consensus/pot/validator_selection.ts:64-80`, validators are selected by **stake order only**:

```typescript
// CURRENT CODE - VULNERABLE
selections.sort((a, b) => {
  if (b.stake > a.stake) return 1;  // Sort by stake
  // ...
});

for (let i = 0; i < targetCount; i++) {
  selections[i].selected = true;  // Top N by stake always selected
}
```

This means the **same validators are always selected** for every verification round.
Top stakers can form a cartel and control all verifications.

### The Fix

Replace stake-ordering with VRF-based probabilistic selection:

```typescript
// FIXED CODE - Uses VRF properly
export function selectValidatorsForVerification(
  claimId: string,
  blockNumber: bigint,
  previousBlockHash: string,
  validatorSet: ValidatorSet,
  params: ConsensusParams
): ValidatorSelection[] {
  // Generate seed for this claim's verification
  const seed = generateVerificationSeed(claimId, blockNumber, previousBlockHash);

  const selections: ValidatorSelection[] = [];

  // Each validator generates VRF output to determine if selected
  for (const [pubKey, validator] of validatorSet.validators) {
    if (!validator.isActive) continue;
    if (validator.stake < params.minValidatorStake) continue;

    // CRITICAL: Each validator must generate their own VRF
    // In production, this happens on the validator's machine
    // Here we check if their submitted VRF qualifies them
    const effectiveStake = validator.stake + validator.delegatedStake;

    // VRF selection is stake-weighted but RANDOM
    // Higher stake = higher probability, but NOT guaranteed
    const selectionResult = isValidatorSelected(
      validator.vrfOutput, // Must be submitted with commitment
      effectiveStake,
      validatorSet.totalStake,
      params.maxValidators
    );

    if (selectionResult.selected) {
      selections.push({
        validator: pubKey as PublicKey,
        vrfOutput: validator.vrfOutput,
        vrfProof: validator.vrfProof,
        stake: effectiveStake,
        priority: selectionResult.priority,
        selected: true,
      });
    }
  }

  // Sort by VRF priority (lower = higher priority)
  // This ensures deterministic ordering among selected validators
  selections.sort((a, b) => {
    if (a.priority < b.priority) return -1;
    if (a.priority > b.priority) return 1;
    return a.validator.localeCompare(b.validator);
  });

  // Limit to maxValidators
  return selections.slice(0, params.maxValidators);
}
```

### Verification Step Required

Add VRF verification when validator commits:

```typescript
export function verifyValidatorSelection(
  validator: PublicKey,
  vrfOutput: VRFOutput,
  vrfProof: VRFProof,
  claimId: string,
  blockNumber: bigint,
  previousBlockHash: string
): { valid: boolean; reason?: string } {
  const seed = generateVerificationSeed(claimId, blockNumber, previousBlockHash);

  const verification = vrfVerify(seed, validator, vrfProof);

  if (!verification.valid) {
    return { valid: false, reason: 'Invalid VRF proof' };
  }

  if (verification.output !== vrfOutput) {
    return { valid: false, reason: 'VRF output mismatch' };
  }

  return { valid: true };
}
```

---

## Fix 2: Race Condition - Claim Before Falsification

### Problem

In `core/rewards/vesting_engine.ts:138-178`, rewards can be claimed even when a challenge
is pending. The current flow:

1. Attacker submits false claim → rewards start vesting
2. Some rewards release over time → attacker claims them
3. Challenge initiated → vesting paused
4. Challenge succeeds → clawback only affects UNRELEASED rewards
5. Attacker keeps already-claimed rewards

### The Fix (Three-Part)

#### Part 1: Track Claimed Amount for Clawback

```typescript
// In core/rewards/types.ts
export interface RewardVesting {
  // ... existing fields ...

  // NEW: Track total amount claimed for potential clawback
  totalClaimed: bigint;

  // NEW: Clawback account - where clawed back funds are held
  clawbackSource?: string;  // DID of recipient for clawback recovery
}
```

#### Part 2: Implement Full Clawback on Falsification

```typescript
// In core/rewards/clawback.ts
export function calculateFullClawback(
  vesting: RewardVesting,
  currentBlock: bigint
): ClawbackResult {
  // Calculate what can be clawed back
  const unreleasedAmount = vesting.totalAmount - vesting.releasedAmount - vesting.reserveAmount;

  // NEW: Clawback 30% of already-claimed amounts
  const claimedClawbackRate = 0.30;
  const claimedClawback = (vesting.totalClaimed * BigInt(Math.floor(claimedClawbackRate * 100))) / 100n;

  // Total clawback = unreleased + reserve + portion of claimed
  const totalClawback = unreleasedAmount + vesting.reserveAmount + claimedClawback;

  return {
    unreleasedClawback: unreleasedAmount,
    reserveClawback: vesting.reserveAmount,
    claimedClawback: claimedClawback,
    totalClawback: totalClawback,
    // Flag that recipient account needs balance reduction
    requiresBalanceClawback: claimedClawback > 0n,
  };
}
```

#### Part 3: Add Claim Delay to Allow Challenge Detection

```typescript
// In core/rewards/types.ts
export const CLAIM_DELAY_BLOCKS = 100n;  // ~7 minutes at 2521ms blocks

// In core/rewards/vesting_engine.ts
export function claimRewards(
  vesting: RewardVesting,
  payload: ClaimRewardsPayload,
  currentBlock: bigint
): { updatedVesting: RewardVesting; claimedAmount: bigint; effectiveBlock: bigint } {
  // Validate claimer
  if (payload.claimer !== vesting.recipient) {
    throw new VestingError('Only recipient can claim rewards', 'NOT_RECIPIENT');
  }

  // NEW: Check if there's a pending challenge
  if (vesting.pendingChallengeId) {
    throw new VestingError(
      'Cannot claim while challenge is pending',
      'CHALLENGE_PENDING'
    );
  }

  // Update release first
  const updated = updateRelease(vesting, currentBlock);

  // Check if there's anything to claim
  if (updated.claimableAmount === 0n) {
    throw new VestingError('No rewards available to claim', 'NOTHING_TO_CLAIM');
  }

  // Determine claim amount
  const claimAmount = payload.amount && payload.amount > 0n
    ? (payload.amount > updated.claimableAmount ? updated.claimableAmount : payload.amount)
    : updated.claimableAmount;

  // NEW: Claims are delayed - funds are not available until effectiveBlock
  const effectiveBlock = currentBlock + CLAIM_DELAY_BLOCKS;

  // Update vesting
  const finalVesting: RewardVesting = {
    ...updated,
    claimableAmount: updated.claimableAmount - claimAmount,
    totalClaimed: updated.totalClaimed + claimAmount,  // Track for clawback
    // NEW: Store pending claim
    pendingClaim: {
      amount: claimAmount,
      requestedAtBlock: currentBlock,
      effectiveAtBlock: effectiveBlock,
    },
    updatedAt: Date.now(),
  };

  return {
    updatedVesting: finalVesting,
    claimedAmount: claimAmount,
    effectiveBlock: effectiveBlock,
  };
}

// NEW: Finalize pending claim after delay
export function finalizeClaim(
  vesting: RewardVesting,
  currentBlock: bigint
): { updatedVesting: RewardVesting; transferAmount: bigint } | null {
  if (!vesting.pendingClaim) {
    return null;
  }

  // Check if delay has passed
  if (currentBlock < vesting.pendingClaim.effectiveAtBlock) {
    return null;  // Not yet effective
  }

  // Check if challenge was initiated during delay
  if (vesting.pendingChallengeId) {
    // Cancel the pending claim
    return {
      updatedVesting: {
        ...vesting,
        claimableAmount: vesting.claimableAmount + vesting.pendingClaim.amount,
        totalClaimed: vesting.totalClaimed - vesting.pendingClaim.amount,
        pendingClaim: undefined,
      },
      transferAmount: 0n,  // Claim cancelled
    };
  }

  // Finalize the claim
  return {
    updatedVesting: {
      ...vesting,
      pendingClaim: undefined,
    },
    transferAmount: vesting.pendingClaim.amount,
  };
}
```

---

## Fix 3: Stake Aging (Flash Loan Protection)

### Problem

Attacker can flash-loan LGM, stake, vote, unstake, repay - all in one transaction.

### The Fix

```typescript
// In consensus/pot/types.ts
export interface Validator {
  // ... existing fields ...

  // NEW: Track when stake was deposited
  stakeHistory: StakeDeposit[];

  // NEW: Effective stake only counts aged deposits
  effectiveStake: bigint;
}

export interface StakeDeposit {
  amount: bigint;
  depositedAtBlock: bigint;
  maturesAtBlock: bigint;
}

export const STAKE_MATURITY_BLOCKS = 1000n;  // ~42 minutes

// In consensus/pot/validator_selection.ts
export function calculateEffectiveStake(
  validator: Validator,
  currentBlock: bigint
): bigint {
  let effective = 0n;

  for (const deposit of validator.stakeHistory) {
    // Only count deposits that have matured
    if (currentBlock >= deposit.maturesAtBlock) {
      effective += deposit.amount;
    }
  }

  return effective;
}

// Update all stake references to use effectiveStake
export function calculateVotingPower(
  validator: Validator,
  currentBlock: bigint
): bigint {
  // Use effective stake, not raw stake
  const stakeWeight = calculateEffectiveStake(validator, currentBlock) +
                      calculateEffectiveDelegatedStake(validator, currentBlock);

  const reputationMultiplier = 0.5 + validator.reputation;

  return BigInt(Math.floor(Number(stakeWeight) * reputationMultiplier));
}
```

---

## Fix 4: Reveal Bond (Selective Withholding Protection)

### Problem

Validators can commit to verification rounds, see other reveals, and only reveal if
their vote would be on the winning side.

### The Fix

```typescript
// In consensus/pot/types.ts
export interface VerificationCommitment {
  validator: PublicKey;
  commitmentHash: string;
  revealBond: bigint;  // NEW: Forfeited if reveal not provided
  committedAtBlock: bigint;
}

export const REVEAL_BOND_RATE = 0.01;  // 1% of stake

// In consensus/pot/verification_round.ts
export function addCommitment(
  round: VerificationRound,
  validator: PublicKey,
  commitment: string,
  signature: string,
  validatorStake: bigint,
  currentBlock: bigint
): VerificationRound {
  // Calculate required reveal bond
  const revealBond = (validatorStake * BigInt(Math.floor(REVEAL_BOND_RATE * 10000))) / 10000n;

  // Bond must be locked when committing
  // This requires a separate bond transaction before commit

  const commitmentRecord: VerificationCommitment = {
    validator,
    commitmentHash: commitment,
    revealBond,
    committedAtBlock: currentBlock,
  };

  return {
    ...round,
    commitments: [...round.commitments, commitmentRecord],
  };
}

// At reveal deadline
export function handleRevealDeadline(
  round: VerificationRound,
  currentBlock: bigint
): {
  updatedRound: VerificationRound;
  slashedValidators: Array<{ validator: PublicKey; bondForfeited: bigint }>;
} {
  const slashed: Array<{ validator: PublicKey; bondForfeited: bigint }> = [];

  for (const commitment of round.commitments) {
    const hasRevealed = round.reveals.some(r => r.validator === commitment.validator);

    if (!hasRevealed) {
      // Validator committed but didn't reveal - forfeit bond
      slashed.push({
        validator: commitment.validator,
        bondForfeited: commitment.revealBond,
      });
    }
  }

  return {
    updatedRound: round,
    slashedValidators: slashed,
  };
}
```

---

## Implementation Priority

1. **Immediate** (blocks consensus attacks):
   - Fix 1: VRF-based validator selection
   - Fix 3: Stake aging

2. **Urgent** (prevents economic extraction):
   - Fix 2: Claim delay + full clawback

3. **Important** (prevents gaming):
   - Fix 4: Reveal bond

---

## Testing Requirements

Each fix requires:
1. Unit tests for new functions
2. Integration tests for full flows
3. Economic simulation for attack resistance
4. Adversarial testing (red team)

---

*These fixes address the CRITICAL and HIGH severity vulnerabilities identified in POT_VULNERABILITY_ANALYSIS.md*
