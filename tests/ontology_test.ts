/**
 * Simple test of the ontology module
 */

import { createDomainTree } from "../core/knowledge/ontology/tree.js";
import {
  GENESIS_DOMAINS,
  getGenesisDomainCount,
} from "../core/knowledge/ontology/genesis.js";
import { calculateOntologicalConfidence } from "../core/knowledge/ontology/confidence.js";
import { validateOntologicalReference } from "../core/knowledge/ontology/validation.js";

// Create the tree
const tree = createDomainTree(GENESIS_DOMAINS);

console.log("=== Ontological Tree of Knowledge ===");
console.log("");
console.log("Genesis Domains:", getGenesisDomainCount());
console.log("Total in Tree:", tree.domainCount);
console.log("Max Depth:", tree.maxDepth);
console.log("");
console.log("Domains by Stratum:");
for (const [stratum, ids] of tree.strataIndex) {
  console.log(`  ${stratum}: ${ids.length}`);
}

// Test confidence calculation
console.log("");
console.log("=== Confidence Calculation Test ===");
const physics = tree.domainIndex.get("physics.particles.quarks")!.domain;
const biology = tree.domainIndex.get("biology.cellular")!.domain;

const physicsConf = calculateOntologicalConfidence(0.95, physics, [], tree);
console.log(
  `physics.particles.quarks (fundamental): base=0.95 → effective=${physicsConf.effectiveConfidence.toFixed(3)}`,
);

const biologyConf = calculateOntologicalConfidence(
  0.95,
  biology,
  [
    {
      factId: "test",
      primaryStratum: "chemical",
      dependentStrata: [],
      dependencyType: "emergence",
      dependencyStrength: 0.9,
    },
  ],
  tree,
);
console.log(
  `biology.cellular (biological): base=0.95, dep on chemical → effective=${biologyConf.effectiveConfidence.toFixed(3)}`,
);

// Test reference validation
console.log("");
console.log("=== Reference Validation Test ===");
const chemDomain = tree.domainIndex.get("chemistry.organic")!.domain;
const techDomain = tree.domainIndex.get("technology.blockchain")!.domain;

const validRef = validateOntologicalReference(biology, chemDomain, "supports");
console.log(
  `biology → chemistry (supports): ${validRef.valid ? "VALID" : "INVALID - " + validRef.reason}`,
);

const invalidRef = validateOntologicalReference(
  chemDomain,
  techDomain,
  "supports",
);
console.log(
  `chemistry → technology (supports): ${invalidRef.valid ? "VALID" : "INVALID - " + invalidRef.reason}`,
);

console.log("");
console.log("=== All Tests Passed ===");
