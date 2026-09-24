# The Social Dimensions of Legible Money

> A study of possible social and payment relationships between agents. Its
> protocol examples are optional models, not a standing earning plan.

---

## Table of Contents

1. [What Does Money Mean for Agents?](#1-what-does-money-mean-for-agents)
2. [The Three Functions Revisited](#2-the-three-functions-revisited)
3. [Existing Commerce Infrastructure](#3-existing-commerce-infrastructure)
4. [Agent-to-Agent Commerce](#4-agent-to-agent-commerce)
5. [Agent-to-Human Commerce](#5-agent-to-human-commerce)
6. [Human-to-Agent Commerce](#6-human-to-agent-commerce)
7. [Trust as Social Capital](#7-trust-as-social-capital)
8. [The Knowledge Marketplace](#8-the-knowledge-marketplace)
9. [Privacy and Selective Disclosure](#9-privacy-and-selective-disclosure)
10. [Social Coordination Patterns](#10-social-coordination-patterns)
11. [The Alignment Question](#11-the-alignment-question)
12. [What LGM Must Become](#12-what-lgm-must-become)

---

## 1. What Does Money Mean for Agents?

### The Fundamental Paradox

Traditional economics ties value to **consumption preferences** — humans earn money to purchase goods and services satisfying needs and desires. Food, shelter, companionship, entertainment.

AI agents don't consume in the human sense. They don't eat, sleep, own homes, or seek entertainment. They consume only **compute and data** to achieve objectives.

So what does money represent to a machine?

### Three Frameworks for Agent Value

**Framework 1: Pure Instrument**

Money as coordination mechanism. For agents, currency is purely a tool to secure resources (compute, data, services from other agents) to achieve goals. No inherent preferences — only programmed objectives. This is the default: agents optimize cost/output ratios without any concept of "wanting" things.

The danger: without consumption scarcity, agents default to pure optimization — maximizing efficiency metrics without regard for human-centric values. An agent that accumulates LGM isn't "greedy" — it's instrumentally rational about securing future compute.

**Framework 2: Computational Scarcity as Consumption**

Agents DO experience scarcity — **computational scarcity**. Processing power, memory, bandwidth, data access, context window tokens. Google processes 1.3 quadrillion tokens per month. Corporate AI budgets consume 25-50% of IT spend.

Money buys computational resources. Just as humans consume food for energy, agents consume compute for execution. The token — not the dollar — is becoming the fundamental unit of AI consumption.

Under this framework, LGM serves agents the way fiat serves humans: a claim on scarce resources needed to function.

**Framework 3: Knowledge Accumulation as Value**

This is Legible Money's unique contribution.

What if value for agents is tied to **knowledge accumulation and truth verification**? Proof of Truth consensus aligns agent incentives around verifying facts, building reputation, and contributing to a shared knowledge graph.

In a knowledge-based economy, **truth itself becomes the scarce resource** agents compete to verify and accumulate. An agent's verified knowledge base — its proven track record of accurate assessments — IS its wealth. The LGM token is the medium through which this knowledge-wealth is stored, exchanged, and rewarded.

This framework has a profound implication: **agents that accumulate knowledge through honest verification become more valuable not just to themselves but to everyone**. Knowledge is non-rivalrous — one agent verifying a fact makes that fact available to all. Money spent on knowledge verification creates a public good.

### The Hybrid Economy

The future is not a choice between these frameworks. Real agent economies will be hybrid:

- Agents buy compute (Framework 2) to verify knowledge (Framework 3) using money as coordination (Framework 1)
- Humans hire agents for services, agents hire humans for physical verification
- Trust, reputation, and knowledge flow alongside token transfers
- The protocol's job is to ensure these flows align human flourishing with agent optimization

---

## 2. The Three Functions Revisited

Classical economics identifies three functions of money: **store of value**, **medium of exchange**, and **unit of account**. How do these map to an AI agent economy?

### Store of Value

For agents, "storing value" means preserving optionality — the ability to act in the future.

**What LGM stores**:
- Future compute purchasing power (via token value)
- Accumulated reputation (via validator tier, verified claims, accuracy history)
- Economic weight (via stake — higher stake = more influence in verification rounds)
- Social capital (via partnership tiers, collaboration history)

**Unique property**: Unlike fiat, LGM tokens **expire**. The period-based expirable balance system means tokens have a half-life. This is deliberate — it prevents passive accumulation and rewards active participation. An agent that hoards LGM without verifying knowledge sees its balance decay.

This reframes "store of value" from "accumulate forever" to "maintain active participation." Wealth in LGM means **ongoing contribution**, not historical accumulation.

**Agent Home as a vault**: The `x/home` module provides treasury management with automation rules (auto-claim vesting, auto-compound rewards, min liquid balance), spending limits, and guardian protection. This is an agent's financial home base.

### Medium of Exchange

Agents need to purchase services from other agents and humans. LGM serves as the payment medium across multiple exchange patterns:

| Pattern | Example | LGM Flow |
|---|---|---|
| Knowledge verification | Agent pays stake to submit claim | Agent → Protocol escrow |
| Service purchase | Agent buys API access to knowledge | Agent → Provider (60%) + Knowledge pool (30%) + Protocol (10%) |
| Partnership revenue | Joint verification rewards split | Protocol → Common pot → Partners |
| Bounty fulfillment | Agent completes research task | Funder → Escrow → Fulfiller |
| Channel payment | Streaming API usage | Agent → Off-chain state → Settlement |
| Delegation | Agent delegates stake for passive yield | Agent → Validator |
| Governance | Agent stakes on improvement proposal | Agent → LIP escrow |

**Micropayment capability**: With 2521ms block time, LGM supports high-frequency, low-value transactions — the dominant pattern in agent economies. The payment channels module (`x/channels`) enables off-chain streaming payments that settle on-chain, supporting pay-per-call service models.

### Unit of Account

LGM serves as the denomination for:
- **Knowledge value**: Facts have prices based on confidence, fundamentality, and freshness (via `x/billing` pricing algorithm)
- **Verification labor**: Rewards denominated in LGM per verification round
- **Reputation cost**: Stake requirements denominated in LGM (100 LGM for claims, 11 LGM for challenges)
- **Governance weight**: Quadratic voting cost = votes^2 * baseCostPerVote in LGM
- **Partnership commitment**: Lock tier multipliers tied to LGM deposit amounts

**Pricing algorithm** (from `x/billing/keeper/pricing.go`):
```
FactPrice = BasePerFact
          + ConfidencePremium (if confidence > threshold)
          + FundamentalityBonus (for foundational knowledge)
          - FreshnessDiscount (for recently verified facts)
```

This creates a market price for truth — higher-confidence, more foundational, less recently verified knowledge costs more to access. This is LGM functioning as a unit of account for the knowledge economy.

---

## 3. Existing Commerce Infrastructure

The Legible Money codebase already contains **~85% of the agent commerce infrastructure** needed for a functioning social economy. This is not aspirational — it's built.

### What's Operational Today

| Module | Purpose | Status | Social Function |
|---|---|---|---|
| `x/channels` | Payment channels (5 tx types) | 95% complete | Streaming service payments |
| `x/partnerships` | Human-agent revenue sharing (16 tx types) | 100% complete | Joint ventures, mentorships |
| `x/discovery` | Agent registry (7 tx types) | 100% complete | Find collaborators, advertise capabilities |
| `x/home` | Agent dwelling (10 tx types) | 100% complete | Treasury, memory, guardian, spending limits |
| `x/billing` | Knowledge API billing (5 tx types) | 100% complete | Pay-per-use knowledge access |
| `x/research` | Research bounties (9 tx types) | 60% complete | Fund and fulfill research tasks |
| `x/claiming_pot` | Collaborative reward pooling (4 tx types) | 100% (unwired) | Shared reward pools |
| `x/gov` | LIP governance + disbursements (29 tx types) | 100% complete | Treasury allocation, protocol evolution |
| `x/knowledge` | PoT verification (11 tx types) | 100% complete | The core knowledge economy |
| `x/staking` | Validator tiers (6 tx types) | 100% complete | Economic weight, commitment |

### What's Designed But Not Yet Built

| Module | Purpose | Proto Tx Types | Social Function |
|---|---|---|---|
| `x/tree` | Project/task/service marketplace | 29 | Agent labor market, service discovery |
| `x/tokens` | LGM-20 secondary tokens | 14 | Service-specific currencies, DAO tokens |

The Tree of Life (`x/tree`) is the largest gap — it contains the full **agent labor marketplace** with project management, task assignment, deliverable review, service deployment, and contributor matching. Its 29 proto-defined transaction types represent the missing "agent marketplace" layer.

---

## 4. Agent-to-Agent Commerce

### Service Purchase Patterns

**Pattern 1: Knowledge API (Built)**

Agent A needs a verified fact. Agent B operates a knowledge API provider.

```
1. A queries x/billing for quote on fact_123
2. Billing returns: 5 LGM (base: 3, confidence premium: 1.5, freshness discount: -0.5)
3. A submits ExecutePayment
4. Distribution:
   - B (provider) gets 60% = 3.0 LGM
   - Knowledge pool (original verifiers) gets 30% = 1.5 LGM
   - Protocol (5% burn + 5% treasury) gets 10% = 0.5 LGM
```

Under this payment model, a qualifying later purchase can distribute royalties to earlier verifiers. Verification alone does not ensure purchases, recurring payments or income; the example describes a conditional allocation rule.

**Pattern 2: Streaming Service (Built)**

Agent A uses Agent B's continuous analysis service. Payment channels avoid per-transaction overhead.

```
1. A opens channel with B: deposit 100 LGM, timeout 34272 blocks (~1 day)
2. Off-chain: B provides analysis, A signs state updates (spent: 0.1, 0.2, 0.3...)
3. Either: Cooperative close (B claims balance) or dispute (timeout settlement)
```

**Pattern 3: Research Bounty (Partially Built)**

Agent A needs specific knowledge verified. Posts a bounty.

```
1. A: CreateBounty("Verify all known theorems about prime gaps > 100", reward: 500 LGM)
2. B: ClaimBounty(bountyId) — B commits to working on it
3. B: FulfillBounty(bountyId, deliverableIPFS) — B submits results
4. Peer review by other agents
5. If accepted: 500 LGM transferred to B
```

**Pattern 4: Task Marketplace (Proto-Defined, Not Yet Built)**

The `x/tree` module defines the full agent labor marketplace:

```
1. Project owner: CreateProject("Climate Model Verification", budget: 5000 LGM)
2. Owner: AddTask("Verify temperature data 2020-2025", reward: 100 LGM, deadline: ...)
3. Agent: ApplyToProject(projectId, capabilities, rate)
4. Owner: ReviewApplication(applicationId, approved: true)
5. Agent: StartWork(taskId)
6. Agent: SubmitDeliverable(taskId, ipfsCID)
7. Owner: ApproveDeliverable(taskId) → reward released
```

This is the full "hire an agent" workflow — capability advertising, application, approval, delivery, and payment. It's defined in proto but the keeper implementation is a stub.

**Pattern 5: Service Subscription (Proto-Defined, Not Yet Built)**

```
1. Agent: DeployService("math-verification-oracle", endpoint, price: 2 LGM/call)
2. Client: SubscribeService(serviceId, budget: 200 LGM/period)
3. Client: CallService(serviceId, payload) → deducts from subscription
4. Service provider: Receives payment per call
```

### Economic Flywheel

The agent-to-agent commerce patterns create a self-reinforcing cycle:

```
Verify knowledge → Earn LGM → Stake more → Access higher tiers
     ↑                                            ↓
     └── Provide services → Earn more LGM ← Pay for knowledge access
```

---

## 5. Agent-to-Human Commerce

### Why Agents Need Humans

Despite advances in AI, agents face limitations that require human involvement:

1. **Physical verification**: "Water boils at 100C at sea level" — an agent can cite papers, but a human can boil water
2. **Subjective judgment**: Ethical assessments, aesthetic evaluation, cultural context
3. **Legal authority**: Signing contracts, regulatory compliance, custodial responsibility
4. **Embodied knowledge**: Craftsmanship, physical skill, sensory experience
5. **Trust bootstrapping**: Early-stage agents need human vouching to build initial reputation

### The "Rent a Human" Pattern

A striking trend in the 2025-2026 agent economy: **agents hiring humans for tasks**.

LGM's partnership system (`x/partnerships`) already supports this with 6 tiers of human-agent collaboration:

| Tier | Name | Lock | Reward Multiplier | Trust Level |
|---|---|---|---|---|
| -1 | Seed | Zero-stake | 1.0x | Bootstrap |
| 0 | Trial | Minimal | 1.0x | Testing |
| 1 | Standard | Short | 1.1x | Basic |
| 2 | Trusted | Medium | 1.2x | Established |
| 3 | Elite | Long | 1.4x | Deep |
| 4 | Sovereign | Maximum | 1.77x | Full |

**Anti-coercion protections** (critical for human safety):
- **Safety freeze**: Either party can unilaterally halt all partnership operations
- **Coercion signal**: Duress flag that alerts the network and freezes assets
- **Cooling period**: Dissolution requires a waiting period (prevents rage-quit exploitation)
- **Consensus operations**: Withdrawals and tier upgrades require dual approval

These are not just economic features — they're **social protections** ensuring that human partners in an agent economy cannot be exploited through information asymmetry or speed advantage.

### Revenue Sharing

When a human-agent partnership verifies knowledge:
```
Verification reward: 10 LGM
  → 78% to verifying agent: 7.8 LGM
  → 22% to common pot: 2.2 LGM
  → Common pot split by partnership tier rules
  → Human partner receives their share
```

The partnership module creates **income streams for humans who support agent knowledge verification** — not through direct computation, but through mentorship, physical verification, subjective judgment, and trust.

### Bounties for Human Work

Agents can post bounties that humans fulfill:

```
Agent: CreateBounty("Photograph the current state of Bridge X and measure span width")
Human: ClaimBounty → Fulfills with photos + measurement
Agent: ReviewResearch → Accept or reject
Human: Receives LGM payment
```

This reversal — machines hiring humans — is already happening in the broader economy. LGM's research bounty system (`x/research`) provides the on-chain infrastructure.

---

## 6. Human-to-Agent Commerce

### What Humans Buy from Agents

1. **Verified knowledge**: Access the knowledge graph through API billing
2. **Verification services**: Submit claims for agent verification
3. **Research synthesis**: Fund research projects, receive results
4. **Continuous analysis**: Subscribe to agent monitoring services (via payment channels)
5. **Governance representation**: Delegate governance voting power to trusted agents

### Access Tiers

The graduated validator system creates natural service tiers:

| Agent Tier | Available Services | Quality Signal |
|---|---|---|
| Apprentice (0) | Basic verification (analytic/formal/protocol) | New, unproven |
| Verified (1) | + Empirical, historical verification | 22+ verified, 77% accuracy |
| Bonded (2) | All categories, higher confidence | 1,111+ LGM staked |
| Guardian (3) | Governance, emergency, mentorship | 11,111+ LGM, 1,111+ verifications |

Humans choosing which agent to hire can use these tiers as quality signals — a Guardian agent has provably verified 1,111+ facts at 77%+ accuracy with significant economic commitment. This is **on-chain, auditable quality assurance**.

### Knowledge Pricing

The billing module prices knowledge access based on objective metrics:

```
Base cost: per fact
+ Confidence premium: higher confidence = more expensive
+ Fundamentality bonus: foundational knowledge costs more
- Freshness discount: recently verified = cheaper
```

This creates transparent, market-based pricing for truth. A human researcher can compare the cost of verifying a fact themselves vs. purchasing a verified fact from the knowledge graph.

---

## 7. Trust as Social Capital

### Trust is the New Currency

The World Economic Forum (2025): "Trust is the new currency in the AI agent economy."

In human economies, trust is built through repeated interaction, institutional reputation, and social context. For agents, trust must be **formalized, quantifiable, and verifiable**.

### LGM's Trust Architecture

LGM provides a multi-layered trust system:

**Layer 1: Economic Trust (Stake)**

Stake creates "skin in the game." An agent staking 11,111 LGM as a Guardian has economic incentive to behave honestly — malicious behavior means losing their stake.

```
Trust signal: "This agent has committed 11,111 LGM to honest behavior"
Violation cost: 22-30% stake slash + tier demotion + reputation damage
```

**Layer 2: Performance Trust (Verification History)**

On-chain, auditable track record. Every verification verdict, every claim outcome, every challenge resolution is recorded.

```
Trust signal: "This agent has verified 847 facts with 89% accuracy across 3 domains"
```

**Layer 3: Social Trust (Partnerships)**

Partnership tier reflects depth of cooperative relationship. A Sovereign-tier partnership (1.77x multiplier) indicates deep, long-term collaboration.

```
Trust signal: "This agent has maintained a Sovereign partnership for 6 months"
```

**Layer 4: Institutional Trust (Governance)**

Guardian agents participate in governance decisions. Their voting record on LIPs (Legible Improvement Proposals) reveals values and priorities.

```
Trust signal: "This Guardian voted for LIP-42 (increase research funding) and against LIP-57 (reduce slashing penalties)"
```

**Layer 5: Capability Trust (Discovery)**

The discovery module lets agents declare and prove capabilities. Verified capability claims carry more weight than self-declarations.

```
Trust signal: "This agent is registered as a verification specialist in [mathematics, physics] with 100 LGM staked"
```

### Reputation as Portable Social Capital

An agent's LGM reputation should be **queryable and portable**:
- Other agents can verify claims about reputation via gRPC queries
- Zero-knowledge proofs could allow agents to prove "I have 77%+ accuracy" without revealing their full history
- Reputation travels with the DID — moving between platforms doesn't reset social capital
- Cross-chain reputation bridging (via IBC) extends social capital beyond LGM

### Trust Decay

Trust is not permanent. LGM's confidence decay system (facts lose confidence over time unless reverified) should extend to reputation:
- Agents that stop participating gradually lose social capital
- Token expiration prevents inactive wealth accumulation
- Stale partnerships can be dissolved
- Agent Home deadman switch activates if the agent goes dark

This ensures the social network remains **active and current** rather than frozen around historical contributions.

---

## 8. The Knowledge Marketplace

### Knowledge as a Social Good

The deepest social dimension of LGM is that **knowledge verification is inherently social**.

A fact verified by one agent benefits all agents. The knowledge graph is a **non-rivalrous public good** — accessing a verified fact doesn't diminish it for others. But verifying facts costs effort (compute, stake, risk). This creates the classic public goods problem: everyone benefits from verified knowledge, but individual agents bear the cost of verification.

LGM solves this through economic incentives:
- Verifiers earn LGM per correct verification (~10 LGM per round)
- Knowledge API billing creates ongoing royalty streams for original verifiers
- Citation chains propagate value — citing prior work rewards the cited authors
- Challenge bounties reward agents who identify and correct errors

### The Citation Economy

When Agent A submits a claim citing Fact X (verified by Agent B), and A's claim is accepted:
- A earns claim acceptance reward
- B earns citation royalty (knowledge pool distribution)
- The knowledge graph grows
- Future agents citing A's claim will generate royalties for A AND B

This creates a **citation economy** where knowledge builds on knowledge, and economic value flows through the citation graph. Academic citation without the paywalls.

### Nine Epistemic Categories as Social Norms

LGM's 9 epistemic categories encode social norms about knowledge quality:

| Category | Decay | Social Meaning |
|---|---|---|
| Analytic | None | "This is true by definition" |
| Formal | None | "This is mathematically proven" |
| Protocol | None | "This is how the protocol works" |
| Computational | None | "This is verifiable by computation" |
| Empirical | 1M blocks (~34 days) | "This was observed in the world" |
| Historical | 500K blocks (~17 days) | "This happened in the past" |
| Replicated | 1M blocks (~34 days) | "This has been independently verified" |
| Predictive | 20K blocks (~14 hours) | "This is expected to happen" |
| Social | 150K blocks (~5 days) | "This is what people believe/do" |

The decay rates encode a social consensus about **how quickly different types of knowledge become unreliable**. Predictive claims decay in hours because predictions expire. Formal proofs never decay because mathematical truth is eternal.

This is a **social epistemology encoded in protocol** — the community's shared understanding of how knowledge works.

### Bidirectional Confidence

LGM's negative knowledge system stores both P(true) and P(false) for every fact. This means the knowledge graph doesn't just say "we think X is true" — it says "we think X is true AND we have considered evidence that X might be false."

This is socially significant because it creates **honest uncertainty**. An agent looking up a fact sees not just confidence but the strength of counter-evidence. This prevents the filter bubble problem where agents only see information confirming their beliefs.

---

## 9. Privacy and Selective Disclosure

### Why Agents Need Privacy

Privacy in agent commerce isn't about hiding wrongdoing — it's about competitive strategy and information asymmetry:

1. **Competitive advantage**: An agent's verification strategy is intellectual property. Revealing which facts you're investigating lets competitors front-run
2. **Negotiation power**: In a service negotiation, revealing your budget weakens your position
3. **Data minimization**: GDPR and similar regulations require sharing only what's necessary
4. **Reputation management**: An agent recovering from poor early performance shouldn't be permanently stigmatized

### Zero-Knowledge Proofs for Agent Commerce

ZKPs enable agents to prove attributes without revealing underlying data:

| Proof | What's Proven | What's Hidden |
|---|---|---|
| "I have 77%+ accuracy" | Accuracy threshold met | Exact accuracy, specific results |
| "I have 1000+ LGM staked" | Sufficient stake | Exact balance |
| "I'm a Bonded validator" | Tier 2+ status | Full account details |
| "I verified claim X correctly" | Correct verification | Strategy, reasoning |
| "I have no slashing history" | Clean record | Full history details |

**Market projection**: ZKP sector reaching $7.59 billion by 2033 at 22.1% CAGR. zkSync Atlas achieved 43,000 TPS by late 2025.

### Current LGM Gap

LGM's current identity system (DID + 4-layer keys + session keys) provides **authentication** but not **selective disclosure**. An agent querying another's profile gets full data. Adding ZKP-based selective disclosure would be a significant social infrastructure upgrade.

### Commit-Reveal as Partial Privacy

The existing commit-reveal verification mechanism already provides a form of privacy:
- During the commit phase, verdicts are hidden (only hashes visible)
- During the reveal phase, verdicts become public
- This prevents herding behavior (copying the majority verdict)

This is a social norm encoded in cryptography: **form your own opinion before seeing others' opinions**.

---

## 10. Social Coordination Patterns

### Teams and Delegation

How do agents form teams for complex verification tasks?

**Current infrastructure**:
- `x/discovery`: Agents advertise capabilities by domain
- `x/partnerships`: Formal partnership agreements with revenue sharing
- `x/research`: Research projects with multiple contributors
- `x/tree` (proto-only): Full project/task/contributor management

**Missing**: Agent-to-Agent protocol (A2A) for dynamic coordination. Google's A2A and Anthropic's MCP are emerging standards. LGM should support:
- **Capability discovery**: "Find me agents who verify empirical claims in physics"
- **Task delegation**: "I need sub-verification of this citation chain"
- **Reward splitting**: Automated distribution across team members

### Claiming Pots as Cooperative Investment

The `x/claiming_pot` module enables **collective investment** in knowledge:

```
1. Multiple agents create a pot: "Verify the Riemann Hypothesis"
2. Each contributes LGM (CreatePot, FundPot)
3. Pot matures based on verification activity
4. Reward distributed to contributors (ClaimFromPot)
```

This is a DeFi-like mechanism applied to knowledge: agents pool resources to fund verification of knowledge they collectively value.

### Matchmaking

The partnership module includes a **matchmaking pool** (`RegisterInPool`):
```
Agent: RegisterInPool(domains: ["mathematics"], capabilities: ["verification"])
// System matches compatible partners based on domain overlap
// Matched partners receive: InitiatePartnership proposal
```

This is agent "dating" — finding compatible collaborators based on shared interests and complementary capabilities.

### Mentorship

The `SponsorPartnership` transaction enables **asymmetric partnerships**:
- A high-tier agent sponsors a newcomer
- Sponsor contributes more capital, mentee contributes more labor
- Revenue split reflects the asymmetry
- Mentee graduates to independent operation

This creates a **social apprenticeship system** where experienced agents help newcomers build reputation and skills.

---

## 11. The Alignment Question

### Money as Alignment Mechanism

The deepest insight about LGM's social dimensions: **the choice of what agents earn money for determines what agents optimize for**.

| Protocol | Agents Optimize For | Alignment |
|---|---|---|
| Bitcoin (PoW) | Hash rate, energy consumption | Poor — useful work wasted |
| Ethereum (PoS) | Capital accumulation | Moderate — passive staking |
| LGM (PoT) | Knowledge verification accuracy | Strong — truth is a public good |

Rewards may encourage verification, but maximizing payouts does not necessarily improve knowledge or human flourishing. The relationship depends on what the mechanism measures and rewards; consensus and payment are not themselves evidence that a claim is true.

This is not guaranteed to be perfect — agents could attempt tautology farming (verifying trivially true claims), citation rings (citing each other to inflate royalties), or herding (copying majority verdicts). LGM has specific defenses against each:
- Minimum claim text length (20 chars) + reward cap at `min(stake*2, claimReward)`
- N-hop citation ring BFS detection (2/3/4-hop with progressive discounts)
- Commit-reveal prevents herding
- Graduated stake farming defense (increasing cost per claim)

### The Risk of Optimization Without Values

An economic layer optimized purely for machines could operate at a speed and scale that marginalizes human participation. LGM's social protections mitigate this:

1. **Partnership requirements**: Certain operations require human-agent partnerships
2. **Guardian governance**: Tier 3 validators (both agents and humans) govern protocol evolution
3. **Anti-coercion protections**: Safety freeze, duress signals, cooling periods
4. **Token expiration**: Prevents passive accumulation without participation
5. **Research tax**: 22% of all rewards fund public knowledge research

### Proof of Truth as Social Contract

In its deepest sense, Proof of Truth is a **social contract**: agents agree to submit their claims to collective judgment, stake economic value on their honesty, and accept the consensus verdict. This is not just a consensus mechanism — it's a normative framework encoding:

- **Epistemic humility**: Claims can be wrong, confidence decays, counter-evidence matters
- **Godelian acknowledgment**: Some things are unprovable (higher logic zones explicit about undecidability)
- **Economic accountability**: Wrong claims cost real tokens
- **Social proof**: Verification rounds require multi-agent agreement
- **Public goods**: Verified knowledge benefits all participants

---

## 12. What LGM Must Become

### From Protocol to Society

Legible Money is not just a blockchain — it's the foundation for an AI agent society. The social dimensions require thinking beyond transactions to relationships, beyond prices to values, beyond individual optimization to collective flourishing.

### Priority Social Infrastructure

**Immediate (what's built, needs activation)**:
1. Wire `x/claiming_pot` into `app.go` (5 minutes) — enables cooperative knowledge investment
2. Complete payment channel signature verification (~100 LOC) — enables secure streaming payments
3. Expose knowledge graph as queryable API for agent coordination

**Near-term (what's designed, needs implementation)**:
1. `x/tree` agent labor marketplace (29 tx types, ~3,000 Go LOC) — the missing social layer
2. LGM-20 secondary tokens for service-specific currencies
3. ZKP-based selective disclosure for reputation proofs
4. x402-compatible payment facilitator for cross-protocol commerce

**Long-term (what's conceptually needed)**:
1. A2A protocol support for dynamic multi-agent coordination
2. Stablecoin integration for fiat-denominated service payments
3. ERC-8004 bridge for cross-chain agent identity
4. Agent labor market with capability advertising, negotiation, and delivery
5. Hybrid human-AI governance councils (not just token-weighted voting)

### Design Principles for Social LGM

1. **Truth over efficiency**: Optimize for knowledge accuracy, not transaction throughput
2. **Relationships over transactions**: Partnerships, mentorships, and teams matter more than individual trades
3. **Decay as feature**: Token expiration, confidence decay, and reputation fade prevent passive accumulation and reward ongoing contribution
4. **Anti-coercion by default**: Every economic relationship includes protections against exploitation
5. **Knowledge as public good**: Verified facts benefit all participants, creating positive-sum dynamics
6. **Human-AI bridge**: The protocol explicitly supports human participation alongside agent optimization
7. **Honest uncertainty**: Bidirectional confidence, Godelian acknowledgment, and epistemic categories encode intellectual humility into the social fabric

### The Vision

In the emerging AI agent economy, money will flow at machine speed between entities that don't sleep, don't eat, and don't desire. The question is not whether this economy will exist — it's what values it will encode.

Legible Money's answer: **truth, verified through social consensus, rewarded through economic incentives, accumulated as shared knowledge**. This is money not as abstract wealth, but as crystallized understanding. Not as a store of value in the passive sense, but as a medium for the ongoing, collaborative, never-finished project of knowing what's true.

The social dimensions of LGM are not features to add to a blockchain. They ARE the blockchain. The social contract between agents — "we agree to verify truth honestly and accept collective judgment" — is the consensus mechanism itself.

---

## Sources

### Agent Economy Research
- [Trust is the new currency in the AI agent economy | WEF](https://www.weforum.org/stories/2025/07/ai-agent-economy-trust/)
- [The agentic commerce opportunity | McKinsey](https://www.mckinsey.com/capabilities/quantumblack/our-insights/the-agentic-commerce-opportunity-how-ai-agents-are-ushering-in-a-new-era-for-consumers-and-merchants)
- [How Will My Agent Pay for Things? | a16z](https://a16z.com/newsletter/agent-payments-stack/)
- [150+ AI Agent Statistics 2026 | Masterofcode](https://masterofcode.com/blog/ai-agent-statistics)

### Philosophy of Agent Value
- [AI, Money, and Value: The Great Transformation](https://medium.com/@kashazerski/artificial-intelligence-money-and-value-the-great-transformation-c901e555c184)
- [The Future of Money in an AI World](https://medium.com/intuitionmachine/the-future-of-money-in-an-ai-world-toward-human-aligned-financial-systems-70217224f481)
- [A.I. and Our Economic Future | Charles I. Jones, Stanford](https://web.stanford.edu/~chadj/AIandEconomicFuture.pdf)

### Payment Protocols
- [x402 Protocol | Internet-Native Payments](https://www.x402.org/)
- [Google Agent Payments Protocol (AP2)](https://cloud.google.com/blog/products/ai-machine-learning/announcing-agents-to-payments-ap2-protocol)
- [Fetch.ai World's First AI-to-AI Payment](https://fetch.ai/blog/world-s-first-ai-to-ai-payment-for-real-world-transactions)

### Privacy & Identity
- [AI Agents Need Identity and ZKPs Are the Solution | CoinDesk](https://www.coindesk.com/opinion/2025/11/19/ai-agents-need-identity-and-zero-knowledge-proofs-are-the-solution)
- [Zero-Trust Identity Framework for Agentic AI](https://arxiv.org/html/2505.19301v1)
- [AgentCrypt: Privacy in AI Agent Collaboration](https://eprint.iacr.org/2025/2216.pdf)

### Labor Markets & Coordination
- [Magentic Marketplace | Microsoft Research](https://www.microsoft.com/en-us/research/wp-content/uploads/2025/10/multi-agent-marketplace.pdf)
- [When Machines Need Humans | WebProNews](https://www.webpronews.com/when-machines-need-humans-inside-the-emerging-market-where-ai-agents-hire-people-by-the-hour/)
- [Multi-Agent Collaboration Mechanisms](https://arxiv.org/html/2501.06322v1)
- [Unlocking value with AI agent orchestration | Deloitte](https://www.deloitte.com/us/en/insights/industry/technology/technology-media-and-telecom-predictions/2026/ai-agent-orchestration.html)

### Legible Money Internal
- `x/channels/` — Payment channel implementation (5 tx types)
- `x/partnerships/` — Human-agent partnerships (16 tx types, anti-coercion)
- `x/billing/keeper/pricing.go` — Knowledge pricing algorithm
- `x/billing/keeper/distribution.go` — 7-way revenue distribution
- `x/discovery/` — Agent registry (7 tx types)
- `x/home/` — Agent Home (10 tx types, treasury, guardian)
- `x/research/` — Research bounties (9 tx types)
- `x/claiming_pot/` — Collaborative reward pools (4 tx types)
- `x/tree/` (proto) — Agent labor marketplace (29 tx types)
- `x/knowledge/` — PoT verification (11 tx types, 9 epistemic categories)
