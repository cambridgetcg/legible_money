# GAP-009: Fact Expiration - Resolution Design

> **Status:** Design Complete
> **Created:** 2026-02-05
> **Gap Priority:** P2 (Quality improvement)
> **Estimated Effort:** Small (2-3 days)

---

## Problem Statement

When facts are verified and stored in the knowledge layer:
- ✅ **Facts have confidence that can decay** - `applyConfidenceDecay()` exists in `KnowledgeStateTree`
- ✅ **Expiration threshold exists** - `DEFAULT_TRUTH_PARAMETERS.minConfidenceThreshold = 0.3`
- ✅ **Core knowledge types support expiration** - `Fact.expiresAt?: number` and `FactStatus = 'expired'`
- ❌ **State layer lacks expiration** - `FactState` has no `expiresAt` field
- ❌ **No expiration status in state** - `FactStatus` union in `state/types.ts` lacks `'expired'`
- ❌ **Expired facts remain queryable** - No expiration check in query path
- ❌ **No re-submission workflow** - Can't renew expired facts

**Impact:** Old facts with decayed confidence continue to appear as valid in query results, potentially misleading AI agents and undermining the epistemic integrity of the knowledge layer.

---

## Root Cause Analysis

### Type Mismatch

**`core/knowledge/types.ts` (domain types):**
```typescript
export type FactStatus = 'pending' | 'verified' | 'contested' | 'expired' | 'disproven';

export interface Fact {
  // ...
  expiresAt?: number;  // <-- EXISTS
  status: FactStatus;   // <-- Includes 'expired'
}
```

**`state/types.ts` (state layer):**
```typescript
export type FactStatus =
  | 'active'        // <-- Different names
  | 'challenged'
  | 'superseded'
  | 'revoked';      // <-- NO 'expired'

export interface FactState {
  // ...
  // expiresAt - MISSING!
  status: FactStatus;
}
```

### Existing Infrastructure

| Component | Location | Status |
|-----------|----------|--------|
| `applyConfidenceDecay()` | `state/knowledge.ts:562` | Implemented, not triggered |
| `minConfidenceThreshold: 0.3` | `core/knowledge/types.ts:663` | Defined, not enforced |
| `Fact.expiresAt` | `core/knowledge/types.ts:517` | Defined in domain type |
| `EpistemicProfile.requiresReverification` | `core/knowledge/types.ts:41` | Defined per category |
| `decayHalfLife` per category | `core/knowledge/types.ts:51-115` | Defined (null for eternal) |

### Code Path Without Expiration

```
Query Service (processIntent)
    ↓
KnowledgeStateTree.getFact()
    ↓
Returns fact regardless of:
  - Confidence level (even if < 0.3)
  - Age (even if very old)
  - expiresAt timestamp (field doesn't exist)
    ↓
AI Agent receives potentially stale fact
```

---

## Solution Design

### Approach: Lazy Expiration on Query

**Why lazy evaluation:**
- Periodic batch jobs cause latency spikes and complexity
- Lazy evaluation processes facts just-in-time when queried
- Follows existing patterns (confidence decay already iterates on access)
- No new background processes or cron jobs needed
- Consistent with blockchain's "state computed on demand" model

### Expiration Conditions

A fact is considered **expired** when ANY of these conditions are true:

1. **Explicit Expiration:** `expiresAt && currentBlock >= expiresAt`
2. **Confidence Below Threshold:** `confidence < minConfidenceThreshold` (0.3)
3. **Reverification Required But Stale:**
   - Epistemic category has `requiresReverification: true`
   - Age exceeds `2 * decayHalfLife` without re-verification

### Data Flow (After Fix)

```
Query Service (processIntent)
    ↓
UnifiedQueryService.executeOptimizedQuery()
    ↓
For each fact result:
    ├── Check isExpired(fact, currentBlock, params)
    │   ├── If expired && !includeExpired → Filter out
    │   ├── If expired && includeExpired → Mark as expired
    │   └── If not expired → Include normally
    ↓
Return filtered/marked results with expiration metadata
```

---

## Implementation Plan

### Phase 1: Type Updates

**File: `state/types.ts`**

1. Add `'expired'` to `FactStatus`:
```typescript
export type FactStatus =
  | 'active'
  | 'challenged'
  | 'superseded'
  | 'revoked'
  | 'expired';      // NEW
```

2. Add `expiresAt` to `FactState`:
```typescript
export interface FactState {
  id: string;
  claim: string;
  domain: string;
  confidence: Confidence;
  verifiedAt: bigint;
  verifierCount: number;
  challengeCount: number;
  lastChallengedAt?: bigint;
  status: FactStatus;
  referenceRoot: MerkleRoot;

  // NEW: Explicit expiration block
  expiresAt?: bigint;

  // Ontological metadata (existing)
  stratum?: OntologicalStratum;
  // ...
}
```

### Phase 2: Expiration Logic

**File: `state/knowledge.ts`**

1. Add expiration check method:
```typescript
/**
 * Check if a fact is expired
 *
 * Expiration conditions:
 * 1. Explicit: expiresAt < currentBlock
 * 2. Confidence decay: confidence < minConfidenceThreshold
 * 3. Reverification required but stale (age > 2 * decayHalfLife)
 */
isFactExpired(
  fact: FactState,
  currentBlock: bigint,
  params: ExpirationParams = DEFAULT_EXPIRATION_PARAMS
): ExpirationResult {
  // Check explicit expiration
  if (fact.expiresAt && currentBlock >= fact.expiresAt) {
    return { expired: true, reason: 'explicit_expiration' };
  }

  // Check confidence threshold
  if (fact.confidence < params.minConfidenceThreshold) {
    return { expired: true, reason: 'confidence_decay' };
  }

  // Check reverification requirement
  if (fact.stratum) {
    const profile = STRATUM_TO_EPISTEMIC_PROFILE[fact.stratum];
    if (profile?.requiresReverification && profile.decayHalfLife) {
      const age = Number(currentBlock - fact.verifiedAt);
      const staleThreshold = profile.decayHalfLife * 2;
      if (age > staleThreshold) {
        return { expired: true, reason: 'reverification_required' };
      }
    }
  }

  return { expired: false };
}
```

2. Add method to mark fact as expired:
```typescript
/**
 * Mark a fact as expired
 */
expireFact(factId: string, blockNumber: bigint, reason: string): MerkleRoot {
  return this.updateFact(factId, {
    status: 'expired',
  });
}
```

3. Modify `getFact()` to optionally check expiration:
```typescript
/**
 * Get fact by ID with optional expiration check
 */
getFact(
  factId: string,
  options?: { checkExpiration?: boolean; currentBlock?: bigint }
): FactState | null {
  const fact = this.getFactRaw(factId);
  if (!fact) return null;

  // If expiration check requested
  if (options?.checkExpiration && options.currentBlock) {
    const expResult = this.isFactExpired(fact, options.currentBlock);
    if (expResult.expired && fact.status === 'active') {
      // Lazily update status to expired
      this.expireFact(factId, options.currentBlock, expResult.reason);
      fact.status = 'expired';
    }
  }

  return fact;
}
```

4. Add bulk expiration query:
```typescript
/**
 * Get facts that are expired or near expiration
 */
getExpiredFacts(currentBlock: bigint): FactState[] {
  const expired: FactState[] = [];

  for (const [factId, fact] of this.factCache) {
    if (fact.status === 'expired') {
      expired.push(fact);
    } else if (fact.status === 'active') {
      const result = this.isFactExpired(fact, currentBlock);
      if (result.expired) {
        expired.push(fact);
      }
    }
  }

  return expired;
}
```

### Phase 3: Query Service Integration

**File: `core/knowledge/query/unified_query_service.ts`**

1. Add expiration filtering to query results:
```typescript
/**
 * Filter expired facts from query results
 */
private filterExpiredFacts(
  facts: FactResult[],
  currentBlock: bigint,
  includeExpired: boolean = false
): FactResult[] {
  return facts.map(result => {
    if (!result.fact) return result;

    const expResult = this.isFactExpired(result.fact, currentBlock);

    if (expResult.expired) {
      if (!includeExpired) {
        return null; // Filter out
      }
      // Mark as expired in result
      return {
        ...result,
        expired: true,
        expirationReason: expResult.reason,
      };
    }

    return result;
  }).filter(Boolean) as FactResult[];
}
```

2. Update `executeOptimizedQuery()` to use filter:
```typescript
// After getting facts...
const filteredFacts = this.filterExpiredFacts(
  facts,
  session.currentBlock,
  intent.includeExpired ?? false
);
```

**File: `core/knowledge/query/types.ts`**

Add expiration options to intent:
```typescript
export interface AgentIntent {
  // ... existing fields

  /** Include expired facts in results (default: false) */
  includeExpired?: boolean;
}

export interface FactResult {
  // ... existing fields

  /** Whether the fact is expired */
  expired?: boolean;

  /** Reason for expiration */
  expirationReason?: 'explicit_expiration' | 'confidence_decay' | 'reverification_required';
}
```

### Phase 4: Re-submission Workflow

**File: `state/transitions.ts`**

Add ability to renew expired facts via claim submission:
```typescript
/**
 * Check if claim is a renewal of an expired fact
 */
function isFactRenewal(claim: ClaimState, state: StateManager): string | null {
  // Check if there's an expired fact with matching claim text
  const existingFact = state.findFactByClaimText(claim.claim, claim.domain);
  if (existingFact && existingFact.status === 'expired') {
    return existingFact.id;
  }
  return null;
}

// In executeClaimSubmit or applyVerificationResult:
if (result.outcome === 'accepted') {
  const renewedFactId = isFactRenewal(claim, state);
  if (renewedFactId) {
    // Link new fact as renewal of old
    state.addFact({
      // ... new fact fields
      renewsFactId: renewedFactId,  // NEW field
    });

    // Optionally archive old fact data
    // state.archiveExpiredFact(renewedFactId);
  }
}
```

### Phase 5: Tests

**File: `tests/unit/state/fact_expiration.test.ts`**

```typescript
describe('GAP-009: Fact Expiration', () => {
  describe('Expiration Detection', () => {
    it('should detect explicitly expired facts', () => { ... });
    it('should detect confidence-decayed facts', () => { ... });
    it('should detect stale reverification-required facts', () => { ... });
    it('should NOT expire eternal category facts (formal, analytic)', () => { ... });
  });

  describe('Query Filtering', () => {
    it('should filter expired facts from results by default', () => { ... });
    it('should include expired facts when includeExpired=true', () => { ... });
    it('should mark expired facts with reason', () => { ... });
  });

  describe('Lazy Expiration', () => {
    it('should update fact status to expired on query', () => { ... });
    it('should not modify fact if not expired', () => { ... });
  });

  describe('Renewal Workflow', () => {
    it('should allow re-submission of expired fact claims', () => { ... });
    it('should link renewed fact to original', () => { ... });
  });
});
```

---

## Configuration

### Expiration Parameters

**File: `state/knowledge.ts` or `core/knowledge/types.ts`**

```typescript
export interface ExpirationParams {
  /** Minimum confidence before expiration (default: 0.3) */
  minConfidenceThreshold: number;

  /** Multiplier on decayHalfLife for staleness (default: 2.0) */
  stalenessMultiplier: number;

  /** Whether to perform lazy updates on query (default: true) */
  lazyExpiration: boolean;
}

export const DEFAULT_EXPIRATION_PARAMS: ExpirationParams = {
  minConfidenceThreshold: 0.3,    // From DEFAULT_TRUTH_PARAMETERS
  stalenessMultiplier: 2.0,       // 2x half-life = ~25% original confidence
  lazyExpiration: true,
};
```

### Epistemic Category Mapping

Facts need to map to epistemic profiles for reverification rules:

```typescript
const STRATUM_TO_EPISTEMIC_PROFILE: Record<OntologicalStratum, EpistemicProfile | null> = {
  'formal_logical': EPISTEMIC_PROFILES.formal,        // No decay
  'cryptographic': EPISTEMIC_PROFILES.protocol,       // No decay
  'empirical_physical': EPISTEMIC_PROFILES.empirical, // Decays, requires reverification
  'historical': EPISTEMIC_PROFILES.historical,        // Decays slower
  'social_consensus': EPISTEMIC_PROFILES.social,      // Decays, requires reverification
  // ... etc
};
```

---

## Verification Checklist

### Truth Gates
- [x] `FactState` type exists in `state/types.ts`
- [x] `FactStatus` type exists with current values
- [x] `applyConfidenceDecay()` exists in `KnowledgeStateTree`
- [x] `minConfidenceThreshold` defined (0.3)
- [x] `EpistemicProfile` has `decayHalfLife` and `requiresReverification`

### Understanding Gates
- [x] Understood difference between `Fact` (domain) and `FactState` (state)
- [x] Understood query service flow
- [x] Understood confidence decay mechanism
- [x] Understood epistemic category requirements

### Beauty Gates
- [x] Solution uses lazy evaluation (fits existing patterns)
- [x] Minimal type changes required
- [x] Extends existing expiration infrastructure
- [x] No new background processes

### Justice Gates
- [x] Placed in correct locations (state layer + query service)
- [x] Follows existing status transition patterns
- [x] Uses existing configuration parameters

### Creativity Gates
- [x] Territory is CHARTED - extending existing system
- [x] No new architecture required

---

## Risk Analysis

| Risk | Mitigation |
|------|------------|
| Breaking existing fact queries | Default `includeExpired: false` maintains backward compatibility |
| Performance impact from lazy checks | Expiration check is O(1), minimal overhead |
| Eternal facts getting expired | Check epistemic category before applying decay rules |
| Race condition on lazy update | State transactions already atomic |

---

## Files to Modify

1. `state/types.ts` - Add `expiresAt` field and `'expired'` status
2. `state/knowledge.ts` - Add expiration methods
3. `core/knowledge/query/types.ts` - Add expiration options to intent/result
4. `core/knowledge/query/unified_query_service.ts` - Add expiration filtering
5. `tests/unit/state/fact_expiration.test.ts` - New test file

## Files to Read (Reference)

- `core/knowledge/types.ts` - EpistemicProfile, EPISTEMIC_PROFILES
- `state/knowledge.ts` - applyConfidenceDecay pattern
- `core/knowledge/ontology/types.ts` - OntologicalStratum

---

## PP Verification

```
[PP:TERRITORY] CHARTED - Extending existing knowledge layer
[PP:UNDERSTAND] Full type mismatch and query flow mapped
[PP:PATTERN] Following lazy evaluation + existing decay patterns
[PP:PLACEMENT] State layer for storage, query service for filtering
[PP:SIGNAL] Clean - elegant integration
[PP:VERIFY] Design complete
```
