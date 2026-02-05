/**
 * Payment Channel State Tree
 *
 * Manages payment channel state with in-memory storage and indexes.
 * Enables streaming micropayments for long-running API sessions.
 *
 * Pattern follows ClaimStateTree (state/claims.ts).
 *
 * GAP-005 Implementation
 */

import { domainHash } from "../core/crypto/hashing.js";
import { bytesToHex } from "../core/crypto/keys.js";
import type {
  PaymentChannelState,
  PaymentChannelStatus,
  SerializedPaymentDistribution,
} from "./types.js";

// ============ Constants ============

export const PAYMENT_CHANNEL_CONSTANTS = {
  // Minimum deposit (100 LGM)
  MIN_DEPOSIT: 100n * 10n ** 18n,

  // Maximum channel duration (30 days at 2.5s blocks = ~1,036,800 blocks)
  MAX_DURATION_BLOCKS: 1036800n,

  // Default settlement frequency (100 blocks = ~4 minutes)
  DEFAULT_SETTLEMENT_FREQUENCY: 100n,

  // Dispute period for unilateral close (1000 blocks = ~42 minutes)
  DISPUTE_PERIOD_BLOCKS: 1000n,

  // Grace period after expiry for claims (500 blocks = ~21 minutes)
  EXPIRY_GRACE_PERIOD: 500n,

  // Fraud penalty (5% of remaining balance)
  FRAUD_PENALTY_RATE: 0.05,

  // Escrow account for channel funds
  ESCROW_DID: "did:lgm:payment-channel-escrow",

  // Treasury for protocol share
  TREASURY_DID: "did:lgm:treasury",
};

// ============ Utility Functions ============

/**
 * Generate deterministic channel ID
 */
export function generateChannelId(
  payer: string,
  provider: string,
  blockNumber: bigint,
): string {
  const hash = bytesToHex(
    domainHash(
      "LGM.payment_channel.id.v1",
      `${payer}:${provider}:${blockNumber}`,
    ),
  );
  return `pc:${hash.slice(0, 32)}`;
}

/**
 * Compute state hash for signing
 */
export function computeStateHash(
  channelId: string,
  nonce: bigint,
  spent: bigint,
  distributions: SerializedPaymentDistribution[],
): string {
  return bytesToHex(
    domainHash(
      "LGM.payment_channel.state.v1",
      JSON.stringify({
        channelId,
        nonce: nonce.toString(),
        spent: spent.toString(),
        distributions,
      }),
    ),
  );
}

/**
 * Serialize a bigint value for JSON
 */
export function serializeBigInt(value: bigint): string {
  return value.toString();
}

/**
 * Deserialize a bigint value from JSON
 */
export function deserializeBigInt(value: string): bigint {
  return BigInt(value);
}

/**
 * Sum all amounts in distributions
 */
export function sumDistributions(
  distributions: SerializedPaymentDistribution[],
): bigint {
  let total = 0n;
  for (const dist of distributions) {
    total += BigInt(dist.apiProvider.amount);
    for (const reward of dist.knowledgeRewards) {
      total += BigInt(reward.amount);
    }
    total += BigInt(dist.protocol.burned);
    total += BigInt(dist.protocol.treasury);
    if (dist.research) {
      total += BigInt(dist.research.amount);
    }
  }
  return total;
}

// ============ Payment Channel State Tree ============

/**
 * Payment Channel State Tree
 *
 * Manages payment channels with in-memory storage and multiple indexes
 * for efficient queries.
 */
export class PaymentChannelStateTree {
  private channels: Map<string, PaymentChannelState> = new Map();

  // Indexes for efficient queries
  private byPayer: Map<string, Set<string>> = new Map();
  private byProvider: Map<string, Set<string>> = new Map();
  private byStatus: Map<PaymentChannelStatus, Set<string>> = new Map();
  private byExpiration: Map<string, Set<string>> = new Map(); // blockNumber -> channelIds

  constructor() {
    // Initialize status indexes
    this.byStatus.set("open", new Set());
    this.byStatus.set("closing", new Set());
    this.byStatus.set("disputed", new Set());
    this.byStatus.set("closed", new Set());
  }

  // ============ Channel Operations ============

  /**
   * Add a new payment channel
   */
  addChannel(channel: PaymentChannelState): void {
    if (this.channels.has(channel.channelId)) {
      throw new Error(`Channel ${channel.channelId} already exists`);
    }

    this.channels.set(channel.channelId, channel);
    this.addToIndexes(channel);
  }

  /**
   * Get a channel by ID
   */
  getChannel(channelId: string): PaymentChannelState | undefined {
    return this.channels.get(channelId);
  }

  /**
   * Check if a channel exists
   */
  hasChannel(channelId: string): boolean {
    return this.channels.has(channelId);
  }

  /**
   * Update a channel's state
   */
  updateChannel(
    channelId: string,
    updates: Partial<PaymentChannelState>,
  ): PaymentChannelState {
    const channel = this.channels.get(channelId);
    if (!channel) {
      throw new Error(`Channel ${channelId} not found`);
    }

    // Remove from old indexes if status is changing
    if (updates.status !== undefined && updates.status !== channel.status) {
      this.removeFromStatusIndex(channel);
    }

    // Apply updates
    const updatedChannel: PaymentChannelState = {
      ...channel,
      ...updates,
    };

    this.channels.set(channelId, updatedChannel);

    // Re-add to indexes if status changed
    if (updates.status !== undefined && updates.status !== channel.status) {
      this.addToStatusIndex(updatedChannel);
    }

    return updatedChannel;
  }

  // ============ Queries ============

  /**
   * Get all channels for a payer
   */
  getChannelsByPayer(payer: string): PaymentChannelState[] {
    const channelIds = this.byPayer.get(payer);
    if (!channelIds) return [];
    return Array.from(channelIds)
      .map((id) => this.channels.get(id)!)
      .filter(Boolean);
  }

  /**
   * Get all channels for a provider
   */
  getChannelsByProvider(provider: string): PaymentChannelState[] {
    const channelIds = this.byProvider.get(provider);
    if (!channelIds) return [];
    return Array.from(channelIds)
      .map((id) => this.channels.get(id)!)
      .filter(Boolean);
  }

  /**
   * Get all channels with a specific status
   */
  getChannelsByStatus(status: PaymentChannelStatus): PaymentChannelState[] {
    const channelIds = this.byStatus.get(status);
    if (!channelIds) return [];
    return Array.from(channelIds)
      .map((id) => this.channels.get(id)!)
      .filter(Boolean);
  }

  /**
   * Get all open channels
   */
  getOpenChannels(): PaymentChannelState[] {
    return this.getChannelsByStatus("open");
  }

  /**
   * Get channels expiring before a given block
   */
  getExpiringChannels(beforeBlock: bigint): PaymentChannelState[] {
    const results: PaymentChannelState[] = [];
    for (const channel of this.channels.values()) {
      if (channel.status === "open" && channel.expiresAt <= beforeBlock) {
        results.push(channel);
      }
    }
    return results;
  }

  /**
   * Get channels that need settlement (open and past settlement frequency)
   */
  getChannelsNeedingSettlement(currentBlock: bigint): PaymentChannelState[] {
    const results: PaymentChannelState[] = [];
    for (const channel of this.channels.values()) {
      if (
        channel.status === "open" &&
        channel.pendingDistributions.length > 0 &&
        currentBlock - channel.lastSettlementBlock >=
          channel.settlementFrequency
      ) {
        results.push(channel);
      }
    }
    return results;
  }

  /**
   * Get channels in dispute that have passed their deadline
   */
  getExpiredDisputes(currentBlock: bigint): PaymentChannelState[] {
    const results: PaymentChannelState[] = [];
    for (const channel of this.channels.values()) {
      if (
        channel.status === "closing" &&
        channel.disputeDeadline &&
        currentBlock > channel.disputeDeadline
      ) {
        results.push(channel);
      }
    }
    return results;
  }

  /**
   * Get open channel between payer and provider (if exists)
   */
  getOpenChannelBetween(
    payer: string,
    provider: string,
  ): PaymentChannelState | undefined {
    const payerChannels = this.byPayer.get(payer);
    if (!payerChannels) return undefined;

    for (const channelId of payerChannels) {
      const channel = this.channels.get(channelId);
      if (
        channel &&
        channel.provider === provider &&
        channel.status === "open"
      ) {
        return channel;
      }
    }
    return undefined;
  }

  // ============ Statistics ============

  /**
   * Get aggregate statistics about payment channels
   */
  getChannelStats(): {
    totalChannels: number;
    openChannels: number;
    closingChannels: number;
    closedChannels: number;
    totalDeposited: bigint;
    totalSpent: bigint;
    totalAvailable: bigint;
    pendingSettlements: number;
  } {
    let totalDeposited = 0n;
    let totalSpent = 0n;
    let totalAvailable = 0n;
    let pendingSettlements = 0;

    for (const channel of this.channels.values()) {
      totalDeposited += channel.deposited;
      totalSpent += channel.spent;
      totalAvailable += channel.available;
      if (channel.pendingDistributions.length > 0) {
        pendingSettlements++;
      }
    }

    return {
      totalChannels: this.channels.size,
      openChannels: this.byStatus.get("open")?.size ?? 0,
      closingChannels: this.byStatus.get("closing")?.size ?? 0,
      closedChannels: this.byStatus.get("closed")?.size ?? 0,
      totalDeposited,
      totalSpent,
      totalAvailable,
      pendingSettlements,
    };
  }

  // ============ Index Management ============

  private addToIndexes(channel: PaymentChannelState): void {
    // Payer index
    let payerSet = this.byPayer.get(channel.payer);
    if (!payerSet) {
      payerSet = new Set();
      this.byPayer.set(channel.payer, payerSet);
    }
    payerSet.add(channel.channelId);

    // Provider index
    let providerSet = this.byProvider.get(channel.provider);
    if (!providerSet) {
      providerSet = new Set();
      this.byProvider.set(channel.provider, providerSet);
    }
    providerSet.add(channel.channelId);

    // Status index
    this.addToStatusIndex(channel);

    // Expiration index
    const expirationKey = channel.expiresAt.toString();
    let expirationSet = this.byExpiration.get(expirationKey);
    if (!expirationSet) {
      expirationSet = new Set();
      this.byExpiration.set(expirationKey, expirationSet);
    }
    expirationSet.add(channel.channelId);
  }

  private addToStatusIndex(channel: PaymentChannelState): void {
    const statusSet = this.byStatus.get(channel.status);
    if (statusSet) {
      statusSet.add(channel.channelId);
    }
  }

  private removeFromStatusIndex(channel: PaymentChannelState): void {
    const statusSet = this.byStatus.get(channel.status);
    if (statusSet) {
      statusSet.delete(channel.channelId);
    }
  }

  // ============ Checkpointing ============

  /**
   * Create a checkpoint of all channel state
   */
  checkpoint(): Map<string, PaymentChannelState> {
    return new Map(this.channels);
  }

  /**
   * Restore from a checkpoint
   */
  restore(checkpoint: Map<string, PaymentChannelState>): void {
    this.channels.clear();
    this.byPayer.clear();
    this.byProvider.clear();
    this.byStatus.set("open", new Set());
    this.byStatus.set("closing", new Set());
    this.byStatus.set("disputed", new Set());
    this.byStatus.set("closed", new Set());
    this.byExpiration.clear();

    for (const channel of checkpoint.values()) {
      this.channels.set(channel.channelId, channel);
      this.addToIndexes(channel);
    }
  }

  /**
   * Clear all channels (for testing)
   */
  clear(): void {
    this.channels.clear();
    this.byPayer.clear();
    this.byProvider.clear();
    this.byStatus.set("open", new Set());
    this.byStatus.set("closing", new Set());
    this.byStatus.set("disputed", new Set());
    this.byStatus.set("closed", new Set());
    this.byExpiration.clear();
  }
}
