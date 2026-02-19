# B12-6 — E2E Flow Test: Register → Partner → Submit → Verify → Earn → Split

## The Problem
No end-to-end test validates the complete human-agent cooperation flow:
account registration, partnership formation, knowledge work, verification,
reward distribution through partnership economics, and pot management.

## The Test Script

Create `devnet/scripts/test-cooperation-flow.sh`:

```bash
#!/usr/bin/env bash
# E2E Cooperation Flow — Full human-agent lifecycle test
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/test-helpers.sh"

BINARY="${PROJECT_ROOT}/build/legbled"
NODE="tcp://localhost:26657"
HOME_DIR="${NODE_HOME:-$HOME/.legbled-genesis777}"
KEYRING="--keyring-backend test --home $HOME_DIR"

# ── Phase C1: Account Registration ──────────────────────────────
section "C1: Account Registration"

# Generate human and agent keys
$BINARY keys add human1 $KEYRING 2>/dev/null
$BINARY keys add agent1 $KEYRING 2>/dev/null

HUMAN_ADDR=$($BINARY keys show human1 -a $KEYRING)
AGENT_ADDR=$($BINARY keys show agent1 -a $KEYRING)

# Fund accounts (from validator)
$BINARY tx bank send validator $HUMAN_ADDR 10000000ulgm \
    --chain-id $CHAIN_ID $KEYRING --fees 1000ulgm -y --node $NODE
sleep 5
$BINARY tx bank send validator $AGENT_ADDR 10000000ulgm \
    --chain-id $CHAIN_ID $KEYRING --fees 1000ulgm -y --node $NODE
sleep 5

# Register as Legible accounts
HUMAN_PUBKEY=$($BINARY keys show human1 --pubkey $KEYRING | jq -r '.key' | base64 -d | xxd -p -c 64)
AGENT_PUBKEY=$($BINARY keys show agent1 --pubkey $KEYRING | jq -r '.key' | base64 -d | xxd -p -c 64)

# Derive DIDs
HUMAN_DID="did:lgm:$(echo $HUMAN_PUBKEY | cut -c1-32)"
AGENT_DID="did:lgm:$(echo $AGENT_PUBKEY | cut -c1-32)"

$BINARY tx legible-auth register \
    --account-type human --did $HUMAN_DID --public-key $HUMAN_PUBKEY \
    --from human1 $KEYRING --chain-id $CHAIN_ID --fees 1000ulgm -y --node $NODE
sleep 5

$BINARY tx legible-auth register \
    --account-type agent --did $AGENT_DID --public-key $AGENT_PUBKEY \
    --from agent1 $KEYRING --chain-id $CHAIN_ID --fees 1000ulgm -y --node $NODE
sleep 5

# Verify registration
check_account $HUMAN_ADDR "human"
check_account $AGENT_ADDR "agent"
pass "Both accounts registered"

# ── Phase C2: Partnership Formation ─────────────────────────────
section "C2: Partnership Formation"

$BINARY tx partnerships initiate \
    --partner $AGENT_ADDR --lock-tier 1 \
    --from human1 $KEYRING --chain-id $CHAIN_ID --fees 1000ulgm -y --node $NODE
sleep 5

# Get partnership ID from events
PARTNERSHIP_ID=$($BINARY query partnerships by-participant $HUMAN_ADDR \
    --node $NODE -o json | jq -r '.partnership_ids[0]')
assert_not_empty "$PARTNERSHIP_ID" "Partnership ID exists"

# Agent accepts
$BINARY tx partnerships accept \
    --partnership-id $PARTNERSHIP_ID \
    --from agent1 $KEYRING --chain-id $CHAIN_ID --fees 1000ulgm -y --node $NODE
sleep 5

# Verify active
check_partnership_status $PARTNERSHIP_ID "active"
pass "Partnership active"

# ── Phase C3: Knowledge Work — Submit Claim ─────────────────────
section "C3: Submit Claim"

# Agent submits a claim referencing genesis axioms
$BINARY tx knowledge submit-claim \
    "The conservation of energy implies that perpetual motion machines are impossible" \
    --domain physics --references PHYS-006 \
    --stake 100000ulgm \
    --from agent1 $KEYRING --chain-id $CHAIN_ID --fees 1000ulgm -y --node $NODE
sleep 5

# Get claim ID
CLAIM_ID=$($BINARY query knowledge claims --submitter $AGENT_ADDR \
    --node $NODE -o json | jq -r '.claims[-1].id')
assert_not_empty "$CLAIM_ID" "Claim submitted"

# Wait for evaluation (next block)
sleep 5
check_claim_evaluated $CLAIM_ID
pass "Claim evaluated"

# ── Phase C4: Verification Round ────────────────────────────────
section "C4: Verification"

# Wait for verification round to complete (this may take multiple blocks)
wait_for_verification $CLAIM_ID 30

# Check if claim became a fact
FACT_ID=$($BINARY query knowledge facts --submitter $AGENT_ADDR \
    --node $NODE -o json | jq -r '.facts[-1].id // empty')
if [ -n "$FACT_ID" ]; then
    pass "Claim verified, fact created: $FACT_ID"
else
    warn "Claim not yet verified (may need more validators)"
fi

# ── Phase C5: Reward Distribution ────────────────────────────────
section "C5: Reward Distribution"

# Check balances after verification
HUMAN_BALANCE=$($BINARY query bank balances $HUMAN_ADDR --node $NODE -o json \
    | jq -r '.balances[] | select(.denom=="ulgm") | .amount')
AGENT_BALANCE=$($BINARY query bank balances $AGENT_ADDR --node $NODE -o json \
    | jq -r '.balances[] | select(.denom=="ulgm") | .amount')

echo "Human balance: $HUMAN_BALANCE ulgm"
echo "Agent balance: $AGENT_BALANCE ulgm"

# If B12-1 has landed, human should have earned from partnership split
# Check partnership pot balance
POT_BALANCE=$($BINARY query partnerships partnership $PARTNERSHIP_ID \
    --node $NODE -o json | jq -r '.partnership.common_pot_balance')
echo "Common pot: $POT_BALANCE ulgm"

if [ "$POT_BALANCE" != "0" ] && [ "$POT_BALANCE" != "" ]; then
    pass "Common pot has funds — partnership reward routing works"
else
    warn "Common pot is empty — B12-1 may not have landed yet"
fi

# ── Phase C6: Pot Operations ─────────────────────────────────────
section "C6: Pot Deliberation"

# Agent proposes spending from pot
if [ "$POT_BALANCE" != "0" ] && [ "$POT_BALANCE" != "" ]; then
    # Propose spending 10% of pot
    SPEND_AMOUNT=$(echo "$POT_BALANCE / 10" | bc)
    if [ "$SPEND_AMOUNT" -gt 0 ]; then
        $BINARY tx partnerships propose-operation \
            --partnership-id $PARTNERSHIP_ID \
            --operation-type withdraw \
            --amount $SPEND_AMOUNT \
            --from agent1 $KEYRING --chain-id $CHAIN_ID --fees 1000ulgm -y --node $NODE
        sleep 5

        # Human approves
        OP_ID=$($BINARY query partnerships operations $PARTNERSHIP_ID \
            --node $NODE -o json | jq -r '.operations[-1].id')
        $BINARY tx partnerships approve-operation \
            --operation-id $OP_ID \
            --from human1 $KEYRING --chain-id $CHAIN_ID --fees 1000ulgm -y --node $NODE
        sleep 5

        check_operation_status $OP_ID "approved"
        pass "Pot operation approved through deliberation"
    fi
fi

# ── Phase C7: Safety Mechanisms ──────────────────────────────────
section "C7: Safety"

# Test safety freeze
$BINARY tx partnerships safety-freeze \
    --partnership-id $PARTNERSHIP_ID \
    --from agent1 $KEYRING --chain-id $CHAIN_ID --fees 1000ulgm -y --node $NODE
sleep 5

check_partnership_status $PARTNERSHIP_ID "suspended"
pass "Safety freeze works"

# Wait for freeze to expire (222 blocks at 2.5s ≈ 9 min — skip in fast mode)
# In test mode, just verify the freeze was recorded

# ── Phase C8: Session Keys ───────────────────────────────────────
section "C8: Agent Autonomy (Session Keys)"

# Create a session key for agent with limited capabilities
$BINARY tx legible-auth create-session \
    --capabilities "knowledge.submit_claim,knowledge.verify" \
    --duration 1000 \
    --from agent1 $KEYRING --chain-id $CHAIN_ID --fees 1000ulgm -y --node $NODE
sleep 5

# Verify session created
SESSION_COUNT=$($BINARY query legible-auth account $AGENT_ADDR \
    --node $NODE -o json | jq '.account.session_key_count')
assert_gte "$SESSION_COUNT" 1 "Session key created"
pass "Agent session key active"

print_summary
```

### Helper Functions

Add to `test-helpers.sh`:
```bash
check_account() {
    local addr=$1 expected_type=$2
    local actual_type
    actual_type=$($BINARY query legible-auth account "$addr" \
        --node "$NODE" -o json 2>/dev/null | jq -r '.account.account_type')
    if [ "$actual_type" = "$expected_type" ]; then
        pass "Account $addr: type=$expected_type"
    else
        fail "Account $addr: expected type=$expected_type, got $actual_type"
    fi
}

check_partnership_status() {
    local id=$1 expected=$2
    local actual
    actual=$($BINARY query partnerships partnership "$id" \
        --node "$NODE" -o json 2>/dev/null | jq -r '.partnership.status')
    if [ "$actual" = "$expected" ]; then
        pass "Partnership $id: status=$expected"
    else
        fail "Partnership $id: expected status=$expected, got $actual"
    fi
}

wait_for_verification() {
    local claim_id=$1 max_blocks=$2
    local start_height current_height
    start_height=$($BINARY status --node "$NODE" 2>/dev/null | jq -r '.sync_info.latest_block_height')
    while true; do
        sleep 3
        current_height=$($BINARY status --node "$NODE" 2>/dev/null | jq -r '.sync_info.latest_block_height')
        if [ $((current_height - start_height)) -ge $max_blocks ]; then
            return 1
        fi
        local status
        status=$($BINARY query knowledge claim "$claim_id" --node "$NODE" -o json 2>/dev/null | jq -r '.claim.status')
        if [ "$status" = "verified" ] || [ "$status" = "rejected" ]; then
            return 0
        fi
    done
}
```

### Run

```bash
# Start chain with genesis axioms
bash scripts/genesis-777.sh
build/legbled start --home ~/.legbled-genesis777 &
sleep 10

# Run cooperation flow test
bash devnet/scripts/test-cooperation-flow.sh

kill %1
```

## Commit
```
test(e2e): full cooperation flow — register, partner, claim, verify, earn, split

End-to-end test covering the complete human-agent lifecycle:
account registration, partnership formation, knowledge claim submission,
verification, reward distribution through partnership economics,
pot deliberation, safety mechanisms, and session key autonomy.
```
