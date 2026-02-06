/**
 * Gödelian Analysis Tests
 *
 * Tests self-reference detection, decidability assessment,
 * paradox rejection, and undecidability handling.
 */

import { describe, it, expect } from "vitest";
import {
  analyzeGoedelianStatus,
  assessDecidability,
  handleUndecidable,
  isMetaSystemClaim,
  respondToMetaClaim,
  GOEDELIAN_PROMISE,
} from "../../../core/logic/goedel.js";

// ============ Self-Reference Detection ============

describe("analyzeGoedelianStatus", () => {
  it("should detect paradoxical statements (Liar paradox)", () => {
    const result = analyzeGoedelianStatus("This statement is false");
    expect(result.status).toBe("paradoxical");
    expect(result.selfReferential).toBe(true);
    expect(result.paradoxical).toBe(true);
    expect(result.goedelian).toBe(false);
  });

  it("should detect Russell-like paradox", () => {
    const result = analyzeGoedelianStatus(
      "the set of all sets that do not contain themselves",
    );
    expect(result.paradoxical).toBe(true);
    expect(result.status).toBe("paradoxical");
  });

  it("should detect Gödelian sentences (true but unprovable)", () => {
    const result = analyzeGoedelianStatus(
      "This statement cannot be proven in this system",
    );
    expect(result.status).toBe("goedelian");
    expect(result.selfReferential).toBe(true);
    expect(result.goedelian).toBe(true);
    expect(result.paradoxical).toBe(false);
  });

  it("should detect provability self-reference", () => {
    const result = analyzeGoedelianStatus("unprovable but true");
    expect(result.goedelian).toBe(true);
    expect(result.status).toBe("goedelian");
  });

  it("should detect system limitation statements", () => {
    const result = analyzeGoedelianStatus(
      "no consistent system can prove its own consistency",
    );
    expect(result.goedelian).toBe(true);
  });

  it("should detect general self-reference", () => {
    const result = analyzeGoedelianStatus("this claim refers to itself");
    expect(result.selfReferential).toBe(true);
    expect(result.status).toBe("goedelian");
  });

  it("should classify benign claims as benign", () => {
    const result = analyzeGoedelianStatus("Water boils at 100 degrees Celsius");
    expect(result.status).toBe("benign");
    expect(result.selfReferential).toBe(false);
    expect(result.paradoxical).toBe(false);
    expect(result.goedelian).toBe(false);
  });

  it("should classify simple arithmetic as benign", () => {
    const result = analyzeGoedelianStatus("2 + 2 = 4");
    expect(result.status).toBe("benign");
  });

  it("should check formal notation for patterns", () => {
    const result = analyzeGoedelianStatus(
      "A formal statement",
      "true but not provable in T",
    );
    expect(result.goedelian).toBe(true);
  });

  it("should prioritize paradox over Gödelian detection", () => {
    // Liar paradox patterns should be caught before Gödelian patterns
    const result = analyzeGoedelianStatus(
      "This statement is false and cannot be proven",
    );
    expect(result.paradoxical).toBe(true);
    expect(result.status).toBe("paradoxical");
  });
});

// ============ Decidability Assessment ============

describe("assessDecidability", () => {
  it("should mark benign claims in decidable zones as decidable", () => {
    const result = assessDecidability(
      "Water boils at 100 degrees",
      "propositional",
    );
    expect(result.decidability).toBe("decidable");
    expect(result.canBeResolved).toBe(true);
    expect(result.suggestedResolutions).toHaveLength(0);
  });

  it("should mark benign claims in presburger as decidable", () => {
    const result = assessDecidability("3 + 5 = 8", "presburger");
    expect(result.decidability).toBe("decidable");
  });

  it("should mark Gödelian claims in peano as undecidable_self_ref", () => {
    const result = assessDecidability(
      "This proposition cannot be proved in this system",
      "peano",
    );
    expect(result.decidability).toBe("undecidable_self_ref");
    expect(result.canBeResolved).toBe(true);
    expect(result.suggestedResolutions).toContain("oracle");
    expect(result.suggestedResolutions).toContain("consensus");
    expect(result.suggestedResolutions).toContain("accept_undecidability");
  });

  it("should mark Gödelian claims in set_theory as undecidable_self_ref", () => {
    const result = assessDecidability(
      "The incompleteness of this system is acknowledged",
      "set_theory",
    );
    expect(result.decidability).toBe("undecidable_self_ref");
  });

  it("should mark self-referential empirical claims as empirical", () => {
    // Note: benign claims in empirical zone hit the decidable path first
    // (goedelApplies is false for empirical, and claim is not goedelian).
    // Only self-referential claims in empirical zone reach the empirical branch.
    const result = assessDecidability(
      "the system itself has observed this behavior",
      "empirical",
    );
    expect(result.decidability).toBe("empirical");
    expect(result.suggestedResolutions).toContain("oracle");
    expect(result.suggestedResolutions).toContain("consensus");
  });

  it("should mark benign empirical claims as decidable", () => {
    // Benign claims in empirical zone return decidable because
    // goedelApplies("empirical") is false and the claim is not goedelian
    const result = assessDecidability(
      "GDP growth was 3% last year",
      "empirical",
    );
    expect(result.decidability).toBe("decidable");
  });

  it("should reject paradoxical claims with unknown decidability", () => {
    const result = assessDecidability(
      "This statement is false",
      "propositional",
    );
    expect(result.decidability).toBe("unknown");
    expect(result.canBeResolved).toBe(false);
    expect(result.suggestedResolutions).toHaveLength(0);
  });

  it("should handle claims in Gödel zones without detected self-reference", () => {
    const result = assessDecidability(
      "Some mathematical statement about primes",
      "peano",
    );
    expect(result.decidability).toBe("unknown");
    expect(result.reason).toContain("unknown");
    expect(result.canBeResolved).toBe(true);
  });

  it("should include formal notation in analysis", () => {
    const result = assessDecidability(
      "A formal claim",
      "peano",
      "unprovable within T",
    );
    expect(result.decidability).toBe("undecidable_self_ref");
  });
});

// ============ Undecidability Handling ============

describe("handleUndecidable", () => {
  it("should create resolution for accept_undecidability", () => {
    const result = handleUndecidable(
      "claim_123",
      "undecidable_self_ref",
      "accept_undecidability",
    );
    expect(result.claimId).toBe("claim_123");
    expect(result.originalDecidability).toBe("undecidable_self_ref");
    expect(result.resolutionPath).toBe("accept_undecidability");
    expect(result.finalConfidence).toBe(0.5);
  });

  it("should assign oracle confidence of 0.8", () => {
    const result = handleUndecidable("claim_456", "undecidable", "oracle");
    expect(result.finalConfidence).toBe(0.8);
  });

  it("should assign consensus confidence of 0.7", () => {
    const result = handleUndecidable("claim_789", "undecidable", "consensus");
    expect(result.finalConfidence).toBe(0.7);
  });

  it("should assign new_axiom confidence of 1.0", () => {
    const result = handleUndecidable(
      "claim_abc",
      "undecidable_timeout",
      "new_axiom",
    );
    expect(result.finalConfidence).toBe(1.0);
  });
});

// ============ Meta-System Claims ============

describe("isMetaSystemClaim", () => {
  it("should detect completeness claims", () => {
    expect(isMetaSystemClaim("this system is complete")).toBe(true);
  });

  it("should detect consistency claims", () => {
    expect(isMetaSystemClaim("this system is consistent")).toBe(true);
  });

  it("should detect decidability claims", () => {
    expect(isMetaSystemClaim("this system is decidable")).toBe(true);
  });

  it("should detect 'all truths provable' claims", () => {
    expect(isMetaSystemClaim("all true statements can be proven")).toBe(true);
  });

  it("should detect 'every statement decidable' claims", () => {
    expect(isMetaSystemClaim("every claim is decidable")).toBe(true);
  });

  it("should not flag normal claims", () => {
    expect(isMetaSystemClaim("the sky is blue")).toBe(false);
  });
});

describe("respondToMetaClaim", () => {
  it("should respond to completeness claims with Gödel I", () => {
    const response = respondToMetaClaim("this system is complete");
    expect(response).toContain("incomplete");
    expect(response).toContain("First Theorem");
  });

  it("should respond to consistency claims with Gödel II", () => {
    const response = respondToMetaClaim("this system is consistent");
    expect(response).toContain("Second Theorem");
  });

  it("should respond to decidability claims", () => {
    const response = respondToMetaClaim("this system is decidable");
    expect(response).toContain("undecidable");
  });

  it("should provide generic response for other meta-claims", () => {
    const response = respondToMetaClaim("something about the system");
    expect(response).toContain("meta-axioms");
  });
});

// ============ Gödelian Promise ============

describe("GOEDELIAN_PROMISE", () => {
  it("should be a non-empty string", () => {
    expect(typeof GOEDELIAN_PROMISE).toBe("string");
    expect(GOEDELIAN_PROMISE.length).toBeGreaterThan(100);
  });

  it("should mention incompleteness", () => {
    expect(GOEDELIAN_PROMISE).toContain("INCOMPLETENESS");
  });

  it("should mention consistency", () => {
    expect(GOEDELIAN_PROMISE).toContain("CONSISTENCY");
  });

  it("should mention undecidability", () => {
    expect(GOEDELIAN_PROMISE).toContain("UNDECIDABILITY");
  });

  it("should mention honesty", () => {
    expect(GOEDELIAN_PROMISE).toContain("HONESTY");
  });
});
