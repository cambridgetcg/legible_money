/**
 * Event Persistence
 *
 * Persistent event storage for:
 * - Event sourcing / replay
 * - Audit logging
 * - State reconstruction
 * - Debugging / analysis
 */

import type {
  ProtocolEvent,
  ProtocolEventType,
} from "../../foundation/contracts/index.js";
import type { BlockHeight } from "../../foundation/types/index.js";
import type { EventMiddleware } from "./event_bus.js";

// ============ Types ============

/**
 * Persisted event with metadata
 */
export interface PersistedEvent {
  /** Unique sequence number */
  sequence: bigint;
  /** The event */
  event: ProtocolEvent;
  /** Persistence timestamp */
  persistedAt: number;
  /** Hash for integrity verification */
  hash: string;
  /** Previous event hash (for chain integrity) */
  previousHash: string;
}

/**
 * Event query options
 */
export interface EventQuery {
  /** Filter by event types */
  types?: ProtocolEventType[];
  /** Filter by source */
  source?: string;
  /** Filter by correlation ID */
  correlationId?: string;
  /** Filter by block height range */
  fromBlock?: BlockHeight;
  toBlock?: BlockHeight;
  /** Filter by time range */
  fromTime?: number;
  toTime?: number;
  /** Maximum results */
  limit?: number;
  /** Offset for pagination */
  offset?: number;
  /** Sort order */
  order?: "asc" | "desc";
}

/**
 * Event store interface
 */
export interface EventStore {
  /** Append event to store */
  append(event: ProtocolEvent): Promise<PersistedEvent>;

  /** Get event by sequence number */
  get(sequence: bigint): Promise<PersistedEvent | null>;

  /** Query events */
  query(query: EventQuery): Promise<PersistedEvent[]>;

  /** Get latest sequence number */
  getLatestSequence(): Promise<bigint>;

  /** Get events since a sequence number */
  getSince(sequence: bigint, limit?: number): Promise<PersistedEvent[]>;

  /** Verify chain integrity */
  verifyIntegrity(fromSequence?: bigint, toSequence?: bigint): Promise<boolean>;

  /** Get store stats */
  getStats(): Promise<EventStoreStats>;

  /** Compact old events (if supported) */
  compact?(beforeSequence: bigint): Promise<number>;
}

/**
 * Event store statistics
 */
export interface EventStoreStats {
  totalEvents: bigint;
  oldestEvent?: PersistedEvent;
  newestEvent?: PersistedEvent;
  eventsByType: Record<string, number>;
  storageBytes?: number;
}

/**
 * Snapshot for state reconstruction
 */
export interface EventSnapshot<T> {
  /** Sequence number this snapshot was taken at */
  sequence: bigint;
  /** The snapshot data */
  data: T;
  /** Timestamp */
  createdAt: number;
  /** Hash of the snapshot data */
  hash: string;
}

// ============ In-Memory Event Store ============

/**
 * In-memory event store (for development/testing)
 */
export class InMemoryEventStore implements EventStore {
  private events: PersistedEvent[] = [];
  private sequence = 0n;
  private lastHash = "0".repeat(64);

  async append(event: ProtocolEvent): Promise<PersistedEvent> {
    const sequence = ++this.sequence;
    const previousHash = this.lastHash;
    const hash = this.computeHash(event, sequence, previousHash);

    const persisted: PersistedEvent = {
      sequence,
      event,
      persistedAt: Date.now(),
      hash,
      previousHash,
    };

    this.events.push(persisted);
    this.lastHash = hash;

    return persisted;
  }

  async get(sequence: bigint): Promise<PersistedEvent | null> {
    return this.events.find((e) => e.sequence === sequence) ?? null;
  }

  async query(query: EventQuery): Promise<PersistedEvent[]> {
    let results = [...this.events];

    // Apply filters
    if (query.types?.length) {
      results = results.filter((e) => query.types!.includes(e.event.type));
    }
    if (query.source) {
      results = results.filter((e) => e.event.source === query.source);
    }
    if (query.correlationId) {
      results = results.filter(
        (e) => e.event.correlationId === query.correlationId,
      );
    }
    if (query.fromBlock !== undefined) {
      results = results.filter(
        (e) =>
          e.event.blockHeight !== undefined &&
          e.event.blockHeight >= query.fromBlock!,
      );
    }
    if (query.toBlock !== undefined) {
      results = results.filter(
        (e) =>
          e.event.blockHeight !== undefined &&
          e.event.blockHeight <= query.toBlock!,
      );
    }
    if (query.fromTime !== undefined) {
      results = results.filter((e) => e.event.timestamp >= query.fromTime!);
    }
    if (query.toTime !== undefined) {
      results = results.filter((e) => e.event.timestamp <= query.toTime!);
    }

    // Sort
    if (query.order === "desc") {
      results.reverse();
    }

    // Pagination
    if (query.offset) {
      results = results.slice(query.offset);
    }
    if (query.limit) {
      results = results.slice(0, query.limit);
    }

    return results;
  }

  async getLatestSequence(): Promise<bigint> {
    return this.sequence;
  }

  async getSince(sequence: bigint, limit = 100): Promise<PersistedEvent[]> {
    return this.events.filter((e) => e.sequence > sequence).slice(0, limit);
  }

  async verifyIntegrity(
    fromSequence?: bigint,
    toSequence?: bigint,
  ): Promise<boolean> {
    const start = fromSequence ? Number(fromSequence) - 1 : 0;
    const end = toSequence ? Number(toSequence) : this.events.length;

    for (let i = start; i < end; i++) {
      const event = this.events[i];
      const expectedPrevHash =
        i === 0 ? "0".repeat(64) : this.events[i - 1].hash;

      if (event.previousHash !== expectedPrevHash) {
        return false;
      }

      const expectedHash = this.computeHash(
        event.event,
        event.sequence,
        event.previousHash,
      );

      if (event.hash !== expectedHash) {
        return false;
      }
    }

    return true;
  }

  async getStats(): Promise<EventStoreStats> {
    const eventsByType: Record<string, number> = {};

    for (const event of this.events) {
      eventsByType[event.event.type] =
        (eventsByType[event.event.type] ?? 0) + 1;
    }

    return {
      totalEvents: BigInt(this.events.length),
      oldestEvent: this.events[0],
      newestEvent: this.events[this.events.length - 1],
      eventsByType,
    };
  }

  async compact(beforeSequence: bigint): Promise<number> {
    const originalLength = this.events.length;
    this.events = this.events.filter((e) => e.sequence >= beforeSequence);
    return originalLength - this.events.length;
  }

  // Simple hash for in-memory store (use proper crypto in production)
  private computeHash(
    event: ProtocolEvent,
    sequence: bigint,
    previousHash: string,
  ): string {
    const data = JSON.stringify({
      event,
      sequence: sequence.toString(),
      previousHash,
    });
    let hash = 0;
    for (let i = 0; i < data.length; i++) {
      const char = data.charCodeAt(i);
      hash = (hash << 5) - hash + char;
      hash = hash & hash;
    }
    return Math.abs(hash).toString(16).padStart(64, "0");
  }

  /**
   * Reset store (for testing)
   */
  reset(): void {
    this.events = [];
    this.sequence = 0n;
    this.lastHash = "0".repeat(64);
  }
}

// ============ Event Replay ============

/**
 * Event replay options
 */
export interface ReplayOptions {
  /** Starting sequence (exclusive) */
  fromSequence?: bigint;
  /** Ending sequence (inclusive) */
  toSequence?: bigint;
  /** Event types to replay */
  types?: ProtocolEventType[];
  /** Replay speed multiplier (1 = real-time, 0 = instant) */
  speed?: number;
  /** Handler called for each event */
  onEvent: (event: PersistedEvent) => void | Promise<void>;
  /** Handler called on completion */
  onComplete?: (stats: ReplayStats) => void;
  /** Handler called on error */
  onError?: (error: Error, event: PersistedEvent) => void;
}

/**
 * Replay statistics
 */
export interface ReplayStats {
  eventsReplayed: number;
  startTime: number;
  endTime: number;
  durationMs: number;
  errors: number;
}

/**
 * Replay events from store
 */
export async function replayEvents(
  store: EventStore,
  options: ReplayOptions,
): Promise<ReplayStats> {
  const stats: ReplayStats = {
    eventsReplayed: 0,
    startTime: Date.now(),
    endTime: 0,
    durationMs: 0,
    errors: 0,
  };

  const query: EventQuery = {
    types: options.types,
    order: "asc",
  };

  const events = await store.query(query);
  let lastTimestamp: number | null = null;

  for (const persisted of events) {
    // Filter by sequence range
    if (
      options.fromSequence !== undefined &&
      persisted.sequence <= options.fromSequence
    ) {
      continue;
    }
    if (
      options.toSequence !== undefined &&
      persisted.sequence > options.toSequence
    ) {
      break;
    }

    // Simulate timing if speed > 0
    if (options.speed && options.speed > 0 && lastTimestamp !== null) {
      const delay = (persisted.event.timestamp - lastTimestamp) / options.speed;
      if (delay > 0) {
        await new Promise((resolve) => setTimeout(resolve, delay));
      }
    }
    lastTimestamp = persisted.event.timestamp;

    // Replay event
    try {
      await options.onEvent(persisted);
      stats.eventsReplayed++;
    } catch (error) {
      stats.errors++;
      if (options.onError) {
        options.onError(
          error instanceof Error ? error : new Error(String(error)),
          persisted,
        );
      }
    }
  }

  stats.endTime = Date.now();
  stats.durationMs = stats.endTime - stats.startTime;

  options.onComplete?.(stats);

  return stats;
}

// ============ Persistence Middleware ============

/**
 * Create persistence middleware
 */
export function createPersistenceMiddleware(
  store: EventStore,
): EventMiddleware {
  return async (event, next) => {
    // Persist before processing
    await store.append(event);
    await next();
  };
}

// ============ State Projector ============

/**
 * State projector for building read models from events
 */
export interface StateProjector<T> {
  /** Initial state */
  initialState: T;
  /** Apply event to state */
  apply(state: T, event: ProtocolEvent): T;
  /** Event types this projector handles */
  eventTypes?: ProtocolEventType[];
}

/**
 * Build state from events
 */
export async function projectState<T>(
  store: EventStore,
  projector: StateProjector<T>,
  options?: {
    fromSequence?: bigint;
    toSequence?: bigint;
  },
): Promise<{ state: T; sequence: bigint }> {
  let state = projector.initialState;
  let lastSequence = 0n;

  const events = await store.query({
    types: projector.eventTypes,
    order: "asc",
  });

  for (const persisted of events) {
    if (
      options?.fromSequence !== undefined &&
      persisted.sequence <= options.fromSequence
    ) {
      continue;
    }
    if (
      options?.toSequence !== undefined &&
      persisted.sequence > options.toSequence
    ) {
      break;
    }

    state = projector.apply(state, persisted.event);
    lastSequence = persisted.sequence;
  }

  return { state, sequence: lastSequence };
}

// ============ Snapshot Manager ============

/**
 * Snapshot manager for efficient state reconstruction
 */
export class SnapshotManager<T> {
  private snapshots: EventSnapshot<T>[] = [];
  private readonly maxSnapshots: number;

  constructor(maxSnapshots = 10) {
    this.maxSnapshots = maxSnapshots;
  }

  /**
   * Create a snapshot
   */
  createSnapshot(data: T, sequence: bigint): EventSnapshot<T> {
    const snapshot: EventSnapshot<T> = {
      sequence,
      data,
      createdAt: Date.now(),
      hash: this.computeHash(data),
    };

    this.snapshots.push(snapshot);

    // Prune old snapshots
    if (this.snapshots.length > this.maxSnapshots) {
      this.snapshots.shift();
    }

    return snapshot;
  }

  /**
   * Get latest snapshot before a sequence
   */
  getSnapshotBefore(sequence: bigint): EventSnapshot<T> | null {
    for (let i = this.snapshots.length - 1; i >= 0; i--) {
      if (this.snapshots[i].sequence <= sequence) {
        return this.snapshots[i];
      }
    }
    return null;
  }

  /**
   * Get all snapshots
   */
  getSnapshots(): EventSnapshot<T>[] {
    return [...this.snapshots];
  }

  /**
   * Clear all snapshots
   */
  clear(): void {
    this.snapshots = [];
  }

  private computeHash(data: T): string {
    const str = JSON.stringify(data);
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = (hash << 5) - hash + char;
      hash = hash & hash;
    }
    return Math.abs(hash).toString(16).padStart(64, "0");
  }
}

// ============ Factory ============

/**
 * Create an in-memory event store
 */
export function createEventStore(): EventStore {
  return new InMemoryEventStore();
}
