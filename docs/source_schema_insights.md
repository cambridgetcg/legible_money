# Source Schema Test Report

Generated: 2026-02-05T21:21:01.795Z

## Summary

- **Total Tests**: 31
- **Passed**: 31 ✅
- **Failed**: 0 ❌
- **Top-Level Categories**: 8
- **Unique Domains**: 31

## Domain Hierarchy

```
clinical/
  └── adverse_events (1 schema)
  └── results (1 schema)
  └── trials (1 schema)
data/
  └── datasets (1 schema)
  └── experimental (1 schema)
  └── measurements (1 schema)
  └── statistics (1 schema)
financial/
  └── insider (1 schema)
  └── metrics (1 schema)
  └── ratings (1 schema)
  └── reports (1 schema)
  └── sec (1 schema)
intellectual_property/
  └── patent_claims (1 schema)
  └── patents (1 schema)
  └── prior_art (1 schema)
legal/
  └── cases (1 schema)
  └── contracts (1 schema)
  └── legislation (1 schema)
  └── opinions (1 schema)
media/
  └── news (1 schema)
  └── press_releases (1 schema)
  └── quotes (1 schema)
  └── social (1 schema)
regulatory/
  └── approvals (1 schema)
  └── enforcement (1 schema)
  └── government (1 schema)
technical/
  └── api (1 schema)
  └── benchmarks (1 schema)
  └── ietf (1 schema)
  └── software (1 schema)
  └── standards (1 schema)
```

## Results by Domain

### Clinical (3/3)

#### `clinical.adverse_events`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| adverse_event_v1 | ✅ | 16 blocks (~40.3s) | reference |

#### `clinical.results`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| clinical_trial_result_v1 | ✅ | 26 blocks (~65.5s) | consensus |

#### `clinical.trials`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| clinical_trial_v1 | ✅ | 22 blocks (~55.5s) | reference |

### Data (4/4)

#### `data.datasets`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| dataset_v1 | ✅ | 20 blocks (~50.4s) | reference |

#### `data.experimental`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| raw_experimental_data_v1 | ✅ | 24 blocks (~60.5s) | reference |

#### `data.measurements`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| measurement_v1 | ✅ | 12 blocks (~30.3s) | reference |

#### `data.statistics`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| statistical_summary_v1 | ✅ | 16 blocks (~40.3s) | automatic |

### Financial (5/5)

#### `financial.insider`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| insider_transaction_v1 | ✅ | 12 blocks (~30.3s) | reference |

#### `financial.metrics`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| financial_metric_v1 | ✅ | 12 blocks (~30.3s) | reference |

#### `financial.ratings`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| analyst_rating_v1 | ✅ | 12 blocks (~30.3s) | reference |

#### `financial.reports`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| financial_report_v1 | ✅ | 20 blocks (~50.4s) | reference |

#### `financial.sec`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| sec_filing_v1 | ✅ | 16 blocks (~40.3s) | reference |

### Intellectual_property (3/3)

#### `intellectual_property.patent_claims`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| patent_claim_v1 | ✅ | 16 blocks (~40.3s) | reference |

#### `intellectual_property.patents`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| patent_document_v1 | ✅ | 18 blocks (~45.4s) | reference |

#### `intellectual_property.prior_art`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| prior_art_v1 | ✅ | 22 blocks (~55.5s) | consensus |

### Legal (4/4)

#### `legal.cases`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| court_case_v1 | ✅ | 20 blocks (~50.4s) | reference |

#### `legal.contracts`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| contract_v1 | ✅ | 16 blocks (~40.3s) | consensus |

#### `legal.legislation`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| legislation_v1 | ✅ | 16 blocks (~40.3s) | reference |

#### `legal.opinions`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| legal_opinion_v1 | ✅ | 16 blocks (~40.3s) | consensus |

### Media (4/4)

#### `media.news`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| news_article_v1 | ✅ | 16 blocks (~40.3s) | reference |

#### `media.press_releases`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| press_release_v1 | ✅ | 12 blocks (~30.3s) | reference |

#### `media.quotes`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| quote_attribution_v1 | ✅ | 12 blocks (~30.3s) | reference |

#### `media.social`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| social_media_post_v1 | ✅ | 10 blocks (~25.2s) | reference |

### Regulatory (3/3)

#### `regulatory.approvals`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| regulatory_approval_v1 | ✅ | 16 blocks (~40.3s) | reference |

#### `regulatory.enforcement`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| enforcement_action_v1 | ✅ | 18 blocks (~45.4s) | reference |

#### `regulatory.government`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| government_document_v1 | ✅ | 16 blocks (~40.3s) | reference |

### Technical (5/5)

#### `technical.api`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| api_documentation_v1 | ✅ | 12 blocks (~30.3s) | reference |

#### `technical.benchmarks`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| benchmark_result_v1 | ✅ | 16 blocks (~40.3s) | consensus |

#### `technical.ietf`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| rfc_v1 | ✅ | 12 blocks (~30.3s) | reference |

#### `technical.software`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| software_release_v1 | ✅ | 12 blocks (~30.3s) | reference |

#### `technical.standards`

| Schema | Status | Timing | Verification |
|--------|--------|--------|--------------|
| technical_standard_v1 | ✅ | 16 blocks (~40.3s) | reference |

## Insights

### Verification Timing

| Longest Round Times | Blocks |
|---------------------|--------|
| clinical_trial_result_v1 | 26 |
| raw_experimental_data_v1 | 24 |
| prior_art_v1 | 22 |
| clinical_trial_v1 | 22 |
| court_case_v1 | 20 |

### Verification Methods

| Method | Count |
|--------|-------|
| reference | 25 |
| consensus | 5 |
| automatic | 1 |

### Trust Modifiers

- Trust modifier calculation: 0.8 → 0.8316
- Max confidence by tier: authoritative=0.95, anonymous=0.4

## Key Observations

1. **Schema Coverage**: All 31 source schemas validated successfully across 8 top-level categories
2. **Domain Hierarchy**: Schemas use hierarchical domains (e.g., `clinical.trials`, `financial.sec`) for precise categorization
3. **Timing Distribution**: Clinical and legal schemas require longer verification rounds (20-26 blocks)
4. **Verification Methods**: Most source schemas use 'reference' or 'consensus' verification
5. **Trust Tiers**: Source reliability directly caps maximum achievable confidence

## Recommendations

1. **Archival Requirements**: All media/web sources should require archive URLs
2. **Cross-Reference**: Patent prior art and legal cases benefit from multiple references
3. **Timing Optimization**: Consider async pre-fetching for document-heavy schemas
4. **Clinical Special Handling**: FDA/clinical data needs institutional reference verification
5. **Domain-Based Routing**: Use domain hierarchy for validator specialization

