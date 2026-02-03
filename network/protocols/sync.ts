/**
 * Sync Protocol
 *
 * Block and state synchronization protocol.
 */

import { EventEmitter } from 'events';

// ============ Types ============

export interface SyncStatus {
  currentBlock: bigint;
  highestBlock: bigint;
  startingBlock: bigint;
  syncing: boolean;
  progress: number;  // 0.0 to 1.0
}

export interface BlockRange {
  start: bigint;
  end: bigint;
}

export interface SyncPeer {
  peerId: string;
  headBlock: bigint;
  latency: number;
  score: number;
}

export interface SyncRequest {
  type: 'status' | 'headers' | 'bodies' | 'state';
  startBlock?: bigint;
  count?: number;
  stateRoot?: string;
}

export interface SyncResponse {
  type: 'status' | 'headers' | 'bodies' | 'state';
  data: unknown;
}

// ============ Protocol Constants ============

export const SYNC_PROTOCOL = '/lgm/sync/1.0.0';

export const SYNC_CONFIG = {
  // How many blocks to request at once
  BATCH_SIZE: 100,

  // Maximum concurrent requests
  MAX_CONCURRENT: 4,

  // Timeout for sync requests
  REQUEST_TIMEOUT: 30000,

  // Minimum peers before starting sync
  MIN_PEERS: 1,

  // How often to check for new blocks
  POLL_INTERVAL: 1000,
};

// ============ Sync Manager ============

/**
 * Manages block and state synchronization
 */
export class SyncManager extends EventEmitter {
  private status: SyncStatus;
  private peers: Map<string, SyncPeer> = new Map();
  private syncing: boolean = false;
  private aborted: boolean = false;

  constructor(
    private currentBlock: bigint = 0n,
    private getBlock: (num: bigint) => Promise<unknown>,
    private saveBlock: (block: unknown) => Promise<void>,
    private validateBlock: (block: unknown) => Promise<boolean>
  ) {
    super();
    this.status = {
      currentBlock,
      highestBlock: currentBlock,
      startingBlock: currentBlock,
      syncing: false,
      progress: 1.0,
    };
  }

  /**
   * Get current sync status
   */
  getStatus(): SyncStatus {
    return { ...this.status };
  }

  /**
   * Add a sync peer
   */
  addPeer(peer: SyncPeer): void {
    this.peers.set(peer.peerId, peer);
    this.updateHighestBlock();
    this.emit('peer:added', peer);

    // Start sync if we're behind
    if (!this.syncing && this.status.highestBlock > this.status.currentBlock) {
      this.startSync();
    }
  }

  /**
   * Remove a sync peer
   */
  removePeer(peerId: string): void {
    const peer = this.peers.get(peerId);
    if (peer) {
      this.peers.delete(peerId);
      this.updateHighestBlock();
      this.emit('peer:removed', peer);
    }
  }

  /**
   * Update peer's head block
   */
  updatePeerHead(peerId: string, headBlock: bigint): void {
    const peer = this.peers.get(peerId);
    if (peer) {
      peer.headBlock = headBlock;
      this.updateHighestBlock();
    }
  }

  /**
   * Start synchronization
   */
  async startSync(): Promise<void> {
    if (this.syncing) return;

    this.syncing = true;
    this.aborted = false;
    this.status.syncing = true;
    this.status.startingBlock = this.status.currentBlock;

    this.emit('sync:started', this.status);

    try {
      await this.syncLoop();
    } catch (error) {
      this.emit('sync:error', error);
    } finally {
      this.syncing = false;
      this.status.syncing = false;
      this.emit('sync:completed', this.status);
    }
  }

  /**
   * Stop synchronization
   */
  stopSync(): void {
    this.aborted = true;
    this.emit('sync:stopped');
  }

  /**
   * Handle incoming sync request
   */
  async handleRequest(request: SyncRequest): Promise<SyncResponse> {
    switch (request.type) {
      case 'status':
        return {
          type: 'status',
          data: this.status,
        };

      case 'headers':
      case 'bodies':
        // Fetch requested blocks
        const blocks: unknown[] = [];
        const start = request.startBlock || 0n;
        const count = BigInt(request.count || SYNC_CONFIG.BATCH_SIZE);

        for (let i = 0n; i < count; i++) {
          const block = await this.getBlock(start + i);
          if (block) {
            blocks.push(block);
          } else {
            break;
          }
        }

        return {
          type: request.type,
          data: blocks,
        };

      case 'state':
        // State sync not implemented yet
        return {
          type: 'state',
          data: null,
        };

      default:
        throw new Error(`Unknown request type: ${request.type}`);
    }
  }

  // ============ Private Methods ============

  private async syncLoop(): Promise<void> {
    while (!this.aborted && this.status.currentBlock < this.status.highestBlock) {
      // Select best peer
      const peer = this.selectBestPeer();
      if (!peer) {
        // No peers, wait
        await this.delay(SYNC_CONFIG.POLL_INTERVAL);
        continue;
      }

      // Request next batch of blocks
      const startBlock = this.status.currentBlock + 1n;
      const endBlock = this.min(
        startBlock + BigInt(SYNC_CONFIG.BATCH_SIZE),
        this.status.highestBlock + 1n
      );

      try {
        await this.syncRange(peer, startBlock, endBlock);
      } catch (error) {
        // Penalize peer on failure
        peer.score -= 10;
        this.emit('sync:peer_error', { peer, error });
      }

      this.updateProgress();
    }
  }

  private async syncRange(peer: SyncPeer, start: bigint, end: bigint): Promise<void> {
    this.emit('sync:range_start', { start, end, peer: peer.peerId });

    // In real implementation, this would request blocks from peer
    // For now, just simulate
    for (let i = start; i < end && !this.aborted; i++) {
      // Placeholder: would receive block from peer
      const block = { number: i }; // Simulated

      // Validate block
      const valid = await this.validateBlock(block);
      if (!valid) {
        throw new Error(`Invalid block at ${i}`);
      }

      // Save block
      await this.saveBlock(block);
      this.status.currentBlock = i;

      this.emit('sync:block', { number: i });
    }

    this.emit('sync:range_complete', { start, end });
  }

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
    this.emit('sync:progress', this.status.progress);
  }

  private min(a: bigint, b: bigint): bigint {
    return a < b ? a : b;
  }

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * Set the request function for fetching blocks from peers
   * This is injected by the network layer
   */
  setRequestFunction(
    requestFn: (peerId: string, start: bigint, count: number) => Promise<unknown[]>
  ): void {
    this.requestBlocks = requestFn;
  }

  private requestBlocks?: (peerId: string, start: bigint, count: number) => Promise<unknown[]>;

  /**
   * Sync blocks from a peer using the network request function
   */
  async syncRangeFromPeer(peer: SyncPeer, start: bigint, end: bigint): Promise<void> {
    if (!this.requestBlocks) {
      throw new Error('Request function not set - call setRequestFunction first');
    }

    this.emit('sync:range_start', { start, end, peer: peer.peerId });

    const count = Number(end - start);
    const blocks = await this.requestBlocks(peer.peerId, start, count);

    for (let i = 0; i < blocks.length && !this.aborted; i++) {
      const block = blocks[i];
      const blockNumber = start + BigInt(i);

      // Validate block
      const valid = await this.validateBlock(block);
      if (!valid) {
        throw new Error(`Invalid block at ${blockNumber}`);
      }

      // Save block
      await this.saveBlock(block);
      this.status.currentBlock = blockNumber;

      this.emit('sync:block', { number: blockNumber, block });
    }

    this.emit('sync:range_complete', { start, end });
  }
}

// ============ Sync Protocol Handler ============

import { pipe } from 'it-pipe';
import * as lp from 'it-length-prefixed';
import type { Stream } from '@libp2p/interface';

/**
 * Protocol handler for sync requests/responses
 * Bridges libp2p streams to SyncManager
 */
export class SyncProtocolHandler {
  constructor(private syncManager: SyncManager) {}

  /**
   * Handle incoming sync stream from a peer
   */
  async handleStream(stream: Stream, peerId: string): Promise<void> {
    try {
      // Read request from stream
      let requestData: Uint8Array | undefined;

      await pipe(
        stream.source,
        lp.decode,
        async function (source) {
          for await (const data of source) {
            requestData = data.subarray();
            break; // Only read first message
          }
        }
      );

      if (!requestData) {
        console.warn(`[Sync] Empty request from ${peerId}`);
        return;
      }

      // Parse request
      const request: SyncRequest = JSON.parse(
        new TextDecoder().decode(requestData)
      );

      // Process request
      const response = await this.syncManager.handleRequest(request);

      // Send response
      const responseData = new TextEncoder().encode(JSON.stringify(response));

      await pipe(
        [responseData],
        lp.encode,
        stream.sink
      );
    } catch (error) {
      console.error(`[Sync] Error handling stream from ${peerId}:`, error);
    }
  }

  /**
   * Request blocks from a peer via network node
   */
  async requestBlocksFromPeer(
    requestFn: <T>(peerId: string, protocol: string, request: unknown, timeout?: number) => Promise<T>,
    peerId: string,
    startBlock: bigint,
    count: number
  ): Promise<unknown[]> {
    const request: SyncRequest = {
      type: 'bodies',
      startBlock,
      count,
    };

    const response = await requestFn<SyncResponse>(
      peerId,
      SYNC_PROTOCOL,
      {
        type: 'sync_request',
        version: '1.0.0',
        timestamp: Date.now(),
        sender: '',
        payload: request,
      },
      SYNC_CONFIG.REQUEST_TIMEOUT
    );

    return (response.data as unknown[]) || [];
  }
}
