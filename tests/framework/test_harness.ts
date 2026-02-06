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

import { EventEmitter } from "events";
import {
  StateManager,
  createStateFromGenesis,
} from "../../state/state_manager.js";
import {
  ProofOfTruthEngine,
  createIntegratedEngine,
} from "../../consensus/pot/engine.js";
import { DEFAULT_CONSENSUS_PARAMS } from "../../consensus/pot/types.js";
import { Mempool } from "../../consensus/mempool/pool.js";
import { MemoryStore } from "../../state/storage/leveldb.js";
import {
  generateKeypairFromSeed,
  type PublicKey,
  type PrivateKey,
} from "../../core/crypto/keys.js";
import { sha512_256Hex, domainHash } from "../../core/crypto/hashing.js";
import type {
  Transaction,
  Block,
  ConsensusParams,
} from "../../consensus/pot/types.js";
import type { DID } from "../../core/agents/identity/did.js";

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

  /**
   * Enable deterministic test mode
   * When true:
   * - All validators are always selected (bypasses VRF lottery)
   * - Block production always succeeds
   * - Phase transitions are predictable
   */
  deterministicMode: boolean;
}

export const DEFAULT_TEST_CONFIG: TestNetworkConfig = {
  validatorCount: 3,
  fullNodeCount: 0,
  totalSupply: 1_000_000_000n * 10n ** 18n,
  validatorStake: 10_000n * 10n ** 18n,
  testAccounts: 5,
  autoBlocks: false,
  blockInterval: 100, // Fast for tests
  deterministicMode: true, // Enable deterministic mode by default for tests
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
        ...this.validators.map((v) => ({
          did: v.did,
          operationalKeyHash: v.publicKey,
          balance: v.stake * 2n, // Give extra for operations
        })),
        // Test accounts
        ...this.accounts.map((a) => ({
          did: a.did,
          operationalKeyHash: a.publicKey,
          balance: a.balance,
        })),
      ],
      validators: this.validators.map((v) => ({
        publicKey: v.publicKey,
        did: v.did,
        stake: v.stake,
      })),
      axioms: [
        { id: "test:axiom:1", claim: "Test axiom for testing", domain: "test" },
      ],
    });

    // Initialize mempool
    this.mempool = new Mempool({ maxSize: 1000, maxPerSender: 100 });

    // Build validator info
    const validatorInfo = this.validators.map((v) => ({
      publicKey: v.publicKey,
      did: v.did,
      stake: v.stake,
    }));

    // First validator is the block producer
    const blockProducer = this.validators[0];

    // Build consensus params - merge defaults with test config
    const consensusParams = {
      ...DEFAULT_CONSENSUS_PARAMS,
      targetBlockTimeMs: this.config.blockInterval,
      verificationRoundBlocks: 6,
      commitPhaseBlocks: 2,
      revealPhaseBlocks: 2,
      minValidators: Math.min(this.config.validatorCount, 3),
      maxValidators: 21,
      validatorQuorum: 0.66,
      acceptanceThreshold: 0.66,
      rejectionThreshold: 0.66,
      minValidatorStake: 1000n * 10n ** 18n,
      minClaimStake: 10n * 10n ** 18n,
      blockReward: 1n * 10n ** 18n,
      verificationReward: 1n * 10n ** 18n,
      ...this.config.consensusParams,
    };

    // Initialize consensus with FULL state integration
    // This ensures facts are recorded when verification completes
    this.consensus = createIntegratedEngine({
      isValidator: true,
      validatorKey: blockProducer.privateKey,
      validatorPublicKey: blockProducer.publicKey,
      initialValidators: validatorInfo,
      stateManager: this.state,
      params: consensusParams,
    });
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
    this.emit("started");
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
    this.emit("stopped");
  }

  /**
   * Produce a block manually
   * In deterministic mode, always uses forced block production.
   * In non-deterministic mode, tries VRF selection first.
   */
  async produceBlock(): Promise<Block | null> {
    if (!this.consensus) return null;

    // In deterministic mode, skip VRF attempts entirely
    // This prevents collectVerificationResults from prematurely removing rounds
    if (this.config.deterministicMode) {
      return this.forceProduceBlock();
    }

    // Try to produce a block - VRF selection is probabilistic
    // In tests, we may need multiple attempts
    for (let attempt = 0; attempt < 10; attempt++) {
      const block = await this.consensus.tryProduceBlock();
      if (block) {
        this.currentBlock = BigInt(block.header.number);
        this.emit("block", block);
        return block;
      }
      // Small delay between attempts to change the timestamp-based seed
      await new Promise((resolve) => setTimeout(resolve, 5));
    }

    // Force block production for tests by manually building
    // This is a fallback for when VRF doesn't select us
    return this.forceProduceBlock();
  }

  /**
   * Force produce a block for testing (bypasses VRF selection)
   * Uses consensus engine's test method for proper state updates
   */
  private async forceProduceBlock(): Promise<Block | null> {
    if (!this.consensus || !this.state) return null;

    // Use engine's test method to advance block properly
    // This updates state, finality tracking, and processes verification rounds
    const newBlockNumber = await this.consensus.advanceBlockForTesting();
    this.currentBlock = newBlockNumber;

    // Get updated state for block construction
    const consensusState = this.consensus.getState();

    // Create a properly formed test block (for return value)
    const testBlock: Block = {
      header: {
        number: newBlockNumber,
        timestamp: Date.now(),
        previousHash: consensusState.lastBlockHash,
        stateRoot: sha512_256Hex(`state-${newBlockNumber}`),
        knowledgeRoot: sha512_256Hex("knowledge"),
        transactionsRoot: sha512_256Hex("txs"),
        verificationsRoot: sha512_256Hex("verifications"),
        proposer: this.validators[0].publicKey,
        proposerVrfOutput: "test-vrf-output",
        proposerVrfProof: "test-vrf-proof",
      },
      transactions: [],
      verificationResults: [],
      signature: "test-signature",
      attestations: [],
    };

    this.emit("block", testBlock);
    return testBlock;
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
    if (!this.mempool) throw new Error("Network not initialized");

    const entry = await this.mempool.add(tx);
    this.emit("transaction", tx);
    return entry.txHash;
  }

  /**
   * Submit a claim for verification
   */
  async submitClaim(
    claim: string,
    domain: string,
    submitter: TestAccount | TestValidator,
    stake: bigint = 100n * 10n ** 18n,
  ): Promise<string> {
    if (!this.consensus) throw new Error("Network not initialized");

    const claimId = await this.consensus.submitClaim({
      id: "",
      claim,
      domain,
      submitter: submitter.publicKey,
      stake,
      submittedAt: Date.now(),
      references: [
        {
          id: "test-ref-1",
          type: "supports",
          source: `https://test.example.com/${domain}`,
          verified: false,
        },
      ],
    });

    this.emit("claim", { claimId, claim, domain });
    return claimId;
  }

  /**
   * Wait for a claim to be verified
   */
  async waitForVerification(
    claimId: string,
    timeoutBlocks = 20,
  ): Promise<"accepted" | "rejected" | "timeout"> {
    const startBlock = this.currentBlock;

    return new Promise((resolve) => {
      const checkInterval = setInterval(async () => {
        // Check if round completed (fact ID may be claimId or "fact:claimId")
        const fact =
          this.state?.getFact(claimId) ??
          this.state?.getFact(`fact:${claimId}`);
        if (fact) {
          clearInterval(checkInterval);
          resolve("accepted");
          return;
        }

        // Check timeout
        if (this.currentBlock - startBlock > BigInt(timeoutBlocks)) {
          clearInterval(checkInterval);
          resolve("timeout");
          return;
        }

        // Produce block if not auto
        if (!this.config.autoBlocks) {
          await this.produceBlock();
        }
      }, 10);
    });
  }

  // ============ Deterministic Mode Helpers ============

  /**
   * Check if deterministic mode is enabled
   */
  isDeterministicMode(): boolean {
    return this.config.deterministicMode;
  }

  /**
   * Submit commitments from validators in deterministic mode
   * Bypasses VRF verification for predictable test results
   *
   * @returns Map of validator public keys to their nonces (for reveal phase)
   */
  async submitCommitmentsDeterministic(
    roundId: string,
    validatorCommitments: Array<{
      validator: TestValidator;
      verdict: "confirmed" | "rejected" | "uncertain";
      confidence: number;
      referencesChecked?: string[];
    }>,
  ): Promise<Map<string, string>> {
    if (!this.consensus) throw new Error("Network not initialized");

    const nonces = new Map<string, string>();

    // Import generateCommitment dynamically to avoid circular deps at top level
    const { generateCommitment } =
      await import("../../consensus/pot/verification_round.js");

    for (const {
      validator,
      verdict,
      confidence,
      referencesChecked = ["ref-1"],
    } of validatorCommitments) {
      const { commitment, nonce } = generateCommitment(
        validator.privateKey,
        validator.publicKey,
        roundId,
        verdict,
        confidence,
        referencesChecked,
      );

      nonces.set(validator.publicKey, nonce);

      try {
        // Use deterministic submission (bypasses VRF)
        await this.consensus.submitCommitmentDeterministic(roundId, {
          validator: validator.publicKey,
          commitment: commitment.commitment,
          signature: commitment.signature,
        });
      } catch (error) {
        console.log(
          `[TEST:DETERMINISTIC] Commitment failed for ${validator.name}:`,
          (error as Error).message,
        );
      }
    }

    return nonces;
  }

  /**
   * Submit reveals from validators in deterministic mode
   */
  async submitRevealsDeterministic(
    roundId: string,
    validatorReveals: Array<{
      validator: TestValidator;
      verdict: "confirmed" | "rejected" | "uncertain";
      confidence: number;
      nonce: string;
      referencesChecked?: string[];
      reasoning?: string;
    }>,
  ): Promise<void> {
    if (!this.consensus) throw new Error("Network not initialized");

    // Import generateReveal dynamically
    const { generateReveal } =
      await import("../../consensus/pot/verification_round.js");

    for (const {
      validator,
      verdict,
      confidence,
      nonce,
      referencesChecked = ["ref-1"],
      reasoning,
    } of validatorReveals) {
      const reveal = generateReveal(
        validator.privateKey,
        validator.publicKey,
        roundId,
        verdict,
        confidence,
        nonce,
        referencesChecked,
        reasoning,
      );

      try {
        await this.consensus.submitReveal(roundId, {
          validator: validator.publicKey,
          verdict: reveal.verdict,
          confidence: reveal.confidence,
          nonce: reveal.nonce,
          referencesChecked: reveal.referencesChecked,
          signature: reveal.signature,
        });
      } catch (error) {
        console.log(
          `[TEST:DETERMINISTIC] Reveal failed for ${validator.name}:`,
          (error as Error).message,
        );
      }
    }
  }

  /**
   * Complete a full verification round deterministically
   * Handles commit phase, mining to reveal phase, and reveal phase
   */
  async completeVerificationRoundDeterministic(
    roundId: string,
    validatorVerdicts: Array<{
      validator: TestValidator;
      verdict: "confirmed" | "rejected" | "uncertain";
      confidence: number;
    }>,
  ): Promise<void> {
    if (!this.consensus) throw new Error("Network not initialized");

    const round = this.consensus
      .getActiveRounds()
      .find((r) => r.id === roundId);
    if (!round) throw new Error(`Round ${roundId} not found`);

    // 1. Submit all commitments
    const nonces = await this.submitCommitmentsDeterministic(
      roundId,
      validatorVerdicts,
    );

    // 2. Mine to reveal phase
    const blocksToReveal = round.commitDeadline - round.startBlock + 1;
    await this.mineBlocks(blocksToReveal);

    // 3. Submit all reveals
    await this.submitRevealsDeterministic(
      roundId,
      validatorVerdicts.map((v) => ({
        ...v,
        nonce: nonces.get(v.validator.publicKey)!,
      })),
    );

    // 4. Mine to completion
    const currentBlock = Number(this.currentBlock);
    const blocksToComplete = round.completionBlock - currentBlock + 1;
    if (blocksToComplete > 0) {
      await this.mineBlocks(blocksToComplete);
    }
  }

  // ============ Getters ============

  getState(): StateManager {
    if (!this.state) throw new Error("Network not initialized");
    return this.state;
  }

  getConsensus(): ProofOfTruthEngine {
    if (!this.consensus) throw new Error("Network not initialized");
    return this.consensus;
  }

  getMempool(): Mempool {
    if (!this.mempool) throw new Error("Network not initialized");
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
      const seed = domainHash("LGM.test.validator.v1", `validator-${i}`);
      const { publicKey, privateKey } = generateKeypairFromSeed(
        seed.slice(0, 32),
      );

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
    const balancePerAccount =
      this.config.totalSupply / BigInt(count + this.config.validatorCount * 3);

    for (let i = 0; i < count; i++) {
      const seed = domainHash("LGM.test.account.v1", `account-${i}`);
      const { publicKey, privateKey } = generateKeypairFromSeed(
        seed.slice(0, 32),
      );

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
    const did = typeof account === "string" ? account : account.did;
    const state = this.network.getState();
    const actual = state.getAccount(did)?.balance ?? 0n;

    if (actual !== expected) {
      throw new AssertionError(
        `Balance mismatch for ${did}: expected ${expected}, got ${actual}`,
      );
    }
  }

  /**
   * Assert account balance is at least
   */
  balanceAtLeast(account: TestAccount | DID, minimum: bigint): void {
    const did = typeof account === "string" ? account : account.did;
    const state = this.network.getState();
    const actual = state.getAccount(did)?.balance ?? 0n;

    if (actual < minimum) {
      throw new AssertionError(
        `Balance too low for ${did}: expected at least ${minimum}, got ${actual}`,
      );
    }
  }

  /**
   * Assert fact exists
   */
  factExists(factId: string): void {
    const state = this.network.getState();
    const fact = state.getFact(factId) ?? state.getFact(`fact:${factId}`);

    if (!fact) {
      throw new AssertionError(`Fact not found: ${factId}`);
    }
  }

  /**
   * Assert fact has minimum confidence
   */
  factConfidenceAtLeast(factId: string, minConfidence: number): void {
    const state = this.network.getState();
    const fact = state.getFact(factId) ?? state.getFact(`fact:${factId}`);

    if (!fact) {
      throw new AssertionError(`Fact not found: ${factId}`);
    }

    if (fact.confidence < minConfidence) {
      throw new AssertionError(
        `Fact confidence too low: expected at least ${minConfidence}, got ${fact.confidence}`,
      );
    }
  }

  /**
   * Assert validator is active
   */
  validatorActive(validator: TestValidator | PublicKey): void {
    const pubKey =
      typeof validator === "string" ? validator : validator.publicKey;
    const state = this.network.getState();
    const v = state.getValidator(pubKey);

    if (!v || !v.isActive) {
      throw new AssertionError(
        `Validator not active: ${pubKey.slice(0, 16)}...`,
      );
    }
  }

  /**
   * Assert block number
   */
  blockNumberEquals(expected: bigint): void {
    const actual = this.network.getCurrentBlock();

    if (actual !== expected) {
      throw new AssertionError(
        `Block number mismatch: expected ${expected}, got ${actual}`,
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
        `Block number too low: expected at least ${minimum}, got ${actual}`,
      );
    }
  }
}

class AssertionError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "AssertionError";
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
    memo?: string,
  ): Transaction {
    const toDid = typeof to === "string" ? to : to.did;
    const nonce = this.getNextNonce(from.did);

    return {
      type: "transfer",
      sender: from.publicKey,
      fee: 1_000_000_000_000_000n,
      nonce,
      signature: "", // Would be signed
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
      type: "stake",
      sender: account.publicKey,
      fee: 1_000_000_000_000_000n,
      nonce,
      signature: "",
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
    references: string[] = [],
  ): Transaction {
    const nonce = this.getNextNonce(submitter.did);

    return {
      type: "submit_claim",
      sender: submitter.publicKey,
      fee: 1_000_000_000_000_000n,
      nonce,
      signature: "",
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
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Create a test network with default configuration
 */
export async function createTestNetwork(
  config: Partial<TestNetworkConfig> = {},
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
  fn: (
    network: TestNetwork,
    assert: StateAssertions,
    tx: TransactionBuilder,
  ) => Promise<T>,
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
