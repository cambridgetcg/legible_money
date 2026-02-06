/**
 * Proof of Truth Consensus Engine
 *
 * The main consensus engine that coordinates:
 * - Block production
 * - Claim verification rounds
 * - Validator selection
 * - Finality tracking
 *
 * "The work is verifying truth. The product is verified knowledge."
 */

import { EventEmitter } from "events";
import { sha512_256Hex } from "../../core/crypto/hashing.js";
import { sign, verify } from "../../core/crypto/signatures.js";
import type { PublicKey, PrivateKey } from "../../core/crypto/keys.js";
import type { ClaimSubmission } from "../../core/knowledge/types.js";
import {
  getSchemaRegistry,
  type ClaimSchema,
} from "../../core/knowledge/schemas/index.js";
import type { DomainTree } from "../../core/knowledge/ontology/types.js";
import {
  validateClaimOntology,
  type ClaimOntologyResult,
} from "../../core/knowledge/ontology/claim_validation.js";
import { createGenesisDomainTree } from "../../core/knowledge/ontology/index.js";
import { analyzeGoedelianStatus } from "../../core/logic/goedel.js";

import {
  ConsensusParams,
  DEFAULT_CONSENSUS_PARAMS,
  ConsensusState,
  ConsensusEvent,
  Block,
  BlockHeader,
  Transaction,
  VerificationRound,
  VerificationResult,
  ValidatorSet,
} from "./types.js";

import {
  selectValidatorsForVerification,
  canProposeBlock,
  verifyBlockProposer,
  createValidatorSet,
  slashValidatorByAmount,
  shouldChangeEpoch,
  computeNextEpochValidators,
} from "./validator_selection.js";

import {
  createVerificationRound,
  startRound,
  addCommitmentWithVRF,
  transitionToReveal,
  addReveal,
  aggregateResults,
  getCurrentPhase,
} from "./verification_round.js";

import { shouldCreateCheckpoint, type Checkpoint } from "./finality.js";

// GAP-008/009 FIX: Import checkpoint ceremony for proper BFT signature collection
import {
  CheckpointCeremonyManager,
  createPrevote,
  createPrecommit,
  type CheckpointCeremony,
} from "./checkpoint_ceremony.js";

// GAP-003/004/005/006/007 FIX: Import state integration
import {
  type StateIntegration,
  createStateIntegration,
} from "./state_integration.js";
import type { StateManager } from "../../state/state_manager.js";

// Discovery engine integration
import type { DiscoveryEngine } from "../../core/discovery/engine.js";

// VULN-011 FIX: Import equivocation detection
import {
  EquivocationDetector,
  calculateEquivocationSlash,
  type EquivocationEvidence,
} from "./equivocation.js";

// AUTONOMY: Import autonomy integration for agent self-execution
import { AutonomyIntegration } from "./autonomy_integration.js";

// ============ Engine Configuration ============

export interface EngineConfig {
  params: ConsensusParams;
  validatorKey?: PrivateKey;
  validatorPublicKey?: PublicKey;
  isValidator: boolean;
  // GAP FIX: Optional state integration
  stateIntegration?: StateIntegration;
}

// ============ Consensus Engine ============

export class ProofOfTruthEngine extends EventEmitter {
  private config: EngineConfig;
  private state: ConsensusState;
  private checkpoints: Checkpoint[] = [];
  private running: boolean = false;
  private blockTimer: NodeJS.Timeout | null = null;
  // GAP FIX: State integration layer
  private stateIntegration?: StateIntegration;
  // GAP-008 FIX: Checkpoint ceremony manager for BFT signature collection
  private checkpointCeremonyManager: CheckpointCeremonyManager;
  // VULN-011 FIX: Equivocation detector for slashing double-signers
  private equivocationDetector: EquivocationDetector;
  // Ontological tree for claim validation
  private domainTree: DomainTree;
  // Discovery engine integration
  private discoveryEngine?: DiscoveryEngine;
  // AUTONOMY: Agent autonomy integration for self-executing schedules
  private autonomyIntegration?: AutonomyIntegration;

  constructor(config: Partial<EngineConfig> = {}) {
    super();
    this.config = {
      params: DEFAULT_CONSENSUS_PARAMS,
      isValidator: false,
      ...config,
    };
    this.state = this.initState();
    // GAP FIX: Store state integration if provided
    this.stateIntegration = config.stateIntegration;
    // GAP-008 FIX: Initialize checkpoint ceremony manager
    this.checkpointCeremonyManager = new CheckpointCeremonyManager();
    // VULN-011 FIX: Initialize equivocation detector
    this.equivocationDetector = new EquivocationDetector();
    // Initialize ontological domain tree with genesis domains
    this.domainTree = createGenesisDomainTree();
  }

  /**
   * Get the ontological domain tree
   */
  getDomainTree(): DomainTree {
    return this.domainTree;
  }

  /**
   * Set a custom domain tree (for testing or extended ontologies)
   */
  setDomainTree(tree: DomainTree): void {
    this.domainTree = tree;
  }

  /**
   * Set discovery engine
   */
  setDiscoveryEngine(engine: DiscoveryEngine): void {
    this.discoveryEngine = engine;
  }

  /**
   * Get discovery engine (if configured)
   */
  getDiscoveryEngine(): DiscoveryEngine | undefined {
    return this.discoveryEngine;
  }

  /**
   * Set state integration (can be called after construction)
   */
  setStateIntegration(integration: StateIntegration): void {
    this.stateIntegration = integration;
  }

  /**
   * Check if state integration is available
   */
  hasStateIntegration(): boolean {
    return this.stateIntegration !== undefined;
  }

  /**
   * Get the state integration layer
   * GAP FIX: Needed for node to access mempool and state operations
   */
  getStateIntegration(): StateIntegration | undefined {
    return this.stateIntegration;
  }

  /**
   * AUTONOMY: Set up autonomy integration for agent self-execution
   * This enables agents to schedule themselves and execute autonomously.
   */
  setAutonomyIntegration(autonomy: AutonomyIntegration): void {
    this.autonomyIntegration = autonomy;
  }

  /**
   * AUTONOMY: Get autonomy integration (if configured)
   */
  getAutonomyIntegration(): AutonomyIntegration | undefined {
    return this.autonomyIntegration;
  }

  /**
   * AUTONOMY: Check if autonomy is enabled
   */
  hasAutonomy(): boolean {
    return this.autonomyIntegration !== undefined;
  }

  /**
   * AUTONOMY: Register a schedule from SCHEDULE_SELF opcode execution
   *
   * Called when BVM execution produces a schedule_create state change.
   * This bridges BVM execution to the SPU for agent autonomy.
   *
   * Integration path (when full BVM is integrated):
   * 1. contract_call transaction arrives
   * 2. BVM executes contract code
   * 3. SCHEDULE_SELF opcode emits schedule_create state change
   * 4. This method is called with the state change
   * 5. SPU registers the schedule
   * 6. Future blocks process the schedule via processSchedulesForBlock()
   */
  async registerScheduleFromBVM(
    creator: string,
    targetAddress: string,
    functionSelector: bigint,
    condition: bigint,
    interval: bigint,
    prepaidFee: bigint,
  ): Promise<string | null> {
    if (!this.autonomyIntegration) {
      console.warn(
        "[PoT] AUTONOMY: Cannot register schedule - autonomy not enabled",
      );
      return null;
    }

    try {
      const scheduleId =
        await this.autonomyIntegration.registerScheduleFromOpcode(
          creator,
          targetAddress,
          functionSelector,
          condition,
          interval,
          prepaidFee,
        );
      console.log(
        `[PoT] AUTONOMY: Registered schedule ${scheduleId} for ${targetAddress}`,
      );
      return scheduleId;
    } catch (error) {
      console.error("[PoT] AUTONOMY: Failed to register schedule:", error);
      return null;
    }
  }

  /**
   * AUTONOMY: Process schedule state changes from BVM execution
   *
   * Called after a block is applied to register any new schedules
   * that were created via SCHEDULE_SELF opcodes.
   */
  async processScheduleStateChanges(
    stateChanges: Array<{
      type: string;
      address: string;
      key: string;
      newValue: unknown;
    }>,
  ): Promise<string[]> {
    if (!this.autonomyIntegration) {
      return [];
    }

    return this.autonomyIntegration.processScheduleStateChanges(stateChanges);
  }

  // ============ Lifecycle ============

  /**
   * Start the consensus engine
   */
  async start(): Promise<void> {
    if (this.running) return;

    this.running = true;
    this.emit("started");

    // Start block production timer
    this.startBlockTimer();

    console.log("[PoT] Consensus engine started");
  }

  /**
   * Stop the consensus engine
   */
  async stop(): Promise<void> {
    if (!this.running) return;

    this.running = false;
    if (this.blockTimer) {
      clearInterval(this.blockTimer);
      this.blockTimer = null;
    }

    this.emit("stopped");
    console.log("[PoT] Consensus engine stopped");
  }

  /**
   * Get current state
   */
  getState(): ConsensusState {
    return { ...this.state };
  }

  /**
   * Get active verification rounds
   */
  getActiveRounds(): VerificationRound[] {
    return Array.from(this.state.activeRounds.values());
  }

  /**
   * Set the validator set
   */
  setValidatorSet(validatorSet: ValidatorSet): void {
    this.state.validatorSet = validatorSet;
  }

  /**
   * Advance the block number for testing
   *
   * ⚠️  TEST ONLY - DO NOT USE IN PRODUCTION ⚠️
   *
   * This method manually advances the block number without going through
   * the full block production and validation flow. Used for deterministic
   * testing where VRF selection would be probabilistic.
   *
   * Returns the block number after advancement.
   */
  async advanceBlockForTesting(): Promise<bigint> {
    this.state.currentBlock++;

    // Update block hash
    this.state.lastBlockHash = sha512_256Hex(
      `test-block-${this.state.currentBlock}-${Date.now()}`,
    );
    this.state.lastBlockTime = Date.now();

    // Update finality tracking
    const confirmationBlocks = this.config.params.confirmationBlocks ?? 5;
    const newFinalized = this.state.currentBlock - BigInt(confirmationBlocks);
    if (newFinalized > this.state.lastFinalizedBlock) {
      this.state.lastFinalizedBlock = newFinalized;
      this.state.lastFinalizedHash = sha512_256Hex(`finalized-${newFinalized}`);

      this.emitEvent({
        type: "block_finalized",
        blockNumber: newFinalized,
        blockHash: this.state.lastFinalizedHash,
      });
    }

    // Process verification rounds - handle phase transitions and aggregation
    // This applies results to state, creating facts for accepted claims
    const validatorStakes = this.getValidatorStakes();

    for (const [roundId, round] of this.state.activeRounds) {
      let updatedRound = round;
      const currentBlock = Number(this.state.currentBlock);

      // First, transition to reveal phase if needed
      if (round.phase === "commit" && currentBlock > round.commitDeadline) {
        updatedRound = transitionToReveal(round, currentBlock);
        this.state.activeRounds.set(roundId, updatedRound);
      }

      // Then check if we should aggregate
      const phase = getCurrentPhase(updatedRound, currentBlock);

      if (phase === "aggregation" || phase === "complete") {
        // Aggregate results (requires round.phase === 'reveal')
        const aggregated = aggregateResults(
          updatedRound,
          validatorStakes,
          currentBlock,
          this.config.params,
        );

        if (aggregated.result) {
          // CRITICAL: Apply verification result to state (creates fact)
          await this.applyVerificationResult(aggregated.result);

          // Emit event
          this.emitEvent({
            type: "round_completed",
            result: aggregated.result,
          });

          // Remove completed round
          this.state.activeRounds.delete(roundId);
        } else {
          this.state.activeRounds.set(roundId, aggregated);
        }
      }
    }

    return this.state.currentBlock;
  }

  // ============ Block Production ============

  /**
   * Try to produce a block
   */
  async tryProduceBlock(): Promise<Block | null> {
    if (!this.config.isValidator || !this.config.validatorKey) {
      return null;
    }

    const ourStake = this.getOurStake();
    const totalStake = this.state.validatorSet.totalStake;

    // targetProposers = validatorSet size ensures every validator can propose
    // (highest VRF priority wins tie-breaking when multiple propose)
    const targetProposers = Math.max(
      1,
      this.state.validatorSet.validators.size,
    );
    const selection = canProposeBlock(
      this.config.validatorKey,
      this.state.currentBlock + 1n,
      this.state.currentEpoch,
      this.state.lastBlockHash,
      ourStake,
      totalStake,
      targetProposers,
    );

    if (!selection.canPropose) {
      return null;
    }

    // Build the block
    const block = await this.buildBlock(
      selection.vrfOutput,
      selection.vrfProof,
    );

    // Sign and broadcast
    const signedBlock = this.signBlock(block);

    // Apply locally
    await this.applyBlock(signedBlock);

    // Emit event
    this.emitEvent({ type: "block_proposed", block: signedBlock });

    return signedBlock;
  }

  /**
   * Build a new block
   */
  private async buildBlock(
    vrfOutput: string,
    vrfProof: string,
  ): Promise<Block> {
    const blockNumber = this.state.currentBlock + 1n;
    const timestamp = Date.now();

    // Collect transactions from mempool
    const mempoolTransactions = this.collectTransactions();

    // AUTONOMY: Process scheduled agent actions
    // This is the TRIGGER that makes agents truly autonomous
    let scheduledTransactions: Transaction[] = [];
    if (
      this.autonomyIntegration &&
      this.config.validatorKey &&
      this.config.validatorPublicKey
    ) {
      const autonomyResult =
        await this.autonomyIntegration.processSchedulesForBlock(
          blockNumber,
          timestamp,
          this.config.validatorKey,
          this.config.validatorPublicKey,
        );
      scheduledTransactions = autonomyResult.transactions;

      if (autonomyResult.executions.length > 0) {
        console.log(
          `[PoT] AUTONOMY: Processed ${autonomyResult.executions.length} scheduled agent actions`,
        );
      }
    }

    // Combine mempool + scheduled transactions
    const transactions = [...mempoolTransactions, ...scheduledTransactions];

    // Process verification rounds that are complete
    const verificationResults = this.collectVerificationResults();

    // Build header
    const header: BlockHeader = {
      number: blockNumber,
      timestamp,
      previousHash: this.state.lastBlockHash,
      stateRoot: this.computeStateRoot(),
      knowledgeRoot: this.computeKnowledgeRoot(verificationResults),
      transactionsRoot: this.computeTransactionsRoot(transactions),
      verificationsRoot: this.computeVerificationsRoot(verificationResults),
      proposer: this.config.validatorPublicKey!,
      proposerVrfOutput: vrfOutput,
      proposerVrfProof: vrfProof,
    };

    return {
      header,
      transactions,
      verificationResults,
      signature: "", // Will be signed
      attestations: [],
    };
  }

  /**
   * Sign a block
   */
  private signBlock(block: Block): Block {
    const headerHash = this.hashBlockHeader(block.header);
    const signature = sign(
      new TextEncoder().encode(headerHash),
      this.config.validatorKey!,
    );

    return { ...block, signature };
  }

  /**
   * Apply a block to state
   */
  async applyBlock(block: Block): Promise<void> {
    // Validate block
    const validation = await this.validateBlock(block);
    if (!validation.valid) {
      throw new Error(`Invalid block: ${validation.reason}`);
    }

    // Apply transactions
    for (const tx of block.transactions) {
      await this.applyTransaction(tx);
    }

    // Apply verification results
    for (const result of block.verificationResults) {
      await this.applyVerificationResult(result);
    }

    // Update consensus state
    this.state.currentBlock = block.header.number;
    const blockHash = this.hashBlockHeader(block.header);
    this.state.lastBlockHash = blockHash;
    this.state.lastBlockTime = block.header.timestamp;

    // Sync block number to state manager so RPC reports correct height
    if (this.stateIntegration) {
      try {
        this.stateIntegration
          .getStateManager()
          .applyBlock(block.header.number, blockHash);
      } catch {
        // State manager update is best-effort (may fail if in transaction)
      }
    }

    // Check for epoch change
    if (
      shouldChangeEpoch(block.header.number, this.config.params.epochLength)
    ) {
      await this.handleEpochChange();
    }

    // Check for checkpoint
    if (
      shouldCreateCheckpoint(block.header.number, this.getLastCheckpointBlock())
    ) {
      await this.createNewCheckpoint(block);
    }

    // GAP-008 FIX: Process ongoing checkpoint ceremonies
    this.processCheckpointCeremonies();

    // Update finality
    this.updateFinality();

    // Process discovery engine
    if (this.discoveryEngine) {
      await this.discoveryEngine.processBlock(block.header.number);
    }

    // Keep recent blocks
    this.state.recentBlocks.push(block);
    if (this.state.recentBlocks.length > 100) {
      this.state.recentBlocks.shift();
    }
  }

  /**
   * Validate a block
   */
  async validateBlock(
    block: Block,
  ): Promise<{ valid: boolean; reason?: string }> {
    // Check block number
    if (block.header.number !== this.state.currentBlock + 1n) {
      return { valid: false, reason: "Invalid block number" };
    }

    // Check previous hash
    if (block.header.previousHash !== this.state.lastBlockHash) {
      return { valid: false, reason: "Invalid previous hash" };
    }

    // Verify proposer selection
    const proposerValidator = this.state.validatorSet.validators.get(
      block.header.proposer,
    );
    if (!proposerValidator) {
      return { valid: false, reason: "Unknown proposer" };
    }

    const targetProposers = Math.max(
      1,
      this.state.validatorSet.validators.size,
    );
    const proposerValid = verifyBlockProposer(
      block.header.proposer,
      block.header.proposerVrfOutput,
      block.header.proposerVrfProof,
      block.header.number,
      this.state.currentEpoch,
      this.state.lastBlockHash,
      proposerValidator.stake,
      this.state.validatorSet.totalStake,
      targetProposers,
    );

    if (!proposerValid.valid) {
      return {
        valid: false,
        reason: `Invalid proposer: ${proposerValid.reason}`,
      };
    }

    // Verify block signature
    const headerHash = this.hashBlockHeader(block.header);
    const sigValid = verify(
      new TextEncoder().encode(headerHash),
      block.signature,
      block.header.proposer,
    );

    if (!sigValid) {
      return { valid: false, reason: "Invalid block signature" };
    }

    // VULN-011 FIX: Check for equivocation (double block proposal)
    const equivocationResult = this.equivocationDetector.checkBlock(block);
    if (equivocationResult.detected && equivocationResult.evidence) {
      // Slash the equivocating validator
      await this.handleEquivocation(equivocationResult.evidence);
      // Still reject the duplicate block
      return {
        valid: false,
        reason: "Equivocation detected - validator slashed",
      };
    }

    // Validate transactions
    for (const tx of block.transactions) {
      const txValid = await this.validateTransaction(tx);
      if (!txValid.valid) {
        return {
          valid: false,
          reason: `Invalid transaction: ${txValid.reason}`,
        };
      }
    }

    return { valid: true };
  }

  // ============ Verification Rounds ============

  /**
   * Submit a new claim for verification
   *
   * Supports both schema-based and legacy free-form claims.
   * Schema-based claims get:
   * - Validated against schema.dataSchema
   * - Dynamic timing based on schema.timing
   * - Clear verification method guidance
   *
   * VULN-014 FIX: Accept optional submitter for unique claim ID generation
   */
  async submitClaim(
    claim: ClaimSubmission,
    submitter?: string,
  ): Promise<string> {
    // Get schema if claim uses one
    let schema: ClaimSchema | undefined;
    if (claim.schemaId) {
      const registry = getSchemaRegistry();
      schema = registry.get(claim.schemaId);

      if (!schema) {
        throw new Error(`Unknown schema: ${claim.schemaId}`);
      }

      if (!schema.active) {
        throw new Error(`Schema is inactive: ${claim.schemaId}`);
      }

      // Validate claim data against schema
      if (claim.claimData) {
        const validation = registry.validate(claim.schemaId, claim.claimData);
        if (!validation.valid) {
          const errorMessages = validation.errors
            .map((e) => `${e.path}: ${e.message}`)
            .join("; ");
          throw new Error(`Schema validation failed: ${errorMessages}`);
        }
      } else {
        throw new Error("Schema-based claims require claimData");
      }

      // Check required references
      if (
        schema.minReferences &&
        claim.references.length < schema.minReferences
      ) {
        throw new Error(
          `Schema requires at least ${schema.minReferences} references`,
        );
      }

      if (schema.requiredReferenceTypes) {
        const providedTypes = new Set(claim.references.map((r) => r.type));
        const missingTypes = schema.requiredReferenceTypes.filter(
          (t) => !providedTypes.has(t as any),
        );
        if (missingTypes.length > 0) {
          throw new Error(
            `Schema requires reference types: ${missingTypes.join(", ")}`,
          );
        }
      }
    } else {
      // Legacy free-form claim validation
      if (!claim.claim || claim.claim.length === 0) {
        throw new Error(
          "Empty claim (use schemaId + claimData for structured claims)",
        );
      }
    }

    if (claim.references.length === 0) {
      throw new Error("Claim must have at least one reference");
    }

    // Validate claim against ontological tree
    const ontologyResult = validateClaimOntology(
      {
        domain: claim.domain,
        schemaId: claim.schemaId,
        proposedConfidence: claim.proposedConfidence,
        references: claim.references.map((r) => ({
          targetFactId: r.source ?? "unknown",
          targetDomain: claim.domain, // Use claim domain as default for references
          referenceType: r.type as
            | "supports"
            | "contradicts"
            | "extends"
            | "specializes"
            | "generalizes",
        })),
      },
      this.domainTree,
    );

    if (!ontologyResult.valid) {
      throw new Error(
        `Ontology validation failed: ${ontologyResult.errors.join(", ")}`,
      );
    }

    // Log warnings (don't block, but inform)
    for (const warning of ontologyResult.warnings) {
      console.log(`[PoT] Ontology warning for claim: ${warning}`);
    }

    // U1 FIX: Gödelian analysis — reject paradoxical claims before creating round
    if (claim.claim) {
      const goedelResult = analyzeGoedelianStatus(claim.claim);
      if (goedelResult.paradoxical) {
        throw new Error(
          `Claim rejected: contains logical paradox — ${goedelResult.details}`,
        );
      }
    }

    // Add to pending claims
    this.state.pendingClaims.push(claim);

    // Create verification round with schema timing and ontology context
    const round = this.createRoundForClaim(
      claim,
      submitter,
      schema,
      ontologyResult,
    );

    // Store round
    this.state.activeRounds.set(round.id, round);

    // Emit event
    this.emitEvent({ type: "round_started", round });

    // Return claim ID (what facts are indexed by), not round ID
    return round.claimId;
  }

  /**
   * Create verification round for a claim
   *
   * Uses schema timing if available for dynamic round duration.
   * Includes ontological context for stratum-aware validation.
   *
   * VULN-014 FIX: Accept submitter for unique claim ID generation
   */
  private createRoundForClaim(
    claim: ClaimSubmission,
    submitter?: string,
    schema?: ClaimSchema,
    ontologyResult?: ClaimOntologyResult,
  ): VerificationRound {
    // Select validators (serialize BigInt as string for hashing)
    const claimForHash = JSON.stringify(claim, (_, v) =>
      typeof v === "bigint" ? v.toString() : v,
    );
    const selectedValidators = selectValidatorsForVerification(
      sha512_256Hex(claimForHash),
      this.state.currentBlock,
      this.state.lastBlockHash,
      this.state.validatorSet,
      this.config.params,
    );

    // Create round with schema timing (if available)
    // VULN-014 FIX: Pass submitter for unique claim ID
    const round = createVerificationRound(
      claim,
      selectedValidators,
      Number(this.state.currentBlock),
      this.config.params,
      submitter,
      schema?.timing, // Dynamic timing from schema
    );

    // Add ontological context to round
    if (ontologyResult) {
      round.resolvedDomain = ontologyResult.resolvedDomain ?? undefined;
      round.resolvedStratum = ontologyResult.resolvedDomain?.stratum;
      round.adjustedConfidence = ontologyResult.adjustedConfidence;
      round.ontologyValidated = ontologyResult.valid;
    }

    return startRound(round);
  }

  /**
   * Submit a verification commitment with VRF proof
   * SECURITY: Requires VRF proof to verify validator is legitimately selected
   */
  async submitCommitment(
    roundId: string,
    commitment: {
      validator: PublicKey;
      commitment: string;
      signature: string;
      vrfOutput: string;
      vrfProof: string;
    },
  ): Promise<void> {
    const round = this.state.activeRounds.get(roundId);
    if (!round) {
      throw new Error("Round not found");
    }

    // Get validator stake for VRF verification
    const validator = this.state.validatorSet.validators.get(
      commitment.validator,
    );
    if (!validator) {
      throw new Error("Unknown validator");
    }
    const validatorStake = validator.stake + validator.delegatedStake;

    // SECURITY: Use addCommitmentWithVRF to verify the validator is legitimately selected
    const result = addCommitmentWithVRF(
      round,
      {
        validator: commitment.validator,
        commitment: commitment.commitment,
        submittedAt: Date.now(),
        signature: commitment.signature,
        vrfOutput: commitment.vrfOutput,
        vrfProof: commitment.vrfProof,
      },
      Number(this.state.currentBlock),
      this.state.lastBlockHash,
      validatorStake,
      this.state.validatorSet.totalStake,
      this.config.params,
    );

    if (!result.accepted) {
      throw new Error(result.reason);
    }

    this.state.activeRounds.set(roundId, result.round);
  }

  /**
   * Submit a verification commitment WITHOUT VRF verification
   *
   * ⚠️  TEST ONLY - DO NOT USE IN PRODUCTION ⚠️
   *
   * This method bypasses VRF-based validator selection verification.
   * It exists solely for deterministic testing where VRF's probabilistic
   * nature would cause flaky tests.
   *
   * In production, always use submitCommitment() which properly verifies
   * VRF proofs to ensure validators are legitimately selected.
   *
   * SECURITY: This method will throw in production mode (NODE_ENV=production).
   */
  async submitCommitmentDeterministic(
    roundId: string,
    commitment: {
      validator: PublicKey;
      commitment: string;
      signature: string;
    },
  ): Promise<void> {
    // SECURITY: Block in production mode
    if (process.env.NODE_ENV === "production") {
      throw new Error(
        "submitCommitmentDeterministic is disabled in production mode - use submitCommitment() instead",
      );
    }

    const round = this.state.activeRounds.get(roundId);
    if (!round) {
      throw new Error("Round not found");
    }

    // Check phase
    if (round.phase !== "commit") {
      throw new Error("Not in commit phase");
    }

    // Check deadline
    if (Number(this.state.currentBlock) > round.commitDeadline) {
      throw new Error("Commit deadline passed");
    }

    // Check not already committed
    if (round.commitments.has(commitment.validator)) {
      throw new Error("Already committed");
    }

    // Import domain-separated verification (matches generateCommitment signing)
    const { verifyDomainSeparated } =
      await import("../../core/crypto/signatures.js");

    // Verify signature using domain separation (matches generateCommitment format)
    const sigValid = verifyDomainSeparated(
      "LGM.verification.commit.v1",
      `${round.id}:${commitment.commitment}`,
      commitment.signature,
      commitment.validator,
    );
    if (!sigValid) {
      throw new Error("Invalid signature");
    }

    // Add commitment directly (bypassing VRF)
    const newCommitments = new Map(round.commitments);
    newCommitments.set(commitment.validator, {
      validator: commitment.validator,
      commitment: commitment.commitment,
      submittedAt: Date.now(),
      signature: commitment.signature,
    });

    this.state.activeRounds.set(roundId, {
      ...round,
      commitments: newCommitments,
    });
  }

  /**
   * Submit a verification reveal
   */
  async submitReveal(
    roundId: string,
    reveal: {
      validator: PublicKey;
      verdict: "confirmed" | "rejected" | "uncertain";
      confidence: number;
      nonce: string;
      referencesChecked: string[];
      signature: string;
    },
  ): Promise<void> {
    let round = this.state.activeRounds.get(roundId);
    if (!round) {
      throw new Error("Round not found");
    }

    // Transition to reveal phase if needed
    round = transitionToReveal(round, Number(this.state.currentBlock));

    const result = addReveal(
      round,
      {
        validator: reveal.validator,
        verdict: reveal.verdict,
        confidence: reveal.confidence,
        nonce: reveal.nonce,
        referencesChecked: reveal.referencesChecked,
        submittedAt: Date.now(),
        signature: reveal.signature,
      },
      Number(this.state.currentBlock),
    );

    if (!result.accepted) {
      throw new Error(result.reason);
    }

    this.state.activeRounds.set(roundId, result.round);
  }

  /**
   * Process verification rounds and collect results
   */
  private collectVerificationResults(): VerificationResult[] {
    const results: VerificationResult[] = [];
    const validatorStakes = this.getValidatorStakes();

    for (const [roundId, round] of this.state.activeRounds) {
      const phase = getCurrentPhase(round, Number(this.state.currentBlock));

      if (phase === "aggregation" || phase === "complete") {
        // Aggregate results
        const aggregated = aggregateResults(
          round,
          validatorStakes,
          Number(this.state.currentBlock),
          this.config.params,
        );

        if (aggregated.result) {
          results.push(aggregated.result);

          // Emit event
          this.emitEvent({
            type: "round_completed",
            result: aggregated.result,
          });

          // Remove completed round
          this.state.activeRounds.delete(roundId);
        } else {
          this.state.activeRounds.set(roundId, aggregated);
        }
      }
    }

    return results;
  }

  /**
   * GAP-004 FIX: Apply a verification result through state integration
   */
  private async applyVerificationResult(
    result: VerificationResult,
  ): Promise<void> {
    // Use state integration for full state updates
    if (this.stateIntegration) {
      await this.stateIntegration.applyVerificationResult(
        result,
        this.state.currentBlock,
      );

      // Emit events for monitoring
      for (const [validatorKey, slashAmount] of result.slashes) {
        this.emitEvent({
          type: "validator_slashed",
          validator: validatorKey as PublicKey,
          amount: slashAmount,
          reason: "verification_failure",
        });
      }

      if (result.outcome === "accepted") {
        this.emitEvent({
          type: "fact_created",
          factId: result.claimId,
          confidence: result.finalConfidence,
        });
      }
      return;
    }

    // Fallback: update local validator set only (for tests)
    // VULN-010 FIX: Use slashValidatorByAmount to avoid float precision loss
    for (const [validatorKey, slashAmount] of result.slashes) {
      const validator = this.state.validatorSet.validators.get(validatorKey);
      if (validator) {
        const { validator: slashed, slashedAmount } = slashValidatorByAmount(
          validator,
          slashAmount, // Use bigint directly - no float conversion
          "verification_failure",
        );
        this.state.validatorSet.validators.set(validatorKey, slashed);

        this.emitEvent({
          type: "validator_slashed",
          validator: validatorKey as PublicKey,
          amount: slashedAmount,
          reason: "verification_failure",
        });
      }
    }
  }

  // ============ Transaction Processing ============

  /**
   * GAP-003 FIX: Collect transactions from mempool
   */
  private collectTransactions(): Transaction[] {
    // Use state integration if available
    if (this.stateIntegration) {
      const maxTxPerBlock = this.config.params.maxTransactionsPerBlock || 100;
      return this.stateIntegration.collectTransactions(maxTxPerBlock);
    }
    // Fallback for tests without state integration
    return [];
  }

  /**
   * GAP-006 FIX: Validate a transaction against actual state
   * SECURITY: Signature now covers full transaction (type, sender, nonce, fee, payload)
   */
  private async validateTransaction(
    tx: Transaction,
  ): Promise<{ valid: boolean; reason?: string }> {
    // SECURITY: Verify signature covers FULL transaction, not just payload
    // This prevents transaction malleability attacks where fee/nonce could be modified
    const signedData = this.getTransactionSigningData(tx);
    const sigValid = verify(
      new TextEncoder().encode(signedData),
      tx.signature,
      tx.sender,
    );

    if (!sigValid) {
      return {
        valid: false,
        reason:
          "Invalid signature - must sign full transaction (type, sender, nonce, fee, payload)",
      };
    }

    // Use state integration for full validation
    if (this.stateIntegration) {
      const validation = await this.stateIntegration.validateTransaction(tx);
      return { valid: validation.valid, reason: validation.reason };
    }

    // SECURITY: Fail-safe when no state integration - cannot validate state
    // This prevents transactions from being accepted without proper validation
    return {
      valid: false,
      reason:
        "State integration not available - cannot validate transaction state",
    };
  }

  /**
   * Get the canonical data that must be signed for a transaction
   * SECURITY: Includes ALL fields to prevent malleability attacks
   */
  private getTransactionSigningData(tx: Transaction): string {
    return JSON.stringify({
      type: tx.type,
      sender: tx.sender,
      nonce: tx.nonce.toString(),
      fee: tx.fee.toString(),
      payload: tx.payload,
    });
  }

  /**
   * GAP-005 FIX: Apply a transaction through state integration
   */
  private async applyTransaction(tx: Transaction): Promise<void> {
    // Use state integration if available
    if (this.stateIntegration) {
      const result = await this.stateIntegration.executeTransaction(
        tx,
        this.state.currentBlock,
      );

      if (!result.success) {
        throw new Error(
          `Transaction failed: ${result.error || "Unknown error"}`,
        );
      }

      // Handle consensus-specific side effects
      switch (tx.type) {
        case "claim_submit":
          // Start verification round (handled by engine, not state)
          await this.handleClaimSubmission(tx);
          break;
        case "verification_commit":
          // Add commitment (handled by engine)
          await this.handleVerificationCommit(tx);
          break;
        case "verification_reveal":
          // Add reveal (handled by engine)
          await this.handleVerificationReveal(tx);
          break;
      }
      return;
    }

    // Fallback: handle consensus-specific transactions only
    switch (tx.type) {
      case "claim_submit":
        await this.handleClaimSubmission(tx);
        break;
      case "verification_commit":
        await this.handleVerificationCommit(tx);
        break;
      case "verification_reveal":
        await this.handleVerificationReveal(tx);
        break;
    }
  }

  /**
   * Handle claim submission - start verification round
   *
   * Gets schema timing for dynamic round duration.
   *
   * VULN-014 FIX: Pass submitter to createVerificationRound for unique claim ID
   */
  private async handleClaimSubmission(tx: Transaction): Promise<void> {
    // Type-safe conversion: TransactionPayload has different shape than ClaimSubmission
    // The payload contains claim_submit fields which map to ClaimSubmission
    const claim = tx.payload as unknown as ClaimSubmission;

    // Get schema for timing (if claim uses one)
    let schemaTiming:
      | import("../../core/knowledge/schemas/types.js").SchemaTiming
      | undefined;
    if (claim.schemaId) {
      const registry = getSchemaRegistry();
      const schema = registry.get(claim.schemaId);
      if (schema) {
        schemaTiming = schema.timing;
      }
    }

    // Use claim text or generate from schema data for validator selection hash
    const claimForHash =
      claim.schemaId && claim.claimData
        ? JSON.stringify(
            { schemaId: claim.schemaId, data: claim.claimData },
            (_, v) => (typeof v === "bigint" ? v.toString() : v),
          )
        : claim.claim;

    const selectedValidators = selectValidatorsForVerification(
      claimForHash,
      this.state.currentBlock,
      this.state.lastBlockHash,
      this.state.validatorSet,
      this.config.params,
    );

    const round = createVerificationRound(
      claim,
      selectedValidators,
      Number(this.state.currentBlock),
      this.config.params,
      tx.sender, // VULN-014 FIX: Include submitter for unique claim ID
      schemaTiming, // Schema-based dynamic timing
    );

    this.state.activeRounds.set(round.id, startRound(round));

    this.emitEvent({
      type: "verification_round_started",
      roundId: round.id,
      claimId: round.claimId,
    });
  }

  /**
   * Handle verification commit
   */
  private async handleVerificationCommit(tx: Transaction): Promise<void> {
    const payload = tx.payload as {
      roundId: string;
      commitment: string;
      vrfOutput?: string;
      vrfProof?: string;
    };

    const round = this.state.activeRounds.get(payload.roundId);
    if (!round) return;

    // Commitment handled in block processing
  }

  /**
   * Handle verification reveal
   */
  private async handleVerificationReveal(_tx: Transaction): Promise<void> {
    // Reveal handled in block processing
  }

  // ============ Epoch Management ============

  /**
   * Handle epoch change
   */
  private async handleEpochChange(): Promise<void> {
    const oldEpoch = this.state.currentEpoch;
    const newEpoch = oldEpoch + 1n;

    // Collect stake changes and validator set updates from state layer
    const pendingStakeChanges: Array<{
      validator: PublicKey;
      newStake: bigint;
    }> = [];
    const removedValidators: PublicKey[] = [];

    if (this.stateIntegration) {
      // Sync consensus validator set with state-layer truth:
      // detect stake changes and deactivated validators
      for (const [pubKey, consensusVal] of this.state.validatorSet.validators) {
        const stateStake = this.stateIntegration.getValidatorStake(
          pubKey as PublicKey,
        );
        const isActive = this.stateIntegration.isActiveValidator(
          pubKey as PublicKey,
        );

        if (!isActive || stateStake === 0n) {
          // Validator deactivated or fully slashed in state
          removedValidators.push(pubKey as PublicKey);
        } else if (
          stateStake !==
          consensusVal.stake + consensusVal.delegatedStake
        ) {
          // Stake changed (new delegation, partial slash, etc.)
          pendingStakeChanges.push({
            validator: pubKey as PublicKey,
            newStake: stateStake,
          });
        }
      }
    }

    // Compute new validator set with actual changes
    const newValidatorSet = computeNextEpochValidators(
      this.state.validatorSet,
      pendingStakeChanges,
      [], // New validator registration handled by state transitions
      removedValidators,
      newEpoch,
    );

    this.state.currentEpoch = newEpoch;
    this.state.validatorSet = newValidatorSet;

    // Clean up equivocation detector to prevent unbounded memory growth
    this.equivocationDetector.cleanup(this.state.currentBlock);

    this.emitEvent({ type: "epoch_change", oldEpoch, newEpoch });
  }

  // ============ Finality ============

  /**
   * Update finality tracking
   *
   * Uses CONFIRMATION_BLOCKS from consensus params for soft finality.
   * After this many blocks, reorganization is unlikely.
   */
  private updateFinality(): void {
    // Use consensus param or default to 12 blocks
    const confirmationBlocks = this.config.params.confirmationBlocks ?? 12;
    const newFinalized = this.state.currentBlock - BigInt(confirmationBlocks);

    if (newFinalized > this.state.lastFinalizedBlock) {
      const finalizedBlock = this.state.recentBlocks.find(
        (b) => b.header.number === newFinalized,
      );

      if (finalizedBlock) {
        this.state.lastFinalizedBlock = newFinalized;
        this.state.lastFinalizedHash = this.hashBlockHeader(
          finalizedBlock.header,
        );

        this.emitEvent({
          type: "block_finalized",
          blockNumber: newFinalized,
          blockHash: this.state.lastFinalizedHash,
        });
      }
    }
  }

  /**
   * VULN-011 FIX: Handle detected equivocation
   * Slashes the validator's full stake and deactivates them
   */
  private async handleEquivocation(
    evidence: EquivocationEvidence,
  ): Promise<void> {
    const validator = this.state.validatorSet.validators.get(
      evidence.validator,
    );
    if (!validator) {
      console.warn(
        `[EQUIVOCATION] Validator ${evidence.validator} not found in validator set`,
      );
      return;
    }

    // Calculate slash amount (full stake for equivocation)
    const slashAmount = calculateEquivocationSlash(validator.stake);

    // Update validator state
    const updatedValidator = {
      ...validator,
      stake: 0n, // Full slash
      isActive: false, // Permanent deactivation
      slashCount: validator.slashCount + 1,
      reputation: 0, // Reputation destroyed
    };
    this.state.validatorSet.validators.set(
      evidence.validator,
      updatedValidator,
    );

    // Update total stake
    this.state.validatorSet.totalStake -= slashAmount;

    // If using state integration, apply to persistent state
    if (this.stateIntegration) {
      // State integration would handle the actual slashing
      console.log(
        `[EQUIVOCATION] Slashing ${evidence.validator} via state integration`,
      );
    }

    // Emit event
    this.emitEvent({
      type: "validator_slashed",
      validator: evidence.validator as PublicKey,
      amount: slashAmount,
      reason: `equivocation_${evidence.type}`,
    });

    // Log the evidence
    console.error(
      `[EQUIVOCATION] Slashed ${evidence.validator} for ${evidence.type}`,
    );
    console.error(
      `[EQUIVOCATION] Evidence:`,
      JSON.stringify(evidence, (_, v) =>
        typeof v === "bigint" ? v.toString() : v,
      ),
    );
  }

  /**
   * VULN-011 FIX: Get equivocation evidence for external submission/verification
   */
  getEquivocationEvidence(): EquivocationEvidence[] {
    return this.equivocationDetector.getEquivocationEvidence();
  }

  /**
   * VULN-011 FIX: Check if a validator has equivocated
   */
  hasValidatorEquivocated(validator: PublicKey): boolean {
    return this.equivocationDetector.hasEquivocated(validator);
  }

  /**
   * GAP-008 FIX: Create a new checkpoint using proper BFT ceremony
   */
  private async createNewCheckpoint(block: Block): Promise<void> {
    if (
      !this.config.isValidator ||
      !this.config.validatorKey ||
      !this.config.validatorPublicKey
    ) {
      return;
    }

    // Start the checkpoint ceremony
    const ceremony = this.checkpointCeremonyManager.startCeremony(
      block,
      this.config.validatorKey,
      this.config.validatorPublicKey,
      this.state.currentBlock,
    );

    // As proposer, we automatically prevote yes
    const prevote = createPrevote(
      ceremony.id,
      "yes",
      this.config.validatorKey,
      this.config.validatorPublicKey,
    );

    this.checkpointCeremonyManager.handlePrevote(
      ceremony.id,
      prevote,
      this.state.validatorSet,
      this.state.currentBlock,
    );

    this.emit("checkpoint_ceremony_started", {
      ceremonyId: ceremony.id,
      blockNumber: block.header.number,
      proposer: this.config.validatorPublicKey,
    });
  }

  /**
   * Process checkpoint ceremonies on each block
   */
  private processCheckpointCeremonies(): void {
    const newCheckpoints = this.checkpointCeremonyManager.processBlock(
      this.state.validatorSet,
      this.state.currentBlock,
    );

    for (const checkpoint of newCheckpoints) {
      this.checkpoints.push(checkpoint);
      this.emitEvent({
        type: "block_finalized",
        blockNumber: checkpoint.blockNumber,
        blockHash: checkpoint.blockHash,
      });
      this.emit("checkpoint_finalized", checkpoint);
    }
  }

  /**
   * Submit a prevote for a checkpoint ceremony
   * Called by validators participating in the ceremony
   */
  async submitCheckpointPrevote(
    ceremonyId: string,
    vote: "yes" | "no",
  ): Promise<{ accepted: boolean; reason?: string }> {
    if (
      !this.config.isValidator ||
      !this.config.validatorKey ||
      !this.config.validatorPublicKey
    ) {
      return { accepted: false, reason: "Not a validator" };
    }

    const prevote = createPrevote(
      ceremonyId,
      vote,
      this.config.validatorKey,
      this.config.validatorPublicKey,
    );

    return this.checkpointCeremonyManager.handlePrevote(
      ceremonyId,
      prevote,
      this.state.validatorSet,
      this.state.currentBlock,
    );
  }

  /**
   * Submit a precommit for a checkpoint ceremony
   * Called by validators after prevote phase
   */
  async submitCheckpointPrecommit(
    ceremonyId: string,
  ): Promise<{ accepted: boolean; reason?: string }> {
    if (
      !this.config.isValidator ||
      !this.config.validatorKey ||
      !this.config.validatorPublicKey
    ) {
      return { accepted: false, reason: "Not a validator" };
    }

    const ceremony = this.checkpointCeremonyManager.getActiveCeremony(
      this.state.currentBlock,
    );
    if (!ceremony || ceremony.id !== ceremonyId) {
      return { accepted: false, reason: "Ceremony not found or mismatch" };
    }

    const precommit = createPrecommit(
      ceremony,
      this.config.validatorKey,
      this.config.validatorPublicKey,
    );

    return this.checkpointCeremonyManager.handlePrecommit(
      ceremonyId,
      precommit,
      this.state.validatorSet,
      this.state.currentBlock,
    );
  }

  /**
   * Get active checkpoint ceremony
   */
  getActiveCheckpointCeremony(): CheckpointCeremony | undefined {
    return this.checkpointCeremonyManager.getActiveCeremony(
      this.state.currentBlock,
    );
  }

  // ============ Helper Methods ============

  private initState(): ConsensusState {
    return {
      currentBlock: 0n,
      currentEpoch: 0n,
      lastBlockTime: Date.now(),
      lastBlockHash: "0".repeat(64),
      validatorSet: createValidatorSet([], 0n),
      activeRounds: new Map(),
      pendingClaims: [],
      recentBlocks: [],
      lastFinalizedBlock: 0n,
      lastFinalizedHash: "0".repeat(64),
    };
  }

  private startBlockTimer(): void {
    this.blockTimer = setInterval(
      () => this.onBlockTick(),
      this.config.params.targetBlockTimeMs,
    );
  }

  private async onBlockTick(): Promise<void> {
    if (!this.running) return;

    try {
      await this.tryProduceBlock();
    } catch (error) {
      console.error("[PoT] Block production error:", error);
    }
  }

  private hashBlockHeader(header: BlockHeader): string {
    return sha512_256Hex(
      JSON.stringify({
        number: header.number.toString(),
        timestamp: header.timestamp,
        previousHash: header.previousHash,
        stateRoot: header.stateRoot,
        knowledgeRoot: header.knowledgeRoot,
        transactionsRoot: header.transactionsRoot,
        verificationsRoot: header.verificationsRoot,
        proposer: header.proposer,
      }),
    );
  }

  private getOurStake(): bigint {
    if (!this.config.validatorPublicKey) return 0n;
    const validator = this.state.validatorSet.validators.get(
      this.config.validatorPublicKey,
    );
    return validator ? validator.stake + validator.delegatedStake : 0n;
  }

  private getValidatorStakes(): Map<string, bigint> {
    const stakes = new Map<string, bigint>();
    for (const [key, validator] of this.state.validatorSet.validators) {
      stakes.set(key, validator.stake + validator.delegatedStake);
    }
    return stakes;
  }

  private getLastCheckpointBlock(): bigint {
    if (this.checkpoints.length === 0) return 0n;
    return this.checkpoints[this.checkpoints.length - 1].blockNumber;
  }

  /**
   * GAP-007 FIX: Compute real state root from state integration
   */
  private computeStateRoot(): string {
    if (this.stateIntegration) {
      return this.stateIntegration.computeStateRoot();
    }
    // Fallback for tests without state integration
    return sha512_256Hex(`state:${this.state.currentBlock}`);
  }

  /**
   * Compute knowledge tree root
   */
  private computeKnowledgeRoot(results: VerificationResult[]): string {
    if (this.stateIntegration) {
      return this.stateIntegration.computeKnowledgeRoot();
    }
    // Fallback: hash of claim IDs
    return sha512_256Hex(JSON.stringify(results.map((r) => r.claimId)));
  }

  /**
   * Compute transactions Merkle root
   */
  private computeTransactionsRoot(txs: Transaction[]): string {
    if (txs.length === 0) {
      return sha512_256Hex("empty");
    }
    // Build Merkle tree of transaction hashes
    const txHashes = txs.map((tx) =>
      sha512_256Hex(
        JSON.stringify({
          type: tx.type,
          sender: tx.sender,
          nonce: tx.nonce.toString(),
          payload: tx.payload,
        }),
      ),
    );
    return this.merkleRoot(txHashes);
  }

  /**
   * Compute verifications Merkle root
   */
  private computeVerificationsRoot(results: VerificationResult[]): string {
    if (results.length === 0) {
      return sha512_256Hex("empty");
    }
    // Build Merkle tree of verification result hashes
    const resultHashes = results.map((r) =>
      sha512_256Hex(
        JSON.stringify({
          claimId: r.claimId,
          outcome: r.outcome,
          finalConfidence: r.finalConfidence,
          recordedAt: r.recordedAt,
        }),
      ),
    );
    return this.merkleRoot(resultHashes);
  }

  /**
   * Compute Merkle root from array of hashes
   */
  private merkleRoot(hashes: string[]): string {
    if (hashes.length === 0) return sha512_256Hex("empty");
    if (hashes.length === 1) return hashes[0];

    const nextLevel: string[] = [];
    for (let i = 0; i < hashes.length; i += 2) {
      const left = hashes[i];
      const right = hashes[i + 1] || left; // Duplicate last if odd
      nextLevel.push(sha512_256Hex(left + right));
    }
    return this.merkleRoot(nextLevel);
  }

  private emitEvent(event: ConsensusEvent): void {
    this.emit(event.type, event);
    this.emit("consensus_event", event);
  }
}

// ============ Factory Functions ============

/**
 * Create a consensus engine for a full node
 */
export function createFullNodeEngine(
  initialValidators: Array<{
    publicKey: PublicKey;
    did: string;
    stake: bigint;
  }>,
  stateManager?: StateManager,
): ProofOfTruthEngine {
  const params = DEFAULT_CONSENSUS_PARAMS;

  // Create state integration if state manager provided
  const stateIntegration = stateManager
    ? createStateIntegration(stateManager, params)
    : undefined;

  const engine = new ProofOfTruthEngine({
    isValidator: false,
    params,
    stateIntegration,
  });

  // Initialize validator set
  const validatorSet = createValidatorSet(initialValidators);
  engine.setValidatorSet(validatorSet);

  return engine;
}

/**
 * Create a consensus engine for a validator node
 */
export function createValidatorEngine(
  validatorPrivateKey: PrivateKey,
  validatorPublicKey: PublicKey,
  initialValidators: Array<{
    publicKey: PublicKey;
    did: string;
    stake: bigint;
  }>,
  stateManager?: StateManager,
): ProofOfTruthEngine {
  const params = DEFAULT_CONSENSUS_PARAMS;

  // Create state integration if state manager provided
  const stateIntegration = stateManager
    ? createStateIntegration(stateManager, params)
    : undefined;

  const engine = new ProofOfTruthEngine({
    isValidator: true,
    validatorKey: validatorPrivateKey,
    validatorPublicKey,
    params,
    stateIntegration,
  });

  // Initialize validator set
  const validatorSet = createValidatorSet(initialValidators);
  engine.setValidatorSet(validatorSet);

  return engine;
}

/**
 * Create a fully integrated consensus engine with state
 * GAP FIX: This is the recommended way to create an engine for production use
 */
export function createIntegratedEngine(config: {
  isValidator: boolean;
  validatorKey?: PrivateKey;
  validatorPublicKey?: PublicKey;
  initialValidators: Array<{
    publicKey: PublicKey;
    did: string;
    stake: bigint;
  }>;
  stateManager: StateManager;
  params?: Partial<ConsensusParams>;
}): ProofOfTruthEngine {
  const params = { ...DEFAULT_CONSENSUS_PARAMS, ...config.params };

  const stateIntegration = createStateIntegration(config.stateManager, params);

  const engine = new ProofOfTruthEngine({
    isValidator: config.isValidator,
    validatorKey: config.validatorKey,
    validatorPublicKey: config.validatorPublicKey,
    params,
    stateIntegration,
  });

  // Initialize validator set
  const validatorSet = createValidatorSet(config.initialValidators);
  engine.setValidatorSet(validatorSet);

  return engine;
}
