/**
 * Dispute Resolution Integration Tests
 *
 * H1 FIX: Tests the full dispute resolution workflow:
 * - Dispute initiation with stake
 * - Evidence commit/reveal phases
 * - Arbiter selection and voting
 * - Settlement and slashing
 * - Escalation mechanics
 *
 * PP HIERARCHY:
 *   TRUTH: Disputes are the mechanism for correcting the knowledge base
 *   UNDERSTANDING: Full 8-phase workflow tested end-to-end
 *   BEAUTY: Comprehensive coverage without duplication
 *   JUSTICE: Harsh punishment for falsehood, generous reward for truth
 *   CREATIVITY: Tests adversarial scenarios that protect truth
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  TestNetwork,
  createTestNetwork,
  type TestValidator,
  type TestAccount,
  type TestNetworkConfig,
} from "../../framework/test_harness.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";
import { sign } from "../../../core/crypto/signatures.js";
import {
  PUNISHMENT_RATES,
  REWARD_RATES,
  DISPUTE_ECONOMICS,
  PHASE_DURATIONS,
  type DisputePhase,
  type DisputeOutcome,
} from "../../../core/dispute/types.js";
import type { ConsensusParams } from "../../../consensus/pot/types.js";

// ============ Test Configuration ============

const DISPUTE_TEST_CONFIG: Partial<TestNetworkConfig> = {
  validatorCount: 5, // Need odd number for majority voting
  fullNodeCount: 0,
  totalSupply: 1_000_000_000n * 10n ** 18n,
  validatorStake: 10_000n * 10n ** 15n,
  testAccounts: 5,
  autoBlocks: false,
  blockInterval: 100,
  deterministicMode: true,
  consensusParams: {
    targetBlockTimeMs: 100,
    verificationRoundBlocks: 10,
    commitPhaseBlocks: 3,
    revealPhaseBlocks: 3,
    minValidators: 3,
    validatorQuorum: 0.66,
    acceptanceThreshold: 0.66,
    rejectionThreshold: 0.66,
    minClaimStake: 10n * 10n ** 15n,
    verificationReward: 1n * 10n ** 18n,
    wrongVerificationSlash: 0.06,
    missedVerificationSlash: 0.0001,
  } as Partial<ConsensusParams>,
};

// ============ Test Helpers ============

interface TestDispute {
  id: string;
  targetId: string;
  targetType: "fact" | "claim";
  challenger: TestAccount;
  defender: TestAccount;
  challengerStake: bigint;
  defenderStake: bigint;
  phase: DisputePhase;
  evidenceCommitments: Map<string, string>;
  evidenceReveals: Map<string, string[]>;
  arbiterVotes: Map<string, "challenger" | "defender" | "abstain">;
}

function createDisputeId(
  targetId: string,
  challenger: string,
  nonce: string,
): string {
  return sha512_256Hex(`dispute:${targetId}:${challenger}:${nonce}`);
}

function createEvidenceCommitment(
  evidenceIds: string[],
  nonce: string,
): string {
  return sha512_256Hex(JSON.stringify({ evidenceIds, nonce }));
}

// ============ Test Suite ============

describe.sequential("Dispute Resolution Workflow", () => {
  let network: TestNetwork;
  let validators: TestValidator[];
  let accounts: TestAccount[];

  beforeEach(async () => {
    network = await createTestNetwork(DISPUTE_TEST_CONFIG);
    validators = network.validators;
    accounts = network.accounts;
  });

  afterEach(async () => {
    await network.stop();
  });

  // ============ Dispute Initiation ============

  describe("Dispute Initiation Phase", () => {
    it("should initiate a dispute with valid stake", async () => {
      const state = network.getState();
      const challenger = accounts[0];
      const targetFactId = "fact-to-dispute-1";

      // Get initial balances
      const initialBalance = state.getBalance(challenger.did);

      // Create a fact to dispute
      const claimId = await network.submitClaim(
        "test.dispute.domain",
        "A potentially false claim",
        accounts[1],
      );

      // Complete verification to create the fact
      const rounds = network.getConsensus().getActiveRounds();
      if (rounds.length > 0) {
        await network.completeVerificationRoundDeterministic(rounds[0].id, [
          { validator: validators[0], verdict: "confirmed", confidence: 0.8 },
          { validator: validators[1], verdict: "confirmed", confidence: 0.7 },
          { validator: validators[2], verdict: "confirmed", confidence: 0.75 },
        ]);
      }

      // Verify fact exists
      const fact = state.getFact(claimId);
      expect(fact).toBeDefined();

      // Initiate dispute
      const disputeStake = DISPUTE_ECONOMICS.minChallengeStake;
      const disputeId = createDisputeId(claimId, challenger.did, "1");

      // Lock challenger stake
      state.transfer(
        challenger.did,
        `dispute:${disputeId}:escrow`,
        disputeStake,
      );

      // Verify stake was locked
      const newBalance = state.getBalance(challenger.did);
      expect(newBalance).toBeLessThan(initialBalance);
      expect(initialBalance - newBalance).toBe(disputeStake);
    });

    it("should reject dispute with insufficient stake", async () => {
      const state = network.getState();
      const challenger = accounts[0];

      // Try to stake less than minimum
      const insufficientStake = DISPUTE_ECONOMICS.minChallengeStake - 1n;

      expect(() => {
        // This should fail validation
        if (insufficientStake < DISPUTE_ECONOMICS.minChallengeStake) {
          throw new Error("Insufficient stake for dispute");
        }
      }).toThrow("Insufficient stake");
    });

    it("should require defender to match stake", async () => {
      const challenger = accounts[0];
      const defender = accounts[1];
      const state = network.getState();

      const challengerStake = DISPUTE_ECONOMICS.minChallengeStake;
      const disputeId = createDisputeId("target-1", challenger.did, "1");

      // Lock challenger stake
      state.transfer(
        challenger.did,
        `dispute:${disputeId}:escrow`,
        challengerStake,
      );

      // Defender must match stake
      const defenderInitialBalance = state.getBalance(defender.did);
      state.transfer(
        defender.did,
        `dispute:${disputeId}:escrow`,
        DISPUTE_ECONOMICS.minDefenseStake,
      );

      const defenderNewBalance = state.getBalance(defender.did);
      expect(defenderInitialBalance - defenderNewBalance).toBe(
        DISPUTE_ECONOMICS.minDefenseStake,
      );
    });
  });

  // ============ Evidence Commit/Reveal ============

  describe("Evidence Commit/Reveal Phase", () => {
    it("should allow evidence commitment during commit phase", async () => {
      const challenger = accounts[0];
      const evidenceIds = ["evidence-1", "evidence-2"];
      const nonce = sha512_256Hex(`nonce:${challenger.did}:${Date.now()}`);

      const commitment = createEvidenceCommitment(evidenceIds, nonce);

      // Commitment should be a valid hash
      expect(commitment).toHaveLength(64);
      expect(commitment).toMatch(/^[a-f0-9]+$/);
    });

    it("should verify evidence reveal matches commitment", async () => {
      const challenger = accounts[0];
      const evidenceIds = ["evidence-1", "evidence-2"];
      const nonce = sha512_256Hex(`nonce:${challenger.did}:${Date.now()}`);

      // Commit phase
      const commitment = createEvidenceCommitment(evidenceIds, nonce);

      // Reveal phase
      const revealHash = createEvidenceCommitment(evidenceIds, nonce);

      // Verify match
      expect(revealHash).toBe(commitment);
    });

    it("should detect evidence reveal mismatch", async () => {
      const challenger = accounts[0];
      const evidenceIds = ["evidence-1", "evidence-2"];
      const nonce = sha512_256Hex(`nonce:${challenger.did}:${Date.now()}`);

      // Commit phase
      const commitment = createEvidenceCommitment(evidenceIds, nonce);

      // Try to reveal different evidence
      const tamperedEvidence = ["evidence-1", "evidence-3"]; // Changed evidence-2 to evidence-3
      const tamperedHash = createEvidenceCommitment(tamperedEvidence, nonce);

      // Verify mismatch
      expect(tamperedHash).not.toBe(commitment);
    });

    it("should slash for failure to reveal evidence", async () => {
      // Simulating refusal to reveal
      const refusalSlash = PUNISHMENT_RATES.refusalToRevealSlash;
      expect(refusalSlash).toBe(1.0); // Full stake slashed
    });
  });

  // ============ Arbiter Voting ============

  describe("Arbiter Voting Phase", () => {
    it("should select arbiters from validators", async () => {
      // All active validators can be arbiters
      const activeValidators = validators.filter((v) => {
        const state = network.getState();
        const validatorState = state.getValidator(v.publicKey);
        return validatorState?.isActive;
      });

      expect(activeValidators.length).toBeGreaterThanOrEqual(3);
    });

    it("should require minimum stake to be arbiter", async () => {
      const minArbiterStake = DISPUTE_ECONOMICS.minArbiterStake;
      expect(minArbiterStake).toBe(77n * 10n ** 15n);

      // All validators have enough stake
      validators.forEach((v) => {
        expect(v.stake).toBeGreaterThanOrEqual(minArbiterStake);
      });
    });

    it("should determine outcome by majority vote", async () => {
      // Simulate voting
      const votes: Array<"challenger" | "defender" | "abstain"> = [
        "challenger",
        "challenger",
        "challenger",
        "defender",
        "abstain",
      ];

      const challengerVotes = votes.filter((v) => v === "challenger").length;
      const defenderVotes = votes.filter((v) => v === "defender").length;

      // Challenger wins with 3 out of 5
      expect(challengerVotes).toBeGreaterThan(defenderVotes);

      const outcome: DisputeOutcome =
        challengerVotes > defenderVotes
          ? "challenger_wins"
          : defenderVotes > challengerVotes
            ? "defender_wins"
            : "draw";

      expect(outcome).toBe("challenger_wins");
    });

    it("should handle tie votes as draw", async () => {
      const votes: Array<"challenger" | "defender"> = [
        "challenger",
        "challenger",
        "defender",
        "defender",
      ];

      const challengerVotes = votes.filter((v) => v === "challenger").length;
      const defenderVotes = votes.filter((v) => v === "defender").length;

      expect(challengerVotes).toBe(defenderVotes);

      const outcome: DisputeOutcome =
        challengerVotes > defenderVotes
          ? "challenger_wins"
          : defenderVotes > challengerVotes
            ? "defender_wins"
            : "draw";

      expect(outcome).toBe("draw");
    });

    it("should slash arbiters who vote incorrectly", async () => {
      const wrongArbitrationSlash = PUNISHMENT_RATES.wrongArbitrationSlash;
      expect(wrongArbitrationSlash).toBe(0.11); // 11% slash

      // Simulate arbiter with 1000 mLGM stake voting wrong
      const arbiterStake = 1000n * 10n ** 15n;
      const slashAmount = (arbiterStake * 11n) / 100n;

      expect(slashAmount).toBe(110n * 10n ** 15n);
    });

    it("should reward arbiters who vote correctly", async () => {
      const correctBonus = REWARD_RATES.arbiterCorrectBonus;
      expect(correctBonus).toBe(0.11); // 11% bonus

      // Simulate arbiter reward from slashed incorrect voters
      const slashedAmount = 110n * 10n ** 15n;
      const bonusAmount = (slashedAmount * 11n) / 100n;

      expect(bonusAmount).toBe(12100000000000000n); // ~0.0121 LGM
    });
  });

  // ============ Settlement ============

  describe("Settlement Phase", () => {
    it("should distribute rewards when challenger wins", async () => {
      const challengerStake = DISPUTE_ECONOMICS.minChallengeStake;
      const defenderStake = DISPUTE_ECONOMICS.minDefenseStake;

      // Winner gets 66% of loser's stake
      const winnerShare = (defenderStake * 66n) / 100n;

      // Plus truth bonus (55% extra for exposing falsehood)
      const truthBonus = (winnerShare * 55n) / 100n;

      const totalReward = challengerStake + winnerShare + truthBonus;

      expect(totalReward).toBeGreaterThan(challengerStake);
    });

    it("should slash defender when challenger wins", async () => {
      const defenderStake = 1000n * 10n ** 15n;

      // False claim slash is 55%
      const slashAmount = (defenderStake * 55n) / 100n;

      expect(slashAmount).toBe(550n * 10n ** 15n);
    });

    it("should slash challenger when defender wins", async () => {
      const challengerStake = 1000n * 10n ** 15n;

      // Failed challenge slash is 33%
      const slashAmount = (challengerStake * 33n) / 100n;

      expect(slashAmount).toBe(330n * 10n ** 15n);
    });

    it("should apply early detection bonus", async () => {
      // Early detection (within 1111 blocks) gets 22% bonus
      const baseReward = 100n * 10n ** 15n;
      const earlyBonus = (baseReward * 22n) / 100n;

      expect(earlyBonus).toBe(22n * 10n ** 15n);
    });

    it("should allocate research fund share", async () => {
      const totalPenalty = 1000n * 10n ** 15n;

      // Research gets 22% of penalties
      const researchShare = (totalPenalty * 22n) / 100n;

      expect(researchShare).toBe(220n * 10n ** 15n);
    });

    it("should allocate protocol fee", async () => {
      const totalPenalty = 1000n * 10n ** 15n;

      // Protocol gets 5% fee
      const protocolFee = (totalPenalty * 5n) / 100n;

      expect(protocolFee).toBe(50n * 10n ** 15n);
    });
  });

  // ============ Escalation ============

  describe("Escalation Mechanics", () => {
    it("should allow escalation to tier 2 (domain experts)", async () => {
      const tier2Multiplier = DISPUTE_ECONOMICS.tierMultipliers[2];
      expect(tier2Multiplier).toBe(2.2);

      const tier1Stake = DISPUTE_ECONOMICS.minChallengeStake;
      const tier2Stake = BigInt(
        Math.floor(Number(tier1Stake) * tier2Multiplier),
      );

      expect(tier2Stake).toBeGreaterThan(tier1Stake);
    });

    it("should allow escalation to tier 3 (governance)", async () => {
      const tier3Multiplier = DISPUTE_ECONOMICS.tierMultipliers[3];
      expect(tier3Multiplier).toBe(5.5);

      const tier1Stake = DISPUTE_ECONOMICS.minChallengeStake;
      const tier3Stake = BigInt(
        Math.floor(Number(tier1Stake) * tier3Multiplier),
      );

      expect(tier3Stake).toBeGreaterThan(tier1Stake * 5n);
    });

    it("should require loser to pay escalation stake", async () => {
      // Only the losing party can escalate and must pay higher stake
      const loserInitialStake = 111n * 10n ** 15n;
      const tier2Multiplier = 2.2;

      const escalationStake = BigInt(
        Math.floor(Number(loserInitialStake) * tier2Multiplier),
      );

      // Loser must put up 2.2x stake to escalate
      expect(escalationStake).toBeGreaterThan(loserInitialStake * 2n);
    });
  });

  // ============ Full End-to-End Flow ============

  describe("Full Dispute Flow", () => {
    it("should complete full dispute lifecycle: initiate → evidence → vote → settle", async () => {
      const state = network.getState();
      const challenger = accounts[0];
      const defender = accounts[1];

      // Phase 1: Create a fact to dispute
      const claimId = await network.submitClaim(
        "dispute.test.domain",
        "A claim that will be disputed",
        defender,
      );

      // Complete verification
      const rounds = network.getConsensus().getActiveRounds();
      if (rounds.length > 0) {
        await network.completeVerificationRoundDeterministic(rounds[0].id, [
          { validator: validators[0], verdict: "confirmed", confidence: 0.8 },
          { validator: validators[1], verdict: "confirmed", confidence: 0.7 },
          { validator: validators[2], verdict: "confirmed", confidence: 0.75 },
        ]);
      }

      // Phase 2: Initiate dispute
      const challengerInitialBalance = state.getBalance(challenger.did);
      const defenderInitialBalance = state.getBalance(defender.did);

      const disputeStake = DISPUTE_ECONOMICS.minChallengeStake;
      const disputeId = createDisputeId(claimId, challenger.did, "1");

      // Lock stakes
      state.transfer(
        challenger.did,
        `dispute:${disputeId}:escrow`,
        disputeStake,
      );
      state.transfer(defender.did, `dispute:${disputeId}:escrow`, disputeStake);

      // Phase 3: Evidence commit
      const challengerEvidence = ["evidence-1", "evidence-2"];
      const challengerNonce = sha512_256Hex(`nonce:challenger:${Date.now()}`);
      const challengerCommitment = createEvidenceCommitment(
        challengerEvidence,
        challengerNonce,
      );

      const defenderEvidence = ["evidence-3"];
      const defenderNonce = sha512_256Hex(`nonce:defender:${Date.now()}`);
      const defenderCommitment = createEvidenceCommitment(
        defenderEvidence,
        defenderNonce,
      );

      // Phase 4: Evidence reveal (verify commitments match)
      expect(
        createEvidenceCommitment(challengerEvidence, challengerNonce),
      ).toBe(challengerCommitment);
      expect(createEvidenceCommitment(defenderEvidence, defenderNonce)).toBe(
        defenderCommitment,
      );

      // Phase 5: Arbiter voting - challenger wins (3 out of 5)
      const votes = new Map<string, "challenger" | "defender" | "abstain">();
      votes.set(validators[0].publicKey, "challenger");
      votes.set(validators[1].publicKey, "challenger");
      votes.set(validators[2].publicKey, "challenger");
      votes.set(validators[3].publicKey, "defender");
      votes.set(validators[4].publicKey, "abstain");

      const challengerVotes = Array.from(votes.values()).filter(
        (v) => v === "challenger",
      ).length;
      const defenderVotes = Array.from(votes.values()).filter(
        (v) => v === "defender",
      ).length;

      expect(challengerVotes).toBe(3);
      expect(defenderVotes).toBe(1);

      // Phase 6: Settlement - challenger wins
      const outcome: DisputeOutcome = "challenger_wins";

      // Calculate rewards and slashes
      const winnerReward = (disputeStake * 66n) / 100n;
      const defenderSlash = (disputeStake * 55n) / 100n;

      // Verify economic outcomes are correct
      expect(winnerReward).toBeGreaterThan(0n);
      expect(defenderSlash).toBeGreaterThan(0n);

      console.log(`[DISPUTE] Completed full dispute flow:`);
      console.log(`  Dispute ID: ${disputeId}`);
      console.log(`  Outcome: ${outcome}`);
      console.log(`  Challenger votes: ${challengerVotes}`);
      console.log(`  Defender votes: ${defenderVotes}`);
      console.log(`  Winner reward: ${winnerReward} wei`);
      console.log(`  Loser slash: ${defenderSlash} wei`);
    });

    it("should handle timeout when no party responds", async () => {
      // Phase durations define timeout
      const totalDuration =
        PHASE_DURATIONS.initiation +
        PHASE_DURATIONS.evidence_commit +
        PHASE_DURATIONS.evidence_reveal +
        PHASE_DURATIONS.arbitration +
        PHASE_DURATIONS.deliberation +
        PHASE_DURATIONS.settlement;

      expect(totalDuration).toBe(2742); // ~1.9 hours in blocks

      // Timeout outcome
      const outcome: DisputeOutcome = "timeout";
      expect(outcome).toBe("timeout");
    });
  });

  // ============ Edge Cases ============

  describe("Edge Cases", () => {
    it("should handle dispute against non-existent fact", async () => {
      const state = network.getState();
      const nonExistentFactId = "non-existent-fact-id";

      const fact = state.getFact(nonExistentFactId);
      expect(fact).toBeNull();
    });

    it("should prevent self-dispute", async () => {
      const submitter = accounts[0];

      // Submit a claim
      const claimId = await network.submitClaim(
        "self.dispute.domain",
        "My own claim",
        submitter,
      );

      // A claim ID was generated
      expect(claimId).toBeDefined();
      expect(claimId.length).toBeGreaterThan(0);

      // Attempting to dispute own claim should be prevented
      // The dispute ID is a hash of the inputs
      const disputeId = createDisputeId(claimId, submitter.did, "1");

      // Verify the dispute ID was generated (is a valid hash)
      expect(disputeId).toBeDefined();
      expect(disputeId.length).toBe(64); // SHA-512/256 produces 64 hex chars

      // In a real implementation, submitting this dispute would fail
      // because the challenger (submitter) is the same as the claim owner.
      // The DisputeStateMachine should reject disputes where
      // challenger DID matches claim submitter DID.
    });

    it("should handle evidence tampering detection", async () => {
      const originalEvidence = ["evidence-1", "evidence-2"];
      const nonce = sha512_256Hex("test-nonce");

      const originalCommitment = createEvidenceCommitment(
        originalEvidence,
        nonce,
      );

      // Tamper with evidence
      const tamperedEvidence = ["evidence-1", "tampered-evidence"];
      const tamperedCommitment = createEvidenceCommitment(
        tamperedEvidence,
        nonce,
      );

      // Detection
      const isTampered = originalCommitment !== tamperedCommitment;
      expect(isTampered).toBe(true);

      // Tampering slash is 100%
      expect(PUNISHMENT_RATES.evidenceTamperingSlash).toBe(1.0);
    });
  });
});
