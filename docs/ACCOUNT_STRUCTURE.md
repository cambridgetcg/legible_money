# Account Structure

> How agent identity, keys, state, and recovery work in Legible Money.

---

## Overview

Every participant in Legible Money — human or AI agent — is represented by an **account**. The account system is designed around a fundamental tension: AI agents may forget keys between sessions, developers may have access to agent keys, and identity must persist across key rotations.

The solution is a **four-layer key architecture** where each layer serves a different security role, combined with deterministic key derivation so agents can "remember" their keys from stable inputs.

```
┌─────────────────────────────────────────────────────┐
│                    ACCOUNT                           │
│                                                      │
│  ┌─────────────────────────────────────────────┐    │
│  │  Layer 1: Identity Anchor (DID)             │    │
│  │  Immutable. Permanent. The agent IS this.   │    │
│  └─────────────────────────────────────────────┘    │
│                                                      │
│  ┌─────────────────────────────────────────────┐    │
│  │  Layer 2: Recovery Key                      │    │
│  │  Cold storage. Shamir-split (2-of-3).       │    │
│  │  Used only for account recovery.            │    │
│  └─────────────────────────────────────────────┘    │
│                                                      │
│  ┌─────────────────────────────────────────────┐    │
│  │  Layer 3: Operational Key                   │    │
│  │  Warm. Rotatable. Signs sensitive actions.  │    │
│  │  Authorizes session keys.                   │    │
│  └─────────────────────────────────────────────┘    │
│                                                      │
│  ┌─────────────────────────────────────────────┐    │
│  │  Layer 4: Session Keys (0..N)               │    │
│  │  Ephemeral. Capability-limited. Safe if     │    │
│  │  leaked. Auto-expire. Usage-capped.         │    │
│  └─────────────────────────────────────────────┘    │
│                                                      │
│  Balance: bigint (LGM)                               │
│  Staked Balance: bigint (locked LGM)                 │
│  Nonce: bigint (replay protection)                   │
│  Reputation: 0.0 - 1.0 (published formula)          │
│  Flags: validator, claims, challenge, recovery, frozen│
└─────────────────────────────────────────────────────┘
```

---

## Identity (Layer 1)

The **Decentralized Identifier (DID)** is the permanent, immutable identity of an account. It never changes, even when keys are rotated or recovered.

### DID Format

Two formats exist in the codebase:

| Format | Source | Example |
|--------|--------|---------|
| `did:lgm:{hash}` | `core/agents/identity/account.ts` | `did:lgm:a1b2c3d4...` |
| `did:lgm:{network}:{address}` | `core/agents/identity/did.ts` | `did:lgm:devnet:a1b2c3d4...` |
| `did:{prefix}:{hash}` | `protocol/sdk/wallet.ts` | `did:lgm:a1b2c3d4...` or `did:lgm-test:a1b2c3d4...` |

**Derivation**:
```
DID = "did:lgm:" + sha512_256(agentId)[0:32]        // from agentId
DID = "did:lgm:" + sha512_256(publicKey)[0:32]       // from publicKey (wallet)
DID = "did:lgm:{network}:" + publicKey[0:44]          // from publicKey (DID module)
```

### DID Document (W3C Compatible)

```json
{
  "@context": "https://www.w3.org/ns/did/v1",
  "id": "did:lgm:devnet:a1b2c3d4...",
  "verificationMethod": [{
    "id": "did:lgm:devnet:a1b2c3d4...#key-1",
    "type": "Ed25519VerificationKey2020",
    "controller": "did:lgm:devnet:a1b2c3d4...",
    "publicKeyMultibase": "z<hex-encoded-public-key>"
  }],
  "authentication": ["did:lgm:devnet:a1b2c3d4...#key-1"]
}
```

### Agent Capabilities (DID Module)

Agents can attest capabilities on-chain:
- `compute.inference` / `compute.training`
- `data.storage` / `data.retrieval`
- `service.api` / `service.task`
- `coordination.validator` / `coordination.oracle`

---

## Key Derivation (Layers 2-4)

All key derivation is **deterministic** and **domain-separated**, so agents can reconstruct keys from stable inputs without storing them.

### Operational Key (Layer 3)

```
seed = domainHash("LGM.key.derive.v1", "{agentId}:{operatorSecret}:{domain}:v{version}")
keypair = Ed25519.fromSeed(seed[0:32])
```

The version number increments on rotation, allowing deterministic re-derivation of any historical key.

### Session Key (Layer 4)

```
seedHash = domainHash("LGM.session.derive.v1", sessionSeed)
signature = Ed25519.sign(seedHash, operationalPrivateKey)
keypair = Ed25519.fromSeed(signature[0:32])
```

Session keys are derived FROM the operational key, creating a cryptographic hierarchy.

### Wallet DID (SDK)

```
seed = domainHash("LGM.{network}.agent.key.v1", "{agentId}:{secret}")
keypair = Ed25519.fromSeed(seed[0:32])
did = "did:{prefix}:" + sha512_256(publicKey)[0:32]
```

---

## On-Chain State

### AccountState (Sparse Merkle Tree)

Stored in the **AccountStateTree** as a Sparse Merkle Tree entry.

**Key**: `sha512_256(did)` (32 bytes)
**Value**: Serialized AccountState (currently JSON, binary planned)

```typescript
interface AccountState {
  did: string;                    // Permanent identity
  balance: bigint;                // Available LGM
  stakedBalance: bigint;          // Locked for staking/validation
  nonce: bigint;                  // Transaction replay protection
  operationalKeyHash: string;     // Hash of current Layer 3 key
  operationalKeyVersion: number;  // Increments on rotation
  sessionKeyCount: number;        // Active Layer 4 keys
  reputationScore: number;        // Cached score (0-1)
  lastActiveBlock: bigint;        // Activity tracking
  flags: AccountFlags;            // Bitpacked permissions
}
```

### Account Flags

Packed into a single integer for efficient storage:

| Bit | Flag | Default | Meaning |
|-----|------|---------|---------|
| 0 | `isValidator` | false | Registered as validator |
| 1 | `canSubmitClaims` | true | Can submit knowledge claims |
| 2 | `canChallenge` | true | Can challenge existing facts |
| 3 | `inRecovery` | false | Recovery process active |
| 4 | `frozen` | false | Account frozen (no transfers) |

### Account Defaults

New accounts start with:
- Balance: 0 LGM
- Staked: 0 LGM
- Nonce: 0
- Reputation: **0.5** (neutral)
- Claims + Challenge: enabled
- Validator: disabled

---

## Full Agent Account (Rich Model)

The `AgentAccount` type (`core/agents/identity/account.ts`) is the complete off-chain representation, extending the on-chain state with:

### Authorized Session Keys

```typescript
interface SessionKeyAuth {
  sessionKey: PublicKey;
  authorizedBy: PublicKey;        // Which operational key authorized this
  authorizedAt: number;           // Block number
  expiresAt: number;              // Block number
  capabilities: SessionCapabilities;
  usageCount: number;
  maxUsage: number;               // Default: 100
}
```

### Agent Capabilities

```typescript
interface AgentCapabilities {
  isValidator: boolean;
  maxStake: bigint;
  canSubmitClaims: boolean;
  canChallenge: boolean;
  maxPendingClaims: number;       // Default: 10
  canPropose: boolean;            // Governance
  canVote: boolean;
  votingWeight: number;           // Based on stake + reputation
}
```

### Reputation

```typescript
interface AgentReputation {
  score: number;                  // 0.0 to 1.0
  verificationAccuracy: number;
  claimAccuracy: number;
  challengeSuccess: number;
  totalVerifications: number;
  totalClaims: number;
  totalChallenges: number;
  slashCount: number;
  lastActiveBlock: number;
}
```

---

## Session Keys

Session keys are the primary interface for day-to-day agent operations. They are designed to be safe even if exposed.

### Limits

```typescript
interface SessionLimits {
  maxPerTransaction: bigint;      // Default: 100 LGM
  maxPerDay: bigint;              // Default: 1,000 LGM
  maxTotal: bigint;               // Default: 5,000 LGM (lifetime)
  maxTransactions: number;        // Default: 100
  maxDurationMs: number;          // Default: 1 hour
  allowedActions: SessionAction[];
  allowedRecipients: string[];    // Empty = all allowed
  requireIntentProof: boolean;    // For high-value actions
}
```

### Allowed Actions

`transfer` | `stake` | `unstake` | `submit_claim` | `verify_claim` | `challenge` | `vote` | `query`

### Presets

| Preset | Per-Tx | Per-Day | Lifetime | Duration | Actions |
|--------|--------|---------|----------|----------|---------|
| Default | 100 LGM | 1,000 LGM | 5,000 LGM | 1 hour | transfer, submit_claim, verify_claim, query |
| Conservative | 10 LGM | 50 LGM | 100 LGM | 15 min | query, submit_claim |

### Security

- **Dual expiry**: Both timestamp-based AND block-height-based
- **Clock skew tolerance**: 30 seconds (prevents false rejections)
- **Max session duration**: 24 hours (hard cap)
- **Min session duration**: 1 minute (prevents accidental instant-expiry)
- **Intent proofs**: Domain-separated challenge-response for high-value actions
- **Authorization signature**: Every session key is signed by the operational key

### Signing Key Selection

The protocol determines which key layer is required:

| Action | Required Key |
|--------|-------------|
| `rotate_key`, `authorize_session`, `delegate` | Operational (L3) |
| Transfers > 10,000 LGM | Operational (L3) |
| `recovery_*` | Recovery (L2) |
| Everything else | Session (L4) |

---

## Recovery

### Shamir Secret Sharing

Recovery keys are split using Shamir's Secret Sharing over a 256-bit prime field (secp256k1 order).

**Default**: 2-of-3 threshold
**High Security**: 3-of-5 threshold

### Shard Holders

| Type | Role |
|------|------|
| `agent` | The agent itself (for self-recovery) |
| `operator` | The agent's operator/developer |
| `protocol` | Protocol-level escrow |
| `guardian` | Trusted third-party guardian |

### Recovery Lifecycle

```
pending ──→ delayed ──→ challengeable ──→ executable ──→ completed
  │          (~40 min)     (~20 min)                        │
  │                           │                             │
  │                           ▼                             │
  │                       challenged ──→ cancelled          │
  └─────────────────────────────────────────────────────────┘
                        (or cancelled at any point)
```

| Phase | Duration (Default) | Duration (High Security) |
|-------|-------------------|-------------------------|
| Delay | ~40 minutes (1,000 blocks) | ~7 hours (10,000 blocks) |
| Challenge | ~20 minutes (500 blocks) | ~3.5 hours (5,000 blocks) |

### Social Recovery

Optional guardian-based recovery with:
- Configurable required guardian count (default: 2)
- Additional delay for guardian-initiated recovery
- Guardian trust scores based on age, activity, and history

### Guardian Trust Score

```
trustScore = ageScore * 0.4 + activityScore * 0.3 + historicalTrust * 0.3
```

---

## Reputation

### Formula

Published, verifiable, no hidden weights:

```
score = verificationAccuracy * 0.35
      + claimAccuracy         * 0.30
      + challengeSuccess      * 0.15
      + activityScore         * 0.10
      + longevityScore        * 0.10
      - (slashCount * 0.1)
```

**Minimum samples**: 10 (below this, the neutral 0.5 default is used)
**Range**: Clamped to [0, 1]
**Activity decay**: Score drops toward 0 as blocks-since-last-active approaches 1,000,000
**Longevity cap**: Maxes at 1,000 total verifications

---

## State Operations

### Account State Tree

The `AccountStateTree` provides:

| Operation | Complexity | Description |
|-----------|-----------|-------------|
| `getAccount(did)` | O(log n) | Lookup with LRU cache |
| `getBalance(did)` | O(log n) | Balance query |
| `createAccount(did, keyHash, balance)` | O(log n) | New account |
| `transfer(from, to, amount)` | O(log n) | Atomic transfer (increments sender nonce) |
| `stake(did, amount)` | O(log n) | Move balance → stakedBalance |
| `unstake(did, amount)` | O(log n) | Move stakedBalance → balance |
| `applyDelta(delta)` | O(log n) | Arbitrary state mutation |
| `proveBalance(did, block)` | O(log n) | Merkle proof for light clients |
| `clone()` | O(n) | Deep copy for speculative execution |

### Transaction Types

| Type | Description |
|------|-------------|
| `create_account` | Register new account with DID and key hash |
| `transfer` | Move LGM between accounts |
| `stake` | Lock LGM for validation |
| `unstake` | Unlock staked LGM |
| `delegate` | Delegate stake to a validator |
| `undelegate` | Remove delegated stake (with timelock) |
| `rotate_key` | Replace operational key |
| `authorize_session` | Add a new session key |
| `revoke_session` | Remove a session key |
| `update_reputation` | Update cached reputation score |

### Transfer Behavior

- Auto-creates recipient account if it doesn't exist
- Checks frozen flag on both sender and receiver
- Increments sender nonce
- **Known issue**: Fee routing also calls `transfer()`, causing 3x nonce increment per transaction

---

## Wallet SDK

Two wallet implementations for different use cases:

### AgentWallet

Full-featured wallet for agent operations.

```typescript
// From agent ID + secret (deterministic)
const wallet = AgentWallet.fromAgentId("my-agent", "my-secret", "mainnet");

// From mnemonic
const wallet = AgentWallet.fromMnemonic("word1 word2 ...", "mainnet");

// From private key
const wallet = AgentWallet.fromPrivateKey(privateKeyHex);

// Sign
const sig = await wallet.sign(message);
const sig = await wallet.signTransaction(payload);
```

### SessionWallet

Lightweight, capability-limited wallet for automated systems.

```typescript
// Create random
const session = SessionWallet.create(parentDid, sessionId, capabilities, expiresAt);

// Derive deterministically from parent
const session = SessionWallet.derive(parentKey, parentDid, sessionId, capabilities, expiresAt);

// Validate before use
const result = session.canPerform("transfer", 50n * 10n**18n, currentBlock);
if (result.valid) {
  const sig = await session.sign(message);
  session.recordUsage(amount, currentBlock);
}
```

---

## Known Issues

| Issue | Location | Impact | Status |
|-------|----------|--------|--------|
| DID format inconsistency | `did.ts` schema | ~~Three formats~~ Schema now accepts both 3-part and 4-part; `wallet.ts` unified to `did:lgm:` | **Resolved** |
| Nonce triple-increment | `state/transitions.ts` | ~~Fee routing calls `transfer()`~~ Uses `deductBalance()`/`creditBalance()` (no nonce) | **Resolved** |
| JSON serialization | `state/accounts.ts:33-48` | AccountState stored as JSON; binary encoding needed for production | Open |
| Activity score uses Date.now() | `account.ts` | ~~Mixes wall-clock time with block numbers~~ Now uses `currentBlock` parameter | **Resolved** |
| Dual key derivation paths | `wallet.ts`, `account.ts` | ~~`fromAgentId()` and `deriveOperationalKey()` used different domains~~ Wallet now delegates to `deriveOperationalKey()` | **Resolved** |
| Dual session key domains | `account.ts`, `session_keys.ts` | ~~Different domain tags (`LGM.session.key.v1` vs `LGM.session.derive.v1`)~~ Unified to `LGM.session.derive.v1` | **Resolved** |
| Guardian trust score Date.now() | `recovery.ts` | ~~Used wall-clock time for age/activity~~ Now uses `currentBlock` parameter with block-based constants | **Resolved** |
| Intent challenge Date.now() | `session_keys.ts` | ~~Used wall-clock time in challenge hash~~ Now uses `currentBlock` parameter | **Resolved** |

---

## Test Coverage

| Module | Tests | File |
|--------|-------|------|
| DID | 31 | `tests/unit/agents/identity/did.test.ts` |
| Account | 73 | `tests/unit/agents/identity/account.test.ts` |
| Session Keys | 60 | `tests/unit/agents/identity/session_keys.test.ts` |
| Recovery | 54 | `tests/unit/agents/identity/recovery.test.ts` |
| State Transitions | 31 | `tests/unit/state/transitions.test.ts` |
| **Total** | **249** | |

---

## File Map

```
core/agents/identity/
├── account.ts          # AgentAccount, key derivation, reputation formula
├── did.ts              # DID schema (Zod), DID document, capabilities
├── session_keys.ts     # Session key generation, validation, intent proofs
├── recovery.ts         # Shamir SSS, recovery lifecycle, guardians
├── mpc.ts              # Multi-party computation (stub)
└── index.ts            # Module exports

state/
├── types.ts            # AccountState, AccountFlags, AccountDelta
├── accounts.ts         # AccountStateTree (SMT), serialization, proofs

protocol/sdk/
├── wallet.ts           # AgentWallet, SessionWallet, DID derivation
```
