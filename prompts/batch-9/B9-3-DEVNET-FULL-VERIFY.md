# B9-3: Full DevNet Verification After Batch-8 Integration

## Context
Batch 8 added: gRPC gateway, secondary indexes, E2E race fixes, Docker hardening, genesis axiom infrastructure. All of this needs end-to-end verification on a live devnet.

## Task

### 1. Clean build
```bash
cd ~/Desktop/legible_money
make build
go vet ./...
go test ./... -count=1 -timeout 600s
```
All must pass. Record exact test count.

### 2. Start single-validator devnet
```bash
rm -rf ~/.lgm/devnet
./build/legbled init testnode --chain-id lgm-devnet-1
# Add genesis accounts, gentx, collect-gentxs
./build/legbled start
```

### 3. Verify REST API (new in B8-2)
```bash
# Cosmos SDK
curl -s http://localhost:1317/cosmos/bank/v1beta1/balances/lgm1... | jq .

# Custom modules — test at least 5
curl -s http://localhost:1317/legible/knowledge/v1/params | jq .
curl -s http://localhost:1317/legible/staking/v1/validators | jq .
curl -s http://localhost:1317/legible/tokens/v1/tokens | jq .
curl -s http://localhost:1317/legible/emergency/v1/status | jq .
curl -s http://localhost:1317/legible/gov/v1/params | jq .
```

### 4. Run full E2E test suite (B8-1 fixes)
```bash
bash devnet/scripts/test-tx-lifecycle.sh
bash devnet/scripts/test-lgm-modules.sh
```
Target: 51/51 pass.

### 5. Verify genesis axiom loading
Create a minimal test genesis with 5 axioms:
```bash
cat > /tmp/test-axioms.json << 'EOF'
[
  {"axiom_id":"MATH-001","statement":"A domain of discourse exists","type":"axiom","domain":"mathematics","epistemic_category":"formal","confidence":1.0,"dependencies":[]},
  {"axiom_id":"MATH-002","statement":"Every object is identical to itself","type":"axiom","domain":"mathematics","epistemic_category":"formal","confidence":1.0,"dependencies":["MATH-001"]},
  ...
]
EOF
```
Inject into genesis, start chain, query:
```bash
./build/legbled query knowledge claim MATH-001
```

### 6. Verify secondary indexes (B8-6)
Submit 10+ claims, then verify indexed queries are fast:
```bash
./build/legbled query knowledge claims --status pending_evaluation
./build/legbled query knowledge claims --domain mathematics
```

### 7. Docker multi-validator (B8-5)
```bash
cd deployment/docker
docker compose -f docker-compose.cosmos.yml up -d
# Wait for consensus
docker compose -f docker-compose.cosmos.yml exec val1 legbled status
# Submit a tx
docker compose -f docker-compose.cosmos.yml exec val1 legbled tx knowledge submit-claim ...
# Check monitoring
curl http://localhost:9093/targets  # Prometheus
curl http://localhost:3000          # Grafana
```

### 8. Version check
```bash
./build/legbled version --long
```
Should show v0.1.0-alpha or later tag.

## Report
Write `reports/batch-9/SESSION-B9-3-DEVNET-VERIFY.md` with pass/fail for each check.
