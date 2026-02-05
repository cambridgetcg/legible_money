/**
 * Technical Source Schemas
 *
 * Schemas for technical standards, specifications, and documentation.
 *
 * Technical sources include:
 * - International standards (ISO, IEC, IEEE)
 * - Industry specifications (W3C, IETF, 3GPP)
 * - API documentation
 * - Software releases
 */

import type { ClaimSchema } from "../types.js";

// ============ Technical Standard Schema ============

/**
 * Technical Standard Schema
 *
 * An official technical standard from a standards body.
 */
export const TECHNICAL_STANDARD_SCHEMA: ClaimSchema = {
  id: "technical_standard_v1",
  version: 1,
  name: "Technical Standard",
  description: "An official technical standard",
  domain: "technical.standards",

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
      // ============ Standard Identification ============
      standardId: {
        type: "string",
        description: "Standard identifier (e.g., ISO 9001:2015)",
      },
      title: {
        type: "string",
        description: "Full title of the standard",
      },
      version: {
        type: "string",
        description: "Version/edition",
      },

      // ============ Standards Body ============
      standardsBody: {
        type: "string",
        description: "Issuing standards organization",
      },
      bodyType: {
        type: "string",
        enum: [
          "international", // ISO, IEC, ITU
          "regional", // CEN, ETSI
          "national", // ANSI, BSI, DIN
          "industry", // IEEE, IETF, W3C
          "consortium", // 3GPP, Bluetooth SIG
          "de_facto", // Widely adopted without formal body
          "other",
        ],
        description: "Type of standards organization",
      },
      technicalCommittee: {
        type: "string",
        description: "Technical committee/working group",
      },

      // ============ Status ============
      status: {
        type: "string",
        enum: [
          "draft",
          "proposed",
          "published",
          "revised",
          "withdrawn",
          "superseded",
          "stabilized",
        ],
        description: "Current status",
      },
      publicationDate: {
        type: "string",
        description: "Date of publication",
      },
      effectiveDate: {
        type: "string",
        description: "Date standard takes effect",
      },
      supersededBy: {
        type: "string",
        description: "Superseding standard (if applicable)",
      },
      supersedes: {
        type: "string",
        description: "Standard this supersedes",
      },

      // ============ Scope ============
      scope: {
        type: "string",
        description: "Scope of the standard",
      },
      abstract: {
        type: "string",
        description: "Abstract/summary",
      },
      keywords: {
        type: "array",
        items: { type: "string" },
        description: "Keywords/tags",
      },
      icsCode: {
        type: "string",
        description: "International Classification for Standards code",
      },

      // ============ Classification ============
      standardType: {
        type: "string",
        enum: [
          "terminology",
          "specification",
          "test_method",
          "practice",
          "guide",
          "classification",
          "symbol",
          "other",
        ],
        description: "Type of standard",
      },
      domain: {
        type: "string",
        description: "Technical domain",
      },

      // ============ Conformance ============
      conformanceLevels: {
        type: "array",
        items: { type: "string" },
        description: "Defined conformance levels",
      },
      testSuiteAvailable: {
        type: "boolean",
        description: "Is a test suite available?",
      },
      certificationProgram: {
        type: "string",
        description: "Associated certification program",
      },

      // ============ References ============
      normativeReferences: {
        type: "array",
        items: { type: "string" },
        description: "Normative references",
      },
      informativeReferences: {
        type: "array",
        items: { type: "string" },
        description: "Informative references",
      },

      // ============ Access ============
      officialUrl: {
        type: "string",
        description: "Official URL",
      },
      publiclyAvailable: {
        type: "boolean",
        description: "Is standard freely available?",
      },
      purchaseUrl: {
        type: "string",
        description: "URL to purchase (if not free)",
      },
    },
    required: ["standardId", "title", "standardsBody", "status"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ RFC/Internet Draft Schema ============

/**
 * RFC Schema
 *
 * An IETF Request for Comments or Internet Draft.
 */
export const RFC_SCHEMA: ClaimSchema = {
  id: "rfc_v1",
  version: 1,
  name: "RFC",
  description: "An IETF RFC or Internet Draft",
  domain: "technical.ietf",

  timing: {
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 4,
    totalRoundBlocks: 12,
    minAnalysisTimeMs: 8000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Identification ============
      rfcNumber: {
        type: "number",
        description: "RFC number (for published RFCs)",
      },
      draftName: {
        type: "string",
        description: "Internet-Draft name (for drafts)",
      },
      title: {
        type: "string",
        description: "Document title",
      },

      // ============ Authors ============
      authors: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            organization: { type: "string" },
            email: { type: "string" },
          },
        },
        description: "Document authors",
      },

      // ============ Status ============
      status: {
        type: "string",
        enum: [
          "informational",
          "experimental",
          "best_current_practice",
          "proposed_standard",
          "draft_standard",
          "internet_standard",
          "historic",
          "unknown",
        ],
        description: "RFC status",
      },
      stream: {
        type: "string",
        enum: ["ietf", "irtf", "iab", "independent"],
        description: "RFC stream",
      },
      workingGroup: {
        type: "string",
        description: "IETF working group",
      },

      // ============ Dates ============
      publicationDate: {
        type: "string",
        description: "Publication date",
      },
      expirationDate: {
        type: "string",
        description: "Expiration date (for drafts)",
      },

      // ============ Updates/Obsoletes ============
      obsoletes: {
        type: "array",
        items: { type: "number" },
        description: "RFC numbers this obsoletes",
      },
      obsoletedBy: {
        type: "array",
        items: { type: "number" },
        description: "RFCs that obsolete this",
      },
      updates: {
        type: "array",
        items: { type: "number" },
        description: "RFC numbers this updates",
      },
      updatedBy: {
        type: "array",
        items: { type: "number" },
        description: "RFCs that update this",
      },

      // ============ Content ============
      abstract: {
        type: "string",
        description: "Document abstract",
      },
      keywords: {
        type: "array",
        items: { type: "string" },
        description: "Keywords",
      },
      area: {
        type: "string",
        description: "IETF area",
      },

      // ============ Errata ============
      hasErrata: {
        type: "boolean",
        description: "Has verified errata?",
      },
      errataUrl: {
        type: "string",
        description: "Errata URL",
      },

      // ============ References ============
      rfcEditorUrl: {
        type: "string",
        description: "RFC Editor URL",
      },
      datatrackerUrl: {
        type: "string",
        description: "Datatracker URL",
      },
    },
    required: ["title", "status", "publicationDate"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ API Documentation Schema ============

/**
 * API Documentation Schema
 *
 * Documentation for a software API.
 */
export const API_DOCUMENTATION_SCHEMA: ClaimSchema = {
  id: "api_documentation_v1",
  version: 1,
  name: "API Documentation",
  description: "Documentation for a software API",
  domain: "technical.api",

  timing: {
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 4,
    totalRoundBlocks: 12,
    minAnalysisTimeMs: 6000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ API Identification ============
      apiName: {
        type: "string",
        description: "API name",
      },
      apiVersion: {
        type: "string",
        description: "API version",
      },
      provider: {
        type: "string",
        description: "API provider/organization",
      },
      baseUrl: {
        type: "string",
        description: "API base URL",
      },

      // ============ API Type ============
      apiType: {
        type: "string",
        enum: [
          "rest",
          "graphql",
          "grpc",
          "soap",
          "websocket",
          "rpc",
          "sdk",
          "other",
        ],
        description: "API architecture type",
      },
      specificationFormat: {
        type: "string",
        enum: [
          "openapi",
          "asyncapi",
          "graphql_schema",
          "protobuf",
          "wsdl",
          "custom",
        ],
        description: "Specification format",
      },

      // ============ Status ============
      status: {
        type: "string",
        enum: ["alpha", "beta", "stable", "deprecated", "retired"],
        description: "API status",
      },
      releaseDate: {
        type: "string",
        description: "Release date",
      },
      deprecationDate: {
        type: "string",
        description: "Deprecation date (if applicable)",
      },
      sunsetDate: {
        type: "string",
        description: "Sunset/retirement date",
      },

      // ============ Endpoints/Operations ============
      endpointCount: {
        type: "number",
        description: "Number of endpoints/operations",
      },
      endpoints: {
        type: "array",
        items: {
          type: "object",
          properties: {
            path: { type: "string" },
            method: { type: "string" },
            description: { type: "string" },
            deprecated: { type: "boolean" },
          },
        },
        description: "Key endpoints (summary)",
      },

      // ============ Authentication ============
      authMethods: {
        type: "array",
        items: {
          type: "string",
          enum: ["api_key", "oauth2", "basic", "bearer", "jwt", "mtls", "none"],
        },
        description: "Supported authentication methods",
      },

      // ============ Rate Limits ============
      rateLimits: {
        type: "object",
        properties: {
          requestsPerSecond: { type: "number" },
          requestsPerDay: { type: "number" },
          burstLimit: { type: "number" },
        },
        description: "Rate limiting details",
      },

      // ============ References ============
      documentationUrl: {
        type: "string",
        description: "Documentation URL",
      },
      specificationUrl: {
        type: "string",
        description: "OpenAPI/spec URL",
      },
      changelogUrl: {
        type: "string",
        description: "Changelog URL",
      },
      archiveUrl: {
        type: "string",
        description: "Archived documentation URL",
      },
      contentHash: {
        type: "string",
        description: "Hash of documentation at archive time",
      },
    },
    required: ["apiName", "apiVersion", "provider", "apiType", "status"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["archived_web"],
  minReferences: 1,

  active: true,
};

// ============ Software Release Schema ============

/**
 * Software Release Schema
 *
 * A software version release.
 */
export const SOFTWARE_RELEASE_SCHEMA: ClaimSchema = {
  id: "software_release_v1",
  version: 1,
  name: "Software Release",
  description: "A software version release",
  domain: "technical.software",

  timing: {
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 4,
    totalRoundBlocks: 12,
    minAnalysisTimeMs: 6000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Software Identification ============
      softwareName: {
        type: "string",
        description: "Software/project name",
      },
      version: {
        type: "string",
        description: "Version number",
      },
      versionType: {
        type: "string",
        enum: [
          "major",
          "minor",
          "patch",
          "prerelease",
          "alpha",
          "beta",
          "rc",
          "lts",
        ],
        description: "Version type",
      },

      // ============ Publisher ============
      publisher: {
        type: "string",
        description: "Publisher/maintainer",
      },
      repository: {
        type: "string",
        description: "Source repository URL",
      },
      license: {
        type: "string",
        description: "Software license (SPDX identifier)",
      },

      // ============ Release Details ============
      releaseDate: {
        type: "string",
        description: "Release date",
      },
      releaseNotes: {
        type: "string",
        description: "Release notes URL",
      },
      changelog: {
        type: "array",
        items: {
          type: "object",
          properties: {
            type: { type: "string" }, // feature, fix, breaking, security
            description: { type: "string" },
          },
        },
        description: "Key changes",
      },

      // ============ Distribution ============
      distributionChannels: {
        type: "array",
        items: {
          type: "object",
          properties: {
            channel: { type: "string" }, // npm, pypi, github, maven, etc.
            packageName: { type: "string" },
            url: { type: "string" },
          },
        },
        description: "Distribution channels",
      },

      // ============ Artifacts ============
      artifacts: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            platform: { type: "string" },
            architecture: { type: "string" },
            downloadUrl: { type: "string" },
            sha256: { type: "string" },
            size: { type: "number" },
          },
        },
        description: "Release artifacts with checksums",
      },

      // ============ Signatures ============
      gpgSignature: {
        type: "string",
        description: "GPG signature URL",
      },
      signingKeyId: {
        type: "string",
        description: "Signing key ID",
      },

      // ============ Support ============
      supportStatus: {
        type: "string",
        enum: ["current", "security_only", "maintenance", "eol"],
        description: "Support status",
      },
      eolDate: {
        type: "string",
        description: "End of life date",
      },

      // ============ Security ============
      securityAdvisories: {
        type: "array",
        items: {
          type: "object",
          properties: {
            cveId: { type: "string" },
            severity: { type: "string" },
            fixed: { type: "boolean" },
          },
        },
        description: "Known security advisories",
      },
    },
    required: ["softwareName", "version", "releaseDate"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["archived_web"],
  minReferences: 1,

  active: true,
};

// ============ Benchmark Result Schema ============

/**
 * Benchmark Result Schema
 *
 * A performance benchmark result.
 */
export const BENCHMARK_RESULT_SCHEMA: ClaimSchema = {
  id: "benchmark_result_v1",
  version: 1,
  name: "Benchmark Result",
  description: "A performance benchmark result",
  domain: "technical.benchmarks",

  timing: {
    commitPhaseBlocks: 8,
    revealPhaseBlocks: 6,
    totalRoundBlocks: 16,
    minAnalysisTimeMs: 10000,
  },

  verificationMethod: "consensus",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Benchmark ============
      benchmarkName: {
        type: "string",
        description: "Benchmark name/suite",
      },
      benchmarkVersion: {
        type: "string",
        description: "Benchmark version",
      },
      benchmarkType: {
        type: "string",
        enum: [
          "cpu",
          "gpu",
          "memory",
          "storage",
          "network",
          "database",
          "ml_inference",
          "ml_training",
          "application",
          "other",
        ],
        description: "Benchmark type",
      },

      // ============ Subject ============
      subjectType: {
        type: "string",
        enum: ["hardware", "software", "system", "cloud_service", "model"],
        description: "What was benchmarked",
      },
      subjectName: {
        type: "string",
        description: "Subject name",
      },
      subjectVersion: {
        type: "string",
        description: "Subject version",
      },

      // ============ Results ============
      results: {
        type: "array",
        items: {
          type: "object",
          properties: {
            metric: { type: "string" },
            value: { type: "number" },
            unit: { type: "string" },
            higherIsBetter: { type: "boolean" },
          },
        },
        description: "Benchmark results",
      },
      overallScore: {
        type: "number",
        description: "Overall composite score",
      },

      // ============ Configuration ============
      configuration: {
        type: "object",
        properties: {
          os: { type: "string" },
          cpu: { type: "string" },
          memory: { type: "string" },
          gpu: { type: "string" },
          settings: { type: "object" },
        },
        description: "Test configuration",
      },

      // ============ Reproducibility ============
      runDate: {
        type: "string",
        description: "Date benchmark was run",
      },
      runCount: {
        type: "number",
        description: "Number of runs",
      },
      methodology: {
        type: "string",
        description: "Testing methodology",
      },

      // ============ Source ============
      sourceType: {
        type: "string",
        enum: ["official", "independent", "community", "self_reported"],
        description: "Source of benchmark",
      },
      sourceUrl: {
        type: "string",
        description: "Source URL",
      },
      rawDataUrl: {
        type: "string",
        description: "Raw data URL (if available)",
      },
    },
    required: [
      "benchmarkName",
      "subjectType",
      "subjectName",
      "results",
      "runDate",
    ],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["archived_web"],
  minReferences: 1,

  active: true,
};

// ============ Exports ============

export const TECHNICAL_SCHEMAS = [
  TECHNICAL_STANDARD_SCHEMA,
  RFC_SCHEMA,
  API_DOCUMENTATION_SCHEMA,
  SOFTWARE_RELEASE_SCHEMA,
  BENCHMARK_RESULT_SCHEMA,
];
