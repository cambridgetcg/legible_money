/**
 * Relay Node Role
 *
 * Network partition bridge for cross-region message propagation.
 * Relay nodes have high fan-out but low storage — they only store
 * block headers for routing decisions and use bloom filters for
 * message deduplication.
 *
 * Capabilities:
 * - Topics: BLOCKS, TRANSACTIONS, CONSENSUS (core propagation only)
 * - Storage: block headers only
 * - Sync: headers only
 * - DHT: server
 * - Protocols: relay, circuit relay
 */

import { EventEmitter } from "events";
import { ROLE_PROTOCOL_IDS } from "./types.js";

// ============ Constants ============

/** Bloom filter size (bits) for message deduplication */
const BLOOM_FILTER_SIZE = 65536; // 64K bits = 8KB

/** Number of hash functions for bloom filter */
const BLOOM_HASH_COUNT = 7;

/** How often to rotate the bloom filter (ms) */
const BLOOM_ROTATION_INTERVAL = 30_000; // 30 seconds

/** Bandwidth priority levels */
const BANDWIDTH_PRIORITIES = {
  CONSENSUS: 3, // Highest
  BLOCKS: 2,
  TRANSACTIONS: 1, // Lowest
} as const;

/** Maximum messages in the relay queue */
const MAX_RELAY_QUEUE = 5_000;

/** Latency measurement interval (ms) */
const LATENCY_MEASURE_INTERVAL = 15_000;

// ============ Types ============

export interface RelayConfig {
  /** Geographic region of this relay node */
  region?: string;

  /** Target regions to connect to */
  targetRegions?: string[];

  /** Maximum relay queue size */
  maxQueueSize: number;

  /** Bloom filter rotation interval (ms) */
  bloomRotationInterval: number;

  /** Latency measurement interval (ms) */
  latencyMeasureInterval: number;
}

export interface RelayStats {
  /** Total messages relayed */
  messagesRelayed: number;

  /** Messages deduplicated (not relayed because already seen) */
  messagesDeduplicated: number;

  /** Current relay queue size */
  queueSize: number;

  /** Number of connected peers by region */
  peersByRegion: Record<string, number>;

  /** Average latency to peers by region (ms) */
  latencyByRegion: Record<string, number>;

  /** Messages relayed by priority */
  relayedByPriority: Record<string, number>;

  /** Bloom filter false positive rate estimate */
  bloomFalsePositiveRate: number;

  /** Uptime in ms */
  uptime: number;
}

export interface RelayMessage {
  /** Message ID for deduplication */
  id: string;

  /** Topic/channel */
  topic: string;

  /** Priority level */
  priority: number;

  /** Message data */
  data: Uint8Array;

  /** Source peer ID */
  fromPeer: string;

  /** Timestamp received */
  receivedAt: number;
}

export interface PeerLatency {
  peerId: string;
  region?: string;
  latencyMs: number;
  measuredAt: number;
}

// ============ Bloom Filter ============

/**
 * Simple counting-less bloom filter for message deduplication.
 * Uses double hashing for multiple hash functions.
 */
class BloomFilter {
  private bits: Uint8Array;
  private size: number;
  private hashCount: number;

  constructor(
    size: number = BLOOM_FILTER_SIZE,
    hashCount: number = BLOOM_HASH_COUNT,
  ) {
    this.size = size;
    this.hashCount = hashCount;
    this.bits = new Uint8Array(Math.ceil(size / 8));
  }

  /**
   * Add an item to the filter.
   */
  add(item: string): void {
    const hashes = this.getHashes(item);
    for (const h of hashes) {
      const byteIdx = Math.floor(h / 8);
      const bitIdx = h % 8;
      this.bits[byteIdx] |= 1 << bitIdx;
    }
  }

  /**
   * Check if an item might be in the filter.
   * Returns false = definitely not seen, true = probably seen.
   */
  mightContain(item: string): boolean {
    const hashes = this.getHashes(item);
    for (const h of hashes) {
      const byteIdx = Math.floor(h / 8);
      const bitIdx = h % 8;
      if ((this.bits[byteIdx] & (1 << bitIdx)) === 0) {
        return false;
      }
    }
    return true;
  }

  /**
   * Estimate the false positive rate based on fill ratio.
   */
  estimateFalsePositiveRate(): number {
    let setBits = 0;
    for (const byte of this.bits) {
      for (let i = 0; i < 8; i++) {
        if (byte & (1 << i)) setBits++;
      }
    }
    const fillRatio = setBits / this.size;
    return Math.pow(fillRatio, this.hashCount);
  }

  /**
   * Reset the filter.
   */
  clear(): void {
    this.bits.fill(0);
  }

  /**
   * Generate hash indices using double hashing.
   * h(i) = (h1 + i * h2) mod size
   */
  private getHashes(item: string): number[] {
    const h1 = this.fnv1a(item);
    const h2 = this.djb2(item);
    const hashes: number[] = [];

    for (let i = 0; i < this.hashCount; i++) {
      hashes.push(Math.abs((h1 + i * h2) % this.size));
    }

    return hashes;
  }

  /** FNV-1a hash */
  private fnv1a(str: string): number {
    let hash = 0x811c9dc5;
    for (let i = 0; i < str.length; i++) {
      hash ^= str.charCodeAt(i);
      hash = (hash * 0x01000193) >>> 0;
    }
    return hash;
  }

  /** DJB2 hash */
  private djb2(str: string): number {
    let hash = 5381;
    for (let i = 0; i < str.length; i++) {
      hash = ((hash << 5) + hash + str.charCodeAt(i)) >>> 0;
    }
    return hash;
  }
}

// ============ Priority Queue ============

/**
 * Priority queue for relay messages.
 * Higher priority messages are dequeued first.
 */
class PriorityRelayQueue {
  private queues: Map<number, RelayMessage[]> = new Map();
  private totalSize: number = 0;
  private maxSize: number;

  constructor(maxSize: number) {
    this.maxSize = maxSize;
  }

  enqueue(message: RelayMessage): boolean {
    if (this.totalSize >= this.maxSize) {
      // Drop lowest priority message
      if (!this.dropLowest(message.priority)) {
        return false;
      }
    }

    const queue = this.queues.get(message.priority) ?? [];
    queue.push(message);
    this.queues.set(message.priority, queue);
    this.totalSize++;
    return true;
  }

  dequeue(): RelayMessage | null {
    // Get highest priority non-empty queue
    const priorities = Array.from(this.queues.keys()).sort((a, b) => b - a);

    for (const priority of priorities) {
      const queue = this.queues.get(priority);
      if (queue && queue.length > 0) {
        this.totalSize--;
        return queue.shift()!;
      }
    }

    return null;
  }

  get size(): number {
    return this.totalSize;
  }

  private dropLowest(minPriority: number): boolean {
    const priorities = Array.from(this.queues.keys()).sort((a, b) => a - b);

    for (const priority of priorities) {
      if (priority >= minPriority) break;
      const queue = this.queues.get(priority);
      if (queue && queue.length > 0) {
        queue.shift();
        this.totalSize--;
        return true;
      }
    }

    return false;
  }
}

// ============ Relay Node Service ============

/**
 * Relay node service for cross-region message propagation.
 *
 * Design:
 * - Uses bloom filters for O(1) message deduplication
 * - Priority queue ensures consensus messages are relayed first
 * - Periodic latency measurements guide routing decisions
 * - Two bloom filters rotated on a timer to prevent staleness
 */
export class RelayNodeService extends EventEmitter {
  private config: RelayConfig;
  private currentBloom: BloomFilter;
  private previousBloom: BloomFilter;
  private queue: PriorityRelayQueue;
  private peerLatencies: Map<string, PeerLatency> = new Map();
  private bloomTimer: ReturnType<typeof setInterval> | null = null;
  private latencyTimer: ReturnType<typeof setInterval> | null = null;
  private startedAt: number = 0;

  // Stats
  private messagesRelayed: number = 0;
  private messagesDeduplicated: number = 0;
  private relayedByPriority: Record<string, number> = {
    consensus: 0,
    blocks: 0,
    transactions: 0,
  };

  constructor(config?: Partial<RelayConfig>) {
    super();
    this.config = {
      maxQueueSize: MAX_RELAY_QUEUE,
      bloomRotationInterval: BLOOM_ROTATION_INTERVAL,
      latencyMeasureInterval: LATENCY_MEASURE_INTERVAL,
      ...config,
    };
    this.currentBloom = new BloomFilter();
    this.previousBloom = new BloomFilter();
    this.queue = new PriorityRelayQueue(this.config.maxQueueSize);
  }

  /**
   * Start the relay service.
   */
  start(): void {
    this.startedAt = Date.now();

    // Rotate bloom filter periodically
    this.bloomTimer = setInterval(
      () => this.rotateBloom(),
      this.config.bloomRotationInterval,
    );

    // Measure latencies periodically
    this.latencyTimer = setInterval(
      () => this.emit("measure_latency"),
      this.config.latencyMeasureInterval,
    );

    this.emit("started");
  }

  /**
   * Stop the relay service.
   */
  stop(): void {
    if (this.bloomTimer) {
      clearInterval(this.bloomTimer);
      this.bloomTimer = null;
    }
    if (this.latencyTimer) {
      clearInterval(this.latencyTimer);
      this.latencyTimer = null;
    }
    this.emit("stopped");
  }

  /**
   * Process an inbound message for relay.
   * Returns true if the message should be relayed (not a duplicate).
   */
  processMessage(
    messageId: string,
    topic: string,
    data: Uint8Array,
    fromPeer: string,
  ): boolean {
    // Deduplication check across both bloom filters
    if (
      this.currentBloom.mightContain(messageId) ||
      this.previousBloom.mightContain(messageId)
    ) {
      this.messagesDeduplicated++;
      return false;
    }

    // Mark as seen
    this.currentBloom.add(messageId);

    // Determine priority
    const priority = this.getTopicPriority(topic);

    // Enqueue for relay
    const queued = this.queue.enqueue({
      id: messageId,
      topic,
      priority,
      data,
      fromPeer,
      receivedAt: Date.now(),
    });

    if (queued) {
      this.messagesRelayed++;
      this.trackPriorityStats(topic);
    }

    return queued;
  }

  /**
   * Get the next batch of messages to relay.
   */
  drainQueue(maxBatch: number = 100): RelayMessage[] {
    const batch: RelayMessage[] = [];
    for (let i = 0; i < maxBatch; i++) {
      const msg = this.queue.dequeue();
      if (!msg) break;
      batch.push(msg);
    }
    return batch;
  }

  /**
   * Record latency measurement for a peer.
   */
  recordLatency(peerId: string, latencyMs: number, region?: string): void {
    this.peerLatencies.set(peerId, {
      peerId,
      region,
      latencyMs,
      measuredAt: Date.now(),
    });
  }

  /**
   * Get relay statistics.
   */
  getStats(): RelayStats {
    // Compute peers by region
    const peersByRegion: Record<string, number> = {};
    const latencyByRegion: Record<string, number[]> = {};

    for (const latency of this.peerLatencies.values()) {
      const region = latency.region ?? "unknown";
      peersByRegion[region] = (peersByRegion[region] ?? 0) + 1;
      latencyByRegion[region] = latencyByRegion[region] ?? [];
      latencyByRegion[region].push(latency.latencyMs);
    }

    const avgLatency: Record<string, number> = {};
    for (const [region, latencies] of Object.entries(latencyByRegion)) {
      avgLatency[region] =
        Math.round(
          (latencies.reduce((a, b) => a + b, 0) / latencies.length) * 10,
        ) / 10;
    }

    return {
      messagesRelayed: this.messagesRelayed,
      messagesDeduplicated: this.messagesDeduplicated,
      queueSize: this.queue.size,
      peersByRegion,
      latencyByRegion: avgLatency,
      relayedByPriority: { ...this.relayedByPriority },
      bloomFalsePositiveRate: this.currentBloom.estimateFalsePositiveRate(),
      uptime: this.startedAt > 0 ? Date.now() - this.startedAt : 0,
    };
  }

  /**
   * Get the protocol ID for relay.
   */
  static get protocolId(): string {
    return ROLE_PROTOCOL_IDS.RELAY;
  }

  // ============ Internal ============

  /**
   * Rotate bloom filters: current becomes previous, new filter created.
   * This ensures old message IDs are eventually forgotten.
   */
  private rotateBloom(): void {
    this.previousBloom = this.currentBloom;
    this.currentBloom = new BloomFilter();
  }

  private getTopicPriority(topic: string): number {
    if (topic.includes("consensus")) return BANDWIDTH_PRIORITIES.CONSENSUS;
    if (topic.includes("blocks")) return BANDWIDTH_PRIORITIES.BLOCKS;
    return BANDWIDTH_PRIORITIES.TRANSACTIONS;
  }

  private trackPriorityStats(topic: string): void {
    if (topic.includes("consensus")) {
      this.relayedByPriority.consensus++;
    } else if (topic.includes("blocks")) {
      this.relayedByPriority.blocks++;
    } else {
      this.relayedByPriority.transactions++;
    }
  }
}
