/**
 * Partnership Formation
 *
 * Handles the creation and finalization of human-agent partnerships.
 * Formation requires both parties to contribute within 77 blocks.
 */

import type {
  PartnershipAccount,
  PartnershipFormationRequest,
  PartnershipFormationState,
  PartnershipMetrics,
  PartnershipTier,
  CommonPot,
} from "./types.js";
import type { DID } from "../core/agents/identity/did.js";
import type { PublicKey } from "../core/crypto/keys.js";
import { FORMATION, LOCK_TIERS } from "./constants.js";
import { createPool } from "./pool.js";
import { createDefaultSplit, createCustomSplit } from "./rewards.js";
import {
  sha512_256,
  sha512_256Hex,
  domainHash,
} from "../core/crypto/hashing.js";
import { generateThresholdKeyConfig, createCommonPot } from "./threshold.js";

// ============ Formation Initiation ============

/**
 * Initiate partnership formation
 */
export function initiateFormation(
  humanDid: DID,
  agentDid: DID,
  humanOperationalKey: PublicKey,
  agentOperationalKey: PublicKey,
  humanContribution: bigint,
  agentContribution: bigint,
  initiatedBy: "human" | "agent",
  currentBlock: bigint,
  proposedSplit?: { human: number; agent: number },
  proposedLockTier?: PartnershipTier,
): { formation: PartnershipFormationState; error?: string } {
  // Validate contributions
  if (humanContribution < FORMATION.MIN_CONTRIBUTION) {
    return {
      formation: null as any,
      error: `Human contribution ${humanContribution} below minimum ${FORMATION.MIN_CONTRIBUTION}`,
    };
  }

  if (agentContribution < FORMATION.MIN_CONTRIBUTION) {
    return {
      formation: null as any,
      error: `Agent contribution ${agentContribution} below minimum ${FORMATION.MIN_CONTRIBUTION}`,
    };
  }

  // Validate split if provided
  if (proposedSplit) {
    if (proposedSplit.human + proposedSplit.agent !== 100) {
      return { formation: null as any, error: "Split must total 100%" };
    }
    if (
      proposedSplit.human < FORMATION.MIN_SPLIT ||
      proposedSplit.human > FORMATION.MAX_SPLIT
    ) {
      return {
        formation: null as any,
        error: `Human split must be ${FORMATION.MIN_SPLIT}-${FORMATION.MAX_SPLIT}%`,
      };
    }
  }

  const request: PartnershipFormationRequest = {
    humanDid,
    agentDid,
    humanOperationalKey,
    agentOperationalKey,
    humanContribution,
    agentContribution,
    proposedSplit,
    proposedLockTier: proposedLockTier ?? 2, // Default tier 2 (~6.5 days)
    initiatedBy,
    initiatedAt: currentBlock,
  };

  const formation: PartnershipFormationState = {
    request,
    humanAccepted: initiatedBy === "human",
    agentAccepted: initiatedBy === "agent",
    humanDeposited: false,
    agentDeposited: false,
    expiresAt: currentBlock + FORMATION.WINDOW_BLOCKS,
    status: "pending",
  };

  return { formation };
}

/**
 * Accept formation request
 */
export function acceptFormation(
  formation: PartnershipFormationState,
  acceptor: "human" | "agent",
  currentBlock: bigint,
): { formation: PartnershipFormationState; error?: string } {
  // Check not expired
  if (currentBlock >= formation.expiresAt) {
    return {
      formation: { ...formation, status: "expired" },
      error: "Formation window expired",
    };
  }

  // Check not already accepted by this party
  if (acceptor === "human" && formation.humanAccepted) {
    return { formation, error: "Human already accepted" };
  }
  if (acceptor === "agent" && formation.agentAccepted) {
    return { formation, error: "Agent already accepted" };
  }

  const updated: PartnershipFormationState = {
    ...formation,
    humanAccepted: acceptor === "human" ? true : formation.humanAccepted,
    agentAccepted: acceptor === "agent" ? true : formation.agentAccepted,
  };

  // Check if both accepted
  if (updated.humanAccepted && updated.agentAccepted) {
    updated.status = "accepted";
  }

  return { formation: updated };
}

/**
 * Record deposit for formation
 */
export function recordDeposit(
  formation: PartnershipFormationState,
  depositor: "human" | "agent",
  amount: bigint,
  currentBlock: bigint,
): { formation: PartnershipFormationState; error?: string } {
  // Check not expired
  if (currentBlock >= formation.expiresAt) {
    return {
      formation: { ...formation, status: "expired" },
      error: "Formation window expired",
    };
  }

  // Check accepted
  if (formation.status !== "accepted") {
    return { formation, error: "Formation not yet accepted by both parties" };
  }

  // Validate amount
  const expectedAmount =
    depositor === "human"
      ? formation.request.humanContribution
      : formation.request.agentContribution;

  if (amount < expectedAmount) {
    return {
      formation,
      error: `Insufficient deposit: ${amount} < ${expectedAmount}`,
    };
  }

  const updated: PartnershipFormationState = {
    ...formation,
    humanDeposited: depositor === "human" ? true : formation.humanDeposited,
    agentDeposited: depositor === "agent" ? true : formation.agentDeposited,
  };

  return { formation: updated };
}

// ============ Formation Finalization ============

/**
 * Finalize formation and create partnership
 */
export function finalizeFormation(
  formation: PartnershipFormationState,
  currentBlock: bigint,
): { partnership: PartnershipAccount; error?: string } {
  // Check not expired
  if (currentBlock >= formation.expiresAt) {
    return { partnership: null as any, error: "Formation window expired" };
  }

  // Check both accepted
  if (!formation.humanAccepted || !formation.agentAccepted) {
    return { partnership: null as any, error: "Both parties must accept" };
  }

  // Check both deposited
  if (!formation.humanDeposited || !formation.agentDeposited) {
    return { partnership: null as any, error: "Both parties must deposit" };
  }

  const { request } = formation;

  // Create partnership ID
  const id = generatePartnershipId(
    request.humanDid,
    request.agentDid,
    currentBlock,
  );

  // Create split
  const split = request.proposedSplit
    ? createCustomSplit(
        request.proposedSplit.human,
        request.proposedSplit.agent,
      ).split!
    : createDefaultSplit();

  // Create pool
  const pool = createPool(
    request.humanContribution,
    request.agentContribution,
    request.proposedLockTier!,
    currentBlock,
  );

  // Create threshold key for common pot (2-of-2 multi-sig)
  // Seeds derived from operational keys for deterministic generation
  const humanSeed = domainHash(
    "LGM.partnership.seed.human.v1",
    request.humanOperationalKey,
  );
  const agentSeed = domainHash(
    "LGM.partnership.seed.agent.v1",
    request.agentOperationalKey,
  );

  const thresholdKey = generateThresholdKeyConfig(
    id,
    humanSeed,
    agentSeed,
    currentBlock,
  );

  // Create common pot with 50% of each contribution
  // The other 50% stays in their individual pool contributions
  const commonPotContribution =
    (request.humanContribution + request.agentContribution) / 2n;

  const commonPot = createCommonPot(
    id,
    thresholdKey,
    request.agentDid,
    commonPotContribution,
    pool.lockExpiry, // Locked same as pool
    currentBlock,
  );

  // Create metrics
  const metrics: PartnershipMetrics = {
    totalVerifications: 0,
    correctVerifications: 0,
    accuracy: 0,
    totalRewardsEarned: 0n,
    totalComputeInvested: 0n,
    consecutiveActivePeriods: 0,
    lastActivityBlock: currentBlock,
  };

  const partnership: PartnershipAccount = {
    id,
    createdAt: currentBlock,
    humanDid: request.humanDid,
    agentDid: request.agentDid,
    humanOperationalKey: request.humanOperationalKey,
    agentOperationalKey: request.agentOperationalKey,
    signingMode: "both_required",
    tier: 0,
    tierAchievedAt: currentBlock,
    pool,
    split,
    commonPot,
    status: "active",
    metrics,
  };

  return { partnership };
}

/**
 * Cancel formation
 */
export function cancelFormation(
  formation: PartnershipFormationState,
  canceller: "human" | "agent",
  currentBlock: bigint,
): {
  formation: PartnershipFormationState;
  refundHuman: bigint;
  refundAgent: bigint;
} {
  // Calculate refunds (if deposits were made)
  const refundHuman = formation.humanDeposited
    ? formation.request.humanContribution
    : 0n;
  const refundAgent = formation.agentDeposited
    ? formation.request.agentContribution
    : 0n;

  return {
    formation: { ...formation, status: "cancelled" },
    refundHuman,
    refundAgent,
  };
}

// ============ Helpers ============

/**
 * Generate unique partnership ID
 */
function generatePartnershipId(
  humanDid: DID,
  agentDid: DID,
  createdAt: bigint,
): string {
  const input = `${humanDid}:${agentDid}:${createdAt.toString()}`;
  return `partnership:${sha512_256Hex(input).slice(0, 32)}`;
}

/**
 * Check if formation is expired
 */
export function isFormationExpired(
  formation: PartnershipFormationState,
  currentBlock: bigint,
): boolean {
  return currentBlock >= formation.expiresAt;
}

/**
 * Get formation status summary
 */
export function getFormationStatus(
  formation: PartnershipFormationState,
  currentBlock: bigint,
): {
  status: string;
  humanAccepted: boolean;
  agentAccepted: boolean;
  humanDeposited: boolean;
  agentDeposited: boolean;
  blocksRemaining: bigint;
  canFinalize: boolean;
} {
  const blocksRemaining =
    formation.expiresAt > currentBlock
      ? formation.expiresAt - currentBlock
      : 0n;

  const canFinalize =
    formation.humanAccepted &&
    formation.agentAccepted &&
    formation.humanDeposited &&
    formation.agentDeposited &&
    currentBlock < formation.expiresAt;

  return {
    status: formation.status,
    humanAccepted: formation.humanAccepted,
    agentAccepted: formation.agentAccepted,
    humanDeposited: formation.humanDeposited,
    agentDeposited: formation.agentDeposited,
    blocksRemaining,
    canFinalize,
  };
}
