# Disputes + Evidence Cluster

> These modules handle post-verification challenges. Disputes provides multi-tier arbitration when facts are contested. Evidence Management handles evidence quality auditing, oracle data feeds, and consensus scoring.
>
> **Related**: [01 — Knowledge](./01_KNOWLEDGE_STAKING_TOKENS.md) for how facts are created (disputes challenge these). [00 — Module Overview](./00_MODULE_OVERVIEW.md) for the `DisputeKnowledgeAdapter` and `DisputeStakingAdapter` that bridge these modules.

---

## x/disputes — Multi-Tier Fact Arbitration

**Store key**: `disputes`
**Store prefixes**: `0x01` disputes, `0x02` evidence, `0x03` votes, `0x04` fact index, `0x05` active index

### What It Does

When someone disagrees with a verified fact, they can open a dispute. Disputes use a tiered arbiter system — simple cases are resolved by 3 arbiters, but complex or contested cases can escalate to 7 or 15 arbiters with higher bond requirements.

### Escalation Tiers

| Tier | Name | Arbiters | Min Bond | Evidence Period | Voting Period | Quorum | Majority |
|------|------|----------|----------|-----------------|---------------|--------|----------|
| 0 | Basic | 3 | 111 LGM | 34,272 blocks (~1d) | 68,544 blocks (~2d) | 66% | 66% |
| 1 | Escalated | 7 | 555 LGM | 68,544 blocks (~2d) | 102,816 blocks (~3d) | 66% | 66% |
| 2 | Final | 15 | 1,111 LGM | 102,816 blocks (~3d) | 137,088 blocks (~4d) | 75% | 75% |

### Key Types (`x/disputes/types/types.go`)

**Dispute**:
- `Id`, `FactId`, `ClaimId` — what's being challenged
- `ChallengerAddress`, `DefenderAddress` — parties
- `BondAmount` — locked stake
- `Status`: open → evidence → voting → resolved/expired/escalated
- `EscalationTier` (0–2)
- `Outcome`: upheld (original stands), overturned (challenger wins), inconclusive
- `EvidenceDeadline`, `VotingDeadline` — phase deadlines
- `ArbiterAddresses[]` — selected arbiters

**DisputeEvidence**:
- `DisputeId`, `Submitter` (must be challenger or defender)
- `ContentHash`, `Description`, `EvidenceType`

**DisputeVote**:
- `DisputeId`, `ArbiterAddress`
- `Direction`: uphold, overturn, abstain
- `StakeWeight` — arbiter's effective stake at vote time

### Lifecycle

```
1. OPEN DISPUTE (MsgOpenDispute)
   ├─ Validate fact exists and is verified
   ├─ Lock challenger's bond (BankKeeper)
   ├─ Select arbiters from active validators (tier-appropriate count)
   ├─ Dispute.Status = "open" → "evidence"
   └─ Set evidence deadline

2. EVIDENCE PHASE
   ├─ Challenger submits evidence (MsgSubmitEvidence)
   ├─ Defender submits counter-evidence
   └─ Anyone can view submitted evidence

3. VOTING PHASE (after evidence deadline)
   ├─ Dispute transitions to "voting" via EndBlocker
   ├─ Arbiters cast votes: uphold, overturn, or abstain
   └─ Each vote weighted by arbiter's effective stake

4. RESOLUTION (EndBlocker or on final vote)
   ├─ Tally: sum stake-weighted votes per direction
   ├─ Check quorum (66% or 75% for Final tier)
   ├─ Check majority threshold
   ├─ Outcomes:
   │   ├─ UPHELD: original fact stands, challenger loses bond
   │   ├─ OVERTURNED: fact falsified, challenger gets bond back + bounty
   │   └─ INCONCLUSIVE: eligible for escalation
   └─ Update fact status via KnowledgeKeeper

5. ESCALATION (MsgEscalateDispute — if inconclusive)
   ├─ Must be within escalation window
   ├─ Additional bond required for higher tier
   ├─ New arbiters selected (larger pool)
   └─ Restart evidence → voting cycle at higher tier
```

### Message Handlers

| Handler | Gas | Action |
|---------|-----|--------|
| `OpenDispute` | 100K | Validate fact, lock bond, select arbiters, set deadlines |
| `SubmitEvidence` | 40K/50K | Validate phase, verify submitter is party, store evidence |
| `VoteDispute` | 30K | Validate arbiter, check voting phase, record stake-weighted vote |
| `EscalateDispute` | 100K | Check inconclusive, lock additional bond, new tier setup |

### Queries

| Query | Returns |
|-------|---------|
| `Dispute(id)` | Single dispute with full details |
| `DisputesByFact(factId)` | All disputes for a fact |
| `ActiveDisputes(limit, offset)` | Paginated open/evidence/voting disputes |
| `Evidence(disputeId)` | All evidence for a dispute |
| `Votes(disputeId)` | All arbiter votes for a dispute |
| `Params` | Module configuration |

### EndBlocker Logic

Every block, `ProcessExpiredDisputes()`:
1. Scan active disputes past their deadlines
2. Evidence phase expired → transition to voting (set voting deadline)
3. Voting phase expired → tally votes, determine outcome
4. Apply outcome: update fact status, distribute/slash bonds

### Cross-Module Dependencies

- **KnowledgeKeeper** (via `DisputeKnowledgeAdapter`): `GetFact()` to validate dispute target, `SetFact()` to update status on resolution, `GetClaim()` for claim reference
- **StakingKeeper** (via `DisputeStakingAdapter`): `GetActiveValidatorInfos()` for arbiter selection, `GetEffectiveStake()` for vote weighting
- **BankKeeper**: Bond locking, distribution, burning

---

## x/evidence_mgmt — Evidence Auditing & Oracles

**Store key**: `evid_mgmt` (renamed from "evidence" to avoid collision with SDK evidence module)
**Store prefixes**: `0x01` evidence, `0x02` audits, `0x03` oracles, `0x04` consensus, `0x05` evidence-by-claim, `0x06` audit-by-evidence, `0x07` active oracle index

### What It Does

Manages evidence quality through stake-weighted auditor consensus. Auditors score evidence reliability, and their assessments are aggregated using stake-weighted formulas. Also manages oracle registration for external data feeds.

### Key Types (`x/evidence_mgmt/types/types.go`)

**Evidence**:
- `Id`, `ClaimId`, `FactId` — what it supports
- `Submitter`, `Type` (on_chain, oracle_feed, attestation, computation, document, experiment)
- `ContentHash`, `Description`, `SourceURI`
- `ReliabilityScore` (0–1,000,000)
- `AuditCount` — number of auditor assessments
- `Status`: submitted → verified/rejected/expired

**EvidenceAudit** — auditor assessment:
- `AuditorAddress`, `StakeWeight`
- `Score` (0–1,000,000) — quality rating
- `Status`: pending → accurate/inaccurate/slashed/rewarded/expired

**AuditConsensus** — aggregated scoring:
- `WeightedScore` — stake-weighted average of audit scores
- `TotalStake` — sum of auditor stakes
- `AuditCount`
- `HighVariance` — flag if auditors disagree significantly

**Oracle** — external data provider:
- `OperatorAddress`, `StakeAmount`
- `Reputation` (0–1,000,000), `AccuracyRate`
- `DataTypes[]` — what kinds of data this oracle provides
- `Status`: active, inactive, slashed

**Params** (defaults):
- `MinAuditStake`: 100 LGM
- `MinOracleStake`: 1,000 LGM
- `AuditReward`: 10 LGM
- `AuditSlashRate`: 500 BPS (5%)
- `MinAuditsForConsensus`: 3
- `EvidenceExpiryBlocks`: 342,720 (~10 days)
- `OracleInactivityLimit`: 171,360 (~5 days)
- `HighVarianceThreshold`: 200,000 (20% disagreement)

### Audit Consensus Flow

```
1. SUBMIT EVIDENCE (MsgSubmitEvidence)
   ├─ User submits evidence for a claim/fact
   ├─ Assign deterministic ID = SHA256("evidence:" || submitter || claimId || height)
   └─ Status = "submitted"

2. AUDIT (MsgAuditEvidence) — multiple auditors
   ├─ Auditor stakes tokens, provides score (0-1M)
   ├─ Audit ID = SHA256("audit:" || auditor || evidenceId || height)
   ├─ Store audit, increment evidence.AuditCount
   └─ IF auditCount >= MinAuditsForConsensus (3):
      ├─ Compute weighted score: sum(score_i * stake_i) / sum(stake_i)
      ├─ Compute variance across auditor scores
      ├─ Flag high variance if disagreement > 20%
      └─ Update AuditConsensus record

3. CONSENSUS THRESHOLDS
   ├─ Score >= 700,000 (70%): evidence "verified"
   ├─ Score < 300,000 (30%): evidence "rejected"
   └─ In between: remains "submitted" (needs more audits)
```

### Oracle System

```
1. REGISTER ORACLE (MsgRegisterOracle)
   ├─ Stake minimum 1,000 LGM
   ├─ Declare supported data types
   └─ Oracle ID = SHA256("oracle:" || operator || height)

2. SUBMIT ORACLE DATA (MsgSubmitOracleData)
   ├─ Only registered oracle operators
   ├─ Creates evidence record with type = "oracle_feed"
   └─ Auto-credited with oracle's reputation as initial score

3. LIFECYCLE (EndBlocker)
   ├─ PruneInactiveOracles(): remove oracles inactive > 5 days
   └─ ExpireStaleEvidence(): mark evidence > 10 days as expired
```

### Message Handlers

| Handler | Gas | Action |
|---------|-----|--------|
| `SubmitEvidence` | — | Create evidence record with deterministic ID |
| `AuditEvidence` | — | Stake-backed score, update consensus if threshold met |
| `RegisterOracle` | — | Lock stake, register data types |
| `SubmitOracleData` | — | Oracle-only evidence submission |
| `UpdateParams` | — | Governance-only parameter update |

### Queries

| Query | Returns |
|-------|---------|
| `Evidence(id)` | Single evidence record |
| `EvidenceByClaimId(claimId)` | All evidence for a claim |
| `Audit(id)` | Single audit record |
| `AuditsForEvidence(evidenceId)` | All audits for evidence |
| `Oracle(id)` | Oracle details |
| `ActiveOracles` | All active oracles |
| `Consensus(evidenceId)` | Aggregated audit consensus |
| `Params` | Module configuration |

---

## How Disputes and Evidence Work Together

```
FACT CHALLENGED
     │
     ├─► x/disputes: MsgOpenDispute
     │   ├─ Select arbiters
     │   └─ Open evidence phase
     │
     ├─► x/evidence_mgmt: Evidence submitted for dispute
     │   ├─ Auditors score evidence quality
     │   └─ Oracle data may be submitted as supporting evidence
     │
     ├─► x/disputes: Arbiters review evidence + audit scores
     │   └─ Cast votes (uphold/overturn/abstain)
     │
     └─► Resolution
         ├─ UPHELD: x/knowledge fact status unchanged
         ├─ OVERTURNED: x/knowledge fact.Status = "falsified"
         └─ INCONCLUSIVE: escalate to higher tier
```

The evidence module provides **quality scoring** that informs arbiter decisions in disputes. Arbiters can reference audit consensus scores when deciding whether to uphold or overturn a challenged fact.
