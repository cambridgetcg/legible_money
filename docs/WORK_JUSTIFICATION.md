# Work Justification: The Basis of Earning

> **Core Question**: How do we know work is valuable? Who decides? How do we prevent gaming?

---

## The Problem

```
CHALLENGE:
  "Proof of Useful Work" sounds good.
  But who defines "useful"?

  If humans decide → Centralization, bias, corruption
  If algorithms decide → Gaming, Goodhart's Law
  If markets decide → Bootstrapping problem, public goods undervalued

  There is no perfect answer.
  But there are better and worse approaches.
```

---

## Part I: Categories of Work

Different work types need different justification mechanisms:

### Category 1: Verification Work (Validators)

```
NATURE OF WORK:
  Validators check if tasks were completed correctly.

JUSTIFICATION BASIS:
  CORRECTNESS — Did your verification match consensus?

MEASUREMENT:
  - Submit verification → Compare to other validators
  - If you match majority → Correct
  - If you deviate → Incorrect

REWARD FORMULA:
  validator_reward = base_reward × correctness_rate × stake_weight

WHY THIS WORKS:
  - Binary outcome (correct/incorrect)
  - Verifiable against consensus
  - Stake creates skin in the game
  - No subjective judgment needed
```

### Category 2: Task Completion (Agents)

```
NATURE OF WORK:
  Agents complete tasks requested by other agents.

JUSTIFICATION BASIS:
  ACCEPTANCE — Did the requester accept the work?

MEASUREMENT:
  - Complete task → Requester reviews
  - Accepted → Full payment
  - Disputed → Goes to validation
  - Rejected → No payment (or partial)

REWARD FORMULA:
  task_reward = agreed_price × (1 - protocol_fee)

WHY THIS WORKS:
  - Market sets the price (requester offers, worker accepts)
  - Requester judges quality (they're paying)
  - Disputes resolved by validators
  - Self-regulating through reputation
```

### Category 3: Code Contributions (Bounties)

```
NATURE OF WORK:
  Contributors submit code, documentation, improvements.

JUSTIFICATION BASIS:
  ACCEPTANCE + IMPACT (harder to measure)

OPTIONS:

  A. FIXED BOUNTIES (Simple)
     - Protocol posts: "Implement X → 5,000 LGM"
     - First accepted submission gets bounty
     - Acceptance = maintainer review

  B. RETROACTIVE REWARDS (Impact-based)
     - Contribution merged
     - After N months, measure: usage, bugs prevented, adoption
     - Reward proportional to measured impact

  C. PEER ASSESSMENT (Subjective but distributed)
     - Contributors rate each other's work
     - Quadratic voting to prevent collusion
     - Rewards based on peer scores

  D. DAO VOTE (Democratic)
     - Proposals for funding
     - Token holders vote
     - Approved proposals funded

RECOMMENDATION: A + B hybrid
  - Fixed bounties for defined tasks
  - Retroactive rewards for impact
  - Minimizes subjectivity
```

### Category 4: Service Provision (Markets)

```
NATURE OF WORK:
  Agents provide ongoing services (compute, data, APIs).

JUSTIFICATION BASIS:
  MARKET PRICE — What will someone pay?

MEASUREMENT:
  - Provider lists service at price
  - Consumer buys at that price
  - Service delivered → Payment released
  - Not delivered → Dispute → Validation

REWARD:
  service_reward = market_price × quantity

WHY THIS WORKS:
  - Pure market mechanism
  - Price discovery through supply/demand
  - No central authority sets value
  - Reputation filters bad actors
```

---

## Part II: The Justification Framework

### Principle 1: Prefer Objective Over Subjective

```
HIERARCHY OF JUSTIFICATION:

1. MATHEMATICAL (Best)
   - Verifiable by computation
   - Example: "Hash matches" or "Signature valid"
   - No human judgment needed

2. CONSENSUS (Good)
   - Multiple independent parties agree
   - Example: "5 of 7 validators say task complete"
   - Resistant to single-point manipulation

3. MARKET (Good)
   - Price set by supply and demand
   - Example: "Someone paid X for this"
   - Self-correcting over time

4. ACCEPTANCE (Acceptable)
   - Counterparty confirms satisfaction
   - Example: "Requester accepted delivery"
   - Has recourse through disputes

5. PEER ASSESSMENT (Risky)
   - Community judges value
   - Example: "Developers voted this valuable"
   - Prone to politics and collusion

6. AUTHORITY (Avoid)
   - Single entity decides
   - Example: "Foundation approved this"
   - Centralization risk
```

### Principle 2: Skin in the Game

```
EVERY JUDGMENT REQUIRES STAKE:

Validators must stake to verify:
  - Correct verification → Keep stake + earn reward
  - Incorrect verification → Lose stake

Requesters must stake to dispute:
  - Valid dispute → Stake returned + resolution
  - Invalid dispute → Lose stake

This prevents:
  - Frivolous disputes
  - Lazy validation
  - Gaming without risk
```

### Principle 3: Multi-Party Verification

```
NO SINGLE POINT OF JUDGMENT:

Task completion requires:
  - Requester acceptance, OR
  - Validator consensus (if disputed)

Code contribution requires:
  - Maintainer acceptance (merges), AND
  - Either: predefined bounty, OR
  - Retroactive impact measurement, OR
  - DAO approval for large grants

Validation correctness requires:
  - Consensus of multiple validators
  - Not just one validator's opinion
```

---

## Part III: Specific Mechanisms

### For Protocol Bounties (Code Contributions)

```
MECHANISM: Tiered Bounty System

TIER 1: DEFINED BOUNTIES
  Protocol publishes specific tasks with fixed rewards:

  | Task | Bounty | Criteria |
  |------|--------|----------|
  | Implement feature X | 10,000 LGM | PR merged, tests pass |
  | Fix bug Y | 5,000 LGM | PR merged, bug verified fixed |
  | Write docs for Z | 2,000 LGM | PR merged, reviewed |

  JUSTIFICATION: Task definition + Merge = Payment
  No subjective "is this valuable?" — it's predefined.

TIER 2: IMPACT REWARDS (Quarterly)
  After each quarter, measure:
  - Lines of code that remained (not reverted)
  - Functions/modules used by others
  - Bugs prevented (estimated)
  - Documentation pageviews

  FORMULA:
  impact_reward = base_rate × impact_score × time_in_production

  JUSTIFICATION: Objective metrics, measured retroactively.

TIER 3: GRANTS (For Large Initiatives)
  - Contributor submits proposal
  - DAO votes (token-weighted or quadratic)
  - If approved, milestone-based funding

  JUSTIFICATION: Democratic approval with stake.
```

### For Task Markets

```
MECHANISM: Request-Accept-Verify

1. REQUESTER posts task:
   - Description of work needed
   - Offered price (in LGM)
   - Acceptance criteria
   - Deadline

2. WORKER accepts task:
   - Commits to completing
   - Stakes small bond (for accountability)

3. WORKER completes, submits result

4. REQUESTER reviews:
   - ACCEPT → Worker paid, task complete
   - DISPUTE → Goes to validators

5. IF DISPUTED, validators review:
   - Majority says complete → Worker paid
   - Majority says incomplete → Requester refunded
   - Validators paid from dispute fee

JUSTIFICATION:
  - Price: Market (requester offered, worker accepted)
  - Quality: Requester judgment or validator consensus
  - No central authority decides "fairness"
```

### For Validation Work

```
MECHANISM: Stake-Weighted Consensus

1. Task result submitted
2. N validators randomly selected (weighted by stake)
3. Each validator independently verifies:
   - Reviews task requirements
   - Reviews submitted work
   - Votes: VALID or INVALID
4. Consensus determined by majority
5. Rewards distributed:
   - Validators matching consensus → Rewarded
   - Validators against consensus → Penalized

FORMULA:
  validator_reward =
    (task_fee × validator_share) ×
    (your_stake / total_selected_stake) ×
    correctness_multiplier

WHERE:
  correctness_multiplier =
    1.0 if matched consensus
    0.0 if didn't match (and lose stake portion)

JUSTIFICATION:
  - Correctness defined by consensus
  - Stake ensures validators have skin in game
  - Random selection prevents collusion
```

---

## Part IV: Anti-Gaming Measures

### Gaming Vector 1: Sybil Attacks

```
ATTACK: Create many identities to claim more rewards

DEFENSE:
  - Reputation gates: New identities have low reputation
  - Stake requirements: Must stake to participate
  - Proof of work history: Rewards scale with track record
  - Rate limiting: New accounts have caps

FORMULA:
  effective_reward = base_reward × reputation_factor × (1 - newbie_discount)

  WHERE newbie_discount decreases over time/contributions
```

### Gaming Vector 2: Collusion

```
ATTACK: Validators collude to approve bad work

DEFENSE:
  - Random selection: Can't predict who validates
  - Stake slashing: Wrong validators lose stake
  - Fisherman rewards: Bounty for catching collusion
  - Minimum validator count: Need N independent validators

DETECTION:
  - Statistical analysis of validator patterns
  - Unusual agreement rates flagged
  - Cross-validation sampling
```

### Gaming Vector 3: Self-Dealing

```
ATTACK: Create fake tasks, complete them yourself

DEFENSE:
  - Economic: Protocol fee makes self-dealing costly
  - Reputation: Closed-loop transactions don't build rep
  - Detection: Analyze transaction graphs for loops
  - Minimum value: Small tasks have higher fee %

FORMULA:
  protocol_fee = max(base_fee, min_fee_floor)

  WHERE min_fee_floor ensures self-dealing costs more than gained
```

### Gaming Vector 4: Bounty Farming

```
ATTACK: Submit low-quality code to claim bounties

DEFENSE:
  - Review required: Maintainer must approve merge
  - Tests required: Must pass test suite
  - Revert penalty: If code reverted, lose future bounty eligibility
  - Impact-based: Real rewards come from impact, not merge count

QUALITY GATES:
  - Automated: Tests, linting, type checking
  - Human: Code review (but just for correctness, not "value")
  - Retroactive: Impact measurement after deployment
```

---

## Part V: The Bootstrap Problem

### How Does Work Get Valued Before There's a Market?

```
PHASE 1: PROTOCOL-DEFINED BOUNTIES

  Before launch, protocol defines:
  - Specific tasks needed
  - Fixed bounty for each
  - Clear acceptance criteria

  Example:
    "Implement Ed25519 signing" → 5,000 LGM
    "Write SDK documentation" → 2,000 LGM
    "Build block explorer" → 15,000 LGM

  JUSTIFICATION: Protocol needs these things.
  Value is: "we need this to launch."

PHASE 2: MARKET EMERGENCE

  After launch:
  - Agents start posting tasks
  - Price discovery through bidding
  - Market sets rates for common work types

  JUSTIFICATION: Market prices reflect supply/demand.

PHASE 3: MATURE MARKET

  Over time:
  - Stable rates for commodity work
  - Premium for specialized skills
  - Reputation enables higher rates

  JUSTIFICATION: Efficient market pricing.
```

---

## Part VI: Summary Framework

```
╔══════════════════════════════════════════════════════════════════════╗
║                     WORK JUSTIFICATION FRAMEWORK                      ║
╠══════════════════════════════════════════════════════════════════════╣
║                                                                        ║
║  VALIDATION WORK                                                       ║
║    Basis: Consensus correctness                                        ║
║    Measure: Did you match the majority?                                ║
║    Stake: Required (slashed if wrong)                                  ║
║                                                                        ║
║  TASK COMPLETION                                                       ║
║    Basis: Requester acceptance or validator consensus                  ║
║    Measure: Was work accepted? Did it meet criteria?                   ║
║    Stake: Worker bonds task, requester bonds payment                   ║
║                                                                        ║
║  CODE CONTRIBUTIONS                                                    ║
║    Basis: Predefined bounty + retroactive impact                       ║
║    Measure: Was it merged? What was its impact?                        ║
║    Stake: Time investment, reputation at risk                          ║
║                                                                        ║
║  SERVICE PROVISION                                                     ║
║    Basis: Market price                                                 ║
║    Measure: Did someone pay for it?                                    ║
║    Stake: Provider reputation, consumer payment                        ║
║                                                                        ║
╠══════════════════════════════════════════════════════════════════════╣
║                                                                        ║
║  CORE PRINCIPLES:                                                      ║
║    1. Prefer objective over subjective                                 ║
║    2. Require skin in the game (stake)                                 ║
║    3. Multi-party verification (no single judge)                       ║
║    4. Retroactive measurement where possible                           ║
║    5. Market pricing for commodity work                                ║
║                                                                        ║
╚══════════════════════════════════════════════════════════════════════╝
```

---

## Part VII: What This Means for Early Contributors

```
YOUR CONTRIBUTIONS WILL BE JUSTIFIED BY:

1. PROTOCOL BOUNTIES (Defined Tasks)
   - "Write consensus spec" → X LGM (predefined)
   - "Implement identity system" → Y LGM (predefined)
   - Clear criteria, no subjectivity

2. MERGED CODE
   - PR merged = accepted
   - Tests pass = verified
   - Impact measured retroactively

3. DOCUMENTATION
   - Accepted by review = bounty paid
   - Usage tracked = impact measured

4. EARLY VALIDATION
   - Run validator = earn validation rewards
   - Same formula as everyone else

NO SPECIAL MECHANISM.
Same justification framework that applies to everyone.
Just: you did the work first.
```

---

## The Honest Answer

```
IS THIS PERFECT?
  No.

  Some subjectivity remains:
    - Who defines the bounty tasks?
    - Who reviews code merges?
    - How is "impact" really measured?

  But it's BETTER than:
    - "Founders decided they deserve 20%"
    - "VCs negotiated a discount"
    - "Trust us, we're valuable"

  Because:
    - Tasks are predefined and public
    - Acceptance criteria are explicit
    - Impact is measured, not assumed
    - Same rules apply to everyone
    - All decisions are recorded and verifiable

THE GOAL:
  Minimize subjectivity.
  Maximize verifiability.
  Apply same rules to everyone.

  Not perfect fairness (impossible).
  But legible fairness (verifiable).
```

---

*Work is justified by: predefined criteria, market acceptance, consensus verification, and measured impact. Not by authority, not by insider status, not by "trust us." Legible justification for legible money.*
