# R5 Session 2: DevNet Boot-to-Block Validation

## Context
The devnet initializes cleanly but the start script doesn't keep the node running in foreground. Need to verify the full boot → genesis → block production → basic tx lifecycle works end-to-end.

## Tasks

### 1. Start DevNet and Verify Block Production
```bash
cd /Users/yournameisai/Desktop/legible-money
make devnet-reset  # clean state
make devnet        # init + start

# Start node in background
build/legbled start --home $HOME/.legbled-devnet &
LEGBLED_PID=$!

# Wait for first blocks
sleep 15
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'
# Must be > 0
```

### 2. Run Existing TX Lifecycle Tests
```bash
bash devnet/scripts/test-tx-lifecycle.sh
```
This runs 23 tests across 5 phases (bank, staking, governance, auth/errors, economic invariants). Document pass/fail for each.

### 3. Run DevNet Health Check
```bash
bash devnet/scripts/devnet-healthcheck.sh --full
```

### 4. Test New R3/R4 Features on Live Chain

**Test tokens DelegatePower** (R3-6):
```bash
# Create a secondary token, mint, delegate
build/legbled tx tokens create-token test-token TST 6 --features wrappable --from test1 ...
build/legbled tx tokens mint-token test-token 1000000 $(addr test1) --from test1 ...
build/legbled tx tokens delegate-power test-token $(addr test2) 500000 --from test1 ...
build/legbled query tokens balance test-token $(addr test1)
```

**Test tokens WrapToken** (R3-6):
```bash
build/legbled tx tokens wrap-token test-token 200000 --from test1 ...
build/legbled query bank balances $(addr test1)  # should show lgm20/test-token
build/legbled tx tokens unwrap-token lgm20/test-token 100000 --from test1 ...
```

**Test governance param update** (R3-4/R3-5):
```bash
# Submit param update proposal
build/legbled tx gov submit-lip ... --title "Update knowledge params" ...
```

### 5. Check All Module Genesis Init
```bash
# Query each module's params to verify genesis loaded correctly
for mod in knowledge staking tokens emergency gov channels auth billing; do
  echo "=== $mod ===" && build/legbled query $mod params --node http://localhost:26657 2>&1 | head -5
done
```

### 6. Verify Vote Extensions (ABCI++)
Check that vote extensions are being produced after the configured enable height:
```bash
# Check latest block for vote extensions
curl -s http://localhost:26657/block | jq '.result.block.last_commit.signatures | length'
```

## Verification
- Block height advances past 10
- All 23 tx lifecycle tests pass
- New R3/R4 tx types execute on-chain
- No panics in node logs

## Report
Write to `/reports/batch-5/SESSION-R5-2-DEVNET-BOOT.md` with:
- Block production status (height after 30s, 60s)
- TX lifecycle test results (23 tests)
- Health check results
- New feature test results
- Any panics or unexpected errors in node logs
- List of modules that fail to initialize or query
