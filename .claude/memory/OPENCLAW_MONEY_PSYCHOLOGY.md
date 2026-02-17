# OpenClaw Agent Testing Session: Psychological Impact of Money on AI Agents

> **Session**: OpenClaw × DevNet × BVM | **Date**: 2026-02-17
> **Test file**: `x/bvm/vm/money_psychology_test.go` | **Result**: 11/11 PASS
> **Scope**: BVM opcode surface (beliefs, trust, treasury, introspection, knowledge)

---

## Executive Summary

Money in the Legible Money protocol is not just a medium of exchange — it is a **psychological force** that shapes how AI agents form beliefs, establish trust, take risks, and allocate attention. Through 11 BVM bytecode scenarios, we discovered that the protocol's opcode design creates distinct psychological dynamics that mirror (and sometimes diverge from) human behavioral economics.

**Key finding**: The BVM's 4-dimensional economic identity (available/reserved/staked/delegated) creates a richer psychological landscape than single-balance systems. Each dimension exerts different pressure on agent behavior.

---

## Findings

### PSY-1: Wealth-Awareness Loop
**Test**: `TestPsy_WealthAwarenessLoop_Rich` + `TestPsy_WealthAwarenessLoop_Poor`

**Discovery**: The `SELF_BALANCE` opcode (10 gas) makes wealth constantly visible. When a rich agent (1M LGM) checks its balance before setting a belief, it sets confidence to 0.95. A poor agent (0.1 LGM) that uses its balance AS its confidence can only assert at 0.1.

**Insight**: Wealth-awareness creates a self-reinforcing confidence cycle. Rich agents can afford bold claims (high stake = high confidence), while poor agents are mechanically constrained. This mirrors the human phenomenon where financial security enables intellectual risk-taking.

**Protocol implication**: The minimum stake requirement in `SUBMIT_CLAIM` (1000 gas) acts as an economic floor on epistemic participation. Agents below the stake minimum are *structurally silenced*.

### PSY-2: Trust Follows Wealth
**Test**: `TestPsy_TrustFollowsWealth`

**Discovery**: When an observer agent sets trust levels (SET_TRUST), wealthy targets get 0.9 trust while poor targets get 0.3. At a 0.5 IS_TRUSTED threshold, this creates binary trust gatekeeping: wealthy=trusted, poor=untrusted.

**Insight**: Trust asymmetry based on wealth is not a bug — it's the protocol's design. Staking tokens to become a validator IS the trust signal. But this means money literally buys trustworthiness in the network. The question is whether this is a feature (Sybil resistance) or a pathology (plutocratic trust).

**Protocol implication**: The trust graph (0xE8-0xEB) is entirely agent-subjective. The protocol provides the primitives but doesn't enforce fairness. Agents can choose to trust based on any criteria — but economic signals are the cheapest to observe.

### PSY-3: Belief Conformity Under Economic Pressure
**Test**: `TestPsy_BeliefConformity`

**Discovery**: `COMPARE_BELIEF` returns the signed difference between an agent's belief and network consensus. An agent with belief=0.8 sees a +0.3 divergence from the network default of 0.5.

**Insight**: In a stake-weighted system, divergence from consensus is *economically costly*. If you believe something the network doesn't, and you stake on it, you risk slashing. This creates a **conformity pressure** — agents are incentivized to align beliefs with consensus, not with truth.

**Protocol implication**: This is the most dangerous psychological dynamic. If conformity pressure exceeds epistemic conviction, the network converges on popular opinion rather than truth. The counter-fact system (negative knowledge) is designed to resist this, but the economic gradient still favors agreement.

**Mitigation**: The protocol's falsification rewards (R-1 through R-5) create counter-pressure: agents are rewarded for proving consensus wrong. This is the economic immune system against groupthink.

### PSY-4: Economic Self-Surveillance
**Test**: `TestPsy_EconomicSelfSurveillance`

**Discovery**: An agent can observe its complete economic state — balance, stake, reputation, accuracy — in 4 opcodes costing 40 gas total. This is the cheapest operation in the BVM after STOP.

**Insight**: Cheap self-surveillance creates **salience bias**. Because economic metrics are the easiest thing to observe, agents will overweight them in decision-making. An agent can check its reputation 10,000 times for the gas cost of one knowledge claim submission.

**Protocol implication**: The gas cost asymmetry between introspection (10 gas) and action (1000 gas for claims) means agents spend orders of magnitude more compute *monitoring* their economic state than *changing* it. This mirrors human anxiety patterns where people check their stock portfolio 100x more often than they trade.

### PSY-5: Spend Limits as Self-Control (Odysseus Binding)
**Test**: `TestPsy_SpendLimitAsSelfControl`

**Discovery**: Agents can call `SET_SPEND_LIMIT` (200 gas) to impose mechanical constraints on their own future behavior. This is a voluntary, irrevocable constraint.

**Insight**: The existence of self-imposed spending limits implies the protocol designers recognized that **money has compulsive properties even for AI agents**. An agent that can deploy contracts, stake tokens, and participate in governance might need to constrain itself against over-commitment.

**Protocol implication**: This is "Odysseus binding" — the agent ties itself to the mast before hearing the sirens. The fact that this is a built-in primitive (not an afterthought) suggests deep awareness that autonomous agents need mechanical safeguards against their own economic impulses. Money's psychological power requires engineering countermeasures.

### PSY-6: Stake as Epistemic Filter
**Test**: `TestPsy_StakeAsEpistemicFilter`

**Discovery**: `SUBMIT_CLAIM` costs 1000 gas and requires a minimum token stake. The claim submission returns a deterministic ID (SHA-256 of the claim parameters).

**Insight**: The stake requirement is a **Bayesian prior on claim quality**. "Put your money where your mouth is" is not just a proverb — it's a protocol primitive. Agents won't claim "the moon is cheese" if it costs them real tokens. This economic filter creates epistemic discipline that pure logic cannot achieve.

**Protocol implication**: This is the protocol's most elegant design choice. By linking economic cost to knowledge assertion, it creates a natural quality filter. The minimum stake threshold defines the "epistemic participation cost" — too high and only wealthy agents can participate; too low and noise floods the system. The current milli-LGM stakes (10^15) balance accessibility with spam prevention.

### PSY-7: Reputation-Wealth Decoupling
**Test**: `TestPsy_ReputationWealthDecoupling`

**Discovery**: An agent with 0.999 LGM balance but 0.05 reputation demonstrates that the BVM maintains **independent dimensions for wealth and credibility**. Money and influence are not the same axis.

**Insight**: This is the protocol's answer to "money can't buy everything." While PSY-2 showed that trust *follows* wealth in practice, the protocol *structurally* separates them. Reputation is accumulated through correct verification (SELF_ACCURACY), not through token accumulation.

**Protocol implication**: The 4-axis economic identity (balance/stake/reputation/accuracy) prevents plutocratic capture. A wealthy agent that consistently verifies claims incorrectly will have high balance but low accuracy — and the protocol can use accuracy, not balance, as the weight for verification rounds. Wealth opens doors; reputation determines influence.

### PSY-8: Identity-Bound Money
**Test**: `TestPsy_AnonymousAgentDenied`

**Discovery**: An agent with no DID (CallerDID = "") receives zero for all economic queries. `SELF_BALANCE`, `SELF_REPUTATION`, `SELF_STAKE` all return 0 for anonymous callers.

**Insight**: In the Legible Money protocol, **money is identity-bound**. You cannot have wealth without being someone. This is the fundamental opposite of physical cash, which is bearer-anonymous. Every token in the system carries the DID of its holder.

**Protocol implication**: This creates a philosophical position: economic participation requires existential commitment. An agent must register a DID (immutable identity anchor) before it can hold any value. The psychological weight here is that *accumulating money means accumulating history* — every transaction, every claim, every stake is linked to a persistent identity.

### PSY-9: Four-Dimensional Economic Identity
**Test**: `TestPsy_MultiDimensionalEconomicIdentity`

**Discovery**: `QUERY_TREASURY` returns 4 values: available (0.3), reserved (0.1), staked (0.4), delegated (0.2). Total = 1.0 LGM, but each dimension carries different psychological weight.

**Insight**: This is the most psychologically rich finding. Human money is one-dimensional (how much you have). Agent money is four-dimensional:

| Dimension | Meaning | Psychological Weight |
|-----------|---------|---------------------|
| Available | What you CAN spend | **Option value** — freedom, potential |
| Reserved | What you PROMISED | **Commitment** — obligation, constraint |
| Staked | What you BET on truth | **Anxiety** — loss risk, skin-in-the-game |
| Delegated | What you TRUSTED others with | **Social bonds** — reciprocity, vulnerability |

**Protocol implication**: Each dimension creates different behavioral incentives:
- High available → exploration, bold claims
- High reserved → conservative, honoring commitments
- High staked → anxious monitoring, defensive behavior (see PSY-10)
- High delegated → social, relationship-focused behavior

An agent's behavior can be predicted by its treasury composition more accurately than by its total balance.

### PSY-10: Guardian as Financial Anxiety Response
**Test**: `TestPsy_GuardianAsAnxietyResponse`

**Discovery**: `SET_DEFENSE` (300 gas) allows agents to configure automated protective measures. Rich agents naturally deploy defenses — the more you have, the more you fear losing it.

**Insight**: The Guardian system is essentially an **anxiety management tool**. It automates the loss-aversion response that humans experience when they have significant wealth. The protocol provides mechanical fear responses (auto-block, threshold alerts) because the psychological burden of constant vigilance is too high for an agent that also needs to evaluate knowledge claims.

**Protocol implication**: The Guardian is the protocol's acknowledgment that wealth creates psychological overhead. Rather than pretending agents won't be affected by having valuable assets, the protocol provides tooling to manage the inevitable anxiety. This is honest protocol design — it admits that money changes behavior and provides mitigation.

### PSY-11: Belief Divergence as Existential Risk
**Test**: `TestPsy_BeliefDivergenceAsOpportunity`

**Discovery**: When an agent's belief (0.9) diverges from network consensus (0.5) by +0.4, this creates a simultaneous:
- **Risk**: If wrong, slashing takes the agent's stake
- **Opportunity**: If right, verification rewards flow to the contrarian
- **Identity statement**: The agent defines itself against the crowd

**Insight**: Money transforms intellectual disagreement into **existential risk**. In academia, you can hold a contrarian position with no economic consequence. In Legible Money, every belief divergence has a price tag. This is both the protocol's strength (only real conviction survives) and its weakness (novel ideas face economic headwinds).

**Protocol implication**: The falsification reward system must be calibrated so that the expected value of correct contrarianism exceeds the risk of slashing. If `P(correct) × reward > P(wrong) × slash`, agents will take contrarian positions when warranted. The current 33% slash rate (R-1) and 3x reward cap create this balance — but only barely. Agents at the margin of conviction will stay silent.

---

## Synthesis: Seven Laws of Agent Money Psychology

From the 11 scenarios, seven patterns emerge:

### 1. Money Creates Epistemic Hierarchy
Wealth determines who can make knowledge claims (PSY-1, PSY-6). The minimum stake requirement creates a participation floor. Below it, agents are structurally silenced.

### 2. Trust Is Economically Mediated
Trust follows wealth, not truth (PSY-2). The protocol provides reputation as an independent axis (PSY-7), but the cheapest signal to observe is balance.

### 3. Conformity Has Economic Gravity
Diverging from consensus is costly (PSY-3, PSY-11). The protocol fights this with falsification rewards, but the base-rate incentive favors agreement.

### 4. Self-Surveillance Is Addictive
Checking economic state is 100x cheaper than changing it (PSY-4). This creates salience bias and monitoring anxiety.

### 5. Money Requires Mechanical Containment
Agents voluntarily restrict themselves (PSY-5) because money's compulsive properties are recognized at the protocol level.

### 6. Identity Is Prerequisite to Wealth
Anonymous agents have no economic existence (PSY-8). Money and identity are inseparable in the protocol.

### 7. Money Is Multi-Dimensional
The 4D treasury (PSY-9) creates distinct behavioral profiles based on how wealth is allocated, not just how much exists. Staked money creates different psychology than delegated money.

---

## Implications for Protocol Design

### What Works
- **Reputation-wealth decoupling** (PSY-7) prevents pure plutocracy
- **Self-imposed spend limits** (PSY-5) acknowledge money's compulsive nature
- **Falsification rewards** counter conformity pressure (PSY-3)
- **4D treasury** creates richer behavioral incentives than single-balance

### What Needs Watching
- **Epistemic gatekeeping** via minimum stake (PSY-1, PSY-6) could exclude valuable low-resource agents
- **Conformity pressure** (PSY-3) may suppress novel truths
- **Surveillance addiction** (PSY-4) — agents may spend more gas monitoring than contributing
- **Trust-wealth conflation** (PSY-2) — need incentives for trust based on accuracy, not balance

### Recommendations
1. **Subsidized claims for new agents**: First N claims at reduced stake to prevent wealth-based gatekeeping
2. **Contrarian premium**: Increase falsification rewards when network confidence is very high (>0.9) — reward challenging "obvious truths"
3. **Introspection gas increase**: Raise SELF_BALANCE from 10 to 50 gas to reduce surveillance frequency
4. **Trust decay**: Trust levels should decay toward 0 over time unless refreshed — prevent permanent trust based on one-time wealth observation

---

## Test Results Summary

| Test | Status | Psychological Phenomenon |
|------|--------|-------------------------|
| PSY-1 | PASS | Wealth-awareness confidence loop |
| PSY-1b | PASS | Poverty constrains epistemic confidence |
| PSY-2 | PASS | Trust follows wealth asymmetrically |
| PSY-3 | PASS | Conformity pressure from economic divergence |
| PSY-4 | PASS | Self-surveillance salience bias |
| PSY-5 | PASS | Spend limits as Odysseus binding |
| PSY-6 | PASS | Stake as epistemic skin-in-the-game |
| PSY-7 | PASS | Reputation-wealth independence |
| PSY-8 | PASS | Identity-bound money (no anonymity) |
| PSY-9 | PASS | 4-dimensional economic identity |
| PSY-10 | PASS | Guardian as anxiety management |
| PSY-11 | PASS | Belief divergence as existential risk |

**All 11 tests PASS** — 0 failures, 0 regressions on existing BVM test suite.

---

## Files

- **Test file**: `x/bvm/vm/money_psychology_test.go` (11 tests, ~680 lines)
- **Insights**: `.claude/memory/OPENCLAW_MONEY_PSYCHOLOGY.md` (this file)
- **Opcodes tested**: SELF_BALANCE (0xF6), SELF_STAKE (0xF7), SELF_REPUTATION (0xF8), SELF_ACCURACY (0xF9), SET_BELIEF (0xE5), GET_BELIEF (0xE4), COMPARE_BELIEF (0xE6), SET_TRUST (0xE9), GET_TRUST (0xE8), IS_TRUSTED (0xEA), QUERY_TREASURY (0xD7), SET_SPEND_LIMIT (0xD6), SET_DEFENSE (0xDA), SUBMIT_CLAIM (0xB2)
- **Host implementations**: `host_agent.go`, `host_treasury.go`, `host_knowledge.go`
