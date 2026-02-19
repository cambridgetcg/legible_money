# B9-5: Testnet Faucet + Block Explorer Setup

## Context
For testnet availability, users need: (a) a way to get test tokens, (b) a way to see what's happening on chain.

## Task

### 1. Faucet
Create a simple HTTP faucet service:
- `tools/faucet/main.go`
- Accepts: `POST /faucet {"address": "lgm1..."}`
- Sends a fixed amount (e.g., 10000000 ulgm) from a funded faucet account
- Rate limiting: 1 request per address per hour
- Simple HTML page at `GET /` for browser use

Technical approach:
- Use the Cosmos SDK client to sign and broadcast tx
- Faucet account funded at genesis with a large allocation
- Config via environment vars: `FAUCET_MNEMONIC`, `CHAIN_ID`, `NODE_URL`

### 2. Block Explorer
Set up Ping.pub (already have config from B6):
- Read `deployment/docker/ping-pub-config/lgm-devnet.json`
- Create a Docker compose service for Ping.pub
- Or: document how to run the existing Cosmos explorer tools

### 3. Integrate into Docker compose
Add faucet and explorer as optional services:
```yaml
services:
  faucet:
    build: ../../tools/faucet
    ports: ["8080:8080"]
    environment:
      FAUCET_MNEMONIC: "..."
      CHAIN_ID: lgm-testnet-1
      NODE_URL: http://val1:26657
    profiles: ["testnet"]

  explorer:
    image: ping-pub/explorer
    ports: ["8088:80"]
    profiles: ["testnet"]
```

### 4. Verify
- Faucet sends tokens successfully
- Explorer shows blocks, txs, validators
- Both accessible from browser

## File Ownership
- `tools/faucet/` (new)
- `deployment/docker/docker-compose.cosmos.yml` (add services)

## Report
Write `reports/batch-9/SESSION-B9-5-FAUCET-EXPLORER.md`.
