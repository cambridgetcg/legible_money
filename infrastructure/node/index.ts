/**
 * Legible Money Node
 *
 * Main entry point for running a full Legible Money node.
 * Orchestrates all components: storage, state, consensus, network.
 *
 * B1 FIX: Wires LevelDB persistence to StateManager
 */

import { EventEmitter } from "events";
import type { KVStore } from "../../state/storage/types.js";
import { createStorage, MemoryStore } from "../../state/storage/leveldb.js";
import { StateManager } from "../../state/state_manager.js";
import type { StateIntegration } from "../../consensus/pot/state_integration.js";
import {
  ProofOfTruthEngine,
  createIntegratedEngine,
} from "../../consensus/pot/engine.js";
import { DEFAULT_CONSENSUS_PARAMS } from "../../consensus/pot/types.js";
import { Mempool } from "../../consensus/mempool/pool.js";
import { LegibleNode } from "../../network/node.js";
import {
  ConsensusNetworkBridge,
  createConsensusBridge,
} from "../../network/adapters/consensus-bridge.js";
import type { ConsensusParams } from "../../consensus/pot/types.js";
import type { PrivateKey, PublicKey } from "../../core/crypto/keys.js";
import { WriteAheadLog } from "../../state/storage/write_ahead_log.js";
import { RPCServer } from "../../protocol/rpc/server.js";
import type { HandlerContext } from "../../protocol/rpc/handlers.js";

// ============ Types ============

export interface NodeConfig {
  /** Node identification */
  node: {
    name: string;
    dataDir: string;
    logLevel: "debug" | "info" | "warn" | "error";
  };

  /** Storage configuration */
  storage: {
    type: "memory" | "leveldb";
    path?: string; // Required for leveldb
  };

  /** Network configuration */
  network: {
    listenAddresses: string[];
    bootstrapPeers: string[];
    maxPeers: number;
  };

  /** RPC configuration */
  rpc: {
    enabled: boolean;
    host: string;
    port: number;
    cors: string[];
  };

  /** WebSocket configuration */
  websocket: {
    enabled: boolean;
    port: number;
  };

  /** Validator configuration */
  validator: {
    enabled: boolean;
    publicKey?: string;
    privateKey?: string;
  };

  /** Mempool configuration */
  mempool: {
    maxSize: number;
    maxPerSender: number;
  };

  /** Telemetry configuration */
  telemetry: {
    enabled: boolean;
    prometheusPort?: number;
  };
}

export interface GenesisConfig {
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
  }>;
  genesisAxioms: Array<{
    id: string;
    claim: string;
    domain: string;
  }>;
  genesisTime: string;
}

export interface NodeComponents {
  store: KVStore;
  wal: WriteAheadLog;
  stateManager: StateManager;
  stateIntegration: StateIntegration;
  consensus: ProofOfTruthEngine;
  mempool: Mempool;
  network: LegibleNode;
  bridge: ConsensusNetworkBridge;
  rpcServer: RPCServer | null;
}

// ============ Node Class ============

/**
 * Full Legible Money Node
 *
 * Orchestrates all components with proper persistence.
 */
export class LegibleMoneyNode extends EventEmitter {
  private config: NodeConfig;
  private genesis: GenesisConfig;
  private components: NodeComponents | null = null;
  private started = false;
  private lastFinalizedBlock: bigint = 0n;

  constructor(config: NodeConfig, genesis: GenesisConfig) {
    super();
    this.config = config;
    this.genesis = genesis;
  }

  // ============ Lifecycle ============

  /**
   * Initialize and start the node
   */
  async start(): Promise<void> {
    if (this.started) {
      throw new Error("Node already started");
    }

    console.log(`[Node] Starting ${this.config.node.name}...`);

    try {
      // Phase 1: Initialize storage
      console.log("[Node] Phase 1: Initializing storage...");
      const store = await this.initStorage();

      // Phase 1b: Initialize WAL and check for crash recovery
      console.log("[Node] Phase 1b: Initializing write-ahead log...");
      const wal = new WriteAheadLog(store);
      await wal.initialize();

      const uncommitted = await wal.findUncommittedEntries();
      if (uncommitted.length > 0) {
        console.warn(
          `[Node] Found ${uncommitted.length} uncommitted WAL entries from previous run`,
        );
        // Our persistBlock() uses a single atomic LevelDB batch, so either
        // all data wrote or none did. Mark these WAL entries as committed
        // since the data is either fully present or absent.
        for (const entry of uncommitted) {
          console.warn(
            `[Node] Marking WAL entry ${entry.id} (block ${entry.blockNumber}) as committed`,
          );
          await wal.markCommitted(entry.id);
        }
      }

      // Clean up old WAL entries
      const cleaned = await wal.cleanup(100);
      if (cleaned > 0) {
        console.log(`[Node] Cleaned up ${cleaned} old WAL entries`);
      }

      // Phase 2: Initialize state
      console.log("[Node] Phase 2: Initializing state manager...");
      const stateManager = new StateManager({}, store);

      // Try to load existing state from storage
      const hasExistingState = await stateManager.initializeFromStorage();
      if (hasExistingState) {
        console.log("[Node] Loaded existing state from storage");
      } else {
        console.log("[Node] Initializing fresh state from genesis");
        await this.initGenesisState(stateManager);
      }

      // Phase 3: Initialize consensus
      console.log("[Node] Phase 3: Initializing consensus...");
      const params = { ...DEFAULT_CONSENSUS_PARAMS, ...this.genesis.consensus };

      const initialValidators = this.genesis.initialValidators.map((v) => ({
        publicKey: v.publicKey as PublicKey,
        did: `did:lgm:${v.name}`,
        stake: BigInt(v.stake),
      }));

      const consensus = createIntegratedEngine({
        isValidator: this.config.validator.enabled,
        validatorKey: this.config.validator.privateKey as
          | PrivateKey
          | undefined,
        validatorPublicKey: this.config.validator.publicKey as
          | PublicKey
          | undefined,
        initialValidators,
        stateManager,
        params,
      });

      // Phase 4: Initialize mempool
      console.log("[Node] Phase 4: Initializing mempool...");
      const mempool = new Mempool({
        maxSize: this.config.mempool.maxSize,
        maxPerSender: this.config.mempool.maxPerSender,
      });

      // Phase 5: Initialize network
      console.log("[Node] Phase 5: Initializing network...");
      const network = new LegibleNode({
        listenAddresses: this.config.network.listenAddresses,
        bootstrapPeers: this.config.network.bootstrapPeers,
        maxConnections: this.config.network.maxPeers,
      });

      // Phase 6: Wire components together
      console.log("[Node] Phase 6: Wiring components...");
      const bridge = createConsensusBridge({
        node: network,
        consensus,
        mempool,
      });

      // Get state integration for persisting blocks
      const stateIntegration = consensus.getStateIntegration();
      if (!stateIntegration) {
        throw new Error("State integration not available from consensus");
      }

      // Phase 7: Initialize RPC server (if enabled)
      let rpcServer: RPCServer | null = null;
      if (this.config.rpc.enabled) {
        console.log("[Node] Phase 7: Initializing RPC server...");
        const handlerContext: HandlerContext = {
          state: stateManager,
          consensus,
          stateIntegration,
          nodeVersion: "0.1.0",
          networkId: "lgm-devnet-1",
          network,
        };
        rpcServer = new RPCServer(handlerContext, {
          host: this.config.rpc.host,
          port: this.config.rpc.port,
          cors: {
            enabled: this.config.rpc.cors.length > 0,
            origins: this.config.rpc.cors.filter((o: string) => o !== "*"),
          },
          // DevNet: increase rate limits for automated testing
          rateLimits: {
            windowMs: 60000,
            maxRequests: {
              free: 600,
              staked100: 3000,
              staked1000: 10000,
              validator: 100000,
            },
          },
        });
      }

      // Store components
      this.components = {
        store,
        wal,
        stateManager,
        stateIntegration,
        consensus,
        mempool,
        network,
        bridge,
        rpcServer,
      };

      // Phase 8: Start network
      console.log("[Node] Phase 8: Starting network...");
      await network.start();

      // Phase 9: Start RPC server
      if (rpcServer) {
        console.log(
          `[Node] Phase 9: Starting RPC on ${this.config.rpc.host}:${this.config.rpc.port}...`,
        );
        await rpcServer.start();
      }

      // Phase 10: Start consensus
      console.log("[Node] Phase 10: Starting consensus...");
      consensus.start();

      // Setup block persistence
      this.setupBlockPersistence();

      this.started = true;
      console.log(`[Node] ${this.config.node.name} started successfully`);
      this.emit("started", { name: this.config.node.name });
    } catch (error) {
      console.error("[Node] Failed to start:", error);
      this.emit("error", error);
      throw error;
    }
  }

  /**
   * Stop the node gracefully
   */
  async stop(): Promise<void> {
    if (!this.started || !this.components) {
      return;
    }

    console.log(`[Node] Stopping ${this.config.node.name}...`);

    try {
      // Stop consensus
      this.components.consensus.stop();

      // Persist final state atomically with WAL
      console.log("[Node] Persisting final state...");
      const { wal, stateManager } = this.components;
      const blockNumber = this.lastFinalizedBlock;

      wal.beginTransaction(blockNumber);
      await wal.prepare();
      await stateManager.persistBlock(blockNumber);
      await wal.commit();

      // Clean up old WAL entries
      await wal.cleanup(10);

      // Stop RPC server
      if (this.components.rpcServer) {
        await this.components.rpcServer.stop();
      }

      // Stop network
      await this.components.network.stop();

      // Close storage
      await this.components.store.close();

      this.started = false;
      this.components = null;

      console.log(`[Node] ${this.config.node.name} stopped`);
      this.emit("stopped");
    } catch (error) {
      console.error("[Node] Error during shutdown:", error);
      this.emit("error", error);
    }
  }

  // ============ Component Access ============

  getComponents(): NodeComponents | null {
    return this.components;
  }

  isRunning(): boolean {
    return this.started;
  }

  // ============ Private Methods ============

  /**
   * Initialize storage based on config
   * B1 FIX: This is the key integration point for persistence
   */
  private async initStorage(): Promise<KVStore> {
    const { storage, node } = this.config;

    if (storage.type === "memory") {
      console.log(
        "[Node] Using in-memory storage (data will be lost on restart)",
      );
      return new MemoryStore();
    }

    if (storage.type === "leveldb") {
      const path = storage.path || `${node.dataDir}/state`;
      console.log(`[Node] Using LevelDB storage at: ${path}`);

      const store = await createStorage({
        type: "leveldb",
        path,
      });

      return store;
    }

    throw new Error(`Unknown storage type: ${storage.type}`);
  }

  /**
   * Initialize state from genesis configuration
   */
  private async initGenesisState(stateManager: StateManager): Promise<void> {
    // Helper to generate operationalKeyHash from DID
    const { sha512_256Hex } = await import("../../core/crypto/hashing.js");
    const generateKeyHash = (did: string) =>
      sha512_256Hex(`${did}:operational:genesis`);

    // Initialize accounts from genesis
    for (const account of this.genesis.initialAccounts) {
      const operationalKeyHash = generateKeyHash(account.did);
      stateManager.createAccount(
        account.did,
        operationalKeyHash,
        BigInt(account.balance),
      );
    }

    // Initialize validators from genesis
    for (const validator of this.genesis.initialValidators) {
      const did = `did:lgm:${validator.name}`;
      stateManager.registerValidator(
        validator.publicKey as PublicKey,
        did,
        BigInt(validator.stake),
      );
    }

    // Initialize axioms from genesis (foundational facts with confidence 1.0)
    for (const axiom of this.genesis.genesisAxioms) {
      stateManager.addFact({
        id: axiom.id,
        claim: axiom.claim,
        domain: axiom.domain,
        confidence: 1.0,
        verifiedAt: 0n,
        verifierCount: 0,
        challengeCount: 0,
        citationCount: 0,
        status: "active",
      });
    }

    // Persist genesis state atomically
    await stateManager.persistBlock(0n);

    console.log(
      `[Node] Genesis state initialized: ${this.genesis.initialAccounts.length} accounts, ${this.genesis.initialValidators.length} validators, ${this.genesis.genesisAxioms.length} axioms`,
    );
  }

  /**
   * Setup block persistence - persist state after each finalized block
   * Uses WAL to fence each persist: if we crash between prepare and commit,
   * recovery knows block N was in-flight.
   */
  private setupBlockPersistence(): void {
    if (!this.components) return;

    const { consensus, stateManager, wal } = this.components;

    // Persist state after each finalized block
    consensus.on(
      "block_finalized",
      async (event: { blockNumber: bigint; blockHash: string }) => {
        try {
          const blockNumber = event.blockNumber;

          // WAL fence: begin → prepare → atomic persist → commit
          wal.beginTransaction(blockNumber);
          await wal.prepare();
          const count = await stateManager.persistBlock(blockNumber);
          await wal.commit();

          this.lastFinalizedBlock = blockNumber;

          if (this.config.node.logLevel === "debug") {
            console.log(
              `[Node] Persisted block ${blockNumber}: ${count} entries`,
            );
          }
        } catch (error) {
          console.error("[Node] Failed to persist block state:", error);
          // Attempt WAL rollback on failure
          try {
            await wal.rollback();
          } catch (rollbackError) {
            console.error("[Node] WAL rollback failed:", rollbackError);
          }
          this.emit("error", error);
        }
      },
    );
  }
}

// ============ Factory Functions ============

/**
 * Create a node from config files
 */
export async function createNode(
  configPath: string,
  genesisPath: string,
): Promise<LegibleMoneyNode> {
  const { readFileSync } = await import("fs");

  const config = JSON.parse(readFileSync(configPath, "utf-8")) as NodeConfig;
  const genesis = JSON.parse(
    readFileSync(genesisPath, "utf-8"),
  ) as GenesisConfig;

  return new LegibleMoneyNode(config, genesis);
}

/**
 * Create a devnet node with in-memory storage
 */
export function createDevnetNode(
  name: string,
  genesis: GenesisConfig,
  options?: {
    rpcPort?: number;
    p2pPort?: number;
    isValidator?: boolean;
    validatorKey?: string;
    validatorPublicKey?: string;
  },
): LegibleMoneyNode {
  const config: NodeConfig = {
    node: {
      name,
      dataDir: `/tmp/legible-devnet/${name}`,
      logLevel: "info",
    },
    storage: {
      type: "memory", // Fast startup for devnet
    },
    network: {
      listenAddresses: [`/ip4/0.0.0.0/tcp/${options?.p2pPort ?? 26656}`],
      bootstrapPeers: [],
      maxPeers: 50,
    },
    rpc: {
      enabled: true,
      host: "0.0.0.0",
      port: options?.rpcPort ?? 8545,
      cors: ["http://localhost:3000"], // Secure default
    },
    websocket: {
      enabled: true,
      port: (options?.rpcPort ?? 8545) + 1,
    },
    validator: {
      enabled: options?.isValidator ?? false,
      privateKey: options?.validatorKey,
      publicKey: options?.validatorPublicKey,
    },
    mempool: {
      maxSize: 10000,
      maxPerSender: 100,
    },
    telemetry: {
      enabled: false,
    },
  };

  return new LegibleMoneyNode(config, genesis);
}

/**
 * Create a production node with LevelDB persistence
 */
export function createProductionNode(
  name: string,
  dataDir: string,
  genesis: GenesisConfig,
  options?: {
    rpcPort?: number;
    p2pPort?: number;
    isValidator?: boolean;
    validatorKey?: string;
    validatorPublicKey?: string;
    bootstrapPeers?: string[];
    corsOrigins?: string[];
  },
): LegibleMoneyNode {
  const config: NodeConfig = {
    node: {
      name,
      dataDir,
      logLevel: "info",
    },
    storage: {
      type: "leveldb",
      path: `${dataDir}/state`,
    },
    network: {
      listenAddresses: [`/ip4/0.0.0.0/tcp/${options?.p2pPort ?? 26656}`],
      bootstrapPeers: options?.bootstrapPeers ?? [],
      maxPeers: 100,
    },
    rpc: {
      enabled: true,
      host: "127.0.0.1", // Secure default - localhost only
      port: options?.rpcPort ?? 8545,
      cors: options?.corsOrigins ?? [], // No CORS by default in production
    },
    websocket: {
      enabled: true,
      port: (options?.rpcPort ?? 8545) + 1,
    },
    validator: {
      enabled: options?.isValidator ?? false,
      privateKey: options?.validatorKey,
      publicKey: options?.validatorPublicKey,
    },
    mempool: {
      maxSize: 50000,
      maxPerSender: 500,
    },
    telemetry: {
      enabled: true,
      prometheusPort: 9090,
    },
  };

  return new LegibleMoneyNode(config, genesis);
}
