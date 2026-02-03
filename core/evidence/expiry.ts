/**
 * Evidence Expiry System
 *
 * Manages evidence lifecycle and freshness:
 * - Eternal evidence (formal proofs, on-chain) never expires
 * - Heartbeat evidence (oracles) expires if no update within threshold
 * - Decay-based evidence gradually loses freshness
 * - Block-based evidence expires after fixed number of blocks
 *
 * FRESHNESS FORMULA:
 *   For decay-based: freshness = e^(-λt) where λ = ln(2) / halfLife
 *   For heartbeat: freshness = max(0, 1 - (blocksSinceUpdate / maxStaleness))
 */

import type {
  Evidence,
  EvidenceExpiry,
  ExpiryType,
  HeartbeatConfig,
  EvidenceType,
} from './types.js';
import { DEFAULT_EXPIRY_CONFIGS } from './types.js';

// ============ Freshness Calculation ============

/**
 * Calculate freshness score for evidence
 * Returns 0-1 where 1 = perfectly fresh, 0 = completely stale
 */
export function calculateFreshness(
  evidence: Evidence,
  currentBlock: bigint
): number {
  const { expiry } = evidence;

  switch (expiry.expiryType) {
    case 'eternal':
      return 1.0; // Never decays

    case 'heartbeat':
      return calculateHeartbeatFreshness(expiry.heartbeatConfig!, currentBlock);

    case 'block_based':
      return calculateBlockBasedFreshness(
        evidence.submittedAtBlock,
        expiry.expiresAfterBlocks!,
        currentBlock
      );

    case 'decay_based':
      return calculateDecayFreshness(
        evidence.submittedAtBlock,
        expiry.expiresAfterBlocks!, // Used as half-life
        currentBlock
      );

    case 'event_based':
      // Event-based is either fresh (event not occurred) or expired (event occurred)
      return expiry.isExpired ? 0 : 1.0;

    default:
      return 1.0;
  }
}

/**
 * Calculate freshness for heartbeat-based evidence (oracles)
 */
export function calculateHeartbeatFreshness(
  heartbeat: HeartbeatConfig,
  currentBlock: bigint
): number {
  const blocksSinceHeartbeat = Number(currentBlock - heartbeat.lastHeartbeatBlock);

  if (blocksSinceHeartbeat <= 0) {
    return 1.0;
  }

  if (blocksSinceHeartbeat >= heartbeat.maxStaleness) {
    return 0; // Completely stale
  }

  // Linear decay from expected interval to max staleness
  if (blocksSinceHeartbeat <= heartbeat.expectedInterval) {
    return 1.0; // Still within expected interval
  }

  // Decay from 1.0 at expectedInterval to 0 at maxStaleness
  const decayRange = heartbeat.maxStaleness - heartbeat.expectedInterval;
  const decayProgress = blocksSinceHeartbeat - heartbeat.expectedInterval;

  return Math.max(0, 1 - (decayProgress / decayRange));
}

/**
 * Calculate freshness for block-based expiry
 */
export function calculateBlockBasedFreshness(
  submittedAtBlock: bigint,
  expiresAfterBlocks: number,
  currentBlock: bigint
): number {
  const blocksSinceSubmission = Number(currentBlock - submittedAtBlock);

  if (blocksSinceSubmission >= expiresAfterBlocks) {
    return 0; // Expired
  }

  // Linear decay
  return 1 - (blocksSinceSubmission / expiresAfterBlocks);
}

/**
 * Calculate freshness for decay-based evidence (exponential decay)
 *
 * Uses exponential decay formula: f(t) = e^(-λt)
 * where λ = ln(2) / halfLife (blocks)
 */
export function calculateDecayFreshness(
  submittedAtBlock: bigint,
  halfLifeBlocks: number,
  currentBlock: bigint
): number {
  const blocksSinceSubmission = Number(currentBlock - submittedAtBlock);

  if (blocksSinceSubmission <= 0) {
    return 1.0;
  }

  const lambda = Math.LN2 / halfLifeBlocks;
  const freshness = Math.exp(-lambda * blocksSinceSubmission);

  // Floor at 0.01 - evidence below this is effectively expired
  return freshness < 0.01 ? 0 : freshness;
}

// ============ Expiry Checking ============

/**
 * Check if evidence has expired
 */
export function isExpired(
  evidence: Evidence,
  currentBlock: bigint
): boolean {
  const freshness = calculateFreshness(evidence, currentBlock);
  return freshness === 0;
}

/**
 * Check if heartbeat evidence is stale (approaching expiry)
 */
export function isStale(
  heartbeat: HeartbeatConfig,
  currentBlock: bigint
): boolean {
  const blocksSinceHeartbeat = Number(currentBlock - heartbeat.lastHeartbeatBlock);
  return blocksSinceHeartbeat > heartbeat.expectedInterval;
}

/**
 * Get time until expiry (in blocks)
 */
export function blocksUntilExpiry(
  evidence: Evidence,
  currentBlock: bigint
): number | null {
  const { expiry } = evidence;

  switch (expiry.expiryType) {
    case 'eternal':
      return null; // Never expires

    case 'heartbeat':
      const blocksSinceHeartbeat = Number(currentBlock - expiry.heartbeatConfig!.lastHeartbeatBlock);
      const remaining = expiry.heartbeatConfig!.maxStaleness - blocksSinceHeartbeat;
      return Math.max(0, remaining);

    case 'block_based':
      const expiresAt = evidence.submittedAtBlock + BigInt(expiry.expiresAfterBlocks!);
      return Math.max(0, Number(expiresAt - currentBlock));

    case 'decay_based':
      // For decay-based, we consider "expired" when freshness < 0.01
      // f(t) = e^(-λt) = 0.01 → t = -ln(0.01) / λ
      const halfLife = expiry.expiresAfterBlocks!;
      const lambda = Math.LN2 / halfLife;
      const fullDecayBlocks = Math.ceil(-Math.log(0.01) / lambda);
      const blocksSince = Number(currentBlock - evidence.submittedAtBlock);
      return Math.max(0, fullDecayBlocks - blocksSince);

    case 'event_based':
      return expiry.isExpired ? 0 : null; // Unknown until event

    default:
      return null;
  }
}

// ============ Expiry Updates ============

/**
 * Update evidence expiry state
 */
export function updateExpiryState(
  evidence: Evidence,
  currentBlock: bigint
): EvidenceExpiry {
  const freshness = calculateFreshness(evidence, currentBlock);
  const expired = freshness === 0;

  return {
    ...evidence.expiry,
    freshnessScore: freshness,
    freshnessUpdatedAt: Date.now(),
    isExpired: expired,
    expiredAt: expired && !evidence.expiry.expiredAt ? Date.now() : evidence.expiry.expiredAt,
  };
}

/**
 * Update heartbeat for oracle evidence
 */
export function updateHeartbeat(
  evidence: Evidence,
  heartbeatBlock: bigint,
  heartbeatTime: number
): EvidenceExpiry {
  if (evidence.expiry.expiryType !== 'heartbeat' || !evidence.expiry.heartbeatConfig) {
    throw new Error('Cannot update heartbeat for non-heartbeat evidence');
  }

  const updatedConfig: HeartbeatConfig = {
    ...evidence.expiry.heartbeatConfig,
    lastHeartbeat: heartbeatTime,
    lastHeartbeatBlock: heartbeatBlock,
    isStale: false,
  };

  return {
    ...evidence.expiry,
    heartbeatConfig: updatedConfig,
    freshnessScore: 1.0, // Reset to fresh
    freshnessUpdatedAt: heartbeatTime,
    isExpired: false,
    expiredAt: undefined,
  };
}

/**
 * Mark evidence as expired due to event
 */
export function markExpiredByEvent(
  evidence: Evidence,
  _reason: string
): EvidenceExpiry {
  return {
    ...evidence.expiry,
    freshnessScore: 0,
    freshnessUpdatedAt: Date.now(),
    isExpired: true,
    expiredAt: Date.now(),
  };
}

// ============ Expiry Processing ============

/**
 * Result of processing expired evidence
 */
export interface ExpiredEvidenceResult {
  evidenceId: string;
  previousFreshness: number;
  newFreshness: number;
  isNewlyExpired: boolean;
  expiryType: ExpiryType;
}

/**
 * Process expiry for a batch of evidence
 */
export function processExpirations(
  evidenceList: Evidence[],
  currentBlock: bigint
): ExpiredEvidenceResult[] {
  const results: ExpiredEvidenceResult[] = [];

  for (const evidence of evidenceList) {
    // Skip already expired evidence
    if (evidence.expiry.isExpired) {
      continue;
    }

    // Skip eternal evidence
    if (evidence.expiry.expiryType === 'eternal') {
      continue;
    }

    const previousFreshness = evidence.expiry.freshnessScore;
    const newFreshness = calculateFreshness(evidence, currentBlock);
    const isNewlyExpired = previousFreshness > 0 && newFreshness === 0;

    if (newFreshness < previousFreshness || isNewlyExpired) {
      results.push({
        evidenceId: evidence.id,
        previousFreshness,
        newFreshness,
        isNewlyExpired,
        expiryType: evidence.expiry.expiryType,
      });
    }
  }

  return results;
}

/**
 * Find evidence that needs heartbeat update
 */
export function findStaleHeartbeats(
  evidenceList: Evidence[],
  currentBlock: bigint
): Evidence[] {
  return evidenceList.filter(evidence => {
    if (evidence.expiry.expiryType !== 'heartbeat') {
      return false;
    }

    if (!evidence.expiry.heartbeatConfig) {
      return false;
    }

    if (evidence.expiry.isExpired) {
      return false;
    }

    return isStale(evidence.expiry.heartbeatConfig, currentBlock);
  });
}

/**
 * Find evidence expiring soon (within threshold blocks)
 */
export function findExpiringEvidence(
  evidenceList: Evidence[],
  currentBlock: bigint,
  thresholdBlocks: number
): Evidence[] {
  return evidenceList.filter(evidence => {
    if (evidence.expiry.isExpired) {
      return false;
    }

    const remaining = blocksUntilExpiry(evidence, currentBlock);
    if (remaining === null) {
      return false; // Eternal
    }

    return remaining <= thresholdBlocks;
  });
}

// ============ Default Expiry Creation ============

/**
 * Create default expiry configuration for evidence type
 */
export function createDefaultExpiry(
  evidenceType: EvidenceType,
  submittedAt: number
): EvidenceExpiry {
  const defaults = DEFAULT_EXPIRY_CONFIGS[evidenceType];

  const base: EvidenceExpiry = {
    expiryType: defaults.expiryType || 'decay_based',
    freshnessScore: 1.0,
    freshnessUpdatedAt: submittedAt,
    isExpired: false,
  };

  // Copy type-specific fields
  if (defaults.expiresAtBlock !== undefined) {
    base.expiresAtBlock = defaults.expiresAtBlock;
  }

  if (defaults.expiresAfterBlocks !== undefined) {
    base.expiresAfterBlocks = defaults.expiresAfterBlocks;
  }

  if (defaults.heartbeatConfig) {
    base.heartbeatConfig = {
      ...defaults.heartbeatConfig,
      lastHeartbeat: submittedAt,
      lastHeartbeatBlock: 0n,
    };
  }

  return base;
}

// ============ Confidence Adjustment ============

/**
 * Apply freshness adjustment to confidence
 *
 * Evidence freshness affects its contribution to overall fact confidence.
 * Stale evidence provides less support.
 */
export function applyFreshnessToConfidence(
  baseConfidence: number,
  freshness: number
): number {
  // Square root to make the penalty gentler
  // freshness=0.5 → multiplier=0.71
  // freshness=0.25 → multiplier=0.5
  const multiplier = Math.sqrt(freshness);
  return baseConfidence * multiplier;
}

/**
 * Calculate effective reliability with freshness
 */
export function calculateEffectiveReliability(
  evidence: Evidence,
  currentBlock: bigint
): number {
  const freshness = calculateFreshness(evidence, currentBlock);
  return applyFreshnessToConfidence(evidence.reliabilityScore, freshness);
}
