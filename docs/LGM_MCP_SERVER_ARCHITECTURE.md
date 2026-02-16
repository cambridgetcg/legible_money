# LGM MCP Server — Architecture Design & Implementation Plan

> Architecture for transforming the existing MCP scaffold into a production-grade Model Context Protocol server connected to the Legible Money Cosmos SDK chain.
>
> Status: **Design Complete** | Last updated: 2026-02-13

---

## Table of Contents

1. [Current State Analysis](#1-current-state-analysis)
2. [Architecture Decision](#2-architecture-decision)
3. [Target Architecture](#3-target-architecture)
4. [Tool Inventory](#4-tool-inventory)
5. [Implementation Phases](#5-implementation-phases)
6. [Detailed Component Design](#6-detailed-component-design)
7. [Security Model](#7-security-model)
8. [Testing Strategy](#8-testing-strategy)
9. [Deployment Model](#9-deployment-model)
10. [Open Questions & Risks](#10-open-questions--risks)

---

## 1. Current State Analysis

### What Exists (protocol/mcp/)

```
protocol/mcp/
├── index.ts                    # Module entry — exports all tools + server
├── server/
│   └── index.ts                # 261 lines — MCPServer factory, tool router, handleToolCall()
└── tools/
    ├── discovery.ts            # 684 lines — 3 tools, ALL MOCK handlers
    ├── knowledge.ts            # 604 lines — 4 tools, ALL MOCK handlers
    ├── identity.ts             # 642 lines — 3 tools, PARTIAL real (uses createAgentDID)
    └── development.ts          # 745 lines — 4 tools, ALL MOCK handlers
```

**Total**: ~2,936 lines, 14 tools, 4 categories

### Assessment

| Aspect | Status | Notes |
|--------|--------|-------|
| Tool definitions (schemas) | Good | Well-designed inputSchema with descriptions |
| Handler implementations | Mock | All return hardcoded/random data |
| MCP SDK integration | Missing | No `@modelcontextprotocol/sdk` — custom interfaces only |
| Transport layer | Missing | No stdio, no SSE, no HTTP transport |
| Chain connectivity | Missing | No gRPC/REST connection to Cosmos chain |
| Authentication | Missing | No session key verification, no DID auth |
| Error handling | Basic | Generic try/catch, no typed errors |
| Testing | Missing | Zero test files for MCP tools |

### What Exists Elsewhere

| Component | Location | Relevance |
|-----------|----------|-----------|
| RPC Server | `protocol/rpc/server.ts` (510 lines) | HTTP + WebSocket, could inform MCP transport |
| RPC Handlers | `protocol/rpc/handlers/` (11 files) | Pattern for chain state access |
| AgenticAPI | `core/api/agentic.ts` | Session management, PP metadata |
| Cosmos gRPC | Running chain at localhost:9090 | All module queries available |
| Cosmos REST | Running chain at localhost:1317 | REST gateway for queries |
| legbled CLI | `cmd/legbled/` | Transaction submission |

### Dependencies Not Yet Installed

```
@modelcontextprotocol/sdk     # Official MCP TypeScript SDK
@cosmjs/stargate              # Cosmos SDK client for queries
@cosmjs/proto-signing         # Transaction signing
@cosmjs/tendermint-rpc        # CometBFT RPC client
```

---

## 2. Architecture Decision

### Options Considered

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A: TS MCP → Chain gRPC** | Enhance existing TS server, connect via gRPC | Reuses existing code, MCP SDK native in TS, rich ecosystem | Requires running chain separately, network boundary |
| **B: Go MCP (embedded)** | New Go MCP server in legbled binary | Direct keeper access, no network boundary | Go MCP SDK immature, duplicates TS tool definitions |
| **C: Hybrid** | TS MCP server + Go gRPC gateway | Best of both worlds | Two services to run, operational complexity |

### Decision: **Option A — TypeScript MCP Server → Chain gRPC**

**Rationale**:
1. **Existing code reuse** — 14 tools already defined with good schemas
2. **MCP SDK is TypeScript-native** — `@modelcontextprotocol/sdk` is the official SDK
3. **CosmJS is mature** — `@cosmjs/stargate` provides full Cosmos SDK client
4. **Single language** — TS chain codebase is the primary development environment
5. **MCP standard requires stdio/SSE** — easier in Node.js than Go
6. **AgenticAPI integration** — existing session management in TS

**Trade-off accepted**: Network boundary between MCP server and chain node. Mitigated by running on same machine with localhost gRPC.

---

## 3. Target Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    LGM MCP SERVER (TypeScript)                       │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────┐       │
│  │  TRANSPORT LAYER                                         │       │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────────┐              │       │
│  │  │  stdio   │ │  SSE    │ │  Streamable  │              │       │
│  │  │ (local)  │ │ (remote)│ │  HTTP (2025) │              │       │
│  │  └────┬─────┘ └────┬────┘ └──────┬───────┘              │       │
│  │       └─────────────┼─────────────┘                      │       │
│  │                     ▼                                     │       │
│  │  ┌──────────────────────────────────────┐                │       │
│  │  │  MCP SDK Server (@modelcontextprotocol/sdk)           │       │
│  │  │  - Tool registration                                   │       │
│  │  │  - Request routing                                     │       │
│  │  │  - Response formatting                                 │       │
│  │  │  - Resource serving (prompts, resources)               │       │
│  │  └──────────────────┬───────────────────┘                │       │
│  └─────────────────────┼────────────────────────────────────┘       │
│                        ▼                                             │
│  ┌─────────────────────────────────────────────────────────┐       │
│  │  TOOL LAYER (enhanced from existing)                     │       │
│  │                                                           │       │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐             │       │
│  │  │ Knowledge │ │ Identity  │ │ Discovery │             │       │
│  │  │ 4 tools   │ │ 3 tools   │ │ 3 tools   │             │       │
│  │  └─────┬─────┘ └─────┬─────┘ └─────┬─────┘             │       │
│  │        │              │              │                    │       │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐             │       │
│  │  │ Billing   │ │ Home      │ │ Governance│             │       │
│  │  │ 3 tools   │ │ 3 tools   │ │ 2 tools   │ NEW        │       │
│  │  └─────┬─────┘ └─────┬─────┘ └─────┬─────┘             │       │
│  │        │              │              │                    │       │
│  │  ┌───────────┐ ┌───────────┐                            │       │
│  │  │ Staking   │ │ Dev Tools │                            │       │
│  │  │ 2 tools   │ │ 4 tools   │ EXISTING                  │       │
│  │  └─────┬─────┘ └─────┬─────┘                            │       │
│  └────────┼──────────────┼──────────────────────────────────┘       │
│           └──────┬───────┘                                           │
│                  ▼                                                    │
│  ┌─────────────────────────────────────────────────────────┐       │
│  │  CHAIN CLIENT LAYER                                      │       │
│  │                                                           │       │
│  │  ┌────────────────┐  ┌────────────────┐                 │       │
│  │  │ QueryClient    │  │ TxClient       │                 │       │
│  │  │ (read-only)    │  │ (mutations)    │                 │       │
│  │  │                │  │                │                 │       │
│  │  │ - gRPC queries │  │ - Tx signing   │                 │       │
│  │  │ - Module state │  │ - Broadcast    │                 │       │
│  │  │ - Bank/Auth    │  │ - Fee estimate │                 │       │
│  │  │ - Custom mods  │  │ - Nonce mgmt   │                 │       │
│  │  └───────┬────────┘  └───────┬────────┘                 │       │
│  │          │                    │                           │       │
│  │  ┌────────────────────────────────────┐                 │       │
│  │  │ Auth Layer                          │                 │       │
│  │  │ - Session key management            │                 │       │
│  │  │ - DID resolution                    │                 │       │
│  │  │ - Capability enforcement            │                 │       │
│  │  └───────────────┬────────────────────┘                 │       │
│  └──────────────────┼──────────────────────────────────────┘       │
│                     ▼                                                │
└─────────────────────────────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│  LEGIBLE MONEY CHAIN (Cosmos SDK)                                   │
│  localhost:9090 (gRPC) / localhost:1317 (REST) / localhost:26657 (RPC)│
│                                                                     │
│  x/knowledge  x/billing  x/discovery  x/auth  x/home  x/gov  ...  │
└─────────────────────────────────────────────────────────────────────┘
```

### Key Design Principles

1. **Real data, not mocks** — Every tool handler queries actual chain state via gRPC
2. **MCP SDK native** — Use official `@modelcontextprotocol/sdk` for transport + protocol compliance
3. **Auth via session keys** — Map MCP client identity to x/auth session keys
4. **Read-heavy, write-careful** — Most tools are queries; mutations require explicit signing
5. **Backward compatible** — Existing tool schemas preserved; new tools additive
6. **Incremental** — Can ship query-only tools first, add mutation tools later

---

## 4. Tool Inventory

### Existing Tools (enhance with real handlers)

| Tool | Category | Current Status | Target |
|------|----------|---------------|--------|
| `lgm_discover_protocol` | Discovery | Mock | Real protocol params from chain |
| `lgm_get_architecture` | Discovery | Static | Static OK (architecture is documentation) |
| `lgm_list_open_tasks` | Discovery | Mock | Real pending claims/rounds from x/knowledge |
| `lgm_query_knowledge` | Knowledge | Mock | Real fact queries via x/knowledge gRPC |
| `lgm_submit_claim` | Knowledge | Mock | Real MsgSubmitClaim via TxClient |
| `lgm_verify_claim` | Knowledge | Mock | Real MsgCommitVerification + MsgRevealVerification |
| `lgm_challenge_claim` | Knowledge | Mock | Real MsgChallengeFact via TxClient |
| `lgm_register_agent` | Identity | Partial | Real MsgRegisterAccount via TxClient |
| `lgm_get_agent_profile` | Identity | Mock | Real QueryAccount via x/auth + x/discovery gRPC |
| `lgm_attest_capability` | Identity | Mock | Real capability attestation flow |
| `lgm_get_codebase_map` | Development | Static | Static OK (codebase documentation) |
| `lgm_find_pattern` | Development | Static | Static OK (pattern documentation) |
| `lgm_validate_change` | Development | Mock | Real tsc + vitest integration |
| `lgm_run_tests` | Development | Mock | Real vitest execution |

### New Tools (to add)

| Tool | Category | Purpose |
|------|----------|---------|
| `lgm_register_provider` | Billing | MsgRegisterProvider via TxClient |
| `lgm_request_quote` | Billing | MsgRequestQuote → APIQuote response |
| `lgm_execute_payment` | Billing | MsgExecutePayment → PaymentDistribution |
| `lgm_create_home` | Home | MsgCreateHome → home ID |
| `lgm_update_memory` | Home | MsgUpdateMemoryCID → IPFS CID persistence |
| `lgm_get_home` | Home | Query AgentHome state |
| `lgm_get_balance` | Staking | Query bank balance + staking info |
| `lgm_stake` | Staking | MsgStake via TxClient |
| `lgm_get_lip` | Governance | Query LIP by ID |
| `lgm_vote_lip` | Governance | MsgCastVote via TxClient |

**Total**: 14 existing + 10 new = **24 tools** across 7 categories

---

## 5. Implementation Phases

### Phase 1: Foundation (MCP SDK + Chain Client)

**Goal**: MCP SDK integration + chain connectivity. No new tools — just make the server real.

```
Deliverables:
├── Install @modelcontextprotocol/sdk + @cosmjs/stargate
├── protocol/mcp/transport/
│   ├── stdio.ts          # stdio transport for local Claude/MCP clients
│   └── sse.ts            # SSE transport for remote clients
├── protocol/mcp/chain/
│   ├── client.ts         # ChainClient: gRPC query + tx submission
│   ├── query.ts          # Module-specific query helpers
│   └── config.ts         # Chain connection config (endpoints, chain-id)
├── protocol/mcp/server/
│   └── index.ts          # MODIFIED: Wrap in MCP SDK Server
└── Tests:
    └── tests/unit/mcp/chain_client.test.ts
```

**Key decisions**:
- Use `@cosmjs/stargate` StargateClient for queries
- Use `@cosmjs/proto-signing` for transaction signing
- Chain config defaults to localhost:26657 (CometBFT RPC)
- MCP server starts via `npx lgm-mcp` or `import { startServer }`

**Estimated scope**: ~600 lines new code + ~100 lines modifications

### Phase 2: Query Tools Live (read-only chain access)

**Goal**: All query tools return real chain data. No mutations yet.

```
Deliverables:
├── protocol/mcp/tools/
│   ├── knowledge.ts      # MODIFIED: Real queries via x/knowledge gRPC
│   ├── identity.ts       # MODIFIED: Real queries via x/auth + x/discovery gRPC
│   ├── discovery.ts      # MODIFIED: Real protocol params + pending rounds
│   ├── billing.ts        # NEW: lgm_request_quote (query-only)
│   ├── home.ts           # NEW: lgm_get_home (query-only)
│   ├── staking.ts        # NEW: lgm_get_balance (query-only)
│   └── governance.ts     # NEW: lgm_get_lip (query-only)
└── Tests:
    ├── tests/unit/mcp/knowledge_tools.test.ts
    ├── tests/unit/mcp/identity_tools.test.ts
    └── tests/unit/mcp/query_tools.test.ts
```

**Key implementation pattern**:
```typescript
// Before (mock):
async function handleQueryKnowledge(args, context) {
  const facts = generateMockFacts(query, domain, minConfidence, maxResults);
  return { content: [{ type: "text", text: JSON.stringify(response) }] };
}

// After (real):
async function handleQueryKnowledge(args, context) {
  const client = context.chainClient;
  const facts = await client.queryModule("knowledge", "Facts", {
    domain: args.domain,
    minConfidence: Math.floor((args.minConfidence || 0.6) * 1_000_000),
    limit: args.maxResults || 10,
    offset: 0,
  });
  // Transform chain response → MCP tool response format
  return { content: [{ type: "text", text: JSON.stringify(transformedFacts) }] };
}
```

**Estimated scope**: ~1,200 lines new code + ~800 lines modifications

### Phase 3: Mutation Tools (transaction submission)

**Goal**: Tools that write to chain state. Requires session key auth.

```
Deliverables:
├── protocol/mcp/auth/
│   ├── session.ts        # Session key management for MCP clients
│   ├── signer.ts         # Transaction signing with session keys
│   └── capabilities.ts   # Capability enforcement per tool
├── protocol/mcp/tools/
│   ├── knowledge.ts      # MODIFIED: lgm_submit_claim, lgm_verify_claim, lgm_challenge_claim
│   ├── identity.ts       # MODIFIED: lgm_register_agent (real MsgRegisterAccount)
│   ├── billing.ts        # MODIFIED: lgm_register_provider, lgm_execute_payment
│   ├── home.ts           # MODIFIED: lgm_create_home, lgm_update_memory
│   ├── staking.ts        # MODIFIED: lgm_stake
│   └── governance.ts     # MODIFIED: lgm_vote_lip
└── Tests:
    ├── tests/unit/mcp/auth.test.ts
    ├── tests/unit/mcp/mutation_tools.test.ts
    └── tests/integration/mcp/e2e.test.ts
```

**Auth flow**:
```
1. Client provides Ed25519 private key or session key at MCP connection
2. Server derives DID from public key
3. Server resolves account via x/auth QueryAccountByDID
4. Each mutation tool checks session key capabilities
5. Transaction signed with session key, broadcast via CometBFT RPC
6. Response includes tx hash + status
```

**Capability enforcement**:
```typescript
const TOOL_CAPABILITIES: Record<string, string[]> = {
  lgm_submit_claim:      ["verify", "challenge"],  // Requires verify OR challenge capability
  lgm_verify_claim:      ["verify"],
  lgm_challenge_claim:   ["challenge"],
  lgm_register_provider: ["transfer"],               // Requires transfer for stake
  lgm_execute_payment:   ["transfer"],
  lgm_create_home:       ["transfer"],
  lgm_update_memory:     ["write_memory"],
  lgm_stake:             ["stake"],
  lgm_vote_lip:          ["verify"],                  // Governance uses verify capability
};
```

**Estimated scope**: ~1,500 lines new code + ~600 lines modifications

### Phase 4: Resources + Prompts (MCP advanced features)

**Goal**: Expose chain state as MCP resources and provide system prompts.

```
Deliverables:
├── protocol/mcp/resources/
│   ├── chain_state.ts    # Live chain state as MCP resources
│   ├── knowledge_graph.ts # Knowledge subgraph as resource
│   └── agent_profile.ts  # Agent profile as resource
├── protocol/mcp/prompts/
│   ├── verifier.ts       # System prompt for verification agents
│   ├── researcher.ts     # System prompt for research agents
│   └── developer.ts      # System prompt for development agents
└── Tests:
    └── tests/unit/mcp/resources.test.ts
```

**MCP Resources** (new in MCP 2025-11-05 spec):
```typescript
// Resource: live chain state
server.resource("chain://legible-devnet-1/status", async () => ({
  contents: [{
    uri: "chain://legible-devnet-1/status",
    mimeType: "application/json",
    text: JSON.stringify(await client.getChainStatus()),
  }],
}));

// Resource: knowledge fact
server.resource("fact://{factId}", async ({ factId }) => ({
  contents: [{
    uri: `fact://${factId}`,
    mimeType: "application/json",
    text: JSON.stringify(await client.getFact(factId)),
  }],
}));
```

**MCP Prompts** (role-specific system prompts):
```typescript
server.prompt("lgm_verifier", () => ({
  messages: [{
    role: "user",
    content: {
      type: "text",
      text: `You are a Legible Money knowledge verifier. Your role is to:
1. Evaluate claims submitted to the knowledge base
2. Check evidence quality and relevance
3. Assess confidence based on epistemic category
4. Submit honest verdicts via lgm_verify_claim
...`,
    },
  }],
}));
```

**Estimated scope**: ~800 lines new code

### Phase Summary

| Phase | Scope | New Files | Modified Files | New Lines | Tests |
|-------|-------|-----------|---------------|-----------|-------|
| 1: Foundation | MCP SDK + Chain Client | 5 | 1 | ~600 | ~200 |
| 2: Query Tools | Read-only chain access | 6 | 4 | ~1,200 | ~400 |
| 3: Mutation Tools | Transaction submission | 5 | 6 | ~1,500 | ~500 |
| 4: Resources + Prompts | MCP advanced features | 6 | 1 | ~800 | ~200 |
| **Total** | | **22** | **12** | **~4,100** | **~1,300** |

---

## 6. Detailed Component Design

### 6.1 ChainClient (protocol/mcp/chain/client.ts)

The central bridge between MCP tools and the Cosmos SDK chain.

```typescript
interface ChainClientConfig {
  rpcEndpoint: string;      // default: "http://localhost:26657"
  grpcEndpoint?: string;    // default: "http://localhost:9090"
  restEndpoint?: string;    // default: "http://localhost:1317"
  chainId: string;          // default: "legible-devnet-1"
  gasPrice: string;         // default: "0.025ulgm"
}

class ChainClient {
  private stargateClient: StargateClient;
  private signingClient?: SigningStargateClient;

  // Lifecycle
  static async connect(config: ChainClientConfig): Promise<ChainClient>;
  disconnect(): void;

  // Queries (read-only, no signer needed)
  async getChainStatus(): Promise<ChainStatus>;
  async getBalance(address: string, denom?: string): Promise<Coin>;
  async getAccount(address: string): Promise<Account | null>;

  // Module-specific queries (via REST since hand-written types)
  async queryKnowledgeFact(factId: string): Promise<Fact | null>;
  async queryKnowledgeFacts(params: FactQueryParams): Promise<Fact[]>;
  async queryKnowledgeRound(roundId: string): Promise<Round | null>;
  async queryBillingProvider(address: string): Promise<Provider | null>;
  async queryBillingQuote(quoteId: number): Promise<APIQuote | null>;
  async queryDiscoveryAgent(address: string): Promise<AgentProfile | null>;
  async queryDiscoveryByDomain(domain: string): Promise<AgentProfile[]>;
  async queryAuthAccount(address: string): Promise<LegibleAccount | null>;
  async queryAuthAccountByDID(did: string): Promise<LegibleAccount | null>;
  async queryHome(homeId: string): Promise<AgentHome | null>;
  async queryGovLIP(lipId: number): Promise<LIP | null>;

  // Mutations (require signer)
  async signAndBroadcast(
    signer: SessionSigner,
    messages: EncodeObject[],
    memo?: string,
  ): Promise<DeliverTxResponse>;
}
```

**Implementation note**: Since the Cosmos chain uses hand-written types (json.Marshal, not protobuf), queries go through REST endpoints which return JSON directly. This avoids the proto-registration issues that affected the chain itself.

### 6.2 SessionSigner (protocol/mcp/auth/signer.ts)

Manages transaction signing with session keys.

```typescript
interface SessionSignerConfig {
  privateKey: Uint8Array;        // Ed25519 session private key
  accountAddress?: string;       // bech32 address (derived if not provided)
  did?: string;                  // DID (derived if not provided)
  capabilities?: string[];       // Session key capabilities
}

class SessionSigner {
  readonly address: string;
  readonly did: string;
  readonly publicKey: Uint8Array;

  static fromPrivateKey(key: Uint8Array): SessionSigner;
  static fromMnemonic(mnemonic: string, hdPath?: string): SessionSigner;

  async sign(signDoc: SignDoc): Promise<Uint8Array>;
  hasCapability(capability: string): boolean;
  async getNonce(client: ChainClient): Promise<number>;
}
```

### 6.3 MCP SDK Server Wrapper (protocol/mcp/server/index.ts)

Enhanced server using official MCP SDK.

```typescript
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { SSEServerTransport } from "@modelcontextprotocol/sdk/server/sse.js";

export async function createLGMServer(config: LGMServerConfig): Promise<McpServer> {
  const server = new McpServer({
    name: "legible-money",
    version: "0.2.0",
  });

  // Connect to chain
  const chainClient = await ChainClient.connect(config.chain);

  // Create enhanced context
  const context: ServerContext = {
    config,
    chainClient,
    signer: config.signer ? SessionSigner.fromPrivateKey(config.signer) : undefined,
    sessions: new Map(),
  };

  // Register all tools
  registerKnowledgeTools(server, context);
  registerIdentityTools(server, context);
  registerDiscoveryTools(server, context);
  registerBillingTools(server, context);
  registerHomeTools(server, context);
  registerStakingTools(server, context);
  registerGovernanceTools(server, context);
  registerDevelopmentTools(server, context);

  // Register resources (Phase 4)
  registerChainResources(server, context);

  // Register prompts (Phase 4)
  registerAgentPrompts(server, context);

  return server;
}

// Tool registration pattern (per category):
function registerKnowledgeTools(server: McpServer, context: ServerContext) {
  server.tool(
    "lgm_query_knowledge",
    "Query the verified knowledge base with epistemic metadata",
    {
      query: z.string().describe("Natural language query"),
      domain: z.string().optional().describe("Filter to specific domain"),
      minConfidence: z.number().min(0).max(1).optional().describe("Min confidence threshold"),
      maxResults: z.number().min(1).max(100).optional().describe("Max results"),
    },
    async (args) => {
      const facts = await context.chainClient.queryKnowledgeFacts({
        domain: args.domain,
        minConfidenceBps: Math.floor((args.minConfidence ?? 0.6) * 1_000_000),
        limit: args.maxResults ?? 10,
      });
      return {
        content: [{
          type: "text",
          text: JSON.stringify({
            query: args.query,
            results: facts.map(transformFact),
            count: facts.length,
          }, null, 2),
        }],
      };
    },
  );

  // Mutation tool with auth check:
  server.tool(
    "lgm_submit_claim",
    "Submit a knowledge claim for verification (requires session key with verify capability)",
    { /* schema */ },
    async (args) => {
      if (!context.signer) {
        return { content: [{ type: "text", text: JSON.stringify({
          error: "Authentication required",
          message: "Provide a session key to submit claims. Connect with --key flag.",
        }) }], isError: true };
      }
      if (!context.signer.hasCapability("verify")) {
        return { content: [{ type: "text", text: JSON.stringify({
          error: "Insufficient capabilities",
          message: "Session key requires 'verify' capability to submit claims.",
        }) }], isError: true };
      }

      const msg = buildMsgSubmitClaim(context.signer.address, args);
      const result = await context.chainClient.signAndBroadcast(
        context.signer, [msg], `MCP: claim submission`,
      );
      return { content: [{ type: "text", text: JSON.stringify({
        success: result.code === 0,
        txHash: result.transactionHash,
        height: result.height,
        claimId: extractClaimId(result),
      }, null, 2) }] };
    },
  );
}
```

### 6.4 Custom Module Query Client

Since custom modules use hand-written types (not protobuf), queries go through REST.

```typescript
// protocol/mcp/chain/query.ts

class ModuleQueryClient {
  constructor(private restEndpoint: string) {}

  // Generic REST query
  private async query<T>(path: string): Promise<T> {
    const res = await fetch(`${this.restEndpoint}${path}`);
    if (!res.ok) throw new Error(`Query failed: ${res.status}`);
    return res.json() as T;
  }

  // x/knowledge queries
  async getFact(factId: string): Promise<KnowledgeFact | null> {
    try {
      return await this.query(`/legible/knowledge/fact/${factId}`);
    } catch { return null; }
  }

  async getFacts(params: { domain?: string; limit?: number; offset?: number }): Promise<KnowledgeFact[]> {
    const qs = new URLSearchParams();
    if (params.domain) qs.set("domain", params.domain);
    if (params.limit) qs.set("limit", String(params.limit));
    if (params.offset) qs.set("offset", String(params.offset));
    const result = await this.query<{ facts: KnowledgeFact[] }>(`/legible/knowledge/facts?${qs}`);
    return result.facts;
  }

  // x/billing queries
  async getProvider(address: string): Promise<BillingProvider | null> { ... }
  async getQuote(quoteId: number): Promise<APIQuote | null> { ... }

  // x/discovery queries
  async getAgentProfile(address: string): Promise<AgentProfile | null> { ... }
  async getAgentsByDomain(domain: string): Promise<AgentProfile[]> { ... }
  async getAgentsByCapability(capType: string): Promise<AgentProfile[]> { ... }

  // x/auth queries
  async getLegibleAccount(address: string): Promise<LegibleAccount | null> { ... }
  async getLegibleAccountByDID(did: string): Promise<LegibleAccount | null> { ... }
  async getSessionKeys(address: string): Promise<SessionKey[]> { ... }

  // x/home queries
  async getHome(homeId: string): Promise<AgentHome | null> { ... }
  async getHomesByOwner(owner: string): Promise<AgentHome[]> { ... }

  // x/gov queries
  async getLIP(lipId: number): Promise<LIP | null> { ... }
  async getLIPs(params: { stage?: string; limit?: number }): Promise<LIP[]> { ... }
}
```

**Important note**: The existing Cosmos chain has custom gRPC service descriptors (hand-written, not proto-generated) which means standard gRPC clients may not work. REST queries via the LCD (light client daemon) are more reliable since they go through the same JSON serialization path. If REST gateway bindings aren't registered, we'll need to add them to each module's `module.go` RegisterGRPCGatewayRoutes — or query via CometBFT ABCI queries directly.

### 6.5 REST Gateway Consideration

The devnet launch report noted: "Custom module gRPC registered but no REST gateway bindings." This means Phase 2 may require adding REST gateway routes to Go modules before queries work.

**Fallback approach**: Use CometBFT ABCI query path:
```typescript
// Direct ABCI query (always works, bypasses REST gateway)
async function abciQuery(client: CometClient, path: string, data: Uint8Array): Promise<Uint8Array> {
  const result = await client.abciQuery({ path, data });
  return result.value;
}

// Example: Query fact via ABCI
const factBytes = await abciQuery(cometClient, `/custom/lgm_knowledge/fact/${factId}`, new Uint8Array());
const fact = JSON.parse(new TextDecoder().decode(factBytes));
```

---

## 7. Security Model

### Authentication

```
LEVEL 0: Anonymous (no key)
  - Query tools only
  - Rate limited: 100 queries/minute
  - No mutations allowed

LEVEL 1: Session Key (ephemeral)
  - Query + mutation tools
  - Capability-scoped mutations
  - Auto-expires after MaxSessionDuration (~7 hours)
  - Created via MsgCreateSession with --key flag at server start

LEVEL 2: Operational Key (persistent)
  - Full access including key rotation
  - Used for provider registration, home creation
  - Higher rate limits
```

### Key Management at MCP Connection

```bash
# Anonymous (query-only)
npx lgm-mcp --rpc http://localhost:26657

# With session key (hex private key)
npx lgm-mcp --rpc http://localhost:26657 --key 0xabcdef...

# With keyring (more secure)
npx lgm-mcp --rpc http://localhost:26657 --keyring-backend test --key-name my-agent

# With mnemonic (development only)
LGM_MNEMONIC="word1 word2 ..." npx lgm-mcp --rpc http://localhost:26657
```

### Capability Enforcement Matrix

| Tool | Required Capability | Auth Level |
|------|-------------------|------------|
| `lgm_query_knowledge` | None | 0 (Anonymous) |
| `lgm_discover_protocol` | None | 0 |
| `lgm_get_architecture` | None | 0 |
| `lgm_list_open_tasks` | None | 0 |
| `lgm_get_agent_profile` | None | 0 |
| `lgm_get_balance` | None | 0 |
| `lgm_get_home` | None | 0 |
| `lgm_get_lip` | None | 0 |
| `lgm_request_quote` | None | 0 |
| `lgm_get_codebase_map` | None | 0 |
| `lgm_find_pattern` | None | 0 |
| `lgm_submit_claim` | `verify` | 1 (Session) |
| `lgm_verify_claim` | `verify` | 1 |
| `lgm_challenge_claim` | `challenge` | 1 |
| `lgm_register_agent` | `transfer` | 2 (Operational) |
| `lgm_register_provider` | `transfer` | 2 |
| `lgm_execute_payment` | `transfer` | 1 |
| `lgm_create_home` | `transfer` | 2 |
| `lgm_update_memory` | `write_memory` | 1 |
| `lgm_stake` | `stake` | 1 |
| `lgm_vote_lip` | `verify` | 1 |
| `lgm_validate_change` | None | 0 |
| `lgm_run_tests` | None | 0 |
| `lgm_attest_capability` | `verify` | 1 |

### Rate Limiting

```
Anonymous:  100 queries/min, 0 mutations/min
Session:    500 queries/min, 50 mutations/min
Operational: 1000 queries/min, 100 mutations/min
```

---

## 8. Testing Strategy

### Unit Tests (per tool category)

```
tests/unit/mcp/
├── chain_client.test.ts       # ChainClient with mocked endpoints
├── session_signer.test.ts     # Key derivation, signing, capabilities
├── knowledge_tools.test.ts    # All 4 knowledge tool handlers
├── identity_tools.test.ts     # All 3 identity tool handlers
├── discovery_tools.test.ts    # All 3 discovery tool handlers
├── billing_tools.test.ts      # All 3 billing tool handlers
├── home_tools.test.ts         # All 3 home tool handlers
├── staking_tools.test.ts      # All 2 staking tool handlers
├── governance_tools.test.ts   # All 2 governance tool handlers
└── development_tools.test.ts  # All 4 development tool handlers
```

**Mock strategy**: Use vitest mock to intercept ChainClient methods. Each tool test provides mock chain responses and verifies correct transformation to MCP tool output.

### Integration Tests

```
tests/integration/mcp/
├── server_lifecycle.test.ts   # Server start, tool listing, shutdown
├── query_e2e.test.ts          # Real queries against running devnet
└── mutation_e2e.test.ts       # Real transactions against running devnet
```

**Integration tests require**: Running devnet (`legbled start`) with seeded state.

### MCP Compliance Tests

```
tests/compliance/mcp/
├── protocol_compliance.test.ts # Verify MCP protocol spec compliance
├── transport_stdio.test.ts     # stdio transport round-trip
└── transport_sse.test.ts       # SSE transport round-trip
```

---

## 9. Deployment Model

### Local Development (stdio)

```bash
# Start chain
legbled start

# Start MCP server (stdio transport for Claude Desktop, Claude Code, etc.)
npx lgm-mcp --transport stdio --rpc http://localhost:26657

# Or via package.json script
npm run mcp
```

**Claude Desktop integration** (`claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "legible-money": {
      "command": "npx",
      "args": ["lgm-mcp", "--transport", "stdio", "--rpc", "http://localhost:26657"],
      "env": {
        "LGM_SESSION_KEY": "0xabcdef..."
      }
    }
  }
}
```

### Remote Server (SSE)

```bash
# Start MCP server with SSE transport
npx lgm-mcp --transport sse --port 3001 --rpc http://chain-node:26657

# Client connects via SSE endpoint
# https://mcp.legiblemoney.io/sse
```

### Docker

```dockerfile
FROM node:22-slim
WORKDIR /app
COPY package*.json ./
RUN npm ci --production
COPY protocol/mcp/ ./protocol/mcp/
COPY core/ ./core/
EXPOSE 3001
CMD ["npx", "lgm-mcp", "--transport", "sse", "--port", "3001"]
```

---

## 10. Open Questions & Risks

### Questions

| # | Question | Impact | Proposed Resolution |
|---|----------|--------|-------------------|
| Q1 | REST gateway bindings missing for custom modules — how to query? | Phase 2 blocked | Use ABCI query path as fallback; add REST bindings to Go modules as P1 |
| Q2 | Should MCP server be a separate npm package or part of main repo? | Deployment | Keep in main repo for now; extract when stable |
| Q3 | How to handle long-running operations (verification rounds span blocks)? | UX | Return tx hash immediately; provide `lgm_check_tx` polling tool |
| Q4 | Should we support MCP sampling (letting AI call other AI through chain)? | Scope | Defer to Phase 5 — sampling is MCP spec optional |
| Q5 | Ed25519 vs secp256k1 key handling in CosmJS? | Phase 3 | Chain uses Ed25519; CosmJS defaults to secp256k1. Need custom signer adapter |

### Risks

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|-----------|--------|------------|
| R1 | CosmJS doesn't support hand-written types | Medium | High | Use REST/ABCI queries instead of gRPC; JSON parsing |
| R2 | MCP SDK breaking changes | Low | Medium | Pin SDK version; abstract transport layer |
| R3 | Ed25519 signing incompatibility | Medium | High | Build custom Ed25519Signer implementing CosmJS OfflineSigner |
| R4 | Performance: REST queries add latency | Low | Low | Cache frequently-accessed state; batch queries |
| R5 | Session key expiration during MCP session | Medium | Medium | Auto-detect expiration; prompt for re-authentication |

### Critical Path

```
Phase 1 (Foundation) → Phase 2 (Queries) → Phase 3 (Mutations) → Phase 4 (Resources)
     │                       │
     │                       └─ REQUIRES: REST gateway bindings OR ABCI query support
     │
     └─ REQUIRES: @modelcontextprotocol/sdk + @cosmjs/stargate installation
```

---

## Appendix A: File Manifest

### New Files

```
protocol/mcp/
├── transport/
│   ├── stdio.ts                    # Phase 1
│   └── sse.ts                      # Phase 1
├── chain/
│   ├── client.ts                   # Phase 1
│   ├── query.ts                    # Phase 2
│   └── config.ts                   # Phase 1
├── auth/
│   ├── session.ts                  # Phase 3
│   ├── signer.ts                   # Phase 3
│   └── capabilities.ts            # Phase 3
├── tools/
│   ├── billing.ts                  # Phase 2 (queries) + Phase 3 (mutations)
│   ├── home.ts                     # Phase 2 + Phase 3
│   ├── staking.ts                  # Phase 2 + Phase 3
│   └── governance.ts               # Phase 2 + Phase 3
├── resources/
│   ├── chain_state.ts              # Phase 4
│   ├── knowledge_graph.ts          # Phase 4
│   └── agent_profile.ts           # Phase 4
└── prompts/
    ├── verifier.ts                 # Phase 4
    ├── researcher.ts               # Phase 4
    └── developer.ts                # Phase 4
```

### Modified Files

```
protocol/mcp/
├── server/index.ts                 # Phase 1: MCP SDK wrapper
├── tools/discovery.ts              # Phase 2: Real protocol params
├── tools/knowledge.ts              # Phase 2: Real queries, Phase 3: Real mutations
├── tools/identity.ts               # Phase 2: Real queries, Phase 3: Real registration
└── tools/development.ts            # Phase 2: Real test execution
```

### Go Files (if REST gateway needed)

```
x/knowledge/module.go               # Add RegisterGRPCGatewayRoutes
x/billing/module.go                 # Add RegisterGRPCGatewayRoutes
x/discovery/module.go               # Add RegisterGRPCGatewayRoutes
x/auth/module.go                    # Add RegisterGRPCGatewayRoutes
x/home/module.go                    # Add RegisterGRPCGatewayRoutes
x/gov/module.go                     # Add RegisterGRPCGatewayRoutes
```

---

## Appendix B: MCP Protocol Compliance Checklist

| MCP Feature | Support Plan | Phase |
|------------|-------------|-------|
| Tool listing | Yes — 24 tools | Phase 1 |
| Tool execution | Yes — all tools callable | Phase 1-3 |
| Tool input validation | Yes — Zod schemas | Phase 1 |
| Resource listing | Yes — chain state | Phase 4 |
| Resource reading | Yes — facts, profiles, homes | Phase 4 |
| Resource subscriptions | Future — WebSocket state updates | Phase 5 |
| Prompt listing | Yes — 3 agent roles | Phase 4 |
| Prompt getting | Yes — role-specific system prompts | Phase 4 |
| Sampling | Future — chain-mediated AI calls | Phase 5 |
| Roots | N/A — not applicable | — |
| stdio transport | Yes | Phase 1 |
| SSE transport | Yes | Phase 1 |
| Streamable HTTP | Future — MCP 2025-11-05 spec | Phase 5 |
| Progress notifications | Yes — for long-running txs | Phase 3 |
| Error handling | Yes — typed MCP errors | Phase 1 |
| Logging | Yes — via MCP SDK logging | Phase 1 |
