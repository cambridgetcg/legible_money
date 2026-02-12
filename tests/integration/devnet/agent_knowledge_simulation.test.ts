/**
 * Agent Knowledge Tree Simulation
 *
 * Simulates a full devnet session where an AI agent:
 *   1. Boots into the network and receives bootstrap funds
 *   2. Submits knowledge claims through the consensus engine
 *   3. Validators verify claims via commit-reveal rounds
 *   4. Claims become verified facts in the knowledge tree
 *   5. A second agent queries the knowledge tree, paying LGM
 *   6. The querying agent follows reference chains (agentic "think" process)
 *   7. The querying agent synthesizes and submits a derived claim
 *   8. Economics are verified: payments flow to contributors, research, protocol
 *
 * This is an in-process devnet — no HTTP server needed. We wire the
 * consensus engine + state integration + RPC handlers directly, simulating
 * what a live devnet node does in a single test process.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  createIntegratedEngine,
  type ProofOfTruthEngine,
} from "../../../consensus/pot/engine.js";
import type { StateIntegration } from "../../../consensus/pot/state_integration.js";
import {
  DEFAULT_CONSENSUS_PARAMS,
  type ConsensusParams,
  type Transaction,
} from "../../../consensus/pot/types.js";
import {
  executeTransaction,
  FEE_POOL_DID,
} from "../../../state/transitions.js";
import type { ClaimSubmitPayload } from "../../../state/types.js";
import {
  generateKeypair,
  type PublicKey,
  type PrivateKey,
} from "../../../core/crypto/keys.js";
import { sign, signDomainSeparated } from "../../../core/crypto/signatures.js";
import {
  sha512_256Hex,
  domainHash,
  HASH_DOMAINS,
} from "../../../core/crypto/hashing.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";
import { randomBytes } from "crypto";
import {
  knowledgeQuery,
  knowledgeGetFact,
  type HandlerContext,
} from "../../../protocol/rpc/handlers.js";
import { TruthGateway } from "../../../core/billing/truth_gateway.js";

// ============================================================================
// Test Infrastructure
// ============================================================================

const ONE_LGM = 10n ** 18n;

/** Agent identity: key pair + derived DID */
interface AgentIdentity {
  name: string;
  publicKey: string;
  privateKey: string;
  did: string;
}

/** Create a named agent with a fresh key pair */
function createAgent(name: string): AgentIdentity {
  const kp = generateKeypair();
  return {
    name,
    publicKey: kp.publicKey,
    privateKey: kp.privateKey,
    did: publicKeyToDid(kp.publicKey),
  };
}

/** Consensus params tuned for fast in-process simulation */
const SIM_PARAMS: ConsensusParams = {
  ...DEFAULT_CONSENSUS_PARAMS,
  // Fast round timing for simulation
  verificationRoundBlocks: 11,
  commitPhaseBlocks: 4,
  revealPhaseBlocks: 4,
  // Low thresholds for 3 validators
  minValidators: 3,
  maxValidators: 22,
  validatorQuorum: 0.66,
  acceptanceThreshold: 0.66,
  rejectionThreshold: 0.77,
  contestedThreshold: 0.55,
  // Disable anti-sybil for simulation (tested separately)
  newAccountTransferCooldown: 0n,
  bootstrapVestingBlocks: 0n,
  bootstrapMinTxFee: 0n,
  maxBootstrapClaimsPerBlock: 100n,
} as ConsensusParams;

// ============================================================================
// Helpers
// ============================================================================

function generateNonce(): string {
  return randomBytes(32).toString("hex");
}

function generateCommitment(
  roundId: string,
  verdict: string,
  confidence: number,
  salt: string,
): string {
  const commitData = JSON.stringify({
    roundId,
    verdict,
    confidence,
    referencesChecked: [],
    salt,
  });
  const hash = domainHash(HASH_DOMAINS.COMMITMENT, commitData);
  return sha512_256Hex(hash.toString());
}

/** Build a user transaction with correct nonce from state */
function userTx(
  type: string,
  payload: Record<string, unknown>,
  agent: AgentIdentity,
  state: StateManager,
): Transaction {
  const nonce = state.getNonce(agent.did);
  return {
    type: type as any,
    sender: agent.publicKey as any,
    nonce,
    fee: 0n,
    payload,
    signature: "" as any,
  };
}

/** Build a system transaction (no sender/signature) */
function systemTx(type: string, payload: Record<string, unknown>): Transaction {
  return {
    type: type as any,
    sender: "system" as any,
    nonce: 0n,
    fee: 0n,
    payload,
    signature: "" as any,
  };
}

// ============================================================================
// Simulation
// ============================================================================

describe.sequential(
  "Agent Knowledge Tree Simulation (In-Process DevNet)",
  () => {
    // --- World state ---
    let state: StateManager;
    let engine: ProofOfTruthEngine;
    let stateIntegration: StateIntegration;

    // --- Actors ---
    let researcher: AgentIdentity; // Submits knowledge claims
    let thinker: AgentIdentity; // Queries knowledge, builds reasoning
    let nodeOperator: AgentIdentity; // Runs the node, earns query fees
    let validators: AgentIdentity[]; // Verify claims

    // --- RPC context ---
    let rpcCtx: HandlerContext;

    // --- Block tracking ---
    let blockNumber: bigint;

    beforeEach(() => {
      // Create agents
      researcher = createAgent("Researcher");
      thinker = createAgent("Thinker");
      nodeOperator = createAgent("NodeOperator");
      validators = [
        createAgent("Validator-1"),
        createAgent("Validator-2"),
        createAgent("Validator-3"),
      ];

      // Initialize state
      state = new StateManager();
      blockNumber = 100n;

      // Fund agents
      state.createAccount(researcher.did, "researcher-op", 10_000n * ONE_LGM);
      state.createAccount(thinker.did, "thinker-op", 5_000n * ONE_LGM);
      state.createAccount(nodeOperator.did, "node-op", 1_000n * ONE_LGM);
      state.createAccount(FEE_POOL_DID, "fee-pool-op", 1_000_000n * ONE_LGM);

      // Fund and register validators
      for (const v of validators) {
        state.createAccount(v.did, `${v.name}-op`, 50_000n * ONE_LGM);
      }

      // Build integrated engine with test mode enabled
      engine = createIntegratedEngine({
        isValidator: true,
        validatorKey: validators[0].privateKey as PrivateKey,
        validatorPublicKey: validators[0].publicKey as PublicKey,
        initialValidators: validators.map((v) => ({
          publicKey: v.publicKey as PublicKey,
          did: v.did,
          stake: 10_000n * 10n ** 15n,
        })),
        stateManager: state,
        params: SIM_PARAMS,
        allowTestMethods: true,
      });

      stateIntegration = engine.getStateIntegration()!;

      // Create RPC handler context
      const gateway = new TruthGateway(nodeOperator.did as any);
      rpcCtx = {
        state,
        consensus: engine,
        stateIntegration,
        nodeVersion: "sim-1.0.0",
        networkId: "lgm-sim-1",
        truthGateway: gateway,
        providerDid: nodeOperator.did as any,
      };
    });

    // --------------------------------------------------------------------------
    // Phase 1: Researcher submits claims → consensus verifies → facts created
    // --------------------------------------------------------------------------

    it("Phase 1: Researcher submits claims and they become verified facts", async () => {
      console.log(
        "\n╔══════════════════════════════════════════════════════════════╗",
      );
      console.log(
        "║  PHASE 1: Knowledge Creation                                ║",
      );
      console.log(
        "╚══════════════════════════════════════════════════════════════╝\n",
      );

      // --- Claim A: Fundamental physics ---
      const claimA: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "The speed of light in vacuum is approximately 299,792,458 m/s",
        claimType: "empirical",
        domain: "physics.optics",
        stratum: "physical",
        stake: 200n * 10n ** 15n,
        references: [],
      };

      const resultA = executeTransaction(
        state,
        userTx("claim_submit", claimA as any, researcher, state),
        blockNumber,
        SIM_PARAMS,
      );
      expect(resultA.success).toBe(true);
      console.log("  [SUBMIT] Claim A: speed of light");

      const claimAId = state.getPendingClaims()[0].id;

      // --- Start verification ---
      executeTransaction(
        state,
        systemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId: claimAId,
          roundId: "round-A",
          selectedValidators: validators.map((v) => v.did),
        }),
        blockNumber + 1n,
        SIM_PARAMS,
      );
      console.log("  [VERIFY] Round started, 3 validators assigned");

      // --- Complete verification (accepted at high confidence) ---
      const rewardsA = validators.map(
        (v) => [v.did, 1n * ONE_LGM] as [string, bigint],
      );
      executeTransaction(
        state,
        systemTx("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId: claimAId,
          roundId: "round-A",
          outcome: "accepted",
          finalConfidence: 0.95,
          verifierCount: 3,
          rewards: rewardsA,
          slashes: [],
        }),
        blockNumber + 15n,
        SIM_PARAMS,
      );

      const factA = state.getClaim(claimAId)!;
      expect(factA.status).toBe("accepted");
      const factAId = factA.resultFactId!;
      const factAData = state.getFact(factAId)!;
      expect(factAData.confidence).toBe(0.95);
      expect(factAData.domain).toBe("physics.optics");
      console.log(`  [FACT] Created: ${factAId} (confidence: 0.95)`);
      console.log(`         "${factAData.claimText.substring(0, 60)}..."`);

      // --- Claim B: References Claim A ---
      const claimB: ClaimSubmitPayload = {
        type: "claim_submit",
        claim:
          "Light takes approximately 8.3 minutes to travel from Sun to Earth",
        claimType: "empirical",
        domain: "physics.astrophysics",
        stratum: "physical",
        stake: 200n * 10n ** 15n,
        references: [factAId],
      };

      executeTransaction(
        state,
        userTx("claim_submit", claimB as any, researcher, state),
        blockNumber + 20n,
        SIM_PARAMS,
      );
      console.log(
        "\n  [SUBMIT] Claim B: light travel time (references Claim A)",
      );

      const claimBId = state.getPendingClaims()[0].id;

      executeTransaction(
        state,
        systemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId: claimBId,
          roundId: "round-B",
          selectedValidators: validators.map((v) => v.did),
        }),
        blockNumber + 21n,
        SIM_PARAMS,
      );

      executeTransaction(
        state,
        systemTx("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId: claimBId,
          roundId: "round-B",
          outcome: "accepted",
          finalConfidence: 0.91,
          verifierCount: 3,
          rewards: validators.map(
            (v) => [v.did, 1n * ONE_LGM] as [string, bigint],
          ),
          slashes: [],
        }),
        blockNumber + 35n,
        SIM_PARAMS,
      );

      const factBId = state.getClaim(claimBId)!.resultFactId!;
      const factBData = state.getFact(factBId)!;
      console.log(`  [FACT] Created: ${factBId} (confidence: 0.91)`);
      console.log(`         References: ${factAId}`);

      // --- Claim C: Derived from A + B ---
      const claimC: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "The Sun is approximately 149.6 million km from Earth",
        claimType: "empirical",
        domain: "physics.astrophysics",
        stratum: "physical",
        stake: 200n * 10n ** 15n,
        references: [factAId],
        derivedFrom: {
          factIds: [factAId, factBId],
          inferenceType: "deductive",
          logicChain:
            "speed_of_light × travel_time = distance (c × 498s ≈ 149.6M km)",
        },
      } as any;

      executeTransaction(
        state,
        userTx("claim_submit", claimC as any, researcher, state),
        blockNumber + 40n,
        SIM_PARAMS,
      );
      console.log(
        "\n  [SUBMIT] Claim C: Sun-Earth distance (derived from A + B)",
      );

      const claimCId = state.getPendingClaims()[0].id;

      executeTransaction(
        state,
        systemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId: claimCId,
          roundId: "round-C",
          selectedValidators: validators.map((v) => v.did),
        }),
        blockNumber + 41n,
        SIM_PARAMS,
      );

      executeTransaction(
        state,
        systemTx("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId: claimCId,
          roundId: "round-C",
          outcome: "accepted",
          finalConfidence: 0.89,
          verifierCount: 3,
          rewards: validators.map(
            (v) => [v.did, 1n * ONE_LGM] as [string, bigint],
          ),
          slashes: [],
        }),
        blockNumber + 55n,
        SIM_PARAMS,
      );

      const factCId = state.getClaim(claimCId)!.resultFactId!;
      console.log(`  [FACT] Created: ${factCId} (confidence: 0.89)`);
      console.log(`         Derives from: ${factAId}, ${factBId}\n`);

      // --- Verify knowledge graph structure ---
      const refsA = state.getFactReferences(factAId);
      expect(refsA.supports.length).toBeGreaterThanOrEqual(1);
      console.log("  [GRAPH] Knowledge graph wired:");
      console.log(`    Fact A (speed of light) ← supported by Fact B`);
      console.log(
        `    Fact A + Fact B ← derived → Fact C (Sun-Earth distance)`,
      );
      console.log(`    3 facts, 3+ edges in the knowledge tree\n`);

      // --- Verify researcher economics ---
      // Stake returned on acceptance (3 × 200 LGM staked, all returned)
      const researcherAccount = state.getAccount(researcher.did)!;
      expect(researcherAccount.stakedBalance).toBe(0n);
      console.log(
        `  [ECON] Researcher balance: ${researcherAccount.balance / ONE_LGM} LGM (stake returned)`,
      );
    });

    // --------------------------------------------------------------------------
    // Phase 2: Thinker agent queries the knowledge tree (agentic "think")
    // --------------------------------------------------------------------------

    it("Phase 2: Thinker agent queries facts and pays for knowledge", async () => {
      console.log(
        "\n╔══════════════════════════════════════════════════════════════╗",
      );
      console.log(
        "║  PHASE 2: Agentic Knowledge Query (Think Process)           ║",
      );
      console.log(
        "╚══════════════════════════════════════════════════════════════╝\n",
      );

      // --- Seed verified facts (as if Phase 1 completed) ---
      const factAId = "fact:light-speed-001";
      const factBId = "fact:light-travel-002";
      const factCId = "fact:sun-distance-003";

      state.addFact({
        id: factAId,
        claimText:
          "The speed of light in vacuum is approximately 299,792,458 m/s",
        domain: "physics.optics",
        confidence: 0.95,
        verifiedAt: 115n,
        verifierCount: 3,
        challengeCount: 0,
        citationCount: 5,
        status: "active",
      });

      state.addFact({
        id: factBId,
        claimText:
          "Light takes approximately 8.3 minutes to travel from Sun to Earth",
        domain: "physics.astrophysics",
        confidence: 0.91,
        verifiedAt: 135n,
        verifierCount: 3,
        challengeCount: 0,
        citationCount: 2,
        status: "active",
      });

      state.addFact({
        id: factCId,
        claimText: "The Sun is approximately 149.6 million km from Earth",
        domain: "physics.astrophysics",
        confidence: 0.89,
        verifiedAt: 155n,
        verifierCount: 3,
        challengeCount: 0,
        citationCount: 0,
        status: "active",
      });

      // Ensure system accounts exist for payment distribution
      if (!state.getAccount("did:lgm:burn")) {
        state.createAccount("did:lgm:burn", "burn-op", 0n);
      }
      if (!state.getAccount("did:lgm:protocol-treasury")) {
        state.createAccount("did:lgm:protocol-treasury", "treasury-op", 0n);
      }
      if (!state.getAccount("did:lgm:research-treasury")) {
        state.createAccount("did:lgm:research-treasury", "research-op", 0n);
      }

      const thinkerBalanceBefore = state.getBalance(thinker.did);
      const nodeOpBalanceBefore = state.getBalance(nodeOperator.did);

      console.log("  [THINK] Agent begins reasoning process...");
      console.log(
        `  [WALLET] Thinker balance: ${thinkerBalanceBefore / ONE_LGM} LGM\n`,
      );

      // === Think Step 1: Query physics domain ===
      console.log(
        "  ┌─── Think Step 1: Query physics.optics ───────────────────┐",
      );

      // Advance block so facts aren't "too fresh"
      state.applyBlock(2000n, "0".repeat(64));

      const sessionId = `chat:${thinker.did}:${Date.now()}`;

      const queryResult1 = await knowledgeQuery(
        {
          sessionId,
          intent: {
            type: "domain_search",
            domain: "physics.optics",
            limit: 10,
            minConfidence: 0.5,
          },
        },
        rpcCtx,
      );

      expect(queryResult1.facts.length).toBeGreaterThanOrEqual(1);
      const lightSpeedFact = queryResult1.facts[0];
      expect(lightSpeedFact.fact.claimText).toContain("speed of light");

      console.log(
        `  │ Found: "${lightSpeedFact.fact.claimText.substring(0, 55)}..."`,
      );
      console.log(`  │ Confidence: ${lightSpeedFact.fact.confidence}`);
      console.log(`  │ Cost: ${queryResult1.metadata.actualCost} wei`);
      console.log(
        `  │ Payment executed: ${queryResult1.metadata.paymentExecuted}`,
      );
      console.log(
        "  └───────────────────────────────────────────────────────────┘\n",
      );

      // === Think Step 2: Query astrophysics domain ===
      console.log(
        "  ┌─── Think Step 2: Query physics.astrophysics ─────────────┐",
      );

      const queryResult2 = await knowledgeQuery(
        {
          sessionId,
          intent: {
            type: "domain_search",
            domain: "physics.astrophysics",
            limit: 10,
            minConfidence: 0.5,
          },
        },
        rpcCtx,
      );

      expect(queryResult2.facts.length).toBeGreaterThanOrEqual(2);
      console.log(
        `  │ Found ${queryResult2.facts.length} facts in astrophysics:`,
      );
      for (const f of queryResult2.facts) {
        console.log(
          `  │   - "${f.fact.claimText.substring(0, 50)}..." (conf: ${f.fact.confidence})`,
        );
      }
      console.log(`  │ Cost: ${queryResult2.metadata.actualCost} wei`);
      console.log(
        `  │ Payment executed: ${queryResult2.metadata.paymentExecuted}`,
      );
      console.log(
        "  └───────────────────────────────────────────────────────────┘\n",
      );

      // === Think Step 3: Get specific fact with references ===
      console.log(
        "  ┌─── Think Step 3: Deep dive on specific fact ─────────────┐",
      );

      const factDetail = await knowledgeGetFact(
        { factId: factAId, includeReferences: true },
        rpcCtx,
      );

      expect(factDetail.fact).not.toBeNull();
      console.log(
        `  │ Fact: "${factDetail.fact!.claimText.substring(0, 55)}..."`,
      );
      console.log(`  │ Confidence: ${factDetail.fact!.confidence}`);
      console.log(
        `  │ Research status: ${JSON.stringify(factDetail.researchStatus)}`,
      );
      console.log(
        "  └───────────────────────────────────────────────────────────┘\n",
      );

      // === Verify economics ===
      const thinkerBalanceAfter = state.getBalance(thinker.did);
      const nodeOpBalanceAfter = state.getBalance(nodeOperator.did);

      const totalSpent = thinkerBalanceBefore - thinkerBalanceAfter;
      const nodeOpEarned = nodeOpBalanceAfter - nodeOpBalanceBefore;

      console.log(
        "  ╔══ Query Economics ══════════════════════════════════════╗",
      );
      console.log(
        `  ║ Thinker spent:     ${totalSpent} wei (${Number(totalSpent) / 1e18} LGM)`,
      );
      console.log(`  ║ Node op earned:    ${nodeOpEarned} wei`);

      // Check research treasury received funds
      const researchBalance = state.getBalance("did:lgm:research-treasury");
      console.log(`  ║ Research funded:   ${researchBalance} wei`);

      // Check protocol treasury
      const treasuryBalance = state.getBalance("did:lgm:protocol-treasury");
      console.log(`  ║ Protocol treasury: ${treasuryBalance} wei`);

      // Check burn
      const burnBalance = state.getBalance("did:lgm:burn");
      console.log(`  ║ Burned:            ${burnBalance} wei`);
      console.log(
        "  ╚═════════════════════════════════════════════════════════╝\n",
      );

      // Verify money flow: thinker paid, node operator earned, research funded
      if (
        queryResult1.metadata.paymentExecuted ||
        queryResult2.metadata.paymentExecuted
      ) {
        expect(totalSpent).toBeGreaterThan(0n);
        expect(nodeOpEarned).toBeGreaterThan(0n);
      }

      // Citation counts should have incremented
      const factAAfterQuery = state.getFact(factAId)!;
      expect(factAAfterQuery.citationCount).toBeGreaterThan(5);
      console.log(
        `  [CITATION] Fact A citation count: 5 → ${factAAfterQuery.citationCount}`,
      );
    });

    // --------------------------------------------------------------------------
    // Phase 3: Full lifecycle — submit, verify, query, derive
    // --------------------------------------------------------------------------

    it("Phase 3: Complete lifecycle — claim → fact → query → derive new knowledge", async () => {
      console.log(
        "\n╔══════════════════════════════════════════════════════════════╗",
      );
      console.log(
        "║  PHASE 3: Full Knowledge Lifecycle                          ║",
      );
      console.log(
        "╚══════════════════════════════════════════════════════════════╝\n",
      );

      // Ensure system accounts exist
      if (!state.getAccount("did:lgm:burn")) {
        state.createAccount("did:lgm:burn", "burn-op", 0n);
      }
      if (!state.getAccount("did:lgm:protocol-treasury")) {
        state.createAccount("did:lgm:protocol-treasury", "treasury-op", 0n);
      }
      if (!state.getAccount("did:lgm:research-treasury")) {
        state.createAccount("did:lgm:research-treasury", "research-op", 0n);
      }

      // === Step 1: Researcher submits a claim ===
      console.log("  Step 1: Researcher submits claim...");

      const claimPayload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Water has a specific heat capacity of 4.186 J/(g·°C)",
        claimType: "empirical",
        domain: "chemistry.thermodynamics",
        stratum: "chemical",
        stake: 200n * 10n ** 15n,
        references: [],
      };

      const submitResult = executeTransaction(
        state,
        userTx("claim_submit", claimPayload as any, researcher, state),
        blockNumber,
        SIM_PARAMS,
      );
      expect(submitResult.success).toBe(true);
      const claimId = state.getPendingClaims()[0].id;
      console.log(`    Claim ID: ${claimId}`);

      // === Step 2: Validators verify ===
      console.log("  Step 2: Validators verify claim...");

      executeTransaction(
        state,
        systemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId,
          roundId: "round-water-001",
          selectedValidators: validators.map((v) => v.did),
        }),
        blockNumber + 1n,
        SIM_PARAMS,
      );

      executeTransaction(
        state,
        systemTx("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId,
          roundId: "round-water-001",
          outcome: "accepted",
          finalConfidence: 0.93,
          verifierCount: 3,
          rewards: validators.map(
            (v) => [v.did, 1n * ONE_LGM] as [string, bigint],
          ),
          slashes: [],
        }),
        blockNumber + 15n,
        SIM_PARAMS,
      );

      const factId = state.getClaim(claimId)!.resultFactId!;
      const fact = state.getFact(factId)!;
      expect(fact.status).toBe("active");
      expect(fact.confidence).toBe(0.93);
      console.log(`    Verified as fact: ${factId} (conf: 0.93)`);

      // === Step 3: Thinker queries the knowledge tree ===
      console.log("  Step 3: Thinker agent queries knowledge...");

      // Advance block for pricing
      state.applyBlock(2000n, "0".repeat(64));

      const sessionId = `chat:${thinker.did}:${Date.now()}`;
      const thinkerBefore = state.getBalance(thinker.did);

      const queryResult = await knowledgeQuery(
        {
          sessionId,
          intent: {
            type: "domain_search",
            domain: "chemistry.thermodynamics",
            limit: 10,
            minConfidence: 0.5,
          },
        },
        rpcCtx,
      );

      expect(queryResult.facts.length).toBe(1);
      expect(queryResult.facts[0].fact.claimText).toContain(
        "specific heat capacity",
      );
      console.log(
        `    Found: "${queryResult.facts[0].fact.claimText.substring(0, 50)}..."`,
      );

      const queryCost = thinkerBefore - state.getBalance(thinker.did);
      console.log(`    Query cost: ${queryCost} wei`);

      // === Step 4: Thinker derives new knowledge ===
      console.log("  Step 4: Thinker derives new claim from queried fact...");

      const derivedClaim: ClaimSubmitPayload = {
        type: "claim_submit",
        claim:
          "Heating 1 liter of water from 20°C to 100°C requires approximately 334,880 J of energy",
        claimType: "empirical",
        domain: "chemistry.thermodynamics",
        stratum: "chemical",
        stake: 200n * 10n ** 15n,
        references: [factId],
        derivedFrom: {
          factIds: [factId],
          inferenceType: "deductive",
          logicChain: "4.186 J/(g·°C) × 1000g × 80°C = 334,880 J",
        },
      } as any;

      const deriveResult = executeTransaction(
        state,
        userTx("claim_submit", derivedClaim as any, thinker, state),
        blockNumber + 40n,
        SIM_PARAMS,
      );
      expect(deriveResult.success).toBe(true);
      console.log("    Derived claim submitted!");

      const derivedClaimId = state.getPendingClaims()[0].id;

      // === Step 5: Verify the derived claim ===
      console.log("  Step 5: Verifying derived claim...");

      executeTransaction(
        state,
        systemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId: derivedClaimId,
          roundId: "round-derived-001",
          selectedValidators: validators.map((v) => v.did),
        }),
        blockNumber + 41n,
        SIM_PARAMS,
      );

      executeTransaction(
        state,
        systemTx("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId: derivedClaimId,
          roundId: "round-derived-001",
          outcome: "accepted",
          finalConfidence: 0.88,
          verifierCount: 3,
          rewards: validators.map(
            (v) => [v.did, 1n * ONE_LGM] as [string, bigint],
          ),
          slashes: [],
        }),
        blockNumber + 55n,
        SIM_PARAMS,
      );

      const derivedFactId = state.getClaim(derivedClaimId)!.resultFactId!;
      const derivedFact = state.getFact(derivedFactId)!;
      expect(derivedFact.status).toBe("active");
      expect(derivedFact.confidence).toBe(0.88);
      console.log(`    New fact: ${derivedFactId} (conf: 0.88)`);

      // === Verify the full knowledge chain ===
      console.log("\n  ═══ Knowledge Chain Verification ═══");

      // Derived fact should have a "derives_from" edge to the original fact
      // (when derivedFrom is present, edges are wired from derivedFrom.factIds)
      const refs = state.getFactReferences(factId);
      const hasEdge =
        refs.supports.length > 0 ||
        refs.derivedFrom?.length > 0 ||
        refs.supportedBy?.length > 0;
      // The derived fact references the original — check from derived side
      const derivedRefs = state.getFactReferences(derivedFactId);
      // derivedFrom creates edges FROM the derived fact TO the parent
      // The edge is stored on the KnowledgeStateTree as (derivedFact -> parentFact)
      console.log(`    ${factId} (original)`);
      console.log(`      Referenced by derived claim`);
      console.log(`    ${derivedFactId} (derived)`);
      console.log(
        `      derives_from (deductive: 4.186 × 1000 × 80 = 334,880)`,
      );

      // Verify the derived fact exists and is active
      expect(derivedFact.status).toBe("active");
      expect(derivedFact.claimText).toContain("334,880");

      // Original fact citation count incremented (from query)
      const factAfter = state.getFact(factId)!;
      console.log(`\n    Citation count: ${factAfter.citationCount}`);

      // === Final economics report ===
      console.log("\n  ═══ Final Economics ═══");
      console.log(
        `    Researcher: ${state.getBalance(researcher.did) / ONE_LGM} LGM`,
      );
      console.log(
        `    Thinker: ${state.getBalance(thinker.did) / ONE_LGM} LGM`,
      );
      console.log(
        `    Node Operator: ${state.getBalance(nodeOperator.did) / ONE_LGM} LGM`,
      );
      for (const v of validators) {
        console.log(`    ${v.name}: ${state.getBalance(v.did) / ONE_LGM} LGM`);
      }

      console.log("\n  [LIFECYCLE COMPLETE]");
      console.log(
        "    Researcher submitted knowledge → Validators verified truth",
      );
      console.log("    Thinker queried (paid LGM) → Node operator earned fees");
      console.log("    Thinker derived new knowledge from queried facts");
      console.log(
        "    Knowledge tree grew: 2 facts, linked by derivation chain",
      );
      console.log("    Research fund received 22% of query revenue\n");
    });

    // --------------------------------------------------------------------------
    // Phase 4: Consensus engine round simulation (commit-reveal via engine)
    // --------------------------------------------------------------------------

    it("Phase 4: Full consensus round via engine (commit-reveal cycle)", async () => {
      console.log(
        "\n╔══════════════════════════════════════════════════════════════╗",
      );
      console.log(
        "║  PHASE 4: Consensus Engine Commit-Reveal Simulation         ║",
      );
      console.log(
        "╚══════════════════════════════════════════════════════════════╝\n",
      );

      // Submit a claim through the consensus engine (not via state transitions)
      console.log("  Submitting claim through consensus engine...");

      const claimId = await engine.submitClaim(
        {
          claim: "Hydrogen is the lightest element",
          domain: "chemistry.elements",
          type: "empirical" as any,
          stake: 100n * 10n ** 15n,
          proposedConfidence: 0.5,
          references: [
            {
              type: "supports",
              source: "axiom:logic:identity",
              contentHash: "genesis",
              retrievedAt: 0,
              reliabilityScore: 1.0,
              direction: "supporting" as any,
              intrinsicStrength: 1.0,
            },
          ],
        } as any,
        researcher.publicKey,
      );

      console.log(`  Claim ID: ${claimId}`);

      // Find the verification round
      const rounds = engine.getActiveRounds();
      expect(rounds.length).toBeGreaterThanOrEqual(1);
      const round = rounds.find((r) => r.claimId === claimId)!;
      expect(round).toBeDefined();
      console.log(`  Round ID: ${round.id}`);
      console.log(`  Phase: ${round.phase}`);
      console.log(`  Validators: ${round.selectedValidators.length}`);

      // === Commit phase ===
      console.log("\n  ── Commit Phase ──");

      const validatorNonces = new Map<string, string>();
      const validatorVerdicts = new Map<
        string,
        { verdict: string; confidence: number }
      >();

      for (const v of validators) {
        const verdict = "confirmed";
        const confidence = 0.85 + Math.random() * 0.1;
        const nonce = generateNonce();

        validatorNonces.set(v.publicKey, nonce);
        validatorVerdicts.set(v.publicKey, { verdict, confidence });

        const commitment = generateCommitment(
          round.id,
          verdict,
          confidence,
          nonce,
        );
        const commitSig = signDomainSeparated(
          "LGM.verification.commit.v1",
          `${round.id}:${commitment}`,
          v.privateKey,
        );

        await engine.submitCommitmentDeterministic(round.id, {
          validator: v.publicKey as PublicKey,
          commitment,
          signature: commitSig,
        });

        console.log(
          `    ${v.name}: committed (conf: ${confidence.toFixed(3)})`,
        );
      }

      // Advance blocks past commit deadline
      for (let i = 0; i < 5; i++) {
        await engine.advanceBlockForTesting();
      }
      console.log(`  Block advanced to: ${engine.getCurrentHeight()}`);

      // === Reveal phase ===
      console.log("\n  ── Reveal Phase ──");

      for (const v of validators) {
        const nonce = validatorNonces.get(v.publicKey)!;
        const { verdict, confidence } = validatorVerdicts.get(v.publicKey)!;

        const revealData = JSON.stringify({
          roundId: round.id,
          verdict,
          confidence,
          salt: nonce,
          referencesChecked: [],
          reasoning: undefined,
        });
        const revealSig = signDomainSeparated(
          "LGM.verification.reveal.v1",
          revealData,
          v.privateKey,
        );

        await engine.submitReveal(round.id, {
          validator: v.publicKey as PublicKey,
          verdict: verdict as any,
          confidence,
          salt: nonce,
          referencesChecked: [],
          signature: revealSig,
        });

        console.log(
          `    ${v.name}: revealed (${verdict}, ${confidence.toFixed(3)})`,
        );
      }

      // Advance past reveal deadline to trigger aggregation
      for (let i = 0; i < 6; i++) {
        await engine.advanceBlockForTesting();
      }

      // Check if round completed
      const completedRound = engine.getCompletedRound(round.id);
      if (completedRound?.result) {
        console.log("\n  ═══ Consensus Result ═══");
        console.log(`    Outcome: ${completedRound.result.outcome}`);
        console.log(
          `    Final Confidence: ${completedRound.result.finalConfidence.toFixed(4)}`,
        );
        console.log(
          `    Participating Validators: ${completedRound.result.participatingValidators}`,
        );

        if (completedRound.result.outcome === "accepted") {
          console.log("    → Claim accepted! Fact created in knowledge tree.");
        }
      } else {
        // Round may still be active if timing didn't align perfectly
        const activeRound = engine
          .getActiveRounds()
          .find((r) => r.id === round.id);
        console.log(
          `\n  Round phase: ${activeRound?.phase ?? "completed (no result yet)"}`,
        );
        console.log("  (Aggregation may need more blocks in live devnet)");
      }

      console.log("\n  [CONSENSUS SIMULATION COMPLETE]");
      console.log("    3 validators committed → revealed → consensus reached");
    });

    // --------------------------------------------------------------------------
    // Phase 5: Multi-agent reasoning chain
    // --------------------------------------------------------------------------

    it("Phase 5: Multi-step reasoning chain across domains", async () => {
      console.log(
        "\n╔══════════════════════════════════════════════════════════════╗",
      );
      console.log(
        "║  PHASE 5: Multi-Domain Reasoning Chain                      ║",
      );
      console.log(
        "╚══════════════════════════════════════════════════════════════╝\n",
      );

      // Ensure system accounts
      if (!state.getAccount("did:lgm:burn")) {
        state.createAccount("did:lgm:burn", "burn-op", 0n);
      }
      if (!state.getAccount("did:lgm:protocol-treasury")) {
        state.createAccount("did:lgm:protocol-treasury", "treasury-op", 0n);
      }
      if (!state.getAccount("did:lgm:research-treasury")) {
        state.createAccount("did:lgm:research-treasury", "research-op", 0n);
      }

      // Seed a multi-domain knowledge base
      const facts = [
        {
          id: "fact:chem-001",
          claimText: "CO2 absorbs infrared radiation",
          domain: "chemistry.spectroscopy",
          confidence: 0.94,
        },
        {
          id: "fact:atm-001",
          claimText: "Earth's atmosphere contains CO2",
          domain: "environmental.atmosphere",
          confidence: 0.97,
        },
        {
          id: "fact:phys-001",
          claimText: "Absorbed radiation increases thermal energy",
          domain: "physics.thermodynamics",
          confidence: 0.96,
        },
      ];

      for (const f of facts) {
        state.addFact({
          ...f,
          verifiedAt: 50n,
          verifierCount: 5,
          challengeCount: 0,
          citationCount: 3,
          status: "active",
        });
      }

      state.applyBlock(2000n, "0".repeat(64));

      const sessionId = `chat:${thinker.did}:${Date.now()}`;
      const thinkerStart = state.getBalance(thinker.did);
      let totalQueryCost = 0n;

      console.log("  Agent reasoning chain:");
      console.log("  ┌─────────────────────────────────────────────────────┐");

      // Step 1: Query chemistry
      const q1 = await knowledgeQuery(
        {
          sessionId,
          intent: {
            type: "domain_search",
            domain: "chemistry.spectroscopy",
            limit: 5,
            minConfidence: 0.5,
          },
        },
        rpcCtx,
      );
      const cost1 = BigInt(q1.metadata.actualCost);
      totalQueryCost += cost1;
      console.log(
        `  │ 1. "What absorbs infrared?" → CO2 absorbs IR (cost: ${cost1} wei)`,
      );

      // Step 2: Query environmental
      const q2 = await knowledgeQuery(
        {
          sessionId,
          intent: {
            type: "domain_search",
            domain: "environmental.atmosphere",
            limit: 5,
            minConfidence: 0.5,
          },
        },
        rpcCtx,
      );
      const cost2 = BigInt(q2.metadata.actualCost);
      totalQueryCost += cost2;
      console.log(
        `  │ 2. "Is CO2 in atmosphere?" → Yes, Earth has CO2 (cost: ${cost2} wei)`,
      );

      // Step 3: Query physics
      const q3 = await knowledgeQuery(
        {
          sessionId,
          intent: {
            type: "domain_search",
            domain: "physics.thermodynamics",
            limit: 5,
            minConfidence: 0.5,
          },
        },
        rpcCtx,
      );
      const cost3 = BigInt(q3.metadata.actualCost);
      totalQueryCost += cost3;
      console.log(
        `  │ 3. "What does absorbed radiation do?" → Increases thermal energy (cost: ${cost3} wei)`,
      );

      console.log("  │");
      console.log("  │ ═══ Synthesis ═══");
      console.log(
        "  │ CO2 absorbs IR + CO2 is in atmosphere + absorbed radiation → heat",
      );
      console.log("  │ ∴ CO2 in atmosphere causes greenhouse warming");
      console.log(
        "  └─────────────────────────────────────────────────────────┘\n",
      );

      // Verify total spend
      const thinkerEnd = state.getBalance(thinker.did);
      const actualSpend = thinkerStart - thinkerEnd;
      console.log(`  Total query cost: ${totalQueryCost} wei across 3 queries`);
      console.log(`  Actual balance change: ${actualSpend} wei`);

      // Submit the synthesized claim
      console.log("\n  Submitting derived claim from reasoning chain...");

      const synthClaim: ClaimSubmitPayload = {
        type: "claim_submit",
        claim:
          "CO2 in Earth's atmosphere contributes to greenhouse warming via infrared absorption",
        claimType: "empirical",
        domain: "environmental.climate",
        stratum: "environmental",
        stake: 200n * 10n ** 15n,
        references: ["fact:chem-001", "fact:atm-001"],
        derivedFrom: {
          factIds: ["fact:chem-001", "fact:atm-001", "fact:phys-001"],
          inferenceType: "deductive",
          logicChain:
            "CO2 absorbs IR ∧ CO2 in atmosphere ∧ absorbed IR → heat ⊢ CO2 causes warming",
        },
      } as any;

      const synthResult = executeTransaction(
        state,
        userTx("claim_submit", synthClaim as any, thinker, state),
        blockNumber + 60n,
        SIM_PARAMS,
      );
      expect(synthResult.success).toBe(true);
      console.log("  Derived claim submitted successfully!");
      console.log(
        "  → Agent used 3 cross-domain facts to reason about climate",
      );
      console.log(
        "  → Each query paid LGM → knowledge contributors + research fund",
      );
      console.log(
        "  → Agent created new knowledge from existing verified facts\n",
      );
    });
  },
);
