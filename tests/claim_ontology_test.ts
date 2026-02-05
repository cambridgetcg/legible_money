/**
 * Test: Claim Integration with Ontological Tree
 *
 * Verifies that claims are validated against the ontological tree:
 * - Domain resolution
 * - Stratum ceiling application
 * - Cross-stratum discount
 * - Reference validation rules
 */

import { createDomainTree } from "../core/knowledge/ontology/tree.js";
import { GENESIS_DOMAINS } from "../core/knowledge/ontology/genesis.js";
import {
  validateClaimOntology,
  resolveDomain,
  validateClaimReferences,
  isValidClaimReference,
  getAdjustedClaimConfidence,
} from "../core/knowledge/ontology/claim_validation.js";
import { validateOntologicalReference } from "../core/knowledge/ontology/validation.js";
import { STRATUM_CONFIDENCE_CEILING } from "../core/knowledge/ontology/types.js";

// Create the domain tree
const tree = createDomainTree(GENESIS_DOMAINS);

console.log("=== Claim Ontology Integration Tests ===\n");

// ============================================================================
// Test 1: Domain Resolution
// ============================================================================

console.log("--- Test 1: Domain Resolution ---");

// Exact match
const exactDomain = resolveDomain("physics.mechanics.quantum", tree);
console.log(
  `physics.mechanics.quantum → ${exactDomain?.id} (${exactDomain?.stratum})`,
);

// Parent fallback - use a subdomain that doesn't exist
const fallbackDomain = resolveDomain(
  "physics.mechanics.quantum.superposition.decoherence",
  tree,
);
console.log(
  `physics.mechanics.quantum.superposition.decoherence → ${fallbackDomain?.id} (fallback to parent)`,
);

// Unknown domain falls back to root
const unknownDomain = resolveDomain("custom.unknown.domain", tree);
console.log(`custom.unknown.domain → ${unknownDomain?.id} (fallback to root)`);

// ============================================================================
// Test 2: Claim Validation - Valid Claims
// ============================================================================

console.log("\n--- Test 2: Valid Claim Validation ---");

// Claim in physics domain
const physicsClaim = validateClaimOntology(
  {
    domain: "physics.mechanics.quantum",
    proposedConfidence: 0.95,
    references: [],
  },
  tree,
);

console.log(`Physics claim (0.95 confidence):`);
console.log(`  Valid: ${physicsClaim.valid}`);
console.log(
  `  Adjusted confidence: ${physicsClaim.adjustedConfidence.toFixed(3)}`,
);
console.log(`  Stratum ceiling: ${physicsClaim.stratumCeiling}`);
console.log(
  `  Warnings: ${physicsClaim.warnings.length > 0 ? physicsClaim.warnings.join(", ") : "none"}`,
);

// Claim in social domain (lower ceiling)
const socialClaim = validateClaimOntology(
  {
    domain: "social.economics",
    proposedConfidence: 0.95,
    references: [],
  },
  tree,
);

console.log(`\nSocial claim (0.95 proposed → capped at 0.85):`);
console.log(`  Valid: ${socialClaim.valid}`);
console.log(
  `  Adjusted confidence: ${socialClaim.adjustedConfidence.toFixed(3)}`,
);
console.log(`  Stratum ceiling: ${socialClaim.stratumCeiling}`);

// ============================================================================
// Test 3: Reference Validation - Cross-Stratum Rules
// ============================================================================

console.log("\n--- Test 3: Reference Validation ---");

const chemDomain = tree.domainIndex.get("chemistry.organic")!.domain;
const bioCell = tree.domainIndex.get("biology.cellular")!.domain;
const physicsFund = tree.domainIndex.get("physics.particles.quarks")!.domain;

// Chemistry can support physics (lower stratum) - VALID
const chemToPhysics = validateOntologicalReference(
  chemDomain,
  physicsFund,
  "supports",
);
console.log(
  `chemistry.organic → physics.particles.quarks (supports): ${chemToPhysics.valid ? "VALID" : "INVALID"}`,
);

// Chemistry CANNOT support biology (higher stratum) - INVALID
const chemToBiology = validateOntologicalReference(
  chemDomain,
  bioCell,
  "supports",
);
console.log(
  `chemistry.organic → biology.cellular (supports): ${chemToBiology.valid ? "VALID" : "INVALID - " + chemToBiology.reason}`,
);

// Contradictions can cross any stratum boundary - VALID
const socialToPhysics = validateOntologicalReference(
  tree.domainIndex.get("social.economics")!.domain,
  physicsFund,
  "contradicts",
);
console.log(
  `social.economics → physics.particles.quarks (contradicts): ${socialToPhysics.valid ? "VALID" : "INVALID"}`,
);

// ============================================================================
// Test 4: Claim with Invalid Reference
// ============================================================================

console.log("\n--- Test 4: Claim with Invalid Reference ---");

const claimWithBadRef = validateClaimOntology(
  {
    domain: "chemistry.organic",
    proposedConfidence: 0.9,
    references: [
      {
        targetFactId: "fact-bio-001",
        targetDomain: "biology.cellular",
        referenceType: "supports", // Chemistry cannot support biology
      },
    ],
  },
  tree,
);

console.log(`Claim referencing higher stratum (chemistry → biology):`);
console.log(`  Valid: ${claimWithBadRef.valid}`);
console.log(`  Errors: ${claimWithBadRef.errors.join(", ")}`);

// ============================================================================
// Test 5: Cross-Stratum Discount
// ============================================================================

console.log("\n--- Test 5: Cross-Stratum Discount ---");

// Claim with dependency on lower stratum (should apply discount)
const claimWithDep = validateClaimOntology(
  {
    domain: "biology.cellular",
    proposedConfidence: 0.95,
    references: [],
    stratumDependencies: [
      {
        factId: "physics-fact-001",
        primaryStratum: "fundamental",
        dependentStrata: [],
        dependencyType: "emergence",
        dependencyStrength: 0.9,
      },
      {
        factId: "chem-fact-001",
        primaryStratum: "chemical",
        dependentStrata: [],
        dependencyType: "emergence",
        dependencyStrength: 0.85,
      },
    ],
  },
  tree,
);

console.log(`Biology claim with physics + chemistry dependencies:`);
console.log(`  Valid: ${claimWithDep.valid}`);
console.log(`  Base confidence: 0.95`);
console.log(`  Stratum ceiling: ${claimWithDep.stratumCeiling}`);
console.log(
  `  Cross-stratum discount: ${claimWithDep.crossStratumDiscount.toFixed(3)}`,
);
console.log(
  `  Adjusted confidence: ${claimWithDep.adjustedConfidence.toFixed(3)}`,
);

// ============================================================================
// Test 6: Utility Functions
// ============================================================================

console.log("\n--- Test 6: Utility Functions ---");

// Check if reference is valid
const validRef = isValidClaimReference(
  "chemistry.organic",
  "physics.particles.quarks",
  "supports",
  tree,
);
console.log(
  `isValidClaimReference(chemistry → physics, supports): ${validRef}`,
);

const invalidRef = isValidClaimReference(
  "chemistry.organic",
  "biology.cellular",
  "supports",
  tree,
);
console.log(
  `isValidClaimReference(chemistry → biology, supports): ${invalidRef}`,
);

// Get adjusted confidence
const adjConf = getAdjustedClaimConfidence("social.economics", 0.95, [], tree);
console.log(`getAdjustedClaimConfidence(social, 0.95): ${adjConf.toFixed(3)}`);

// ============================================================================
// Test 7: All Strata Ceilings
// ============================================================================

console.log("\n--- Test 7: Stratum Confidence Ceilings ---");

const strata = [
  "fundamental",
  "physical",
  "chemical",
  "biological",
  "cognitive",
  "social",
  "technological",
] as const;
for (const stratum of strata) {
  const ceiling = STRATUM_CONFIDENCE_CEILING[stratum];
  console.log(`  ${stratum}: ${ceiling}`);
}

// ============================================================================
// Summary
// ============================================================================

console.log("\n=== All Tests Complete ===\n");

// Final validation
// Fallback should resolve to closest existing parent (physics.mechanics.quantum)
const allTestsPassed =
  exactDomain?.id === "physics.mechanics.quantum" &&
  fallbackDomain?.id === "physics.mechanics.quantum" &&
  physicsClaim.valid &&
  socialClaim.adjustedConfidence <= 0.85 &&
  chemToPhysics.valid &&
  !chemToBiology.valid &&
  !claimWithBadRef.valid &&
  validRef === true &&
  invalidRef === false;

console.log("Test details:");
console.log(
  `  exactDomain: ${exactDomain?.id} === physics.mechanics.quantum: ${exactDomain?.id === "physics.mechanics.quantum"}`,
);
console.log(
  `  fallbackDomain: ${fallbackDomain?.id} === physics.mechanics.quantum: ${fallbackDomain?.id === "physics.mechanics.quantum"}`,
);
console.log(`  physicsClaim.valid: ${physicsClaim.valid}`);
console.log(
  `  socialClaim.adjustedConfidence <= 0.85: ${socialClaim.adjustedConfidence <= 0.85}`,
);
console.log(`  chemToPhysics.valid: ${chemToPhysics.valid}`);
console.log(`  !chemToBiology.valid: ${!chemToBiology.valid}`);
console.log(`  !claimWithBadRef.valid: ${!claimWithBadRef.valid}`);
console.log(`  validRef: ${validRef}`);
console.log(`  !invalidRef: ${!invalidRef}`);

if (allTestsPassed) {
  console.log("SUCCESS: All claim ontology tests passed!");
} else {
  console.log("FAILURE: Some tests failed.");
  process.exit(1);
}
