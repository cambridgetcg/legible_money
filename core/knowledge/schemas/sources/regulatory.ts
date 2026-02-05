/**
 * Regulatory Source Schemas
 *
 * Schemas for government documents, regulatory filings, and official records.
 *
 * Regulatory sources are authoritative but jurisdiction-specific:
 * - Official government publications
 * - Agency decisions and guidance
 * - Permits, licenses, approvals
 * - Enforcement actions
 */

import type { ClaimSchema } from "../types.js";

// ============ Government Document Schema ============

/**
 * Government Document Schema
 *
 * An official government publication or record.
 */
export const GOVERNMENT_DOCUMENT_SCHEMA: ClaimSchema = {
  id: "government_document_v1",
  version: 1,
  name: "Government Document",
  description: "An official government publication or record",
  domain: "regulatory.government",

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
      // ============ Document Identification ============
      documentId: {
        type: "string",
        description:
          "Official document identifier (e.g., Federal Register citation)",
      },
      documentType: {
        type: "string",
        enum: [
          "statute",
          "regulation",
          "executive_order",
          "agency_guidance",
          "notice",
          "proposed_rule",
          "final_rule",
          "proclamation",
          "memorandum",
          "report",
          "decision",
          "other",
        ],
        description: "Type of government document",
      },
      title: {
        type: "string",
        description: "Document title",
      },

      // ============ Jurisdiction ============
      country: {
        type: "string",
        description: "Country (ISO 3166-1 alpha-2)",
      },
      subdivision: {
        type: "string",
        description: "State/province (if applicable)",
      },
      jurisdiction: {
        type: "string",
        enum: ["federal", "state", "local", "international", "treaty"],
        description: "Jurisdictional level",
      },

      // ============ Issuing Authority ============
      issuingAgency: {
        type: "string",
        description: "Agency or body that issued the document",
      },
      agencyType: {
        type: "string",
        enum: [
          "executive",
          "legislative",
          "judicial",
          "independent",
          "advisory",
        ],
        description: "Type of issuing authority",
      },

      // ============ Dates ============
      publicationDate: {
        type: "string",
        description: "Date of publication (ISO 8601)",
      },
      effectiveDate: {
        type: "string",
        description: "Date the document takes effect",
      },
      expirationDate: {
        type: "string",
        description: "Expiration date (if applicable)",
      },

      // ============ Legal Status ============
      status: {
        type: "string",
        enum: [
          "proposed",
          "effective",
          "superseded",
          "repealed",
          "expired",
          "stayed",
        ],
        description: "Current legal status",
      },
      supersededBy: {
        type: "string",
        description: "Document that superseded this one",
      },
      amends: {
        type: "array",
        items: { type: "string" },
        description: "Documents this amends",
      },

      // ============ References ============
      officialUrl: {
        type: "string",
        description: "Official government URL",
      },
      archiveUrl: {
        type: "string",
        description: "Archive URL for permanence",
      },
      contentHash: {
        type: "string",
        description: "Hash of document content",
      },
      citations: {
        type: "array",
        items: { type: "string" },
        description: "Legal citations (e.g., 42 U.S.C. § 1234)",
      },
    },
    required: [
      "documentId",
      "documentType",
      "title",
      "country",
      "issuingAgency",
      "publicationDate",
    ],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ Regulatory Approval Schema ============

/**
 * Regulatory Approval Schema
 *
 * An approval, license, or permit from a regulatory agency.
 */
export const REGULATORY_APPROVAL_SCHEMA: ClaimSchema = {
  id: "regulatory_approval_v1",
  version: 1,
  name: "Regulatory Approval",
  description: "A regulatory approval, license, or permit",
  domain: "regulatory.approvals",

  timing: {
    commitPhaseBlocks: 8,
    revealPhaseBlocks: 6,
    totalRoundBlocks: 16,
    minAnalysisTimeMs: 8000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Approval Identification ============
      approvalNumber: {
        type: "string",
        description: "Official approval/license number",
      },
      approvalType: {
        type: "string",
        enum: [
          "drug_approval", // FDA NDA/BLA
          "device_approval", // FDA PMA/510k
          "environmental", // EPA permits
          "securities", // SEC registration
          "banking", // Banking charter
          "broadcast", // FCC license
          "import_export", // Trade permits
          "professional", // Professional license
          "construction", // Building permits
          "other",
        ],
        description: "Type of approval",
      },

      // ============ Regulator ============
      regulatoryAgency: {
        type: "string",
        description: "Issuing regulatory agency (e.g., FDA, SEC, EPA)",
      },
      agencyDivision: {
        type: "string",
        description: "Specific division within agency",
      },
      jurisdiction: {
        type: "string",
        description: "Jurisdiction of approval",
      },

      // ============ Applicant ============
      applicant: {
        type: "string",
        description: "Entity that received approval",
      },
      applicantType: {
        type: "string",
        enum: [
          "corporation",
          "individual",
          "government",
          "nonprofit",
          "partnership",
        ],
        description: "Type of applicant",
      },

      // ============ Subject ============
      subjectName: {
        type: "string",
        description: "What was approved (product name, facility, etc.)",
      },
      subjectDescription: {
        type: "string",
        description: "Description of approved subject",
      },
      indication: {
        type: "string",
        description: "Approved indication/use (for medical products)",
      },

      // ============ Dates ============
      applicationDate: {
        type: "string",
        description: "Date of application submission",
      },
      approvalDate: {
        type: "string",
        description: "Date of approval",
      },
      expirationDate: {
        type: "string",
        description: "Expiration date (if applicable)",
      },

      // ============ Status ============
      status: {
        type: "string",
        enum: [
          "active",
          "suspended",
          "revoked",
          "expired",
          "withdrawn",
          "pending",
        ],
        description: "Current status of approval",
      },
      conditions: {
        type: "array",
        items: { type: "string" },
        description: "Conditions attached to approval",
      },

      // ============ References ============
      officialUrl: {
        type: "string",
        description: "Official database URL",
      },
      approvalLetter: {
        type: "string",
        description: "URL to approval letter",
      },
      labelUrl: {
        type: "string",
        description: "URL to approved labeling (if applicable)",
      },
    },
    required: [
      "approvalNumber",
      "approvalType",
      "regulatoryAgency",
      "applicant",
      "approvalDate",
      "status",
    ],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ Enforcement Action Schema ============

/**
 * Enforcement Action Schema
 *
 * A regulatory enforcement action or penalty.
 */
export const ENFORCEMENT_ACTION_SCHEMA: ClaimSchema = {
  id: "enforcement_action_v1",
  version: 1,
  name: "Enforcement Action",
  description: "A regulatory enforcement action or penalty",
  domain: "regulatory.enforcement",

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
      // ============ Action Identification ============
      actionId: {
        type: "string",
        description: "Official action/case number",
      },
      actionType: {
        type: "string",
        enum: [
          "warning_letter",
          "cease_and_desist",
          "civil_penalty",
          "criminal_referral",
          "consent_decree",
          "suspension",
          "revocation",
          "import_alert",
          "recall",
          "injunction",
          "administrative_order",
          "settlement",
          "other",
        ],
        description: "Type of enforcement action",
      },

      // ============ Agency ============
      enforcementAgency: {
        type: "string",
        description: "Agency taking action",
      },
      jurisdiction: {
        type: "string",
        description: "Jurisdiction",
      },

      // ============ Target ============
      targetEntity: {
        type: "string",
        description: "Entity subject to enforcement",
      },
      targetType: {
        type: "string",
        enum: ["corporation", "individual", "facility", "product", "other"],
        description: "Type of enforcement target",
      },

      // ============ Violation ============
      violations: {
        type: "array",
        items: {
          type: "object",
          properties: {
            statute: { type: "string" },
            description: { type: "string" },
            severity: { type: "string" },
          },
        },
        description: "Violations alleged/found",
      },

      // ============ Penalties ============
      monetaryPenalty: {
        type: "number",
        description: "Monetary penalty amount",
      },
      penaltyCurrency: {
        type: "string",
        description: "Currency of penalty",
      },
      nonMonetaryPenalties: {
        type: "array",
        items: { type: "string" },
        description: "Non-monetary penalties (bans, required actions, etc.)",
      },

      // ============ Dates ============
      actionDate: {
        type: "string",
        description: "Date of enforcement action",
      },
      complianceDeadline: {
        type: "string",
        description: "Deadline for compliance",
      },

      // ============ Status ============
      status: {
        type: "string",
        enum: ["pending", "active", "resolved", "appealed", "vacated"],
        description: "Current status",
      },
      resolution: {
        type: "string",
        description: "How the action was resolved",
      },

      // ============ References ============
      officialUrl: {
        type: "string",
        description: "Official announcement URL",
      },
      pressRelease: {
        type: "string",
        description: "Press release URL",
      },
    },
    required: [
      "actionId",
      "actionType",
      "enforcementAgency",
      "targetEntity",
      "actionDate",
    ],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ Exports ============

export const REGULATORY_SCHEMAS = [
  GOVERNMENT_DOCUMENT_SCHEMA,
  REGULATORY_APPROVAL_SCHEMA,
  ENFORCEMENT_ACTION_SCHEMA,
];
