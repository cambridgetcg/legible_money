# Claim Submission Workflow Resolution

> **PP Analysis Document**
> **Gap:** GAP-002 (Claim Submission Workflow)
> **Status:** Design Complete
> **Created:** 2026-02-05

---

## Executive Summary

The claim submission workflow has **good building blocks** but **critical gaps in layer connections**. This document designs the resolution to create a complete, traceable flow from user submission to verified fact.

### The Problem

```
CURRENT STATE (Disconnected):

User ──► submitClaim() ──► Memory Queue ──► Verification ──► ??? ──► Fact (maybe)
              │                                    │
              └── No state record                  └── No guaranteed state update
```

### The Solution

```
PROPOSED STATE (Connected):

User ──► submitClaim() ──► ClaimRecord ──► Verification ──► FactRecord
              │                │                 │               │
              ▼                ▼                 ▼               ▼
         claim_submit    ClaimState Tree   VerificationRound   FactState Tree
         transaction     (persistent)      (consensus)         (persistent)
```

---

## PP Verification Gates

### [PP:TRUTH] Grounding
- Verified: Claim types exist (`core/knowledge/types.ts:533-579`)
- Verified: FactState exists (`state/types.ts:174-195`)
- Verified: Verification rounds work (`consensus/pot/verification_round.ts`)
- Verified: `add_fact` handler exists but is incomplete (`state/transitions.ts:1994-2021`)
- Gap confirmed: No `ClaimState` tree, no submission persistence

### [PP:UNDERSTANDING] What Must Change
1. Add `ClaimState` to state layer (new state tree)
2. Fix `executeClaimSubmit()` to record submission
3. Create `StateIntegration` implementation
4. Enhance `executeAddFact()` with full metadata
5. Add claim status tracking throughout lifecycle

### [PP:BEAUTY] Design Principles
- **Fit**: Follow existing transaction handler patterns
- **Elegance**: Single responsibility per handler
- **Coherence**: Consistent state tree structure
- **Clarity**: Clear lifecycle states
- **Appropriateness**: Minimal changes, maximum impact

### [PP:JUSTICE] Placement
- `ClaimState` type → `state/types.ts` (alongside FactState)
- `ClaimStateTree` → `state/claims.ts` (new file, pattern from validators.ts)
- `executeClaimSubmit` fix → `state/transitions.ts:558-588`
- `StateIntegration` → `consensus/pot/state_integration.ts` (complete existing stub)
- Tests → `tests/unit/state/claim_transitions.test.ts`

### [PP:CREATIVITY] New Structure
Building new `ClaimState` tree that bridges consensus and state layers.

---

## Detailed Design

### 1. ClaimState Type

**Location:** `state/types.ts` (after FactState, ~line 196)

```typescript
/**
 * On-chain claim submission record.
 * Tracks claim from submission through verification to fact creation.
 */
export interface ClaimState {
  /** Unique claim ID (hash of claim + submitter + block) */
  id: string;

  /** Original claim text or schema-based claim data */
  claim: string;

  /** Claim type (empirical, logical, definitional, etc.) */
  claimType: string;

  /** Resolved ontological domain */
  domain: string;

  /** Resolved ontological stratum */
  stratum: OntologicalStratum;

  /** Submitter DID */
  submitter: DID;

  /** Stake locked for this claim */
  stake: bigint;

  /** Block number of submission */
  submittedAt: bigint;

  /** References to supporting facts */
  references: string[];

  /** Current status */
  status: ClaimStatus;

  /** Verification round ID (once assigned) */
  verificationRoundId?: string;

  /** Resulting fact ID (if accepted) */
  resultFactId?: string;

  /** Block when status last changed */
  statusChangedAt: bigint;

  /** Reason for rejection (if rejected) */
  rejectionReason?: string;
}

export type ClaimStatus =
  | 'pending'      // Submitted, awaiting verification round
  | 'verifying'    // In active verification round
  | 'accepted'     // Verification passed, fact created
  | 'rejected'     // Verification failed
  | 'expired'      // Timed out without verification
  | 'withdrawn';   // Submitter withdrew claim

/**
 * Payload for claim_submit transaction
 */
export interface ClaimSubmitPayload {
  type: 'claim_submit';
  claim: string;
  claimType: string;
  domain: string;
  stratum: OntologicalStratum;
  stake: bigint;
  references: string[];
  /** Optional schema ID for schema-based claims */
  schemaId?: string;
  /** Optional structured claim data */
  claimData?: Record<string, unknown>;
}

/**
 * Payload for claim_start_verification transaction (system)
 */
export interface ClaimStartVerificationPayload {
  type: 'claim_start_verification';
  claimId: string;
  roundId: string;
  selectedValidators: string[];
}

/**
 * Payload for claim_complete_verification transaction (system)
 */
export interface ClaimCompleteVerificationPayload {
  type: 'claim_complete_verification';
  claimId: string;
  roundId: string;
  outcome: 'accepted' | 'rejected' | 'contested' | 'insufficient';
  finalConfidence: number;
  verifierCount: number;
  /** Rewards to distribute */
  rewards: Map<string, bigint>;
  /** Slashes to apply */
  slashes: Map<string, bigint>;
}
```

### 2. State Transition Types Update

**Location:** `state/types.ts` (in StateTransitionType union, ~line 260)

```typescript
export type StateTransitionType =
  | 'transfer'
  | 'stake'
  | 'unstake'
  | 'delegate'
  | 'undelegate'
  // Claim lifecycle (NEW)
  | 'claim_submit'              // User submits claim
  | 'claim_start_verification'  // System starts verification round
  | 'claim_complete_verification' // System completes verification
  | 'claim_withdraw'            // User withdraws pending claim
  // Fact management
  | 'add_fact'
  | 'challenge_fact'
  | 'update_fact_confidence'
  | 'revoke_fact'
  | 'add_reference'
  // ... rest unchanged
```

### 3. StateManager Claims Extension

**Location:** `state/state_manager.ts` (add claims tree)

```typescript
export class StateManager {
  // Existing trees
  private accounts: AccountStateTree;
  private validators: ValidatorStateTree;
  private knowledge: KnowledgeStateTree;

  // NEW: Claims tree
  private claims: ClaimStateTree;

  // Claim operations
  addClaim(claim: ClaimState): void {
    this.claims.set(claim.id, claim);
    this.dirty = true;
  }

  getClaim(claimId: string): ClaimState | undefined {
    return this.claims.get(claimId);
  }

  updateClaimStatus(
    claimId: string,
    status: ClaimStatus,
    blockNumber: bigint,
    updates?: Partial<ClaimState>
  ): void {
    const claim = this.claims.get(claimId);
    if (!claim) throw new StateError(`Claim ${claimId} not found`, 'INVALID_TRANSITION');

    this.claims.set(claimId, {
      ...claim,
      ...updates,
      status,
      statusChangedAt: blockNumber,
    });
    this.dirty = true;
  }

  getPendingClaims(): ClaimState[] {
    return this.claims.getByStatus('pending');
  }

  getClaimsBySubmitter(did: DID): ClaimState[] {
    return this.claims.getBySubmitter(did);
  }
}
```

### 4. Fixed executeClaimSubmit Handler

**Location:** `state/transitions.ts` (replace lines 558-588)

```typescript
/**
 * Execute claim submission transaction.
 * Records claim in state and locks stake.
 */
function executeClaimSubmit(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as ClaimSubmitPayload;
  const senderDid = publicKeyToDid(tx.sender);

  // 1. Validate stake meets minimum
  if (payload.stake < params.minClaimStake) {
    throw new StateError(
      `Minimum claim stake is ${params.minClaimStake}, got ${payload.stake}`,
      'INVALID_TRANSITION'
    );
  }

  // 2. Validate sender has sufficient balance
  const account = state.getAccount(senderDid);
  if (!account || account.balance < payload.stake) {
    throw new StateError('Insufficient balance for claim stake', 'INSUFFICIENT_BALANCE');
  }

  // 3. Validate references exist (if provided)
  for (const refId of payload.references) {
    const refFact = state.getFact(refId);
    if (!refFact || refFact.status !== 'active') {
      throw new StateError(`Referenced fact ${refId} not found or inactive`, 'INVALID_REFERENCE');
    }
  }

  // 4. Generate claim ID
  const claimId = generateClaimId(payload.claim, senderDid, blockNumber);

  // 5. Lock stake (transfer to escrow)
  state.lockStake(senderDid, payload.stake, `claim:${claimId}`);

  // 6. Create claim record
  const claimState: ClaimState = {
    id: claimId,
    claim: payload.claim,
    claimType: payload.claimType,
    domain: payload.domain,
    stratum: payload.stratum,
    submitter: senderDid,
    stake: payload.stake,
    submittedAt: blockNumber,
    references: payload.references,
    status: 'pending',
    statusChangedAt: blockNumber,
  };

  state.addClaim(claimState);

  // 7. Emit event for consensus layer to pick up
  state.emitEvent({
    type: 'claim_submitted',
    claimId,
    submitter: senderDid,
    domain: payload.domain,
    stratum: payload.stratum,
    block: blockNumber,
  });

  // Gas: base cost + per-reference cost
  return 100000n + BigInt(payload.references.length) * 10000n;
}

/**
 * Generate deterministic claim ID.
 */
function generateClaimId(claim: string, submitter: DID, block: bigint): string {
  const input = `${claim}:${submitter}:${block}`;
  return `claim:${domainHash('LGM.claim.id.v1', Buffer.from(input)).slice(0, 32)}`;
}
```

### 5. New Handler: executeClaimStartVerification

**Location:** `state/transitions.ts` (new function)

```typescript
/**
 * System transaction: Start verification round for a claim.
 * Called by consensus engine when selecting claim for verification.
 */
function executeClaimStartVerification(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint
): bigint {
  // System transaction validation
  if (!isSystemTransaction(tx)) {
    throw new StateError('claim_start_verification is system-only', 'UNAUTHORIZED');
  }

  const payload = tx.payload as ClaimStartVerificationPayload;

  // 1. Get claim
  const claim = state.getClaim(payload.claimId);
  if (!claim) {
    throw new StateError(`Claim ${payload.claimId} not found`, 'INVALID_TRANSITION');
  }

  // 2. Validate claim is pending
  if (claim.status !== 'pending') {
    throw new StateError(
      `Claim ${payload.claimId} is ${claim.status}, expected pending`,
      'INVALID_STATE'
    );
  }

  // 3. Update claim status
  state.updateClaimStatus(payload.claimId, 'verifying', blockNumber, {
    verificationRoundId: payload.roundId,
  });

  // 4. Emit event
  state.emitEvent({
    type: 'claim_verification_started',
    claimId: payload.claimId,
    roundId: payload.roundId,
    validators: payload.selectedValidators,
    block: blockNumber,
  });

  return 0n; // System transaction, no gas
}
```

### 6. New Handler: executeClaimCompleteVerification

**Location:** `state/transitions.ts` (new function)

```typescript
/**
 * System transaction: Complete verification and create fact if accepted.
 * Called by consensus engine when verification round completes.
 */
function executeClaimCompleteVerification(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint
): bigint {
  // System transaction validation
  if (!isSystemTransaction(tx)) {
    throw new StateError('claim_complete_verification is system-only', 'UNAUTHORIZED');
  }

  const payload = tx.payload as ClaimCompleteVerificationPayload;

  // 1. Get claim
  const claim = state.getClaim(payload.claimId);
  if (!claim) {
    throw new StateError(`Claim ${payload.claimId} not found`, 'INVALID_TRANSITION');
  }

  // 2. Validate claim is verifying
  if (claim.status !== 'verifying') {
    throw new StateError(
      `Claim ${payload.claimId} is ${claim.status}, expected verifying`,
      'INVALID_STATE'
    );
  }

  // 3. Process based on outcome
  if (payload.outcome === 'accepted') {
    // Create fact
    const factId = `fact:${claim.id.replace('claim:', '')}`;

    state.addFact({
      id: factId,
      claim: claim.claim,
      domain: claim.domain,
      stratum: claim.stratum,
      confidence: payload.finalConfidence,
      verifiedAt: blockNumber,
      verifierCount: payload.verifierCount,
      challengeCount: 0,
      status: 'active',
      submitter: claim.submitter,
      verificationRoundId: payload.roundId,
      references: claim.references,
    });

    // Update claim with fact reference
    state.updateClaimStatus(payload.claimId, 'accepted', blockNumber, {
      resultFactId: factId,
    });

    // Return stake to submitter
    state.unlockStake(claim.submitter, claim.stake, `claim:${claim.id}`);

    // Emit fact created event
    state.emitEvent({
      type: 'fact_created',
      factId,
      claimId: claim.id,
      confidence: payload.finalConfidence,
      domain: claim.domain,
      stratum: claim.stratum,
      block: blockNumber,
    });

  } else if (payload.outcome === 'rejected') {
    // Slash submitter stake
    const slashAmount = claim.stake / 2n; // 50% slash
    state.slashLockedStake(claim.submitter, slashAmount, `claim:${claim.id}`);
    state.unlockStake(claim.submitter, claim.stake - slashAmount, `claim:${claim.id}`);

    state.updateClaimStatus(payload.claimId, 'rejected', blockNumber, {
      rejectionReason: 'Verification failed',
    });

    state.emitEvent({
      type: 'claim_rejected',
      claimId: claim.id,
      reason: 'verification_failed',
      block: blockNumber,
    });

  } else if (payload.outcome === 'contested' || payload.outcome === 'insufficient') {
    // Return stake, claim can be resubmitted
    state.unlockStake(claim.submitter, claim.stake, `claim:${claim.id}`);

    state.updateClaimStatus(payload.claimId, 'rejected', blockNumber, {
      rejectionReason: payload.outcome === 'contested'
        ? 'Verification contested - no consensus'
        : 'Insufficient validators participated',
    });
  }

  // 4. Distribute rewards
  for (const [validatorDid, reward] of payload.rewards) {
    state.credit(validatorDid as DID, reward);
    state.emitEvent({
      type: 'verification_reward',
      validator: validatorDid,
      amount: reward,
      claimId: claim.id,
      block: blockNumber,
    });
  }

  // 5. Apply slashes
  for (const [validatorDid, slashAmount] of payload.slashes) {
    state.slashValidator(validatorDid as DID, slashAmount, 'verification_failure');
    state.emitEvent({
      type: 'validator_slashed',
      validator: validatorDid,
      amount: slashAmount,
      reason: 'verification_failure',
      claimId: claim.id,
      block: blockNumber,
    });
  }

  return 0n; // System transaction, no gas
}
```

### 7. State Integration Implementation

**Location:** `consensus/pot/state_integration.ts` (complete implementation)

```typescript
import type { StateManager } from '../../state/state_manager.js';
import type { Transaction } from '../../state/types.js';
import type { VerificationResult, VerificationRound } from './types.js';
import type { DID } from '../../agents/identity/did.js';

/**
 * Bridge between consensus layer and state layer.
 * Creates state transitions for consensus events.
 */
export class StateIntegration {
  constructor(
    private state: StateManager,
    private createSystemTransaction: (payload: unknown) => Transaction
  ) {}

  /**
   * Start verification for a pending claim.
   * Called when consensus selects a claim for verification.
   */
  async startVerification(
    claimId: string,
    round: VerificationRound
  ): Promise<void> {
    const tx = this.createSystemTransaction({
      type: 'claim_start_verification',
      claimId,
      roundId: round.roundId,
      selectedValidators: Array.from(round.selectedValidators),
    });

    await this.state.executeTransaction(tx, round.startBlock);
  }

  /**
   * Apply verification result to state.
   * Called when verification round completes.
   */
  async applyVerificationResult(
    result: VerificationResult,
    blockNumber: bigint
  ): Promise<void> {
    const tx = this.createSystemTransaction({
      type: 'claim_complete_verification',
      claimId: result.claimId,
      roundId: result.roundId,
      outcome: result.outcome,
      finalConfidence: result.finalConfidence,
      verifierCount: result.verifierCount,
      rewards: result.rewards,
      slashes: result.slashes,
    });

    await this.state.executeTransaction(tx, blockNumber);
  }

  /**
   * Get pending claims ready for verification.
   * Called by consensus to select next claims.
   */
  getPendingClaims(): Array<{
    claimId: string;
    domain: string;
    stratum: string;
    submittedAt: bigint;
    stake: bigint;
  }> {
    return this.state.getPendingClaims().map(claim => ({
      claimId: claim.id,
      domain: claim.domain,
      stratum: claim.stratum,
      submittedAt: claim.submittedAt,
      stake: claim.stake,
    }));
  }

  /**
   * Check if a claim exists and is valid.
   */
  isValidClaim(claimId: string): boolean {
    const claim = this.state.getClaim(claimId);
    return claim !== undefined && claim.status === 'pending';
  }
}
```

### 8. Enhanced FactState

**Location:** `state/types.ts` (update FactState, lines 174-195)

```typescript
export interface FactState {
  /** Unique fact ID */
  id: string;

  /** The verified claim text */
  claim: string;

  /** Ontological domain (e.g., "physics.relativity") */
  domain: string;

  /** Ontological stratum */
  stratum: OntologicalStratum;

  /** Current confidence level (0.0 - 1.0) */
  confidence: number;

  /** Block when fact was verified */
  verifiedAt: bigint;

  /** Number of validators who verified */
  verifierCount: number;

  /** Number of challenges received */
  challengeCount: number;

  /** Current status */
  status: 'active' | 'challenged' | 'superseded' | 'revoked';

  // NEW FIELDS

  /** Original claim submitter */
  submitter: DID;

  /** Verification round that created this fact */
  verificationRoundId: string;

  /** References to supporting facts */
  references: string[];

  /** Root hash of reference Merkle tree */
  referenceRoot?: string;

  /** Last confidence update block */
  lastConfidenceUpdate?: bigint;

  /** Confidence at last decay application */
  confidenceAtLastDecay?: number;
}
```

---

## Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        CLAIM SUBMISSION WORKFLOW                             │
└─────────────────────────────────────────────────────────────────────────────┘

  USER                    STATE LAYER                CONSENSUS LAYER
    │                          │                           │
    │  1. Submit Claim         │                           │
    │─────────────────────────►│                           │
    │                          │                           │
    │                    ┌─────┴─────┐                     │
    │                    │claim_submit│                    │
    │                    │transaction │                    │
    │                    └─────┬─────┘                     │
    │                          │                           │
    │                    ┌─────▼─────┐                     │
    │                    │ClaimState │                     │
    │                    │ status:   │                     │
    │                    │ 'pending' │                     │
    │                    └─────┬─────┘                     │
    │                          │                           │
    │                          │  2. Event: claim_submitted│
    │                          │──────────────────────────►│
    │                          │                           │
    │                          │     3. Select for         │
    │                          │        verification       │
    │                          │◄──────────────────────────│
    │                          │                           │
    │                    ┌─────┴─────────┐                 │
    │                    │claim_start_   │                 │
    │                    │verification   │                 │
    │                    └─────┬─────────┘                 │
    │                          │                           │
    │                    ┌─────▼─────┐                     │
    │                    │ClaimState │          ┌──────────┴──────────┐
    │                    │ status:   │          │VerificationRound    │
    │                    │'verifying'│◄────────►│ commit/reveal/agg   │
    │                    └─────┬─────┘          └──────────┬──────────┘
    │                          │                           │
    │                          │  4. Verification complete │
    │                          │◄──────────────────────────│
    │                          │                           │
    │                    ┌─────┴─────────────┐             │
    │                    │claim_complete_    │             │
    │                    │verification       │             │
    │                    └─────┬─────────────┘             │
    │                          │                           │
    │              ┌───────────┼───────────┐               │
    │              │           │           │               │
    │              ▼           ▼           ▼               │
    │         ACCEPTED    REJECTED    CONTESTED            │
    │              │           │           │               │
    │        ┌─────▼─────┐    │     Return stake           │
    │        │FactState  │    │                            │
    │        │ created   │  Slash                          │
    │        │           │  stake                          │
    │        └───────────┘                                 │
    │                                                      │
    │  5. Query fact                                       │
    │─────────────────────────────────────────────────────►│
    │                                                      │
```

---

## Test Plan

### Unit Tests: `tests/unit/state/claim_transitions.test.ts`

```typescript
describe('Claim Submission Transitions', () => {
  describe('executeClaimSubmit', () => {
    it('should create ClaimState with pending status');
    it('should lock stake from submitter');
    it('should validate minimum stake requirement');
    it('should validate references exist');
    it('should generate deterministic claim ID');
    it('should emit claim_submitted event');
    it('should reject insufficient balance');
  });

  describe('executeClaimStartVerification', () => {
    it('should update claim status to verifying');
    it('should record verification round ID');
    it('should reject non-pending claims');
    it('should reject non-system transactions');
  });

  describe('executeClaimCompleteVerification', () => {
    describe('accepted outcome', () => {
      it('should create FactState with full metadata');
      it('should update claim status to accepted');
      it('should return stake to submitter');
      it('should record factId on claim');
      it('should distribute rewards to validators');
    });

    describe('rejected outcome', () => {
      it('should slash submitter stake by 50%');
      it('should return remaining stake');
      it('should record rejection reason');
      it('should apply validator slashes');
    });

    describe('contested outcome', () => {
      it('should return full stake');
      it('should mark claim as rejected with reason');
    });
  });
});

describe('StateIntegration', () => {
  it('should create system transactions for startVerification');
  it('should create system transactions for applyVerificationResult');
  it('should return pending claims for selection');
});
```

### Integration Tests: `tests/integration/consensus/claim_flow.test.ts`

```typescript
describe('Complete Claim Flow', () => {
  it('should flow from submission to verified fact');
  it('should handle rejection with proper stake handling');
  it('should preserve ontological context through flow');
  it('should track claim through all status transitions');
});
```

---

## Implementation Order

### Phase 1: Types (1 day)
1. Add `ClaimState`, `ClaimStatus` to `state/types.ts`
2. Add payload types for new transitions
3. Update `StateRootComponents` to include claims tree

### Phase 2: State Layer (2 days)
1. Create `state/claims.ts` with `ClaimStateTree`
2. Update `StateManager` with claim operations
3. Implement `executeClaimSubmit` (fix existing)
4. Implement `executeClaimStartVerification` (new)
5. Implement `executeClaimCompleteVerification` (new)

### Phase 3: Consensus Integration (1 day)
1. Complete `StateIntegration` class
2. Wire into `PoTEngine`
3. Update `submitClaim()` to use state layer

### Phase 4: Testing (2 days)
1. Unit tests for all handlers
2. Integration test for complete flow
3. Edge case testing (insufficient validators, timeouts)

**Total Estimate:** 6 days

---

## Migration Notes

### Backward Compatibility
- Existing `add_fact` handler remains for direct fact insertion (admin/genesis)
- New `claim_complete_verification` is the normal path for user claims

### State Migration
- No migration needed - claims tree is new
- Existing facts remain unchanged

### API Changes
- `Engine.submitClaim()` return type changes to include `claimId`
- New RPC methods: `getClaim`, `getClaimsBySubmitter`, `getClaimStatus`

---

## PP Verification Checklist

- [x] **TRUTH**: All file locations and line numbers verified
- [x] **UNDERSTANDING**: Complete flow documented with gaps identified
- [x] **BEAUTY**: Follows existing patterns, minimal surface area
- [x] **JUSTICE**: Placed in correct modules (state for state, consensus for consensus)
- [x] **CREATIVITY**: New ClaimState tree bridges the gap elegantly

---

## References

- `core/knowledge/types.ts:533-579` - ClaimSubmission type
- `state/types.ts:174-195` - Existing FactState
- `state/transitions.ts:558-588` - Current executeClaimSubmit (to fix)
- `state/transitions.ts:1994-2021` - Current executeAddFact (to enhance)
- `consensus/pot/verification_round.ts:112-546` - Verification mechanics
- `consensus/pot/engine.ts:524-608` - submitClaim entry point
- `consensus/pot/engine.ts:857-902` - applyVerificationResult (to connect)
