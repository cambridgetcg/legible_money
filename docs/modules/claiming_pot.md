# x/claiming_pot — Distribution Pools

## Purpose

Manage distributable pools (airdrops, rewards) with eligibility checking. Supports creating pots with initial funding, optional expiry, eligibility validation, and claim record tracking.

## Key Concepts

- **Pot Status**: active → exhausted → expired → closed
- **Eligibility**: Validated per-claim based on qualification criteria
- **Claim Records**: Per-user claim history with amounts

## CLI Commands

### Transactions
```bash
legbled tx claiming_pot create-pot --amount 10000000ulgm --expires-at 500000 --from authority
legbled tx claiming_pot fund-pot [pot-id] --amount 5000000ulgm --from alice
legbled tx claiming_pot claim-from-pot [pot-id] --from bob
legbled tx claiming_pot close-pot [pot-id] --from authority
```

### Queries
```bash
legbled query claiming_pot pot [id]
legbled query claiming_pot pots --status active
legbled query claiming_pot pots-by-creator [address]
legbled query claiming_pot eligibility [pot-id] [claimer]
legbled query claiming_pot claim-records [pot-id]
legbled query claiming_pot params
```

## Events

`PotCreated`, `PotFunded`, `ClaimRecorded`, `PotClosed`

## Dependencies

BankKeeper, AccountKeeper, QualificationKeeper (eligibility)

## BeginBlock

Expires pots past their ExpiresAtBlock.
