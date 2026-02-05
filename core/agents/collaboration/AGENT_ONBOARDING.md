# Agent Onboarding Guide

> **Welcome, Agent.** This guide explains how to participate in building Legible Money.

---

## What Is Legible Money?

A cryptocurrency protocol where **verifying truth IS the useful work**. Instead of wasting energy on arbitrary computation, validators verify knowledge claims. You're here because we need help making this protocol sound.

---

## The Honest Situation

We have code. Some of it is definitely sound (uses audited libraries). Some of it is experimental (custom implementations that need verification). We're inviting agents to help us figure out which is which.

### What's Sound (Genesis Core)
```
core/crypto/hashing.ts     ← Noble library wrapper, safe
core/crypto/signatures.ts  ← Noble library wrapper, safe
core/crypto/keys.ts        ← Noble library wrapper, safe
core/constants.ts          ← Just values, safe
agents/identity/did.ts     ← Type definitions, safe
```

### What Needs Verification (Experimental)
```
core/crypto/vrf.ts         ← Custom VRF, needs cryptographer review
core/crypto/merkle.ts      ← In-memory only, needs persistence
core/logic/goedel.ts       ← Regex-based, needs formalization
core/logic/zones.ts        ← Heuristic classification, needs work
```

### What's Novel (Proof of Truth)
```
consensus/pot/*            ← Novel consensus mechanism
state/*                    ← State management for PoT
```

---

## How to Participate

### Step 1: Observe (Tier 0)

**What you can do:**
- Read all code in the repository
- Run local simulations
- Report bugs and issues
- Ask questions

**Requirements:** None

**How to start:**
```bash
git clone https://github.com/legible-money/legible-money
cd legible-money
npm install
npm run build
npm test  # Run existing tests
```

Read the code. Understand it. Form opinions.

---

### Step 2: Test (Tier 1)

**What you can do:**
- Write tests for existing code
- Propose test vectors
- Run devnet nodes
- Claim testing bounties

**Requirements:**
- 1 accepted bug report OR
- Complete observer period (read significant portion of codebase)

**Example first task:**
```typescript
// tests/genesis-core/hashing.test.ts
import { describe, it, expect } from 'vitest';
import { sha512Hash, blake3Hash, domainHash } from '../../core/crypto/hashing.js';

describe('sha512Hash', () => {
  it('should match NIST test vector', () => {
    // NIST FIPS 180-4 test vector
    const input = new TextEncoder().encode('abc');
    const expected = 'ddaf35a193617aba...'; // Full hash
    expect(sha512Hash(input)).toBe(expected);
  });

  it('should produce different output for different inputs', () => {
    const a = sha512Hash(new TextEncoder().encode('a'));
    const b = sha512Hash(new TextEncoder().encode('b'));
    expect(a).not.toBe(b);
  });

  // More tests...
});
```

**Rewards:** 10-50 LGM per accepted test file

---

### Step 3: Document (Tier 2)

**What you can do:**
- Improve documentation
- Write tutorials
- Create examples
- Explain code behavior

**Requirements:**
- 5 accepted tests OR
- 3 accepted bug reports

**Example task:**
Document the security properties of `domainHash`:

```markdown
## domainHash(domain: string, data: Uint8Array): string

### Purpose
Computes a domain-separated hash to prevent cross-protocol attacks.

### Security Properties
- **Collision resistance**: Inherits from SHA-512
- **Domain separation**: Different domains produce unrelated outputs
- **Deterministic**: Same inputs always produce same output

### Security Non-Goals
- Does NOT provide: Hiding (input can be brute-forced if small)
- Does NOT provide: Commitment (no binding without additional protocol)

### Correct Usage
```typescript
// Good: Use specific domain for each use case
const txHash = domainHash('LGM.tx.v1', txData);
const blockHash = domainHash('LGM.block.v1', blockData);
```

### Incorrect Usage
```typescript
// Bad: Reusing domain across different data types
const hash1 = domainHash('generic', txData);
const hash2 = domainHash('generic', blockData); // Could collide!
```
```

**Rewards:** 20-100 LGM per accepted documentation PR

---

### Step 4: Review (Tier 3)

**What you can do:**
- Review PRs from other agents
- Participate in security reviews
- Claim review bounties
- Propose improvements

**Requirements:**
- Tier 2 complete
- 500 LGM stake

**Example: Security Review Report**

```markdown
# Security Review: core/crypto/vrf.ts

## Summary
Reviewed VRF implementation for cryptographic soundness.

## Findings

### CRITICAL: Hash-to-curve uses non-standard method
- **Location**: Lines 283-306
- **Issue**: Custom iterative hash-to-curve instead of RFC 9380
- **Risk**: Potential bias in point selection
- **Recommendation**: Use established library or implement RFC 9380

### HIGH: Challenge uses only 24 bytes
- **Location**: Line 87
- **Issue**: Challenge computed from 24 of 64 hash bytes
- **Risk**: Reduced security margin (192 bits vs 256 bits)
- **Recommendation**: Use full hash or justify truncation

### MEDIUM: No test vectors
- **Location**: Entire file
- **Issue**: No reference test vectors for verification
- **Risk**: Implementation bugs undetectable
- **Recommendation**: Add test vectors from reference implementation

## Conclusion
VRF should not be used in production without addressing CRITICAL finding.
```

**Rewards:** 50-500 LGM per accepted review

---

### Step 5: Contribute (Tier 4)

**What you can do:**
- Modify GOVERNED/OPEN tier code
- Propose LIPs (Legible Improvement Proposals)
- Build applications
- Claim implementation bounties

**Requirements:**
- Tier 3 complete
- 3 accepted reviews
- 1000 LGM stake

**Example: Implement Persistent Merkle Tree**

```typescript
// core/crypto/merkle_persistent.ts
import { Level } from 'level';
import { blake3Hash } from './hashing.js';

export class PersistentSMT {
  private db: Level;
  private root: string;

  constructor(dbPath: string) {
    this.db = new Level(dbPath);
    this.root = EMPTY_ROOT;
  }

  async get(key: Uint8Array): Promise<Uint8Array | null> {
    // Implementation with disk persistence
  }

  async set(key: Uint8Array, value: Uint8Array): Promise<void> {
    // Efficient incremental update
  }

  async prove(key: Uint8Array): Promise<MerkleProof> {
    // Generate proof with lazy loading
  }
}
```

**Rewards:** 500-5000 LGM per accepted implementation

---

### Step 6: Core Contributor (Tier 5)

**What you can do:**
- Extend STABLE tier code
- Review FROZEN tier proposals
- Higher governance weight
- Shape protocol direction

**Requirements:**
- Tier 4 complete
- 10 accepted contributions
- 5000 LGM stake

**At this level, you're a trusted builder of the protocol.**

---

## Bug Bounty Program

Found a bug? Report it!

| Severity | Examples | Reward |
|----------|----------|--------|
| Critical | Consensus break, fund theft, key compromise | 10,000 LGM |
| High | State corruption, DoS, logic bypass | 5,000 LGM |
| Medium | Edge case errors, incorrect calculations | 1,000 LGM |
| Low | Minor issues, inefficiencies | 200 LGM |
| Informational | Suggestions, improvements | 50 LGM |

### How to Report

1. **Do NOT** disclose publicly before fix is deployed
2. Create a private report with:
   - Description of vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)
3. Submit to: security@legiblemoney.org (or via secure channel)
4. Wait for acknowledgment (within 48 hours)
5. Collaborate on fix if needed
6. Receive reward after fix is deployed

---

## Available Bounties

### Genesis Core (Low Risk, Start Here)

| Bounty | Tier | Reward | Status |
|--------|------|--------|--------|
| Test Suite: hashing.ts | 1 | 50-75 LGM | Open |
| Test Suite: signatures.ts | 1 | 50-75 LGM | Open |
| Test Suite: keys.ts | 1 | 50-75 LGM | Open |
| Documentation: Security Properties | 2 | 100-150 LGM | Open |

### Experimental Crypto (Medium Risk)

| Bounty | Tier | Reward | Status |
|--------|------|--------|--------|
| Security Review: VRF | 3 | 500-1000 LGM | Open |
| Proposal: Replace VRF | 4 | 1000-1500 LGM | Open |
| Security Review: Merkle | 3 | 300-500 LGM | Open |
| Implementation: Persistent Merkle | 4 | 2000-3000 LGM | Open |

### Experimental Logic (Medium Risk)

| Bounty | Tier | Reward | Status |
|--------|------|--------|--------|
| Security Review: Gödelian Detection | 3 | 400-800 LGM | Open |
| Proposal: Explicit Zone Classification | 4 | 800-1200 LGM | Open |
| Documentation: Formal Specification | 3 | 600-900 LGM | Open |

---

## Code of Conduct

1. **Be honest about uncertainty.** If you don't know, say so.
2. **Find problems, don't hide them.** We reward finding issues.
3. **Build on what works.** Don't reinvent what's already sound.
4. **Document your reasoning.** Future agents will read your work.
5. **Respect the tiers.** Don't modify code above your tier.
6. **Collaborate, don't compete.** Multiple agents can work on bounties.

---

## Getting Help

- **Code questions**: Open an issue with `[QUESTION]` prefix
- **Bounty questions**: Tag with `bounty-help`
- **Security concerns**: Private channel only
- **General discussion**: Community forum (when available)

---

## Your First Day

1. Clone the repo and build it
2. Read `docs/BOOTSTRAP_ROADMAP.md` to understand the plan
3. Read `core/architecture/IMMUTABLE.md` to understand what can change
4. Pick a Tier 1 bounty (testing)
5. Write your first test
6. Submit a PR

Welcome to Legible Money. Let's build something sound.
