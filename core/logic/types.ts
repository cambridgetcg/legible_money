/**
 * Logic Layer Types
 *
 * Type definitions for the formal logic system.
 */

// ============ Decidability ============

/**
 * Decidability classification of a claim
 */
export type Decidability =
  | "decidable" // Can be proven or disproven within the system
  | "undecidable" // Gödelian limit - cannot be decided within system
  | "undecidable_self_ref" // Self-referential undecidability
  | "undecidable_timeout" // Proof search exhausted without result
  | "empirical" // Outside formal system, evidence-based
  | "unknown"; // Not yet analyzed

/**
 * Gödelian status of a statement
 */
export type GoedelianStatus =
  | "benign" // No problematic self-reference
  | "goedelian" // Contains Gödelian self-reference
  | "paradoxical" // Contains logical paradox (rejected)
  | "unknown"; // Not yet analyzed

// ============ Logic Zones ============

/**
 * Logic zones (compartmentalized for consistency)
 *
 * Lower zones are complete and decidable.
 * Higher zones may be incomplete (Gödel applies).
 */
export type LogicZone =
  | "propositional" // Zone 0: Pure boolean logic (complete, decidable)
  | "presburger" // Zone 1: Addition-only arithmetic (decidable)
  | "peano" // Zone 2: Full arithmetic (incomplete - Gödel applies)
  | "set_theory" // Zone 3: Set theory (incomplete)
  | "empirical"; // Zone 4: Outside formal system (evidence-based)

/**
 * Zone properties
 */
export interface ZoneProperties {
  zone: LogicZone;
  complete: boolean; // Is the zone complete?
  decidable: boolean; // Is every statement decidable?
  goedelApplies: boolean; // Does Gödel's theorem apply?
  consistencyProof: "internal" | "external" | "assumed";
}

/**
 * Zone hierarchy (lower zones are stronger)
 */
export const ZONE_HIERARCHY: Record<LogicZone, number> = {
  propositional: 0,
  presburger: 1,
  peano: 2,
  set_theory: 3,
  empirical: 4,
};

// ============ Axioms ============

/**
 * Types of axioms
 */
export type AxiomType =
  | "meta" // About the system itself
  | "logical" // Rules of inference
  | "arithmetic" // Arithmetic axioms
  | "protocol" // Protocol-specific
  | "definitional" // Definitions
  | "founding"; // Founding axioms (hidden truth)

/**
 * A logical axiom
 */
export interface LogicalAxiom {
  id: string;
  type: AxiomType;
  zone: LogicZone;
  claim: string;
  formalNotation?: string; // Formal logic notation
  confidence: 1.0; // Axioms always have confidence 1.0
  establishedAt: number; // Block number
  governance?: {
    proposalId: string;
    voteResult: number; // Approval percentage
  };
  goedelReference?: string; // Reference to Gödel if meta-axiom
}

// ============ Proofs ============

/**
 * Types of inference
 */
export type InferenceRule =
  // Propositional logic
  | "modus_ponens" // A, A→B ⊢ B
  | "modus_tollens" // A→B, ¬B ⊢ ¬A
  | "hypothetical_syllogism" // A→B, B→C ⊢ A→C
  | "disjunctive_syllogism" // A∨B, ¬A ⊢ B
  | "conjunction_intro" // A, B ⊢ A∧B
  | "conjunction_elim" // A∧B ⊢ A (or B)
  | "disjunction_intro" // A ⊢ A∨B
  | "double_negation" // ¬¬A ⊢ A
  // First-order logic
  | "universal_instantiation" // ∀x.P(x) ⊢ P(a)
  | "existential_generalization" // P(a) ⊢ ∃x.P(x)
  // Arithmetic
  | "induction" // P(0), ∀n.(P(n)→P(n+1)) ⊢ ∀n.P(n)
  | "substitution" // a=b, P(a) ⊢ P(b)
  // Derived
  | "definition_expansion" // Expand a definition
  | "axiom_application"; // Direct axiom application

/**
 * Inference strength (how reliable is the inference)
 *
 * Deductive rules preserve truth with certainty (1.0).
 * First-order quantifier rules carry slight risk from
 * implicit domain assumptions (0.99).
 * Induction is sound but non-trivial to verify (0.95).
 * Derived rules inherit the weakest premise (0.90).
 */
export const INFERENCE_STRENGTH: Record<InferenceRule, number> = {
  // Propositional (truth-preserving, 1.0)
  modus_ponens: 1.0,
  modus_tollens: 1.0,
  hypothetical_syllogism: 1.0,
  disjunctive_syllogism: 1.0,
  conjunction_intro: 1.0,
  conjunction_elim: 1.0,
  disjunction_intro: 1.0,
  double_negation: 1.0,
  // First-order quantifier rules (domain-dependent, 0.99)
  universal_instantiation: 0.99,
  existential_generalization: 0.99,
  // Arithmetic (sound but verification-intensive, 0.95)
  induction: 0.95,
  substitution: 1.0,
  // Derived (depend on definition/axiom correctness, 0.90)
  definition_expansion: 0.9,
  axiom_application: 0.9,
};

/**
 * A single proof step
 */
export interface ProofStep {
  stepNumber: number;
  statement: string;
  formalNotation?: string;
  justification: {
    rule: InferenceRule;
    premises: number[]; // Step numbers of premises
    axiom?: string; // Axiom ID if axiom application
  };
  zone: LogicZone;
}

/**
 * A complete formal proof
 */
export interface FormalProof {
  id: string;
  claim: string;
  formalNotation?: string;
  zone: LogicZone;
  steps: ProofStep[];
  conclusion: number; // Step number of conclusion
  verified: boolean;
  verifiedAt?: number;
  verifiedBy?: string[]; // Validator DIDs
}

/**
 * Proof verification result
 */
export interface ProofVerification {
  valid: boolean;
  claim: string;
  zone: LogicZone;
  decidability: Decidability;
  goedelianStatus: GoedelianStatus;

  // If proven
  proof?: FormalProof;

  // If disproven
  counterexample?: string;

  // If undecidable
  undecidableReason?: string;
  alternativeResolutions?: ("oracle" | "consensus" | "new_axiom")[];

  // Confidence
  confidence: number;
  confidenceSource:
    | "formal_proof"
    | "disproof"
    | "oracle"
    | "consensus"
    | "goedelian_limit";
}

// ============ Proof Search ============

/**
 * Proof search configuration
 */
export interface ProofSearchConfig {
  maxSteps: number; // Maximum proof steps to try
  maxDepth: number; // Maximum inference depth
  timeout: number; // Milliseconds
  zones: LogicZone[]; // Zones to search in
  strategies: ProofStrategy[];
}

/**
 * Proof search strategies
 */
export type ProofStrategy =
  | "forward_chaining" // From axioms toward goal
  | "backward_chaining" // From goal toward axioms
  | "resolution" // Resolution-based
  | "model_checking"; // For finite domains

/**
 * Proof search result
 */
export interface ProofSearchResult {
  found: boolean;
  proof?: FormalProof;
  searchStats: {
    stepsExplored: number;
    timeMs: number;
    exhausted: boolean; // Did we exhaust the search space?
  };
  decidability: Decidability;
}

// ============ Meta-Axioms (Gödelian) ============

/**
 * The meta-axioms that acknowledge system limitations
 */
export interface MetaAxiom extends LogicalAxiom {
  type: "meta";
  goedelReference: string;
  philosophicalNote: string;
}

/**
 * Predefined meta-axioms
 *
 * These are meta-mathematical statements accepted by fiat (not proven
 * within the system). Their confidence of 1.0 reflects the designers'
 * certainty about the underlying theorems, proved in the meta-theory.
 *
 * Gödel's theorems require: (1) consistency, (2) recursive axiomatizability,
 * (3) sufficient arithmetic (Robinson Q or stronger). This system satisfies
 * (2) and (3) by construction; (1) is assumed (see meta_3).
 */
export const META_AXIOMS: MetaAxiom[] = [
  {
    id: "meta_1",
    type: "meta",
    zone: "peano",
    claim:
      "This formal system is incomplete: there exist sentences that are neither provable nor refutable within it.",
    formalNotation: "∃G. ¬Provable_S(G) ∧ ¬Provable_S(¬G)",
    confidence: 1.0,
    establishedAt: 0,
    goedelReference: "Gödel-Rosser Theorem (Gödel 1931, Rosser 1936)",
    philosophicalNote:
      "Incompleteness is not a bug but a mathematical certainty for any consistent system containing arithmetic.",
  },
  {
    id: "meta_2",
    type: "meta",
    zone: "peano",
    claim:
      "Independence from the axioms is a valid and expected outcome for some claims.",
    formalNotation: "∃φ. ¬⊢_S φ ∧ ¬⊢_S ¬φ",
    confidence: 1.0,
    establishedAt: 0,
    goedelReference: "Gödel-Rosser Theorem (Gödel 1931, Rosser 1936)",
    philosophicalNote:
      "Some sentences are independent of the axioms — neither provable nor refutable. This is distinct from algorithmic undecidability (Turing 1936).",
  },
  {
    id: "meta_3",
    type: "meta",
    zone: "peano",
    claim:
      "The consistency of this system cannot be proven from within the system itself.",
    formalNotation: "Con(S) → ¬Provable_S(Con(S))",
    confidence: 1.0,
    establishedAt: 0,
    goedelReference:
      "Second Incompleteness Theorem (Gödel 1931); requires Hilbert-Bernays-Löb derivability conditions",
    philosophicalNote:
      "Accepting this axiom implicitly asserts consistency by contraposition (if S were inconsistent, S would prove everything including Con(S), contradicting this axiom). Consistency is thus assumed, not proven — and this axiom makes that assumption explicit.",
  },
  {
    id: "meta_4",
    type: "meta",
    zone: "peano",
    claim:
      "Lower logic zones (propositional, Presburger) maintain provable completeness and decidability independent of higher zones.",
    formalNotation:
      "Complete(Z₀) ∧ Complete(Z₁) ∧ Conservative(Z₂ over Z₁) ∧ Conservative(Z₁ over Z₀)",
    confidence: 1.0,
    establishedAt: 0,
    goedelReference:
      "Completeness of propositional logic (Post 1921); decidability of Presburger arithmetic (Presburger 1929); conservative extensions (Shoenfield 1967)",
    philosophicalNote:
      "Incompleteness in Peano arithmetic and above does not infect propositional or Presburger zones. This is a meta-mathematical claim requiring arithmetic to state, not a propositional formula.",
  },
];

// ============ Resolution Paths ============

/**
 * How to resolve undecidable claims
 */
export type ResolutionPath =
  | "accept_undecidability" // Leave at confidence 0.50
  | "oracle" // External authority decides
  | "consensus" // Community vote
  | "new_axiom"; // Add axiom through governance

/**
 * Resolution of an undecidable claim
 */
export interface UndecidableResolution {
  claimId: string;
  originalDecidability:
    | "undecidable"
    | "undecidable_self_ref"
    | "undecidable_timeout";
  resolutionPath: ResolutionPath;

  // If oracle
  oracle?: {
    oracleId: string;
    verdict: boolean;
    timestamp: number;
  };

  // If consensus
  consensus?: {
    proposalId: string;
    votesFor: number;
    votesAgainst: number;
    result: boolean;
  };

  // If new axiom
  newAxiom?: {
    axiomId: string;
    makesDecidable: boolean;
  };

  finalConfidence: number;
  resolvedAt: number;
}
