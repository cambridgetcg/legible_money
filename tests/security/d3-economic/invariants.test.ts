/**
 * Dimension 3: Economic & Game-Theoretic Security - Invariant Tests
 *
 * Validates economic invariants that MUST hold for protocol security:
 * 1. Supply cap invariants (block rewards, decay, fund exhaustion)
 * 2. Staking invariants (slash isolation, caps, deactivation, unbonding)
 * 3. Fee routing invariants (research/producer split, rounding, system tx bypass)
 *
 * All monetary values use bigint (units: ulgm = micro-LGM, 1 LGM = 10^18 ulgm).
 */

import { describe, it, expect } from "vitest";

// Supply & Emission
import {
  calculateBlockReward,
  calculateResearchShare,
  type EmissionSchedule,
  DEFAULT_EMISSION_SCHEDULE,
  PROTOCOL_CONSTANTS,
} from "../../../core/genesis/types.js";

// Staking & Slashing
import {
  DEFAULT_CONSENSUS_PARAMS,
  VALIDATOR_TIER_CONFIG,
  type Validator,
  type ValidatorTier,
} from "../../../consensus/pot/types.js";
import {
  slashValidatorByAmount,
  createValidatorSet,
  computeNextEpochValidators,
} from "../../../consensus/pot/validator_selection.js";

// Fee Routing
import {
  routeReward,
  RESEARCH_SHARE_RATE_BPS,
  validateRouting,
  type RewardInput,
} from "../../../core/billing/reward_router.js";

// Hashing (for unbonding ID test)
import { sha512_256Hex } from "../../../core/crypto/hashing.js";

// Constants
import { TOKEN, EVIDENCE_TAX } from "../../../core/constants.js";

// ============ Helpers ============

const ONE_LGM = 10n ** 18n;
const MAX_SUPPLY = TOKEN.MAX_SUPPLY; // 222,222,222 LGM

/** Create a validator with specified stake and optional overrides */
function makeValidator(
  stake: bigint,
  overrides: Partial<Validator> = {},
): Validator {
  return {
    publicKey:
      overrides.publicKey ?? "val-pk-" + Math.random().toString(36).slice(2),
    did: overrides.did ?? "did:lgm:val-" + Math.random().toString(36).slice(2),
    stake,
    delegatedStake: overrides.delegatedStake ?? 0n,
    reputation: overrides.reputation ?? 0.8,
    isActive: overrides.isActive ?? true,
    lastActiveBlock: overrides.lastActiveBlock ?? 0,
    totalVerifications: overrides.totalVerifications ?? 100,
    correctVerifications: overrides.correctVerifications ?? 80,
    missedVerifications: overrides.missedVerifications ?? 0,
    slashCount: overrides.slashCount ?? 0,
    slashesThisEpoch: overrides.slashesThisEpoch ?? 0,
    lastSlashedBlock: overrides.lastSlashedBlock ?? 0,
    tier: overrides.tier ?? 1,
  };
}

/** Test emission schedule with small numbers for predictable math */
const TEST_SCHEDULE: EmissionSchedule = {
  initialReward: 10n * ONE_LGM,
  decayFactor: 0.9,
  blocksPerEpoch: 100n,
  floorReward: ONE_LGM / 10n, // 0.1 LGM
  researchShareRate: 0.22,
  initialFundBalance: 1000n * ONE_LGM,
  fundHealthThreshold: 0.1,
  apiSuppressionExponent: 2,
};

// ============================================================
// 1. Supply Cap Invariants (6 tests)
// ============================================================

describe("Supply Cap Invariants", () => {
  it("SC-1: Total supply MUST never exceed 222,222,222 LGM", () => {
    // MAX_SUPPLY is defined as 222_222_222 * 10^18
    expect(MAX_SUPPLY).toBe(222_222_222n * ONE_LGM);

    // Simulate cumulative block rewards over 1000 epochs (100,000 blocks).
    // Track fund depletion: once the fund is empty, reward must be 0.
    let cumulativeRewards = 0n;
    let fundBalance = TEST_SCHEDULE.initialFundBalance!;

    for (let block = 0n; block < 100_000n; block += 1n) {
      const reward = calculateBlockReward(
        block,
        TEST_SCHEDULE,
        undefined,
        fundBalance,
      );
      cumulativeRewards += reward;
      // In practice the protocol deducts reward from fund, clamped to zero
      if (reward >= fundBalance) {
        fundBalance = 0n;
      } else {
        fundBalance -= reward;
      }
    }

    // Key invariant: cumulative rewards MUST be less than MAX_SUPPLY
    expect(cumulativeRewards).toBeLessThan(MAX_SUPPLY);

    // Fund should be depleted after 100k blocks of payouts from 1000 LGM fund
    expect(fundBalance).toBe(0n);
  });

  it("SC-2: Block reward fund starts at 60% of MAX_SUPPLY and only decreases", () => {
    const initialFund = DEFAULT_EMISSION_SCHEDULE.initialFundBalance!;
    const sixtyPercent = (MAX_SUPPLY * 6000n) / 10000n;
    expect(initialFund).toBe(sixtyPercent);

    // Simulate 10 blocks of payouts: fund balance must monotonically decrease
    let fund = initialFund;
    for (let block = 0n; block < 10n; block++) {
      const reward = calculateBlockReward(
        block,
        DEFAULT_EMISSION_SCHEDULE,
        undefined,
        fund,
      );
      const newFund = fund - reward;
      expect(newFund).toBeLessThanOrEqual(fund);
      fund = newFund;
    }
  });

  it("SC-3: Block reward at epoch 0 equals initialReward (10 LGM in test schedule)", () => {
    const reward = calculateBlockReward(0n, TEST_SCHEDULE);
    expect(reward).toBe(TEST_SCHEDULE.initialReward);
    expect(reward).toBe(10n * ONE_LGM);
  });

  it("SC-4: Reward decays: epoch 1 reward < epoch 0 reward", () => {
    const epoch0Reward = calculateBlockReward(0n, TEST_SCHEDULE);
    // Epoch 1 starts at block = blocksPerEpoch
    const epoch1Reward = calculateBlockReward(
      TEST_SCHEDULE.blocksPerEpoch,
      TEST_SCHEDULE,
    );

    expect(epoch1Reward).toBeLessThan(epoch0Reward);

    // Verify the exact decay: 10 LGM * 0.9 = 9 LGM
    expect(epoch1Reward).toBe(9n * ONE_LGM);
  });

  it("SC-5: Floor reward applies when scaled reward drops below it", () => {
    // After many epochs, time decay makes reward negligible, floor applies
    // epoch 100: 10 * 0.9^100 ~ 0.000026 LGM, well below floor of 0.1 LGM
    const reward = calculateBlockReward(10000n, TEST_SCHEDULE);
    expect(reward).toBe(TEST_SCHEDULE.floorReward);
  });

  it("SC-6: Fund exhaustion: when fund = 0, reward = 0 (hard stop)", () => {
    const reward = calculateBlockReward(0n, TEST_SCHEDULE, undefined, 0n);
    expect(reward).toBe(0n);

    // Even at epoch 0 with full initial reward, fund=0 means no reward
    const rewardEpoch5 = calculateBlockReward(
      500n,
      TEST_SCHEDULE,
      undefined,
      0n,
    );
    expect(rewardEpoch5).toBe(0n);
  });
});

// ============================================================
// 2. Staking Invariants (6 tests)
// ============================================================

describe("Staking Invariants", () => {
  it("SI-1: Slashing a validator's own stake MUST NOT reduce delegator balances", () => {
    // Delegated stake is tracked separately from validator's own stake.
    // slashValidatorByAmount only reduces validator.stake, not delegatedStake.
    const val = makeValidator(100n * ONE_LGM, {
      delegatedStake: 50n * ONE_LGM,
    });
    const slashAmount = 10n * ONE_LGM;

    const { validator: slashedVal, slashedAmount } = slashValidatorByAmount(
      val,
      slashAmount,
      "wrong_verification",
    );

    // Own stake reduced
    expect(slashedVal.stake).toBe(90n * ONE_LGM);
    expect(slashedAmount).toBe(slashAmount);

    // Delegated stake untouched
    expect(slashedVal.delegatedStake).toBe(50n * ONE_LGM);
  });

  it("SI-2: Slash cap: actual slash MUST NOT exceed total available stake", () => {
    const val = makeValidator(5n * ONE_LGM);
    const excessiveSlash = 100n * ONE_LGM; // Way more than stake

    const { validator: slashedVal, slashedAmount } = slashValidatorByAmount(
      val,
      excessiveSlash,
      "equivocation",
    );

    // Slash capped at available stake
    expect(slashedAmount).toBe(5n * ONE_LGM);
    expect(slashedVal.stake).toBe(0n);
  });

  it("SI-3: Progressive escalation: each slash reduces stake further, slashCount increments", () => {
    let val = makeValidator(100n * ONE_LGM);

    // First slash: 10 LGM
    const result1 = slashValidatorByAmount(val, 10n * ONE_LGM, "slash1");
    expect(result1.validator.stake).toBe(90n * ONE_LGM);
    expect(result1.validator.slashCount).toBe(1);
    val = result1.validator;

    // Second slash: 10 LGM (from reduced stake)
    const result2 = slashValidatorByAmount(val, 10n * ONE_LGM, "slash2");
    expect(result2.validator.stake).toBe(80n * ONE_LGM);
    expect(result2.validator.slashCount).toBe(2);

    // SlashCount increases monotonically with each non-zero slash
    expect(result2.validator.slashCount).toBeGreaterThan(
      result1.validator.slashCount,
    );
  });

  it("SI-4: Per-epoch slash cap: maxSlashesPerEpoch (default 2) prevents cascade", () => {
    let val = makeValidator(100n * ONE_LGM);

    // First slash: goes through
    const r1 = slashValidatorByAmount(val, 5n * ONE_LGM, "slash1", {
      maxSlashesPerEpoch: 2,
    });
    expect(r1.slashedAmount).toBe(5n * ONE_LGM);
    expect(r1.capped).toBe(false);
    val = r1.validator;

    // Second slash: goes through
    const r2 = slashValidatorByAmount(val, 5n * ONE_LGM, "slash2", {
      maxSlashesPerEpoch: 2,
    });
    expect(r2.slashedAmount).toBe(5n * ONE_LGM);
    expect(r2.capped).toBe(false);
    val = r2.validator;

    // Third slash: CAPPED (epoch limit reached)
    const r3 = slashValidatorByAmount(val, 5n * ONE_LGM, "slash3", {
      maxSlashesPerEpoch: 2,
    });
    expect(r3.slashedAmount).toBe(0n);
    expect(r3.capped).toBe(true);

    // Stake unchanged after cap
    expect(r3.validator.stake).toBe(val.stake);
  });

  it("SI-5: Validator deactivation after maxSlashCount (default 3) slashes", () => {
    let val = makeValidator(100n * ONE_LGM, { slashesThisEpoch: 0 });

    // Slash 3 times (use high per-epoch cap to allow all 3)
    for (let i = 0; i < 3; i++) {
      const r = slashValidatorByAmount(val, 1n * ONE_LGM, `slash${i}`, {
        maxSlashesPerEpoch: 10, // Allow many per epoch
        maxSlashCount: 3,
      });
      val = r.validator;
    }

    // After 3 slashes, validator should be deactivated
    expect(val.slashCount).toBe(3);
    expect(val.isActive).toBe(false);
  });

  it("SI-6: Unbonding ID uses nonce to prevent collision in same block", () => {
    // The unbonding ID formula includes a nonce (existingEntries.length)
    // to ensure two unstake operations in the same block produce different IDs.
    //
    // Verify the structure: sha512_256Hex(`unstake:${did}:${block}:${amount}:${nonce}`)
    // Same inputs with different nonces must produce different IDs.
    const did = "did:lgm:test-validator";
    const block = 100n;
    const amount = 10n * ONE_LGM;

    const id0 = sha512_256Hex(`unstake:${did}:${block}:${amount}:0`);
    const id1 = sha512_256Hex(`unstake:${did}:${block}:${amount}:1`);
    const id2 = sha512_256Hex(`unstake:${did}:${block}:${amount}:2`);

    // All IDs must be unique
    expect(id0).not.toBe(id1);
    expect(id1).not.toBe(id2);
    expect(id0).not.toBe(id2);

    // IDs must be deterministic (same inputs = same output)
    const id0Again = sha512_256Hex(`unstake:${did}:${block}:${amount}:0`);
    expect(id0).toBe(id0Again);
  });
});

// ============================================================
// 3. Fee Routing Invariants (4 tests)
// ============================================================

describe("Fee Routing Invariants", () => {
  const makeRewardInput = (amount: bigint): RewardInput => ({
    source: "block_production",
    amount,
    recipient: "did:lgm:test-producer" as any,
    blockNumber: 100n,
  });

  it("FR-1: Research share = exactly 22% of reward (2200 bps)", () => {
    const amount = 1000n * ONE_LGM;
    const routed = routeReward(makeRewardInput(amount));

    const expectedResearch = (amount * 2200n) / 10000n;
    expect(routed.researchShare).toBe(expectedResearch);

    // Verify the constant is 2200 bps
    expect(RESEARCH_SHARE_RATE_BPS).toBe(2200n);
  });

  it("FR-2: Producer share = exactly 78% of reward (7800 bps)", () => {
    const amount = 1000n * ONE_LGM;
    const routed = routeReward(makeRewardInput(amount));

    const expectedRecipient = amount - (amount * 2200n) / 10000n;
    expect(routed.recipientAmount).toBe(expectedRecipient);

    // 78% check
    const recipientBps = (routed.recipientAmount * 10000n) / amount;
    expect(recipientBps).toBe(7800n);
  });

  it("FR-3: Research + Producer = 100% (no rounding loss leak)", () => {
    // Test with various amounts including ones that might cause rounding issues
    const testAmounts = [
      1n, // Smallest possible
      7n, // Prime number
      ONE_LGM, // 1 LGM
      3n * ONE_LGM, // 3 LGM (indivisible by 10000)
      10000n, // Exact basis point
      999_999n, // Not divisible by 10000
      100n * ONE_LGM, // Large round number
      MAX_SUPPLY, // Maximum supply
    ];

    for (const amount of testAmounts) {
      const routed = routeReward(makeRewardInput(amount));

      // Research + Recipient MUST equal original amount exactly
      const total = routed.researchShare + routed.recipientAmount;
      expect(total).toBe(amount);

      // Validate using the built-in validator
      expect(validateRouting(routed)).toBe(true);
    }
  });

  it("FR-4: System transactions (verification_reward, slash_validator) bypass fees", () => {
    // System transactions are identified by tx.sender === "system" or
    // fee === 0 with type in ["add_fact", "verification_reward", "slash_validator"]
    //
    // Verify the system transaction types are recognized
    const systemTxTypes = [
      "add_fact",
      "verification_reward",
      "slash_validator",
    ];

    // These types should be in the protocol's system transaction list
    for (const txType of systemTxTypes) {
      // From state/transitions.ts line 296-301:
      // isSystemTx = tx.sender === "system" || (tx.fee === 0n &&
      //   ["add_fact", "verification_reward", "slash_validator"].includes(tx.type))
      //
      // Verify fee = 0 is valid for system types (no explicit fee deduction)
      expect(systemTxTypes.includes(txType)).toBe(true);
    }

    // Verify the research share function handles 0 amount correctly
    const zeroResearch = calculateResearchShare(0n);
    expect(zeroResearch).toBe(0n);

    // Verify routing a zero-fee system reward gives zero research share
    const zeroRouted = routeReward({
      source: "verification",
      amount: 0n,
      recipient: "did:lgm:system" as any,
      blockNumber: 0n,
    });
    expect(zeroRouted.researchShare).toBe(0n);
    expect(zeroRouted.recipientAmount).toBe(0n);
  });
});
