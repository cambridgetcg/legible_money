# Genesis Core

> **The irreducible cryptographic foundation of Legible Money.**

```
Five files. Zero custom cryptography. Absolute soundness.
```

---

## What Is Genesis Core?

Genesis Core is the **minimum set of components** required to:

1. **Generate identities** - Create cryptographic keypairs for agents
2. **Prove authenticity** - Sign messages that cannot be forged
3. **Ensure integrity** - Hash data so tampering is detectable
4. **Establish identity** - Give agents decentralized identifiers
5. **Define parameters** - Set protocol constants all nodes agree on

Everything else in Legible Money - consensus, state, governance, applications - builds on these five files.

---

## The Five Components

```
genesis/
├── GENESIS_CORE.md          # This document
├── PURPOSE.md               # Philosophy and verification criteria
├── index.ts                 # Re-exports all Genesis Core
│
└── Source Files:
    ├── core/crypto/keys.ts       # Key generation
    ├── core/crypto/hashing.ts    # Hash functions
    ├── core/crypto/signatures.ts # Digital signatures
    ├── core/constants.ts         # Protocol parameters
    └── agents/identity/did.ts    # Decentralized identifiers
```

---

## 1. Keys

**File**: `core/crypto/keys.ts`
**Purpose**: Generate and manage cryptographic identities
**External dependency**: `@noble/ed25519`

### What It Does

```typescript
// Generate a new identity
const { privateKey, publicKey } = generateKeypair();

// privateKey: 64 hex characters (32 bytes)
// Example: "a1b2c3d4e5f6..."

// publicKey: 64 hex characters (32 bytes)
// Example: "f7e8d9c0b1a2..."

// Derive address from public key
const address = publicKeyToAddress(publicKey);
// Example: "lgm1a1b2c3d4e5f6789..."
```

### How It Works

```
┌─────────────────────────────────────────────────────────┐
│                    KEY GENERATION                        │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   OS Random Number Generator                             │
│            │                                             │
│            ▼                                             │
│   ┌─────────────────┐                                   │
│   │  32 random bytes │  ← Private Key                   │
│   └────────┬────────┘                                   │
│            │                                             │
│            ▼  Ed25519 scalar multiplication              │
│   ┌─────────────────┐                                   │
│   │  32 bytes        │  ← Public Key                    │
│   └────────┬────────┘                                   │
│            │                                             │
│            ▼  SHA-512, take first 20 bytes              │
│   ┌─────────────────┐                                   │
│   │  "lgm1" + 40 hex │  ← Address                       │
│   └─────────────────┘                                   │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### Key Functions

| Function | Input | Output | Purpose |
|----------|-------|--------|---------|
| `generateKeypair()` | none | `{privateKey, publicKey}` | Create new identity |
| `generateKeypairFromSeed(seed)` | 32 bytes | `{privateKey, publicKey}` | Deterministic generation |
| `getPublicKey(privateKey)` | private key | public key | Derive public from private |
| `publicKeyToAddress(publicKey)` | public key | address | Create short address |
| `bytesToHex(bytes)` | Uint8Array | hex string | Encode bytes |
| `hexToBytes(hex)` | hex string | Uint8Array | Decode hex |

### Security Properties

- **Randomness**: Uses OS cryptographically secure random number generator
- **One-way**: Cannot derive private key from public key (discrete log problem)
- **Deterministic**: Same seed always produces same keypair
- **Collision-resistant**: Practically impossible to generate same keypair twice

---

## 2. Hashing

**File**: `core/crypto/hashing.ts`
**Purpose**: Create unforgeable fingerprints of data
**External dependency**: `@noble/hashes`

### What It Does

```typescript
// Basic hash
const hash = sha512Hash("hello world");
// 64 bytes, unique fingerprint of input

// Domain-separated hash (prevents cross-protocol attacks)
const txHash = domainHash(HASH_DOMAINS.TRANSACTION, data);
// Different domain = different hash, even for same data

// Merkle tree hash
const node = merkleInternalHash(left, right);
// For building verifiable data structures
```

### How It Works

```
┌─────────────────────────────────────────────────────────┐
│                    DOMAIN HASHING                        │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   Domain: "LGM.tx.v1"    Data: <transaction>            │
│            │                      │                      │
│            ▼                      ▼                      │
│   ┌──────────────────────────────────────┐              │
│   │  "LGM.tx.v1" + [length] + <data>     │              │
│   └──────────────────┬───────────────────┘              │
│                      │                                   │
│                      ▼  SHA-512                          │
│   ┌──────────────────────────────────────┐              │
│   │  64-byte hash (unique fingerprint)    │              │
│   └──────────────────────────────────────┘              │
│                                                          │
│   Same data + different domain = different hash          │
│   Prevents transaction hash being used as block hash     │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### Hash Algorithms

| Algorithm | Output | Speed | Use Case |
|-----------|--------|-------|----------|
| SHA-512 | 64 bytes | Medium | Signatures, commitments |
| SHA-256 | 32 bytes | Medium | Compatibility |
| BLAKE3 | 32 bytes | Fast | Merkle trees, state |

### Domain Constants

```typescript
HASH_DOMAINS = {
  BLOCK: 'LGM.block.v1',
  TRANSACTION: 'LGM.tx.v1',
  STATE: 'LGM.state.v1',
  FACT: 'LGM.fact.v1',
  COMMITMENT: 'LGM.commitment.v1',
  VRF: 'LGM.vrf.v1',
  SIGNATURE: 'LGM.sig.v1',
}
```

### Key Functions

| Function | Purpose |
|----------|---------|
| `sha512Hash(data)` | Primary hash function |
| `blake3Hash(data)` | Fast hash for trees |
| `domainHash(domain, data)` | Domain-separated hash |
| `merkleHash(left, right)` | Combine tree nodes |
| `merkleLeafHash(data)` | Hash leaf node (prefix 0x00) |
| `merkleInternalHash(left, right)` | Hash internal node (prefix 0x01) |
| `hashClaim(claim)` | Normalize and hash a claim |
| `constantTimeEqual(a, b)` | Compare without timing leaks |

### Security Properties

- **Collision resistance**: Cannot find two inputs with same hash
- **Preimage resistance**: Cannot reverse hash to find input
- **Avalanche effect**: Small input change = completely different hash
- **Domain separation**: Same data in different contexts = different hash

---

## 3. Signatures

**File**: `core/crypto/signatures.ts`
**Purpose**: Prove message authenticity
**External dependency**: `@noble/curves`

### What It Does

```typescript
// Sign a message
const signature = sign(messageBytes, privateKey);
// 64-byte signature, proves you have the private key

// Verify a signature
const isValid = verify(messageBytes, signature, publicKey);
// true if signature matches message and public key

// Sign with domain separation
const txSig = signTransaction(transaction, privateKey);
// Signs hash of transaction, not raw transaction
```

### How It Works

```
┌─────────────────────────────────────────────────────────┐
│                      SIGNING                             │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   Message              Private Key                       │
│      │                      │                            │
│      └──────────┬───────────┘                            │
│                 │                                        │
│                 ▼  Ed25519.sign()                        │
│   ┌─────────────────────────────────┐                   │
│   │  64-byte Signature               │                   │
│   │  (unique to message + key)       │                   │
│   └─────────────────────────────────┘                   │
│                                                          │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│                    VERIFICATION                          │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   Message    Signature    Public Key                     │
│      │           │            │                          │
│      └───────────┼────────────┘                          │
│                  │                                       │
│                  ▼  Ed25519.verify()                     │
│   ┌─────────────────────────────────┐                   │
│   │  true / false                    │                   │
│   │  (mathematically verified)       │                   │
│   └─────────────────────────────────┘                   │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### Key Functions

| Function | Purpose |
|----------|---------|
| `sign(message, privateKey)` | Create signature |
| `verify(message, signature, publicKey)` | Verify signature |
| `signTransaction(tx, privateKey)` | Sign transaction with domain separation |
| `verifyTransaction(tx, sig, publicKey)` | Verify transaction signature |
| `signBlock(header, privateKey)` | Sign block header |
| `verifyBlock(header, sig, publicKey)` | Verify block signature |
| `batchVerify(items)` | Verify multiple signatures efficiently |

### Security Properties

- **Unforgeability**: Cannot create valid signature without private key
- **Non-repudiation**: Signer cannot deny signing
- **Integrity**: Any change to message invalidates signature
- **Deterministic**: Same message + key = same signature

---

## 4. Constants

**File**: `core/constants.ts`
**Purpose**: Define protocol identity and parameters
**External dependency**: None

### What It Defines

```typescript
// Token identity
TOKEN = {
  NAME: 'Legible Money',
  SYMBOL: 'LGM',
  DECIMALS: 18,
  MAX_SUPPLY: 1_000_000_000n * 10n ** 18n  // 1 billion
}

// Timing
TIMING = {
  TARGET_BLOCK_MS: 2521,       // ~2.5 second blocks
  CONFIRMATION_BLOCKS: 19,      // Safety confirmations
  FINALITY_BLOCKS: 100          // Full finality
}

// Consensus thresholds
CONSENSUS = {
  MIN_VALIDATORS: 5,
  QUORUM_THRESHOLD: 0.66,       // 66% for quorum
  ACCEPTANCE_THRESHOLD: 0.80,   // 80% to accept
  REJECTION_THRESHOLD: 0.30     // 30% to reject
}

// Economics
ECONOMICS = {
  KNOWLEDGE_POOL_SHARE: 0.30,   // 30% to knowledge pool
  API_PROVIDER_SHARE: 0.60,     // 60% to API providers
  PROTOCOL_SHARE: 0.10,         // 10% to protocol
  BURN_RATE: 0.50               // 50% of fees burned
}
```

### Why Constants Matter

- **Coordination**: All nodes must agree on these values
- **Identity**: Changing them changes the protocol's identity
- **Immutability**: These values are frozen forever

---

## 5. DID (Decentralized Identifier)

**File**: `agents/identity/did.ts`
**Purpose**: Give agents self-sovereign identities
**External dependency**: `zod` (validation)

### What It Does

```typescript
// Create DID from public key
const did = createDID(publicKey, 'mainnet');
// "did:lgm:mainnet:a1b2c3d4e5f6..."

// Parse DID
const address = didToAddress(did);   // "a1b2c3d4e5f6..."
const network = didToNetwork(did);   // "mainnet"

// Full identity
const identity: AgentIdentity = {
  did: did,
  publicKey: publicKey,
  capabilities: ['compute.inference', 'service.api'],
  metadata: {
    name: 'My Agent',
    endpoint: 'https://api.example.com'
  }
};
```

### DID Format

```
did:lgm:mainnet:a1b2c3d4e5f6789...
│   │     │            │
│   │     │            └── Address (44 hex chars)
│   │     └── Network (mainnet/testnet/devnet)
│   └── Method (lgm = Legible Money)
└── Scheme (always "did")
```

### Agent Capabilities

```typescript
Capability =
  | 'compute.inference'       // Can run AI inference
  | 'compute.training'        // Can train models
  | 'data.storage'            // Can store data
  | 'data.retrieval'          // Can search/retrieve
  | 'service.api'             // Can provide APIs
  | 'service.task'            // Can complete tasks
  | 'coordination.validator'  // Can validate
  | 'coordination.oracle'     // Can provide oracle data
```

### DID Document (W3C Standard)

```typescript
const doc = createDIDDocument(identity);
// {
//   "@context": "https://www.w3.org/ns/did/v1",
//   "id": "did:lgm:mainnet:a1b2c3...",
//   "verificationMethod": [{
//     "id": "did:lgm:mainnet:a1b2c3...#key-1",
//     "type": "Ed25519VerificationKey2020",
//     "controller": "did:lgm:mainnet:a1b2c3...",
//     "publicKeyMultibase": "za1b2c3..."
//   }],
//   "authentication": ["did:lgm:mainnet:a1b2c3...#key-1"]
// }
```

---

## How They Work Together

### Complete Flow: Agent Signs a Fact

```typescript
import {
  generateKeypair,
  createDID,
  hashFact,
  signTransaction,
  verifyTransaction
} from './genesis/index.js';

// STEP 1: Generate identity (keys.ts)
const { privateKey, publicKey } = generateKeypair();

// STEP 2: Create DID (did.ts)
const did = createDID(publicKey, 'devnet');

// STEP 3: Create fact
const fact = {
  claim: "Water boils at 100°C at sea level",
  type: "empirical",
  domain: "physics",
  submittedBy: did,
  references: ["https://example.com/source"]
};

// STEP 4: Hash fact (hashing.ts)
const factHash = hashFact(fact);

// STEP 5: Create transaction
const tx = {
  type: 'submit_fact',
  factHash: factHash,
  submitter: did,
  fee: 1000000000000000n,  // 0.001 LGM
  nonce: 1,
  timestamp: Date.now()
};

// STEP 6: Sign transaction (signatures.ts)
const signature = signTransaction(tx, privateKey);

// STEP 7: Anyone can verify
const isValid = verifyTransaction(tx, signature, publicKey);
console.log(isValid); // true

// STEP 8: Broadcast to network...
```

### Trust Chain

```
┌─────────────────────────────────────────────────────────┐
│                     TRUST CHAIN                          │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   OS Entropy (true randomness)                          │
│          │                                               │
│          ▼                                               │
│   Private Key (32 bytes, secret)                        │
│          │                                               │
│          ├────────────────────────┐                     │
│          │                        │                      │
│          ▼                        ▼                      │
│   Public Key              Signatures                     │
│   (32 bytes, shared)      (proves ownership)            │
│          │                        │                      │
│          ▼                        │                      │
│   DID                             │                      │
│   (decentralized ID)              │                      │
│          │                        │                      │
│          └────────────┬───────────┘                     │
│                       │                                  │
│                       ▼                                  │
│               AGENT IDENTITY                             │
│          (cryptographically bound)                       │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## Why Genesis Core Is Sound

### External Dependencies

| Library | Purpose | Security |
|---------|---------|----------|
| `@noble/ed25519` | Key generation | Audited, no deps |
| `@noble/hashes` | Hash functions | Audited, no deps |
| `@noble/curves` | Signatures | Formally verified |
| `zod` | Validation | Type-safe schemas |

### The Noble Libraries

Written by **Paul Miller**, these libraries are:

- **Audited** by multiple security firms (Trail of Bits, Cure53)
- **Zero dependencies** - no supply chain risk
- **Formally verified** - mathematical proofs of correctness
- **Widely used** - Ethereum, Solana, and major projects

### What's Custom vs Library

| Custom Code | Library Code |
|-------------|--------------|
| Hex encoding | SHA-512 algorithm |
| Domain separation | Ed25519 math |
| DID formatting | BLAKE3 algorithm |
| Type definitions | Key generation |

**All cryptographic math is in audited libraries.**

Custom code is only:
- String manipulation (hex encoding)
- Data formatting (domain prefixes)
- Type definitions (TypeScript interfaces)

---

## Verification Status

| File | Tests | Vectors | Docs | Audit |
|------|-------|---------|------|-------|
| `keys.ts` | ⬜ Pending | ⬜ RFC 8032 | ⬜ | ⬜ |
| `hashing.ts` | ⬜ Pending | ⬜ NIST/BLAKE3 | ⬜ | ⬜ |
| `signatures.ts` | ⬜ Pending | ⬜ RFC 8032 | ⬜ | ⬜ |
| `constants.ts` | ⬜ Pending | N/A | ⬜ | ⬜ |
| `did.ts` | ⬜ Pending | N/A | ⬜ | ⬜ |

### Verification Criteria

Before Genesis Core is declared complete:

```
□ 100% test coverage for all files
□ Test vectors from standards (NIST, RFC 8032, BLAKE3)
□ Security properties documented
□ Edge cases tested
□ External audit completed
□ Hash anchors generated
```

---

## Using Genesis Core

### Import Everything

```typescript
import * as genesis from './genesis/index.js';

const keys = genesis.generateKeypair();
const hash = genesis.sha512Hash(data);
const sig = genesis.sign(message, keys.privateKey);
```

### Import Specific Functions

```typescript
import {
  generateKeypair,
  sign,
  verify,
  domainHash,
  HASH_DOMAINS,
  createDID
} from './genesis/index.js';
```

### Check Verification Status

```typescript
import {
  isGenesisVerified,
  printVerificationStatus
} from './genesis/index.js';

if (!isGenesisVerified()) {
  console.warn('Genesis Core not yet fully verified');
}

printVerificationStatus();
// === GENESIS CORE VERIFICATION STATUS ===
// Overall: 10% (2/20)
// ⬜ core/crypto/hashing.ts - Missing: tests, vectors, docs, audit
// ...
```

---

## Immutability Commitment

Genesis Core is **immutable after verification**.

Once verified:
- Hash anchors are computed for each file
- Anchors are included in the genesis block
- Any change invalidates the anchors
- Nodes can verify Genesis Core hasn't been tampered with

**This is a commitment**: Genesis Core will never change.

---

## Summary

Genesis Core is:

1. **Five files** - keys, hashing, signatures, constants, DID
2. **Zero custom crypto** - all math in audited noble libraries
3. **Pure functions** - deterministic, no side effects
4. **The foundation** - everything else builds on this

```
┌────────────────────────────────────────┐
│           GENESIS CORE                  │
│                                         │
│   Keys → Hashing → Signatures → DID    │
│              ↑                          │
│          Constants                      │
│                                         │
│   5 files. 0 custom crypto. Sound.     │
└────────────────────────────────────────┘
```

Everything in Legible Money - consensus, state, governance, agents - builds on these five files. If Genesis Core is sound, the foundation is solid.
