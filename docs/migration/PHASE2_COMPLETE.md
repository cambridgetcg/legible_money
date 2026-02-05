# Phase 2: Interface Definition - COMPLETE

**Date**: 2026-02-03
**Status**: COMPLETE

## Summary

Created domain and infrastructure contracts (interfaces) for all major modules. These interfaces enable dependency injection and decouple modules from their implementations.

## Created Files

```
foundation/contracts/
├── index.ts          # Public API (100+ exports)
├── state.ts          # StateProvider, AccountReader/Writer, ValidatorReader/Writer, KnowledgeReader/Writer
├── events.ts         # EventBus, ProtocolEvent, EventHandler
├── knowledge.ts      # KnowledgeService, Fact, Claim types
├── consensus.ts      # ConsensusEngine, VerificationRound, Block types
├── rewards.ts        # RewardsEngine, VestingSchedule, Clawback
├── disputes.ts       # DisputeResolver, Dispute, Settlement
└── identity.ts       # IdentityService, AgentAccount, SessionKey
```

## Contract Summary

### Infrastructure Contracts

| Contract | Purpose | Key Methods |
|----------|---------|-------------|
| **StateProvider** | State access abstraction | getBalance, transfer, getValidator, getFact |
| **EventBus** | Cross-module communication | emit, on, once, waitFor |

### Domain Contracts

| Contract | Purpose | Key Methods |
|----------|---------|-------------|
| **KnowledgeService** | Facts & claims | submitClaim, getFact, challengeFact, updateConfidence |
| **ConsensusEngine** | PoT verification | startRound, submitCommitment, createBlock, processBlock |
| **RewardsEngine** | Incentives | createVestingSchedule, claimVested, executeClawback |
| **DisputeResolver** | Arbitration | initiateDispute, castVote, executeSettlement |
| **IdentityService** | DIDs & keys | createAccount, rotateOperationalKey, authorizeSessionKey |

## Interface Segregation (MOD-2)

Each contract is split into focused interfaces:

```typescript
// State is split by concern
AccountReader   // Read account data
AccountWriter   // Modify account data
ValidatorReader // Read validator data
ValidatorWriter // Modify validator data
KnowledgeReader // Read facts
KnowledgeWriter // Modify facts
StateProvider   // Combined (for full access)
StateReader     // Combined read-only
```

```typescript
// Each domain has reader variant
KnowledgeService      // Full interface
KnowledgeServiceReader // Read-only

ConsensusEngine       // Full interface
ConsensusReader       // Read-only

// etc.
```

## Event-Driven Communication (FLOW-2)

Defined protocol events for cross-module communication:

```typescript
type ProtocolEventType =
  | 'block_proposed' | 'block_finalized'
  | 'round_started' | 'round_completed'
  | 'fact_verified' | 'fact_challenged'
  | 'validator_slashed'
  | 'dispute_initiated' | 'dispute_resolved'
  | 'reward_distributed' | 'clawback_executed'
  // ... 20+ event types
```

## Architecture Invariants Satisfied

- [x] **ARCH-3**: Dependency Injection - All interfaces defined, no direct instantiation
- [x] **MOD-2**: Interface Segregation - Split into focused reader/writer interfaces
- [x] **FLOW-2**: Event-Driven - EventBus contract with typed events

## Type Safety

All contracts use foundation primitive types:
- `DID` for identity
- `PublicKey`, `Signature` for crypto
- `Amount`, `Stake` for economics
- `BlockHeight`, `Confidence` for protocol

## Verification

```bash
# Type check passes
npx tsc --noEmit -p tsconfig.json 2>&1 | grep "foundation/"
# (no errors)

# Property tests pass
npx vitest run tests/property/crypto/signatures.prop.ts
# ✓ 16 tests passed
```

## Usage Example

```typescript
import {
  type StateProvider,
  type KnowledgeService,
  type EventBus,
  createEvent,
} from './foundation/index.js';

// Domain module receives interfaces via injection
class KnowledgeModule {
  constructor(
    private state: StateProvider,
    private events: EventBus
  ) {}

  async submitClaim(claim: ClaimSubmission, submitter: DID) {
    // Use injected state provider
    const balance = await this.state.getBalance(submitter);
    if (balance < claim.stake) {
      throw new Error('Insufficient balance');
    }

    // ... process claim

    // Emit event for other modules
    this.events.emit(createEvent('claim_submitted', 'knowledge', {
      claimId,
      submitter,
      claim: claim.claim,
      domain: claim.domain,
      stake: claim.stake,
    }));
  }
}
```

## Next Steps (Phase 3-4)

### Phase 3: Dependency Inversion
- Create factory functions for each domain module
- Wire up dependency injection in application layer
- Update tests to use mock implementations

### Phase 4: Event-Driven Refactor
- Implement EventBus (in-memory for start)
- Replace direct calls with event emission
- Add event persistence for replay
