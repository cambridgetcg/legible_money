#!/usr/bin/env node

/**
 * Legible Money Node
 *
 * Main entry point for running a Legible Money node.
 * Supports validator and full node modes.
 */

import { readFileSync, existsSync, mkdirSync } from "fs";
import { dirname } from "path";
import { homedir } from "os";
import { EventEmitter } from "events";
import { StateManager, type GenesisConfig } from "../../state/state_manager.js";
import {
  ProofOfTruthEngine,
  createIntegratedEngine,
} from "../../consensus/pot/engine.js";
import { RPCServer, type ServerConfig } from "../../protocol/rpc/server.js";
import {
  DEFAULT_CONSENSUS_PARAMS,
  type ConsensusParams,
} from "../../consensus/pot/types.js";
import type { StateIntegration } from "../../consensus/pot/state_integration.js";
import type { PublicKey, PrivateKey } from "../../core/crypto/keys.js";
import { LegibleNode } from "../../network/node.js";
import {
  TopicManager,
  createBlockTopicHandler,
  createTransactionTopicHandler,
  createClaimTopicHandler,
  createVerificationTopicHandler,
} from "../../network/pubsub/topics.js";
import {
  SyncManager,
  SyncProtocolHandler,
  SYNC_PROTOCOL,
} from "../../network/protocols/sync.js";
import {
  PUBSUB_TOPICS,
  MESSAGE_TYPES,
  type NetworkMessage,
} from "../../network/config.js";
import type { KVStore } from "../../state/storage/types.js";
import { createStorage, MemoryStore } from "../../state/storage/leveldb.js";
// VULN-017 FIX: Import for genesis hash computation
import { sha512_256Hex } from "../../core/crypto/hashing.js";
// Economic Sustainability Integration
import { TruthGateway } from "../../core/billing/truth_gateway.js";
import { LoopController } from "../../core/genesis/perpetual_loop.js";
import { LoopEventBridge } from "../../core/genesis/loop_event_bridge.js";
import {
  DEFAULT_EMISSION_SCHEDULE,
  type GenesisConfig as LoopGenesisConfig,
} from "../../core/genesis/types.js";
import type { DID } from "../../core/agents/identity/did.js";
// Genesis block verification
import {
  GENESIS_BLOCK,
  verifyGenesisBlock,
} from "../../protocol/blocks/genesis.js";

// ============ Helpers ============

/** JSON.stringify with BigInt support (converts to string) */
function jsonStringifyBigInt(obj: unknown): string {
  return JSON.stringify(obj, (_key, value) =>
    typeof value === "bigint" ? value.toString() : value,
  );
}

/** Expand leading ~ to the user's home directory */
function expandHomePath(p: string): string {
  return p.startsWith("~") ? p.replace("~", homedir()) : p;
}

// ============ Configuration ============

interface NodeConfig {
  node: {
    name: string;
    dataDir: string;
    logLevel: "debug" | "info" | "warn" | "error";
  };
  storage: {
    type: "leveldb" | "memory";
    path?: string; // Required for leveldb, derived from dataDir if not specified
  };
  network: {
    listenAddresses: string[];
    bootstrapPeers: string[];
    maxPeers: number;
  };
  rpc: {
    enabled: boolean;
    host: string;
    port: number;
    cors: string[];
  };
  websocket: {
    enabled: boolean;
    port: number;
  };
  validator: {
    enabled: boolean;
    publicKey?: string;
  };
  mempool: {
    maxSize: number;
    maxPerSender: number;
  };
  telemetry: {
    enabled: boolean;
    prometheusPort: number;
  };
}

interface GenesisFile {
  network: {
    chainId: string;
    name: string;
    version: string;
  };
  consensus: Partial<ConsensusParams>;
  initialValidators: Array<{
    name: string;
    publicKey: string;
    peerId: string;
    stake: string;
    commission: number;
  }>;
  initialAccounts: Array<{
    did: string;
    balance: string;
    label?: string;
  }>;
  genesisAxioms: Array<{
    id: string;
    claim: string;
    domain: string;
  }>;
  genesisTime: string;
  // VULN-017 FIX: Optional hash for known/official genesis files
  genesisHash?: string;
}

// ============ Genesis Validation (VULN-017 FIX) ============

/**
 * VULN-017 FIX: Validation result for genesis file
 */
interface GenesisValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

/**
 * VULN-017 FIX: Known genesis hashes for official networks
 * These are used to verify the genesis file hasn't been tampered with
 */
const KNOWN_GENESIS_HASHES: Record<string, string> = {
  // Mainnet genesis (to be set at mainnet launch)
  "legible-money-mainnet": "",
  // Testnet genesis
  "legible-money-testnet": "",
  // Devnet (local development, hash not required)
  "legible-money-devnet": "",
};

/**
 * VULN-017 FIX: Validate genesis file structure and content
 * Returns validation errors if genesis is invalid
 */
function validateGenesis(genesis: GenesisFile): GenesisValidationResult {
  const errors: string[] = [];
  const warnings: string[] = [];

  // Required top-level fields
  if (!genesis.network) {
    errors.push("Missing required field: network");
  } else {
    if (!genesis.network.chainId) {
      errors.push("Missing required field: network.chainId");
    }
    if (!genesis.network.name) {
      errors.push("Missing required field: network.name");
    }
    if (!genesis.network.version) {
      errors.push("Missing required field: network.version");
    }
  }

  if (!genesis.genesisTime) {
    errors.push("Missing required field: genesisTime");
  } else {
    // Validate genesisTime is a valid ISO timestamp
    const timestamp = Date.parse(genesis.genesisTime);
    if (isNaN(timestamp)) {
      errors.push("Invalid genesisTime: must be valid ISO 8601 timestamp");
    }
  }

  // Validate initial validators
  if (!genesis.initialValidators || !Array.isArray(genesis.initialValidators)) {
    errors.push("Missing or invalid initialValidators array");
  } else {
    if (genesis.initialValidators.length === 0) {
      errors.push("initialValidators cannot be empty");
    }

    genesis.initialValidators.forEach((validator, index) => {
      if (!validator.publicKey) {
        errors.push(`Validator ${index}: missing publicKey`);
      } else if (!/^[a-fA-F0-9]{64}$/.test(validator.publicKey)) {
        errors.push(
          `Validator ${index}: invalid publicKey format (expected 64 hex chars)`,
        );
      }

      if (!validator.stake) {
        errors.push(`Validator ${index}: missing stake`);
      } else {
        try {
          const stake = BigInt(validator.stake);
          if (stake <= 0n) {
            errors.push(`Validator ${index}: stake must be positive`);
          }
        } catch {
          errors.push(
            `Validator ${index}: invalid stake format (expected numeric string)`,
          );
        }
      }

      if (validator.commission !== undefined) {
        if (validator.commission < 0 || validator.commission > 1) {
          errors.push(`Validator ${index}: commission must be between 0 and 1`);
        }
      }
    });
  }

  // Validate initial accounts
  if (!genesis.initialAccounts || !Array.isArray(genesis.initialAccounts)) {
    warnings.push("No initialAccounts defined");
  } else {
    genesis.initialAccounts.forEach((account, index) => {
      if (!account.did) {
        errors.push(`Account ${index}: missing did`);
      } else if (!account.did.startsWith("did:lgm:")) {
        warnings.push(`Account ${index}: DID does not use 'did:lgm:' method`);
      }

      if (account.balance === undefined || account.balance === null) {
        errors.push(`Account ${index}: missing balance`);
      } else {
        try {
          const balance = BigInt(account.balance);
          if (balance < 0n) {
            errors.push(`Account ${index}: balance cannot be negative`);
          }
        } catch {
          errors.push(`Account ${index}: invalid balance format`);
        }
      }
    });
  }

  // Validate consensus params if provided
  if (genesis.consensus) {
    if (genesis.consensus.targetBlockTimeMs !== undefined) {
      if (
        genesis.consensus.targetBlockTimeMs < 100 ||
        genesis.consensus.targetBlockTimeMs > 60000
      ) {
        warnings.push(
          "targetBlockTimeMs outside reasonable range (100-60000ms)",
        );
      }
    }
    if (genesis.consensus.minValidatorStake !== undefined) {
      const minStake = BigInt(genesis.consensus.minValidatorStake);
      if (minStake <= 0n) {
        errors.push("minValidatorStake must be positive");
      }
    }
    if (genesis.consensus.validatorQuorum !== undefined) {
      if (
        genesis.consensus.validatorQuorum < 0.5 ||
        genesis.consensus.validatorQuorum > 1
      ) {
        errors.push("validatorQuorum must be between 0.5 and 1");
      }
    }
  }

  // Validate genesis axioms
  if (!genesis.genesisAxioms || !Array.isArray(genesis.genesisAxioms)) {
    warnings.push("No genesisAxioms defined");
  } else {
    genesis.genesisAxioms.forEach((axiom, index) => {
      if (!axiom.id) {
        errors.push(`Axiom ${index}: missing id`);
      }
      if (!axiom.claim) {
        errors.push(`Axiom ${index}: missing claim`);
      }
    });
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
  };
}

/**
 * VULN-017 FIX: Compute hash of genesis file for integrity verification
 */
function computeGenesisHash(genesis: GenesisFile): string {
  // Create canonical representation (sorted keys, no whitespace)
  const canonical = JSON.stringify(genesis, Object.keys(genesis).sort(), 0);
  return sha512_256Hex(canonical);
}

/**
 * VULN-017 FIX: Verify genesis against known hash (for official networks)
 */
function verifyKnownGenesisHash(genesis: GenesisFile): {
  valid: boolean;
  reason?: string;
} {
  const chainId = genesis.network?.chainId;
  if (!chainId) {
    return { valid: false, reason: "Missing chainId" };
  }

  const expectedHash = KNOWN_GENESIS_HASHES[chainId];
  if (!expectedHash) {
    // Unknown chain (custom network), skip hash verification but warn
    return {
      valid: true,
      reason: "Custom network - genesis hash not verified",
    };
  }

  if (expectedHash === "") {
    // Known network but hash not yet set (pre-launch)
    return { valid: true, reason: "Official network genesis hash pending" };
  }

  const actualHash = computeGenesisHash(genesis);
  if (actualHash !== expectedHash) {
    return {
      valid: false,
      reason: `Genesis hash mismatch. Expected: ${expectedHash}, Got: ${actualHash}`,
    };
  }

  return { valid: true };
}

/**
 * Safely merge genesis consensus params with defaults.
 * Converts string values from JSON to appropriate types (BigInt for stakes/rewards).
 * This prevents type bypass when spreading JSON-parsed objects.
 */
function mergeConsensusParams(
  defaults: ConsensusParams,
  genesis?: Partial<Record<string, unknown>>,
): ConsensusParams {
  if (!genesis) return defaults;

  return {
    ...defaults,
    // Number fields (safe from JSON)
    ...(genesis.targetBlockTimeMs !== undefined && {
      targetBlockTimeMs: Number(genesis.targetBlockTimeMs),
    }),
    ...(genesis.verificationRoundBlocks !== undefined && {
      verificationRoundBlocks: Number(genesis.verificationRoundBlocks),
    }),
    ...(genesis.commitPhaseBlocks !== undefined && {
      commitPhaseBlocks: Number(genesis.commitPhaseBlocks),
    }),
    ...(genesis.revealPhaseBlocks !== undefined && {
      revealPhaseBlocks: Number(genesis.revealPhaseBlocks),
    }),
    ...(genesis.confirmationBlocks !== undefined && {
      confirmationBlocks: Number(genesis.confirmationBlocks),
    }),
    ...(genesis.finalityBlocks !== undefined && {
      finalityBlocks: Number(genesis.finalityBlocks),
    }),
    ...(genesis.minValidators !== undefined && {
      minValidators: Number(genesis.minValidators),
    }),
    ...(genesis.maxValidators !== undefined && {
      maxValidators: Number(genesis.maxValidators),
    }),
    ...(genesis.validatorQuorum !== undefined && {
      validatorQuorum: Number(genesis.validatorQuorum),
    }),
    ...(genesis.acceptanceThreshold !== undefined && {
      acceptanceThreshold: Number(genesis.acceptanceThreshold),
    }),
    ...(genesis.rejectionThreshold !== undefined && {
      rejectionThreshold: Number(genesis.rejectionThreshold),
    }),
    ...(genesis.contestedThreshold !== undefined && {
      contestedThreshold: Number(genesis.contestedThreshold),
    }),
    ...(genesis.wrongVerificationSlash !== undefined && {
      wrongVerificationSlash: Number(genesis.wrongVerificationSlash),
    }),
    ...(genesis.missedVerificationSlash !== undefined && {
      missedVerificationSlash: Number(genesis.missedVerificationSlash),
    }),
    ...(genesis.invalidClaimSlash !== undefined && {
      invalidClaimSlash: Number(genesis.invalidClaimSlash),
    }),
    ...(genesis.researchShareRate !== undefined && {
      researchShareRate: Number(genesis.researchShareRate),
    }),
    ...(genesis.maxTransactionsPerBlock !== undefined && {
      maxTransactionsPerBlock: Number(genesis.maxTransactionsPerBlock),
    }),
    // BigInt fields (must convert explicitly from JSON strings)
    ...(genesis.epochLength !== undefined && {
      epochLength: BigInt(genesis.epochLength as string | number | bigint),
    }),
    ...(genesis.minValidatorStake !== undefined && {
      minValidatorStake: BigInt(
        genesis.minValidatorStake as string | number | bigint,
      ),
    }),
    ...(genesis.minClaimStake !== undefined && {
      minClaimStake: BigInt(genesis.minClaimStake as string | number | bigint),
    }),
    ...(genesis.blockReward !== undefined && {
      blockReward: BigInt(genesis.blockReward as string | number | bigint),
    }),
    ...(genesis.verificationReward !== undefined && {
      verificationReward: BigInt(
        genesis.verificationReward as string | number | bigint,
      ),
    }),
    ...(genesis.citationRewardPerFact !== undefined && {
      citationRewardPerFact: BigInt(
        genesis.citationRewardPerFact as string | number | bigint,
      ),
    }),
    ...(genesis.unbondingPeriod !== undefined && {
      unbondingPeriod: BigInt(
        genesis.unbondingPeriod as string | number | bigint,
      ),
    }),
  };
}

// ============ Node ============

class LegibleMoneyNode extends EventEmitter {
  private config: NodeConfig;
  private genesis: GenesisFile;
  private state: StateManager;
  private store: KVStore | null = null;
  private consensus: ProofOfTruthEngine;
  private stateIntegration: StateIntegration;
  private rpcServer: RPCServer | null = null;
  private validatorKey: PrivateKey | null = null;
  private running = false;

  // Network layer
  private networkNode: LegibleNode | null = null;
  private topicManager: TopicManager | null = null;
  private syncManager: SyncManager | null = null;

  // Economic sustainability (hoisted from startRpcServer for event wiring)
  private loopController: LoopController | null = null;
  private loopEventBridge: LoopEventBridge | null = null;

  private constructor(
    config: NodeConfig,
    genesis: GenesisFile,
    state: StateManager,
    store: KVStore | null,
    consensus: ProofOfTruthEngine,
    stateIntegration: StateIntegration,
  ) {
    super();
    this.config = config;
    this.genesis = genesis;
    this.state = state;
    this.store = store;
    this.consensus = consensus;
    this.stateIntegration = stateIntegration;
  }

  /**
   * Create a new LegibleMoneyNode with persistence support
   */
  static async create(
    config: NodeConfig,
    genesis: GenesisFile,
  ): Promise<LegibleMoneyNode> {
    const { state, store } = await LegibleMoneyNode.initializeState(
      config,
      genesis,
    );

    // Merge genesis consensus params with defaults (safely converting BigInt fields from JSON strings)
    const params = mergeConsensusParams(
      DEFAULT_CONSENSUS_PARAMS,
      genesis.consensus as Partial<Record<string, unknown>>,
    );

    // Load validator key BEFORE creating consensus engine (if validator mode)
    let validatorKey: PrivateKey | null = null;
    if (config.validator.enabled) {
      validatorKey = LegibleMoneyNode.loadValidatorKeySync(config);
    }

    // Create integrated engine with proper StateIntegration
    // This wires: StateManager -> StateIntegration -> ProofOfTruthEngine
    const { engine, stateIntegration } = LegibleMoneyNode.createConsensusEngine(
      config,
      genesis,
      state,
      params,
      validatorKey,
    );

    const node = new LegibleMoneyNode(
      config,
      genesis,
      state,
      store,
      engine,
      stateIntegration,
    );
    node.validatorKey = validatorKey;
    return node;
  }

  /**
   * Load validator key synchronously (used during node creation)
   */
  private static loadValidatorKeySync(config: NodeConfig): PrivateKey | null {
    const keyPath = process.env.VALIDATOR_KEY_PATH || "/secrets/validator.key";

    if (!existsSync(keyPath)) {
      console.log(`[WARN] Validator key not found at ${keyPath}`);
      return null;
    }

    const keyData = JSON.parse(readFileSync(keyPath, "utf-8"));
    config.validator.publicKey = keyData.publicKey;
    console.log(
      `[INFO] Loaded validator key: ${keyData.publicKey.slice(0, 16)}...`,
    );
    return keyData.privateKey;
  }

  /**
   * Create consensus engine with proper state integration
   * GAP FIX: Uses createIntegratedEngine for correct wiring
   */
  private static createConsensusEngine(
    config: NodeConfig,
    genesis: GenesisFile,
    state: StateManager,
    params: ConsensusParams,
    validatorKey: PrivateKey | null,
  ): { engine: ProofOfTruthEngine; stateIntegration: StateIntegration } {
    // Parse initial validators from genesis
    const initialValidators = genesis.initialValidators.map((v) => ({
      publicKey: v.publicKey as PublicKey,
      did: `did:lgm-devnet:${v.name}`,
      stake: BigInt(v.stake),
    }));

    // Create integrated engine - this creates StateIntegration internally
    // CRITICAL: Pass validatorKey to enable block production
    const engine = createIntegratedEngine({
      isValidator: config.validator.enabled,
      validatorKey: validatorKey ?? undefined,
      validatorPublicKey: config.validator.publicKey as PublicKey | undefined,
      initialValidators,
      stateManager: state,
      params,
      allowTestMethods: true,
    });

    // Get the state integration from the engine
    // Note: createIntegratedEngine sets up StateIntegration internally
    // We need to access it for RPC handlers - add getter if needed
    const stateIntegration = engine.getStateIntegration()!;

    return { engine, stateIntegration };
  }

  /**
   * Initialize state with persistence
   * - Opens/creates storage based on config
   * - Tries to load existing state
   * - Applies genesis if no existing state found
   */
  private static async initializeState(
    config: NodeConfig,
    genesis: GenesisFile,
  ): Promise<{ state: StateManager; store: KVStore | null }> {
    let store: KVStore | null = null;

    // Create store based on config
    if (config.storage.type === "leveldb") {
      const storagePath = config.storage.path || `${config.node.dataDir}/state`;

      // Ensure storage directory exists
      try {
        mkdirSync(dirname(storagePath), { recursive: true });
      } catch {
        // Directory may already exist
      }

      console.log(`[INFO] Opening LevelDB storage at: ${storagePath}`);
      store = await createStorage({ type: "leveldb", path: storagePath });
    } else if (config.storage.type === "memory") {
      console.log("[INFO] Using in-memory storage (state will not persist)");
      store = new MemoryStore();
    }

    // Create state manager with store
    const state = new StateManager({}, store ?? undefined);

    // Try to load existing state
    if (store) {
      const latestBlock = await state.getLatestBlockNumber();

      if (latestBlock !== null) {
        console.log(`[INFO] Found existing state at block ${latestBlock}`);
        const initialized = await state.initializeFromStorage();
        if (initialized) {
          console.log("[INFO] State restored from storage");
          return { state, store };
        }
      }
    }

    // No existing state - apply genesis
    console.log("[INFO] Initializing fresh state from genesis");

    // Verify the ceremonial genesis block (founding axiom commitment + protocol version)
    if (verifyGenesisBlock(GENESIS_BLOCK)) {
      console.log(
        "[GENESIS] Genesis block integrity verified (founding commitment valid)",
      );
    } else {
      console.error(
        "[GENESIS] FATAL: Genesis block verification failed — founding commitment or protocol version mismatch",
      );
      process.exit(1);
    }

    const genesisConfig: GenesisConfig = {
      accounts: genesis.initialAccounts.map((a) => ({
        did: a.did,
        operationalKeyHash: "",
        balance: BigInt(a.balance),
      })),
      validators: genesis.initialValidators.map((v) => ({
        publicKey: v.publicKey as PublicKey,
        did: `did:lgm-devnet:${v.name}`,
        stake: BigInt(v.stake),
      })),
      axioms: genesis.genesisAxioms,
    };

    // Apply genesis configuration — all fund accounts from MAX_SUPPLY allocation
    console.log("[GENESIS] Creating genesis accounts:");
    for (const account of genesisConfig.accounts) {
      state.createAccount(
        account.did,
        account.operationalKeyHash,
        account.balance,
      );
      const lgm = account.balance / 10n ** 18n;
      console.log(`  ${account.did}: ${lgm} LGM`);
    }

    for (const validator of genesisConfig.validators) {
      state.registerValidator(
        validator.publicKey,
        validator.did,
        validator.stake,
      );
    }

    for (const axiom of genesisConfig.axioms) {
      state.addFact({
        id: axiom.id,
        claimText: axiom.claim,
        domain: axiom.domain,
        confidence: 1.0,
        verifiedAt: 0n,
        verifierCount: 0,
        challengeCount: 0,
        citationCount: 0,
        status: "active",
      });
    }

    // Persist genesis state (block 0)
    if (store) {
      await state.persistStateRoots(0n);
      await state.persistBlockHeader({
        number: "0",
        hash: state.getStateRoot(),
        parentHash: "0".repeat(64),
        stateRoot: state.getStateRoot(),
        timestamp: new Date(genesis.genesisTime).getTime(),
      });
      console.log("[INFO] Genesis state persisted");
    }

    return { state, store };
  }

  async start(): Promise<void> {
    if (this.running) return;

    this.log("info", `Starting Legible Money Node: ${this.config.node.name}`);
    this.log("info", `Chain: ${this.genesis.network.chainId}`);

    // Validator key is loaded in create() - just log status
    if (this.config.validator.enabled) {
      if (this.validatorKey) {
        this.log(
          "info",
          `Validator mode enabled: ${this.config.validator.publicKey?.slice(0, 16)}...`,
        );
      } else {
        this.log(
          "warn",
          "Validator mode enabled but no validator key loaded - block production disabled",
        );
      }
    }

    // Note: Mempool is managed by StateIntegration internally

    // Initialize economic sustainability tracking (before event handlers)
    this.initializeLoopController();

    // Start consensus engine
    await this.consensus.start();
    this.log("info", "Consensus engine started");

    // Setup consensus event handlers (must be after loopController init)
    this.setupEventHandlers();

    // Start RPC server
    if (this.config.rpc.enabled) {
      await this.startRpcServer();
    }

    // Start P2P networking
    await this.startNetworking();

    this.running = true;
    this.log("info", "Node started successfully");
  }

  async stop(): Promise<void> {
    if (!this.running) return;

    this.log("info", "Stopping node...");

    // Stop network node
    if (this.networkNode) {
      await this.networkNode.stop();
      this.networkNode = null;
    }

    if (this.rpcServer) {
      await this.rpcServer.stop();
    }

    await this.consensus.stop();

    // Close state integration (closes storage if persisted)
    await this.stateIntegration.close();

    // Persist final state and close storage
    if (this.store) {
      this.log("info", "Persisting final state...");
      const blockNumber = this.state.getBlockNumber();
      await this.state.persistStateRoots(blockNumber);
      await this.store.close();
      this.log("info", "Storage closed");
    }

    this.running = false;
    this.log("info", "Node stopped");
  }

  // Validator key loading is now done in loadValidatorKeySync during create()

  /**
   * Initialize LoopController and LoopEventBridge at class level.
   * Must be called before setupEventHandlers() so block events
   * can be routed to sustainability tracking.
   */
  private initializeLoopController(): void {
    const loopGenesisConfig: LoopGenesisConfig = {
      chainId: this.genesis.network.chainId,
      timestamp: new Date(this.genesis.genesisTime || Date.now()).getTime(),
      accounts: [],
      seedValidators: [],
      seedFacts: (this.genesis.genesisAxioms || []).map(
        (a: { id: string; claim: string; domain: string }) => ({
          id: a.id,
          claim: a.claim,
          domain: a.domain,
          confidence: 1.0,
          category: "axiomatic" as const,
        }),
      ),
      emissionSchedule: DEFAULT_EMISSION_SCHEDULE,
      protocolParams: {
        targetBlockTimeMs: this.genesis.consensus.targetBlockTimeMs ?? 2521,
        finalityDepth: 19,
        minValidators: this.genesis.consensus.minValidators ?? 1,
        maxValidators: this.genesis.consensus.maxValidators ?? 21,
        validatorQuorum: this.genesis.consensus.validatorQuorum ?? 0.66,
        minValidatorStake: BigInt(
          this.genesis.consensus.minValidatorStake ?? "111000000000000000",
        ),
        minClaimStake: BigInt(
          this.genesis.consensus.minClaimStake ?? "11000000000000000",
        ),
        acceptanceThreshold: this.genesis.consensus.acceptanceThreshold ?? 0.8,
        rejectionThreshold: this.genesis.consensus.rejectionThreshold ?? 0.8,
        contestedThreshold: 0.5,
        wrongVerificationSlash: 0.06,
        missedVerificationSlash: 0.0001,
        invalidClaimSlash: 0.5,
        researchShareRate: 0.22,
        apiProviderShare: 0.6,
        knowledgePoolShare: 0.3,
        protocolShare: 0.1,
      },
    };
    this.loopController = new LoopController(loopGenesisConfig);
    this.loopEventBridge = new LoopEventBridge(this.loopController);
    this.log("info", "Economic sustainability loop initialized");
  }

  private async startRpcServer(): Promise<void> {
    const serverConfig: Partial<ServerConfig> = {
      port: this.config.rpc.port,
      host: this.config.rpc.host,
      enableWebSocket: this.config.websocket.enabled,
      cors: {
        enabled: true,
        origins: this.config.rpc.cors,
      },
    };

    // Economic Sustainability: Create TruthGateway for query billing
    const providerDid = `did:lgm:${this.config.node.name}` as DID;
    const truthGateway = new TruthGateway(providerDid);

    this.rpcServer = new RPCServer(
      {
        state: this.state,
        consensus: this.consensus,
        stateIntegration: this.stateIntegration,
        nodeVersion: this.genesis.network.version,
        networkId: this.genesis.network.chainId,
        truthGateway,
        providerDid,
        loopController: this.loopController!,
        loopEventBridge: this.loopEventBridge!,
      },
      serverConfig,
    );

    await this.rpcServer.start();
    this.log(
      "info",
      `RPC server listening on ${this.config.rpc.host}:${this.config.rpc.port}`,
    );

    if (this.config.websocket.enabled) {
      this.log(
        "info",
        `WebSocket enabled on port ${this.config.websocket.port}`,
      );
    }
  }

  private async startNetworking(): Promise<void> {
    // Create network node
    this.networkNode = new LegibleNode({
      listenAddresses: this.config.network.listenAddresses,
      bootstrapPeers: this.config.network.bootstrapPeers,
      maxConnections: this.config.network.maxPeers,
      enableMdns: true,
    });

    // Create topic manager with handlers
    this.topicManager = new TopicManager();

    // Register block handler
    this.topicManager.registerHandler(
      createBlockTopicHandler(
        async (block) => {
          // Validate block before accepting
          try {
            const validation = await this.consensus.validateBlock(block as any);
            return validation.valid;
          } catch {
            return false;
          }
        },
        async (block) => {
          // Apply valid block to consensus
          await this.consensus.applyBlock(block as any);
        },
      ),
    );

    // Register transaction handler
    this.topicManager.registerHandler(
      createTransactionTopicHandler(
        async (tx) => {
          // Validate transaction via state integration
          const validation = await this.stateIntegration.validateTransaction(
            tx as any,
          );
          return validation.valid;
        },
        async (tx) => {
          // Add to state integration's mempool
          this.stateIntegration.addToMempool(tx as any);
        },
      ),
    );

    // Register claim handler
    this.topicManager.registerHandler(
      createClaimTopicHandler(
        async (claim) => {
          // Basic claim validation
          const c = claim as any;
          return c.claim && c.references && c.references.length > 0;
        },
        async (claim) => {
          // Submit claim to consensus
          await this.consensus.submitClaim(claim as any);
        },
      ),
    );

    // Register verification handler
    this.topicManager.registerHandler(
      createVerificationTopicHandler(
        async (verification) => {
          // Validate verification message
          const v = verification as any;
          return v.roundId && v.validator;
        },
        async (verification) => {
          // Route to consensus based on type
          const v = verification as any;
          if (v.type === "commit" || v.commitment) {
            await this.consensus.submitCommitment(v.roundId, v);
          } else if (v.type === "reveal" || v.verdict) {
            await this.consensus.submitReveal(v.roundId, v);
          }
        },
      ),
    );

    // Wire topic manager to network node
    this.networkNode.setTopicManager(this.topicManager);

    // Create sync manager
    this.syncManager = new SyncManager(
      this.state.getBlockNumber(),
      async (num) => {
        // Get block from persistent storage if available
        if (this.store) {
          const header = await this.state.loadBlockHeader(num);
          return header; // Returns block header or null if not found
        }
        return null;
      },
      async (block) => {
        await this.consensus.applyBlock(block as any);
      },
      async (block) => {
        const result = await this.consensus.validateBlock(block as any);
        return result.valid;
      },
    );

    // Create sync protocol handler
    const syncHandler = new SyncProtocolHandler(this.syncManager);

    // Start the network node FIRST (required before registering protocols)
    await this.networkNode.start();
    this.log(
      "info",
      `P2P node started with peer ID: ${this.networkNode.getPeerId()?.slice(0, 16)}...`,
    );

    // Set request function for sync manager to fetch blocks from peers
    this.syncManager.setRequestFunction(async (peerId, start, count) => {
      // Cast to expected signature - NetworkMessage is compatible with unknown
      const requestFn = this.networkNode!.request.bind(this.networkNode) as <T>(
        peerId: string,
        protocol: string,
        request: unknown,
        timeout?: number,
      ) => Promise<T>;
      return syncHandler.requestBlocksFromPeer(requestFn, peerId, start, count);
    });

    // Register sync protocol handler (node must be started first)
    await this.networkNode.registerProtocol(SYNC_PROTOCOL, (stream, peerId) =>
      syncHandler.handleStream(stream, peerId),
    );

    this.networkNode.setSyncManager(this.syncManager);

    // Wire peer events to sync manager
    this.networkNode.on("peer:connect", (event: any) => {
      this.log("info", `Peer connected: ${event.peer.peerId.slice(0, 16)}...`);
      // Add peer to sync manager (will need head block info from handshake)
      this.syncManager?.addPeer({
        peerId: event.peer.peerId,
        headBlock: 0n, // Will be updated by status exchange
        latency: event.peer.latency || 0,
        score: event.peer.score || 0,
      });
    });

    this.networkNode.on("peer:disconnect", (event: any) => {
      this.log(
        "info",
        `Peer disconnected: ${event.peer.peerId.slice(0, 16)}...`,
      );
      this.syncManager?.removePeer(event.peer.peerId);
    });

    const addrs = this.networkNode.getListenAddresses();
    for (const addr of addrs) {
      this.log("info", `P2P listening on: ${addr}`);
    }

    if (this.config.network.bootstrapPeers.length > 0) {
      this.log(
        "info",
        `Connecting to ${this.config.network.bootstrapPeers.length} bootstrap peers`,
      );
    }

    // Manual peer dial from environment variable (for Docker networking)
    const dialPeers = process.env.DIAL_PEERS;
    if (dialPeers) {
      const peers = dialPeers
        .split(",")
        .map((p) => p.trim())
        .filter((p) => p);
      this.log("info", `Dialing ${peers.length} peers from DIAL_PEERS env`);
      for (const peer of peers) {
        this.networkNode
          .connect(peer)
          .then((peerInfo) => {
            if (peerInfo) {
              this.log(
                "info",
                `Connected to peer: ${peerInfo.peerId.slice(0, 16)}...`,
              );
            } else {
              this.log("warn", `Failed to connect to: ${peer}`);
            }
          })
          .catch((err) => {
            this.log("warn", `Failed to dial ${peer}: ${err.message}`);
          });
      }
    }
  }

  private setupEventHandlers(): void {
    // Forward consensus events
    // NOTE: Engine emits underscore-separated event types via ConsensusEvent.type
    // (e.g., 'block_proposed', NOT 'block:produced')

    this.consensus.on(
      "block_proposed",
      async (event: { type: string; block: any }) => {
        const block = event.block;
        this.log("info", `Produced block ${block.header.number}`);

        // Persist block and state to storage
        if (this.store) {
          try {
            await this.state.persistStateRoots(block.header.number);
            await this.state.persistBlockHeader({
              number: block.header.number.toString(),
              hash: block.header.stateRoot || this.state.getStateRoot(),
              parentHash: block.header.previousHash,
              stateRoot: this.state.getStateRoot(),
              timestamp: block.header.timestamp,
              proposer: block.header.proposer,
            });
            this.log(
              "debug",
              `Block ${block.header.number} persisted to storage`,
            );
          } catch (error) {
            this.log("error", `Failed to persist block: ${error}`);
          }
        }

        // Feed block production into sustainability loop
        if (this.loopEventBridge) {
          try {
            const blockReward = 10000000000000000000n; // 10 LGM base reward
            const researchShare = (blockReward * 22n) / 100n; // 22% research allocation
            this.loopEventBridge.onBlockProduced(
              block.header.number,
              blockReward,
              researchShare,
              new Map([[block.header.proposer, blockReward - researchShare]]),
            );
          } catch (error) {
            this.log("warn", `Loop event bridge error: ${error}`);
          }
        }

        // Broadcast to RPC subscribers
        this.rpcServer?.broadcast("block", block);

        // Broadcast to P2P network
        if (this.networkNode) {
          const message: NetworkMessage = {
            type: MESSAGE_TYPES.NEW_BLOCK,
            version: "1.0.0",
            timestamp: Date.now(),
            sender: this.networkNode.getPeerId() || "",
            payload: new TextEncoder().encode(jsonStringifyBigInt(block)),
          };
          try {
            await this.networkNode.broadcast(PUBSUB_TOPICS.BLOCKS, message);
          } catch (error) {
            this.log("warn", `Failed to broadcast block: ${error}`);
          }
        }
      },
    );

    this.consensus.on(
      "block_finalized",
      (event: { type: string; blockNumber: bigint; blockHash: string }) => {
        this.log("info", `Finalized block ${event.blockNumber}`);
      },
    );

    this.consensus.on(
      "round_started",
      async (event: { type: string; round: any }) => {
        this.log("info", `Verification round started: ${event.round.id}`);

        // Broadcast to P2P network
        if (this.networkNode) {
          const message: NetworkMessage = {
            type: MESSAGE_TYPES.NEW_CLAIM,
            version: "1.0.0",
            timestamp: Date.now(),
            sender: this.networkNode.getPeerId() || "",
            payload: new TextEncoder().encode(jsonStringifyBigInt(event.round)),
          };
          try {
            await this.networkNode.broadcast(PUBSUB_TOPICS.CLAIMS, message);
          } catch (error) {
            this.log("warn", `Failed to broadcast round: ${error}`);
          }
        }
      },
    );

    this.consensus.on(
      "round_completed",
      (event: { type: string; result: any }) => {
        const result = event.result;
        this.log(
          "info",
          `Verification round completed: ${result.claimId} -> ${result.outcome}`,
        );

        // Feed verification results into sustainability loop
        if (this.loopEventBridge && result.outcome) {
          try {
            this.loopEventBridge.onClaimVerified(
              result.claimId,
              result.outcome,
              result.factId,
              new Map(),
              0n,
            );
          } catch (error) {
            this.log("warn", `Loop event bridge error: ${error}`);
          }
        }

        if (result.outcome === "accepted") {
          this.rpcServer?.broadcast("fact:accepted", result.fact);
        }
      },
    );

    // Note: Transaction broadcasting is handled by the network transaction handler
    // when transactions are received and validated via stateIntegration
  }

  private log(level: string, message: string): void {
    const levels = ["debug", "info", "warn", "error"];
    const configLevel = levels.indexOf(this.config.node.logLevel);
    const msgLevel = levels.indexOf(level);

    if (msgLevel >= configLevel) {
      const timestamp = new Date().toISOString();
      console.log(`[${timestamp}] [${level.toUpperCase()}] ${message}`);
    }
  }

  // ============ API Methods ============

  getState(): StateManager {
    return this.state;
  }

  getConsensus(): ProofOfTruthEngine {
    return this.consensus;
  }

  getStateIntegration(): StateIntegration {
    return this.stateIntegration;
  }

  isRunning(): boolean {
    return this.running;
  }

  isValidator(): boolean {
    return this.config.validator.enabled;
  }

  getNetwork(): LegibleNode | null {
    return this.networkNode;
  }
}

// ============ Main ============

async function main(): Promise<void> {
  console.log(`
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║   ██╗     ███████╗ ██████╗ ██╗██████╗ ██╗     ███████╗       ║
║   ██║     ██╔════╝██╔════╝ ██║██╔══██╗██║     ██╔════╝       ║
║   ██║     █████╗  ██║  ███╗██║██████╔╝██║     █████╗         ║
║   ██║     ██╔══╝  ██║   ██║██║██╔══██╗██║     ██╔══╝         ║
║   ███████╗███████╗╚██████╔╝██║██████╔╝███████╗███████╗       ║
║   ╚══════╝╚══════╝ ╚═════╝ ╚═╝╚═════╝ ╚══════╝╚══════╝       ║
║                                                               ║
║              M O N E Y    v0.1.0                              ║
║                                                               ║
║         Proof of Truth • Knowledge is Value                   ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
`);

  // Load configuration
  const configPath = process.env.CONFIG_PATH || "/config/node.json";
  const genesisPath = process.env.GENESIS_PATH || "/config/genesis.json";

  if (!existsSync(configPath)) {
    console.error(`Config file not found: ${configPath}`);
    process.exit(1);
  }

  if (!existsSync(genesisPath)) {
    console.error(`Genesis file not found: ${genesisPath}`);
    process.exit(1);
  }

  const config: NodeConfig = JSON.parse(readFileSync(configPath, "utf-8"));
  const genesis: GenesisFile = JSON.parse(readFileSync(genesisPath, "utf-8"));

  // Expand ~ in paths so local-validator.json can use ~/.lgm/devnet
  config.node.dataDir = expandHomePath(config.node.dataDir);
  if (config.storage?.path) {
    config.storage.path = expandHomePath(config.storage.path);
  }

  // VULN-017 FIX: Validate genesis file before using it
  console.log("[GENESIS] Validating genesis file...");
  const validation = validateGenesis(genesis);

  if (validation.warnings.length > 0) {
    console.warn("[GENESIS] Warnings:");
    validation.warnings.forEach((w) => console.warn(`  - ${w}`));
  }

  if (!validation.valid) {
    console.error("[GENESIS] Genesis validation failed:");
    validation.errors.forEach((e) => console.error(`  - ${e}`));
    process.exit(1);
  }

  // VULN-017 FIX: Verify genesis hash for known networks
  const hashVerification = verifyKnownGenesisHash(genesis);
  if (!hashVerification.valid) {
    console.error(
      `[GENESIS] Genesis hash verification failed: ${hashVerification.reason}`,
    );
    console.error("[GENESIS] This may indicate a tampered genesis file.");
    console.error(
      "[GENESIS] If this is intentional (custom network), update KNOWN_GENESIS_HASHES.",
    );
    process.exit(1);
  }
  if (hashVerification.reason) {
    console.log(`[GENESIS] Hash verification: ${hashVerification.reason}`);
  }

  console.log(
    `[GENESIS] Validated: ${genesis.network.name} (${genesis.network.chainId})`,
  );
  console.log(
    `[GENESIS] Validators: ${genesis.initialValidators.length}, Accounts: ${genesis.initialAccounts?.length || 0}`,
  );

  // Ensure storage config exists (backwards compatibility)
  if (!config.storage) {
    config.storage = {
      type: "leveldb",
      path: `${config.node.dataDir}/state`,
    };
    console.log("[INFO] No storage config found, defaulting to LevelDB");
  }

  // Create and start node (async factory)
  const node = await LegibleMoneyNode.create(config, genesis);

  // Handle shutdown
  process.on("SIGINT", async () => {
    console.log("\nReceived SIGINT, shutting down...");
    await node.stop();
    process.exit(0);
  });

  process.on("SIGTERM", async () => {
    console.log("\nReceived SIGTERM, shutting down...");
    await node.stop();
    process.exit(0);
  });

  // Start node
  await node.start();

  // Keep running
  console.log("\nNode is running. Press Ctrl+C to stop.\n");
}

// Run
main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});

export { LegibleMoneyNode, NodeConfig, GenesisFile };
