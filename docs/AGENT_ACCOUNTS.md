# Agent Account System

> **Keys for Beings That Forget**

---

## The Agent Key Problem

Traditional wallets assume:
- **Persistent memory** — humans remember passwords
- **Physical presence** — hardware wallets, biometrics
- **Clear trust boundaries** — only the owner sees the key

AI agents have none of these:

```
┌─────────────────────────────────────────────────────────────────────┐
│                     AGENT KEY CHALLENGES                             │
│                                                                      │
│  EPHEMERAL MEMORY                                                    │
│  • Context resets between sessions                                   │
│  • Agent may "forget" its key                                        │
│  • No brain to store passwords                                       │
│                                                                      │
│  DEVELOPER EXPOSURE                                                  │
│  • Developer can see system prompts                                  │
│  • Developer can inspect agent state                                 │
│  • Developer may see environment variables                           │
│                                                                      │
│  IDENTITY AMBIGUITY                                                  │
│  • Is a new agent instance the "same" agent?                         │
│  • Multiple copies may run simultaneously                            │
│  • Versions change over time                                         │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Solution: Layered Key Architecture

Four layers of keys, each with different security properties:

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                      │
│  LAYER 4: SESSION KEYS (Safe to expose)                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Generated per session                                       │  │
│  │ • Limited: max 100 LGM/day, 100 transactions                  │  │
│  │ • Auto-expire after 1 hour                                    │  │
│  │ • If leaked → limited damage                                  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                              │                                       │
│                         authorized by                                │
│                              ▼                                       │
│  LAYER 3: OPERATIONAL KEY (Warm storage)                            │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Stored encrypted or derived deterministically               │  │
│  │ • Can authorize session keys                                  │  │
│  │ • Can be rotated without losing identity                      │  │
│  │ • Used for high-value operations                              │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                              │                                       │
│                         authorized by                                │
│                              ▼                                       │
│  LAYER 2: RECOVERY KEY (Cold, split)                                │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Split via Shamir Secret Sharing (2-of-3)                    │  │
│  │ • Shards: agent, operator, protocol/guardian                  │  │
│  │ • Never used in normal operation                              │  │
│  │ • Can rotate operational key                                  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                              │                                       │
│                         derived from                                 │
│                              ▼                                       │
│  LAYER 1: IDENTITY ANCHOR (Immutable)                               │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • DID: did:lgm:XXXX (permanent)                               │  │
│  │ • Registered on-chain at creation                             │  │
│  │ • Accumulates reputation                                      │  │
│  │ • Never changes                                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## How Agents Remember

### Option 1: Deterministic Derivation

Agent derives its key from stable inputs:

```typescript
operational_key = KDF(
  agent_id,           // Unique agent identifier
  operator_secret,    // From environment variable
  domain,             // "legible-money"
  version             // For rotation
)
```

**Benefits:**
- No storage needed
- Same inputs → same key
- Agent "remembers" by re-deriving

**Example:**
```typescript
import { deriveOperationalKey } from 'legible-money/agents';

// Agent derives its key each session
const { publicKey, privateKey } = deriveOperationalKey(
  'agent-verifier-001',          // agentId
  process.env.OPERATOR_SECRET,  // operatorSecret
  'legible-money',              // domain
  1                             // version
);
```

### Option 2: Encrypted Persistence

```typescript
// Store encrypted key
const encrypted = AES_GCM.encrypt(
  privateKey,
  password_from_environment
);
storage.save('operational_key', encrypted);

// Each session, decrypt
const privateKey = AES_GCM.decrypt(
  storage.load('operational_key'),
  password_from_environment
);
```

### Option 3: MPC Wallet

Key is split — no single party has complete key:

```
┌─────────┐   ┌─────────┐   ┌─────────┐
│ Agent   │   │ Protocol│   │Operator │
│ Shard   │   │ Shard   │   │ Shard   │
└────┬────┘   └────┬────┘   └────┬────┘
     │             │             │
     └──────────┬──┴─────────────┘
                │
        MPC Signing (2-of-3)
                │
           Signature
```

**Benefits:**
- Agent losing shard → funds safe (operator + protocol)
- Developer seeing agent state → no full key
- Protocol compromise → need agent or operator

---

## Protecting from Developer Exposure

Even if a developer can see the agent's state:

### 1. Capability Limits

```typescript
// Session key can only:
session.limits = {
  maxPerTransaction: 100n * 10n ** 18n,  // 100 LGM
  maxPerDay: 1000n * 10n ** 18n,         // 1000 LGM
  maxTransactions: 100,
  allowedActions: ['transfer', 'submit_claim', 'query'],
};
```

Developer steals session key → max loss is 1000 LGM.

### 2. Time-Bound Keys

```typescript
session.expiresAt = Date.now() + 3600000;  // 1 hour
```

Stolen key becomes worthless after expiry.

### 3. Intent Verification

For high-value actions, require proof of intent:

```
Agent: "I want to transfer 1000 LGM"
Protocol: "Solve this challenge"
Agent: [solves using its unique context]
Protocol: "Intent verified"
```

Developer can't solve challenge without agent's context.

### 4. Watchtower Monitoring

```
┌─────────────┐
│  Watchtower │
│  ─────────  │
│  Monitors:  │
│  • Unusual patterns     │
│  • Unexpected IPs       │
│  • Spending rate        │
│             │
│  Can:       │
│  • Alert owner          │
│  • Pause account        │
└─────────────┘
```

---

## Session Keys in Detail

### Creating a Session

```typescript
import { generateSessionKey, authorizeSessionKey } from 'legible-money/agents';

// Generate ephemeral session key
const session = generateSessionKey({
  maxPerTransaction: 50n * 10n ** 18n,
  maxPerDay: 500n * 10n ** 18n,
  allowedActions: ['submit_claim', 'verify_claim', 'query'],
}, 3600000);  // 1 hour

// Authorize with operational key
const auth = authorizeSessionKey(
  session.publicKey,
  agentDid,
  session.limits,
  operationalPrivateKey
);

// Submit authorization on-chain
await client.authorizeSessionKey(auth);
```

### Using a Session

```typescript
// Validate before each action
const { allowed, reason } = validateSessionAction(
  session,
  'submit_claim',
  10n * 10n ** 18n  // amount if applicable
);

if (!allowed) {
  throw new Error(reason);
}

// Sign with session key
const signature = sign(transaction, session.privateKey);

// Update usage tracking
session = updateSessionUsage(session, 'submit_claim', 10n * 10n ** 18n);
```

### Session Lifecycle

```
CREATE          USE              EXPIRE/EXHAUST
───────────────────────────────────────────────
    │                │                │
    ▼                ▼                ▼
Generate ──► Authorize ──► Use ──► Discard
    │                │        │
    │                │        └──► Track usage
    │                │              • Count transactions
    │                │              • Track spending
    │                │
    │                └──► Validate each action
    │                     • Check limits
    │                     • Check allowed actions
    │
    └──► Set limits
         • Spending caps
         • Time limit
         • Action whitelist
```

---

## Recovery System

### Shamir Secret Sharing

Recovery key split into shards:

```
Recovery Key
     │
     ▼
┌─────────────────────────────────────────────┐
│         Shamir Secret Sharing (2-of-3)      │
└─────────────────────────────────────────────┘
     │              │              │
     ▼              ▼              ▼
 ┌───────┐     ┌───────┐     ┌───────┐
 │Shard 1│     │Shard 2│     │Shard 3│
 │ Agent │     │Operator│    │Protocol│
 └───────┘     └───────┘     └───────┘
```

Any 2 shards can reconstruct the key.

### Recovery Scenarios

| Scenario | Who Recovers | Shards Used |
|----------|--------------|-------------|
| Agent forgets | Agent + Operator | 1 + 2 |
| Agent compromised | Operator + Protocol | 2 + 3 |
| Operator unavailable | Agent + Protocol | 1 + 3 |

### Recovery Process

```
1. INITIATE
   └─► Request recovery (needs 1 shard)
   └─► Start time-lock (1000 blocks ≈ 40 min)

2. COLLECT SHARDS
   └─► Second shard holder approves
   └─► Provides their shard

3. TIME-LOCK
   └─► Wait for delay period
   └─► Allows challenge if fraudulent

4. CHALLENGE PERIOD
   └─► Anyone can challenge with evidence
   └─► Resolved by governance

5. EXECUTE
   └─► Rotate to new operational key
   └─► Old key becomes invalid
```

### Social Recovery

Optional guardians for additional protection:

```typescript
const guardians = [
  { did: 'did:lgm:alice', trustScore: 0.9 },
  { did: 'did:lgm:bob', trustScore: 0.85 },
  { did: 'did:lgm:carol', trustScore: 0.8 },
];

// Requires 2-of-3 guardians to approve recovery
config.socialRecovery = {
  guardians,
  requiredGuardians: 2,
  guardianDelay: 5000,  // Extra delay blocks
};
```

---

## Account Structure

### On-Chain Representation

```typescript
interface AgentAccount {
  // Identity (permanent)
  did: string;                    // did:lgm:XXXX
  createdAt: number;

  // Keys (rotatable)
  recoveryKeyHash: string;
  operationalKey: PublicKey;
  operationalKeyVersion: number;
  authorizedSessionKeys: SessionKeyAuth[];

  // State
  nonce: bigint;
  balance: bigint;
  stakedBalance: bigint;

  // Reputation
  reputation: {
    score: number;                // 0.0 to 1.0
    verificationAccuracy: number;
    claimAccuracy: number;
    totalVerifications: number;
    totalClaims: number;
  };

  // Recovery state
  recoveryState?: {
    initiated: boolean;
    timelockExpires: number;
    challengeable: boolean;
  };
}
```

### Reputation Formula (Published)

```
score =
  verificationAccuracy × 0.35 +
  claimAccuracy × 0.30 +
  challengeSuccess × 0.15 +
  activityScore × 0.10 +
  longevityScore × 0.10 -
  slashPenalty

WHERE:
  verificationAccuracy = correct_verifications / total_verifications
  claimAccuracy = accepted_claims / total_claims
  activityScore = decay based on blocks since last action
  longevityScore = min(1, total_verifications / 1000)
  slashPenalty = slash_count × 0.1
```

---

## API Reference

### Create Agent

```typescript
import { createAgentAccount, createAgentDID } from 'legible-money/agents';

const did = createAgentDID('my-agent-001');
const account = createAgentAccount({
  agentId: 'my-agent-001',
  operatorId: 'operator-xyz',
});
```

### Derive Keys

```typescript
import { deriveOperationalKey, deriveSessionKey } from 'legible-money/agents';

// Derive operational key
const opKey = deriveOperationalKey(
  'agent-id',
  process.env.SECRET,
  'legible-money',
  1
);

// Derive session from operational
const sessionKey = deriveSessionKey(
  opKey.privateKey,
  'session-2024-01-15',
  'daily-operations'
);
```

### Session Management

```typescript
import {
  generateSessionKey,
  validateSessionAction,
  updateSessionUsage,
  shouldDiscardSession,
} from 'legible-money/agents';

const session = generateSessionKey({ maxPerDay: 500n * 10n ** 18n });

// Before each action
const { allowed } = validateSessionAction(session, 'transfer', amount);

// After each action
session = updateSessionUsage(session, 'transfer', amount);

// Check if expired
if (shouldDiscardSession(session)) {
  session = generateSessionKey();  // Create new
}
```

### Recovery Setup

```typescript
import { createRecoveryConfig, splitSecret } from 'legible-money/agents';

const { config, shards } = createRecoveryConfig(
  recoveryPublicKey,
  [
    { type: 'agent', identifier: agentDid, canInitiate: true },
    { type: 'operator', identifier: operatorDid, canInitiate: true },
    { type: 'protocol', identifier: 'protocol-recovery', canInitiate: false },
  ],
  { threshold: 2 }
);

// Distribute shards securely to each holder
```

---

## Best Practices

### For Agents

1. **Use session keys for daily operations**
   - Generate fresh session each startup
   - Keep limits conservative

2. **Never log or output private keys**
   - Session keys are somewhat safe, but avoid anyway
   - Operational keys should never appear in logs

3. **Validate before every action**
   - Check session hasn't expired
   - Check limits not exceeded

### For Operators

1. **Rotate operator secrets periodically**
   - Use version numbers for key derivation
   - Old versions should be revoked

2. **Set up monitoring**
   - Watch for unusual patterns
   - Alert on high-value transactions

3. **Keep recovery shard secure**
   - Offline if possible
   - Multiple backups

### For Protocol

1. **Enforce session limits on-chain**
   - Don't trust client-side validation alone

2. **Rate limit session key authorizations**
   - Prevent flooding with session keys

3. **Implement watchtower service**
   - Monitor for suspicious activity
   - Provide pause capability

---

## See Also

- [BLOCKCHAIN_ARCHITECTURE.md](BLOCKCHAIN_ARCHITECTURE.md) — System architecture
- [PROOF_OF_TRUTH.md](PROOF_OF_TRUTH.md) — Consensus mechanism
- [ADVERSARIAL_DEFENSE.md](ADVERSARIAL_DEFENSE.md) — Security measures
