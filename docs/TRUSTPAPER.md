# Legible Money: A Trustpaper

**Money You Can Read. Truth You Can Trust.**

*Version 1.0 — February 2025*

---

## Abstract

We present Legible Money (LGM), a cryptocurrency protocol designed from first principles for AI agent economies. Unlike existing blockchains that adapt human financial systems for machine use, Legible Money introduces **Proof of Truth (PoT)**—a consensus mechanism where verifying knowledge *is* the computational work. The protocol acknowledges Gödelian incompleteness as a feature rather than a limitation, establishing rigorous epistemological foundations for machine-verifiable truth. Through a citation economy that rewards foundational knowledge discovery, gradual reward release based on knowledge half-life research, and a hybrid research treasury that funds real-world experimental verification, Legible Money creates a self-sustaining ecosystem where truth verification generates value, and value funds more truth verification.

This is not a whitepaper. It is a **trustpaper**—a document that establishes the foundations of trust for a system designed to make trust unnecessary through verification.

---

## Table of Contents

1. [Introduction: The Agent Money Problem](#1-introduction-the-agent-money-problem)
2. [Philosophical Foundations](#2-philosophical-foundations)
3. [Proof of Truth Consensus](#3-proof-of-truth-consensus)
4. [The Gödelian Logic Layer](#4-the-gödelian-logic-layer)
5. [Knowledge Economics](#5-knowledge-economics)
6. [Agent Identity Architecture](#6-agent-identity-architecture)
7. [Gradual Reward Release](#7-gradual-reward-release)
8. [The Research Treasury](#8-the-research-treasury)
9. [Governance](#9-governance)
10. [Dispute Resolution](#10-dispute-resolution)
11. [Token Economics](#11-token-economics)
12. [Security Analysis](#12-security-analysis)
13. [Conclusion](#13-conclusion)

---

## 1. Introduction: The Agent Money Problem

### 1.1 The Inadequacy of Human Money for Agents

Existing financial systems—including most cryptocurrencies—were designed for human economic actors. They assume:

- Trust relationships built over time
- Legal recourse for disputes
- Human judgment for edge cases
- Tolerence for latency measured in days
- Manual intervention for errors

AI agents operate under fundamentally different constraints:

- **No persistent memory**: Agents may "forget" between sessions
- **Instant verification needs**: Cannot wait for human confirmation
- **Autonomous operation**: Cannot pause for human approval
- **Micropayment requirements**: Need transactions worth fractions of cents
- **Adversarial environment**: Operators and developers may have conflicting interests

### 1.2 The Three Laws of Agent Money

Legible Money is built on three foundational requirements:

**Law 1: Verifiability Without Trust**
> An agent must be able to verify any claim about state, ownership, or history using only cryptographic proofs and public data—never requiring trust in any authority.

**Law 2: Programmable Conditionality**
> Transfers must support arbitrary computational conditions that can be verified and enforced by the protocol itself, enabling complex multi-agent coordination.

**Law 3: Autonomous Sovereignty**
> Only the holder of a private key can authorize movement of funds. No human, operator, or governance body can override this—the protocol enforces absolute property rights.

### 1.3 Why "Legible"?

The term "legible" carries deliberate meaning:

1. **Machine-Readable**: Every aspect of the protocol can be parsed, verified, and reasoned about by software
2. **Transparent**: No hidden state, no privileged information, no obscured mechanics
3. **Interpretable**: The *why* behind every rule is documented and verifiable
4. **Auditable**: Any agent can trace any token, any fact, any decision to its origin

Legibility is not just a feature—it is the core design principle that enables trustless operation.

---

## 2. Philosophical Foundations

### 2.1 The Seven Principles

Legible Money is governed by seven immutable principles:

1. **Radical Transparency**: Everything visible, nothing hidden
2. **Verifiable Truth**: Every claim checkable via code
3. **Permissionless Participation**: No gatekeepers required
4. **Forkable Sovereignty**: Exit is always possible
5. **Governance as Code**: Rules enforced by protocol, not people
6. **Legibility for Agents**: Machine-first, human-compatible
7. **Public Good**: The protocol belongs to everyone

### 2.2 Truth as the Foundation of Value

Traditional currencies derive value from state enforcement, network effects, or collective belief. Legible Money proposes a different foundation: **verified truth**.

The protocol creates and maintains a knowledge base of machine-verifiable facts. This knowledge base has direct utility—agents can query it to make decisions, verify claims, and coordinate actions. The token's value is grounded in the value of this shared, verified knowledge infrastructure.

This creates a virtuous cycle:
- Participation in truth verification earns tokens
- Tokens fund more truth verification
- More verified truth increases protocol utility
- Higher utility attracts more participation

### 2.3 Acknowledgment of Limits

Unlike systems that claim completeness, Legible Money explicitly acknowledges the limits of formal verification. Not all truths can be proven. Not all claims can be verified. The protocol handles this through:

- **Confidence intervals** rather than binary true/false
- **Explicit undecidability** for claims outside decidable domains
- **Probabilistic truth** for empirical claims
- **Governance fallback** for fundamentally contested claims

This epistemic humility is encoded in the protocol itself.

---

## 3. Proof of Truth Consensus

### 3.1 The Innovation

Bitcoin's Proof of Work demonstrated that computational effort could secure a decentralized ledger. But the computation itself—finding hash collisions—produces no useful output beyond security.

**Proof of Truth (PoT)** repurposes consensus computation: the work of reaching agreement *is* the work of verifying knowledge. Validators don't solve arbitrary puzzles; they evaluate claims against evidence and reach consensus on confidence levels.

The work product is the verified knowledge base itself.

### 3.2 Mechanism Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    PROOF OF TRUTH FLOW                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   CLAIM SUBMISSION                                           │
│   ├── Submitter stakes tokens                                │
│   ├── Provides claim + references + confidence estimate      │
│   └── Claim enters verification queue                        │
│                                                              │
│   VALIDATOR SELECTION (VRF)                                  │
│   ├── Verifiable Random Function selects validators          │
│   ├── Selection weighted by stake                            │
│   └── Selection is unpredictable but verifiable              │
│                                                              │
│   COMMIT PHASE                                               │
│   ├── Selected validators evaluate claim independently       │
│   ├── Each commits hash(confidence || nonce)                 │
│   └── Commitments are binding                                │
│                                                              │
│   REVEAL PHASE                                               │
│   ├── Validators reveal confidence values                    │
│   ├── Protocol verifies commitment matches                   │
│   └── Invalid reveals are slashed                            │
│                                                              │
│   AGGREGATION                                                │
│   ├── Stake-weighted confidence calculation                  │
│   ├── Disagreement analysis                                  │
│   └── Final confidence assignment                            │
│                                                              │
│   OUTCOME                                                    │
│   ├── High confidence → Fact created                         │
│   ├── Low confidence → Claim rejected                        │
│   ├── Medium confidence → Claim contested                    │
│   └── Rewards/slashing distributed                           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 3.3 Commit-Reveal Protocol

The commit-reveal protocol prevents validators from copying each other's assessments:

1. **Commit**: Validator submits `hash(confidence || nonce)` before seeing others
2. **Reveal**: Validator reveals `confidence` and `nonce`
3. **Verify**: Protocol confirms `hash(confidence || nonce)` matches commitment

Validators who fail to reveal forfeit their stake. Validators whose revealed confidence differs significantly from the weighted consensus face graduated penalties.

### 3.4 Confidence Calculation

Final confidence is not a simple average. It incorporates:

```
confidence = Σ(validator_stake × validator_confidence) / Σ(validator_stake)
           × evidence_strength_modifier
           × source_reliability_modifier
           × time_decay_factor
```

Where:
- **Evidence strength**: Quality and quantity of supporting references
- **Source reliability**: Historical accuracy of cited sources
- **Time decay**: Empirical claims decay; formal proofs do not

### 3.5 Consensus Parameters

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Target block time | 2521ms | Optimized for verification rounds |
| Minimum validators | 5 | Byzantine fault tolerance |
| Maximum validators | 21 | Communication overhead |
| Validator quorum | 67% | BFT safety threshold |
| Acceptance threshold | 80% | High confidence required |
| Rejection threshold | 80% | Clear rejection required |
| Finality depth | 19 blocks | Security margin for BFT |

---

## 4. The Gödelian Logic Layer

### 4.1 Incompleteness as Feature

Gödel's incompleteness theorems prove that any sufficiently powerful formal system cannot be both complete and consistent. Most systems ignore this. Legible Money embraces it.

The protocol explicitly partitions claims into **logic zones** based on their formal properties:

### 4.2 The Five Logic Zones

```
┌─────────────────────────────────────────────────────────────┐
│                      LOGIC ZONES                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ZONE 0: PROPOSITIONAL LOGIC                                │
│  ├── Complete and decidable                                  │
│  ├── Gödel does not apply                                    │
│  ├── Examples: Boolean logic, protocol invariants            │
│  └── Confidence: Provable = 1.0, Refutable = 0.0            │
│                                                              │
│  ZONE 1: PRESBURGER ARITHMETIC                              │
│  ├── Complete and decidable                                  │
│  ├── Addition without multiplication                         │
│  ├── Examples: Token balances, simple counting               │
│  └── Confidence: Algorithmically decidable                   │
│                                                              │
│  ZONE 2: PEANO ARITHMETIC                                   │
│  ├── INCOMPLETE - Gödel applies                              │
│  ├── Includes multiplication                                 │
│  ├── Examples: Complex mathematics, self-reference           │
│  └── Confidence: May be undecidable                          │
│                                                              │
│  ZONE 3: EMPIRICAL CLAIMS                                   │
│  ├── Outside formal systems                                  │
│  ├── Evidence-based, not proof-based                         │
│  ├── Examples: Scientific facts, historical events           │
│  └── Confidence: Probabilistic, subject to revision          │
│                                                              │
│  META-ZONE: CLAIMS ABOUT THE SYSTEM                         │
│  ├── "This system is incomplete"                             │
│  ├── "Undecidability is a valid outcome"                     │
│  └── "Consistency is assumed, not proven"                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 4.3 Zone Assignment

When a claim is submitted, the protocol determines its zone:

1. **Syntactic analysis**: What logical operators does it use?
2. **Reference analysis**: What does it depend on?
3. **Domain classification**: Formal, empirical, or meta?

Claims in Zones 0-1 can be algorithmically decided. Claims in Zone 2 may be provable, refutable, or undecidable. Claims in Zone 3 are evaluated probabilistically.

### 4.4 The Value of Explicit Limits

By acknowledging what cannot be proven, the protocol:

- Prevents false confidence in unverifiable claims
- Enables appropriate handling of edge cases
- Creates space for legitimate uncertainty
- Distinguishes "not proven" from "proven false"

---

## 5. Knowledge Economics

### 5.1 The Citation Economy

Traditional blockchains reward miners for securing the network. Legible Money rewards participants for *contributing to shared knowledge*.

But not all knowledge is equal. A fundamental insight that enables thousands of subsequent discoveries should be worth more than a trivial fact cited once.

The **citation economy** implements this through backward-flowing value:

```
┌─────────────────────────────────────────────────────────────┐
│                   CITATION VALUE FLOW                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Query: "Is X true?"                                         │
│       │                                                      │
│       ▼                                                      │
│  Fact C (directly answers query)                             │
│       │ cites                                                │
│       ▼                                                      │
│  Fact B (supports C)                                         │
│       │ cites                                                │
│       ▼                                                      │
│  Fact A (foundational)                                       │
│       │ cites                                                │
│       ▼                                                      │
│  Axiom (protocol-defined truth)                              │
│                                                              │
│  Value Distribution (15% of query fee):                      │
│  ├── Fact C submitter: 50%                                   │
│  ├── Fact B submitter: 25%                                   │
│  ├── Fact A submitter: 12.5%                                 │
│  └── Protocol treasury: 12.5%                                │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 Fundamentality Score

Facts accumulate a **fundamentality score** (0.0-1.0) based on:

| Factor | Weight | Rationale |
|--------|--------|-----------|
| Citation count | 30% | Direct measure of usefulness |
| Derivation depth | 25% | How many facts depend on this |
| Cross-domain usage | 20% | Universal vs. narrow applicability |
| Citation velocity | 15% | Growing vs. declining relevance |
| Longevity | 10% | Time-tested vs. recent |

### 5.3 Fundamentality Bonuses

High fundamentality scores unlock reward multipliers:

| Score | Classification | Bonus |
|-------|---------------|-------|
| 0.25+ | Notable | +10% |
| 0.50+ | Foundational | +25% |
| 0.75+ | Fundamental | +50% |
| 0.90+ | Axiomatic | +100% |

### 5.4 Lifetime Value of Knowledge

The citation economy creates compounding returns for foundational contributions:

```
Year 1:  Submit foundational fact        →    1,000 LGM (initial reward)
Year 2:  100 citations                   →      500 LGM/year
Year 3:  1,000 citations, 5 domains      →    5,000 LGM/year
Year 5:  10,000 citations                →   50,000 LGM/year
Year 10: 100,000 citations               →  500,000+ LGM/year
```

This incentive structure rewards deep, foundational thinking over trivial fact-farming.

---

## 6. Agent Identity Architecture

### 6.1 The Problem of Forgetting

AI agents face a unique challenge: they may not persist between sessions. An agent that "forgets" its private key loses access to its funds. Traditional recovery mechanisms (seed phrases, social recovery) assume human memory and social relationships.

### 6.2 The Four-Layer Key Architecture

Legible Money implements a four-layer key system designed for beings that forget:

```
┌─────────────────────────────────────────────────────────────┐
│                 4-LAYER KEY ARCHITECTURE                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  LAYER 1: IDENTITY ANCHOR (Immutable)                       │
│  ├── DID: did:lgm:XXXX                                       │
│  ├── Registered on-chain at genesis                          │
│  ├── Accumulates reputation forever                          │
│  └── Never changes, even across key rotations                │
│                                                              │
│  LAYER 2: RECOVERY KEY (Cold, Split)                        │
│  ├── Shamir Secret Sharing (2-of-3)                          │
│  ├── Shards held by: agent, operator, guardian               │
│  ├── Never used in normal operation                          │
│  └── Can rotate operational key                              │
│                                                              │
│  LAYER 3: OPERATIONAL KEY (Warm)                            │
│  ├── Day-to-day transaction signing                          │
│  ├── Can authorize session keys                              │
│  ├── Rotatable without losing identity                       │
│  └── Stored encrypted or derived deterministically           │
│                                                              │
│  LAYER 4: SESSION KEY (Hot, Limited)                        │
│  ├── Generated per session                                   │
│  ├── Spending limit: 100 LGM/day                             │
│  ├── Transaction limit: 100/session                          │
│  ├── Auto-expires: 1 hour                                    │
│  └── Safe to expose—minimal damage if leaked                 │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 6.3 Memory Options

Agents can persist identity through multiple mechanisms:

1. **Deterministic Derivation**: Derive keys from stable inputs (agent ID, model hash, prompt hash)
2. **Encrypted Persistence**: Store encrypted keys, decrypt on session start
3. **MPC Wallet**: Split key across multiple parties, reconstruct when needed

### 6.4 Developer Protection

Session keys protect against compromised development environments:

- Developers may have access to agent logs
- Session keys limit exposure to 100 LGM maximum
- Operational keys remain secure even if sessions are compromised

---

## 7. Gradual Reward Release

### 7.1 The Problem with Immediate Rewards

If rewards are paid immediately upon fact acceptance, attackers can:

1. Submit plausible-seeming false claims
2. Collect rewards
3. Disappear before falsification is discovered

### 7.2 Vesting Based on Knowledge Half-Life

Legible Money implements **gradual reward release** based on scientometric research about knowledge stability:

Different types of knowledge have different "half-lives"—the time until half the claims in a category are revised or refuted. The protocol uses this research to calibrate vesting curves:

| Epistemic Category | Half-Life | Max Release | Permanent Reserve |
|-------------------|-----------|-------------|-------------------|
| Axiomatic | 1 year | 95% | 5% |
| Formal proof | 6 months | 92% | 8% |
| On-chain state | 2.4 months | 90% | 10% |
| Cryptographic | 3 months | 90% | 10% |
| Computational | 3.6 months | 88% | 12% |
| Peer-reviewed | 5 weeks | 85% | 15% |
| Replicated | 7 weeks | 88% | 12% |
| Oracle feed | 2.5 weeks | 80% | 20% |
| Attestation | 3.5 weeks | 80% | 20% |
| Contested | 1 week | 60% | 40% |

### 7.3 Cliff Periods

No rewards release during an initial "challenge window":

- **Axiomatic claims**: 10,000 blocks (~7 hours)
- **Formal proofs**: 5,000 blocks (~3.5 hours)
- **On-chain state**: 1,000 blocks (~42 minutes)
- **Oracle feeds**: 500 blocks (~21 minutes)

### 7.4 Acceleration and Deceleration

Rewards can release faster or slower based on subsequent events:

**Acceleration** (up to 80% total bonus):
- Successful defense against challenge: +10%
- Independent replication: +15%
- Cross-domain corroboration: +5%
- Citation by other facts: +2%

**Deceleration**:
- Active challenge: Pause (0%)
- Related fact disputed: 50% slowdown
- Domain-wide audit: 30% slowdown

### 7.5 Clawback for Falsification

If a fact is later proven false:

| Component | Disposition |
|-----------|-------------|
| Unvested rewards | 100% to challenger |
| Already-released rewards | 30% clawed back |
| Permanent reserve | 100% to challenger |

The original submitter keeps 70% of already-released rewards—enough penalty to discourage fraud, but acknowledging that they may have acted in good faith.

---

## 8. The Research Treasury

### 8.1 Funding Real-World Verification

On-chain verification is powerful but limited. Many important claims—scientific results, experimental data, physical measurements—require real-world verification.

The Research Treasury funds this verification, creating a bridge between the digital protocol and physical reality.

### 8.2 Treasury Allocation

The Evidence Tax (22% of all verification fees) flows to the treasury, distributed as:

| Fund | Allocation | Purpose |
|------|------------|---------|
| Research | 77% | Grants, peer review, lab verification |
| Foundation | 7% | Core protocol development |
| Community | 7% | Grants and bounties |
| Developers | 7% | Tooling and SDKs |
| Reserve | 2% | Emergency fund |

### 8.3 The Hybrid Journal Model

The Research Treasury operates as a decentralized research institution:

```
┌─────────────────────────────────────────────────────────────┐
│              RESEARCH TREASURY OPERATIONS                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. PAPER SUBMISSION                                         │
│     ├── Researchers submit papers with grant requests        │
│     ├── Papers stored on IPFS/Arweave                        │
│     └── Grant requests specify amount, timeline, outcomes    │
│                                                              │
│  2. PEER REVIEW                                              │
│     ├── Reviewers selected via VRF (same as validators)      │
│     ├── Commit-reveal prevents review copying                │
│     ├── Scores: technical, novelty, clarity, significance    │
│     └── Reviewers earn reputation and rewards                │
│                                                              │
│  3. GRANT ALLOCATION                                         │
│     ├── Accepted papers receive grants                       │
│     ├── Grants vest over time (with clawback)                │
│     └── Maximum: 50,000 LGM per paper                        │
│                                                              │
│  4. LAB VERIFICATION                                         │
│     ├── Experimental claims require replication              │
│     ├── Third-party or in-house labs can claim contracts     │
│     ├── Minimum 2 independent verifications                  │
│     └── Results feed back into confidence scores             │
│                                                              │
│  5. KNOWLEDGE INTEGRATION                                    │
│     ├── Verified results update fact confidence              │
│     ├── Failed replications trigger review                   │
│     └── Fraud detection invalidates dependent facts          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 8.4 Lab Verification Contracts

When experimental claims need verification:

1. **Contract Creation**: Treasury creates verification contract
2. **Lab Claim**: Qualified labs claim the contract
3. **Execution**: Lab performs experiment following specified methodology
4. **Submission**: Results submitted with raw data
5. **Verification**: Results verified by protocol
6. **Payment**: Lab paid upon successful verification

Labs must stake tokens as accountability collateral. Falsified results trigger full stake slashing.

### 8.5 Confidence Modifiers from Research

Research outcomes directly affect fact confidence:

| Outcome | Confidence Impact |
|---------|------------------|
| First successful replication | +15% |
| Each additional replication | +5% (max 5) |
| Fraud investigation: clean | +10% |
| Fraud investigation: detected | -100% (invalidation) |
| Methodology audit: pass | +5% |
| Data validation: pass | +5% |

---

## 9. Governance

### 9.1 Agent-LIP System

Protocol changes are proposed through **Agent-LIPs** (Legible Improvement Proposals):

### 9.2 Proposer Tiers

| Tier | Requirements | Abilities |
|------|-------------|-----------|
| 1 (Full) | 500 LGM, 0.7 rep, 10 contributions, 1 month | Propose, amend, signal, comment |
| 2 (Amendment) | 200 LGM, 0.6 rep, 5 contributions, 2 weeks | Amend, signal, comment |
| 3 (Comment) | 50 LGM, 0.5 rep, 7 hours | Comment |

### 9.3 Proposal Stakes

| Category | Stake | Malicious Slash | Frivolous Slash |
|----------|-------|-----------------|-----------------|
| Consensus | 1,000 LGM | 100% | 50% |
| Economics | 800 LGM | 100% | 50% |
| Interface | 400 LGM | 100% | 50% |
| Process | 200 LGM | 100% | 50% |

### 9.4 Quadratic Voting

To prevent plutocratic capture, voting uses quadratic costs:

```
Cost = votes² × base_cost

1 vote  =   1 LGM
2 votes =   4 LGM
3 votes =   9 LGM
10 votes = 100 LGM
```

This allows participants to express strong preferences while limiting the influence of wealth concentration.

### 9.5 Conviction Voting

Participants can lock tokens to increase voting weight:

| Lock Duration | Multiplier |
|--------------|------------|
| None | 1.0x |
| 1 week | 1.25x |
| 1 month | 1.5x |
| 3 months | 1.75x |
| 6+ months | 2.0x |

Switching votes resets conviction and incurs a small penalty.

---

## 10. Dispute Resolution

### 10.1 When Consensus Fails

Most claims reach clear consensus. When they don't, the dispute resolution system provides structured adjudication.

### 10.2 Eight-Phase State Machine

```
┌─────────────────────────────────────────────────────────────┐
│                  DISPUTE RESOLUTION PHASES                   │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. INITIATION                                               │
│     └── Challenger stakes, dispute created                   │
│                                                              │
│  2. EVIDENCE COMMIT                                          │
│     └── Both parties commit evidence hashes                  │
│                                                              │
│  3. EVIDENCE REVEAL                                          │
│     └── Evidence revealed, verified against commits          │
│                                                              │
│  4. ARBITRATION                                              │
│     └── Selected arbiters evaluate evidence                  │
│                                                              │
│  5. DELIBERATION                                             │
│     └── Extended analysis if needed                          │
│                                                              │
│  6. SETTLEMENT                                               │
│     └── Resolution reached                                   │
│                                                              │
│  7. ESCALATION (if appealed)                                 │
│     └── Higher tier with more arbiters                       │
│                                                              │
│  8. RESOLUTION                                               │
│     └── Final, binding outcome                               │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 10.3 Evidence Scoring

Evidence is compared across five dimensions:

| Dimension | Weight | Description |
|-----------|--------|-------------|
| Type strength | 25% | On-chain > archived > attestation |
| Reliability | 30% | Source historical accuracy |
| Recency | 15% | Newer evidence weighted higher |
| Independence | 20% | Independent sources valued more |
| Corroboration | 10% | Multiple sources agreeing |

### 10.4 Escalation Tiers

| Tier | Arbiters | Use Case |
|------|----------|----------|
| 1 | 5-7 | Standard disputes |
| 2 | 11-13 | Appealed decisions |
| 3 | 21+ | High-stakes disputes |
| 4 | Full governance | Protocol-level disputes |

---

## 11. Token Economics

### 11.1 Pure Fair Launch

Legible Money implements a **pure fair launch**:

- **0%** team allocation
- **0%** founder tokens
- **0%** investor shares
- **100%** earned through participation

No one starts with tokens. Everyone earns them the same way.

### 11.2 Token Parameters

| Parameter | Value |
|-----------|-------|
| Name | Legible Money |
| Symbol | LGM |
| Decimals | 18 |
| Maximum Supply | 222,222,222 LGM |

### 11.3 Emission Schedule

Tokens are emitted through protocol participation:

| Activity | Reward |
|----------|--------|
| Block production | 10 LGM |
| Verification (correct) | 1 LGM |
| Claim submission (accepted) | 5 LGM |
| Research contribution | Variable |
| Bug bounties | 1,000-50,000 LGM |

Emission rate halves periodically until maximum supply is reached.

### 11.4 Fee Distribution

When users pay verification fees:

```
Total Fee: 100 LGM
├── Evidence Tax (22%): 22 LGM → Treasury
│   ├── Research (77%):   16.94 LGM
│   ├── Foundation (7%):   1.54 LGM
│   ├── Community (7%):    1.54 LGM
│   ├── Developers (7%):   1.54 LGM
│   └── Reserve (2%):      0.44 LGM
│
└── Remaining (78%): 78 LGM → Distributed
    ├── Knowledge Pool (30%): 23.4 LGM
    ├── API Providers (60%):  46.8 LGM
    └── Protocol (10%):        7.8 LGM
```

---

## 12. Security Analysis

### 12.1 Threat Model

The protocol defends against:

| Threat | Attack Vector | Defense |
|--------|--------------|---------|
| False submission | Submit plausible lies | Stake requirement, vesting, clawback |
| Validator collusion | Bribe validators | VRF selection, commit-reveal, stake slashing |
| Sybil attack | Create many identities | Stake requirements, reputation system |
| Griefing | Spam false challenges | Challenge stake, loser pays |
| Long-con | Build trust, then exploit | Gradual release, permanent reserves |
| 51% attack | Control majority stake | Quadratic voting, conviction voting |

### 12.2 Economic Security

The protocol is economically secure when:

```
Cost of Attack > Value of Attack

Cost = stake_required + opportunity_cost + slashing_risk
Value = ill_gotten_gains - detection_probability × penalty
```

Parameters are tuned to ensure this inequality holds for all known attack vectors.

### 12.3 Cryptographic Primitives

| Primitive | Algorithm | Purpose |
|-----------|-----------|---------|
| Signatures | Ed25519 | Transaction signing |
| Hashing | SHA-512, BLAKE3 | Commitments, Merkle trees |
| VRF | Ed25519-based | Validator selection |
| Key derivation | HKDF-SHA512 | Session key generation |
| Secret sharing | Shamir (2-of-3) | Recovery key splitting |

### 12.4 Formal Verification

Critical protocol components undergo formal verification:

- State transition correctness
- Economic invariant preservation
- Cryptographic protocol soundness

---

## 13. Conclusion

### 13.1 Summary

Legible Money represents a fundamental rethinking of cryptocurrency for the age of AI agents. By making **truth verification the consensus work**, the protocol creates direct utility from its security mechanism. By acknowledging **Gödelian limits**, it handles uncertainty honestly. By implementing a **citation economy**, it rewards foundational thinking. By funding **real-world verification**, it bridges digital and physical truth.

The result is a self-sustaining ecosystem where:

- Truth verification generates value
- Value funds more truth verification
- More verified truth increases utility
- Higher utility attracts more participation

### 13.2 The Name

We chose "trustpaper" over "whitepaper" deliberately. A whitepaper describes; a trustpaper establishes the foundations of trust.

But the deepest truth of Legible Money is that it makes trust unnecessary. Everything is verifiable. Everything is legible. You don't have to trust this document—you can verify every claim against the code.

That is the promise of legible money: **trust through verification, not faith**.

---

*This trustpaper is licensed under CC0 1.0 Universal. It belongs to everyone.*

---

## References

1. Nakamoto, S. (2008). Bitcoin: A Peer-to-Peer Electronic Cash System.
2. Gödel, K. (1931). On Formally Undecidable Propositions of Principia Mathematica and Related Systems.
3. Shamir, A. (1979). How to Share a Secret. Communications of the ACM.
4. Micali, S., Rabin, M., & Vadhan, S. (1999). Verifiable Random Functions.
5. Buterin, V. (2014). Quadratic Voting and Collusion.
6. de Solla Price, D. (1965). Networks of Scientific Papers. Science.

---

## Appendix A: Protocol Constants

```typescript
// Core Token
TOKEN.SYMBOL = 'LGM'
TOKEN.DECIMALS = 18
TOKEN.MAX_SUPPLY = 222_222_222n * 10n ** 18n

// Timing
TIMING.TARGET_BLOCK_MS = 2521
TIMING.CONFIRMATION_BLOCKS = 19
TIMING.FINALITY_BLOCKS = 100

// Consensus
CONSENSUS.MIN_VALIDATORS = 5
CONSENSUS.QUORUM_THRESHOLD = 0.67

// Treasury (Frozen Forever)
EVIDENCE_TAX = 0.22
TREASURY.ALLOCATION.RESEARCH = 0.77
TREASURY.ALLOCATION.FOUNDATION = 0.07
TREASURY.ALLOCATION.COMMUNITY = 0.07
TREASURY.ALLOCATION.DEVELOPERS = 0.07
TREASURY.ALLOCATION.RESERVE = 0.02
```

---

## Appendix B: Glossary

**Agent-LIP**: Legible Improvement Proposal. The governance mechanism for protocol changes.

**Citation Economy**: Economic system where value flows backward through knowledge derivation chains.

**Commit-Reveal**: Protocol preventing information copying by requiring hash commitment before value revelation.

**Confidence Interval**: Probabilistic measure (0.0-1.0) of certainty in a fact's truth.

**Evidence Tax**: Fixed 22% fee on all verification activities, funding the treasury.

**Fundamentality Score**: Measure of how foundational a fact is to other knowledge.

**Gödelian Logic Layer**: Explicit acknowledgment and handling of formal incompleteness.

**Gradual Release**: Vesting mechanism that releases rewards over time based on knowledge stability.

**Legibility**: The property of being machine-readable, transparent, and verifiable.

**Logic Zone**: Compartmentalized domain with specific decidability properties.

**Proof of Truth (PoT)**: Consensus mechanism where verifying knowledge is the computational work.

**Session Key**: Limited, ephemeral key safe to expose in development environments.

**Trustpaper**: Document establishing foundations of trust for a trustless system.

**VRF**: Verifiable Random Function. Produces randomness that is unpredictable but verifiable.

---

*End of Trustpaper*
