/**
 * Logic Layer Types Tests
 *
 * Tests exported constants, type structures, meta-axioms,
 * inference strengths, and zone hierarchy from the types module.
 */

import { describe, it, expect } from "vitest";
import {
  ZONE_HIERARCHY,
  INFERENCE_STRENGTH,
  META_AXIOMS,
  type LogicZone,
  type Decidability,
  type GoedelianStatus,
  type AxiomType,
  type InferenceRule,
  type ProofStrategy,
  type ResolutionPath,
  type MetaAxiom,
  type LogicalAxiom,
  type ZoneProperties,
  type ProofStep,
  type FormalProof,
  type ProofVerification,
  type ProofSearchConfig,
  type ProofSearchResult,
  type UndecidableResolution,
} from "../../../core/logic/types.js";

// ============ ZONE_HIERARCHY Constant ============

describe("ZONE_HIERARCHY", () => {
  it("should define all five logic zones", () => {
    const zones: LogicZone[] = [
      "propositional",
      "presburger",
      "peano",
      "set_theory",
      "empirical",
    ];
    for (const zone of zones) {
      expect(ZONE_HIERARCHY[zone]).toBeDefined();
      expect(typeof ZONE_HIERARCHY[zone]).toBe("number");
    }
  });

  it("should assign propositional the lowest level (0)", () => {
    expect(ZONE_HIERARCHY.propositional).toBe(0);
  });

  it("should assign empirical the highest level (4)", () => {
    expect(ZONE_HIERARCHY.empirical).toBe(4);
  });

  it("should maintain strict ascending order from propositional to empirical", () => {
    expect(ZONE_HIERARCHY.propositional).toBeLessThan(
      ZONE_HIERARCHY.presburger,
    );
    expect(ZONE_HIERARCHY.presburger).toBeLessThan(ZONE_HIERARCHY.peano);
    expect(ZONE_HIERARCHY.peano).toBeLessThan(ZONE_HIERARCHY.set_theory);
    expect(ZONE_HIERARCHY.set_theory).toBeLessThan(ZONE_HIERARCHY.empirical);
  });

  it("should use consecutive integers starting at 0", () => {
    const values = Object.values(ZONE_HIERARCHY);
    values.sort((a, b) => a - b);
    for (let i = 0; i < values.length; i++) {
      expect(values[i]).toBe(i);
    }
  });

  it("should contain exactly 5 entries", () => {
    expect(Object.keys(ZONE_HIERARCHY)).toHaveLength(5);
  });
});

// ============ INFERENCE_STRENGTH Constant ============

describe("INFERENCE_STRENGTH", () => {
  it("should define strengths for all inference rules", () => {
    const rules: InferenceRule[] = [
      "modus_ponens",
      "modus_tollens",
      "hypothetical_syllogism",
      "disjunctive_syllogism",
      "conjunction_intro",
      "conjunction_elim",
      "disjunction_intro",
      "double_negation",
      "universal_instantiation",
      "existential_generalization",
      "induction",
      "substitution",
      "definition_expansion",
      "axiom_application",
    ];
    for (const rule of rules) {
      expect(INFERENCE_STRENGTH[rule]).toBeDefined();
      expect(typeof INFERENCE_STRENGTH[rule]).toBe("number");
    }
  });

  it("should have exactly 14 inference rules", () => {
    expect(Object.keys(INFERENCE_STRENGTH)).toHaveLength(14);
  });

  it("should assign all propositional rules strength 1.0 (truth-preserving)", () => {
    const propositionalRules: InferenceRule[] = [
      "modus_ponens",
      "modus_tollens",
      "hypothetical_syllogism",
      "disjunctive_syllogism",
      "conjunction_intro",
      "conjunction_elim",
      "disjunction_intro",
      "double_negation",
    ];
    for (const rule of propositionalRules) {
      expect(INFERENCE_STRENGTH[rule]).toBe(1.0);
    }
  });

  it("should assign first-order quantifier rules strength 0.99", () => {
    expect(INFERENCE_STRENGTH.universal_instantiation).toBe(0.99);
    expect(INFERENCE_STRENGTH.existential_generalization).toBe(0.99);
  });

  it("should assign induction strength 0.95 (sound but verification-intensive)", () => {
    expect(INFERENCE_STRENGTH.induction).toBe(0.95);
  });

  it("should assign substitution strength 1.0 (equality preserving)", () => {
    expect(INFERENCE_STRENGTH.substitution).toBe(1.0);
  });

  it("should assign derived rules the weakest strength (0.9)", () => {
    expect(INFERENCE_STRENGTH.definition_expansion).toBe(0.9);
    expect(INFERENCE_STRENGTH.axiom_application).toBe(0.9);
  });

  it("should keep all strengths in range (0, 1.0]", () => {
    for (const [rule, strength] of Object.entries(INFERENCE_STRENGTH)) {
      expect(strength).toBeGreaterThan(0);
      expect(strength).toBeLessThanOrEqual(1.0);
    }
  });

  it("should have graduated strengths forming a natural ordering", () => {
    // Propositional > First-order > Induction > Derived
    expect(INFERENCE_STRENGTH.modus_ponens).toBeGreaterThan(
      INFERENCE_STRENGTH.universal_instantiation,
    );
    expect(INFERENCE_STRENGTH.universal_instantiation).toBeGreaterThan(
      INFERENCE_STRENGTH.induction,
    );
    expect(INFERENCE_STRENGTH.induction).toBeGreaterThan(
      INFERENCE_STRENGTH.definition_expansion,
    );
  });
});

// ============ META_AXIOMS Constant ============

describe("META_AXIOMS", () => {
  it("should contain exactly 4 meta-axioms", () => {
    expect(META_AXIOMS).toHaveLength(4);
  });

  it("should have unique IDs across all meta-axioms", () => {
    const ids = META_AXIOMS.map((a) => a.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("every meta-axiom should have type 'meta'", () => {
    for (const axiom of META_AXIOMS) {
      expect(axiom.type).toBe("meta");
    }
  });

  it("every meta-axiom should have confidence exactly 1.0", () => {
    for (const axiom of META_AXIOMS) {
      expect(axiom.confidence).toBe(1.0);
    }
  });

  it("every meta-axiom should be established at block 0 (genesis)", () => {
    for (const axiom of META_AXIOMS) {
      expect(axiom.establishedAt).toBe(0);
    }
  });

  it("every meta-axiom should have a non-empty goedelReference", () => {
    for (const axiom of META_AXIOMS) {
      expect(axiom.goedelReference).toBeTruthy();
      expect(typeof axiom.goedelReference).toBe("string");
      expect(axiom.goedelReference.length).toBeGreaterThan(5);
    }
  });

  it("every meta-axiom should have a non-empty philosophicalNote", () => {
    for (const axiom of META_AXIOMS) {
      expect(axiom.philosophicalNote).toBeTruthy();
      expect(typeof axiom.philosophicalNote).toBe("string");
      expect(axiom.philosophicalNote.length).toBeGreaterThan(10);
    }
  });

  it("every meta-axiom should have formalNotation", () => {
    for (const axiom of META_AXIOMS) {
      expect(axiom.formalNotation).toBeTruthy();
    }
  });

  it("every meta-axiom should reside in the peano zone", () => {
    for (const axiom of META_AXIOMS) {
      expect(axiom.zone).toBe("peano");
    }
  });

  describe("meta_1 (First Incompleteness)", () => {
    it("should reference Godel-Rosser theorem", () => {
      const meta1 = META_AXIOMS.find((a) => a.id === "meta_1")!;
      expect(meta1).toBeDefined();
      expect(meta1.goedelReference).toContain("Rosser");
      expect(meta1.goedelReference).toContain("1931");
    });

    it("should use Provable_S in formal notation", () => {
      const meta1 = META_AXIOMS.find((a) => a.id === "meta_1")!;
      expect(meta1.formalNotation).toContain("Provable_S");
    });

    it("should state incompleteness in the claim", () => {
      const meta1 = META_AXIOMS.find((a) => a.id === "meta_1")!;
      expect(meta1.claim.toLowerCase()).toContain("incomplete");
    });
  });

  describe("meta_2 (Independence)", () => {
    it("should describe independence from axioms", () => {
      const meta2 = META_AXIOMS.find((a) => a.id === "meta_2")!;
      expect(meta2).toBeDefined();
      expect(meta2.claim.toLowerCase()).toContain("independence");
    });

    it("should distinguish independence from algorithmic undecidability", () => {
      const meta2 = META_AXIOMS.find((a) => a.id === "meta_2")!;
      expect(meta2.philosophicalNote).toContain("independent");
      expect(meta2.philosophicalNote).toContain("Turing");
    });
  });

  describe("meta_3 (Second Incompleteness)", () => {
    it("should reference consistency and Con(S) in formal notation", () => {
      const meta3 = META_AXIOMS.find((a) => a.id === "meta_3")!;
      expect(meta3).toBeDefined();
      expect(meta3.formalNotation).toContain("Con(S)");
    });

    it("should reference Hilbert-Bernays-Lob derivability conditions", () => {
      const meta3 = META_AXIOMS.find((a) => a.id === "meta_3")!;
      expect(meta3.goedelReference).toContain("Hilbert-Bernays");
    });

    it("should note the implicit consistency assumption", () => {
      const meta3 = META_AXIOMS.find((a) => a.id === "meta_3")!;
      expect(meta3.philosophicalNote).toContain("contraposition");
    });
  });

  describe("meta_4 (Lower Zone Completeness)", () => {
    it("should assert conservative extension property", () => {
      const meta4 = META_AXIOMS.find((a) => a.id === "meta_4")!;
      expect(meta4).toBeDefined();
      expect(meta4.formalNotation).toContain("Conservative");
      expect(meta4.formalNotation).toContain("Complete");
    });

    it("should reference Presburger and Post theorems", () => {
      const meta4 = META_AXIOMS.find((a) => a.id === "meta_4")!;
      expect(meta4.goedelReference).toContain("Presburger");
      expect(meta4.goedelReference).toContain("Post");
    });

    it("should reference Shoenfield for conservative extensions", () => {
      const meta4 = META_AXIOMS.find((a) => a.id === "meta_4")!;
      expect(meta4.goedelReference).toContain("Shoenfield");
      expect(meta4.goedelReference).toContain("conservative");
    });
  });
});

// ============ Type Structure Validation ============

describe("Type Structure Conformance", () => {
  it("META_AXIOMS entries should conform to MetaAxiom interface", () => {
    for (const axiom of META_AXIOMS) {
      // Required MetaAxiom fields
      expect(typeof axiom.id).toBe("string");
      expect(axiom.type).toBe("meta");
      expect(typeof axiom.zone).toBe("string");
      expect(typeof axiom.claim).toBe("string");
      expect(axiom.confidence).toBe(1.0);
      expect(typeof axiom.establishedAt).toBe("number");
      expect(typeof axiom.goedelReference).toBe("string");
      expect(typeof axiom.philosophicalNote).toBe("string");
    }
  });

  it("ZONE_HIERARCHY should be a complete Record<LogicZone, number>", () => {
    const expectedZones: LogicZone[] = [
      "propositional",
      "presburger",
      "peano",
      "set_theory",
      "empirical",
    ];
    for (const zone of expectedZones) {
      expect(typeof ZONE_HIERARCHY[zone]).toBe("number");
    }
  });

  it("INFERENCE_STRENGTH should be a complete Record<InferenceRule, number>", () => {
    const expectedRules: InferenceRule[] = [
      "modus_ponens",
      "modus_tollens",
      "hypothetical_syllogism",
      "disjunctive_syllogism",
      "conjunction_intro",
      "conjunction_elim",
      "disjunction_intro",
      "double_negation",
      "universal_instantiation",
      "existential_generalization",
      "induction",
      "substitution",
      "definition_expansion",
      "axiom_application",
    ];
    for (const rule of expectedRules) {
      expect(typeof INFERENCE_STRENGTH[rule]).toBe("number");
    }
  });
});
