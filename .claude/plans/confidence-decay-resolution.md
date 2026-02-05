# GAP-008: Confidence Decay Resolution

## Summary

Resolve the confidence decay gap by implementing **lazy decay calculation** that returns effective (decayed) confidence values on query without modifying stored values.

**Core Principle**: Follow GAP-009's pattern - don't run background jobs to update stored values; calculate decay on-demand when querying facts.

---

## Problem Statement

**Current State**: Decay logic EXISTS but is not APPLIED

- `applyConfidenceDecay()` in `state/knowledge.ts:577` - Batch updates stored values (never called)
- `calculateDecayedConfidence()` in `consensus/pot/confidence.ts:230` - Pure decay calculation
- `calculateCategoryDecayedConfidence()` in `consensus/pot/confidence.ts:247` - Category-aware decay

**Gap**: When consumers query facts, they get the **stored confidence** (the value at last verification), not the **effective confidence** (after time decay). This means:
1. A fact verified 6 months ago shows 0.95 confidence when it should show ~0.5
2. Consumers have no way to know the time-decayed confidence
3. Decisions are made on stale confidence values

---

## Design

### Approach: Lazy Decay Calculation

Following GAP-009's lazy expiration pattern:

```
Query for fact
    ↓
Calculate time since verification
    ↓
Apply decay formula based on category
    ↓
Return effective confidence (don't modify stored value)
```

### Key Insight: Stored vs Effective

| Term | Description |
|------|-------------|
| **Stored Confidence** | The confidence value at last verification (immutable until reverification) |
| **Effective Confidence** | Stored confidence × decay factor (calculated on query) |

Stored values remain immutable - they represent the verification result at that point in time. Effective values are calculated dynamically based on current block.

### Integration Points

1. **KnowledgeStateTree** - Add `getEffectiveConfidence()` method
2. **UnifiedQueryService** - Return effective confidence in query results
3. **FactResult type** - Add `effectiveConfidence` field alongside `fact.confidence`

---

## Implementation

### Phase 1: Core Method

**File**: `state/knowledge.ts`

Add method to calculate effective confidence for a fact:

```typescript
/**
 * Get effective (time-decayed) confidence for a fact
 *
 * GAP-008: Lazy decay calculation - doesn't modify stored values
 *
 * @param fact - The fact to calculate decay for
 * @param currentBlock - Current block number
 * @returns Effective confidence after time decay
 */
getEffectiveConfidence(
  fact: FactState,
  currentBlock: bigint
): {
  effectiveConfidence: Confidence;
  storedConfidence: Confidence;
  decayFactor: number;
  isEternal: boolean;
  blocksSinceVerification: number;
} {
  const blocksSinceVerification = Number(currentBlock - fact.verifiedAt);

  // Map stratum to category for decay calculation
  const category = this.stratumToCategory(fact.stratum);

  // Use category-aware decay
  const decay = calculateCategoryDecayedConfidence(
    fact.confidence,
    blocksSinceVerification,
    category
  );

  return {
    effectiveConfidence: decay.confidence,
    storedConfidence: fact.confidence,
    decayFactor: decay.decayFactor,
    isEternal: decay.isEternal,
    blocksSinceVerification,
  };
}
```

### Phase 2: Enhanced Fact Retrieval

**File**: `state/knowledge.ts`

Add method that returns fact with decay info:

```typescript
/**
 * Get fact with effective confidence calculated
 *
 * GAP-008: Returns fact plus decay information
 */
getFactWithDecay(
  factId: string,
  currentBlock: bigint
): {
  fact: FactState;
  effectiveConfidence: Confidence;
  decayInfo: DecayInfo;
} | undefined
```

### Phase 3: Type Updates

**File**: `core/knowledge/query/types.ts`

Update FactResult to include decay info:

```typescript
export interface FactResult {
  factId: string;
  fact: FactState;
  domain: OntologicalDomain;
  stratum: OntologicalStratum;
  relevance: number;
  cached: boolean;
  prefetched: boolean;
  // GAP-008: Decay information
  effectiveConfidence: number;
  decayInfo?: {
    storedConfidence: number;
    decayFactor: number;
    isEternal: boolean;
    blocksSinceVerification: number;
  };
}
```

### Phase 4: Query Service Integration

**File**: `core/knowledge/query/unified_query_service.ts`

Update fact results to include effective confidence:

```typescript
// In processIntent(), when building FactResult:
const decayInfo = knowledgeTree.getEffectiveConfidence(
  fact,
  session.currentBlock
);

return {
  factId,
  fact,
  // ... existing fields
  effectiveConfidence: decayInfo.effectiveConfidence,
  decayInfo: {
    storedConfidence: decayInfo.storedConfidence,
    decayFactor: decayInfo.decayFactor,
    isEternal: decayInfo.isEternal,
    blocksSinceVerification: decayInfo.blocksSinceVerification,
  },
};
```

---

## Files to Modify

| File | Change |
|------|--------|
| `state/knowledge.ts` | Add `getEffectiveConfidence()`, `getFactWithDecay()`, `stratumToCategory()` |
| `core/knowledge/query/types.ts` | Add `effectiveConfidence` and `decayInfo` to FactResult |
| `core/knowledge/query/unified_query_service.ts` | Calculate and return effective confidence |

---

## Stratum to Category Mapping

```typescript
const STRATUM_TO_CATEGORY: Record<OntologicalStratum, string> = {
  'formal_logical': 'analytic',    // Eternal - no decay
  'cryptographic': 'protocol',     // Eternal - no decay
  'empirical_physical': 'empirical', // Half-life: ~30 days
  'historical': 'historical',      // Half-life: ~180 days
  'social_consensus': 'social',    // Half-life: ~7 days
};
```

---

## Verification

### Test Cases

1. **Eternal facts don't decay**
   - Formal logical fact → effectiveConfidence = storedConfidence
   - decayFactor = 1.0, isEternal = true

2. **Empirical facts decay correctly**
   - Fact verified 30 days ago → ~50% of original confidence
   - Fact verified 60 days ago → ~25% of original confidence

3. **Category-aware decay**
   - Social facts decay faster than empirical
   - Historical facts decay slower than empirical

4. **Query results include decay info**
   - FactResult contains effectiveConfidence
   - decayInfo provides transparency

### Manual Test

```typescript
const factResult = await queryService.processIntent(sessionId, {
  type: 'verify',
  factId: 'fact-123',
});

// Should see:
// fact.confidence = 0.9 (stored at verification)
// effectiveConfidence = 0.6 (after decay)
// decayInfo.decayFactor = 0.67
// decayInfo.blocksSinceVerification = 50000
```

---

## PP Alignment

| Dimension | Implementation |
|-----------|----------------|
| **TRUTH** | Returns accurate time-decayed confidence |
| **UNDERSTANDING** | Clear separation of stored vs effective |
| **BEAUTY** | Follows existing lazy evaluation pattern (GAP-009) |
| **JUSTICE** | Consumers get honest confidence values |
| **CREATIVITY** | Enables time-aware knowledge queries |
