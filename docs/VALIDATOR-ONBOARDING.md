# Validator Onboarding Guide

## Overview

Legible Money uses a **4-tier graduated validator system** built on top of Cosmos SDK staking. Validators start as Apprentices and advance through Verified, Bonded, and Guardian tiers based on stake, verification performance, and reputation.

This guide covers joining the network as a validator, both at genesis and post-launch.

## Prerequisites

### Hardware

| Spec | Minimum (testnet) | Recommended |
|------|-------------------|-------------|
| CPU | 4 cores | 8 cores |
| RAM | 8 GB | 16 GB |
| Disk | 100 GB SSD | 500 GB NVMe |
| Network | 100 Mbps | 1 Gbps |

### Software

- **Go** 1.23 or later
- **jq** 1.6+ (for genesis patching)
- **git** (to clone the repository)

## Step 1: Build the Binary

```bash
git clone https://github.com/legible-money/legible.git
cd legible
make build
```

This produces `build/legbled` with version metadata injected via ldflags. Verify:

```bash
./build/legbled version
```

## Step 2: Initialize Your Node

```bash
legbled init <your-moniker> --chain-id lgm-testnet-1 --default-denom ulgm
```

This creates the default configuration at `~/.legbled/`.

## Step 3: Get the Genesis File

Replace the generated genesis with the canonical testnet genesis:

```bash
# Download from the testnet coordinator or seed node
curl -o ~/.legbled/config/genesis.json <GENESIS_URL>

# Validate
legbled validate-genesis
```

## Step 4: Configure Your Node

### CometBFT Config (`~/.legbled/config/config.toml`)

```toml
# Block time (must match network)
[consensus]
timeout_commit = "2521ms"
timeout_propose = "2000ms"

# Persistent peers (get from testnet coordinator)
[p2p]
persistent_peers = "node_id1@ip1:26656,node_id2@ip2:26656"

# Enable prometheus metrics
[instrumentation]
prometheus = true
```

### App Config (`~/.legbled/config/app.toml`)

```toml
# Minimum gas price
minimum-gas-prices = "0.025ulgm"

# Enable REST API
[api]
enable = true
swagger = true

# Enable gRPC
[grpc]
enable = true
```

## Step 5: Sync the Chain

Start the node to sync:

```bash
legbled start --minimum-gas-prices "0.025ulgm"
```

Wait until the node is fully synced. Check sync status:

```bash
curl -s localhost:26657/status | jq '.result.sync_info.catching_up'
# Should return "false" when synced
```

## Step 6: Create Your Validator Key

```bash
legbled keys add <your-key-name> --keyring-backend file
```

Save the mnemonic securely. Note your address:

```bash
legbled keys show <your-key-name> -a --keyring-backend file
```

## Step 7: Fund Your Account

You need LGM to self-delegate. On testnet, request from the faucet or another account:

```bash
# Check your balance via REST API
curl -s "http://localhost:1317/cosmos/bank/v1beta1/balances/$(legbled keys show <key> -a)"
```

Minimum self-delegation is **0.111 LGM** (111,000 ulgm), but more stake means higher tier eligibility.

## Step 8: Register as a Validator

Legible Money uses its own staking module (`legible_staking`) alongside the standard Cosmos staking.

### Generate a Consensus Key

Your consensus key is the Ed25519 key in `~/.legbled/config/priv_validator_key.json`. Extract the hex pubkey:

```bash
# Extract the base64 pubkey and convert to hex
cat ~/.legbled/config/priv_validator_key.json | \
  jq -r '.pub_key.value' | \
  base64 -d | \
  xxd -p -c 32
```

### Register

```bash
legbled tx legible_staking register-validator <pubkey-hex> <self-delegation-ulgm> \
  --commission 1000 \
  --moniker "<your-moniker>" \
  --details "<description>" \
  --website "<your-url>" \
  --identity "<did:lgm:your-hex>" \
  --from <your-key> \
  --keyring-backend file \
  --chain-id lgm-testnet-1 \
  --fees 300000ulgm \
  -y
```

**Arguments:**
- `<pubkey-hex>`: 32-byte hex Ed25519 consensus public key
- `<self-delegation-ulgm>`: Amount in ulgm (plain number, no denom suffix). E.g., `10000000000` for 10,000 LGM
- `--commission`: Basis points. 1000 = 10%
- `--identity`: Optional DID for Proof of Truth identity linkage

### Verify Registration

```bash
legbled query legible_staking validators
legbled query legible_staking validator <your-lgm-address>
```

## Tier Advancement

| Tier | Min Stake | Min Verifications | Min Accuracy | Categories |
|------|-----------|-------------------|--------------|------------|
| Apprentice | 0.111 LGM | 0 | 0% | protocol, computational, formal |
| Verified | 1.11 LGM | 22 | 77% | + empirical |
| Bonded | 1,111 LGM | 11 | 50% | + oracle, attestation |
| Guardian | 11,111 LGM | 333 | 77% | + predictive, social, contested |

Tier transitions happen automatically when stake and performance thresholds are met. Only **Bonded** and **Guardian** validators count toward `MaxValidators` for block production. Apprentice and Verified validators participate as **verifiers only**.

## Common Operations

### Increase Stake

```bash
legbled tx legible_staking update-stake <amount-ulgm> --increase \
  --from <key> --keyring-backend file \
  --chain-id lgm-testnet-1 --fees 300000ulgm -y
```

### Delegate to a Validator

```bash
legbled tx legible_staking delegate <validator-address> <amount-ulgm> \
  --from <key> --keyring-backend file \
  --chain-id lgm-testnet-1 --fees 300000ulgm -y
```

### Undelegate

```bash
legbled tx legible_staking undelegate <validator-address> <amount-ulgm> \
  --from <key> --keyring-backend file \
  --chain-id lgm-testnet-1 --fees 300000ulgm -y
```

Undelegated tokens enter an **unbonding period** (268,560 blocks, ~7 days on testnet 102,672 blocks ~3 days).

### Redelegate

```bash
legbled tx legible_staking redelegate <from-validator> <to-validator> <amount-ulgm> \
  --from <key> --keyring-backend file \
  --chain-id lgm-testnet-1 --fees 300000ulgm -y
```

Redelegation has a cooldown of 1,111 blocks.

### Query Delegations

```bash
# Delegations to a validator
legbled query legible_staking delegations-for-validator <validator-address>

# Unbonding entries for a delegator
legbled query legible_staking unbondings <delegator-address>

# Module params
legbled query legible_staking params
```

## Monitoring

### Prometheus Endpoints

When `prometheus = true` in config.toml:

| Endpoint | Port | Description |
|----------|------|-------------|
| CometBFT metrics | 26660 | Block production, P2P, consensus |
| App metrics | 26660 | Module-specific metrics |

### Useful Queries

```bash
# Node status
curl -s localhost:26657/status | jq '.result.sync_info'

# Latest block
curl -s localhost:26657/status | jq '.result.sync_info.latest_block_height'

# Validator info
legbled query legible_staking validator <your-address>

# Check balance via REST
curl -s "localhost:1317/cosmos/bank/v1beta1/balances/<your-address>"
```

### Checking Consensus Participation

Your validator is signing blocks if:
- `voting_power > 0` in `curl localhost:26657/status`
- Block headers include your validator's signature

## Slashing

Legible Money uses progressive slashing:

- **Downtime**: 1% slash (slashing window: 100 blocks on devnet)
- **Escalation**: Each subsequent slash within an epoch increases by `slash_escalation_bps` (10x)
- **Max slashes per epoch**: 2 (then deactivated)
- **Reputation damage**: 10,000 per slash event (out of 1,000,000 max)
- **Deactivation**: After `max_slash_count_deactivate` (3) total slashes

## Troubleshooting

### "account not found" on first transaction

Your account must exist on-chain before signing. Fund it first (even a small amount). The first transaction registers the on-chain public key.

### Self-delegation amount format

The `register-validator` command takes the amount as a **plain number** (in ulgm), NOT with denom suffix:

```bash
# CORRECT
legbled tx legible_staking register-validator <pubkey> 10000000000

# WRONG (will fail with "invalid initial stake")
legbled tx legible_staking register-validator <pubkey> 10000000000ulgm
```

### Bank queries not available in CLI

The SDK `bank` module queries are not wired as custom CLI commands. Use the REST API instead:

```bash
curl -s "localhost:1317/cosmos/bank/v1beta1/balances/<address>"
```

### Minimum fees

The minimum gas price is `0.025ulgm` with a default gas limit of 200,000. Use `--fees 300000ulgm` as a safe default for most transactions.

### Vote Extensions

Vote extensions must be enabled at height 1 (`vote_extensions_enable_height = "1"` in genesis). Without this, VRF-based validator selection for PoT will not function.
