# P0 Gap Test Design

> **Status:** Design Complete
> **Created:** 2026-02-05
> **Purpose:** Comprehensive test coverage for all P0 critical gaps

---

## Executive Summary

| Gap | Implementation | Tested | Coverage | Priority |
|-----|----------------|--------|----------|----------|
| **GAP-001: Persistence** | 4,082 LOC | ~1,000 LOC | ~25% | ⚠️ Partial |
| **GAP-002: Claims** | 3,789 LOC | ~769 LOC | ~20% | ⚠️ Partial |
| **GAP-003: Disputes** | 3,925 LOC | 0 LOC | **0%** | 🚨 **CRITICAL** |
| **TOTAL** | **11,796 LOC** | **~1,769 LOC** | **~15%** | **URGENT** |

**Critical Finding:** GAP-003 (Challenge/Dispute Integration) has nearly 4,000 lines of untested code.

---

## GAP-001: Persistence Backend

### Existing Coverage ✅
**Files:** `tests/unit/storage/leveldb.test.ts`, `tests/unit/storage/persistence.test.ts`

- ✅ LevelDB CRUD operations
- ✅ Batch operations
- ✅ Range queries
- ✅ Namespaced storage isolation
- ✅ StatePersistenceManager (accounts, validators, facts)
- ✅ WriteAheadLog (WAL) transaction lifecycle
- ✅ RecoveryManager basic scenarios
- ✅ Checkpoint creation/loading

### Missing Coverage ❌

#### 1. Advanced LevelDB Features
```typescript
// tests/unit/storage/leveldb-advanced.test.ts
describe('Advanced LevelDB', () => {
  describe('Error Handling', () => {
    it('should handle disk full error gracefully');
    it('should handle file system permission errors');
    it('should handle corrupted data file');
    it('should retry on transient failures');
  });

  describe('Concurrent Access', () => {
    it('should handle multiple readers correctly');
    it('should serialize writes properly');
    it('should not deadlock under contention');
    it('should maintain consistency with parallel batch writes');
  });

  describe('Large Dataset Performance', () => {
    it('should handle 1M keys without degradation');
    it('should maintain O(log n) lookup times');
    it('should not exceed memory limits');
    it('should complete compaction in reasonable time');
  });
});
```

#### 2. WAL Crash Recovery
```typescript
// tests/unit/storage/wal-recovery.test.ts
describe('WAL Crash Recovery', () => {
  describe('Crash Simulation', () => {
    it('should recover from crash during commit');
    it('should recover from crash during prepare');
    it('should detect and handle partial write');
    it('should replay uncommitted entries after restart');
    it('should not apply rolled-back entries');
  });

  describe('Corruption Detection', () => {
    it('should detect checksum mismatch');
    it('should skip corrupt entries and continue');
    it('should log corruption for monitoring');
    it('should handle truncated entry gracefully');
  });

  describe('Long Transaction Chains', () => {
    it('should handle 1000+ operations in single transaction');
    it('should rollback entire chain on failure');
    it('should not OOM on large entries');
    it('should clean up properly on abort');
  });

  describe('Concurrent WAL Access', () => {
    it('should serialize concurrent writes');
    it('should allow concurrent reads');
    it('should not corrupt on multi-process access');
  });
});
```

#### 3. Advanced Recovery Scenarios
```typescript
// tests/unit/storage/recovery-advanced.test.ts
describe('Advanced Recovery', () => {
  describe('Partial Recovery', () => {
    it('should recover when accounts tree intact but validators corrupt');
    it('should recover when knowledge tree empty');
    it('should rebuild indexes from raw data');
    it('should detect and report missing data');
  });

  describe('Fork Recovery', () => {
    it('should recover to specific block height');
    it('should discard blocks above target height');
    it('should recompute state root after recovery');
    it('should handle multiple fork points');
  });

  describe('Checkpoint Integrity', () => {
    it('should verify checkpoint hash matches contents');
    it('should reject corrupted checkpoint');
    it('should handle missing checkpoint file');
    it('should upgrade checkpoint format');
  });

  describe('State Root Verification', () => {
    it('should verify account tree root matches computed');
    it('should verify validator tree root matches computed');
    it('should verify knowledge tree root matches computed');
    it('should detect state root mismatch');
  });
});
```

#### 4. Snapshot and Pruning (UNTESTED - 1,223 LOC)
```typescript
// tests/unit/storage/snapshot-pruning.test.ts
describe('Snapshot Management', () => {
  describe('Snapshot Creation', () => {
    it('should create snapshot at current block');
    it('should include all state trees in snapshot');
    it('should chunk large snapshots');
    it('should compress snapshot data');
    it('should generate valid snapshot hash');
    it('should not block consensus during snapshot');
  });

  describe('Snapshot Loading', () => {
    it('should restore state from snapshot');
    it('should verify snapshot integrity');
    it('should reject corrupt or partial snapshot');
    it('should handle format version mismatch');
    it('should apply incremental updates after snapshot');
  });
});

describe('Pruning Engine', () => {
  describe('Archive Strategy', () => {
    it('should keep all historical data');
    it('should not delete any blocks');
    it('should maintain all proofs');
  });

  describe('Retention Strategy', () => {
    it('should keep last N blocks');
    it('should delete blocks older than retention window');
    it('should not delete finalized checkpoints');
    it('should clean up associated WAL entries');
  });

  describe('Snapshot Strategy', () => {
    it('should only keep snapshot points');
    it('should delete intermediate blocks');
    it('should maintain proof paths to snapshots');
    it('should handle missing snapshot gracefully');
  });

  describe('Pruning Safety', () => {
    it('should verify state root before pruning');
    it('should not prune during active transactions');
    it('should rollback pruning on verification failure');
    it('should handle disk space exhaustion');
  });
});
```

---

## GAP-002: Claim Submission Workflow

### Existing Coverage ✅
**File:** `tests/unit/state/claim_transitions.test.ts` (25 tests)

- ✅ Claim submission with stake locking
- ✅ Minimum stake validation
- ✅ Reference validation
- ✅ Verification state transitions
- ✅ Accepted/rejected/contested outcomes
- ✅ Stake return/slash mechanics
- ✅ Submitter-only withdrawal

### Missing Coverage ❌

#### 1. Advanced Claim Scenarios
```typescript
// tests/unit/state/claim-advanced.test.ts
describe('Advanced Claim Scenarios', () => {
  describe('Multiple Claims', () => {
    it('should allow same submitter to have multiple pending claims');
    it('should track claims independently');
    it('should not share stake between claims');
    it('should handle batch claim submission');
    it('should reject duplicate claim text');
  });

  describe('Reference Chain Validation', () => {
    it('should reject circular references (A→B→A)');
    it('should reject self-reference');
    it('should validate deep reference chains (10+ levels)');
    it('should handle reference to expired fact');
    it('should handle reference to contested fact');
    it('should verify cross-domain references allowed');
    it('should calculate reference confidence ceiling');
  });

  describe('Stake Conflicts', () => {
    it('should handle claim submission while unstaking');
    it('should reject claim when stake is locked in dispute');
    it('should handle concurrent stake operations');
    it('should validate stake still sufficient at verification start');
  });

  describe('Domain/Stratum Classification', () => {
    it('should validate domain format');
    it('should validate stratum is known type');
    it('should apply domain-specific rules');
    it('should enforce stratum hierarchy');
  });
});
```

#### 2. Verification Flow Details
```typescript
// tests/unit/state/claim-verification-flow.test.ts
describe('Verification Flow Details', () => {
  describe('Validator Selection', () => {
    it('should select correct number of validators');
    it('should not select claim submitter as validator');
    it('should weight by stake and expertise');
    it('should handle insufficient validators available');
    it('should replace offline validators');
  });

  describe('Verification Outcomes', () => {
    it('should handle partial validator participation');
    it('should distribute rewards to participating validators');
    it('should slash non-participating validators');
    it('should handle tie votes correctly');
    it('should escalate contested claims');
  });

  describe('Verification Timeout', () => {
    it('should expire verification after deadline');
    it('should return stake on timeout');
    it('should mark validators as non-participating');
    it('should allow re-submission after timeout');
  });

  describe('Multiple Rounds', () => {
    it('should escalate contested claims to larger committee');
    it('should increase stake requirements per escalation');
    it('should track escalation history');
    it('should cap maximum escalation tiers');
  });
});
```

#### 3. Fact Creation from Claims
```typescript
// tests/unit/state/claim-fact-creation.test.ts
describe('Fact Creation from Claims', () => {
  describe('FactState Completeness', () => {
    it('should set all required FactState fields');
    it('should initialize referenceRoot correctly');
    it('should set stratum from claim');
    it('should set verifierCount from round');
    it('should set initial confidence from consensus');
  });

  describe('Duplicate Prevention', () => {
    it('should reject identical claim text');
    it('should allow minor text variations');
    it('should link related facts');
    it('should supersede outdated facts');
  });

  describe('Fact Activation', () => {
    it('should set status to active');
    it('should make fact queryable immediately');
    it('should index by domain');
    it('should emit FactCreated event');
  });
});
```

#### 4. Error Recovery
```typescript
// tests/unit/state/claim-error-recovery.test.ts
describe('Claim Error Recovery', () => {
  describe('Transaction Rollback', () => {
    it('should rollback claim on verification failure');
    it('should restore stake on rollback');
    it('should maintain index consistency');
    it('should not leave orphaned data');
  });

  describe('Stuck Claims', () => {
    it('should timeout stuck pending claims');
    it('should timeout stuck verifying claims');
    it('should allow manual intervention for stuck claims');
    it('should emit alerts for long-running claims');
  });

  describe('Resource Exhaustion', () => {
    it('should handle OOM during large reference validation');
    it('should limit reference set size');
    it('should timeout long-running validations');
  });
});
```

---

## GAP-003: Challenge/Dispute Integration

### Existing Coverage ✅
**NONE** - This is the critical gap.

### Missing Coverage ❌ (ALL - 3,925 LOC untested)

#### 1. Dispute Engine Core
```typescript
// tests/unit/core/dispute-engine.test.ts
describe('Dispute Engine', () => {
  describe('Dispute Creation', () => {
    it('should create dispute with valid parameters');
    it('should reject dispute on non-existent fact');
    it('should reject dispute on non-active fact');
    it('should enforce minimum challenge stake');
    it('should not allow duplicate disputes');
    it('should generate unique dispute ID');
    it('should emit DisputeCreated event');
  });

  describe('Evidence Commit Phase', () => {
    it('should allow challenger to commit evidence');
    it('should allow defender to commit evidence');
    it('should enforce commit deadline');
    it('should reject late commits');
    it('should verify commitment hash format');
    it('should handle commitment collision');
    it('should emit EvidenceCommitted event');
  });

  describe('Evidence Reveal Phase', () => {
    it('should verify reveal matches commitment');
    it('should reject mismatched reveal');
    it('should enforce reveal deadline');
    it('should handle no-show by one party');
    it('should validate evidence format');
    it('should emit EvidenceRevealed event');
  });

  describe('Arbitration Phase', () => {
    it('should select arbiters for dispute');
    it('should weight by expertise and stake');
    it('should exclude conflicted parties');
    it('should allow arbiters to vote');
    it('should enforce voting deadline');
    it('should handle partial arbiter participation');
    it('should emit ArbitrationStarted event');
  });

  describe('Deliberation Phase', () => {
    it('should aggregate arbiter votes');
    it('should calculate consensus threshold');
    it('should determine winner');
    it('should handle ties');
    it('should emit DeliberationComplete event');
  });

  describe('Settlement Phase', () => {
    it('should distribute rewards to winner');
    it('should slash loser stake');
    it('should pay arbiters');
    it('should handle draw settlement');
    it('should update fact confidence');
    it('should mark dispute as resolved');
    it('should emit DisputeSettled event');
  });
});
```

#### 2. Dispute State Machine
```typescript
// tests/unit/core/dispute-state-machine.test.ts
describe('Dispute State Machine', () => {
  describe('Phase Transitions', () => {
    it('should transition initiation → evidence_commit');
    it('should transition evidence_commit → evidence_reveal');
    it('should transition evidence_reveal → arbitration');
    it('should transition arbitration → deliberation');
    it('should transition deliberation → settlement');
    it('should transition settlement → resolved');
    it('should reject invalid transitions');
  });

  describe('Phase Entry Conditions', () => {
    it('should require both parties to commit before reveal');
    it('should require both reveals before arbitration');
    it('should require arbiter selection before deliberation');
    it('should require vote quorum before settlement');
  });

  describe('Timeout Handling', () => {
    it('should timeout evidence_commit phase');
    it('should timeout evidence_reveal phase');
    it('should timeout arbitration phase');
    it('should timeout deliberation phase');
    it('should award default winner on timeout');
    it('should handle timeout at each phase');
  });

  describe('Escalation Flow', () => {
    it('should allow escalation after deliberation');
    it('should reset to evidence_commit on escalation');
    it('should increase arbiter pool on escalation');
    it('should increase stake on escalation');
    it('should cap escalation tiers');
  });
});
```

#### 3. Evidence Comparison
```typescript
// tests/unit/core/evidence-comparison.test.ts
describe('Evidence Comparison', () => {
  describe('Evidence Strength', () => {
    it('should score on-chain evidence highest');
    it('should score cryptographic proof high');
    it('should score witness testimony medium');
    it('should score circumstantial evidence low');
    it('should combine multiple evidence types');
    it('should apply recency weighting');
  });

  describe('Commitment Protocol', () => {
    it('should generate commitment from evidence');
    it('should verify commitment matches reveal');
    it('should reject tampered reveal');
    it('should handle commitment collision');
    it('should enforce reveal window');
  });

  describe('Auto-Resolution', () => {
    it('should detect clear winner');
    it('should detect clear loser');
    it('should identify ambiguous cases');
    it('should not auto-resolve without threshold');
  });
});
```

#### 4. Arbitrator Selection
```typescript
// tests/unit/core/arbitrator-selection.test.ts
describe('Arbitrator Selection', () => {
  describe('VRF-Based Selection', () => {
    it('should use VRF for randomness');
    it('should be verifiable');
    it('should be unpredictable');
    it('should be deterministic given same seed');
  });

  describe('Expertise Weighting', () => {
    it('should weight by domain expertise');
    it('should weight by stake');
    it('should weight by historical accuracy');
    it('should exclude low-reputation arbiters');
  });

  describe('Conflict Detection', () => {
    it('should exclude challenger from arbiters');
    it('should exclude defender from arbiters');
    it('should exclude related parties');
    it('should exclude recently slashed arbiters');
  });

  describe('Pool Management', () => {
    it('should handle insufficient arbiters');
    it('should expand pool for escalation');
    it('should replace non-responsive arbiters');
    it('should track arbiter availability');
  });
});
```

#### 5. Escalation System
```typescript
// tests/unit/core/escalation.test.ts
describe('Escalation System', () => {
  describe('Escalation Eligibility', () => {
    it('should allow loser to escalate');
    it('should not allow winner to escalate');
    it('should enforce escalation window');
    it('should require escalation stake');
    it('should cap maximum escalations');
  });

  describe('Tier Progression', () => {
    it('should increase arbiter count per tier');
    it('should increase stake per tier');
    it('should increase time limits per tier');
    it('should apply tier-specific rules');
  });

  describe('Cost Calculation', () => {
    it('should calculate escalation cost');
    it('should increase cost exponentially');
    it('should refund on successful escalation');
    it('should burn on failed escalation');
  });
});
```

#### 6. Settlement System
```typescript
// tests/unit/core/settlement.test.ts
describe('Settlement System', () => {
  describe('Winner Payout', () => {
    it('should return winner stake');
    it('should add portion of loser stake');
    it('should deduct arbitration fees');
    it('should handle overflow protection');
  });

  describe('Loser Slash', () => {
    it('should slash loser stake');
    it('should burn portion of slash');
    it('should send portion to fee pool');
    it('should handle insufficient balance');
  });

  describe('Arbiter Rewards', () => {
    it('should distribute to correct voters');
    it('should not pay abstaining arbiters');
    it('should weight by stake');
    it('should handle no correct voters');
  });

  describe('Draw Settlement', () => {
    it('should return stakes to both parties');
    it('should pay arbiters from fee pool');
    it('should emit DrawSettled event');
  });

  describe('Fact Confidence Update', () => {
    it('should decrease confidence on challenger win');
    it('should increase confidence on defender win');
    it('should decrease slightly on draw');
    it('should revoke fact below threshold');
  });
});
```

#### 7. Challenge Integration
```typescript
// tests/unit/consensus/challenge-integration.test.ts
describe('Challenge Integration', () => {
  describe('challengeFact()', () => {
    it('should create dispute from challenge');
    it('should mark fact as challenged');
    it('should lock challenger stake');
    it('should return dispute ID');
    it('should reject challenge on non-existent fact');
    it('should reject challenge on non-active fact');
    it('should reject challenge with insufficient stake');
    it('should enforce cooldown between challenges');
  });

  describe('routeDisputeOutcomeToFact()', () => {
    it('should update fact confidence on challenger win');
    it('should update fact confidence on defender win');
    it('should revoke fact if confidence below threshold');
    it('should restore fact status after challenge upheld');
    it('should emit FactConfidenceChanged event');
    it('should emit FactRetracted event if revoked');
  });

  describe('checkActiveDisputes()', () => {
    it('should return active disputes for fact');
    it('should return empty for no disputes');
    it('should filter resolved disputes');
    it('should return dispute count');
  });

  describe('getDisputeAdjustedConfidence()', () => {
    it('should apply penalty for ongoing dispute');
    it('should scale penalty by dispute count');
    it('should not penalize resolved disputes');
    it('should return original confidence if no disputes');
  });
});
```

#### 8. Full Dispute Lifecycle Integration
```typescript
// tests/integration/dispute/lifecycle.test.ts
describe('Full Dispute Lifecycle', () => {
  it('should complete dispute: initiation → evidence → arbitration → settlement');
  it('should handle escalation: deliberation → escalate → evidence → settlement');
  it('should handle challenger timeout: commit timeout → defender wins');
  it('should handle defender timeout: reveal timeout → challenger wins');
  it('should handle arbiter timeout: deliberation timeout → draw');
  it('should update fact after dispute resolution');
  it('should distribute funds correctly');
  it('should persist dispute state through restart');
  it('should handle concurrent disputes on same fact');
  it('should handle dispute while claim verifying');
});
```

---

## Test Priority Matrix

### Immediate (P0 - This Week)
| Test File | Target LOC | Reason |
|-----------|------------|--------|
| `dispute-engine.test.ts` | 500+ | **0% coverage on 3,925 LOC** |
| `dispute-state-machine.test.ts` | 300+ | Phase transitions critical |
| `challenge-integration.test.ts` | 200+ | Fact-dispute link untested |

### High Priority (P1 - Next Sprint)
| Test File | Target LOC | Reason |
|-----------|------------|--------|
| `snapshot-pruning.test.ts` | 300+ | 1,223 LOC untested |
| `wal-recovery.test.ts` | 200+ | Crash recovery critical |
| `claim-verification-flow.test.ts` | 200+ | Verification details untested |

### Medium Priority (P2 - Backlog)
| Test File | Target LOC | Reason |
|-----------|------------|--------|
| `recovery-advanced.test.ts` | 150+ | Fork recovery |
| `claim-advanced.test.ts` | 150+ | Reference chains |
| `evidence-comparison.test.ts` | 150+ | Evidence scoring |
| `arbitrator-selection.test.ts` | 150+ | VRF selection |

---

## Implementation Estimate

| Gap | New Tests Needed | Estimated Time |
|-----|------------------|----------------|
| GAP-001 (Persistence) | 4 files, ~150 tests | 3-4 days |
| GAP-002 (Claims) | 4 files, ~100 tests | 2-3 days |
| GAP-003 (Disputes) | 8 files, ~400 tests | 6-8 days |
| **Total** | **16 files, ~650 tests** | **11-15 days** |

---

## PP Verification

```
[PP:TERRITORY] CHARTED - Tests follow established patterns
[PP:UNDERSTAND] Full coverage analysis complete
[PP:PATTERN] Following existing test file structure
[PP:PLACEMENT] tests/unit/{module}/*.test.ts
[PP:SIGNAL] Clean - design complete
[PP:VERIFY] Design ready for implementation
```
