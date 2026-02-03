/**
 * Test Harness
 *
 * Framework for running integration tests on Legible Money.
 * Provides:
 * - Multi-node test networks
 * - State assertions
 * - Transaction builders
 * - Mock validators
 *
 * PP HIERARCHY:
 *   TRUTH: Tests verify the protocol behaves correctly
 *   UNDERSTANDING: Clear test APIs and assertions
 *   BEAUTY: Elegant test composition
 *   JUSTICE: Fair test coverage across components
 *   CREATIVITY: Flexible for complex test scenarios
 */

import { EventEmitter } from 'events';
import { StateManager, createStateFromGenesis } from '../../state/state_manager.js';
import { ProofOfTruthEngine } from '../../consensus/pot/engine.js';
import { Mempool } from '../../consensus/mempool/pool.js';
import { MemoryStore } from '../../storage/leveldb.js';
import { generateKeyPair, type PublicKey, type PrivateKey } from '../../core/crypto/keys.js';
import { sha512_256Hex, domainHash } from '../../core/crypto/hashing.js';
import type { Transaction, Block, ConsensusParams } from '../../consensus/pot/types.js';
import type { DID } from '../../agents/identity/did.js';

// ============ Test Network ============

/**
 * Configuration for test network
 */
export interface TestNetworkConfig {
  /** Number of validators */
  validatorCount: number;

  /** Number of non-validator nodes */
  fullNodeCount: number;

  /** Initial token supply */
  totalSupply: bigint;

  /** Validator stake amount */
  validatorStake: bigint;

  /** Test accounts to create */
  testAccounts: number;

  /** Custom consensus params */
  consensusParams?: Partial<ConsensusParams>;

  /** Enable automatic block production */
  autoBlocks: boolean;

  /** Block production interval (ms) */
  blockInterval: number;
}

export const DEFAULT_TEST_CONFIG: TestNetworkConfig = {
  validatorCount: 3,
  fullNodeCount: 0,
  totalSupply: 1_000_000_000n * 10n ** 18n,
  validatorStake: 10_000n * 10n ** 18n,
  testAccounts: 5,
  autoBlocks: false,
  blockInterval: 100, // Fast for tests
};

/**
 * A test validator node
 */
export interface TestValidator {
  name: string;
  publicKey: PublicKey;
  privateKey: PrivateKey;
  did: DID;
  stake: bigint;
}

/**
 * A test account
 */
export interface TestAccount {
  name: string;
  publicKey: PublicKey;
  privateKey: PrivateKey;
  did: DID;
  balance: bigint;
}

/**
 * Test network for integration testing
 */
export class TestNetwork extends EventEmitter {
  private config: TestNetworkConfig;
  private state: StateManager | null = null;
  private consensus: ProofOfTruthEngine | null = null;
  private mempool: Mempool | null = null;
  private store: MemoryStore | null = null;

  public validators: TestValidator[] = [];
  public accounts: TestAccount[] = [];

  private blockTimer: ReturnType<typeof setInterval> | null = null;
  private currentBlock = 0n;
  private isRunning = false;

  constructor(config: Partial<TestNetworkConfig> = {}) {
    super();
    this.config = { ...DEFAULT_TEST_CONFIG, ...config };
  }

  /**
   * Initialize the test network
   */
  async init(): Promise<void> {
    // Generate validators
    this.validators = this.generateValidators(this.config.validatorCount);

    // Generate test accounts
    this.accounts = this.generateAccounts(this.config.testAccounts);

    // Initialize storage
    this.store = new MemoryStore();

    // Initialize state
    this.state = createStateFromGenesis({
      accounts: [
        // Validators
        ...this.validators.map(v => ({
          did: v.did,
          operationalKeyHash: v.publicKey,
          balance: v.stake * 2n, // Give extra for operations
        })),
        // Test accounts
        ...this.accounts.map(a => ({
          did: a.did,
          operationalKeyHash: a.publicKey,
          balance: a.balance,
        })),
      ],
      validators: this.validators.map(v => ({
        publicKey: v.publicKey,
        did: v.did,
        stake: v.stake,
      })),
      axioms: [
        { id: 'test:axiom:1', claim: 'Test axiom for testing', domain: 'test' },
      ],
    });

    // Initialize mempool
    this.mempool = new Mempool({ maxSize: 1000, maxPerSender: 100 });

    // Initialize consensus
    this.consensus = new ProofOfTruthEngine(
      this.state,
      this.mempool,
      {
        targetBlockTimeMs: this.config.blockInterval,
        verificationRoundBlocks: 3, // Fast for tests
        commitPhaseBlocks: 1,
        revealPhaseBlocks: 1,
        minValidators: Math.min(this.config.validatorCount, 3),
        maxValidators: 21,
        validatorQuorum: 0.66,
        acceptanceThreshold: 0.66,
        rejectionThreshold: 0.66,
        minValidatorStake: 1000n * 10n ** 18n,
        minClaimStake: 10n * 10n ** 18n,
        blockReward: 1n * 10n ** 18n,
        verificationReward: 100_000_000_000_000_000n,
        ...this.config.consensusParams,
      } as ConsensusParams,
      this.validators[0]?.publicKey // First validator produces blocks
    );
  }

  /**
   * Start the test network
   */
  async start(): Promise<void> {
    if (this.isRunning) return;

    this.mempool?.start();
    await this.consensus?.start();

    if (this.config.autoBlocks) {
      this.startAutoBlocks();
    }

    this.isRunning = true;
    this.emit('started');
  }

  /**
   * Stop the test network
   */
  async stop(): Promise<void> {
    if (!this.isRunning) return;

    this.stopAutoBlocks();
    await this.consensus?.stop();
    this.mempool?.stop();

    this.isRunning = false;
    this.emit('stopped');
  }

  /**
   * Produce a block manually
   */
  async produceBlock(): Promise<Block | null> {
    if (!this.consensus) return null;

    const block = await this.consensus.produceBlock();
    if (block) {
      this.currentBlock = BigInt(block.header.number);
      this.emit('block', block);
    }
    return block;
  }

  /**
   * Mine N blocks
   */
  async mineBlocks(count: number): Promise<Block[]> {
    const blocks: Block[] = [];
    for (let i = 0; i < count; i++) {
      const block = await this.produceBlock();
      if (block) blocks.push(block);
    }
    return blocks;
  }

  /**
   * Submit a transaction
   */
  async submitTransaction(tx: Transaction): Promise<string> {
    if (!this.mempool) throw new Error('Network not initialized');

    const entry = await this.mempool.add(tx);
    this.emit('transaction', tx);
    return entry.txHash;
  }

  /**
   * Submit a claim for verification
   */
  async submitClaim(
    claim: string,
    domain: string,
    submitter: TestAccount | TestValidator,
    stake: bigint = 100n * 10n ** 18n
  ): Promise<string> {
    if (!this.consensus) throw new Error('Network not initialized');

    const claimId = await this.consensus.submitClaim({
      id: '',
      claim,
      domain,
      submitter: submitter.publicKey,
      stake,
      submittedAt: Date.now(),
      references: [],
    });

    this.emit('claim', { claimId, claim, domain });
    return claimId;
  }

  /**
   * Wait for a claim to be verified
   */
  async waitForVerification(claimId: string, timeoutBlocks = 20): Promise<'accepted' | 'rejected' | 'timeout'> {
    const startBlock = this.currentBlock;

    return new Promise((resolve) => {
      const checkInterval = setInterval(async () => {
        // Check if round completed
        const fact = this.state?.getFact(claimId);
        if (fact) {
          clearInterval(checkInterval);
          resolve('accepted');
          return;
        }

        // Check timeout
        if (this.currentBlock - startBlock > BigInt(timeoutBlocks)) {
          clearInterval(checkInterval);
          resolve('timeout');
          return;
        }

        // Produce block if not auto
        if (!this.config.autoBlocks) {
          await this.produceBlock();
        }
      }, 10);
    });
  }

  // ============ Getters ============

  getState(): StateManager {
    if (!this.state) throw new Error('Network not initialized');
    return this.state;
  }

  getConsensus(): ProofOfTruthEngine {
    if (!this.consensus) throw new Error('Network not initialized');
    return this.consensus;
  }

  getMempool(): Mempool {
    if (!this.mempool) throw new Error('Network not initialized');
    return this.mempool;
  }

  getCurrentBlock(): bigint {
    return this.currentBlock;
  }

  getValidator(index: number): TestValidator {
    return this.validators[index];
  }

  getAccount(index: number): TestAccount {
    return this.accounts[index];
  }

  // ============ Private Methods ============

  private generateValidators(count: number): TestValidator[] {
    const validators: TestValidator[] = [];

    for (let i = 0; i < count; i++) {
      const seed = domainHash('LGM.test.validator.v1', `validator-${i}`);
      const { publicKey, privateKey } = generateKeyPair(seed.slice(0, 32));

      validators.push({
        name: `validator-${i}`,
        publicKey,
        privateKey,
        did: `did:lgm-test:validator-${i}` as DID,
        stake: this.config.validatorStake,
      });
    }

    return validators;
  }

  private generateAccounts(count: number): TestAccount[] {
    const accounts: TestAccount[] = [];
    const balancePerAccount = this.config.totalSupply / BigInt(count + this.config.validatorCount * 3);

    for (let i = 0; i < count; i++) {
      const seed = domainHash('LGM.test.account.v1', `account-${i}`);
      const { publicKey, privateKey } = generateKeyPair(seed.slice(0, 32));

      accounts.push({
        name: `account-${i}`,
        publicKey,
        privateKey,
        did: `did:lgm-test:account-${i}` as DID,
        balance: balancePerAccount,
      });
    }

    return accounts;
  }

  private startAutoBlocks(): void {
    this.blockTimer = setInterval(async () => {
      await this.produceBlock();
    }, this.config.blockInterval);
  }

  private stopAutoBlocks(): void {
    if (this.blockTimer) {
      clearInterval(this.blockTimer);
      this.blockTimer = null;
    }
  }
}

// ============ Assertions ============

/**
 * Test assertions for Legible Money state
 */
export class StateAssertions {
  constructor(private network: TestNetwork) {}

  /**
   * Assert account balance
   */
  balanceEquals(account: TestAccount | DID, expected: bigint): void {
    const did = typeof account === 'string' ? account : account.did;
    const state = this.network.getState();
    const actual = state.getAccount(did)?.balance ?? 0n;

    if (actual !== expected) {
      throw new AssertionError(
        `Balance mismatch for ${did}: expected ${expected}, got ${actual}`
      );
    }
  }

  /**
   * Assert account balance is at least
   */
  balanceAtLeast(account: TestAccount | DID, minimum: bigint): void {
    const did = typeof account === 'string' ? account : account.did;
    const state = this.network.getState();
    const actual = state.getAccount(did)?.balance ?? 0n;

    if (actual < minimum) {
      throw new AssertionError(
        `Balance too low for ${did}: expected at least ${minimum}, got ${actual}`
      );
    }
  }

  /**
   * Assert fact exists
   */
  factExists(factId: string): void {
    const state = this.network.getState();
    const fact = state.getFact(factId);

    if (!fact) {
      throw new AssertionError(`Fact not found: ${factId}`);
    }
  }

  /**
   * Assert fact has minimum confidence
   */
  factConfidenceAtLeast(factId: string, minConfidence: number): void {
    const state = this.network.getState();
    const fact = state.getFact(factId);

    if (!fact) {
      throw new AssertionError(`Fact not found: ${factId}`);
    }

    if (fact.confidence < minConfidence) {
      throw new AssertionError(
        `Fact confidence too low: expected at least ${minConfidence}, got ${fact.confidence}`
      );
    }
  }

  /**
   * Assert validator is active
   */
  validatorActive(validator: TestValidator | PublicKey): void {
    const pubKey = typeof validator === 'string' ? validator : validator.publicKey;
    const state = this.network.getState();
    const v = state.getValidator(pubKey);

    if (!v || !v.isActive) {
      throw new AssertionError(`Validator not active: ${pubKey.slice(0, 16)}...`);
    }
  }

  /**
   * Assert block number
   */
  blockNumberEquals(expected: bigint): void {
    const actual = this.network.getCurrentBlock();

    if (actual !== expected) {
      throw new AssertionError(
        `Block number mismatch: expected ${expected}, got ${actual}`
      );
    }
  }

  /**
   * Assert block number is at least
   */
  blockNumberAtLeast(minimum: bigint): void {
    const actual = this.network.getCurrentBlock();

    if (actual < minimum) {
      throw new AssertionError(
        `Block number too low: expected at least ${minimum}, got ${actual}`
      );
    }
  }
}

class AssertionError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'AssertionError';
  }
}

// ============ Transaction Builders ============

/**
 * Builders for creating test transactions
 */
export class TransactionBuilder {
  private network: TestNetwork;

  constructor(network: TestNetwork) {
    this.network = network;
  }

  /**
   * Build a transfer transaction
   */
  transfer(
    from: TestAccount,
    to: TestAccount | DID,
    amount: bigint,
    memo?: string
  ): Transaction {
    const toDid = typeof to === 'string' ? to : to.did;
    const nonce = this.getNextNonce(from.did);

    return {
      type: 'transfer',
      sender: from.publicKey,
      fee: 1_000_000_000_000_000n,
      nonce,
      signature: '', // Would be signed
      data: {
        to: toDid,
        amount,
        memo,
      },
    } as Transaction;
  }

  /**
   * Build a stake transaction
   */
  stake(account: TestAccount, amount: bigint): Transaction {
    const nonce = this.getNextNonce(account.did);

    return {
      type: 'stake',
      sender: account.publicKey,
      fee: 1_000_000_000_000_000n,
      nonce,
      signature: '',
      data: {
        amount,
      },
    } as Transaction;
  }

  /**
   * Build a claim submission transaction
   */
  submitClaim(
    submitter: TestAccount,
    claim: string,
    domain: string,
    stake: bigint,
    references: string[] = []
  ): Transaction {
    const nonce = this.getNextNonce(submitter.did);

    return {
      type: 'submit_claim',
      sender: submitter.publicKey,
      fee: 1_000_000_000_000_000n,
      nonce,
      signature: '',
      data: {
        claim,
        domain,
        stake,
        references,
      },
    } as Transaction;
  }

  private getNextNonce(did: DID): bigint {
    const account = this.network.getState().getAccount(did);
    return (account?.nonce ?? 0n) + 1n;
  }
}

// ============ Test Utilities ============

/**
 * Sleep for a duration
 */
export function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Create a test network with default configuration
 */
export async function createTestNetwork(
  config: Partial<TestNetworkConfig> = {}
): Promise<TestNetwork> {
  const network = new TestNetwork(config);
  await network.init();
  await network.start();
  return network;
}

/**
 * Run a test with automatic cleanup
 */
export async function withTestNetwork<T>(
  config: Partial<TestNetworkConfig>,
  fn: (network: TestNetwork, assert: StateAssertions, tx: TransactionBuilder) => Promise<T>
): Promise<T> {
  const network = await createTestNetwork(config);

  try {
    const assert = new StateAssertions(network);
    const tx = new TransactionBuilder(network);
    return await fn(network, assert, tx);
  } finally {
    await network.stop();
  }
}
