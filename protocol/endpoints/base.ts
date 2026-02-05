/**
 * Base Endpoint Implementation
 *
 * Provides common functionality for all truth endpoints:
 * - Message creation and signing
 * - Message sending via event bus
 * - Response handling
 * - Rate limiting
 *
 * Specialized endpoints (Claimant, Witness, etc.) extend this base.
 */

import type { DID } from "../../core/agents/identity/did.js";
import type { PrivateKey, PublicKey } from "../../core/crypto/keys.js";
import type { EventBus } from "../../foundation/contracts/index.js";
import type {
  TruthMessage,
  TruthMessageType,
  TruthPayload,
  AcknowledgmentMessage,
} from "../../core/messaging/types.js";
import type {
  TruthEndpoint,
  EndpointRole,
  EndpointCapability,
} from "./types.js";
import {
  generateMessageId,
  generateContentHash,
  signTruthMessage,
} from "../../core/messaging/integrity.js";
import { signString } from "../../core/crypto/signatures.js";

// ============ Types ============

/**
 * Message sending options
 */
export interface SendOptions {
  /** Target recipient (null = broadcast) */
  recipient?: DID | "broadcast";
  /** Causal dependencies */
  referencedMessages?: string[];
  /** Message that triggered this */
  causedBy?: string;
  /** Knowledge domain */
  domain?: string;
  /** Time-to-live in blocks */
  ttlBlocks?: number;
  /** Custom network fee */
  networkFeeUnit?: bigint;
  networkFeeLimit?: bigint;
}

/**
 * Rate limit configuration
 */
export interface RateLimitConfig {
  /** Maximum messages per window */
  maxMessages: number;
  /** Window size in milliseconds */
  windowMs: number;
}

/**
 * Endpoint configuration
 */
export interface EndpointConfig {
  /** Endpoint DID */
  did: DID;
  /** Signing key */
  privateKey: PrivateKey;
  /** Public key for verification */
  publicKey: PublicKey;
  /** Event bus for communication */
  eventBus: EventBus;
  /** Endpoint roles */
  roles: EndpointRole[];
  /** Endpoint capabilities */
  capabilities: EndpointCapability[];
  /** Staked amount */
  stakedAmount: bigint;
  /** Rate limit configuration */
  rateLimit?: RateLimitConfig;
  /** Default network fee unit */
  defaultFeeUnit?: bigint;
  /** Default network fee limit */
  defaultFeeLimit?: bigint;
}

// ============ Base Endpoint ============

/**
 * Base class for truth endpoints
 *
 * Provides common message creation, signing, and sending functionality.
 */
export abstract class BaseEndpoint implements TruthEndpoint {
  // TruthEndpoint fields
  readonly did: DID;
  readonly roles: EndpointRole[];
  readonly capabilities: EndpointCapability[];
  readonly stakedAmount: bigint;
  readonly registeredAt: number;
  lastActiveAt?: number;
  name?: string;
  description?: string;
  p2pAddress?: string;
  httpEndpoint?: string;
  wsEndpoint?: string;
  responseTimeMs?: number;
  availability?: number;
  reputation?: number;
  validatorStatus?: "active" | "inactive" | "jailed";

  // Internal state
  protected readonly privateKey: PrivateKey;
  protected readonly publicKey: PublicKey;
  protected readonly eventBus: EventBus;
  protected readonly rateLimit?: RateLimitConfig;
  protected readonly defaultFeeUnit: bigint;
  protected readonly defaultFeeLimit: bigint;

  // Rate limiting state
  private messageCount = 0;
  private windowStart = Date.now();

  // Pending responses
  protected readonly pendingResponses = new Map<
    string,
    {
      resolve: (msg: TruthMessage) => void;
      reject: (error: Error) => void;
      timeout: ReturnType<typeof setTimeout>;
    }
  >();

  constructor(config: EndpointConfig) {
    this.did = config.did;
    this.privateKey = config.privateKey;
    this.publicKey = config.publicKey;
    this.eventBus = config.eventBus;
    this.roles = config.roles;
    this.capabilities = config.capabilities;
    this.stakedAmount = config.stakedAmount;
    this.rateLimit = config.rateLimit;
    this.defaultFeeUnit = config.defaultFeeUnit ?? 1n;
    this.defaultFeeLimit = config.defaultFeeLimit ?? 100000n;
    this.registeredAt = Date.now();

    // Subscribe to acknowledgments
    this.eventBus.on("acknowledgment" as never, (event) => {
      this.handleAcknowledgment(
        event as unknown as { payload: AcknowledgmentMessage },
      );
    });
  }

  // ============ Message Creation ============

  /**
   * Create a TruthMessage with proper structure
   */
  protected createMessage<T extends TruthPayload>(
    messageType: TruthMessageType,
    payload: T,
    options: SendOptions = {},
  ): Omit<TruthMessage, "messageId" | "contentHash" | "signature"> {
    return {
      messageType,
      version: "1.0.0",
      sender: this.did,
      recipient: options.recipient,
      payload,
      referencedMessages: options.referencedMessages ?? [],
      causedBy: options.causedBy,
      networkFeeUnit: options.networkFeeUnit ?? this.defaultFeeUnit,
      networkFeeLimit: options.networkFeeLimit ?? this.defaultFeeLimit,
      timestamp: Date.now(),
      domain: options.domain,
      ttlBlocks: options.ttlBlocks,
    };
  }

  /**
   * Finalize and sign a message
   */
  protected async finalizeMessage(
    message: Omit<TruthMessage, "messageId" | "contentHash" | "signature">,
  ): Promise<TruthMessage> {
    // Generate content hash
    const contentHash = generateContentHash(message.payload);

    // Generate message ID
    const messageId = generateMessageId({
      ...message,
      contentHash,
    } as TruthMessage);

    // Create complete message (without signature yet)
    const completeMessage: TruthMessage = {
      ...message,
      messageId,
      contentHash,
      signature: "" as never, // Placeholder
    };

    // Sign the message
    const signedMessage = signTruthMessage(
      completeMessage,
      this.privateKey,
      signString,
    );

    return signedMessage;
  }

  // ============ Message Sending ============

  /**
   * Send a message via the event bus
   */
  protected async send(message: TruthMessage): Promise<void> {
    // Check rate limit
    this.checkRateLimit();

    // Update activity timestamp
    this.lastActiveAt = Date.now();

    // Emit to event bus
    this.eventBus.emit({
      type: message.messageType as never,
      timestamp: message.timestamp,
      source: "endpoint",
      payload: message as never,
    });
  }

  /**
   * Send a message and wait for acknowledgment
   */
  protected async sendAndWait(
    message: TruthMessage,
    timeoutMs = 30000,
  ): Promise<AcknowledgmentMessage> {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.pendingResponses.delete(message.messageId);
        reject(
          new Error(`Timeout waiting for acknowledgment: ${message.messageId}`),
        );
      }, timeoutMs);

      this.pendingResponses.set(message.messageId, {
        resolve: resolve as (msg: TruthMessage) => void,
        reject,
        timeout,
      });

      this.send(message).catch((error) => {
        clearTimeout(timeout);
        this.pendingResponses.delete(message.messageId);
        reject(error);
      });
    });
  }

  /**
   * Handle incoming acknowledgment
   */
  private handleAcknowledgment(event: {
    payload: AcknowledgmentMessage;
  }): void {
    const ack = event.payload;
    const pending = this.pendingResponses.get(
      ack.payload.acknowledgedMessageId,
    );

    if (pending) {
      clearTimeout(pending.timeout);
      this.pendingResponses.delete(ack.payload.acknowledgedMessageId);

      if (ack.payload.status === "rejected") {
        pending.reject(
          new Error(
            `Message rejected: ${ack.payload.error?.message ?? "Unknown error"}`,
          ),
        );
      } else {
        pending.resolve(ack);
      }
    }
  }

  // ============ Rate Limiting ============

  /**
   * Check if we're within rate limits
   */
  private checkRateLimit(): void {
    if (!this.rateLimit) return;

    const now = Date.now();

    // Reset window if expired
    if (now - this.windowStart >= this.rateLimit.windowMs) {
      this.windowStart = now;
      this.messageCount = 0;
    }

    // Check limit
    if (this.messageCount >= this.rateLimit.maxMessages) {
      throw new Error(
        `Rate limit exceeded: ${this.rateLimit.maxMessages} messages per ${this.rateLimit.windowMs}ms`,
      );
    }

    this.messageCount++;
  }

  // ============ Utility Methods ============

  /**
   * Check if endpoint has a capability
   */
  hasCapability(cap: EndpointCapability): boolean {
    return this.capabilities.includes(cap);
  }

  /**
   * Check if endpoint has a role
   */
  hasRole(role: EndpointRole): boolean {
    return this.roles.includes(role);
  }

  /**
   * Update reputation score
   */
  updateReputation(delta: number): void {
    const current = this.reputation ?? 0.5;
    this.reputation = Math.max(0, Math.min(1, current + delta));
  }

  /**
   * Shutdown endpoint
   */
  shutdown(): void {
    // Clear all pending responses
    for (const [id, pending] of this.pendingResponses) {
      clearTimeout(pending.timeout);
      pending.reject(new Error("Endpoint shutdown"));
      this.pendingResponses.delete(id);
    }
  }
}
