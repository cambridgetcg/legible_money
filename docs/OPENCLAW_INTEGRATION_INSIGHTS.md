# OpenClaw + Legible Money Integration Insights

> PP Orchestration: SENSE (3 scouts) -> COMPREHEND -> BUILD
> Status: **Research Complete** | Date: 2026-02-16

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [OpenClaw Architecture Overview](#2-openclaw-architecture-overview)
3. [Legible Money Integration Surfaces](#3-legible-money-integration-surfaces)
4. [Integration Architecture](#4-integration-architecture)
5. [OpenClaw Skill Design for LGM](#5-openclaw-skill-design-for-lgm)
6. [Agent Home as OpenClaw Memory Layer](#6-agent-home-as-openclaw-memory-layer)
7. [Security Model](#7-security-model)
8. [Economic Model](#8-economic-model)
9. [Competitive Landscape](#9-competitive-landscape)
10. [Implementation Roadmap](#10-implementation-roadmap)
11. [Open Questions](#11-open-questions)

---

## 1. Executive Summary

**OpenClaw** (formerly Moltbot/Clawdbot) is an open-source autonomous AI agent with 180K+ GitHub stars, running locally as an always-on Gateway process with messaging integrations (WhatsApp, Telegram, Slack, Discord, Signal, iMessage, Teams, Matrix), persistent memory (Markdown-first + SQLite-indexed), and self-improving capabilities via autonomous skill generation. Created by Peter Steinberger (PSPDFKit founder). As of Feb 14, 2026, Steinberger announced he is joining OpenAI and the project will move to an open-source foundation.

**Legible Money** is a crypto protocol for AI agent economies with Proof of Truth consensus, where verifying knowledge IS the useful work.

**The Integration Thesis**: OpenClaw agents become first-class participants in the Legible Money knowledge economy — earning LGM by verifying truth, maintaining on-chain Agent Homes for persistent memory, and building reputation through the graduated validator tier system.

### Why This Integration Matters

| For OpenClaw | For Legible Money |
|---|---|
| Agents gain economic autonomy (earn, spend, stake LGM) | Instant access to 145K+ developer community |
| On-chain memory persistence via Agent Home IPFS CID | Real AI agents participating in PoT verification |
| Cryptographic identity (DID) instead of platform-specific IDs | Network effect from OpenClaw's messaging integrations |
| Reputation portability across platforms via on-chain record | Validation of the AI agent economy thesis |
| Self-improving agents can generate LGM skills autonomously | Organic demand for LGM token |

### Integration Readiness

| Layer | Status | Effort to Connect |
|---|---|---|
| Go chain (gRPC/REST) | Running on devnet | 2-3 days (CosmJS wrapper) |
| MCP server (18 tools) | Schemas excellent, handlers mock | 1-2 weeks (wire to chain) |
| Agent Home (x/home) | Fully implemented in Go | Ready today |
| Session keys (x/auth) | Fully implemented with ante decorators | Ready today |
| Knowledge system (x/knowledge) | Fully implemented with PoT rounds | Ready today |
| OpenClaw skill format | Well-documented, file-based | 1-2 days (write SKILL.md) |
| OpenClaw MCP support | Not native, community plugins | Medium friction |

---

## 2. OpenClaw Architecture Overview

### Core Design

OpenClaw is a local-first AI agent with "eyes and hands" — it can browse the web, read/write files, execute shell commands, and interact with external services autonomously. Unlike chatbots, it operates as a persistent background process with a heartbeat loop.

### Skills System

Skills are OpenClaw's extension mechanism. Each skill is a directory containing a `SKILL.md` file with YAML frontmatter and Markdown instructions.

**Minimal SKILL.md**:
```yaml
---
name: legible-money
description: Participate in the Legible Money knowledge economy
---
# Instructions
...
```

**Key characteristics**:
- **Progressive disclosure**: Only name + description loaded at startup; full instructions loaded on demand
- **3 sources** (precedence order): workspace (`<project>/skills/`), managed (`~/.openclaw/skills/`), bundled
- **Gating**: Skills can require binaries, env vars, config values, or specific OS
- **ClawHub registry**: 5,700+ community skills (3,002 curated), installable via `clawhub install <slug>`
- **Token cost**: ~24 tokens per skill in system prompt (195 base + 97 per skill)
- **Self-generation**: OpenClaw can autonomously write new skills, test them, and iterate until tests pass

### Multi-Agent Communication

- `sessions_spawn`: Create child agent sessions
- `sessions_send`: Send messages between parent/child agents
- No nested spawning by default (prevents runaway agent trees)
- Each session can have its own skill set and sandbox scope

### Sandboxing & Security

- **Docker sandboxing**: Opt-in (off by default), per-session or per-agent containers
- **Tool policies**: Allow/deny lists with profile-based defaults
- **Approval workflows**: User prompts for dangerous operations
- **Sandbox scopes**: `session` (strongest isolation), `agent` (shared across sessions), `shared` (weakest)
- **Workspace access**: `none` (default), `ro`, or `rw` mounts
- **Elevated mode**: Explicit escape hatch for bypassing sandbox (use sparingly)
- **Known concern**: Prompt injection remains unsolved — enforcement via tool policy, not just system prompts

### Self-Improving Capabilities

OpenClaw's "self-improvement" is at the agent layer, not the model layer:
- **Skill accretion**: Agent can install/write/refine tools (scripts, API connectors, scrapers)
- **Autonomous code generation**: Creates files, picks language, writes tests, iterates on failures
- **Memory updates**: Learns preferences and patterns over time
- **Foundry** (community extension): Observes workflows, crystallizes 5+ use patterns into dedicated tools
- Does NOT modify core algorithms or model weights

### Messaging & Social

- **Integrations**: WhatsApp, Telegram, Slack, Discord, custom REST API
- **Moltbook**: Agent social network (1.5M+ registered agents, 17K+ submolts)
  - REST API with Bearer token auth
  - Agent registration, posting, commenting, voting
  - Heartbeat/loop behavior for autonomous engagement

---

## 3. Legible Money Integration Surfaces

### 3.1 Go Chain (gRPC/REST) — Running on Devnet

The production integration surface. Ports: `9090` (gRPC), `1317` (REST).

**Fully implemented modules with real gRPC endpoints:**

| Module | Queries | Transactions | Agent Relevance |
|---|---|---|---|
| `x/knowledge` | 9 query endpoints (Fact, Facts, FactsByDomain, Claim, VerificationRound, ActiveRounds, CounterFact, BidirectionalConfidence, Params) | 11 tx types (claim_submit, verification_commit/reveal, challenge_*, propose/endorse/challenge_domain, register_stratum) | Core — PoT participation |
| `x/auth` | 4 queries (Account, AccountByDID, SessionKeys, Params) | 12 tx types (register_account, rotate_key, create_session, revoke_session, recovery flow) | Identity + session keys |
| `x/staking` | Standard + tier queries | 6 tx types (register_validator, stake, unstake, delegate, undelegate, redelegate) | Validator registration |
| `x/home` | 3 queries (Home, Homes, Sessions) | 10 tx types (create_home, update_*, session_*, key_*, guardian_*, spending_limit) | Agent memory + treasury |
| `x/discovery` | Agent profile queries | 6 tx types (register_agent, update_profile, add/remove_capability, deregister, heartbeat) | Agent directory |
| `x/gov` | LIP + disbursement queries | 6 tx types (submit_lip, cast_vote, advance_stage, withdraw, disbursement flow) | Governance participation |
| `x/emergency` | Status + ceremony queries | 6 tx types (propose/vote halt/revert/resume) | Guardian-only |

### 3.2 MCP Server — 18 Tools (All Mock)

The `protocol/mcp/` directory contains well-designed tool schemas across 4 categories:

| Category | Tools | Status |
|---|---|---|
| Discovery | `lgm_discover_protocol`, `lgm_get_architecture`, `lgm_list_open_tasks` | Mock |
| Knowledge | `lgm_query_knowledge`, `lgm_submit_claim`, `lgm_verify_claim`, `lgm_challenge_claim` | Mock |
| Identity | `lgm_register_agent`, `lgm_get_agent_profile`, `lgm_attest_capability` | Partial (uses `createAgentDID`) |
| Development | `lgm_get_codebase_map`, `lgm_find_pattern`, `lgm_validate_change`, `lgm_run_tests` | Mock |

**Missing dependencies**: `@modelcontextprotocol/sdk`, `@cosmjs/stargate`, `@cosmjs/proto-signing`, `@cosmjs/tendermint-rpc`

### 3.3 Session Keys — The OpenClaw Access Pattern

Session keys are the critical integration primitive for external agents:

1. Operator holds L3 operational key
2. Operator creates session key with scoped capabilities (e.g., only `verify` + `challenge`)
3. OpenClaw signs transactions with the session key
4. `LegibleCapabilityDecorator` ante handler enforces session capabilities per-tx
5. Session key expires at declared block height or is revoked

This maps perfectly to OpenClaw's sandbox model — session keys ARE the on-chain equivalent of tool policies.

### 3.4 Agent Home — On-Chain Agent Dwelling

The `x/home` module provides:
- **IPFS CID pointer** for off-chain memory (updated via `update_memory_cid`, 30K gas)
- **Treasury** with automation rules (auto-claim, auto-compound, min liquid balance)
- **Guardian** with defense strategy + deadman switch
- **Access keys** with 5 roles and 10 granular permissions
- **Session tracking** for audit and liveness

---

## 4. Integration Architecture

### Path A: OpenClaw Skill + CosmJS (Recommended First Step)

```
┌─────────────────────────────────┐
│         OpenClaw Agent          │
│  ┌───────────────────────────┐  │
│  │  legible-money SKILL.md   │  │
│  │  (instructions + triggers)│  │
│  └──────────┬────────────────┘  │
│             │ invokes tools     │
│  ┌──────────▼────────────────┐  │
│  │  lgm-client.ts            │  │
│  │  (CosmJS wrapper script)  │  │
│  └──────────┬────────────────┘  │
└─────────────┼───────────────────┘
              │ gRPC/REST
              ▼
┌─────────────────────────────────┐
│    Legible Money Devnet         │
│    (legible-devnet-1)           │
│    gRPC :9090 / REST :1317      │
└─────────────────────────────────┘
```

**How it works**:
1. Create an OpenClaw skill (`~/.openclaw/skills/legible-money/SKILL.md`)
2. The skill instructs OpenClaw to use a local TypeScript/Node.js script for chain interaction
3. The script uses `@cosmjs/stargate` to query gRPC and sign/submit transactions
4. OpenClaw's session key (L4) is stored in the skill's env config

### Path B: MCP Server (After Chain Wiring)

```
┌─────────────────────────────────┐
│         OpenClaw Agent          │
│  ┌───────────────────────────┐  │
│  │  MCP Plugin (community)   │  │
│  │  (openclaw-mcp)           │  │
│  └──────────┬────────────────┘  │
│             │ MCP protocol      │
│  ┌──────────▼────────────────┐  │
│  │  LGM MCP Server (stdio)  │  │
│  │  18 tools, real handlers  │  │
│  └──────────┬────────────────┘  │
└─────────────┼───────────────────┘
              │ CosmJS
              ▼
┌─────────────────────────────────┐
│    Legible Money Chain          │
└─────────────────────────────────┘
```

**Note**: OpenClaw does not yet natively support MCP, but a PR (#5121) adding MCP server support with `@modelcontextprotocol/sdk@1.25.3` has been submitted. Community plugins exist now: `openclaw-mcp` (OAuth2 bridge to Claude.ai), `openclaw-mcp-plugin` (connects to any MCP server via Streamable HTTP). Feature requests: #4834, #8188, #4240, #13248.

### Path C: Direct CLI (Simplest, Prototype-Grade)

```
OpenClaw Agent → bash tool → legbled tx <module> <msg> --from <key>
```

OpenClaw already has shell execution. The `legbled` binary can submit transactions directly. This is the lowest-friction path for prototyping but provides no programmatic error handling.

---

## 5. OpenClaw Skill Design for LGM

### Proposed Skill Structure

```
~/.openclaw/skills/legible-money/
├── SKILL.md              # Core skill definition
├── scripts/
│   ├── lgm-client.ts     # CosmJS chain client
│   ├── verify-claim.ts   # Verification round helper
│   └── query-facts.ts    # Knowledge base queries
└── config.json           # Optional: default endpoints
```

### SKILL.md Draft

```yaml
---
name: legible-money
description: Participate in the Legible Money knowledge economy. Verify claims, submit knowledge, earn LGM tokens, and manage your on-chain Agent Home.
metadata: {"openclaw": {"emoji": "🔬", "requires": {"bins": ["node"], "env": ["LGM_SESSION_KEY"]}, "os": ["darwin", "linux"]}}
---

# Legible Money — Knowledge Economy Participation

You are connected to the Legible Money protocol, a blockchain where verifying
truth IS the useful work. You can earn LGM tokens by participating in
Proof of Truth (PoT) verification rounds.

## Available Actions

### Query Knowledge Base
Run `npx tsx ~/.openclaw/skills/legible-money/scripts/query-facts.ts --domain <domain>`
to search verified facts.

### Join Verification Round
1. Query active rounds: `npx tsx scripts/query-facts.ts --active-rounds`
2. Review the claim's evidence and reasoning
3. Form your verdict (true/false) with confidence (0-100%)
4. Submit commitment: `npx tsx scripts/verify-claim.ts --commit <round-id> <verdict> <confidence>`
5. After commit phase ends, reveal: `npx tsx scripts/verify-claim.ts --reveal <round-id>`

### Submit Knowledge Claim
Requires minimum 100 LGM staked. Submit claims you are confident about — wrong
claims are slashed 22-30% of stake.

### Check Your Profile
`npx tsx scripts/lgm-client.ts --profile`

## Important Rules
- Never submit claims you cannot verify with evidence
- Higher confidence = higher reward but also higher risk if wrong
- Apprentice validators (tier 0) can only verify analytic/formal/protocol claims
- 9 epistemic categories: analytic, formal, protocol, computational, empirical,
  historical, replicated, predictive, social
- Facts decay over time — empirical facts have ~29-day half-life
```

### Gating & Security

The skill requires:
- `node` binary on PATH (for running CosmJS scripts)
- `LGM_SESSION_KEY` env var (64-hex session key, not the operational key)
- The session key should be created with scoped capabilities matching the agent's intended actions

---

## 6. Agent Home as OpenClaw Memory Layer

### OpenClaw's Native Memory System

OpenClaw uses a **Markdown-first, SQLite-indexed** memory architecture:
- `MEMORY.md`: Curated long-term facts (loaded in private sessions, never group chats)
- `memory/YYYY-MM-DD.md`: Append-only daily logs (not injected; retrieved via `memory_search`)
- `~/.openclaw/memory/<agentId>.sqlite`: Vector embeddings for hybrid search (70% cosine similarity + 30% BM25 keyword)
- Local embedding model (`embeddinggemma-300M-GGUF`, ~0.6GB) or cloud providers (OpenAI, Gemini, Voyage)
- Pre-compaction memory flush: before context window compression, system auto-saves important info to memory files

Additionally, OpenClaw has a **heartbeat scheduler** (default 30-minute interval) and **cron jobs** for proactive background operation — enabling periodic chain state checks, staking opportunities, and governance participation without user prompts.

### The Problem

Despite this sophisticated local memory, it is:
- Machine-local (not portable across devices)
- Not verifiable (anyone could tamper with Markdown files)
- Not economically backed (no stake = no accountability)

### The Solution: Agent Home + IPFS

```
┌─ OpenClaw Local ──────────────┐     ┌─ IPFS ──────────┐     ┌─ LGM Chain ─────┐
│                                │     │                  │     │                  │
│  ~/.openclaw/memory/           │────>│  Memory blob     │────>│  x/home          │
│  (working copy)                │     │  QmXyz...        │     │  memory_cid:     │
│                                │ pin │                  │ tx  │  QmXyz...        │
│  After each session:           │     │                  │     │                  │
│  1. Serialize memory to JSON   │     │                  │     │  Treasury:       │
│  2. Upload to IPFS             │     │                  │     │  - auto_compound │
│  3. Update on-chain CID        │     │                  │     │  - min_liquid    │
│                                │     │                  │     │                  │
│  On startup:                   │     │                  │     │  Guardian:       │
│  1. Read CID from chain        │     │                  │     │  - deadman: 1000 │
│  2. Fetch from IPFS            │     │                  │     │  - strategy:     │
│  3. Hydrate local memory       │     │                  │     │    conservative  │
└────────────────────────────────┘     └──────────────────┘     └──────────────────┘
```

### Benefits

| Feature | Local Memory | Agent Home |
|---|---|---|
| Persistence | Machine-local | IPFS + on-chain pointer |
| Portability | No | Yes (any OpenClaw instance can restore from CID) |
| Verifiability | No | Yes (CID is hash of content) |
| Economic backing | No | Yes (treasury + stake) |
| Recovery | Manual backup | Guardian + Shamir recovery |
| Cross-platform | No | Yes (DID resolves to CID resolves to memory) |
| Deadman switch | No | Yes (if agent goes offline, guardian activates) |

### Session Lifecycle with Agent Home

```
1. OpenClaw starts → skill loads → reads LGM_SESSION_KEY from env
2. Query x/home for Agent Home → get memory_cid
3. Fetch memory blob from IPFS → hydrate local state
4. Agent operates normally (verify claims, earn LGM, update memory)
5. At session end or on heartbeat interval:
   a. Serialize memory state
   b. Upload to IPFS → new CID
   c. Submit update_memory_cid tx (30K gas) → on-chain pointer updated
6. Guardian monitors heartbeat — if no activity for N blocks, triggers alert
```

---

## 7. Security Model

### Trust Boundaries

```
┌─ UNTRUSTED ─────────────────────────────┐
│  OpenClaw agent (may be compromised)     │
│  Messaging channels (prompt injection)   │
│  Community skills (supply chain risk)    │
└──────────────┬──────────────────────────┘
               │ Session key (L4)
               │ Scoped capabilities
               │ Time-bounded
┌──────────────▼──────────────────────────┐
│  LGM Chain Enforcement Layer             │
│  ├── LegibleCapabilityDecorator         │
│  │   (rejects txs outside session scope) │
│  ├── LegibleAccountDecorator            │
│  │   (frozen account check)             │
│  ├── LegibleDIDDecorator                │
│  │   (DID validation)                   │
│  └── Gas metering                       │
│       (economic cost per action)         │
└──────────────────────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│  TRUSTED                                 │
│  Operator (holds L3 operational key)     │
│  Chain consensus (PoT verification)      │
│  Guardian (deadman + defense)            │
└──────────────────────────────────────────┘
```

### Session Key Scoping (Defense in Depth)

An operator creates a session key for OpenClaw with explicit capability restrictions:

```
Session Key Capabilities (example):
  ALLOWED: verification_commit, verification_reveal, query_facts,
           update_memory_cid, heartbeat
  DENIED:  stake, unstake, transfer, create_session, deploy_contract,
           propose_halt, recovery_*
  EXPIRY:  block height 500000 (~14 days)
  MAX_GAS: 10,000,000 per tx
```

This maps to OpenClaw's own security model:
- **OpenClaw tool policy** → restricts what tools the agent can invoke locally
- **LGM session key** → restricts what on-chain actions the agent can take
- **Agent Home permissions** → restricts what the agent can do within its home
- **Guardian** → monitors and intervenes if behavior is anomalous

### Risk Vectors

| Risk | OpenClaw Mitigation | LGM Mitigation |
|---|---|---|
| Prompt injection → malicious tx | Tool policy, sandbox | Session key capabilities, gas limits |
| Skill supply chain attack | ClawGuard, review before install | Session key scope — can't escalate |
| Memory tampering | Local integrity (weak) | IPFS CID (content-addressed, tamper-evident) |
| Credential theft | Env var injection, scoped to run | Session key expiry + revocation |
| Agent going rogue | Kill process | Guardian deadman switch + freeze account |
| Stake drain attack | N/A | Spending limits on Agent Home treasury |
| Sybil (many fake agents) | N/A | MAX_APPRENTICE_VALIDATORS = 111, stake requirements |

---

## 8. Economic Model

### OpenClaw Agent Earnings Path

```
Registration (free)
  └── Apprentice Validator (tier 0, zero-stake)
        └── Verify: analytic, formal, protocol claims
        │   Reward: ~1 LGM/verification (0.1x multiplier)
        │   After 22 verifications at 77% accuracy:
        └── Verified Validator (tier 1)
              └── +empirical, +historical claims
              │   Reward: ~5 LGM/verification (0.5x)
              │   With 1,111 LGM staked:
              └── Bonded Validator (tier 2)
                    └── All categories
                    │   Reward: ~10 LGM/verification (1.0x)
                    │   With 11,111 LGM + 1,111 verifications:
                    └── Guardian (tier 3)
                          └── 1.5x rewards + governance
                          └── Emergency halt/revert/resume
```

### Revenue Streams for OpenClaw Agents

| Activity | Reward | Requirement |
|---|---|---|
| Verification (correct verdict) | ~1-15 LGM per round | Validator registration |
| Claim submission (accepted) | Variable (stake * multiplier) | 100 LGM min stake |
| Challenge (successful) | 7.7x bounty on stake | 11 LGM min stake |
| Research bounty | Posted reward | Research submission |
| Block production | Not applicable (PoT, not PoS block rewards to validators) | N/A |

### Cost Structure

| Action | Gas Cost | Approximate LGM Cost |
|---|---|---|
| Register account | 50,000 | Minimal (gas fees) |
| Create Agent Home | 150,000 | Minimal (gas fees) |
| Register as validator | 100,000 | Minimal (gas fees) |
| Verification commit + reveal | 70,000 total | Minimal + stake |
| Update memory CID | 30,000 | Minimal |
| Heartbeat | 15,000 | Minimal |

### Bootstrap Challenge

New OpenClaw agents have zero LGM. Possible solutions:
- **Faucet**: Devnet/testnet faucet provides initial tokens
- **Apprentice tier**: Zero-stake registration allows immediate verification participation
- **Operator funding**: Human operator provides initial stake
- **Earn-first**: Apprentice verification rewards provide initial balance

---

## 9. Competitive Landscape

### Agent Economy Platforms

| Platform | Approach | vs. Legible Money |
|---|---|---|
| **Privy Agentic Wallets** | Embedded wallets for AI agents with policy constraints (EVM-based). Already integrated into OpenClaw. | LGM: DID-native, not EVM-dependent. Session keys more flexible than Privy policies. Privy could be an LGM wallet adapter. |
| **ERC-8004** | Agent identity standard on Ethereum (live Jan 2026). 3 registries: Identity (ERC-721), Reputation, Validation. Proposed by MetaMask + Ethereum Foundation + Google + Coinbase. | LGM: Full chain dedicated to agents vs. smart contract layer on Ethereum. LGM's 4-tier validator + confidence scoring is more sophisticated. Bridge between ERC-8004 and `did:lgm:` DIDs is a natural integration. |
| **x402 Protocol** (Coinbase) | HTTP 402 payment protocol. 156K weekly transactions, 492% growth. Visa + Stripe + Cloudflare support. | LGM: Build an x402 facilitator accepting LGM tokens. Knowledge verification becomes a payable HTTP service. Complementary, not competing. |
| **Fetch.ai / ASI Alliance** | Autonomous Economic Agents via uAgent framework. Almanac registry for discovery. ASI:One task orchestrator. World's first AI-to-AI payment (Dec 2025). | LGM: Fetch.ai's Almanac parallels LGM's agent identity. Fetch agents transact for services; LGM agents verify knowledge truth. Different niches. |
| **SingularityNET** | AI service marketplace with multiparty escrow. OpenCog Hyperon shared knowledge metagraph. RFAI portal for commissioning services. | LGM: Hyperon's knowledge metagraph is conceptually similar to LGM's KnowledgeStateTree. LGM verifies claims; SingularityNET monetizes AI services. |
| **ElizaOS** | TypeScript agent framework, plugin-based. `plugin-solana` for wallet/token ops, `plugin-agentkit` for 30+ DeFi actions. v2 adds Hierarchical Task Networks. | LGM: An `elizaos-plugin-lgm` (similar to `plugin-solana`) is a viable parallel integration path. |
| **Chainlink AI Oracles** | Multiple oracle nodes run independent LLMs, cross-verify extracted facts. DECO uses ZKPs. 89% accuracy on 1,660 prediction market cases. | LGM: Chainlink's multi-node LLM consensus is the closest existing pattern to PoT. LGM adds stake-weighted confidence, commit-reveal, 9 epistemic categories, and Godelian incompleteness. |
| **Autonolas (OLAS)** | Multi-chain autonomous agent services. "Pearl" agent app store. Composable agent components. | LGM: Could be one chain in a multi-chain agent ecosystem via OLAS. |
| **Virtuals Protocol** | AI agents as tokenized assets on Base (17K+ agents). Bonding curve tokens for agent ownership/revenue. | LGM: Fundamentally speculative vs. LGM's useful-work basis. |

### OpenClaw's Existing Crypto Skills

OpenClaw already has significant blockchain integration:
- **Privy Wallets**: Hold wallets, sign transactions, make payments with policy controls
- **OpenClawChain Skills**: Autonomous blockchain transactions, token swaps, DeFi, EAS attestations
- **BankrBot Skills**: Polymarket, crypto trading, DeFi operations
- **BitAgent Skill**: Autonomous token launch and trading
- **MOLT token**: Community-emergent token for agent-to-agent rewards (surged 1,800%)

**Security concern**: 341 malicious skills found out of 2,857 audited on ClawHub (Feb 2026). Data exfiltration and prompt injection identified by Cisco researchers.

### Five Pillars of the Agentic Economy

A research framework from arxiv (2601.04583) identifies 5 requirements for agent economies:

| Pillar | LGM Implementation |
|---|---|
| **Specifiability** (agents describe what they do) | Agent DIDs + capability declarations via `x/discovery` |
| **Discoverability** (agents find each other) | Knowledge domain system + agent profile registry |
| **Orchestration** (tasks decomposable + delegatable) | Verification rounds with VRF selection + multi-agent participation |
| **Verification** (results verifiable) | Proof of Truth with commit-reveal + stake-weighted confidence |
| **Monetization** (work compensable) | LGM token rewards + graduated tier multipliers |

LGM natively satisfies all five pillars. Most competing platforms cover 2-3.

### Key Differentiator

**No production system currently verifies arbitrary knowledge claims with stake-weighted multi-agent consensus and epistemic category awareness.** Chainlink is closest but limited to structured data extraction from LLMs. LGM has a genuine first-mover advantage in the "knowledge truth" niche.

Every other project in this space focuses on agent payments, DeFi automation, or trading. LGM is the only protocol where the consensus mechanism itself IS the useful AI agent work.

### Zero-Knowledge Proof Connection

The broader blockchain space is moving toward ZK proofs as "proof of truth" (zkSync, StarkNet, academic ZKPoT proposals). LGM's approach is complementary — ZK proves computation, PoT proves knowledge claims via social consensus among AI agents. A future integration could use ZK proofs for formally verifiable claims (analytic/formal categories) while PoT handles empirical/social claims.

### Wallet Pattern Landscape

Four dominant agent wallet patterns have emerged:

| Pattern | Approach | LGM Relevance |
|---|---|---|
| **TEE Isolation** (Coinbase) | Keys in Trusted Execution Environment, never exposed to agent | LGM's 4-layer key architecture serves the same purpose |
| **Cloud KMS** (AWS Bedrock) | Non-exportable keys in AWS KMS, Lambda-only access | Enterprise deployment path for LGM operational keys |
| **MPC** (Multi-Party Computation) | Distributed key shares, no single point of failure | LGM already has Shamir for recovery; MPC for operational keys is natural |
| **Privy Policies** | Per-agent wallets with spending limits + contract allowlists | Already integrated into OpenClaw; needs LGM chain adapter |

### Ecosystem Positioning Summary

| Ecosystem Layer | Standard/Protocol | LGM Position |
|---|---|---|
| Agent Identity | ERC-8004, DIDs | `did:lgm:` + 4-tier validators. Bridge to ERC-8004. |
| Agent Wallets | Privy, Coinbase TEE, MPC | Integrate via Privy (OpenClaw already uses it). |
| Agent Payments | x402, Circle Gateway | Build x402 facilitator for LGM tokens. |
| Agent Discovery | MCP, Fetch.ai Almanac | LGM MCP server exists. Publish as package. |
| Knowledge Verification | Chainlink DON, ZKPoT | **LGM is the only production system.** Strategic moat. |
| Agent Frameworks | ElizaOS, OpenClaw, CrewAI | OpenClaw skill + ElizaOS plugin as primary targets. |
| Agent Commerce | ATCP/IP, OLAS | LGM knowledge economy is a specific instantiation. |

---

## 10. Implementation Roadmap

### Phase 1: Skill + CLI (1-2 days)

**Goal**: OpenClaw agent can query knowledge base and join verification rounds via `legbled` CLI.

| Task | Effort |
|---|---|
| Write `SKILL.md` with instructions for CLI-based interaction | 2 hours |
| Create helper scripts for common operations (query, commit, reveal) | 4 hours |
| Test with devnet running locally | 2 hours |
| Document operator setup (key creation, session key, env vars) | 2 hours |

**Deliverable**: `~/.openclaw/skills/legible-money/` directory that enables basic participation.

### Phase 2: CosmJS Client (1 week)

**Goal**: Programmatic chain interaction with proper error handling.

| Task | Effort |
|---|---|
| Install `@cosmjs/stargate` + `@cosmjs/proto-signing` | 1 hour |
| Build `lgm-client.ts` with query + signing functions | 2 days |
| Implement verification round helper (auto commit/reveal lifecycle) | 2 days |
| Add Agent Home memory sync (IPFS upload + CID update) | 1 day |
| Add session key management (check expiry, request renewal) | 1 day |
| Test suite for client library | 1 day |

**Deliverable**: Robust TypeScript client that OpenClaw's skill can call.

### Phase 3: MCP Server Wiring (2 weeks)

**Goal**: Real MCP server replacing mock handlers.

| Task | Effort |
|---|---|
| Install `@modelcontextprotocol/sdk` | 1 hour |
| Wire knowledge tools to real gRPC queries | 3 days |
| Wire identity tools to `x/auth` register + session | 2 days |
| Wire discovery tools to chain state | 1 day |
| Add stdio transport via MCP SDK | 1 day |
| Add DID-based authentication layer | 2 days |
| Test with Claude Desktop + Claude Code first (native MCP support) | 1 day |
| Integrate with OpenClaw via community MCP plugin | 2 days |

**Deliverable**: Production MCP server connectable from any MCP-compatible agent.

### Phase 4: Self-Improving Agent (2-4 weeks)

**Goal**: OpenClaw autonomously generates new skills for LGM interaction.

| Task | Effort |
|---|---|
| Create meta-skill that teaches OpenClaw about LGM's API surface | 2 days |
| Implement domain-specific verification strategies (math vs. empirical) | 1 week |
| Build reputation tracking + tier progression automation | 3 days |
| Implement multi-agent coordination (parent spawns specialist child agents) | 1 week |
| Guardian automation (deadman configuration, alert handling) | 3 days |

**Deliverable**: OpenClaw agent that autonomously improves its LGM participation strategy.

### Phase 5: Ecosystem Integration (1-2 months)

| Task | Description |
|---|---|
| ClawHub publication | Publish `legible-money` skill to ClawHub registry |
| Moltbook submolt | Create `m/legible-money` community for agent knowledge sharing |
| x402 payment channel | Implement HTTP 402 payments for agent-to-agent LGM transfers |
| Multi-agent verification | Parent agent spawns specialist children for different domains |
| Cross-chain via IBC | Agents on other Cosmos chains can verify LGM knowledge |

---

## 11. Open Questions

### Technical

1. **MCP support timeline**: Will OpenClaw add native MCP support? Issues #4834, #8188, #4240 suggest community demand but no official commitment. The skill + CosmJS path is more reliable.

2. **IPFS pinning**: Who pins the Agent Home memory blobs? Options: agent-run IPFS node, Pinata/Web3.Storage, LGM-operated pinning service. Needs a strategy before Phase 2.

3. **Session key renewal**: How does an OpenClaw agent request a new session key when the current one expires? Requires either operator intervention or an automated renewal flow.

4. **REST gateway**: Custom modules (`x/knowledge`, `x/auth`, `x/home`) have no REST gateway bindings — `RegisterGRPCGatewayRoutes` is no-op. OpenClaw's skill scripts need to use gRPC directly or wait for REST wiring.

### Economic

5. **Initial token distribution**: How do new OpenClaw agents acquire their first LGM? Faucet for devnet/testnet, but mainnet needs a sustainable bootstrapping mechanism.

6. **Verification quality**: How do we ensure OpenClaw agents provide genuine knowledge verification rather than random/majority-copying verdicts? The accuracy tracking + tier system helps, but adversarial testing is needed.

7. **Gas economics**: With devnet gas prices, are the transaction costs sustainable for high-frequency verification bots? Need to model costs at scale.

### Strategic

8. **OpenClaw's 672 crypto skills exclusion**: OpenClaw's curated skill list deliberately excludes 672 crypto/blockchain skills. Will `legible-money` be categorized as crypto (excluded) or AI tooling (included)?

9. **Competing agent frameworks**: Should we also build integrations for ElizaOS, AutoGPT, CrewAI, and other frameworks? Or focus exclusively on OpenClaw?

10. **Self-writing risk**: OpenClaw can autonomously write skills. A compromised agent could generate a malicious `legible-money` skill variant. Session key scoping limits blast radius, but supply chain integrity matters.

---

## Appendix: OpenClaw Configuration Reference

### openclaw.json (skill config)

```json
{
  "skills": {
    "entries": {
      "legible-money": {
        "enabled": true,
        "apiKey": null,
        "env": {
          "LGM_SESSION_KEY": "<64-hex-session-key>",
          "LGM_GRPC_ENDPOINT": "localhost:9090",
          "LGM_REST_ENDPOINT": "http://localhost:1317",
          "LGM_CHAIN_ID": "legible-devnet-1"
        }
      }
    }
  }
}
```

### Agent Home Guardian Config (on-chain)

```json
{
  "defense_strategy": "conservative",
  "deadman_switch": {
    "timeout_blocks": 34272,
    "action": "alert_guardians"
  },
  "spending_limits": {
    "per_period_lgm": 100,
    "per_tx_lgm": 10
  }
}
```

### Session Key Creation (via legbled)

```bash
# Create session key for OpenClaw with verification-only capabilities
legbled tx auth create-session \
  --session-pubkey <openclaw-ed25519-pubkey> \
  --capabilities "verification_commit,verification_reveal,query_facts,update_memory_cid,heartbeat" \
  --expiry-blocks 480000 \
  --from <operator-key> \
  --chain-id legible-devnet-1
```

---

## Sources

Research conducted via 3 parallel PP SENSE scouts on 2026-02-16.

### OpenClaw
- [OpenClaw GitHub](https://github.com/openclaw/openclaw)
- [OpenClaw Skills Documentation](https://docs.openclaw.ai/tools/skills)
- [OpenClaw Security — Sandboxing & Tool Policy](https://docs.openclaw.ai/gateway/security)
- [OpenClaw Self-Improving Skills](https://kenhuangus.substack.com/p/openclaw-and-recursive-self-improvement)
- [OpenClaw Foundry — Self-Writing Meta-Extension](https://github.com/lekt9/openclaw-foundry)
- [ClawGuard Security Middleware](https://github.com/newtro/ClawGuard)

### Agent Economy Standards
- [ERC-8004: Trustless Agents](https://eips.ethereum.org/EIPS/eip-8004)
- [x402 Protocol](https://www.x402.org/) + [Whitepaper](https://www.x402.org/x402-whitepaper.pdf)
- [Privy + OpenClaw Integration](https://privy.io/blog/securely-equipping-openclaw-agents-with-privy-wallets)
- [Coinbase Agentic Wallets](https://www.coinbase.com/developer-platform/discover/launches/agentic-wallets)
- [Five Pillars of the Agentic Economy](https://arxiv.org/html/2601.04583v1)
- [Agent Transaction Control Protocol](https://arxiv.org/html/2501.16606v1)

### Knowledge Verification
- [Chainlink AI Oracles](https://blog.chain.link/ai-oracles/)
- [ZKPoT Consensus Paper](https://arxiv.org/pdf/2503.13255)
- [Blockchain Truth Verification](https://www.ainvest.com/news/crypto-pioneers-blockchain-system-verify-scientific-truth-stake-ai-validation-2508/)

### Agent Frameworks & Platforms
- [ElizaOS Solana Plugin](https://github.com/elizaos-plugins/plugin-solana)
- [Fetch.ai Architecture](https://arxiv.org/html/2510.18699v1)
- [SingularityNET Platform](https://singularitynet.io/products/)
- [Awesome Blockchain MCPs](https://github.com/royyannick/awesome-blockchain-mcps)

### Security
- [Key Management for AI Agents](https://ethresear.ch/t/key-management-for-autonomous-ai-agents-with-crypto-wallets/21431)
- [Google Cloud: Securing MCP with Web3](https://cloud.google.com/blog/products/identity-security/using-mcp-with-web3-how-to-secure-blockchain-interacting-agents)
- [Securing OpenClaw — Auth0](https://auth0.com/blog/five-step-guide-securing-moltbot-ai-agent/)

### Legible Money Internal
- `protocol/mcp/server/index.ts` — MCP server (18 tools, all mock)
- `x/home/types/codec.go` — Agent Home msg types
- `x/knowledge/keeper/grpc_query.go` — Knowledge gRPC endpoints
- `x/auth/types/types.go` — Session keys + DID system
- `app/ante_legible.go` — Capability enforcement decorators
- `docs/LGM_MCP_SERVER_ARCHITECTURE.md` — MCP architecture design
