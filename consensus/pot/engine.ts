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

import { EventEmitter } from 'events';
import { sha512_256Hex } from '../../core/crypto/hashing.js';
import { sign, verify } from '../../core/crypto/signatures.js';
import { bytesToHex } from '../../core/crypto/keys.js';
import type { PublicKey, PrivateKey } from '../../core/crypto/keys.js';
import type { ClaimSubmission, Fact } from '../../core/knowledge/types.js';

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
  Validator,
} from './types.js';

import {
  selectValidatorsForVerification,
  canProposeBlock,
  verifyBlockProposer,
  createValidatorSet,
  updateValidatorStats,
  slashValidator,
  shouldChangeEpoch,
  computeNextEpochValidators,
} from './validator_selection.js';

import {
  createVerificationRound,
  startRound,
  addCommitment,
  transitionToReveal,
  addReveal,
  aggregateResults,
  getCurrentPhase,
} from './verification_round.js';

import {
  isBlockFinalized,
  shouldCreateCheckpoint,
  createCheckpoint,
  chooseFork,
  isReorgAllowed,
  type Checkpoint,
} from './finality.js';

// ============ Engine Configuration ============

export interface EngineConfig {
  params: ConsensusParams;
  validatorKey?: PrivateKey;
  validatorPublicKey?: PublicKey;
  isValidator: boolean;
}

// ============ Consensus Engine ============

export class ProofOfTruthEngine extends EventEmitter {
  private config: EngineConfig;
  private state: ConsensusState;
  private checkpoints: Checkpoint[] = [];
  private running: boolean = false;
  private blockTimer: NodeJS.Timeout | null = null;

  constructor(config: Partial<EngineConfig> = {}) {
    super();
    this.config = {
      params: DEFAULT_CONSENSUS_PARAMS,
      isValidator: false,
      ...config,
    };
    this.state = this.initState();
  }

  // ============ Lifecycle ============

  /**
   * Start the consensus engine
   */
  async start(): Promise<void> {
    if (this.running) return;

    this.running = true;
    this.emit('started');

    // Start block production timer
    this.startBlockTimer();

    console.log('[PoT] Consensus engine started');
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

    this.emit('stopped');
    console.log('[PoT] Consensus engine stopped');
  }

  /**
   * Get current state
   */
  getState(): ConsensusState {
    return { ...this.state };
  }

  // ============ Block Production ============

  /**
   * Try to produce a block
   */
  async tryProduceBlock(): Promise<Block | null> {
    if (!this.config.isValidator || !this.config.validatorKey) {
      return null;
    }

    // Check if we're selected to propose
    const selection = canProposeBlock(
      this.config.validatorKey,
      this.state.currentBlock + 1n,
      this.state.currentEpoch,
      this.state.lastBlockHash,
      this.getOurStake(),
      this.state.validatorSet.totalStake
    );

    if (!selection.canPropose) {
      return null;
    }

    // Build the block
    const block = await this.buildBlock(
      selection.vrfOutput,
      selection.vrfProof
    );

    // Sign and broadcast
    const signedBlock = this.signBlock(block);

    // Apply locally
    await this.applyBlock(signedBlock);

    // Emit event
    this.emitEvent({ type: 'block_proposed', block: signedBlock });

    return signedBlock;
  }

  /**
   * Build a new block
   */
  private async buildBlock(vrfOutput: string, vrfProof: string): Promise<Block> {
    const blockNumber = this.state.currentBlock + 1n;

    // Collect transactions from mempool
    const transactions = this.collectTransactions();

    // Process verification rounds that are complete
    const verificationResults = this.collectVerificationResults();

    // Build header
    const header: BlockHeader = {
      number: blockNumber,
      timestamp: Date.now(),
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
      signature: '', // Will be signed
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
      this.config.validatorKey!
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

    // Update state
    this.state.currentBlock = block.header.number;
    this.state.lastBlockHash = this.hashBlockHeader(block.header);
    this.state.lastBlockTime = block.header.timestamp;

    // Check for epoch change
    if (shouldChangeEpoch(block.header.number)) {
      await this.handleEpochChange();
    }

    // Check for checkpoint
    if (shouldCreateCheckpoint(block.header.number, this.getLastCheckpointBlock())) {
      await this.createNewCheckpoint(block);
    }

    // Update finality
    this.updateFinality();

    // Keep recent blocks
    this.state.recentBlocks.push(block);
    if (this.state.recentBlocks.length > 100) {
      this.state.recentBlocks.shift();
    }
  }

  /**
   * Validate a block
   */
  async validateBlock(block: Block): Promise<{ valid: boolean; reason?: string }> {
    // Check block number
    if (block.header.number !== this.state.currentBlock + 1n) {
      return { valid: false, reason: 'Invalid block number' };
    }

    // Check previous hash
    if (block.header.previousHash !== this.state.lastBlockHash) {
      return { valid: false, reason: 'Invalid previous hash' };
    }

    // Verify proposer selection
    const proposerValidator = this.state.validatorSet.validators.get(block.header.proposer);
    if (!proposerValidator) {
      return { valid: false, reason: 'Unknown proposer' };
    }

    const proposerValid = verifyBlockProposer(
      block.header.proposer,
      block.header.proposerVrfOutput,
      block.header.proposerVrfProof,
      block.header.number,
      this.state.currentEpoch,
      this.state.lastBlockHash,
      proposerValidator.stake,
      this.state.validatorSet.totalStake
    );

    if (!proposerValid.valid) {
      return { valid: false, reason: `Invalid proposer: ${proposerValid.reason}` };
    }

    // Verify block signature
    const headerHash = this.hashBlockHeader(block.header);
    const sigValid = verify(
      new TextEncoder().encode(headerHash),
      block.signature,
      block.header.proposer
    );

    if (!sigValid) {
      return { valid: false, reason: 'Invalid block signature' };
    }

    // Validate transactions
    for (const tx of block.transactions) {
      const txValid = await this.validateTransaction(tx);
      if (!txValid.valid) {
        return { valid: false, reason: `Invalid transaction: ${txValid.reason}` };
      }
    }

    return { valid: true };
  }

  // ============ Verification Rounds ============

  /**
   * Submit a new claim for verification
   */
  async submitClaim(claim: ClaimSubmission): Promise<string> {
    // Validate claim
    if (!claim.claim || claim.claim.length === 0) {
      throw new Error('Empty claim');
    }

    if (claim.references.length === 0) {
      throw new Error('Claim must have at least one reference');
    }

    // Add to pending claims
    this.state.pendingClaims.push(claim);

    // Create verification round
    const round = this.createRoundForClaim(claim);

    // Store round
    this.state.activeRounds.set(round.id, round);

    // Emit event
    this.emitEvent({ type: 'round_started', round });

    return round.id;
  }

  /**
   * Create verification round for a claim
   */
  private createRoundForClaim(claim: ClaimSubmission): VerificationRound {
    // Select validators
    const selectedValidators = selectValidatorsForVerification(
      sha512_256Hex(JSON.stringify(claim)),
      this.state.currentBlock,
      this.state.lastBlockHash,
      this.state.validatorSet,
      this.config.params
    );

    // Create round
    const round = createVerificationRound(
      claim,
      selectedValidators,
      Number(this.state.currentBlock),
      this.config.params
    );

    return startRound(round);
  }

  /**
   * Submit a verification commitment
   */
  async submitCommitment(
    roundId: string,
    commitment: {
      validator: PublicKey;
      commitment: string;
      signature: string;
    }
  ): Promise<void> {
    const round = this.state.activeRounds.get(roundId);
    if (!round) {
      throw new Error('Round not found');
    }

    const result = addCommitment(
      round,
      {
        validator: commitment.validator,
        commitment: commitment.commitment,
        submittedAt: Date.now(),
        signature: commitment.signature,
      },
      Number(this.state.currentBlock)
    );

    if (!result.accepted) {
      throw new Error(result.reason);
    }

    this.state.activeRounds.set(roundId, result.round);
  }

  /**
   * Submit a verification reveal
   */
  async submitReveal(
    roundId: string,
    reveal: {
      validator: PublicKey;
      verdict: 'confirmed' | 'rejected' | 'uncertain';
      confidence: number;
      nonce: string;
      referencesChecked: string[];
      signature: string;
    }
  ): Promise<void> {
    let round = this.state.activeRounds.get(roundId);
    if (!round) {
      throw new Error('Round not found');
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
      Number(this.state.currentBlock)
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

      if (phase === 'aggregation' || phase === 'complete') {
        // Aggregate results
        const aggregated = aggregateResults(
          round,
          validatorStakes,
          Number(this.state.currentBlock),
          this.config.params
        );

        if (aggregated.result) {
          results.push(aggregated.result);

          // Emit event
          this.emitEvent({ type: 'round_completed', result: aggregated.result });

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
   * Apply a verification result
   */
  private async applyVerificationResult(result: VerificationResult): Promise<void> {
    // Apply rewards
    for (const [validator, reward] of result.rewards) {
      // Add reward to validator's balance
      // In production, this updates the state tree
    }

    // Apply slashes
    for (const [validatorKey, slashAmount] of result.slashes) {
      const validator = this.state.validatorSet.validators.get(validatorKey);
      if (validator) {
        const { validator: slashed, slashedAmount } = slashValidator(
          validator,
          Number(slashAmount) / Number(validator.stake),
          'verification_failure'
        );
        this.state.validatorSet.validators.set(validatorKey, slashed);

        this.emitEvent({
          type: 'validator_slashed',
          validator: validatorKey as PublicKey,
          amount: slashedAmount,
          reason: 'verification_failure',
        });
      }
    }

    // If accepted, create fact (in production, this goes to knowledge layer)
    if (result.outcome === 'accepted') {
      // Create fact from claim
      // Store in knowledge layer
    }
  }

  // ============ Transaction Processing ============

  /**
   * Collect transactions from mempool
   */
  private collectTransactions(): Transaction[] {
    // In production, this would get transactions from mempool
    // Sorted by fee, filtered by validity
    return [];
  }

  /**
   * Validate a transaction
   */
  private async validateTransaction(tx: Transaction): Promise<{ valid: boolean; reason?: string }> {
    // Verify signature
    const valid = verify(
      new TextEncoder().encode(JSON.stringify(tx.payload)),
      tx.signature,
      tx.sender
    );

    if (!valid) {
      return { valid: false, reason: 'Invalid signature' };
    }

    // Check nonce, balance, etc.
    // In production, this checks against state

    return { valid: true };
  }

  /**
   * Apply a transaction
   */
  private async applyTransaction(tx: Transaction): Promise<void> {
    // In production, this updates state based on transaction type
    switch (tx.type) {
      case 'transfer':
        // Update balances
        break;
      case 'stake':
        // Add to stake
        break;
      case 'claim_submit':
        // Start verification round
        break;
      case 'verification_commit':
        // Add commitment
        break;
      case 'verification_reveal':
        // Add reveal
        break;
      // ... other types
    }
  }

  // ============ Epoch Management ============

  /**
   * Handle epoch change
   */
  private async handleEpochChange(): Promise<void> {
    const oldEpoch = this.state.currentEpoch;
    const newEpoch = oldEpoch + 1n;

    // Compute new validator set
    const newValidatorSet = computeNextEpochValidators(
      this.state.validatorSet,
      [], // Pending stake changes
      [], // New validators
      [], // Removed validators
      newEpoch
    );

    this.state.currentEpoch = newEpoch;
    this.state.validatorSet = newValidatorSet;

    this.emitEvent({ type: 'epoch_change', oldEpoch, newEpoch });
  }

  // ============ Finality ============

  /**
   * Update finality tracking
   */
  private updateFinality(): void {
    const newFinalized = this.state.currentBlock - BigInt(19); // CONFIRMATION_BLOCKS

    if (newFinalized > this.state.lastFinalizedBlock) {
      const finalizedBlock = this.state.recentBlocks.find(
        b => b.header.number === newFinalized
      );

      if (finalizedBlock) {
        this.state.lastFinalizedBlock = newFinalized;
        this.state.lastFinalizedHash = this.hashBlockHeader(finalizedBlock.header);

        this.emitEvent({
          type: 'block_finalized',
          blockNumber: newFinalized,
          blockHash: this.state.lastFinalizedHash,
        });
      }
    }
  }

  /**
   * Create a new checkpoint
   */
  private async createNewCheckpoint(block: Block): Promise<void> {
    // Collect signatures from validators
    // In production, this would be a multi-round process
    const signatures: Array<{ validator: string; signature: string; stake: bigint }> = [];

    const checkpoint = createCheckpoint(block, signatures);
    this.checkpoints.push(checkpoint);
  }

  // ============ Helper Methods ============

  private initState(): ConsensusState {
    return {
      currentBlock: 0n,
      currentEpoch: 0n,
      lastBlockTime: Date.now(),
      lastBlockHash: '0'.repeat(64),
      validatorSet: createValidatorSet([], 0n),
      activeRounds: new Map(),
      pendingClaims: [],
      recentBlocks: [],
      lastFinalizedBlock: 0n,
      lastFinalizedHash: '0'.repeat(64),
    };
  }

  private startBlockTimer(): void {
    this.blockTimer = setInterval(
      () => this.onBlockTick(),
      this.config.params.targetBlockTimeMs
    );
  }

  private async onBlockTick(): Promise<void> {
    if (!this.running) return;

    try {
      await this.tryProduceBlock();
    } catch (error) {
      console.error('[PoT] Block production error:', error);
    }
  }

  private hashBlockHeader(header: BlockHeader): string {
    return sha512_256Hex(JSON.stringify({
      number: header.number.toString(),
      timestamp: header.timestamp,
      previousHash: header.previousHash,
      stateRoot: header.stateRoot,
      knowledgeRoot: header.knowledgeRoot,
      transactionsRoot: header.transactionsRoot,
      verificationsRoot: header.verificationsRoot,
      proposer: header.proposer,
    }));
  }

  private getOurStake(): bigint {
    if (!this.config.validatorPublicKey) return 0n;
    const validator = this.state.validatorSet.validators.get(this.config.validatorPublicKey);
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

  private computeStateRoot(): string {
    // In production, this computes Merkle root of state tree
    return sha512_256Hex('state');
  }

  private computeKnowledgeRoot(results: VerificationResult[]): string {
    return sha512_256Hex(JSON.stringify(results.map(r => r.claimId)));
  }

  private computeTransactionsRoot(txs: Transaction[]): string {
    return sha512_256Hex(JSON.stringify(txs.map(t => t.signature)));
  }

  private computeVerificationsRoot(results: VerificationResult[]): string {
    return sha512_256Hex(JSON.stringify(results));
  }

  private emitEvent(event: ConsensusEvent): void {
    this.emit(event.type, event);
    this.emit('consensus_event', event);
  }
}

// ============ Factory Functions ============

/**
 * Create a consensus engine for a full node
 */
export function createFullNodeEngine(
  initialValidators: Array<{ publicKey: PublicKey; did: string; stake: bigint }>
): ProofOfTruthEngine {
  const engine = new ProofOfTruthEngine({
    isValidator: false,
  });

  // Initialize validator set
  const validatorSet = createValidatorSet(initialValidators);
  engine.getState().validatorSet = validatorSet;

  return engine;
}

/**
 * Create a consensus engine for a validator node
 */
export function createValidatorEngine(
  validatorPrivateKey: PrivateKey,
  validatorPublicKey: PublicKey,
  initialValidators: Array<{ publicKey: PublicKey; did: string; stake: bigint }>
): ProofOfTruthEngine {
  const engine = new ProofOfTruthEngine({
    isValidator: true,
    validatorKey: validatorPrivateKey,
    validatorPublicKey,
  });

  // Initialize validator set
  const validatorSet = createValidatorSet(initialValidators);
  engine.getState().validatorSet = validatorSet;

  return engine;
}
