# x/vesting_rewards — Block Rewards & Vesting

## Purpose

Block reward distribution with research tax and category-dependent vesting schedules. Awards block producers proportionally. Empty blocks get 0% reward (PoT principle: no useful work = no reward). Supports exponential decay per epoch, adaptive multipliers via autopoiesis, and clawback on falsification.

## Key Concepts

- **Research Tax**: 7% of all rewards to research fund
- **Epistemic Vesting**: Category determines vesting speed (axiomatic/formal fastest, empirical/historical slowest)
- **Empty Block Penalty**: 0% reward for blocks with no transactions
- **Decay**: Exponential decay per epoch (default 0.9x), with floor rate
- **Clawback**: Falsified facts trigger vesting clawback
- **Adaptive**: Autopoiesis can adjust block reward multiplier

## CLI Commands

No transaction commands (governance-only via MsgUpdateParams).

### Queries
```bash
legbled query vesting_rewards params
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| research_share_bps | 2,200 | 22% research fund share |
| block_reward_amount | "1000000" | 1 LGM base reward |
| empty_block_reward_rate | 0 | 0% for empty blocks |
| min_validators_for_full_reward | 3 | Target validator count (testnet) |
| decay_factor_bps | 900,000 | 0.9x per epoch |
| floor_reward | "100" | Minimum reward per block |

13 total governance-adjustable parameters.

## Events

`vesting_created`, `claim_vested_rewards`, `falsify_vesting`, `vesting_paused`, `vesting_resumed`, `vesting_accelerated`, `vesting_completed`

## Dependencies

BankKeeper, StakingKeeper, AutopoiesisKeeper (optional)
