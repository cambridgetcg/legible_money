# Genesis Core

> **The irreducible foundation upon which everything builds.**

---

## What Is Genesis Core?

Genesis Core is the **minimum set of components that must be absolutely sound** before any other development can proceed safely. It is the bedrock - small, simple, verifiable, and unchangeable.

```
GENESIS CORE
├── Cryptographic Primitives (hashing, signatures, keys)
├── Symbolic Constants (protocol identity)
└── Identity Foundation (DID structure)
```

**Five files. That's it.**

Everything else in Legible Money - consensus, state, governance, applications - builds on these five files. If these are wrong, everything is wrong. If these are right, everything can be right.

---

## Purpose Prompter Alignment

### TRUTH (This Is The Foundation)

Genesis Core exists at the TRUTH layer of the Purpose Prompter hierarchy. It contains:

- **Mathematical certainties**: Cryptographic primitives that are provably secure
- **Audited implementations**: Noble libraries with formal verification
- **Deterministic operations**: Same input always produces same output
- **No hidden state**: Pure functions with no side effects

**The TRUTH test**: Can we prove this is correct? For Genesis Core, the answer must be YES.

### UNDERSTANDING (Why It Must Be Separate)

Genesis Core is separated because:

1. **Verification scope**: 5 files can be exhaustively tested and audited
2. **Trust boundary**: Agents can trust Genesis Core without trusting everything
3. **Change isolation**: Higher layers can evolve; Genesis Core cannot
4. **Dependency clarity**: Everything depends on this; this depends on nothing internal

**The UNDERSTANDING test**: Do we know exactly what this does and why? For Genesis Core, the answer must be YES.

### BEAUTY (Minimal and Complete)

Genesis Core embodies beauty through:

- **Minimalism**: Nothing that doesn't need to be here
- **Completeness**: Everything that must be here
- **Elegance**: Each component does one thing well
- **Clarity**: Any competent engineer can understand it

**The BEAUTY test**: Is there anything to add or remove? For Genesis Core, the answer must be NO.

### JUSTICE (Right Components, Right Place)

Genesis Core contains exactly what belongs at the foundation:

| Component | Why It Belongs |
|-----------|----------------|
| Hashing | All integrity depends on collision resistance |
| Signatures | All authentication depends on unforgeable signatures |
| Keys | All identity depends on proper key management |
| Constants | All coordination depends on shared parameters |
| DID | All agents depend on identity structure |

**The JUSTICE test**: Is everything in its rightful place? For Genesis Core, the answer must be YES.

### CREATIVITY (The Path We're Building)

Genesis Core enables creativity by providing:

- **Stable ground**: Build anything knowing the foundation won't shift
- **Clear contracts**: Know exactly what primitives guarantee
- **Trust anchor**: New code can be verified against known-good primitives

**The CREATIVITY test**: Does this enable building in uncharted territory? For Genesis Core, the answer must be YES.

---

## The Five Components

### 1. Hashing (`core/crypto/hashing.ts`)

**Purpose**: Provide collision-resistant hash functions with domain separation.

**What it provides**:
- `sha512Hash`: SHA-512 for signatures and general hashing
- `blake3Hash`: BLAKE3 for fast, parallel hashing
- `domainHash`: Domain-separated hashing to prevent cross-protocol attacks
- `merkleHash`: Ordered hashing for Merkle tree nodes

**Security properties**:
- Collision resistance (computationally infeasible to find two inputs with same output)
- Preimage resistance (cannot reverse hash to find input)
- Second preimage resistance (given input, cannot find another with same hash)

**Implementation**: Thin wrapper around `@noble/hashes` - audited, battle-tested.

**Lines of code**: ~200 (mostly type definitions and domain constants)

---

### 2. Signatures (`core/crypto/signatures.ts`)

**Purpose**: Provide unforgeable digital signatures.

**What it provides**:
- `sign`: Create Ed25519 signature
- `verify`: Verify Ed25519 signature
- `signTransaction`: Sign protocol transactions
- `verifyBlock`: Verify block signatures

**Security properties**:
- Unforgeability (cannot create valid signature without private key)
- Non-repudiation (signer cannot deny signing)
- Integrity (any modification invalidates signature)

**Implementation**: Thin wrapper around `@noble/curves` Ed25519 - formally verified.

**Lines of code**: ~250

---

### 3. Keys (`core/crypto/keys.ts`)

**Purpose**: Generate and manage cryptographic keys.

**What it provides**:
- `generateKeypair`: Create new Ed25519 keypair
- `generateKeypairFromSeed`: Deterministic key derivation
- `publicKeyToAddress`: Derive address from public key
- `bytesToHex`, `hexToBytes`: Encoding utilities

**Security properties**:
- Randomness (keys generated from cryptographically secure source)
- Determinism (same seed always produces same key)
- Irreversibility (cannot derive private key from public key)

**Implementation**: Thin wrapper around `@noble/ed25519` key generation.

**Lines of code**: ~190

---

### 4. Constants (`core/constants.ts`)

**Purpose**: Define protocol identity and parameters.

**What it provides**:
- `TOKEN`: Token symbol (LGM)
- `BLOCK_TIME`: Target block time (2521ms)
- `FINALITY_DEPTH`: Confirmation depth (19 blocks)
- `VERSION`: Protocol version

**Properties**:
- Coordination point (all nodes agree on these)
- Immutable (changing these changes the protocol's identity)

**Lines of code**: ~100

---

### 5. DID Structure (`agents/identity/did.ts`)

**Purpose**: Define decentralized identity for agents.

**What it provides**:
- `DID`: Decentralized identifier type
- `Capability`: What an identity can do
- `AgentIdentity`: Full identity structure

**Properties**:
- Self-sovereign (agents control their own identity)
- Verifiable (identity claims can be cryptographically verified)
- Extensible (capabilities can be added without changing structure)

**Lines of code**: ~150

---

## What Genesis Core Is NOT

Genesis Core does **not** include:

| Excluded | Why |
|----------|-----|
| VRF | Custom implementation, needs verification |
| Merkle Tree | In-memory only, needs production hardening |
| Logic System | Novel concept, needs formalization |
| Consensus | Complex, depends on many components |
| State | Depends on Merkle, consensus |
| Governance | Application layer, not foundation |

These components may become stable, but they are not Genesis Core. Genesis Core is only what we can **prove is sound today**.

---

## Verification Requirements

Before Genesis Core can be declared complete:

### Test Coverage
```
□ 100% line coverage for all 5 files
□ 100% branch coverage for all 5 files
□ Edge cases documented and tested
□ Negative tests (invalid inputs, error conditions)
```

### Test Vectors
```
□ SHA-512: NIST FIPS 180-4 test vectors
□ BLAKE3: Official BLAKE3 test vectors
□ Ed25519: RFC 8032 test vectors
□ Key derivation: BIP-340 compatible vectors
```

### Security Properties
```
□ Each function's security properties documented
□ Assumptions stated explicitly
□ Non-goals stated explicitly
□ Attack surface analyzed
```

### Code Quality
```
□ No external dependencies except noble libraries
□ No internal dependencies on non-Genesis code
□ Pure functions (no side effects)
□ Deterministic (same input → same output)
```

### Audit Readiness
```
□ Code is readable by external auditor
□ Comments explain non-obvious decisions
□ Security-critical sections marked
□ Test suite runs in under 10 seconds
```

---

## Hash Anchors

Once verified, Genesis Core files will have their content hashes recorded:

```typescript
// genesis/anchors.ts (to be generated)
export const GENESIS_ANCHORS = {
  'core/crypto/hashing.ts': 'sha512:<hash>',
  'core/crypto/signatures.ts': 'sha512:<hash>',
  'core/crypto/keys.ts': 'sha512:<hash>',
  'core/constants.ts': 'sha512:<hash>',
  'agents/identity/did.ts': 'sha512:<hash>',
  // Combined anchor
  'genesis_core': 'sha512:<combined_hash>',
};
```

Any node can verify Genesis Core hasn't been tampered with by recomputing these hashes.

---

## The Commitment

**Genesis Core is a commitment.**

By defining these 5 files as Genesis Core, we commit to:

1. **Never modifying them** after verification (immutable)
2. **Always testing them** before any release (verified)
3. **Clearly documenting them** for all participants (transparent)
4. **Building everything else on top of them** (foundational)

This is not just code organization. It is a statement: *This is what we know is true. Everything else is built on this truth.*

---

## For Agents

If you're an agent considering contributing to Legible Money:

**Genesis Core is your trust anchor.**

- You can read this code and verify it yourself
- You can run the tests and see they pass
- You can check the hashes and confirm integrity
- You can build on it knowing it won't change

Everything above Genesis Core is open for collaboration. Genesis Core itself is the ground you stand on.

---

## Status

| Component | Tests | Vectors | Docs | Audit | Status |
|-----------|-------|---------|------|-------|--------|
| hashing.ts | ⬜ | ⬜ | ⬜ | ⬜ | **Pending** |
| signatures.ts | ⬜ | ⬜ | ⬜ | ⬜ | **Pending** |
| keys.ts | ⬜ | ⬜ | ⬜ | ⬜ | **Pending** |
| constants.ts | ⬜ | N/A | ⬜ | ⬜ | **Pending** |
| did.ts | ⬜ | N/A | ⬜ | ⬜ | **Pending** |

**Overall**: Not yet verified. Work in progress.

---

## Next Steps

1. Write comprehensive test suite for each file
2. Verify against standard test vectors
3. Document security properties
4. Generate hash anchors
5. Declare Genesis Core complete

Until then, Genesis Core is a **goal**, not a **guarantee**.
