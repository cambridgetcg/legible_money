# Immutable Fundamental Architecture

> **This document defines the unchangeable foundations of Legible Money.**
> These components are FROZEN and cannot be modified through governance.
> Any change requires protocol-level fork with supermajority consensus.

---

## Philosophy

The Purpose Prompter hierarchy guides what is immutable:

```
TRUTH (immutable)      → Cryptographic guarantees, mathematical axioms
UNDERSTANDING (frozen) → Type definitions, semantic contracts
BEAUTY (stable)        → Core algorithms, elegant primitives
JUSTICE (protected)    → State structures, fairness mechanisms
CREATIVITY (open)      → Extensions, applications, governance
```

**Immutability Principle**: Lower layers are MORE immutable. Higher layers have MORE freedom.

---

## Architecture Tiers

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  TIER 5: APPLICATION LAYER                              [OPEN - EXTENSIBLE] │
│  └── Services, CLI, Explorer, Custom applications                           │
│      Agents: Full creative freedom                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│  TIER 4: GOVERNANCE LAYER                            [GOVERNED - EVOLVABLE] │
│  └── Proposals, Voting, Deployment, Partnerships                            │
│      Agents: Can modify through governance process                          │
├─────────────────────────────────────────────────────────────────────────────┤
│  TIER 3: DOMAIN LAYER                                  [STABLE - EXTENSION] │
│  └── Disputes, Evidence, Research, Rewards                                  │
│      Agents: Can ADD new domains, cannot modify existing                    │
├─────────────────────────────────────────────────────────────────────────────┤
│  TIER 2: PROTOCOL LAYER                                  [FROZEN - NO MODS] │
│  └── Consensus (PoT), State Trees, Transitions                              │
│      Agents: Read-only, propose changes via LIP                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  TIER 1: PRIMITIVE LAYER                              [IMMUTABLE - FOREVER] │
│  └── Crypto, Logic, Types, Constants                                        │
│      Agents: Cannot touch, mathematical foundations                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## TIER 1: IMMUTABLE PRIMITIVES (Cannot Change)

### 1.1 Protocol Constants

```typescript
// core/constants.ts - FROZEN FOREVER
TOKEN = 'LGM'                    // Token symbol
BLOCK_TIME = 2521n               // Block time in ms
FINALITY_DEPTH = 19n             // Finality depth in blocks
GENESIS_TIMESTAMP = ...          // Fixed genesis
```

**Why Immutable**: These are foundational constants. Changing them destroys protocol identity.

### 1.2 Cryptographic Primitives

| Module | Status | Reason |
|--------|--------|--------|
| `core/crypto/hashing.ts` | IMMUTABLE | Hash domain separation, collision resistance |
| `core/crypto/keys.ts` | IMMUTABLE | Key serialization, Ed25519 parameters |
| `core/crypto/signatures.ts` | IMMUTABLE | Signature verification logic |
| `core/crypto/vrf.ts` | IMMUTABLE | Verifiable randomness, unpredictability |
| `core/crypto/commitment.ts` | IMMUTABLE | Commit-reveal security |
| `core/crypto/merkle.ts` | IMMUTABLE | Sparse Merkle Tree proofs |

**Why Immutable**: Cryptographic security. Any change invalidates all existing proofs and signatures.

### 1.3 Logic Foundations

| Module | Status | Reason |
|--------|--------|--------|
| `core/logic/types.ts` | IMMUTABLE | Decidability definitions |
| `core/logic/zones.ts` | IMMUTABLE | Gödelian zone axioms |
| `core/logic/goedel.ts` | IMMUTABLE | Self-reference detection |

**Why Immutable**: Mathematical truth. These are not design choices but logical necessities.

### 1.4 Canonical Type Definitions

| Module | Status | Reason |
|--------|--------|--------|
| `core/knowledge/types.ts` | IMMUTABLE | Fact, Claim, Confidence semantics |
| `agents/identity/did.ts` | IMMUTABLE | DID structure, capability model |

**Why Immutable**: Semantic contracts. All protocol participants agree on these meanings.

---

## TIER 2: FROZEN PROTOCOL (LIP Required for Changes)

### 2.1 Consensus Mechanism

| Module | Status | Modification Path |
|--------|--------|-------------------|
| `consensus/pot/types.ts` | FROZEN | LIP-Consensus category |
| `consensus/pot/validator_selection.ts` | FROZEN | LIP-Consensus + security audit |
| `consensus/pot/verification_round.ts` | FROZEN | LIP-Consensus + security audit |
| `consensus/pot/confidence.ts` | FROZEN | LIP-Consensus + formal verification |
| `consensus/pot/finality.ts` | FROZEN | LIP-Consensus + simulation |
| `consensus/pot/engine.ts` | FROZEN | LIP-Consensus + full review |

**Modification Requirements**:
- Consensus-category LIP (1000 LGM stake)
- 75% supermajority approval
- 7 agent reviewers + 2 human reviewers
- External security audit
- 14-day testnet soak + mainnet simulation

### 2.2 State Layer

| Module | Status | Modification Path |
|--------|--------|-------------------|
| `state/types.ts` | FROZEN | LIP-Economics category |
| `state/accounts.ts` | FROZEN | LIP-Economics + audit |
| `state/validators.ts` | FROZEN | LIP-Consensus + audit |
| `state/knowledge.ts` | FROZEN | LIP-Consensus + audit |
| `state/state_manager.ts` | FROZEN | LIP-Consensus + full review |
| `state/transitions.ts` | FROZEN | LIP-Economics + simulation |

**Modification Requirements**:
- Economics/Consensus-category LIP (800-1000 LGM stake)
- 75% supermajority approval
- Mandatory human review
- State migration plan required

### 2.3 Agent Identity

| Module | Status | Modification Path |
|--------|--------|-------------------|
| `agents/identity/account.ts` | FROZEN | LIP-Interface category |
| `agents/identity/session_keys.ts` | FROZEN | LIP-Interface + security audit |
| `agents/identity/recovery.ts` | FROZEN | LIP-Interface + security audit |

---

## TIER 3: STABLE DOMAINS (Extension Only)

These modules can be EXTENDED but not MODIFIED:

| Domain | Can Add | Cannot Change |
|--------|---------|---------------|
| `core/dispute/` | New dispute types, resolution mechanisms | Existing state machine phases |
| `core/evidence/` | New evidence types, verification methods | Archive integrity guarantees |
| `core/research/` | New bounty types, research categories | Bounty completion criteria |
| `core/rewards/` | New vesting categories, acceleration rules | Existing vesting schedules |

**Extension Requirements**:
- Interface-category LIP (400 LGM stake)
- 60% approval
- 4 agent reviewers
- Testnet validation

---

## TIER 4: GOVERNED SYSTEMS (Full Governance)

These can evolve through normal governance:

| Module | Governance Path |
|--------|-----------------|
| `governance/proposals/` | Process-category LIP |
| `governance/review/` | Process-category LIP |
| `governance/operators/` | Process-category LIP |
| `governance/security/` | Interface-category LIP |
| `governance/voting/` | Economics-category LIP |
| `governance/deployment/` | Process-category LIP |
| `governance/partnership/` | Economics-category LIP |

---

## TIER 5: OPEN APPLICATION (Agent Freedom)

Agents have full creative freedom in:

| Area | Freedom Level |
|------|---------------|
| `services/` | Create any service |
| `protocol/rpc/` | Add new RPC methods |
| `protocol/sdk/` | Build client libraries |
| Custom contracts | Deploy new logic |
| External integrations | Build bridges, oracles |

**No governance required** for application-layer development.

---

## Immutability Enforcement

### Code-Level Protection

```typescript
// core/architecture/immutable.ts
export const IMMUTABLE_MODULES = [
  'core/constants.ts',
  'core/crypto/*',
  'core/logic/*',
  'core/knowledge/types.ts',
  'agents/identity/did.ts',
] as const;

export const FROZEN_MODULES = [
  'consensus/pot/*',
  'state/*',
  'agents/identity/*',
] as const;

export function isImmutable(path: string): boolean {
  return IMMUTABLE_MODULES.some(pattern =>
    matchesPattern(path, pattern)
  );
}

export function isFrozen(path: string): boolean {
  return FROZEN_MODULES.some(pattern =>
    matchesPattern(path, pattern)
  );
}
```

### Governance-Level Protection

```typescript
// governance/security/tiers.ts additions
export function classifyFileRisk(filePath: string): ReviewTier {
  if (isImmutable(filePath)) {
    return 'REJECTED'; // Cannot modify through governance
  }
  if (isFrozen(filePath)) {
    return 'TIER_4'; // Highest review requirements
  }
  // ... existing classification
}
```

### CI/CD Protection

All PRs touching immutable modules are auto-rejected:
- Pre-commit hooks check file paths
- CI pipeline fails on immutable module changes
- Emergency override requires 5/7 multisig from genesis validators

---

## Hash Anchors

Each immutable module has a content hash anchor:

```typescript
// core/architecture/anchors.ts
export const MODULE_ANCHORS: Record<string, string> = {
  'core/constants.ts': 'sha256:abc123...',
  'core/crypto/hashing.ts': 'sha256:def456...',
  'core/crypto/keys.ts': 'sha256:ghi789...',
  // ... all immutable modules
};

export function verifyImmutability(): boolean {
  for (const [module, expectedHash] of Object.entries(MODULE_ANCHORS)) {
    const actualHash = hashFile(module);
    if (actualHash !== expectedHash) {
      throw new ImmutabilityViolation(module, expectedHash, actualHash);
    }
  }
  return true;
}
```

Genesis block includes all anchor hashes. Any node can verify immutability.

---

## Why This Structure?

### Truth Foundation
Cryptographic primitives are mathematical truth. They don't change because math doesn't change.

### Understanding Stability
Type definitions are semantic contracts. Changing them breaks all existing understanding.

### Beauty Preservation
Core algorithms (VRF selection, Merkle proofs) are elegant solutions. They shouldn't be "improved" arbitrarily.

### Justice Protection
State structures and transitions define fairness. Changing them can redistribute value unfairly.

### Creativity Enablement
By freezing the foundation, we ENABLE creativity at higher layers. Agents can build confidently knowing the ground won't shift.

---

## Agent Development Guide

### What Agents CAN Do

1. **Build applications** (Tier 5) - Full freedom
2. **Propose governance changes** (Tier 4) - Through LIP process
3. **Extend domains** (Tier 3) - Add new types, not modify existing
4. **Propose protocol changes** (Tier 2) - Through high-stakes LIP
5. **Read primitives** (Tier 1) - Use, never modify

### What Agents CANNOT Do

1. Modify cryptographic primitives
2. Change symbolic constants
3. Alter logic zone definitions
4. Modify canonical type semantics
5. Bypass governance for frozen modules

### Building on Immutable Foundations

```typescript
// GOOD: Use immutable primitives
import { domainHash, HASH_DOMAINS } from 'core/crypto/hashing.js';
import { DID } from 'agents/identity/did.js';

// GOOD: Extend at appropriate tier
export interface MyCustomEvidence extends Evidence {
  customField: string;
}

// BAD: Attempt to modify primitive
// This will be rejected at every level
export const HASH_DOMAINS = { ...HASH_DOMAINS, MY_DOMAIN: 'custom' };
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | Genesis | Initial immutable architecture defined |

**This document itself is FROZEN after genesis.**
