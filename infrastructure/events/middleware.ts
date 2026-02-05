/**
 * Event Middleware
 *
 * Cross-cutting concerns for event processing:
 * - Logging
 * - Validation
 * - Metrics collection
 * - Rate limiting
 * - Tracing/correlation
 */

import type {
  ProtocolEvent,
  ProtocolEventType,
} from "../../foundation/contracts/index.js";
import type { EventMiddleware } from "./event_bus.js";

// ============ Logging Middleware ============

/**
 * Log level for events
 */
export type LogLevel = "debug" | "info" | "warn" | "error";

/**
 * Logger interface
 */
export interface EventLogger {
  debug(message: string, data?: Record<string, unknown>): void;
  info(message: string, data?: Record<string, unknown>): void;
  warn(message: string, data?: Record<string, unknown>): void;
  error(message: string, data?: Record<string, unknown>): void;
}

/**
 * Logging middleware configuration
 */
export interface LoggingConfig {
  logger: EventLogger;
  logLevel?: LogLevel;
  /** Event types to always log (regardless of level) */
  alwaysLog?: ProtocolEventType[];
  /** Event types to never log */
  neverLog?: ProtocolEventType[];
  /** Include payload in logs */
  includePayload?: boolean;
}

/**
 * Create logging middleware
 */
export function createLoggingMiddleware(
  config: LoggingConfig,
): EventMiddleware {
  const {
    logger,
    logLevel = "info",
    alwaysLog = [],
    neverLog = [],
    includePayload = false,
  } = config;

  const shouldLog = (eventType: ProtocolEventType): boolean => {
    if (neverLog.includes(eventType)) return false;
    if (alwaysLog.includes(eventType)) return true;
    return true;
  };

  const getLogLevel = (eventType: ProtocolEventType): LogLevel => {
    // Error-related events get higher priority
    if (eventType.includes("rejected") || eventType.includes("slashed")) {
      return "warn";
    }
    if (eventType.includes("error") || eventType.includes("failed")) {
      return "error";
    }
    return logLevel;
  };

  return async (event, next) => {
    if (!shouldLog(event.type)) {
      await next();
      return;
    }

    const level = getLogLevel(event.type);
    const data: Record<string, unknown> = {
      eventType: event.type,
      source: event.source,
      timestamp: event.timestamp,
      correlationId: event.correlationId,
      blockHeight: event.blockHeight?.toString(),
    };

    if (includePayload) {
      data.payload = event.payload;
    }

    const startTime = Date.now();

    try {
      logger[level](`Event emitted: ${event.type}`, data);
      await next();
      logger.debug(`Event processed: ${event.type}`, {
        ...data,
        processingTimeMs: Date.now() - startTime,
      });
    } catch (error) {
      logger.error(`Event processing failed: ${event.type}`, {
        ...data,
        error: error instanceof Error ? error.message : String(error),
        processingTimeMs: Date.now() - startTime,
      });
      throw error;
    }
  };
}

// ============ Validation Middleware ============

/**
 * Event validator function
 */
export type EventValidator = (event: ProtocolEvent) => boolean | string;

/**
 * Validation middleware configuration
 */
export interface ValidationConfig {
  /** Validators by event type */
  validators?: Partial<Record<ProtocolEventType, EventValidator>>;
  /** Global validator applied to all events */
  globalValidator?: EventValidator;
  /** Whether to throw on validation failure */
  throwOnFailure?: boolean;
  /** Handler for validation failures */
  onValidationFailure?: (event: ProtocolEvent, reason: string) => void;
}

/**
 * Create validation middleware
 */
export function createValidationMiddleware(
  config: ValidationConfig,
): EventMiddleware {
  const {
    validators = {},
    globalValidator,
    throwOnFailure = true,
    onValidationFailure,
  } = config;

  const validate = (event: ProtocolEvent): string | null => {
    // Global validation
    if (globalValidator) {
      const result = globalValidator(event);
      if (result !== true) {
        return typeof result === "string" ? result : "Global validation failed";
      }
    }

    // Type-specific validation
    const validator = validators[event.type];
    if (validator) {
      const result = validator(event);
      if (result !== true) {
        return typeof result === "string"
          ? result
          : `Validation failed for ${event.type}`;
      }
    }

    return null;
  };

  return async (event, next) => {
    const failure = validate(event);

    if (failure) {
      onValidationFailure?.(event, failure);

      if (throwOnFailure) {
        throw new Error(`Event validation failed: ${failure}`);
      }

      // Skip processing invalid event
      return;
    }

    await next();
  };
}

// ============ Metrics Middleware ============

/**
 * Metrics collector interface
 */
export interface MetricsCollector {
  increment(metric: string, tags?: Record<string, string>): void;
  timing(metric: string, value: number, tags?: Record<string, string>): void;
  gauge(metric: string, value: number, tags?: Record<string, string>): void;
}

/**
 * Metrics middleware configuration
 */
export interface MetricsConfig {
  collector: MetricsCollector;
  /** Prefix for all metrics */
  prefix?: string;
}

/**
 * Create metrics middleware
 */
export function createMetricsMiddleware(
  config: MetricsConfig,
): EventMiddleware {
  const { collector, prefix = "events" } = config;

  return async (event, next) => {
    const tags = {
      event_type: event.type,
      source: event.source,
    };

    // Count event
    collector.increment(`${prefix}.emitted`, tags);

    const startTime = Date.now();

    try {
      await next();

      // Success metrics
      collector.increment(`${prefix}.processed`, tags);
      collector.timing(
        `${prefix}.processing_time`,
        Date.now() - startTime,
        tags,
      );
    } catch (error) {
      // Failure metrics
      collector.increment(`${prefix}.failed`, {
        ...tags,
        error_type: error instanceof Error ? error.name : "unknown",
      });
      collector.timing(
        `${prefix}.processing_time`,
        Date.now() - startTime,
        tags,
      );
      throw error;
    }
  };
}

// ============ Rate Limiting Middleware ============

/**
 * Rate limit configuration
 */
export interface RateLimitConfig {
  /** Maximum events per window */
  maxEvents: number;
  /** Window size in milliseconds */
  windowMs: number;
  /** Event types to rate limit (empty = all) */
  eventTypes?: ProtocolEventType[];
  /** Handler for rate limit exceeded */
  onRateLimitExceeded?: (event: ProtocolEvent) => void;
}

/**
 * Create rate limiting middleware
 */
export function createRateLimitMiddleware(
  config: RateLimitConfig,
): EventMiddleware {
  const { maxEvents, windowMs, eventTypes, onRateLimitExceeded } = config;

  const windows = new Map<string, { count: number; resetAt: number }>();

  const getWindowKey = (event: ProtocolEvent): string => {
    return `${event.type}:${event.source}`;
  };

  const shouldLimit = (eventType: ProtocolEventType): boolean => {
    if (!eventTypes || eventTypes.length === 0) return true;
    return eventTypes.includes(eventType);
  };

  return async (event, next) => {
    if (!shouldLimit(event.type)) {
      await next();
      return;
    }

    const key = getWindowKey(event);
    const now = Date.now();

    let window = windows.get(key);

    // Reset window if expired
    if (!window || now >= window.resetAt) {
      window = { count: 0, resetAt: now + windowMs };
      windows.set(key, window);
    }

    // Check rate limit
    if (window.count >= maxEvents) {
      onRateLimitExceeded?.(event);
      throw new Error(
        `Rate limit exceeded for ${event.type} from ${event.source}`,
      );
    }

    window.count++;
    await next();
  };
}

// ============ Tracing Middleware ============

/**
 * Tracing configuration
 */
export interface TracingConfig {
  /** Generate correlation ID if missing */
  generateCorrelationId?: boolean;
  /** Trace context propagation */
  propagateContext?: boolean;
  /** Trace recorder */
  onTrace?: (trace: EventTrace) => void;
}

/**
 * Event trace record
 */
export interface EventTrace {
  correlationId: string;
  eventType: ProtocolEventType;
  source: string;
  timestamp: number;
  processingTimeMs: number;
  success: boolean;
  error?: string;
}

/**
 * Create tracing middleware
 */
export function createTracingMiddleware(
  config: TracingConfig,
): EventMiddleware {
  const { generateCorrelationId = true, onTrace } = config;

  const generateId = (): string => {
    return `trace-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  };

  return async (event, next) => {
    // Ensure correlation ID
    if (generateCorrelationId && !event.correlationId) {
      (event as { correlationId: string }).correlationId = generateId();
    }

    const startTime = Date.now();
    let success = true;
    let error: string | undefined;

    try {
      await next();
    } catch (err) {
      success = false;
      error = err instanceof Error ? err.message : String(err);
      throw err;
    } finally {
      if (onTrace) {
        onTrace({
          correlationId: event.correlationId ?? "unknown",
          eventType: event.type,
          source: event.source,
          timestamp: event.timestamp,
          processingTimeMs: Date.now() - startTime,
          success,
          error,
        });
      }
    }
  };
}

// ============ Filter Middleware ============

/**
 * Filter configuration
 */
export interface FilterConfig {
  /** Event types to allow (whitelist) */
  allow?: ProtocolEventType[];
  /** Event types to block (blacklist) */
  block?: ProtocolEventType[];
  /** Custom filter function */
  filter?: (event: ProtocolEvent) => boolean;
}

/**
 * Create filter middleware
 */
export function createFilterMiddleware(config: FilterConfig): EventMiddleware {
  const { allow, block, filter } = config;

  const shouldProcess = (event: ProtocolEvent): boolean => {
    // Blacklist takes precedence
    if (block?.includes(event.type)) return false;

    // Whitelist check
    if (allow && !allow.includes(event.type)) return false;

    // Custom filter
    if (filter && !filter(event)) return false;

    return true;
  };

  return async (event, next) => {
    if (shouldProcess(event)) {
      await next();
    }
    // Silently skip filtered events
  };
}

// ============ Retry Middleware ============

/**
 * Retry configuration
 */
export interface RetryConfig {
  /** Maximum retry attempts */
  maxRetries: number;
  /** Base delay between retries in ms */
  baseDelayMs: number;
  /** Exponential backoff multiplier */
  backoffMultiplier?: number;
  /** Errors that should trigger retry */
  retryableErrors?: string[];
}

/**
 * Create retry middleware
 */
export function createRetryMiddleware(config: RetryConfig): EventMiddleware {
  const {
    maxRetries,
    baseDelayMs,
    backoffMultiplier = 2,
    retryableErrors,
  } = config;

  const shouldRetry = (error: Error): boolean => {
    if (!retryableErrors || retryableErrors.length === 0) return true;
    return retryableErrors.some(
      (e) => error.message.includes(e) || error.name === e,
    );
  };

  const delay = (ms: number): Promise<void> => {
    return new Promise((resolve) => setTimeout(resolve, ms));
  };

  return async (_event, next) => {
    let lastError: Error | null = null;

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        await next();
        return;
      } catch (error) {
        lastError = error instanceof Error ? error : new Error(String(error));

        if (attempt < maxRetries && shouldRetry(lastError)) {
          const delayMs = baseDelayMs * Math.pow(backoffMultiplier, attempt);
          await delay(delayMs);
        }
      }
    }

    if (lastError) {
      throw lastError;
    }
  };
}

// ============ Composite Middleware ============

/**
 * Combine multiple middleware into one
 */
export function composeMiddleware(
  ...middleware: EventMiddleware[]
): EventMiddleware {
  return async (event, next) => {
    const runMiddleware = async (index: number): Promise<void> => {
      if (index >= middleware.length) {
        await next();
        return;
      }
      await middleware[index](event, () => runMiddleware(index + 1));
    };

    await runMiddleware(0);
  };
}
