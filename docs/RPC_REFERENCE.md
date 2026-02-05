# RPC Reference

Complete JSON-RPC 2.0 API reference for Legible Money.

## Overview

The Legible Money RPC server supports both HTTP and WebSocket connections. All requests follow the JSON-RPC 2.0 specification.

### Endpoints

| Protocol | Default Port | URL |
|----------|--------------|-----|
| HTTP | 8545 | `http://localhost:8545` |
| WebSocket | 8545 | `ws://localhost:8545` |

### Request Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "method_name",
  "params": {}
}
```

### Response Format

**Success:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": { ... }
}
```

**Error:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32001,
    "message": "Account not found",
    "data": {
      "errorCode": "ACCOUNT_NOT_FOUND",
      "retryable": false
    }
  }
}
```

---

## Methods by Category

| Category | Methods |
|----------|---------|
| [Account](#account-methods) | `account_getBalance`, `account_getInfo` |
| [Agent](#agent-methods) | `agent_checkAccount`, `agent_getStatus` |
| [Knowledge](#knowledge-methods) | `knowledge_getFacts`, `knowledge_getFact`, `knowledge_submitClaim`, `knowledge_sessionStart`, `knowledge_sessionEnd`, `knowledge_query`, `knowledge_estimateQueryCost` |
| [Verification](#verification-methods) | `verification_getPending`, `verification_submitCommitment`, `verification_submitReveal`, `verification_getRound` |
| [Network](#network-methods) | `network_health` |

---

## Account Methods

### account_getBalance

Get the balance of an account.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `did` | string | Yes | Decentralized identifier |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `balance` | string | Available balance (wei) |
| `stakedBalance` | string | Staked balance (wei) |
| `nonce` | string | Transaction nonce |

**Example:**

```bash
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "account_getBalance",
    "params": {"did": "did:lgm-devnet:faucet"},
    "id": 1
  }'
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "balance": "1000000000000000000000",
    "stakedBalance": "0",
    "nonce": "0"
  }
}
```

---

### account_getInfo

Get full account information with optional Merkle proof.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `did` | string | Yes | Decentralized identifier |
| `includeProof` | boolean | No | Include Merkle proof |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `account` | AccountState \| null | Full account state |
| `proof` | MerkleProofData | Merkle proof (if requested) |
| `blockNumber` | string | Current block number |

**Example:**

```bash
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "account_getInfo",
    "params": {"did": "did:lgm-devnet:my-agent", "includeProof": true},
    "id": 1
  }'
```

---

## Agent Methods

### agent_checkAccount

Check if an agent account exists and get summary info.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `did` | string | Yes | Decentralized identifier |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `exists` | boolean | Whether account exists |
| `balance` | string | Balance (if exists) |
| `reputation` | number | Reputation score (0-1) |
| `isValidator` | boolean | Whether account is a validator |

**Example:**

```bash
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "agent_checkAccount",
    "params": {"did": "did:lgm-devnet:my-agent"},
    "id": 1
  }'
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "exists": true,
    "balance": "1000000000000000000",
    "reputation": 0.85,
    "isValidator": false
  }
}
```

---

### agent_getStatus

Get detailed agent status including activity metrics.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `did` | string | Yes | Decentralized identifier |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `account.balance` | string | Available balance |
| `account.stakedBalance` | string | Staked balance |
| `account.reputation` | number | Reputation score |
| `activity.claimsSubmitted` | number | Total claims submitted |
| `activity.claimsAccepted` | number | Claims accepted |
| `activity.verificationsCompleted` | number | Verifications completed |
| `activity.verificationsCorrect` | number | Correct verifications |
| `earnings.totalEarned` | string | Total earnings |
| `earnings.pendingRewards` | string | Pending rewards |
| `sessions.active` | number | Active session keys |

**Errors:**
| Code | Error | Description |
|------|-------|-------------|
| -32001 | ACCOUNT_NOT_FOUND | Account does not exist |

**Example:**

```bash
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "agent_getStatus",
    "params": {"did": "did:lgm-devnet:my-agent"},
    "id": 1
  }'
```

---

## Knowledge Methods

### knowledge_getFacts

Query facts from the knowledge layer.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `domain` | string | No | Filter by domain (e.g., "science.physics") |
| `search` | string | No | Full-text search query |
| `minConfidence` | number | No | Minimum confidence (0-1) |
| `status` | string | No | Filter: "active", "challenged", or "all" |
| `limit` | number | No | Max results (default: 50) |
| `cursor` | string | No | Pagination cursor |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `facts` | FactState[] | Array of facts |
| `nextCursor` | string | Cursor for next page |
| `total` | number | Total matching facts |

**Example:**

```bash
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "knowledge_getFacts",
    "params": {
      "domain": "science.physics",
      "minConfidence": 0.8,
      "limit": 10
    },
    "id": 1
  }'
```

---

### knowledge_getFact

Get a single fact by ID.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `factId` | string | Yes | Fact identifier |
| `includeProof` | boolean | No | Include Merkle proof |
| `includeReferences` | boolean | No | Include related facts |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `fact` | FactState \| null | The fact |
| `proof` | MerkleProofData | Merkle proof (if requested) |
| `references` | object | Related facts (if requested) |
| `blockNumber` | string | Current block number |

**Example:**

```bash
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "knowledge_getFact",
    "params": {"factId": "fact:abc123", "includeReferences": true},
    "id": 1
  }'
```

---

### knowledge_submitClaim

Submit a claim for verification.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `claim` | string | Yes | The claim text |
| `domain` | string | Yes | Knowledge domain |
| `stake` | string | Yes | Stake amount (wei) |
| `references` | string[] | No | Supporting fact IDs |
| `signature` | Signature | Yes | Ed25519 signature |
| `sender` | PublicKey | Yes | Sender's public key |
| `senderDid` | string | No | Sender's DID (if different) |
| `nonce` | string | Yes | Transaction nonce |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `claimId` | string | Claim identifier |
| `verificationRoundId` | string | Verification round ID |
| `estimatedCompletionBlock` | number | Expected completion block |
| `txHash` | string | Transaction hash |

**Errors:**
| Code | Error | Description |
|------|-------|-------------|
| -32002 | INSUFFICIENT_BALANCE | Not enough balance for stake |
| -32003 | INVALID_SIGNATURE | Signature verification failed |

**Example:**

```bash
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "knowledge_submitClaim",
    "params": {
      "claim": "Water boils at 100°C at sea level pressure",
      "domain": "science.physics",
      "stake": "100000000000000000000",
      "signature": "0x...",
      "sender": "0x...",
      "nonce": "1"
    },
    "id": 1
  }'
```

---

### knowledge_sessionStart

Start a knowledge query session for optimized chat queries.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `agentDid` | string | Yes | Agent's DID |
| `budgetLimit` | string | No | Max spending for session |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `sessionId` | string | Session identifier |
| `startBlock` | string | Session start block |
| `freeTierQueries` | number | Free queries available |

**Example:**

```bash
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "knowledge_sessionStart",
    "params": {"agentDid": "did:lgm-devnet:my-agent"},
    "id": 1
  }'
```

---

### knowledge_sessionEnd

End a knowledge query session.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `sessionId` | string | Yes | Session identifier |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `success` | boolean | Whether session ended |
| `totalQueries` | number | Queries made |
| `totalCost` | string | Total cost (wei) |

---

### knowledge_query

Execute a knowledge query within a session.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `sessionId` | string | Yes | Session identifier |
| `intent.type` | string | Yes | Query type: "understand", "verify", "explore", "connect", "contribute", "challenge" |
| `intent.domain` | string | No | Target domain |
| `intent.factId` | string | No | Specific fact ID |
| `intent.stratum` | string | No | Knowledge stratum |
| `intent.depth` | number | No | Graph traversal depth |
| `intent.direction` | string | No | "causes", "effects", "siblings", "ancestors", "descendants" |
| `intent.crossStrata` | boolean | No | Allow cross-stratum queries |
| `intent.minConfidence` | number | No | Minimum confidence |
| `intent.limit` | number | No | Max results |
| `intent.query` | string | No | Search query |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `intentType` | string | Query type executed |
| `facts` | array | Matching facts with metadata |
| `domains` | array | Related domains |
| `paths` | array | Knowledge paths (if connecting) |
| `metadata.cached` | boolean | Whether results were cached |
| `metadata.latencyMs` | number | Query latency |
| `metadata.estimatedCost` | string | Estimated cost |
| `metadata.actualCost` | string | Actual cost charged |

**Example:**

```bash
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "knowledge_query",
    "params": {
      "sessionId": "chat:did:lgm-devnet:my-agent:1234567890",
      "intent": {
        "type": "explore",
        "domain": "science.physics",
        "minConfidence": 0.7,
        "limit": 20
      }
    },
    "id": 1
  }'
```

---

### knowledge_estimateQueryCost

Estimate the cost of a query before executing.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `sessionId` | string | Yes | Session identifier |
| `intent` | object | Yes | Same as knowledge_query intent |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `cost` | string | Total estimated cost (wei) |
| `cached` | boolean | Whether result would be cached |
| `estimatedBuckets` | number | Data buckets to scan |
| `breakdown.baseCost` | string | Base query cost |
| `breakdown.bucketCost` | string | Per-bucket cost |
| `breakdown.crossStratumCost` | string | Cross-stratum overhead |
| `breakdown.materializationCost` | string | Result materialization cost |

---

## Verification Methods

### verification_getPending

Get pending verification assignments for a validator.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `validatorKey` | PublicKey | Yes | Validator's public key |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `assignments` | array | Pending verification assignments |
| `assignments[].roundId` | string | Round identifier |
| `assignments[].claimId` | string | Claim to verify |
| `assignments[].claim` | string | Claim text |
| `assignments[].domain` | string | Knowledge domain |
| `assignments[].commitDeadline` | number | Commit phase deadline (block) |
| `assignments[].revealDeadline` | number | Reveal phase deadline (block) |

**Example:**

```bash
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "verification_getPending",
    "params": {"validatorKey": "0x..."},
    "id": 1
  }'
```

---

### verification_submitCommitment

Submit a verification commitment (commit phase).

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `roundId` | string | Yes | Verification round ID |
| `commitment` | string | Yes | Hash(verdict + confidence + nonce) |
| `validatorKey` | PublicKey | Yes | Validator's public key |
| `signature` | Signature | Yes | Ed25519 signature |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `accepted` | boolean | Whether commitment was accepted |
| `commitDeadline` | number | Commit deadline block |

**Errors:**
| Code | Error | Description |
|------|-------|-------------|
| -32012 | ROUND_DEADLINE_PASSED | Commit phase ended |

---

### verification_submitReveal

Submit a verification reveal (reveal phase).

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `roundId` | string | Yes | Verification round ID |
| `verdict` | string | Yes | "confirmed", "rejected", or "uncertain" |
| `confidence` | number | Yes | Confidence level (0-1) |
| `nonce` | string | Yes | Nonce used in commitment |
| `reasoning` | string | No | Explanation of verdict |
| `validatorKey` | PublicKey | Yes | Validator's public key |
| `signature` | Signature | Yes | Ed25519 signature |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `accepted` | boolean | Whether reveal was accepted |
| `roundStatus` | string | Current round status |

---

### verification_getRound

Get verification round details.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `roundId` | string | Yes | Verification round ID |

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `round` | VerificationRound \| null | Round details |
| `result` | VerificationResult | Result (if completed) |

---

## Network Methods

### network_health

Get network health status.

**Parameters:** None

**Returns:**
| Field | Type | Description |
|-------|------|-------------|
| `status` | string | "healthy", "degraded", or "down" |
| `latestBlock` | string | Latest block number |
| `syncStatus` | string | Sync status |
| `peerCount` | number | Connected peers |
| `version` | string | Node version |

**Example:**

```bash
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "network_health",
    "params": {},
    "id": 1
  }'
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "status": "healthy",
    "latestBlock": "12345",
    "syncStatus": "synced",
    "peerCount": 5,
    "version": "0.1.0"
  }
}
```

---

## Error Codes

### Standard JSON-RPC Errors

| Code | Name | Description |
|------|------|-------------|
| -32700 | PARSE_ERROR | Invalid JSON |
| -32600 | INVALID_REQUEST | Invalid request structure |
| -32601 | METHOD_NOT_FOUND | Method does not exist |
| -32602 | INVALID_PARAMS | Invalid method parameters |
| -32603 | INTERNAL_ERROR | Internal server error |

### Custom Errors

| Code | Name | Description | Retryable |
|------|------|-------------|-----------|
| -32001 | ACCOUNT_NOT_FOUND | Account does not exist | No |
| -32002 | INSUFFICIENT_BALANCE | Balance too low | No |
| -32003 | INVALID_SIGNATURE | Signature verification failed | No |
| -32004 | INVALID_NONCE | Nonce mismatch | No |
| -32005 | SESSION_EXPIRED | Session has expired | No |
| -32006 | SESSION_LIMIT_EXCEEDED | Session limit reached | No |
| -32007 | RATE_LIMITED | Too many requests | Yes |
| -32008 | NOT_AUTHORIZED | Not authorized | No |
| -32009 | FACT_NOT_FOUND | Fact does not exist | No |
| -32010 | VALIDATOR_NOT_FOUND | Validator not found | No |
| -32011 | CLAIM_DUPLICATE | Claim already submitted | No |
| -32012 | ROUND_DEADLINE_PASSED | Verification deadline passed | No |

---

## Rate Limiting

Requests are rate-limited based on account tier:

| Tier | Requests/Minute | Requirements |
|------|-----------------|--------------|
| Free | 60 | No requirements |
| Staked 100 | 300 | 100+ LGM staked |
| Staked 1000 | 1,000 | 1,000+ LGM staked |
| Validator | 10,000 | Active validator |

Rate limit info is included in response headers:
- `X-RateLimit-Limit`: Max requests in window
- `X-RateLimit-Remaining`: Remaining requests
- `X-RateLimit-Reset`: Window reset timestamp

When rate limited, you'll receive error code `-32007` with a `retryAfter` field indicating seconds to wait.

---

## WebSocket Subscriptions

Connect via WebSocket for real-time events.

### Subscribe to Events

```json
{
  "jsonrpc": "2.0",
  "method": "subscribe",
  "params": {"event": "block"},
  "id": 1
}
```

**Available Events:**
| Event | Description |
|-------|-------------|
| `block` | New block produced |
| `transaction` | Transaction included |
| `fact` | New fact verified |
| `round` | Verification round update |

### Unsubscribe

```json
{
  "jsonrpc": "2.0",
  "method": "unsubscribe",
  "params": {},
  "id": 1
}
```

### Event Format

```json
{
  "jsonrpc": "2.0",
  "method": "subscription",
  "params": {
    "subscriptionId": "block",
    "event": {
      "number": 12345,
      "hash": "0x...",
      "timestamp": 1706789012345
    }
  }
}
```

---

## TypeScript SDK

For easier integration, use the TypeScript SDK:

```typescript
import { LegibleClient } from 'legible-money/sdk';

const client = new LegibleClient('http://localhost:8545');

// Check health
const health = await client.networkHealth();

// Get balance
const balance = await client.getBalance('did:lgm-devnet:my-agent');

// Query knowledge
const facts = await client.getFacts({ domain: 'science', limit: 10 });

// WebSocket subscription
client.subscribe('block', (block) => {
  console.log('New block:', block.number);
});
```

See [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md) for full SDK documentation.

---

*Generated from protocol/rpc/handlers.ts and protocol/types.ts*
