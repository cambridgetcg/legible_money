# Legible Money Validator Guide

## Hardware Requirements

### Minimum (testnet)
- **CPU**: 4 cores
- **RAM**: 8 GB
- **Disk**: 100 GB SSD
- **Network**: 100 Mbps

### Recommended (production)
- **CPU**: 8 cores
- **RAM**: 16 GB
- **Disk**: 500 GB NVMe SSD
- **Network**: 1 Gbps

### Software
- Go 1.23+
- jq 1.6+

## Setup from Scratch

### 1. Build the binary

```bash
git clone https://github.com/yournameisai/legible-money.git
cd legible-money
make build
```

Verify:
```bash
./build/legbled version --long
# Should show: name=legible, server_name=legbled, version=v0.1.0-...
```

### 2. Initialize your node

```bash
legbled init <your-moniker> --chain-id lgm-testnet-1
```

This creates `~/.legbled/` with default config.

### 3. Configure CometBFT

Edit `~/.legbled/config/config.toml`:
```toml
# Block time (must match network)
timeout_commit = "2521ms"
timeout_propose = "2000ms"

# Persistent peers (get from network operators)
persistent_peers = "node_id1@ip1:26656,node_id2@ip2:26656"

# Vote extensions (REQUIRED for PoT)
# This is set in genesis, do not override
```

### 4. Configure app

Edit `~/.legbled/config/app.toml`:
```toml
minimum-gas-prices = "0.025ulgm"

# gRPC is disabled due to reflection panics on hand-written types
[grpc]
enable = false
```

### 5. Get genesis file

Replace the default genesis with the canonical testnet genesis:
```bash
cp /path/to/lgm-testnet-1/genesis.json ~/.legbled/config/genesis.json
```

### 6. Start your node

```bash
legbled start --minimum-gas-prices "0.025ulgm"
```

Wait for sync to complete. Check status:
```bash
legbled status | jq '.SyncInfo.catching_up'
# Should be "false" when synced
```

### 7. Register as validator

Once synced, create your validator:
```bash
# First, fund your account (get tokens from faucet or genesis allocation)

# Register with Legible staking (4-tier system)
legbled tx legible_staking register-validator \
  --moniker "<your-moniker>" \
  --stake 111000ulgm \
  --from <your-key> \
  --chain-id lgm-testnet-1 \
  --gas auto --gas-adjustment 1.5 \
  -y
```

Note: You start as Tier 0 (Apprentice). Advance to higher tiers through stake and performance.

### 8. Register your account (DID)

```bash
legbled tx legible_auth register-account \
  <your-did> <your-pubkey-hex> "validator" \
  --from <your-key> \
  --chain-id lgm-testnet-1 \
  --gas auto --gas-adjustment 1.5 \
  -y
```

## Monitoring

### Key metrics to watch

```bash
# Node status
legbled status | jq '.SyncInfo'

# Validator info
legbled query legible_staking validator <your-address>

# Emergency status (should be "normal")
legbled query emergency status

# Active verification rounds
legbled query knowledge active-rounds
```

### Log level

For debugging, increase log level:
```bash
legbled start --log_level "info"  # default
legbled start --log_level "debug" # verbose
```

### Disk usage

Monitor disk growth. Default pruning keeps recent state:
```bash
du -sh ~/.legbled/data/
```

## Troubleshooting

### Node won't start

1. **Port conflict**: Check if port 26656 is in use
   ```bash
   lsof -i :26656
   ```

2. **Wrong genesis**: Verify genesis hash matches network
   ```bash
   sha256sum ~/.legbled/config/genesis.json
   ```

3. **Missing vote extensions**: `vote_extensions_enable_height` must be set to "1" in genesis

### Node falls behind

1. Check peer connections:
   ```bash
   legbled status | jq '.NodeInfo.other.peer_count'
   ```

2. Verify system clock is synced (NTP)

3. Check disk I/O (SSD required, HDD will fall behind)

### Validator jailed/slashed

1. Check your validator status:
   ```bash
   legbled query legible_staking validator <your-address>
   ```

2. Common causes:
   - Missed reveals in verification rounds (0.01% slash)
   - Incorrect verification votes (6% slash)
   - Downtime / missed blocks

3. To unjail, increase stake or wait for slash decay period (34,272 blocks)

### Account pubkey not found

The first transaction from a new account bootstraps the on-chain pubkey. Until sent, `legbled query auth account <addr>` shows null pubkey. Send any transaction to register.

## Bootstrap Period

During the first 480,000 blocks (~14 days), these transaction types are gas-free:
- `submit-claim`
- `submit-commitment`
- `submit-reveal`
- `register-validator`
- `register-account`

This bootstraps the knowledge verification network without requiring validators to hold gas tokens.

## Chain Constants

| Property | Value |
|----------|-------|
| Chain ID | lgm-testnet-1 |
| Block time | 2521ms |
| Finality depth | 19 blocks |
| Unbonding period | 268,560 blocks (~7 days) |
| Bootstrap gas-free | 480,000 blocks (~14 days) |
| Bech32 prefix | lgm / lgmvaloper |
| Token | ulgm (micro-LGM, 6 decimals) |
| Blocks per day | ~34,272 |
