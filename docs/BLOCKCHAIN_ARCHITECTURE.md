# Blockchain Architecture

> **Custom L1 with libp2p → Cosmos SDK Migration Path**

---

## Overview

Legible Money uses a custom blockchain implementation with libp2p networking, designed for eventual migration to Cosmos SDK for production.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         LEGIBLE MONEY STACK                                  │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     APPLICATION LAYER                                │    │
│  │  SDK (TypeScript/Python/Rust) │ CLI │ RPC API │ Explorer            │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                   │                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     KNOWLEDGE LAYER                                  │    │
│  │  Axioms │ Facts │ Derivations │ Citations │ Logic Zones │ Queries   │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                   │                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     PROTOCOL LAYER                                   │    │
│  │  Transactions │ Blocks │ Validation │ Execution │ State Transitions │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                   │                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     CONSENSUS LAYER                                  │    │
│  │  Proof of Truth │ Validator Selection │ Finality │ Slashing         │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                   │                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     STATE LAYER                                      │    │
│  │  Sparse Merkle Tree │ MVCC │ Snapshots │ Persistence (LevelDB)      │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                   │                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     NETWORK LAYER                                    │    │
│  │  libp2p │ GossipSub │ Kademlia DHT │ mDNS │ Protocol Handlers       │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                   │                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     CRYPTOGRAPHIC LAYER                              │    │
│  │  Ed25519 │ SHA-512 │ BLAKE3 │ VRF │ Merkle Proofs │ Commitments     │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Layer Details

### 1. Cryptographic Layer

Foundation for all security properties.

```typescript
// core/crypto/

keys.ts          // Ed25519 key generation and management
signatures.ts    // Sign/verify with domain separation
hashing.ts       // SHA-512, BLAKE3, domain-separated hashing
vrf.ts           // Verifiable Random Function for validator selection
merkle.ts        // Sparse Merkle Tree for state proofs
commitment.ts    // Commit-reveal schemes for consensus
```

**Key Properties:**
- **Ed25519**: Fast, secure signatures (~15,000 verifications/sec)
- **SHA-512**: Primary hash, aligned with Ed25519
- **BLAKE3**: High-performance hashing for Merkle trees
- **VRF**: Unpredictable, verifiable validator selection

### 2. Network Layer

P2P communication using libp2p.

```typescript
// network/

node.ts              // libp2p node setup
config.ts            // Network configuration
discovery/
  bootstrap.ts       // Bootstrap peer discovery
  mdns.ts            // Local network discovery
pubsub/
  topics.ts          // GossipSub topic management
protocols/
  sync.ts            // Block synchronization
  consensus.ts       // Consensus message protocol
  claims.ts          // Claim submission protocol
  queries.ts         // Knowledge query protocol
```

**PubSub Topics:**
```
/lgm/blocks/1.0.0        - Block propagation
/lgm/transactions/1.0.0  - Transaction propagation
/lgm/claims/1.0.0        - Claim submissions
/lgm/verifications/1.0.0 - Verification votes
/lgm/consensus/1.0.0     - Consensus messages
```

### 3. State Layer

Authenticated state storage with efficient proofs.

```typescript
// state/

tree/
  sparse_merkle.ts   // Sparse Merkle Tree implementation
  proofs.ts          // Inclusion/exclusion proofs
mvcc/
  version.ts         // Version management
  snapshot.ts        // State snapshots for parallel execution
store/
  leveldb.ts         // Persistent storage adapter
  memory.ts          // In-memory storage for testing
schema/
  accounts.ts        // Account state (balances, nonces)
  knowledge.ts       // Knowledge state (facts, axioms)
  staking.ts         // Staking state (validators, delegations)
```

**State Structure:**
```
/accounts/{address}      → { balance, nonce, ... }
/knowledge/axioms/{id}   → Axiom
/knowledge/facts/{id}    → Fact
/knowledge/citations/... → Citation graph
/staking/validators/...  → Validator info
/logic/zones/...         → Logic zone state
```

### 4. Consensus Layer

Proof of Truth consensus mechanism.

```typescript
// consensus/

pot/
  engine.ts              // Consensus engine
  validator_selection.ts // VRF-based selection
  verification_round.ts  // Claim verification rounds
  confidence.ts          // Confidence aggregation
  finality.ts            // Finality conditions
blocks/
  producer.ts            // Block production
  validator.ts           // Block validation
mempool/
  pool.ts                // Transaction mempool
fork/
  choice.ts              // Fork choice rule
```

**Consensus Flow:**
```
1. Validators selected via VRF (stake-weighted)
2. Claims submitted with commit-reveal
3. Validators verify claims independently
4. Confidence aggregated (stake-weighted)
5. Block produced with verified facts
6. Finality after confirmation depth
```

### 5. Protocol Layer

Transaction processing and execution.

```typescript
// protocol/

transactions/
  transfer.ts    // Token transfers
  stake.ts       // Staking operations
  claim.ts       // Claim submissions
  verify.ts      // Verification votes
  derive.ts      // Logical derivations
  challenge.ts   // Fact challenges
validation/
  transaction.ts // Transaction validation
  block.ts       // Block validation
  state.ts       // State transition validation
execution/
  executor.ts    // Transaction executor
  context.ts     // Execution context
```

### 6. Knowledge Layer

The core value proposition - verified knowledge.

```typescript
// knowledge/

axioms/
  registry.ts        // Axiom management
  genesis.ts         // Genesis axioms
facts/
  submission.ts      // Fact submission flow
  verification.ts    // Verification process
derivations/
  engine.ts          // Derivation engine
  inference.ts       // Inference rules
logic/
  zones.ts           // Logic zone management
  proofs.ts          // Formal proofs
  goedel.ts          // Gödelian handling
citations/
  graph.ts           // Citation tracking
  rewards.ts         // Citation rewards
queries/
  search.ts          // Knowledge queries
```

### 7. Application Layer

User-facing interfaces.

```typescript
// sdk/typescript/      - TypeScript SDK
// sdk/python/          - Python SDK
// sdk/rust/            - Rust SDK
// node/cli.ts          - Command-line interface
// node/api/            - RPC APIs
// tools/explorer/      - Block explorer
```

---

## Block Structure

```typescript
interface Block {
  // Header
  number: bigint;
  timestamp: number;
  previousHash: Hash;
  stateRoot: Hash;
  knowledgeRoot: Hash;  // Merkle root of knowledge changes
  transactionsRoot: Hash;

  // Proposer
  proposer: PublicKey;
  proposerSignature: Signature;
  vrfProof: VRFProof;

  // Content
  transactions: Transaction[];
  knowledgeUpdates: KnowledgeUpdate[];

  // Consensus
  verificationResults: VerificationResult[];
  validatorSignatures: AggregatedSignature;
}
```

---

## Transaction Types

```
TRANSFER           - Token transfer
STAKE              - Stake tokens
UNSTAKE            - Unstake tokens
DELEGATE           - Delegate stake
CLAIM_SUBMIT       - Submit new claim
VERIFICATION_COMMIT - Commit verification vote
VERIFICATION_REVEAL - Reveal verification vote
DERIVE             - Submit derivation
CHALLENGE          - Challenge existing fact
GOVERNANCE_VOTE    - Vote on proposal
```

---

## Data Flow

### Claim Submission Flow

```
┌─────────┐    ┌─────────┐    ┌───────────┐    ┌─────────┐    ┌─────────┐
│  Agent  │───►│ Submit  │───►│ Mempool   │───►│ Select  │───►│ Commit  │
│         │    │ Claim   │    │           │    │ Valids  │    │ Phase   │
└─────────┘    └─────────┘    └───────────┘    └─────────┘    └────┬────┘
                                                                    │
┌─────────┐    ┌─────────┐    ┌───────────┐    ┌─────────┐    ┌────▼────┐
│ On-chain│◄───│ Include │◄───│ Aggregate │◄───│ Reveal  │◄───│ Reveal  │
│  Fact   │    │ in Block│    │ Confidence│    │ Votes   │    │ Phase   │
└─────────┘    └─────────┘    └───────────┘    └─────────┘    └─────────┘
```

### Block Production Flow

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ VRF Output  │───►│ Check if    │───►│ Build Block │
│ Generation  │    │ Selected    │    │ Proposal    │
└─────────────┘    └─────────────┘    └──────┬──────┘
                                             │
┌─────────────┐    ┌─────────────┐    ┌──────▼──────┐
│ Finalized   │◄───│ Collect     │◄───│ Broadcast   │
│ Block       │    │ Signatures  │    │ Proposal    │
└─────────────┘    └─────────────┘    └─────────────┘
```

---

## Configuration

### Node Configuration

```typescript
interface NodeConfig {
  // Identity
  privateKey?: string;
  dataDir: string;

  // Network
  listenAddresses: string[];
  bootstrapPeers: string[];
  maxPeers: number;

  // Consensus
  isValidator: boolean;
  validatorKey?: string;

  // Performance
  mempool: {
    maxSize: number;
    maxTxSize: number;
  };

  // APIs
  rpc: {
    enabled: boolean;
    port: number;
  };
  websocket: {
    enabled: boolean;
    port: number;
  };
}
```

### Genesis Configuration

```typescript
interface GenesisConfig {
  chainId: string;
  timestamp: number;

  // Initial state
  initialAccounts: Array<{
    address: string;
    balance: bigint;
  }>;

  // Initial validators
  initialValidators: Array<{
    publicKey: string;
    stake: bigint;
  }>;

  // Genesis axioms
  axioms: LogicalAxiom[];

  // Protocol parameters
  params: {
    blockTime: number;
    minStake: bigint;
    slashingRate: number;
    // ...
  };
}
```

---

## Performance Targets

| Metric | Target | Notes |
|--------|--------|-------|
| Block Time | 2.5 seconds | 2521ms |
| Throughput | 1,000 TPS | Transactions per second |
| Finality | 10 blocks | ~25 seconds |
| State Proof | < 1 KB | Sparse Merkle proofs |
| Signature Verify | 15,000/sec | Ed25519 batch verification |
| Knowledge Verify | 100/sec | Fact verifications per second |

---

## Security Model

### Threat Model

1. **Byzantine Validators**: Up to 1/3 malicious
2. **Network Partitions**: Temporary isolation handled
3. **Sybil Attacks**: Stake-weighted participation
4. **Eclipse Attacks**: Diverse peer connections
5. **Long-Range Attacks**: Weak subjectivity checkpoints

### Cryptographic Security

- **Ed25519**: 128-bit security level
- **SHA-512**: 256-bit collision resistance
- **VRF**: Unpredictable validator selection
- **Merkle Proofs**: Succinct state verification

---

## Development Phases

### Phase 1: Core (Weeks 1-12)
- [x] Cryptographic primitives
- [x] Logic axiom layer
- [x] Network layer (libp2p)
- [ ] State management
- [ ] Basic consensus

### Phase 2: Protocol (Weeks 13-24)
- [ ] Transaction types
- [ ] Block production
- [ ] Knowledge layer
- [ ] Staking system

### Phase 3: Devnet (Weeks 25-32)
- [ ] Multi-node testing
- [ ] SDK completion
- [ ] Explorer
- [ ] Documentation

### Phase 4: Testnet (Weeks 33-40)
- [ ] Public testnet
- [ ] Security audit
- [ ] Performance tuning
- [ ] Bug bounty

### Phase 5: Cosmos Migration (Future)
- [ ] Port to Cosmos SDK
- [ ] IBC integration
- [ ] Mainnet launch

---

## See Also

- [PROOF_OF_TRUTH.md](PROOF_OF_TRUTH.md) - Consensus mechanism
- [FOUNDATION_ARCHITECTURE.md](FOUNDATION_ARCHITECTURE.md) - System design
- [COSMOS_MIGRATION.md](COSMOS_MIGRATION.md) - Migration plan
- [LOGIC_AXIOM_LAYER.md](LOGIC_AXIOM_LAYER.md) - Formal logic system
