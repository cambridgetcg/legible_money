# GAP-006: Verification Reward Routing - Resolution Design

> **Status:** Design Complete
> **Created:** 2026-02-05
> **Gap Priority:** P1 (Required for full functionality)
> **Estimated Effort:** Small (3-4 days)

---

## Problem Statement

When a verification round completes successfully (outcome === "accepted"):
- ✅ **Validators receive rewards** - 1 LGM each, routed through reward router (22% research, 78% validator)
- ✅ **Incorrect validators are slashed** - 11% of stake
- ✅ **Fact is created** - added to knowledge layer
- ❌ **Claim submitter receives NOTHING** - despite `claimReward: 5 LGM` existing in consensus params

The claim reward (5 LGM) is defined but never distributed. Claim submitters have no incentive to contribute knowledge.

---

## Root Cause Analysis

### Code Path

```
Verification Round Completes (verification_round.ts:aggregateResults)
    ↓
VerificationResult created with:
    - claimId, outcome, finalConfidence
    - rewards (validators ONLY)
    - slashes
    - fact (if accepted)
    ↓
applyVerificationResult (state/transitions.ts:2344)
    ↓
For each validator reward → routeVerificationReward → transfer
For each slash → slashValidator
If accepted → addFact
    ↓
NOTHING for claim submitter ← THE GAP
```

### What's Missing

1. **No claim lookup** - `applyVerificationResult` doesn't look up the ClaimState to get submitter
2. **No vesting creation** - `createVesting()` exists but is never called
3. **No reward routing** - `claimReward` (5 LGM) exists but is never used

### What Already Exists

| Component | Location | Status |
|-----------|----------|--------|
| `claimReward: 5n * 10n ** 18n` | `consensus/pot/types.ts:85` | Defined, unused |
| `createVesting()` | `core/rewards/vesting_engine.ts:51` | Implemented, never called |
| `determineVestingCategory()` | `core/rewards/vesting_engine.ts:370` | Implemented, never called |
| `VestingStateTree` | `state/vesting.ts` | Implemented |
| `vesting_create` handler | `state/transitions.ts` | Implemented |
| `ClaimState.submitter` | `state/types.ts:229` | Contains submitter DID |
| `ClaimState.claimType` | `state/types.ts:220` | For vesting category |
| `routeReward()` | `core/rewards/reward_router.ts` | For 22% research split |

---

## Solution Design

### Approach: Extend applyVerificationResult

**Why this approach:**
- Minimal type changes (no changes to `VerificationResult`)
- ClaimState already contains all needed data
- Follows existing patterns (state lookup already used)
- Single integration point

### Data Flow (After Fix)

```
applyVerificationResult
    │
    ├── Route validator rewards (existing)
    │   └── 22% research, 78% validator
    │
    ├── Apply slashes (existing)
    │
    ├── If accepted:
    │   │
    │   ├── Add fact (existing)
    │   │
    │   └── NEW: Create claim submitter vesting
    │       │
    │       ├── Lookup ClaimState by claimId
    │       ├── Get submitter, claimType, domain
    │       ├── Determine vesting category
    │       ├── Route claimReward (22% research, 78% vesting)
    │       ├── Create vesting schedule
    │       └── Store in VestingStateTree
    │
    └── If rejected:
        └── Return 50% of stake (existing in claim transitions)
```

---

## Implementation Plan

### Phase 1: Core Integration

**File: `state/transitions.ts`**

1. Import vesting functions:
```typescript
import {
  createVesting,
  determineVestingCategory,
  createVestingEvent,
} from '../core/rewards/vesting_engine.js';
import type { VestingCategory } from '../core/rewards/types.js';
```

2. Modify `applyVerificationResult` to create vesting after fact creation:

```typescript
// If claim accepted, add fact to knowledge AND create vesting
if (result.outcome === 'accepted' && result.fact) {
  // ... existing fact creation code ...

  // NEW: Create vesting for claim submitter
  const claim = state.getClaim(result.claimId);
  if (claim) {
    // Determine vesting category from claim properties
    const evidenceTypes = inferEvidenceTypes(claim);
    const category = determineVestingCategory(
      claim.claimType,
      evidenceTypes,
      false, // Not replicated initially
      false  // Not contested
    );

    // Route claim reward (22% research, 78% to vesting)
    const claimRewardAmount = params.claimReward;
    const routed = routeReward({
      source: 'claim',
      amount: claimRewardAmount,
      recipient: claim.submitter,
      blockNumber,
      claimId: result.claimId,
    });

    // Transfer research share
    if (routed.researchShare > 0n) {
      state.transfer(REWARD_POOL_DID, RESEARCH_TREASURY_DID, routed.researchShare);
    }

    // Create vesting schedule with remaining amount
    if (routed.recipientAmount > 0n) {
      const vesting = createVesting({
        claimId: result.claimId,
        recipient: claim.submitter,
        amount: routed.recipientAmount,
        category,
      }, blockNumber);

      state.addVesting(vesting);
    }

    // Update claim status to accepted (if not already)
    state.updateClaimStatus(result.claimId, 'accepted', blockNumber, {
      resultFactId: result.claimId,
    });
  }
}
```

### Phase 2: Evidence Type Inference

**File: `state/transitions.ts`** (helper function)

```typescript
/**
 * Infer evidence types from claim properties for vesting category
 */
function inferEvidenceTypes(claim: ClaimState): string[] {
  const types: string[] = [];

  // Infer from claim type
  switch (claim.claimType) {
    case 'formal':
      types.push('formal_proof');
      break;
    case 'on_chain':
      types.push('on_chain');
      break;
    case 'computational':
      types.push('computational');
      break;
    case 'empirical':
      types.push('peer_reviewed'); // Default for empirical
      break;
    case 'attestation':
      types.push('attestation');
      break;
  }

  // Could also infer from stratum
  if (claim.stratum === 'formal_logical') {
    types.push('formal_proof');
  } else if (claim.stratum === 'cryptographic') {
    types.push('cryptographic_proof');
  }

  // Default if nothing inferred
  if (types.length === 0) {
    types.push('peer_reviewed');
  }

  return types;
}
```

### Phase 3: State Manager Methods

**Verify `state/state_manager.ts` has:**

```typescript
// Vesting state access
addVesting(vesting: RewardVesting): void;
getVesting(vestingId: string): RewardVesting | undefined;
updateVesting(vesting: RewardVesting): void;
```

If missing, add them with VestingStateTree integration.

### Phase 4: Tests

**File: `tests/unit/state/verification_reward_routing.test.ts`**

```typescript
describe('Verification Reward Routing', () => {
  it('should create vesting for claim submitter when claim is accepted', async () => {
    // Setup: Create claim, run verification round
    const claim = createTestClaim('did:lgm:submitter', 'Test claim');
    state.addClaim(claim);

    // Execute: Apply verification result with outcome=accepted
    const result: VerificationResult = {
      claimId: claim.id,
      outcome: 'accepted',
      finalConfidence: 0.85,
      // ...
    };
    applyVerificationResult(state, result, blockNumber, params);

    // Verify: Vesting was created
    const vestings = state.getVestingsByRecipient('did:lgm:submitter');
    expect(vestings).toHaveLength(1);
    expect(vestings[0].claimId).toBe(claim.id);
    expect(vestings[0].totalAmount).toBe(params.claimReward * 78n / 100n);
  });

  it('should route 22% of claim reward to research treasury', async () => {
    // ...
  });

  it('should determine correct vesting category based on claim type', async () => {
    // ...
  });

  it('should NOT create vesting when claim is rejected', async () => {
    // ...
  });
});
```

---

## Verification Checklist

### Truth Gates
- [x] `claimReward` exists in `DEFAULT_CONSENSUS_PARAMS` (verified: types.ts:85)
- [x] `createVesting()` exists and works (verified: vesting_engine.ts:51)
- [x] `ClaimState` contains submitter and metadata (verified: types.ts:212-257)
- [x] Reward router exists (verified: reward_router.ts)

### Understanding Gates
- [x] Understood full verification flow
- [x] Understood why vesting exists (gradual release, challenge reserves)
- [x] Understood reward router purpose (22% research share)

### Beauty Gates
- [x] Solution fits existing patterns (state lookup, reward routing)
- [x] Minimal code changes
- [x] No new types required

### Justice Gates
- [x] Placed in `applyVerificationResult` (right location)
- [x] Uses existing components (reward router, vesting engine)
- [x] Follows 22% research share rule (consistency)

### Creativity Gates
- [x] Territory is CHARTED - extending existing system
- [x] No new architecture required

---

## Risk Analysis

| Risk | Mitigation |
|------|------------|
| Double vesting creation | Check if vesting already exists for claimId |
| Missing claim in state | Claim should exist; log warning if not |
| Reward pool insufficient | Same risk as validator rewards; existing safeguard |
| Race condition | State transactions already atomic |

---

## Dependencies

- GAP-002 (Claim Submission) - **Complete ✅**
- VestingStateTree implementation - **Exists**
- StateManager.addVesting() - **Needs verification**

---

## Estimated Implementation

| Task | Time |
|------|------|
| Phase 1: Core integration | 2-3 hours |
| Phase 2: Evidence inference | 1 hour |
| Phase 3: State manager verification | 1 hour |
| Phase 4: Tests | 2-3 hours |
| **Total** | **6-8 hours** |

---

## Files to Modify

1. `state/transitions.ts` - Main integration point
2. `state/state_manager.ts` - Add/verify vesting methods if needed
3. `tests/unit/state/verification_reward_routing.test.ts` - New test file

## Files to Read (Reference)

- `core/rewards/vesting_engine.ts` - createVesting, determineVestingCategory
- `core/rewards/types.ts` - VestingCategory, CreateVestingPayload
- `core/rewards/reward_router.ts` - routeReward
- `consensus/pot/types.ts` - DEFAULT_CONSENSUS_PARAMS.claimReward

---

## PP Verification

```
[PP:TERRITORY] CHARTED - Extending existing verification flow
[PP:UNDERSTAND] Full data flow mapped
[PP:PATTERN] Following existing reward routing pattern
[PP:PLACEMENT] applyVerificationResult is the correct location
[PP:SIGNAL] Clean - elegant integration
[PP:VERIFY] Design complete
```
