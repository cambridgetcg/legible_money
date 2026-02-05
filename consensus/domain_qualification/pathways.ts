/**
 * Domain Qualification Pathways
 *
 * Four pathways to earn domain qualification:
 * 1. Stake Commitment - Lock stake in domain
 * 2. Track Record - Prove competence in related domain
 * 3. Cross-Reference - Endorsed by qualified validators
 * 4. Stratum Inheritance - Qualified in lower stratum
 */

import type { PublicKey } from "../../core/crypto/keys.js";
import type {
  OntologicalStratum,
  DomainTree,
} from "../../core/knowledge/ontology/types.js";
import { STRATUM_HIERARCHY } from "../../core/knowledge/ontology/types.js";
import {
  type DomainQualification,
  type QualificationResult,
  type ValidatorQualificationState,
  DOMAIN_QUALIFICATION_CONFIG,
  createEmptyMetrics,
  calculateQualificationStake,
} from "./types.js";

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Get domain depth from domain tree
 */
function getDomainDepth(domain: string, domainTree: DomainTree): number {
  const node = domainTree.domainIndex.get(domain);
  return node?.depth ?? domain.split(".").length;
}

/**
 * Get domain stratum from domain tree
 */
function getDomainStratum(
  domain: string,
  domainTree: DomainTree,
): OntologicalStratum {
  const node = domainTree.domainIndex.get(domain);
  return node?.domain.stratum ?? ("empirical" as unknown as OntologicalStratum);
}

/**
 * Get parent domain from domain ID
 */
function getParentDomain(
  domain: string,
  _domainTree: DomainTree,
): string | null {
  const parts = domain.split(".");
  if (parts.length <= 1) return null;
  return parts.slice(0, -1).join(".");
}

/**
 * Get sibling domains (same parent)
 */
function getSiblingDomains(domain: string, domainTree: DomainTree): string[] {
  const parentDomain = getParentDomain(domain, domainTree);
  if (!parentDomain) return [];

  const parentNode = domainTree.domainIndex.get(parentDomain);
  if (!parentNode) return [];

  return parentNode.children
    .map((child) => child.domain.id)
    .filter((id) => id !== domain);
}

// ============================================================================
// Pathway 1: Stake Commitment
// ============================================================================

/**
 * Qualify by locking stake in a domain
 *
 * Pros: Fast, anyone can do it
 * Cons: Expensive, stake at risk
 */
export function qualifyByStakeCommitment(
  validatorId: PublicKey,
  domain: string,
  stake: bigint,
  currentBlock: bigint,
  domainTree: DomainTree,
): QualificationResult {
  const depth = getDomainDepth(domain, domainTree);
  const requiredStake = calculateQualificationStake(depth);

  if (stake < requiredStake) {
    return {
      granted: false,
      errors: [`Insufficient stake: ${stake} < ${requiredStake}`],
      requiredStake,
    };
  }

  // Validate domain exists
  if (!domainTree.domainIndex.has(domain)) {
    return {
      granted: false,
      errors: [`Domain "${domain}" does not exist`],
      requiredStake,
    };
  }

  const qualification: DomainQualification = {
    validatorId,
    domain,
    stratum: getDomainStratum(domain, domainTree),
    pathway: "stake_commitment",
    earnedAt: currentBlock,
    expiresAt: null, // Stake commitment doesn't expire
    status: "probationary", // Start on probation
    metrics: createEmptyMetrics(),
    lockedStake: stake,
    stakeUnlocksAt: currentBlock + DOMAIN_QUALIFICATION_CONFIG.stakeLockPeriod,
    inheritanceDiscount: 1.0, // Full weight
  };

  return { granted: true, qualification, errors: [] };
}

// ============================================================================
// Pathway 2: Track Record
// ============================================================================

/**
 * Qualify by demonstrating competence in parent/adjacent domain
 *
 * Pros: Free (no stake), proves competence
 * Cons: Takes time, requires existing qualification
 */
export function qualifyByTrackRecord(
  validatorId: PublicKey,
  domain: string,
  qualificationState: ValidatorQualificationState,
  currentBlock: bigint,
  domainTree: DomainTree,
): QualificationResult {
  const existingQualifications = qualificationState.qualifications;

  // Find qualification in parent or adjacent domain
  const parentDomain = getParentDomain(domain, domainTree);
  const siblingDomains = getSiblingDomains(domain, domainTree);

  const eligibleDomains = [parentDomain, ...siblingDomains].filter(
    (d): d is string => d !== null,
  );

  let bestQualification: DomainQualification | null = null;

  for (const eligibleDomain of eligibleDomains) {
    const qual = existingQualifications.get(eligibleDomain);
    if (qual && qual.status === "active") {
      if (
        !bestQualification ||
        qual.metrics.accuracy > bestQualification.metrics.accuracy
      ) {
        bestQualification = qual;
      }
    }
  }

  if (!bestQualification) {
    return {
      granted: false,
      errors: ["No active qualification in parent or adjacent domain"],
      requiredVerifications:
        DOMAIN_QUALIFICATION_CONFIG.minVerificationsForQualification,
    };
  }

  // Check metrics
  const { minVerificationsForQualification, minAccuracyForQualification } =
    DOMAIN_QUALIFICATION_CONFIG;

  if (
    bestQualification.metrics.totalVerifications <
    minVerificationsForQualification
  ) {
    return {
      granted: false,
      errors: [
        `Insufficient verifications: ${bestQualification.metrics.totalVerifications} < ${minVerificationsForQualification}`,
      ],
      requiredVerifications: minVerificationsForQualification,
    };
  }

  if (bestQualification.metrics.accuracy < minAccuracyForQualification) {
    return {
      granted: false,
      errors: [
        `Insufficient accuracy: ${(bestQualification.metrics.accuracy * 100).toFixed(1)}% < ${minAccuracyForQualification * 100}%`,
      ],
    };
  }

  const qualification: DomainQualification = {
    validatorId,
    domain,
    stratum: getDomainStratum(domain, domainTree),
    pathway: "track_record",
    earnedAt: currentBlock,
    expiresAt:
      currentBlock + DOMAIN_QUALIFICATION_CONFIG.inactivityExpirationBlocks,
    status: "active", // Track record = immediate active
    metrics: createEmptyMetrics(),
    inheritanceDiscount: 1.0,
  };

  return { granted: true, qualification, errors: [] };
}

// ============================================================================
// Pathway 3: Cross-Reference (Web of Trust)
// ============================================================================

/**
 * Qualify by endorsement from N qualified validators
 *
 * Pros: Social proof, distributed trust
 * Cons: Requires network, endorsers at risk
 */
export function qualifyByCrossReference(
  validatorId: PublicKey,
  domain: string,
  endorsers: PublicKey[],
  endorserQualifications: Map<string, DomainQualification>,
  endorserStakes: Map<string, bigint>,
  currentBlock: bigint,
  domainTree: DomainTree,
): QualificationResult {
  const { minEndorsersForQualification, endorserStakeRequirement } =
    DOMAIN_QUALIFICATION_CONFIG;

  // Validate endorsers
  const validEndorsers: PublicKey[] = [];
  const errors: string[] = [];

  for (const endorser of endorsers) {
    // Can't endorse yourself
    if (endorser === validatorId) {
      errors.push("Cannot endorse yourself");
      continue;
    }

    const qual = endorserQualifications.get(endorser);
    const stake = endorserStakes.get(endorser) || 0n;

    if (!qual || qual.status !== "active") {
      errors.push(
        `Endorser ${endorser.slice(0, 8)}... not qualified in domain`,
      );
      continue;
    }

    if (stake < endorserStakeRequirement) {
      errors.push(
        `Endorser ${endorser.slice(0, 8)}... insufficient stake: ${stake} < ${endorserStakeRequirement}`,
      );
      continue;
    }

    validEndorsers.push(endorser);
  }

  if (validEndorsers.length < minEndorsersForQualification) {
    return {
      granted: false,
      errors: [
        `Insufficient valid endorsers: ${validEndorsers.length} < ${minEndorsersForQualification}`,
        ...errors,
      ],
      requiredEndorsers: minEndorsersForQualification,
    };
  }

  const qualification: DomainQualification = {
    validatorId,
    domain,
    stratum: getDomainStratum(domain, domainTree),
    pathway: "cross_reference",
    earnedAt: currentBlock,
    expiresAt:
      currentBlock + DOMAIN_QUALIFICATION_CONFIG.endorsementDecayBlocks,
    status: "probationary",
    metrics: createEmptyMetrics(),
    endorsers: validEndorsers,
    endorserStakes: new Map(
      validEndorsers.map((e) => [e, endorserStakes.get(e)!]),
    ),
    inheritanceDiscount: 1.0,
  };

  return { granted: true, qualification, errors };
}

// ============================================================================
// Pathway 4: Stratum Inheritance
// ============================================================================

/**
 * Qualify in higher stratum by being qualified in lower stratum
 *
 * Rationale: Physics knowledge is relevant to chemistry claims
 *
 * Pros: Leverages foundational knowledge
 * Cons: Reduced weight (70%^distance)
 */
export function qualifyByStratumInheritance(
  validatorId: PublicKey,
  targetDomain: string,
  qualificationState: ValidatorQualificationState,
  currentBlock: bigint,
  domainTree: DomainTree,
): QualificationResult {
  const existingQualifications = qualificationState.qualifications;
  const targetStratum = getDomainStratum(targetDomain, domainTree);
  const targetLevel = STRATUM_HIERARCHY[targetStratum];

  // Find qualification in a lower stratum
  let lowerQualification: DomainQualification | null = null;
  let lowerStratumLevel = -1;

  for (const qual of existingQualifications.values()) {
    if (qual.status !== "active") continue;

    const qualLevel = STRATUM_HIERARCHY[qual.stratum];
    if (qualLevel < targetLevel && qualLevel > lowerStratumLevel) {
      lowerQualification = qual;
      lowerStratumLevel = qualLevel;
    }
  }

  if (!lowerQualification) {
    return {
      granted: false,
      errors: [
        `No active qualification in lower stratum. Target: ${targetStratum} (level ${targetLevel})`,
      ],
    };
  }

  // Calculate inheritance discount based on stratum distance
  const stratumDistance = targetLevel - lowerStratumLevel;
  const discount = Math.pow(
    DOMAIN_QUALIFICATION_CONFIG.stratumInheritanceDiscount,
    stratumDistance,
  );

  const qualification: DomainQualification = {
    validatorId,
    domain: targetDomain,
    stratum: targetStratum,
    pathway: "stratum_inheritance",
    earnedAt: currentBlock,
    expiresAt:
      currentBlock + DOMAIN_QUALIFICATION_CONFIG.inactivityExpirationBlocks,
    status: "active",
    metrics: createEmptyMetrics(),
    inheritedFrom: lowerQualification.domain,
    inheritanceDiscount: discount, // 0.7^distance
  };

  return { granted: true, qualification, errors: [] };
}

// ============================================================================
// Qualification Management
// ============================================================================

/**
 * Update qualification metrics after verification
 */
export function updateQualificationMetrics(
  qualification: DomainQualification,
  wasCorrect: boolean,
  currentBlock: bigint,
): DomainQualification {
  const metrics = { ...qualification.metrics };

  metrics.totalVerifications++;
  metrics.lastVerificationBlock = currentBlock;

  if (wasCorrect) {
    metrics.correctVerifications++;
    metrics.consecutiveCorrect++;
    metrics.consecutiveIncorrect = 0;
  } else {
    metrics.consecutiveIncorrect++;
    metrics.consecutiveCorrect = 0;
  }

  metrics.accuracy = metrics.correctVerifications / metrics.totalVerifications;

  // Update status based on performance
  let newStatus = qualification.status;

  // Promote from probationary to active after enough verifications
  if (
    qualification.status === "probationary" &&
    metrics.totalVerifications >=
      DOMAIN_QUALIFICATION_CONFIG.minVerificationsForQualification &&
    metrics.accuracy >= DOMAIN_QUALIFICATION_CONFIG.minAccuracyForQualification
  ) {
    newStatus = "active";
  }

  // Suspend if accuracy drops too low
  if (
    metrics.accuracy <
      DOMAIN_QUALIFICATION_CONFIG.accuracyThresholdForSuspension &&
    metrics.totalVerifications >=
      DOMAIN_QUALIFICATION_CONFIG.minVerificationsForQualification
  ) {
    newStatus = "suspended";
  }

  // Suspend if too many consecutive incorrect
  if (
    metrics.consecutiveIncorrect >=
    DOMAIN_QUALIFICATION_CONFIG.consecutiveIncorrectForSuspension
  ) {
    newStatus = "suspended";
  }

  return {
    ...qualification,
    metrics,
    status: newStatus,
  };
}

/**
 * Record challenge result for qualification
 */
export function recordChallengeResult(
  qualification: DomainQualification,
  challengeLost: boolean,
): DomainQualification {
  const metrics = { ...qualification.metrics };

  metrics.challengesReceived++;
  if (challengeLost) {
    metrics.challengesLost++;
  }

  // Revoke if too many challenges lost
  let newStatus = qualification.status;
  if (
    metrics.challengesLost >=
    DOMAIN_QUALIFICATION_CONFIG.challengesLostForRevocation
  ) {
    newStatus = "revoked";
  }

  return {
    ...qualification,
    metrics,
    status: newStatus,
  };
}

/**
 * Check and update qualification expiration
 */
export function checkQualificationExpiration(
  qualification: DomainQualification,
  currentBlock: bigint,
): DomainQualification {
  if (
    qualification.expiresAt !== null &&
    currentBlock > qualification.expiresAt
  ) {
    return {
      ...qualification,
      status: "expired",
    };
  }
  return qualification;
}

/**
 * Renew expiring qualification
 */
export function renewQualification(
  qualification: DomainQualification,
  currentBlock: bigint,
): DomainQualification {
  if (
    qualification.status !== "active" &&
    qualification.status !== "probationary"
  ) {
    return qualification;
  }

  // Check if eligible for renewal (must have good metrics)
  if (
    qualification.metrics.accuracy <
    DOMAIN_QUALIFICATION_CONFIG.minAccuracyForQualification
  ) {
    return qualification;
  }

  // Extend expiration
  const newExpiration =
    qualification.expiresAt !== null
      ? qualification.expiresAt +
        DOMAIN_QUALIFICATION_CONFIG.inactivityExpirationBlocks
      : currentBlock + DOMAIN_QUALIFICATION_CONFIG.inactivityExpirationBlocks;

  return {
    ...qualification,
    expiresAt: newExpiration,
  };
}

/**
 * Find inheritable qualification for a domain (stratum inheritance fallback)
 */
export function findInheritableQualification(
  targetDomain: string,
  qualificationState: ValidatorQualificationState,
  domainTree: DomainTree,
): DomainQualification | null {
  const targetStratum = getDomainStratum(targetDomain, domainTree);
  const targetLevel = STRATUM_HIERARCHY[targetStratum];

  let bestQualification: DomainQualification | null = null;
  let bestLevel = -1;

  for (const qual of qualificationState.qualifications.values()) {
    if (qual.status !== "active") continue;

    const qualLevel = STRATUM_HIERARCHY[qual.stratum];

    // Must be lower stratum and better than current best
    if (qualLevel < targetLevel && qualLevel > bestLevel) {
      bestQualification = qual;
      bestLevel = qualLevel;
    }
  }

  if (!bestQualification) return null;

  // Create an inherited qualification on-the-fly
  const stratumDistance = targetLevel - bestLevel;
  const discount = Math.pow(
    DOMAIN_QUALIFICATION_CONFIG.stratumInheritanceDiscount,
    stratumDistance,
  );

  return {
    ...bestQualification,
    domain: targetDomain,
    stratum: targetStratum,
    pathway: "stratum_inheritance",
    inheritedFrom: bestQualification.domain,
    inheritanceDiscount: discount,
  };
}
