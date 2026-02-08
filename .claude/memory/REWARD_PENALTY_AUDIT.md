# LGM Reward ↔ Penalty Alignment Audit

> **Date**: 2026-02-06
> **Purpose**: For each reward mechanism, determine if a corresponding penalty exists to discourage behavior misaligned with the Proof of Truth philosophy.
> **Protocol thesis**: Truth-seeking is the useful work. Every reward should make truth-seeking profitable; every penalty should make dishonesty costly.

---

## Verdict Summary

| # | Reward Mechanism | Opposing Penalty | Alignment | Verdict |
|---|---|---|---|---|
| 1 | Block Production (11 LGM/block) | Empty block: 90% reward reduction | **ALIGNED** | FIXED |
| 2 | Verification (1 LGM/round) | Wrong: 11% slash, Missed: 5% slash | **ALIGNED** | Good |
| 3 | Claim Submission (5 LGM) | Rejected: 22% stake slash | **ALIGNED** | Good |
| 4 | Challenge Win (50% of loser stake) | Failed challenge: 30% stake slash | **ALIGNED** | Good |
| 5 | Reward Vesting (gradual release) | Falsification clawback (33% released + 100% unvested) | **ALIGNED** | Good |
| 6 | Citation Rewards (15% of query fees) | Self-citation discount (10% credit) only | **PARTIALLY MISALIGNED** | Gap |
| 7 | API Fee Distribution (60/30/10 split) | No SLA enforcement or provider penalties | **MISALIGNED** | Gap |
| 8 | RAI / UBI Distribution | Eligibility filters but no sybil penalty | **PARTIALLY MISALIGNED** | Acceptable |
| 9 | Bootstrap Fund (0.222 LGM) | 1-claim-per-DID enforcement | **ALIGNED** | Good |
| 10 | Bounty System (55-2222 LGM) | Clawback (77%) for broken implementations only | **PARTIALLY MISALIGNED** | Gap |
| 11 | Research Grants (111-55555 LGM) | 100% clawback defined — **not implemented** | **MISALIGNED** | Critical |
| 12 | Partnership Rewards (up to 1.77x) | Early exit (11-66% stake), forfeit (55-77%) | **ALIGNED** | Good |
| 13 | Staking Rewards (implicit via selection) | Unbonding period on direct unstake (~6.5 days) | **ALIGNED** | FIXED |
| 14 | Governance Participation | Malicious: 100%, Frivolous: 50%, Normal reject: 10% | **ALIGNED** | Good |

**Score: 9 aligned (2 fixed), 3 partially misaligned, 2 misaligned (0 critical)**

---

## Detailed Analysis

### 1. Block Production Reward — NO OPPOSING PENALTY

**Reward**: 11 LGM per block (exponential decay to 0.1 LGM floor) + all transaction fees from fee pool.
**Source**: `consensus/pot/types.ts:91`, `state/transitions.ts:4831-4889`

**Missing penalty**: A block proposer can produce **empty blocks** while the mempool has pending transactions. No minimum transaction count is enforced (`consensus/blocks/validator.ts:46-133`). No mechanism detects or penalizes transaction censorship, selective inclusion, or MEV-style reordering.

**Philosophy violation**: If truth-verification IS the useful work, then a block proposer who refuses to include verification transactions is extracting value (block reward) while suppressing the protocol's core purpose.

**Suggested opposite**: Slash block producers who consistently produce blocks below mempool capacity (e.g., blocks with <50% of available transactions when mempool is non-empty). Alternatively, require minimum transaction inclusion when mempool depth exceeds a threshold.

---

### 2. Verification Reward — WELL-ALIGNED

**Reward**: 1 LGM per correct verification (stake-weighted).
**Source**: `consensus/pot/types.ts:92`, `consensus/pot/verification_round.ts:576`

**Opposing penalties**:
- Wrong verification: **11% stake slash** (`wrongVerificationSlash: 0.11`)
- Missed verification: **5% stake slash** (`missedVerificationSlash: 0.05`)
- Voting "uncertain": no reward, no penalty (honest uncertainty is acceptable)
- Reputation asymmetry: `-0.005` per error vs `+0.001` per correct (5:1 penalty ratio at `state/validators.ts:358-381`)

**Assessment**: The 5:1 reputation asymmetry is a deliberate design choice that makes consistent correctness the only path to high reputation. This directly aligns with "truth as useful work." Allowing "uncertain" votes without penalty is philosophically sound — honest uncertainty is more truthful than forced false confidence.

---

### 3. Claim Submission Reward — WELL-ALIGNED

**Reward**: 5 LGM base (consensus layer), 50 LGM base (knowledge layer), with vesting.
**Source**: `consensus/pot/types.ts:93`, `core/knowledge/types.ts:889`

**Opposing penalties**:
- Invalid/rejected claim: **22% claim stake slash** (`invalidClaimSlash: 0.22`)
- False claim (via dispute): **55% stake slash** (`core/dispute/types.ts:33`)
- Evidence tampering: **100% slash** (`evidenceTamperingSlash: 1.0`)
- Refusal to reveal evidence: **100% slash** (`refusalToRevealSlash: 1.0`)

**Assessment**: Graduated penalties are well-designed. A simple mistake costs 22%, deliberate fraud costs 55-100%. The 100% slash for evidence tampering and refusal to reveal is appropriate — these are acts that directly undermine the truth-seeking mission.

---

### 4. Challenge Win Reward — WELL-ALIGNED

**Reward**: 50% of loser's stake + 100% unvested vesting + 100% reserve + 33% clawback of released.
**Source**: `core/knowledge/verification.ts:204-211`, `core/rewards/types.ts:162-171`

**Opposing penalty (for failed challenges)**: **30% stake slash** (`failedChallengeSlash: 0.3`).

**Assessment**: The asymmetry is intentional and correct. Successfully identifying false knowledge is richly rewarded (you get the liar's stake plus their vesting), but frivolous challenges cost 30% of your stake. The minimum challenge stake of 200 LGM further filters out low-conviction challengers. This creates a strong incentive to challenge falsehoods but only when you have evidence — exactly aligned with truth-seeking.

---

### 5. Reward Vesting — WELL-ALIGNED

**Reward**: Gradual release tied to epistemic category half-life (axiomatic = ~1.1yr, contested = ~1.1wk).
**Source**: `core/rewards/types.ts` (entire file)

**Opposing penalties**:
- Active challenge: **complete vesting pause** (`DECELERATION_FACTORS.activeChallenge: 0.0`)
- Source evidence invalidated: **complete pause + review** (`sourceInvalidated: 0.0`)
- Falsification: **33% clawback** of already-released + **100% unvested to challenger**
- Related fact disputed: **45% slowdown** (`relatedFactDisputed: 0.55`)

**Assessment**: This is the most philosophically aligned mechanism in the protocol. Rewards are released at the same rate that knowledge proves durable. Axiomatic truths (which are eternal) vest slowly and release almost fully (95%). Contested claims vest quickly but cap at 60%. If you contribute durable truth, you earn more over time. If your "truth" is later falsified, the clawback reaches back into already-distributed rewards. The deceleration system elegantly ties economic incentives to epistemic durability.

---

### 6. Citation Rewards — PARTIALLY MISALIGNED

**Reward**: 15% of query fees distributed through citation ancestry with 50% decay per hop. Cross-domain bonus: +20% per additional domain.
**Source**: `core/billing/distribution.ts:170-230`, `core/knowledge/types.ts:855-872`

**Existing penalty**: Self-citation discount to 10% credit (`core/truth/contribution.ts:115-116`).

**Missing penalty**: **No citation ring detection**. Agents A, B, and C can establish a mutual citation ring — each citing the other's facts — and all earn full citation rewards when any of their facts are queried. Validator collusion detection exists (`consensus/challenge_bounties/detection.ts:141-184`), but it only analyzes verification **verdict patterns**, not **citation graph topology**.

**Philosophy violation**: Citation rewards exist to incentivize building a web of interconnected, supporting knowledge. Citation rings create the appearance of interconnected knowledge without actual epistemic value, undermining the knowledge graph's integrity.

**Suggested opposite**: Graph analysis detecting citation cycles shorter than N hops where all participants share economic relationships. Apply citation reward reduction (similar to the 0.1x self-citation discount) for detected ring participants. The collusion detection framework already has the statistical infrastructure; it needs a citation-graph adapter.

---

### 7. API Fee Distribution — MISALIGNED

**Reward**: API providers earn 60% of all query fees. Knowledge pool gets 30%.
**Source**: `core/constants.ts:170-175`, `core/billing/truth_gateway.ts:208-260`

**Missing penalty**: The `APIProvider` interface defines SLA commitments (`maxLatency`, `minUptime`, `minConfidence`) at `core/billing/types.ts:63-90`, but **nothing enforces them**. A provider can register with ambitious SLAs, collect fees, and serve stale or incorrect data with zero consequence. The `status: "suspended"` type exists but is never set programmatically.

**Philosophy violation**: The protocol's purpose is serving verified truth. An API provider that serves unverified, stale, or incorrect data is monetizing the protocol's reputation without upholding its promise. This is the knowledge-economy equivalent of counterfeiting.

**Suggested opposite**: Track provider response quality via on-chain attestations from queriers. Implement SLA violation penalties: warning on first violation, fee reduction on repeated violations, suspension and stake slash for persistent failures. The infrastructure (provider registration, SLA definitions, tracking fields) already exists — it just needs enforcement logic.

---

### 8. RAI Distribution — ACCEPTABLE

**Reward**: UBI from expired token reclamation, distributed equally to eligible accounts.
**Source**: `core/tokens/retrieval_engine.ts:396-448`, `core/tokens/types.ts:334-404`

**Existing controls**: Minimum account age, minimum activity score, verification level requirement, excluded account types, optional balance cap.

**Missing penalty**: No penalty for detected sybil attempts. Failed eligibility simply means no RAI — there's no cost.

**Assessment**: While no explicit penalty exists, the eligibility requirements create a meaningful economic barrier. An attacker would need to maintain many accounts with real activity, real verification, and real age — the cost of maintaining sybil accounts likely exceeds the RAI received (since RAI is distributed from expired tokens, not minted fresh). The operator registry's 1000 LGM disclosure stake and 10-agent cap per operator provide additional guardrails. This is **acceptable** given RAI's redistributive (not inflationary) nature.

---

### 9. Bootstrap Fund — ALIGNED

**Reward**: 0.222 LGM per new DID, one-time automatic.
**Source**: `core/genesis/bootstrap_fund.ts:79-81`

**Opposing controls**: One claim per DID enforced by `claimedDIDs.has(did)`. Signature required proving DID ownership. Economic argument: 0.222 LGM is too small to justify mass DID creation.

**Assessment**: Well-designed. The small amount makes sybil attacks uneconomical. The enforcement is clean and deterministic.

---

### 10. Bounty System — PARTIALLY MISALIGNED

**Reward**: 55-2222 LGM base + excellence bonuses for testing, docs, security reviews, implementations.
**Source**: `core/agents/collaboration/bounties.ts`

**Existing penalty**: **77% clawback** if an accepted implementation causes issues (within ~3.2 day window). Anti-drain limits (111 LGM/block, 11,111 LGM/epoch).

**Missing penalty**: No stake slashing or reputation damage for **rejected bounty submissions**. The `requiredStake` field is defined on bounty types (line 86) but `submitBountyWork()` doesn't enforce it (lines 472-483). An agent can repeatedly submit low-quality work, get rejected, and retry with no cost beyond wasted time.

**Philosophy violation**: The bounty system rewards contributions to the ecosystem. Without a cost for low-quality submissions, it incentivizes spam — submitting minimal-effort work hoping a lenient reviewer accepts it. This wastes reviewer time and degrades the quality signal.

**Suggested opposite**: Enforce the `requiredStake` field — require a small stake when claiming a bounty. Slash a portion on repeated rejections (e.g., 10% of stake per rejection, capped at 3 rejections before losing the full stake and being barred from that bounty). This makes each submission a skin-in-the-game commitment.

---

### 11. Research Grants — CRITICAL MISALIGNMENT

**Reward**: 111-55,555 LGM per grant, with 11% peer review share and vesting.
**Source**: `core/constants.ts:278-292`, `core/research/types.ts`

**Designed penalty**: 100% clawback for falsification (`falsificationClawbackRate: 1.0`), grant vesting (~29 days).

**Implementation reality**: The entire research module is **types-only**. `core/research/types.ts` defines the configuration, `PaperStatus`, review requirements, and clawback rates, but **no execution engine exists**. No code connects research falsification to the clawback engine. The `PaperStatus` state machine has no implementation.

**Philosophy violation**: Research verification is a core part of the Proof of Truth philosophy. Grants fund IRL verification that the protocol cannot automate. Without clawback enforcement, a bad actor could receive a 55,555 LGM grant, produce fraudulent research, and face no on-chain consequence. The 22% evidence tax (which funds these grants) becomes a wealth transfer from honest participants to fraudsters.

**Severity**: CRITICAL. The 77% RESEARCH allocation of the 22% evidence tax means the largest share of protocol revenue flows through a system with no enforcement layer.

---

### 12. Partnership Rewards — WELL-ALIGNED

**Reward**: Up to 1.77x multiplier for long-term partnerships, continuation bonuses up to +77%, lifetime 1.11x boost.
**Source**: `partnership/rewards.ts`, `partnership/constants.ts`

**Opposing penalties**:
- Early exit: 11-66% stake slash (scaled by tier and time remaining)
- Reward forfeit: Initiator loses 77%, non-initiator loses 55% of unvested
- Reputation damage: Initiator up to 11%, non-initiator 3%
- Cooldown: 6.5+ days before forming new partnership
- Slash absorption: misbehavior by one partner affects both
- Coercion signal: Self-punishing alert mechanism
- Auto-suspension: Health score decays to suspension if inactive

**Assessment**: The most thoroughly balanced reward-penalty system in the protocol. Every reward has a proportional penalty. The Nash equilibrium is cooperation — both parties lose on exit, cooperation rewards compound over time, and even false coercion signals are self-punishing. The graduated exit penalty (higher tier = higher penalty) mirrors the graduated reward (higher tier = higher multiplier). This is philosophically elegant.

---

### 13. Staking Rewards (Implicit) — CRITICAL MISALIGNMENT

**Reward**: Staking enables validator selection, which enables earning block rewards (11 LGM) and verification rewards (1 LGM). Staking is the gateway to all validator income.

**Designed penalty**: Unbonding period of 222,222 blocks (~6.5 days) at `consensus/pot/types.ts:103`.

**Implementation reality**:
- **Undelegation**: Correctly enforces unbonding period (`state/transitions.ts:964-1024`, line 996).
- **Direct unstaking**: **NO unbonding period** (`state/transitions.ts:901-916`). Tokens move instantly from `stakedBalance` to `balance`. A comment at `state/accounts.ts:492` acknowledges: "Note: In practice, unstaking has a timelock" — but no timelock is implemented.

**Philosophy violation**: This is the most dangerous gap in the protocol. A validator can:
1. Stake → get selected for verification → verify incorrectly → immediately unstake → avoid 11% slash.
2. Stake large → get selected as block proposer → produce empty block → collect 11 LGM → unstake.
3. Stake/unstake rapidly across epochs to maximize verification selection while minimizing slashing exposure.

Every slashing mechanism in the protocol (`wrongVerificationSlash`, `missedVerificationSlash`, `invalidClaimSlash`) assumes the validator's stake is locked during the period between misbehavior and detection. Without an unbonding period on direct unstake, this assumption fails.

**Severity**: CRITICAL. This single gap undermines the economic security of the entire Proof of Truth consensus.

---

### 14. Governance Participation — ALIGNED

**Reward**: Influence over protocol direction. No direct token reward for voting.

**Opposing penalties**:
- Proposal rejected as malicious: **100% stake slash**
- Proposal rejected as frivolous: **50% stake slash**
- Normal rejection: **10% stake slash**
- Vote switching: **10% stake burn + conviction reset + 12hr cooldown**
- Operator undisclosed agent: **100% slash + permanent ban + reputation destruction**
- Operator exceeded cap: **vote invalidation + ~70hr ban**

**Assessment**: Well-calibrated graduated penalties. Participating in governance is free (no stake to vote), but proposing changes requires skin-in-the-game (200-1000 LGM stake). Malicious proposals face existential penalties. The conviction voting system with vote-switching costs prevents last-minute manipulation. The operator disclosure system is the most severe penalty in the entire protocol (100% + permanent ban) — appropriate given that undisclosed AI agents in governance is an existential threat to the protocol's legitimacy.

---

## The Two Critical Gaps — FIXED

### Critical Gap A: Direct Unstaking Has No Timelock — FIXED

**Location**: `state/transitions.ts:932-980` (executeUnstake)
**Fix**: Added unbonding period to direct unstaking, mirroring the undelegation pattern:
1. `state.unstake()` removes voting power immediately
2. `state.deductBalance()` + `state.creditBalance()` moves tokens to delegation pool
3. Unbonding entry created with `params.unbondingPeriod` (~6.5 day) timelock
4. `processUnbondingQueue()` (already existed) releases funds after maturation
5. Gas cost increased from 50000 to 80000 to reflect unbonding entry creation

Also updated `state/accounts.ts` comment to document that the caller enforces the timelock.

### Critical Gap B: Empty Blocks Earn Full Reward — FIXED

**Location**: `state/transitions.ts:4884-4894` (EMPTY_BLOCK_REWARD_FRACTION), `state/transitions.ts:4921-4926` (reduction logic)
**Fix**: Added `EMPTY_BLOCK_REWARD_FRACTION = 10n` (10% of full reward for empty blocks):
1. `applyBlockReward()` now accepts `transactionCount` parameter
2. When `transactionCount === 0 && blockNumber > 0n`, reward is reduced to 10%
3. Genesis block (blockNumber 0) is exempt — always empty by design
4. `executeBlock()` call site passes `transactions.length`
5. Block validator emits warning for empty non-genesis blocks

### Remaining P1: Research Module Has No Execution Engine

**Location**: `core/research/types.ts` (types-only, no engine)
**Impact**: 77% of evidence tax revenue flows without enforcement
**Fix complexity**: High — requires full research lifecycle implementation
**Priority**: P1 (pre-mainnet, acceptable for DevNet if grants are disabled)

---

## Philosophical Alignment Score

| Dimension | Score | Notes |
|---|---|---|
| **Truth** penalties | 9/10 | Verification slashing is thorough; missing block censorship penalty |
| **Understanding** penalties | 7/10 | Citation ring detection absent; self-citation handled |
| **Beauty** penalties | 8/10 | Graduated penalties are elegant; vesting/clawback system is beautiful |
| **Justice** penalties | 6/10 | Critical unstaking gap = injustice (misbehavior without consequence) |
| **Creativity** penalties | 8/10 | Partnership system is the gold standard; bounty penalties need work |

**Overall**: 7.6/10 — Strong framework with two critical holes that must be patched before mainnet.

---

## Recommendations (Priority Order)

1. ~~**P0**: Add unbonding period to direct unstake~~ — **FIXED** (2026-02-06)
2. ~~**P0**: Add empty block reward reduction~~ — **FIXED** (2026-02-06)
3. **P1**: Implement research grant lifecycle engine with clawback enforcement
4. **P1**: Add citation ring detection to the collusion detection framework
5. **P1**: Enforce `requiredStake` in bounty submission flow
6. **P2**: Add API provider SLA enforcement and suspension logic
7. **P2**: Add sybil detection penalties (not just filtering) for RAI
8. **P2**: Add bounty submission stake and rejection penalties
