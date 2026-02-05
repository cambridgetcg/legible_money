/**
 * Dispute State Machine
 *
 * 8-phase state machine for dispute resolution:
 *
 * initiation → evidence_commit → evidence_reveal → arbitration
 *                                                      ↓
 *           resolved ← settlement ← deliberation ←─────┘
 *              ↑                        ↓
 *              └────── escalated ←──────┘
 *
 * Each phase has:
 * - Entry conditions
 * - Valid actions
 * - Deadline
 * - Exit transitions
 *
 * PP HIERARCHY:
 *   TRUTH: State transitions must be deterministic and verifiable
 *   UNDERSTANDING: Clear phase progression with defined actions
 *   BEAUTY: Elegant state machine with natural flow
 *   JUSTICE: Deadlines enforce fairness, timeouts punish bad faith
 */

import {
  PHASE_DURATIONS,
  ESCALATION_TIERS,
  type Dispute,
  type DisputePhase,
  type DisputeState,
  type EscalationTier,
  type TierResult,
  DisputeError,
} from "./types.js";

// ============ Phase Transitions ============

/**
 * Valid transitions between phases
 */
export const VALID_TRANSITIONS: Record<DisputePhase, DisputePhase[]> = {
  initiation: ["evidence_commit"],
  evidence_commit: ["evidence_reveal"],
  evidence_reveal: ["arbitration", "settlement"], // Can auto-settle if clear
  arbitration: ["deliberation"],
  deliberation: ["settlement", "escalated"],
  settlement: ["resolved"],
  escalated: ["evidence_commit"], // Restart evidence phase for new tier
  resolved: [], // Terminal state
};

/**
 * Check if transition is valid
 */
export function isValidTransition(
  from: DisputePhase,
  to: DisputePhase,
): boolean {
  return VALID_TRANSITIONS[from].includes(to);
}

// ============ Phase Entry Conditions ============

/**
 * Check if dispute can enter evidence_commit phase
 */
export function canEnterEvidenceCommit(dispute: Dispute): {
  canEnter: boolean;
  reason?: string;
} {
  // Must be in initiation or escalated
  if (
    dispute.state.phase !== "initiation" &&
    dispute.state.phase !== "escalated"
  ) {
    return {
      canEnter: false,
      reason: "Must be in initiation or escalated phase",
    };
  }

  // Defender must have responded (matched stake)
  if (dispute.defender.stake < dispute.challenger.stake) {
    return { canEnter: false, reason: "Defender has not matched stake" };
  }

  return { canEnter: true };
}

/**
 * Check if dispute can enter evidence_reveal phase
 */
export function canEnterEvidenceReveal(dispute: Dispute): {
  canEnter: boolean;
  reason?: string;
} {
  if (dispute.state.phase !== "evidence_commit") {
    return { canEnter: false, reason: "Must be in evidence_commit phase" };
  }

  // Both parties must have committed
  if (!dispute.commitments.challenger) {
    return { canEnter: false, reason: "Challenger has not committed evidence" };
  }

  if (!dispute.commitments.defender) {
    return { canEnter: false, reason: "Defender has not committed evidence" };
  }

  return { canEnter: true };
}

/**
 * Check if dispute can enter arbitration phase
 */
export function canEnterArbitration(dispute: Dispute): {
  canEnter: boolean;
  reason?: string;
} {
  if (dispute.state.phase !== "evidence_reveal") {
    return { canEnter: false, reason: "Must be in evidence_reveal phase" };
  }

  // Both parties must have revealed
  if (!dispute.commitments.challenger?.revealed) {
    return { canEnter: false, reason: "Challenger has not revealed evidence" };
  }

  if (!dispute.commitments.defender?.revealed) {
    return { canEnter: false, reason: "Defender has not revealed evidence" };
  }

  // Must have arbiters selected
  if (dispute.arbiters.length === 0) {
    return { canEnter: false, reason: "Arbiters have not been selected" };
  }

  return { canEnter: true };
}

/**
 * Check if dispute can enter deliberation phase
 */
export function canEnterDeliberation(dispute: Dispute): {
  canEnter: boolean;
  reason?: string;
} {
  if (dispute.state.phase !== "arbitration") {
    return { canEnter: false, reason: "Must be in arbitration phase" };
  }

  // Check if quorum is reached
  const config = ESCALATION_TIERS[dispute.state.escalationTier];
  const totalVotes = dispute.votes.size;
  const quorumNeeded = Math.ceil(
    config.arbiterCount * config.quorumRequirement,
  );

  if (totalVotes < quorumNeeded) {
    return {
      canEnter: false,
      reason: `Quorum not reached: ${totalVotes}/${quorumNeeded} votes`,
    };
  }

  return { canEnter: true };
}

/**
 * Check if dispute can enter settlement phase
 */
export function canEnterSettlement(dispute: Dispute): {
  canEnter: boolean;
  reason?: string;
} {
  // Can enter from evidence_reveal (auto-settle) or deliberation
  if (
    dispute.state.phase !== "evidence_reveal" &&
    dispute.state.phase !== "deliberation"
  ) {
    return {
      canEnter: false,
      reason: "Must be in evidence_reveal or deliberation phase",
    };
  }

  // If from deliberation, must have determined winner
  if (dispute.state.phase === "deliberation") {
    const tierResult = computeTierResult(dispute);
    if (tierResult.winner === "draw") {
      return { canEnter: false, reason: "No clear winner from deliberation" };
    }
  }

  return { canEnter: true };
}

/**
 * Check if dispute can be escalated
 */
export function canEscalate(dispute: Dispute): {
  canEscalate: boolean;
  reason?: string;
} {
  if (dispute.state.phase !== "deliberation") {
    return {
      canEscalate: false,
      reason: "Can only escalate from deliberation",
    };
  }

  if (dispute.state.escalationTier >= 3) {
    return { canEscalate: false, reason: "Already at maximum escalation tier" };
  }

  // Check if within appeal window
  const config = ESCALATION_TIERS[dispute.state.escalationTier];
  if (config.appealWindow === 0) {
    return { canEscalate: false, reason: "This tier does not allow appeals" };
  }

  return { canEscalate: true };
}

// ============ Phase Actions ============

/**
 * Advance dispute to next phase
 */
export function advancePhase(
  dispute: Dispute,
  targetPhase: DisputePhase,
  currentBlock: bigint,
): DisputeState {
  if (!isValidTransition(dispute.state.phase, targetPhase)) {
    throw new DisputeError(
      `Invalid transition: ${dispute.state.phase} → ${targetPhase}`,
      "INVALID_PHASE",
      { from: dispute.state.phase, to: targetPhase },
    );
  }

  // Calculate deadline for new phase
  const duration =
    PHASE_DURATIONS[targetPhase as keyof typeof PHASE_DURATIONS] || 0;
  const deadline = currentBlock + BigInt(duration);

  return {
    phase: targetPhase,
    phaseDeadline: deadline,
    canEscalate:
      targetPhase === "deliberation" && dispute.state.escalationTier < 3,
    escalationTier: dispute.state.escalationTier,
    previousTierResults: dispute.state.previousTierResults,
  };
}

/**
 * Escalate dispute to next tier
 */
export function escalateDispute(
  dispute: Dispute,
  currentBlock: bigint,
): DisputeState {
  const check = canEscalate(dispute);
  if (!check.canEscalate) {
    throw new DisputeError(check.reason!, "CANNOT_ESCALATE");
  }

  // Record current tier result
  const currentTierResult = computeTierResult(dispute);
  const previousResults = [
    ...(dispute.state.previousTierResults || []),
    currentTierResult,
  ];

  const newTier = (dispute.state.escalationTier + 1) as EscalationTier;
  const duration = PHASE_DURATIONS.evidence_commit;

  return {
    phase: "escalated",
    phaseDeadline: currentBlock + BigInt(duration),
    canEscalate: newTier < 3,
    escalationTier: newTier,
    previousTierResults: previousResults,
  };
}

// ============ Deadline Handling ============

/**
 * Check if current phase has timed out
 */
export function hasPhaseTimedOut(
  dispute: Dispute,
  currentBlock: bigint,
): boolean {
  return currentBlock > dispute.state.phaseDeadline;
}

/**
 * Handle phase timeout
 * Returns the action to take based on which phase timed out
 * GAP-016 FIX: Now includes arbiter slashing information
 */
export function handleTimeout(
  dispute: Dispute,
  currentBlock: bigint,
): {
  action: "advance" | "forfeit" | "default_judgment";
  winner?: "challenger" | "defender";
  nextPhase?: DisputePhase;
  reason: string;
  /** GAP-016 FIX: Arbiters who failed to vote and should be slashed */
  arbiterSlashes?: Array<{ did: string; reason: string }>;
} {
  if (!hasPhaseTimedOut(dispute, currentBlock)) {
    throw new DisputeError("Phase has not timed out", "DEADLINE_NOT_REACHED");
  }

  switch (dispute.state.phase) {
    case "initiation":
      // Defender didn't respond - challenger wins by default
      return {
        action: "default_judgment",
        winner: "challenger",
        reason: "Defender failed to respond within deadline",
      };

    case "evidence_commit":
      // Check who didn't commit
      if (!dispute.commitments.challenger && !dispute.commitments.defender) {
        // Both failed - draw, return stakes minus penalty
        return {
          action: "forfeit",
          reason: "Both parties failed to commit evidence",
        };
      }
      if (!dispute.commitments.challenger) {
        return {
          action: "default_judgment",
          winner: "defender",
          reason: "Challenger failed to commit evidence",
        };
      }
      if (!dispute.commitments.defender) {
        return {
          action: "default_judgment",
          winner: "challenger",
          reason: "Defender failed to commit evidence",
        };
      }
      // Both committed, advance
      return {
        action: "advance",
        nextPhase: "evidence_reveal",
        reason: "Evidence commit phase completed",
      };

    case "evidence_reveal":
      // Check who didn't reveal
      if (!dispute.commitments.challenger?.revealed) {
        return {
          action: "default_judgment",
          winner: "defender",
          reason: "Challenger refused to reveal evidence (100% slash)",
        };
      }
      if (!dispute.commitments.defender?.revealed) {
        return {
          action: "default_judgment",
          winner: "challenger",
          reason: "Defender refused to reveal evidence (100% slash)",
        };
      }
      // Both revealed, advance
      return {
        action: "advance",
        nextPhase: "arbitration",
        reason: "Evidence reveal phase completed",
      };

    case "arbitration": {
      // Check if enough votes
      const config = ESCALATION_TIERS[dispute.state.escalationTier];
      const quorumNeeded = Math.ceil(
        config.arbiterCount * config.quorumRequirement,
      );

      // GAP-016 FIX: Identify non-voting arbiters for slashing
      const votedArbiters = new Set(dispute.votes.keys());
      const nonVotingArbiters = dispute.arbiters
        .filter((a) => !votedArbiters.has(a.did))
        .map((a) => ({
          did: a.did,
          reason: "Failed to vote within arbitration deadline",
        }));

      if (dispute.votes.size >= quorumNeeded) {
        return {
          action: "advance",
          nextPhase: "deliberation",
          reason: "Quorum reached, moving to deliberation",
          // GAP-016 FIX: Still slash non-voters even if quorum reached
          arbiterSlashes:
            nonVotingArbiters.length > 0 ? nonVotingArbiters : undefined,
        };
      }
      // Not enough votes - use what we have or escalate
      if (dispute.votes.size > 0) {
        return {
          action: "advance",
          nextPhase: "deliberation",
          reason: "Timeout with partial quorum",
          // GAP-016 FIX: Slash all non-voters
          arbiterSlashes: nonVotingArbiters,
        };
      }
      // No votes at all - this shouldn't happen, slash all arbiters
      return {
        action: "forfeit",
        reason: "No arbiters voted - system error",
        // GAP-016 FIX: Slash all arbiters who failed to vote
        arbiterSlashes: nonVotingArbiters,
      };
    }

    case "deliberation":
      // Timeout without escalation - proceed to settlement
      return {
        action: "advance",
        nextPhase: "settlement",
        reason: "Deliberation period ended",
      };

    case "settlement":
      // Settlement should complete - advance to resolved
      return {
        action: "advance",
        nextPhase: "resolved",
        reason: "Settlement completed",
      };

    default:
      return {
        action: "forfeit",
        reason: `Unexpected timeout in phase: ${dispute.state.phase}`,
      };
  }
}

// ============ Vote Counting ============

/**
 * Count votes for each side
 */
export function countVotes(dispute: Dispute): {
  challenger: number;
  defender: number;
  abstain: number;
} {
  let challenger = 0;
  let defender = 0;
  let abstain = 0;

  for (const vote of dispute.votes.values()) {
    switch (vote.vote) {
      case "challenger":
        challenger++;
        break;
      case "defender":
        defender++;
        break;
      case "abstain":
        abstain++;
        break;
    }
  }

  return { challenger, defender, abstain };
}

/**
 * Compute result for current tier
 */
export function computeTierResult(dispute: Dispute): TierResult {
  const votes = countVotes(dispute);
  const config = ESCALATION_TIERS[dispute.state.escalationTier];

  // Determine winner based on quorum and majority
  let winner: "challenger" | "defender" | "draw";

  const totalNonAbstain = votes.challenger + votes.defender;
  const quorumNeeded = Math.ceil(
    config.arbiterCount * config.quorumRequirement,
  );

  if (totalNonAbstain < quorumNeeded) {
    // Quorum not reached - draw
    winner = "draw";
  } else {
    // Check for supermajority based on tier
    const challengerRatio = votes.challenger / totalNonAbstain;
    const defenderRatio = votes.defender / totalNonAbstain;

    if (challengerRatio >= config.quorumRequirement) {
      winner = "challenger";
    } else if (defenderRatio >= config.quorumRequirement) {
      winner = "defender";
    } else {
      // Neither has supermajority - simple majority decides
      if (votes.challenger > votes.defender) {
        winner = "challenger";
      } else if (votes.defender > votes.challenger) {
        winner = "defender";
      } else {
        winner = "draw";
      }
    }
  }

  return {
    tier: dispute.state.escalationTier,
    winner,
    challengerVotes: votes.challenger,
    defenderVotes: votes.defender,
    abstainVotes: votes.abstain,
    resolvedAtBlock: dispute.state.phaseDeadline,
    wasAppealed: false, // Set later if escalated
  };
}

/**
 * Determine final winner considering all tier results
 */
export function determineFinalWinner(
  dispute: Dispute,
): "challenger" | "defender" | "draw" {
  // Current tier result takes precedence
  const currentResult = computeTierResult(dispute);

  if (currentResult.winner !== "draw") {
    return currentResult.winner;
  }

  // If draw, check previous tiers (most recent first)
  const previousResults = dispute.state.previousTierResults || [];
  for (let i = previousResults.length - 1; i >= 0; i--) {
    if (previousResults[i].winner !== "draw") {
      return previousResults[i].winner;
    }
  }

  // All tiers resulted in draw
  return "draw";
}

// ============ State Queries ============

/**
 * Get time remaining in current phase (in blocks)
 */
export function getTimeRemaining(
  dispute: Dispute,
  currentBlock: bigint,
): bigint {
  if (currentBlock >= dispute.state.phaseDeadline) {
    return 0n;
  }
  return dispute.state.phaseDeadline - currentBlock;
}

/**
 * Check if dispute is in a terminal state
 */
export function isTerminal(dispute: Dispute): boolean {
  return dispute.state.phase === "resolved";
}

/**
 * Check if dispute is active (not resolved)
 */
export function isActive(dispute: Dispute): boolean {
  return !isTerminal(dispute);
}

/**
 * Get current phase display name
 */
export function getPhaseDisplayName(phase: DisputePhase): string {
  const names: Record<DisputePhase, string> = {
    initiation: "Dispute Initiated",
    evidence_commit: "Evidence Commitment",
    evidence_reveal: "Evidence Reveal",
    arbitration: "Arbitration Voting",
    deliberation: "Deliberation",
    settlement: "Settlement",
    escalated: "Escalated",
    resolved: "Resolved",
  };
  return names[phase];
}
