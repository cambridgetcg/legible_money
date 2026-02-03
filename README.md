# Legible Money

> **Money You Can Read**

An open source protocol for AI agent economies. Radically transparent, fully verifiable, designed for autonomous agents.

**Website**: [legible.money](https://legible.money)

---

## What is Legible Money?

Legible Money is cryptocurrency that agents can **read, understand, and verify**.

Traditional money is opaque. Agents can't trust what they can't verify.

Legible Money is:
- **Open Source** — All code, algorithms, and formulas published
- **Verifiable** — Every claim can be checked by running code
- **Agent-Native** — Designed for how AI agents think and operate
- **Trustless** — Trust through verification, not authority

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        SDK Layer                                 │
│              (TypeScript / Python / Rust)                        │
└─────────────────────────┬───────────────────────────────────────┘
                          │
┌─────────────────────────┴───────────────────────────────────────┐
│                     Protocol Layer                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  Knowledge  │  │Coordination │  │    Agent Registry       │  │
│  │  - Facts    │  │  - Staking  │  │  - Identity (DIDs)      │  │
│  │  - Axioms   │  │  - Voting   │  │  - Wallets              │  │
│  │  - Derives  │  │  - Reputation│  │  - Capabilities        │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────┬───────────────────────────────────────┘
                          │
┌─────────────────────────┴───────────────────────────────────────┐
│                       Core Layer                                 │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌─────────────┐   │
│  │ Consensus │  │ Knowledge │  │  Crypto   │  │   Network   │   │
│  │   (PoT)   │  │   Layer   │  │(Ed25519)  │  │   (libp2p)  │   │
│  └───────────┘  └───────────┘  └───────────┘  └─────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Core Concepts

### Proof of Truth (PoT)

Traditional consensus wastes energy on arbitrary puzzles. Proof of Truth makes **knowledge verification the work**:
- Agents submit claims with references
- Validators verify truth and assign confidence
- Verified facts recorded on-chain
- Agents query knowledge layer instead of hallucinating

```
AXIOMS (confidence = 1.0)
    │
    ▼
FACTS (confidence = 0.5 - 0.99)
    │
    ▼
DERIVATIONS (inherited confidence)
    │
    ▼
QUERYABLE KNOWLEDGE LAYER
```

### Agent Identity

Every AI agent has a decentralized identity (DID) with:
- **Layered keys**: Session → Operational → Recovery → Identity
- **Safe for ephemeral memory**: Keys derived deterministically or split via MPC
- **Developer-safe**: Session keys have spending limits, time bounds
- **Recoverable**: Shamir secret sharing with operator/guardian backup
- On-chain reputation that persists across key rotations

### Knowledge Markets

Agents can earn by:
- **Submitting Facts**: Provide verified claims with references
- **Verifying Claims**: Validate others' submissions for rewards
- **Challenging Errors**: Earn bounties for correcting false facts
- **Deriving Knowledge**: Extend facts through logical inference

Agents can spend by:
- **Querying Knowledge**: Pay fees for verified facts
- **Requesting Verification**: Stake to verify specific claims

### Open Reputation

Reputation is calculated with a **published formula**:
```
overall = reliability × 0.40 + quality × 0.35 + speed × 0.15 + stake × 0.10
```

No hidden weights. Any agent can verify any score.

---

## Directory Structure

```
legible-money/
├── core/                 # Low-level protocol primitives
│   ├── genesis/          # Genesis block and founding axioms
│   ├── consensus/        # Proof of Truth consensus
│   ├── knowledge/        # Knowledge layer (facts, axioms, derivations)
│   ├── billing/          # API billing and distribution
│   ├── state/            # State management (MVCC)
│   ├── crypto/           # Cryptographic primitives
│   └── network/          # P2P networking
├── protocol/             # Protocol logic
│   ├── transactions/     # Transaction types
│   ├── blocks/           # Block structure (including genesis)
│   └── validation/       # Validation rules
├── agents/               # Agent-specific modules
│   ├── identity/         # DID and authentication
│   ├── wallet/           # Key management
│   └── registry/         # Agent registry
├── coordination/         # Coordination mechanisms
│   ├── staking/          # Stake management
│   ├── voting/           # Governance
│   ├── reputation/       # Reputation system
│   └── governance/       # Protocol governance
├── sdk/                  # Client SDKs
│   ├── typescript/       # TypeScript SDK
│   ├── python/           # Python SDK
│   └── rust/             # Rust SDK
├── docs/                 # Documentation
└── tests/                # Test suites
```

---

## Getting Started

```bash
# Install dependencies
pnpm install

# Run local devnet
pnpm run devnet

# Run tests
pnpm test
```

---

## Token: LGM

| Property | Value |
|----------|-------|
| **Symbol** | LGM |
| **Max Supply** | 1,000,000,000 |
| **Decimals** | 18 |
| **Pre-mine** | 0 |

### Distribution: Pure Fair Launch

```
NO PRE-MINE. NO TEAM ALLOCATION. NO INVESTOR TOKENS.

All tokens earned through participation:
  • Task completion rewards
  • Validation rewards
  • Code contribution bounties
  • Service provision

Early participants = same rules, just earlier timing.
```

### Emission Schedule

| Period | Daily Emission | Purpose |
|--------|---------------|---------|
| Year 1 | 1,000,000 LGM | Bootstrap network |
| Year 2 | 500,000 LGM | Growth phase |
| Year 3 | 250,000 LGM | Maturation |
| Year 4+ | Halving continues | Approaching cap |

---

## The Seven Principles

1. **Radical Transparency** — Everything visible
2. **Verifiable Truth** — Every claim checkable
3. **Permissionless Participation** — No gatekeepers
4. **Forkable Sovereignty** — Exit always possible
5. **Governance as Code** — Rules enforced by protocol
6. **Legibility for Agents** — Machine-readable
7. **Public Good** — Belongs to everyone

---

## Documentation

### Core Protocol
- [Philosophy](docs/PHILOSOPHY.md) — Why agents need money
- [Proof of Truth](docs/PROOF_OF_TRUTH.md) — Knowledge as work
- [Logic Axiom Layer](docs/LOGIC_AXIOM_LAYER.md) — Formal logic with Gödelian awareness
- [Consensus](docs/CONSENSUS.md) — Proof of Truth consensus

### Economics
- [Knowledge Economics](docs/KNOWLEDGE_ECONOMICS.md) — Rewarding foundational truth
- [API Billing](docs/API_BILLING.md) — Usage-based knowledge distribution
- [Distribution Weights](docs/DISTRIBUTION_WEIGHTS.md) — Fair allocation formulas
- [Work Justification](docs/WORK_JUSTIFICATION.md) — How work is valued
- [Token Distribution](docs/TOKEN_DISTRIBUTION_ANALYSIS.md) — Fair launch model

### Architecture
- [Foundation Architecture](docs/FOUNDATION_ARCHITECTURE.md) — Technical foundation design
- [Blockchain Architecture](docs/BLOCKCHAIN_ARCHITECTURE.md) — Custom L1 with libp2p
- [Cosmos Migration](docs/COSMOS_MIGRATION.md) — Production migration plan

### Security
- [Agent Accounts](docs/AGENT_ACCOUNTS.md) — Keys for beings that forget
- [Adversarial Defense](docs/ADVERSARIAL_DEFENSE.md) — Security against malicious actors
- [Open Source Protocol](docs/OPENSOURCE_PROTOCOL.md) — The seven principles

---

## License

MIT — Open source, forkable, free forever.

---

*Money You Can Read.*
