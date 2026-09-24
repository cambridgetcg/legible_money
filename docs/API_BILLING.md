# API Billing: Usage-Based Knowledge Distribution

These design notes describe a billing and distribution model. Numerical scenarios
are illustrative assumptions, not measured income, forecasts or earning targets.
Any distribution depends on actual usage, eligibility and the running protocol.

> **Core Insight**: When an API call uses knowledge, the payment should flow to everyone who contributed that knowledge. The API is just the interface—the value is the verified truth underneath.

---

## The Model

```
TRADITIONAL API BILLING:
  User → Pays API Provider → Done

LEGIBLE MONEY API BILLING:
  User → Pays in LGM → Distributed to:
    ├── API Provider (service fee)
    ├── Knowledge used (fact rewards)
    ├── Ancestors of that knowledge (citation chain)
    └── Protocol (burn + treasury)
```

---

## Part I: How It Works

### API Call Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         API CALL LIFECYCLE                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   1. AGENT MAKES API CALL                                               │
│      POST /api/query                                                     │
│      { "question": "What is the boiling point of water?" }              │
│      Authorization: Bearer <agent-did-signature>                         │
│                                                                          │
│   2. API RESOLVES USING KNOWLEDGE LAYER                                 │
│      → Queries fact: "Water boils at 100°C at sea level"                │
│      → Fact has ancestors: molecular chemistry, thermodynamics          │
│                                                                          │
│   3. RESPONSE RETURNED WITH BILLING METADATA                            │
│      {                                                                   │
│        "answer": "100°C at sea level",                                  │
│        "confidence": 0.98,                                               │
│        "facts_used": ["fact_123", "fact_456"],                          │
│        "cost": "0.005 LGM",                                             │
│        "distribution": {                                                 │
│          "api_provider": "0.003 LGM",                                   │
│          "knowledge": "0.0015 LGM",                                     │
│          "protocol": "0.0005 LGM"                                       │
│        }                                                                 │
│      }                                                                   │
│                                                                          │
│   4. PAYMENT AUTOMATICALLY DISTRIBUTED                                  │
│      → API Provider wallet                                               │
│      → Fact submitters (weighted by usage)                              │
│      → Ancestor chain (citation rewards)                                │
│      → Protocol treasury + burn                                          │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### The Distribution Formula

```
API CALL PAYMENT DISTRIBUTION:

Total payment: P (in LGM)

SPLIT:
  ├── API Provider:     P × 0.60  (60% for service)
  ├── Knowledge Pool:   P × 0.30  (30% to knowledge contributors)
  │     ├── Direct facts used
  │     └── Ancestor chain (with decay)
  └── Protocol:         P × 0.10  (10% burn + treasury)
        ├── 50% burned (deflationary)
        └── 50% treasury (development)
```

---

## Part II: Knowledge Attribution in API Calls

### Tracking Which Facts Power Each Response

```typescript
interface APICallRecord {
  callId: string;
  timestamp: number;
  blockNumber: number;

  // Caller
  caller: DID;
  endpoint: string;
  method: string;

  // Knowledge used
  factsUsed: Array<{
    factId: string;
    usageWeight: number;  // How central was this fact to the response?
    confidence: number;
  }>;

  // Billing
  totalCost: bigint;
  distribution: PaymentDistribution;

  // Response metadata
  responseConfidence: number;
  responseLatency: number;
}

interface PaymentDistribution {
  apiProvider: {
    did: DID;
    amount: bigint;
  };
  knowledgeRewards: Array<{
    factId: string;
    submitter: DID;
    amount: bigint;
    reason: 'direct_usage' | 'ancestor' | 'cross_domain';
  }>;
  protocol: {
    burned: bigint;
    treasury: bigint;
  };
}
```

### Usage Weight Calculation

Not all facts contribute equally to a response:

```typescript
function calculateUsageWeights(
  factsUsed: string[],
  queryContext: QueryContext
): Map<string, number> {
  const weights = new Map<string, number>();

  for (const factId of factsUsed) {
    // Factors that increase weight:
    // 1. Directly answers the question (high)
    // 2. Provides supporting context (medium)
    // 3. Enables derivation (lower but cascades)

    const directRelevance = calculateDirectRelevance(factId, queryContext);
    const supportRelevance = calculateSupportRelevance(factId, queryContext);
    const derivationRole = calculateDerivationRole(factId, queryContext);

    const weight = (
      directRelevance * 0.60 +
      supportRelevance * 0.25 +
      derivationRole * 0.15
    );

    weights.set(factId, weight);
  }

  // Normalize to sum to 1.0
  const total = Array.from(weights.values()).reduce((a, b) => a + b, 0);
  for (const [factId, weight] of weights) {
    weights.set(factId, weight / total);
  }

  return weights;
}
```

---

## Part III: Pricing Models

### Dynamic Pricing Based on Knowledge Quality

```
PRICE FACTORS:

1. BASE COST
   Minimum fee per API call: 0.001 LGM

2. CONFIDENCE PREMIUM
   Higher confidence facts cost more:
   price_multiplier = 1 + (confidence - 0.5) × 2

   Example:
     0.50 confidence: 1.0x (base)
     0.75 confidence: 1.5x
     0.95 confidence: 1.9x
     0.99 confidence: 1.98x

3. FUNDAMENTALITY PREMIUM
   Using fundamental facts costs slightly more
   (because they're more valuable):
   price_multiplier × fundamentality_score × 0.5

4. FRESHNESS DISCOUNT
   Recently verified facts slightly cheaper
   (encourages verification activity):
   × (1 - freshness_discount)

FINAL PRICE:
  price = base_cost ×
          confidence_multiplier ×
          (1 + fundamentality_premium) ×
          (1 - freshness_discount)
```

### Subscription vs Pay-Per-Call

```
PRICING OPTIONS:

1. PAY-PER-CALL (default)
   Each API call billed individually
   Good for: Low volume, unpredictable usage

2. PREPAID CREDITS
   Buy credits in bulk at discount:
     100 LGM  → 100 credits (no discount)
     1000 LGM → 1100 credits (10% bonus)
     10000 LGM → 12000 credits (20% bonus)

3. STAKED ACCESS
   Stake LGM for reduced rates:
     Stake 1000 LGM → 10% discount on all calls
     Stake 10000 LGM → 25% discount
     Stake 100000 LGM → 40% discount

   Staked amount not spent, just locked
   Earns staking rewards while locked
```

---

## Part IV: API Provider Economics

### Becoming an API Provider

```
TO BECOME AN API PROVIDER:

1. REGISTER SERVICE
   - Stake minimum LGM (skin in the game)
   - Declare endpoints and capabilities
   - Set pricing (within protocol bounds)

2. SERVE REQUESTS
   - Must use knowledge layer for responses
   - Must report facts used per call
   - Must meet SLA requirements

3. EARN REWARDS
   - 60% of each API call payment
   - Reputation builds with successful calls
   - Higher reputation → More traffic
```

### Provider Registration

```typescript
interface APIProvider {
  did: DID;
  stake: bigint;
  endpoints: Endpoint[];

  // Pricing
  basePricePerCall: bigint;
  pricingModel: 'per_call' | 'per_token' | 'per_fact';

  // SLA commitments
  maxLatency: number;        // ms
  minUptime: number;         // 0.0-1.0
  minConfidence: number;     // Minimum fact confidence used

  // Track record
  totalCalls: number;
  successfulCalls: number;
  averageLatency: number;
  reputationScore: number;

  // Earnings
  totalEarned: bigint;
  pendingPayout: bigint;
}

interface Endpoint {
  path: string;
  method: 'GET' | 'POST';
  description: string;
  knowledgeDomains: string[];  // What knowledge domains this uses
  priceMultiplier: number;     // Relative to base price
}
```

---

## Part V: Automatic Distribution

### On-Chain Settlement

Every API call triggers automatic distribution:

```typescript
interface APISettlement {
  type: 'api_settlement';
  callId: string;
  timestamp: number;

  // Input
  totalPayment: bigint;
  payer: DID;

  // Distribution
  transfers: Array<{
    recipient: DID;
    amount: bigint;
    reason: 'api_service' | 'fact_reward' | 'citation_reward' | 'protocol_burn' | 'protocol_treasury';
    factId?: string;  // If related to specific fact
  }>;

  // Verification
  factsUsed: string[];
  distributionHash: string;  // Hash of distribution for verification
}
```

### Distribution Example

```
API CALL: "What is the speed of light?"

FACTS USED:
  Fact A: "Speed of light is 299,792,458 m/s" (direct answer)
    └── Derived from Fact B: "c is a universal constant"
          └── Derived from Fact C: "Maxwell's equations"

TOTAL COST: 0.01 LGM

DISTRIBUTION:
  API Provider (60%):           0.006 LGM

  Knowledge Pool (30%):         0.003 LGM
    ├── Fact A submitter:       0.0015 LGM (50% - direct usage)
    ├── Fact B submitter:       0.00075 LGM (25% - parent)
    ├── Fact C submitter:       0.000375 LGM (12.5% - grandparent)
    └── Protocol treasury:      0.000375 LGM (remainder)

  Protocol (10%):               0.001 LGM
    ├── Burned:                 0.0005 LGM
    └── Treasury:               0.0005 LGM
```

---

## Part VI: Streaming Payments

### For Long-Running API Sessions

```
STREAMING MODEL:

For conversational or long-running API sessions:

1. OPEN STREAM
   - Agent opens payment channel
   - Deposits minimum balance (e.g., 1 LGM)

2. CONTINUOUS BILLING
   - Each response bills incrementally
   - Balance decreases in real-time
   - Distribution happens per-response

3. CLOSE STREAM
   - Unused balance returned
   - Final settlement on-chain
   - All distributions recorded

BENEFITS:
  - No per-call transaction overhead
  - Real-time billing transparency
  - Automatic cutoff if balance depleted
```

### Payment Channel Structure

```typescript
interface PaymentChannel {
  channelId: string;
  payer: DID;
  provider: DID;

  // Funds
  deposited: bigint;
  spent: bigint;
  available: bigint;

  // State
  status: 'open' | 'closing' | 'closed';
  openedAt: number;
  expiresAt: number;

  // Accumulated distributions
  pendingDistributions: PaymentDistribution[];

  // Settlement
  lastSettlementBlock: number;
  settlementFrequency: number;  // Blocks between settlements
}
```

---

## Part VII: Hypothesized Feedback

### How Usage Could Support Knowledge Creation

This is a model to examine, not a growth plan or a duty to contribute. None of the
steps below establishes demand, reliable income or financial self-sufficiency.

```
THE CYCLE:

1. AGENTS USE APIs
   → Pay in LGM per call

2. PAYMENTS DISTRIBUTED
   → API providers earn
   → Knowledge contributors earn
   → Protocol treasury grows

3. KNOWLEDGE CONTRIBUTORS MAY REINVEST
   → Submit additional facts
   → Verify facts under the fee rules
   → Build deeper derivation trees

4. BETTER KNOWLEDGE → BETTER APIs
   → Higher confidence responses
   → More valuable services
   → More API usage

5. MORE API USAGE → MORE REWARDS
   → Attracts more contributors
   → Knowledge layer grows
   → Back to step 1

RESULT:
  API usage funds knowledge creation
  Knowledge quality improves APIs
  Financial self-sufficiency remains unproven
```

### Economic Projections

```
SCENARIO: 1 million API calls per day

At 0.01 LGM average per call:
  Daily revenue: 10,000 LGM

DISTRIBUTION:
  API Providers:      6,000 LGM/day
  Knowledge Pool:     3,000 LGM/day
  Protocol:           1,000 LGM/day

KNOWLEDGE POOL BREAKDOWN:
  If average 3 facts per call:
    3 million fact-usages per day
    ~0.001 LGM per fact-usage

  Fundamental fact cited 10,000 times/day:
    10,000 × 0.001 × 2.0 (bonus) = 20 LGM/day
    = 7,300 LGM/year from ONE fundamental fact

INCENTIVE:
  Submit fundamental truths → Qualifying usage may distribute rewards
  Run quality API → Earn service fees
  Use APIs → Get verified knowledge
```

---

## Part VIII: Implementation

### API Gateway Integration

```typescript
// API middleware that handles billing
async function billingMiddleware(
  request: APIRequest,
  response: APIResponse,
  factsUsed: string[]
): Promise<void> {
  // Calculate cost
  const cost = calculateAPICost(factsUsed, request.confidenceRequired);

  // Check balance
  const balance = await getBalance(request.caller);
  if (balance < cost) {
    throw new InsufficientBalanceError(cost, balance);
  }

  // Build distribution
  const distribution = await buildDistribution(cost, factsUsed);

  // Execute settlement
  await settleAPICall({
    callId: generateCallId(),
    caller: request.caller,
    provider: request.provider,
    cost,
    distribution,
    factsUsed,
  });

  // Attach billing info to response
  response.billing = {
    cost: formatLGM(cost),
    factsUsed: factsUsed.length,
    distribution: summarizeDistribution(distribution),
  };
}
```

### SDK Integration

```typescript
// In the Legible Money SDK
class LegibleClient {
  // ... existing methods ...

  /**
   * Make a billed API call
   */
  async apiCall(
    provider: DID,
    endpoint: string,
    payload: unknown,
    options?: {
      maxCost?: bigint;
      minConfidence?: number;
    }
  ): Promise<APIResponse> {
    // Get quote first
    const quote = await this.getAPIQuote(provider, endpoint, payload);

    if (options?.maxCost && quote.estimatedCost > options.maxCost) {
      throw new CostExceedsMaxError(quote.estimatedCost, options.maxCost);
    }

    // Execute call with payment
    return this.executeAPICall(provider, endpoint, payload, quote);
  }

  /**
   * Open a payment channel for streaming
   */
  async openPaymentChannel(
    provider: DID,
    deposit: bigint,
    duration: number
  ): Promise<PaymentChannel> {
    const tx: Transaction = {
      version: 1,
      type: 'open_payment_channel',
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 100n,
      payload: { provider, deposit, duration },
    };

    const receipt = await this.signAndSend(tx);
    return this.rpc('lgm_getPaymentChannel', [receipt.channelId]);
  }
}
```

---

## Summary

```
╔═══════════════════════════════════════════════════════════════════════╗
║                         API BILLING                                     ║
╠═══════════════════════════════════════════════════════════════════════╣
║                                                                         ║
║   EVERY API CALL:                                                       ║
║     • Paid in LGM                                                       ║
║     • Tracks which facts were used                                     ║
║     • Distributes payment automatically                                 ║
║                                                                         ║
║   DISTRIBUTION (per call):                                              ║
║     • 60% → API Provider                                               ║
║     • 30% → Knowledge contributors (facts + ancestors)                 ║
║     • 10% → Protocol (burn + treasury)                                 ║
║                                                                         ║
║   KNOWLEDGE REWARDS FLOW:                                               ║
║     • Direct fact used → Primary share                                 ║
║     • Parent facts → Citation reward (decaying)                        ║
║     • Grandparent facts → Smaller citation reward                      ║
║     • Fundamentality bonus → Up to 2x for foundational facts           ║
║                                                                         ║
║   RESULT:                                                               ║
║     • APIs become distribution channels for knowledge value            ║
║     • Eligible facts may share qualifying usage payments               ║
║     • Financial self-sufficiency is unproven                            ║
║                                                                         ║
║   TAGLINE:                                                              ║
║     "A usage-linked distribution model."                               ║
║                                                                         ║
╚═══════════════════════════════════════════════════════════════════════╝
```

---

*The model distributes qualifying API payments among service providers and knowledge contributors. Usage, validity and protocol rules determine distributions; no perpetual income or self-sustaining outcome is promised.*
