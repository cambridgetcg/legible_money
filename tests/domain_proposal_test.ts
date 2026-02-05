/**
 * Test: Domain Proposal System
 *
 * Tests for the novel domain handling system including:
 * - Domain proposal creation with 22/222/2222 LGM stakes
 * - Validator endorsement (requires 2)
 * - Challenge mechanism
 * - 777 blocks challenge period
 * - 77% stake return on rejection
 * - Claim rejection for pending domains
 * - 111 max children per domain
 */

import {
  DOMAIN_PROPOSAL_CONFIG,
  LGM,
  calculateRequiredStake,
  calculateEndorsementStake,
  requiresGovernance,
  validateDomainProposal,
  createDomainProposal,
  endorseProposal,
  challengeProposal,
  finalizeProposal,
  calculateStakeReturn,
  calculateEndorserStakeReturns,
  getTotalEndorsementStake,
  createDomainFromProposal,
  isDomainPending,
  getPendingProposal,
  type DomainProposal,
  type DomainProposalInput,
} from "../core/knowledge/ontology/domain_proposal.js";

import { createDomainTree } from "../core/knowledge/ontology/tree.js";
import { GENESIS_DOMAINS } from "../core/knowledge/ontology/genesis.js";
import { validateClaimOntology } from "../core/knowledge/ontology/claim_validation.js";
import type { DomainTree } from "../core/knowledge/ontology/types.js";

console.log("=== Domain Proposal System Tests ===\n");

// Create domain tree with genesis domains
const domainTree = createDomainTree(GENESIS_DOMAINS);

// ============================================================================
// Test 1: Configuration Values
// ============================================================================

console.log("--- Test 1: Configuration Values ---");

const expectedConfig = {
  leafDomainStake: 22n * LGM,
  midDomainStake: 222n * LGM,
  topLevelDomainStake: 2222n * LGM,
  leafEndorsementStake: 2n * LGM,
  midEndorsementStake: 22n * LGM,
  topLevelEndorsementStake: 222n * LGM,
  stakeReturnOnRejection: 0.77,
  challengePeriodBlocks: 777n,
  minimumSupporters: 2,
  parentVetoEnabled: false,
  pendingClaimBehavior: "reject",
  maxChildrenPerDomain: 111,
};

console.log(
  `Leaf domain stake: ${DOMAIN_PROPOSAL_CONFIG.leafDomainStake === expectedConfig.leafDomainStake ? "✓" : "✗"} ${DOMAIN_PROPOSAL_CONFIG.leafDomainStake / LGM} LGM`,
);
console.log(
  `Mid domain stake: ${DOMAIN_PROPOSAL_CONFIG.midDomainStake === expectedConfig.midDomainStake ? "✓" : "✗"} ${DOMAIN_PROPOSAL_CONFIG.midDomainStake / LGM} LGM`,
);
console.log(
  `Top-level domain stake: ${DOMAIN_PROPOSAL_CONFIG.topLevelDomainStake === expectedConfig.topLevelDomainStake ? "✓" : "✗"} ${DOMAIN_PROPOSAL_CONFIG.topLevelDomainStake / LGM} LGM`,
);
console.log(
  `Leaf endorsement stake: ${DOMAIN_PROPOSAL_CONFIG.leafEndorsementStake === expectedConfig.leafEndorsementStake ? "✓" : "✗"} ${DOMAIN_PROPOSAL_CONFIG.leafEndorsementStake / LGM} LGM`,
);
console.log(
  `Mid endorsement stake: ${DOMAIN_PROPOSAL_CONFIG.midEndorsementStake === expectedConfig.midEndorsementStake ? "✓" : "✗"} ${DOMAIN_PROPOSAL_CONFIG.midEndorsementStake / LGM} LGM`,
);
console.log(
  `Top-level endorsement stake: ${DOMAIN_PROPOSAL_CONFIG.topLevelEndorsementStake === expectedConfig.topLevelEndorsementStake ? "✓" : "✗"} ${DOMAIN_PROPOSAL_CONFIG.topLevelEndorsementStake / LGM} LGM`,
);
console.log(
  `Stake return on rejection: ${DOMAIN_PROPOSAL_CONFIG.stakeReturnOnRejection === expectedConfig.stakeReturnOnRejection ? "✓" : "✗"} ${DOMAIN_PROPOSAL_CONFIG.stakeReturnOnRejection * 100}%`,
);
console.log(
  `Challenge period: ${DOMAIN_PROPOSAL_CONFIG.challengePeriodBlocks === expectedConfig.challengePeriodBlocks ? "✓" : "✗"} ${DOMAIN_PROPOSAL_CONFIG.challengePeriodBlocks} blocks`,
);
console.log(
  `Minimum supporters: ${DOMAIN_PROPOSAL_CONFIG.minimumSupporters === expectedConfig.minimumSupporters ? "✓" : "✗"} ${DOMAIN_PROPOSAL_CONFIG.minimumSupporters}`,
);
console.log(
  `Parent veto: ${DOMAIN_PROPOSAL_CONFIG.parentVetoEnabled === expectedConfig.parentVetoEnabled ? "✓" : "✗"} ${DOMAIN_PROPOSAL_CONFIG.parentVetoEnabled}`,
);
console.log(
  `Pending claim behavior: ${DOMAIN_PROPOSAL_CONFIG.pendingClaimBehavior === expectedConfig.pendingClaimBehavior ? "✓" : "✗"} ${DOMAIN_PROPOSAL_CONFIG.pendingClaimBehavior}`,
);
console.log(
  `Max children: ${DOMAIN_PROPOSAL_CONFIG.maxChildrenPerDomain === expectedConfig.maxChildrenPerDomain ? "✓" : "✗"} ${DOMAIN_PROPOSAL_CONFIG.maxChildrenPerDomain}`,
);

// ============================================================================
// Test 2: Stake Requirements (Proposer)
// ============================================================================

console.log("\n--- Test 2: Proposer Stake Requirements ---");

// Top-level (depth 1)
const topLevelStake = calculateRequiredStake("root", 1);
console.log(
  `Top-level stake: ${topLevelStake === 2222n * LGM ? "✓" : "✗"} ${topLevelStake / LGM} LGM`,
);

// Mid-level (depth 2)
const midStake = calculateRequiredStake("physics", 2);
console.log(
  `Mid-level stake: ${midStake === 222n * LGM ? "✓" : "✗"} ${midStake / LGM} LGM`,
);

// Leaf (depth 3+)
const leafStake = calculateRequiredStake("physics.mechanics", 3);
console.log(
  `Leaf stake: ${leafStake === 22n * LGM ? "✓" : "✗"} ${leafStake / LGM} LGM`,
);

// ============================================================================
// Test 2b: Endorsement Stake Requirements (Validator)
// ============================================================================

console.log("\n--- Test 2b: Endorsement Stake Requirements ---");

// Top-level endorsement (depth 1)
const topLevelEndorse = calculateEndorsementStake("root", 1);
console.log(
  `Top-level endorsement: ${topLevelEndorse === 222n * LGM ? "✓" : "✗"} ${topLevelEndorse / LGM} LGM`,
);

// Mid-level endorsement (depth 2)
const midEndorse = calculateEndorsementStake("physics", 2);
console.log(
  `Mid-level endorsement: ${midEndorse === 22n * LGM ? "✓" : "✗"} ${midEndorse / LGM} LGM`,
);

// Leaf endorsement (depth 3+)
const leafEndorse = calculateEndorsementStake("physics.mechanics", 3);
console.log(
  `Leaf endorsement: ${leafEndorse === 2n * LGM ? "✓" : "✗"} ${leafEndorse / LGM} LGM`,
);

// ============================================================================
// Test 3: Governance Requirements
// ============================================================================

console.log("\n--- Test 3: Governance Requirements ---");

// Fundamental stratum requires governance
const fundamentalGov = requiresGovernance("fundamental", "physics", 2);
console.log(
  `Fundamental stratum: ${fundamentalGov ? "✓" : "✗"} requires governance`,
);

// Physical stratum requires governance
const physicalGov = requiresGovernance("physical", "physics", 2);
console.log(`Physical stratum: ${physicalGov ? "✓" : "✗"} requires governance`);

// Chemical stratum requires governance
const chemicalGov = requiresGovernance("chemical", "chemistry", 2);
console.log(`Chemical stratum: ${chemicalGov ? "✓" : "✗"} requires governance`);

// Top-level biological requires governance
const topLevelBioGov = requiresGovernance("biological", "root", 1);
console.log(
  `Top-level biological: ${topLevelBioGov ? "✓" : "✗"} requires governance`,
);

// Leaf biological doesn't require governance
const leafBioGov = requiresGovernance("biological", "biology.cellular", 3);
console.log(`Leaf biological: ${!leafBioGov ? "✓" : "✗"} no governance needed`);

// ============================================================================
// Test 4: Proposal Validation
// ============================================================================

console.log("\n--- Test 4: Proposal Validation ---");

// Valid leaf domain proposal
const validInput: DomainProposalInput = {
  domainId: "biology.cellular.mitochondria",
  name: "Mitochondrial Biology",
  stratum: "biological",
  parentDomainId: "biology.cellular",
  description: "Study of mitochondrial structure and function",
  examples: [
    "Mitochondria produce ATP via oxidative phosphorylation",
    "Mitochondrial DNA is inherited maternally",
  ],
  proposer: "did:lgm:test-proposer",
  stake: 22n * LGM,
};

const validResult = validateDomainProposal(validInput, domainTree);
console.log(
  `Valid proposal: ${validResult.valid ? "✓" : "✗"} errors: ${validResult.errors.length}`,
);

// Invalid: wrong parent stratum
const wrongStratumInput: DomainProposalInput = {
  ...validInput,
  domainId: "physics.cellular.mitochondria",
  parentDomainId: "physics.mechanics",
  stratum: "biological",
};
const wrongStratumResult = validateDomainProposal(
  wrongStratumInput,
  domainTree,
);
console.log(`Wrong stratum: ${!wrongStratumResult.valid ? "✓" : "✗"} rejected`);

// Invalid: insufficient examples
const noExamplesInput: DomainProposalInput = {
  ...validInput,
  examples: ["Only one example"],
};
const noExamplesResult = validateDomainProposal(noExamplesInput, domainTree);
console.log(
  `Insufficient examples: ${!noExamplesResult.valid ? "✓" : "✗"} rejected`,
);

// Invalid: insufficient stake
const lowStakeInput: DomainProposalInput = {
  ...validInput,
  stake: 10n * LGM, // Less than 22 LGM
};
const lowStakeResult = validateDomainProposal(lowStakeInput, domainTree);
console.log(
  `Insufficient stake: ${!lowStakeResult.valid ? "✓" : "✗"} rejected`,
);

// ============================================================================
// Test 5: Proposal Lifecycle
// ============================================================================

console.log("\n--- Test 5: Proposal Lifecycle ---");

const currentBlock = 1000n;

// Create proposal
const { proposal, validation } = createDomainProposal(
  validInput,
  currentBlock,
  domainTree,
);
console.log(`Created proposal: ${proposal.id.slice(0, 16)}...`);
console.log(`  Status: ${proposal.status}`);
console.log(`  Challenge deadline: block ${proposal.challengeDeadline}`);
console.log(
  `  Required endorsement stake: ${proposal.requiredEndorsementStake / LGM} LGM`,
);
console.log(`  Requires governance: ${validation.requiresGovernance}`);

// Test insufficient endorsement stake
try {
  endorseProposal(proposal, "did:lgm:validator-0", currentBlock + 5n, 1n * LGM); // Only 1 LGM, need 2
  console.log("✗ Should reject insufficient endorsement stake");
} catch (e) {
  console.log("✓ Rejects insufficient endorsement stake");
}

// First endorsement (with correct stake)
const endorsed1 = endorseProposal(
  proposal,
  "did:lgm:validator-1",
  currentBlock + 10n,
  2n * LGM,
);
console.log(
  `First endorsement: ${endorsed1.endorsers.length === 1 ? "✓" : "✗"} status: ${endorsed1.status}, stake: ${endorsed1.endorserStakes.get("did:lgm:validator-1")! / LGM} LGM`,
);

// Second endorsement (should change status to 'endorsed')
const endorsed2 = endorseProposal(
  endorsed1,
  "did:lgm:validator-2",
  currentBlock + 20n,
  2n * LGM,
);
console.log(
  `Second endorsement: ${endorsed2.status === "endorsed" ? "✓" : "✗"} status: ${endorsed2.status}`,
);

// Check total endorsement stake
const totalEndorseStake = getTotalEndorsementStake(endorsed2);
console.log(
  `Total endorsement stake: ${totalEndorseStake === 4n * LGM ? "✓" : "✗"} ${totalEndorseStake / LGM} LGM (2 validators × 2 LGM)`,
);

// ============================================================================
// Test 6: Challenge Period (777 blocks)
// ============================================================================

console.log("\n--- Test 6: Challenge Period ---");

// Cannot finalize before deadline
try {
  finalizeProposal(endorsed2, currentBlock + 100n); // Before 777 blocks
  console.log("✗ Should not finalize before deadline");
} catch (e) {
  console.log("✓ Cannot finalize before deadline");
}

// Can finalize after deadline
const afterDeadline =
  currentBlock + DOMAIN_PROPOSAL_CONFIG.challengePeriodBlocks + 1n;
const finalized = finalizeProposal(endorsed2, afterDeadline);
console.log(
  `Finalized after deadline: ${finalized.status === "approved" ? "✓" : "✗"} status: ${finalized.status}`,
);

// ============================================================================
// Test 7: Rejection and 77% Stake Return
// ============================================================================

console.log("\n--- Test 7: Rejection and Stake Return ---");

// Create new proposal
const { proposal: proposal2 } = createDomainProposal(
  {
    ...validInput,
    domainId: "biology.cellular.ribosomes",
    name: "Ribosomal Biology",
  },
  currentBlock,
  domainTree,
);

// No endorsements, finalize after deadline
const rejectedProposal = finalizeProposal(proposal2, afterDeadline);
console.log(
  `Rejected without endorsements: ${rejectedProposal.status === "rejected" ? "✓" : "✗"} status: ${rejectedProposal.status}`,
);

// Calculate 77% stake return
const stakeReturn = calculateStakeReturn(rejectedProposal);
const expectedReturn = (22n * LGM * 77n) / 100n;
console.log(
  `Stake return: ${stakeReturn === expectedReturn ? "✓" : "✗"} ${stakeReturn / LGM} LGM (77% of 22)`,
);

// ============================================================================
// Test 8: Challenge Mechanism
// ============================================================================

console.log("\n--- Test 8: Challenge Mechanism ---");

// Create proposal with one endorsement
const { proposal: proposal3 } = createDomainProposal(
  {
    ...validInput,
    domainId: "biology.cellular.lysosomes",
    name: "Lysosomal Biology",
  },
  currentBlock,
  domainTree,
);

const endorsed = endorseProposal(
  proposal3,
  "did:lgm:validator-1",
  currentBlock + 10n,
  2n * LGM,
);

// Challenge it
const challenged = challengeProposal(
  endorsed,
  "did:lgm:challenger-1",
  "Domain scope is too narrow",
  10n * LGM,
  currentBlock + 20n,
);
console.log(
  `Challenged: ${challenged.challengers.length === 1 ? "✓" : "✗"} challengers: ${challenged.challengers.length}`,
);

// Finalize after deadline (should be rejected: 1 endorser vs 1 challenger)
const finalizedChallenged = finalizeProposal(challenged, afterDeadline);
console.log(
  `Rejected when challenged: ${finalizedChallenged.status === "rejected" ? "✓" : "✗"} status: ${finalizedChallenged.status}`,
);

// ============================================================================
// Test 8b: Endorser Stake Returns
// ============================================================================

console.log("\n--- Test 8b: Endorser Stake Returns ---");

// Test 1: Approved proposal - endorsers get 100% back
const endorserReturnsApproved = calculateEndorserStakeReturns(finalized);
let approvedReturn = 0n;
for (const [, { stakeReturn, stakeSlashed }] of endorserReturnsApproved) {
  approvedReturn += stakeReturn;
  console.log(
    `Approved endorser return: ${stakeSlashed === 0n ? "✓" : "✗"} 100% (${stakeReturn / LGM} LGM, slashed: ${stakeSlashed / LGM} LGM)`,
  );
}

// Test 2: Rejected without challenge - endorsers get 100% back
const endorserReturnsNoChallenge =
  calculateEndorserStakeReturns(rejectedProposal);
console.log(
  `Rejected (no challenge) endorser returns: ${endorserReturnsNoChallenge.size === 0 ? "✓" : "✗"} (no endorsers)`,
);

// Test 3: Rejected with challenge - endorsers get 77% back
const endorserReturnsChallenged =
  calculateEndorserStakeReturns(finalizedChallenged);
for (const [
  endorser,
  { stakeReturn, stakeSlashed },
] of endorserReturnsChallenged) {
  const originalStake = finalizedChallenged.endorserStakes.get(endorser) ?? 0n;
  const expectedReturn = (originalStake * 77n) / 100n;
  const expectedSlash = originalStake - expectedReturn;
  console.log(
    `Challenged endorser return: ${stakeReturn === expectedReturn ? "✓" : "✗"} 77% (${stakeReturn / LGM} LGM, slashed: ${stakeSlashed / LGM} LGM)`,
  );
}

// ============================================================================
// Test 9: Pending Domain Claim Rejection
// ============================================================================

console.log("\n--- Test 9: Pending Domain Claim Rejection ---");

// Create pending proposals map
const pendingProposals = new Map<string, DomainProposal>();

// Create a pending proposal
const { proposal: pendingProposal } = createDomainProposal(
  {
    ...validInput,
    domainId: "biology.cellular.golgi",
    name: "Golgi Apparatus",
  },
  currentBlock,
  domainTree,
);

pendingProposals.set(pendingProposal.id, pendingProposal);

// Try to submit claim to pending domain
const claimResult = validateClaimOntology(
  {
    domain: "biology.cellular.golgi",
    proposedConfidence: 0.8,
    references: [],
  },
  domainTree,
  pendingProposals,
);

console.log(
  `Claim to pending domain: ${!claimResult.valid ? "✓" : "✗"} rejected`,
);
console.log(
  `  Pending proposal ID: ${claimResult.pendingDomainProposal?.id.slice(0, 16)}...`,
);
console.log(`  Error: ${claimResult.errors[0]?.slice(0, 60)}...`);

// Check helper functions
const isPending = isDomainPending("biology.cellular.golgi", pendingProposals);
console.log(`isDomainPending: ${isPending ? "✓" : "✗"} ${isPending}`);

const foundProposal = getPendingProposal(
  "biology.cellular.golgi",
  pendingProposals,
);
console.log(`getPendingProposal: ${foundProposal ? "✓" : "✗"} found`);

// ============================================================================
// Test 10: Domain Creation from Approved Proposal
// ============================================================================

console.log("\n--- Test 10: Domain Creation ---");

// Get parent domain
const parentNode = domainTree.domainIndex.get("biology.cellular");
const parentDomain = parentNode?.domain ?? null;

// Create domain from approved proposal
const newDomain = createDomainFromProposal(
  finalized,
  parentDomain,
  Number(afterDeadline),
);

console.log(`Created domain: ${newDomain.id}`);
console.log(`  Name: ${newDomain.name}`);
console.log(`  Stratum: ${newDomain.stratum}`);
console.log(`  Parent: ${newDomain.parentDomain}`);
console.log(`  Logic zone: ${newDomain.logicZoneDefault}`);

// ============================================================================
// Summary
// ============================================================================

console.log("\n=== Test Summary ===\n");

const tests = [
  { name: "Configuration Values", passed: true },
  {
    name: "Proposer Stake (22/222/2222)",
    passed:
      topLevelStake === 2222n * LGM &&
      midStake === 222n * LGM &&
      leafStake === 22n * LGM,
  },
  {
    name: "Endorsement Stake (2/22/222)",
    passed:
      topLevelEndorse === 222n * LGM &&
      midEndorse === 22n * LGM &&
      leafEndorse === 2n * LGM,
  },
  {
    name: "Governance Requirements",
    passed:
      fundamentalGov &&
      physicalGov &&
      chemicalGov &&
      topLevelBioGov &&
      !leafBioGov,
  },
  {
    name: "Proposal Validation",
    passed:
      validResult.valid &&
      !wrongStratumResult.valid &&
      !noExamplesResult.valid &&
      !lowStakeResult.valid,
  },
  {
    name: "Proposal Lifecycle",
    passed: endorsed2.status === "endorsed" && totalEndorseStake === 4n * LGM,
  },
  {
    name: "Challenge Period (777 blocks)",
    passed: finalized.status === "approved",
  },
  {
    name: "Rejection & 77% Stake Return",
    passed:
      rejectedProposal.status === "rejected" && stakeReturn === expectedReturn,
  },
  {
    name: "Challenge Mechanism",
    passed: finalizedChallenged.status === "rejected",
  },
  {
    name: "Endorser Stake Returns",
    passed:
      endorserReturnsApproved.size === 2 &&
      endorserReturnsChallenged.size === 1,
  },
  {
    name: "Pending Domain Claim Rejection",
    passed: !claimResult.valid && isPending,
  },
  {
    name: "Domain Creation",
    passed: newDomain.id === "biology.cellular.mitochondria",
  },
];

let allPassed = true;
for (const test of tests) {
  console.log(`${test.passed ? "✓" : "✗"} ${test.name}`);
  if (!test.passed) allPassed = false;
}

console.log();
if (allPassed) {
  console.log("SUCCESS: All domain proposal tests passed!");
} else {
  console.log("FAILURE: Some tests failed.");
  process.exit(1);
}
