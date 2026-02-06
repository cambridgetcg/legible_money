/**
 * Validator State Tree
 *
 * Sparse Merkle Tree for validator state.
 * Key: hash(publicKey) - 32 bytes
 * Value: serialized ValidatorState
 *
 * Also tracks:
 * - Delegations (separate SMT)
 * - Epoch transitions
 * - Performance metrics
 */

import {
  InMemorySMT,
  type MerkleRoot,
  type MerkleProofData,
} from "../core/crypto/merkle.js";
import { sha512_256 } from "../core/crypto/hashing.js";
import type { PublicKey } from "../core/crypto/keys.js";
import {
  type ValidatorState,
  type DelegationState,
  type ValidatorSetState,
  type UnbondingEntry,
  StateError,
} from "./types.js";

// ============ Serialization ============

function serializeValidatorState(validator: ValidatorState): Uint8Array {
  const json = JSON.stringify({
    publicKey: validator.publicKey,
    did: validator.did,
    stake: validator.stake.toString(),
    delegatedStake: validator.delegatedStake.toString(),
    totalStake: validator.totalStake.toString(),
    reputation: validator.reputation,
    isActive: validator.isActive,
    activeSinceBlock: validator.activeSinceBlock.toString(),
    lastProposedBlock: validator.lastProposedBlock.toString(),
    blocksProposed: validator.blocksProposed,
    verificationsCompleted: validator.verificationsCompleted,
    verificationsCorrect: validator.verificationsCorrect,
    slashCount: validator.slashCount,
    commissionRate: validator.commissionRate,
  });
  return new TextEncoder().encode(json);
}

function deserializeValidatorState(bytes: Uint8Array): ValidatorState {
  const json = new TextDecoder().decode(bytes);
  const obj = JSON.parse(json);
  return {
    publicKey: obj.publicKey as PublicKey,
    did: obj.did,
    stake: BigInt(obj.stake),
    delegatedStake: BigInt(obj.delegatedStake),
    totalStake: BigInt(obj.totalStake),
    reputation: obj.reputation,
    isActive: obj.isActive,
    activeSinceBlock: BigInt(obj.activeSinceBlock),
    lastProposedBlock: BigInt(obj.lastProposedBlock),
    blocksProposed: obj.blocksProposed,
    verificationsCompleted: obj.verificationsCompleted,
    verificationsCorrect: obj.verificationsCorrect,
    slashCount: obj.slashCount,
    commissionRate: obj.commissionRate,
  };
}

function serializeDelegation(delegation: DelegationState): Uint8Array {
  const json = JSON.stringify({
    delegator: delegation.delegator,
    validator: delegation.validator,
    amount: delegation.amount.toString(),
    delegatedAt: delegation.delegatedAt.toString(),
    unlocksAt: delegation.unlocksAt?.toString(),
  });
  return new TextEncoder().encode(json);
}

function deserializeDelegation(bytes: Uint8Array): DelegationState {
  const json = new TextDecoder().decode(bytes);
  const obj = JSON.parse(json);
  return {
    delegator: obj.delegator,
    validator: obj.validator as PublicKey,
    amount: BigInt(obj.amount),
    delegatedAt: BigInt(obj.delegatedAt),
    unlocksAt: obj.unlocksAt ? BigInt(obj.unlocksAt) : undefined,
  };
}

function validatorToKey(publicKey: PublicKey): Uint8Array {
  return sha512_256(publicKey);
}

function delegationToKey(delegator: string, validator: PublicKey): Uint8Array {
  return sha512_256(`${delegator}:${validator}`);
}

// ============ Validator State Tree ============

export class ValidatorStateTree {
  private validatorSmt: InMemorySMT;
  private delegationSmt: InMemorySMT;
  private validatorCache: Map<string, ValidatorState> = new Map();
  private delegationCache: Map<string, DelegationState> = new Map();

  // Unbonding queue for tracking pending undelegations
  // SECURITY: Tracks stake being undelegated to enforce unbonding period
  private unbondingQueue: Map<string, UnbondingEntry> = new Map();

  // Aggregate stats
  private totalStake: bigint = 0n;
  private activeCount: number = 0;
  private currentEpoch: bigint = 0n;

  constructor() {
    this.validatorSmt = new InMemorySMT();
    this.delegationSmt = new InMemorySMT();
  }

  // ============ Roots ============

  getValidatorRoot(): MerkleRoot {
    return this.validatorSmt.getRoot();
  }

  getDelegationRoot(): MerkleRoot {
    return this.delegationSmt.getRoot();
  }

  // ============ Validator Operations ============

  /**
   * Get validator by public key
   */
  getValidator(publicKey: PublicKey): ValidatorState | null {
    const cached = this.validatorCache.get(publicKey);
    if (cached) return cached;

    const key = validatorToKey(publicKey);
    const value = this.validatorSmt.get(key);
    if (!value) return null;

    const validator = deserializeValidatorState(value);
    this.validatorCache.set(publicKey, validator);
    return validator;
  }

  /**
   * Check if validator exists
   */
  isValidator(publicKey: PublicKey): boolean {
    return this.getValidator(publicKey) !== null;
  }

  /**
   * Check if validator is active
   */
  isActiveValidator(publicKey: PublicKey): boolean {
    const validator = this.getValidator(publicKey);
    return validator?.isActive ?? false;
  }

  /**
   * Register a new validator
   */
  registerValidator(
    publicKey: PublicKey,
    did: string,
    initialStake: bigint,
    blockNumber: bigint,
  ): MerkleRoot {
    if (this.isValidator(publicKey)) {
      throw new StateError(
        "Validator already registered",
        "INVALID_TRANSITION",
        { publicKey },
      );
    }

    const validator: ValidatorState = {
      publicKey,
      did,
      stake: initialStake,
      delegatedStake: 0n,
      totalStake: initialStake,
      reputation: 0.5,
      isActive: true,
      activeSinceBlock: blockNumber,
      lastProposedBlock: 0n,
      blocksProposed: 0,
      verificationsCompleted: 0,
      verificationsCorrect: 0,
      slashCount: 0,
      commissionRate: 0.1, // 10% default commission
    };

    const key = validatorToKey(publicKey);
    const value = serializeValidatorState(validator);
    const newRoot = this.validatorSmt.set(key, value);

    this.validatorCache.set(publicKey, validator);
    this.totalStake += initialStake;
    this.activeCount++;

    return newRoot;
  }

  /**
   * Update validator state
   */
  updateValidator(
    publicKey: PublicKey,
    update: Partial<ValidatorState>,
  ): MerkleRoot {
    const existing = this.getValidator(publicKey);
    if (!existing) {
      throw new StateError("Validator not found", "VALIDATOR_NOT_FOUND", {
        publicKey,
      });
    }

    // Track stake changes
    const oldTotalStake = existing.totalStake;
    const wasActive = existing.isActive;

    const updated: ValidatorState = {
      ...existing,
      ...update,
      totalStake:
        (update.stake ?? existing.stake) +
        (update.delegatedStake ?? existing.delegatedStake),
    };

    const key = validatorToKey(publicKey);
    const value = serializeValidatorState(updated);
    const newRoot = this.validatorSmt.set(key, value);

    // Update aggregates
    this.totalStake = this.totalStake - oldTotalStake + updated.totalStake;
    if (wasActive && !updated.isActive) {
      this.activeCount--;
    } else if (!wasActive && updated.isActive) {
      this.activeCount++;
    }

    this.validatorCache.set(publicKey, updated);
    return newRoot;
  }

  /**
   * Add stake to validator
   */
  addStake(publicKey: PublicKey, amount: bigint): MerkleRoot {
    const validator = this.getValidator(publicKey);
    if (!validator) {
      throw new StateError("Validator not found", "VALIDATOR_NOT_FOUND", {
        publicKey,
      });
    }

    return this.updateValidator(publicKey, {
      stake: validator.stake + amount,
    });
  }

  /**
   * Remove stake from validator
   */
  removeStake(publicKey: PublicKey, amount: bigint): MerkleRoot {
    const validator = this.getValidator(publicKey);
    if (!validator) {
      throw new StateError("Validator not found", "VALIDATOR_NOT_FOUND", {
        publicKey,
      });
    }

    if (validator.stake < amount) {
      throw new StateError("Insufficient stake", "INSUFFICIENT_BALANCE", {
        publicKey,
        stake: validator.stake.toString(),
        requested: amount.toString(),
      });
    }

    return this.updateValidator(publicKey, {
      stake: validator.stake - amount,
    });
  }

  /**
   * Deactivate a validator
   */
  deactivateValidator(publicKey: PublicKey): MerkleRoot {
    return this.updateValidator(publicKey, { isActive: false });
  }

  /**
   * Activate a validator
   */
  activateValidator(publicKey: PublicKey, blockNumber: bigint): MerkleRoot {
    return this.updateValidator(publicKey, {
      isActive: true,
      activeSinceBlock: blockNumber,
    });
  }

  /**
   * Slash a validator
   */
  slashValidator(
    publicKey: PublicKey,
    amount: bigint,
    _reason: string,
  ): MerkleRoot {
    const validator = this.getValidator(publicKey);
    if (!validator) {
      throw new StateError("Validator not found", "VALIDATOR_NOT_FOUND", {
        publicKey,
      });
    }

    const slashFromStake = amount > validator.stake ? validator.stake : amount;
    const slashFromDelegated = amount - slashFromStake;

    return this.updateValidator(publicKey, {
      stake: validator.stake - slashFromStake,
      delegatedStake: validator.delegatedStake - slashFromDelegated,
      reputation: Math.max(0, validator.reputation - 0.1),
      slashCount: validator.slashCount + 1,
    });
  }

  /**
   * Record block proposal
   */
  recordBlockProposal(publicKey: PublicKey, blockNumber: bigint): MerkleRoot {
    const validator = this.getValidator(publicKey);
    if (!validator) {
      throw new StateError("Validator not found", "VALIDATOR_NOT_FOUND", {
        publicKey,
      });
    }

    return this.updateValidator(publicKey, {
      lastProposedBlock: blockNumber,
      blocksProposed: validator.blocksProposed + 1,
    });
  }

  /**
   * Record verification outcome
   */
  recordVerification(publicKey: PublicKey, correct: boolean): MerkleRoot {
    const validator = this.getValidator(publicKey);
    if (!validator) {
      throw new StateError("Validator not found", "VALIDATOR_NOT_FOUND", {
        publicKey,
      });
    }

    const newCorrect = validator.verificationsCorrect + (correct ? 1 : 0);
    const newTotal = validator.verificationsCompleted + 1;
    // accuracy = newTotal > 0 ? newCorrect / newTotal : 0 (available for future metrics)

    // Reputation adjustment
    const reputationDelta = correct ? 0.001 : -0.005;

    return this.updateValidator(publicKey, {
      verificationsCompleted: newTotal,
      verificationsCorrect: newCorrect,
      reputation: Math.max(
        0,
        Math.min(1, validator.reputation + reputationDelta),
      ),
    });
  }

  // ============ Delegation Operations ============

  /**
   * Get delegation
   */
  getDelegation(
    delegator: string,
    validator: PublicKey,
  ): DelegationState | null {
    const cacheKey = `${delegator}:${validator}`;
    const cached = this.delegationCache.get(cacheKey);
    if (cached) return cached;

    const key = delegationToKey(delegator, validator);
    const value = this.delegationSmt.get(key);
    if (!value) return null;

    const delegation = deserializeDelegation(value);
    this.delegationCache.set(cacheKey, delegation);
    return delegation;
  }

  /**
   * Create or update delegation
   */
  delegate(
    delegator: string,
    validator: PublicKey,
    amount: bigint,
    blockNumber: bigint,
  ): MerkleRoot {
    const validatorState = this.getValidator(validator);
    if (!validatorState) {
      throw new StateError("Validator not found", "VALIDATOR_NOT_FOUND", {
        validator,
      });
    }

    const existing = this.getDelegation(delegator, validator);
    const newAmount = (existing?.amount ?? 0n) + amount;

    const delegation: DelegationState = {
      delegator,
      validator,
      amount: newAmount,
      delegatedAt: existing?.delegatedAt ?? blockNumber,
    };

    const key = delegationToKey(delegator, validator);
    const value = serializeDelegation(delegation);
    const newDelegationRoot = this.delegationSmt.set(key, value);

    // Update validator's delegated stake
    this.updateValidator(validator, {
      delegatedStake: validatorState.delegatedStake + amount,
    });

    const cacheKey = `${delegator}:${validator}`;
    this.delegationCache.set(cacheKey, delegation);

    return newDelegationRoot;
  }

  /**
   * Undelegate (start unbonding)
   */
  undelegate(
    delegator: string,
    validator: PublicKey,
    amount: bigint,
    currentBlock: bigint,
    unbondingPeriod: bigint,
  ): MerkleRoot {
    const delegation = this.getDelegation(delegator, validator);
    if (!delegation) {
      throw new StateError("Delegation not found", "INVALID_TRANSITION", {
        delegator,
        validator,
      });
    }

    if (delegation.amount < amount) {
      throw new StateError("Insufficient delegation", "INSUFFICIENT_BALANCE", {
        delegator,
        validator,
        delegated: delegation.amount.toString(),
        requested: amount.toString(),
      });
    }

    const validatorState = this.getValidator(validator);
    if (validatorState) {
      this.updateValidator(validator, {
        delegatedStake: validatorState.delegatedStake - amount,
      });
    }

    const newAmount = delegation.amount - amount;
    if (newAmount === 0n) {
      // Remove delegation entirely
      const key = delegationToKey(delegator, validator);
      this.delegationSmt.delete(key);
      this.delegationCache.delete(`${delegator}:${validator}`);
    } else {
      // Update remaining delegation
      const updated: DelegationState = {
        ...delegation,
        amount: newAmount,
        unlocksAt: currentBlock + unbondingPeriod,
      };

      const key = delegationToKey(delegator, validator);
      const value = serializeDelegation(updated);
      this.delegationSmt.set(key, value);
      this.delegationCache.set(`${delegator}:${validator}`, updated);
    }

    return this.getDelegationRoot();
  }

  /**
   * Reduce delegation amount without full undelegation logic
   * SECURITY: Used when starting undelegation - removes stake from active delegation
   */
  reduceDelegation(
    delegator: string,
    validator: PublicKey,
    amount: bigint,
  ): void {
    const delegation = this.getDelegation(delegator, validator);
    if (!delegation) {
      throw new StateError("Delegation not found", "INSUFFICIENT_DELEGATION", {
        delegator,
        validator,
      });
    }

    if (delegation.amount < amount) {
      throw new StateError(
        "Insufficient delegation",
        "INSUFFICIENT_DELEGATION",
        {
          delegator,
          validator,
          delegated: delegation.amount.toString(),
          requested: amount.toString(),
        },
      );
    }

    // Update validator's delegated stake
    const validatorState = this.getValidator(validator);
    if (validatorState) {
      this.updateValidator(validator, {
        delegatedStake: validatorState.delegatedStake - amount,
      });
    }

    // Update or remove delegation record
    const newAmount = delegation.amount - amount;
    if (newAmount === 0n) {
      const key = delegationToKey(delegator, validator);
      this.delegationSmt.delete(key);
      this.delegationCache.delete(`${delegator}:${validator}`);
    } else {
      const updated: DelegationState = {
        ...delegation,
        amount: newAmount,
      };

      const key = delegationToKey(delegator, validator);
      const value = serializeDelegation(updated);
      this.delegationSmt.set(key, value);
      this.delegationCache.set(`${delegator}:${validator}`, updated);
    }
  }

  /**
   * Add unbonding entry to track pending undelegation
   * SECURITY: Creates a time-locked record that prevents immediate stake withdrawal
   */
  addUnbondingEntry(entry: UnbondingEntry): void {
    this.unbondingQueue.set(entry.id, entry);
  }

  /**
   * Get all unbonding entries that have matured (timelock expired)
   * SECURITY: Only returns entries whose completesAt block has passed
   */
  getMaturedUnbondingEntries(currentBlock: bigint): UnbondingEntry[] {
    const matured: UnbondingEntry[] = [];

    for (const entry of this.unbondingQueue.values()) {
      if (entry.status === "pending" && currentBlock >= entry.completesAt) {
        matured.push(entry);
      }
    }

    return matured;
  }

  /**
   * Mark unbonding entry as completed
   * SECURITY: Called after funds have been released to delegator
   */
  completeUnbondingEntry(entryId: string): void {
    const entry = this.unbondingQueue.get(entryId);
    if (entry) {
      entry.status = "completed";
      // Optionally remove from queue after completion
      // this.unbondingQueue.delete(entryId);
    }
  }

  /**
   * Get unbonding entry by ID
   */
  getUnbondingEntry(entryId: string): UnbondingEntry | undefined {
    return this.unbondingQueue.get(entryId);
  }

  /**
   * Get all pending unbonding entries for a delegator
   */
  getPendingUnbondingEntries(delegator: string): UnbondingEntry[] {
    const entries: UnbondingEntry[] = [];

    for (const entry of this.unbondingQueue.values()) {
      if (entry.delegator === delegator && entry.status === "pending") {
        entries.push(entry);
      }
    }

    return entries;
  }

  // ============ Queries ============

  /**
   * Get total network stake
   */
  getTotalStake(): bigint {
    return this.totalStake;
  }

  /**
   * Get active validator count
   */
  getActiveCount(): number {
    return this.activeCount;
  }

  /**
   * Get current epoch
   */
  getCurrentEpoch(): bigint {
    return this.currentEpoch;
  }

  /**
   * Export all validators for persistence (includes inactive/jailed)
   */
  exportValidators(): ValidatorState[] {
    return Array.from(this.validatorCache.values());
  }

  /**
   * Get all active validators sorted by stake
   */
  getActiveValidators(): ValidatorState[] {
    return Array.from(this.validatorCache.values())
      .filter((v) => v.isActive)
      .sort((a, b) => {
        if (b.totalStake > a.totalStake) return 1;
        if (b.totalStake < a.totalStake) return -1;
        return 0;
      });
  }

  /**
   * Get validator set state (for snapshots/epoch boundaries)
   */
  getValidatorSetState(): ValidatorSetState {
    return {
      epoch: this.currentEpoch,
      totalStake: this.totalStake,
      activeValidators: this.activeCount,
      validatorRoot: this.getValidatorRoot(),
      delegationRoot: this.getDelegationRoot(),
    };
  }

  /**
   * Generate proof for validator
   */
  proveValidator(publicKey: PublicKey): MerkleProofData {
    const key = validatorToKey(publicKey);
    return this.validatorSmt.prove(key);
  }

  // ============ Epoch Transitions ============

  /**
   * Process epoch transition
   * - Recalculate reputations
   * - Apply rewards
   * - Process unbonding
   */
  processEpochTransition(
    newEpoch: bigint,
    rewards: Map<string, bigint>,
    _currentBlock: bigint,
  ): ValidatorSetState {
    this.currentEpoch = newEpoch;

    // Apply rewards
    for (const [validatorKey, reward] of rewards) {
      const validator = this.getValidator(validatorKey as PublicKey);
      if (validator) {
        // Reward goes to validator stake
        // Delegator rewards would be distributed separately
        this.addStake(validatorKey as PublicKey, reward);
      }
    }

    // Process any matured unbonding delegations
    // (would iterate unbonding queue in production)

    return this.getValidatorSetState();
  }

  /**
   * Clone for speculative execution
   */
  clone(): ValidatorStateTree {
    const cloned = new ValidatorStateTree();
    cloned.totalStake = this.totalStake;
    cloned.activeCount = this.activeCount;
    cloned.currentEpoch = this.currentEpoch;

    for (const [key, validator] of this.validatorCache) {
      cloned.validatorCache.set(key, { ...validator });
      const smtKey = validatorToKey(key as PublicKey);
      cloned.validatorSmt.set(smtKey, serializeValidatorState(validator));
    }

    for (const [key, delegation] of this.delegationCache) {
      cloned.delegationCache.set(key, { ...delegation });
      const [delegator, validator] = key.split(":");
      const smtKey = delegationToKey(delegator, validator as PublicKey);
      cloned.delegationSmt.set(smtKey, serializeDelegation(delegation));
    }

    return cloned;
  }
}

// ============ Factory Functions ============

/**
 * Minimum stake required to be a validator
 */
export const MIN_VALIDATOR_STAKE = 1000n * 10n ** 18n; // 1000 LGM

/**
 * Unbonding period in blocks
 */
export const UNBONDING_PERIOD = (7n * 24n * 60n * 60n * 1000n) / 2521n; // ~7 days at 2521ms blocks

/**
 * Create initial validator set from genesis
 */
export function createValidatorSetFromGenesis(
  validators: Array<{ publicKey: PublicKey; did: string; stake: bigint }>,
): ValidatorStateTree {
  const tree = new ValidatorStateTree();

  for (const v of validators) {
    tree.registerValidator(v.publicKey, v.did, v.stake, 0n);
  }

  return tree;
}
