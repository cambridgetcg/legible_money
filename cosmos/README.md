# Cosmos SDK Migration

This directory contains the architecture and types for migrating Legible Money to Cosmos SDK.

## Migration Strategy

### Phase 1: Module Structure
Define the Cosmos SDK module structure that maps to existing components.

### Phase 2: State Migration
Convert Sparse Merkle Tree state to IAVL tree format.

### Phase 3: Message Types
Map existing transactions to Cosmos SDK message types.

### Phase 4: Keeper Implementation
Implement module keepers with existing business logic.

### Phase 5: IBC Integration
Enable cross-chain knowledge sharing via IBC.

## Directory Structure

```
cosmos/
├── README.md           # This file
├── types/              # Cosmos-compatible type definitions
│   ├── module.ts       # Module interface types
│   ├── messages.ts     # Msg types for transactions
│   ├── queries.ts      # Query types
│   └── genesis.ts      # Genesis state types
├── keeper/             # Keeper interface definitions
│   ├── account.ts      # Account keeper
│   ├── knowledge.ts    # Knowledge keeper
│   └── consensus.ts    # Consensus keeper
├── ibc/                # IBC integration types
│   ├── packets.ts      # IBC packet definitions
│   └── middleware.ts   # IBC middleware hooks
└── migration/          # Migration utilities
    ├── state.ts        # State migration tools
    └── genesis.ts      # Genesis export/import
```

## Module Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    COSMOS SDK ARCHITECTURE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   x/lgm      │  │  x/knowledge │  │  x/research  │          │
│  │   (Core)     │  │  (Facts)     │  │  (Bounties)  │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                 │                 │                    │
│         ▼                 ▼                 ▼                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Keeper Layer                          │   │
│  │  - AccountKeeper (balances, staking)                    │   │
│  │  - KnowledgeKeeper (facts, verification)                │   │
│  │  - ResearchKeeper (bounties, funding)                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│         │                 │                 │                    │
│         ▼                 ▼                 ▼                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Store Layer (IAVL)                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## IBC Integration

Legible Money can share knowledge across chains via IBC:

```
Chain A (LGM)          IBC              Chain B (Consumer)
┌──────────────┐    ┌───────┐    ┌──────────────────────┐
│              │    │       │    │                      │
│  Knowledge   │───▶│Packet │───▶│  Knowledge Oracle    │
│  Base        │    │       │    │  (read-only facts)   │
│              │◀───│ Ack   │◀───│                      │
└──────────────┘    └───────┘    └──────────────────────┘
```

## PP Alignment

The Cosmos migration maintains PP principles:

- **TRUTH**: IAVL tree provides cryptographic verification
- **UNDERSTANDING**: Clear module boundaries
- **BEAUTY**: Elegant SDK integration
- **JUSTICE**: IBC enables fair cross-chain access
- **CREATIVITY**: Novel knowledge-sharing protocol
