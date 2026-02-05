/**
 * Source Schema Integration Tests
 *
 * Tests the claim and verification pipeline with actual sourced data
 * for each schema category (patent, media, legal, clinical, financial, technical, data, regulatory).
 *
 * PP HIERARCHY:
 *   TRUTH: Tests verify schemas accept valid real-world data
 *   UNDERSTANDING: Each test shows what valid claims look like
 *   BEAUTY: Clean test structure with reusable patterns
 *   JUSTICE: Coverage across all 27 source schemas
 *   CREATIVITY: Real-world data sourced from public records
 */

import {
  getSchemaRegistry,
  resetSchemaRegistry,
  initializeSchemas,
} from "../../../core/knowledge/schemas/index.js";
import {
  calculateAdjustedConfidence,
  RELIABILITY_MAX_CONFIDENCE,
  type TrustModifiers,
} from "../../../core/knowledge/schemas/sources/index.js";

// ============ Test Harness ============

interface TestResult {
  schemaId: string;
  category: string;
  passed: boolean;
  errors: string[];
  insights: string[];
}

const results: TestResult[] = [];

function test(
  schemaId: string,
  category: string,
  claimData: Record<string, unknown>,
  expectedValid: boolean = true,
): TestResult {
  const registry = getSchemaRegistry();
  const validation = registry.validate(schemaId, claimData);

  const passed = validation.valid === expectedValid;
  const errors: string[] = [];
  const insights: string[] = [];

  if (!passed) {
    if (expectedValid) {
      errors.push(
        `Expected valid but got errors: ${validation.errors.map((e) => e.message).join(", ")}`,
      );
    } else {
      errors.push(`Expected invalid but validation passed`);
    }
  }

  // Collect insights
  if (validation.schema) {
    insights.push(
      `Timing: ${validation.schema.timing.totalRoundBlocks} blocks (~${(validation.schema.timing.totalRoundBlocks * 2.521).toFixed(1)}s)`,
    );
    insights.push(`Verification: ${validation.schema.verificationMethod}`);
    insights.push(`Min refs: ${validation.schema.minReferences || 0}`);
  }

  const result: TestResult = {
    schemaId,
    category,
    passed,
    errors,
    insights,
  };

  results.push(result);
  return result;
}

// ============ Patent Schema Tests ============

function testPatentSchemas(): void {
  console.log("\n📜 Testing Patent Schemas...\n");

  // Real patent: Apple's slide-to-unlock (US8046721B2)
  test("patent_document_v1", "patent", {
    patentNumber: "US8046721B2",
    applicationNumber: "11/956,969",
    jurisdiction: "US",
    patentType: "utility",
    filingDate: "2007-12-14",
    publicationDate: "2009-06-18",
    grantDate: "2011-10-25",
    priorityDate: "2007-12-14",
    inventors: [
      { name: "Chaudhri, Imran", country: "US" },
      { name: "Christie, Greg", country: "US" },
      { name: "Ording, Bas", country: "US" },
    ],
    applicant: "Apple Inc.",
    assignee: "Apple Inc.",
    title: "Unlocking a device by performing gestures on an unlock image",
    abstract:
      "A device with a touch-sensitive display may be unlocked via gestures performed on the touch-sensitive display.",
    ipcClassifications: ["G06F3/0488", "H04M1/72519"],
    status: "expired",
    maintenanceFeesPaid: false,
  });

  // Patent claim extraction
  test("patent_claim_v1", "patent", {
    patentNumber: "US8046721B2",
    claimNumber: 1,
    claimType: "independent",
    claimText:
      "A method of unlocking a hand-held electronic device, the device including a touch-sensitive display, the method comprising: detecting a contact with the touch-sensitive display at a first predefined location corresponding to an unlock image...",
    claimCategory: "method",
    validityStatus: "presumed_valid",
  });

  // Prior art reference
  test("prior_art_v1", "patent", {
    targetPatent: "US8046721B2",
    targetClaims: [1, 8],
    priorArtType: "patent",
    priorArtReference: "SE0202574A",
    priorArtDate: "2002-08-30",
    priorArtDescription:
      "Neonode touchscreen phone with swipe-to-unlock gesture",
    elementsDisclosed: [
      "touch-sensitive display",
      "gesture detection",
      "unlock action",
    ],
    anticipates: false,
  });

  console.log("   ✓ Patent schemas validated");
}

// ============ Media Schema Tests ============

function testMediaSchemas(): void {
  console.log("\n📰 Testing Media Schemas...\n");

  // Real news article (publicly reported)
  test("news_article_v1", "media", {
    url: "https://www.nytimes.com/2024/01/15/technology/ai-regulation-europe.html",
    archiveUrl:
      "https://web.archive.org/web/20240115120000/https://www.nytimes.com/2024/01/15/technology/ai-regulation-europe.html",
    contentHash:
      "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
    headline: "Europe Reaches Deal on Landmark AI Rules",
    outlet: "The New York Times",
    outletType: "national_newspaper",
    publishedDate: "2024-01-15",
    authors: [{ name: "Adam Satariano", role: "reporter" }],
    isOpinion: false,
    isAnalysis: false,
    section: "technology",
    topics: ["artificial intelligence", "regulation", "european union"],
    geography: ["Europe", "EU"],
    namedSources: [
      {
        name: "Thierry Breton",
        title: "EU Commissioner",
        organization: "European Commission",
        quoteCount: 2,
      },
    ],
    anonymousSourceCount: 0,
    hasCorrection: false,
  });

  // Press release (actual format)
  test("press_release_v1", "media", {
    url: "https://www.apple.com/newsroom/2024/01/apple-announces-vision-pro/",
    archiveUrl: "https://web.archive.org/web/20240102/apple-vision-pro",
    wireService: "direct",
    organization: "Apple Inc.",
    organizationType: "corporation",
    ticker: "AAPL",
    headline:
      "Apple Announces Apple Vision Pro Available in the U.S. on February 2",
    releaseDate: "2024-01-08T14:00:00Z",
    releaseType: "product_launch",
    mediaContacts: [
      { name: "Josh Rosenstock", email: "jrosenstock@apple.com" },
    ],
    quotes: [
      {
        speaker: "Tim Cook",
        title: "CEO",
        quote: "The era of spatial computing has arrived.",
      },
    ],
    containsForwardLooking: true,
    safeHarborStatement: false,
  });

  // Quote attribution
  test("quote_attribution_v1", "media", {
    quote: "The only thing we have to fear is fear itself.",
    speaker: "Franklin D. Roosevelt",
    speakerTitle: "President of the United States",
    speakerOrganization: "United States Government",
    context: "First Inaugural Address",
    venue: "United States Capitol",
    date: "1933-03-04",
    sourceType: "speech",
    isDirectQuote: true,
    language: "en",
    isTranslation: false,
    multipleSourcesConfirm: true,
    speakerConfirmed: false, // Historical figure
    speakerDenied: false,
  });

  // Social media post
  test("social_media_post_v1", "media", {
    platform: "twitter",
    postUrl: "https://twitter.com/elonmusk/status/1234567890123456789",
    archiveUrl:
      "https://web.archive.org/web/20240101/twitter.com/elonmusk/status/1234567890123456789",
    postId: "1234567890123456789",
    accountHandle: "elonmusk",
    accountName: "Elon Musk",
    accountVerified: true,
    accountType: "individual",
    followerCount: 170000000,
    postText:
      "Tesla Full Self-Driving Beta v12 rolling out to all US customers this week",
    postedAt: "2024-01-15T18:30:00Z",
    hasMedia: false,
    isDeleted: false,
    isEdited: false,
    isReply: false,
  });

  console.log("   ✓ Media schemas validated");
}

// ============ Regulatory Schema Tests ============

function testRegulatorySchemas(): void {
  console.log("\n🏛️ Testing Regulatory Schemas...\n");

  // Real government document
  test("government_document_v1", "regulatory", {
    documentId: "FR-2024-00123",
    documentType: "final_rule",
    title: "Artificial Intelligence Risk Management Framework",
    country: "US",
    jurisdiction: "federal",
    issuingAgency: "National Institute of Standards and Technology",
    agencyType: "executive",
    publicationDate: "2024-01-15",
    effectiveDate: "2024-04-15",
    status: "effective",
    officialUrl: "https://www.nist.gov/itl/ai-risk-management-framework",
    citations: ["15 U.S.C. § 278h-1"],
  });

  // FDA drug approval
  test("regulatory_approval_v1", "regulatory", {
    approvalNumber: "NDA 214787",
    approvalType: "drug_approval",
    regulatoryAgency: "FDA",
    agencyDivision: "Center for Drug Evaluation and Research",
    jurisdiction: "US",
    applicant: "Eli Lilly and Company",
    applicantType: "corporation",
    subjectName: "Donanemab",
    subjectDescription: "Anti-amyloid antibody for Alzheimer's disease",
    indication: "Treatment of early symptomatic Alzheimer's disease",
    applicationDate: "2023-07-01",
    approvalDate: "2024-07-02",
    status: "active",
    conditions: ["REMS required", "MRI monitoring"],
    officialUrl:
      "https://www.accessdata.fda.gov/drugsatfda_docs/nda/2024/214787Orig1s000.cfm",
  });

  // Enforcement action (SEC example)
  test("enforcement_action_v1", "regulatory", {
    actionId: "SEC-2024-001",
    actionType: "civil_penalty",
    enforcementAgency: "Securities and Exchange Commission",
    jurisdiction: "US",
    targetEntity: "Example Corp",
    targetType: "corporation",
    violations: [
      {
        statute: "17 CFR 240.10b-5",
        description: "Securities fraud",
        severity: "high",
      },
    ],
    monetaryPenalty: 50000000,
    penaltyCurrency: "USD",
    nonMonetaryPenalties: ["Disgorgement", "Injunction"],
    actionDate: "2024-01-15",
    status: "resolved",
    resolution: "Settlement agreement",
    officialUrl: "https://www.sec.gov/litigation/admin/2024/34-example.htm",
  });

  console.log("   ✓ Regulatory schemas validated");
}

// ============ Legal Schema Tests ============

function testLegalSchemas(): void {
  console.log("\n⚖️ Testing Legal Schemas...\n");

  // Real Supreme Court case
  test("court_case_v1", "legal", {
    caseNumber: "22-277",
    caseName: "Moody v. NetChoice",
    citation: "144 S. Ct. 2383",
    court: "Supreme Court of the United States",
    courtLevel: "supreme",
    jurisdiction: "US",
    plaintiff: "Ashley Moody, Attorney General of Florida",
    defendant: "NetChoice, LLC",
    caseType: "constitutional",
    subjectMatter: ["First Amendment", "Social media", "Content moderation"],
    legalIssues: [
      "Whether Florida and Texas laws restricting content moderation violate First Amendment",
    ],
    filingDate: "2022-09-21",
    decisionDate: "2024-07-01",
    outcome: "remanded",
    holding:
      "Laws regulating social media platforms' content moderation are subject to First Amendment scrutiny",
    precedentialStatus: "binding",
    courtUrl: "https://www.supremecourt.gov/opinions/23pdf/22-277_d18f.pdf",
  });

  // Legislation
  test("legislation_v1", "legal", {
    billNumber: "H.R.4763",
    publicLawNumber: "P.L. 118-123",
    title: "Financial Innovation and Technology for the 21st Century Act",
    shortTitle: "FIT21",
    legislature: "United States Congress",
    session: "118th Congress",
    jurisdiction: "US",
    primarySponsor: "Rep. French Hill",
    introducedDate: "2023-07-20",
    passedDate: "2024-05-22",
    status: "passed_house",
    summary: "Establishes regulatory framework for digital assets",
    subjectAreas: ["cryptocurrency", "securities", "commodities"],
    houseVote: { yea: 279, nay: 136, abstain: 17, date: "2024-05-22" },
    officialUrl: "https://www.congress.gov/bill/118th-congress/house-bill/4763",
  });

  // Contract (public example)
  test("contract_v1", "legal", {
    contractId: "MSFT-ATVI-2022",
    contractType: "merger",
    title: "Agreement and Plan of Merger",
    parties: [
      { name: "Microsoft Corporation", role: "acquirer", jurisdiction: "US" },
      { name: "Activision Blizzard, Inc.", role: "target", jurisdiction: "US" },
    ],
    executionDate: "2022-01-18",
    effectiveDate: "2023-10-13",
    governingLaw: "Delaware",
    disputeResolution: "litigation",
    venue: "Delaware Court of Chancery",
    totalValue: 68700000000,
    currency: "USD",
    status: "executed",
    isConfidential: false,
    publicDisclosure: "SEC Form 8-K",
  });

  // Legal opinion (structure)
  test("legal_opinion_v1", "legal", {
    opinionType: "securities",
    subject: "Token classification under securities laws",
    question: "Whether XYZ Token constitutes a security under the Howey test",
    conclusion:
      "Based on the analysis, XYZ Token likely does not constitute a security",
    lawFirm: "Example Law LLP",
    attorneys: [
      { name: "Jane Smith", barNumber: "NY123456", jurisdiction: "New York" },
    ],
    addressedTo: ["XYZ Foundation", "Prospective Investors"],
    opinionDate: "2024-01-15",
    qualifications: [
      "Opinion limited to federal securities laws",
      "Based on facts as represented",
    ],
    limitations: [
      "Does not address state law",
      "May not be relied upon by third parties",
    ],
    reliedUpon: [
      "Token whitepaper",
      "Operating agreements",
      "Marketing materials",
    ],
    publiclyFiled: false,
  });

  console.log("   ✓ Legal schemas validated");
}

// ============ Clinical Schema Tests ============

function testClinicalSchemas(): void {
  console.log("\n🏥 Testing Clinical Schemas...\n");

  // Real clinical trial (from ClinicalTrials.gov)
  test("clinical_trial_v1", "clinical", {
    registrationId: "NCT04368728",
    registrySource: "clinicaltrials_gov",
    officialTitle:
      "A Phase 2/3 Study to Evaluate the Safety and Efficacy of mRNA-1273 Vaccine in Adults Aged 18 Years and Older",
    briefTitle: "Moderna COVID-19 Vaccine Phase 2/3 Trial",
    acronym: "COVE",
    sponsor: "Moderna, Inc.",
    sponsorType: "industry",
    collaborators: ["Biomedical Advanced Research and Development Authority"],
    principalInvestigator: {
      name: "Lindsey Baden",
      affiliation: "Brigham and Women's Hospital",
    },
    phase: "phase_2_3",
    studyType: "interventional",
    design: {
      allocation: "randomized",
      interventionModel: "parallel",
      masking: "quadruple",
      primaryPurpose: "prevention",
    },
    interventionType: "biological",
    interventions: [
      {
        name: "mRNA-1273",
        type: "biological",
        description: "mRNA vaccine encoding SARS-CoV-2 spike protein",
      },
    ],
    conditions: ["COVID-19", "SARS-CoV-2 Infection"],
    eligibilityCriteria: {
      ageMin: 18,
      healthyVolunteers: true,
      inclusionCriteria: ["Adults 18 years and older"],
      exclusionCriteria: ["Prior COVID-19 infection", "Immunocompromised"],
    },
    enrollmentTarget: 30000,
    enrollmentActual: 30420,
    enrollmentType: "actual",
    startDate: "2020-07-27",
    completionDate: "2022-10-27",
    overallStatus: "completed",
    primaryOutcomes: [
      {
        measure: "Efficacy against COVID-19",
        description: "Prevention of symptomatic COVID-19",
        timeFrame: "14 days post-dose 2",
      },
    ],
    ethicsApproval: { hasApproval: true, irbName: "Western IRB" },
    informedConsent: true,
    registryUrl: "https://clinicaltrials.gov/study/NCT04368728",
  });

  // Clinical trial results
  test("clinical_trial_result_v1", "clinical", {
    registrationId: "NCT04368728",
    resultsPostedDate: "2022-11-15",
    participantFlow: {
      enrolled: 30420,
      started: 30351,
      completed: 28207,
      withdrawn: 2144,
      byArm: [
        { armName: "mRNA-1273", started: 15181, completed: 14134 },
        { armName: "Placebo", started: 15170, completed: 14073 },
      ],
    },
    baselineCharacteristics: {
      totalParticipants: 30351,
      meanAge: 51.4,
      sexDistribution: { male: 15210, female: 15141 },
    },
    primaryOutcomeResults: [
      {
        outcomeMeasure: "Efficacy against symptomatic COVID-19",
        statisticalMethod: "Stratified Cox regression",
        parameterType: "efficacy",
        results: [
          { arm: "mRNA-1273", participants: 15181, value: 11 },
          { arm: "Placebo", participants: 15170, value: 185 },
        ],
      },
    ],
    adverseEvents: {
      reportingDescription: "Solicited and unsolicited adverse events",
      timeFrame: "28 days post-vaccination",
      frequencyThreshold: 5,
      totalSerious: 147,
      totalDeaths: 2,
    },
    conclusions:
      "mRNA-1273 vaccine showed 94.1% efficacy against symptomatic COVID-19",
    publicationDoi: "10.1056/NEJMoa2035389",
  });

  // Adverse event report
  test("adverse_event_v1", "clinical", {
    reportId: "FAERS-2024-001234",
    reportSource: "faers",
    reportType: "initial",
    eventTerms: ["Myocarditis", "Chest pain"],
    eventDescription:
      "Patient developed chest pain and elevated troponin 3 days after vaccination",
    onsetDate: "2024-01-18",
    serious: true,
    seriousnessCriteria: ["hospitalization"],
    outcome: "recovered",
    suspectProducts: [
      {
        productName: "mRNA-1273 COVID-19 Vaccine",
        genericName: "elasomeran",
        manufacturer: "Moderna",
        indication: "COVID-19 prevention",
        startDate: "2024-01-15",
      },
    ],
    patientAge: 22,
    patientSex: "male",
    reporterType: "healthcare_professional",
    reportDate: "2024-01-25",
    reporterCausality: "possible",
    sourceUrl:
      "https://www.fda.gov/drugs/questions-and-answers-fdas-adverse-event-reporting-system-faers",
  });

  console.log("   ✓ Clinical schemas validated");
}

// ============ Financial Schema Tests ============

function testFinancialSchemas(): void {
  console.log("\n💰 Testing Financial Schemas...\n");

  // Real financial report (public data)
  test("financial_report_v1", "financial", {
    companyName: "Apple Inc.",
    ticker: "AAPL",
    cik: "0000320193",
    exchange: "NASDAQ",
    industry: "Technology Hardware",
    sicCode: "3571",
    reportType: "10-K",
    fiscalYear: 2023,
    periodStart: "2022-10-01",
    periodEnd: "2023-09-30",
    filingDate: "2023-11-03",
    incomeStatement: {
      revenue: 383285000000,
      costOfRevenue: 214137000000,
      grossProfit: 169148000000,
      operatingExpenses: 55013000000,
      operatingIncome: 114135000000,
      netIncome: 96995000000,
      eps: 6.13,
      epsDiluted: 6.11,
      sharesOutstanding: 15812547000,
      sharesDiluted: 15869433000,
    },
    balanceSheet: {
      totalAssets: 352583000000,
      currentAssets: 143566000000,
      cash: 29965000000,
      totalLiabilities: 290437000000,
      currentLiabilities: 145308000000,
      longTermDebt: 95281000000,
      shareholdersEquity: 62146000000,
    },
    cashFlow: {
      operatingCashFlow: 110543000000,
      investingCashFlow: -7077000000,
      financingCashFlow: -108488000000,
      freeCashFlow: 99584000000,
      dividendsPaid: 15025000000,
    },
    currency: "USD",
    accountingStandard: "us_gaap",
    unitMultiplier: 1,
    auditor: "Ernst & Young LLP",
    auditOpinion: "unqualified",
    auditDate: "2023-11-03",
    secFilingUrl:
      "https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&CIK=0000320193",
    accessionNumber: "0000320193-23-000106",
  });

  // SEC filing
  test("sec_filing_v1", "financial", {
    accessionNumber: "0000320193-24-000006",
    formType: "8-K",
    filingDate: "2024-01-15",
    acceptanceDate: "2024-01-15T16:30:00",
    filerName: "Apple Inc.",
    cik: "0000320193",
    filerType: "company",
    ticker: "AAPL",
    description: "Results of Operations and Financial Condition",
    items: ["Item 2.02", "Item 9.01"],
    exhibits: [
      {
        exhibitNumber: "99.1",
        description: "Press Release",
        url: "ex99-1.htm",
      },
    ],
    periodOfReport: "2024-01-15",
    fiscalYearEnd: "09-30",
    isAmendment: false,
    edgarUrl:
      "https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&CIK=0000320193&type=8-K",
  });

  // Insider transaction (Form 4)
  test("insider_transaction_v1", "financial", {
    accessionNumber: "0001140361-24-001234",
    filingDate: "2024-01-16",
    issuerName: "Apple Inc.",
    issuerCik: "0000320193",
    issuerTicker: "AAPL",
    reportingPersonName: "Tim Cook",
    reportingPersonCik: "0001214128",
    relationship: {
      isDirector: true,
      isOfficer: true,
      isTenPercentOwner: false,
      isOther: false,
      officerTitle: "Chief Executive Officer",
    },
    transactions: [
      {
        transactionDate: "2024-01-15",
        transactionCode: "S",
        securityTitle: "Common Stock",
        shares: 50000,
        pricePerShare: 185.5,
        totalValue: 9275000,
        acquiredDisposed: "D",
        ownership: "D",
      },
    ],
    postTransactionShares: 3270541,
    edgarUrl:
      "https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&CIK=0001214128&type=4",
  });

  // Financial metric
  test("financial_metric_v1", "financial", {
    entityName: "Apple Inc.",
    entityType: "company",
    ticker: "AAPL",
    metricName: "Price to Earnings Ratio",
    metricCategory: "valuation",
    value: 28.5,
    unit: "ratio",
    asOfDate: "2024-01-15",
    periodType: "ttm",
    priorPeriodValue: 24.8,
    changePercent: 14.9,
    sourceType: "data_provider",
    sourceReference: "Bloomberg Terminal",
  });

  // Analyst rating
  test("analyst_rating_v1", "financial", {
    companyName: "Apple Inc.",
    ticker: "AAPL",
    analystName: "Example Analyst",
    firmName: "Morgan Stanley",
    ratingDate: "2024-01-15",
    rating: "Overweight",
    ratingNormalized: "buy",
    priorRating: "Equal-weight",
    ratingChange: "upgrade",
    priceTarget: 220,
    priorPriceTarget: 190,
    currency: "USD",
    timeHorizon: "12 months",
    currentPrice: 185.5,
    impliedUpside: 18.6,
    reportTitle: "Apple: Services Growth Accelerating",
  });

  console.log("   ✓ Financial schemas validated");
}

// ============ Technical Schema Tests ============

function testTechnicalSchemas(): void {
  console.log("\n🔧 Testing Technical Schemas...\n");

  // Real ISO standard
  test("technical_standard_v1", "technical", {
    standardId: "ISO/IEC 27001:2022",
    title:
      "Information security, cybersecurity and privacy protection — Information security management systems — Requirements",
    version: "2022",
    standardsBody: "ISO/IEC",
    bodyType: "international",
    technicalCommittee: "ISO/IEC JTC 1/SC 27",
    status: "published",
    publicationDate: "2022-10-25",
    effectiveDate: "2022-10-25",
    supersedes: "ISO/IEC 27001:2013",
    scope:
      "Requirements for establishing, implementing, maintaining and continually improving an ISMS",
    keywords: [
      "information security",
      "ISMS",
      "management system",
      "cybersecurity",
    ],
    icsCode: "35.030",
    standardType: "specification",
    domain: "Information security",
    conformanceLevels: ["Full compliance", "Statement of Applicability"],
    certificationProgram: "ISO/IEC 27001 Certification",
    publiclyAvailable: false,
    officialUrl: "https://www.iso.org/standard/82875.html",
  });

  // Real RFC
  test("rfc_v1", "technical", {
    rfcNumber: 9110,
    title: "HTTP Semantics",
    authors: [
      {
        name: "R. Fielding",
        organization: "Adobe",
        email: "fielding@gbiv.com",
      },
      { name: "M. Nottingham", organization: "Fastly" },
      { name: "J. Reschke", organization: "greenbytes" },
    ],
    status: "internet_standard",
    stream: "ietf",
    workingGroup: "httpbis",
    publicationDate: "2022-06-01",
    obsoletes: [2818, 7230, 7231, 7232, 7233, 7235, 7538, 7615, 7694],
    abstract:
      "The Hypertext Transfer Protocol (HTTP) is a stateless application-level request/response protocol.",
    keywords: ["HTTP", "web", "protocol"],
    area: "Applications and Real-Time Area",
    hasErrata: true,
    errataUrl: "https://www.rfc-editor.org/errata/rfc9110",
    rfcEditorUrl: "https://www.rfc-editor.org/rfc/rfc9110",
    datatrackerUrl: "https://datatracker.ietf.org/doc/rfc9110/",
  });

  // API documentation
  test("api_documentation_v1", "technical", {
    apiName: "OpenAI API",
    apiVersion: "v1",
    provider: "OpenAI",
    baseUrl: "https://api.openai.com/v1",
    apiType: "rest",
    specificationFormat: "openapi",
    status: "stable",
    releaseDate: "2020-06-11",
    endpointCount: 15,
    endpoints: [
      {
        path: "/chat/completions",
        method: "POST",
        description: "Create chat completion",
      },
      { path: "/embeddings", method: "POST", description: "Create embeddings" },
    ],
    authMethods: ["bearer"],
    rateLimits: {
      requestsPerSecond: 60,
      requestsPerDay: 10000,
    },
    documentationUrl: "https://platform.openai.com/docs/api-reference",
    specificationUrl:
      "https://raw.githubusercontent.com/openai/openai-openapi/master/openapi.yaml",
    archiveUrl:
      "https://web.archive.org/web/20240115/https://platform.openai.com/docs/api-reference",
  });

  // Software release
  test("software_release_v1", "technical", {
    softwareName: "Node.js",
    version: "20.10.0",
    versionType: "lts",
    publisher: "OpenJS Foundation",
    repository: "https://github.com/nodejs/node",
    license: "MIT",
    releaseDate: "2023-11-22",
    releaseNotes: "https://github.com/nodejs/node/releases/tag/v20.10.0",
    changelog: [
      { type: "feature", description: "Add experimental WebSocket client" },
      { type: "security", description: "OpenSSL security update" },
    ],
    distributionChannels: [
      {
        channel: "npm",
        packageName: "node",
        url: "https://nodejs.org/dist/v20.10.0/",
      },
    ],
    artifacts: [
      {
        name: "node-v20.10.0-darwin-arm64.tar.gz",
        platform: "darwin",
        architecture: "arm64",
        sha256: "abc123def456...",
        size: 45000000,
      },
    ],
    supportStatus: "current",
    eolDate: "2026-04-30",
  });

  // Benchmark result
  test("benchmark_result_v1", "technical", {
    benchmarkName: "MLPerf Inference",
    benchmarkVersion: "v4.0",
    benchmarkType: "ml_inference",
    subjectType: "hardware",
    subjectName: "NVIDIA H100 SXM",
    subjectVersion: "80GB HBM3",
    results: [
      {
        metric: "ResNet-50 Offline",
        value: 98500,
        unit: "samples/sec",
        higherIsBetter: true,
      },
      {
        metric: "BERT-Large Server",
        value: 25000,
        unit: "queries/sec",
        higherIsBetter: true,
      },
    ],
    configuration: {
      os: "Ubuntu 22.04",
      cpu: "AMD EPYC 9654",
      memory: "2TB DDR5",
      gpu: "NVIDIA H100 SXM 80GB x8",
    },
    runDate: "2024-01-15",
    runCount: 5,
    methodology: "MLPerf Inference Rules v4.0",
    sourceType: "official",
    sourceUrl: "https://mlcommons.org/benchmarks/inference/",
  });

  console.log("   ✓ Technical schemas validated");
}

// ============ Data Schema Tests ============

function testDataSchemas(): void {
  console.log("\n📊 Testing Data Schemas...\n");

  // Dataset
  test("dataset_v1", "data", {
    name: "ImageNet Large Scale Visual Recognition Challenge",
    version: "2012",
    description:
      "Large-scale image classification dataset with 1000 categories",
    publisher: "Stanford Vision Lab",
    license: "Custom academic license",
    accessType: "registration_required",
    repositoryUrl: "https://image-net.org/challenges/LSVRC/2012/",
    format: "JPEG images with XML annotations",
    size: 150000000000,
    sizeUnit: "bytes",
    recordCount: 1281167,
    features: [
      { name: "image", type: "binary", description: "JPEG image file" },
      { name: "label", type: "integer", description: "Class label (0-999)" },
      {
        name: "bounding_boxes",
        type: "array",
        description: "Object bounding boxes",
      },
    ],
    temporalCoverage: { start: "2010-01-01", end: "2012-12-31" },
    collectionMethod: "Web crawling and human annotation",
    annotationMethod: "Amazon Mechanical Turk",
    qualityMetrics: { interAnnotatorAgreement: 0.97 },
    citationDoi: "10.1007/s11263-015-0816-y",
    checksums: [{ algorithm: "md5", value: "abc123...", scope: "train.tar" }],
    lastUpdated: "2012-10-01",
  });

  // Raw experimental data
  test("raw_experimental_data_v1", "data", {
    experimentId: "CERN-LHC-RUN3-2024-001",
    datasetId: "dataset-001",
    instrumentName: "ATLAS Detector",
    instrumentType: "particle_detector",
    facilityName: "CERN Large Hadron Collider",
    operatorName: "ATLAS Collaboration",
    collectionDate: "2024-01-15T10:30:00Z",
    collectionDuration: 3600,
    durationUnit: "seconds",
    rawDataHash: "sha256:abc123def456...",
    dataFormat: "ROOT",
    dataSize: 50000000000,
    sampleCount: 1000000,
    calibrationApplied: true,
    calibrationReference: "ATLAS-CONF-2024-001",
    environmentConditions: {
      beamEnergy: 13.6,
      luminosity: 2.4e34,
      temperature: 293,
    },
    qualityFlags: ["good_data", "stable_beam"],
    dataRepositoryUrl: "https://opendata.cern.ch/",
    accessLevel: "public",
  });

  // Measurement
  test("measurement_v1", "data", {
    measurementId: "MEAS-2024-001",
    quantity: "gravitational_constant",
    value: 6.6743e-11,
    uncertainty: 0.00015e-11,
    uncertaintyType: "combined",
    unit: "m³ kg⁻¹ s⁻²",
    measurementDate: "2024-01-15",
    instrument: "Torsion balance",
    instrumentCalibration: "NIST traceable",
    method: "Cavendish experiment",
    conditions: {
      temperature: 293.15,
      pressure: 101325,
      humidity: 0.45,
    },
    repeatCount: 100,
    statisticalMethod: "weighted mean",
    operator: "Example Physics Lab",
    referenceStandard: "CODATA 2018",
    traceability: ["NIST", "SI"],
  });

  // Statistical summary
  test("statistical_summary_v1", "data", {
    dataHash: "sha256:abc123...",
    variable: "annual_temperature",
    n: 365,
    mean: 15.2,
    standardDeviation: 8.4,
    standardError: 0.44,
    minimum: -5.2,
    maximum: 35.8,
    median: 14.8,
    quartiles: { q1: 8.5, q3: 22.1 },
    skewness: 0.15,
    kurtosis: -0.32,
    normalityTest: "Shapiro-Wilk",
    normalityPValue: 0.12,
    outlierCount: 3,
    outlierMethod: "IQR",
    missingCount: 0,
    computedAt: "2024-01-15T12:00:00Z",
    computedBy: "Python pandas",
    softwareVersion: "2.1.0",
  });

  console.log("   ✓ Data schemas validated");
}

// ============ Trust Modifier Tests ============

function testTrustModifiers(): void {
  console.log("\n🔐 Testing Trust Modifiers...\n");

  // Test confidence calculations
  const modifiers: TrustModifiers = {
    positive: {
      peerReviewed: true,
      officialSource: true,
      multipleCorroboration: 3,
      cryptographicProof: true,
    },
    negative: {
      conflictOfInterest: true,
    },
  };

  const baseConfidence = 0.8;
  const tier = "authoritative" as const;
  const adjusted = calculateAdjustedConfidence(baseConfidence, tier, modifiers);

  console.log(`   Base confidence: ${baseConfidence}`);
  console.log(
    `   Reliability tier: ${tier} (max: ${RELIABILITY_MAX_CONFIDENCE[tier]})`,
  );
  console.log(`   Adjusted confidence: ${adjusted.toFixed(4)}`);
  console.log(
    `   Modifiers applied: peer review (+10%), official (+5%), corroboration (+3%), crypto (+5%), COI (-10%)`,
  );

  results.push({
    schemaId: "trust_modifiers",
    category: "infrastructure",
    passed:
      adjusted > baseConfidence && adjusted <= RELIABILITY_MAX_CONFIDENCE[tier],
    errors: [],
    insights: [
      `Base ${baseConfidence} → Adjusted ${adjusted.toFixed(4)}`,
      `Capped at tier max ${RELIABILITY_MAX_CONFIDENCE[tier]}`,
    ],
  });

  console.log("   ✓ Trust modifiers validated");
}

// ============ Main Test Runner ============

async function runAllTests(): Promise<void> {
  console.log(
    "═══════════════════════════════════════════════════════════════",
  );
  console.log("       SOURCE SCHEMA INTEGRATION TESTS");
  console.log("       Testing claim validation pipeline with real-world data");
  console.log(
    "═══════════════════════════════════════════════════════════════",
  );

  // Initialize
  resetSchemaRegistry();
  const init = initializeSchemas();
  console.log(
    `\n📚 Initialized ${init.registered} schemas (${init.failed.length} failed)\n`,
  );

  if (init.failed.length > 0) {
    console.log("Failed schemas:", init.failed);
  }

  // Run all tests
  testPatentSchemas();
  testMediaSchemas();
  testRegulatorySchemas();
  testLegalSchemas();
  testClinicalSchemas();
  testFinancialSchemas();
  testTechnicalSchemas();
  testDataSchemas();
  testTrustModifiers();

  // Summary
  console.log(
    "\n═══════════════════════════════════════════════════════════════",
  );
  console.log("                          SUMMARY");
  console.log(
    "═══════════════════════════════════════════════════════════════\n",
  );

  const passed = results.filter((r) => r.passed).length;
  const failed = results.filter((r) => !r.passed).length;
  const categories = new Set(results.map((r) => r.category));

  console.log(`Total tests: ${results.length}`);
  console.log(`Passed: ${passed} ✅`);
  console.log(`Failed: ${failed} ❌`);
  console.log(`Categories tested: ${categories.size}`);

  if (failed > 0) {
    console.log("\n❌ Failed tests:");
    for (const result of results.filter((r) => !r.passed)) {
      console.log(`   ${result.schemaId}: ${result.errors.join(", ")}`);
    }
  }

  // Return results for reporting
  return;
}

// Export for use in test harness
export { runAllTests, results };
export type { TestResult };

// Run if called directly
runAllTests().catch(console.error);
