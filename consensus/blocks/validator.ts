/**
 * Block Validator
 *
 * Validates blocks before applying them to state.
 */

import { verify } from "../../core/crypto/signatures.js";
import { verifyBlockProposer } from "../pot/validator_selection.js";
import {
  hashBlockHeader,
  checkBlockLimits,
  BLOCK_LIMITS,
  computeTransactionsRoot,
  computeVerificationsRoot,
} from "./producer.js";
import type { PublicKey } from "../../core/crypto/keys.js";
import type {
  Block,
  ValidatorSet,
  ConsensusParams,
  Transaction,
  VerificationResult,
} from "../pot/types.js";

// ============ Validation ============

export interface ValidationContext {
  expectedBlockNumber: bigint;
  previousBlockHash: string;
  currentEpoch: bigint;
  validatorSet: ValidatorSet;
  params: ConsensusParams;
  /** Timestamp of the previous block (for monotonicity check). Optional for backward compat. */
  previousBlockTimestamp?: number;
}

export interface ValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

/**
 * Fully validate a block
 */
export function validateBlock(
  block: Block,
  context: ValidationContext,
): ValidationResult {
  const errors: string[] = [];
  const warnings: string[] = [];

  // 1. Check block limits
  const limits = checkBlockLimits(block);
  if (!limits.valid) {
    errors.push(limits.reason!);
  }

  // 2. Check block number
  if (block.header.number !== context.expectedBlockNumber) {
    errors.push(
      `Invalid block number: expected ${context.expectedBlockNumber}, got ${block.header.number}`,
    );
  }

  // 3. Check previous hash
  if (block.header.previousHash !== context.previousBlockHash) {
    errors.push("Invalid previous block hash");
  }

  // 4. Check timestamp (future bound + monotonicity)
  // Allow ~2 block times (5s) of clock drift, not 60s which spans ~24 blocks
  const now = Date.now();
  if (block.header.timestamp > now + 5000) {
    errors.push("Block timestamp too far in future");
  }
  if (
    context.previousBlockTimestamp !== undefined &&
    block.header.timestamp < context.previousBlockTimestamp
  ) {
    errors.push("Block timestamp before previous block");
  }

  // 5. Validate proposer
  const proposerValidation = validateProposer(block, context);
  if (!proposerValidation.valid) {
    errors.push(proposerValidation.reason!);
  }

  // 6. Validate block signature
  const sigValidation = validateBlockSignature(block);
  if (!sigValidation.valid) {
    errors.push(sigValidation.reason!);
  }

  // 7. Validate transactions
  for (let i = 0; i < block.transactions.length; i++) {
    const tx = block.transactions[i];
    const txValidation = validateTransaction(tx, context);
    if (!txValidation.valid) {
      errors.push(`Transaction ${i}: ${txValidation.reason}`);
    }
  }

  // 8. Validate verification results
  for (let i = 0; i < block.verificationResults.length; i++) {
    const result = block.verificationResults[i];
    const resultValidation = validateVerificationResult(result);
    if (!resultValidation.valid) {
      errors.push(`Verification result ${i}: ${resultValidation.reason}`);
    }
  }

  // 9. Validate Merkle roots
  const rootsValidation = validateMerkleRoots(block);
  if (!rootsValidation.valid) {
    errors.push(...rootsValidation.errors);
  }

  // 10. Validate attestations
  const attestationsValidation = validateAttestations(
    block,
    context.validatorSet,
  );
  if (attestationsValidation.warnings.length > 0) {
    warnings.push(...attestationsValidation.warnings);
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
  };
}

/**
 * Validate block proposer
 */
function validateProposer(
  block: Block,
  context: ValidationContext,
): { valid: boolean; reason?: string } {
  const validator = context.validatorSet.validators.get(block.header.proposer);
  if (!validator) {
    return { valid: false, reason: "Unknown proposer" };
  }

  if (!validator.isActive) {
    return { valid: false, reason: "Proposer is not active" };
  }

  // Verify VRF proof (targetProposers = validator set size for fair scheduling)
  return verifyBlockProposer(
    block.header.proposer,
    block.header.proposerVrfOutput,
    block.header.proposerVrfProof,
    block.header.number,
    context.currentEpoch,
    context.previousBlockHash,
    validator.stake + validator.delegatedStake,
    context.validatorSet.totalStake,
    Math.max(1, context.validatorSet.validators.size),
  );
}

/**
 * Validate block signature
 */
function validateBlockSignature(block: Block): {
  valid: boolean;
  reason?: string;
} {
  const headerHash = hashBlockHeader(block.header);
  const valid = verify(
    new TextEncoder().encode(headerHash),
    block.signature,
    block.header.proposer,
  );

  return valid
    ? { valid: true }
    : { valid: false, reason: "Invalid block signature" };
}

/**
 * Validate a single transaction
 */
function validateTransaction(
  tx: unknown,
  _context: ValidationContext,
): { valid: boolean; reason?: string } {
  const transaction = tx as Transaction;

  // Check required fields
  if (!transaction.type || !transaction.sender || !transaction.signature) {
    return { valid: false, reason: "Missing required transaction fields" };
  }

  // Check nonce is non-negative
  if (typeof transaction.nonce !== "bigint" || transaction.nonce < 0n) {
    return { valid: false, reason: "Invalid nonce" };
  }

  // Check fee is non-negative
  if (typeof transaction.fee !== "bigint" || transaction.fee < 0n) {
    return { valid: false, reason: "Invalid fee" };
  }

  // Verify transaction signature against payload.
  // NOTE: Signature covers payload only (which includes type). The top-level fee is NOT signed.
  // Fee malleability is low-risk (requires block producer collusion, who already sees all txs).
  // TODO: Migrate to getSignableMessage() which covers type+nonce+fee+payload — requires
  // coordinated wallet SDK + RPC update (see consensus/pot/types.ts:getSignableMessage).
  const payload = JSON.stringify(transaction.payload);
  const sigValid = verify(
    new TextEncoder().encode(payload),
    transaction.signature,
    transaction.sender,
  );

  if (!sigValid) {
    return { valid: false, reason: "Invalid transaction signature" };
  }

  return { valid: true };
}

/**
 * Validate a verification result
 */
function validateVerificationResult(result: unknown): {
  valid: boolean;
  reason?: string;
} {
  const vr = result as VerificationResult;

  // Check required fields
  if (!vr.claimId) {
    return { valid: false, reason: "Missing claimId" };
  }

  // Check valid outcome
  const validOutcomes = ["accepted", "rejected", "contested", "insufficient"];
  if (!validOutcomes.includes(vr.outcome)) {
    return { valid: false, reason: `Invalid outcome: ${vr.outcome}` };
  }

  // Check confidence range
  if (
    typeof vr.finalConfidence !== "number" ||
    vr.finalConfidence < 0 ||
    vr.finalConfidence > 1
  ) {
    return { valid: false, reason: "Final confidence must be between 0 and 1" };
  }

  // Check breakdown sums are valid (allow small floating-point error)
  if (vr.confidenceBreakdown) {
    const sum =
      vr.confidenceBreakdown.confirmed +
      vr.confidenceBreakdown.rejected +
      vr.confidenceBreakdown.uncertain;
    if (sum > 1.01 || sum < 0) {
      return { valid: false, reason: "Confidence breakdown fractions invalid" };
    }
  }

  // Check participation
  if (vr.participatingValidators > vr.totalValidators) {
    return { valid: false, reason: "Participating validators exceeds total" };
  }

  return { valid: true };
}

/**
 * Validate Merkle roots
 *
 * Recomputes transaction and verification Merkle roots from block contents
 * and compares against the header values to detect tampering.
 */
function validateMerkleRoots(block: Block): {
  valid: boolean;
  errors: string[];
} {
  const errors: string[] = [];

  // Recompute and verify transactions root
  const expectedTxRoot = computeTransactionsRoot(block.transactions);
  if (block.header.transactionsRoot !== expectedTxRoot) {
    errors.push(
      `Transactions root mismatch: header=${block.header.transactionsRoot}, computed=${expectedTxRoot}`,
    );
  }

  // Recompute and verify verifications root
  const expectedVrRoot = computeVerificationsRoot(block.verificationResults);
  if (block.header.verificationsRoot !== expectedVrRoot) {
    errors.push(
      `Verifications root mismatch: header=${block.header.verificationsRoot}, computed=${expectedVrRoot}`,
    );
  }

  return { valid: errors.length === 0, errors };
}

/**
 * Validate block attestations
 */
function validateAttestations(
  block: Block,
  validatorSet: ValidatorSet,
): { valid: boolean; warnings: string[] } {
  const warnings: string[] = [];

  // Check attestation count
  if (block.attestations.length === 0) {
    warnings.push("Block has no attestations");
  }

  // Verify each attestation signature
  for (const att of block.attestations) {
    const validator = validatorSet.validators.get(att.validator);
    if (!validator) {
      warnings.push(`Unknown attestor: ${att.validator}`);
      continue;
    }

    const headerHash = hashBlockHeader(block.header);
    const valid = verify(
      new TextEncoder().encode(headerHash),
      att.signature,
      att.validator as PublicKey,
    );

    if (!valid) {
      warnings.push(`Invalid attestation signature from ${att.validator}`);
    }
  }

  return { valid: true, warnings };
}

// ============ Quick Validation ============

/**
 * Quick validation for gossiped blocks
 * Only checks basics, full validation done before applying
 */
export function quickValidateBlock(
  block: Block,
  expectedNumber: bigint,
  previousHash: string,
): { valid: boolean; reason?: string } {
  // Check block number
  if (block.header.number !== expectedNumber) {
    return { valid: false, reason: "Wrong block number" };
  }

  // Check previous hash
  if (block.header.previousHash !== previousHash) {
    return { valid: false, reason: "Wrong previous hash" };
  }

  // Check basic structure
  if (!block.header || !block.signature) {
    return { valid: false, reason: "Missing required fields" };
  }

  // P1-B1 FIX: Use byte length, not character count (bigintReplacer prevents crash)
  const size = new TextEncoder().encode(
    JSON.stringify(block, (_k, v) =>
      typeof v === "bigint" ? v.toString() : v,
    ),
  ).byteLength;
  if (size > BLOCK_LIMITS.MAX_BLOCK_SIZE_BYTES) {
    return { valid: false, reason: "Block too large" };
  }

  return { valid: true };
}

// ============ Fork Validation ============

/**
 * Validate a potential fork block
 */
export function validateForkBlock(
  _block: Block,
  forkPoint: bigint,
  currentHead: bigint,
): { valid: boolean; reason?: string } {
  // Check that fork isn't too deep
  const MAX_REORG_DEPTH = 100n;
  if (currentHead - forkPoint > MAX_REORG_DEPTH) {
    return { valid: false, reason: "Fork too deep" };
  }

  // Additional fork-specific validation
  return { valid: true };
}
