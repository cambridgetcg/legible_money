/**
 * Loop Event Bridge
 *
 * Converts payment execution results and state transition events into
 * LoopEvents for the LoopController's sustainability tracking.
 *
 * This is the integration layer between:
 *   - Payment distribution (billing) → APICallEvent
 *   - Claim verification (consensus) → ClaimVerifiedEvent
 *   - Block production (consensus)   → BlockProducedEvent
 *   - Bounty lifecycle (state)       → BountyCreatedEvent / BountyFulfilledEvent
 *
 * Flow: Real events → Bridge → LoopController.processEvent() → Sustainability metrics
 */

import type {
  LoopController,
  APICallEvent,
  ClaimVerifiedEvent,
  BlockProducedEvent,
  BountyCreatedEvent,
  BountyFulfilledEvent,
} from "./perpetual_loop.js";
import type { PaymentExecutionResult } from "../billing/payment_executor.js";
import type { PaymentDistribution } from "../billing/types.js";

// ============ Bridge ============

export class LoopEventBridge {
  constructor(private readonly loopController: LoopController) {}

  /**
   * Emit an API call event after a knowledge query payment is executed.
   *
   * Called from the knowledgeQuery RPC handler after executePaymentDistribution().
   */
  onQueryExecuted(
    queryId: string,
    _distribution: PaymentDistribution,
    result: PaymentExecutionResult,
    factIds: string[],
  ): void {
    if (!result.success) return;

    // Build knowledge distribution map from transfers
    const knowledgeDistribution = new Map<string, bigint>();
    for (const transfer of result.transfers) {
      if (
        transfer.reason === "direct_citation" ||
        transfer.reason === "ancestor" ||
        transfer.reason === "cross_domain_bonus" ||
        transfer.reason === "fundamentality_bonus"
      ) {
        const current = knowledgeDistribution.get(transfer.to) ?? 0n;
        knowledgeDistribution.set(transfer.to, current + transfer.amount);
      }
    }

    const event: APICallEvent = {
      type: "api_call",
      callId: queryId,
      revenue: result.totalTransferred,
      factsUsed: factIds,
      researchShare: result.researchFunded,
      knowledgeDistribution,
    };

    this.loopController.processEvent(event);
  }

  /**
   * Emit a claim verified event after a verification round completes.
   *
   * Called from state transitions when a claim passes or fails verification.
   */
  onClaimVerified(
    claimId: string,
    outcome: "accepted" | "rejected" | "contested",
    factId: string | undefined,
    rewards: Map<string, bigint>,
    researchShare: bigint,
  ): void {
    const event: ClaimVerifiedEvent = {
      type: "claim_verified",
      claimId,
      outcome,
      factId,
      rewards,
      researchShare,
    };

    this.loopController.processEvent(event);
  }

  /**
   * Emit a block produced event after a new block is finalized.
   *
   * Called from block production/epoch transition.
   */
  onBlockProduced(
    blockNumber: bigint,
    blockReward: bigint,
    researchShare: bigint,
    validatorRewards: Map<string, bigint>,
  ): void {
    const event: BlockProducedEvent = {
      type: "block_produced",
      blockNumber,
      blockReward,
      researchShare,
      validatorRewards,
    };

    this.loopController.processEvent(event);
  }

  /**
   * Emit a bounty created event when a new verification bounty is posted.
   */
  onBountyCreated(
    bountyId: string,
    value: bigint,
    targetClaim: string,
    domain: string,
  ): void {
    const event: BountyCreatedEvent = {
      type: "bounty_created",
      bountyId,
      value,
      targetClaim,
      domain,
    };

    this.loopController.processEvent(event);
  }

  /**
   * Emit a bounty fulfilled event when IRL verification is completed.
   */
  onBountyFulfilled(
    bountyId: string,
    fulfiller: string,
    researchId: string,
    outcome: "confirms" | "refutes" | "inconclusive" | "fraud_detected",
  ): void {
    const event: BountyFulfilledEvent = {
      type: "bounty_fulfilled",
      bountyId,
      fulfiller,
      researchId,
      outcome,
    };

    this.loopController.processEvent(event);
  }
}
