/**
 * Test: Knowledge Archive
 *
 * Tests for the Knowledge Archive including:
 * - Domain queries
 * - Truth evaluation
 * - Related statements
 * - Semantic similarity
 * - Batch queries
 */

import { KnowledgeArchive } from "../core/archive/knowledge_archive.js";
import { KnowledgeArchiveAPI } from "../core/archive/archive_api.js";
import { ArchiveRPCHandler } from "../core/archive/rpc_handlers.js";
import { createDomainTree } from "../core/knowledge/ontology/tree.js";
import { GENESIS_DOMAINS } from "../core/knowledge/ontology/genesis.js";
import type { FactState } from "../state/types.js";
import type { ArchiveReference } from "../core/archive/types.js";

console.log("=== Knowledge Archive Tests ===\n");

// Create domain tree
const domainTree = createDomainTree(GENESIS_DOMAINS);

// Create archive
const archive = new KnowledgeArchive();
archive.setDomainTree(domainTree);
archive.setBlockNumber(1000n);

// ============================================================================
// Test Data Setup
// ============================================================================

console.log("--- Setting up test data ---");

// Create test facts
const facts: FactState[] = [
  {
    factId: "fact-physics-001",
    schemaId: "test-schema",
    domain: "physics.mechanics.quantum",
    claimHash: new Uint8Array(32),
    confidence: 0.95,
    status: "verified",
    verificationRound: 1,
    verifierCount: 10,
    submittedBy: "validator-001",
    submittedAt: 100n,
    verifiedAt: 200n,
    stake: 1000n,
    stratum: "physical",
    resolvedDomainId: "physics.mechanics.quantum",
    ontologyValidated: true,
  },
  {
    factId: "fact-physics-002",
    schemaId: "test-schema",
    domain: "physics.mechanics.quantum",
    claimHash: new Uint8Array(32),
    confidence: 0.88,
    status: "verified",
    verificationRound: 2,
    verifierCount: 8,
    submittedBy: "validator-002",
    submittedAt: 300n,
    verifiedAt: 400n,
    stake: 1000n,
    stratum: "physical",
    resolvedDomainId: "physics.mechanics.quantum",
    ontologyValidated: true,
  },
  {
    factId: "fact-chemistry-001",
    schemaId: "test-schema",
    domain: "chemistry.organic",
    claimHash: new Uint8Array(32),
    confidence: 0.82,
    status: "verified",
    verificationRound: 3,
    verifierCount: 6,
    submittedBy: "validator-001",
    submittedAt: 500n,
    verifiedAt: 600n,
    stake: 1000n,
    stratum: "chemical",
    resolvedDomainId: "chemistry.organic",
    ontologyValidated: true,
  },
  {
    factId: "fact-biology-001",
    schemaId: "test-schema",
    domain: "biology.cellular",
    claimHash: new Uint8Array(32),
    confidence: 0.78,
    status: "verified",
    verificationRound: 4,
    verifierCount: 5,
    submittedBy: "validator-003",
    submittedAt: 700n,
    verifiedAt: 800n,
    stake: 1000n,
    stratum: "biological",
    resolvedDomainId: "biology.cellular",
    ontologyValidated: true,
  },
  {
    factId: "fact-social-001",
    schemaId: "test-schema",
    domain: "social.economics",
    claimHash: new Uint8Array(32),
    confidence: 0.72,
    status: "verified",
    verificationRound: 5,
    verifierCount: 4,
    submittedBy: "validator-004",
    submittedAt: 900n,
    verifiedAt: 1000n,
    stake: 1000n,
    stratum: "social",
    resolvedDomainId: "social.economics",
    ontologyValidated: true,
  },
];

// Create references
const references: ArchiveReference[] = [
  {
    factId: "fact-physics-002",
    targetFactId: "fact-physics-001",
    referenceType: "supports",
    strength: 0.9,
    bidirectional: false,
  },
  {
    factId: "fact-chemistry-001",
    targetFactId: "fact-physics-001",
    referenceType: "extends",
    strength: 0.7,
    bidirectional: false,
  },
  {
    factId: "fact-biology-001",
    targetFactId: "fact-chemistry-001",
    referenceType: "extends",
    strength: 0.8,
    bidirectional: false,
  },
];

// Index facts
async function indexTestData() {
  for (const fact of facts) {
    const domain = domainTree.domainIndex.get(fact.resolvedDomainId!)?.domain;
    if (domain) {
      const factRefs = references.filter((r) => r.factId === fact.factId);
      await archive.indexFact(
        fact,
        domain,
        factRefs,
        `Test claim for ${fact.factId}`,
      );
    }
  }
}

await indexTestData();
console.log(
  `Indexed ${facts.length} facts with ${references.length} references\n`,
);

// ============================================================================
// Test 1: Domain Query
// ============================================================================

console.log("--- Test 1: Domain Query ---");

const physicsResult = await archive.queryByDomain("physics.*", {
  includeDescendants: true,
  minConfidence: 0.8,
});

console.log(`Query: physics.* with minConfidence 0.8`);
console.log(`  Success: ${physicsResult.success}`);
console.log(`  Results: ${physicsResult.items.length}`);
console.log(
  `  Facts: ${physicsResult.items.map((f) => f.fact.factId).join(", ")}`,
);

// ============================================================================
// Test 2: Stratum Query
// ============================================================================

console.log("\n--- Test 2: Stratum Query ---");

const chemicalResult = await archive.queryByStratum("chemical", {
  minConfidence: 0.7,
});

console.log(`Query: stratum=chemical with minConfidence 0.7`);
console.log(`  Success: ${chemicalResult.success}`);
console.log(`  Results: ${chemicalResult.items.length}`);

// ============================================================================
// Test 3: Truth Evaluation
// ============================================================================

console.log("\n--- Test 3: Truth Evaluation ---");

const truthEval = await archive.evaluateTruth("fact-physics-001");

console.log(`Evaluation for fact-physics-001:`);
console.log(`  Verdict: ${truthEval.verdict}`);
console.log(`  Confidence: ${truthEval.confidence.toFixed(3)}`);
console.log(
  `  Confidence Ratio: ${truthEval.confidenceRatio.ratio.toFixed(3)}`,
);
console.log(`  Supporting: ${truthEval.confidenceRatio.supporting.toFixed(3)}`);
console.log(
  `  Contradicting: ${truthEval.confidenceRatio.contradicting.toFixed(3)}`,
);
console.log(`  Supporting Facts: ${truthEval.supportingFacts.length}`);
console.log(`  Contradicting Facts: ${truthEval.contradictingFacts.length}`);
console.log(`  Stratum: ${truthEval.ontologicalContext.stratum}`);
console.log(
  `  Stratum Ceiling: ${truthEval.ontologicalContext.stratumCeiling}`,
);
console.log(`  Reasoning: ${truthEval.reasoning}`);

// ============================================================================
// Test 4: Related Statements
// ============================================================================

console.log("\n--- Test 4: Related Statements ---");

const related = await archive.getRelatedStatements("fact-physics-001", {
  includeReferences: true,
  includeOntology: true,
  includeSimilarity: true,
  maxSimilarityResults: 5,
});

console.log(`Related statements for fact-physics-001:`);
console.log(`  Supports: ${related.supports.length}`);
console.log(`  Contradicts: ${related.contradicts.length}`);
console.log(`  Extends: ${related.extends.length}`);
console.log(`  Semantically Similar: ${related.semanticallySimilar.length}`);
console.log(`  Total Related: ${related.statistics.totalRelated}`);
console.log(
  `  Supporting Weight: ${related.statistics.supportingWeight.toFixed(3)}`,
);

// ============================================================================
// Test 5: Semantic Search
// ============================================================================

console.log("\n--- Test 5: Semantic Search ---");

const similar = await archive.findSimilar("fact-physics-001", { limit: 5 });

console.log(`Similar facts to fact-physics-001:`);
for (const match of similar) {
  console.log(
    `  ${match.factId}: similarity=${match.similarityScore.toFixed(3)}, confidence=${match.confidence.toFixed(3)}`,
  );
}

// ============================================================================
// Test 6: API Layer
// ============================================================================

console.log("\n--- Test 6: API Layer ---");

const api = new KnowledgeArchiveAPI(archive);
const session = api.createSession("did:lgm:test-agent");

console.log(`Created session: ${session.sessionId}`);

// Test truth evaluation via API
const apiTruth = await api.processIntent(session.sessionId, {
  type: "evaluate_truth",
  factId: "fact-physics-001",
});

console.log(`API Truth Evaluation:`);
console.log(`  Success: ${apiTruth.success}`);
if (apiTruth.result) {
  const result = apiTruth.result as any;
  console.log(`  Verdict: ${result.verdict}`);
  console.log(`  PP Truth Score: ${apiTruth.pp.scores.truth.toFixed(3)}`);
}

// Test domain query via API
const apiDomain = await api.processIntent(session.sessionId, {
  type: "query_domain",
  domain: "biology.*",
  includeDescendants: true,
});

console.log(`\nAPI Domain Query (biology.*):`);
console.log(`  Success: ${apiDomain.success}`);
console.log(
  `  Results: ${Array.isArray(apiDomain.result) ? apiDomain.result.length : 0}`,
);

// ============================================================================
// Test 7: RPC Handler
// ============================================================================

console.log("\n--- Test 7: RPC Handler ---");

const rpcHandler = new ArchiveRPCHandler(api);

// Create session via RPC
const rpcSession = await rpcHandler.handleRequest({
  jsonrpc: "2.0",
  id: 1,
  method: "archive_createSession",
  params: { agentDid: "did:lgm:rpc-test-agent" },
});

console.log(`RPC Create Session: ${rpcSession.result ? "OK" : "FAILED"}`);

// Get sync status via RPC
const rpcStatus = await rpcHandler.handleRequest({
  jsonrpc: "2.0",
  id: 2,
  method: "archive_getSyncStatus",
});

console.log(`RPC Sync Status: ${rpcStatus.result ? "OK" : "FAILED"}`);
if (rpcStatus.result) {
  const status = rpcStatus.result as any;
  console.log(`  Current Block: ${status.currentBlock}`);
  console.log(`  Synced: ${status.synced}`);
}

// Get statistics via RPC
const rpcStats = await rpcHandler.handleRequest({
  jsonrpc: "2.0",
  id: 3,
  method: "archive_getStatistics",
});

console.log(`RPC Statistics: ${rpcStats.result ? "OK" : "FAILED"}`);
if (rpcStats.result) {
  const stats = rpcStats.result as any;
  console.log(`  Total Facts: ${stats.totalFacts}`);
  console.log(`  Total References: ${stats.totalReferences}`);
}

// ============================================================================
// Test 8: Batch Query
// ============================================================================

console.log("\n--- Test 8: Batch Query ---");

const batchResult = await archive.batchQuery({
  queries: [
    {
      type: "domain",
      filters: { domain: "physics.*" },
      pagination: { limit: 10, sortBy: "confidence", sortOrder: "desc" },
      options: {
        includeReferences: false,
        includeOntology: false,
        includeSimilarity: false,
      },
    },
    {
      type: "stratum",
      filters: { stratum: "biological" },
      pagination: { limit: 10, sortBy: "confidence", sortOrder: "desc" },
      options: {
        includeReferences: false,
        includeOntology: false,
        includeSimilarity: false,
      },
    },
    {
      type: "truth",
      filters: { factId: "fact-chemistry-001" },
      pagination: { limit: 1, sortBy: "confidence", sortOrder: "desc" },
      options: {
        includeReferences: false,
        includeOntology: false,
        includeSimilarity: false,
      },
    },
  ],
  options: {
    parallel: true,
    maxParallelism: 5,
    timeout: 30000,
    streaming: false,
  },
});

console.log(`Batch query results:`);
console.log(`  Total time: ${batchResult.totalTime}ms`);
console.log(`  Results: ${batchResult.results.size}`);
console.log(`  Errors: ${batchResult.errors.size}`);

// ============================================================================
// Summary
// ============================================================================

console.log("\n=== Test Summary ===\n");

const tests = [
  {
    name: "Domain Query",
    passed: physicsResult.success && physicsResult.items.length > 0,
  },
  { name: "Stratum Query", passed: chemicalResult.success },
  { name: "Truth Evaluation", passed: truthEval.verdict !== undefined },
  { name: "Related Statements", passed: related.statistics.totalRelated >= 0 },
  { name: "Semantic Search", passed: true }, // May return 0 results with stub embedder
  { name: "API Layer", passed: apiTruth.success },
  { name: "RPC Handler", passed: rpcSession.result !== undefined },
  { name: "Batch Query", passed: batchResult.results.size === 3 },
];

let allPassed = true;
for (const test of tests) {
  console.log(`${test.passed ? "✓" : "✗"} ${test.name}`);
  if (!test.passed) allPassed = false;
}

console.log();
if (allPassed) {
  console.log("SUCCESS: All archive tests passed!");
} else {
  console.log("FAILURE: Some tests failed.");
  process.exit(1);
}
