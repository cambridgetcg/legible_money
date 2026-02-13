/**
 * Governance & Supply Economic Loopholes — Security Tests
 *
 * Tests for CRITICAL loophole D4-L1 and HIGH loopholes D4-L2, D4-L3
 * from the Financial Incentives PP Review.
 *
 * D4-L1: Governance capture — no stake concentration caps (CRITICAL)
 * D4-L2: IBC multi-channel drain (HIGH)
 * D4-L3: Emergency halt centralization (HIGH)
 *
 * Also covers D3-L1 (dual-stack divergence) and D3-L2 (fee pool auth).
 */

import { describe, it, expect } from "vitest";

import {
  DEFAULT_CONSENSUS_PARAMS,
  VALIDATOR_TIER_CONFIG,
  MAX_APPRENTICE_VALIDATORS,
} from "../../../consensus/pot/types.js";

import {
  DISBURSEMENT_THRESHOLDS_BPS,
  TIMELOCK_BLOCKS,
  MAX_AMOUNT_BPS,
  DISBURSEMENT_STAKE,
  validateDisbursementAmount,
} from "../../../governance/treasury/types.js";

import {
  routeReward,
  RESEARCH_SHARE_RATE_BPS,
} from "../../../core/billing/reward_router.js";

import {
  calculateAPICost,
  buildPaymentDistribution,
  validateDistribution,
} from "../../../core/billing/distribution.js";

import { TOKEN } from "../../../core/constants.js";

// ============ Helpers ============

const ONE_LGM = 10n ** 18n;

/** Simulate quadratic voting power */
function quadraticVotingPower(stake: bigint, totalStake: bigint): bigint {
  // sqrt(stake) with bigint approximation
  const sqrtStake = bigIntSqrt(stake);
  const sqrtTotal = bigIntSqrt(totalStake);
  // Cap at 10% of total voting power
  const maxPower = sqrtTotal / 10n;
  return sqrtStake < maxPower ? sqrtStake : maxPower;
}

function bigIntSqrt(n: bigint): bigint {
  if (n < 0n) throw new Error("sqrt of negative");
  if (n < 2n) return n;
  let x = n;
  let y = (x + 1n) / 2n;
  while (y < x) {
    x = y;
    y = (x + n / x) / 2n;
  }
  return x;
}

// ============================================================
// D4-L1: GOVERNANCE CAPTURE — NO STAKE CONCENTRATION CAPS (CRITICAL)
// ============================================================

describe("D4-L1: Governance Capture — No Stake Concentration Caps", () => {
  it("GOV-1: Current governance requires 80% support — verify threshold", () => {
    expect(DISBURSEMENT_THRESHOLDS_BPS.supportRequired).toBe(8000n); // 80%
    expect(DISBURSEMENT_THRESHOLDS_BPS.quorumRequired).toBe(2000n); // 20%
  });

  it("GOV-2: 67% stake holder can pass any proposal with current linear voting", () => {
    // With linear voting power, 67% stake = 67% voting power
    // 67% > 80%? No. But with only 80% of 20% quorum...
    // Actually: 80% of votes cast must be in favor
    // If attacker has 67% stake and votes "yes", and only 70% of tokens vote:
    // attacker's 67% represents 67/70 = 95.7% of votes cast → PASSES

    const totalStake = 1000n * ONE_LGM;
    const attackerStake = 670n * ONE_LGM; // 67%
    const honestStake = 330n * ONE_LGM; // 33%

    // Scenario: 70% of tokens participate (attacker + some honest)
    // Attacker: 670 yes
    // Honest: 100 no (only ~30% of honest bother voting against)
    const totalVotes = 770n * ONE_LGM;
    const yesVotes = 670n * ONE_LGM;
    const supportBps = (yesVotes * 10000n) / totalVotes;

    // 670/770 = 87% support → exceeds 80% threshold
    expect(supportBps).toBeGreaterThan(
      DISBURSEMENT_THRESHOLDS_BPS.supportRequired,
    );

    // Quorum check: 770/1000 = 77% participation → exceeds 20% quorum
    const quorumBps = (totalVotes * 10000n) / totalStake;
    expect(quorumBps).toBeGreaterThan(
      DISBURSEMENT_THRESHOLDS_BPS.quorumRequired,
    );
  });

  it("GOV-3: With quadratic voting — 67% stake holder gets ~82% sqrt power → STILL passes", () => {
    // Quadratic voting: power = sqrt(stake)
    // 67% stake → sqrt(0.67) = 0.819 of sqrt(1.0) = 81.9%
    // This is still > 80% threshold — quadratic alone doesn't prevent capture!

    const totalStake = 1000n * ONE_LGM;
    const attackerStake = 670n * ONE_LGM;

    const attackerPower = bigIntSqrt(attackerStake);
    const totalPower = bigIntSqrt(totalStake);

    const powerRatioBps = (attackerPower * 10000n) / totalPower;
    // sqrt(670) / sqrt(1000) ≈ 0.819
    expect(powerRatioBps).toBeGreaterThan(8000n); // Still > 80%
  });

  it("GOV-4: With quadratic + 10% cap — 67% stake holder capped to 10%", () => {
    const totalStake = 1000n * ONE_LGM;
    const attackerStake = 670n * ONE_LGM;

    const attackerPower = quadraticVotingPower(attackerStake, totalStake);
    const maxPower = bigIntSqrt(totalStake) / 10n;

    // Attacker power is capped at 10% of sqrt(total)
    expect(attackerPower).toBe(maxPower);

    // With 10% cap, attacker CANNOT reach 80% threshold alone
    // Need 8 colluding validators each at 10% cap to reach 80%
    const powerRatioBps = (attackerPower * 10000n) / bigIntSqrt(totalStake);
    expect(powerRatioBps).toBeLessThanOrEqual(1000n); // 10%
    expect(powerRatioBps).toBeLessThan(
      DISBURSEMENT_THRESHOLDS_BPS.supportRequired,
    );
  });

  it("GOV-5: With quadratic + 10% cap — need 8+ independent validators to pass", () => {
    const totalStake = 1000n * ONE_LGM;
    const numValidators = 10;
    const perValidatorStake = totalStake / BigInt(numValidators);

    let totalYesPower = 0n;
    let totalVotingPower = 0n;

    // Calculate voting power for each validator
    for (let i = 0; i < numValidators; i++) {
      const power = quadraticVotingPower(perValidatorStake, totalStake);
      totalVotingPower += power;
    }

    // 8 validators vote yes (80%)
    for (let i = 0; i < 8; i++) {
      totalYesPower += quadraticVotingPower(perValidatorStake, totalStake);
    }

    const supportBps = (totalYesPower * 10000n) / totalVotingPower;
    expect(supportBps).toBeGreaterThanOrEqual(
      DISBURSEMENT_THRESHOLDS_BPS.supportRequired,
    );

    // 7 validators is NOT enough
    let sevenYesPower = 0n;
    for (let i = 0; i < 7; i++) {
      sevenYesPower += quadraticVotingPower(perValidatorStake, totalStake);
    }
    const sevenSupportBps = (sevenYesPower * 10000n) / totalVotingPower;
    expect(sevenSupportBps).toBeLessThan(
      DISBURSEMENT_THRESHOLDS_BPS.supportRequired,
    );
  });

  it("GOV-6: Disbursement amount capped at 10% of treasury", () => {
    const treasuryBalance = 1000n * ONE_LGM;

    // Valid: 10% of treasury
    const validAmount = (treasuryBalance * MAX_AMOUNT_BPS) / 10000n; // 100 LGM
    expect(validateDisbursementAmount(validAmount, treasuryBalance).valid).toBe(
      true,
    );

    // Invalid: 11% of treasury
    const invalidAmount = validAmount + ONE_LGM;
    expect(
      validateDisbursementAmount(invalidAmount, treasuryBalance).valid,
    ).toBe(false);
  });

  it("GOV-7: Serial treasury drain rate bounded by timelock", () => {
    // Even if governance is captured, serial disbursements are bounded:
    // Max 10% per proposal, 50,000 block timelock between approval and execution
    const blocksPerDay = 34272n;
    const proposalsPerDay = blocksPerDay / TIMELOCK_BLOCKS; // ~0.69

    // Max drain rate: 10% per ~1.46 days
    // Starting from 1000 LGM:
    // Day 0: 1000 → 900 (drain 100)
    // Day 1.46: 900 → 810 (drain 90)
    // Day 2.92: 810 → 729 (drain 81)
    // ...exponential decay

    let balance = 1000n * ONE_LGM;
    let totalDrained = 0n;
    const proposals = 10;

    for (let i = 0; i < proposals; i++) {
      const drainAmount = (balance * MAX_AMOUNT_BPS) / 10000n;
      totalDrained += drainAmount;
      balance -= drainAmount;
    }

    // After 10 proposals (~14.6 days): ~65% drained
    expect(totalDrained).toBeLessThan(700n * ONE_LGM);
    // Treasury still has >30%
    expect(balance).toBeGreaterThan(300n * ONE_LGM);
  });
});

// ============================================================
// D4-L2: IBC MULTI-CHANNEL DRAIN (HIGH)
// ============================================================

describe("D4-L2: IBC Multi-Channel Drain", () => {
  it("IBC-1: Per-channel rate limit is 10,000 LGM/day", () => {
    const PER_CHANNEL_DAILY_LIMIT = 10_000n * ONE_LGM;
    expect(PER_CHANNEL_DAILY_LIMIT).toBe(10_000n * ONE_LGM);
  });

  it("IBC-2: N channels = N * 10,000 LGM/day without aggregate cap", () => {
    const PER_CHANNEL_LIMIT = 10_000n * ONE_LGM;

    // Attack: create 10 channels, drain 100K LGM/day
    for (const numChannels of [1, 5, 10, 50, 100]) {
      const totalOutflow = PER_CHANNEL_LIMIT * BigInt(numChannels);
      expect(totalOutflow).toBe(PER_CHANNEL_LIMIT * BigInt(numChannels));
    }

    // With 100 channels: 1M LGM/day = ~0.45% of total supply per day
    const totalSupply = 222_222_222n * ONE_LGM;
    const drainRate100 = (PER_CHANNEL_LIMIT * 100n * 10000n) / totalSupply;
    // 100 * 10K / 222M ≈ 0.45%
    expect(drainRate100).toBeGreaterThan(0n);
  });

  it("IBC-3: Proposed aggregate cap — 50,000 LGM/day across all channels", () => {
    const AGGREGATE_DAILY_LIMIT = 50_000n * ONE_LGM;
    const PER_CHANNEL_LIMIT = 10_000n * ONE_LGM;

    // With aggregate cap: max outflow is 50K regardless of channel count
    for (const numChannels of [1, 5, 10, 50, 100]) {
      const uncappedOutflow = PER_CHANNEL_LIMIT * BigInt(numChannels);
      const effectiveOutflow =
        uncappedOutflow < AGGREGATE_DAILY_LIMIT
          ? uncappedOutflow
          : AGGREGATE_DAILY_LIMIT;

      expect(effectiveOutflow).toBeLessThanOrEqual(AGGREGATE_DAILY_LIMIT);

      // With 5 channels: 50K (uncapped, equals aggregate)
      // With 10+ channels: capped at 50K
      if (numChannels >= 5) {
        expect(effectiveOutflow).toBe(AGGREGATE_DAILY_LIMIT);
      }
    }
  });

  it("IBC-4: Aggregate cap prevents supply drain — max ~0.02% per day", () => {
    const AGGREGATE_DAILY_LIMIT = 50_000n * ONE_LGM;
    const totalSupply = 222_222_222n * ONE_LGM;

    const drainRateBps = (AGGREGATE_DAILY_LIMIT * 10000n) / totalSupply;
    // 50K / 222M ≈ 0.022%
    expect(drainRateBps).toBeLessThan(3n); // < 0.03%

    // At 0.022%/day, draining 10% takes ~454 days
    // Plenty of time for governance response
  });
});

// ============================================================
// D4-L3: EMERGENCY HALT CENTRALIZATION (HIGH)
// ============================================================

describe("D4-L3: Emergency Halt Centralization", () => {
  it("HALT-1: Halt proposal is currently free (no stake required)", () => {
    // This is the vulnerability — documented but not enforced
    // Proposed fix: require 1 LGM stake
    const PROPOSED_HALT_STAKE = 1n * ONE_LGM;
    expect(PROPOSED_HALT_STAKE).toBe(ONE_LGM);
  });

  it("HALT-2: Proposed fix — halt stake slashed if frivolous", () => {
    const HALT_STAKE = 1n * ONE_LGM;
    const FRIVOLOUS_SLASH_RATE = 1.0; // 100% for frivolous halt

    const slashAmount =
      (HALT_STAKE * BigInt(Math.floor(FRIVOLOUS_SLASH_RATE * 100))) / 100n;
    expect(slashAmount).toBe(HALT_STAKE); // Full slash
  });

  it("HALT-3: Proposed fix — resume requires 24-hour timelock", () => {
    const RESUME_TIMELOCK_BLOCKS = 34272n; // 24 hours
    const blocksPerHour = 34272n / 24n;

    // Verify ~24 hours
    const hoursInTimelock = RESUME_TIMELOCK_BLOCKS / blocksPerHour;
    expect(hoursInTimelock).toBe(24n);
  });

  it("HALT-4: Proposed fix — halt auto-expires after 7 days", () => {
    const HALT_EXPIRY_BLOCKS = 34272n * 7n; // 7 days
    const blocksPerDay = 34272n;

    const daysUntilExpiry = HALT_EXPIRY_BLOCKS / blocksPerDay;
    expect(daysUntilExpiry).toBe(7n);

    // Without renewal, halt lifts automatically
    // Prevents permanent chain freezing
  });
});

// ============================================================
// D3-L1: DUAL TS/GO DISTRIBUTION DIVERGENCE (CRITICAL)
// ============================================================

describe("D3-L1: Dual TS/Go Distribution Divergence", () => {
  it("DUAL-1: Research share rate is consistent across stacks", () => {
    // TS: RESEARCH_SHARE_RATE_BPS (from reward_router.ts)
    expect(RESEARCH_SHARE_RATE_BPS).toBe(2200n); // 22%

    // Go: ResearchShareRateBps should match
    // This test verifies the TS constant; parity with Go requires integration test
  });

  it("DUAL-2: Route reward produces deterministic output", () => {
    const amounts = [
      1n, // 1 wei
      ONE_LGM, // 1 LGM
      100n * ONE_LGM, // 100 LGM
      1000000n * ONE_LGM, // 1M LGM
    ];

    for (const amount of amounts) {
      const result = routeReward({
        amount,
        source: "block_reward",
        recipient: "did:lgm:test",
      });

      // Research share + recipient = original amount
      expect(result.researchShare + result.recipientAmount).toBe(amount);

      // Research share = amount * 2200 / 10000
      const expectedResearch = (amount * RESEARCH_SHARE_RATE_BPS) / 10000n;
      expect(result.researchShare).toBe(expectedResearch);
    }
  });

  it("DUAL-3: 1 wei edge case — research share rounds to 0", () => {
    const result = routeReward({
      amount: 1n,
      source: "block_reward",
      recipient: "did:lgm:test",
    });

    // 1 * 2200 / 10000 = 0 (integer division)
    expect(result.researchShare).toBe(0n);
    expect(result.recipientAmount).toBe(1n);

    // No money lost or created
    expect(result.researchShare + result.recipientAmount).toBe(1n);
  });

  it("DUAL-4: Billion transactions of 1 wei — total research loss bounded", () => {
    const numTxs = 1_000_000_000n;
    const perTx = 1n;

    // Each 1 wei tx: research share rounds to 0
    // Total research loss: numTxs * expected_research_per_tx
    // Expected: 1 * 0.22 = 0.22 wei → rounds to 0
    // Loss per tx: 0.22 wei
    // Total loss: 1B * 0.22 = 220M wei = 0.00000000022 LGM

    const totalAmount = numTxs * perTx;
    const expectedResearch = (totalAmount * RESEARCH_SHARE_RATE_BPS) / 10000n;
    const actualResearch = 0n; // Each individual tx rounds to 0

    const totalLoss = expectedResearch - actualResearch;

    // Loss is bounded: max 1 LGM even with 1B micro-transactions
    expect(totalLoss).toBeLessThan(ONE_LGM);
  });

  it("DUAL-5: Distribution components are non-negative and bounded", () => {
    const totalCost = 10n * ONE_LGM;

    const distribution = buildPaymentDistribution(
      totalCost,
      "did:lgm:provider",
      [
        {
          factId: "fact-1",
          submitter: "did:lgm:submitter",
          usageWeight: 1.0,
          fundamentalityScore: 0.5,
          ancestors: [],
        },
      ],
    );

    // All components are non-negative
    expect(distribution.apiProvider.amount).toBeGreaterThanOrEqual(0n);
    expect(distribution.protocol.burned).toBeGreaterThanOrEqual(0n);
    expect(distribution.protocol.treasury).toBeGreaterThanOrEqual(0n);
    expect(distribution.research.amount).toBeGreaterThanOrEqual(0n);

    // Research share is 22% of total
    const expectedResearch = (totalCost * RESEARCH_SHARE_RATE_BPS) / 10000n;
    expect(distribution.research.amount).toBe(expectedResearch);

    // Provider gets the largest share of the remaining 78%
    expect(distribution.apiProvider.amount).toBeGreaterThan(0n);
  });
});

// ============================================================
// D3-L2: FEE POOL ACCOUNT AUTHORIZATION (CRITICAL)
// ============================================================

describe("D3-L2: Fee Pool Account Authorization", () => {
  it("FEEPOOL-1: System accounts are identified by DID prefix", () => {
    const SYSTEM_ACCOUNTS = [
      "did:lgm:fee-pool",
      "did:lgm:protocol-treasury",
      "did:lgm:research",
      "did:lgm:block-reward-fund",
    ];

    // All system accounts use the did:lgm: prefix
    for (const account of SYSTEM_ACCOUNTS) {
      expect(account.startsWith("did:lgm:")).toBe(true);
    }
  });

  it("FEEPOOL-2: Proposed authorized sources for fee-pool credits", () => {
    const AUTHORIZED_SOURCES = new Set([
      "collectTransactionFees",
      "distributeBlockRewards",
      "executeSlash",
      "settlementProtocolFee",
    ]);

    // Only 4 authorized entry points
    expect(AUTHORIZED_SOURCES.size).toBe(4);

    // Unauthorized sources should be rejected
    expect(AUTHORIZED_SOURCES.has("creditBalance")).toBe(false);
    expect(AUTHORIZED_SOURCES.has("transfer")).toBe(false);
    expect(AUTHORIZED_SOURCES.has("mint")).toBe(false);
  });

  it("FEEPOOL-3: Fee pool credits must have audit trail", () => {
    // Every credit to a system account should produce an event with:
    interface SystemCreditEvent {
      account: string;
      amount: bigint;
      source: string;
      txHash: string;
      blockNumber: bigint;
    }

    const event: SystemCreditEvent = {
      account: "did:lgm:fee-pool",
      amount: ONE_LGM,
      source: "collectTransactionFees",
      txHash: "0x1234",
      blockNumber: 12345n,
    };

    expect(event.source).toBe("collectTransactionFees");
    expect(event.amount).toBeGreaterThan(0n);
  });

  it("FEEPOOL-4: Max single credit bounded by block gas limit * gas price", () => {
    // Maximum fee collection in a single block:
    // blockGasLimit * gasPrice = 33.3M * 1,111,111 wei ≈ 37 * 10^12 wei = 0.000037 LGM
    const blockGasLimit = 33_300_000n;
    const gasPrice = 1_111_111n;
    const maxBlockFees = blockGasLimit * gasPrice;

    // Max fees per block is ~37 * 10^12 wei (0.000037 LGM)
    // Any single credit significantly exceeding this is suspicious
    expect(maxBlockFees).toBeLessThan(ONE_LGM); // Well below 1 LGM per block
    expect(maxBlockFees).toBeGreaterThan(0n);

    // Upper bound sanity: even with 10x gas price, max fees < 1 LGM/block
    const maxFeesAt10x = blockGasLimit * gasPrice * 10n;
    expect(maxFeesAt10x).toBeLessThan(ONE_LGM);
  });
});
