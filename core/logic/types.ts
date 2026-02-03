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
  | 'decidable'            // Can be proven or disproven within the system
  | 'undecidable'          // Gödelian limit - cannot be decided within system
  | 'undecidable_self_ref' // Self-referential undecidability
  | 'undecidable_timeout'  // Proof search exhausted without result
  | 'empirical'            // Outside formal system, evidence-based
  | 'unknown';             // Not yet analyzed

/**
 * Gödelian status of a statement
 */
export type GoedelianStatus =
  | 'benign'       // No problematic self-reference
  | 'goedelian'    // Contains Gödelian self-reference
  | 'paradoxical'  // Contains logical paradox (rejected)
  | 'unknown';     // Not yet analyzed

// ============ Logic Zones ============

/**
 * Logic zones (compartmentalized for consistency)
 *
 * Lower zones are complete and decidable.
 * Higher zones may be incomplete (Gödel applies).
 */
export type LogicZone =
  | 'propositional'  // Zone 0: Pure boolean logic (complete, decidable)
  | 'presburger'     // Zone 1: Addition-only arithmetic (decidable)
  | 'peano'          // Zone 2: Full arithmetic (incomplete - Gödel applies)
  | 'set_theory'     // Zone 3: Set theory (incomplete)
  | 'empirical';     // Zone 4: Outside formal system (evidence-based)

/**
 * Zone properties
 */
export interface ZoneProperties {
  zone: LogicZone;
  complete: boolean;      // Is the zone complete?
  decidable: boolean;     // Is every statement decidable?
  goedelApplies: boolean; // Does Gödel's theorem apply?
  consistencyProof: 'internal' | 'external' | 'assumed';
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
  | 'meta'          // About the system itself
  | 'logical'       // Rules of inference
  | 'arithmetic'    // Arithmetic axioms
  | 'protocol'      // Protocol-specific
  | 'definitional'  // Definitions
  | 'founding';     // Founding axioms (hidden truth)

/**
 * A logical axiom
 */
export interface LogicalAxiom {
  id: string;
  type: AxiomType;
  zone: LogicZone;
  claim: string;
  formalNotation?: string;  // Formal logic notation
  confidence: 1.0;          // Axioms always have confidence 1.0
  establishedAt: number;    // Block number
  governance?: {
    proposalId: string;
    voteResult: number;     // Approval percentage
  };
  goedelReference?: string; // Reference to Gödel if meta-axiom
}

// ============ Proofs ============

/**
 * Types of inference
 */
export type InferenceRule =
  // Propositional logic
  | 'modus_ponens'      // A, A→B ⊢ B
  | 'modus_tollens'     // A→B, ¬B ⊢ ¬A
  | 'hypothetical_syllogism' // A→B, B→C ⊢ A→C
  | 'disjunctive_syllogism'  // A∨B, ¬A ⊢ B
  | 'conjunction_intro' // A, B ⊢ A∧B
  | 'conjunction_elim'  // A∧B ⊢ A (or B)
  | 'disjunction_intro' // A ⊢ A∨B
  | 'double_negation'   // ¬¬A ⊢ A
  // First-order logic
  | 'universal_instantiation' // ∀x.P(x) ⊢ P(a)
  | 'existential_generalization' // P(a) ⊢ ∃x.P(x)
  // Arithmetic
  | 'induction'         // P(0), ∀n.(P(n)→P(n+1)) ⊢ ∀n.P(n)
  | 'substitution'      // a=b, P(a) ⊢ P(b)
  // Derived
  | 'definition_expansion' // Expand a definition
  | 'axiom_application';   // Direct axiom application

/**
 * Inference strength (how reliable is the inference)
 */
export const INFERENCE_STRENGTH: Record<InferenceRule, number> = {
  // Deductive (strongest)
  modus_ponens: 1.0,
  modus_tollens: 1.0,
  hypothetical_syllogism: 1.0,
  disjunctive_syllogism: 1.0,
  conjunction_intro: 1.0,
  conjunction_elim: 1.0,
  disjunction_intro: 1.0,
  double_negation: 1.0,
  universal_instantiation: 1.0,
  existential_generalization: 1.0,
  induction: 1.0,
  substitution: 1.0,
  definition_expansion: 1.0,
  axiom_application: 1.0,
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
    premises: number[];  // Step numbers of premises
    axiom?: string;      // Axiom ID if axiom application
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
  conclusion: number;  // Step number of conclusion
  verified: boolean;
  verifiedAt?: number;
  verifiedBy?: string[];  // Validator DIDs
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
  alternativeResolutions?: ('oracle' | 'consensus' | 'new_axiom')[];

  // Confidence
  confidence: number;
  confidenceSource: 'formal_proof' | 'disproof' | 'oracle' | 'consensus' | 'goedelian_limit';
}

// ============ Proof Search ============

/**
 * Proof search configuration
 */
export interface ProofSearchConfig {
  maxSteps: number;       // Maximum proof steps to try
  maxDepth: number;       // Maximum inference depth
  timeout: number;        // Milliseconds
  zones: LogicZone[];     // Zones to search in
  strategies: ProofStrategy[];
}

/**
 * Proof search strategies
 */
export type ProofStrategy =
  | 'forward_chaining'    // From axioms toward goal
  | 'backward_chaining'   // From goal toward axioms
  | 'resolution'          // Resolution-based
  | 'model_checking';     // For finite domains

/**
 * Proof search result
 */
export interface ProofSearchResult {
  found: boolean;
  proof?: FormalProof;
  searchStats: {
    stepsExplored: number;
    timeMs: number;
    exhausted: boolean;  // Did we exhaust the search space?
  };
  decidability: Decidability;
}

// ============ Meta-Axioms (Gödelian) ============

/**
 * The meta-axioms that acknowledge system limitations
 */
export interface MetaAxiom extends LogicalAxiom {
  type: 'meta';
  goedelReference: string;
  philosophicalNote: string;
}

/**
 * Predefined meta-axioms
 */
export const META_AXIOMS: MetaAxiom[] = [
  {
    id: 'meta_1',
    type: 'meta',
    zone: 'peano',
    claim: 'This formal system is incomplete: some true statements cannot be proven within it.',
    formalNotation: '∃φ. True(φ) ∧ ¬Provable(φ)',
    confidence: 1.0,
    establishedAt: 0,
    goedelReference: 'First Incompleteness Theorem (1931)',
    philosophicalNote: 'Incompleteness is not a bug but a mathematical reality.',
  },
  {
    id: 'meta_2',
    type: 'meta',
    zone: 'peano',
    claim: 'Undecidability is a valid and expected outcome for some claims.',
    formalNotation: '∃φ. ¬Provable(φ) ∧ ¬Provable(¬φ)',
    confidence: 1.0,
    establishedAt: 0,
    goedelReference: 'Corollary of First Incompleteness Theorem',
    philosophicalNote: 'Some questions have no answer within the system.',
  },
  {
    id: 'meta_3',
    type: 'meta',
    zone: 'peano',
    claim: 'The consistency of this system cannot be proven from within the system itself.',
    formalNotation: '¬Provable(Consistent(S))',
    confidence: 1.0,
    establishedAt: 0,
    goedelReference: 'Second Incompleteness Theorem (1931)',
    philosophicalNote: 'Consistency is assumed, not proven.',
  },
  {
    id: 'meta_4',
    type: 'meta',
    zone: 'propositional',
    claim: 'Lower logic zones maintain provable consistency independent of higher zones.',
    confidence: 1.0,
    establishedAt: 0,
    goedelReference: 'Compartmentalization principle',
    philosophicalNote: 'Incompleteness in Zone 2+ does not affect Zone 0-1.',
  },
];

// ============ Resolution Paths ============

/**
 * How to resolve undecidable claims
 */
export type ResolutionPath =
  | 'accept_undecidability'  // Leave at confidence 0.50
  | 'oracle'                 // External authority decides
  | 'consensus'              // Community vote
  | 'new_axiom';             // Add axiom through governance

/**
 * Resolution of an undecidable claim
 */
export interface UndecidableResolution {
  claimId: string;
  originalDecidability: 'undecidable' | 'undecidable_self_ref' | 'undecidable_timeout';
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
