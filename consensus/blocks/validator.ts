/**
 * Block Validator
 *
 * Validates blocks before applying them to state.
 */

import { verify } from '../../core/crypto/signatures.js';
import { verifyBlockProposer } from '../pot/validator_selection.js';
import { hashBlockHeader, checkBlockLimits, BLOCK_LIMITS } from './producer.js';
import type { PublicKey } from '../../core/crypto/keys.js';
import type { Block, ValidatorSet, ConsensusParams } from '../pot/types.js';

// ============ Validation ============

export interface ValidationContext {
  expectedBlockNumber: bigint;
  previousBlockHash: string;
  currentEpoch: bigint;
  validatorSet: ValidatorSet;
  params: ConsensusParams;
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
  context: ValidationContext
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
      `Invalid block number: expected ${context.expectedBlockNumber}, got ${block.header.number}`
    );
  }

  // 3. Check previous hash
  if (block.header.previousHash !== context.previousBlockHash) {
    errors.push('Invalid previous block hash');
  }

  // 4. Check timestamp
  const now = Date.now();
  if (block.header.timestamp > now + 60000) {
    errors.push('Block timestamp too far in future');
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
  const attestationsValidation = validateAttestations(block, context.validatorSet);
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
  context: ValidationContext
): { valid: boolean; reason?: string } {
  const validator = context.validatorSet.validators.get(block.header.proposer);
  if (!validator) {
    return { valid: false, reason: 'Unknown proposer' };
  }

  if (!validator.isActive) {
    return { valid: false, reason: 'Proposer is not active' };
  }

  // Verify VRF proof
  return verifyBlockProposer(
    block.header.proposer,
    block.header.proposerVrfOutput,
    block.header.proposerVrfProof,
    block.header.number,
    context.currentEpoch,
    context.previousBlockHash,
    validator.stake + validator.delegatedStake,
    context.validatorSet.totalStake
  );
}

/**
 * Validate block signature
 */
function validateBlockSignature(block: Block): { valid: boolean; reason?: string } {
  const headerHash = hashBlockHeader(block.header);
  const valid = verify(
    new TextEncoder().encode(headerHash),
    block.signature,
    block.header.proposer
  );

  return valid
    ? { valid: true }
    : { valid: false, reason: 'Invalid block signature' };
}

/**
 * Validate a single transaction
 */
function validateTransaction(
  tx: unknown,
  context: ValidationContext
): { valid: boolean; reason?: string } {
  // Type checking and basic validation
  // In production, this would be much more comprehensive
  return { valid: true };
}

/**
 * Validate a verification result
 */
function validateVerificationResult(
  result: unknown
): { valid: boolean; reason?: string } {
  // Validate result structure and values
  // In production, this would check that the result matches the round
  return { valid: true };
}

/**
 * Validate Merkle roots
 */
function validateMerkleRoots(block: Block): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  // In production, recompute Merkle roots and compare
  // This ensures the block proposer didn't tamper with the roots

  return { valid: errors.length === 0, errors };
}

/**
 * Validate block attestations
 */
function validateAttestations(
  block: Block,
  validatorSet: ValidatorSet
): { valid: boolean; warnings: string[] } {
  const warnings: string[] = [];

  // Check attestation count
  if (block.attestations.length === 0) {
    warnings.push('Block has no attestations');
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
      att.validator as PublicKey
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
  previousHash: string
): { valid: boolean; reason?: string } {
  // Check block number
  if (block.header.number !== expectedNumber) {
    return { valid: false, reason: 'Wrong block number' };
  }

  // Check previous hash
  if (block.header.previousHash !== previousHash) {
    return { valid: false, reason: 'Wrong previous hash' };
  }

  // Check basic structure
  if (!block.header || !block.signature) {
    return { valid: false, reason: 'Missing required fields' };
  }

  // Check size
  const size = JSON.stringify(block).length;
  if (size > BLOCK_LIMITS.MAX_BLOCK_SIZE_BYTES) {
    return { valid: false, reason: 'Block too large' };
  }

  return { valid: true };
}

// ============ Fork Validation ============

/**
 * Validate a potential fork block
 */
export function validateForkBlock(
  block: Block,
  forkPoint: bigint,
  currentHead: bigint
): { valid: boolean; reason?: string } {
  // Check that fork isn't too deep
  const MAX_REORG_DEPTH = 100n;
  if (currentHead - forkPoint > MAX_REORG_DEPTH) {
    return { valid: false, reason: 'Fork too deep' };
  }

  // Additional fork-specific validation
  return { valid: true };
}
