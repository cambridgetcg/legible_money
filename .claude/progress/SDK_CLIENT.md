# SDK Client - E2E Tests Progress

**Status**: 🔴 Not Started
**Priority**: Critical
**Estimated Work**: 2-3 days
**Blocking**: Developer adoption, third-party integrations

---

## Overview

The SDK client (`protocol/sdk/client.ts`, 755 lines) provides the primary interface for developers to interact with Legible Money. It includes wallet management, transaction building, and RPC communication. While the implementation is complete, E2E tests are missing.

---

## Current State

### Implemented
- [x] `LegibleClient` - Main client class
- [x] `AgentWallet` - Key management and signing
- [x] Transaction builders for all tx types
- [x] RPC method wrappers
- [x] WebSocket subscription support
- [x] Automatic nonce management
- [x] Gas estimation helpers
- [x] Error handling and retries

### Missing
- [ ] Client connection tests
- [ ] Wallet creation/import tests
- [ ] Transaction submission tests
- [ ] Query method tests
- [ ] Subscription tests
- [ ] Error recovery tests
- [ ] Payment channel SDK methods
- [ ] Example usage files

---

## Test Plan

### Phase 1: Client Connection (Day 1)
```
tests/integration/sdk/connection.test.ts
```

| Test | Description | Status |
|------|-------------|--------|
| `client connects to RPC endpoint` | HTTP connection works | [ ] |
| `client connects via WebSocket` | WS upgrade works | [ ] |
| `client handles connection failure` | Graceful error | [ ] |
| `client reconnects on disconnect` | Auto-reconnect works | [ ] |
| `client respects timeout config` | Timeout triggers | [ ] |
| `client validates endpoint URL` | Bad URL rejected | [ ] |

### Phase 2: Wallet Operations (Day 1)
```
tests/integration/sdk/wallet.test.ts
```

| Test | Description | Status |
|------|-------------|--------|
| `wallet generates new keypair` | Random key works | [ ] |
| `wallet imports from private key` | Import works | [ ] |
| `wallet imports from mnemonic` | BIP39 works | [ ] |
| `wallet signs message correctly` | Signature valid | [ ] |
| `wallet signs transaction` | Tx signature valid | [ ] |
| `wallet derives DID correctly` | DID format correct | [ ] |
| `wallet exports private key` | Export matches import | [ ] |

### Phase 3: Transaction Submission (Day 1-2)
```
tests/integration/sdk/transactions.test.ts
```

| Category | Transactions | Tests | Status |
|----------|--------------|-------|--------|
| **Transfer** | `transfer` | 4 | [ ] |
| **Staking** | `stake`, `unstake`, `delegate` | 6 | [ ] |
| **Knowledge** | `submitClaim`, `commitVerification`, `revealVerification` | 8 | [ ] |
| **Governance** | `propose`, `vote` | 4 | [ ] |
| **Challenge** | `challenge`, `submitEvidence` | 4 | [ ] |

Individual tests per category:
- Submit and confirm
- Submit with insufficient balance (expect failure)
- Submit with invalid nonce (expect failure)
- Submit with gas estimation

### Phase 4: Query Methods (Day 2)
```
tests/integration/sdk/queries.test.ts
```

| Test | Description | Status |
|------|-------------|--------|
| `getBalance returns correct balance` | Balance query | [ ] |
| `getAccount returns full account` | Account details | [ ] |
| `getFact returns fact with confidence` | Fact query | [ ] |
| `queryFacts with filters works` | Filtered query | [ ] |
| `getDomain returns domain tree` | Domain query | [ ] |
| `getBlock returns block data` | Block query | [ ] |
| `getTransaction returns tx details` | Tx query | [ ] |
| `getValidatorSet returns validators` | Validator query | [ ] |

### Phase 5: Subscriptions (Day 2-3)
```
tests/integration/sdk/subscriptions.test.ts
```

| Test | Description | Status |
|------|-------------|--------|
| `subscribe to new blocks` | Block events received | [ ] |
| `subscribe to pending transactions` | Tx events received | [ ] |
| `subscribe to account updates` | Account events received | [ ] |
| `subscribe to fact verifications` | Fact events received | [ ] |
| `unsubscribe stops events` | Cleanup works | [ ] |
| `multiple subscriptions work` | Concurrent subs | [ ] |
| `subscription survives reconnect` | Re-subscribe works | [ ] |

### Phase 6: Payment Channels (Day 3)
```
tests/integration/sdk/payment_channels.test.ts
```

| Test | Description | Status |
|------|-------------|--------|
| `openChannel creates channel` | Channel creation | [ ] |
| `updateChannel with signatures` | State update | [ ] |
| `closeChannel cooperative` | Mutual close | [ ] |
| `settleChannel after timeout` | Unilateral close | [ ] |
| `disputeChannel with proof` | Dispute flow | [ ] |

**Note**: Payment channel SDK methods need to be implemented first.

### Phase 7: Error Handling (Day 3)
```
tests/integration/sdk/errors.test.ts
```

| Test | Description | Status |
|------|-------------|--------|
| `handles RPC error gracefully` | Error propagation | [ ] |
| `retries on transient failure` | Retry logic | [ ] |
| `respects max retries` | Gives up eventually | [ ] |
| `handles timeout correctly` | Timeout error | [ ] |
| `handles invalid response` | Parse error | [ ] |

---

## Files to Create

```
tests/integration/sdk/
├── connection.test.ts      # Connection tests
├── wallet.test.ts          # Wallet operations
├── transactions.test.ts    # Tx submission tests
├── queries.test.ts         # Query method tests
├── subscriptions.test.ts   # Subscription tests
├── payment_channels.test.ts # Payment channel tests
├── errors.test.ts          # Error handling tests
└── helpers/
    ├── test_node.ts        # Spin up test node
    └── fixtures.ts         # Test data
```

---

## SDK Methods to Implement

The following SDK methods are missing and should be added:

```typescript
// protocol/sdk/client.ts - Payment Channel Methods

async openPaymentChannel(
  counterparty: DID,
  deposit: bigint,
  duration: bigint
): Promise<ChannelId>

async updatePaymentChannel(
  channelId: ChannelId,
  newBalance: bigint,
  signature: Signature
): Promise<void>

async closePaymentChannel(
  channelId: ChannelId,
  finalState: ChannelState,
  signatures: [Signature, Signature]
): Promise<void>

async settlePaymentChannel(
  channelId: ChannelId,
  proof: ChannelProof
): Promise<void>

async disputePaymentChannel(
  channelId: ChannelId,
  evidence: ChannelState
): Promise<void>

async getPaymentChannel(
  channelId: ChannelId
): Promise<PaymentChannel | null>
```

---

## Example Files to Create

```
examples/
├── connect_to_node.ts      # Basic connection example
├── create_wallet.ts        # Wallet creation
├── send_transfer.ts        # Simple transfer
├── submit_claim.ts         # Knowledge claim submission
├── verify_claim.ts         # Participate in verification
├── subscribe_blocks.ts     # Block subscription
├── payment_channel.ts      # Payment channel flow
└── README.md               # Examples documentation
```

---

## Dependencies

- Running RPC server for tests
- Test accounts with funded balances
- Genesis state with test facts/domains
- `vitest` for test runner

---

## Acceptance Criteria

1. All public SDK methods have E2E tests
2. Wallet can sign and submit all transaction types
3. Subscriptions deliver events within 100ms
4. Error messages are helpful for debugging
5. Examples run successfully against devnet
6. Payment channel lifecycle works end-to-end
7. SDK TypeScript types are accurate

---

## Progress Log

| Date | Update |
|------|--------|
| 2026-02-05 | Progress tracking created |
| | |

---

## Notes

- SDK already has good TypeScript types
- Consider adding JSDoc comments for better IDE support
- Wallet class in `protocol/sdk/wallet.ts` is well-implemented
- May need mock RPC server for unit tests (vs integration tests)
- Payment channels blocked on `state/payment_channels.ts` completion
