# Documentation - Critical for Onboarding Progress

**Status**: 🔴 Incomplete
**Priority**: Critical
**Estimated Work**: 2-3 days
**Blocking**: Developer adoption, community contributions, open-source launch

---

## Overview

Documentation is the gateway to developer adoption. While internal docs (`.claude/`, `docs/`) are comprehensive for maintainers, public-facing documentation for new developers is incomplete. This blocks the open-source launch.

---

## Current State

### Existing Documentation
- [x] `README.md` - Basic project description (needs expansion)
- [x] `CONTRIBUTING.md` - Contribution guidelines
- [x] `docs/PROOF_OF_TRUTH.md` - Consensus explanation
- [x] `docs/BLOCKCHAIN_ARCHITECTURE.md` - Technical architecture
- [x] `.claude/CLAUDE.md` - Project intelligence (internal)
- [x] JSDoc comments in code (partial)

### Missing Documentation
- [ ] `QUICKSTART.md` - 5-minute getting started
- [ ] `ARCHITECTURE_OVERVIEW.md` - Visual system design
- [ ] `RUNNING_A_NODE.md` - Full node setup guide
- [ ] `DEVELOPER_GUIDE.md` - SDK/API usage guide
- [ ] `RPC_REFERENCE.md` - Complete RPC API docs
- [ ] `GLOSSARY.md` - Protocol terminology
- [ ] `FAQ.md` - Frequently asked questions
- [ ] `TROUBLESHOOTING.md` - Common issues and fixes
- [ ] `examples/README.md` - Example code index
- [ ] API reference (generated from JSDoc)

---

## Documentation Plan

### Phase 1: Getting Started (Day 1)

#### 1. QUICKSTART.md
**Purpose**: Get a developer from zero to running node in 5-10 minutes.

```markdown
# Quickstart

## Prerequisites
- Node.js 20+
- npm or pnpm

## Install
npm install -g legible-money

## Start a Local Node
lgm init --devnet
lgm start

## Your First Transaction
lgm balance                    # Check balance (devnet gives you 1000 LGM)
lgm transfer <address> 10      # Send 10 LGM

## Submit Your First Fact
lgm submit-claim "Water boils at 100°C at sea level" --domain chemistry

## Query the Knowledge Tree
lgm query chemistry

## Next Steps
- [Run a Validator](RUNNING_A_NODE.md)
- [Build with the SDK](DEVELOPER_GUIDE.md)
- [Explore the Architecture](ARCHITECTURE_OVERVIEW.md)
```

**Status**: [ ] Not written
**Owner**: TBD
**Review**: TBD

---

#### 2. ARCHITECTURE_OVERVIEW.md
**Purpose**: High-level visual overview for newcomers.

Sections:
1. **System Diagram** - ASCII art or Mermaid diagram showing layers
2. **Core Concepts** - One paragraph each:
   - Proof of Truth consensus
   - Knowledge tree structure
   - Agent identity system
   - Verification rounds
3. **Data Flow** - How a claim becomes a verified fact
4. **Key Components** - Brief description of each module
5. **Technology Stack** - TypeScript, libp2p, Ed25519, etc.

**Status**: [ ] Not written
**Owner**: TBD
**Review**: TBD

---

### Phase 2: Operational Guides (Day 1-2)

#### 3. RUNNING_A_NODE.md
**Purpose**: Complete guide to running a production node.

Sections:
1. **Hardware Requirements**
   - Minimum: 2 CPU, 4GB RAM, 50GB SSD
   - Recommended: 4 CPU, 8GB RAM, 100GB SSD
2. **Software Requirements**
   - Node.js 20+, npm/pnpm
   - Optional: Docker, systemd
3. **Installation Methods**
   - From npm (recommended)
   - From source
   - Using Docker
4. **Configuration**
   - Config file format
   - Environment variables
   - Network settings
5. **Running Modes**
   - Full node (default)
   - Validator node
   - Archive node
6. **Monitoring**
   - Built-in metrics endpoint
   - Prometheus integration
   - Log interpretation
7. **Maintenance**
   - Upgrading
   - Backup/restore
   - Key rotation
8. **Security**
   - Firewall rules
   - Key management
   - Rate limiting

**Status**: [ ] Not written
**Owner**: TBD
**Review**: TBD

---

#### 4. DEVELOPER_GUIDE.md
**Purpose**: How to build applications using the SDK.

Sections:
1. **Installation**
   ```bash
   npm install @legible-money/sdk
   ```
2. **Connecting to a Node**
   ```typescript
   import { LegibleClient } from '@legible-money/sdk';
   const client = new LegibleClient('http://localhost:8545');
   ```
3. **Wallet Management**
   - Creating a wallet
   - Importing existing wallet
   - Signing transactions
4. **Querying Data**
   - Account balances
   - Knowledge tree facts
   - Block/transaction data
5. **Submitting Transactions**
   - Transfers
   - Staking
   - Claims and verifications
6. **Subscriptions**
   - Block events
   - Transaction events
   - Fact updates
7. **Error Handling**
   - Common errors
   - Retry strategies
8. **Best Practices**
   - Gas estimation
   - Nonce management
   - Security considerations

**Status**: [ ] Not written
**Owner**: TBD
**Review**: TBD

---

### Phase 3: Reference Documentation (Day 2)

#### 5. RPC_REFERENCE.md
**Purpose**: Complete API reference for all RPC methods.

Format for each method:
```markdown
### lgm_getBalance

Get the balance of an account.

**Parameters**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| address | string | Yes | The DID address |

**Returns**
| Field | Type | Description |
|-------|------|-------------|
| balance | string | Balance in wei |
| nonce | number | Account nonce |

**Example**
Request:
{
  "jsonrpc": "2.0",
  "method": "lgm_getBalance",
  "params": ["did:lgm:abc123"],
  "id": 1
}

Response:
{
  "jsonrpc": "2.0",
  "result": { "balance": "1000000000000", "nonce": 5 },
  "id": 1
}

**Errors**
| Code | Message | Description |
|------|---------|-------------|
| -32602 | Invalid address | Address format invalid |
```

Methods to document:
- [ ] Chain methods (5)
- [ ] Account methods (4)
- [ ] Knowledge methods (8)
- [ ] Transaction methods (6)
- [ ] Validator methods (4)
- [ ] Subscription methods (3)

**Status**: [ ] Not written
**Owner**: TBD
**Review**: TBD

---

#### 6. GLOSSARY.md
**Purpose**: Define protocol-specific terminology.

Terms to include:
| Term | Definition |
|------|------------|
| **Proof of Truth (PoT)** | Consensus mechanism where verifying knowledge is the useful work |
| **Verification Round** | Commit-reveal process for validating claims |
| **Confidence** | 0-1 score representing certainty in a fact |
| **Stratum** | Ontological layer (fundamental → technological) |
| **DID** | Decentralized Identifier for agents |
| **Claim** | Unverified statement submitted for validation |
| **Fact** | Verified claim with confidence score |
| **Challenge** | Dispute against an existing fact |
| **Finality** | 19 blocks until transaction is irreversible |
| **Block Time** | 2.521 seconds between blocks |
| **LGM** | Legible Money token |
| **Agent** | Participant in the network (human or AI) |
| **Validator** | Node that participates in consensus |
| **Stake** | Tokens locked to become a validator |
| **Slash** | Penalty for validator misbehavior |
| **Domain** | Knowledge category (e.g., physics.mechanics) |

**Status**: [ ] Not written
**Owner**: TBD
**Review**: TBD

---

### Phase 4: Support Documentation (Day 2-3)

#### 7. FAQ.md
**Purpose**: Answer common questions upfront.

Categories:
1. **General**
   - What is Legible Money?
   - How is it different from other blockchains?
   - Is it production-ready?
2. **Technical**
   - What programming languages are supported?
   - How do I run a validator?
   - What are the hardware requirements?
3. **Economic**
   - How do I earn LGM?
   - What's the token supply?
   - How does staking work?
4. **Knowledge**
   - How do I submit a fact?
   - How is confidence calculated?
   - What happens if I'm wrong?

**Status**: [ ] Not written
**Owner**: TBD
**Review**: TBD

---

#### 8. TROUBLESHOOTING.md
**Purpose**: Help developers solve common problems.

Format:
```markdown
## Problem: Node won't start

### Symptoms
- Error: "EADDRINUSE: address already in use"
- Node exits immediately after starting

### Cause
Another process is using port 8545.

### Solution
1. Find the process: `lsof -i :8545`
2. Kill it: `kill -9 <PID>`
3. Or use a different port: `lgm start --port 8546`
```

Issues to document:
- [ ] Connection failures
- [ ] Sync issues
- [ ] Transaction failures
- [ ] Key/wallet problems
- [ ] Memory/disk issues
- [ ] Network configuration

**Status**: [ ] Not written
**Owner**: TBD
**Review**: TBD

---

### Phase 5: Examples (Day 3)

#### 9. examples/README.md
**Purpose**: Index and explain all example code.

```markdown
# Examples

## Basic
- [connect.ts](./connect.ts) - Connect to a node
- [wallet.ts](./wallet.ts) - Create and use a wallet
- [transfer.ts](./transfer.ts) - Send a transfer

## Knowledge
- [submit_claim.ts](./submit_claim.ts) - Submit a claim
- [verify_claim.ts](./verify_claim.ts) - Participate in verification
- [query_facts.ts](./query_facts.ts) - Query the knowledge tree

## Advanced
- [subscribe.ts](./subscribe.ts) - Subscribe to events
- [payment_channel.ts](./payment_channel.ts) - Payment channels
- [batch_operations.ts](./batch_operations.ts) - Batch transactions

## Running Examples
npm run example:connect
npm run example:transfer
```

**Status**: [ ] Not written
**Owner**: TBD
**Review**: TBD

---

## Files to Create

```
/
├── QUICKSTART.md                    # Getting started guide
├── docs/
│   ├── ARCHITECTURE_OVERVIEW.md     # System design overview
│   ├── RUNNING_A_NODE.md            # Node operation guide
│   ├── DEVELOPER_GUIDE.md           # SDK/API guide
│   ├── RPC_REFERENCE.md             # API reference
│   ├── GLOSSARY.md                  # Terminology
│   ├── FAQ.md                       # Common questions
│   └── TROUBLESHOOTING.md           # Problem solving
└── examples/
    ├── README.md                    # Examples index
    ├── connect.ts                   # Connection example
    ├── wallet.ts                    # Wallet example
    ├── transfer.ts                  # Transfer example
    ├── submit_claim.ts              # Claim submission
    ├── verify_claim.ts              # Verification
    ├── query_facts.ts               # Query example
    ├── subscribe.ts                 # Subscription example
    └── payment_channel.ts           # Payment channel example
```

---

## README.md Improvements

Current README is ~9KB. Should be ~20-30KB with:

- [ ] Better project description
- [ ] Feature highlights with badges
- [ ] Quick install instructions
- [ ] Architecture diagram
- [ ] Link to all documentation
- [ ] Contributing section
- [ ] License section
- [ ] Acknowledgments

---

## Documentation Tools

Consider adding:
- **TypeDoc** - Generate API docs from JSDoc
- **Docusaurus** - Documentation website
- **Mermaid** - Diagrams in markdown
- **shields.io** - Status badges

---

## Acceptance Criteria

1. New developer can start a node in < 10 minutes
2. All RPC methods documented with examples
3. Glossary covers all protocol terms
4. FAQ answers top 20 questions
5. Troubleshooting covers top 10 issues
6. All examples run successfully
7. Documentation passes spell check
8. Links are not broken

---

## Progress Log

| Date | Update |
|------|--------|
| 2026-02-05 | Progress tracking created |
| | |

---

## Notes

- Keep language simple, avoid jargon
- Use consistent terminology (refer to GLOSSARY.md)
- Include copy-pasteable code snippets
- Test all code examples before publishing
- Consider translations (future)
- Add "Edit this page" links to GitHub
