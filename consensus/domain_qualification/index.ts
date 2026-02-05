/**
 * Domain Qualification System
 *
 * Prevents unqualified validators from dominating specialized domains
 * by requiring demonstrated competence or stake commitment.
 *
 * Four pathways to qualify:
 * 1. Stake Commitment - Lock 22/222/2222 LGM in domain
 * 2. Track Record - 19 verifications @ 77% accuracy in related domain
 * 3. Cross-Reference - 3 endorsers each stake 22 LGM
 * 4. Stratum Inheritance - Qualified in lower stratum (70%^distance weight)
 *
 * @module consensus/domain_qualification
 */

// Types
export type {
  QualificationPathway,
  QualificationStatus,
  QualificationMetrics,
  DomainQualification,
  QualificationRequest,
  QualificationResult,
  ValidatorQualificationState,
  QualificationEndorsement,
} from "./types.js";

export {
  DOMAIN_QUALIFICATION_CONFIG,
  LGM,
  createEmptyMetrics,
  createEmptyQualificationState,
  calculateQualificationStake,
  isQualificationValid,
  calculateQualificationWeight,
} from "./types.js";

// Pathways
export {
  qualifyByStakeCommitment,
  qualifyByTrackRecord,
  qualifyByCrossReference,
  qualifyByStratumInheritance,
  updateQualificationMetrics,
  recordChallengeResult,
  checkQualificationExpiration,
  renewQualification,
  findInheritableQualification,
} from "./pathways.js";
