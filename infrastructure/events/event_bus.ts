/**
 * Production Event Bus Implementation
 *
 * FLOW-2: Event-Driven Communication
 * Cross-module communication uses events, not direct calls.
 *
 * Features:
 * - Async event delivery with error isolation
 * - Middleware pipeline for cross-cutting concerns
 * - Event prioritization
 * - Dead letter queue for failed events
 * - Metrics and observability hooks
 */

import type {
  EventBus,
  ProtocolEvent,
  ProtocolEventType,
  EventHandler,
  Subscription,
} from "../../foundation/contracts/index.js";

// ============ Types ============

/**
 * Event delivery options
 */
export interface DeliveryOptions {
  /** Maximum time to wait for handler completion */
  timeout?: number;
  /** Priority (higher = processed first) */
  priority?: number;
  /** Whether to persist the event */
  persist?: boolean;
}

/**
 * Handler registration with metadata
 */
interface HandlerRegistration<T extends ProtocolEventType = ProtocolEventType> {
  id: string;
  eventType: T;
  handler: EventHandler<T>;
  priority: number;
  createdAt: number;
}

/**
 * Middleware function type
 */
export type EventMiddleware = (
  event: ProtocolEvent,
  next: () => Promise<void>,
) => Promise<void>;

/**
 * Event bus configuration
 */
export interface EventBusConfig {
  /** Maximum events to keep in history */
  historyLimit?: number;
  /** Default handler timeout in ms */
  defaultTimeout?: number;
  /** Enable async delivery (handlers run in parallel) */
  asyncDelivery?: boolean;
  /** Maximum concurrent handlers */
  maxConcurrency?: number;
  /** Dead letter queue handler */
  onDeadLetter?: (
    event: ProtocolEvent,
    error: Error,
    handlerId: string,
  ) => void;
  /** Metrics collector */
  onMetrics?: (metrics: EventMetrics) => void;
}

/**
 * Event metrics
 */
export interface EventMetrics {
  eventType: ProtocolEventType;
  handlerCount: number;
  deliveryTimeMs: number;
  successCount: number;
  failureCount: number;
}

/**
 * Dead letter entry
 */
export interface DeadLetterEntry {
  event: ProtocolEvent;
  error: Error;
  handlerId: string;
  timestamp: number;
  retryCount: number;
}

// ============ Production Event Bus ============

/**
 * Production-ready event bus implementation
 */
export class ProductionEventBus implements EventBus {
  private handlers = new Map<ProtocolEventType, Set<HandlerRegistration>>();
  private history: ProtocolEvent[] = [];
  private middleware: EventMiddleware[] = [];
  private subscriptionCounter = 0;
  private deadLetterQueue: DeadLetterEntry[] = [];
  private readonly config: Required<EventBusConfig>;

  constructor(config: EventBusConfig = {}) {
    this.config = {
      historyLimit: config.historyLimit ?? 1000,
      defaultTimeout: config.defaultTimeout ?? 5000,
      asyncDelivery: config.asyncDelivery ?? true,
      maxConcurrency: config.maxConcurrency ?? 10,
      onDeadLetter: config.onDeadLetter ?? (() => {}),
      onMetrics: config.onMetrics ?? (() => {}),
    };
  }

  // ============ Core EventBus Interface ============

  emit<T extends ProtocolEventType>(event: ProtocolEvent<T>): void {
    // Add to history
    this.addToHistory(event);

    // Run through middleware and deliver
    this.deliverEvent(event).catch((error) => {
      console.error(`Event delivery failed for ${event.type}:`, error);
    });
  }

  on<T extends ProtocolEventType>(
    eventType: T,
    handler: EventHandler<T>,
    priority = 0,
  ): Subscription {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set());
    }

    const registration: HandlerRegistration<T> = {
      id: `sub-${++this.subscriptionCounter}`,
      eventType,
      handler,
      priority,
      createdAt: Date.now(),
    };

    this.handlers
      .get(eventType)!
      .add(registration as unknown as HandlerRegistration);

    return {
      id: registration.id,
      eventType,
      unsubscribe: () => {
        this.handlers
          .get(eventType)
          ?.delete(registration as unknown as HandlerRegistration);
      },
    };
  }

  once<T extends ProtocolEventType>(
    eventType: T,
    handler: EventHandler<T>,
  ): Subscription {
    const wrappedHandler: EventHandler<T> = (event) => {
      subscription.unsubscribe();
      return handler(event);
    };
    const subscription = this.on(eventType, wrappedHandler);
    return subscription;
  }

  onMany(
    eventTypes: ProtocolEventType[],
    handler: EventHandler<ProtocolEventType>,
  ): Subscription[] {
    return eventTypes.map((type) => this.on(type, handler));
  }

  off(eventType: ProtocolEventType): void {
    this.handlers.delete(eventType);
  }

  async waitFor<T extends ProtocolEventType>(
    eventType: T,
    predicate?: (event: ProtocolEvent<T>) => boolean,
    timeoutMs = 5000,
  ): Promise<ProtocolEvent<T>> {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        subscription.unsubscribe();
        reject(new Error(`Timeout waiting for event: ${eventType}`));
      }, timeoutMs);

      const subscription = this.on(eventType, (event) => {
        if (!predicate || predicate(event as ProtocolEvent<T>)) {
          clearTimeout(timeout);
          subscription.unsubscribe();
          resolve(event as ProtocolEvent<T>);
        }
      });
    });
  }

  getRecentEvents(count = 100): ProtocolEvent[] {
    return this.history.slice(-count);
  }

  clearHistory(): void {
    this.history = [];
  }

  // ============ Extended API ============

  /**
   * Add middleware to the event pipeline
   */
  use(middleware: EventMiddleware): void {
    this.middleware.push(middleware);
  }

  /**
   * Emit and wait for all handlers to complete
   */
  async emitAsync<T extends ProtocolEventType>(
    event: ProtocolEvent<T>,
  ): Promise<void> {
    this.addToHistory(event);
    await this.deliverEvent(event);
  }

  /**
   * Get handler count for an event type
   */
  getHandlerCount(eventType: ProtocolEventType): number {
    return this.handlers.get(eventType)?.size ?? 0;
  }

  /**
   * Get all registered event types
   */
  getRegisteredEventTypes(): ProtocolEventType[] {
    return Array.from(this.handlers.keys());
  }

  /**
   * Get dead letter queue entries
   */
  getDeadLetters(): DeadLetterEntry[] {
    return [...this.deadLetterQueue];
  }

  /**
   * Retry dead letter entries
   */
  async retryDeadLetters(maxRetries = 3): Promise<number> {
    let retried = 0;
    const remaining: DeadLetterEntry[] = [];

    for (const entry of this.deadLetterQueue) {
      if (entry.retryCount < maxRetries) {
        entry.retryCount++;
        try {
          await this.deliverEvent(entry.event);
          retried++;
        } catch {
          remaining.push(entry);
        }
      } else {
        remaining.push(entry);
      }
    }

    this.deadLetterQueue = remaining;
    return retried;
  }

  /**
   * Clear dead letter queue
   */
  clearDeadLetters(): void {
    this.deadLetterQueue = [];
  }

  // ============ Internal Methods ============

  private addToHistory(event: ProtocolEvent): void {
    this.history.push(event);
    if (this.history.length > this.config.historyLimit) {
      this.history.shift();
    }
  }

  private async deliverEvent(event: ProtocolEvent): Promise<void> {
    const startTime = Date.now();
    let successCount = 0;
    let failureCount = 0;

    // Run through middleware pipeline
    const runMiddleware = async (index: number): Promise<void> => {
      if (index >= this.middleware.length) {
        // End of middleware, deliver to handlers
        await this.deliverToHandlers(event);
        return;
      }

      await this.middleware[index](event, () => runMiddleware(index + 1));
    };

    try {
      await runMiddleware(0);
    } catch (error) {
      failureCount++;
      throw error;
    }

    // Collect metrics
    const handlers = this.handlers.get(event.type);
    if (handlers) {
      successCount = handlers.size;
    }

    this.config.onMetrics({
      eventType: event.type,
      handlerCount: successCount + failureCount,
      deliveryTimeMs: Date.now() - startTime,
      successCount,
      failureCount,
    });
  }

  private async deliverToHandlers(event: ProtocolEvent): Promise<void> {
    const registrations = this.handlers.get(event.type);
    if (!registrations || registrations.size === 0) {
      return;
    }

    // Sort by priority (higher first)
    const sorted = Array.from(registrations).sort(
      (a, b) => b.priority - a.priority,
    );

    if (this.config.asyncDelivery) {
      // Parallel delivery with concurrency limit
      await this.deliverParallel(event, sorted);
    } else {
      // Sequential delivery
      await this.deliverSequential(event, sorted);
    }
  }

  private async deliverParallel(
    event: ProtocolEvent,
    handlers: HandlerRegistration[],
  ): Promise<void> {
    const chunks = this.chunk(handlers, this.config.maxConcurrency);

    for (const chunk of chunks) {
      await Promise.all(chunk.map((reg) => this.invokeHandler(event, reg)));
    }
  }

  private async deliverSequential(
    event: ProtocolEvent,
    handlers: HandlerRegistration[],
  ): Promise<void> {
    for (const reg of handlers) {
      await this.invokeHandler(event, reg);
    }
  }

  private async invokeHandler(
    event: ProtocolEvent,
    registration: HandlerRegistration,
  ): Promise<void> {
    try {
      const result = registration.handler(
        event as ProtocolEvent<ProtocolEventType>,
      );

      if (result instanceof Promise) {
        await this.withTimeout(result, this.config.defaultTimeout);
      }
    } catch (error) {
      const err = error instanceof Error ? error : new Error(String(error));

      // Add to dead letter queue
      this.deadLetterQueue.push({
        event,
        error: err,
        handlerId: registration.id,
        timestamp: Date.now(),
        retryCount: 0,
      });

      this.config.onDeadLetter(event, err, registration.id);
    }
  }

  private async withTimeout<T>(
    promise: Promise<T>,
    timeoutMs: number,
  ): Promise<T> {
    return Promise.race([
      promise,
      new Promise<T>((_, reject) =>
        setTimeout(
          () => reject(new Error(`Handler timeout after ${timeoutMs}ms`)),
          timeoutMs,
        ),
      ),
    ]);
  }

  private chunk<T>(array: T[], size: number): T[][] {
    const chunks: T[][] = [];
    for (let i = 0; i < array.length; i += size) {
      chunks.push(array.slice(i, i + size));
    }
    return chunks;
  }
}

// ============ Factory ============

/**
 * Create a production event bus
 */
export function createEventBus(config?: EventBusConfig): EventBus {
  return new ProductionEventBus(config);
}
