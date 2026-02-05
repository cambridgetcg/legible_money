/**
 * Source Document Schemas
 *
 * Central export for all source document schemas.
 *
 * Source schemas define the structure of external documents from which
 * claims can be extracted and verified. Different source types have
 * different reliability tiers and verification requirements.
 *
 * Source Categories:
 * - Academic: Research papers, preprints, peer-reviewed studies
 * - Patent: Patents, IP filings, prior art
 * - Regulatory: Government documents, approvals, enforcement
 * - Legal: Court cases, legislation, contracts
 * - Clinical: Clinical trials, adverse events, medical data
 * - Financial: SEC filings, financial reports, analyst ratings
 * - Technical: Standards, RFCs, API docs, software releases
 * - Media: News articles, press releases, social media
 * - Data: Datasets, measurements, experimental data
 */

// ============ Source Types ============

export {
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
} from "./types.js";

// ============ Patent Schemas ============

export {
  PATENT_DOCUMENT_SCHEMA,
  PATENT_CLAIM_SCHEMA,
  PRIOR_ART_SCHEMA,
  PATENT_SCHEMAS,
} from "./patent.js";

// ============ Media Schemas ============

export {
  NEWS_ARTICLE_SCHEMA,
  PRESS_RELEASE_SCHEMA,
  QUOTE_ATTRIBUTION_SCHEMA,
  SOCIAL_MEDIA_POST_SCHEMA,
  MEDIA_SCHEMAS,
} from "./media.js";

// ============ Data Schemas ============

export {
  DATASET_SCHEMA,
  RAW_EXPERIMENTAL_DATA_SCHEMA,
  MEASUREMENT_SCHEMA,
  STATISTICAL_SUMMARY_SCHEMA,
  DATA_SCHEMAS,
} from "./data.js";

// ============ Regulatory Schemas ============

export {
  GOVERNMENT_DOCUMENT_SCHEMA,
  REGULATORY_APPROVAL_SCHEMA,
  ENFORCEMENT_ACTION_SCHEMA,
  REGULATORY_SCHEMAS,
} from "./regulatory.js";

// ============ Legal Schemas ============

export {
  COURT_CASE_SCHEMA,
  LEGISLATION_SCHEMA,
  CONTRACT_SCHEMA,
  LEGAL_OPINION_SCHEMA,
  LEGAL_SCHEMAS,
} from "./legal.js";

// ============ Clinical Schemas ============

export {
  CLINICAL_TRIAL_SCHEMA,
  CLINICAL_TRIAL_RESULT_SCHEMA,
  ADVERSE_EVENT_SCHEMA,
  CLINICAL_SCHEMAS,
} from "./clinical.js";

// ============ Financial Schemas ============

export {
  FINANCIAL_REPORT_SCHEMA,
  SEC_FILING_SCHEMA,
  INSIDER_TRANSACTION_SCHEMA,
  FINANCIAL_METRIC_SCHEMA,
  ANALYST_RATING_SCHEMA,
  FINANCIAL_SCHEMAS,
} from "./financial.js";

// ============ Technical Schemas ============

export {
  TECHNICAL_STANDARD_SCHEMA,
  RFC_SCHEMA,
  API_DOCUMENTATION_SCHEMA,
  SOFTWARE_RELEASE_SCHEMA,
  BENCHMARK_RESULT_SCHEMA,
  TECHNICAL_SCHEMAS,
} from "./technical.js";

// ============ All Source Schemas ============

import { PATENT_SCHEMAS } from "./patent.js";
import { MEDIA_SCHEMAS } from "./media.js";
import { DATA_SCHEMAS } from "./data.js";
import { REGULATORY_SCHEMAS } from "./regulatory.js";
import { LEGAL_SCHEMAS } from "./legal.js";
import { CLINICAL_SCHEMAS } from "./clinical.js";
import { FINANCIAL_SCHEMAS } from "./financial.js";
import { TECHNICAL_SCHEMAS } from "./technical.js";

/**
 * All source document schemas combined
 *
 * Total: 27 source schemas across 8 categories
 */
export const ALL_SOURCE_SCHEMAS = [
  ...PATENT_SCHEMAS, // 3 schemas
  ...MEDIA_SCHEMAS, // 4 schemas
  ...DATA_SCHEMAS, // 4 schemas
  ...REGULATORY_SCHEMAS, // 3 schemas
  ...LEGAL_SCHEMAS, // 4 schemas
  ...CLINICAL_SCHEMAS, // 3 schemas
  ...FINANCIAL_SCHEMAS, // 5 schemas
  ...TECHNICAL_SCHEMAS, // 5 schemas
];

/**
 * Source schemas grouped by category
 */
export const SOURCE_SCHEMAS_BY_CATEGORY = {
  patent: PATENT_SCHEMAS,
  media: MEDIA_SCHEMAS,
  data: DATA_SCHEMAS,
  regulatory: REGULATORY_SCHEMAS,
  legal: LEGAL_SCHEMAS,
  clinical: CLINICAL_SCHEMAS,
  financial: FINANCIAL_SCHEMAS,
  technical: TECHNICAL_SCHEMAS,
} as const;

/**
 * Get schemas by source category
 */
export function getSchemasByCategory(
  category: keyof typeof SOURCE_SCHEMAS_BY_CATEGORY,
) {
  return SOURCE_SCHEMAS_BY_CATEGORY[category];
}
