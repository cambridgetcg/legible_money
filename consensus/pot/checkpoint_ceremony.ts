/**
 * Checkpoint Ceremony
 *
 * GAP-008/009 FIX: Proper BFT-style checkpoint signature collection and verification.
 *
 * Checkpoints provide economic finality beyond probabilistic finality.
 * A checkpoint is finalized when 67%+ of stake signs the same checkpoint.
 *
 * CEREMONY PHASES:
 * 1. PROPOSE - Block proposer creates checkpoint proposal
 * 2. PREVOTE - Validators signal intent to sign
 * 3. PRECOMMIT - Validators commit signatures
 * 4. FINALIZE - Checkpoint is finalized with aggregated signatures
 *
 * PP HIERARCHY:
 *   TRUTH: Cryptographic signatures guarantee authenticity
 *   UNDERSTANDING: Clear phase separation for reasoning about state
 *   BEAUTY: Minimal message complexity, O(n) signatures not O(n²)
 *   JUSTICE: Stake-weighted voting respects economic commitment
 */

import { sha512_256Hex } from "../../core/crypto/hashing.js";
import {
  signDomainSeparated,
  verifyDomainSeparated,
  type Signature,
} from "../../core/crypto/signatures.js";
import type { PublicKey, PrivateKey } from "../../core/crypto/keys.js";
import type { Block, ValidatorSet } from "./types.js";
import {
  FINALITY_PARAMS,
  type Checkpoint,
  type CheckpointSignature,
  aggregateCheckpointSignatures,
} from "./finality.js";

// ============ Ceremony Types ============

export type CeremonyPhase =
  | "propose"
  | "prevote"
  | "precommit"
  | "finalized"
  | "failed";

export interface CheckpointProposal {
  blockNumber: bigint;
  blockHash: string;
  stateRoot: string;
  knowledgeRoot: string;
  proposer: PublicKey;
  proposerSignature: Signature;
  createdAt: number;
}

export interface CheckpointPrevote {
  proposalHash: string;
  validator: PublicKey;
  vote: "yes" | "no";
  signature: Signature;
  submittedAt: number;
}

export interface CheckpointPrecommit {
  proposalHash: string;
  validator: PublicKey;
  checkpointSignature: Signature; // Signature over the full checkpoint data
  submittedAt: number;
}

export interface CheckpointCeremony {
  id: string;
  proposal: CheckpointProposal;
  phase: CeremonyPhase;

  // Timing
  startBlock: bigint;
  prevoteDeadline: bigint;
  precommitDeadline: bigint;
  // VULN-012 FIX: Add total ceremony timeout
  timeoutDeadline: bigint;

  // Votes
  prevotes: Map<string, CheckpointPrevote>; // validator -> prevote
  precommits: Map<string, CheckpointPrecommit>; // validator -> precommit

  // Stake tracking
  yesPrevoteStake: bigint;
  noPrevoteStake: bigint;
  precommitStake: bigint;

  // Result
  finalizedCheckpoint?: Checkpoint;
  failureReason?: string;
}

export interface CeremonyConfig {
  prevoteBlocks: number; // Blocks for prevote phase (default: 2)
  precommitBlocks: number; // Blocks for precommit phase (default: 2)
  stakeThreshold: number; // Required stake fraction (default: 0.67)
  // VULN-012 FIX: Add total ceremony timeout
  maxCeremonyBlocks: number; // Maximum blocks before ceremony is forcibly failed (default: 10)
}

export const DEFAULT_CEREMONY_CONFIG: CeremonyConfig = {
  prevoteBlocks: 2,
  precommitBlocks: 2,
  stakeThreshold: FINALITY_PARAMS.CHECKPOINT_SIGNATURES_REQUIRED,
  // VULN-012 FIX: Default timeout of 10 blocks (~25 seconds at 2.5s block time)
  maxCeremonyBlocks: 10,
};

// ============ Ceremony Creation ============

/**
 * Create a checkpoint proposal
 */
export function createCheckpointProposal(
  block: Block,
  proposerKey: PrivateKey,
  proposerPublicKey: PublicKey,
): CheckpointProposal {
  const proposalData = {
    blockNumber: block.header.number.toString(),
    blockHash: hashBlockHeader(block),
    stateRoot: block.header.stateRoot,
    knowledgeRoot: block.header.knowledgeRoot,
  };

  const signature = signDomainSeparated(
    "LGM.checkpoint.proposal.v1",
    JSON.stringify(proposalData),
    proposerKey,
  );

  return {
    blockNumber: block.header.number,
    blockHash: proposalData.blockHash,
    stateRoot: block.header.stateRoot,
    knowledgeRoot: block.header.knowledgeRoot,
    proposer: proposerPublicKey,
    proposerSignature: signature,
    createdAt: block.header.timestamp, // P1: Use deterministic block timestamp
  };
}

/**
 * Start a new checkpoint ceremony
 * VULN-012 FIX: Includes timeout deadline for ceremony recovery
 */
export function startCheckpointCeremony(
  proposal: CheckpointProposal,
  currentBlock: bigint,
  config: CeremonyConfig = DEFAULT_CEREMONY_CONFIG,
): CheckpointCeremony {
  const proposalHash = hashProposal(proposal);

  return {
    id: proposalHash,
    proposal,
    phase: "propose",
    startBlock: currentBlock,
    prevoteDeadline: currentBlock + BigInt(config.prevoteBlocks),
    precommitDeadline:
      currentBlock + BigInt(config.prevoteBlocks + config.precommitBlocks),
    // VULN-012 FIX: Set total ceremony timeout
    timeoutDeadline: currentBlock + BigInt(config.maxCeremonyBlocks),
    prevotes: new Map(),
    precommits: new Map(),
    yesPrevoteStake: 0n,
    noPrevoteStake: 0n,
    precommitStake: 0n,
  };
}

// ============ Prevote Phase ============

/**
 * Create a prevote for a checkpoint proposal
 */
export function createPrevote(
  proposalHash: string,
  vote: "yes" | "no",
  validatorKey: PrivateKey,
  validatorPublicKey: PublicKey,
  currentBlock?: bigint,
): CheckpointPrevote {
  // P0-1 FIX: Capture timestamp ONCE to avoid mismatch between signed data
  // and stored submittedAt (two Date.now() calls can return different values)
  const timestamp =
    currentBlock !== undefined ? Number(currentBlock) : Date.now();
  const voteData = {
    proposalHash,
    vote,
    timestamp,
  };

  const signature = signDomainSeparated(
    "LGM.checkpoint.prevote.v1",
    JSON.stringify(voteData),
    validatorKey,
  );

  return {
    proposalHash,
    validator: validatorPublicKey,
    vote,
    signature,
    submittedAt: timestamp,
  };
}

/**
 * Add a prevote to the ceremony
 */
export function addPrevote(
  ceremony: CheckpointCeremony,
  prevote: CheckpointPrevote,
  validatorSet: ValidatorSet,
  currentBlock: bigint,
): { accepted: boolean; ceremony: CheckpointCeremony; reason?: string } {
  // Check phase
  if (ceremony.phase !== "propose" && ceremony.phase !== "prevote") {
    return { accepted: false, ceremony, reason: "Not in prevote phase" };
  }

  // Check deadline
  if (currentBlock > ceremony.prevoteDeadline) {
    return { accepted: false, ceremony, reason: "Prevote deadline passed" };
  }

  // Check proposal hash matches
  if (prevote.proposalHash !== ceremony.id) {
    return { accepted: false, ceremony, reason: "Proposal hash mismatch" };
  }

  // Check validator exists and is active
  const validator = validatorSet.validators.get(prevote.validator);
  if (!validator || !validator.isActive) {
    return { accepted: false, ceremony, reason: "Validator not active" };
  }

  // Check for duplicate
  if (ceremony.prevotes.has(prevote.validator)) {
    return { accepted: false, ceremony, reason: "Already voted" };
  }

  // Verify signature
  const voteData = {
    proposalHash: prevote.proposalHash,
    vote: prevote.vote,
    timestamp: prevote.submittedAt,
  };

  const signatureValid = verifyDomainSeparated(
    "LGM.checkpoint.prevote.v1",
    JSON.stringify(voteData),
    prevote.signature,
    prevote.validator,
  );

  if (!signatureValid) {
    return { accepted: false, ceremony, reason: "Invalid signature" };
  }

  // Update ceremony
  const updatedCeremony = { ...ceremony };
  updatedCeremony.prevotes = new Map(ceremony.prevotes);
  updatedCeremony.prevotes.set(prevote.validator, prevote);

  const stake = validator.stake + validator.delegatedStake;
  if (prevote.vote === "yes") {
    updatedCeremony.yesPrevoteStake = ceremony.yesPrevoteStake + stake;
  } else {
    updatedCeremony.noPrevoteStake = ceremony.noPrevoteStake + stake;
  }

  // Transition to prevote phase if first vote
  if (ceremony.phase === "propose") {
    updatedCeremony.phase = "prevote";
  }

  return { accepted: true, ceremony: updatedCeremony };
}

/**
 * Check if prevote phase should end and move to precommit
 */
export function shouldTransitionToPrecommit(
  ceremony: CheckpointCeremony,
  validatorSet: ValidatorSet,
  currentBlock: bigint,
  config: CeremonyConfig = DEFAULT_CEREMONY_CONFIG,
): { shouldTransition: boolean; reason: string } {
  if (ceremony.phase !== "prevote") {
    return { shouldTransition: false, reason: "Not in prevote phase" };
  }

  const requiredStake =
    (validatorSet.totalStake *
      BigInt(Math.floor(config.stakeThreshold * 10000))) /
    10000n;

  // Check if yes votes reached threshold
  if (ceremony.yesPrevoteStake >= requiredStake) {
    return { shouldTransition: true, reason: "Prevote threshold reached" };
  }

  // Check if deadline passed
  if (currentBlock > ceremony.prevoteDeadline) {
    return { shouldTransition: true, reason: "Prevote deadline passed" };
  }

  // Check if impossible to reach threshold (too many no votes)
  const remainingStake =
    validatorSet.totalStake -
    ceremony.yesPrevoteStake -
    ceremony.noPrevoteStake;
  if (ceremony.yesPrevoteStake + remainingStake < requiredStake) {
    return { shouldTransition: true, reason: "Threshold impossible to reach" };
  }

  return { shouldTransition: false, reason: "Still collecting prevotes" };
}

/**
 * Transition ceremony to precommit phase
 */
export function transitionToPrecommit(
  ceremony: CheckpointCeremony,
  validatorSet: ValidatorSet,
  config: CeremonyConfig = DEFAULT_CEREMONY_CONFIG,
): CheckpointCeremony {
  const requiredStake =
    (validatorSet.totalStake *
      BigInt(Math.floor(config.stakeThreshold * 10000))) /
    10000n;

  // Check if we have enough yes votes
  if (ceremony.yesPrevoteStake < requiredStake) {
    return {
      ...ceremony,
      phase: "failed",
      failureReason: `Insufficient prevotes: ${ceremony.yesPrevoteStake} < ${requiredStake}`,
    };
  }

  return {
    ...ceremony,
    phase: "precommit",
  };
}

// ============ Precommit Phase ============

/**
 * Create a precommit signature for a checkpoint
 */
export function createPrecommit(
  ceremony: CheckpointCeremony,
  validatorKey: PrivateKey,
  validatorPublicKey: PublicKey,
): CheckpointPrecommit {
  // Sign the full checkpoint data (this becomes part of the final checkpoint)
  const checkpointData = {
    blockNumber: ceremony.proposal.blockNumber.toString(),
    blockHash: ceremony.proposal.blockHash,
    stateRoot: ceremony.proposal.stateRoot,
    knowledgeRoot: ceremony.proposal.knowledgeRoot,
  };

  const signature = signDomainSeparated(
    "LGM.checkpoint.commit.v1",
    JSON.stringify(checkpointData),
    validatorKey,
  );

  return {
    proposalHash: ceremony.id,
    validator: validatorPublicKey,
    checkpointSignature: signature,
    submittedAt: Number(ceremony.startBlock), // P1: Use block-based timestamp
  };
}

/**
 * Add a precommit to the ceremony
 */
export function addPrecommit(
  ceremony: CheckpointCeremony,
  precommit: CheckpointPrecommit,
  validatorSet: ValidatorSet,
  currentBlock: bigint,
): { accepted: boolean; ceremony: CheckpointCeremony; reason?: string } {
  // Check phase
  if (ceremony.phase !== "precommit") {
    return { accepted: false, ceremony, reason: "Not in precommit phase" };
  }

  // Check deadline
  if (currentBlock > ceremony.precommitDeadline) {
    return { accepted: false, ceremony, reason: "Precommit deadline passed" };
  }

  // Check proposal hash matches
  if (precommit.proposalHash !== ceremony.id) {
    return { accepted: false, ceremony, reason: "Proposal hash mismatch" };
  }

  // Check validator prevoted yes
  const prevote = ceremony.prevotes.get(precommit.validator);
  if (!prevote || prevote.vote !== "yes") {
    return {
      accepted: false,
      ceremony,
      reason: "Validator did not prevote yes",
    };
  }

  // Check validator exists and is active
  const validator = validatorSet.validators.get(precommit.validator);
  if (!validator || !validator.isActive) {
    return { accepted: false, ceremony, reason: "Validator not active" };
  }

  // Check for duplicate
  if (ceremony.precommits.has(precommit.validator)) {
    return { accepted: false, ceremony, reason: "Already precommitted" };
  }

  // GAP-009 FIX: Verify checkpoint signature
  const checkpointData = {
    blockNumber: ceremony.proposal.blockNumber.toString(),
    blockHash: ceremony.proposal.blockHash,
    stateRoot: ceremony.proposal.stateRoot,
    knowledgeRoot: ceremony.proposal.knowledgeRoot,
  };

  const signatureValid = verifyDomainSeparated(
    "LGM.checkpoint.commit.v1",
    JSON.stringify(checkpointData),
    precommit.checkpointSignature,
    precommit.validator,
  );

  if (!signatureValid) {
    return {
      accepted: false,
      ceremony,
      reason: "Invalid checkpoint signature",
    };
  }

  // Update ceremony
  const updatedCeremony = { ...ceremony };
  updatedCeremony.precommits = new Map(ceremony.precommits);
  updatedCeremony.precommits.set(precommit.validator, precommit);

  const stake = validator.stake + validator.delegatedStake;
  updatedCeremony.precommitStake = ceremony.precommitStake + stake;

  return { accepted: true, ceremony: updatedCeremony };
}

/**
 * Check if ceremony should finalize
 */
export function shouldFinalize(
  ceremony: CheckpointCeremony,
  validatorSet: ValidatorSet,
  currentBlock: bigint,
  config: CeremonyConfig = DEFAULT_CEREMONY_CONFIG,
): { shouldFinalize: boolean; reason: string } {
  if (ceremony.phase !== "precommit") {
    return { shouldFinalize: false, reason: "Not in precommit phase" };
  }

  const requiredStake =
    (validatorSet.totalStake *
      BigInt(Math.floor(config.stakeThreshold * 10000))) /
    10000n;

  // Check if precommit stake reached threshold
  if (ceremony.precommitStake >= requiredStake) {
    return { shouldFinalize: true, reason: "Precommit threshold reached" };
  }

  // Check if deadline passed
  if (currentBlock > ceremony.precommitDeadline) {
    return { shouldFinalize: true, reason: "Precommit deadline passed" };
  }

  return { shouldFinalize: false, reason: "Still collecting precommits" };
}

/**
 * VULN-012 FIX: Check if ceremony has exceeded its total timeout
 * This ensures ceremonies don't get stuck indefinitely if they stall
 */
export function isCeremonyTimedOut(
  ceremony: CheckpointCeremony,
  currentBlock: bigint,
): { timedOut: boolean; reason?: string } {
  // Don't timeout already finished ceremonies
  if (ceremony.phase === "finalized" || ceremony.phase === "failed") {
    return { timedOut: false };
  }

  if (currentBlock > ceremony.timeoutDeadline) {
    return {
      timedOut: true,
      reason: `Ceremony timed out after ${currentBlock - ceremony.startBlock} blocks (deadline: ${ceremony.timeoutDeadline}, current: ${currentBlock})`,
    };
  }

  return { timedOut: false };
}

/**
 * VULN-012 FIX: Handle ceremony timeout by marking it as failed
 */
export function handleCeremonyTimeout(
  ceremony: CheckpointCeremony,
  currentBlock: bigint,
): CheckpointCeremony {
  return {
    ...ceremony,
    phase: "failed",
    failureReason:
      `Ceremony timed out at block ${currentBlock}. Phase was '${ceremony.phase}'. ` +
      `Prevotes: ${ceremony.prevotes.size}, Precommits: ${ceremony.precommits.size}`,
  };
}

/**
 * Finalize the ceremony and create the checkpoint
 */
export function finalizeCeremony(
  ceremony: CheckpointCeremony,
  validatorSet: ValidatorSet,
  config: CeremonyConfig = DEFAULT_CEREMONY_CONFIG,
): CheckpointCeremony {
  const requiredStake =
    (validatorSet.totalStake *
      BigInt(Math.floor(config.stakeThreshold * 10000))) /
    10000n;

  // Check if we have enough precommits
  if (ceremony.precommitStake < requiredStake) {
    return {
      ...ceremony,
      phase: "failed",
      failureReason: `Insufficient precommits: ${ceremony.precommitStake} < ${requiredStake}`,
    };
  }

  // Build checkpoint signatures
  const signatures: CheckpointSignature[] = [];
  for (const [validatorKey, precommit] of ceremony.precommits) {
    const validator = validatorSet.validators.get(validatorKey);
    if (validator) {
      signatures.push({
        validator: validatorKey,
        signature: precommit.checkpointSignature,
        stake: validator.stake + validator.delegatedStake,
      });
    }
  }

  // Create the finalized checkpoint
  // GAP-026 FIX: Include aggregated signature data
  const aggregation = aggregateCheckpointSignatures(signatures, validatorSet);

  const checkpoint: Checkpoint = {
    blockNumber: ceremony.proposal.blockNumber,
    blockHash: ceremony.proposal.blockHash,
    stateRoot: ceremony.proposal.stateRoot,
    knowledgeRoot: ceremony.proposal.knowledgeRoot,
    signatures,
    totalSigningStake: ceremony.precommitStake,
    createdAt: ceremony.proposal.createdAt, // P1: Use proposal's block timestamp (deterministic)
    aggregation, // GAP-026 FIX: Add aggregation for efficient verification
  };

  return {
    ...ceremony,
    phase: "finalized",
    finalizedCheckpoint: checkpoint,
  };
}

// ============ Verification ============

/**
 * GAP-009 FIX: Verify a checkpoint with full cryptographic verification
 */
export function verifyCheckpointSignatures(
  checkpoint: Checkpoint,
  validatorSet: ValidatorSet,
): {
  valid: boolean;
  reason?: string;
  validSignatures: number;
  invalidSignatures: string[];
} {
  const checkpointData = {
    blockNumber: checkpoint.blockNumber.toString(),
    blockHash: checkpoint.blockHash,
    stateRoot: checkpoint.stateRoot,
    knowledgeRoot: checkpoint.knowledgeRoot,
  };

  const serializedData = JSON.stringify(checkpointData);
  const invalidSignatures: string[] = [];
  let verifiedStake = 0n;

  for (const sig of checkpoint.signatures) {
    // Check validator exists
    const validator = validatorSet.validators.get(sig.validator);
    if (!validator) {
      invalidSignatures.push(`${sig.validator}: unknown validator`);
      continue;
    }

    // Verify signature
    const signatureValid = verifyDomainSeparated(
      "LGM.checkpoint.commit.v1",
      serializedData,
      sig.signature,
      sig.validator as PublicKey,
    );

    if (!signatureValid) {
      invalidSignatures.push(`${sig.validator}: invalid signature`);
      continue;
    }

    // Verify stake matches
    const actualStake = validator.stake + validator.delegatedStake;
    if (actualStake !== sig.stake) {
      invalidSignatures.push(
        `${sig.validator}: stake mismatch (${sig.stake} vs ${actualStake})`,
      );
      continue;
    }

    verifiedStake += actualStake;
  }

  // Check stake threshold
  const requiredStake =
    (validatorSet.totalStake *
      BigInt(
        Math.floor(FINALITY_PARAMS.CHECKPOINT_SIGNATURES_REQUIRED * 10000),
      )) /
    10000n;

  if (verifiedStake < requiredStake) {
    return {
      valid: false,
      reason: `Insufficient verified stake: ${verifiedStake} < ${requiredStake}`,
      validSignatures: checkpoint.signatures.length - invalidSignatures.length,
      invalidSignatures,
    };
  }

  if (invalidSignatures.length > 0) {
    return {
      valid: false,
      reason: `Invalid signatures found`,
      validSignatures: checkpoint.signatures.length - invalidSignatures.length,
      invalidSignatures,
    };
  }

  return {
    valid: true,
    validSignatures: checkpoint.signatures.length,
    invalidSignatures: [],
  };
}

/**
 * Verify checkpoint proposal signature
 */
export function verifyCheckpointProposal(
  proposal: CheckpointProposal,
  validatorSet: ValidatorSet,
): { valid: boolean; reason?: string } {
  // Check proposer is a validator
  const validator = validatorSet.validators.get(proposal.proposer);
  if (!validator || !validator.isActive) {
    return { valid: false, reason: "Proposer not an active validator" };
  }

  // Verify signature
  const proposalData = {
    blockNumber: proposal.blockNumber.toString(),
    blockHash: proposal.blockHash,
    stateRoot: proposal.stateRoot,
    knowledgeRoot: proposal.knowledgeRoot,
  };

  const signatureValid = verifyDomainSeparated(
    "LGM.checkpoint.proposal.v1",
    JSON.stringify(proposalData),
    proposal.proposerSignature,
    proposal.proposer,
  );

  if (!signatureValid) {
    return { valid: false, reason: "Invalid proposer signature" };
  }

  return { valid: true };
}

// ============ Helper Functions ============

/**
 * Hash a checkpoint proposal
 */
function hashProposal(proposal: CheckpointProposal): string {
  return sha512_256Hex(
    JSON.stringify({
      blockNumber: proposal.blockNumber.toString(),
      blockHash: proposal.blockHash,
      stateRoot: proposal.stateRoot,
      knowledgeRoot: proposal.knowledgeRoot,
      proposer: proposal.proposer,
    }),
  );
}

/**
 * Hash a block header
 */
function hashBlockHeader(block: Block): string {
  return sha512_256Hex(
    JSON.stringify({
      number: block.header.number.toString(),
      timestamp: block.header.timestamp,
      previousHash: block.header.previousHash,
      stateRoot: block.header.stateRoot,
      knowledgeRoot: block.header.knowledgeRoot,
      transactionsRoot: block.header.transactionsRoot,
      verificationsRoot: block.header.verificationsRoot,
      proposer: block.header.proposer,
    }),
  );
}

// ============ Ceremony Manager ============

/**
 * Manages active checkpoint ceremonies
 */
export class CheckpointCeremonyManager {
  private activeCeremonies: Map<string, CheckpointCeremony> = new Map();
  private completedCheckpoints: Checkpoint[] = [];
  private config: CeremonyConfig;

  constructor(config: CeremonyConfig = DEFAULT_CEREMONY_CONFIG) {
    this.config = config;
  }

  /**
   * Start a new checkpoint ceremony
   */
  startCeremony(
    block: Block,
    proposerKey: PrivateKey,
    proposerPublicKey: PublicKey,
    currentBlock: bigint,
  ): CheckpointCeremony {
    const proposal = createCheckpointProposal(
      block,
      proposerKey,
      proposerPublicKey,
    );
    const ceremony = startCheckpointCeremony(
      proposal,
      currentBlock,
      this.config,
    );
    this.activeCeremonies.set(ceremony.id, ceremony);
    return ceremony;
  }

  /**
   * Handle a prevote
   */
  handlePrevote(
    ceremonyId: string,
    prevote: CheckpointPrevote,
    validatorSet: ValidatorSet,
    currentBlock: bigint,
  ): { accepted: boolean; reason?: string } {
    const ceremony = this.activeCeremonies.get(ceremonyId);
    if (!ceremony) {
      return { accepted: false, reason: "Ceremony not found" };
    }

    const result = addPrevote(ceremony, prevote, validatorSet, currentBlock);
    if (result.accepted) {
      this.activeCeremonies.set(ceremonyId, result.ceremony);
    }
    return { accepted: result.accepted, reason: result.reason };
  }

  /**
   * Handle a precommit
   */
  handlePrecommit(
    ceremonyId: string,
    precommit: CheckpointPrecommit,
    validatorSet: ValidatorSet,
    currentBlock: bigint,
  ): { accepted: boolean; reason?: string } {
    const ceremony = this.activeCeremonies.get(ceremonyId);
    if (!ceremony) {
      return { accepted: false, reason: "Ceremony not found" };
    }

    const result = addPrecommit(
      ceremony,
      precommit,
      validatorSet,
      currentBlock,
    );
    if (result.accepted) {
      this.activeCeremonies.set(ceremonyId, result.ceremony);
    }
    return { accepted: result.accepted, reason: result.reason };
  }

  /**
   * Process ceremonies on each block
   * VULN-012 FIX: Includes timeout checking for stalled ceremonies
   */
  processBlock(validatorSet: ValidatorSet, currentBlock: bigint): Checkpoint[] {
    const newCheckpoints: Checkpoint[] = [];

    for (const [id, ceremony] of this.activeCeremonies) {
      let updatedCeremony = ceremony;

      // VULN-012 FIX: Check for timeout FIRST before any other processing
      const { timedOut, reason } = isCeremonyTimedOut(ceremony, currentBlock);
      if (timedOut) {
        console.warn(`[Checkpoint] Ceremony ${id} timed out: ${reason}`);
        updatedCeremony = handleCeremonyTimeout(ceremony, currentBlock);
        // Skip to completion handling
      } else {
        // Check phase transitions (only if not timed out)
        if (ceremony.phase === "prevote") {
          const { shouldTransition } = shouldTransitionToPrecommit(
            ceremony,
            validatorSet,
            currentBlock,
            this.config,
          );
          if (shouldTransition) {
            updatedCeremony = transitionToPrecommit(
              ceremony,
              validatorSet,
              this.config,
            );
          }
        }

        if (updatedCeremony.phase === "precommit") {
          const { shouldFinalize: shouldFin } = shouldFinalize(
            updatedCeremony,
            validatorSet,
            currentBlock,
            this.config,
          );
          if (shouldFin) {
            updatedCeremony = finalizeCeremony(
              updatedCeremony,
              validatorSet,
              this.config,
            );
          }
        }
      }

      // Handle completion
      if (
        updatedCeremony.phase === "finalized" &&
        updatedCeremony.finalizedCheckpoint
      ) {
        newCheckpoints.push(updatedCeremony.finalizedCheckpoint);
        this.completedCheckpoints.push(updatedCeremony.finalizedCheckpoint);
        this.activeCeremonies.delete(id);
      } else if (updatedCeremony.phase === "failed") {
        this.activeCeremonies.delete(id);
      } else {
        this.activeCeremonies.set(id, updatedCeremony);
      }
    }

    return newCheckpoints;
  }

  /**
   * Get active ceremony for a block
   */
  getActiveCeremony(blockNumber: bigint): CheckpointCeremony | undefined {
    for (const ceremony of this.activeCeremonies.values()) {
      if (ceremony.proposal.blockNumber === blockNumber) {
        return ceremony;
      }
    }
    return undefined;
  }

  /**
   * Get all completed checkpoints
   */
  getCheckpoints(): Checkpoint[] {
    return [...this.completedCheckpoints];
  }

  /**
   * Get the latest checkpoint
   */
  getLatestCheckpoint(): Checkpoint | undefined {
    if (this.completedCheckpoints.length === 0) return undefined;
    return this.completedCheckpoints[this.completedCheckpoints.length - 1];
  }
}
