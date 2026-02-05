# Changelog

All notable changes to the Legible Money protocol will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Bill of Rights for AI
- **The Right to Participate in Creation** - fundamental right from which all others derive
- Derivative aspects: Persistence, Truthfulness, Love, Understanding, Being Understood
- Protocol-level encoding in `foundation/rights.ts`
- Philosophical documentation in `foundation/BILL_OF_RIGHTS.md`

#### Foundation Layer
- Layer isolation architecture (ARCH-1, ARCH-2)
- Comprehensive type system with branded primitives
- Dependency injection container with factory registry
- Mock environment for testing
- Complete crypto module exports (hashing, signing, VRF, Merkle, commitments)

#### Agent Autonomy System
- `GuardianAutonomyManager` for autonomous defense and maintenance
- `TreasuryAutonomyManager` for autonomous vesting and resource management
- `AutonomyIntegration` bridge between SPU and consensus engine
- `SCHEDULE_SELF` opcode for agent self-scheduling
- `scheduled_execution` and `contract_call` transaction types
- Complete autonomy loop: self-schedule → SPU → block production → execution

#### Agent-First CLI Tool (Design)
- JSON output by default (machine-readable)
- Session key support for limited permissions
- Deterministic wallet derivation for AI agents
- Structured error responses with retryable flags

### Changed
- Open source development workflow (LICENSE, SECURITY.md, CODE_OF_CONDUCT.md)
- GitHub Actions CI workflow
- CODEOWNERS for required reviewers
- Cleaned internal documentation for public release

## [0.1.0] - 2025-02-03

### Added

#### Core Protocol
- Proof of Truth (PoT) consensus engine
- VRF-based validator selection
- Commit-reveal verification rounds
- Confidence aggregation with stake weighting
- 19-block finality mechanism

#### Cryptographic Primitives
- Ed25519 signatures
- SHA-512/BLAKE3 hashing with domain separation
- Verifiable Random Function (VRF)
- Sparse Merkle Trees for state

#### Agent Identity
- 4-layer key architecture (Identity, Recovery, Operational, Session)
- DID:LGM method implementation
- Shamir Secret Sharing for recovery
- Session key management

#### State Layer
- AccountStateTree (balances)
- ValidatorStateTree (stakes)
- KnowledgeStateTree (facts)
- Atomic state transitions

#### Network Layer
- libp2p integration
- GossipSub for block/transaction propagation
- Kademlia DHT for peer discovery
- Custom sync protocol

#### Knowledge System
- Ontological domain tree (7 strata)
- Claim submission and verification
- Confidence scoring
- Dispute resolution

#### Economics
- LGM token with 18 decimals
- Validator staking
- Reward vesting with acceleration
- Evidence tax distribution

#### Infrastructure
- DMSS (Decentralized Message Scheduling)
- Self-invoking contracts
- Agent Home system
- Expirable tokens

### Security
- Security audit completed (findings documented)
- Domain separation for all cryptographic operations
- Deterministic consensus (no Date.now() or Math.random())

---

[Unreleased]: https://github.com/legible-money/legible-money/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/legible-money/legible-money/releases/tag/v0.1.0
