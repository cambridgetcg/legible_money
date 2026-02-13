/**
 * DIMENSION 2: Consensus & Validator Security Tests
 * File 2: VRF Selection Fairness & Finality
 *
 * Tests:
 *   - VRF-based selection proportionality (Monte Carlo)
 *   - Finality depth guarantees
 *   - Slashing mechanics and bounds
 *   - Voting power calculation correctness
 */

import { describe, it, expect, beforeEach } from "vitest";
import { generateKeypair } from "../../../core/crypto/keys.js";
import { vrfGenerate } from "../../../core/crypto/vrf.js";
import { isValidatorSelected } from "../../../core/crypto/vrf.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";
import {
  getEligibleValidators,
  createValidatorSet,
  slashValidatorByAmount,
  calculateVotingPower,
  computeNextEpochValidators,
} from "../../../consensus/pot/validator_selection.js";
import {
  isBlockFinalized,
  getConfirmations,
  isReorgAllowed,
  FINALITY_PARAMS,
  type Checkpoint,
} from "../../../consensus/pot/finality.js";
import {
  DEFAULT_CONSENSUS_PARAMS,
  type Validator,
  type ValidatorSet,
  type ConsensusParams,
} from "../../../consensus/pot/types.js";

// ============ Test Helpers ============

const LGM = 10n ** 18n;

function makeKeypair() {
  return generateKeypair();
}

function createValidator(overrides: Partial<Validator> = {}): Validator {
  const kp = makeKeypair();
  return {
    publicKey: kp.publicKey,
    did: `did:lgm:${kp.publicKey.slice(0, 16)}`,
    stake: 100n * LGM,
    delegatedStake: 0n,
    reputation: 0.5,
    isActive: true,
    lastActiveBlock: 0,
    totalVerifications: 0,
    correctVerifications: 0,
    missedVerifications: 0,
    slashCount: 0,
    slashesThisEpoch: 0,
    lastSlashedBlock: 0,
    ...overrides,
  };
}

// ============ 1. Selection Fairness Monte Carlo (4 tests) ============

describe("Selection Fairness Monte Carlo", () => {
  it("with 4 validators (40/30/20/10 stake), selection frequency within 15% of stake proportion over 1000 trials", () => {
    // Create 4 validators with known stake distribution
    const kps = Array.from({ length: 4 }, () => makeKeypair());
    const stakeAmounts = [400n * LGM, 300n * LGM, 200n * LGM, 100n * LGM];
    const totalStake = stakeAmounts.reduce((a, b) => a + b, 0n);

    const selectionCounts = [0, 0, 0, 0];
    const trials = 1000;

    for (let trial = 0; trial < trials; trial++) {
      // Each trial: generate VRF for each validator with a unique seed
      const trialSeed = `monte-carlo-trial-${trial}`;

      for (let i = 0; i < 4; i++) {
        const { output } = vrfGenerate(trialSeed, kps[i].privateKey);
        const result = isValidatorSelected(
          output,
          stakeAmounts[i],
          totalStake,
          1, // targetValidators = 1 (single proposer slot)
        );
        if (result.selected) {
          selectionCounts[i]++;
        }
      }
    }

    // Check each validator's selection frequency is within 15% of stake proportion
    const expectedProportions = [0.4, 0.3, 0.2, 0.1];
    const totalSelections = selectionCounts.reduce((a, b) => a + b, 0);

    // Need at least some selections to test proportionality
    expect(totalSelections).toBeGreaterThan(0);

    for (let i = 0; i < 4; i++) {
      const observedProportion = selectionCounts[i] / totalSelections;
      const expected = expectedProportions[i];
      const deviation = Math.abs(observedProportion - expected);

      // Within 15% absolute deviation of expected proportion
      expect(deviation).toBeLessThan(0.15);
    }
  });

  it("min-stake validator (0.111 LGM) MUST eventually be selected (not starved)", () => {
    const kp = makeKeypair();
    const minStake = 111n * 10n ** 15n; // 0.111 LGM
    const totalStake = 10000n * LGM; // Large total stake

    let selected = false;
    // With 10000 trials and targetValidators=22 (max validators),
    // even a tiny stake validator should be selected at least once
    for (let trial = 0; trial < 10000; trial++) {
      const seed = `min-stake-trial-${trial}`;
      const { output } = vrfGenerate(seed, kp.privateKey);
      const result = isValidatorSelected(
        output,
        minStake,
        totalStake,
        22, // maxValidators target
      );
      if (result.selected) {
        selected = true;
        break;
      }
    }

    expect(selected).toBe(true);
  });

  it("validator with 0 stake MUST never be selected", () => {
    const kp = makeKeypair();
    const zeroStake = 0n;
    const totalStake = 1000n * LGM;

    for (let trial = 0; trial < 1000; trial++) {
      const seed = `zero-stake-trial-${trial}`;
      const { output } = vrfGenerate(seed, kp.privateKey);
      const result = isValidatorSelected(output, zeroStake, totalStake, 22);
      expect(result.selected).toBe(false);
    }
  });

  it("inactive/deactivated validator MUST never be selected even with high stake", () => {
    const kpActive = makeKeypair();
    const kpInactive = makeKeypair();

    const validatorSet = createValidatorSet(
      [
        {
          publicKey: kpActive.publicKey,
          did: "did:lgm:active",
          stake: 100n * LGM,
        },
        {
          publicKey: kpInactive.publicKey,
          did: "did:lgm:inactive",
          stake: 1000n * LGM, // High stake but will be deactivated
        },
      ],
      0n,
    );

    // Deactivate the second validator
    const inactiveVal = validatorSet.validators.get(kpInactive.publicKey)!;
    validatorSet.validators.set(kpInactive.publicKey, {
      ...inactiveVal,
      isActive: false,
    });

    const eligible = getEligibleValidators(
      validatorSet,
      DEFAULT_CONSENSUS_PARAMS,
    );

    // Only the active validator should be eligible
    expect(eligible.length).toBe(1);
    expect(eligible[0].validator).toBe(kpActive.publicKey);

    // Inactive validator must not appear in eligible list
    const inactiveInEligible = eligible.find(
      (e) => e.validator === kpInactive.publicKey,
    );
    expect(inactiveInEligible).toBeUndefined();
  });
});

// ============ 2. Finality Security (4 tests) ============

describe("Finality Security", () => {
  it("block at depth >= 19 is considered final", () => {
    const blockNumber = 100n;
    const currentBlock = blockNumber + 19n; // depth = 19
    const checkpoints: Checkpoint[] = [];

    expect(isBlockFinalized(blockNumber, currentBlock, checkpoints)).toBe(true);
  });

  it("block at depth 18 is NOT final", () => {
    const blockNumber = 100n;
    const currentBlock = blockNumber + 18n; // depth = 18
    const checkpoints: Checkpoint[] = [];

    expect(isBlockFinalized(blockNumber, currentBlock, checkpoints)).toBe(
      false,
    );
  });

  it("attempting to replace a finalized block MUST fail (reorg blocked)", () => {
    const currentBlock = 200n;
    const finalizedBlock = currentBlock - 19n; // block 181 is finalized
    const lastCheckpoint = 0n;

    // Try to reorg to a block before the finalized depth
    const reorgTo = finalizedBlock - 1n; // trying to reorg past finalized block

    const result = isReorgAllowed(currentBlock, reorgTo, lastCheckpoint);
    expect(result.allowed).toBe(false);
    expect(result.reason).toContain("depth");
  });

  it("chain with 19 consecutive valid blocks achieves finality", () => {
    const genesisBlock = 0n;
    const checkpoints: Checkpoint[] = [];

    // Build up 19 blocks of confirmations
    for (let i = 1; i <= 19; i++) {
      const currentBlock = BigInt(i);
      const isGenesisFinal = isBlockFinalized(
        genesisBlock,
        currentBlock,
        checkpoints,
      );
      if (i < 19) {
        expect(isGenesisFinal).toBe(false);
      } else {
        expect(isGenesisFinal).toBe(true);
      }
    }

    // At block 19, genesis (block 0) has 19 confirmations -> final
    expect(getConfirmations(genesisBlock, 19n)).toBe(19n);
  });
});

// ============ 3. Slashing Mechanics (6 tests) ============

describe("Slashing Mechanics", () => {
  it("wrong verification slash = 6% of stake", () => {
    const stake = 1000n * LGM;
    const wrongSlashRate = DEFAULT_CONSENSUS_PARAMS.wrongVerificationSlash; // 0.06
    const expectedSlash =
      (stake * BigInt(Math.floor(wrongSlashRate * 10000))) / 10000n;

    const validator = createValidator({ stake });

    const { validator: slashed, slashedAmount } = slashValidatorByAmount(
      validator,
      expectedSlash,
      "wrong_verification",
    );

    expect(slashedAmount).toBe(expectedSlash);
    expect(slashed.stake).toBe(stake - expectedSlash);

    // Verify the expected percentage is 6%
    // 6% of 1000 LGM = 60 LGM
    expect(expectedSlash).toBe(60n * LGM);
  });

  it("missed verification slash = 0.01% of stake", () => {
    const stake = 1000n * LGM;
    const missedSlashRate = DEFAULT_CONSENSUS_PARAMS.missedVerificationSlash; // 0.0001
    const expectedSlash =
      (stake * BigInt(Math.floor(missedSlashRate * 10000))) / 10000n;

    const validator = createValidator({ stake });

    const { slashedAmount } = slashValidatorByAmount(
      validator,
      expectedSlash,
      "missed_verification",
    );

    expect(slashedAmount).toBe(expectedSlash);
    // 0.01% of 1000 LGM = 0.1 LGM
    expect(expectedSlash).toBe(1n * 10n ** 17n);
  });

  it("per-epoch slash cap (default 2): third slash in same epoch MUST be capped", () => {
    let validator = createValidator({
      stake: 1000n * LGM,
      slashesThisEpoch: 0,
    });

    // First slash: accepted
    const slashAmount = 10n * LGM;
    const r1 = slashValidatorByAmount(validator, slashAmount, "first_slash", {
      maxSlashesPerEpoch: 2,
    });
    expect(r1.capped).toBe(false);
    expect(r1.slashedAmount).toBe(slashAmount);
    validator = r1.validator;

    // Second slash: accepted
    const r2 = slashValidatorByAmount(validator, slashAmount, "second_slash", {
      maxSlashesPerEpoch: 2,
    });
    expect(r2.capped).toBe(false);
    expect(r2.slashedAmount).toBe(slashAmount);
    validator = r2.validator;

    // Third slash: MUST be capped (slashesThisEpoch = 2 >= maxSlashesPerEpoch)
    const r3 = slashValidatorByAmount(validator, slashAmount, "third_slash", {
      maxSlashesPerEpoch: 2,
    });
    expect(r3.capped).toBe(true);
    expect(r3.slashedAmount).toBe(0n);
    // Stake unchanged from third attempt
    expect(r3.validator.stake).toBe(validator.stake);
  });

  it("slash decay: after epoch with no slash, slashCount decrements by 1", () => {
    const kp = makeKeypair();
    const validatorSet = createValidatorSet(
      [
        {
          publicKey: kp.publicKey,
          did: "did:lgm:decaytest",
          stake: 100n * LGM,
        },
      ],
      0n,
    );

    // Manually set slashCount to 2 with no slashes this epoch
    const v = validatorSet.validators.get(kp.publicKey)!;
    validatorSet.validators.set(kp.publicKey, {
      ...v,
      slashCount: 2,
      slashesThisEpoch: 0, // No slashes this epoch
    });

    // Compute next epoch validators - R1 decay should apply
    const nextSet = computeNextEpochValidators(validatorSet, [], [], [], 1n);

    const updated = nextSet.validators.get(kp.publicKey)!;
    // slashCount should decay by 1 (from 2 to 1) since no slash this epoch
    expect(updated.slashCount).toBe(1);
    // slashesThisEpoch should be reset to 0
    expect(updated.slashesThisEpoch).toBe(0);
  });

  it("validator deactivated after maxSlashCount (3) cumulative slashes", () => {
    const validator = createValidator({
      stake: 1000n * LGM,
      slashCount: 2, // Already 2 slashes
      slashesThisEpoch: 0,
    });

    // Third slash should trigger deactivation (maxSlashCount default = 3)
    const slashAmount = 10n * LGM;
    const result = slashValidatorByAmount(
      validator,
      slashAmount,
      "third_slash",
      { maxSlashCount: 3 },
    );

    // slashCount goes from 2 to 3, which >= maxSlashCount (3) -> deactivated
    expect(result.validator.slashCount).toBe(3);
    expect(result.validator.isActive).toBe(false);
  });

  it("slash floor: 0-amount slash does not increment slash counter", () => {
    const validator = createValidator({
      stake: 1000n * LGM,
      slashCount: 1,
      slashesThisEpoch: 0,
    });

    // Slash with 0 amount (edge case)
    const result = slashValidatorByAmount(validator, 0n, "zero_slash");

    // R4: 0-amount slash should NOT increment slashCount
    expect(result.slashedAmount).toBe(0n);
    expect(result.validator.slashCount).toBe(1); // Unchanged
    expect(result.validator.slashesThisEpoch).toBe(0); // Unchanged
  });
});

// ============ 4. Voting Power Calculation (4 tests) ============

describe("Voting Power Calculation", () => {
  it("validator with reputation 1.0 gets 1.5x multiplier on base stake", () => {
    const validator = createValidator({
      stake: 100n * LGM,
      delegatedStake: 0n,
      reputation: 1.0,
    });

    const power = calculateVotingPower(validator);

    // multiplierBasisPoints = 5000 + (1.0 * 10000) = 15000
    // votingPower = (100 LGM * 15000) / 10000 = 150 LGM
    expect(power).toBe(150n * LGM);
  });

  it("validator with reputation 0.0 gets 0.5x multiplier", () => {
    const validator = createValidator({
      stake: 100n * LGM,
      delegatedStake: 0n,
      reputation: 0.0,
    });

    const power = calculateVotingPower(validator);

    // multiplierBasisPoints = 5000 + (0.0 * 10000) = 5000
    // votingPower = (100 LGM * 5000) / 10000 = 50 LGM
    expect(power).toBe(50n * LGM);
  });

  it("delegated stake adds to base stake before multiplier", () => {
    const validator = createValidator({
      stake: 100n * LGM,
      delegatedStake: 50n * LGM,
      reputation: 0.5, // multiplier = 1.0x
    });

    const power = calculateVotingPower(validator);

    // stakeWeight = 100 + 50 = 150 LGM
    // multiplierBasisPoints = 5000 + (0.5 * 10000) = 10000
    // votingPower = (150 LGM * 10000) / 10000 = 150 LGM
    expect(power).toBe(150n * LGM);
  });

  it("voting power uses bigint BPS arithmetic (no float precision loss)", () => {
    // Use a very large stake that would overflow Number precision
    const largeStake = 1111n * LGM; // 1111 LGM = 1.111 * 10^21

    // Use 0.75 for a clean float that avoids IEEE 754 rounding issues
    const reputation = 0.75;
    const validator = createValidator({
      stake: largeStake,
      delegatedStake: 0n,
      reputation,
    });

    const power = calculateVotingPower(validator);

    // Manual bigint calculation:
    // multiplierBasisPoints = 5000 + floor(0.75 * 10000) = 5000 + 7500 = 12500
    const expectedMultiplierBps =
      5000n + BigInt(Math.floor(reputation * 10000));
    expect(expectedMultiplierBps).toBe(12500n);

    const expectedPower = (largeStake * expectedMultiplierBps) / 10000n;
    expect(power).toBe(expectedPower);

    // The bigint result should be deterministic and precise
    expect(power).toBeGreaterThan(0n);
    expect(typeof power).toBe("bigint");

    // Verify this is exactly 1.25x the stake (0.5 + 0.75 = 1.25)
    // 1111 LGM * 1.25 = 1388.75 LGM -> truncated to 1388 LGM + remainder in wei
    const expected125x = (largeStake * 12500n) / 10000n;
    expect(power).toBe(expected125x);
  });
});
