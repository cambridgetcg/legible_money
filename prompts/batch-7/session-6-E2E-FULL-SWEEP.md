# Session: E2E-FULL-SWEEP — Run ALL E2E tests against Docker testnet after all fixes

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Final E2E test sweep against 3-validator Docker testnet

### Context
All blockers should be fixed by now (maccPerms, CLI wiring, events, pubkey bootstrap). This session verifies EVERYTHING works end-to-end on a live multi-validator network.

### Step 1: Fresh Docker build + init
```bash
# Clean rebuild with latest changes
docker build -f deployment/docker/Dockerfile.cosmos -t legbled:latest . --no-cache

# Fresh init
docker compose -f deployment/docker/docker-compose.yml down -v
bash deployment/docker/init-cosmos-docker.sh
docker compose -f deployment/docker/docker-compose.yml up -d
sleep 15  # wait for consensus
```

### Step 2: Run existing test suites
```bash
# Base TX lifecycle (23 tests)
bash devnet/scripts/test-tx-lifecycle.sh

# LGM module tests (28 tests)
bash devnet/scripts/test-lgm-modules.sh
```

**Target: ALL 51 tests pass, 0 skips.**

The 3 previously skipped tests should now pass:
- A3: Claim ID extraction from events (fixed by B6-5)
- E3: Query LIP by ID (fixed by B6-1 maccPerms)
- E5: Cast vote on LIP (fixed by B6-1 maccPerms)

### Step 3: Test newly-fixed modules
Test modules that were previously blocked by missing maccPerms:

```bash
# Ontology
legbled tx ontology propose-domain "test_domain" --parent "general" --from test1 --yes --gas 300000 --fees 500000ulgm

# Research
legbled tx research submit-research --title "Test" --abstract "Test research" --domain "general" --from test1 --yes --gas 300000 --fees 500000ulgm

# Discovery
legbled tx discovery register-agent --moniker "agent1" --from test1 --yes --gas 300000 --fees 500000ulgm

# Schedule
legbled tx schedule create-schedule --from test1 --yes --gas 300000 --fees 500000ulgm

# BVM with value transfer
legbled tx bvm deploy-contract <bytecode> --from test1 --yes --gas 500000 --fees 500000ulgm
legbled tx bvm call-contract <addr> <input> --amount 1000ulgm --from test1 --yes --gas 500000 --fees 500000ulgm

# Tokens with features
legbled tx tokens create-token "Test" TST 6 1000000 --mintable --burnable --from test1 --yes --gas 300000 --fees 500000ulgm
legbled tx tokens mint-token <token-id> <to-addr> 500000 --from test1 --yes --gas 300000 --fees 500000ulgm
```

### Step 4: Cross-validator queries
For each successful tx, verify the state is queryable from a DIFFERENT validator:
```bash
# Submit on validator-1, query on validator-3
docker exec lgm-devnet-validator-3 legbled query knowledge facts
docker exec lgm-devnet-validator-3 legbled query home homes
docker exec lgm-devnet-validator-3 legbled query lgm_gov params
```

### Step 5: Consensus stability
After all tests, verify all 3 validators remain in sync:
```bash
for i in 1 2 3; do
  docker exec lgm-devnet-validator-$i legbled status 2>&1 | jq '{height: .sync_info.latest_block_height, hash: .sync_info.latest_block_hash}'
done
```

### Step 6: Cleanup
```bash
docker compose -f deployment/docker/docker-compose.yml down -v
```

## Report format
Write to reports/batch-7/SESSION-B7-6-E2E-FULL-SWEEP.md:
- Test suite results: pass/fail/skip counts
- Previously-skipped tests: now passing?
- Newly-tested modules (previously blocked by maccPerms): results
- Cross-validator query results
- Consensus status after all tests
- Any remaining failures with root cause
- Final verdict: testnet ready?
