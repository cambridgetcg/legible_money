/**
 * Event Bus Contracts
 *
 * FLOW-2: Event-Driven Communication
 * Cross-module communication MUST use events, not direct calls
 * (except for synchronous queries).
 *
 * Domain modules emit events; other modules subscribe.
 * This decouples modules and enables async processing.
 */

import type {
  DID,
  PublicKey,
  BlockHeight,
  Amount,
  Confidence,
} from "../types/index.js";

// ============ Event Types ============

/**
 * All protocol event types
 */
export type ProtocolEventType =
  // Block events
  | "block_proposed"
  | "block_finalized"
  | "block_rejected"

  // Consensus events
  | "round_started"
  | "round_completed"
  | "epoch_changed"

  // Knowledge events
  | "claim_submitted"
  | "fact_verified"
  | "fact_challenged"
  | "fact_revoked"
  | "confidence_updated"

  // Validator events
  | "validator_registered"
  | "validator_slashed"
  | "validator_deactivated"
  | "stake_changed"

  // Dispute events
  | "dispute_initiated"
  | "dispute_escalated"
  | "dispute_resolved"

  // Rewards events
  | "reward_distributed"
  | "vesting_created"
  | "vesting_claimed"
  | "clawback_executed"

  // Network events
  | "peer_connected"
  | "peer_disconnected"
  | "sync_started"
  | "sync_completed";

// ============ Event Payloads ============

/**
 * Base event structure
 */
export interface ProtocolEvent<
  T extends ProtocolEventType = ProtocolEventType,
> {
  type: T;
  timestamp: number;
  blockHeight?: BlockHeight;
  source: string; // Module that emitted the event
  correlationId?: string; // For tracing related events
  payload: EventPayload<T>;
}

/**
 * Map event types to their payloads
 */
export type EventPayload<T extends ProtocolEventType> =
  T extends "block_proposed"
    ? BlockProposedPayload
    : T extends "block_finalized"
      ? BlockFinalizedPayload
      : T extends "round_started"
        ? RoundStartedPayload
        : T extends "round_completed"
          ? RoundCompletedPayload
          : T extends "epoch_changed"
            ? EpochChangedPayload
            : T extends "claim_submitted"
              ? ClaimSubmittedPayload
              : T extends "fact_verified"
                ? FactVerifiedPayload
                : T extends "fact_challenged"
                  ? FactChallengedPayload
                  : T extends "fact_revoked"
                    ? FactRevokedPayload
                    : T extends "confidence_updated"
                      ? ConfidenceUpdatedPayload
                      : T extends "validator_slashed"
                        ? ValidatorSlashedPayload
                        : T extends "dispute_initiated"
                          ? DisputeInitiatedPayload
                          : T extends "dispute_resolved"
                            ? DisputeResolvedPayload
                            : T extends "reward_distributed"
                              ? RewardDistributedPayload
                              : T extends "vesting_created"
                                ? VestingCreatedPayload
                                : T extends "clawback_executed"
                                  ? ClawbackExecutedPayload
                                  : Record<string, unknown>;

// ============ Payload Definitions ============

export interface BlockProposedPayload {
  blockNumber: BlockHeight;
  blockHash: string;
  proposer: PublicKey;
  transactionCount: number;
}

export interface BlockFinalizedPayload {
  blockNumber: BlockHeight;
  blockHash: string;
  finalizedAt: number;
}

export interface RoundStartedPayload {
  roundId: string;
  claimId: string;
  selectedValidators: PublicKey[];
  commitDeadline: BlockHeight;
}

export interface RoundCompletedPayload {
  roundId: string;
  claimId: string;
  outcome: "accepted" | "rejected" | "contested" | "insufficient";
  finalConfidence: Confidence;
  participationRate: number;
}

export interface EpochChangedPayload {
  oldEpoch: bigint;
  newEpoch: bigint;
  validatorSetChanged: boolean;
  totalRewardsDistributed: Amount;
}

export interface ClaimSubmittedPayload {
  claimId: string;
  submitter: DID;
  claim: string;
  domain: string;
  stake: Amount;
}

export interface FactVerifiedPayload {
  factId: string;
  claimId: string;
  confidence: Confidence;
  verifierCount: number;
  submitter: DID;
}

export interface FactChallengedPayload {
  factId: string;
  challengeId: string;
  challenger: DID;
  reason: string;
  stake: Amount;
}

export interface FactRevokedPayload {
  factId: string;
  reason: string;
  revokedBy: "challenge" | "governance" | "expiry";
}

export interface ConfidenceUpdatedPayload {
  factId: string;
  oldConfidence: Confidence;
  newConfidence: Confidence;
  reason: string;
}

export interface ValidatorSlashedPayload {
  validator: PublicKey;
  amount: Amount;
  reason: string;
  newStake: Amount;
}

export interface DisputeInitiatedPayload {
  disputeId: string;
  targetId: string;
  targetType: "evidence" | "fact" | "claim";
  challenger: DID;
  stake: Amount;
}

export interface DisputeResolvedPayload {
  disputeId: string;
  outcome: "challenger_wins" | "defender_wins" | "draw" | "timeout";
  winnerReward: Amount;
  loserPenalty: Amount;
}

export interface RewardDistributedPayload {
  recipient: DID;
  amount: Amount;
  reason:
    | "block_production"
    | "verification"
    | "claim_submission"
    | "citation"
    | "dispute_win";
  sourceId?: string; // Block hash, claim ID, etc.
}

export interface VestingCreatedPayload {
  vestingId: string;
  recipient: DID;
  totalAmount: Amount;
  cliffEndBlock: BlockHeight;
  fullVestBlock: BlockHeight;
  claimId: string;
}

export interface ClawbackExecutedPayload {
  vestingId: string;
  recipient: DID;
  clawedBackAmount: Amount;
  reason: "falsification" | "dispute_loss";
  disputeId?: string;
}

// ============ Event Bus Interface ============

/**
 * Event handler function type
 */
export type EventHandler<T extends ProtocolEventType> = (
  event: ProtocolEvent<T>,
) => void | Promise<void>;

/**
 * Subscription handle for unsubscribing
 */
export interface Subscription {
  id: string;
  eventType: ProtocolEventType;
  unsubscribe(): void;
}

/**
 * Event bus for cross-module communication
 */
export interface EventBus {
  /**
   * Emit an event
   */
  emit<T extends ProtocolEventType>(event: ProtocolEvent<T>): void;

  /**
   * Subscribe to an event type
   */
  on<T extends ProtocolEventType>(
    eventType: T,
    handler: EventHandler<T>,
  ): Subscription;

  /**
   * Subscribe to an event type (one-time)
   */
  once<T extends ProtocolEventType>(
    eventType: T,
    handler: EventHandler<T>,
  ): Subscription;

  /**
   * Subscribe to multiple event types
   */
  onMany(
    eventTypes: ProtocolEventType[],
    handler: EventHandler<ProtocolEventType>,
  ): Subscription[];

  /**
   * Unsubscribe all handlers for an event type
   */
  off(eventType: ProtocolEventType): void;

  /**
   * Wait for a specific event (returns promise)
   */
  waitFor<T extends ProtocolEventType>(
    eventType: T,
    predicate?: (event: ProtocolEvent<T>) => boolean,
    timeoutMs?: number,
  ): Promise<ProtocolEvent<T>>;

  /**
   * Get recent events (for replay/debugging)
   */
  getRecentEvents(count?: number): ProtocolEvent[];

  /**
   * Clear event history
   */
  clearHistory(): void;
}

/**
 * Event emitter interface (for modules that emit events)
 */
export interface EventEmitter {
  emit<T extends ProtocolEventType>(event: ProtocolEvent<T>): void;
}

/**
 * Event listener interface (for modules that listen to events)
 */
export interface EventListener {
  on<T extends ProtocolEventType>(
    eventType: T,
    handler: EventHandler<T>,
  ): Subscription;
}

// ============ Event Builder Helpers ============

/**
 * Create a protocol event
 */
export function createEvent<T extends ProtocolEventType>(
  type: T,
  source: string,
  payload: EventPayload<T>,
  options?: {
    blockHeight?: BlockHeight;
    correlationId?: string;
  },
): ProtocolEvent<T> {
  return {
    type,
    timestamp: Date.now(),
    blockHeight: options?.blockHeight,
    source,
    correlationId: options?.correlationId,
    payload,
  };
}
