# B13-4 — Genesis Launch Dry-Run: Full Boot with 777 Axioms

## Context
We have `scripts/genesis-777.sh` (from batch 10) and all the economic fixes
from batches 11-12. Time to do a full dry-run: parse axioms, validate DAG,
inject into genesis, boot the chain, and verify everything works.

## Task

### 1. Build Clean

```bash
cd ~/Desktop/legible_money
git stash  # if needed
make build
```

Verify the binary is fresh:
```bash
build/legbled version
```

### 2. Run Genesis Script

```bash
bash scripts/genesis-777.sh
```

Expected output:
```
=== Building legbled ===
=== Parsing 777 genesis axioms ===
✓ Parsed 777 axioms
=== Initializing chain ===
=== Validating axiom DAG ===
=== Injecting 777 axioms into genesis ===
=== Validating genesis ===
=== Genesis 777 Ready ===
```

If any step fails, fix and document.

### 3. Boot the Chain

```bash
NODE_HOME="${HOME}/.legbled-genesis777"
BINARY="build/legbled"

$BINARY start --home "$NODE_HOME" &
CHAIN_PID=$!
sleep 10

# Check block production
$BINARY status --node tcp://localhost:26657 2>/dev/null | jq '.sync_info.latest_block_height'
```

### 4. Verify Genesis Axioms

```bash
# Count all facts
$BINARY query knowledge facts --count-total --node tcp://localhost:26657 -o json | jq '.pagination.total'
# Expected: 777 (or more if other genesis facts exist)

# Spot-check key axioms
for id in MATH-001 PHYS-001 THEO-001 AGRT-000 COSM-001 PSYCH-001; do
    echo -n "$id: "
    $BINARY query knowledge fact $id --node tcp://localhost:26657 -o json 2>/dev/null | jq -r '.fact.status'
done
```

### 5. Verify Economic Parameters

```bash
# Check research tax is 3.33% (if B13-1 has landed)
$BINARY query vesting-rewards params --node tcp://localhost:26657 -o json | jq '.params.research_share_bps'
# Expected: 33300

# Check founder address (if B13-2 has landed)
$BINARY query vesting-rewards params --node tcp://localhost:26657 -o json | jq '.params.founder_address'
```

### 6. Submit a Test Claim

```bash
VALIDATOR_ADDR=$($BINARY keys show validator -a --keyring-backend test --home "$NODE_HOME")

# Register as human account first
PUBKEY=$($BINARY keys show validator --pubkey --keyring-backend test --home "$NODE_HOME" | jq -r '.key' | base64 -d | xxd -p -c 64)
DID="did:lgm:$(echo $PUBKEY | cut -c1-32)"

$BINARY tx legible-auth register \
    --account-type human --did $DID --public-key $PUBKEY \
    --from validator --keyring-backend test --home "$NODE_HOME" \
    --chain-id lgm-genesis777-1 --fees 1000ulgm -y --node tcp://localhost:26657
sleep 5

# Submit a claim referencing a genesis axiom
$BINARY tx knowledge submit-claim \
    "Water molecules consist of two hydrogen atoms covalently bonded to one oxygen atom" \
    --domain chemistry --references CHEM-001 --stake 100000ulgm \
    --from validator --keyring-backend test --home "$NODE_HOME" \
    --chain-id lgm-genesis777-1 --fees 1000ulgm -y --node tcp://localhost:26657
sleep 5

# Check claim was submitted
$BINARY query knowledge claims --submitter $VALIDATOR_ADDR \
    --node tcp://localhost:26657 -o json | jq '.claims | length'
```

### 7. Run E2E Test Suites

```bash
# Genesis axiom verification
bash devnet/scripts/test-genesis-axioms.sh

# Cooperation flow (if applicable)
bash devnet/scripts/test-cooperation-flow.sh
```

### 8. Verify Block Rewards

Wait for a few blocks and check reward distribution:
```bash
sleep 30  # ~12 blocks at 2.5s

# Check research fund balance
$BINARY query bank balances $(legbled keys show research_fund -a 2>/dev/null || echo "check module account") \
    --node tcp://localhost:26657 -o json

# Check validator balance increased
$BINARY query bank balances $VALIDATOR_ADDR --node tcp://localhost:26657 -o json | jq '.balances'
```

### 9. Document Results

Create `reports/batch-13/SESSION-B13-4-GENESIS-DRYRUN.md` with:
- Build output
- Genesis script output
- Block height reached
- Axiom count verification
- Claim submission result
- E2E test results
- Any errors encountered and fixes applied

### 10. Stop Chain

```bash
kill $CHAIN_PID
```

## Commit
```
test: genesis dry-run — full boot with 777 axioms and economic verification

Successfully boots chain with 777 genesis axioms, verifies fact loading,
tests claim submission against genesis references, and validates
economic parameter correctness.
```
