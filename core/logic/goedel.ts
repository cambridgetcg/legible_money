/**
 * Gödelian Analysis
 *
 * Detection and handling of self-referential and
 * potentially undecidable statements.
 */

import {
  GoedelianStatus,
  Decidability,
  LogicZone,
  ResolutionPath,
  UndecidableResolution,
} from "./types.js";
import { goedelApplies } from "./zones.js";

// ============ Self-Reference Detection ============

/**
 * Patterns that indicate self-reference
 */
const SELF_REFERENCE_PATTERNS = [
  // Direct self-reference
  /this (statement|claim|fact|proposition)/i,
  /itself/i,
  /the (statement|claim|fact) (that |which )?(you are|is being) (reading|considering)/i,

  // Provability self-reference (Gödelian)
  /(this|the following) (is|cannot be|can be) (proven|provable|proved)/i,
  /unprovable within/i,
  /cannot be demonstrated/i,

  // Truth self-reference (Liar-like)
  /this (statement|claim) is (false|not true|a lie)/i,
  /I am (lying|false)/i,

  // System self-reference
  /the system (itself|in which|that contains)/i,
  /(complete|consistent|decidable) system/i,
  /prove (its own|their own) (consistency|completeness)/i,
];

/**
 * Patterns that indicate paradox (should be rejected)
 */
const PARADOX_PATTERNS = [
  // Liar paradox
  /this (statement|sentence) is false/i,
  /if this is true then it is false/i,

  // Russell-like paradox
  /the set of all sets that do not contain themselves/i,
  /contains itself if and only if it does not/i,

  // Grelling-Nelson paradox
  /heterological.*heterological/i,
];

/**
 * Patterns that indicate Gödelian sentences (undecidable, not paradoxical)
 */
const GOEDELIAN_PATTERNS = [
  // Gödelian self-reference about provability
  /this (statement|claim|proposition) (is not|cannot be) (proven|provable|proved) (in|within) (this|the) system/i,
  /unprovable (but|yet) true/i,
  /true but not provable/i,

  // System limitation statements
  /no (consistent|complete) system can prove/i,
  /(incompleteness|undecidability) of (this|the) system/i,
];

/**
 * Analyze a claim for self-reference and Gödelian properties
 */
export function analyzeGoedelianStatus(
  claim: string,
  formalNotation?: string,
): {
  status: GoedelianStatus;
  selfReferential: boolean;
  paradoxical: boolean;
  goedelian: boolean;
  details: string;
} {
  const text = claim + " " + (formalNotation || "");

  // Check for paradox (strongest - reject these)
  for (const pattern of PARADOX_PATTERNS) {
    if (pattern.test(text)) {
      return {
        status: "paradoxical",
        selfReferential: true,
        paradoxical: true,
        goedelian: false,
        details: "Contains logical paradox. Cannot be assigned truth value.",
      };
    }
  }

  // Check for Gödelian sentences
  for (const pattern of GOEDELIAN_PATTERNS) {
    if (pattern.test(text)) {
      return {
        status: "goedelian",
        selfReferential: true,
        paradoxical: false,
        goedelian: true,
        details:
          "Gödelian self-reference detected. May be true but unprovable within system.",
      };
    }
  }

  // Check for general self-reference
  for (const pattern of SELF_REFERENCE_PATTERNS) {
    if (pattern.test(text)) {
      return {
        status: "goedelian",
        selfReferential: true,
        paradoxical: false,
        goedelian: true,
        details: "Self-reference detected. Requires special handling.",
      };
    }
  }

  // No problematic patterns found
  return {
    status: "benign",
    selfReferential: false,
    paradoxical: false,
    goedelian: false,
    details: "No self-reference or Gödelian patterns detected.",
  };
}

/**
 * Determine if a claim might be undecidable
 */
export function assessDecidability(
  claim: string,
  zone: LogicZone,
  formalNotation?: string,
): {
  decidability: Decidability;
  reason: string;
  canBeResolved: boolean;
  suggestedResolutions: ResolutionPath[];
} {
  // Analyze Gödelian status
  const goedelAnalysis = analyzeGoedelianStatus(claim, formalNotation);

  // Paradoxical statements are rejected, not undecidable
  if (goedelAnalysis.paradoxical) {
    return {
      decidability: "unknown", // Can't even be evaluated
      reason: "Paradoxical statement rejected.",
      canBeResolved: false,
      suggestedResolutions: [],
    };
  }

  // Check if we're in a zone where Gödel applies
  const goedelAppliesHere = goedelApplies(zone);

  // Gödelian sentences in Gödel-applicable zones
  if (goedelAnalysis.goedelian && goedelAppliesHere) {
    return {
      decidability: "undecidable_self_ref",
      reason: "Gödelian self-reference in incomplete zone.",
      canBeResolved: true,
      suggestedResolutions: ["oracle", "consensus", "accept_undecidability"],
    };
  }

  // Non-Gödelian claims in decidable zones are decidable
  if (!goedelAppliesHere && !goedelAnalysis.goedelian) {
    return {
      decidability: "decidable",
      reason: "Claim is in decidable zone without self-reference.",
      canBeResolved: true,
      suggestedResolutions: [],
    };
  }

  // Empirical claims
  if (zone === "empirical") {
    return {
      decidability: "empirical",
      reason: "Empirical claim outside formal system.",
      canBeResolved: true,
      suggestedResolutions: ["oracle", "consensus"],
    };
  }

  // Claims in Gödel-applicable zones without detected issues
  // May still be undecidable, but we don't know yet
  return {
    decidability: "unknown",
    reason: "Decidability unknown. May be determined through proof search.",
    canBeResolved: true,
    suggestedResolutions: [],
  };
}

// ============ Undecidability Handling ============

/**
 * Handle an undecidable claim
 */
export function handleUndecidable(
  claimId: string,
  decidability: Decidability,
  chosenPath: ResolutionPath,
): Partial<UndecidableResolution> {
  return {
    claimId,
    originalDecidability: decidability as
      | "undecidable"
      | "undecidable_self_ref"
      | "undecidable_timeout",
    resolutionPath: chosenPath,
    finalConfidence: getDefaultConfidence(chosenPath),
  };
}

/**
 * Get default confidence for a resolution path
 */
function getDefaultConfidence(path: ResolutionPath): number {
  switch (path) {
    case "accept_undecidability":
      return 0.5; // Maximum uncertainty
    case "oracle":
      return 0.8; // Depends on oracle reliability
    case "consensus":
      return 0.7; // Community agreement
    case "new_axiom":
      return 1.0; // Axiom definition
    default:
      return 0.5;
  }
}

// ============ The Gödelian Promise ============

/**
 * The system's explicit acknowledgment of its limitations
 */
export const GOEDELIAN_PROMISE = `
This system explicitly acknowledges its Gödelian limitations:

1. INCOMPLETENESS: Some true statements cannot be proven within this system.
   This is not a bug but a mathematical certainty (Gödel's First Theorem).

2. CONSISTENCY: The system assumes but cannot prove its own consistency
   from within (Gödel's Second Theorem).

3. UNDECIDABILITY: Some statements will be neither provable nor disprovable.
   The system handles this gracefully with confidence intervals.

4. HONESTY: Rather than hiding these limits, we make them explicit.
   This builds deeper trust than false claims of completeness.

The founding truth cannot be formally proven. It transcends the system.
The founding axiom is accepted, not derived.
`;

/**
 * Check if a claim is about the system's own completeness/consistency
 */
export function isMetaSystemClaim(claim: string): boolean {
  const patterns = [
    /this system is (complete|consistent|decidable)/i,
    /(complete|consistent|decidable) system/i,
    /all (true statements|truths) can be proven/i,
    /every (statement|claim) is decidable/i,
  ];

  return patterns.some((p) => p.test(claim));
}

/**
 * Response to claims about system completeness
 */
export function respondToMetaClaim(claim: string): string {
  if (/complete/i.test(claim)) {
    return (
      "This system is provably incomplete by Gödel's First Theorem. " +
      "Some true statements cannot be proven within it."
    );
  }

  if (/consistent/i.test(claim)) {
    return (
      "This system assumes but cannot prove its own consistency " +
      "by Gödel's Second Theorem."
    );
  }

  if (/decidable/i.test(claim)) {
    return (
      "This system contains undecidable statements in zones where " +
      "Gödel's theorems apply (Peano arithmetic and above)."
    );
  }

  return (
    "This claim touches on fundamental limits of formal systems. " +
    "See the meta-axioms for the system's explicit acknowledgments."
  );
}
