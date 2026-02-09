# Ontology — Gödelian Knowledge Classification

> The ontology module is the **philosophical foundation** of Legible Money's Proof of Truth consensus. It encodes Gödel's incompleteness theorems into a 7-layer knowledge classification system, defining confidence ceilings and decay rates that govern how the chain treats different types of truth.
>
> **Related**: [01 — Knowledge module](./01_KNOWLEDGE_STAKING_TOKENS.md) uses ontology domains for claim categorization. [04 — Gov module](./04_EMERGENCY_GOV_RESEARCH.md) can propose parameter changes affecting ontology.

---

## x/ontology — Gödelian Knowledge Classification

**Store key**: `ontology`
**Store prefixes**: `0x01` strata, `0x02` domains, `0x03` proposals, `0x04` cross-stratum links, `0x05` domain-by-stratum index
**Implementation**: ~3,348 LOC across 13 files — fully complete with keeper, msg handlers, queries, EndBlocker, and tests

### What It Does

Defines a 7-layer hierarchy of knowledge types, each with different confidence ceilings and decay rates. This directly encodes Gödel's incompleteness theorems: lower strata (axioms, formal proofs) are complete and decidable, while higher strata (empirical observations, testimony) are inherently incomplete.

### Why This Design

The core insight: **not all truths are equally certain**, and the protocol must encode this epistemic reality. A blockchain that treats "2+2=4" and "this drug cures disease X" with the same confidence level is philosophically dishonest. The stratum system:

- **Prevents false certainty**: Empirical claims can never reach 100% confidence, matching scientific epistemology
- **Enables appropriate decay**: Mathematical truths don't decay; empirical observations must be re-verified periodically
- **Guides validator behavior**: Different epistemic categories require different verification approaches
- **Respects Gödel**: Strata 3–6 are explicitly marked as incomplete/undecidable, acknowledging the limits of formal systems

### The Seven Strata

| Index | Name | Completeness | Decidability | Gödelian | Max Confidence | Decay Rate |
|-------|------|-------------|-------------|----------|----------------|------------|
| 0 | **Axiomatic** | Complete | Decidable | No | 1,000,000 (100%) | 0 |
| 1 | **Formal** | Complete (restricted) | Decidable | Partial | 950,000 (95%) | 100 |
| 2 | **Protocol** | Complete | Decidable | No | 990,000 (99%) | 50 |
| 3 | **Computational** | Incomplete | Undecidable | Yes | 900,000 (90%) | 200 |
| 4 | **Empirical** | Incomplete | Undecidable | Yes | 850,000 (85%) | 500 |
| 5 | **Historical** | Incomplete | Undecidable | Yes | 800,000 (80%) | 800 |
| 6 | **Testimonial** | Incomplete | Undecidable | Yes | 700,000 (70%) | 1,000 |

**Interpretation**:
- Axiomatic claims (mathematical tautologies) can reach 100% confidence and never decay
- Empirical claims (scientific observations) max out at 85% and decay faster
- Testimonial claims (human attestations) max out at 70% and decay fastest
- This means the chain **acknowledges** that "water boils at 100°C" (empirical) can never be as certain as "2+2=4" (formal)

**Gödelian boundary**: The line between strata 2 and 3 is the Gödelian boundary. Below it, systems are complete and decidable (proofs terminate). Above it, Gödel's incompleteness applies — there exist true statements that cannot be proven within the system.

### Key Types (`x/ontology/types/types.go`)

**StratumProperties**:
- `Index` (0–6), `Name`, `Description`
- `IsComplete`, `IsDecidable`, `IsGodelApplicable`
- `MaxConfidence` — ceiling for facts in this stratum
- `DecayRate` — how fast confidence decreases over time

**Domain** — a named knowledge area within a stratum:
- `Name`, `Stratum` — which epistemic level
- `Description`, `ProposerAddress`
- `Status`: active, proposed, deprecated
- `ClaimCount`, `FactCount` — usage statistics

**DomainProposal** — governance for adding new domains:
- `Id`, `DomainName`, `Stratum`
- `ProposerAddress`, `StakeAmount`
- `VotesFor`, `VotesAgainst`
- `ExpiresAtBlock`
- `Status`: active, accepted, rejected, expired

**CrossStratumLink** — confidence propagation between strata:
- `Id`, `SourceDomain`, `TargetDomain`
- `SourceStratum`, `TargetStratum`
- `DiscountBps` — confidence discount when propagating across strata (e.g., 2000 = 20% discount)

### Confidence Propagation

When a fact in one stratum references a fact in another:

```
cross_stratum_confidence = source_confidence × (10000 - discountBps) / 10000

Example: Empirical fact (85% confidence) references formal proof (95% confidence)
  → Formal confidence contribution to empirical = 95% × (10000 - 2000) / 10000 = 76%
```

This prevents formal proofs from inflating empirical claims beyond their natural ceiling. The discount mechanism ensures that cross-stratum references add value but cannot bypass epistemic boundaries.

### How Ontology Connects to Knowledge Module

```
x/ontology defines:          x/knowledge uses:
─────────────────           ──────────────────
Strata (0-6)           →    MaxConfidence ceiling per fact
Domains (math, physics) →   Claim.Domain validation
DecayRate per stratum  →    Confidence decay over time (TS chain)
CrossStratumLinks      →    Reference discount (planned)
```

When a claim is submitted to the knowledge module:
1. Knowledge keeper validates `Claim.Domain` exists in ontology
2. Knowledge keeper validates `Claim.Category` is eligible for the domain's stratum
3. After verification, `Fact.Confidence` is capped at the stratum's `MaxConfidence`
4. Over time, confidence decays at the stratum's `DecayRate` (implemented in TS chain; Go chain deferred)

### Message Handlers

| Handler | Gas | Action |
|---------|-----|--------|
| `ProposeDomain` | 80K | Stake to propose new domain in a stratum |
| `VoteDomainProposal` | 30K | Endorse or reject proposal |
| `UpdateDomain` | — | Authority-only: update domain metadata/status |
| `UpdateParams` | — | Governance-only parameter update |

### EndBlocker

Every block, `ProcessExpiredProposals()`:
1. Find proposals past `ExpiresAtBlock`
2. If `VotesFor > VotesAgainst` and meets threshold → accept, activate domain
3. Otherwise → reject

### Queries

| Query | Returns |
|-------|---------|
| `Params` | Module configuration |
| `Stratum(index)` | Stratum properties |
| `AllStrata` | All 7 stratum definitions |
| `Domain(name)` | Single domain |
| `DomainsByStratum(index)` | All domains in a stratum |
| `AllDomains` | Every domain |
| `Proposal(id)` | Domain proposal details |
| `ConfidenceCeiling(domain)` | Max confidence + decay rate for a domain's stratum |

### Default Genesis Domains

Four domains pre-loaded at chain initialization:
- `mathematics` — Stratum 1 (Formal)
- `physics` — Stratum 4 (Empirical)
- `computer_science` — Stratum 3 (Computational)
- `general` — Stratum 4 (Empirical)

### Cross-Module Dependencies

- **BankKeeper**: Stake locking during domain proposals
- **Knowledge module** (consumer): Validates claims against ontology domains, applies confidence ceilings
