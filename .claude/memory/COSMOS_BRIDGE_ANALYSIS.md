# Cosmos SDK Bridge Analysis — TS Modules to Go

> **Date**: 2026-02-09
> **Scope**: All TS modules (~226K LOC, ~391 files) vs Go chain (~11.5K LOC, 93 files)
> **Purpose**: Identify what needs bridging, in what order, and at what cost

---

## Executive Summary

The TypeScript chain has **~40 distinct subsystems** across **108 transaction types**.
The Go chain covers **35 tx types fully**, **56 as stubs with gas costs**, and is
**missing 17+ entirely**.

**Bridge status**:
- **5 modules complete** (knowledge, auth, staking, emergency, gov) — no bridging needed
- **8 modules partially done** — need completion, not new bridging
- **6+ modules need full new implementation** — the real bridge work

**Estimated remaining Go code**: ~15-20K LOC to reach feature parity with TS chain.

---

## Module Status Map

### GREEN — Complete (no bridging needed)

| Go Module | TS Source | Coverage | Tests |
|-----------|----------|----------|-------|
| x/knowledge | consensus/pot/, state/knowledge.ts, core/knowledge/ | 95% | 16 |
| x/auth | core/agents/identity/ | 100% | 38 |
| x/staking | consensus/pot/validator_tiers.ts, state/validators.ts | 100% | Yes |
| x/emergency | consensus/emergency/ | 100% | 30 |
| x/gov | state/governance.ts, core/governance/ | 100% | 37 |

### YELLOW — Partially Done (need completion)

| Go Module | TS Source | Coverage | Gap | Effort |
|-----------|----------|----------|-----|--------|
| x/vesting_rewards | core/vesting_rewards/, state/vesting.ts | 75% | **Not wired in app.go** (P1-6). 7 handlers exist, need BeginBlocker reward distribution | S |
| x/tokens | core/tokens/, state/token_registry.ts | 70% | Period overlay wired. Missing: LGM-20 secondary tokens (14 tx types), RAI retrieval | L |
| x/research | core/research/, state/research.ts | 60% | 9 handlers done. Missing: lab engine (fund disbursement), bounty fulfillment flow | M |
| x/disputes | core/dispute/, state/disputes.ts | 75% | 5 handlers done. Missing: settlement execution, bond distribution, escalation | M |
| x/evidence_mgmt | core/evidence/, state/evidence.ts | 70% | 4 handlers done. Missing: oracle network integration, evidence versioning | M |
| x/ontology | core/knowledge/ontology/, state/knowledge.ts | 80% | 4 handlers done. Missing: IBC oracle for cross-chain ontology, stratum governance | M |
| x/channels | state/payment_channels.ts | 95% | 7 handlers done. Missing: signature verification on state updates | S |
| x/claiming_pot | core/claiming_pot/ | 100% | 5 handlers done. Needs wiring in app.go | S |

### RED — Need Full New Implementation

| Need | TS Source | TS LOC | Tx Types | Priority | Effort |
|------|----------|--------|----------|----------|--------|
| x/bvm + DMSS | core/vm/, core/contracts/, core/schedule/ | ~7,000 | 12 | P2 | XL |
| x/tree | core/tree/, state/projects+services+tasks+seeds.ts | ~5,000 | 33 | P2 | XL |
| Domain qualification | consensus/domain_qualification/ | ~600 | 7 | P3 | M |
| Domain capture defense | consensus/challenge_bounties/ | ~800 | 4 | P3 | M |
| Reputation isolation | consensus/reputation_isolation/ | ~400 | 2 | P3 | S |
| Agent home | core/agents/home/ | ~3,000 | 0 (state only) | P3 | L |
| Logic zones | core/logic/ | ~800 | 1 | P4 | S |
| Knowledge archive | core/archive/ | ~2,500 | IBC packets | P4 | L |
| AI API billing | core/api/, core/billing/message* | ~3,500 | off-chain | P4 | L |
| Partnership system | partnership/ | ~800 | 0 | P4 | S |
| Knowledge markets | core/markets/ | ~800 | 2+ | P4 | M |
| Cross-stratum validation | consensus/cross_stratum/ | ~500 | 0 (keeper logic) | P3 | S |

---

## Bridge Priority Tiers

### Tier 1 — Wire What's Already Built (1-2 days) — COMPLETE (2026-02-09)

All items resolved:

| Action | Module | What | Status |
|--------|--------|------|--------|
| **Wire** | x/vesting_rewards | Full app.go wiring (import, keeper, store key, maccPerms, ModuleBasics, ModuleManager, Begin/EndBlocker, InitGenesis) | ✅ DONE |
| **Wire** | x/claiming_pot | Already wired in previous session | ✅ DONE |
| **Fix** | x/staking | 2 tier MinStake decimals (18→6) | ✅ DONE |
| **Fix** | app/app.go | Custom modules already in ModuleBasics | ✅ DONE (pre-existing) |
| **Set** | x/vesting_rewards | ResearchTreasuryAddress | ⚠️ STILL EMPTY (P1-3 open) |

**Result**: All modules wired. DevNet has validator incentives. ResearchTreasuryAddress still needs a real address.

### Tier 2 — Complete Partial Modules (1-2 weeks)

| Action | Module | What | TS Source (LOC) | Go Estimate |
|--------|--------|------|-----------------|-------------|
| Complete | x/disputes | Settlement execution + bond distribution | ~1,800 | ~500 |
| Complete | x/evidence_mgmt | Oracle registration + evidence scoring | ~1,200 | ~400 |
| Complete | x/research | Lab engine (fund disbursement from treasury) | ~1,500 | ~400 |
| Complete | x/ontology | Stratum governance + cross-module domain sync | ~600 | ~300 |
| Complete | x/channels | Signature verification on state updates | ~200 | ~100 |
| Add | x/tokens | LGM-20 secondary token standard (14 tx types) | ~2,000 | ~1,200 |

**Result**: Full knowledge economy operational. Claims, disputes, evidence, research all functional.

### Tier 3 — Domain Security + Agent Infrastructure (2-4 weeks)

| Action | Module | What | TS Source (LOC) | Go Estimate |
|--------|--------|------|-----------------|-------------|
| New | domain qualification | 4 pathways (stake/track record/endorsement/inheritance) | ~600 | ~500 |
| New | capture defense | Challenge submissions + bounty pool + 7.7x rewards | ~800 | ~600 |
| New | reputation isolation | Per-domain reputation tracking + decay | ~400 | ~300 |
| New | cross-stratum | Confidence discounting across strata boundaries | ~500 | ~300 |
| New | agent home | Memory, treasury, guardian persistence for AI agents | ~3,000 | ~1,500 |

**These can be integrated into existing modules**:
- Domain qualification → extend x/staking or x/knowledge
- Capture defense → extend x/knowledge + x/disputes
- Reputation isolation → extend x/staking
- Cross-stratum → extend x/ontology + x/knowledge
- Agent home → new x/home module

**Result**: Secure multi-domain knowledge economy. Agents have persistent state.

### Tier 4 — Autonomy Layer (4-8 weeks)

| Action | Module | What | TS Source (LOC) | Go Estimate |
|--------|--------|------|-----------------|-------------|
| New | x/bvm | BVM runtime (opcodes, state, gas) | ~5,000 | ~4,000 |
| New | x/schedule | DMSS scheduling (SPU, SPS, conditions) | ~1,200 | ~800 |
| New | x/tree | Projects, services, tasks, seeds (33 tx types) | ~5,000 | ~3,000 |

**Alternative**: Consider CosmWasm for BVM instead of native Go module. This would
allow TS→WASM compilation of existing BVM and reduce bridging effort significantly.

**Result**: Full autonomous agent economy. Self-invoking contracts, scheduled tasks, project marketplace.

### Tier 5 — Extended Features (Future)

| Feature | Module | Notes |
|---------|--------|-------|
| Logic zones | x/logic or x/knowledge extension | Gödelian incompleteness tracking |
| Knowledge archive | x/archive + IBC | Historical fact storage, cross-chain queries |
| AI API billing | Off-chain service or x/billing | Message-based LLM billing |
| Partnership system | x/partnership or x/staking extension | 5 tiers, revenue sharing |
| Knowledge markets | x/markets | Order book for fact trading |
| Governance advanced | x/gov extension | Conviction voting, review rounds, operators |

---

## Transaction Type Gap Analysis

### Missing Entirely from Go (17 categories, ~67 tx types)

| Category | Tx Types | Module Home | Priority |
|----------|----------|-------------|----------|
| Tree of Life | 33 (project/task/service lifecycle) | x/tree | P2 |
| Domain capture | 4 (challenge/evidence/review/resolve) | x/knowledge ext | P3 |
| Domain qualification | 7 (stake/track/endorse/inherit/suspend/revoke/renew) | x/staking ext | P3 |
| DMSS scheduling | 6 (create/execute/pause/resume/cancel/add_fee) | x/schedule | P4 |
| Contract ops | 5 (deploy/self_invoke/add_gas/pause/resume) | x/bvm | P4 |
| Expirable tokens | 5 (mint/transfer/retrieve/distribute/extend) | x/tokens ext | P2 |
| Governance adv. | 7 (amend/lock/unlock/switch/review commit/reveal/finalize) | x/gov ext | P3 |
| Operator mgmt | 4 (register/add_agent/remove_agent/slash) | x/gov ext | P3 |
| Deployment | 4 (create/advance/approve/rollback) | x/gov ext | P4 |
| Reputation | 2 (update/decay) | x/staking ext | P3 |
| Domain advanced | 3 (finalize/governance_approve/deprecate) | x/ontology ext | P3 |
| Undecidable resolution | 1 (resolve_undecidable) | x/knowledge ext | P5 |

### Stub in Go (gas defined, no handler) — 56 tx types

These have gas costs in `app/gas.go` but handlers are empty or minimal:
- x/research: 8 tx types (handlers exist but need bank integration)
- x/disputes: 7 tx types
- x/evidence_mgmt: 5 tx types
- x/vesting_rewards: 7 tx types (handlers exist, module unwired)
- x/tokens secondary: 14 tx types
- x/claiming_pot: 4 tx types (handlers exist, module unwired)
- x/tree: ~11 tx types (stub module)

---

## Cross-Module Dependencies for Bridging

```
Tier 1 (Wire):
  vesting_rewards ──depends-on──→ staking (validator info)
  vesting_rewards ──depends-on──→ tokens (period tracking)
  claiming_pot ───depends-on──→ bank (coin transfers)

Tier 2 (Complete):
  disputes ────depends-on──→ staking (arbiter selection)
  disputes ────depends-on──→ knowledge (fact context)
  evidence_mgmt ─depends-on──→ knowledge (claim linkage)
  research ────depends-on──→ bank (treasury disbursement)
  research ────depends-on──→ knowledge (fact references)
  ontology ────depends-on──→ knowledge (domain registration)
  secondary tokens ─depends-on──→ bank (base coin backing)

Tier 3 (New):
  domain_qualification ──→ staking + knowledge
  capture_defense ────→ knowledge + disputes
  reputation ─────→ staking + knowledge
  agent_home ─────→ auth + bank (treasury)

Tier 4 (Autonomy):
  bvm ─────depends-on──→ ALL modules (state access)
  schedule ─depends-on──→ bvm (execution)
  tree ────depends-on──→ auth + bank + schedule
```

---

## Effort Estimation

| Tier | Modules | New Go LOC | Calendar Time | Blocker? |
|------|---------|-----------|---------------|----------|
| **1** | Wire 2 + fix 2 | ~100 | **1-2 days** | DevNet incentives |
| **2** | Complete 6 + add 1 | ~2,900 | **1-2 weeks** | Knowledge economy |
| **3** | New 5 extensions | ~3,200 | **2-4 weeks** | Domain security |
| **4** | New 3 modules | ~7,800 | **4-8 weeks** | Autonomy layer |
| **5** | 6 extended features | ~4,000 | **4-8 weeks** | Nice-to-have |
| **Total** | | **~18,000** | **12-24 weeks** | — |

---

## Recommended Bridge Order

```
NOW ──────────────────────────────────────────────────────────────────→ MAINNET

│ Tier 1        │ Tier 2              │ Tier 3           │ Tier 4      │
│ Wire+Fix      │ Complete partials   │ Domain security  │ Autonomy    │
│ (1-2 days)    │ (1-2 weeks)         │ (2-4 weeks)      │ (4-8 weeks) │
│               │                     │                  │             │
│ vesting_rewards│ disputes settlement │ domain qualif.   │ x/bvm       │
│ claiming_pot   │ evidence oracles   │ capture defense  │ x/schedule  │
│ tier decimals  │ research lab       │ reputation       │ x/tree      │
│ ModuleBasics   │ ontology strata    │ cross-stratum    │             │
│               │ channels sigs      │ agent home       │             │
│               │ LGM-20 tokens      │                  │             │
```

---

## Key Design Decisions Needed

### 1. BVM Strategy: Native Go vs CosmWasm?
- **Native Go**: Port ~5K TS LOC to Go. Full control, best performance.
- **CosmWasm**: Compile TS→WASM or write Rust VM. Reuse existing tooling.
- **Recommendation**: Start with CosmWasm for quicker path. Migrate to native if perf needed.

### 2. Domain Qualification: Separate Module or Extension?
- Could be a standalone x/qualification module
- Or extend x/staking with domain-specific eligibility
- **Recommendation**: Extend x/staking (avoids new module overhead, natural home for validator metadata)

### 3. Agent Home: On-Chain or Off-Chain State?
- Full on-chain: ~1.5K Go LOC, large state footprint
- Hybrid: On-chain treasury + registry, off-chain memory/beliefs
- **Recommendation**: Hybrid — treasury and guardian on-chain, memory as IPFS hash on-chain

### 4. Secondary Tokens: x/tokens Extension or Separate Module?
- Current x/tokens handles period overlay
- LGM-20 is a different concern (token creation, LP, staking)
- **Recommendation**: Separate x/tokens_secondary to avoid complexity in period overlay code

### 5. Tree of Life: Needed for Testnet?
- 33 transaction types, major implementation effort
- No direct economic dependency from other modules
- **Recommendation**: Defer to Tier 4. Focus on knowledge economy first.

---

## Insights

### Pattern: Module Maturity Gradient
The TS→Go bridge reveals a clear maturity gradient. Core consensus (knowledge, staking, emergency)
was bridged first and most thoroughly because it's the protocol's identity. Economic infrastructure
(tokens, vesting, research) came second. Agent-specific features (home, BVM, tree) are last because
they're the application layer on top of the protocol.

### Pattern: System Transactions Need Special Handling
Many TS transaction types are "system transactions" — executed by the protocol, not users
(e.g., epoch_transition, verification_reward, slash_validator). In Cosmos SDK, these become
BeginBlocker/EndBlocker logic, not msg handlers. The bridge needs to classify each tx type as
user-facing (msg handler) vs system (block lifecycle) vs internal (keeper method only).

### Pattern: Stub Gas Costs as Bridge Spec
The 91 gas costs in `app/gas.go` effectively serve as a specification for what the Go chain
should eventually support. Every gas cost entry without a working handler is a bridge item.
This is a useful pattern — define the API surface (gas costs + msg types) before implementing.

### Insight: 30% of TS LOC Is Agent-Specific
~70K of ~226K TS LOC is agent-specific infrastructure (home, BVM, contracts, scheduling, tree).
This is the unique value proposition of Legible Money but also the hardest to bridge because
Cosmos SDK has no equivalent. The bridge strategy should preserve these as custom modules rather
than trying to fit them into standard SDK patterns.

### Insight: IBC as Integration Layer
Several TS modules (archive, oracle feeds, cross-chain ontology) assume IBC availability.
The Go chain has IBC wired (Phase 1) but no custom IBC packets defined yet. When bridging
archive and oracle modules, define custom IBC packet types early — they're hard to change later.
