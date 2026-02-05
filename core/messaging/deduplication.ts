/**
 * Message Deduplication
 *
 * Prevents duplicate message processing using:
 * - Content-addressed message IDs
 * - Content hash tracking (detects resends with new IDs)
 * - Time-windowed tracking with automatic cleanup
 * - Bloom filter for memory-efficient approximate checks
 *
 * Ensures idempotent message processing even with network retries.
 */

// ============ Configuration ============

/**
 * Deduplication configuration
 */
export interface DeduplicationConfig {
  /** Window size in blocks */
  windowBlocks: number;

  /** Window size in milliseconds (alternative to blocks) */
  windowMs?: number;

  /** Maximum entries before forced cleanup */
  maxEntries: number;

  /** Use bloom filter for approximate checks */
  useBloomFilter: boolean;

  /** Bloom filter expected elements */
  bloomFilterExpected?: number;

  /** Bloom filter false positive rate */
  bloomFilterFpRate?: number;
}

/**
 * Default deduplication configuration
 */
export const DEFAULT_DEDUP_CONFIG: DeduplicationConfig = {
  windowBlocks: 100,
  windowMs: 300000, // 5 minutes
  maxEntries: 10000,
  useBloomFilter: true,
  bloomFilterExpected: 10000,
  bloomFilterFpRate: 0.001,
};

// ============ Bloom Filter ============

/**
 * Simple bloom filter implementation
 *
 * Memory-efficient probabilistic set membership check.
 * May return false positives, never false negatives.
 */
export class BloomFilter {
  private bits: Uint8Array;
  private numHashes: number;
  private size: number;

  /**
   * Create a bloom filter
   *
   * @param expectedElements - Expected number of elements
   * @param fpRate - Desired false positive rate (default 0.001 = 0.1%)
   */
  constructor(expectedElements: number, fpRate: number = 0.001) {
    // Calculate optimal size and hash count
    // m = -n * ln(p) / (ln(2)^2)
    // k = m/n * ln(2)
    const ln2 = Math.LN2;
    const ln2Squared = ln2 * ln2;

    this.size = Math.ceil((-expectedElements * Math.log(fpRate)) / ln2Squared);
    this.numHashes = Math.ceil((this.size / expectedElements) * ln2);

    // Round up to nearest byte
    const numBytes = Math.ceil(this.size / 8);
    this.bits = new Uint8Array(numBytes);
  }

  /**
   * Add an element to the filter
   */
  add(element: string): void {
    const hashes = this.getHashes(element);
    for (const hash of hashes) {
      const index = hash % this.size;
      const byteIndex = Math.floor(index / 8);
      const bitIndex = index % 8;
      this.bits[byteIndex] |= 1 << bitIndex;
    }
  }

  /**
   * Check if element might be in the filter
   *
   * @returns true if element might be present (could be false positive)
   *          false if element is definitely not present
   */
  mightContain(element: string): boolean {
    const hashes = this.getHashes(element);
    for (const hash of hashes) {
      const index = hash % this.size;
      const byteIndex = Math.floor(index / 8);
      const bitIndex = index % 8;
      if ((this.bits[byteIndex] & (1 << bitIndex)) === 0) {
        return false;
      }
    }
    return true;
  }

  /**
   * Clear the filter
   */
  clear(): void {
    this.bits.fill(0);
  }

  /**
   * Get estimated fill ratio
   */
  getFillRatio(): number {
    let setBits = 0;
    for (const byte of this.bits) {
      setBits += this.popCount(byte);
    }
    return setBits / this.size;
  }

  private getHashes(element: string): number[] {
    // Use double hashing: h(i) = h1 + i * h2
    const h1 = this.hash1(element);
    const h2 = this.hash2(element);

    const hashes: number[] = [];
    for (let i = 0; i < this.numHashes; i++) {
      hashes.push(Math.abs((h1 + i * h2) | 0));
    }
    return hashes;
  }

  private hash1(str: string): number {
    // djb2 hash
    let hash = 5381;
    for (let i = 0; i < str.length; i++) {
      hash = (hash << 5) + hash + str.charCodeAt(i);
    }
    return hash >>> 0;
  }

  private hash2(str: string): number {
    // sdbm hash
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      hash = str.charCodeAt(i) + (hash << 6) + (hash << 16) - hash;
    }
    return hash >>> 0;
  }

  private popCount(n: number): number {
    // Count set bits in a byte
    n = n - ((n >> 1) & 0x55);
    n = (n & 0x33) + ((n >> 2) & 0x33);
    return (n + (n >> 4)) & 0x0f;
  }
}

// ============ Seen Entry ============

/**
 * Record of a seen message
 */
interface SeenEntry {
  /** Message ID */
  messageId: string;

  /** Content hash */
  contentHash: string;

  /** Block height when seen */
  blockHeight: bigint;

  /** Timestamp when seen */
  timestamp: number;
}

// ============ Deduplicator ============

/**
 * Deduplication result
 */
export interface DeduplicationResult {
  /** Whether this is a duplicate */
  isDuplicate: boolean;

  /** How duplicate was detected */
  reason?: "exact_id" | "content_hash" | "bloom_filter";

  /** Original message ID if duplicate by content */
  originalMessageId?: string;
}

/**
 * MessageDeduplicator - Prevents duplicate message processing
 *
 * Usage:
 * ```typescript
 * const dedup = new MessageDeduplicator(config);
 *
 * // Check and mark as seen
 * const result = dedup.checkAndMark(messageId, contentHash, blockHeight);
 * if (result.isDuplicate) {
 *   console.log('Duplicate detected:', result.reason);
 *   return;
 * }
 *
 * // Process message...
 * ```
 */
export class MessageDeduplicator {
  private config: DeduplicationConfig;

  /** Exact message IDs seen */
  private seenIds: Map<string, SeenEntry> = new Map();

  /** Content hashes seen (for detecting resends) */
  private seenHashes: Map<string, string> = new Map(); // hash → messageId

  /** Bloom filter for fast approximate checks */
  private bloomFilter?: BloomFilter;

  /** Current block height (for window calculation) */
  private currentBlock: bigint = 0n;

  constructor(config: DeduplicationConfig = DEFAULT_DEDUP_CONFIG) {
    this.config = config;

    if (config.useBloomFilter) {
      this.bloomFilter = new BloomFilter(
        config.bloomFilterExpected || 10000,
        config.bloomFilterFpRate || 0.001,
      );
    }
  }

  /**
   * Check if a message is a duplicate
   *
   * @param messageId - Message ID to check
   * @param contentHash - Content hash to check
   * @returns Deduplication result
   */
  check(messageId: string, contentHash: string): DeduplicationResult {
    // Fast bloom filter check first
    if (this.bloomFilter) {
      if (
        !this.bloomFilter.mightContain(messageId) &&
        !this.bloomFilter.mightContain(contentHash)
      ) {
        // Definitely not seen
        return { isDuplicate: false };
      }
    }

    // Exact ID check
    if (this.seenIds.has(messageId)) {
      return {
        isDuplicate: true,
        reason: "exact_id",
        originalMessageId: messageId,
      };
    }

    // Content hash check (detects resends with different ID)
    const originalId = this.seenHashes.get(contentHash);
    if (originalId) {
      return {
        isDuplicate: true,
        reason: "content_hash",
        originalMessageId: originalId,
      };
    }

    return { isDuplicate: false };
  }

  /**
   * Mark a message as seen
   *
   * @param messageId - Message ID
   * @param contentHash - Content hash
   * @param blockHeight - Current block height
   */
  markSeen(messageId: string, contentHash: string, blockHeight: bigint): void {
    const entry: SeenEntry = {
      messageId,
      contentHash,
      blockHeight,
      timestamp: Date.now(),
    };

    this.seenIds.set(messageId, entry);
    this.seenHashes.set(contentHash, messageId);

    if (this.bloomFilter) {
      this.bloomFilter.add(messageId);
      this.bloomFilter.add(contentHash);
    }

    // Update current block
    if (blockHeight > this.currentBlock) {
      this.currentBlock = blockHeight;
    }

    // Check if cleanup needed
    if (this.seenIds.size > this.config.maxEntries) {
      this.cleanup();
    }
  }

  /**
   * Check and mark in one operation
   *
   * @param messageId - Message ID
   * @param contentHash - Content hash
   * @param blockHeight - Current block height
   * @returns Deduplication result
   */
  checkAndMark(
    messageId: string,
    contentHash: string,
    blockHeight: bigint,
  ): DeduplicationResult {
    const result = this.check(messageId, contentHash);

    if (!result.isDuplicate) {
      this.markSeen(messageId, contentHash, blockHeight);
    }

    return result;
  }

  /**
   * Update current block height (for window calculation)
   */
  updateBlockHeight(blockHeight: bigint): void {
    this.currentBlock = blockHeight;
  }

  /**
   * Clean up entries outside the window
   */
  cleanup(): void {
    const now = Date.now();
    const cutoffBlock = this.currentBlock - BigInt(this.config.windowBlocks);
    const cutoffTime = this.config.windowMs ? now - this.config.windowMs : 0;

    const toRemove: string[] = [];

    for (const [messageId, entry] of this.seenIds) {
      const isOldByBlock = entry.blockHeight < cutoffBlock;
      const isOldByTime = cutoffTime > 0 && entry.timestamp < cutoffTime;

      if (isOldByBlock || isOldByTime) {
        toRemove.push(messageId);
      }
    }

    for (const messageId of toRemove) {
      const entry = this.seenIds.get(messageId);
      if (entry) {
        this.seenIds.delete(messageId);
        this.seenHashes.delete(entry.contentHash);
      }
    }
  }

  /**
   * Force cleanup to reduce memory
   */
  forceCleanup(): void {
    // If still over limit after normal cleanup, remove oldest entries
    if (this.seenIds.size > this.config.maxEntries) {
      const entries = Array.from(this.seenIds.entries()).sort(
        (a, b) => a[1].timestamp - b[1].timestamp,
      );

      const toRemove = entries.slice(
        0,
        entries.length - this.config.maxEntries,
      );
      for (const [messageId, entry] of toRemove) {
        this.seenIds.delete(messageId);
        this.seenHashes.delete(entry.contentHash);
      }
    }
  }

  /**
   * Get statistics
   */
  getStats(): {
    seenCount: number;
    bloomFilterFillRatio?: number;
    currentBlock: bigint;
  } {
    return {
      seenCount: this.seenIds.size,
      bloomFilterFillRatio: this.bloomFilter?.getFillRatio(),
      currentBlock: this.currentBlock,
    };
  }

  /**
   * Clear all entries
   */
  clear(): void {
    this.seenIds.clear();
    this.seenHashes.clear();
    this.bloomFilter?.clear();
    this.currentBlock = 0n;
  }

  /**
   * Check if message is still within the valid window
   *
   * @param messageId - Message ID to check
   * @returns True if message is still tracked
   */
  isTracked(messageId: string): boolean {
    return this.seenIds.has(messageId);
  }

  /**
   * Get the original message ID for a content hash
   *
   * @param contentHash - Content hash to look up
   * @returns Original message ID or undefined
   */
  getOriginalId(contentHash: string): string | undefined {
    return this.seenHashes.get(contentHash);
  }
}

// ============ Batch Deduplication ============

/**
 * Deduplicate a batch of messages
 *
 * @param messages - Messages to deduplicate
 * @param dedup - Deduplicator instance
 * @param blockHeight - Current block height
 * @returns Unique messages (duplicates filtered out)
 */
export function deduplicateBatch<
  T extends { messageId: string; contentHash: string },
>(messages: T[], dedup: MessageDeduplicator, blockHeight: bigint): T[] {
  const unique: T[] = [];

  for (const msg of messages) {
    const result = dedup.checkAndMark(
      msg.messageId,
      msg.contentHash,
      blockHeight,
    );
    if (!result.isDuplicate) {
      unique.push(msg);
    }
  }

  return unique;
}

/**
 * Deduplicate messages within an array (no external state)
 *
 * @param messages - Messages to deduplicate
 * @returns Unique messages
 */
export function deduplicateArray<
  T extends { messageId: string; contentHash: string },
>(messages: T[]): T[] {
  const seenIds = new Set<string>();
  const seenHashes = new Set<string>();
  const unique: T[] = [];

  for (const msg of messages) {
    if (seenIds.has(msg.messageId) || seenHashes.has(msg.contentHash)) {
      continue;
    }

    seenIds.add(msg.messageId);
    seenHashes.add(msg.contentHash);
    unique.push(msg);
  }

  return unique;
}
