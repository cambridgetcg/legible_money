/**
 * Claim Schemas
 *
 * Schema system for structured, verifiable claims.
 *
 * Schemas enable:
 * - Type-safe claim submission with validation
 * - Dynamic verification timing based on complexity
 * - Automatic verification for deterministic claims
 * - Clear guidance for validators on HOW to verify
 */

// Types
export * from "./types.js";

// Registry
export {
  SchemaRegistry,
  getSchemaRegistry,
  resetSchemaRegistry,
} from "./registry.js";

// Paper Decomposition Helpers
export {
  // Types
  type PaperClaimType,
  type ExtractedClaim,
  type PaperDecomposition,
  // Constants
  CLAIM_TYPE_MAX_CONFIDENCE,
  CLAIM_TYPE_SCHEMA,
  // Functions
  createPaperDecomposition,
  addMethodologyClaim,
  addMeasurementClaim,
  addStatisticalClaim,
  addCausalClaim,
  addInterpretationClaim,
  computeSubmissionOrder,
  generateClaimSubmissions,
} from "./paper_decomposition.js";

// Built-in schemas
export {
  BUILTIN_SCHEMAS,
  registerBuiltinSchemas,
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
} from "./builtin/index.js";

// Source Document Schemas
export {
  // Types
  type SourceCategory,
  type SourceReliabilityTier,
  RELIABILITY_MAX_CONFIDENCE,
  type SourceMetadata,
  type SourceAuthor,
  type SourceVerification,
  type ClaimExtractionPattern,
  EXTRACTION_PATTERNS,
  type TrustModifiers,
  calculateAdjustedConfidence,
  // All source schemas combined
  ALL_SOURCE_SCHEMAS,
  SOURCE_SCHEMAS_BY_CATEGORY,
  getSchemasByCategory,
  // Patent
  PATENT_DOCUMENT_SCHEMA,
  PATENT_CLAIM_SCHEMA,
  PRIOR_ART_SCHEMA,
  PATENT_SCHEMAS,
  // Media
  NEWS_ARTICLE_SCHEMA,
  PRESS_RELEASE_SCHEMA,
  QUOTE_ATTRIBUTION_SCHEMA,
  SOCIAL_MEDIA_POST_SCHEMA,
  MEDIA_SCHEMAS,
  // Data
  DATASET_SCHEMA,
  RAW_EXPERIMENTAL_DATA_SCHEMA,
  MEASUREMENT_SCHEMA,
  STATISTICAL_SUMMARY_SCHEMA,
  DATA_SCHEMAS,
  // Regulatory
  GOVERNMENT_DOCUMENT_SCHEMA,
  REGULATORY_APPROVAL_SCHEMA,
  ENFORCEMENT_ACTION_SCHEMA,
  REGULATORY_SCHEMAS,
  // Legal
  COURT_CASE_SCHEMA,
  LEGISLATION_SCHEMA,
  CONTRACT_SCHEMA,
  LEGAL_OPINION_SCHEMA,
  LEGAL_SCHEMAS,
  // Clinical
  CLINICAL_TRIAL_SCHEMA,
  CLINICAL_TRIAL_RESULT_SCHEMA,
  ADVERSE_EVENT_SCHEMA,
  CLINICAL_SCHEMAS,
  // Financial
  FINANCIAL_REPORT_SCHEMA,
  SEC_FILING_SCHEMA,
  INSIDER_TRANSACTION_SCHEMA,
  FINANCIAL_METRIC_SCHEMA,
  ANALYST_RATING_SCHEMA,
  FINANCIAL_SCHEMAS,
  // Technical
  TECHNICAL_STANDARD_SCHEMA,
  RFC_SCHEMA,
  API_DOCUMENTATION_SCHEMA,
  SOFTWARE_RELEASE_SCHEMA,
  BENCHMARK_RESULT_SCHEMA,
  TECHNICAL_SCHEMAS,
} from "./sources/index.js";

// ============ Initialization ============

import { getSchemaRegistry } from "./registry.js";
import { registerBuiltinSchemas } from "./builtin/index.js";
import { ALL_SOURCE_SCHEMAS } from "./sources/index.js";
import type { ClaimSchema } from "./types.js";

/**
 * Register source schemas with a registry
 */
function registerSourceSchemas(registry: {
  register: (schema: ClaimSchema) => { success: boolean; error?: string };
}): { registered: number; failed: string[] } {
  const failed: string[] = [];
  let registered = 0;

  for (const schema of ALL_SOURCE_SCHEMAS) {
    const result = registry.register(schema);
    if (result.success) {
      registered++;
    } else {
      failed.push(`${schema.id}: ${result.error}`);
    }
  }

  return { registered, failed };
}

/**
 * Initialize the schema system with all schemas
 *
 * Call this at protocol startup to register all standard schemas.
 * Registers:
 * - 12 built-in schemas (crypto, oracle, logic, research)
 * - 27 source document schemas (patent, media, legal, clinical, financial, technical)
 */
export function initializeSchemas(): { registered: number; failed: string[] } {
  const registry = getSchemaRegistry();

  // Register built-in schemas
  const builtinResult = registerBuiltinSchemas(registry);

  // Register source schemas
  const sourceResult = registerSourceSchemas(registry);

  return {
    registered: builtinResult.registered + sourceResult.registered,
    failed: [...builtinResult.failed, ...sourceResult.failed],
  };
}
