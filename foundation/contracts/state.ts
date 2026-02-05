/**
 * State Provider Contracts
 *
 * ARCH-3: Dependency Injection for Infrastructure
 * Domain modules MUST NOT directly instantiate infrastructure.
 * Infrastructure MUST be injected via these interfaces.
 *
 * MOD-2: Interface Segregation
 * Modules MUST NOT depend on interfaces they don't use.
 * State is split into focused reader/writer interfaces.
 */

import type {
  DID,
  PublicKey,
  Amount,
  Stake,
  BlockHeight,
  MerkleRoot,
  Confidence,
} from "../types/index.js";

// ============ Account State ============

/**
 * Account state reader - minimal interface for reading account data
 */
export interface AccountReader {
  /**
   * Get account balance
   */
  getBalance(did: DID): Promise<Amount>;

  /**
   * Get account staked balance
   */
  getStakedBalance(did: DID): Promise<Stake>;

  /**
   * Get account nonce (for replay protection)
   */
  getNonce(did: DID): Promise<bigint>;

  /**
   * Check if account exists
   */
  accountExists(did: DID): Promise<boolean>;

  /**
   * Get account reputation score (0-1)
   */
  getReputation(did: DID): Promise<number>;
}

/**
 * Account state writer - for modifying account state
 */
export interface AccountWriter {
  /**
   * Create a new account
   */
  createAccount(did: DID, initialBalance: Amount): Promise<void>;

  /**
   * Transfer balance between accounts
   */
  transfer(from: DID, to: DID, amount: Amount): Promise<void>;

  /**
   * Add to staked balance (lock tokens)
   */
  addStake(did: DID, amount: Stake): Promise<void>;

  /**
   * Remove from staked balance (unlock tokens)
   */
  removeStake(did: DID, amount: Stake): Promise<void>;

  /**
   * Increment account nonce
   */
  incrementNonce(did: DID): Promise<bigint>;

  /**
   * Update account reputation
   */
  updateAccountReputation(did: DID, delta: number): Promise<number>;
}

// ============ Validator State ============

/**
 * Validator state reader
 */
export interface ValidatorReader {
  /**
   * Get validator by public key
   */
  getValidator(publicKey: PublicKey): Promise<ValidatorInfo | null>;

  /**
   * Get validator by DID
   */
  getValidatorByDID(did: DID): Promise<ValidatorInfo | null>;

  /**
   * Get all active validators
   */
  getActiveValidators(): Promise<ValidatorInfo[]>;

  /**
   * Get total staked amount across all validators
   */
  getTotalStake(): Promise<Stake>;

  /**
   * Check if public key is an active validator
   */
  isActiveValidator(publicKey: PublicKey): Promise<boolean>;

  /**
   * Get validator count
   */
  getValidatorCount(): Promise<number>;
}

/**
 * Validator info returned by reader
 */
export interface ValidatorInfo {
  publicKey: PublicKey;
  did: DID;
  stake: Stake;
  delegatedStake: Stake;
  totalStake: Stake;
  reputation: number;
  isActive: boolean;
  activeSinceBlock: BlockHeight;
  lastProposedBlock: BlockHeight;
  blocksProposed: number;
  verificationsCompleted: number;
  verificationsCorrect: number;
  slashCount: number;
  commissionRate: number;
}

/**
 * Validator state writer
 */
export interface ValidatorWriter {
  /**
   * Register a new validator
   */
  registerValidator(
    publicKey: PublicKey,
    did: DID,
    stake: Stake,
  ): Promise<void>;

  /**
   * Update validator stake
   */
  updateStake(publicKey: PublicKey, delta: Stake): Promise<void>;

  /**
   * Slash validator (remove stake as punishment)
   */
  slash(publicKey: PublicKey, amount: Stake, reason: string): Promise<void>;

  /**
   * Update validator reputation
   */
  updateValidatorReputation(publicKey: PublicKey, delta: number): Promise<void>;

  /**
   * Record verification (correct or incorrect)
   */
  recordVerification(publicKey: PublicKey, correct: boolean): Promise<void>;

  /**
   * Deactivate validator
   */
  deactivate(publicKey: PublicKey): Promise<void>;
}

// ============ Knowledge State ============

/**
 * Knowledge state reader - for reading facts
 */
export interface KnowledgeReader {
  /**
   * Get fact by ID
   */
  getFact(factId: string): Promise<FactInfo | null>;

  /**
   * Query facts by domain
   */
  getFactsByDomain(domain: string, limit?: number): Promise<FactInfo[]>;

  /**
   * Get fact confidence
   */
  getConfidence(factId: string): Promise<Confidence | null>;

  /**
   * Check if fact exists
   */
  factExists(factId: string): Promise<boolean>;

  /**
   * Get facts derived from a given fact
   */
  getDerivedFacts(factId: string): Promise<string[]>;
}

/**
 * Fact info returned by reader
 */
export interface FactInfo {
  id: string;
  claim: string;
  claimHash: string;
  domain: string;
  confidence: Confidence;
  verifiedAt: BlockHeight;
  verifierCount: number;
  challengeCount: number;
  status: "active" | "challenged" | "superseded" | "revoked";
  submittedBy: DID;
}

/**
 * Knowledge state writer
 */
export interface KnowledgeWriter {
  /**
   * Add a verified fact
   */
  addFact(fact: Omit<FactInfo, "verifiedAt">): Promise<string>;

  /**
   * Update fact confidence
   */
  updateConfidence(factId: string, confidence: Confidence): Promise<void>;

  /**
   * Mark fact as challenged
   */
  markChallenged(factId: string, challengeId: string): Promise<void>;

  /**
   * Revoke a fact (proven false)
   */
  revokeFact(factId: string, reason: string): Promise<void>;

  /**
   * Supersede a fact with a new one
   */
  supersedeFact(oldFactId: string, newFactId: string): Promise<void>;
}

// ============ Combined State Provider ============

/**
 * Full state provider interface
 * Use this when you need multiple state operations
 */
export interface StateProvider
  extends
    AccountReader,
    AccountWriter,
    ValidatorReader,
    ValidatorWriter,
    KnowledgeReader,
    KnowledgeWriter {
  /**
   * Get current block height
   */
  getCurrentBlock(): Promise<BlockHeight>;

  /**
   * Get state root for given block
   */
  getStateRoot(block?: BlockHeight): Promise<MerkleRoot>;

  /**
   * Begin atomic transaction
   */
  beginTransaction(): Promise<TransactionHandle>;

  /**
   * Commit transaction
   */
  commitTransaction(handle: TransactionHandle): Promise<void>;

  /**
   * Rollback transaction
   */
  rollbackTransaction(handle: TransactionHandle): Promise<void>;
}

/**
 * Handle for atomic transactions
 */
export interface TransactionHandle {
  id: string;
  startedAt: number;
}

// ============ Read-Only Provider ============

/**
 * Read-only state provider (for queries)
 */
export interface StateReader
  extends AccountReader, ValidatorReader, KnowledgeReader {
  getCurrentBlock(): Promise<BlockHeight>;
  getStateRoot(block?: BlockHeight): Promise<MerkleRoot>;
}
