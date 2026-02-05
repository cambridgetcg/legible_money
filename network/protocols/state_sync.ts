/**
 * State Sync Protocol
 *
 * Network protocol for state synchronization between nodes.
 * Enables new validators to join by downloading state snapshots.
 *
 * B2 FIX: Implements the missing state sync protocol
 *
 * Sync Strategy:
 * 1. Request available snapshots from peers
 * 2. Select best snapshot (highest block with enough seeders)
 * 3. Download snapshot chunks in parallel
 * 4. Verify chunks against snapshot metadata
 * 5. Apply snapshot to local state
 * 6. Resume block sync from snapshot block
 *
 * PP HIERARCHY:
 *   TRUTH: All chunks verified against hashes
 *   UNDERSTANDING: Clear request/response protocol
 *   BEAUTY: Parallel chunk downloads
 *   JUSTICE: Placed in protocols/ alongside other sync
 *   CREATIVITY: Adaptive sync strategy (snapshot vs block)
 */

import { EventEmitter } from "events";
import { pipe } from "it-pipe";
import * as lp from "it-length-prefixed";
import type { Stream } from "@libp2p/interface";
import { sha512_256Hex } from "../../core/crypto/hashing.js";
import type {
  StateSnapshot,
  SnapshotChunk,
} from "../../state/storage/types.js";
import { SnapshotManager } from "../../state/storage/snapshots.js";
import { SYNC_CONFIG } from "./sync.js";
import { PROTOCOL_IDS, networkReplacer, networkReviver } from "../config.js";

// ============ Protocol Constants ============

export const STATE_SYNC_PROTOCOL = PROTOCOL_IDS.STATE_SYNC;

/**
 * State sync configuration
 */
export const STATE_SYNC_CONFIG = {
  /** Minimum blocks behind before switching to state sync */
  SNAPSHOT_THRESHOLD: 1000n,

  /** Maximum concurrent chunk downloads */
  MAX_CONCURRENT_CHUNKS: 8,

  /** Timeout for chunk requests (ms) */
  CHUNK_TIMEOUT: 60000,

  /** Minimum peers with snapshot before downloading */
  MIN_SNAPSHOT_PEERS: 2,

  /** Retry count for failed chunk downloads */
  CHUNK_RETRY_COUNT: 3,

  /** Delay between chunk retries (ms) */
  CHUNK_RETRY_DELAY: 1000,
};

// ============ Types ============

/**
 * State sync request types
 */
export interface StateSyncRequest {
  type: "list_snapshots" | "snapshot_metadata" | "chunk";
  snapshotId?: string;
  chunkIndex?: number;
}

/**
 * State sync response types
 */
export interface StateSyncResponse {
  type: "snapshot_list" | "snapshot_metadata" | "chunk" | "error";
  snapshots?: SnapshotInfo[];
  snapshot?: StateSnapshot;
  chunk?: SnapshotChunk;
  error?: string;
}

/**
 * Snapshot info advertised to peers
 */
export interface SnapshotInfo {
  id: string;
  blockNumber: bigint;
  blockHash: string;
  stateRoot: string;
  chunkCount: number;
  compressedSize: bigint;
}

/**
 * Peer snapshot availability
 */
export interface PeerSnapshotInfo {
  peerId: string;
  snapshots: SnapshotInfo[];
  lastUpdated: number;
}

/**
 * State sync status
 */
export interface StateSyncStatus {
  syncing: boolean;
  snapshotId?: string;
  totalChunks: number;
  downloadedChunks: number;
  verifiedChunks: number;
  appliedChunks: number;
  bytesDownloaded: bigint;
  currentPhase:
    | "idle"
    | "selecting"
    | "downloading"
    | "verifying"
    | "applying"
    | "complete";
  progress: number;
  error?: string;
}

/**
 * Chunk download task
 */
interface ChunkTask {
  snapshotId: string;
  chunkIndex: number;
  expectedHash: string;
  retryCount: number;
  assignedPeer?: string;
}

// ============ State Sync Manager ============

/**
 * Manages state synchronization via snapshots
 *
 * B2 FIX: Core component for new validator sync
 */
export class StateSyncManager extends EventEmitter {
  private status: StateSyncStatus;
  private peerSnapshots: Map<string, PeerSnapshotInfo> = new Map();
  private syncing = false;
  private aborted = false;

  // Download tracking
  private pendingChunks: ChunkTask[] = [];
  private downloadingChunks: Map<number, ChunkTask> = new Map();
  private completedChunks: Map<number, SnapshotChunk> = new Map();

  constructor(
    private snapshotManager: SnapshotManager,
    private requestFn?: <T>(
      peerId: string,
      protocol: string,
      request: StateSyncRequest,
      timeout?: number,
    ) => Promise<T>,
  ) {
    super();
    this.status = this.initialStatus();
  }

  // ============ Status ============

  getStatus(): StateSyncStatus {
    return { ...this.status };
  }

  /**
   * Check if state sync should be used instead of block sync
   */
  shouldUseStateSync(currentBlock: bigint, targetBlock: bigint): boolean {
    const blocksBehind = targetBlock - currentBlock;
    return blocksBehind >= STATE_SYNC_CONFIG.SNAPSHOT_THRESHOLD;
  }

  // ============ Peer Management ============

  /**
   * Add peer's snapshot info
   */
  addPeerSnapshots(peerId: string, snapshots: SnapshotInfo[]): void {
    this.peerSnapshots.set(peerId, {
      peerId,
      snapshots,
      lastUpdated: Date.now(),
    });
    this.emit("peer:snapshots", { peerId, snapshots });
  }

  /**
   * Remove peer's snapshot info
   */
  removePeer(peerId: string): void {
    this.peerSnapshots.delete(peerId);
  }

  /**
   * Get best snapshot to download
   * Prefers highest block number with multiple seeders
   */
  selectBestSnapshot(): { snapshot: SnapshotInfo; peers: string[] } | null {
    // Collect all snapshots with their available peers
    const snapshotPeers: Map<
      string,
      { snapshot: SnapshotInfo; peers: string[] }
    > = new Map();

    for (const [peerId, peerInfo] of this.peerSnapshots) {
      for (const snapshot of peerInfo.snapshots) {
        const existing = snapshotPeers.get(snapshot.id);
        if (existing) {
          existing.peers.push(peerId);
        } else {
          snapshotPeers.set(snapshot.id, {
            snapshot,
            peers: [peerId],
          });
        }
      }
    }

    // Filter to snapshots with enough peers
    const candidates = Array.from(snapshotPeers.values()).filter(
      (s) => s.peers.length >= STATE_SYNC_CONFIG.MIN_SNAPSHOT_PEERS,
    );

    if (candidates.length === 0) {
      // Fallback: use any snapshot with at least one peer
      const any = Array.from(snapshotPeers.values())[0];
      return any || null;
    }

    // Select highest block number
    candidates.sort((a, b) =>
      a.snapshot.blockNumber > b.snapshot.blockNumber ? -1 : 1,
    );

    return candidates[0];
  }

  // ============ State Sync ============

  /**
   * Start state sync from snapshot
   */
  async startStateSync(): Promise<void> {
    if (this.syncing) {
      return;
    }

    this.syncing = true;
    this.aborted = false;
    this.status = {
      ...this.initialStatus(),
      syncing: true,
      currentPhase: "selecting",
    };

    this.emit("state_sync:started");

    try {
      // Phase 1: Select best snapshot
      const selection = this.selectBestSnapshot();
      if (!selection) {
        throw new Error("No snapshots available from peers");
      }

      const { snapshot, peers } = selection;
      this.status.snapshotId = snapshot.id;
      this.status.totalChunks = snapshot.chunkCount;

      this.emit("state_sync:snapshot_selected", { snapshot, peers });

      // Phase 2: Get full snapshot metadata
      this.status.currentPhase = "downloading";
      const fullSnapshot = await this.requestSnapshotMetadata(
        peers[0],
        snapshot.id,
      );

      // Phase 3: Download chunks in parallel
      await this.downloadChunks(fullSnapshot, peers);

      if (this.aborted) {
        throw new Error("State sync aborted");
      }

      // Phase 4: Verify all chunks
      this.status.currentPhase = "verifying";
      await this.verifyChunks(fullSnapshot);

      // Phase 5: Apply snapshot
      this.status.currentPhase = "applying";
      await this.applySnapshot(fullSnapshot);

      this.status.currentPhase = "complete";
      this.status.progress = 1.0;
      this.emit("state_sync:completed", { snapshot: fullSnapshot });
    } catch (error) {
      this.status.error = String(error);
      this.emit("state_sync:error", error);
      throw error;
    } finally {
      this.syncing = false;
      this.status.syncing = false;
    }
  }

  /**
   * Stop state sync
   */
  stopStateSync(): void {
    this.aborted = true;
    this.emit("state_sync:stopped");
  }

  // ============ Chunk Download ============

  /**
   * Download all chunks in parallel
   */
  private async downloadChunks(
    snapshot: StateSnapshot,
    peers: string[],
  ): Promise<void> {
    // Initialize chunk tasks
    this.pendingChunks = [];
    this.downloadingChunks.clear();
    this.completedChunks.clear();

    for (let i = 0; i < snapshot.chunkCount; i++) {
      this.pendingChunks.push({
        snapshotId: snapshot.id,
        chunkIndex: i,
        expectedHash: snapshot.chunkHashes[i],
        retryCount: 0,
      });
    }

    // Download with concurrency limit
    const concurrency = Math.min(
      STATE_SYNC_CONFIG.MAX_CONCURRENT_CHUNKS,
      peers.length,
    );

    const downloadPromises: Promise<void>[] = [];

    for (let i = 0; i < concurrency; i++) {
      downloadPromises.push(this.downloadWorker(peers));
    }

    await Promise.all(downloadPromises);

    // Check for failures
    if (this.pendingChunks.length > 0) {
      throw new Error(
        `Failed to download ${this.pendingChunks.length} chunks after retries`,
      );
    }
  }

  /**
   * Worker that downloads chunks from queue
   */
  private async downloadWorker(peers: string[]): Promise<void> {
    let peerIndex = 0;

    while (this.pendingChunks.length > 0 && !this.aborted) {
      const task = this.pendingChunks.shift();
      if (!task) break;

      // Round-robin peer selection
      const peer = peers[peerIndex % peers.length];
      peerIndex++;

      task.assignedPeer = peer;
      this.downloadingChunks.set(task.chunkIndex, task);

      try {
        const chunk = await this.downloadChunk(peer, task);

        // Verify hash immediately
        const actualHash = sha512_256Hex(chunk.data);
        if (actualHash !== task.expectedHash) {
          throw new Error(
            `Chunk ${task.chunkIndex} hash mismatch: expected ${task.expectedHash}, got ${actualHash}`,
          );
        }

        this.completedChunks.set(task.chunkIndex, chunk);
        this.status.downloadedChunks++;
        this.status.bytesDownloaded += BigInt(chunk.data.length);
        this.updateProgress();

        this.emit("state_sync:chunk_downloaded", {
          chunkIndex: task.chunkIndex,
          peer,
        });
      } catch (error) {
        task.retryCount++;

        if (task.retryCount < STATE_SYNC_CONFIG.CHUNK_RETRY_COUNT) {
          // Re-queue for retry
          this.pendingChunks.push(task);
          await this.delay(STATE_SYNC_CONFIG.CHUNK_RETRY_DELAY);
        } else {
          this.emit("state_sync:chunk_failed", {
            chunkIndex: task.chunkIndex,
            error,
          });
        }
      } finally {
        this.downloadingChunks.delete(task.chunkIndex);
      }
    }
  }

  /**
   * Download a single chunk from peer
   */
  private async downloadChunk(
    peerId: string,
    task: ChunkTask,
  ): Promise<SnapshotChunk> {
    if (!this.requestFn) {
      throw new Error("Request function not set");
    }

    const response = await this.requestFn<StateSyncResponse>(
      peerId,
      STATE_SYNC_PROTOCOL,
      {
        type: "chunk",
        snapshotId: task.snapshotId,
        chunkIndex: task.chunkIndex,
      },
      STATE_SYNC_CONFIG.CHUNK_TIMEOUT,
    );

    if (response.error) {
      throw new Error(response.error);
    }

    if (!response.chunk) {
      throw new Error(`No chunk in response for index ${task.chunkIndex}`);
    }

    return response.chunk;
  }

  // ============ Verification & Application ============

  /**
   * Verify all downloaded chunks
   */
  private async verifyChunks(snapshot: StateSnapshot): Promise<void> {
    for (let i = 0; i < snapshot.chunkCount; i++) {
      const chunk = this.completedChunks.get(i);
      if (!chunk) {
        throw new Error(`Missing chunk ${i}`);
      }

      const actualHash = sha512_256Hex(chunk.data);
      if (actualHash !== snapshot.chunkHashes[i]) {
        throw new Error(
          `Chunk ${i} verification failed: expected ${snapshot.chunkHashes[i]}, got ${actualHash}`,
        );
      }

      this.status.verifiedChunks++;
      this.updateProgress();
    }

    this.emit("state_sync:verified", { chunkCount: snapshot.chunkCount });
  }

  /**
   * Apply snapshot to local state
   */
  private async applySnapshot(snapshot: StateSnapshot): Promise<void> {
    // Import all chunks to snapshot manager
    for (let i = 0; i < snapshot.chunkCount; i++) {
      const chunk = this.completedChunks.get(i);
      if (!chunk) {
        throw new Error(`Missing chunk ${i} during apply`);
      }

      await this.snapshotManager.importChunk(chunk);
      this.status.appliedChunks++;
      this.updateProgress();
    }

    // Restore snapshot to state
    await this.snapshotManager.restoreSnapshot(snapshot.id);

    this.emit("state_sync:applied", { snapshot });

    // Clear completed chunks to free memory
    this.completedChunks.clear();
  }

  // ============ Request Helpers ============

  /**
   * Request snapshot metadata from peer
   */
  private async requestSnapshotMetadata(
    peerId: string,
    snapshotId: string,
  ): Promise<StateSnapshot> {
    if (!this.requestFn) {
      throw new Error("Request function not set");
    }

    const response = await this.requestFn<StateSyncResponse>(
      peerId,
      STATE_SYNC_PROTOCOL,
      { type: "snapshot_metadata", snapshotId },
      SYNC_CONFIG.REQUEST_TIMEOUT,
    );

    if (response.error) {
      throw new Error(response.error);
    }

    if (!response.snapshot) {
      throw new Error("No snapshot in response");
    }

    return response.snapshot;
  }

  /**
   * Request snapshot list from peer
   */
  async requestSnapshotList(peerId: string): Promise<SnapshotInfo[]> {
    if (!this.requestFn) {
      throw new Error("Request function not set");
    }

    const response = await this.requestFn<StateSyncResponse>(
      peerId,
      STATE_SYNC_PROTOCOL,
      { type: "list_snapshots" },
      SYNC_CONFIG.REQUEST_TIMEOUT,
    );

    if (response.error) {
      throw new Error(response.error);
    }

    return response.snapshots || [];
  }

  /**
   * Set the request function for peer communication
   */
  setRequestFunction(
    fn: <T>(
      peerId: string,
      protocol: string,
      request: StateSyncRequest,
      timeout?: number,
    ) => Promise<T>,
  ): void {
    this.requestFn = fn;
  }

  // ============ Helpers ============

  private initialStatus(): StateSyncStatus {
    return {
      syncing: false,
      totalChunks: 0,
      downloadedChunks: 0,
      verifiedChunks: 0,
      appliedChunks: 0,
      bytesDownloaded: 0n,
      currentPhase: "idle",
      progress: 0,
    };
  }

  private updateProgress(): void {
    const total = this.status.totalChunks * 3; // download + verify + apply
    const done =
      this.status.downloadedChunks +
      this.status.verifiedChunks +
      this.status.appliedChunks;
    this.status.progress = total > 0 ? done / total : 0;
    this.emit("state_sync:progress", this.status.progress);
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}

// ============ Protocol Handler ============

/**
 * Handles state sync protocol requests
 */
export class StateSyncProtocolHandler {
  constructor(private snapshotManager: SnapshotManager) {}

  /**
   * Handle incoming state sync stream
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
        console.warn(`[StateSync] Empty request from ${peerId}`);
        return;
      }

      const request: StateSyncRequest = JSON.parse(
        new TextDecoder().decode(requestData),
        networkReviver,
      );

      const response = await this.handleRequest(request);
      const responseData = new TextEncoder().encode(
        JSON.stringify(response, networkReplacer),
      );

      await pipe([responseData], lp.encode, streamAny.sink);
    } catch (error) {
      console.error(`[StateSync] Error handling stream from ${peerId}:`, error);
    }
  }

  /**
   * Handle a state sync request
   */
  async handleRequest(request: StateSyncRequest): Promise<StateSyncResponse> {
    switch (request.type) {
      case "list_snapshots": {
        const snapshots = await this.snapshotManager.listSnapshots();
        return {
          type: "snapshot_list",
          snapshots: snapshots.map((s) => ({
            id: s.id,
            blockNumber: s.blockNumber,
            blockHash: s.blockHash,
            stateRoot: s.stateRoot,
            chunkCount: s.chunkCount,
            compressedSize: s.compressedSize,
          })),
        };
      }

      case "snapshot_metadata": {
        if (!request.snapshotId) {
          return { type: "error", error: "snapshotId required" };
        }

        const snapshot = await this.snapshotManager.getSnapshot(
          request.snapshotId,
        );
        if (!snapshot) {
          return {
            type: "error",
            error: `Snapshot not found: ${request.snapshotId}`,
          };
        }

        return { type: "snapshot_metadata", snapshot };
      }

      case "chunk": {
        if (!request.snapshotId || request.chunkIndex === undefined) {
          return { type: "error", error: "snapshotId and chunkIndex required" };
        }

        // Stream chunks from manager
        let chunkIndex = 0;
        for await (const chunk of this.snapshotManager.streamChunks(
          request.snapshotId,
        )) {
          if (chunkIndex === request.chunkIndex) {
            return { type: "chunk", chunk };
          }
          chunkIndex++;
        }

        return {
          type: "error",
          error: `Chunk not found: ${request.chunkIndex}`,
        };
      }

      default:
        return {
          type: "error",
          error: `Unknown request type: ${request.type}`,
        };
    }
  }
}

// ============ Factory ============

/**
 * Create state sync manager with dependencies
 */
export function createStateSyncManager(
  snapshotManager: SnapshotManager,
): StateSyncManager {
  return new StateSyncManager(snapshotManager);
}
