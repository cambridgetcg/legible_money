/**
 * Causal Message Ordering
 *
 * Ensures messages are processed in causal order using:
 * - Vector clocks for tracking causality
 * - Dependency tracking via referencedMessages
 * - Message buffering for out-of-order arrivals
 *
 * Causal ordering guarantees:
 * - If message A causally precedes message B, A is delivered before B
 * - Messages with no causal relationship can be delivered in any order
 */

import type { TruthMessage } from "./types.js";

// ============ Vector Clock ============

/**
 * Vector clock for tracking causality
 *
 * Each entry maps a sender ID to the latest sequence number
 * we've seen from that sender.
 */
export type VectorClock = Map<string, number>;

/**
 * Create an empty vector clock
 */
export function createVectorClock(): VectorClock {
  return new Map();
}

/**
 * Clone a vector clock
 */
export function cloneVectorClock(clock: VectorClock): VectorClock {
  return new Map(clock);
}

/**
 * Increment the local entry in a vector clock
 *
 * @param clock - Vector clock to increment
 * @param localId - Local sender ID
 * @returns New clock with incremented local entry
 */
export function incrementClock(
  clock: VectorClock,
  localId: string,
): VectorClock {
  const newClock = cloneVectorClock(clock);
  const current = newClock.get(localId) || 0;
  newClock.set(localId, current + 1);
  return newClock;
}

/**
 * Merge two vector clocks (take max of each entry)
 *
 * @param clock1 - First clock
 * @param clock2 - Second clock
 * @returns Merged clock
 */
export function mergeClock(
  clock1: VectorClock,
  clock2: VectorClock,
): VectorClock {
  const merged = cloneVectorClock(clock1);

  for (const [id, time] of clock2) {
    const existing = merged.get(id) || 0;
    merged.set(id, Math.max(existing, time));
  }

  return merged;
}

/**
 * Compare two vector clocks
 *
 * @returns -1 if clock1 < clock2, 1 if clock1 > clock2, 0 if concurrent
 */
export function compareClock(
  clock1: VectorClock,
  clock2: VectorClock,
): -1 | 0 | 1 {
  let less = false;
  let greater = false;

  // Get all unique IDs
  const allIds = new Set([...clock1.keys(), ...clock2.keys()]);

  for (const id of allIds) {
    const t1 = clock1.get(id) || 0;
    const t2 = clock2.get(id) || 0;

    if (t1 < t2) less = true;
    if (t1 > t2) greater = true;
  }

  if (less && !greater) return -1; // clock1 happened before clock2
  if (greater && !less) return 1; // clock1 happened after clock2
  return 0; // concurrent
}

/**
 * Check if clock1 happened before clock2
 */
export function happenedBefore(
  clock1: VectorClock,
  clock2: VectorClock,
): boolean {
  return compareClock(clock1, clock2) === -1;
}

/**
 * Serialize vector clock for transmission
 */
export function serializeClock(clock: VectorClock): Record<string, number> {
  return Object.fromEntries(clock);
}

/**
 * Deserialize vector clock from transmission
 */
export function deserializeClock(data: Record<string, number>): VectorClock {
  return new Map(Object.entries(data));
}

// ============ Causal Message ============

/**
 * Message with causal ordering metadata
 */
export interface CausalMessage extends TruthMessage {
  /** Vector clock at time of sending */
  vectorClock: Record<string, number>;

  /** Sequence number from sender */
  senderSequence: number;
}

/**
 * Check if a message has causal metadata
 */
export function isCausalMessage(msg: TruthMessage): msg is CausalMessage {
  return "vectorClock" in msg && "senderSequence" in msg;
}

// ============ Causal Order Manager ============

/**
 * Buffered message waiting for dependencies
 */
interface BufferedMessage {
  message: CausalMessage;
  receivedAt: number;
  missingDeps: Set<string>;
}

/**
 * Delivery callback type
 */
export type DeliveryCallback = (message: CausalMessage) => void | Promise<void>;

/**
 * CausalOrderManager - Ensures messages are delivered in causal order
 *
 * Usage:
 * ```typescript
 * const manager = new CausalOrderManager('my-node-id');
 *
 * // Prepare outgoing messages
 * const causalMsg = manager.prepareOutgoing(message);
 *
 * // Process incoming messages
 * manager.setDeliveryCallback(async (msg) => {
 *   console.log('Delivered:', msg.messageId);
 * });
 * await manager.processIncoming(receivedMessage);
 * ```
 */
export class CausalOrderManager {
  private localId: string;
  private localClock: VectorClock;
  private localSequence: number = 0;

  /** Messages waiting for dependencies */
  private buffer: Map<string, BufferedMessage> = new Map();

  /** Delivered message IDs (for dependency resolution) */
  private delivered: Set<string> = new Set();

  /** Map from dependency → messages waiting for it */
  private waitingFor: Map<string, Set<string>> = new Map();

  /** Delivery callback */
  private deliveryCallback?: DeliveryCallback;

  /** Maximum buffer size */
  private maxBufferSize: number;

  /** Maximum time to wait for dependencies (ms) */
  private maxWaitTimeMs: number;

  constructor(
    localId: string,
    options: {
      maxBufferSize?: number;
      maxWaitTimeMs?: number;
    } = {},
  ) {
    this.localId = localId;
    this.localClock = createVectorClock();
    this.localClock.set(localId, 0);
    this.maxBufferSize = options.maxBufferSize || 1000;
    this.maxWaitTimeMs = options.maxWaitTimeMs || 60000;
  }

  /**
   * Set the delivery callback
   */
  setDeliveryCallback(callback: DeliveryCallback): void {
    this.deliveryCallback = callback;
  }

  /**
   * Prepare an outgoing message with causal metadata
   *
   * @param message - Message to send
   * @returns Message with vector clock and sequence
   */
  prepareOutgoing(message: TruthMessage): CausalMessage {
    // Increment local clock
    this.localSequence++;
    this.localClock = incrementClock(this.localClock, this.localId);

    return {
      ...message,
      vectorClock: serializeClock(this.localClock),
      senderSequence: this.localSequence,
    };
  }

  /**
   * Process an incoming message
   *
   * Delivers immediately if dependencies are satisfied,
   * otherwise buffers until dependencies arrive.
   *
   * @param message - Incoming causal message
   */
  async processIncoming(message: CausalMessage): Promise<void> {
    // Check if already delivered
    if (this.delivered.has(message.messageId)) {
      return; // Duplicate
    }

    // Check dependencies (referencedMessages)
    const missingDeps = this.findMissingDependencies(message);

    if (missingDeps.size === 0) {
      // All dependencies satisfied, deliver immediately
      await this.deliver(message);
    } else {
      // Buffer until dependencies arrive
      this.bufferMessage(message, missingDeps);
    }
  }

  /**
   * Check if a message can be delivered (all dependencies satisfied)
   */
  canDeliver(message: CausalMessage): boolean {
    return this.findMissingDependencies(message).size === 0;
  }

  /**
   * Get the current local vector clock
   */
  getLocalClock(): VectorClock {
    return cloneVectorClock(this.localClock);
  }

  /**
   * Get buffered message count
   */
  getBufferSize(): number {
    return this.buffer.size;
  }

  /**
   * Get messages waiting for a specific dependency
   */
  getWaitingFor(messageId: string): string[] {
    const waiting = this.waitingFor.get(messageId);
    return waiting ? Array.from(waiting) : [];
  }

  /**
   * Force delivery of buffered messages that have waited too long
   *
   * Called periodically to prevent indefinite blocking.
   */
  flushStaleMessages(): number {
    const now = Date.now();
    const cutoff = now - this.maxWaitTimeMs;
    let flushed = 0;

    for (const [messageId, buffered] of this.buffer) {
      if (buffered.receivedAt < cutoff) {
        // Force delivery despite missing dependencies
        this.forceDeliver(messageId);
        flushed++;
      }
    }

    return flushed;
  }

  /**
   * Clear all state
   */
  clear(): void {
    this.buffer.clear();
    this.delivered.clear();
    this.waitingFor.clear();
    this.localClock = createVectorClock();
    this.localClock.set(this.localId, 0);
    this.localSequence = 0;
  }

  // ============ Internal Methods ============

  private findMissingDependencies(message: CausalMessage): Set<string> {
    const missing = new Set<string>();

    // Check referencedMessages
    for (const refId of message.referencedMessages) {
      if (!this.delivered.has(refId)) {
        missing.add(refId);
      }
    }

    // Check causedBy
    if (message.causedBy && !this.delivered.has(message.causedBy)) {
      missing.add(message.causedBy);
    }

    return missing;
  }

  private bufferMessage(
    message: CausalMessage,
    missingDeps: Set<string>,
  ): void {
    // Check buffer size limit
    if (this.buffer.size >= this.maxBufferSize) {
      // Drop oldest message to make room
      const oldest = this.findOldestBuffered();
      if (oldest) {
        this.removeFromBuffer(oldest);
      }
    }

    // Add to buffer
    this.buffer.set(message.messageId, {
      message,
      receivedAt: Date.now(),
      missingDeps,
    });

    // Track what we're waiting for
    for (const dep of missingDeps) {
      let waiting = this.waitingFor.get(dep);
      if (!waiting) {
        waiting = new Set();
        this.waitingFor.set(dep, waiting);
      }
      waiting.add(message.messageId);
    }
  }

  private async deliver(message: CausalMessage): Promise<void> {
    // Mark as delivered
    this.delivered.add(message.messageId);

    // Update local clock (merge with sender's clock)
    const senderClock = deserializeClock(message.vectorClock);
    this.localClock = mergeClock(this.localClock, senderClock);

    // Call delivery callback
    if (this.deliveryCallback) {
      await this.deliveryCallback(message);
    }

    // Check if any buffered messages can now be delivered
    await this.checkBufferedMessages(message.messageId);
  }

  private async checkBufferedMessages(deliveredId: string): Promise<void> {
    // Get messages waiting for this one
    const waiting = this.waitingFor.get(deliveredId);
    if (!waiting) return;

    // Remove from waiting map
    this.waitingFor.delete(deliveredId);

    // Check each waiting message
    for (const waitingId of waiting) {
      const buffered = this.buffer.get(waitingId);
      if (!buffered) continue;

      // Remove the satisfied dependency
      buffered.missingDeps.delete(deliveredId);

      // If no more missing dependencies, deliver
      if (buffered.missingDeps.size === 0) {
        this.buffer.delete(waitingId);
        await this.deliver(buffered.message);
      }
    }
  }

  private forceDeliver(messageId: string): void {
    const buffered = this.buffer.get(messageId);
    if (!buffered) return;

    // Remove from buffer
    this.removeFromBuffer(messageId);

    // Deliver despite missing dependencies
    this.deliver(buffered.message).catch(() => {
      // Ignore errors from forced delivery
    });
  }

  private removeFromBuffer(messageId: string): void {
    const buffered = this.buffer.get(messageId);
    if (!buffered) return;

    // Remove from waiting maps
    for (const dep of buffered.missingDeps) {
      const waiting = this.waitingFor.get(dep);
      if (waiting) {
        waiting.delete(messageId);
        if (waiting.size === 0) {
          this.waitingFor.delete(dep);
        }
      }
    }

    this.buffer.delete(messageId);
  }

  private findOldestBuffered(): string | undefined {
    let oldest: string | undefined;
    let oldestTime = Infinity;

    for (const [id, buffered] of this.buffer) {
      if (buffered.receivedAt < oldestTime) {
        oldestTime = buffered.receivedAt;
        oldest = id;
      }
    }

    return oldest;
  }
}

// ============ Dependency Graph ============

/**
 * Build a dependency graph from messages
 *
 * @param messages - Messages to analyze
 * @returns Map from message ID to dependencies
 */
export function buildDependencyGraph(
  messages: TruthMessage[],
): Map<string, Set<string>> {
  const graph = new Map<string, Set<string>>();

  for (const msg of messages) {
    const deps = new Set<string>();

    for (const ref of msg.referencedMessages) {
      deps.add(ref);
    }

    if (msg.causedBy) {
      deps.add(msg.causedBy);
    }

    graph.set(msg.messageId, deps);
  }

  return graph;
}

/**
 * Topological sort of messages based on dependencies
 *
 * @param messages - Messages to sort
 * @returns Sorted messages (dependencies first)
 * @throws Error if circular dependency detected
 */
export function topologicalSort(messages: TruthMessage[]): TruthMessage[] {
  const graph = buildDependencyGraph(messages);
  const messageMap = new Map(messages.map((m) => [m.messageId, m]));
  const sorted: TruthMessage[] = [];
  const visited = new Set<string>();
  const visiting = new Set<string>();

  function visit(id: string): void {
    if (visited.has(id)) return;
    if (visiting.has(id)) {
      throw new Error(`Circular dependency detected involving message ${id}`);
    }

    visiting.add(id);

    const deps = graph.get(id);
    if (deps) {
      for (const dep of deps) {
        if (messageMap.has(dep)) {
          visit(dep);
        }
      }
    }

    visiting.delete(id);
    visited.add(id);

    const msg = messageMap.get(id);
    if (msg) sorted.push(msg);
  }

  for (const msg of messages) {
    visit(msg.messageId);
  }

  return sorted;
}
