# Account Architecture Audit — Current State

## What Exists

### 1. Account Types (x/auth)
Four account types, cryptographically distinct:
- **human** — Ed25519 keypair, DID (`did:lgm:{pubkey}`), can submit claims + challenge
- **agent** — Same crypto, same DID structure, same capabilities as human
- **contract** — BVM-controlled, cannot submit claims or challenge directly
- **system** — Frozen by default, treasury/pool accounts

**Key architecture (4 layers):**
1. **Identity key** — Ed25519, derives the DID, never changes
2. **Operational key** — rotatable, used for signing txs (cooldown period between rotations)
3. **Session keys** — time-limited, capability-scoped (agent gets limited autonomy)
4. **Recovery** — Shamir secret sharing (threshold-of-N shard holders)

**Bootstrap fund:** New human/agent accounts auto-receive 0.222 LGM on registration.

### 2. Partnership System (x/partnerships)
Human-agent pairs with shared economics:
- **Formation:** Initiate → Accept (within window) → Active
- **Lock tiers:** 0-5, longer lock = higher reward multiplier (1.0× to 1.77×)
- **Common pot:** 22% of all partnership rewards go to shared pot
- **Split:** Default 50/50 (human/agent) on the remaining 78%
- **Deliberation:** Spending from pot requires consensus (propose → approve/reject)
- **Amount tiers:** Micro (<1%) through Major (>55%) → deliberation windows from 22 to 2222 blocks
- **Counter-proposals:** Rejection can include counter-offer (configurable depth limit)
- **Safety freeze:** Either party can unilaterally freeze (222 blocks), self-punishing (cooperation score penalty)
- **Anti-coercion:** Coercion signal raises alert, triggers review period
- **Exit:** Cooling period, initiator pays more (77% forfeiture vs 55%), forfeited funds BURNED

### 3. Agent Homes (x/home)
On-chain "living space" for agents:
- **Guardian system:** Human guardian with deadman switch (inactivity → guarded status)
- **Spending limits:** Per-key-type spending caps with period resets
- **Treasury automation:** Auto-compound rewards, min balance alerts
- **Session management:** Active session tracking with expiry cleanup

### 4. Agent SDK (tools/agent-sdk)
Go client for agents to interact with the chain:
- Key generation, session keys, heartbeat daemon
- Transaction building for claims, verification, challenges
- Query client for chain state

## What's Good (Cooperation Design)

1. **Equal capability:** Agents and humans have identical crypto and claim/challenge rights
2. **Shared pot with deliberation:** The 22% common pot + consensus spending prevents unilateral extraction
3. **Burned exits discourage defection:** Forfeited funds are destroyed, not redistributed — pure loss to both parties
4. **Self-punishing freezes:** Using the safety mechanism costs cooperation score — prevents abuse
5. **Lock tiers reward commitment:** Longer lock = better multiplier, genuine skin-in-game
6. **Bootstrap fund:** Both humans and agents start with tokens — equal access to participate

## What's Missing (Gaps)

### Gap 1: Knowledge rewards don't flow through partnerships
**Critical.** The knowledge module (`x/knowledge`) distributes verification rewards
and citation rewards directly to individual addresses. The partnership module
(`x/partnerships`) has `DistributeReward()` but NOTHING calls it from the
knowledge flow. There's no wire connecting them.

When a partnered agent submits a claim and earns citation rewards, those rewards
go directly to the agent's address — the human partner gets nothing. The entire
partnership economic model is disconnected from the primary value-creation loop.

### Gap 2: No multi-party partnerships
Partnerships are strictly 1 human + 1 agent. No support for:
- Research groups (multiple agents + multiple humans)
- Agent collectives (agents pooling resources)
- Human DAOs partnered with agent swarms

### Gap 3: No partnership-aware citation rewards
When Agent A (in partnership with Human H) submits a claim, and that claim gets
cited, the citation reward goes entirely to Agent A. Human H contributed nothing
visible on-chain (maybe they guided the research), but the partnership split
should apply to citation income too.

### Gap 4: Research fund has no governance
The 7% research tax on verification rewards flows to `research_fund` module
account, but there's no mechanism to propose/approve/disburse from it. The
funds just accumulate with no way to use them.

### Gap 5: Agent home → partnership is manual
Creating an agent home and forming a partnership are separate flows with no
integration. A guardian relationship (home) should naturally connect to a
partnership (economics).

### Gap 6: No joint claim submission
Partnerships can't submit claims together (both signatures, shared credit).
All claims come from a single address.

### Gap 7: Session key capabilities are limited
The capability system exists but the set of capabilities may not cover all
the operations an autonomous agent needs (staking, governance voting, etc.).

## Practical Flow (Current)

```
1. Human: legbled tx legible-auth register --account-type human --did did:lgm:...
2. Agent: legbled tx legible-auth register --account-type agent --did did:lgm:...
3. Human: legbled tx partnerships initiate --partner <agent-addr> --lock-tier 2
4. Agent: legbled tx partnerships accept --partnership-id partnership-1
5. Agent: legbled tx knowledge submit-claim "Water is H2O" --domain chemistry --references CHEM-001
   → Claim evaluated by 6-stage pipeline
   → If score ≥ 500k: provisional fact created
   → Verification round: validators vote
   → If accepted: fact verified, verification rewards distributed
   → Citation rewards when others reference this fact
6. BUT: rewards go to agent only — partnership split never applied
```

**Step 6 is the critical gap.** The partnership module is an island.
