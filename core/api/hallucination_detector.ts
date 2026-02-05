/**
 * Hallucination Detector
 *
 * Core detection algorithms for identifying ungrounded AI agent outputs.
 * Verifies agent claims against on-chain verified facts before allowing submission.
 *
 * PP HIERARCHY:
 *   TRUTH: Detects claims ungrounded in verified facts
 *   UNDERSTANDING: Analyzes reasoning chains for drift from knowledge
 *   BEAUTY: Clean algorithm separation, coherent flow
 *   JUSTICE: Placed in core/api alongside agentic system
 *   CREATIVITY: Novel integration of ontology + PP + confidence
 */

import type {
  AgentOutputClaim,
  GroundingCitation,
  OutputVerificationResult,
  ReasoningChainAnalysis,
  ReasoningNode,
  AntiHallucinationConfig,
  CitationRelationship,
  VerificationBreakdown,
  ReasoningStepType,
} from "./output_verification.js";
import { DEFAULT_ANTI_HALLUCINATION_CONFIG } from "./output_verification.js";
import type {
  OntologicalStratum,
  OntologicalDomain,
} from "../knowledge/ontology/types.js";
import { STRATUM_CONFIDENCE_CEILING } from "../knowledge/ontology/types.js";
import { getStratumDistance } from "../knowledge/ontology/strata.js";
import { calculatePPScore, type PPScore, type PPSignal } from "./philosophy.js";

// ============================================================================
// Fact State Type (for lookup function)
// ============================================================================

/**
 * Minimal fact state needed for verification.
 */
export interface FactStateForVerification {
  id: string;
  claim: string;
  confidence: number;
  domain: string;
  stratum?: OntologicalStratum;
}

// ============================================================================
// Main Verification Function
// ============================================================================

/**
 * Verify an agent's output claim against on-chain facts.
 *
 * The verification flow:
 * 1. Retrieve cited facts from knowledge layer
 * 2. Validate citations against the claim
 * 3. Calculate groundedness score
 * 4. Apply ontological constraints
 * 5. Calculate PP scores
 * 6. Detect signals (ugliness, injustice, stagnation)
 * 7. Return comprehensive result
 */
export async function verifyAgentOutput(
  claim: AgentOutputClaim,
  factLookup: (factId: string) => Promise<FactStateForVerification | undefined>,
  domainLookup: (domainId: string) => Promise<OntologicalDomain | undefined>,
  config: AntiHallucinationConfig = DEFAULT_ANTI_HALLUCINATION_CONFIG,
): Promise<OutputVerificationResult> {
  // 1. Retrieve and validate citations
  const citations = await validateCitations(claim, factLookup, domainLookup);

  // 2. Calculate groundedness score
  const groundednessScore = calculateGroundednessScore(
    claim,
    citations,
    config,
  );

  // 3. Get ontological constraints
  const domain = await domainLookup(claim.domain);
  const ontologyCeiling = domain
    ? STRATUM_CONFIDENCE_CEILING[domain.stratum]
    : STRATUM_CONFIDENCE_CEILING.social; // Default to social stratum ceiling (0.85)

  // 4. Calculate cross-stratum discount
  const crossStratumDiscount = calculateCrossStratumDiscountForCitations(
    citations,
    domain,
    config,
  );

  // 5. Calculate effective confidence
  const effectiveConfidence = calculateEffectiveConfidence(
    claim.selfConfidence,
    groundednessScore,
    ontologyCeiling,
    crossStratumDiscount,
    citations,
    config,
  );

  // 6. Analyze reasoning chain
  const reasoningChainAnalysis = analyzeReasoningChain(
    claim,
    citations,
    config,
  );

  // 7. Calculate PP scores
  const ppScores = calculateOutputPPScores(
    claim,
    citations,
    groundednessScore,
    reasoningChainAnalysis,
  );

  // 8. Detect signals
  const signals = detectOutputSignals(
    claim,
    citations,
    groundednessScore,
    ppScores,
  );

  // 9. Build verification breakdown
  const breakdown = buildVerificationBreakdown(
    citations,
    claim.selfConfidence,
    effectiveConfidence,
    crossStratumDiscount,
    ppScores,
  );

  // 10. Determine if passed
  const { passed, failureReasons } = determinePassStatus(
    groundednessScore,
    ppScores,
    reasoningChainAnalysis,
    config,
  );

  // 11. Generate suggestions
  const suggestions = generateImprovementSuggestions(
    claim,
    citations,
    groundednessScore,
    reasoningChainAnalysis,
    failureReasons,
  );

  return {
    outputId: claim.outputId,
    passed,
    groundednessScore,
    effectiveConfidence,
    citations,
    ontologyCeiling,
    crossStratumDiscount,
    ppScores,
    signals,
    reasoningChainAnalysis,
    breakdown,
    failureReasons: passed ? undefined : failureReasons,
    suggestions: suggestions.length > 0 ? suggestions : undefined,
  };
}

// ============================================================================
// Citation Validation
// ============================================================================

/**
 * Validate citations and calculate grounding strength for each.
 */
async function validateCitations(
  claim: AgentOutputClaim,
  factLookup: (factId: string) => Promise<FactStateForVerification | undefined>,
  domainLookup: (domainId: string) => Promise<OntologicalDomain | undefined>,
): Promise<GroundingCitation[]> {
  const citations: GroundingCitation[] = [];

  for (const factId of claim.citedFactIds) {
    const fact = await factLookup(factId);
    if (!fact) continue;

    const factDomain = await domainLookup(fact.domain);
    // Default to 'social' stratum if unknown (conservative confidence ceiling)
    const factStratum: OntologicalStratum =
      fact.stratum ?? factDomain?.stratum ?? "social";

    // Calculate semantic similarity
    const semanticSimilarity = calculateSemanticSimilarity(
      claim.claim,
      fact.claim,
    );

    // Determine relationship type
    const relationship = determineRelationship(
      claim.claim,
      fact.claim,
      semanticSimilarity,
    );

    // Calculate grounding strength
    const groundingStrength = calculateGroundingStrength(
      semanticSimilarity,
      relationship,
      fact.confidence,
    );

    citations.push({
      factId,
      factClaim: fact.claim,
      factConfidence: fact.confidence,
      factDomain: fact.domain,
      factStratum,
      relationship,
      groundingStrength,
      semanticSimilarity,
    });
  }

  // Sort by grounding strength (strongest first)
  return citations.sort((a, b) => b.groundingStrength - a.groundingStrength);
}

/**
 * Calculate semantic similarity between claim and fact.
 *
 * Uses Jaccard similarity on tokens. Production implementation
 * would use embeddings for better semantic matching.
 */
function calculateSemanticSimilarity(claim: string, factClaim: string): number {
  // Normalize and tokenize
  const claimTokens = new Set(
    claim
      .toLowerCase()
      .split(/\W+/)
      .filter((t) => t.length > 2),
  );
  const factTokens = new Set(
    factClaim
      .toLowerCase()
      .split(/\W+/)
      .filter((t) => t.length > 2),
  );

  if (claimTokens.size === 0 || factTokens.size === 0) return 0;

  // Jaccard similarity: |intersection| / |union|
  const intersection = new Set(
    [...claimTokens].filter((t) => factTokens.has(t)),
  );
  const union = new Set([...claimTokens, ...factTokens]);

  return union.size > 0 ? intersection.size / union.size : 0;
}

/**
 * Determine the relationship between claim and cited fact.
 */
function determineRelationship(
  claim: string,
  factClaim: string,
  similarity: number,
): CitationRelationship {
  // Check for contradiction indicators
  const contradictionWords = [
    "not",
    "never",
    "false",
    "incorrect",
    "wrong",
    "opposite",
  ];
  const claimLower = claim.toLowerCase();
  const factLower = factClaim.toLowerCase();

  const claimHasContradiction = contradictionWords.some((w) =>
    claimLower.includes(w),
  );
  const factHasContradiction = contradictionWords.some((w) =>
    factLower.includes(w),
  );

  // XOR on contradiction words suggests contradiction
  if (claimHasContradiction !== factHasContradiction && similarity > 0.3) {
    return "contradicts";
  }

  // High similarity = direct support
  if (similarity >= 0.7) return "directly_supports";
  if (similarity >= 0.4) return "partially_supports";
  if (similarity >= 0.2) return "implies";
  if (similarity >= 0.1) return "analogous";

  return "contradicts"; // Very low similarity provides no grounding
}

/**
 * Calculate grounding strength from components.
 */
function calculateGroundingStrength(
  similarity: number,
  relationship: CitationRelationship,
  factConfidence: number,
): number {
  // Relationship multipliers
  const relationshipMultiplier: Record<CitationRelationship, number> = {
    directly_supports: 1.0,
    partially_supports: 0.7,
    implies: 0.5,
    analogous: 0.3,
    contradicts: 0.0, // Contradictions provide no grounding
  };

  return similarity * relationshipMultiplier[relationship] * factConfidence;
}

// ============================================================================
// Groundedness Calculation
// ============================================================================

/**
 * Calculate overall groundedness score.
 *
 * Formula: 0.4 * bestStrength + 0.3 * coverage + 0.3 * avgQuality
 */
function calculateGroundednessScore(
  claim: AgentOutputClaim,
  citations: GroundingCitation[],
  config: AntiHallucinationConfig,
): number {
  // No citations = no grounding (unless speculation allowed)
  if (citations.length === 0) {
    return config.allowSpeculation && claim.outputType === "speculation"
      ? 0.1
      : 0;
  }

  // Filter out contradicting citations
  const validCitations = citations.filter(
    (c) => c.relationship !== "contradicts",
  );

  if (validCitations.length === 0) {
    return 0; // Only contradictions = no grounding
  }

  // Components:
  // 1. Best citation strength (40%)
  const bestStrength = Math.max(
    ...validCitations.map((c) => c.groundingStrength),
  );

  // 2. Citation coverage (30%)
  const coverage = calculateCitationCoverage(validCitations);

  // 3. Average citation quality (30%)
  const avgQuality = calculateAverageCitationQuality(validCitations);

  return bestStrength * 0.4 + coverage * 0.3 + avgQuality * 0.3;
}

/**
 * Calculate citation coverage (how many valid citations).
 */
function calculateCitationCoverage(citations: GroundingCitation[]): number {
  // Count citations with reasonable strength
  const strongCitations = citations.filter((c) => c.groundingStrength > 0.2);

  // More citations = better coverage (diminishing returns)
  // 3+ citations = full coverage
  return Math.min(1, strongCitations.length / 3);
}

/**
 * Calculate average citation quality.
 */
function calculateAverageCitationQuality(
  citations: GroundingCitation[],
): number {
  if (citations.length === 0) return 0;

  return (
    citations.reduce((sum, c) => sum + c.groundingStrength, 0) /
    citations.length
  );
}

// ============================================================================
// Cross-Stratum Discount
// ============================================================================

/**
 * Calculate cross-stratum discount based on citations.
 *
 * Each stratum boundary crossed reduces confidence by crossStratumPenalty.
 */
function calculateCrossStratumDiscountForCitations(
  citations: GroundingCitation[],
  claimDomain: OntologicalDomain | undefined,
  config: AntiHallucinationConfig,
): number {
  if (!claimDomain || citations.length === 0) return 1.0;

  // Find maximum stratum distance crossed
  let maxDistance = 0;
  for (const citation of citations) {
    if (citation.relationship === "contradicts") continue;
    const distance = getStratumDistance(
      claimDomain.stratum,
      citation.factStratum,
    );
    maxDistance = Math.max(maxDistance, distance);
  }

  // Apply penalty per stratum boundary
  return Math.pow(config.crossStratumPenalty, maxDistance);
}

// ============================================================================
// Effective Confidence
// ============================================================================

/**
 * Calculate effective confidence after all adjustments.
 */
function calculateEffectiveConfidence(
  selfConfidence: number,
  groundednessScore: number,
  ontologyCeiling: number,
  crossStratumDiscount: number,
  citations: GroundingCitation[],
  config: AntiHallucinationConfig,
): number {
  // Base: weighted average of self-confidence and grounded confidence
  const groundedConfidence =
    citations.length > 0
      ? Math.max(
          ...citations
            .filter((c) => c.relationship !== "contradicts")
            .map((c) => c.factConfidence * c.groundingStrength),
        ) || 0
      : 0;

  // Weight self-confidence by groundedness
  // Highly grounded = trust grounded confidence more
  // Poorly grounded = penalize self-confidence
  const baseConfidence =
    groundedConfidence * groundednessScore +
    selfConfidence * (1 - groundednessScore) * 0.5;

  // Apply cross-stratum discount
  let effective = baseConfidence * crossStratumDiscount;

  // Apply ontology ceiling if configured
  if (config.enforceOntologyCeilings) {
    effective = Math.min(effective, ontologyCeiling);
  }

  // Floor at 0.01 (never completely zero for non-contradiction)
  return Math.max(0.01, Math.min(1, effective));
}

// ============================================================================
// Reasoning Chain Analysis
// ============================================================================

/**
 * Analyze the reasoning chain from citations to claim.
 */
function analyzeReasoningChain(
  claim: AgentOutputClaim,
  citations: GroundingCitation[],
  config: AntiHallucinationConfig,
): ReasoningChainAnalysis {
  const chain: ReasoningNode[] = [];
  let stepNumber = 1;

  // Add grounded facts first (distance 0)
  const validCitations = citations.filter(
    (c) => c.relationship !== "contradicts",
  );
  for (const citation of validCitations) {
    chain.push({
      stepNumber: stepNumber++,
      claim: citation.factClaim,
      stepType: "grounded_fact" as ReasoningStepType,
      groundedFactId: citation.factId,
      confidence: citation.factConfidence,
      groundingDistance: 0,
      stratum: citation.factStratum,
    });
  }

  // Determine grounding distance for the output claim
  const hasGrounding = validCitations.length > 0;
  const groundingDistance = hasGrounding
    ? 1
    : config.maxSpeculationDistance + 1;

  // Determine step type based on output type and grounding
  let stepType: ReasoningStepType;
  if (!hasGrounding) {
    stepType = "speculation";
  } else if (claim.outputType === "derivation") {
    stepType = "inference";
  } else if (claim.outputType === "speculation") {
    stepType = "speculation";
  } else {
    // Assertion with grounding = inference from facts
    stepType = "inference";
  }

  // Add the claim itself
  chain.push({
    stepNumber: stepNumber++,
    claim: claim.claim,
    stepType,
    inferenceType:
      stepType === "inference"
        ? claim.outputType === "derivation"
          ? "deductive"
          : "inductive"
        : undefined,
    confidence: claim.selfConfidence,
    groundingDistance,
  });

  // Calculate metrics
  const groundedNodeCount = chain.filter(
    (n) => n.stepType === "grounded_fact",
  ).length;
  const inferenceNodeCount = chain.filter(
    (n) => n.stepType === "inference",
  ).length;
  const assumptionNodeCount = chain.filter(
    (n) => n.stepType === "assumption",
  ).length;
  const speculationNodeCount = chain.filter(
    (n) => n.stepType === "speculation",
  ).length;

  // Find strata traversed
  const strataTraversed = [
    ...new Set(chain.filter((n) => n.stratum).map((n) => n.stratum!)),
  ] as OntologicalStratum[];

  // Calculate confidence decay
  const initialConfidence = chain[0]?.confidence ?? 0;
  const finalConfidence = chain[chain.length - 1]?.confidence ?? 0;
  const confidenceDecay =
    initialConfidence > 0 ? finalConfidence / initialConfidence : 0;

  // Calculate drift score (normalized by max grounding distance)
  const maxDistance = Math.max(...chain.map((n) => n.groundingDistance));
  const driftScore = maxDistance / config.maxGroundingDistance;

  return {
    chain,
    chainLength: chain.length,
    groundedNodeCount,
    inferenceNodeCount,
    assumptionNodeCount,
    speculationNodeCount,
    groundingRatio: chain.length > 0 ? groundedNodeCount / chain.length : 0,
    maxGroundingDistance: maxDistance,
    crossesStratumBoundaries: strataTraversed.length > 1,
    strataTraversed,
    confidenceDecay,
    driftScore,
  };
}

// ============================================================================
// PP Integration
// ============================================================================

/**
 * Calculate PP scores for the output.
 */
function calculateOutputPPScores(
  claim: AgentOutputClaim,
  citations: GroundingCitation[],
  groundednessScore: number,
  reasoningChain: ReasoningChainAnalysis,
): PPScore {
  // Truth: Based on groundedness in verified facts
  const truth = groundednessScore;

  // Understanding: Based on citation quality and chain coherence
  const validCitations = citations.filter(
    (c) => c.relationship !== "contradicts",
  );
  const understanding =
    validCitations.length > 0
      ? calculateAverageCitationQuality(validCitations) *
        reasoningChain.groundingRatio
      : 0.1;

  // Beauty: Based on chain coherence and stratum consistency
  const beauty = reasoningChain.crossesStratumBoundaries
    ? Math.max(0.3, 0.8 / reasoningChain.strataTraversed.length)
    : 0.8;

  // Justice: Based on proper domain placement (citations match claim domain)
  const domainMatch = citations.some((c) => c.factDomain === claim.domain);
  const justice = domainMatch ? 0.9 : 0.5;

  // Creativity: Based on novel derivations (but penalized if ungrounded)
  let creativity: number;
  if (claim.outputType === "derivation" && groundednessScore > 0.5) {
    creativity = 0.7;
  } else if (claim.outputType === "speculation" && groundednessScore > 0.3) {
    creativity = 0.5;
  } else if (claim.outputType === "assertion" && groundednessScore > 0.7) {
    creativity = 0.3; // Assertions are not very creative
  } else {
    creativity = 0.1;
  }

  return calculatePPScore({
    truth,
    understanding,
    beauty,
    justice,
    creativity,
  });
}

/**
 * Detect PP signals (ugliness, injustice, stagnation).
 */
function detectOutputSignals(
  claim: AgentOutputClaim,
  citations: GroundingCitation[],
  groundednessScore: number,
  _ppScores: PPScore, // Used for potential future signal detection based on PP dimensions
): PPSignal[] {
  const signals: PPSignal[] = [];

  // UGLINESS: Low groundedness signals incomplete understanding
  if (groundednessScore < 0.5) {
    signals.push({
      type: "ugliness",
      description: "Output has low groundedness in verified facts",
      understandingNeeded: "More citations or stronger grounding required",
      suggestion: "Query for supporting facts before making this claim",
    });
  }

  // INJUSTICE: Wrong domain placement
  const validCitations = citations.filter(
    (c) => c.relationship !== "contradicts",
  );
  if (
    validCitations.length > 0 &&
    !validCitations.some((c) => c.factDomain === claim.domain)
  ) {
    signals.push({
      type: "injustice",
      description: "Claim domain does not match cited facts",
      misplacedElement: "Domain classification",
      correctPlacement: validCitations[0]?.factDomain || "Unknown",
    });
  }

  // STAGNATION: No citations at all for non-speculation
  if (citations.length === 0 && claim.outputType !== "speculation") {
    signals.push({
      type: "stagnation",
      description: "No grounding citations provided",
      unchartedTerritory: claim.domain,
      creativePath: "Submit as speculation or find supporting facts",
    });
  }

  return signals;
}

// ============================================================================
// Verification Breakdown
// ============================================================================

/**
 * Build detailed verification breakdown.
 */
function buildVerificationBreakdown(
  citations: GroundingCitation[],
  selfConfidence: number,
  effectiveConfidence: number,
  crossStratumDiscount: number,
  ppScores: PPScore,
): VerificationBreakdown {
  const validCitations = citations.filter(
    (c) => c.relationship !== "contradicts",
  );

  return {
    citationCoverage: calculateCitationCoverage(validCitations),
    averageCitationQuality: calculateAverageCitationQuality(validCitations),
    maxCitationStrength:
      validCitations.length > 0
        ? Math.max(...validCitations.map((c) => c.groundingStrength))
        : 0,
    minCitationStrength:
      validCitations.length > 0
        ? Math.min(...validCitations.map((c) => c.groundingStrength))
        : 0,
    confidenceCalibration: 1 - Math.abs(selfConfidence - effectiveConfidence),
    ontologicalConsistency: crossStratumDiscount,
    ppDimensions: {
      truth: ppScores.truth,
      understanding: ppScores.understanding,
      beauty: ppScores.beauty,
      justice: ppScores.justice,
      creativity: ppScores.creativity,
    },
  };
}

// ============================================================================
// Pass/Fail Determination
// ============================================================================

/**
 * Determine if output passes verification.
 */
function determinePassStatus(
  groundednessScore: number,
  ppScores: PPScore,
  reasoningChain: ReasoningChainAnalysis,
  config: AntiHallucinationConfig,
): { passed: boolean; failureReasons: string[] } {
  const failureReasons: string[] = [];

  // Check groundedness threshold
  if (groundednessScore < config.minGroundednessScore) {
    failureReasons.push(
      `Groundedness score ${groundednessScore.toFixed(2)} below minimum ${config.minGroundednessScore}`,
    );
  }

  // Check PP truth score
  if (ppScores.truth < config.minTruthScore) {
    failureReasons.push(
      `Truth score ${ppScores.truth.toFixed(2)} below minimum ${config.minTruthScore}`,
    );
  }

  // Check grounding distance
  if (reasoningChain.maxGroundingDistance > config.maxGroundingDistance) {
    failureReasons.push(
      `Grounding distance ${reasoningChain.maxGroundingDistance} exceeds maximum ${config.maxGroundingDistance}`,
    );
  }

  // Check drift threshold
  if (reasoningChain.driftScore > 1.0) {
    failureReasons.push(
      `Drift score ${reasoningChain.driftScore.toFixed(2)} indicates excessive drift from grounded knowledge`,
    );
  }

  // Speculation checks
  if (reasoningChain.speculationNodeCount > 0 && !config.allowSpeculation) {
    failureReasons.push("Speculation not allowed in current configuration");
  }

  return {
    passed: failureReasons.length === 0,
    failureReasons,
  };
}

// ============================================================================
// Improvement Suggestions
// ============================================================================

/**
 * Generate suggestions for improving groundedness.
 */
function generateImprovementSuggestions(
  claim: AgentOutputClaim,
  citations: GroundingCitation[],
  groundednessScore: number,
  reasoningChain: ReasoningChainAnalysis,
  _failureReasons: string[],
): string[] {
  const suggestions: string[] = [];

  // No citations
  if (citations.length === 0) {
    suggestions.push(
      "Query the knowledge layer for facts supporting this claim",
    );
    suggestions.push(
      `Search domain "${claim.domain}" for related verified facts`,
    );
  }

  // Weak citations
  if (groundednessScore < 0.5 && citations.length > 0) {
    suggestions.push(
      "Cited facts have weak relationship to claim - find more directly relevant facts",
    );
  }

  // Cross-stratum issues
  if (
    reasoningChain.crossesStratumBoundaries &&
    reasoningChain.strataTraversed.length > 2
  ) {
    suggestions.push("Consider gateway domains to bridge stratum boundaries");
  }

  // High drift
  if (reasoningChain.driftScore > 0.5) {
    suggestions.push(
      "Reasoning has drifted far from grounded knowledge - re-anchor with fresh fact queries",
    );
  }

  // Overconfident
  if (claim.selfConfidence > groundednessScore + 0.3) {
    suggestions.push(
      "Self-confidence exceeds grounding - calibrate confidence to evidence strength",
    );
  }

  // Speculation without marking
  if (claim.outputType === "assertion" && groundednessScore < 0.3) {
    suggestions.push(
      "Consider marking this output as speculation if grounding cannot be improved",
    );
  }

  return suggestions;
}

// ============================================================================
// Utility Exports
// ============================================================================

/**
 * Quick check if a claim has sufficient grounding.
 */
export function hasMinimalGrounding(
  citedFactIds: string[],
  _config: AntiHallucinationConfig = DEFAULT_ANTI_HALLUCINATION_CONFIG,
): boolean {
  // Must have at least one citation
  if (citedFactIds.length === 0) return false;

  // For quick check, assume presence of citations is minimal grounding
  return true;
}

/**
 * Calculate quick groundedness estimate without full verification.
 */
export function estimateGroundedness(citedFactCount: number): number {
  // Quick estimate: diminishing returns after 3 citations
  return Math.min(1, citedFactCount / 3);
}
