/**
 * Paper Decomposition Helper
 *
 * Utilities for breaking down research papers into individual verifiable claims.
 *
 * A single paper typically contains multiple types of claims:
 * - Methodology claims (how the study was done)
 * - Measurement claims (what was observed)
 * - Statistical claims (quantitative findings)
 * - Interpretation claims (what it means)
 * - Causal claims (cause-effect relationships)
 *
 * This module helps structure these claims for submission to the protocol.
 */

import type { ClaimSubmission, Reference } from "../types.js";

// ============ Paper Structure Types ============

/**
 * Claim type within a paper
 */
export type PaperClaimType =
  | "methodology" // How the study was done
  | "sample" // Sample characteristics
  | "measurement" // Quantitative results
  | "statistical" // Statistical findings
  | "observation" // Qualitative observations
  | "interpretation" // Authors' interpretation
  | "conclusion" // Main conclusions
  | "causal" // Cause-effect claims
  | "limitation"; // Stated limitations

/**
 * Confidence guidelines by claim type
 *
 * These are maximum recommended confidence values.
 * Actual confidence depends on evidence quality.
 */
export const CLAIM_TYPE_MAX_CONFIDENCE: Record<PaperClaimType, number> = {
  methodology: 0.95, // Stated fact - high confidence
  sample: 0.95, // Stated fact
  measurement: 0.85, // Single study measurement
  statistical: 0.85, // Depends on methods
  observation: 0.8, // Qualitative, some subjectivity
  interpretation: 0.7, // Authors' view
  conclusion: 0.7, // Summary interpretation
  causal: 0.6, // Hardest to verify
  limitation: 0.9, // Stated by authors
};

/**
 * Schema mapping by claim type
 */
export const CLAIM_TYPE_SCHEMA: Record<PaperClaimType, string> = {
  methodology: "methodology_claim_v1",
  sample: "methodology_claim_v1",
  measurement: "experimental_result_v1",
  statistical: "statistical_claim_v1",
  observation: "research_claim_v1",
  interpretation: "research_claim_v1",
  conclusion: "research_claim_v1",
  causal: "causal_claim_v1",
  limitation: "research_claim_v1",
};

/**
 * Extracted claim from a paper
 */
export interface ExtractedClaim {
  /** Claim type */
  type: PaperClaimType;

  /** Natural language claim */
  claimText: string;

  /** Location in paper */
  location: string;

  /** Direct quote supporting claim */
  quote?: string;

  /** Structured data for schema */
  schemaData: Record<string, unknown>;

  /** Recommended confidence */
  recommendedConfidence: number;

  /** Depends on these other claims (indices) */
  dependsOn?: number[];

  /** Notes for validators */
  validatorNotes?: string;
}

/**
 * Paper decomposition result
 */
export interface PaperDecomposition {
  /** Paper metadata */
  paper: {
    doi: string;
    title: string;
    authors: string[];
    journal: string;
    publicationDate: string;
    field: string;
    subfield?: string;
    peerReviewStatus:
      | "preprint"
      | "under_review"
      | "peer_reviewed"
      | "retracted";
    abstractHash?: string;
    fullTextHash?: string;
  };

  /** Extracted claims */
  claims: ExtractedClaim[];

  /** Claim dependency graph (indices) */
  dependencyGraph: Map<number, number[]>;

  /** Submission order (topologically sorted) */
  submissionOrder: number[];
}

// ============ Decomposition Helpers ============

/**
 * Create a paper decomposition structure
 */
export function createPaperDecomposition(
  paper: PaperDecomposition["paper"],
): PaperDecomposition {
  return {
    paper,
    claims: [],
    dependencyGraph: new Map(),
    submissionOrder: [],
  };
}

/**
 * Add a methodology claim
 */
export function addMethodologyClaim(
  decomposition: PaperDecomposition,
  claim: {
    methodologyType:
      | "sample_selection"
      | "experimental_design"
      | "measurement_technique"
      | "statistical_analysis"
      | "data_collection"
      | "control_conditions"
      | "blinding"
      | "randomization";
    claimText: string;
    location: string;
    quote?: string;
    sampleSize?: number;
    sampleCharacteristics?: Record<string, unknown>;
    technique?: string;
    equipment?: string;
  },
): number {
  const index = decomposition.claims.length;

  decomposition.claims.push({
    type: "methodology",
    claimText: claim.claimText,
    location: claim.location,
    quote: claim.quote,
    schemaData: {
      paperDoi: decomposition.paper.doi,
      methodologyType: claim.methodologyType,
      claim: claim.claimText,
      location: claim.location,
      quote: claim.quote,
      sampleSize: claim.sampleSize,
      sampleCharacteristics: claim.sampleCharacteristics,
      technique: claim.technique,
      equipment: claim.equipment,
    },
    recommendedConfidence: CLAIM_TYPE_MAX_CONFIDENCE.methodology,
  });

  return index;
}

/**
 * Add a measurement/experimental result claim
 */
export function addMeasurementClaim(
  decomposition: PaperDecomposition,
  claim: {
    experimentType: string;
    hypothesis: string;
    result: string;
    location: string;
    measurements: Array<{
      name: string;
      value: number;
      unit?: string;
      uncertainty?: number;
    }>;
    conditions?: Record<string, unknown>;
    dataHash?: string;
    dependsOn?: number[];
  },
): number {
  const index = decomposition.claims.length;

  decomposition.claims.push({
    type: "measurement",
    claimText: claim.result,
    location: claim.location,
    schemaData: {
      experimentType: claim.experimentType,
      hypothesis: claim.hypothesis,
      result: claim.result,
      measurements: claim.measurements,
      conditions: claim.conditions,
      dataHash: claim.dataHash,
    },
    recommendedConfidence: CLAIM_TYPE_MAX_CONFIDENCE.measurement,
    dependsOn: claim.dependsOn,
  });

  if (claim.dependsOn) {
    decomposition.dependencyGraph.set(index, claim.dependsOn);
  }

  return index;
}

/**
 * Add a statistical claim
 */
export function addStatisticalClaim(
  decomposition: PaperDecomposition,
  claim: {
    statisticalTest: string;
    testDetails?: string;
    comparison: string;
    location: string;
    pValue?: number;
    effectSize?: number;
    effectSizeType?: string;
    confidenceInterval?: { lower: number; upper: number; level: number };
    sampleSize?: number;
    degreesOfFreedom?: number;
    testStatistic?: number;
    dependsOn?: number[];
  },
): number {
  const index = decomposition.claims.length;

  decomposition.claims.push({
    type: "statistical",
    claimText: `${claim.comparison}: ${claim.statisticalTest}${claim.pValue ? `, p=${claim.pValue}` : ""}`,
    location: claim.location,
    schemaData: {
      paperDoi: decomposition.paper.doi,
      statisticalTest: claim.statisticalTest,
      testDetails: claim.testDetails,
      comparison: claim.comparison,
      location: claim.location,
      pValue: claim.pValue,
      effectSize: claim.effectSize,
      effectSizeType: claim.effectSizeType,
      confidenceInterval: claim.confidenceInterval,
      sampleSize: claim.sampleSize,
      degreesOfFreedom: claim.degreesOfFreedom,
      testStatistic: claim.testStatistic,
    },
    recommendedConfidence: CLAIM_TYPE_MAX_CONFIDENCE.statistical,
    dependsOn: claim.dependsOn,
  });

  if (claim.dependsOn) {
    decomposition.dependencyGraph.set(index, claim.dependsOn);
  }

  return index;
}

/**
 * Add a causal claim
 */
export function addCausalClaim(
  decomposition: PaperDecomposition,
  claim: {
    cause: string;
    effect: string;
    relationship:
      | "causes"
      | "prevents"
      | "increases"
      | "decreases"
      | "necessary_for"
      | "sufficient_for"
      | "modulates";
    evidenceType: string;
    proposedMechanism?: string;
    confoundsAddressed?: string[];
    confoundsRemaining?: string[];
    location: string;
    dependsOn: number[]; // Causal claims always depend on evidence
  },
): number {
  const index = decomposition.claims.length;

  decomposition.claims.push({
    type: "causal",
    claimText: `${claim.cause} ${claim.relationship} ${claim.effect}`,
    location: claim.location,
    schemaData: {
      cause: claim.cause,
      effect: claim.effect,
      relationship: claim.relationship,
      evidenceType: claim.evidenceType,
      proposedMechanism: claim.proposedMechanism,
      confoundsAddressed: claim.confoundsAddressed,
      confoundsRemaining: claim.confoundsRemaining,
      paperDoi: decomposition.paper.doi,
      supportingClaimIds: [], // Will be filled after submission
    },
    recommendedConfidence: CLAIM_TYPE_MAX_CONFIDENCE.causal,
    dependsOn: claim.dependsOn,
    validatorNotes:
      "Causal claims require careful evaluation of confounds and evidence type.",
  });

  decomposition.dependencyGraph.set(index, claim.dependsOn);

  return index;
}

/**
 * Add an interpretation/conclusion claim
 */
export function addInterpretationClaim(
  decomposition: PaperDecomposition,
  claim: {
    claimText: string;
    location: string;
    isConclusion: boolean;
    dependsOn: number[];
  },
): number {
  const index = decomposition.claims.length;

  decomposition.claims.push({
    type: claim.isConclusion ? "conclusion" : "interpretation",
    claimText: claim.claimText,
    location: claim.location,
    schemaData: {
      field: decomposition.paper.field,
      claim: claim.claimText,
      methodology: "interpretation",
      paperDoi: decomposition.paper.doi,
      peerReviewStatus: decomposition.paper.peerReviewStatus,
    },
    recommendedConfidence: claim.isConclusion
      ? CLAIM_TYPE_MAX_CONFIDENCE.conclusion
      : CLAIM_TYPE_MAX_CONFIDENCE.interpretation,
    dependsOn: claim.dependsOn,
  });

  decomposition.dependencyGraph.set(index, claim.dependsOn);

  return index;
}

// ============ Submission Helpers ============

/**
 * Compute topological order for claim submission
 *
 * Claims must be submitted in order so that dependencies
 * are available when dependent claims are verified.
 */
export function computeSubmissionOrder(
  decomposition: PaperDecomposition,
): number[] {
  const order: number[] = [];
  const visited = new Set<number>();
  const visiting = new Set<number>();

  function visit(index: number): void {
    if (visited.has(index)) return;
    if (visiting.has(index)) {
      throw new Error(`Circular dependency detected at claim ${index}`);
    }

    visiting.add(index);

    const deps = decomposition.dependencyGraph.get(index) || [];
    for (const dep of deps) {
      visit(dep);
    }

    visiting.delete(index);
    visited.add(index);
    order.push(index);
  }

  for (let i = 0; i < decomposition.claims.length; i++) {
    visit(i);
  }

  decomposition.submissionOrder = order;
  return order;
}

/**
 * Generate ClaimSubmission objects for all claims
 */
export function generateClaimSubmissions(
  decomposition: PaperDecomposition,
  stake: bigint,
  paperReference: Omit<Reference, "id">,
): ClaimSubmission[] {
  // Ensure submission order is computed
  if (decomposition.submissionOrder.length === 0) {
    computeSubmissionOrder(decomposition);
  }

  return decomposition.submissionOrder.map((index) => {
    const claim = decomposition.claims[index];
    const schemaId = CLAIM_TYPE_SCHEMA[claim.type];

    return {
      schemaId,
      claimData: claim.schemaData,
      claim: claim.claimText,
      type: "empirical" as const,
      domain: decomposition.paper.field,
      references: [paperReference],
      proposedConfidence: claim.recommendedConfidence,
      stake,
    };
  });
}

// ============ Example Usage ============

/**
 * Example: Decompose a neuroscience paper
 *
 * ```typescript
 * const decomposition = createPaperDecomposition({
 *   doi: '10.1038/s41586-024-00001-1',
 *   title: 'Sleep enhances memory consolidation via hippocampal replay',
 *   authors: ['Smith, J.', 'Jones, A.'],
 *   journal: 'Nature',
 *   publicationDate: '2024-01-15',
 *   field: 'neuroscience',
 *   subfield: 'memory',
 *   peerReviewStatus: 'peer_reviewed',
 * });
 *
 * // Add methodology claim
 * const methodIdx = addMethodologyClaim(decomposition, {
 *   methodologyType: 'sample_selection',
 *   claimText: 'Study used N=24 C57BL/6 mice aged 8-12 weeks',
 *   location: 'Methods, section 2.1',
 *   sampleSize: 24,
 *   sampleCharacteristics: { strain: 'C57BL/6', ageWeeks: [8, 12] },
 * });
 *
 * // Add measurement claim (depends on methodology)
 * const measureIdx = addMeasurementClaim(decomposition, {
 *   experimentType: 'behavioral_maze_test',
 *   hypothesis: 'Sleep deprivation impairs spatial memory',
 *   result: 'Sleep-deprived group showed 43% reduction in accuracy',
 *   location: 'Results, paragraph 2',
 *   measurements: [
 *     { name: 'control_accuracy', value: 0.82, uncertainty: 0.07 },
 *     { name: 'deprived_accuracy', value: 0.47, uncertainty: 0.12 },
 *   ],
 *   dependsOn: [methodIdx],
 * });
 *
 * // Add statistical claim
 * const statsIdx = addStatisticalClaim(decomposition, {
 *   statisticalTest: 't_test',
 *   testDetails: 'two-tailed independent samples t-test',
 *   comparison: 'control vs sleep-deprived accuracy',
 *   location: 'Results, paragraph 2',
 *   pValue: 0.001,
 *   effectSize: 1.2,
 *   effectSizeType: 'cohens_d',
 *   dependsOn: [measureIdx],
 * });
 *
 * // Add causal claim (depends on all evidence)
 * addCausalClaim(decomposition, {
 *   cause: 'hippocampal replay during sleep',
 *   effect: 'memory consolidation',
 *   relationship: 'necessary_for',
 *   evidenceType: 'experimental_intervention',
 *   location: 'Discussion, paragraph 1',
 *   dependsOn: [measureIdx, statsIdx],
 * });
 *
 * // Generate submissions in correct order
 * const submissions = generateClaimSubmissions(decomposition, stake, paperRef);
 * ```
 */
