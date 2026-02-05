# Billing Module

Usage-based billing with automatic knowledge distribution.

## Overview

The billing module handles:

- **API call billing** - Charge for knowledge queries
- **Payment distribution** - Route payments to contributors
- **Message billing** - Per-message chat billing (GAP-007)
- **Gas accounting** - Unified gas costs (GAP-5)
- **Research fund** - 22% to Protocol Research Repository

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      API Request                         │
└────────────────────────┬────────────────────────────────┘
                         │
         ┌───────────────▼───────────────┐
         │        TruthGateway           │
         │   (Billing + Truth Access)    │
         └───────────────┬───────────────┘
                         │
    ┌────────────────────┼────────────────────┐
    │                    │                    │
    ▼                    ▼                    ▼
┌─────────┐      ┌─────────────┐      ┌─────────────┐
│ Message │      │   Token     │      │    Gas      │
│ Billing │      │  Counter    │      │  Calculator │
└────┬────┘      └──────┬──────┘      └──────┬──────┘
     │                  │                    │
     └──────────────────┼────────────────────┘
                        │
         ┌──────────────▼──────────────┐
         │       RewardRouter          │
         │   (Distribution Engine)     │
         └──────────────┬──────────────┘
                        │
    ┌───────────────────┼───────────────────┐
    │                   │                   │
    ▼                   ▼                   ▼
┌─────────┐      ┌───────────┐      ┌───────────┐
│Knowledge│      │ Research  │      │  Protocol │
│ Authors │      │   Fund    │      │  Treasury │
└─────────┘      └───────────┘      └───────────┘
```

## Components

### MessageBilling (GAP-007)

Per-message billing for agent chat sessions.

```typescript
import { MessageBillingService, DEFAULT_MESSAGE_BILLING_CONFIG } from './billing';

const billing = new MessageBillingService({
  baseCostPerMessage: 100n,      // 100 wei base
  costPerToken: 10n,             // 10 wei per token
  freeTierMessages: 10,          // Free messages per session
  maxMessagesPerSession: 1000,
});

// Calculate message cost
const cost = billing.calculateMessageCost({
  inputTokens: 500,
  outputTokens: 200,
  factsUsed: 3,
  complexity: 'medium',
});

// Track session usage
const session = billing.startSession(agentDid);
await billing.chargeMessage(session.id, cost);
```

### TokenCounter

Count tokens in messages.

```typescript
import { TokenCounter } from './billing';

const counter = new TokenCounter();

// Count tokens
const tokens = counter.countTokens('Hello, world!');
// Returns: { total: 4, words: 2, punctuation: 2 }

// Estimate from content
const estimate = counter.estimateTokens(message);
```

### MessageTracker

Track message history and costs.

```typescript
import { MessageTracker } from './billing';

const tracker = new MessageTracker();

// Start session
const session = tracker.startSession(agentDid);

// Track message
tracker.recordMessage(session.id, {
  role: 'user',
  content: 'What is the speed of light?',
  tokens: 8,
  cost: 80n,
});

// Get session stats
const stats = tracker.getSessionStats(session.id);
console.log('Total cost:', stats.totalCost);
console.log('Messages:', stats.messageCount);
```

### Distribution

Payment distribution to knowledge contributors.

```typescript
import { PaymentDistributor, calculateDistribution } from './billing';

// Calculate distribution for API call
const distribution = calculateDistribution({
  totalCost: 1000n,
  factsUsed: [
    { factId: 'fact:1', confidence: 0.9, author: 'did:lgm:a' },
    { factId: 'fact:2', confidence: 0.8, author: 'did:lgm:b' },
  ],
  apiProvider: 'did:lgm:provider',
});

// Distribution result:
// - 22% to research fund
// - 20% to API provider
// - 58% to fact authors (weighted by confidence)

// Execute distribution
const distributor = new PaymentDistributor(stateManager);
await distributor.distribute(distribution);
```

### RewardRouter

Route rewards to recipients.

```typescript
import { RewardRouter } from './billing';

const router = new RewardRouter(stateManager);

// Route verification reward
await router.routeVerificationReward({
  validator: validatorPubKey,
  amount: 1000000000000000000n,  // 1 LGM
  roundId: 'round:123',
  verdict: 'confirmed',
});

// Route claim reward
await router.routeClaimReward({
  submitter: 'did:lgm:submitter',
  amount: 5000000000000000000n,  // 5 LGM
  claimId: 'claim:456',
});
```

### Gas (GAP-5)

Unified gas cost calculations.

```typescript
import { GasCalculator, GAS_COSTS, estimateGas } from './billing';

// Estimate transaction gas
const gas = estimateGas({
  type: 'transfer',
  dataSize: 100,
});

// Gas costs
console.log('Transfer:', GAS_COSTS.transfer);
console.log('Submit claim:', GAS_COSTS.submitClaim);
console.log('Verification:', GAS_COSTS.verification);

// Calculate fee
const fee = GasCalculator.calculateFee(gas, gasPrice);
```

### TruthGateway

Unified interface for billed truth access.

```typescript
import { TruthGateway } from './billing';

const gateway = new TruthGateway(stateManager, billingService);

// Query with billing
const result = await gateway.query({
  agentDid: 'did:lgm:agent',
  intent: { type: 'explore', domain: 'science' },
  maxCost: 1000n,
});

// Result includes facts and billing info
console.log('Facts:', result.facts);
console.log('Cost:', result.cost);
console.log('Cached:', result.cached);
```

## Configuration

### Message Billing Config

```typescript
interface MessageBillingConfig {
  baseCostPerMessage: bigint;  // Base cost per message
  costPerToken: bigint;        // Additional per token
  costPerFactUsed: bigint;     // Cost for each fact cited
  freeTierMessages: number;    // Free messages per session
  maxMessagesPerSession: number;
}
```

### Distribution Config

```typescript
interface DistributionConfig {
  researchFundShare: number;   // 0.22 (22%)
  apiProviderShare: number;    // 0.20 (20%)
  knowledgeShare: number;      // 0.58 (58%)
  protocolBurn: number;        // 0.00 (no burn)
}
```

### Gas Costs

| Operation | Gas Units |
|-----------|----------|
| Transfer | 21,000 |
| Submit Claim | 50,000 |
| Verification Commit | 30,000 |
| Verification Reveal | 35,000 |
| Knowledge Query | 10,000 |
| Session Start | 5,000 |

## API Provider Registration

```typescript
interface APIProvider {
  did: string;
  stake: bigint;
  endpoints: APIEndpoint[];
  basePricePerCall: bigint;
  pricingModel: 'per_call' | 'per_token' | 'per_fact';
  maxLatency: number;
  minUptime: number;
  status: 'active' | 'suspended' | 'deregistered';
}
```

## Distribution Formula

For each API call:

```
Total Cost = Base Fee + Token Fee + Fact Fee

Distribution:
  Research Fund:    22% of Total
  API Provider:     20% of Total
  Fact Authors:     58% of Total (weighted by confidence × usage)

Per-Author Share = (confidence × usageWeight) / Σ(confidence × usageWeight) × 58%
```

## Error Handling

```typescript
import { BillingError, InsufficientBalanceError } from './billing';

try {
  await gateway.query(params);
} catch (error) {
  if (error instanceof InsufficientBalanceError) {
    console.log('Balance needed:', error.required);
    console.log('Balance available:', error.available);
  }
}
```

## Testing

```bash
# Run billing tests
npm run test -- tests/unit/billing/

# Test message billing
npm run test -- tests/unit/billing/message_billing.test.ts
```

## Files

| File | Description |
|------|-------------|
| `types.ts` | Type definitions |
| `message_billing.ts` | Per-message billing (GAP-007) |
| `message_tracker.ts` | Session tracking |
| `token_counter.ts` | Token counting |
| `distribution.ts` | Payment distribution |
| `reward_router.ts` | Reward routing |
| `gas.ts` | Gas calculations (GAP-5) |
| `truth_gateway.ts` | Billed truth access |

## See Also

- [Knowledge Economics](../../docs/KNOWLEDGE_ECONOMICS.md)
- [API Billing](../../docs/API_BILLING.md)
- [GLOSSARY.md](../../docs/GLOSSARY.md)

---

*Billing module implementation (GAP-5, GAP-007)*
