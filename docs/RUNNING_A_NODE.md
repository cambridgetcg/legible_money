# Running a Node

Complete guide to operating Legible Money nodes.

## Overview

Legible Money supports three node types:

| Type | Description | Validates | Produces Blocks | Requirements |
|------|-------------|-----------|-----------------|--------------|
| **Full Node** | Syncs chain, serves RPC | No | No | 4GB RAM, 100GB disk |
| **Validator Node** | Participates in consensus | Yes | Yes | 8GB RAM, 200GB disk, 1111 LGM stake |
| **Archive Node** | Stores full history | No | No | 16GB RAM, 1TB+ disk |

## Prerequisites

- **Node.js** >= 20.0.0
- **Docker** (recommended) or native build tools
- **4GB+ RAM** (8GB+ for validators)
- **100GB+ SSD** storage
- **Stable internet** connection

## Quick Start with Docker

### Pull the Image

```bash
docker pull legibleprotocol/lgm-node:latest
```

### Run a Full Node

```bash
docker run -d \
  --name lgm-fullnode \
  -p 8545:8545 \
  -p 8546:8546 \
  -p 26656:26656 \
  -v lgm-data:/data \
  legibleprotocol/lgm-node:latest \
  --mode fullnode \
  --network mainnet
```

### Run a Validator Node

```bash
docker run -d \
  --name lgm-validator \
  -p 8545:8545 \
  -p 8546:8546 \
  -p 26656:26656 \
  -v lgm-data:/data \
  -v ./validator.key:/secrets/validator.key:ro \
  legibleprotocol/lgm-node:latest \
  --mode validator \
  --network mainnet
```

---

## Installation Methods

### Method 1: Docker (Recommended)

```bash
# Clone repository
git clone https://github.com/legible-protocol/legible-money.git
cd legible-money

# Build Docker image
docker build -t lgm-node -f devnet/docker/Dockerfile .

# Run with config
docker run -d \
  --name lgm-node \
  -p 8545:8545 \
  -p 26656:26656 \
  -v lgm-data:/data \
  -v ./config/node.json:/config/node.json:ro \
  lgm-node
```

### Method 2: From Source

```bash
# Clone repository
git clone https://github.com/legible-protocol/legible-money.git
cd legible-money

# Install dependencies
npm install

# Build
npm run build

# Run
npm run node -- --config ./config/node.json
```

### Method 3: Binary (Coming Soon)

```bash
# Download binary
curl -LO https://github.com/legible-protocol/legible-money/releases/latest/download/lgm-node-linux-amd64

# Make executable
chmod +x lgm-node-linux-amd64

# Run
./lgm-node-linux-amd64 --config ./config/node.json
```

---

## Configuration

### Node Configuration File

Create `config/node.json`:

```json
{
  "node": {
    "name": "my-node",
    "dataDir": "/data",
    "logLevel": "info"
  },
  "storage": {
    "type": "leveldb",
    "path": "/data/state"
  },
  "network": {
    "listenAddresses": ["/ip4/0.0.0.0/tcp/26656"],
    "externalAddress": "/ip4/YOUR_PUBLIC_IP/tcp/26656",
    "bootstrapPeers": [
      "/ip4/bootnode1.legible.money/tcp/26656/p2p/PEER_ID",
      "/ip4/bootnode2.legible.money/tcp/26656/p2p/PEER_ID"
    ],
    "maxPeers": 50,
    "minPeers": 3
  },
  "rpc": {
    "enabled": true,
    "host": "127.0.0.1",
    "port": 8545,
    "cors": [],
    "rateLimit": {
      "enabled": true,
      "windowMs": 60000,
      "maxRequests": 1000
    }
  },
  "websocket": {
    "enabled": true,
    "port": 8546
  },
  "validator": {
    "enabled": false,
    "privateKeyPath": "/secrets/validator.key"
  },
  "mempool": {
    "maxSize": 10000,
    "maxPerSender": 100
  },
  "state": {
    "pruning": "default",
    "keepRecent": 1000,
    "snapshotInterval": 100
  },
  "telemetry": {
    "enabled": true,
    "prometheusPort": 9090
  }
}
```

### Configuration Options

#### Node Settings

| Option | Description | Default |
|--------|-------------|---------|
| `node.name` | Node identifier | hostname |
| `node.dataDir` | Data directory | `./data` |
| `node.logLevel` | Log level: debug, info, warn, error | `info` |

#### Network Settings

| Option | Description | Default |
|--------|-------------|---------|
| `network.listenAddresses` | P2P listen addresses | `["/ip4/0.0.0.0/tcp/26656"]` |
| `network.externalAddress` | Advertised address | auto-detect |
| `network.bootstrapPeers` | Bootstrap peer addresses | mainnet bootnodes |
| `network.maxPeers` | Maximum peers | 50 |
| `network.minPeers` | Minimum peers | 3 |

#### RPC Settings

| Option | Description | Default |
|--------|-------------|---------|
| `rpc.enabled` | Enable HTTP RPC | true |
| `rpc.host` | Bind address | `127.0.0.1` |
| `rpc.port` | HTTP port | 8545 |
| `rpc.cors` | CORS origins | `[]` |
| `rpc.rateLimit.maxRequests` | Max requests/minute | 1000 |

#### Validator Settings

| Option | Description | Default |
|--------|-------------|---------|
| `validator.enabled` | Enable validation | false |
| `validator.privateKeyPath` | Path to validator key | - |

#### State Settings

| Option | Description | Default |
|--------|-------------|---------|
| `state.pruning` | Pruning mode: default, nothing, everything | `default` |
| `state.keepRecent` | Blocks to keep (if pruning) | 1000 |
| `state.snapshotInterval` | Snapshot frequency | 100 |

---

## Running a Full Node

### Step 1: Create Configuration

```bash
mkdir -p config data

cat > config/node.json << 'EOF'
{
  "node": { "name": "my-fullnode", "dataDir": "./data" },
  "network": {
    "listenAddresses": ["/ip4/0.0.0.0/tcp/26656"],
    "bootstrapPeers": []
  },
  "rpc": { "enabled": true, "host": "127.0.0.1", "port": 8545 },
  "validator": { "enabled": false }
}
EOF
```

### Step 2: Start the Node

```bash
npm run node -- --config ./config/node.json
```

### Step 3: Verify Sync

```bash
# Check sync status
curl http://localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"network_health","params":{},"id":1}'
```

Expected response when synced:
```json
{
  "result": {
    "status": "healthy",
    "syncStatus": "synced",
    "latestBlock": "12345",
    "peerCount": 5
  }
}
```

---

## Running a Validator

### Step 1: Generate Validator Key

```bash
npm run lgm -- keygen validator my-validator-secret

# Output:
# Validator Key Generated
# Public Key: 87b59695691cba...
# DID: did:lgm:validator-xyz
# Key saved to: ./validator.key
```

### Step 2: Stake Tokens

You need 1,111 LGM minimum to become a validator.

```bash
# Set up client
export LGM_ENDPOINT=http://localhost:8545
export LGM_AGENT_ID=my-validator
export LGM_SECRET=my-validator-secret

# Register as validator
npm run lgm -- validator register 1111
```

### Step 3: Configure Validator

```bash
cat > config/validator.json << 'EOF'
{
  "node": { "name": "my-validator", "dataDir": "./data" },
  "network": {
    "listenAddresses": ["/ip4/0.0.0.0/tcp/26656"],
    "externalAddress": "/ip4/YOUR_PUBLIC_IP/tcp/26656"
  },
  "rpc": { "enabled": true, "host": "127.0.0.1", "port": 8545 },
  "validator": {
    "enabled": true,
    "privateKeyPath": "./validator.key"
  }
}
EOF
```

### Step 4: Start Validator

```bash
npm run node -- --config ./config/validator.json
```

### Step 5: Monitor Performance

```bash
# Check validator status
curl http://localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"validator_getInfo","params":{"publicKey":"YOUR_PUBLIC_KEY"},"id":1}'
```

---

## Monitoring

### Prometheus Metrics

Metrics are exposed on port 9090 by default.

```bash
curl http://localhost:9090/metrics
```

### Available Metrics

| Metric | Description |
|--------|-------------|
| `lgm_block_height` | Current block height |
| `lgm_peer_count` | Connected peers |
| `lgm_mempool_size` | Pending transactions |
| `lgm_blocks_produced_total` | Blocks produced (validators) |
| `lgm_verifications_completed_total` | Verifications completed |
| `lgm_verification_accuracy` | Verification accuracy rate |
| `lgm_knowledge_facts_total` | Total verified facts |

### Grafana Dashboard

Import the Legible Money dashboard from `devnet/monitoring/grafana-dashboard.json`.

```bash
# Start monitoring stack
docker-compose -f devnet/docker-compose.yml --profile monitoring up -d

# Access Grafana at http://localhost:3001
# Default credentials: admin / devnet
```

### Log Monitoring

```bash
# View logs
docker logs -f lgm-node

# Filter warnings and errors
docker logs lgm-node 2>&1 | grep -E "WARN|ERROR"
```

---

## Maintenance

### Backup

```bash
# Stop node first
docker stop lgm-node

# Backup data directory
tar -czvf lgm-backup-$(date +%Y%m%d).tar.gz ./data

# Restart
docker start lgm-node
```

### Restore from Backup

```bash
# Stop node
docker stop lgm-node

# Restore data
rm -rf ./data
tar -xzvf lgm-backup-20260205.tar.gz

# Restart
docker start lgm-node
```

### Upgrade

```bash
# Pull new version
docker pull legibleprotocol/lgm-node:latest

# Stop and remove old container
docker stop lgm-node
docker rm lgm-node

# Start with new image (data persists in volume)
docker run -d \
  --name lgm-node \
  -p 8545:8545 \
  -p 26656:26656 \
  -v lgm-data:/data \
  legibleprotocol/lgm-node:latest
```

### State Pruning

```bash
# Stop node
docker stop lgm-node

# Compact state
npm run lgm -- state prune --keep-recent 1000

# Restart
docker start lgm-node
```

---

## Troubleshooting

### Node Not Syncing

```bash
# Check peer connections
curl http://localhost:8545 \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"network_health","params":{},"id":1}' | jq .result.peerCount

# If 0 peers, check:
# 1. Firewall allows port 26656
# 2. Bootstrap peers are correct
# 3. Network connectivity
```

### RPC Not Responding

```bash
# Check if port is bound
netstat -tlnp | grep 8545

# Check logs for errors
docker logs lgm-node | tail -100
```

### Validator Not Producing Blocks

```bash
# Check validator registration
npm run lgm -- validator status

# Check stake
npm run lgm -- balance

# Verify key matches registered key
npm run lgm -- validator info
```

### Out of Disk Space

```bash
# Check disk usage
du -sh ./data/*

# Enable aggressive pruning
# In config: "state": { "pruning": "everything" }

# Or manually prune
npm run lgm -- state prune --keep-recent 100
```

### Memory Issues

```bash
# Check memory usage
docker stats lgm-node

# Reduce mempool size in config
# "mempool": { "maxSize": 1000 }
```

---

## Network Parameters

### Mainnet

| Parameter | Value |
|-----------|-------|
| Chain ID | `lgm-mainnet-1` |
| Block Time | 2521ms |
| P2P Port | 26656 |
| RPC Port | 8545 |
| Bootstrap Peers | See docs.legible.money |

### Testnet

| Parameter | Value |
|-----------|-------|
| Chain ID | `lgm-testnet-1` |
| Block Time | 2521ms |
| P2P Port | 26656 |
| RPC Port | 8545 |
| Faucet | https://faucet.testnet.legible.money |

### Devnet (Local)

| Parameter | Value |
|-----------|-------|
| Chain ID | `lgm-devnet-1` |
| Block Time | 2521ms |
| Validators | 3 |
| Faucet | http://localhost:8080 |

---

## Security Best Practices

### Firewall Rules

```bash
# Allow P2P from anywhere
ufw allow 26656/tcp

# Allow RPC only from localhost
ufw allow from 127.0.0.1 to any port 8545

# Or use a reverse proxy for public RPC
```

### Validator Key Security

1. **Generate keys offline** if possible
2. **Use encrypted storage** for validator keys
3. **Never share private keys**
4. **Use session keys** for automated operations
5. **Regular key rotation** recommended

### RPC Security

```json
{
  "rpc": {
    "host": "127.0.0.1",  // Bind to localhost
    "cors": [],           // No CORS by default
    "rateLimit": {
      "enabled": true,
      "maxRequests": 1000
    }
  }
}
```

For public RPC, use a reverse proxy (nginx, caddy) with:
- HTTPS termination
- IP-based rate limiting
- Request filtering

---

## See Also

- [QUICKSTART.md](../QUICKSTART.md) - Quick start guide
- [RPC_REFERENCE.md](RPC_REFERENCE.md) - API documentation
- [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md) - SDK usage
- [GLOSSARY.md](GLOSSARY.md) - Terminology

---

*Node operation guide based on network/node.ts and devnet configuration*
