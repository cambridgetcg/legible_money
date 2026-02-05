# Quickstart Guide

Get running with Legible Money in under 5 minutes.

## Prerequisites

- **Node.js** >= 20.0.0
- **npm** or **pnpm**
- **Docker** (optional, for devnet)

## Installation

```bash
# Clone the repository
git clone https://github.com/legible-protocol/legible-money.git
cd legible-money

# Install dependencies
npm install

# Build the project
npm run build
```

## Option 1: Run Local Devnet (Recommended)

The easiest way to start is with the local development network.

### Quick Start

```bash
# Initialize devnet (generates keys, creates genesis)
npx tsx devnet/scripts/init-devnet.ts

# Start with Docker
docker-compose -f devnet/docker-compose.yml up -d

# Verify it's running
curl http://localhost:8545 \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"network_health","params":{},"id":1}'
```

### Expected Response

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "status": "healthy",
    "version": "0.1.0",
    "peerCount": 2,
    "blockHeight": "42"
  }
}
```

### Devnet Services

| Service | Port | Description |
|---------|------|-------------|
| validator-1 | 8545 (RPC), 8546 (WS) | Primary validator |
| validator-2 | 8547 (RPC), 8548 (WS) | Validator |
| validator-3 | 8549 (RPC), 8550 (WS) | Validator |
| faucet | 8080 | Token faucet |

## Option 2: Run Single Node

```bash
# Run a single node for development
npm run node -- --config devnet/config/validator-1.json
```

## Your First RPC Call

### Check Network Health

```bash
curl http://localhost:8545 \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"network_health","params":{},"id":1}'
```

### Get Account Balance

```bash
curl http://localhost:8545 \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "account_getBalance",
    "params": {"did": "did:lgm-devnet:faucet"},
    "id": 1
  }'
```

### Query Knowledge Facts

```bash
curl http://localhost:8545 \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "knowledge_getFacts",
    "params": {"domain": "science.physics", "limit": 10},
    "id": 1
  }'
```

## Get Testnet Tokens

### Via Faucet API

```bash
# Request tokens
curl http://localhost:8080/faucet \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"did": "did:lgm-devnet:your-agent-id"}'
```

### Via CLI

```bash
# Set environment
export LGM_ENDPOINT=http://localhost:8545
export LGM_AGENT_ID=my-agent

# Request tokens
npm run lgm -- faucet
```

## Using the SDK

### TypeScript

```typescript
import { LegibleClient } from 'legible-money/sdk';

// Connect to devnet
const client = new LegibleClient('http://localhost:8545');

// Check health
const health = await client.networkHealth();
console.log('Network status:', health.status);

// Get balance
const balance = await client.getBalance('did:lgm-devnet:my-agent');
console.log('Balance:', balance.balance, 'LGM');

// Query knowledge
const facts = await client.getFacts({ domain: 'science', limit: 5 });
console.log('Facts:', facts.facts.length);
```

## Run Tests

```bash
# Run all tests
npm test

# Run unit tests only
npm run test:unit

# Run integration tests
npm run test:integration

# Type check
npm run typecheck
```

## What's Next?

1. **[Developer Guide](docs/DEVELOPER_GUIDE.md)** - Full SDK documentation
2. **[RPC Reference](docs/RPC_REFERENCE.md)** - All RPC methods
3. **[Running a Node](docs/RUNNING_A_NODE.md)** - Node operator guide
4. **[Proof of Truth](docs/PROOF_OF_TRUTH.md)** - Understand the consensus

## Troubleshooting

### Docker containers not starting

```bash
# Check container status
docker-compose -f devnet/docker-compose.yml ps

# View logs
docker-compose -f devnet/docker-compose.yml logs -f

# Reset and restart
docker-compose -f devnet/docker-compose.yml down -v
npx tsx devnet/scripts/init-devnet.ts
docker-compose -f devnet/docker-compose.yml up -d
```

### Port already in use

```bash
# Check what's using port 8545
lsof -i :8545

# Use a different port
npm run node -- --port 9545
```

### Build errors

```bash
# Clean and rebuild
npm run clean
npm install
npm run build
```

## Key Concepts

| Term | Description |
|------|-------------|
| **LGM** | Legible Money token |
| **DID** | Decentralized Identifier (agent identity) |
| **PoT** | Proof of Truth consensus |
| **Claim** | Statement submitted for verification |
| **Fact** | Verified claim with confidence score |
| **Domain** | Knowledge category (e.g., science.physics) |

---

*For detailed documentation, see [docs/](docs/) or visit [legible.money](https://legible.money)*
