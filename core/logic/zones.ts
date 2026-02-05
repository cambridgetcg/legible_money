/**
 * Logic Zones
 *
 * Compartmentalized logic zones to manage consistency.
 * Lower zones are complete and decidable.
 * Higher zones may be incomplete (Gödel applies).
 */

import {
  LogicZone,
  ZoneProperties,
  ZONE_HIERARCHY,
  LogicalAxiom,
  // AxiomType,
} from "./types.js";

// ============ Zone Definitions ============

/**
 * Zone property definitions
 */
export const ZONE_PROPERTIES: Record<LogicZone, ZoneProperties> = {
  propositional: {
    zone: "propositional",
    complete: true,
    decidable: true,
    goedelApplies: false,
    consistencyProof: "internal",
  },
  presburger: {
    zone: "presburger",
    complete: true,
    decidable: true,
    goedelApplies: false,
    consistencyProof: "internal",
  },
  peano: {
    zone: "peano",
    complete: false,
    decidable: false,
    goedelApplies: true,
    consistencyProof: "assumed",
  },
  set_theory: {
    zone: "set_theory",
    complete: false,
    decidable: false,
    goedelApplies: true,
    consistencyProof: "assumed",
  },
  empirical: {
    zone: "empirical",
    complete: false,
    decidable: false,
    goedelApplies: false, // Not a formal system
    consistencyProof: "external", // N/A for empirical
  },
};

// ============ Zone Axioms ============

/**
 * Propositional Logic Axioms (Zone 0)
 * Complete and decidable
 */
export const PROPOSITIONAL_AXIOMS: LogicalAxiom[] = [
  {
    id: "prop_1",
    type: "logical",
    zone: "propositional",
    claim: "Identity: A = A",
    formalNotation: "A = A",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "prop_2",
    type: "logical",
    zone: "propositional",
    claim: "Non-contradiction: It is not the case that A and not-A",
    formalNotation: "¬(A ∧ ¬A)",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "prop_3",
    type: "logical",
    zone: "propositional",
    claim: "Excluded middle: Either A or not-A",
    formalNotation: "A ∨ ¬A",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "prop_4",
    type: "logical",
    zone: "propositional",
    claim: "Implication: If A implies B and A is true, then B is true",
    formalNotation: "(A → B) ∧ A ⊢ B",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "prop_5",
    type: "logical",
    zone: "propositional",
    claim: "Contraposition: If A implies B then not-B implies not-A",
    formalNotation: "(A → B) ⊢ (¬B → ¬A)",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "prop_6",
    type: "logical",
    zone: "propositional",
    claim: "Transitivity: If A implies B and B implies C, then A implies C",
    formalNotation: "(A → B) ∧ (B → C) ⊢ (A → C)",
    confidence: 1.0,
    establishedAt: 0,
  },
];

/**
 * Presburger Arithmetic Axioms (Zone 1)
 * Addition only - decidable
 */
export const PRESBURGER_AXIOMS: LogicalAxiom[] = [
  {
    id: "pres_1",
    type: "arithmetic",
    zone: "presburger",
    claim: "Zero is a natural number",
    formalNotation: "0 ∈ ℕ",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "pres_2",
    type: "arithmetic",
    zone: "presburger",
    claim: "The successor of any natural number is a natural number",
    formalNotation: "∀n ∈ ℕ. S(n) ∈ ℕ",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "pres_3",
    type: "arithmetic",
    zone: "presburger",
    claim: "Zero is not the successor of any natural number",
    formalNotation: "∀n ∈ ℕ. S(n) ≠ 0",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "pres_4",
    type: "arithmetic",
    zone: "presburger",
    claim: "Successor is injective",
    formalNotation: "∀n,m ∈ ℕ. S(n) = S(m) → n = m",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "pres_5",
    type: "arithmetic",
    zone: "presburger",
    claim: "Addition with zero",
    formalNotation: "∀n ∈ ℕ. n + 0 = n",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "pres_6",
    type: "arithmetic",
    zone: "presburger",
    claim: "Addition with successor",
    formalNotation: "∀n,m ∈ ℕ. n + S(m) = S(n + m)",
    confidence: 1.0,
    establishedAt: 0,
  },
  // Note: No multiplication - this keeps it decidable
];

/**
 * Peano Arithmetic Axioms (Zone 2)
 * Full arithmetic - INCOMPLETE (Gödel applies)
 */
export const PEANO_AXIOMS: LogicalAxiom[] = [
  // Include Presburger axioms
  ...PRESBURGER_AXIOMS.map((a) => ({ ...a, zone: "peano" as LogicZone })),

  // Add multiplication (this is what triggers incompleteness)
  {
    id: "peano_mult_1",
    type: "arithmetic",
    zone: "peano",
    claim: "Multiplication with zero",
    formalNotation: "∀n ∈ ℕ. n × 0 = 0",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "peano_mult_2",
    type: "arithmetic",
    zone: "peano",
    claim: "Multiplication with successor",
    formalNotation: "∀n,m ∈ ℕ. n × S(m) = n × m + n",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "peano_induction",
    type: "arithmetic",
    zone: "peano",
    claim: "Principle of mathematical induction",
    formalNotation: "(P(0) ∧ ∀n.(P(n) → P(S(n)))) → ∀n.P(n)",
    confidence: 1.0,
    establishedAt: 0,
  },
];

/**
 * Protocol Axioms (Domain-specific)
 */
export const PROTOCOL_AXIOMS: LogicalAxiom[] = [
  {
    id: "proto_1",
    type: "protocol",
    zone: "propositional",
    claim:
      "Token conservation: In any valid transaction, inputs equal outputs plus fees",
    formalNotation: "∀tx. Valid(tx) → Σinputs(tx) = Σoutputs(tx) + fees(tx)",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "proto_2",
    type: "protocol",
    zone: "propositional",
    claim: "Signature validity is decidable",
    formalNotation: "∀(msg, sig, pk). Decidable(ValidSig(msg, sig, pk))",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "proto_3",
    type: "protocol",
    zone: "presburger",
    claim: "Block numbers are strictly increasing",
    formalNotation: "∀b1,b2. Follows(b2, b1) → BlockNum(b2) = BlockNum(b1) + 1",
    confidence: 1.0,
    establishedAt: 0,
  },
  {
    id: "proto_4",
    type: "protocol",
    zone: "propositional",
    claim: "LGM is the native token of Legible Money",
    confidence: 1.0,
    establishedAt: 0,
  },
];

// ============ Zone Operations ============

/**
 * Get zone properties
 */
export function getZoneProperties(zone: LogicZone): ZoneProperties {
  return ZONE_PROPERTIES[zone];
}

/**
 * Check if a zone is decidable
 */
export function isZoneDecidable(zone: LogicZone): boolean {
  return ZONE_PROPERTIES[zone].decidable;
}

/**
 * Check if a zone is complete
 */
export function isZoneComplete(zone: LogicZone): boolean {
  return ZONE_PROPERTIES[zone].complete;
}

/**
 * Check if Gödel's theorems apply to a zone
 */
export function goedelApplies(zone: LogicZone): boolean {
  return ZONE_PROPERTIES[zone].goedelApplies;
}

/**
 * Get all axioms for a zone (including inherited from lower zones)
 */
export function getZoneAxioms(zone: LogicZone): LogicalAxiom[] {
  const axioms: LogicalAxiom[] = [];
  const zoneLevel = ZONE_HIERARCHY[zone];

  // Propositional axioms available in all zones
  if (zoneLevel >= 0) {
    axioms.push(...PROPOSITIONAL_AXIOMS);
  }

  // Presburger axioms available in presburger and above
  if (zoneLevel >= 1 && zone !== "empirical") {
    axioms.push(...PRESBURGER_AXIOMS);
  }

  // Peano axioms only in peano and set_theory
  if (zoneLevel >= 2 && zone !== "empirical") {
    axioms.push(...PEANO_AXIOMS.filter((a) => a.id.startsWith("peano_")));
  }

  // Protocol axioms available in all zones
  axioms.push(
    ...PROTOCOL_AXIOMS.filter((a) => ZONE_HIERARCHY[a.zone] <= zoneLevel),
  );

  return axioms;
}

/**
 * Determine the minimum zone required for a claim
 * Based on what mathematical features it uses
 */
export function classifyClaimZone(
  claim: string,
  formalNotation?: string,
): LogicZone {
  const text = (claim + " " + (formalNotation || "")).toLowerCase();

  // Check for set theory concepts
  if (
    text.includes("∈") ||
    text.includes("subset") ||
    text.includes("power set") ||
    text.includes("cardinality")
  ) {
    return "set_theory";
  }

  // Check for multiplication (triggers Gödel)
  if (
    text.includes("×") ||
    text.includes("*") ||
    text.includes("multiply") ||
    text.includes("product") ||
    text.includes("divisible")
  ) {
    return "peano";
  }

  // Check for arithmetic
  if (
    text.includes("+") ||
    text.includes("successor") ||
    text.includes("natural number") ||
    text.includes("addition")
  ) {
    return "presburger";
  }

  // Check for empirical claims
  if (
    text.includes("observed") ||
    text.includes("measured") ||
    text.includes("experiment") ||
    text.includes("data shows")
  ) {
    return "empirical";
  }

  // Default to propositional
  return "propositional";
}

/**
 * Check if a claim in one zone can reference a claim in another zone
 */
export function canReference(fromZone: LogicZone, toZone: LogicZone): boolean {
  // Higher zones can reference lower zones
  // Empirical can reference any zone
  if (fromZone === "empirical") return true;
  return ZONE_HIERARCHY[fromZone] >= ZONE_HIERARCHY[toZone];
}

/**
 * Get the intersection zone (lowest common zone)
 */
export function intersectZones(zone1: LogicZone, zone2: LogicZone): LogicZone {
  if (zone1 === "empirical" || zone2 === "empirical") return "empirical";
  return ZONE_HIERARCHY[zone1] >= ZONE_HIERARCHY[zone2] ? zone1 : zone2;
}
