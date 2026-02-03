/**
 * State Manager
 *
 * Unified state management combining:
 * - Account state tree
 * - Validator state tree
 * - Knowledge state tree
 *
 * Provides:
 * - Atomic state transitions
 * - Unified state root computation
 * - Rollback capability for failed transactions
 * - State snapshots for sync
 */

import { sha512_256Hex } from '../core/crypto/hashing.js';
import { AccountStateTree } from './accounts.js';
import { ValidatorStateTree } from './validators.js';
import { KnowledgeStateTree } from './knowledge.js';
import { ResearchStateTree } from './research.js';
import { EvidenceStateTree } from './evidence.js';
import { DisputeStateTree } from './disputes.js';
import { VestingStateTree } from './vesting.js';
import {
  type StateRoot,
  type StateRootComponents,
  type StateSnapshot,
  type StateDiff,
  type StateConfig,
  type StateProof,
  type StateSubProof,
  type AccountState,
  type ValidatorState,
  type FactState,
  DEFAULT_STATE_CONFIG,
  StateError,
} from './types.js';
import type { PublicKey } from '../core/crypto/keys.js';
import type { DID } from '../agents/identity/did.js';
import type {
  ResearchState,
  BountyState,
  FactResearchSummary,
} from '../core/research/types.js';
import type {
  Evidence,
  ArchiveLocation,
  ArchiveRequest,
  ChainlinkFeedRegistration,
  ChainlinkFeedType,
} from '../core/evidence/types.js';
import type { EvidenceQuery } from './evidence.js';
import type {
  Dispute,
  DisputePhase,
  DisputeSettlement,
  DomainExpertise,
} from '../core/dispute/types.js';
import type {
  RewardVesting,
  VestingCategory,
  CreateVestingPayload,
} from '../core/rewards/types.js';

// ============ State Manager ============

export class StateManager {
  private accounts: AccountStateTree;
  private validators: ValidatorStateTree;
  private knowledge: KnowledgeStateTree;
  private research: ResearchStateTree;
  private evidence: EvidenceStateTree;
  private disputes: DisputeStateTree;
  private vesting: VestingStateTree;

  private config: StateConfig;
  private version: bigint = 0n;
  private lastBlockHash: string = '0'.repeat(64);

  // Checkpoint for rollback
  private checkpoint: {
    accounts: AccountStateTree;
    validators: ValidatorStateTree;
    knowledge: KnowledgeStateTree;
    research: ResearchStateTree;
    evidence: EvidenceStateTree;
    disputes: DisputeStateTree;
    vesting: VestingStateTree;
    version: bigint;
  } | null = null;

  // Historical state roots (for proving past state)
  private stateHistory: Map<bigint, StateRootComponents> = new Map();

  constructor(config: Partial<StateConfig> = {}) {
    this.config = { ...DEFAULT_STATE_CONFIG, ...config };
    this.accounts = new AccountStateTree(this.config.accountCacheSize);
    this.validators = new ValidatorStateTree();
    this.knowledge = new KnowledgeStateTree();
    this.research = new ResearchStateTree();
    this.evidence = new EvidenceStateTree();
    this.disputes = new DisputeStateTree();
    this.vesting = new VestingStateTree();
  }

  // ============ State Root Computation ============

  /**
   * Compute unified state root from all sub-trees
   */
  getStateRoot(): StateRoot {
    const components = this.getStateRootComponents();
    return this.computeStateRoot(components);
  }

  /**
   * Get individual state root components
   */
  getStateRootComponents(): StateRootComponents {
    return {
      accountRoot: this.accounts.getRoot(),
      validatorRoot: this.validators.getValidatorRoot(),
      knowledgeRoot: this.knowledge.getKnowledgeRoot(),
      researchRoot: this.research.getResearchRoot(),
      evidenceRoot: this.evidence.getRoot(),
      disputeRoot: this.disputes.getRoot(),
      vestingRoot: this.vesting.getRoot(),
      version: this.version,
    };
  }

  /**
   * Compute state root from components
   */
  private computeStateRoot(components: StateRootComponents): StateRoot {
    const data = [
      components.accountRoot,
      components.validatorRoot,
      components.knowledgeRoot,
      components.researchRoot ?? '',
      components.evidenceRoot ?? '',
      components.disputeRoot ?? '',
      components.version.toString(),
    ].join(':');

    return sha512_256Hex(data);
  }

  // ============ Account Access ============

  /**
   * Get account state
   */
  getAccount(did: string): AccountState | null {
    return this.accounts.getAccount(did);
  }

  /**
   * Get account balance
   */
  getBalance(did: string): bigint {
    return this.accounts.getBalance(did);
  }

  /**
   * Check if account exists
   */
  accountExists(did: string): boolean {
    return this.accounts.exists(did);
  }

  /**
   * Get account nonce
   */
  getNonce(did: string): bigint {
    return this.accounts.getNonce(did);
  }

  /**
   * Create account
   */
  createAccount(did: string, operationalKeyHash: string, initialBalance: bigint = 0n): void {
    this.accounts.createAccount(did, operationalKeyHash, initialBalance);
  }

  /**
   * Transfer between accounts
   */
  transfer(from: string, to: string, amount: bigint): void {
    this.accounts.transfer(from, to, amount);
  }

  /**
   * Stake tokens
   */
  stake(did: string, amount: bigint): void {
    this.accounts.stake(did, amount);
  }

  /**
   * Unstake tokens
   */
  unstake(did: string, amount: bigint): void {
    this.accounts.unstake(did, amount);
  }

  // ============ Validator Access ============

  /**
   * Get validator state
   */
  getValidator(publicKey: PublicKey): ValidatorState | null {
    return this.validators.getValidator(publicKey);
  }

  /**
   * Check if address is active validator
   */
  isActiveValidator(publicKey: PublicKey): boolean {
    return this.validators.isActiveValidator(publicKey);
  }

  /**
   * Register a new validator
   */
  registerValidator(publicKey: PublicKey, did: string, stake: bigint): void {
    this.validators.registerValidator(publicKey, did, stake, this.version);
  }

  /**
   * Get total network stake
   */
  getTotalStake(): bigint {
    return this.validators.getTotalStake();
  }

  /**
   * Get active validators
   */
  getActiveValidators(): ValidatorState[] {
    return this.validators.getActiveValidators();
  }

  /**
   * Delegate to validator
   */
  delegate(delegator: string, validator: PublicKey, amount: bigint): void {
    this.validators.delegate(delegator, validator, amount, this.version);
  }

  /**
   * Slash validator
   */
  slashValidator(publicKey: PublicKey, amount: bigint, reason: string): void {
    this.validators.slashValidator(publicKey, amount, reason);
  }

  // ============ Knowledge Access ============

  /**
   * Get fact state
   */
  getFact(factId: string): FactState | null {
    return this.knowledge.getFact(factId);
  }

  /**
   * Check if fact exists and is active
   */
  isActiveFact(factId: string): boolean {
    return this.knowledge.isActiveFact(factId);
  }

  /**
   * Add a new verified fact
   */
  addFact(fact: Omit<FactState, 'referenceRoot'>): void {
    this.knowledge.addFact(fact);
  }

  /**
   * Challenge a fact
   */
  challengeFact(factId: string): void {
    this.knowledge.challengeFact(factId, this.version);
  }

  /**
   * Revoke a fact
   */
  revokeFact(factId: string): void {
    this.knowledge.revokeFact(factId);
  }

  /**
   * Get facts by domain
   */
  getFactsByDomain(domain: string): FactState[] {
    return this.knowledge.getFactsByDomain(domain);
  }

  /**
   * Search facts
   */
  searchFacts(query: string): FactState[] {
    return this.knowledge.searchFacts(query);
  }

  // ============ Research Access ============

  /**
   * Get research state by ID
   */
  getResearch(id: string): ResearchState | null {
    return this.research.getResearch(id);
  }

  /**
   * Add research submission
   */
  addResearch(research: ResearchState): void {
    this.research.addResearch(research);
  }

  /**
   * Update research status
   */
  updateResearch(id: string, update: Partial<ResearchState>): void {
    this.research.updateResearch(id, update);
  }

  /**
   * Get all research for a fact
   */
  getResearchForFact(factId: string): ResearchState[] {
    return this.research.getResearchForFact(factId);
  }

  /**
   * Get research summary for a fact
   */
  getFactResearchSummary(factId: string): FactResearchSummary {
    return this.research.getFactResearchSummary(factId);
  }

  /**
   * Get bounty by ID
   */
  getBounty(id: string): BountyState | null {
    return this.research.getBounty(id);
  }

  /**
   * Add research bounty
   */
  addBounty(bounty: BountyState): void {
    this.research.addBounty(bounty);
  }

  /**
   * Claim a bounty
   */
  claimBounty(id: string, claimer: DID): void {
    this.research.claimBounty(id, claimer);
  }

  /**
   * Fulfill a bounty
   */
  fulfillBounty(id: string, fulfiller: DID): void {
    this.research.fulfillBounty(id, fulfiller);
  }

  /**
   * Get open bounties
   */
  getOpenBounties(): BountyState[] {
    return this.research.getOpenBounties();
  }

  /**
   * Get research count
   */
  getResearchCount(): number {
    return this.research.getResearchCount();
  }

  /**
   * Get bounty count
   */
  getBountyCount(): number {
    return this.research.getBountyCount();
  }

  // ============ Evidence Access ============

  /**
   * Get evidence by ID
   */
  getEvidence(id: string): Evidence | null {
    return this.evidence.getEvidence(id);
  }

  /**
   * Submit new evidence
   */
  submitEvidence(
    evidence: Evidence,
    content: string
  ): {
    success: boolean;
    evidenceId: string;
    isDuplicate: boolean;
    duplicateOf?: string;
    error?: string;
  } {
    return this.evidence.submitEvidence(evidence, content, this.version);
  }

  /**
   * Update existing evidence
   */
  updateEvidence(evidence: Evidence): void {
    this.evidence.updateEvidence(evidence);
  }

  /**
   * Add archive location to evidence
   */
  addArchive(evidenceId: string, archive: ArchiveLocation): void {
    this.evidence.addArchive(evidenceId, archive);
  }

  /**
   * Verify an archive
   */
  verifyArchive(evidenceId: string, archiveIndex: number): void {
    this.evidence.verifyArchive(evidenceId, archiveIndex);
  }

  /**
   * Create new version of evidence
   */
  createEvidenceVersion(
    originalId: string,
    newEvidence: Evidence,
    content: string
  ): string {
    return this.evidence.createVersion(originalId, newEvidence, content, this.version);
  }

  /**
   * Get evidence version chain
   */
  getEvidenceVersionChain(evidenceId: string): Evidence[] {
    return this.evidence.getVersionChain(evidenceId);
  }

  /**
   * Query evidence with filters
   */
  queryEvidence(query: EvidenceQuery): Evidence[] {
    return this.evidence.queryEvidence(query);
  }

  /**
   * Get evidence for a specific fact
   */
  getEvidenceForFact(factId: string): Evidence[] {
    return this.evidence.getEvidenceForFact(factId);
  }

  /**
   * Process evidence expirations
   */
  processEvidenceExpirations(): {
    processed: number;
    newlyExpired: string[];
  } {
    return this.evidence.processExpirations(this.version);
  }

  /**
   * Calculate effective confidence for evidence
   */
  calculateEvidenceConfidence(evidenceId: string): number {
    return this.evidence.calculateEffectiveConfidence(evidenceId, this.version);
  }

  /**
   * Register a Chainlink feed
   */
  registerChainlinkFeed(registration: ChainlinkFeedRegistration): void {
    this.evidence.registerChainlinkFeed(registration);
  }

  /**
   * Get Chainlink feed by address
   */
  getChainlinkFeed(feedAddress: string, chainId: number): ChainlinkFeedRegistration | undefined {
    return this.evidence.getChainlinkFeed(feedAddress, chainId);
  }

  /**
   * Get Chainlink feeds by type
   */
  getChainlinkFeedsByType(feedType: ChainlinkFeedType): ChainlinkFeedRegistration[] {
    return this.evidence.getChainlinkFeedsByType(feedType);
  }

  /**
   * Get next archive request to process
   */
  getNextArchiveRequest(): ArchiveRequest | undefined {
    return this.evidence.getNextArchiveRequest();
  }

  /**
   * Queue archive request
   */
  queueArchiveRequest(request: ArchiveRequest): void {
    this.evidence.queueArchiveRequest(request);
  }

  /**
   * Get evidence statistics
   */
  getEvidenceStats(): ReturnType<EvidenceStateTree['getStatistics']> {
    return this.evidence.getStatistics();
  }

  // ============ Dispute Access ============

  /**
   * Get a dispute by ID
   */
  getDispute(id: string): Dispute | null {
    return this.disputes.getDispute(id);
  }

  /**
   * Create a new dispute
   */
  createDispute(dispute: Dispute): void {
    this.disputes.createDispute(dispute);
  }

  /**
   * Update an existing dispute
   */
  updateDispute(dispute: Dispute): void {
    this.disputes.updateDispute(dispute);
  }

  /**
   * Record dispute settlement (for statistics)
   */
  recordDisputeSettlement(settlement: DisputeSettlement): void {
    this.disputes.recordSettlement(settlement);
  }

  /**
   * Get expertise for a validator
   */
  getValidatorExpertise(validatorDid: string): DomainExpertise[] {
    return this.disputes.getExpertise(validatorDid);
  }

  /**
   * Update expertise for a validator
   */
  updateValidatorExpertise(validatorDid: string, expertise: DomainExpertise[]): void {
    this.disputes.updateExpertise(validatorDid, expertise);
  }

  /**
   * Get all expertise (for arbiter selection)
   */
  getAllExpertise(): Map<string, DomainExpertise[]> {
    return this.disputes.getAllExpertise();
  }

  /**
   * Get all active disputes
   */
  getActiveDisputes(): Dispute[] {
    return this.disputes.getActiveDisputes();
  }

  /**
   * Get disputes by phase
   */
  getDisputesByPhase(phase: DisputePhase): Dispute[] {
    return this.disputes.getDisputesByPhase(phase);
  }

  /**
   * Get disputes by domain
   */
  getDisputesByDomain(domain: string): Dispute[] {
    return this.disputes.getDisputesByDomain(domain);
  }

  /**
   * Get disputes where DID is a party
   */
  getDisputesAsParty(did: string): Dispute[] {
    return this.disputes.getDisputesAsParty(did);
  }

  /**
   * Get disputes where DID is an arbiter
   */
  getDisputesAsArbiter(did: string): Dispute[] {
    return this.disputes.getDisputesAsArbiter(did);
  }

  /**
   * Get disputes targeting a specific evidence/fact/claim
   */
  getDisputesForTarget(targetId: string): Dispute[] {
    return this.disputes.getDisputesForTarget(targetId);
  }

  /**
   * Get disputes that need phase advancement
   */
  getDisputesNeedingAdvancement(currentBlock: bigint): Dispute[] {
    return this.disputes.getDisputesNeedingAdvancement(currentBlock);
  }

  /**
   * Get dispute statistics
   */
  getDisputeStats(): ReturnType<DisputeStateTree['getStatistics']> {
    return this.disputes.getStatistics();
  }

  // ============ Vesting State Operations ============

  /**
   * Create a new vesting schedule for an accepted claim
   */
  createVesting(payload: CreateVestingPayload, currentBlock: bigint): RewardVesting {
    return this.vesting.createVesting(payload, currentBlock);
  }

  /**
   * Get vesting by ID
   */
  getVesting(vestingId: string): RewardVesting | null {
    return this.vesting.getVesting(vestingId);
  }

  /**
   * Get vesting for a specific claim
   */
  getVestingForClaim(claimId: string): RewardVesting | null {
    return this.vesting.getVestingForClaim(claimId);
  }

  /**
   * Claim released rewards from a vesting schedule
   */
  claimVestingRewards(
    vestingId: string,
    claimer: string,
    amount: bigint | undefined,
    currentBlock: bigint
  ): { vesting: RewardVesting; claimedAmount: bigint } {
    return this.vesting.claimRewards(vestingId, claimer, amount, currentBlock);
  }

  /**
   * Pause vesting for a challenge
   */
  pauseVesting(vestingId: string, challengeId: string, currentBlock: bigint): RewardVesting {
    return this.vesting.pauseForChallenge(vestingId, challengeId, currentBlock);
  }

  /**
   * Resume vesting after challenge resolution
   */
  resumeVesting(
    vestingId: string,
    challengerWon: boolean,
    challengerDid: string,
    disputeId: string,
    currentBlock: bigint
  ): { vesting: RewardVesting; clawback?: any } {
    return this.vesting.resumeAfterChallenge(
      vestingId,
      challengerWon,
      challengerDid,
      disputeId,
      currentBlock
    );
  }

  /**
   * Apply acceleration for replication
   */
  accelerateVesting(vestingId: string, researchId: string, currentBlock: bigint): RewardVesting {
    return this.vesting.applyReplication(vestingId, researchId, currentBlock);
  }

  /**
   * Get all vestings for a recipient
   */
  getVestingsForRecipient(recipient: string): RewardVesting[] {
    return this.vesting.getVestingsForRecipient(recipient);
  }

  /**
   * Get vesting summary
   */
  getVestingSummary(vestingId: string, currentBlock: bigint) {
    return this.vesting.getVestingSummary(vestingId, currentBlock);
  }

  /**
   * Batch update releases for all active vestings
   */
  batchUpdateVestingReleases(currentBlock: bigint): number {
    return this.vesting.batchUpdateReleases(currentBlock);
  }

  /**
   * Get vesting statistics
   */
  getVestingStats(): ReturnType<VestingStateTree['getStats']> {
    return this.vesting.getStats();
  }

  // ============ Atomic Operations ============

  /**
   * Begin an atomic transaction
   * Creates a checkpoint that can be rolled back to
   */
  beginTransaction(): void {
    if (this.checkpoint) {
      throw new StateError('Transaction already in progress', 'INVALID_TRANSITION');
    }

    this.checkpoint = {
      accounts: this.accounts.clone(),
      validators: this.validators.clone(),
      knowledge: this.knowledge.clone(),
      research: this.research.clone(),
      evidence: this.evidence.clone(),
      disputes: this.disputes.clone(),
      vesting: this.vesting.clone(),
      version: this.version,
    };
  }

  /**
   * Commit the current transaction
   * Discards checkpoint, changes become permanent
   */
  commitTransaction(): StateRoot {
    if (!this.checkpoint) {
      throw new StateError('No transaction in progress', 'INVALID_TRANSITION');
    }

    this.checkpoint = null;
    this.version++;

    // Store in history for proving past state
    const components = this.getStateRootComponents();
    this.stateHistory.set(this.version, components);

    // Prune old history
    this.pruneHistory();

    return this.getStateRoot();
  }

  /**
   * Rollback to checkpoint
   * Discards all changes since beginTransaction
   */
  rollbackTransaction(): void {
    if (!this.checkpoint) {
      throw new StateError('No transaction in progress', 'ROLLBACK_FAILED');
    }

    this.accounts = this.checkpoint.accounts;
    this.validators = this.checkpoint.validators;
    this.knowledge = this.checkpoint.knowledge;
    this.research = this.checkpoint.research;
    this.evidence = this.checkpoint.evidence;
    this.disputes = this.checkpoint.disputes;
    this.vesting = this.checkpoint.vesting;
    this.version = this.checkpoint.version;
    this.checkpoint = null;
  }

  /**
   * Check if transaction is in progress
   */
  hasActiveTransaction(): boolean {
    return this.checkpoint !== null;
  }

  // ============ Block Application ============

  /**
   * Apply a block to state
   * This is the main entry point for state updates
   */
  applyBlock(
    blockNumber: bigint,
    blockHash: string,
    expectedStateRoot?: StateRoot
  ): StateRoot {
    // Increment version
    this.version = blockNumber;
    this.lastBlockHash = blockHash;

    // Compute new state root
    const newRoot = this.getStateRoot();

    // Verify against expected if provided
    if (expectedStateRoot && newRoot !== expectedStateRoot) {
      throw new StateError('State root mismatch', 'STATE_MISMATCH', {
        expected: expectedStateRoot,
        computed: newRoot,
      });
    }

    // Store in history
    this.stateHistory.set(this.version, this.getStateRootComponents());
    this.pruneHistory();

    return newRoot;
  }

  /**
   * Get current block number (version)
   */
  getBlockNumber(): bigint {
    return this.version;
  }

  // ============ State Proofs ============

  /**
   * Generate comprehensive state proof
   */
  generateStateProof(queries: Array<{
    tree: 'account' | 'validator' | 'knowledge' | 'research' | 'evidence';
    key: string;
  }>): StateProof {
    const proofs: StateSubProof[] = [];

    for (const query of queries) {
      let proof;

      switch (query.tree) {
        case 'account':
          proof = this.accounts.proveAccount(query.key);
          break;
        case 'validator':
          proof = this.validators.proveValidator(query.key as PublicKey);
          break;
        case 'knowledge':
          proof = this.knowledge.proveFact(query.key, this.version).proof;
          break;
        case 'research':
          proof = this.research.proveResearch(query.key);
          break;
        case 'evidence':
          // Note: Evidence proofs return simplified proof object pending full SMT implementation
          proof = this.evidence.getEvidenceProof(query.key) as unknown as typeof proof;
          break;
      }

      proofs.push({
        tree: query.tree,
        key: query.key,
        proof,
      });
    }

    return {
      root: this.getStateRoot(),
      blockNumber: this.version,
      proofs,
    };
  }

  // ============ Snapshots ============

  /**
   * Create state snapshot
   */
  createSnapshot(): StateSnapshot {
    const roots = this.getStateRootComponents();

    return {
      version: this.version,
      blockNumber: this.version,
      blockHash: this.lastBlockHash,
      roots,
      accountCount: this.accounts.getAllAccounts().length,
      validatorCount: this.validators.getActiveCount(),
      factCount: this.knowledge.getFactCount(),
      researchCount: this.research.getResearchCount(),
      bountyCount: this.research.getBountyCount(),
      evidenceCount: this.evidence.getStatistics().totalEvidence,
      checksum: sha512_256Hex(JSON.stringify(roots)),
    };
  }

  /**
   * Get state diff between two blocks
   */
  getStateDiff(fromBlock: bigint, toBlock: bigint): StateDiff | null {
    const fromRoots = this.stateHistory.get(fromBlock);
    const toRoots = this.stateHistory.get(toBlock);

    if (!fromRoots || !toRoots) {
      return null;
    }

    // In production, this would compute actual diffs
    // For now, return structure
    return {
      fromBlock,
      toBlock,
      fromRoot: this.computeStateRoot(fromRoots),
      toRoot: this.computeStateRoot(toRoots),
      accountChanges: [],
      validatorChanges: [],
      factChanges: [],
      transitions: [],
    };
  }

  // ============ Statistics ============

  /**
   * Get state statistics
   */
  getStats(): StateStats {
    const evidenceStats = this.evidence.getStatistics();
    return {
      version: this.version,
      stateRoot: this.getStateRoot(),
      accounts: {
        count: this.accounts.getAllAccounts().length,
        totalSupply: this.accounts.getTotalSupply(),
      },
      validators: {
        count: this.validators.getActiveCount(),
        totalStake: this.validators.getTotalStake(),
      },
      knowledge: {
        factCount: this.knowledge.getFactCount(),
        activeFactCount: this.knowledge.getActiveFactCount(),
        domainCount: this.knowledge.getDomains().length,
      },
      research: {
        researchCount: this.research.getResearchCount(),
        bountyCount: this.research.getBountyCount(),
        openBountyCount: this.research.getOpenBounties().length,
      },
      evidence: {
        totalEvidence: evidenceStats.totalEvidence,
        totalArchives: evidenceStats.totalArchives,
        verifiedArchives: evidenceStats.verifiedArchives,
        pendingArchiveRequests: evidenceStats.pendingArchiveRequests,
        chainlinkFeeds: evidenceStats.chainlinkFeeds,
      },
    };
  }

  // ============ Private Methods ============

  private pruneHistory(): void {
    const cutoff = this.version - BigInt(this.config.keepHistoricalStates);
    for (const block of this.stateHistory.keys()) {
      if (block < cutoff) {
        this.stateHistory.delete(block);
      }
    }
  }
}

// ============ Types ============

export interface StateStats {
  version: bigint;
  stateRoot: StateRoot;
  accounts: {
    count: number;
    totalSupply: bigint;
  };
  validators: {
    count: number;
    totalStake: bigint;
  };
  knowledge: {
    factCount: number;
    activeFactCount: number;
    domainCount: number;
  };
  research: {
    researchCount: number;
    bountyCount: number;
    openBountyCount: number;
  };
  evidence: {
    totalEvidence: number;
    totalArchives: number;
    verifiedArchives: number;
    pendingArchiveRequests: number;
    chainlinkFeeds: number;
  };
}

// ============ Factory Functions ============

/**
 * Create state manager from genesis configuration
 */
export function createStateFromGenesis(genesis: GenesisConfig): StateManager {
  const manager = new StateManager();

  // Create initial accounts
  for (const account of genesis.accounts) {
    manager.createAccount(account.did, account.operationalKeyHash, account.balance);
  }

  // Create initial validators
  for (const validator of genesis.validators) {
    manager.registerValidator(validator.publicKey, validator.did, validator.stake);
  }

  // Add genesis axioms (foundational facts)
  for (const axiom of genesis.axioms) {
    manager.addFact({
      id: axiom.id,
      claim: axiom.claim,
      domain: axiom.domain,
      confidence: 1.0,
      verifiedAt: 0n,
      verifierCount: 0,
      challengeCount: 0,
      status: 'active',
    });
  }

  return manager;
}

export interface GenesisConfig {
  accounts: Array<{
    did: string;
    operationalKeyHash: string;
    balance: bigint;
  }>;
  validators: Array<{
    publicKey: PublicKey;
    did: string;
    stake: bigint;
  }>;
  axioms: Array<{
    id: string;
    claim: string;
    domain: string;
  }>;
}

/**
 * Default genesis for development/testing
 */
export const DEV_GENESIS: GenesisConfig = {
  accounts: [],
  validators: [],
  axioms: [
    {
      id: 'axiom:logic:identity',
      claim: 'A = A (Law of Identity)',
      domain: 'logic.foundational',
    },
    {
      id: 'axiom:logic:non-contradiction',
      claim: '¬(A ∧ ¬A) (Law of Non-Contradiction)',
      domain: 'logic.foundational',
    },
    {
      id: 'axiom:logic:excluded-middle',
      claim: 'A ∨ ¬A (Law of Excluded Middle)',
      domain: 'logic.foundational',
    },
    {
      id: 'axiom:math:successor',
      claim: 'Every natural number has a successor',
      domain: 'math.arithmetic',
    },
    {
      id: 'axiom:math:zero',
      claim: 'Zero is a natural number',
      domain: 'math.arithmetic',
    },
  ],
};
