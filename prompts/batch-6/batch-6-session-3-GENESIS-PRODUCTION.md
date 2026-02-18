# Session: GENESIS-PRODUCTION — Production-ready genesis configuration for testnet

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Create a production-ready testnet genesis configuration

### Context
Currently `scripts/genesis-ceremony.sh` has the most complete genesis config (11 knowledge params, 5 gov params, emergency params, vesting params). But it was written before R3-R4 changes (ExtendedParams, TierConfigs, CleanupInterval, QuorumThreshold, etc.). The devnet scripts use minimal configs.

We need ONE canonical genesis configuration that includes ALL governance-adjustable parameters with sensible testnet defaults.

### Step 1: Inventory all module params
For EVERY module that has a `DefaultParams()` in its types package, read the defaults and document them. Key modules:

1. `x/knowledge/types/genesis.go` — ExtendedParams (60+ fields from R3-4)
2. `x/staking/types/genesis.go` — TierConfigs, SlashDecayPeriodBlocks, etc.
3. `x/gov/types/genesis.go` — QuorumThresholdBps, VotingPeriod, etc.
4. `x/tokens/types/genesis.go` — token params
5. `x/bvm/types/genesis.go` — MaxContractGasLimit, MaxScheduledGasPerBlock
6. `x/channels/types/genesis.go` — channel params
7. `x/ibcratelimit/types/genesis.go` — rate limit params
8. `x/icaauth/types/genesis.go` — ICA params
9. `x/schedule/types/genesis.go` — CleanupIntervalBlocks
10. `x/home/types/genesis.go` — home params
11. `x/emergency/types/genesis.go` — halt params
12. `x/vesting_rewards/types/genesis.go` — rewards params
13. All other modules with params

### Step 2: Create testnet genesis template
Create `scripts/testnet-genesis-config.json` with a jq patch script that sets ALL params explicitly. Group by category:

```json
{
  "consensus": {
    "vote_extensions_enable_height": "1",
    "block_max_gas": "33333333"
  },
  "knowledge": {
    "min_verifiers": 3,
    "commit_phase_blocks": 50,
    "reveal_phase_blocks": 50,
    ...all 60+ ExtendedParams fields...
  },
  "staking": {
    "tier_configs": [...4 tier configs...],
    "slash_decay_period_blocks": 34272,
    ...
  },
  ...
}
```

### Step 3: Create genesis ceremony script for testnet
Create `scripts/testnet-genesis.sh` that:
1. Initializes a chain with `--chain-id lgm-testnet-1`
2. Applies ALL params from the config
3. Creates validator accounts with appropriate stake
4. Sets up genesis domains for knowledge module
5. Validates the genesis
6. Outputs instructions for additional validators to join

### Step 4: Verify
- Run the script, validate genesis
- Boot a single-validator devnet with the genesis
- Verify params are set correctly:
  ```bash
  legbled query knowledge params
  legbled query legible_staking params
  legbled query lgm_gov params
  ```

### Step 5: Document
Create `docs/TESTNET-LAUNCH.md` with:
- Hardware requirements
- Genesis ceremony steps
- How to join as a validator
- Known limitations

## Report format
Write to reports/batch-6/SESSION-B6-3-GENESIS-PRODUCTION.md:
- Complete parameter inventory (module → param → default → testnet value)
- Genesis ceremony script location
- Validation results
- Launch documentation location
