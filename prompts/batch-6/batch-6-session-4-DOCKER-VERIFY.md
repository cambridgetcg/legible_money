# Session: DOCKER-VERIFY — Verify Docker multi-validator testnet end-to-end

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Verify the Docker 3-validator testnet works end-to-end after all batch 1-5 fixes

### Context
R5-5 got Docker working but had to fix 14 packages with pointer/value proto mismatches during the Docker build. We need to verify those fixes are committed and the full Docker pipeline works cleanly.

### Step 1: Verify all proto fixes are committed
```bash
git status
git log --oneline -5
```
Check that the R5-4 (vet cleanup, ~120 files) and R5-5 (Docker proto fixes, ~40 files) changes are committed.

### Step 2: Clean Docker build
```bash
docker build -f deployment/docker/Dockerfile.cosmos -t legbled:latest . --no-cache 2>&1
```
Should build cleanly with zero errors. Document build time and image size.

### Step 3: Initialize 3-validator devnet
```bash
bash deployment/docker/init-cosmos-docker.sh
```
Should create genesis with 3 validators and start the network.

### Step 4: Verify consensus
Wait 30 seconds, then check all 3 validators:
```bash
# All 3 nodes should be at similar heights
for i in 1 2 3; do
  docker exec lgm-devnet-validator-$i legbled status | jq '.sync_info.latest_block_height'
done
```
Heights should be within 1-2 blocks of each other.

### Step 5: Cross-validator transaction
Send a bank transfer from validator-1 to validator-2 and verify it appears on validator-3:
```bash
docker exec lgm-devnet-validator-1 legbled tx bank send validator1_addr validator2_addr 100000ulgm --yes --gas 300000 --fees 300000ulgm
# Wait 5 seconds
docker exec lgm-devnet-validator-3 legbled query bank balances validator2_addr
```

### Step 6: Run E2E tests against Docker
Adapt the test scripts to run against Docker:
```bash
# Point tests at docker validator-1
export NODE="tcp://localhost:26657"
bash devnet/scripts/test-tx-lifecycle.sh
bash devnet/scripts/test-lgm-modules.sh
```
If the scripts use hardcoded localhost, they should work if Docker maps ports correctly.

### Step 7: Test custom modules in multi-validator context
- Submit a knowledge claim on validator-1, query it on validator-3
- Create an agent home on validator-2, query it on validator-1
- Verify LGM staking registration (after F-1 fix from CRITICAL-FIXES session)

### Step 8: Cleanup
```bash
docker compose -f deployment/docker/docker-compose.yml down -v
```

## Report format
Write to reports/batch-6/SESSION-B6-4-DOCKER-VERIFY.md:
- Docker build: clean? time? size?
- Consensus: all 3 validators synced?
- Cross-validator transaction confirmed?
- E2E test results against Docker
- Custom module tests in multi-validator context
- Any issues found
