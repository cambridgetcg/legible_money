# Legible Money Devnet

Local development network for testing the Legible Money protocol.

## Quick Start

```bash
# 1. Initialize devnet (generates keys, creates genesis)
npx ts-node scripts/init-devnet.ts

# 2. Start the network
docker-compose up -d

# 3. Check logs
docker-compose logs -f

# 4. Stop the network
docker-compose down
```

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                       DEVNET TOPOLOGY                         │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐        │
│   │ validator-1 │◄──│ validator-2 │──►│ validator-3 │        │
│   │  :8545      │   │  :8547      │   │  :8549      │        │
│   └──────┬──────┘   └──────┬──────┘   └──────┬──────┘        │
│          │                 │                 │                │
│          └─────────────────┼─────────────────┘                │
│                            │                                  │
│                    ┌───────▼───────┐                          │
│                    │   fullnode    │  (optional)              │
│                    │    :8551      │                          │
│                    └───────────────┘                          │
│                                                               │
│   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐        │
│   │   faucet    │   │  explorer   │   │  grafana    │        │
│   │   :8080     │   │   :3000     │   │   :3001     │        │
│   └─────────────┘   └─────────────┘   └─────────────┘        │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

## Services

### Core Validators (always running)

| Service | Port | Description |
|---------|------|-------------|
| validator-1 | 8545 (RPC), 8546 (WS), 26656 (P2P) | Primary validator |
| validator-2 | 8547 (RPC), 8548 (WS), 26657 (P2P) | Validator |
| validator-3 | 8549 (RPC), 8550 (WS), 26658 (P2P) | Validator |
| faucet | 8080 | Testnet token faucet |

### Optional Services (with profiles)

```bash
# Start with fullnode
docker-compose --profile fullnode up -d

# Start with explorer
docker-compose --profile explorer up -d

# Start with monitoring (Prometheus + Grafana)
docker-compose --profile monitoring up -d

# Start everything
docker-compose --profile fullnode --profile explorer --profile monitoring up -d
```

| Service | Port | Profile | Description |
|---------|------|---------|-------------|
| fullnode | 8551 (RPC), 8552 (WS) | fullnode | Non-validator node |
| explorer | 3000 | explorer | Block explorer |
| prometheus | 9093 | monitoring | Metrics collection |
| grafana | 3001 | monitoring | Dashboards |

## Using the CLI

```bash
# Set environment
export LGM_ENDPOINT=http://localhost:8545
export LGM_AGENT_ID=my-agent
export LGM_SECRET=my-secret

# Generate wallet
npx ts-node cmd/cli.ts keygen my-agent my-secret

# Request testnet tokens
npx ts-node cmd/cli.ts faucet

# Check balance
npx ts-node cmd/cli.ts balance

# Check network health
npx ts-node cmd/cli.ts health

# Submit a claim
npx ts-node cmd/cli.ts claim science.physics "Water boils at 100°C at sea level"

# View facts
npx ts-node cmd/cli.ts facts science.physics
```

## Faucet API

```bash
# Request tokens via POST
curl -X POST http://localhost:8080/faucet \
  -H "Content-Type: application/json" \
  -d '{"did": "did:lgm-devnet:your-did"}'

# Request tokens via GET
curl "http://localhost:8080/faucet?did=did:lgm-devnet:your-did"

# Check faucet status
curl http://localhost:8080/health
```

## Explorer API

```bash
# Network overview
curl http://localhost:3000/api/overview

# Recent blocks
curl http://localhost:3000/api/blocks

# Block details
curl http://localhost:3000/api/block/1

# Account info
curl http://localhost:3000/api/account/did:lgm-devnet:faucet

# Facts
curl http://localhost:3000/api/facts

# Validators
curl http://localhost:3000/api/validators
```

## RPC Methods

The validators expose a JSON-RPC 2.0 API:

```bash
# Get network health
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"lgm_networkHealth","params":[],"id":1}'

# Get block number
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"lgm_blockNumber","params":[],"id":1}'

# Get account
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"lgm_getAccount","params":["did:lgm-devnet:faucet"],"id":1}'
```

## Configuration

### Genesis (config/genesis.json)

- Chain ID: `lgm-devnet-1`
- Block time: 2521ms
- Initial validators: 3
- Initial stake: 10,000 LGM each
- Faucet balance: 1 billion LGM

### Consensus Parameters

| Parameter | Value |
|-----------|-------|
| Block time | 2521ms |
| Verification round | 10 blocks |
| Commit phase | 4 blocks |
| Reveal phase | 4 blocks |
| Min validators | 3 |
| Max validators | 21 |
| Validator quorum | 66% |
| Acceptance threshold | 80% |
| Rejection threshold | 80% |

## Monitoring

### Grafana (http://localhost:3001)

- Username: admin
- Password: devnet

### Available Metrics

- `lgm_block_height` - Current block height
- `lgm_peer_count` - Connected peers
- `lgm_mempool_size` - Pending transactions
- `lgm_blocks_produced_total` - Blocks produced
- `lgm_verifications_completed_total` - Verifications completed
- `lgm_verification_accuracy` - Verification accuracy
- `lgm_knowledge_facts_total` - Total facts

## Troubleshooting

### Validators not connecting

```bash
# Check if all containers are running
docker-compose ps

# Check logs for connection issues
docker-compose logs validator-1 | grep -i "peer\|connect"
```

### Reset devnet

```bash
# Stop and remove all data
docker-compose down -v

# Re-initialize
npx ts-node scripts/init-devnet.ts

# Start fresh
docker-compose up -d
```

### Check node sync status

```bash
# Compare block heights
for port in 8545 8547 8549; do
  echo "Port $port:"
  curl -s -X POST http://localhost:$port \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"lgm_blockNumber","params":[],"id":1}' | jq .result
done
```

## Development

### Adding a new validator

1. Generate keys: `npx ts-node scripts/init-devnet.ts` (modify NUM_VALIDATORS)
2. Add service to docker-compose.yml
3. Add to genesis.json
4. Restart network

### Testing consensus

```bash
# Submit multiple claims rapidly
for i in {1..10}; do
  npx ts-node cmd/cli.ts claim test.domain "Test claim $i"
done

# Watch verification in logs
docker-compose logs -f | grep -i verification
```
