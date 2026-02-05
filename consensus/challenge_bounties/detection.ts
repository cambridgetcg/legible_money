/**
 * Capture Detection
 *
 * Automatic detection of domain capture patterns through
 * statistical analysis of verification history.
 */

import {
  type CaptureMetrics,
  type CaptureRiskAnalysis,
  type VerificationHistoryEntry,
  CHALLENGE_BOUNTY_CONFIG,
  createEmptyCaptureMetrics,
} from "./types.js";

// ============================================================================
// Herfindahl-Hirschman Index (Concentration)
// ============================================================================

/**
 * Calculate Herfindahl-Hirschman Index for validator concentration
 *
 * HHI = sum of squared market shares
 * Range: 0 (perfect competition) to 1 (monopoly)
 *
 * Interpretation:
 * - HHI < 0.15: Low concentration
 * - 0.15 <= HHI < 0.25: Moderate concentration
 * - HHI >= 0.25: High concentration
 */
export function calculateHerfindahlIndex(
  verificationHistory: VerificationHistoryEntry[],
): number {
  if (verificationHistory.length === 0) return 0;

  // Count participations per validator
  const validatorCounts = new Map<string, number>();
  let totalParticipations = 0;

  for (const entry of verificationHistory) {
    for (const validator of entry.validators) {
      validatorCounts.set(validator, (validatorCounts.get(validator) || 0) + 1);
      totalParticipations++;
    }
  }

  if (totalParticipations === 0) return 0;

  // Calculate HHI
  let hhi = 0;
  for (const count of validatorCounts.values()) {
    const share = count / totalParticipations;
    hhi += share * share;
  }

  return hhi;
}

/**
 * Calculate share of top N validators
 */
export function calculateTopNShare(
  verificationHistory: VerificationHistoryEntry[],
  n: number,
): number {
  if (verificationHistory.length === 0) return 0;

  // Count participations per validator
  const validatorCounts = new Map<string, number>();
  let totalParticipations = 0;

  for (const entry of verificationHistory) {
    for (const validator of entry.validators) {
      validatorCounts.set(validator, (validatorCounts.get(validator) || 0) + 1);
      totalParticipations++;
    }
  }

  if (totalParticipations === 0) return 0;

  // Sort by count and get top N
  const sorted = Array.from(validatorCounts.values()).sort((a, b) => b - a);
  const topNTotal = sorted.slice(0, n).reduce((sum, c) => sum + c, 0);

  return topNTotal / totalParticipations;
}

// ============================================================================
// Timing Correlation
// ============================================================================

/**
 * Calculate timing correlation between validators
 *
 * Measures how synchronized validator submissions are.
 * High correlation suggests coordination.
 */
export function calculateTimingCorrelation(
  verificationHistory: VerificationHistoryEntry[],
): number {
  if (verificationHistory.length < 2) return 0;

  // For each verification round, calculate timing variance
  let totalVariance = 0;
  let roundsWithTiming = 0;

  for (const entry of verificationHistory) {
    if (entry.submissionTimestamps.length < 2) continue;

    const timestamps = entry.submissionTimestamps.map(Number);
    const mean = timestamps.reduce((a, b) => a + b, 0) / timestamps.length;
    const variance =
      timestamps.reduce((sum, t) => sum + Math.pow(t - mean, 2), 0) /
      timestamps.length;

    // Normalize variance (lower variance = higher correlation)
    // Assume normal variance is ~1000 blocks
    const normalizedVariance = Math.min(1, variance / 1000);

    totalVariance += normalizedVariance;
    roundsWithTiming++;
  }

  if (roundsWithTiming === 0) return 0;

  // Invert: high correlation = low variance
  const avgVariance = totalVariance / roundsWithTiming;
  return 1 - avgVariance;
}

// ============================================================================
// Verdict Correlation
// ============================================================================

/**
 * Calculate verdict correlation between validators
 *
 * Measures how often validators vote the same way.
 * High correlation suggests collusion.
 */
export function calculateVerdictCorrelation(
  verificationHistory: VerificationHistoryEntry[],
): number {
  if (verificationHistory.length < 2) return 0;

  // Build validator pair agreement matrix
  const pairAgreements = new Map<string, { agreed: number; total: number }>();

  for (const entry of verificationHistory) {
    const validators = entry.validators;
    const verdicts = entry.verdicts;

    // For each pair of validators
    for (let i = 0; i < validators.length; i++) {
      for (let j = i + 1; j < validators.length; j++) {
        const pairKey = [validators[i], validators[j]].sort().join(":");

        const existing = pairAgreements.get(pairKey) || { agreed: 0, total: 0 };
        existing.total++;

        if (verdicts[i] === verdicts[j]) {
          existing.agreed++;
        }

        pairAgreements.set(pairKey, existing);
      }
    }
  }

  if (pairAgreements.size === 0) return 0;

  // Calculate average agreement rate
  let totalAgreementRate = 0;
  for (const { agreed, total } of pairAgreements.values()) {
    totalAgreementRate += agreed / total;
  }

  const avgAgreementRate = totalAgreementRate / pairAgreements.size;

  // Expected agreement by chance is ~33% (3 possible verdicts)
  // High correlation means agreement significantly above chance
  // Normalize: 0.33 -> 0, 1.0 -> 1.0
  return Math.max(0, (avgAgreementRate - 0.33) / 0.67);
}

// ============================================================================
// Composite Risk Analysis
// ============================================================================

/**
 * Analyze domain for capture risk
 *
 * Returns composite risk score based on:
 * - Herfindahl Index (40% weight)
 * - Timing correlation (30% weight)
 * - Verdict correlation (30% weight)
 */
export function analyzeCaptureRisk(
  domain: string,
  verificationHistory: VerificationHistoryEntry[],
  currentBlock: bigint,
): CaptureRiskAnalysis {
  // Need minimum verifications for meaningful analysis
  if (
    verificationHistory.length <
    CHALLENGE_BOUNTY_CONFIG.minVerificationsForAnalysis
  ) {
    return {
      domain,
      riskScore: 0,
      metrics: createEmptyCaptureMetrics(),
      shouldCreateBounty: false,
      suggestedBountySize: 0n,
      analyzedAt: currentBlock,
      verificationCount: verificationHistory.length,
    };
  }

  // Calculate individual metrics
  const herfindahlIndex = calculateHerfindahlIndex(verificationHistory);
  const timingCorrelation = calculateTimingCorrelation(verificationHistory);
  const verdictCorrelation = calculateVerdictCorrelation(verificationHistory);
  const top3Share = calculateTopNShare(verificationHistory, 3);

  // Count unique validators
  const uniqueValidators = new Set<string>();
  for (const entry of verificationHistory) {
    for (const validator of entry.validators) {
      uniqueValidators.add(validator);
    }
  }

  const metrics: CaptureMetrics = {
    herfindahlIndex,
    timingCorrelation,
    verdictCorrelation,
    uniqueValidators: uniqueValidators.size,
    totalVerifications: verificationHistory.length,
    top3ValidatorShare: top3Share,
  };

  // Composite risk score
  const riskScore =
    herfindahlIndex * 0.4 + timingCorrelation * 0.3 + verdictCorrelation * 0.3;

  // Determine if bounty should be created
  const shouldCreateBounty =
    riskScore > CHALLENGE_BOUNTY_CONFIG.herfindahlTriggerThreshold;

  // Suggest bounty size based on risk
  const suggestedBountySize = shouldCreateBounty
    ? BigInt(
        Math.floor(Number(CHALLENGE_BOUNTY_CONFIG.baseBountyPool) * riskScore),
      )
    : 0n;

  return {
    domain,
    riskScore,
    metrics,
    shouldCreateBounty,
    suggestedBountySize,
    analyzedAt: currentBlock,
    verificationCount: verificationHistory.length,
  };
}

/**
 * Get risk level description
 */
export function getRiskLevelDescription(riskScore: number): string {
  if (riskScore < 0.15) return "Low";
  if (riskScore < 0.25) return "Moderate";
  if (riskScore < 0.4) return "Elevated";
  if (riskScore < 0.6) return "High";
  return "Critical";
}

/**
 * Analyze specific validator pair for coordination
 */
export function analyzeValidatorPair(
  validator1: string,
  validator2: string,
  verificationHistory: VerificationHistoryEntry[],
): {
  coOccurrenceRate: number;
  agreementRate: number;
  timingSync: number;
  suspicious: boolean;
} {
  let coOccurrences = 0;
  let agreements = 0;
  const timingDiffs: number[] = [];

  for (const entry of verificationHistory) {
    const idx1 = entry.validators.indexOf(validator1);
    const idx2 = entry.validators.indexOf(validator2);

    if (idx1 !== -1 && idx2 !== -1) {
      coOccurrences++;

      if (entry.verdicts[idx1] === entry.verdicts[idx2]) {
        agreements++;
      }

      if (entry.submissionTimestamps.length > Math.max(idx1, idx2)) {
        const timeDiff = Math.abs(
          Number(entry.submissionTimestamps[idx1]) -
            Number(entry.submissionTimestamps[idx2]),
        );
        timingDiffs.push(timeDiff);
      }
    }
  }

  const totalRounds = verificationHistory.length;
  const coOccurrenceRate = coOccurrences / totalRounds;
  const agreementRate = coOccurrences > 0 ? agreements / coOccurrences : 0;

  // Calculate timing synchronization
  let timingSync = 0;
  if (timingDiffs.length > 0) {
    const avgDiff = timingDiffs.reduce((a, b) => a + b, 0) / timingDiffs.length;
    // Normalize: 0 diff = 1.0 sync, 100+ blocks diff = 0 sync
    timingSync = Math.max(0, 1 - avgDiff / 100);
  }

  // Flag as suspicious if all metrics are high
  const suspicious =
    coOccurrenceRate > 0.8 && agreementRate > 0.9 && timingSync > 0.8;

  return {
    coOccurrenceRate,
    agreementRate,
    timingSync,
    suspicious,
  };
}

/**
 * Find most suspicious validator pairs
 */
export function findSuspiciousPairs(
  verificationHistory: VerificationHistoryEntry[],
  threshold: number = 0.8,
): Array<{
  validator1: string;
  validator2: string;
  suspicionScore: number;
}> {
  // Get all validators
  const validators = new Set<string>();
  for (const entry of verificationHistory) {
    for (const v of entry.validators) {
      validators.add(v);
    }
  }

  const validatorList = Array.from(validators);
  const suspiciousPairs: Array<{
    validator1: string;
    validator2: string;
    suspicionScore: number;
  }> = [];

  // Analyze each pair
  for (let i = 0; i < validatorList.length; i++) {
    for (let j = i + 1; j < validatorList.length; j++) {
      const analysis = analyzeValidatorPair(
        validatorList[i],
        validatorList[j],
        verificationHistory,
      );

      const suspicionScore =
        analysis.coOccurrenceRate * 0.3 +
        analysis.agreementRate * 0.4 +
        analysis.timingSync * 0.3;

      if (suspicionScore > threshold) {
        suspiciousPairs.push({
          validator1: validatorList[i],
          validator2: validatorList[j],
          suspicionScore,
        });
      }
    }
  }

  // Sort by suspicion score
  return suspiciousPairs.sort((a, b) => b.suspicionScore - a.suspicionScore);
}
