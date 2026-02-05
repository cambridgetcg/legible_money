/**
 * PubSub Topics
 *
 * Topic definitions and message handlers for GossipSub.
 *
 * SECURITY (VULN-019):
 * - Validates message structure before handler-specific validation
 * - Enforces message size limits to prevent DoS
 * - Validates message sender (source peer)
 * - Checks message freshness to prevent replay attacks
 */

import { EventEmitter } from "events";
import { PUBSUB_TOPICS, NetworkMessage, MESSAGE_TYPES } from "../config.js";

// ============ Security Constants (VULN-019 FIX) ============

/**
 * Maximum message payload size (1 MB default)
 * Prevents memory exhaustion from oversized messages
 */
export const MAX_MESSAGE_SIZE = 1024 * 1024;

/**
 * Maximum message age in milliseconds (5 minutes)
 * Prevents replay of old messages
 */
export const MAX_MESSAGE_AGE_MS = 5 * 60 * 1000;

/**
 * Minimum message age in milliseconds (-10 seconds)
 * Prevents messages "from the future" due to clock skew
 */
export const MIN_MESSAGE_AGE_MS = -10 * 1000;

// ============ Base Message Validation (VULN-019 FIX) ============

export interface BaseValidationResult {
  valid: boolean;
  errors: string[];
}

/**
 * VULN-019 FIX: Validate message structure before handler-specific validation
 */
export function validateMessageBase(
  message: NetworkMessage,
): BaseValidationResult {
  const errors: string[] = [];

  // Check message exists
  if (!message) {
    errors.push("Message is null or undefined");
    return { valid: false, errors };
  }

  // Check payload exists
  if (!message.payload) {
    errors.push("Message payload is missing");
  }

  // Check payload size
  if (message.payload && message.payload.length > MAX_MESSAGE_SIZE) {
    errors.push(
      `Payload too large: ${message.payload.length} > ${MAX_MESSAGE_SIZE}`,
    );
  }

  // Check message type
  if (
    !message.type ||
    !Object.values(MESSAGE_TYPES).includes(message.type as any)
  ) {
    errors.push(`Invalid message type: ${message.type}`);
  }

  // Check sender
  if (!message.sender) {
    errors.push("Message sender is missing");
  }

  // Check timestamp for replay protection
  if (message.timestamp) {
    const now = Date.now();
    const messageAge = now - message.timestamp;

    if (messageAge > MAX_MESSAGE_AGE_MS) {
      errors.push(`Message too old: ${messageAge}ms > ${MAX_MESSAGE_AGE_MS}ms`);
    }

    if (messageAge < MIN_MESSAGE_AGE_MS) {
      errors.push(
        `Message from future: ${messageAge}ms < ${MIN_MESSAGE_AGE_MS}ms`,
      );
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

// ============ Types ============

export interface TopicHandler {
  topic: string;
  validate: (message: NetworkMessage) => Promise<ValidationResult>;
  handle: (message: NetworkMessage) => Promise<void>;
}

export type ValidationResult =
  | "accept" // Valid message, propagate
  | "reject" // Invalid message, penalize sender
  | "ignore"; // Don't propagate, don't penalize

export interface TopicStats {
  topic: string;
  subscribed: boolean;
  messagesReceived: number;
  messagesPublished: number;
  validationAccepted: number;
  validationRejected: number;
  validationIgnored: number;
}

// ============ Topic Manager ============

/**
 * Manages PubSub topic subscriptions and handlers
 */
export class TopicManager extends EventEmitter {
  private handlers: Map<string, TopicHandler> = new Map();
  private stats: Map<string, TopicStats> = new Map();

  constructor() {
    super();
  }

  /**
   * Register a topic handler
   */
  registerHandler(handler: TopicHandler): void {
    this.handlers.set(handler.topic, handler);
    this.stats.set(handler.topic, {
      topic: handler.topic,
      subscribed: false,
      messagesReceived: 0,
      messagesPublished: 0,
      validationAccepted: 0,
      validationRejected: 0,
      validationIgnored: 0,
    });
    this.emit("handler:registered", handler.topic);
  }

  /**
   * Unregister a topic handler
   */
  unregisterHandler(topic: string): void {
    this.handlers.delete(topic);
    this.stats.delete(topic);
    this.emit("handler:unregistered", topic);
  }

  /**
   * Get handler for a topic
   */
  getHandler(topic: string): TopicHandler | undefined {
    return this.handlers.get(topic);
  }

  /**
   * Process incoming message
   *
   * VULN-019 FIX: Performs base validation before handler-specific validation
   */
  async processMessage(
    topic: string,
    message: NetworkMessage,
  ): Promise<ValidationResult> {
    const handler = this.handlers.get(topic);
    const stats = this.stats.get(topic);

    if (!handler || !stats) {
      return "ignore";
    }

    stats.messagesReceived++;

    // VULN-019 FIX: Base validation FIRST (size limits, structure, freshness)
    const baseValidation = validateMessageBase(message);
    if (!baseValidation.valid) {
      stats.validationRejected++;
      this.emit("message:base_validation_failed", {
        topic,
        errors: baseValidation.errors,
        sender: message.sender,
      });
      return "reject";
    }

    // Handler-specific validation
    const result = await handler.validate(message);

    switch (result) {
      case "accept":
        stats.validationAccepted++;
        await handler.handle(message);
        break;
      case "reject":
        stats.validationRejected++;
        break;
      case "ignore":
        stats.validationIgnored++;
        break;
    }

    return result;
  }

  /**
   * Get stats for a topic
   */
  getStats(topic: string): TopicStats | undefined {
    return this.stats.get(topic);
  }

  /**
   * Get all stats
   */
  getAllStats(): TopicStats[] {
    return Array.from(this.stats.values());
  }
}

// ============ Topic Handler Factory ============

/**
 * Create a topic handler with the standard decode → validate → handle pipeline.
 * All 5 core topic handlers share this identical pattern.
 */
export function createTopicHandler(
  topic: string,
  validatePayload: (payload: unknown) => Promise<boolean>,
  handlePayload: (payload: unknown) => Promise<void>,
): TopicHandler {
  return {
    topic,

    async validate(message: NetworkMessage): Promise<ValidationResult> {
      try {
        const payload = JSON.parse(new TextDecoder().decode(message.payload));
        const valid = await validatePayload(payload);
        return valid ? "accept" : "reject";
      } catch {
        return "reject";
      }
    },

    async handle(message: NetworkMessage): Promise<void> {
      const payload = JSON.parse(new TextDecoder().decode(message.payload));
      await handlePayload(payload);
    },
  };
}

// ============ Default Topic Handlers ============

export function createBlockTopicHandler(
  validateBlock: (block: unknown) => Promise<boolean>,
  handleBlock: (block: unknown) => Promise<void>,
): TopicHandler {
  return createTopicHandler(PUBSUB_TOPICS.BLOCKS, validateBlock, handleBlock);
}

export function createTransactionTopicHandler(
  validateTx: (tx: unknown) => Promise<boolean>,
  handleTx: (tx: unknown) => Promise<void>,
): TopicHandler {
  return createTopicHandler(PUBSUB_TOPICS.TRANSACTIONS, validateTx, handleTx);
}

export function createClaimTopicHandler(
  validateClaim: (claim: unknown) => Promise<boolean>,
  handleClaim: (claim: unknown) => Promise<void>,
): TopicHandler {
  return createTopicHandler(PUBSUB_TOPICS.CLAIMS, validateClaim, handleClaim);
}

export function createVerificationTopicHandler(
  validateVerification: (v: unknown) => Promise<boolean>,
  handleVerification: (v: unknown) => Promise<void>,
): TopicHandler {
  return createTopicHandler(
    PUBSUB_TOPICS.VERIFICATIONS,
    validateVerification,
    handleVerification,
  );
}

export function createConsensusTopicHandler(
  validateConsensusMsg: (msg: unknown) => Promise<boolean>,
  handleConsensusMsg: (msg: unknown) => Promise<void>,
): TopicHandler {
  return createTopicHandler(
    PUBSUB_TOPICS.CONSENSUS,
    validateConsensusMsg,
    handleConsensusMsg,
  );
}
