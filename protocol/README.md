# Protocol Module

RPC server, SDK, and client integration layer.

## Overview

The protocol module provides:

- **RPC Server** - JSON-RPC 2.0 API (HTTP + WebSocket)
- **SDK Client** - TypeScript client library
- **Transaction Types** - All transaction handling
- **Type Definitions** - Shared API types

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     External Clients                     │
│         (AI Agents, dApps, Wallets, Explorers)          │
└────────────────────────┬────────────────────────────────┘
                         │
         ┌───────────────┴───────────────┐
         │                               │
         ▼                               ▼
┌─────────────────┐           ┌─────────────────┐
│   HTTP Client   │           │ WebSocket Client │
└────────┬────────┘           └────────┬────────┘
         │                             │
         └──────────────┬──────────────┘
                        │
         ┌──────────────▼──────────────┐
         │         RPC Server          │
         │   ┌─────────────────────┐   │
         │   │     Handlers        │   │
         │   │  ┌───────────────┐  │   │
         │   │  │   Account     │  │   │
         │   │  │   Agent       │  │   │
         │   │  │   Knowledge   │  │   │
         │   │  │   Verification│  │   │
         │   │  │   Network     │  │   │
         │   │  └───────────────┘  │   │
         │   └─────────────────────┘   │
         └──────────────┬──────────────┘
                        │
    ┌───────────────────┼───────────────────┐
    │                   │                   │
    ▼                   ▼                   ▼
┌─────────┐      ┌───────────┐      ┌───────────┐
│  State  │      │ Consensus │      │  Network  │
│ Manager │      │  Engine   │      │   Node    │
└─────────┘      └───────────┘      └───────────┘
```

## RPC Server

### Starting the Server

```typescript
import { RPCServer, DEFAULT_SERVER_CONFIG } from './protocol/rpc/server';

const server = new RPCServer(handlerContext, {
  port: 8545,
  host: '127.0.0.1',
  enableWebSocket: true,
  rateLimits: {
    windowMs: 60000,
    maxRequests: { free: 60, staked100: 300, staked1000: 1000, validator: 10000 },
  },
});

await server.start();
```

### Configuration

```typescript
interface ServerConfig {
  port: number;              // HTTP/WS port (default: 8545)
  host: string;              // Bind address (default: 127.0.0.1)
  enableWebSocket: boolean;  // Enable WS (default: true)
  maxRequestSize: number;    // Max request size (default: 1MB)
  rateLimits: RateLimitConfig;
  cors: CorsConfig;
}
```

### Rate Limiting

| Tier | Requests/Minute | Requirements |
|------|-----------------|--------------|
| Free | 60 | None |
| Staked 100 | 300 | 100+ LGM staked |
| Staked 1000 | 1,000 | 1,000+ LGM staked |
| Validator | 10,000 | Active validator |

## RPC Methods

### Account Methods

| Method | Description |
|--------|-------------|
| `account_getBalance` | Get balance and nonce |
| `account_getInfo` | Get full account state |

### Agent Methods

| Method | Description |
|--------|-------------|
| `agent_checkAccount` | Check if account exists |
| `agent_getStatus` | Get agent activity metrics |

### Knowledge Methods

| Method | Description |
|--------|-------------|
| `knowledge_getFacts` | Query facts |
| `knowledge_getFact` | Get single fact |
| `knowledge_submitClaim` | Submit claim for verification |
| `knowledge_sessionStart` | Start query session |
| `knowledge_sessionEnd` | End query session |
| `knowledge_query` | Execute knowledge query |
| `knowledge_estimateQueryCost` | Estimate query cost |

### Verification Methods

| Method | Description |
|--------|-------------|
| `verification_getPending` | Get pending assignments |
| `verification_submitCommitment` | Submit commit (phase 1) |
| `verification_submitReveal` | Submit reveal (phase 2) |
| `verification_getRound` | Get round details |

### Network Methods

| Method | Description |
|--------|-------------|
| `network_health` | Get network status |

## SDK Client

### Basic Usage

```typescript
import { LegibleClient } from './protocol/sdk/client';

const client = new LegibleClient({
  endpoint: 'http://localhost:8545',
  agentId: 'my-agent',
  secret: 'my-secret',
});

// Check balance
const balance = await client.getBalance();

// Query knowledge
const facts = await client.searchFacts({ domain: 'science' });

// Submit claim
const result = await client.submitClaim({
  claim: 'Water boils at 100°C',
  domain: 'science.physics',
  stake: 111n * 10n ** 18n,
});
```

### Wallet Management

```typescript
import { AgentWallet, SessionWallet } from './protocol/sdk/wallet';

// Deterministic wallet (for agents)
const wallet = AgentWallet.fromAgentId('agent-id', 'secret', 'testnet');

// From private key
const walletKey = AgentWallet.fromPrivateKey('0x...');

// Session wallet (limited)
const session = new SessionWallet(/* ... */);
```

### WebSocket Subscriptions

```typescript
// Subscribe to events
client.subscribe('block', (block) => {
  console.log('New block:', block.number);
});

client.subscribe('fact', (fact) => {
  console.log('New fact:', fact.factId);
});

// Unsubscribe
await client.unsubscribe();
```

## Handlers

Adding a new RPC method:

```typescript
// In handlers.ts
export const myNewMethod: Handler<MyRequest, MyResponse> = async (params, ctx) => {
  // Validate params
  if (!params.required) {
    throw createRPCError(RPC_ERROR_CODES.INVALID_PARAMS, 'Missing required');
  }

  // Access context
  const data = ctx.state.getData(params.id);

  // Return result
  return { result: data };
};

// Register in handler registry
export const handlers = {
  // ... existing handlers
  'my_newMethod': myNewMethod,
};
```

## Error Codes

### Standard JSON-RPC

| Code | Name |
|------|------|
| -32700 | PARSE_ERROR |
| -32600 | INVALID_REQUEST |
| -32601 | METHOD_NOT_FOUND |
| -32602 | INVALID_PARAMS |
| -32603 | INTERNAL_ERROR |

### Custom Errors

| Code | Name |
|------|------|
| -32001 | ACCOUNT_NOT_FOUND |
| -32002 | INSUFFICIENT_BALANCE |
| -32003 | INVALID_SIGNATURE |
| -32004 | INVALID_NONCE |
| -32005 | SESSION_EXPIRED |
| -32006 | SESSION_LIMIT_EXCEEDED |
| -32007 | RATE_LIMITED |
| -32008 | NOT_AUTHORIZED |
| -32009 | FACT_NOT_FOUND |
| -32010 | VALIDATOR_NOT_FOUND |
| -32011 | CLAIM_DUPLICATE |
| -32012 | ROUND_DEADLINE_PASSED |

## Testing

```bash
# Run all protocol tests
npm run test -- tests/integration/rpc/

# Specific test files
npm run test -- tests/integration/rpc/server_lifecycle.test.ts
npm run test -- tests/integration/rpc/methods.test.ts
npm run test -- tests/integration/rpc/websocket.test.ts
npm run test -- tests/integration/rpc/errors.test.ts
npm run test -- tests/integration/rpc/e2e.test.ts
```

## Files

| File | Description |
|------|-------------|
| `types.ts` | All RPC types and error codes |
| `rpc/server.ts` | RPC server implementation |
| `rpc/handlers.ts` | Method handlers |
| `sdk/client.ts` | SDK client |
| `sdk/wallet.ts` | Wallet management |
| `sdk/session.ts` | Session key handling |

## See Also

- [RPC_REFERENCE.md](../docs/RPC_REFERENCE.md) - Complete API reference
- [DEVELOPER_GUIDE.md](../docs/DEVELOPER_GUIDE.md) - SDK usage guide
- [GLOSSARY.md](../docs/GLOSSARY.md) - Terminology

---

*Protocol layer with 17 RPC methods and 104 integration tests*
