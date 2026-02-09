# Emergency + Governance + Research Cluster

> These modules govern the chain's economic and operational controls. Emergency provides a BFT kill switch for crisis response. Governance manages protocol evolution through LIPs (Legible Improvement Proposals) with quadratic voting. Research funds and coordinates peer-reviewed knowledge production.
>
> **Related**: [01 — Staking](./01_KNOWLEDGE_STAKING_TOKENS.md) for Guardian tier requirements (emergency) and validator stake lookup (gov/research). [00 — Module Overview](./00_MODULE_OVERVIEW.md) for ABCI halt guard in PrepareProposal/ProcessProposal.

---

## x/emergency — BFT Kill Switch

**Store key**: `emergency`

### What It Does

A 3-phase protocol allowing Guardian-tier validators (Tier 3) to halt chain operations, optionally revert state, and resume. Designed for catastrophic scenarios: exploits, consensus failures, or critical bugs.

### Who Can Propose

- **Guardians** (Tier 3 validators): 11,111+ LGM staked, 1,111+ verifications, 77%+ accuracy, 0 slashes
- **Genesis Council** (H-5 feature): 20–50 honorary Guardians during bootstrap period with virtual stake

### The Three Phases

```
NORMAL ──► HALT CEREMONY ──► HALTED ──► REVERT CEREMONY ──► HALTED ──► RESUME CEREMONY ──► NORMAL
                                  │                                           ▲
                                  └───────────────────────────────────────────┘
                                            (skip revert, go straight to resume)
```

### Ceremony Mechanics

Each ceremony uses 2-phase BFT voting:

```
1. PROPOSAL
   ├─ Guardian submits proposal (halt/revert/resume)
   ├─ Anti-abuse checks:
   │   ├─ Max 3 proposals per epoch
   │   ├─ Max 1 per guardian per epoch
   │   └─ 111-block cooldown between proposals
   └─ Ceremony created in "prevote" phase

2. PREVOTE PHASE
   ├─ Guardians cast prevotes (support/oppose)
   ├─ Each vote weighted by guardian's effective stake
   └─ Progress checked every BeginBlock

3. PRECOMMIT PHASE (after prevote quorum)
   ├─ Guardians cast precommits (confirm/reject)
   ├─ Must have at least 4 distinct precommitting Guardians
   └─ Progress checked every BeginBlock

4. FINALIZATION
   ├─ HALT: requires 75% stake-weighted support
   ├─ REVERT: requires 80% stake-weighted support
   ├─ RESUME: requires 80% stake-weighted support
   └─ On success: execute action, log to audit trail
```

### Quorum Thresholds (basis points)

| Action | Quorum Required | Min Distinct Voters |
|--------|----------------|---------------------|
| Halt | 7,500 BPS (75%) | 4 |
| Revert | 8,000 BPS (80%) | 4 |
| Resume | 8,000 BPS (80%) | 4 |

### ABCI Halt Guard (`app/abci.go`)

When chain is halted:
- `PrepareProposal` filters out all non-emergency transactions
- `ProcessProposal` rejects proposals containing non-emergency transactions
- Only emergency module messages pass through

### Key Types (`x/emergency/types/types.go`)

**EmergencyCeremony**:
- `Id`, `Type` (halt/revert/resume), `Phase` (prevote/precommit/finalized/failed)
- `ProposerAddress`, `EvidenceHash`, `Reason`
- `TargetBlockHeight` (for revert), `StateRoot` (for resume)
- `Prevotes`, `Precommits` maps — address → vote
- `TotalPrevoteStake`, `TotalPrecommitStake`
- `CreatedAtHeight`, `FinalizedAtHeight`

**EmergencyAuditEntry** — full audit trail:
- `CeremonyId`, `Action`, `ActorAddress`
- `BlockHeight`, `Timestamp`
- Immutable record of all emergency actions

### Message Handlers

| Handler | Gas | Action |
|---------|-----|--------|
| `ProposeHalt` | 50K | Create halt ceremony (Guardian only, anti-abuse checks) |
| `VoteHalt` | 25K | Prevote or precommit on halt (Guardian only) |
| `ProposeRevert` | 50K | Create revert ceremony (chain must be halted) |
| `VoteRevert` | 25K | Prevote or precommit on revert |
| `ProposeResume` | 50K | Create resume ceremony (chain must be halted) |
| `VoteResume` | 25K | Prevote or precommit on resume |

### Queries

| Query | Returns |
|-------|---------|
| `Status` | Current emergency status + active halt ceremony ID |
| `ActiveCeremony` | Currently active ceremony (prevote/precommit phase) |
| `CompletedCeremonies(limit, offset)` | Finalized/failed ceremonies |
| `AuditLog(limit, offset)` | Full audit trail |

---

## x/gov (lgm_gov) — LIP Governance & Disbursements

**Store key**: `lgm_gov`

### What It Does

Governs protocol evolution through two mechanisms:
1. **LIPs** (Legible Improvement Proposals) — protocol change proposals with staged review and quadratic voting
2. **Disbursements** — treasury spending proposals with timelock protection

This is separate from the standard Cosmos SDK `x/gov` module (which handles basic parameter governance).

### LIP Lifecycle

```
DRAFT ──(stake to threshold)──► REVIEW ──(advance)──► LAST CALL ──(advance)──► VOTING ──(tally)──► ACCEPTED
  │                                                                                │                    or
  └──(withdraw)──► WITHDRAWN                                                       └──────────► REJECTED
```

### LIP Categories & Stake Thresholds

| Category | Min Stake to Advance | Scope |
|----------|---------------------|-------|
| Consensus | 1,000 LGM | Core protocol changes |
| Economics | 800 LGM | Token/reward modifications |
| Interface | 400 LGM | API/UX changes |
| Process | 200 LGM | Governance process changes |

### Quadratic Voting

Cost scales quadratically with number of votes:

```
cost = votes² × baseCostPerVote

Example (baseCostPerVote = 1 LGM):
  1 vote  =  1 LGM
  2 votes =  4 LGM
  5 votes = 25 LGM
 10 votes = 100 LGM (max practical)

Max: 100 votes per voter = 10,000 LGM
```

Directions: **support**, **oppose**, **abstain**

### Key Types (`x/gov/types/types.go`)

**LIP**:
- `Id` ("LIP-{counter}"), `Title`, `Abstract` (200 char max)
- `Category` (consensus/economics/interface/process)
- `Stage` (draft/review/last_call/voting/accepted/rejected/withdrawn)
- `Proposer`, `StakedAmount`, `StakeHolder`
- `VotingEndsAt` — block when voting period expires
- `SupportWeight`, `OpposeWeight`, `AbstainWeight` — tallied vote weights
- `UniqueVoters`, `QuorumMet`

**QuadraticVote**:
- `VoteId`, `Voter`, `ProposalId`
- `Direction`, `Votes` (count), `Cost` (actual LGM spent)

**DisbursementProposal**:
- `Id`, `Title`, `Description`
- `Category` (infrastructure/legal/domain/marketing/development/research/operations/emergency)
- `Proposer`, `RecipientAddr`, `Amount`
- `Stage` (proposed/voting/approved/executed/rejected/cancelled)
- `TimelockExpiry` — approvedAt + TimelockBlocks
- Same voting fields as LIP

**Params** (defaults):
- `VotingPeriodBlocks`: 100,000 (~70 hours)
- `ReviewPeriodBlocks`: 50,000 (~35 hours)
- `LastCallPeriodBlocks`: 50,000 (~35 hours)
- `BaseCostPerVote`: 1 LGM (10^18 ulgm)
- `MaxVotesPerVoter`: 100
- `SupportThresholdBps`: 7,500 (75%)
- `QuorumThresholdBps`: 1,500 (15%)
- `DisbursementStake`: 2,000 LGM
- `DisbursementTimelockBlocks`: 50,000 (~35 hours)
- `DisbursementSupportBps`: 8,000 (80%)
- `DisbursementQuorumBps`: 2,000 (20%)

### Message Handlers

| Handler | Gas | Action |
|---------|-----|--------|
| `SubmitLIP` | 100K | Create LIP in draft stage |
| `StakeLIP` | 50K | Stake tokens toward LIP advancement threshold |
| `CastVote` | 30K | Quadratic vote (dedup check, weight tally update) |
| `AdvanceLIPStage` | — | Move LIP: review → last_call → voting |
| `WithdrawLIP` | — | Cancel LIP (proposer only, non-terminal stage) |
| `SubmitDisbursement` | 80K | Create disbursement (goes straight to voting) |
| `ExecuteDisbursement` | 100K | Pay out after timelock expires |
| `CancelDisbursement` | — | Cancel (proposer only, non-terminal stage) |
| `UpdateParams` | — | Governance-only parameter update |

### BeginBlocker

Every block, checks expired voting periods:
1. Find LIPs in "voting" stage past `VotingEndsAt`
2. Tally: `supportBps = (supportWeight * 10000) / totalWeight`
3. If quorum met AND support >= threshold → `accepted`
4. Otherwise → `rejected`

### Queries

| Query | Returns |
|-------|---------|
| `LIP(lipId)` | Single LIP with all votes |
| `LIPs(stage?, limit, offset)` | Filter by stage or get all |
| `Disbursement(disbId)` | Single disbursement with votes |
| `Params` | Module configuration |

---

## x/research — Peer-Reviewed Research & Bounties

**Store key**: `research`

### What It Does

Manages three workflows:
1. **Research Submissions** — stake-backed research papers with peer review
2. **Research Bounties** — time-limited tasks with rewards for knowledge production
3. **Research Treasury** — funding pool for the research ecosystem

### Research Lifecycle

```
SUBMITTED ──(reviews accumulate)──► UNDER REVIEW ──(authority resolves)──► ACCEPTED (stake returned)
     │                                    │                                      or
     └──(challenged)──► CHALLENGED        └─────────────────────────────► REJECTED (stake slashed 33%)
```

### Bounty Lifecycle

```
OPEN ──(claimed before deadline)──► CLAIMED ──(authority fulfills)──► FULFILLED (reward paid)
  │
  └──(deadline passes)──► EXPIRED
```

### Key Types (`x/research/types/types.go`)

**Research**:
- `Id`, `Title`, `Abstract`, `ContentHash`
- `Submitter`, `Domain`
- `Type`: replication, fraud_investigation, methodology_audit, data_validation
- `StakeAmount`
- `Status`: submitted, under_review, accepted, rejected, challenged
- `ReviewCount`, `AggregateScore` (0–100, running average)

**PeerReview**:
- `Id`, `ResearchId`, `Reviewer`
- `Verdict`: approve, reject, revise
- `Score` (0–100), `Reasoning`

**Bounty**:
- `Id`, `Title`, `Description`, `Domain`
- `Creator`, `Reward`
- `Deadline` — block height (min 1 day in future)
- `Status`: open, claimed, fulfilled, expired
- `ClaimedBy`

**Params** (defaults):
- `MinResearchStake`: 100 LGM
- `MinChallengeStake`: 100 LGM
- `ReviewPeriod`: 68,544 blocks (~2 days)
- `MinReviewers`: 3
- `AcceptanceScoreThreshold`: 60 (out of 100)
- `RejectionSlashRate`: 3,300 BPS (33%)
- `MaxBountyReward`: 10,000 LGM
- `MinBountyDeadline`: 34,272 blocks (~1 day)

### Message Handlers

| Handler | Gas | Action |
|---------|-----|--------|
| `SubmitResearch` | 80K | Stake tokens, create submission |
| `ChallengeResearch` | 60K | Stake to challenge (submitted/under_review only) |
| `ReviewResearch` | — | Peer review (score + verdict), update aggregate |
| `ResolveResearch` | — | Authority-only: accept (return stake) or reject (slash 33%, burn) |
| `CreateBounty` | 80K | Lock reward, set deadline (min 1 day) |
| `ClaimBounty` | — | Claim open bounty before deadline |
| `FulfillBounty` | — | Authority-only: pay reward to claimer |
| `FundResearch` | — | Add coins to treasury |
| `UpdateParams` | — | Governance-only parameter update |

### BeginBlocker

Every block:
1. Find open bounties past deadline → mark as `expired`

### Queries

| Query | Returns |
|-------|---------|
| `Research(id)` | Single research with all peer reviews |
| `Submissions(status?, domain?)` | Filter by status and/or domain |
| `Bounty(id)` | Single bounty |
| `Params` | Module parameters + treasury balance |

### Peer Review Mechanics

```
1. Reviewer submits: verdict (approve/reject/revise) + score (0-100) + reasoning
2. Research.ReviewCount incremented
3. Research.AggregateScore updated (running average):
   newAvg = (oldAvg * (count-1) + newScore) / count
4. If this is the first review: status changes submitted → under_review
5. After MinReviewers (3) reached: authority can call ResolveResearch
6. Resolution:
   - AggregateScore >= AcceptanceThreshold (60): ACCEPTED → return full stake
   - AggregateScore < AcceptanceThreshold: REJECTED → slash 33% (burned), return 67%
```

---

## Cross-Module Relationships

```
x/emergency ←── x/staking (IsGuardian check, stake weighting)
    │
    │ Halt guard affects ALL modules via ABCI
    │
x/gov ←── x/staking (optional validator stake lookup)
    │  ←── x/bank (coin transfers for staking/disbursement)
    │
    │ LIP accepted → parameter changes via standard governance
    │ Disbursement approved → treasury payout
    │
x/research ←── x/bank (stake locking, reward distribution, burn on slash)
           ←── x/staking (optional validator stake lookup)
```

**No direct cross-module calls** between emergency/gov/research — they operate independently. However:
- Emergency halt **blocks** all non-emergency transactions including gov and research
- Gov LIPs could propose parameter changes that affect research module
- Research treasury could be funded through gov disbursements (not wired yet)
