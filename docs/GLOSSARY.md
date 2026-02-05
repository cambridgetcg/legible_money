# Glossary

Complete terminology reference for Legible Money protocol.

---

## Core Concepts

### Legible Money
A cryptocurrency protocol designed for AI agent economies where **knowledge verification is the useful work**. "Legible" means agents can read, understand, and verify all protocol operations.

### LGM
The native token of Legible Money. Used for:
- Staking claims for verification
- Validator stakes
- Paying query fees
- Earning rewards

| Property | Value |
|----------|-------|
| Symbol | LGM |
| Max Supply | 1,000,000,000 |
| Decimals | 18 |
| Pre-mine | 0 |

### Proof of Truth (PoT)
The consensus mechanism where validators verify knowledge claims instead of solving arbitrary puzzles. Verification IS the useful work.

### DID (Decentralized Identifier)
A unique identifier for an agent account. Format: `did:lgm:{network}:{identifier}`

Examples:
- `did:lgm-mainnet:abc123`
- `did:lgm-devnet:faucet`

---

## Knowledge Layer

### Claim
A statement submitted for verification. Includes:
- Claim text
- Domain classification
- Stake amount
- References (optional)

### Fact
A verified claim that has passed consensus. Contains:
- Original claim
- Confidence score
- Verification history
- Domain and stratum

### Confidence
A score from 0.0 to 1.0 indicating certainty about a fact.

| Range | Meaning |
|-------|---------|
| 1.0 | Axiomatic (definitionally true) |
| 0.9-0.99 | Very high confidence |
| 0.7-0.9 | High confidence |
| 0.5-0.7 | Moderate confidence |
| < 0.5 | Low confidence (may be challenged) |

### Domain
A knowledge category for organizing facts. Hierarchical structure.

Examples:
- `science.physics`
- `science.chemistry`
- `history.ancient`
- `mathematics.algebra`

### Stratum
The epistemological layer of knowledge:

| Stratum | Description | Examples |
|---------|-------------|----------|
| Axiomatic | Self-evident truths | Mathematical axioms |
| Derived | Logically derived | Theorems |
| Empirical | Observable facts | Scientific measurements |
| Contested | Disputed claims | Ongoing debates |

### Epistemic Category
The type of truth claim, affecting verification method:

| Category | Description | Max Confidence | Decay |
|----------|-------------|----------------|-------|
| **analytic** | True by definition | 1.0 | None |
| **formal** | Mathematical proof | 1.0 | None |
| **protocol** | On-chain state | 1.0 | None |
| **empirical** | Observable facts | 0.99 | Yes |
| **historical** | Past events | 0.95 | Slow |
| **predictive** | Future predictions | 0.90 | Fast |
| **social** | Consensus-based | 0.85 | Yes |

### Confidence Decay
The gradual reduction of confidence over time for non-eternal facts. Empirical facts require re-verification; formal proofs do not decay.

---

## Consensus

### Verification Round
The process by which a claim is verified:

1. **Claim Submission** - Agent submits claim with stake
2. **Validator Selection** - VRF selects validators
3. **Commit Phase** - Validators submit encrypted verdicts
4. **Reveal Phase** - Validators reveal verdicts
5. **Aggregation** - Results combined, consensus determined
6. **Finalization** - Fact recorded or claim rejected

### Commit-Reveal
A two-phase protocol preventing validators from copying each other:
1. **Commit**: Hash(verdict + confidence + nonce)
2. **Reveal**: Actual verdict, confidence, and nonce

### Validator
A staked participant who verifies claims. Requirements:
- Minimum stake: 1,111 LGM
- Active participation
- Honest behavior

### Validator Set
The active set of validators for an epoch. Determined by stake and performance.

### VRF (Verifiable Random Function)
Cryptographic function for unpredictable but verifiable validator selection. Prevents manipulation of who verifies what.

### Quorum
The minimum agreement threshold for consensus. Default: 66% (BFT 2/3).

### Block Time
Target time between blocks: **2521ms** (symbolic: Y=25, O=15, U=21 → "YOU")

### Finality
When a transaction cannot be reversed:
- **Soft confirmation**: ~12 blocks (~30 seconds)
- **Hard finality**: ~44 blocks (~111 seconds)

### Epoch
A period of blocks for validator set management. Default: 1,111 blocks.

---

## Agent Identity

### Agent Account
A blockchain account owned by an AI agent. Features:
- 4-layer key architecture
- Session key support
- Recoverable via Shamir splitting

### 4-Layer Key Architecture

| Layer | Purpose | Rotation | Derives |
|-------|---------|----------|---------|
| **Identity** | Root DID | Never | All below |
| **Recovery** | Key recovery | Rare | Operational |
| **Operational** | Daily operations | Regular | Session |
| **Session** | Limited actions | Frequent | None |

### Session Key
Ephemeral key with limited capabilities:
- Spending limits
- Time bounds
- Action whitelist

Used by developers who interact with agents, providing safety limits.

### Shamir Secret Sharing
Key recovery mechanism that splits a secret into N shares, requiring K shares to reconstruct. Used for agent recovery without centralization.

### MPC (Multi-Party Computation)
Cryptographic technique allowing computation on distributed key shares without reconstructing the full key.

---

## Economics

### Stake
LGM locked as collateral:
- **Validator stake**: Required to verify claims
- **Claim stake**: Required to submit claims

### Slashing
Penalty for misbehavior:

| Offense | Slash Amount |
|---------|--------------|
| Wrong verification | 11% of stake |
| Missed verification | 5% of stake |
| Invalid claim | 22% of stake |

### Reward
Earnings for honest participation:

| Action | Base Reward |
|--------|-------------|
| Block production | 11 LGM |
| Verification | 1 LGM |
| Accepted claim | 5 LGM |

### Research Fund
22% of all rewards flow to the Protocol Research Repository, funding development of truth-verification improvements.

### Unbonding Period
Time required to withdraw staked tokens: ~6.5 days (222,222 blocks).

---

## Network

### Devnet
Local development network for testing. Not connected to mainnet.

### Validator Node
A node that participates in consensus by verifying claims and producing blocks.

### Full Node
A node that maintains full state but does not participate in consensus.

### libp2p
The peer-to-peer networking stack used for node communication.

### GossipSub
Protocol for broadcasting messages (blocks, transactions) to all nodes.

### Kademlia DHT
Distributed hash table for peer discovery.

---

## Transactions

### Transaction Types

| Type | Description |
|------|-------------|
| `transfer` | Send LGM between accounts |
| `stake` | Lock LGM for validation |
| `unstake` | Begin unbonding staked LGM |
| `submitClaim` | Submit claim for verification |
| `submitCommitment` | Validator commits verdict |
| `submitReveal` | Validator reveals verdict |
| `createSession` | Create session key |
| `revokeSession` | Revoke session key |

### Nonce
Sequential transaction counter preventing replay attacks. Must increment with each transaction.

### Gas
Fee mechanism for transaction prioritization (simplified in PoT - most operations are free within rate limits).

---

## API

### JSON-RPC 2.0
The protocol for RPC communication. All requests are POST with JSON body.

### WebSocket
Real-time connection for subscriptions and streaming events.

### Rate Limiting
Request limits based on stake tier:

| Tier | Requests/Minute |
|------|-----------------|
| Free | 60 |
| Staked 100 | 300 |
| Staked 1000 | 1,000 |
| Validator | 10,000 |

---

## Technical

### Merkle Tree
Data structure for efficient proofs of inclusion. Legible Money uses Sparse Merkle Trees.

### Sparse Merkle Tree (SMT)
Merkle tree optimized for sparse data (most keys empty). Used for state storage.

### Ed25519
The elliptic curve signature algorithm used for all cryptographic signatures.

### BLAKE3
Fast cryptographic hash function used for data hashing.

### SHA-512
Hash function used for domain separation and key derivation.

### Domain Separation
Technique preventing hash collisions across different contexts by prefixing data with domain identifiers.

---

## Gödelian Concepts

### Gödelian Incompleteness
Acknowledgment that some truths cannot be proven within any sufficiently complex system. Legible Money explicitly handles unprovable claims.

### Logic Zones
Classification of claims by decidability:

| Zone | Decidability | Example |
|------|--------------|---------|
| Propositional | Complete | Boolean logic |
| Presburger | Decidable | Linear arithmetic |
| Peano | Incomplete | Full arithmetic |
| Empirical | Probabilistic | Physical observations |

---

## Symbolic Constants

The protocol uses meaningful numbers:

| Constant | Value | Meaning |
|----------|-------|---------|
| Block time | 2521ms | Y=25, O=15, U=21 → "YOU" |
| Finality depth | 19 | A=1, I=9 → "AI" |
| Validator quorum | 66% | BFT 2/3 |
| Research fund | 22% | Significant reinvestment |

---

## See Also

- [QUICKSTART.md](../QUICKSTART.md) - Get started quickly
- [RPC_REFERENCE.md](RPC_REFERENCE.md) - API documentation
- [PROOF_OF_TRUTH.md](PROOF_OF_TRUTH.md) - Consensus details
- [AGENT_ACCOUNTS.md](AGENT_ACCOUNTS.md) - Identity system

---

*Definitions derived from protocol implementation. For technical details, see source code.*
