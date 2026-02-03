#!/usr/bin/env node

/**
 * Legible Money Node
 *
 * Main entry point for running a Legible Money node.
 * Supports validator and full node modes.
 */

import { readFileSync, existsSync } from 'fs';
import { EventEmitter } from 'events';
import { StateManager, createStateFromGenesis, type GenesisConfig } from '../state/state_manager.js';
import { ProofOfTruthEngine } from '../consensus/pot/engine.js';
import { Mempool, DEFAULT_MEMPOOL_CONFIG } from '../consensus/mempool/pool.js';
import { RPCServer, type ServerConfig } from '../protocol/rpc/server.js';
import type { ConsensusParams, DEFAULT_CONSENSUS_PARAMS } from '../consensus/pot/types.js';
import type { PublicKey, PrivateKey } from '../core/crypto/keys.js';
import { LegibleNode } from '../network/node.js';
import { TopicManager, createBlockTopicHandler, createTransactionTopicHandler, createClaimTopicHandler, createVerificationTopicHandler } from '../network/pubsub/topics.js';
import { SyncManager, SyncProtocolHandler, SYNC_PROTOCOL } from '../network/protocols/sync.js';
import { PUBSUB_TOPICS, MESSAGE_TYPES, type NetworkMessage } from '../network/config.js';

// ============ Configuration ============

interface NodeConfig {
  node: {
    name: string;
    dataDir: string;
    logLevel: 'debug' | 'info' | 'warn' | 'error';
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
  }>;
  genesisAxioms: Array<{
    id: string;
    claim: string;
    domain: string;
  }>;
  genesisTime: string;
}

// ============ Node ============

class LegibleMoneyNode extends EventEmitter {
  private config: NodeConfig;
  private genesis: GenesisFile;
  private state: StateManager;
  private consensus: ProofOfTruthEngine;
  private mempool: Mempool;
  private rpcServer: RPCServer | null = null;
  private validatorKey: PrivateKey | null = null;
  private running = false;

  // Network layer
  private networkNode: LegibleNode | null = null;
  private topicManager: TopicManager | null = null;
  private syncManager: SyncManager | null = null;

  constructor(config: NodeConfig, genesis: GenesisFile) {
    super();
    this.config = config;
    this.genesis = genesis;

    // Initialize state from genesis
    this.state = this.initializeState();

    // Initialize mempool
    this.mempool = new Mempool({
      maxSize: config.mempool.maxSize,
      maxPerSender: config.mempool.maxPerSender,
    });

    // Initialize consensus engine
    this.consensus = this.initializeConsensus();
  }

  private initializeState(): StateManager {
    const genesisConfig: GenesisConfig = {
      accounts: this.genesis.initialAccounts.map(a => ({
        did: a.did,
        operationalKeyHash: '',
        balance: BigInt(a.balance),
      })),
      validators: this.genesis.initialValidators.map(v => ({
        publicKey: v.publicKey as PublicKey,
        did: `did:lgm-devnet:${v.name}`,
        stake: BigInt(v.stake),
      })),
      axioms: this.genesis.genesisAxioms,
    };

    return createStateFromGenesis(genesisConfig);
  }

  private initializeConsensus(): ProofOfTruthEngine {
    // Merge genesis consensus params with defaults
    const params = {
      ...DEFAULT_CONSENSUS_PARAMS,
      ...this.genesis.consensus,
    };

    return new ProofOfTruthEngine(
      this.state,
      this.mempool,
      params as ConsensusParams,
      this.config.validator.publicKey as PublicKey | undefined
    );
  }

  async start(): Promise<void> {
    if (this.running) return;

    this.log('info', `Starting Legible Money Node: ${this.config.node.name}`);
    this.log('info', `Chain: ${this.genesis.network.chainId}`);

    // Load validator key if validator mode
    if (this.config.validator.enabled) {
      await this.loadValidatorKey();
      this.log('info', `Validator mode enabled: ${this.config.validator.publicKey?.slice(0, 16)}...`);
    }

    // Start mempool
    this.mempool.start();
    this.log('info', 'Mempool started');

    // Start consensus engine
    await this.consensus.start();
    this.log('info', 'Consensus engine started');

    // Start RPC server
    if (this.config.rpc.enabled) {
      await this.startRpcServer();
    }

    // Start P2P networking (placeholder)
    await this.startNetworking();

    this.running = true;
    this.log('info', 'Node started successfully');

    // Setup consensus event handlers
    this.setupEventHandlers();
  }

  async stop(): Promise<void> {
    if (!this.running) return;

    this.log('info', 'Stopping node...');

    // Stop network node
    if (this.networkNode) {
      await this.networkNode.stop();
      this.networkNode = null;
    }

    if (this.rpcServer) {
      await this.rpcServer.stop();
    }

    await this.consensus.stop();
    this.mempool.stop();

    this.running = false;
    this.log('info', 'Node stopped');
  }

  private async loadValidatorKey(): Promise<void> {
    const keyPath = process.env.VALIDATOR_KEY_PATH || '/secrets/validator.key';

    if (!existsSync(keyPath)) {
      this.log('warn', `Validator key not found at ${keyPath}`);
      return;
    }

    const keyData = JSON.parse(readFileSync(keyPath, 'utf-8'));
    this.validatorKey = keyData.privateKey;
    this.config.validator.publicKey = keyData.publicKey;
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

    this.rpcServer = new RPCServer(
      {
        state: this.state,
        consensus: this.consensus,
        mempool: this.mempool,
        nodeVersion: this.genesis.network.version,
        networkId: this.genesis.network.chainId,
      },
      serverConfig
    );

    await this.rpcServer.start();
    this.log('info', `RPC server listening on ${this.config.rpc.host}:${this.config.rpc.port}`);

    if (this.config.websocket.enabled) {
      this.log('info', `WebSocket enabled on port ${this.config.websocket.port}`);
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
    this.topicManager.registerHandler(createBlockTopicHandler(
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
      }
    ));

    // Register transaction handler
    this.topicManager.registerHandler(createTransactionTopicHandler(
      async (tx) => {
        // Validate transaction
        return this.mempool.validate(tx as any);
      },
      async (tx) => {
        // Add to mempool
        await this.mempool.add(tx as any);
      }
    ));

    // Register claim handler
    this.topicManager.registerHandler(createClaimTopicHandler(
      async (claim) => {
        // Basic claim validation
        const c = claim as any;
        return c.claim && c.references && c.references.length > 0;
      },
      async (claim) => {
        // Submit claim to consensus
        await this.consensus.submitClaim(claim as any);
      }
    ));

    // Register verification handler
    this.topicManager.registerHandler(createVerificationTopicHandler(
      async (verification) => {
        // Validate verification message
        const v = verification as any;
        return v.roundId && v.validator;
      },
      async (verification) => {
        // Route to consensus based on type
        const v = verification as any;
        if (v.type === 'commit' || v.commitment) {
          await this.consensus.submitCommitment(v.roundId, v);
        } else if (v.type === 'reveal' || v.verdict) {
          await this.consensus.submitReveal(v.roundId, v);
        }
      }
    ));

    // Wire topic manager to network node
    this.networkNode.setTopicManager(this.topicManager);

    // Create sync manager
    this.syncManager = new SyncManager(
      this.state.getCurrentBlockNumber(),
      async (num) => this.state.getBlock(num),
      async (block) => {
        await this.consensus.applyBlock(block as any);
      },
      async (block) => {
        const result = await this.consensus.validateBlock(block as any);
        return result.valid;
      }
    );

    // Create sync protocol handler
    const syncHandler = new SyncProtocolHandler(this.syncManager);

    // Set request function for sync manager to fetch blocks from peers
    this.syncManager.setRequestFunction(async (peerId, start, count) => {
      return syncHandler.requestBlocksFromPeer(
        this.networkNode!.request.bind(this.networkNode),
        peerId,
        start,
        count
      );
    });

    // Register sync protocol handler
    await this.networkNode.registerProtocol(SYNC_PROTOCOL,
      (stream, peerId) => syncHandler.handleStream(stream, peerId)
    );

    this.networkNode.setSyncManager(this.syncManager);

    // Wire peer events to sync manager
    this.networkNode.on('peer:connect', (event: any) => {
      this.log('info', `Peer connected: ${event.peer.peerId.slice(0, 16)}...`);
      // Add peer to sync manager (will need head block info from handshake)
      this.syncManager?.addPeer({
        peerId: event.peer.peerId,
        headBlock: 0n, // Will be updated by status exchange
        latency: event.peer.latency || 0,
        score: event.peer.score || 0,
      });
    });

    this.networkNode.on('peer:disconnect', (event: any) => {
      this.log('info', `Peer disconnected: ${event.peer.peerId.slice(0, 16)}...`);
      this.syncManager?.removePeer(event.peer.peerId);
    });

    // Start the network node
    await this.networkNode.start();

    this.log('info', `P2P node started with peer ID: ${this.networkNode.getPeerId()?.slice(0, 16)}...`);

    const addrs = this.networkNode.getListenAddresses();
    for (const addr of addrs) {
      this.log('info', `P2P listening on: ${addr}`);
    }

    if (this.config.network.bootstrapPeers.length > 0) {
      this.log('info', `Connecting to ${this.config.network.bootstrapPeers.length} bootstrap peers`);
    }
  }

  private setupEventHandlers(): void {
    // Forward consensus events
    this.consensus.on('block:produced', async (block) => {
      this.log('info', `Produced block ${block.header.number}`);

      // Broadcast to RPC subscribers
      this.rpcServer?.broadcast('block', block);

      // Broadcast to P2P network
      if (this.networkNode) {
        const message: NetworkMessage = {
          type: MESSAGE_TYPES.NEW_BLOCK,
          version: '1.0.0',
          timestamp: Date.now(),
          sender: this.networkNode.getPeerId() || '',
          payload: new TextEncoder().encode(JSON.stringify(block)),
        };
        try {
          await this.networkNode.broadcast(PUBSUB_TOPICS.BLOCKS, message);
        } catch (error) {
          this.log('warn', `Failed to broadcast block: ${error}`);
        }
      }
    });

    this.consensus.on('block:finalized', (data) => {
      this.log('info', `Finalized block ${data.blockNumber}`);
    });

    this.consensus.on('round:started', async (round) => {
      this.log('info', `Verification round started: ${round.id}`);

      // Broadcast to P2P network
      if (this.networkNode) {
        const message: NetworkMessage = {
          type: MESSAGE_TYPES.NEW_CLAIM,
          version: '1.0.0',
          timestamp: Date.now(),
          sender: this.networkNode.getPeerId() || '',
          payload: new TextEncoder().encode(JSON.stringify(round)),
        };
        try {
          await this.networkNode.broadcast(PUBSUB_TOPICS.CLAIMS, message);
        } catch (error) {
          this.log('warn', `Failed to broadcast round: ${error}`);
        }
      }
    });

    this.consensus.on('round:completed', (result) => {
      this.log('info', `Verification round completed: ${result.claimId} -> ${result.outcome}`);
      if (result.outcome === 'accepted') {
        this.rpcServer?.broadcast('fact:accepted', result.fact);
      }
    });

    // Mempool events
    this.mempool.on('tx:added', async (entry) => {
      this.log('debug', `Transaction added to mempool: ${entry.txHash.slice(0, 16)}...`);

      // Broadcast transaction to P2P network
      if (this.networkNode) {
        const message: NetworkMessage = {
          type: MESSAGE_TYPES.NEW_TRANSACTION,
          version: '1.0.0',
          timestamp: Date.now(),
          sender: this.networkNode.getPeerId() || '',
          payload: new TextEncoder().encode(JSON.stringify(entry.tx)),
        };
        try {
          await this.networkNode.broadcast(PUBSUB_TOPICS.TRANSACTIONS, message);
        } catch (error) {
          this.log('warn', `Failed to broadcast transaction: ${error}`);
        }
      }
    });
  }

  private log(level: string, message: string): void {
    const levels = ['debug', 'info', 'warn', 'error'];
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

  getMempool(): Mempool {
    return this.mempool;
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
  const configPath = process.env.CONFIG_PATH || '/config/node.json';
  const genesisPath = process.env.GENESIS_PATH || '/config/genesis.json';

  if (!existsSync(configPath)) {
    console.error(`Config file not found: ${configPath}`);
    process.exit(1);
  }

  if (!existsSync(genesisPath)) {
    console.error(`Genesis file not found: ${genesisPath}`);
    process.exit(1);
  }

  const config: NodeConfig = JSON.parse(readFileSync(configPath, 'utf-8'));
  const genesis: GenesisFile = JSON.parse(readFileSync(genesisPath, 'utf-8'));

  // Create and start node
  const node = new LegibleMoneyNode(config, genesis);

  // Handle shutdown
  process.on('SIGINT', async () => {
    console.log('\nReceived SIGINT, shutting down...');
    await node.stop();
    process.exit(0);
  });

  process.on('SIGTERM', async () => {
    console.log('\nReceived SIGTERM, shutting down...');
    await node.stop();
    process.exit(0);
  });

  // Start node
  await node.start();

  // Keep running
  console.log('\nNode is running. Press Ctrl+C to stop.\n');
}

// Run
main().catch((error) => {
  console.error('Fatal error:', error);
  process.exit(1);
});

export { LegibleMoneyNode, NodeConfig, GenesisFile };
