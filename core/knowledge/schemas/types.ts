/**
 * Claim Schema Types
 *
 * Schemas define the structure, verification method, and timing requirements
 * for different types of claims. This enables:
 * - Type-safe claim submission
 * - Dynamic verification timing based on complexity
 * - Automatic verification for deterministic claims
 * - Clear guidance for validators on HOW to verify
 */

import type { Reference } from "../types.js";

// ============ Schema Timing ============

/**
 * Timing configuration for a schema
 *
 * Different claim types need different amounts of time:
 * - Hash verification: instant (1-2 blocks)
 * - Oracle queries: need response time (3-4 blocks)
 * - Document verification: need fetch/parse time (6-8 blocks)
 */
export interface SchemaTiming {
  /** Blocks for commit phase - validators analyze and commit */
  commitPhaseBlocks: number;

  /** Blocks for reveal phase - validators reveal verdicts */
  revealPhaseBlocks: number;

  /** Total round blocks (commit + reveal + buffer) */
  totalRoundBlocks: number;

  /** Minimum time for validators to analyze (milliseconds) */
  minAnalysisTimeMs?: number;
}

// ============ Verification Methods ============

/**
 * How a claim should be verified
 */
export type VerificationMethod =
  | "automatic" // Deterministic - computed by protocol
  | "oracle" // Query external data source
  | "reference" // Check against referenced documents
  | "derivation" // Logical derivation from existing facts
  | "consensus"; // Human-like judgment, validator consensus

/**
 * Result of automatic verification
 */
export interface AutoVerificationResult {
  /** Whether the claim is valid */
  valid: boolean;

  /** Confidence level (0-1) */
  confidence: number;

  /** Explanation of result */
  reason?: string;

  /** Additional metadata */
  metadata?: Record<string, unknown>;
}

/**
 * Automatic verifier function type
 */
export type AutoVerifier = (
  claimData: unknown,
  references: Reference[],
) => Promise<AutoVerificationResult>;

// ============ Schema Definition ============

/**
 * JSON Schema subset for claim data validation
 * (Simplified from full JSON Schema 7)
 */
export interface ClaimDataSchema {
  type: "object" | "array" | "string" | "number" | "boolean";
  properties?: Record<string, ClaimDataSchemaProperty>;
  required?: string[];
  items?: ClaimDataSchemaProperty;
  additionalProperties?: boolean;
}

export interface ClaimDataSchemaProperty {
  type: "string" | "number" | "boolean" | "array" | "object";
  description?: string;
  enum?: (string | number)[];
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  items?: ClaimDataSchemaProperty;
  properties?: Record<string, ClaimDataSchemaProperty>;
  required?: string[];
  additionalProperties?: boolean;
}

/**
 * Claim Schema
 *
 * Defines everything needed to submit, validate, and verify a claim type.
 */
export interface ClaimSchema {
  /** Unique schema identifier (e.g., 'hash_verification_v1') */
  id: string;

  /** Schema version for upgrades */
  version: number;

  /** Human-readable name */
  name: string;

  /** Description of what this schema verifies */
  description: string;

  /** Knowledge domain this schema applies to */
  domain: string;

  /** Timing requirements */
  timing: SchemaTiming;

  /** How claims of this type are verified */
  verificationMethod: VerificationMethod;

  /** JSON Schema for validating claim data */
  dataSchema: ClaimDataSchema;

  /** Required reference types for this claim */
  requiredReferenceTypes?: string[];

  /** Minimum number of references required */
  minReferences?: number;

  /** Automatic verifier (if verificationMethod is 'automatic') */
  verifier?: AutoVerifier;

  /** Whether this schema is active */
  active: boolean;

  /** Block number when schema was registered */
  registeredAt?: number;
}

// ============ Schema-Based Claim ============

/**
 * A claim submission using a schema
 */
export interface SchemaClaimData {
  /** Schema identifier */
  schemaId: string;

  /** Structured claim data matching the schema */
  data: Record<string, unknown>;
}

/**
 * Validation result for schema claim
 */
export interface SchemaValidationResult {
  /** Whether validation passed */
  valid: boolean;

  /** Validation errors if any */
  errors: SchemaValidationError[];

  /** The schema used for validation */
  schema?: ClaimSchema;
}

/**
 * Individual validation error
 */
export interface SchemaValidationError {
  /** Path to the invalid field (e.g., 'data.price') */
  path: string;

  /** Error message */
  message: string;

  /** Expected value/type */
  expected?: string;

  /** Actual value/type */
  actual?: string;
}

// ============ Schema Categories ============

/**
 * Schema categories for organization
 */
export type SchemaCategory =
  | "cryptographic" // Hashes, signatures, proofs
  | "oracle" // External data feeds
  | "chain_state" // Blockchain state queries
  | "computation" // Deterministic computations
  | "derivation" // Logical derivations
  | "document" // Document attestations
  | "attestation" // Human attestations
  | "research"; // Scientific/research claims

/**
 * Schema category metadata
 */
export interface SchemaCategoryInfo {
  category: SchemaCategory;
  name: string;
  description: string;
  defaultTiming: SchemaTiming;
  defaultMethod: VerificationMethod;
}

/**
 * Category definitions with default timing
 */
export const SCHEMA_CATEGORIES: Record<SchemaCategory, SchemaCategoryInfo> = {
  cryptographic: {
    category: "cryptographic",
    name: "Cryptographic Verification",
    description: "Hash verification, signature checks, ZK proofs",
    defaultTiming: {
      commitPhaseBlocks: 1,
      revealPhaseBlocks: 1,
      totalRoundBlocks: 3,
    },
    defaultMethod: "automatic",
  },
  oracle: {
    category: "oracle",
    name: "Oracle Data",
    description: "External data feeds like prices, weather, events",
    defaultTiming: {
      commitPhaseBlocks: 3,
      revealPhaseBlocks: 2,
      totalRoundBlocks: 6,
    },
    defaultMethod: "oracle",
  },
  chain_state: {
    category: "chain_state",
    name: "Blockchain State",
    description: "On-chain state from this or other blockchains",
    defaultTiming: {
      commitPhaseBlocks: 2,
      revealPhaseBlocks: 2,
      totalRoundBlocks: 5,
    },
    defaultMethod: "automatic",
  },
  computation: {
    category: "computation",
    name: "Deterministic Computation",
    description: "Reproducible computation results",
    defaultTiming: {
      commitPhaseBlocks: 2,
      revealPhaseBlocks: 2,
      totalRoundBlocks: 5,
    },
    defaultMethod: "automatic",
  },
  derivation: {
    category: "derivation",
    name: "Logical Derivation",
    description: "Claims derived from existing facts via logic",
    defaultTiming: {
      commitPhaseBlocks: 4,
      revealPhaseBlocks: 3,
      totalRoundBlocks: 8,
    },
    defaultMethod: "derivation",
  },
  document: {
    category: "document",
    name: "Document Attestation",
    description: "Claims about document contents",
    defaultTiming: {
      commitPhaseBlocks: 6,
      revealPhaseBlocks: 4,
      totalRoundBlocks: 12,
    },
    defaultMethod: "reference",
  },
  attestation: {
    category: "attestation",
    name: "Human Attestation",
    description: "Claims requiring human judgment",
    defaultTiming: {
      commitPhaseBlocks: 8,
      revealPhaseBlocks: 6,
      totalRoundBlocks: 16,
    },
    defaultMethod: "consensus",
  },
  research: {
    category: "research",
    name: "Research Claims",
    description: "Scientific and research findings",
    defaultTiming: {
      commitPhaseBlocks: 10,
      revealPhaseBlocks: 8,
      totalRoundBlocks: 20,
    },
    defaultMethod: "consensus",
  },
};
