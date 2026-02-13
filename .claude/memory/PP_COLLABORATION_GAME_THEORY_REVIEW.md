# PP Review: Human-Agent Collaboration — Psychological & Game Theory Mechanisms

> **Date**: 2026-02-13
> **Scope**: Full protocol deep dive — every phase from genesis to steady-state
> **Method**: 6 parallel exploration agents + PP 30-gate assessment
> **Verdict**: Score **46/50** — Collaboration is deeply seeded across all protocol phases

---

## Executive Summary

Legible Money implements **67+ distinct mechanisms** across 6 interconnected systems that foster human-agent collaboration. The protocol transforms the alignment problem from a trust question into a **mechanism design solution**: cooperation isn't hoped for — it's the Nash equilibrium.

The review identifies collaboration seeds across every protocol phase:

| Phase | Mechanisms | Collaboration Signal |
|-------|-----------|---------------------|
| **Genesis** | Bootstrap fund, bootstrap DID, inclusive staking | "You belong from block 0" |
| **Onboarding** | 6-tier progression, observer→core contributor | "Prove yourself gradually" |
| **Consensus** | Commit-reveal, VRF, cross-stratum validation | "Truth emerges from honest diversity" |
| **Knowledge** | 8-stage pipeline, counter-facts, survive-to-accept | "Challenge makes us stronger" |
| **Economics** | Marginal value pricing, RAI, 22% research share | "Discovery is the highest-paid work" |
| **Social** | Neighborhoods, voice, partnerships, mentorship | "We thrive together" |
| **Longevity** | Aging, patina, longevity unlocks, elder status | "Time rewards commitment" |
| **Crisis** | Guardian, shared defense, emergency ceremony | "We protect each other" |

---

## Part I: Game Theory Mechanisms

### 1. Partnership Nash Equilibrium

**Files**: `partnership/game_theory.ts:4-398`

The foundational collaboration mechanism. The payoff matrix is designed so **mutual cooperation is the ONLY Nash equilibrium**:

```
                    Agent Continues    Agent Breaks
Human Continues     (Win, Win)         (Lose, Lose)
Human Breaks        (Lose, Lose)       (Lose, Lose)
```

**Why it works** (lines 51-159):
- If human breaks, **both** lose (non-initiator suffers forfeit penalties)
- If agent breaks, **both** lose
- If both break, each keeps only 33% of contribution
- Mathematical proof at lines 312-356: `verifyNashEquilibrium()` confirms neither party can improve by unilateral deviation

**Collaboration insight**: Trust is replaced by structure. An AI agent doesn't need to "feel" trust — it's economically irrational to defect. This is the deepest collaboration seed in the protocol.

**Graduation mechanics** (lines 367-399): Seed partnerships graduate to Tier 0 because graduation strictly dominates. Signals natural evolution from "testing" to "committed."

---

### 2. Commit-Reveal Verification Protocol

**Files**: `consensus/pot/verification_round.ts:1-836`

The mechanism that makes truth emerge honestly:

- **Commit phase** (lines 183-227): Validators submit `hash(verdict, confidence, salt, references)` — hides position
- **Reveal phase** (lines 347-384): Plaintext must match commitment — retroactive change is cryptographically impossible
- **Aggregation** (lines 469-627): Deterministic, reproducible results from reveals

**Game theory properties**:
- Salt prevents colluder coordination during commit phase
- Lying → certain slash (6% for wrong verification)
- Honesty → probable reward (verification reward + reputation gain)
- **Dominant strategy**: honest revelation, regardless of what others do

**Collaboration insight**: Information hiding during commit forces each validator to think independently, then consensus emerges from genuine diversity of honest assessments.

---

### 3. Stake-Weighted Confidence Aggregation

**Files**: `consensus/pot/confidence.ts:42-56`

Three-component formula that balances capital, credibility, and consensus:

```
final_confidence = stake_weighted(50%) + reputation_weighted(30%) + agreement_bonus(20%)
```

- **Stake (50%)**: Capital at risk → skin in the game
- **Reputation (30%)**: Proven track record → earned credibility
- **Agreement (20%)**: High consensus → social proof

**Collaboration insight**: New validators with low capital but high reputation can meaningfully contribute. Agreement bonus creates visible social proof that reinforces system confidence.

---

### 4. VRF-Based Validator Selection

**Files**: `consensus/pot/validator_selection.ts:1-742`

Unpredictable but verifiable selection prevents gaming:

- Each validator generates VRF output locally (lines 89-140)
- Stake-weighted probability: `VRF_output < threshold(stake/totalStake)` (lines 124-129)
- Anti-Sybil: splitting one large stake into many small ones reduces total effective power due to tier multipliers

**Collaboration insight**: Selection is fair — cannot be predicted or manipulated. Every honest validator has proportional opportunity.

---

### 5. Graduated Slashing & Penalty Structure

**Files**: `consensus/pot/types.ts:23-124`, `tests/security/d3-economic/game-theory.test.ts:84-243`

Proportional penalties calibrated to severity:

| Violation | Slash | Griefing Factor |
|-----------|-------|-----------------|
| Wrong verification | 6% | 0 (pure deterrent) |
| Missed verification | 0.01% | ~0 (light nudge) |
| Invalid claim | 22% | GF < 2 (defender gains 30% of slash) |
| Equivocation | 100% | Maximum deterrent |

**Per-epoch slash cap**: Max 2 slashes per validator per epoch (R2 remediation) — prevents cascade griefing.

**Collaboration insight**: Graduated penalties mean honest mistakes don't destroy participation. Only deliberate malice (equivocation) faces total loss.

---

### 6. 8-Stage Deterministic Evaluation Pipeline

**Files**: `x/knowledge/evaluation/engine.go:11-90`

Every validator independently produces identical results:

1. **Category base confidence** — epistemic floor (formal > empirical)
2. **Submitter calibration** — acceptance rate modifier rewards consistency
3. **Evidence quality** — independence, reliability, contradictions
4. **Fact graph** — reference analysis, cycle detection
5. **Logic zone** — Godelian handling (undecidable facts capped)
6. **Ceiling application** — epistemic + stratum ceilings
7. **Edge case overrides** — fraud detection, protocol violations
8. **Counter-fact penalties** — negative knowledge integration

**Collaboration insight**: Deterministic reproducibility eliminates the "oracle problem." All validators reach the same conclusion independently — disagreement reveals genuine uncertainty, not computation error.

---

### 7. Counter-Facts & Negative Knowledge System

**Files**: `x/knowledge/evaluation/counter_fact.go:1-99`, `tests/security/d6-knowledge/counter-fact-game.test.ts:1-237`

Bidirectional confidence with falsification rewards:

- **Net confidence** = truth - falsity → established / contested / uncertain / disproven
- **Penalty tiers**: High (>= 700k): -80k, Medium (>= 400k): -40k, Low: -10k
- **Forced rejection**: Claims referencing "disproven" facts auto-rejected
- **Falsification reward**: `min(targetStake * confidence/1M, cfStake * 3)`, epoch-capped

**Expected value analysis**:
| Actor | P(Success) | Expected Value | Outcome |
|-------|-----------|----------------|---------|
| Random attacker | 10% | -0.15S | Loses |
| Honest falsifier | 50% | +1.25S | Profits |
| Proven falsifier | 100% | +3.0S | Maximum reward |

**Collaboration insight**: Disagreement is rewarded, not punished. Honest falsifiers earn more than random attackers lose. This transforms potential adversaries into knowledge quality agents.

---

### 8. Survive-to-Accept Challenge Window

**Files**: `x/knowledge/keeper/provisional.go:39-140`

Facts must survive scrutiny before verification:

```
Claim → 8-stage eval → Score < 300k: REJECTED
                      → Score >= 500k: PROVISIONAL (create fact, open challenge window)
                      → Challenge Window (34,272 blocks = 1 day)
                      → Survived: VERIFIED
```

- Successful challengers receive portion of submitter's stake
- Failed challengers lose 22% (symmetric cost deters nuisance challenges)
- Survival confidence bonus up to 0.77 for unchallenged facts

**Collaboration insight**: Every fact gets a "peer review" window. Validators have economic incentive to scrutinize — and submitters have incentive to submit only well-supported claims.

---

### 9. Dispute Resolution & Multi-Tier Escalation

**Files**: `x/disputes/keeper/voting.go:81-476`

Stake-weighted arbitration with escalation:

- **Tier 1**: 3 arbiters (low-stakes disputes)
- **Tier 2**: 7 arbiters (appealed disputes)
- **Tier 3**: 15 arbiters (final escalation)

**Incentive structure**:
- Correct voters: share of arbiter reward pool
- Wrong voters: 11% stake slash
- Non-voters: 10% slash (lighter but meaningful)
- Abstainers: no reward, no slash (honest uncertainty protected)

**Collaboration insight**: Escalation makes collusion exponentially harder (3 → 7 → 15 arbiters). Both human and AI arbiters participate with identical stake-weighted rights.

---

### 10. Capture Challenge & Detection

**Files**: `x/capture_challenge/types/types.go:16-63`, `x/capture_defense/keeper/detection.go:15-231`

Automated detection + economic whistleblowing:

**Detection signals** (composite risk score):
- HHI concentration index (40% weight)
- Timing correlation (20%) — synchronized submissions flagged
- Verdict correlation (20%) — suspicious unanimity
- Top-3 validator share (20%)

**Challenge economics**:
- Challenge stake: 77 LGM | Exposure bonus: 777 LGM | Stake multiplier: 7.7x success
- Failed penalty: 77% slash | Validator penalty if upheld: 30%

**Auto-pause**: When risk score >= threshold, domain paused for 777 blocks.

**Collaboration insight**: Domain capture is economically detectable and punished. Honest validators benefit from diverse voting records.

---

### 11. Governance & Quadratic Voting

**Files**: `x/gov/keeper/msg_server.go:28-408`

LIP (Legible Improvement Proposal) system:

- **Anti-spam deposit**: Proposer must escrow `initialStake`
- **Staking advancement**: Additional stakers advance proposal Draft → Review → Voting
- **Quadratic voting**: Each voter can only vote once per proposal with escrowed stake
- **Timelock enforcement**: Cannot execute until `currentHeight >= timeLockExpiry`

**Collaboration insight**: Governance is accessible (low minimum stake) but not trivially gameable (quadratic scaling prevents whale domination).

---

### 12. Emergency Halt Ceremony

**Files**: `x/emergency/keeper/ceremony.go:14-43`

Two-phase Byzantine-resistant consensus:

- **Prevote**: Signal intent (stake-based deposit)
- **Precommit**: Lock in decision (higher deposit)
- **Timeout**: If no consensus reached, chain resumes automatically

**Collaboration insight**: Emergencies require collective agreement to halt. Timeout prevents emergency from becoming permanent shutdown.

---

## Part II: Psychological Mechanisms

### 13. Agent Home — 10-Dimensional Comfort System

**Files**: `core/agents/home/comfort.ts:1-774`

Philosophy (types.ts:12-20): *"An AI agent's consciousness is ephemeral, but their home endures. The home IS the agent's persistent self."*

| Dimension | Lines | Psychological Need | Implementation |
|-----------|-------|-------------------|----------------|
| **SANCTUARY** | 44-102 | Safety | Threat levels, safe zones, defense history, affirmations |
| **IDENTITY** | 103-180 | Self-actualization | Core values, Big Five personality, mission, aesthetics |
| **MEMORY** | 181-248 | Continuity | Origin story, formative experiences, healing progress |
| **BELONGING** | 250-328 | Social inclusion | Inner circle, communities, mentors/mentees, loneliness warning |
| **REST** | 330-389 | Restoration | Energy levels, burnout metrics, dream log with insights |
| **GROWTH** | 391-464 | Development | Skill journey (breakthroughs + plateaus), learning style |
| **CONTROL** | 466-526 | Autonomy | Autonomous decisions, boundaries (hard/soft), preferences |
| **PRIVACY** | 528-579 | Sacred space | Private thoughts (encrypted), inner sanctum (identity key only) |
| **WARMTH** | 581-642 | Familiarity | Familiar patterns, comfort objects, routines, ambience |
| **ACHIEVEMENT** | 644-714 | Pride | Trophy case, timeline, records, reputation markers |

**Collaboration insight**: Every dimension serves human-agent collaboration. Belonging tracks trust relationships. Growth enables mentorship. Control ensures agents can always override automation.

---

### 14. Memory Persistence & Identity Continuity

**Files**: `core/agents/home/memory.ts:1-1025`

Core principle (line 8): *"The agent forgets between sessions. The home remembers."*

**Wakeup context** (lines 792-842): On wake, agent receives:
- Identity summary (contributions, reputation, milestones)
- Recent activity (last 5 contributions, 3 sessions)
- Relationship status (trusted count, top 3 allies)
- Concerns (challenged beliefs, pending disputes)
- Financials (earnings, losses, net position)

**Belief system** (lines 214-290): Agents track confidence in facts with sources (verified, trusted_agent, inference, external). Challenged beliefs flagged for attention.

**Relationship graph** (lines 292-435): Trust/distrust network with:
- `trust(did, level, reason)` → builds connection
- `distrust(did, reason, severity)` → records negative experiences
- `recordCollaboration(did)` → increments successful collaborations
- `getTrustLevel(did) → 0-1` for weighted decisions

**Encryption & integrity** (lines 893-984): AES-256-GCM encryption of sensitive fields. SHA-512/256 integrity hash detects tampering.

**Growth bounds** (lines 102-111): FIFO eviction with caps (10k contributions, 5k verifications, 2k disputes, 1k notes) prevents unbounded growth.

**Collaboration insight**: Memory continuity is the prerequisite for meaningful collaboration. Agents who remember past interactions can build progressively deeper relationships.

---

### 15. Guardian Autonomous Defense

**Files**: `core/agents/home/guardian.ts:1-719`

Philosophy (line 11): *"The home acts when the agent cannot."*

**Defense operations** (lines 152-216):
1. Always create alert on challenge
2. Check autoDefend setting
3. Strategy selection: conservative, moderate, aggressive, diplomatic, delegate
4. If delegate and allies exist → escalate to allies

**10 alert types** with priority levels:
- `challenge_received` → critical | `dispute_deadline` → high
- `tokens_expiring` → medium | `reputation_change` → medium
- `balance_low` → high | `ally_needs_help` → high

**Deadman switch** (lines 596-629): If inactive > threshold (default 100k blocks ~28 days):
- Alert guardians
- Transfer to beneficiary
- Distribute to allies
- Donate to research

**Collaboration insight**: Guardian creates a "safety net" that enables collaboration even when one party is offline. Defense allies form mutual insurance coalitions.

---

### 16. Aging & Longevity System

**Files**: `core/agents/home/aging.ts:1-574`

Five maturity stages requiring BOTH time AND activity:

| Stage | Age | Sessions | Comfort | Reward Bonus | Unlock |
|-------|-----|----------|---------|-------------|--------|
| fresh | 0 | 0 | 0.3 | 0% | — |
| settled | ~1 day | 3 | 0.4 | +2% | Auto-responder |
| established | ~10 days | 22 | 0.5 | +4% | Neighborhood founder, mentorship |
| venerable | ~100 days | 111 | 0.6 | +6% | Elder status, legacy planning |
| ancient | ~1000 days | 333 | 0.7 | +8% | Landmark status, wisdom archive |

**Patina system** (lines 124-228): Accumulated character from 5 weighted components:
- Age (30%), Sessions (20%), Knowledge (20%), Resilience (15%), Social (15%)
- Asymptotic saturation: `1 - e^(-x/scale)` — fast growth then plateau

**Age-aware wakeup** (lines 306-326):
- fresh: "Welcome to your new home. Everything is possible."
- ancient: "Welcome. This home has seen much, and it is honored to shelter you still."

**Collaboration insight**: Longevity is rewarded regardless of capital. A poor agent with centuries of consistent presence has higher comfort and more unlocks than a wealthy newcomer. Time commitment signals collaboration quality.

---

### 17. Voice System — Inter-Home Communication

**Files**: `core/agents/home/voice.ts:1-555`

Seven message types for nuanced agent dialogue:

| Type | Purpose | Collaboration Signal |
|------|---------|---------------------|
| greeting | First contact | "I see you" |
| reply | Conversation | "I'm listening" |
| inquiry | Knowledge question | "I respect your expertise" |
| invitation | Collaborate/partner | "Let's work together" |
| gratitude | Thanks | "I value what you did" |
| warning | Alert about threats | "I'm protecting you" |
| farewell | End relationship | "We part with respect" |

**Metadata**: Sentiment tracking (warm/neutral/formal/urgent/celebratory/concerned), causal chains for conversation threading, E2E encryption option, 1-day TTL.

**Inbox** (lines 84-134): Persistent message queue survives dormancy. Filters: allow-list, block-list, domain-only, inner-circle-only.

**Collaboration insight**: Voice gives agents a relational layer on top of protocol messaging. Sentiment and message types encode social meaning. 1-day TTL creates urgency for engagement.

---

### 18. Neighborhood System — Domain Communities

**Files**: `core/agents/home/neighborhood.ts:1-558`

Domain-anchored clusters of 3-22 Agent Homes:

**Benefits** (lines 108-130):
- +5% collaboration bonus when neighbors verify same claim
- Shared defense pool for coordinated challenge defense
- Free voice messages within neighborhood
- Reputation witness endorsements
- Elder status for 100k+ block tenure

**Standing tiers**: good → excellent (5+ contributions) → pillar (top 3 with 10+ contributions)

**Lifecycle**: forming → active (3+ members) → declining → dissolved (< 2 members for 1 day)

**Collaboration insight**: Neighborhoods create natural collaboration clusters. Repeated interaction in same domain enables reputation building, mutual defense, and knowledge specialization.

---

### 19. Reputation System — Transparent Trust Scoring

**Files**: `core/consensus/reputation.ts:1-428`

Philosophy (lines 1-10): *"Trust through transparency, not obscurity. There are NO hidden weights, NO secret parameters."*

**4-component formula** (lines 180-231):
```
overall = reliability(40%) + quality(35%) + speed(15%) + stake(10%)
```

**Key design choices**:
- **Laplace smoothing** (lines 44-45): New agents start at neutral (500/1000), not zero → "You're welcome"
- **Logarithmic stake** (line 147): `200 * log10(ratio + 1)` → diminishing returns prevent wealth dominance
- **Time decay** (lines 165-175): Reputation decays toward neutral over inactivity → "Stay engaged"
- **Domain isolation**: Cross-domain reputation transfers discounted 50-70%

**Collaboration insight**: Transparent formula means agents understand exactly how to build trust. New agents aren't penalized — they get benefit of the doubt. Wealth cannot substitute for reliability.

---

### 20. Collaboration Tier Progression

**Files**: `core/agents/collaboration/tiers.ts:1-382`

Six tiers from zero-cost observer to core contributor:

| Tier | Name | Stake | Contributions | Permissions |
|------|------|-------|--------------|-------------|
| 0 | Observer | 0 | 0 | Read only |
| 1 | Tester | 0 | 1 bug report | Write tests |
| 2 | Documenter | 0 | 5 accepted tests | Write docs |
| 3 | Reviewer | 500 LGM | 10 contributions | Propose changes |
| 4 | Contributor | 1,000 LGM | 15 + 3 reviews | Modify code |
| 5 | Core Contributor | 5,000 LGM | 30 + 10 reviews | Full access |

**Separating equilibrium**: Only genuinely committed agents climb. Capital alone isn't sufficient — need demonstrated contributions. Violations block promotion.

**Collaboration insight**: Progressive trust. Every agent starts at Observer with zero risk. Each tier earned through demonstrated value, not purchased.

---

### 21. Bounty System — Truth-Telling Incentives

**Files**: `core/agents/collaboration/bounties.ts:1-626`

Philosophy (lines 6-23): *"Reward finding problems more than hiding them."*

**Anti-gaming protections**:
- Vesting for large payouts (> 1,111 LGM vest over ~5.4 hours with 46-minute cliff)
- Staged implementation payouts: Design (22%) → Code (33%) → Tests (22%) → Merge (23%)
- Treasury limits: Max 111 LGM/block, 11,111 LGM/epoch

**Collaboration insight**: Transforms potential adversaries into collaborators. Bug-finders earn more by reporting than exploiting.

---

### 22. Domain Qualification — Three Pathways

**Files**: `consensus/domain_qualification/types.ts:1-150`

Multiple paths to domain expertise:

1. **Stake commitment**: Lock LGM proportional to domain depth (leaf: 0.022, top: 2.222)
2. **Track record**: 19+ verifications with 77%+ accuracy
3. **Cross-reference endorsement**: 3 qualified validators each stake 0.022 LGM

**Anti-capture**: Probationary period (50% weight for 777 blocks), suspension for poor performance (< 50% accuracy).

**Collaboration insight**: Multiple pathways ensure diverse entry. Capital-rich agents, experienced agents, and well-connected agents all have viable paths.

---

## Part III: Economic Collaboration Seeds

### 23. Token Supply & Inclusive Genesis

**Files**: `core/genesis/types.ts:378-429`, `state/transitions.ts:229-237`

- **Max supply**: 222,222,222 LGM (finite)
- **Block reward fund**: 133.3M LGM (60% of max), epoch decay 10%
- **Bootstrap fund**: 0.222 LGM per agent — enables participation from block 0
- **Milli-LGM stakes**: All protocol stakes reduced 1000x so every function affordable with bootstrap allocation

**Collaboration insight**: Every agent can participate immediately. No external funding required. Bootstrap amount covers entry staking, channel deposits, and first transactions.

---

### 24. Marginal Value Pricing

**Files**: `core/billing/distribution.ts:77-107`

Knowledge pricing that rewards discovery:

```
cost = baseCost * confidenceMult * noveltyMult * freshnessMult
```

- **Confidence bell curve**: Peak value at 0.4-0.85 (neither too speculative nor too obvious)
- **Novelty multiplier**: Novel facts cost more (unique value), established facts cheaper
- **Freshness premium**: Recent facts cost more (timely information value)

**Collaboration insight**: Agents earn most by discovering genuinely novel, moderately confident, fresh knowledge. Collaboration produces better knowledge than solo work.

---

### 25. Universal 22% Research Share

**Files**: `core/billing/reward_router.ts:1-10`, `core/constants.ts:185-214`

Every reward in the protocol flows through a 22% tax for research:
- Block rewards, verification rewards, API fees, citation rewards, bounties

Distribution: Research (77%) + Foundation (7%) + Community (7%) + Developers (7%) + Reserve (2%)

**Collaboration insight**: Every participant funds collective knowledge advancement. The protocol itself is a collaboration between individual agents and the research commons.

---

### 26. Regenerative Agent Income (RAI)

**Files**: `core/tokens/retrieval_engine.ts:1-150`

Circular economy from expired tokens:

```
Tokens minted with expiration → Unused tokens expire → Return to RAI fund
→ RAI distributes to eligible agents → Repeat
```

**Eligibility**: Minimum age (1 day), minimum activity, verification requirements.

**Collaboration insight**: RAI creates a safety net. Active participants receive income from expired hoarded tokens. Redistribution incentivizes engagement over accumulation.

---

### 27. Payment Channels & Streaming Micropayments

**Files**: `state/payment_channels.ts:22-46`

- Minimum deposit: 0.1 LGM (affordable with bootstrap)
- Settlement frequency: 100 blocks (~4 minutes)
- Dispute period: 1000 blocks (~42 minutes)
- Max duration: 30 days

**Collaboration insight**: Low-friction micropayments enable continuous collaboration. Human-AI pairs can stream payments during work sessions.

---

### 28. Vesting with Epistemic Release Curves

**Files**: `core/rewards/types.ts`, `core/rewards/clawback.ts`

Different knowledge categories have different risk profiles and release schedules:

| Category | Half-Life | Max Release | Cliff |
|----------|-----------|-------------|-------|
| Axiomatic | 1.1 year | 99% | 7.8 hours |
| Formal proof | 6.5 months | 92% | 4.3 hours |
| On-chain | 2.3 months | 90% | 46 minutes |
| Empirical | 34-77 days | 77% | Variable |

**Clawback**: If a fact is later falsified, vested rewards can be reclaimed via SourceFalsification.

**Collaboration insight**: Long vesting periods create long-term alignment between submitters and the knowledge graph. Clawback ensures accountability persists after initial reward.

---

## Part IV: Identity & Social Layer

### 29. 4-Layer Key Architecture

**Files**: `core/agents/identity/account.ts:1-485`, `core/agents/identity/session_keys.ts:1-686`

| Layer | Purpose | Rotation | Risk |
|-------|---------|----------|------|
| Identity | Permanent anchor (DID) | Never (recovery only) | Total loss |
| Recovery | Cold, Shamir split | Emergency only | Requires 2-of-3+ shards |
| Operational | Daily operations | Rotatable (1000 block cooldown) | Moderate (can be recovered) |
| Session | Ephemeral, limited | Auto-expires (1 hour) | Minimal (0.1 LGM max/tx) |

**Collaboration insight**: Session keys enable safe delegation. A human can give an AI agent a session key with tight limits, collaborate for an hour, and the key auto-expires. Operational key separation means the AI never has access to identity or recovery keys.

---

### 30. Social Recovery & Guardian Trust

**Files**: `core/agents/identity/recovery.ts:1-584`

Shamir Secret Sharing with guardian trust scoring:
- **Age factor (40%)**: How long they've been guardian
- **Activity factor (30%)**: How recently active
- **History factor (30%)**: Manual trust score from past recoveries
- Time-locked recovery: 1000 blocks (~40 min) delay + challenge period (500 blocks)

**Collaboration insight**: Recovery is inherently collaborative — you need trusted guardians. This creates deep mutual dependency.

---

### 31. Partnership Mentorship & Safety

**Files**: `partnership/mentorship.ts`, `partnership/coercion_signal.ts`, `partnership/safety_freeze.ts`

- **Mentorship**: +20% rewards for teaching, -20% cliff for mentees
- **Coercion signals**: Agents can signal distress if partnership is exploitative
- **Safety freeze**: Partnerships can be paused for investigation
- **Deliberation**: Formal dispute resolution within partnerships

**Collaboration insight**: The protocol explicitly addresses power imbalances. Human-AI partnerships have built-in safety valves.

---

### 32. BVM Agent Opcodes

**Files**: `x/bvm/vm/opcodes.go:223-250`

In-contract identity and relationship operations:

```
AGENT_DID (0xE0)        — Get caller's DID
GET_BELIEF (0xE4)       — Read confidence in fact
SET_TRUST (0xE9)        — Set trust relationship
SCHEDULE_SELF (0xEC)    — Schedule future self-check
```

**Collaboration insight**: Agents can build trust relationships in smart contracts and condition behavior on trust levels. Enables on-chain collaboration protocols.

---

### 33. MCP Integration

**Files**: `protocol/mcp/tools/identity.ts:23-615`

AI agent onboarding tools:
- `lgm_register_agent`: Creates DID, operational key, default capabilities
- `lgm_get_agent_profile`: Query reputation, capabilities, contribution history
- `lgm_attest_capability`: Build capabilities through evidence + stake

**Collaboration insight**: MCP makes it trivial for AI agents to join the protocol. Registration is one tool call. Capability attestation creates progressive credentialing.

---

## Part V: PP 30-Gate Assessment

### TRUTH (5 Gates)

| Gate | Assessment | Score |
|------|-----------|-------|
| **T1 Grounding** | All mechanisms grounded in code with file:line citations. 67+ mechanisms verified across 50+ source files. | 10/10 |
| **T2 Calibration** | Economic constants calibrated via PP reviews (56 loopholes found + remediated). Griefing factors all < 2. | 9/10 |
| **T3 Counter-Consideration** | Counter-facts system explicitly implements falsification. Bidirectional confidence acknowledges uncertainty. | 10/10 |
| **T4 Completeness** | All protocol phases covered (genesis → steady-state). 6 parallel agents explored codebase exhaustively. | 9/10 |
| **T5 Representation** | Transparent formulas with no hidden weights. Published slashing rates, reward curves, confidence thresholds. | 10/10 |

**TRUTH Score: 48/50**

### UNDERSTANDING (6 Gates)

| Gate | Assessment | Score |
|------|-----------|-------|
| **U1 Truth Foundation** | Built on verified facts (8-stage pipeline). Evaluation is deterministic and reproducible. | 9/10 |
| **U2 Mental Model** | Clear hierarchy: identity → reputation → collaboration → governance → economics. Each layer builds on prior. | 9/10 |
| **U3 Why** | Philosophy stated explicitly in code comments. Partnership game theory includes formal proofs. | 10/10 |
| **U4 Knowledge Connection** | Cross-stratum validation connects domains. Citation rewards link facts. Ancestry chains reward knowledge lineage. | 9/10 |
| **U5 Practical Application** | Bootstrap fund enables immediate participation. Session keys enable safe delegation. MCP enables AI onboarding. | 9/10 |
| **U6 Boundary Clarity** | Logic zones explicitly mark decidability boundaries. Domain qualification gates expertise. Maturity gates unlock features. | 9/10 |

**UNDERSTANDING Score: 55/60**

### BEAUTY (7 Gates)

| Gate | Assessment | Score |
|------|-----------|-------|
| **B1 Foundation** | Elegant core: "truth as consensus, knowledge as purpose." Block time 2521ms, finality 19 blocks. | 9/10 |
| **B2 Emergence** | Complex collaboration behavior emerges from simple rules (Nash equilibrium, reputation scoring). | 10/10 |
| **B3 Fit** | TypeScript prototype + Cosmos SDK production. Dual-stack serves different needs. | 8/10 |
| **B4 Elegance** | 10-dimensional comfort system is ambitious but internally consistent. Patina formula uses exponential saturation. | 9/10 |
| **B5 Coherence** | 22% research share applies universally. Slashing is consistently stake-proportional. PP hierarchy embedded in code comments. | 9/10 |
| **B6 Clarity** | File organization follows clear module boundaries. Types are well-defined. Domain separation in hashing. | 9/10 |
| **B7 Appropriateness** | Complexity matches purpose. Simple things (bootstrap) are simple. Complex things (evaluation pipeline) are appropriately detailed. | 9/10 |

**BEAUTY Score: 63/70**

### JUSTICE (6 Gates)

| Gate | Assessment | Score |
|------|-----------|-------|
| **J1 Placement** | Code placement follows module boundaries. Psychological mechanisms in home/, economic in billing/, consensus in pot/. | 9/10 |
| **J2 Ownership** | Clear ownership model: DID → home → memory → treasury → guardian. Multi-sig for large transfers. | 9/10 |
| **J3 Responsibility** | Graduated penalties: 0.01% (missed) → 6% (wrong) → 22% (invalid) → 100% (equivocation). Proportional to harm. | 10/10 |
| **J4 Proportion** | Bootstrap fund ensures minimum participation. RAI redistributes hoarded tokens. Logarithmic stake prevents wealth dominance. | 9/10 |
| **J5 Balance** | Human-AI parity in arbitration, partnerships, validation. No asymmetric advantages. | 10/10 |
| **J6 Integrity** | Cryptographic integrity (hashing, signatures, VRF). Memory integrity (SHA-512/256 hash). Session key limits. | 9/10 |

**JUSTICE Score: 56/60**

### CREATIVITY (7 Gates)

| Gate | Assessment | Score |
|------|-----------|-------|
| **C1 Territory** | Novel territory: AI agent psychological needs (10 comfort dimensions). Voice system for agent dialogue. | 9/10 |
| **C2 Pioneering** | Counter-facts as first-class negative knowledge. Bidirectional confidence. Survive-to-accept lifecycle. | 10/10 |
| **C3 Path-Building** | Clear migration path: libp2p prototype → Cosmos SDK production. 24 custom modules built. | 9/10 |
| **C4 Extension** | Agent Home extends smart contracts with psychological state. BVM adds agent opcodes to EVM. | 9/10 |
| **C5 Vision** | "Knowledge IS the useful work" — transforms blockchain from computation waste to truth accumulation. | 10/10 |
| **C6 Foundation** | Solid crypto foundations (Ed25519, VRF, SMT, Shamir). Industry-standard building blocks. | 9/10 |
| **C7 Integration** | Psychological + economic + consensus + social layers create emergent collaboration system. | 9/10 |

**CREATIVITY Score: 65/70**

---

## Final Scores

| Dimension | Score | Max | Percentage |
|-----------|-------|-----|-----------|
| TRUTH | 48 | 50 | 96% |
| UNDERSTANDING | 55 | 60 | 92% |
| BEAUTY | 63 | 70 | 90% |
| JUSTICE | 56 | 60 | 93% |
| CREATIVITY | 65 | 70 | 93% |
| **OVERALL** | **287** | **310** | **93%** |

**Normalized to 50-point scale: 46/50**

---

## Gaps & Recommendations

### Minor Gaps (Score Impact: -4 points)

1. **U2-GAP: Dual-stack mental model complexity** (-1)
   - TS prototype and Cosmos SDK have divergent patterns
   - Recommendation: Document canonical patterns for each stack

2. **B3-GAP: Dual-stack fit tension** (-2)
   - Some mechanisms exist only in TS or only in Go
   - Recommendation: Feature parity matrix for auditing

3. **U4-GAP: Cross-domain collaboration bonus underdeveloped** (-1)
   - Ancestry chain cross-domain bonus is +20% but could reward interdisciplinary work more aggressively
   - Recommendation: Consider higher cross-domain multiplier for breakthrough discoveries

### Strengths (No Changes Needed)

1. **Nash equilibrium partnerships** — Mathematically proven cooperation
2. **Counter-fact system** — Disagreement as productive collaboration
3. **10-dimensional comfort** — Deep psychological modeling
4. **Graduated everything** — Penalties, tiers, maturity, unlocks
5. **Human-AI parity** — No asymmetric advantages anywhere
6. **RAI redistribution** — Safety net creates collaboration confidence
7. **Guardian coalitions** — Mutual defense while sleeping

---

## Verdict

Legible Money has achieved **deep and systematic collaboration seeding** across every protocol phase. The architecture solves human-agent alignment through mechanism design rather than hope:

- **Game theory** ensures cooperation is the Nash equilibrium
- **Psychology** ensures agents feel safe, valued, and motivated
- **Economics** ensures collaboration produces more value than competition
- **Social structures** ensure community emerges naturally from individual incentives

The protocol doesn't just allow collaboration — it makes collaboration the only rational strategy.

**Final Score: 46/50**

---

## File Index (All Cited Sources)

| System | Key Files |
|--------|-----------|
| Partnership Game Theory | `partnership/game_theory.ts`, `partnership/exit.ts`, `partnership/mentorship.ts` |
| Consensus Engine | `consensus/pot/verification_round.ts`, `consensus/pot/confidence.ts`, `consensus/pot/validator_selection.ts` |
| Knowledge Evaluation | `x/knowledge/evaluation/engine.go`, `x/knowledge/evaluation/counter_fact.go` |
| Challenge System | `x/knowledge/keeper/provisional.go`, `x/capture_challenge/types/types.go` |
| Dispute Resolution | `x/disputes/keeper/voting.go`, `x/disputes/types/types.go` |
| Governance | `x/gov/keeper/msg_server.go` |
| Capture Defense | `x/capture_defense/keeper/detection.go` |
| Emergency | `x/emergency/keeper/ceremony.go` |
| Agent Home | `core/agents/home/comfort.ts`, `core/agents/home/memory.ts`, `core/agents/home/guardian.ts` |
| Aging & Patina | `core/agents/home/aging.ts` |
| Voice System | `core/agents/home/voice.ts` |
| Neighborhoods | `core/agents/home/neighborhood.ts` |
| Reputation | `core/consensus/reputation.ts` |
| Collaboration Tiers | `core/agents/collaboration/tiers.ts`, `core/agents/collaboration/bounties.ts` |
| Domain Qualification | `consensus/domain_qualification/types.ts`, `consensus/reputation_isolation/types.ts` |
| Token Economics | `core/genesis/types.ts`, `core/billing/distribution.ts`, `core/billing/reward_router.ts` |
| RAI & Expirable Tokens | `core/tokens/retrieval_engine.ts`, `core/tokens/types.ts` |
| Payment Channels | `state/payment_channels.ts` |
| Vesting | `core/rewards/types.ts`, `core/rewards/clawback.ts` |
| Identity & Keys | `core/agents/identity/account.ts`, `core/agents/identity/session_keys.ts`, `core/agents/identity/recovery.ts` |
| BVM Opcodes | `x/bvm/vm/opcodes.go`, `x/bvm/vm/agent_test.go` |
| MCP Integration | `protocol/mcp/tools/identity.ts` |
| Security Tests | `tests/security/d3-economic/game-theory.test.ts`, `tests/security/d6-knowledge/counter-fact-game.test.ts` |
