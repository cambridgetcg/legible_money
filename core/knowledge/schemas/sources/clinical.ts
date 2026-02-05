/**
 * Clinical Source Schemas
 *
 * Schemas for clinical trials, medical studies, and healthcare data.
 *
 * Clinical sources are highly regulated and require:
 * - IRB/Ethics approval documentation
 * - Trial registration (ClinicalTrials.gov, etc.)
 * - Strict outcome reporting
 * - Adverse event tracking
 */

import type { ClaimSchema } from "../types.js";

// ============ Clinical Trial Schema ============

/**
 * Clinical Trial Schema
 *
 * A registered clinical trial with structured outcomes.
 */
export const CLINICAL_TRIAL_SCHEMA: ClaimSchema = {
  id: "clinical_trial_v1",
  version: 1,
  name: "Clinical Trial",
  description: "A registered clinical trial",
  domain: "clinical.trials",

  timing: {
    commitPhaseBlocks: 12,
    revealPhaseBlocks: 8,
    totalRoundBlocks: 22,
    minAnalysisTimeMs: 20000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Trial Identification ============
      registrationId: {
        type: "string",
        description: "Trial registration ID (e.g., NCT12345678)",
      },
      registrySource: {
        type: "string",
        enum: [
          "clinicaltrials_gov",
          "eudract",
          "isrctn",
          "anzctr",
          "ctri",
          "chictr",
          "jprn",
          "who_ictrp",
          "other",
        ],
        description: "Registry where trial is registered",
      },
      secondaryIds: {
        type: "array",
        items: { type: "string" },
        description: "Secondary registration IDs",
      },
      officialTitle: {
        type: "string",
        description: "Official trial title",
      },
      briefTitle: {
        type: "string",
        description: "Brief/common title",
      },
      acronym: {
        type: "string",
        description: "Trial acronym",
      },

      // ============ Sponsor & Collaborators ============
      sponsor: {
        type: "string",
        description: "Primary sponsor",
      },
      sponsorType: {
        type: "string",
        enum: ["industry", "academic", "government", "nonprofit", "other"],
        description: "Type of sponsor",
      },
      collaborators: {
        type: "array",
        items: { type: "string" },
        description: "Collaborating organizations",
      },
      principalInvestigator: {
        type: "object",
        properties: {
          name: { type: "string" },
          affiliation: { type: "string" },
        },
        description: "Principal investigator",
      },

      // ============ Study Design ============
      phase: {
        type: "string",
        enum: [
          "early_phase_1",
          "phase_1",
          "phase_1_2",
          "phase_2",
          "phase_2_3",
          "phase_3",
          "phase_4",
          "not_applicable",
        ],
        description: "Trial phase",
      },
      studyType: {
        type: "string",
        enum: [
          "interventional",
          "observational",
          "expanded_access",
          "registry",
        ],
        description: "Study type",
      },
      design: {
        type: "object",
        properties: {
          allocation: { type: "string" }, // randomized, non-randomized
          interventionModel: { type: "string" }, // parallel, crossover, factorial
          masking: { type: "string" }, // none, single, double, triple, quadruple
          primaryPurpose: { type: "string" }, // treatment, prevention, diagnostic
        },
        description: "Study design details",
      },

      // ============ Intervention ============
      interventionType: {
        type: "string",
        enum: [
          "drug",
          "biological",
          "device",
          "procedure",
          "radiation",
          "behavioral",
          "genetic",
          "dietary",
          "combination",
          "other",
        ],
        description: "Type of intervention",
      },
      interventions: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            type: { type: "string" },
            description: { type: "string" },
            armGroupLabels: { type: "array", items: { type: "string" } },
          },
        },
        description: "Interventions being studied",
      },

      // ============ Condition ============
      conditions: {
        type: "array",
        items: { type: "string" },
        description: "Conditions being studied",
      },
      meshTerms: {
        type: "array",
        items: { type: "string" },
        description: "MeSH terms for conditions",
      },

      // ============ Eligibility ============
      eligibilityCriteria: {
        type: "object",
        properties: {
          ageMin: { type: "number" },
          ageMax: { type: "number" },
          sex: { type: "string" },
          healthyVolunteers: { type: "boolean" },
          inclusionCriteria: { type: "array", items: { type: "string" } },
          exclusionCriteria: { type: "array", items: { type: "string" } },
        },
        description: "Eligibility criteria",
      },

      // ============ Enrollment ============
      enrollmentTarget: {
        type: "number",
        description: "Target enrollment",
      },
      enrollmentActual: {
        type: "number",
        description: "Actual enrollment",
      },
      enrollmentType: {
        type: "string",
        enum: ["anticipated", "actual"],
        description: "Enrollment type",
      },

      // ============ Dates ============
      startDate: {
        type: "string",
        description: "Study start date",
      },
      completionDate: {
        type: "string",
        description: "Primary completion date",
      },
      studyCompletionDate: {
        type: "string",
        description: "Study completion date",
      },
      firstPostedDate: {
        type: "string",
        description: "First posted on registry",
      },
      lastUpdateDate: {
        type: "string",
        description: "Last update posted",
      },

      // ============ Status ============
      overallStatus: {
        type: "string",
        enum: [
          "not_yet_recruiting",
          "recruiting",
          "enrolling_by_invitation",
          "active_not_recruiting",
          "suspended",
          "terminated",
          "completed",
          "withdrawn",
          "unknown",
        ],
        description: "Overall recruitment status",
      },
      whyStopped: {
        type: "string",
        description: "Reason if stopped early",
      },

      // ============ Outcomes ============
      primaryOutcomes: {
        type: "array",
        items: {
          type: "object",
          properties: {
            measure: { type: "string" },
            description: { type: "string" },
            timeFrame: { type: "string" },
          },
        },
        description: "Primary outcome measures",
      },
      secondaryOutcomes: {
        type: "array",
        items: {
          type: "object",
          properties: {
            measure: { type: "string" },
            description: { type: "string" },
            timeFrame: { type: "string" },
          },
        },
        description: "Secondary outcome measures",
      },

      // ============ Ethics ============
      ethicsApproval: {
        type: "object",
        properties: {
          hasApproval: { type: "boolean" },
          irbName: { type: "string" },
          approvalNumber: { type: "string" },
        },
        description: "IRB/Ethics approval",
      },
      informedConsent: {
        type: "boolean",
        description: "Requires informed consent?",
      },

      // ============ References ============
      registryUrl: {
        type: "string",
        description: "Registry URL",
      },
      publicationRefs: {
        type: "array",
        items: { type: "string" },
        description: "Publication DOIs/PMIDs",
      },
    },
    required: [
      "registrationId",
      "registrySource",
      "officialTitle",
      "sponsor",
      "phase",
      "studyType",
      "overallStatus",
    ],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ Clinical Trial Result Schema ============

/**
 * Clinical Trial Result Schema
 *
 * Results from a completed clinical trial.
 */
export const CLINICAL_TRIAL_RESULT_SCHEMA: ClaimSchema = {
  id: "clinical_trial_result_v1",
  version: 1,
  name: "Clinical Trial Result",
  description: "Results from a clinical trial",
  domain: "clinical.results",

  timing: {
    commitPhaseBlocks: 14,
    revealPhaseBlocks: 10,
    totalRoundBlocks: 26,
    minAnalysisTimeMs: 25000,
  },

  verificationMethod: "consensus",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Trial Reference ============
      registrationId: {
        type: "string",
        description: "Trial registration ID",
      },
      resultsPostedDate: {
        type: "string",
        description: "Date results were posted",
      },

      // ============ Participant Flow ============
      participantFlow: {
        type: "object",
        properties: {
          enrolled: { type: "number" },
          started: { type: "number" },
          completed: { type: "number" },
          withdrawn: { type: "number" },
          byArm: {
            type: "array",
            items: {
              type: "object",
              properties: {
                armName: { type: "string" },
                started: { type: "number" },
                completed: { type: "number" },
              },
            },
          },
        },
        description: "Participant flow through trial",
      },

      // ============ Baseline Characteristics ============
      baselineCharacteristics: {
        type: "object",
        properties: {
          totalParticipants: { type: "number" },
          meanAge: { type: "number" },
          sexDistribution: {
            type: "object",
            properties: {
              male: { type: "number" },
              female: { type: "number" },
            },
          },
          byArm: { type: "array", items: { type: "object" } },
        },
        description: "Baseline demographics",
      },

      // ============ Primary Outcomes ============
      primaryOutcomeResults: {
        type: "array",
        items: {
          type: "object",
          properties: {
            outcomeMeasure: { type: "string" },
            analysisPopulation: { type: "string" },
            statisticalMethod: { type: "string" },
            parameterType: { type: "string" }, // mean, median, proportion, etc.
            results: {
              type: "array",
              items: {
                type: "object",
                properties: {
                  arm: { type: "string" },
                  participants: { type: "number" },
                  value: { type: "number" },
                  lowerLimit: { type: "number" },
                  upperLimit: { type: "number" },
                  pValue: { type: "number" },
                },
              },
            },
          },
        },
        description: "Primary outcome results",
      },

      // ============ Secondary Outcomes ============
      secondaryOutcomeResults: {
        type: "array",
        items: {
          type: "object",
          properties: {
            outcomeMeasure: { type: "string" },
            results: { type: "array", items: { type: "object" } },
          },
        },
        description: "Secondary outcome results",
      },

      // ============ Adverse Events ============
      adverseEvents: {
        type: "object",
        properties: {
          reportingDescription: { type: "string" },
          timeFrame: { type: "string" },
          frequencyThreshold: { type: "number" },
          seriousEvents: {
            type: "array",
            items: {
              type: "object",
              properties: {
                term: { type: "string" },
                organSystem: { type: "string" },
                byArm: {
                  type: "array",
                  items: {
                    type: "object",
                    properties: {
                      arm: { type: "string" },
                      affected: { type: "number" },
                      atRisk: { type: "number" },
                    },
                  },
                },
              },
            },
          },
          otherEvents: {
            type: "array",
            items: {
              type: "object",
              properties: {
                term: { type: "string" },
                organSystem: { type: "string" },
                byArm: { type: "array", items: { type: "object" } },
              },
            },
          },
          totalSerious: { type: "number" },
          totalDeaths: { type: "number" },
        },
        description: "Adverse events summary",
      },

      // ============ Analysis ============
      limitations: {
        type: "string",
        description: "Study limitations",
      },
      conclusions: {
        type: "string",
        description: "Investigator conclusions",
      },

      // ============ Publication ============
      publicationDoi: {
        type: "string",
        description: "Publication DOI",
      },
      publicationPmid: {
        type: "string",
        description: "PubMed ID",
      },
    },
    required: ["registrationId", "participantFlow", "primaryOutcomeResults"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional", "doi"],
  minReferences: 1,

  active: true,
};

// ============ Adverse Event Schema ============

/**
 * Adverse Event Report Schema
 *
 * An individual adverse event report.
 */
export const ADVERSE_EVENT_SCHEMA: ClaimSchema = {
  id: "adverse_event_v1",
  version: 1,
  name: "Adverse Event Report",
  description: "An adverse event report",
  domain: "clinical.adverse_events",

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
      // ============ Report Identification ============
      reportId: {
        type: "string",
        description: "Adverse event report ID (e.g., FAERS case number)",
      },
      reportSource: {
        type: "string",
        enum: [
          "faers",
          "vaers",
          "eudravigilance",
          "mhra",
          "manufacturer",
          "literature",
          "other",
        ],
        description: "Reporting system source",
      },
      reportType: {
        type: "string",
        enum: ["initial", "followup", "final"],
        description: "Report type",
      },

      // ============ Event Details ============
      eventTerms: {
        type: "array",
        items: { type: "string" },
        description: "MedDRA preferred terms",
      },
      eventDescription: {
        type: "string",
        description: "Narrative description",
      },
      onsetDate: {
        type: "string",
        description: "Event onset date",
      },
      resolutionDate: {
        type: "string",
        description: "Event resolution date",
      },

      // ============ Seriousness ============
      serious: {
        type: "boolean",
        description: "Is this a serious adverse event?",
      },
      seriousnessCriteria: {
        type: "array",
        items: {
          type: "string",
          enum: [
            "death",
            "life_threatening",
            "hospitalization",
            "disability",
            "congenital_anomaly",
            "medically_important",
          ],
        },
        description: "Seriousness criteria met",
      },
      outcome: {
        type: "string",
        enum: ["recovered", "recovering", "not_recovered", "fatal", "unknown"],
        description: "Patient outcome",
      },

      // ============ Suspect Product ============
      suspectProducts: {
        type: "array",
        items: {
          type: "object",
          properties: {
            productName: { type: "string" },
            genericName: { type: "string" },
            manufacturer: { type: "string" },
            ndc: { type: "string" },
            lotNumber: { type: "string" },
            doseInfo: { type: "string" },
            indication: { type: "string" },
            startDate: { type: "string" },
            stopDate: { type: "string" },
            dechallenge: { type: "string" }, // positive, negative, unknown, not done
            rechallenge: { type: "string" },
          },
        },
        description: "Suspect products",
      },

      // ============ Patient ============
      patientAge: {
        type: "number",
        description: "Patient age at event",
      },
      patientSex: {
        type: "string",
        enum: ["male", "female", "unknown"],
        description: "Patient sex",
      },
      patientWeight: {
        type: "number",
        description: "Patient weight (kg)",
      },
      medicalHistory: {
        type: "array",
        items: { type: "string" },
        description: "Relevant medical history",
      },
      concomitantMedications: {
        type: "array",
        items: { type: "string" },
        description: "Other medications being taken",
      },

      // ============ Reporter ============
      reporterType: {
        type: "string",
        enum: ["healthcare_professional", "consumer", "manufacturer", "other"],
        description: "Type of reporter",
      },
      reportDate: {
        type: "string",
        description: "Date report was submitted",
      },

      // ============ Causality ============
      reporterCausality: {
        type: "string",
        enum: [
          "certain",
          "probable",
          "possible",
          "unlikely",
          "conditional",
          "unassessable",
        ],
        description: "Reporter causality assessment",
      },

      // ============ Reference ============
      sourceUrl: {
        type: "string",
        description: "URL to source database record",
      },
    },
    required: [
      "reportId",
      "reportSource",
      "eventTerms",
      "serious",
      "suspectProducts",
    ],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ Exports ============

export const CLINICAL_SCHEMAS = [
  CLINICAL_TRIAL_SCHEMA,
  CLINICAL_TRIAL_RESULT_SCHEMA,
  ADVERSE_EVENT_SCHEMA,
];
