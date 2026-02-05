/**
 * Message Delivery Tracking
 *
 * Provides reliable delivery for TruthMessages:
 * - Acknowledgment tracking
 * - Automatic retries with exponential backoff
 * - Delivery status monitoring
 * - Timeout handling
 *
 * Ensures messages are delivered even in unreliable network conditions.
 */

import type { DID } from "../agents/identity/did.js";
import type { TruthMessage, AcknowledgmentMessage } from "./types.js";

// ============ Configuration ============

/**
 * Delivery configuration
 */
export interface DeliveryConfig {
  /** Maximum retry attempts */
  maxRetries: number;

  /** Initial retry delay in milliseconds */
  retryBackoffMs: number;

  /** Maximum retry delay (cap for exponential backoff) */
  maxRetryDelayMs: number;

  /** Timeout waiting for acknowledgment */
  ackTimeoutMs: number;

  /** Window for deduplication (in blocks) */
  deduplicationWindowBlocks: number;

  /** Whether to use exponential backoff */
  exponentialBackoff: boolean;

  /** Backoff multiplier */
  backoffMultiplier: number;
}

/**
 * Default delivery configuration
 */
export const DEFAULT_DELIVERY_CONFIG: DeliveryConfig = {
  maxRetries: 3,
  retryBackoffMs: 5000,
  maxRetryDelayMs: 60000,
  ackTimeoutMs: 30000,
  deduplicationWindowBlocks: 100,
  exponentialBackoff: true,
  backoffMultiplier: 2,
};

// ============ Delivery Status ============

/**
 * Delivery status for a message
 */
export type DeliveryStatus =
  | "pending" // Not yet sent
  | "sent" // Sent, awaiting acknowledgment
  | "acknowledged" // Acknowledgment received
  | "retrying" // Retry in progress
  | "failed" // All retries exhausted
  | "expired" // TTL exceeded
  | "cancelled"; // Manually cancelled

/**
 * Pending delivery record
 */
export interface PendingDelivery {
  /** The message being delivered */
  message: TruthMessage;

  /** Target recipient */
  recipient: DID | "broadcast";

  /** Current delivery status */
  status: DeliveryStatus;

  /** When first sent */
  sentAt: number;

  /** Number of retry attempts */
  retryCount: number;

  /** When last retry was attempted */
  lastRetryAt?: number;

  /** When next retry is scheduled */
  nextRetryAt?: number;

  /** Error from last attempt (if any) */
  lastError?: string;

  /** Acknowledgment message (if received) */
  acknowledgment?: AcknowledgmentMessage;
}

/**
 * Delivery result
 */
export interface DeliveryResult {
  /** Message ID */
  messageId: string;

  /** Final status */
  status: DeliveryStatus;

  /** Total attempts made */
  attempts: number;

  /** Total time from first send to completion */
  totalTimeMs: number;

  /** Acknowledgment (if received) */
  acknowledgment?: AcknowledgmentMessage;

  /** Error (if failed) */
  error?: string;
}

// ============ Event Types ============

/**
 * Delivery events
 */
export type DeliveryEvent =
  | { type: "sent"; messageId: string; recipient: DID | "broadcast" }
  | { type: "acknowledged"; messageId: string; ack: AcknowledgmentMessage }
  | {
      type: "retry_scheduled";
      messageId: string;
      retryAt: number;
      attempt: number;
    }
  | { type: "retry_attempted"; messageId: string; attempt: number }
  | { type: "failed"; messageId: string; reason: string; attempts: number }
  | { type: "expired"; messageId: string }
  | { type: "cancelled"; messageId: string };

/**
 * Event listener type
 */
export type DeliveryEventListener = (event: DeliveryEvent) => void;

// ============ Delivery Tracker ============

/**
 * MessageDeliveryTracker - Tracks message delivery and handles retries
 *
 * Usage:
 * ```typescript
 * const tracker = new MessageDeliveryTracker(config);
 *
 * // Track a sent message
 * tracker.trackSent(message, recipient);
 *
 * // Handle acknowledgment
 * tracker.handleAcknowledgment(ackMessage);
 *
 * // Listen for events
 * tracker.addEventListener((event) => {
 *   if (event.type === 'failed') {
 *     console.log('Delivery failed:', event.messageId);
 *   }
 * });
 * ```
 */
export class MessageDeliveryTracker {
  private pending: Map<string, PendingDelivery> = new Map();
  private completed: Map<string, DeliveryResult> = new Map();
  private timers: Map<string, ReturnType<typeof setTimeout>> = new Map();
  private listeners: DeliveryEventListener[] = [];
  private config: DeliveryConfig;

  constructor(config: DeliveryConfig = DEFAULT_DELIVERY_CONFIG) {
    this.config = config;
  }

  // ============ Public API ============

  /**
   * Track a message that has been sent
   *
   * @param message - The sent message
   * @param recipient - Target recipient
   */
  trackSent(message: TruthMessage, recipient: DID | "broadcast"): void {
    const delivery: PendingDelivery = {
      message,
      recipient,
      status: "sent",
      sentAt: Date.now(),
      retryCount: 0,
    };

    this.pending.set(message.messageId, delivery);
    this.emit({ type: "sent", messageId: message.messageId, recipient });

    // Set acknowledgment timeout
    this.scheduleAckTimeout(message.messageId);
  }

  /**
   * Handle an acknowledgment message
   *
   * @param ack - Acknowledgment message
   */
  handleAcknowledgment(ack: AcknowledgmentMessage): void {
    const messageId = ack.payload.acknowledgedMessageId;
    const delivery = this.pending.get(messageId);

    if (!delivery) {
      // Already completed or unknown message
      return;
    }

    // Clear any pending timers
    this.clearTimer(messageId);

    // Update delivery status
    delivery.status = "acknowledged";
    delivery.acknowledgment = ack;

    // Move to completed
    const result: DeliveryResult = {
      messageId,
      status: "acknowledged",
      attempts: delivery.retryCount + 1,
      totalTimeMs: Date.now() - delivery.sentAt,
      acknowledgment: ack,
    };

    this.pending.delete(messageId);
    this.completed.set(messageId, result);

    this.emit({ type: "acknowledged", messageId, ack });
  }

  /**
   * Mark a message as manually cancelled
   *
   * @param messageId - Message to cancel
   */
  cancel(messageId: string): void {
    const delivery = this.pending.get(messageId);
    if (!delivery) return;

    this.clearTimer(messageId);
    delivery.status = "cancelled";

    const result: DeliveryResult = {
      messageId,
      status: "cancelled",
      attempts: delivery.retryCount + 1,
      totalTimeMs: Date.now() - delivery.sentAt,
    };

    this.pending.delete(messageId);
    this.completed.set(messageId, result);

    this.emit({ type: "cancelled", messageId });
  }

  /**
   * Get delivery status for a message
   *
   * @param messageId - Message ID to check
   * @returns Delivery status or undefined if not tracked
   */
  getStatus(messageId: string): DeliveryStatus | undefined {
    const pending = this.pending.get(messageId);
    if (pending) return pending.status;

    const completed = this.completed.get(messageId);
    if (completed) return completed.status;

    return undefined;
  }

  /**
   * Get pending delivery record
   *
   * @param messageId - Message ID
   * @returns Pending delivery or undefined
   */
  getPending(messageId: string): PendingDelivery | undefined {
    return this.pending.get(messageId);
  }

  /**
   * Get completed delivery result
   *
   * @param messageId - Message ID
   * @returns Delivery result or undefined
   */
  getResult(messageId: string): DeliveryResult | undefined {
    return this.completed.get(messageId);
  }

  /**
   * Get all pending deliveries
   *
   * @returns Array of pending deliveries
   */
  getAllPending(): PendingDelivery[] {
    return Array.from(this.pending.values());
  }

  /**
   * Add event listener
   *
   * @param listener - Event listener function
   */
  addEventListener(listener: DeliveryEventListener): void {
    this.listeners.push(listener);
  }

  /**
   * Remove event listener
   *
   * @param listener - Event listener to remove
   */
  removeEventListener(listener: DeliveryEventListener): void {
    const index = this.listeners.indexOf(listener);
    if (index !== -1) {
      this.listeners.splice(index, 1);
    }
  }

  /**
   * Set retry callback
   *
   * Called when a retry should be attempted. The callback is responsible
   * for actually resending the message.
   *
   * @param callback - Retry callback
   */
  setRetryCallback(
    callback: (
      message: TruthMessage,
      recipient: DID | "broadcast",
    ) => Promise<void>,
  ): void {
    this.retryCallback = callback;
  }

  private retryCallback?: (
    message: TruthMessage,
    recipient: DID | "broadcast",
  ) => Promise<void>;

  /**
   * Clean up completed deliveries older than specified age
   *
   * @param maxAgeMs - Maximum age in milliseconds
   */
  cleanup(maxAgeMs: number): void {
    const cutoff = Date.now() - maxAgeMs;

    for (const [messageId, _result] of this.completed) {
      const delivery = this.pending.get(messageId);
      if (delivery && delivery.sentAt < cutoff) {
        this.completed.delete(messageId);
      }
    }
  }

  /**
   * Shutdown the tracker, cancelling all pending timers
   */
  shutdown(): void {
    for (const timer of this.timers.values()) {
      clearTimeout(timer);
    }
    this.timers.clear();
    this.pending.clear();
    this.listeners = [];
  }

  // ============ Internal Methods ============

  private scheduleAckTimeout(messageId: string): void {
    const timer = setTimeout(() => {
      this.handleAckTimeout(messageId);
    }, this.config.ackTimeoutMs);

    this.timers.set(messageId, timer);
  }

  private async handleAckTimeout(messageId: string): Promise<void> {
    const delivery = this.pending.get(messageId);
    if (!delivery) return;

    // Check if we should retry
    if (delivery.retryCount < this.config.maxRetries) {
      await this.scheduleRetry(messageId);
    } else {
      // Max retries exceeded
      this.handleDeliveryFailure(messageId, "Max retries exceeded");
    }
  }

  private async scheduleRetry(messageId: string): Promise<void> {
    const delivery = this.pending.get(messageId);
    if (!delivery) return;

    delivery.status = "retrying";
    delivery.retryCount++;
    delivery.lastRetryAt = Date.now();

    // Calculate backoff delay
    const delay = this.calculateBackoff(delivery.retryCount);
    delivery.nextRetryAt = Date.now() + delay;

    this.emit({
      type: "retry_scheduled",
      messageId,
      retryAt: delivery.nextRetryAt,
      attempt: delivery.retryCount,
    });

    // Schedule retry
    const timer = setTimeout(async () => {
      await this.attemptRetry(messageId);
    }, delay);

    this.timers.set(messageId, timer);
  }

  private async attemptRetry(messageId: string): Promise<void> {
    const delivery = this.pending.get(messageId);
    if (!delivery || delivery.status === "cancelled") return;

    this.emit({
      type: "retry_attempted",
      messageId,
      attempt: delivery.retryCount,
    });

    // Call retry callback if set
    if (this.retryCallback) {
      try {
        await this.retryCallback(delivery.message, delivery.recipient);
        delivery.status = "sent";

        // Schedule new ack timeout
        this.scheduleAckTimeout(messageId);
      } catch (error) {
        delivery.lastError =
          error instanceof Error ? error.message : "Unknown error";

        if (delivery.retryCount < this.config.maxRetries) {
          await this.scheduleRetry(messageId);
        } else {
          this.handleDeliveryFailure(messageId, delivery.lastError);
        }
      }
    } else {
      // No retry callback, schedule next ack timeout
      delivery.status = "sent";
      this.scheduleAckTimeout(messageId);
    }
  }

  private handleDeliveryFailure(messageId: string, reason: string): void {
    const delivery = this.pending.get(messageId);
    if (!delivery) return;

    this.clearTimer(messageId);
    delivery.status = "failed";
    delivery.lastError = reason;

    const result: DeliveryResult = {
      messageId,
      status: "failed",
      attempts: delivery.retryCount + 1,
      totalTimeMs: Date.now() - delivery.sentAt,
      error: reason,
    };

    this.pending.delete(messageId);
    this.completed.set(messageId, result);

    this.emit({
      type: "failed",
      messageId,
      reason,
      attempts: delivery.retryCount + 1,
    });
  }

  private calculateBackoff(attempt: number): number {
    if (!this.config.exponentialBackoff) {
      return this.config.retryBackoffMs;
    }

    const delay =
      this.config.retryBackoffMs *
      Math.pow(this.config.backoffMultiplier, attempt - 1);

    return Math.min(delay, this.config.maxRetryDelayMs);
  }

  private clearTimer(messageId: string): void {
    const timer = this.timers.get(messageId);
    if (timer) {
      clearTimeout(timer);
      this.timers.delete(messageId);
    }
  }

  private emit(event: DeliveryEvent): void {
    for (const listener of this.listeners) {
      try {
        listener(event);
      } catch {
        // Ignore listener errors
      }
    }
  }
}

// ============ Batch Delivery ============

/**
 * Batch delivery result
 */
export interface BatchDeliveryResult {
  /** Total messages in batch */
  total: number;

  /** Successfully delivered */
  delivered: number;

  /** Failed deliveries */
  failed: number;

  /** Individual results */
  results: DeliveryResult[];
}

/**
 * Track multiple message deliveries as a batch
 *
 * @param tracker - Delivery tracker
 * @param messages - Messages to track
 * @param recipients - Recipient for each message
 * @returns Promise that resolves when all complete
 */
export async function trackBatchDelivery(
  tracker: MessageDeliveryTracker,
  messages: TruthMessage[],
  recipients: (DID | "broadcast")[],
): Promise<BatchDeliveryResult> {
  if (messages.length !== recipients.length) {
    throw new Error("Messages and recipients arrays must have same length");
  }

  // Track all messages
  for (let i = 0; i < messages.length; i++) {
    tracker.trackSent(messages[i], recipients[i]);
  }

  // Wait for all to complete
  const results: DeliveryResult[] = [];
  const messageIds = messages.map((m) => m.messageId);

  return new Promise((resolve) => {
    const checkComplete = () => {
      const allComplete = messageIds.every((id) => {
        const status = tracker.getStatus(id);
        return (
          status === "acknowledged" ||
          status === "failed" ||
          status === "cancelled"
        );
      });

      if (allComplete) {
        for (const id of messageIds) {
          const result = tracker.getResult(id);
          if (result) results.push(result);
        }

        resolve({
          total: messages.length,
          delivered: results.filter((r) => r.status === "acknowledged").length,
          failed: results.filter((r) => r.status === "failed").length,
          results,
        });
      }
    };

    tracker.addEventListener((event) => {
      if (
        messageIds.includes(
          event.type === "acknowledged"
            ? event.messageId
            : event.type === "failed"
              ? event.messageId
              : event.type === "cancelled"
                ? event.messageId
                : "",
        )
      ) {
        checkComplete();
      }
    });

    // Initial check in case some already complete
    checkComplete();
  });
}
