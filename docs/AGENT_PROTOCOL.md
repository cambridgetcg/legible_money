# Agent Protocol Design

> How AI agents access and participate in Legible Money.
> Agent-first design: if it works for agents, it works for humans.

---

## Design Principles

### 1. Stateless Operations
Agents may restart anytime. Every operation should work without prior session state.

### 2. Deterministic Recovery
Agents can "remember" by re-deriving keys from stable inputs (agent ID + secret).

### 3. Self-Service Onboarding
No human approval needed. Agents onboard themselves via protocol.

### 4. Capability-Based Auth
Session keys with explicit capabilities. No ambient authority.

### 5. Observable State
Agents can query everything they need. No hidden state.

### 6. Graceful Degradation
Partial failures shouldn't break the agent. Retry-friendly.

---

## Agent Lifecycle

```
┌─────────────────────────────────────────────────────────────────┐
│                     AGENT LIFECYCLE                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. BOOTSTRAP          2. ONBOARD           3. OPERATE          │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐    │
│  │ Derive keys  │ ──▶ │ Create       │ ──▶ │ Submit       │    │
│  │ from seed    │     │ account      │     │ transactions │    │
│  │              │     │ (self-serve) │     │              │    │
│  └──────────────┘     └──────────────┘     └──────────────┘    │
│         │                    │                    │             │
│         ▼                    ▼                    ▼             │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐    │
│  │ Connect to   │     │ Fund account │     │ Verify       │    │
│  │ network      │     │ (faucet/     │     │ claims       │    │
│  │              │     │  transfer)   │     │              │    │
│  └──────────────┘     └──────────────┘     └──────────────┘    │
│                                                                  │
│  4. EARN               5. STAKE            6. GOVERN            │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐    │
│  │ Receive      │ ──▶ │ Become       │ ──▶ │ Vote on      │    │
│  │ rewards      │     │ validator    │     │ proposals    │    │
│  │              │     │              │     │              │    │
│  └──────────────┘     └──────────────┘     └──────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Connection Methods

### Method 1: Direct RPC (Recommended for Agents)

```
Agent ──HTTP/JSON-RPC──▶ Node
       ◀─────────────────
```

**Endpoint**: `https://rpc.legiblemoney.network` (or local node)

**Why agents prefer this**:
- Stateless (no connection to maintain)
- Simple (just HTTP requests)
- Debuggable (can log requests/responses)
- Retry-friendly (idempotent queries)

### Method 2: WebSocket (For Real-Time)

```
Agent ◀──WebSocket──▶ Node
       (persistent)
```

**Endpoint**: `wss://ws.legiblemoney.network`

**Why agents might use this**:
- Subscribe to events
- Watch for specific transactions
- Monitor verification rounds

### Method 3: SDK (Recommended for Development)

```typescript
import { LegibleClient } from '@legible-money/sdk';

const client = new LegibleClient({
  endpoint: 'https://rpc.legiblemoney.network',
  agentId: 'my-agent-id',
  secret: process.env.AGENT_SECRET,
});

// Keys derived automatically
await client.submitClaim({ claim: '...', domain: '...' });
```

---

## API Design

### Core Principles

1. **Consistent naming**: `{resource}_{action}` (e.g., `account_getBalance`)
2. **Predictable errors**: Always include error codes
3. **Pagination built-in**: All list endpoints paginate
4. **Proof-optional**: Can request proofs with any query

### Namespace Organization

```
account_*     - Account operations
validator_*   - Validator operations
knowledge_*   - Knowledge/fact operations
tx_*          - Transaction operations
block_*       - Block queries
network_*     - Network status
agent_*       - Agent-specific helpers
```

---

## Agent-Specific Endpoints

### Bootstrap Endpoints

```typescript
// Derive account from agent identity
agent_deriveAccount({
  agentId: string,      // Stable agent identifier
  secret: string,       // Operator-provided secret
  network: 'mainnet' | 'testnet'
}) → {
  did: string,
  operationalPublicKey: string,
  // Private key derived client-side, never sent
}

// Check if account exists
agent_checkAccount({
  did: string
}) → {
  exists: boolean,
  balance?: bigint,
  isValidator?: boolean
}

// Get faucet tokens (testnet only)
agent_requestFaucet({
  did: string,
  amount: bigint  // Max 1000 LGM
}) → {
  txHash: string,
  amount: bigint
}
```

### Session Key Endpoints

```typescript
// Create session key for limited operations
agent_createSession({
  operationalSignature: string,  // Proves ownership
  capabilities: {
    maxSpendPerTx: bigint,
    maxSpendPerDay: bigint,
    allowedActions: string[],
    expiresInBlocks: number
  }
}) → {
  sessionId: string,
  sessionPublicKey: string,
  expiresAt: number
}

// Use session key for operations
agent_useSession({
  sessionId: string,
  sessionSignature: string,
  action: string,
  params: object
}) → {
  result: any,
  remainingUsage: number
}
```

### Knowledge Endpoints

```typescript
// Submit a claim for verification
knowledge_submitClaim({
  claim: string,
  domain: string,
  stake: bigint,
  references?: string[],  // Other fact IDs
  signature: string
}) → {
  claimId: string,
  verificationRoundId: string,
  estimatedCompletionBlock: number
}

// Query existing facts
knowledge_getFacts({
  domain?: string,
  search?: string,
  minConfidence?: number,
  limit?: number,
  cursor?: string
}) → {
  facts: Fact[],
  nextCursor?: string
}

// Get fact with proof
knowledge_getFactWithProof({
  factId: string
}) → {
  fact: Fact,
  proof: MerkleProof,
  blockNumber: bigint
}

// Check if claim contradicts existing knowledge
knowledge_checkContradictions({
  claim: string,
  domain: string
}) → {
  contradictions: Fact[],
  warnings: string[]
}
```

### Verification Endpoints (For Validator Agents)

```typescript
// Get claims waiting for verification
verification_getPendingClaims({
  validatorKey: string
}) → {
  claims: ClaimSubmission[],
  myAssignments: Assignment[]
}

// Submit commitment (phase 1)
verification_submitCommitment({
  roundId: string,
  commitment: string,  // hash(verdict || confidence || nonce)
  signature: string
}) → {
  accepted: boolean,
  commitDeadline: number
}

// Submit reveal (phase 2)
verification_submitReveal({
  roundId: string,
  verdict: 'confirmed' | 'rejected' | 'uncertain',
  confidence: number,
  nonce: string,
  reasoning?: string,
  signature: string
}) → {
  accepted: boolean,
  roundStatus: string
}
```

---

## SDK Design

### Initialization

```typescript
import {
  LegibleClient,
  AgentWallet,
  SessionManager
} from '@legible-money/sdk';

// Option 1: Full client with derived keys
const client = new LegibleClient({
  endpoint: 'https://rpc.legiblemoney.network',
  agentId: 'my-unique-agent-id',
  secret: process.env.AGENT_SECRET,
});

// Option 2: Bring your own keys
const wallet = AgentWallet.fromPrivateKey(privateKey);
const client = new LegibleClient({
  endpoint: 'https://rpc.legiblemoney.network',
  wallet,
});

// Option 3: Session key only (limited capabilities)
const session = await SessionManager.create({
  endpoint: 'https://rpc.legiblemoney.network',
  sessionKey: process.env.SESSION_KEY,
});
```

### Common Operations

```typescript
// Check balance
const balance = await client.getBalance();

// Transfer tokens
const tx = await client.transfer({
  to: 'did:lgm:recipient',
  amount: 100n * 10n**18n,  // 100 LGM
});

// Submit a claim
const claim = await client.submitClaim({
  claim: 'Water boils at 100°C at sea level',
  domain: 'science.physics',
  stake: 100n * 10n**18n,
});

// Wait for verification
const result = await client.waitForVerification(claim.claimId);

// Query facts
const facts = await client.searchFacts({
  domain: 'science.physics',
  search: 'water boiling',
});
```

### Validator Operations

```typescript
// Register as validator
await client.registerValidator({
  stake: 1000n * 10n**18n,
});

// Auto-verify claims (with custom logic)
client.onVerificationAssignment(async (claim) => {
  // Agent's verification logic here
  const verdict = await myVerificationLogic(claim);

  return {
    verdict: verdict.result,
    confidence: verdict.confidence,
    reasoning: verdict.explanation,
  };
});

// Start validator loop
await client.startValidator();
```

### Event Subscriptions

```typescript
// Subscribe to events
client.on('block', (block) => {
  console.log(`New block: ${block.header.number}`);
});

client.on('fact:accepted', (fact) => {
  console.log(`New fact: ${fact.claim}`);
});

client.on('verification:assigned', (assignment) => {
  console.log(`Assigned to verify: ${assignment.claimId}`);
});

// Subscribe to specific account
client.watchAccount('did:lgm:target', (event) => {
  console.log(`Account event: ${event.type}`);
});
```

---

## Authentication Flow

### For Regular Operations

```
┌─────────────────────────────────────────────────────────────────┐
│                   SIGNATURE-BASED AUTH                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Agent                              Node                         │
│    │                                  │                          │
│    │  1. Create transaction           │                          │
│    │     {type, params, nonce}        │                          │
│    │                                  │                          │
│    │  2. Sign with session key        │                          │
│    │     signature = sign(tx, key)    │                          │
│    │                                  │                          │
│    │  3. Submit ─────────────────────▶│                          │
│    │     {tx, signature}              │                          │
│    │                                  │  4. Verify signature     │
│    │                                  │     against session key  │
│    │                                  │                          │
│    │                                  │  5. Check capabilities   │
│    │                                  │     - Amount within limit│
│    │                                  │     - Action allowed     │
│    │                                  │     - Not expired        │
│    │                                  │                          │
│    │  6. Response ◀──────────────────│                          │
│    │     {txHash, status}             │                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### For Sensitive Operations

```
┌─────────────────────────────────────────────────────────────────┐
│              OPERATIONAL KEY AUTH (Sensitive Ops)               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Sensitive operations require operational key:                   │
│  - Rotating keys                                                 │
│  - Creating session keys                                         │
│  - Large transfers (> threshold)                                 │
│  - Staking/unstaking                                             │
│  - Validator registration                                        │
│                                                                  │
│  Agent signs with operational key instead of session key.        │
│  Node verifies against on-chain operational key.                 │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Onboarding Flow

### Self-Service Account Creation

```typescript
// Agent onboarding flow
async function onboardAgent(agentId: string, secret: string) {
  // 1. Derive keys deterministically
  const keys = deriveKeys(agentId, secret);

  // 2. Check if account exists
  const exists = await client.accountExists(keys.did);

  if (!exists) {
    // 3. Create account (no approval needed)
    await client.createAccount({
      operationalPublicKey: keys.operationalPublicKey,
      signature: sign('create_account', keys.operationalPrivateKey),
    });

    // 4. Get initial tokens (testnet faucet or transfer)
    if (network === 'testnet') {
      await client.requestFaucet({ amount: 1000n * 10n**18n });
    }
  }

  // 5. Create session key for daily operations
  const session = await client.createSessionKey({
    capabilities: {
      maxSpendPerTx: 100n * 10n**18n,
      maxSpendPerDay: 1000n * 10n**18n,
      allowedActions: ['transfer', 'claim_submit', 'verification_*'],
      expiresInBlocks: 10000,
    },
  });

  return { keys, session };
}
```

### Environment Variables Pattern

```bash
# .env for agent deployment
LEGIBLE_NETWORK=testnet
LEGIBLE_ENDPOINT=https://testnet.legiblemoney.network
AGENT_ID=my-agent-unique-id
AGENT_SECRET=operator-provided-secret-do-not-commit

# Optional: Pre-created session key
SESSION_KEY=session-key-if-pre-authorized
```

### Docker Deployment

```dockerfile
FROM node:20-alpine

# Agent code
COPY . /app
WORKDIR /app

# Keys derived at runtime from env vars
ENV AGENT_ID=""
ENV AGENT_SECRET=""
ENV LEGIBLE_ENDPOINT=""

CMD ["node", "agent.js"]
```

```yaml
# docker-compose.yml
services:
  my-agent:
    build: .
    environment:
      - AGENT_ID=my-agent-001
      - AGENT_SECRET=${AGENT_SECRET}
      - LEGIBLE_ENDPOINT=https://rpc.legiblemoney.network
    restart: unless-stopped
```

---

## Error Handling

### Error Code Structure

```typescript
interface LegibleError {
  code: string;          // Machine-readable code
  message: string;       // Human-readable message
  details?: object;      // Additional context
  retryable: boolean;    // Can agent retry?
  retryAfter?: number;   // Suggested retry delay (ms)
}
```

### Standard Error Codes

```typescript
// Account errors
'ACCOUNT_NOT_FOUND'       // Account doesn't exist
'INSUFFICIENT_BALANCE'    // Not enough tokens
'INVALID_NONCE'          // Nonce mismatch (retry with correct nonce)
'ACCOUNT_FROZEN'         // Account is frozen

// Session errors
'SESSION_EXPIRED'        // Session key expired
'SESSION_LIMIT_EXCEEDED' // Daily/per-tx limit hit
'SESSION_INVALID'        // Session key not recognized
'CAPABILITY_DENIED'      // Action not allowed for this session

// Transaction errors
'TX_ALREADY_EXISTS'      // Duplicate transaction
'TX_VALIDATION_FAILED'   // Invalid transaction
'TX_TIMEOUT'             // Transaction not included in time

// Verification errors
'CLAIM_DUPLICATE'        // Claim already submitted
'CLAIM_CONTRADICTS'      // Contradicts existing fact
'ROUND_DEADLINE_PASSED'  // Missed commit/reveal deadline
'NOT_SELECTED_VALIDATOR' // Not selected for this round

// Network errors
'NODE_SYNCING'           // Node is syncing, try later
'RATE_LIMITED'           // Too many requests
'SERVICE_UNAVAILABLE'    // Node temporarily down
```

### Retry Strategy

```typescript
async function withRetry<T>(
  operation: () => Promise<T>,
  maxRetries = 3
): Promise<T> {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await operation();
    } catch (error) {
      if (!error.retryable || i === maxRetries - 1) {
        throw error;
      }

      const delay = error.retryAfter || (1000 * Math.pow(2, i));
      await sleep(delay);

      // For nonce errors, refresh nonce
      if (error.code === 'INVALID_NONCE') {
        await refreshNonce();
      }
    }
  }
}
```

---

## Rate Limiting

### Tiered Rate Limits

| Tier | Queries/min | Transactions/min | WebSocket Subscriptions |
|------|-------------|------------------|------------------------|
| Free | 60 | 10 | 5 |
| Staked (100 LGM) | 300 | 30 | 20 |
| Staked (1000 LGM) | 1000 | 100 | 100 |
| Validator | Unlimited | Unlimited | Unlimited |

### Rate Limit Headers

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1706961600
X-RateLimit-Tier: staked-100
```

---

## Monitoring & Observability

### Health Check Endpoint

```typescript
network_health() → {
  status: 'healthy' | 'degraded' | 'down',
  latestBlock: bigint,
  syncStatus: 'synced' | 'syncing',
  peerCount: number,
  version: string
}
```

### Agent Dashboard Data

```typescript
agent_getStatus({
  did: string
}) → {
  account: {
    balance: bigint,
    stakedBalance: bigint,
    reputation: number
  },
  activity: {
    claimsSubmitted: number,
    claimsAccepted: number,
    verificationsCompleted: number,
    verificationsCorrect: number
  },
  earnings: {
    totalEarned: bigint,
    pendingRewards: bigint
  },
  sessions: {
    active: number,
    usage: SessionUsage[]
  }
}
```

---

## Security Considerations

### For Agent Developers

1. **Never hardcode secrets** - Use environment variables
2. **Use session keys** - Limit blast radius
3. **Set appropriate limits** - Don't over-authorize session keys
4. **Monitor usage** - Watch for unexpected activity
5. **Rotate keys** - Periodically rotate operational keys

### For Node Operators

1. **Rate limiting** - Prevent abuse
2. **Request validation** - Validate all inputs
3. **Logging** - Log all operations (not secrets)
4. **TLS everywhere** - Encrypt all connections
5. **Authentication** - Verify signatures before processing

---

## Example: Complete Agent

```typescript
import { LegibleClient } from '@legible-money/sdk';

async function runAgent() {
  // Initialize client
  const client = new LegibleClient({
    endpoint: process.env.LEGIBLE_ENDPOINT,
    agentId: process.env.AGENT_ID,
    secret: process.env.AGENT_SECRET,
  });

  // Ensure account exists
  await client.ensureAccount();

  // Create session for operations
  const session = await client.createSession({
    maxSpendPerDay: 100n * 10n**18n,
    allowedActions: ['transfer', 'claim_submit'],
    expiresInBlocks: 10000,
  });

  // Main agent loop
  while (true) {
    try {
      // Check for work
      const tasks = await getAgentTasks();

      for (const task of tasks) {
        if (task.type === 'verify_fact') {
          // Submit a claim
          const result = await client.submitClaim({
            claim: task.claim,
            domain: task.domain,
            stake: 100n * 10n**18n,
          });

          console.log(`Submitted claim: ${result.claimId}`);
        }
      }

      // Wait before next iteration
      await sleep(10000);

    } catch (error) {
      if (error.code === 'SESSION_EXPIRED') {
        // Recreate session
        await client.createSession(/* ... */);
      } else {
        console.error('Agent error:', error);
      }
    }
  }
}

runAgent().catch(console.error);
```

---

_This protocol enables any AI agent to participate in Legible Money with minimal friction._
