/**
 * State Integration Layer
 *
 * GAP-003/004/005/006/007 FIX: Bridge between consensus engine and state layer.
 *
 * This module connects the Proof of Truth consensus engine to the actual
 * persistent state trees (accounts, validators, knowledge, etc.).
 *
 * Before: Engine had stub functions returning empty/hardcoded values
 * After: Engine uses this integration layer for real state operations
 *
 * PP HIERARCHY:
 *   TRUTH: State reflects actual verified facts and balances
 *   UNDERSTANDING: Clear separation between consensus and state
 *   BEAUTY: Single integration point, clean interface
 *   JUSTICE: All transactions executed through same pathway
 */

import {
  StateManager,
  type PersistedBlockHeader,
} from "../../state/state_manager.js";
import {
  executeTransaction,
  type TransitionResult,
} from "../../state/transitions.js";
import type {
  ConsensusParams,
  VerificationResult,
  Block,
  Transaction,
} from "./types.js";
import type { PublicKey } from "../../core/crypto/keys.js";
import type { KVStore } from "../../state/storage/types.js";
import { MemoryStore, createStorage } from "../../state/storage/leveldb.js";
import { Mempool as CanonicalMempool } from "../mempool/pool.js";
import type { DomainTree } from "../../core/knowledge/ontology/types.js";
import type { DomainProposal } from "../../core/knowledge/ontology/domain_proposal.js";

// ============ Integration Interface ============

export interface StateIntegration {
  /** Get the underlying state manager */
  getStateManager(): StateManager;

  /** Compute the current state root */
  computeStateRoot(): string;

  /** Compute knowledge tree root */
  computeKnowledgeRoot(): string;

  /** Validate a transaction against current state */
  validateTransaction(tx: Transaction): Promise<TransactionValidation>;

  /** Execute a transaction and update state */
  executeTransaction(
    tx: Transaction,
    blockNumber: bigint,
  ): Promise<TransitionResult>;

  /** Apply a verification result (create fact, distribute rewards) */
  applyVerificationResult(
    result: VerificationResult,
    blockNumber: bigint,
  ): Promise<void>;

  /** Get pending transactions from mempool (sorted by fee) */
  collectTransactions(maxCount: number): Transaction[];

  /** Add transaction to mempool */
  addToMempool(tx: Transaction): boolean;

  /** Apply a complete block to state */
  applyBlock(block: Block): Promise<string>;

  /** Get account balance */
  getBalance(did: string): bigint;

  /** Get account nonce */
  getNonce(did: string): bigint;

  /** Check if validator is active */
  isActiveValidator(publicKey: PublicKey): boolean;

  /** Get validator stake */
  getValidatorStake(publicKey: PublicKey): bigint;

  /** Get total stake in system */
  getTotalStake(): bigint;

  /** Initialize from persisted storage (if available) */
  initialize(): Promise<boolean>;

  /** Check if persistence is enabled */
  hasPersistence(): boolean;

  /** Get the latest persisted block number */
  getLatestBlockNumber(): Promise<bigint | null>;

  /** Close resources (storage, etc.) */
  close(): Promise<void>;

  // ============ Claim Management ============

  /** Get pending claims ready for verification */
  getPendingClaims(): Array<{
    claimId: string;
    domain: string;
    stratum: string;
    submittedAt: bigint;
    stake: bigint;
  }>;

  /** Start verification for a pending claim */
  startClaimVerification(
    claimId: string,
    roundId: string,
    validators: string[],
    blockNumber: bigint,
  ): Promise<void>;

  /** Complete claim verification with result */
  completeClaimVerification(
    claimId: string,
    roundId: string,
    outcome: "accepted" | "rejected" | "contested" | "insufficient",
    finalConfidence: number,
    verifierCount: number,
    rewards: Array<[string, bigint]>,
    slashes: Array<[string, bigint]>,
    blockNumber: bigint,
  ): Promise<void>;

  /** Check if a claim exists and is valid for verification */
  isValidClaim(claimId: string): boolean;

  /** Get the current domain tree from state (synced with governance) */
  getDomainTree(): DomainTree;

  /** Get pending domain proposals (for rejecting claims to in-flight domains) */
  getPendingDomainProposals(): Map<string, DomainProposal>;
}

export interface TransactionValidation {
  valid: boolean;
  reason?: string;
  estimatedGas?: bigint;
}

// ============ Cache Invalidation Hooks ============

/**
 * Callback interface for knowledge index cache invalidation.
 * Used by UnifiedQueryService to keep caches in sync with state changes.
 */
export interface KnowledgeCacheHooks {
  /** Called when a new fact is indexed */
  onFactIndexed?: (
    factId: string,
    domainId: string,
    confidence: number,
    blockNumber: bigint,
  ) => void;
  /** Called when fact confidence changes */
  onConfidenceChanged?: (
    factId: string,
    oldConfidence: number,
    newConfidence: number,
  ) => void;
  /** Called when a fact is removed */
  onFactRemoved?: (factId: string, domainId: string) => void;
  /** Called when domain trajectory should be updated */
  onDomainUpdated?: (domainId: string) => void;
}

// ============ Mempool (delegates to canonical consensus/mempool/pool.ts) ============

/**
 * Thin adapter around the canonical Mempool from consensus/mempool/pool.ts.
 * Bridges the state-integration API to the canonical implementation,
 * eliminating the previous code duplication.
 */
class IntegrationMempool {
  private pool: CanonicalMempool;
  private txHashes: Map<string, string> = new Map(); // txId → txHash

  constructor(maxSize: number = 10000, maxPerAccount: number = 100) {
    this.pool = new CanonicalMempool({
      maxSize,
      maxPerSender: maxPerAccount,
      minFee: 0n, // State integration handles its own fee validation
    });
  }

  add(tx: Transaction, _fee: bigint, _gasEstimate: bigint): boolean {
    const result = this.pool.add(tx);
    if (result.added && result.txHash) {
      const txId = `${tx.sender}:${tx.nonce}:${tx.type}`;
      this.txHashes.set(txId, result.txHash);
    }
    return result.added;
  }

  getTransactionsByFee(maxCount: number): Transaction[] {
    return this.pool.getForBlock(maxCount, 1_000_000);
  }

  remove(tx: Transaction): void {
    const txId = `${tx.sender}:${tx.nonce}:${tx.type}`;
    const txHash = this.txHashes.get(txId);
    if (txHash) {
      this.pool.remove(txHash);
      this.txHashes.delete(txId);
    }
  }

  size(): number {
    return this.pool.size();
  }
}

// ============ State Integration Implementation ============

export class StateIntegrationImpl implements StateIntegration {
  private stateManager: StateManager;
  private mempool: IntegrationMempool;
  private consensusParams: ConsensusParams;
  private store?: KVStore;
  private cacheHooks?: KnowledgeCacheHooks;

  constructor(
    stateManager: StateManager,
    consensusParams: ConsensusParams,
    mempoolConfig?: { maxSize?: number; maxPerAccount?: number },
    store?: KVStore,
  ) {
    this.stateManager = stateManager;
    this.consensusParams = consensusParams;
    this.store = store;
    this.mempool = new IntegrationMempool(
      mempoolConfig?.maxSize,
      mempoolConfig?.maxPerAccount,
    );
  }

  /**
   * Set cache invalidation hooks for knowledge index.
   * Called by UnifiedQueryService during initialization.
   */
  setCacheHooks(hooks: KnowledgeCacheHooks): void {
    this.cacheHooks = hooks;
  }

  /**
   * Notify cache hooks of fact index event.
   */
  private notifyFactIndexed(
    factId: string,
    domainId: string,
    confidence: number,
    blockNumber: bigint,
  ): void {
    if (this.cacheHooks?.onFactIndexed) {
      try {
        this.cacheHooks.onFactIndexed(
          factId,
          domainId,
          confidence,
          blockNumber,
        );
      } catch (e) {
        // Cache hook errors shouldn't break consensus
        console.warn("[StateIntegration] Cache hook error:", e);
      }
    }
  }

  /**
   * Notify cache hooks of confidence change.
   * @internal Exposed for testing and future use when confidence updates are implemented
   */
  notifyConfidenceChanged(
    factId: string,
    oldConfidence: number,
    newConfidence: number,
  ): void {
    if (this.cacheHooks?.onConfidenceChanged) {
      try {
        this.cacheHooks.onConfidenceChanged(
          factId,
          oldConfidence,
          newConfidence,
        );
      } catch (e) {
        console.warn("[StateIntegration] Cache hook error:", e);
      }
    }
  }

  /**
   * Notify cache hooks of fact removal.
   * @internal Exposed for testing and future use when fact removal is implemented
   */
  notifyFactRemoved(factId: string, domainId: string): void {
    if (this.cacheHooks?.onFactRemoved) {
      try {
        this.cacheHooks.onFactRemoved(factId, domainId);
      } catch (e) {
        console.warn("[StateIntegration] Cache hook error:", e);
      }
    }
  }

  /**
   * Notify cache hooks of domain update.
   */
  private notifyDomainUpdated(domainId: string): void {
    if (this.cacheHooks?.onDomainUpdated) {
      try {
        this.cacheHooks.onDomainUpdated(domainId);
      } catch (e) {
        console.warn("[StateIntegration] Cache hook error:", e);
      }
    }
  }

  getStateManager(): StateManager {
    return this.stateManager;
  }

  /**
   * GAP-007 FIX: Compute real state root from StateManager
   */
  computeStateRoot(): string {
    return this.stateManager.getStateRoot();
  }

  /**
   * Compute knowledge tree root
   */
  computeKnowledgeRoot(): string {
    const components = this.stateManager.getStateRootComponents();
    return components.knowledgeRoot;
  }

  /**
   * GAP-006 FIX: Validate transaction against actual state
   */
  async validateTransaction(tx: Transaction): Promise<TransactionValidation> {
    // Check account exists
    const account = this.stateManager.getAccount(tx.sender);
    if (!account) {
      return { valid: false, reason: "Account not found" };
    }

    // Check nonce
    if (tx.nonce !== account.nonce) {
      return {
        valid: false,
        reason: `Invalid nonce: expected ${account.nonce}, got ${tx.nonce}`,
      };
    }

    // Check balance for fee
    const fee = tx.fee || 0n;
    if (account.balance < fee) {
      return {
        valid: false,
        reason: `Insufficient balance for fee: have ${account.balance}, need ${fee}`,
      };
    }

    // Type-specific validation
    switch (tx.type) {
      case "transfer": {
        const payload = tx.payload as { to: string; amount: bigint };
        const totalNeeded = fee + payload.amount;
        if (account.balance < totalNeeded) {
          return {
            valid: false,
            reason: `Insufficient balance: have ${account.balance}, need ${totalNeeded}`,
          };
        }
        break;
      }

      case "stake": {
        const payload = tx.payload as { amount: bigint };
        const totalNeeded = fee + payload.amount;
        if (account.balance < totalNeeded) {
          return {
            valid: false,
            reason: `Insufficient balance for staking: have ${account.balance}, need ${totalNeeded}`,
          };
        }
        break;
      }

      case "unstake": {
        const payload = tx.payload as { amount: bigint };
        if (account.stakedBalance < payload.amount) {
          return {
            valid: false,
            reason: `Insufficient staked balance: have ${account.stakedBalance}, need ${payload.amount}`,
          };
        }
        break;
      }

      case "register_validator": {
        const payload = tx.payload as { stake: bigint };
        if (payload.stake < this.consensusParams.minValidatorStake) {
          return {
            valid: false,
            reason: `Stake below minimum: need ${this.consensusParams.minValidatorStake}`,
          };
        }
        break;
      }

      case "claim_submit": {
        // Check if account can submit claims
        if (!account.flags.canSubmitClaims) {
          return {
            valid: false,
            reason: "Account not authorized to submit claims",
          };
        }
        break;
      }
    }

    // Estimate gas
    const gasEstimate = this.estimateGas(tx);

    return { valid: true, estimatedGas: gasEstimate };
  }

  /**
   * GAP-005 FIX: Execute transaction through state transitions
   */
  async executeTransaction(
    tx: Transaction,
    blockNumber: bigint,
  ): Promise<TransitionResult> {
    return executeTransaction(
      this.stateManager,
      tx,
      blockNumber,
      this.consensusParams,
    );
  }

  /**
   * Apply verification result to state.
   *
   * Routes through completeClaimVerification() which uses the full
   * claim_complete_verification transaction handler. This ensures:
   * - Facts are created with full metadata (domain, stratum, claim text)
   * - Claim references are wired into knowledge graph edges
   * - Submitter stake is properly returned/slashed
   * - Rewards and slashes are distributed atomically
   * - Cache hooks receive correct domain information
   */
  async applyVerificationResult(
    result: VerificationResult,
    blockNumber: bigint,
  ): Promise<void> {
    // Ensure claim exists in state before completing verification.
    // The engine's submitClaim() creates rounds without a state-layer claim_submit,
    // so the claim may not exist in the StateManager. Bootstrap it if missing.
    const existingClaim = this.stateManager.getClaim(result.claimId);
    if (!existingClaim) {
      this.stateManager.addClaim({
        id: result.claimId,
        claim: result.fact?.claim ?? result.claimId,
        claimType: "empirical",
        domain: result.fact?.domain ?? "unknown",
        stratum: (result.fact as any)?.stratum ?? "physical",
        submitter: "system",
        stake: 0n,
        submittedAt: blockNumber,
        references: [],
        status: "in_verification",
        statusChangedAt: blockNumber,
      });
    }

    // Convert Map entries to Arrays for the transaction payload
    const rewards = Array.from(result.rewards.entries());
    const slashes = Array.from(result.slashes.entries());

    await this.completeClaimVerification(
      result.claimId,
      result.claimId, // roundId — used for tracking, not consumed by handler logic
      result.outcome,
      result.finalConfidence,
      result.participatingValidators,
      rewards,
      slashes,
      blockNumber,
    );
  }

  /**
   * GAP-003 FIX: Get transactions from mempool
   */
  collectTransactions(maxCount: number): Transaction[] {
    return this.mempool.getTransactionsByFee(maxCount);
  }

  /**
   * Add transaction to mempool after validation
   */
  addToMempool(tx: Transaction): boolean {
    // Quick validation (full validation happens at block time)
    const account = this.stateManager.getAccount(tx.sender);
    if (!account) return false;
    if (tx.nonce < account.nonce) return false;

    const fee = tx.fee || 0n;
    const gasEstimate = this.estimateGas(tx);

    return this.mempool.add(tx, fee, gasEstimate);
  }

  /**
   * Apply a complete block to state
   */
  async applyBlock(block: Block): Promise<string> {
    this.stateManager.beginTransaction();

    try {
      // Execute all transactions
      for (const tx of block.transactions) {
        const result = await this.executeTransaction(tx, block.header.number);
        if (!result.success) {
          throw new Error(
            `Transaction failed: ${result.error || "Unknown error"}`,
          );
        }

        // Remove from mempool
        this.mempool.remove(tx);
      }

      // Apply verification results
      for (const result of block.verificationResults) {
        await this.applyVerificationResult(result, block.header.number);
      }

      // Finalize block application
      const stateRoot = this.stateManager.applyBlock(
        block.header.number,
        block.header.previousHash,
        block.header.stateRoot,
      );

      this.stateManager.commitTransaction();

      // Persist block and state if storage is available
      if (this.store) {
        await this.persistBlock(block, stateRoot);
      }

      return stateRoot;
    } catch (error) {
      this.stateManager.rollbackTransaction();
      throw error;
    }
  }

  /**
   * Persist block header and state roots to storage
   */
  private async persistBlock(block: Block, stateRoot: string): Promise<void> {
    // Persist block header
    const header: PersistedBlockHeader = {
      number: block.header.number.toString(),
      hash: stateRoot, // Use state root as block identifier for now
      parentHash: block.header.previousHash,
      stateRoot,
      timestamp: block.header.timestamp,
      proposer: block.header.proposer,
    };

    await this.stateManager.persistBlockHeader(header);
    await this.stateManager.persistStateRoots(block.header.number);
  }

  /**
   * Get account balance
   */
  getBalance(did: string): bigint {
    const account = this.stateManager.getAccount(did);
    return account?.balance ?? 0n;
  }

  /**
   * Get account nonce
   */
  getNonce(did: string): bigint {
    const account = this.stateManager.getAccount(did);
    return account?.nonce ?? 0n;
  }

  /**
   * Check if validator is active
   */
  isActiveValidator(publicKey: PublicKey): boolean {
    return this.stateManager.isActiveValidator(publicKey);
  }

  /**
   * Get validator stake
   */
  getValidatorStake(publicKey: PublicKey): bigint {
    const validator = this.stateManager.getValidator(publicKey);
    return validator ? validator.stake + validator.delegatedStake : 0n;
  }

  /**
   * Get total stake in system
   */
  getTotalStake(): bigint {
    return this.stateManager.getTotalStake();
  }

  /**
   * P0-D1 FIX: Get current domain tree from state layer.
   * Ensures engine always uses the latest domain tree (including governance-added domains),
   * instead of a frozen genesis copy.
   */
  getDomainTree(): DomainTree {
    return this.stateManager.getKnowledgeDomainTree();
  }

  /**
   * P0-D2 FIX: Get pending domain proposals as Map for validateClaimOntology.
   * Enables rejection of claims targeting domains that are still in proposal stage.
   */
  getPendingDomainProposals(): Map<string, DomainProposal> {
    const proposals = this.stateManager.getAllDomainProposals();
    const pending = new Map<string, DomainProposal>();
    for (const p of proposals) {
      if (
        p.status === "pending" ||
        p.status === "endorsed" ||
        p.status === "governance"
      ) {
        pending.set(p.proposedDomainId, p);
      }
    }
    return pending;
  }

  /**
   * Estimate gas for a transaction
   */
  private estimateGas(tx: Transaction): bigint {
    // Base gas + type-specific gas
    const baseGas = 21000n;

    const typeGas: Record<string, bigint> = {
      transfer: 0n,
      stake: 10000n,
      unstake: 10000n,
      claim_submit: 50000n,
      verification_commit: 30000n,
      verification_reveal: 30000n,
      add_fact: 100000n,
      register_validator: 50000n,
    };

    return baseGas + (typeGas[tx.type] || 0n);
  }

  // ============ Persistence Methods ============

  /**
   * Initialize from persisted storage
   * Loads the latest state on startup
   */
  async initialize(): Promise<boolean> {
    if (!this.store) {
      return false;
    }

    return this.stateManager.initializeFromStorage();
  }

  /**
   * Check if persistence is enabled
   */
  hasPersistence(): boolean {
    return this.store !== undefined && this.stateManager.hasPersistence();
  }

  /**
   * Get the latest persisted block number
   */
  async getLatestBlockNumber(): Promise<bigint | null> {
    return this.stateManager.getLatestBlockNumber();
  }

  /**
   * Close resources (storage, etc.)
   */
  async close(): Promise<void> {
    if (this.store) {
      await this.store.close();
    }
  }

  // ============ Claim Management ============

  /**
   * Get pending claims ready for verification.
   * Called by consensus to select next claims.
   */
  getPendingClaims(): Array<{
    claimId: string;
    domain: string;
    stratum: string;
    submittedAt: bigint;
    stake: bigint;
  }> {
    return this.stateManager.getPendingClaims().map((claim) => ({
      claimId: claim.id,
      domain: claim.domain,
      stratum: claim.stratum,
      submittedAt: claim.submittedAt,
      stake: claim.stake,
    }));
  }

  /**
   * Start verification for a pending claim.
   * Creates a system transaction to update claim status.
   */
  async startClaimVerification(
    claimId: string,
    roundId: string,
    validators: string[],
    blockNumber: bigint,
  ): Promise<void> {
    const tx = {
      type: "claim_start_verification",
      sender: "system",
      nonce: 0n,
      fee: 0n,
      _systemInternal: true,
      payload: {
        type: "claim_start_verification",
        claimId,
        roundId,
        selectedValidators: validators,
      },
      signature: "",
    } as any as Transaction;

    const result = await executeTransaction(
      this.stateManager,
      tx,
      blockNumber,
      this.consensusParams,
    );
    if (!result.success) {
      throw new Error(`Failed to start claim verification: ${result.error}`);
    }
  }

  /**
   * Complete claim verification with result.
   * Creates fact if accepted, handles stake distribution.
   */
  async completeClaimVerification(
    claimId: string,
    roundId: string,
    outcome: "accepted" | "rejected" | "contested" | "insufficient",
    finalConfidence: number,
    verifierCount: number,
    rewards: Array<[string, bigint]>,
    slashes: Array<[string, bigint]>,
    blockNumber: bigint,
  ): Promise<void> {
    const tx = {
      type: "claim_complete_verification",
      sender: "system",
      nonce: 0n,
      fee: 0n,
      _systemInternal: true,
      payload: {
        type: "claim_complete_verification",
        claimId,
        roundId,
        outcome,
        finalConfidence,
        verifierCount,
        rewards,
        slashes,
      },
      signature: "",
    } as any as Transaction;

    const result = await executeTransaction(
      this.stateManager,
      tx,
      blockNumber,
      this.consensusParams,
    );
    if (!result.success) {
      throw new Error(`Failed to complete claim verification: ${result.error}`);
    }

    // Notify cache hooks if accepted
    if (outcome === "accepted") {
      const claim = this.stateManager.getClaim(claimId);
      if (claim) {
        this.notifyFactIndexed(
          claim.resultFactId!,
          claim.domain,
          finalConfidence,
          blockNumber,
        );
        this.notifyDomainUpdated(claim.domain);
      }
    }
  }

  /**
   * Check if a claim exists and is valid for verification.
   */
  isValidClaim(claimId: string): boolean {
    const claim = this.stateManager.getClaim(claimId);
    return claim !== undefined && claim.status === "pending";
  }
}

// ============ Factory Functions ============

/**
 * Create state integration instance (in-memory, no persistence)
 */
export function createStateIntegration(
  stateManager: StateManager,
  consensusParams: ConsensusParams,
  mempoolConfig?: { maxSize?: number; maxPerAccount?: number },
): StateIntegration {
  return new StateIntegrationImpl(stateManager, consensusParams, mempoolConfig);
}

/**
 * Create state integration instance with persistence
 *
 * @param storePath - Path for LevelDB storage (or ':memory:' for MemoryStore)
 * @param consensusParams - Consensus parameters
 * @param options - Optional mempool and state configuration
 * @returns Initialized StateIntegration with persistence
 */
export async function createPersistedStateIntegration(
  storePath: string,
  consensusParams: ConsensusParams,
  options?: {
    mempoolSize?: number;
    mempoolMaxPerAccount?: number;
    stateConfig?: Record<string, unknown>;
  },
): Promise<StateIntegration> {
  // Create store based on path
  const store: KVStore =
    storePath === ":memory:"
      ? new MemoryStore()
      : await createStorage({ type: "leveldb", path: storePath });

  // Create state manager with store
  const stateManager = new StateManager(options?.stateConfig ?? {}, store);

  // Create integration
  const integration = new StateIntegrationImpl(
    stateManager,
    consensusParams,
    {
      maxSize: options?.mempoolSize,
      maxPerAccount: options?.mempoolMaxPerAccount,
    },
    store,
  );

  // Initialize from storage (load existing state if any)
  await integration.initialize();

  return integration;
}
