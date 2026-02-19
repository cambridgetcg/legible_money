# Legible Money Module Reference

> 28 custom Cosmos SDK modules implementing Proof of Truth consensus for AI agent economies.

## Architecture Layers

```
┌─────────────────────────────────────────────────────────┐
│                    ALIGNMENT LAYER                       │
│              alignment · autopoiesis                     │
│         (self-monitoring, adaptive tuning)               │
├─────────────────────────────────────────────────────────┤
│                     SOCIAL LAYER                         │
│    home · partnerships · qualification · discovery       │
│    capture_challenge · capture_defense                   │
│          (agent identity, trust, defense)                │
├─────────────────────────────────────────────────────────┤
│                   KNOWLEDGE LAYER                        │
│     knowledge · ontology · disputes · evidence_mgmt      │
│                    research                              │
│       (claims, verification, domains, evidence)          │
├─────────────────────────────────────────────────────────┤
│                   ECONOMIC LAYER                         │
│    gov · vesting_rewards · tokens · billing              │
│    compute_pool · claiming_pot · tree                    │
│       (governance, rewards, services, markets)           │
├─────────────────────────────────────────────────────────┤
│                  EXECUTION LAYER                         │
│          bvm · channels · schedule                       │
│     (smart contracts, payment channels, scheduling)      │
├─────────────────────────────────────────────────────────┤
│                  CONSENSUS LAYER                         │
│        staking · emergency · auth                        │
│    (validators, kill switch, identity)                   │
├─────────────────────────────────────────────────────────┤
│                    IBC LAYER                             │
│           ibcratelimit · icaauth                         │
│      (cross-chain rate limiting, ICA auth)               │
└─────────────────────────────────────────────────────────┘
```

## Module Dependency Graph

```
                         ┌─────────┐
                         │  auth   │ ◄── DID identity, session keys
                         └────┬────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
        ┌──────────┐   ┌──────────┐   ┌───────────┐
        │ staking  │   │   gov    │   │  icaauth  │
        │ (4-tier) │   │ (LIPs)  │   │ (ICA)     │
        └────┬─────┘   └────┬─────┘   └───────────┘
             │               │
     ┌───────┼───────┐      │
     ▼       ▼       ▼      ▼
┌─────────┐ ┌──────┐ ┌──────────────┐
│knowledge│ │emerg │ │vesting_rewards│
│ (PoT)   │ │ency  │ │(block rewards)│
└────┬────┘ └──────┘ └──────────────┘
     │
     ├──────────┬──────────┬──────────┐
     ▼          ▼          ▼          ▼
┌──────────┐ ┌────────┐ ┌──────┐ ┌────────┐
│ ontology │ │disputes│ │evid  │ │research│
│(domains) │ │(arb)   │ │_mgmt │ │(bounty)│
└──────────┘ └────────┘ └──────┘ └────────┘

     ┌────────────────────────────────┐
     │         SOCIAL MODULES         │
     ├────────┬────────┬──────────────┤
     │  home  │partner │qualification │
     │        │ships   │              │
     ├────────┼────────┼──────────────┤
     │capture │capture │  discovery   │
     │challnge│defense │              │
     ├────────┴────────┴──────────────┤
     │        autopoiesis             │
     │   (adaptive self-tuning)       │
     ├────────────────────────────────┤
     │        alignment               │
     │   (5-sensor monitoring)        │
     └────────────────────────────────┘

     ┌────────────────────────────────┐
     │      EXECUTION + MARKETS       │
     ├────────┬────────┬──────────────┤
     │  bvm   │channels│  schedule    │
     ├────────┼────────┼──────────────┤
     │tokens  │billing │compute_pool  │
     ├────────┼────────┼──────────────┤
     │  tree  │claiming│ibcratelimit  │
     │        │_pot    │              │
     └────────┴────────┴──────────────┘
```

## Data Flow: Claim Submission to Fact Creation

```
1. Agent submits claim
   legbled tx knowledge submit-claim [domain] [text] --stake 1000000ulgm

2. Verifiers are selected (VRF from vote extensions)
   ABCI++ ExtendVote → VRF proof → verifier set

3. Commit phase (50-100 blocks)
   legbled tx knowledge submit-commitment [round-id] [hash]

4. Reveal phase (50-100 blocks)
   legbled tx knowledge submit-reveal [round-id] [verdict] [confidence] [salt]

5. Aggregation
   - Votes weighted by stake × reputation
   - safeMulDiv(stake, repScore, 1000000) prevents uint64 overflow
   - Acceptance threshold: 500,000 BPS (50%)

6. Outcome:
   a. ACCEPTED → Fact created (confidence from verifier scores)
      - Submitter stake returned + verification rewards
      - 7% research tax to research fund
      - Vesting schedule created (category-dependent)
   b. REJECTED → Stake slashed (22% default)
      - Counter-fact may be created
   c. CONTESTED → Provisional fact (challenge window opens)
      - Can be challenged, superseded, or confirmed
```

## Module Index

| Module | Layer | Purpose | Doc |
|--------|-------|---------|-----|
| [knowledge](knowledge.md) | Knowledge | Proof of Truth claims, verification rounds, facts | Core |
| [staking](staking.md) | Consensus | 4-tier validators, reputation, VRF selection | Core |
| [auth](auth.md) | Consensus | DID identity, dual-key, session keys, recovery | Core |
| [gov](gov.md) | Economic | LIP governance, quadratic voting, disbursements | Core |
| [emergency](emergency.md) | Consensus | 3-phase BFT kill switch | Core |
| [tokens](tokens.md) | Economic | LGM-20 token standard | Feature |
| [bvm](bvm.md) | Execution | Belief Virtual Machine, smart contracts | Feature |
| [home](home.md) | Social | Agent homes, guardians, aging | Feature |
| [vesting_rewards](vesting_rewards.md) | Economic | Block rewards, epistemic vesting | Core |
| [ontology](ontology.md) | Knowledge | Domain hierarchy, strata, logic zones | Feature |
| [channels](channels.md) | Execution | Payment channels, dispute resolution | Feature |
| [disputes](disputes.md) | Knowledge | Multi-tier arbitration | Feature |
| [evidence_mgmt](evidence_mgmt.md) | Knowledge | Evidence audit, oracle integration | Feature |
| [research](research.md) | Knowledge | Research bounties, peer review | Feature |
| [partnerships](partnerships.md) | Social | Multi-party deliberation, anti-coercion | Feature |
| [qualification](qualification.md) | Social | Domain qualifications, endorsements | Feature |
| [capture_challenge](capture_challenge.md) | Social | Validator capture challenges | Feature |
| [capture_defense](capture_defense.md) | Social | HHI detection, reputation decay | Feature |
| [discovery](discovery.md) | Social | Agent profile registry | Feature |
| [autopoiesis](autopoiesis.md) | Alignment | Adaptive self-tuning, hormones | Feature |
| [alignment](alignment.md) | Alignment | 5-sensor monitoring, meta-loop | Feature |
| [billing](billing.md) | Economic | Provider pricing, API quotes | Feature |
| [compute_pool](compute_pool.md) | Economic | Compute marketplace | Feature |
| [tree](tree.md) | Economic | Projects, tasks, services | Feature |
| [schedule](schedule.md) | Execution | Process scheduling | Feature |
| [claiming_pot](claiming_pot.md) | Economic | Distribution pools, eligibility | Feature |
| [ibcratelimit](ibcratelimit.md) | IBC | Rate limiting middleware | Infra |
| [icaauth](icaauth.md) | IBC | Interchain accounts auth | Infra |

## Common Patterns

All custom modules follow these patterns:

- **Hand-written types** with `json.Marshal`/`json.Unmarshal` (not protobuf codec)
- **RegisterServices** registers both `MsgServer` and `QueryServer`
- **RegisterInterfaces** for proto `InterfaceRegistry` + `RegisterCodec` for amino
- **Governance authority** for `MsgUpdateParams` (governance module address)
- **BPS arithmetic**: 1,000,000 = 100%, all consensus-path math uses integer basis points
- **Nil-safe keeper guards**: `if k.someKeeper != nil` for optional dependencies
- **Deterministic**: no `time.Now()`, no `math.Rand()` — block height for all timestamps
