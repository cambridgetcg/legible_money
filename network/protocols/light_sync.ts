/**
 * Light Sync Protocol
 *
 * Lightweight synchronization for chatnode instances.
 * Syncs only block headers and knowledge tree, enabling
 * API chat sessions without full state download.
 *
 * Sync Strategy:
 * 1. Download block headers (proof of chain)
 * 2. Download knowledge tree root and proofs
 * 3. Subscribe to knowledge updates via GossipSub
 * 4. On-demand fact retrieval with Merkle proofs
 *
 * PP HIERARCHY:
 *   TRUTH: Verifies knowledge proofs against header roots
 *   UNDERSTANDING: Enables knowledge queries without full state
 *   BEAUTY: Minimal sync for maximal utility
 *   JUSTICE: Placed in network/protocols alongside full sync
 *   CREATIVITY: Novel light client for knowledge-centric queries
 */

import { EventEmitter } from "events";
import { pipe } from "it-pipe";
import * as lp from "it-length-prefixed";
import type { Stream } from "@libp2p/interface";
import { SYNC_CONFIG, type SyncPeer, type SyncStatus } from "./sync.js";
import { PROTOCOL_IDS, networkReplacer } from "../config.js";
import { sha512_256Hex } from "../../core/crypto/hashing.js";

// ============================================================================
// Types
// ============================================================================

/**
 * Light sync configuration.
 */
export interface LightSyncConfig {
  /** Sync block headers */
  syncHeaders: boolean;
  /** Sync knowledge tree */
  syncKnowledgeTree: boolean;
  /** Sync account tree (disabled for chatnode) */
  syncAccountTree: boolean;
  /** Sync validator tree (disabled for chatnode) */
  syncValidatorTree: boolean;
  /** Maximum blocks behind before triggering resync */
  maxBlocksBehind: number;
  /** Batch size for header requests */
  headerBatchSize: number;
  /** Knowledge tree sync depth (0 = root only) */
  knowledgeTreeDepth: number;
  /** Subscribe to live knowledge updates */
  subscribeLiveUpdates: boolean;
}

export const DEFAULT_LIGHT_SYNC_CONFIG: LightSyncConfig = {
  syncHeaders: true,
  syncKnowledgeTree: true,
  syncAccountTree: false,
  syncValidatorTree: false,
  maxBlocksBehind: 100,
  headerBatchSize: 100,
  knowledgeTreeDepth: 3,
  subscribeLiveUpdates: true,
};

/**
 * Block header for light sync.
 */
export interface BlockHeader {
  /** Block number */
  number: bigint;
  /** Block hash */
  hash: string;
  /** Parent block hash */
  parentHash: string;
  /** State root (accounts + validators + knowledge) */
  stateRoot: string;
  /** Knowledge tree root */
  knowledgeRoot: string;
  /** Block timestamp */
  timestamp: number;
  /** Block producer */
  producer: string;
}

/**
 * Knowledge tree node for sync.
 */
export interface KnowledgeTreeNode {
  /** Node hash */
  hash: string;
  /** Left child hash (if branch) */
  left?: string;
  /** Right child hash (if branch) */
  right?: string;
  /** Leaf data (if leaf) */
  data?: {
    factId: string;
    domain: string;
    confidence: number;
    status: string;
  };
}

/**
 * Knowledge proof for fact verification.
 */
export interface KnowledgeProof {
  /** Fact ID */
  factId: string;
  /** Merkle proof path */
  proof: string[];
  /** Leaf data */
  leaf: KnowledgeTreeNode;
  /** Root hash (should match block's knowledgeRoot) */
  root: string;
  /** Block number */
  blockNumber: bigint;
}

/**
 * Light sync request types.
 */
export interface LightSyncRequest {
  type: "headers" | "knowledge_root" | "knowledge_proof" | "knowledge_nodes";
  startBlock?: bigint;
  count?: number;
  factId?: string;
  nodeHashes?: string[];
}

/**
 * Light sync response types.
 */
export interface LightSyncResponse {
  type: "headers" | "knowledge_root" | "knowledge_proof" | "knowledge_nodes";
  data:
    | BlockHeader[]
    | KnowledgeTreeNode
    | KnowledgeProof
    | KnowledgeTreeNode[];
}

/**
 * Light sync status.
 */
export interface LightSyncStatus extends SyncStatus {
  /** Headers synced */
  headersSynced: boolean;
  /** Knowledge root synced */
  knowledgeRootSynced: boolean;
  /** Latest knowledge root */
  latestKnowledgeRoot?: string;
  /** Subscribed to live updates */
  subscribedToUpdates: boolean;
  /** Pending proof requests */
  pendingProofs: number;
}

// ============================================================================
// Protocol Constants
// ============================================================================

export const LIGHT_SYNC_PROTOCOL = PROTOCOL_IDS.LIGHT_SYNC;

export const LIGHT_SYNC_TOPICS = {
  /** Knowledge tree updates (new facts, confidence changes) */
  KNOWLEDGE_UPDATES: "lgm/knowledge/updates/1.0.0",
  /** Block header announcements */
  HEADER_ANNOUNCEMENTS: "lgm/headers/1.0.0",
};

// ============================================================================
// Light Sync Manager
// ============================================================================

/**
 * Manages light synchronization for chatnode instances.
 */
export class LightSyncManager extends EventEmitter {
  private config: LightSyncConfig;
  private status: LightSyncStatus;
  private peers: Map<string, SyncPeer> = new Map();
  private syncing: boolean = false;
  private aborted: boolean = false;

  /** Cached block headers */
  private headers: Map<bigint, BlockHeader> = new Map();

  /** Cached knowledge tree nodes */
  private knowledgeNodes: Map<string, KnowledgeTreeNode> = new Map();

  /** Pending proof requests */
  private pendingProofs: Map<string, Promise<KnowledgeProof>> = new Map();

  /** Subscribe callback for knowledge updates */
  private knowledgeUpdateCallback?: (update: KnowledgeUpdate) => void;

  constructor(
    currentBlock: bigint = 0n,
    private saveHeader: (header: BlockHeader) => Promise<void>,
    _getHeaderFn: (num: bigint) => Promise<BlockHeader | undefined>,
    private validateHeader: (
      header: BlockHeader,
      parent?: BlockHeader,
    ) => Promise<boolean>,
    config: LightSyncConfig = DEFAULT_LIGHT_SYNC_CONFIG,
  ) {
    super();
    this.config = { ...config };
    this.status = {
      currentBlock,
      highestBlock: currentBlock,
      startingBlock: currentBlock,
      syncing: false,
      progress: 1.0,
      headersSynced: false,
      knowledgeRootSynced: false,
      subscribedToUpdates: false,
      pendingProofs: 0,
    };
  }

  // ============================================================================
  // Status
  // ============================================================================

  /**
   * Get current light sync status.
   */
  getStatus(): LightSyncStatus {
    return { ...this.status };
  }

  /**
   * Check if fully synced.
   */
  isSynced(): boolean {
    return (
      this.status.headersSynced &&
      this.status.knowledgeRootSynced &&
      this.status.currentBlock >=
        this.status.highestBlock - BigInt(this.config.maxBlocksBehind)
    );
  }

  // ============================================================================
  // Peer Management
  // ============================================================================

  /**
   * Add a sync peer.
   */
  addPeer(peer: SyncPeer): void {
    this.peers.set(peer.peerId, peer);
    this.updateHighestBlock();
    this.emit("peer:added", peer);

    // Start sync if we're behind
    if (!this.syncing && this.shouldSync()) {
      this.startSync();
    }
  }

  /**
   * Remove a sync peer.
   */
  removePeer(peerId: string): void {
    this.peers.delete(peerId);
    this.updateHighestBlock();
    this.emit("peer:removed", peerId);
  }

  /**
   * Update peer's head block.
   */
  updatePeerHead(peerId: string, headBlock: bigint): void {
    const peer = this.peers.get(peerId);
    if (peer) {
      peer.headBlock = headBlock;
      this.updateHighestBlock();
    }
  }

  // ============================================================================
  // Header Sync
  // ============================================================================

  /**
   * Start light synchronization.
   */
  async startSync(): Promise<void> {
    if (this.syncing) return;

    this.syncing = true;
    this.aborted = false;
    this.status.syncing = true;
    this.status.startingBlock = this.status.currentBlock;

    this.emit("sync:started", this.status);

    try {
      // Phase 1: Sync headers
      if (this.config.syncHeaders) {
        await this.syncHeaders();
      }

      // Phase 2: Sync knowledge root
      if (this.config.syncKnowledgeTree) {
        await this.syncKnowledgeRoot();
      }

      // Phase 3: Subscribe to live updates
      if (this.config.subscribeLiveUpdates) {
        this.subscribeToUpdates();
      }

      this.status.headersSynced = true;
      this.status.knowledgeRootSynced = true;
      this.emit("sync:completed", this.status);
    } catch (error) {
      this.emit("sync:error", error);
    } finally {
      this.syncing = false;
      this.status.syncing = false;
    }
  }

  /**
   * Stop synchronization.
   */
  stopSync(): void {
    this.aborted = true;
    this.emit("sync:stopped");
  }

  /**
   * Sync block headers from peers.
   */
  private async syncHeaders(): Promise<void> {
    this.emit("sync:headers_start");

    while (
      !this.aborted &&
      this.status.currentBlock < this.status.highestBlock
    ) {
      const peer = this.selectBestPeer();
      if (!peer) {
        await this.delay(SYNC_CONFIG.POLL_INTERVAL);
        continue;
      }

      const startBlock = this.status.currentBlock + 1n;
      const count = Math.min(
        this.config.headerBatchSize,
        Number(this.status.highestBlock - this.status.currentBlock),
      );

      try {
        const headers = await this.requestHeaders(
          peer.peerId,
          startBlock,
          count,
        );
        await this.processHeaders(headers);
      } catch (error) {
        peer.score -= 10;
        this.emit("sync:peer_error", { peer, error });
      }

      this.updateProgress();
    }

    this.emit("sync:headers_complete");
  }

  /**
   * Process received headers.
   */
  private async processHeaders(headers: BlockHeader[]): Promise<void> {
    for (const header of headers) {
      if (this.aborted) break;

      // Get parent for validation
      const parent = this.headers.get(header.number - 1n);

      // Validate header
      const valid = await this.validateHeader(header, parent);
      if (!valid) {
        throw new Error(`Invalid header at block ${header.number}`);
      }

      // Save and cache header
      await this.saveHeader(header);
      this.headers.set(header.number, header);
      this.status.currentBlock = header.number;

      this.emit("sync:header", header);
    }
  }

  // ============================================================================
  // Knowledge Tree Sync
  // ============================================================================

  /**
   * Sync knowledge tree root from latest block.
   */
  private async syncKnowledgeRoot(): Promise<void> {
    this.emit("sync:knowledge_start");

    const peer = this.selectBestPeer();
    if (!peer) {
      throw new Error("No peers available for knowledge sync");
    }

    // Get latest header
    const latestHeader = this.headers.get(this.status.currentBlock);
    if (!latestHeader) {
      throw new Error("No header available for knowledge sync");
    }

    // Request knowledge root node
    const rootNode = await this.requestKnowledgeRoot(
      peer.peerId,
      latestHeader.knowledgeRoot,
    );
    this.knowledgeNodes.set(rootNode.hash, rootNode);
    this.status.latestKnowledgeRoot = rootNode.hash;

    // Optionally sync deeper nodes
    if (this.config.knowledgeTreeDepth > 0) {
      await this.syncKnowledgeNodes(
        peer.peerId,
        rootNode,
        this.config.knowledgeTreeDepth,
      );
    }

    this.emit("sync:knowledge_complete", { root: rootNode.hash });
  }

  /**
   * Recursively sync knowledge tree nodes to specified depth.
   */
  private async syncKnowledgeNodes(
    peerId: string,
    node: KnowledgeTreeNode,
    depth: number,
  ): Promise<void> {
    if (depth <= 0 || this.aborted) return;

    const childHashes: string[] = [];
    if (node.left) childHashes.push(node.left);
    if (node.right) childHashes.push(node.right);

    if (childHashes.length === 0) return;

    // Request child nodes
    const childNodes = await this.requestKnowledgeNodes(peerId, childHashes);

    for (const child of childNodes) {
      this.knowledgeNodes.set(child.hash, child);
      await this.syncKnowledgeNodes(peerId, child, depth - 1);
    }
  }

  // ============================================================================
  // On-Demand Proof Retrieval
  // ============================================================================

  /**
   * Get knowledge proof for a fact.
   * Returns cached proof or fetches from peer.
   */
  async getKnowledgeProof(factId: string): Promise<KnowledgeProof> {
    // Check for pending request
    const pending = this.pendingProofs.get(factId);
    if (pending) {
      return pending;
    }

    // Create new request
    const promise = this.fetchKnowledgeProof(factId);
    this.pendingProofs.set(factId, promise);
    this.status.pendingProofs = this.pendingProofs.size;

    try {
      const proof = await promise;
      return proof;
    } finally {
      this.pendingProofs.delete(factId);
      this.status.pendingProofs = this.pendingProofs.size;
    }
  }

  /**
   * Fetch knowledge proof from peer.
   */
  private async fetchKnowledgeProof(factId: string): Promise<KnowledgeProof> {
    const peer = this.selectBestPeer();
    if (!peer) {
      throw new Error("No peers available for proof request");
    }

    const proof = await this.requestKnowledgeProof(peer.peerId, factId);

    // Verify proof against current knowledge root
    const valid = this.verifyKnowledgeProof(proof);
    if (!valid) {
      throw new Error(`Invalid proof for fact ${factId}`);
    }

    return proof;
  }

  /**
   * Verify a knowledge proof against the current root.
   */
  verifyKnowledgeProof(proof: KnowledgeProof): boolean {
    if (proof.root !== this.status.latestKnowledgeRoot) {
      return false;
    }

    // Verify Merkle path
    let currentHash = proof.leaf.hash;
    for (const sibling of proof.proof) {
      // Combine hashes (simplified - actual implementation would use proper hashing)
      currentHash = this.hashNodes(currentHash, sibling);
    }

    return currentHash === proof.root;
  }

  private hashNodes(left: string, right: string): string {
    return sha512_256Hex(Buffer.from(left + right));
  }

  // ============================================================================
  // Live Updates
  // ============================================================================

  /**
   * Subscribe to knowledge updates via GossipSub.
   */
  private subscribeToUpdates(): void {
    this.status.subscribedToUpdates = true;
    this.emit("sync:subscribed", LIGHT_SYNC_TOPICS.KNOWLEDGE_UPDATES);
  }

  /**
   * Handle incoming knowledge update from GossipSub.
   */
  handleKnowledgeUpdate(update: KnowledgeUpdate): void {
    this.emit("knowledge:update", update);

    // Update cached nodes if relevant
    if (update.type === "fact_added" || update.type === "confidence_changed") {
      // Invalidate affected cache entries
      this.knowledgeNodes.delete(update.affectedRoot);
    }

    // Notify callback if set
    if (this.knowledgeUpdateCallback) {
      this.knowledgeUpdateCallback(update);
    }
  }

  /**
   * Set callback for knowledge updates.
   */
  onKnowledgeUpdate(callback: (update: KnowledgeUpdate) => void): void {
    this.knowledgeUpdateCallback = callback;
  }

  /**
   * Handle new header announcement.
   */
  handleHeaderAnnouncement(header: BlockHeader): void {
    // Update highest block
    if (header.number > this.status.highestBlock) {
      this.status.highestBlock = header.number;

      // Trigger sync if too far behind
      if (!this.syncing && this.shouldSync()) {
        this.startSync();
      }
    }

    this.emit("header:announced", header);
  }

  // ============================================================================
  // Request Functions (to be wired by network layer)
  // ============================================================================

  private requestFn?: <T>(
    peerId: string,
    protocol: string,
    request: LightSyncRequest,
    timeout?: number,
  ) => Promise<T>;

  /**
   * Set the request function for peer communication.
   */
  setRequestFunction(
    fn: <T>(
      peerId: string,
      protocol: string,
      request: LightSyncRequest,
      timeout?: number,
    ) => Promise<T>,
  ): void {
    this.requestFn = fn;
  }

  private async requestHeaders(
    peerId: string,
    startBlock: bigint,
    count: number,
  ): Promise<BlockHeader[]> {
    if (!this.requestFn) {
      throw new Error("Request function not set");
    }

    const response = await this.requestFn<LightSyncResponse>(
      peerId,
      LIGHT_SYNC_PROTOCOL,
      { type: "headers", startBlock, count },
      SYNC_CONFIG.REQUEST_TIMEOUT,
    );

    return response.data as BlockHeader[];
  }

  private async requestKnowledgeRoot(
    peerId: string,
    rootHash: string,
  ): Promise<KnowledgeTreeNode> {
    if (!this.requestFn) {
      throw new Error("Request function not set");
    }

    const response = await this.requestFn<LightSyncResponse>(
      peerId,
      LIGHT_SYNC_PROTOCOL,
      { type: "knowledge_root", nodeHashes: [rootHash] },
      SYNC_CONFIG.REQUEST_TIMEOUT,
    );

    return response.data as KnowledgeTreeNode;
  }

  private async requestKnowledgeNodes(
    peerId: string,
    hashes: string[],
  ): Promise<KnowledgeTreeNode[]> {
    if (!this.requestFn) {
      throw new Error("Request function not set");
    }

    const response = await this.requestFn<LightSyncResponse>(
      peerId,
      LIGHT_SYNC_PROTOCOL,
      { type: "knowledge_nodes", nodeHashes: hashes },
      SYNC_CONFIG.REQUEST_TIMEOUT,
    );

    return response.data as KnowledgeTreeNode[];
  }

  private async requestKnowledgeProof(
    peerId: string,
    factId: string,
  ): Promise<KnowledgeProof> {
    if (!this.requestFn) {
      throw new Error("Request function not set");
    }

    const response = await this.requestFn<LightSyncResponse>(
      peerId,
      LIGHT_SYNC_PROTOCOL,
      { type: "knowledge_proof", factId },
      SYNC_CONFIG.REQUEST_TIMEOUT,
    );

    return response.data as KnowledgeProof;
  }

  // ============================================================================
  // Helpers
  // ============================================================================

  private selectBestPeer(): SyncPeer | null {
    let best: SyncPeer | null = null;
    for (const peer of this.peers.values()) {
      if (peer.headBlock > this.status.currentBlock) {
        if (!best || peer.score > best.score) {
          best = peer;
        }
      }
    }
    return best;
  }

  private updateHighestBlock(): void {
    let highest = this.status.currentBlock;
    for (const peer of this.peers.values()) {
      if (peer.headBlock > highest) {
        highest = peer.headBlock;
      }
    }
    this.status.highestBlock = highest;
  }

  private updateProgress(): void {
    const total = this.status.highestBlock - this.status.startingBlock;
    const done = this.status.currentBlock - this.status.startingBlock;
    this.status.progress = total > 0n ? Number(done) / Number(total) : 1.0;
    this.emit("sync:progress", this.status.progress);
  }

  private shouldSync(): boolean {
    return (
      this.status.highestBlock - this.status.currentBlock >
      BigInt(this.config.maxBlocksBehind)
    );
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  /**
   * Get cached header.
   */
  getCachedHeader(blockNumber: bigint): BlockHeader | undefined {
    return this.headers.get(blockNumber);
  }

  /**
   * Get cached knowledge node.
   */
  getCachedKnowledgeNode(hash: string): KnowledgeTreeNode | undefined {
    return this.knowledgeNodes.get(hash);
  }

  /**
   * Clear caches.
   */
  clearCaches(): void {
    this.headers.clear();
    this.knowledgeNodes.clear();
  }
}

// ============================================================================
// Knowledge Update Types
// ============================================================================

/**
 * Knowledge update from GossipSub.
 */
export interface KnowledgeUpdate {
  /** Update type */
  type: "fact_added" | "fact_removed" | "confidence_changed" | "status_changed";
  /** Fact ID */
  factId: string;
  /** Domain ID */
  domain: string;
  /** New confidence (if changed) */
  newConfidence?: number;
  /** New status (if changed) */
  newStatus?: string;
  /** Block number of update */
  blockNumber: bigint;
  /** Affected knowledge root (for cache invalidation) */
  affectedRoot: string;
}

// ============================================================================
// Light Sync Protocol Handler
// ============================================================================

/**
 * Protocol handler for light sync requests/responses.
 */
export class LightSyncProtocolHandler {
  constructor(
    _lightSyncManager: LightSyncManager,
    private getHeader: (num: bigint) => Promise<BlockHeader | undefined>,
    private getKnowledgeNode: (
      hash: string,
    ) => Promise<KnowledgeTreeNode | undefined>,
    private getKnowledgeProof: (
      factId: string,
    ) => Promise<KnowledgeProof | undefined>,
  ) {}

  /**
   * Handle incoming light sync stream from a peer.
   */
  async handleStream(stream: Stream, peerId: string): Promise<void> {
    try {
      let requestData: Uint8Array | undefined;

      // Cast stream to any for libp2p stream interface compatibility
      const streamAny = stream as any;

      await pipe(streamAny.source, lp.decode, async function (source) {
        for await (const data of source) {
          requestData = data.subarray();
          break;
        }
      });

      if (!requestData) {
        console.warn(`[LightSync] Empty request from ${peerId}`);
        return;
      }

      const request: LightSyncRequest = JSON.parse(
        new TextDecoder().decode(requestData),
      );
      const response = await this.handleRequest(request);
      const responseData = new TextEncoder().encode(
        JSON.stringify(response, networkReplacer),
      );

      await pipe([responseData], lp.encode, streamAny.sink);
    } catch (error) {
      console.error(`[LightSync] Error handling stream from ${peerId}:`, error);
    }
  }

  /**
   * Handle a light sync request.
   */
  async handleRequest(request: LightSyncRequest): Promise<LightSyncResponse> {
    switch (request.type) {
      case "headers": {
        const headers: BlockHeader[] = [];
        const start = request.startBlock ?? 0n;
        const count = request.count ?? 100;

        for (let i = 0; i < count; i++) {
          const header = await this.getHeader(start + BigInt(i));
          if (header) {
            headers.push(header);
          } else {
            break;
          }
        }

        return { type: "headers", data: headers };
      }

      case "knowledge_root":
      case "knowledge_nodes": {
        const nodes: KnowledgeTreeNode[] = [];
        for (const hash of request.nodeHashes ?? []) {
          const node = await this.getKnowledgeNode(hash);
          if (node) {
            nodes.push(node);
          }
        }

        if (request.type === "knowledge_root" && nodes.length > 0) {
          return { type: "knowledge_root", data: nodes[0] };
        }

        return { type: "knowledge_nodes", data: nodes };
      }

      case "knowledge_proof": {
        if (!request.factId) {
          throw new Error("factId required for knowledge_proof request");
        }

        const proof = await this.getKnowledgeProof(request.factId);
        if (!proof) {
          throw new Error(`Proof not found for fact ${request.factId}`);
        }

        return { type: "knowledge_proof", data: proof };
      }

      default:
        throw new Error(`Unknown request type: ${request.type}`);
    }
  }
}
