# Session: POT-DEVNET — Verify Proof of Truth works on multi-validator devnet

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Verify the Proof of Truth consensus mechanism works end-to-end on a live devnet

### Context
All P0 fixes for ABCI, vote extensions, commitment/reveal, and VRF selection are committed (batches 1-2). But PoT has never been tested on a live multi-validator network. We need to verify the full cycle: claim → VRF selection → commit → reveal → aggregation → fact creation.

### Step 1: Setup
Boot the Docker 3-validator devnet (from session 4) or a local multi-validator setup.

Key requirements:
- `min_verifiers` must be ≤ 3 (number of validators)
- Vote extensions must be enabled (height 1)
- Knowledge params must have reasonable phase durations for testing:
  ```
  commit_phase_blocks: 10
  reveal_phase_blocks: 10
  aggregation_phase_duration: 5
  ```

If the genesis has `min_verifiers=1`, that's fine for testing — only 1 validator needs to participate.

### Step 2: Submit a knowledge claim
```bash
legbled tx knowledge submit-claim \
  --domain "general" \
  --content "The Earth orbits the Sun" \
  --epistemic-category "empirical" \
  --from test1 \
  --yes --gas 500000 --fees 500000ulgm
```

### Step 3: Monitor verification round
```bash
# Check if a verification round was created
legbled query knowledge active-rounds

# Watch block heights and check phase transitions
# Should see: commit → reveal → aggregation → complete
for i in $(seq 1 50); do
  HEIGHT=$(legbled status | jq -r '.sync_info.latest_block_height')
  ROUNDS=$(legbled query knowledge active-rounds --output json 2>/dev/null | jq '.rounds | length')
  echo "Height: $HEIGHT, Active rounds: $ROUNDS"
  sleep 3
done
```

### Step 4: Check vote extensions
```bash
# Check validator logs for ExtendVote activity
docker logs lgm-devnet-validator-1 2>&1 | grep -i "extend\|commit\|reveal\|vrf" | tail -20
```

### Step 5: Verify fact creation
After the round completes (commit + reveal + aggregation phases):
```bash
legbled query knowledge facts --output json | jq '.facts | length'
```

If a fact was created, the full PoT cycle works.

### Step 6: Verify rewards/slashes
```bash
# Check if validators received verification rewards
legbled query legible_staking validator <validator_addr> --output json | jq '.verifications_completed'
```

### Step 7: Troubleshooting
If the round doesn't complete:
1. Check if VRF selection is working: validator logs should show "selected for round" or similar
2. Check if commitments are being stored: `legbled query knowledge round <round_id>`
3. Check LocalCommitmentStore: look in validator data directories for local state
4. Check phase transitions: are blocks advancing past commit/reveal deadlines?
5. Single-validator issue: if only 1 validator, it should self-select via VRF. If `min_verifiers > 1`, the round may never complete.

### Step 8: Document the PoT lifecycle
Write a clear narrative of what happened block-by-block:
- Block N: claim submitted, round created
- Block N+1 to N+10: commit phase, which validators committed
- Block N+11 to N+20: reveal phase, which validators revealed
- Block N+21 to N+25: aggregation
- Block N+26: fact created (or not — explain why)

## Report format
Write to reports/batch-6/SESSION-B6-6-POT-DEVNET.md:
- PoT lifecycle narrative (block-by-block)
- Vote extension activity from logs
- Round state at each phase transition
- Fact creation result
- Reward/slash distribution
- Issues found
- If PoT doesn't work end-to-end: exact failure point and recommended fix
