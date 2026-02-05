/**
 * Patent Source Schemas
 *
 * Schemas for patent documents and intellectual property filings.
 *
 * Patents are authoritative sources for:
 * - Technical inventions and methods
 * - Prior art references
 * - Legal status of IP
 * - Ownership and licensing
 */

import type { ClaimSchema } from "../types.js";

// ============ Patent Document Schema ============

/**
 * Patent Document Schema
 *
 * A granted patent or patent application.
 */
export const PATENT_DOCUMENT_SCHEMA: ClaimSchema = {
  id: "patent_document_v1",
  version: 1,
  name: "Patent Document",
  description: "A patent or patent application with extractable claims",
  domain: "intellectual_property.patents",

  timing: {
    commitPhaseBlocks: 10,
    revealPhaseBlocks: 6,
    totalRoundBlocks: 18,
    minAnalysisTimeMs: 12000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Patent Identification ============
      patentNumber: {
        type: "string",
        description: "Patent or application number (e.g., US10123456B2)",
      },
      applicationNumber: {
        type: "string",
        description: "Original application number",
      },
      jurisdiction: {
        type: "string",
        enum: [
          "US",
          "EP",
          "WO",
          "CN",
          "JP",
          "KR",
          "DE",
          "GB",
          "FR",
          "CA",
          "AU",
          "IN",
          "other",
        ],
        description: "Patent office jurisdiction",
      },
      patentType: {
        type: "string",
        enum: [
          "utility",
          "design",
          "plant",
          "provisional",
          "continuation",
          "divisional",
          "reissue",
        ],
        description: "Type of patent",
      },

      // ============ Dates ============
      filingDate: {
        type: "string",
        description: "Original filing date (ISO 8601)",
      },
      publicationDate: {
        type: "string",
        description: "Publication date",
      },
      grantDate: {
        type: "string",
        description: "Grant date (if granted)",
      },
      priorityDate: {
        type: "string",
        description: "Priority date (earliest filing)",
      },
      expirationDate: {
        type: "string",
        description: "Expected expiration date",
      },

      // ============ Parties ============
      inventors: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            country: { type: "string" },
          },
        },
        description: "Named inventors",
      },
      applicant: {
        type: "string",
        description: "Original applicant",
      },
      assignee: {
        type: "string",
        description: "Current assignee/owner",
      },
      attorneys: {
        type: "array",
        items: { type: "string" },
        description: "Patent attorneys/agents",
      },

      // ============ Content ============
      title: {
        type: "string",
        description: "Patent title",
      },
      abstract: {
        type: "string",
        description: "Patent abstract",
      },
      abstractHash: {
        type: "string",
        description: "Hash of abstract for verification",
      },

      // ============ Classification ============
      ipcClassifications: {
        type: "array",
        items: { type: "string" },
        description: "International Patent Classification codes",
      },
      cpcClassifications: {
        type: "array",
        items: { type: "string" },
        description: "Cooperative Patent Classification codes",
      },
      uspcClassifications: {
        type: "array",
        items: { type: "string" },
        description: "US Patent Classification codes",
      },

      // ============ Legal Status ============
      status: {
        type: "string",
        enum: [
          "pending",
          "granted",
          "expired",
          "abandoned",
          "withdrawn",
          "rejected",
          "lapsed",
        ],
        description: "Current legal status",
      },
      maintenanceFeesPaid: {
        type: "boolean",
        description: "Are maintenance fees current?",
      },

      // ============ Claims ============
      independentClaimCount: {
        type: "number",
        description: "Number of independent claims",
      },
      dependentClaimCount: {
        type: "number",
        description: "Number of dependent claims",
      },
      claimText: {
        type: "array",
        items: {
          type: "object",
          properties: {
            claimNumber: { type: "number" },
            isIndependent: { type: "boolean" },
            text: { type: "string" },
            dependsOn: { type: "array", items: { type: "number" } },
          },
        },
        description: "Patent claims",
      },

      // ============ References ============
      citedPatents: {
        type: "array",
        items: { type: "string" },
        description: "Patents cited as prior art",
      },
      citedNonPatent: {
        type: "array",
        items: { type: "string" },
        description: "Non-patent literature cited",
      },
      citedByCount: {
        type: "number",
        description: "Number of patents citing this one",
      },

      // ============ Family ============
      familyId: {
        type: "string",
        description: "Patent family identifier",
      },
      familyMembers: {
        type: "array",
        items: { type: "string" },
        description: "Related patents in the family",
      },
    },
    required: ["patentNumber", "jurisdiction", "title", "status"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ Patent Claim Schema ============

/**
 * Patent Claim Schema
 *
 * A specific claim from a patent (technical assertion).
 */
export const PATENT_CLAIM_SCHEMA: ClaimSchema = {
  id: "patent_claim_v1",
  version: 1,
  name: "Patent Claim",
  description: "A specific claim from a patent document",
  domain: "intellectual_property.patent_claims",

  timing: {
    commitPhaseBlocks: 8,
    revealPhaseBlocks: 6,
    totalRoundBlocks: 16,
    minAnalysisTimeMs: 10000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      patentNumber: {
        type: "string",
        description: "Source patent number",
      },
      claimNumber: {
        type: "number",
        description: "Claim number within patent",
      },
      claimType: {
        type: "string",
        enum: ["independent", "dependent"],
        description: "Whether claim is independent or dependent",
      },
      claimText: {
        type: "string",
        description: "Full text of the claim",
      },
      claimCategory: {
        type: "string",
        enum: [
          "method",
          "apparatus",
          "composition",
          "system",
          "article",
          "use",
        ],
        description: "Category of claim",
      },
      dependsOn: {
        type: "array",
        items: { type: "number" },
        description: "Claim numbers this depends on",
      },

      // ============ Claim Elements ============
      elements: {
        type: "array",
        items: {
          type: "object",
          properties: {
            element: { type: "string" },
            description: { type: "string" },
          },
        },
        description: "Claim elements/limitations",
      },

      // ============ Validity Status ============
      validityStatus: {
        type: "string",
        enum: [
          "presumed_valid",
          "challenged",
          "invalidated",
          "narrowed",
          "confirmed",
        ],
        description: "Current validity status",
      },
      challengeHistory: {
        type: "array",
        items: {
          type: "object",
          properties: {
            proceeding: { type: "string" },
            date: { type: "string" },
            outcome: { type: "string" },
          },
        },
        description: "History of validity challenges",
      },
    },
    required: ["patentNumber", "claimNumber", "claimText"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ Prior Art Schema ============

/**
 * Prior Art Reference Schema
 *
 * A claim that something constitutes prior art.
 */
export const PRIOR_ART_SCHEMA: ClaimSchema = {
  id: "prior_art_v1",
  version: 1,
  name: "Prior Art Reference",
  description: "A claim that prior art exists for a patent claim",
  domain: "intellectual_property.prior_art",

  timing: {
    commitPhaseBlocks: 12,
    revealPhaseBlocks: 8,
    totalRoundBlocks: 22,
    minAnalysisTimeMs: 15000,
  },

  verificationMethod: "consensus",

  dataSchema: {
    type: "object",
    properties: {
      targetPatent: {
        type: "string",
        description: "Patent being challenged",
      },
      targetClaims: {
        type: "array",
        items: { type: "number" },
        description: "Specific claims this is prior art for",
      },

      // ============ Prior Art Source ============
      priorArtType: {
        type: "string",
        enum: [
          "patent",
          "publication",
          "product",
          "public_use",
          "oral_disclosure",
          "other",
        ],
        description: "Type of prior art",
      },
      priorArtReference: {
        type: "string",
        description: "Reference to the prior art (patent number, DOI, etc.)",
      },
      priorArtDate: {
        type: "string",
        description: "Date of prior art (must be before priority date)",
      },
      priorArtDescription: {
        type: "string",
        description: "Description of the prior art disclosure",
      },

      // ============ Analysis ============
      elementsDisclosed: {
        type: "array",
        items: { type: "string" },
        description: "Which claim elements are disclosed",
      },
      elementsMissing: {
        type: "array",
        items: { type: "string" },
        description: "Which claim elements are not disclosed",
      },
      anticipates: {
        type: "boolean",
        description: "Does this anticipate (disclose all elements)?",
      },
      obviousnessAnalysis: {
        type: "string",
        description: "Analysis for obviousness combination",
      },
      combinedWith: {
        type: "array",
        items: { type: "string" },
        description: "Other references for obviousness combination",
      },
    },
    required: [
      "targetPatent",
      "priorArtType",
      "priorArtReference",
      "priorArtDate",
    ],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional", "archived_web"],
  minReferences: 2,

  active: true,
};

// ============ Exports ============

export const PATENT_SCHEMAS = [
  PATENT_DOCUMENT_SCHEMA,
  PATENT_CLAIM_SCHEMA,
  PRIOR_ART_SCHEMA,
];
