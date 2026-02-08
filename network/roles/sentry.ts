/**
 * Sentry Node Role
 *
 * Validator DDoS shield that acts as a public-facing proxy.
 * The sentry node subscribes to all topics and relays messages
 * to/from its protected validator over a private channel.
 * The validator's peer ID is never exposed to the public network.
 *
 * Capabilities:
 * - Topics: all 12 (proxies everything)
 * - Storage: full (mirrors validator state)
 * - Sync: full
 * - DHT: server (public-facing)
 * - Protocols: all core + private sentry link
 */

import { EventEmitter } from "events";
import { ROLE_PROTOCOL_IDS } from "./types.js";

// ============ Constants ============

/** Maximum messages per second from a single peer before throttling */
const DEFAULT_RATE_LIMIT = 100;

/** Interval for health checks to the validator (ms) */
const VALIDATOR_HEALTH_INTERVAL = 5_000;

/** Maximum time without validator heartbeat before alert (ms) */
const VALIDATOR_TIMEOUT = 30_000;

/** Maximum queued messages for the validator */
const MAX_VALIDATOR_QUEUE = 10_000;

// ============ Types ============

export interface SentryConfig {
  /** Peer ID of the protected validator (private, never shared) */
  validatorPeerId: string;

  /** Whether to use an encrypted private channel */
  privateChannel: boolean;

  /** Rate limit per peer (messages/second) */
  rateLimitPerPeer: number;

  /** Connection allowlist (peer IDs) — empty means allow all */
  connectionAllowlist: string[];

  /** Connection blocklist (peer IDs) */
  connectionBlocklist: string[];

  /** Health check interval (ms) */
  healthCheckInterval: number;

  /** Validator timeout before alert (ms) */
  validatorTimeout: number;
}

export interface SentryStats {
  /** Total messages relayed to validator */
  messagesRelayedToValidator: number;

  /** Total messages relayed from validator */
  messagesRelayedFromValidator: number;

  /** Messages dropped due to rate limiting */
  messagesDropped: number;

  /** Blocked connection attempts */
  connectionsBlocked: number;

  /** Whether the validator is currently reachable */
  validatorConnected: boolean;

  /** Last successful validator heartbeat */
  lastValidatorHeartbeat: number;

  /** Uptime in ms */
  uptime: number;
}

export type SentryEvent =
  | { type: "validator:connected" }
  | { type: "validator:disconnected" }
  | { type: "validator:timeout" }
  | { type: "peer:blocked"; peerId: string; reason: string }
  | { type: "rate_limit:triggered"; peerId: string };

// ============ Rate Limiter ============

/**
 * Simple sliding-window rate limiter per peer.
 */
class PeerRateLimiter {
  private windows: Map<string, number[]> = new Map();
  private limit: number;

  constructor(limit: number) {
    this.limit = limit;
  }

  /**
   * Check if a message from this peer should be allowed.
   * Returns true if within rate limit.
   */
  allow(peerId: string): boolean {
    const now = Date.now();
    const window = this.windows.get(peerId) ?? [];

    // Remove entries older than 1 second
    const recent = window.filter((t) => now - t < 1000);
    recent.push(now);

    this.windows.set(peerId, recent);
    return recent.length <= this.limit;
  }

  /**
   * Clear stale entries periodically.
   */
  cleanup(): void {
    const now = Date.now();
    for (const [peerId, window] of this.windows) {
      const recent = window.filter((t) => now - t < 1000);
      if (recent.length === 0) {
        this.windows.delete(peerId);
      } else {
        this.windows.set(peerId, recent);
      }
    }
  }
}

// ============ Sentry Node Service ============

/**
 * Sentry node service that shields a validator from direct network exposure.
 *
 * Architecture:
 * ```
 * Public Network <---> [Sentry Node] <--- private link ---> [Validator]
 *                      (public peer ID)                     (hidden peer ID)
 * ```
 *
 * The sentry:
 * 1. Accepts all public connections
 * 2. Rate-limits and filters messages
 * 3. Relays valid messages to the validator over a private channel
 * 4. Relays validator's outbound messages to the public network
 * 5. Monitors validator health
 */
export class SentryNodeService extends EventEmitter {
  private config: SentryConfig;
  private rateLimiter: PeerRateLimiter;
  private healthTimer: ReturnType<typeof setInterval> | null = null;
  private cleanupTimer: ReturnType<typeof setInterval> | null = null;
  private startedAt: number = 0;
  private validatorConnected: boolean = false;
  private lastValidatorHeartbeat: number = 0;

  // Stats
  private messagesRelayedToValidator: number = 0;
  private messagesRelayedFromValidator: number = 0;
  private messagesDropped: number = 0;
  private connectionsBlocked: number = 0;

  // Message queue for validator
  private validatorQueue: Array<{
    topic: string;
    data: Uint8Array;
    from: string;
  }> = [];

  constructor(config: Partial<SentryConfig> & { validatorPeerId: string }) {
    super();
    this.config = {
      privateChannel: true,
      rateLimitPerPeer: DEFAULT_RATE_LIMIT,
      connectionAllowlist: [],
      connectionBlocklist: [],
      healthCheckInterval: VALIDATOR_HEALTH_INTERVAL,
      validatorTimeout: VALIDATOR_TIMEOUT,
      ...config,
    };
    this.rateLimiter = new PeerRateLimiter(this.config.rateLimitPerPeer);
  }

  /**
   * Start the sentry service.
   */
  start(): void {
    this.startedAt = Date.now();

    // Start health monitoring
    this.healthTimer = setInterval(
      () => this.checkValidatorHealth(),
      this.config.healthCheckInterval,
    );

    // Start rate limiter cleanup
    this.cleanupTimer = setInterval(() => this.rateLimiter.cleanup(), 10_000);

    this.emit("started");
  }

  /**
   * Stop the sentry service.
   */
  stop(): void {
    if (this.healthTimer) {
      clearInterval(this.healthTimer);
      this.healthTimer = null;
    }
    if (this.cleanupTimer) {
      clearInterval(this.cleanupTimer);
      this.cleanupTimer = null;
    }
    this.validatorQueue = [];
    this.emit("stopped");
  }

  /**
   * Check if a peer connection should be allowed.
   * Returns false if the peer is blocklisted or not in the allowlist.
   */
  shouldAllowConnection(peerId: string): boolean {
    // Always block blocklisted peers
    if (this.config.connectionBlocklist.includes(peerId)) {
      this.connectionsBlocked++;
      this.emit("event", {
        type: "peer:blocked",
        peerId,
        reason: "blocklisted",
      } as SentryEvent);
      return false;
    }

    // If allowlist is set, only allow listed peers (plus the validator)
    if (this.config.connectionAllowlist.length > 0) {
      if (
        !this.config.connectionAllowlist.includes(peerId) &&
        peerId !== this.config.validatorPeerId
      ) {
        this.connectionsBlocked++;
        this.emit("event", {
          type: "peer:blocked",
          peerId,
          reason: "not_allowlisted",
        } as SentryEvent);
        return false;
      }
    }

    return true;
  }

  /**
   * Filter an inbound message before relaying to the validator.
   * Returns true if the message should be relayed.
   */
  filterInboundMessage(
    fromPeerId: string,
    topic: string,
    data: Uint8Array,
  ): boolean {
    // Rate limit check
    if (!this.rateLimiter.allow(fromPeerId)) {
      this.messagesDropped++;
      this.emit("event", {
        type: "rate_limit:triggered",
        peerId: fromPeerId,
      } as SentryEvent);
      return false;
    }

    // Never relay from blocklisted peers
    if (this.config.connectionBlocklist.includes(fromPeerId)) {
      this.messagesDropped++;
      return false;
    }

    // Queue for validator
    if (this.validatorQueue.length < MAX_VALIDATOR_QUEUE) {
      this.validatorQueue.push({ topic, data, from: fromPeerId });
      this.messagesRelayedToValidator++;
      return true;
    }

    // Queue full — drop
    this.messagesDropped++;
    return false;
  }

  /**
   * Get the next batch of messages to relay to the validator.
   */
  drainValidatorQueue(
    maxBatch: number = 100,
  ): Array<{ topic: string; data: Uint8Array; from: string }> {
    const batch = this.validatorQueue.splice(0, maxBatch);
    return batch;
  }

  /**
   * Handle an outbound message from the validator.
   * This is relayed to the public network with the sentry's peer ID.
   */
  handleValidatorOutbound(_topic: string, _data: Uint8Array): void {
    this.messagesRelayedFromValidator++;
    this.lastValidatorHeartbeat = Date.now();
    // Actual relay happens in the network layer — this just tracks stats
  }

  /**
   * Record validator heartbeat (called when any message from validator arrives).
   */
  recordValidatorHeartbeat(): void {
    const wasDisconnected = !this.validatorConnected;
    this.validatorConnected = true;
    this.lastValidatorHeartbeat = Date.now();

    if (wasDisconnected) {
      this.emit("event", { type: "validator:connected" } as SentryEvent);
    }
  }

  /**
   * Get the protected validator's peer ID.
   * This should NEVER be shared with external peers.
   */
  getValidatorPeerId(): string {
    return this.config.validatorPeerId;
  }

  /**
   * Get sentry statistics.
   */
  getStats(): SentryStats {
    return {
      messagesRelayedToValidator: this.messagesRelayedToValidator,
      messagesRelayedFromValidator: this.messagesRelayedFromValidator,
      messagesDropped: this.messagesDropped,
      connectionsBlocked: this.connectionsBlocked,
      validatorConnected: this.validatorConnected,
      lastValidatorHeartbeat: this.lastValidatorHeartbeat,
      uptime: this.startedAt > 0 ? Date.now() - this.startedAt : 0,
    };
  }

  /**
   * Get the protocol ID for the sentry-validator private link.
   */
  static get protocolId(): string {
    return ROLE_PROTOCOL_IDS.SENTRY_LINK;
  }

  // ============ Internal ============

  private checkValidatorHealth(): void {
    if (
      this.lastValidatorHeartbeat > 0 &&
      Date.now() - this.lastValidatorHeartbeat > this.config.validatorTimeout
    ) {
      if (this.validatorConnected) {
        this.validatorConnected = false;
        this.emit("event", {
          type: "validator:disconnected",
        } as SentryEvent);
        this.emit("event", { type: "validator:timeout" } as SentryEvent);
      }
    }
  }
}
