/**
 * Experimental Data Source Schemas
 *
 * Schemas for raw data, datasets, and experimental results.
 *
 * Data sources are critical for:
 * - Reproducibility verification
 * - Replication studies
 * - Meta-analyses
 * - Fraud detection
 */

import type { ClaimSchema } from "../types.js";

// ============ Dataset Schema ============

/**
 * Dataset Schema
 *
 * A published dataset with metadata for verification.
 */
export const DATASET_SCHEMA: ClaimSchema = {
  id: "dataset_v1",
  version: 1,
  name: "Dataset",
  description: "A published dataset with verification metadata",
  domain: "data.datasets",

  timing: {
    commitPhaseBlocks: 10,
    revealPhaseBlocks: 8,
    totalRoundBlocks: 20,
    minAnalysisTimeMs: 12000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Identification ============
      doi: {
        type: "string",
        description: "Dataset DOI",
      },
      repositoryId: {
        type: "string",
        description: "Repository-specific identifier",
      },
      repository: {
        type: "string",
        enum: [
          "zenodo",
          "figshare",
          "dryad",
          "dataverse",
          "osf",
          "github",
          "kaggle",
          "huggingface",
          "aws_open_data",
          "google_dataset_search",
          "institutional",
          "other",
        ],
        description: "Data repository",
      },
      url: {
        type: "string",
        description: "Dataset URL",
      },

      // ============ Metadata ============
      title: {
        type: "string",
        description: "Dataset title",
      },
      description: {
        type: "string",
        description: "Dataset description",
      },
      version: {
        type: "string",
        description: "Dataset version",
      },
      publishedDate: {
        type: "string",
        description: "Publication date",
      },
      lastUpdated: {
        type: "string",
        description: "Last update date",
      },

      // ============ Creators ============
      creators: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            orcid: { type: "string" },
            affiliation: { type: "string" },
          },
        },
        description: "Dataset creators",
      },
      fundingSource: {
        type: "array",
        items: { type: "string" },
        description: "Funding sources",
      },

      // ============ Content ============
      dataType: {
        type: "string",
        enum: [
          "tabular",
          "images",
          "video",
          "audio",
          "text",
          "time_series",
          "geospatial",
          "genomic",
          "simulation",
          "sensor",
          "mixed",
        ],
        description: "Type of data",
      },
      fileFormats: {
        type: "array",
        items: { type: "string" },
        description: "File formats (csv, parquet, json, etc.)",
      },
      totalSize: {
        type: "string",
        description: 'Total size (e.g., "2.5 GB")',
      },
      recordCount: {
        type: "number",
        description: "Number of records/samples",
      },
      variableCount: {
        type: "number",
        description: "Number of variables/features",
      },

      // ============ Verification ============
      checksumAlgorithm: {
        type: "string",
        enum: ["md5", "sha256", "sha512", "blake3"],
        description: "Checksum algorithm used",
      },
      checksum: {
        type: "string",
        description: "Dataset checksum",
      },
      fileChecksums: {
        type: "array",
        items: {
          type: "object",
          properties: {
            filename: { type: "string" },
            checksum: { type: "string" },
            size: { type: "number" },
          },
        },
        description: "Individual file checksums",
      },

      // ============ Provenance ============
      collectionMethod: {
        type: "string",
        description: "How data was collected",
      },
      collectionPeriod: {
        type: "object",
        properties: {
          start: { type: "string" },
          end: { type: "string" },
        },
        description: "Data collection period",
      },
      instruments: {
        type: "array",
        items: { type: "string" },
        description: "Instruments/tools used for collection",
      },
      processingSteps: {
        type: "array",
        items: { type: "string" },
        description: "Processing steps applied",
      },
      qualityControl: {
        type: "string",
        description: "Quality control measures",
      },

      // ============ Licensing ============
      license: {
        type: "string",
        description: "Data license (CC-BY, MIT, etc.)",
      },
      accessLevel: {
        type: "string",
        enum: ["open", "restricted", "embargoed", "proprietary"],
        description: "Access level",
      },
      usageRestrictions: {
        type: "string",
        description: "Any usage restrictions",
      },

      // ============ Related ============
      relatedPapers: {
        type: "array",
        items: { type: "string" },
        description: "DOIs of papers using this data",
      },
      relatedDatasets: {
        type: "array",
        items: { type: "string" },
        description: "Related datasets",
      },
    },
    required: ["title", "repository", "dataType", "checksum"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["computational"],
  minReferences: 1,

  active: true,
};

// ============ Raw Experimental Data Schema ============

/**
 * Raw Experimental Data Schema
 *
 * Raw data from a specific experiment.
 */
export const RAW_EXPERIMENTAL_DATA_SCHEMA: ClaimSchema = {
  id: "raw_experimental_data_v1",
  version: 1,
  name: "Raw Experimental Data",
  description: "Raw data from a specific experiment for verification",
  domain: "data.experimental",

  timing: {
    commitPhaseBlocks: 12,
    revealPhaseBlocks: 10,
    totalRoundBlocks: 24,
    minAnalysisTimeMs: 15000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Experiment Link ============
      experimentId: {
        type: "string",
        description: "Identifier for the experiment",
      },
      paperDoi: {
        type: "string",
        description: "DOI of paper this data supports",
      },
      figureOrTable: {
        type: "string",
        description: "Which figure/table this data appears in",
      },

      // ============ Data Location ============
      dataUrl: {
        type: "string",
        description: "URL to access the data",
      },
      dataHash: {
        type: "string",
        description: "SHA-256 hash of the data file",
      },
      dataFormat: {
        type: "string",
        description: "Data format (CSV, JSON, HDF5, etc.)",
      },
      dataSize: {
        type: "number",
        description: "Data size in bytes",
      },

      // ============ Data Description ============
      variables: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            description: { type: "string" },
            type: { type: "string" }, // numeric, categorical, datetime, etc.
            unit: { type: "string" },
            range: {
              type: "object",
              properties: {
                min: { type: "number" },
                max: { type: "number" },
              },
            },
          },
        },
        description: "Variables in the dataset",
      },
      sampleCount: {
        type: "number",
        description: "Number of samples/observations",
      },

      // ============ Collection ============
      collectionDate: {
        type: "string",
        description: "When data was collected",
      },
      collectedBy: {
        type: "array",
        items: { type: "string" },
        description: "Who collected the data",
      },
      labOrFacility: {
        type: "string",
        description: "Lab or facility where data was collected",
      },

      // ============ Instrument Details ============
      instruments: {
        type: "array",
        items: {
          type: "object",
          properties: {
            name: { type: "string" },
            manufacturer: { type: "string" },
            model: { type: "string" },
            serialNumber: { type: "string" },
            calibrationDate: { type: "string" },
            settings: { type: "object", additionalProperties: true },
          },
        },
        description: "Instruments used for data collection",
      },

      // ============ Processing ============
      isRaw: {
        type: "boolean",
        description: "Is this completely unprocessed data?",
      },
      processingApplied: {
        type: "array",
        items: {
          type: "object",
          properties: {
            step: { type: "string" },
            description: { type: "string" },
            softwareUsed: { type: "string" },
            parameters: { type: "object", additionalProperties: true },
          },
        },
        description: "Processing steps applied",
      },
      processingCodeUrl: {
        type: "string",
        description: "URL to processing code/scripts",
      },
      processingCodeHash: {
        type: "string",
        description: "Hash of processing code",
      },

      // ============ Quality ============
      qualityMetrics: {
        type: "object",
        properties: {
          completeness: { type: "number" }, // Fraction of non-missing values
          outlierCount: { type: "number" },
          duplicateCount: { type: "number" },
          validationPassed: { type: "boolean" },
        },
        description: "Data quality metrics",
      },
      knownIssues: {
        type: "array",
        items: { type: "string" },
        description: "Known issues with the data",
      },

      // ============ Reproducibility ============
      randomSeed: {
        type: "number",
        description: "Random seed used (if applicable)",
      },
      environmentHash: {
        type: "string",
        description:
          "Hash of software environment (Docker image, requirements.txt)",
      },
    },
    required: ["experimentId", "dataHash", "variables", "sampleCount"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["computational"],
  minReferences: 1,

  active: true,
};

// ============ Measurement Schema ============

/**
 * Measurement Schema
 *
 * A specific measurement or observation.
 */
export const MEASUREMENT_SCHEMA: ClaimSchema = {
  id: "measurement_v1",
  version: 1,
  name: "Measurement",
  description: "A specific measurement with uncertainty",
  domain: "data.measurements",

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
      // ============ The Measurement ============
      quantity: {
        type: "string",
        description: 'What was measured (e.g., "temperature", "reaction time")',
      },
      value: {
        type: "number",
        description: "Measured value",
      },
      unit: {
        type: "string",
        description: "Unit of measurement",
      },

      // ============ Uncertainty ============
      uncertainty: {
        type: "number",
        description: "Measurement uncertainty (±)",
      },
      uncertaintyType: {
        type: "string",
        enum: [
          "standard_error",
          "standard_deviation",
          "confidence_interval",
          "95ci",
          "99ci",
          "range",
          "other",
        ],
        description: "Type of uncertainty reported",
      },
      confidenceLevel: {
        type: "number",
        description: "Confidence level for interval (e.g., 0.95)",
      },

      // ============ Context ============
      condition: {
        type: "string",
        description: "Experimental condition",
      },
      sampleId: {
        type: "string",
        description: "Sample identifier",
      },
      replicateNumber: {
        type: "number",
        description: "Replicate number",
      },
      measurementDate: {
        type: "string",
        description: "When measurement was taken",
      },

      // ============ Method ============
      instrument: {
        type: "string",
        description: "Instrument used",
      },
      method: {
        type: "string",
        description: "Measurement method/protocol",
      },
      calibrationStatus: {
        type: "string",
        description: "Instrument calibration status",
      },

      // ============ Source ============
      sourceDataHash: {
        type: "string",
        description: "Hash of source data file",
      },
      locationInSource: {
        type: "string",
        description: "Where in source (row, column, etc.)",
      },
      paperDoi: {
        type: "string",
        description: "Paper DOI if from publication",
      },
    },
    required: ["quantity", "value", "unit"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["computational"],
  minReferences: 1,

  active: true,
};

// ============ Statistical Summary Schema ============

/**
 * Statistical Summary Schema
 *
 * Summary statistics from a dataset.
 */
export const STATISTICAL_SUMMARY_SCHEMA: ClaimSchema = {
  id: "statistical_summary_v1",
  version: 1,
  name: "Statistical Summary",
  description: "Summary statistics from a dataset",
  domain: "data.statistics",

  timing: {
    commitPhaseBlocks: 8,
    revealPhaseBlocks: 6,
    totalRoundBlocks: 16,
    minAnalysisTimeMs: 8000,
  },

  verificationMethod: "automatic", // Can be automatically verified if data is available

  dataSchema: {
    type: "object",
    properties: {
      // ============ Data Source ============
      datasetId: {
        type: "string",
        description: "Source dataset identifier",
      },
      dataHash: {
        type: "string",
        description: "Hash of source data",
      },
      variable: {
        type: "string",
        description: "Variable being summarized",
      },
      subset: {
        type: "string",
        description: "Subset/filter applied (if any)",
      },

      // ============ Sample Info ============
      n: {
        type: "number",
        description: "Sample size",
      },
      nMissing: {
        type: "number",
        description: "Number of missing values",
      },

      // ============ Central Tendency ============
      mean: {
        type: "number",
        description: "Arithmetic mean",
      },
      median: {
        type: "number",
        description: "Median",
      },
      mode: {
        type: "number",
        description: "Mode",
      },
      geometricMean: {
        type: "number",
        description: "Geometric mean",
      },

      // ============ Dispersion ============
      standardDeviation: {
        type: "number",
        description: "Standard deviation",
      },
      variance: {
        type: "number",
        description: "Variance",
      },
      standardError: {
        type: "number",
        description: "Standard error of the mean",
      },
      range: {
        type: "object",
        properties: {
          min: { type: "number" },
          max: { type: "number" },
        },
        description: "Range (min-max)",
      },
      iqr: {
        type: "number",
        description: "Interquartile range",
      },

      // ============ Percentiles ============
      percentiles: {
        type: "object",
        properties: {
          p5: { type: "number" },
          p10: { type: "number" },
          p25: { type: "number" },
          p50: { type: "number" },
          p75: { type: "number" },
          p90: { type: "number" },
          p95: { type: "number" },
        },
        description: "Percentile values",
      },

      // ============ Shape ============
      skewness: {
        type: "number",
        description: "Skewness",
      },
      kurtosis: {
        type: "number",
        description: "Kurtosis",
      },

      // ============ Confidence Interval ============
      confidenceInterval: {
        type: "object",
        properties: {
          lower: { type: "number" },
          upper: { type: "number" },
          level: { type: "number" },
        },
        description: "Confidence interval for mean",
      },

      // ============ Computation ============
      computedWith: {
        type: "string",
        description: "Software used for computation",
      },
      computationCodeHash: {
        type: "string",
        description: "Hash of computation code",
      },
    },
    required: ["dataHash", "variable", "n"],
    additionalProperties: false,
  },

  // Verifier function for automatic verification
  verifier: async (_claimData) => {
    // This would verify the statistics against the source data
    // For now, return manual verification required
    return {
      valid: false,
      confidence: 0,
      reason: "Automatic verification requires access to source data",
    };
  },

  requiredReferenceTypes: ["computational"],
  minReferences: 1,

  active: true,
};

// ============ Exports ============

export const DATA_SCHEMAS = [
  DATASET_SCHEMA,
  RAW_EXPERIMENTAL_DATA_SCHEMA,
  MEASUREMENT_SCHEMA,
  STATISTICAL_SUMMARY_SCHEMA,
];
