# GAP-005: Payment Channels Implementation Design

> **Status:** Design Complete
> **Created:** 2026-02-05
> **Author:** Claude (PP-guided design)
> **Related:** GAP-007 (Per-Message Billing depends on this)

---

## PP Verification Gates

### Truth Gates
- [x] **Citation**: All existing code referenced with file:line
- [x] **Grounding**: Payment channel pattern is well-established (Lightning Network, state channels)
- [x] **Evidence**: `PaymentChannel` type already defined at `core/billing/types.ts:106-127`

### Understanding Gates
- [x] **Mental Model**: State channels with off-chain updates and on-chain settlement
- [x] **Why**: Reduces transaction overhead for streaming API billing
- [x] **Boundaries**: Not a general-purpose L2 - specific to API billing

### Beauty Gates
- [x] **Fit**: Follows existing state tree patterns (`ClaimStateTree`)
- [x] **Elegance**: Reuses existing `PaymentChannel` and `PaymentDistribution` types
- [x] **Coherence**: Integrates with existing billing calculation in `truth_gateway.ts`

### Justice Gates
- [x] **Placement**: State layer for channels, transitions for handlers
- [x] **Ownership**: `state/payment_channels.ts` owns channel state
- [x] **Integrity**: Follows existing transaction handler patterns

### Creativity Gates (Uncharted Territory)
- [x] **Pioneering**: Off-chain signed state updates (new to codebase)
- [x] **Path-Building**: Clear upgrade path from current session billing
- [x] **Integration**: Connects to existing `QueryCostEstimator` and `BillingParameters`

---

## Executive Summary

Payment channels enable **streaming micropayments** for long-running API sessions without per-call transaction overhead. The design follows a **state channel pattern**:

1. **Open**: Deposit funds on-chain, create channel
2. **Use**: Off-chain state updates (signed by both parties)
3. **Settle**: Periodic on-chain settlement of accumulated distributions
4. **Close**: Cooperative or unilateral close with timelock

---

## Part I: What Exists

### Types (Complete)
```
core/billing/types.ts:106-127 - PaymentChannel interface
core/billing/types.ts:43-58   - PaymentDistribution interface
core/billing/types.ts:149-170 - APISettlement type
```

### SDK Methods (Defined, need backend)
```
sdk/typescript/index.ts:522-539 - openPaymentChannel()
sdk/typescript/index.ts:544-556 - closePaymentChannel()
sdk/typescript/index.ts:561-563 - getPaymentChannel()
```

### Billing Logic (Complete)
```
core/billing/distribution.ts     - calculateAPICost(), buildPaymentDistribution()
core/billing/truth_gateway.ts    - calculateQueryPrice(), calculateDistribution()
core/knowledge/query/cost_estimator.ts - QueryCostEstimator class
```

### Documentation (Complete)
```
docs/API_BILLING.md:357-410 - Streaming payments design
```

---

## Part II: What's Missing

### 1. Transaction Types
- `open_payment_channel` - not in `TransactionType` union
- `close_payment_channel` - not in `TransactionType` union
- `settle_payment_channel` - new (periodic settlement)
- `dispute_payment_channel` - new (fraud proof)

### 2. State Layer
- `PaymentChannelStateTree` - no SMT for channels
- Channel state persistence
- Indexes: by payer, by provider, by status

### 3. Transaction Handlers
- `executeOpenPaymentChannel()` - no handler
- `executeClosePaymentChannel()` - no handler
- `executeSettlePaymentChannel()` - no handler
- `executeDisputePaymentChannel()` - no handler

### 4. Off-Chain State
- Signed state proof format
- State update protocol
- Nonce-based ordering

### 5. RPC Handlers
- `lgm_getPaymentChannel` - not implemented
- `lgm_openPaymentChannel` - not implemented
- `lgm_closePaymentChannel` - not implemented

---

## Part III: Architecture

### State Channel Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    PAYMENT CHANNEL LIFECYCLE                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. OPEN (on-chain)                                             │
│     ┌──────────┐                                                │
│     │  Payer   │──── open_payment_channel tx ────►┌──────────┐  │
│     │          │     (deposit: 10 LGM)            │  Chain   │  │
│     │          │◄── channel created ──────────────│          │  │
│     └──────────┘                                  └──────────┘  │
│          │                                                       │
│  2. USE (off-chain)                                             │
│          │                                                       │
│     ┌────▼─────┐                      ┌──────────┐              │
│     │  Payer   │◄─── API response ────│ Provider │              │
│     │          │                      │          │              │
│     │          │── signed state ─────►│          │              │
│     │          │   (nonce: 1,         │          │              │
│     │          │    spent: 0.01 LGM)  │          │              │
│     │          │                      │          │              │
│     │          │◄── countersigned ────│          │              │
│     │          │    state proof       │          │              │
│     └──────────┘                      └──────────┘              │
│          │              ... many API calls ...                   │
│          │                                                       │
│  3. SETTLE (on-chain, periodic)                                 │
│     ┌──────────┐                                                │
│     │ Provider │── settle_payment_channel tx ───►┌──────────┐   │
│     │          │   (latest signed state,         │  Chain   │   │
│     │          │    distributions)               │          │   │
│     │          │◄── settlement recorded ─────────│          │   │
│     └──────────┘                                 └──────────┘   │
│                                                                  │
│  4. CLOSE (on-chain)                                            │
│     ┌──────────┐                                                │
│     │  Either  │── close_payment_channel tx ───►┌──────────┐    │
│     │  Party   │   (final state)                │  Chain   │    │
│     │          │◄── channel closed, ────────────│          │    │
│     │          │    funds distributed           │          │    │
│     └──────────┘                                └──────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Off-Chain State Update Protocol

```
┌─────────────────────────────────────────────────────────────────┐
│                    STATE UPDATE PROTOCOL                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  SIGNED STATE PROOF:                                            │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ {                                                          │ │
│  │   channelId: "pc:abc123...",                               │ │
│  │   nonce: 42,                          // Monotonic         │ │
│  │   spent: 1500000000000000000n,        // 1.5 LGM          │ │
│  │   available: 8500000000000000000n,    // 8.5 LGM          │ │
│  │   pendingDistributions: [...],        // Accumulated       │ │
│  │   payerSignature: "...",              // Payer signs       │ │
│  │   providerSignature: "..."            // Provider co-signs │ │
│  │ }                                                          │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  RULES:                                                         │
│  - Nonce must be > previous nonce                               │
│  - spent + available = deposited (invariant)                    │
│  - Both signatures required for valid state                     │
│  - Latest nonce wins in disputes                                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Part IV: Type Definitions

### New Types (`state/types.ts`)

```typescript
/**
 * Payment channel status
 */
export type PaymentChannelStatus = 'open' | 'closing' | 'disputed' | 'closed';

/**
 * On-chain payment channel state
 * Extends core/billing/types.ts:PaymentChannel with state layer fields
 */
export interface PaymentChannelState {
  // Identity
  channelId: string;
  payer: string;        // DID
  provider: string;     // DID

  // Funds (all bigint for precision)
  deposited: bigint;
  spent: bigint;
  available: bigint;

  // State
  status: PaymentChannelStatus;
  openedAt: bigint;     // Block number
  expiresAt: bigint;    // Block number

  // Off-chain state tracking
  lastNonce: bigint;    // Latest confirmed nonce
  lastStateHash: string; // Hash of latest confirmed state

  // Pending distributions (not yet settled)
  pendingDistributions: SerializedPaymentDistribution[];
  totalDistributed: bigint;

  // Settlement
  lastSettlementBlock: bigint;
  settlementFrequency: bigint; // Blocks between settlements (default: 100)

  // Dispute handling
  disputeStartBlock?: bigint;
  disputeDeadline?: bigint;
  challengedStateHash?: string;
}

/**
 * Signed off-chain state proof
 */
export interface SignedChannelState {
  channelId: string;
  nonce: bigint;
  spent: bigint;
  available: bigint;
  stateHash: string;    // Hash of (channelId, nonce, spent, distributions)

  // Accumulated distributions since last settlement
  distributions: SerializedPaymentDistribution[];

  // Signatures
  payerSignature: string;
  providerSignature: string;
}

/**
 * Serialized payment distribution (bigint-safe)
 */
export interface SerializedPaymentDistribution {
  apiProvider: { did: string; amount: string };
  knowledgeRewards: Array<{ did: string; factId: string; amount: string }>;
  protocol: { burned: string; treasury: string };
  research?: { amount: string; treasury: string };
}

// Transaction payloads
export interface OpenPaymentChannelPayload {
  type: 'open_payment_channel';
  provider: string;      // Provider DID
  deposit: bigint;
  durationBlocks: bigint;
  settlementFrequency?: bigint;
}

export interface UpdatePaymentChannelPayload {
  type: 'update_payment_channel';
  channelId: string;
  signedState: SignedChannelState;
}

export interface ClosePaymentChannelPayload {
  type: 'close_payment_channel';
  channelId: string;
  finalState?: SignedChannelState; // Cooperative close
  unilateral?: boolean;            // Unilateral close (starts dispute period)
}

export interface SettlePaymentChannelPayload {
  type: 'settle_payment_channel';
  channelId: string;
  signedState: SignedChannelState;
}

export interface DisputePaymentChannelPayload {
  type: 'dispute_payment_channel';
  channelId: string;
  challengeState: SignedChannelState; // State with higher nonce
}
```

### Transaction Types (`consensus/pot/types.ts`)

Add to `TransactionType` union:
```typescript
// Payment channel transactions
| 'open_payment_channel'
| 'update_payment_channel'
| 'close_payment_channel'
| 'settle_payment_channel'
| 'dispute_payment_channel'
```

Add to `TransactionPayload` union:
```typescript
| { type: 'open_payment_channel'; provider: string; deposit: bigint; durationBlocks: bigint; settlementFrequency?: bigint }
| { type: 'update_payment_channel'; channelId: string; signedState: SignedChannelState }
| { type: 'close_payment_channel'; channelId: string; finalState?: SignedChannelState; unilateral?: boolean }
| { type: 'settle_payment_channel'; channelId: string; signedState: SignedChannelState }
| { type: 'dispute_payment_channel'; channelId: string; challengeState: SignedChannelState }
```

---

## Part V: State Layer

### PaymentChannelStateTree (`state/payment_channels.ts`)

```typescript
/**
 * Payment Channel State Tree
 *
 * Manages payment channel state with SMT storage.
 * Pattern follows ClaimStateTree (state/claims.ts).
 */
export class PaymentChannelStateTree {
  private channels: Map<string, PaymentChannelState> = new Map();

  // Indexes for efficient queries
  private byPayer: Map<string, Set<string>> = new Map();
  private byProvider: Map<string, Set<string>> = new Map();
  private byStatus: Map<PaymentChannelStatus, Set<string>> = new Map();
  private byExpiration: Map<bigint, Set<string>> = new Map(); // For expiry processing

  // Channel operations
  addChannel(channel: PaymentChannelState): void;
  getChannel(channelId: string): PaymentChannelState | undefined;
  updateChannel(channelId: string, updates: Partial<PaymentChannelState>): PaymentChannelState;

  // Queries
  getChannelsByPayer(payer: string): PaymentChannelState[];
  getChannelsByProvider(provider: string): PaymentChannelState[];
  getChannelsByStatus(status: PaymentChannelStatus): PaymentChannelState[];
  getExpiringChannels(beforeBlock: bigint): PaymentChannelState[];
  getChannelsNeedingSettlement(currentBlock: bigint): PaymentChannelState[];

  // State proofs
  verifyStateProof(state: SignedChannelState): boolean;

  // Statistics
  getChannelStats(): {
    totalChannels: number;
    openChannels: number;
    totalDeposited: bigint;
    totalSpent: bigint;
    pendingSettlements: number;
  };
}

/**
 * Generate deterministic channel ID
 */
export function generateChannelId(
  payer: string,
  provider: string,
  blockNumber: bigint
): string {
  const hash = domainHash(
    'LGM.payment_channel.id.v1',
    `${payer}:${provider}:${blockNumber}`
  );
  return `pc:${hash.slice(0, 32)}`;
}

/**
 * Compute state hash for signing
 */
export function computeStateHash(
  channelId: string,
  nonce: bigint,
  spent: bigint,
  distributions: SerializedPaymentDistribution[]
): string {
  return domainHash(
    'LGM.payment_channel.state.v1',
    JSON.stringify({ channelId, nonce: nonce.toString(), spent: spent.toString(), distributions })
  );
}

/**
 * Verify signed state proof
 */
export function verifySignedState(
  state: SignedChannelState,
  payerPublicKey: string,
  providerPublicKey: string
): boolean {
  const expectedHash = computeStateHash(
    state.channelId,
    state.nonce,
    state.spent,
    state.distributions
  );

  if (state.stateHash !== expectedHash) return false;

  // Verify both signatures
  const payerValid = verifySignature(payerPublicKey, state.stateHash, state.payerSignature);
  const providerValid = verifySignature(providerPublicKey, state.stateHash, state.providerSignature);

  return payerValid && providerValid;
}
```

---

## Part VI: Transaction Handlers

### Handler Implementations (`state/transitions.ts`)

```typescript
/**
 * Open a payment channel with deposit
 */
function executeOpenPaymentChannel(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as OpenPaymentChannelPayload;
  const payerDid = publicKeyToDid(tx.sender);

  // 1. Validate deposit meets minimum (from BillingParameters)
  const minDeposit = 100n * 10n ** 18n; // 100 LGM minimum
  if (payload.deposit < minDeposit) {
    throw new StateError(`Minimum deposit is ${minDeposit}`, 'INVALID_DEPOSIT');
  }

  // 2. Validate payer has sufficient balance
  const account = state.getAccount(payerDid);
  if (!account || account.balance < payload.deposit) {
    throw new StateError('Insufficient balance', 'INSUFFICIENT_BALANCE');
  }

  // 3. Validate provider exists and is active
  const providerAccount = state.getAccount(payload.provider);
  if (!providerAccount) {
    throw new StateError('Provider not found', 'PROVIDER_NOT_FOUND');
  }

  // 4. Generate channel ID
  const channelId = generateChannelId(payerDid, payload.provider, blockNumber);

  // 5. Lock deposit (move to escrow)
  state.transfer(payerDid, ESCROW_DID, payload.deposit);

  // 6. Create channel state
  const channel: PaymentChannelState = {
    channelId,
    payer: payerDid,
    provider: payload.provider,
    deposited: payload.deposit,
    spent: 0n,
    available: payload.deposit,
    status: 'open',
    openedAt: blockNumber,
    expiresAt: blockNumber + payload.durationBlocks,
    lastNonce: 0n,
    lastStateHash: '',
    pendingDistributions: [],
    totalDistributed: 0n,
    lastSettlementBlock: blockNumber,
    settlementFrequency: payload.settlementFrequency ?? 100n,
  };

  state.addPaymentChannel(channel);

  return 150000n; // Gas cost
}

/**
 * Update channel state with signed proof (for state sync)
 * This is primarily for recording agreed states on-chain
 */
function executeUpdatePaymentChannel(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint
): bigint {
  const payload = tx.payload as UpdatePaymentChannelPayload;
  const senderDid = publicKeyToDid(tx.sender);

  // 1. Get channel
  const channel = state.getPaymentChannel(payload.channelId);
  if (!channel) {
    throw new StateError('Channel not found', 'CHANNEL_NOT_FOUND');
  }

  // 2. Validate sender is payer or provider
  if (senderDid !== channel.payer && senderDid !== channel.provider) {
    throw new StateError('Not channel participant', 'UNAUTHORIZED');
  }

  // 3. Validate channel is open
  if (channel.status !== 'open') {
    throw new StateError(`Channel is ${channel.status}`, 'CHANNEL_NOT_OPEN');
  }

  // 4. Verify signed state
  const payerKey = state.getAccount(channel.payer)?.operationalKeyHash;
  const providerKey = state.getAccount(channel.provider)?.operationalKeyHash;
  if (!verifySignedState(payload.signedState, payerKey!, providerKey!)) {
    throw new StateError('Invalid state signatures', 'INVALID_SIGNATURES');
  }

  // 5. Validate nonce is higher
  if (payload.signedState.nonce <= channel.lastNonce) {
    throw new StateError('Nonce too low', 'INVALID_NONCE');
  }

  // 6. Validate invariant: spent + available = deposited
  if (payload.signedState.spent + payload.signedState.available !== channel.deposited) {
    throw new StateError('Balance invariant violated', 'INVALID_BALANCE');
  }

  // 7. Update channel state
  state.updatePaymentChannel(payload.channelId, {
    spent: payload.signedState.spent,
    available: payload.signedState.available,
    lastNonce: payload.signedState.nonce,
    lastStateHash: payload.signedState.stateHash,
    pendingDistributions: payload.signedState.distributions,
  });

  return 50000n;
}

/**
 * Settle pending distributions on-chain
 */
function executeSettlePaymentChannel(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as SettlePaymentChannelPayload;
  const senderDid = publicKeyToDid(tx.sender);

  // 1. Get and validate channel
  const channel = state.getPaymentChannel(payload.channelId);
  if (!channel) {
    throw new StateError('Channel not found', 'CHANNEL_NOT_FOUND');
  }

  // 2. Only provider can settle (they receive the payments)
  if (senderDid !== channel.provider) {
    throw new StateError('Only provider can settle', 'UNAUTHORIZED');
  }

  // 3. Validate channel is open or closing
  if (channel.status !== 'open' && channel.status !== 'closing') {
    throw new StateError(`Cannot settle ${channel.status} channel`, 'INVALID_STATUS');
  }

  // 4. Verify signed state
  // ... (same verification as update)

  // 5. Validate settlement frequency
  if (blockNumber - channel.lastSettlementBlock < channel.settlementFrequency) {
    throw new StateError('Settlement too soon', 'SETTLEMENT_TOO_SOON');
  }

  // 6. Process distributions
  const distributions = payload.signedState.distributions;
  for (const dist of distributions) {
    // Transfer to API provider
    state.transfer(ESCROW_DID, dist.apiProvider.did, BigInt(dist.apiProvider.amount));

    // Transfer knowledge rewards
    for (const reward of dist.knowledgeRewards) {
      state.transfer(ESCROW_DID, reward.did, BigInt(reward.amount));
    }

    // Protocol treasury and burn
    state.transfer(ESCROW_DID, TREASURY_DID, BigInt(dist.protocol.treasury));
    state.burn(ESCROW_DID, BigInt(dist.protocol.burned));

    // Research share
    if (dist.research) {
      state.transfer(ESCROW_DID, dist.research.treasury, BigInt(dist.research.amount));
    }
  }

  // 7. Update channel
  state.updatePaymentChannel(payload.channelId, {
    totalDistributed: channel.totalDistributed + sumDistributions(distributions),
    pendingDistributions: [],
    lastSettlementBlock: blockNumber,
    lastNonce: payload.signedState.nonce,
    lastStateHash: payload.signedState.stateHash,
  });

  return 200000n + BigInt(distributions.length) * 50000n;
}

/**
 * Close payment channel
 */
function executeClosePaymentChannel(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint,
  params: ConsensusParams
): bigint {
  const payload = tx.payload as ClosePaymentChannelPayload;
  const senderDid = publicKeyToDid(tx.sender);

  const channel = state.getPaymentChannel(payload.channelId);
  if (!channel) {
    throw new StateError('Channel not found', 'CHANNEL_NOT_FOUND');
  }

  // Validate sender is participant
  if (senderDid !== channel.payer && senderDid !== channel.provider) {
    throw new StateError('Not channel participant', 'UNAUTHORIZED');
  }

  if (payload.unilateral) {
    // UNILATERAL CLOSE: Start dispute period
    // Other party has DISPUTE_PERIOD blocks to challenge with higher nonce state
    const DISPUTE_PERIOD = 1000n; // ~42 minutes at 2.5s blocks

    state.updatePaymentChannel(payload.channelId, {
      status: 'closing',
      disputeStartBlock: blockNumber,
      disputeDeadline: blockNumber + DISPUTE_PERIOD,
    });

    return 100000n;

  } else if (payload.finalState) {
    // COOPERATIVE CLOSE: Both parties signed, close immediately
    // Verify final state has both signatures
    // ... (same verification)

    // Process final settlement
    // ... (same as settle)

    // Return remaining funds to payer
    state.transfer(ESCROW_DID, channel.payer, payload.finalState.available);

    // Close channel
    state.updatePaymentChannel(payload.channelId, {
      status: 'closed',
      spent: payload.finalState.spent,
      available: 0n,
    });

    return 200000n;

  } else {
    // EXPIRY CLOSE: Channel expired, close with last known state
    if (blockNumber < channel.expiresAt) {
      throw new StateError('Channel not expired', 'NOT_EXPIRED');
    }

    // Settle any pending distributions
    // Return remaining to payer
    state.transfer(ESCROW_DID, channel.payer, channel.available);

    state.updatePaymentChannel(payload.channelId, {
      status: 'closed',
      available: 0n,
    });

    return 150000n;
  }
}

/**
 * Dispute with higher nonce state
 */
function executeDisputePaymentChannel(
  state: StateManager,
  tx: Transaction,
  blockNumber: bigint
): bigint {
  const payload = tx.payload as DisputePaymentChannelPayload;
  const senderDid = publicKeyToDid(tx.sender);

  const channel = state.getPaymentChannel(payload.channelId);
  if (!channel) {
    throw new StateError('Channel not found', 'CHANNEL_NOT_FOUND');
  }

  // Only during dispute period
  if (channel.status !== 'closing' || !channel.disputeDeadline) {
    throw new StateError('Channel not in dispute', 'NOT_IN_DISPUTE');
  }

  if (blockNumber > channel.disputeDeadline) {
    throw new StateError('Dispute period ended', 'DISPUTE_EXPIRED');
  }

  // Verify challenger state has higher nonce
  if (payload.challengeState.nonce <= channel.lastNonce) {
    throw new StateError('Challenge nonce not higher', 'INVALID_CHALLENGE');
  }

  // Verify signatures on challenge state
  // ... (same verification)

  // Update with challenged state
  state.updatePaymentChannel(payload.channelId, {
    lastNonce: payload.challengeState.nonce,
    lastStateHash: payload.challengeState.stateHash,
    spent: payload.challengeState.spent,
    available: payload.challengeState.available,
    pendingDistributions: payload.challengeState.distributions,
    challengedStateHash: payload.challengeState.stateHash,
  });

  // Slash original closer for attempted fraud (5% of their balance)
  // ... (slashing logic)

  return 100000n;
}
```

---

## Part VII: Constants

```typescript
// state/payment_channels.ts or core/billing/constants.ts

export const PAYMENT_CHANNEL_CONSTANTS = {
  // Minimum deposit (100 LGM)
  MIN_DEPOSIT: 100n * 10n ** 18n,

  // Maximum channel duration (30 days at 2.5s blocks = ~1,036,800 blocks)
  MAX_DURATION_BLOCKS: 1036800n,

  // Default settlement frequency (100 blocks = ~4 minutes)
  DEFAULT_SETTLEMENT_FREQUENCY: 100n,

  // Dispute period for unilateral close (1000 blocks = ~42 minutes)
  DISPUTE_PERIOD_BLOCKS: 1000n,

  // Grace period after expiry for claims (500 blocks = ~21 minutes)
  EXPIRY_GRACE_PERIOD: 500n,

  // Fraud penalty (5% of remaining balance)
  FRAUD_PENALTY_RATE: 0.05,

  // Escrow account for channel funds
  ESCROW_DID: 'did:lgm:payment-channel-escrow',
};
```

---

## Part VIII: StateManager Integration

### New Methods

```typescript
// state/state_manager.ts

// Payment channel methods
addPaymentChannel(channel: PaymentChannelState): void;
getPaymentChannel(channelId: string): PaymentChannelState | undefined;
updatePaymentChannel(channelId: string, updates: Partial<PaymentChannelState>): PaymentChannelState;
getPaymentChannelsByPayer(payer: string): PaymentChannelState[];
getPaymentChannelsByProvider(provider: string): PaymentChannelState[];
getOpenPaymentChannels(): PaymentChannelState[];
getExpiringPaymentChannels(beforeBlock: bigint): PaymentChannelState[];
getPaymentChannelsNeedingSettlement(currentBlock: bigint): PaymentChannelState[];
getPaymentChannelStats(): PaymentChannelStats;
```

---

## Part IX: Test Plan

### Unit Tests (`tests/unit/state/payment_channel_transitions.test.ts`)

```typescript
describe('Payment Channel Workflow', () => {
  describe('executeOpenPaymentChannel', () => {
    it('should create channel with correct initial state');
    it('should lock deposit in escrow');
    it('should reject insufficient balance');
    it('should reject below minimum deposit');
    it('should generate deterministic channel ID');
    it('should set correct expiration');
  });

  describe('executeUpdatePaymentChannel', () => {
    it('should update state with valid signed proof');
    it('should reject invalid signatures');
    it('should reject lower nonce');
    it('should enforce balance invariant');
    it('should reject non-participants');
  });

  describe('executeSettlePaymentChannel', () => {
    it('should distribute to all recipients');
    it('should respect 22% research share');
    it('should clear pending distributions');
    it('should enforce settlement frequency');
    it('should only allow provider to settle');
  });

  describe('executeClosePaymentChannel', () => {
    describe('cooperative close', () => {
      it('should close immediately with final state');
      it('should return remaining funds to payer');
      it('should settle pending distributions');
    });

    describe('unilateral close', () => {
      it('should start dispute period');
      it('should allow challenge with higher nonce');
      it('should close after dispute period');
    });

    describe('expiry close', () => {
      it('should close after expiration');
      it('should return remaining to payer');
    });
  });

  describe('executeDisputePaymentChannel', () => {
    it('should accept state with higher nonce');
    it('should reject lower nonce');
    it('should reject after dispute period');
    it('should slash fraudulent closer');
  });

  describe('PaymentChannelStateTree', () => {
    it('should index by payer');
    it('should index by provider');
    it('should index by status');
    it('should track expiring channels');
  });

  describe('Complete Flow', () => {
    it('should flow from open to settlement to close');
    it('should handle multiple settlements');
    it('should handle dispute scenario');
  });
});
```

### Expected Test Count: ~30 tests

---

## Part X: Implementation Phases

### Phase 1: Types (1 day)
- [ ] Add `PaymentChannelState` and related types to `state/types.ts`
- [ ] Add transaction types to `consensus/pot/types.ts`
- [ ] Add payload types for all channel operations

### Phase 2: State Layer (2 days)
- [ ] Create `state/payment_channels.ts` with `PaymentChannelStateTree`
- [ ] Implement indexes (payer, provider, status, expiration)
- [ ] Add `generateChannelId()`, `computeStateHash()`, `verifySignedState()`
- [ ] Integrate with `StateManager`

### Phase 3: Transaction Handlers (2 days)
- [ ] Implement `executeOpenPaymentChannel`
- [ ] Implement `executeUpdatePaymentChannel`
- [ ] Implement `executeSettlePaymentChannel`
- [ ] Implement `executeClosePaymentChannel`
- [ ] Implement `executeDisputePaymentChannel`

### Phase 4: Testing (2 days)
- [ ] Create comprehensive test suite
- [ ] Test all edge cases and error paths
- [ ] Integration tests with billing calculation

### Phase 5: SDK/RPC Integration (1 day)
- [ ] Implement RPC handlers (`lgm_getPaymentChannel`, etc.)
- [ ] Connect existing SDK methods to backends
- [ ] Add off-chain state signing helpers

**Total Effort: 8 days**

---

## Part XI: Dependencies

### Depends On
- [x] GAP-002 (Claim Submission) - Pattern for state trees ✅
- [x] Billing calculation (`core/billing/distribution.ts`) ✅
- [x] Existing `PaymentChannel` type ✅

### Blocks
- [ ] GAP-007 (Per-Message Billing) - Will use payment channels for streaming

---

## Part XII: Security Considerations

### Double-Spend Prevention
- Nonce must be monotonically increasing
- Both signatures required for valid state
- Latest nonce always wins in disputes

### Fraud Prevention
- Dispute period allows challenge of outdated states
- Fraud penalty (5%) discourages cheating
- State hash ensures integrity

### Denial of Service
- Minimum deposit prevents spam channels
- Settlement frequency limits on-chain load
- Expiration prevents infinite channels

### Key Compromise
- Session keys can be used for channel operations (limited scope)
- Recovery mechanism if operational key compromised

---

## Appendix: Off-Chain State Signing (SDK Helper)

```typescript
// sdk/typescript/payment_channel.ts

export class PaymentChannelClient {
  private channel: PaymentChannel;
  private currentNonce: bigint = 0n;
  private pendingDistributions: SerializedPaymentDistribution[] = [];

  /**
   * Process an API call and update local state
   */
  async processAPICall(
    cost: bigint,
    distribution: PaymentDistribution
  ): Promise<SignedChannelState> {
    // Update local state
    this.currentNonce += 1n;
    const newSpent = this.channel.spent + cost;
    const newAvailable = this.channel.available - cost;

    this.pendingDistributions.push(serializeDistribution(distribution));

    // Compute state hash
    const stateHash = computeStateHash(
      this.channel.channelId,
      this.currentNonce,
      newSpent,
      this.pendingDistributions
    );

    // Sign state
    const payerSignature = await this.wallet.sign(stateHash);

    // Create partial state (awaits provider countersign)
    return {
      channelId: this.channel.channelId,
      nonce: this.currentNonce,
      spent: newSpent,
      available: newAvailable,
      stateHash,
      distributions: this.pendingDistributions,
      payerSignature,
      providerSignature: '', // Provider will countersign
    };
  }

  /**
   * Verify and accept provider's countersigned state
   */
  acceptCountersignedState(state: SignedChannelState): void {
    // Verify provider signature
    if (!verifySignature(this.channel.provider, state.stateHash, state.providerSignature)) {
      throw new Error('Invalid provider signature');
    }

    // Update local channel state
    this.channel.spent = state.spent;
    this.channel.available = state.available;
    this.latestConfirmedState = state;
  }
}
```

---

## PP Completion Checklist

### Truth Gates
- [x] All claims cite file:line references
- [x] Pattern is grounded in established state channel design
- [x] Existing code correctly understood

### Understanding Gates
- [x] Complete mental model documented
- [x] Why payment channels needed (transaction overhead reduction)
- [x] Boundaries clear (API billing only, not general L2)

### Beauty Gates
- [x] Follows existing patterns (`ClaimStateTree`)
- [x] Reuses existing types where possible
- [x] Minimal new complexity

### Justice Gates
- [x] Correct placement (`state/payment_channels.ts`)
- [x] Right ownership (state layer owns channels)
- [x] Architectural integrity preserved

### Creativity Gates
- [x] Off-chain state protocol is genuinely new
- [x] Path well-documented for implementers
- [x] Integrates with existing billing system

---

**Design Status: COMPLETE**
**Ready for Implementation: YES**
