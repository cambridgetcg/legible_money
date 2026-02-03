# Cosmos SDK Migration Plan

> **From Custom TypeScript to Production Cosmos Chain**

---

## Overview

The custom TypeScript implementation serves as a prototype to validate Proof of Truth consensus. Once proven, we migrate to Cosmos SDK for production.

```
PHASE 1: TypeScript Prototype     PHASE 2: Cosmos SDK Production
┌─────────────────────────────┐   ┌─────────────────────────────┐
│  Custom Implementation      │   │  Cosmos SDK Implementation  │
│                             │   │                             │
│  • Rapid iteration          │──►│  • Battle-tested            │
│  • Prove PoT concept        │   │  • IBC interoperability     │
│  • TypeScript (matches SDK) │   │  • Production security      │
│  • libp2p networking        │   │  • Validator ecosystem      │
└─────────────────────────────┘   └─────────────────────────────┘
```

---

## Why Cosmos SDK?

### Benefits

1. **Proven Infrastructure**
   - 100+ chains running in production
   - Billions in value secured
   - Years of security hardening

2. **IBC Interoperability**
   - Native cross-chain communication
   - Asset transfers to/from other chains
   - Knowledge sharing between chains

3. **Validator Ecosystem**
   - Established validator operators
   - Tooling for node operation
   - Monitoring and alerting

4. **Modular Architecture**
   - Custom modules for knowledge layer
   - Swap consensus if needed
   - Upgrade without hard forks

### Trade-offs

| Aspect | TypeScript Custom | Cosmos SDK |
|--------|-------------------|------------|
| Language | TypeScript | Go |
| Iteration Speed | Very fast | Moderate |
| Security Maturity | Low | High |
| Ecosystem | Small | Large |
| Customization | Total | High |
| Maintenance | Self | Shared |

---

## Architecture Mapping

### Current TypeScript → Cosmos SDK

```
TypeScript                          Cosmos SDK
──────────────────────────────────────────────────────────────

core/crypto/                   →    SDK built-in + custom
  keys.ts                           (ed25519 native)
  signatures.ts                     (secp256k1 or ed25519)
  hashing.ts                        (sha256 native)
  merkle.ts                         (IAVL tree)

network/                       →    Tendermint/CometBFT
  libp2p implementation             (built-in P2P)

state/                         →    Cosmos SDK Store
  sparse_merkle.ts                  (IAVL + SMT module)
  mvcc/                             (built-in versioning)

consensus/                     →    Custom Consensus Module
  pot/                              (replace CometBFT or ABCI++)
  blocks/                           (built-in block handling)

protocol/                      →    Custom Cosmos Modules
  transactions/                     x/knowledge
                                    x/truth
                                    x/staking (extended)

knowledge/                     →    x/knowledge module
  axioms/                           KnowledgeKeeper
  facts/                            FactKeeper
  derivations/                      DerivationKeeper

economics/                     →    x/lgm module
  token/                            Extended x/bank
  staking/                          Extended x/staking
  billing/                          Custom billing module
```

### Module Structure (Cosmos SDK)

```
legible-money-cosmos/
├── app/
│   ├── app.go              # Application setup
│   ├── genesis.go          # Genesis handling
│   └── export.go           # State export
│
├── x/                       # Custom modules
│   ├── knowledge/           # Knowledge layer module
│   │   ├── keeper/
│   │   │   ├── keeper.go
│   │   │   ├── axioms.go
│   │   │   ├── facts.go
│   │   │   └── derivations.go
│   │   ├── types/
│   │   │   ├── axiom.go
│   │   │   ├── fact.go
│   │   │   ├── derivation.go
│   │   │   └── msgs.go
│   │   ├── genesis.go
│   │   └── module.go
│   │
│   ├── truth/               # Proof of Truth consensus
│   │   ├── keeper/
│   │   │   ├── keeper.go
│   │   │   ├── verification.go
│   │   │   ├── confidence.go
│   │   │   └── rewards.go
│   │   ├── types/
│   │   │   ├── claim.go
│   │   │   ├── verification.go
│   │   │   └── msgs.go
│   │   └── module.go
│   │
│   ├── logic/               # Logic axiom layer
│   │   ├── keeper/
│   │   │   ├── zones.go
│   │   │   ├── proofs.go
│   │   │   └── decidability.go
│   │   └── types/
│   │
│   └── billing/             # API billing
│       ├── keeper/
│       │   ├── metering.go
│       │   ├── channels.go
│       │   └── distribution.go
│       └── types/
│
├── proto/                   # Protobuf definitions
│   └── legible/
│       ├── knowledge/
│       ├── truth/
│       ├── logic/
│       └── billing/
│
└── scripts/
    ├── genesis.sh
    └── testnet.sh
```

---

## Migration Steps

### Phase 1: Preparation (While Running TypeScript)

```
WEEK 1-4: Setup
├── Initialize Cosmos SDK project
├── Define protobuf schemas
├── Create module scaffolding
└── Set up CI/CD for Go

WEEK 5-8: Core Modules
├── Implement x/knowledge module
│   ├── Axiom storage
│   ├── Fact storage
│   └── Query handlers
├── Implement x/logic module
│   ├── Zone management
│   └── Decidability checking
└── Write unit tests
```

### Phase 2: Consensus Integration

```
WEEK 9-12: Proof of Truth in Cosmos
├── Evaluate options:
│   ├── Option A: Custom ABCI++ implementation
│   ├── Option B: Modified CometBFT
│   └── Option C: Separate verification layer
├── Implement verification rounds
├── Integrate VRF for validator selection
└── Test consensus under various conditions
```

**Consensus Options:**

**Option A: ABCI++ (Application BlockChain Interface)**
```
Cosmos SDK allows custom consensus via ABCI++
- PrepareProposal: Select claims for verification
- ProcessProposal: Validate verification results
- ExtendVote: Include verification votes
- VerifyVoteExtension: Verify vote validity
```

**Option B: Modified CometBFT**
```
Fork CometBFT and modify:
- Block proposal includes verification round
- Validators perform PoT verification
- Block acceptance requires truth consensus
```

**Option C: Separate Layer**
```
Run PoT as separate verification network:
- Cosmos chain handles transactions
- PoT network handles verification
- Results committed to chain
```

### Phase 3: State Migration

```
WEEK 13-16: State Transfer
├── Export TypeScript chain state
├── Transform to Cosmos genesis format
├── Validate state integrity
├── Test with exported state
└── Document migration procedure
```

**State Migration Script:**
```go
// Transform TypeScript state to Cosmos genesis
func MigrateState(tsState TSState) GenesisState {
    genesis := GenesisState{}

    // Migrate accounts
    for _, acc := range tsState.Accounts {
        genesis.BankGenesis.Balances = append(
            genesis.BankGenesis.Balances,
            banktypes.Balance{
                Address: acc.Address,
                Coins:   sdk.NewCoins(sdk.NewCoin("ulgm", acc.Balance)),
            },
        )
    }

    // Migrate knowledge
    for _, axiom := range tsState.Axioms {
        genesis.KnowledgeGenesis.Axioms = append(
            genesis.KnowledgeGenesis.Axioms,
            knowledgetypes.Axiom{
                Id:         axiom.Id,
                Claim:      axiom.Claim,
                Confidence: axiom.Confidence,
                // ...
            },
        )
    }

    return genesis
}
```

### Phase 4: Testing

```
WEEK 17-20: Comprehensive Testing
├── Unit tests for all modules
├── Integration tests
├── Testnet deployment
├── Validator onboarding test
├── IBC transfer tests
└── Adversarial testing
```

### Phase 5: Launch

```
WEEK 21-24: Production Launch
├── Security audit
├── Genesis ceremony
├── Validator coordination
├── Mainnet launch
├── TypeScript chain sunset
└── SDK updates for Cosmos
```

---

## IBC Integration

### Cross-Chain Knowledge

```
┌─────────────────┐         IBC          ┌─────────────────┐
│  Legible Money  │◄───────────────────►│  Other Cosmos   │
│    (LGM)        │                      │    Chain        │
└────────┬────────┘                      └────────┬────────┘
         │                                        │
         │  Knowledge packets:                    │
         │  • Verified facts                      │
         │  • Confidence proofs                   │
         │  • Citation references                 │
         │                                        │
         └────────────────────────────────────────┘
```

### IBC Packet Types

```protobuf
// Knowledge transfer packet
message KnowledgePacket {
  string fact_id = 1;
  string claim = 2;
  string confidence = 3;  // decimal string
  repeated string references = 4;
  bytes proof = 5;  // Merkle proof
}

// Query request packet
message QueryPacket {
  string query = 1;
  string min_confidence = 2;
  string domain = 3;
}

// Query response packet
message QueryResponsePacket {
  repeated Fact facts = 1;
  string query_confidence = 2;
}
```

---

## SDK Compatibility

### TypeScript SDK Updates

The TypeScript SDK will need updates for Cosmos compatibility:

```typescript
// Before (custom chain)
const client = new LegibleClient({
  endpoint: 'http://localhost:9000',
  // libp2p direct connection
});

// After (Cosmos chain)
const client = new LegibleClient({
  endpoint: 'http://localhost:26657',  // Tendermint RPC
  chainId: 'legible-1',
  // Standard Cosmos connection
});

// SDK methods remain the same
const fact = await client.getFact('fact_123');
await client.submitClaim({ claim: '...', references: [...] });
```

### Breaking Changes

1. **Transaction Format**: Cosmos SDK message types
2. **Query Interface**: ABCI queries via Tendermint
3. **Event Format**: Cosmos event attributes
4. **Address Format**: Bech32 addresses

---

## Timeline

```
Month 1-6:   TypeScript prototype development
Month 7-9:   Cosmos SDK module development (parallel)
Month 10:    Testnet migration testing
Month 11:    Security audit
Month 12:    Mainnet launch on Cosmos

Total: ~12 months to production Cosmos chain
```

---

## Risk Mitigation

### Technical Risks

| Risk | Mitigation |
|------|------------|
| Consensus complexity | Start with ABCI++, iterate |
| State migration errors | Multiple test migrations |
| Performance regression | Benchmark early and often |
| IBC integration issues | Test with devnets first |

### Operational Risks

| Risk | Mitigation |
|------|------------|
| Validator coordination | Early engagement |
| Community confusion | Clear communication |
| SDK breaking changes | Compatibility layer |

---

## Resources

- [Cosmos SDK Documentation](https://docs.cosmos.network)
- [CometBFT Documentation](https://docs.cometbft.com)
- [IBC Protocol Specification](https://github.com/cosmos/ibc)
- [ABCI++ Specification](https://github.com/cometbft/cometbft/blob/main/spec/abci)

---

## See Also

- [BLOCKCHAIN_ARCHITECTURE.md](BLOCKCHAIN_ARCHITECTURE.md) - Current architecture
- [FOUNDATION_ARCHITECTURE.md](FOUNDATION_ARCHITECTURE.md) - System design
- [PROOF_OF_TRUTH.md](PROOF_OF_TRUTH.md) - Consensus mechanism
