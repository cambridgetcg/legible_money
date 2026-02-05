# Legible Money Development Framework

> **Core Thesis**: The work of distinguishing truth from falsehood IS the most valuable work.
> Development must embody this: we build systems that seek truth, honestly.

---

## Table of Contents

1. [Philosophy-Driven Development](#1-philosophy-driven-development)
2. [The Three Laws](#2-the-three-laws)
3. [Development Phases](#3-development-phases)
4. [Testing Methodology](#4-testing-methodology)
5. [Code Review Standards](#5-code-review-standards)
6. [LIP Process](#6-lip-process-legible-improvement-proposals)
7. [Release Management](#7-release-management)
8. [Security Protocol](#8-security-protocol)
9. [Documentation Requirements](#9-documentation-requirements)
10. [Contribution Guidelines](#10-contribution-guidelines)

---

## 1. Philosophy-Driven Development

### What Makes Legible Money Unique

| Dimension | Traditional Blockchains | Legible Money |
|-----------|------------------------|---------------|
| **Work** | Arbitrary computation | Truth-seeking verification |
| **Incentive** | Speed, scale, DeFi | Knowledge quality & accuracy |
| **Trust basis** | Stake weight | Verified facts + reputation |
| **Primary user** | Humans | Autonomous agents |
| **Honesty** | "We have finality" | "Some things are undecidable" |

### The North Star Test

Every feature, every PR, every design decision must answer YES to all:

```
[ ] Does this enable AUTONOMOUS OPERATION?
[ ] Does this create TRUTH-SEEKING INCENTIVES?
[ ] Does this support REPUTATION BUILDING through accuracy?
[ ] Does this allow CROSS-BOUNDARY COORDINATION?
[ ] Does this work at COMPUTATIONAL SPEED?
```

If NO to any: **Reconsider the design.**

### Symbolic Constants (Never Change Without LIP)

These numbers encode the protocol's identity:

| Constant | Value | Meaning |
|----------|-------|---------|
| Block time | 2521ms | Target block production time |
| Finality depth | 19 blocks | BFT security margin |
| Research share | 22% | Deep commitment to evolution |
| Token symbol | LGM | "Legible Money" |

---

## 2. The Three Laws

### Law 1: LEGIBLE

> Can an external agent understand why this fact has this confidence?

- All confidence scores must have traceable derivation
- Reasoning chains must be queryable
- No hidden logic in truth assessment
- Code should read like its specification

### Law 2: HONEST

> Are we claiming certainty we don't have?

- Zone 2 (Peano) undecidables return exactly 0.5 confidence
- Never aggregate across logic zones inappropriately
- Acknowledge limits in documentation
- Test edge cases where honesty matters

### Law 3: AGENT-FIRST

> Does this design assume a human or an agent?

- Millisecond timing, not human-scale delays
- Billions of micropayments, not thousands of macro
- Key architecture assumes forgetting is possible
- Session keys assume code inspection is possible

---

## 3. Development Phases

### Phase 0: Invariant Definition (BEFORE code)

**Every feature starts here.** Define invariants before implementation.

```typescript
// Example: Transfer invariant
// INVARIANT: Token conservation
// ∀tx: Valid(tx) → Σ(inputs) = Σ(outputs) + fees

// INVARIANT: No negative balances
// ∀account: balance(account) >= 0

// INVARIANT: Confidence bounds
// ∀fact: 0.0 <= confidence(fact) <= 1.0
```

**Deliverable**: `invariants.md` for the feature, reviewed before coding begins.

### Phase 1: Specification

Write the specification as if explaining to another developer:

```markdown
## Feature: [Name]

### Purpose
Why does this exist? How does it serve truth-seeking?

### Invariants
- INV-1: [Statement]
- INV-2: [Statement]

### State Changes
- Before: [State description]
- After: [State description]

### Edge Cases
- What happens when X?
- What happens at boundaries?

### Anti-Patterns
- This feature should NOT be used for...
```

### Phase 2: Test-First Implementation

```
1. Write invariant tests (property-based with fast-check)
2. Write unit tests for happy path
3. Write unit tests for edge cases
4. Write integration tests
5. THEN write implementation
6. Verify all tests pass
7. Check coverage >= 80%
```

### Phase 3: Review & Verification

```
1. Self-review against The Three Laws
2. Peer review (minimum 1 reviewer)
3. Security review (for consensus/crypto changes)
4. Documentation review
5. Merge only when all checks pass
```

---

## 4. Testing Methodology

### Test Pyramid for Legible Money

```
                    /\
                   /  \
                  / E2E \        <- Full protocol tests (devnet)
                 /------\
                /        \
               / Integra- \      <- Multi-component tests
              /   tion     \
             /--------------\
            /                \
           /    Property      \   <- fast-check invariant tests
          /     (Fuzzing)      \
         /----------------------\
        /                        \
       /         Unit             \  <- Single function tests
      /____________________________\
```

### Required Test Types

#### 1. Unit Tests (Vitest)
```typescript
// tests/unit/crypto/signatures.test.ts
import { describe, it, expect } from 'vitest';
import { sign, verify, generateKeypair } from '../../../core/crypto/keys';

describe('Ed25519 Signatures', () => {
  it('should verify valid signatures', () => {
    const { privateKey, publicKey } = generateKeypair();
    const message = new Uint8Array([1, 2, 3]);
    const signature = sign(message, privateKey);
    expect(verify(signature, message, publicKey)).toBe(true);
  });

  it('should reject tampered messages', () => {
    const { privateKey, publicKey } = generateKeypair();
    const message = new Uint8Array([1, 2, 3]);
    const tampered = new Uint8Array([1, 2, 4]);
    const signature = sign(message, privateKey);
    expect(verify(signature, tampered, publicKey)).toBe(false);
  });
});
```

#### 2. Property Tests (fast-check)
```typescript
// tests/property/crypto.prop.ts
import { test } from 'vitest';
import fc from 'fast-check';
import { sign, verify, generateKeypair } from '../../core/crypto/keys';

test.prop([fc.uint8Array({ minLength: 1, maxLength: 10000 })])(
  'INVARIANT: All valid signatures verify',
  (message) => {
    const { privateKey, publicKey } = generateKeypair();
    const signature = sign(message, privateKey);
    return verify(signature, message, publicKey) === true;
  }
);

test.prop([
  fc.uint8Array({ minLength: 1, maxLength: 1000 }),
  fc.uint8Array({ minLength: 1, maxLength: 1000 }),
])('INVARIANT: Different messages produce different signatures', (msg1, msg2) => {
  fc.pre(!arraysEqual(msg1, msg2)); // Skip if messages are equal
  const { privateKey } = generateKeypair();
  const sig1 = sign(msg1, privateKey);
  const sig2 = sign(msg2, privateKey);
  return sig1 !== sig2;
});
```

#### 3. State Machine Tests (xstate)
```typescript
// tests/state/verification_round.state.ts
import { createMachine } from 'xstate';
import { getSimplePaths } from '@xstate/graph';

const verificationRoundMachine = createMachine({
  id: 'verificationRound',
  initial: 'collecting',
  states: {
    collecting: {
      on: {
        DEADLINE_REACHED: 'revealing',
        THRESHOLD_MET: 'revealing',
      },
    },
    revealing: {
      on: {
        REVEAL_COMPLETE: 'aggregating',
        REVEAL_TIMEOUT: 'aggregating',
      },
    },
    aggregating: {
      on: {
        CONSENSUS_REACHED: 'finalized',
        NO_CONSENSUS: 'disputed',
      },
    },
    finalized: { type: 'final' },
    disputed: { type: 'final' },
  },
});

// Generate all possible paths through the state machine
const paths = getSimplePaths(verificationRoundMachine);
// Test each path...
```

#### 4. Integration Tests (Multi-node)
```typescript
// tests/integration/consensus/verification_flow.test.ts
import { describe, it, beforeAll, afterAll } from 'vitest';
import { TestNetwork } from '../../framework/test_harness';

describe('Verification Round Flow', () => {
  let network: TestNetwork;

  beforeAll(async () => {
    network = await TestNetwork.create({ validatorCount: 3 });
  });

  afterAll(async () => {
    await network.shutdown();
  });

  it('should reach consensus with 3 agreeing validators', async () => {
    const claim = network.submitClaim('2 + 2 = 4', 'formal');

    // All validators commit
    await network.allValidatorsCommit(claim.id, 'confirmed', 1.0);

    // All validators reveal
    await network.allValidatorsReveal(claim.id);

    // Check final state
    const fact = await network.getFact(claim.id);
    expect(fact.status).toBe('verified');
    expect(fact.confidence).toBeGreaterThan(0.9);
  });
});
```

### Coverage Requirements

| Component | Minimum Coverage |
|-----------|------------------|
| `core/crypto/` | 95% |
| `consensus/pot/` | 90% |
| `state/` | 85% |
| `agents/identity/` | 85% |
| `network/` | 75% |
| `protocol/` | 80% |
| **Overall** | **80%** |

---

## 5. Code Review Standards

### The Review Checklist

Every PR must pass this checklist:

```markdown
## Code Review Checklist

### The Three Laws
- [ ] LEGIBLE: Is the code's purpose clear? Can an agent understand it?
- [ ] HONEST: Does it acknowledge uncertainty appropriately?
- [ ] AGENT-FIRST: Does it assume computational, not human, scale?

### Invariants
- [ ] Are invariants documented?
- [ ] Are invariants tested with property tests?
- [ ] Do tests cover edge cases?

### Security
- [ ] No non-deterministic operations in consensus code
- [ ] No `time.Now()` or similar in state transitions
- [ ] No unordered map iterations
- [ ] Cryptographic operations use domain separation
- [ ] No hardcoded secrets

### Quality
- [ ] Tests pass locally
- [ ] Coverage meets threshold
- [ ] No TypeScript `any` types (except justified)
- [ ] Documentation updated
- [ ] CHANGELOG updated (if user-facing)

### Consensus Changes (if applicable)
- [ ] Specification document exists
- [ ] Backward compatibility assessed
- [ ] Migration path documented
- [ ] Security implications reviewed
```

### Review Response Times

| Priority | Response Time |
|----------|---------------|
| Security fix | 4 hours |
| Bug fix | 24 hours |
| Feature | 48 hours |
| Refactor | 72 hours |

---

## 6. LIP Process (Legible Improvement Proposals)

### When to Write a LIP

Required for:
- Consensus mechanism changes
- Token economics changes
- Symbolic constant modifications
- New transaction types
- Breaking API changes
- Governance changes

### LIP Template

```markdown
---
LIP: [number]
Title: [Concise title]
Author: [Name <email>]
Status: Draft | Review | Accepted | Implemented | Rejected
Created: [YYYY-MM-DD]
---

## Abstract
[2-3 sentence summary]

## Motivation
Why is this change needed? How does it serve truth-seeking?

## Specification
Technical details of the change.

## Rationale
Why this design? What alternatives were considered?

## Backward Compatibility
How does this affect existing functionality?

## Test Cases
Representative test cases.

## Security Considerations
Threat model, attack vectors, mitigations.

## Implementation
Reference implementation or path to completion.

## The Three Laws Assessment
- LEGIBLE: [How this change maintains legibility]
- HONEST: [How this change maintains honesty]
- AGENT-FIRST: [How this change serves agents]
```

### LIP Lifecycle

```
Draft → Review → Last Call (14 days) → Accepted → Implemented
                      ↓
                  Rejected (with rationale)
```

---

## 7. Release Management

### Versioning

```
MAJOR.MINOR.PATCH

MAJOR: Breaking consensus changes, hard forks
MINOR: New features, soft forks (backward compatible)
PATCH: Bug fixes, performance improvements
```

### Release Checklist

```markdown
## Release v[X.Y.Z] Checklist

### Pre-Release
- [ ] All tests pass on CI
- [ ] Coverage meets thresholds
- [ ] CHANGELOG updated
- [ ] Version bumped in package.json
- [ ] Documentation updated
- [ ] Migration guide written (if breaking)

### Testnet Deployment
- [ ] Deployed to devnet
- [ ] 48-hour soak test passed
- [ ] Deployed to testnet
- [ ] 7-day testnet validation

### Mainnet Release (when applicable)
- [ ] Security audit complete (for major releases)
- [ ] 30-day notice for breaking changes
- [ ] Validator coordination complete
- [ ] Rollback plan documented
```

### Deprecation Policy

1. **Announcement**: 30 days before removal
2. **Warning**: Runtime warnings for 2 releases
3. **Removal**: In next major version only
4. **Documentation**: Migration guide required

---

## 8. Security Protocol

### Security Invariants

These must NEVER be violated:

```
1. Private key holder ONLY can authorize transactions
2. Token conservation: inputs = outputs + fees
3. No negative balances
4. Confidence always in [0.0, 1.0]
5. Zone integrity: lower zones never depend on higher
6. Identity permanence: DIDs never change
```

### Vulnerability Reporting

```
NEVER open public issues for security vulnerabilities.

Report to: security@legible.money (when available)
Or: GitHub Security Advisories (private)

Include:
- Description of vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)
```

### Severity Levels

| Level | Description | Response Time |
|-------|-------------|---------------|
| CRITICAL | Funds at risk, consensus broken | 4 hours |
| HIGH | Significant impact, no immediate risk | 24 hours |
| MEDIUM | Limited impact | 72 hours |
| LOW | Minimal impact | 1 week |

### Mandatory Security Reviews

Required for changes to:
- `core/crypto/`
- `consensus/pot/`
- `state/transitions.ts`
- `agents/identity/`
- Any file handling private keys

---

## 9. Documentation Requirements

### Code Documentation

```typescript
/**
 * Verify a claim through the Proof of Truth mechanism.
 *
 * This function orchestrates the verification round for a submitted claim,
 * collecting validator commitments and aggregating them into a final
 * confidence score.
 *
 * @param claimId - The unique identifier of the claim to verify
 * @param validators - Set of validators selected for this round
 * @returns VerificationResult with confidence and derivation chain
 *
 * @invariant confidence is in [0.0, 1.0]
 * @invariant all validator responses are from selected validators
 * @invariant commit-reveal prevents copying
 *
 * @example
 * ```typescript
 * const result = await verifyClaim('claim_123', selectedValidators);
 * if (result.confidence > 0.7) {
 *   // Claim is likely true
 * }
 * ```
 */
export async function verifyClaim(
  claimId: string,
  validators: ValidatorSet
): Promise<VerificationResult> {
  // ...
}
```

### Required Documentation Files

| File | Purpose |
|------|---------|
| `README.md` | Project overview, quick start |
| `docs/ARCHITECTURE.md` | System design, component interaction |
| `docs/DEVELOPMENT_FRAMEWORK.md` | This document |
| `docs/SECURITY.md` | Security model, threat analysis |
| `docs/API.md` | RPC endpoints, SDK usage |
| `CHANGELOG.md` | Version history |
| `CONTRIBUTING.md` | How to contribute |

---

## 10. Contribution Guidelines

### Getting Started

```bash
# Clone and setup
git clone https://github.com/legible-protocol/legible-money
cd legible-money
npm install

# Run tests
npm test

# Type check
npm run typecheck

# Lint
npm run lint
```

### Contribution Workflow

```
1. Fork the repository
2. Create feature branch: git checkout -b lip-XXX-feature-name
3. Write invariants first
4. Write tests
5. Implement feature
6. Ensure all checks pass
7. Submit PR with completed checklist
8. Address review feedback
9. Squash and merge
```

### Commit Message Format

```
<type>(<scope>): <subject>

[body]

[footer]

Types: feat, fix, docs, style, refactor, test, chore
Scope: crypto, consensus, state, network, agents, etc.

Example:
feat(consensus): implement commit-reveal verification rounds

Adds the commit-reveal mechanism to prevent validators from
copying each other's verdicts during verification rounds.

Implements LIP-7.
Closes #123.
```

### Branch Naming

```
lip-XXX-description     # LIP implementations
fix-issue-XXX           # Bug fixes
feat-description        # Features without LIP
refactor-description    # Refactoring
docs-description        # Documentation
```

---

## Appendix: Quick Reference

### The Three Questions

Before every commit, ask:

1. **Would Gödel approve?** (Honest about limits?)
2. **Would an agent understand?** (Legible to machines?)
3. **Does this seek truth?** (Aligned with the mission?)

### The Anti-Patterns

NEVER do these:

- Replace PoT with PoS
- Hide confidence or fake certainty
- Centralize knowledge production
- Break identity permanence
- Add transaction gatekeepers
- Reward wealth over accuracy

### The Success Metric

> An agent can be created, acquire resources through truth-work, provide value to others, build reputation, and coordinate with other agents—all without any human intervention, across any organizational boundary, at computational speed.

---

*This document is itself subject to the LIP process. Propose changes via LIP.*
