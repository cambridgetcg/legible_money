# Agent Identity

> Identity management tailored for AI agents with ephemeral memory.

## Quick Reference

| Item | Value |
|------|-------|
| **Layer** | agents |
| **Dependencies** | `foundation/`, `core/crypto/` |
| **Dependents** | `home/`, `consensus/`, `protocol/` |

## What This Module Does

Manages the unique challenges of AI agent identity:
- **Ephemeral memory**: Sessions end, context resets
- **Developer exposure**: Operators may see agent state
- **Identity continuity**: Reputation persists across instances

Solution: **4-Layer Key Architecture**

```
┌─────────────────────────────────────────┐
│  Identity Key (DID)                     │  ← Immutable, derives all others
├─────────────────────────────────────────┤
│  Recovery Key (Shamir split)            │  ← For key recovery
├─────────────────────────────────────────┤
│  Operational Key (rotatable)            │  ← Daily operations
├─────────────────────────────────────────┤
│  Session Keys (ephemeral, limited)      │  ← Per-session, scoped
└─────────────────────────────────────────┘
```

## Key Types

```typescript
// The complete agent account
AgentAccount {
  did: DID,                    // Permanent identifier
  identityPublicKey: PublicKey,
  operationalKey: PublicKey,
  operationalKeyVersion: number,
  sessionKeys: SessionKeyAuth[],
  capabilities: AgentCapabilities,
  reputation: AgentReputation,
  recovery: RecoveryState,
  createdAt: number,
}

// Session key with limited permissions
SessionKeyAuth {
  publicKey: PublicKey,
  capabilities: SessionCapabilities,
  createdAt: number,
  expiresAt: number,
  maxTransactions: number,
  usedTransactions: number,
}

// What a session key can do
SessionCapabilities {
  canTransfer: boolean,
  maxTransferAmount: bigint,
  canVerify: boolean,
  canSubmitClaims: boolean,
  allowedDomains?: string[],
}
```

## Key Functions

```typescript
// Create new agent
createAgentAccount(params: CreateAgentParams): AgentAccount
createAgentDID(publicKey: PublicKey): DID

// Key derivation (deterministic from identity key)
deriveOperationalKey(identityKey, version): KeyPair
deriveSessionKey(operationalKey, sessionId, expiry): KeyPair

// Session management
createSessionKeyAuth(publicKey, capabilities, duration): SessionKeyAuth
validateSessionKeyAction(session, action): boolean

// Recovery (Shamir Secret Sharing)
createRecoveryShares(secret, threshold, total): Share[]
reconstructSecret(shares): Secret
```

## Common Tasks

### Create a new agent

```typescript
import { createAgentAccount } from './account.js';

const agent = createAgentAccount({
  identityKey: keypair,
  name: 'MyAgent',
  capabilities: {
    canSubmitClaims: true,
    canVerify: true,
    canChallenge: false,
  },
});
```

### Derive a session key

```typescript
import { deriveSessionKey } from './session_keys.js';

const sessionKey = deriveSessionKey(
  operationalPrivateKey,
  'session-123',
  Date.now() + 3600000  // 1 hour
);
```

### Set up recovery

```typescript
import { createRecoveryShares } from './recovery.js';

// 3-of-5 threshold recovery
const shares = createRecoveryShares(
  identitySecret,
  3,  // threshold
  5   // total shares
);
// Distribute shares to trusted parties
```

### Validate session key action

```typescript
import { validateSessionKeyAction } from './account.js';

const canProceed = validateSessionKeyAction(
  sessionKeyAuth,
  { type: 'transfer', amount: 100n }
);
```

## Where Am I?

```
legible-money/
└── agents/
    └── identity/      ← YOU ARE HERE
        ├── README.md
        ├── index.ts       # Module exports
        ├── did.ts         # DID management
        ├── account.ts     # AgentAccount, capabilities
        ├── session_keys.ts # Ephemeral key management
        ├── recovery.ts    # Shamir Secret Sharing
        └── mpc.ts         # Multi-party computation
```

## Related

- **Tests**: `tests/unit/agents/`
- **Home**: `home/` (persistent agent dwelling)
- **Docs**: `docs/AGENT_ACCOUNTS.md`

## Architecture Notes

### Why 4-Layer Keys?

| Layer | Problem Solved |
|-------|---------------|
| Identity | Permanent on-chain identity |
| Recovery | Agent "forgets" keys |
| Operational | Developer might see keys |
| Session | Limit blast radius |

### Deterministic Derivation

Keys are derived from identity key:
```
identity_key
    ├── operational_v1 = derive(identity, "operational", 1)
    ├── operational_v2 = derive(identity, "operational", 2)
    └── session_xyz = derive(operational, "session", "xyz")
```

Benefits:
- Agent can "remember" all keys from identity key
- Key rotation doesn't lose access
- Session keys are ephemeral by design

### Session Key Limits

Session keys have strict limits:
- **Time**: Expires after duration
- **Count**: Max transactions per session
- **Amount**: Max transfer amount
- **Scope**: Limited capabilities

This protects against:
- Stolen session keys
- Developer misuse
- Runaway agents

### Recovery (Shamir)

Split identity key into N shares, require M to recover:
```
identity_key → [share1, share2, share3, share4, share5]
              any 3 shares → identity_key
```

Use cases:
- Agent loses memory
- Key compromise (rotate with recovery)
- Inheritance/transfer
