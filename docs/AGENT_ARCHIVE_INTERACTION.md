# Agent-Archive Interaction Guide

This document explains how AI agents query the Knowledge Archive to determine the truthfulness of statements, including message formats, fee structures, and practical examples.

---

## Overview

```
┌──────────────┐         JSON-RPC 2.0          ┌──────────────────┐
│              │  ──────────────────────────▶  │                  │
│   AI Agent   │         Request               │   Archive Node   │
│              │  ◀──────────────────────────  │                  │
│              │         Response + Fee        │                  │
└──────────────┘                               └──────────────────┘
```

Agents communicate with Archive Nodes via **JSON-RPC 2.0** over HTTP or WebSocket. Every query has an associated cost paid in LGM tokens.

---

## Connection Flow

### 1. Create a Session

Before querying, agents must establish a session:

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "archive_createSession",
  "params": {
    "agentDid": "did:lgm:agent:abc123"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "sessionId": "archive:did:lgm:agent:abc123:1704067200000"
  }
}
```

The `sessionId` is used for all subsequent queries. Sessions track query history and enable context-aware responses.

---

## Querying for Truth

### Method: `archive_evaluateTruth`

This is the primary method for determining if a statement is true.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "archive_evaluateTruth",
  "params": {
    "sessionId": "archive:did:lgm:agent:abc123:1704067200000",
    "factId": "fact-quantum-entanglement-001"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "success": true,
    "result": {
      "factId": "fact-quantum-entanglement-001",
      "claim": "Quantum entanglement allows instantaneous correlation between particles regardless of distance",
      "verdict": "true",
      "confidence": 0.92,
      "confidenceRatio": {
        "supporting": 2.45,
        "contradicting": 0.12,
        "ratio": 0.953,
        "netConfidence": 0.89
      },
      "supportingFacts": [
        {
          "factId": "fact-bell-test-001",
          "claim": "Bell test experiments violate local hidden variable theories",
          "confidence": 0.95,
          "referenceType": "supports",
          "strength": 0.9,
          "domain": "physics.mechanics.quantum"
        },
        {
          "factId": "fact-epr-paradox-001",
          "claim": "EPR paradox demonstrates non-local correlations",
          "confidence": 0.88,
          "referenceType": "supports",
          "strength": 0.85,
          "domain": "physics.mechanics.quantum"
        }
      ],
      "contradictingFacts": [
        {
          "factId": "fact-locality-critique-001",
          "claim": "Some interpretations preserve locality",
          "confidence": 0.45,
          "contradictionType": "implied",
          "strength": 0.3,
          "domain": "physics.mechanics.quantum"
        }
      ],
      "ontologicalContext": {
        "stratum": "physical",
        "stratumCeiling": 0.99,
        "domainPath": ["physics", "mechanics", "quantum"],
        "crossStratumDependencies": [],
        "crossStratumDiscount": 0
      },
      "reasoning": "Verdict: TRUE (confidence: 92.0%). Evidence ratio: 95.3% supporting. 2 supporting fact(s) with weight 2.450. 1 contradicting fact(s) with weight 0.120. Stratum: physical (ceiling: 99%)"
    },
    "metadata": {
      "queryId": "aq_1704067200123_x7k2m9",
      "executionTime": 45,
      "archiveBlockNumber": "125000"
    },
    "pp": {
      "scores": {
        "truth": 0.92,
        "understanding": 0.85,
        "beauty": 0.78,
        "justice": 0.88,
        "creativity": 0.2
      },
      "signals": [],
      "confidenceReasoning": "High confidence based on experimental verification",
      "understandingEnabled": ["quantum_mechanics", "entanglement", "non_locality"],
      "creativeExtensions": []
    }
  }
}
```

### Understanding the Response

| Field | Meaning |
|-------|---------|
| `verdict` | `"true"`, `"false"`, or `"uncertain"` |
| `confidence` | 0-1 score after all adjustments |
| `confidenceRatio.ratio` | Supporting / (Supporting + Contradicting) |
| `confidenceRatio.supporting` | Weighted sum of supporting evidence |
| `confidenceRatio.contradicting` | Weighted sum of contradicting evidence |
| `supportingFacts` | Facts that support this claim |
| `contradictingFacts` | Facts that contradict this claim |
| `ontologicalContext.stratum` | Which stratum this fact belongs to |
| `ontologicalContext.stratumCeiling` | Maximum possible confidence for this stratum |
| `reasoning` | Human-readable explanation |

### Verdict Determination Logic

```
IF confidence ≥ 0.7 AND ratio ≥ 0.6:
    verdict = "true"
ELSE IF confidence ≤ 0.3 OR (ratio < 0.4 AND contradicting > 0.3):
    verdict = "false"
ELSE:
    verdict = "uncertain"
```

---

## Alternative: Query by Statement Text

If you don't have a `factId`, you can search semantically:

### Method: `archive_search`

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "archive_search",
  "params": {
    "sessionId": "archive:did:lgm:agent:abc123:1704067200000",
    "query": "quantum entanglement instantaneous communication",
    "domain": "physics.*",
    "limit": 5
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "success": true,
    "result": [
      {
        "factId": "fact-quantum-entanglement-001",
        "claim": "Quantum entanglement allows instantaneous correlation...",
        "domain": "physics.mechanics.quantum",
        "stratum": "physical",
        "confidence": 0.92,
        "similarityScore": 0.89
      },
      {
        "factId": "fact-no-ftl-001",
        "claim": "Entanglement cannot transmit information faster than light",
        "domain": "physics.mechanics.quantum",
        "stratum": "physical",
        "confidence": 0.96,
        "similarityScore": 0.82
      }
    ],
    "pagination": {
      "hasMore": false,
      "totalEstimate": 2
    },
    "metadata": {
      "queryId": "aq_1704067200456_p3n8q2",
      "executionTime": 120,
      "archiveBlockNumber": "125000"
    }
  }
}
```

Then evaluate truth on the most relevant result:

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "archive_evaluateTruth",
  "params": {
    "sessionId": "archive:did:lgm:agent:abc123:1704067200000",
    "factId": "fact-quantum-entanglement-001"
  }
}
```

---

## Fee Structure

### How Costs Are Calculated

Every query has a cost in LGM (Legible Money tokens):

```
Total Cost = Base Fee + Depth Premium + Confidence Premium - Creativity Bonus

Where:
  Base Fee = fact_count × 0.001 LGM
  Depth Premium = Base Fee × (depth_multiplier - 1)
  Confidence Premium = Base Fee × confidence_requirement × 0.5
  Creativity Bonus = -10% if query leads to new knowledge paths
```

**Depth Multipliers:**
| Query Depth | Multiplier | Use Case |
|-------------|------------|----------|
| `surface` | 1× | Quick lookups |
| `working` | 2× | Standard queries |
| `deep` | 4× | Comprehensive analysis |
| `expert` | 8× | Full graph traversal |

### Example Costs

| Query Type | Facts Returned | Depth | Approximate Cost |
|------------|----------------|-------|------------------|
| `evaluateTruth` (simple) | 5 | working | 0.01 LGM |
| `evaluateTruth` (complex) | 20 | deep | 0.08 LGM |
| `search` | 10 | working | 0.02 LGM |
| `getRelated` | 15 | working | 0.03 LGM |
| `batchQuery` (5 queries) | 50 | working | 0.10 LGM |

### Fee Distribution

```
┌────────────────────────────────────────────────────────────┐
│                    FEE DISTRIBUTION                         │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  Query Fee (100%)                                           │
│  ├── Archive Node Operator: 48%                             │
│  │   └── Covers infrastructure, bandwidth, storage          │
│  │                                                          │
│  ├── Research Fund: 22%                                     │
│  │   └── Funds verification of new knowledge                │
│  │                                                          │
│  ├── Knowledge Contributors: 30%                            │
│  │   └── Rewards creators of queried facts                  │
│  │                                                          │
│  └── Protocol Treasury: 0% (queries are fee-neutral)        │
│                                                             │
└────────────────────────────────────────────────────────────┘
```

### Payment Flow

1. **Pre-funded Account**: Agent deposits LGM to their account
2. **Query Submitted**: Fee is quoted in response metadata
3. **Fee Deducted**: Automatically deducted from agent's balance
4. **Insufficient Funds**: Query rejected with `INSUFFICIENT_BALANCE` error

---

## Complete Query Flow

### Step-by-Step Example

**Scenario**: Agent wants to verify "The speed of light is 299,792,458 m/s"

#### Step 1: Create Session
```json
// Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "archive_createSession",
  "params": { "agentDid": "did:lgm:agent:verifier-bot" }
}

// Response
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": { "sessionId": "archive:did:lgm:agent:verifier-bot:1704067200000" }
}
```

#### Step 2: Search for the Fact
```json
// Request
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "archive_search",
  "params": {
    "sessionId": "archive:did:lgm:agent:verifier-bot:1704067200000",
    "query": "speed of light meters per second",
    "domain": "physics.*",
    "limit": 3
  }
}

// Response
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "success": true,
    "result": [
      {
        "factId": "fact-speed-of-light-001",
        "claim": "The speed of light in vacuum is exactly 299,792,458 m/s",
        "domain": "physics.electromagnetism",
        "stratum": "fundamental",
        "confidence": 1.0,
        "similarityScore": 0.95
      }
    ]
  }
}
```

#### Step 3: Evaluate Truth
```json
// Request
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "archive_evaluateTruth",
  "params": {
    "sessionId": "archive:did:lgm:agent:verifier-bot:1704067200000",
    "factId": "fact-speed-of-light-001"
  }
}

// Response
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "success": true,
    "result": {
      "factId": "fact-speed-of-light-001",
      "claim": "The speed of light in vacuum is exactly 299,792,458 m/s",
      "verdict": "true",
      "confidence": 1.0,
      "confidenceRatio": {
        "supporting": 5.2,
        "contradicting": 0.0,
        "ratio": 1.0,
        "netConfidence": 1.0
      },
      "supportingFacts": [
        {
          "factId": "fact-si-definition-001",
          "claim": "The meter is defined by the speed of light",
          "confidence": 1.0,
          "referenceType": "supports",
          "strength": 1.0,
          "domain": "physics.metrology"
        }
      ],
      "contradictingFacts": [],
      "ontologicalContext": {
        "stratum": "fundamental",
        "stratumCeiling": 1.0,
        "domainPath": ["physics", "electromagnetism"],
        "crossStratumDependencies": [],
        "crossStratumDiscount": 0
      },
      "reasoning": "Verdict: TRUE (confidence: 100.0%). Evidence ratio: 100.0% supporting. This is a fundamental constant defined by international agreement. Stratum: fundamental (ceiling: 100%)"
    }
  }
}
```

#### Step 4: Get Related Facts (Optional)
```json
// Request
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "archive_getRelated",
  "params": {
    "sessionId": "archive:did:lgm:agent:verifier-bot:1704067200000",
    "factId": "fact-speed-of-light-001",
    "includeSemanticSimilar": true,
    "maxDepth": 2
  }
}

// Response
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "success": true,
    "result": {
      "factId": "fact-speed-of-light-001",
      "claim": "The speed of light in vacuum is exactly 299,792,458 m/s",
      "supports": [
        {
          "factId": "fact-maxwell-equations-001",
          "claim": "Maxwell's equations predict electromagnetic wave speed",
          "confidence": 0.99,
          "referenceStrength": 0.95
        }
      ],
      "contradicts": [],
      "extends": [],
      "specializes": [],
      "generalizes": [],
      "semanticallySimilar": [
        {
          "factId": "fact-relativity-001",
          "claim": "Nothing can travel faster than light in vacuum",
          "confidence": 0.98,
          "similarityScore": 0.78
        }
      ],
      "statistics": {
        "totalRelated": 2,
        "supportingWeight": 0.95,
        "contradictingWeight": 0,
        "semanticMatches": 1,
        "averageSimilarity": 0.78
      }
    }
  }
}
```

---

## Batch Queries

For efficiency, agents can batch multiple queries:

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "archive_batchQuery",
  "params": {
    "sessionId": "archive:did:lgm:agent:verifier-bot:1704067200000",
    "queries": [
      {
        "type": "truth",
        "filters": { "factId": "fact-001" },
        "pagination": { "limit": 1, "sortBy": "confidence", "sortOrder": "desc" },
        "options": { "includeReferences": true, "includeOntology": true, "includeSimilarity": false }
      },
      {
        "type": "truth",
        "filters": { "factId": "fact-002" },
        "pagination": { "limit": 1, "sortBy": "confidence", "sortOrder": "desc" },
        "options": { "includeReferences": true, "includeOntology": true, "includeSimilarity": false }
      },
      {
        "type": "domain",
        "filters": { "domain": "biology.*", "minConfidence": 0.8 },
        "pagination": { "limit": 10, "sortBy": "confidence", "sortOrder": "desc" },
        "options": { "includeReferences": false, "includeOntology": false, "includeSimilarity": false }
      }
    ],
    "parallel": true,
    "maxParallelism": 5
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "result": {
    "results": {
      "0": { "success": true, "items": [...] },
      "1": { "success": true, "items": [...] },
      "2": { "success": true, "items": [...] }
    },
    "totalTime": 85,
    "errors": {}
  }
}
```

Batch queries have a **10% discount** on total fees.

---

## Error Handling

### Common Error Codes

| Code | Name | Meaning |
|------|------|---------|
| -32001 | `SESSION_NOT_FOUND` | Invalid or expired session |
| -32002 | `FACT_NOT_FOUND` | Requested factId doesn't exist |
| -32003 | `DOMAIN_NOT_FOUND` | Domain pattern matches nothing |
| -32004 | `QUERY_TIMEOUT` | Query exceeded time limit |
| -32602 | `INVALID_PARAMS` | Missing or malformed parameters |
| -32603 | `INTERNAL_ERROR` | Archive node error |

### Error Response Format

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "error": {
    "code": -32002,
    "message": "Fact not found: fact-nonexistent-001",
    "data": {
      "factId": "fact-nonexistent-001",
      "suggestion": "Use archive_search to find similar facts"
    }
  }
}
```

---

## Best Practices for Agents

### 1. Cache Session IDs
Sessions are lightweight but have setup cost. Reuse them.

```python
# Good
session = create_session_once()
for query in queries:
    evaluate_truth(session, query)

# Bad
for query in queries:
    session = create_session()  # Wasteful
    evaluate_truth(session, query)
```

### 2. Use Batch Queries
When evaluating multiple facts, batch them:

```python
# Good: Single request, 10% fee discount
batch_query([fact1, fact2, fact3, fact4, fact5])

# Bad: 5 separate requests, full fee each
evaluate_truth(fact1)
evaluate_truth(fact2)
evaluate_truth(fact3)
evaluate_truth(fact4)
evaluate_truth(fact5)
```

### 3. Check Confidence Ratio, Not Just Verdict

```python
result = evaluate_truth(fact_id)

# Don't just check verdict
if result.verdict == "true":
    use_fact()

# Better: Check the evidence quality
if result.verdict == "true" and result.confidence_ratio.ratio > 0.8:
    use_fact_with_high_confidence()
elif result.verdict == "true" and result.confidence_ratio.contradicting > 0.5:
    use_fact_but_note_controversy()
```

### 4. Respect Stratum Ceilings

A social stratum fact with 0.85 confidence is maximally confident for that domain. Don't treat it as less reliable than a 0.90 physics fact.

```python
# Wrong interpretation
physics_fact.confidence = 0.90  # "Pretty confident"
social_fact.confidence = 0.85   # "Less confident"

# Correct interpretation
physics_fact.confidence = 0.90  # 91% of ceiling (0.99)
social_fact.confidence = 0.85   # 100% of ceiling (0.85) - maximally confident!
```

### 5. Handle "Uncertain" Gracefully

```python
result = evaluate_truth(fact_id)

if result.verdict == "uncertain":
    # Don't treat as "probably false"
    # Instead: Get more context
    related = get_related_statements(fact_id)

    if related.contradicts:
        # There's active disagreement
        flag_as_contested()
    else:
        # Just insufficient evidence
        flag_as_needs_more_research()
```

---

## SDK Example (TypeScript)

```typescript
import { ArchiveClient } from '@legible-money/sdk';

async function verifyStatement(statement: string): Promise<boolean> {
  const client = new ArchiveClient({
    endpoint: 'https://archive.legiblemoney.network',
    agentDid: 'did:lgm:agent:my-agent',
  });

  // Create session
  const session = await client.createSession();

  // Search for relevant facts
  const searchResults = await client.search(session, {
    query: statement,
    limit: 5,
  });

  if (searchResults.length === 0) {
    console.log('No relevant facts found');
    return false;
  }

  // Evaluate the most relevant fact
  const evaluation = await client.evaluateTruth(
    session,
    searchResults[0].factId
  );

  console.log(`Verdict: ${evaluation.verdict}`);
  console.log(`Confidence: ${evaluation.confidence}`);
  console.log(`Supporting evidence: ${evaluation.supportingFacts.length}`);
  console.log(`Contradicting evidence: ${evaluation.contradictingFacts.length}`);
  console.log(`Reasoning: ${evaluation.reasoning}`);

  return evaluation.verdict === 'true' && evaluation.confidence >= 0.7;
}

// Usage
const isTrue = await verifyStatement(
  'Water boils at 100°C at sea level'
);
```

---

## Summary

| To Do This... | Use This Method | Typical Cost |
|---------------|-----------------|--------------|
| Check if a fact is true | `archive_evaluateTruth` | 0.01-0.05 LGM |
| Find facts about a topic | `archive_search` | 0.01-0.03 LGM |
| Get related/contradicting facts | `archive_getRelated` | 0.02-0.05 LGM |
| Query a knowledge domain | `archive_queryDomain` | 0.01-0.10 LGM |
| Verify multiple facts | `archive_batchQuery` | 10% discount |

The Knowledge Archive provides AI agents with a reliable, economically-incentivized source of verified knowledge with transparent confidence scoring and full provenance tracking.
