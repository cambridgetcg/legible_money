/**
 * Challenge/Dispute Integration Functions
 *
 * Entry points and routing functions for integrating the dispute
 * resolution system with the Proof of Truth consensus layer.
 *
 * GAP-003 FIX: Provides the missing integration between:
 * - challengeFact() - Entry point for initiating fact challenges
 * - routeDisputeOutcomeToFact() - Routes dispute outcomes to fact confidence
 * - checkActiveDisputes() - Verification round dispute awareness
 *
 * PP HIERARCHY:
 *   TRUTH: Challenges test the truth of accepted facts
 *   UNDERSTANDING: Clear flow from challenge → dispute → outcome → fact update
 *   BEAUTY: Single responsibility functions, clean interfaces
 *   JUSTICE: Outcomes justly affect fact status and confidence
 *   CREATIVITY: Bridges two systems without tight coupling
 */

import type { StateManager } from "../../state/state_manager.js";
import { createDispute } from "../../core/dispute/dispute_engine.js";
import {
  DISPUTE_ECONOMICS,
  PHASE_DURATIONS,
  type EscalationTier,
} from "../../core/dispute/types.js";
import {
  type FactChallengeRequest,
  type FactChallengeResult,
  type DisputeOutcomeRoutingRequest,
  type DisputeOutcomeRoutingResult,
  type ConfidenceAdjustmentInput,
  type ConfidenceAdjustmentResult,
  type ActiveDisputeCheckRequest,
  type ActiveDisputeCheckResult,
  CONFIDENCE_ADJUSTMENT_CONFIG,
  DISPUTE_AWARE_CONFIG,
  isChallengerWin,
  isDefenderWin,
} from "../../core/dispute/challenge_integration.js";

// ============ Challenge Entry Point ============

/**
 * Initiate a challenge against a verified fact
 *
 * This is the primary entry point for disputing knowledge in the tree.
 * Creates a dispute and marks the fact as challenged.
 *
 * @param request - Challenge request details
 * @param state - State manager for querying/updating state
 * @param currentBlock - Current block number
 * @returns Challenge result with dispute ID if accepted
 */
export function challengeFact(
  request: FactChallengeRequest,
  state: StateManager,
  currentBlock: bigint,
): FactChallengeResult {
  // 1. Validate fact exists and is active
  const fact = state.getFact(request.factId);
  if (!fact) {
    return {
      accepted: false,
      rejectionReason: `Fact ${request.factId} not found`,
    };
  }

  if (fact.status !== "active") {
    return {
      accepted: false,
      rejectionReason: `Fact ${request.factId} is not active (status: ${fact.status})`,
    };
  }

  // 2. Validate minimum stake
  if (request.stake < DISPUTE_ECONOMICS.minChallengeStake) {
    return {
      accepted: false,
      rejectionReason: `Minimum challenge stake is ${DISPUTE_ECONOMICS.minChallengeStake}, got ${request.stake}`,
    };
  }

  // 3. Check if fact is already under challenge
  const existingDisputes = checkActiveDisputes(
    {
      targetId: request.factId,
      targetType: "fact",
    },
    state,
  );

  if (
    existingDisputes.hasActiveDisputes &&
    existingDisputes.activeDisputeCount >=
      DISPUTE_AWARE_CONFIG.maxActiveDisputesBeforePause
  ) {
    return {
      accepted: false,
      rejectionReason: `Fact ${request.factId} already has maximum active disputes (${existingDisputes.activeDisputeCount})`,
    };
  }

  // 4. Create the dispute using the proper payload format
  const dispute = createDispute(
    {
      targetId: request.factId,
      targetType: "fact",
      domain: request.domain,
      challenger: request.challenger,
      stake: request.stake,
      initialEvidence: request.initialEvidence,
      reasonHash: request.reasonHash,
    },
    currentBlock,
  );

  // 5. Add dispute to state
  state.addDispute(dispute);

  // 6. Mark fact as challenged
  state.challengeFact(request.factId);

  // 7. Emit challenge event
  state.emitEvent("ChallengeInitiated", {
    factId: request.factId,
    disputeId: dispute.id,
    challenger: request.challenger,
    stake: request.stake.toString(),
    blockNumber: currentBlock.toString(),
  });

  return {
    accepted: true,
    disputeId: dispute.id,
    factMarkedChallenged: true,
    originalConfidence: fact.confidence,
  };
}

// ============ Dispute Outcome Routing ============

/**
 * Route a dispute outcome to update fact confidence
 *
 * Called when a dispute reaches resolution (settlement phase complete).
 * Updates the fact's confidence based on the dispute outcome.
 *
 * @param request - Outcome routing request
 * @param state - State manager for querying/updating state
 * @returns Routing result with new confidence
 */
export function routeDisputeOutcomeToFact(
  request: DisputeOutcomeRoutingRequest,
  state: StateManager,
): DisputeOutcomeRoutingResult {
  const { dispute, settlement: _settlement, settlementBlock } = request;

  // 1. Validate dispute targets a fact
  if (dispute.targetType !== "fact") {
    return {
      success: false,
      error: `Dispute ${dispute.id} does not target a fact (targets: ${dispute.targetType})`,
    };
  }

  // 2. Get the fact
  const fact = state.getFact(dispute.targetId);
  if (!fact) {
    return {
      success: false,
      error: `Fact ${dispute.targetId} not found`,
    };
  }

  // 3. Calculate confidence adjustment
  const adjustmentInput: ConfidenceAdjustmentInput = {
    currentConfidence: fact.confidence,
    outcome: dispute.outcome!,
    finalTier: dispute.state.escalationTier,
    exposedFalsehood: dispute.exposedFalsehood,
    previousChallengeCount: fact.challengeCount,
    stratum: fact.stratum,
  };

  const adjustment = calculateConfidenceAdjustment(adjustmentInput);

  // 4. Determine new fact status
  const newStatus = determineFactStatus(
    dispute.outcome!,
    adjustment.newConfidence,
    CONFIDENCE_ADJUSTMENT_CONFIG.retractionThreshold,
  );

  // 5. Update fact in state
  state.updateFactConfidence(dispute.targetId, adjustment.newConfidence);
  state.updateFactStatus(dispute.targetId, newStatus);
  state.incrementFactChallengeCount(dispute.targetId);

  // 6. If fact is revoked, emit retraction event
  if (newStatus === "revoked") {
    state.emitEvent("FactRetracted", {
      factId: dispute.targetId,
      disputeId: dispute.id,
      finalConfidence: adjustment.newConfidence.toString(),
      retractionReason: adjustment.reason,
      blockNumber: settlementBlock.toString(),
    });
  }

  // 7. Emit confidence change event
  state.emitEvent("FactConfidenceChanged", {
    factId: dispute.targetId,
    previousConfidence: fact.confidence.toString(),
    newConfidence: adjustment.newConfidence.toString(),
    disputeId: dispute.id,
    outcome: dispute.outcome || "unknown",
    blockNumber: settlementBlock.toString(),
  });

  return {
    success: true,
    newConfidence: adjustment.newConfidence,
    confidenceAdjustment: adjustment.adjustmentFactor,
    newStatus,
    factRetracted: newStatus === "revoked",
  };
}

/**
 * Determine fact status based on dispute outcome and confidence
 *
 * Maps to dispute-relevant FactStatus values only.
 * Note: 'expired' is excluded as expiration is time-based, not dispute-based.
 */
function determineFactStatus(
  outcome: string,
  newConfidence: number,
  retractionThreshold: number,
): "active" | "challenged" | "superseded" | "revoked" {
  if (isChallengerWin(outcome as any) && newConfidence <= retractionThreshold) {
    return "revoked";
  }
  if (outcome === "draw" || outcome === "timeout") {
    return "challenged"; // Remains contested
  }
  return "active";
}

// ============ Confidence Adjustment ============

/**
 * Calculate confidence adjustment based on dispute outcome
 *
 * Uses the configured adjustment factors to determine the new
 * confidence value for a fact after a dispute resolution.
 *
 * @param input - Adjustment calculation input
 * @returns Adjustment result with new confidence
 */
export function calculateConfidenceAdjustment(
  input: ConfidenceAdjustmentInput,
): ConfidenceAdjustmentResult {
  const {
    currentConfidence,
    outcome,
    finalTier,
    exposedFalsehood,
    previousChallengeCount,
  } = input;

  let adjustmentFactor: number;
  let reason: string;

  // Calculate base adjustment based on outcome
  if (isChallengerWin(outcome)) {
    // Challenger wins: reduce confidence significantly
    adjustmentFactor = 1 - CONFIDENCE_ADJUSTMENT_CONFIG.challengerWinsReduction;
    reason = "Challenger won dispute - falsehood exposed";

    // Apply additional penalty if falsehood was exposed
    if (exposedFalsehood) {
      adjustmentFactor *= 0.8; // Extra 20% reduction
      reason += " (falsehood confirmation bonus applied)";
    }
  } else if (isDefenderWin(outcome)) {
    // Defender wins: boost confidence (truth defended)
    const boost = Math.min(
      CONFIDENCE_ADJUSTMENT_CONFIG.defenderWinsBoost,
      CONFIDENCE_ADJUSTMENT_CONFIG.maxDefenseBoost,
    );
    adjustmentFactor = 1 + boost;
    reason = "Defender won dispute - truth defended";
  } else if (outcome === "draw") {
    // Draw: moderate confidence reduction (uncertainty)
    adjustmentFactor = 1 - CONFIDENCE_ADJUSTMENT_CONFIG.drawReduction;
    reason = "Dispute ended in draw - truth remains uncertain";
  } else {
    // Timeout: small confidence reduction
    adjustmentFactor = 1 - CONFIDENCE_ADJUSTMENT_CONFIG.timeoutReduction;
    reason = "Dispute timed out - unresolved challenge";
  }

  // Apply escalation tier penalty
  // Higher tiers mean truth required more scrutiny - slight penalty
  const tierPenalty =
    (finalTier - 1) * CONFIDENCE_ADJUSTMENT_CONFIG.escalationTierPenalty;
  adjustmentFactor *= 1 - tierPenalty;

  // Apply challenge history decay
  // Repeated challenges reduce confidence even if defended
  if (previousChallengeCount > 0 && !isDefenderWin(outcome)) {
    const decayFactor = Math.pow(
      CONFIDENCE_ADJUSTMENT_CONFIG.challengeDecayRate,
      previousChallengeCount,
    );
    adjustmentFactor *= decayFactor;
    reason += ` (history decay: ${previousChallengeCount} previous challenges)`;
  }

  // Calculate new confidence
  let newConfidence = currentConfidence * adjustmentFactor;

  // Clamp to valid range [0, 1]
  newConfidence = Math.max(0, Math.min(1, newConfidence));

  // Determine if fact should be retracted
  const shouldRetract =
    newConfidence <= CONFIDENCE_ADJUSTMENT_CONFIG.retractionThreshold &&
    isChallengerWin(outcome);

  return {
    newConfidence,
    adjustmentFactor,
    shouldRetract,
    reason,
  };
}

// ============ Active Dispute Check ============

/**
 * Check for active disputes on a fact or claim
 *
 * Used by verification rounds to determine if verification
 * should proceed or be paused due to ongoing disputes.
 *
 * @param request - Check request
 * @param state - State manager
 * @returns Check result with dispute info
 */
export function checkActiveDisputes(
  request: ActiveDisputeCheckRequest,
  state: StateManager,
): ActiveDisputeCheckResult {
  // Query disputes targeting this fact/claim
  const disputes = state.getDisputesByTarget(
    request.targetId,
    request.targetType,
  );

  // Filter to active disputes (not resolved)
  const activeDisputes = disputes.filter(
    (d) => d.state.phase !== "resolved" && d.state.phase !== "settlement",
  );

  if (activeDisputes.length === 0) {
    return {
      hasActiveDisputes: false,
      activeDisputeCount: 0,
      activeDisputeIds: [],
      highestTier: 1,
      shouldPauseVerification: false,
      recommendedAction: "proceed",
    };
  }

  // Find highest tier among active disputes
  const highestTier = activeDisputes.reduce(
    (max, d) => Math.max(max, d.state.escalationTier) as EscalationTier,
    1 as EscalationTier,
  );

  // Determine if verification should pause
  const shouldPause =
    highestTier >= DISPUTE_AWARE_CONFIG.pauseVerificationAtTier ||
    activeDisputes.length >= DISPUTE_AWARE_CONFIG.maxActiveDisputesBeforePause;

  // Recommend action
  let recommendedAction: "proceed" | "pause" | "wait_for_resolution";
  if (shouldPause) {
    // Check if any dispute is close to resolution
    const closestDeadline = activeDisputes.reduce(
      (min, d) => (d.state.phaseDeadline < min ? d.state.phaseDeadline : min),
      BigInt(Number.MAX_SAFE_INTEGER),
    );
    recommendedAction =
      closestDeadline < BigInt(PHASE_DURATIONS.settlement)
        ? "wait_for_resolution"
        : "pause";
  } else {
    recommendedAction = "proceed";
  }

  return {
    hasActiveDisputes: true,
    activeDisputeCount: activeDisputes.length,
    activeDisputeIds: activeDisputes.map((d) => d.id),
    highestTier,
    shouldPauseVerification: shouldPause,
    recommendedAction,
  };
}

// ============ Verification Round Integration ============

/**
 * Get confidence penalty for ongoing disputes
 *
 * When a fact has active disputes, its effective confidence
 * during verification should be reduced.
 *
 * @param factId - ID of the fact
 * @param baseConfidence - Base confidence value
 * @param state - State manager
 * @returns Adjusted confidence with dispute penalty
 */
export function getDisputeAdjustedConfidence(
  factId: string,
  baseConfidence: number,
  state: StateManager,
): { adjustedConfidence: number; penalty: number; reason: string } {
  if (!DISPUTE_AWARE_CONFIG.penalizeOngoingDisputes) {
    return {
      adjustedConfidence: baseConfidence,
      penalty: 0,
      reason: "Ongoing dispute penalty disabled",
    };
  }

  const disputeCheck = checkActiveDisputes(
    { targetId: factId, targetType: "fact" },
    state,
  );

  if (!disputeCheck.hasActiveDisputes) {
    return {
      adjustedConfidence: baseConfidence,
      penalty: 0,
      reason: "No active disputes",
    };
  }

  // Calculate penalty based on number and tier of disputes
  const basePenalty =
    disputeCheck.activeDisputeCount *
    DISPUTE_AWARE_CONFIG.ongoingDisputePenalty;
  const tierMultiplier = disputeCheck.highestTier; // Higher tier = more significant dispute
  const totalPenalty = basePenalty * tierMultiplier;

  const adjustedConfidence = Math.max(0, baseConfidence - totalPenalty);

  return {
    adjustedConfidence,
    penalty: totalPenalty,
    reason: `${disputeCheck.activeDisputeCount} active dispute(s), highest tier: ${disputeCheck.highestTier}`,
  };
}

// ============ Post-Resolution Processing ============

/**
 * Process fact state after dispute resolution
 *
 * Called after a dispute settles to update the fact and
 * potentially resume verification rounds.
 *
 * @param disputeId - ID of the resolved dispute
 * @param state - State manager
 * @param currentBlock - Current block number
 * @returns Whether processing was successful
 */
export function processPostResolution(
  disputeId: string,
  state: StateManager,
  currentBlock: bigint,
): { success: boolean; error?: string } {
  // Get the dispute
  const dispute = state.getDispute(disputeId);
  if (!dispute) {
    return { success: false, error: `Dispute ${disputeId} not found` };
  }

  if (dispute.state.phase !== "resolved") {
    return { success: false, error: `Dispute ${disputeId} is not resolved` };
  }

  // If this was a fact dispute, check if verification can resume
  if (dispute.targetType === "fact") {
    const remainingDisputes = checkActiveDisputes(
      {
        targetId: dispute.targetId,
        targetType: "fact",
      },
      state,
    );

    if (!remainingDisputes.hasActiveDisputes) {
      // All disputes resolved - fact can return to normal state
      const fact = state.getFact(dispute.targetId);
      if (fact && fact.status === "challenged") {
        // If challenger won, status was already updated by routeDisputeOutcomeToFact
        // If defender won, restore to active
        if (dispute.outcome && isDefenderWin(dispute.outcome)) {
          state.updateFactStatus(dispute.targetId, "active");
        }
      }

      // Emit event
      state.emitEvent("FactDisputesResolved", {
        factId: dispute.targetId,
        finalDisputeId: disputeId,
        outcome: dispute.outcome || "unknown",
        blockNumber: currentBlock.toString(),
      });
    }
  }

  return { success: true };
}

// ============ Exports ============

export {
  type FactChallengeRequest,
  type FactChallengeResult,
  type DisputeOutcomeRoutingRequest,
  type DisputeOutcomeRoutingResult,
  type ConfidenceAdjustmentInput,
  type ConfidenceAdjustmentResult,
  type ActiveDisputeCheckRequest,
  type ActiveDisputeCheckResult,
  CONFIDENCE_ADJUSTMENT_CONFIG,
  DISPUTE_AWARE_CONFIG,
} from "../../core/dispute/challenge_integration.js";
