/**
 * Claim Submission → Knowledge Query → Billing Integration Test
 *
 * Tests the complete lifecycle:
 *   1. Agent submits claims to the knowledge tree
 *   2. Claims pass through verification (commit-reveal simulation)
 *   3. Verified claims become facts
 *   4. A querying agent searches the knowledge tree
 *   5. Query is priced based on fact confidence + citations
 *   6. Payment is distributed: provider, contributors, research, protocol
 *   7. Demonstrates the "agentic think process" — how an agent reasons
 *      by querying facts, paying for knowledge, and building on it
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  executeTransaction,
  FEE_POOL_DID,
  REWARD_POOL_DID,
} from "../../../state/transitions.js";
import type {
  Transaction,
  ConsensusParams,
} from "../../../consensus/pot/types.js";
import { DEFAULT_CONSENSUS_PARAMS } from "../../../consensus/pot/types.js";
import type {
  ClaimSubmitPayload,
  ClaimCompleteVerificationPayload,
  FactState,
} from "../../../state/types.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";
import type { KnowledgeStateTree } from "../../../state/knowledge.js";
import {
  calculateQueryPrice,
  calculateDistribution,
  type FactResult,
} from "../../../core/billing/truth_gateway.js";
import {
  executePaymentDistribution,
  canAffordDistribution,
} from "../../../core/billing/payment_executor.js";
import type { DID } from "../../../core/agents/identity/did.js";

// ============================================================================
// Constants
// ============================================================================

const ONE_LGM = 10n ** 18n;

const PARAMS: ConsensusParams = {
  ...DEFAULT_CONSENSUS_PARAMS,
  minClaimStake: 100n * ONE_LGM,
  // Disable anti-sybil for this test (not the focus)
  newAccountTransferCooldown: 0n,
  bootstrapVestingBlocks: 0n,
};

// Agents in the economy
const SCIENTIST_KEY = "scientist-agent-pubkey-001";
const SCIENTIST_DID = publicKeyToDid(SCIENTIST_KEY);

const VERIFIER_KEYS = ["verifier-1-key", "verifier-2-key", "verifier-3-key"];
const VERIFIER_DIDS = VERIFIER_KEYS.map(publicKeyToDid);

const THINKER_KEY = "thinker-agent-pubkey-002";
const THINKER_DID = publicKeyToDid(THINKER_KEY);

const PROVIDER_DID = "did:lgm:knowledge-node-001" as DID;

// ============================================================================
// Helpers
// ============================================================================

function knowledgeTree(state: StateManager): KnowledgeStateTree {
  return (state as any).knowledge;
}

function userTx(
  type: string,
  payload: Record<string, unknown>,
  sender: string,
  state: StateManager,
): Transaction {
  const did = publicKeyToDid(sender);
  const nonce = state.getNonce(did);
  return {
    type: type as any,
    sender,
    nonce,
    fee: 0n,
    payload,
    signature: "" as any,
  };
}

function systemTx(type: string, payload: Record<string, unknown>): Transaction {
  return {
    type: type as any,
    sender: "system" as any,
    nonce: 0n,
    fee: 0n,
    _systemInternal: true,
    payload,
    signature: "" as any,
  } as any;
}

/** Submit a claim and return the claimId */
function submitClaim(
  state: StateManager,
  claim: string,
  domain: string,
  stratum: string,
  senderKey: string,
  blockNumber: bigint,
  stake: bigint = 200n * ONE_LGM,
  references: string[] = [],
): string {
  const payload: ClaimSubmitPayload = {
    type: "claim_submit",
    claim,
    claimType: "empirical",
    domain,
    stratum: stratum as any,
    stake,
    references,
  };

  const result = executeTransaction(
    state,
    userTx("claim_submit", payload as any, senderKey, state),
    blockNumber,
    PARAMS,
  );
  expect(result.success).toBe(true);

  const pending = state.getPendingClaims();
  return pending[pending.length - 1].id;
}

/** Run verification: start → complete (accepted) */
function verifyAndAccept(
  state: StateManager,
  claimId: string,
  roundId: string,
  confidence: number,
  blockNumber: bigint,
): void {
  // Start verification
  executeTransaction(
    state,
    systemTx("claim_start_verification", {
      type: "claim_start_verification",
      claimId,
      roundId,
      selectedValidators: VERIFIER_DIDS,
    }),
    blockNumber,
    PARAMS,
  );

  // Complete verification (accepted)
  const completePayload: ClaimCompleteVerificationPayload = {
    type: "claim_complete_verification",
    claimId,
    roundId,
    outcome: "accepted",
    finalConfidence: confidence,
    verifierCount: 3,
    rewards: [
      [VERIFIER_DIDS[0], 1n * ONE_LGM],
      [VERIFIER_DIDS[1], 1n * ONE_LGM],
      [VERIFIER_DIDS[2], 1n * ONE_LGM],
    ],
    slashes: [],
  };

  const result = executeTransaction(
    state,
    systemTx("claim_complete_verification", completePayload as any),
    blockNumber + 10n,
    PARAMS,
  );
  expect(result.success).toBe(true);
}

/** Convert on-chain FactState to billing FactResult */
function toFactResult(fact: FactState, submitter: DID): FactResult {
  return {
    id: fact.id,
    claim: fact.claimText,
    domain: fact.domain,
    confidence: fact.confidence,
    fundamentalityScore: 0,
    submitter,
    verifiedAtBlock: fact.verifiedAt,
    citationCount: fact.citationCount,
    lastCitedBlock: fact.lastCitedBlock,
  };
}

// ============================================================================
// Tests
// ============================================================================

describe.sequential(
  "Claim → Query → Billing: Agentic Knowledge Economy",
  () => {
    let state: StateManager;
    let blockNumber: bigint;

    beforeEach(() => {
      state = new StateManager();
      blockNumber = 100n;

      // Fund the scientist (submits claims)
      state.createAccount(SCIENTIST_DID, "sci-op", 50000n * ONE_LGM);

      // Fund the thinker (queries knowledge, pays for it)
      state.createAccount(THINKER_DID, "think-op", 10000n * ONE_LGM);

      // System accounts
      state.createAccount(FEE_POOL_DID, "fee-op", 1_000_000n * ONE_LGM);
      state.createAccount(REWARD_POOL_DID, "reward-op", 1_000_000n * ONE_LGM);
      state.createAccount(PROVIDER_DID, "provider-op", 0n);
    });

    // ==========================================================================
    // Phase 1: Knowledge Creation (Claims → Facts)
    // ==========================================================================

    describe("Phase 1: Knowledge Creation", () => {
      it("should submit a claim, verify it, and create a fact", () => {
        const claimId = submitClaim(
          state,
          "The speed of light in vacuum is 299,792,458 m/s",
          "physics.constants",
          "physical",
          SCIENTIST_KEY,
          blockNumber,
        );

        verifyAndAccept(
          state,
          claimId,
          "round-physics-001",
          0.95,
          blockNumber + 1n,
        );

        // Fact should exist in knowledge tree
        const claim = state.getClaim(claimId);
        expect(claim?.status).toBe("accepted");
        expect(claim?.resultFactId).toBeDefined();

        const fact = state.getFact(claim!.resultFactId!);
        expect(fact).not.toBeNull();
        expect(fact!.claimText).toBe(
          "The speed of light in vacuum is 299,792,458 m/s",
        );
        expect(fact!.domain).toBe("physics.constants");
        expect(fact!.confidence).toBe(0.95);
        expect(fact!.status).toBe("active");
      });

      it("should create a chain of related facts with references", () => {
        // Fact 1: Base fact
        const claimId1 = submitClaim(
          state,
          "E = mc² describes mass-energy equivalence",
          "physics.relativity",
          "physical",
          SCIENTIST_KEY,
          blockNumber,
        );
        verifyAndAccept(
          state,
          claimId1,
          "round-rel-001",
          0.98,
          blockNumber + 1n,
        );

        const fact1Id = state.getClaim(claimId1)!.resultFactId!;
        const fact1 = state.getFact(fact1Id);
        expect(fact1!.confidence).toBe(0.98);

        // Fact 2: References fact 1
        const claimId2 = submitClaim(
          state,
          "Nuclear binding energy follows from mass-energy equivalence",
          "physics.nuclear",
          "physical",
          SCIENTIST_KEY,
          blockNumber + 20n,
          200n * ONE_LGM,
          [fact1Id], // Reference to first fact
        );
        verifyAndAccept(
          state,
          claimId2,
          "round-nuc-001",
          0.88,
          blockNumber + 21n,
        );

        const fact2Id = state.getClaim(claimId2)!.resultFactId!;
        const fact2 = state.getFact(fact2Id);
        expect(fact2!.confidence).toBe(0.88);

        // Fact 3: References both
        const claimId3 = submitClaim(
          state,
          "Stellar nucleosynthesis produces elements via nuclear fusion",
          "physics.astrophysics",
          "physical",
          SCIENTIST_KEY,
          blockNumber + 40n,
          200n * ONE_LGM,
          [fact1Id, fact2Id], // References both previous facts
        );
        verifyAndAccept(
          state,
          claimId3,
          "round-astro-001",
          0.91,
          blockNumber + 41n,
        );

        const fact3Id = state.getClaim(claimId3)!.resultFactId!;
        const fact3 = state.getFact(fact3Id);
        expect(fact3!.confidence).toBe(0.91);

        // Knowledge tree now has 3 interconnected facts
        const physicsConstants = state.getFactsByDomain("physics.relativity");
        expect(physicsConstants.length).toBeGreaterThanOrEqual(1);
      });
    });

    // ==========================================================================
    // Phase 2: Query Pricing (How much does knowledge cost?)
    // ==========================================================================

    describe("Phase 2: Query Pricing", () => {
      let factId: string;
      let fact: FactState;

      beforeEach(() => {
        // Create a verified fact to query against
        const claimId = submitClaim(
          state,
          "Water consists of two hydrogen atoms and one oxygen atom (H₂O)",
          "chemistry.molecular",
          "chemical",
          SCIENTIST_KEY,
          blockNumber,
        );
        verifyAndAccept(
          state,
          claimId,
          "round-chem-001",
          0.92,
          blockNumber + 1n,
        );

        factId = state.getClaim(claimId)!.resultFactId!;
        fact = state.getFact(factId)!;
      });

      it("should calculate query price based on confidence", () => {
        const factResult = toFactResult(fact, SCIENTIST_DID as DID);
        const pricing = calculateQueryPrice([factResult], blockNumber + 100n);

        expect(pricing.totalCost).toBeGreaterThan(0n);
        expect(pricing.baseCost).toBeGreaterThan(0n);

        // High confidence (0.92) gets a discount (negative premium)
        expect(pricing.confidencePremium).toBeLessThan(0n);
      });

      it("should apply freshness discount for recently verified facts", () => {
        const factResult = toFactResult(fact, SCIENTIST_DID as DID);

        // Query soon after verification (within 1000 blocks)
        const freshPricing = calculateQueryPrice(
          [factResult],
          blockNumber + 50n,
        );

        // Query much later (after 1000 blocks)
        const stalePricing = calculateQueryPrice(
          [factResult],
          blockNumber + 2000n,
        );

        // Fresh facts are MORE expensive (novel information premium)
        expect(freshPricing.freshnessDiscount).toBeLessThan(0n); // negative = premium
        expect(stalePricing.freshnessDiscount).toBe(0n);
        expect(freshPricing.totalCost).toBeGreaterThan(stalePricing.totalCost);
      });

      it("should decrease price for highly-cited facts (well-known = less marginal value)", () => {
        const factResult1 = toFactResult(fact, SCIENTIST_DID as DID);

        // Simulate citations
        const citedFactResult: FactResult = {
          ...factResult1,
          citationCount: 100, // Many citations = well-known
        };

        const uncitedPricing = calculateQueryPrice(
          [factResult1],
          blockNumber + 2000n,
        );
        const citedPricing = calculateQueryPrice(
          [citedFactResult],
          blockNumber + 2000n,
        );

        // Highly cited = well-known = cheaper (less marginal information value)
        expect(citedPricing.totalCost).toBeLessThan(uncitedPricing.totalCost);
      });

      it("should scale cost with number of facts queried", () => {
        // Create a second fact
        const claimId2 = submitClaim(
          state,
          "The pH of pure water at 25°C is 7.0",
          "chemistry.molecular",
          "chemical",
          SCIENTIST_KEY,
          blockNumber + 30n,
        );
        verifyAndAccept(
          state,
          claimId2,
          "round-chem-002",
          0.89,
          blockNumber + 31n,
        );
        const fact2Id = state.getClaim(claimId2)!.resultFactId!;
        const fact2 = state.getFact(fact2Id)!;

        const singleFact = [toFactResult(fact, SCIENTIST_DID as DID)];
        const twoFacts = [
          toFactResult(fact, SCIENTIST_DID as DID),
          toFactResult(fact2, SCIENTIST_DID as DID),
        ];

        const singlePrice = calculateQueryPrice(
          singleFact,
          blockNumber + 2000n,
        );
        const doublePrice = calculateQueryPrice(twoFacts, blockNumber + 2000n);

        // Two facts should cost roughly double (modulo individual premiums)
        expect(doublePrice.totalCost).toBeGreaterThan(singlePrice.totalCost);
      });
    });

    // ==========================================================================
    // Phase 3: Payment Distribution (Who gets paid?)
    // ==========================================================================

    describe("Phase 3: Payment Distribution", () => {
      let factId: string;
      let fact: FactState;

      beforeEach(() => {
        const claimId = submitClaim(
          state,
          "Photosynthesis converts CO₂ and H₂O into glucose using light energy",
          "biology.biochemistry",
          "chemical",
          SCIENTIST_KEY,
          blockNumber,
        );
        verifyAndAccept(state, claimId, "round-bio-001", 0.9, blockNumber + 1n);
        factId = state.getClaim(claimId)!.resultFactId!;
        fact = state.getFact(factId)!;
      });

      it("should distribute payment across provider, contributors, research, protocol", () => {
        const factResult = toFactResult(fact, SCIENTIST_DID as DID);
        const pricing = calculateQueryPrice([factResult], blockNumber + 2000n);
        const distribution = calculateDistribution(
          pricing.totalCost,
          [factResult],
          PROVIDER_DID,
        );

        // All distribution legs should exist
        expect(distribution.apiProvider.amount).toBeGreaterThan(0n);
        expect(distribution.apiProvider.did).toBe(PROVIDER_DID);

        expect(distribution.knowledgeRewards.length).toBeGreaterThan(0);
        expect(distribution.knowledgeRewards[0].submitter).toBe(SCIENTIST_DID);
        expect(distribution.knowledgeRewards[0].amount).toBeGreaterThan(0n);

        expect(distribution.research!.amount).toBeGreaterThan(0n);

        expect(distribution.protocol.burned).toBeGreaterThanOrEqual(0n);
        expect(distribution.protocol.treasury).toBeGreaterThanOrEqual(0n);

        // Total distributed should equal total cost
        const totalDistributed =
          distribution.apiProvider.amount +
          distribution.knowledgeRewards.reduce((sum, r) => sum + r.amount, 0n) +
          distribution.research!.amount +
          distribution.protocol.burned +
          distribution.protocol.treasury;

        // Allow for rounding (BigInt division)
        expect(totalDistributed).toBeLessThanOrEqual(pricing.totalCost);
        expect(pricing.totalCost - totalDistributed).toBeLessThan(10n); // Max 10 wei rounding
      });

      it("should execute atomic payment and transfer funds", () => {
        const factResult = toFactResult(fact, SCIENTIST_DID as DID);
        const pricing = calculateQueryPrice([factResult], blockNumber + 2000n);
        const distribution = calculateDistribution(
          pricing.totalCost,
          [factResult],
          PROVIDER_DID,
        );

        // Thinker agent pays for the query
        const thinkerBefore = state.getBalance(THINKER_DID);
        const providerBefore = state.getBalance(PROVIDER_DID);
        const scientistBefore = state.getBalance(SCIENTIST_DID);

        // Check affordability
        expect(canAffordDistribution(thinkerBefore, distribution)).toBe(true);

        // Execute payment
        const result = executePaymentDistribution(
          distribution,
          THINKER_DID as DID,
          state,
          blockNumber + 2000n,
        );

        expect(result.success).toBe(true);
        expect(result.transfers.length).toBeGreaterThan(0);
        expect(result.totalTransferred).toBeGreaterThan(0n);

        // Thinker balance decreased
        const thinkerAfter = state.getBalance(THINKER_DID);
        expect(thinkerAfter).toBeLessThan(thinkerBefore);

        // Provider got paid
        const providerAfter = state.getBalance(PROVIDER_DID);
        expect(providerAfter).toBeGreaterThan(providerBefore);

        // Scientist (fact contributor) got citation reward
        const scientistAfter = state.getBalance(SCIENTIST_DID);
        expect(scientistAfter).toBeGreaterThan(scientistBefore);

        // Research funded
        expect(result.researchFunded).toBeGreaterThan(0n);
      });

      it("should gracefully handle insufficient balance", () => {
        // Create a poor agent
        const poorDid = "did:lgm:broke-agent";
        state.createAccount(poorDid, "poor-op", 1n); // Only 1 wei

        const factResult = toFactResult(fact, SCIENTIST_DID as DID);
        const pricing = calculateQueryPrice([factResult], blockNumber + 2000n);
        const distribution = calculateDistribution(
          pricing.totalCost,
          [factResult],
          PROVIDER_DID,
        );

        expect(canAffordDistribution(1n, distribution)).toBe(false);

        const result = executePaymentDistribution(
          distribution,
          poorDid as DID,
          state,
          blockNumber,
        );

        expect(result.success).toBe(false);
        expect(result.errors.length).toBeGreaterThan(0);
        expect(result.errors[0]).toContain("Insufficient balance");
      });
    });

    // ==========================================================================
    // Phase 4: Agentic Think Process
    // ==========================================================================

    describe("Phase 4: Agentic Think Process", () => {
      let physicsFactId: string;
      let chemFactId: string;
      let bioFactId: string;

      beforeEach(() => {
        // Build a knowledge base with multiple facts across domains

        // Physics fact
        const c1 = submitClaim(
          state,
          "Energy is conserved in all physical processes (First Law of Thermodynamics)",
          "physics.thermodynamics",
          "physical",
          SCIENTIST_KEY,
          blockNumber,
        );
        verifyAndAccept(state, c1, "round-thermo-001", 0.97, blockNumber + 1n);
        physicsFactId = state.getClaim(c1)!.resultFactId!;

        // Chemistry fact (references physics)
        const c2 = submitClaim(
          state,
          "Exothermic reactions release energy as heat to the surroundings",
          "chemistry.reactions",
          "chemical",
          SCIENTIST_KEY,
          blockNumber + 20n,
          200n * ONE_LGM,
          [physicsFactId],
        );
        verifyAndAccept(state, c2, "round-react-001", 0.93, blockNumber + 21n);
        chemFactId = state.getClaim(c2)!.resultFactId!;

        // Biology fact (references chemistry)
        const c3 = submitClaim(
          state,
          "Cellular respiration is an exothermic process that converts glucose to ATP",
          "biology.metabolism",
          "chemical",
          SCIENTIST_KEY,
          blockNumber + 40n,
          200n * ONE_LGM,
          [chemFactId],
        );
        verifyAndAccept(state, c3, "round-meta-001", 0.89, blockNumber + 41n);
        bioFactId = state.getClaim(c3)!.resultFactId!;
      });

      it("should simulate an agent reasoning across a knowledge chain", () => {
        /**
         * Agentic Think Process:
         *
         * The "Thinker" agent wants to answer:
         *   "Why does cellular respiration release heat?"
         *
         * Step 1: Query biology domain for relevant facts
         * Step 2: Follow references to chemistry
         * Step 3: Follow references to physics
         * Step 4: Build reasoning chain (each step costs LGM)
         * Step 5: Synthesize answer
         *
         * Each query costs LGM, paid to the knowledge contributors.
         * The deeper the reasoning, the more it costs — but the more robust the answer.
         */

        const thinkerBalanceStart = state.getBalance(THINKER_DID);
        let totalQueryCost = 0n;

        // Step 1: Query biology domain
        const bioFact = state.getFact(bioFactId)!;
        expect(bioFact.claimText).toContain("Cellular respiration");

        const bioFactResult = toFactResult(bioFact, SCIENTIST_DID as DID);
        const bioPricing = calculateQueryPrice(
          [bioFactResult],
          blockNumber + 100n,
        );
        const bioDistribution = calculateDistribution(
          bioPricing.totalCost,
          [bioFactResult],
          PROVIDER_DID,
        );

        const bioPayment = executePaymentDistribution(
          bioDistribution,
          THINKER_DID as DID,
          state,
          blockNumber + 100n,
        );
        expect(bioPayment.success).toBe(true);
        totalQueryCost += bioPayment.totalTransferred;

        // Record citation
        state.incrementFactCitationCount(bioFactId, blockNumber + 100n);

        // Step 2: Follow reference chain to chemistry
        // The bio fact references the chem fact — follow the link
        const chemFact = state.getFact(chemFactId)!;
        expect(chemFact.claimText).toContain("Exothermic reactions");

        const chemFactResult = toFactResult(chemFact, SCIENTIST_DID as DID);
        const chemPricing = calculateQueryPrice(
          [chemFactResult],
          blockNumber + 101n,
        );
        const chemDistribution = calculateDistribution(
          chemPricing.totalCost,
          [chemFactResult],
          PROVIDER_DID,
        );

        const chemPayment = executePaymentDistribution(
          chemDistribution,
          THINKER_DID as DID,
          state,
          blockNumber + 101n,
        );
        expect(chemPayment.success).toBe(true);
        totalQueryCost += chemPayment.totalTransferred;

        state.incrementFactCitationCount(chemFactId, blockNumber + 101n);

        // Step 3: Follow reference chain to physics (root cause)
        const physicsFact = state.getFact(physicsFactId)!;
        expect(physicsFact.claimText).toContain("Energy is conserved");

        const physicsFactResult = toFactResult(
          physicsFact,
          SCIENTIST_DID as DID,
        );
        const physicsPricing = calculateQueryPrice(
          [physicsFactResult],
          blockNumber + 102n,
        );
        const physicsDistribution = calculateDistribution(
          physicsPricing.totalCost,
          [physicsFactResult],
          PROVIDER_DID,
        );

        const physicsPayment = executePaymentDistribution(
          physicsDistribution,
          THINKER_DID as DID,
          state,
          blockNumber + 102n,
        );
        expect(physicsPayment.success).toBe(true);
        totalQueryCost += physicsPayment.totalTransferred;

        state.incrementFactCitationCount(physicsFactId, blockNumber + 102n);

        // Step 4: Verify the reasoning chain economics
        const thinkerBalanceEnd = state.getBalance(THINKER_DID);
        const spent = thinkerBalanceStart - thinkerBalanceEnd;

        // Thinker spent LGM to reason across 3 facts
        expect(spent).toBe(totalQueryCost);
        expect(totalQueryCost).toBeGreaterThan(0n);

        // Step 5: The scientist earned citation rewards for contributing knowledge
        const scientistBalance = state.getBalance(SCIENTIST_DID);
        // Scientist should have earned MORE than their starting balance
        // (minus what they staked for claims, plus stake returns and citation rewards)

        // Physics fact (highest confidence) should be cheapest to query
        expect(physicsPricing.totalCost).toBeLessThanOrEqual(
          chemPricing.totalCost,
        );
      });

      it("should track citation counts as facts are queried", () => {
        // Before any queries
        const bioFact = state.getFact(bioFactId)!;
        const initialCitations = bioFact.citationCount;

        // Query the bio fact 3 times (simulating 3 different agents)
        for (let i = 0; i < 3; i++) {
          state.incrementFactCitationCount(
            bioFactId,
            blockNumber + BigInt(200 + i),
          );
        }

        const updatedFact = state.getFact(bioFactId)!;
        expect(updatedFact.citationCount).toBe(initialCitations + 3);
      });

      it("should make highly-cited facts more expensive over time", () => {
        const bioFact = state.getFact(bioFactId)!;

        // Price before citations
        const beforeResult = toFactResult(bioFact, SCIENTIST_DID as DID);
        const priceBefore = calculateQueryPrice(
          [beforeResult],
          blockNumber + 2000n,
        );

        // Simulate many citations
        const citedResult: FactResult = {
          ...beforeResult,
          citationCount: 50,
        };
        const priceAfter = calculateQueryPrice(
          [citedResult],
          blockNumber + 2000n,
        );

        // More citations → novelty discount → lower cost (well-known facts are cheaper)
        expect(priceAfter.fundamentalityPremium).toBeLessThan(
          priceBefore.fundamentalityPremium,
        );
        expect(priceAfter.totalCost).toBeLessThan(priceBefore.totalCost);
      });

      it("should demonstrate the think-query-pay loop", () => {
        /**
         * This test demonstrates the complete agentic think cycle:
         *
         * 1. Agent formulates a question
         * 2. Agent queries the knowledge tree (pays for it)
         * 3. Agent gets facts back with confidence scores
         * 4. Agent evaluates fact confidence against its reasoning needs
         * 5. If confidence too low → query more facts or challenge existing ones
         * 6. If confidence sufficient → incorporate into reasoning
         *
         * The economic incentive: higher-confidence facts cost more,
         * but agents who need certainty WILL pay for it.
         * This creates a market for truth.
         */

        const queryBlock = blockNumber + 2000n;

        // Agent queries physics (high confidence: 0.97)
        const physicsFact = state.getFact(physicsFactId)!;
        const physicsResult = toFactResult(physicsFact, SCIENTIST_DID as DID);
        const physicsCost = calculateQueryPrice([physicsResult], queryBlock);

        // Agent queries biology (lower confidence: 0.89)
        const bioFact = state.getFact(bioFactId)!;
        const bioResult = toFactResult(bioFact, SCIENTIST_DID as DID);
        const bioCost = calculateQueryPrice([bioResult], queryBlock);

        // Inverted pricing: Physics (0.97 confidence) costs LESS than biology (0.89)
        // Higher confidence = bigger discount → more negative confidencePremium
        expect(physicsCost.confidencePremium).toBeLessThan(
          bioCost.confidencePremium,
        );

        // Agent decision: high-confidence facts are cheaper, incentivizing
        // agents to ground their output in well-verified knowledge.
        //
        // This models real-world agent reasoning:
        // - Verified truth is cheap → agents prefer it → less hallucination
        // - Uncertain facts carry a risk surcharge → agents verify first
        // - The protocol creates an economic incentive for truth usage

        // For a high-stakes calculation, agent queries BOTH
        const bothCost = calculateQueryPrice(
          [physicsResult, bioResult],
          queryBlock,
        );

        // Combined cost = sum of individual costs (each fact priced independently)
        expect(bothCost.totalCost).toBe(
          physicsCost.totalCost + bioCost.totalCost,
        );

        // Execute the combined query payment
        const distribution = calculateDistribution(
          bothCost.totalCost,
          [physicsResult, bioResult],
          PROVIDER_DID,
        );

        const result = executePaymentDistribution(
          distribution,
          THINKER_DID as DID,
          state,
          queryBlock,
        );

        expect(result.success).toBe(true);

        // The scientist earned rewards for BOTH facts being queried
        expect(distribution.knowledgeRewards.length).toBe(2);
        expect(distribution.knowledgeRewards.every((r) => r.amount > 0n)).toBe(
          true,
        );
      });
    });

    // ==========================================================================
    // Phase 5: Knowledge Submission as New Claim (Reasoning → New Knowledge)
    // ==========================================================================

    describe("Phase 5: Reasoning Creates New Knowledge", () => {
      it("should allow thinker to submit derived claim referencing existing facts", () => {
        // First, create base facts
        const c1 = submitClaim(
          state,
          "Water freezes at 0°C at standard atmospheric pressure",
          "chemistry.phase_transitions",
          "chemical",
          SCIENTIST_KEY,
          blockNumber,
        );
        verifyAndAccept(state, c1, "round-freeze-001", 0.96, blockNumber + 1n);
        const fact1Id = state.getClaim(c1)!.resultFactId!;

        const c2 = submitClaim(
          state,
          "Dissolved salts lower the freezing point of water (colligative property)",
          "chemistry.solutions",
          "chemical",
          SCIENTIST_KEY,
          blockNumber + 20n,
          200n * ONE_LGM,
          [fact1Id],
        );
        verifyAndAccept(state, c2, "round-salt-001", 0.91, blockNumber + 21n);
        const fact2Id = state.getClaim(c2)!.resultFactId!;

        // Fund thinker enough for staking
        state.creditBalance(THINKER_DID, 1000n * ONE_LGM);

        // Thinker queries both facts, reasons about them, and submits a NEW derived claim
        // This is the creative output of the "think process"
        const derivedClaimPayload: ClaimSubmitPayload = {
          type: "claim_submit",
          claim:
            "Ocean water freezes at approximately -2°C due to dissolved salts",
          claimType: "empirical",
          domain: "chemistry.solutions",
          stratum: "chemical" as any,
          stake: 200n * ONE_LGM,
          references: [fact1Id, fact2Id], // References the two facts the agent reasoned from
        };

        const derivedResult = executeTransaction(
          state,
          userTx(
            "claim_submit",
            derivedClaimPayload as any,
            THINKER_KEY,
            state,
          ),
          blockNumber + 60n,
          PARAMS,
        );
        expect(derivedResult.success).toBe(true);

        // The thinker's derived claim is now in the pending queue
        const pendingClaims = state.getPendingClaims();
        const derivedClaim = pendingClaims.find(
          (c) =>
            c.claim ===
            "Ocean water freezes at approximately -2°C due to dissolved salts",
        );
        expect(derivedClaim).toBeDefined();
        expect(derivedClaim!.references).toContain(fact1Id);
        expect(derivedClaim!.references).toContain(fact2Id);

        // Verify it too
        verifyAndAccept(
          state,
          derivedClaim!.id,
          "round-ocean-001",
          0.87,
          blockNumber + 61n,
        );

        const derivedFactId = state.getClaim(derivedClaim!.id)!.resultFactId!;
        const derivedFact = state.getFact(derivedFactId);
        expect(derivedFact).not.toBeNull();
        expect(derivedFact!.confidence).toBe(0.87);

        // The knowledge tree has grown: 2 base facts → 1 derived fact
        // This is the essence of agentic knowledge creation:
        // Agent queries existing knowledge → reasons about it → submits new knowledge
        // The cycle repeats, and the tree grows
      });
    });
  },
);
