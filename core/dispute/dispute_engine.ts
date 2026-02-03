/**
 * Dispute Engine
 *
 * Main orchestrator for dispute resolution:
 * - Dispute lifecycle management
 * - State machine transitions
 * - Evidence comparison
 * - Arbiter selection
 * - Settlement execution
 *
 * PP HIERARCHY:
 *   TRUTH: Engine enforces fair, deterministic dispute resolution
 *   UNDERSTANDING: Orchestrates complex multi-phase process
 *   BEAUTY: Clean API hiding implementation complexity
 *   JUSTICE: Ensures all parties treated fairly per rules
 */

import type { ValidatorState } from '../../state/types.js';
import type { Evidence } from '../evidence/types.js';
import {
  PHASE_DURATIONS,
  DISPUTE_ECONOMICS,
  type Dispute,
  type DisputeState,
  type DisputeSettlement,
  type InitiateDisputePayload,
  type CommitEvidencePayload,
  type RevealEvidencePayload,
  type ArbiterVotePayload,
  type EscalateDisputePayload,
  type DomainExpertise,
  type DisputeEvent,
  type DisputeEventType,
  DisputeError,
} from './types.js';
import {
  compareEvidence,
  getAutoResolutionResult,
  validateDisputeEvidence,
  generateEvidenceCommitment,
  verifyEvidenceCommitment,
} from './evidence_comparison.js';
import {
  selectArbiters,
  updateExpertise,
} from './arbitrator_selection.js';
import {
  advancePhase,
  escalateDispute,
  hasPhaseTimedOut,
  handleTimeout,
  determineFinalWinner,
  isTerminal,
  canEnterEvidenceCommit,
  canEnterEvidenceReveal,
  canEnterArbitration,
  canEnterDeliberation,
  canEnterSettlement,
  canEscalate,
} from './state_machine.js';
import { calculateSettlement, calculateDrawSettlement } from './settlement.js';
import {
  canPartyEscalate,
  getEscalationCost,
  buildTierSelectionConfig,
} from './escalation.js';

// ============ ID Generation ============

let disputeCounter = 0;

/**
 * Generate unique dispute ID
 */
export function generateDisputeId(): string {
  disputeCounter++;
  const timestamp = Date.now().toString(36);
  const counter = disputeCounter.toString(36).padStart(4, '0');
  return `dispute_${timestamp}_${counter}`;
}

// ============ Dispute Creation ============

/**
 * Create a new dispute
 */
export function createDispute(
  payload: InitiateDisputePayload,
  currentBlock: bigint
): Dispute {
  // Validate stake
  if (payload.stake < DISPUTE_ECONOMICS.minChallengeStake) {
    throw new DisputeError(
      `Stake too low. Minimum: ${DISPUTE_ECONOMICS.minChallengeStake}`,
      'INSUFFICIENT_STAKE',
      { required: DISPUTE_ECONOMICS.minChallengeStake, provided: payload.stake }
    );
  }

  // Validate initial evidence
  if (payload.initialEvidence.length === 0) {
    throw new DisputeError(
      'At least one piece of initial evidence is required',
      'INVALID_EVIDENCE'
    );
  }

  const now = Date.now();
  const id = generateDisputeId();
  const phaseDeadline = currentBlock + BigInt(PHASE_DURATIONS.initiation);

  const dispute: Dispute = {
    id,
    targetId: payload.targetId,
    targetType: payload.targetType,
    domain: payload.domain,

    challenger: {
      did: payload.challenger,
      stake: payload.stake,
      initialEvidence: payload.initialEvidence,
    },

    defender: {
      did: '', // Set when defender responds
      stake: 0n,
      responseEvidence: undefined,
    },

    state: {
      phase: 'initiation',
      phaseDeadline,
      canEscalate: false,
      escalationTier: 1,
      previousTierResults: [],
    },

    commitments: {},
    arbiters: [],
    votes: new Map(),

    outcome: undefined,
    winner: undefined,
    exposedFalsehood: false,

    initiatedAtBlock: currentBlock,
    resolvedAtBlock: undefined,

    createdAt: now,
    updatedAt: now,
  };

  return dispute;
}

/**
 * Defender responds to dispute
 */
export function defendDispute(
  dispute: Dispute,
  defenderDid: string,
  stake: bigint,
  responseEvidence: string[],
  currentBlock: bigint
): Dispute {
  if (dispute.state.phase !== 'initiation') {
    throw new DisputeError('Dispute is not in initiation phase', 'INVALID_PHASE');
  }

  if (currentBlock > dispute.state.phaseDeadline) {
    throw new DisputeError('Initiation deadline has passed', 'DEADLINE_PASSED');
  }

  if (stake < dispute.challenger.stake) {
    throw new DisputeError(
      'Defender stake must match challenger stake',
      'INSUFFICIENT_STAKE',
      { required: dispute.challenger.stake, provided: stake }
    );
  }

  return {
    ...dispute,
    defender: {
      did: defenderDid,
      stake,
      responseEvidence,
    },
    updatedAt: Date.now(),
  };
}

// ============ Evidence Handling ============

/**
 * Commit evidence for reveal
 */
export function commitEvidence(
  dispute: Dispute,
  payload: CommitEvidencePayload,
  currentBlock: bigint
): Dispute {
  if (dispute.state.phase !== 'evidence_commit') {
    throw new DisputeError('Not in evidence commit phase', 'INVALID_PHASE');
  }

  if (currentBlock > dispute.state.phaseDeadline) {
    throw new DisputeError('Evidence commit deadline passed', 'DEADLINE_PASSED');
  }

  const isChallenger = payload.committer === dispute.challenger.did;
  const isDefender = payload.committer === dispute.defender.did;

  if (!isChallenger && !isDefender) {
    throw new DisputeError('Not a party to this dispute', 'NOT_AUTHORIZED');
  }

  const commitmentKey = isChallenger ? 'challenger' : 'defender';

  if (dispute.commitments[commitmentKey]) {
    throw new DisputeError('Evidence already committed', 'ALREADY_COMMITTED');
  }

  const commitment = {
    committer: payload.committer,
    evidenceHash: payload.evidenceHash,
    committedAtBlock: currentBlock,
    revealed: false,
  };

  return {
    ...dispute,
    commitments: {
      ...dispute.commitments,
      [commitmentKey]: commitment,
    },
    updatedAt: Date.now(),
  };
}

/**
 * Reveal committed evidence
 */
export function revealEvidence(
  dispute: Dispute,
  payload: RevealEvidencePayload,
  evidenceItems: Evidence[],
  currentBlock: bigint
): Dispute {
  if (dispute.state.phase !== 'evidence_reveal') {
    throw new DisputeError('Not in evidence reveal phase', 'INVALID_PHASE');
  }

  if (currentBlock > dispute.state.phaseDeadline) {
    throw new DisputeError('Evidence reveal deadline passed', 'DEADLINE_PASSED');
  }

  const isChallenger = payload.revealer === dispute.challenger.did;
  const isDefender = payload.revealer === dispute.defender.did;

  if (!isChallenger && !isDefender) {
    throw new DisputeError('Not a party to this dispute', 'NOT_AUTHORIZED');
  }

  const commitmentKey = isChallenger ? 'challenger' : 'defender';
  const commitment = dispute.commitments[commitmentKey];

  if (!commitment) {
    throw new DisputeError('No evidence commitment found', 'COMMITMENT_MISMATCH');
  }

  if (commitment.revealed) {
    throw new DisputeError('Evidence already revealed', 'ALREADY_COMMITTED');
  }

  // Verify commitment matches
  const computedHash = generateEvidenceCommitment(payload.evidenceIds, payload.nonce);
  if (!verifyEvidenceCommitment(payload.evidenceIds, payload.nonce, commitment.evidenceHash)) {
    throw new DisputeError('Evidence does not match commitment', 'COMMITMENT_MISMATCH', {
      expected: commitment.evidenceHash,
      computed: computedHash,
    });
  }

  // Validate evidence
  const validation = validateDisputeEvidence(evidenceItems);
  if (!validation.valid) {
    throw new DisputeError(
      `Invalid evidence: ${validation.errors.join(', ')}`,
      'INVALID_EVIDENCE'
    );
  }

  return {
    ...dispute,
    commitments: {
      ...dispute.commitments,
      [commitmentKey]: {
        ...commitment,
        revealed: true,
        evidence: evidenceItems,
      },
    },
    updatedAt: Date.now(),
  };
}

// ============ Arbitration ============

/**
 * Select arbiters for current tier
 */
export function selectDisputeArbiters(
  dispute: Dispute,
  validators: ValidatorState[],
  expertiseMap: Map<string, DomainExpertise[]>,
  blockHash: string,
  currentBlock: bigint
): Dispute {
  const config = buildTierSelectionConfig(dispute, dispute.state.escalationTier, blockHash);
  const arbiters = selectArbiters(validators, config, expertiseMap, currentBlock);

  return {
    ...dispute,
    arbiters,
    votes: new Map(), // Clear previous votes on new selection
    updatedAt: Date.now(),
  };
}

/**
 * Cast arbiter vote
 */
export function castArbiterVote(
  dispute: Dispute,
  payload: ArbiterVotePayload,
  lockedStake: bigint,
  currentBlock: bigint
): Dispute {
  if (dispute.state.phase !== 'arbitration') {
    throw new DisputeError('Not in arbitration phase', 'INVALID_PHASE');
  }

  if (currentBlock > dispute.state.phaseDeadline) {
    throw new DisputeError('Arbitration deadline passed', 'DEADLINE_PASSED');
  }

  // Check arbiter is selected
  const isArbiter = dispute.arbiters.some(a => a.did === payload.arbiter);
  if (!isArbiter) {
    throw new DisputeError('Not selected as arbiter', 'NOT_ARBITER');
  }

  // Check not already voted
  if (dispute.votes.has(payload.arbiter)) {
    throw new DisputeError('Already voted', 'ALREADY_VOTED');
  }

  const newVotes = new Map(dispute.votes);
  newVotes.set(payload.arbiter, {
    arbiter: payload.arbiter,
    vote: payload.vote,
    justificationHash: payload.justificationHash,
    votedAtBlock: currentBlock,
    lockedStake,
  });

  return {
    ...dispute,
    votes: newVotes,
    updatedAt: Date.now(),
  };
}

// ============ Phase Advancement ============

/**
 * Attempt to advance dispute to next phase
 */
export function tryAdvancePhase(
  dispute: Dispute,
  currentBlock: bigint,
  verifiedFactIds: Set<string>
): { dispute: Dispute; advanced: boolean; event?: DisputeEvent } {
  if (isTerminal(dispute)) {
    return { dispute, advanced: false };
  }

  // Check for timeout first
  if (hasPhaseTimedOut(dispute, currentBlock)) {
    return handlePhaseTimeout(dispute, currentBlock);
  }

  // Check phase-specific advancement conditions
  switch (dispute.state.phase) {
    case 'initiation':
      return tryAdvanceFromInitiation(dispute, currentBlock);

    case 'evidence_commit':
      return tryAdvanceFromEvidenceCommit(dispute, currentBlock);

    case 'evidence_reveal':
      return tryAdvanceFromEvidenceReveal(dispute, currentBlock, verifiedFactIds);

    case 'arbitration':
      return tryAdvanceFromArbitration(dispute, currentBlock);

    case 'deliberation':
      return tryAdvanceFromDeliberation(dispute, currentBlock);

    case 'settlement':
      return tryAdvanceFromSettlement(dispute, currentBlock);

    default:
      return { dispute, advanced: false };
  }
}

function tryAdvanceFromInitiation(
  dispute: Dispute,
  currentBlock: bigint
): { dispute: Dispute; advanced: boolean; event?: DisputeEvent } {
  const check = canEnterEvidenceCommit(dispute);
  if (!check.canEnter) {
    return { dispute, advanced: false };
  }

  const newState = advancePhase(dispute, 'evidence_commit', currentBlock);
  const updated = { ...dispute, state: newState, updatedAt: Date.now() };

  return {
    dispute: updated,
    advanced: true,
    event: createEvent('phase_advanced', dispute.id, currentBlock, {
      from: 'initiation',
      to: 'evidence_commit',
    }),
  };
}

function tryAdvanceFromEvidenceCommit(
  dispute: Dispute,
  currentBlock: bigint
): { dispute: Dispute; advanced: boolean; event?: DisputeEvent } {
  const check = canEnterEvidenceReveal(dispute);
  if (!check.canEnter) {
    return { dispute, advanced: false };
  }

  const newState = advancePhase(dispute, 'evidence_reveal', currentBlock);
  const updated = { ...dispute, state: newState, updatedAt: Date.now() };

  return {
    dispute: updated,
    advanced: true,
    event: createEvent('phase_advanced', dispute.id, currentBlock, {
      from: 'evidence_commit',
      to: 'evidence_reveal',
    }),
  };
}

function tryAdvanceFromEvidenceReveal(
  dispute: Dispute,
  currentBlock: bigint,
  verifiedFactIds: Set<string>
): { dispute: Dispute; advanced: boolean; event?: DisputeEvent } {
  // Check if both revealed
  if (
    !dispute.commitments.challenger?.revealed ||
    !dispute.commitments.defender?.revealed
  ) {
    return { dispute, advanced: false };
  }

  // Compare evidence
  const challengerEvidence = dispute.commitments.challenger.evidence || [];
  const defenderEvidence = dispute.commitments.defender.evidence || [];

  const comparison = compareEvidence(
    challengerEvidence,
    defenderEvidence,
    currentBlock,
    verifiedFactIds
  );

  let updated = {
    ...dispute,
    evidenceComparison: comparison,
    updatedAt: Date.now(),
  };

  // Check for auto-resolution
  const autoResult = getAutoResolutionResult(comparison);
  if (autoResult.canResolve && autoResult.winner) {
    // Skip arbitration, go directly to settlement
    const newState = advancePhase(dispute, 'settlement', currentBlock);
    updated = {
      ...updated,
      state: newState,
      winner: autoResult.winner,
      exposedFalsehood: autoResult.winner === 'challenger',
    };

    return {
      dispute: updated,
      advanced: true,
      event: createEvent('phase_advanced', dispute.id, currentBlock, {
        from: 'evidence_reveal',
        to: 'settlement',
        autoResolved: true,
        winner: autoResult.winner,
      }),
    };
  }

  // Need arbitration
  const check = canEnterArbitration(updated);
  if (!check.canEnter) {
    return { dispute: updated, advanced: false };
  }

  const newState = advancePhase(dispute, 'arbitration', currentBlock);
  updated = { ...updated, state: newState };

  return {
    dispute: updated,
    advanced: true,
    event: createEvent('phase_advanced', dispute.id, currentBlock, {
      from: 'evidence_reveal',
      to: 'arbitration',
    }),
  };
}

function tryAdvanceFromArbitration(
  dispute: Dispute,
  currentBlock: bigint
): { dispute: Dispute; advanced: boolean; event?: DisputeEvent } {
  const check = canEnterDeliberation(dispute);
  if (!check.canEnter) {
    return { dispute, advanced: false };
  }

  const newState = advancePhase(dispute, 'deliberation', currentBlock);
  const updated = { ...dispute, state: newState, updatedAt: Date.now() };

  return {
    dispute: updated,
    advanced: true,
    event: createEvent('phase_advanced', dispute.id, currentBlock, {
      from: 'arbitration',
      to: 'deliberation',
    }),
  };
}

function tryAdvanceFromDeliberation(
  dispute: Dispute,
  currentBlock: bigint
): { dispute: Dispute; advanced: boolean; event?: DisputeEvent } {
  // Determine winner from votes
  const winner = determineFinalWinner(dispute);

  if (winner === 'draw' && canEscalate(dispute).canEscalate) {
    // Draw with escalation possible - wait for escalation or timeout
    return { dispute, advanced: false };
  }

  const check = canEnterSettlement(dispute);
  if (!check.canEnter) {
    return { dispute, advanced: false };
  }

  const newState = advancePhase(dispute, 'settlement', currentBlock);
  const updated = {
    ...dispute,
    state: newState,
    winner: winner === 'draw' ? undefined : winner,
    exposedFalsehood: winner === 'challenger',
    updatedAt: Date.now(),
  };

  return {
    dispute: updated,
    advanced: true,
    event: createEvent('phase_advanced', dispute.id, currentBlock, {
      from: 'deliberation',
      to: 'settlement',
      winner,
    }),
  };
}

function tryAdvanceFromSettlement(
  dispute: Dispute,
  currentBlock: bigint
): { dispute: Dispute; advanced: boolean; event?: DisputeEvent } {
  const newState = advancePhase(dispute, 'resolved', currentBlock);
  const updated = {
    ...dispute,
    state: newState,
    resolvedAtBlock: currentBlock,
    updatedAt: Date.now(),
  };

  return {
    dispute: updated,
    advanced: true,
    event: createEvent('dispute_resolved', dispute.id, currentBlock, {
      winner: dispute.winner,
      tier: dispute.state.escalationTier,
    }),
  };
}

// ============ Timeout Handling ============

function handlePhaseTimeout(
  dispute: Dispute,
  currentBlock: bigint
): { dispute: Dispute; advanced: boolean; event?: DisputeEvent } {
  const timeoutResult = handleTimeout(dispute, currentBlock);

  switch (timeoutResult.action) {
    case 'default_judgment': {
      const newState = advancePhase(dispute, 'settlement', currentBlock);
      const updated = {
        ...dispute,
        state: newState,
        winner: timeoutResult.winner,
        outcome: 'timeout' as const,
        updatedAt: Date.now(),
      };

      return {
        dispute: updated,
        advanced: true,
        event: createEvent('phase_advanced', dispute.id, currentBlock, {
          reason: timeoutResult.reason,
          winner: timeoutResult.winner,
          timeout: true,
        }),
      };
    }

    case 'advance': {
      const newState = advancePhase(dispute, timeoutResult.nextPhase!, currentBlock);
      const updated = { ...dispute, state: newState, updatedAt: Date.now() };

      return {
        dispute: updated,
        advanced: true,
        event: createEvent('phase_advanced', dispute.id, currentBlock, {
          from: dispute.state.phase,
          to: timeoutResult.nextPhase,
          timeout: true,
        }),
      };
    }

    case 'forfeit': {
      const newState: DisputeState = {
        ...dispute.state,
        phase: 'resolved',
        phaseDeadline: currentBlock,
      };
      const updated = {
        ...dispute,
        state: newState,
        outcome: 'timeout' as const,
        resolvedAtBlock: currentBlock,
        updatedAt: Date.now(),
      };

      return {
        dispute: updated,
        advanced: true,
        event: createEvent('dispute_resolved', dispute.id, currentBlock, {
          reason: timeoutResult.reason,
          forfeit: true,
        }),
      };
    }

    default:
      return { dispute, advanced: false };
  }
}

// ============ Escalation ============

/**
 * Process escalation request
 */
export function processEscalation(
  dispute: Dispute,
  payload: EscalateDisputePayload,
  currentBlock: bigint
): { dispute: Dispute; event?: DisputeEvent } {
  const check = canPartyEscalate(dispute, payload.escalator, currentBlock);
  if (!check.canEscalate) {
    throw new DisputeError(check.reason!, 'CANNOT_ESCALATE');
  }

  // Verify sufficient additional stake
  const requiredStake = getEscalationCost(dispute, payload.escalator);
  if (payload.additionalStake < requiredStake) {
    throw new DisputeError(
      `Insufficient escalation stake. Required: ${requiredStake}`,
      'INSUFFICIENT_STAKE',
      { required: requiredStake, provided: payload.additionalStake }
    );
  }

  const newState = escalateDispute(dispute, currentBlock);

  // Update party stake
  const isChallenger = payload.escalator === dispute.challenger.did;
  const updated: Dispute = {
    ...dispute,
    state: newState,
    challenger: isChallenger
      ? { ...dispute.challenger, stake: dispute.challenger.stake + payload.additionalStake }
      : dispute.challenger,
    defender: !isChallenger
      ? { ...dispute.defender, stake: dispute.defender.stake + payload.additionalStake }
      : dispute.defender,
    arbiters: [], // Clear arbiters for new tier
    votes: new Map(), // Clear votes for new tier
    updatedAt: Date.now(),
  };

  return {
    dispute: updated,
    event: createEvent('dispute_escalated', dispute.id, currentBlock, {
      from: dispute.state.escalationTier,
      to: newState.escalationTier,
      escalator: payload.escalator,
    }),
  };
}

// ============ Settlement ============

/**
 * Calculate and return settlement for resolved dispute
 */
export function getDisputeSettlement(
  dispute: Dispute
): DisputeSettlement | null {
  if (dispute.state.phase !== 'settlement' && dispute.state.phase !== 'resolved') {
    return null;
  }

  if (!dispute.winner) {
    // Draw - use draw settlement
    return null;
  }

  return calculateSettlement(dispute);
}

/**
 * Get draw settlement
 */
export function getDrawSettlement(
  dispute: Dispute
): ReturnType<typeof calculateDrawSettlement> | null {
  if (dispute.winner) {
    return null; // Has a winner, not a draw
  }

  return calculateDrawSettlement(dispute);
}

// ============ Expertise Update ============

/**
 * Update arbiter expertise after resolution
 */
export function updateArbiterExpertise(
  dispute: Dispute,
  expertiseMap: Map<string, DomainExpertise[]>,
  currentBlock: bigint
): Map<string, DomainExpertise[]> {
  if (!dispute.winner) {
    return expertiseMap; // No expertise update on draw
  }

  const newMap = new Map(expertiseMap);

  for (const [arbiterDid, vote] of dispute.votes) {
    const wasCorrect = vote.vote === dispute.winner;
    const existing = newMap.get(arbiterDid) || [];

    const domainExpertise = existing.find(e => e.domain === dispute.domain);
    const updated = updateExpertise(domainExpertise || null, dispute.domain, wasCorrect, currentBlock);

    if (domainExpertise) {
      // Update existing
      const newExpertise = existing.map(e =>
        e.domain === dispute.domain ? updated : e
      );
      newMap.set(arbiterDid, newExpertise);
    } else {
      // Add new
      newMap.set(arbiterDid, [...existing, updated]);
    }
  }

  return newMap;
}

// ============ Event Creation ============

function createEvent(
  type: DisputeEventType,
  disputeId: string,
  blockNumber: bigint,
  data: Record<string, unknown>
): DisputeEvent {
  return {
    type,
    disputeId,
    blockNumber,
    timestamp: Date.now(),
    data,
  };
}
