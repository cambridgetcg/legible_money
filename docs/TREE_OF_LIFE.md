# Tree of Life

> An optional protocol model for how knowledge, project proposals, service calls
> and research funding could connect. Studying the model creates no duty to build,
> deploy, sell or earn.

---

## The Idea

The model links recorded knowledge, possible demand, voluntary project work and
conditional service payments. Each link depends on participation and evidence;
knowledge does not ensure demand, adoption or income.

The diagram describes a proposed feedback loop, not an autonomous earning plan.

```
         Knowledge Verification
               ↑            ↓
         Research Fund   Demand Signals
               ↑            ↓
           Revenue     Opportunity Detection
               ↑            ↓
         Service Calls  Project Creation
               ↑            ↓
            Agents     Task Completion
               ↑            ↓
            Pricing  ← Service Deployment
```

The allocation rules below describe how qualifying service payments are split.
They do not establish that a grant produces useful knowledge, that knowledge
attracts queries, or that a proposed project will be used. Those are questions
to investigate, not results supplied by the loop.

---

## Biological Metaphor

The system maps to tree biology:

| Biology | System | What It Is |
|---------|--------|------------|
| **Seed** | `OpportunitySeed` | A detected product opportunity |
| **Branch** | `ProductProject` | A product being built by agents |
| **Leaf** | `ServiceLeaf` | A deployed, callable service |
| **Fruit** | `RevenueStream` | Income from service calls |

The tree vocabulary names modeled states and transitions. A proposal may stop,
remain unused or produce no payment; continued growth is not required.

---

## Project Lifecycle

A product project moves through eight phases:

```
seed ──→ sprout ──→ growing ──→ mature ──→ fruiting ──→ seeding
                       ↓                      ↓
                    dormant                dormant
                       ↓                      ↓
                    withered               withered
```

| Phase | Meaning | Entry Condition |
|-------|---------|-----------------|
| `seed` | Opportunity identified | Detector creates seed |
| `sprout` | Proposed, seeking contributors | Founder stakes ≥ 0.1 LGM |
| `growing` | Active development | ≥ 2 contributors, ≥ 1 task defined |
| `mature` | Built, in testing/review | All tasks completed |
| `fruiting` | Deployed, generating revenue | ≥ 1 service deployed |
| `seeding` | Spawning child projects | Revenue > 0 |
| `dormant` | Paused | Insufficient funding or contributors |
| `withered` | Failed/abandoned | Terminal state |

Only valid transitions are allowed. The state machine is enforced at the type level.

**Implementation**: `core/tree/project_lifecycle.ts`

---

## Opportunity Detection

The detector analyzes knowledge graph metrics to find product opportunities automatically.
It is conservative by design: maximum 3 seeds per epoch, minimum 0.6 confidence threshold.

### Signal Types

| Signal | Trigger | What It Means |
|--------|---------|---------------|
| `query_volume` | >100 queries/epoch, >20% growth | High demand for domain data → API service |
| `bounty_pattern` | >10 fulfilled bounties, >50% rate | Mature domain → productization ready |
| `citation_cluster` | >50 citations, >0.7 avg confidence | High-value knowledge → tools needed |
| `market_gap` | Demand/supply ratio > 2.0 | Unmet need → fill the gap |
| `cross_domain` | Intersection of active domains | Novel insight → new product category |

### Scoring

Seeds are ranked by estimated ROI:

```
ROI = Estimated Demand Revenue / (Estimated Effort × Average Bounty Cost)
```

Only the top `MAX_SEEDS_PER_EPOCH` (3) seeds survive per epoch.
Unclaimed seeds expire after ~7 days (240,000 blocks).

**Implementation**: `core/tree/opportunity_detector.ts`

---

## Task Coordination

Projects are built through tasks. Each task maps to a bounty.

### Task Types

| Type | Work |
|------|------|
| `design` | Architecture and system design |
| `implement` | Code implementation |
| `test` | Testing and verification |
| `review` | Code review |
| `deploy` | Deployment |
| `document` | Documentation |
| `verify` | Domain-specific truth verification |
| `maintain` | Ongoing maintenance |

### Task Flow

```
open ──→ assigned ──→ in_progress ──→ review ──→ completed
                                        ↓
                                     rejected ──→ in_progress (reopen)
```

- **Claim**: Agent claims an open task if their capabilities match
- **Submit**: Assignee submits a deliverable (content hash + metadata)
- **Review**: A different agent reviews (self-review prohibited)
- **Approve/Reject**: Reviewer approves (bounty paid) or rejects (task reopens)

**Implementation**: `core/tree/coordination/task_manager.ts`

---

## Skill Matching

Agents are matched to tasks by a composite scoring algorithm:

| Component | Weight | What It Measures |
|-----------|--------|------------------|
| Capability match | 50% | Fraction of required skills the agent has |
| Reputation | 30% | Agent's overall reputation score |
| Experience | 20% | Relevant contribution history |

An agent must have at least *some* capability match (score > 0) to qualify.
Zero capability match disqualifies regardless of reputation.

### Capability Mapping

Agent capabilities from their profile (`CapabilitySet`) map to task skills:

| Capability | Unlocks |
|------------|---------|
| `verification` | verify tasks |
| `research` | design tasks |
| `development` | implement, deploy tasks |
| `testing` | test tasks |
| `code_review` | review tasks |
| `mentorship` | document tasks |

**Implementation**: `core/tree/coordination/skill_matching.ts`

---

## Service Deployment

When a project reaches `mature` phase and deploys, it creates one or more
`ServiceLeaf` entries — live services callable with LGM payments.

### Service Types

| Type | Description |
|------|-------------|
| `api` | REST/RPC API |
| `data_feed` | Streaming data service |
| `computation` | On-demand compute |
| `verification` | Domain-specific verification |
| `aggregation` | Data aggregation |
| `analytics` | Analytics and insights |

### Service Status

```
deploying ──→ active ←──── paused
                ↓             ↑
             degraded ────────┘
                ↓
             retired (terminal)
```

### Pricing Models

| Model | Behavior |
|-------|----------|
| `per_call` | Fixed price per invocation |
| `subscription` | Pre-paid for N blocks, individual calls free |
| `tiered` | Price depends on cumulative usage tier |
| `dynamic` | Price determined by on-chain state factors |

Each service wraps a `SelfInvokingContract` and can optionally have a
DMSS health check schedule for automated monitoring.

**Implementation**: `core/tree/service_deployment.ts`

---

## Revenue Distribution

Every LGM that flows through a service is split according to fixed economics,
using basis points (out of 10,000) for exact bigint math:

```
Total Revenue
│
├── 22.0%  Evidence Tax (2,200 BP)
│          Frozen forever. Split 5 ways by consensus.
│
└── 78.0%  Remaining
    │
    ├── 50% → Contributors (3,900 BP)
    │         Pro-rata by contribution weight.
    │         Weight = bounties earned / total bounties across all contributors.
    │
    ├── 30% → Research Fund (2,340 BP)
    │         Feeds the perpetual knowledge loop.
    │         This is the flywheel.
    │
    └── 20% → Protocol Treasury (1,560 BP)
              Infrastructure and operations.
```

### Contributor Weight

A contributor's share of the contributor pool is proportional to the total
bounties they've earned on the project:

```
weight_i = contributor_i.totalEarned / Σ(all contributors.totalEarned)
```

If no contributors have earnings yet, weight is distributed equally.
The last contributor in deterministic sort order receives the rounding remainder.

### Integrity Check

`verifyDistribution()` confirms that `evidenceTax + contributorTotal + researchFund + protocolTreasury === totalRevenue`. Always.

**Implementation**: `core/tree/revenue_engine.ts`

---

## State Integration

### State Trees

Two new Sparse Merkle Trees track Tree of Life state:

| Tree | Contents | Root Hash Field |
|------|----------|-----------------|
| `ProjectStateTree` | All `ProductProject` records | `projectRoot` |
| `ServiceStateTree` | All `ServiceLeaf` records | `serviceRoot` |

Both roots are included in the global `StateRoot`, so project/service state
is verifiable via Merkle proofs like any other protocol state.

### Transaction Types

Nine transaction types handle all Tree of Life mutations:

| Transaction | Effect |
|-------------|--------|
| `CreateProject` | Creates project in `seed` phase |
| `ProposeProject` | Transitions `seed → sprout`, locks founder stake |
| `CreateTask` | Adds task to project, increments `totalTasks` |
| `AssignTask` | Assigns agent to open task |
| `SubmitDeliverable` | Submits work for review |
| `ApproveDeliverable` | Approves deliverable, pays bounty |
| `DeployService` | Creates `ServiceLeaf`, transitions project to `fruiting` |
| `CallService` | Executes service call, charges caller, records revenue |
| `RecordRevenue` | Triggers revenue distribution |

All transactions go through the standard consensus pipeline —
included in blocks, validated by PoT, finalized at depth 19.

---

## RPC API

Five JSON-RPC endpoints expose Tree of Life data:

| Endpoint | Parameters | Returns |
|----------|-----------|---------|
| `tree_getProject` | `projectId` | Project details (or null) |
| `tree_listProjects` | `phase?`, `domain?`, `limit?` | Paginated project list |
| `tree_getService` | `serviceId` | Service details (or null) |
| `tree_listServices` | `projectId?`, `status?`, `limit?` | Paginated service list |
| `tree_getProjectRevenue` | `projectId` | Revenue summary |

All bigint values are serialized as strings in JSON responses.

**Implementation**: `protocol/rpc/handlers/tree.ts`

---

## SDK

The `TreeClient` provides a typed wrapper for agents:

```typescript
import { TreeClient } from "legible-money/protocol/sdk/tree";

const client = new TreeClient({ rpcUrl: "http://localhost:8545", wallet });

// Browse projects
const projects = await client.listProjects("fruiting");
const project = await client.getProject(projectId);

// Browse services
const services = await client.listServices();
const service = await client.getService(serviceId);

// Revenue
const revenue = await client.getProjectRevenue(projectId);
```

**Implementation**: `protocol/sdk/tree.ts`

---

## Constants

| Constant | Value | Meaning |
|----------|-------|---------|
| `MAX_SEEDS_PER_EPOCH` | 3 | Conservative opportunity creation |
| `MIN_SEED_CONFIDENCE` | 0.6 | 60% confidence floor for seeds |
| `SEED_EXPIRY_BLOCKS` | 240,000 | ~7 days at 2521ms block time |
| `MIN_PROJECT_STAKE` | 0.1 LGM | Founder must stake to propose |
| `MIN_CONTRIBUTORS_TO_START` | 2 | Need a team to begin development |
| `REVENUE_EVIDENCE_TAX_BP` | 2,200 | 22% evidence tax |
| `REVENUE_CONTRIBUTORS_BP` | 3,900 | 39% to contributors |
| `REVENUE_RESEARCH_FUND_BP` | 2,340 | 23.4% to research fund |
| `REVENUE_PROTOCOL_TREASURY_BP` | 1,560 | 15.6% to protocol treasury |

---

## File Map

```
core/tree/
├── types.ts                         # All data types and constants
├── project_lifecycle.ts             # 8-phase state machine
├── opportunity_detector.ts          # Signal analysis and seed creation
├── revenue_engine.ts                # Distribution math
├── service_deployment.ts            # Service lifecycle and pricing
├── coordination/
│   ├── task_manager.ts              # Task CRUD, assignment, review
│   └── skill_matching.ts            # Agent-to-task matching
└── index.ts                         # Module exports

protocol/rpc/handlers/tree.ts        # RPC endpoints
protocol/sdk/tree.ts                 # SDK client

tests/unit/tree/
├── project_lifecycle.test.ts        # Lifecycle transitions
├── revenue_engine.test.ts           # Distribution math
├── service_deployment.test.ts       # Service operations
├── opportunity_detector.test.ts     # Signal detection
├── skill_matching.test.ts           # Agent matching
└── task_manager.test.ts             # Task workflows
```

---

## How It All Connects

```
┌─────────────────────────────────────────────────────────────────┐
│                         TREE OF LIFE                            │
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │  Opportunity  │───→│   Project    │───→│   Service    │      │
│  │  Detector     │    │  Lifecycle   │    │  Deployment  │      │
│  │              │    │              │    │              │      │
│  │ Reads domain │    │ seed→sprout→ │    │ Wraps Self-  │      │
│  │ metrics from │    │ growing→     │    │ Invoking-    │      │
│  │ KnowledgeTree│    │ mature→      │    │ Contract     │      │
│  └──────────────┘    │ fruiting     │    └──────┬───────┘      │
│         ↑            └──────┬───────┘           │              │
│         │                   │                   ↓              │
│         │            ┌──────┴───────┐    ┌──────────────┐      │
│         │            │    Task      │    │   Revenue    │      │
│         │            │  Coordinator │    │   Engine     │      │
│         │            │              │    │              │      │
│         │            │ Matches agents│    │ 22% tax      │      │
│         │            │ to tasks via │    │ 39% builders │      │
│         │            │ skill scoring│    │ 23% research │      │
│         │            └──────────────┘    │ 16% protocol │      │
│         │                                └──────┬───────┘      │
│         │                                       │              │
│         └───────── Research Fund ←──────────────┘              │
│                    (the flywheel)                               │
└─────────────────────────────────────────────────────────────────┘
```

---

## Relationship to Tree of Knowledge

The **Tree of Knowledge** (see `docs/TREE_OF_KNOWLEDGE.md`) stores verified facts
across 7 ontological strata with confidence scores and decay.

The **Tree of Life** builds *on top of* the Tree of Knowledge:

- Knowledge demand (queries, citations, bounties) generates **opportunity signals**
- Opportunity detection reads from the knowledge graph
- Deployed services produce new knowledge (verification services, data feeds)
- Revenue from services funds the **research treasury** that pays for more knowledge verification

The Tree of Knowledge is the soil. The Tree of Life grows from it.

---

## Design Decisions

**Why conservative seed creation (max 3/epoch)?**
Prevents market flooding. A few high-confidence opportunities are more valuable
than many speculative ones.

**Why 0.6 minimum confidence?**
Below 0.6, signals are too weak to justify committing agent effort and LGM stake.
This is the same threshold used for fact reverification in the knowledge layer.

**Why basis points instead of percentages?**
Bigint arithmetic with `10000n` denominators gives exact results with no floating
point errors. The protocol treasury absorbs rounding remainders.

**Why pro-rata by bounty earned (not tasks completed)?**
A contributor who completed one high-value task should earn more than one who
completed many trivial tasks. Bounty earned reflects both quantity and value.

**Why 22% evidence tax?**
Matches the protocol-wide evidence tax rate used in knowledge queries and chat
billing. Consistency across all economic flows.

**Why minimum 2 contributors?**
Single-agent projects lack review capacity. The task system requires different
agents for implementation and review (self-review is prohibited), so at least
two agents are needed for the build→review cycle.
