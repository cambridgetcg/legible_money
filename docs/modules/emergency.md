# x/emergency — Emergency Halt Kill Switch

## Purpose

3-phase BFT consensus kill switch for emergency chain halt, state revert, and resume. Only Guardian-tier validators (tier 3) can propose. Uses prevote → precommit ceremony matching CometBFT phases. Anti-abuse: per-guardian limits, global epoch limits, cooldown periods.

## Key Concepts

- **3 Ceremony Types**: halt, revert, resume
- **3 Phases**: prevote → precommit → finalized (or failed)
- **Guardian-Only**: Only tier 3 validators can propose
- **Quorum-Based**: Configurable quorum per ceremony type
- **Auto-Resume**: Expired halts automatically resume
- **Audit Log**: All actions logged with 18 action types

## CLI Commands

### Transactions
```bash
legbled tx emergency propose-halt --reason "critical bug" --from guardian1
legbled tx emergency vote-halt [ceremony-id] --support --phase prevote --from guardian2
legbled tx emergency propose-revert --target-height 100000 --from guardian1
legbled tx emergency vote-revert [ceremony-id] --support --phase precommit --from guardian2
legbled tx emergency propose-resume --from guardian1
legbled tx emergency vote-resume [ceremony-id] --support --from guardian2
```

### Queries
```bash
legbled query emergency status
legbled query emergency active-ceremony
legbled query emergency completed-ceremonies
legbled query emergency audit-log
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| halt_quorum | 666,666 BPS | 66.7% quorum for halt |
| revert_quorum | 750,000 BPS | 75% quorum for revert |
| resume_quorum | 500,000 BPS | 50% quorum for resume |
| min_distinct_voters | 3 | Min unique voters (testnet) |
| max_revert_depth | 1,000 blocks | Max blocks to revert |
| cooldown_blocks | 1,111 | Between proposals |

20 total governance-adjustable parameters.

## Events

`emergency_halt_proposed`, `emergency_revert_proposed`, `emergency_resume_proposed`, `emergency_revert_required`, `emergency_params_updated`

## Dependencies

StakingKeeper (Guardian tier lookup, voting power)
