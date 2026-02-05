/**
 * Legal Source Schemas
 *
 * Schemas for court cases, contracts, legislation, and legal documents.
 *
 * Legal sources have varying authority levels:
 * - Supreme Court > Appellate > Trial courts
 * - Statutes > Regulations > Guidance
 * - Contracts bind only parties
 */

import type { ClaimSchema } from "../types.js";

// ============ Court Case Schema ============

/**
 * Court Case Schema
 *
 * A judicial decision or ruling.
 */
export const COURT_CASE_SCHEMA: ClaimSchema = {
  id: "court_case_v1",
  version: 1,
  name: "Court Case",
  description: "A court case with judicial decisions",
  domain: "legal.cases",

  timing: {
    commitPhaseBlocks: 10,
    revealPhaseBlocks: 8,
    totalRoundBlocks: 20,
    minAnalysisTimeMs: 15000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Case Identification ============
      caseNumber: {
        type: "string",
        description: "Official case number/docket number",
      },
      caseName: {
        type: "string",
        description: "Case name (e.g., Smith v. Jones)",
      },
      citation: {
        type: "string",
        description: "Legal citation (e.g., 123 F.3d 456)",
      },
      alternativeCitations: {
        type: "array",
        items: { type: "string" },
        description: "Alternative citations (parallel citations)",
      },

      // ============ Court ============
      court: {
        type: "string",
        description: "Court name",
      },
      courtLevel: {
        type: "string",
        enum: [
          "supreme",
          "appellate",
          "trial",
          "administrative",
          "arbitration",
          "international",
        ],
        description: "Level of court",
      },
      jurisdiction: {
        type: "string",
        description: "Jurisdiction (country, state, circuit)",
      },
      judge: {
        type: "string",
        description: "Presiding judge (or panel)",
      },

      // ============ Parties ============
      plaintiff: {
        type: "string",
        description: "Plaintiff/Petitioner",
      },
      defendant: {
        type: "string",
        description: "Defendant/Respondent",
      },
      otherParties: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            role: { type: "string" },
          },
        },
        description: "Intervenors, amici, etc.",
      },

      // ============ Case Details ============
      caseType: {
        type: "string",
        enum: [
          "civil",
          "criminal",
          "administrative",
          "constitutional",
          "bankruptcy",
          "family",
          "tax",
          "patent",
          "trademark",
          "copyright",
          "antitrust",
          "securities",
          "environmental",
          "labor",
          "immigration",
          "other",
        ],
        description: "Type of case",
      },
      subjectMatter: {
        type: "array",
        items: { type: "string" },
        description: "Subject matter areas",
      },
      legalIssues: {
        type: "array",
        items: { type: "string" },
        description: "Legal issues addressed",
      },

      // ============ Dates ============
      filingDate: {
        type: "string",
        description: "Date case was filed",
      },
      decisionDate: {
        type: "string",
        description: "Date of decision",
      },

      // ============ Decision ============
      outcome: {
        type: "string",
        enum: [
          "plaintiff_judgment",
          "defendant_judgment",
          "affirmed",
          "reversed",
          "remanded",
          "dismissed",
          "settled",
          "vacated",
          "partial",
          "pending",
        ],
        description: "Case outcome",
      },
      holding: {
        type: "string",
        description: "Main holding of the case",
      },
      holdingHash: {
        type: "string",
        description: "Hash of holding text",
      },

      // ============ Precedential Value ============
      precedentialStatus: {
        type: "string",
        enum: [
          "binding",
          "persuasive",
          "unpublished",
          "overruled",
          "distinguished",
          "questioned",
        ],
        description: "Precedential status",
      },
      overruledBy: {
        type: "string",
        description: "Citation of overruling case (if applicable)",
      },
      keyHeadnotes: {
        type: "array",
        items: { type: "string" },
        description: "Key legal points/headnotes",
      },

      // ============ Appeals ============
      appealedFrom: {
        type: "string",
        description: "Lower court citation (if appeal)",
      },
      appealedTo: {
        type: "string",
        description: "Higher court citation (if appealed)",
      },

      // ============ References ============
      courtUrl: {
        type: "string",
        description: "Official court URL",
      },
      westlawCitation: {
        type: "string",
        description: "Westlaw citation",
      },
      lexisCitation: {
        type: "string",
        description: "LexisNexis citation",
      },
    },
    required: ["caseNumber", "caseName", "court", "decisionDate"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ Legislation Schema ============

/**
 * Legislation Schema
 *
 * A statute, law, or legislative act.
 */
export const LEGISLATION_SCHEMA: ClaimSchema = {
  id: "legislation_v1",
  version: 1,
  name: "Legislation",
  description: "A statute, law, or legislative act",
  domain: "legal.legislation",

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
      // ============ Identification ============
      billNumber: {
        type: "string",
        description: "Bill number (e.g., H.R. 1234)",
      },
      publicLawNumber: {
        type: "string",
        description: "Public law number when enacted",
      },
      title: {
        type: "string",
        description: "Official title",
      },
      shortTitle: {
        type: "string",
        description: "Popular/short title",
      },
      codification: {
        type: "string",
        description: "Where codified (e.g., 42 U.S.C. § 1234)",
      },

      // ============ Legislature ============
      legislature: {
        type: "string",
        description: "Legislative body",
      },
      session: {
        type: "string",
        description: "Legislative session",
      },
      jurisdiction: {
        type: "string",
        description: "Jurisdiction (country, state)",
      },

      // ============ Sponsors ============
      primarySponsor: {
        type: "string",
        description: "Primary sponsor",
      },
      cosponsors: {
        type: "array",
        items: { type: "string" },
        description: "Cosponsors",
      },

      // ============ Dates ============
      introducedDate: {
        type: "string",
        description: "Date introduced",
      },
      passedDate: {
        type: "string",
        description: "Date passed legislature",
      },
      enactedDate: {
        type: "string",
        description: "Date enacted/signed",
      },
      effectiveDate: {
        type: "string",
        description: "Date law takes effect",
      },

      // ============ Status ============
      status: {
        type: "string",
        enum: [
          "introduced",
          "committee",
          "passed_house",
          "passed_senate",
          "passed_both",
          "signed",
          "enacted",
          "vetoed",
          "veto_overridden",
          "expired",
          "repealed",
        ],
        description: "Current status",
      },
      repealedBy: {
        type: "string",
        description: "Repealing legislation (if repealed)",
      },

      // ============ Content ============
      summary: {
        type: "string",
        description: "Summary of the legislation",
      },
      subjectAreas: {
        type: "array",
        items: { type: "string" },
        description: "Subject matter areas",
      },
      amends: {
        type: "array",
        items: { type: "string" },
        description: "What this legislation amends",
      },

      // ============ Votes ============
      houseVote: {
        type: "object",
        properties: {
          yea: { type: "number" },
          nay: { type: "number" },
          abstain: { type: "number" },
          date: { type: "string" },
        },
        description: "House/lower chamber vote",
      },
      senateVote: {
        type: "object",
        properties: {
          yea: { type: "number" },
          nay: { type: "number" },
          abstain: { type: "number" },
          date: { type: "string" },
        },
        description: "Senate/upper chamber vote",
      },

      // ============ References ============
      officialUrl: {
        type: "string",
        description: "Official government URL",
      },
      textUrl: {
        type: "string",
        description: "Full text URL",
      },
      congressGovUrl: {
        type: "string",
        description: "Congress.gov URL (for US)",
      },
    },
    required: ["title", "legislature", "jurisdiction", "status"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ Contract Schema ============

/**
 * Contract Schema
 *
 * A legally binding contract or agreement.
 */
export const CONTRACT_SCHEMA: ClaimSchema = {
  id: "contract_v1",
  version: 1,
  name: "Contract",
  description: "A legally binding contract or agreement",
  domain: "legal.contracts",

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
      // ============ Contract Identification ============
      contractId: {
        type: "string",
        description: "Unique contract identifier",
      },
      contractType: {
        type: "string",
        enum: [
          "sale",
          "service",
          "employment",
          "lease",
          "license",
          "partnership",
          "merger",
          "loan",
          "insurance",
          "settlement",
          "nda",
          "franchise",
          "distribution",
          "agency",
          "other",
        ],
        description: "Type of contract",
      },
      title: {
        type: "string",
        description: "Contract title",
      },

      // ============ Parties ============
      parties: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            role: { type: "string" }, // buyer, seller, lessor, lessee, etc.
            jurisdiction: { type: "string" },
            signatoryName: { type: "string" },
            signatoryTitle: { type: "string" },
          },
        },
        description: "Contracting parties",
      },

      // ============ Dates ============
      executionDate: {
        type: "string",
        description: "Date contract was signed",
      },
      effectiveDate: {
        type: "string",
        description: "Date contract takes effect",
      },
      terminationDate: {
        type: "string",
        description: "Contract termination/expiration date",
      },

      // ============ Governing Law ============
      governingLaw: {
        type: "string",
        description: "Governing law jurisdiction",
      },
      disputeResolution: {
        type: "string",
        enum: ["litigation", "arbitration", "mediation", "choice"],
        description: "Dispute resolution mechanism",
      },
      venue: {
        type: "string",
        description: "Venue for disputes",
      },

      // ============ Value ============
      totalValue: {
        type: "number",
        description: "Total contract value",
      },
      currency: {
        type: "string",
        description: "Currency of value",
      },
      paymentTerms: {
        type: "string",
        description: "Payment terms summary",
      },

      // ============ Key Terms ============
      keyTerms: {
        type: "array",
        items: {
          type: "object",
          properties: {
            termName: { type: "string" },
            termDescription: { type: "string" },
            section: { type: "string" },
          },
        },
        description: "Key contractual terms",
      },

      // ============ Status ============
      status: {
        type: "string",
        enum: [
          "negotiating",
          "executed",
          "active",
          "terminated",
          "expired",
          "breached",
          "disputed",
        ],
        description: "Contract status",
      },
      amendments: {
        type: "array",
        items: {
          type: "object",
          properties: {
            date: { type: "string" },
            description: { type: "string" },
          },
        },
        description: "Contract amendments",
      },

      // ============ Confidentiality ============
      isConfidential: {
        type: "boolean",
        description: "Is contract confidential?",
      },
      publicDisclosure: {
        type: "string",
        description: "Where publicly disclosed (if applicable)",
      },

      // ============ Verification ============
      contentHash: {
        type: "string",
        description: "Hash of full contract text",
      },
      notarized: {
        type: "boolean",
        description: "Is contract notarized?",
      },
      filedWith: {
        type: "string",
        description: "Where filed (if publicly recorded)",
      },
    },
    required: ["contractType", "parties", "executionDate", "status"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["attestation"],
  minReferences: 1,

  active: true,
};

// ============ Legal Opinion Schema ============

/**
 * Legal Opinion Schema
 *
 * A formal legal opinion from counsel.
 */
export const LEGAL_OPINION_SCHEMA: ClaimSchema = {
  id: "legal_opinion_v1",
  version: 1,
  name: "Legal Opinion",
  description: "A formal legal opinion from counsel",
  domain: "legal.opinions",

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
      // ============ Opinion Details ============
      opinionType: {
        type: "string",
        enum: [
          "securities",
          "tax",
          "corporate",
          "regulatory",
          "litigation",
          "transaction",
          "compliance",
          "ip",
          "general",
        ],
        description: "Type of legal opinion",
      },
      subject: {
        type: "string",
        description: "Subject matter of opinion",
      },
      question: {
        type: "string",
        description: "Legal question addressed",
      },
      conclusion: {
        type: "string",
        description: "Opinion conclusion",
      },

      // ============ Counsel ============
      lawFirm: {
        type: "string",
        description: "Issuing law firm",
      },
      attorneys: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            barNumber: { type: "string" },
            jurisdiction: { type: "string" },
          },
        },
        description: "Attorneys signing the opinion",
      },

      // ============ Client ============
      client: {
        type: "string",
        description: "Client name (if disclosed)",
      },
      addressedTo: {
        type: "array",
        items: { type: "string" },
        description: "Parties addressed",
      },

      // ============ Opinion Content ============
      opinionDate: {
        type: "string",
        description: "Date of opinion",
      },
      qualifications: {
        type: "array",
        items: { type: "string" },
        description: "Qualifications and assumptions",
      },
      limitations: {
        type: "array",
        items: { type: "string" },
        description: "Limitations on the opinion",
      },
      reliedUpon: {
        type: "array",
        items: { type: "string" },
        description: "Documents/facts relied upon",
      },

      // ============ References ============
      publiclyFiled: {
        type: "boolean",
        description: "Is opinion publicly filed?",
      },
      filingReference: {
        type: "string",
        description: "SEC filing, etc. where attached",
      },
      contentHash: {
        type: "string",
        description: "Hash of opinion content",
      },
    },
    required: [
      "opinionType",
      "subject",
      "conclusion",
      "lawFirm",
      "opinionDate",
    ],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["attestation"],
  minReferences: 1,

  active: true,
};

// ============ Exports ============

export const LEGAL_SCHEMAS = [
  COURT_CASE_SCHEMA,
  LEGISLATION_SCHEMA,
  CONTRACT_SCHEMA,
  LEGAL_OPINION_SCHEMA,
];
