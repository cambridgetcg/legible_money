# R5 Session 5: Docker 3-Validator Testnet Verification

## Context
`deployment/docker/` has a complete 3-validator Docker setup:
- `Dockerfile.cosmos` — multi-stage build
- `docker-compose.cosmos.yml` — 3 validators + explorer
- `init-cosmos-docker.sh` — genesis ceremony for Docker

This was built in a prior session but has never been tested against the current codebase (R3+R4 changes). Need to verify it works.

## Tasks

### 1. Build Docker Image
```bash
cd /Users/yournameisai/Desktop/legible-money
docker build -f deployment/docker/Dockerfile.cosmos -t legbled:latest .
```
If this fails, fix the Dockerfile (likely Go version, module path, or build args).

### 2. Initialize Docker Devnet
```bash
bash deployment/docker/init-cosmos-docker.sh
```
This should:
- Generate 3 validator keys
- Create shared genesis with all validators' gentxs
- Configure persistent peers
- Populate Docker volumes

### 3. Start Network
```bash
docker compose -f deployment/docker/docker-compose.cosmos.yml up -d
sleep 30
docker compose -f deployment/docker/docker-compose.cosmos.yml ps
```
All 3 containers should be `Up`.

### 4. Verify Consensus
```bash
# Check all 3 validators are producing blocks
for port in 26657 26667 26677; do
  echo "=== Port $port ===" && curl -s http://localhost:$port/status | jq '.result.sync_info.latest_block_height'
done
```
All 3 should show the same (or very close) block height, advancing.

### 5. Verify No gRPC Errors
```bash
docker compose -f deployment/docker/docker-compose.cosmos.yml logs validator-1 2>&1 | grep "gRPC service registration FAILED"
```
If Session R5-1 fixed the gRPC issues, this should show zero failures.

### 6. Cross-Validator TX Test
```bash
# Send a tx from validator-1's account and verify it appears on validator-2
docker exec lgm-validator-1 legbled tx bank send <val1_addr> <val2_addr> 100ulgm --chain-id lgm-devnet-1 ...
sleep 5
docker exec lgm-validator-2 legbled query bank balances <val2_addr>
```

### 7. Verify API Endpoints
```bash
# REST API
curl -s http://localhost:1317/cosmos/base/tendermint/v1beta1/node_info | jq '.default_node_info.network'

# gRPC (if grpcurl available)
grpcurl -plaintext localhost:9090 list | head -10
```

### 8. Stop and Cleanup
```bash
docker compose -f deployment/docker/docker-compose.cosmos.yml down -v
```

## If Docker is Not Available
If Docker isn't installed on this machine, instead:
1. Verify the Dockerfile syntax and build configuration are correct by reading them
2. Update any outdated references (Go version, module paths)
3. Update `init-cosmos-docker.sh` for R3/R4 genesis changes
4. Document what needs to be tested manually

## Report
Write to `/reports/batch-5/SESSION-R5-5-DOCKER-MULTIVALIDATOR.md` with:
- Docker build result
- Container status (all 3)
- Block height after 30s
- gRPC/REST endpoint availability
- Cross-validator TX result
- Any failures and fixes applied
