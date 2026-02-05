/**
 * Financial Source Schemas
 *
 * Schemas for financial reports, SEC filings, and economic data.
 *
 * Financial sources are highly structured:
 * - GAAP/IFRS standardized reporting
 * - SEC/regulatory filing requirements
 * - Audit attestations
 * - Forward-looking statement disclaimers
 */

import type { ClaimSchema } from "../types.js";

// ============ Financial Report Schema ============

/**
 * Financial Report Schema
 *
 * A quarterly or annual financial report.
 */
export const FINANCIAL_REPORT_SCHEMA: ClaimSchema = {
  id: "financial_report_v1",
  version: 1,
  name: "Financial Report",
  description: "A quarterly or annual financial report",
  domain: "financial.reports",

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
      // ============ Company Identification ============
      companyName: {
        type: "string",
        description: "Company legal name",
      },
      ticker: {
        type: "string",
        description: "Stock ticker symbol",
      },
      cik: {
        type: "string",
        description: "SEC Central Index Key",
      },
      exchange: {
        type: "string",
        description: "Stock exchange",
      },
      industry: {
        type: "string",
        description: "Industry classification",
      },
      sicCode: {
        type: "string",
        description: "SIC code",
      },

      // ============ Report Period ============
      reportType: {
        type: "string",
        enum: [
          "10-K",
          "10-Q",
          "20-F",
          "8-K",
          "annual_report",
          "quarterly_report",
          "earnings_release",
          "other",
        ],
        description: "Type of report",
      },
      fiscalYear: {
        type: "number",
        description: "Fiscal year",
      },
      fiscalQuarter: {
        type: "number",
        description: "Fiscal quarter (1-4)",
      },
      periodStart: {
        type: "string",
        description: "Period start date",
      },
      periodEnd: {
        type: "string",
        description: "Period end date",
      },
      filingDate: {
        type: "string",
        description: "SEC filing date",
      },

      // ============ Income Statement ============
      incomeStatement: {
        type: "object",
        properties: {
          revenue: { type: "number" },
          costOfRevenue: { type: "number" },
          grossProfit: { type: "number" },
          operatingExpenses: { type: "number" },
          operatingIncome: { type: "number" },
          interestExpense: { type: "number" },
          preTaxIncome: { type: "number" },
          incomeTaxExpense: { type: "number" },
          netIncome: { type: "number" },
          eps: { type: "number" },
          epsDiluted: { type: "number" },
          sharesOutstanding: { type: "number" },
          sharesDiluted: { type: "number" },
        },
        description: "Income statement metrics",
      },

      // ============ Balance Sheet ============
      balanceSheet: {
        type: "object",
        properties: {
          totalAssets: { type: "number" },
          currentAssets: { type: "number" },
          cash: { type: "number" },
          inventory: { type: "number" },
          receivables: { type: "number" },
          totalLiabilities: { type: "number" },
          currentLiabilities: { type: "number" },
          longTermDebt: { type: "number" },
          shareholdersEquity: { type: "number" },
          retainedEarnings: { type: "number" },
        },
        description: "Balance sheet metrics",
      },

      // ============ Cash Flow ============
      cashFlow: {
        type: "object",
        properties: {
          operatingCashFlow: { type: "number" },
          investingCashFlow: { type: "number" },
          financingCashFlow: { type: "number" },
          netChangeInCash: { type: "number" },
          freeCashFlow: { type: "number" },
          capex: { type: "number" },
          dividendsPaid: { type: "number" },
          shareRepurchases: { type: "number" },
        },
        description: "Cash flow statement metrics",
      },

      // ============ Currency & Standards ============
      currency: {
        type: "string",
        description: "Reporting currency (ISO 4217)",
      },
      accountingStandard: {
        type: "string",
        enum: ["us_gaap", "ifrs", "other"],
        description: "Accounting standard used",
      },
      unitMultiplier: {
        type: "number",
        description: "Unit multiplier (1000 = thousands, 1000000 = millions)",
      },

      // ============ Audit ============
      auditor: {
        type: "string",
        description: "Audit firm",
      },
      auditOpinion: {
        type: "string",
        enum: [
          "unqualified",
          "qualified",
          "adverse",
          "disclaimer",
          "going_concern",
        ],
        description: "Audit opinion type",
      },
      auditDate: {
        type: "string",
        description: "Date of audit opinion",
      },

      // ============ Reference ============
      secFilingUrl: {
        type: "string",
        description: "SEC EDGAR URL",
      },
      accessionNumber: {
        type: "string",
        description: "SEC accession number",
      },
    },
    required: [
      "companyName",
      "reportType",
      "periodEnd",
      "filingDate",
      "currency",
    ],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ SEC Filing Schema ============

/**
 * SEC Filing Schema
 *
 * Any SEC filing document.
 */
export const SEC_FILING_SCHEMA: ClaimSchema = {
  id: "sec_filing_v1",
  version: 1,
  name: "SEC Filing",
  description: "An SEC filing document",
  domain: "financial.sec",

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
      // ============ Filing Identification ============
      accessionNumber: {
        type: "string",
        description: "SEC accession number",
      },
      formType: {
        type: "string",
        description: "SEC form type (10-K, 10-Q, 8-K, S-1, etc.)",
      },
      filingDate: {
        type: "string",
        description: "Date filed with SEC",
      },
      acceptanceDate: {
        type: "string",
        description: "Date accepted by SEC",
      },

      // ============ Filer ============
      filerName: {
        type: "string",
        description: "Filing entity name",
      },
      cik: {
        type: "string",
        description: "Central Index Key",
      },
      filerType: {
        type: "string",
        enum: ["company", "individual", "fund", "trust", "other"],
        description: "Type of filer",
      },
      ticker: {
        type: "string",
        description: "Trading symbol",
      },

      // ============ Filing Content ============
      description: {
        type: "string",
        description: "Filing description",
      },
      items: {
        type: "array",
        items: { type: "string" },
        description: "Items reported (for 8-K)",
      },
      exhibits: {
        type: "array",
        items: {
          type: "object",
          properties: {
            exhibitNumber: { type: "string" },
            description: { type: "string" },
            url: { type: "string" },
          },
        },
        description: "Attached exhibits",
      },

      // ============ Period ============
      periodOfReport: {
        type: "string",
        description: "Period the filing covers",
      },
      fiscalYearEnd: {
        type: "string",
        description: "Fiscal year end date",
      },

      // ============ Amendments ============
      isAmendment: {
        type: "boolean",
        description: "Is this an amendment?",
      },
      amendsAccession: {
        type: "string",
        description: "Original filing accession (if amendment)",
      },
      amendmentReason: {
        type: "string",
        description: "Reason for amendment",
      },

      // ============ Reference ============
      edgarUrl: {
        type: "string",
        description: "EDGAR filing URL",
      },
      documentUrls: {
        type: "array",
        items: { type: "string" },
        description: "Document URLs within filing",
      },
    },
    required: ["accessionNumber", "formType", "filingDate", "filerName", "cik"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ Insider Transaction Schema ============

/**
 * Insider Transaction Schema
 *
 * A Form 4 insider trading report.
 */
export const INSIDER_TRANSACTION_SCHEMA: ClaimSchema = {
  id: "insider_transaction_v1",
  version: 1,
  name: "Insider Transaction",
  description: "A Form 4 insider trading report",
  domain: "financial.insider",

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
      // ============ Filing ============
      accessionNumber: {
        type: "string",
        description: "SEC accession number",
      },
      filingDate: {
        type: "string",
        description: "Filing date",
      },

      // ============ Company ============
      issuerName: {
        type: "string",
        description: "Issuing company name",
      },
      issuerCik: {
        type: "string",
        description: "Issuer CIK",
      },
      issuerTicker: {
        type: "string",
        description: "Issuer ticker",
      },

      // ============ Insider ============
      reportingPersonName: {
        type: "string",
        description: "Reporting person name",
      },
      reportingPersonCik: {
        type: "string",
        description: "Reporting person CIK",
      },
      relationship: {
        type: "object",
        properties: {
          isDirector: { type: "boolean" },
          isOfficer: { type: "boolean" },
          isTenPercentOwner: { type: "boolean" },
          isOther: { type: "boolean" },
          officerTitle: { type: "string" },
        },
        description: "Relationship to company",
      },

      // ============ Transactions ============
      transactions: {
        type: "array",
        items: {
          type: "object",
          properties: {
            transactionDate: { type: "string" },
            transactionCode: { type: "string" }, // P, S, A, D, etc.
            securityTitle: { type: "string" },
            shares: { type: "number" },
            pricePerShare: { type: "number" },
            totalValue: { type: "number" },
            acquiredDisposed: { type: "string" }, // A or D
            ownership: { type: "string" }, // D (direct) or I (indirect)
          },
        },
        description: "Transaction details",
      },

      // ============ Holdings ============
      postTransactionShares: {
        type: "number",
        description: "Shares owned after transaction",
      },

      // ============ Reference ============
      edgarUrl: {
        type: "string",
        description: "EDGAR filing URL",
      },
    },
    required: [
      "accessionNumber",
      "filingDate",
      "issuerName",
      "reportingPersonName",
      "transactions",
    ],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ Financial Metric Schema ============

/**
 * Financial Metric Schema
 *
 * A specific financial metric claim.
 */
export const FINANCIAL_METRIC_SCHEMA: ClaimSchema = {
  id: "financial_metric_v1",
  version: 1,
  name: "Financial Metric",
  description: "A specific financial metric",
  domain: "financial.metrics",

  timing: {
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 4,
    totalRoundBlocks: 12,
    minAnalysisTimeMs: 5000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Entity ============
      entityName: {
        type: "string",
        description: "Entity name",
      },
      entityType: {
        type: "string",
        enum: [
          "company",
          "fund",
          "government",
          "index",
          "commodity",
          "currency",
          "other",
        ],
        description: "Type of entity",
      },
      ticker: {
        type: "string",
        description: "Ticker symbol",
      },

      // ============ Metric ============
      metricName: {
        type: "string",
        description: "Name of the metric",
      },
      metricCategory: {
        type: "string",
        enum: [
          "revenue",
          "profitability",
          "liquidity",
          "leverage",
          "valuation",
          "efficiency",
          "growth",
          "market",
          "credit",
          "other",
        ],
        description: "Metric category",
      },
      value: {
        type: "number",
        description: "Metric value",
      },
      unit: {
        type: "string",
        description: "Unit of measurement",
      },
      currency: {
        type: "string",
        description: "Currency (if monetary)",
      },

      // ============ Period ============
      asOfDate: {
        type: "string",
        description: "Date the metric applies to",
      },
      periodType: {
        type: "string",
        enum: ["point_in_time", "quarterly", "annual", "ttm", "ytd", "mtd"],
        description: "Period type",
      },
      periodStart: {
        type: "string",
        description: "Period start (if range)",
      },
      periodEnd: {
        type: "string",
        description: "Period end (if range)",
      },

      // ============ Comparison ============
      priorPeriodValue: {
        type: "number",
        description: "Prior period value",
      },
      changePercent: {
        type: "number",
        description: "Percentage change",
      },
      changeAbsolute: {
        type: "number",
        description: "Absolute change",
      },

      // ============ Source ============
      sourceType: {
        type: "string",
        enum: [
          "sec_filing",
          "earnings_call",
          "press_release",
          "data_provider",
          "company_website",
          "other",
        ],
        description: "Source type",
      },
      sourceReference: {
        type: "string",
        description: "Source reference (accession number, URL, etc.)",
      },
    },
    required: ["entityName", "metricName", "value", "asOfDate", "sourceType"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["institutional"],
  minReferences: 1,

  active: true,
};

// ============ Analyst Rating Schema ============

/**
 * Analyst Rating Schema
 *
 * A stock analyst rating or price target.
 */
export const ANALYST_RATING_SCHEMA: ClaimSchema = {
  id: "analyst_rating_v1",
  version: 1,
  name: "Analyst Rating",
  description: "A stock analyst rating or price target",
  domain: "financial.ratings",

  timing: {
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 4,
    totalRoundBlocks: 12,
    minAnalysisTimeMs: 5000,
  },

  verificationMethod: "reference",

  dataSchema: {
    type: "object",
    properties: {
      // ============ Target Company ============
      companyName: {
        type: "string",
        description: "Target company name",
      },
      ticker: {
        type: "string",
        description: "Ticker symbol",
      },

      // ============ Analyst ============
      analystName: {
        type: "string",
        description: "Analyst name",
      },
      firmName: {
        type: "string",
        description: "Research firm name",
      },

      // ============ Rating ============
      ratingDate: {
        type: "string",
        description: "Date of rating",
      },
      rating: {
        type: "string",
        description: "Rating (Buy, Hold, Sell, etc.)",
      },
      ratingNormalized: {
        type: "string",
        enum: ["strong_buy", "buy", "hold", "sell", "strong_sell"],
        description: "Normalized rating",
      },
      priorRating: {
        type: "string",
        description: "Previous rating",
      },
      ratingChange: {
        type: "string",
        enum: ["upgrade", "downgrade", "maintain", "initiate", "drop"],
        description: "Rating change type",
      },

      // ============ Price Target ============
      priceTarget: {
        type: "number",
        description: "Price target",
      },
      priorPriceTarget: {
        type: "number",
        description: "Previous price target",
      },
      currency: {
        type: "string",
        description: "Price target currency",
      },
      timeHorizon: {
        type: "string",
        description: "Target time horizon",
      },

      // ============ Context ============
      currentPrice: {
        type: "number",
        description: "Stock price at time of rating",
      },
      impliedUpside: {
        type: "number",
        description: "Implied upside percentage",
      },

      // ============ Source ============
      reportTitle: {
        type: "string",
        description: "Research report title",
      },
      sourceUrl: {
        type: "string",
        description: "Source URL",
      },
    },
    required: ["companyName", "ticker", "firmName", "ratingDate", "rating"],
    additionalProperties: false,
  },

  requiredReferenceTypes: ["archived_web"],
  minReferences: 1,

  active: true,
};

// ============ Exports ============

export const FINANCIAL_SCHEMAS = [
  FINANCIAL_REPORT_SCHEMA,
  SEC_FILING_SCHEMA,
  INSIDER_TRANSACTION_SCHEMA,
  FINANCIAL_METRIC_SCHEMA,
  ANALYST_RATING_SCHEMA,
];
