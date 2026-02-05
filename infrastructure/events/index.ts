/**
 * Event Infrastructure Module
 *
 * Production-ready event system for cross-module communication.
 *
 * Usage:
 *   import { createEventBus, createEventStore } from '@legible/infrastructure/events';
 *
 *   const eventBus = createEventBus({ asyncDelivery: true });
 *   const eventStore = createEventStore();
 *
 *   // Add middleware
 *   eventBus.use(createLoggingMiddleware({ logger: console }));
 *   eventBus.use(createPersistenceMiddleware(eventStore));
 *
 *   // Subscribe to events
 *   eventBus.on('fact_verified', (event) => {
 *     console.log('Fact verified:', event.payload.factId);
 *   });
 *
 *   // Emit events
 *   eventBus.emit(createEvent('fact_verified', 'knowledge', { ... }));
 */

// Event Bus
export {
  ProductionEventBus,
  createEventBus,
  type DeliveryOptions,
  type EventMiddleware,
  type EventBusConfig,
  type EventMetrics,
  type DeadLetterEntry,
} from "./event_bus.js";

// Middleware
export {
  // Logging
  createLoggingMiddleware,
  type LogLevel,
  type EventLogger,
  type LoggingConfig,

  // Validation
  createValidationMiddleware,
  type EventValidator,
  type ValidationConfig,

  // Metrics
  createMetricsMiddleware,
  type MetricsCollector,
  type MetricsConfig,

  // Rate Limiting
  createRateLimitMiddleware,
  type RateLimitConfig,

  // Tracing
  createTracingMiddleware,
  type TracingConfig,
  type EventTrace,

  // Filter
  createFilterMiddleware,
  type FilterConfig,

  // Retry
  createRetryMiddleware,
  type RetryConfig,

  // Composition
  composeMiddleware,
} from "./middleware.js";

// Persistence
export {
  // Event Store
  InMemoryEventStore,
  createEventStore,
  type EventStore,
  type PersistedEvent,
  type EventQuery,
  type EventStoreStats,

  // Replay
  replayEvents,
  type ReplayOptions,
  type ReplayStats,

  // Persistence Middleware
  createPersistenceMiddleware,

  // State Projection
  projectState,
  type StateProjector,

  // Snapshots
  SnapshotManager,
  type EventSnapshot,
} from "./persistence.js";
