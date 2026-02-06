/**
 * Logic Zones Tests
 *
 * Tests zone properties, axiom inheritance, zone classification,
 * cross-zone referencing, and meta-axiom validation.
 */

import { describe, it, expect } from "vitest";
import {
  ZONE_PROPERTIES,
  PROPOSITIONAL_AXIOMS,
  PRESBURGER_AXIOMS,
  PEANO_AXIOMS,
  PROTOCOL_AXIOMS,
  getZoneProperties,
  isZoneDecidable,
  isZoneComplete,
  goedelApplies,
  getZoneAxioms,
  classifyClaimZone,
  canReference,
  intersectZones,
} from "../../../core/logic/zones.js";
import {
  META_AXIOMS,
  ZONE_HIERARCHY,
  INFERENCE_STRENGTH,
  type MetaAxiom,
} from "../../../core/logic/types.js";

// ============ Zone Properties ============

describe("Zone Properties", () => {
  it("propositional zone should be complete and decidable", () => {
    const props = getZoneProperties("propositional");
    expect(props.complete).toBe(true);
    expect(props.decidable).toBe(true);
    expect(props.goedelApplies).toBe(false);
    expect(props.consistencyProof).toBe("internal");
  });

  it("presburger zone should be complete and decidable", () => {
    const props = getZoneProperties("presburger");
    expect(props.complete).toBe(true);
    expect(props.decidable).toBe(true);
    expect(props.goedelApplies).toBe(false);
    expect(props.consistencyProof).toBe("internal");
  });

  it("peano zone should be incomplete and undecidable", () => {
    const props = getZoneProperties("peano");
    expect(props.complete).toBe(false);
    expect(props.decidable).toBe(false);
    expect(props.goedelApplies).toBe(true);
    expect(props.consistencyProof).toBe("assumed");
  });

  it("set_theory zone should be incomplete and undecidable", () => {
    const props = getZoneProperties("set_theory");
    expect(props.complete).toBe(false);
    expect(props.decidable).toBe(false);
    expect(props.goedelApplies).toBe(true);
    expect(props.consistencyProof).toBe("assumed");
  });

  it("empirical zone should not have Gödel apply", () => {
    const props = getZoneProperties("empirical");
    expect(props.complete).toBe(false);
    expect(props.decidable).toBe(false);
    expect(props.goedelApplies).toBe(false);
    expect(props.consistencyProof).toBe("external");
  });
});

describe("Zone Helper Functions", () => {
  it("isZoneDecidable should return true for propositional and presburger", () => {
    expect(isZoneDecidable("propositional")).toBe(true);
    expect(isZoneDecidable("presburger")).toBe(true);
    expect(isZoneDecidable("peano")).toBe(false);
    expect(isZoneDecidable("set_theory")).toBe(false);
    expect(isZoneDecidable("empirical")).toBe(false);
  });

  it("isZoneComplete should return true only for propositional and presburger", () => {
    expect(isZoneComplete("propositional")).toBe(true);
    expect(isZoneComplete("presburger")).toBe(true);
    expect(isZoneComplete("peano")).toBe(false);
    expect(isZoneComplete("set_theory")).toBe(false);
    expect(isZoneComplete("empirical")).toBe(false);
  });

  it("goedelApplies should return true only for peano and set_theory", () => {
    expect(goedelApplies("propositional")).toBe(false);
    expect(goedelApplies("presburger")).toBe(false);
    expect(goedelApplies("peano")).toBe(true);
    expect(goedelApplies("set_theory")).toBe(true);
    expect(goedelApplies("empirical")).toBe(false);
  });
});

// ============ Zone Hierarchy ============

describe("Zone Hierarchy", () => {
  it("should have correct ordering", () => {
    expect(ZONE_HIERARCHY.propositional).toBe(0);
    expect(ZONE_HIERARCHY.presburger).toBe(1);
    expect(ZONE_HIERARCHY.peano).toBe(2);
    expect(ZONE_HIERARCHY.set_theory).toBe(3);
    expect(ZONE_HIERARCHY.empirical).toBe(4);
  });

  it("should have strictly increasing levels", () => {
    const levels = Object.values(ZONE_HIERARCHY);
    for (let i = 1; i < levels.length; i++) {
      expect(levels[i]).toBeGreaterThan(levels[i - 1]);
    }
  });
});

// ============ Axiom Sets ============

describe("Axiom Sets", () => {
  it("propositional axioms should all be in propositional zone", () => {
    for (const axiom of PROPOSITIONAL_AXIOMS) {
      expect(axiom.zone).toBe("propositional");
      expect(axiom.type).toBe("logical");
      expect(axiom.confidence).toBe(1.0);
    }
  });

  it("propositional axioms should include fundamental laws", () => {
    const ids = PROPOSITIONAL_AXIOMS.map((a) => a.id);
    expect(ids).toContain("prop_1"); // Identity
    expect(ids).toContain("prop_2"); // Non-contradiction
    expect(ids).toContain("prop_3"); // Excluded middle
  });

  it("presburger axioms should all be in presburger zone", () => {
    for (const axiom of PRESBURGER_AXIOMS) {
      expect(axiom.zone).toBe("presburger");
      expect(axiom.type).toBe("arithmetic");
      expect(axiom.confidence).toBe(1.0);
    }
  });

  it("presburger axioms should NOT include multiplication", () => {
    const claims = PRESBURGER_AXIOMS.map((a) => a.claim.toLowerCase());
    for (const claim of claims) {
      expect(claim).not.toContain("multiplication");
    }
  });

  it("peano axioms should include multiplication (trigger for incompleteness)", () => {
    const peanoOnly = PEANO_AXIOMS.filter((a) => a.id.startsWith("peano_"));
    const claims = peanoOnly.map((a) => a.claim.toLowerCase());
    expect(claims.some((c) => c.includes("multiplication"))).toBe(true);
  });

  it("peano axioms should include induction", () => {
    const ids = PEANO_AXIOMS.map((a) => a.id);
    expect(ids).toContain("peano_induction");
  });

  it("protocol axioms should include token conservation", () => {
    const ids = PROTOCOL_AXIOMS.map((a) => a.id);
    expect(ids).toContain("proto_1");
    const tokenAxiom = PROTOCOL_AXIOMS.find((a) => a.id === "proto_1")!;
    expect(tokenAxiom.claim.toLowerCase()).toContain("conservation");
  });
});

// ============ Axiom Inheritance ============

describe("getZoneAxioms", () => {
  it("propositional zone should have propositional + protocol axioms", () => {
    const axioms = getZoneAxioms("propositional");
    expect(axioms.length).toBeGreaterThanOrEqual(PROPOSITIONAL_AXIOMS.length);
    // Should include propositional axioms
    const ids = axioms.map((a) => a.id);
    expect(ids).toContain("prop_1");
    // Should include propositional-zone protocol axioms
    expect(ids).toContain("proto_1");
  });

  it("presburger zone should inherit propositional axioms", () => {
    const axioms = getZoneAxioms("presburger");
    const ids = axioms.map((a) => a.id);
    // Should include propositional
    expect(ids).toContain("prop_1");
    // Should include presburger
    expect(ids).toContain("pres_1");
  });

  it("peano zone should inherit propositional and presburger axioms", () => {
    const axioms = getZoneAxioms("peano");
    const ids = axioms.map((a) => a.id);
    expect(ids).toContain("prop_1");
    expect(ids).toContain("pres_1");
    expect(ids).toContain("peano_mult_1");
  });

  it("empirical zone should have propositional but not arithmetic axioms", () => {
    const axioms = getZoneAxioms("empirical");
    const ids = axioms.map((a) => a.id);
    expect(ids).toContain("prop_1");
    expect(ids).not.toContain("pres_1");
    expect(ids).not.toContain("peano_mult_1");
  });

  it("higher zones should have more axioms than lower zones", () => {
    const propAxioms = getZoneAxioms("propositional");
    const presAxioms = getZoneAxioms("presburger");
    const peanoAxioms = getZoneAxioms("peano");
    expect(presAxioms.length).toBeGreaterThan(propAxioms.length);
    expect(peanoAxioms.length).toBeGreaterThan(presAxioms.length);
  });
});

// ============ Zone Classification ============

describe("classifyClaimZone", () => {
  it("should classify set theory claims", () => {
    expect(classifyClaimZone("x ∈ A")).toBe("set_theory");
    expect(classifyClaimZone("A is a subset of B")).toBe("set_theory");
    expect(classifyClaimZone("the power set of S")).toBe("set_theory");
  });

  it("should classify multiplication claims as peano", () => {
    expect(classifyClaimZone("3 × 5 = 15")).toBe("peano");
    expect(classifyClaimZone("the product of a and b")).toBe("peano");
    expect(classifyClaimZone("n is divisible by 7")).toBe("peano");
  });

  it("should classify addition claims as presburger", () => {
    expect(classifyClaimZone("3 + 5 = 8")).toBe("presburger");
    expect(classifyClaimZone("the successor of n")).toBe("presburger");
    expect(classifyClaimZone("every natural number")).toBe("presburger");
  });

  it("should classify empirical claims", () => {
    expect(classifyClaimZone("data shows a correlation")).toBe("empirical");
    expect(classifyClaimZone("the experiment confirmed")).toBe("empirical");
    expect(classifyClaimZone("observed temperature was 25C")).toBe("empirical");
  });

  it("should default to propositional", () => {
    expect(classifyClaimZone("if A then B")).toBe("propositional");
    expect(classifyClaimZone("the sky is blue")).toBe("propositional");
  });

  it("should check formal notation too", () => {
    expect(classifyClaimZone("A formula", "n × m")).toBe("peano");
  });
});

// ============ Cross-Zone References ============

describe("canReference", () => {
  it("higher zones can reference lower zones", () => {
    expect(canReference("peano", "propositional")).toBe(true);
    expect(canReference("set_theory", "presburger")).toBe(true);
    expect(canReference("presburger", "propositional")).toBe(true);
  });

  it("lower zones cannot reference higher zones", () => {
    expect(canReference("propositional", "peano")).toBe(false);
    expect(canReference("presburger", "set_theory")).toBe(false);
  });

  it("same zone can reference itself", () => {
    expect(canReference("peano", "peano")).toBe(true);
    expect(canReference("propositional", "propositional")).toBe(true);
  });

  it("empirical can reference any zone", () => {
    expect(canReference("empirical", "propositional")).toBe(true);
    expect(canReference("empirical", "set_theory")).toBe(true);
  });
});

describe("intersectZones", () => {
  it("should return the higher zone", () => {
    expect(intersectZones("propositional", "peano")).toBe("peano");
    expect(intersectZones("presburger", "set_theory")).toBe("set_theory");
  });

  it("should return empirical if either is empirical", () => {
    expect(intersectZones("empirical", "peano")).toBe("empirical");
    expect(intersectZones("propositional", "empirical")).toBe("empirical");
  });

  it("should return same zone for identical inputs", () => {
    expect(intersectZones("peano", "peano")).toBe("peano");
  });
});

// ============ Meta-Axiom Validation ============

describe("META_AXIOMS", () => {
  it("should have exactly 4 meta-axioms", () => {
    expect(META_AXIOMS).toHaveLength(4);
  });

  it("all meta-axioms should have type 'meta'", () => {
    for (const axiom of META_AXIOMS) {
      expect(axiom.type).toBe("meta");
    }
  });

  it("all meta-axioms should have confidence 1.0", () => {
    for (const axiom of META_AXIOMS) {
      expect(axiom.confidence).toBe(1.0);
    }
  });

  it("all meta-axioms should have goedelReference", () => {
    for (const axiom of META_AXIOMS) {
      expect(axiom.goedelReference).toBeTruthy();
      expect(axiom.goedelReference.length).toBeGreaterThan(5);
    }
  });

  it("all meta-axioms should have philosophicalNote", () => {
    for (const axiom of META_AXIOMS) {
      expect(axiom.philosophicalNote).toBeTruthy();
      expect(axiom.philosophicalNote.length).toBeGreaterThan(10);
    }
  });

  it("all meta-axioms should have formalNotation", () => {
    for (const axiom of META_AXIOMS) {
      expect(axiom.formalNotation).toBeTruthy();
    }
  });

  it("meta_1 should reference Gödel-Rosser", () => {
    const meta1 = META_AXIOMS.find((a) => a.id === "meta_1")!;
    expect(meta1.goedelReference).toContain("Rosser");
    expect(meta1.zone).toBe("peano");
    expect(meta1.formalNotation).toContain("Provable_S");
  });

  it("meta_2 should reference independence (not undecidability)", () => {
    const meta2 = META_AXIOMS.find((a) => a.id === "meta_2")!;
    expect(meta2.claim.toLowerCase()).toContain("independence");
    expect(meta2.philosophicalNote).toContain("independent");
  });

  it("meta_3 should note implicit consistency assertion", () => {
    const meta3 = META_AXIOMS.find((a) => a.id === "meta_3")!;
    expect(meta3.philosophicalNote).toContain("contraposition");
    expect(meta3.goedelReference).toContain("Hilbert-Bernays");
    expect(meta3.formalNotation).toContain("Con(S)");
  });

  it("meta_4 should be in peano zone (meta-mathematical claim)", () => {
    const meta4 = META_AXIOMS.find((a) => a.id === "meta_4")!;
    expect(meta4.zone).toBe("peano");
    expect(meta4.goedelReference).toContain("Presburger");
    expect(meta4.goedelReference).toContain("conservative");
    expect(meta4.formalNotation).toContain("Conservative");
  });

  it("meta-axiom IDs should be unique", () => {
    const ids = META_AXIOMS.map((a) => a.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
});

// ============ Inference Strength ============

describe("INFERENCE_STRENGTH", () => {
  it("propositional rules should all be 1.0", () => {
    expect(INFERENCE_STRENGTH.modus_ponens).toBe(1.0);
    expect(INFERENCE_STRENGTH.modus_tollens).toBe(1.0);
    expect(INFERENCE_STRENGTH.conjunction_intro).toBe(1.0);
    expect(INFERENCE_STRENGTH.disjunction_intro).toBe(1.0);
    expect(INFERENCE_STRENGTH.double_negation).toBe(1.0);
  });

  it("first-order quantifier rules should be < 1.0", () => {
    expect(INFERENCE_STRENGTH.universal_instantiation).toBeLessThan(1.0);
    expect(INFERENCE_STRENGTH.existential_generalization).toBeLessThan(1.0);
  });

  it("induction should be < 1.0 (verification-intensive)", () => {
    expect(INFERENCE_STRENGTH.induction).toBeLessThan(1.0);
    expect(INFERENCE_STRENGTH.induction).toBeGreaterThan(0.9);
  });

  it("derived rules should be weakest", () => {
    expect(INFERENCE_STRENGTH.definition_expansion).toBeLessThanOrEqual(0.9);
    expect(INFERENCE_STRENGTH.axiom_application).toBeLessThanOrEqual(0.9);
  });

  it("all strengths should be > 0 and <= 1.0", () => {
    for (const [, strength] of Object.entries(INFERENCE_STRENGTH)) {
      expect(strength).toBeGreaterThan(0);
      expect(strength).toBeLessThanOrEqual(1.0);
    }
  });

  it("should have graduated values (not all identical)", () => {
    const values = new Set(Object.values(INFERENCE_STRENGTH));
    expect(values.size).toBeGreaterThan(1);
  });
});
