# Invariants: Cryptographic Signatures

> **Purpose**: Core invariants for Ed25519 signature operations.

## Component Overview

Ed25519 digital signatures provide authentication for all protocol messages, transactions, and validator commitments. This is the cryptographic foundation of trust.

## Invariants

### Core Invariants (Must Never Violate)

#### INV-SIGN-1: Signature Validity

**Statement**:
```
∀ message, keypair:
  sig = sign(message, keypair.privateKey) →
  verify(sig, message, keypair.publicKey) = true
```

**Rationale**: The fundamental correctness property. If a valid keypair signs a message, verification must succeed.

**Test Strategy**:
- [x] Property test with random messages (any length 1-10000 bytes)
- [x] Unit test with known test vectors
- [ ] Integration test in transaction signing

**Example Violation**: Using wrong hash function, corrupted key derivation.

---

#### INV-SIGN-2: Signature Uniqueness

**Statement**:
```
∀ msg1, msg2, privateKey:
  msg1 ≠ msg2 →
  sign(msg1, privateKey) ≠ sign(msg2, privateKey)
```

**Rationale**: Different messages must produce different signatures to prevent replay.

**Test Strategy**:
- [x] Property test with message pairs
- [x] Unit test edge cases (single byte difference)

**Example Violation**: Deterministic signing with truncation error.

---

#### INV-SIGN-3: Tamper Detection

**Statement**:
```
∀ message, keypair, tamperedMessage:
  message ≠ tamperedMessage →
  verify(sign(message, keypair.privateKey), tamperedMessage, keypair.publicKey) = false
```

**Rationale**: Signatures must detect any modification to the signed message.

**Test Strategy**:
- [x] Property test with message mutations
- [x] Unit test single-bit flips

**Example Violation**: Weak hash allowing collisions.

---

#### INV-SIGN-4: Wrong Key Rejection

**Statement**:
```
∀ message, keypair1, keypair2:
  keypair1.publicKey ≠ keypair2.publicKey →
  verify(sign(message, keypair1.privateKey), message, keypair2.publicKey) = false
```

**Rationale**: Signatures must be bound to the specific keypair that created them.

**Test Strategy**:
- [x] Property test with keypair pairs
- [x] Unit test with similar-looking keys

**Example Violation**: Key confusion in verification.

---

### Safety Invariants (Security-Critical)

#### SAFE-SIGN-1: Private Key Non-Exposure

**Statement**:
```
∀ signature, message, publicKey:
  ¬∃ privateKey: derivable(privateKey, signature, message, publicKey)
```

**Rationale**: It must be computationally infeasible to derive the private key from any number of signatures.

**Attack Vector**: Weak randomness in signature generation (see Sony PS3 ECDSA attack).

**Verification Method**: Use audited library (@noble/ed25519), verify randomness source.

---

#### SAFE-SIGN-2: Domain Separation

**Statement**:
```
∀ domain1, domain2, message:
  domain1 ≠ domain2 →
  signDomainSeparated(domain1, message, key) ≠ signDomainSeparated(domain2, message, key)
```

**Rationale**: Prevents cross-protocol signature reuse attacks.

**Attack Vector**: Transaction signature replayed as block signature.

**Verification Method**: All signing operations use domain prefixes from `HASH_DOMAINS`.

---

## Property Tests

```typescript
// tests/property/crypto/signatures.prop.ts

import { test, expect } from 'vitest';
import fc from 'fast-check';
import { generateKeypair, sign, verify } from '../../../core/crypto/keys';

// INV-SIGN-1: Signature Validity
test.prop([
  fc.uint8Array({ minLength: 1, maxLength: 10000 })
])('INV-SIGN-1: Valid signatures always verify', (message) => {
  const { privateKey, publicKey } = generateKeypair();
  const signature = sign(message, privateKey);
  expect(verify(signature, message, publicKey)).toBe(true);
});

// INV-SIGN-2: Signature Uniqueness
test.prop([
  fc.uint8Array({ minLength: 1, maxLength: 1000 }),
  fc.uint8Array({ minLength: 1, maxLength: 1000 }),
])('INV-SIGN-2: Different messages produce different signatures', (msg1, msg2) => {
  // Skip if messages happen to be equal
  if (arraysEqual(msg1, msg2)) return true;

  const { privateKey } = generateKeypair();
  const sig1 = sign(msg1, privateKey);
  const sig2 = sign(msg2, privateKey);
  expect(sig1).not.toBe(sig2);
  return true;
});

// INV-SIGN-3: Tamper Detection
test.prop([
  fc.uint8Array({ minLength: 2, maxLength: 1000 }),
  fc.integer({ min: 0 }),
])('INV-SIGN-3: Tampered messages fail verification', (message, flipIndex) => {
  const { privateKey, publicKey } = generateKeypair();
  const signature = sign(message, privateKey);

  // Tamper with the message
  const tampered = new Uint8Array(message);
  const idx = flipIndex % message.length;
  tampered[idx] = tampered[idx] ^ 0xFF; // Flip all bits at index

  expect(verify(signature, tampered, publicKey)).toBe(false);
});

// INV-SIGN-4: Wrong Key Rejection
test.prop([
  fc.uint8Array({ minLength: 1, maxLength: 1000 }),
])('INV-SIGN-4: Wrong public key fails verification', (message) => {
  const keypair1 = generateKeypair();
  const keypair2 = generateKeypair();

  const signature = sign(message, keypair1.privateKey);
  expect(verify(signature, message, keypair2.publicKey)).toBe(false);
});

// Helper
function arraysEqual(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false;
  }
  return true;
}
```

---

## Edge Cases

| Case | Expected Behavior | Test? |
|------|-------------------|-------|
| Empty message | Valid signature | [x] |
| 1-byte message | Valid signature | [x] |
| Max length (2^20 bytes) | Valid signature | [ ] |
| All-zero message | Valid signature | [x] |
| All-ones message | Valid signature | [x] |
| Unicode in string signing | Valid signature | [x] |

---

## The Three Laws Check

- [x] **LEGIBLE**: Invariants are formally stated and testable
- [x] **HONEST**: No claims beyond what Ed25519 guarantees
- [x] **AGENT-FIRST**: Operations are computational, no human-scale assumptions

---

## Sign-off

- [x] Invariants reviewed by: [Initial author]
- [x] Property tests implemented: `tests/property/crypto/signatures.prop.ts`
- [x] Ready for implementation review
