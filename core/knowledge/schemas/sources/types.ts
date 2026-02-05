/**
 * Source Document Types
 *
 * Defines the taxonomy of source documents from which claims can be extracted.
 * Different source types have different:
 * - Verification methods
 * - Trust levels
 * - Required metadata
 * - Claim extraction patterns
 */

// ============ Source Categories ============

/**
 * Primary source category
 */
export type SourceCategory =
  | "academic" // Peer-reviewed research, preprints
  | "patent" // Patents and IP filings
  | "regulatory" // Government, regulatory documents
  | "legal" // Court cases, contracts, legislation
  | "clinical" // Clinical trials, medical records
  | "financial" // Financial reports, filings
  | "technical" // Standards, specifications, documentation
  | "media" // News, press releases, announcements
  | "data" // Raw datasets, experimental data
  | "social"; // Social media, forums, user content

/**
 * Source reliability tiers
 *
 * Affects maximum confidence for claims from this source.
 */
export type SourceReliabilityTier =
  | "authoritative" // 0.95 max - Official records, peer-reviewed
  | "verified" // 0.85 max - Verified but not peer-reviewed
  | "attributed" // 0.75 max - Clear authorship, some verification
  | "unverified" // 0.60 max - Claimed authorship, limited verification
  | "anonymous"; // 0.40 max - Unknown or pseudonymous source

/**
 * Maximum confidence by reliability tier
 */
export const RELIABILITY_MAX_CONFIDENCE: Record<SourceReliabilityTier, number> =
  {
    authoritative: 0.95,
    verified: 0.85,
    attributed: 0.75,
    unverified: 0.6,
    anonymous: 0.4,
  };

// ============ Source Metadata ============

/**
 * Base source document metadata
 */
export interface SourceMetadata {
  /** Source category */
  category: SourceCategory;

  /** Reliability tier */
  reliability: SourceReliabilityTier;

  /** Unique identifier (DOI, URL, filing number, etc.) */
  identifier: string;

  /** Identifier type */
  identifierType: string;

  /** Title or headline */
  title: string;

  /** Publication/filing date */
  date: string;

  /** Content hash for verification */
  contentHash?: string;

  /** Archive URL (wayback machine, perma.cc, etc.) */
  archiveUrl?: string;

  /** Original URL (may change/disappear) */
  originalUrl?: string;
}

/**
 * Author/creator information
 */
export interface SourceAuthor {
  name: string;
  identifier?: string; // ORCID, email, handle
  affiliation?: string;
  role?: string; // author, inventor, editor, spokesperson
  verified?: boolean;
}

/**
 * Source verification status
 */
export interface SourceVerification {
  /** Has the source been independently verified? */
  verified: boolean;

  /** Verification method used */
  method?: "hash" | "archive" | "attestation" | "blockchain" | "notarization";

  /** When was it verified? */
  verifiedAt?: number;

  /** Who/what verified it? */
  verifier?: string;

  /** Verification proof (hash, signature, etc.) */
  proof?: string;
}

// ============ Claim Extraction Patterns ============

/**
 * How claims are typically extracted from this source type
 */
export interface ClaimExtractionPattern {
  /** Source category this applies to */
  category: SourceCategory;

  /** Typical claim types found in this source */
  claimTypes: string[];

  /** Sections where claims are typically found */
  claimLocations: string[];

  /** Required context for claims from this source */
  requiredContext: string[];

  /** Verification approach */
  verificationApproach: string;
}

/**
 * Claim extraction patterns by source category
 */
export const EXTRACTION_PATTERNS: Record<
  SourceCategory,
  ClaimExtractionPattern
> = {
  academic: {
    category: "academic",
    claimTypes: [
      "methodology",
      "measurement",
      "statistical",
      "causal",
      "interpretation",
    ],
    claimLocations: [
      "abstract",
      "methods",
      "results",
      "discussion",
      "conclusions",
    ],
    requiredContext: ["methodology", "sample_size", "statistical_test"],
    verificationApproach: "peer_review_and_replication",
  },
  patent: {
    category: "patent",
    claimTypes: [
      "invention",
      "prior_art",
      "technical_specification",
      "legal_status",
    ],
    claimLocations: ["claims", "abstract", "description", "drawings"],
    requiredContext: ["patent_number", "filing_date", "jurisdiction"],
    verificationApproach: "patent_office_records",
  },
  regulatory: {
    category: "regulatory",
    claimTypes: ["rule", "requirement", "approval", "violation", "status"],
    claimLocations: ["body", "appendix", "schedule", "amendment"],
    requiredContext: ["agency", "jurisdiction", "effective_date"],
    verificationApproach: "official_registry",
  },
  legal: {
    category: "legal",
    claimTypes: [
      "ruling",
      "precedent",
      "interpretation",
      "obligation",
      "right",
    ],
    claimLocations: ["holding", "dicta", "order", "terms"],
    requiredContext: ["court", "case_number", "date"],
    verificationApproach: "court_records",
  },
  clinical: {
    category: "clinical",
    claimTypes: ["efficacy", "safety", "adverse_event", "endpoint", "protocol"],
    claimLocations: ["results", "endpoints", "adverse_events", "protocol"],
    requiredContext: ["trial_id", "phase", "sponsor", "sample_size"],
    verificationApproach: "registry_and_publication",
  },
  financial: {
    category: "financial",
    claimTypes: ["metric", "projection", "risk", "audit_opinion", "disclosure"],
    claimLocations: ["financials", "notes", "mda", "risk_factors"],
    requiredContext: ["company", "period", "auditor", "currency"],
    verificationApproach: "audit_and_filing",
  },
  technical: {
    category: "technical",
    claimTypes: [
      "specification",
      "requirement",
      "conformance",
      "compatibility",
    ],
    claimLocations: ["normative", "informative", "annex", "conformance"],
    requiredContext: ["standard_body", "version", "status"],
    verificationApproach: "standards_body_publication",
  },
  media: {
    category: "media",
    claimTypes: ["event", "statement", "attribution", "analysis"],
    claimLocations: ["headline", "lead", "body", "quotes"],
    requiredContext: ["outlet", "author", "date", "sources_cited"],
    verificationApproach: "multiple_source_corroboration",
  },
  data: {
    category: "data",
    claimTypes: ["measurement", "observation", "aggregation", "transformation"],
    claimLocations: ["values", "metadata", "provenance"],
    requiredContext: ["collection_method", "instruments", "processing"],
    verificationApproach: "reproducibility_and_audit",
  },
  social: {
    category: "social",
    claimTypes: ["statement", "announcement", "opinion", "event"],
    claimLocations: ["post", "thread", "comment", "profile"],
    requiredContext: ["platform", "account", "timestamp"],
    verificationApproach: "archive_and_attestation",
  },
};

// ============ Source Trust Modifiers ============

/**
 * Factors that increase/decrease trust in a source
 */
export interface TrustModifiers {
  /** Positive factors (increase trust) */
  positive: {
    peerReviewed?: boolean;
    officialSource?: boolean;
    multipleCorroboration?: number;
    longTrackRecord?: boolean;
    verifiedIdentity?: boolean;
    auditedContent?: boolean;
    cryptographicProof?: boolean;
  };

  /** Negative factors (decrease trust) */
  negative: {
    previousRetractions?: number;
    conflictOfInterest?: boolean;
    anonymousSource?: boolean;
    singleSource?: boolean;
    editedAfterPublication?: boolean;
    paywalled?: boolean;
    knownBias?: string;
  };
}

/**
 * Calculate adjusted confidence based on trust modifiers
 */
export function calculateAdjustedConfidence(
  baseConfidence: number,
  reliability: SourceReliabilityTier,
  modifiers: TrustModifiers,
): number {
  let confidence = Math.min(
    baseConfidence,
    RELIABILITY_MAX_CONFIDENCE[reliability],
  );

  // Apply positive modifiers
  if (modifiers.positive.peerReviewed) confidence *= 1.1;
  if (modifiers.positive.officialSource) confidence *= 1.05;
  if (
    modifiers.positive.multipleCorroboration &&
    modifiers.positive.multipleCorroboration > 2
  ) {
    confidence *=
      1 + (Math.min(modifiers.positive.multipleCorroboration, 5) - 2) * 0.03;
  }
  if (modifiers.positive.cryptographicProof) confidence *= 1.05;
  if (modifiers.positive.verifiedIdentity) confidence *= 1.02;

  // Apply negative modifiers
  if (modifiers.negative.previousRetractions) {
    confidence *= Math.max(
      0.5,
      1 - modifiers.negative.previousRetractions * 0.1,
    );
  }
  if (modifiers.negative.conflictOfInterest) confidence *= 0.9;
  if (modifiers.negative.anonymousSource) confidence *= 0.8;
  if (modifiers.negative.singleSource) confidence *= 0.9;
  if (modifiers.negative.editedAfterPublication) confidence *= 0.85;

  // Cap at reliability tier maximum
  return Math.min(confidence, RELIABILITY_MAX_CONFIDENCE[reliability]);
}
