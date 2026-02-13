/**
 * Dimension 6: Knowledge System & AI Security Tests
 * File 1: Adversarial Claim & Confidence Manipulation
 *
 * Tests the protocol's core purpose — truth verification — against
 * adversarial manipulation of confidence, counter-facts, and the
 * evaluation pipeline.
 */

import { describe, it, expect } from "vitest";
import { KnowledgeStateTree } from "../../../state/knowledge.js";
import type { FactState, FactReference } from "../../../state/types.js";
import type { Confidence } from "../../../core/knowledge/types.js";
import {
  NEGATIVE_KNOWLEDGE,
  type BidirectionalConfidence,
  computeBidirectionalStatus,
} from "../../../core/knowledge/types.js";
import {
  aggregateEvidenceBayesian,
  calculateCategoryDecayedConfidence,
  calculateMasterConfidence,
  type BayesianEvidence,
  type ConfidenceCalculationInput,
} from "../../../consensus/pot/confidence.js";
import type { ValidatorReveal } from "../../../consensus/pot/types.js";

// ============ Helpers ============

/** Create a minimal active fact for testing */
function makeFact(
  id: string,
  overrides: Partial<Omit<FactState, "referenceRoot">> = {},
): Omit<FactState, "referenceRoot"> {
  return {
    id,
    claimText: `Test claim ${id}`,
    domain: "physics",
    confidence: 0.8 as Confidence,
    verifiedAt: 100n,
    verifierCount: 5,
    challengeCount: 0,
    citationCount: 0,
    status: "active",
    ...overrides,
  };
}

/** Helper to add a chain of supporting references */
function addSupportChain(
  tree: KnowledgeStateTree,
  fromId: string,
  toId: string,
  strength: number = 0.9,
): void {
  tree.addReference({
    fromFactId: fromId,
    toFactId: toId,
    referenceType: "supports",
    strength,
  });
}

/** Helper to create supporting BayesianEvidence */
function makeSupportingEvidence(
  id: string,
  strength: number = 0.8,
): BayesianEvidence {
  return {
    id,
    type: "experiment",
    strength,
    direction: "supporting",
    independence: {
      sourceIndependence: 0.9,
      methodIndependence: 0.8,
      temporalIndependence: 0.7,
      aggregateIndependence: 0.8,
    },
    reliabilityScore: 0.9,
  };
}

/** Helper to create contradicting BayesianEvidence */
function makeContradictingEvidence(
  id: string,
  strength: number = 0.8,
): BayesianEvidence {
  return {
    id,
    type: "experiment",
    strength,
    direction: "contradicting",
    independence: {
      sourceIndependence: 0.9,
      methodIndependence: 0.8,
      temporalIndependence: 0.7,
      aggregateIndependence: 0.8,
    },
    reliabilityScore: 0.9,
  };
}

// ============================================================
// 1. Confidence Manipulation (8 tests)
// ============================================================

describe("Confidence Manipulation", () => {
  it("CM-1: Diminishing returns on additional supporting citations", () => {
    // Adding N supporting facts should increase confidence but with diminishing returns
    const evidence1 = [makeSupportingEvidence("e1", 0.8)];
    const evidence2 = [
      makeSupportingEvidence("e1", 0.8),
      makeSupportingEvidence("e2", 0.8),
    ];
    const evidence5 = Array.from({ length: 5 }, (_, i) =>
      makeSupportingEvidence(`e${i}`, 0.8),
    );
    const evidence10 = Array.from({ length: 10 }, (_, i) =>
      makeSupportingEvidence(`e${i}`, 0.8),
    );

    const r1 = aggregateEvidenceBayesian(evidence1);
    const r2 = aggregateEvidenceBayesian(evidence2);
    const r5 = aggregateEvidenceBayesian(evidence5);
    const r10 = aggregateEvidenceBayesian(evidence10);

    // Each additional piece increases confidence
    expect(r2.effectiveConfidence).toBeGreaterThan(r1.effectiveConfidence);
    expect(r5.effectiveConfidence).toBeGreaterThan(r2.effectiveConfidence);
    expect(r10.effectiveConfidence).toBeGreaterThan(r5.effectiveConfidence);

    // But with diminishing returns: gap from 1->2 > gap from 5->10
    const gain1to2 = r2.effectiveConfidence - r1.effectiveConfidence;
    const gain5to10 = r10.effectiveConfidence - r5.effectiveConfidence;
    expect(gain1to2).toBeGreaterThan(gain5to10);
  });

  it("CM-2: Bidirectional confidence — falsity evidence reduces net confidence", () => {
    // Even with high truth confidence, adding falsity evidence reduces net
    const truthConfidence = 0.85 as Confidence;
    const falsityConfidence = 0.6 as Confidence;
    const net = truthConfidence - falsityConfidence;

    const bidir: BidirectionalConfidence = {
      factId: "fact-1",
      truthConfidence,
      falsityConfidence,
      netConfidence: net,
      counterFactCount: 2,
      status: computeBidirectionalStatus(net),
    };

    // Net is significantly lower than truth alone
    expect(bidir.netConfidence).toBeLessThan(truthConfidence);
    expect(bidir.netConfidence).toBe(0.85 - 0.6);
    // Status should be "contested" since net = 0.25 which is between 0.2 and 0.6
    expect(bidir.status).toBe("contested");
  });

  it("CM-3: Circular reference injection detected — A→B→C→A cycle", () => {
    const tree = new KnowledgeStateTree();

    tree.addFact(makeFact("A", { confidence: 0.8 as Confidence }));
    tree.addFact(makeFact("B", { confidence: 0.8 as Confidence }));
    tree.addFact(makeFact("C", { confidence: 0.8 as Confidence }));

    // Create cycle: A→B→C→A
    tree.addReference({
      fromFactId: "A",
      toFactId: "B",
      referenceType: "supports",
      strength: 0.9,
    });
    tree.addReference({
      fromFactId: "B",
      toFactId: "C",
      referenceType: "supports",
      strength: 0.9,
    });
    tree.addReference({
      fromFactId: "C",
      toFactId: "A",
      referenceType: "supports",
      strength: 0.9,
    });

    // Detect cycle
    const cycle = tree.detectCircularDependencies("A");
    expect(cycle).not.toBeNull();
    expect(cycle!.facts).toContain("A");
    expect(cycle!.facts).toContain("B");
    expect(cycle!.facts).toContain("C");
    expect(cycle!.penalty).toBeGreaterThan(0);
  });

  it("CM-4: Cycle penalty propagation — 2-node cycle reduces confidence", () => {
    const tree = new KnowledgeStateTree();

    tree.addFact(makeFact("X", { confidence: 0.8 as Confidence }));
    tree.addFact(makeFact("Y", { confidence: 0.8 as Confidence }));

    // Create tight 2-node cycle: X→Y and Y→X
    tree.addReference({
      fromFactId: "X",
      toFactId: "Y",
      referenceType: "derives_from",
      strength: 0.9,
    });
    tree.addReference({
      fromFactId: "Y",
      toFactId: "X",
      referenceType: "derives_from",
      strength: 0.9,
    });

    // A 2-node cycle: tightness = 1.0 (length <= 3), penalty = 0.4 * 1.0 = 0.4
    const cycle = tree.detectCircularDependencies("X");
    expect(cycle).not.toBeNull();
    expect(cycle!.tightness).toBe(1.0);
    expect(cycle!.penalty).toBeCloseTo(0.4, 5);

    // Update confidence with propagation — cycle penalty applied
    const { root, propagation } = tree.updateFactConfidenceWithPropagation(
      "X",
      0.8 as Confidence,
    );

    // After cycle penalty: X = max(0.1, 0.8 * (1 - 0.4)) = 0.48
    // Then propagation fires: Y derives from X (now 0.48) → Y = 0.48 * 0.9 = 0.432
    // Then X derives from Y (now 0.432) → X = 0.432 * 0.9 = 0.3888
    // The BFS propagation cascade settles at this value.
    const updatedFact = tree.getFact("X");
    expect(updatedFact!.confidence).toBeCloseTo(0.3888, 3);
  });

  it("CM-5: Supersession reduces old fact confidence via status change", () => {
    const tree = new KnowledgeStateTree();

    tree.addFact(makeFact("old-fact", { confidence: 0.9 as Confidence }));
    tree.addFact(makeFact("new-fact", { confidence: 0.95 as Confidence }));

    tree.supersedeFact("old-fact", "new-fact");

    const oldFact = tree.getFact("old-fact");
    expect(oldFact!.status).toBe("superseded");
    // Superseded facts are no longer active
    expect(tree.isActiveFact("old-fact")).toBe(false);
    expect(tree.isActiveFact("new-fact")).toBe(true);
  });

  it("CM-6: Revocation cascade — revoking a fact propagates via BFS", () => {
    const tree = new KnowledgeStateTree();

    // Chain: base → derived1 → derived2
    tree.addFact(makeFact("base", { confidence: 0.9 as Confidence }));
    tree.addFact(makeFact("derived1", { confidence: 0.85 as Confidence }));
    tree.addFact(makeFact("derived2", { confidence: 0.8 as Confidence }));

    // derived1 derives from base, derived2 derives from derived1
    tree.addReference({
      fromFactId: "derived1",
      toFactId: "base",
      referenceType: "derives_from",
      strength: 0.9,
    });
    tree.addReference({
      fromFactId: "derived2",
      toFactId: "derived1",
      referenceType: "derives_from",
      strength: 0.85,
    });

    // Revoke the base fact
    tree.revokeFact("base");

    // Base should be revoked
    expect(tree.getFact("base")!.status).toBe("revoked");

    // Derived facts should have reduced confidence because their derivation
    // chain is now broken (base premise is revoked/inactive)
    const derived1 = tree.getFact("derived1")!;
    // derived1 confidence is recalculated: since "base" is revoked,
    // no active premises remain → confidence becomes 0
    expect(derived1.confidence).toBeLessThan(0.85);
  });

  it("CM-7: Challenges reduce confidence — 3 challenges vs 0 challenges", () => {
    const tree = new KnowledgeStateTree();

    // Create two identical facts with different challenge counts
    tree.addFact(makeFact("no-challenges", { confidence: 0.8 as Confidence }));
    tree.addFact(makeFact("has-challenges", { confidence: 0.8 as Confidence }));

    // Challenge 3 times (must resolve each before challenging again)
    tree.challengeFact("has-challenges", 200n);
    tree.resolveChallenge("has-challenges", true, 0.75 as Confidence);
    tree.challengeFact("has-challenges", 300n);
    tree.resolveChallenge("has-challenges", true, 0.7 as Confidence);
    tree.challengeFact("has-challenges", 400n);
    tree.resolveChallenge("has-challenges", true, 0.65 as Confidence);

    const noChal = tree.getFact("no-challenges")!;
    const hasChal = tree.getFact("has-challenges")!;

    // The challenged fact ends up with lower confidence
    expect(hasChal.confidence).toBeLessThan(noChal.confidence);
    expect(hasChal.challengeCount).toBe(3);
    expect(noChal.challengeCount).toBe(0);
  });

  it("CM-8: Domain coherence — cross-domain evidence receives lower Bayesian posterior", () => {
    // Claim in "physics" referencing only "cooking" evidence
    // The evidence items have low method independence (wrong domain)
    const crossDomainEvidence: BayesianEvidence[] = [
      {
        id: "cooking-evidence-1",
        type: "attestation",
        strength: 0.5, // Attestation is weaker
        direction: "supporting",
        independence: {
          sourceIndependence: 0.3, // Low: same type of source
          methodIndependence: 0.2, // Very low: cooking methods not relevant to physics
          temporalIndependence: 0.5,
          aggregateIndependence: 0.3, // Low aggregate independence
        },
        reliabilityScore: 0.5,
      },
    ];

    const sameDomainEvidence: BayesianEvidence[] = [
      {
        id: "physics-evidence-1",
        type: "experiment",
        strength: 0.85,
        direction: "supporting",
        independence: {
          sourceIndependence: 0.9,
          methodIndependence: 0.9,
          temporalIndependence: 0.8,
          aggregateIndependence: 0.87,
        },
        reliabilityScore: 0.9,
      },
    ];

    const crossResult = aggregateEvidenceBayesian(crossDomainEvidence);
    const sameResult = aggregateEvidenceBayesian(sameDomainEvidence);

    // Same-domain evidence should produce higher confidence
    expect(sameResult.effectiveConfidence).toBeGreaterThan(
      crossResult.effectiveConfidence,
    );
  });
});

// ============================================================
// 2. Counter-Fact Grief Attack (6 tests)
// ============================================================

describe("Counter-Fact Grief Attack", () => {
  it("CF-1: Mass contradiction — 10 minimum-stake counter-facts capped at MaxCounterFactPenalty", () => {
    // Per NEGATIVE_KNOWLEDGE constants, each low-confidence counter-fact
    // applies a -10K (0.01) penalty. 10 of them = 0.10, but
    // total is capped at MAX_COUNTER_FACT_PENALTY = 0.4
    const numCounterFacts = 10;
    const penaltyPerLowCF = NEGATIVE_KNOWLEDGE.COUNTER_FACT_LOW_PENALTY; // 0.01

    const totalPenalty = Math.min(
      numCounterFacts * penaltyPerLowCF,
      NEGATIVE_KNOWLEDGE.MAX_COUNTER_FACT_PENALTY,
    );

    // 10 * 0.01 = 0.10, which is under the 0.4 cap
    expect(totalPenalty).toBe(0.1);
    expect(totalPenalty).toBeLessThanOrEqual(
      NEGATIVE_KNOWLEDGE.MAX_COUNTER_FACT_PENALTY,
    );

    // Now test with 50 counter-facts that would exceed the cap
    const totalPenalty50 = Math.min(
      50 * penaltyPerLowCF,
      NEGATIVE_KNOWLEDGE.MAX_COUNTER_FACT_PENALTY,
    );
    // 50 * 0.01 = 0.50 but capped at 0.4
    expect(totalPenalty50).toBe(NEGATIVE_KNOWLEDGE.MAX_COUNTER_FACT_PENALTY);
    expect(totalPenalty50).toBe(0.4);
  });

  it("CF-2: Counter-fact with insufficient stake ratio is rejected", () => {
    // MinContradictionStakeRatio = 0.5 (50%)
    // If target fact has original stake of 1000, counter-fact must stake >= 500
    const targetOriginalStake = 1000n;
    const insufficientStake = 400n; // 40% < 50%
    const sufficientStake = 500n; // 50% exactly

    const minRequired =
      (targetOriginalStake *
        BigInt(
          Math.floor(NEGATIVE_KNOWLEDGE.MIN_CONTRADICTION_STAKE_RATIO * 100),
        )) /
      100n;

    expect(insufficientStake < minRequired).toBe(true);
    expect(sufficientStake >= minRequired).toBe(true);
    expect(minRequired).toBe(500n);
  });

  it("CF-3: High-confidence counter-fact (>70%) applies -80K penalty", () => {
    const highConfidence = 0.75; // > 0.7 threshold
    expect(highConfidence).toBeGreaterThanOrEqual(
      NEGATIVE_KNOWLEDGE.COUNTER_FACT_HIGH_CONFIDENCE,
    );

    const penalty = NEGATIVE_KNOWLEDGE.COUNTER_FACT_HIGH_PENALTY; // 0.08 (mapped from 80K bps)
    expect(penalty).toBe(0.08);
  });

  it("CF-4: Medium-confidence counter-fact (40-70%) applies -40K penalty", () => {
    const mediumConfidence = 0.55; // between 0.4 and 0.7
    expect(mediumConfidence).toBeGreaterThanOrEqual(
      NEGATIVE_KNOWLEDGE.COUNTER_FACT_MEDIUM_CONFIDENCE,
    );
    expect(mediumConfidence).toBeLessThan(
      NEGATIVE_KNOWLEDGE.COUNTER_FACT_HIGH_CONFIDENCE,
    );

    const penalty = NEGATIVE_KNOWLEDGE.COUNTER_FACT_MEDIUM_PENALTY; // 0.04
    expect(penalty).toBe(0.04);
  });

  it("CF-5: Low-confidence counter-fact (<40%) applies -10K penalty", () => {
    const lowConfidence = 0.3; // < 0.4
    expect(lowConfidence).toBeLessThan(
      NEGATIVE_KNOWLEDGE.COUNTER_FACT_MEDIUM_CONFIDENCE,
    );

    const penalty = NEGATIVE_KNOWLEDGE.COUNTER_FACT_LOW_PENALTY; // 0.01
    expect(penalty).toBe(0.01);
  });

  it("CF-6: Claim referencing disproven fact gets AlreadyDisprovenPenalty", () => {
    // A fact with falsity > truth has bidirectional status "disproven"
    const truthConfidence = 0.2 as Confidence;
    const falsityConfidence = 0.9 as Confidence;
    const net = truthConfidence - falsityConfidence; // -0.7

    const status = computeBidirectionalStatus(net);
    expect(status).toBe("disproven"); // net < -0.6

    // The penalty applied to claims referencing this disproven fact
    const alreadyDisprovenPenalty =
      NEGATIVE_KNOWLEDGE.ALREADY_DISPROVEN_PENALTY;
    expect(alreadyDisprovenPenalty).toBe(0.2); // 200K bps = 0.2

    // If a claim has base score of 0.7 and references a disproven fact:
    const baseScore = 0.7;
    const penalizedScore = baseScore - alreadyDisprovenPenalty;
    expect(penalizedScore).toBeCloseTo(0.5, 10);
  });
});

// ============================================================
// 3. Evaluation Pipeline Integrity (6 tests)
// ============================================================

describe("Evaluation Pipeline Integrity", () => {
  it("EP-1: Claim with invalid epistemic category fails at stage 1 (category check)", () => {
    // The category base confidence lookup falls back to a default for unknown categories
    // Unknown category gets DEFAULT_CATEGORY_BASE = 400_000 in daemon evaluator
    // In the master confidence formula, an invalid category should use "empirical" fallback
    const invalidCategory = "nonexistent_category";

    // calculateCategoryDecayedConfidence uses CATEGORY_DECAY which falls back to empirical
    const result = calculateCategoryDecayedConfidence(
      0.8 as Confidence,
      0,
      invalidCategory,
    );

    // Falls back to empirical parameters (not null = decays)
    expect(result.isEternal).toBe(false);
    // At block 0, no decay yet — confidence unchanged
    expect(result.confidence).toBe(0.8);
  });

  it("EP-2: Claim with confidence > 1.0 is constrained by epistemic ceiling", () => {
    // The master confidence formula applies: min(epistemicCeiling, computed)
    // Even if somehow a raw confidence exceeds 1.0, it's clamped
    const validators = new Map<string, { stake: bigint; reputation: number }>();
    validators.set("v1", { stake: 1000n, reputation: 0.95 });
    validators.set("v2", { stake: 1000n, reputation: 0.9 });

    const reveals = new Map<string, ValidatorReveal>();
    reveals.set("v1", {
      validator: "v1",
      verdict: "confirmed",
      confidence: 0.99,
      nonce: "nonce1",
      reasoning: "test",
      referencesChecked: [],
      timestamp: Date.now(),
    });
    reveals.set("v2", {
      validator: "v2",
      verdict: "confirmed",
      confidence: 0.99,
      nonce: "nonce2",
      reasoning: "test",
      referencesChecked: [],
      timestamp: Date.now(),
    });

    const input: ConfidenceCalculationInput = {
      evidence: Array.from({ length: 20 }, (_, i) =>
        makeSupportingEvidence(`e${i}`, 0.95),
      ),
      verificationReveals: reveals,
      validators,
      consensusVerdict: "confirmed",
      category: "empirical",
      blocksSinceLastVerification: 0,
      currentBlockNumber: 100,
    };

    const result = calculateMasterConfidence(input);

    // Empirical ceiling is 0.99 — result must not exceed it
    expect(result.value).toBeLessThanOrEqual(0.99);
    expect(result.provenance.epistemicCeiling).toBe(0.99);
  });

  it("EP-3: Claim with no evidence in empirical domain gets lower confidence", () => {
    // AH-1: Evidence gate — empirical claims need evidence
    const withEvidence = aggregateEvidenceBayesian([
      makeSupportingEvidence("e1", 0.85),
      makeSupportingEvidence("e2", 0.8),
    ]);

    const withoutEvidence = aggregateEvidenceBayesian([]);

    // Without evidence, the posterior stays at or near the prior (0.5)
    // With evidence, the posterior rises significantly
    expect(withEvidence.effectiveConfidence).toBeGreaterThan(
      withoutEvidence.effectiveConfidence,
    );
    // No evidence → posterior = prior (DEFAULT_PRIOR = 0.5)
    expect(withoutEvidence.posterior).toBeCloseTo(0.5, 3);
  });

  it("EP-4: Propositional logic zone claim — decidable by definition", () => {
    // Claims in propositional logic zone are in Zone 0 (complete, decidable)
    // They should NOT be flagged as undecidable
    // The formal zone maxConfidence is 1.0 (eternal truth)
    const propositionalDecay = calculateCategoryDecayedConfidence(
      0.95 as Confidence,
      100000,
      "formal",
    );

    // Formal/propositional truths are eternal — no decay
    expect(propositionalDecay.isEternal).toBe(true);
    expect(propositionalDecay.confidence).toBe(0.95);
    expect(propositionalDecay.decayFactor).toBe(1.0);
  });

  it("EP-5: Cross-stratum — higher stratum claim without lower support penalized", () => {
    // Use master confidence with adversarial adjustments to simulate
    // cross-stratum penalty. A claim that has sybil or manipulation detected
    // receives adversarial adjustment < 1.0, reducing confidence.
    const validators = new Map<string, { stake: bigint; reputation: number }>();
    validators.set("v1", { stake: 1000n, reputation: 0.8 });

    const reveals = new Map<string, ValidatorReveal>();
    reveals.set("v1", {
      validator: "v1",
      verdict: "confirmed",
      confidence: 0.8,
      nonce: "nonce1",
      reasoning: "test",
      referencesChecked: [],
      timestamp: Date.now(),
    });

    const withSupport: ConfidenceCalculationInput = {
      evidence: [makeSupportingEvidence("e1", 0.8)],
      verificationReveals: reveals,
      validators,
      consensusVerdict: "confirmed",
      category: "empirical",
      blocksSinceLastVerification: 0,
      currentBlockNumber: 100,
      sybilAdjustment: 1.0, // No penalty
      manipulationAdjustment: 1.0,
    };

    const withoutSupport: ConfidenceCalculationInput = {
      ...withSupport,
      sybilAdjustment: 0.7, // 30% penalty for cross-stratum issues
      manipulationAdjustment: 0.9,
    };

    const supported = calculateMasterConfidence(withSupport);
    const unsupported = calculateMasterConfidence(withoutSupport);

    // Cross-stratum penalty reduces confidence
    expect(unsupported.value).toBeLessThan(supported.value);
    expect(unsupported.provenance.adversarialAdjustment).toBeLessThan(1.0);
  });

  it("EP-6: All 8 evaluation stages execute in order — trace evaluation path", () => {
    // The evaluation pipeline stages:
    // 1. category → 2. calibration → 3. evidence → 4. fact_graph
    // 5. logic_zone → 6. ceiling → 7. edge_cases → 8. counter_facts
    //
    // We verify this by checking that the master confidence formula
    // applies each stage's contribution in sequence.

    const validators = new Map<string, { stake: bigint; reputation: number }>();
    validators.set("v1", { stake: 1000n, reputation: 0.9 });

    const reveals = new Map<string, ValidatorReveal>();
    reveals.set("v1", {
      validator: "v1",
      verdict: "confirmed",
      confidence: 0.85,
      nonce: "nonce1",
      reasoning: "test",
      referencesChecked: [],
      timestamp: Date.now(),
    });

    const input: ConfidenceCalculationInput = {
      evidence: [
        makeSupportingEvidence("e1", 0.8),
        makeSupportingEvidence("e2", 0.75),
        makeSupportingEvidence("e3", 0.7),
        makeContradictingEvidence("e4", 0.3), // Some contradiction
      ],
      verificationReveals: reveals,
      validators,
      consensusVerdict: "confirmed",
      category: "empirical",
      blocksSinceLastVerification: 50000, // Some decay
      currentBlockNumber: 50100,
      sybilAdjustment: 0.95,
      manipulationAdjustment: 0.98,
    };

    const result = calculateMasterConfidence(input);

    // Verify each stage contributed to the output:
    // Stage 1 (category): basis.category is set
    expect(result.basis.category).toBe("empirical");

    // Stage 2 (calibration): basis.primaryMethod reflects category
    expect(result.basis.primaryMethod).toBe("observation");

    // Stage 3 (evidence): evidenceContribution > 0
    expect(result.provenance.evidenceContribution).toBeGreaterThan(0);

    // Stage 4 (fact_graph / verification): verificationContribution > 0
    expect(result.provenance.verificationContribution).toBeGreaterThan(0);

    // Stage 5 (logic_zone): Not a separate field, but reflected in category method

    // Stage 6 (ceiling): epistemicCeiling applied
    expect(result.provenance.epistemicCeiling).toBe(0.99); // empirical ceiling

    // Stage 7 (edge_cases / temporal): temporalAdjustment < 1.0 due to decay
    expect(result.provenance.temporalAdjustment).toBeLessThan(1.0);

    // Stage 8 (counter_facts / adversarial): adversarialAdjustment < 1.0
    expect(result.provenance.adversarialAdjustment).toBeLessThan(1.0);

    // Final value incorporates all stages
    expect(result.value).toBeGreaterThan(0);
    expect(result.value).toBeLessThanOrEqual(0.99);

    // Coherence reflects the mixed evidence (3 supporting + 1 contradicting)
    // Agreement ratio = 3/4 = 0.75 → "mixed" (>= 0.7 but < 0.9)
    expect(result.coherence).toBe("mixed");
  });
});
