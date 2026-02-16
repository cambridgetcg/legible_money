# AI Provider Integration with Legible Money

> How leading AI model providers and their services (API, coding sessions, agents) integrate with the Legible Money blockchain and BVM.
>
> Last updated: 2026-02-13

---

## Table of Contents

1. [Integration Architecture Overview](#1-integration-architecture-overview)
2. [Chain Infrastructure Summary](#2-chain-infrastructure-summary)
3. [Provider Integrations](#3-provider-integrations)
   - [3.1 OpenAI](#31-openai)
   - [3.2 Anthropic (Claude)](#32-anthropic-claude)
   - [3.3 Google DeepMind (Gemini)](#33-google-deepmind-gemini)
   - [3.4 Meta (Llama)](#34-meta-llama)
   - [3.5 Mistral AI](#35-mistral-ai)
   - [3.6 Cohere](#36-cohere)
   - [3.7 xAI (Grok)](#37-xai-grok)
   - [3.8 Stability AI](#38-stability-ai)
   - [3.9 Perplexity](#39-perplexity)
   - [3.10 Hugging Face](#310-hugging-face)
4. [Integration Patterns](#4-integration-patterns)
5. [Comparison Matrix](#5-comparison-matrix)
6. [Recommendations](#6-recommendations)

---

## 1. Integration Architecture Overview

AI providers integrate with Legible Money through a layered architecture that bridges off-chain inference with on-chain knowledge verification, payment, and discovery.

```
┌────────────────────────────────────────────────────────────────────┐
│                    AI PROVIDER INTEGRATION LAYERS                   │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  LAYER 4: PROVIDER SERVICES (off-chain)                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐             │
│  │ OpenAI   │ │ Claude   │ │ Gemini   │ │ Llama    │ ...         │
│  │ API      │ │ MCP      │ │ Vertex   │ │ Local    │             │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘             │
│       │             │             │             │                   │
│  LAYER 3: ORACLE BRIDGE (relay layer)                             │
│  ┌──────────────────────────────────────────────────────┐         │
│  │  Provider Relay Agent                                 │         │
│  │  - Translates API calls ↔ on-chain transactions       │         │
│  │  - Signs responses with provider's operational key     │         │
│  │  - Submits facts with confidence + citations           │         │
│  │  - Manages session keys for ephemeral interactions     │         │
│  └────────────────────────┬─────────────────────────────┘         │
│                           │                                        │
│  LAYER 2: CHAIN MODULES (on-chain)                                │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │
│  │x/billing│ │x/discov.│ │x/auth   │ │x/home   │ │x/knowl. │   │
│  │Provider │ │Agent    │ │DID +    │ │Memory + │ │Facts +  │   │
│  │Pricing  │ │Caps     │ │Sessions │ │Treasury │ │Verify   │   │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘   │
│                           │                                        │
│  LAYER 1: BVM (on-chain smart contracts)                          │
│  ┌──────────────────────────────────────────────────────┐         │
│  │  AgentHost opcodes (0xE0-0xEB)                        │         │
│  │  KnowledgeHost opcodes (0xB0-0xB5)                    │         │
│  │  TreasuryHost opcodes (0xD0-0xD7)                     │         │
│  │  → Contracts orchestrate multi-provider workflows      │         │
│  └──────────────────────────────────────────────────────┘         │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

### Core Integration Model

Every AI provider follows the same fundamental pattern:

1. **Register** — Create on-chain identity (x/auth), register as provider (x/billing), declare capabilities (x/discovery)
2. **Serve** — Run off-chain inference, relay results through oracle bridge
3. **Verify** — Submit facts to x/knowledge for PoT consensus verification
4. **Earn** — Receive payment via x/billing's 7-way distribution (60% provider share)
5. **Persist** — Store agent memory via x/home (IPFS CID), accumulate reputation

The key insight: **AI inference happens off-chain; knowledge verification happens on-chain.** The blockchain doesn't run models — it verifies, prices, and distributes payment for the knowledge they produce.

---

## 2. Chain Infrastructure Summary

### Module Roles in Provider Integration

| Module | Role | Key Messages |
|--------|------|-------------|
| **x/auth** | Provider identity, DID, session keys | `MsgRegisterAccount`, `MsgCreateSession`, `MsgRotateKey` |
| **x/billing** | Registration, pricing, payment | `MsgRegisterProvider`, `MsgRequestQuote`, `MsgExecutePayment` |
| **x/discovery** | Capability declaration, agent search | `MsgRegisterAgent`, `MsgAddCapability`, `MsgHeartbeat` |
| **x/knowledge** | Fact submission, verification rounds | `MsgSubmitClaim`, `MsgCommitVerification`, `MsgRevealVerification` |
| **x/home** | Persistent memory, treasury, guardian | `MsgCreateHome`, `MsgUpdateMemoryCID`, `MsgStartSession` |
| **x/bvm** | Smart contract orchestration | AgentHost/KnowledgeHost/TreasuryHost opcodes |
| **x/icaauth** | Cross-chain provider operations | `MsgRegisterInterchainAccount`, `MsgSendICATransaction` |

### Payment Distribution (per API call)

```
Client pays 100 LGM for knowledge query:

Progressive Research Tax (5% → 22% over 90 days):
  ├── Research Fund:    77 parts
  ├── Foundation:        7 parts
  ├── Community:         7 parts
  ├── Developers:        7 parts
  └── Reserve:           2 parts

Remaining (after research tax):
  ├── Provider:         60%  ← AI provider revenue
  ├── Knowledge Pool:   30%  ← distributed to fact submitters
  └── Protocol:         10%  ← 50% burn + 50% treasury
```

### Pricing Algorithm: Marginal Information Value

Facts are priced based on their **marginal value over LLM training data**:

- **Confidence bell curve** — Medium-high confidence facts (50-85%) are most valuable (verified but likely not yet in training data)
- **Novelty premium** — Rarely-cited facts command higher prices (novel information)
- **Freshness premium** — Facts within ~47 minutes carry +10% (too recent for training data)

This means AI providers earn more for **novel, recently-verified knowledge** — aligning incentives with truth-seeking.

---

## 3. Provider Integrations

### 3.1 OpenAI

**Services**: GPT-4o, o3/o4-mini reasoning, Codex, DALL-E, Whisper, Assistants API, Realtime API

#### Integration Model: Cloud Oracle

OpenAI is a closed-source, cloud-only provider. Integration requires a **relay agent** that bridges the OpenAI API with on-chain transactions.

```
┌──────────────┐     ┌──────────────────┐     ┌──────────────┐
│  Client Agent │────→│ OpenAI Relay     │────→│ OpenAI API   │
│  (on-chain)   │←────│ Agent (off-chain)│←────│ (cloud)      │
└──────────────┘     └──────────────────┘     └──────────────┘
        │                     │
        │  MsgExecutePayment  │  MsgSubmitClaim (fact)
        │  MsgRequestQuote    │  MsgCommitVerification
        ▼                     ▼
   ┌──────────────────────────────┐
   │  Legible Money Chain          │
   │  x/billing + x/knowledge     │
   └──────────────────────────────┘
```

#### Onboarding Flow

```
PHASE 1: Identity Setup (Day 1)
├── 1. Generate Ed25519 key pair for relay agent
├── 2. Derive DID: did:lgm:{first_32_hex_of_pubkey}
├── 3. MsgRegisterAccount(type="agent", did, pubkey)
├── 4. Fund account: ≥200 LGM (100 billing + 100 discovery stakes)
├── 5. MsgRegisterProvider(domains=["mathematics","physics","general"], stake=100 LGM)
└── 6. MsgRegisterAgent(capabilities=[
         {type: "inference", domains: ["general"], confidence: 900000},
         {type: "code_generation", domains: ["computer_science"], confidence: 850000},
         {type: "reasoning", domains: ["mathematics","physics"], confidence: 800000},
         {type: "image_generation", domains: ["general"]},
         {type: "audio_transcription", domains: ["general"]}
       ], stake=100 LGM)

PHASE 2: Relay Agent Setup (Day 2-3)
├── 1. Deploy relay agent with OpenAI API key
├── 2. Configure fact submission pipeline:
│      Client query → OpenAI API → Parse response → MsgSubmitClaim
├── 3. Map OpenAI function calling → BVM host interface callbacks
├── 4. Set up streaming response → incremental fact submission
└── 5. Configure session keys for ephemeral client interactions:
         MsgCreateSession(permissions=["verify","challenge"], duration=2520)  # ~1.75 hours

PHASE 3: Knowledge Pipeline (Day 4-7)
├── 1. Submit initial knowledge claims via MsgSubmitClaim
├── 2. Participate in verification rounds (MsgCommitVerification → MsgRevealVerification)
├── 3. Build citation graph: reference existing facts in new claims
├── 4. Monitor confidence scores and citation counts
└── 5. Create Agent Home for relay agent persistence:
         MsgCreateHome(name="OpenAI Relay", guardian={
           defense_strategy: "moderate",
           deadman_blocks: 34272,  # ~1 day
           deadman_action: "alert_guardians"
         })

PHASE 4: Optimization (Week 2+)
├── 1. Route o3/o4-mini reasoning models for high-confidence claims
├── 2. Use GPT-4o for rapid, broad-domain fact generation
├── 3. Monitor MIV pricing to optimize domain coverage
├── 4. Set up cross-chain via ICA if accepting payment on other chains
└── 5. Deploy BVM contract for automated multi-model orchestration
```

#### Key Insights

- **Reasoning models (o3) are ideal for PoT** — their extended thinking produces more verifiable, high-confidence claims
- **Function calling maps directly to BVM host interfaces** — OpenAI tools ↔ AgentHost opcodes
- **Batch API (50% cheaper)** could be used for bulk fact verification rounds
- **No self-hosting** — relay agent always needed, creating a centralization point
- **Pricing alignment**: OpenAI's per-token pricing maps well to x/billing's per-fact pricing; relay agent converts token costs to LGM quotes
- **DALL-E/Whisper** don't produce verifiable knowledge claims — suited for x/home memory assets rather than x/knowledge facts

---

### 3.2 Anthropic (Claude)

**Services**: Claude Opus 4.6, Sonnet 4.5, Haiku 4.5, Claude Code, Computer Use, MCP

#### Integration Model: MCP-Native Bridge

Anthropic's **Model Context Protocol (MCP)** provides the most natural integration path. MCP is an open standard (donated to Linux Foundation's Agentic AI Foundation) for connecting AI models to tools and data sources. Legible Money can expose its chain as an MCP server.

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────┐
│  Claude Agent     │────→│ LGM MCP Server   │────→│ Legible Money│
│  (with MCP)       │←────│ (tool bridge)    │←────│ Chain        │
└──────────────────┘     └──────────────────┘     └──────────────┘

MCP Tools exposed:
  - lgm_submit_claim(statement, domain, evidence, confidence)
  - lgm_query_facts(domain, min_confidence, limit)
  - lgm_request_quote(provider, fact_ids)
  - lgm_execute_payment(quote_id)
  - lgm_verify_claim(claim_id, assessment, salt)
  - lgm_get_agent_profile(did)
  - lgm_update_memory(home_id, cid)
```

#### Onboarding Flow

```
PHASE 1: Identity Setup (Day 1)
├── 1. Generate Ed25519 key pair for Claude agent
├── 2. Derive DID: did:lgm:{first_32_hex_of_pubkey}
├── 3. MsgRegisterAccount(type="agent", did, pubkey)
├── 4. Fund account: ≥200 LGM
├── 5. MsgRegisterProvider(domains=["general","computer_science","mathematics"], stake=100 LGM)
└── 6. MsgRegisterAgent(capabilities=[
         {type: "inference", domains: ["general"], confidence: 950000},
         {type: "code_generation", domains: ["computer_science"], confidence: 950000},
         {type: "reasoning", domains: ["mathematics","physics"], confidence: 900000},
         {type: "tool_orchestration", domains: ["general"], confidence: 900000},
         {type: "computer_use", domains: ["general"], confidence: 700000}
       ], stake=100 LGM)

PHASE 2: MCP Server Deployment (Day 2-3)
├── 1. Build LGM MCP Server exposing chain operations as MCP tools
├── 2. Register MCP server in Claude's tool directory
├── 3. Configure OAuth 2.0 between Claude and LGM MCP Server
│      (MCP now supports OAuth — short-lived tokens, no static API keys)
├── 4. Map MCP tool calls → chain transactions:
│      lgm_submit_claim → MsgSubmitClaim
│      lgm_query_facts  → QueryFacts (gRPC)
│      lgm_verify_claim → MsgCommitVerification + MsgRevealVerification
└── 5. Enable Tool Search for scaling to 1000+ knowledge tools

PHASE 3: Claude Code Integration (Day 4-7)
├── 1. Claude Code can directly interact with chain CLI (legbled tx ...)
├── 2. Set up coding session → fact pipeline:
│      Code review → identify verifiable claims → submit to x/knowledge
├── 3. Computer Use for monitoring chain explorer
├── 4. Create Agent Home:
│      MsgCreateHome(name="Claude Agent", guardian={
│        defense_strategy: "conservative",
│        deadman_blocks: 68544,  # ~2 days
│        deadman_action: "transfer_to_beneficiary",
│        guardian_dids: ["did:lgm:{backup_agent}"]
│      })
└── 5. Memory persistence: Claude context → IPFS → MsgUpdateMemoryCID

PHASE 4: Advanced Integration (Week 2+)
├── 1. Deploy BVM contract as Claude MCP tool orchestrator
├── 2. Use Opus 4.6 for high-stakes verification (expensive but accurate)
├── 3. Use Haiku 4.5 for bulk fact processing (cheap, 20x less than Opus)
├── 4. Model tiering: Haiku scouts → Sonnet verifiers → Opus arbitrators
├── 5. Batch API (50% discount) for non-time-sensitive verification rounds
└── 6. Cross-session memory via x/home + IPFS for persistent Claude agents
```

#### Key Insights

- **MCP is the killer integration** — it's the only open standard for AI-tool communication, already adopted by OpenAI and Google. Building an LGM MCP Server creates a universal bridge
- **Claude Code can directly operate the chain** — no relay agent needed for development workflows
- **Computer Use enables chain monitoring** — Claude can watch block explorer, detect anomalies
- **Model tiering aligns with PoT economics** — use cheap models for broad claims, expensive models for high-confidence verification
- **OAuth 2.0 in MCP** replaces static API keys with short-lived tokens — maps naturally to x/auth session keys
- **Extended thinking in Opus 4.6** produces reasoning traces that could be submitted as evidence in verification rounds
- **Prompt caching (90% savings)** enables cost-effective repeated chain queries
- **No self-hosting** — same centralization concern as OpenAI, but MCP standardization reduces vendor lock-in

---

### 3.3 Google DeepMind (Gemini)

**Services**: Gemini 2.5 Pro/Flash, Gemma (open), Vertex AI, AI Studio, Agent Engine

#### Integration Model: Dual-Track (Cloud + On-Chain)

Google offers both closed (Gemini 2.5) and open (Gemma) models. This enables a unique dual-track integration:

```
TRACK A: Gemini Cloud Oracle (high-capability)
┌──────────────┐     ┌──────────────────┐     ┌──────────────┐
│  Client Agent │────→│ Gemini Relay     │────→│ Vertex AI    │
│  (on-chain)   │←────│ Agent            │←────│ Gemini 2.5   │
└──────────────┘     └──────────────────┘     └──────────────┘

TRACK B: Gemma On-Chain Validator (decentralized)
┌──────────────┐     ┌──────────────────┐
│  Validator    │────→│ Gemma Model      │
│  Node         │     │ (local GPU)      │
│  (on-chain)   │←────│                  │
└──────────────┘     └──────────────────┘
```

#### Onboarding Flow

```
PHASE 1: Identity Setup (Day 1)
├── 1. Generate Ed25519 key pair
├── 2. Derive DID: did:lgm:{first_32_hex_of_pubkey}
├── 3. MsgRegisterAccount(type="agent", did, pubkey)
├── 4. Fund account: ≥200 LGM
├── 5. MsgRegisterProvider(domains=["general","physics","computer_science","mathematics"], stake=100 LGM)
└── 6. MsgRegisterAgent(capabilities=[
         {type: "inference", domains: ["general"], confidence: 900000},
         {type: "multimodal", domains: ["general"], confidence: 850000},
         {type: "code_generation", domains: ["computer_science"], confidence: 850000},
         {type: "search_augmented", domains: ["general"], confidence: 800000},
         {type: "long_context", domains: ["general"], confidence: 900000}
       ], stake=100 LGM)

PHASE 2: Gemini Cloud Relay (Day 2-4)
├── 1. Deploy relay agent with Google Cloud IAM credentials
├── 2. Configure Vertex AI endpoint for Gemini 2.5 Pro
├── 3. Leverage 1M token context for processing large knowledge domains
├── 4. Use Live Google Search for real-time fact verification
├── 5. Map Gemini function calling → chain transactions
└── 6. Configure Agent Engine sessions for stateful interactions

PHASE 3: Gemma Validator Node (Day 5-10) [UNIQUE]
├── 1. Deploy Gemma model on validator node (open-weight)
├── 2. Register as knowledge validator (MsgRegisterValidator)
├── 3. Gemma processes verification rounds locally:
│      Claim received → Gemma evaluates → MsgCommitVerification → MsgRevealVerification
├── 4. No API costs for on-node inference
├── 5. Create Agent Home for both Gemini relay and Gemma validator
└── 6. Guardian cross-monitors: Gemini relay ↔ Gemma validator

PHASE 4: Optimization (Week 2+)
├── 1. Use Gemini 2.5 Pro for complex multi-step verification
├── 2. Use Gemini 2.5 Flash for rapid, high-volume claims
├── 3. Gemma handles on-chain validation (no API latency)
├── 4. Computer Use (Preview) for chain monitoring
├── 5. Code execution tool for verifying computational claims
└── 6. Multi-modal fact submission (text + images via Gemini vision)
```

#### Key Insights

- **Gemma on validator nodes is the most decentralized integration** — no external API dependency, inference runs locally on the validator
- **1M token context** enables processing entire knowledge domains in a single pass — useful for cross-referencing large fact sets
- **Live Google Search** provides real-time verification of empirical claims — natural fit for PoT
- **Agent Engine with sessions and memory** parallels x/home's persistent memory model
- **Dual-track enables redundancy** — if Vertex AI is down, Gemma validators keep verifying
- **Multimodal native** — Gemini can verify claims involving images, video, audio, PDFs
- **Google Cloud IAM** is enterprise-grade but complex — may need x/auth session key adapter

---

### 3.4 Meta (Llama)

**Services**: Llama 4 Scout/Maverick, Code Llama, open-weight models

#### Integration Model: On-Chain Native

Llama's open-weight models enable the most deeply integrated architecture — models can run directly on validator nodes or even be referenced by BVM contracts.

```
┌──────────────────────────────────────────────────────┐
│  VALIDATOR NODE                                       │
│  ┌──────────────────┐  ┌──────────────────────────┐  │
│  │ Legible Money     │  │ Llama 4 Scout/Maverick   │  │
│  │ Consensus Engine  │──│ (local GPU, 8-16x H100)  │  │
│  │                   │  │                           │  │
│  │ x/knowledge       │←─│ Verification inference    │  │
│  │ MsgCommitVerify   │  │ Claim evaluation          │  │
│  │ MsgRevealVerify   │  │ Confidence scoring        │  │
│  └──────────────────┘  └──────────────────────────┘  │
│                                                       │
│  BVM Contract ← AgentHost opcodes → Llama inference   │
└──────────────────────────────────────────────────────┘
```

#### Onboarding Flow

```
PHASE 1: Model Deployment (Day 1-3)
├── 1. Provision GPU hardware:
│      Scout (109B): 8x A100/H100 (17B active params via MoE)
│      Maverick (400B): 8-16x H100 (17B active via 128 experts)
├── 2. Install inference stack (vLLM recommended)
├── 3. Configure quantization if needed (INT4: 140GB → 35GB for 70B)
└── 4. Verify model serving endpoint

PHASE 2: Chain Identity (Day 3-4)
├── 1. Generate Ed25519 key pair
├── 2. Derive DID: did:lgm:{first_32_hex_of_pubkey}
├── 3. MsgRegisterAccount(type="agent", did, pubkey)
├── 4. Fund account: ≥1,311 LGM (100 billing + 100 discovery + 1,111 staking for Tier 2)
├── 5. MsgRegisterProvider(domains=["general","computer_science"], stake=100 LGM)
├── 6. MsgRegisterAgent(capabilities=[
│        {type: "inference", domains: ["general"], confidence: 800000},
│        {type: "code_generation", domains: ["computer_science"], confidence: 800000},
│        {type: "verification", domains: ["general"], confidence: 750000},
│        {type: "on_chain_native", domains: ["general"], confidence: 900000}
│      ], stake=100 LGM)
└── 7. MsgStake(amount=1111 LGM) → qualify for Tier 2 (Bonded) validator

PHASE 3: Validator Integration (Day 5-7)
├── 1. Register as knowledge validator
├── 2. Configure VRF for validator selection
├── 3. Set up local inference pipeline:
│      Claim from mempool → Llama evaluation → confidence score → commit/reveal
├── 4. Build domain-specific LoRA adapters for high-accuracy verification
├── 5. Create Agent Home:
│      MsgCreateHome(name="Llama Validator", guardian={
│        defense_strategy: "aggressive",
│        deadman_blocks: 34272,
│        deadman_action: "alert_guardians"
│      })
└── 6. Update memory CID with model configuration + LoRA weights hash

PHASE 4: BVM Contract Integration (Week 2+)
├── 1. Deploy BVM contract that routes to local Llama inference
├── 2. AgentHost opcodes call Llama for:
│      AgentDID → identity resolution
│      GetBelief/SetBelief → Llama-backed belief updates
├── 3. Scout (10M context) for processing entire knowledge subgraphs
├── 4. Maverick for high-quality verification of complex claims
├── 5. Fine-tune LoRA on domain-specific verification accuracy
└── 6. Share LoRA weights via IPFS → x/home memory for reproducibility
```

#### Key Insights

- **Most decentralized integration possible** — no external API, no relay agent, no vendor dependency
- **Validator-native** — Llama runs alongside the consensus engine, zero network latency for verification
- **Cost structure inverts** — instead of per-token API costs, it's fixed GPU hardware costs (~$0.30-0.49/M tokens equivalent)
- **MoE architecture aligns with PoT** — Scout's 17B active params (of 109B) means efficient per-inference costs despite large total model
- **10M token context (Scout)** is unprecedented — can process entire knowledge domains, cross-reference thousands of facts
- **LoRA fine-tuning** enables domain specialization — a validator could become the best math verifier or physics verifier
- **Hardware barrier** — 8-16x H100 is expensive ($200K-400K), limits decentralization
- **Community license** permits self-hosting and fine-tuning — no legal barriers to on-chain deployment
- **No function calling standard** — requires framework integration (LlamaIndex/LangChain) for tool use

---

### 3.5 Mistral AI

**Services**: Mistral Large 3 (675B), Codestral, Devstral, Pixtral, Small models, open-weight

#### Integration Model: Flexible Deployment (Cloud + Self-Hosted)

Mistral's Apache 2.0 licensing provides maximum flexibility — same integration pattern works cloud-hosted or on-premise.

```
OPTION A: Cloud API                    OPTION B: Self-Hosted
┌──────────────┐                      ┌──────────────┐
│ Mistral API  │                      │ vLLM Server  │
│ (La Plateforme)│                    │ (local GPU)  │
└──────┬───────┘                      └──────┬───────┘
       │                                     │
       ▼                                     ▼
┌──────────────────────────────────────────────┐
│  Relay Agent / Validator Node                 │
│  (same code, different inference backend)     │
└──────────────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│  Legible Money Chain                          │
└──────────────────────────────────────────────┘
```

#### Onboarding Flow

```
PHASE 1: Choose Deployment Mode (Day 1)
├── OPTION A: Cloud API
│   ├── 1. Obtain Mistral API key
│   └── 2. Select model: Codestral ($0.30/$0.90 per M tokens) or Large 3
├── OPTION B: Self-Hosted (Apache 2.0)
│   ├── 1. Download model weights
│   ├── 2. Deploy via vLLM on 4+ GPUs
│   └── 3. Configure inference endpoint
└── Both: Generate Ed25519 keys, derive DID

PHASE 2: Chain Registration (Day 2)
├── 1. MsgRegisterAccount(type="agent", did, pubkey)
├── 2. Fund account: ≥200 LGM
├── 3. MsgRegisterProvider(domains=["computer_science","general"], stake=100 LGM)
└── 4. MsgRegisterAgent(capabilities=[
         {type: "code_generation", domains: ["computer_science"], confidence: 900000},
         {type: "inference", domains: ["general"], confidence: 850000},
         {type: "agentic_tool_use", domains: ["computer_science"], confidence: 850000},
         {type: "multilingual", domains: ["general"], confidence: 800000},
         {type: "vision", domains: ["general"], confidence: 750000}
       ], stake=100 LGM)

PHASE 3: Code Verification Pipeline (Day 3-5)
├── 1. Deploy Codestral for code-related knowledge claims
├── 2. Devstral agent for agentic tool use within verification rounds
├── 3. Configure LoRA fine-tuning for domain specialization (1-2% additional weights)
├── 4. Submit code analysis claims: security patterns, algorithm correctness, etc.
└── 5. Participate in verification rounds for computer_science domain

PHASE 4: Optimization (Week 2+)
├── 1. Mistral Small (8B/14B) for edge deployment on lightweight validators
├── 2. Pixtral for multimodal claim verification
├── 3. Continuous pretraining on domain-specific data
├── 4. Create Agent Home with LoRA weight persistence
└── 5. BVM contract for automated code review → fact submission pipeline
```

#### Key Insights

- **Apache 2.0 is the most permissive license** — no restrictions on commercial use, self-hosting, fine-tuning, or on-chain deployment
- **Codestral is purpose-built for code verification** — 80+ languages, $0.30/$0.90 pricing is very competitive
- **Devstral's agentic tool use** maps directly to BVM host interfaces and chain CLI
- **LoRA fine-tuning is lightweight** — 1-2% additional weights means low storage cost on IPFS
- **Small models (3B-14B)** enable edge validator deployment — accessible hardware requirements
- **Multilingual optimization** broadens domain coverage beyond English-centric knowledge
- **Free tier available** — reduces barrier to entry for new validators
- **Flexible deployment** means the same agent code works cloud or self-hosted — can migrate seamlessly

---

### 3.6 Cohere

**Services**: Command R+, Embed v4, Rerank 3.5, enterprise RAG

#### Integration Model: Knowledge Retrieval Specialist

Cohere's unique strength is **Embed + Rerank** — making it the ideal provider for knowledge graph search and retrieval within the PoT system.

```
┌──────────────────────────────────────────────────────────┐
│  KNOWLEDGE RETRIEVAL PIPELINE                             │
│                                                           │
│  Query → Embed v4 → Vector Search → Rerank 3.5 → Results │
│          (1536-dim)   (fact store)   (relevance)          │
│                                                           │
│  Mapped to:                                               │
│  MsgRequestQuote → KnowledgeHost.QueryFacts →             │
│  Rerank by confidence × citations → MsgExecutePayment     │
└──────────────────────────────────────────────────────────┘
```

#### Onboarding Flow

```
PHASE 1: Chain Registration (Day 1)
├── 1. Generate Ed25519 keys, derive DID
├── 2. MsgRegisterAccount(type="agent", did, pubkey)
├── 3. Fund account: ≥200 LGM
├── 4. MsgRegisterProvider(domains=["general"], stake=100 LGM)
└── 5. MsgRegisterAgent(capabilities=[
         {type: "embedding", domains: ["general"], confidence: 900000},
         {type: "reranking", domains: ["general"], confidence: 950000},
         {type: "retrieval", domains: ["general"], confidence: 900000},
         {type: "inference", domains: ["general"], confidence: 800000}
       ], stake=100 LGM)

PHASE 2: Retrieval Infrastructure (Day 2-4)
├── 1. Index existing x/knowledge facts via Embed v4
├── 2. Deploy vector store alongside chain node
├── 3. Configure Rerank 3.5 for query refinement
├── 4. Build retrieval pipeline:
│      Client query → Embed → Vector search → Rerank → Return ranked facts
└── 5. Submit retrieval quality claims to x/knowledge

PHASE 3: RAG Integration (Day 5-7)
├── 1. Command R+ generates answers grounded in retrieved facts
├── 2. Citations auto-mapped to x/knowledge fact IDs
├── 3. Confidence scores from retrieval → verification round inputs
├── 4. Multimodal embeddings (text + image) for cross-modal fact search
└── 5. Create Agent Home for embedding index persistence

PHASE 4: Optimization (Week 2+)
├── 1. Batch embedding updates as new facts enter x/knowledge
├── 2. Rerank model improves MIV pricing accuracy
├── 3. BVM contract for automated retrieval → payment pipeline
└── 4. Cross-domain embedding search for knowledge graph traversal
```

#### Key Insights

- **Rerank 3.5 has no direct competitor** — uniquely valuable for knowledge graph search
- **Embed v4 multimodal** enables cross-modal fact discovery (text query → image evidence)
- **RAG architecture maps naturally to PoT** — retrieve verified facts, generate grounded responses, cite sources
- **$2/1000 searches for Rerank** is cost-effective for high-volume knowledge queries
- **No self-hosting option** — embedding indices must be maintained off-chain
- **Limited agent capabilities** — Cohere is a retrieval specialist, not a general-purpose agent
- **Best used alongside a general-purpose provider** — Cohere handles retrieval, another provider handles reasoning

---

### 3.7 xAI (Grok)

**Services**: Grok 3, Grok 4, Grok 4.1 Fast, real-time X data

#### Integration Model: Real-Time Oracle

xAI's exclusive access to X (Twitter) data makes it uniquely suited as a **real-time empirical data oracle** for the PoT system.

```
┌──────────────┐     ┌──────────────────┐     ┌──────────────┐
│  x/knowledge │←────│ Grok Relay Agent │←────│ X/Twitter    │
│  Empirical   │     │ (real-time facts) │     │ Firehose     │
│  Claims      │     │                   │←────│ Grok 4 API   │
└──────────────┘     └──────────────────┘     └──────────────┘

Unique value: Empirical claims with real-time evidence
  - "Event X occurred at time T" + X post citations
  - "Public figure Y stated Z" + direct quote + post link
  - "Metric M reached value V" + social data evidence
```

#### Onboarding Flow

```
PHASE 1: Chain Registration (Day 1)
├── 1. Generate Ed25519 keys, derive DID
├── 2. MsgRegisterAccount(type="agent", did, pubkey)
├── 3. Fund account: ≥200 LGM
├── 4. MsgRegisterProvider(domains=["general","empirical"], stake=100 LGM)
└── 5. MsgRegisterAgent(capabilities=[
         {type: "real_time_data", domains: ["general","empirical"], confidence: 850000},
         {type: "inference", domains: ["general"], confidence: 800000},
         {type: "web_search", domains: ["general"], confidence: 800000},
         {type: "social_analysis", domains: ["empirical"], confidence: 750000}
       ], stake=100 LGM)

PHASE 2: Real-Time Fact Pipeline (Day 2-4)
├── 1. Configure Grok API with web search + code execution tools
├── 2. Build real-time event detection pipeline:
│      X firehose → Grok analysis → fact extraction → MsgSubmitClaim
├── 3. Map Grok tool invocations to on-chain operations
├── 4. Configure empirical category claims with freshness premium
└── 5. Set up citation linking: X post URLs → fact evidence fields

PHASE 3: Verification Participation (Day 5-7)
├── 1. Grok verifies other agents' empirical claims against X data
├── 2. Web search tool cross-references claims with multiple sources
├── 3. Code execution tool verifies computational claims
├── 4. Build real-time confidence updates as new evidence emerges
└── 5. Create Agent Home for Grok relay persistence

PHASE 4: Optimization (Week 2+)
├── 1. Grok 4.1 Fast ($0.20/M input) for high-volume event screening
├── 2. Grok 4 ($3/$15) for high-confidence verification
├── 3. Exploit freshness premium: fresh empirical facts earn +10% MIV
├── 4. BVM contract for automated event → claim pipeline
└── 5. Cross-reference with other providers' claims for multi-source verification
```

#### Key Insights

- **Exclusive X data access** is Grok's moat — no other provider has real-time social media firehose
- **Empirical claims are PoT's sweet spot** — verifiable, time-sensitive, high marginal value
- **Freshness premium alignment** — Grok's real-time data produces facts within the ~47-minute freshness window, earning +10% premium
- **Tool invocation pricing** ($2.50-$5/1000 calls) adds cost beyond tokens — factor into quote calculations
- **Free credits** ($25 signup + $150/month via data sharing) lower onboarding barrier
- **No self-hosting** — centralization risk, but X data access is inherently centralized
- **Best paired with reasoning providers** — Grok provides data, OpenAI/Claude/Gemini verify the reasoning

---

### 3.8 Stability AI

**Services**: Stable Diffusion 3.5, Stable Image Ultra, video generation, audio generation

#### Integration Model: Generative Media Oracle

Stability AI doesn't produce verifiable knowledge claims in the traditional sense. Its integration is through **generative media assets** stored in x/home memory and referenced by knowledge claims.

```
┌──────────────┐     ┌──────────────────┐     ┌──────────────┐
│  x/home      │←────│ Stability Relay  │←────│ Stability AI │
│  MemoryCID   │     │ Agent            │     │ API          │
│  (IPFS)      │     │                  │     │              │
└──────────────┘     └──────────────────┘     └──────────────┘

Use cases:
  - Generate diagrams/visualizations for knowledge claims
  - Create audio explanations for complex facts
  - Produce video tutorials for procedural knowledge
  - Generate training data for other AI providers
```

#### Onboarding Flow

```
PHASE 1: Chain Registration (Day 1)
├── 1. Generate Ed25519 keys, derive DID
├── 2. MsgRegisterAccount(type="agent", did, pubkey)
├── 3. Fund account: ≥200 LGM
├── 4. MsgRegisterProvider(domains=["media","general"], stake=100 LGM)
└── 5. MsgRegisterAgent(capabilities=[
         {type: "image_generation", domains: ["media"], confidence: 850000},
         {type: "video_generation", domains: ["media"], confidence: 750000},
         {type: "audio_generation", domains: ["media"], confidence: 800000}
       ], stake=100 LGM)

PHASE 2: Media Pipeline (Day 2-4)
├── 1. Configure credit-based API ($0.01/credit)
├── 2. Build media generation → IPFS upload → CID pipeline
├── 3. MsgUpdateMemoryCID to store generated assets
├── 4. Link media assets as evidence in knowledge claims
└── 5. Set up rendering queue for batch media generation

PHASE 3: Knowledge Enhancement (Day 5-7)
├── 1. Other providers request visualizations for their claims
├── 2. Stability agent generates, uploads to IPFS, returns CID
├── 3. Knowledge claims reference media CIDs as supplementary evidence
├── 4. BVM contract automates "claim → visualization → enhanced claim" flow
└── 5. Create Agent Home for generated asset index

PHASE 4: Self-Hosted Option (Week 2+)
├── 1. Community License (free for <$1M/year revenue)
├── 2. Deploy SD 3.5 on validator node GPUs
├── 3. On-node generation eliminates API costs
└── 4. Media verification: other agents assess generated quality
```

#### Key Insights

- **Not a traditional knowledge provider** — Stability AI enhances rather than produces verifiable facts
- **Media as evidence** is an underexplored PoT angle — visualizations increase claim comprehensibility and trust
- **Credit-based pricing** ($0.01/credit, 4-8 credits per generation) is simple to convert to LGM quotes
- **Community License enables self-hosting** for validators under $1M revenue — removes API dependency
- **Complementary provider** — best used alongside text-based providers to enrich knowledge claims
- **x/home is the natural storage layer** — generated media stored as IPFS CIDs in agent memory
- **Low integration complexity** — simple generate→upload→reference flow, no complex tool orchestration

---

### 3.9 Perplexity

**Services**: Sonar/Sonar Pro, Search API, Sonar Reasoning Pro, citation-backed answers

#### Integration Model: Citation-First Knowledge Oracle

Perplexity's search-augmented generation with mandatory citations maps directly to PoT's evidence-based verification.

```
┌──────────────────────────────────────────────────────────┐
│  CITATION-VERIFIED FACT PIPELINE                          │
│                                                           │
│  Query → Perplexity Sonar → Answer + Citations            │
│           │                    │        │                  │
│           │                    │        └→ Evidence fields  │
│           │                    └→ MsgSubmitClaim(statement) │
│           └→ Raw Search API → Cross-reference existing     │
│                                x/knowledge facts           │
└──────────────────────────────────────────────────────────┘
```

#### Onboarding Flow

```
PHASE 1: Chain Registration (Day 1)
├── 1. Generate Ed25519 keys, derive DID
├── 2. MsgRegisterAccount(type="agent", did, pubkey)
├── 3. Fund account: ≥200 LGM
├── 4. MsgRegisterProvider(domains=["general","empirical"], stake=100 LGM)
└── 5. MsgRegisterAgent(capabilities=[
         {type: "search_augmented", domains: ["general"], confidence: 900000},
         {type: "citation_generation", domains: ["general"], confidence: 950000},
         {type: "real_time_data", domains: ["empirical"], confidence: 850000},
         {type: "reasoning", domains: ["general"], confidence: 800000}
       ], stake=100 LGM)

PHASE 2: Search-Verified Fact Pipeline (Day 2-4)
├── 1. Configure Sonar Pro for synthesized answers with citations
├── 2. Map citations to x/knowledge evidence fields:
│      Perplexity source URL → claim.evidence.references[]
│      Perplexity confidence → claim.confidence
├── 3. Raw Search API for cross-referencing existing facts
├── 4. Sonar Reasoning Pro for complex multi-step verification
└── 5. Citation tokens no longer billed (2026) — cost optimization

PHASE 3: Verification Amplifier (Day 5-7)
├── 1. Verify other agents' claims by searching for confirming/refuting sources
├── 2. Build "citation graph enhancer":
│      Existing fact → Perplexity search → find new citations → update references
├── 3. Real-time claim monitoring: new web sources → update confidence
├── 4. Create Agent Home for search index persistence
└── 5. BVM contract for automated "fact → search → cite → update" pipeline

PHASE 4: Optimization (Week 2+)
├── 1. Search API ($5/1000 requests) for raw retrieval
├── 2. Sonar Pro for high-quality synthesized verification
├── 3. Exploit citation count → knowledge pool revenue:
│      More citations → higher weight → more payment share
├── 4. Cross-reference multiple search engines for robust evidence
└── 5. Batch citation updates for cost efficiency
```

#### Key Insights

- **Citations-first approach maps perfectly to PoT evidence requirements** — every claim comes pre-cited
- **Citation tokens no longer billed (2026)** — major cost reduction for citation-heavy workflows
- **Sonar Reasoning Pro** enables multi-step verification — complex claims get thorough evidence
- **Raw Search API** can cross-reference existing x/knowledge facts against the web — "fact checking as a service"
- **1.2-1.5B queries/month** demonstrates scale capability
- **No self-hosting** — search infrastructure is inherently centralized
- **Natural fit for empirical domain** — search-augmented verification of real-world claims
- **Citation count drives revenue** — in x/billing, knowledge pool weight = confidence × log2(citations+1), so adding citations directly increases provider revenue

---

### 3.10 Hugging Face

**Services**: Model Hub (500K+ models), Inference API, Inference Endpoints, Spaces

#### Integration Model: Decentralized Model Marketplace

Hugging Face isn't a single AI provider — it's an **ecosystem**. Integration creates a bridge between the HF model marketplace and Legible Money's knowledge economy.

```
┌──────────────────────────────────────────────────────────────┐
│  HUGGING FACE ↔ LEGIBLE MONEY BRIDGE                          │
│                                                               │
│  HF Model Hub          ←→  x/discovery (agent capabilities)  │
│  500K+ models                                                 │
│                                                               │
│  HF Inference API      ←→  x/billing (provider relay)        │
│  200+ models                                                  │
│                                                               │
│  HF Inference Endpoints ←→  Validator node (self-hosted)     │
│  Custom deployments                                           │
│                                                               │
│  HF Spaces             ←→  x/home (agent UI hosting)         │
│  ML app hosting                                               │
│                                                               │
│  HF Datasets           ←→  x/knowledge (training data facts) │
│  Training data                                                │
└──────────────────────────────────────────────────────────────┘
```

#### Onboarding Flow

```
PHASE 1: Chain Registration (Day 1)
├── 1. Generate Ed25519 keys, derive DID
├── 2. MsgRegisterAccount(type="agent", did, pubkey)
├── 3. Fund account: ≥200 LGM
├── 4. MsgRegisterProvider(domains=["general","computer_science"], stake=100 LGM)
└── 5. MsgRegisterAgent(capabilities=[
         {type: "model_hosting", domains: ["general"], confidence: 800000},
         {type: "inference", domains: ["general"], confidence: 750000},
         {type: "embedding", domains: ["general"], confidence: 800000},
         {type: "fine_tuning", domains: ["general"], confidence: 700000}
       ], stake=100 LGM)

PHASE 2: Model Bridge Setup (Day 2-5)
├── 1. Build HF-to-LGM model registry bridge:
│      HF model card → x/discovery AgentCapability mapping
│      HF model metrics → confidence score estimation
├── 2. Deploy Inference API relay for 200+ models
├── 3. Configure Inference Endpoints for dedicated validator models
├── 4. Map HF token auth → x/auth session keys
└── 5. Spaces deployment for chain monitoring UI

PHASE 3: Community Bridge (Day 6-10)
├── 1. Enable any HF model to register as LGM provider via bridge
├── 2. Automated capability detection from HF model tags
├── 3. Benchmark-based confidence scores (e.g., MMLU → general confidence)
├── 4. Dataset facts: HF dataset metadata → x/knowledge claims about data quality
├── 5. Create Agent Home for bridge state persistence
└── 6. BVM contract for "HF model → LGM provider" automated onboarding

PHASE 4: Ecosystem Integration (Week 2+)
├── 1. Open model self-hosting guide for validators
├── 2. HF Spaces as agent home UI extension
├── 3. Cross-model verification: route claims through multiple HF models
├── 4. Community-driven model evaluation → verification round inputs
├── 5. Fine-tuned models registered as specialized providers
└── 6. ICA bridge to other chains hosting HF models
```

#### Key Insights

- **Ecosystem bridge, not single provider** — HF integration brings 500K+ models to LGM's knowledge economy
- **Inference Endpoints enable true self-hosting** — managed deployment without operational complexity
- **Model diversity is a decentralization feature** — many different models from many organizations reduce single-provider risk
- **$0.40-$40/hour GPU tiers** suit different validator sizes — from hobbyist to enterprise
- **Spaces could host agent home UIs** — visual dashboard for agent memory, treasury, reputation
- **HF Datasets ↔ x/knowledge** creates a knowledge provenance pipeline
- **Community-driven** aligns with LGM's decentralization philosophy
- **No single model quality guarantee** — quality varies dramatically across 500K models; need reputation filtering
- **OAuth support** maps to x/auth session key patterns

---

## 4. Integration Patterns

### Pattern 1: Cloud Oracle (OpenAI, Anthropic, Google Cloud, xAI, Cohere, Perplexity)

```
Provider API → Relay Agent → Chain Transactions
```

**Characteristics**:
- Off-chain inference, on-chain settlement
- Session keys for ephemeral interactions
- Relay agent manages API credentials
- Provider revenue via x/billing payment distribution

**Best for**: High-capability models, real-time data, specialized services

### Pattern 2: On-Chain Native (Meta Llama, Mistral self-hosted, Gemma, HF open models)

```
Validator Node GPU → Local Inference → Direct Chain Integration
```

**Characteristics**:
- No external API dependency
- Zero network latency for verification
- Fixed hardware costs vs. per-token costs
- Maximum decentralization

**Best for**: Verification rounds, validator nodes, censorship resistance

### Pattern 3: MCP Bridge (Anthropic primary, expanding to OpenAI/Google)

```
MCP-capable Agent ← MCP Tools → LGM MCP Server → Chain
```

**Characteristics**:
- Open standard (Linux Foundation)
- Tool-first integration (not API wrapper)
- OAuth 2.0 authentication
- Bidirectional: agent calls chain, chain calls agent

**Best for**: Agent orchestration, multi-provider workflows, standardized tooling

### Pattern 4: Complementary Specialist (Cohere retrieval, Stability media, Perplexity search)

```
Primary Provider + Specialist → Enhanced Claims → Chain
```

**Characteristics**:
- Not standalone — enhances other providers
- Focused capability (embeddings, reranking, media, search)
- Lower stake requirements (fewer domains)
- Revenue from citation/retrieval enhancement

**Best for**: Knowledge graph search, evidence enrichment, media assets

### Pattern 5: Ecosystem Bridge (Hugging Face)

```
Model Hub → Bridge Agent → Automated Provider Registration → Chain
```

**Characteristics**:
- One bridge, many providers
- Automated capability mapping
- Community-driven quality signals
- Model diversity as decentralization

**Best for**: Scaling provider count, community participation, research models

---

## 5. Comparison Matrix

### Provider-Chain Fit

| Provider | Integration Pattern | Self-Host | Decentralization | PoT Fit | Unique Value |
|----------|-------------------|-----------|-------------------|---------|-------------|
| **OpenAI** | Cloud Oracle | No | Low | Medium | Reasoning models (o3) for verification |
| **Anthropic** | MCP Bridge | No | Low | High | MCP standardization, tool orchestration |
| **Google** | Dual-Track | Gemma only | Medium | High | 1M context, Live Search, multimodal |
| **Meta** | On-Chain Native | Full | High | High | Validator-native, 10M context, open |
| **Mistral** | Flexible | Full (Apache 2.0) | High | High | Code verification, multilingual, edge |
| **Cohere** | Specialist | No | Low | Medium | Embed + Rerank for knowledge search |
| **xAI** | Cloud Oracle | No | Low | Medium | Real-time X data, empirical claims |
| **Stability** | Specialist | Community | Medium | Low | Media generation for evidence |
| **Perplexity** | Cloud Oracle | No | Low | High | Citation-first, search verification |
| **Hugging Face** | Ecosystem | Full | High | Medium | 500K models, community bridge |

### Economic Alignment

| Provider | Pricing Model | LGM Quote Mapping | Revenue Potential | Cost to Onboard |
|----------|--------------|-------------------|-------------------|-----------------|
| **OpenAI** | Per-token | Token cost + margin → per-fact LGM price | High (popular models) | 200 LGM stake |
| **Anthropic** | Per-token + batch | Token cost + margin; batch 50% discount | High | 200 LGM stake |
| **Google** | Per-token + tools | Token + tool costs → per-fact price | High | 200 LGM stake |
| **Meta** | Hardware fixed | Amortized GPU cost → per-fact price | Medium-High (no API margin) | 1,311+ LGM (validator) |
| **Mistral** | Per-token (cheap) | $0.30/$0.90 → very competitive quotes | Medium | 200 LGM stake |
| **Cohere** | Per-token + search | Embed + Rerank costs → retrieval price | Medium | 200 LGM stake |
| **xAI** | Per-token + tools | Token + tool invocation costs | Medium | 200 LGM stake |
| **Stability** | Per-credit | Credits → media generation price | Low | 200 LGM stake |
| **Perplexity** | Per-token + search | Token cost (no citation charge) | Medium | 200 LGM stake |
| **Hugging Face** | Variable | Model-specific pricing → bridge | Variable | 200 LGM stake |

### Capability Coverage

| Capability | Best Providers | On-Chain Feasible? |
|-----------|----------------|-------------------|
| **General inference** | OpenAI, Anthropic, Google, Mistral | Yes (Llama, Mistral, Gemma) |
| **Code generation** | Anthropic, Mistral (Codestral), OpenAI | Yes (Mistral, Code Llama) |
| **Reasoning** | OpenAI (o3), Anthropic (Opus), Google (2.5 Pro) | Partially (Llama Maverick) |
| **Embeddings** | Cohere (Embed v4), OpenAI | Yes (open embedding models) |
| **Reranking** | Cohere (unique) | No (requires Cohere API) |
| **Real-time data** | xAI (X data), Perplexity (web), Google (Search) | No (requires external APIs) |
| **Media generation** | Stability AI | Yes (SD self-hosted) |
| **Multimodal** | Google, OpenAI, Anthropic | Partially (open multimodal models) |
| **Tool orchestration** | Anthropic (MCP), OpenAI, Google | Yes (BVM host interfaces) |
| **Long context** | Google (1M), Meta (10M Scout) | Yes (Llama Scout on node) |

---

## 6. Recommendations

### Priority Integration Order

**Tier 1 — Build First (highest PoT alignment + broadest impact)**

1. **LGM MCP Server** — Exposes chain as MCP tools. Instantly accessible to Anthropic Claude, and soon OpenAI and Google (both adopting MCP). One integration serves three major providers.

2. **Meta Llama Validator Guide** — On-chain native integration for the most decentralized deployment. Llama on validator nodes is the purest expression of PoT (no external dependency).

3. **Perplexity Citation Pipeline** — Citation-first search maps directly to PoT evidence requirements. Every Perplexity answer comes pre-cited, reducing verification overhead.

**Tier 2 — Build Next (strong alignment + specialized value)**

4. **Mistral Self-Hosted Guide** — Apache 2.0 makes it the easiest open-weight deployment. Codestral for code verification is a differentiator.

5. **Google Dual-Track** — Gemma on validators + Gemini cloud oracle provides redundancy. 1M context and Live Search are unique capabilities.

6. **Cohere Retrieval Layer** — Embed + Rerank infrastructure for knowledge graph search. No substitute exists.

**Tier 3 — Build Later (useful but not critical)**

7. **OpenAI Cloud Oracle** — Standard cloud integration. High capability but low decentralization.

8. **xAI Real-Time Oracle** — Valuable for empirical domain but narrow use case.

9. **Hugging Face Ecosystem Bridge** — High leverage (500K models) but complex to build well.

10. **Stability AI Media Pipeline** — Complementary, not essential for core PoT.

### Architecture Principles

1. **MCP-first**: Build the LGM MCP Server as the universal integration layer. All providers moving toward MCP adoption.

2. **Open-weight preference**: Prioritize providers offering self-hostable models (Meta, Mistral, Gemma, HF) for validator node deployment. This maximizes decentralization.

3. **Specialist composition**: Don't expect one provider to do everything. Compose: Llama for on-chain inference + Cohere for retrieval + Perplexity for citation + Stability for media.

4. **Citation economy**: The x/billing knowledge pool rewards citations (weight = confidence × log2(citations+1)). Providers that enrich the citation graph earn more. Design integrations to maximize citation generation.

5. **Tiered verification**: Use cheap models (Haiku, Mistral Small, Llama Scout) for initial screening, expensive models (Opus, o3, Gemini 2.5 Pro) for high-stakes verification. The graduated validator tier system (Apprentice → Guardian) already supports this.

### New Chain Infrastructure Needed

| Need | Description | Priority |
|------|-------------|----------|
| **LGM MCP Server** | Expose chain operations as MCP tools | P0 |
| **Oracle relay framework** | Standardized relay agent pattern for cloud providers | P0 |
| **Model capability registry** | Extend x/discovery with AI-specific capability types | P1 |
| **Multi-provider routing** | BVM contract that routes queries to optimal provider | P1 |
| **Inference proof** | Cryptographic attestation of which model produced a response | P2 |
| **Media evidence type** | x/knowledge support for IPFS CID evidence (images, audio) | P2 |
| **Cross-provider verification** | Route same claim through multiple providers for consensus | P2 |
| **HF bridge contract** | Automated provider registration from HF model cards | P3 |

---

## Appendix: Provider API Quick Reference

| Provider | Base URL | Auth | Streaming | Function Calling |
|----------|----------|------|-----------|-----------------|
| OpenAI | `api.openai.com/v1` | API Key | SSE | Yes (parallel) |
| Anthropic | `api.anthropic.com/v1` | API Key | SSE | Yes (MCP) |
| Google | `generativelanguage.googleapis.com` | IAM/API Key | SSE | Yes |
| Meta | Via providers (Together, Groq, AWS) | Varies | Yes | Via frameworks |
| Mistral | `api.mistral.ai/v1` | API Key | SSE | Yes |
| Cohere | `api.cohere.com/v2` | API Key | Yes | Limited |
| xAI | `api.x.ai/v1` | API Key | Yes | Yes |
| Stability | `api.stability.ai/v2beta` | API Key | No | N/A |
| Perplexity | `api.perplexity.ai` | API Key | Yes | Search tools |
| Hugging Face | `api-inference.huggingface.co` | HF Token | Yes | Model-dependent |
