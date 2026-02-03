/**
 * PubSub Topics
 *
 * Topic definitions and message handlers for GossipSub.
 */

import { EventEmitter } from 'events';
import { PUBSUB_TOPICS, NetworkMessage, MESSAGE_TYPES } from '../config.js';

// ============ Types ============

export interface TopicHandler {
  topic: string;
  validate: (message: NetworkMessage) => Promise<ValidationResult>;
  handle: (message: NetworkMessage) => Promise<void>;
}

export type ValidationResult =
  | 'accept'    // Valid message, propagate
  | 'reject'    // Invalid message, penalize sender
  | 'ignore';   // Don't propagate, don't penalize

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
    this.emit('handler:registered', handler.topic);
  }

  /**
   * Unregister a topic handler
   */
  unregisterHandler(topic: string): void {
    this.handlers.delete(topic);
    this.stats.delete(topic);
    this.emit('handler:unregistered', topic);
  }

  /**
   * Get handler for a topic
   */
  getHandler(topic: string): TopicHandler | undefined {
    return this.handlers.get(topic);
  }

  /**
   * Process incoming message
   */
  async processMessage(topic: string, message: NetworkMessage): Promise<ValidationResult> {
    const handler = this.handlers.get(topic);
    const stats = this.stats.get(topic);

    if (!handler || !stats) {
      return 'ignore';
    }

    stats.messagesReceived++;

    // Validate message
    const result = await handler.validate(message);

    switch (result) {
      case 'accept':
        stats.validationAccepted++;
        await handler.handle(message);
        break;
      case 'reject':
        stats.validationRejected++;
        break;
      case 'ignore':
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

// ============ Default Topic Handlers ============

/**
 * Create block topic handler
 */
export function createBlockTopicHandler(
  validateBlock: (block: unknown) => Promise<boolean>,
  handleBlock: (block: unknown) => Promise<void>
): TopicHandler {
  return {
    topic: PUBSUB_TOPICS.BLOCKS,

    async validate(message: NetworkMessage): Promise<ValidationResult> {
      try {
        // Decode block from payload
        const block = JSON.parse(new TextDecoder().decode(message.payload));

        // Validate block
        const valid = await validateBlock(block);
        return valid ? 'accept' : 'reject';
      } catch {
        return 'reject';
      }
    },

    async handle(message: NetworkMessage): Promise<void> {
      const block = JSON.parse(new TextDecoder().decode(message.payload));
      await handleBlock(block);
    },
  };
}

/**
 * Create transaction topic handler
 */
export function createTransactionTopicHandler(
  validateTx: (tx: unknown) => Promise<boolean>,
  handleTx: (tx: unknown) => Promise<void>
): TopicHandler {
  return {
    topic: PUBSUB_TOPICS.TRANSACTIONS,

    async validate(message: NetworkMessage): Promise<ValidationResult> {
      try {
        const tx = JSON.parse(new TextDecoder().decode(message.payload));
        const valid = await validateTx(tx);
        return valid ? 'accept' : 'reject';
      } catch {
        return 'reject';
      }
    },

    async handle(message: NetworkMessage): Promise<void> {
      const tx = JSON.parse(new TextDecoder().decode(message.payload));
      await handleTx(tx);
    },
  };
}

/**
 * Create claim topic handler
 */
export function createClaimTopicHandler(
  validateClaim: (claim: unknown) => Promise<boolean>,
  handleClaim: (claim: unknown) => Promise<void>
): TopicHandler {
  return {
    topic: PUBSUB_TOPICS.CLAIMS,

    async validate(message: NetworkMessage): Promise<ValidationResult> {
      try {
        const claim = JSON.parse(new TextDecoder().decode(message.payload));
        const valid = await validateClaim(claim);
        return valid ? 'accept' : 'reject';
      } catch {
        return 'reject';
      }
    },

    async handle(message: NetworkMessage): Promise<void> {
      const claim = JSON.parse(new TextDecoder().decode(message.payload));
      await handleClaim(claim);
    },
  };
}

/**
 * Create verification topic handler
 */
export function createVerificationTopicHandler(
  validateVerification: (v: unknown) => Promise<boolean>,
  handleVerification: (v: unknown) => Promise<void>
): TopicHandler {
  return {
    topic: PUBSUB_TOPICS.VERIFICATIONS,

    async validate(message: NetworkMessage): Promise<ValidationResult> {
      try {
        const verification = JSON.parse(new TextDecoder().decode(message.payload));
        const valid = await validateVerification(verification);
        return valid ? 'accept' : 'reject';
      } catch {
        return 'reject';
      }
    },

    async handle(message: NetworkMessage): Promise<void> {
      const verification = JSON.parse(new TextDecoder().decode(message.payload));
      await handleVerification(verification);
    },
  };
}

/**
 * Create consensus topic handler
 */
export function createConsensusTopicHandler(
  validateConsensusMsg: (msg: unknown) => Promise<boolean>,
  handleConsensusMsg: (msg: unknown) => Promise<void>
): TopicHandler {
  return {
    topic: PUBSUB_TOPICS.CONSENSUS,

    async validate(message: NetworkMessage): Promise<ValidationResult> {
      try {
        const consensusMsg = JSON.parse(new TextDecoder().decode(message.payload));
        const valid = await validateConsensusMsg(consensusMsg);
        return valid ? 'accept' : 'reject';
      } catch {
        return 'reject';
      }
    },

    async handle(message: NetworkMessage): Promise<void> {
      const consensusMsg = JSON.parse(new TextDecoder().decode(message.payload));
      await handleConsensusMsg(consensusMsg);
    },
  };
}
