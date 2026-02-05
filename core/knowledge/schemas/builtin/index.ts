/**
 * Built-in Claim Schemas
 *
 * Standard schemas for common claim types.
 * These are registered at protocol initialization.
 */

import { sha512_256Hex } from "../../../crypto/hashing.js";
import type { ClaimSchema } from "../types.js";

// ============ Cryptographic Schemas ============

/**
 * Hash Verification Schema
 *
 * Verifies that hash(algorithm, input) = expectedHash
 * Fully automatic - deterministic computation
 */
export const HASH_VERIFICATION_SCHEMA: ClaimSchema = {
  id: "hash_verification_v1",
  version: 1,
  name: "Hash Verification",
  description: "Verify that a hash computation produces the expected result",
  domain: "cryptography",

  timing: {
    commitPhaseBlocks: 1,
    revealPhaseBlocks: 1,
    totalRoundBlocks: 3,
    minAnalysisTimeMs: 100,
  },

  verificationMethod: "automatic",

  dataSchema: {
    type: "object",
    properties: {
      algorithm: {
        type: "string",
        enum: ["sha256", "sha512", "sha512_256", "blake3"],
        description: "Hash algorithm to use",
      },
      input: {
        type: "string",
        description: "Input to hash (hex-encoded for binary)",
      },
      inputEncoding: {
        type: "string",
        enum: ["utf8", "hex"],
        description: "How to interpret the input",
      },
      expectedHash: {
        type: "string",
        pattern: "^[a-f0-9]+$",
        description: "Expected hash output (hex)",
      },
    },
    required: ["algorithm", "input", "expectedHash"],
    additionalProperties: false,
  },

  verifier: async (claimData) => {
    const data = claimData as {
      algorithm: string;
      input: string;
      inputEncoding?: string;
      expectedHash: string;
    };

    // Only sha512_256 is currently implemented
    if (data.algorithm !== "sha512_256") {
      return {
        valid: false,
        confidence: 0,
        reason: `Algorithm not supported: ${data.algorithm}. Only sha512_256 is currently available.`,
      };
    }

    const actualHash = sha512_256Hex(data.input);
    const matches =
      actualHash.toLowerCase() === data.expectedHash.toLowerCase();

    return {
      valid: matches,
      confidence: matches ? 1.0 : 0,
      reason: matches
        ? "Hash matches"
        : `Hash mismatch: expected ${data.expectedHash}, got ${actualHash}`,
      metadata: { actualHash },
    };
  },

  active: true,
};

/**
 * Signature Verification Schema
 *
 * Verifies that a signature is valid for a message and public key
 */
export const SIGNATURE_VERIFICATION_SCHEMA: ClaimSchema = {
  id: "signature_verification_v1",
  version: 1,
  name: "Signature Verification",
  description: "Verify that a cryptographic signature is valid",
  domain: "cryptography",

  timing: {
    commitPhaseBlocks: 1,
    revealPhaseBlocks: 1,
    totalRoundBlocks: 3,
    minAnalysisTimeMs: 100,
  },

  verificationMethod: "automatic",

  dataSchema: {
    type: "object",
    properties: {
      algorithm: {
        type: "string",
        enum: ["ed25519"],
        description: "Signature algorithm",
      },
      message: {
        type: "string",
        description: "Message that was signed",
      },
      signature: {
        type: "string",
        pattern: "^[a-f0-9]+$",
        description: "Signature (hex)",
      },
      publicKey: {
        type: "string",
        pattern: "^[a-f0-9]+$",
        description: "Public key (hex)",
      },
    },
    required: ["algorithm", "message", "signature", "publicKey"],
    additionalProperties: false,
  },

  // Verifier would use crypto/signatures.ts verify function
  // Left as undefined - validators must verify manually for now

  active: true,
};

// ============ Oracle Schemas ============

/**
 * Oracle Price Schema
 *
 * Claims about asset prices from oracle feeds
 */
export const ORACLE_PRICE_SCHEMA: ClaimSchema = {
  id: "oracle_price_v1",
  version: 1,
  name: "Oracle Price",
  description: "Verify asset price from oracle data feeds",
  domain: "oracle.prices",

  timing: {
    commitPhaseBlocks: 3,
    revealPhaseBlocks: 2,
    totalRoundBlocks: 6,
    minAnalysisTimeMs: 2000, // Need time to query oracles
  },

  verificationMethod: "oracle",

  dataSchema: {
    type: "object",
    properties: {
      asset: {
        type: "string",
        description: "Asset symbol (e.g., BTC, ETH)",
      },
      currency: {
        type: "string",
        description: "Quote currency (e.g., USD, EUR)",
      },
      price: {
        type: "number",
        minimum: 0,
        description: "Claimed price",
      },
      tolerance: {
        type: "number",
        minimum: 0,
        maximum: 0.1,
        description: "Acceptable deviation (e.g., 0.01 for 1%)",
      },
      timestamp: {
        type: "number",
        description: "Unix timestamp for the price",
      },
      oracleIds: {
        type: "array",
        items: { type: "string" },
        description: "Oracle sources to check",
      },
    },
    required: ["asset", "currency", "price", "timestamp", "oracleIds"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["oracle"],
  minReferences: 1,

  active: true,
};

// ============ Chain State Schemas ============

/**
 * Blockchain State Schema
 *
 * Claims about state on this or other blockchains
 */
export const CHAIN_STATE_SCHEMA: ClaimSchema = {
  id: "chain_state_v1",
  version: 1,
  name: "Blockchain State",
  description: "Verify state from a blockchain",
  domain: "chain.state",

  timing: {
    commitPhaseBlocks: 2,
    revealPhaseBlocks: 2,
    totalRoundBlocks: 5,
    minAnalysisTimeMs: 1000,
  },

  verificationMethod: "automatic",

  dataSchema: {
    type: "object",
    properties: {
      chain: {
        type: "string",
        description: "Blockchain identifier (e.g., ethereum, bitcoin, lgm)",
      },
      queryType: {
        type: "string",
        enum: ["balance", "block_hash", "transaction", "contract_state"],
        description: "Type of state query",
      },
      address: {
        type: "string",
        description: "Address to query (if applicable)",
      },
      blockNumber: {
        type: "number",
        description: "Block number for the query",
      },
      expectedValue: {
        type: "string",
        description: "Expected state value",
      },
    },
    required: ["chain", "queryType", "expectedValue"],
    additionalProperties: false,
  },

  active: true,
};

// ============ Derivation Schemas ============

/**
 * Logical Derivation Schema
 *
 * Claims derived from existing facts via logical rules
 */
export const LOGICAL_DERIVATION_SCHEMA: ClaimSchema = {
  id: "logical_derivation_v1",
  version: 1,
  name: "Logical Derivation",
  description: "Verify a logical derivation from existing facts",
  domain: "logic",

  timing: {
    commitPhaseBlocks: 4,
    revealPhaseBlocks: 3,
    totalRoundBlocks: 8,
    minAnalysisTimeMs: 3000, // Need time to check premises
  },

  verificationMethod: "derivation",

  dataSchema: {
    type: "object",
    properties: {
      premises: {
        type: "array",
        items: { type: "string" },
        description: "Fact IDs used as premises",
      },
      rule: {
        type: "string",
        enum: [
          "modus_ponens", // If P→Q and P, then Q
          "modus_tollens", // If P→Q and ¬Q, then ¬P
          "conjunction", // If P and Q, then P∧Q
          "disjunction", // If P, then P∨Q
          "hypothetical_syllogism", // If P→Q and Q→R, then P→R
          "universal_instantiation", // If ∀x P(x), then P(a)
        ],
        description: "Logical inference rule applied",
      },
      conclusion: {
        type: "string",
        description: "The derived conclusion",
      },
      formalRepresentation: {
        type: "string",
        description: "Formal logic representation of the derivation",
      },
    },
    required: ["premises", "rule", "conclusion"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["on_chain"],
  minReferences: 1, // At least one premise must reference existing fact

  active: true,
};

// ============ Document Schemas ============

/**
 * Document Attestation Schema
 *
 * Claims about contents of documents
 */
export const DOCUMENT_ATTESTATION_SCHEMA: ClaimSchema = {
  id: "document_attestation_v1",
  version: 1,
  name: "Document Attestation",
  description: "Verify a claim about document contents",
  domain: "documents",

  timing: {
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 4,
    totalRoundBlocks: 12,
    minAnalysisTimeMs: 5000, // Need time to fetch and parse
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      documentHash: {
        type: "string",
        pattern: "^[a-f0-9]+$",
        description: "Hash of the document content",
      },
      documentUrl: {
        type: "string",
        description: "URL where document can be retrieved",
      },
      archiveUrl: {
        type: "string",
        description: "Archive.org or similar permanent URL",
      },
      claim: {
        type: "string",
        description: "The claim about the document",
      },
      location: {
        type: "string",
        description: "Where in the document (page, section, etc.)",
      },
      quote: {
        type: "string",
        description: "Exact quote from document supporting the claim",
      },
    },
    required: ["documentHash", "claim"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["archived_web"],
  minReferences: 1,

  active: true,
};

// ============ Research Schemas ============

/**
 * Research Claim Schema
 *
 * Scientific/research findings with methodology requirements
 */
export const RESEARCH_CLAIM_SCHEMA: ClaimSchema = {
  id: "research_claim_v1",
  version: 1,
  name: "Research Claim",
  description: "Scientific or research finding with methodology",
  domain: "research",

  timing: {
    commitPhaseBlocks: 10,
    revealPhaseBlocks: 8,
    totalRoundBlocks: 20,
    minAnalysisTimeMs: 10000, // Complex analysis needed
  },

  verificationMethod: "consensus",

  dataSchema: {
    type: "object",
    properties: {
      field: {
        type: "string",
        description: "Research field (e.g., neuroscience, physics)",
      },
      claim: {
        type: "string",
        description: "The research finding",
      },
      methodology: {
        type: "string",
        description: "Research methodology used",
      },
      sampleSize: {
        type: "number",
        minimum: 0,
        description: "Sample size (if applicable)",
      },
      statisticalSignificance: {
        type: "number",
        minimum: 0,
        maximum: 1,
        description: "P-value or confidence level",
      },
      paperDoi: {
        type: "string",
        description: "DOI of the research paper",
      },
      replicationStatus: {
        type: "string",
        enum: ["not_attempted", "pending", "replicated", "failed_replication"],
        description: "Whether the result has been replicated",
      },
      peerReviewStatus: {
        type: "string",
        enum: ["preprint", "under_review", "peer_reviewed", "retracted"],
        description: "Peer review status",
      },
    },
    required: ["field", "claim", "methodology"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["peer_reviewed"],
  minReferences: 1,

  active: true,
};

/**
 * Experimental Result Schema
 *
 * Results from specific experiments that can be replicated
 */
export const EXPERIMENTAL_RESULT_SCHEMA: ClaimSchema = {
  id: "experimental_result_v1",
  version: 1,
  name: "Experimental Result",
  description: "A specific experimental result that can be replicated",
  domain: "research.experiments",

  timing: {
    commitPhaseBlocks: 8,
    revealPhaseBlocks: 6,
    totalRoundBlocks: 16,
    minAnalysisTimeMs: 8000,
  },

  verificationMethod: "consensus",

  dataSchema: {
    type: "object",
    properties: {
      experimentType: {
        type: "string",
        description: "Type of experiment",
      },
      hypothesis: {
        type: "string",
        description: "Hypothesis being tested",
      },
      result: {
        type: "string",
        description: "The experimental result",
      },
      conditions: {
        type: "object",
        description: "Experimental conditions",
        additionalProperties: true,
      },
      measurements: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            value: { type: "number" },
            unit: { type: "string" },
            uncertainty: { type: "number" },
          },
        },
        description: "Measured values",
      },
      dataHash: {
        type: "string",
        description: "Hash of raw experimental data",
      },
      protocolHash: {
        type: "string",
        description: "Hash of experimental protocol document",
      },
    },
    required: ["experimentType", "hypothesis", "result"],
    additionalProperties: false,
  },

  active: true,
};

// ============ Research Paper Schema ============

/**
 * Research Paper Schema
 *
 * A published research paper with extractable claims.
 * This is a "composite" schema - a paper contains multiple claims
 * that can be individually verified and tracked.
 *
 * The paper itself is verified for:
 * - Existence (DOI resolves)
 * - Peer review status
 * - Content hash (paper hasn't been modified)
 *
 * Individual claims within are extracted and submitted separately
 * using research_claim_v1 or experimental_result_v1 schemas.
 */
export const RESEARCH_PAPER_SCHEMA: ClaimSchema = {
  id: "research_paper_v1",
  version: 1,
  name: "Research Paper",
  description: "A published research paper with extractable claims",
  domain: "research.publications",

  timing: {
    commitPhaseBlocks: 12, // Need time to fetch and verify paper
    revealPhaseBlocks: 8,
    totalRoundBlocks: 24, // ~60 seconds
    minAnalysisTimeMs: 15000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Paper Identification ============
      doi: {
        type: "string",
        pattern: "^10\\..+/.+$",
        description:
          "Digital Object Identifier (e.g., 10.1038/s41586-024-00001-1)",
      },
      pmid: {
        type: "string",
        description: "PubMed ID (if applicable)",
      },
      arxivId: {
        type: "string",
        description: "arXiv identifier (e.g., 2401.00001)",
      },

      // ============ Paper Metadata ============
      title: {
        type: "string",
        minLength: 10,
        maxLength: 500,
        description: "Full paper title",
      },
      authors: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            orcid: { type: "string" },
            affiliation: { type: "string" },
          },
        },
        description: "Paper authors with optional ORCID",
      },
      journal: {
        type: "string",
        description: "Journal or venue name",
      },
      publicationDate: {
        type: "string",
        description: "Publication date (ISO 8601)",
      },
      volume: {
        type: "string",
        description: "Journal volume",
      },
      pages: {
        type: "string",
        description: "Page numbers or article number",
      },

      // ============ Content Verification ============
      abstractHash: {
        type: "string",
        pattern: "^[a-f0-9]{64}$",
        description: "SHA-256 hash of the abstract text",
      },
      fullTextHash: {
        type: "string",
        pattern: "^[a-f0-9]{64}$",
        description: "SHA-256 hash of the full text (PDF or text)",
      },
      supplementaryHash: {
        type: "string",
        description: "Hash of supplementary materials",
      },

      // ============ Review Status ============
      peerReviewStatus: {
        type: "string",
        enum: ["preprint", "under_review", "peer_reviewed", "retracted"],
        description: "Current peer review status",
      },
      retractionNotice: {
        type: "string",
        description: "Retraction notice DOI (if retracted)",
      },

      // ============ Research Domain ============
      field: {
        type: "string",
        description: "Primary research field (e.g., neuroscience, physics)",
      },
      subfield: {
        type: "string",
        description: "Specific subfield (e.g., memory, quantum computing)",
      },
      keywords: {
        type: "array",
        items: { type: "string" },
        description: "Paper keywords",
      },

      // ============ Extracted Claims ============
      /**
       * Claims extracted from this paper.
       * Each becomes a separate claim in the knowledge base,
       * linked back to this paper as a reference.
       */
      extractedClaims: {
        type: "array",
        items: {
          type: "object",
          properties: {
            claimId: {
              type: "string",
              description: "ID of the submitted claim (after submission)",
            },
            claimType: {
              type: "string",
              enum: [
                "methodology", // How the study was done
                "sample", // Sample characteristics
                "measurement", // Quantitative results
                "statistical", // Statistical findings
                "observation", // Qualitative observations
                "interpretation", // Authors interpretation
                "conclusion", // Main conclusions
                "limitation", // Stated limitations
              ],
              description: "Type of claim",
            },
            claimText: {
              type: "string",
              description: "The claim in natural language",
            },
            location: {
              type: "string",
              description: 'Where in paper (e.g., "Results, paragraph 3")',
            },
            schemaUsed: {
              type: "string",
              description:
                "Schema ID used for this claim (e.g., experimental_result_v1)",
            },
            confidence: {
              type: "number",
              minimum: 0,
              maximum: 1,
              description: "Proposed confidence for this claim",
            },
          },
        },
        description: "Claims extracted from this paper",
      },

      // ============ Replication Status ============
      replicationAttempts: {
        type: "array",
        items: {
          type: "object",
          properties: {
            doi: { type: "string" },
            outcome: {
              type: "string",
              enum: ["replicated", "partial", "failed", "inconclusive"],
            },
            notes: { type: "string" },
          },
        },
        description: "Known replication attempts",
      },

      // ============ Funding & Conflicts ============
      fundingSources: {
        type: "array",
        items: { type: "string" },
        description: "Funding sources as stated in paper",
      },
      conflictsOfInterest: {
        type: "string",
        description: "Declared conflicts of interest",
      },
      dataAvailability: {
        type: "string",
        enum: ["available", "on_request", "restricted", "not_available"],
        description: "Raw data availability",
      },
      dataRepositoryUrl: {
        type: "string",
        description: "URL to data repository (if available)",
      },
    },
    required: ["doi", "title", "peerReviewStatus", "field"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["peer_reviewed", "archived_web"],
  minReferences: 1,

  active: true,
};

/**
 * Methodology Claim Schema
 *
 * Claims about research methodology used in a study.
 * Lower uncertainty than results - methodology is stated fact.
 */
export const METHODOLOGY_CLAIM_SCHEMA: ClaimSchema = {
  id: "methodology_claim_v1",
  version: 1,
  name: "Methodology Claim",
  description: "A claim about research methodology used",
  domain: "research.methodology",

  timing: {
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 4,
    totalRoundBlocks: 12,
    minAnalysisTimeMs: 5000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      paperDoi: {
        type: "string",
        description: "DOI of the source paper",
      },
      methodologyType: {
        type: "string",
        enum: [
          "sample_selection",
          "experimental_design",
          "measurement_technique",
          "statistical_analysis",
          "data_collection",
          "control_conditions",
          "blinding",
          "randomization",
        ],
        description: "Type of methodology claim",
      },
      claim: {
        type: "string",
        description: "The methodology claim",
      },
      location: {
        type: "string",
        description: 'Location in paper (e.g., "Methods, section 2.1")',
      },
      quote: {
        type: "string",
        description: "Direct quote from paper supporting claim",
      },

      // Specific methodology details
      sampleSize: {
        type: "number",
        description: "Sample size (if sample_selection)",
      },
      sampleCharacteristics: {
        type: "object",
        description: "Sample characteristics (species, age, etc.)",
        additionalProperties: true,
      },
      technique: {
        type: "string",
        description: "Specific technique used",
      },
      equipment: {
        type: "string",
        description: "Equipment or software used",
      },
      controlConditions: {
        type: "array",
        items: { type: "string" },
        description: "Control conditions used",
      },
    },
    required: ["paperDoi", "methodologyType", "claim"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["peer_reviewed"],
  minReferences: 1,

  active: true,
};

/**
 * Statistical Claim Schema
 *
 * Claims about statistical results from research.
 * Includes effect sizes, p-values, confidence intervals.
 */
export const STATISTICAL_CLAIM_SCHEMA: ClaimSchema = {
  id: "statistical_claim_v1",
  version: 1,
  name: "Statistical Claim",
  description: "A statistical finding from research",
  domain: "research.statistics",

  timing: {
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 4,
    totalRoundBlocks: 12,
    minAnalysisTimeMs: 5000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      paperDoi: {
        type: "string",
        description: "DOI of the source paper",
      },
      statisticalTest: {
        type: "string",
        enum: [
          "t_test",
          "anova",
          "chi_square",
          "mann_whitney",
          "wilcoxon",
          "correlation",
          "regression",
          "mixed_model",
          "bayesian",
          "other",
        ],
        description: "Statistical test used",
      },
      testDetails: {
        type: "string",
        description: 'Specific test details (e.g., "two-tailed paired t-test")',
      },

      // Results
      pValue: {
        type: "number",
        minimum: 0,
        maximum: 1,
        description: "P-value",
      },
      effectSize: {
        type: "number",
        description: "Effect size (Cohen's d, r, etc.)",
      },
      effectSizeType: {
        type: "string",
        enum: [
          "cohens_d",
          "hedges_g",
          "r",
          "r_squared",
          "odds_ratio",
          "eta_squared",
          "other",
        ],
        description: "Type of effect size measure",
      },
      confidenceInterval: {
        type: "object",
        properties: {
          lower: { type: "number" },
          upper: { type: "number" },
          level: { type: "number" }, // e.g., 0.95 for 95% CI
        },
        description: "Confidence interval",
      },
      sampleSize: {
        type: "number",
        description: "Sample size for this analysis",
      },
      degreesOfFreedom: {
        type: "number",
        description: "Degrees of freedom",
      },
      testStatistic: {
        type: "number",
        description: "Test statistic value (t, F, chi-square, etc.)",
      },

      // Context
      comparison: {
        type: "string",
        description: 'What was compared (e.g., "treatment vs control")',
      },
      location: {
        type: "string",
        description: "Location in paper",
      },
    },
    required: ["paperDoi", "statisticalTest"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["peer_reviewed"],
  minReferences: 1,

  active: true,
};

/**
 * Causal Claim Schema
 *
 * Claims about causal relationships.
 * Highest uncertainty - requires strong evidence.
 */
export const CAUSAL_CLAIM_SCHEMA: ClaimSchema = {
  id: "causal_claim_v1",
  version: 1,
  name: "Causal Claim",
  description: "A claim about causal relationships",
  domain: "research.causation",

  timing: {
    commitPhaseBlocks: 12,
    revealPhaseBlocks: 10,
    totalRoundBlocks: 24,
    minAnalysisTimeMs: 15000,
  },

  verificationMethod: "consensus",

  dataSchema: {
    type: "object",
    properties: {
      cause: {
        type: "string",
        description: "The proposed cause",
      },
      effect: {
        type: "string",
        description: "The proposed effect",
      },
      relationship: {
        type: "string",
        enum: [
          "causes",
          "prevents",
          "increases",
          "decreases",
          "necessary_for",
          "sufficient_for",
          "modulates",
        ],
        description: "Type of causal relationship",
      },

      // Evidence strength
      evidenceType: {
        type: "string",
        enum: [
          "randomized_controlled_trial",
          "experimental_intervention",
          "natural_experiment",
          "longitudinal_observational",
          "cross_sectional",
          "case_control",
          "mechanistic",
          "meta_analysis",
        ],
        description: "Type of evidence supporting causation",
      },
      confoundsAddressed: {
        type: "array",
        items: { type: "string" },
        description: "Potential confounds that were addressed",
      },
      confoundsRemaining: {
        type: "array",
        items: { type: "string" },
        description: "Potential confounds not addressed",
      },

      // Supporting evidence
      supportingClaimIds: {
        type: "array",
        items: { type: "string" },
        description: "IDs of claims supporting this causal relationship",
      },
      paperDoi: {
        type: "string",
        description: "Primary paper DOI",
      },

      // Replication
      replicationCount: {
        type: "number",
        description: "Number of successful replications",
      },
      failedReplicationCount: {
        type: "number",
        description: "Number of failed replications",
      },

      // Mechanism
      proposedMechanism: {
        type: "string",
        description: "Proposed mechanistic explanation",
      },
      mechanismSupported: {
        type: "boolean",
        description: "Whether mechanism has empirical support",
      },
    },
    required: ["cause", "effect", "relationship", "evidenceType"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["peer_reviewed"],
  minReferences: 2, // Causal claims need more evidence

  active: true,
};

// ============ All Built-in Schemas ============

/**
 * All built-in schemas
 */
export const BUILTIN_SCHEMAS: ClaimSchema[] = [
  // Cryptographic
  HASH_VERIFICATION_SCHEMA,
  SIGNATURE_VERIFICATION_SCHEMA,

  // Oracle & Chain
  ORACLE_PRICE_SCHEMA,
  CHAIN_STATE_SCHEMA,

  // Logic
  LOGICAL_DERIVATION_SCHEMA,

  // Documents
  DOCUMENT_ATTESTATION_SCHEMA,

  // Research - General
  RESEARCH_CLAIM_SCHEMA,
  EXPERIMENTAL_RESULT_SCHEMA,

  // Research - Specialized
  RESEARCH_PAPER_SCHEMA,
  METHODOLOGY_CLAIM_SCHEMA,
  STATISTICAL_CLAIM_SCHEMA,
  CAUSAL_CLAIM_SCHEMA,
];

/**
 * Register all built-in schemas with a registry
 */
export function registerBuiltinSchemas(registry: {
  register: (schema: ClaimSchema) => { success: boolean; error?: string };
}): { registered: number; failed: string[] } {
  const failed: string[] = [];
  let registered = 0;

  for (const schema of BUILTIN_SCHEMAS) {
    const result = registry.register(schema);
    if (result.success) {
      registered++;
    } else {
      failed.push(`${schema.id}: ${result.error}`);
    }
  }

  return { registered, failed };
}
